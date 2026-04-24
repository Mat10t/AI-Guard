export const TOKEN_STORAGE_KEY = "llm_gateway_access_token";

export type ApiResult = {
  ok: boolean;
  status: number;
  data: any;
  text: string;
};

export async function apiRequest(options: {
  path: string;
  method?: string;
  token?: string;
  body?: unknown;
}): Promise<ApiResult> {
  const { path, method = "GET", token = "", body } = options;
  const headers: Record<string, string> = {};

  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }

  const response = await fetch(`/api/proxy${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    credentials: "include",
    cache: "no-store"
  });

  const text = await response.text();
  const contentType = (response.headers.get("content-type") || "").toLowerCase();

  let data: any = null;
  if (text && contentType.includes("application/json")) {
    try {
      data = JSON.parse(text);
    } catch {
      data = null;
    }
  }

  return { ok: response.ok, status: response.status, data, text };
}

export function readToken(): string {
  if (typeof window === "undefined") {
    return "";
  }
  return localStorage.getItem(TOKEN_STORAGE_KEY) || "";
}

export function saveToken(token: string): void {
  if (typeof window === "undefined") {
    return;
  }
  localStorage.setItem(TOKEN_STORAGE_KEY, token);
}

export function clearToken(): void {
  if (typeof window === "undefined") {
    return;
  }
  localStorage.removeItem(TOKEN_STORAGE_KEY);
}
