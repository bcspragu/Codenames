package game

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bcspragu/Codenames/codenames"
)

// Game represents a game of codenames. It supports two modes of operation:
// - Play() mode: Plays the whole game out at once. It expects that all of the
// roles in the config have been configured, and everything set up.
// - Move() mode: Plays out a single move, through the Move() function. Will
// reject actions that are requested out of turn, or where callers aren't
// configured for it. This means a *Game can be partially constructed with only
// the information necessary for a given move.
type Game struct {
	state *codenames.GameState
	cfg   *Config
}

// Config holds configuration options for a game of Codenames.
type Config struct {
	RedSpymaster  codenames.Spymaster
	BlueSpymaster codenames.Spymaster

	RedOperative  codenames.Operative
	BlueOperative codenames.Operative
}

func NewForMove(state *codenames.GameState) *Game {
	return &Game{state: state}
}

// New validates and initializes a game of Codenames.
func New(b *codenames.Board, startingTeam codenames.Team, cfg *Config) (*Game, error) {
	if err := validateBoard(b, startingTeam); err != nil {
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
		state: &codenames.GameState{
			StartingTeam: startingTeam,
			ActiveTeam:   startingTeam,
			ActiveRole:   codenames.SpymasterRole,
			Board:        b,
		},
		cfg: cfg,
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

type Action string

const (
	ActionGiveClue = Action("GIVE_CLUE")
	ActionGuess    = Action("GUESS")
)

type Move struct {
	Team codenames.Team

	Action Action
	// Only populated for Action == ActionGiveClue
	GiveClue *codenames.Clue
	// Only populated for Action == ActionGuess
	Guess string
}

func (g *Game) Move(mv *Move) (*codenames.GameState, codenames.GameStatus, error) {
	switch mv.Action {
	case ActionGiveClue:
		if g.state.ActiveRole != codenames.SpymasterRole {
			return nil, "", fmt.Errorf("can't give a clue when %q %q should be acting", g.state.ActiveTeam, g.state.ActiveRole)
		}
		if mv.GiveClue == nil {
			return nil, "", errors.New("no clue was given")
		}
		g.handleGiveClue(mv.GiveClue)
	case ActionGuess:
		if g.state.ActiveRole != codenames.OperativeRole {
			return nil, "", fmt.Errorf("can't guess when %q %q should be acting", g.state.ActiveTeam, g.state.ActiveRole)
		}
		if mv.Guess == "" {
			// This is passing.
			g.endTurn()
		} else {
			if err := g.handleGuess(mv.Guess); err != nil {
				return nil, "", fmt.Errorf("handleGuess(%q): %w", mv.Guess, err)
			}
		}
	default:
		return nil, "", fmt.Errorf("unknown action %q", mv.Action)
	}

	state := codenames.Pending
	if over, _ := g.GameOver(); over {
		state = codenames.Finished
	}

	return g.state, state, nil
}

func (g *Game) handleGiveClue(clue *codenames.Clue) {
	numGuesses := clue.Count
	if numGuesses == 0 {
		numGuesses = -1
	}

	g.state.NumGuessesLeft = numGuesses
	g.state.ActiveRole = codenames.OperativeRole
}

func (g *Game) handleGuess(guess string) error {
	g.state.NumGuessesLeft--

	c, err := g.reveal(guess)
	if err != nil {
		return fmt.Errorf("reveal(%q) on %q: %v", guess, g.state.ActiveTeam, err)
	}

	// Check if their guess ended the game.
	if over, _ := g.GameOver(); over {
		return nil
	}

	if !g.canKeepGuessing(c) {
		g.endTurn()
	}

	return nil
}

func (g *Game) endTurn() {
	curTeam := g.state.ActiveTeam
	if curTeam == codenames.BlueTeam {
		curTeam = codenames.RedTeam
	} else {
		curTeam = codenames.BlueTeam
	}
	g.state.NumGuessesLeft = 0
	g.state.ActiveTeam = curTeam
	g.state.ActiveRole = codenames.SpymasterRole
}

func (g *Game) Play() (*Outcome, error) {
	for {
		// Let's play a round.
		sm, op := g.cfg.RedSpymaster, g.cfg.RedOperative
		if g.state.ActiveTeam == codenames.BlueTeam {
			sm, op = g.cfg.BlueSpymaster, g.cfg.BlueOperative
		}

		clue, err := sm.GiveClue(codenames.CloneBoard(g.state.Board))
		if err != nil {
			return nil, fmt.Errorf("GiveClue on %q: %w", g.state.ActiveTeam, err)
		}
		if _, _, err := g.Move(&Move{
			Action:   ActionGiveClue,
			Team:     g.state.ActiveTeam,
			GiveClue: clue,
		}); err != nil {
			return nil, fmt.Errorf("error giving clue: %w", err)
		}

		for g.state.ActiveRole == codenames.OperativeRole {
			guess, err := op.Guess(codenames.Revealed(g.state.Board), clue)
			if err != nil {
				return nil, fmt.Errorf("Guess on %q: %v", g.state.ActiveTeam, err)
			}
			if _, _, err = g.Move(&Move{
				Action: ActionGuess,
				Team:   g.state.ActiveTeam,
				Guess:  guess,
			}); err != nil {
				return nil, fmt.Errorf("Guess on %q: %v", g.state.ActiveTeam, err)
			}
		}
	}
}

func (g *Game) reveal(word string) (codenames.Card, error) {
	for i, card := range g.state.Board.Cards {
		if strings.ToLower(card.Codename) != strings.ToLower(word) {
			continue
		}

		if g.state.Board.Cards[i].Revealed {
			return codenames.Card{}, fmt.Errorf("%q has already been guessed", word)
		}

		// If the card hasn't been reveal, reveal it.
		g.state.Board.Cards[i].Revealed = true
		g.state.Board.Cards[i].RevealedBy = g.state.ActiveTeam
		return card, nil
	}
	return codenames.Card{}, fmt.Errorf("no card found for guess %q", word)
}

func (g *Game) canKeepGuessing(card codenames.Card) bool {
	targetAgent := codenames.RedAgent
	if g.state.ActiveTeam == codenames.BlueTeam {
		targetAgent = codenames.BlueAgent
	}

	// They can keep guessing if the card was for their team and they have
	// guesses left.
	return card.Agent == targetAgent && g.state.NumGuessesLeft != 0
}

func (g *Game) GameOver() (bool, codenames.Team) {
	got := make(map[codenames.Agent]int)
	for i, cn := range g.state.Board.Cards {
		if g.state.Board.Cards[i].Revealed {
			got[cn.Agent]++
		}
	}

	for ag, wc := range want(g.state.StartingTeam) {
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
				switch g.state.ActiveTeam {
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
