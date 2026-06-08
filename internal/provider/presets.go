package provider

import (
	"fmt"
	"sort"

	"github.com/pocikode/opencommit/internal/config"
)

// Provider-type (request-shape) constants. These mirror the config enum values
// so the factory can map config -> adapter without importing the values
// indirectly.
const (
	ProviderTypeOpenAI    = config.ProviderTypeOpenAICompatible
	ProviderTypeAnthropic = config.ProviderTypeAnthropicCompatible
)

// Preset describes a built-in named provider: its default base URL, request
// shape, and a static list of known models.
type PresetDef struct {
	Name    string
	BaseURL string
	Shape   string
	Models  []string
}

// presets is the registry of built-in named providers (PRD FR11).
var presets = map[string]PresetDef{
	"openai": {
		Name:    "openai",
		BaseURL: "https://api.openai.com/v1",
		Shape:   ProviderTypeOpenAI,
		Models:  []string{"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "o1", "o1-mini"},
	},
	"anthropic": {
		Name:    "anthropic",
		BaseURL: "https://api.anthropic.com/v1",
		Shape:   ProviderTypeAnthropic,
		Models:  []string{"claude-3-5-sonnet-latest", "claude-3-5-haiku-latest", "claude-3-opus-latest"},
	},
	"gemini": {
		Name:    "gemini",
		BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
		Shape:   ProviderTypeOpenAI,
		Models:  []string{"gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash"},
	},
	"ollama": {
		Name:    "ollama",
		BaseURL: "http://localhost:11434/v1",
		Shape:   ProviderTypeOpenAI,
		Models:  []string{"llama3.1", "qwen2.5-coder", "mistral"},
	},
	"groq": {
		Name:    "groq",
		BaseURL: "https://api.groq.com/openai/v1",
		Shape:   ProviderTypeOpenAI,
		Models:  []string{"llama-3.3-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768"},
	},
}

// Preset looks up a named preset.
func Preset(name string) (PresetDef, bool) {
	p, ok := presets[name]
	return p, ok
}

// PresetNames returns the sorted list of preset names.
func PresetNames() []string {
	out := make([]string, 0, len(presets))
	for n := range presets {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// New builds a Provider from resolved config. The base URL falls back to the
// named preset's URL when api_url is unset. The adapter is chosen by
// provider_type.
func New(cfg config.Config) (Provider, error) {
	s := Settings{
		BaseURL: cfg.APIURL,
		APIKey:  cfg.APIKey,
		Headers: cfg.APICustomHeaders,
		Proxy:   cfg.Proxy,
	}
	if s.BaseURL == "" {
		if p, ok := Preset(cfg.AIProvider); ok {
			s.BaseURL = p.BaseURL
		}
	}
	if s.BaseURL == "" {
		return nil, fmt.Errorf("no api_url set and no base URL for provider %q", cfg.AIProvider)
	}

	switch cfg.ProviderType {
	case ProviderTypeOpenAI:
		return NewOpenAI(s), nil
	case ProviderTypeAnthropic:
		return NewAnthropic(s), nil
	default:
		return nil, fmt.Errorf("unsupported provider_type %q", cfg.ProviderType)
	}
}
