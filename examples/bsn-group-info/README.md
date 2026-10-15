# Group Info Example

Retrieves detailed information about a BrightSign group from BSN.cloud. Groups organize devices for easier management and content deployment.

**What This Program Does:**
- Retrieves group information by ID
- Displays group name, description, and metadata
- Supports both human-readable and JSON output formats
- Shows which network the group belongs to

**What This Program Does NOT Do:**
- Does not list devices within the group (use list-devices with group filter)
- Does not modify group settings
- Does not create or delete groups

## Quick Start

```bash
# Build all examples
make build-examples

# Set your credentials
export BS_CLIENT_ID="your-client-id"
export BS_SECRET="your-client-secret"
export BS_NETWORK="your-network-name"

# Get group information
./bin/group-info --id 12345

# Get group as JSON
./bin/group-info --id 12345 --json
```

## Command Line Options

```
Usage: group-info [options]

Options:
  --id int             Group ID to retrieve (required)
  --network string     Network name to use (overrides BS_NETWORK)
  -n string            Network name to use [alias for --network]
  --json               Output as JSON
  --verbose            Show detailed information
  --timeout int        Request timeout in seconds (default: 30)
  --help               Display usage information

Environment Variables:
  BS_CLIENT_ID        BSN.cloud API client ID (required)
  BS_SECRET           BSN.cloud API client secret (required)
  BS_NETWORK          BSN.cloud network name (optional)
```

## Usage Examples

### Basic Usage

```bash
# Get group information
./bin/group-info --id 12345
```

Output:
```
═══════════════════════════════════════════════════════════════════
Group Information
═══════════════════════════════════════════════════════════════════
ID:           12345
Name:         Conference Rooms
Description:  All conference room displays
Type:         Regular
═══════════════════════════════════════════════════════════════════

✅ Group 'Conference Rooms' retrieved from network 'Production Network'
```

### JSON Output

```bash
# Get group as JSON for scripting
./bin/group-info --id 12345 --json
```

Output:
```json
{
  "id": 12345,
  "name": "Conference Rooms",
  "description": "All conference room displays",
  "type": "Regular"
}
```

This is useful for:
- Parsing in scripts
- Piping to jq for filtering
- Integrating with automation tools
- Storing in configuration files

### Verbose Mode

```bash
# Show detailed progress information
./bin/group-info --id 12345 --verbose
```

Output includes:
- Authentication progress
- Network context information
- Group lookup details

### Specify Network

```bash
# Override BS_NETWORK environment variable
./bin/group-info --id 12345 -n "Test Network"

# Useful when managing multiple networks
./bin/group-info --id 12345 --network "Production"
```

## Workflow

The program follows this sequence:

1. **Authentication** - OAuth2 login to BSN.cloud
2. **Network Context** - Select target network
3. **Group Lookup** - Retrieve group by ID
4. **Display Info** - Show group details

## Common Use Cases

### 1. Verify Group Exists

```bash
# Check if a group ID is valid
./bin/group-info --id 12345
```

If the group doesn't exist, you'll get an error:
```
❌ Failed to get group: group not found
```

### 2. Get Group Name for Scripts

```bash
# Extract group name using jq
GROUP_NAME=$(./bin/group-info --id 12345 --json | jq -r '.name')
echo "Group name: $GROUP_NAME"
```

### 3. Compare Groups Across Networks

```bash
# Compare the same group ID across different networks
./bin/group-info --id 100 --network "Production" --json > prod-group.json
./bin/group-info --id 100 --network "Staging" --json > staging-group.json
diff prod-group.json staging-group.json
```

### 4. Automation/CI Pipeline

```bash
# Verify group configuration in CI
GROUP_JSON=$(./bin/group-info --id "$GROUP_ID" --json --timeout 60)
EXPECTED_NAME="Production Displays"
ACTUAL_NAME=$(echo "$GROUP_JSON" | jq -r '.name')

if [ "$ACTUAL_NAME" != "$EXPECTED_NAME" ]; then
    echo "Error: Group name mismatch"
    exit 1
fi
```

### 5. Export Group Metadata

```bash
# Export all group information for documentation
./bin/group-info --id 12345 --json | jq '.' > group-12345-metadata.json
```

## Group Types

BSN.cloud supports different group types:

- **Regular** - Standard device groups created by users
- **Dynamic** - Groups with automatic membership based on rules
- **All Devices** - Built-in group containing all devices

This tool retrieves any group type.

## Troubleshooting

### Group Not Found

```
❌ Failed to get group: group not found
```

**Solutions:**
- Verify the group ID is correct
- Check you're using the correct network
- List all groups to find the correct ID:
  ```bash
  # If you have a list-groups example
  ./bin/list-groups
  ```

### Invalid Group ID

```
❌ Error: Must specify a valid --id (greater than 0)
```

**Solution:** Provide a positive integer for the group ID:
```bash
./bin/group-info --id 12345
```

### Permission Denied

```
❌ Failed to get group: insufficient permissions
```

**Solution:** Ensure your API credentials have group read permissions.

### Network Context Error

```
❌ Failed to set network context: network not found
```

**Solution:** Verify the network name:
```bash
# Check available networks with auth-info example
./bin/auth-info
```

## Output Fields

### Standard Fields

- **ID** - Unique numeric identifier for the group
- **Name** - Human-readable group name
- **Description** - Optional description of the group's purpose
- **Type** - Group type (Regular, Dynamic, etc.)

### Optional Fields

Depending on the group configuration, you might also see:
- Parent group information
- Creation/modification timestamps
- Custom metadata fields

Use `--json` to see all available fields.

## Integration Examples

### Bash Script

```bash
#!/bin/bash
# Check if group exists before performing operations

GROUP_ID="$1"
if [ -z "$GROUP_ID" ]; then
    echo "Usage: $0 <group-id>"
    exit 1
fi

# Try to get group info
if ./bin/group-info --id "$GROUP_ID" > /dev/null 2>&1; then
    echo "✅ Group $GROUP_ID exists"
    # Proceed with operations...
else
    echo "❌ Group $GROUP_ID not found"
    exit 1
fi
```

### Python Integration

```python
import subprocess
import json

def get_group_info(group_id):
    """Retrieve group information as Python dict."""
    result = subprocess.run(
        ['./bin/group-info', '--id', str(group_id), '--json'],
        capture_output=True,
        text=True,
        check=True
    )
    return json.loads(result.stdout)

# Usage
group = get_group_info(12345)
print(f"Group name: {group['name']}")
print(f"Description: {group['description']}")
```

### jq Processing

```bash
# Extract specific fields
./bin/group-info --id 12345 --json | jq '{name, description}'

# Filter groups by name pattern (in a loop)
for id in 100 101 102; do
    ./bin/group-info --id $id --json | jq 'select(.name | contains("Production"))'
done
```

## Best Practices

1. **Use JSON for Automation**
   - Always use `--json` flag in scripts
   - Parse with jq or similar tools
   - Handle errors with proper exit code checks

2. **Cache Group Information**
   - Group metadata changes infrequently
   - Consider caching results to reduce API calls
   - Respect rate limits (1000 requests/hour)

3. **Verify Before Operations**
   - Check group exists before updating or deleting
   - Validate group type matches expectations
   - Confirm network context is correct

4. **Error Handling**
   - Always check exit codes ($?)
   - Capture stderr for error messages
   - Provide user-friendly error context

5. **Timeout Configuration**
   - Use longer timeouts for unreliable networks
   - Default 30 seconds is usually sufficient
   - Increase for slow connections: `--timeout 60`

## Related Examples

- `list-devices` - List devices (can filter by group)
- `group-update` - Modify group settings
- `group-delete` - Remove a group
- `device-change-group` - Move devices between groups

## API Reference

This example uses the Group Management API:
- Endpoint: `GET https://api.bsn.cloud/2022/06/REST/Groups/Regular/{id}`
- Authentication: OAuth2 Bearer token
- Required Scope: `bsn.api.main.groups.read`
- Documentation: See `docs/api-guide.md`

## Exit Codes

- `0` - Success (group retrieved)
- `1` - Error (authentication, network, group not found, etc.)

## Notes

- Group IDs are unique within a network
- Same group ID may refer to different groups in different networks
- Group information is read-only (use group-update to modify)
- This tool only retrieves metadata, not device membership lists
