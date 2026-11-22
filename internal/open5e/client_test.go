package open5e

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetMonsterHTTPtest(t *testing.T) {
	const body = `{
		"slug": "goblin",
		"name": "Goblin",
		"size": "Small",
		"type": "Humanoid",
		"alignment": "neutral evil",
		"armor_class": 15,
		"armor_desc": "leather armor, shield",
		"hit_points": 7,
		"hit_dice": "2d6",
		"challenge_rating": "1/4",
		"cr": 0.25,
		"actions": [
			{
				"name": "Scimitar",
				"desc": "Melee Weapon Attack: +4 to hit.",
				"attack_bonus": 4,
				"damage_dice": "1d6",
				"damage_bonus": 2
			}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/monsters/goblin/" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	cache := newCache(time.Minute, time.Hour, realClock{})
	defer cache.Close()
	client := NewClient(cache)
	client.BaseURL = srv.URL
	client.HTTPClient = srv.Client()

	m, err := client.GetMonster("goblin")
	if err != nil {
		t.Fatal(err)
	}
	if m.Name != "Goblin" || m.ArmorClass != 15 || m.HitPoints != 7 {
		t.Fatalf("unexpected monster: %+v", m)
	}
	atk, ok := m.PrimaryAttack()
	if !ok || atk.AttackBonus != 4 {
		t.Fatalf("primary attack: %+v ok=%v", atk, ok)
	}

	m2, err := client.GetMonster("goblin")
	if err != nil || m2.Name != "Goblin" {
		t.Fatalf("cached get failed: %v %+v", err, m2)
	}
}

func TestPrefetchReportsErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cache := newCache(time.Minute, time.Hour, realClock{})
	defer cache.Close()
	client := NewClient(cache)
	client.BaseURL = srv.URL
	client.HTTPClient = srv.Client()

	err := client.PrefetchMonsters([]string{"goblin", "skeleton"})
	if err == nil {
		t.Fatal("expected prefetch error")
	}
}
