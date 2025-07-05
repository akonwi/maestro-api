package main

import (
	"database/sql"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/mattn/go-sqlite3"
)

type DBConnected = *sql.DB

type ErrMsg struct{ err error }

func (msg ErrMsg) Error() string {
	return msg.Error()
}

func openDB() tea.Msg {
	conn, err := sql.Open("sqlite3", "../db.sqlite")
	if err != nil {
		return ErrMsg{err: err}
	}
	return conn
}
