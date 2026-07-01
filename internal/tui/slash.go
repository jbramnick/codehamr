package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jbramnick/codehamr/internal/cloud"
	chmctx "github.com/jbramnick/codehamr/internal/ctx"
	"github.com/jbramnick/codehamr/internal/config"
	"github.com/jbramnick/codehamr/internal/llm"
)

// argOption is one popover entry, used at command-level (one row per command)
// and argument-level (one row per accepted value for the active command).
type argOption struct {
	value       string // what gets inserted / committed
	description string // right-aligned help text
	current     bool   // rendered bold; default-selected when the popover opens
}

// command is one row in the popover, --help, and the dispatch table.
// args, if non-nil, supplies the argument-level popover entries.
type command struct {
	name        string
	description string
	handler     func(Model, []string) (tea.Model, tea.Cmd)
	args        func(Model) []argOption
}

// commands lists every slash command, in popover/--help order. Keep it short.
var commands = []command{
	{
		name:        "/export",
		description: "export conversation summary to hamr_session_export.md",
		handler:     (Model).cmdExport,
	},
	{
		name:        "/import",
		description: "load hamr_session_export.md into context and delete it",
		handler:     (Model).cmdImport,
	},
	{
		name:        "/clear",
		description: "reset the conversation",
		handler:     (Model).cmdClear,
	},
	{
		name:        "/models",
		description: "list · <name> set (Tab cycles in the popover)",
		handler:     (Model).cmdModel,
		args: func(m Model) []argOption {
			out := make([]argOption, 0, len(m.cfg.Models))
			for _, n := range m.cfg.ModelNames() {
				p := m.cfg.Models[n]
				out = append(out, argOption{
					value:       n,
					description: p.LLM + " @ " + p.URL,
					current:     n == m.cfg.Active,
				})
			}
			return out
		},
	},
}

// commandByName returns the registered command for a slash name, or nil.
// Centralises the linear scan shared by completion, dispatch, and runSlash.
func commandByName(name string) *command {
	for i := range commands {
		if commands[i].name == name {
			return &commands[i]
		}
	}
	return nil
}

// runSlash dispatches a slash-prefixed submission; unknown commands produce a
// quiet hint, not an error. config.yaml is re-read before every slash so
// hand-edits take effect without a restart (see reloadConfigFromDisk).
func (m Model) runSlash(text string) (tea.Model, tea.Cmd) {
	if err := m.reloadConfigFromDisk(); err != nil {
		m.appendLine(styleWarn.Render("⚠ " + err.Error()))
	}
	fields := strings.Fields(text)
	if c := commandByName(fields[0]); c != nil {
		return c.handler(m, fields[1:])
	}
	m.appendLine(styleWarn.Render("unknown command - type / to see options"))
	return m, nil
}

// reloadConfigFromDisk re-runs config.Bootstrap and replaces m.cfg so hand-edits
// to config.yaml between slash commands take effect immediately. URLOverride
// (from CODEHAMR_URL) is carried across the swap so the env var keeps applying.
//
// Returns the Bootstrap error verbatim; callers decide whether to surface it
// (runSlash warns on submit; the popover-refresh path ignores it so a broken
// file doesn't spam a warning on every keystroke).
//
// Rebuilds the llm.Client when the active profile's resolved (URL, model, key)
// triple changed: covers both within-profile edits and a moved active.
func (m *Model) reloadConfigFromDisk() error {
	projectRoot := filepath.Dir(m.cfg.Dir)
	fresh, _, err := config.Bootstrap(projectRoot)
	if err != nil {
		return err
	}
	fresh.URLOverride = m.cfg.URLOverride

	prevURL := m.cfg.ActiveURL()
	prevProfile := m.cfg.ActiveProfile()
	prevLLM, prevKey := prevProfile.LLM, prevProfile.ResolvedKey()

	m.cfg = fresh

	newProfile := m.cfg.ActiveProfile()
	if prevURL != m.cfg.ActiveURL() || prevLLM != newProfile.LLM || prevKey != newProfile.ResolvedKey() {
		m.rebuildClient()
	}
	return nil
}

// PrintHelp writes the canonical human-readable command list. Used by --help.
func PrintHelp(out io.Writer) {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	for _, c := range commands {
		fmt.Fprintf(w, "  %s\t%s\n", c.name, c.description)
	}
	w.Flush()
}

// --- handlers ---------------------------------------------------------------

// cmdModel: `/models` lists, `/models <name>` sets. Cycling is Tab/Shift+Tab
// in the popover, no separate "next" command.
func (m Model) cmdModel(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		m.printModelList()
		return m, nil
	}
	if err := m.cfg.SetActive(args[0]); err != nil {
		m.appendLine(styleError.Render("⚠ " + err.Error()))
		return m, nil
	}
	m.rebuildClient()
	return m, m.confirmActive(args[0])
}

// printModelList writes the "▸ active, name, llm @ url" rollup to scroll.
func (m *Model) printModelList() {
	m.appendLine(styleDim.Render("models (▸ active, /models <name> to switch):"))
	for _, n := range m.cfg.ModelNames() {
		mark := "  "
		if n == m.cfg.Active {
			mark = "▸ "
		}
		p := m.cfg.Models[n]
		m.appendLine(fmt.Sprintf("%s%s  %s",
			mark, n, styleDim.Render(p.LLM+" @ "+p.URL)))
	}
}

// confirmActive emits the activation line for the active profile and returns
// its reachability cmd. Keyed profiles (cloud) probe: the success line is
// delayed until the response arrives so it can carry the live ctx window from
// X-Context-Window. Keyless profiles (local Ollama) ping and print
// synchronously. Shared by /models.
func (m *Model) confirmActive(profile string) tea.Cmd {
	p := m.cfg.ActiveProfile()
	if p.ResolvedKey() != "" {
		m.appendLine(styleDim.Render(fmt.Sprintf("▶ probing %s · %s @ %s", profile, p.LLM, p.URL)))
		return probeBackend(m.cli, profile, false)
	}
	m.appendLine(styleOK.Render(fmt.Sprintf("✓ active: %s · %s @ %s", profile, p.LLM, p.URL)))
	return pingBackend(m.cli.BaseURL)
}

// rebuildClient swaps in a fresh llm.Client for the now-active profile.
// Replacing the pointer (not mutating fields) drops the prior Client's sticky
// state (noReasoningEffort, keep-alive pool tied to the old URL): new
// endpoint, fresh slate.
func (m *Model) rebuildClient() {
	p := m.cfg.ActiveProfile()
	m.cli = llm.New(m.cfg.ActiveURL(), p.LLM, p.ResolvedKey())
	// Drop the prior profile's cached BudgetStatus. m.budget has no profile
	// association, so without this reset the footer keeps rendering the old
	// "88% pass" segment after switching to a local profile that emits no
	// X-Budget-* headers (nothing would overwrite it). A fresh BudgetStatus{}
	// hides the segment until the new backend reports its own.
	m.budget = cloud.BudgetStatus{}
}

func (m Model) cmdClear(_ []string) (tea.Model, tea.Cmd) {
	m.history = nil
	m.scroll.Reset()
	m.sessionTokens = 0
	m.streamingEstimate = 0
	// Drop any queued follow-up: it targeted the conversation just wiped.
	m.queued = nil
	// Reset the repeated-failure streak so the next turn starts clean.
	m.failKey, m.failStreak = "", 0
	// Wipe prompt recall too: in-memory ring and on-disk .codehamr/history,
	// or leftover history would contradict the "fresh start" promise.
	m.promptHistory = nil
	m.histIdx = -1
	_ = clearPromptHistory(m.cfg.Dir)
	// Full wipe (unlike Ctrl+L, which redraws but keeps scrollback).
	// tea.ClearScreen emits \x1b[2J, which only wipes the viewport; the
	// saved-lines buffer needs eraseScrollback (DECSED 3) too, or old replies
	// stay scrollable above the reset line. tea.Sequence keeps the print from
	// racing past the clear (tea.Batch runs both concurrently and the print
	// could land first, then get wiped). scroll keeps the line for resize
	// replay; outbox is cleared because the Sequence owns the print now.
	line := styleOK.Render("✓ conversation reset")
	m.scroll.WriteString(line + "\n")
	m.outbox = nil
	return m, tea.Sequence(tea.ClearScreen, eraseScrollback, tea.Println(line))
}

// cmdExport writes a markdown summary of the current conversation to
// hamr_session_export.md at the project root.
func (m Model) cmdExport(_ []string) (tea.Model, tea.Cmd) {
	projectRoot := filepath.Dir(m.cfg.Dir)
	path := filepath.Join(projectRoot, "hamr_session_export.md")

	var sb strings.Builder
	sb.WriteString("# Session Export\n\n")
	for i, msg := range m.history {
		switch msg.Role {
		case chmctx.RoleUser:
			sb.WriteString("## User\n\n```\n")
		case chmctx.RoleAssistant:
			sb.WriteString("## Assistant\n\n```\n")
		case chmctx.RoleTool:
			sb.WriteString(fmt.Sprintf("## Tool (%s)\n\n```\n", msg.ToolName))
		default:
			sb.WriteString("## System\n\n```\n")
		}
		sb.WriteString(msg.Content)
		if !strings.HasSuffix(msg.Content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("```\n\n")
		_ = i // keep for potential future use (line numbers)
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		m.appendLine(styleError.Render("⚠ export failed: " + err.Error()))
		return m, nil
	}
	m.appendLine(styleOK.Render(fmt.Sprintf("✓ exported %d messages to hamr_session_export.md", len(m.history))))
	return m, nil
}

// cmdImport reads hamr_session_export.md from the project root, loads its
// contents as a user message into history, and deletes the file.
func (m Model) cmdImport(_ []string) (tea.Model, tea.Cmd) {
	projectRoot := filepath.Dir(m.cfg.Dir)
	path := filepath.Join(projectRoot, "hamr_session_export.md")

	data, err := os.ReadFile(path)
	if err != nil {
		m.appendLine(styleError.Render("⚠ import failed: no hamr_session_export.md found at project root"))
		return m, nil
	}

	content := string(data)
	m.history = append(m.history, chmctx.Message{Role: chmctx.RoleUser, Content: content})

	if err := os.Remove(path); err != nil {
		m.appendLine(styleWarn.Render("⚠ imported but could not delete file: " + err.Error()))
	} else {
		m.appendLine(styleOK.Render("✓ imported hamr_session_export.md into context"))
	}
	return m, nil
}
