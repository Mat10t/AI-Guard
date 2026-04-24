import { NextRequest, NextResponse } from "next/server";

export const dynamic = "force-dynamic";

const SERVICE_BASES = {
  gateway: process.env.GATEWAY_SERVICE_URL || "http://localhost:8080",
  auth: process.env.AUTH_SERVICE_URL || "http://localhost:8081",
  project: process.env.PROJECT_SERVICE_URL || "http://localhost:8082",
  limits: process.env.LIMITS_SERVICE_URL || "http://localhost:8083",
  catalog: process.env.CATALOG_SERVICE_URL || "http://localhost:8084",
  analytics: process.env.ANALYTICS_SERVICE_URL || "http://localhost:8085"
};

function resolveService(first: string): string {
  switch (first) {
    case "auth":
    case "org":
      return SERVICE_BASES.auth;
    case "projects":
      return SERVICE_BASES.project;
    case "limits":
      return SERVICE_BASES.limits;
    case "catalog":
      return SERVICE_BASES.catalog;
    case "audit":
    case "logs":
    case "analytics":
    case "reports":
      return SERVICE_BASES.analytics;
    case "v1":
    case "healthz":
      return SERVICE_BASES.gateway;
    default:
      return "";
  }
}

async function proxy(request: NextRequest, path: string[]): Promise<NextResponse> {
  if (path.length === 0) {
    return NextResponse.json(
      { code: "validation_error", message: "missing path" },
      { status: 400 }
    );
  }

  const base = resolveService(path[0]);
  if (!base) {
    return NextResponse.json(
      { code: "not_found", message: "unknown service path" },
      { status: 404 }
    );
  }

  const upstreamURL = new URL(base);
  upstreamURL.pathname = `/${path.join("/")}`;
  upstreamURL.search = request.nextUrl.search;

  const headers = new Headers();
  const contentType = request.headers.get("content-type");
  const authorization = request.headers.get("authorization");
  const cookie = request.headers.get("cookie");

  if (contentType) {
    headers.set("content-type", contentType);
  }
  if (authorization) {
    headers.set("authorization", authorization);
  }
  if (cookie) {
    headers.set("cookie", cookie);
  }

  const method = request.method.toUpperCase();
  const hasBody = method !== "GET" && method !== "HEAD";

  let body: ArrayBuffer | undefined;
  if (hasBody) {
    body = await request.arrayBuffer();
  }

  let upstream: Response;
  try {
    upstream = await fetch(upstreamURL.toString(), {
      method,
      headers,
      body,
      cache: "no-store",
      redirect: "manual"
    });
  } catch {
    return NextResponse.json(
      { code: "upstream_unavailable", message: "cannot reach upstream service" },
      { status: 502 }
    );
  }

  const responseHeaders = new Headers();
  const upstreamContentType = upstream.headers.get("content-type");
  const upstreamSetCookie = upstream.headers.get("set-cookie");
  const upstreamDisposition = upstream.headers.get("content-disposition");

  if (upstreamContentType) {
    responseHeaders.set("content-type", upstreamContentType);
  }
  if (upstreamSetCookie) {
    responseHeaders.set("set-cookie", upstreamSetCookie);
  }
  if (upstreamDisposition) {
    responseHeaders.set("content-disposition", upstreamDisposition);
  }

  const raw = await upstream.arrayBuffer();
  return new NextResponse(raw, {
    status: upstream.status,
    headers: responseHeaders
  });
}

export async function GET(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
): Promise<NextResponse> {
  const { path } = await context.params;
  return proxy(request, path);
}

export async function POST(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
): Promise<NextResponse> {
  const { path } = await context.params;
  return proxy(request, path);
}

export async function PUT(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
): Promise<NextResponse> {
  const { path } = await context.params;
  return proxy(request, path);
}

export async function DELETE(
  request: NextRequest,
  context: { params: Promise<{ path: string[] }> }
): Promise<NextResponse> {
  const { path } = await context.params;
  return proxy(request, path);
}
