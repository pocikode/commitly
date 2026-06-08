// Package config defines the OpenCommit-Go configuration model, its on-disk
// YAML representation, the key registry (YAML key <-> OCO_ env var), and the
// precedence resolver (flag > env > project > global > default).
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Provider-type enum values.
const (
	ProviderTypeOpenAICompatible    = "openai_compatible"
	ProviderTypeAnthropicCompatible = "anthropic_compatible"
)

// Known named providers (presets) plus the custom escape hatch. Used to
// validate the ai_provider key. The provider package owns the full preset
// metadata (task 3.0); this list only gates config validation.
var KnownProviders = []string{"openai", "anthropic", "gemini", "ollama", "groq", "custom"}

// Profile is a named bundle of provider/model credentials. Users define several
// (e.g. "openai", "local-ollama") and switch the active one with
// `oco config use <name>`. The active profile's fields overlay the top-level
// provider fields during Resolve.
type Profile struct {
	AIProvider   string `yaml:"ai_provider"`
	ProviderType string `yaml:"provider_type"`
	APIKey       string `yaml:"api_key"`
	APIURL       string `yaml:"api_url"`
	Model        string `yaml:"model"`
}

// Config is the full set of OpenCommit-Go settings. Field order mirrors the
// YAML keys are lowercase snake_case.
type Config struct {
	AIProvider                 string             `yaml:"ai_provider"`
	ProviderType               string             `yaml:"provider_type"`
	APIKey                     string             `yaml:"api_key"`
	APIURL                     string             `yaml:"api_url"`
	Model                      string             `yaml:"model"`
	APICustomHeaders           map[string]string  `yaml:"api_custom_headers"`
	Proxy                      string             `yaml:"proxy"`
	TokensMaxInput             int                `yaml:"tokens_max_input"`
	TokensMaxOutput            int                `yaml:"tokens_max_output"`
	Description                bool               `yaml:"description"`
	Emoji                      bool               `yaml:"emoji"`
	OmitScope                  bool               `yaml:"omit_scope"`
	OneLineCommit              bool               `yaml:"one_line_commit"`
	GitPush                    bool               `yaml:"gitpush"`
	MessageTemplatePlaceholder string             `yaml:"message_template_placeholder"`
	PromptModule               string             `yaml:"prompt_module"`
	HookAutoUncomment          bool               `yaml:"hook_auto_uncomment"`
	ActiveProfile              string             `yaml:"active_profile"`
	Profiles                   map[string]Profile `yaml:"profiles,omitempty"`
}

// ProfileFromConfig snapshots the top-level provider fields into a Profile.
func ProfileFromConfig(c *Config) Profile {
	return Profile{
		AIProvider:   c.AIProvider,
		ProviderType: c.ProviderType,
		APIKey:       c.APIKey,
		APIURL:       c.APIURL,
		Model:        c.Model,
	}
}

// ApplyProfile overlays a profile's non-empty provider fields onto the
// top-level provider fields of cfg.
func ApplyProfile(cfg *Config, p Profile) {
	if p.AIProvider != "" {
		cfg.AIProvider = p.AIProvider
	}
	if p.ProviderType != "" {
		cfg.ProviderType = p.ProviderType
	}
	if p.APIKey != "" {
		cfg.APIKey = p.APIKey
	}
	if p.APIURL != "" {
		cfg.APIURL = p.APIURL
	}
	if p.Model != "" {
		cfg.Model = p.Model
	}
}

// Defaults returns a Config populated with the built-in default values. This
// is the base layer of the precedence chain.
func Defaults() Config {
	return Config{
		AIProvider:                 "openai",
		ProviderType:               ProviderTypeOpenAICompatible,
		APIKey:                     "",
		APIURL:                     "https://api.openai.com/v1",
		Model:                      "gpt-4o-mini",
		APICustomHeaders:           map[string]string{},
		Proxy:                      "",
		TokensMaxInput:             40960,
		TokensMaxOutput:            4096,
		Description:                false,
		Emoji:                      false,
		OmitScope:                  false,
		OneLineCommit:              false,
		GitPush:                    false,
		MessageTemplatePlaceholder: "$msg",
		PromptModule:               "conventional-commit",
		HookAutoUncomment:          false,
	}
}

// Load reads and parses a YAML config file, layered on top of Defaults so that
// keys absent from the file keep their default value. A missing file is not an
// error: it returns Defaults. Parse errors are returned.
func Load(path string) (Config, error) {
	cfg := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// merge overlays any keys present in the YAML at path onto cfg, leaving keys
// absent from the file untouched. Used to layer project-local config over
// global config. A missing file is a no-op.
func merge(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	return nil
}

// Save writes cfg as YAML to path with 0600 (user-only) permissions, creating
// parent directories as needed.
func Save(cfg Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create config dir %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}
