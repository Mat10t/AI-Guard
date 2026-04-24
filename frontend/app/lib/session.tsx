"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { apiRequest, ApiResult, clearToken, readToken, saveToken } from "./api";
import { AccessClaims, decodeAccessToken, isTokenExpired } from "./jwt";

type SessionStatus = "loading" | "anonymous" | "authenticated";

type SessionContextValue = {
  status: SessionStatus;
  token: string;
  claims: AccessClaims | null;
  isAuthenticated: boolean;
  establishSession: (accessToken: string) => void;
  refreshSession: () => Promise<boolean>;
  logout: () => Promise<void>;
  authRequest: (options: {
    path: string;
    method?: string;
    body?: unknown;
  }) => Promise<ApiResult>;
};

const SessionContext = createContext<SessionContextValue | null>(null);

function unauthorizedResult(): ApiResult {
  return {
    ok: false,
    status: 401,
    data: {
      code: "unauthorized",
      message: "missing access token"
    },
    text: "unauthorized"
  };
}

export function SessionProvider({ children }: { children: React.ReactNode }) {
  const [status, setStatus] = useState<SessionStatus>("loading");
  const [token, setToken] = useState("");
  const [claims, setClaims] = useState<AccessClaims | null>(null);
  const tokenRef = useRef("");
  const refreshRef = useRef<Promise<boolean> | null>(null);

  const applySessionToken = useCallback((accessToken: string) => {
    const parsed = decodeAccessToken(accessToken);
    if (!parsed) {
      clearToken();
      tokenRef.current = "";
      setToken("");
      setClaims(null);
      setStatus("anonymous");
      return;
    }

    saveToken(accessToken);
    tokenRef.current = accessToken;
    setToken(accessToken);
    setClaims(parsed);
    setStatus("authenticated");
  }, []);

  const clearSession = useCallback(() => {
    clearToken();
    tokenRef.current = "";
    setToken("");
    setClaims(null);
    setStatus("anonymous");
  }, []);

  const refreshSession = useCallback(async (): Promise<boolean> => {
    if (refreshRef.current) {
      return refreshRef.current;
    }

    const request = (async () => {
      const refreshed = await apiRequest({
        path: "/auth/refresh",
        method: "POST"
      });
      if (!refreshed.ok) {
        return false;
      }

      const accessToken = (refreshed.data?.access_token as string) || "";
      if (!accessToken) {
        return false;
      }

      applySessionToken(accessToken);
      return true;
    })();

    refreshRef.current = request;
    const ok = await request;
    refreshRef.current = null;

    if (!ok) {
      clearSession();
    }
    return ok;
  }, [applySessionToken, clearSession]);

  const logout = useCallback(async () => {
    await apiRequest({
      path: "/auth/logout",
      method: "POST"
    });
    clearSession();
  }, [clearSession]);

  const authRequest = useCallback(
    async (options: { path: string; method?: string; body?: unknown }): Promise<ApiResult> => {
      const current = tokenRef.current;
      if (!current) {
        return unauthorizedResult();
      }

      const first = await apiRequest({
        path: options.path,
        method: options.method,
        body: options.body,
        token: current
      });
      if (first.status !== 401) {
        return first;
      }

      const refreshed = await refreshSession();
      if (!refreshed) {
        return first;
      }

      const retriedToken = tokenRef.current;
      if (!retriedToken) {
        return unauthorizedResult();
      }

      const retried = await apiRequest({
        path: options.path,
        method: options.method,
        body: options.body,
        token: retriedToken
      });
      if (retried.status === 401) {
        clearSession();
      }
      return retried;
    },
    [refreshSession, clearSession]
  );

  useEffect(() => {
    let cancelled = false;

    async function bootstrap() {
      const stored = readToken();
      if (stored) {
        const parsed = decodeAccessToken(stored);
        if (parsed && !isTokenExpired(parsed)) {
          if (cancelled) {
            return;
          }
          tokenRef.current = stored;
          setToken(stored);
          setClaims(parsed);
          setStatus("authenticated");
          return;
        }
      }

      const refreshed = await refreshSession();
      if (cancelled) {
        return;
      }
      if (!refreshed) {
        setStatus("anonymous");
      }
    }

    void bootstrap();
    return () => {
      cancelled = true;
    };
  }, [refreshSession]);

  const value = useMemo<SessionContextValue>(
    () => ({
      status,
      token,
      claims,
      isAuthenticated: status === "authenticated",
      establishSession: applySessionToken,
      refreshSession,
      logout,
      authRequest
    }),
    [status, token, claims, applySessionToken, refreshSession, logout, authRequest]
  );

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>;
}

export function useSession(): SessionContextValue {
  const ctx = useContext(SessionContext);
  if (!ctx) {
    throw new Error("useSession must be used inside SessionProvider");
  }
  return ctx;
}
