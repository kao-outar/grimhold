package open5e

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const defaultBaseURL = "https://api.open5e.com/v1"

type MonsterAction struct {
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	AttackBonus int    `json:"attack_bonus"`
	DamageDice  string `json:"damage_dice"`
	DamageBonus int    `json:"damage_bonus"`
}

type Monster struct {
	Slug            string          `json:"slug"`
	Name            string          `json:"name"`
	Size            string          `json:"size"`
	Type            string          `json:"type"`
	Alignment       string          `json:"alignment"`
	ArmorClass      int             `json:"armor_class"`
	ArmorDesc       string          `json:"armor_desc"`
	HitPoints       int             `json:"hit_points"`
	HitDice         string          `json:"hit_dice"`
	ChallengeRating string          `json:"challenge_rating"`
	CR              float64         `json:"cr"`
	Actions         []MonsterAction `json:"actions"`
}

type Page struct {
	Count    int       `json:"count"`
	Next     *string   `json:"next"`
	Previous *string   `json:"previous"`
	Results  []Monster `json:"results"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Cache      *Cache
}

func NewClient(cache *Cache) *Client {
	return &Client{
		BaseURL: defaultBaseURL,
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		Cache: cache,
	}
}

func (c *Client) getJSON(rawURL string, dest any) error {
	if c.Cache != nil {
		if data, ok := c.Cache.Get(rawURL); ok {
			if err := json.Unmarshal(data, dest); err != nil {
				return fmt.Errorf("decode cached %s: %w", rawURL, err)
			}
			return nil
		}
	}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("GET %s: %w", rawURL, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: status %d: %s", rawURL, resp.StatusCode, truncate(string(body), 200))
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	if c.Cache != nil {
		c.Cache.Set(rawURL, body)
	}
	return nil
}

func (c *Client) ListMonsters(pageURL string) (*Page, error) {
	if pageURL == "" {
		pageURL = c.BaseURL + "/monsters/?limit=10"
	}
	var page Page
	if err := c.getJSON(pageURL, &page); err != nil {
		return nil, fmt.Errorf("list monsters: %w", err)
	}
	return &page, nil
}

func (c *Client) GetMonster(slug string) (*Monster, error) {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return nil, fmt.Errorf("monster slug is empty")
	}
	u := c.BaseURL + "/monsters/" + url.PathEscape(slug) + "/"
	var m Monster
	if err := c.getJSON(u, &m); err != nil {
		return nil, fmt.Errorf("get monster %q: %w", slug, err)
	}
	return &m, nil
}

func (c *Client) PrefetchMonsters(slugs []string) error {
	if len(slugs) == 0 {
		return nil
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error
	for _, slug := range slugs {
		wg.Add(1)
		s := slug
		go func() {
			defer wg.Done()
			if _, err := c.GetMonster(s); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("prefetch foes: %w", errors.Join(errs...))
}

func (m *Monster) PrimaryAttack() (MonsterAction, bool) {
	for _, a := range m.Actions {
		if a.AttackBonus != 0 || a.DamageDice != "" {
			return a, true
		}
	}
	if len(m.Actions) > 0 {
		return m.Actions[0], true
	}
	return MonsterAction{}, false
}

func (m *Monster) FormatStatBlock() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s %s)\n", m.Name, m.Size, m.Type)
	fmt.Fprintf(&b, "  AC %d", m.ArmorClass)
	if m.ArmorDesc != "" {
		fmt.Fprintf(&b, " (%s)", m.ArmorDesc)
	}
	fmt.Fprintf(&b, "  HP %d", m.HitPoints)
	if m.HitDice != "" {
		fmt.Fprintf(&b, " (%s)", m.HitDice)
	}
	fmt.Fprintf(&b, "\n  CR %s\n", m.ChallengeRating)
	if len(m.Actions) > 0 {
		b.WriteString("  Actions:\n")
		for _, a := range m.Actions {
			fmt.Fprintf(&b, "    - %s", a.Name)
			if a.Desc != "" {
				fmt.Fprintf(&b, ": %s", a.Desc)
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
