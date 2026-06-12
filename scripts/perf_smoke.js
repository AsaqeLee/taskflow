import http from "k6/http";
import { check, sleep } from "k6";

const baseUrl = __ENV.BASE_URL || "http://127.0.0.1:8080";
const userId = __ENV.PERF_USER_ID || "u_test_001";
const password = __ENV.PERF_USER_PASSWORD || "creator-pass-123";

export const options = {
  vus: Number(__ENV.VUS || 10),
  duration: __ENV.DURATION || "30s",
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<500"],
  },
};

export function setup() {
  const login = http.post(
    `${baseUrl}/auth/login`,
    JSON.stringify({ id: userId, password }),
    {
      headers: { "Content-Type": "application/json" },
    },
  );

  check(login, {
    "login succeeded": (response) => response.status === 200,
  });

  const body = login.json();
  return {
    accessToken: body.access_token,
  };
}

export default function (data) {
  const headers = {
    Authorization: `Bearer ${data.accessToken}`,
    "Content-Type": "application/json",
  };

  const health = http.get(`${baseUrl}/health`);
  check(health, {
    "health ok": (response) => response.status === 200,
  });

  const list = http.get(`${baseUrl}/tasks`, { headers });
  check(list, {
    "list ok": (response) => response.status === 200,
  });

  sleep(1);
}
