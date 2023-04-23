package parser

import (
	"reflect"
	"testing"
	"time"
)

func assertEqual(t *testing.T, got interface{}, want interface{}) {
	if got != want {
		t.Errorf("Got %v (type %v), want %v (type %v)", got, reflect.TypeOf(got), want, reflect.TypeOf(want))
	}
}

func TestAxelsenVsChou(t *testing.T) {
	match, err := Parse(
		`{
			"info": {
				"mode": 21,
				"team1": [
					{
						"country": "DK",
						"player": "Viktor AXELSEN [1]"
					}
				],
				"team2": [
					{
						"country": "TW",
						"player": "CHOU Tien Chen [3]"
					}
				],
				"start": 1679684400,
				"end": 1679686320
			},
			"games": [
				{
					"points": [1, 2, 1, 2, 2, 1, 2, 2, 2, 2, 1, 2, 2, 2, 2, 2, 1, 2, 1, 2, 2, 1, 1, 2, 2, 2, 2, 2, 1, 1, 2]
				},
				{
					"points": [2, 1, 1, 1, 1, 2, 2, 1, 2, 1, 2, 1, 2, 2, 2, 2, 2, 1, 1, 2, 1, 2, 2, 1, 1, 2, 2, 1, 2, 1, 2, 1, 2, 2, 2, 2]
				}
			]
		}`, true)

	if err != nil {
		t.Fatal(err.Error())
	}

	assertEqual(t, match.Info.Mode, Mode21)
	assertEqual(t, match.Info.Team1[0], Player{Country: "DK", Player: "Viktor AXELSEN [1]"})
	assertEqual(t, match.Info.Team2[0], Player{Country: "TW", Player: "CHOU Tien Chen [3]"})
	assertEqual(t, match.Info.Start, UnixTime{Time: time.Date(2023, time.March, 24, 20, 0, 0, 0, time.Local)})
	assertEqual(t, match.Info.End, UnixTime{Time: time.Date(2023, time.March, 24, 20, 32, 0, 0, time.Local)})

	assertEqual(t, match.Games[0].Winner, Team2)
	assertEqual(t, match.Games[0].PointsPlayed, 31)
	assertEqual(t, match.Games[0].Team1PointsWon, 10)
	assertEqual(t, match.Games[0].Team1ConsPoints, 2)
	assertEqual(t, match.Games[0].Team1GamePoints, 0)
	assertEqual(t, match.Games[0].Team2PointsWon, 21)
	assertEqual(t, match.Games[0].Team2ConsPoints, 5)
	assertEqual(t, match.Games[0].Team2GamePoints, 3)

	assertEqual(t, match.Games[1].Winner, Team2)
	assertEqual(t, match.Games[1].PointsPlayed, 36)
	assertEqual(t, match.Games[1].Team1PointsWon, 15)
	assertEqual(t, match.Games[1].Team1ConsPoints, 4)
	assertEqual(t, match.Games[1].Team1GamePoints, 0)
	assertEqual(t, match.Games[1].Team2PointsWon, 21)
	assertEqual(t, match.Games[1].Team2ConsPoints, 5)
	assertEqual(t, match.Games[1].Team2GamePoints, 1)

	assertEqual(t, match.Winner, Team2)
	assertEqual(t, match.Duration, 32)
	assertEqual(t, match.PointsPlayed, 67)
	assertEqual(t, match.Team1PointsWon, 25)
	assertEqual(t, match.Team1ConsPoints, 4)
	assertEqual(t, match.Team1GamePoints, 0)
	assertEqual(t, match.Team2PointsWon, 42)
	assertEqual(t, match.Team2ConsPoints, 5)
	assertEqual(t, match.Team2GamePoints, 4)
}
