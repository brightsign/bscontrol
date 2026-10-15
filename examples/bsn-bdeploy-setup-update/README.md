# Update B-Deploy Setup Example

Updates an existing B-Deploy setup record with new configuration values. This allows you to modify setup configurations without creating new records or reassociating players.

**What This Program Does:**
- Fetches an existing B-Deploy setup record by ID or name
- Applies updates from a JSON configuration file (partial or full updates)
- Updates the setup record via the B-Deploy API
- Preserves existing values for fields not specified in the update config

**What This Program Does NOT Do:**
- Does not create new setup records (use `bdeploy-add-setup` for that)
- Does not automatically update players (players get the new config on next check-in)
- Does not reassociate players with different setups

## Quick Start

```bash
# Build all examples
make build-examples

# Set your credentials
export BS_CLIENT_ID="your-client-id"
export BS_SECRET="your-client-secret"

# Update a setup by ID with partial changes
./bin/bsn-bdeploy-setup-update --setup-id <your-setup-id> examples/bsn-bdeploy-setup-update/example-config.json

# Update a setup by name (requires network context)
export BS_NETWORK="Production"
./bin/bsn-bdeploy-setup-update --setup-name "production-setup" examples/bsn-bdeploy-setup-update/example-config.json

# Or specify network in config file instead
./bin/bsn-bdeploy-setup-update --setup-name "production-setup" config-with-network.json

# Update with full configuration
./bin/bsn-bdeploy-setup-update --setup-id <your-setup-id> examples/bsn-bdeploy-setup-update/example-full-update.json

# Get verbose output
./bin/bsn-bdeploy-setup-update --setup-id <your-setup-id> --verbose examples/bsn-bdeploy-setup-update/example-config.json
```

## Using Setup ID vs Setup Name

You can identify the setup to update using either:

### Option 1: Setup ID (--setup-id)

Use the setup ID if you know it (e.g., from the output of `bdeploy-add-setup` or `bdeploy-list-setups`):

```bash
./bin/bsn-bdeploy-setup-update --setup-id 618fb7363a682fe7a40c73ca config.json
```

**Pros:**
- Direct lookup - faster
- Works without network context

**Cons:**
- Need to know or look up the ID first

### Option 2: Setup Name (--setup-name)

Use the setup name (packageName) for more intuitive identification:

```bash
export BS_NETWORK="Production"
./bin/bsn-bdeploy-setup-update --setup-name "production-setup" config.json
```

**Pros:**
- More human-readable
- Don't need to remember cryptic IDs

**Cons:**
- Requires network context (via `BS_NETWORK` env var or `networkName` in config)
- Searches all setups in the network (slightly slower for large networks)
- If multiple setups have the same name, uses the first match (with a warning)

**Network Context for --setup-name:**

When using `--setup-name`, you must specify which network to search. Two options:

1. **Environment variable** (recommended):
   ```bash
   export BS_NETWORK="Production"
   ./bin/bsn-bdeploy-setup-update --setup-name "production-setup" config.json
   ```

2. **In config file**:
   ```json
   {
     "networkName": "Production",
     "timeZone": "America/Chicago"
   }
   ```
   ```bash
   ./bin/bsn-bdeploy-setup-update --setup-name "production-setup" config.json
   ```

## Configuration Files

### example-config.json
Minimal configuration showing **partial updates**. Only fields specified in this file will be updated - all other fields remain unchanged.

```json
{
  "packageName": "retail-display-v2-updated",
  "bsnGroupName": "Production Displays",
  "timeZone": "America/Los_Angeles"
}
```

This updates only 3 fields - everything else (network interfaces, device tokens, etc.) stays the same.

### example-full-update.json
Complete configuration showing **full setup update**. Updates all major fields including network configuration.

Use this when you need to completely reconfigure a setup.

## Configuration Reference

### Update Behavior

**Partial Updates (Recommended):**
- Specify only the fields you want to change
- Empty/missing fields are **not updated** (existing values preserved)
- Useful for quick changes like updating package name or timezone

**Full Updates:**
- Specify all fields you want in the final configuration
- Empty fields will still preserve existing values
- Useful for major reconfiguration

### Updatable Fields

All fields from the original setup can be updated:

**B-Deploy Configuration:**
- `networkName` - Target BSN.cloud network
- `username` - BSN.cloud username
- `packageName` - Setup package identifier

**Setup Configuration:**
- `setupType` - Type: `standalone`, `bsn`, etc.
- `timeZone` - Player timezone
- `bsnGroupName` - BSN group assignment

**Network Configuration:**
- `network.timeServers` - Array of time server URLs
- `network.interfaces` - Array of network interface configs

**Note:** The device registration token is preserved and cannot be updated via this method. Generate a new token and update separately if needed.

## Workflow Steps

The program automatically performs these steps:

1. **Load Update Config** - Parse JSON with fields to update
2. **Authenticate** - OAuth2 with BSN.cloud
3. **Fetch Existing Setup** - Get current setup record by ID
4. **Set Network Context** - Configure B-Deploy for target network
5. **Apply Updates** - Merge update fields with existing record
6. **Update Setup Record** - Save updated configuration via API
7. **Display Results** - Show the updated setup details

## Usage Examples

### Update Package Name Only

```bash
# Create minimal config
cat > update-package.json << 'EOF'
{
  "packageName": "retail-display-v3"
}
EOF

# Apply update
./bin/bsn-bdeploy-setup-update --setup-id 618fb7363a682fe7a40c73ca update-package.json
```

### Change Timezone and Group

```bash
cat > update-location.json << 'EOF'
{
  "timeZone": "America/Chicago",
  "bsnGroupName": "Midwest Region"
}
EOF

./bin/bsn-bdeploy-setup-update --setup-id 618fb7363a682fe7a40c73ca update-location.json
```

### Update Network Configuration

```bash
cat > update-network.json << 'EOF'
{
  "network": {
    "timeServers": [
      "http://time.brightsignnetwork.com",
      "http://time.nist.gov"
    ],
    "interfaces": [
      {
        "id": "wired_eth0",
        "name": "eth0",
        "type": "Ethernet",
        "proto": "DHCPv4",
        "contentDownloadEnabled": true,
        "healthReportingEnabled": true
      }
    ]
  }
}
EOF

./bin/bsn-bdeploy-setup-update --setup-id 618fb7363a682fe7a40c73ca update-network.json
```

### Full Reconfiguration

```bash
# Use the complete config file
./bin/bsn-bdeploy-setup-update --setup-id 618fb7363a682fe7a40c73ca example-full-update.json
```

## Command Line Options

```
Usage: bdeploy-update-setup --setup-id <id> [options] <config.json>

Required:
  --setup-id string    Setup ID to update (required)

Options:
  --help              Display usage information
  --verbose           Show detailed information including before/after comparison
  --timeout int       Request timeout in seconds (default: 30)

Environment Variables:
  BS_CLIENT_ID        BSN.cloud API client ID (required)
  BS_SECRET           BSN.cloud API client secret (required)
```

## When Players Get Updated Configurations

**Important:** Updating a setup record does **not** immediately push changes to players. The update behavior depends on how players are provisioned:

### Players Associated via Setup ID
Players that reference a setup ID will:
- Check for updates on next network connection
- Download the new setup configuration automatically
- Apply changes on next reboot (or immediately for certain settings)

### Players with Direct URL Provisioning
Players provisioned with a direct presentation URL will:
- **Not** automatically get setup updates
- Continue using their locally cached configuration
- Require manual re-provisioning to get changes

## Common Use Cases

### 1. Rolling Out Package Updates
```bash
# Update package name to deploy new content version
echo '{"packageName": "retail-v2.1"}' | ./bin/bsn-bdeploy-setup-update --setup-id $SETUP_ID /dev/stdin
```

### 2. Reorganizing Player Groups
```bash
# Move players to different BSN group
echo '{"bsnGroupName": "West Coast Stores"}' | ./bin/bsn-bdeploy-setup-update --setup-id $SETUP_ID /dev/stdin
```

### 3. Timezone Adjustments (DST Changes)
```bash
# Update timezone for region
echo '{"timeZone": "America/Phoenix"}' | ./bin/bsn-bdeploy-setup-update --setup-id $SETUP_ID /dev/stdin
```

### 4. Network Reconfiguration
```bash
# Switch from DHCP to static IP (full network config required)
./bin/bsn-bdeploy-setup-update --setup-id $SETUP_ID network-static-config.json
```

## Integration Workflow

### Get Setup ID from Existing Setup
```bash
# List all setups and find the one you want to update
./bin/bsn-bdeploy-setup-list | jq '.[] | select(.packageName == "retail-v1") | ._id'

# Or get setup details
./bin/bsn-bdeploy-setup-get --setup-id <id>
```

### Complete Update Workflow
```bash
#!/bin/bash
SETUP_ID="618fb7363a682fe7a40c73ca"

# 1. Get current setup
echo "Current setup:"
./bin/bsn-bdeploy-setup-get --setup-id $SETUP_ID

# 2. Prepare update config
cat > update.json << 'EOF'
{
  "packageName": "retail-display-v2",
  "timeZone": "America/Los_Angeles"
}
EOF

# 3. Apply update
./bin/bsn-bdeploy-setup-update --setup-id $SETUP_ID --verbose update.json

# 4. Verify update
echo "Updated setup:"
./bin/bsn-bdeploy-setup-get --setup-id $SETUP_ID
```

## Troubleshooting

### Setup Not Found Error
```
❌ Failed to fetch setup record: setup not found
```
**Solution:** Verify the setup ID is correct:
```bash
./bin/bsn-bdeploy-setup-list | jq '.[].\_id'
```

### Network Context Error
```
❌ Failed to set network context
```
**Solution:** Ensure the network name in your config matches your BSN.cloud network:
```bash
# If config specifies networkName, it must exist
# Otherwise, the existing setup's network is used
```

### Authentication Failed
```
❌ Authentication failed
```
**Solution:** Check your credentials:
```bash
echo "Client ID: $BS_CLIENT_ID"
echo "Secret: ${BS_SECRET:0:8}..." # Shows first 8 chars only
```

### Partial Update Not Working
```
# Fields not updating as expected
```
**Solution:** Remember that **empty string values are ignored**. To update a field to empty, you need to explicitly set it in code or use a full update.

## Best Practices

1. **Use Partial Updates** - Only specify fields that need to change
2. **Test First** - Test updates on a non-production setup ID first
3. **Document Changes** - Keep a log of when and why setups were updated
4. **Version Package Names** - Use versioned package names (v1, v2, v3) for tracking
5. **Coordinate with Players** - Plan updates during maintenance windows
6. **Verify After Update** - Always verify the update with `bdeploy-get-setup`

## Security Considerations

- Device registration tokens are preserved during updates
- Token expiration is not modified by setup updates
- Network scope is maintained (can't move setup to different network without proper context)
- Updates require valid BSN.cloud authentication

## Related Examples

- `bdeploy-add-setup` - Create new setup records
- `bdeploy-get-setup` - View setup record details
- `bdeploy-list-setups` - List all setup records
- `bdeploy-delete-setup` - Delete setup records
- `bdeploy-associate` - Associate players with setups

## API Reference

This example uses the B-Deploy v3 Setup API:
- Endpoint: `PUT https://provision.bsn.cloud/rest-setup/v3/setup`
- Authentication: OAuth2 Bearer token
- Documentation: See `docs/b-deploy.md`
