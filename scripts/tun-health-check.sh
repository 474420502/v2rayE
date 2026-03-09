#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_ADDR="${V2RAYN_API_ADDR:-127.0.0.1:18000}"
BACKEND_URL="http://${BACKEND_ADDR}"
API_TOKEN="${V2RAYN_API_TOKEN:-}"
PROBE_URL="${TUN_HEALTHCHECK_URL:-https://www.cloudflare.com/cdn-cgi/trace}"
RESTART_VERIFY="${TUN_HEALTHCHECK_RESTART_VERIFY:-1}"
RESTART_WAIT_SECS="${TUN_HEALTHCHECK_RESTART_WAIT_SECS:-12}"

usage() {
    cat <<'USAGE'
Usage:
  sudo ./scripts/tun-health-check.sh

What this script checks:
  1) Core is running and TUN mode is enabled
  2) TUN takeover is active in policy-routing mode
    3) Direct-bypass fwmark rule is present in kernel state
    4) IPv6 policy routing is present when an IPv6 default route exists
    5) Local HTTP/SOCKS proxy ports can reach an external probe URL
    6) Optional dev restart restores the core and TUN takeover automatically

Environment overrides:
  V2RAYN_API_ADDR                 backend api address, default 127.0.0.1:18000
  V2RAYN_API_TOKEN                bearer token if API auth is enabled
  TUN_HEALTHCHECK_URL             probe URL for HTTP/SOCKS connectivity
  TUN_HEALTHCHECK_RESTART_VERIFY  1 to verify restart restore, 0 to skip
  TUN_HEALTHCHECK_RESTART_WAIT_SECS seconds to wait for restore after restart
USAGE
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
fi

require_cmd() {
    local cmd="$1"
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "missing required command: $cmd" >&2
        exit 1
    fi
}

require_cmd curl
require_cmd ip
require_cmd node

if [[ "$EUID" -ne 0 ]]; then
    echo "this script should be run with sudo/root so it can inspect policy routing and restart the dev stack" >&2
    exit 1
fi

api_call() {
    local method="$1"
    local path="$2"
    local payload="${3:-}"

    local -a args
    args=(
        -sS
        --max-time 10
        -X "$method"
        -H "Content-Type: application/json"
    )

    if [[ -n "$API_TOKEN" ]]; then
        args+=(-H "Authorization: Bearer ${API_TOKEN}")
    fi

    if [[ -n "$payload" ]]; then
        args+=(-d "$payload")
    fi

    curl "${args[@]}" "${BACKEND_URL}${path}"
}

json_get() {
    local json="$1"
    local path="$2"
    printf '%s' "$json" | node -e '
const fs = require("fs");
const input = fs.readFileSync(0, "utf8");
const path = process.argv[1];
let value = "";
try {
  const obj = JSON.parse(input);
  const parts = path.split(".").filter(Boolean);
  let cur = obj;
  for (const part of parts) {
    if (cur == null || !(part in cur)) {
      cur = undefined;
      break;
    }
    cur = cur[part];
  }
  if (Array.isArray(cur) || (cur && typeof cur === "object")) {
    value = JSON.stringify(cur);
  } else if (cur !== undefined && cur !== null) {
    value = String(cur);
  }
} catch {}
process.stdout.write(value);
' "$path"
}

json_array_len() {
    local json="$1"
    local path="$2"
    printf '%s' "$json" | node -e '
const fs = require("fs");
const input = fs.readFileSync(0, "utf8");
const path = process.argv[1];
let value = 0;
try {
  const obj = JSON.parse(input);
  const parts = path.split(".").filter(Boolean);
  let cur = obj;
  for (const part of parts) {
    if (cur == null || !(part in cur)) {
      cur = undefined;
      break;
    }
    cur = cur[part];
  }
  if (Array.isArray(cur)) {
    value = cur.length;
  }
} catch {}
process.stdout.write(String(value));
' "$path"
}

print_header() {
    local title="$1"
    echo
    echo "========== ${title} =========="
}

PASS_COUNT=0
FAIL_COUNT=0

pass() {
    PASS_COUNT=$((PASS_COUNT + 1))
    echo "PASS: $1"
}

fail() {
    FAIL_COUNT=$((FAIL_COUNT + 1))
    echo "FAIL: $1" >&2
}

expect_eq() {
    local actual="$1"
    local expected="$2"
    local label="$3"
    if [[ "$actual" == "$expected" ]]; then
        pass "$label = $actual"
    else
        fail "$label expected $expected, got ${actual:-<empty>}"
    fi
}

expect_nonempty() {
    local value="$1"
    local label="$2"
    if [[ -n "$value" ]]; then
        pass "$label = $value"
    else
        fail "$label is empty"
    fi
}

expect_gt_zero() {
    local value="$1"
    local label="$2"
    if [[ "$value" =~ ^[0-9]+$ ]] && (( value > 0 )); then
        pass "$label = $value"
    else
        fail "$label expected > 0, got ${value:-<empty>}"
    fi
}

probe_http_proxy() {
    local host="$1"
    local port="$2"
    curl -sS -o /dev/null --max-time 15 --proxy "http://${host}:${port}" "$PROBE_URL"
}

probe_socks_proxy() {
    local host="$1"
    local port="$2"
    curl -sS -o /dev/null --max-time 15 --socks5-hostname "${host}:${port}" "$PROBE_URL"
}

poll_backend_ready() {
    local waited=0
    while (( waited < RESTART_WAIT_SECS )); do
        if curl -sS --max-time 2 "${BACKEND_URL}/api/health" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
        waited=$((waited + 1))
    done
    return 1
}

poll_core_restored() {
    local expected_profile="$1"
    local waited=0
    while (( waited < RESTART_WAIT_SECS )); do
        local status_json diag_json
        status_json="$(api_call GET /api/core/status || true)"
        diag_json="$(api_call GET /api/routing/diagnostics || true)"

        local running current_profile takeover_active takeover_mode direct_bypass_rule
        running="$(json_get "$status_json" data.running)"
        current_profile="$(json_get "$status_json" data.currentProfileId)"
        takeover_active="$(json_get "$diag_json" data.tunTakeoverActive)"
        takeover_mode="$(json_get "$diag_json" data.tunTakeoverMode)"
        direct_bypass_rule="$(json_get "$diag_json" data.tunDirectBypassRule)"

        if [[ "$running" == "true" && "$current_profile" == "$expected_profile" && "$takeover_active" == "true" && "$takeover_mode" == "policy-routing" && "$direct_bypass_rule" == "true" ]]; then
            return 0
        fi

        sleep 1
        waited=$((waited + 1))
    done
    return 1
}

collect_runtime_state() {
    CONFIG_JSON="$(api_call GET /api/config)"
    STATUS_JSON="$(api_call GET /api/core/status)"
    DIAG_JSON="$(api_call GET /api/routing/diagnostics)"

    LISTEN_ADDR="$(json_get "$CONFIG_JSON" data.listenAddr)"
    SOCKS_PORT="$(json_get "$CONFIG_JSON" data.socksPort)"
    HTTP_PORT="$(json_get "$CONFIG_JSON" data.httpPort)"
    TUN_MODE="$(json_get "$CONFIG_JSON" data.tunMode)"
    TUN_NAME="$(json_get "$CONFIG_JSON" data.tunName)"
    STATUS_RUNNING="$(json_get "$STATUS_JSON" data.running)"
    CURRENT_PROFILE_ID="$(json_get "$STATUS_JSON" data.currentProfileId)"
    TUN_TAKEOVER_ACTIVE="$(json_get "$DIAG_JSON" data.tunTakeoverActive)"
    TUN_TAKEOVER_MODE="$(json_get "$DIAG_JSON" data.tunTakeoverMode)"
    TUN_DIRECT_BYPASS_RULE="$(json_get "$DIAG_JSON" data.tunDirectBypassRule)"
    TUN_DIRECT_BYPASS_MARK="$(json_get "$DIAG_JSON" data.tunDirectBypassMark)"
    POLICY_ROUTE_TABLE="$(json_get "$DIAG_JSON" data.tunPolicyRouteTable)"
    POLICY_RULE_COUNT="$(json_array_len "$DIAG_JSON" data.tunPolicyRules)"
    DEFAULT_ROUTE_DEVICE="$(json_get "$DIAG_JSON" data.defaultRouteDevice)"
    DIAG_WARNING="$(json_get "$DIAG_JSON" data.warning)"
    IPV6_DEFAULT_ROUTE_PRESENT=0
    if ip -6 route show default 2>/dev/null | grep -q .; then
        IPV6_DEFAULT_ROUTE_PRESENT=1
    fi
}

print_runtime_summary() {
    print_header "$1"
    echo "backend: ${BACKEND_URL}"
    echo "probe: ${PROBE_URL}"
    echo "tunMode: ${TUN_MODE:-<empty>}"
    echo "tunName: ${TUN_NAME:-<empty>}"
    echo "running: ${STATUS_RUNNING:-<empty>}"
    echo "currentProfileId: ${CURRENT_PROFILE_ID:-<empty>}"
    echo "takeover: ${TUN_TAKEOVER_ACTIVE:-<empty>} (${TUN_TAKEOVER_MODE:-inactive})"
    echo "directBypassRule: ${TUN_DIRECT_BYPASS_RULE:-<empty>}"
    echo "directBypassMark: ${TUN_DIRECT_BYPASS_MARK:-<empty>}"
    echo "policyTable: ${POLICY_ROUTE_TABLE:-<empty>}"
    echo "policyRules: ${POLICY_RULE_COUNT:-0}"
    echo "defaultRouteDevice: ${DEFAULT_ROUTE_DEVICE:-<empty>}"
    echo "ipv6DefaultRoute: ${IPV6_DEFAULT_ROUTE_PRESENT}"
    echo "warning: ${DIAG_WARNING:-<empty>}"
    echo "httpProxy: ${LISTEN_ADDR:-127.0.0.1}:${HTTP_PORT:-?}"
    echo "socksProxy: ${LISTEN_ADDR:-127.0.0.1}:${SOCKS_PORT:-?}"
}

check_live_state() {
    print_header "Live Checks"
    expect_eq "$STATUS_RUNNING" "true" "core running"
    expect_nonempty "$CURRENT_PROFILE_ID" "current profile"

    if [[ "$TUN_MODE" == "off" || -z "$TUN_MODE" ]]; then
        fail "tun mode is off; enable TUN before running this health check"
    else
        pass "tun mode = $TUN_MODE"
    fi

    expect_eq "$TUN_TAKEOVER_ACTIVE" "true" "tun takeover active"
    expect_eq "$TUN_TAKEOVER_MODE" "policy-routing" "tun takeover mode"
    expect_eq "$TUN_DIRECT_BYPASS_RULE" "true" "tun direct bypass rule"
    expect_gt_zero "$TUN_DIRECT_BYPASS_MARK" "tun direct bypass mark"
    expect_gt_zero "$POLICY_ROUTE_TABLE" "policy route table"
    expect_gt_zero "$POLICY_RULE_COUNT" "policy rule count"

    local table_route catch_all_rule kernel_rule_count direct_bypass_rule
    table_route="$(ip -4 route show table "$POLICY_ROUTE_TABLE" 2>/dev/null | head -n 1 || true)"
    catch_all_rule="$(ip -4 rule show | grep -E "lookup ${POLICY_ROUTE_TABLE}$" | head -n 1 || true)"
    kernel_rule_count="$(ip -4 rule show | grep -E -c "lookup (main|${POLICY_ROUTE_TABLE})$" || true)"
    direct_bypass_rule="$(ip -4 rule show | grep -E "fwmark (0x)?$(printf '%x' "$TUN_DIRECT_BYPASS_MARK") .*lookup main|fwmark ${TUN_DIRECT_BYPASS_MARK} .*lookup main" | head -n 1 || true)"

    if [[ "$table_route" == *"default dev ${TUN_NAME}"* ]]; then
        pass "policy route table default = $table_route"
    else
        fail "policy route table ${POLICY_ROUTE_TABLE} missing default dev ${TUN_NAME}"
    fi

    if [[ -n "$catch_all_rule" ]]; then
        pass "catch-all policy rule = $catch_all_rule"
    else
        fail "missing catch-all ip rule for table ${POLICY_ROUTE_TABLE}"
    fi

    if [[ "$kernel_rule_count" =~ ^[0-9]+$ ]] && (( kernel_rule_count >= POLICY_RULE_COUNT )); then
        pass "kernel rule count >= diagnostics count (${kernel_rule_count} >= ${POLICY_RULE_COUNT})"
    else
        fail "kernel rule count is inconsistent with diagnostics (${kernel_rule_count:-0} < ${POLICY_RULE_COUNT:-0})"
    fi

    if [[ -n "$direct_bypass_rule" ]]; then
        pass "direct-bypass fwmark rule = $direct_bypass_rule"
    else
        fail "missing IPv4 direct-bypass fwmark rule for mark ${TUN_DIRECT_BYPASS_MARK}"
    fi

    if [[ "$IPV6_DEFAULT_ROUTE_PRESENT" == "1" ]]; then
        local table_route6 catch_all_rule6 direct_bypass_rule6
        table_route6="$(ip -6 route show table "$POLICY_ROUTE_TABLE" default 2>/dev/null | head -n 1 || true)"
        catch_all_rule6="$(ip -6 rule show | grep -E "lookup ${POLICY_ROUTE_TABLE}$" | head -n 1 || true)"
        direct_bypass_rule6="$(ip -6 rule show | grep -E "fwmark (0x)?$(printf '%x' "$TUN_DIRECT_BYPASS_MARK") .*lookup main|fwmark ${TUN_DIRECT_BYPASS_MARK} .*lookup main" | head -n 1 || true)"

        if [[ "$table_route6" == *"default dev ${TUN_NAME}"* ]]; then
            pass "IPv6 policy route table default = $table_route6"
        else
            fail "IPv6 policy route table ${POLICY_ROUTE_TABLE} missing default dev ${TUN_NAME}"
        fi

        if [[ -n "$catch_all_rule6" ]]; then
            pass "IPv6 catch-all policy rule = $catch_all_rule6"
        else
            fail "missing IPv6 catch-all ip rule for table ${POLICY_ROUTE_TABLE}"
        fi

        if [[ -n "$direct_bypass_rule6" ]]; then
            pass "IPv6 direct-bypass fwmark rule = $direct_bypass_rule6"
        else
            fail "missing IPv6 direct-bypass fwmark rule for mark ${TUN_DIRECT_BYPASS_MARK}"
        fi
    fi

    if probe_http_proxy "$LISTEN_ADDR" "$HTTP_PORT"; then
        pass "HTTP proxy connectivity via ${LISTEN_ADDR}:${HTTP_PORT}"
    else
        fail "HTTP proxy connectivity failed via ${LISTEN_ADDR}:${HTTP_PORT}"
    fi

    if probe_socks_proxy "$LISTEN_ADDR" "$SOCKS_PORT"; then
        pass "SOCKS proxy connectivity via ${LISTEN_ADDR}:${SOCKS_PORT}"
    else
        fail "SOCKS proxy connectivity failed via ${LISTEN_ADDR}:${SOCKS_PORT}"
    fi
}

check_restart_restore() {
    if [[ "$RESTART_VERIFY" != "1" ]]; then
        print_header "Restart Restore Check"
        echo "skipped: set TUN_HEALTHCHECK_RESTART_VERIFY=1 to enable"
        return 0
    fi

    print_header "Restart Restore Check"
    echo "restarting dev stack to verify core auto-restore"
    V2RAYE_COUPLED_LIFECYCLE=0 "$ROOT_DIR/scripts/stop-dev.sh" >/dev/null
    "$ROOT_DIR/scripts/start-backend.sh" >/dev/null

    if ! poll_backend_ready; then
        fail "backend did not become healthy after restart"
        return 0
    fi

    if poll_core_restored "$CURRENT_PROFILE_ID"; then
        pass "core and TUN policy routing restored after restart"
    else
        fail "core or TUN policy routing did not restore within ${RESTART_WAIT_SECS}s"
    fi
}

collect_runtime_state
print_runtime_summary "Initial Snapshot"
check_live_state
check_restart_restore
collect_runtime_state
print_runtime_summary "Post-Restart Snapshot"

print_header "Result"
echo "passes: ${PASS_COUNT}"
echo "failures: ${FAIL_COUNT}"

if (( FAIL_COUNT > 0 )); then
    exit 1
fi

echo "PASS: TUN VPN health check completed"
exit 0