#!/bin/bash

# Get DWS password status on BrightSign device via rDWS API
# Usage: ./rdws-dws-password-get.sh <device-serial>

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

echo "Getting DWS password status for device: $DEVICE_SERIAL"

curl -X GET "https://ws.bsn.cloud/rest/v1/control/dws-password/?destinationType=player&destinationName=${DEVICE_SERIAL}" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H 'Accept: application/json, application/vnd.bsn.error+json'

echo ""
echo "Note: This returns password status (blank/valid), not the actual password."