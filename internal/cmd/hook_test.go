package cmd

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeProviderConfig writes a config file pointing at a mock OpenAI endpoint
// and sets it as the active config path for the duration of the test.
func writeProviderConfig(t *testing.T, baseURL string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cfg.yaml")
	content := "ai_provider: openai\nprovider_type: openai_compatible\napi_url: " + baseURL + "\napi_key: sk-test\nmodel: gpt-4o\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	prev := flagConfig
	flagConfig = path
	t.Cleanup(func() { flagConfig = prev })
}

func TestHookRuntimeGeneratesMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"choices":[{"message":{"content":"feat: hooktest"}}]}`)
	}))
	defer srv.Close()
	writeProviderConfig(t, srv.URL)

	dir := gitInit(t)
	write(t, dir, "a.go", "package main\n")
	stage(t, dir)
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	msgFile := filepath.Join(dir, "COMMIT_EDITMSG")
	if err := os.WriteFile(msgFile, []byte("# template\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := runHookRuntime(context.Background(), &bytes.Buffer{}, msgFile, ""); err != nil {
		t.Fatalf("runtime: %v", err)
	}
	data, _ := os.ReadFile(msgFile)
	if !strings.HasPrefix(string(data), "feat: hooktest") {
		t.Errorf("commit message not written: %q", data)
	}
}

func TestHookRuntimeDegradesOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":{"message":"bad"}}`)
	}))
	defer srv.Close()
	writeProviderConfig(t, srv.URL)

	dir := gitInit(t)
	write(t, dir, "a.go", "x\n")
	stage(t, dir)
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	msgFile := filepath.Join(dir, "COMMIT_EDITMSG")
	os.WriteFile(msgFile, []byte("# template\n"), 0o644)

	var errBuf bytes.Buffer
	// Must not block (returns nil) but should warn.
	if err := runHookRuntime(context.Background(), &errBuf, msgFile, ""); err != nil {
		t.Fatalf("hook should degrade gracefully, got %v", err)
	}
	if !strings.Contains(errBuf.String(), "cly hook:") {
		t.Errorf("expected warning, got %q", errBuf.String())
	}
}

func TestHookSetUnsetViaCommand(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := execute(t, "hook", "set"); err != nil {
		t.Fatalf("hook set: %v", err)
	}
	if _, err := execute(t, "hook", "unset"); err != nil {
		t.Fatalf("hook unset: %v", err)
	}
}

func TestHookSetAndUnset(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	var buf bytes.Buffer
	if err := runHookSet(ctx, &buf); err != nil {
		t.Fatalf("set: %v", err)
	}
	hp := filepath.Join(dir, ".git", "hooks", hookName)
	data, err := os.ReadFile(hp)
	if err != nil {
		t.Fatalf("hook not written: %v", err)
	}
	if !isOcoHook(data) {
		t.Error("installed hook missing marker")
	}
	info, _ := os.Stat(hp)
	if info.Mode().Perm()&0o100 == 0 {
		t.Error("hook should be executable")
	}

	buf.Reset()
	if err := runHookUnset(ctx, &buf); err != nil {
		t.Fatalf("unset: %v", err)
	}
	if _, err := os.Stat(hp); !os.IsNotExist(err) {
		t.Error("hook should be removed")
	}
}

func TestHookSetRefusesForeignHook(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	hp := filepath.Join(dir, ".git", "hooks", hookName)
	if err := os.WriteFile(hp, []byte("#!/bin/sh\necho custom\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runHookSet(context.Background(), &bytes.Buffer{}); err == nil {
		t.Fatal("expected refusal to overwrite foreign hook")
	}
}

func TestHookUnsetForeignHookUntouched(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	hp := filepath.Join(dir, ".git", "hooks", hookName)
	if err := os.WriteFile(hp, []byte("#!/bin/sh\necho custom\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := runHookUnset(context.Background(), &bytes.Buffer{}); err == nil {
		t.Fatal("expected error leaving foreign hook untouched")
	}
	if _, err := os.Stat(hp); err != nil {
		t.Error("foreign hook should remain")
	}
}

func TestHookUnsetMissing(t *testing.T) {
	dir := gitInit(t)
	wd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := runHookUnset(context.Background(), &buf); err != nil {
		t.Fatalf("unset on missing hook should be a no-op, got %v", err)
	}
	if !strings.Contains(buf.String(), "No prepare-commit-msg") {
		t.Errorf("unexpected output: %q", buf.String())
	}
}

func TestHookRuntimeSkipsWhenSourceSet(t *testing.T) {
	// With a message source, the hook must not touch the file.
	msgFile := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	if err := os.WriteFile(msgFile, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runHookRuntime(context.Background(), &bytes.Buffer{}, msgFile, "message"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(msgFile)
	if string(data) != "original\n" {
		t.Errorf("file should be untouched, got %q", data)
	}
}

func TestWriteHookMessage(t *testing.T) {
	msgFile := filepath.Join(t.TempDir(), "COMMIT_EDITMSG")
	template := "# Please enter the commit message\n# lines starting with # are ignored\n"
	if err := os.WriteFile(msgFile, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}

	// Without auto-uncomment, the template comments are preserved below.
	if err := writeHookMessage(msgFile, "feat: x", false); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(msgFile)
	if !strings.HasPrefix(string(data), "feat: x\n") {
		t.Errorf("message should be first: %q", data)
	}
	if !strings.Contains(string(data), "# Please enter") {
		t.Errorf("template comments should be kept: %q", data)
	}

	// With auto-uncomment, comment lines are dropped.
	if err := os.WriteFile(msgFile, []byte(template), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeHookMessage(msgFile, "feat: y", true); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(msgFile)
	if strings.Contains(string(data), "#") {
		t.Errorf("comments should be dropped with auto-uncomment: %q", data)
	}
}

func TestDropCommentLines(t *testing.T) {
	in := "# c1\nreal line\n#c2\n"
	out := dropCommentLines(in)
	if strings.Contains(out, "#") || !strings.Contains(out, "real line") {
		t.Errorf("dropCommentLines = %q", out)
	}
}
