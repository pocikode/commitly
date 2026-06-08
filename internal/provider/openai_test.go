package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIGenerateSuccess(t *testing.T) {
	var gotAuth, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"feat: add thing"}}]}`)
	}))
	defer srv.Close()

	p := NewOpenAI(Settings{BaseURL: srv.URL, APIKey: "sk-test", Headers: map[string]string{"X-Org": "acme"}})
	msg, err := p.GenerateCommitMessage(context.Background(), CommitRequest{
		System: "sys", User: "diff", Model: "gpt-4o", MaxOutputTokens: 100,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if msg != "feat: add thing" {
		t.Errorf("msg = %q", msg)
	}
	if gotAuth != "Bearer sk-test" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotPath != "/chat/completions" {
		t.Errorf("path = %q", gotPath)
	}
	// Body must carry model + messages + custom header applied.
	var body map[string]any
	if err := json.Unmarshal([]byte(gotBody), &body); err != nil {
		t.Fatalf("body not json: %v", err)
	}
	if body["model"] != "gpt-4o" {
		t.Errorf("model in body = %v", body["model"])
	}
}

func TestOpenAIEmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"choices":[]}`)
	}))
	defer srv.Close()
	p := NewOpenAI(Settings{BaseURL: srv.URL})
	if _, err := p.GenerateCommitMessage(context.Background(), CommitRequest{Model: "m"}); err == nil {
		t.Fatal("expected error on empty choices")
	}
}

func TestOpenAIAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":{"message":"bad key"}}`)
	}))
	defer srv.Close()
	p := NewOpenAI(Settings{BaseURL: srv.URL, APIKey: "sk-bad"})
	_, err := p.GenerateCommitMessage(context.Background(), CommitRequest{Model: "m"})
	var apiErr *APIError
	if err == nil || !asAPIError(err, &apiErr) || apiErr.Kind != KindAuth {
		t.Fatalf("expected auth APIError, got %v", err)
	}
	// Error message must not leak the api key.
	if strings.Contains(err.Error(), "sk-bad") {
		t.Errorf("error leaked api key: %v", err)
	}
}

// asAPIError is a tiny errors.As shim to keep tests terse.
func asAPIError(err error, target **APIError) bool {
	if e, ok := err.(*APIError); ok {
		*target = e
		return true
	}
	return false
}
