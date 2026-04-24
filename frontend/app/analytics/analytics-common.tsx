"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Field } from "../components/ui";
import { UICapabilities } from "../lib/rbac";

export type AnalyticsScope = "org" | "project" | "key";

export type ScopeProjectOption = {
  id: string;
  name: string;
};

export type ScopeKeyOption = {
  id: string;
  prefix: string;
  status: string;
};

export function describeError(result: { data: any; text: string }): string {
  const code = result.data?.code || "request_failed";
  const message = result.data?.message || result.text || "request failed";
  return `${code}: ${message}`;
}

export function firstAnalyticsSection(caps: UICapabilities): string {
  if (caps.canViewUsage) {
    return "/analytics/usage";
  }
  if (caps.canViewTechnicalLogs) {
    return "/analytics/logs";
  }
  if (caps.canViewAudit) {
    return "/analytics/audit";
  }
  return "/analytics/providers";
}

export function defaultScopeForCaps(caps: UICapabilities): AnalyticsScope {
  if (caps.canUseOrgScope) {
    return "org";
  }
  return "project";
}

function isActive(pathname: string, target: string): boolean {
  return pathname === target || pathname.startsWith(`${target}/`);
}

export function AnalyticsSectionNav({ caps }: { caps: UICapabilities }) {
  const pathname = usePathname();

  return (
    <div className="row compact analytics-nav">
      {caps.canViewUsage ? (
        <Link className={isActive(pathname, "/analytics/usage") ? "topnav-link active" : "topnav-link"} href="/analytics/usage">
          Usage
        </Link>
      ) : null}
      {caps.canViewTechnicalLogs ? (
        <Link className={isActive(pathname, "/analytics/logs") ? "topnav-link active" : "topnav-link"} href="/analytics/logs">
          Logs
        </Link>
      ) : null}
      {caps.canViewAudit ? (
        <Link className={isActive(pathname, "/analytics/audit") ? "topnav-link active" : "topnav-link"} href="/analytics/audit">
          Audit
        </Link>
      ) : null}
      <Link className={isActive(pathname, "/analytics/providers") ? "topnav-link active" : "topnav-link"} href="/analytics/providers">
        Providers
      </Link>
    </div>
  );
}

export function AnalyticsScopeControls(props: {
  caps: UICapabilities;
  scope: AnalyticsScope;
  projectID: string;
  apiKeyID: string;
  projects: ScopeProjectOption[];
  keys: ScopeKeyOption[];
  loading?: boolean;
  onScopeChange: (scope: AnalyticsScope) => void;
  onProjectChange: (projectID: string) => void;
  onKeyChange: (apiKeyID: string) => void;
}) {
  const {
    caps,
    scope,
    projectID,
    apiKeyID,
    projects,
    keys,
    loading = false,
    onScopeChange,
    onProjectChange,
    onKeyChange
  } = props;

  return (
    <div className="subcard">
      <h3>Scope</h3>
      <div className="grid two">
        <Field label="Уровень аналитики">
          <select
            value={scope}
            onChange={(event) => onScopeChange(event.target.value as AnalyticsScope)}
            disabled={loading}
          >
            {caps.canUseOrgScope ? <option value="org">Organization</option> : null}
            <option value="project">Project</option>
            <option value="key">Key</option>
          </select>
        </Field>
        {scope !== "org" ? (
          <Field label="Project">
            <select
              value={projectID}
              onChange={(event) => onProjectChange(event.target.value)}
              disabled={loading || projects.length === 0}
            >
              <option value="">Выберите проект</option>
              {projects.map((project) => (
                <option key={project.id} value={project.id}>
                  {project.name}
                </option>
              ))}
            </select>
          </Field>
        ) : null}
      </div>
      {scope === "key" ? (
        <Field label="API key">
          <select value={apiKeyID} onChange={(event) => onKeyChange(event.target.value)} disabled={loading || keys.length === 0}>
            <option value="">Выберите ключ</option>
            {keys.map((key) => (
              <option key={key.id} value={key.id}>
                {key.prefix} ({key.status})
              </option>
            ))}
          </select>
        </Field>
      ) : null}
    </div>
  );
}

export function downloadCSV(content: string, filename: string) {
  const blob = new Blob([content], { type: "text/csv" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  link.click();
  URL.revokeObjectURL(url);
}
