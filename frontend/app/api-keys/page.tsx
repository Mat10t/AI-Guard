"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Button, Card, EmptyState, Field, Notice, SectionHeader } from "../components/ui";
import { useAuthGuard } from "../lib/guard";
import { capabilitiesForRole, firstAllowedRoute } from "../lib/rbac";
import { useSession } from "../lib/session";

type Project = {
  id: string;
  name: string;
  created_at: string;
};

type KeyInfo = {
  id: string;
  name: string;
  status: string;
  prefix: string;
  api_key?: string;
  created_at?: string;
  revoked_at?: string;
  project_id: string;
  project_name: string;
};

type GroupMode = "key" | "project";
const CREATE_PROJECT_OPTION = "__create_new_project__";

function describeError(result: { data: any; text: string }): string {
  const code = result.data?.code || "request_failed";
  const message = result.data?.message || result.text || "request failed";
  return `${code}: ${message}`;
}

function maskAPIKey(apiKey?: string, prefix?: string): string {
  const full = String(apiKey || "").trim();
  if (full) {
    return `${full.slice(0, 6)}...${full.slice(-4)}`;
  }
  const short = String(prefix || "").trim();
  if (short) {
    return `${short}...`;
  }
  return "-";
}

function formatDate(value?: string): string {
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

export default function APIKeysPage() {
  const guard = useAuthGuard();
  const session = useSession();
  const router = useRouter();
  const caps = useMemo(() => capabilitiesForRole(session.claims?.role), [session.claims?.role]);

  const [projects, setProjects] = useState<Project[]>([]);
  const [keysByProject, setKeysByProject] = useState<Record<string, KeyInfo[]>>({});
  const [keySecrets, setKeySecrets] = useState<Record<string, string>>({});
  const [groupMode, setGroupMode] = useState<GroupMode>("key");
  const [notice, setNotice] = useState("Управление ключами интеграции по проектам.");
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [projectCreateModalOpen, setProjectCreateModalOpen] = useState(false);
  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyProjectID, setNewKeyProjectID] = useState("");
  const [newProjectName, setNewProjectName] = useState("");
  const [pendingRevokeKeyID, setPendingRevokeKeyID] = useState("");

  useEffect(() => {
    if (!guard.isReady) {
      return;
    }
    if (!caps.showAPIKeys) {
      router.replace(firstAllowedRoute(session.claims?.role));
      return;
    }
    void bootstrapPage();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [guard.isReady, caps.showAPIKeys, session.claims?.role]);

  const flatKeys = useMemo(() => {
    return Object.values(keysByProject)
      .flat()
      .sort((a, b) => String(b.created_at || "").localeCompare(String(a.created_at || "")));
  }, [keysByProject]);

  function resetCreateForm(projectID = "") {
    setNewKeyName("");
    setNewKeyProjectID(projectID);
    setNewProjectName("");
  }

  function closeCreateKeyModal() {
    setProjectCreateModalOpen(false);
    setModalOpen(false);
  }

  async function bootstrapPage(success = "Список проектов и API-ключей обновлен.") {
    setLoading(true);
    const projectsResult = await session.authRequest({ path: "/projects", method: "GET" });
    if (!projectsResult.ok) {
      setLoading(false);
      setNotice(describeError(projectsResult));
      return;
    }

    const loadedProjects = ((projectsResult.data?.items as Project[]) || []).map((item) => ({
      id: String(item.id || ""),
      name: String(item.name || ""),
      created_at: String(item.created_at || "")
    }));
    setProjects(loadedProjects);
    if (!newKeyProjectID && loadedProjects[0]?.id) {
      setNewKeyProjectID(loadedProjects[0].id);
    }

    const nextKeysByProject: Record<string, KeyInfo[]> = {};
    const nextSecrets: Record<string, string> = {};
    await Promise.all(
      loadedProjects.map(async (project) => {
        const keysResult = await session.authRequest({
          path: `/projects/${project.id}/keys`,
          method: "GET"
        });
        if (!keysResult.ok) {
          nextKeysByProject[project.id] = [];
          return;
        }
        const activeRows = ((keysResult.data?.items as any[]) || [])
          .map((row) => ({
            id: String(row?.id || ""),
            name: String(row?.name || ""),
            status: String(row?.status || ""),
            prefix: String(row?.prefix || ""),
            api_key: String(row?.api_key || "").trim(),
            created_at: String(row?.created_at || ""),
            revoked_at: String(row?.revoked_at || ""),
            project_id: project.id,
            project_name: project.name
          }))
          .filter((row) => row.id && row.status === "active");
        for (const row of activeRows) {
          if (row.api_key) {
            nextSecrets[row.id] = row.api_key;
          }
        }
        nextKeysByProject[project.id] = activeRows;
      })
    );

    setKeysByProject(nextKeysByProject);
    setKeySecrets(nextSecrets);
    setLoading(false);
    setNotice(success);
  }

  async function createProjectFromModal(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!caps.canMutateProjects) {
      return;
    }
    const name = newProjectName.trim();
    if (!name) {
      setNotice("Введите название проекта.");
      return;
    }

    setLoading(true);
    const result = await session.authRequest({
      path: "/projects",
      method: "POST",
      body: { name }
    });
    setLoading(false);
    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }

    const createdProjectID = String(result.data?.id || "").trim();
    await bootstrapPage(`Проект "${name}" создан.`);
    setNewProjectName("");
    if (createdProjectID) {
      setNewKeyProjectID(createdProjectID);
    }
    setProjectCreateModalOpen(false);
  }

  async function createKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!caps.canMutateProjects) {
      return;
    }
    if (!newKeyProjectID) {
      setNotice("Выберите проект для выпуска ключа.");
      return;
    }

    setLoading(true);
    const result = await session.authRequest({
      path: `/projects/${newKeyProjectID}/keys`,
      method: "POST",
      body: { name: newKeyName.trim() }
    });
    setLoading(false);
    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }

    const keyID = String(result.data?.id || "").trim();
    const secret = String(result.data?.api_key || "").trim();
    if (keyID && secret) {
      setKeySecrets((prev) => ({ ...prev, [keyID]: secret }));
    }

    await bootstrapPage("API-ключ создан.");
    closeCreateKeyModal();
    resetCreateForm(newKeyProjectID);
  }

  async function revokeKey(projectID: string, keyID: string) {
    if (!caps.canMutateProjects) {
      return;
    }
    setPendingRevokeKeyID(keyID);
    const result = await session.authRequest({
      path: `/projects/${projectID}/keys/${keyID}/revoke`,
      method: "POST"
    });
    setPendingRevokeKeyID("");
    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }
    await bootstrapPage("Ключ отозван.");
  }

  async function copyKey(keyID: string) {
    const value = keySecrets[keyID] || "";
    if (!value) {
      setNotice("Полный ключ недоступен для копирования.");
      return;
    }
    try {
      await navigator.clipboard.writeText(value);
      setNotice("API-ключ скопирован.");
    } catch {
      setNotice("Не удалось скопировать ключ автоматически.");
    }
  }

  function openUsage(projectID: string, keyID: string) {
    const q = new URLSearchParams();
    q.set("scope", "key");
    q.set("project_id", projectID);
    q.set("api_key_id", keyID);
    router.push(`/analytics/usage?${q.toString()}`);
  }

  if (!guard.isReady) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Loading" title="API Keys" subtitle="Проверяем доступ к защищенному разделу..." />
        </Card>
      </main>
    );
  }

  if (!caps.showAPIKeys) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Redirect" title="Раздел API Keys недоступен" subtitle="Выполняем переход в доступный раздел." />
        </Card>
      </main>
    );
  }

  return (
    <main>
      <Card>
        <SectionHeader
          badge="API Keys"
          title="API Keys"
          subtitle="Выпуск, отзыв и обзор ключей по проектам."
          actions={
            <div className="row compact">
              <div className="segmented">
                <Button variant={groupMode === "key" ? "primary" : "ghost"} onClick={() => setGroupMode("key")}>
                  By API key
                </Button>
                <Button variant={groupMode === "project" ? "primary" : "ghost"} onClick={() => setGroupMode("project")}>
                  By Project
                </Button>
              </div>
              <Button variant="secondary" onClick={() => void bootstrapPage()} disabled={loading}>
                Обновить
              </Button>
              {caps.canMutateProjects ? (
                <Button
                  onClick={() => {
                    resetCreateForm(projects[0]?.id || "");
                    setModalOpen(true);
                  }}
                  disabled={loading}
                >
                  Create API key
                </Button>
              ) : null}
            </div>
          }
        />
        <Notice tone="info">{notice}</Notice>
      </Card>

      {projects.length === 0 ? <EmptyState title="Нет проектов" description="Создайте проект, чтобы выпустить API-ключ." /> : null}

      {groupMode === "key" ? (
        <Card>
          <h2>Keys</h2>
          {flatKeys.length === 0 ? (
            <EmptyState title="Ключей нет" description="Создайте первый ключ через кнопку Create API key." />
          ) : (
            <div className="table-wrap">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Key</th>
                    <th>Project</th>
                    <th>Status</th>
                    <th>Created</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {flatKeys.map((key) => (
                    <tr key={key.id}>
                      <td>
                        <strong>{key.name || "Unnamed key"}</strong>
                        <p className="muted mono">{maskAPIKey(keySecrets[key.id], key.prefix)}</p>
                      </td>
                      <td>{key.project_name}</td>
                      <td>{key.status}</td>
                      <td>{formatDate(key.created_at)}</td>
                      <td>
                        <div className="table-actions">
                          <Button variant="secondary" onClick={() => void copyKey(key.id)}>
                            Copy
                          </Button>
                          <Button variant="secondary" onClick={() => openUsage(key.project_id, key.id)}>
                            Open usage
                          </Button>
                          {caps.canMutateProjects && key.status === "active" ? (
                            <Button
                              variant="ghost"
                              onClick={() => void revokeKey(key.project_id, key.id)}
                              disabled={pendingRevokeKeyID === key.id}
                            >
                              {pendingRevokeKeyID === key.id ? "Revoking..." : "Revoke"}
                            </Button>
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Card>
      ) : (
        <div className="stack">
          {projects.map((project) => {
            const projectKeys = keysByProject[project.id] || [];
            return (
              <Card key={project.id}>
                <h2>{project.name}</h2>
                <p className="muted mono">project_id: {project.id}</p>
                {projectKeys.length === 0 ? (
                  <EmptyState title="Ключей нет" description="В этом проекте пока нет ключей." />
                ) : (
                  <div className="table-wrap">
                    <table className="data-table">
                      <thead>
                        <tr>
                          <th>Key</th>
                          <th>Status</th>
                          <th>Created</th>
                          <th>Actions</th>
                        </tr>
                      </thead>
                      <tbody>
                        {projectKeys.map((key) => (
                          <tr key={key.id}>
                            <td>
                              <strong>{key.name || "Unnamed key"}</strong>
                              <p className="muted mono">{maskAPIKey(keySecrets[key.id], key.prefix)}</p>
                            </td>
                            <td>{key.status}</td>
                            <td>{formatDate(key.created_at)}</td>
                            <td>
                              <div className="table-actions">
                                <Button variant="secondary" onClick={() => void copyKey(key.id)}>
                                  Copy
                                </Button>
                                <Button variant="secondary" onClick={() => openUsage(project.id, key.id)}>
                                  Open usage
                                </Button>
                                {caps.canMutateProjects && key.status === "active" ? (
                                  <Button
                                    variant="ghost"
                                    onClick={() => void revokeKey(project.id, key.id)}
                                    disabled={pendingRevokeKeyID === key.id}
                                  >
                                    {pendingRevokeKeyID === key.id ? "Revoking..." : "Revoke"}
                                  </Button>
                                ) : null}
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </Card>
            );
          })}
        </div>
      )}

      {modalOpen ? (
        <div className="modal-backdrop" role="dialog" aria-modal="true">
          <Card className="modal-card">
            <div className="modal-head">
              <h2>Create a new key</h2>
              <Button variant="ghost" onClick={closeCreateKeyModal}>
                ✕
              </Button>
            </div>
            <form onSubmit={createKey}>
              <Field label="Name your key">
                <input value={newKeyName} onChange={(event) => setNewKeyName(event.target.value)} placeholder="Gateway key" />
              </Field>
              <Field label="Choose project">
                <select
                  value={newKeyProjectID}
                  onChange={(event) => {
                    const value = event.target.value;
                    if (value === CREATE_PROJECT_OPTION) {
                      setProjectCreateModalOpen(true);
                      return;
                    }
                    setNewKeyProjectID(value);
                  }}
                  required
                >
                  <option value="">Выберите проект</option>
                  {projects.map((project) => (
                    <option key={project.id} value={project.id}>
                      {project.name}
                    </option>
                  ))}
                  {caps.canMutateProjects ? <option value={CREATE_PROJECT_OPTION}>+ Создать новый проект</option> : null}
                </select>
              </Field>

              <div className="row">
                <Button type="submit" disabled={loading}>
                  Create key
                </Button>
                <Button type="button" variant="ghost" onClick={closeCreateKeyModal}>
                  Cancel
                </Button>
              </div>
            </form>
          </Card>
        </div>
      ) : null}

      {modalOpen && projectCreateModalOpen ? (
        <div className="modal-backdrop modal-backdrop-top" role="dialog" aria-modal="true">
          <Card className="modal-card modal-card-compact">
            <div className="modal-head">
              <h2>Создать проект</h2>
              <Button variant="ghost" onClick={() => setProjectCreateModalOpen(false)}>
                ✕
              </Button>
            </div>
            <form onSubmit={createProjectFromModal}>
              <Field label="Название проекта">
                <input
                  value={newProjectName}
                  onChange={(event) => setNewProjectName(event.target.value)}
                  placeholder="New project"
                  required
                />
              </Field>
              <div className="row">
                <Button type="submit" disabled={loading}>
                  Создать проект
                </Button>
                <Button type="button" variant="secondary" onClick={() => setProjectCreateModalOpen(false)} disabled={loading}>
                  Отмена
                </Button>
              </div>
            </form>
          </Card>
        </div>
      ) : null}
    </main>
  );
}
