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
