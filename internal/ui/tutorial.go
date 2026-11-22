package ui

import (
	"fmt"

	"github.com/kaoutar/grimhold/internal/game"
)

func tutorialActive(s *game.State) bool {
	if s == nil || s.Quest.Ending != "" {
		return false
	}
	return !s.Quest.Has("shard_marsh")
}

func tutorialTip(s *game.State) string {
	if !tutorialActive(s) {
		return ""
	}
	if s.Encounter != nil && s.Encounter.Active {
		return "Combat: [f] attaquer · [d] défendre (+AC) · [p] potion · flee. L'étoile * marque la cible."
	}
	if !s.Quest.Has("met_herald") {
		if s.Location == "crossroads" {
			return "Lis Quête + Actions en bas. En-tête=PV · Carte=sorties numérotées · Récit=histoire. Tape [t] talk."
		}
		return "Le héraut est aux Carrefours. Reviens (travel / numéro) puis tape [t] talk."
	}
	if s.Location != "ashen-marsh" {
		if n, ok := exitIndex(s, "ashen-marsh"); ok {
			return fmt.Sprintf("Mission 1: Marais Cendreux. Tape [%d] pour y aller (ou: travel ashen-marsh).", n)
		}
		return "Mission 1: Marais Cendreux. Tape travel ashen-marsh."
	}
	return "Sur place: [e] explore pour chercher l'Éclat (peut déclencher un combat). [j] journal = objectif."
}

func tutorialIntro(s *game.State) []string {
	if s.Quest.Has("met_herald") {
		return []string{"Reprise. Tape 'help' pour les commandes — la barre Quête rappelle ton objectif."}
	}
	return []string{
		"— Tutoriel (première mission) —",
		"L'écran: en-tête (PV, or, potions) · Lieu · Quête (ton objectif) · Carte (sorties [1][2]…) · Récit · Actions.",
		"Tu tapes une commande après « grimhold > ». Raccourcis entre crochets: t=talk, e=explore, 1=sortie 1.",
		"Objectif: parle au héraut de pierre — tape talk (ou t).",
	}
}

func exitIndex(s *game.State, dest string) (int, bool) {
	reg, err := s.CurrentRegion()
	if err != nil {
		return 0, false
	}
	for i, e := range reg.Exits {
		if e == dest {
			return i + 1, true
		}
	}
	return 0, false
}
