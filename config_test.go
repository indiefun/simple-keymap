package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigPathFollowsExecutableInsteadOfWorkingDirectory(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(previous)
	path, err := configPath()
	if err != nil || path != filepath.Join(filepath.Dir(executable), "config.json") {
		t.Fatalf("configuration followed the launch directory: %s, %v", path, err)
	}
}

func TestPortableConfigMovesWithFolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	want := config{Enabled: false, Rules: []rule{{"mouse_x1", "mouse_middle", false}, {"A", "B", true}}, CheckUpdatesOnStartup: true}
	if err := saveConfig(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := loadPortableConfig(path)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("existing portable settings were not loaded")
	}
	// Simulate copying the whole folder to another location or machine.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	copyPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(copyPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	got, err = loadPortableConfig(copyPath)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("moving the folder lost rules, pause states or update preference")
	}
}

func TestPortableFirstLaunchCreatesConfigAndNeverOverwrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	got, err := loadPortableConfig(path)
	if err != nil || !got.Enabled || len(got.Rules) != 0 || got.CheckUpdatesOnStartup {
		t.Fatalf("first launch: %#v, %v", got, err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("first launch did not create a portable file")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := createConfig(path, config{Enabled: false}); !errors.Is(err, os.ErrExist) {
		t.Fatalf("initial creation replaced existing configuration: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatal("existing file was overwritten")
	}
}

func TestPortableConfigFailuresPreserveFileAndPause(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := loadPortableConfig(path)
	if err == nil || got.Enabled {
		t.Fatal("damaged local file did not pause mappings")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "broken" {
		t.Fatal("damaged portable file was overwritten")
	}
	// Deterministic write failure, without changing the user's folder ACLs.
	missingParent := filepath.Join(t.TempDir(), "missing", "config.json")
	got, err = loadPortableConfig(missingParent)
	if err == nil || got.Enabled {
		t.Fatal("failed creation did not pause mappings")
	}
}

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
