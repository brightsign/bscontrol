#!/bin/bash

# Get B-Deploy Device Record by Serial Number
# Retrieves device record and associated setup information for a specific player
# Base URL: https://provision.bsn.cloud/rest-device/v2/device/

set -e

# Default values
SERIAL=""
FORMAT="json"

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Retrieves a B-Deploy device record by player serial number.

REQUIRED OPTIONS:
    -s, --serial SERIAL         Player serial number

OPTIONAL:
    -f, --format FORMAT         Output format: json (default), summary, setup-only
    -h, --help                  Display this help message

ENVIRONMENT VARIABLES:
    BS_ACCESS_TOKEN            OAuth2 access token (required)

EXAMPLES:
    # Get device record in JSON format
    $0 --serial "ABC123456789"

    # Get summary view
    $0 -s "ABC123456789" --format summary

    # Check setup association only
    $0 -s "ABC123456789" -f setup-only

EOF
    exit 1
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--serial)
            SERIAL="$2"
            shift 2
            ;;
        -f|--format)
            FORMAT="$2"
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

# Check required parameters
if [ -z "$SERIAL" ]; then
    echo "Error: --serial is required"
    echo ""
    usage
fi

# Validate format
case "$FORMAT" in
    json|summary|setup-only)
        ;;
    *)
        echo "Error: Invalid format '$FORMAT'"
        echo "Valid options: json, summary, setup-only"
        echo ""
        exit 1
        ;;
esac

echo "📡 Fetching device record for player: $SERIAL"
echo "   Format: $FORMAT"
echo

# Make the API request
RESPONSE=$(curl -s -X GET \
  "https://provision.bsn.cloud/rest-device/v2/device/?serial=$SERIAL" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H "Accept: application/json" \
  -H "Accept-Encoding: gzip,deflate" \
  -H "Connection: Keep-Alive")

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
        echo "Serial: $SERIAL"
        echo

        # Extract key fields using jq if available, otherwise use grep
        if command -v jq >/dev/null 2>&1; then
            DEVICE_ID=$(echo "$RESPONSE" | jq -r '.players[0]._id // "Not found"')
            DEVICE_NAME=$(echo "$RESPONSE" | jq -r '.players[0].name // "Not set"')
            SETUP_ID=$(echo "$RESPONSE" | jq -r '.players[0].setupId // "Not associated"')
            USERNAME=$(echo "$RESPONSE" | jq -r '.players[0].username // "Not set"')
            MODEL=$(echo "$RESPONSE" | jq -r '.players[0].model // "Not set"')
            URL=$(echo "$RESPONSE" | jq -r '.players[0].url // "Not set"')
            CREATED=$(echo "$RESPONSE" | jq -r '.players[0].createdAt // "Not set"')
            UPDATED=$(echo "$RESPONSE" | jq -r '.players[0].updatedAt // "Not set"')

            echo "Device ID:     $DEVICE_ID"
            echo "Name:          $DEVICE_NAME"
            echo "Model:         $MODEL"
            echo "Setup ID:      $SETUP_ID"
            echo "Username:      $USERNAME"
            echo "Direct URL:    $URL"
            echo "Created:       $CREATED"
            echo "Last Updated:  $UPDATED"
        else
            echo "Note: Install 'jq' for better formatted output"
            echo "$RESPONSE"
        fi
        ;;
    "setup-only")
        echo "=== Setup Association ==="
        echo "Serial: $SERIAL"
        echo

        if command -v jq >/dev/null 2>&1; then
            SETUP_ID=$(echo "$RESPONSE" | jq -r '.players[0].setupId // "Not associated"')
            DEVICE_NAME=$(echo "$RESPONSE" | jq -r '.players[0].name // "Not set"')

            echo "Device Name: $DEVICE_NAME"
            echo "Setup ID:    $SETUP_ID"
            echo

            if [ "$SETUP_ID" != "Not associated" ] && [ "$SETUP_ID" != "null" ] && [ -n "$SETUP_ID" ]; then
                echo "✅ Status: Associated with setup"
            else
                echo "⚠️  Status: No setup association"
            fi
        else
            echo "Note: Install 'jq' for better formatted output"
            echo "$RESPONSE"
        fi
        ;;
esac

echo
echo "✅ Query completed successfully."
