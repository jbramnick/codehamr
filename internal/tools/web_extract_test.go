package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebExtractSchema(t *testing.T) {
	schema := WebExtractSchema()
	fn, ok := schema["function"].(map[string]any)
	if !ok {
		t.Fatal("missing function in schema")
	}
	name, ok := fn["name"].(string)
	if !ok || name != WebExtractName {
		t.Fatalf("wrong name: %v", fn["name"])
	}
	params, ok := fn["parameters"].(map[string]any)
	if !ok {
		t.Fatal("missing parameters in schema")
	}
	required, ok := params["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "url" {
		t.Fatalf("wrong required: %v", params["required"])
	}
}

func TestWebExtractUnavailable(t *testing.T) {
	prev := tavilyBaseURL
	tavilyBaseURL = ""
	defer func() { tavilyBaseURL = prev }()

	result := WebExtract(context.Background(), "http://example.com")
	if !strings.Contains(result, "unavailable") {
		t.Fatalf("expected unavailable message, got: %s", result)
	}
}

func TestWebExtractHappyPath(t *testing.T) {
	expected := `{"results":[{"url":"http://example.com","raw_content":"page content here"}]}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/extract" {
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

	result := WebExtract(context.Background(), "http://example.com")
	if result != expected {
		t.Fatalf("wrong result: %s", result)
	}
}

func TestWebExtractBadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	prev := tavilyBaseURL
	tavilyBaseURL = server.URL
	defer func() { tavilyBaseURL = prev }()

	result := WebExtract(context.Background(), "http://example.com")
	if !strings.Contains(result, "bad request") {
		t.Fatalf("expected bad request error, got: %s", result)
	}
}

func TestWebExtractAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	prev := tavilyBaseURL
	tavilyBaseURL = server.URL
	defer func() { tavilyBaseURL = prev }()

	result := WebExtract(context.Background(), "http://example.com")
	if !strings.Contains(result, "Tavily API error") {
		t.Fatalf("expected Tavily API error, got: %s", result)
	}
}

func TestWebExtractContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {}
	}))
	defer server.Close()

	prev := tavilyBaseURL
	tavilyBaseURL = server.URL
	defer func() { tavilyBaseURL = prev }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := WebExtract(ctx, "http://example.com")
	if !strings.Contains(result, "error") && !strings.Contains(result, "cancelled") {
		t.Fatalf("expected error or cancelled message, got: %s", result)
	}
}
