package tui

import (
	"fmt"
	"strings"

	"github.com/yasomaru/git-wt/internal/git"
)

// BuildTags returns a formatted tag string for a worktree, showing status
// indicators like [current], [merged], [3 modified, 1 untracked], sync info,
// and staleness warnings. Used by both the deletion TUI and the selector TUI.
func BuildTags(wt git.Worktree) string {
	var tags []string

	if wt.IsCurrent {
		tags = append(tags, currentStyle.Render("current"))
	}

	if !wt.IsClean() {
		tags = append(tags, dirtyStyle.Render(wt.StatusText()))
	}

	if wt.IsMerged {
		tags = append(tags, mergedStyle.Render("merged"))
	}

	sync := wt.SyncText()
	if sync != "-" {
		tags = append(tags, dimStyle.Render(sync))
	}

	days := wt.InactiveDays()
	if days > 30 {
		tags = append(tags, staleStyle.Render(fmt.Sprintf("%dd stale", days)))
	}

	if len(tags) == 0 {
		return ""
	}
	return dimStyle.Render("[") + strings.Join(tags, dimStyle.Render(", ")) + dimStyle.Render("]")
}
