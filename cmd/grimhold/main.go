package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kaoutar/grimhold/internal/game"
	"github.com/kaoutar/grimhold/internal/open5e"
	"github.com/kaoutar/grimhold/internal/quest"
	"github.com/kaoutar/grimhold/internal/repl"
	"github.com/kaoutar/grimhold/internal/ui"
	"github.com/kaoutar/grimhold/internal/world"
)

func main() {
	continueFlag := flag.Bool("continue", false, "load save.json on startup")
	worldPath := flag.String("world", "world.json", "path to world map JSON")
	storyPath := flag.String("story", "story.json", "path to quest story JSON")
	name := flag.String("name", "Wanderer", "character name for a new game")
	plain := flag.Bool("plain", false, "disable TUI (plain REPL)")
	flag.Parse()

	if err := run(*continueFlag, *worldPath, *storyPath, *name, *plain); err != nil {
		fmt.Fprintf(os.Stderr, "grimhold: %v\n", err)
		os.Exit(1)
	}
}

func run(cont bool, worldPath, storyPath, name string, plain bool) error {
	worldPath = resolveDataFile(worldPath, "world.json")
	storyPath = resolveDataFile(storyPath, "story.json")

	w, err := world.Load(worldPath)
	if err != nil {
		return fmt.Errorf("load world: %w", err)
	}
	story, err := quest.LoadStory(storyPath)
	if err != nil {
		return fmt.Errorf("load story: %w", err)
	}

	cache := open5e.NewCache(5*time.Minute, 1*time.Minute)
	defer cache.Close()
	client := open5e.NewClient(cache)

	state := game.NewState(w, client, story, name)
	defer state.StopAmbush()

	if cont {
		if err := state.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v — nouvelle partie\n", err)
		}
	}

	if plain {
		return repl.Run(state, os.Stdin, os.Stdout)
	}
	return ui.Run(state, os.Stdin, os.Stdout)
}

func resolveDataFile(path, fallbackName string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}
	alt := filepath.Join("..", "..", fallbackName)
	if _, err := os.Stat(alt); err == nil {
		return alt
	}
	return path
}
