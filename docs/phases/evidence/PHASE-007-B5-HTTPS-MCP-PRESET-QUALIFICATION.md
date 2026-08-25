# Phase 7 B5 — HTTPS MCP Preset Qualification

> Status: **RESEARCH COMPLETE — PRODUCT REGISTRY REMAINS EMPTY**
> Exact Hermes snapshot: `0.20.5` / `a0ca7c19204e514f9590ce3b812e029b315ab9e9`
> Date: 2026-08-25

## Result

No HTTPS MCP preset is qualified for the product registry yet. ADR-0014 permits a
closed, reviewed descriptor; it does not make a provider qualified. The current
zero-entry registry is therefore the correct product truth.

Hermes 0.20.5's built-in `_MCP_PRESETS` contains only a local stdio preset. Its packaged
HTTPS catalog entries use OAuth. OAuth remains capability-false because initiating-session
isolation, credential lifecycle and cancellation cleanup are not qualified.

## Exact Hermes surface

The exact snapshot stores MCP configuration under the active Profile's
`config.yaml` and resolves raw static bearer material from the same Profile's `.env`.
The relevant official source is:

- [`hermes_cli/mcp_config.py`](https://github.com/NousResearch/hermes-agent/blob/a0ca7c19204e514f9590ce3b812e029b315ab9e9/hermes_cli/mcp_config.py);
- [`tools/mcp_tool.py`](https://github.com/NousResearch/hermes-agent/blob/a0ca7c19204e514f9590ce3b812e029b315ab9e9/tools/mcp_tool.py);
- [`web_routers/mcp.py`](https://github.com/NousResearch/hermes-agent/blob/a0ca7c19204e514f9590ce3b812e029b315ab9e9/web_routers/mcp.py).

The CLI add flow is interactive and may prompt for overwrite, authentication, token,
tool selection and save-on-failure. It is not a qualified YORVA mutation surface. The
CLI and Dashboard test paths also expose native detail that does not satisfy YORVA's
bounded normalized `READY` result. A future adapter must own the fixed descriptor,
Profile-exact atomic write, bounded acquisition, normalization, timeout, cancellation
and cleanup postcondition.

The Hermes HTTP client follows redirects. A first YORVA preset must independently prove
that its canonical endpoint does not redirect, or enforce an accepted fixed-origin
policy before carrying credentials.

## Conditional candidate 1 — GitHub Remote MCP repos read-only

Recommendation: **first qualification candidate; not product-qualified**.

```text
fixed endpoint: https://api.githubcopilot.com/mcp/x/repos/readonly
credential class: STATIC_BEARER
credential authority: exact Hermes Profile .env
transport: Streamable HTTP over HTTPS
```

GitHub documents its hosted remote MCP endpoint and PAT Bearer authentication in the
[GitHub Copilot MCP setup documentation](https://docs.github.com/en/copilot/how-tos/provide-context/use-mcp-in-your-ide/set-up-the-github-mcp-server),
[official server authentication guide](https://github.com/github/github-mcp-server/blob/main/docs/host-integration.md),
and [remote server endpoint documentation](https://github.com/github/github-mcp-server/blob/main/docs/remote-server.md).

Before a descriptor may be added, qualification must freeze a small exact tool allowlist
from the then-current official source and an authenticated live list. It must also prove
PAT scope behavior, no redirect, Hermes 0.20.5 handshake compatibility, response bounds,
timeout/cancellation cleanup, Profile drift handling and `CONFIGURED` versus `READY`.
No tool ID is inferred or frozen by this document.

## Conditional candidate 2 — Postman Remote MCP Minimal EU

Recommendation: **weaker candidate; not product-qualified**.

```text
fixed endpoint: https://mcp.eu.postman.com/minimal
credential class: STATIC_BEARER
credential authority: exact Hermes Profile .env
transport: Streamable HTTP over HTTPS
```

Postman documents the EU endpoint and API-key Bearer authentication in its
[remote server setup](https://learning.postman.com/latest-v-12/docs/reference/postman-api/postman-mcp-server/postman-mcp-remote-server),
[server overview](https://learning.postman.com/latest-v-12/docs/reference/postman-api/postman-mcp-server/overview),
and [API-key authentication documentation](https://learning.postman.com/latest-v-12/docs/reference/postman-api/authentication).

This candidate is weaker because the API key may reach mutation-capable Postman resources
and the static documentation does not provide a sufficiently small exact read-only tool
allowlist to freeze. Region suitability, redirect behavior, live tool identities, bounds
and cleanup also remain unverified. Without Owner acceptance of that authority and a
closed allowlist, this candidate is NO-GO.

## Explicit OAuth NO-GO

Notion's hosted endpoint is an example of the packaged OAuth group. Its official
[MCP documentation](https://developers.notion.com/guides/mcp/get-started-with-mcp)
requires interactive OAuth and does not provide a qualified non-interactive static
credential path for this boundary. Airtable, Asana, Atlassian, Comfy, Datadog, Figma,
Hugging Face, Intercom, Linear, Netlify, PayPal, Sentry, Square, Stripe, Supabase,
Vercel and Webflow remain closed for the same packaged OAuth/session-isolation reason.

## Owner decision and next Gate

The smallest next step is to select GitHub repos read-only as the sole first candidate.
Selection authorizes qualification work only; it does not authorize adding the registry
entry. The entry remains blocked until official endpoint/auth/tool evidence, disposable
Profile write/read/delete tests, authenticated live initialization and tool listing,
strict response bounds, no-redirect behavior, cancellation cleanup, external-drift
handling and Windows smoke all pass.

No credential, account identifier, native path, URL supplied by a user, or live MCP
response is recorded in this evidence.
