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
});
