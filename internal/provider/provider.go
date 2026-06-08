// Package provider defines the AI provider abstraction and HTTP adapters
// (OpenAI-compatible, Anthropic-compatible, and a generic custom adapter) used
// to generate commit messages. Adapters never log or echo the API key.
package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// CommitRequest carries everything an adapter needs for one generation call.
type CommitRequest struct {
	System          string  // system prompt
	User            string  // user prompt (diff + instructions)
	Model           string  // model id
	MaxOutputTokens int     // upper bound on generated tokens
	Temperature     float64 // sampling temperature
}

// Provider generates a commit message from a request. Implementations talk to a
// specific API shape.
type Provider interface {
	GenerateCommitMessage(ctx context.Context, req CommitRequest) (string, error)
}

// Settings holds the transport-level configuration shared by all adapters.
type Settings struct {
	BaseURL string
	APIKey  string
	Headers map[string]string
	Proxy   string
}

// ErrorKind classifies provider failures for human-readable reporting.
type ErrorKind string

const (
	KindAuth    ErrorKind = "auth"
	KindRate    ErrorKind = "rate_limit"
	KindNetwork ErrorKind = "network"
	KindAPI     ErrorKind = "api"
)

// APIError is a classified provider error. Its message never contains the API
// key.
type APIError struct {
	Kind       ErrorKind
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	switch e.Kind {
	case KindAuth:
		return "authentication failed: check your api_key (" + e.Message + ")"
	case KindRate:
		return "rate limited by provider: slow down or check your quota (" + e.Message + ")"
	case KindNetwork:
		return "network error contacting provider: " + e.Message
	default:
		return fmt.Sprintf("provider error (HTTP %d): %s", e.StatusCode, e.Message)
	}
}

// newHTTPClient builds an HTTP client honoring an optional proxy.
func newHTTPClient(s Settings) (*http.Client, error) {
	tr := &http.Transport{}
	if s.Proxy != "" {
		pu, err := url.Parse(s.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy url: %w", err)
		}
		tr.Proxy = http.ProxyURL(pu)
	}
	return &http.Client{Timeout: 60 * time.Second, Transport: tr}, nil
}

// joinURL joins a base URL and path, tolerating a trailing slash on base.
func joinURL(base, path string) string {
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}

// doJSON marshals body, POSTs it to url with the given headers, and returns the
// raw response bytes. HTTP/transport errors are mapped to *APIError. The caller
// supplies auth headers; this function never inspects the API key.
func doJSON(ctx context.Context, client *http.Client, endpoint string, headers map[string]string, body any) ([]byte, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, &APIError{Kind: KindNetwork, Message: err.Error()}
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return nil, mapHTTPError(resp.StatusCode, data)
	}
	return data, nil
}

// mapHTTPError classifies an HTTP error status into an *APIError.
func mapHTTPError(status int, body []byte) error {
	msg := extractErrMessage(body)
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return &APIError{Kind: KindAuth, StatusCode: status, Message: msg}
	case status == http.StatusTooManyRequests:
		return &APIError{Kind: KindRate, StatusCode: status, Message: msg}
	default:
		return &APIError{Kind: KindAPI, StatusCode: status, Message: msg}
	}
}

// extractErrMessage pulls a human message out of a provider error body, falling
// back to a truncated raw body. It looks for the common {"error":{"message"}}
// and {"error":"..."} shapes.
func extractErrMessage(body []byte) string {
	var probe struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(body, &probe); err == nil && len(probe.Error) > 0 {
		var obj struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(probe.Error, &obj) == nil && obj.Message != "" {
			return obj.Message
		}
		var s string
		if json.Unmarshal(probe.Error, &s) == nil && s != "" {
			return s
		}
	}
	t := strings.TrimSpace(string(body))
	if len(t) > 200 {
		t = t[:200]
	}
	if t == "" {
		return "no response body"
	}
	return t
}
