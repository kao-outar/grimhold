package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/kaoutar/grimhold/internal/game"
)

const defaultWidth = 78

type Frame struct {
	State   *game.State
	Log     []string
	Suggest []string
}

func Render(w io.Writer, f Frame) {
	ClearHome(w)
	width := defaultWidth
	s := f.State
	c := s.Character

	locName := s.Location
	if reg, err := s.CurrentRegion(); err == nil {
		locName = reg.Name
	}

	questLine := "—"
	if s.Story != nil {
		questLine = s.Story.ObjectiveLine(s.Quest)
	}

	header := fmt.Sprintf("HP %s  Lv%d  Or:%d  Pot:%d", HPBar(c.HP, c.MaxHP, 10), c.Level, c.Gold, c.Potions)
	fmt.Fprintln(w, topLine(width, "GRIMHOLD — "+s.StoryTitle()))
	fmt.Fprintln(w, row(width, header))
	fmt.Fprintln(w, row(width, "Lieu: "+locName))
	fmt.Fprintln(w, row(width, "Quête: "+trim(questLine, width-12)))
	if tip := tutorialTip(s); tip != "" {
		fmt.Fprintln(w, row(width, "Tuto: "+trim(tip, width-12)))
	}

	fmt.Fprintln(w, hline(width, "Carte / sorties"))
	mapLines := mapPanel(s)
	for _, line := range mapLines {
		fmt.Fprintln(w, row(width, line))
	}

	if s.Encounter != nil && s.Encounter.Active {
		fmt.Fprintln(w, hline(width, "Combat"))
		enc := s.Encounter
		for i := range enc.Foes {
			f := &enc.Foes[i]
			mark := " "
			if i == enc.Focus && f.Alive() {
				mark = "*"
			}
			status := HPBar(f.HP, f.MaxHP, 10)
			if !f.Alive() {
				if f.Fled {
					status = "fui"
				} else {
					status = "mort"
				}
			}
			line := fmt.Sprintf("%s[%d] %s %s (%s)", mark, i+1, f.Monster.Name, status, f.Role)
			fmt.Fprintln(w, row(width, line))
		}
	}

	fmt.Fprintln(w, hline(width, "Récit"))
	logLines := f.Log
	if len(logLines) == 0 {
		logLines = []string{"La brume attend. Tape 'talk' aux carrefours pour commencer."}
	}

	wrapped := wrapLines(logLines, width-6)
	if len(wrapped) > 10 {
		wrapped = wrapped[len(wrapped)-10:]
	}
	for _, line := range wrapped {
		fmt.Fprintln(w, row(width, line))
	}

	fmt.Fprintln(w, hline(width, "Actions suggérées"))
	sug := f.Suggest
	if len(sug) == 0 {
		sug = Suggest(s)
	}
	fmt.Fprintln(w, row(width, strings.Join(sug, "   ")))
	fmt.Fprintln(w, bottomLine(width))
	fmt.Fprint(w, "grimhold > ")
}

func mapPanel(s *game.State) []string {
	reg, err := s.CurrentRegion()
	if err != nil {
		return []string{s.Location}
	}
	lines := []string{fmt.Sprintf("* %s (%s)", reg.ID, DangerLabel(reg.Danger))}
	if reg.QuestHint != "" {
		lines = append(lines, "  hint: "+trim(reg.QuestHint, 60))
	}
	for i, exit := range reg.Exits {
		mark := " "
		dest, ok := s.World.Get(exit)
		name := exit
		if ok {
			name = fmt.Sprintf("%s — %s", exit, dest.Name)
		}
		lines = append(lines, fmt.Sprintf("%s[%d] %s", mark, i+1, name))
	}
	return lines
}

func Suggest(s *game.State) []string {
	if s.Quest.Ending != "" {
		return []string{"[j] journal", "[s] save", "[q] quit"}
	}
	if s.Encounter != nil && s.Encounter.Active {
		return []string{"[f] fight", "[d] defend", "[p] potion", "[flee]", "[c] character"}
	}
	if s.Quest.Has("boss_felled") && s.Quest.Ending == "" {
		return []string{"choose mercy", "choose wrath", "[j] journal"}
	}
	if !s.Quest.Has("met_herald") {
		return []string{"[t] talk ← commence", "[l] look", "[j] journal", "[h] help"}
	}
	if s.Story != nil {
		if act := s.Story.CurrentAct(s.Quest); act != nil && act.Region != "" && s.Location != act.Region {
			if n, ok := exitIndex(s, act.Region); ok {
				return []string{
					fmt.Sprintf("[%d] → %s", n, act.Region),
					"[m] map",
					"[j] journal",
					"[l] look",
				}
			}
			return []string{"travel " + act.Region, "[m] map", "[j] journal"}
		}
		if act := s.Story.CurrentAct(s.Quest); act != nil && (act.Region == "" || act.Region == s.Location) {
			if act.KillsNeeded > 0 || act.ScriptedExplore || act.BossSlug != "" {
				return []string{"[e] explore ← quête", "[l] look", "[j] journal", "[r] rest"}
			}
		}
	}
	out := []string{"[l] look", "[e] explore", "[t] talk", "[j] journal"}
	if reg, err := s.CurrentRegion(); err == nil && len(reg.Exits) > 0 {
		out = append([]string{fmt.Sprintf("[1-%d] travel", len(reg.Exits))}, out...)
	}
	out = append(out, "[r] rest", "[s] save")
	return out
}

func wrapLines(lines []string, width int) []string {
	var out []string
	for _, line := range lines {
		for _, part := range strings.Split(line, "\n") {
			part = strings.TrimRight(part, "\r")
			if part == "" {
				out = append(out, "")
				continue
			}
			runes := []rune(part)
			for len(runes) > width {
				out = append(out, string(runes[:width]))
				runes = runes[width:]
			}
			out = append(out, string(runes))
		}
	}
	return out
}
