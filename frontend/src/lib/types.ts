export type Doc = {
  path: string
  name: string
  html: string
  source: string
  baseDir: string
  modified: number
  // MCP-presented documents come from an external LLM via the MCP server,
  // not from disk. They get interactive task lists and a "via mcp" badge.
  isMCP?: boolean
  mcpId?: string
  // Built-in docs (welcome) and MCP docs have no disk path — readOnly hides
  // the edit/save controls so users don't try to write to a synthetic path.
  readOnly?: boolean
}

export type MCPDoc = {
  id: string
  title: string
  source: string
  rendered: string
  presentedAt: string
  updatedAt: string
  tasks: Array<{ id: number; text: string; checked: boolean }>
}

export type MCPStatus = {
  enabled: boolean
  running: boolean
  port: number
  url: string
}

export type RecentEntry = {
  path: string
  name: string
  openedAt: number
}

export type FolderEntry = {
  name: string
  path: string
  isDir: boolean
  children?: FolderEntry[]
}

export type Folder = {
  root: string
  name: string
  entries: FolderEntry[]
}

export type TocItem = {
  id: string
  text: string
  level: number
}

export type Command = {
  id: string
  label: string
  hint?: string         // right-side text (kbd, date, etc.)
  group: 'action' | 'heading' | 'folder' | 'recent'
  run: () => void | Promise<void>
  matchText?: string    // text fuzzy-matched against (defaults to label)
}
