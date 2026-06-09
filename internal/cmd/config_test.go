package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pocikode/commitly/internal/config"
)

// press feeds a single key to the model and returns the updated model and
// whether it asked to quit.
func press(m profileTUIModel, key string) (profileTUIModel, bool) {
	var msg tea.Msg
	switch key {
	case "up", "down", "enter", "esc":
		msg = tea.KeyMsg{Type: map[string]tea.KeyType{"up": tea.KeyUp, "down": tea.KeyDown, "enter": tea.KeyEnter, "esc": tea.KeyEsc}[key]}
	default:
		msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
	next, cmd := m.Update(msg)
	nm := next.(profileTUIModel)
	quit := cmd != nil // tea.Quit is the only command this model returns
	return nm, quit
}

func newTUIModel() profileTUIModel {
	cfg := config.Config{
		ActiveProfile: "b",
		Profiles: map[string]config.Profile{
			"a": {AIProvider: "openai", Model: "gpt-4o"},
			"b": {AIProvider: "ollama", Model: "llama3.1"},
		},
	}
	return profileTUIModel{names: sortedProfileNames(cfg), cfg: cfg}
}

func TestProfileTUIKeys(t *testing.T) {
	if m := newTUIModel(); m.Init() != nil {
		t.Error("Init should return nil cmd")
	}

	// enter on the first profile (cursor 0 = "a") selects use.
	m, quit := press(newTUIModel(), "enter")
	if !quit || m.action != actionUse || m.sel != "a" {
		t.Errorf("enter: action=%q sel=%q quit=%v", m.action, m.sel, quit)
	}

	// down then edit selects the second profile.
	m, _ = press(newTUIModel(), "down")
	m, quit = press(m, "e")
	if !quit || m.action != actionEdit || m.sel != "b" {
		t.Errorf("edit: action=%q sel=%q", m.action, m.sel)
	}

	// delete on default cursor.
	m, _ = press(newTUIModel(), "d")
	if m.action != actionDelete || m.sel != "a" {
		t.Errorf("delete: action=%q sel=%q", m.action, m.sel)
	}

	// add does not need a selection.
	m, _ = press(newTUIModel(), "a")
	if m.action != actionCreate {
		t.Errorf("add: action=%q", m.action)
	}

	// q quits with no action -> runConfigManager treats as quit.
	m, quit = press(newTUIModel(), "q")
	if !quit || m.action != actionQuit {
		t.Errorf("quit: action=%q quit=%v", m.action, quit)
	}

	// cursor clamps at bounds.
	m, _ = press(newTUIModel(), "up") // already at top
	if m.cursor != 0 {
		t.Errorf("up at top: cursor=%d", m.cursor)
	}
}

func TestProfileTUIView(t *testing.T) {
	v := newTUIModel().View()
	for _, want := range []string{"> ", "* b", "a (openai/gpt-4o)", "[a]dd", "[enter]use"} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing %q:\n%s", want, v)
		}
	}

	// Empty list shows the hint.
	empty := profileTUIModel{cfg: config.Config{}}
	if !strings.Contains(empty.View(), "none yet") {
		t.Errorf("empty view: %q", empty.View())
	}
}

func TestConfigSetAndGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")

	out, err := execute(t, "config", "set", "model=gpt-4o-mini", "emoji=true", "--config", path)
	if err != nil {
		t.Fatalf("config set: %v", err)
	}
	if !strings.Contains(out, "Saved 2 value(s)") {
		t.Errorf("unexpected set output: %q", out)
	}

	out, err = execute(t, "config", "get", "model", "emoji", "--config", path)
	if err != nil {
		t.Fatalf("config get: %v", err)
	}
	if !strings.Contains(out, "model=gpt-4o-mini") || !strings.Contains(out, "emoji=true") {
		t.Errorf("unexpected get output: %q", out)
	}
}

func TestConfigGetRedactsAPIKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if _, err := execute(t, "config", "set", "api_key=sk-supersecret", "--config", path); err != nil {
		t.Fatalf("set: %v", err)
	}
	out, err := execute(t, "config", "get", "api_key", "--config", path)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.Contains(out, "supersecret") {
		t.Errorf("api_key was not redacted: %q", out)
	}
	if !strings.Contains(out, "****") {
		t.Errorf("expected redaction marker: %q", out)
	}
}

func TestRedact(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"ab":          "****",
		"sk-longsecret": "sk-l****",
	}
	for in, want := range cases {
		if got := redact(in); got != want {
			t.Errorf("redact(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestConfigSetInvalidPair(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if _, err := execute(t, "config", "set", "noequalsign", "--config", path); err == nil {
		t.Fatal("expected error for malformed pair")
	}
}

func TestConfigSetInvalidValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	if _, err := execute(t, "config", "set", "provider_type=grpc", "--config", path); err == nil {
		t.Fatal("expected validation error")
	}
}

// managerCreate runs one manager session that creates a single profile then
// quits at EOF. fields are the wizard answers after the leading "a" command:
// profile name, provider, provider_type, api_url, api_key, model.
func managerCreate(t *testing.T, path string, fields ...string) {
	t.Helper()
	script := "a\n" + strings.Join(fields, "\n") + "\n"
	if err := runConfigManager(strings.NewReader(script), &bytes.Buffer{}, path); err != nil {
		t.Fatalf("manager create %v: %v", fields, err)
	}
}

func TestConfigManagerCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	// "a" -> add; then name=work, provider=anthropic (2), type=anthropic (2),
	// blank url, key sk-key, model free text. EOF then quits the loop.
	out := &bytes.Buffer{}
	script := strings.NewReader("a\nwork\n2\n2\n\nsk-key\nclaude-3-5-sonnet-latest\n")
	if err := runConfigManager(script, out, path); err != nil {
		t.Fatalf("manager: %v", err)
	}
	if !strings.Contains(out.String(), `Saved profile "work"`) {
		t.Errorf("manager output: %q", out.String())
	}

	got, err := execute(t, "config", "get", "ai_provider", "model", "active_profile", "--config", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ai_provider=anthropic", "model=claude-3-5-sonnet-latest", "active_profile=work"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestConfigManagerCustomProvider(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	managerCreate(t, path, "local", "6", "openai_compatible", "https://llm.local/v1", "sk-c", "my-model")

	got, err := execute(t, "config", "get", "ai_provider", "api_url", "model", "--config", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ai_provider=custom", "api_url=https://llm.local/v1", "model=my-model"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestConfigManagerBlankDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	managerCreate(t, path, "", "", "", "", "", "")

	got, err := execute(t, "config", "get", "ai_provider", "provider_type", "model", "active_profile", "--config", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ai_provider=openai", "provider_type=openai_compatible", "model=gpt-4o-mini", "active_profile=default"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestConfigManagerCreateTwoThenUse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	// One session: add oai, add ollama, then pick "2" (ollama, alphabetically
	// second) to make it active and exit.
	script := strings.NewReader(
		"a\noai\n1\n1\nhttps://api.openai.com/v1\nsk-o\ngpt-4o\n" +
			"a\nollama\n4\n1\nhttp://localhost:11434/v1\n\nllama3.1\n" +
			"2\n")
	out := &bytes.Buffer{}
	if err := runConfigManager(script, out, path); err != nil {
		t.Fatalf("manager: %v", err)
	}
	if !strings.Contains(out.String(), `Active profile is now "ollama"`) {
		t.Errorf("manager output: %q", out.String())
	}

	got, err := execute(t, "config", "get", "active_profile", "ai_provider", "model", "--config", path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"active_profile=ollama", "ai_provider=ollama", "model=llama3.1"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}

	// `config profiles` lists both; ollama marked active.
	list, err := execute(t, "config", "profiles", "--config", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(list, "oai") || !strings.Contains(list, "* ollama") {
		t.Errorf("profiles output: %q", list)
	}
}

func TestConfigManagerDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	// add tmp, then "d 1" deletes it, EOF quits.
	script := strings.NewReader("a\ntmp\n1\n1\nu\nk\nm\nd 1\n")
	out := &bytes.Buffer{}
	if err := runConfigManager(script, out, path); err != nil {
		t.Fatalf("manager: %v", err)
	}
	if !strings.Contains(out.String(), `Deleted profile "tmp"`) {
		t.Errorf("manager output: %q", out.String())
	}

	list, err := execute(t, "config", "profiles", "--config", path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(list, "No profiles saved") {
		t.Errorf("expected empty profile list, got %q", list)
	}
}

func TestConfigUseUnknownProfileErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	managerCreate(t, path, "oai", "1", "1", "https://api.openai.com/v1", "sk-o", "gpt-4o")

	if _, err := execute(t, "config", "use", "nope", "--config", path); err == nil {
		t.Error("expected error for unknown profile")
	}
}
