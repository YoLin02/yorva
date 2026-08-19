import { afterEach, describe, expect, it, vi } from "vitest";
import { createDaemonClient, YorvaApiError } from "./client";

const session = {
  baseUrl: "http://127.0.0.1:49152",
  token: "session-secret",
  protocolVersion: "1",
};

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("daemon client", () => {
  it("authenticates Node requests without putting the token in the URL", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: "node_test" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await createDaemonClient(session).getNode();

    expect(fetchMock).toHaveBeenCalledWith(
      "http://127.0.0.1:49152/api/v1/node",
      expect.objectContaining({
        headers: expect.objectContaining({ Authorization: "Bearer session-secret" }),
      }),
    );
    expect(fetchMock.mock.calls[0][0]).not.toContain(session.token);
  });

  it("decodes the stable YORVA error envelope", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: { code: "UNAUTHORIZED", message: "Authentication is required.", retryable: false, details: {} },
          }),
          { status: 401, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );

    await expect(createDaemonClient(session).getNode()).rejects.toMatchObject({
      code: "UNAUTHORIZED",
      retryable: false,
    } satisfies Partial<YorvaApiError>);
  });

  it("performs authenticated Hermes discovery with cancellation support", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          runtimeKind: "hermes",
          state: "NOT_INSTALLED",
          errorCode: "RUNTIME_NOT_INSTALLED",
          selected: null,
          candidates: [],
          warnings: [],
          detectedAt: "2026-08-14T00:00:00Z",
          supportedRange: ">=0.19.0 <0.21.0",
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);
    const controller = new AbortController();

    await createDaemonClient(session).detectHermes(controller.signal);

    expect(fetchMock).toHaveBeenCalledWith(
      "http://127.0.0.1:49152/api/v1/runtimes/hermes/detect",
      expect.objectContaining({
        method: "POST",
        signal: expect.any(AbortSignal),
        headers: expect.objectContaining({ Authorization: "Bearer session-secret" }),
      }),
    );
    const requestSignal = (fetchMock.mock.calls[0][1] as RequestInit).signal as AbortSignal;
    expect(requestSignal.aborted).toBe(false);
    controller.abort();
    expect(requestSignal.aborted).toBe(true);
  });

  it("uses the write-only credential route without putting the secret in URL or headers", async () => {
    const secret = "sk-client-phase5-do-not-leak";
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({
        providerPresetId: "deepseek",
        modelId: "deepseek-v4-pro",
        state: "CONFIGURED",
        credentialConfigured: true,
        observedAt: "2026-08-19T12:00:00Z",
        validation: { state: "NOT_RUN", errorCode: null, completedAt: null },
      }), { status: 200, headers: { "Content-Type": "application/json" } }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await createDaemonClient(session).saveModelCredential("inst_coder", "deepseek", "deepseek-v4-pro", secret);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://127.0.0.1:49152/api/v1/instances/inst_coder/credentials/model-provider");
    expect(url).not.toContain(secret);
    expect(JSON.stringify(init.headers)).not.toContain(secret);
    expect(init.method).toBe("PUT");
    expect(JSON.parse(String(init.body))).toEqual({
      providerPresetId: "deepseek",
      modelId: "deepseek-v4-pro",
      value: secret,
    });
  });
});
