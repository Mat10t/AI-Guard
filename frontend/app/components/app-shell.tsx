"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Button } from "./ui";
import { useSession } from "../lib/session";
import { capabilitiesForRole, firstAllowedRoute } from "../lib/rbac";

function isActive(pathname: string, target: string): boolean {
  if (target === "/") {
    return pathname === "/";
  }
  return pathname === target || pathname.startsWith(`${target}/`);
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const session = useSession();
  const [drawerOpen, setDrawerOpen] = useState(false);
  const caps = capabilitiesForRole(session.claims?.role);
  const homePath = firstAllowedRoute(session.claims?.role);
  const pageTitle = useMemo(() => {
    if (pathname.startsWith("/api-keys")) {
      return "API Keys";
    }
    if (pathname.startsWith("/projects")) {
      return "Projects";
    }
    if (pathname.startsWith("/analytics/usage")) {
      return "Usage";
    }
    if (pathname.startsWith("/analytics/logs")) {
      return "Technical logs";
    }
    if (pathname.startsWith("/analytics/audit")) {
      return "Audit trail";
    }
    if (pathname.startsWith("/analytics/providers")) {
      return "Providers";
    }
    if (pathname.startsWith("/auth")) {
      return "Organization";
    }
    return "AI Guard";
  }, [pathname]);

  useEffect(() => {
    setDrawerOpen(false);
  }, [pathname]);

  async function handleLogout() {
    await session.logout();
    router.replace("/auth");
  }

  return (
    <div className="app-shell-root">
      <aside className={`app-sidebar ${drawerOpen ? "open" : ""}`}>
        <div className="sidebar-head">
          <Link className="brand-link" href={session.isAuthenticated ? homePath : "/"}>
            AI Guard
          </Link>
          <button className="icon-btn mobile-only" onClick={() => setDrawerOpen(false)} aria-label="Close menu">
            ✕
          </button>
        </div>
        <nav className="sidebar-nav">
          {session.isAuthenticated ? <span className="role-chip">{session.claims?.role}</span> : null}
            {caps.showAPIKeys ? (
              <Link className={isActive(pathname, "/api-keys") ? "sidebar-link active" : "sidebar-link"} href="/api-keys">
                API Keys
              </Link>
            ) : null}
            {caps.showProjects ? (
              <Link className={isActive(pathname, "/projects") ? "sidebar-link active" : "sidebar-link"} href="/projects">
                Projects
              </Link>
            ) : null}
            {caps.showAnalytics && caps.canViewUsage ? (
              <Link
                className={isActive(pathname, "/analytics/usage") ? "sidebar-link active" : "sidebar-link"}
                href="/analytics/usage"
              >
                Usage
              </Link>
            ) : null}
            {caps.showAnalytics && caps.canViewTechnicalLogs ? (
              <Link
                className={isActive(pathname, "/analytics/logs") ? "sidebar-link active" : "sidebar-link"}
                href="/analytics/logs"
              >
                Logs
              </Link>
            ) : null}
            {caps.showAnalytics && caps.canViewAudit ? (
              <Link
                className={isActive(pathname, "/analytics/audit") ? "sidebar-link active" : "sidebar-link"}
                href="/analytics/audit"
              >
                Audit
              </Link>
            ) : null}
            {caps.showAnalytics ? (
              <Link
                className={isActive(pathname, "/analytics/providers") ? "sidebar-link active" : "sidebar-link"}
                href="/analytics/providers"
              >
                Providers
              </Link>
            ) : null}
            {caps.showOrganization ? (
              <Link className={isActive(pathname, "/auth") ? "sidebar-link active" : "sidebar-link"} href="/auth">
                Organization
              </Link>
            ) : null}
            {!session.isAuthenticated ? (
              <>
                <Link className={isActive(pathname, "/") ? "sidebar-link active" : "sidebar-link"} href="/">
                  Home
                </Link>
                <Link className={isActive(pathname, "/auth") ? "sidebar-link active" : "sidebar-link"} href="/auth">
                  Login
                </Link>
              </>
            ) : null}
        </nav>
        {session.isAuthenticated ? (
          <div className="sidebar-footer">
            <p className="muted">org: {session.claims?.org_id || "-"}</p>
            <Button variant="secondary" onClick={() => void handleLogout()}>
              Logout
            </Button>
          </div>
        ) : (
          <div className="sidebar-footer">
            <p className="muted">Войдите, чтобы открыть разделы консоли.</p>
          </div>
        )}
      </aside>

      {drawerOpen ? <button className="drawer-overlay" onClick={() => setDrawerOpen(false)} aria-label="Close menu" /> : null}

      <div className="app-content">
        <header className="app-header">
          <button className="icon-btn mobile-only" onClick={() => setDrawerOpen(true)} aria-label="Open menu">
            ☰
          </button>
          <div className="app-header-main">
            <h1 className="app-header-title">{pageTitle}</h1>
            <p className="muted">
              {session.isAuthenticated ? `Role: ${session.claims?.role || "-"}` : "Public access"}
            </p>
          </div>
        </header>
        {children}
      </div>
    </div>
  );
}
