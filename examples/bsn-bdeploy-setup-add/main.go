package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/brightsign/bscontrol"
	"github.com/brightsign/bscontrol/examples/internal/setuptemplate"
)

// SetupConfig represents the JSON configuration file with optional timeout field
type SetupConfig struct {
	bscontrol.BDeploySetupRecord
	Timeout int `json:"timeout,omitempty"` // HTTP timeout (not part of B-Deploy API)
}

func main() {
	var (
		helpFlag         = flag.Bool("help", false, "Display usage information")
		verboseFlag      = flag.Bool("verbose", false, "Show detailed information")
		jsonFlag         = flag.Bool("json", false, "Output as JSON")
		timeoutFlag      = flag.Int("timeout", 30, "Request timeout in seconds (overrides config file)")
		templateFlag     = flag.String("template", setuptemplate.DefaultTemplate, "Path to setup template file")
		packageNameFlag  = flag.String("package-name", "", "Package name for the setup (required in template mode, env: BS_PACKAGE_NAME)")
		deviceNameFlag   = flag.String("device-name", "", "Device name (optional, env: BS_DEVICE_NAME)")
		deviceDescFlag   = flag.String("device-description", "", "Device description (optional, env: BS_DEVICE_DESCRIPTION)")
		networkFlag      = flag.String("network", "", "Network name (env: BS_NETWORK)")
		usernameFlag     = flag.String("username", "", "BSN.cloud username (env: BS_CLIENT_ID)")
		groupFlag        = flag.String("group", "Default", "BSN group name")
	)

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [config.json]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to create a B-Deploy setup record.\n\n")
		fmt.Fprintf(os.Stderr, "Two modes of operation:\n")
		fmt.Fprintf(os.Stderr, "  Template mode: Use --package-name (or BS_PACKAGE_NAME) with the setup template\n")
		fmt.Fprintf(os.Stderr, "  Config mode:   Pass a JSON configuration file as a positional argument\n\n")
		fmt.Fprintf(os.Stderr, "This program performs the following workflow:\n")
		fmt.Fprintf(os.Stderr, "  1. Authenticate with BSN.cloud\n")
		fmt.Fprintf(os.Stderr, "  2. Select and set the network context\n")
		fmt.Fprintf(os.Stderr, "  3. Generate a device registration token\n")
		fmt.Fprintf(os.Stderr, "  4. Create a B-Deploy setup record\n")
		fmt.Fprintf(os.Stderr, "  5. Output setup-id for use with player association\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID           BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET              BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK             BSN.cloud network name\n")
		fmt.Fprintf(os.Stderr, "  BS_PACKAGE_NAME        Setup package name\n")
		fmt.Fprintf(os.Stderr, "  BS_DEVICE_NAME         Device name\n")
		fmt.Fprintf(os.Stderr, "  BS_DEVICE_DESCRIPTION  Device description\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Create setup using template (recommended):\n")
		fmt.Fprintf(os.Stderr, "    %s --package-name \"retail-v1\" --network \"Production\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Create setup using template with all options:\n")
		fmt.Fprintf(os.Stderr, "    %s --package-name \"retail-v1\" --network \"Production\" --device-name \"Lobby\" --group \"Stores\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Create setup using config file (backward compatible):\n")
		fmt.Fprintf(os.Stderr, "    %s config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Create setup with verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --verbose config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Use custom template:\n")
		fmt.Fprintf(os.Stderr, "    %s --template my-template.json --package-name \"retail-v1\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --json config.json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Resolve environment variables
	resolvedPackageName := setuptemplate.ResolveVar(*packageNameFlag, setuptemplate.EnvPackageName)
	resolvedDeviceName := setuptemplate.ResolveVar(*deviceNameFlag, setuptemplate.EnvDeviceName)
	resolvedDeviceDesc := setuptemplate.ResolveVar(*deviceDescFlag, setuptemplate.EnvDeviceDescription)
	resolvedNetwork := setuptemplate.ResolveVar(*networkFlag, "BS_NETWORK")
	resolvedUsername := setuptemplate.ResolveVar(*usernameFlag, "BS_CLIENT_ID")

	// Determine mode: template mode (no positional arg) or config file mode (positional arg)
	var setupConfig *SetupConfig

	if flag.NArg() == 1 {
		// Config file mode (backward compatible)
		configFile := flag.Arg(0)

		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "📋 Loading configuration from: %s\n", configFile)
		}
		var err error
		setupConfig, err = loadConfig(configFile)
		if err != nil {
			log.Fatalf("❌ Failed to load config: %v", err)
		}
	} else if flag.NArg() == 0 && resolvedPackageName != "" {
		// Template mode
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "📋 Rendering setup from template: %s\n", *templateFlag)
		}

		vars := setuptemplate.TemplateVars{
			Username:          resolvedUsername,
			NetworkName:       resolvedNetwork,
			PackageName:       resolvedPackageName,
			GroupName:         *groupFlag,
			DeviceName:        resolvedDeviceName,
			DeviceDescription: resolvedDeviceDesc,
		}

		record, err := setuptemplate.Render(*templateFlag, vars)
		if err != nil {
			log.Fatalf("❌ Failed to render template: %v", err)
		}

		// Clear the token entity so it gets auto-generated after authentication
		if record.BSNDeviceRegistrationTokenEntity != nil && record.BSNDeviceRegistrationTokenEntity.Token == "" {
			record.BSNDeviceRegistrationTokenEntity = nil
		}

		setupConfig = &SetupConfig{BDeploySetupRecord: *record}
	} else {
		fmt.Fprintf(os.Stderr, "Error: provide a config file or use --package-name (env: BS_PACKAGE_NAME)\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate configuration
	if err := validateConfig(&setupConfig.BDeploySetupRecord); err != nil {
		log.Fatalf("❌ Invalid configuration: %v", err)
	}

	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Configuration loaded successfully\n")
		fmt.Fprintf(os.Stderr, "   Network: %s\n", setupConfig.BDeploy.NetworkName)
		fmt.Fprintf(os.Stderr, "   Username: %s\n", setupConfig.BDeploy.Username)
		fmt.Fprintf(os.Stderr, "   Package: %s\n", setupConfig.BDeploy.PackageName)
		fmt.Fprintf(os.Stderr, "   Setup Type: %s\n", setupConfig.SetupType)
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Use timeout from flag if specified, otherwise from config
	timeout := *timeoutFlag
	if timeout == 30 && setupConfig.Timeout > 0 {
		timeout = setupConfig.Timeout
	}

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if timeout > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(timeout)*time.Second))
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔧 Creating BSN.cloud client...\n")
	}
	client, err := bscontrol.New(opts...)
	if err != nil {
		if bscontrol.IsConfigurationError(err) {
			log.Fatalf("❌ Configuration error: %v", err)
		}
		log.Fatalf("❌ Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Step 1: Authenticate
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔐 Authenticating with BSN.cloud...\n")
	}
	if err := client.Authenticate(ctx); err != nil {
		if bscontrol.IsAuthenticationError(err) {
			log.Fatalf("❌ Authentication failed: %v", err)
		}
		log.Fatalf("❌ Authentication error: %v", err)
	}
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Authentication successful!\n")
	}

	// Step 2: Verify network exists
	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📡 Verifying network access...\n")
		networks, err := client.GetNetworks(ctx)
		if err != nil {
			log.Fatalf("❌ Failed to get networks: %v", err)
		}

		found := false
		for _, net := range networks {
			if net.Name == setupConfig.BDeploy.NetworkName {
				found = true
				fmt.Fprintf(os.Stderr, "✅ Network found: %s (ID: %d)\n", net.Name, net.ID)
				break
			}
		}

		if !found {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Network '%s' not found in available networks\n", setupConfig.BDeploy.NetworkName)
			fmt.Fprintf(os.Stderr, "   Available networks:\n")
			for _, net := range networks {
				fmt.Fprintf(os.Stderr, "     - %s (ID: %d)\n", net.Name, net.ID)
			}
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Step 3: Set network context
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📡 Setting network context to: %s\n", setupConfig.BDeploy.NetworkName)
	}
	if err := client.BDeploy.SetNetworkContext(ctx, setupConfig.BDeploy.NetworkName); err != nil {
		log.Fatalf("❌ Failed to set network context: %v", err)
	}
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Network context set successfully!\n")
	}

	// Step 3.5: Generate device registration token (if not already provided)
	if setupConfig.BSNDeviceRegistrationTokenEntity == nil {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔑 Generating device registration token...\n")
		}
		deviceToken, err := client.Provisioning.GenerateDeviceToken(ctx)
		if err != nil {
			log.Fatalf("❌ Failed to generate device token: %v", err)
		}
		setupConfig.BSNDeviceRegistrationTokenEntity = deviceToken
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "✅ Device registration token generated successfully!\n")
		}
		if *verboseFlag && !*jsonFlag {
			fmt.Fprintf(os.Stderr, "   Token: %s...\n", deviceToken.Token[:32])
			fmt.Fprintf(os.Stderr, "   Scope: %s\n", deviceToken.Scope)
			fmt.Fprintf(os.Stderr, "   Valid from: %s\n", deviceToken.ValidFrom)
			fmt.Fprintf(os.Stderr, "   Valid to: %s\n", deviceToken.ValidTo)
		}
	} else {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "✅ Using provided device registration token\n")
		}
		if *verboseFlag && !*jsonFlag {
			fmt.Fprintf(os.Stderr, "   Token: %s...\n", setupConfig.BSNDeviceRegistrationTokenEntity.Token[:32])
			fmt.Fprintf(os.Stderr, "   Scope: %s\n", setupConfig.BSNDeviceRegistrationTokenEntity.Scope)
		}
	}
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Set version if not specified
	if setupConfig.Version == "" {
		setupConfig.Version = "3.0.0"
	}

	// Set defaults for optional fields
	if setupConfig.TimeZone == "" {
		setupConfig.TimeZone = "America/New_York"
	}
	if setupConfig.BSNGroupName == "" {
		setupConfig.BSNGroupName = "Default"
	}

	// Step 4: Create B-Deploy setup record
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📝 Creating B-Deploy setup record...\n")
	}

	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "   Setup details:\n")
		fmt.Fprintf(os.Stderr, "     Version: %s\n", setupConfig.Version)
		fmt.Fprintf(os.Stderr, "     Username: %s\n", setupConfig.BDeploy.Username)
		fmt.Fprintf(os.Stderr, "     Network: %s\n", setupConfig.BDeploy.NetworkName)
		fmt.Fprintf(os.Stderr, "     Package: %s\n", setupConfig.BDeploy.PackageName)
		fmt.Fprintf(os.Stderr, "     Type: %s\n", setupConfig.SetupType)
		fmt.Fprintf(os.Stderr, "     Time Zone: %s\n", setupConfig.TimeZone)
		fmt.Fprintf(os.Stderr, "     BSN Group: %s\n", setupConfig.BSNGroupName)
		if setupConfig.BSNDeviceRegistrationTokenEntity != nil {
			fmt.Fprintf(os.Stderr, "     Token Scope: %s\n", setupConfig.BSNDeviceRegistrationTokenEntity.Scope)
		}
		if setupConfig.Network != nil {
			fmt.Fprintf(os.Stderr, "     Interfaces: %d\n", len(setupConfig.Network.Interfaces))
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	response, err := client.BDeploy.AddSetupRecord(ctx, &setupConfig.BDeploySetupRecord)
	if err != nil {
		log.Fatalf("❌ Failed to create B-Deploy setup record: %v", err)
	}

	// Step 5: Display results
	if *jsonFlag {
		// Output as JSON
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(response); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Normal output
	fmt.Fprintf(os.Stderr, "✅ B-Deploy setup record created successfully!\n")

	// Debug: Show what we got back
	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "\n🔍 Response Debug:\n")
		fmt.Fprintf(os.Stderr, "   ID: '%s'\n", response.ID)
		fmt.Fprintf(os.Stderr, "   Success: %v\n", response.Success)
		fmt.Fprintf(os.Stderr, "   Error: '%s'\n", response.Error)
	}

	if response.Error != "" {
		fmt.Fprintf(os.Stderr, "   ⚠️  API Error: %s\n", response.Error)
	}

	// Prominently display the setup-id
	fmt.Fprintf(os.Stderr, "\n%s\n", strings.Repeat("=", 70))
	fmt.Fprintf(os.Stderr, "SETUP-ID: %s\n", response.ID)
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("=", 70))
	fmt.Fprintf(os.Stderr, "Save this setup-id - you'll need it to associate players with this setup\n")

	// Generate setup URLs
	fmt.Fprintf(os.Stderr, "\n🔗 Setup URLs:\n")
	fmt.Fprintf(os.Stderr, "   Web: https://provision.bsn.cloud/setup/%s\n", response.ID)
	fmt.Fprintf(os.Stderr, "   API: https://provision.bsn.cloud/rest-setup/v3/setup/%s\n", response.ID)

	fmt.Fprintf(os.Stderr, "\n📱 Next Steps:\n")
	fmt.Fprintf(os.Stderr, "   Option 1: Associate existing players with this setup\n")
	fmt.Fprintf(os.Stderr, "     Use the associate-player-with-setup example with the setup-id above\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "   Option 2: Provision new players directly\n")
	fmt.Fprintf(os.Stderr, "     - Enter the setup URL in the player's web interface\n")
	fmt.Fprintf(os.Stderr, "     - Generate a QR code from the URL and scan it with the player\n")
	fmt.Fprintf(os.Stderr, "     - Load the setup configuration via USB storage\n")

	fmt.Fprintf(os.Stderr, "\n🎉 Setup record created successfully!\n")
}

// loadConfig reads and parses the JSON configuration file
func loadConfig(filename string) (*SetupConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config SetupConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &config, nil
}

// validateConfig checks that required configuration fields are present
func validateConfig(config *bscontrol.BDeploySetupRecord) error {
	if config.BDeploy.NetworkName == "" {
		return fmt.Errorf("bDeploy.networkName is required")
	}
	if config.BDeploy.Username == "" {
		return fmt.Errorf("bDeploy.username is required")
	}
	if config.BDeploy.PackageName == "" {
		return fmt.Errorf("bDeploy.packageName is required")
	}
	if config.SetupType == "" {
		return fmt.Errorf("setupType is required")
	}
	// Token is optional - will be auto-generated if not provided
	// Network interfaces are optional for some setup types
	return nil
}
