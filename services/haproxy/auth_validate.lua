local function send_get_request(host, port, path, extra_headers)
    local socket = core.tcp()
    local address = host .. ":" .. tostring(port or "8080")

    socket:settimeout(2000)

    if not socket:connect(address) then
        socket:close()
        return nil, "connect_failed"
    end

    local lines = {
        "GET " .. path .. " HTTP/1.0",
        "Host: " .. host,
    }

    if extra_headers then
        for _, header in ipairs(extra_headers) do
            table.insert(lines, header)
        end
    end

    table.insert(lines, "")
    table.insert(lines, "")

    local request = table.concat(lines, "\r\n")

    if not socket:send(request) then
        socket:close()
        return nil, "send_failed"
    end

    local status_line = socket:receive("*l")
    if not status_line then
        socket:close()
        return nil, "receive_failed"
    end

    while true do
        local line = socket:receive("*l")
        if not line or line == "" then
            break
        end
    end

    local body = socket:receive("*a") or ""

    socket:close()

    local status_code = tonumber(string.match(status_line, "^HTTP/%d+%.%d+ (%d%d%d)"))
    if not status_code then
        return nil, "bad_status_line"
    end

    return status_code, body, nil
end

local function trim(value)
    return string.match(value or "", "^%s*(.-)%s*$") or ""
end

local function origin_is_allowed(origin)
    origin = trim(origin)
    if origin == "" then
        return false
    end

    local allowed_origins = trim(os.getenv("HAPROXY_ALLOWED_CORS_ORIGINS") or "")
    if allowed_origins == "" then
        allowed_origins = os.getenv("ALLOWED_CORS_ORIGINS") or ""
    end

    for item in string.gmatch(allowed_origins, "([^,]+)") do
        local allowed = trim(item)
        if allowed == "*" or allowed == origin then
            return true
        end
    end

    return false
end

core.register_action("mark_cors_origin", { "http-req" }, function(txn)
    local origin = trim(txn.sf:req_hdr("Origin") or "")
    local allowed = origin_is_allowed(origin)

    txn:set_var("txn.cors_origin", origin)
    txn:set_var("txn.cors_origin_allowed", allowed)
end)

core.register_action("validate_sso_access_token", { "http-req" }, function(txn)
    local auth_header = txn:get_var("txn.auth_header")
    if not auth_header or auth_header == "" then
        local bearer = txn:get_var("txn.bearer")
        if bearer and bearer ~= "" then
            auth_header = "Bearer " .. bearer
        else
            txn:set_var("txn.sso_validate_ok", false)
            txn:set_var("txn.sso_validate_outage", false)
            return
        end
    end

    local host = os.getenv("HAPROXY_LOOPBACK_HOST") or "127.0.0.1"
    local port = os.getenv("HAPROXY_LOOPBACK_PORT") or "80"
    local status_code, _, err = send_get_request(host, port, "/__internal/auth/validate", {
        "Authorization: " .. auth_header,
    })

    if not status_code then
        txn:set_var("txn.sso_validate_ok", false)
        txn:set_var("txn.sso_validate_outage", true)
        txn:set_var("txn.sso_validate_error", err or "unknown_error")
        return
    end

    txn:set_var("txn.sso_validate_status", status_code)

    if status_code == 200 then
        txn:set_var("txn.sso_validate_ok", true)
        txn:set_var("txn.sso_validate_outage", false)
        return
    end

    txn:set_var("txn.sso_validate_ok", false)
    txn:set_var("txn.sso_validate_outage", status_code >= 500)
end)

core.register_action("authorize_internal_service", { "http-req" }, function(txn)
    local provided = txn.sf:req_hdr("X-Internal-Proxy-Token") or ""
    local expected = os.getenv("INTERNAL_PROXY_TOKEN") or ""

    if expected == "" or provided == "" then
        txn:set_var("txn.internal_service_ok", false)
        return
    end

    txn:set_var("txn.internal_service_ok", provided == expected)
end)

core.register_action("resolve_game_room_owner", { "http-req" }, function(txn)
    local room_id = txn:get_var("txn.game_room_id")
    if not room_id or room_id == "" then
        txn:set_var("txn.game_owner_lookup_ok", false)
        txn:set_var("txn.game_owner_lookup_outage", false)
        return
    end

    local host = os.getenv("HAPROXY_LOOPBACK_HOST") or "127.0.0.1"
    local port = os.getenv("HAPROXY_LOOPBACK_PORT") or "80"
    local status_code, body, err = send_get_request(host, port, "/__internal/matchmaking/rooms/" .. room_id .. "/owner", nil)

    if not status_code then
        txn:set_var("txn.game_owner_lookup_ok", false)
        txn:set_var("txn.game_owner_lookup_outage", true)
        txn:set_var("txn.game_owner_lookup_error", err or "unknown_error")
        return
    end

    txn:set_var("txn.game_owner_lookup_status", status_code)

    if status_code ~= 200 then
        txn:set_var("txn.game_owner_lookup_ok", false)
        txn:set_var("txn.game_owner_lookup_outage", status_code >= 500)
        return
    end

    local instance_key = string.match(body or "", '"instance_key"%s*:%s*"([^"]+)"')
    if not instance_key or instance_key == "" then
        txn:set_var("txn.game_owner_lookup_ok", false)
        txn:set_var("txn.game_owner_lookup_outage", false)
        txn:set_var("txn.game_owner_lookup_error", "instance_key_missing")
        return
    end

    txn:set_var("txn.game_owner_instance_key", instance_key)
    txn:set_var("txn.game_owner_lookup_ok", true)
    txn:set_var("txn.game_owner_lookup_outage", false)
end)

core.register_action("extract_game_room_id", { "http-req" }, function(txn)
    local function numeric_room_id(value)
        if not value or value == "" then
            return nil
        end

        value = string.match(value, "^%s*(.-)%s*$") or ""
        if string.match(value, "^%d+$") then
            return value
        end

        return nil
    end

    local room_id = numeric_room_id(txn.sf:req_hdr("X-Room-ID"))

    if not room_id or room_id == "" then
        room_id = numeric_room_id(txn.sf:urlp("room_id"))
    end

    if (not room_id or room_id == "") then
        local path = txn.sf:path() or ""
        room_id = string.match(path, "^/api/game/rooms/(%d+)/")
            or string.match(path, "^/api/game/rooms/(%d+)$")
    end

    if room_id and room_id ~= "" then
        txn:set_var("txn.game_room_id", room_id)
    end
end)
