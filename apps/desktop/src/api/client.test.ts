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

  it("decodes the stable Yorva error envelope", async () => {
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
          supportedRange: "=0.20.2",
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

	it("keeps one ephemeral channel session in memory and sends the WeCom secret only in the write body", async () => {
		const secret = "wecom-client-secret-sentinel";
		const fetchMock = vi.fn()
			.mockResolvedValueOnce(new Response(JSON.stringify({ id: "op_channel" }), { status: 202, headers: { "Content-Type": "application/json" } }))
			.mockResolvedValueOnce(new Response(JSON.stringify({ payload: "qr-source", expiresAt: "2026-08-20T12:01:00Z" }), { status: 200, headers: { "Content-Type": "application/json" } }));
		vi.stubGlobal("fetch", fetchMock);
		const client = createDaemonClient(session);

		await client.connectWeCom("inst_coder", "bot-one", secret, "connect-wecom-1");
		await client.getChannelQr("op_channel");

		const [connectURL, connectInit] = fetchMock.mock.calls[0] as [string, RequestInit];
		const [qrURL, qrInit] = fetchMock.mock.calls[1] as [string, RequestInit];
		const connectHeaders = connectInit.headers as Record<string, string>;
		const qrHeaders = qrInit.headers as Record<string, string>;
		expect(connectURL).not.toContain(secret);
		expect(qrURL).not.toContain(secret);
		expect(JSON.stringify(connectHeaders)).not.toContain(secret);
		expect(connectHeaders["Yorva-Session-Id"]).toMatch(/^[A-Za-z0-9_-]{20,128}$/);
		expect(qrHeaders["Yorva-Session-Id"]).toBe(connectHeaders["Yorva-Session-Id"]);
		expect(JSON.parse(String(connectInit.body))).toEqual({ botId: "bot-one", secret });
	});

  it("sends a pairing code only in the no-store approval body", async () => {
    const code = "ABCD2345";
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ approved: true }), { status: 200, headers: { "Content-Type": "application/json" } }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await createDaemonClient(session).approveWeixinPairing("inst_coder", code);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toBe("http://127.0.0.1:49152/api/v1/instances/inst_coder/channels/weixin/pairings/approve");
    expect(url).not.toContain(code);
    expect(JSON.stringify(init.headers)).not.toContain(code);
    expect(init.cache).toBe("no-store");
    expect(JSON.parse(String(init.body))).toEqual({ code });
  });

  it("reads, saves, and resets Hermes download sources through the authenticated settings route", async () => {
    const sources = {
      hermesArchiveUrl: "https://github.com/example/hermes.zip",
      nodeArchiveUrl: "https://npmmirror.com/mirrors/node/node.zip",
      npmArchiveUrl: "https://registry.npmmirror.com/npm/-/npm.tgz",
      pythonIndexUrl: "https://pypi.tuna.tsinghua.edu.cn/simple",
      npmRegistryUrl: "https://registry.npmmirror.com",
    };
    const fetchMock = vi.fn().mockImplementation(async () =>
      new Response(JSON.stringify(sources), { status: 200, headers: { "Content-Type": "application/json" } }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const client = createDaemonClient(session);

    await client.getHermesDownloadSources();
    await client.saveHermesDownloadSources(sources);
    await client.resetHermesDownloadSources();

    const [getURL, getInit] = fetchMock.mock.calls[0] as [string, RequestInit];
    const [putURL, putInit] = fetchMock.mock.calls[1] as [string, RequestInit];
    const [deleteURL, deleteInit] = fetchMock.mock.calls[2] as [string, RequestInit];
    expect(getURL).toBe("http://127.0.0.1:49152/api/v1/settings/hermes/download-sources");
    expect(getInit.cache).toBe("no-store");
    expect(putURL).toBe(getURL);
    expect(putInit.method).toBe("PUT");
    expect(JSON.parse(String(putInit.body))).toEqual(sources);
    expect(deleteURL).toBe(getURL);
    expect(deleteInit.method).toBe("DELETE");
    for (const init of [getInit, putInit, deleteInit]) {
      expect(init.headers).toEqual(expect.objectContaining({ Authorization: "Bearer session-secret" }));
    }
  });
});
