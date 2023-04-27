package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/biter777/countries"
)

const (
	ERR_INVALID_JSON    = "JSON is invalid"
	ERR_INVALID_MATCH   = "match is invalid"
	ERR_INVALID_MODE    = "mode is invalid"
	ERR_INVALID_COUNTRY = "country is invalid"
	ERR_INVALID_NAME    = "name is invalid"
	ERR_INVALID_TEAMS   = "teams are invalid"
	ERR_INVALID_TIMES   = "times are invalid"
	ERR_INVALID_POINT   = "point is invalid"
	ERR_INVALID_GAME    = "game is invalid"
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
	Mode11WinPoints = 11
	Mode11TiePoints = 15
	Mode11WinGames  = 3
	Mode11MaxGames  = 5

	Mode21WinPoints = 21
	Mode21TiePoints = 30
	Mode21WinGames  = 2
	Mode21MaxGames  = 3
)

type Country string

const (
	CountryDE = "DE"
	CountryDK = "DK"
	CountryTW = "TW"
)

func numWinPoints(mode Mode) int {
	switch mode {
	case Mode11:
		return Mode11WinPoints
	case Mode21:
		return Mode21WinPoints
	default:
		return -1
	}
}

func numTiePoints(mode Mode) int {
	switch mode {
	case Mode11:
		return Mode11TiePoints
	case Mode21:
		return Mode21TiePoints
	default:
		return -1
	}
}

func numWinGames(mode Mode) int {
	switch mode {
	case Mode11:
		return Mode11WinGames
	case Mode21:
		return Mode21WinGames
	default:
		return -1
	}
}

func numMaxGames(mode Mode) int {
	switch mode {
	case Mode11:
		return Mode11MaxGames
	case Mode21:
		return Mode21MaxGames
	default:
		return -1
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

func isValidMode(mode Mode) bool {
	return mode == Mode11 || mode == Mode21
}

func isValidCountry(country string) bool {
	return countries.ByName(country) != countries.Unknown
}

func isValidName(name string) bool {
	return len(name) != 0
}

func count(slice []Team, team Team) int {
	cnt := 0

	for _, val := range slice {
		if val == team {
			cnt++
		}
	}

	return cnt
}

func validateGames(games []Game, mode Mode) error {
	winPoints := numWinPoints(mode)
	tiePoints := numTiePoints(mode)
	winGames := numWinGames(mode)
	maxGames := numMaxGames(mode)

	var winner []Team

	if len(games) > maxGames {
		return errors.New(ERR_INVALID_GAME)
	}

	for i, game := range games {
		scoreTeam1 := 0
		scoreTeam2 := 0

		for j, point := range game.Points {
			// check if point was given to either Team1 or Team2
			if !(point == Team1 || point == Team2) {
				return errors.New(ERR_INVALID_POINT)
			}

			if point == Team1 {
				scoreTeam1++
			} else if point == Team2 {
				scoreTeam2++
			}

			// check if Team1 has won
			if scoreTeam1 == tiePoints || (scoreTeam1 >= winPoints && scoreTeam1-scoreTeam2 >= 2) {
				winner = append(winner, Team1)

				// check if points were counted afterwards
				if j != len(game.Points)-1 {
					return errors.New(ERR_INVALID_POINT)
				}

				break
			}

			// check if Team2 has won
			if scoreTeam2 == tiePoints || (scoreTeam2 >= winPoints && scoreTeam2-scoreTeam1 >= 2) {
				winner = append(winner, Team1)

				// check if points were counted afterwards
				if j != len(game.Points)-1 {
					return errors.New(ERR_INVALID_POINT)
				}

				break
			}
		}

		// If there is no winner for this game yet, then the game is still running and no later game must exist
		if len(winner) < i+1 && len(games) > i+1 {
			return errors.New(ERR_INVALID_GAME)
		}

		numWinsTeam1 := count(winner, Team1)
		numWinsTeam2 := count(winner, Team2)

		// If there is a winner for this match, then no later games must exist
		if (numWinsTeam1 == winGames && numWinsTeam2 < winGames || numWinsTeam2 == winGames && numWinsTeam1 < winGames) &&
			(len(games) > i+1) {
			return errors.New(ERR_INVALID_GAME)
		}
	}

	return nil
}

func (match *Match) validate() error {
	if !isValidMode(match.Info.Mode) {
		return errors.New(ERR_INVALID_MODE)
	}

	for _, player := range match.Info.Team1 {
		if !isValidCountry(player.Country) {
			return errors.New(ERR_INVALID_COUNTRY)
		}

		if !isValidName(player.Player) {
			return errors.New(ERR_INVALID_NAME)
		}
	}

	for _, player := range match.Info.Team2 {
		if !isValidCountry(player.Country) {
			return errors.New(ERR_INVALID_COUNTRY)
		}

		if !isValidName(player.Player) {
			return errors.New(ERR_INVALID_NAME)
		}
	}

	if !(len(match.Info.Team1) == len(match.Info.Team2) &&
		(len(match.Info.Team1) == 1 || len(match.Info.Team1) == 2)) {
		return errors.New(ERR_INVALID_TEAMS)
	}

	if !match.Info.Start.Time.Before(match.Info.End.Time) {
		return errors.New(ERR_INVALID_TIMES)
	}

	if err := validateGames(match.Games, match.Info.Mode); err != nil {
		return err
	}

	return nil
}

func calculatePointsInGame(points []Team, team Team) int {
	return count(points, team)
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
		} else if team == Team2 {
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
		} else if team == Team2 {
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
		} else if team == Team2 {
			sum += game.Team2GamePoints
		}
	}

	return sum
}

func calculateGamesWonInMatch(games []Game, team Team) int {
	winners := []Team{}

	for _, game := range games {
		winners = append(winners, game.Winner)
	}

	return count(winners, team)
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

		scoreTeam1 := m.Games[i].Team1PointsWon
		scoreTeam2 := m.Games[i].Team2PointsWon

		if scoreTeam1 == numTiePoints(m.Info.Mode) || (scoreTeam1 >= numWinPoints(m.Info.Mode) && scoreTeam1-scoreTeam2 >= 2) {
			m.Games[i].Winner = Team1
		} else if scoreTeam2 == numTiePoints(m.Info.Mode) || (scoreTeam2 >= numWinPoints(m.Info.Mode) && scoreTeam2-scoreTeam1 >= 2) {
			m.Games[i].Winner = Team2
		} else {
			m.Games[i].Winner = Unknown
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

	if calculateGamesWonInMatch(m.Games, Team1) == numWinGames(m.Info.Mode) {
		m.Winner = Team1
	} else if calculateGamesWonInMatch(m.Games, Team2) == numWinGames(m.Info.Mode) {
		m.Winner = Team2
	} else {
		m.Winner = Unknown
	}
}

func Parse(data string, validate bool) (Match, error) {
	var match Match

	if err := json.Unmarshal([]byte(data), &match); err != nil {
		return match, fmt.Errorf("%s %s", ERR_INVALID_JSON, err.Error())
	}

	if validate {
		if err := match.validate(); err != nil {
			return match, fmt.Errorf("%s: %s", ERR_INVALID_MATCH, err.Error())
		}
	}

	match.determineStats()

	return match, nil
}
