# Phase 5 Batch 1 — Pinned Hermes Qualification Evidence

- Date: 2026-08-19
- Hermes version: `0.20.2`
- Hermes commit: `df4b65147d7ddd74dd449f9067aabbca5aef0ec7`
- Reviewed archive SHA-256: `2ED02F76AAF5DAB0BFD320BDBFA10AAD0F67E00CBBF87906CDE05462681708BA`
- Scope: model provider/config/credential/validation surfaces only

## Historical qualification stop and approved resolution

The first Batch 1 pass correctly stopped because `hermes auth add --api-key`
places a secret in argv, while the safe Web/TUI setters need an already-running
Hermes service. The Owner subsequently approved ADR-0007's pinned,
Profile-scoped, provider-allowlisted canonical `.env` writer. The production
writer accepts only `nativeId`, `providerPresetId` and the secret value; paths,
environment names and config keys are adapter-private constants.

No Hermes Python module is imported. YORVA does not edit `config.yaml` or expose
a generic file API.

## Qualified catalog

| Product preset | Region | Hermes provider | Canonical credential key | Built-in endpoint behavior | Reviewed models |
|---|---|---|---|---|---|
| DeepSeek | China | `deepseek` | `DEEPSEEK_API_KEY` | `https://api.deepseek.com/v1` | `deepseek-v4-pro`, `deepseek-v4-flash` |
| Qwen / Alibaba | China-first | `alibaba` | `DASHSCOPE_API_KEY` | DashScope international compatible endpoint | `qwen3.7-max`, `qwen3.7-plus`, `qwen3.6-plus` |
| Kimi / Moonshot | China | `kimi-coding-cn` | `KIMI_CN_API_KEY` | `https://api.moonshot.cn/v1` | `kimi-k3`, `kimi-k2.7-code`, `kimi-k2.6` |
| MiniMax | China | `minimax-cn` | `MINIMAX_CN_API_KEY` | `https://api.minimaxi.com/anthropic` | `MiniMax-M3`, `MiniMax-M2.7`, `MiniMax-M2.5` |
| GLM / Zhipu | China | `zai` | `GLM_API_KEY` | Hermes-owned Z.AI endpoint selection | `glm-5.2`, `glm-5.1`, `glm-5` |
| OpenRouter | Global | `openrouter` | `OPENROUTER_API_KEY` | Hermes special provider path | `anthropic/claude-sonnet-4.6`, `openai/gpt-5.4` |
| OpenAI | Global | `openai-api` | `OPENAI_API_KEY` | `https://api.openai.com/v1` | `gpt-5.6-sol`, `gpt-5.6-terra`, `gpt-5.4` |
| Anthropic | Global | `anthropic` | `ANTHROPIC_API_KEY` | `https://api.anthropic.com` | `claude-fable-5`, `claude-sonnet-5`, `claude-opus-4-8` |

The product exposes neither the Hermes provider id nor the credential key.
Recommended models are reviewed suggestions, not a complete dynamic catalog.

## Profile, config and credential surfaces

- Profile selection: official `--profile <nativeId>` and the existing clean
  `HERMES_HOME` process environment. The default Profile maps to the Hermes root;
  a named Profile maps to `profiles/<nativeId>` under that root.
- Non-secret configuration: official `hermes --profile <nativeId> config get
  model.provider --json`, `model.default --json`, and corresponding `config set`
  scalar calls.
- Canonical credential storage: exact selected Profile `.env`; only the approved
  writer opens it. Status is a boolean presence/usable-value result.
- Writer bounds: 256 KiB file, 16 KiB printable secret, exact allowlisted key,
  duplicate-key rejection, reparse/symlink rejection, unknown-line preservation,
  same-directory temp, sync, atomic replace, read-back and cleanup.
- Concurrency: the application reuses Instance/Profile coordination; the writer
  also compares the observed file immediately before replace and fails closed on
  change.
- Secret transport: HTTP request to in-memory use case to writer. It is absent
  from argv, child environment, output, SQLite, events and ordinary responses.

## Tools-disabled validation surface

The only pinned built-in empty toolset is `context_engine`. Hermes adds dynamic
context-engine tools only when `context.engine` selects a plugin. Before a
validation call, YORVA reads `context.engine` using the official JSON config
surface and proceeds only when it is exactly the built-in `compressor`. The
qualified invocation is:

```text
hermes --profile <nativeId> --provider <qualified-id> --model <model-id> \
  --toolsets context_engine --oneshot <fixed harmless prompt>
```

The fixed prompt contains no user input. Provider/model/Profile values pass
closed validation before argv construction. Model stdout and raw stderr are
discarded, never logged or returned. The existing process-tree owner, clean
environment and bounded separate stream capture are reused. One 45-second
whole-operation deadline controls query plus validation; timeout, cancellation
and output limit normalize to `UNKNOWN` without changing config or credentials.

## Locked failure behavior

Unsupported version/provider, unsafe Profile path, ambiguous/oversized `.env`,
duplicate target assignments, external modification, unknown config output,
non-default context engine, process timeout/cancellation/output overflow and
unknown native result all fail closed. No fallback creates a custom provider,
custom URL, shell command, Python import or global Windows environment change.
