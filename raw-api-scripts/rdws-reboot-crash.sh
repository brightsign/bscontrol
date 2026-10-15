#!/bin/bash

# Reboot BrightSign device with crash report via rDWS API
# Usage: ./rdws-reboot-crash.sh <device-serial>

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

echo "Rebooting device with crash report: $DEVICE_SERIAL"

curl -X PUT "https://ws.bsn.cloud/rest/v1/control/reboot/?destinationType=player&destinationName=${DEVICE_SERIAL}" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H 'Accept: application/json, application/vnd.bsn.error+json' \
  -H 'Content-Type: application/json' \
  -d '{
    "data": {
        "crash_report": true
    }
}'

echo ""
echo "Reboot with crash report command sent. The device will restart and save crash diagnostics."