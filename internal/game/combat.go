package game

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kaoutar/grimhold/internal/dice"
	"github.com/kaoutar/grimhold/internal/open5e"
)

type FoeRole string

const (
	RoleBruiser    FoeRole = "bruiser"
	RoleSkirmisher FoeRole = "skirmisher"
	RoleSupport    FoeRole = "support"
)

type Foe struct {
	Monster        *open5e.Monster
	HP             int
	MaxHP          int
	Role           FoeRole
	IsBoss         bool
	Phase          int
	PhaseTriggered bool
	Fled           bool
	DeathNoted     bool
}

func (f *Foe) Alive() bool {
	return f != nil && f.Monster != nil && f.HP > 0
}

type Encounter struct {
	Foes            []Foe
	Active          bool
	Focus           int
	PlayerDefending bool
	IsBossFight     bool
}

func StartEncounter(m *open5e.Monster) Encounter {
	return StartPack([]*open5e.Monster{m}, nil)
}

func StartPack(monsters []*open5e.Monster, roles []FoeRole) Encounter {
	foes := make([]Foe, 0, len(monsters))
	for i, m := range monsters {
		if m == nil {
			continue
		}
		role := RoleBruiser
		if i < len(roles) && roles[i] != "" {
			role = roles[i]
		} else {
			role = AssignRole(m)
		}
		foes = append(foes, Foe{
			Monster: m,
			HP:      m.HitPoints,
			MaxHP:   m.HitPoints,
			Role:    role,
		})
	}
	return Encounter{
		Foes:   foes,
		Active: len(foes) > 0,
		Focus:  0,
	}
}

func (e *Encounter) AliveFoes() []int {
	var out []int
	for i := range e.Foes {
		if e.Foes[i].Alive() {
			out = append(out, i)
		}
	}
	return out
}

func (e *Encounter) AllDown() bool {
	return len(e.AliveFoes()) == 0
}

func (e *Encounter) Target() (*Foe, int, error) {
	if e == nil || !e.Active {
		return nil, -1, fmt.Errorf("pas de rencontre active")
	}
	if e.Focus >= 0 && e.Focus < len(e.Foes) && e.Foes[e.Focus].Alive() {
		return &e.Foes[e.Focus], e.Focus, nil
	}
	alive := e.AliveFoes()
	if len(alive) == 0 {
		return nil, -1, fmt.Errorf("aucun ennemi vivant")
	}
	e.Focus = alive[0]
	return &e.Foes[e.Focus], e.Focus, nil
}

func (e *Encounter) SetFocus(spec string) error {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		_, _, err := e.Target()
		return err
	}
	if n, err := strconv.Atoi(spec); err == nil {
		idx := n - 1
		if idx < 0 || idx >= len(e.Foes) || !e.Foes[idx].Alive() {
			return fmt.Errorf("cible invalide %q — utilise 1..%d", spec, len(e.Foes))
		}
		e.Focus = idx
		return nil
	}
	lower := strings.ToLower(spec)
	for i := range e.Foes {
		if !e.Foes[i].Alive() {
			continue
		}
		name := strings.ToLower(e.Foes[i].Monster.Name)
		slug := strings.ToLower(e.Foes[i].Monster.Slug)
		if strings.Contains(name, lower) || strings.Contains(slug, lower) {
			e.Focus = i
			return nil
		}
	}
	return fmt.Errorf("aucun ennemi vivant nommé %q", spec)
}

func (e *Encounter) SummaryLine() string {
	if e == nil || !e.Active {
		return ""
	}
	parts := make([]string, 0, len(e.Foes))
	for i := range e.Foes {
		f := &e.Foes[i]
		if !f.Alive() {
			tag := "mort"
			if f.Fled {
				tag = "fui"
			}
			parts = append(parts, fmt.Sprintf("[%d] %s (%s)", i+1, f.Monster.Name, tag))
			continue
		}
		mark := " "
		if i == e.Focus {
			mark = "*"
		}
		parts = append(parts, fmt.Sprintf("%s[%d] %s HP %d/%d (%s)",
			mark, i+1, f.Monster.Name, f.HP, f.MaxHP, f.Role))
	}
	return strings.Join(parts, "\n")
}

type AttackResult struct {
	Hit     bool
	Roll    int
	Total   int
	Damage  int
	MissMsg string
	HitMsg  string
}

func PlayerAC(c Character, defending bool) int {
	ac := 12 + c.DexterityMod
	if defending {
		ac += 2
	}
	return ac
}

func PlayerAttack(r dice.Roller, c Character, enc *Encounter) AttackResult {
	foe, idx, err := enc.Target()
	if err != nil {
		return AttackResult{MissMsg: err.Error()}
	}
	roll := dice.D20(r)
	crit := roll == 20
	total := roll + c.AttackBonus
	ac := foe.Monster.ArmorClass
	if !crit && total < ac {
		return AttackResult{
			Hit:     false,
			Roll:    roll,
			Total:   total,
			MissMsg: fmt.Sprintf("Tu rates [%d] %s (%d+%d=%d vs AC %d).", idx+1, foe.Monster.Name, roll, c.AttackBonus, total, ac),
		}
	}
	dmg, err := dice.RollExpr(r, c.WeaponDice)
	if err != nil || dmg < 1 {
		dmg = 1
	}
	if crit {
		extra, _ := dice.RollExpr(r, c.WeaponDice)
		if extra < 1 {
			extra = 1
		}
		dmg += extra
	}
	foe.HP -= dmg
	if foe.HP < 0 {
		foe.HP = 0
	}
	critTag := ""
	if crit {
		critTag = " CRITIQUE !"
	}
	return AttackResult{
		Hit:    true,
		Roll:   roll,
		Total:  total,
		Damage: dmg,
		HitMsg: fmt.Sprintf("Tu frappes [%d] %s avec ton %s (%d+%d=%d vs AC %d)%s pour %d dégâts ! HP %d/%d.",
			idx+1, foe.Monster.Name, c.WeaponName, roll, c.AttackBonus, total, ac, critTag, dmg, foe.HP, foe.MaxHP),
	}
}

func ResolveAttack(r dice.Roller, c *Character, foe *Foe, atk open5e.MonsterAction, defending bool) AttackResult {
	return strike(r, c, foe, atk, defending, false)
}

func FreeHit(r dice.Roller, c *Character, foe *Foe, atk open5e.MonsterAction) AttackResult {
	return strike(r, c, foe, atk, false, true)
}

func strike(r dice.Roller, c *Character, foe *Foe, atk open5e.MonsterAction, defending, autoHit bool) AttackResult {
	name := foe.Monster.Name
	playerAC := PlayerAC(*c, defending)
	bonus := atk.AttackBonus
	if bonus == 0 && atk.DamageDice == "" {
		bonus = 2
	}
	if foe.IsBoss && foe.Phase >= 1 {
		bonus += 2
	}
	atkName := atk.Name
	if atkName == "" {
		atkName = "slam"
	}
	roll := 0
	total := 0
	if !autoHit {
		roll = dice.D20(r)
		total = roll + bonus
		if total < playerAC {
			return AttackResult{
				Hit:     false,
				Roll:    roll,
				Total:   total,
				MissMsg: fmt.Sprintf("Le %s rate avec %s (%d+%d=%d vs AC %d).", name, atkName, roll, bonus, total, playerAC),
			}
		}
	}
	expr := atk.DamageDice
	if expr == "" {
		expr = "1d4"
	}
	if atk.DamageBonus != 0 && !strings.ContainsAny(expr, "+-") {
		expr = fmt.Sprintf("%s%+d", expr, atk.DamageBonus)
	}
	dmg, err := dice.RollExpr(r, expr)
	if err != nil || dmg < 1 {
		dmg = 1 + atk.DamageBonus
		if dmg < 1 {
			dmg = 1
		}
	}
	if foe.IsBoss && foe.Phase >= 1 {
		dmg += 2
	}
	c.HP -= dmg
	if c.HP < 0 {
		c.HP = 0
	}
	hitMsg := fmt.Sprintf("Le %s touche avec %s pour %d ! Tes HP %d/%d.", name, atkName, dmg, c.HP, c.MaxHP)
	if autoHit {
		hitMsg = fmt.Sprintf("Le %s te porte un coup gratuit avec %s pour %d ! Tes HP %d/%d.", name, atkName, dmg, c.HP, c.MaxHP)
	}
	return AttackResult{
		Hit:    true,
		Roll:   roll,
		Total:  total,
		Damage: dmg,
		HitMsg: hitMsg,
	}
}

func FoeAttack(r dice.Roller, c *Character, enc *Encounter) AttackResult {
	alive := enc.AliveFoes()
	if len(alive) == 0 {
		return AttackResult{MissMsg: "Aucun ennemi n'attaque."}
	}
	foe := &enc.Foes[alive[0]]
	atk, ok := foe.Monster.PrimaryAttack()
	if !ok {
		atk = open5e.MonsterAction{Name: "slam", AttackBonus: 2, DamageDice: "1d4", DamageBonus: 1}
	}
	return ResolveAttack(r, c, foe, atk, enc.PlayerDefending)
}

func FleeAttempt(r dice.Roller, c *Character, enc *Encounter) (escaped bool, log string) {
	alive := enc.AliveFoes()
	dc := 10 + len(alive)*2
	if enc.IsBossFight {
		dc += 4
	}
	roll := dice.D20(r)
	total := roll + c.DexterityMod
	if total >= dc {
		enc.Active = false
		return true, fmt.Sprintf("Tu romps le combat (%d+%d=%d vs DC %d) et t'enfuis !", roll, c.DexterityMod, total, dc)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Tu échoues à fuir (%d+%d=%d vs DC %d). Un ennemi en profite !\n", roll, c.DexterityMod, total, dc)
	if len(alive) > 0 {
		foe := &enc.Foes[alive[r.Intn(len(alive))]]
		atk, ok := foe.Monster.PrimaryAttack()
		if !ok {
			atk = open5e.MonsterAction{Name: "slam", AttackBonus: 2, DamageDice: "1d4", DamageBonus: 1}
		}
		res := FreeHit(r, c, foe, atk)
		b.WriteString(res.HitMsg)
	}
	return false, b.String()
}

func XPFromCR(cr float64) int {
	switch {
	case cr <= 0:
		return 10
	case cr < 0.5:
		return 25
	case cr < 1:
		return 50
	case cr < 2:
		return 100
	case cr < 3:
		return 200
	case cr < 5:
		return 450
	default:
		return int(cr * 200)
	}
}

func GoldFromCR(r dice.Roller, cr float64) int {
	base := 1 + int(cr*3)
	if base < 1 {
		base = 1
	}
	return base + r.Intn(3)
}
