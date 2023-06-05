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

type TeamID int

const (
	Unknown TeamID = 0
	Team1   TeamID = 1
	Team2   TeamID = 2
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

func count(slice []TeamID, team TeamID) int {
	cnt := 0

	for _, val := range slice {
		if val == team {
			cnt++
		}
	}

	return cnt
}

type Country string
type PlayerName string

type Player struct {
	Country Country    `json:"country"`
	Player  PlayerName `json:"player"`
}

type Team []Player

type MatchInfo struct {
	Mode  Mode     `json:"mode"`
	Team1 Team     `json:"team1"`
	Team2 Team     `json:"team2"`
	Start UnixTime `json:"start"`
	End   UnixTime `json:"end"`
}

type Game struct {
	Points          []TeamID `json:"points"`
	Winner          TeamID   `json:"-"`
	PointsPlayed    int      `json:"-"`
	Team1PointsWon  int      `json:"-"`
	Team1ConsPoints int      `json:"-"`
	Team1GamePoints int      `json:"-"`
	Team2PointsWon  int      `json:"-"`
	Team2ConsPoints int      `json:"-"`
	Team2GamePoints int      `json:"-"`
}

type Match struct {
	Info            MatchInfo `json:"info"`
	Games           []Game    `json:"games"`
	Winner          TeamID    `json:"-"`
	Duration        int       `json:"-"`
	PointsPlayed    int       `json:"-"`
	Team1PointsWon  int       `json:"-"`
	Team1ConsPoints int       `json:"-"`
	Team1GamePoints int       `json:"-"`
	Team2PointsWon  int       `json:"-"`
	Team2ConsPoints int       `json:"-"`
	Team2GamePoints int       `json:"-"`
}

func (c Country) isValid() bool {
	return countries.ByName(string(c)) != countries.Unknown
}

func (n PlayerName) isValid() bool {
	return len(string(n)) != 0
}

func (p Player) isValid() bool {
	return p.Country.isValid() && p.Player.isValid()
}

func (m Mode) isValid() bool {
	return m == Mode11 || m == Mode21
}

func (t Team) isValid() bool {
	for _, player := range t {
		if !player.isValid() {
			return false
		}
	}

	return true
}

func (m MatchInfo) validate() error {
	if !m.Mode.isValid() {
		return errors.New(ERR_INVALID_MODE)
	}

	if !m.Team1.isValid() || !m.Team2.isValid() {
		return errors.New(ERR_INVALID_TEAMS)
	}

	if len(m.Team1) != len(m.Team2) || (len(m.Team1) != 1 && len(m.Team1) != 2) {
		return errors.New(ERR_INVALID_TEAMS)
	}

	if m.Start.IsZero() {
		return errors.New(ERR_INVALID_TIMES)
	}

	// m.End is allowed to be zero for running matches
	if !m.End.IsZero() && !m.Start.Time.Before(m.End.Time) {
		return errors.New(ERR_INVALID_TIMES)
	}

	return nil
}

func (g *Game) validate(mode Mode, endTime UnixTime) error {
	winPoints := numWinPoints(mode)
	tiePoints := numTiePoints(mode)

	scoreTeam1 := 0
	scoreTeam2 := 0

	for j, point := range g.Points {
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
			g.Winner = Team1

			// check if points were counted afterwards
			if j != len(g.Points)-1 {
				return errors.New(ERR_INVALID_POINT)
			}

			break
		}

		// check if Team2 has won
		if scoreTeam2 == tiePoints || (scoreTeam2 >= winPoints && scoreTeam2-scoreTeam1 >= 2) {
			g.Winner = Team2

			// check if points were counted afterwards
			if j != len(g.Points)-1 {
				return errors.New(ERR_INVALID_POINT)
			}

			break
		}
	}

	g.Team1PointsWon = scoreTeam1
	g.Team2PointsWon = scoreTeam2

	g.PointsPlayed = scoreTeam1 + scoreTeam2

	g.Team1ConsPoints = calculateConsecutivePointsInGame(g.Points, Team1)
	g.Team2ConsPoints = calculateConsecutivePointsInGame(g.Points, Team2)

	g.Team1GamePoints = calculateGamePointsInGame(g.Points, Team1, mode)
	g.Team2GamePoints = calculateGamePointsInGame(g.Points, Team2, mode)

	return nil
}

func validateGames(match *Match, mode Mode, endTime UnixTime) error {
	winGames := numWinGames(mode)
	maxGames := numMaxGames(mode)

	if len(match.Games) > maxGames {
		return errors.New(ERR_INVALID_GAME)
	}

	// Validate and calculate statistics for each game
	for i := range match.Games {
		if err := (&match.Games[i]).validate(mode, endTime); err != nil {
			return err
		}
	}

	var winner []TeamID

	for i, game := range match.Games {
		winner = append(winner, game.Winner)

		// If there is no winner for this game yet, then the game is still running and no later game must exist
		if len(winner) < i+1 && len(match.Games) > i+1 {
			return errors.New(ERR_INVALID_GAME)
		}

		numWinsTeam1 := count(winner, Team1)
		numWinsTeam2 := count(winner, Team2)

		// If there is a winner for this match, then no later games must exist.
		// Also, the end time must be set.
		if (numWinsTeam1 == winGames && numWinsTeam2 < winGames || numWinsTeam2 == winGames && numWinsTeam1 < winGames) &&
			(len(match.Games) > i+1 || endTime.IsZero()) {
			return errors.New(ERR_INVALID_GAME)
		}
	}

	return nil
}

func (m *Match) validate() error {
	if err := m.Info.validate(); err != nil {
		return err
	}

	if err := validateGames(m, m.Info.Mode, m.Info.End); err != nil {
		return err
	}

	if m.Info.End.IsZero() {
		m.Info.End.Time = time.Now()
	}

	m.Duration = int(m.Info.End.Sub(m.Info.Start.Time).Round(time.Minute).Minutes())
	if m.Duration < 0 {
		// It is possible that the Start date is set to some future date by the client
		// and the end date set to time.Now() by the parser. This results in a negative duration.
		m.Duration = 0
	}

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

	return nil
}

// Calculates the maximum of consecutive points the given team has
// scored, before the opponent scored again.
func calculateConsecutivePointsInGame(points []TeamID, team TeamID) int {
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
func calculateGamePointsInGame(points []TeamID, team TeamID, mode Mode) int {
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

func calculatePointsWonInMatch(games []Game, team TeamID) int {
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

func calculateConsecutivePointsInMatch(games []Game, team TeamID) int {
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

func calculateGamePointsInMatch(games []Game, team TeamID) int {
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

func calculateGamesWonInMatch(games []Game, team TeamID) int {
	winners := []TeamID{}

	for _, game := range games {
		winners = append(winners, game.Winner)
	}

	return count(winners, team)
}

func Parse(data string) (Match, error) {
	var match Match

	if err := json.Unmarshal([]byte(data), &match); err != nil {
		return match, fmt.Errorf("%s %s", ERR_INVALID_JSON, err.Error())
	}

	if err := match.validate(); err != nil {
		return match, fmt.Errorf("%s: %s", ERR_INVALID_MATCH, err.Error())
	}

	return match, nil
}
