package game

import (
	"fmt"
	"log"
	"strings"

	"github.com/bcspragu/Codenames/codenames"
)

// Game represents a game of codenames.
type Game struct {
	groundTruth *codenames.Board
	cfg         *Config
	activeTeam  codenames.Team
}

// Config holds configuration options for a game of Codenames.
type Config struct {
	// Starter is the team that goes first.
	Starter codenames.Team

	RedSpymaster  codenames.Spymaster
	BlueSpymaster codenames.Spymaster

	RedOperative  codenames.Operative
	BlueOperative codenames.Operative
}

// New validates and initializes a game of Codenames.
func New(b *codenames.Board, cfg *Config) (*Game, error) {
	if err := validateBoard(b, cfg.Starter); err != nil {
		return nil, fmt.Errorf("invalid board given: %v", err)
	}

	if cfg.RedSpymaster == nil {
		return nil, fmt.Errorf("RedSpymaster cannot be nil")
	}
	if cfg.BlueSpymaster == nil {
		return nil, fmt.Errorf("BlueSpymaster cannot be nil")
	}
	if cfg.RedOperative == nil {
		return nil, fmt.Errorf("RedOperative cannot be nil")
	}
	if cfg.BlueOperative == nil {
		return nil, fmt.Errorf("BlueOperative cannot be nil")
	}

	return &Game{
		groundTruth: b,
		cfg:         cfg,
		activeTeam:  cfg.Starter,
	}, nil
}

// validateBoard validates that the board has the correct number of cards of
// each type.
func validateBoard(b *codenames.Board, starter codenames.Team) error {
	if len(b.Cards) != codenames.Size {
		return fmt.Errorf("board must contain %d codenames, found %d", codenames.Size, len(b.Cards))
	}

	got := make(map[codenames.Agent]int)
	for _, cn := range b.Cards {
		got[cn.Agent]++
	}

	for ag, wc := range want(starter) {
		if gc := got[ag]; gc != wc {
			return fmt.Errorf("got %d cards of type %q, want %d", gc, ag, wc)
		}
	}

	return nil
}

func want(starter codenames.Team) map[codenames.Agent]int {
	w := map[codenames.Agent]int{
		codenames.RedAgent:  9,
		codenames.BlueAgent: 8,
		codenames.Bystander: 7,
		codenames.Assassin:  1,
	}
	if starter == codenames.BlueTeam {
		w[codenames.BlueAgent], w[codenames.RedAgent] = 9, 8
	}
	return w
}

type Outcome struct {
	Winner codenames.Team
	// TODO: Add more stats, like correct guesses, misses, guesses for the other
	// team, if anyone hit the assassin, etc.
}

func (g *Game) Play() (*Outcome, error) {
	for {
		// Let's play a round.
		sm, op := g.cfg.RedSpymaster, g.cfg.RedOperative
		if g.activeTeam == codenames.BlueTeam {
			sm, op = g.cfg.BlueSpymaster, g.cfg.BlueOperative
		}

		clue, err := sm.GiveClue(codenames.CloneBoard(g.groundTruth))
		if err != nil {
			return nil, fmt.Errorf("GiveClue on %q: %v", g.activeTeam, err)
		}
		numGuesses := clue.Count
		if numGuesses == 0 {
			numGuesses = -1
		}

		for {
			log.Println(numGuesses)
			guess, err := op.Guess(codenames.Revealed(g.groundTruth), clue)
			if err != nil {
				return nil, fmt.Errorf("Guess on %q: %v", g.activeTeam, err)
			}
			numGuesses--

			// TODO: If their guess is totally invalid, give them some sort of
			// recovery mechanism to try again?
			c, err := g.reveal(guess)
			if err != nil {
				return nil, fmt.Errorf("reveal(%q) on %q: %v", guess, g.activeTeam, err)
			}

			// Check if their guess ended the game.
			over, winner := g.gameOver()
			if over {
				return &Outcome{Winner: winner}, nil
			}
			log.Printf("Guess %s was a %s", guess, c.Agent)

			if g.canKeepGuessing(numGuesses, c) {
				continue
			}
			if numGuesses == 0 {
				log.Println("Out of guesses")
			}

			break
		}

		if g.activeTeam == codenames.BlueTeam {
			g.activeTeam = codenames.RedTeam
		} else {
			g.activeTeam = codenames.BlueTeam
		}
	}
}

func (g *Game) reveal(word string) (codenames.Card, error) {
	for i, card := range g.groundTruth.Cards {
		if strings.ToLower(card.Codename) == strings.ToLower(word) {
			// If the card hasn't been reveal, reveal it.
			if !g.groundTruth.Cards[i].Revealed {
				g.groundTruth.Cards[i].Revealed = true
				return card, nil
			}
			return codenames.Card{}, fmt.Errorf("%q has already been guessed", word)
		}
	}
	return codenames.Card{}, fmt.Errorf("no card found for guess %q", word)
}

func (g *Game) canKeepGuessing(numGuesses int, card codenames.Card) bool {
	targetAgent := codenames.RedAgent
	if g.activeTeam == codenames.BlueTeam {
		targetAgent = codenames.BlueAgent
	}

	// They can keep guessing if the card was for their team and they have
	// guesses left.
	return card.Agent == targetAgent && numGuesses != 0
}

func (g *Game) gameOver() (bool, codenames.Team) {
	got := make(map[codenames.Agent]int)
	for i, cn := range g.groundTruth.Cards {
		if g.groundTruth.Cards[i].Revealed {
			got[cn.Agent]++
		}
	}

	for ag, wc := range want(g.cfg.Starter) {
		if gc := got[ag]; gc == wc {
			switch ag {
			case codenames.RedAgent:
				// If we've revealed all the red cards, the red team has won.
				return true, codenames.RedTeam
			case codenames.BlueAgent:
				// If we've revealed all the blue cards, the blue team has won.
				return true, codenames.BlueTeam
			case codenames.Assassin:
				// If we've revealed the assassin, the not-active team wins.
				switch g.activeTeam {
				case codenames.BlueTeam:
					return true, codenames.RedTeam
				case codenames.RedTeam:
					return true, codenames.BlueTeam
				}
			}
		}
	}

	return false, codenames.NoTeam
}
