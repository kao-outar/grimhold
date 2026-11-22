package quest

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Story struct {
	Title       string                 `json:"title"`
	Prologue    string                 `json:"prologue"`
	Acts        []Act                  `json:"acts"`
	RegionFlags map[string]RegionEvent `json:"region_flags"`
	Endings     map[string]Ending      `json:"endings"`
}

type Act struct {
	ID              int      `json:"id"`
	Title           string   `json:"title"`
	Objective       string   `json:"objective"`
	Region          string   `json:"region"`
	GrantFlag       string   `json:"grant_flag"`
	OnGrant         string   `json:"on_grant"`
	KillsNeeded     int      `json:"kills_needed"`
	ScriptedExplore bool     `json:"scripted_explore"`
	RequiresFlags   []string `json:"requires_flags"`
	BossSlug        string   `json:"boss_slug"`
	MinLevel        int      `json:"min_level"`
}

type RegionEvent struct {
	Flag        string `json:"flag"`
	KillsNeeded int    `json:"kills_needed"`
	OnGrant     string `json:"on_grant"`
}

type Ending struct {
	Flag  string `json:"flag"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

type Progress struct {
	Flags       map[string]bool `json:"flags"`
	RegionKills map[string]int  `json:"region_kills"`
	Stage       int             `json:"stage"`
	Ending      string          `json:"ending"`
}

func NewProgress() Progress {
	return Progress{
		Flags:       make(map[string]bool),
		RegionKills: make(map[string]int),
		Stage:       0,
	}
}

func LoadStory(path string) (*Story, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read story: %w", err)
	}
	var s Story
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("decode story: %w", err)
	}
	if s.Title == "" {
		return nil, fmt.Errorf("story: missing title")
	}
	return &s, nil
}

func (p Progress) Has(flag string) bool {
	return p.Flags[flag]
}

func (p *Progress) SetFlag(flag string) bool {
	if flag == "" || p.Flags[flag] {
		return false
	}
	if p.Flags == nil {
		p.Flags = make(map[string]bool)
	}
	p.Flags[flag] = true
	return true
}

func (s *Story) CurrentAct(p Progress) *Act {
	if p.Ending != "" {
		return nil
	}
	for i := range s.Acts {
		a := &s.Acts[i]
		if !p.Has(a.GrantFlag) {
			return a
		}
	}
	return nil
}

func (s *Story) ObjectiveLine(p Progress) string {
	if p.Ending != "" {
		if e, ok := s.Endings[p.Ending]; ok {
			return e.Title
		}
		return "Quête terminée"
	}
	a := s.CurrentAct(p)
	if a == nil {
		return "Choisis ton destin: choose mercy | choose wrath"
	}
	done := 0
	for _, act := range s.Acts {
		if p.Has(act.GrantFlag) {
			done++
		}
	}
	return fmt.Sprintf("%s (%d/%d)", a.Objective, done, len(s.Acts))
}

func (s *Story) Journal(p Progress) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n%s\n\n", s.Title, s.Prologue)
	b.WriteString("Journal de quête:\n")
	for _, a := range s.Acts {
		mark := "[ ]"
		if p.Has(a.GrantFlag) {
			mark = "[x]"
		}
		fmt.Fprintf(&b, "  %s %s\n     → %s\n", mark, a.Title, a.Objective)
	}
	if len(s.RegionFlags) > 0 {
		b.WriteString("\nFragments:\n")
		ids := make([]string, 0, len(s.RegionFlags))
		for id := range s.RegionFlags {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			ev := s.RegionFlags[id]
			mark := "[ ]"
			if p.Has(ev.Flag) {
				mark = "[x]"
			}
			fmt.Fprintf(&b, "  %s %s (%s)\n", mark, ev.Flag, id)
		}
	}
	if p.Ending != "" {
		if e, ok := s.Endings[p.Ending]; ok {
			fmt.Fprintf(&b, "\n%s\n%s\n", e.Title, e.Text)
		}
	} else if p.Has("boss_felled") {
		b.WriteString("\nLe Wraith est à terre. Tape: choose mercy   ou   choose wrath\n")
	}
	b.WriteString("\nObjectif actuel: ")
	b.WriteString(s.ObjectiveLine(p))
	return b.String()
}

func (s *Story) Talk(p *Progress, location string) (string, error) {
	if location != "crossroads" {
		return "", fmt.Errorf("personne à qui parler ici — retourne aux carrefours")
	}
	act := s.Acts[0]
	if p.Has(act.GrantFlag) {
		return "Le héraut de pierre reste silencieux. Tu as déjà entendu son avertissement.\n" + s.ObjectiveLine(*p), nil
	}
	p.SetFlag(act.GrantFlag)
	p.Stage = 1
	return act.OnGrant + "\n\nNouvel objectif: " + s.ObjectiveLine(*p), nil
}

func (s *Story) OnVictory(p *Progress, location, monsterSlug string, playerLevel int) string {
	if p.RegionKills == nil {
		p.RegionKills = make(map[string]int)
	}
	p.RegionKills[location]++

	var parts []string

	if ev, ok := s.RegionFlags[location]; ok && !p.Has(ev.Flag) {
		need := ev.KillsNeeded
		if need < 1 {
			need = 1
		}
		if p.RegionKills[location] >= need {
			if p.SetFlag(ev.Flag) {
				parts = append(parts, ev.OnGrant)
			}
		}
	}

	for i := range s.Acts {
		a := &s.Acts[i]
		if p.Has(a.GrantFlag) {
			continue
		}
		if a.ID == 0 {
			continue
		}
		if !p.Has("met_herald") {
			continue
		}

		if len(a.RequiresFlags) > 0 {
			ok := true
			for _, f := range a.RequiresFlags {
				if !p.Has(f) {
					ok = false
					break
				}
			}
			if ok && p.SetFlag(a.GrantFlag) {
				p.Stage = a.ID + 1
				parts = append(parts, a.OnGrant)
			}
			continue
		}
		if a.Region != "" && a.Region != location {
			continue
		}
		if a.BossSlug != "" {
			if monsterSlug == a.BossSlug || strings.Contains(strings.ToLower(monsterSlug), a.BossSlug) {
				if playerLevel < a.MinLevel && a.MinLevel > 0 {
					continue
				}
				if p.SetFlag(a.GrantFlag) {
					p.Stage = a.ID + 1
					parts = append(parts, a.OnGrant)
				}
			}
			continue
		}
		if a.ScriptedExplore {

			continue
		}
		need := a.KillsNeeded
		if need < 1 {
			need = 1
		}
		if a.Region != "" && p.RegionKills[a.Region] >= need {
			if p.SetFlag(a.GrantFlag) {
				p.Stage = a.ID + 1
				parts = append(parts, a.OnGrant)
			}
		}
	}

	for i := range s.Acts {
		a := &s.Acts[i]
		if len(a.RequiresFlags) == 0 || p.Has(a.GrantFlag) {
			continue
		}
		ok := true
		for _, f := range a.RequiresFlags {
			if !p.Has(f) {
				ok = false
				break
			}
		}
		if ok && p.SetFlag(a.GrantFlag) {
			p.Stage = a.ID + 1
			parts = append(parts, a.OnGrant)
		}
	}

	return strings.Join(parts, "\n")
}

func (s *Story) OnScriptedExplore(p *Progress, location string) (triggered bool, msg string) {
	for i := range s.Acts {
		a := &s.Acts[i]
		if !a.ScriptedExplore || p.Has(a.GrantFlag) {
			continue
		}
		if a.Region != location {
			continue
		}

		if !p.Has("met_herald") {
			return false, ""
		}
		if a.ID >= 2 && !p.Has("shard_marsh") {
			return false, ""
		}
		if p.SetFlag(a.GrantFlag) {
			p.Stage = a.ID + 1
			return true, a.OnGrant
		}
	}
	return false, ""
}

func (s *Story) ChooseEnding(p *Progress, choice string) (string, error) {
	choice = strings.ToLower(strings.TrimSpace(choice))
	if !p.Has("boss_felled") {
		return "", fmt.Errorf("le choix final n'est pas encore ouvert — vaincs d'abord le Wraith de la Flèche")
	}
	if p.Ending != "" {
		return "", fmt.Errorf("tu as déjà choisi ta fin (%s)", p.Ending)
	}
	e, ok := s.Endings[choice]
	if !ok {
		return "", fmt.Errorf("usage: choose mercy | choose wrath")
	}
	p.Ending = choice
	p.SetFlag(e.Flag)
	return fmt.Sprintf("%s\n\n%s", e.Title, e.Text), nil
}

func (s *Story) SpireUnlocked(p Progress, level int) (bool, string) {
	if !p.Has("key_complete") {
		return false, "la Flèche Lugubre refuse ton passage — il te faut la Clé de Brume complète"
	}
	min := 2
	for _, a := range s.Acts {
		if a.BossSlug != "" && a.MinLevel > 0 {
			min = a.MinLevel
			break
		}
	}
	if level < min {
		return false, fmt.Sprintf("la Flèche exige le niveau %d (tu es niveau %d)", min, level)
	}
	return true, ""
}
