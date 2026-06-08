package config

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// KeyDef describes one configuration key: its YAML name, the matching OCO_ env
// var, and typed get/set accessors. Set parses + validates the raw string and
// assigns it to the Config; Get renders the current value as a string.
type KeyDef struct {
	Key string
	Env string
	Get func(*Config) string
	Set func(*Config, string) error
}

// registry is the single source of truth for all configuration keys. Both the
// env-override resolver and `oco config get/set` are driven by it.
var registry = []KeyDef{
	{
		Key: "ai_provider", Env: "OCO_AI_PROVIDER",
		Get: func(c *Config) string { return c.AIProvider },
		Set: func(c *Config, v string) error {
			if !contains(KnownProviders, v) {
				return fmt.Errorf("invalid ai_provider %q: must be one of %s", v, strings.Join(KnownProviders, ", "))
			}
			c.AIProvider = v
			return nil
		},
	},
	{
		Key: "provider_type", Env: "OCO_PROVIDER_TYPE",
		Get: func(c *Config) string { return c.ProviderType },
		Set: func(c *Config, v string) error {
			if v != ProviderTypeOpenAICompatible && v != ProviderTypeAnthropicCompatible {
				return fmt.Errorf("invalid provider_type %q: must be %s or %s", v, ProviderTypeOpenAICompatible, ProviderTypeAnthropicCompatible)
			}
			c.ProviderType = v
			return nil
		},
	},
	{
		Key: "api_key", Env: "OCO_API_KEY",
		Get: func(c *Config) string { return c.APIKey },
		Set: func(c *Config, v string) error { c.APIKey = v; return nil },
	},
	{
		Key: "api_url", Env: "OCO_API_URL",
		Get: func(c *Config) string { return c.APIURL },
		Set: func(c *Config, v string) error { c.APIURL = v; return nil },
	},
	{
		Key: "model", Env: "OCO_MODEL",
		Get: func(c *Config) string { return c.Model },
		Set: func(c *Config, v string) error { c.Model = v; return nil },
	},
	{
		Key: "api_custom_headers", Env: "OCO_API_CUSTOM_HEADERS",
		Get: func(c *Config) string {
			if len(c.APICustomHeaders) == 0 {
				return "{}"
			}
			b, _ := json.Marshal(c.APICustomHeaders)
			return string(b)
		},
		Set: func(c *Config, v string) error {
			m := map[string]string{}
			if strings.TrimSpace(v) != "" {
				if err := json.Unmarshal([]byte(v), &m); err != nil {
					return fmt.Errorf("invalid api_custom_headers: expected JSON object: %w", err)
				}
			}
			c.APICustomHeaders = m
			return nil
		},
	},
	{
		Key: "proxy", Env: "OCO_PROXY",
		Get: func(c *Config) string { return c.Proxy },
		Set: func(c *Config, v string) error { c.Proxy = v; return nil },
	},
	{
		Key: "tokens_max_input", Env: "OCO_TOKENS_MAX_INPUT",
		Get: func(c *Config) string { return strconv.Itoa(c.TokensMaxInput) },
		Set: func(c *Config, v string) error {
			n, err := parsePositiveInt("tokens_max_input", v)
			if err != nil {
				return err
			}
			c.TokensMaxInput = n
			return nil
		},
	},
	{
		Key: "tokens_max_output", Env: "OCO_TOKENS_MAX_OUTPUT",
		Get: func(c *Config) string { return strconv.Itoa(c.TokensMaxOutput) },
		Set: func(c *Config, v string) error {
			n, err := parsePositiveInt("tokens_max_output", v)
			if err != nil {
				return err
			}
			c.TokensMaxOutput = n
			return nil
		},
	},
	boolKey("description", "OCO_DESCRIPTION", func(c *Config) *bool { return &c.Description }),
	boolKey("emoji", "OCO_EMOJI", func(c *Config) *bool { return &c.Emoji }),
	boolKey("omit_scope", "OCO_OMIT_SCOPE", func(c *Config) *bool { return &c.OmitScope }),
	boolKey("one_line_commit", "OCO_ONE_LINE_COMMIT", func(c *Config) *bool { return &c.OneLineCommit }),
	boolKey("gitpush", "OCO_GITPUSH", func(c *Config) *bool { return &c.GitPush }),
	{
		Key: "message_template_placeholder", Env: "OCO_MESSAGE_TEMPLATE_PLACEHOLDER",
		Get: func(c *Config) string { return c.MessageTemplatePlaceholder },
		Set: func(c *Config, v string) error { c.MessageTemplatePlaceholder = v; return nil },
	},
	{
		Key: "prompt_module", Env: "OCO_PROMPT_MODULE",
		Get: func(c *Config) string { return c.PromptModule },
		Set: func(c *Config, v string) error { c.PromptModule = v; return nil },
	},
	boolKey("hook_auto_uncomment", "OCO_HOOK_AUTO_UNCOMMENT", func(c *Config) *bool { return &c.HookAutoUncomment }),
}

// keyIndex maps key name -> KeyDef for O(1) lookup.
var keyIndex = func() map[string]KeyDef {
	m := make(map[string]KeyDef, len(registry))
	for _, k := range registry {
		m[k.Key] = k
	}
	return m
}()

// Keys returns the sorted list of known configuration key names.
func Keys() []string {
	out := make([]string, 0, len(registry))
	for _, k := range registry {
		out = append(out, k.Key)
	}
	sort.Strings(out)
	return out
}

// Lookup returns the KeyDef for a key name and whether it exists.
func Lookup(key string) (KeyDef, bool) {
	d, ok := keyIndex[key]
	return d, ok
}

// Get returns the string value of key from cfg. Unknown keys error.
func Get(cfg *Config, key string) (string, error) {
	d, ok := Lookup(key)
	if !ok {
		return "", unknownKeyErr(key)
	}
	return d.Get(cfg), nil
}

// Set parses and assigns value to key on cfg, validating it. Unknown keys
// error.
func Set(cfg *Config, key, value string) error {
	d, ok := Lookup(key)
	if !ok {
		return unknownKeyErr(key)
	}
	return d.Set(cfg, value)
}

// boolKey builds a KeyDef for a boolean field via a pointer accessor.
func boolKey(key, env string, ptr func(*Config) *bool) KeyDef {
	return KeyDef{
		Key: key, Env: env,
		Get: func(c *Config) string { return strconv.FormatBool(*ptr(c)) },
		Set: func(c *Config, v string) error {
			b, err := strconv.ParseBool(strings.TrimSpace(v))
			if err != nil {
				return fmt.Errorf("invalid %s %q: expected true or false", key, v)
			}
			*ptr(c) = b
			return nil
		},
	}
}

func parsePositiveInt(key, v string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q: expected an integer", key, v)
	}
	if n <= 0 {
		return 0, fmt.Errorf("invalid %s %q: must be greater than 0", key, v)
	}
	return n, nil
}

func unknownKeyErr(key string) error {
	return fmt.Errorf("unknown config key %q: valid keys are %s", key, strings.Join(Keys(), ", "))
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
