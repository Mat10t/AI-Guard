import http from "k6/http";
import { check, sleep } from "k6";

const gatewayURL = __ENV.K6_GATEWAY_URL || "http://api-gateway:8080";
const apiKey = __ENV.K6_API_KEY || "";
const model = __ENV.K6_MODEL || "mock-fast";
const p95Threshold = Number(__ENV.K6_P95_THRESHOLD_MS || 3500);

export const options = {
  vus: Number(__ENV.K6_GATEWAY_VUS || 10),
  duration: __ENV.K6_GATEWAY_DURATION || "30s",
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: [`p(95)<${p95Threshold}`],
  },
};

if (!apiKey) {
  throw new Error("K6_API_KEY is required");
}

export default function () {
  const payload = JSON.stringify({
    model,
    messages: [{ role: "user", content: "Load test ping" }],
  });
  const params = {
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${apiKey}`,
    },
  };

  const res = http.post(`${gatewayURL}/v1/chat/completions`, payload, params);
  check(res, {
    "gateway status is 200": (r) => r.status === 200,
    "gateway response has choices": (r) => {
      try {
        const body = r.json();
        return Array.isArray(body.choices) && body.choices.length > 0;
      } catch (_) {
        return false;
      }
    },
  });
  sleep(0.2);
}
