"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis
} from "recharts";
import { Button, Card, EmptyState, Notice, SectionHeader } from "../../components/ui";
import { useAuthGuard } from "../../lib/guard";
import { capabilitiesForRole, firstAllowedRoute } from "../../lib/rbac";
import { useSession } from "../../lib/session";
import {
  AnalyticsScope,
  AnalyticsScopeControls,
  AnalyticsSectionNav,
  defaultScopeForCaps,
  describeError,
  downloadCSV,
  firstAnalyticsSection
} from "../analytics-common";

type UsageRow = {
  group: string;
  total_tokens: number;
  total_cost: number;
};

type SeriesPoint = {
  bucket_start: string;
  value: number;
};

type ScopeProject = {
  id: string;
  name: string;
};

type ScopeKey = {
  id: string;
  status: string;
  prefix: string;
};

const SCALE_OPTIONS = ["hour", "day", "week", "month", "year", "all"] as const;
type Scale = (typeof SCALE_OPTIONS)[number];

function formatBucket(value: string): string {
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) {
    return value;
  }
  return d.toLocaleDateString("ru-RU", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  });
}

function normalizeSeries(result: any): SeriesPoint[] {
  const items = (result?.items as any[]) || [];
  return items.map((row) => ({
    bucket_start: String(row?.bucket_start || "").trim(),
    value: Number(row?.value || 0)
  }));
}

function mergeSeries(
  inputTokens: SeriesPoint[],
  outputTokens: SeriesPoint[],
  cost: SeriesPoint[],
  errorRate: SeriesPoint[],
  fallbackRate: SeriesPoint[]
): Array<{
  bucket_start: string;
  input_tokens: number;
  output_tokens: number;
  cost: number;
  error_rate: number;
  fallback_rate: number;
}> {
  const map = new Map<
    string,
    { bucket_start: string; input_tokens: number; output_tokens: number; cost: number; error_rate: number; fallback_rate: number }
  >();

  for (const point of inputTokens) {
    const existing = map.get(point.bucket_start) || {
      bucket_start: point.bucket_start,
      input_tokens: 0,
      output_tokens: 0,
      cost: 0,
      error_rate: 0,
      fallback_rate: 0
    };
    existing.input_tokens = point.value;
    map.set(point.bucket_start, existing);
  }
  for (const point of outputTokens) {
    const existing = map.get(point.bucket_start) || {
      bucket_start: point.bucket_start,
      input_tokens: 0,
      output_tokens: 0,
      cost: 0,
      error_rate: 0,
      fallback_rate: 0
    };
    existing.output_tokens = point.value;
    map.set(point.bucket_start, existing);
  }
  for (const point of cost) {
    const existing = map.get(point.bucket_start) || {
      bucket_start: point.bucket_start,
      input_tokens: 0,
      output_tokens: 0,
      cost: 0,
      error_rate: 0,
      fallback_rate: 0
    };
    existing.cost = point.value;
    map.set(point.bucket_start, existing);
  }
  for (const point of errorRate) {
    const existing = map.get(point.bucket_start) || {
      bucket_start: point.bucket_start,
      input_tokens: 0,
      output_tokens: 0,
      cost: 0,
      error_rate: 0,
      fallback_rate: 0
    };
    existing.error_rate = point.value;
    map.set(point.bucket_start, existing);
  }
  for (const point of fallbackRate) {
    const existing = map.get(point.bucket_start) || {
      bucket_start: point.bucket_start,
      input_tokens: 0,
      output_tokens: 0,
      cost: 0,
      error_rate: 0,
      fallback_rate: 0
    };
    existing.fallback_rate = point.value;
    map.set(point.bucket_start, existing);
  }

  return Array.from(map.values()).sort((a, b) => a.bucket_start.localeCompare(b.bucket_start));
}

export default function AnalyticsUsagePage() {
  const guard = useAuthGuard();
  const session = useSession();
  const router = useRouter();

  const caps = useMemo(() => capabilitiesForRole(session.claims?.role), [session.claims?.role]);
  const [scope, setScope] = useState<AnalyticsScope>("org");
  const [scopeProjectID, setScopeProjectID] = useState("");
  const [scopeAPIKeyID, setScopeAPIKeyID] = useState("");
  const [scopeProjects, setScopeProjects] = useState<ScopeProject[]>([]);
  const [scopeKeys, setScopeKeys] = useState<ScopeKey[]>([]);
  const [scale, setScale] = useState<Scale>("day");
  const [groupBy, setGroupBy] = useState<"project" | "model">("model");
  const [usageRows, setUsageRows] = useState<UsageRow[]>([]);
  const [inputSeries, setInputSeries] = useState<SeriesPoint[]>([]);
  const [outputSeries, setOutputSeries] = useState<SeriesPoint[]>([]);
  const [costSeries, setCostSeries] = useState<SeriesPoint[]>([]);
  const [errorSeries, setErrorSeries] = useState<SeriesPoint[]>([]);
  const [fallbackSeries, setFallbackSeries] = useState<SeriesPoint[]>([]);
  const [loading, setLoading] = useState(false);
  const [notice, setNotice] = useState("Графики usage и стабильности запросов по выбранному масштабу времени.");

  useEffect(() => {
    setScope(defaultScopeForCaps(caps));
    setScopeProjectID("");
    setScopeAPIKeyID("");
  }, [caps.canUseOrgScope, session.claims?.role]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const query = new URLSearchParams(window.location.search);
    const scopeParam = String(query.get("scope") || "").trim();
    const projectParam = String(query.get("project_id") || "").trim();
    const keyParam = String(query.get("api_key_id") || "").trim();
    if (!scopeParam && !projectParam && !keyParam) {
      return;
    }

    let nextScope: AnalyticsScope = scopeParam === "key" ? "key" : scopeParam === "project" ? "project" : "org";
    if (nextScope === "org" && !caps.canUseOrgScope) {
      nextScope = "project";
    }

    setScope(nextScope);
    setScopeProjectID(nextScope === "org" ? "" : projectParam);
    setScopeAPIKeyID(nextScope === "key" ? keyParam : "");
  }, [caps.canUseOrgScope]);

  useEffect(() => {
    if (!guard.isReady) {
      return;
    }
    if (!caps.showAnalytics) {
      router.replace(firstAllowedRoute(session.claims?.role));
      return;
    }
    if (!caps.canViewUsage) {
      router.replace(firstAnalyticsSection(caps));
      return;
    }
    void loadScopeProjects();
    void loadData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [guard.isReady, caps.showAnalytics, caps.canViewUsage, scale, groupBy, scope, scopeProjectID, scopeAPIKeyID]);

  useEffect(() => {
    if (scope !== "key" || !scopeProjectID) {
      setScopeKeys([]);
      setScopeAPIKeyID("");
      return;
    }
    void loadScopeKeys(scopeProjectID);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scope, scopeProjectID]);

  const mergedSeries = useMemo(
    () => mergeSeries(inputSeries, outputSeries, costSeries, errorSeries, fallbackSeries),
    [inputSeries, outputSeries, costSeries, errorSeries, fallbackSeries]
  );

  function scopeQueryParams(): string {
    const query = new URLSearchParams();
    query.set("scope", scope);
    if (scope === "project" || scope === "key") {
      if (scopeProjectID) {
        query.set("project_id", scopeProjectID);
      }
    }
    if (scope === "key" && scopeAPIKeyID) {
      query.set("api_key_id", scopeAPIKeyID);
    }
    return query.toString();
  }

  async function loadScopeProjects() {
    const result = await session.authRequest({ path: "/projects", method: "GET" });
    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }
    const items = ((result.data?.items as any[]) || []).map((item) => ({
      id: String(item?.id || ""),
      name: String(item?.name || "")
    }));
    setScopeProjects(items.filter((item) => item.id));
  }

  async function loadScopeKeys(projectID: string) {
    const result = await session.authRequest({ path: `/projects/${projectID}/keys`, method: "GET" });
    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }
    const items = ((result.data?.items as any[]) || [])
      .map((item) => ({
        id: String(item?.id || ""),
        status: String(item?.status || ""),
        prefix: String(item?.prefix || "")
      }))
      .filter((item) => item.id && item.status === "active");
    if (scopeAPIKeyID && !items.some((item) => item.id === scopeAPIKeyID)) {
      setScopeAPIKeyID("");
      setNotice("Выбранный ключ недоступен. Выберите активный ключ.");
    }
    setScopeKeys(items);
  }

  async function loadData() {
    if ((scope === "project" || scope === "key") && !scopeProjectID) {
      setNotice("Выберите проект для project/key scope.");
      return;
    }
    if (scope === "key" && !scopeAPIKeyID) {
      setNotice("Выберите API-ключ для key scope.");
      return;
    }
    if (scope === "key" && scopeKeys.length > 0 && !scopeKeys.some((item) => item.id === scopeAPIKeyID)) {
      setScopeAPIKeyID("");
      setNotice("Выбранный ключ недоступен. Выберите активный ключ.");
      return;
    }
    setLoading(true);
    const scopeQuery = scopeQueryParams();

    const [usageResult, inputResult, outputResult, costResult, errorResult, fallbackResult] = await Promise.all([
      session.authRequest({ path: `/analytics/usage?group_by=${groupBy}&${scopeQuery}`, method: "GET" }),
      session.authRequest({ path: `/analytics/timeseries?metric=input_tokens&bucket=${scale}&${scopeQuery}`, method: "GET" }),
      session.authRequest({ path: `/analytics/timeseries?metric=output_tokens&bucket=${scale}&${scopeQuery}`, method: "GET" }),
      session.authRequest({ path: `/analytics/timeseries?metric=cost&bucket=${scale}&${scopeQuery}`, method: "GET" }),
      session.authRequest({ path: `/analytics/timeseries?metric=error_rate&bucket=${scale}&${scopeQuery}`, method: "GET" }),
      session.authRequest({ path: `/analytics/timeseries?metric=fallback_rate&bucket=${scale}&${scopeQuery}`, method: "GET" })
    ]);

    setLoading(false);

    if (!usageResult.ok) {
      setNotice(describeError(usageResult));
      return;
    }
    if (!inputResult.ok) {
      setNotice(describeError(inputResult));
      return;
    }
    if (!outputResult.ok) {
      setNotice(describeError(outputResult));
      return;
    }
    if (!costResult.ok) {
      setNotice(describeError(costResult));
      return;
    }
    if (!errorResult.ok) {
      setNotice(describeError(errorResult));
      return;
    }
    if (!fallbackResult.ok) {
      setNotice(describeError(fallbackResult));
      return;
    }

    setUsageRows(
      ((usageResult.data?.items as UsageRow[]) || []).map((row) => ({
        group: String(row.group || ""),
        total_tokens: Number(row.total_tokens || 0),
        total_cost: Number(row.total_cost || 0)
      }))
    );
    setInputSeries(normalizeSeries(inputResult.data));
    setOutputSeries(normalizeSeries(outputResult.data));
    setCostSeries(normalizeSeries(costResult.data));
    setErrorSeries(normalizeSeries(errorResult.data));
    setFallbackSeries(normalizeSeries(fallbackResult.data));
    setNotice("Usage и timeseries данные обновлены.");
  }

  async function exportUsageCSV() {
    if (!caps.canExportCSV) {
      return;
    }

    setLoading(true);
    const query = scopeQueryParams();
    const result = await session.authRequest({ path: `/reports/csv/usage?${query}`, method: "GET" });
    setLoading(false);

    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }

    downloadCSV(result.text, "usage-report.csv");
    setNotice("CSV usage-отчет экспортирован.");
  }

  if (!guard.isReady) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Loading" title="Usage" subtitle="Проверяем доступ к аналитике..." />
        </Card>
      </main>
    );
  }

  if (!caps.showAnalytics || !caps.canViewUsage) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Redirect" title="Usage недоступен" subtitle="Выполняем переход в доступный раздел аналитики." />
        </Card>
      </main>
    );
  }

  return (
    <main>
      <Card>
        <SectionHeader
          badge="Analytics / Usage"
          title="Usage и timeseries"
          subtitle="Токены, стоимость, error/fallback-rate и лидеры потребления."
          actions={
            <div className="row compact">
              <Button variant="secondary" onClick={() => void loadData()} disabled={loading}>
                Обновить
              </Button>
              {caps.canExportCSV ? (
                <Button onClick={() => void exportUsageCSV()} disabled={loading}>
                  Export CSV
                </Button>
              ) : null}
            </div>
          }
        />
        <AnalyticsSectionNav caps={caps} />
        <Notice tone="info">{notice}</Notice>
        <AnalyticsScopeControls
          caps={caps}
          scope={scope}
          projectID={scopeProjectID}
          apiKeyID={scopeAPIKeyID}
          projects={scopeProjects}
          keys={scopeKeys}
          loading={loading}
          onScopeChange={(nextScope) => {
            setScope(nextScope);
            setScopeAPIKeyID("");
            if (nextScope === "org") {
              setScopeProjectID("");
            }
          }}
          onProjectChange={(projectID) => {
            setScopeProjectID(projectID);
            setScopeAPIKeyID("");
          }}
          onKeyChange={(apiKeyID) => setScopeAPIKeyID(apiKeyID)}
        />

        <div className="row">
          {SCALE_OPTIONS.map((option) => (
            <Button
              key={option}
              variant={scale === option ? "primary" : "secondary"}
              onClick={() => setScale(option)}
              disabled={loading}
            >
              {option}
            </Button>
          ))}
        </div>

        <div className="row">
          <Button variant={groupBy === "model" ? "primary" : "secondary"} onClick={() => setGroupBy("model")}>
            Top by model
          </Button>
          <Button variant={groupBy === "project" ? "primary" : "secondary"} onClick={() => setGroupBy("project")}>
            Top by project
          </Button>
        </div>
      </Card>

      <div className="grid two">
        <Card>
          <h2>Tokens over time</h2>
          {mergedSeries.length === 0 ? (
            <EmptyState title="Нет данных" description="Сделайте запросы через gateway и обновите график." />
          ) : (
            <div className="chart-box">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={mergedSeries}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="bucket_start" tickFormatter={formatBucket} minTickGap={24} />
                  <YAxis />
                  <Tooltip labelFormatter={formatBucket} />
                  <Legend />
                  <Line type="monotone" dataKey="input_tokens" stroke="#7aa2ff" dot={false} name="input_tokens" />
                  <Line type="monotone" dataKey="output_tokens" stroke="#5dd39e" dot={false} name="output_tokens" />
                </LineChart>
              </ResponsiveContainer>
            </div>
          )}
        </Card>

        <Card>
          <h2>Cost over time</h2>
          {mergedSeries.length === 0 ? (
            <EmptyState title="Нет данных" description="Сделайте запросы через gateway и обновите график." />
          ) : (
            <div className="chart-box">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={mergedSeries}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="bucket_start" tickFormatter={formatBucket} minTickGap={24} />
                  <YAxis />
                  <Tooltip labelFormatter={formatBucket} />
                  <Legend />
                  <Line type="monotone" dataKey="cost" stroke="#7aa2ff" dot={false} name="cost" />
                </LineChart>
              </ResponsiveContainer>
            </div>
          )}
        </Card>

        <Card>
          <h2>Error rate over time</h2>
          {mergedSeries.length === 0 ? (
            <EmptyState title="Нет данных" description="Сделайте запросы через gateway и обновите график." />
          ) : (
            <div className="chart-box">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={mergedSeries}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="bucket_start" tickFormatter={formatBucket} minTickGap={24} />
                  <YAxis domain={[0, 1]} />
                  <Tooltip labelFormatter={formatBucket} />
                  <Legend />
                  <Line type="monotone" dataKey="error_rate" stroke="#ff7a7a" dot={false} name="error_rate" />
                </LineChart>
              </ResponsiveContainer>
            </div>
          )}
        </Card>

        <Card>
          <h2>Fallback rate over time</h2>
          {mergedSeries.length === 0 ? (
            <EmptyState title="Нет данных" description="Сделайте запросы через gateway и обновите график." />
          ) : (
            <div className="chart-box">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={mergedSeries}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="bucket_start" tickFormatter={formatBucket} minTickGap={24} />
                  <YAxis domain={[0, 1]} />
                  <Tooltip labelFormatter={formatBucket} />
                  <Legend />
                  <Line type="monotone" dataKey="fallback_rate" stroke="#ffbd59" dot={false} name="fallback_rate" />
                </LineChart>
              </ResponsiveContainer>
            </div>
          )}
        </Card>
      </div>

      <Card>
        <h2>Top groups</h2>
        {usageRows.length === 0 ? (
          <EmptyState title="Нет usage-агрегатов" description="Сделайте запросы и обновите данные." />
        ) : (
          <div className="chart-box tall">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={usageRows.slice(0, 12)} layout="vertical" margin={{ left: 24 }}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis type="number" />
                <YAxis type="category" dataKey="group" width={170} />
                <Tooltip />
                <Legend />
                <Bar dataKey="total_tokens" fill="#7aa2ff" name="total_tokens" />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>
    </main>
  );
}
