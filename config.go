package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	CheckmarkChecked   string            `toml:"checkmark_checked"`
	CheckmarkUnchecked string            `toml:"checkmark_unchecked"`
	MenuIndicator      string            `toml:"menu_indicator"`
	MenuSeparator      string            `toml:"menu_separator"`
	Names              map[string]string `toml:"names"`
	Separator          string            `toml:"separator"`
	ShowChildren       bool              `toml:"show_children"`
	ShowParent         bool              `toml:"show_parent"`
}

func defaultConfig() Config {
	return Config{
		CheckmarkChecked:   "[x]",
		CheckmarkUnchecked: "[ ]",
		MenuIndicator:      ">",
		MenuSeparator:      ">",
		Names:              map[string]string{},
		Separator:          "---",
		ShowChildren:       true,
		ShowParent:         false,
	}
}

func defaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}

	return filepath.Join(configDir, "systray", "config.toml"), nil
}

func loadConfig(path string) (Config, error) {

	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Names == nil {
		cfg.Names = map[string]string{}
	}

	return cfg, nil
}

func loadRuntimeConfig() Config {
	cfg := defaultConfig()

	path, err := defaultConfigPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "systray: unable to resolve config path: %v\n", err)
		return cfg
	}

	loadedCfg, err := loadConfig(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "systray: unable to load config from %s: %v\n", path, err)
		return cfg
	}

	return loadedCfg
}
