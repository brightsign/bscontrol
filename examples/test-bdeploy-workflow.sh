#!/bin/bash

# Test workflow for B-Deploy operations
# This demonstrates the typical workflow: list -> add -> delete

set -e

if [ -z "$BS_CLIENT_ID" ] || [ -z "$BS_SECRET" ] || [ -z "$BS_NETWORK" ]; then
    echo "Please set BS_CLIENT_ID, BS_SECRET, and BS_NETWORK environment variables"
    exit 1
fi

echo "🧪 Testing B-Deploy workflow..."
echo "Network: $BS_NETWORK"
echo

# Step 1: List existing records to see what's available
echo "📋 1. Listing existing B-Deploy records..."
./bin/bdeploy-get-records --network "$BS_NETWORK"
echo

# Step 2: Add a test record (uncomment to test)
# echo "➕ 2. Adding a test record..."
#   --network "$BS_NETWORK" \
#   --package-name "test-delete-me" \
#   --setup-type "standalone"
# echo

# Step 3: List again to see the new record and get its ID
# echo "📋 3. Listing records again to see the new record..."
# ./bin/bdeploy-get-records --network "$BS_NETWORK"
# echo

# Step 4: Delete the test record (replace with actual ID)
# echo "🗑️  4. Deleting the test record..."
# echo "To delete a record, run:"
# echo "./bin/bsn-bdeploy-setup-delete --network \"$BS_NETWORK\" --setup-id \"<RECORD_ID_FROM_STEP_3>\""

echo "✅ Test completed!"
echo
echo "💡 To test deletion:"
echo "1. Look at the record IDs from the list above"
echo "2. Run: ./bin/bsn-bdeploy-setup-delete --network \"$BS_NETWORK\" --setup-id \"<VALID_ID>\""
echo "3. Use --force to skip confirmation: ./bin/bsn-bdeploy-setup-delete --network \"$BS_NETWORK\" --setup-id \"<VALID_ID>\" --force"