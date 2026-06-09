# Commitly (`cly`)

AI-powered git commit message generator — a single static Go binary with
**provider freedom**. Point it at any OpenAI-compatible or Anthropic-compatible
endpoint (cloud or self-hosted), or pick a built-in preset. No Node, no Python,
no vendor lock-in.

Port of [opencommit](https://github.com/di-sukharev/opencommit). See [Credits](#credits).

## Features

- Generates Conventional Commits messages from your **staged** diff.
- Works with **any** OpenAI- or Anthropic-compatible API, plus presets for
  OpenAI, Anthropic, Gemini, Ollama, and Groq.
- Interactive review: spinner, colored preview, and a
  confirm / edit / regenerate / cancel menu.
- Optional emoji/gitmoji, one-line mode, body description, and custom prompt
  templates.
- `prepare-commit-msg` git hook, model picker, and commitlint rule injection.
- Single YAML config at `~/.commitly.yaml` with `CLY_`-prefixed env overrides.

## Install

```sh
# From source (requires Go)
go install github.com/pocikode/commitly@latest

# Or build locally
make build      # produces ./bin/cly
```

Prebuilt binaries for Linux/macOS/Windows are attached to each
[GitHub Release](https://github.com/pocikode/commitly/releases).

## Quick start

```sh
# 1. Configure a provider (interactive profile manager)
cly config

# ...or set values directly
cly config set ai_provider=openai api_key=sk-... model=gpt-4o

# 2. Stage changes and generate a commit
git add .
cly
```

Bare `cly` is an alias for `cly commit`.

## Commands

| Command | Description |
|---|---|
| `cly` / `cly commit` | Generate a commit message from staged changes and commit. |
| `cly config` | Interactive profile manager (no args): list, add, edit, delete, switch profiles. |
| `cly config get <KEY>...` | Print config values (`api_key` is redacted). |
| `cly config set <KEY>=<VALUE>...` | Set and persist config values. |
| `cly config profiles` (alias `list`) | List saved provider/model profiles; `*` marks active. |
| `cly config use <NAME>` | Switch the active profile. |
| `cly models [--set <model>]` | List models for the active provider; select and persist. |
| `cly hook set` / `cly hook unset` | Install/remove the `prepare-commit-msg` hook. |
| `cly commitlint` | Show commitlint rules and enable rule injection. |
| `cly --version` / `cly --help` | Version and help. |

Global flags: `--yes/-y` (auto-confirm, non-interactive) and
`--config <path>` (override config file).

## Configuration

Config lives at `~/.commitly.yaml` (override with `CLY_CONFIG_PATH` or
`--config`). A project-local `.commitly.yaml` in the repo root overrides the
global file. Precedence, highest to lowest:

> command-line flag → environment variable → project config → global config → built-in default

Each YAML key maps to a `CLY_`-prefixed env var.

| YAML key | Env var | Default | Notes |
|---|---|---|---|
| `ai_provider` | `CLY_AI_PROVIDER` | `openai` | Preset name or `custom`. |
| `provider_type` | `CLY_PROVIDER_TYPE` | `openai_compatible` | `openai_compatible` or `anthropic_compatible`. |
| `api_key` | `CLY_API_KEY` | — | Redacted in output; never logged. |
| `api_url` | `CLY_API_URL` | `https://api.openai.com/v1` | Falls back to the preset URL if empty. |
| `model` | `CLY_MODEL` | `gpt-4o-mini` | |
| `active_profile` | `CLY_ACTIVE_PROFILE` | — | Name of the profile whose provider fields are applied. |
| `profiles` | — | — | Map of named provider/model bundles (managed by `cly config`). |
| `api_custom_headers` | `CLY_API_CUSTOM_HEADERS` | `{}` | JSON object of extra headers. |
| `proxy` | `CLY_PROXY` | — | HTTP(S) proxy URL. |
| `tokens_max_input` | `CLY_TOKENS_MAX_INPUT` | `40960` | Diff is budget-fitted to this. |
| `tokens_max_output` | `CLY_TOKENS_MAX_OUTPUT` | `4096` | |
| `description` | `CLY_DESCRIPTION` | `false` | Add a body explaining WHY. |
| `emoji` | `CLY_EMOJI` | `false` | Gitmoji prefix. |
| `omit_scope` | `CLY_OMIT_SCOPE` | `false` | Use `<type>: <subject>`. |
| `one_line_commit` | `CLY_ONE_LINE_COMMIT` | `false` | Collapse to one line. |
| `gitpush` | `CLY_GITPUSH` | `false` | Offer to push after commit. |
| `message_template_placeholder` | `CLY_MESSAGE_TEMPLATE_PLACEHOLDER` | `$msg` | Placeholder for hook templating. |
| `prompt_module` | `CLY_PROMPT_MODULE` | `conventional-commit` | `@commitlint`, or a path to a prompt file. |
| `hook_auto_uncomment` | `CLY_HOOK_AUTO_UNCOMMENT` | `false` | Drop template comment lines in the hook. |

### Example `~/.commitly.yaml`

```yaml
ai_provider: openai
provider_type: openai_compatible
api_url: https://api.openai.com/v1
api_key: sk-...
model: gpt-4o-mini
tokens_max_input: 40960
tokens_max_output: 4096
emoji: false
gitpush: false

# Multiple provider/model setups; active_profile selects one.
active_profile: openai-work
profiles:
  openai-work:
    ai_provider: openai
    provider_type: openai_compatible
    api_url: https://api.openai.com/v1
    api_key: sk-...
    model: gpt-4o
  ollama-local:
    ai_provider: ollama
    provider_type: openai_compatible
    api_url: http://localhost:11434/v1
    model: qwen2.5-coder
```

## Profiles

A **profile** is a named bundle of provider/model credentials (`ai_provider`,
`provider_type`, `api_url`, `api_key`, `model`). Define several and switch
between them — e.g. a cloud model for work and a local Ollama model offline.

Run `cly config` (no args) for the interactive manager:

```
✨ Commitly Profiles
╭───────────────────────────────────────────────────╮
│      claude (anthropic/claude-3-5-sonnet-latest)  │
│  > * ollama-local (ollama/llama3.1)               │
│      openai-work (openai/gpt-4o)                   │
╰───────────────────────────────────────────────────╯
  [a]dd  [e]dit  [d]elete  [enter]use  [q]uit
```

- `↑`/`↓` (or `j`/`k`) move, `*` marks the active profile
- **a** add a new profile, **e** edit selected, **d** delete selected
- **enter** activate the selected profile, **q** quit

The active profile's fields overlay the top-level provider settings when a
commit runs. Switch non-interactively too:

```sh
cly config profiles          # list, * marks active
cly config use ollama-local  # set active profile
CLY_ACTIVE_PROFILE=claude cly commit   # per-run override
```

Profile selection precedence: `CLY_ACTIVE_PROFILE` env → `active_profile` in
the config file. An unknown name falls back to the file's active profile.
Per-key env/flags (e.g. `CLY_MODEL`) still win over the profile.

## Providers

### Presets

```sh
cly config set ai_provider=openai    model=gpt-4o
cly config set ai_provider=anthropic provider_type=anthropic_compatible model=claude-3-5-sonnet-latest
cly config set ai_provider=groq      model=llama-3.3-70b-versatile
```

### Self-hosted / custom endpoint (e.g. Ollama, vLLM, LM Studio)

Any OpenAI-compatible server works — just set `api_url`:

```sh
cly config set ai_provider=ollama api_url=http://localhost:11434/v1 model=qwen2.5-coder
```

Custom headers and proxy:

```sh
cly config set api_custom_headers='{"X-Org":"acme"}' proxy=http://localhost:8080
```

## Git hook

```sh
cly hook set     # install prepare-commit-msg
git commit       # message is generated automatically
cly hook unset   # remove it (only removes the cly-owned hook)
```

The hook generates non-interactively and degrades gracefully: if generation
fails it warns and leaves your commit unblocked.

## Commitlint

```sh
cly config set prompt_module=@commitlint
cly commitlint   # view the injected rules
```

When enabled, a simplified Conventional-Commits rule set is appended to the
prompt so generated messages pass typical commit linting.

## Development

```sh
make test         # run tests
make cover-check  # enforce >= 80% coverage
make build        # build ./bin/cly
make smoke        # build + run --version/--help
```

## Credits

Port of [opencommit](https://github.com/di-sukharev/opencommit) by
[@di-sukharev](https://github.com/di-sukharev) and contributors. Prompt design
and command surface follow the original.

## License

MIT — see [LICENSE](LICENSE). © 2026 Agus Supriyatna.
