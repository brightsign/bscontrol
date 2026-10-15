#!/bin/bash
#
# Reproduce the missing rDWS routes documented in
# workarounds/rdws-missing-supervisor-and-crashdump-routes.md.
#
# Authenticates against BSN.cloud exactly the way the bscontrol SDK does
# (OAuth2 client_credentials via HTTP Basic, then a network-context PUT on the
# same token), then probes the two routes reported missing plus the working
# routes the SDK workaround stands on. Prints a verdict and leaves every raw
# response on disk as ticket evidence.
#
# Usage:
#   BS_CLIENT_ID=... BS_SECRET=... BS_NETWORK=... ./probe-rdws-supervisor-routes.sh <SERIAL>
#
# Optional overrides (defaults match internal/config/config.go DefaultConfig):
#   TOKEN_ENDPOINT  https://auth.bsn.cloud/realms/bsncloud/protocol/openid-connect/token
#   BSN_BASE_URL    https://api.bsn.cloud
#   RDWS_BASE_URL   https://ws.bsn.cloud/rest/v1
#   API_VERSION     2022/06/REST
#   OUTPUT_DIR      ./rdws-probe-<serial>-<timestamp>
#
# This script is strictly read-only. It never calls
# POST /system/supervisors/delete/, which is in the same unverified route
# family but destroys downloaded supervisor builds.

set -euo pipefail

readonly TOKEN_ENDPOINT="${TOKEN_ENDPOINT:-https://auth.bsn.cloud/realms/bsncloud/protocol/openid-connect/token}"
readonly BSN_BASE_URL="${BSN_BASE_URL:-https://api.bsn.cloud}"
readonly API_VERSION="${API_VERSION:-2022/06/REST}"
readonly CURL_CONNECT_TIMEOUT_SECONDS=10
readonly CURL_MAX_TIME_SECONDS=60

# A trailing slash would produce a double slash in every probe URL, which some
# gateways route differently than the single-slash form.
RDWS_BASE_URL="${RDWS_BASE_URL:-https://ws.bsn.cloud/rest/v1}"
RDWS_BASE_URL="${RDWS_BASE_URL%/}"
readonly RDWS_BASE_URL

# Preconditions: fail loudly rather than emitting confusing 401s later.
if [[ $# -ne 1 ]]; then
    echo "usage: $0 <SERIAL>" >&2
    exit 2
fi
readonly DEVICE_SERIAL="$1"

for required_command in curl jq; do
    if ! command -v "$required_command" >/dev/null 2>&1; then
        echo "error: $required_command is required but not installed" >&2
        exit 2
    fi
done

for required_variable in BS_CLIENT_ID BS_SECRET BS_NETWORK; do
    if [[ -z "${!required_variable:-}" ]]; then
        echo "error: $required_variable must be set (see internal/config/config.go LoadFromEnv)" >&2
        exit 2
    fi
done

readonly OUTPUT_DIR="${OUTPUT_DIR:-./rdws-probe-${DEVICE_SERIAL}-$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$OUTPUT_DIR"

echo "rDWS supervisor/crash-dump route probe"
echo "  serial:    $DEVICE_SERIAL"
echo "  network:   $BS_NETWORK"
echo "  rdws base: $RDWS_BASE_URL"
echo "  evidence:  $OUTPUT_DIR"
echo

# ---------------------------------------------------------------------------
# Step 1: OAuth2 client_credentials, client id/secret as HTTP Basic.
# Mirrors AuthManager.Authenticate (internal/auth/auth.go:43).
# ---------------------------------------------------------------------------

echo "==> authenticating at $TOKEN_ENDPOINT"
token_response_file="$OUTPUT_DIR/00-token-response.json"
token_status="$(curl --silent --show-error \
    --connect-timeout "$CURL_CONNECT_TIMEOUT_SECONDS" --max-time "$CURL_MAX_TIME_SECONDS" \
    --user "$BS_CLIENT_ID:$BS_SECRET" \
    --header 'Content-Type: application/x-www-form-urlencoded' \
    --data-urlencode 'grant_type=client_credentials' \
    --output "$token_response_file" \
    --write-out '%{http_code}' \
    "$TOKEN_ENDPOINT")"

if [[ "$token_status" != "200" ]]; then
    echo "error: token request returned HTTP $token_status" >&2
    jq . "$token_response_file" 2>/dev/null || cat "$token_response_file" >&2
    exit 1
fi

ACCESS_TOKEN="$(jq -r '.access_token // empty' "$token_response_file")"
if [[ -z "$ACCESS_TOKEN" ]]; then
    echo "error: token response contained no access_token" >&2
    exit 1
fi
readonly ACCESS_TOKEN

# The token itself must never reach the evidence directory or the terminal --
# it is a live credential for the whole network.
jq 'del(.access_token, .refresh_token) | .access_token = "<redacted>"' \
    "$token_response_file" > "$token_response_file.tmp"
mv "$token_response_file.tmp" "$token_response_file"

echo "    got access token (expires_in $(jq -r '.expires_in // "?"' "$token_response_file")s)"

# ---------------------------------------------------------------------------
# Step 2: establish network context on that same token.
# Mirrors AuthManager.SetNetwork (internal/auth/auth.go:82). rDWS calls fail
# without this even though the token is otherwise valid.
# ---------------------------------------------------------------------------

network_url="$BSN_BASE_URL/$API_VERSION/Self/Session/Network"
echo "==> setting network context: PUT $network_url"
network_response_file="$OUTPUT_DIR/01-set-network.json"
network_status="$(curl --silent --show-error \
    --connect-timeout "$CURL_CONNECT_TIMEOUT_SECONDS" --max-time "$CURL_MAX_TIME_SECONDS" \
    --request PUT \
    --header "Authorization: Bearer $ACCESS_TOKEN" \
    --header 'Content-Type: application/json' \
    --data "$(jq --null-input --arg name "$BS_NETWORK" '{name: $name}')" \
    --output "$network_response_file" \
    --write-out '%{http_code}' \
    "$network_url")"

if [[ "$network_status" != "200" && "$network_status" != "204" ]]; then
    echo "error: setting network '$BS_NETWORK' returned HTTP $network_status" >&2
    cat "$network_response_file" >&2
    exit 1
fi
echo "    network context set (HTTP $network_status)"
echo

# ---------------------------------------------------------------------------
# Step 3: probe each route and record the result.
# ---------------------------------------------------------------------------

declare -a probe_labels=()
declare -a probe_statuses=()

# probe <slug> <label> <path-with-query>
probe() {
    local slug="$1" label="$2" path="$3"
    local url="$RDWS_BASE_URL/$path"
    local body_file="$OUTPUT_DIR/$slug.json"

    local status
    status="$(curl --silent --show-error \
        --connect-timeout "$CURL_CONNECT_TIMEOUT_SECONDS" --max-time "$CURL_MAX_TIME_SECONDS" \
        --header "Authorization: Bearer $ACCESS_TOKEN" \
        --output "$body_file" \
        --write-out '%{http_code}' \
        --get \
        --data-urlencode 'destinationType=player' \
        --data-urlencode "destinationName=$DEVICE_SERIAL" \
        "$url" || echo "000")"

    printf '==> %s\n' "$label"
    printf '    GET %s?destinationType=player&destinationName=%s\n' "$url" "$DEVICE_SERIAL"
    printf '    HTTP %s -> %s\n' "$status" "$body_file"
    if jq --exit-status . "$body_file" >/dev/null 2>&1; then
        jq --compact-output . "$body_file" | head --bytes=800 | sed 's/^/    /'
    else
        head --bytes=400 "$body_file" | sed 's/^/    /'
    fi
    echo

    probe_labels+=("$label")
    probe_statuses+=("$status")
}

# Control: proves the rDWS passthrough reaches this player at all, so a 404 on
# the routes below cannot be blamed on the serial or the network context.
probe "02-control-info" "CONTROL  GET /info/ (known working)" "info/"

# The two routes reported missing.
probe "03-missing-supervisors" "SUSPECT  GET /system/supervisors/" "system/supervisors/"
probe "04-missing-crash-dumps" "SUSPECT  GET /logs/crash-dumps/" "logs/crash-dumps/"

# The routes the bscontrol workaround actually uses. An empty file list here is
# not a failure: it means the player has no stored builds / no crash dumps.
probe "05-workaround-supervisors" "WORKAROUND  GET /files/sys/supervisors/" "files/sys/supervisors/"
probe "06-workaround-crash-dumps" "WORKAROUND  GET /files/sd/brightsign-dumps/" "files/sd/brightsign-dumps/"

# ---------------------------------------------------------------------------
# Step 4: verdict.
# ---------------------------------------------------------------------------

echo "=========================================================="
for index in "${!probe_labels[@]}"; do
    printf '  HTTP %-4s %s\n' "${probe_statuses[$index]}" "${probe_labels[$index]}"
done
echo "=========================================================="

readonly control_status="${probe_statuses[0]}"
readonly supervisors_status="${probe_statuses[1]}"
readonly crash_dumps_status="${probe_statuses[2]}"

if [[ "$control_status" != "200" ]]; then
    echo
    echo "INCONCLUSIVE: the control route GET /info/ returned HTTP $control_status."
    echo "The player may be offline or unreachable. Nothing can be concluded about"
    echo "the supervisor and crash-dump routes from this run."
    exit 1
fi

echo
if [[ "$supervisors_status" == "404" && "$crash_dumps_status" == "404" ]]; then
    echo "REPRODUCED: both routes 404 while the control route succeeds."
    echo "The workaround in internal/services/rdws_supervisor.go is still required."
    echo "Attach $OUTPUT_DIR to the ticket."
    exit 0
fi

if [[ "$supervisors_status" == "200" && "$crash_dumps_status" == "200" ]]; then
    echo "FIXED: both routes now return 200."
    echo "Verify the payload shapes above against the acceptance criteria, then"
    echo "revert GetStoredSupervisors/GetCrashDumpFiles to the direct routes and"
    echo "delete storedSupervisorsFromFileList/crashDumpEntriesFromFileList."
    exit 0
fi

echo "MIXED RESULT: supervisors=$supervisors_status crash-dumps=$crash_dumps_status."
echo "This is neither the documented failure nor a clean fix. Inspect the bodies"
echo "above before updating the ticket."
exit 1
