# Maestro
This project is a TUI (Terminal UI) for managing and viewing soccer statistics.

# Technology
The technology stack is:
* Go
* Sqlite
* [Bubble Tea](https://github.com/charmbracelet/bubbletea)
* [Bubbles](https://github.com/charmbracelet/bubbles)
* [Lipgloss](https://github.com/charmbracelet/lipgloss)

# Data
The leagues, teams, and matches come from the database and are updated regularly by a separate program

# Product Requirements

* Upon startup, the user selects a league
* Within the league, matches are listed
  * the initial list is upcoming or unfinished matches and it can be toggled to finished matches
* while a match is selected:
  * the stats of both teams are displayed in a portion of the screen
  * the user can enter bets for the match
  * the user can see the bets recorded for the match

## Displayed Stats
In a match overview, some key metrics will be displayed
- # of games
- W-L-D record
- # of goals against
- # of goals for
- Goal difference
- cleansheet ratio
- average goals for
- average goals against
