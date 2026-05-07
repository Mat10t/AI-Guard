import http from "k6/http";
import { check, sleep } from "k6";

const authURL = __ENV.K6_AUTH_URL || "http://auth-org-service:8081";
const projectURL = __ENV.K6_PROJECT_URL || "http://project-key-service:8082";
const email = __ENV.K6_EMAIL || "";
const password = __ENV.K6_PASSWORD || "";
const p95Threshold = Number(__ENV.K6_P95_THRESHOLD_MS || 3500);

export const options = {
  vus: Number(__ENV.K6_AUTH_VUS || 5),
  duration: __ENV.K6_AUTH_DURATION || "20s",
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: [`p(95)<${p95Threshold}`],
  },
};

if (!email || !password) {
  throw new Error("K6_EMAIL and K6_PASSWORD are required");
}

export default function () {
  const loginRes = http.post(
    `${authURL}/auth/login`,
    JSON.stringify({ email, password }),
    { headers: { "Content-Type": "application/json" } }
  );

  let accessToken = "";
  let loginOk = false;
  try {
    accessToken = loginRes.json("access_token") || "";
    loginOk = loginRes.status === 200 && accessToken.length > 0;
  } catch (_) {
    loginOk = false;
  }

  check(loginRes, {
    "login status is 200": () => loginOk,
  });

  if (accessToken) {
    const listRes = http.get(`${projectURL}/projects`, {
      headers: { Authorization: `Bearer ${accessToken}` },
    });
    check(listRes, {
      "projects status is 200": (r) => r.status === 200,
    });
  }

  sleep(0.2);
}
