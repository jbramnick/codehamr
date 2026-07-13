package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const WebSearchName = "web_search"

// WebSearchSchema returns the OpenAI tool definition for web_search.
func WebSearchSchema() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        WebSearchName,
			"description": "Searches the web for current information. Today's date is automatically appended for recency ranking — you don't need to include a year in your query. For accurate date references in your own analysis, call `get_current_date` first. Returns ranked results with titles, URLs, and content snippets.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query string",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// WebSearch executes a web search via the Tavily API.
func WebSearch(ctx context.Context, query string) string {
	if tavilyBaseURL == "" {
		return "(web_search unavailable: set tavily_base_url in .jimmyhamr/config.yaml)"
	}

	// Inject current date for recency ranking. The schema tells the model this
	// is automatic, so it doesn't need to include dates in its query.
	if !strings.Contains(strings.ToLower(query), "as of") {
		query = fmt.Sprintf("%s as of %s", query, time.Now().Format("2006-01-02"))
	}

	body := map[string]string{"query": query}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Sprintf("(web_search error: failed to marshal request: %v)", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", tavilyBaseURL+"/search", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Sprintf("(web_search error: failed to create request: %v)", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("(web_search error: request failed: %v)", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Sprintf("(web_search error: failed to read response: %v)", err)
		}
		return string(b)
	case http.StatusBadRequest:
		return "(web_search error: bad request)"
	case http.StatusBadGateway, http.StatusInternalServerError:
		return "(web_search error: Tavily API error)"
	default:
		return fmt.Sprintf("(web_search error: HTTP %d)", resp.StatusCode)
	}
}
