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

type ProviderModel = {
  id: string;
  provider: string;
  status: string;
  input_cost: number;
  output_cost: number;
};

type SyncSource = "tokens" | "budget";

type LimitDraft = {
  token_limit: number;
  budget_limit_usd: number;
  billing_model: string;
  usd_per_token: number;
  period: string;
  sync_source: SyncSource;
};

const DEFAULT_BILLING_MODEL = "gpt-5.4-mini";

function describeError(result: { data: any; text: string }): string {
  const code = result.data?.code || "request_failed";
  const message = result.data?.message || result.text || "request failed";
  return `${code}: ${message}`;
}

function blendedUSDPerToken(inputCost: number, outputCost: number): number {
  return (inputCost + outputCost) / 2 / 1000;
}

function roundBudgetUSD(value: number): number {
  return Math.round(value * 1e12) / 1e12;
}

function asPositiveInt(value: number, fallback = 1): number {
  if (!Number.isFinite(value) || value <= 0) {
    return fallback;
  }
  return Math.floor(value);
}

function asNonNegativeFloat(value: number, fallback = 0): number {
  if (!Number.isFinite(value) || value < 0) {
    return fallback;
  }
  return value;
}

function modelUSDPerToken(modelID: string, catalog: ProviderModel[]): number {
  const model = catalog.find((item) => item.id === modelID);
  if (!model) {
    return 0;
  }
  return blendedUSDPerToken(Number(model.input_cost || 0), Number(model.output_cost || 0));
}

function defaultDraft(catalog: ProviderModel[]): LimitDraft {
  const defaultModel = catalog.find((item) => item.id === DEFAULT_BILLING_MODEL)?.id || catalog[0]?.id || DEFAULT_BILLING_MODEL;
  const usdPerToken = modelUSDPerToken(defaultModel, catalog);
  const tokenLimit = 2000;
  return {
    token_limit: tokenLimit,
    budget_limit_usd: roundBudgetUSD(tokenLimit * usdPerToken),
    billing_model: defaultModel,
    usd_per_token: usdPerToken,
    period: "day",
    sync_source: "tokens"
  };
}

function recomputeDraft(next: LimitDraft): LimitDraft {
  if (next.sync_source === "budget") {
    if (next.usd_per_token > 0) {
      const tokens = Math.floor(next.budget_limit_usd / next.usd_per_token);
      return {
        ...next,
        token_limit: asPositiveInt(tokens, 1)
      };
    }
    return next;
  }

  if (next.usd_per_token <= 0) {
    return {
      ...next,
      budget_limit_usd: 0
    };
  }

  return {
    ...next,
    budget_limit_usd: roundBudgetUSD(next.token_limit * next.usd_per_token)
  };
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

export default function ProjectsPage() {
  const guard = useAuthGuard();
  const session = useSession();
  const router = useRouter();
  const caps = useMemo(() => capabilitiesForRole(session.claims?.role), [session.claims?.role]);

  const [projects, setProjects] = useState<Project[]>([]);
  const [models, setModels] = useState<ProviderModel[]>([]);
  const [limitDrafts, setLimitDrafts] = useState<Record<string, LimitDraft>>({});
  const [routingDrafts, setRoutingDrafts] = useState<Record<string, string>>({});
  const [deleteDrafts, setDeleteDrafts] = useState<Record<string, string>>({});

  const [createProjectModalOpen, setCreateProjectModalOpen] = useState(false);
  const [createProjectName, setCreateProjectName] = useState("Demo Project");
  const [limitModalProjectID, setLimitModalProjectID] = useState("");
  const [routingModalProjectID, setRoutingModalProjectID] = useState("");
  const [deleteModalProjectID, setDeleteModalProjectID] = useState("");

  const [notice, setNotice] = useState("Управляйте проектами через таблицу и модальные действия.");
  const [loading, setLoading] = useState(false);
  const [pendingDeleteProjectID, setPendingDeleteProjectID] = useState("");

  useEffect(() => {
    if (!guard.isReady) {
      return;
    }
    if (!caps.showProjects) {
      router.replace(firstAllowedRoute(session.claims?.role));
      return;
    }
    void bootstrapPage();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [guard.isReady, caps.showProjects, session.claims?.role]);

  function draftFor(projectID: string): LimitDraft {
    return limitDrafts[projectID] || defaultDraft(models);
  }

  function updateDraft(projectID: string, patch: Partial<LimitDraft>) {
    const current = draftFor(projectID);
    const next = recomputeDraft({ ...current, ...patch });
    setLimitDrafts((prev) => ({
      ...prev,
      [projectID]: next
    }));
  }

  function updateRoutingDraft(projectID: string, value: string) {
    setRoutingDrafts((prev) => ({
      ...prev,
      [projectID]: value
    }));
  }

  function routingDraftFor(projectID: string): string {
    return routingDrafts[projectID] || "";
  }

  function updateDeleteDraft(projectID: string, value: string) {
    setDeleteDrafts((prev) => ({
      ...prev,
      [projectID]: value
    }));
  }

  function deleteDraftFor(projectID: string): string {
    return deleteDrafts[projectID] || "";
  }

  async function bootstrapPage(successNotice = "Проекты, лимиты и fallback-настройки загружены.") {
    setLoading(true);
    const [projectsResult, modelsResult] = await Promise.all([
      session.authRequest({ path: "/projects", method: "GET" }),
      session.authRequest({ path: "/catalog/models", method: "GET" })
    ]);

    if (!projectsResult.ok) {
      setLoading(false);
      setNotice(describeError(projectsResult));
      return;
    }
    if (!modelsResult.ok) {
      setLoading(false);
      setNotice(describeError(modelsResult));
      return;
    }

    const loadedProjects = (projectsResult.data?.items as Project[]) || [];
    const loadedModels = ((modelsResult.data?.items as ProviderModel[]) || []).map((item) => ({
      ...item,
      input_cost: Number(item.input_cost || 0),
      output_cost: Number(item.output_cost || 0)
    }));

    setProjects(loadedProjects);
    setModels(loadedModels);

    const limitEntries: Record<string, LimitDraft> = {};
    const routingEntries: Record<string, string> = {};
    await Promise.all(
      loadedProjects.map(async (project) => {
        limitEntries[project.id] = defaultDraft(loadedModels);
        routingEntries[project.id] = "";

        const [limitResult, routingResult] = await Promise.all([
          session.authRequest({ path: `/limits/projects/${project.id}`, method: "GET" }),
          session.authRequest({ path: `/projects/${project.id}/routing`, method: "GET" })
        ]);

        if (limitResult.ok) {
          const model = String(limitResult.data?.billing_model || "").trim() || DEFAULT_BILLING_MODEL;
          const usdPerToken = Number(limitResult.data?.usd_per_token || modelUSDPerToken(model, loadedModels) || 0);
          const syncSource = (String(limitResult.data?.sync_source || "tokens") === "budget" ? "budget" : "tokens") as SyncSource;

          limitEntries[project.id] = recomputeDraft({
            token_limit: asPositiveInt(Number(limitResult.data?.token_limit || 0), 1),
            budget_limit_usd: asNonNegativeFloat(Number(limitResult.data?.budget_limit_usd || 0), 0),
            billing_model: model,
            usd_per_token: usdPerToken,
            period: String(limitResult.data?.period || "day"),
            sync_source: syncSource
          });
        }

        if (routingResult.ok) {
          routingEntries[project.id] = String(routingResult.data?.fallback_model_id || "").trim();
        }
      })
    );

    setLimitDrafts(limitEntries);
    setRoutingDrafts(routingEntries);
    setLoading(false);
    setNotice(successNotice);
  }

  async function createProject(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!caps.canMutateProjects) {
      return;
    }
    const name = createProjectName.trim();
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

    setCreateProjectModalOpen(false);
    setCreateProjectName("Demo Project");
    await bootstrapPage(`Проект "${name}" создан.`);
  }

  async function setProjectLimit(project: Project) {
    if (!caps.canMutateProjects) {
      return;
    }

    const draft = draftFor(project.id);
    setLoading(true);
    const result = await session.authRequest({
      path: `/limits/projects/${project.id}`,
      method: "PUT",
      body: {
        token_limit: Number(draft.token_limit),
        budget_limit_usd: Number(draft.budget_limit_usd),
        billing_model: draft.billing_model,
        period: draft.period,
        sync_source: draft.sync_source
      }
    });
    setLoading(false);

    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }

    setLimitModalProjectID("");
    await bootstrapPage(`Лимиты проекта "${project.name}" обновлены.`);
  }

  async function saveProjectRouting(project: Project) {
    if (!caps.canMutateProjects) {
      return;
    }

    const fallbackModelID = routingDraftFor(project.id).trim();
    setLoading(true);
    const result = await session.authRequest({
      path: `/projects/${project.id}/routing`,
      method: "PUT",
      body: {
        fallback_model_id: fallbackModelID || null
      }
    });
    setLoading(false);

    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }

    setRoutingModalProjectID("");
    await bootstrapPage(
      fallbackModelID
        ? `Fallback для проекта "${project.name}" сохранен: ${fallbackModelID}.`
        : `Fallback для проекта "${project.name}" сброшен в режим по умолчанию.`
    );
  }

  async function deleteProject(project: Project) {
    if (!caps.canMutateProjects) {
      return;
    }
    const typed = deleteDraftFor(project.id).trim();
    if (typed !== project.name) {
      setNotice(`Для удаления введите точное имя проекта: ${project.name}`);
      return;
    }

    setPendingDeleteProjectID(project.id);
    setLoading(true);
    const result = await session.authRequest({
      path: `/projects/${project.id}`,
      method: "DELETE"
    });
    setLoading(false);
    setPendingDeleteProjectID("");

    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }

    setDeleteModalProjectID("");
    await bootstrapPage(`Проект "${project.name}" удален.`);
  }

  const limitModalProject = useMemo(
    () => projects.find((project) => project.id === limitModalProjectID) || null,
    [projects, limitModalProjectID]
  );

  const routingModalProject = useMemo(
    () => projects.find((project) => project.id === routingModalProjectID) || null,
    [projects, routingModalProjectID]
  );

  const deleteModalProject = useMemo(
    () => projects.find((project) => project.id === deleteModalProjectID) || null,
    [projects, deleteModalProjectID]
  );

  if (!guard.isReady) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Loading" title="Projects" subtitle="Проверяем доступ к защищенному разделу..." />
        </Card>
      </main>
    );
  }

  if (!caps.showProjects) {
    return (
      <main>
        <Card>
          <SectionHeader
            badge="Redirect"
            title="Раздел Projects недоступен"
            subtitle="Для вашей роли этот раздел скрыт. Выполняем переход в доступный раздел."
          />
        </Card>
      </main>
    );
  }

  return (
    <main>
      <Card>
        <SectionHeader
          badge="Projects"
          title="Projects"
          subtitle="Текущие лимиты и fallback видны сразу в списке, управление — через модальные окна."
          actions={
            <div className="row compact">
              <Button variant="secondary" onClick={() => void bootstrapPage()} disabled={loading}>
                Обновить
              </Button>
              {caps.canMutateProjects ? (
                <Button onClick={() => setCreateProjectModalOpen(true)} disabled={loading}>
                  Создать проект
                </Button>
              ) : null}
            </div>
          }
        />
        <Notice tone="info">{notice}</Notice>
      </Card>

      {caps.canMutateProjects ? null : (
        <Card>
          <Notice tone="warning">Роль {session.claims?.role}: изменения проектных настроек недоступны.</Notice>
        </Card>
      )}

      {projects.length === 0 ? <EmptyState title="Проектов нет" description="Создайте первый проект для настройки лимитов и fallback." /> : null}

      {projects.length > 0 ? (
        <Card>
          <h2>Список проектов</h2>
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Project</th>
                  <th>Created</th>
                  <th>Token limit</th>
                  <th>Budget (USD)</th>
                  <th>Period</th>
                  <th>Fallback</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {projects.map((project) => {
                  const draft = draftFor(project.id);
                  const fallbackModel = routingDraftFor(project.id);
                  return (
                    <tr key={project.id}>
                      <td>
                        <strong>{project.name}</strong>
                        <p className="muted mono">{project.id}</p>
                      </td>
                      <td>{formatDate(project.created_at)}</td>
                      <td>{draft.token_limit}</td>
                      <td>${Number(draft.budget_limit_usd || 0).toFixed(6)}</td>
                      <td>{draft.period}</td>
                      <td>{fallbackModel || "по умолчанию"}</td>
                      <td>
                        <div className="table-actions">
                          {caps.canMutateProjects ? (
                            <>
                              <Button variant="secondary" onClick={() => setLimitModalProjectID(project.id)} disabled={loading}>
                                Настроить лимиты
                              </Button>
                              <Button variant="secondary" onClick={() => setRoutingModalProjectID(project.id)} disabled={loading}>
                                Настроить fallback
                              </Button>
                              <Button variant="ghost" onClick={() => setDeleteModalProjectID(project.id)} disabled={loading}>
                                Удалить
                              </Button>
                            </>
                          ) : (
                            "-"
                          )}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </Card>
      ) : null}

      {createProjectModalOpen && caps.canMutateProjects ? (
        <div className="modal-backdrop" role="dialog" aria-modal="true">
          <Card className="modal-card modal-card-compact">
            <div className="modal-head">
              <h2>Создать проект</h2>
              <Button variant="ghost" onClick={() => setCreateProjectModalOpen(false)}>
                ✕
              </Button>
            </div>
            <form onSubmit={createProject}>
              <Field label="Название проекта">
                <input value={createProjectName} onChange={(event) => setCreateProjectName(event.target.value)} required />
              </Field>
              <div className="row">
                <Button type="submit" disabled={loading}>
                  Создать
                </Button>
                <Button type="button" variant="secondary" onClick={() => setCreateProjectModalOpen(false)}>
                  Отмена
                </Button>
              </div>
            </form>
          </Card>
        </div>
      ) : null}

      {limitModalProject ? (
        <div className="modal-backdrop" role="dialog" aria-modal="true">
          <Card className="modal-card">
            <div className="modal-head">
              <h2>Настройка лимитов: {limitModalProject.name}</h2>
              <Button variant="ghost" onClick={() => setLimitModalProjectID("")}>
                ✕
              </Button>
            </div>
            {(() => {
              const draft = draftFor(limitModalProject.id);
              return (
                <>
                  <div className="grid two">
                    <Field label="Billing model">
                      <select
                        value={draft.billing_model}
                        onChange={(event) => {
                          const nextModel = event.target.value;
                          const usd = modelUSDPerToken(nextModel, models);
                          updateDraft(limitModalProject.id, {
                            billing_model: nextModel,
                            usd_per_token: usd
                          });
                        }}
                      >
                        {models.map((model) => (
                          <option key={model.id} value={model.id}>
                            {model.id} ({model.provider}, {model.status})
                          </option>
                        ))}
                      </select>
                    </Field>
                    <Field label="Period">
                      <select value={draft.period} onChange={(event) => updateDraft(limitModalProject.id, { period: event.target.value })}>
                        <option value="day">day</option>
                        <option value="week">week</option>
                        <option value="month">month</option>
                      </select>
                    </Field>
                  </div>

                  <div className="grid two">
                    <Field label="Token limit">
                      <input
                        type="number"
                        min={1}
                        value={draft.token_limit}
                        onChange={(event) =>
                          updateDraft(limitModalProject.id, {
                            token_limit: asPositiveInt(Number(event.target.value), 1),
                            sync_source: "tokens"
                          })
                        }
                      />
                    </Field>
                    <Field label="Budget limit (USD)">
                      <input
                        type="number"
                        step="0.000001"
                        min={0}
                        value={draft.budget_limit_usd}
                        onChange={(event) =>
                          updateDraft(limitModalProject.id, {
                            budget_limit_usd: asNonNegativeFloat(Number(event.target.value), 0),
                            sync_source: "budget"
                          })
                        }
                      />
                    </Field>
                  </div>

                  <p className="muted">USD per token: {draft.usd_per_token.toFixed(12)}</p>
                  {draft.usd_per_token <= 0 ? (
                    <Notice tone="warning">
                      Для выбранной модели цена за токен равна 0. Пересчет из бюджета недоступен, используйте token_limit.
                    </Notice>
                  ) : null}
                  <div className="row">
                    <Button variant="secondary" onClick={() => void setProjectLimit(limitModalProject)} disabled={loading}>
                      Сохранить лимит
                    </Button>
                    <Button variant="secondary" onClick={() => setLimitModalProjectID("")} disabled={loading}>
                      Отмена
                    </Button>
                  </div>
                </>
              );
            })()}
          </Card>
        </div>
      ) : null}

      {routingModalProject ? (
        <div className="modal-backdrop" role="dialog" aria-modal="true">
          <Card className="modal-card modal-card-compact">
            <div className="modal-head">
              <h2>Настройка fallback: {routingModalProject.name}</h2>
              <Button variant="ghost" onClick={() => setRoutingModalProjectID("")}>
                ✕
              </Button>
            </div>
            <Field label="Fallback model">
              <select value={routingDraftFor(routingModalProject.id)} onChange={(event) => updateRoutingDraft(routingModalProject.id, event.target.value)}>
                <option value="">По умолчанию (из routing rules)</option>
                {models.map((model) => (
                  <option key={model.id} value={model.id}>
                    {model.id} ({model.provider}, {model.status})
                  </option>
                ))}
              </select>
            </Field>
            <Notice tone="info">
              Текущее значение: {routingDraftFor(routingModalProject.id) || "по умолчанию (глобальное правило для модели)"}.
            </Notice>
            <div className="row">
              <Button variant="secondary" onClick={() => void saveProjectRouting(routingModalProject)} disabled={loading}>
                Сохранить fallback
              </Button>
              <Button variant="secondary" onClick={() => setRoutingModalProjectID("")} disabled={loading}>
                Отмена
              </Button>
            </div>
          </Card>
        </div>
      ) : null}

      {deleteModalProject ? (
        <div className="modal-backdrop" role="dialog" aria-modal="true">
          <Card className="modal-card modal-card-compact">
            <div className="modal-head">
              <h2>Удаление проекта</h2>
              <Button variant="ghost" onClick={() => setDeleteModalProjectID("")}>
                ✕
              </Button>
            </div>
            <Notice tone="warning">Введите точное имя проекта для подтверждения удаления.</Notice>
            <Field label={`Введите "${deleteModalProject.name}"`}>
              <input
                value={deleteDraftFor(deleteModalProject.id)}
                onChange={(event) => updateDeleteDraft(deleteModalProject.id, event.target.value)}
                placeholder={deleteModalProject.name}
              />
            </Field>
            <div className="row">
              <Button
                variant="ghost"
                onClick={() => void deleteProject(deleteModalProject)}
                disabled={
                  loading ||
                  pendingDeleteProjectID === deleteModalProject.id ||
                  deleteDraftFor(deleteModalProject.id).trim() !== deleteModalProject.name
                }
              >
                {pendingDeleteProjectID === deleteModalProject.id ? "Удаляем..." : "Удалить проект"}
              </Button>
              <Button variant="secondary" onClick={() => setDeleteModalProjectID("")}>
                Отмена
              </Button>
            </div>
          </Card>
        </div>
      ) : null}
    </main>
  );
}
