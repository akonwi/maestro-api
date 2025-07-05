package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/mattn/go-sqlite3"
)

func wrapError(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

var db *sql.DB

type DBConnected = *sql.DB

type ErrMsg struct{ err error }

func (msg ErrMsg) Error() string {
	return msg.err.Error()
}

func openDB() tea.Msg {
	conn, err := sql.Open("sqlite3", "../db.sqlite")
	if err != nil {
		return ErrMsg{err: err}
	}
	db = conn
	return conn
}

type LeaguesLoaded = []League

func getLeagues() tea.Msg {
	if db == nil {
		return ErrMsg{err: fmt.Errorf("No database connection")}
	}

	rows, err := db.Query("SELECT id, name, code FROM leagues")
	if err != nil {
		return ErrMsg{err: wrapError(err, "failed to query leagues")}
	}
	defer rows.Close()

	var leagues []League
	for rows.Next() {
		var league League
		err := rows.Scan(&league.id, &league.name, &league.code)
		if err != nil {
			return ErrMsg{err: wrapError(err, "failed to scan league")}
		}
		leagues = append(leagues, league)
	}

	if err = rows.Err(); err != nil {
		return ErrMsg{err: wrapError(err, "failed to scan leagues")}
	}

	return LeaguesLoaded(leagues)
}

type MatchesLoaded = []Match

func toDateString(t time.Time) string {
	y, m, d := t.Date()
	return fmt.Sprintf("%d-%02d-%02d", y, m, d)
}

func getMatches(leagueID int) tea.Cmd {
	return func() tea.Msg {
		today := toDateString(time.Now())
		url := fmt.Sprintf("https://v3.football.api-sports.io/fixtures?season=2025&league=%d&date=%s", leagueID, today)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return ErrMsg{err: err}
		}

		req.Header.Set("x-rapidapi-key", "91be9b12c36d01fd71847355d020c8d7")
		req.Header.Set("Accept", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return ErrMsg{err: err}
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return ErrMsg{err: wrapError(err, "unable to read response")}
		}

		var fixturesResp FixturesResponse
		if err := json.Unmarshal(body, &fixturesResp); err != nil {
			return ErrMsg{err: err}
		}

		// Check for API errors
		if len(fixturesResp.Errors) > 0 {
			var errorMsg string
			for _, apiError := range fixturesResp.Errors {
				if errorMsg != "" {
					errorMsg += "; "
				}
				errorMsg += fmt.Sprintf("%s: %s", apiError.Field, apiError.Message)
			}
			return ErrMsg{err: fmt.Errorf("API error: %s", errorMsg)}
		}

		var matches []Match
		for _, entry := range fixturesResp.Response {
			var homeGoals int8
			var awayGoals int8
			if entry.Goals.Home != nil {
				homeGoals = int8(*entry.Goals.Home)
			}
			if entry.Goals.Away != nil {
				awayGoals = int8(*entry.Goals.Away)
			}

			match := &Match{
				id:   entry.Fixture.ID,
				date: toDateString(entry.Fixture.Date),
				homeTeam: Team{
					id:   entry.Teams.Home.ID,
					name: entry.Teams.Home.Name,
				},
				awayTeam: Team{
					id:   entry.Teams.Away.ID,
					name: entry.Teams.Away.Name,
				},
				homeGoals: homeGoals,
				awayGoals: awayGoals,
				status:    entry.Fixture.Status.Short,
			}
			matches = append(matches, *match)
		}

		return MatchesLoaded(matches)
	}
}
