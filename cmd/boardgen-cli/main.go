package main

import (
	"bytes"
	"fmt"

	codenames "github.com/bcspragu/Codenames"
	"github.com/bcspragu/Codenames/boardgen"
)

var (
	agentNames = map[codenames.Agent]string{
		codenames.RedAgent:  "red",
		codenames.BlueAgent: "blue",
		codenames.Bystander: "bystander",
		codenames.Assassin:  "assassin",
	}
)

func main() {
	// For now, assume Red Team goes first.
	bd := boardgen.New(codenames.RedTeam)

	var buf bytes.Buffer
	for i, card := range bd.Cards {
		buf.WriteString(fmt.Sprintf("%s:%s", card.Codename, agentNames[card.Agent]))
		if i != len(bd.Cards)-1 {
			buf.WriteString(",")
		}
	}

	fmt.Print(buf.String())
}
