# BrightSign Control SDK for Go

![BrightSign Control Logo](images/logo.png)

> **Derived from [gopurple](https://github.com/BrightDevelopers/gopurple).**
> This SDK was split out of the [gopurple](https://github.com/BrightDevelopers/gopurple) project into its own repository. We are building fast and expanding well beyond the original scope, so expect frequent changes as the API grows. For the original project, see [BrightDevelopers/gopurple](https://github.com/BrightDevelopers/gopurple).

**Status:** EXTERNAL BETA RELEASE

## Why Go Instead of Python?

Python is incredibly popular and would be an easy starting point for many developers. Its vast ecosystem, readable syntax, and extensive libraries make it a natural first choice. However, after careful consideration, we chose Go for this SDK. Here's why:

### Cross-Platform Deployment: Single Binary Simplicity

One of Python's biggest challenges is deployment complexity. Running Python reliably across Windows, macOS, Linux, and cloud environments requires managing:
- **Python interpreter installation** - Ensuring the correct Python version is available on each target system
- **Virtual environments** - Isolating dependencies per project to avoid conflicts
- **Dependency resolution** - Managing pip, requirements.txt, and potential version conflicts between packages
- **Platform-specific issues** - Some packages require different installation procedures or have OS-specific dependencies

As noted in the Python community, [dependency management can be challenging](https://nielscautaerts.xyz/python-dependency-management-is-a-dumpster-fire.html), and many teams resort to containerization just to achieve consistent deployments.

Go takes a different approach: **compile once, run anywhere**. The Go compiler produces a single, self-contained binary with no external dependencies. Deploy to Windows, Mac, Linux, or any cloud platform by simply copying one file. No runtime installation, no virtual environments, no dependency management on target systems.

### Easy to Learn

Go was designed with simplicity in mind. Its [small, consistent syntax](https://go.dev/doc/effective_go) means developers can become productive quickly. The language has:
- **25 keywords** (compared to Python's 35+)
- **One way to do things** - Go's philosophy reduces decision fatigue
- **Excellent documentation** - The standard library is well-documented with runnable examples
- **Built-in formatting** - `go fmt` eliminates style debates

### Excellent Concurrency Model

Go's concurrency primitives—goroutines and channels—provide an intuitive model for parallel operations. This is essential for an SDK that needs to:
- Handle multiple device connections simultaneously
- Process large batches of API requests efficiently
- Manage concurrent uploads and downloads

Python's Global Interpreter Lock (GIL) [limits true parallelism](https://rubyroidlabs.com/blog/2024/09/go-vs-python/), making concurrent I/O-bound operations more complex to implement correctly.

### AI and LLMs Write Excellent Go Code

Modern AI assistants and LLMs are remarkably proficient at generating Go code. This is increasingly important as developers leverage AI tools for productivity. Go's characteristics make it particularly well-suited for AI-assisted development:

- **Strong typing** catches errors at compile time, providing immediate feedback on generated code
- **Simple, consistent syntax** means AI models produce more reliable output
- **Comprehensive standard library** reduces the need for third-party dependencies
- **Clear error handling patterns** result in more predictable generated code

As the Go team notes, [Go is excellent for building LLM-powered applications](https://go.dev/blog/llmpowered), and the same qualities that make it good for *using* AI make it good for AI to *write*.

### Performance Where It Matters

Go compiles to native machine code, delivering [significantly better performance](https://uvik.net/blog/go-vs-python/) than interpreted Python for CPU-bound operations. For an SDK making hundreds of API calls, parsing large responses, and processing device data, this translates to faster execution and lower resource usage.

A comprehensive Go SDK providing type-safe access to BrightSign Network (BSN.cloud) APIs. Includes both the main REST API and Remote Diagnostic Web Server (RDWS) functionality with 66 working example programs.

## What It Does

This SDK abstracts the complexity of BSN.cloud integration:

- **OAuth2 Authentication** - Automatic token management and refresh
- **Device Management** - List, monitor, control, and provision BrightSign players
- **Remote Operations** - Screenshots, reboots, file management, diagnostics via RDWS
- **B-Deploy Provisioning** - Create and manage player setup configurations
- **Network Management** - Handle multiple networks, groups, permissions

**Key Features:**
- **Full type safety** - All nested types exported for compile-time checking and IDE autocomplete
- **Device status monitoring** - Type-safe access to firmware, network, storage, and sync information
- **Screenshot capture** - Request and response types for device snapshots with region support
- Structured error types with context
- Pagination support for large datasets
- Network context management with `BS_NETWORK` environment variable support
- Concurrent-safe token handling
- 64 example CLI tools demonstrating all features
- Flexible output modes (JSON, stdout, file) for automation and scripting

## Quick Start

### Installation

**Recommended: Pin to a specific version**
```bash
go get github.com/brightsign/bscontrol@v1.2.0
```

**Or use latest (development)**
```bash
go get github.com/brightsign/bscontrol
```

**In your go.mod:**
```go
require github.com/brightsign/bscontrol v1.2.0
```

See [versioning docs](docs/versioning.md) for version management details.

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "github.com/brightsign/bscontrol"
)

func main() {
    // Create client (uses BS_CLIENT_ID and BS_SECRET env vars)
    client, err := bscontrol.New()
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Authenticate and set network
    if err := client.Authenticate(ctx); err != nil {
        log.Fatal(err)
    }

    // List devices
    devices, err := client.Devices.List(ctx, nil)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Found %d devices", len(devices.Items))
}
```

### Working with Device Status

The SDK provides full type safety for device status information, including firmware, network, storage, and synchronization details:

```go
// Get device with full status information
device, err := client.Devices.Get(ctx, "ABC123")
if err != nil {
    log.Fatal(err)
}

// Access device status with full type safety
if device.Status != nil {
    // Check firmware version
    if device.Status.Firmware != nil {
        log.Printf("Firmware: %s", device.Status.Firmware.Version)
    }

    // Check network connectivity
    if device.Status.Network != nil {
        log.Printf("IP: %s", device.Status.Network.ExternalIP)
        for _, iface := range device.Status.Network.Interfaces {
            log.Printf("  %s: %s", iface.Name, iface.Type)
        }
    }

    // Monitor storage usage
    for _, storage := range device.Status.Storage {
        pct := float64(storage.Used) / float64(storage.Total) * 100
        log.Printf("Storage: %.1f%% used", pct)
    }
}
```

### Taking Screenshots

Capture device screenshots with specific regions and quality settings:

```go
// Take a full-screen screenshot
req := &bscontrol.SnapshotRequest{
    Format:  "png",
    Quality: 90,
}

resp, err := client.RDWS.Snapshot(ctx, "ABC123", req)
if err != nil {
    log.Fatal(err)
}
log.Printf("Screenshot: %s (%dx%d)", resp.Filename, resp.Width, resp.Height)

// Capture specific region
regionReq := &bscontrol.SnapshotRequest{
    Format:  "jpeg",
    Quality: 85,
    Region: &bscontrol.Region{
        X:      100,
        Y:      100,
        Width:  1920,
        Height: 1080,
    },
}
```

### Type-Safe B-Deploy Configuration

Build device setup records with full compile-time type checking:

```go
setup := &bscontrol.BDeploySetupRecord{
    Version: "1.0",
    BDeploy: &bscontrol.BDeployInfo{
        // ... your B-Deploy configuration
    },
    IdleScreenColor: &bscontrol.IdleScreenColor{
        R: 0, G: 0, B: 0, A: 1,  // Black idle screen
    },
}
```

## Version Management

### Using Versioned Releases (Recommended)

Always pin to specific versions in production to ensure reproducible builds:

```bash
# Install specific version
go get github.com/brightsign/bscontrol@v1.2.0
```

**In your go.mod:**
```go
require github.com/brightsign/bscontrol v1.2.0
```

### Checking Available Versions

```bash
# List all available versions
go list -m -versions github.com/brightsign/bscontrol

# View current installed version
go list -m github.com/brightsign/bscontrol

# View release notes
gh release list --repo brightdevelopers/bscontrol
```

### Updating to Newer Versions

```bash
# Update to specific version
go get github.com/brightsign/bscontrol@v1.3.0

# Update to latest version
go get -u github.com/brightsign/bscontrol

# Update to latest patch only (safer)
go get -u=patch github.com/brightsign/bscontrol
```

### Semantic Versioning

The SDK follows [Semantic Versioning 2.0.0](https://semver.org/):

- **v1.x.x → v1.x.x** - Backward compatible (safe to update)
- **v1.x.x → v2.0.0** - Breaking changes (review migration guide)
- **v1.x.x-beta.x** - Pre-release versions (test before production)

**Version compatibility:**
```bash
# All v1 releases are compatible with each other
v1.0.0, v1.1.0, v1.2.0  ✅ Compatible

# Major version changes may require code updates
v1.x.x → v2.0.0  ⚠️ Review breaking changes
```

### CI/CD Integration

**GitHub Actions:**
```yaml
- name: Install bscontrol
  run: go get github.com/brightsign/bscontrol@v1.2.0
```

**Dockerfile:**
```dockerfile
FROM golang:1.24-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
# bscontrol version pinned in go.mod
COPY . .
RUN go build -o app
```

### For Maintainers: Creating Releases

```bash
# Create a new release
./scripts/release.sh v1.2.0 "Release Description"

# Create a pre-release
./scripts/release.sh v1.3.0-beta.1 "Beta: New Features"
```

See [RELEASING.md](RELEASING.md) for the complete release process and [docs/versioning.md](docs/versioning.md) for detailed version management documentation.

## Authentication

### Get API Credentials

1. Log into BSN.cloud
2. Navigate to **Admin** → **API Access**
3. Create new OAuth2 credentials
4. Note your Client ID and Secret

### Configure SDK

```bash
export BS_CLIENT_ID=your_client_id
export BS_SECRET=your_client_secret
export BS_NETWORK=your_network_name  # optional
```

Or configure programmatically:

```go
client, err := bscontrol.New(
    bscontrol.WithCredentials("client-id", "client-secret"),
    bscontrol.WithNetwork("Production"),
    bscontrol.WithTimeout(60*time.Second),
)
```

## Examples

The SDK includes **66 working example programs** demonstrating all functionality. Each example is a standalone CLI tool you can use immediately. Every program is named `bsn-[<subsystem>-]<object>-<verb>` and builds to `bin/<name>`.

See **[examples/README.md](examples/README.md)** for complete documentation of all examples with usage instructions, and the agent skill in **[skills/brightsign-control-cli/](skills/brightsign-control-cli/)** for a decision-oriented command index.

### Example Categories

| Category | Count | Examples |
|----------|-------|----------|
| **Authentication & helpers** | 4 | `bsn-auth-info`, `bsn-token-test`, `bsn-generate-token-json`, `bsn-endpoints-test` |
| **Account device management** | 11 | List, find, info, status, errors, operations, downloads, delete, group change, local-DWS |
| **Group management** | 3 | Group info, updates, deletion |
| **Subscription management** | 3 | Device subscriptions, counts, operations |
| **Registration & provisioning** | 3 | Mint/revoke registration tokens, generate a player registration script |
| **B-Deploy provisioning** | 12 | Setup records, device association, template rendering |
| **Remote DWS (RDWS)** | 30 | Remote control, diagnostics, files, registry, network, screenshots |

### Quick Example

```bash
# Build all examples
make build

# List devices
./bin/bsn-device-list --network Production

# Get device info via RDWS
./bin/bsn-rdws-info --serial BS123456789

# Capture screenshot
./bin/bsn-rdws-snapshot --serial BS123456789 --output screenshot.png

# Get device logs (flexible output modes)
./bin/bsn-rdws-logs-get --serial BS123456789 --logfile device.log  # to file
./bin/bsn-rdws-logs-get --serial BS123456789 --stdout | grep ERROR # to stdout
./bin/bsn-rdws-logs-get --serial BS123456789 --json                # as JSON
```

## Using with AI Coding Agents

This repo ships an **Agent Skill** at [`skills/brightsign-control-cli/`](skills/brightsign-control-cli/) that teaches a coding agent how to drive these 66 `bsn-*` CLIs — choosing the right command, the shared flag conventions, and multi-step fleet workflows (including joining a fielded player to a network by registry keys, "opt-out" provisioning). It follows the portable **`SKILL.md`** standard (YAML frontmatter + Markdown), so it loads wherever Agent Skills are supported and can be referenced as plain context anywhere else.

Build and install the CLIs first so the agent can run them:

```bash
make build            # bin/bsn-*
make install          # copies all bsn-* into ~/bin (override INSTALLDIR=...)
```

### Claude Code / Claude apps (native skills)

Skills are auto-discovered from `.claude/skills/` (project) or `~/.claude/skills/` (personal). Symlink or copy the folder; it then triggers automatically when a task mentions BrightSign, BSN.cloud, a player serial, provisioning, etc.

```bash
# project-scoped (everyone who clones the repo gets it)
mkdir -p .claude/skills
ln -s ../../skills/brightsign-control-cli .claude/skills/brightsign-control-cli

# or personal (available in every project)
cp -r skills/brightsign-control-cli ~/.claude/skills/
```

### Cursor

Cursor loads project rules from `.cursor/rules/`. Add a rule that points at the skill:

```bash
mkdir -p .cursor/rules
cat > .cursor/rules/brightsign-control-cli.mdc <<'EOF'
---
description: BrightSign player/fleet CLI (bsn-* tools)
alwaysApply: false
---
For any task involving BrightSign players, BSN.cloud / BrightSign Control, B-Deploy,
rDWS, registration tokens, or provisioning, follow the skill at
skills/brightsign-control-cli/SKILL.md and the files in its references/ directory.
EOF
```

### GitHub Copilot

Copilot reads repo-wide custom instructions from `.github/copilot-instructions.md`:

```bash
mkdir -p .github
printf '%s\n' \
  'For BrightSign player/fleet tasks (BSN.cloud, B-Deploy, rDWS, provisioning), follow' \
  '`skills/brightsign-control-cli/SKILL.md` and the files in its `references/` directory.' \
  >> .github/copilot-instructions.md
```

### Codex CLI, Gemini CLI, and other `AGENTS.md`-aware agents

Agents that honor `AGENTS.md` pick up repo conventions from it:

```bash
printf '\n## BrightSign CLI tools\nFor BrightSign player/fleet tasks, follow `skills/brightsign-control-cli/SKILL.md`.\n' >> AGENTS.md
```

### Any other agent

The skill is plain Markdown. Point the agent at [`skills/brightsign-control-cli/SKILL.md`](skills/brightsign-control-cli/SKILL.md) (and the `references/` files it links) as context, or register it with a cross-agent skill manager such as skills-mapper (`sm`), which syncs one skill source to Claude Code, Copilot, and others.

## Configuration Files

Example configuration files are provided in multiple locations:

### B-Deploy Setup Configurations (`examples/bsn-bdeploy-setup-add/`)

- **`config.json`** - Basic B-Deploy setup configuration
- **`config-comprehensive.json`** - Full setup with all debugging/logging enabled
- **`config-wifi-example.json`** - Wireless network configuration example
- **`config-annotated.json`** - Annotated configuration showing all available options

The comprehensive configuration includes:
- Local DWS enabled with password protection
- Remote DWS enabled
- All logging types enabled (playback, event, diagnostic, state, variable)
- Serial and system log debugging enabled
- Remote screenshots enabled
- Network diagnostics enabled

### Global Configuration Examples (`configs/`)

- **`bsn-control.json`** - BSN.cloud network control setup example
- **`lfn-control.json`** - Local File Network (LFN) control setup example

### Update Configurations (`examples/bsn-bdeploy-setup-update/`)

- **`example-config.json`** - Partial setup update example
- **`example-full-update.json`** - Complete setup reconfiguration example

These show the JSON structure expected by various SDK methods and can be used as templates for your own configurations.

**Note:** SSH and Telnet must be enabled via RDWS after device provisioning.

## API Coverage

**Implementation Status:** 43 of ~294 endpoints

See **[docs/all-apis.md](docs/all-apis.md)** for a comprehensive list of all BSN.cloud API endpoints with implementation status (`[DONE]` or `[NOT-DONE]`).

### What's Implemented

✅ **Device Management** (Complete)
- List devices with pagination, filtering, sorting
- Get device details, status, errors, operations
- Update device properties, change groups
- Delete devices

✅ **Remote Diagnostic Web Server (RDWS)** (Complete)
- Device info, health, time management
- Remote reboot (normal, crash, factory reset)
- Screenshots and diagnostics
- File operations (list, upload, download, delete, rename)
- Network diagnostics (ping, traceroute, DNS lookup)
- Registry operations (read, write)
- Log and crashdump retrieval
- Firmware downloads
- SSH/Telnet management
- Local DWS control
- Storage reformatting

✅ **B-Deploy Provisioning** (Complete)
- Create, update, delete setup records (struct-based and raw JSON)
- Associate devices with setups
- List and query provisioning configurations
- Template-based setup generation with dynamic token injection
- Raw JSON round-trip preserving all fields (firmware entries, network config)

✅ **Group Management** (Core Features)
- Get group information
- Update group properties
- Delete groups

✅ **Subscription Management** (Read Operations)
- List device subscriptions
- Get subscription counts
- Query permissions

### What's Not Yet Implemented

See [docs/all-apis.md](docs/all-apis.md) for the complete list of unimplemented endpoints, including:

- Autorun/Plugin management
- Dynamic playlists
- Live text feeds
- Data feeds
- Device web pages
- Live media feeds
- Scheduled downloads
- Tag management
- User management
- Web folder management
- And more...

The SDK focuses on the most commonly used operations first. Additional endpoints can be added as needed.

## Architecture

```mermaid
graph TB
    App[Your Application] --> Client[bscontrol.Client]
    Client --> Auth[Auth Manager]
    Client --> HTTP[HTTP Client]
    Client --> Services[Service Layers]

    Auth --> Token[Token Management]
    Auth --> Network[Network Context]

    HTTP --> Retry[Retry Logic]
    HTTP --> Error[Error Handling]

    Services --> Devices[Device Service]
    Services --> RDWS[RDWS Service]
    Services --> BDeploy[B-Deploy Service]

    HTTP --> BSN[BSN Main API<br/>api.bsn.cloud]
    HTTP --> RDWS_API[Remote DWS API<br/>ws.bsn.cloud]
```

## Authentication Flow

```mermaid
sequenceDiagram
    participant App
    participant Client
    participant Auth
    participant BSN as BSN.cloud

    App->>Client: New()
    Client->>Auth: NewAuthManager()

    App->>Client: Authenticate(ctx)
    Client->>Auth: Authenticate()
    Auth->>BSN: POST /token (client_credentials)
    BSN-->>Auth: access_token + expires_in
    Auth-->>Client: success

    App->>Client: SetNetwork(ctx, "MyNetwork")
    Client->>Auth: SetNetwork()
    Auth->>BSN: PUT /Self/Session/Network
    BSN-->>Auth: 204 No Content
    Auth-->>Client: network set

    App->>Client: API Operation
    Client->>Auth: GetToken()
    Auth->>Auth: Check expiry
    Auth-->>Client: valid token
    Client->>BSN: API request with token
    BSN-->>Client: response
    Client-->>App: result
```

## Usage Examples

### Device Operations

```go
// List devices with filtering
params := &bscontrol.DeviceListParams{
    Filter:   "model=XD1033",
    Sort:     "lastModifiedDate desc",
    PageSize: 50,
}
devices, err := client.Devices.List(ctx, params)

// Get specific device
device, err := client.Devices.GetBySerial(ctx, "BS123456789")

// Change device group
err = client.Devices.Update(ctx, deviceID, &bscontrol.DeviceUpdate{
    GroupID: newGroupID,
})

// Delete device
err = client.Devices.Delete(ctx, deviceID)
```

### Remote Operations (RDWS)

```go
// Get device info
info, err := client.RDWS.GetInfo(ctx, serial)

// Capture screenshot
screenshot, err := client.RDWS.Snapshot(ctx, serial)
os.WriteFile("screenshot.png", screenshot, 0644)

// Reboot device
success, err := client.RDWS.Reboot(ctx, serial, "normal")

// List files on device
files, err := client.RDWS.ListFiles(ctx, serial, "/storage/sd/")

// Upload file to device
err = client.RDWS.UploadFile(ctx, serial, localPath, remotePath)
```

### B-Deploy Provisioning

```go
// Create setup record from config
config := &types.BDeploySetupConfig{
    NetworkName: "Production",
    Username:    "admin@example.com",
    PackageName: "retail-display",
    SetupType:   "bsn",
    Network: types.NetworkConfig{
        TimeServers: []string{"time.bsn.cloud"},
        Interfaces: []types.NetworkInterface{
            {
                ID:                      "eth0",
                Type:                    "Ethernet",
                Proto:                   "DHCPv4",
                ContentDownloadEnabled:  true,
                HealthReportingEnabled:  true,
            },
        },
    },
}

setupID, err := client.BDeploy.CreateSetup(ctx, config)

// Associate device with setup
err = client.BDeploy.AssociateDevice(ctx, serial, setupID)
```

### Error Handling

```go
devices, err := client.Devices.List(ctx, nil)
if err != nil {
    if bscontrol.IsAuthenticationError(err) {
        log.Println("Auth failed - check credentials")
    } else if bscontrol.IsNetworkError(err) {
        log.Println("Network issue - check connectivity")
    } else if bscontrol.IsRateLimitError(err) {
        log.Println("Rate limited - retry later")
    } else if bscontrol.IsRetryableError(err) {
        log.Println("Temporary error - will retry")
    } else {
        log.Printf("API error: %v", err)
    }
    return
}
```

## Development

### Build and Test

```bash
# Run all tests
make test

# Build all example programs into bin/
make build

# List all example programs
make list-examples

# Install all example programs to ~/bin (override with INSTALLDIR=...)
make install

# Clean build artifacts
make clean

# Run all (test + build)
make all
```

### Project Structure

```
bscontrol/
├── bscontrol.go                      # Main SDK client interface
├── internal/
│   ├── auth/                       # OAuth2 authentication
│   ├── config/                     # Configuration management
│   ├── errors/                     # Error types
│   ├── http/                       # HTTP client wrapper
│   ├── services/                   # API service implementations
│   └── types/                      # Data structures
├── templates/                       # Setup template files
│   └── DefaultSetupPackageTemplateMaster.json  # Master Go-template for B-Deploy setups
├── setups/                          # Rendered setup templates
│   └── setup-template.json         # Reference setup template
├── examples/                        # 64 example CLI programs
│   ├── bsn-render-setup-template/      # Render template + generate token -> stdout
│   ├── bsn-generate-token-json/        # Generate registration token JSON
│   ├── bsn-bdeploy-setup-add-raw/      # Create setup from raw JSON (preserves all fields)
│   ├── bsn-bdeploy-setup-add/          # Create setup via Go struct
│   │   ├── config.json             # Basic setup example
│   │   ├── config-comprehensive.json
│   │   └── config-wifi-example.json
│   ├── bsn-bdeploy-setup-update/
│   │   ├── example-config.json
│   │   └── example-full-update.json
│   ├── README.md                   # Complete examples documentation
│   └── ... (55+ more examples)
├── configs/                         # Global configuration examples
│   ├── bsn-control.json
│   └── lfn-control.json
├── docs/
│   ├── all-apis.md                 # Complete API endpoint reference
│   ├── bdeploy-config-reference.md # B-Deploy configuration guide
│   ├── bdeploy-association.md      # Device association documentation
│   └── need-json-out.md            # Programs needing JSON output
├── images/
│   └── logo.png                    # Project logo
└── Makefile                         # Build targets
```

## Requirements

- **Go 1.21+**
- **BSN.cloud API credentials** (OAuth2 Client ID and Secret)
- **Network access** to:
  - `auth.bsn.cloud` (authentication)
  - `api.bsn.cloud` (main API)
  - `ws.bsn.cloud` (RDWS API)

## Contributing

This SDK prioritizes the most commonly used BSN.cloud operations. If you need additional endpoints:

1. Check [docs/all-apis.md](docs/all-apis.md) to see if it's documented
2. Look at existing service implementations in `internal/services/`
3. Follow the established patterns for consistency
4. Add tests and example programs
5. Update documentation

### For Maintainers

**Creating releases:**
```bash
./scripts/release.sh v1.2.0 "Release Description"
```

See [RELEASING.md](RELEASING.md) for the complete release workflow and [docs/versioning.md](docs/versioning.md) for version management guidelines.

## License

This is an unofficial SDK for BSN.cloud integration.

## Resources

### Documentation

- **BSN.cloud Documentation**: https://docs.brightsign.biz/
- **Complete API Endpoint Reference**: [docs/all-apis.md](docs/all-apis.md)
- **Type Safety Guide**: [docs/type-safety.md](docs/type-safety.md)
- **Version Management**: [docs/versioning.md](docs/versioning.md)
- **Release Process**: [RELEASING.md](RELEASING.md)
- **Example Programs Guide**: [examples/README.md](examples/README.md)
- **Agent Skill (`bsn-*` CLI usage)**: [skills/brightsign-control-cli/](skills/brightsign-control-cli/)
- **B-Deploy Template Guide**: [docs/bdeploy-template.md](docs/bdeploy-template.md)
- **B-Deploy Configuration Reference**: [docs/bdeploy-config-reference.md](docs/bdeploy-config-reference.md)
- **B-Deploy Device Association**: [docs/bdeploy-association.md](docs/bdeploy-association.md)
- **Programs Needing JSON Output**: [docs/need-json-out.md](docs/need-json-out.md)

### Links

- **BSN.cloud Admin Panel**: https://bsn.cloud/
- **GitHub Repository**: https://github.com/brightsign/bscontrol

## Disclaimer

**EXPERIMENTAL ONLY** - This SDK is provided as-is for integration with BrightSign Network. While actively used in production, it is not officially supported by BrightSign LLC. Use at your own discretion.
