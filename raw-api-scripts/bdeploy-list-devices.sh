#!/bin/bash

# List B-Deploy Registered Devices
# Retrieves all device records for the current network using the B-Deploy Device Endpoints (v2)
# Base URL: https://provision.bsn.cloud/rest-device/v2/device/

set -e

# Default values
FORMAT="json"
DEVICE_ID=""
SETUP_NAME=""

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Lists all B-Deploy device records for the current network.

OPTIONAL:
    -f, --format FORMAT         Output format (default: json)
                                Options: json, summary, table, serials-only, setup-associations
    -d, --device-id DEVICE_ID   Filter by specific device database ID
    -s, --setup-name SETUP_NAME Filter by setup name
    -h, --help                  Display this help message

ENVIRONMENT VARIABLES:
    BS_ACCESS_TOKEN            OAuth2 access token (required)

EXAMPLES:
    # List all devices in JSON format
    $0

    # List in table format
    $0 --format table

    # Show only serial numbers
    $0 -f serials-only

    # Show setup associations
    $0 --format setup-associations

    # Get specific device by ID
    $0 --device-id "device123" --format summary

    # Filter by setup name
    $0 --setup-name "MySetup" --format table

EOF
    exit 1
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--format)
            FORMAT="$2"
            shift 2
            ;;
        -d|--device-id)
            DEVICE_ID="$2"
            shift 2
            ;;
        -s|--setup-name)
            SETUP_NAME="$2"
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

# Check required environment variable
if [ -z "$BS_ACCESS_TOKEN" ]; then
    echo "Error: BS_ACCESS_TOKEN environment variable is required"
    echo "Get a token with: ./scripts/get-token.sh"
    echo ""
    exit 1
fi

# Validate format
case "$FORMAT" in
    json|summary|table|serials-only|setup-associations)
        ;;
    *)
        echo "Error: Invalid format '$FORMAT'"
        echo "Valid options: json, summary, table, serials-only, setup-associations"
        echo ""
        exit 1
        ;;
esac

echo "📡 Fetching B-Deploy device records for current network..."

# Build query parameters
QUERY_PARAMS=""
SEPARATOR="?"

if [ -n "$DEVICE_ID" ]; then
    QUERY_PARAMS="${QUERY_PARAMS}${SEPARATOR}_id=$DEVICE_ID"
    SEPARATOR="&"
    echo "   Filtering by device ID: $DEVICE_ID"
fi

if [ -n "$SETUP_NAME" ]; then
    QUERY_PARAMS="${QUERY_PARAMS}${SEPARATOR}query%5BsetupName%5D=$SETUP_NAME"
    SEPARATOR="&"
    echo "   Filtering by setup name: $SETUP_NAME"
fi

echo "   Format: $FORMAT"
echo

# Make the API request
RESPONSE=$(curl -s -X GET \
  "https://provision.bsn.cloud/rest-device/v2/device/$QUERY_PARAMS" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H "Accept: application/json")

# Check if response contains error
if echo "$RESPONSE" | grep -q '"error"'; then
    echo "❌ Error: API request failed"
    echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
    exit 1
fi

# Parse response based on format
case "$FORMAT" in
    "json")
        echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
        ;;
    "summary")
        echo "=== Device Summary ==="
        echo

        if command -v jq >/dev/null 2>&1; then
            TOTAL=$(echo "$RESPONSE" | jq -r '.total // 0')
            MATCHED=$(echo "$RESPONSE" | jq -r '.matched // 0')

            echo "Total devices:   $TOTAL"
            echo "Matched devices: $MATCHED"
            echo

            if [ "$TOTAL" -gt 0 ]; then
                echo "Device Details:"
                echo "$RESPONSE" | jq -r '.players[] | "- \(.serial) (\(.name // "No name")) - Setup: \(.setupId // "None") - Model: \(.model // "Unknown")"'
            else
                echo "No devices found."
            fi
        else
            echo "Note: Install 'jq' for better formatted output"
            echo "$RESPONSE"
        fi
        ;;
    "table")
        echo "=== Device Table ==="
        echo

        if command -v jq >/dev/null 2>&1; then
            printf "%-15s %-20s %-25s %-15s %-30s\n" "SERIAL" "NAME" "SETUP_ID" "MODEL" "LAST_UPDATED"
            printf "%-15s %-20s %-25s %-15s %-30s\n" "===============" "====================" "=========================" "===============" "=============================="

            echo "$RESPONSE" | jq -r '.players[] | [
                .serial // "N/A",
                (.name // "No name")[0:19],
                (.setupId // "None")[0:24],
                (.model // "Unknown")[0:14],
                (.updatedAt // "Unknown")[0:29]
            ] | @tsv' | while IFS=$'\t' read -r serial name setup_id model updated; do
                printf "%-15s %-20s %-25s %-15s %-30s\n" "$serial" "$name" "$setup_id" "$model" "$updated"
            done
        else
            echo "Note: Install 'jq' for table format"
            echo "$RESPONSE"
        fi
        ;;
    "serials-only")
        echo "=== Device Serial Numbers ==="
        echo

        if command -v jq >/dev/null 2>&1; then
            echo "$RESPONSE" | jq -r '.players[].serial // empty' | sort
        else
            echo "Note: Install 'jq' for serial extraction"
            echo "$RESPONSE"
        fi
        ;;
    "setup-associations")
        echo "=== Setup Associations ==="
        echo

        if command -v jq >/dev/null 2>&1; then
            TOTAL=$(echo "$RESPONSE" | jq -r '.total // 0')
            ASSOCIATED=$(echo "$RESPONSE" | jq -r '[.players[] | select(.setupId != null and .setupId != "")] | length')
            UNASSOCIATED=$(echo "$RESPONSE" | jq -r '[.players[] | select(.setupId == null or .setupId == "")] | length')

            echo "Total devices:           $TOTAL"
            echo "Associated with setup:   $ASSOCIATED"
            echo "Not associated:          $UNASSOCIATED"
            echo

            if [ "$ASSOCIATED" -gt 0 ]; then
                echo "Associated Devices:"
                echo "$RESPONSE" | jq -r '.players[] | select(.setupId != null and .setupId != "") | "- \(.serial): \(.setupId)"'
            fi

            if [ "$UNASSOCIATED" -gt 0 ]; then
                echo
                echo "Unassociated Devices:"
                echo "$RESPONSE" | jq -r '.players[] | select(.setupId == null or .setupId == "") | "- \(.serial): \(.name // "No name")"'
            fi
        else
            echo "Note: Install 'jq' for association analysis"
            echo "$RESPONSE"
        fi
        ;;
esac

echo
echo "✅ Query completed successfully."
