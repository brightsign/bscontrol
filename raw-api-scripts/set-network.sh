#!/bin/bash

# Set BSN.cloud Network Context
# Sets the network context for API operations - required before calling B-Deploy APIs
# Base URL: https://api.bsn.cloud/2022/06/REST/Self/Session/Network

# Required environment variables:
# BS_ACCESS_TOKEN - OAuth2 access token
# BS_NETWORK - The BSN.cloud network name

# Check required environment variables
if [ -z "$BS_ACCESS_TOKEN" ]; then
    echo "Error: BS_ACCESS_TOKEN environment variable is required"
    exit 1
fi

if [ -z "$BS_NETWORK" ]; then
    echo "Error: BS_NETWORK environment variable is required"
    exit 1
fi

echo "Setting network context for: $BS_NETWORK"

# Set the network context
curl --location --request PUT 'https://api.bsn.cloud/2022/06/REST/Self/Session/Network' \
  --header 'Content-Type: application/json' \
  --header "Authorization: Bearer $BS_ACCESS_TOKEN" \
  --data "{\"name\": \"$BS_NETWORK\"}"

echo
echo "✅ Network context set successfully"

