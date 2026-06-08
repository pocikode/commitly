package provider

import (
	"context"
	"fmt"
)

// Custom is a generic adapter for endpoints that mostly follow the OpenAI or
// Anthropic shape but need extra request fields or headers. Per PRD Q2 (v1
// default) it does minimal mapping: it reuses a base shape, merges custom
// headers, and overlays ExtraBody fields onto the request — no full template
// engine.
type Custom struct {
	Settings
	// Shape selects the base request/response mapping: "openai_compatible" or
	// "anthropic_compatible".
	Shape string
	// ExtraBody holds additional top-level JSON fields merged into the request
	// body (overriding base fields on key collision).
	ExtraBody map[string]any
}

// NewCustom builds a custom adapter over the given base shape.
func NewCustom(s Settings, shape string, extra map[string]any) *Custom {
	return &Custom{Settings: s, Shape: shape, ExtraBody: extra}
}

// GenerateCommitMessage implements Provider.
func (c *Custom) GenerateCommitMessage(ctx context.Context, req CommitRequest) (string, error) {
	client, err := newHTTPClient(c.Settings)
	if err != nil {
		return "", err
	}

	var (
		endpoint string
		headers  map[string]string
		body     map[string]any
		parse    func([]byte) (string, error)
	)

	switch c.Shape {
	case ProviderTypeAnthropic:
		endpoint = joinURL(c.BaseURL, "messages")
		headers = anthropicHeaders(c.Settings)
		body = anthropicBody(req)
		parse = parseAnthropicResponse
	case ProviderTypeOpenAI, "":
		endpoint = joinURL(c.BaseURL, "chat/completions")
		headers = openAIHeaders(c.Settings)
		body = openAIBody(req)
		parse = parseOpenAIResponse
	default:
		return "", fmt.Errorf("custom provider: unknown shape %q", c.Shape)
	}

	for k, v := range c.ExtraBody {
		body[k] = v
	}

	data, err := doJSON(ctx, client, endpoint, headers, body)
	if err != nil {
		return "", err
	}
	return parse(data)
}
