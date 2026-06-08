# OpenCommit-Go (`oco`)

AI-powered git commit message generator — a single static Go binary with
**provider freedom**. Point it at any OpenAI-compatible or Anthropic-compatible
endpoint (cloud or self-hosted), or pick a built-in preset. No Node, no Python,
no vendor lock-in.

This is a Go port of [opencommit](https://github.com/di-sukharev/opencommit).
See [Credits](#credits).

## Features

- Generates Conventional Commits messages from your **staged** diff.
- Works with **any** OpenAI- or Anthropic-compatible API, plus presets for
  OpenAI, Anthropic, Gemini, Ollama, and Groq.
- Interactive review: spinner, colored preview, and a
  confirm / edit / regenerate / cancel menu.
- Optional emoji/gitmoji, one-line mode, body description, and custom prompt
  templates.
- `prepare-commit-msg` git hook, model picker, and commitlint rule injection.
- Single YAML config at `~/.opencommit.yaml` with `OCO_`-prefixed env overrides.

## Install

```sh
# From source (requires Go)
go install github.com/pocikode/opencommit@latest

# Or build locally
make build      # produces ./bin/oco
```

Prebuilt binaries for Linux/macOS/Windows are attached to each
[GitHub Release](https://github.com/pocikode/opencommit-go/releases).

## Quick start

```sh
# 1. Configure a provider (interactive wizard)
oco config

# ...or set values directly
oco config set ai_provider=openai api_key=sk-... model=gpt-4o

# 2. Stage changes and generate a commit
git add .
oco
```

Bare `oco` is an alias for `oco commit`.

## Commands

| Command | Description |
|---|---|
| `oco` / `oco commit` | Generate a commit message from staged changes and commit. |
| `oco config` | Interactive setup wizard (no args). |
| `oco config get <KEY>...` | Print config values (`api_key` is redacted). |
| `oco config set <KEY>=<VALUE>...` | Set and persist config values. |
| `oco models [--set <model>]` | List models for the active provider; select and persist. |
| `oco hook set` / `oco hook unset` | Install/remove the `prepare-commit-msg` hook. |
| `oco commitlint` | Show commitlint rules and enable rule injection. |
| `oco --version` / `oco --help` | Version and help. |

Global flags: `--yes/-y` (auto-confirm, non-interactive) and
`--config <path>` (override config file).

## Configuration

Config lives at `~/.opencommit.yaml` (override with `OCO_CONFIG_PATH` or
`--config`). A project-local `.opencommit.yaml` in the repo root overrides the
global file. Precedence, highest to lowest:

> command-line flag → environment variable → project config → global config → built-in default

Each YAML key maps to an `OCO_`-prefixed env var.

| YAML key | Env var | Default | Notes |
|---|---|---|---|
| `ai_provider` | `OCO_AI_PROVIDER` | `openai` | Preset name or `custom`. |
| `provider_type` | `OCO_PROVIDER_TYPE` | `openai_compatible` | `openai_compatible` or `anthropic_compatible`. |
| `api_key` | `OCO_API_KEY` | — | Redacted in output; never logged. |
| `api_url` | `OCO_API_URL` | `https://api.openai.com/v1` | Falls back to the preset URL if empty. |
| `model` | `OCO_MODEL` | `gpt-4o` | |
| `api_custom_headers` | `OCO_API_CUSTOM_HEADERS` | `{}` | JSON object of extra headers. |
| `proxy` | `OCO_PROXY` | — | HTTP(S) proxy URL. |
| `tokens_max_input` | `OCO_TOKENS_MAX_INPUT` | `40960` | Diff is budget-fitted to this. |
| `tokens_max_output` | `OCO_TOKENS_MAX_OUTPUT` | `4096` | |
| `description` | `OCO_DESCRIPTION` | `false` | Add a body explaining WHY. |
| `emoji` | `OCO_EMOJI` | `false` | Gitmoji prefix. |
| `omit_scope` | `OCO_OMIT_SCOPE` | `false` | Use `<type>: <subject>`. |
| `one_line_commit` | `OCO_ONE_LINE_COMMIT` | `false` | Collapse to one line. |
| `gitpush` | `OCO_GITPUSH` | `false` | Offer to push after commit. |
| `message_template_placeholder` | `OCO_MESSAGE_TEMPLATE_PLACEHOLDER` | `$msg` | Placeholder for hook templating. |
| `prompt_module` | `OCO_PROMPT_MODULE` | `conventional-commit` | `@commitlint`, or a path to a prompt file. |
| `hook_auto_uncomment` | `OCO_HOOK_AUTO_UNCOMMENT` | `false` | Drop template comment lines in the hook. |

### Example `~/.opencommit.yaml`

```yaml
ai_provider: openai
provider_type: openai_compatible
api_url: https://api.openai.com/v1
api_key: sk-...
model: gpt-4o
tokens_max_input: 40960
tokens_max_output: 4096
emoji: false
gitpush: false
```

## Providers

### Presets

```sh
oco config set ai_provider=openai    model=gpt-4o
oco config set ai_provider=anthropic provider_type=anthropic_compatible model=claude-3-5-sonnet-latest
oco config set ai_provider=groq      model=llama-3.3-70b-versatile
```

### Self-hosted / custom endpoint (e.g. Ollama, vLLM, LM Studio)

Any OpenAI-compatible server works — just set `api_url`:

```sh
oco config set ai_provider=ollama api_url=http://localhost:11434/v1 model=qwen2.5-coder
```

Custom headers and proxy:

```sh
oco config set api_custom_headers='{"X-Org":"acme"}' proxy=http://localhost:8080
```

## Git hook

```sh
oco hook set     # install prepare-commit-msg
git commit       # message is generated automatically
oco hook unset   # remove it (only removes the oco-owned hook)
```

The hook generates non-interactively and degrades gracefully: if generation
fails it warns and leaves your commit unblocked.

## Commitlint

```sh
oco config set prompt_module=@commitlint
oco commitlint   # view the injected rules
```

When enabled, a simplified Conventional-Commits rule set is appended to the
prompt so generated messages pass typical commit linting.

## Development

```sh
make test         # run tests
make cover-check  # enforce >= 80% coverage
make build        # build ./bin/oco
make smoke        # build + run --version/--help
```

## Credits

Port of [opencommit](https://github.com/di-sukharev/opencommit) by
[@di-sukharev](https://github.com/di-sukharev) and contributors. Prompt design
and command surface follow the original.

## License

MIT — see [LICENSE](LICENSE). © 2026 Agus Supriyatna.
