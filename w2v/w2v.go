package w2v

import (
	"fmt"
	"os"
	"strings"

	"github.com/sajari/word2vec"
)

type gsr struct {
	model *word2vec.Model
}

var guesser *gsr

func init() {
	fmt.Println("Opening w2v model...")
	f, err := os.Open("w2v.bin")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Println("Reading w2v model...")
	model, err := word2vec.FromReader(f)
	if err != nil {
		panic(err)
	}

	guesser = &gsr{
		model: model,
	}
	fmt.Println("Read w2v model")
}

// Similarity returns a value from 0 to 1, that is the similarity of the two
// input words.
func Similarity(a, b string) (float32, error) {
	s, err := guesser.model.Cos(exp(strings.ToLower(a)), exp(strings.ToLower(b)))
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
