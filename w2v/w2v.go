package w2v

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/bcspragu/Codenames/codenames"

	"code.sajari.com/word2vec"
)

type AI struct {
	Model *word2vec.Model
}

// Init initializes the word2vec model.
func New(file string) (*AI, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open model file %q: %w", file, err)
	}
	defer f.Close()

	model, err := word2vec.FromReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse model file %q: %w", file, err)
	}

	return &AI{Model: model}, nil
}

func (ai *AI) GiveClue(b *codenames.Board, agent codenames.Agent) (*codenames.Clue, error) {
	bestScore := float32(-1.0)
	clue := "???"

	for _, word := range toWordList(codenames.Unrevealed(codenames.Targets(b.Cards, agent))) {
		expr := word2vec.Expr{}
		expr.Add(1, word)
		matches, err := ai.Model.CosN(expr, 5)
		if errors.Is(err, word2vec.NotFoundError{}) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load similar words: %w", err)
		}

		for _, match := range matches {
			if tooCloseToBoardWord(match.Word, b) {
				continue
			}
			if match.Score > bestScore {
				bestScore = match.Score
				clue = match.Word
			}
		}
	}

	return &codenames.Clue{Word: clue, Count: 1}, nil
}

func tooCloseToBoardWord(clue string, b *codenames.Board) bool {
	for _, card := range b.Cards {
		if strings.Contains(clue, card.Codename) || strings.Contains(card.Codename, clue) {
			return true
		}
	}
	return false
}

func toWordList(targets []codenames.Card) []string {
	var available []string
	for _, c := range targets {
		// Some cards contain underscores, which makes them unlikely to appear in
		// the model corpus. So what we do is we try to insert two copies of the
		// word, one with the underscore removed, and one with the underscore
		// replaced with a space. The idea is that hopefully one of these appears
		// in the source corpus.
		if strings.Contains(c.Codename, "_") {
			available = append(available, strings.Replace(c.Codename, "_", "", -1))
			available = append(available, strings.Replace(c.Codename, "_", " ", -1))
		} else {
			available = append(available, c.Codename)
		}
	}
	return available
}

func (ai *AI) Guess(b *codenames.Board, c *codenames.Clue) (string, error) {
	type pair struct {
		Word       string
		Similarity float32
	}

	var pairs []pair
	for _, word := range toWordList(codenames.Unused(b.Cards)) {
		sim, err := ai.similarity(c.Word, word)
		if errors.Is(err, word2vec.NotFoundError{}) {
			continue
		}
		if err != nil {
			return "", fmt.Errorf("failed to get similarity of %q and %q: %w", c.Word, word, err)
		}

		pairs = append(pairs, pair{
			Word:       word,
			Similarity: sim,
		})
	}

	// Sort the board words most similar -> least similar.
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Similarity > pairs[j].Similarity
	})

	return pairs[0].Word, nil
}

// Similarity returns a value from 0 to 1, that is the similarity of the two
// input words.
func (ai *AI) similarity(a, b string) (float32, error) {
	s, err := ai.Model.Cos(exp(strings.ToLower(a)), exp(strings.ToLower(b)))
	if err != nil {
		return 0.0, fmt.Errorf("failed to determine similarity: %w", err)
	}
	return s, nil
}

func exp(w string) word2vec.Expr {
	expr := word2vec.Expr{}
	expr.Add(1, w)
	return expr
}
