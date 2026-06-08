# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`oco` (OpenCommit-Go) — a single static Go binary that generates Conventional Commits messages from a staged git diff using any OpenAI- or Anthropic-compatible API. Go port of [opencommit](https://github.com/di-sukharev/opencommit); prompt design and command surface follow the original.

Module path: `github.com/pocikode/opencommit`. Binary name: `oco`.

## Commands

```sh
make build        # CGO_ENABLED=0 static binary -> ./bin/oco (with version ldflags)
make install      # go install with version stamping
make test         # go test ./...
make cover-check  # tests + enforce >= 80% total coverage (CI gate)
make vet          # go vet ./...
make smoke        # build then run --version and --help
make tidy         # go mod tidy
```

Run a single test:

```sh
go test ./internal/cmd -run TestRunCommitFlow -v
```

Go 1.26.4. CI (`.github/workflows/ci.yml`) runs vet + cover-check; releases are built with goreleaser (`.goreleaser.yaml`).

## Architecture

Entry: `main.go` -> `cmd.Execute()` -> `newRootCmd()` (cobra). Bare `oco` aliases `oco commit`. All code lives under `internal/`.

Request flow for a commit (`internal/cmd/commit.go`):

1. `config.Resolve` builds the effective config (see Config layering).
2. `provider.New(cfg)` returns a `Provider` adapter.
3. `git.New(".")` + `IsRepo` guard.
4. `runCommitFlow` orchestrates: ensure staged diff -> `prepareGeneration` (load ignore rules, filtered diff, token-budget fit, build system prompt) -> `generate` (provider call behind spinner) -> interactive review loop (confirm / edit / regenerate / cancel) -> commit -> optional push.

Key seams to understand before editing:

- **Provider abstraction** (`internal/provider`): `Provider` interface has one method, `GenerateCommitMessage(ctx, CommitRequest) (string, error)`. Three adapters — `OpenAICompatible`, `AnthropicCompatible`, `Custom` — all built from a shared `Settings` (BaseURL, APIKey, Headers, Proxy). `provider.New(cfg)` is the factory mapping config to adapter; `presets.go` holds the built-in named providers (openai, anthropic, gemini, ollama, groq) with default URLs, request shape, and static model lists. Adapters classify failures into `APIError` with an `ErrorKind` (auth/rate_limit/network/api) and **never log or echo the API key**.

- **Config layering** (`internal/config`): `Resolve` merges sources lowest-to-highest: built-in default < global file (`~/.opencommit.yaml`) < project file (`./.opencommit.yaml`) < env vars (`OCO_`-prefixed) < command-line flags. `keys.go` `registry` is the **single source of truth** for every config key — it drives both env-override resolution and `oco config get/set`. Add a new config key by adding one `KeyDef` (YAML name, env var, typed Get/Set with validation), not by editing multiple files. `api_key` is redacted on get.

- **Prompt building** (`internal/prompt`): `prompt.System(cfg, Options)` assembles the system prompt from config flags (emoji, omit_scope, one_line, description). `prompt_module=@commitlint` injects a Conventional-Commits rule set; a filesystem path loads a custom template via `LoadOverride`. `prompt.Clean` post-processes model output (respecting one-line mode).

- **Diff + ignore + tokens**: `git.DiffFiltered` returns the staged diff minus paths matched by `.opencommitignore` rules (`git.LoadIgnore`). `tokens.FitDiff(diff, max)` budget-trims the diff to `tokens_max_input`, returning a `truncated` flag surfaced to the user.

- **UI** (`internal/ui`): `ui.Select(yes, out)` returns a `UI` implementation; the `--yes/-y` flag selects a non-interactive auto-confirm path. The interactive path uses charmbracelet huh/spinner/lipgloss. The commit flow depends on the `UI` interface (not a concrete type) so it can be unit-tested with fakes — see `commitDeps`.

- **Git hook** (`internal/cmd/hook.go`): `oco hook set/unset` installs a `prepare-commit-msg` hook that calls `prepareGeneration` non-interactively and degrades gracefully (warns, leaves commit unblocked, on failure). `unset` only removes the oco-owned hook.

## Conventions

- Commands are built as functions (`newRootCmd`, `newCommitCmd`, …), not package-level vars, so each test gets fresh, isolated cobra flag state.
- Command flows take their collaborators via a deps struct (e.g. `commitDeps`) holding interfaces, enabling fakes in tests. Mirror this when adding a new command with external dependencies.
- `ErrCancelled` is a sentinel: user-initiated abort maps to a non-zero exit without an alarming `Error:` prefix. Use sentinels for expected, non-error exits.
- Coverage gate is 80% (`COVER_MIN` in Makefile); new packages should ship with tests.
