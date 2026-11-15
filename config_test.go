package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigRoundTripAndReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings", "config.json")
	want := config{Enabled: true, Rules: []rule{{"mouse_x1", "mouse_middle", true}}, CheckUpdatesOnStartup: true}
	for _, enabled := range []bool{true, false} {
		want.Enabled = enabled
		if err := saveConfig(path, want); err != nil {
			t.Fatal(err)
		}
		got, err := loadConfig(path)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("load = %#v, %v; want %#v", got, err, want)
		}
	}
	matches, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "*.tmp"))
	if len(matches) != 0 {
		t.Fatalf("temporary files left behind: %v", matches)
	}
}

func TestInvalidConfigIsNotOverwrittenOnLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{"enabled":true,"rules":[{"source":"invalid","target":"A","enabled":true}]}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	c, err := loadConfig(path)
	if err == nil || c.Enabled {
		t.Fatalf("invalid config must fail paused: %#v, %v", c, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(data) {
		t.Fatal("loading must preserve invalid file")
	}
}

func TestRulesRejectUnknownSameAndDuplicateKeys(t *testing.T) {
	for _, rules := range [][]rule{
		{{"missing", "A", true}}, {{"A", "missing", true}},
		{{"A", "A", true}}, {{"A", "B", true}, {"A", "C", false}},
	} {
		if validateRules(rules) == nil {
			t.Fatalf("accepted invalid rules: %#v", rules)
		}
	}
}

func TestMissingConfigDefaultsToNoRules(t *testing.T) {
	c, err := loadConfig(filepath.Join(t.TempDir(), "config.json"))
	if err != nil || !c.Enabled || len(c.Rules) != 0 || c.CheckUpdatesOnStartup {
		t.Fatalf("unexpected defaults: %#v, %v", c, err)
	}
}

func TestUnreadableConfigDefaultsToPaused(t *testing.T) {
	c, err := loadConfig(t.TempDir()) // A directory cannot be read as config JSON.
	if err == nil || c.Enabled {
		t.Fatalf("failed config read must pause: %#v, %v", c, err)
	}
}

func TestKeyCatalogHasUniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, k := range keys {
		if seen[k.ID] || k.ID == "" || k.Label == "" {
			t.Fatalf("invalid key: %#v", k)
		}
		seen[k.ID] = true
	}
}
