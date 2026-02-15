package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/yasomaru/git-wt/internal/git"
)

// selectorModel is a single-select TUI for choosing a worktree.
// Unlike the main deletion TUI, it has no checkboxes — just a cursor.
type selectorModel struct {
	items    []git.Worktree
	cursor   int
	selected *git.Worktree // nil = cancelled
	done     bool
	width    int
	height   int
}

// NewSelector creates a new selector model from a list of worktrees.
func NewSelector(worktrees []git.Worktree) selectorModel {
	return selectorModel{
		items: worktrees,
	}
}

// RunSelector launches the TUI selector on stderr and returns the chosen
// worktree, or nil if the user cancelled. stderr is used for rendering so
// that stdout remains clean for outputting the selected path.
func RunSelector(worktrees []git.Worktree) (*git.Worktree, error) {
	m := NewSelector(worktrees)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	final := result.(selectorModel)
	return final.selected, nil
}

func (m selectorModel) Init() tea.Cmd {
	return nil
}

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.done = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case "enter":
			if len(m.items) > 0 {
				wt := m.items[m.cursor]
				m.selected = &wt
			}
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectorModel) View() string {
	if m.done {
		return ""
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render(" Switch Worktree "))
	b.WriteString("\n\n")

	for i, wt := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}

		// Branch name
		branch := wt.BranchShort()
		if wt.IsDetached {
			branch = wt.ShortHead() + " (detached)"
		}
		if wt.IsBare {
			branch = "(bare)"
		}

		var branchStr string
		if i == m.cursor {
			branchStr = selectedStyle.Render(branch)
		} else if wt.IsCurrent {
			branchStr = currentStyle.Render(branch)
		} else {
			branchStr = normalStyle.Render(branch)
		}

		tags := BuildTags(wt)

		line := fmt.Sprintf("%s%s %s", cursor, branchStr, tags)
		b.WriteString(line)
		b.WriteString("\n")

		// Show path on second line for cursor item
		if i == m.cursor {
			pathStr := dimStyle.Render(fmt.Sprintf("    %s", wt.Path))
			b.WriteString(pathStr)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑↓/jk move  enter select  q/esc cancel"))
	b.WriteString("\n")

	return b.String()
}
