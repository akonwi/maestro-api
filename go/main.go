package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type Team struct {
	id   int8
	name string
}

type League struct {
	id   int8
	name string
}

type State struct {
	db      *sql.DB
	err     error
	leagues list.Model
}

func newState() *State {
	state := &State{}
	state.leagues = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	state.leagues.Title = "Leagues"
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
		if msg.String() == "ctrl+c" {
			return s, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		s.leagues.SetSize(msg.Width-h, msg.Height-v)
	case ErrMsg:
		s.err = msg
		return s, tea.Quit
	case DBConnected:
		s.db = msg
	}

	var cmd tea.Cmd
	s.leagues, cmd = s.leagues.Update(msg)
	return s, cmd
}

// View implements tea.Model.
func (s *State) View() string {
	if s.err != nil {
		return docStyle.Render(s.err.Error())
	}

	return docStyle.Render(s.leagues.View())
}

func main() {
	state := newState()
	p := tea.NewProgram(state, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
