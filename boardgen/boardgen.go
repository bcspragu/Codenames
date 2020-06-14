package boardgen

import (
	"math/rand"

	"github.com/bcspragu/Codenames/codenames"
)

var baseAgents = []codenames.Agent{
	codenames.RedAgent,
	codenames.RedAgent,
	codenames.RedAgent,
	codenames.RedAgent,
	codenames.RedAgent,
	codenames.RedAgent,
	codenames.RedAgent,
	codenames.RedAgent,
	codenames.BlueAgent,
	codenames.BlueAgent,
	codenames.BlueAgent,
	codenames.BlueAgent,
	codenames.BlueAgent,
	codenames.BlueAgent,
	codenames.BlueAgent,
	codenames.BlueAgent,
	codenames.Bystander,
	codenames.Bystander,
	codenames.Bystander,
	codenames.Bystander,
	codenames.Bystander,
	codenames.Bystander,
	codenames.Bystander,
	codenames.Assassin,
}

func New(starter codenames.Team, r *rand.Rand) *codenames.Board {
	used := make(map[string]struct{})
	agents := make([]codenames.Agent, len(baseAgents))
	copy(agents, baseAgents)

	switch starter {
	case codenames.RedTeam:
		agents = append(agents, codenames.RedAgent)
	case codenames.BlueTeam:
		agents = append(agents, codenames.BlueAgent)
	}

	// Pick words at random from our list.
	for len(used) < codenames.Size {
		used[codenames.Words[r.Intn(len(codenames.Words))]] = struct{}{}
	}

	var selected []string
	for word := range used {
		selected = append(selected, word)
	}

	var cards []codenames.Card
	for i, idx := range r.Perm(len(agents)) {
		cards = append(cards, codenames.Card{
			Agent:    agents[idx],
			Codename: selected[i],
		})
	}

	return &codenames.Board{Cards: cards}
}
