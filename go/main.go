package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

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
func (m Match) Description() string {
	score := m.score()
	dateStr := m.formatDate()

	// Create a description with score on left and date on right
	// Assuming a width of about 50 characters for the description area
	totalWidth := 50
	scoreLen := len(score)
	dateLen := len(dateStr)

	if scoreLen+dateLen >= totalWidth {
		// If too long, just show score and date separated by space
		return fmt.Sprintf("%s %s", score, dateStr)
	}

	// Right-align the date
	padding := totalWidth - scoreLen - dateLen
	return fmt.Sprintf("%s%s%s", score, strings.Repeat(" ", padding), dateStr)
}

func (m Match) isNil() bool {
	return m == (Match{})
}

func (m Match) score() string {
	if m.status == "NS" {
		return "TBD"
	}
	return fmt.Sprintf("%d - %d (%s)", m.homeGoals, m.awayGoals, m.status)
}

func (m Match) formatDate() string {
	// Handle full timestamp format like "2025-07-12T23:30:00+00:00"
	// First try to parse as full timestamp
	t, err := time.Parse(time.RFC3339, m.date)
	if err != nil {
		// If that fails, try just the date part
		t, err = time.Parse("2006-01-02", m.date)
		if err != nil {
			// If parsing fails, return the original date
			return m.date
		}
	}

	// Format as M/D/YYYY
	return t.Format("01/02/2006")
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

type KeyMap struct {
	Back        key.Binding
	GameStatus  key.Binding
	ShowBetForm key.Binding
	ToggleBets  key.Binding
	BetWin      key.Binding
	BetLose     key.Binding
	BetPush     key.Binding
	DeleteBet   key.Binding
}

var defaultKeyMap = KeyMap{
	Back: key.NewBinding(key.WithKeys("esc")),
	GameStatus: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "toggle played/unplayed"),
	),
	ToggleBets: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "focus bets|matches"),
	),
	ShowBetForm: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "bet form"),
	),
	BetWin: key.NewBinding(
		key.WithKeys("w"),
		key.WithHelp("w", "mark win"),
	),
	BetLose: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "mark lose"),
	),
	BetPush: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "mark push"),
	),
	DeleteBet: key.NewBinding(
		key.WithKeys("backspace"),
		key.WithHelp("⌫", "delete bet"),
	),
}

type State struct {
	db  *sql.DB
	err error

	loading bool
	spinner spinner.Model

	currentView ViewState

	leagues        list.Model
	selectedLeague League

	/* Matches Screen */
	matches           list.Model
	showPlayedMatches bool
	headToHeadStats   HeadToHeadStats
	showBetForm       bool
	currentMatchBets  list.Model
	betForm           BetForm
	betsFocused       bool
	showDeleteConfirm bool
	betToDelete       *Bet
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

	state.currentMatchBets = list.New([]list.Item{}, listDelegate, 0, 0)
	state.currentMatchBets.Title = "Match Bets"
	state.currentMatchBets.SetShowHelp(false)

	// Add help key bindings for matches list
	// betKey := key.NewBinding(
	// 	key.WithKeys("b"),
	// 	key.WithHelp("b", "place bet"),
	// )
	// viewBetsKey := key.NewBinding(
	// 	key.WithKeys("v"),
	// 	key.WithHelp("v", "view bets"),
	// )

	// Priority keys shown in main help view
	state.currentMatchBets.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{defaultKeyMap.ToggleBets, defaultKeyMap.ShowBetForm, defaultKeyMap.BetWin, defaultKeyMap.BetLose, defaultKeyMap.BetPush, defaultKeyMap.DeleteBet}
	}

	// // Additional keys shown in "more" help section
	// state.matches.AdditionalFullHelpKeys = func() []key.Binding {
	// 	return []key.Binding{toggleKey, betKey, viewBetsKey}
	// }

	state.currentView = ViewLeagues
	state.showPlayedMatches = false
	state.betForm = newBetForm()
	return state
}

func (s State) getCurrentMatch() Match {
	var match Match

	if len(s.matches.VisibleItems()) > 0 {
		match = s.matches.VisibleItems()[s.matches.Index()].(Match)
	}

	return match
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

func (s *State) toggleBetFocus() {
	s.betsFocused = !s.betsFocused
	s.matches.SetShowHelp(!s.betsFocused)
	s.currentMatchBets.SetShowHelp(s.betsFocused)
}

// Update implements tea.Model.
func (s *State) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ErrMsg:
		s.err = msg
		return s, nil
	case tea.KeyMsg:
		{
			if s.showBetForm {
				return handleBetFormKey(s, msg)
			}

			// Handle delete confirmation
			if s.showDeleteConfirm {
				switch msg.String() {
				case "y", "Y":
					s.showDeleteConfirm = false
					if s.betToDelete != nil {
						betID := s.betToDelete.id
						s.betToDelete = nil
						return s, deleteBet(betID)
					}
					return s, nil
				case "n", "N", "esc":
					s.showDeleteConfirm = false
					s.betToDelete = nil
					return s, nil
				}
				return s, nil
			}

			// Handle bet result updates when focused on bet list
			if s.betsFocused && s.currentView == ViewMatches {
				switch msg.String() {
				case "w":
					return s, s.updateBetResult(Win)
				case "l":
					return s, s.updateBetResult(Lose)
				case "p":
					return s, s.updateBetResult(Push)
				case "backspace":
					if len(s.currentMatchBets.Items()) > 0 {
						selectedBet := s.currentMatchBets.SelectedItem().(Bet)
						s.betToDelete = &selectedBet
						s.showDeleteConfirm = true
						return s, nil
					}
				}
			}

			switch {
			case key.Matches(msg, defaultKeyMap.Back):
				if s.showBetForm {
					// if s.betForm.isDirty() {
					//   // s.resetForm()
					// }

					s.showBetForm = false
					return s, nil
				}
				if s.betsFocused {
					s.toggleBetFocus()
					return s, nil
				}
				if s.currentView == ViewMatches {
					s.currentView = ViewLeagues
					return s, nil
				}
			case key.Matches(msg, defaultKeyMap.GameStatus):
				if s.currentView == ViewMatches {
					s.showPlayedMatches = !s.showPlayedMatches
					s.loading = true
					s.updateMatchesTitle()
					return s, getMatches(s.selectedLeague.id, s.showPlayedMatches)
				}
			case key.Matches(msg, defaultKeyMap.ToggleBets):
				if s.currentView == ViewMatches {
					s.toggleBetFocus()
					return s, nil
				}
			case key.Matches(msg, defaultKeyMap.ShowBetForm):
				if s.betsFocused {
					s.showBetForm = true
					return s, nil
				}
			}

			switch keypress := msg.String(); keypress {
			case "ctrl+c":
				return s, tea.Quit
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
				if s.currentView == ViewMatches {
					return s, getMatchDetails(s.getCurrentMatch())
				}
				return s, nil
			}
		}
	case tea.WindowSizeMsg:
		w, h := docStyle.GetFrameSize()
		s.leagues.SetSize(msg.Width-w, msg.Height-h)
		s.matches.SetSize(msg.Width-w, msg.Height-h)
		s.currentMatchBets.SetSize((msg.Width-w)/2, (msg.Height-h)/2)
	case DBConnected:
		s.db = msg
		return s, getLeagues
	case LeaguesLoaded:
		items := make([]list.Item, len(msg))
		for i, league := range msg {
			items[i] = league
		}
		return s, s.leagues.SetItems(items)
	case MatchesLoaded:
		items := make([]list.Item, len(msg))
		for i, match := range msg {
			items[i] = match
		}
		s.loading = false

		return s, tea.Batch(s.matches.SetItems(items), getMatchDetails(s.getCurrentMatch()))
	case StatsLoaded:
		s.headToHeadStats = msg
		s.loading = false
		return s, nil
	case BetSaved:
		s.showBetForm = false
		s.resetBetForm()
		return s, nil
	case BetsLoaded:
		items := make([]list.Item, len(msg))
		for i, bet := range msg {
			items[i] = bet
		}
		s.loading = false
		return s, s.currentMatchBets.SetItems(items)
	case BetResultUpdated:
		// Reload bets to reflect the updated result
		return s, loadBets(s.getCurrentMatch().id)
	case BetDeleted:
		// Reload bets to reflect the deletion
		return s, loadBets(s.getCurrentMatch().id)
	}

	var listCmd tea.Cmd
	var spinnerCmd tea.Cmd
	var formCmd tea.Cmd
	switch s.currentView {
	case ViewLeagues:
		s.leagues, listCmd = s.leagues.Update(msg)
	case ViewMatches:
		if s.showBetForm {
			return s, s.updateBetFormInputs(msg)
		}
		if s.betsFocused {
			s.currentMatchBets, listCmd = s.currentMatchBets.Update(msg)
		} else {
			s.matches, listCmd = s.matches.Update(msg)
			return s, tea.Batch(listCmd, getMatchDetails(s.getCurrentMatch()))
		}
	}
	s.spinner, spinnerCmd = s.spinner.Update(msg)

	return s, tea.Batch(listCmd, spinnerCmd, formCmd)
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
		return s.renderMatchSplitView()
	default:
		return docStyle.Render(s.leagues.View())
	}
}

func (s *State) resetBetForm() {
	s.betForm.nameInput.SetValue("")
	s.betForm.lineInput.SetValue("")
	s.betForm.amountInput.SetValue("")
	s.betForm.oddsInput.SetValue("")
	s.betForm.focused = 0
	s.updateBetFormFocus()
}

func (s *State) updateBetFormFocus() {
	s.betForm.nameInput.Blur()
	s.betForm.lineInput.Blur()
	s.betForm.oddsInput.Blur()
	s.betForm.amountInput.Blur()

	switch s.betForm.focused {
	case 0:
		s.betForm.nameInput.Focus()
	case 1:
		s.betForm.lineInput.Focus()
	case 2:
		s.betForm.oddsInput.Focus()
	case 3:
		s.betForm.amountInput.Focus()
	}
}

func (s *State) renderMatchSplitView() string {
	matchesView := s.matches.View()

	// Create right column with stats on top and bets on bottom
	rightColumnView := s.renderDetailMatchColumn()

	// Split the viewport horizontally
	termWidth := s.matches.Width()
	matchesWidth := termWidth / 2
	rightColumnWidth := termWidth - matchesWidth

	// Create side-by-side layout
	matchesStyle := lipgloss.NewStyle().Width(matchesWidth)
	rightColumnStyle := lipgloss.NewStyle().Width(rightColumnWidth)

	return docStyle.Render(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			matchesStyle.Render(matchesView),
			rightColumnStyle.Render(rightColumnView),
		),
	)
}

func (s *State) renderDetailMatchColumn() string {
	if s.getCurrentMatch().isNil() {
		return ""
	}

	// Calculate 50% height for each section
	rightColumnHeight := s.matches.Height()
	sectionHeight := rightColumnHeight / 2

	// Get the stats view
	statsView := s.renderHeadToHeadStats()

	// Get the bets view
	betsView := s.renderMatchBetsSection()

	// Apply height constraints to make each section exactly 50%
	statsStyle := lipgloss.NewStyle().Height(sectionHeight)
	betsStyle := lipgloss.NewStyle().Height(sectionHeight)

	// Create vertical split: stats on top, bets on bottom
	return lipgloss.JoinVertical(
		lipgloss.Left,
		statsStyle.Render(statsView),
		betsStyle.Render(betsView),
	)
}

func (s *State) renderMatchBetsSection() string {
	if s.showBetForm {
		return s.renderBetForm()
	}

	if s.showDeleteConfirm {
		return s.renderDeleteConfirmation()
	}

	return s.currentMatchBets.View()
}

func (s *State) renderDeleteConfirmation() string {
	if s.betToDelete == nil {
		return ""
	}

	title := lipgloss.NewStyle().Bold(true).Render("Delete Bet")
	betName := s.betToDelete.name
	if betName == "" {
		betName = "Unnamed bet"
	}

	confirmation := fmt.Sprintf(`%s

Are you sure you want to delete:
"%s"?

Y: Yes, delete | N: No, cancel`, title, betName)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")). // Red border
		Padding(1).
		Render(confirmation)
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
