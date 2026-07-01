package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jbramnick/codehamr/internal/config"
)

func TestBuildSystemWithoutAgentsMD(t *testing.T) {
	dir := t.TempDir()
	sys := buildSystem(dir)

	if !strings.Contains(sys, config.DefaultSystemPrompt) {
		t.Error("expected embedded system prompt in output")
	}
	if !strings.Contains(sys, "Working directory: "+dir) {
		t.Error("expected working directory anchor in output")
	}
	if strings.Contains(sys, "AGENTS.md") {
		t.Error("should not contain AGENTS.md content when file is absent")
	}
}

func TestBuildSystemWithAgentsMD(t *testing.T) {
	dir := t.TempDir()
	agentsPath := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte("# Repo Conventions\nUse tabs for indentation."), 0o644); err != nil {
		t.Fatal(err)
	}

	sys := buildSystem(dir)

	if !strings.Contains(sys, config.DefaultSystemPrompt) {
		t.Error("expected embedded system prompt in output")
	}
	if !strings.Contains(sys, "# Repo Conventions\nUse tabs for indentation.") {
		t.Error("expected AGENTS.md content in output")
	}
	if !strings.Contains(sys, "Working directory: "+dir) {
		t.Error("expected working directory anchor in output")
	}

	// Verify ordering: prompt before agents.md before working dir
	promptIdx := strings.Index(sys, config.DefaultSystemPrompt)
	agentsIdx := strings.Index(sys, "# Repo Conventions")
	dirIdx := strings.Index(sys, "Working directory:")
	if !(promptIdx < agentsIdx && agentsIdx < dirIdx) {
		t.Errorf("expected order prompt < agents.md < working-dir, got %d < %d < %d", promptIdx, agentsIdx, dirIdx)
	}
}

func TestBuildSystemWithEmptyAgentsMD(t *testing.T) {
	dir := t.TempDir()
	agentsPath := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(agentsPath, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	sys := buildSystem(dir)
	if strings.Contains(sys, "AGENTS.md") {
		t.Error("empty AGENTS.md should not add content")
	}
}
