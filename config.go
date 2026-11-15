package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type config struct {
	Enabled               bool   `json:"enabled"`
	Rules                 []rule `json:"rules"`
	CheckUpdatesOnStartup bool   `json:"checkUpdatesOnStartup,omitempty"`
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
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(executable), "config.json"), nil
}

func loadPortableConfig(path string) (config, error) {
	if _, err := os.Lstat(path); err == nil {
		return loadConfig(path)
	} else if !os.IsNotExist(err) {
		return config{Enabled: false}, err
	}
	c := config{Enabled: true, Rules: []rule{}}
	if err := createConfig(path, c); err != nil {
		if os.IsExist(err) {
			// Another writer created it after the existence check. Respect it.
			return loadConfig(path)
		}
		return config{Enabled: false}, fmt.Errorf("无法在软件同目录创建 config.json，请将整个软件文件夹移到可写位置后重试：%w", err)
	}
	return c, nil
}

// Initial creation must never replace an existing portable configuration.
// Later user edits use saveConfig's temporary file and atomic replacement.
func createConfig(path string, c config) error {
	data, err := configData(c)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err != nil {
		file.Close()
		os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(path)
		return err
	}
	return nil
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

func configData(c config) ([]byte, error) {
	if err := validateRules(c.Rules); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func saveConfig(path string, c config) error {
	data, err := configData(c)
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
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
