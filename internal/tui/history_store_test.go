package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPromptHistoryRoundTrip: append a few values (including one with a
// newline and one with quotes), reload from disk, expect identical
// payloads in append order. Anchors the contract that the dumb on-disk
// format survives every byte the textarea can submit.
func TestPromptHistoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if got := loadPromptHistory(dir); len(got) != 0 {
		t.Fatalf("fresh dir should have 0 entries, got %d", len(got))
	}
	want := []string{"first", "second\nwith newline", `third "quoted"`, "café 🐹"}
	for _, v := range want {
		if err := appendPromptHistory(dir, v); err != nil {
			t.Fatal(err)
		}
	}
	got := loadPromptHistory(dir)
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].display != w {
			t.Errorf("entry %d: got %q, want %q", i, got[i].display, w)
		}
	}
}

// TestPromptHistoryCap: exceeding historyMaxEntries drops the oldest, not
// the newest. Without this guarantee the file grows unbounded over a
// project's lifetime.
func TestPromptHistoryCap(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < historyMaxEntries+50; i++ {
		if err := appendPromptHistory(dir, "p"+strings.Repeat("x", i%3)); err != nil {
			t.Fatal(err)
		}
	}
	got := loadPromptHistory(dir)
	if len(got) != historyMaxEntries {
		t.Fatalf("cap not enforced: %d entries on disk, want %d", len(got), historyMaxEntries)
	}
	// First on-disk entry should correspond to the 50th submitted prompt
	// (oldest 50 dropped) — proves trim is from the head, not the tail.
	wantHead := "p" + strings.Repeat("x", 50%3)
	if got[0].display != wantHead {
		t.Errorf("trim direction wrong: head=%q want %q", got[0].display, wantHead)
	}
}

// TestPromptHistorySkipEmpty: empty submits never reach the file. Stops a
// stray ↵ from polluting the recall list with blanks.
func TestPromptHistorySkipEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := appendPromptHistory(dir, ""); err != nil {
		t.Fatal(err)
	}
	if got := loadPromptHistory(dir); len(got) != 0 {
		t.Errorf("empty prompt should not be saved, got %d", len(got))
	}
	if _, err := os.Stat(filepath.Join(dir, historyFileName)); !os.IsNotExist(err) {
		t.Errorf("history file should not exist after only-empty append, err=%v", err)
	}
}

// TestPromptHistoryClear: clearPromptHistory removes the file, and a
// missing file is not an error. Mirrors the /clear semantics.
func TestPromptHistoryClear(t *testing.T) {
	dir := t.TempDir()
	if err := appendPromptHistory(dir, "x"); err != nil {
		t.Fatal(err)
	}
	if err := clearPromptHistory(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, historyFileName)); !os.IsNotExist(err) {
		t.Fatalf("history file should be gone, err=%v", err)
	}
	// Idempotent: clearing an already-clean dir is a no-op.
	if err := clearPromptHistory(dir); err != nil {
		t.Errorf("second clear must be no-op, got %v", err)
	}
}

// TestPromptHistoryCorruptLineSkipped: a malformed line should not
// poison subsequent valid entries. Important because users may edit the
// file by hand and we should not wipe their good entries on a typo.
func TestPromptHistoryCorruptLineSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, historyFileName)
	body := "not a quoted line\n" + `"valid"` + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got := loadPromptHistory(dir)
	if len(got) != 1 || got[0].display != "valid" {
		t.Errorf("expected 1 valid entry, got %+v", got)
	}
}

// TestPromptHistoryConcurrentAppendsKeepBoth is the regression for the
// load-then-rewrite race: when two codehamr instances (same project dir)
// submit prompts simultaneously, both readers saw the same N entries,
// each appended their own, and the second writer's full-file overwrite
// silently dropped the first one's submit. With O_APPEND each instance
// only writes its own line and both survive.
func TestPromptHistoryConcurrentAppendsKeepBoth(t *testing.T) {
	dir := t.TempDir()

	const n = 50
	done := make(chan struct{}, 2)
	go func() {
		for i := 0; i < n; i++ {
			_ = appendPromptHistory(dir, fmt.Sprintf("alpha-%d", i))
		}
		done <- struct{}{}
	}()
	go func() {
		for i := 0; i < n; i++ {
			_ = appendPromptHistory(dir, fmt.Sprintf("beta-%d", i))
		}
		done <- struct{}{}
	}()
	<-done
	<-done

	got := loadPromptHistory(dir)
	if len(got) != 2*n {
		t.Fatalf("concurrent appends lost entries: got %d/%d", len(got), 2*n)
	}
	seen := map[string]bool{}
	for _, e := range got {
		seen[e.display] = true
	}
	for i := 0; i < n; i++ {
		if !seen[fmt.Sprintf("alpha-%d", i)] {
			t.Fatalf("alpha-%d missing from history", i)
		}
		if !seen[fmt.Sprintf("beta-%d", i)] {
			t.Fatalf("beta-%d missing from history", i)
		}
	}
}

// TestPromptHistoryRejectsHugeEntry: a multi-MiB paste must not get
// stored verbatim — the prior load path used a 1 MiB scanner cap and
// would silently drop oversized entries on next load anyway, so
// declining to store is the consistent behaviour. Anything sane
// (a paragraph of code, a stack trace) still survives.
func TestPromptHistoryRejectsHugeEntry(t *testing.T) {
	dir := t.TempDir()
	huge := strings.Repeat("x", historyMaxEntryBytes+1)
	if err := appendPromptHistory(dir, huge); err != nil {
		t.Fatal(err)
	}
	got := loadPromptHistory(dir)
	if len(got) != 0 {
		t.Fatalf("oversized entry should not be saved, got %d", len(got))
	}
	// At-the-cap entry still saves.
	atCap := strings.Repeat("y", historyMaxEntryBytes)
	if err := appendPromptHistory(dir, atCap); err != nil {
		t.Fatal(err)
	}
	got = loadPromptHistory(dir)
	if len(got) != 1 || got[0].display != atCap {
		t.Fatalf("at-cap entry should round-trip, got %d entries", len(got))
	}
}
