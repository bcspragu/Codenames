package consensus

import (
	"sync"

	"github.com/bcspragu/Codenames/codenames"
)

func New() *Guesser {
	return &Guesser{
		guesses: make(map[codenames.GameID][]*Vote),
	}
}

type Vote struct {
	UserID codenames.UserID
	Word   string
}

type Guesser struct {
	mu      sync.Mutex
	guesses map[codenames.GameID][]*Vote
}

func (g *Guesser) RecordVote(gID codenames.GameID, uID codenames.UserID, word string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, vote := range g.guesses[gID] {
		if vote.UserID == uID {
			// Update an existing player's vote.
			vote.Word = word
			return
		}
	}

	// If we're here, this is a new vote.
	g.guesses[gID] = append(g.guesses[gID], &Vote{
		UserID: uID,
		Word:   word,
	})
}

func (g *Guesser) ReachedConsensus(gID codenames.GameID, totalVoters int) (string, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()

	votes := make(map[string]int)
	for _, vote := range g.guesses[gID] {
		votes[vote.Word]++
	}

	// We require a strict majority, meaning > 50%. E.g.
	// totalVoters == 2, majority == 2
	// totalVoters == 3, majority == 2
	// totalVoters == 4, majority == 3
	// totalVoters == 5, majority == 3
	// totalVoters == 6, majority == 4
	majority := totalVoters/2 + 1
	for word, cnt := range votes {
		if cnt >= majority {
			return word, true
		}
	}

	return "", false
}

func (g *Guesser) Clear(gID codenames.GameID) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.guesses, gID)
}
