# Group Update Example

Updates BrightSign group settings in BSN.cloud. This tool allows you to modify group names and other metadata.

**What This Program Does:**
- Updates group name by ID
- Shows current settings before making changes
- Displays the proposed changes for review
- Confirms successful update with new settings

**What This Program Does NOT Do:**
- Does not change device membership (use device-change-group)
- Does not delete groups (use group-delete)
- Does not create new groups
- Does not move devices between groups

## Quick Start

```bash
# Build all examples
make build-examples

# Set your credentials
export BS_CLIENT_ID="your-client-id"
export BS_SECRET="your-client-secret"
export BS_NETWORK="your-network-name"

# Update group name
./bin/group-update --id 12345 --name "New Group Name"
```

## Command Line Options

```
Usage: group-update [options]

Options:
  --id int             Group ID to update (required)
  --name string        New group name (required)
  --network string     Network name to use (overrides BS_NETWORK)
  -n string            Network name to use [alias for --network]
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
# Update group name
./bin/group-update --id 12345 --name "Conference Room Displays"
```

Output:
```
═══════════════════════════════════════════════════════════════════
Current Group Information
═══════════════════════════════════════════════════════════════════
ID:           12345
Name:         Conference Rooms
═══════════════════════════════════════════════════════════════════

═══════════════════════════════════════════════════════════════════
Proposed Changes
═══════════════════════════════════════════════════════════════════
Name:         Conference Rooms → Conference Room Displays
═══════════════════════════════════════════════════════════════════

═══════════════════════════════════════════════════════════════════
✅ Group Updated Successfully
═══════════════════════════════════════════════════════════════════
ID:           12345
Name:         Conference Room Displays
═══════════════════════════════════════════════════════════════════

💡 Group 'Conference Room Displays' has been updated in network 'Production Network'
```

### Verbose Mode

```bash
# Show detailed progress information
./bin/group-update --id 12345 --name "Updated Name" --verbose
```

Output includes:
- Authentication progress
- Network context information
- Group retrieval details
- Update operation progress

### Specify Network

```bash
# Override BS_NETWORK environment variable
./bin/group-update --id 12345 --name "New Name" -n "Test Network"

# Useful when managing multiple networks
./bin/group-update --id 12345 --name "Staging Displays" --network "Staging"
```

## Workflow

The program follows this sequence:

1. **Authentication** - OAuth2 login to BSN.cloud
2. **Network Context** - Select target network
3. **Retrieve Current** - Get current group settings
4. **Display Current** - Show existing configuration
5. **Display Changes** - Show what will change
6. **Update Group** - Apply the changes
7. **Confirm Success** - Display updated settings

## Common Use Cases

### 1. Rename Group for Clarity

```bash
# Make group names more descriptive
./bin/group-update --id 100 --name "Lobby Displays - Main Campus"
```

### 2. Standardize Naming

```bash
#!/bin/bash
# Standardize group names across network

# Add prefix to all groups
./bin/group-update --id 100 --name "PROD-Conference-Rooms"
./bin/group-update --id 101 --name "PROD-Lobbies"
./bin/group-update --id 102 --name "PROD-Cafeteria"
```

### 3. Update After Reorganization

```bash
# Reflect organizational changes
./bin/group-update --id 200 --name "North Building - Floor 1"
./bin/group-update --id 201 --name "North Building - Floor 2"
```

### 4. Automation/CI Pipeline

```bash
# Update group names from configuration file
GROUP_ID=12345
NEW_NAME=$(jq -r '.groups[] | select(.id == '$GROUP_ID') | .name' config.json)
./bin/group-update --id "$GROUP_ID" --name "$NEW_NAME" --timeout 60
```

### 5. Batch Updates

```bash
#!/bin/bash
# Update multiple groups from CSV file
# Format: id,new_name

while IFS=, read -r id name; do
    echo "Updating group $id to: $name"
    ./bin/group-update --id "$id" --name "$name" || echo "Failed to update group $id"
    sleep 1  # Rate limiting
done < group-updates.csv
```

## Important Notes

### Group Name Requirements

- Group names must be unique within a network
- Special characters are allowed
- Maximum length varies by BSN.cloud API (typically 255 characters)
- Empty names are not allowed

### Update Behavior

- Updates are immediate and take effect immediately
- Devices in the group are not affected
- Content deployments continue normally
- Group ID remains unchanged

### Network Scope

- Group names only need to be unique within a network
- Same name can exist in different networks
- Updates only affect the specified network

## Troubleshooting

### Group Not Found

```
❌ Failed to get group: group not found
```

**Solutions:**
- Verify the group ID is correct
- Check you're using the correct network
- List all groups to find the correct ID

### Invalid Group ID

```
❌ Error: Must specify a valid --id (greater than 0)
```

**Solution:** Provide a positive integer for the group ID:
```bash
./bin/group-update --id 12345 --name "New Name"
```

### Missing Group Name

```
❌ Error: Must specify --name with new group name
```

**Solution:** Always provide the new name:
```bash
./bin/group-update --id 12345 --name "Conference Rooms"
```

### Duplicate Name Error

```
❌ Failed to update group: group name already exists
```

**Solution:** Choose a unique name within the network:
```bash
# Add a prefix or suffix to make it unique
./bin/group-update --id 12345 --name "Conference Rooms - East Wing"
```

### Permission Denied

```
❌ Failed to update group: insufficient permissions
```

**Solution:** Ensure your API credentials have group update permissions (`bsn.api.main.groups.update` scope).

### Network Context Error

```
❌ Failed to set network context: network not found
```

**Solution:** Verify the network name is correct.

## Best Practices

1. **Review Before Updating**
   - The tool shows current settings and proposed changes
   - Verify the group ID is correct before confirming
   - Check that the new name is what you intend

2. **Use Descriptive Names**
   - Make group names clear and meaningful
   - Include location, purpose, or environment in the name
   - Follow consistent naming conventions

3. **Test on Non-Production First**
   - Test update workflows on test/staging groups first
   - Verify naming conventions work as expected
   - Ensure automation scripts handle errors correctly

4. **Keep Records**
   - Document group name changes
   - Track the reason for updates
   - Maintain naming convention documentation

5. **Rate Limiting**
   - Add delays between bulk updates
   - Respect API rate limits (1000 requests/hour)
   - Use appropriate timeout values

6. **Validation**
   - Check group exists before updating
   - Validate new name meets requirements
   - Confirm network context is correct

## Integration Examples

### Bash Script

```bash
#!/bin/bash
# Safely update group name with validation

GROUP_ID="$1"
NEW_NAME="$2"

if [ -z "$GROUP_ID" ] || [ -z "$NEW_NAME" ]; then
    echo "Usage: $0 <group-id> <new-name>"
    exit 1
fi

# Check if group exists
if ! ./bin/group-info --id "$GROUP_ID" > /dev/null 2>&1; then
    echo "❌ Group $GROUP_ID not found"
    exit 1
fi

# Update the group
./bin/group-update --id "$GROUP_ID" --name "$NEW_NAME"
```

### Python Integration

```python
import subprocess
import sys

def update_group(group_id, new_name):
    """Update a group name."""
    try:
        subprocess.run(
            ['./bin/group-update', '--id', str(group_id), '--name', new_name],
            check=True,
            capture_output=True,
            text=True
        )
        print(f"✅ Updated group {group_id} to '{new_name}'")
        return True
    except subprocess.CalledProcessError as e:
        print(f"❌ Failed to update group {group_id}: {e.stderr}")
        return False

# Usage
update_group(12345, "Conference Room Displays")
```

### Bulk Update from CSV

```bash
#!/bin/bash
# Update groups from CSV file
# CSV format: group_id,new_name

CSV_FILE="group-updates.csv"

if [ ! -f "$CSV_FILE" ]; then
    echo "Error: $CSV_FILE not found"
    exit 1
fi

echo "Starting bulk group updates..."
TOTAL=$(wc -l < "$CSV_FILE")
CURRENT=0
FAILED=0

while IFS=, read -r group_id new_name; do
    CURRENT=$((CURRENT + 1))
    echo "[$CURRENT/$TOTAL] Updating group $group_id..."

    if ./bin/group-update --id "$group_id" --name "$new_name" > /dev/null 2>&1; then
        echo "  ✅ Success: $new_name"
    else
        echo "  ❌ Failed: $group_id"
        FAILED=$((FAILED + 1))
    fi

    # Rate limiting
    sleep 1
done < "$CSV_FILE"

echo ""
echo "Update complete: $((TOTAL - FAILED))/$TOTAL succeeded, $FAILED failed"
```

## Related Examples

- `group-info` - View group details
- `group-delete` - Remove a group
- `list-devices` - List devices in a group
- `device-change-group` - Move devices between groups

## API Reference

This example uses the Group Management API:
- Endpoint: `PUT https://api.bsn.cloud/2022/06/REST/Groups/Regular/{id}`
- Authentication: OAuth2 Bearer token
- Required Scope: `bsn.api.main.groups.update`
- Documentation: See `docs/api-guide.md`

## Exit Codes

- `0` - Success (group updated)
- `1` - Error (authentication, network, group not found, validation, etc.)

## Notes

- Group updates are immediate
- All devices remain in the group after update
- Content deployments continue normally
- Group ID never changes
- Only modifiable fields can be updated (currently: name)
