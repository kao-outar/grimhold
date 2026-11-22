package game

import (
	"math/rand"
	"testing"

	"github.com/kaoutar/grimhold/internal/dice"
	"github.com/kaoutar/grimhold/internal/open5e"
)

func TestPlayerAttackSeededHit(t *testing.T) {
	r := dice.Rand{R: rand.New(rand.NewSource(1))}
	c := NewAdventurer("Test")
	m := &open5e.Monster{
		Name:       "Goblin",
		ArmorClass: 10,
		HitPoints:  7,
		Actions: []open5e.MonsterAction{
			{Name: "Scimitar", AttackBonus: 4, DamageDice: "1d6", DamageBonus: 2},
		},
	}
	enc := StartEncounter(m)

	var sawHit, sawMiss bool
	for i := 0; i < 40; i++ {
		enc.Foes[0].HP = enc.Foes[0].MaxHP
		res := PlayerAttack(r, c, &enc)
		if res.Hit {
			sawHit = true
			if res.Damage < 1 {
				t.Fatalf("hit with non-positive damage %d", res.Damage)
			}
		} else {
			sawMiss = true
		}
	}
	if !sawHit {
		t.Fatal("expected at least one hit with seeded rolls")
	}
	_ = sawMiss
}

func TestFleeFailureAllowsFreeHit(t *testing.T) {
	r := dice.Rand{R: rand.New(rand.NewSource(99))}
	c := NewAdventurer("Test")
	c.HP = 20
	m := &open5e.Monster{
		Name:       "Skeleton",
		ArmorClass: 13,
		HitPoints:  13,
		Actions: []open5e.MonsterAction{
			{Name: "Shortsword", AttackBonus: 4, DamageDice: "1d6", DamageBonus: 2},
		},
	}
	enc := StartEncounter(m)

	escapedOnce := false
	failedOnce := false
	for i := 0; i < 50; i++ {
		c.HP = 20
		enc.Active = true
		enc.Foes[0].HP = enc.Foes[0].MaxHP
		escaped, log := FleeAttempt(r, &c, &enc)
		if escaped {
			escapedOnce = true
			if enc.Active {
				t.Fatal("encounter should be inactive after escape")
			}
		} else {
			failedOnce = true
			if log == "" {
				t.Fatal("expected flee failure log")
			}
			if c.HP >= 20 {
				t.Fatalf("failed flee should land a free hit, hp=%d log=%q", c.HP, log)
			}
		}
	}
	if !escapedOnce || !failedOnce {
		t.Fatalf("expected both escape and fail, got escape=%v fail=%v", escapedOnce, failedOnce)
	}
}

func TestXPFromCR(t *testing.T) {
	if XPFromCR(0.25) != 25 {
		t.Fatalf("CR 1/4 XP = %d", XPFromCR(0.25))
	}
	if XPFromCR(1) != 100 {
		t.Fatalf("CR 1 XP = %d", XPFromCR(1))
	}
}

func TestMultiFoeFocusAndAllDown(t *testing.T) {
	g := &open5e.Monster{Name: "Goblin", Slug: "goblin", ArmorClass: 10, HitPoints: 5, CR: 0.25}
	s := &open5e.Monster{Name: "Skeleton", Slug: "skeleton", ArmorClass: 13, HitPoints: 8, CR: 0.25}
	enc := StartPack([]*open5e.Monster{g, s}, []FoeRole{RoleSkirmisher, RoleBruiser})
	if len(enc.AliveFoes()) != 2 {
		t.Fatalf("alive=%d", len(enc.AliveFoes()))
	}
	if err := enc.SetFocus("2"); err != nil {
		t.Fatal(err)
	}
	if enc.Focus != 1 {
		t.Fatalf("focus=%d", enc.Focus)
	}
	if err := enc.SetFocus("goblin"); err != nil {
		t.Fatal(err)
	}
	if enc.Focus != 0 {
		t.Fatalf("focus goblin=%d", enc.Focus)
	}
	enc.Foes[0].HP = 0
	enc.Foes[1].HP = 0
	if !enc.AllDown() {
		t.Fatal("expected all down")
	}
}

func TestPlayerACDefend(t *testing.T) {
	c := NewAdventurer("T")
	base := PlayerAC(c, false)
	def := PlayerAC(c, true)
	if def != base+2 {
		t.Fatalf("defend AC %d want %d", def, base+2)
	}
}

func TestDrinkPotion(t *testing.T) {
	r := dice.Rand{R: rand.New(rand.NewSource(2))}
	c := NewAdventurer("T")
	c.HP = 5
	before := c.Potions
	healed, err := c.DrinkPotion(r)
	if err != nil {
		t.Fatal(err)
	}
	if healed < 1 {
		t.Fatalf("healed %d", healed)
	}
	if c.Potions != before-1 {
		t.Fatalf("potions %d", c.Potions)
	}
	c.Potions = 0
	if _, err := c.DrinkPotion(r); err == nil {
		t.Fatal("expected error with no potions")
	}
}
