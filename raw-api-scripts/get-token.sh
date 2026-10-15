#!/bin/bash

# Get OAuth2 Access Token
# Gets a fresh access token from BSN.cloud and optionally sets network context

# Required environment variables:
# BS_CLIENT_ID - OAuth2 client ID
# BS_SECRET - OAuth2 client secret

# Optional environment variables:
# BS_NETWORK - Network name to set context for

# Check required environment variables
if [ -z "$BS_CLIENT_ID" ]; then
    echo "Error: BS_CLIENT_ID environment variable is required"
    exit 1
fi

if [ -z "$BS_SECRET" ]; then
    echo "Error: BS_SECRET environment variable is required"
    exit 1
fi

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo "Error: jq is required but not installed"
    echo "Install with: apt-get install jq (Debian/Ubuntu) or brew install jq (macOS)"
    exit 1
fi

# Get access token
echo "🔐 Getting access token..."
echo "   Client ID: ${BS_CLIENT_ID:0:10}..."
echo "   Endpoint: https://auth.bsn.cloud/realms/bsncloud/protocol/openid-connect/token"
echo ""

TOKEN_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X POST \
  "https://auth.bsn.cloud/realms/bsncloud/protocol/openid-connect/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials" \
  -u "$BS_CLIENT_ID:$BS_SECRET")

# Extract HTTP status code
HTTP_STATUS=$(echo "$TOKEN_RESPONSE" | grep "HTTP_STATUS:" | cut -d':' -f2)
RESPONSE_BODY=$(echo "$TOKEN_RESPONSE" | sed '/HTTP_STATUS:/d')

echo "📥 Response Status: $HTTP_STATUS"
echo ""

# Check if response is empty
if [ -z "$RESPONSE_BODY" ]; then
    echo "❌ Failed to get access token"
    echo "   Error: Empty response from server"
    echo "   HTTP Status: $HTTP_STATUS"
    exit 1
fi

# Try to parse as JSON
if ! echo "$RESPONSE_BODY" | jq . > /dev/null 2>&1; then
    echo "❌ Failed to get access token"
    echo "   Error: Response is not valid JSON"
    echo "   Raw response:"
    echo "$RESPONSE_BODY"
    exit 1
fi

ACCESS_TOKEN=$(echo "$RESPONSE_BODY" | jq -r '.access_token')
EXPIRES_IN=$(echo "$RESPONSE_BODY" | jq -r '.expires_in')

if [ -z "$ACCESS_TOKEN" ] || [ "$ACCESS_TOKEN" = "null" ]; then
    echo "❌ Failed to get access token"
    echo "   Error response from API:"
    echo "$RESPONSE_BODY" | jq .
    exit 1
fi

echo "✅ Access token obtained"
echo "   Token: ${ACCESS_TOKEN:0:20}..."
echo "   Expires in: $EXPIRES_IN seconds"

# Set network context if specified
if [ -n "$BS_NETWORK" ]; then
    echo ""
    echo "📡 Setting network context to: $BS_NETWORK"

    NETWORK_RESPONSE=$(curl -s -w "\nHTTP_STATUS:%{http_code}" -X PUT \
      "https://api.bsn.cloud/2022/06/REST/Self/Session/Network" \
      -H "Authorization: Bearer $ACCESS_TOKEN" \
      -H "Content-Type: application/json" \
      -d "{\"name\": \"$BS_NETWORK\"}")

    NETWORK_HTTP_STATUS=$(echo "$NETWORK_RESPONSE" | grep "HTTP_STATUS:" | cut -d':' -f2)
    NETWORK_BODY=$(echo "$NETWORK_RESPONSE" | sed '/HTTP_STATUS:/d')

    echo "   Status: $NETWORK_HTTP_STATUS"

    if [ "$NETWORK_HTTP_STATUS" != "200" ] && [ "$NETWORK_HTTP_STATUS" != "204" ]; then
        echo "❌ Failed to set network context (HTTP $NETWORK_HTTP_STATUS)"
        echo "$NETWORK_BODY" | jq . 2>/dev/null || echo "$NETWORK_BODY"
        exit 1
    fi

    echo "✅ Network context set to: $BS_NETWORK"
fi

echo ""
echo "To use this token, export it:"
echo "export BS_ACCESS_TOKEN=\"$ACCESS_TOKEN\""
echo ""
echo "Or run:"
echo "export BS_ACCESS_TOKEN=\$(BS_CLIENT_ID=\"\$BS_CLIENT_ID\" BS_SECRET=\"\$BS_SECRET\" $0 | grep 'export BS_ACCESS_TOKEN' | cut -d'\"' -f2)"
