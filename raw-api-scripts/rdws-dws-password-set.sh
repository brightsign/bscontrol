#!/bin/bash

# Set DWS password on BrightSign device via rDWS API
# Usage: ./rdws-dws-password-set.sh <device-serial> <new-password> [old-password]

if [ $# -lt 2 ] || [ $# -gt 3 ]; then
    echo "Usage: $0 <device-serial> <new-password> [old-password]"
    echo "Example: $0 UTD41X000009 newpass123"
    echo "Example: $0 UTD41X000009 newpass123 oldpass456"
    echo ""
    echo "Note: Use empty string for new-password to remove password protection"
    echo "      Old password can be omitted if there's no current password"
    exit 1
fi

DEVICE_SERIAL=$1
NEW_PASSWORD=$2
OLD_PASSWORD=${3:-""}

# Check if BS_ACCESS_TOKEN is set
if [ -z "$BS_ACCESS_TOKEN" ]; then
    echo "Error: BS_ACCESS_TOKEN environment variable is not set"
    echo "Run: source ../.token"
    exit 1
fi

if [ -z "$NEW_PASSWORD" ]; then
    echo "Setting DWS to no password (removing protection) on device: $DEVICE_SERIAL"
else
    echo "Setting DWS password on device: $DEVICE_SERIAL"
fi

curl -X PUT "https://ws.bsn.cloud/rest/v1/control/dws-password/?destinationType=player&destinationName=${DEVICE_SERIAL}" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H 'Accept: application/json, application/vnd.bsn.error+json' \
  -H 'Content-Type: application/json' \
  -d "{
    \"data\": {
        \"password\": \"${NEW_PASSWORD}\",
        \"previous_password\": \"${OLD_PASSWORD}\"
    }
}"

echo ""
echo "DWS password command sent. The device may reboot to apply this change."