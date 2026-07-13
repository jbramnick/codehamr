package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const WebExtractName = "web_extract"

// WebExtractSchema returns the OpenAI tool definition for web_extract.
func WebExtractSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        WebExtractName,
			"description": "Extracts clean page content from a single URL. Use it when you have a specific URL and want its readable text/markdown. Call once per URL for multiple pages.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The single URL to extract content from",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}

// WebExtract executes content extraction via the Tavily API.
func WebExtract(ctx context.Context, url string) string {
	if tavilyBaseURL == "" {
		return "(web_extract unavailable: set tavily_base_url in .jimmyhamr/config.yaml)"
	}

	body := map[string]string{"urls": url}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Sprintf("(web_extract error: failed to marshal request: %v)", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tavilyBaseURL+"/extract", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Sprintf("(web_extract error: failed to create request: %v)", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("(web_extract error: request failed: %v)", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Sprintf("(web_extract error: failed to read response: %v)", err)
		}
		return string(b)
	case http.StatusBadRequest:
		return "(web_extract error: bad request)"
	case http.StatusBadGateway, http.StatusInternalServerError:
		return "(web_extract error: Tavily API error)"
	default:
		return fmt.Sprintf("(web_extract error: HTTP %d)", resp.StatusCode)
	}
}
