"use client";

import { FormEvent, Suspense, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Button, Card, EmptyState, Field, Notice, SectionHeader } from "../components/ui";
import { apiRequest } from "../lib/api";
import { capabilitiesForRole, firstAllowedRoute } from "../lib/rbac";
import { useSession } from "../lib/session";

type Member = {
  id: string;
  email: string;
  role: string;
  created_at: string;
};

type Project = {
  id: string;
  name: string;
};

type ProjectMember = {
  user_id: string;
  email: string;
  role: string;
};

type InviteResult = {
  email: string;
  role: string;
  invitation_token: string;
  invitation_link: string;
  project_ids?: string[];
};

type GroupMode = "member" | "project";

function describeError(result: { data: any; text: string }): string {
  const code = result.data?.code || "request_failed";
  const message = result.data?.message || result.text || "request failed";
  return `${code}: ${message}`;
}

function buildInviteLink(invitationToken: string): string {
  const encoded = encodeURIComponent(invitationToken);
  if (typeof window !== "undefined") {
    return `${window.location.origin}/auth?invite_token=${encoded}`;
  }
  return `/auth?invite_token=${encoded}`;
}

function roleNeedsProjects(role: string): boolean {
  return role === "PM" || role === "Dev";
}

function AuthPageContent() {
  const session = useSession();
  const router = useRouter();
  const searchParams = useSearchParams();
  const inviteToken = (searchParams.get("invite_token") || "").trim();

  const [mode, setMode] = useState<"login" | "register">("login");
  const [orgName, setOrgName] = useState("Demo Org");
  const [email, setEmail] = useState("demo@example.local");
  const [password, setPassword] = useState("password123");

  const [acceptPassword, setAcceptPassword] = useState("password123");
  const [acceptMissingToken, setAcceptMissingToken] = useState(false);

  const [members, setMembers] = useState<Member[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [projectMembers, setProjectMembers] = useState<Record<string, ProjectMember[]>>({});
  const [groupMode, setGroupMode] = useState<GroupMode>("member");

  const [inviteModalOpen, setInviteModalOpen] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("teammate@example.local");
  const [inviteRole, setInviteRole] = useState("Dev");
  const [inviteProjectIDs, setInviteProjectIDs] = useState<string[]>([]);
  const [inviteResult, setInviteResult] = useState<InviteResult | null>(null);

  const [developerMode, setDeveloperMode] = useState(false);
  const [manualToken, setManualToken] = useState("");
  const [deletingMemberID, setDeletingMemberID] = useState("");
  const [notice, setNotice] = useState("Войдите в аккаунт, чтобы открыть рабочее пространство организации.");
  const [loading, setLoading] = useState(false);

  const caps = useMemo(() => capabilitiesForRole(session.claims?.role), [session.claims?.role]);
  const canInvite = caps.canInviteMembers;
  const canDeleteMembers = session.claims?.role === "Admin";

  useEffect(() => {
    const m = searchParams.get("mode");
    if (m === "register" || m === "login") {
      setMode(m);
    }
  }, [searchParams]);

  useEffect(() => {
    setManualToken(session.token);
  }, [session.token]);

  useEffect(() => {
    if (session.isAuthenticated && !caps.showOrganization) {
      const target = firstAllowedRoute(session.claims?.role);
      if (target !== "/auth") {
        router.replace(target);
      }
    }
  }, [session.isAuthenticated, session.claims?.role, caps.showOrganization, router]);

  useEffect(() => {
    if (!session.isAuthenticated || !caps.showOrganization) {
      return;
    }
    void loadOrganizationSnapshot();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [session.isAuthenticated, caps.showOrganization]);

  useEffect(() => {
    if (!roleNeedsProjects(inviteRole)) {
      setInviteProjectIDs([]);
    }
  }, [inviteRole]);

  function projectNamesForMember(userID: string): string[] {
    const result: string[] = [];
    for (const project of projects) {
      const assigned = (projectMembers[project.id] || []).some((row) => row.user_id === userID);
      if (assigned) {
        result.push(project.name);
      }
    }
    return result;
  }

  async function submitRegister(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setInviteResult(null);
    setAcceptMissingToken(false);
    setLoading(true);

    const result = await apiRequest({
      path: "/auth/register",
      method: "POST",
      body: { org_name: orgName, email, password }
    });
    setLoading(false);

    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }

    const accessToken = (result.data?.access_token as string) || "";
    if (!accessToken) {
      setNotice("Регистрация выполнена, но access token не получен.");
      return;
    }

    session.establishSession(accessToken);
    router.replace(firstAllowedRoute(result.data?.role as string));
  }

  async function submitLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setInviteResult(null);
    setAcceptMissingToken(false);
    setLoading(true);

    const result = await apiRequest({
      path: "/auth/login",
      method: "POST",
      body: { email, password }
    });
    setLoading(false);

    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }

    const accessToken = (result.data?.access_token as string) || "";
    if (!accessToken) {
      setNotice("Вход выполнен, но access token не получен.");
      return;
    }

    session.establishSession(accessToken);
    router.replace(firstAllowedRoute(result.data?.role as string));
  }

  async function submitAcceptInvite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!inviteToken) {
      setNotice("В ссылке отсутствует invite_token.");
      return;
    }

    setInviteResult(null);
    setAcceptMissingToken(false);
    setLoading(true);

    const result = await apiRequest({
      path: "/org/members/accept",
      method: "POST",
      body: { token: inviteToken, password: acceptPassword }
    });
    setLoading(false);

    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }

    const accessToken = (result.data?.access_token as string) || "";
    if (!accessToken) {
      setAcceptMissingToken(true);
      setNotice("Приглашение принято, но auto-login не выполнен: сервер не вернул access_token.");
      return;
    }

    session.establishSession(accessToken);
    setNotice("Приглашение принято. Сессия создана.");
    router.replace(firstAllowedRoute(result.data?.role as string));
  }

  async function loadMembers() {
    const result = await session.authRequest({
      path: "/org/members",
      method: "GET"
    });
    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }
    const items = ((result.data?.items as any[]) || []).map((row) => ({
      id: String(row?.id || ""),
      email: String(row?.email || ""),
      role: String(row?.role || ""),
      created_at: String(row?.created_at || "")
    }));
    setMembers(items.filter((member) => member.id));
  }

  async function loadProjectMembership() {
    const projectsResult = await session.authRequest({
      path: "/projects",
      method: "GET"
    });
    if (!projectsResult.ok) {
      setProjects([]);
      setProjectMembers({});
      return;
    }

    const loadedProjects = ((projectsResult.data?.items as any[]) || []).map((row) => ({
      id: String(row?.id || ""),
      name: String(row?.name || "")
    }));
    setProjects(loadedProjects.filter((project) => project.id));

    const nextProjectMembers: Record<string, ProjectMember[]> = {};
    await Promise.all(
      loadedProjects.map(async (project) => {
        const membersResult = await session.authRequest({
          path: `/projects/${project.id}/members`,
          method: "GET"
        });
        if (!membersResult.ok) {
          nextProjectMembers[project.id] = [];
          return;
        }
        nextProjectMembers[project.id] = ((membersResult.data?.items as any[]) || []).map((row) => ({
          user_id: String(row?.user_id || ""),
          email: String(row?.email || ""),
          role: String(row?.role || "")
        }));
      })
    );
    setProjectMembers(nextProjectMembers);
  }

  async function loadOrganizationSnapshot(success = "Список участников обновлен.") {
    setLoading(true);
    await Promise.all([loadMembers(), loadProjectMembership()]);
    setLoading(false);
    setNotice(success);
  }

  async function submitInvite(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canInvite) {
      setNotice("Недостаточно прав для приглашения участников.");
      return;
    }
    if (roleNeedsProjects(inviteRole) && inviteProjectIDs.length === 0) {
      setNotice("Для роли PM/Dev нужно выбрать хотя бы один проект.");
      return;
    }

    setLoading(true);
    const result = await session.authRequest({
      path: "/org/members/invite",
      method: "POST",
      body: {
        email: inviteEmail,
        role: inviteRole,
        project_ids: roleNeedsProjects(inviteRole) ? inviteProjectIDs : []
      }
    });
    setLoading(false);
    if (!result.ok) {
      setInviteResult(null);
      setNotice(describeError(result));
      return;
    }

    const invitationToken = String(result.data?.invitation_token || "");
    const invitedEmail = String(result.data?.email || inviteEmail);
    const invitedRole = String(result.data?.role || inviteRole);
    const invitedProjects = ((result.data?.project_ids as string[]) || []).map((value) => String(value));

    setInviteResult({
      email: invitedEmail,
      role: invitedRole,
      invitation_token: invitationToken,
      invitation_link: buildInviteLink(invitationToken),
      project_ids: invitedProjects
    });
    setNotice("Приглашение создано. Передайте invite link пользователю.");
  }

  async function deleteMember(member: Member) {
    if (!canDeleteMembers) {
      setNotice("Недостаточно прав для удаления участника.");
      return;
    }
    if (member.id === session.claims?.user_id) {
      setNotice("Нельзя удалить собственную учетную запись.");
      return;
    }
    setDeletingMemberID(member.id);
    const result = await session.authRequest({
      path: `/org/members/${member.id}`,
      method: "DELETE"
    });
    setDeletingMemberID("");
    if (!result.ok) {
      setNotice(describeError(result));
      return;
    }
    await loadOrganizationSnapshot(`Участник ${member.email} удален.`);
  }

  async function handleRefresh() {
    setLoading(true);
    const ok = await session.refreshSession();
    setLoading(false);
    if (!ok) {
      setNotice("Не удалось обновить сессию. Войдите снова.");
      router.replace("/auth");
      return;
    }
    await loadOrganizationSnapshot("Сессия обновлена, данные организации синхронизированы.");
  }

  async function handleLogout() {
    setLoading(true);
    await session.logout();
    setLoading(false);
    setMembers([]);
    setProjects([]);
    setProjectMembers({});
    setInviteResult(null);
    setNotice("Вы вышли из системы.");
    router.replace("/auth");
  }

  if (session.status === "loading") {
    return (
      <main>
        <Card>
          <SectionHeader badge="Loading" title="Organization Access" subtitle="Проверяем состояние сессии..." />
        </Card>
      </main>
    );
  }

  if (!session.isAuthenticated) {
    if (inviteToken) {
      return (
        <main>
          <Card>
            <SectionHeader
              badge="Invite"
              title="Принятие приглашения"
              subtitle="Установите пароль, чтобы завершить приглашение и автоматически войти в кабинет."
            />
            <Notice tone="info">{notice}</Notice>
          </Card>

          <form onSubmit={submitAcceptInvite}>
            <Card>
              <h2>Accept invite</h2>
              <Field label="Invite token">
                <textarea value={inviteToken} rows={2} readOnly />
              </Field>
              <Field label="Новый пароль">
                <input
                  value={acceptPassword}
                  onChange={(e) => setAcceptPassword(e.target.value)}
                  type="password"
                  minLength={8}
                  required
                />
              </Field>
              <div className="row">
                <Button type="submit" disabled={loading}>
                  Принять приглашение и войти
                </Button>
                <Button type="button" variant="secondary" onClick={() => router.replace("/auth?mode=login")}>
                  Перейти к входу
                </Button>
              </div>

              {acceptMissingToken ? (
                <Notice tone="error">
                  Сервер не вернул `access_token`, auto-login невозможен. Используйте обычный вход.
                </Notice>
              ) : null}
            </Card>
          </form>
        </main>
      );
    }

    return (
      <main>
        <Card>
          <SectionHeader
            badge="Auth"
            title="Вход в рабочее пространство"
            subtitle="Зарегистрируйте организацию или войдите в существующий аккаунт."
          />
          <Notice tone="info">{notice}</Notice>
          <div className="row">
            <Button variant={mode === "login" ? "primary" : "secondary"} onClick={() => setMode("login")}>
              Вход
            </Button>
            <Button variant={mode === "register" ? "primary" : "secondary"} onClick={() => setMode("register")}>
              Регистрация
            </Button>
          </div>
        </Card>

        {mode === "register" ? (
          <form onSubmit={submitRegister}>
            <Card>
              <h2>Регистрация организации</h2>
              <Field label="Organization">
                <input value={orgName} onChange={(e) => setOrgName(e.target.value)} required />
              </Field>
              <Field label="Email">
                <input value={email} onChange={(e) => setEmail(e.target.value)} type="email" required />
              </Field>
              <Field label="Password">
                <input value={password} onChange={(e) => setPassword(e.target.value)} type="password" minLength={8} required />
              </Field>
              <Button type="submit" disabled={loading}>
                Создать организацию
              </Button>
            </Card>
          </form>
        ) : (
          <form onSubmit={submitLogin}>
            <Card>
              <h2>Вход</h2>
              <Field label="Email">
                <input value={email} onChange={(e) => setEmail(e.target.value)} type="email" required />
              </Field>
              <Field label="Password">
                <input value={password} onChange={(e) => setPassword(e.target.value)} type="password" minLength={8} required />
              </Field>
              <Button type="submit" disabled={loading}>
                Войти
              </Button>
            </Card>
          </form>
        )}
      </main>
    );
  }

  if (!caps.showOrganization) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Redirect" title="Переход в доступный раздел" subtitle="Раздел Organization скрыт для вашей роли." />
        </Card>
      </main>
    );
  }

  return (
    <main>
      <Card>
        <SectionHeader
          badge="Organization"
          title="Organization workspace"
          subtitle="Участники, доступы и сессия пользователя."
          actions={
            <div className="row compact">
              <Button variant="secondary" onClick={() => void loadOrganizationSnapshot()} disabled={loading}>
                Обновить
              </Button>
              {canInvite ? (
                <Button variant="secondary" onClick={() => setInviteModalOpen(true)} disabled={loading}>
                  Пригласить участника
                </Button>
              ) : null}
              <Button variant="secondary" onClick={() => void handleRefresh()} disabled={loading}>
                Refresh
              </Button>
              <Button variant="ghost" onClick={() => void handleLogout()} disabled={loading}>
                Logout
              </Button>
            </div>
          }
        />
        <Notice tone="success">Role: {session.claims?.role}. Org: {session.claims?.org_id}</Notice>
        <Notice tone="info">{notice}</Notice>
      </Card>

      <Card>
        <h2>Сотрудники</h2>
        <div className="segmented">
          <Button variant={groupMode === "member" ? "primary" : "ghost"} onClick={() => setGroupMode("member")}>
            By Member
          </Button>
          <Button variant={groupMode === "project" ? "primary" : "ghost"} onClick={() => setGroupMode("project")}>
            By Project
          </Button>
        </div>

        {groupMode === "member" ? (
          members.length === 0 ? (
            <EmptyState title="Список пуст" description="Участники появятся после принятия приглашения." />
          ) : (
            <div className="table-wrap">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>Member</th>
                    <th>Role</th>
                    <th>Projects</th>
                    <th>Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {members.map((member) => {
                    const projectNames = projectNamesForMember(member.id);
                    return (
                      <tr key={member.id}>
                        <td>
                          <strong>{member.email}</strong>
                        </td>
                        <td>{member.role}</td>
                        <td>{projectNames.length > 0 ? projectNames.join(", ") : "-"}</td>
                        <td>
                          {canDeleteMembers && member.id !== session.claims?.user_id ? (
                            <Button
                              variant="ghost"
                              onClick={() => void deleteMember(member)}
                              disabled={loading || deletingMemberID === member.id}
                            >
                              {deletingMemberID === member.id ? "Удаляем..." : "Удалить"}
                            </Button>
                          ) : (
                            "-"
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )
        ) : projects.length === 0 ? (
          <EmptyState title="Нет проектов" description="Для группировки по проектам нужен хотя бы один доступный проект." />
        ) : (
          <div className="stack">
            {projects.map((project) => {
              const rows = projectMembers[project.id] || [];
              return (
                <div className="subcard" key={project.id}>
                  <h3>{project.name}</h3>
                  <p className="muted mono">project_id: {project.id}</p>
                  {rows.length === 0 ? (
                    <p className="muted">Сотрудники на проект не назначены.</p>
                  ) : (
                    <div className="table-wrap">
                      <table className="data-table">
                        <thead>
                          <tr>
                            <th>Member</th>
                            <th>Role</th>
                          </tr>
                        </thead>
                        <tbody>
                          {rows.map((row) => (
                            <tr key={`${project.id}:${row.user_id}`}>
                              <td>{row.email}</td>
                              <td>{row.role}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </Card>

      <Card>
        <h2>Developer mode</h2>
        <div className="row">
          <Button variant="secondary" onClick={() => setDeveloperMode((v) => !v)}>
            {developerMode ? "Скрыть access token" : "Показать access token"}
          </Button>
        </div>
        {developerMode ? (
          <>
            <Field label="JWT access token">
              <textarea rows={4} value={manualToken} onChange={(e) => setManualToken(e.target.value.trim())} />
            </Field>
            <Notice tone="warning">Developer mode включен: используйте токен только для отладки.</Notice>
          </>
        ) : null}
      </Card>

      {inviteModalOpen ? (
        <div className="modal-backdrop" role="dialog" aria-modal="true">
          <Card className="modal-card">
            <div className="modal-head">
              <h2>Приглашение участника</h2>
              <Button variant="ghost" onClick={() => setInviteModalOpen(false)}>
                ✕
              </Button>
            </div>

            <form onSubmit={submitInvite}>
              <Field label="Email">
                <input value={inviteEmail} onChange={(e) => setInviteEmail(e.target.value)} type="email" required />
              </Field>
              <Field label="Role">
                <select value={inviteRole} onChange={(e) => setInviteRole(e.target.value)}>
                  <option value="PM">PM</option>
                  <option value="Dev">Dev</option>
                  <option value="Finance">Finance</option>
                </select>
              </Field>
              {roleNeedsProjects(inviteRole) ? (
                <Field label="Доступные проекты (обязательно)">
                  <select
                    multiple
                    value={inviteProjectIDs}
                    onChange={(event) => {
                      const selected = Array.from(event.currentTarget.selectedOptions).map((option) => option.value);
                      setInviteProjectIDs(selected);
                    }}
                    required
                  >
                    {projects.map((project) => (
                      <option key={project.id} value={project.id}>
                        {project.name}
                      </option>
                    ))}
                  </select>
                </Field>
              ) : null}
              <div className="row">
                <Button type="submit" disabled={loading}>
                  Пригласить
                </Button>
                <Button type="button" variant="secondary" onClick={() => setInviteModalOpen(false)}>
                  Закрыть
                </Button>
              </div>
            </form>

            {inviteResult ? (
              <div className="subcard">
                <Notice tone="success">
                  Приглашение создано для {inviteResult.email} ({inviteResult.role}).
                </Notice>
                {inviteResult.project_ids && inviteResult.project_ids.length > 0 ? (
                  <p className="muted mono">project_ids: {inviteResult.project_ids.join(", ")}</p>
                ) : null}
                <Field label="Invite link">
                  <textarea value={inviteResult.invitation_link} readOnly rows={2} />
                </Field>
                <Field label="Invitation token">
                  <textarea value={inviteResult.invitation_token} readOnly rows={2} />
                </Field>
                <div className="row">
                  <Button
                    variant="secondary"
                    onClick={async () => {
                      try {
                        await navigator.clipboard.writeText(inviteResult.invitation_link);
                        setNotice("Invite link скопирован.");
                      } catch {
                        setNotice("Не удалось скопировать ссылку автоматически.");
                      }
                    }}
                  >
                    Скопировать invite link
                  </Button>
                </div>
              </div>
            ) : null}
          </Card>
        </div>
      ) : null}
    </main>
  );
}

export default function AuthPage() {
  return (
    <Suspense
      fallback={
        <main>
          <Card>
            <SectionHeader badge="Loading" title="Auth" subtitle="Инициализируем параметры маршрута..." />
          </Card>
        </main>
      }
    >
      <AuthPageContent />
    </Suspense>
  );
}
