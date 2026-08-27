# Phase 7 B5 — HTTPS MCP Preset Qualification

> Status: **CANDIDATE HARNESS PASS — LIVE / PRODUCT QUALIFICATION REMAINS NO-GO**
> Exact Hermes snapshot: `0.20.5` / `a0ca7c19204e514f9590ce3b812e029b315ab9e9`
> Date: 2026-08-25

## Result

No HTTPS MCP preset is qualified for the product registry yet. P7-D3 permits a closed,
reviewed descriptor; it does not make a provider qualified. The current zero-entry
registry is therefore the correct product truth. The Owner selected GitHub
repos read-only as the sole first candidate on 2026-08-25; that decision authorized
qualification only.

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

### Frozen source qualification

The candidate harness freezes GitHub's official `github-mcp-server` source at commit
[`8898db96b8043db1ddacff444600e408cf824fa2`](https://github.com/github/github-mcp-server/tree/8898db96b8043db1ddacff444600e408cf824fa2).
At that revision, the official
[`remote-server.md`](https://github.com/github/github-mcp-server/blob/8898db96b8043db1ddacff444600e408cf824fa2/docs/remote-server.md)
defines `/x/{toolset}/readonly`, and the official
[`README.md`](https://github.com/github/github-mcp-server/blob/8898db96b8043db1ddacff444600e408cf824fa2/README.md)
defines PAT Bearer authentication. The `repos` tool registrations in
[`repositories.go`](https://github.com/github/github-mcp-server/blob/8898db96b8043db1ddacff444600e408cf824fa2/pkg/github/repositories.go)
and [`search.go`](https://github.com/github/github-mcp-server/blob/8898db96b8043db1ddacff444600e408cf824fa2/pkg/github/search.go)
uniquely yield this exact `ReadOnlyHint=true` inventory:

```text
get_commit
get_file_blame
get_file_contents
get_latest_release
get_release_by_tag
get_tag
list_branches
list_commits
list_releases
list_repository_collaborators
list_tags
search_code
search_commits
search_repositories
```

`github_qualification.go` is a qualification-only Streamable HTTP probe with:

- one compiled HTTPS origin and path;
- request-lifetime static Bearer input and redacted result/error types;
- direct TLS 1.2 minimum with environment proxies disabled, and redirects rejected
  before a second request;
- a 20-second total deadline, bounded response bodies and bounded tool count;
- `initialize`, `notifications/initialized`, then `tools/list` with session propagation;
- exact tool-set equality and `ReadOnlyHint=true` for every returned tool;
- response-body closure and idle-connection cleanup on every exit.

It has no caller-controlled URL, path, header, command, environment map or tool list.
It does not read or write Profile configuration and is not referenced by the reviewed
registry, Bundle, local HTTP API or Desktop.

### Automated fixture evidence

Executed on 2026-08-25 without a real credential or network account:

```text
go test ./internal/runtime/hermes/mcpmanagement
ok github.com/YoLin02/yorva/services/node/internal/runtime/hermes/mcpmanagement

go vet ./internal/runtime/hermes/mcpmanagement
PASS
```

TLS `httptest` fixtures prove the exact three-message handshake, fixed session
propagation, exact 14-tool acceptance, extra-tool rejection, response bound, redirect
rejection without credential forwarding, cancellation and pre-network credential
validation. They do not prove GitHub's live service or Hermes compatibility.

The one manual live entry point is intentionally skipped unless a request-lifetime
`YORVA_GITHUB_MCP_QUALIFICATION_TOKEN` environment value is supplied. Its output is
limited to stable status, tool count and pinned source revision. No live token was
available for this evidence, so it was not run and no live response is recorded.

### Production local Profile lifecycle Preset

Phase 7 now contains one compile-time `yorva-mcp-test` descriptor in the product
registry. Normal `yorvad` owns its ephemeral IPv4 loopback protocol server and private
fixed-identity redirect. The production Profile adapter lifecycle covers atomic
`config.yaml` write, fixed handshake, tool-scope verification, authoritative READY
readback, binding to a second Profile, deletion and authoritative absence readback. A
tagged cross-layer gate exercises authenticated local HTTP routes, the real SQLite
binding repository and application Operations. The emitted Profile configuration is
asserted to contain no authorization, `command`, `args` or `env` fields. See
`PHASE-007-B5-MCP-TEST-SERVER.md` for the protocol and gate command. This qualifies only
the YORVA-owned local test endpoint, not a third-party provider.

### Remaining gate — NO-GO

Before a third-party registry descriptor may be added, a later authorized run must still prove:

- authenticated live initialization and exact tool listing against the fixed endpoint;
- the accepted PAT scope and account policy;
- exact Hermes 0.20.5 handshake compatibility, not only the independent probe;
- disposable Profile-exact atomic write/read/delete and external-drift behavior;
- `CONFIGURED` versus `READY` lifecycle behavior and Windows smoke.

All five are `UNKNOWN`, not inferred from fixtures. Registry mutation and product
authorization therefore remain **NO-GO**.

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

The source/fixture portion is now complete. Selection still does not authorize adding
the registry entry. The product remains capability-false until every remaining live,
Hermes/Profile and Windows item above is evidenced and a separate gate explicitly
authorizes registry mutation.

No credential, account identifier, native path, URL supplied by a user, or live MCP
response is recorded in this evidence.
