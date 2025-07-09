package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
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

type Match struct {
	id           int
	date         string
	timestamp    int
	leagueId     int
	status       string
	homeTeamId   int
	awayTeamId   int
	homeTeamName string
	awayTeamName string
	homeGoals    int
	awayGoals    int
	winnerId     *int
}

// Implement list.Item interface for Match
func (m Match) FilterValue() string { return m.homeTeamName + " vs " + m.awayTeamName }
func (m Match) Title() string       { return m.homeTeamName + " vs " + m.awayTeamName }
func (m Match) Description() string { return m.score() }

func (m Match) score() string {
	if m.status == "NS" {
		return "TBD"
	}
	return fmt.Sprintf("%d - %d (%s)", m.homeGoals, m.awayGoals, m.status)
}

// Implement list.Item interface for League
func (l League) FilterValue() string { return l.name }
func (l League) Title() string       { return l.name }
func (l League) Description() string { return l.code }

type ViewState int

const (
	ViewLeagues ViewState = iota
	ViewMatches
)

type State struct {
	db                *sql.DB
	err               error
	leagues           list.Model
	matches           list.Model
	selectedLeague    League
	currentView       ViewState
	loading           bool
	spinner           spinner.Model
	showPlayedMatches bool
}

func newState() *State {
	state := &State{}

	// Initialize spinner
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	state.spinner = sp

	listDelegate := list.NewDefaultDelegate()
	blue1 := lipgloss.Color("38")
	blue2 := lipgloss.Color("32")
	listDelegate.Styles.SelectedTitle = listDelegate.Styles.SelectedTitle.Foreground(blue1).BorderLeftForeground(blue1)
	listDelegate.Styles.SelectedDesc = listDelegate.Styles.SelectedDesc.Foreground(blue2).BorderLeftForeground(blue1)

	state.leagues = list.New([]list.Item{}, listDelegate, 0, 0)
	state.leagues.Title = "Leagues"

	matchDelegate := list.NewDefaultDelegate()
	matchDelegate.Styles.SelectedTitle = matchDelegate.Styles.SelectedTitle.Foreground(blue1).BorderLeftForeground(blue1)
	matchDelegate.Styles.SelectedDesc = matchDelegate.Styles.SelectedDesc.Foreground(blue2).BorderLeftForeground(blue1)
	state.matches = list.New([]list.Item{}, matchDelegate, 0, 0)
	state.matches.Title = "Matches"

	// Add help key binding for matches list
	helpKey := key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "toggle played/unplayed"),
	)
	state.matches.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{helpKey}
	}

	state.currentView = ViewLeagues
	state.showPlayedMatches = false
	return state
}

func (s *State) updateMatchesTitle() {
	if s.showPlayedMatches {
		s.matches.Title = "Played Matches"
	} else {
		s.matches.Title = "Upcoming Matches"
	}
}

// Init implements tea.Model.
func (s *State) Init() tea.Cmd {
	return openDB
}

// Update implements tea.Model.
func (s *State) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ErrMsg:
		s.err = msg
		return s, nil
	case tea.KeyMsg:
		{
			switch keypress := msg.String(); keypress {
			case "ctrl+c":
				return s, tea.Quit
			case "esc":
				if s.currentView == ViewMatches {
					s.currentView = ViewLeagues
					return s, nil
				}
			case "s":
				if s.currentView == ViewMatches {
					s.showPlayedMatches = !s.showPlayedMatches
					s.loading = true
					s.updateMatchesTitle()
					return s, getMatches(s.selectedLeague.id, s.showPlayedMatches)
				}
			case "enter":
				if s.currentView == ViewLeagues {
					l, ok := s.leagues.SelectedItem().(League)
					if ok {
						s.selectedLeague = l
						s.currentView = ViewMatches
						s.loading = true
						s.showPlayedMatches = false
						s.updateMatchesTitle()
						return s, getMatches(l.id, s.showPlayedMatches)
					}
				}
				// if s.currentView == ViewMatches {
				// 	m, ok := s.matches.SelectedItem().(Match)
				// 	if ok {
				// 		s.selectedMatch = m
				// 		return s, getHeadToHeadStats(m.id)
				// 	}
				// }

				return s, nil
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		s.leagues.SetSize(msg.Width-h, msg.Height-v)
		s.matches.SetSize(msg.Width-h, msg.Height-v)
	case DBConnected:
		s.db = msg
		return s, getLeagues
	case LeaguesLoaded:
		items := make([]list.Item, len(msg))
		for i, league := range msg {
			items[i] = league
		}
		s.leagues.SetItems(items)
		return s, nil
	case MatchesLoaded:
		items := make([]list.Item, len(msg))
		for i, match := range msg {
			items[i] = match
		}
		s.matches.SetItems(items)
		s.loading = false
		return s, nil
	}

	var listCmd tea.Cmd
	var spinnerCmd tea.Cmd
	switch s.currentView {
	case ViewLeagues:
		s.leagues, listCmd = s.leagues.Update(msg)
	case ViewMatches:
		s.matches, listCmd = s.matches.Update(msg)
	}
	s.spinner, spinnerCmd = s.spinner.Update(msg)

	return s, tea.Batch(listCmd, spinnerCmd)
}

// View implements tea.Model.
func (s *State) View() string {
	if s.err != nil {
		return fmt.Sprintf("\nError: %v\n\n", s.err)
	}

	switch s.currentView {
	case ViewLeagues:
		return docStyle.Render(s.leagues.View())
	case ViewMatches:
		if s.loading {
			return docStyle.Render(s.spinner.View() + " Loading Matches")
		}
		return docStyle.Render(s.matches.View())
	default:
		return docStyle.Render(s.leagues.View())
	}
}

func main() {
	state := newState()
	p := tea.NewProgram(state, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
