#!/bin/bash

# Get BSN.cloud OAuth2 access token using client credentials
# Usage: ./bsn-get-token.sh

# Check if credentials are set
if [ -z "$BS_CLIENT_ID" ]; then
    echo "Error: BS_CLIENT_ID environment variable is not set"
    exit 1
fi

if [ -z "$BS_SECRET" ]; then
    echo "Error: BS_SECRET environment variable is not set"
    exit 1
fi

echo "Getting BSN.cloud access token..."

# Get the token
response=$(curl -s -X POST 'https://auth.bsn.cloud/realms/bsncloud/protocol/openid-connect/token' \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'grant_type=client_credentials' \
  -u "$BS_CLIENT_ID:$BS_SECRET")

# Extract the access_token
access_token=$(echo "$response" | grep -o '"access_token":"[^"]*' | sed 's/"access_token":"//')

if [ -z "$access_token" ]; then
    echo "Failed to get access token. Response:"
    echo "$response"
    exit 1
fi

# Save to file
echo "export BS_ACCESS_TOKEN=$access_token" > ../.token

echo "✅ Token saved to ../.token"
echo ""
echo "To use the token:"
echo "  source ../.token"
echo "  echo \$BS_ACCESS_TOKEN"
echo ""
echo "Export statement:"
echo "export BS_ACCESS_TOKEN=$access_token"