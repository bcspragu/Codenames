package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bcspragu/Codenames/client"
	"github.com/bcspragu/Codenames/codenames"
	"github.com/bcspragu/Codenames/web"
	"github.com/olekukonko/tablewriter"
)

func main() {
	var (
		serverScheme = flag.String("server_scheme", "http", "The scheme of the server to connect to to play the game.")
		serverAddr   = flag.String("server_addr", "localhost:8080", "The address of the server to connect to to play the game.")
	)
	flag.Parse()

	// Now that we've joined the game, connect via WebSockets.
	reader := bufio.NewReader(os.Stdin)

	name := prompt(reader, "Enter a username: ")
	gameToJoin := prompt(reader, "Enter a game ID to join, or blank to create a game: ", allowEmpty())

	c, err := client.New(*serverScheme, *serverAddr)
	if err != nil {
		log.Fatalf("failed to create client: %v", err)
	}

	userID, err := c.CreateUser(name)
	if err != nil {
		log.Fatalf("failed to create user: %v", err)
	}

	var gameID codenames.GameID
	if gameToJoin == "" {
		gID, err := c.CreateGame()
		if err != nil {
			log.Fatalf("failed to create game: %v", err)
		}
		gameID = gID
		fmt.Printf("Created game %q\n", gameID)
	} else {
		gameID = codenames.GameID(gameToJoin)
	}

	if err := c.JoinGame(gameID, codenames.PlayerTypeHuman); err != nil {
		log.Fatalf("failed to join game: %v", err)
	}

	var (
		team codenames.Team
		role codenames.Role
	)

	// defer termui.Close()
	err = c.ListenForUpdates(gameID, client.WSHooks{
		OnConnect: func() {
			if gameToJoin == "" {
				// Means we created the game, so we need to start it.
				lobbyShell(reader, c, gameID)
			}
		},
		OnStart: func(gs *web.GameStart) {
			for _, p := range gs.Players {
				if p.PlayerID.ID == userID {
					team = p.Team
					role = p.Role
				}
			}
			// if err := termui.Init(); err != nil {
			// 	log.Fatalf("failed to initialize termui: %v", err)
			// }
			printBoard(gs.Game.State.Board)

			// If the game started, and we're the starter spymaster, give a clue.
			if role == codenames.SpymasterRole && gs.Game.State.ActiveTeam == team {
				if err := giveAClue(c, gameID, reader); err != nil {
					log.Fatalf("failed to give clue: %v", err)
				}
			}
		},
		OnClueGiven: func(cg *web.ClueGiven) {
			fmt.Printf("Clue Given: %q %d\n", cg.Clue.Word, cg.Clue.Count)

			if role != codenames.OperativeRole || team != cg.Team {
				return
			}

			// If we're an operative, and the clue was given for our team, let's
			// guess.
			if err := giveAGuess(c, gameID, cg.Game.State.Board, reader); err != nil {
				log.Fatalf("failed to give clue: %v", err)
			}
		},
		OnPlayerVote: func(pv *web.PlayerVote) {
			// TODO: Show the vote
		},
		OnGuessGiven: func(gg *web.GuessGiven) {
			fmt.Printf("Guess was %q, card was %+v\n", gg.Guess, gg.RevealedCard)

			// We're an operative on the active team and we got the last one correct
			// and have guesses left.
			if gg.CanKeepGuessing && role == codenames.OperativeRole && team == gg.Team {
				if err := giveAGuess(c, gameID, gg.Game.State.Board, reader); err != nil {
					log.Fatalf("failed to give clue: %v", err)
				}
			}

			// We're the opposing spymaster and the other team is done guessing.
			if !gg.CanKeepGuessing && role == codenames.SpymasterRole && team != gg.Team {
				if err := giveAClue(c, gameID, reader); err != nil {
					log.Fatalf("failed to give clue: %v", err)
				}
			}
		},
		OnEnd: func(ge *web.GameEnd) {
			fmt.Printf("Game over, %q won!", ge.WinningTeam)
		},
	})
	if err != nil {
		log.Fatalf("failed to listen for updates: %v", err)
	}
}

func giveAClue(c *client.Client, gameID codenames.GameID, reader *bufio.Reader) error {
	clue := getAClue(reader)
	if err := c.GiveClue(gameID, clue); err != nil {
		return fmt.Errorf("failed to send clue: %w", err)
	}
	return nil
}

func getAClue(reader *bufio.Reader) *codenames.Clue {
	for {
		fmt.Print("Enter a clue: ")
		clueStr, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("failed to read clue: %v", err)
		}
		clue, err := codenames.ParseClue(strings.TrimSpace(clueStr))
		if err != nil {
			fmt.Printf("malformed clue, please try again: %v", err)
			continue
		}
		return clue
	}
}

func giveAGuess(c *client.Client, gameID codenames.GameID, board *codenames.Board, reader *bufio.Reader) error {
	guess, confirmed := getAGuess(reader, board)
	if err := c.GiveGuess(gameID, guess, confirmed); err != nil {
		return fmt.Errorf("failed to send guess: %w", err)
	}
	return nil
}

func getAGuess(reader *bufio.Reader, board *codenames.Board) (string, bool) {
	for {
		fmt.Print("Enter a guess: ")
		guess, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("failed to read guess: %v", err)
		}
		guess = strings.ToLower(strings.TrimSpace(guess))
		if !guessInCards(guess, board.Cards) {
			fmt.Println("guess was not found on board, please try again")
			continue
		}

		fmt.Print("Confirmed? (Y/n): ")
		confirmedStr, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("failed to read guess: %v", err)
		}
		confirmedStr = strings.TrimSpace(confirmedStr)
		confirmed := len(confirmedStr) == 0 || strings.ToLower(confirmedStr[0:1]) == "y"

		return guess, confirmed
	}
}

func guessInCards(guess string, cards []codenames.Card) bool {
	for _, c := range cards {
		if guess == strings.ToLower(c.Codename) {
			return true
		}
	}
	return false
}

func printBoard(b *codenames.Board) {
	table := tablewriter.NewWriter(os.Stdout)

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

func lobbyShell(reader *bufio.Reader, c *client.Client, gameID codenames.GameID) {
	fmt.Println("Welcome to the pre-game lobby! Enter 'help' for help")
	for {
		txt, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Failed to get text: %v", err)
			continue
		}
		txt = strings.TrimSpace(txt)

		switch {
		case txt == "help":
			printHelp()
			continue
		case txt == "players":
			players, err := c.Players(gameID)
			if err != nil {
				log.Printf("failed to list players: %v", err)
				continue
			}
			printPlayers(players)
			continue
		case txt == "start":
			if err := c.StartGame(gameID); err != nil {
				log.Printf("failed to start game: %v", err)
				continue
			}
			return
		case strings.HasPrefix(txt, "assign"):
			ps := strings.Split(txt, " ")
			if len(ps) != 4 {
				log.Println("invalid args, expected 4")
				continue
			}
			pID := codenames.PlayerID{
				PlayerType: codenames.PlayerTypeHuman,
				ID:         ps[1],
			}
			team, ok := codenames.ToTeam(ps[2])
			if !ok {
				log.Printf("invalid team %q", ps[2])
				continue
			}

			role, ok := codenames.ToRole(ps[3])
			if !ok {
				log.Printf("invalid role %q", ps[3])
				continue
			}

			if err := c.AssignRole(gameID, pID, team, role); err != nil {
				log.Printf("failed to assign role: %v", err)
			}
			continue
		}
		break
	}
}

func printHelp() {
	help := []string{
		"help\t\t\t\tShow this help text",
		"start\t\t\t\tTry to start the game",
		"players\t\t\t\tList the players in the lobby",
		"assign PLAYER TEAM ROLE\t\tAssign a player (by ID) to a given team/role",
	}

	fmt.Println()
	fmt.Println(strings.Join(help, "\n"))
	fmt.Println()
}

func printPlayers(ps []*web.Player) {
	for _, p := range ps {
		fmt.Printf("[%s] %s", p.PlayerID.ID, p.Name)
		if p.Role != codenames.NoRole && p.Team != codenames.NoTeam {
			fmt.Printf(" - %s %s", p.Team, p.Role)
		}
		fmt.Println()
	}
}

type promptOption func(*promptOptions)

type promptOptions struct {
	allowEmpty bool
}

func allowEmpty() promptOption {
	return func(po *promptOptions) {
		po.allowEmpty = true
	}
}

func prompt(reader *bufio.Reader, prmpt string, opts ...promptOption) string {
	pos := &promptOptions{}
	for _, opt := range opts {
		opt(pos)
	}

	for {
		fmt.Print(prmpt)
		txt, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("failed to get text: %v", err)
			continue
		}
		txt = strings.TrimSpace(txt)
		if txt == "" && !pos.allowEmpty {
			log.Print("no text entered")
			continue
		}
		return txt
	}
}
