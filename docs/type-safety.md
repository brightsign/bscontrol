# Type Safety in bscontrol

## Overview

The bscontrol SDK exports all necessary types for complete compile-time type safety. This means you get IDE autocomplete, type checking, and refactoring support when working with device data, snapshots, and configuration records.

## Exported Type Categories

### Device Status Types

These types allow you to work with device status information returned from the BSN.cloud API:

- **DeviceStatusEmbed** - The main status container embedded in Device objects
- **FirmwareInfo** - Firmware version and model information
- **ScriptInfo** - BrightScript or plugin information
- **SyncSettings** - Synchronization configuration and status
- **SyncPeriod** - Synchronization timing periods
- **StorageInfo** - Storage device capacity and usage
- **NetworkInfo** - Network connectivity details
- **NetworkInterfaceBSN** - Individual network interface information

#### Example: Monitoring Device Health

```go
device, err := client.Devices.Get(ctx, deviceID)
if err != nil {
    return err
}

// Type-safe access to status fields
status := device.Status
if status == nil {
    return errors.New("no status information")
}

// Check firmware version with full type safety
if status.Firmware != nil {
    if !strings.HasPrefix(status.Firmware.Version, "8.") {
        log.Printf("Warning: Device %s has outdated firmware %s",
            device.Serial, status.Firmware.Version)
    }
}

// Monitor storage usage
for _, storage := range status.Storage {
    usagePercent := float64(storage.Used) / float64(storage.Total) * 100
    if usagePercent > 90 {
        log.Printf("Critical: Device %s storage at %.1f%%",
            device.Serial, usagePercent)
    }
}

// Check network connectivity
if status.Network != nil {
    log.Printf("Device %s IP: %s", device.Serial, status.Network.ExternalIP)
    for _, iface := range status.Network.Interfaces {
        log.Printf("  Interface %s (%s): %s",
            iface.Name, iface.Type, iface.Address)
    }
}
```

#### Example: Building Monitoring Functions

```go
// Type-safe function signatures
func checkFirmwareVersion(status *bscontrol.DeviceStatusEmbed) error {
    if status.Firmware == nil {
        return errors.New("no firmware information")
    }

    version := status.Firmware.Version
    if !isCurrentVersion(version) {
        return fmt.Errorf("outdated firmware: %s", version)
    }
    return nil
}

func calculateStorageMetrics(status *bscontrol.DeviceStatusEmbed) map[string]float64 {
    metrics := make(map[string]float64)

    for i, storage := range status.Storage {
        usagePercent := float64(storage.Used) / float64(storage.Total) * 100
        metrics[fmt.Sprintf("storage_%d_usage_percent", i)] = usagePercent
    }

    return metrics
}
```

### Screenshot Types

Capture device screenshots with precise control over format, quality, and region:

- **SnapshotRequest** - Screenshot capture parameters
- **SnapshotResponse** - Screenshot result information
- **Region** - Rectangular screen region specification

#### Example: Capturing Screenshots

```go
// Full-screen screenshot
req := &bscontrol.SnapshotRequest{
    Format:  "png",
    Quality: 90,
}

resp, err := client.RDWS.Snapshot(ctx, deviceID, req)
if err != nil {
    return err
}

log.Printf("Captured %dx%d screenshot: %s", resp.Width, resp.Height, resp.Filename)

// Region-based screenshot
regionReq := &bscontrol.SnapshotRequest{
    Format:  "jpeg",
    Quality: 85,
    Region: &bscontrol.Region{
        X:      0,
        Y:      0,
        Width:  1920,
        Height: 1080,
    },
}

regionResp, err := client.RDWS.Snapshot(ctx, deviceID, regionReq)
if err != nil {
    return err
}
```

#### Example: Screenshot Comparison Service

```go
func captureForComparison(deviceID string) (*bscontrol.SnapshotResponse, error) {
    // Standardized screenshot settings for comparison
    req := &bscontrol.SnapshotRequest{
        Format:  "png",
        Quality: 100,  // Lossless for comparison
        Region: &bscontrol.Region{
            X:      0,
            Y:      0,
            Width:  1920,
            Height: 1080,
        },
    }

    return client.RDWS.Snapshot(ctx, deviceID, req)
}

func compareScreenshots(a, b *bscontrol.SnapshotResponse) bool {
    // Type-safe comparison of screenshot metadata
    return a.Width == b.Width &&
           a.Height == b.Height &&
           a.Format == b.Format
}
```

### RDWS Information Types

Access detailed player information from the Remote Diagnostic Web Server:

- **RDWSInfo** - Complete player diagnostic information
- **RDWSInfoSubResult** - Nested result objects for power, POE, extensions, etc.

#### Example: Accessing RDWS Data

```go
info, err := client.RDWS.GetInfo(ctx, deviceID)
if err != nil {
    return err
}

// Type-safe access to nested power information
if info.Power != nil {
    if voltage, ok := info.Power.Result["voltage"].(string); ok {
        log.Printf("Device power: %s", voltage)
    }
}

// Access networking information
if info.Networking != nil {
    log.Printf("Network config: %+v", info.Networking.Result)
}
```

### B-Deploy Configuration Types

Build device setup records with full type safety:

- **BDeploySetupRecord** - Complete setup record for device provisioning
- **BDeployInfo** - B-Deploy configuration section
- **IdleScreenColor** - RGBA color specification for idle screens
- **BSNTokenEntity** - BSN.cloud registration token

#### Example: Creating Setup Records

```go
setup := &bscontrol.BDeploySetupRecord{
    Version:  "1.0",
    SetupType: "standard",
    BDeploy: &bscontrol.BDeployInfo{
        NetworkID: "12345",
        GroupID:   "67890",
    },
    IdleScreenColor: &bscontrol.IdleScreenColor{
        R: 0,
        G: 0,
        B: 0,
        A: 1,
    },
}

response, err := client.BDeploy.CreateSetup(ctx, setup)
if err != nil {
    return err
}
```

## Benefits of Type Safety

### 1. IDE Autocomplete

```go
device.Status.  // IDE shows all available fields
device.Status.Firmware.  // IDE shows Version, Model, UpdatedAt, etc.
```

### 2. Compile-Time Error Detection

```go
// Typo caught at compile time
fmt.Println(device.Status.Firmwre.Version)  // ✗ Compiler error

// Correct spelling required
fmt.Println(device.Status.Firmware.Version)  // ✓ Compiles
```

### 3. Refactoring Support

When types are exported, IDE refactoring tools can:
- Rename fields across the entire codebase
- Find all usages of a type
- Generate safe refactoring transformations

### 4. Better Documentation

```bash
# View type documentation
go doc bscontrol.DeviceStatusEmbed
go doc bscontrol.SnapshotRequest
go doc bscontrol.IdleScreenColor
```

### 5. Unit Testing

Create test fixtures without HTTP mocking:

```go
func TestDeviceHealthCheck(t *testing.T) {
    tests := []struct {
        name    string
        status  *bscontrol.DeviceStatusEmbed
        wantErr bool
    }{
        {
            name: "healthy device",
            status: &bscontrol.DeviceStatusEmbed{
                Firmware: &bscontrol.FirmwareInfo{Version: "8.5.42"},
                Storage:  []bscontrol.StorageInfo{{Total: 1000, Used: 500}},
            },
            wantErr: false,
        },
        {
            name: "outdated firmware",
            status: &bscontrol.DeviceStatusEmbed{
                Firmware: &bscontrol.FirmwareInfo{Version: "7.1.0"},
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := checkDeviceHealth(tt.status)
            if (err != nil) != tt.wantErr {
                t.Errorf("checkDeviceHealth() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Type-Safe Function Patterns

### Pattern 1: Status Extractors

```go
func extractFirmwareVersion(status *bscontrol.DeviceStatusEmbed) string {
    if status == nil || status.Firmware == nil {
        return "unknown"
    }
    return status.Firmware.Version
}

func extractNetworkInfo(status *bscontrol.DeviceStatusEmbed) []string {
    if status == nil || status.Network == nil {
        return nil
    }

    var interfaces []string
    for _, iface := range status.Network.Interfaces {
        interfaces = append(interfaces, fmt.Sprintf("%s: %s", iface.Name, iface.Type))
    }
    return interfaces
}
```

### Pattern 2: Validation Functions

```go
func validateSnapshotRequest(req *bscontrol.SnapshotRequest) error {
    if req.Quality < 0 || req.Quality > 100 {
        return errors.New("quality must be 0-100")
    }

    if req.Region != nil {
        if req.Region.Width <= 0 || req.Region.Height <= 0 {
            return errors.New("region dimensions must be positive")
        }
    }

    return nil
}
```

### Pattern 3: Builder Functions

```go
func buildStandardSetup(networkID, groupID string) *bscontrol.BDeploySetupRecord {
    return &bscontrol.BDeploySetupRecord{
        Version:   "1.0",
        SetupType: "standard",
        BDeploy: &bscontrol.BDeployInfo{
            NetworkID: networkID,
            GroupID:   groupID,
        },
        IdleScreenColor: &bscontrol.IdleScreenColor{
            R: 0, G: 0, B: 0, A: 1,  // Black
        },
    }
}

func buildHighQualitySnapshot() *bscontrol.SnapshotRequest {
    return &bscontrol.SnapshotRequest{
        Format:  "png",
        Quality: 100,
    }
}
```

## Migration from Older Versions

If you were previously working around missing type exports using `interface{}` or reflection, you can now use direct type-safe access:

### Before (Without Type Exports)

```go
device, _ := client.Devices.Get(ctx, deviceID)
// Had to use interface{} and lose type safety
statusAny := interface{}(device.Status)
fmt.Printf("%+v\n", statusAny)
```

### After (With Type Exports)

```go
device, _ := client.Devices.Get(ctx, deviceID)
// Direct type-safe access
status := device.Status
if status.Firmware != nil {
    fmt.Printf("Firmware: %s\n", status.Firmware.Version)
}
```

## All Exported Types

For a complete list of all exported types, see the package documentation:

```bash
go doc github.com/brightsign/bscontrol
```

Or view online at: https://pkg.go.dev/github.com/brightsign/bscontrol
