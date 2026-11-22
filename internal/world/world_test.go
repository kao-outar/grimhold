package world

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndTravel(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "world.json")
	data := `{
		"start": "crossroads",
		"regions": [
			{"id":"crossroads","name":"Crossroads","description":"hub","danger":0,"foes":["rat"],"exits":["marsh"]},
			{"id":"marsh","name":"Marsh","description":"wet","danger":1,"foes":["goblin"],"exits":["crossroads"]}
		]
	}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	w, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !w.CanTravel("crossroads", "marsh") {
		t.Fatal("expected exit to marsh")
	}
	if w.CanTravel("crossroads", "nowhere") {
		t.Fatal("should refuse invalid travel")
	}
}
