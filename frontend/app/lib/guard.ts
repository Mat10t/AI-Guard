"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useSession } from "./session";

export function useAuthGuard() {
  const router = useRouter();
  const session = useSession();

  useEffect(() => {
    if (session.status === "anonymous") {
      router.replace("/auth");
    }
  }, [session.status, router]);

  return {
    isReady: session.status === "authenticated",
    status: session.status
  };
}
