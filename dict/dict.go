package dict

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

type Dictionary struct {
	words map[string]struct{}
}

// Init initializes the dictionary.
func New(file string) (*Dictionary, error) {
	d := &Dictionary{words: make(map[string]struct{})}

	log.Println("Opening dictionary...")
	f, err := os.Open(file)
	if os.IsNotExist(err) {
		// If the dictionary file doesn't exist, we just let everything pass.
		log.Println("Dictionary doesn't exist, will allow all words.")
		return d, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open dictionary file %q: %v", file, err)
	}
	defer f.Close()

	log.Println("Reading dictionary...")
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		d.words[strings.ToLower(sc.Text())] = struct{}{}
	}
	if sc.Err() != nil {
		return nil, fmt.Errorf("failed to read dictionary: %v", err)
	}
	log.Println("Read dictionary")

	return d, nil
}

// Valid returns if the given word is a valid Codenames word. If the dictionary
// was not initialized (i.e. it has no words in it), consider all words valid.
func (d *Dictionary) Valid(word string) bool {
	if len(d.words) == 0 {
		return true
	}
	_, valid := d.words[strings.ToLower(word)]
	return valid
}
