package tui

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
)

// Prompt history persists across restarts in .codehamr/history alongside
// the project's other state, so recall is per-project (cd-stable) and
// /clear can wipe it as part of the project-scoped reset. One
// strconv-quoted entry per line keeps the format dumb (cat-friendly),
// handles multi-line prompts without a separator, and lets a corrupt line
// be skipped without poisoning the rest of the file.
const (
	historyFileName   = "history"
	historyMaxEntries = 500
	// historyMaxEntryBytes caps a single quoted line. Pasted attachments
	// arrive as chips in the live UI but the on-disk history still stores
	// their expanded text, so without this cap a multi-megabyte log paste
	// would balloon the history file every submit and eventually exceed
	// the 1 MiB scanner buffer in loadPromptHistory — at which point the
	// entry would silently disappear on the next load. 256 KiB comfortably
	// holds any sane prompt; longer pastes are simply not recalled, which
	// is the right tradeoff for a dumb cat-friendly store.
	historyMaxEntryBytes = 256 * 1024
)

func historyPath(dir string) string { return filepath.Join(dir, historyFileName) }

// loadPromptHistory returns every saved prompt as a chip-less promptEntry,
// oldest first to match the in-memory append order so historyUp/Down walk
// the same direction whether entries were just typed or came off disk. A
// missing file is the first-run state, not an error.
func loadPromptHistory(dir string) []promptEntry {
	f, err := os.Open(historyPath(dir))
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []promptEntry
	sc := bufio.NewScanner(f)
	// One prompt may carry a pasted log of tens of KB; raise the per-line
	// cap well past Scanner's 64KB default so we don't silently drop the
	// tail of a long entry on load.
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		v, err := strconv.Unquote(sc.Text())
		if err != nil {
			continue
		}
		out = append(out, promptEntry{display: v})
	}
	return out
}

// appendPromptHistory writes value as one more entry, trimming the file to
// historyMaxEntries so on-disk growth stays bounded. Empty prompts are
// skipped so stray ↵ presses don't pollute recall, and entries longer
// than historyMaxEntryBytes are dropped on the floor (recall would have
// silently failed anyway thanks to the 1 MiB scanner cap; declining to
// store is the honest answer).
//
// The append uses O_APPEND so two codehamr processes in the same project
// can each add a line without one's "load + rewrite" silently eating the
// other's submit. The trim-to-historyMaxEntries pass is wrapped in
// best-effort: on any IO error during the rewrite we leave the appended
// file alone — the user keeps their newest entries, and the next start
// will trim back down to the limit on the natural rewrite path.
func appendPromptHistory(dir, value string) error {
	if value == "" {
		return nil
	}
	if len(value) > historyMaxEntryBytes {
		return nil
	}
	line := strconv.Quote(value) + "\n"
	path := historyPath(dir)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(line); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	// Lazy trim: only when the file has clearly outgrown the cap. Counting
	// lines is O(file) but bounded — we cap fast-path opportunism at 4× the
	// limit, past which we always rewrite.
	count, err := countHistoryLines(path)
	if err != nil || count <= historyMaxEntries {
		return nil
	}
	all := loadPromptHistory(dir)
	if len(all) > historyMaxEntries {
		all = all[len(all)-historyMaxEntries:]
	}
	var buf []byte
	for _, e := range all {
		buf = append(buf, strconv.Quote(e.display)...)
		buf = append(buf, '\n')
	}
	// Best-effort rewrite — failures here keep the over-cap-but-correct
	// file rather than reporting an error that would obscure the
	// successful append above.
	_ = os.WriteFile(path, buf, 0o600)
	return nil
}

// countHistoryLines tallies newline-terminated lines without parsing them
// — fast path for the trim-decision in appendPromptHistory.
func countHistoryLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	n := 0
	for sc.Scan() {
		n++
	}
	return n, sc.Err()
}

// clearPromptHistory removes the on-disk file so /clear's full-reset
// gesture also wipes prompt recall. A missing file is not an error — the
// caller's intent (an empty history) is already satisfied.
func clearPromptHistory(dir string) error {
	err := os.Remove(historyPath(dir))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
