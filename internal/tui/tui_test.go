package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yasomaru/git-wt/internal/git"
)

// testWorktrees returns a reusable slice of worktrees for test setup.
// Index 0 is the current (main) worktree, index 1 is a merged feature branch,
// index 2 is an unmerged feature branch, and index 3 is a dirty worktree.
func testWorktrees() []git.Worktree {
	return []git.Worktree{
		{
			Path:      "/repo",
			Head:      "aaaa1111bbbb2222cccc3333dddd4444eeee5555",
			Branch:    "refs/heads/main",
			IsCurrent: true,
		},
		{
			Path:     "/repo-feature-a",
			Head:     "bbbb2222cccc3333dddd4444eeee5555ffff6666",
			Branch:   "refs/heads/feature-a",
			IsMerged: true,
		},
		{
			Path:   "/repo-feature-b",
			Head:   "cccc3333dddd4444eeee5555ffff6666aaaa1111",
			Branch: "refs/heads/feature-b",
		},
		{
			Path:      "/repo-feature-c",
			Head:      "dddd4444eeee5555ffff6666aaaa1111bbbb2222",
			Branch:    "refs/heads/feature-c",
			Modified:  3,
			Untracked: 1,
		},
	}
}

func keyMsg(r rune) tea.Msg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func specialKeyMsg(k tea.KeyType) tea.Msg {
	return tea.KeyMsg{Type: k}
}

func updateModel(t *testing.T, m model, msg tea.Msg) model {
	t.Helper()
	result, _ := m.Update(msg)
	updated, ok := result.(model)
	if !ok {
		t.Fatalf("Update did not return a model, got %T", result)
	}
	return updated
}

func TestNew(t *testing.T) {
	t.Parallel()

	wts := testWorktrees()
	m := New(wts, "/repo")

	if len(m.items) != len(wts) {
		t.Fatalf("New() created %d items, want %d", len(m.items), len(wts))
	}
	for i, it := range m.items {
		if it.worktree.Branch != wts[i].Branch {
			t.Errorf("item[%d].worktree.Branch = %q, want %q", i, it.worktree.Branch, wts[i].Branch)
		}
		if it.checked {
			t.Errorf("item[%d].checked = true, want false on initialization", i)
		}
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.mode != modeList {
		t.Errorf("mode = %d, want modeList (%d)", m.mode, modeList)
	}
	if m.repoDir != "/repo" {
		t.Errorf("repoDir = %q, want %q", m.repoDir, "/repo")
	}
	if m.confirmCursor != 0 {
		t.Errorf("confirmCursor = %d, want 0", m.confirmCursor)
	}
}

func TestCursorMovement(t *testing.T) {
	t.Parallel()

	t.Run("j moves cursor down", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")

		m = updateModel(t, m, keyMsg('j'))
		if m.cursor != 1 {
			t.Errorf("after 'j': cursor = %d, want 1", m.cursor)
		}

		m = updateModel(t, m, keyMsg('j'))
		if m.cursor != 2 {
			t.Errorf("after 'j' x2: cursor = %d, want 2", m.cursor)
		}
	})

	t.Run("k moves cursor up", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.cursor = 2

		m = updateModel(t, m, keyMsg('k'))
		if m.cursor != 1 {
			t.Errorf("after 'k': cursor = %d, want 1", m.cursor)
		}
	})

	t.Run("down arrow moves cursor down", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")

		m = updateModel(t, m, specialKeyMsg(tea.KeyDown))
		if m.cursor != 1 {
			t.Errorf("after KeyDown: cursor = %d, want 1", m.cursor)
		}
	})

	t.Run("up arrow moves cursor up", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.cursor = 3

		m = updateModel(t, m, specialKeyMsg(tea.KeyUp))
		if m.cursor != 2 {
			t.Errorf("after KeyUp: cursor = %d, want 2", m.cursor)
		}
	})

	t.Run("cursor does not go below zero", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.cursor = 0

		m = updateModel(t, m, keyMsg('k'))
		if m.cursor != 0 {
			t.Errorf("after 'k' at top: cursor = %d, want 0", m.cursor)
		}

		m = updateModel(t, m, specialKeyMsg(tea.KeyUp))
		if m.cursor != 0 {
			t.Errorf("after KeyUp at top: cursor = %d, want 0", m.cursor)
		}
	})

	t.Run("cursor does not exceed last item", func(t *testing.T) {
		t.Parallel()
		wts := testWorktrees()
		m := New(wts, "/repo")
		m.cursor = len(wts) - 1

		m = updateModel(t, m, keyMsg('j'))
		if m.cursor != len(wts)-1 {
			t.Errorf("after 'j' at bottom: cursor = %d, want %d", m.cursor, len(wts)-1)
		}

		m = updateModel(t, m, specialKeyMsg(tea.KeyDown))
		if m.cursor != len(wts)-1 {
			t.Errorf("after KeyDown at bottom: cursor = %d, want %d", m.cursor, len(wts)-1)
		}
	})
}

func TestToggleSelection(t *testing.T) {
	t.Parallel()

	t.Run("space toggles selection on non-current worktree", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.cursor = 1 // feature-a (not current)

		m = updateModel(t, m, specialKeyMsg(tea.KeySpace))
		if !m.items[1].checked {
			t.Error("after space: item[1].checked = false, want true")
		}

		// Toggle back off
		m = updateModel(t, m, specialKeyMsg(tea.KeySpace))
		if m.items[1].checked {
			t.Error("after second space: item[1].checked = true, want false")
		}
	})

	t.Run("x toggles selection on non-current worktree", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.cursor = 2 // feature-b (not current)

		m = updateModel(t, m, keyMsg('x'))
		if !m.items[2].checked {
			t.Error("after 'x': item[2].checked = false, want true")
		}
	})

	t.Run("space does not toggle current worktree", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.cursor = 0 // main (current)

		m = updateModel(t, m, specialKeyMsg(tea.KeySpace))
		if m.items[0].checked {
			t.Error("after space on current: item[0].checked = true, want false")
		}
	})

	t.Run("x does not toggle current worktree", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.cursor = 0 // main (current)

		m = updateModel(t, m, keyMsg('x'))
		if m.items[0].checked {
			t.Error("after 'x' on current: item[0].checked = true, want false")
		}
	})
}

func TestSelectAllMerged(t *testing.T) {
	t.Parallel()

	m := New(testWorktrees(), "/repo")

	m = updateModel(t, m, keyMsg('a'))

	// item[0]: main (current) -- should NOT be selected even if merged
	if m.items[0].checked {
		t.Error("item[0] (current) should not be selected by 'a'")
	}

	// item[1]: feature-a (merged, not current) -- should be selected
	if !m.items[1].checked {
		t.Error("item[1] (merged, not current) should be selected by 'a'")
	}

	// item[2]: feature-b (not merged) -- should NOT be selected
	if m.items[2].checked {
		t.Error("item[2] (not merged) should not be selected by 'a'")
	}

	// item[3]: feature-c (not merged) -- should NOT be selected
	if m.items[3].checked {
		t.Error("item[3] (not merged) should not be selected by 'a'")
	}
}

func TestDeselectAll(t *testing.T) {
	t.Parallel()

	m := New(testWorktrees(), "/repo")

	// Select a few items manually
	m.items[1].checked = true
	m.items[2].checked = true
	m.items[3].checked = true

	m = updateModel(t, m, keyMsg('n'))

	for i, it := range m.items {
		if it.checked {
			t.Errorf("after 'n': item[%d].checked = true, want false", i)
		}
	}
}

func TestEnterConfirmMode(t *testing.T) {
	t.Parallel()

	t.Run("d enters confirm mode with selection", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.items[1].checked = true

		m = updateModel(t, m, keyMsg('d'))
		if m.mode != modeConfirm {
			t.Errorf("after 'd' with selection: mode = %d, want modeConfirm (%d)", m.mode, modeConfirm)
		}
		if m.confirmCursor != 0 {
			t.Errorf("confirmCursor = %d, want 0 (No) by default", m.confirmCursor)
		}
	})

	t.Run("enter enters confirm mode with selection", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.items[2].checked = true

		m = updateModel(t, m, specialKeyMsg(tea.KeyEnter))
		if m.mode != modeConfirm {
			t.Errorf("after enter with selection: mode = %d, want modeConfirm (%d)", m.mode, modeConfirm)
		}
	})

	t.Run("d is no-op without selection", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")

		m = updateModel(t, m, keyMsg('d'))
		if m.mode != modeList {
			t.Errorf("after 'd' without selection: mode = %d, want modeList (%d)", m.mode, modeList)
		}
	})

	t.Run("enter is no-op without selection", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")

		m = updateModel(t, m, specialKeyMsg(tea.KeyEnter))
		if m.mode != modeList {
			t.Errorf("after enter without selection: mode = %d, want modeList (%d)", m.mode, modeList)
		}
	})
}

func TestConfirmNavigation(t *testing.T) {
	t.Parallel()

	setup := func(t *testing.T) model {
		t.Helper()
		m := New(testWorktrees(), "/repo")
		m.items[1].checked = true
		m.mode = modeConfirm
		m.confirmCursor = 0
		return m
	}

	t.Run("right toggles confirmCursor", func(t *testing.T) {
		t.Parallel()
		m := setup(t)

		m = updateModel(t, m, specialKeyMsg(tea.KeyRight))
		if m.confirmCursor != 1 {
			t.Errorf("after right: confirmCursor = %d, want 1", m.confirmCursor)
		}

		m = updateModel(t, m, specialKeyMsg(tea.KeyRight))
		if m.confirmCursor != 0 {
			t.Errorf("after right x2: confirmCursor = %d, want 0", m.confirmCursor)
		}
	})

	t.Run("left toggles confirmCursor", func(t *testing.T) {
		t.Parallel()
		m := setup(t)

		m = updateModel(t, m, specialKeyMsg(tea.KeyLeft))
		if m.confirmCursor != 1 {
			t.Errorf("after left: confirmCursor = %d, want 1", m.confirmCursor)
		}
	})

	t.Run("tab toggles confirmCursor", func(t *testing.T) {
		t.Parallel()
		m := setup(t)

		m = updateModel(t, m, specialKeyMsg(tea.KeyTab))
		if m.confirmCursor != 1 {
			t.Errorf("after tab: confirmCursor = %d, want 1", m.confirmCursor)
		}

		m = updateModel(t, m, specialKeyMsg(tea.KeyTab))
		if m.confirmCursor != 0 {
			t.Errorf("after tab x2: confirmCursor = %d, want 0", m.confirmCursor)
		}
	})

	t.Run("h toggles confirmCursor", func(t *testing.T) {
		t.Parallel()
		m := setup(t)

		m = updateModel(t, m, keyMsg('h'))
		if m.confirmCursor != 1 {
			t.Errorf("after 'h': confirmCursor = %d, want 1", m.confirmCursor)
		}
	})

	t.Run("l toggles confirmCursor", func(t *testing.T) {
		t.Parallel()
		m := setup(t)

		m = updateModel(t, m, keyMsg('l'))
		if m.confirmCursor != 1 {
			t.Errorf("after 'l': confirmCursor = %d, want 1", m.confirmCursor)
		}
	})
}

func TestConfirmYes(t *testing.T) {
	t.Parallel()

	// NOTE: We cannot fully test executeRemoval because it calls git commands.
	// We verify that pressing 'y' sets confirmCursor to 1 and transitions to
	// modeDone (executeRemoval will fail without a real repo, but the model
	// state should still transition). Since executeRemoval catches errors per
	// worktree, the mode transitions to modeDone regardless.

	m := New(testWorktrees(), "/repo")
	m.items[1].checked = true
	m.mode = modeConfirm
	m.confirmCursor = 0

	m = updateModel(t, m, keyMsg('y'))

	if m.confirmCursor != 1 {
		t.Errorf("after 'y': confirmCursor = %d, want 1", m.confirmCursor)
	}
	if m.mode != modeDone {
		t.Errorf("after 'y': mode = %d, want modeDone (%d)", m.mode, modeDone)
	}
}

func TestConfirmNo(t *testing.T) {
	t.Parallel()

	m := New(testWorktrees(), "/repo")
	m.items[1].checked = true
	m.mode = modeConfirm

	m = updateModel(t, m, keyMsg('n'))

	if m.mode != modeList {
		t.Errorf("after 'n' in confirm: mode = %d, want modeList (%d)", m.mode, modeList)
	}
}

func TestConfirmEsc(t *testing.T) {
	t.Parallel()

	keys := []struct {
		name string
		msg  tea.Msg
	}{
		{"esc", specialKeyMsg(tea.KeyEsc)},
		{"q", keyMsg('q')},
		{"ctrl+c", specialKeyMsg(tea.KeyCtrlC)},
	}

	for _, k := range keys {
		t.Run(k.name+" returns to list from confirm", func(t *testing.T) {
			t.Parallel()
			m := New(testWorktrees(), "/repo")
			m.items[1].checked = true
			m.mode = modeConfirm

			m = updateModel(t, m, k.msg)
			if m.mode != modeList {
				t.Errorf("after %s in confirm: mode = %d, want modeList (%d)", k.name, m.mode, modeList)
			}
		})
	}
}

func TestDoneMode(t *testing.T) {
	t.Parallel()

	keys := []struct {
		name string
		msg  tea.Msg
	}{
		{"q", keyMsg('q')},
		{"enter", specialKeyMsg(tea.KeyEnter)},
		{"space", specialKeyMsg(tea.KeySpace)},
		{"esc", specialKeyMsg(tea.KeyEsc)},
		{"any letter", keyMsg('z')},
	}

	for _, k := range keys {
		t.Run(k.name+" quits from done mode", func(t *testing.T) {
			t.Parallel()
			m := New(testWorktrees(), "/repo")
			m.mode = modeDone
			m.removed = []string{"feature-a"}

			result, cmd := m.Update(k.msg)
			updated, ok := result.(model)
			if !ok {
				t.Fatalf("Update did not return a model, got %T", result)
			}
			// Model should remain in modeDone (state unchanged)
			if updated.mode != modeDone {
				t.Errorf("after %s in done: mode = %d, want modeDone (%d)", k.name, updated.mode, modeDone)
			}
			// cmd should be tea.Quit
			if cmd == nil {
				t.Errorf("after %s in done: cmd = nil, want tea.Quit", k.name)
			}
		})
	}
}

func TestViewModes(t *testing.T) {
	t.Parallel()

	t.Run("viewList renders without panic", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.width = 80
		m.height = 24

		output := m.View()
		if output == "" {
			t.Error("viewList returned empty string")
		}
		if !strings.Contains(output, "Git Worktrees") {
			t.Error("viewList output missing title 'Git Worktrees'")
		}
		if !strings.Contains(output, "feature-a") {
			t.Error("viewList output missing branch name 'feature-a'")
		}
		if !strings.Contains(output, "main") {
			t.Error("viewList output missing branch name 'main'")
		}
	})

	t.Run("viewConfirm renders without panic", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.items[1].checked = true
		m.mode = modeConfirm
		m.width = 80
		m.height = 24

		output := m.View()
		if output == "" {
			t.Error("viewConfirm returned empty string")
		}
		if !strings.Contains(output, "Confirm Removal") {
			t.Error("viewConfirm output missing title 'Confirm Removal'")
		}
		if !strings.Contains(output, "feature-a") {
			t.Error("viewConfirm output missing selected branch 'feature-a'")
		}
		if !strings.Contains(output, "No") {
			t.Error("viewConfirm output missing 'No' option")
		}
		if !strings.Contains(output, "Yes") {
			t.Error("viewConfirm output missing 'Yes' option")
		}
	})

	t.Run("viewDone renders without panic", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.mode = modeDone
		m.removed = []string{"feature-a", "feature-b"}
		m.errors = []string{"feature-c: permission denied"}
		m.width = 80
		m.height = 24

		output := m.View()
		if output == "" {
			t.Error("viewDone returned empty string")
		}
		if !strings.Contains(output, "Cleanup Complete") {
			t.Error("viewDone output missing title 'Cleanup Complete'")
		}
		if !strings.Contains(output, "feature-a") {
			t.Error("viewDone output missing removed branch 'feature-a'")
		}
		if !strings.Contains(output, "feature-c: permission denied") {
			t.Error("viewDone output missing error message")
		}
		if !strings.Contains(output, "2 worktree(s)") {
			t.Error("viewDone output missing removal count")
		}
	})

	t.Run("viewList shows selected count when items checked", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.items[1].checked = true
		m.items[2].checked = true

		output := m.View()
		if !strings.Contains(output, "2 selected") {
			t.Error("viewList output missing '2 selected' count")
		}
	})
}

func TestWindowSizeMsg(t *testing.T) {
	t.Parallel()

	m := New(testWorktrees(), "/repo")

	result, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated, ok := result.(model)
	if !ok {
		t.Fatalf("Update did not return a model, got %T", result)
	}

	if updated.width != 120 {
		t.Errorf("width = %d, want 120", updated.width)
	}
	if updated.height != 40 {
		t.Errorf("height = %d, want 40", updated.height)
	}
}

func TestSelectedCount(t *testing.T) {
	t.Parallel()

	t.Run("zero when nothing selected", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")

		if got := m.selectedCount(); got != 0 {
			t.Errorf("selectedCount() = %d, want 0", got)
		}
	})

	t.Run("counts checked items", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.items[1].checked = true
		m.items[3].checked = true

		if got := m.selectedCount(); got != 2 {
			t.Errorf("selectedCount() = %d, want 2", got)
		}
	})

	t.Run("count updates after toggle", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		m.cursor = 1

		m = updateModel(t, m, specialKeyMsg(tea.KeySpace))
		if got := m.selectedCount(); got != 1 {
			t.Errorf("after select: selectedCount() = %d, want 1", got)
		}

		m = updateModel(t, m, specialKeyMsg(tea.KeySpace))
		if got := m.selectedCount(); got != 0 {
			t.Errorf("after deselect: selectedCount() = %d, want 0", got)
		}
	})
}

func TestBuildTags(t *testing.T) {
	t.Parallel()

	t.Run("current worktree shows current tag", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		tags := m.buildTags(m.items[0].worktree)

		if !strings.Contains(tags, "current") {
			t.Errorf("buildTags for current worktree missing 'current', got %q", tags)
		}
	})

	t.Run("merged worktree shows merged tag", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		tags := m.buildTags(m.items[1].worktree)

		if !strings.Contains(tags, "merged") {
			t.Errorf("buildTags for merged worktree missing 'merged', got %q", tags)
		}
	})

	t.Run("dirty worktree shows status text", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		tags := m.buildTags(m.items[3].worktree)

		if !strings.Contains(tags, "modified") {
			t.Errorf("buildTags for dirty worktree missing 'modified', got %q", tags)
		}
		if !strings.Contains(tags, "untracked") {
			t.Errorf("buildTags for dirty worktree missing 'untracked', got %q", tags)
		}
	})

	t.Run("clean non-special worktree has empty tags", func(t *testing.T) {
		t.Parallel()
		m := New(testWorktrees(), "/repo")
		tags := m.buildTags(m.items[2].worktree)

		if tags != "" {
			t.Errorf("buildTags for clean worktree = %q, want empty string", tags)
		}
	})

	t.Run("stale worktree shows stale tag", func(t *testing.T) {
		t.Parallel()
		staleWt := git.Worktree{
			Path:       "/repo-stale",
			Branch:     "refs/heads/stale-branch",
			LastCommit: time.Now().Add(-60 * 24 * time.Hour), // 60 days ago
		}
		m := New([]git.Worktree{staleWt}, "/repo")
		tags := m.buildTags(m.items[0].worktree)

		if !strings.Contains(tags, "stale") {
			t.Errorf("buildTags for stale worktree missing 'stale', got %q", tags)
		}
	})

	t.Run("worktree with sync info shows sync text", func(t *testing.T) {
		t.Parallel()
		syncWt := git.Worktree{
			Path:   "/repo-sync",
			Branch: "refs/heads/sync-branch",
			Ahead:  2,
			Behind: 1,
		}
		m := New([]git.Worktree{syncWt}, "/repo")
		tags := m.buildTags(m.items[0].worktree)

		if !strings.Contains(tags, "↑2") {
			t.Errorf("buildTags for ahead worktree missing ahead indicator, got %q", tags)
		}
		if !strings.Contains(tags, "↓1") {
			t.Errorf("buildTags for behind worktree missing behind indicator, got %q", tags)
		}
	})

	t.Run("detached worktree renders branch as short head", func(t *testing.T) {
		t.Parallel()
		detachedWt := git.Worktree{
			Path:       "/repo-detached",
			Head:       "abcdef1234567890",
			IsDetached: true,
		}
		m := New([]git.Worktree{detachedWt}, "/repo")
		// We test through View to verify detached rendering
		m.cursor = 0
		output := m.View()

		if !strings.Contains(output, "abcdef12") {
			t.Errorf("viewList for detached worktree missing short head, got output:\n%s", output)
		}
		if !strings.Contains(output, "(detached)") {
			t.Errorf("viewList for detached worktree missing '(detached)', got output:\n%s", output)
		}
	})

	t.Run("bare worktree renders as bare", func(t *testing.T) {
		t.Parallel()
		bareWt := git.Worktree{
			Path:   "/repo-bare",
			IsBare: true,
		}
		m := New([]git.Worktree{bareWt}, "/repo")
		m.cursor = 0
		output := m.View()

		if !strings.Contains(output, "(bare)") {
			t.Errorf("viewList for bare worktree missing '(bare)', got output:\n%s", output)
		}
	})
}

func TestQuitFromList(t *testing.T) {
	t.Parallel()

	quitKeys := []struct {
		name string
		msg  tea.Msg
	}{
		{"q", keyMsg('q')},
		{"esc", specialKeyMsg(tea.KeyEsc)},
		{"ctrl+c", specialKeyMsg(tea.KeyCtrlC)},
	}

	for _, k := range quitKeys {
		t.Run(k.name+" quits from list mode", func(t *testing.T) {
			t.Parallel()
			m := New(testWorktrees(), "/repo")

			_, cmd := m.Update(k.msg)
			if cmd == nil {
				t.Errorf("after %s in list: cmd = nil, want tea.Quit", k.name)
			}
		})
	}
}

func TestInitReturnsNil(t *testing.T) {
	t.Parallel()

	m := New(testWorktrees(), "/repo")
	cmd := m.Init()
	if cmd != nil {
		t.Errorf("Init() returned non-nil cmd: %v", cmd)
	}
}

func TestConfirmEnterWithCursorOnNo(t *testing.T) {
	t.Parallel()

	m := New(testWorktrees(), "/repo")
	m.items[1].checked = true
	m.mode = modeConfirm
	m.confirmCursor = 0 // cursor on No

	m = updateModel(t, m, specialKeyMsg(tea.KeyEnter))

	if m.mode != modeList {
		t.Errorf("enter with confirmCursor=0: mode = %d, want modeList (%d)", m.mode, modeList)
	}
}

func TestConfirmEnterWithCursorOnYes(t *testing.T) {
	t.Parallel()

	// NOTE: executeRemoval will fail without a real git repo, but the model
	// transitions to modeDone regardless because errors are collected per item.
	m := New(testWorktrees(), "/repo")
	m.items[1].checked = true
	m.mode = modeConfirm
	m.confirmCursor = 1 // cursor on Yes

	m = updateModel(t, m, specialKeyMsg(tea.KeyEnter))

	if m.mode != modeDone {
		t.Errorf("enter with confirmCursor=1: mode = %d, want modeDone (%d)", m.mode, modeDone)
	}
}

func TestEmptyWorktreeList(t *testing.T) {
	t.Parallel()

	m := New([]git.Worktree{}, "/repo")

	if len(m.items) != 0 {
		t.Errorf("items = %d, want 0", len(m.items))
	}

	// View should not panic with empty items
	output := m.View()
	if output == "" {
		t.Error("View returned empty string for empty model")
	}
}

// ===========================================================================
// BuildTags (extracted helper) tests
// ===========================================================================

func TestBuildTagsFunction(t *testing.T) {
	t.Parallel()

	t.Run("current worktree", func(t *testing.T) {
		t.Parallel()
		wt := git.Worktree{Branch: "refs/heads/main", IsCurrent: true}
		tags := BuildTags(wt)
		if !strings.Contains(tags, "current") {
			t.Errorf("BuildTags missing 'current', got %q", tags)
		}
	})

	t.Run("merged worktree", func(t *testing.T) {
		t.Parallel()
		wt := git.Worktree{Branch: "refs/heads/feat", IsMerged: true}
		tags := BuildTags(wt)
		if !strings.Contains(tags, "merged") {
			t.Errorf("BuildTags missing 'merged', got %q", tags)
		}
	})

	t.Run("dirty worktree", func(t *testing.T) {
		t.Parallel()
		wt := git.Worktree{Branch: "refs/heads/feat", Modified: 2, Untracked: 1}
		tags := BuildTags(wt)
		if !strings.Contains(tags, "modified") {
			t.Errorf("BuildTags missing 'modified', got %q", tags)
		}
	})

	t.Run("clean worktree returns empty", func(t *testing.T) {
		t.Parallel()
		wt := git.Worktree{Branch: "refs/heads/feat"}
		tags := BuildTags(wt)
		if tags != "" {
			t.Errorf("BuildTags for clean worktree = %q, want empty", tags)
		}
	})

	t.Run("stale worktree", func(t *testing.T) {
		t.Parallel()
		wt := git.Worktree{
			Branch:     "refs/heads/old",
			LastCommit: time.Now().Add(-60 * 24 * time.Hour),
		}
		tags := BuildTags(wt)
		if !strings.Contains(tags, "stale") {
			t.Errorf("BuildTags missing 'stale', got %q", tags)
		}
	})
}

// ===========================================================================
// Selector TUI tests
// ===========================================================================

func updateSelectorModel(t *testing.T, m selectorModel, msg tea.Msg) selectorModel {
	t.Helper()
	result, _ := m.Update(msg)
	updated, ok := result.(selectorModel)
	if !ok {
		t.Fatalf("Update did not return a selectorModel, got %T", result)
	}
	return updated
}

func TestSelectorNew(t *testing.T) {
	t.Parallel()
	wts := testWorktrees()
	m := NewSelector(wts)

	if len(m.items) != len(wts) {
		t.Fatalf("NewSelector created %d items, want %d", len(m.items), len(wts))
	}
	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.selected != nil {
		t.Error("selected should be nil on initialization")
	}
	if m.done {
		t.Error("done should be false on initialization")
	}
}

func TestSelectorNavigation(t *testing.T) {
	t.Parallel()

	t.Run("j moves cursor down", func(t *testing.T) {
		t.Parallel()
		m := NewSelector(testWorktrees())
		m = updateSelectorModel(t, m, keyMsg('j'))
		if m.cursor != 1 {
			t.Errorf("after 'j': cursor = %d, want 1", m.cursor)
		}
	})

	t.Run("k moves cursor up", func(t *testing.T) {
		t.Parallel()
		m := NewSelector(testWorktrees())
		m.cursor = 2
		m = updateSelectorModel(t, m, keyMsg('k'))
		if m.cursor != 1 {
			t.Errorf("after 'k': cursor = %d, want 1", m.cursor)
		}
	})

	t.Run("down arrow moves cursor down", func(t *testing.T) {
		t.Parallel()
		m := NewSelector(testWorktrees())
		m = updateSelectorModel(t, m, specialKeyMsg(tea.KeyDown))
		if m.cursor != 1 {
			t.Errorf("after KeyDown: cursor = %d, want 1", m.cursor)
		}
	})

	t.Run("up arrow moves cursor up", func(t *testing.T) {
		t.Parallel()
		m := NewSelector(testWorktrees())
		m.cursor = 3
		m = updateSelectorModel(t, m, specialKeyMsg(tea.KeyUp))
		if m.cursor != 2 {
			t.Errorf("after KeyUp: cursor = %d, want 2", m.cursor)
		}
	})

	t.Run("cursor does not go below zero", func(t *testing.T) {
		t.Parallel()
		m := NewSelector(testWorktrees())
		m = updateSelectorModel(t, m, keyMsg('k'))
		if m.cursor != 0 {
			t.Errorf("cursor went below 0: %d", m.cursor)
		}
	})

	t.Run("cursor does not exceed last item", func(t *testing.T) {
		t.Parallel()
		wts := testWorktrees()
		m := NewSelector(wts)
		m.cursor = len(wts) - 1
		m = updateSelectorModel(t, m, keyMsg('j'))
		if m.cursor != len(wts)-1 {
			t.Errorf("cursor exceeded last: %d", m.cursor)
		}
	})
}

func TestSelectorConfirm(t *testing.T) {
	t.Parallel()

	wts := testWorktrees()
	m := NewSelector(wts)
	m.cursor = 1 // feature-a

	result, cmd := m.Update(specialKeyMsg(tea.KeyEnter))
	updated, ok := result.(selectorModel)
	if !ok {
		t.Fatalf("Update did not return a selectorModel, got %T", result)
	}

	if updated.selected == nil {
		t.Fatal("expected selected to be set after enter")
	}
	if updated.selected.BranchShort() != "feature-a" {
		t.Errorf("expected selected = feature-a, got %s", updated.selected.BranchShort())
	}
	if !updated.done {
		t.Error("expected done = true after enter")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command after enter")
	}
}

func TestSelectorCancel(t *testing.T) {
	t.Parallel()

	cancelKeys := []struct {
		name string
		msg  tea.Msg
	}{
		{"q", keyMsg('q')},
		{"esc", specialKeyMsg(tea.KeyEsc)},
		{"ctrl+c", specialKeyMsg(tea.KeyCtrlC)},
	}

	for _, k := range cancelKeys {
		t.Run(k.name+" cancels selector", func(t *testing.T) {
			t.Parallel()
			m := NewSelector(testWorktrees())

			result, cmd := m.Update(k.msg)
			updated, ok := result.(selectorModel)
			if !ok {
				t.Fatalf("Update did not return a selectorModel, got %T", result)
			}

			if updated.selected != nil {
				t.Errorf("after %s: selected should be nil", k.name)
			}
			if !updated.done {
				t.Errorf("after %s: done should be true", k.name)
			}
			if cmd == nil {
				t.Errorf("after %s: expected tea.Quit command", k.name)
			}
		})
	}
}

func TestSelectorView(t *testing.T) {
	t.Parallel()

	t.Run("renders title and branches", func(t *testing.T) {
		t.Parallel()
		m := NewSelector(testWorktrees())
		output := m.View()

		if !strings.Contains(output, "Switch Worktree") {
			t.Error("view missing title 'Switch Worktree'")
		}
		if !strings.Contains(output, "feature-a") {
			t.Error("view missing branch 'feature-a'")
		}
		if !strings.Contains(output, "main") {
			t.Error("view missing branch 'main'")
		}
	})

	t.Run("done state returns empty", func(t *testing.T) {
		t.Parallel()
		m := NewSelector(testWorktrees())
		m.done = true
		output := m.View()
		if output != "" {
			t.Errorf("done view should be empty, got %q", output)
		}
	})

	t.Run("shows help text", func(t *testing.T) {
		t.Parallel()
		m := NewSelector(testWorktrees())
		output := m.View()
		if !strings.Contains(output, "enter select") {
			t.Error("view missing help text")
		}
	})
}

func TestSelectorWindowSize(t *testing.T) {
	t.Parallel()

	m := NewSelector(testWorktrees())
	result, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	updated := result.(selectorModel)

	if updated.width != 100 {
		t.Errorf("width = %d, want 100", updated.width)
	}
	if updated.height != 30 {
		t.Errorf("height = %d, want 30", updated.height)
	}
}

func TestSelectorInit(t *testing.T) {
	t.Parallel()

	m := NewSelector(testWorktrees())
	cmd := m.Init()
	if cmd != nil {
		t.Errorf("Init() returned non-nil cmd: %v", cmd)
	}
}

func TestSelectorEmptyList(t *testing.T) {
	t.Parallel()

	m := NewSelector([]git.Worktree{})
	// Enter on empty list should not set selected
	result, _ := m.Update(specialKeyMsg(tea.KeyEnter))
	updated := result.(selectorModel)

	if updated.selected != nil {
		t.Error("enter on empty list should not set selected")
	}
}
