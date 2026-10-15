#!/bin/bash

# Associate Player to Setup
# Associates a BrightSign player (by serial number) with a B-Deploy setup record
# Uses B-Deploy Device Endpoints (v2)
# Base URL: https://provision.bsn.cloud/rest-device/v2/device/

set -e

# Default values
SERIAL=""
SETUP_ID=""
NETWORK=""
PLAYER_NAME=""
PLAYER_DESC=""
PLAYER_MODEL=""

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Associates a BrightSign player with a B-Deploy setup record.

REQUIRED OPTIONS:
    -s, --serial SERIAL         Player serial number (e.g., ABCD00000001)
    -i, --setup-id SETUP_ID     Setup record ID to associate with
    -n, --network NETWORK       Network name

OPTIONAL:
    -N, --name NAME             Player name (defaults to serial number)
    -d, --desc DESCRIPTION      Player description
    -m, --model MODEL           Player model (e.g., HD1024, XT1144)
    -h, --help                  Display this help message

ENVIRONMENT VARIABLES:
    BS_ACCESS_TOKEN            OAuth2 access token (required)
    BS_USERNAME                BSN.cloud username (optional, will use token's user)

EXAMPLES:
    # Basic association
    $0 --serial "ABCD00000001" --setup-id "618fb7363a682fe7a40c73ca" --network "Production"

    # With player details
    $0 -s "ABCD00000001" -i "618fb7363a682fe7a40c73ca" -n "Production" \\
       -N "Store Display #1" -d "Main entrance display" -m "HD1024"

WORKFLOW:
    1. Sets network context
    2. Checks if player exists by serial number
    3. If exists, updates with setupId
    4. If not exists, creates new device record with setupId

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
        -N|--name)
            PLAYER_NAME="$2"
            shift 2
            ;;
        -d|--desc)
            PLAYER_DESC="$2"
            shift 2
            ;;
        -m|--model)
            PLAYER_MODEL="$2"
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
    echo "❌ Error: BS_ACCESS_TOKEN environment variable is required"
    echo "   Get a token with: ./scripts/get-token.sh"
    echo ""
    exit 1
fi

# Check required parameters
if [ -z "$SERIAL" ]; then
    echo "❌ Error: --serial is required"
    echo ""
    usage
fi

if [ -z "$SETUP_ID" ]; then
    echo "❌ Error: --setup-id is required"
    echo ""
    usage
fi

if [ -z "$NETWORK" ]; then
    echo "❌ Error: --network is required"
    echo ""
    usage
fi

# Set default player name if not provided
if [ -z "$PLAYER_NAME" ]; then
    PLAYER_NAME="$SERIAL"
fi

echo "🔗 Associating Player to Setup"
echo "   Serial:   $SERIAL"
echo "   Setup ID: $SETUP_ID"
echo "   Network:  $NETWORK"
if [ -n "$PLAYER_NAME" ]; then
    echo "   Name:     $PLAYER_NAME"
fi
echo ""

# Step 1: Set network context (REQUIRED)
echo "🔧 Setting network context to '$NETWORK'..."
NETWORK_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X PUT \
  "https://api.bsn.cloud/2022/06/REST/Self/Session/Network" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$NETWORK\"}")

NETWORK_HTTP_CODE=$(echo "$NETWORK_RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)

if [ "$NETWORK_HTTP_CODE" != "200" ] && [ "$NETWORK_HTTP_CODE" != "204" ]; then
    echo "❌ Failed to set network context (HTTP $NETWORK_HTTP_CODE)"
    NETWORK_BODY=$(echo "$NETWORK_RESPONSE" | sed '/HTTP_CODE:/d')
    if [ -n "$NETWORK_BODY" ]; then
        echo "$NETWORK_BODY" | jq . 2>/dev/null || echo "$NETWORK_BODY"
    fi
    exit 1
fi
echo "✅ Network context set"
echo ""

# Step 2: Check if device exists
echo "🔍 Checking if device exists with serial '$SERIAL'..."
DEVICE_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X GET \
  "https://provision.bsn.cloud/rest-device/v2/device/?serial=$SERIAL" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN")

DEVICE_HTTP_CODE=$(echo "$DEVICE_RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
DEVICE_BODY=$(echo "$DEVICE_RESPONSE" | sed '/HTTP_CODE:/d')

if [ "$DEVICE_HTTP_CODE" != "200" ]; then
    echo "❌ Failed to check device (HTTP $DEVICE_HTTP_CODE)"
    echo "$DEVICE_BODY" | jq . 2>/dev/null || echo "$DEVICE_BODY"
    exit 1
fi

# Parse response to check if device exists
DEVICE_COUNT=$(echo "$DEVICE_BODY" | jq -r '.result.total // 0')
EXISTING_DEVICE_ID=$(echo "$DEVICE_BODY" | jq -r '.result.players[0]._id // ""')
EXISTING_NAME=$(echo "$DEVICE_BODY" | jq -r '.result.players[0].name // ""')
EXISTING_MODEL=$(echo "$DEVICE_BODY" | jq -r '.result.players[0].model // ""')
EXISTING_DESC=$(echo "$DEVICE_BODY" | jq -r '.result.players[0].desc // ""')
EXISTING_SETUP_ID=$(echo "$DEVICE_BODY" | jq -r '.result.players[0].setupId // ""')

if [ "$DEVICE_COUNT" -gt 0 ] && [ -n "$EXISTING_DEVICE_ID" ]; then
    echo "✅ Device found (ID: $EXISTING_DEVICE_ID)"

    # Check if already associated with this setup
    if [ "$EXISTING_SETUP_ID" = "$SETUP_ID" ]; then
        echo "✅ Device is already associated with setup ID: $SETUP_ID"
        echo ""
        echo "💡 No changes needed - association already exists"
        exit 0
    fi

    if [ -n "$EXISTING_SETUP_ID" ]; then
        echo "⚠️  Device currently associated with different setup: $EXISTING_SETUP_ID"
        echo "   Will update to new setup: $SETUP_ID"
    else
        echo "   Device not currently associated with any setup"
    fi

    # Use existing values if new ones not provided
    if [ -z "$PLAYER_NAME" ]; then
        PLAYER_NAME="$EXISTING_NAME"
    fi
    if [ -z "$PLAYER_MODEL" ]; then
        PLAYER_MODEL="$EXISTING_MODEL"
    fi
    if [ -z "$PLAYER_DESC" ]; then
        PLAYER_DESC="$EXISTING_DESC"
    fi

    echo ""
    echo "🔄 Updating device with setup association..."

    # Build update payload
    UPDATE_PAYLOAD=$(jq -n \
      --arg id "$EXISTING_DEVICE_ID" \
      --arg serial "$SERIAL" \
      --arg name "$PLAYER_NAME" \
      --arg network "$NETWORK" \
      --arg setupId "$SETUP_ID" \
      --arg model "$PLAYER_MODEL" \
      --arg desc "$PLAYER_DESC" \
      '{
        "_id": $id,
        "serial": $serial,
        "name": $name,
        "NetworkName": $network,
        "setupId": $setupId,
        "model": $model,
        "desc": $desc
      }')

    UPDATE_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X PUT \
      "https://provision.bsn.cloud/rest-device/v2/device/?_id=$EXISTING_DEVICE_ID" \
      -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
      -H "Content-Type: application/json" \
      -d "$UPDATE_PAYLOAD")

    UPDATE_HTTP_CODE=$(echo "$UPDATE_RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
    UPDATE_BODY=$(echo "$UPDATE_RESPONSE" | sed '/HTTP_CODE:/d')

    if [ "$UPDATE_HTTP_CODE" = "200" ] || [ "$UPDATE_HTTP_CODE" = "204" ]; then
        echo "✅ Device updated successfully!"
        echo ""
        echo "📝 Association Details:"
        echo "   Player Serial: $SERIAL"
        echo "   Player Name:   $PLAYER_NAME"
        echo "   Setup ID:      $SETUP_ID"
        echo "   Network:       $NETWORK"
    else
        echo "❌ Failed to update device (HTTP $UPDATE_HTTP_CODE)"
        echo "$UPDATE_BODY" | jq . 2>/dev/null || echo "$UPDATE_BODY"
        exit 1
    fi
else
    echo "⚠️  Device not found - creating new device record..."
    echo ""

    # Build create payload
    CREATE_PAYLOAD=$(jq -n \
      --arg serial "$SERIAL" \
      --arg name "$PLAYER_NAME" \
      --arg network "$NETWORK" \
      --arg setupId "$SETUP_ID" \
      --arg model "$PLAYER_MODEL" \
      --arg desc "$PLAYER_DESC" \
      --arg username "${BS_USERNAME:-admin}" \
      '{
        "serial": $serial,
        "name": $name,
        "NetworkName": $network,
        "setupId": $setupId,
        "model": $model,
        "desc": $desc,
        "username": $username
      }')

    CREATE_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST \
      "https://provision.bsn.cloud/rest-device/v2/device/" \
      -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
      -H "Content-Type: application/json" \
      -d "$CREATE_PAYLOAD")

    CREATE_HTTP_CODE=$(echo "$CREATE_RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
    CREATE_BODY=$(echo "$CREATE_RESPONSE" | sed '/HTTP_CODE:/d')

    if [ "$CREATE_HTTP_CODE" = "201" ] || [ "$CREATE_HTTP_CODE" = "200" ]; then
        NEW_DEVICE_ID=$(echo "$CREATE_BODY" | jq -r '._id // ""')
        echo "✅ Device created and associated successfully!"
        echo ""
        echo "📝 Association Details:"
        echo "   Player Serial: $SERIAL"
        echo "   Player Name:   $PLAYER_NAME"
        echo "   Device ID:     $NEW_DEVICE_ID"
        echo "   Setup ID:      $SETUP_ID"
        echo "   Network:       $NETWORK"
    else
        echo "❌ Failed to create device (HTTP $CREATE_HTTP_CODE)"
        echo "$CREATE_BODY" | jq . 2>/dev/null || echo "$CREATE_BODY"
        exit 1
    fi
fi

echo ""
echo "💡 Next Steps:"
echo "   1. Power on the player with serial '$SERIAL'"
echo "   2. Ensure it has network connectivity"
echo "   3. Player will auto-provision using the associated setup"
echo "   4. Monitor with: ./scripts/bdeploy-list-devices.sh -n '$NETWORK'"
