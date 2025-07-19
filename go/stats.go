package main

import (
	"database/sql"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type Snapshot struct {
	id     int
	name   string
	wins   int
	losses int
	draws  int

	goalsAgainst int
	goalsFor     int

	cleansheets     int
	oneConceded     int
	twoPlusConceded int
}

func (s Snapshot) avgGoalsFor() float64 {
	if s.totalGames() == 0 {
		return 0
	}
	return float64(s.goalsFor) / float64(s.totalGames())
}

func (s Snapshot) avgGoalsAgainst() float64 {
	if s.totalGames() == 0 {
		return 0
	}
	return float64(s.goalsAgainst) / float64(s.totalGames())
}

func (s Snapshot) goalDiff() int {
	return s.goalsFor - s.goalsAgainst
}

func (s Snapshot) totalGames() int {
	return s.wins + s.losses + s.draws
}

func (s Snapshot) fmtRecord() string {
	return fmt.Sprintf("%d-%d-%d", s.wins, s.draws, s.losses)
}

func (s Snapshot) fmtGoals() string {
	return fmt.Sprintf("%d:%d", s.goalsFor, s.goalsAgainst)
}

type MatchupStats struct {
	away Snapshot
	home Snapshot
}

// Commands
func getStatsForMatchup(match Match) tea.Cmd {
	if match.isNil() {
		return nil
	}

	return func() tea.Msg {
		var stats MatchupStats
		stats.home.name = match.homeTeamName
		stats.home.id = match.homeTeamId
		stats.away.name = match.awayTeamName
		stats.away.id = match.awayTeamId

		// Query all finished matches for home team
		homeRows, err := db.Query(`
			SELECT home_team_id, away_team_id, home_goals, away_goals, winner_id
			FROM matches
			WHERE (home_team_id = ? OR away_team_id = ?) AND status = 'FT'
		`, match.homeTeamId, match.homeTeamId)
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

			if winnerId.Valid {
				if int(winnerId.Int64) == match.homeTeamId {
					stats.home.wins++
				} else {
					stats.home.losses++
				}
			} else {
				stats.home.draws++
			}

			// Home team is playing at home
			if homeId == match.homeTeamId {
				stats.home.goalsFor += homeGoals
				stats.home.goalsAgainst += awayGoals

				if awayGoals == 0 {
					stats.home.cleansheets++
				} else if awayGoals == 1 {
					stats.home.oneConceded++
				} else if awayGoals >= 2 {
					stats.home.twoPlusConceded++
				}
			} else {
				// current home team was away
				stats.home.goalsFor += awayGoals
				stats.home.goalsAgainst += homeGoals

				if homeGoals == 0 {
					stats.home.cleansheets++
				} else if awayGoals == 1 {
					stats.home.oneConceded++
				} else if awayGoals >= 2 {
					stats.home.twoPlusConceded++
				}
			}
		}

		// Query all finished matches for away team
		awayRows, err := db.Query(`
			SELECT home_team_id, away_team_id, home_goals, away_goals, winner_id
			FROM matches
			WHERE (home_team_id = ? OR away_team_id = ?) AND status = 'FT'
		`, match.awayTeamId, match.awayTeamId)
		if err != nil {
			return ErrMsg{err: err}
		}
		defer awayRows.Close()

		// Process as the away team
		for awayRows.Next() {
			var homeId, awayId, homeGoals, awayGoals int
			var winnerId sql.NullInt64
			err := awayRows.Scan(&homeId, &awayId, &homeGoals, &awayGoals, &winnerId)
			if err != nil {
				return ErrMsg{err: err}
			}

			if winnerId.Valid {
				if int(winnerId.Int64) == match.awayTeamId {
					stats.away.wins++
				} else {
					stats.away.losses++
				}
			} else {
				stats.away.draws++
			}

			if homeId == match.awayTeamId {
				stats.away.goalsFor += homeGoals
				stats.away.goalsAgainst += awayGoals

				if awayGoals == 0 {
					stats.away.cleansheets++
				} else if awayGoals == 1 {
					stats.away.oneConceded++
				} else if awayGoals >= 2 {
					stats.away.twoPlusConceded++
				}
			} else {
				// current team was away
				stats.away.goalsFor += awayGoals
				stats.away.goalsAgainst += homeGoals

				if homeGoals == 0 {
					stats.away.cleansheets++
				} else if awayGoals == 1 {
					stats.away.oneConceded++
				} else if awayGoals >= 2 {
					stats.away.twoPlusConceded++
				}
			}
		}

		return StatsLoaded(stats)
	}
}
