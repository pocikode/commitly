package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// anthropicVersion is the required API version header value.
const anthropicVersion = "2023-06-01"

// AnthropicCompatible adapts any endpoint speaking the Anthropic messages API
// shape.
type AnthropicCompatible struct {
	Settings
}

// NewAnthropic builds an Anthropic-compatible adapter.
func NewAnthropic(s Settings) *AnthropicCompatible { return &AnthropicCompatible{Settings: s} }

// anthropicBody builds the messages request body. Anthropic requires
// max_tokens, so a positive default is enforced by the caller-supplied value.
func anthropicBody(req CommitRequest) map[string]any {
	maxTokens := req.MaxOutputTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	body := map[string]any{
		"model":      req.Model,
		"max_tokens": maxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": req.User},
		},
		"temperature": req.Temperature,
	}
	if req.System != "" {
		body["system"] = req.System
	}
	return body
}

// anthropicHeaders returns the auth/version headers, merged with custom headers.
func anthropicHeaders(s Settings) map[string]string {
	h := map[string]string{
		"anthropic-version": anthropicVersion,
	}
	if s.APIKey != "" {
		h["x-api-key"] = s.APIKey
	}
	for k, v := range s.Headers {
		h[k] = v
	}
	return h
}

// parseAnthropicResponse extracts the text from a messages-API body.
func parseAnthropicResponse(data []byte) (string, error) {
	var out struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	var b strings.Builder
	for _, c := range out.Content {
		if c.Type == "text" || c.Type == "" {
			b.WriteString(c.Text)
		}
	}
	msg := strings.TrimSpace(b.String())
	if msg == "" {
		return "", fmt.Errorf("provider returned an empty message")
	}
	return msg, nil
}

// GenerateCommitMessage implements Provider.
func (a *AnthropicCompatible) GenerateCommitMessage(ctx context.Context, req CommitRequest) (string, error) {
	client, err := newHTTPClient(a.Settings)
	if err != nil {
		return "", err
	}
	data, err := doJSON(ctx, client, joinURL(a.BaseURL, "messages"), anthropicHeaders(a.Settings), anthropicBody(req))
	if err != nil {
		return "", err
	}
	return parseAnthropicResponse(data)
}
