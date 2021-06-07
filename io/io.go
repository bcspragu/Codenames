package io

import (
	"bufio"
	"fmt"
	"io"

	"github.com/bcspragu/Codenames/codenames"
	"github.com/olekukonko/tablewriter"
)

// Spymaster asks the user on the terminal to enter a clue. It assumes they
// have the board already available to them and don't need to see it.
type Spymaster struct {
	// in is a reader where the user's clue is read from.
	In io.Reader
	// out is where the prompts should be written out to.
	Out io.Writer
}

func agentStr(a codenames.Agent) string {
	switch a {
	case codenames.BlueAgent:
		return "Blue"
	case codenames.RedAgent:
		return "Red"
	default:
		return "Unknown"
	}
}

func (s *Spymaster) GiveClue(b *codenames.Board, agent codenames.Agent) (*codenames.Clue, error) {
	s.printBoard(b)
	fmt.Fprintf(s.Out, "%s Spymaster, enter a clue [ex. 'Muffins 3']: ", agentStr(agent))
	sc := bufio.NewScanner(s.In)
	if !sc.Scan() {
		return nil, fmt.Errorf("scanner error: %v", sc.Err())
	}
	return codenames.ParseClue(sc.Text())
}

func (s *Spymaster) printBoard(b *codenames.Board) {
	table := tablewriter.NewWriter(s.Out)

	for i := 0; i < 5; i++ {
		var row []string
		var colors []tablewriter.Colors
		for j := 0; j < 5; j++ {
			card := b.Cards[i*5+j]
			var c tablewriter.Colors
			switch card.Agent {
			case codenames.BlueAgent:
				c = append(c, tablewriter.FgBlueColor)
			case codenames.RedAgent:
				c = append(c, tablewriter.FgHiRedColor)
			case codenames.Assassin:
				c = append(c, tablewriter.BgHiRedColor)
			}
			if card.Revealed {
				c = append(c, tablewriter.UnderlineSingle)
			}
			colors = append(colors, c)
			row = append(row, card.Codename)
		}
		table.Rich(row, colors)
	}

	table.Render()
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
