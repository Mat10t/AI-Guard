"use client";

import { SessionProvider } from "./lib/session";
import { AppShell } from "./components/app-shell";

export default function Providers({ children }: { children: React.ReactNode }) {
  return (
    <SessionProvider>
      <AppShell>{children}</AppShell>
    </SessionProvider>
  );
}
