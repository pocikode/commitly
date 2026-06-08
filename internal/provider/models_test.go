package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocikode/opencommit/internal/config"
)

func TestFetchModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("path = %q", r.URL.Path)
		}
		io.WriteString(w, `{"data":[{"id":"m-a"},{"id":"m-b"},{"id":""}]}`)
	}))
	defer srv.Close()

	models, err := FetchModels(context.Background(), Settings{BaseURL: srv.URL, APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0] != "m-a" || models[1] != "m-b" {
		t.Errorf("models = %v", models)
	}
}

func TestFetchModelsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := FetchModels(context.Background(), Settings{BaseURL: srv.URL}); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestListModelsMergesPresetAndEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":[{"id":"live-model"},{"id":"gpt-4o"}]}`)
	}))
	defer srv.Close()

	cfg := config.Defaults()
	cfg.AIProvider = "openai"
	cfg.ProviderType = config.ProviderTypeOpenAICompatible
	cfg.APIURL = srv.URL

	models := ListModels(context.Background(), cfg)
	// Preset models come first, live ones appended, gpt-4o not duplicated.
	if !containsStr(models, "gpt-4o") || !containsStr(models, "live-model") {
		t.Errorf("merged models missing entries: %v", models)
	}
	if countStr(models, "gpt-4o") != 1 {
		t.Errorf("gpt-4o duplicated: %v", models)
	}
}

func TestListModelsEndpointErrorFallsBackToPreset(t *testing.T) {
	cfg := config.Defaults()
	cfg.AIProvider = "anthropic" // anthropic shape -> no endpoint fetch
	cfg.ProviderType = config.ProviderTypeAnthropicCompatible
	cfg.APIURL = "http://127.0.0.1:1"

	models := ListModels(context.Background(), cfg)
	if len(models) == 0 || !containsStr(models, "claude-3-5-sonnet-latest") {
		t.Errorf("expected preset models, got %v", models)
	}
}

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func countStr(s []string, v string) int {
	n := 0
	for _, x := range s {
		if x == v {
			n++
		}
	}
	return n
}
