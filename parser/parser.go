package parser

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	ERR_INVALID_JSON  = "JSON is invalid."
	ERR_INVALID_MATCH = "Match is invalid."
)

func max(a int, b int) int {
	if a >= b {
		return a
	} else {
		return b
	}
}

type Team int

const (
	Unknown Team = 0
	Team1   Team = 1
	Team2   Team = 2
)

type Mode int

const (
	Mode11 Mode = 11
	Mode21 Mode = 21
)

const (
	Mode11WinPoints  = 11
	Mode11TiePoints  = 15
	Mode11WinSets    = 3
	Mode11WinMaxSets = 5

	Mode21WinPoints  = 21
	Mode21TiePoints  = 30
	Mode21WinSets    = 2
	Mode21WinMaxSets = 3
)

func numWinPoints(mode Mode) int {
	switch mode {
	case Mode11:
		return Mode11WinPoints
	case Mode21:
		fallthrough
	default:
		return Mode21WinPoints
	}
}

func numTiePoints(mode Mode) int {
	switch mode {
	case Mode11:
		return Mode11TiePoints
	case Mode21:
		fallthrough
	default:
		return Mode21TiePoints
	}
}

type Player struct {
	Country string `json:"country"`
	Player  string `json:"player"`
}

type MatchInfo struct {
	Mode  Mode     `json:"mode"`
	Team1 []Player `json:"team1"`
	Team2 []Player `json:"team2"`
	Start UnixTime `json:"start"`
	End   UnixTime `json:"end"`
}

type Game struct {
	Points          []Team `json:"points"`
	Winner          Team   `json:"-"`
	PointsPlayed    int    `json:"-"`
	Team1PointsWon  int    `json:"-"`
	Team1ConsPoints int    `json:"-"`
	Team1GamePoints int    `json:"-"`
	Team2PointsWon  int    `json:"-"`
	Team2ConsPoints int    `json:"-"`
	Team2GamePoints int    `json:"-"`
}

type Match struct {
	Info            MatchInfo `json:"info"`
	Games           []Game    `json:"games"`
	Winner          Team      `json:"-"`
	Duration        int       `json:"-"`
	PointsPlayed    int       `json:"-"`
	Team1PointsWon  int       `json:"-"`
	Team1ConsPoints int       `json:"-"`
	Team1GamePoints int       `json:"-"`
	Team2PointsWon  int       `json:"-"`
	Team2ConsPoints int       `json:"-"`
	Team2GamePoints int       `json:"-"`
}

func (match *Match) validate() error {
	//   Validate Mode
	//   Validate Country
	//   Validate Name not empty
	//   Validate Points

	return nil
}

func calculatePointsInGame(points []Team, team Team) int {
	count := 0

	for _, s := range points {
		if s == team {
			count++
		}
	}

	return count
}

// Calculates the maximum of consecutive points the given team has
// scored, before the opponent scored again.
func calculateConsecutivePointsInGame(points []Team, team Team) int {
	maxConsPoints := 0
	curConsPoints := 0

	for _, s := range points {
		if s == team {
			curConsPoints++
		} else {
			maxConsPoints = max(maxConsPoints, curConsPoints)
			curConsPoints = 0
		}
	}

	return maxConsPoints
}

// Calculates the number of game points for the given team.
// A game point is a situation in which a team only needs one
// more point to win.
// A has 2 game points and B 1 game point in the following
// example of a game to 21 points:
//
//	A vs. B
//	  ...
//	19 : 19
//	20 : 19  <- Game point for A
//	20 : 20
//	20 : 21  <- Game point for B
//	21 : 21
//	22 : 21  <- Game point for A
//	23 : 21  <- A wins
func calculateGamePointsInGame(points []Team, team Team, mode Mode) int {
	gamePoints := 0

	ownScore := 0
	otherScore := 0

	winPoints := numWinPoints(mode)
	tiePoints := numTiePoints(mode)

	for _, point := range points {
		if point == team {
			ownScore++
		} else {
			otherScore++
		}

		if (ownScore == winPoints-1 && otherScore < ownScore) ||
			(ownScore == tiePoints-1 && otherScore < tiePoints) {
			gamePoints++
		}
	}

	return gamePoints
}

func calculatePointsPlayedInMatch(games []Game) int {
	sum := 0

	for _, game := range games {
		sum += game.PointsPlayed
	}

	return sum
}

func calculatePointsWonInMatch(games []Game, team Team) int {
	sum := 0

	for _, game := range games {
		if team == Team1 {
			sum += game.Team1PointsWon
		} else {
			sum += game.Team2PointsWon
		}
	}

	return sum
}

func calculateConsecutivePointsInMatch(games []Game, team Team) int {
	maxCons := 0

	for _, game := range games {
		if team == Team1 {
			maxCons = max(maxCons, game.Team1ConsPoints)
		} else {
			maxCons = max(maxCons, game.Team2ConsPoints)
		}
	}

	return maxCons
}

func calculateGamePointsInMatch(games []Game, team Team) int {
	sum := 0

	for _, game := range games {
		if team == Team1 {
			sum += game.Team1GamePoints
		} else {
			sum += game.Team2GamePoints
		}
	}

	return sum
}

func calculateGamesWonInMatch(games []Game, team Team) int {
	won := 0

	for _, game := range games {
		if game.Winner == team {
			won++
		}
	}

	return won
}

// Calculates game/match winners and statistics such as consecutive points won,
// game points played etc.
// This function assumes that the match is valid, this should be checked using
// validate() before.
func (m *Match) determineStats() {
	// Calculate all game statistics first
	for i, game := range m.Games {
		m.Games[i].Team1PointsWon = calculatePointsInGame(game.Points, Team1)
		m.Games[i].Team1ConsPoints = calculateConsecutivePointsInGame(game.Points, Team1)
		m.Games[i].Team1GamePoints = calculateGamePointsInGame(game.Points, Team1, m.Info.Mode)

		m.Games[i].Team2PointsWon = calculatePointsInGame(game.Points, Team2)
		m.Games[i].Team2ConsPoints = calculateConsecutivePointsInGame(game.Points, Team2)
		m.Games[i].Team2GamePoints = calculateGamePointsInGame(game.Points, Team2, m.Info.Mode)

		m.Games[i].PointsPlayed = m.Games[i].Team1PointsWon + m.Games[i].Team2PointsWon

		if m.Games[i].Team1PointsWon > m.Games[i].Team2PointsWon {
			m.Games[i].Winner = Team1
		} else {
			m.Games[i].Winner = Team2
		}
	}

	// Calculate match statistics based on game statistics
	m.Duration = int(m.Info.End.Time.Sub(m.Info.Start.Time).Round(time.Minute).Minutes())
	m.PointsPlayed = calculatePointsPlayedInMatch(m.Games)
	m.Team1PointsWon = calculatePointsWonInMatch(m.Games, Team1)
	m.Team1ConsPoints = calculateConsecutivePointsInMatch(m.Games, Team1)
	m.Team1GamePoints = calculateGamePointsInMatch(m.Games, Team1)
	m.Team2PointsWon = calculatePointsWonInMatch(m.Games, Team2)
	m.Team2ConsPoints = calculateConsecutivePointsInMatch(m.Games, Team2)
	m.Team2GamePoints = calculateGamePointsInMatch(m.Games, Team2)

	if calculateGamesWonInMatch(m.Games, Team1) > calculateGamesWonInMatch(m.Games, Team2) {
		m.Winner = Team1
	} else {
		m.Winner = Team2
	}
}

func Parse(data string, validate bool) (Match, error) {
	var match Match

	if err := json.Unmarshal([]byte(data), &match); err != nil {
		return match, fmt.Errorf("%s %s", ERR_INVALID_JSON, err.Error())
	}

	if validate {
		if err := match.validate(); err != nil {
			return match, fmt.Errorf("%s %s", ERR_INVALID_MATCH, err.Error())
		}
	}

	match.determineStats()

	return match, nil
}
