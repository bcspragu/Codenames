package main

import "github.com/bcspragu/Codenames/w2v"

func main() {
    var result Result
    operative := w2v.New("model.bin")
    for _, scenario := range Scenarios {
        board = OperativeBoard(scenario)
        operative.Guess(board, scenario.Clue)

    }
}

