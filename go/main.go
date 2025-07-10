package main

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
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
	ViewStats
	ViewBetForm
	ViewBets
)

type HeadToHeadStats struct {
	homeTeamName           string
	awayTeamName           string
	homeWins               int
	awayWins               int
	draws                  int
	homeCleanSheets        int
	awayCleanSheets        int
	home1GoalConceded      int
	away1GoalConceded      int
	home2PlusGoalsConceded int
	away2PlusGoalsConceded int
	homeGoalsFor           int
	homeGoalsAgainst       int
	awayGoalsFor           int
	awayGoalsAgainst       int
	homeAvgGoalsFor        float64
	homeAvgGoalsAgainst    float64
	awayAvgGoalsFor        float64
	awayAvgGoalsAgainst    float64
	homeGamesPlayed        int
	awayGamesPlayed        int
}

type BetForm struct {
	nameInput textinput.Model
	lineInput textinput.Model
	oddsInput textinput.Model
	focused   int
}

type BetSaved struct {
	bet Bet
}

type State struct {
	db                *sql.DB
	err               error
	leagues           list.Model
	matches           list.Model
	selectedLeague    League
	selectedMatch     Match
	currentView       ViewState
	loading           bool
	spinner           spinner.Model
	showPlayedMatches bool
	headToHeadStats   HeadToHeadStats
	betForm           BetForm
	showBetForm       bool
	bets              []Bet
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

	// Add help key bindings for matches list
	toggleKey := key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "toggle played/unplayed"),
	)
	betKey := key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "place bet"),
	)
	viewBetsKey := key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "view bets"),
	)
	state.matches.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{toggleKey, betKey, viewBetsKey}
	}

	state.currentView = ViewLeagues
	state.showPlayedMatches = false
	state.initBetForm()
	return state
}

func (s *State) initBetForm() {
	s.betForm.nameInput = textinput.New()
	s.betForm.nameInput.Placeholder = "Bet name (e.g., Over 2.5 goals)"
	s.betForm.nameInput.Focus()
	s.betForm.nameInput.CharLimit = 100
	s.betForm.nameInput.Width = 40

	s.betForm.lineInput = textinput.New()
	s.betForm.lineInput.Placeholder = "Line (e.g., 2.5)"
	s.betForm.lineInput.CharLimit = 10
	s.betForm.lineInput.Width = 20

	s.betForm.oddsInput = textinput.New()
	s.betForm.oddsInput.Placeholder = "Odds (e.g., -110)"
	s.betForm.oddsInput.CharLimit = 10
	s.betForm.oddsInput.Width = 20

	s.betForm.focused = 0
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
				if s.currentView == ViewStats {
					if s.showBetForm {
						s.showBetForm = false
						s.resetBetForm()
						return s, nil
					}
					s.currentView = ViewMatches
					return s, nil
				}
				if s.currentView == ViewBets {
					s.currentView = ViewMatches
					return s, nil
				}
				if s.currentView == ViewBetForm {
					s.currentView = ViewStats
					s.showBetForm = false
					s.resetBetForm()
					return s, nil
				}
			case "s":
				if s.currentView == ViewMatches {
					s.showPlayedMatches = !s.showPlayedMatches
					s.loading = true
					s.updateMatchesTitle()
					return s, getMatches(s.selectedLeague.id, s.showPlayedMatches)
				}
			case "b":
				if s.currentView == ViewStats && !s.showBetForm {
					s.showBetForm = true
					s.resetBetForm()
					return s, nil
				}
			case "v":
				if s.currentView == ViewStats && !s.showBetForm {
					s.currentView = ViewBets
					return s, nil
				}
			case "tab":
				if s.showBetForm {
					s.betForm.focused = (s.betForm.focused + 1) % 3
					s.updateBetFormFocus()
					return s, nil
				}
			case "enter":
				if s.showBetForm {
					s.loading = true
					return s, s.saveBet()
				}
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
				if s.currentView == ViewMatches {
					m, ok := s.matches.SelectedItem().(Match)
					if ok {
						s.selectedMatch = m
						s.currentView = ViewStats
						s.loading = true
						return s, getHeadToHeadStats(m.homeTeamId, m.awayTeamId, m.homeTeamName, m.awayTeamName)
					}
				}
				if s.currentView == ViewStats {
					m, ok := s.matches.SelectedItem().(Match)
					if ok {
						s.selectedMatch = m
						s.loading = true
						return s, getHeadToHeadStats(m.homeTeamId, m.awayTeamId, m.homeTeamName, m.awayTeamName)
					}
				}
				return s, nil
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		s.leagues.SetSize(msg.Width-h, msg.Height-v)
		if s.currentView == ViewStats {
			// In stats view, matches list takes half the width
			s.matches.SetSize((msg.Width-h)/2, msg.Height-v)
		} else {
			s.matches.SetSize(msg.Width-h, msg.Height-v)
		}
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
	case StatsLoaded:
		s.headToHeadStats = msg
		s.loading = false
		return s, nil
	case BetSaved:
		s.showBetForm = false
		s.resetBetForm()
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
	case ViewStats:
		if s.showBetForm {
			s.updateBetFormInputs(msg)
		}
		s.matches, listCmd = s.matches.Update(msg)
	case ViewBets:
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
	case ViewStats:
		if s.loading {
			return docStyle.Render(s.spinner.View() + " Loading Stats")
		}
		return s.renderSplitView()
	case ViewBets:
		return s.renderBetsView()
	default:
		return docStyle.Render(s.leagues.View())
	}
}

func (s *State) resetBetForm() {
	s.betForm.nameInput.SetValue("")
	s.betForm.lineInput.SetValue("")
	s.betForm.oddsInput.SetValue("")
	s.betForm.focused = 0
	s.updateBetFormFocus()
}

func (s *State) updateBetFormFocus() {
	s.betForm.nameInput.Blur()
	s.betForm.lineInput.Blur()
	s.betForm.oddsInput.Blur()

	switch s.betForm.focused {
	case 0:
		s.betForm.nameInput.Focus()
	case 1:
		s.betForm.lineInput.Focus()
	case 2:
		s.betForm.oddsInput.Focus()
	}
}

func (s *State) updateBetFormInputs(msg tea.Msg) {
	var cmd tea.Cmd
	switch s.betForm.focused {
	case 0:
		s.betForm.nameInput, cmd = s.betForm.nameInput.Update(msg)
	case 1:
		s.betForm.lineInput, cmd = s.betForm.lineInput.Update(msg)
	case 2:
		s.betForm.oddsInput, cmd = s.betForm.oddsInput.Update(msg)
	}
	_ = cmd
}

func (s *State) submitBet() tea.Cmd {
	return nil
}

func (s *State) renderSplitView() string {
	matchesView := s.matches.View()
	var statsView string

	if s.showBetForm {
		statsView = s.renderStatsWithBetForm()
	} else {
		statsView = s.renderHeadToHeadStats()
	}

	// Split the viewport horizontally
	termWidth := s.matches.Width()
	matchesWidth := termWidth / 2
	statsWidth := termWidth - matchesWidth

	// Create side-by-side layout
	matchesStyle := lipgloss.NewStyle().Width(matchesWidth).Padding(0, 1)
	statsStyle := lipgloss.NewStyle().Width(statsWidth).Padding(0, 1)

	return docStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			matchesStyle.Render(matchesView),
			statsStyle.Render(statsView),
		),
	)
}

func (s *State) renderBetsView() string {
	matchesView := s.matches.View()
	betsView := s.renderSavedBets()

	// Split the viewport horizontally
	termWidth := s.matches.Width()
	matchesWidth := termWidth / 2
	betsWidth := termWidth - matchesWidth

	// Create side-by-side layout
	matchesStyle := lipgloss.NewStyle().Width(matchesWidth).Padding(0, 1)
	betsStyle := lipgloss.NewStyle().Width(betsWidth).Padding(0, 1)

	return docStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			matchesStyle.Render(matchesView),
			betsStyle.Render(betsView),
		),
	)
}

func (s *State) renderSavedBets() string {
	title := lipgloss.NewStyle().Bold(true).Render("Saved Bets")

	if len(s.bets) == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			"No bets saved yet.",
			"",
			"Press 'Esc' to go back",
		)
	}

	// Filter bets for current match
	var matchBets []Bet
	for _, bet := range s.bets {
		if bet.matchID == s.selectedMatch.id {
			matchBets = append(matchBets, bet)
		}
	}

	if len(matchBets) == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			"No bets for this match.",
			"",
			"Press 'Esc' to go back",
		)
	}

	var betLines []string
	betLines = append(betLines, title)
	betLines = append(betLines, "")

	for _, bet := range matchBets {
		betLine := fmt.Sprintf("• %s", bet.name)
		if bet.line != 0 {
			betLine += fmt.Sprintf(" (Line: %.1f)", bet.line)
		}
		if bet.odds != 0 {
			betLine += fmt.Sprintf(" (Odds: %+d)", bet.odds)
		}
		betLines = append(betLines, betLine)
	}

	betLines = append(betLines, "")
	betLines = append(betLines, "Press 'Esc' to go back")

	return lipgloss.JoinVertical(lipgloss.Left, betLines...)
}

func (s *State) renderStatsWithBetForm() string {
	statsView := s.renderHeadToHeadStats()
	betFormView := s.renderBetForm()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		statsView,
		"",
		betFormView,
	)
}

func (s *State) renderBetForm() string {
	title := lipgloss.NewStyle().Bold(true).Render("Place Bet")

	form := fmt.Sprintf(`%s

Name: %s
Line: %s
Odds: %s

Tab: Next field | Esc: Cancel | Enter: Save`,
		title,
		s.betForm.nameInput.View(),
		s.betForm.lineInput.View(),
		s.betForm.oddsInput.View(),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Render(form)
}

func (s *State) saveBet() tea.Cmd {
	return func() tea.Msg {
		// Validate inputs
		name := s.betForm.nameInput.Value()
		lineStr := s.betForm.lineInput.Value()
		oddsStr := s.betForm.oddsInput.Value()

		if name == "" {
			return ErrMsg{err: fmt.Errorf("bet name is required")}
		}

		var line float64
		var odds int
		var err error

		if lineStr != "" {
			line, err = strconv.ParseFloat(lineStr, 64)
			if err != nil {
				return ErrMsg{err: fmt.Errorf("invalid line value: %v", err)}
			}
		}

		if oddsStr != "" {
			odds, err = strconv.Atoi(oddsStr)
			if err != nil {
				return ErrMsg{err: fmt.Errorf("invalid odds value: %v", err)}
			}
		}

		// Create and store bet
		bet := Bet{
			id:      len(s.bets) + 1,
			matchID: s.selectedMatch.id,
			name:    name,
			line:    line,
			odds:    odds,
		}

		s.bets = append(s.bets, bet)
		s.showBetForm = false
		s.resetBetForm()

		return BetSaved{bet: bet}
	}
}

func (s *State) renderHeadToHeadStats() string {
	stats := s.headToHeadStats

	// Header with team names
	header := fmt.Sprintf("%s VS %s", stats.homeTeamName, stats.awayTeamName)
	headerStyle := lipgloss.NewStyle().Bold(true).Align(lipgloss.Center)

	// Format each stat line with side-by-side comparison
	lines := []string{
		headerStyle.Render("Team Comparison"),
		"",
		headerStyle.Render(header),
		"",
		fmt.Sprintf("%-15s %20s %15s", fmt.Sprintf("%d", stats.homeGamesPlayed), "Games Played", fmt.Sprintf("%d", stats.awayGamesPlayed)),
		fmt.Sprintf("%-15s %20s %15s", fmt.Sprintf("%d-%d-%d", stats.homeWins, stats.awayWins, stats.draws), "W-L-D Record", fmt.Sprintf("%d-%d-%d", stats.awayWins, stats.homeWins, stats.draws)),
		fmt.Sprintf("%-15s %20s %15s", fmt.Sprintf("%d:%d", stats.homeGoalsFor, stats.homeGoalsAgainst), "Goals (For:Against)", fmt.Sprintf("%d:%d", stats.awayGoalsFor, stats.awayGoalsAgainst)),
		fmt.Sprintf("%-15s %20s %15s", fmt.Sprintf("%+d", stats.homeGoalsFor-stats.homeGoalsAgainst), "Goal Difference", fmt.Sprintf("%+d", stats.awayGoalsFor-stats.awayGoalsAgainst)),
		"",
		fmt.Sprintf("%-15s %20s %15s", fmt.Sprintf("%d", stats.homeCleanSheets), "Clean Sheets", fmt.Sprintf("%d", stats.awayCleanSheets)),
		fmt.Sprintf("%-15s %20s %15s", fmt.Sprintf("%d", stats.home1GoalConceded), "1 Goal Conceded", fmt.Sprintf("%d", stats.away1GoalConceded)),
		fmt.Sprintf("%-15s %20s %15s", fmt.Sprintf("%d", stats.home2PlusGoalsConceded), "2+ Goals Conceded", fmt.Sprintf("%d", stats.away2PlusGoalsConceded)),
		"",
		fmt.Sprintf("%-15s %20s %15s", fmt.Sprintf("%.1f", stats.homeAvgGoalsFor), "Avg Goals For", fmt.Sprintf("%.1f", stats.awayAvgGoalsFor)),
		fmt.Sprintf("%-15s %20s %15s", fmt.Sprintf("%.1f", stats.homeAvgGoalsAgainst), "Avg Goals Against", fmt.Sprintf("%.1f", stats.awayAvgGoalsAgainst)),
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func main() {
	state := newState()
	p := tea.NewProgram(state, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
