#!/bin/bash

# Disable local DWS on BrightSign device via rDWS API
# Usage: ./rdws-local-dws-disable.sh <device-serial>

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

echo "Disabling local DWS on device: $DEVICE_SERIAL"

curl -X PUT "https://ws.bsn.cloud/rest/v1/control/local-dws/?destinationType=player&destinationName=${DEVICE_SERIAL}" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H 'Accept: application/json, application/vnd.bsn.error+json' \
  -H 'Content-Type: application/json' \
  -d '{
    "data": {
        "enable": false
    }
}'

echo ""
echo "Disable local DWS command sent. The device may reboot to apply this change."