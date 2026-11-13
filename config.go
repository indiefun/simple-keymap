package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type config struct {
	Enabled bool   `json:"enabled"`
	Rules   []rule `json:"rules"`
}

func validateRules(rules []rule) error {
	seen := map[string]bool{}
	for _, r := range rules {
		if _, ok := findKey(r.Source); !ok {
			return fmt.Errorf("不支持的原按键：%s", r.Source)
		}
		if _, ok := findKey(r.Target); !ok {
			return fmt.Errorf("不支持的目标按键：%s", r.Target)
		}
		if r.Source == r.Target {
			return fmt.Errorf("原按键和目标按键相同，请换一个目标按键")
		}
		if seen[r.Source] {
			return fmt.Errorf("「%s」已有映射，请先删除原规则再添加", keyLabel(r.Source))
		}
		seen[r.Source] = true
	}
	return nil
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "SimpleKeymap", "config.json"), nil
}

func loadConfig(path string) (config, error) {
	c := config{Enabled: true, Rules: []rule{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return config{Enabled: false}, err
	}
	if err = json.Unmarshal(data, &c); err != nil {
		return config{Enabled: false}, fmt.Errorf("配置文件格式有误：%w", err)
	}
	if err = validateRules(c.Rules); err != nil {
		return config{Enabled: false}, err
	}
	return c, nil
}

func saveConfig(path string, c config) error {
	if err := validateRules(c.Rules); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "config-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err = tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
