package main

import codenames "github.com/bcspragu/Codenames"

// Scenario describes a certain setting of a board.
// Only fill in the minimum fields needed for a particular
// scenario; unfilled fields will be handled sensibly depending
// on the benchmark being run.
type Scenario struct {
	Red       []string
	Blue      []string
	Assassin  []string
	Bystander []string
	Clue      codenames.Clue
	Target    []string
}

type Result struct {
	Correct            int
	IncorrectTeam      int
	IncorrectAssassin  int
	IncorrectBystander int
	IncorrectInvalid   int
	Skipped            int
}

var (
	Scenarios = []Scenario{
        {
            Red: []string{"lead"},
            Clue: codenames.Clue{"follow", 1},
            Target: []string{"lead"},
        },
        {
            Red: []string{"maple"},
            Clue: codenames.Clue{"syrup", 1},
            Target: []string{"maple"},
        },
	}
)

// Score calculates a measure for how "good" a result is.
// Higher is better; 1.0 is a perfect score.
func Score(r Result) float32 {
	maxPotentialScore := float32(r.Correct + r.IncorrectTeam + r.IncorrectAssassin + r.IncorrectBystander + r.IncorrectInvalid + r.Skipped)
	points := float32(r.Correct) - float32(r.IncorrectTeam) - 2.0*float32(r.IncorrectAssassin) - 0.5*float32(r.IncorrectBystander) - 100.0*float32(r.IncorrectInvalid)
	return points / maxPotentialScore
}

// OperativeBoard generates a Board from a Scenario.
// This Board can be passed to an Opertive's Guess.
func OperativeBoard(s Scenario) codenames.Board {
	var cards []codenames.Card
	for _, cardSet := range [][]string{s.Red, s.Blue, s.Assassin, s.Bystander} {
		for _, card := range cardSet {
			cards = append(cards, codenames.Card{Codename: card, Agent: codenames.UnknownAgent})
		}
	}
	return codenames.Board{Cards: cards}
}
