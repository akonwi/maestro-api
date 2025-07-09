package main

import (
	"database/sql"
	"fmt"

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

func getMatches(leagueID int, showPlayedMatches bool) tea.Cmd {
	return func() tea.Msg {
		if db == nil {
			return ErrMsg{err: fmt.Errorf("database not connected")}
		}

		// Query matches for the given league
		var status string
		if showPlayedMatches {
			status = "FT"
		} else {
			status = "NS"
		}

		rows, err := db.Query(
			`SELECT
				m.id, m.date, m.timestamp, m.league_id, m.status, m.home_team_id, m.away_team_id, m.home_goals, m.away_goals, m.winner_id,
				ht.name as home_team_name,
				at.name as away_team_name
			FROM matches m
			JOIN teams ht ON m.home_team_id = ht.id JOIN teams at ON m.away_team_id = at.id
			WHERE m.status = ?
			AND m.league_id = ?
			ORDER BY m.timestamp ASC`,
			status, leagueID,
		)
		if err != nil {
			return ErrMsg{err: err}
		}
		defer rows.Close()

		var matches []Match
		for rows.Next() {
			var match Match
			var winnerId sql.NullInt64
			var homeTeamName, awayTeamName string
			err := rows.Scan(&match.id, &match.date, &match.timestamp, &match.leagueId, &match.status, &match.homeTeamId, &match.awayTeamId, &match.homeGoals, &match.awayGoals, &winnerId, &homeTeamName, &awayTeamName)
			if err != nil {
				return ErrMsg{err: err}
			}

			if winnerId.Valid {
				winnerIdInt := int(winnerId.Int64)
				match.winnerId = &winnerIdInt
			}

			// Assign team names to match struct
			match.homeTeamName = homeTeamName
			match.awayTeamName = awayTeamName

			matches = append(matches, match)
		}

		if err = rows.Err(); err != nil {
			return ErrMsg{err: err}
		}

		return MatchesLoaded(matches)
	}
}
