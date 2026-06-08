package provider

import (
	"context"
	"strings"
	"testing"
)

func TestNetworkError(t *testing.T) {
	// Unroutable/closed address -> network APIError.
	p := NewOpenAI(Settings{BaseURL: "http://127.0.0.1:1"})
	_, err := p.GenerateCommitMessage(context.Background(), CommitRequest{Model: "m"})
	var apiErr *APIError
	if err == nil || !asAPIError(err, &apiErr) || apiErr.Kind != KindNetwork {
		t.Fatalf("expected network APIError, got %v", err)
	}
}

func TestInvalidProxy(t *testing.T) {
	p := NewOpenAI(Settings{BaseURL: "http://x", Proxy: "://bad"})
	if _, err := p.GenerateCommitMessage(context.Background(), CommitRequest{Model: "m"}); err == nil {
		t.Fatal("expected proxy parse error")
	}
}

func TestExtractErrMessage(t *testing.T) {
	cases := map[string]string{
		`{"error":{"message":"nested"}}`: "nested",
		`{"error":"flat"}`:               "flat",
		`plain text`:                     "plain text",
		``:                               "no response body",
	}
	for body, want := range cases {
		if got := extractErrMessage([]byte(body)); got != want {
			t.Errorf("extractErrMessage(%q) = %q, want %q", body, got, want)
		}
	}
}

func TestAPIErrorMessagesDistinct(t *testing.T) {
	kinds := []ErrorKind{KindAuth, KindRate, KindNetwork, KindAPI}
	seen := map[string]bool{}
	for _, k := range kinds {
		e := &APIError{Kind: k, StatusCode: 500, Message: "m"}
		msg := e.Error()
		if msg == "" || seen[msg] {
			t.Errorf("kind %s produced empty/dup message %q", k, msg)
		}
		seen[msg] = true
	}
}

func TestJoinURL(t *testing.T) {
	if got := joinURL("http://x/v1/", "/chat"); got != "http://x/v1/chat" {
		t.Errorf("joinURL = %q", got)
	}
	if !strings.HasSuffix(joinURL("http://x/v1", "messages"), "/v1/messages") {
		t.Errorf("joinURL suffix wrong")
	}
}
