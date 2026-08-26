import { afterEach, describe, expect, expectTypeOf, it, vi } from "vitest";
import { createDaemonClient } from "./client";
import type { DaemonClient } from "./client";
import type { ManagementLogCategory } from "./types";

const session = {
  baseUrl: "http://127.0.0.1:49152",
  token: "session-secret",
  protocolVersion: "1",
};

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("daemon client management reads", () => {
  it("types the log category from the generated closed union", () => {
    expectTypeOf<ManagementLogCategory>().toEqualTypeOf<"RUNTIME" | "ERRORS" | "GATEWAY" | "MCP">();
    expectTypeOf<Parameters<DaemonClient["getInstanceLogSnapshot"]>[1]>().toEqualTypeOf<ManagementLogCategory>();
  });

  it("uses encoded Instance and category values for authenticated health and log GET requests", async () => {
    const fetchMock = vi.fn().mockImplementation(async () =>
      new Response(JSON.stringify({ entries: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const client = createDaemonClient(session);

    await client.getInstanceHealth("instance/a b");
    await client.getInstanceLogSnapshot("instance/a b", "MCP&ignored=value" as ManagementLogCategory);

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "http://127.0.0.1:49152/api/v1/instances/instance%2Fa%20b/health",
      "http://127.0.0.1:49152/api/v1/instances/instance%2Fa%20b/logs?category=MCP%26ignored%3Dvalue",
    ]);
    for (const [, init] of fetchMock.mock.calls as [string, RequestInit][]) {
      expect(init.method ?? "GET").toBe("GET");
      expect(init.body).toBeUndefined();
      expect(init.headers).toEqual(expect.objectContaining({ Authorization: "Bearer session-secret" }));
    }
  });

  it("uses encoded Instance and Skill path segments with authenticated GET requests", async () => {
    const fetchMock = vi.fn().mockImplementation(async () =>
      new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const client = createDaemonClient(session);

    await client.listInstanceSkills("instance/a b");
    await client.inspectInstanceSkill("instance/a b", "skill/id?#");
    await client.listInstanceMCPServers("instance/a b");
    await client.listRuntimeMCPDefinitions("hermes");

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "http://127.0.0.1:49152/api/v1/instances/instance%2Fa%20b/skills",
      "http://127.0.0.1:49152/api/v1/instances/instance%2Fa%20b/skills/skill%2Fid%3F%23",
      "http://127.0.0.1:49152/api/v1/instances/instance%2Fa%20b/mcp-bindings",
      "http://127.0.0.1:49152/api/v1/runtimes/hermes/mcp-definitions",
    ]);
    for (const [, init] of fetchMock.mock.calls as [string, RequestInit][]) {
      expect(init.method ?? "GET").toBe("GET");
      expect(init.body).toBeUndefined();
      expect(init.headers).toEqual(expect.objectContaining({ Authorization: "Bearer session-secret" }));
    }
  });

  it("uses closed managed Skill mutation bodies and idempotency headers", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => new Response(JSON.stringify({ id: "op-skill" }), { status: 202 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = createDaemonClient(session);

    await client.listInstanceSkillSources("instance/a b");
    await client.installManagedSkill("instance/a b", "skill/id", "approved-source", "install-key");
    await client.updateManagedSkill("instance/a b", "skill/id", "update-key");
    await client.enableManagedSkill("instance/a b", "skill/id", "enable-key");
    await client.disableManagedSkill("instance/a b", "skill/id", "disable-key");
    await client.removeManagedSkill("instance/a b", "skill/id", "remove-key");

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "http://127.0.0.1:49152/api/v1/instances/instance%2Fa%20b/skill-sources",
      "http://127.0.0.1:49152/api/v1/instances/instance%2Fa%20b/skills/skill%2Fid/install",
      "http://127.0.0.1:49152/api/v1/instances/instance%2Fa%20b/skills/skill%2Fid/update",
      "http://127.0.0.1:49152/api/v1/instances/instance%2Fa%20b/skills/skill%2Fid/enable",
      "http://127.0.0.1:49152/api/v1/instances/instance%2Fa%20b/skills/skill%2Fid/disable",
      "http://127.0.0.1:49152/api/v1/instances/instance%2Fa%20b/skills/skill%2Fid",
    ]);
    expect((fetchMock.mock.calls[1][1] as RequestInit).body).toBe(JSON.stringify({ sourceId: "approved-source" }));
    for (let index = 2; index < fetchMock.mock.calls.length; index += 1) {
      expect((fetchMock.mock.calls[index][1] as RequestInit).body).toBe("{}");
    }
    expect((fetchMock.mock.calls[1][1] as RequestInit).headers).toEqual(expect.objectContaining({ "Idempotency-Key": "install-key", Authorization: "Bearer session-secret" }));
    expect((fetchMock.mock.calls[5][1] as RequestInit).method).toBe("DELETE");
  });

  it("bounds management reads and propagates caller cancellation", async () => {
    const timeoutController = new AbortController();
    const timeoutSpy = vi.spyOn(AbortSignal, "timeout").mockReturnValue(timeoutController.signal);
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const callerController = new AbortController();

    await createDaemonClient(session).getInstanceHealth("instance-one", callerController.signal);

    expect(timeoutSpy).toHaveBeenCalledWith(15_000);
    const requestSignal = (fetchMock.mock.calls[0][1] as RequestInit).signal as AbortSignal;
    expect(requestSignal.aborted).toBe(false);
    callerController.abort();
    expect(requestSignal.aborted).toBe(true);
  });

  it("reads the encoded Runtime upgrade plan with authenticated bounded GET", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ state: "UNKNOWN" }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);

    await createDaemonClient(session).getRuntimeUpgradePlan("hermes/unsafe?");

    expect(fetchMock).toHaveBeenCalledWith(
      "http://127.0.0.1:49152/api/v1/runtimes/hermes%2Funsafe%3F/upgrade-plan",
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.method ?? "GET").toBe("GET");
    expect(init.body).toBeUndefined();
    expect(init.headers).toEqual(expect.objectContaining({ Authorization: "Bearer session-secret" }));
  });

  it("reads only encoded Runtime backup inventory resources", async () => {
    const fetchMock = vi.fn().mockImplementation(async () => new Response(JSON.stringify({ scope: "RUNTIME", items: [] }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = createDaemonClient(session);

    await client.listRuntimeBackups("hermes/unsafe?");
    await client.getRuntimeBackup("hermes/unsafe?", "backup/id#");

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "http://127.0.0.1:49152/api/v1/runtimes/hermes%2Funsafe%3F/backups",
      "http://127.0.0.1:49152/api/v1/runtimes/hermes%2Funsafe%3F/backups/backup%2Fid%23",
    ]);
    for (const [, init] of fetchMock.mock.calls as [string, RequestInit][]) {
      expect(init.method ?? "GET").toBe("GET");
      expect(init.body).toBeUndefined();
      expect(init.headers).toEqual(expect.objectContaining({ Authorization: "Bearer session-secret" }));
    }
  });

  it("aborts management reads when the bounded Desktop timeout fires", async () => {
    const timeoutController = new AbortController();
    vi.spyOn(AbortSignal, "timeout").mockReturnValue(timeoutController.signal);
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ items: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await createDaemonClient(session).getInstanceLogSnapshot("instance-one", "ERRORS");

    const requestSignal = (fetchMock.mock.calls[0][1] as RequestInit).signal as AbortSignal;
    expect(requestSignal.aborted).toBe(false);
    timeoutController.abort(new DOMException("The operation timed out.", "TimeoutError"));
    expect(requestSignal.aborted).toBe(true);
    expect(requestSignal.reason).toMatchObject({ name: "TimeoutError" });
  });
});
