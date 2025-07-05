package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type Team struct {
	id   int
	name string
}

type League struct {
	id   int
	name string
	code string
}

func (l League) isNil() bool {
	return l == League{}
}

// Implement list.Item interface for League
func (l League) FilterValue() string { return l.code }
func (l League) Title() string       { return l.code }
func (l League) Description() string { return l.name }

type State struct {
	err     error
	leagues list.Model
	league  League
}

func newState() *State {
	state := &State{}
	listDelegate := list.NewDefaultDelegate()
	blue1 := lipgloss.Color("38")
	blue2 := lipgloss.Color("32")
	listDelegate.Styles.SelectedTitle = listDelegate.Styles.SelectedTitle.Foreground(blue1).BorderLeftForeground(blue1)
	listDelegate.Styles.SelectedDesc = listDelegate.Styles.SelectedDesc.Foreground(blue2).BorderLeftForeground(blue1)
	state.leagues = list.New([]list.Item{}, listDelegate, 0, 0)
	state.leagues.Title = "Leagues"
	// state.leagues.SetShowHelp(false)
	return state
}

// Init implements tea.Model.
func (s *State) Init() tea.Cmd {
	return openDB
}

// Update implements tea.Model.
func (s *State) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		{
			switch keypress := msg.String(); keypress {
			case "ctrl+c":
				return s, tea.Quit
			case "enter":
				l, ok := s.leagues.SelectedItem().(League)
				if ok {
					s.league = l
				}
				return s, nil
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		s.leagues.SetSize(msg.Width-h, msg.Height-v)
	case ErrMsg:
		s.err = msg
		return s, tea.Quit
	case DBConnected:
		return s, getLeagues
	case LeaguesLoaded:
		items := make([]list.Item, len(msg))
		for i, league := range msg {
			items[i] = league
		}
		return s, s.leagues.SetItems(items)
	}

	var cmd tea.Cmd
	s.leagues, cmd = s.leagues.Update(msg)
	return s, cmd
}

// View implements tea.Model.
func (s *State) View() string {
	if s.err != nil {
		return fmt.Sprintf("\nError: %v\n\n", s.err)
	}

	if s.league.isNil() {
		return docStyle.Render(s.leagues.View())
	}

	return docStyle.Render(fmt.Sprintf("you chose %s", s.league.name))
}

func main() {
	state := newState()
	p := tea.NewProgram(state, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
