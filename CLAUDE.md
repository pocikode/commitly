# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`cly` (Commitly) — a single static Go binary that generates Conventional Commits messages from a staged git diff using any OpenAI- or Anthropic-compatible API. Go port of [opencommit](https://github.com/di-sukharev/opencommit); prompt design and command surface follow the original.

Module path: `github.com/pocikode/commitly`. Binary name: `cly`.

## Commands

```sh
make build        # CGO_ENABLED=0 static binary -> ./bin/cly (with version ldflags)
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

Entry: `main.go` -> `cmd.Execute()` -> `newRootCmd()` (cobra). Bare `cly` aliases `cly commit`. All code lives under `internal/`.

Request flow for a commit (`internal/cmd/commit.go`):

1. `config.Resolve` builds the effective config (see Config layering).
2. `provider.New(cfg)` returns a `Provider` adapter.
3. `git.New(".")` + `IsRepo` guard.
4. `runCommitFlow` orchestrates: ensure staged diff -> `prepareGeneration` (load ignore rules, filtered diff, token-budget fit, build system prompt) -> `generate` (provider call behind spinner) -> interactive review loop (confirm / edit / regenerate / cancel) -> commit -> optional push.

Key seams to understand before editing:

- **Provider abstraction** (`internal/provider`): `Provider` interface has one method, `GenerateCommitMessage(ctx, CommitRequest) (string, error)`. Three adapters — `OpenAICompatible`, `AnthropicCompatible`, `Custom` — all built from a shared `Settings` (BaseURL, APIKey, Headers, Proxy). `provider.New(cfg)` is the factory mapping config to adapter; `presets.go` holds the built-in named providers (openai, anthropic, gemini, ollama, groq) with default URLs, request shape, and static model lists. Adapters classify failures into `APIError` with an `ErrorKind` (auth/rate_limit/network/api) and **never log or echo the API key**.

- **Config layering** (`internal/config`): `Resolve` merges sources lowest-to-highest: built-in default < global file (`~/.commitly.yaml`) < project file (`./.commitly.yaml`) < active profile overlay < env vars (`CLY_`-prefixed) < command-line flags. `keys.go` `registry` is the **single source of truth** for every config key — it drives both env-override resolution and `cly config get/set`. Add a new config key by adding one `KeyDef` (YAML name, env var, typed Get/Set with validation), not by editing multiple files. `api_key` is redacted on get.

- **Profiles** (`internal/config`): a `Profile` is a named bundle of provider fields (ai_provider, provider_type, api_key, api_url, model) stored in `Config.Profiles`; `ActiveProfile` selects one. `Resolve` calls `applyActiveProfile` (file `active_profile` < `CLY_ACTIVE_PROFILE` env; unknown name falls back to the file value) and overlays the chosen profile's non-empty fields onto the top-level provider fields **before** the per-key env/flag layers, so e.g. `CLY_MODEL` still wins. `active_profile` is a registry key (strict-validated in `config set`, but skipped in `applyEnv` since the resolver handles it leniently). Writers mirror the active profile onto the top-level fields so `config get`/raw file reads stay accurate.

- **Prompt building** (`internal/prompt`): `prompt.System(cfg, Options)` assembles the system prompt from config flags (emoji, omit_scope, one_line, description). `prompt_module=@commitlint` injects a Conventional-Commits rule set; a filesystem path loads a custom template via `LoadOverride`. `prompt.Clean` post-processes model output (respecting one-line mode).

- **Diff + ignore + tokens**: `git.DiffFiltered` returns the staged diff minus paths matched by `.commitlyignore` rules (`git.LoadIgnore`). `tokens.FitDiff(diff, max)` budget-trims the diff to `tokens_max_input`, returning a `truncated` flag surfaced to the user.

- **UI** (`internal/ui`): `ui.Select(yes, out)` returns a `UI` implementation; the `--yes/-y` flag selects a non-interactive auto-confirm path. The interactive path uses charmbracelet huh/spinner/lipgloss. The commit flow depends on the `UI` interface (not a concrete type) so it can be unit-tested with fakes — see `commitDeps`.

- **Config manager** (`internal/cmd/config.go`): bare `cly config` runs `runConfigManager`, an interactive profile manager (list / add / edit / delete / use). On a TTY it renders a bubbletea list (`profileTUIModel`, lipgloss-styled) with key bindings a/e/d/enter/q; for piped/test input it falls back to `profileMenuText` + a line-based wizard. Both share one `*bufio.Reader` so the flow stays scriptable. The TTY/non-TTY split is keyed on `inputIsTerminal`. `profileTUIModel.Update/View` are pure and unit-tested directly via `tea.KeyMsg` (no real terminal). lipgloss emits no escape codes on a non-TTY writer, so styled output degrades to plain ASCII in tests — render each row as a single `Style.Render` call to keep visible text contiguous for substring assertions.

- **Git hook** (`internal/cmd/hook.go`): `cly hook set/unset` installs a `prepare-commit-msg` hook that calls `prepareGeneration` non-interactively and degrades gracefully (warns, leaves commit unblocked, on failure). `unset` only removes the cly-owned hook.

## Conventions

- Commands are built as functions (`newRootCmd`, `newCommitCmd`, …), not package-level vars, so each test gets fresh, isolated cobra flag state.
- Command flows take their collaborators via a deps struct (e.g. `commitDeps`) holding interfaces, enabling fakes in tests. Mirror this when adding a new command with external dependencies.
- `ErrCancelled` is a sentinel: user-initiated abort maps to a non-zero exit without an alarming `Error:` prefix. Use sentinels for expected, non-error exits.
- Coverage gate is 80% (`COVER_MIN` in Makefile); new packages should ship with tests.
