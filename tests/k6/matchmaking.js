import http from "k6/http";
import { check, group, sleep } from "k6";

import { getAccessToken, getBaseUrl, authHeaders } from "./lib/auth.js";
import {
  buildQuery,
  countStatus,
  envBool,
  envInt,
  must,
  readJson,
  recordUnexpectedStatus,
  withEndpoint,
} from "./lib/common.js";

const BASE_URL = getBaseUrl();
const THINK_TIME_SECONDS = envInt("SLEEP_MS", 100) / 1000;
const ENABLE_QUICK_MATCH = envBool("ENABLE_QUICK_MATCH", true);

const query = buildQuery({
  game_id: envInt("GAME_ID", 0) || "",
  min_registration_price: envInt("MIN_REGISTRATION_PRICE", 0) || "",
  max_registration_price: envInt("MAX_REGISTRATION_PRICE", 0) || "",
  min_capacity: envInt("MIN_CAPACITY", 0) || "",
  max_capacity: envInt("MAX_CAPACITY", 0) || "",
  is_boost: envBool("IS_BOOST", false) ? "true" : "",
  min_boost_power: envInt("MIN_BOOST_POWER", 0) || "",
  page: envInt("PAGE", 1),
  page_size: envInt("PAGE_SIZE", 10),
});

export const options = {
  scenarios: {
    matchmaking_flow: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "15s", target: 10 },
        { duration: "30s", target: 30 },
        { duration: "30s", target: 60 },
        { duration: "15s", target: 0 },
      ],
      gracefulRampDown: "10s",
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.05"],
    http_req_duration: ["p(95)<2000"],
    checks: ["rate>0.95"],
  },
};

export default function () {
  group("healthz", () => {
    const res = http.get(`${BASE_URL}/api/matchmaking/healthz`, {
      tags: withEndpoint({}, "matchmaking_healthz"),
    });

    countStatus(res, "matchmaking_healthz");

    const ok = check(res, {
      "matchmaking healthz status 200": (r) => r.status === 200,
    });

    must(ok, `Matchmaking healthz failed: status=${res.status}`);
  });

  const accessToken = getAccessToken(BASE_URL);
  const headers = authHeaders(accessToken);

  group("recommendations", () => {
    const res = http.get(`${BASE_URL}/api/matchmaking/rooms/recommendations${query}`, {
      headers,
      tags: withEndpoint({}, "matchmaking_recommendations"),
    });

    countStatus(res, "matchmaking_recommendations");

    const ok = check(res, {
      "recommendations status 200": (r) => r.status === 200,
      "recommendations has items array": (r) => Array.isArray(readJson(r, "items")),
    });

    must(ok, `Recommendations failed: status=${res.status}`);
  });

  if (ENABLE_QUICK_MATCH) {
    group("quick_match", () => {
      const res = http.get(`${BASE_URL}/api/matchmaking/rooms/quick-match${query}`, {
        headers,
        tags: withEndpoint({}, "matchmaking_quick_match"),
      });

      countStatus(res, "matchmaking_quick_match");

      const ok = check(res, {
        "quick match status 200 or 404": (r) => r.status === 200 || r.status === 404,
        "quick match has room payload when 200": (r) => {
          if (r.status !== 200) {
            return true;
          }

          return Number(readJson(r, "room.room_id")) > 0 && Number(readJson(r, "round_id")) > 0;
        },
      });

      if (res.status !== 200 && res.status !== 404) {
        recordUnexpectedStatus("matchmaking_quick_match");
      }

      must(ok, `Quick match failed: status=${res.status}`);
    });
  }

  sleep(THINK_TIME_SECONDS);
}
