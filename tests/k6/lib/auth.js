import http from "k6/http";
import { check, fail } from "k6";

import {
  bearer,
  countStatus,
  env,
  jsonHeaders,
  readJson,
  trimTrailingSlash,
  withEndpoint,
} from "./common.js";

export function getBaseUrl() {
  return trimTrailingSlash(env("BASE_URL", "http://localhost"));
}

export function getAccessToken(baseUrl) {
  const providedToken = env("ACCESS_TOKEN", "");
  if (providedToken !== "") {
    return providedToken;
  }

  const login = env("LOGIN", "");
  const password = env("PASSWORD", "");
  if (login === "" || password === "") {
    fail("Set ACCESS_TOKEN or provide LOGIN and PASSWORD for SSO login");
  }

  const res = http.post(
    `${trimTrailingSlash(baseUrl)}/api/auth/login`,
    JSON.stringify({ login, password }),
    {
      headers: jsonHeaders(),
      tags: withEndpoint({}, "auth_login"),
    },
  );

  countStatus(res, "auth_login");

  const ok = check(res, {
    "auth login status 200": (r) => r.status === 200,
    "auth login has access_token": (r) => {
      const token = readJson(r, "access_token");
      return typeof token === "string" && token.length > 0;
    },
  });

  if (!ok) {
    fail(`SSO login failed: status=${res.status}`);
  }

  return readJson(res, "access_token");
}

export function authHeaders(token, extra = {}) {
  return jsonHeaders({ ...bearer(token), ...extra });
}
