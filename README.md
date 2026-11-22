# Grimhold — La Couronne de Brume

Dungeon crawler textuel dans le terminal, avec **interface TUI ANSI** et une
**quête narrative complète**. Les monstres viennent de l’[API Open5e](https://open5e.com)
(D&D 5e SRD, gratuit, sans clé). La carte est dans `world.json`, l’histoire dans
`story.json`.

## Prérequis

- Go 1.22+ (stdlib uniquement pour le cœur)

## Build & lancement

```bash
go build -o grimhold.exe ./cmd/grimhold
./grimhold.exe
```

Ou :

```bash
go run ./cmd/grimhold
```

Lance depuis la racine du projet pour trouver `world.json` et `story.json`.

### Flags

| Flag | Effet |
|---|---|
| `--name <nom>` | Nom du personnage (défaut `Wanderer`) |
| `--continue` | Charge `save.json` au démarrage |
| `--world <path>` | Carte JSON |
| `--story <path>` | Quête JSON |
| `--plain` | REPL classique sans TUI |

## Comment jouer (parcours recommandé)

1. `talk` aux **Carrefours** — le héraut lance la quête  
2. `1` ou `travel ashen-marsh` → `explore` / `fight` jusqu’à l’**Éclat de Brume**  
3. Chapelle Noyée → `explore` pour la **Cloche Immersée**  
4. Terriers de Fer + Tumulus → combats pour les **demi-clés**  
5. Monter au niveau 2 (`rest`, combats), puis `travel bleak-spire`  
6. `explore` → boss Wraith → `choose mercy` ou `choose wrath`

### Raccourcis TUI

`e` explore · `f` fight · `d` defend · `p` potion · `l` look · `t` talk · `j` journal · `r` rest · `s` save · `1..N` travel

L’écran affiche toujours : barre HP, potions, lieu, objectif de quête, sorties numérotées, récit, meute en combat, actions suggérées.

## Commandes

| Commande | Rôle |
|---|---|
| `help` | Liste générée depuis la map de commandes |
| `map` / `look` / `travel` | Monde |
| `explore` / `fight [n\|nom]` / `defend` / `potion` / `flee` | Combat multi-ennemis |
| `character` / `rest` | Fiche & soin |
| `talk` / `journal` | PNJ & journal de quête |
| `choose mercy` / `choose wrath` | Fin |
| `bestiary` / `next` / `prev` / `inspect` | Open5e |
| `save` / `load` | Persistance (inclut flags de quête) |
| `quit` | Quitter |

## Session d’exemple

```text
┌─ GRIMHOLD — La Couronne de Brume ── HP [██████████] 20/20  Lv1  Or:5 ─┐
│ Lieu: Les Carrefours                                                 │
│ Quête: Parle au héraut de pierre… (0/5)                              │
…
grimhold > talk
Le héraut de pierre s'anime. « Quatre éclats… »

grimhold > 1
Tu entres dans Marais Cendreux. …

grimhold > explore
Un(e) Goblin surgit…

grimhold > f
You hit… The Goblin falls! …
```

### Combat (meutes + IA)

- Packs de **1–3** ennemis selon danger / niveau (`BuildPack`)
- Danger élevé : plusieurs tirages, on garde le pire de la liste (le plus loin dans `foes`)
- `explore` lance toujours une rencontre. Hors de la région de quête, un rappel d’objectif s’ajoute au combat
- Rôles IA : **bruiser** (gros dégâts), **skirmisher** (hits sûrs / fuite basse vie), **support** (soin d’allié)
- Actions joueur : `fight [n|nom]`, `defend` (+2 AC), `potion` (2d4+2), `flee` (DC selon meute ; échec = coup gratuit)
- Boss Wraith : phase 2 sous 50 % HP + invocation
- Embuscade : une goroutine envoie un tick ; le `select` interrompt le prompt et la rencontre est créée sur la goroutine principale

## Architecture

```text
cmd/grimhold/main.go     entrée (--continue, --plain, --story)
internal/ui              TUI ANSI (clear, cadres, barre HP)
internal/repl            map[string]cliCommand + aliases + Session
internal/game            état, combat, IA, packs, save/load, hooks quête
internal/quest           story.json → actes, flags, 2 fins
internal/open5e          HTTP + cache TTL (mutex + reaper)
internal/world           world.json (+ quest_hint, min_level, boss)
internal/dice            jets injectables
```

## Bonus (barème)

- **Ambush ticker** — goroutine + `select` entre la saisie et le channel. Le tick interrompt le prompt ; la rencontre est lancée sur la goroutine qui possède l’état
- **Faster dungeon** — au `travel`, prefetch concurrent des foes (`WaitGroup`). Une erreur API remonte dans le message de voyage
- **Boss region** — Flèche Lugubre verrouillée (Clé de Brume + niveau ≥ 2), boss Wraith à actions multiples, phase 2 et invocation
- **Test suite** — `go test ./...`, sans réseau (combat seedé, expiration du cache à horloge injectée, décodage Open5e via `httptest`)
- **ANSI colors** — codes échappement faits main : barre de PV (vert / jaune / rouge) et niveau de danger (vert, jaune, magenta, rouge). La largeur des cadres ignore ces codes
- Combat multi-ennemis, IA à rôles et potions (en plus du barème)

```bash
go test ./...
go vet ./...
```

## Licence

Devoir de cours — usage pédagogique.
