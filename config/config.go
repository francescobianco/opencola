package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
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

type EnvConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".opencola.json"
	}
	return filepath.Join(home, ".config", "opencola", "config.json")
}

func DefaultEnvPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".opencolarc"
	}
	return filepath.Join(home, ".opencolarc")
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
	if err := unmarshalJSON(data, &cfg); err != nil {
		return &Config{}, nil
	}
	return &cfg, nil
}

func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := marshalJSON(c)
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

func LoadEnv(path string) *EnvConfig {
	cfg := &EnvConfig{}

	f, err := os.Open(path)
	if err != nil {
		return cfg
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, "\"'")

		switch key {
		case "OPENAI_API_KEY", "API_KEY":
			cfg.APIKey = val
		case "OPENAI_BASE_URL", "BASE_URL":
			cfg.BaseURL = val
		case "OPENAI_MODEL", "MODEL":
			cfg.Model = val
		}
	}

	return cfg
}

func (e *EnvConfig) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString("# OpenCola configuration (.env format)\n")
	if e.APIKey != "" {
		f.WriteString("OPENAI_API_KEY=" + e.APIKey + "\n")
	}
	if e.BaseURL != "" {
		f.WriteString("OPENAI_BASE_URL=" + e.BaseURL + "\n")
	}
	if e.Model != "" {
		f.WriteString("OPENAI_MODEL=" + e.Model + "\n")
	}

	return nil
}
