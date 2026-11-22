package game

import (
	"strings"
	"testing"

	"github.com/kaoutar/grimhold/internal/open5e"
	"github.com/kaoutar/grimhold/internal/quest"
	"github.com/kaoutar/grimhold/internal/world"
)

func TestExploreFightsOffQuestWithHint(t *testing.T) {
	w, err := world.Load("../../world.json")
	if err != nil {
		t.Skip(err)
	}
	story, err := quest.LoadStory("../../story.json")
	if err != nil {
		t.Skip(err)
	}
	client := stubClient{monsters: map[string]*open5e.Monster{
		"rat":       {Slug: "rat", Name: "Rat", HitPoints: 1, ArmorClass: 10, CR: 0, Type: "beast"},
		"giant-rat": {Slug: "giant-rat", Name: "Giant Rat", HitPoints: 7, ArmorClass: 12, CR: 0.125, Type: "beast"},
	}}
	s := NewState(w, client, story, "Test")
	s.StopAmbush()
	s.Quest.SetFlag("met_herald")
	s.Location = "crossroads"

	msg, err := s.Explore()
	if err != nil {
		t.Fatal(err)
	}
	if s.Encounter == nil || !s.Encounter.Active {
		t.Fatal("explore should still start an encounter off the quest region")
	}
	if !strings.Contains(msg, "ashen-marsh") && !strings.Contains(msg, "Marais") {
		t.Fatalf("expected quest hint, got %q", msg)
	}
}
