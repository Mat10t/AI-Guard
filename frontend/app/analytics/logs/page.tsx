"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
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

type LogItem = {
  request_id: string;
  project_id: string;
  api_key_id: string;
  model: string;
  status: string;
  error_code: string;
  retries: number;
  fallback_used: boolean;
  input_tokens: number;
  output_tokens: number;
  created_at: string;
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

function formatDate(value: string): string {
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) {
    return value;
  }
  return d.toLocaleString("ru-RU");
}

export default function AnalyticsLogsPage() {
  const guard = useAuthGuard();
  const session = useSession();
  const router = useRouter();
  const caps = useMemo(() => capabilitiesForRole(session.claims?.role), [session.claims?.role]);

  const [scope, setScope] = useState<AnalyticsScope>("org");
  const [scopeProjectID, setScopeProjectID] = useState("");
  const [scopeAPIKeyID, setScopeAPIKeyID] = useState("");
  const [scopeProjects, setScopeProjects] = useState<ScopeProject[]>([]);
  const [scopeKeys, setScopeKeys] = useState<ScopeKey[]>([]);
  const [items, setItems] = useState<LogItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [notice, setNotice] = useState("Технические логи gateway: статус, retries, fallback, токены.");

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
    if (!caps.canViewTechnicalLogs) {
      router.replace(firstAnalyticsSection(caps));
      return;
    }
    void loadScopeProjects();
    void loadLogs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [guard.isReady, caps.showAnalytics, caps.canViewTechnicalLogs, scope, scopeProjectID, scopeAPIKeyID]);

  useEffect(() => {
    if (scope !== "key" || !scopeProjectID) {
      setScopeKeys([]);
      setScopeAPIKeyID("");
      return;
    }
    void loadScopeKeys(scopeProjectID);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scope, scopeProjectID]);

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
    query.set("limit", "200");
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

  async function loadLogs() {
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
    const result = await session.authRequest({
      path: `/logs/technical?${scopeQueryParams()}`,
      method: "GET"
    });
    setLoading(false);

    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }
    setItems((result.data?.items as LogItem[]) || []);
    setNotice("Технические логи обновлены.");
  }

  async function exportLogsCSV() {
    if (!caps.canExportCSV) {
      return;
    }

    setLoading(true);
    const result = await session.authRequest({
      path: `/reports/csv/logs?${scopeQueryParams()}`,
      method: "GET"
    });
    setLoading(false);

    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }
    downloadCSV(result.text, "technical-logs-report.csv");
    setNotice("CSV logs-отчет экспортирован.");
  }

  if (!guard.isReady) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Loading" title="Logs" subtitle="Проверяем доступ к аналитике..." />
        </Card>
      </main>
    );
  }

  if (!caps.showAnalytics || !caps.canViewTechnicalLogs) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Redirect" title="Logs недоступен" subtitle="Выполняем переход в доступный раздел аналитики." />
        </Card>
      </main>
    );
  }

  return (
    <main>
      <Card>
        <SectionHeader
          badge="Analytics / Logs"
          title="Technical logs"
          subtitle="Диагностика запросов по проектам и ключам."
          actions={
            <div className="row compact">
              <Button variant="secondary" onClick={() => void loadLogs()} disabled={loading}>
                Обновить
              </Button>
              {caps.canExportCSV ? (
                <Button onClick={() => void exportLogsCSV()} disabled={loading}>
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
      </Card>

      <Card>
        <h2>Логи запросов</h2>
        {items.length === 0 ? (
          <EmptyState title="Нет логов" description="Логи появятся после запросов к `/v1/chat/completions`." />
        ) : (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Model</th>
                  <th>Status</th>
                  <th>Fallback</th>
                  <th>Tokens (in/out)</th>
                  <th>Retries</th>
                  <th>Created</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={item.request_id}>
                    <td>
                      <strong>{item.model}</strong>
                      <p className="muted mono">{item.request_id}</p>
                    </td>
                    <td>
                      {item.status}
                      {item.error_code ? <p className="muted">{item.error_code}</p> : null}
                    </td>
                    <td>{item.fallback_used ? "yes" : "no"}</td>
                    <td>
                      {item.input_tokens} / {item.output_tokens}
                    </td>
                    <td>{item.retries}</td>
                    <td>{formatDate(item.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </main>
  );
}
