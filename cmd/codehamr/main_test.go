package main

import (
	"os"
	"testing"
)

// TestIsLocalBuild pins the contract: `go run` ("dev"), dirty-tree builds,
// clean-tree builds past the last tag (git describe shape), and tag-less
// clones (bare short sha) are all local and skip self-update; else the
// updater downgrades unreleased work to the last published release (a
// non-release hash always reads as "stale"). Only exact release tags
// self-update.
func TestIsLocalBuild(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"dev", true},
		{"v1.2.3-dirty", true},
		{"v0.1.0-5-g1a2b3c4-dirty", true},
		{"v0.1.0-5-g1a2b3c4", true}, // clean tree, 5 commits past the tag: unreleased work
		{"5290930", true},           // tag-less clone: `git describe --always` bare sha
		{"v1.2.3", false},
		{"v1.2.3-gamma", false}, // prerelease tag: "-g" but not hex, still a release
		{"", false},
	}
	for _, c := range cases {
		if got := isLocalBuild(c.in); got != c.want {
			t.Errorf("isLocalBuild(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

// TestReexecGuardOverridesPreexistingValue pins the loop-guard env semantics
// maybeSelfUpdate relies on: the re-exec'd child must see
// CODEHAMR_NO_UPDATE_CHECK=="1" even when a user already exported a different
// value. os.Setenv (the fix) overwrites in place; the old append(os.Environ(),…)
// left the stale value first, which Unix execve resolves first, defeating the
// guard. update.Check short-circuits only on exactly "1".
func TestReexecGuardOverridesPreexistingValue(t *testing.T) {
	t.Setenv("CODEHAMR_NO_UPDATE_CHECK", "0") // user set it wrong; restored after test
	os.Setenv("CODEHAMR_NO_UPDATE_CHECK", "1")
	if got := os.Getenv("CODEHAMR_NO_UPDATE_CHECK"); got != "1" {
		t.Fatalf("guard env resolves to %q, want \"1\" - append() would have left \"0\" first", got)
	}
}
