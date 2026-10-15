# Delete Device Example

Permanently removes a BrightSign device from BSN.cloud network. This tool provides a safe, interactive way to delete devices with confirmation prompts and detailed device information display before deletion.

**What This Program Does:**
- Deletes a device from BSN.cloud by ID or serial number
- Shows device information before deletion for verification
- Requires confirmation before deleting (can be skipped with --force)
- Provides detailed feedback on the deletion process

**What This Program Does NOT Do:**
- Does not physically reset or affect the device hardware
- Does not automatically reprovision the device
- Does not remove the device from other systems/databases

## ⚠️ Important Warning

**Deleting a device is permanent!**
- The device will be immediately removed from BSN.cloud
- All device settings, group memberships, and history will be lost
- The device must be re-provisioned to rejoin the network
- Content deployments to the device will stop

**Use Cases:**
- Decommissioning devices that are no longer in use
- Cleaning up test devices
- Removing devices that need to be reprovisioned
- Transferring devices to different networks

## Quick Start

```bash
# Build all examples
make build-examples

# Set your credentials
export BS_CLIENT_ID="your-client-id"
export BS_SECRET="your-client-secret"
export BS_NETWORK="your-network-name"

# Delete device with confirmation prompt
./bin/device-delete --serial UTD41X000009

# Delete device without confirmation (use with caution!)
./bin/device-delete --serial UTD41X000009 --force
```

## Command Line Options

```
Usage: device-delete [options]

Options:
  --serial string      Delete device with serial number
  --id int             Delete device with ID
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
# Delete by serial number with confirmation
./bin/device-delete --serial UTD41X000009
```

Output:
```
==================================================
Device Information
==================================================
Name:         Conference Room Display
Serial:       UTD41X000009
ID:           12345
Model:        XD1034
Description:  Main conference room screen
Group:        Conference Rooms
==================================================

⚠️  WARNING: This will permanently delete device 'Conference Room Display' from BSN.cloud!
           Serial number: UTD41X000009
           The device will need to be re-provisioned to rejoin the network.

Are you sure you want to delete this device? [yes/no]: yes

🗑️  Deleting device 'Conference Room Display' (Serial: UTD41X000009)...
✅ Device deleted successfully

==================================================
Device 'Conference Room Display' has been removed from network 'Production Network'
==================================================

💡 Note: The physical device will need to be re-provisioned to rejoin BSN.cloud
```

### Force Delete (No Confirmation)

```bash
# Delete without confirmation - use carefully!
./bin/device-delete --serial UTD41X000009 --force
```

**WARNING**: Using `--force` skips the confirmation prompt. This is useful for automation but dangerous if used carelessly.

### Delete by Device ID

```bash
# Delete using device ID instead of serial
./bin/device-delete --id 12345
```

### Verbose Mode

```bash
# Show detailed progress information
./bin/device-delete --serial UTD41X000009 --verbose
```

Output includes:
- Authentication progress
- Network context information
- Device lookup details
- Step-by-step deletion progress

### Specify Network

```bash
# Override BS_NETWORK environment variable
./bin/device-delete --serial UTD41X000009 -n "Test Network"

# Useful when managing multiple networks
./bin/device-delete --serial UTD41X000009 --network "Production"
```

## Workflow

The program follows this sequence:

1. **Authentication** - OAuth2 login to BSN.cloud
2. **Network Context** - Select target network
3. **Device Lookup** - Find device by serial or ID
4. **Display Info** - Show device details for verification
5. **Confirmation** - Ask user to confirm (unless --force)
6. **Delete** - Remove device from BSN.cloud
7. **Confirm Success** - Show deletion confirmation

## Safety Features

### Confirmation Prompt
By default, the program requires explicit confirmation:
- Displays full device information
- Shows warning message
- Requires typing "yes" or "y" to proceed
- Any other input cancels the operation

### Device Information Display
Before deletion, you see:
- Device name
- Serial number
- Device ID
- Model
- Description (if set)
- Group membership

This helps prevent accidental deletion of the wrong device.

### Error Handling
- Validates device exists before attempting deletion
- Handles authentication failures gracefully
- Provides clear error messages
- Safe exits on invalid input

## Common Use Cases

### 1. Decommissioning a Device

```bash
# Interactive deletion with full information
./bin/device-delete --serial UTD41X000009 --verbose
```

### 2. Bulk Deletion Script

```bash
#!/bin/bash
# Delete multiple devices from a list

while read serial; do
    echo "Deleting device: $serial"
    ./bin/device-delete --serial "$serial" --force || echo "Failed to delete $serial"
    sleep 1  # Rate limiting
done < devices-to-delete.txt
```

### 3. Automation/CI Pipeline

```bash
# Non-interactive deletion for automated workflows
./bin/device-delete --serial "$DEVICE_SERIAL" --force --timeout 60
```

### 4. Testing/Development

```bash
# Clean up test devices
for serial in TEST001 TEST002 TEST003; do
    ./bin/device-delete --serial $serial -y
done
```

## What Happens After Deletion

### On BSN.cloud:
- ✅ Device immediately removed from device list
- ✅ Device removed from all groups
- ✅ Device subscription/license freed up
- ✅ Device history and settings deleted

### On the Physical Device:
- ❌ No automatic changes occur
- ❌ Device continues running current presentation
- ❌ Device may show network connection errors
- ❌ Device will not receive new content updates

### To Use the Device Again:
1. Factory reset the device (recommended)
2. Reprovision through B-Deploy or BSN.cloud
3. Device will appear as new in the network
4. Reconfigure groups and settings

## Troubleshooting

### Device Not Found
```
❌ Failed to find device: device not found
```
**Solution:** Verify the serial number or ID is correct:
```bash
# List devices to find correct serial/ID
./bin/list-devices | grep "device-name"
```

### Permission Denied
```
❌ Failed to delete device: insufficient permissions
```
**Solution:** Ensure your API credentials have device delete permissions (`bsn.api.main.devices.delete` scope).

### Device Already Deleted
```
❌ Failed to find device: device not found
```
**Solution:** The device may have already been deleted. Verify with:
```bash
./bin/device-info --serial UTD41X000009
```

### Network Context Error
```
❌ Failed to set network context: network not found
```
**Solution:** Verify the network name:
```bash
# Check available networks
./bin/auth-info
```

## Best Practices

1. **Always Review First**
   - Use interactive mode (without --force) for manual deletions
   - Verify device information before confirming

2. **Document Deletions**
   - Keep records of deleted devices
   - Note the reason for deletion
   - Track serial numbers for inventory

3. **Test on Non-Production First**
   - Test deletion workflows on test devices
   - Verify automation scripts work correctly

4. **Use Force Mode Carefully**
   - Only use --force in trusted automation
   - Double-check serial numbers in scripts
   - Add safety checks in bulk deletion scripts

5. **Rate Limiting**
   - Add delays between bulk deletions
   - Respect API rate limits (1000 requests/hour)

6. **Backup Device Settings**
   - Export device configurations before deletion (if needed)
   - Document group memberships
   - Save presentation assignments

## Related Examples

- `list-devices` - List all devices to find serial numbers/IDs
- `device-info` - Get detailed device information
- `device-change-group` - Move devices between groups
- `bdeploy-delete-device` - Delete from B-Deploy provisioning system

## API Reference

This example uses the Device Management API:
- Endpoint: `DELETE https://api.bsn.cloud/2022/06/REST/Devices/{id}`
- Authentication: OAuth2 Bearer token
- Required Scope: `bsn.api.main.devices.delete`
- Documentation: See `docs/api-guide.md`

## Security Considerations

- **Requires Delete Permissions**: API credentials must have device delete scope
- **Audit Trail**: BSN.cloud maintains audit logs of device deletions
- **Irreversible**: Deleted devices cannot be restored from BSN.cloud
- **License Management**: Deletion frees up device subscription slots

## Exit Codes

- `0` - Success (device deleted or cancelled by user)
- `1` - Error (authentication, network, device not found, etc.)

## Notes

- Deletion is immediate and cannot be undone
- Device history and logs are permanently removed
- Group assignments are cleared
- Presentation schedules are removed
- The physical device is not affected
- Re-provisioning creates a "new" device in BSN.cloud
