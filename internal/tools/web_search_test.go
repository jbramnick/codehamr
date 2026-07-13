package tools

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebSearchSchema(t *testing.T) {
	schema := WebSearchSchema()
	fn, ok := schema["function"].(map[string]any)
	if !ok {
		t.Fatal("missing function in schema")
	}
	name, ok := fn["name"].(string)
	if !ok || name != WebSearchName {
		t.Fatalf("wrong name: %v", fn["name"])
	}
	params, ok := fn["parameters"].(map[string]any)
	if !ok {
		t.Fatal("missing parameters in schema")
	}
	required, ok := params["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "query" {
		t.Fatalf("wrong required: %v", params["required"])
	}
}

func TestWebSearchUnavailable(t *testing.T) {
	prev := tavilyBaseURL
	tavilyBaseURL = ""
	defer func() { tavilyBaseURL = prev }()

	result := WebSearch(context.Background(), "test query")
	if !strings.Contains(result, "unavailable") {
		t.Fatalf("expected unavailable message, got: %s", result)
	}
}

func TestWebSearchDateInjected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		wantBody := `"query":"test query as of 20`
		if !strings.Contains(string(body), wantBody) {
			t.Fatalf("expected date injection, got body: %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	prev := tavilyBaseURL
	tavilyBaseURL = server.URL
	defer func() { tavilyBaseURL = prev }()

	result := WebSearch(context.Background(), "test query")
	if result != `{}` {
		t.Fatalf("expected success, got: %s", result)
	}
}

func TestWebSearchNoDoubleInjection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		// Should contain "as of" only once (injected), not twice.
		count := strings.Count(string(body), `"query":"`)
		if count != 1 {
			t.Fatalf("expected exactly one query field, got body: %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	prev := tavilyBaseURL
	tavilyBaseURL = server.URL
	defer func() { tavilyBaseURL = prev }()

	result := WebSearch(context.Background(), "latest news as of 2025")
	if result != `{}` {
		t.Fatalf("expected success, got: %s", result)
	}
}

func TestWebSearchHappyPath(t *testing.T) {
	expected := `{"results":[{"title":"Test","url":"http://example.com","content":"test content"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/search" {
			t.Fatalf("wrong request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expected))
	}))
	defer server.Close()

	prev := tavilyBaseURL
	tavilyBaseURL = server.URL
	defer func() { tavilyBaseURL = prev }()

	result := WebSearch(context.Background(), "test query")
	if result != expected {
		t.Fatalf("wrong result: %s", result)
	}
}

func TestWebSearchBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	prev := tavilyBaseURL
	tavilyBaseURL = server.URL
	defer func() { tavilyBaseURL = prev }()

	result := WebSearch(context.Background(), "test query")
	if !strings.Contains(result, "bad request") {
		t.Fatalf("expected bad request error, got: %s", result)
	}
}

func TestWebSearchAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	prev := tavilyBaseURL
	tavilyBaseURL = server.URL
	defer func() { tavilyBaseURL = prev }()

	result := WebSearch(context.Background(), "test query")
	if !strings.Contains(result, "Tavily API error") {
		t.Fatalf("expected Tavily API error, got: %s", result)
	}
}

func TestWebSearchContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer server.Close()

	prev := tavilyBaseURL
	tavilyBaseURL = server.URL
	defer func() { tavilyBaseURL = prev }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := WebSearch(ctx, "test query")
	if !strings.Contains(result, "error") && !strings.Contains(result, "cancelled") {
		t.Fatalf("expected error or cancelled message, got: %s", result)
	}
}
