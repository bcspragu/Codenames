package boardgen

import (
	"math/rand"
	"testing"

	"github.com/bcspragu/Codenames/codenames"
	"github.com/google/go-cmp/cmp"
)

func TestNew(t *testing.T) {
	got := New(codenames.RedTeam, rand.New(rand.NewSource(0)))

	want := &codenames.Board{
		Cards: []codenames.Card{
			{Codename: "cliff", Agent: codenames.BlueAgent},
			{Codename: "dwarf", Agent: codenames.BlueAgent},
			{Codename: "green", Agent: codenames.RedAgent},
			{Codename: "doctor", Agent: codenames.Bystander},
			{Codename: "ship", Agent: codenames.BlueAgent},
			{Codename: "dance", Agent: codenames.BlueAgent},
			{Codename: "time", Agent: codenames.Bystander},
			{Codename: "pool", Agent: codenames.RedAgent},
			{Codename: "cover", Agent: codenames.Bystander},
			{Codename: "fighter", Agent: codenames.BlueAgent},
			{Codename: "horse", Agent: codenames.RedAgent},
			{Codename: "strike", Agent: codenames.RedAgent},
			{Codename: "cast", Agent: codenames.RedAgent},
			{Codename: "string", Agent: codenames.RedAgent},
			{Codename: "greece", Agent: codenames.Assassin},
			{Codename: "fence", Agent: codenames.Bystander},
			{Codename: "drill", Agent: codenames.RedAgent},
			{Codename: "button", Agent: codenames.BlueAgent},
			{Codename: "cycle", Agent: codenames.Bystander},
			{Codename: "chest", Agent: codenames.RedAgent},
			{Codename: "pitch", Agent: codenames.BlueAgent},
			{Codename: "unicorn", Agent: codenames.RedAgent},
			{Codename: "agent", Agent: codenames.BlueAgent},
			{Codename: "kiwi", Agent: codenames.Bystander},
			{Codename: "swing", Agent: codenames.Bystander},
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected board (-want +got)\n%s", diff)
	}
}
