package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/mathetake/gann"
	"github.com/mathetake/gann/metric"
)

var (
	dim    = 3
	nTrees = 2
	k      = 10
	nItem  = 1000
)

func main() {
	var (
		modelPath = flag.String("model_path", "", "Path to the raw model output to load")
	)

	f, err := os.Open(*modelPath)
	if err != nil {
		log.Fatalf("failed to open model file: %v", err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)

	if !sc.Scan() {
		log.Fatalf("no first row containing file info found")
	}

	nItem, dim, err := parseHeader(sc.Text())
	if err != nil {
		log.Fatalf("failed to parse header: %v", err)
	}

	rawItems := make([][]float64, nItem)
	rand.Seed(time.Now().UnixNano())

	i := 0
	for sc.Scan() {
		row, err := parseRow(sc.Text(), dim)
		if err != nil {
			log.Fatalf("failed to parse row: %v", err)
		}
		rawItems[i] = row
		i++
	}

	m, err := metric.NewCosineMetric(dim)
	if err != nil {
		// err handling
		return
	}

	// create index
	idx, err := gann.CreateNewIndex(rawItems, dim, nTrees, k, m)
	if err != nil {
		// error handling
		return
	}

	// search
	var searchNum = 5
	var bucketScale float64 = 10
	q := []float64{0.1, 0.02, 0.001}
	res, err := idx.GetANNbyVector(q, searchNum, bucketScale)
	if err != nil {
		// error handling
		return
	}

	fmt.Printf("res: %v\n", res)
}

func parseHeader(in string) (int, int, error) {
	ps := strings.Split(in, " ")
	if n := len(ps); n != 2 {
		return 0, 0, fmt.Errorf("unexpected header %q had %d parts", in, n)
	}

	nItems, err := strconv.Atoi(ps[0])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse number of items: %w", err)
	}

	nDims, err := strconv.Atoi(ps[1])
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse number of dimensions: %w", err)
	}

	return nItems, nDims, nil
}

func parseRow(in string) ([]float
