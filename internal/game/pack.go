package game

import (
	"fmt"
	"strings"

	"github.com/kaoutar/grimhold/internal/dice"
	"github.com/kaoutar/grimhold/internal/open5e"
)

func PackSize(r dice.Roller, danger, level int) int {
	base := 1
	if danger >= 1 {
		base = 1 + r.Intn(2)
	}
	if danger >= 2 {
		base = 1 + r.Intn(3)
	}
	if danger >= 4 {
		base = 2 + r.Intn(2)
	}
	if level >= 3 && base < 2 && danger >= 1 {
		base = 2
	}
	if base < 1 {
		base = 1
	}
	if base > 3 {
		base = 3
	}
	return base
}

// pickFoeSlug tire 1+danger fois. Les listes de world.json vont du plus faible
// au plus fort : on garde l'index le plus élevé, donc le pire ennemi.
func pickFoeSlug(r dice.Roller, foes []string, danger int) string {
	if len(foes) == 0 {
		return ""
	}
	rolls := 1 + danger
	if rolls < 1 {
		rolls = 1
	}
	bestIdx := -1
	pick := foes[0]
	for i := 0; i < rolls; i++ {
		candidate := dice.Pick(r, foes)
		idx := foeIndex(foes, candidate)
		if idx > bestIdx {
			bestIdx = idx
			pick = candidate
		}
	}
	return pick
}

func foeIndex(foes []string, slug string) int {
	for i, f := range foes {
		if f == slug {
			return i
		}
	}
	return 0
}

type FetchMonster interface {
	GetMonster(slug string) (*open5e.Monster, error)
}

func BuildPack(r dice.Roller, client FetchMonster, foes []string, danger, level int) ([]*open5e.Monster, []FoeRole, error) {
	if len(foes) == 0 {
		return nil, nil, fmt.Errorf("aucun foe dans la région")
	}
	n := PackSize(r, danger, level)
	monsters := make([]*open5e.Monster, 0, n)
	roles := make([]FoeRole, 0, n)
	usedSupport := false
	for i := 0; i < n; i++ {
		slug := pickFoeSlug(r, foes, danger)
		m, err := client.GetMonster(slug)
		if err != nil {
			return nil, nil, err
		}
		role := AssignRole(m)
		if i == n-1 && n >= 3 && !usedSupport && role != RoleSupport {
			role = RoleSupport
		}
		if role == RoleSupport {
			usedSupport = true
		}
		if i == 1 && n >= 2 && role == RoleBruiser {
			role = RoleSkirmisher
		}
		monsters = append(monsters, m)
		roles = append(roles, role)
	}
	return monsters, roles, nil
}

func FormatPackIntro(enc *Encounter, place string) string {
	alive := enc.AliveFoes()
	if len(alive) == 0 {
		return "Rien ne bouge."
	}
	if len(alive) == 1 {
		f := &enc.Foes[alive[0]]
		return fmt.Sprintf("Un(e) %s surgit de %s ! (AC %d, HP %d, CR %s, rôle %s)\nCibles: fight [n] · defend · potion · flee",
			f.Monster.Name, place, f.Monster.ArmorClass, f.HP, f.Monster.ChallengeRating, f.Role)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Une meute surgit de %s (%d ennemis) !\n", place, len(alive))
	b.WriteString(enc.SummaryLine())
	b.WriteString("\nCibles: fight [n|nom] · defend · potion · flee")
	return b.String()
}

func BuildBossEncounter(m *open5e.Monster) Encounter {
	enc := StartPack([]*open5e.Monster{m}, []FoeRole{RoleBruiser})
	if len(enc.Foes) > 0 {
		enc.Foes[0].HP = m.HitPoints + 15
		enc.Foes[0].MaxHP = enc.Foes[0].HP
		enc.Foes[0].IsBoss = true
		enc.Foes[0].Role = RoleBruiser
	}
	enc.IsBossFight = true
	return enc
}

func MaybeTriggerBossPhase(r dice.Roller, enc *Encounter, client FetchMonster) string {
	if enc == nil || !enc.IsBossFight {
		return ""
	}
	var logs []string
	for i := range enc.Foes {
		f := &enc.Foes[i]
		if !f.IsBoss || f.PhaseTriggered || !f.Alive() || f.MaxHP <= 0 {
			continue
		}
		if float64(f.HP)/float64(f.MaxHP) > 0.5 {
			continue
		}
		f.PhaseTriggered = true
		f.Phase = 1
		logs = append(logs, fmt.Sprintf("✦ Le %s hurle — phase 2 ! Ses coups s'intensifient.", f.Monster.Name))

		if len(enc.AliveFoes()) >= 3 || client == nil {
			continue
		}
		slug := "vampire-spawn"
		summon, err := client.GetMonster(slug)
		if err != nil {
			summon, err = client.GetMonster("shadow")
		}
		if err != nil || summon == nil {
			continue
		}
		enc.Foes = append(enc.Foes, Foe{
			Monster: summon,
			HP:      summon.HitPoints,
			MaxHP:   summon.HitPoints,
			Role:    RoleSkirmisher,
		})
		logs = append(logs, fmt.Sprintf("Un(e) %s émerge des ombres pour le servir !", summon.Name))
	}
	return strings.Join(logs, "\n")
}
