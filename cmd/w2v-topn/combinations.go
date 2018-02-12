package main

// Returns the unique subsets of size k
// of integers 0..n-1, in order.
//
// There are n! / (k! * (n-k)!) values.
//
// Usage:
//    for combo := range Combinations(5,3) {
//      ...
//    }
//
// Example: Combinations(5,3) ->
//   [0,1,2], [0,1,3], [0,1,4], [0,2,3]
//   [0,2,4], [0,3,4], [1,2,3], [1,2,4],
//   [1,2,4], [2,3,4]
func combinations(n, k int) chan []int {
	ch := make(chan []int)
	go combinationsInternal(ch, n, k)
	return ch
}

func combinationsInternal(ch chan []int, n int, k int) {
    defer close(ch)

	a := make([]int, k+1)  // a[0] holds a dummy value
	for i := range a {
		a[i] = i-1
	}

    if (k < 0 || k > n) {
        return
    }

    if (k == n) {
        ch <- a[1:]
        return
    }

	for {
        // Need to make a copy to put in the channel,
        // otherwise future values with overwrite old ones.
	    b := make([]int, len(a)-1)
	    copy(b, a[1:])
	    ch <- b

		var j int

        // Look right to left to find the
        // first digit that can be incremented.
		for j = k; a[j] == n-k+j-1; j-- {}

		if j == 0 {
			break
		}

		a[j] += 1

        // Reset all the values after a[j]
        // to be a[j]+1, a[j]+2, a[j]+3, etc.
		for i := j + 1; i <= k; i++ {
			a[i] = a[i-1] + 1
		}
	}
}
