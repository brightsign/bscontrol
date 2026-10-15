package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/brightsign/bscontrol"
	"github.com/brightsign/bscontrol/examples/internal/setuptemplate"
)

func main() {
	var (
		helpFlag     = flag.Bool("help", false, "Display usage information")
		jsonFlag     = flag.Bool("json", false, "Output as JSON")
		verboseFlag  = flag.Bool("verbose", false, "Show detailed information")
		debugFlag    = flag.Bool("debug", false, "Show raw API request and response details")
		networkFlag  = flag.String("network", "", "Network name (uses BS_NETWORK env var if not provided)")
		usernameFlag = flag.String("username", "", "Username to filter setups (optional)")
		packageFlag = flag.String("package-name", "", "Package name to filter setups (optional, env: BS_PACKAGE_NAME)")
	)

	flag.StringVar(packageFlag, "package", "", "Alias for --package-name")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Lists all B-Deploy setup records for the specified network.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional, can use --network flag)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  List all setups using BS_NETWORK environment variable:\n")
		fmt.Fprintf(os.Stderr, "    export BS_NETWORK=\"Production Network\"\n")
		fmt.Fprintf(os.Stderr, "    %s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  List all setups for a network:\n")
		fmt.Fprintf(os.Stderr, "    %s --network \"Production Network\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  List setups with detailed output:\n")
		fmt.Fprintf(os.Stderr, "    %s --network \"Production Network\" --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Filter by username:\n")
		fmt.Fprintf(os.Stderr, "    %s --network \"Production Network\" --username \"admin@example.com\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Filter by package name:\n")
		fmt.Fprintf(os.Stderr, "    %s --network \"Production Network\" --package \"retail-display-v1\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --network \"Production Network\" --json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Show debug information (API request/response):\n")
		fmt.Fprintf(os.Stderr, "    %s --network \"Production Network\" --debug\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Resolve package name from flag or environment variable
	resolvedPackageName := setuptemplate.ResolveVar(*packageFlag, setuptemplate.EnvPackageName)

	// Create client with optional debug mode
	clientOpts := []bscontrol.Option{bscontrol.WithEnv()}
	if *debugFlag {
		clientOpts = append(clientOpts, bscontrol.WithDebug(true))
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔧 Creating BSN.cloud client...\n")
	}
	client, err := bscontrol.New(clientOpts...)
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

	// Get network name: check flag first, then BS_NETWORK env var, then prompt
	networkName := getNetworkName(*networkFlag, client, ctx, *verboseFlag, *jsonFlag)

	// Step 2: Set network context
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📡 Setting network context to: %s\n", networkName)
	}
	if err := client.BDeploy.SetNetworkContext(ctx, networkName); err != nil {
		log.Fatalf("❌ Failed to set network context: %v", err)
	}
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Network context set successfully!\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Step 3: Build options for filtering
	var opts []bscontrol.BDeployListOption
	opts = append(opts, bscontrol.WithNetworkName(networkName))

	if *usernameFlag != "" {
		opts = append(opts, bscontrol.WithUsername(*usernameFlag))
		if *verboseFlag && !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔍 Filtering by username: %s\n", *usernameFlag)
		}
	}

	if resolvedPackageName != "" {
		opts = append(opts, bscontrol.WithPackageName(resolvedPackageName))
		if *verboseFlag && !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔍 Filtering by package name: %s\n", resolvedPackageName)
		}
	}

	// Debug: Show the request that will be made
	if *debugFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\n🔍 DEBUG: Request Parameters:\n")
		fmt.Fprintf(os.Stderr, "   Endpoint: https://provision.bsn.cloud/rest-setup/v3/setup\n")
		fmt.Fprintf(os.Stderr, "   NetworkName: %s\n", networkName)
		if *usernameFlag != "" {
			fmt.Fprintf(os.Stderr, "   username: %s\n", *usernameFlag)
		}
		if resolvedPackageName != "" {
			fmt.Fprintf(os.Stderr, "   packageName: %s\n", resolvedPackageName)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Step 4: Get setup records
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📋 Retrieving setup records...\n")
	}
	records, err := client.BDeploy.GetSetupRecords(ctx, opts...)
	if err != nil {
		log.Fatalf("❌ Failed to get setup records: %v", err)
	}

	// Output as JSON if requested
	if *jsonFlag {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(records); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Step 5: Display results
	if records.TotalCount == 0 {
		fmt.Fprintf(os.Stderr, "\n📭 No setup records found for the specified network\n")
		if *usernameFlag != "" || resolvedPackageName != "" {
			fmt.Fprintf(os.Stderr, "   Try removing filters to see all setups\n")
		}
		return
	}

	fmt.Fprintf(os.Stderr, "\n✅ Found %d setup record(s)\n\n", records.TotalCount)

	if *verboseFlag {
		// Verbose output with all details
		displayDetailedRecords(records.Items)
	} else {
		// Standard table output
		displayTableRecords(records.Items)
	}
}

// displayTableRecords shows setup records in a compact table format
func displayTableRecords(items []bscontrol.BDeployRecord) {
	w := tabwriter.NewWriter(os.Stderr, 0, 0, 3, ' ', 0)

	// Print header
	fmt.Fprintln(w, "SETUP-ID\tPACKAGE NAME\tSETUP TYPE\tGROUP\tNETWORK\t")
	fmt.Fprintln(w, strings.Repeat("-", 40)+"\t"+strings.Repeat("-", 30)+"\t"+strings.Repeat("-", 15)+"\t"+strings.Repeat("-", 20)+"\t"+strings.Repeat("-", 20)+"\t")

	// Print each record
	for _, record := range items {
		setupID := record.ID
		packageName := record.PackageName
		setupType := record.SetupType
		group := record.BSNGroupName
		network := record.NetworkName

		// Use defaults for empty fields
		if setupID == "" {
			setupID = "(no ID)"
		}
		if packageName == "" {
			packageName = "(no package name)"
		}
		if setupType == "" {
			setupType = "(not specified)"
		}
		if group == "" {
			group = "(default)"
		}
		if network == "" {
			network = "(not specified)"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t\n", setupID, packageName, setupType, group, network)
	}

	w.Flush()
	fmt.Fprintf(os.Stderr, "\n")
}

// displayDetailedRecords shows setup records with full details
func displayDetailedRecords(items []bscontrol.BDeployRecord) {
	for i, record := range items {
		fmt.Fprintf(os.Stderr, "Setup Record #%d:\n", i+1)
		fmt.Fprintf(os.Stderr, "  Setup-ID:     %s\n", valueOrDefault(record.ID, "(no ID)"))
		fmt.Fprintf(os.Stderr, "  Package Name: %s\n", valueOrDefault(record.PackageName, "(not specified)"))
		fmt.Fprintf(os.Stderr, "  Setup Type:   %s\n", valueOrDefault(record.SetupType, "(not specified)"))
		fmt.Fprintf(os.Stderr, "  BSN Group:    %s\n", valueOrDefault(record.BSNGroupName, "(default)"))
		fmt.Fprintf(os.Stderr, "  Network:      %s\n", valueOrDefault(record.NetworkName, "(not specified)"))
		fmt.Fprintf(os.Stderr, "  Username:     %s\n", valueOrDefault(record.Username, "(not specified)"))

		if record.CreatedDate != "" {
			fmt.Fprintf(os.Stderr, "  Created:      %s\n", record.CreatedDate)
		}
		if record.UpdatedDate != "" {
			fmt.Fprintf(os.Stderr, "  Updated:      %s\n", record.UpdatedDate)
		}
		fmt.Fprintf(os.Stderr, "  Active:       %v\n", record.IsActive)

		fmt.Fprintf(os.Stderr, "  Setup URL:    https://provision.bsn.cloud/setup/%s\n", record.ID)

		if i < len(items)-1 {
			fmt.Fprintf(os.Stderr, "\n")
		}
	}
}

// valueOrDefault returns the value if non-empty, otherwise returns the default
func valueOrDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

// getNetworkName determines the network name to use by checking in order:
// 1. --network flag
// 2. BS_NETWORK environment variable
// 3. Prompt user to select from available networks
func getNetworkName(requestedNetwork string, client *bscontrol.Client, ctx context.Context, verbose bool, jsonMode bool) string {
	// If network was specified via flag, use it
	if requestedNetwork != "" {
		if verbose && !jsonMode {
			fmt.Fprintf(os.Stderr, "📡 Using network from --network flag: %s\n", requestedNetwork)
		}
		return requestedNetwork
	}

	// Check BS_NETWORK environment variable
	if envNetwork := os.Getenv("BS_NETWORK"); envNetwork != "" {
		if verbose && !jsonMode {
			fmt.Fprintf(os.Stderr, "📡 Using network from BS_NETWORK env var: %s\n", envNetwork)
		}
		return envNetwork
	}

	// Need to select a network interactively
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "📡 Getting available networks...\n")
	}

	networks, err := client.GetNetworks(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to get networks: %v", err)
	}

	if len(networks) == 0 {
		log.Fatalf("❌ No networks available")
	}

	// If only one network, use it automatically
	if len(networks) == 1 {
		networkName := networks[0].Name
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "📡 Using only available network: %s (ID: %d)\n", networkName, networks[0].ID)
		}
		return networkName
	}

	// Show available networks and let user choose
	fmt.Fprintf(os.Stderr, "\n📡 Available networks:\n")
	for i, network := range networks {
		fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
		if verbose && !jsonMode {
			fmt.Fprintf(os.Stderr, "     Created: %s, Modified: %s\n",
				network.CreationDate.Format("2006-01-02"),
				network.LastModifiedDate.Format("2006-01-02"))
		}
	}

	// Get user selection
	fmt.Fprintf(os.Stderr, "\nSelect network (1-%d): ", len(networks))
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		log.Fatalf("❌ Failed to read input")
	}

	selection, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || selection < 1 || selection > len(networks) {
		log.Fatalf("❌ Invalid selection: must be between 1 and %d", len(networks))
	}

	selectedNetwork := networks[selection-1]
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "📡 Selected network: %s (ID: %d)\n\n", selectedNetwork.Name, selectedNetwork.ID)
	}

	return selectedNetwork.Name
}
