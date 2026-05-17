// mcp-test is a tiny verification client for upmark's MCP server.
//
// Usage:
//  1. Launch upmark
//  2. Open the command palette (Ctrl+K) → "mcp server: turn on"
//  3. From the upmark root directory, run:
//     go run ./cmd/mcp-test
//
// The script will connect to the SSE endpoint, push a sample markdown document
// (with a task list), wait, then poll for the user's task selections.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const sampleContent = `# Design review

Hi! I'm a language model writing to you through the MCP server. Take a look
at the proposal below and check whatever you agree with — I'll poll your
selections back through the MCP protocol.

## What's being proposed

A new color palette, a tightened sidebar, and the * removal * of the floppy
disk icon from the toolbar.

## Things to approve

- [ ] Switch the accent from blue to rust
- [ ] Adopt Newsreader for the reading body
- [ ] Drop the floppy disk save icon
- [ ] Add a settings pane in v0.8
- [x] Keep the command palette as the main surface

## Math (for fun)

The constraint we're optimizing for is roughly $\text{joy} \propto \text{type quality}$.

` + "```" + `mermaid
flowchart LR
  A[LLM] -- presents --> B[upmark]
  B -- user checkboxes --> C[status]
  C -- get_document_status --> A
` + "```" + `

Take your time.
`

func main() {
	c, err := client.NewSSEMCPClient("http://127.0.0.1:11451/sse")
	if err != nil {
		log.Fatal("new client: ", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.Start(ctx); err != nil {
		log.Fatal("start: ", err)
	}

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{Name: "mcp-test", Version: "0.1"}
	if _, err := c.Initialize(ctx, initRequest); err != nil {
		log.Fatal("initialize: ", err)
	}
	fmt.Println("✓ initialized")

	// Push a document
	callReq := mcp.CallToolRequest{}
	callReq.Params.Name = "present_document"
	callReq.Params.Arguments = map[string]any{
		"content": sampleContent,
		"title":   "Design review",
	}
	res, err := c.CallTool(ctx, callReq)
	if err != nil {
		log.Fatal("present_document: ", err)
	}
	var presentBody struct {
		DocumentID string `json:"document_id"`
		Title      string `json:"title"`
		TaskCount  int    `json:"task_count"`
	}
	if txt := firstTextContent(res); txt != "" {
		_ = json.Unmarshal([]byte(txt), &presentBody)
	}
	fmt.Printf("✓ presented doc id=%s with %d tasks — flip checkboxes in upmark\n",
		presentBody.DocumentID, presentBody.TaskCount)

	// Poll status every 5s for a minute
	deadline := time.Now().Add(60 * time.Second)
	statusReq := mcp.CallToolRequest{}
	statusReq.Params.Name = "get_document_status"
	statusReq.Params.Arguments = map[string]any{"document_id": presentBody.DocumentID}

	for time.Now().Before(deadline) {
		time.Sleep(5 * time.Second)
		r, err := c.CallTool(ctx, statusReq)
		if err != nil {
			fmt.Println("  status error:", err)
			continue
		}
		txt := firstTextContent(r)
		var status struct {
			Tasks []struct {
				ID      int    `json:"id"`
				Text    string `json:"text"`
				Checked bool   `json:"checked"`
			} `json:"tasks"`
			ViewedByUser bool `json:"viewedByUser"`
			ClosedByUser bool `json:"closedByUser"`
		}
		_ = json.Unmarshal([]byte(txt), &status)
		fmt.Printf("  viewing=%v closed=%v · ", status.ViewedByUser, status.ClosedByUser)
		for _, t := range status.Tasks {
			mark := " "
			if t.Checked {
				mark = "x"
			}
			fmt.Printf("[%s] %s · ", mark, t.Text)
		}
		fmt.Println()
		if status.ClosedByUser {
			fmt.Println("✓ user closed the document — exiting")
			return
		}
	}
}

func firstTextContent(r *mcp.CallToolResult) string {
	for _, c := range r.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}
