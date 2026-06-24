package providers

import "encoding/json"

type contextKey string

const (
	ProjectPathKey contextKey = "project_path"
	LogFuncKey     contextKey = "log_func"
	MissionIDKey   contextKey = "mission_id"
)

// Tool represents a single MCP tool definition.
type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Message is a single chat message (role + content).
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
