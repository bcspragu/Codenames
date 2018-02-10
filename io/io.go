package io

import (
	"bufio"
	"fmt"
	"io"

	codenames "github.com/bcspragu/Codenames"
)

// Spymaster asks the user on the terminal to enter a clue. It assumes they
// have the board already available to them and don't need to see it.
type Spymaster struct {
	// in is a reader where the user's clue is read from.
	In io.Reader
	// out is where the prompts should be written out to.
	Out io.Writer
	// team is which team this Spymaster is on.
	Team codenames.Team
}

func (s *Spymaster) GiveClue(_ *codenames.Board) (*codenames.Clue, error) {
	fmt.Fprintf(s.Out, "%s Spymaster, enter a clue [ex. 'Muffins 3']: ", s.Team)
	sc := bufio.NewScanner(s.In)
	if !sc.Scan() {
		return nil, fmt.Errorf("scanner error: %v", sc.Err())
	}
	return codenames.ParseClue(sc.Text())
}

// Operative asks the user on the terminal to enter a guess.
type Operative struct {
	// in is a reader where the user's guess is read from.
	In io.Reader
	// out is where the prompts should be written out to.
	Out io.Writer
	// team is which team this Operative is on.
	Team codenames.Team
}

func (o *Operative) Guess(_ *codenames.Board, c *codenames.Clue) (string, error) {
	fmt.Fprintf(o.Out, "%s Operative, enter a guess for hint '%s': ", o.Team, c)
	sc := bufio.NewScanner(o.In)
	if !sc.Scan() {
		return "", fmt.Errorf("scanner error: %v", sc.Err())
	}
	return sc.Text(), nil
}
