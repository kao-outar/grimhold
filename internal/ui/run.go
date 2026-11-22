package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/kaoutar/grimhold/internal/game"
	"github.com/kaoutar/grimhold/internal/repl"
)

func Run(state *game.State, in io.Reader, out io.Writer) error {
	log := []string{}
	if state.Story != nil {
		log = append(log, state.Story.Title, state.Story.Prologue)
	}
	log = append(log, tutorialIntro(state)...)

	session := repl.NewSession(state)
	reader := repl.NewLineReader(in)
	lastTip := tutorialTip(state)
	tutDoneAnnounced := state.Quest.Has("shard_marsh")

	for {
		Render(out, Frame{State: state, Log: log, Suggest: Suggest(state)})

		line, ambush, eof, err := reader.Wait(state.AmbushTick, state.RollAmbush)
		if err != nil {
			return err
		}
		if ambush != "" {
			log = append(log, strings.Split(strings.TrimRight(ambush, "\n"), "\n")...)
			continue
		}
		if eof {
			fmt.Fprintln(out)
			return nil
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		msg, quit, err := session.Exec(line)
		if msg != "" {
			log = append(log, strings.Split(strings.TrimRight(msg, "\n"), "\n")...)
		}
		if err != nil {
			log = append(log, "⚠ "+err.Error())
		}

		if tip := tutorialTip(state); tip != "" && tip != lastTip {
			log = append(log, "Tuto → "+tip)
			lastTip = tip
		}
		if !tutDoneAnnounced && state.Quest.Has("shard_marsh") {
			log = append(log, "Tutoriel terminé. Continue avec journal (j) pour la suite — help liste tout.")
			tutDoneAnnounced = true
			lastTip = ""
		}
		if len(log) > 40 {
			log = log[len(log)-40:]
		}

		if quit {
			ClearHome(out)
			fmt.Fprintln(out, "Farewell, wanderer.")
			return nil
		}
	}
}
