package game

import (
	"errors"
	"fmt"

	codenames "github.com/bcspragu/Codenames"
)

// Game represents a game of codenames.
type Game struct {
	b          *codenames.Board
	cfg        *Config
	activeTeam codenames.Team
}

// Config holds configuration options for a game of Codenames.
type Config struct {
	// Team is which team the AI is on.
	Team codenames.Team
	// Role is this computer's role in the game.
	Role codenames.Role
	// Starter is the team that goese first
	Starter codenames.Team

	SpymasterAI codenames.SpymasterAI
	OperativeAI codenames.OperativeAI
}

// New validates and initializes a game of Codenames.
func New(b *codenames.Board, cfg *Config) (*Game, error) {
	if err := validateBoard(b); err != nil {
		return nil, fmt.Errorf("invalid board given: %v", err)
	}

	if cfg.Team == codenames.NoTeam {
		return nil, errors.New("team must be set")
	}

	if cfg.Role == codenames.Spymaster && cfg.SpymasterAI == nil {
		return nil, errors.New("spymaster AI must be specified when playing as Spymaster")
	}

	if cfg.Role == codenames.Operative && cfg.OperativeAI == nil {
		return nil, errors.New("operative AI must be specified when playing as an Operative")
	}

	if cfg.Starter == codenames.NoTeam {
		return nil, errors.New("starter team must be set")
	}

	if err := validateForRole(b, cfg.Role); err != nil {
		return nil, fmt.Errorf("invalid %q board: %v", cfg.Role, err)
	}

	return &Game{
		b:          b,
		cfg:        cfg,
		activeTeam: cfg.Starter,
	}, nil
}

func validateBoard(b *codenames.Board) error {
	if len(b.Cards) != codenames.Size {
		return fmt.Errorf("board must contain %d codenames, found %d", codenames.Size, len(b.Cards))
	}
	return nil
}

// Guess takes a guess at a clue, given what is known about the board.
func (g *Game) Guess(c *codenames.Clue) (string, error) {
	if g.cfg.Role != codenames.Operative {
		return "", errors.New("game isn't playing as Operative, it doesn't know how guess")
	}
	return g.cfg.OperativeAI.Guess(g.b, c)
}

// GiveClue generates a clue for the board.
func (g *Game) GiveClue() (*codenames.Clue, error) {
	if g.cfg.Role != codenames.Spymaster {
		return nil, errors.New("game isn't playing as Spymaster, it doesn't know how to give clues")
	}
	return g.cfg.SpymasterAI.GiveClue(g.b)
}

func validateForRole(b *codenames.Board, role codenames.Role) error {
	if role == codenames.NoRole {
		return errors.New("role must be set")
	}

	var red, blue, assassin bool
	for _, cn := range b.Cards {
		switch cn.Agent {
		case codenames.RedAgent:
			red = true
		case codenames.BlueAgent:
			blue = true
		case codenames.Assassin:
			assassin = true
		}
	}

	// We want red/blue/assassin to be set if	we're the Spymaster
	want := role == codenames.Spymaster

	if red != want {
		return errors.New("missing red team codenames")
	}

	if blue != want {
		return errors.New("missing blue team codenames")
	}

	if assassin != want {
		return errors.New("missing assassin")
	}
	return nil
}
