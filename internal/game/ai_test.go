package game

import (
	"math/rand"
	"testing"

	"github.com/kaoutar/grimhold/internal/dice"
	"github.com/kaoutar/grimhold/internal/open5e"
)

func TestAssignRoleHeuristics(t *testing.T) {
	cases := []struct {
		m    *open5e.Monster
		want FoeRole
	}{
		{&open5e.Monster{Name: "Goblin", Slug: "goblin", Type: "humanoid", CR: 0.25}, RoleSkirmisher},
		{&open5e.Monster{Name: "Ogre", Slug: "ogre", Type: "giant", CR: 2}, RoleBruiser},
		{&open5e.Monster{Name: "Shadow", Slug: "shadow", Type: "undead", CR: 0.5}, RoleSkirmisher},
	}
	for _, tc := range cases {
		got := AssignRole(tc.m)
		if got != tc.want {
			t.Fatalf("%s: got %s want %s", tc.m.Name, got, tc.want)
		}
	}
}

func TestChooseActionPrefersBruiserDamage(t *testing.T) {
	r := dice.Rand{R: rand.New(rand.NewSource(3))}
	foe := &Foe{
		Monster: &open5e.Monster{
			Name: "Bugbear",
			Actions: []open5e.MonsterAction{
				{Name: "Morningstar", AttackBonus: 4, DamageDice: "2d8", DamageBonus: 2},
				{Name: "Javelin", AttackBonus: 4, DamageDice: "1d6", DamageBonus: 2},
			},
		},
		Role:  RoleBruiser,
		HP:    20,
		MaxHP: 20,
	}
	enc := &Encounter{Foes: []Foe{*foe}, Active: true}
	atk := ChooseAction(r, foe, enc, 20, 20)
	if atk.Name != "Morningstar" {
		t.Fatalf("bruiser chose %s, want Morningstar", atk.Name)
	}
}

func TestSupportHealsWoundedAlly(t *testing.T) {
	r := dice.Rand{R: rand.New(rand.NewSource(1))}
	bruiser := Foe{
		Monster: &open5e.Monster{Name: "Ogre", Slug: "ogre", HitPoints: 20},
		HP:      5,
		MaxHP:   20,
		Role:    RoleBruiser,
	}
	support := Foe{
		Monster: &open5e.Monster{
			Name: "Priest",
			Slug: "priest",
			Actions: []open5e.MonsterAction{
				{Name: "Mace", AttackBonus: 2, DamageDice: "1d6"},
			},
		},
		HP:    10,
		MaxHP: 10,
		Role:  RoleSupport,
	}
	enc := &Encounter{Foes: []Foe{bruiser, support}, Active: true}
	c := NewAdventurer("T")

	sawHeal := false
	for i := 0; i < 80; i++ {
		enc.Foes[0].HP = 5
		enc.Foes[1].HP = 10
		turn := RunFoeAI(r, &c, enc, 1)
		if turn.Healed {
			sawHeal = true
			if enc.Foes[0].HP <= 5 {
				t.Fatalf("expected heal, HP=%d", enc.Foes[0].HP)
			}
			break
		}
	}
	if !sawHeal {
		t.Fatal("expected support heal within 80 attempts")
	}
}

func TestRunAllFoeTurnsAttacks(t *testing.T) {
	r := dice.Rand{R: rand.New(rand.NewSource(7))}
	m := &open5e.Monster{
		Name: "Goblin",
		Actions: []open5e.MonsterAction{
			{Name: "Scimitar", AttackBonus: 20, DamageDice: "1d6", DamageBonus: 2},
		},
		HitPoints: 7,
	}
	enc := StartEncounter(m)
	c := NewAdventurer("T")
	c.HP = 20
	log := RunAllFoeTurns(r, &c, &enc)
	if log == "" {
		t.Fatal("expected foe log")
	}
}
