package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// OpenAICompatible adapts any endpoint speaking the OpenAI chat-completions
// API shape (OpenAI, Groq, Ollama, vLLM, LM Studio, ...).
type OpenAICompatible struct {
	Settings
}

// NewOpenAI builds an OpenAI-compatible adapter.
func NewOpenAI(s Settings) *OpenAICompatible { return &OpenAICompatible{Settings: s} }

// openAIBody builds the chat-completions request body for a CommitRequest.
func openAIBody(req CommitRequest) map[string]any {
	body := map[string]any{
		"model": req.Model,
		"messages": []map[string]string{
			{"role": "system", "content": req.System},
			{"role": "user", "content": req.User},
		},
		"temperature": req.Temperature,
	}
	if req.MaxOutputTokens > 0 {
		body["max_tokens"] = req.MaxOutputTokens
	}
	return body
}

// openAIHeaders returns the auth headers for an OpenAI-compatible request,
// merged with any user-supplied custom headers (custom headers win).
func openAIHeaders(s Settings) map[string]string {
	h := map[string]string{}
	if s.APIKey != "" {
		h["Authorization"] = "Bearer " + s.APIKey
	}
	for k, v := range s.Headers {
		h[k] = v
	}
	return h
}

// parseOpenAIResponse extracts the message text from a chat-completions body.
func parseOpenAIResponse(data []byte) (string, error) {
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("provider returned no choices")
	}
	msg := strings.TrimSpace(out.Choices[0].Message.Content)
	if msg == "" {
		return "", fmt.Errorf("provider returned an empty message")
	}
	return msg, nil
}

// GenerateCommitMessage implements Provider.
func (o *OpenAICompatible) GenerateCommitMessage(ctx context.Context, req CommitRequest) (string, error) {
	client, err := newHTTPClient(o.Settings)
	if err != nil {
		return "", err
	}
	data, err := doJSON(ctx, client, joinURL(o.BaseURL, "chat/completions"), openAIHeaders(o.Settings), openAIBody(req))
	if err != nil {
		return "", err
	}
	return parseOpenAIResponse(data)
}
