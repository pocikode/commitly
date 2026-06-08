package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropicGenerateSuccess(t *testing.T) {
	var gotKey, gotVersion, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		gotVersion = r.Header.Get("anthropic-version")
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"content":[{"type":"text","text":"fix: bug"}]}`)
	}))
	defer srv.Close()

	p := NewAnthropic(Settings{BaseURL: srv.URL, APIKey: "ak-test"})
	msg, err := p.GenerateCommitMessage(context.Background(), CommitRequest{
		System: "sys", User: "diff", Model: "claude-3-5-sonnet-latest", MaxOutputTokens: 256,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if msg != "fix: bug" {
		t.Errorf("msg = %q", msg)
	}
	if gotKey != "ak-test" {
		t.Errorf("x-api-key = %q", gotKey)
	}
	if gotVersion != anthropicVersion {
		t.Errorf("anthropic-version = %q", gotVersion)
	}
	if gotPath != "/messages" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody["system"] != "sys" {
		t.Errorf("system in body = %v", gotBody["system"])
	}
}

func TestAnthropicDefaultsMaxTokens(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"content":[{"type":"text","text":"x"}]}`)
	}))
	defer srv.Close()
	p := NewAnthropic(Settings{BaseURL: srv.URL})
	if _, err := p.GenerateCommitMessage(context.Background(), CommitRequest{Model: "m"}); err != nil {
		t.Fatal(err)
	}
	if gotBody["max_tokens"].(float64) <= 0 {
		t.Errorf("max_tokens should default positive, got %v", gotBody["max_tokens"])
	}
}

func TestAnthropicRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"error":{"message":"slow down"}}`)
	}))
	defer srv.Close()
	p := NewAnthropic(Settings{BaseURL: srv.URL})
	_, err := p.GenerateCommitMessage(context.Background(), CommitRequest{Model: "m"})
	var apiErr *APIError
	if err == nil || !asAPIError(err, &apiErr) || apiErr.Kind != KindRate {
		t.Fatalf("expected rate-limit APIError, got %v", err)
	}
}

func TestAnthropicEmptyMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"content":[]}`)
	}))
	defer srv.Close()
	p := NewAnthropic(Settings{BaseURL: srv.URL})
	if _, err := p.GenerateCommitMessage(context.Background(), CommitRequest{Model: "m"}); err == nil {
		t.Fatal("expected empty-message error")
	}
}
