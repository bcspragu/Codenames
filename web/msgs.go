package web

import (
	"encoding/json"

	"github.com/bcspragu/Codenames/codenames"
)

type GameStart struct {
	Game    *codenames.Game         `json:"game"`
	Players []*codenames.PlayerRole `json:"players"`
}

func (gs *GameStart) MarshalJSON() ([]byte, error) {
	return withAction("GAME_START", gs)
}

type embed interface{}

func withAction(action string, msg interface{}) ([]byte, error) {
	toMarshal := struct {
		embed
		Action string `json:"action"`
	}{msg, action}

	return json.Marshal(toMarshal)
}
