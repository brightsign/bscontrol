#!/bin/bash

# Capture screenshot from BrightSign device via rDWS API
# Usage: ./rdws-snapshot.sh <device-serial> [format] [quality] [output-file]

if [ $# -lt 1 ] || [ $# -gt 4 ]; then
    echo "Usage: $0 <device-serial> [format] [quality] [output-file]"
    echo "Example: $0 UTD41X000009"
    echo "Example: $0 UTD41X000009 png"
    echo "Example: $0 UTD41X000009 jpeg 85"
    echo "Example: $0 UTD41X000009 png 90 screenshot.png"
    echo ""
    echo "Parameters:"
    echo "  device-serial  - BrightSign device serial number"
    echo "  format         - Image format: png (default) or jpeg"
    echo "  quality        - JPEG quality 1-100 (default: 90)"
    echo "  output-file    - Output filename (default: snapshot-<serial>-<timestamp>.format)"
    exit 1
fi

DEVICE_SERIAL=$1
FORMAT=${2:-"png"}
QUALITY=${3:-90}
OUTPUT_FILE=$4

# Check if BS_ACCESS_TOKEN is set
if [ -z "$BS_ACCESS_TOKEN" ]; then
    echo "Error: BS_ACCESS_TOKEN environment variable is not set"
    echo "Run: source ../.token"
    exit 1
fi

# Validate format
if [ "$FORMAT" != "png" ] && [ "$FORMAT" != "jpeg" ]; then
    echo "Error: Format must be 'png' or 'jpeg'"
    exit 1
fi

# Validate quality
if [ "$QUALITY" -lt 1 ] || [ "$QUALITY" -gt 100 ]; then
    echo "Error: Quality must be between 1 and 100"
    exit 1
fi

# Generate output filename if not provided
if [ -z "$OUTPUT_FILE" ]; then
    TIMESTAMP=$(date +"%Y%m%d-%H%M%S")
    OUTPUT_FILE="snapshot-${DEVICE_SERIAL}-${TIMESTAMP}.${FORMAT}"
fi

echo "Capturing screenshot from device: $DEVICE_SERIAL"
echo "Format: $FORMAT, Quality: $QUALITY"
echo "Output: $OUTPUT_FILE"

# Make the API request and save response
RESPONSE=$(curl -s -X POST "https://ws.bsn.cloud/rest/v1/snapshot/?destinationType=player&destinationName=${DEVICE_SERIAL}" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H 'Accept: application/json' \
  -H 'Content-Type: application/json' \
  -d "{
    \"data\": {
      \"format\": \"${FORMAT}\",
      \"quality\": ${QUALITY},
      \"includeMetadata\": true,
      \"output\": \"base64\",
      \"compression\": \"medium\",
      \"displayId\": 0
    }
  }")

# Check if response contains error
if echo "$RESPONSE" | grep -q '"error"'; then
    echo "Error: API request failed"
    echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
    exit 1
fi

# Check if response contains base64 data
BASE64_DATA=$(echo "$RESPONSE" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    if 'data' in data and 'data' in data['data']:
        print(data['data']['data'])
    elif 'data' in data:
        print(data['data'])
    else:
        print('')
except:
    print('')
" 2>/dev/null)

if [ -z "$BASE64_DATA" ]; then
    echo "No base64 image data found in response"
    echo "Full response:"
    echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
    exit 1
fi

# Decode base64 and save to file
echo "$BASE64_DATA" | base64 -d > "$OUTPUT_FILE"

if [ $? -eq 0 ] && [ -f "$OUTPUT_FILE" ]; then
    FILE_SIZE=$(stat -c%s "$OUTPUT_FILE" 2>/dev/null || stat -f%z "$OUTPUT_FILE" 2>/dev/null || echo "unknown")
    echo "✅ Screenshot saved to: $OUTPUT_FILE"
    echo "📊 File size: $FILE_SIZE bytes"
    
    # Try to show image info if 'file' command is available
    if command -v file >/dev/null 2>&1; then
        echo "📄 File type: $(file "$OUTPUT_FILE")"
    fi
else
    echo "❌ Failed to save screenshot"
    exit 1
fi