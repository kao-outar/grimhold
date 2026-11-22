package world

import (
	"encoding/json"
	"fmt"
	"os"
)

type Region struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Danger      int      `json:"danger"`
	Foes        []string `json:"foes"`
	Exits       []string `json:"exits"`
	QuestHint   string   `json:"quest_hint,omitempty"`
	MinLevel    int      `json:"min_level,omitempty"`
	Boss        string   `json:"boss,omitempty"`
}

type World struct {
	Start   string            `json:"start"`
	Regions map[string]Region `json:"-"`
	raw     worldFile
}

type worldFile struct {
	Start   string   `json:"start"`
	Regions []Region `json:"regions"`
}

func Load(path string) (*World, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read world: %w", err)
	}
	var raw worldFile
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode world: %w", err)
	}
	if raw.Start == "" {
		return nil, fmt.Errorf("world: missing start region")
	}
	w := &World{
		Start:   raw.Start,
		Regions: make(map[string]Region, len(raw.Regions)),
		raw:     raw,
	}
	for _, r := range raw.Regions {
		if r.ID == "" {
			return nil, fmt.Errorf("world: region missing id")
		}
		w.Regions[r.ID] = r
	}
	if _, ok := w.Regions[raw.Start]; !ok {
		return nil, fmt.Errorf("world: start %q not found", raw.Start)
	}
	return w, nil
}

func (w *World) Get(id string) (Region, bool) {
	r, ok := w.Regions[id]
	return r, ok
}

func (w *World) CanTravel(from, to string) bool {
	r, ok := w.Regions[from]
	if !ok {
		return false
	}
	for _, e := range r.Exits {
		if e == to {
			return true
		}
	}
	return false
}

func (w *World) ExitByIndex(from string, index int) (string, error) {
	r, ok := w.Regions[from]
	if !ok {
		return "", fmt.Errorf("unknown region %q", from)
	}
	if index < 1 || index > len(r.Exits) {
		return "", fmt.Errorf("exit number must be 1..%d", len(r.Exits))
	}
	return r.Exits[index-1], nil
}

func (w *World) All() []Region {
	return append([]Region(nil), w.raw.Regions...)
}

func DangerHint(danger int) string {
	switch {
	case danger <= 0:
		return "relativement calme — pour l'instant"
	case danger == 1:
		return "inquiétant ; des bruits dans la brume"
	case danger == 2:
		return "hostile ; quelque chose observe"
	case danger == 3:
		return "péril grave ; les morts ne dorment pas"
	default:
		return "catastrophique ; rebrousse chemin"
	}
}
