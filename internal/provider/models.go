package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/pocikode/commitly/internal/config"
)

// FetchModels queries an OpenAI-compatible /models endpoint and returns the
// model ids. Used as a best-effort live source on top of static preset lists.
func FetchModels(ctx context.Context, s Settings) ([]string, error) {
	client, err := newHTTPClient(s)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, joinURL(s.BaseURL, "models"), nil)
	if err != nil {
		return nil, err
	}
	for k, v := range openAIHeaders(s) {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &APIError{Kind: KindNetwork, Message: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, mapHTTPError(resp.StatusCode, nil)
	}

	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(out.Data))
	for _, m := range out.Data {
		if m.ID != "" {
			ids = append(ids, m.ID)
		}
	}
	return ids, nil
}

// ListModels returns the available models for the configured provider: the
// static preset list, merged (de-duplicated) with the live /models endpoint
// when the provider is OpenAI-compatible. Endpoint errors are non-fatal — the
// static list is still returned.
func ListModels(ctx context.Context, cfg config.Config) []string {
	set := map[string]bool{}
	var ordered []string
	add := func(m string) {
		if m != "" && !set[m] {
			set[m] = true
			ordered = append(ordered, m)
		}
	}

	for _, m := range PresetModels(cfg.AIProvider) {
		add(m)
	}

	if cfg.ProviderType == ProviderTypeOpenAI {
		s := Settings{BaseURL: cfg.APIURL, APIKey: cfg.APIKey, Headers: cfg.APICustomHeaders, Proxy: cfg.Proxy}
		if s.BaseURL == "" {
			if p, ok := Preset(cfg.AIProvider); ok {
				s.BaseURL = p.BaseURL
			}
		}
		if s.BaseURL != "" {
			if live, err := FetchModels(ctx, s); err == nil {
				sort.Strings(live)
				for _, m := range live {
					add(m)
				}
			}
		}
	}

	return ordered
}
