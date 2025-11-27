package tools

import (
	"context"
)

// ToolContext brings metadata of the call to the tool
type ToolContext struct {
	UserID    string
	SessionID string
	RequestID string
}

// Tool represents a tool agents can invoke
// input/output is a generic map to maintain flexibility.
type Tool interface {
	Name() string
	Call(ctx context.Context, tctx ToolContext, input map[string]any) (map[string]any, error)
}
