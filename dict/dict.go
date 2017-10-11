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

// Init initializes the dictionary.
func Init(dictFile string) error {
	dictionary = &dict{
		words: make(map[string]struct{}),
	}
	fmt.Println("Opening dictionary...")
	f, err := os.Open(dictFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Dictionary doesn't exist, will allow all words.")
			// If the dictionary file doesn't exist, we just let everything pass.
			return nil
		}
		return fmt.Errorf("failed to open dictionary file %q: %v", dictFile, err)
	}
	defer f.Close()

	fmt.Println("Reading dictionary...")
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		dictionary.words[sc.Text()] = struct{}{}
	}
	if sc.Err() != nil {
		return fmt.Errorf("failed to read dictionary: %v", err)
	}
	fmt.Println("Read dictionary")
	return nil
}

// Valid returns if the given word is a valid Codenames word. If the dictionary
// was not initialized (i.e. it has no words in it), consider all words valid.
func Valid(word string) bool {
	if len(dictionary.words) == 0 {
		return true
	}
	_, valid := dictionary.words[strings.ToUpper(word)]
	return valid
}
