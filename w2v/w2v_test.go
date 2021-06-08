package w2v

import (
	"testing"

	"github.com/bcspragu/Codenames/codenames"
	"github.com/google/go-cmp/cmp"
)

func TestToWordList(t *testing.T) {
	tests := []struct {
		desc string
		in   []codenames.Card
		want []string
	}{
		{
			desc: "no underscores",
			in: []codenames.Card{
				card("helicopter"),
				card("mail"),
				card("revolution"),
			},
			want: []string{
				"helicopter",
				"mail",
				"revolution",
			},
		},
		{
			desc: "some underscores",
			in: []codenames.Card{
				card("helicopter"),
				card("ice_cream"),
				card("revolution"),
			},
			want: []string{
				"helicopter",
				"icecream",
				"ice cream",
				"revolution",
			},
		},
		{
			desc: "all underscores",
			in: []codenames.Card{
				card("ice_cream"),
				card("loch_ness"),
				card("scuba_diver"),
			},
			want: []string{
				"icecream",
				"ice cream",
				"lochness",
				"loch ness",
				"scubadiver",
				"scuba diver",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := toWordList(test.in)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Fatalf("unexpected word list (-want +got)\n%s", diff)
			}
		})
	}
}

func card(word string) codenames.Card {
	return codenames.Card{Codename: word}
}
