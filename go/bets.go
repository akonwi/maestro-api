package main

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
