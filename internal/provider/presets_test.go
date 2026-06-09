package provider

import (
	"testing"

	"github.com/pocikode/commitly/internal/config"
)

func TestPresetLookup(t *testing.T) {
	for _, name := range []string{"openai", "anthropic", "gemini", "ollama", "groq"} {
		p, ok := Preset(name)
		if !ok {
			t.Errorf("preset %q missing", name)
			continue
		}
		if p.BaseURL == "" || p.Shape == "" || len(p.Models) == 0 {
			t.Errorf("preset %q incomplete: %+v", name, p)
		}
	}
	if _, ok := Preset("nope"); ok {
		t.Error("unknown preset should not be found")
	}
}

func TestPresetNamesSorted(t *testing.T) {
	names := PresetNames()
	if len(names) != 5 {
		t.Fatalf("want 5 presets, got %d", len(names))
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Errorf("preset names not sorted: %v", names)
		}
	}
}

func TestFactoryOpenAI(t *testing.T) {
	cfg := config.Defaults()
	cfg.ProviderType = config.ProviderTypeOpenAICompatible
	cfg.APIURL = "https://example.com/v1"
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(*OpenAICompatible); !ok {
		t.Errorf("want *OpenAICompatible, got %T", p)
	}
}

func TestFactoryAnthropic(t *testing.T) {
	cfg := config.Defaults()
	cfg.ProviderType = config.ProviderTypeAnthropicCompatible
	cfg.APIURL = "https://example.com/v1"
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(*AnthropicCompatible); !ok {
		t.Errorf("want *AnthropicCompatible, got %T", p)
	}
}

func TestFactoryFallsBackToPresetURL(t *testing.T) {
	cfg := config.Defaults()
	cfg.AIProvider = "groq"
	cfg.ProviderType = config.ProviderTypeOpenAICompatible
	cfg.APIURL = "" // force preset fallback
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	oa, ok := p.(*OpenAICompatible)
	if !ok {
		t.Fatalf("want *OpenAICompatible, got %T", p)
	}
	groq, _ := Preset("groq")
	if oa.BaseURL != groq.BaseURL {
		t.Errorf("base url = %q, want %q", oa.BaseURL, groq.BaseURL)
	}
}

func TestFactoryErrors(t *testing.T) {
	// No URL and unknown provider name -> error.
	cfg := config.Defaults()
	cfg.AIProvider = "mystery"
	cfg.APIURL = ""
	if _, err := New(cfg); err == nil {
		t.Error("expected error for missing base url")
	}

	// Unknown provider_type -> error.
	cfg = config.Defaults()
	cfg.APIURL = "https://x"
	cfg.ProviderType = "grpc"
	if _, err := New(cfg); err == nil {
		t.Error("expected error for unknown provider_type")
	}
}
