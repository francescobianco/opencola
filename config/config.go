package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type ProviderConfig struct {
	Name     string `json:"name"`
	BaseURL  string `json:"base_url"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	Provider string `json:"provider"`
}

type Config struct {
	Providers   []ProviderConfig `json:"providers"`
	ActiveIndex int              `json:"active_index"`
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".opencola.json"
	}
	return filepath.Join(home, ".config", "opencola", "config.json")
}

func DefaultHistoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".opencola_history"
	}
	return filepath.Join(home, ".config", "opencola", "history")
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Config{}, nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &Config{}, nil
	}
	return &cfg, nil
}

func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c *Config) ActiveProvider() *ProviderConfig {
	if len(c.Providers) == 0 || c.ActiveIndex >= len(c.Providers) {
		return nil
	}
	return &c.Providers[c.ActiveIndex]
}

func (c *Config) AddProvider(p ProviderConfig) {
	c.Providers = append(c.Providers, p)
	c.ActiveIndex = len(c.Providers) - 1
}

func (c *Config) ListProviders() []ProviderConfig {
	return c.Providers
}
