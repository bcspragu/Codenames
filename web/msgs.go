package web

import (
	"encoding/json"

	"github.com/bcspragu/Codenames/codenames"
)

type Player struct {
	PlayerID codenames.PlayerID `json:"player_id"`
	Name     string             `json:"name"`
	Team     codenames.Team     `json:"team"`
	Role     codenames.Role     `json:"role"`
}

type GameStart struct {
	Game    *codenames.Game `json:"game"`
	Players []*Player       `json:"players"`
}

func (gs *GameStart) MarshalJSON() ([]byte, error) {
	return withAction("GAME_START", gs)
}

type ClueGiven struct {
	Clue *codenames.Clue `json:"clue"`
	Team codenames.Team  `json:"team"`
}

func (cg *ClueGiven) MarshalJSON() ([]byte, error) {
	return withAction("CLUE_GIVEN", cg)
}

type PlayerVote struct {
	UserID    codenames.UserID `json:"user_id"`
	Guess     string           `json:"guess"`
	Confirmed bool             `json:"confirmed"`
}

func (pv *PlayerVote) MarshalJSON() ([]byte, error) {
	return withAction("PLAYER_VOTE", pv)
}

type GuessGiven struct {
	Guess        string          `json:"guess"`
	RevealedCard *codenames.Card `json:"card"`
}

func (gg *GuessGiven) MarshalJSON() ([]byte, error) {
	return withAction("GUESS_GIVEN", gg)
}

type GameEnd struct {
	WinningTeam codenames.Team `json:"winning_team"`
}

func (ge *GameEnd) MarshalJSON() ([]byte, error) {
	return withAction("GAME_END", ge)
}

type embed interface{}

func withAction(action string, msg interface{}) ([]byte, error) {
	toMarshal := struct {
		embed
		Action string `json:"action"`
	}{msg, action}

	return json.Marshal(toMarshal)
}
