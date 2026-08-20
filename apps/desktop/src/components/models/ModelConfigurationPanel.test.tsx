import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { YorvaApiError, type DaemonClient } from "../../api/client";
import type { Instance, ModelConfiguration } from "../../api/types";
import { messages } from "../../i18n";
import { ModelConfigurationPanel } from "./ModelConfigurationPanel";

const instance: Instance = {
  instanceId: "inst_coder",
  runtimeInstallationId: "rtinst_test",
  name: "coder",
  default: false,
  protected: false,
  availability: "AVAILABLE",
  lastSyncedAt: "2026-08-19T12:00:00Z",
  createdAt: "2026-08-19T12:00:00Z",
  updatedAt: "2026-08-19T12:00:00Z",
  capabilities: { instances: true, lifecycle: false },
};

const unconfigured: ModelConfiguration = {
  providerPresetId: "",
  modelId: "",
  state: "UNCONFIGURED",
  credentialConfigured: false,
  observedAt: "2026-08-19T12:00:00Z",
  validation: { state: "NOT_RUN", errorCode: null, completedAt: null },
};

function modelClient(overrides: Partial<DaemonClient> = {}) {
  return {
    scope: "http://127.0.0.1:49152",
    listModelProviderPresets: vi.fn().mockResolvedValue({
      items: [
        { id: "deepseek", displayName: "DeepSeek", region: "CHINA", recommendedModels: ["deepseek-v4-pro"] },
        { id: "qwen", displayName: "Qwen / Alibaba DashScope", region: "CHINA", recommendedModels: ["qwen3.7-max"], helpText: "Hermes 0.20.2 uses the DashScope international compatible endpoint." },
        { id: "kimi", displayName: "Kimi / Moonshot (China)", region: "CHINA", recommendedModels: ["kimi-k3"] },
        { id: "minimax", displayName: "MiniMax (China)", region: "CHINA", recommendedModels: ["MiniMax-M3"] },
        { id: "glm", displayName: "GLM / Zhipu", region: "CHINA", recommendedModels: ["glm-5.2"] },
        { id: "openrouter", displayName: "OpenRouter", region: "GLOBAL", recommendedModels: ["openai/gpt-5.4"] },
        { id: "openai", displayName: "OpenAI", region: "GLOBAL", recommendedModels: ["gpt-5.4"] },
        { id: "anthropic", displayName: "Anthropic", region: "GLOBAL", recommendedModels: ["claude-sonnet-5"] },
      ],
    }),
    getModelConfiguration: vi.fn().mockResolvedValue(unconfigured),
    getModelCredential: vi.fn().mockResolvedValue({ providerPresetId: "", configured: false, observedAt: "2026-08-19T12:00:00Z" }),
    listOperations: vi.fn().mockResolvedValue({ operations: [] }),
    getOperation: vi.fn(),
    saveModelCredential: vi.fn().mockResolvedValue({ ...unconfigured, providerPresetId: "deepseek", modelId: "deepseek-v4-pro", state: "CONFIGURED", credentialConfigured: true }),
    patchModelConfiguration: vi.fn().mockResolvedValue(unconfigured),
    deleteModelCredential: vi.fn(),
    startModelValidation: vi.fn(),
    cancelOperation: vi.fn(),
    ...overrides,
  } as unknown as DaemonClient;
}

function renderPanel(client: DaemonClient, locale: "en-US" | "zh-CN" = "en-US") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const rendered = render(
    <QueryClientProvider client={queryClient}>
      <ModelConfigurationPanel
        client={client}
        instance={instance}
        copy={messages[locale]}
        locale={locale}
        onClose={() => undefined}
      />
    </QueryClientProvider>,
  );
  return { ...rendered, queryClient };
}

describe("ModelConfigurationPanel", () => {
  it("shows the China-first bilingual preset groups and safe status labels", async () => {
    renderPanel(modelClient(), "zh-CN");
    expect(await screen.findByText("国内推荐")).toBeInTheDocument();
    expect(screen.getByText("其他兼容 Provider")).toBeInTheDocument();
    expect(await screen.findByText("DeepSeek")).toBeInTheDocument();
    for (const provider of ["DeepSeek", "Qwen / Alibaba DashScope", "Kimi / Moonshot (China)", "MiniMax (China)", "GLM / Zhipu", "OpenRouter", "OpenAI", "Anthropic"]) {
      expect(screen.getByText(provider)).toBeInTheDocument();
    }
    expect(screen.getByText("未配置")).toBeInTheDocument();
    expect(screen.getByText("未测试")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("radio", { name: "Qwen / Alibaba DashScope" }));
    expect(screen.getByText("Hermes 0.20.2 使用 DashScope 国际兼容端点。")).toBeInTheDocument();
    expect(screen.queryByText("Hermes 0.20.2 uses the DashScope international compatible endpoint.")).not.toBeInTheDocument();
  });

  it("clears the password after save and never places it in query or browser storage", async () => {
    const secret = "sk-desktop-phase5-do-not-persist";
    const client = modelClient();
    const { queryClient } = renderPanel(client);
    fireEvent.click(await screen.findByRole("radio", { name: "DeepSeek" }));
    const password = screen.getByLabelText("API Key") as HTMLInputElement;
    fireEvent.change(password, { target: { value: secret } });
    expect(password.value).toBe(secret);
    fireEvent.click(screen.getByRole("button", { name: "Save configuration" }));

    await waitFor(() => expect(client.saveModelCredential).toHaveBeenCalledWith("inst_coder", "deepseek", "deepseek-v4-pro", secret));
    await waitFor(() => expect(password.value).toBe(""));
    expect(screen.getByText("Configuration saved")).toBeInTheDocument();
    expect(JSON.stringify(queryClient.getQueryCache().getAll().map((query) => ({ key: query.queryKey, data: query.state.data })))).not.toContain(secret);
    expect(JSON.stringify(window.localStorage)).not.toContain(secret);
    expect(JSON.stringify(window.sessionStorage)).not.toContain(secret);
  });

  it("clears the password and shows the stable incomplete code after a partial save", async () => {
    const secret = "sk-partial-desktop-do-not-persist";
    const client = modelClient({
      saveModelCredential: vi.fn().mockRejectedValue(new YorvaApiError({
        code: "MODEL_CONFIG_INCOMPLETE",
        message: "The credential was saved but the model configuration is incomplete.",
        retryable: true,
        details: { configuration: unconfigured },
      })),
    });
    renderPanel(client);
    fireEvent.click(await screen.findByRole("radio", { name: "DeepSeek" }));
    const password = screen.getByLabelText("API Key") as HTMLInputElement;
    fireEvent.change(password, { target: { value: secret } });
    fireEvent.click(screen.getByRole("button", { name: "Save configuration" }));

    await waitFor(() => expect(client.saveModelCredential).toHaveBeenCalled());
    await waitFor(() => expect(password.value).toBe(""));
    expect(screen.getByRole("alert")).toHaveTextContent("MODEL_CONFIG_INCOMPLETE");
    expect(JSON.stringify(window.localStorage)).not.toContain(secret);
    expect(JSON.stringify(window.sessionStorage)).not.toContain(secret);
  });

  it("starts validation only from the explicit button and shows local-time result metadata", async () => {
    const configured: ModelConfiguration = {
      providerPresetId: "deepseek",
      modelId: "deepseek-v4-pro",
      state: "CONFIGURED",
      credentialConfigured: true,
      observedAt: "2026-08-19T12:00:00Z",
      validation: { state: "PASSED", errorCode: null, completedAt: "2026-08-19T12:01:00Z" },
    };
    const client = modelClient({
      getModelConfiguration: vi.fn().mockResolvedValue(configured),
      getModelCredential: vi.fn().mockResolvedValue({ providerPresetId: "deepseek", configured: true, observedAt: "2026-08-19T12:00:00Z" }),
      startModelValidation: vi.fn().mockResolvedValue({ id: "op_validate", type: "model.validate", targetType: "instance", targetId: "inst_coder", status: "PENDING", stage: "preflight", progress: null, message: "", errorCode: null, retryable: false, correlationId: "cor", createdAt: "2026-08-19T12:02:00Z", startedAt: null, completedAt: null, updatedAt: "2026-08-19T12:02:00Z" }),
      getOperation: vi.fn().mockResolvedValue({ id: "op_validate", type: "model.validate", targetType: "instance", targetId: "inst_coder", status: "SUCCEEDED", stage: "model.validate", progress: null, message: "", errorCode: null, retryable: false, correlationId: "cor", createdAt: "2026-08-19T12:02:00Z", startedAt: "2026-08-19T12:02:00Z", completedAt: "2026-08-19T12:02:01Z", updatedAt: "2026-08-19T12:02:01Z" }),
    });
    renderPanel(client);
    expect(await screen.findByText("Passed")).toBeInTheDocument();
    expect(client.startModelValidation).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Test connection" }));
    await waitFor(() => expect(client.startModelValidation).toHaveBeenCalledTimes(1));
    expect(screen.getByText(/Last tested:/)).toBeInTheDocument();
  });

  it("renders only a stable code and safe advice for a failed validation", async () => {
    const client = modelClient({
      getModelConfiguration: vi.fn().mockResolvedValue({
        ...unconfigured,
        providerPresetId: "deepseek",
        modelId: "deepseek-v4-pro",
        state: "CONFIGURED",
        credentialConfigured: true,
        validation: { state: "FAILED", errorCode: "MODEL_VALIDATION_FAILED", completedAt: "2026-08-19T12:01:00Z" },
      }),
    });
    renderPanel(client);
    expect(await screen.findByText(/MODEL_VALIDATION_FAILED/)).toHaveTextContent(
      "Check the Provider key and model access, then run the test again.",
    );
  });

  it("disables model mutations when the Hermes model surface version is unsupported", async () => {
    const client = modelClient({
      getModelConfiguration: vi.fn().mockRejectedValue(new YorvaApiError({
        code: "MODEL_PROVIDER_UNSUPPORTED",
        message: "The model provider surface is unsupported.",
        retryable: false,
        details: {},
      })),
    });
    renderPanel(client);
    expect(await screen.findByText(messages["en-US"].models.unsupported)).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "DeepSeek" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Save configuration" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Save configuration" }));
    expect(client.patchModelConfiguration).not.toHaveBeenCalled();
    expect(client.saveModelCredential).not.toHaveBeenCalled();
  });
});
