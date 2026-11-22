package ui

import (
	"strings"
	"testing"

	"github.com/kaoutar/grimhold/internal/game"
	"github.com/kaoutar/grimhold/internal/quest"
	"github.com/kaoutar/grimhold/internal/world"
)

func testState(t *testing.T) *game.State {
	t.Helper()
	w, err := world.Load("../../world.json")
	if err != nil {
		t.Fatal(err)
	}
	story, err := quest.LoadStory("../../story.json")
	if err != nil {
		t.Fatal(err)
	}
	s := game.NewState(w, nil, story, "Test")
	t.Cleanup(s.StopAmbush)
	return s
}

func TestTutorialIntroExplainsInterface(t *testing.T) {
	s := testState(t)
	lines := tutorialIntro(s)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"Tutoriel", "Quête", "talk"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("intro missing %q:\n%s", want, joined)
		}
	}
}

func TestTutorialTipAndSuggestFirstMission(t *testing.T) {
	s := testState(t)

	tip := tutorialTip(s)
	if !strings.Contains(tip, "talk") {
		t.Fatalf("prologue tip should mention talk, got %q", tip)
	}
	sug := Suggest(s)
	if len(sug) == 0 || !strings.Contains(sug[0], "talk") {
		t.Fatalf("suggest should prioritize talk, got %#v", sug)
	}

	s.Quest.SetFlag("met_herald")
	tip = tutorialTip(s)
	if !strings.Contains(tip, "Marais") && !strings.Contains(tip, "[1]") {
		t.Fatalf("after talk tip should push marsh travel, got %q", tip)
	}
	sug = Suggest(s)
	if len(sug) == 0 || !strings.Contains(sug[0], "ashen-marsh") {
		t.Fatalf("suggest should point to marsh exit, got %#v", sug)
	}

	s.Location = "ashen-marsh"
	tip = tutorialTip(s)
	if !strings.Contains(tip, "explore") {
		t.Fatalf("at marsh tip should mention explore, got %q", tip)
	}
	sug = Suggest(s)
	if len(sug) == 0 || !strings.Contains(sug[0], "explore") {
		t.Fatalf("suggest should prioritize explore, got %#v", sug)
	}

	s.Quest.SetFlag("shard_marsh")
	if tutorialTip(s) != "" {
		t.Fatalf("tutorial should end after shard_marsh")
	}
}

func TestTutorialTipCombat(t *testing.T) {
	s := testState(t)
	s.Quest.SetFlag("met_herald")
	s.Location = "ashen-marsh"
	s.Encounter = &game.Encounter{Active: true}
	tip := tutorialTip(s)
	if !strings.Contains(tip, "fight") && !strings.Contains(tip, "Combat") {
		t.Fatalf("combat tip missing, got %q", tip)
	}
}
