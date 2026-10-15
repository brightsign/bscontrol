#!/bin/bash

# Associate Player with B-Deploy Setup Record
# Links a player (by serial number) to a specific setup configuration
# Base URL: https://provision.bsn.cloud/rest-device/v2/device/

set -e

# Default values
SERIAL=""
SETUP_ID=""
NETWORK=""
USERNAME=""
DEVICE_NAME=""
DEVICE_MODEL=""
DEVICE_DESC=""
PRESENTATION_URL=""
FORCE=false

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Associates a player with a B-Deploy setup record.

REQUIRED OPTIONS:
    -s, --serial SERIAL         Player serial number
    -i, --setup-id SETUP_ID     Setup record ID
    -n, --network NETWORK       Network name

OPTIONAL:
    -u, --username USERNAME     Username for device record
    --name NAME                 Human-readable device name
    --model MODEL               Player model (e.g., HD1024, XT1143)
    --desc DESCRIPTION          Device description
    --url URL                   Direct presentation URL
    -f, --force                 Skip confirmation when creating new device
    -h, --help                  Display this help message

ENVIRONMENT VARIABLES:
    BS_ACCESS_TOKEN            OAuth2 access token (required)

EXAMPLES:
    # Associate existing player
    $0 --serial "ABC123" --setup-id "setup789" --network "Production"

    # Associate with custom name
    $0 -s "ABC123" -i "setup789" -n "Production" --name "Lobby Display"

    # Create new device with full details
    $0 -s "XYZ456" -i "setup789" -n "Production" \\
       --name "Conference Room" --model "XT1144" --force

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
        -i|--setup-id)
            SETUP_ID="$2"
            shift 2
            ;;
        -n|--network)
            NETWORK="$2"
            shift 2
            ;;
        -u|--username)
            USERNAME="$2"
            shift 2
            ;;
        --name)
            DEVICE_NAME="$2"
            shift 2
            ;;
        --model)
            DEVICE_MODEL="$2"
            shift 2
            ;;
        --desc)
            DEVICE_DESC="$2"
            shift 2
            ;;
        --url)
            PRESENTATION_URL="$2"
            shift 2
            ;;
        -f|--force)
            FORCE=true
            shift
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

if [ -z "$SETUP_ID" ]; then
    echo "Error: --setup-id is required"
    echo ""
    usage
fi

if [ -z "$NETWORK" ]; then
    echo "Error: --network is required"
    echo ""
    usage
fi

# Set defaults for optional fields if not specified
USERNAME=${USERNAME:-"user@example.com"}
DEVICE_NAME=${DEVICE_NAME:-"Player $SERIAL"}
DEVICE_DESC=${DEVICE_DESC:-"Player associated with setup $SETUP_ID"}

# Check if device already exists
echo "🔍 Checking if player $SERIAL is already registered..."
EXISTING_DEVICE=$(curl -s -X GET \
  "https://provision.bsn.cloud/rest-device/v2/device/" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN")

if [ $? -ne 0 ]; then
    echo "❌ Error: Failed to check existing devices"
    exit 1
fi

# Parse response to find device by serial
DEVICE_ID=$(echo "$EXISTING_DEVICE" | jq -r --arg serial "$SERIAL" '.players[]? | select(.serial == $serial) | ._id // empty')

if [ -n "$DEVICE_ID" ] && [ "$DEVICE_ID" != "null" ]; then
    echo "✅ Player $SERIAL found with device ID: $DEVICE_ID"
    echo "📝 Updating existing device record to associate with setup $SETUP_ID..."
    echo

    # Get current device details to preserve existing fields
    CURRENT_DEVICE=$(echo "$EXISTING_DEVICE" | jq -r --arg serial "$SERIAL" '.players[] | select(.serial == $serial)')

    # Extract existing fields
    EXISTING_NAME=$(echo "$CURRENT_DEVICE" | jq -r '.name // empty')
    EXISTING_MODEL=$(echo "$CURRENT_DEVICE" | jq -r '.model // empty')
    EXISTING_DESC=$(echo "$CURRENT_DEVICE" | jq -r '.desc // empty')
    EXISTING_USERDATA=$(echo "$CURRENT_DEVICE" | jq -r '.userdata // empty')
    EXISTING_URL=$(echo "$CURRENT_DEVICE" | jq -r '.url // empty')
    EXISTING_USERNAME=$(echo "$CURRENT_DEVICE" | jq -r '.username // empty')

    # Use existing values if new ones weren't explicitly provided via CLI
    [ "$DEVICE_NAME" = "Player $SERIAL" ] && DEVICE_NAME=${EXISTING_NAME:-"Player $SERIAL"}
    [ -z "$DEVICE_MODEL" ] && DEVICE_MODEL=$EXISTING_MODEL
    [ "$DEVICE_DESC" = "Player associated with setup $SETUP_ID" ] && DEVICE_DESC=${EXISTING_DESC:-"Player associated with setup $SETUP_ID"}
    [ "$USERNAME" = "user@example.com" ] && USERNAME=${EXISTING_USERNAME:-"user@example.com"}
    [ -z "$PRESENTATION_URL" ] && PRESENTATION_URL=$EXISTING_URL

    # Build JSON payload for update
    JSON_PAYLOAD=$(cat <<EOF
{
  "_id": "$DEVICE_ID",
  "username": "$USERNAME",
  "serial": "$SERIAL",
  "name": "$DEVICE_NAME",
  "NetworkName": "$NETWORK",
  "setupId": "$SETUP_ID"
EOF

    # Add optional fields if they exist
    if [ -n "$DEVICE_MODEL" ] && [ "$DEVICE_MODEL" != "null" ]; then
        JSON_PAYLOAD+=",
  \"model\": \"$DEVICE_MODEL\""
    fi

    if [ -n "$DEVICE_DESC" ] && [ "$DEVICE_DESC" != "null" ]; then
        JSON_PAYLOAD+=",
  \"desc\": \"$DEVICE_DESC\""
    fi

    if [ -n "$EXISTING_USERDATA" ] && [ "$EXISTING_USERDATA" != "null" ]; then
        JSON_PAYLOAD+=",
  \"userdata\": \"$EXISTING_USERDATA\""
    fi

    if [ -n "$PRESENTATION_URL" ] && [ "$PRESENTATION_URL" != "null" ]; then
        JSON_PAYLOAD+=",
  \"url\": \"$PRESENTATION_URL\""
    fi

    JSON_PAYLOAD+="
}"

    # Update existing device
    RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X PUT \
      "https://provision.bsn.cloud/rest-device/v2/device/?_id=$DEVICE_ID" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
      -d "$JSON_PAYLOAD")

    HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
    BODY=$(echo "$RESPONSE" | sed '/HTTP_CODE:/d')

    if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
        echo "✅ Player $SERIAL updated with setup ID: $SETUP_ID"
    else
        echo "❌ Failed to update device (HTTP $HTTP_CODE)"
        echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
        exit 1
    fi

else
    echo "⚠️  Player $SERIAL not found. Creating new device record..."
    echo

    # Confirm creation unless --force is used
    if [ "$FORCE" != "true" ]; then
        echo "This will create a new device record and associate it with setup $SETUP_ID"
        echo "Player Serial: $SERIAL"
        echo "Setup ID:      $SETUP_ID"
        echo "Network:       $NETWORK"
        echo "Device Name:   $DEVICE_NAME"
        echo
        read -p "Continue with device creation? (y/N): " confirm

        if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
            echo "❌ Operation cancelled."
            exit 0
        fi
    fi

    # Build JSON payload for new device
    JSON_PAYLOAD=$(cat <<EOF
{
  "username": "$USERNAME",
  "serial": "$SERIAL",
  "name": "$DEVICE_NAME",
  "NetworkName": "$NETWORK",
  "setupId": "$SETUP_ID"
EOF

    # Add optional fields if provided
    if [ -n "$DEVICE_MODEL" ]; then
        JSON_PAYLOAD+=",
  \"model\": \"$DEVICE_MODEL\""
    fi

    if [ -n "$DEVICE_DESC" ]; then
        JSON_PAYLOAD+=",
  \"desc\": \"$DEVICE_DESC\""
    fi

    if [ -n "$PRESENTATION_URL" ]; then
        JSON_PAYLOAD+=",
  \"url\": \"$PRESENTATION_URL\""
    fi

    JSON_PAYLOAD+="
}"

    # Create new device
    RESPONSE=$(curl -s -X POST \
      "https://provision.bsn.cloud/rest-device/v2/device/" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
      -d "$JSON_PAYLOAD")

    # Extract new device ID
    NEW_DEVICE_ID=$(echo "$RESPONSE" | jq -r '._id // empty')

    if [ -n "$NEW_DEVICE_ID" ] && [ "$NEW_DEVICE_ID" != "null" ]; then
        echo
        echo "✅ Successfully created new device record!"
        echo "   Player Serial: $SERIAL"
        echo "   Device ID:     $NEW_DEVICE_ID"
        echo "   Setup ID:      $SETUP_ID"
    else
        echo
        echo "❌ Error: Failed to create device record"
        echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
        exit 1
    fi
fi

echo
echo "✅ Association completed successfully!"
echo "   Player $SERIAL is now associated with setup record $SETUP_ID"
