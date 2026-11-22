package game

import (
	"fmt"
	"strings"

	"github.com/kaoutar/grimhold/internal/dice"
	"github.com/kaoutar/grimhold/internal/open5e"
)

func AssignRole(m *open5e.Monster) FoeRole {
	if m == nil {
		return RoleBruiser
	}
	blob := strings.ToLower(m.Type + " " + m.Name + " " + m.Slug)
	switch {
	case strings.Contains(blob, "undead") && (strings.Contains(blob, "shadow") ||
		strings.Contains(blob, "ghost") || strings.Contains(blob, "wraith") ||
		strings.Contains(blob, "specter")):
		return RoleSkirmisher
	case strings.Contains(blob, "goblin") || strings.Contains(blob, "kobold") ||
		strings.Contains(blob, "rat") || strings.Contains(blob, "spider"):
		return RoleSkirmisher
	case strings.Contains(blob, "cleric") || strings.Contains(blob, "priest") ||
		strings.Contains(blob, "shaman") || strings.Contains(blob, "mage") ||
		strings.Contains(blob, "wizard") || strings.Contains(blob, "warlock"):
		return RoleSupport
	case strings.Contains(blob, "hobgoblin") || strings.Contains(blob, "bugbear") ||
		strings.Contains(blob, "ogre") || strings.Contains(blob, "troll") ||
		strings.Contains(blob, "vampire") || strings.Contains(blob, "mummy") ||
		strings.Contains(blob, "wight") || strings.Contains(blob, "ghoul"):
		return RoleBruiser
	default:
		if m.CR >= 2 {
			return RoleBruiser
		}
		if m.CR < 0.5 {
			return RoleSkirmisher
		}
		return RoleBruiser
	}
}

func ChooseAction(r dice.Roller, foe *Foe, enc *Encounter, playerHP, playerMax int) open5e.MonsterAction {
	actions := combatActions(foe.Monster)
	if len(actions) == 0 {
		return open5e.MonsterAction{Name: "slam", AttackBonus: 2, DamageDice: "1d4", DamageBonus: 1}
	}
	if len(actions) == 1 {
		return actions[0]
	}

	best := actions[0]
	bestScore := -1e9
	for _, a := range actions {
		score := scoreAction(a, foe.Role, playerHP, playerMax)
		score += float64(r.Intn(3))
		if score > bestScore {
			bestScore = score
			best = a
		}
	}
	return best
}

func combatActions(m *open5e.Monster) []open5e.MonsterAction {
	if m == nil {
		return nil
	}
	var out []open5e.MonsterAction
	for _, a := range m.Actions {
		if a.AttackBonus != 0 || a.DamageDice != "" {
			out = append(out, a)
		}
	}
	if len(out) == 0 && len(m.Actions) > 0 {
		return m.Actions[:1]
	}
	return out
}

func scoreAction(a open5e.MonsterAction, role FoeRole, playerHP, playerMax int) float64 {
	avgDmg := estimateAvgDamage(a)
	hitWeight := float64(a.AttackBonus)
	switch role {
	case RoleBruiser:
		return avgDmg*3 + hitWeight
	case RoleSkirmisher:
		score := hitWeight*2 + avgDmg
		if playerMax > 0 && float64(playerHP)/float64(playerMax) < 0.35 {
			score += avgDmg * 2
		}
		return score
	case RoleSupport:
		return hitWeight*1.5 + avgDmg*0.5
	default:
		return avgDmg + hitWeight
	}
}

func estimateAvgDamage(a open5e.MonsterAction) float64 {
	if a.DamageDice == "" {
		return float64(1 + a.DamageBonus)
	}
	expr := a.DamageDice
	if a.DamageBonus != 0 && !strings.ContainsAny(expr, "+-") {
		expr = fmt.Sprintf("%s%+d", expr, a.DamageBonus)
	}
	n, sides, bonus, err := dice.ParseExpr(expr)
	if err != nil || sides <= 0 {
		return float64(1 + a.DamageBonus)
	}
	return float64(n)*float64(sides+1)/2 + float64(bonus)
}

type FoeTurnResult struct {
	Log     string
	Healed  bool
	Skipped bool
}

func RunFoeAI(r dice.Roller, c *Character, enc *Encounter, foeIdx int) FoeTurnResult {
	if foeIdx < 0 || foeIdx >= len(enc.Foes) {
		return FoeTurnResult{Skipped: true}
	}
	foe := &enc.Foes[foeIdx]
	if !foe.Alive() {
		return FoeTurnResult{Skipped: true}
	}

	if foe.Role == RoleSupport {
		if allyIdx, ok := woundedAlly(enc, foeIdx); ok && r.Intn(100) < 45 {
			ally := &enc.Foes[allyIdx]
			heal := 3 + r.Intn(4)
			ally.HP += heal
			if ally.HP > ally.MaxHP {
				ally.HP = ally.MaxHP
			}
			return FoeTurnResult{
				Healed: true,
				Log: fmt.Sprintf("Le %s soutient [%d] %s (+%d HP → %d/%d).",
					foe.Monster.Name, allyIdx+1, ally.Monster.Name, heal, ally.HP, ally.MaxHP),
			}
		}
	}

	if foe.Role == RoleSkirmisher && !foe.IsBoss && foe.MaxHP > 0 {
		ratio := float64(foe.HP) / float64(foe.MaxHP)
		if ratio < 0.25 && r.Intn(100) < 35 {
			foe.HP = 0
			foe.Fled = true
			foe.DeathNoted = true
			return FoeTurnResult{
				Log: fmt.Sprintf("Le %s décroche et s'enfuit dans la brume !", foe.Monster.Name),
			}
		}
	}

	atk := ChooseAction(r, foe, enc, c.HP, c.MaxHP)
	res := ResolveAttack(r, c, foe, atk, enc.PlayerDefending)
	if res.Hit {
		return FoeTurnResult{Log: res.HitMsg}
	}
	return FoeTurnResult{Log: res.MissMsg}
}

func woundedAlly(enc *Encounter, selfIdx int) (int, bool) {
	best := -1
	bestRatio := 1.0
	for i := range enc.Foes {
		if i == selfIdx || !enc.Foes[i].Alive() {
			continue
		}
		f := &enc.Foes[i]
		if f.MaxHP <= 0 {
			continue
		}
		ratio := float64(f.HP) / float64(f.MaxHP)
		if ratio < 0.5 && ratio < bestRatio {
			bestRatio = ratio
			best = i
		}
	}
	if best < 0 {
		return -1, false
	}
	return best, true
}

func RunAllFoeTurns(r dice.Roller, c *Character, enc *Encounter) string {
	var b strings.Builder
	indices := append([]int(nil), enc.AliveFoes()...)
	for _, idx := range indices {
		if !enc.Foes[idx].Alive() {
			continue
		}
		turn := RunFoeAI(r, c, enc, idx)
		if turn.Skipped || turn.Log == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(turn.Log)
		if c.HP <= 0 {
			break
		}
	}
	return b.String()
}
