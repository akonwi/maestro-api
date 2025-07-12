package main

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BetOutcome = string

const (
	Pending BetOutcome = "pending"
	Win     BetOutcome = "win"
	Lose    BetOutcome = "lose"
	Push    BetOutcome = "push"
)

type Bet struct {
	id      int
	matchID int
	name    string
	line    float64
	amount  float64
	odds    int
	result  BetOutcome
}

// Implement list.Item interface for Bet
func (b Bet) FilterValue() string { return b.name }
func (b Bet) Title() string       { return b.name }
func (b Bet) Description() string {
	return fmt.Sprintf("%s : %d : %v", strconv.FormatFloat(b.line, 'f', 2, 64), b.odds, b.result)
}

type BetForm struct {
	nameInput   textinput.Model
	lineInput   textinput.Model
	amountInput textinput.Model
	oddsInput   textinput.Model
	focused     int
}

func newBetForm() BetForm {
	betForm := &BetForm{}
	betForm.nameInput = textinput.New()
	betForm.nameInput.Placeholder = "Bet name (e.g., Over 2.5 goals)"
	betForm.nameInput.Focus()
	betForm.nameInput.CharLimit = 100
	betForm.nameInput.Width = 40

	betForm.lineInput = textinput.New()
	betForm.lineInput.Placeholder = "Line (e.g., 2.5)"
	betForm.lineInput.CharLimit = 10
	betForm.lineInput.Width = 20

	betForm.amountInput = textinput.New()
	betForm.amountInput.Placeholder = "Amount (e.g., 100.00)"
	betForm.amountInput.CharLimit = 10
	betForm.amountInput.Width = 20

	betForm.oddsInput = textinput.New()
	betForm.oddsInput.Placeholder = "Odds (e.g., -110)"
	betForm.oddsInput.CharLimit = 10
	betForm.oddsInput.Width = 20

	betForm.focused = 0
	return *betForm
}

func (f BetForm) isDirty() bool {
	if f.nameInput.Value() != "" || f.lineInput.Value() != "" || f.amountInput.Value() != "" || f.oddsInput.Value() != "" {
		return true
	}
	return false
}

func (f BetForm) escapeDesc() string {
	if f.isDirty() {
		return "Clear"
	}

	return "Close"
}

/*
 * -------------
 * Commands
 * -------------
 */

type BetSaved = Bet

type BetsLoaded = []Bet

type BetResultUpdated struct {
	betID  int
	result BetOutcome
}

func loadBets(matchID int) tea.Cmd {
	return func() tea.Msg {
		rows, err := db.Query("SELECT id, name, line, amount, odds, result FROM bets WHERE match_id = ?", matchID)
		if err != nil {
			return ErrMsg{err: fmt.Errorf("failed to load bets: %v", err)}
		}
		defer rows.Close()

		var bets []Bet
		for rows.Next() {
			var bet Bet
			err := rows.Scan(&bet.id, &bet.name, &bet.line, &bet.amount, &bet.odds, &bet.result)
			if err != nil {
				continue
			}
			bet.matchID = matchID
			bets = append(bets, bet)
		}

		return BetsLoaded(bets)
	}
}

func updateBetResult(betID int, result BetOutcome) tea.Cmd {
	return func() tea.Msg {
		_, err := db.Exec("UPDATE bets SET result = ? WHERE id = ?", result, betID)
		if err != nil {
			return ErrMsg{err: fmt.Errorf("failed to update bet result: %v", err)}
		}

		return BetResultUpdated{betID: betID, result: result}
	}
}

func (s *State) saveBet() tea.Cmd {
	return func() tea.Msg {
		// Validate inputs
		name := s.betForm.nameInput.Value()
		lineStr := s.betForm.lineInput.Value()
		amountStr := s.betForm.amountInput.Value()
		oddsStr := s.betForm.oddsInput.Value()

		if name == "" {
			return ErrMsg{err: fmt.Errorf("bet name is required")}
		}

		if amountStr == "" {
			return ErrMsg{err: fmt.Errorf("bet amount is required")}
		}

		var line float64
		var amount float64
		var odds int
		var err error

		if lineStr != "" {
			line, err = strconv.ParseFloat(lineStr, 64)
			if err != nil {
				return ErrMsg{err: fmt.Errorf("invalid line value: %v", err)}
			}
		}

		amount, err = strconv.ParseFloat(amountStr, 64)
		if err != nil {
			return ErrMsg{err: fmt.Errorf("invalid amount value: %v", err)}
		}

		if oddsStr != "" {
			odds, err = strconv.Atoi(oddsStr)
			if err != nil {
				return ErrMsg{err: fmt.Errorf("invalid odds value: %v", err)}
			}
		}

		selectedMatch := s.getCurrentMatch()
		// Insert bet into database
		_, err = db.Exec("INSERT INTO bets (match_id, name, line, amount, odds, result) VALUES (?, ?, ?, ?, ?, ?)",
			selectedMatch.id, name, line, amount, odds, Pending)
		if err != nil {
			return ErrMsg{err: fmt.Errorf("failed to save bet: %v", err)}
		}

		return BetSaved(Bet{matchID: selectedMatch.id, name: name, line: line, amount: amount, odds: odds, result: Pending})
	}
}

func (s *State) updateBetResult(result BetOutcome) tea.Cmd {
	if len(s.currentMatchBets.Items()) == 0 {
		return nil
	}

	selectedBet := s.currentMatchBets.SelectedItem().(Bet)
	return updateBetResult(selectedBet.id, result)
}

/*
 * -------------
 * Updates
 * -------------
 */
func (s *State) updateBetFormInputs(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch s.betForm.focused {
	case 0:
		s.betForm.nameInput, cmd = s.betForm.nameInput.Update(msg)
	case 1:
		s.betForm.lineInput, cmd = s.betForm.lineInput.Update(msg)
	case 2:
		s.betForm.oddsInput, cmd = s.betForm.oddsInput.Update(msg)
	case 3:
		s.betForm.amountInput, cmd = s.betForm.amountInput.Update(msg)
	}
	return cmd
}

func handleBetFormKey(s *State, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, defaultKeyMap.Back):
		if s.betForm.isDirty() {
			s.resetBetForm()
			return s, nil
		}
		s.showBetForm = false
		return s, nil
	case msg.String() == "tab":
		s.betForm.focused = (s.betForm.focused + 1) % 4
		s.updateBetFormFocus()
		return s, nil
	case msg.String() == "enter":
		return s, s.saveBet()
	}
	return s, s.updateBetFormInputs(msg)
}

/*
 * -------------
 * Views
 * -------------
 */

func (s *State) renderBetForm() string {
	title := lipgloss.NewStyle().Bold(true).Render("Place Bet")

	form := fmt.Sprintf(`%s

Name: %s
Line: %s
Odds: %s
Amount: %s

Tab: Next field | Esc: %s | Enter: Save`,
		title,
		s.betForm.nameInput.View(),
		s.betForm.lineInput.View(),
		s.betForm.oddsInput.View(),
		s.betForm.amountInput.View(),
		s.betForm.escapeDesc(),
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		Render(form)
}
