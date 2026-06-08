package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCustomOpenAIShapeWithExtraBody(t *testing.T) {
	var gotBody map[string]any
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"choices":[{"message":{"content":"chore: x"}}]}`)
	}))
	defer srv.Close()

	c := NewCustom(Settings{BaseURL: srv.URL}, ProviderTypeOpenAI, map[string]any{"top_p": 0.9})
	msg, err := c.GenerateCommitMessage(context.Background(), CommitRequest{Model: "m", User: "d"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if msg != "chore: x" {
		t.Errorf("msg = %q", msg)
	}
	if gotPath != "/chat/completions" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody["top_p"].(float64) != 0.9 {
		t.Errorf("extra body not merged: %v", gotBody["top_p"])
	}
}

func TestCustomAnthropicShape(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		io.WriteString(w, `{"content":[{"type":"text","text":"docs: y"}]}`)
	}))
	defer srv.Close()
	c := NewCustom(Settings{BaseURL: srv.URL}, ProviderTypeAnthropic, nil)
	msg, err := c.GenerateCommitMessage(context.Background(), CommitRequest{Model: "m", User: "d"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if msg != "docs: y" || gotPath != "/messages" {
		t.Errorf("msg=%q path=%q", msg, gotPath)
	}
}

func TestCustomUnknownShape(t *testing.T) {
	c := NewCustom(Settings{BaseURL: "http://x"}, "weird", nil)
	if _, err := c.GenerateCommitMessage(context.Background(), CommitRequest{Model: "m"}); err == nil {
		t.Fatal("expected unknown-shape error")
	}
}
