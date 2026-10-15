#!/bin/bash

# List available BSN.cloud networks
# Usage: ./bsn-list-networks.sh

# Check if BS_ACCESS_TOKEN is set
if [ -z "$BS_ACCESS_TOKEN" ]; then
    echo "Error: BS_ACCESS_TOKEN environment variable is not set"
    echo "Run: source ../.token"
    exit 1
fi

echo "Getting available BSN.cloud networks..."

curl -X GET 'https://api.bsn.cloud/2022/06/REST/Self/Networks' \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H 'Accept: application/json' | python3 -m json.tool

echo ""