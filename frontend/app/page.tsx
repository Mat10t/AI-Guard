"use client";

import { useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Card, Notice, SectionHeader } from "./components/ui";
import { useSession } from "./lib/session";

export default function HomePage() {
  const session = useSession();
  const router = useRouter();

  useEffect(() => {
    if (session.status === "authenticated") {
      router.replace("/api-keys");
    }
  }, [session.status, router]);

  if (session.status === "loading") {
    return (
      <main>
        <Card>
          <SectionHeader title="AI Guard" badge="Loading" subtitle="Проверяем состояние сессии..." />
        </Card>
      </main>
    );
  }

  if (session.isAuthenticated) {
    return (
      <main>
        <Card>
          <SectionHeader badge="Redirect" title="AI Guard" subtitle="Открываем раздел API Keys..." />
        </Card>
      </main>
    );
  }

  return (
    <main>
      <Card>
        <SectionHeader
          badge="AI Guard"
          title="Единый LLM Gateway для команды"
          subtitle="Войдите или зарегистрируйтесь, чтобы управлять проектами, ключами, лимитами и аналитикой."
        />
        <div className="row">
          <Link className="btn btn-primary" href="/auth?mode=login">
            Войти
          </Link>
          <Link className="btn btn-secondary" href="/auth?mode=register">
            Зарегистрироваться
          </Link>
        </div>
        <Notice tone="info">После входа стартовая точка кабинета: раздел API Keys.</Notice>
      </Card>
    </main>
  );
}
