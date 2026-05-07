package cloud

import (
	"net/http"
	"testing"
)

func TestFromHeadersMissing(t *testing.T) {
	if got := FromHeaders(http.Header{}); got != (BudgetStatus{}) {
		t.Fatalf("missing X-Budget-Remaining must produce zero BudgetStatus, got %+v", got)
	}
}

func TestFromHeadersClampsAndParses(t *testing.T) {
	cases := []struct {
		name string
		val  string
		set  bool
		want float64
	}{
		{"fresh", "1.0", true, 1.0},
		{"mid", "0.73", true, 0.73},
		{"depleted", "0", true, 0.0},
		{"clamped above 1", "1.05", true, 1.0},
		{"clamped below 0", "-0.1", true, 0.0},
		{"garbage", "abc", false, 0.0},
		{"empty", "", false, 0.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			if tc.val != "" {
				h.Set("X-Budget-Remaining", tc.val)
			}
			got := FromHeaders(h)
			if got.Set != tc.set {
				t.Fatalf("Set: got %v want %v", got.Set, tc.set)
			}
			if got.Remaining != tc.want {
				t.Fatalf("Remaining: got %v want %v", got.Remaining, tc.want)
			}
		})
	}
}

// TestFromHeadersRejectsNaNAndInf is the regression: strconv.ParseFloat
// happily returns NaN/±Inf without an error, all NaN comparisons evaluate
// to false (so the [0,1] clamp is a no-op), and the resulting "valid"
// BudgetStatus then renders through int(NaN*100+0.5) → MinInt64 in the
// status bar. Both must collapse to the same zero value as a missing header.
func TestFromHeadersRejectsNaNAndInf(t *testing.T) {
	cases := []string{"NaN", "+Inf", "-Inf", "Infinity"}
	for _, raw := range cases {
		t.Run(raw, func(t *testing.T) {
			h := http.Header{}
			h.Set("X-Budget-Remaining", raw)
			got := FromHeaders(h)
			if got.Set {
				t.Fatalf("X-Budget-Remaining=%q must be treated as missing, got %+v", raw, got)
			}
		})
	}
}

func TestStatusSuffix(t *testing.T) {
	if got := (BudgetStatus{}).StatusSuffix(); got != "" {
		t.Fatalf("zero value must render empty, got %q", got)
	}
	cases := []struct {
		remaining float64
		want      string
	}{
		{1.0, " · 100% pass"},
		{0.73, " · 73% pass"},
		{0.005, " · 1% pass"},
		{0.0, " · 0% pass"},
	}
	for _, tc := range cases {
		got := BudgetStatus{Set: true, Remaining: tc.remaining}.StatusSuffix()
		if got != tc.want {
			t.Fatalf("Remaining=%v: got %q want %q", tc.remaining, got, tc.want)
		}
	}
}

func TestContextWindowFromHeaders(t *testing.T) {
	cases := []struct {
		name string
		val  string
		want int
	}{
		{"missing", "", 0},
		{"valid", "262144", 262144},
		{"min boundary", "1024", 1024},
		{"too small", "1023", 0},
		{"too large", "9999999999", 0},
		{"negative", "-1", 0},
		{"garbage", "abc", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			if tc.val != "" {
				h.Set("X-Context-Window", tc.val)
			}
			if got := ContextWindowFromHeaders(h); got != tc.want {
				t.Fatalf("want %d, got %d", tc.want, got)
			}
		})
	}
}

func TestAuthHeader(t *testing.T) {
	if AuthHeader("hp_abc") != "Bearer hp_abc" {
		t.Fatal("bad auth header")
	}
}
