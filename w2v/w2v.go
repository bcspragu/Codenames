package w2v

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	codenames "github.com/bcspragu/Codenames"

	"code.sajari.com/word2vec"
)

type AI struct {
	model *word2vec.Model
}

// Init initializes the word2vec model.
func New(file string) (*AI, error) {
	log.Println("Opening w2v model...")
	f, err := os.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open model file %q: %v", file, err)
	}
	defer f.Close()

	log.Println("Reading w2v model...")
	model, err := word2vec.FromReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse model file %q: %v", file, err)
	}

	log.Println("Read w2v model")
	return &AI{model: model}, nil
}

func (ai *AI) GiveClue(*codenames.Board) (*codenames.Clue, error) {
	// TODO: Figure out an algorithm for generating clues.
	return nil, errors.New("not implemented")
}

func (ai *AI) Guess(b *codenames.Board, c *codenames.Clue) (string, error) {
	unused := codenames.Unused(b.Cards)

	// TODO: Probably remove this check, maybe when we support the sneaky 0-count
	// clue.
	if c.Count > len(unused) {
		return "", fmt.Errorf("clue was for %d words, only %d words are available", c.Count, len(unused))
	}

	pairs := make([]struct {
		Word       string
		Similarity float32
	}, len(unused))

	for i, card := range unused {
		sim, err := ai.similarity(c.Word, card.Codename)
		if err != nil {
			return "", fmt.Errorf("failed to get similarity of %q and %q: %v", c.Word, card.Codename, err)
		}

		pairs[i].Word = card.Codename
		pairs[i].Similarity = sim
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
	s, err := ai.model.Cos(exp(strings.ToLower(a)), exp(strings.ToLower(b)))
	if err != nil {
		return 0.0, fmt.Errorf("failed to determine similarity: %v", err)
	}
	return s, nil
}

func exp(w string) word2vec.Expr {
	expr := word2vec.Expr{}
	expr.Add(1, w)
	return expr
}
