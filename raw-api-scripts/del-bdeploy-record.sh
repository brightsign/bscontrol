#!/bin/bash

# Disassociate Player from B-Deploy Setup Record
# Removes the setupId field from a device record, effectively disassociating
# the player from its setup configuration while keeping the device registered
# Base URL: https://provision.bsn.cloud/rest-device/v2/device/

# Required environment variables:
# BS_NETWORK - The BSN.cloud network name
# BS_ACCESS_TOKEN - OAuth2 access token
# BS_DEVICE_ID - The device database ID to update

# Optional environment variables:
# BS_USERNAME - Username for the device record
# BS_DIRECT_URL - Direct presentation URL (if switching to direct provisioning)

# Check required environment variables
if [ -z "$BS_NETWORK" ]; then
    echo "Error: BS_NETWORK environment variable is required"
    exit 1
fi

if [ -z "$BS_ACCESS_TOKEN" ]; then
    echo "Error: BS_ACCESS_TOKEN environment variable is required"
    exit 1
fi

if [ -z "$BS_DEVICE_ID" ]; then
    echo "Error: BS_DEVICE_ID environment variable is required"
    echo "Usage: BS_DEVICE_ID=<device-id> BS_NETWORK=<network> BS_ACCESS_TOKEN=<token> $0"
    exit 1
fi

# Get current device record to preserve existing fields
echo "Fetching current device record..."
CURRENT_DEVICE=$(curl -s -X GET \
  "https://provision.bsn.cloud/rest-device/v2/device/?_id=$BS_DEVICE_ID" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN")

if [ $? -ne 0 ]; then
    echo "Error: Failed to fetch current device record"
    exit 1
fi

# Extract fields from current record
SERIAL=$(echo "$CURRENT_DEVICE" | jq -r '.players[0].serial // empty')
NAME=$(echo "$CURRENT_DEVICE" | jq -r '.players[0].name // empty')
USERNAME=$(echo "$CURRENT_DEVICE" | jq -r '.players[0].username // empty')
MODEL=$(echo "$CURRENT_DEVICE" | jq -r '.players[0].model // empty')
DESC=$(echo "$CURRENT_DEVICE" | jq -r '.players[0].desc // empty')
USERDATA=$(echo "$CURRENT_DEVICE" | jq -r '.players[0].userdata // empty')
CURRENT_URL=$(echo "$CURRENT_DEVICE" | jq -r '.players[0].url // empty')

# Check if device was found
if [ -z "$SERIAL" ] || [ "$SERIAL" = "null" ]; then
    echo "Error: Device with ID $BS_DEVICE_ID not found"
    exit 1
fi

# Use environment variable overrides if provided
USERNAME=${BS_USERNAME:-$USERNAME}
DIRECT_URL=${BS_DIRECT_URL:-$CURRENT_URL}

# Confirm disassociation
echo "WARNING: This will disassociate the player from its setup configuration!"
echo "Device ID: $BS_DEVICE_ID"
echo "Serial: $SERIAL"
echo "Name: $NAME"
echo "Network: $BS_NETWORK"
echo
read -p "Are you sure you want to disassociate this player from its setup? (y/N): " confirm

if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo "Operation cancelled."
    exit 0
fi

# Build JSON payload
JSON_PAYLOAD=$(cat <<EOF
{
  "_id": "$BS_DEVICE_ID",
  "username": "$USERNAME",
  "serial": "$SERIAL",
  "name": "$NAME",
  "NetworkName": "$BS_NETWORK",
  "setupId": null
EOF

# Add optional fields if they exist
if [ -n "$MODEL" ] && [ "$MODEL" != "null" ]; then
    JSON_PAYLOAD+=",
  \"model\": \"$MODEL\""
fi

if [ -n "$DESC" ] && [ "$DESC" != "null" ]; then
    JSON_PAYLOAD+=",
  \"desc\": \"$DESC\""
fi

if [ -n "$USERDATA" ] && [ "$USERDATA" != "null" ]; then
    JSON_PAYLOAD+=",
  \"userdata\": \"$USERDATA\""
fi

if [ -n "$DIRECT_URL" ] && [ "$DIRECT_URL" != "null" ]; then
    JSON_PAYLOAD+=",
  \"url\": \"$DIRECT_URL\""
fi

JSON_PAYLOAD+="
}"

# Make the API request
echo "Disassociating player from setup record..."
echo

curl -X PUT \
  "https://provision.bsn.cloud/rest-device/v2/device/?_id=$BS_DEVICE_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -d "$JSON_PAYLOAD"

echo
echo "Disassociation request completed."
echo "Player $SERIAL is no longer associated with a setup configuration."