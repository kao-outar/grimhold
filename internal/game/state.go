package game

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/kaoutar/grimhold/internal/dice"
	"github.com/kaoutar/grimhold/internal/open5e"
	"github.com/kaoutar/grimhold/internal/quest"
	"github.com/kaoutar/grimhold/internal/world"
)

const defaultSavePath = "save.json"

type MonsterSource interface {
	GetMonster(slug string) (*open5e.Monster, error)
	ListMonsters(pageURL string) (*open5e.Page, error)
	PrefetchMonsters(slugs []string) error
}

type State struct {
	World     *world.World
	Client    MonsterSource
	Story     *quest.Story
	Quest     quest.Progress
	Character Character
	Location  string
	Slain     map[string]int
	RestedIn  map[string]bool
	Encounter *Encounter
	Roller    dice.Roller

	BestiaryNext *string
	BestiaryPrev *string

	AmbushTick <-chan struct{}
	ambushStop chan struct{}

	SavePath string
	rng      *rand.Rand
}

type Persistable struct {
	Character   Character       `json:"character"`
	Location    string          `json:"location"`
	Slain       map[string]int  `json:"slain"`
	RestedIn    map[string]bool `json:"rested_in"`
	QuestFlags  map[string]bool `json:"quest_flags"`
	RegionKills map[string]int  `json:"region_kills"`
	QuestStage  int             `json:"quest_stage"`
	Ending      string          `json:"ending"`
}

func NewState(w *world.World, client MonsterSource, story *quest.Story, name string) *State {
	src := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(src)
	s := &State{
		World:     w,
		Client:    client,
		Story:     story,
		Quest:     quest.NewProgress(),
		Character: NewAdventurer(name),
		Location:  w.Start,
		Slain:     make(map[string]int),
		RestedIn:  make(map[string]bool),
		Roller:    dice.Rand{R: rng},
		SavePath:  defaultSavePath,
		rng:       rng,
	}
	s.StartAmbushTicker(45 * time.Second)
	return s
}

func (s *State) StoryTitle() string {
	if s.Story != nil && s.Story.Title != "" {
		return s.Story.Title
	}
	return "Grimhold"
}

func (s *State) StartAmbushTicker(every time.Duration) {
	if s.ambushStop != nil {
		select {
		case <-s.ambushStop:
		default:
			close(s.ambushStop)
		}
	}
	ch := make(chan struct{}, 1)
	stop := make(chan struct{})
	s.AmbushTick = ch
	s.ambushStop = stop
	go func() {
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				select {
				case ch <- struct{}{}:
				default:
				}
			}
		}
	}()
}

// RollAmbush runs on the goroutine that owns State. The ticker only signals.
func (s *State) RollAmbush() (string, bool) {
	if s.Encounter != nil && s.Encounter.Active {
		return "", false
	}
	reg, ok := s.World.Get(s.Location)
	if !ok || reg.Danger <= 0 || len(reg.Foes) == 0 {
		return "", false
	}
	if s.rng.Intn(100) >= 25 {
		return "", false
	}
	msg, err := s.forceAmbush()
	if err != nil {
		return fmt.Sprintf("*** EMBUSCADE dans %s ! (échec : %v) ***", reg.Name, err), true
	}
	return msg, true
}

func (s *State) forceAmbush() (string, error) {
	if s.Encounter != nil && s.Encounter.Active {
		return "", fmt.Errorf("déjà en combat")
	}
	reg, err := s.CurrentRegion()
	if err != nil {
		return "", err
	}
	if len(reg.Foes) == 0 {
		return "", fmt.Errorf("aucun foe")
	}
	monsters, roles, err := BuildPack(s.Roller, s.Client, reg.Foes, reg.Danger, s.Character.Level)
	if err != nil {
		return "", err
	}
	enc := StartPack(monsters, roles)
	s.Encounter = &enc
	return "*** EMBUSCADE ! ***\n" + FormatPackIntro(&enc, reg.Name), nil
}

func (s *State) StopAmbush() {
	if s.ambushStop != nil {
		select {
		case <-s.ambushStop:
		default:
			close(s.ambushStop)
		}
		s.ambushStop = nil
	}
}

func (s *State) CurrentRegion() (world.Region, error) {
	r, ok := s.World.Get(s.Location)
	if !ok {
		return world.Region{}, fmt.Errorf("lieu inconnu %q", s.Location)
	}
	return r, nil
}

func (s *State) ResolveTravelDest(arg string) (string, error) {
	if n, err := strconv.Atoi(arg); err == nil {
		return s.World.ExitByIndex(s.Location, n)
	}
	return arg, nil
}

func (s *State) Travel(dest string) (string, error) {
	if s.Encounter != nil && s.Encounter.Active {
		return "", fmt.Errorf("tu es en combat — fight / defend / potion / flee d'abord")
	}
	resolved, err := s.ResolveTravelDest(dest)
	if err != nil {
		return "", err
	}
	dest = resolved

	if !s.World.CanTravel(s.Location, dest) {
		return "", fmt.Errorf("pas de chemin vers %q depuis ici", dest)
	}
	reg, ok := s.World.Get(dest)
	if !ok {
		return "", fmt.Errorf("région inconnue %q", dest)
	}

	if dest == "bleak-spire" && s.Story != nil {
		ok, reason := s.Story.SpireUnlocked(s.Quest, s.Character.Level)
		if !ok {
			return "", fmt.Errorf("%s", reason)
		}
	} else if reg.MinLevel > 0 && s.Character.Level < reg.MinLevel {
		return "", fmt.Errorf("%s exige le niveau %d (tu es niveau %d)", reg.Name, reg.MinLevel, s.Character.Level)
	}

	s.Location = dest
	delete(s.RestedIn, dest)

	msg := fmt.Sprintf("Tu entres dans %s. %s", reg.Name, firstSentence(reg.Description))
	if reg.QuestHint != "" {
		msg += "\n(" + reg.QuestHint + ")"
	}
	if s.Client != nil && len(reg.Foes) > 0 {
		if err := s.Client.PrefetchMonsters(reg.Foes); err != nil {
			msg += fmt.Sprintf("\n(présignal des ennemis incomplet : %v)", err)
		}
	}
	return msg, nil
}

func firstSentence(desc string) string {
	for i, r := range desc {
		if r == '.' {
			return desc[:i+1]
		}
	}
	return desc
}

func (s *State) Look() (string, error) {
	reg, err := s.CurrentRegion()
	if err != nil {
		return "", err
	}
	msg := fmt.Sprintf("%s\n%s\nDanger: %s (niv. %d)\nSorties:",
		reg.Name, reg.Description, world.DangerHint(reg.Danger), reg.Danger)
	for i, e := range reg.Exits {
		msg += fmt.Sprintf("\n  [%d] %s", i+1, e)
	}
	if reg.QuestHint != "" {
		msg += "\nIndice: " + reg.QuestHint
	}
	if s.Story != nil {
		msg += "\nQuête: " + s.Story.ObjectiveLine(s.Quest)
	}
	if s.Encounter != nil && s.Encounter.Active {
		msg += "\n\nCombat en cours:\n" + s.Encounter.SummaryLine()
	}
	return msg, nil
}

func (s *State) Talk() (string, error) {
	if s.Story == nil {
		return "", fmt.Errorf("aucune histoire chargée")
	}
	return s.Story.Talk(&s.Quest, s.Location)
}

func (s *State) Journal() string {
	if s.Story == nil {
		return "Pas de journal de quête."
	}
	return s.Story.Journal(s.Quest)
}

func (s *State) ChooseEnding(choice string) (string, error) {
	if s.Story == nil {
		return "", fmt.Errorf("aucune histoire chargée")
	}
	return s.Story.ChooseEnding(&s.Quest, choice)
}

func (s *State) MapListing() string {
	var out string
	for _, r := range s.World.All() {
		mark := " "
		if r.ID == s.Location {
			mark = "*"
		}
		out += fmt.Sprintf("%s %-16s  danger %d  exits %v\n", mark, r.ID, r.Danger, r.Exits)
	}
	out += "(* = tu es ici)"
	return out
}

func (s *State) Explore() (string, error) {
	if s.Encounter != nil && s.Encounter.Active {
		return "", fmt.Errorf("tu combats déjà — %d ennemi(s)", len(s.Encounter.AliveFoes()))
	}
	reg, err := s.CurrentRegion()
	if err != nil {
		return "", err
	}

	note := ""
	if s.Story != nil {
		if act := s.Story.CurrentAct(s.Quest); act != nil && act.Region != "" && act.Region != s.Location {
			dest := act.Region
			if r, ok := s.World.Get(act.Region); ok {
				dest = r.Name
			}
			note = fmt.Sprintf("Objectif ailleurs : %s\nLa suite de la quête est vers %s (travel %s).", act.Objective, dest, act.Region)
		}
		if ok, msg := s.Story.OnScriptedExplore(&s.Quest, s.Location); ok {
			return "✦ " + msg, nil
		}
	}

	if reg.Boss != "" && s.Quest.Has("key_complete") && !s.Quest.Has("boss_felled") {
		m, err := s.Client.GetMonster(reg.Boss)
		if err != nil {
			return "", fmt.Errorf("fetch boss: %w", err)
		}
		enc := BuildBossEncounter(m)
		s.Encounter = &enc
		f := &enc.Foes[0]
		return withNote(note, fmt.Sprintf("Le %s descend la Flèche ! (AC %d, HP %d, CR %s)\nCibles: fight · defend · potion · flee",
			f.Monster.Name, f.Monster.ArmorClass, f.HP, f.Monster.ChallengeRating)), nil
	}

	if len(reg.Foes) == 0 {
		return withNote(note, "Rien ne bouge ici."), nil
	}
	monsters, roles, err := BuildPack(s.Roller, s.Client, reg.Foes, reg.Danger, s.Character.Level)
	if err != nil {
		return "", fmt.Errorf("fetch pack: %w", err)
	}
	enc := StartPack(monsters, roles)
	s.Encounter = &enc
	return withNote(note, FormatPackIntro(&enc, reg.Name)), nil
}

func withNote(note, msg string) string {
	if note == "" {
		return msg
	}
	return note + "\n\n" + msg
}

func (s *State) FightRound(targetSpec string) (string, error) {
	if s.Encounter == nil || !s.Encounter.Active {
		return "", fmt.Errorf("rien à combattre — essaie 'explore'")
	}
	enc := s.Encounter
	enc.PlayerDefending = false
	if err := enc.SetFocus(targetSpec); err != nil {
		return "", err
	}

	var b strings.Builder
	res := PlayerAttack(s.Roller, s.Character, enc)
	if res.Hit {
		b.WriteString(res.HitMsg)
	} else {
		b.WriteString(res.MissMsg)
	}

	if phase := MaybeTriggerBossPhase(s.Roller, enc, s.Client); phase != "" {
		b.WriteByte('\n')
		b.WriteString(phase)
	}

	if fell := s.noteFallenFoes(); fell != "" {
		b.WriteByte('\n')
		b.WriteString(fell)
	}
	if enc.AllDown() {
		b.WriteString(s.resolveVictory())
		return b.String(), nil
	}

	b.WriteByte('\n')
	foeLog := RunAllFoeTurns(s.Roller, &s.Character, enc)
	if foeLog != "" {
		b.WriteString(foeLog)
	}
	if s.Character.HP <= 0 {
		b.WriteString(s.resolveDefeat())
		return b.String(), nil
	}
	if enc.AllDown() {
		b.WriteString(s.resolveVictory())
	}
	return b.String(), nil
}

func (s *State) Defend() (string, error) {
	if s.Encounter == nil || !s.Encounter.Active {
		return "", fmt.Errorf("rien à défendre — pas de combat")
	}
	enc := s.Encounter
	enc.PlayerDefending = true
	var b strings.Builder
	fmt.Fprintf(&b, "Tu te mets en garde (+2 AC ce tour, AC %d).", PlayerAC(s.Character, true))
	b.WriteByte('\n')
	foeLog := RunAllFoeTurns(s.Roller, &s.Character, enc)
	enc.PlayerDefending = false
	if foeLog != "" {
		b.WriteString(foeLog)
	}
	if s.Character.HP <= 0 {
		b.WriteString(s.resolveDefeat())
		return b.String(), nil
	}
	if enc.AllDown() {
		b.WriteString(s.resolveVictory())
	}
	return b.String(), nil
}

func (s *State) UsePotion() (string, error) {
	healed, err := s.Character.DrinkPotion(s.Roller)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Tu bois une potion (+%d HP → %d/%d, potions restantes: %d).",
		healed, s.Character.HP, s.Character.MaxHP, s.Character.Potions)
	if s.Encounter == nil || !s.Encounter.Active {
		return b.String(), nil
	}
	enc := s.Encounter
	enc.PlayerDefending = false
	b.WriteByte('\n')
	foeLog := RunAllFoeTurns(s.Roller, &s.Character, enc)
	if foeLog != "" {
		b.WriteString(foeLog)
	}
	if s.Character.HP <= 0 {
		b.WriteString(s.resolveDefeat())
		return b.String(), nil
	}
	if enc.AllDown() {
		b.WriteString(s.resolveVictory())
	}
	return b.String(), nil
}

func (s *State) noteFallenFoes() string {
	enc := s.Encounter
	if enc == nil {
		return ""
	}
	var parts []string
	for i := range enc.Foes {
		f := &enc.Foes[i]
		if f.Monster == nil || f.HP > 0 || f.DeathNoted || f.Fled {
			continue
		}
		parts = append(parts, fmt.Sprintf("[%d] %s tombe !", i+1, f.Monster.Name))
		f.DeathNoted = true
	}
	return strings.Join(parts, "\n")
}

func (s *State) resolveVictory() string {
	enc := s.Encounter
	enc.Active = false

	totalXP := 0
	totalGold := 0
	var names []string
	var lastSlug string
	for i := range enc.Foes {
		f := &enc.Foes[i]
		if f.Monster == nil || f.HP > 0 || f.Fled {
			continue
		}
		name := f.Monster.Name
		slug := f.Monster.Slug
		lastSlug = slug
		names = append(names, name)
		s.Slain[name]++
		totalXP += XPFromCR(f.Monster.CR)
		totalGold += GoldFromCR(s.Roller, f.Monster.CR)
	}
	if enc.IsBossFight {
		totalXP += 50
		totalGold += 10
	}
	s.Character.Gold += totalGold
	potionDrop := ""
	if s.Roller.Intn(100) < 30 {
		s.Character.Potions++
		potionDrop = " +1 potion!"
	}

	s.Encounter = nil
	if len(names) == 0 {
		return "\nLe champ est vide — les survivants ont fui."
	}
	msg := fmt.Sprintf("\nVictoire ! %s — +%d XP, +%d or.%s",
		strings.Join(names, ", "), totalXP, totalGold, potionDrop)
	msg += s.Character.AddXP(totalXP)

	if s.Story != nil && lastSlug != "" {
		if qmsg := s.Story.OnVictory(&s.Quest, s.Location, lastSlug, s.Character.Level); qmsg != "" {
			msg += "\n\n✦ " + qmsg
		}
	}
	return msg
}

func (s *State) resolveDefeat() string {
	s.Encounter = nil
	s.Location = s.World.Start
	s.Character.HP = 1
	return fmt.Sprintf("\nLes ténèbres… Tu te réveilles aux %s avec 1 HP et l'orgueil en miettes.", s.World.Start)
}

func (s *State) Flee() (string, error) {
	if s.Encounter == nil || !s.Encounter.Active {
		return "", fmt.Errorf("tu n'es pas en combat")
	}
	escaped, log := FleeAttempt(s.Roller, &s.Character, s.Encounter)
	if escaped {
		s.Encounter = nil
		return log, nil
	}
	if s.Character.HP <= 0 {
		log += s.resolveDefeat()
	}
	return log, nil
}

func (s *State) Rest() (string, error) {
	if s.Encounter != nil && s.Encounter.Active {
		return "", fmt.Errorf("pas de repos en plein combat")
	}
	if s.RestedIn[s.Location] {
		return "", fmt.Errorf("déjà reposé ici — voyage ailleurs d'abord")
	}
	s.Character.HP = s.Character.MaxHP
	s.RestedIn[s.Location] = true
	return fmt.Sprintf("Tu te reposes. HP %d/%d.", s.Character.HP, s.Character.MaxHP), nil
}

func (s *State) SlainBestiary() string {
	if len(s.Slain) == 0 {
		return "Bestiaire des tués vide. Va faire des entrées."
	}
	out := "Bestiaire des tués:\n"
	for name, n := range s.Slain {
		out += fmt.Sprintf("  %s × %d\n", name, n)
	}
	return out
}

func (s *State) Save() error {
	p := Persistable{
		Character:   s.Character,
		Location:    s.Location,
		Slain:       s.Slain,
		RestedIn:    s.RestedIn,
		QuestFlags:  s.Quest.Flags,
		RegionKills: s.Quest.RegionKills,
		QuestStage:  s.Quest.Stage,
		Ending:      s.Quest.Ending,
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal save: %w", err)
	}
	tmp := s.SavePath + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write save: %w", err)
	}
	if err := os.Rename(tmp, s.SavePath); err != nil {
		if err2 := os.WriteFile(s.SavePath, data, 0o644); err2 != nil {
			return fmt.Errorf("commit save: %w", err2)
		}
		_ = os.Remove(tmp)
	}
	return nil
}

func (s *State) Load() error {
	data, err := os.ReadFile(s.SavePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("aucune sauvegarde à %s", s.SavePath)
		}
		return fmt.Errorf("read save: %w", err)
	}
	var p Persistable
	if err := json.Unmarshal(data, &p); err != nil {
		return fmt.Errorf("decode save: %w", err)
	}
	if _, ok := s.World.Get(p.Location); !ok {
		return fmt.Errorf("lieu de sauvegarde %q absent du monde", p.Location)
	}
	s.Character = p.Character
	s.Location = p.Location
	if p.Slain == nil {
		p.Slain = make(map[string]int)
	}
	if p.RestedIn == nil {
		p.RestedIn = make(map[string]bool)
	}
	s.Slain = p.Slain
	s.RestedIn = p.RestedIn
	s.Quest = quest.Progress{
		Flags:       p.QuestFlags,
		RegionKills: p.RegionKills,
		Stage:       p.QuestStage,
		Ending:      p.Ending,
	}
	if s.Quest.Flags == nil {
		s.Quest.Flags = make(map[string]bool)
	}
	if s.Quest.RegionKills == nil {
		s.Quest.RegionKills = make(map[string]int)
	}
	s.Encounter = nil
	return nil
}
