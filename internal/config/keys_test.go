package config

import (
	"strings"
	"testing"
)

func TestKeysCoverAllRegistryEntries(t *testing.T) {
	// Every PRD FR18 key must be registered.
	want := []string{
		"ai_provider", "provider_type", "api_key", "api_url", "model",
		"api_custom_headers", "proxy", "tokens_max_input", "tokens_max_output",
		"description", "emoji", "omit_scope", "one_line_commit", "gitpush",
		"message_template_placeholder", "prompt_module", "hook_auto_uncomment",
	}
	have := map[string]bool{}
	for _, k := range Keys() {
		have[k] = true
	}
	for _, k := range want {
		if !have[k] {
			t.Errorf("missing registered key %q", k)
		}
	}
}

func TestEveryKeyHasEnvMapping(t *testing.T) {
	for _, k := range registry {
		if !strings.HasPrefix(k.Env, "OCO_") {
			t.Errorf("key %q env var %q must start with OCO_", k.Key, k.Env)
		}
	}
}

func TestSetGetRoundTrip(t *testing.T) {
	cfg := Defaults()
	cases := map[string]string{
		"model":            "claude-3",
		"emoji":            "true",
		"tokens_max_input": "1234",
		"proxy":            "http://localhost:8080",
	}
	for k, v := range cases {
		if err := Set(&cfg, k, v); err != nil {
			t.Fatalf("set %s=%s: %v", k, v, err)
		}
		got, err := Get(&cfg, k)
		if err != nil {
			t.Fatalf("get %s: %v", k, err)
		}
		if got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestSetValidationErrors(t *testing.T) {
	cfg := Defaults()
	bad := map[string]string{
		"ai_provider":       "no-such-provider",
		"provider_type":     "grpc",
		"tokens_max_input":  "-5",
		"tokens_max_output": "abc",
		"emoji":             "maybe",
	}
	for k, v := range bad {
		if err := Set(&cfg, k, v); err == nil {
			t.Errorf("expected error for %s=%s", k, v)
		}
	}
}

func TestUnknownKey(t *testing.T) {
	cfg := Defaults()
	if err := Set(&cfg, "bogus", "x"); err == nil {
		t.Error("expected unknown key error on set")
	}
	if _, err := Get(&cfg, "bogus"); err == nil {
		t.Error("expected unknown key error on get")
	}
}

func TestCustomHeadersJSON(t *testing.T) {
	cfg := Defaults()
	if err := Set(&cfg, "api_custom_headers", `{"X-Org":"acme"}`); err != nil {
		t.Fatalf("set headers: %v", err)
	}
	if cfg.APICustomHeaders["X-Org"] != "acme" {
		t.Errorf("header not set: %+v", cfg.APICustomHeaders)
	}
	got, _ := Get(&cfg, "api_custom_headers")
	if !strings.Contains(got, "acme") {
		t.Errorf("get headers = %q", got)
	}
	// Invalid JSON rejected.
	if err := Set(&cfg, "api_custom_headers", "not-json"); err == nil {
		t.Error("expected JSON parse error")
	}
}
