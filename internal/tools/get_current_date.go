package tools

import (
	"context"
	"fmt"
	"time"
)

// GetCurrenDateSchema returns the OpenAI tool definition for get_current_date.
func GetCurrenDateSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        GetCurrenDateName,
			"description": "Returns today's date as YYYY-MM-DD. Call this once at the start of a turn so your analysis uses accurate timing.",
			"parameters": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
				"required":   []string{},
			},
		},
	}
}

// GetCurrenDate returns today's date formatted as YYYY-MM-DD. Context is accepted for signature consistency but never used (instant operation).
func GetCurrenDate(ctx context.Context) string {
	if ctx != nil && ctx.Err() != nil {
		return fmt.Sprintf("(get_current_date error: %v)", ctx.Err())
	}
	return time.Now().Format("2006-01-02")
}
