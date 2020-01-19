package web

import (
	"testing"

	"github.com/bcspragu/Codenames/codenames"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestToJSBoard(t *testing.T) {
	tests := []struct {
		desc    string
		in      *codenames.Board
		want    *jsBoard
		wantErr bool
	}{
		{
			desc: "empty in, empty out",
			in:   &codenames.Board{},
			want: &jsBoard{},
		},
		{
			desc: "one in, one out",
			in:   &codenames.Board{Cards: []codenames.Card{{Codename: "test"}}},
			want: &jsBoard{Cards: [][]codenames.Card{{{Codename: "test"}}}},
		},
		{
			desc:    "two in fails",
			in:      &codenames.Board{Cards: []codenames.Card{{Codename: "test"}, {Codename: "test2"}}},
			wantErr: true,
		},
		{
			desc: "four in, 2x2 out",
			in: &codenames.Board{Cards: []codenames.Card{
				{Codename: "test"}, {Codename: "test2"},
				{Codename: "test3"}, {Codename: "test4"},
			}},
			want: &jsBoard{Cards: [][]codenames.Card{
				{{Codename: "test"}, {Codename: "test2"}},
				{{Codename: "test3"}, {Codename: "test4"}},
			}},
		},
		{
			desc: "nine in, 3x3 out",
			in: &codenames.Board{Cards: []codenames.Card{
				{Codename: "test"}, {Codename: "test2"}, {Codename: "test3"},
				{Codename: "test4"}, {Codename: "test5"}, {Codename: "test6"},
				{Codename: "test7"}, {Codename: "test8"}, {Codename: "test9"},
			}},
			want: &jsBoard{Cards: [][]codenames.Card{
				{{Codename: "test"}, {Codename: "test2"}, {Codename: "test3"}},
				{{Codename: "test4"}, {Codename: "test5"}, {Codename: "test6"}},
				{{Codename: "test7"}, {Codename: "test8"}, {Codename: "test9"}},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got, err := toJSBoard(test.in)
			if err != nil {
				if test.wantErr {
					// This is expected.
					return
				}
				t.Errorf("toJSBoard: %v", err)
				return
			}

			if diff := cmp.Diff(test.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("toJSBoard (-want +got):\n%s", diff)
			}
		})
	}
}

func TestSqrt(t *testing.T) {
	tests := []struct {
		in       int
		want     int
		wantFail bool
	}{
		{
			in:   0,
			want: 0,
		},
		{
			in:   1,
			want: 1,
		},
		{
			in:       2,
			wantFail: true,
		},
		{
			in:       3,
			wantFail: true,
		},
		{
			in:   4,
			want: 2,
		},
		{
			in:   9,
			want: 3,
		},
		{
			in:       31328,
			wantFail: true,
		},
		{
			in:   31329,
			want: 177,
		},
	}

	for _, test := range tests {
		got, ok := sqrt(test.in)
		if test.wantFail {
			if !ok {
				// This is expected.
				continue
			}
			t.Errorf("Got %d, wanted failure", got)
			continue
		}

		if got != test.want {
			t.Errorf("sqrt(%d) = %d, want %d", test.in, got, test.want)
		}
	}
}
