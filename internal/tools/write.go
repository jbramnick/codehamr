package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile writes content to path, creating parent dirs. Errors return as
// part of the output string (bash convention), never as a Go error, so the
// model sees a write failure the way it sees a non-zero bash exit.
func WriteFile(path, content string) string {
	if path == "" {
		return "(empty path)"
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Sprintf("(mkdir error: %v)", err)
		}
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Sprintf("(write error: %v)", err)
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(content), path)
}

// WriteFileSchema is the OpenAI tool definition for write_file. The description
// steers the model away from bash heredocs (shell-quoting failure mode) for
// small-to-medium writes, and toward heredoc appends for large files (streamed
// tool-call args truncate server-side), mirroring the system prompt's rule so
// the two instruction channels can't contradict each other.
func WriteFileSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        WriteFileName,
			"description": "Write content bytes to a file at path. Creates parent directories. Overwrites existing files. Use this instead of bash heredocs for small-to-medium multi line content or content with single quotes, dollar signs, or backticks - no shell quoting issues. Content beyond a few hundred lines gets truncated by the server mid-stream: build large files with bash heredoc appends (cat > path <<'EOF' first, then cat >> path <<'EOF' per part) from the first call.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Absolute or relative file path. Relative paths resolve against the working directory.",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Exact bytes to write to the file.",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}
