export type AccessClaims = {
  user_id: string;
  org_id: string;
  role: string;
  exp?: number;
  iat?: number;
};

function decodeBase64URL(value: string): string {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
  if (typeof globalThis.atob === "function") {
    return globalThis.atob(padded);
  }
  throw new Error("base64 decoder is not available");
}

export function decodeAccessToken(token: string): AccessClaims | null {
  const parts = token.split(".");
  if (parts.length < 2) {
    return null;
  }

  try {
    const json = decodeBase64URL(parts[1]);
    const parsed = JSON.parse(json) as AccessClaims;

    if (!parsed.user_id || !parsed.org_id || !parsed.role) {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

export function isTokenExpired(claims: AccessClaims | null, leewaySeconds = 10): boolean {
  if (!claims?.exp) {
    return false;
  }
  const now = Math.floor(Date.now() / 1000);
  return claims.exp <= now + leewaySeconds;
}

export function hasAnyRole(role: string | undefined, allowed: string[]): boolean {
  if (!role) {
    return false;
  }
  return allowed.includes(role);
}
