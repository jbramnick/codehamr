package tools

import (
	"context"
	"testing"
)

func TestGetCurrenDateSchema(t *testing.T) {
	schema := GetCurrenDateSchema()
	fn, ok := schema["function"].(map[string]any)
	if !ok {
		t.Fatal("missing function in schema")
	}
	name, ok := fn["name"].(string)
	if !ok || name != GetCurrenDateName {
		t.Fatalf("wrong name: %v", fn["name"])
	}
	params, ok := fn["parameters"].(map[string]any)
	if !ok {
		t.Fatal("missing parameters in schema")
	}
	required, ok := params["required"].([]string)
	if !ok || len(required) != 0 {
		t.Fatalf("expected no required fields, got: %v", required)
	}
}

func TestGetCurrenDateReturnsValidFormat(t *testing.T) {
	result := GetCurrenDate(context.Background())
	if result == "" {
		t.Fatal("got empty date string")
	}
	parts := splitParts(result, "-")
	if len(parts) != 3 {
		t.Fatalf("expected YYYY-MM-DD format, got: %s", result)
	}
	if len(parts[0]) != 4 || parts[1] < "01" || parts[1] > "12" {
		t.Fatalf("invalid date parts: %v in %s", parts, result)
	}
	if parts[2] < "01" || parts[2] > "31" {
		t.Fatalf("invalid day in %s", result)
	}
}

func TestGetCurrenDateCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := GetCurrenDate(ctx)
	if result == "" {
		t.Fatal("got empty result for cancelled context")
	}
}

func splitParts(s string, sep string) []string {
	var out []string
	start := 0
	for i := 0; ; i++ {
		idx := -1
		for j := start; j < len(s)-len(sep)+1; j++ {
			if s[j:j+len(sep)] == sep {
				idx = j
				break
			}
		}
		if idx == -1 {
			out = append(out, s[start:])
			break
		}
		out = append(out, s[start:idx])
		start = idx + len(sep)
	}
	return out
}
