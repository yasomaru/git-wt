package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/yasomaru/git-wt/internal/git"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	normalStyle   = lipgloss.NewStyle()
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	currentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	dirtyStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	mergedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	staleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	checkStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	confirmYesStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
	confirmNoStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("114"))
)

type mode int

const (
	modeList mode = iota
	modeConfirm
	modeDone
)

type item struct {
	worktree git.Worktree
	checked  bool
}

type model struct {
	items         []item
	cursor        int
	mode          mode
	confirmCursor int // 0=No, 1=Yes
	repoDir       string
	removed       []string
	errors        []string
	width         int
	height        int
}

func New(worktrees []git.Worktree, repoDir string) model {
	items := make([]item, len(worktrees))
	for i, wt := range worktrees {
		items[i] = item{worktree: wt}
	}
	return model{
		items:   items,
		repoDir: repoDir,
	}
}

func Run(worktrees []git.Worktree, repoDir string) error {
	m := New(worktrees, repoDir)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch m.mode {
		case modeList:
			return m.updateList(msg)
		case modeConfirm:
			return m.updateConfirm(msg)
		case modeDone:
			return m.updateDone(msg)
		}
	}
	return m, nil
}

func (m model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}

	case " ", "x":
		// Don't allow selecting current worktree
		if !m.items[m.cursor].worktree.IsCurrent {
			m.items[m.cursor].checked = !m.items[m.cursor].checked
		}

	case "a":
		// Select all non-current merged worktrees
		for i := range m.items {
			if !m.items[i].worktree.IsCurrent && m.items[i].worktree.IsMerged {
				m.items[i].checked = true
			}
		}

	case "n":
		// Deselect all
		for i := range m.items {
			m.items[i].checked = false
		}

	case "d", "enter":
		selected := m.selectedCount()
		if selected > 0 {
			m.mode = modeConfirm
			m.confirmCursor = 0
		}
	}

	return m, nil
}

func (m model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.mode = modeList
		return m, nil

	case "left", "h", "right", "l", "tab":
		m.confirmCursor = 1 - m.confirmCursor

	case "y":
		m.confirmCursor = 1
		return m.executeRemoval()

	case "n":
		m.mode = modeList
		return m, nil

	case "enter":
		if m.confirmCursor == 1 {
			return m.executeRemoval()
		}
		m.mode = modeList
		return m, nil
	}
	return m, nil
}

func (m model) updateDone(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

func (m model) executeRemoval() (tea.Model, tea.Cmd) {
	for i := range m.items {
		if !m.items[i].checked {
			continue
		}
		wt := m.items[i].worktree
		branch := wt.BranchShort()
		deleteBranch := wt.IsMerged
		if err := git.RemoveWorktree(m.repoDir, wt.Path, deleteBranch); err != nil {
			m.errors = append(m.errors, fmt.Sprintf("%s: %v", branch, err))
		} else {
			m.removed = append(m.removed, branch)
		}
	}
	_ = git.PruneWorktrees(m.repoDir)
	m.mode = modeDone
	return m, nil
}

func (m model) selectedCount() int {
	count := 0
	for _, it := range m.items {
		if it.checked {
			count++
		}
	}
	return count
}

func (m model) View() string {
	switch m.mode {
	case modeList:
		return m.viewList()
	case modeConfirm:
		return m.viewConfirm()
	case modeDone:
		return m.viewDone()
	}
	return ""
}

func (m model) viewList() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" Git Worktrees "))
	b.WriteString("\n\n")

	for i, it := range m.items {
		wt := it.worktree
		cursor := "  "
		if i == m.cursor {
			cursor = "▸ "
		}

		// Checkbox
		check := "○"
		if it.checked {
			check = checkStyle.Render("●")
		}
		if wt.IsCurrent {
			check = currentStyle.Render("◆")
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

		// Tags
		tags := m.buildTags(wt)

		line := fmt.Sprintf("%s%s %s %s", cursor, check, branchStr, tags)
		b.WriteString(line)
		b.WriteString("\n")

		// Show path on second line for selected item
		if i == m.cursor {
			pathStr := dimStyle.Render(fmt.Sprintf("     %s", wt.Path))
			b.WriteString(pathStr)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Footer
	selected := m.selectedCount()
	if selected > 0 {
		b.WriteString(fmt.Sprintf("  %d selected  ", selected))
	}
	b.WriteString(helpStyle.Render("↑↓/jk move  space select  a merged  d delete  q quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) buildTags(wt git.Worktree) string {
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

func (m model) viewConfirm() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" Confirm Removal "))
	b.WriteString("\n\n")

	b.WriteString("  Remove the following worktrees?\n\n")

	for _, it := range m.items {
		if !it.checked {
			continue
		}
		branch := it.worktree.BranchShort()
		b.WriteString(fmt.Sprintf("    %s %s\n", checkStyle.Render("●"), branch))
	}

	b.WriteString("\n")

	no := "  No  "
	yes := "  Yes  "
	if m.confirmCursor == 0 {
		no = confirmNoStyle.Render("▸ No ")
		yes = dimStyle.Render("  Yes ")
	} else {
		no = dimStyle.Render("  No ")
		yes = confirmYesStyle.Render("▸ Yes ")
	}
	b.WriteString("  " + no + "  " + yes + "\n")
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  ←→/hl switch  enter confirm  y yes  n no  esc back"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewDone() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(" Cleanup Complete "))
	b.WriteString("\n\n")

	for _, name := range m.removed {
		b.WriteString(fmt.Sprintf("  %s %s\n", mergedStyle.Render("✓"), name))
	}
	for _, e := range m.errors {
		b.WriteString(fmt.Sprintf("  %s %s\n", staleStyle.Render("✗"), e))
	}

	b.WriteString(fmt.Sprintf("\n  Removed %d worktree(s).\n", len(m.removed)))
	b.WriteString(helpStyle.Render("\n  Press any key to exit."))
	b.WriteString("\n")

	return b.String()
}
