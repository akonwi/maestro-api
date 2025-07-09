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

type StatsLoaded = HeadToHeadStats

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

func getHeadToHeadStats(homeTeamId, awayTeamId int, homeTeamName, awayTeamName string) tea.Cmd {
	return func() tea.Msg {
		if db == nil {
			return ErrMsg{err: fmt.Errorf("database not connected")}
		}

		var stats HeadToHeadStats
		stats.homeTeamName = homeTeamName
		stats.awayTeamName = awayTeamName

		// Query all finished matches for home team
		homeRows, err := db.Query(`
			SELECT home_team_id, away_team_id, home_goals, away_goals, winner_id
			FROM matches
			WHERE (home_team_id = ? OR away_team_id = ?) AND status = 'FT'
		`, homeTeamId, homeTeamId)
		if err != nil {
			return ErrMsg{err: err}
		}
		defer homeRows.Close()

		// Process home team stats
		for homeRows.Next() {
			var homeId, awayId, homeGoals, awayGoals int
			var winnerId sql.NullInt64
			err := homeRows.Scan(&homeId, &awayId, &homeGoals, &awayGoals, &winnerId)
			if err != nil {
				return ErrMsg{err: err}
			}

			stats.homeGamesPlayed++

			if homeId == homeTeamId {
				// Home team is playing at home
				stats.homeGoalsFor += homeGoals
				stats.homeGoalsAgainst += awayGoals

				if awayGoals == 0 {
					stats.homeCleanSheets++
				} else if awayGoals == 1 {
					stats.home1GoalConceded++
				} else if awayGoals >= 2 {
					stats.home2PlusGoalsConceded++
				}

				if winnerId.Valid && int(winnerId.Int64) == homeTeamId {
					stats.homeWins++
				} else if winnerId.Valid && int(winnerId.Int64) == awayId {
					// Home team lost
				} else {
					stats.draws++
				}
			} else {
				// Home team is playing away
				stats.homeGoalsFor += awayGoals
				stats.homeGoalsAgainst += homeGoals

				if homeGoals == 0 {
					stats.homeCleanSheets++
				} else if homeGoals == 1 {
					stats.home1GoalConceded++
				} else if homeGoals >= 2 {
					stats.home2PlusGoalsConceded++
				}

				if winnerId.Valid && int(winnerId.Int64) == homeTeamId {
					stats.homeWins++
				} else if winnerId.Valid && int(winnerId.Int64) == homeId {
					// Home team lost
				} else {
					stats.draws++
				}
			}
		}

		// Query all finished matches for away team
		awayRows, err := db.Query(`
			SELECT home_team_id, away_team_id, home_goals, away_goals, winner_id
			FROM matches
			WHERE (home_team_id = ? OR away_team_id = ?) AND status = 'FT'
		`, awayTeamId, awayTeamId)
		if err != nil {
			return ErrMsg{err: err}
		}
		defer awayRows.Close()

		// Process away team stats
		for awayRows.Next() {
			var homeId, awayId, homeGoals, awayGoals int
			var winnerId sql.NullInt64
			err := awayRows.Scan(&homeId, &awayId, &homeGoals, &awayGoals, &winnerId)
			if err != nil {
				return ErrMsg{err: err}
			}

			stats.awayGamesPlayed++

			if homeId == awayTeamId {
				// Away team is playing at home
				stats.awayGoalsFor += homeGoals
				stats.awayGoalsAgainst += awayGoals

				if awayGoals == 0 {
					stats.awayCleanSheets++
				} else if awayGoals == 1 {
					stats.away1GoalConceded++
				} else if awayGoals >= 2 {
					stats.away2PlusGoalsConceded++
				}

				if winnerId.Valid && int(winnerId.Int64) == awayTeamId {
					stats.awayWins++
				}
			} else {
				// Away team is playing away
				stats.awayGoalsFor += awayGoals
				stats.awayGoalsAgainst += homeGoals

				if homeGoals == 0 {
					stats.awayCleanSheets++
				} else if homeGoals == 1 {
					stats.away1GoalConceded++
				} else if homeGoals >= 2 {
					stats.away2PlusGoalsConceded++
				}

				if winnerId.Valid && int(winnerId.Int64) == awayTeamId {
					stats.awayWins++
				}
			}
		}

		// Calculate averages
		if stats.homeGamesPlayed > 0 {
			stats.homeAvgGoalsFor = float64(stats.homeGoalsFor) / float64(stats.homeGamesPlayed)
			stats.homeAvgGoalsAgainst = float64(stats.homeGoalsAgainst) / float64(stats.homeGamesPlayed)
		}
		if stats.awayGamesPlayed > 0 {
			stats.awayAvgGoalsFor = float64(stats.awayGoalsFor) / float64(stats.awayGamesPlayed)
			stats.awayAvgGoalsAgainst = float64(stats.awayGoalsAgainst) / float64(stats.awayGamesPlayed)
		}

		return StatsLoaded(stats)
	}
}
