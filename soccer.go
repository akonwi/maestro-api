package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const api_key string = "IaF7Bu1ERddp13pUOg0l-vfc95MvbWMFUAohi_yk840"
const england string = "ENG"
const premier_league int = 9
const man_united string = "19538871"

type match struct {
	MatchId      string `json:"match_id"`
	LeagueId     int    `json:"league_id"`
	Result       string `json:"result"`
	GoalsFor     int    `json:"gf"`
	GoalsAgainst int    `json:"ga"`
}

type teamResponse struct {
	TeamSchedule struct {
		Data []match `json:"data"`
	} `json:"team_schedule"`
}

type snapshot struct {
	NumGames              int     `json:"numGames"`
	GoalsAgainst          int     `json:"goalsAgainst"`
	AvgGoalsAgainst       float32 `json:"averageGoalsAgainst"`
	CleanSheetRatio       float32 `json:"cleanSheetRatio"`
	DirtySheetRatio       float32 `json:"dirtySheetRatio"`
	TwoGoalsConcededRatio float32 `json:"twoGoalsConceededRatio"`
	XOverTwo              float32 `json:"x1.5+GA"`

	CleanSheets struct {
		Total int `json:"total"`
		Wins  int `json:"wins"`
		Draws int `json:"draws"`
	} `json:"cleanSheets"`
	OneGoalConceded struct {
		Total  int `json:"total"`
		Wins   int `json:"wins"`
		Draws  int `json:"draws"`
		Losses int `json:"losses"`
	} `json:"oneGoalConceded"`
	TwoPlusGoalsConceded struct {
		Total  int `json:"total"`
		Wins   int `json:"wins"`
		Draws  int `json:"draws"`
		Losses int `json:"losses"`
	} `json:"twoPlusGoalsConceded"`
}

func main() {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://fbrapi.com/teams?team_id=19538871", nil)
	if err != nil {
		panic("Error creating request: " + err.Error())
	}
	req.Header.Add("X-API-Key", api_key)
	req.Header.Add("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		panic("Error fetching team: " + err.Error())
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic("Error reading response body: " + err.Error())
	}

	team := teamResponse{}
	json.Unmarshal(body, &team)

	snap := snapshot{}
	for _, m := range team.TeamSchedule.Data {
		if m.LeagueId == premier_league && m.MatchId != "" {
			snap.NumGames++
			snap.GoalsAgainst += m.GoalsAgainst

			switch m.GoalsAgainst {
			case 0:
				snap.CleanSheets.Total++
				if m.Result == "W" {
					snap.CleanSheets.Wins++
				} else {
					snap.CleanSheets.Draws++
				}

			case 1:
				snap.OneGoalConceded.Total++
				switch m.Result {
				case "W":
					snap.OneGoalConceded.Wins++
				case "L":
					snap.OneGoalConceded.Losses++
				case "D":
					snap.OneGoalConceded.Draws++
				}

			default:
				// todo: what's the goal difference in these matches?
				snap.TwoPlusGoalsConceded.Total++
				switch m.Result {
				case "W":
					snap.TwoPlusGoalsConceded.Wins++
				case "L":
					snap.TwoPlusGoalsConceded.Losses++
				case "D":
					snap.TwoPlusGoalsConceded.Draws++
				}
			}
		}
	}

	snap.AvgGoalsAgainst = float32(snap.GoalsAgainst) / float32(snap.NumGames)
	snap.CleanSheetRatio = float32(snap.CleanSheets.Total) / float32(snap.NumGames)
	snap.DirtySheetRatio = 1 - snap.CleanSheetRatio
	snap.TwoGoalsConcededRatio = float32(snap.TwoPlusGoalsConceded.Total) / float32(snap.NumGames)
	// DirtySheetRatio * (TwoPlusGoalsConceded.Total / (OneGoalConceded.Total + TwoPlusGoalsConceded.Total))
	snap.XOverTwo = snap.DirtySheetRatio * (float32(snap.TwoPlusGoalsConceded.Total) / float32(snap.OneGoalConceded.Total+snap.TwoPlusGoalsConceded.Total))
	pretty, err := json.MarshalIndent(snap, "", "  ")
	fmt.Printf("%s\n", pretty)
}
