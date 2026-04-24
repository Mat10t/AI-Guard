"use client";

import { useEffect, useMemo } from "react";
import { useRouter } from "next/navigation";
import { Card, SectionHeader } from "../components/ui";
import { useAuthGuard } from "../lib/guard";
import { capabilitiesForRole, firstAllowedRoute } from "../lib/rbac";
import { useSession } from "../lib/session";
import { firstAnalyticsSection } from "./analytics-common";

export default function AnalyticsRedirectPage() {
  const guard = useAuthGuard();
  const session = useSession();
  const router = useRouter();
  const caps = useMemo(() => capabilitiesForRole(session.claims?.role), [session.claims?.role]);

  useEffect(() => {
    if (!guard.isReady) {
      return;
    }
    if (!caps.showAnalytics) {
      router.replace(firstAllowedRoute(session.claims?.role));
      return;
    }
    router.replace(firstAnalyticsSection(caps));
  }, [guard.isReady, caps, router, session.claims?.role]);

  return (
    <main>
      <Card>
        <SectionHeader badge="Redirect" title="Analytics" subtitle="Перенаправляем в доступный раздел аналитики." />
      </Card>
    </main>
  );
}
