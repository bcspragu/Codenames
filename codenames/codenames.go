package codenames

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bcspragu/Codenames/types"
	"github.com/bcspragu/Codenames/w2v"
)

// Game represents a game of codenames.
type Game struct {
	b   *types.Board
	cfg *Config
}

// Config holds configuration options for a game of Codenames.
type Config struct {
	// Team is which team the compute is on.
	Team types.Team
	// Role is this computer's role in the game.
	Role types.Role
}

// NewGame validates and initializes a game of Codenames.
func NewGame(b *types.Board, cfg *Config) (*Game, error) {
	if err := validateBoard(b); err != nil {
		return nil, fmt.Errorf("invalid board given: %v", err)
	}

	switch cfg.Team {
	case types.NoTeam:
		return nil, errors.New("cfg.Team cannot be 'NoTeam'")
	case types.Assassin:
		return nil, errors.New("cfg.Team cannot be 'Assassin'")
	}

	switch cfg.Role {
	case types.NoRole:
		return nil, errors.New("Config.Role cannot be 'NoRole'")
	case types.Spymaster:
		if err := validateForSpymaster(b); err != nil {
			return nil, fmt.Errorf("invalid Spymaster board: %v", err)
		}
	case types.Operative:
		if err := validateForOperative(b); err != nil {
			return nil, fmt.Errorf("invalid Operative board: %v", err)
		}
	}

	return &Game{
		b:   b,
		cfg: cfg,
	}, nil
}

func validateBoard(b *types.Board) error {
	if len(b.Codenames) != types.Size {
		return fmt.Errorf("board must contain %d codenames, found %d", types.Size, len(b.Codenames))
	}
	return nil
}

func (g *Game) availNames() []types.Codename {
	var cns []types.Codename
	for _, cn := range g.b.Codenames {
		if cn.Team == types.NoTeam {
			cns = append(cns, cn)
		}
	}
	return cns
}

// Guess takes a guess at a clue, given what is known about the board.
func (g *Game) Guess(clue string) ([]string, error) {
	word, n := parseClue(clue)
	if n == 0 || word == "" {
		return nil, fmt.Errorf("failed to parse clue %q", clue)
	}

	availNames := g.availNames()

	if n > len(availNames) {
		return nil, fmt.Errorf("gave a clue for %d names...which is too many. %d cards available", n, len(availNames))
	}

	pairs := make([]struct {
		Word       string
		Similarity float32
	}, len(availNames))

	for i, cn := range availNames {
		sim, err := w2v.Similarity(word, cn.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get similarity of %q and %q: %v", word, cn.Name, err)
		}
		pairs[i].Word = cn.Name
		pairs[i].Similarity = sim
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Similarity > pairs[j].Similarity
	})

	var guesses []string
	for i := 0; i < n; i++ {
		guesses = append(guesses, pairs[i].Word)
	}
	return guesses, nil
}

func (g *Game) Assign(name string, team types.Team) {
	for i, cn := range g.b.Codenames {
		if cn.Name == name {
			g.b.Codenames[i].Team = team
			return
		}
	}
}

func parseClue(clue string) (string, int) {
	ps := strings.Split(clue, " ")
	if len(ps) != 2 {
		return "", 0
	}
	i, err := strconv.Atoi(ps[1])
	if err != nil {
		return "", 0
	}

	return ps[0], i
}

func validateForSpymaster(b *types.Board) error {
	var red, blue, assassin bool
	for _, cn := range b.Codenames {
		switch cn.Team {
		case types.RedTeam:
			red = true
		case types.BlueTeam:
			blue = true
		case types.Assassin:
			assassin = true
		}
	}
	if !red {
		return errors.New("missing red team codenames")
	}

	if !blue {
		return errors.New("missing blue team codenames")
	}

	if !assassin {
		return errors.New("missing assassin")
	}
	return nil
}

func validateForOperative(b *types.Board) error {
	var red, blue, assassin bool
	for _, cn := range b.Codenames {
		switch cn.Team {
		case types.RedTeam:
			red = true
		case types.BlueTeam:
			blue = true
		case types.Assassin:
			assassin = true
		}
	}
	if red {
		return errors.New("board should have no red team codenames")
	}

	if blue {
		return errors.New("board should have no blue team codenames")
	}

	if assassin {
		return errors.New("board should have no assassin")
	}
	return nil
}
