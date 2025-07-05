package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

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

type Match struct {
	id        int
	date      string
	homeTeam  Team
	awayTeam  Team
	homeGoals int8
	awayGoals int8
	status    string
}

// Implement list.Item interface for Match
func (m Match) FilterValue() string { return m.homeTeam.name + " vs " + m.awayTeam.name }
func (m Match) Title() string       { return m.homeTeam.name + " vs " + m.awayTeam.name }
func (m Match) Description() string { return m.score() }

func (m Match) score() string {
	if m.status == "NS" {
		return "TBD"
	}
	return fmt.Sprintf("%d - %d", m.homeGoals, m.awayGoals)
}

type FixtureTeam struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

type FixtureTeams struct {
	Home FixtureTeam `json:"home"`
	Away FixtureTeam `json:"away"`
}

type FixtureGoals struct {
	Home *int `json:"home"`
	Away *int `json:"away"`
}

type FixtureStatus struct {
	Short string `json:"short"`
	Long  string `json:"long"`
}

type Fixture struct {
	ID     int           `json:"id"`
	Date   time.Time     `json:"date"`
	Status FixtureStatus `json:"status"`
}

type FixtureEntry struct {
	Fixture Fixture      `json:"fixture"`
	Teams   FixtureTeams `json:"teams"`
	Goals   FixtureGoals `json:"goals"`
}

type ApiError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type FixturesResponse struct {
	Results  int            `json:"results"`
	Response []FixtureEntry `json:"response"`
	Errors   []ApiError     `json:"errors"`
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
	db             *sql.DB
	err            error
	leagues        list.Model
	matches        list.Model
	selectedLeague League
	currentView    ViewState
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

	matchDelegate := list.NewDefaultDelegate()
	matchDelegate.Styles.SelectedTitle = matchDelegate.Styles.SelectedTitle.Foreground(blue1).BorderLeftForeground(blue1)
	matchDelegate.Styles.SelectedDesc = matchDelegate.Styles.SelectedDesc.Foreground(blue2).BorderLeftForeground(blue1)
	state.matches = list.New([]list.Item{}, matchDelegate, 0, 0)
	state.matches.Title = "Today's Matches"

	state.currentView = ViewLeagues
	return state
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
			case "enter":
				if s.currentView == ViewLeagues {
					l, ok := s.leagues.SelectedItem().(League)
					if ok {
						s.selectedLeague = l
						s.currentView = ViewMatches
						return s, getMatches(l.id)
					}
				}
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
		return s, nil
	}

	var cmd tea.Cmd
	switch s.currentView {
	case ViewLeagues:
		s.leagues, cmd = s.leagues.Update(msg)
	case ViewMatches:
		s.leagues, cmd = s.matches.Update(msg)
	}

	return s, cmd
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
