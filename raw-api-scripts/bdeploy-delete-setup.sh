#!/bin/bash

# Delete B-Deploy Setup Record
# Deletes a specific setup record using the B-Deploy Setup Endpoints (v3)
# Base URL: https://provision.bsn.cloud/rest-setup/v3/setup/

set -e

# Default values
SETUP_ID=""
NETWORK=""
FORCE=false

# Usage function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Deletes a B-Deploy setup record by setup-id.

REQUIRED OPTIONS:
    -s, --setup-id SETUP_ID     Setup ID to delete
    -n, --network NETWORK       Network name

OPTIONAL:
    -f, --force                 Skip confirmation prompt
    -h, --help                  Display this help message

ENVIRONMENT VARIABLES:
    BS_ACCESS_TOKEN            OAuth2 access token (required)

EXAMPLES:
    # Delete with confirmation
    $0 --setup-id "abc123" --network "Production"

    # Force delete without prompt
    $0 -s "abc123" -n "Production" --force

EOF
    exit 1
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -s|--setup-id)
            SETUP_ID="$2"
            shift 2
            ;;
        -n|--network)
            NETWORK="$2"
            shift 2
            ;;
        -f|--force)
            FORCE=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Error: Unknown option '$1'"
            echo ""
            usage
            ;;
    esac
done

# Check required environment variable
if [ -z "$BS_ACCESS_TOKEN" ]; then
    echo "Error: BS_ACCESS_TOKEN environment variable is required"
    echo "Get a token with: ./scripts/get-token.sh"
    echo ""
    exit 1
fi

# Check required parameters
if [ -z "$SETUP_ID" ]; then
    echo "Error: --setup-id is required"
    echo ""
    usage
fi

if [ -z "$NETWORK" ]; then
    echo "Error: --network is required"
    echo ""
    usage
fi

# Confirm deletion unless --force is used
if [ "$FORCE" != "true" ]; then
    echo "⚠️  WARNING: This will permanently delete the B-Deploy setup record!"
    echo "Network:  $NETWORK"
    echo "Setup ID: $SETUP_ID"
    echo
    read -p "Are you sure you want to delete this record? (y/N): " confirm

    if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
        echo "❌ Operation cancelled."
        exit 0
    fi
fi

# Step 1: Set network context (REQUIRED before any B-Deploy API calls)
echo "🔧 Setting network context to '$NETWORK'..."
NETWORK_RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X PUT \
  "https://api.bsn.cloud/2022/06/REST/Self/Session/Network" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$NETWORK\"}")

NETWORK_HTTP_CODE=$(echo "$NETWORK_RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)

if [ "$NETWORK_HTTP_CODE" != "200" ] && [ "$NETWORK_HTTP_CODE" != "204" ]; then
    echo "❌ Failed to set network context (HTTP $NETWORK_HTTP_CODE)"
    NETWORK_BODY=$(echo "$NETWORK_RESPONSE" | sed '/HTTP_CODE:/d')
    if [ -n "$NETWORK_BODY" ]; then
        echo "$NETWORK_BODY" | jq . 2>/dev/null || echo "$NETWORK_BODY"
    fi
    exit 1
fi
echo "✅ Network context set"
echo

# Step 2: Delete the setup record
echo "🗑️  Deleting B-Deploy setup record..."
echo

RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X DELETE \
  "https://provision.bsn.cloud/rest-setup/v3/setup/?_id=$SETUP_ID" \
  -H "Authorization: Bearer $BS_ACCESS_TOKEN" \
  -H "Content-Type: application/json")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d':' -f2)
BODY=$(echo "$RESPONSE" | sed '/HTTP_CODE:/d')

echo "Response Status: $HTTP_CODE"

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "204" ]; then
    echo "✅ Setup record deleted successfully!"
else
    echo "❌ Failed to delete setup record (HTTP $HTTP_CODE)"
    if [ -n "$BODY" ]; then
        echo "Response:"
        echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
    fi
    exit 1
fi

echo
echo "💡 Note: If players were associated with this setup, those associations"
echo "   may still exist. Use bdeploy-get-record.sh to check player status."
