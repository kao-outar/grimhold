package game

import (
	"fmt"

	"github.com/kaoutar/grimhold/internal/dice"
)

type Character struct {
	Name         string `json:"name"`
	HP           int    `json:"hp"`
	MaxHP        int    `json:"max_hp"`
	Level        int    `json:"level"`
	XP           int    `json:"xp"`
	AttackBonus  int    `json:"attack_bonus"`
	WeaponName   string `json:"weapon_name"`
	WeaponDice   string `json:"weapon_dice"`
	Gold         int    `json:"gold"`
	DexterityMod int    `json:"dexterity_mod"`
	Potions      int    `json:"potions"`
}

func NewAdventurer(name string) Character {
	if name == "" {
		name = "Wanderer"
	}
	return Character{
		Name:         name,
		HP:           20,
		MaxHP:        20,
		Level:        1,
		XP:           0,
		AttackBonus:  3,
		WeaponName:   "shortsword",
		WeaponDice:   "1d6+2",
		Gold:         5,
		DexterityMod: 2,
		Potions:      2,
	}
}

func XPToLevel(level int) int {
	if level < 1 {
		level = 1
	}
	return level * level * 50
}

func (c *Character) AddXP(amount int) string {
	if amount <= 0 {
		return ""
	}
	c.XP += amount
	var msg string
	for c.XP >= XPToLevel(c.Level) {
		c.XP -= XPToLevel(c.Level)
		c.Level++
		c.MaxHP += 6
		c.HP = c.MaxHP
		c.AttackBonus++
		msg += fmt.Sprintf("\nYou rise to level %d! Max HP %d, attack bonus %+d.", c.Level, c.MaxHP, c.AttackBonus)
	}
	return msg
}

func (c *Character) DrinkPotion(r dice.Roller) (healed int, err error) {
	if c.Potions <= 0 {
		return 0, fmt.Errorf("plus de potions")
	}
	heal, err := dice.RollExpr(r, "2d4+2")
	if err != nil || heal < 1 {
		heal = 5
	}
	c.Potions--
	before := c.HP
	c.HP += heal
	if c.HP > c.MaxHP {
		c.HP = c.MaxHP
	}
	return c.HP - before, nil
}

func (c Character) Sheet() string {
	next := XPToLevel(c.Level)
	return fmt.Sprintf(
		"%s — Level %d\n  HP %d/%d\n  XP %d / %d to next level\n  Attack bonus %+d\n  Weapon: %s (%s)\n  Gold: %d\n  Potions: %d",
		c.Name, c.Level, c.HP, c.MaxHP, c.XP, next, c.AttackBonus, c.WeaponName, c.WeaponDice, c.Gold, c.Potions,
	)
}
