#!/bin/bash

# Re-provision BrightSign device via rDWS API
# Usage: ./rdws-reprovision.sh <device-serial> [skip-confirmation]

if [ $# -lt 1 ] || [ $# -gt 2 ]; then
    echo "Usage: $0 <device-serial> [skip-confirmation]"
    echo "Example: $0 UTD41X000009"
    echo "Example: $0 UTD41X000009 -y"
    echo ""
    echo "⚠️  WARNING: Re-provisioning will:"
    echo "  • Reset the device to factory defaults"
    echo "  • Clear all content and settings"
    echo "  • Reboot the device and run B-Deploy setup again"
    echo "  • This action is IRREVERSIBLE"
    exit 1
fi

DEVICE_SERIAL=$1
SKIP_CONFIRMATION=${2:-""}

# Check if BS_ACCESS_TOKEN is set
if [ -z "$BS_ACCESS_TOKEN" ]; then
    echo "Error: BS_ACCESS_TOKEN environment variable is not set"
    echo "Run: source ../.token"
    exit 1
fi

# Confirmation prompt unless -y flag is used
if [ "$SKIP_CONFIRMATION" != "-y" ]; then
    echo "🚨 CRITICAL WARNING: Re-provisioning device: $DEVICE_SERIAL"
    echo ""
    echo "This operation will:"
    echo "  ❌ PERMANENTLY DELETE all content and settings"
    echo "  ❌ Reset device to factory defaults"
    echo "  ❌ Clear all networking configuration (except essential setup keys)"
    echo "  🔄 Reboot and restart B-Deploy provisioning process"
    echo "  ⏱️  Device will be offline during re-provisioning"
    echo ""
    echo "📋 Re-provisioning process:"
    echo "  1. Clear device registry (keeping essential network keys)"
    echo "  2. Format storage device (SD card)"
    echo "  3. Reboot device"
    echo "  4. Device fetches setup package from B-Deploy"
    echo "  5. Device provisions itself with new configuration"
    echo ""
    echo "⚠️  THIS ACTION CANNOT BE UNDONE!"
    echo ""
    read -p "Are you absolutely sure? Type 'yes' to continue: " confirmation
    
    if [ "$confirmation" != "yes" ]; then
        echo "❌ Re-provisioning cancelled. Only 'yes' confirms this action."
        exit 0
    fi
fi

echo "🔄 Initiating re-provisioning for device: $DEVICE_SERIAL"

# Make the API request
RESPONSE=$(curl -s -X GET "https://ws.bsn.cloud/rest/v1/re-provision/?destinationType=player&destinationName=${DEVICE_SERIAL}" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H 'Accept: application/json')

# Check if response contains error
if echo "$RESPONSE" | grep -q '"error"'; then
    echo "❌ Error: API request failed"
    echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
    exit 1
fi

# Check if response indicates success
SUCCESS=$(echo "$RESPONSE" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    if 'data' in data and 'result' in data['data'] and 'success' in data['data']['result']:
        print('true' if data['data']['result']['success'] else 'false')
    else:
        print('unknown')
except:
    print('error')
" 2>/dev/null)

if [ "$SUCCESS" = "true" ]; then
    echo "✅ Re-provisioning initiated successfully!"
    echo "🔄 Device $DEVICE_SERIAL is now rebooting and will re-provision itself"
    echo ""
    echo "📋 Next steps:"
    echo "  1. Device is rebooting (this may take 1-2 minutes)"
    echo "  2. Device will fetch B-Deploy setup package"
    echo "  3. Device will run through provisioning setup"
    echo "  4. Monitor device status in BSN.cloud dashboard"
    echo ""
    echo "💡 The device will appear offline until re-provisioning completes."
elif [ "$SUCCESS" = "false" ]; then
    echo "❌ Re-provisioning request failed"
    echo "Response:"
    echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
    exit 1
else
    echo "❓ Unexpected response format:"
    echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
    exit 1
fi