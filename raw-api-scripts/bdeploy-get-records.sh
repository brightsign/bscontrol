#!/bin/bash

# Get B-Deploy Setup Records
# Retrieves all setup records for a network using the B-Deploy Setup Endpoints (v3)
# Base URL: https://provision.bsn.cloud/rest-setup/v3/setup/

set -e

# Default values
FILTER_USERNAME=""
FILTER_PACKAGE=""

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Retrieves all B-Deploy setup records for a network.

OPTIONAL:
    -u, --username USERNAME     Filter by user who created the setup
    -p, --package PACKAGE       Filter by setup package name
    -h, --help                  Display this help message

ENVIRONMENT VARIABLES:
    BS_NETWORK                  BSN.cloud network name (required)
    BS_ACCESS_TOKEN            OAuth2 access token (required)

EXAMPLES:
    # List all setup records
    $0

    # Filter by username
    $0 --username john@example.com

    # Filter by package name
    $0 --package "MySetup"

    # Filter by both
    $0 -u john@example.com -p "MySetup"

EOF
    exit 1
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -u|--username)
            FILTER_USERNAME="$2"
            shift 2
            ;;
        -p|--package)
            FILTER_PACKAGE="$2"
            shift 2
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Error: Unknown option '$1'"
            echo ""
            usage
            ;;
    esac
done

# Check required environment variables
if [ -z "$BS_NETWORK" ]; then
    echo "Error: BS_NETWORK environment variable is required"
    exit 1
fi

if [ -z "$BS_ACCESS_TOKEN" ]; then
    echo "Error: BS_ACCESS_TOKEN environment variable is required"
    exit 1
fi

# Step 1: Set network context (REQUIRED before any B-Deploy API calls)
echo "🔧 Setting network context to '$BS_NETWORK'..."
NETWORK_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X PUT \
  "https://api.bsn.cloud/2022/06/REST/Self/Session/Network" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$BS_NETWORK\"}")

NETWORK_HTTP_CODE=$(echo "$NETWORK_RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)

if [ "$NETWORK_HTTP_CODE" != "200" ] && [ "$NETWORK_HTTP_CODE" != "204" ]; then
    echo "❌ Failed to set network context (HTTP $NETWORK_HTTP_CODE)"
    NETWORK_BODY=$(echo "$NETWORK_RESPONSE" | sed '/HTTP_CODE:/d')
    if [ -n "$NETWORK_BODY" ]; then
        echo "$NETWORK_BODY" | jq '.' 2>/dev/null || echo "$NETWORK_BODY"
    fi
    exit 1
fi
echo "✅ Network context set"
echo

# Step 2: Fetch B-Deploy setup records
echo "📋 Fetching B-Deploy setup records for network: $BS_NETWORK"

# Build curl command with proper URL encoding using -G and --data-urlencode
CURL_CMD=(curl -s -G "https://provision.bsn.cloud/rest-setup/v3/setup"
  --data-urlencode "NetworkName=$BS_NETWORK"
  -H "Authorization: Bearer $BS_ACCESS_TOKEN"
  -H "Content-Type: application/json"
)

# Add optional filters if provided via command line
if [ -n "$FILTER_USERNAME" ]; then
    echo "Filter: username=$FILTER_USERNAME"
    CURL_CMD+=(--data-urlencode "username=$FILTER_USERNAME")
fi

if [ -n "$FILTER_PACKAGE" ]; then
    echo "Filter: packageName=$FILTER_PACKAGE"
    CURL_CMD+=(--data-urlencode "packageName=$FILTER_PACKAGE")
fi

echo

# Execute the curl command and capture response
RESPONSE=$("${CURL_CMD[@]}")

# Check if response is empty
if [ -z "$RESPONSE" ]; then
    echo "❌ Error: Empty response from API"
    exit 1
fi

# Check for error in response
if echo "$RESPONSE" | grep -q '"error"'; then
    echo "❌ Error: API request failed"
    echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
    exit 1
fi

# Try to parse and display the response
if command -v jq >/dev/null 2>&1; then
    # Pretty print with jq
    ITEM_COUNT=$(echo "$RESPONSE" | jq '.items | length' 2>/dev/null || echo "0")

    echo "✅ Found $ITEM_COUNT setup record(s)"
    echo

    if [ "$ITEM_COUNT" -gt 0 ]; then
        # Display in table format
        printf "%-40s %-35s %-15s\n" "SETUP-ID" "PACKAGE NAME" "SETUP TYPE"
        printf "%-40s %-35s %-15s\n" "$(printf '%.0s-' {1..40})" "$(printf '%.0s-' {1..35})" "$(printf '%.0s-' {1..15})"

        echo "$RESPONSE" | jq -r '.items[] | [._id, .packageName, .setupType] | @tsv' | \
        while IFS=$'\t' read -r id package type; do
            printf "%-40s %-35s %-15s\n" "$id" "${package:0:35}" "${type:0:15}"
        done
    fi

    echo
    echo "💡 For full JSON output: $0 [options] | jq '.'"
else
    echo "Note: Install 'jq' for formatted output"
    echo "$RESPONSE"
fi

echo