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
	"github.com/brightsign/bscontrol/internal/types"
)

// Config represents the JSON configuration file structure
type Config struct {
	NetworkName  string       `json:"networkName,omitempty"`
	Username     string       `json:"username,omitempty"`
	PackageName  string       `json:"packageName,omitempty"`
	SetupType    string       `json:"setupType,omitempty"`
	TimeZone     string       `json:"timeZone,omitempty"`
	BSNGroupName string       `json:"bsnGroupName,omitempty"`
	Network      NetworkSetup `json:"network,omitempty"`
	Timeout      int          `json:"timeout,omitempty"`
}

// NetworkSetup represents network interface configuration
type NetworkSetup struct {
	TimeServers []string           `json:"timeServers,omitempty"`
	Interfaces  []NetworkInterface `json:"interfaces,omitempty"`
}

// NetworkInterface represents a single network interface configuration
type NetworkInterface struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Type                   string `json:"type"`
	Proto                  string `json:"proto"`
	ContentDownloadEnabled bool   `json:"contentDownloadEnabled"`
	HealthReportingEnabled bool   `json:"healthReportingEnabled"`
}

func main() {
	var (
		helpFlag      = flag.Bool("help", false, "Display usage information")
		jsonFlag      = flag.Bool("json", false, "Output as JSON")
		verboseFlag   = flag.Bool("verbose", false, "Show detailed information")
		setupIDFlag   = flag.String("setup-id", "", "Setup ID to update")
		setupNameFlag   = flag.String("setup-name", "", "Setup name to update (alternative to --setup-id)")
		packageNameFlag = flag.String("package-name", "", "Package name to set on the setup (optional, env: BS_PACKAGE_NAME)")
		timeoutFlag     = flag.Int("timeout", 30, "Request timeout in seconds (overrides config file)")
	)

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [--setup-id <id> | --setup-name <name>] [options] <config.json>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to update an existing B-Deploy setup record using a JSON configuration file.\n\n")
		fmt.Fprintf(os.Stderr, "This program performs the following workflow:\n")
		fmt.Fprintf(os.Stderr, "  1. Authenticate with BSN.cloud\n")
		fmt.Fprintf(os.Stderr, "  2. Set the network context\n")
		fmt.Fprintf(os.Stderr, "  3. Fetch the existing setup record (by ID or name)\n")
		fmt.Fprintf(os.Stderr, "  4. Apply updates from the config file\n")
		fmt.Fprintf(os.Stderr, "  5. Update the B-Deploy setup record\n")
		fmt.Fprintf(os.Stderr, "  6. Display the updated setup details\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Update setup by ID:\n")
		fmt.Fprintf(os.Stderr, "    %s --setup-id 618fb7363a682fe7a40c73ca config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Update setup by name:\n")
		fmt.Fprintf(os.Stderr, "    %s --setup-name \"production-setup\" config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Update with verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --setup-name \"production-setup\" --verbose config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Use custom timeout:\n")
		fmt.Fprintf(os.Stderr, "    %s --setup-id 618fb7363a682fe7a40c73ca --timeout 60 config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --setup-id 618fb7363a682fe7a40c73ca config.json --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Resolve package name from flag or environment variable
	resolvedPackageName := setuptemplate.ResolveVar(*packageNameFlag, setuptemplate.EnvPackageName)

	// Validate that either setup-id or setup-name is provided
	if *setupIDFlag == "" && *setupNameFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: either --setup-id or --setup-name is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate that both are not provided
	if *setupIDFlag != "" && *setupNameFlag != "" {
		fmt.Fprintf(os.Stderr, "Error: cannot specify both --setup-id and --setup-name\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate command line arguments
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Error: config file is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	configFile := flag.Arg(0)

	// Load configuration
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📋 Loading configuration from: %s\n", configFile)
	}
	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Override package name from flag/env if provided
	if resolvedPackageName != "" {
		config.PackageName = resolvedPackageName
	}

	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Configuration loaded successfully\n")
		if config.NetworkName != "" {
			fmt.Fprintf(os.Stderr, "   Network: %s\n", config.NetworkName)
		}
		if config.Username != "" {
			fmt.Fprintf(os.Stderr, "   Username: %s\n", config.Username)
		}
		if config.PackageName != "" {
			fmt.Fprintf(os.Stderr, "   Package: %s\n", config.PackageName)
		}
		if config.SetupType != "" {
			fmt.Fprintf(os.Stderr, "   Setup Type: %s\n", config.SetupType)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Use timeout from flag if specified, otherwise from config
	timeout := *timeoutFlag
	if timeout == 30 && config.Timeout > 0 {
		timeout = config.Timeout
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

	// Step 1.5: Resolve setup ID from name if needed
	setupID := *setupIDFlag
	if *setupNameFlag != "" {
		// Need to determine network context first
		networkName := config.NetworkName
		if networkName == "" {
			// Try to use BS_NETWORK environment variable
			networkName = os.Getenv("BS_NETWORK")
		}
		if networkName == "" {
			log.Fatalf("❌ Network name required when using --setup-name. Specify in config file or set BS_NETWORK environment variable")
		}

		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "📡 Setting network context to: %s\n", networkName)
		}
		if err := client.BDeploy.SetNetworkContext(ctx, networkName); err != nil {
			log.Fatalf("❌ Failed to set network context: %v", err)
		}

		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔍 Looking up setup by name: %s\n", *setupNameFlag)
		}
		resolvedID, err := findSetupByName(ctx, client, *setupNameFlag, networkName, *verboseFlag, *jsonFlag)
		if err != nil {
			log.Fatalf("❌ Failed to find setup by name: %v", err)
		}
		setupID = resolvedID
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "✅ Found setup ID: %s\n", setupID)
		}
	}

	// Step 2: Fetch existing setup record
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📖 Fetching existing setup record: %s\n", setupID)
	}
	existingSetup, err := client.BDeploy.GetSetupRecord(ctx, setupID)
	if err != nil {
		log.Fatalf("❌ Failed to fetch setup record: %v", err)
	}
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Setup record fetched successfully!\n")
	}

	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\n📄 Current Setup Details:\n")
		fmt.Fprintf(os.Stderr, "   Version: %s\n", existingSetup.Version)
		fmt.Fprintf(os.Stderr, "   Username: %s\n", existingSetup.BDeploy.Username)
		fmt.Fprintf(os.Stderr, "   Network: %s\n", existingSetup.BDeploy.NetworkName)
		fmt.Fprintf(os.Stderr, "   Package: %s\n", existingSetup.BDeploy.PackageName)
		fmt.Fprintf(os.Stderr, "   Type: %s\n", existingSetup.SetupType)
		fmt.Fprintf(os.Stderr, "   Time Zone: %s\n", existingSetup.TimeZone)
		fmt.Fprintf(os.Stderr, "   BSN Group: %s\n", existingSetup.BSNGroupName)
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Step 3: Set network context (use existing network if not specified in config)
	networkName := config.NetworkName
	if networkName == "" {
		networkName = existingSetup.BDeploy.NetworkName
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📡 Setting network context to: %s\n", networkName)
	}
	if err := client.BDeploy.SetNetworkContext(ctx, networkName); err != nil {
		log.Fatalf("❌ Failed to set network context: %v", err)
	}
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Network context set successfully!\n")
	}

	// Step 4: Apply updates from config
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✏️  Applying updates from configuration...\n")
	}
	updatedSetup := applyUpdates(existingSetup, config)

	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\n📝 Updated Setup Details:\n")
		fmt.Fprintf(os.Stderr, "   Version: %s\n", updatedSetup.Version)
		fmt.Fprintf(os.Stderr, "   Username: %s\n", updatedSetup.BDeploy.Username)
		fmt.Fprintf(os.Stderr, "   Network: %s\n", updatedSetup.BDeploy.NetworkName)
		fmt.Fprintf(os.Stderr, "   Package: %s\n", updatedSetup.BDeploy.PackageName)
		fmt.Fprintf(os.Stderr, "   Type: %s\n", updatedSetup.SetupType)
		fmt.Fprintf(os.Stderr, "   Time Zone: %s\n", updatedSetup.TimeZone)
		fmt.Fprintf(os.Stderr, "   BSN Group: %s\n", updatedSetup.BSNGroupName)
		if updatedSetup.Network != nil {
			fmt.Fprintf(os.Stderr, "   Interfaces: %d\n", len(updatedSetup.Network.Interfaces))
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Step 5: Update B-Deploy setup record
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔄 Updating B-Deploy setup record...\n")
	}
	result, err := client.BDeploy.UpdateSetupRecord(ctx, setupID, updatedSetup)
	if err != nil {
		log.Fatalf("❌ Failed to update B-Deploy setup record: %v", err)
	}

	// Output as JSON if requested
	if *jsonFlag {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Step 6: Display results
	fmt.Fprintf(os.Stderr, "✅ B-Deploy setup record updated successfully!\n")

	// Prominently display the setup-id
	fmt.Fprintf(os.Stderr, "\n%s\n", strings.Repeat("=", 70))
	fmt.Fprintf(os.Stderr, "SETUP-ID: %s\n", result.ID)
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("=", 70))

	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "\n📊 Updated Setup Summary:\n")
		fmt.Fprintf(os.Stderr, "   Package Name: %s\n", result.BDeploy.PackageName)
		fmt.Fprintf(os.Stderr, "   Setup Type: %s\n", result.SetupType)
		fmt.Fprintf(os.Stderr, "   BSN Group: %s\n", result.BSNGroupName)
		fmt.Fprintf(os.Stderr, "   Time Zone: %s\n", result.TimeZone)

		if result.Network != nil {
			fmt.Fprintf(os.Stderr, "   Network Interfaces: %d\n", len(result.Network.Interfaces))
			for i, iface := range result.Network.Interfaces {
				fmt.Fprintf(os.Stderr, "     %d. %s (%s) - %s\n", i+1, iface.Name, iface.Type, iface.Proto)
			}
		}
	}

	// Generate setup URLs
	fmt.Fprintf(os.Stderr, "\n🔗 Setup URLs:\n")
	fmt.Fprintf(os.Stderr, "   Web: https://provision.bsn.cloud/setup/%s\n", result.ID)
	fmt.Fprintf(os.Stderr, "   API: https://provision.bsn.cloud/rest-setup/v3/setup/%s\n", result.ID)

	fmt.Fprintf(os.Stderr, "\n🎉 Setup record updated successfully!\n")
}

// loadConfig reads and parses the JSON configuration file
func loadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &config, nil
}

// applyUpdates merges config updates into the existing setup record
func applyUpdates(existing *bscontrol.BDeploySetupRecord, config *Config) *bscontrol.BDeploySetupRecord {
	// Start with existing setup
	updated := *existing

	// Apply updates from config (only update non-empty fields)
	if config.NetworkName != "" {
		updated.BDeploy.NetworkName = config.NetworkName
	}
	if config.Username != "" {
		updated.BDeploy.Username = config.Username
	}
	if config.PackageName != "" {
		updated.BDeploy.PackageName = config.PackageName
	}
	if config.SetupType != "" {
		updated.SetupType = config.SetupType
	}
	if config.TimeZone != "" {
		updated.TimeZone = config.TimeZone
	}
	if config.BSNGroupName != "" {
		updated.BSNGroupName = config.BSNGroupName
	}

	// Update network configuration if provided
	if len(config.Network.TimeServers) > 0 || len(config.Network.Interfaces) > 0 {
		if updated.Network == nil {
			updated.Network = &bscontrol.NetworkConfig{}
		}

		if len(config.Network.TimeServers) > 0 {
			updated.Network.TimeServers = config.Network.TimeServers
		}

		if len(config.Network.Interfaces) > 0 {
			interfaces := make([]bscontrol.NetworkInterface, len(config.Network.Interfaces))
			for i, iface := range config.Network.Interfaces {
				interfaces[i] = bscontrol.NetworkInterface{
					ID:                     iface.ID,
					Name:                   iface.Name,
					Type:                   iface.Type,
					Proto:                  iface.Proto,
					ContentDownloadEnabled: iface.ContentDownloadEnabled,
					HealthReportingEnabled: iface.HealthReportingEnabled,
				}
			}
			updated.Network.Interfaces = interfaces
		}
	}

	return &updated
}

// findSetupByName searches for a setup record by package name
func findSetupByName(ctx context.Context, client *bscontrol.Client, setupName string, networkName string, verbose bool, jsonMode bool) (string, error) {
	// Get all setup records in the network
	records, err := client.BDeploy.GetSetupRecords(ctx,
		bscontrol.WithNetworkName(networkName),
	)
	if err != nil {
		return "", fmt.Errorf("failed to list setup records: %w", err)
	}

	if verbose && !jsonMode {
		fmt.Fprintf(os.Stderr, "   Searching through %d setup record(s)...\n", len(records.Items))
	}

	// Search for matching setup name (case-insensitive)
	var matches []types.BDeployRecord
	for _, record := range records.Items {
		if strings.EqualFold(record.PackageName, setupName) {
			matches = append(matches, record)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no setup found with name '%s'", setupName)
	}

	if len(matches) > 1 && !jsonMode {
		fmt.Fprintf(os.Stderr, "⚠️  Warning: Found %d setups with name '%s':\n", len(matches), setupName)
		for i, match := range matches {
			fmt.Fprintf(os.Stderr, "   %d. ID: %s, Network: %s, User: %s\n",
				i+1, match.ID, match.NetworkName, match.Username)
		}
		fmt.Fprintf(os.Stderr, "   Using the first match...\n")
	}

	return matches[0].ID, nil
}
