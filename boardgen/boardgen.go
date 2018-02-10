package boardgen

import (
	"math/rand"
	"time"

	codenames "github.com/bcspragu/Codenames"
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

func New(starter codenames.Team) *codenames.Board {
	rand.Seed(time.Now().UnixNano())

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
		used[words[rand.Intn(len(words))]] = struct{}{}
	}

	var selected []string
	for word := range used {
		selected = append(selected, word)
	}

	var cards []codenames.Card
	for i, idx := range rand.Perm(len(agents)) {
		cards = append(cards, codenames.Card{
			Agent:    agents[idx],
			Codename: selected[i],
		})
	}

	return &codenames.Board{Cards: cards}
}
