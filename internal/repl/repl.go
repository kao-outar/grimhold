package repl

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/kaoutar/grimhold/internal/game"
)

type cliCommand struct {
	name        string
	description string
	callback    func(cfg *config, args []string) error
}

type config struct {
	state    *game.State
	commands map[string]cliCommand
	out      io.Writer
	quit     bool
}

var aliases = map[string]string{
	"h":  "help",
	"?":  "help",
	"m":  "map",
	"l":  "look",
	"e":  "explore",
	"f":  "fight",
	"c":  "character",
	"r":  "rest",
	"t":  "talk",
	"j":  "journal",
	"s":  "save",
	"q":  "quit",
	"d":  "defend",
	"p":  "potion",
	"go": "travel",
	"cd": "travel",
}

type Session struct {
	state    *game.State
	commands map[string]cliCommand
}

func NewSession(state *game.State) *Session {
	return &Session{state: state, commands: registerCommands()}
}

func (s *Session) Exec(line string) (output string, quit bool, err error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false, nil
	}
	parts := strings.Fields(line)
	name := strings.ToLower(parts[0])
	args := parts[1:]

	if _, atoiErr := strconv.Atoi(name); atoiErr == nil && len(args) == 0 {
		args = []string{name}
		name = "travel"
	}

	if full, ok := aliases[name]; ok {
		name = full
	}

	cmd, ok := s.commands[name]
	if !ok {
		return "", false, fmt.Errorf("commande inconnue %q — tape 'help'", name)
	}

	var buf strings.Builder
	cfg := &config{state: s.state, commands: s.commands, out: &buf}
	if e := cmd.callback(cfg, args); e != nil {
		return buf.String(), cfg.quit, e
	}
	return strings.TrimRight(buf.String(), "\n"), cfg.quit, nil
}

type LineReader struct {
	lines <-chan string
	errs  <-chan error
}

func NewLineReader(in io.Reader) *LineReader {
	lines := make(chan string)
	errs := make(chan error, 1)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(in)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("read input: %w", err)
		}
	}()
	return &LineReader{lines: lines, errs: errs}
}

// Wait blocks until the next line, EOF, a read error, or an ambush.
// A nil tick is ignored. roll runs on the caller goroutine.
func (r *LineReader) Wait(tick <-chan struct{}, roll func() (string, bool)) (line, ambush string, eof bool, err error) {
	for {
		select {
		case <-tick:
			if roll == nil {
				continue
			}
			text, ok := roll()
			if ok && text != "" {
				return "", text, false, nil
			}
		case line, ok := <-r.lines:
			if !ok {
				select {
				case err := <-r.errs:
					return "", "", false, err
				default:
					return "", "", true, nil
				}
			}
			return line, "", false, nil
		case err := <-r.errs:
			return "", "", false, err
		}
	}
}

func Run(state *game.State, in io.Reader, out io.Writer) error {
	session := NewSession(state)
	reader := NewLineReader(in)
	fmt.Fprintln(out, "Grimhold — a dungeon crawler. Type 'help' for commands.")
	for {
		fmt.Fprint(out, "grimhold > ")
		line, ambush, eof, err := reader.Wait(state.AmbushTick, state.RollAmbush)
		if err != nil {
			return err
		}
		if ambush != "" {
			fmt.Fprintln(out)
			fmt.Fprintln(out, ambush)
			continue
		}
		if eof {
			fmt.Fprintln(out)
			return nil
		}
		msg, quit, err := session.Exec(line)
		if msg != "" {
			fmt.Fprintln(out, msg)
		}
		if err != nil {
			fmt.Fprintf(out, "%v\n", err)
		}
		if quit {
			return nil
		}
	}
}

func registerCommands() map[string]cliCommand {
	cmds := map[string]cliCommand{}
	add := func(c cliCommand) { cmds[c.name] = c }

	add(cliCommand{name: "help", description: "Show this message", callback: cmdHelp})
	add(cliCommand{
		name:        "map",
		description: "List the known regions",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: map")
			}
			fmt.Fprintln(cfg.out, cfg.state.MapListing())
			return nil
		},
	})
	add(cliCommand{
		name:        "travel",
		description: "Travel to a connected region (name or exit number)",
		callback: func(cfg *config, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("usage: travel <region|number>")
			}
			msg, err := cfg.state.Travel(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cfg.out, msg)
			return nil
		},
	})
	add(cliCommand{
		name:        "look",
		description: "Describe your surroundings",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: look")
			}
			msg, err := cfg.state.Look()
			if err != nil {
				return err
			}
			fmt.Fprintln(cfg.out, msg)
			return nil
		},
	})
	add(cliCommand{
		name:        "explore",
		description: "Wander the region (risk an encounter / quest beat)",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: explore")
			}
			msg, err := cfg.state.Explore()
			if err != nil {
				return err
			}
			fmt.Fprintln(cfg.out, msg)
			return nil
		},
	})
	add(cliCommand{
		name:        "fight",
		description: "Attack a foe (fight [n|name])",
		callback: func(cfg *config, args []string) error {
			target := ""
			if len(args) > 1 {
				return fmt.Errorf("usage: fight [n|name]")
			}
			if len(args) == 1 {
				target = args[0]
			}
			msg, err := cfg.state.FightRound(target)
			if err != nil {
				return err
			}
			fmt.Fprintln(cfg.out, msg)
			return nil
		},
	})
	add(cliCommand{
		name:        "defend",
		description: "Guard (+2 AC) then let foes act",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: defend")
			}
			msg, err := cfg.state.Defend()
			if err != nil {
				return err
			}
			fmt.Fprintln(cfg.out, msg)
			return nil
		},
	})
	add(cliCommand{
		name:        "potion",
		description: "Drink a healing potion (2d4+2)",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: potion")
			}
			msg, err := cfg.state.UsePotion()
			if err != nil {
				return err
			}
			fmt.Fprintln(cfg.out, msg)
			return nil
		},
	})
	add(cliCommand{
		name:        "flee",
		description: "Try to escape combat",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: flee")
			}
			msg, err := cfg.state.Flee()
			if err != nil {
				return err
			}
			fmt.Fprintln(cfg.out, msg)
			return nil
		},
	})
	add(cliCommand{
		name:        "character",
		description: "Show your character sheet",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: character")
			}
			fmt.Fprintln(cfg.out, cfg.state.Character.Sheet())
			return nil
		},
	})
	add(cliCommand{
		name:        "rest",
		description: "Restore HP (once per region visit)",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: rest")
			}
			msg, err := cfg.state.Rest()
			if err != nil {
				return err
			}
			fmt.Fprintln(cfg.out, msg)
			return nil
		},
	})
	add(cliCommand{
		name:        "talk",
		description: "Speak to a local NPC / stone herald",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: talk")
			}
			msg, err := cfg.state.Talk()
			if err != nil {
				return err
			}
			fmt.Fprintln(cfg.out, msg)
			return nil
		},
	})
	add(cliCommand{
		name:        "choose",
		description: "Choose an ending: choose mercy | choose wrath",
		callback: func(cfg *config, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("usage: choose mercy | choose wrath")
			}
			msg, err := cfg.state.ChooseEnding(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cfg.out, msg)
			return nil
		},
	})
	add(cliCommand{
		name:        "journal",
		description: "Show the quest journal and current objective",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: journal")
			}
			fmt.Fprintln(cfg.out, cfg.state.Journal())
			return nil
		},
	})
	add(cliCommand{
		name:        "bestiary",
		description: "List Open5e monsters (paginated) or 'bestiary slain'",
		callback:    cmdBestiary,
	})
	add(cliCommand{
		name:        "next",
		description: "Next page of the Open5e bestiary",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: next")
			}
			if cfg.state.BestiaryNext == nil || *cfg.state.BestiaryNext == "" {
				return fmt.Errorf("pas de page suivante — lance 'bestiary' d'abord")
			}
			return printMonsterPage(cfg, *cfg.state.BestiaryNext)
		},
	})
	add(cliCommand{
		name:        "prev",
		description: "Previous page of the Open5e bestiary",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: prev")
			}
			if cfg.state.BestiaryPrev == nil || *cfg.state.BestiaryPrev == "" {
				return fmt.Errorf("pas de page précédente")
			}
			return printMonsterPage(cfg, *cfg.state.BestiaryPrev)
		},
	})
	add(cliCommand{
		name:        "inspect",
		description: "Print a monster stat block from Open5e",
		callback: func(cfg *config, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("usage: inspect <monster>")
			}
			m, err := cfg.state.Client.GetMonster(args[0])
			if err != nil {
				return err
			}
			fmt.Fprint(cfg.out, m.FormatStatBlock())
			return nil
		},
	})
	add(cliCommand{
		name:        "save",
		description: "Write game state to save.json",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: save")
			}
			if err := cfg.state.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cfg.out, "Sauvegardé dans %s.", cfg.state.SavePath)
			return nil
		},
	})
	add(cliCommand{
		name:        "load",
		description: "Restore game state from save.json",
		callback: func(cfg *config, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("usage: load")
			}
			if err := cfg.state.Load(); err != nil {
				return err
			}
			fmt.Fprintf(cfg.out, "Chargé. Tu es à %s.", cfg.state.Location)
			return nil
		},
	})
	add(cliCommand{
		name:        "quit",
		description: "Exit the game",
		callback: func(cfg *config, args []string) error {
			cfg.quit = true
			fmt.Fprintln(cfg.out, "Farewell, wanderer.")
			return nil
		},
	})
	add(cliCommand{
		name:        "exit",
		description: "Exit the game",
		callback: func(cfg *config, args []string) error {
			cfg.quit = true
			fmt.Fprintln(cfg.out, "Farewell, wanderer.")
			return nil
		},
	})
	return cmds
}

func cmdHelp(cfg *config, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: help")
	}
	fmt.Fprintln(cfg.out, "Grimhold — La Couronne de Brume")
	fmt.Fprintln(cfg.out)
	fmt.Fprintln(cfg.out, "Commands:")
	names := make([]string, 0, len(cfg.commands))
	for name := range cfg.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		c := cfg.commands[name]
		fmt.Fprintf(cfg.out, "  %-22s %s\n", c.name, c.description)
	}
	fmt.Fprintln(cfg.out)
	fmt.Fprintln(cfg.out, "Raccourcis: e=explore f=fight d=defend p=potion l=look t=talk j=journal r=rest s=save  1..N=travel")
	return nil
}

func cmdBestiary(cfg *config, args []string) error {
	if len(args) == 1 && args[0] == "slain" {
		fmt.Fprintln(cfg.out, cfg.state.SlainBestiary())
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: bestiary [slain]")
	}
	return printMonsterPage(cfg, "")
}

func printMonsterPage(cfg *config, pageURL string) error {
	page, err := cfg.state.Client.ListMonsters(pageURL)
	if err != nil {
		return err
	}
	cfg.state.BestiaryNext = page.Next
	cfg.state.BestiaryPrev = page.Previous
	fmt.Fprintf(cfg.out, "Monsters (%d total):\n", page.Count)
	for _, m := range page.Results {
		fmt.Fprintf(cfg.out, "  %-24s CR %-4s  HP %d  AC %d\n", m.Slug, m.ChallengeRating, m.HitPoints, m.ArmorClass)
	}
	if page.Next != nil {
		fmt.Fprintln(cfg.out, "(type 'next' / 'prev')")
	}
	return nil
}
