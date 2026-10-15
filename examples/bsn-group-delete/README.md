# Delete Group Example

Permanently removes a BrightSign group from BSN.cloud network. This tool provides a safe, interactive way to delete groups with confirmation prompts and detailed group information display before deletion.

**What This Program Does:**
- Deletes a group from BSN.cloud by ID
- Shows group information before deletion for verification
- Requires confirmation before deleting (can be skipped with --force)
- Provides detailed feedback on the deletion process

**What This Program Does NOT Do:**
- Does not delete devices in the group (devices remain but become ungrouped)
- Does not affect device content or presentations
- Does not delete other groups or cascading deletions

## ⚠️ Important Warning

**Deleting a group is permanent!**
- The group will be immediately removed from BSN.cloud
- Devices in the group will become ungrouped
- Group-specific settings and metadata will be lost
- Content deployments targeting the group will need to be reassigned

**Use Cases:**
- Cleaning up unused groups
- Removing test groups
- Reorganizing group structure
- Consolidating group hierarchy

## Quick Start

```bash
# Build all examples
make build-examples

# Set your credentials
export BS_CLIENT_ID="your-client-id"
export BS_SECRET="your-client-secret"
export BS_NETWORK="your-network-name"

# Delete group with confirmation prompt
./bin/group-delete --id 12345

# Delete group without confirmation (use with caution!)
./bin/group-delete --id 12345 --force
```

## Command Line Options

```
Usage: group-delete [options]

Options:
  --id int             Delete group with ID (required)
  --network string     Network name to use (overrides BS_NETWORK)
  -n string            Network name to use [alias for --network]
  --force              Skip confirmation prompt
  -y                   Skip confirmation prompt [alias for --force]
  --verbose            Show detailed information
  --timeout int        Request timeout in seconds (default: 30)
  --help               Display usage information

Environment Variables:
  BS_CLIENT_ID        BSN.cloud API client ID (required)
  BS_SECRET           BSN.cloud API client secret (required)
  BS_NETWORK          BSN.cloud network name (optional)
```

## Usage Examples

### Basic Usage (Interactive Mode)

```bash
# Delete group with confirmation
./bin/group-delete --id 12345
```

Output:
```
═══════════════════════════════════════════════════════════════════
Group Information
═══════════════════════════════════════════════════════════════════
ID:           12345
Name:         Old Test Group
═══════════════════════════════════════════════════════════════════

⚠️  WARNING: This will permanently delete group 'Old Test Group' from BSN.cloud!
           Group ID: 12345
           Devices in this group will NOT be deleted but will be ungrouped.

Are you sure you want to delete this group? [yes/no]: yes

🗑️  Deleting group 'Old Test Group' (ID: 12345)...
✅ Group deleted successfully

═══════════════════════════════════════════════════════════════════
Group 'Old Test Group' has been removed from network 'Production Network'
═══════════════════════════════════════════════════════════════════

💡 Note: Devices that were in this group are still in your network but are now ungrouped
```

### Force Delete (No Confirmation)

```bash
# Delete without confirmation - use carefully!
./bin/group-delete --id 12345 --force
```

**WARNING**: Using `--force` skips the confirmation prompt. This is useful for automation but dangerous if used carelessly.

### Verbose Mode

```bash
# Show detailed progress information
./bin/group-delete --id 12345 --verbose
```

Output includes:
- Authentication progress
- Network context information
- Group lookup details
- Step-by-step deletion progress

### Specify Network

```bash
# Override BS_NETWORK environment variable
./bin/group-delete --id 12345 -n "Test Network"

# Useful when managing multiple networks
./bin/group-delete --id 12345 --network "Staging"
```

## Workflow

The program follows this sequence:

1. **Authentication** - OAuth2 login to BSN.cloud
2. **Network Context** - Select target network
3. **Group Lookup** - Find group by ID
4. **Display Info** - Show group details for verification
5. **Confirmation** - Ask user to confirm (unless --force)
6. **Delete** - Remove group from BSN.cloud
7. **Confirm Success** - Show deletion confirmation

## Safety Features

### Confirmation Prompt
By default, the program requires explicit confirmation:
- Displays full group information
- Shows warning message
- Requires typing "yes" or "y" to proceed
- Any other input cancels the operation

### Group Information Display
Before deletion, you see:
- Group ID
- Group name
- Any additional metadata

This helps prevent accidental deletion of the wrong group.

### Error Handling
- Validates group exists before attempting deletion
- Handles authentication failures gracefully
- Provides clear error messages
- Safe exits on invalid input

## Common Use Cases

### 1. Cleaning Up Test Groups

```bash
# Interactive deletion with full information
./bin/group-delete --id 12345 --verbose
```

### 2. Bulk Deletion Script

```bash
#!/bin/bash
# Delete multiple groups from a list

while read group_id; do
    echo "Deleting group: $group_id"
    ./bin/group-delete --id "$group_id" --force || echo "Failed to delete group $group_id"
    sleep 1  # Rate limiting
done < groups-to-delete.txt
```

### 3. Automation/CI Pipeline

```bash
# Non-interactive deletion for automated workflows
./bin/group-delete --id "$GROUP_ID" --force --timeout 60
```

### 4. Cleanup After Reorganization

```bash
# Remove old groups after restructuring
for id in 100 101 102; do
    ./bin/group-delete --id $id -y
done
```

## What Happens After Deletion

### On BSN.cloud:
- ✅ Group immediately removed from group list
- ✅ Devices become ungrouped but remain in network
- ✅ Group metadata is deleted
- ⚠️  Content deployments to the group stop working

### On Devices:
- ❌ No immediate changes to devices
- ❌ Devices continue playing current content
- ⚠️  May show warnings about missing group
- ⚠️  Group-targeted deployments will fail

### To Reorganize Devices:
1. Move devices to new groups before deletion (recommended)
2. Or reassign devices to groups after deletion
3. Update content deployment targets

## Troubleshooting

### Group Not Found

```
❌ Failed to find group: group not found
```

**Solution:** Verify the group ID is correct:
```bash
# Get group info to verify ID
./bin/group-info --id 12345
```

### Permission Denied

```
❌ Failed to delete group: insufficient permissions
```

**Solution:** Ensure your API credentials have group delete permissions (`bsn.api.main.groups.delete` scope).

### Group Already Deleted

```
❌ Failed to find group: group not found
```

**Solution:** The group may have already been deleted. Verify with:
```bash
./bin/group-info --id 12345
```

### Network Context Error

```
❌ Failed to set network context: network not found
```

**Solution:** Verify the network name is correct.

### Cannot Delete Built-in Groups

```
❌ Failed to delete group: cannot delete system group
```

**Solution:** Some groups like "All Devices" are system groups and cannot be deleted.

## Best Practices

1. **Always Review First**
   - Use interactive mode (without --force) for manual deletions
   - Verify group information before confirming
   - Check group doesn't have critical devices

2. **Move Devices First**
   - Reassign devices to new groups before deletion
   - Ensures devices maintain proper organization
   - Prevents accidental loss of group memberships

3. **Document Deletions**
   - Keep records of deleted groups
   - Note the reason for deletion
   - Track which devices were affected

4. **Test on Non-Production First**
   - Test deletion workflows on test groups
   - Verify automation scripts work correctly
   - Ensure downstream systems handle group deletion

5. **Use Force Mode Carefully**
   - Only use --force in trusted automation
   - Double-check group IDs in scripts
   - Add safety checks in bulk deletion scripts

6. **Rate Limiting**
   - Add delays between bulk deletions
   - Respect API rate limits (1000 requests/hour)

7. **Update Deployments**
   - Review content deployments targeting the group
   - Update deployment targets before deletion
   - Verify presentations still reach intended devices

## Impact on Other Systems

### Content Deployments
- Deployments targeting the deleted group will fail
- Update deployment targets before deletion
- Consider moving devices to alternate groups first

### Reporting and Analytics
- Historical reports may reference the deleted group
- Group name will no longer resolve
- Use group ID for historical reference if needed

### Integrations
- Third-party systems may reference the group
- Update external configurations
- Notify integration owners of group deletion

## Related Examples

- `group-info` - Get detailed group information
- `group-update` - Modify group settings
- `list-devices` - List devices (can filter by group)
- `device-change-group` - Move devices between groups

## API Reference

This example uses the Group Management API:
- Endpoint: `DELETE https://api.bsn.cloud/2022/06/REST/Groups/Regular/{id}`
- Authentication: OAuth2 Bearer token
- Required Scope: `bsn.api.main.groups.delete`
- Documentation: See `docs/api-guide.md`

## Security Considerations

- **Requires Delete Permissions**: API credentials must have group delete scope
- **Audit Trail**: BSN.cloud maintains audit logs of group deletions
- **Irreversible**: Deleted groups cannot be restored
- **Device Impact**: Devices lose group membership but remain in network

## Exit Codes

- `0` - Success (group deleted or cancelled by user)
- `1` - Error (authentication, network, group not found, etc.)

## Notes

- Deletion is immediate and cannot be undone
- Group metadata is permanently removed
- Devices in the group are NOT deleted
- Devices become ungrouped after deletion
- Content deployments to the group will stop working
- Re-creating a group with the same name creates a new group (different ID)
