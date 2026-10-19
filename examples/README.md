# Examples Documentation

This directory contains 66 example programs demonstrating all SDK features. Each builds to `bin/<name>`; names follow `bsn-[<subsystem>-]<object>-<verb>`.

## Quick Start

```bash
# Set credentials
export BS_CLIENT_ID=your_client_id
export BS_SECRET=your_client_secret
export BS_NETWORK=your_network_name  # Optional

# Build all examples
make build

# Run any example with --help
./bin/bsn-device-list --help
```

## Common Environment Variables

All examples require:
- `BS_CLIENT_ID`: BSN.cloud API client ID (required)
- `BS_SECRET`: BSN.cloud API client secret (required)
- `BS_NETWORK`: Default network name (optional)

## Common Flags

Most examples support:
- `--help`: Display usage information
- `--network` / `-n`: Network name selection
- `--verbose`: Show detailed output
- `--json`: Output as JSON for scripting
- `--timeout 30`: Request timeout in seconds
- `--force` / `-y`: Skip confirmation prompts (destructive operations)

---

## Authentication Examples (2)

### bsn-auth-info
Authenticate with BSN.cloud and display curl command examples.

**Flags:**
- `--help`: Display usage information
- `--timeout 30`: Request timeout in seconds
- `--show`: Show current token from .token file without re-authenticating

**Output:** Creates `.token` file with access token, provides curl examples

**Usage:**
```bash
./bin/bsn-auth-info
./bin/bsn-auth-info --show
```

### bsn-token-test
Test authentication and token validity.

**Usage:**
```bash
./bin/bsn-token-test
```

---

## B-Deploy Setup Management (10)

### bsn-bdeploy-setup-add
Create a B-Deploy setup record using JSON configuration.

**Flags:**
- `--help`: Display usage information
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds (overrides config file)

**Required Input:** JSON configuration file (see [Configuration Files](#configuration-files))

**Output:** Setup ID for device association

**Usage:**
```bash
./bin/bsn-bdeploy-setup-add config.json
./bin/bsn-bdeploy-setup-add --verbose examples/bsn-bdeploy-setup-add/config.json
```

### bsn-bdeploy-device-associate
Associate or dissociate a device with a B-Deploy setup.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--setup-id <id>`: Setup ID to associate (required unless --dissociate or --setup-name)
- `--setup-name <name>`: Setup package name to associate (alternative to --setup-id)
- `--dissociate`: Remove setup association
- `--network <name>`: Network name
- `--username <user>`: BSN.cloud username
- `--description <desc>`: Device description
- `--name <name>`: Device name
- `--create`: Create device if it doesn't exist
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-bdeploy-device-associate --serial BS123456789 --setup-id setup-abc123
./bin/bsn-bdeploy-device-associate --serial BS123456789 --setup-name "my-retail-setup"
./bin/bsn-bdeploy-device-associate --serial BS123456789 --dissociate
```

### bsn-bdeploy-setup-delete
Delete B-Deploy setup records by ID or package name.

**Flags:**
- `--setup-id <id>`: Setup ID to delete (use this OR --setup-name)
- `--setup-name <name>`: Package name to delete (use this OR --setup-id)
- `--network <name>` / `-n`: Network name
- `--force`: Skip confirmation prompt
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
# Delete by setup ID
./bin/bsn-bdeploy-setup-delete --setup-id setup-abc123

# Delete by package name (searches for matching setup)
./bin/bsn-bdeploy-setup-delete --setup-name retail-display-v1

# Force delete without confirmation
./bin/bsn-bdeploy-setup-delete --setup-id setup-abc123 --force
```

### bsn-bdeploy-device-delete
Permanently remove a device from B-Deploy.

**Flags:**
- `--serial <serial>`: Device serial number to delete
- `--device-id <id>`: Device ID to delete (alternative)
- `--network <name>` / `-n`: Network name
- `--force`: Skip confirmation prompt
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-bdeploy-device-delete --serial BS123456789
./bin/bsn-bdeploy-device-delete --device-id 12345 --force
```

### bsn-bdeploy-device-get
Get B-Deploy device setup record by serial number.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-bdeploy-device-get --serial BS123456789
```

### bsn-bdeploy-setup-get
Retrieve a specific B-Deploy setup record.

**Flags:**
- `--setup-id <id>`: Setup ID to retrieve (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output raw JSON instead of formatted structure
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-bdeploy-setup-get --setup-id setup-abc123
./bin/bsn-bdeploy-setup-get --setup-id setup-abc123 --json
```

### bsn-bdeploy-device-list
List all B-Deploy devices on a network, with optional filtering by setup.

**Flags:**
- `--network <name>` / `-n`: Network name
- `--setup-id <id>`: Filter devices by setup ID (optional)
- `--setup-name <name>`: Filter devices by setup package name (optional)
- `--summary`: Show only summary count
- `--debug`: Show raw API response
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-bdeploy-device-list
./bin/bsn-bdeploy-device-list --summary
./bin/bsn-bdeploy-device-list --setup-id "658f1dbef1d46c829f60a14f"
./bin/bsn-bdeploy-device-list --setup-name "my-setup"
```

### bsn-bdeploy-setup-list
List all B-Deploy setup records for a network.

**Flags:**
- `--network <name>`: Network name (required)
- `--username <user>`: Filter by username
- `--package <name>`: Filter by package name
- `--verbose`: Show detailed information

**Output:** Table format with Setup ID, Package Name, Setup Type, Group, Network

**Usage:**
```bash
./bin/bsn-bdeploy-setup-list --network Production
./bin/bsn-bdeploy-setup-list --network Production --package retail
```

### bsn-bdeploy-setup-update
Update an existing B-Deploy setup record.

**Flags:**
- `--setup-id <id>`: Setup ID to update (required)
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Required Input:** JSON configuration file (same structure as add-setup)

**Usage:**
```bash
./bin/bsn-bdeploy-setup-update --setup-id setup-abc123 updated-config.json
```

### bsn-device-reg-script
Mint a registration token for a network and emit a ready-to-run player **registration script** that writes the five "opt-out" registry keys (`bsnce`, `a`, `bsnrt`, `sut`, `inp`) and reboots. The script is written to stdout for redirection; status/diagnostics go to stderr.

**Flags:**
- `--network <name>` / `-n`: Network name to join (required; overrides `BS_NETWORK`)
- `--type <brs|sh>`: Output format — `brs` (BrightScript via `roRegistrySection`) or `sh` (shell script using the on-player `registry` CLI, for insecure players). Required.
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
# write to a file with --file
./bin/bsn-device-reg-script --network "My Network" --type brs --file register-player.brs

# or redirect stdout
./bin/bsn-device-reg-script --network "My Network" --type sh  > register-player.sh
# sh is for an insecure player, run from its shell, e.g.:  bash /storage/sd/register-player.sh
```

---

## Device Management (8)

### bsn-device-info
Get real-time BrightSign device information via rDWS.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output raw JSON response
- `--verbose`: Show detailed information
- `--debug`: Enable debug mode (show HTTP requests/responses)
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-device-info --serial BS123456789
./bin/bsn-device-info --serial BS123456789 --json
```

### bsn-device-status
Get device status information.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-device-status --serial BS123456789
```

### bsn-device-errors
Retrieve device error logs.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--limit 100`: Number of error entries to retrieve
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-device-errors --serial BS123456789
./bin/bsn-device-errors --serial BS123456789 --limit 50
```

### bsn-device-downloads
View content downloads on devices.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-device-downloads --serial BS123456789
```

### bsn-device-delete
Delete devices from BSN.cloud.

**Flags:**
- `--serial <serial>`: Device serial number to delete
- `--id <id>`: Device ID to delete (alternative)
- `--network <name>` / `-n`: Network name
- `--force`: Skip confirmation
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-device-delete --serial BS123456789
./bin/bsn-device-delete --id 12345 --force
```

### bsn-device-group-change
Change a device's group assignment.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--group <id>`: Target group ID (required)
- `--network <name>` / `-n`: Network name
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-device-group-change --serial BS123456789 --group 42
```

### bsn-device-ldws-enable
Manage local Diagnostic Web Server (DWS) on devices.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--enable`: Enable local DWS
- `--disable`: Disable local DWS
- `--password <pass>`: Set DWS password
- `--network <name>` / `-n`: Network name
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-device-ldws-enable --serial BS123456789 --enable
./bin/bsn-device-ldws-enable --serial BS123456789 --password mypass
```

### bsn-device-list
List and query devices with filtering and pagination.

**Flags:**
- `--network <name>` / `-n`: Network name
- `--page-size 20`: Number of devices per page
- `--filter <expr>`: Filter expression
- `--sort <expr>`: Sort expression
- `--serial <serial>`: Get specific device by serial
- `--id <id>`: Get specific device by ID
- `--json`: Output as JSON
- `--verbose`: Show detailed information
- `--quiet`: Suppress non-essential output
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-device-list
./bin/bsn-device-list --serial BS123456789
./bin/bsn-device-list --filter 'model=XD1033' --sort 'registrationDate'
```

---

## Group Management (3)

### bsn-group-info
Retrieve BrightSign group information.

**Flags:**
- `--id <id>`: Group ID to retrieve (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-group-info --id 42
```

### bsn-group-update
Update BrightSign group information.

**Flags:**
- `--id <id>`: Group ID to update (required)
- `--name <name>`: New group name
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-group-update --id 42 --name "Retail Stores"
```

### bsn-group-delete
Delete BrightSign groups.

**Flags:**
- `--id <id>`: Group ID to delete (required)
- `--network <name>` / `-n`: Network name
- `--force`: Skip confirmation
- `--json`: Output as JSON
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-group-delete --id 42
./bin/bsn-group-delete --id 42 --force
```

---

## Subscription Management (3)

### bsn-subscription-operations
List subscription operations and permissions.

**Flags:**
- `--network <name>` / `-n`: Network name
- `--device-id <id>`: Filter by device ID
- `--json`: Output as JSON
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-subscription-operations
./bin/bsn-subscription-operations --device-id 12345
```

### bsn-subscription-count
Get count of device subscriptions on network.

**Flags:**
- `--network <name>` / `-n`: Network name
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-subscription-count
```

### bsn-subscription-list
List device subscriptions on network.

**Flags:**
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-subscription-list
```

---

## Remote DWS Operations (34)

Remote Diagnostic Web Server (rDWS) operations allow you to manage and troubleshoot BrightSign devices remotely.

### bsn-rdws-info
Retrieve player information via rDWS.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output raw JSON response
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-info --serial BS123456789
```

### bsn-rdws-reboot
Remotely reboot BrightSign devices.

**Flags:**
- `--serial <serial>`: Device serial number
- `--id <id>`: Device ID
- `--type <type>`: Reboot type (normal, crash, factoryreset, disableautorun)
- `-t`: Alias for --type
- `--network <name>` / `-n`: Network name
- `-y`: Skip confirmation prompt
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-reboot --serial BS123456789 --type normal
./bin/bsn-rdws-reboot --serial BS123456789 -t factoryreset -y
```

### bsn-rdws-reprovision
Re-provision BrightSign devices.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `-y`: Skip confirmation
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-reprovision --serial BS123456789
./bin/bsn-rdws-reprovision --serial BS123456789 -y
```

### bsn-rdws-snapshot
Capture screenshots from BrightSign devices.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--output <path>`: Output file path (default: screenshot.png)
- `--network <name>` / `-n`: Network name
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-snapshot --serial BS123456789
./bin/bsn-rdws-snapshot --serial BS123456789 --output capture.png
```

### bsn-rdws-time-get
Manage player time via rDWS.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-time-get --serial BS123456789
```

### bsn-rdws-time-set
Set player time via rDWS.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--time <time>`: Time to set (RFC3339 format or Unix timestamp)
- `--network <name>` / `-n`: Network name
- `-y`: Skip confirmation
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-time-set --serial BS123456789 --time "2025-01-15T10:00:00Z"
./bin/bsn-rdws-time-set --serial BS123456789 --time 1736935200
```

### bsn-rdws-dws-password
Get and set DWS passwords on BrightSign devices.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--set <password>`: Set DWS password
- `--network <name>` / `-n`: Network name
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-dws-password --serial BS123456789
./bin/bsn-rdws-dws-password --serial BS123456789 --set newpassword
```

### bsn-rdws-health
Check player health status.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-health --serial BS123456789
```

### bsn-rdws-local-dws
Manage local DWS on players.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--enable`: Enable local DWS
- `--disable`: Disable local DWS
- `--password <pass>`: Set DWS password
- `--network <name>` / `-n`: Network name
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-local-dws --serial BS123456789 --enable
./bin/bsn-rdws-local-dws --serial BS123456789 --password mypass
```

### File Operations

#### bsn-rdws-files-list
List files and directories on player storage.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--path <path>`: Directory path on device
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-files-list --serial BS123456789
./bin/bsn-rdws-files-list --serial BS123456789 --path /storage/sd/
```

#### bsn-rdws-files-upload
Upload a file to player storage.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--file <path>`: Local file path to upload (required)
- `--device-path <path>`: Destination path on device
- `--network <name>` / `-n`: Network name
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-files-upload --serial BS123456789 --file config.xml
./bin/bsn-rdws-files-upload --serial BS123456789 --file config.xml --device-path /storage/sd/
```

#### bsn-rdws-files-delete
Delete a file from player storage.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--path <path>`: File path on device (required)
- `--network <name>` / `-n`: Network name
- `-y`: Skip confirmation
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-files-delete --serial BS123456789 --path /storage/sd/old.xml
./bin/bsn-rdws-files-delete --serial BS123456789 --path /storage/sd/old.xml -y
```

#### bsn-rdws-files-rename
Rename a file on player storage.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--path <path>`: File path on device (required)
- `--new-name <name>`: New file name (required)
- `--network <name>` / `-n`: Network name
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-files-rename --serial BS123456789 --path /storage/sd/old.xml --new-name new.xml
```

#### bsn-rdws-files-create-folder
Create folders on player storage.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--path <path>`: Folder path to create (required)
- `--network <name>` / `-n`: Network name
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-files-create-folder --serial BS123456789 --path /storage/sd/configs/
```

### Network Operations

#### bsn-rdws-network-config
Manage network configuration on players.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-network-config --serial BS123456789
```

#### bsn-rdws-ping
Ping hosts from players.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--host <host>`: Host to ping (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-ping --serial BS123456789 --host google.com
./bin/bsn-rdws-ping --serial BS123456789 --host 8.8.8.8
```

#### bsn-rdws-dns-lookup
Perform DNS lookups on players.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--host <host>`: Host to lookup (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-dns-lookup --serial BS123456789 --host brightsignnetwork.com
```

#### bsn-rdws-diagnostics
Run network diagnostics on players.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-diagnostics --serial BS123456789
```

#### bsn-rdws-traceroute
Perform traceroute from players.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--host <host>`: Target host (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-traceroute --serial BS123456789 --host 8.8.8.8
```

#### bsn-rdws-packet-capture
Packet capture on players.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `--output <path>`: Output file path
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-packet-capture --serial BS123456789 --output capture.pcap
```

#### bsn-rdws-network-neighborhood
Discover network neighborhood from players.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-network-neighborhood --serial BS123456789
```

### System Operations

#### bsn-rdws-registry-get
Read BrightSign player registry values.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--key <key>`: Registry key to read (required)
- `--network <name>` / `-n`: Network name
- `--json`: Output as JSON
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-registry-get --serial BS123456789 --key "networking"
```

#### bsn-rdws-registry-set
Write BrightSign player registry values.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--key <key>`: Registry key to set (required)
- `--value <value>`: Registry value (required)
- `--type <type>`: Value type (string, int, binary, etc.)
- `--network <name>` / `-n`: Network name
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-registry-set --serial BS123456789 --key "mykey" --value "myvalue"
```

#### bsn-rdws-logs-get
Retrieve BrightSign player log files.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--output <path>`: Output directory for logs
- `--network <name>` / `-n`: Network name
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-logs-get --serial BS123456789 --output /tmp/logs/
```

#### bsn-rdws-crashdump-get
Retrieve BrightSign player crash dumps.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--output <path>`: Output directory for dumps
- `--network <name>` / `-n`: Network name
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-crashdump-get --serial BS123456789 --output /tmp/crashdumps/
```

#### bsn-rdws-firmware-download
Remotely download and apply firmware updates.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--url <url>`: Firmware download URL (required)
- `--network <name>` / `-n`: Network name
- `-y`: Skip confirmation
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-firmware-download --serial BS123456789 --url https://example.com/firmware.bsfw
```

#### bsn-rdws-custom-data
Send custom data to BrightSign players via UDP.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--data <data>`: Data to send (required)
- `--network <name>` / `-n`: Network name
- `--port 5000`: UDP port
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-custom-data --serial BS123456789 --data "command:refresh"
./bin/bsn-rdws-custom-data --serial BS123456789 --data "alert" --port 6000
```

#### bsn-rdws-ssh
Manage SSH access on players, including enable/disable and password management.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--get`: Get current SSH status
- `--enable`: Enable SSH
- `--disable`: Disable SSH
- `--password <password>`: Set SSH password (optional, if omitted existing password unchanged)
- `--port <port>`: SSH port (default: 22)
- `--network <name>` / `-n`: Network name (uses BS_NETWORK env var if not specified)
- `--json`: Output raw JSON response
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
# Get SSH status
./bin/bsn-rdws-ssh --serial BS123456789 --get

# Enable SSH with password
./bin/bsn-rdws-ssh --serial BS123456789 --enable --password mypassword

# Enable SSH on custom port with password
./bin/bsn-rdws-ssh --serial BS123456789 --enable --port 2222 --password secure123

# Enable SSH without changing password
./bin/bsn-rdws-ssh --serial BS123456789 --enable

# Change password on already-enabled SSH
./bin/bsn-rdws-ssh --serial BS123456789 --enable --password newpassword

# Disable SSH
./bin/bsn-rdws-ssh --serial BS123456789 --disable
```

#### bsn-rdws-telnet
Manage telnet access on players.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--enable`: Enable telnet
- `--disable`: Disable telnet
- `--network <name>` / `-n`: Network name
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-telnet --serial BS123456789 --enable
./bin/bsn-rdws-telnet --serial BS123456789 --disable
```

#### bsn-rdws-storage-reformat
Remotely reformat storage devices on players.

**Flags:**
- `--serial <serial>`: Device serial number (required)
- `--device <device>`: Storage device to reformat (sd, ssd, usb)
- `--network <name>` / `-n`: Network name (uses BS_NETWORK env var if not specified)
- `-y`: Skip confirmation
- `--verbose`: Show detailed information
- `--timeout 30`: Request timeout in seconds

**Usage:**
```bash
./bin/bsn-rdws-storage-reformat --serial BS123456789 --device sd
export BS_NETWORK="Production"
./bin/bsn-rdws-storage-reformat --serial BS123456789 --device ssd -y
```

---

## Configuration Files

### B-Deploy Setup Configuration (JSON)

Used by `bsn-bdeploy-setup-add` and `bsn-bdeploy-setup-update`:

```json
{
  "networkName": "Production Network",
  "username": "admin@example.com",
  "packageName": "retail-display-v1",
  "setupType": "standalone",
  "timeZone": "America/New_York",
  "bsnGroupName": "Default",
  "network": {
    "timeServers": ["http://time.brightsignnetwork.com"],
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
  },
  "timeout": 30
}
```

**Required Fields:**
- `networkName`: BSN.cloud network name (must match exactly)
- `username`: Your BSN.cloud username/email
- `packageName`: Unique identifier for this setup configuration
- `setupType`: Setup type: `standalone` or `bsn`
- `network.interfaces`: At least one network interface configuration

**Optional Fields:**
- `timeZone`: Player timezone (default: `America/New_York`)
- `bsnGroupName`: BSN group name (default: `Default`)
- `timeout`: HTTP request timeout in seconds (default: `30`)

**Example configurations** can be found in:
- `examples/bsn-bdeploy-setup-add/config.json` - Basic Ethernet setup
- `examples/bsn-bdeploy-setup-add/config-annotated.json` - Detailed with comments
- `examples/bsn-bdeploy-setup-add/config-wifi-example.json` - WiFi configuration

---

## Common Patterns

### Network Selection
Most examples support flexible network selection:
- Use `--network` / `-n` flag to specify network name
- Falls back to `BS_NETWORK` environment variable
- If multiple networks available, prompts for interactive selection
- If only one network, uses it automatically

### Output Formats
- Default: Human-readable formatted output
- `--json`: Structured JSON for scripting and automation
- `--verbose`: Additional details (timestamps, IDs, debug info)
- `--debug`: HTTP request/response logging (some examples)

### Confirmation Prompts
Destructive operations (delete, reformat, factory reset) require confirmation:
- Interactive prompt asks for verification
- Use `--force` or `-y` flag to skip confirmation
- Useful for automation and scripting

### Authentication
All examples use OAuth2 Client Credentials flow:
- Credentials from environment: `BS_CLIENT_ID`, `BS_SECRET`
- Token obtained automatically by SDK
- Token cached and reused until expiration
- No manual token management required

### Pagination & Filtering
List operations support flexible querying:
- `--page-size N`: Control results per page
- `--filter <expr>`: OData-like filter expressions
- `--sort <expr>`: Sort by field(s)
- `--all`: Auto-paginate through all results

**Filter Expression Examples:**
```bash
# Exact match
--filter "model=XD1033"

# Contains
--filter "name contains 'retail'"

# Multiple conditions (AND)
--filter "mediaType eq 'Video' and fileSize lt 1000000"

# Comparison operators
--filter "registrationDate gt '2025-01-01'"
```

---

## Typical Workflows

### 1. Device Provisioning
Complete workflow for adding new devices:

```bash
# Step 1: Create B-Deploy setup record
./bin/bsn-bdeploy-setup-add examples/bsn-bdeploy-setup-add/config.json
# Output: setup-id (e.g., setup-abc123)

# Step 2: Associate device with setup
./bin/bsn-bdeploy-device-associate --serial BS123456789 --setup-id setup-abc123

# Step 3: Device boots and receives setup automatically
# Monitor with:
./bin/bsn-device-status --serial BS123456789
```

### 2. Remote Device Management
Monitor and manage devices remotely:

```bash
# Get device info
./bin/bsn-rdws-info --serial BS123456789

# Check health status
./bin/bsn-rdws-health --serial BS123456789

# Capture screenshot
./bin/bsn-rdws-snapshot --serial BS123456789 --output current.png

# Reboot if needed
./bin/bsn-rdws-reboot --serial BS123456789 --type normal

# View logs for troubleshooting
./bin/bsn-rdws-logs-get --serial BS123456789 --output /tmp/logs/
```

### 3. Bulk Device Operations
Automate operations across multiple devices:

```bash
# List all devices and save to file
./bin/bsn-device-list --json > devices.json

# Process each device (example script)
for serial in $(jq -r '.items[].serial' devices.json); do
  echo "Processing $serial"
  ./bin/bsn-rdws-health --serial $serial --json
done
```

---

## Exit Codes

All examples use standard exit codes:
- **0**: Success
- **1**: Error (invalid arguments, authentication failed, operation failed)

Use in scripts:
```bash
if ./bin/bsn-device-info --serial BS123456789; then
  echo "Device is accessible"
else
  echo "Device error or offline"
fi
```

---

## Building Examples

```bash
# Build all examples into bin/
make build

# List all example programs
make list-examples

# Install all examples to ~/bin (override with INSTALLDIR=...)
make install

# Clean build artifacts
make clean
```

---

## Adding New Examples

When creating new examples:

1. Create directory under `examples/`
2. Follow naming convention: `main-<feature>` or `<service>-<operation>`
3. Include `--help` flag with clear usage information
4. Support common flags (`--network`, `--verbose`, `--json`, `--timeout`)
5. Use consistent error handling and exit codes
6. Add entry to this README
7. Test thoroughly with different scenarios

Example structure:
```
examples/
└── main-new-feature/
    ├── main.go
    └── README.md  # Optional for complex examples
```
