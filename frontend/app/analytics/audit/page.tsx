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

type AuditItem = {
  id: number;
  action: string;
  object_type: string;
  object_id: string;
  actor_user_id: string;
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

export default function AnalyticsAuditPage() {
  const guard = useAuthGuard();
  const session = useSession();
  const router = useRouter();
  const caps = useMemo(() => capabilitiesForRole(session.claims?.role), [session.claims?.role]);

  const [scope, setScope] = useState<AnalyticsScope>("org");
  const [scopeProjectID, setScopeProjectID] = useState("");
  const [scopeAPIKeyID, setScopeAPIKeyID] = useState("");
  const [scopeProjects, setScopeProjects] = useState<ScopeProject[]>([]);
  const [scopeKeys, setScopeKeys] = useState<ScopeKey[]>([]);
  const [items, setItems] = useState<AuditItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [notice, setNotice] = useState("История административных и системных действий в организации.");

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
    if (!caps.canViewAudit) {
      router.replace(firstAnalyticsSection(caps));
      return;
    }
    void loadScopeProjects();
    void loadAudit();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [guard.isReady, caps.showAnalytics, caps.canViewAudit, scope, scopeProjectID, scopeAPIKeyID]);

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

  async function loadAudit() {
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
      path: `/audit?${scopeQueryParams()}`,
      method: "GET"
    });
    setLoading(false);

    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }

    setItems((result.data?.items as AuditItem[]) || []);
    setNotice("Аудит обновлен.");
  }

  async function exportAuditCSV() {
    if (!caps.canExportCSV) {
      return;
    }

    setLoading(true);
    const result = await session.authRequest({ path: `/reports/csv/audit?${scopeQueryParams()}`, method: "GET" });
    setLoading(false);
    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }
    downloadCSV(result.text, "audit-report.csv");
    setNotice("CSV audit-отчет экспортирован.");
  }

  if (!guard.isReady) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Loading" title="Audit" subtitle="Проверяем доступ к аналитике..." />
        </Card>
      </main>
    );
  }

  if (!caps.showAnalytics || !caps.canViewAudit) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Redirect" title="Audit недоступен" subtitle="Выполняем переход в доступный раздел аналитики." />
        </Card>
      </main>
    );
  }

  return (
    <main>
      <Card>
        <SectionHeader
          badge="Analytics / Audit"
          title="Audit trail"
          subtitle="Кто, когда и какие изменения выполнил в рамках организации."
          actions={
            <div className="row compact">
              <Button variant="secondary" onClick={() => void loadAudit()} disabled={loading}>
                Обновить
              </Button>
              {caps.canExportCSV ? (
                <Button onClick={() => void exportAuditCSV()} disabled={loading}>
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
        <h2>События аудита</h2>
        {items.length === 0 ? (
          <EmptyState title="Нет событий" description="Действия появятся после изменений в проектах, ключах, лимитах и участниках." />
        ) : (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Action</th>
                  <th>Object</th>
                  <th>Actor</th>
                  <th>Time</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item) => (
                  <tr key={`${item.id}-${item.action}`}>
                    <td>
                      <strong>{item.action}</strong>
                    </td>
                    <td className="mono">
                      {item.object_type}/{item.object_id}
                    </td>
                    <td>{item.actor_user_id || "-"}</td>
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
