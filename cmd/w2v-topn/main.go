package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"code.sajari.com/word2vec"
	w2v "github.com/bcspragu/Codenames/w2v"
)

func main() {
	var (
		modelFile = flag.String("model_file", "", "A binary-formatted word2vec pre-trained model file.")

		wordList = flag.String("words", "", "Comma-separated list of words. Use --word_file to pass a file of words instead.")
		wordFile = flag.String("word_file", "official.txt", "File with list of words (one word per line). Use --words to pass a list in manually instead.")

		inputN = flag.Int("input_n", 2, "The number of target words to find matches for.")
		topN   = flag.Int("top_n", 3, "The number of closest words from the model to output.")

		omitSubstringMatch = flag.Bool("omit_substring_match", true, "Whether to omit clue/answer pairs where one is a fully contained substring of the other.")
	)
	flag.Parse()

	if *modelFile == "" {
		fmt.Println("ERROR: You need to pass in a --model_file.")
		return
	}

	var words []string
	if *wordList == "" {
		content, err := ioutil.ReadFile(*wordFile)
		if err != nil {
			fmt.Printf("Could not read %s: %s\n", *wordFile, err)
		}
		words = strings.Split(string(content), "\n")
	} else {
		words = strings.Split(*wordList, ",")

	}

	ai, err := w2v.New(*modelFile)
	if err != nil {
		fmt.Printf("Failed to read in %s\n", *modelFile)
	}
	model := ai.Model

	for combo := range combinations(len(words), *inputN) {
		var buffer bytes.Buffer
		expr := word2vec.Expr{}
		for _, index := range combo {
			expr.Add(1, words[index])
			buffer.WriteString(words[index])
			buffer.WriteString(" ")
		}
		buffer.WriteString("-> ")

		var matches []word2vec.Match
		if *omitSubstringMatch {
			matches, _ = topNOmitSubstringMatches(model, expr, *topN)
		} else {
			matches, _ = model.CosN(expr, *topN)
		}

		for _, match := range matches {
			buffer.WriteString(match.Word)
			buffer.WriteString(" (")
			buffer.WriteString(strconv.FormatFloat(float64(match.Score), 'f', 3, 32))
			buffer.WriteString(") ")
		}
		fmt.Println(buffer.String())
	}
}

func topNOmitSubstringMatches(model *word2vec.Model, expr word2vec.Expr, topN int) ([]word2vec.Match, error) {
	n := topN
	var valid_matches []word2vec.Match
	for len(valid_matches) < topN {
		valid_matches = nil
		matches, _ := model.CosN(expr, n)
		for _, match := range matches {
			valid := true
			for word := range expr {
				if strings.Contains(word, match.Word) || strings.Contains(match.Word, word) {
					valid = false
					break
				}
			}
			if valid {
				valid_matches = append(valid_matches, match)
			}
		}
		n *= 2
	}

	return valid_matches[:topN], nil
}
