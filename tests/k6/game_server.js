import http from "k6/http";
import { check, group, sleep } from "k6";

import { authHeaders, getAccessToken, getBaseUrl } from "./lib/auth.js";
import {
  buildQuery,
  countStatus,
  env,
  envBool,
  envInt,
  must,
  readJson,
  recordUnexpectedStatus,
  withEndpoint,
} from "./lib/common.js";

const BASE_URL = getBaseUrl();
const THINK_TIME_SECONDS = envInt("SLEEP_MS", 100) / 1000;
const LEAVE_AFTER = envBool("LEAVE_AFTER", true);

const quickMatchQuery = buildQuery({
  game_id: envInt("GAME_ID", 0) || "",
  min_registration_price: envInt("MIN_REGISTRATION_PRICE", 0) || "",
  max_registration_price: envInt("MAX_REGISTRATION_PRICE", 0) || "",
  min_capacity: envInt("MIN_CAPACITY", 0) || "",
  max_capacity: envInt("MAX_CAPACITY", 0) || "",
  is_boost: envBool("IS_BOOST", false) ? "true" : "",
  min_boost_power: envInt("MIN_BOOST_POWER", 0) || "",
});

export const options = {
  scenarios: {
    game_server_flow: {
      executor: "ramping-vus",
      startVUs: 0,
      stages: [
        { duration: "15s", target: 10 },
        { duration: "30s", target: 25 },
        { duration: "30s", target: 50 },
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

function resolveRoomContext(headers) {
  const explicitRoomID = env("ROOM_ID", "");
  const explicitRoundID = env("ROUND_ID", "");
  if (explicitRoomID !== "") {
    return {
      roomID: explicitRoomID,
      roundID: explicitRoundID,
      joinedViaQuickMatch: false,
    };
  }

  const res = http.get(`${BASE_URL}/api/matchmaking/rooms/quick-match${quickMatchQuery}`, {
    headers,
    tags: withEndpoint({}, "game_server_quick_match"),
  });

  countStatus(res, "game_server_quick_match");

  const ok = check(res, {
    "game quick match status 200 or 404": (r) => r.status === 200 || r.status === 404,
    "game quick match has room payload when 200": (r) => {
      if (r.status !== 200) {
        return true;
      }

      return Number(readJson(r, "room.room_id")) > 0 && Number(readJson(r, "round_id")) > 0;
    },
  });

  if (res.status !== 200 && res.status !== 404) {
    recordUnexpectedStatus("game_server_quick_match");
  }

  must(ok, `Game quick match failed: status=${res.status}`);

  if (res.status !== 200) {
    return {
      roomID: "",
      roundID: "",
      joinedViaQuickMatch: false,
    };
  }

  return {
    roomID: String(readJson(res, "room.room_id")),
    roundID: String(readJson(res, "round_id")),
    joinedViaQuickMatch: true,
  };
}

export default function () {
  group("healthz", () => {
    const res = http.get(`${BASE_URL}/api/game/healthz`, {
      tags: withEndpoint({}, "game_server_healthz"),
    });

    countStatus(res, "game_server_healthz");

    const ok = check(res, {
      "game server healthz status 200": (r) => r.status === 200,
    });

    must(ok, `Game server healthz failed: status=${res.status}`);
  });

  const accessToken = getAccessToken(BASE_URL);
  const headers = authHeaders(accessToken);
  const context = resolveRoomContext(headers);

  if (context.roomID === "") {
    sleep(THINK_TIME_SECONDS);
    return;
  }

  let roundID = context.roundID;

  group("room_state", () => {
    const roomPath = `${BASE_URL}/api/game/rooms/${context.roomID}`;
    const res = http.get(roomPath, {
      headers,
      tags: withEndpoint({}, "game_server_room_state"),
    });

    countStatus(res, "game_server_room_state");

    const ok = check(res, {
      "room state status 200": (r) => r.status === 200,
      "room state room_id matches": (r) => Number(readJson(r, "room_id")) === Number(context.roomID),
    });

    must(ok, `Room state failed: status=${res.status}`);

    if (roundID === "" && res.status === 200) {
      const derivedRoundID = Number(readJson(res, "round_id"));
      if (derivedRoundID > 0) {
        roundID = String(derivedRoundID);
      }
    }
  });

  if (roundID !== "") {
    group("round_status", () => {
      const roundPath = `${BASE_URL}/api/game/rooms/${context.roomID}/rounds/${roundID}`;
      const res = http.get(roundPath, {
        headers,
        tags: withEndpoint({}, "game_server_round_status"),
      });

      countStatus(res, "game_server_round_status");

      const ok = check(res, {
        "round status 200": (r) => r.status === 200,
        "round status round_id matches": (r) => Number(readJson(r, "round_id")) === Number(roundID),
      });

      must(ok, `Round status failed: status=${res.status}`);
    });
  }

  if (context.joinedViaQuickMatch && LEAVE_AFTER) {
    group("leave_room", () => {
      const res = http.post(`${BASE_URL}/api/game/rooms/${context.roomID}/leave`, null, {
        headers,
        tags: withEndpoint({}, "game_server_leave_room"),
      });

      countStatus(res, "game_server_leave_room");

      const ok = check(res, {
        "leave room status 200, 404 or 409": (r) => r.status === 200 || r.status === 404 || r.status === 409,
      });

      if (res.status !== 200 && res.status !== 404 && res.status !== 409) {
        recordUnexpectedStatus("game_server_leave_room");
      }

      must(ok, `Leave room failed: status=${res.status}`);
    });
  }

  sleep(THINK_TIME_SECONDS);
}
