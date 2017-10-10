package dict

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type dict struct {
	words map[string]struct{}
}

var dictionary *dict

func init() {
	dictionary = &dict{
		words: make(map[string]struct{}),
	}
	fmt.Println("Opening dictionary...")
	f, err := os.Open("words.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	fmt.Println("Reading dictionary...")
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		dictionary.words[sc.Text()] = struct{}{}
	}
	if sc.Err() != nil {
		panic(err)
	}
	fmt.Println("Read didctionary")
}

// Valid returns if the given word is a valid Codenames word.
func Valid(word string) bool {
	_, valid := dictionary.words[strings.ToUpper(word)]
	return valid
}
