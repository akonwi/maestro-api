package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BetDash struct {
	bettingPerformance BettingPerformance
	betHistoryTable    table.Model
	allBetsData        []Bet // Store bet data for table operations
	overviewMaxHeight  int   // Maximum height for overview section
	tableMaxHeight     int   // Maximum height for table section
	terminalWidth      int   // Current terminal width for card sizing
	showDeleteConfirm  bool  // Delete confirmation state
	betToDelete        *Bet  // Bet being deleted
}

// NewBetDash creates a new BetDash instance
func NewBetDash() BetDash {
	// Initialize bet history table
	columns := []table.Column{
		{Title: "Date", Width: 12},
		{Title: "Match", Width: 35},
		{Title: "Bet", Width: 30},
		{Title: "Odds", Width: 8},
		{Title: "Wager", Width: 10},
		{Title: "Result", Width: 10},
		{Title: "P&L", Width: 10},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(20), // Will be dynamically sized in window resize
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return BetDash{
		betHistoryTable: t,
	}
}

// Init implements tea.Model
func (bd BetDash) Init() tea.Cmd {
	return tea.Batch(loadAllBets(), loadBettingPerformance())
}

// Update implements tea.Model
func (bd BetDash) Update(msg tea.Msg) (BetDash, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle delete confirmation
		if bd.showDeleteConfirm {
			switch msg.String() {
			case "y", "Y":
				bd.showDeleteConfirm = false
				if bd.betToDelete != nil {
					betID := bd.betToDelete.id
					bd.betToDelete = nil
					return bd, deleteBet(betID)
				}
				return bd, nil
			case "n", "N", "esc":
				bd.showDeleteConfirm = false
				bd.betToDelete = nil
				return bd, nil
			}
			return bd, nil
		}

		// Handle bet result updates
		switch msg.String() {
		case "w":
			return bd, bd.updateBetResult(Win)
		case "l":
			return bd, bd.updateBetResult(Lose)
		case "p":
			return bd, bd.updateBetResult(Push)
		case "backspace":
			return bd, bd.deleteBet()
		}
	}

	// Update table
	bd.betHistoryTable, cmd = bd.betHistoryTable.Update(msg)
	return bd, cmd
}

// View implements tea.Model
func (bd BetDash) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("Betting Performance")

	overview := bd.renderPerformanceOverview()
	betHistory := bd.renderBetHistoryTable()

	// Constrain overview section to exactly 50% of viewport
	maxHeight := bd.overviewMaxHeight
	if maxHeight == 0 {
		maxHeight = 8 // Fallback if not set yet
	}

	constrainedOverview := lipgloss.NewStyle().
		Height(maxHeight). // Use exactly 50% of viewport height
		Render(overview)

	// Use minimal margins to maximize width usage
	return lipgloss.NewStyle().Margin(0, 1).Render(
		lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			constrainedOverview,
			"",
			betHistory,
		),
	)
}

// SetSize updates the table size when window is resized
func (bd *BetDash) SetSize(width, height int) {
	// Split available height evenly between overview and table sections
	sectionHeight := height / 2

	// Overview section gets 50% of viewport
	bd.overviewMaxHeight = sectionHeight

	// Table section gets 50% of viewport (minus small space for help text)
	helpHeight := 1 // Just one line for help
	tableHeight := sectionHeight - helpHeight
	if tableHeight < 3 {
		tableHeight = 3 // Minimum table height
	}

	bd.betHistoryTable.SetWidth(width)
	bd.betHistoryTable.SetHeight(tableHeight)
	bd.tableMaxHeight = tableHeight
	bd.terminalWidth = width
}

// LoadData loads betting performance and bets data
func (bd *BetDash) LoadData(performance BettingPerformance, bets []Bet) {
	bd.bettingPerformance = performance
	bd.allBetsData = bets
	rows := bd.createTableRowsFromBets(bets)
	bd.betHistoryTable.SetRows(rows)
}

// updateBetResult updates the result of the currently selected bet
func (bd BetDash) updateBetResult(result BetOutcome) tea.Cmd {
	if len(bd.allBetsData) == 0 {
		return nil
	}

	selectedRow := bd.betHistoryTable.Cursor()
	if selectedRow >= len(bd.allBetsData) {
		return nil
	}

	selectedBet := bd.allBetsData[selectedRow]
	return updateTableBetResult(selectedBet.id, result)
}

// deleteBet initiates bet deletion with confirmation
func (bd *BetDash) deleteBet() tea.Cmd {
	if len(bd.allBetsData) == 0 {
		return nil
	}

	selectedRow := bd.betHistoryTable.Cursor()
	if selectedRow >= len(bd.allBetsData) {
		return nil
	}

	selectedBet := bd.allBetsData[selectedRow]
	bd.betToDelete = &selectedBet
	bd.showDeleteConfirm = true
	return nil
}

func (bd BetDash) renderPerformanceOverview() string {
	perf := bd.bettingPerformance

	// Calculate card width to fill the full viewport width
	// Use actual terminal width for dynamic sizing
	terminalWidth := bd.terminalWidth
	if terminalWidth == 0 {
		terminalWidth = 120 // Fallback if not set yet
	}

	// Account for: margins (4 chars) + 3 spacers between cards (2 chars each = 6 total)
	// Formula: (terminalWidth - margins - spacers) / 4 cards
	margins := 4 // Left + right margins from lipgloss
	spacers := 6 // 3 spacers of 2 chars each between 4 cards
	cardWidth := (terminalWidth - margins - spacers) / 4

	// Ensure reasonable width bounds
	if cardWidth < 18 {
		cardWidth = 18 // Minimum for readability
	} else if cardWidth > 30 {
		cardWidth = 30 // Maximum to prevent overly wide cards
	}

	// Create the overview cards in a 4x2 grid
	card1 := bd.createStatsCardWithWidth("Total Bets", fmt.Sprintf("%d", perf.totalBets), "", cardWidth)
	card2 := bd.createStatsCardWithWidth("Total Wagered", fmt.Sprintf("$%.2f", perf.totalWagered), "Amount bet", cardWidth)
	card3 := bd.createStatsCardWithWidth("Net Profit", fmt.Sprintf("$%.2f", perf.netProfit), "Profit", cardWidth)
	card4 := bd.createStatsCardWithWidth("ROI", fmt.Sprintf("%.1f%%", perf.roi), "Return on investment", cardWidth)

	card5 := bd.createStatsCardWithWidth("Win Rate", fmt.Sprintf("%.1f%%", perf.winRate), fmt.Sprintf("%d settled bets", perf.totalBets-perf.pendingBets), cardWidth)
	card6 := bd.createStatsCardWithWidth("Total Winnings", fmt.Sprintf("$%.2f", perf.totalWinnings), "Gross winnings", cardWidth)
	card7 := bd.createStatsCardWithWidth("Total Losses", fmt.Sprintf("$%.2f", perf.totalLosses), "Amount lost", cardWidth)
	card8 := bd.createStatsCardWithWidth("Pending Bets", fmt.Sprintf("%d", perf.pendingBets), "Awaiting results", cardWidth)

	row1 := lipgloss.JoinHorizontal(lipgloss.Top, card1, "  ", card2, "  ", card3, "  ", card4)
	row2 := lipgloss.JoinHorizontal(lipgloss.Top, card5, "  ", card6, "  ", card7, "  ", card8)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Render("Overview"),
		"",
		row1,
		"",
		row2,
	)
}

func (bd BetDash) renderBetHistoryTable() string {
	title := lipgloss.NewStyle().Bold(true).Render("Recent Activity")

	if bd.showDeleteConfirm {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			bd.renderDeleteConfirmation(),
		)
	}

	helpText := bd.renderBettingOverviewHelp()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		bd.betHistoryTable.View(),
		"",
		helpText,
	)
}

func (bd BetDash) createStatsCardWithWidth(title, value, subtitle string, width int) string {
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	valueStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Italic(true)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		valueStyle.Render(value),
	)

	if subtitle != "" {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			content,
			subtitleStyle.Render(subtitle),
		)
	}

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(width).
		Height(4).
		Render(content)
}

func (bd BetDash) createTableRowsFromBets(bets []Bet) []table.Row {
	rows := make([]table.Row, len(bets))

	for i, bet := range bets {
		// Calculate P&L
		var pnl string
		if bet.result == Win {
			// Calculate winnings based on odds
			winAmount := bd.calculateWinnings(bet.amount, bet.odds)
			profit := winAmount - bet.amount
			pnl = fmt.Sprintf("+$%.2f", profit)
		} else if bet.result == Lose {
			pnl = fmt.Sprintf("-$%.2f", bet.amount)
		} else if bet.result == Push {
			pnl = "$0.00"
		} else {
			pnl = "-"
		}

		// Format odds
		oddsStr := fmt.Sprintf("%+d", bet.odds)

		// Format result with color coding
		resultStr := string(bet.result)

		// Format match date
		date := formatDate(bet.matchDate)

		// Format match name
		matchName := fmt.Sprintf("%s vs %s", bet.homeTeamName, bet.awayTeamName)

		rows[i] = table.Row{
			date,                             // Date
			matchName,                        // Match
			bet.name,                         // Bet name
			oddsStr,                          // Odds
			fmt.Sprintf("$%.2f", bet.amount), // Wager
			resultStr,                        // Result
			pnl,                              // P&L
		}
	}

	return rows
}

func (bd BetDash) calculateWinnings(amount float64, odds int) float64 {
	if odds > 0 {
		// Positive odds: +150 means bet $100 to win $150
		return amount + (amount * float64(odds) / 100.0)
	} else {
		// Negative odds: -150 means bet $150 to win $100
		return amount + (amount * 100.0 / float64(-odds))
	}
}


func (bd BetDash) renderBettingOverviewHelp() string {
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	helpItems := []string{
		"↑/↓: navigate",
		"w: mark win",
		"l: mark lose",
		"p: mark push",
		"⌫: delete bet",
		"esc: back to leagues",
	}

	return helpStyle.Render("• " + strings.Join(helpItems, " • "))
}

func (bd BetDash) renderDeleteConfirmation() string {
	if bd.betToDelete == nil {
		return ""
	}

	title := lipgloss.NewStyle().Bold(true).Render("Delete Bet")
	betName := bd.betToDelete.name
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
