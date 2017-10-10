package vision

import "testing"

func TestRound(t *testing.T) {
	testcases := []struct {
		x, nearest, want int
	}{
		{17, 5, 15},
		{18, 5, 20},
		{17, 10, 20},
		{18, 10, 20},
		{17, 1, 17},
		{17, 2, 18},
		{18, 2, 18},
	}

	for _, tc := range testcases {
		if got := round(tc.x, tc.nearest); got != tc.want {
			t.Errorf("round(%d, %d) = %d, want %d", tc.x, tc.nearest, got, tc.want)
		}
	}
}
