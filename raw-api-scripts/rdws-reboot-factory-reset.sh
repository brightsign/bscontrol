#!/bin/bash

# Factory reset BrightSign device via rDWS API
# Usage: ./rdws-reboot-factory-reset.sh <device-serial>

if [ $# -ne 1 ]; then
    echo "Usage: $0 <device-serial>"
    echo "Example: $0 UTD41X000009"
    exit 1
fi

DEVICE_SERIAL=$1

# Check if BS_ACCESS_TOKEN is set
if [ -z "$BS_ACCESS_TOKEN" ]; then
    echo "Error: BS_ACCESS_TOKEN environment variable is not set"
    echo "Run: source ../.token"
    exit 1
fi

# Confirmation prompt
echo "⚠️  WARNING: This will FACTORY RESET device: $DEVICE_SERIAL"
echo "This will erase ALL settings, networking, security, and application data!"
read -p "Are you sure? Type 'yes' to continue: " confirmation

if [ "$confirmation" != "yes" ]; then
    echo "Factory reset cancelled."
    exit 0
fi

echo "Performing factory reset on device: $DEVICE_SERIAL"

curl -X PUT "https://ws.bsn.cloud/rest/v1/control/reboot/?destinationType=player&destinationName=${DEVICE_SERIAL}" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H 'Accept: application/json, application/vnd.bsn.error+json' \
  -H 'Content-Type: application/json' \
  -d '{
    "data": {
        "factory_reset": true
    }
}'

echo ""
echo "Factory reset command sent. The device will restart and reset to factory defaults."