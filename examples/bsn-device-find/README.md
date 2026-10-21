# Find Device Across Networks

A tool to find a BrightSign player across all available BSN.cloud networks.

## Overview

This tool searches through all networks you have access to and locates a specified device by serial number. It's useful when you have multiple networks and need to quickly find which network contains a specific player.

This example demonstrates:
- Authenticating with BSN.cloud
- Retrieving all available networks
- Searching for a device across multiple networks
- Displaying device information when found
- Providing both human-friendly and JSON output formats

## Prerequisites

- BSN.cloud API credentials (client ID and secret)
- Access to at least one BSN.cloud network
- A device serial number to search for

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `BS_CLIENT_ID` | Yes | BSN.cloud API client ID |
| `BS_SECRET` | Yes | BSN.cloud API client secret |

## Usage

### Basic Usage

Find a device by serial number:

```bash
export BS_CLIENT_ID="your-client-id"
export BS_SECRET="your-client-secret"

./bin/bsn-device-find --serial UTD41X000009
```

### Verbose Output

Show detailed search progress including which networks are being searched:

```bash
./bin/bsn-device-find --serial UTD41X000009 --verbose
```

### JSON Output

Get machine-readable JSON output:

```bash
./bin/bsn-device-find --serial UTD41X000009 --json
```

Save to file:

```bash
./bin/bsn-device-find --serial UTD41X000009 --json > device-info.json
```

## Command-Line Options

| Option | Default | Description |
|--------|---------|-------------|
| `--help` | false | Display usage information |
| `--serial <string>` | | Device serial number to search for (required) |
| `--json` | false | Output as JSON |
| `--verbose` | false | Show detailed search progress |
| `--timeout <seconds>` | 30 | Request timeout in seconds |

## Output Format

### Standard Output (Device Found)

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Device Found!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📱 Device Information:
  Serial:       UTD41X000009
  Network:      Production Network (ID: 12345)
  Name:         Lobby Display
  Description:  Main lobby digital signage
  Type:         BrightSign
  Model:        HD1024
  Group:        Retail Stores

📊 Search Statistics:
  Networks searched: 3

💡 Tip: Use --json flag to get machine-readable output
```

### Standard Output (Device Not Found)

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
❌ Device Not Found
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Serial:            UTD41X000009
  Networks searched: 3

💡 Troubleshooting:
  • Verify the serial number is correct
  • Ensure the device is registered in one of your networks
  • Check that you have access to the network containing the device
  • Use --verbose flag to see which networks were searched
```

### JSON Output (Device Found)

```json
{
  "found": true,
  "serial": "UTD41X000009",
  "networkName": "Production Network",
  "networkId": 12345,
  "device": {
    "name": "Lobby Display",
    "description": "Main lobby digital signage",
    "serial": "UTD41X000009",
    "type": "BrightSign",
    "model": "HD1024",
    "group": "Retail Stores",
    "lastConnected": "2024-05-23T15:30:00Z"
  },
  "networksSearched": 3
}
```

### JSON Output (Device Not Found)

```json
{
  "found": false,
  "serial": "UTD41X000009",
  "networksSearched": 3
}
```

## Examples

### Find a Device with Verbose Output

```bash
./bin/bsn-device-find --serial UTD41X000009 --verbose
```

Output:
```
🔧 Creating BSN.cloud client...
🔐 Authenticating with BSN.cloud...
✅ Authentication successful!
📡 Retrieving available networks...
✅ Found 3 network(s) to search

🔍 Searching for device: UTD41X000009

  [1/3] Searching network: Development (ID: 11111)
        ⊘ Not found
  [2/3] Searching network: Staging (ID: 22222)
        ⊘ Not found
  [3/3] Searching network: Production Network (ID: 12345)
        ✅ Found!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✅ Device Found!
...
```

### Parse JSON Output with jq

Extract just the network name:
```bash
./bin/bsn-device-find --serial UTD41X000009 --json | jq -r '.networkName'
```

Check if device was found:
```bash
./bin/bsn-device-find --serial UTD41X000009 --json | jq -r '.found'
```

Get device model:
```bash
./bin/bsn-device-find --serial UTD41X000009 --json | jq -r '.device.model'
```

### Use in Scripts

```bash
#!/bin/bash

SERIAL="UTD41X000009"
RESULT=$(./bin/bsn-device-find --serial "$SERIAL" --json)

if echo "$RESULT" | jq -e '.found' > /dev/null; then
    NETWORK=$(echo "$RESULT" | jq -r '.networkName')
    echo "Device $SERIAL found in network: $NETWORK"
else
    echo "Device $SERIAL not found in any network"
    exit 1
fi
```

### Search for Multiple Devices

```bash
#!/bin/bash

SERIALS=("UTD41X000009" "UTD41X000010" "UTD41X000011")

for serial in "${SERIALS[@]}"; do
    echo "Searching for $serial..."
    ./bin/bsn-device-find --serial "$serial" --json > "device-${serial}.json"
done
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Device found successfully |
| 1 | Device not found or error occurred |

## Use Cases

### 1. Multi-Network Management

When managing multiple networks, quickly locate which network contains a specific device:

```bash
./bin/bsn-device-find --serial UTD41X000009
```

### 2. Device Migration

Before migrating a device, confirm which network it's currently in:

```bash
CURRENT_NETWORK=$(./bin/bsn-device-find --serial UTD41X000009 --json | jq -r '.networkName')
echo "Device is currently in: $CURRENT_NETWORK"
```

### 3. Inventory Auditing

Generate a report of devices across networks:

```bash
for serial in $(cat device-serials.txt); do
    ./bin/bsn-device-find --serial "$serial" --json
done | jq -s '.'
```

### 4. Automated Monitoring

Use in monitoring scripts to verify device presence:

```bash
if ./bin/bsn-device-find --serial UTD41X000009 --json | jq -e '.found' > /dev/null; then
    echo "OK: Device is registered"
else
    echo "CRITICAL: Device not found"
    exit 2
fi
```

## Performance

The tool searches networks sequentially and stops as soon as the device is found. Search time depends on:
- Number of networks you have access to
- Which network contains the device (earlier = faster)
- Network response time

Use `--verbose` flag to monitor search progress.

## Related Examples

- `bsn-device-list` - List all devices in a specific network
- `bsn-device-info` - Get detailed information about a device
- `bsn-device-status` - Get status information for a device
- `bsn-bdeploy-device-find` - Find B-Deploy provisioned devices

## Troubleshooting

### Error: "Failed to get networks"

- Verify your API credentials are correct
- Ensure your account has access to at least one network
- Check network connectivity

### Error: "Device Not Found"

- Verify the serial number is spelled correctly
- Ensure the device is registered in BSN.cloud
- Confirm you have access to the network containing the device
- The device might be in a network you don't have access to

### Error: "Authentication failed"

- Check that `BS_CLIENT_ID` and `BS_SECRET` are set correctly
- Verify your API credentials are still valid
- Ensure your account has the necessary permissions

## Notes

- The search stops as soon as the device is found (doesn't search all networks)
- Searches are performed sequentially, not in parallel
- The tool only searches networks you have access to
- Serial numbers are case-sensitive
- Use `--verbose` to see detailed search progress
