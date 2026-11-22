package game

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/kaoutar/grimhold/internal/dice"
	"github.com/kaoutar/grimhold/internal/open5e"
)

type stubClient struct {
	monsters map[string]*open5e.Monster
}

func (s stubClient) GetMonster(slug string) (*open5e.Monster, error) {
	m, ok := s.monsters[slug]
	if !ok {
		return nil, fmt.Errorf("missing %s", slug)
	}
	cp := *m
	return &cp, nil
}

func (s stubClient) ListMonsters(string) (*open5e.Page, error) {
	return &open5e.Page{}, nil
}

func (s stubClient) PrefetchMonsters([]string) error {
	return nil
}

func TestPackSizeBounds(t *testing.T) {
	r := dice.Rand{R: rand.New(rand.NewSource(11))}
	for i := 0; i < 40; i++ {
		n := PackSize(r, 3, 2)
		if n < 1 || n > 3 {
			t.Fatalf("pack size %d out of bounds", n)
		}
	}
}

func TestBuildPack(t *testing.T) {
	r := dice.Rand{R: rand.New(rand.NewSource(5))}
	client := stubClient{monsters: map[string]*open5e.Monster{
		"goblin":   {Slug: "goblin", Name: "Goblin", Type: "humanoid", HitPoints: 7, ArmorClass: 15, CR: 0.25},
		"skeleton": {Slug: "skeleton", Name: "Skeleton", Type: "undead", HitPoints: 13, ArmorClass: 13, CR: 0.25},
		"ghoul":    {Slug: "ghoul", Name: "Ghoul", Type: "undead", HitPoints: 22, ArmorClass: 12, CR: 1},
	}}
	foes := []string{"goblin", "skeleton", "ghoul"}
	monsters, roles, err := BuildPack(r, client, foes, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(monsters) == 0 || len(monsters) != len(roles) {
		t.Fatalf("monsters=%d roles=%d", len(monsters), len(roles))
	}
	enc := StartPack(monsters, roles)
	if !enc.Active || enc.AllDown() {
		t.Fatal("pack should be active with living foes")
	}
}

func TestBossPhaseSummon(t *testing.T) {
	r := dice.Rand{R: rand.New(rand.NewSource(1))}
	wraith := &open5e.Monster{
		Slug: "wraith", Name: "Wraith", HitPoints: 40, ArmorClass: 13, CR: 5,
		Actions: []open5e.MonsterAction{{Name: "Life Drain", AttackBonus: 6, DamageDice: "4d8"}},
	}
	enc := BuildBossEncounter(wraith)
	if !enc.IsBossFight || !enc.Foes[0].IsBoss {
		t.Fatal("expected boss encounter")
	}
	client := stubClient{monsters: map[string]*open5e.Monster{
		"vampire-spawn": {Slug: "vampire-spawn", Name: "Vampire Spawn", HitPoints: 30, Type: "undead"},
		"shadow":        {Slug: "shadow", Name: "Shadow", HitPoints: 16, Type: "undead"},
	}}
	enc.Foes[0].HP = enc.Foes[0].MaxHP / 4
	msg := MaybeTriggerBossPhase(r, &enc, client)
	if msg == "" {
		t.Fatal("expected phase trigger message")
	}
	if enc.Foes[0].Phase != 1 || !enc.Foes[0].PhaseTriggered {
		t.Fatal("boss should be in phase 2")
	}
	if len(enc.Foes) < 2 {
		t.Fatal("expected summon")
	}
	if MaybeTriggerBossPhase(r, &enc, client) != "" {
		t.Fatal("phase should not re-trigger")
	}
}

type scriptRoller struct {
	rolls []int
	i     int
}

func (s *scriptRoller) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	v := 0
	if s.i < len(s.rolls) {
		v = s.rolls[s.i]
	}
	s.i++
	if v < 0 {
		v = 0
	}
	return v % n
}

func TestPickFoeKeepsWorst(t *testing.T) {
	foes := []string{"goblin", "skeleton", "ghoul"}
	r := &scriptRoller{rolls: []int{0, 2, 1}}
	got := pickFoeSlug(r, foes, 2)
	if got != "ghoul" {
		t.Fatalf("danger 2 should keep ghoul, got %s", got)
	}
	r = &scriptRoller{rolls: []int{0, 2, 1}}
	if got := pickFoeSlug(r, foes, 0); got != "goblin" {
		t.Fatalf("danger 0 rolls once, got %s", got)
	}
}

func TestFormatPackIntro(t *testing.T) {
	m := &open5e.Monster{Name: "Goblin", ArmorClass: 15, HitPoints: 7, ChallengeRating: "1/4"}
	enc := StartEncounter(m)
	s := FormatPackIntro(&enc, "Marais")
	if s == "" {
		t.Fatal("empty intro")
	}
}
