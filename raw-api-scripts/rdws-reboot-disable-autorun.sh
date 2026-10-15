#!/bin/bash

# Reboot BrightSign device with autorun disabled via rDWS API
# Usage: ./rdws-reboot-disable-autorun.sh <device-serial>

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

echo "Rebooting device with autorun disabled: $DEVICE_SERIAL"

curl -X PUT "https://ws.bsn.cloud/rest/v1/control/reboot/?destinationType=player&destinationName=${DEVICE_SERIAL}" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H 'Accept: application/json, application/vnd.bsn.error+json' \
  -H 'Content-Type: application/json' \
  -d '{
    "data": {
        "autorun": "disable"
    }
}'

echo ""
echo "Reboot with autorun disabled command sent. The device will restart without running autorun script."