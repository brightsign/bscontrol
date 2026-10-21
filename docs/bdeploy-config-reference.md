# B-Deploy Setup Configuration Reference

This document describes all available configuration fields for B-Deploy setup records using the `bsn-bdeploy-setup-add` command.

## Configuration File Format

B-Deploy setup configurations are JSON files that support the complete B-Deploy API v2.0.0 and v3.0.0 specification. This SDK now supports **all 80+ configuration fields** available in the B-Deploy API.

**Reference Example:** See `testdata/lfn-control.json` for a complete working example with all fields.

## Setup Types

The `setupType` field determines how the player provisions and operates. This is one of the most important configuration decisions.

| Setup Type | Description | Use Case | BSN.cloud Connection |
|------------|-------------|----------|---------------------|
| `bsn` | **BrightSign Network** | Standard BSN.cloud deployment with centrally managed content and presentations | Required - player downloads content from BSN.cloud |
| `lfn` | **Local File Network** | Player serves content from local network storage (NAS, SMB shares, etc.) | Optional - for monitoring and diagnostics only |
| `standalone` | **Standalone** | Player runs independently without network content management | Optional - minimal or no cloud connectivity |
| `sfn` | **Simple File Network** | Player downloads content from a web server via HTTP/HTTPS | Optional - connects to custom web servers |
| `partnerApplication` | **Partner Application** | Player provisioned through partner integration or third-party application | Varies - depends on partner configuration |

**Most Common:**
- Use `bsn` for typical BSN.cloud deployments where you manage content through the BSN.cloud web interface
- Use `lfn` for local content serving scenarios (kiosks, digital signage with local content servers)

---

## Required Core Fields

### `bDeploy` (required)
**Type:** `object`

Contains the core B-Deploy provisioning information.

#### `bDeploy.username` (required)
**Type:** `string`
**Description:** BSN.cloud username (email address) associated with the network
**Example:** `"admin@example.com"`, `"user@company.com"`

#### `bDeploy.networkName` (required)
**Type:** `string`
**Description:** BSN.cloud network name where the setup will be created
**Example:** `"Production"`, `"gch-control-only"`

#### `bDeploy.packageName` (required)
**Type:** `string`
**Description:** Unique identifier/name for this setup configuration
**Example:** `"production-retail-2024"`, `"FullSetup2"`

#### `bDeploy.client` (optional)
**Type:** `string`
**Description:** Client identifier for tracking/organization
**Default:** `""` (empty)

### `setupType` (required)
**Type:** `string`
**Valid Values:** `"bsn"`, `"lfn"`, `"standalone"`, `"sfn"`, `"partnerApplication"`
**Description:** Determines how the player provisions and operates (see Setup Types section above)
**Example:** `"lfn"`

### `version` (optional)
**Type:** `string`
**Valid Values:** `"2.0.0"`, `"3.0.0"`
**Default:** `"3.0.0"`
**Description:** B-Deploy API version to use
**Note:** SDK automatically sets this if not provided

---

## Device Configuration

### `deviceName` (optional)
**Type:** `string`
**Description:** Friendly name for the device
**Example:** `"Lobby Display 1"`, `"Conference Room A"`

### `deviceDescription` (optional)
**Type:** `string`
**Description:** Longer description of the device's purpose/location
**Example:** `"Main lobby entrance display showing company announcements"`

### `unitNamingMethod` (optional)
**Type:** `string`
**Valid Values:** `"appendUnitIDToUnitName"`, `"useSerialNumber"`, etc.
**Default:** `"appendUnitIDToUnitName"`
**Description:** How to generate unique device names when multiple devices use same setup

### `timeZone` (optional)
**Type:** `string`
**Default:** `"America/New_York"`
**Description:** IANA timezone identifier for the player
**Examples:**
- `"America/Los_Angeles"` - Pacific Time
- `"America/Chicago"` - Central Time
- `"America/Denver"` - Mountain Time
- `"Europe/London"` - GMT/BST
- `"Asia/Tokyo"` - Japan Standard Time
- `"PST"` (legacy format, prefer full IANA names)

### `bsnGroupName` (optional)
**Type:** `string`
**Default:** `"Default"`
**Description:** BSN.cloud group to assign provisioned players to
**Example:** `"Retail Stores"`, `"Corporate Lobbies"`

---

## Firmware & Debugging

### `firmwareUpdateType` (optional)
**Type:** `string`
**Valid Values:** `"standard"`, `"latest"`, `"specific"`
**Default:** `"standard"`
**Description:** Firmware update policy for the player

### `enableSerialDebugging` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable serial console debugging output (UART at 115200 baud)
**Use Case:** Development and troubleshooting

### `enableSystemLogDebugging` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable detailed system-level logging
**Use Case:** Advanced troubleshooting and diagnostics

---

## DWS (Diagnostic Web Server)

The Diagnostic Web Server provides a local web interface for player diagnostics and configuration.

### `dwsEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable local DWS access at `http://<player-ip>:8008/`

### `dwsPassword` (optional)
**Type:** `string`
**Default:** `""` (empty - no password required)
**Description:** Password to access local DWS
**Special Value:** `"none"` - explicitly no password required
**Security:** Use a strong password for production deployments

### `remoteDwsEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable remote DWS access via BSN.cloud at `https://ws.bsn.cloud/`
**Use Case:** Remote diagnostics and troubleshooting without direct network access

---

## LWS (Local Web Server)

The Local Web Server provides local access to player status, content, and configuration.

### `lwsEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable local web server

### `lwsConfig` (optional)
**Type:** `string`
**Valid Values:** `"status"`, `"content"`, `"diagnostic"`
**Default:** `"status"`
**Description:** LWS configuration mode

### `lwsUserName` (optional)
**Type:** `string`
**Default:** `""` (no username required)
**Description:** Username for LWS authentication

### `lwsPassword` (optional)
**Type:** `string`
**Default:** `""` (no password required)
**Description:** Password for LWS authentication

### `lwsEnableUpdateNotifications` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable notifications when player updates content

---

## BSN.cloud Connection Settings

### `timeBetweenNetConnects` (optional)
**Type:** `number`
**Unit:** seconds
**Default:** `300` (5 minutes)
**Description:** How often player connects to BSN.cloud to check for content updates

### `timeBetweenHeartbeats` (optional)
**Type:** `number`
**Unit:** seconds
**Default:** `900` (15 minutes)
**Description:** How often player sends heartbeat/health reports to BSN.cloud

---

## Simple File Network (SFN)

Simple File Networking (`setupType: "sfn"`) configures a player to download
content from a standard HTTP/HTTPS web server instead of BSN.cloud. The player
polls the web folder URL on a schedule and downloads updated content
automatically.

SFN setups share the same template structure as BSN setups. The differences are
the `setupType` value and the four SFN-specific fields below, which **must** be
configured for a working SFN deployment.

### SFN-Specific Fields

#### `sfnWebFolderUrl` (required for SFN)
**Type:** `string`
**Default:** `""` (empty)
**Description:** URL to the web folder containing content files. The player
polls this URL on the `timeBetweenNetConnects` interval to check for updates.
**Example:** `"http://content.example.com/player1/"`, `"https://cdn.company.com/signage/lobby/"`

#### `sfnUserName` (required if authentication enabled)
**Type:** `string`
**Default:** `""` (empty)
**Description:** Username for HTTP basic authentication to the web folder
**Example:** `"wfusername"`

#### `sfnPassword` (required if authentication enabled)
**Type:** `string`
**Default:** `""` (empty)
**Description:** Password for HTTP basic authentication to the web folder
**Security:** Stored in plain text in the setup record. Use HTTPS to protect
credentials in transit.
**Example:** `"PaSsWoRd#1!"`

#### `sfnEnableBasicAuthentication` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable HTTP basic authentication when downloading content from
the web folder. Set to `true` and provide `sfnUserName` / `sfnPassword` if
your web server requires authentication.

### SFN Behavioral Differences

When `setupType` is `"sfn"`, several other fields behave differently compared
to BSN setups:

| Field | Typical BSN Value | Typical SFN Value | Reason |
|---|---|---|---|
| `setupType` | `"bsn"` | `"sfn"` | Switches content source to web folder |
| `bsnCloudEnabled` | `true` | `true` | Still needed for monitoring/diagnostics |
| `inheritNetworkProperties` | `false` | `true` | SFN setups typically inherit network config |
| `timeBetweenNetConnects` | `300` | `300` | Controls web folder polling interval |
| Logging fields | `true` | `false` | SFN deployments often disable logging |

### SFN Example

Minimal SFN setup with authenticated web folder:

```json
{
  "bDeploy": {
    "username": "admin@example.com",
    "networkName": "Production",
    "packageName": "lobby-sfn-setup",
    "client": "bacon"
  },
  "setupType": "sfn",
  "sfnWebFolderUrl": "https://content.example.com/lobby/",
  "sfnUserName": "player",
  "sfnPassword": "secret",
  "sfnEnableBasicAuthentication": true,
  "timeBetweenNetConnects": 300,
  "deviceName": "Lobby Display",
  "deviceDescription": "Main entrance SFN display",
  "bsnGroupName": "Default",
  "timeZone": "PST",
  "bsnCloudEnabled": true,
  "inheritNetworkProperties": true,
  "network": {
    "timeServers": ["http://time.brightsignnetwork.com"],
    "interfaces": [
      {
        "id": "wired_eth0",
        "name": "eth0",
        "type": "Ethernet",
        "proto": "DHCPv4",
        "ip": [],
        "gateway": null,
        "dns": [],
        "contentDownloadEnabled": true,
        "healthReportingEnabled": true
      }
    ]
  }
}
```

### SFN vs BSN Template Usage

The `DefaultSetupPackageTemplateMaster.json` template can be used for SFN
setups by passing `--setup-type sfn` to `render-setup-template`. After
rendering, manually set the SFN-specific fields (`sfnWebFolderUrl`,
`sfnUserName`, `sfnPassword`, `sfnEnableBasicAuthentication`) in the output
JSON before posting via `bsn-bdeploy-setup-add-raw`, since these fields are
deployment-specific and not templated.

```bash
# Render the template for SFN
bin/bsn-render-setup-template \
  --setup-type sfn \
  --package-name "lobby-sfn" \
  --network "Production" \
  templates/DefaultSetupPackageTemplateMaster.json > setup.json

# Inject SFN-specific fields with jq
jq '.sfnWebFolderUrl = "https://content.example.com/lobby/" |
    .sfnUserName = "player" |
    .sfnPassword = "secret" |
    .sfnEnableBasicAuthentication = true |
    .inheritNetworkProperties = true' setup.json > setup-sfn.json

# Post to BSN.cloud
bin/bsn-bdeploy-setup-add-raw setup-sfn.json
```

See also: [B-Deploy Template Guide](bdeploy-template.md) for full template
documentation.

---

## Local File Networking (LFN)

Local File Networking (`setupType: "lfn"`) configures a player to serve content
from local network storage (NAS, SMB shares, USB, or a local content server)
rather than BSN.cloud or a web folder. The player still connects to BSN.cloud
for monitoring, diagnostics, and remote management, but content delivery is
handled locally.

LFN and SFN setups use the exact same template structure. The only structural
difference is the `setupType` value. Comparing the official
`DefaultLocalFileNetworkingSetupPackageV3.json` and
`DefaultSimpleFileNetworkingSetupPackageV3.json` reference files confirms they
are identical except for `setupType` and the SFN web folder fields.

### LFN-Specific Configuration

LFN has **no unique fields** beyond `setupType`. The four SFN web folder fields
(`sfnWebFolderUrl`, `sfnUserName`, `sfnPassword`,
`sfnEnableBasicAuthentication`) are present in LFN setups but left empty since
content is delivered through local mechanisms, not HTTP polling.

The only field that must be set for LFN:

#### `setupType` (required)
**Value:** `"lfn"`
**Description:** Switches the player to local file networking mode. Content is
served from local storage or a local content server rather than BSN.cloud or a
web folder.

### LFN vs SFN vs BSN

| Aspect | BSN (`bsn`) | SFN (`sfn`) | LFN (`lfn`) |
|---|---|---|---|
| Content source | BSN.cloud | HTTP/HTTPS web folder | Local storage / network share |
| `sfnWebFolderUrl` | empty | **required** (web folder URL) | empty |
| `sfnUserName` / `sfnPassword` | empty | optional (if auth needed) | empty |
| `sfnEnableBasicAuthentication` | `false` | `true` if auth needed | `false` |
| BSN.cloud connection | required (content + mgmt) | optional (mgmt only) | optional (mgmt only) |
| `bsnCloudEnabled` | `true` | `true` | `true` |
| `inheritNetworkProperties` | `false` | `true` | `true` |
| Typical logging | enabled | disabled | disabled |

### LFN Example

Minimal LFN setup:

```json
{
  "bDeploy": {
    "username": "admin@example.com",
    "networkName": "Production",
    "packageName": "lobby-lfn-setup",
    "client": "bacon"
  },
  "setupType": "lfn",
  "deviceName": "Lobby Display",
  "deviceDescription": "Main entrance LFN display",
  "bsnGroupName": "Default",
  "timeZone": "PST",
  "bsnCloudEnabled": true,
  "inheritNetworkProperties": true,
  "remoteDwsEnabled": true,
  "lwsEnabled": true,
  "lwsConfig": "status",
  "timeBetweenNetConnects": 300,
  "network": {
    "timeServers": ["http://time.brightsignnetwork.com"],
    "interfaces": [
      {
        "id": "wired_eth0",
        "name": "eth0",
        "type": "Ethernet",
        "proto": "DHCPv4",
        "ip": [],
        "gateway": null,
        "dns": [],
        "contentDownloadEnabled": true,
        "healthReportingEnabled": true
      }
    ]
  }
}
```

### LFN Template Usage

Use `render-setup-template` with `--setup-type lfn`:

```bash
bin/bsn-render-setup-template \
  --setup-type lfn \
  --package-name "lobby-lfn" \
  --network "Production" \
  templates/DefaultSetupPackageTemplateMaster.json > setup-lfn.json

bin/bsn-bdeploy-setup-add-raw setup-lfn.json
```

No post-rendering `jq` injection is needed for LFN (unlike SFN) since there
are no LFN-specific fields to populate.

---

## Partner Application (partnerApplication)

Partner Application (`setupType: "partnerApplication"`) configures a player to
be managed by a third-party CMS (Content Management System) rather than
BSN.cloud or local content mechanisms. The player still registers with
BSN.cloud for provisioning and remote diagnostics, but content delivery and
scheduling are handled entirely by the external partner application.

### partnerApplication-Specific Configuration

Partner Application has **no unique fields** beyond `setupType`. It uses the
same template structure as BSN, SFN, and LFN setups. The only required change
is setting the setup type:

#### `setupType` (required)
**Value:** `"partnerApplication"`
**Description:** Switches the player to partner application mode. An external
CMS controls content delivery and presentation scheduling.

### Typical partnerApplication Defaults

Compared to BSN setups, partner application setups typically differ in these
ways:

| Field | Typical BSN Value | Typical partnerApplication Value | Reason |
|---|---|---|---|
| `setupType` | `"bsn"` | `"partnerApplication"` | External CMS manages content |
| `lwsEnabled` | `true` | `false` | Partner CMS provides its own local interface |
| `dwsPassword` | `"§default"` | `""` | No default DWS password |
| `timeBetweenHeartbeats` | `900` | (omitted) | Partner CMS may manage its own heartbeat |
| `heartbeatsRestricted` | `false` | (omitted) | Heartbeat windowing not applicable |
| Logging fields | varies | `false` | Partner CMS typically handles its own logging |
| `remoteSnapshotJpegQualityLevel` | `100` | `50` | Lower quality default |

### partnerApplication vs Other Setup Types

| Aspect | BSN | SFN | LFN | partnerApplication |
|---|---|---|---|---|
| Content source | BSN.cloud | HTTP web folder | Local storage | External CMS |
| Content management | BSN.cloud UI | Manual web folder | Manual local files | Partner CMS |
| BSN.cloud role | Content + mgmt | Monitoring only | Monitoring only | Provisioning + monitoring |
| LWS | enabled | enabled | enabled | **disabled** |
| Unique fields | none | `sfnWebFolderUrl`, auth | none | none |

### partnerApplication Example

Minimal partner application setup:

```json
{
  "bDeploy": {
    "username": "admin@example.com",
    "networkName": "Production",
    "packageName": "partner-cms-setup",
    "client": "bacon"
  },
  "setupType": "partnerApplication",
  "deviceName": "Partner Display",
  "deviceDescription": "Managed by external CMS",
  "bsnGroupName": "Default",
  "timeZone": "PST",
  "bsnCloudEnabled": true,
  "inheritNetworkProperties": true,
  "remoteDwsEnabled": true,
  "dwsEnabled": false,
  "lwsEnabled": false,
  "timeBetweenNetConnects": 300,
  "network": {
    "timeServers": ["http://time.brightsignnetwork.com"],
    "interfaces": [
      {
        "id": "wired_eth0",
        "name": "eth0",
        "type": "Ethernet",
        "proto": "DHCPv4",
        "ip": [],
        "gateway": null,
        "dns": [],
        "contentDownloadEnabled": true,
        "healthReportingEnabled": true
      }
    ]
  }
}
```

### partnerApplication Template Usage

Use `render-setup-template` with `--setup-type partnerApplication`:

```bash
bin/bsn-render-setup-template \
  --setup-type partnerApplication \
  --package-name "partner-cms" \
  --network "Production" \
  templates/DefaultSetupPackageTemplateMaster.json > setup.json

# Disable LWS (typical for partner app setups)
jq '.lwsEnabled = false' setup.json > setup-partner.json

bin/bsn-bdeploy-setup-add-raw setup-partner.json
```

---

## Logging Configuration

### `playbackLoggingEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Log playback events (media start/stop, transitions)

### `eventLoggingEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Log custom events from presentations

### `diagnosticLoggingEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Log diagnostic events and errors

### `stateLoggingEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Log state machine transitions

### `variableLoggingEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Log variable changes and script output

### `uploadLogFilesAtBoot` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Upload log files to BSN.cloud when player boots

### `uploadLogFilesAtSpecificTime` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Upload log files at a scheduled time

### `uploadLogFilesTime` (optional)
**Type:** `number`
**Unit:** hour (0-23)
**Default:** `0`
**Description:** Hour to upload logs if `uploadLogFilesAtSpecificTime` is enabled
**Example:** `3` for 3:00 AM

### `logHandlerUrl` (optional)
**Type:** `string`
**Description:** Custom URL for log uploads (instead of BSN.cloud)
**Example:** `"https://logs.company.com/endpoint"`

---

## Remote Snapshots (Screenshots)

Remote snapshots allow BSN.cloud to capture screenshots from the player.

### `enableRemoteSnapshot` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable remote screenshot capture

### `remoteSnapshotInterval` (optional)
**Type:** `number`
**Unit:** minutes
**Default:** `15`
**Range:** `1` to `1440` (24 hours)
**Description:** How often to capture screenshots

### `remoteSnapshotMaxImages` (optional)
**Type:** `number`
**Default:** `10`
**Description:** Maximum number of screenshots to keep

### `remoteSnapshotJpegQualityLevel` (optional)
**Type:** `number`
**Range:** `1` to `100`
**Default:** `85`
**Description:** JPEG quality level (higher = better quality, larger files)

### `remoteSnapshotScreenOrientation` (optional)
**Type:** `string`
**Valid Values:** `"Landscape"`, `"Portrait"`
**Default:** `"Landscape"`
**Description:** Screenshot orientation

### `remoteSnapshotHandlerUrl` (optional)
**Type:** `string`
**Description:** Custom URL for screenshot uploads
**Default:** `""` (uploads to BSN.cloud)

---

## Device Screenshots

Local screenshot capture (different from remote snapshots).

### `deviceScreenShotsEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable device-side screenshot capture

### `deviceScreenShotsInterval` (optional)
**Type:** `number`
**Unit:** seconds
**Default:** `300`
**Description:** Screenshot capture interval

### `deviceScreenShotsCountLimit` (optional)
**Type:** `number`
**Default:** `10`
**Description:** Maximum screenshots to store locally

### `deviceScreenShotsQuality` (optional)
**Type:** `number`
**Range:** `1` to `100`
**Default:** `85`
**Description:** JPEG quality level

### `deviceScreenShotsOrientation` (optional)
**Type:** `string`
**Valid Values:** `"Landscape"`, `"Portrait"`
**Default:** `"Landscape"`
**Description:** Screenshot orientation

---

## Display Settings

### `idleScreenColor` (optional)
**Type:** `object`
**Description:** RGBA color for idle screen
**Default:** Black `{"r": 0, "g": 0, "b": 0, "a": 1}`
**Format:**
```json
{
  "r": 0,    // Red: 0-255
  "g": 0,    // Green: 0-255
  "b": 0,    // Blue: 0-255
  "a": 1     // Alpha: 0-1
}
```

### `useCustomSplashScreen` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Use custom splash screen instead of default BrightSign logo

---

## Network Configuration - Wired Interface

### `useDHCP` (optional)
**Type:** `boolean`
**Default:** `true`
**Description:** Use DHCP for wired interface (eth0)

### `staticIPAddress` (optional)
**Type:** `string`
**Description:** Static IP address for wired interface
**Example:** `"192.168.1.100"`
**Required if:** `useDHCP` is `false`

### `subnetMask` (optional)
**Type:** `string`
**Description:** Subnet mask for wired interface
**Example:** `"255.255.255.0"`

### `gateway` (optional)
**Type:** `string`
**Description:** Default gateway for wired interface
**Example:** `"192.168.1.1"`

### `dns1`, `dns2`, `dns3` (optional)
**Type:** `string`
**Description:** DNS servers for wired interface
**Example:** `"8.8.8.8"`, `"8.8.4.4"`, `"1.1.1.1"`

---

## Network Configuration - Wireless Interface

### `useWireless` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable wireless networking

### `ssid` (optional)
**Type:** `string`
**Description:** WiFi network SSID
**Required if:** `useWireless` is `true`

### `passphrase` (optional)
**Type:** `string`
**Description:** WiFi network password
**Security:** Stored in plain text - use with caution

### `useDHCP_2` (optional)
**Type:** `boolean`
**Default:** `true`
**Description:** Use DHCP for wireless interface

### `staticIPAddress_2`, `subnetMask_2`, `gateway_2` (optional)
**Type:** `string`
**Description:** Static IP configuration for wireless interface
**Format:** Same as wired interface

### `dns1_2`, `dns2_2`, `dns3_2` (optional)
**Type:** `string`
**Description:** DNS servers for wireless interface

---

## Network Data Types - Wired

Control which types of data can be transferred over the wired interface.

### `contentDataTypeEnabledWired` (optional)
**Type:** `boolean`
**Default:** `true`
**Description:** Allow content downloads over wired interface

### `textFeedsDataTypeEnabledWired` (optional)
**Type:** `boolean`
**Default:** `true`
**Description:** Allow text feed updates over wired interface

### `healthDataTypeEnabledWired` (optional)
**Type:** `boolean`
**Default:** `true`
**Description:** Allow health reporting over wired interface

### `mediaFeedsDataTypeEnabledWired` (optional)
**Type:** `boolean`
**Default:** `true`
**Description:** Allow media feed updates over wired interface

### `logUploadsXfersEnabledWired` (optional)
**Type:** `boolean`
**Default:** `true`
**Description:** Allow log uploads over wired interface

---

## Network Data Types - Wireless

Same controls for wireless interface.

### `contentDataTypeEnabledWireless` (optional)
**Type:** `boolean`
**Default:** `true`

### `textFeedsDataTypeEnabledWireless` (optional)
**Type:** `boolean`
**Default:** `true`

### `healthDataTypeEnabledWireless` (optional)
**Type:** `boolean`
**Default:** `true`

### `mediaFeedsDataTypeEnabledWireless` (optional)
**Type:** `boolean`
**Default:** `true`

### `logUploadsXfersEnabledWireless` (optional)
**Type:** `boolean`
**Default:** `true`

---

## Rate Limiting - Wired Interface

Control bandwidth usage for the wired interface.

### `rateLimitModeOutsideWindow` (optional)
**Type:** `string`
**Valid Values:** `"default"`, `"unlimited"`, `"limited"`
**Default:** `"default"`
**Description:** Rate limiting mode outside content download window

### `rateLimitRateOutsideWindow` (optional)
**Type:** `number`
**Unit:** kbps (kilobits per second)
**Default:** `0` (no limit)
**Description:** Bandwidth limit outside download window

### `rateLimitModeInWindow` (optional)
**Type:** `string`
**Valid Values:** `"default"`, `"unlimited"`, `"limited"`
**Default:** `"unlimited"`
**Description:** Rate limiting mode inside content download window

### `rateLimitRateInWindow` (optional)
**Type:** `number`
**Unit:** kbps
**Default:** `0`
**Description:** Bandwidth limit inside download window

### `rateLimitModeInitialDownloads` (optional)
**Type:** `string`
**Valid Values:** `"default"`, `"unlimited"`, `"limited"`
**Default:** `"unlimited"`
**Description:** Rate limiting for initial content downloads

### `rateLimitRateInitialDownloads` (optional)
**Type:** `number`
**Unit:** kbps
**Default:** `0`
**Description:** Bandwidth limit for initial downloads

---

## Rate Limiting - Wireless Interface

Same rate limiting controls for wireless interface (suffix `_2`).

### `rateLimitModeOutsideWindow_2` (optional)
**Type:** `string`
**Valid Values:** `"default"`, `"unlimited"`, `"limited"`
**Default:** `"default"`

### `rateLimitRateOutsideWindow_2` (optional)
**Type:** `number`
**Unit:** kbps
**Default:** `0`

### `rateLimitModeInWindow_2` (optional)
**Type:** `string`
**Valid Values:** `"default"`, `"unlimited"`, `"limited"`
**Default:** `"unlimited"`

### `rateLimitRateInWindow_2` (optional)
**Type:** `number`
**Unit:** kbps
**Default:** `0`

### `rateLimitModeInitialDownloads_2` (optional)
**Type:** `string`
**Valid Values:** `"default"`, `"unlimited"`, `"limited"`
**Default:** `"unlimited"`

### `rateLimitRateInitialDownloads_2` (optional)
**Type:** `number`
**Unit:** kbps
**Default:** `0`

---

## Network Priority & Diagnostics

### `networkConnectionPriority` (optional)
**Type:** `string`
**Valid Values:** `"wired"`, `"wireless"`
**Default:** `"wired"`
**Description:** Which interface to prefer when both are available

### `networkDiagnosticsEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable network diagnostics and testing

### `testEthernetEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable ethernet connectivity testing

### `testWirelessEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable wireless connectivity testing

### `testInternetEnabled` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Enable internet connectivity testing

---

## Proxy Configuration

### `useProxy` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Use HTTP proxy server

### `proxyAddress` (optional)
**Type:** `string`
**Description:** Proxy server address
**Example:** `"proxy.company.com"`, `"192.168.1.50"`
**Required if:** `useProxy` is `true`

### `proxyPort` (optional)
**Type:** `number`
**Default:** `0`
**Description:** Proxy server port
**Example:** `8080`, `3128`

---

## Hostname Configuration

### `specifyHostname` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Set a custom hostname for the player

### `hostname` (optional)
**Type:** `string`
**Description:** Custom hostname
**Example:** `"lobby-display-1"`
**Required if:** `specifyHostname` is `true`

---

## BrightWall Configuration

BrightWall enables synchronized multi-screen video walls.

### `BrightWallName` (optional)
**Type:** `string`
**Description:** BrightWall configuration/wall name
**Example:** `"LobbyWall"`

### `BrightWallScreenNumber` (optional)
**Type:** `string`
**Description:** Screen number in the wall
**Example:** `"1"`, `"2"`, `"3x2"` (row x column)

---

## Download & Heartbeat Time Windows

Restrict when content downloads and heartbeats can occur.

### `contentDownloadsRestricted` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Restrict content downloads to specific time window

### `contentDownloadRangeStart` (optional)
**Type:** `number`
**Unit:** minutes from midnight
**Default:** `0`
**Description:** Start of content download window
**Example:** `180` (3:00 AM), `1320` (10:00 PM)

### `contentDownloadRangeEnd` (optional)
**Type:** `number`
**Unit:** minutes from midnight
**Default:** `0`
**Description:** End of content download window
**Example:** `360` (6:00 AM)

### `heartbeatsRestricted` (optional)
**Type:** `boolean`
**Default:** `false`
**Description:** Restrict heartbeats to specific time window

### `heartbeatsRangeStart` (optional)
**Type:** `number`
**Unit:** minutes from midnight
**Default:** `0`
**Description:** Start of heartbeat window

### `heartbeatsRangeEnd` (optional)
**Type:** `number`
**Unit:** minutes from midnight
**Default:** `0`
**Description:** End of heartbeat window

---

## Network Hosts

### `networkHosts` (optional)
**Type:** `array of strings`
**Default:** `[]`
**Description:** Custom network hosts for DNS resolution
**Example:** `["host1.local", "nas.company.com"]`

---

## Time Server (Legacy)

### `timeServer` (optional)
**Type:** `string`
**Default:** `"http://time.brightsignnetwork.com"`
**Description:** NTP time server URL
**Note:** Prefer using `network.timeServers` array for v3.0.0 configurations

---

## Network Configuration (v3.0.0 Format)

### `network` (optional)
**Type:** `object`
**Description:** Network configuration in v3.0.0 format

#### `network.timeServers` (optional)
**Type:** `array of strings`
**Description:** NTP time server URLs
**Example:**
```json
[
  "http://time.brightsignnetwork.com",
  "http://time.nist.gov"
]
```

#### `network.proxy` (optional)
**Type:** `string`
**Description:** Proxy server URL
**Example:** `"http://proxy.company.com:8080"`

#### `network.proxyBypass` (optional)
**Type:** `string`
**Description:** Proxy bypass list
**Example:** `"localhost,127.0.0.1,*.local"`

#### `network.interfaces` (optional)
**Type:** `array of objects`
**Description:** Network interface configurations

**Interface Object:**
```json
{
  "id": "wired_eth0",
  "name": "eth0",
  "type": "Ethernet",
  "proto": "DHCPv4",
  "contentDownloadEnabled": true,
  "healthReportingEnabled": true
}
```

**Interface Fields:**
- `id` - Interface identifier (`"wired_eth0"`, `"wifi"`)
- `name` - Linux interface name (`"eth0"`, `"wlan0"`)
- `type` - Interface type (`"Ethernet"`, `"WiFi"`)
- `proto` - Protocol (`"DHCPv4"`, `"Static"`, `"DHCPv6"`)
- `contentDownloadEnabled` - Allow content downloads
- `healthReportingEnabled` - Allow health reporting

---

## USB Updates

### `usbUpdatePassword` (optional)
**Type:** `string`
**Description:** Password required for USB-based updates
**Default:** `""` (no password required)
**Security:** Set a password to prevent unauthorized USB updates

---

## Device Registration Token

### `bsnDeviceRegistrationTokenEntity` (auto-generated)
**Type:** `object`
**Description:** BSN.cloud device registration token
**Note:** **Automatically generated by the SDK** - you do NOT need to include this in your config file

If you want to provide your own pre-generated token, use this structure:
```json
{
  "token": "long-token-string...",
  "scope": "cert",
  "validFrom": "2025-01-13T20:00:00.000Z",
  "validTo": "2027-01-13T20:00:00.000Z"
}
```

---

## SDK-Specific Fields

### `timeout` (optional)
**Type:** `number`
**Unit:** seconds
**Default:** `30`
**Range:** `1` to `600`
**Description:** HTTP request timeout for API calls
**Note:** This field is **not** sent to the B-Deploy API - it's used only by the SDK

---

## Complete Example

See `testdata/lfn-control.json` for a complete configuration with all fields:

```bash
cat testdata/lfn-control.json
```

## Minimal Example

Minimal BSN setup (most fields use defaults):

```json
{
  "bDeploy": {
    "username": "user@example.com",
    "networkName": "Production",
    "packageName": "simple-bsn-setup"
  },
  "setupType": "bsn",
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
  }
}
```

## Development Configuration Example

Maximum debugging and logging enabled:

```json
{
  "bDeploy": {
    "username": "dev@example.com",
    "networkName": "Development",
    "packageName": "dev-debug-setup"
  },
  "setupType": "bsn",
  "timeZone": "America/Los_Angeles",
  "enableSerialDebugging": true,
  "enableSystemLogDebugging": true,
  "dwsEnabled": true,
  "dwsPassword": "none",
  "remoteDwsEnabled": true,
  "lwsEnabled": true,
  "lwsConfig": "content",
  "playbackLoggingEnabled": true,
  "eventLoggingEnabled": true,
  "diagnosticLoggingEnabled": true,
  "stateLoggingEnabled": true,
  "variableLoggingEnabled": true,
  "uploadLogFilesAtBoot": true,
  "uploadLogFilesAtSpecificTime": true,
  "uploadLogFilesTime": 3,
  "enableRemoteSnapshot": true,
  "remoteSnapshotInterval": 15,
  "remoteSnapshotMaxImages": 10,
  "networkDiagnosticsEnabled": true,
  "testEthernetEnabled": true,
  "testInternetEnabled": true,
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
  }
}
```

## Usage

Create a setup record:

```bash
./bin/bsn-bdeploy-setup-add config.json
```

With verbose output:

```bash
./bin/bsn-bdeploy-setup-add --verbose config.json
```

With custom timeout:

```bash
./bin/bsn-bdeploy-setup-add --timeout 120 config.json
```

## Field Validation

**Required fields:**
- `bDeploy.username`
- `bDeploy.networkName`
- `bDeploy.packageName`
- `setupType`

**All other fields are optional** and will use sensible defaults if not specified.

## Notes

1. **Token Generation**: The device registration token is automatically generated - you don't need to provide it

2. **Version**: The SDK automatically sets `version` to `"3.0.0"` if not specified

3. **Defaults**: Most fields have sensible defaults - only specify fields you want to customize

4. **Security**: Be cautious with passwords in config files:
   - Use `"none"` explicitly for no password
   - Avoid storing sensitive passwords in version control
   - Consider environment variables for sensitive values

5. **Network Interfaces**: You can mix legacy flat fields (e.g., `useDHCP`, `staticIPAddress`) with the v3 `network.interfaces` array - both are supported

6. **Setup Types**: Choose the setup type carefully as it determines the player's fundamental operation mode

7. **Testing**: Always test configurations in a development environment before deploying to production
