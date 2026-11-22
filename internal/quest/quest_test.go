package quest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTalkAndEnding(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "story.json")
	data := `{
		"title": "Test",
		"prologue": "Once",
		"acts": [
			{"id":0,"title":"P","objective":"talk","region":"crossroads","grant_flag":"met_herald","on_grant":"hello"},
			{"id":4,"title":"Boss","objective":"boss","region":"bleak-spire","grant_flag":"boss_felled","boss_slug":"wraith","on_grant":"down"}
		],
		"region_flags": {},
		"endings": {
			"mercy": {"flag":"ending_mercy","title":"Mercy","text":"soft"},
			"wrath": {"flag":"ending_wrath","title":"Wrath","text":"hard"}
		}
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadStory(path)
	if err != nil {
		t.Fatal(err)
	}
	p := NewProgress()
	msg, err := s.Talk(&p, "crossroads")
	if err != nil || !p.Has("met_herald") {
		t.Fatalf("talk: %v %q flags=%v", err, msg, p.Flags)
	}
	p.SetFlag("boss_felled")
	end, err := s.ChooseEnding(&p, "mercy")
	if err != nil || p.Ending != "mercy" {
		t.Fatalf("ending: %v %q", err, end)
	}
}

func TestSpireGate(t *testing.T) {
	s := &Story{
		Acts: []Act{{BossSlug: "wraith", MinLevel: 2, GrantFlag: "boss_felled"}},
	}
	p := NewProgress()
	if ok, _ := s.SpireUnlocked(p, 5); ok {
		t.Fatal("should need key")
	}
	p.SetFlag("key_complete")
	if ok, _ := s.SpireUnlocked(p, 1); ok {
		t.Fatal("should need level")
	}
	if ok, _ := s.SpireUnlocked(p, 2); !ok {
		t.Fatal("should unlock")
	}
}
