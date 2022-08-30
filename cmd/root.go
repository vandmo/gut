package cmd

import (
	"context"
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	cp "github.com/otiai10/copy"
	"github.com/spf13/cobra"
	"github.com/vandmo/gut/internal"
	"os"
	"path"
)

var rootCmd = &cobra.Command{
	Use:     "gut <folder>",
	Short:   "Interactively copies selected content from a folder",
	Version: internal.Version(),
	Args:    cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return DoIt(args[0])
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.ExecuteContext(context.Background()))
}

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type errMsg struct{ error }

type dirReadMsg struct {
	entries []os.DirEntry
	parent  string
}
type entryCopiedMsg struct{}
type item struct {
	id, title string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return "" }

type model struct {
	source string
	list   list.Model
	err    error
	done   bool
	job    *job
}

type job struct {
	entry  os.DirEntry
	parent string
	prev   *job
}

func (j job) Name() string { return path.Join(j.parent, j.entry.Name()) }

func readDir(root string, name string) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(path.Join(root, name))
		if err != nil {
			return errMsg{err}
		}
		return dirReadMsg{parent: name, entries: entries}
	}
}

func doCopy(sourceRoot string, name string) tea.Cmd {
	return func() tea.Msg {
		cwd, getCwdErr := os.Getwd()
		if getCwdErr != nil {
			return errMsg{getCwdErr}
		}
		src := path.Join(sourceRoot, name)
		dst := path.Join(cwd, name)
		copyErr := cp.Copy(src, dst)
		if copyErr != nil {
			return errMsg{copyErr}
		}
		return entryCopiedMsg{}
	}
}

func (m model) Init() tea.Cmd {
	return readDir(m.source, "")
}

func (m model) updateForJob() (tea.Model, tea.Cmd) {
	if m.job == nil || m.done {
		m.list.Title = ""
		return m, m.list.SetItems(nil)
	}
	if m.job.entry.IsDir() {
		m.list.Title = fmt.Sprintf("Do you want to copy folder %s", m.job.Name())
		items := []list.Item{
			item{id: "n", title: "(N)o"},
			item{id: "c", title: "(C)ompletetly"},
			item{id: "a", title: "(A)sk"},
		}
		cmd := m.list.SetItems(items)
		return m, cmd
	} else {
		m.list.Title = fmt.Sprintf("Do you want to copy file %s", m.job.Name())
		items := []list.Item{
			item{id: "n", title: "(N)o"},
			item{id: "y", title: "(Y)es"},
		}
		cmd := m.list.SetItems(items)
		return m, cmd
	}
}

func processLetter(m model, letter string) (tea.Model, tea.Cmd) {
	switch letter {
	case "y":
		if m.job != nil && !m.job.entry.IsDir() {
			return m, doCopy(m.source, m.job.Name())
		}
	case "n":
		if m.job != nil {
			m.job = m.job.prev
			return m.updateForJob()
		}
	case "c":
		if m.job != nil && m.job.entry.IsDir() {
			return m, doCopy(m.source, m.job.Name())
		}
	case "a":
		if m.job != nil && m.job.entry.IsDir() {
			name := m.job.Name()
			m.job = m.job.prev
			return m, readDir(m.source, name)
		}
	}
	return m, nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return m, tea.Quit
		case "y", "n", "c", "a":
			return processLetter(m, keypress)
		case "enter":
			i, ok := m.list.SelectedItem().(item)
			if ok {
				return processLetter(m, i.id)
			}
			return nil, nil
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case errMsg:
		m.err = msg
		return m, nil
	case dirReadMsg:
		for _, entry := range msg.entries {
			m.job = &job{parent: msg.parent, entry: entry, prev: m.job}
		}
		return m.updateForJob()
	case entryCopiedMsg:
		m.job = m.job.prev
		return m.updateForJob()
	}

	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	return m, cmd
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Something went wrong, press Q to quit: %s\n", m.err)
	} else if m.done {
		return fmt.Sprintf("All done, press Q to quit!")
	} else {
		return docStyle.Render(m.list.View())
	}
}

func DoIt(source string) error {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	m := model{list: list.New(nil, delegate, 0, 0), source: source}
	m.list.SetShowPagination(false)
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(false)

	p := tea.NewProgram(m, tea.WithAltScreen())

	return p.Start()
}
