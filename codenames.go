package codenames

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// Rows is the number of rows of cards in Codenames.
	Rows = 5
	// Columns is the number of columns of cards in Codenames.
	Columns = 5
	// Size is the total number of cards on a Codenames board.
	Size = Rows * Columns
)

type Spymaster interface {
	// GiveClue takes in a board and returns a clue for players to guess with.
	GiveClue(*Board) (*Clue, error)
}

type Operative interface {
	// Guess takes in a board and a clue and returns the guessed Codename from
	// the board.
	Guess(*Board, *Clue) (string, error)
}

// Board contains all of the information about a game of Codenames.
type Board struct {
	// Cards is a list of the 25 words on the board. The zeroth card corresponds
	// to the top-left, the fourth to the top-right, and the twenty-fourth to the
	// bottom-right.
	Cards []Card
}

// Clue is a word and a count from the Spymaster.
type Clue struct {
	Word  string
	Count int
}

func (c *Clue) String() string {
	return c.Word + " " + strconv.Itoa(c.Count)
}

// Codename is a single game card, and its corresponding affiliation.
type Card struct {
	Codename string
	// Agent is the card
	Agent Agent
}

// Agent is the affiliation of a codename.
type Agent int

func (a Agent) String() string {
	switch a {
	case UnknownAgent:
		return "Agent Status Unknown"
	case RedAgent:
		return "Red Agent"
	case BlueAgent:
		return "Blue Agent"
	case Bystander:
		return "Bystander"
	case Assassin:
		return "Assassin"
	}
	return ""
}

const (
	// UnknownAgent means we don't know who the codename belongs to.
	UnknownAgent Agent = iota
	// RedAgent means the codename belongs to an agent on the red team.
	RedAgent
	// BlueAgent means the codename belongs to an agent on the blue team.
	BlueAgent
	// Bystander means the codename doesn't belong to an agent.
	Bystander
	// Assassin means the codename belongs to the assassin.
	Assassin
)

type Team int

func (t Team) String() string {
	switch t {
	case RedTeam:
		return "Red Team"
	case BlueTeam:
		return "Blue Team"
	}
	return ""
}

const (
	// NoTeam is an error case.
	NoTeam Team = iota
	RedTeam
	BlueTeam
)

// Unused returns a list of cards that haven't been assigned an Agent type yet.
func Unused(cards []Card) []Card {
	return Targets(cards, UnknownAgent)
}

// Unused returns a list of cards that haven't been assigned an Agent type yet.
func Targets(cards []Card, agent Agent) []Card {
	var out []Card
	for _, card := range cards {
		if card.Agent == agent {
			out = append(out, card)
		}
	}
	return out
}

func ParseClue(clue string) (*Clue, error) {
	ps := strings.Split(clue, " ")
	if len(ps) != 2 {
		return nil, fmt.Errorf("malformed clue %q", clue)
	}
	i, err := strconv.Atoi(ps[1])
	if err != nil {
		return nil, fmt.Errorf("malformed number of words in clue %q: %v", clue, err)
	}

	return &Clue{Word: ps[0], Count: i}, nil
}
