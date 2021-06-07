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
	GiveClue(*Board, Agent) (*Clue, error)
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
	Cards []Card `json:"cards"`
}

func (b *Board) Clone() *Board {
	if b == nil {
		return nil
	}

	cards := make([]Card, len(b.Cards))
	copy(cards, b.Cards)

	return &Board{Cards: cards}
}

// Clue is a word and a count from the Spymaster.
type Clue struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

func (c *Clue) String() string {
	return c.Word + " " + strconv.Itoa(c.Count)
}

// Codename is a single game card, and its corresponding affiliation.
type Card struct {
	// Codename is the word on the card, the "codename" of the agent.
	Codename string `json:"codeword"`
	// Agent is the type of the card, or UnknownAgent if the player doesn't yet
	// know the affiliation.
	Agent Agent `json:"agent"`
	// Revealed is true if the card has been guessed and the identity has been
	// shown to operatives.
	Revealed bool `json:"revealed"`
	// Revealed by is set to the team that chose this card to turnover. This is
	// set to NoTeam unless Revealed is true.
	RevealedBy Team `json:"revealed_by"`
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

type Team string

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
	NoTeam   = Team("")
	RedTeam  = Team("RED")
	BlueTeam = Team("BLUE")
)

func ToTeam(team string) (Team, bool) {
	switch team {
	case "RED":
		return RedTeam, true
	case "BLUE":
		return BlueTeam, true
	default:
		return NoTeam, false
	}
}

// Unused returns a list of cards that haven't been assigned an Agent type yet.
func Unused(cards []Card) []Card {
	return Targets(cards, UnknownAgent)
}

// Targets returns a list of cards that have been assigned to the given Agent
// type.
func Targets(cards []Card, agent Agent) []Card {
	var out []Card
	for _, card := range cards {
		if card.Agent == agent {
			out = append(out, card)
		}
	}
	return out
}

// Revealed takes in a fully-filled out Spymaster board, and returns a new
// board where the card Agent is only populated for revealed cards.
func Revealed(b *Board) *Board {
	out := make([]Card, len(b.Cards))
	copy(out, b.Cards)
	for i, card := range b.Cards {
		if !card.Revealed {
			out[i].Agent = UnknownAgent
		}
	}
	return &Board{Cards: out}
}

// CloneBoard returns a deep copy of the given board.
func CloneBoard(b *Board) *Board {
	out := make([]Card, len(b.Cards))
	for i, card := range b.Cards {
		out[i] = card
	}
	return &Board{Cards: out}
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
