package codenames

import (
	"math/rand"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRandomGameID(t *testing.T) {
	// I tried seeds 0-16, but none of them picked a word that has an underscore
	// in it, and that's the behavior I want to test. This one has ice_cream,
	// which gets converted into IceCream.
	r := rand.New(rand.NewSource(17))

	var got []GameID
	for i := 0; i < 5; i++ {
		got = append(got, RandomGameID(r))
	}

	want := []GameID{
		"KingTripCloak",
		"SwingCardScreen",
		"HandSchoolGermany",
		"PianoIceCreamMug",
		"BootAmazonNovel",
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected game IDs (-want +got)\n%s", diff)
	}
}
