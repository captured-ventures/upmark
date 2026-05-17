package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// MCPDocument is one document an LLM has pushed into upmark for the user to
// review. The frontend renders these like regular docs but tags them as
// MCP-originated so task lists become interactive and the chrome shows a
// "presented via mcp" badge.
type MCPDocument struct {
	ID           string         `json:"id"`
	Title        string         `json:"title"`
	Source       string         `json:"source"`
	Rendered     string         `json:"rendered"`
	PresentedAt  time.Time      `json:"presentedAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	Tasks        []MCPTaskState `json:"tasks"`
	ClosedByUser bool           `json:"closedByUser"`
	ViewedByUser bool           `json:"viewedByUser"`
}

// MCPTaskState is one row of a GFM task list (`- [ ] foo`). IDs are assigned
// in source order so the LLM can correlate state changes back to the
// markdown it pushed.
type MCPTaskState struct {
	ID      int    `json:"id"`
	Text    string `json:"text"`
	Checked bool   `json:"checked"`
}

// MCPStatus is what get_mcp_status returns — used by the frontend status pill.
type MCPStatus struct {
	Enabled bool   `json:"enabled"`
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	URL     string `json:"url"`
}

const defaultMCPPort = 11451

type MCPManager struct {
	app *App

	mu       sync.RWMutex
	docs     map[string]*MCPDocument
	docOrder []string // present order

	mcpServer  *server.MCPServer
	httpServer *http.Server
	running    bool
	port       int

	// lastActivity tracks the most recent tool-handler invocation. The idle
	// monitor (server mode only) reads this to decide when to auto-exit.
	lastActivity time.Time
}

func newMCPManager(a *App) *MCPManager {
	return &MCPManager{
		app:  a,
		docs: make(map[string]*MCPDocument),
	}
}

func (m *MCPManager) Start(port int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("already running")
	}
	if port <= 0 {
		port = defaultMCPPort
	}
	m.port = port

	s := server.NewMCPServer(
		"upmark",
		"0.7.0",
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)
	m.registerTools(s)
	m.mcpServer = s

	// Use the SSE transport — long-lived HTTP server LLM clients can connect to.
	sseSrv := server.NewSSEServer(s)

	addr := fmt.Sprintf("127.0.0.1:%d", port)

	// Bind first so we can surface bind errors synchronously to the caller.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("mcp listen %s: %w", addr, err)
	}

	m.httpServer = &http.Server{
		Handler:      sseSrv,
		ReadTimeout:  0, // SSE streams stay open
		WriteTimeout: 0,
	}
	m.running = true
	m.lastActivity = time.Now()

	// Record this process as the active MCP server so the bridge (and any
	// future tooling) can find us. Don't fail the start if the write errors;
	// the lockfile is a convenience, not a correctness requirement.
	mode := "ui"
	if m.app != nil && m.app.serverMode {
		mode = "server"
	}
	if err := writeMCPLock(port, mode); err != nil {
		fmt.Println("mcp lockfile write:", err)
	}

	go func() {
		_ = m.httpServer.Serve(ln)
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	return nil
}

func (m *MCPManager) Stop() error {
	m.mu.Lock()
	srv := m.httpServer
	m.httpServer = nil
	m.running = false
	m.mu.Unlock()

	clearMCPLock()

	if srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return srv.Shutdown(ctx)
}

// runIdleMonitor is started in server mode only. It polls every minute; if
// there are no presented docs and no tool activity within `timeout`, it
// shuts the app down so abandoned bridge-launched processes don't linger.
func (m *MCPManager) runIdleMonitor(ctx context.Context, timeout time.Duration) {
	tick := time.NewTicker(1 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			m.mu.RLock()
			idle := time.Since(m.lastActivity)
			docCount := len(m.docs)
			m.mu.RUnlock()
			if docCount == 0 && idle >= timeout {
				fmt.Println("mcp idle timeout reached, exiting")
				wailsruntime.Quit(ctx)
				return
			}
		}
	}
}

// touchActivity is called from every tool handler to reset the idle clock.
func (m *MCPManager) touchActivity() {
	m.mu.Lock()
	m.lastActivity = time.Now()
	m.mu.Unlock()
}

// ───── tool registration ─────

func (m *MCPManager) registerTools(s *server.MCPServer) {
	s.AddTool(
		mcp.NewTool("present_document",
			mcp.WithDescription(
				"Display a markdown document in upmark for the user to read and "+
					"interact with. Returns the document id, which can be passed to "+
					"get_document_status to poll for the user's task-list "+
					"selections, or to update_document to revise the content.",
			),
			mcp.WithString("content", mcp.Required(),
				mcp.Description("Markdown source to display.")),
			mcp.WithString("title",
				mcp.Description("Optional display title. Defaults to the first H1 in the content, or 'untitled'.")),
		),
		m.handlePresent,
	)

	s.AddTool(
		mcp.NewTool("update_document",
			mcp.WithDescription(
				"Replace the content of a previously-presented document. Task-list "+
					"selections that match by line text are preserved; new tasks "+
					"start unchecked.",
			),
			mcp.WithString("document_id", mcp.Required()),
			mcp.WithString("content", mcp.Required()),
		),
		m.handleUpdate,
	)

	s.AddTool(
		mcp.NewTool("get_document_status",
			mcp.WithDescription(
				"Return the current status of a presented document, including which "+
					"task-list items the user has checked, whether the doc is "+
					"currently in view, and whether the user has explicitly closed it.",
			),
			mcp.WithString("document_id", mcp.Required()),
		),
		m.handleStatus,
	)

	s.AddTool(
		mcp.NewTool("close_document",
			mcp.WithDescription("Remove a presented document from upmark."),
			mcp.WithString("document_id", mcp.Required()),
		),
		m.handleClose,
	)

	s.AddTool(
		mcp.NewTool("list_presented",
			mcp.WithDescription("List all documents currently presented in upmark."),
		),
		m.handleList,
	)
}

// ───── tool handlers ─────

func (m *MCPManager) handlePresent(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.touchActivity()
	content := req.GetString("content", "")
	title := req.GetString("title", "")
	if strings.TrimSpace(content) == "" {
		return mcp.NewToolResultError("content is required"), nil
	}

	rendered, err := m.app.renderer.render([]byte(content), "")
	if err != nil {
		return mcp.NewToolResultError("render: " + err.Error()), nil
	}

	if title == "" {
		title = inferTitle(content)
	}

	doc := &MCPDocument{
		ID:          newMCPID(),
		Title:       title,
		Source:      content,
		Rendered:    rendered,
		PresentedAt: time.Now(),
		UpdatedAt:   time.Now(),
		Tasks:       extractTasks(content, nil),
	}

	m.mu.Lock()
	m.docs[doc.ID] = doc
	m.docOrder = append(m.docOrder, doc.ID)
	m.mu.Unlock()

	wailsruntime.EventsEmit(m.app.ctx, "mcp-doc-presented", doc)

	// In server mode (launched by the bridge), the window starts hidden.
	// A doc presentation is the natural moment to surface it. The pref
	// controls whether we also pull focus.
	if m.app != nil && m.app.serverMode {
		m.app.ShowWindowForPresent()
	}

	return mcpJSON(map[string]any{
		"document_id":  doc.ID,
		"title":        doc.Title,
		"task_count":   len(doc.Tasks),
		"presented_at": doc.PresentedAt.Format(time.RFC3339),
	})
}

func (m *MCPManager) handleUpdate(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.touchActivity()
	id := req.GetString("document_id", "")
	content := req.GetString("content", "")
	if id == "" {
		return mcp.NewToolResultError("document_id is required"), nil
	}
	if strings.TrimSpace(content) == "" {
		return mcp.NewToolResultError("content is required"), nil
	}

	rendered, err := m.app.renderer.render([]byte(content), "")
	if err != nil {
		return mcp.NewToolResultError("render: " + err.Error()), nil
	}

	m.mu.Lock()
	doc, ok := m.docs[id]
	if !ok {
		m.mu.Unlock()
		return mcp.NewToolResultError("unknown document_id"), nil
	}
	doc.Source = content
	doc.Rendered = rendered
	doc.UpdatedAt = time.Now()
	doc.Tasks = extractTasks(content, doc.Tasks)
	m.mu.Unlock()

	wailsruntime.EventsEmit(m.app.ctx, "mcp-doc-updated", doc)

	return mcpJSON(map[string]any{
		"document_id": id,
		"updated_at":  doc.UpdatedAt.Format(time.RFC3339),
		"task_count":  len(doc.Tasks),
	})
}

func (m *MCPManager) handleStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.touchActivity()
	id := req.GetString("document_id", "")
	if id == "" {
		return mcp.NewToolResultError("document_id is required"), nil
	}
	m.mu.RLock()
	doc, ok := m.docs[id]
	if !ok {
		m.mu.RUnlock()
		return mcp.NewToolResultError("unknown document_id"), nil
	}
	statusDoc := *doc
	m.mu.RUnlock()
	return mcpJSON(statusDoc)
}

func (m *MCPManager) handleClose(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.touchActivity()
	id := req.GetString("document_id", "")
	if id == "" {
		return mcp.NewToolResultError("document_id is required"), nil
	}
	m.mu.Lock()
	_, ok := m.docs[id]
	if !ok {
		m.mu.Unlock()
		return mcp.NewToolResultError("unknown document_id"), nil
	}
	delete(m.docs, id)
	for i, d := range m.docOrder {
		if d == id {
			m.docOrder = append(m.docOrder[:i], m.docOrder[i+1:]...)
			break
		}
	}
	m.mu.Unlock()

	wailsruntime.EventsEmit(m.app.ctx, "mcp-doc-closed", id)
	return mcpJSON(map[string]any{"document_id": id, "closed": true})
}

func (m *MCPManager) handleList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	m.touchActivity()
	m.mu.RLock()
	list := make([]map[string]any, 0, len(m.docOrder))
	for _, id := range m.docOrder {
		d := m.docs[id]
		if d == nil {
			continue
		}
		list = append(list, map[string]any{
			"document_id":  d.ID,
			"title":        d.Title,
			"presented_at": d.PresentedAt.Format(time.RFC3339),
			"task_count":   len(d.Tasks),
			"viewed":       d.ViewedByUser,
			"closed":       d.ClosedByUser,
		})
	}
	m.mu.RUnlock()
	return mcpJSON(map[string]any{"documents": list})
}

// ───── frontend-side methods (Wails-bound on App) ─────

// MCPSetTaskChecked is called by the frontend when the user toggles an
// interactive task-list checkbox in an MCP-presented doc.
func (m *MCPManager) SetTaskChecked(docID string, taskID int, checked bool) {
	m.mu.Lock()
	doc, ok := m.docs[docID]
	if !ok {
		m.mu.Unlock()
		return
	}
	for i := range doc.Tasks {
		if doc.Tasks[i].ID == taskID {
			doc.Tasks[i].Checked = checked
			break
		}
	}
	doc.UpdatedAt = time.Now()
	m.mu.Unlock()
}

// MCPSetViewState records whether the user has the doc currently in view
// and/or has explicitly closed it.
func (m *MCPManager) SetViewState(docID string, viewed, closed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if doc, ok := m.docs[docID]; ok {
		doc.ViewedByUser = viewed
		if closed {
			doc.ClosedByUser = true
		}
	}
}

// ───── helpers ─────

func newMCPID() string {
	b := make([]byte, 9)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func mcpJSON(v any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

// inferTitle picks the first ATX H1 from the source, or returns "untitled".
func inferTitle(src string) string {
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(t[2:])
		}
	}
	return "untitled"
}

// extractTasks parses GFM task list items from source markdown in document
// order. If previous tasks are supplied, items whose text matches are
// preserved (so update_document doesn't lose user selections that still
// apply).
var taskRE = regexp.MustCompile(`^\s*[-*+]\s+\[([ xX])\]\s+(.+?)\s*$`)

func extractTasks(src string, previous []MCPTaskState) []MCPTaskState {
	prevByText := make(map[string]bool, len(previous))
	for _, p := range previous {
		prevByText[p.Text] = p.Checked
	}
	var tasks []MCPTaskState
	id := 0
	for _, line := range strings.Split(src, "\n") {
		m := taskRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		text := strings.TrimSpace(m[2])
		checked := m[1] == "x" || m[1] == "X"
		if v, ok := prevByText[text]; ok {
			checked = v
		}
		tasks = append(tasks, MCPTaskState{ID: id, Text: text, Checked: checked})
		id++
	}
	return tasks
}
