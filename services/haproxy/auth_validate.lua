local function send_validate_request(host, port, auth_header)
    local socket = core.tcp()

    socket:settimeout(2000)

    if not socket:connect(host, port) then
        socket:close()
        return nil, "connect_failed"
    end

    local request = table.concat({
        "GET /api/auth/validate HTTP/1.0",
        "Host: " .. host,
        "Authorization: " .. auth_header,
        "",
        ""
    }, "\r\n")

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

    socket:close()

    local status_code = tonumber(string.match(status_line, "^HTTP/%d+%.%d+ (%d%d%d)"))
    if not status_code then
        return nil, "bad_status_line"
    end

    return status_code, nil
end

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

    local host = os.getenv("SSO_VALIDATE_HOST") or "sso-core"
    local port = os.getenv("SSO_VALIDATE_PORT") or "8080"
    local status_code, err = send_validate_request(host, port, auth_header)

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
