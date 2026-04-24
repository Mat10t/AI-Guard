"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Button, Card, EmptyState, Field, Notice, SectionHeader } from "../../components/ui";
import { apiRequest } from "../../lib/api";
import { useAuthGuard } from "../../lib/guard";
import { capabilitiesForRole, firstAllowedRoute } from "../../lib/rbac";
import { useSession } from "../../lib/session";
import { AnalyticsSectionNav, describeError } from "../analytics-common";

type ProviderStatus = {
  provider: string;
  status: string;
  checked_at: string;
  latency_ms: number;
  error?: string;
};

type ProviderModel = {
  id: string;
  provider: string;
  status: string;
  input_cost: number;
  output_cost: number;
  pricing_source: string;
  pricing_updated_at: string;
};

type PricingDraft = {
  input_cost: string;
  output_cost: string;
  pricing_source: string;
};

function formatCheckedAt(value: string): string {
  const raw = String(value || "").trim();
  if (!raw) {
    return "-";
  }
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) {
    return raw;
  }
  return d.toLocaleString("ru-RU");
}

export default function AnalyticsProvidersPage() {
  const guard = useAuthGuard();
  const session = useSession();
  const router = useRouter();
  const caps = useMemo(() => capabilitiesForRole(session.claims?.role), [session.claims?.role]);

  const [providers, setProviders] = useState<ProviderStatus[]>([]);
  const [models, setModels] = useState<ProviderModel[]>([]);
  const [drafts, setDrafts] = useState<Record<string, PricingDraft>>({});
  const [loading, setLoading] = useState(false);
  const [savingModelID, setSavingModelID] = useState("");
  const [notice, setNotice] = useState("Статусы провайдеров, задержка и управление ценами моделей.");

  useEffect(() => {
    if (!guard.isReady) {
      return;
    }
    if (!caps.showAnalytics) {
      router.replace(firstAllowedRoute(session.claims?.role));
      return;
    }
    void bootstrap();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [guard.isReady, caps.showAnalytics]);

  function updateDraft(modelID: string, patch: Partial<PricingDraft>) {
    setDrafts((prev) => ({
      ...prev,
      [modelID]: {
        input_cost: prev[modelID]?.input_cost ?? "",
        output_cost: prev[modelID]?.output_cost ?? "",
        pricing_source: prev[modelID]?.pricing_source ?? "manual",
        ...patch
      }
    }));
  }

  async function bootstrap() {
    setLoading(true);
    const [statusResult, modelsResult] = await Promise.all([
      apiRequest({ path: "/catalog/providers/status", method: "GET" }),
      apiRequest({ path: "/catalog/models", method: "GET" })
    ]);
    setLoading(false);

    if (!statusResult.ok) {
      setNotice(describeError(statusResult));
      return;
    }
    if (!modelsResult.ok) {
      setNotice(describeError(modelsResult));
      return;
    }

    const nextProviders = ((statusResult.data?.items as ProviderStatus[]) || []).map((item) => ({
      provider: String(item.provider || "").trim(),
      status: String(item.status || "").trim(),
      checked_at: String(item.checked_at || "").trim(),
      latency_ms: Number(item.latency_ms || 0),
      error: String(item.error || "").trim()
    }));

    const nextModels = ((modelsResult.data?.items as ProviderModel[]) || []).map((item) => ({
      id: String(item.id || "").trim(),
      provider: String(item.provider || "").trim(),
      status: String(item.status || "").trim(),
      input_cost: Number(item.input_cost || 0),
      output_cost: Number(item.output_cost || 0),
      pricing_source: String(item.pricing_source || "seed").trim(),
      pricing_updated_at: String(item.pricing_updated_at || "").trim()
    }));

    const nextDrafts: Record<string, PricingDraft> = {};
    for (const model of nextModels) {
      nextDrafts[model.id] = {
        input_cost: String(model.input_cost),
        output_cost: String(model.output_cost),
        pricing_source: model.pricing_source || "manual"
      };
    }

    setProviders(nextProviders);
    setModels(nextModels);
    setDrafts(nextDrafts);
    setNotice("Статусы провайдеров и каталог цен обновлены.");
  }

  async function forceRefreshProviders() {
    setLoading(true);
    const result = await apiRequest({
      path: "/catalog/providers/status?refresh=1",
      method: "GET"
    });
    setLoading(false);

    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }

    setProviders(
      ((result.data?.items as ProviderStatus[]) || []).map((item) => ({
        provider: String(item.provider || "").trim(),
        status: String(item.status || "").trim(),
        checked_at: String(item.checked_at || "").trim(),
        latency_ms: Number(item.latency_ms || 0),
        error: String(item.error || "").trim()
      }))
    );
    setNotice("Live-проверка провайдеров выполнена.");
  }

  async function savePricing(modelID: string) {
    if (!caps.canUpdatePricing) {
      setNotice("Только Admin может обновлять цены моделей.");
      return;
    }
    const draft = drafts[modelID];
    if (!draft) {
      setNotice("Драфт цены для модели не найден.");
      return;
    }

    const inputCost = Number(draft.input_cost);
    const outputCost = Number(draft.output_cost);
    if (!Number.isFinite(inputCost) || inputCost < 0 || !Number.isFinite(outputCost) || outputCost < 0) {
      setNotice("Стоимость должна быть неотрицательным числом.");
      return;
    }

    setSavingModelID(modelID);
    const result = await session.authRequest({
      path: `/catalog/models/${modelID}/pricing`,
      method: "PUT",
      body: {
        input_cost: inputCost,
        output_cost: outputCost,
        pricing_source: String(draft.pricing_source || "manual").trim() || "manual"
      }
    });
    setSavingModelID("");

    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }

    setModels((prev) =>
      prev.map((item) => {
        if (item.id !== modelID) {
          return item;
        }
        return {
          ...item,
          input_cost: Number(result.data?.input_cost || item.input_cost),
          output_cost: Number(result.data?.output_cost || item.output_cost),
          pricing_source: String(result.data?.pricing_source || item.pricing_source),
          pricing_updated_at: String(result.data?.pricing_updated_at || item.pricing_updated_at)
        };
      })
    );
    const affected = Number(result.data?.affected_project_limits || 0);
    setNotice(`Цена модели ${modelID} обновлена. Автопересчитано ${affected} project-лимитов.`);
  }

  if (!guard.isReady) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Loading" title="Providers" subtitle="Проверяем доступ к аналитике..." />
        </Card>
      </main>
    );
  }

  if (!caps.showAnalytics) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Redirect" title="Providers недоступен" subtitle="Выполняем переход в доступный раздел." />
        </Card>
      </main>
    );
  }

  return (
    <main>
      <Card>
        <SectionHeader
          badge="Analytics / Providers"
          title="Provider status + pricing"
          subtitle="Live-статус провайдеров (latency/error) и каталог цен моделей."
          actions={
            <div className="row compact">
              <Button variant="secondary" onClick={() => void bootstrap()} disabled={loading}>
                Обновить каталог
              </Button>
              <Button variant="secondary" onClick={() => void forceRefreshProviders()} disabled={loading}>
                Обновить статус
              </Button>
            </div>
          }
        />
        <AnalyticsSectionNav caps={caps} />
        <Notice tone="info">{notice}</Notice>
      </Card>

      <Card>
        <h2>Состояние провайдеров</h2>
        {providers.length === 0 ? (
          <EmptyState title="Нет данных" description="Проверьте доступность каталога и обновите статус." />
        ) : (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Provider</th>
                  <th>Status</th>
                  <th>Latency</th>
                  <th>Checked</th>
                  <th>Error</th>
                </tr>
              </thead>
              <tbody>
                {providers.map((item) => (
                  <tr key={item.provider}>
                    <td>
                      <strong>{item.provider}</strong>
                    </td>
                    <td>{item.status}</td>
                    <td>{item.latency_ms} ms</td>
                    <td>{formatCheckedAt(item.checked_at)}</td>
                    <td>{item.error || "-"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Card>
        <h2>Каталог цен моделей</h2>
        {models.length === 0 ? (
          <EmptyState title="Каталог пуст" description="Модели появятся после инициализации provider catalog." />
        ) : (
          <div className="stack">
            {models.map((model) => {
              const draft = drafts[model.id] || {
                input_cost: String(model.input_cost),
                output_cost: String(model.output_cost),
                pricing_source: model.pricing_source || "manual"
              };

              return (
                <div className="subcard" key={model.id}>
                  <div className="table-wrap">
                    <table className="data-table">
                      <thead>
                        <tr>
                          <th>Model</th>
                          <th>Provider</th>
                          <th>Status</th>
                          <th>Input / 1K</th>
                          <th>Output / 1K</th>
                          <th>Source</th>
                          <th>Updated</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr>
                          <td>{model.id}</td>
                          <td>{model.provider}</td>
                          <td>{model.status}</td>
                          <td>{model.input_cost}</td>
                          <td>{model.output_cost}</td>
                          <td>{model.pricing_source}</td>
                          <td>{formatCheckedAt(model.pricing_updated_at)}</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>

                  {caps.canUpdatePricing ? (
                    <>
                      <div className="grid two">
                        <Field label="Input cost (USD / 1K)">
                          <input
                            type="number"
                            min={0}
                            step="0.000001"
                            value={draft.input_cost}
                            onChange={(e) => updateDraft(model.id, { input_cost: e.target.value })}
                          />
                        </Field>
                        <Field label="Output cost (USD / 1K)">
                          <input
                            type="number"
                            min={0}
                            step="0.000001"
                            value={draft.output_cost}
                            onChange={(e) => updateDraft(model.id, { output_cost: e.target.value })}
                          />
                        </Field>
                      </div>
                      <Field label="Pricing source">
                        <input
                          value={draft.pricing_source}
                          onChange={(e) => updateDraft(model.id, { pricing_source: e.target.value })}
                          placeholder="manual"
                        />
                      </Field>
                      <Button
                        variant="secondary"
                        onClick={() => void savePricing(model.id)}
                        disabled={loading || savingModelID === model.id}
                      >
                        {savingModelID === model.id ? "Сохраняем..." : "Сохранить цену"}
                      </Button>
                    </>
                  ) : null}
                </div>
              );
            })}
          </div>
        )}
      </Card>
    </main>
  );
}
