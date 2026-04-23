import { fail } from "k6";
import { Counter } from "k6/metrics";

export const http401 = new Counter("http_401");
export const http5xx = new Counter("http_5xx");
export const unexpectedStatus = new Counter("unexpected_status");

export const STRICT = envBool("STRICT", false);

export function env(name, fallback = "") {
  const value = __ENV[name];
  return value === undefined || value === "" ? fallback : value;
}

export function envInt(name, fallback) {
  const raw = env(name, "");
  if (raw === "") {
    return fallback;
  }

  const parsed = Number.parseInt(raw, 10);
  return Number.isNaN(parsed) ? fallback : parsed;
}

export function envBool(name, fallback = false) {
  const raw = env(name, "");
  if (raw === "") {
    return fallback;
  }

  return ["1", "true", "yes", "on"].includes(String(raw).toLowerCase());
}

export function jsonHeaders(extra = {}) {
  return { "Content-Type": "application/json", ...extra };
}

export function bearer(token) {
  return { Authorization: `Bearer ${token}` };
}

export function withEndpoint(tags = {}, endpoint = "") {
  return endpoint === "" ? tags : { ...tags, endpoint };
}

export function countStatus(res, endpoint) {
  if (res.status === 401) {
    http401.add(1, { endpoint });
  }

  if (res.status >= 500) {
    http5xx.add(1, { endpoint });
  }
}

export function must(ok, message) {
  if (!ok && STRICT) {
    fail(message);
  }
}

export function recordUnexpectedStatus(endpoint) {
  unexpectedStatus.add(1, { endpoint });
}

export function buildQuery(params) {
  const parts = [];

  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") {
      continue;
    }

    parts.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`);
  }

  const rendered = parts.join("&");
  return rendered === "" ? "" : `?${rendered}`;
}

export function trimTrailingSlash(value) {
  return value.replace(/\/+$/, "");
}

export function readJson(res, selector = undefined) {
  try {
    return selector === undefined ? res.json() : res.json(selector);
  } catch {
    return undefined;
  }
}
