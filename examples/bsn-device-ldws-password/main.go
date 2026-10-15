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
	"syscall"
	"time"

	"github.com/brightsign/bscontrol"
	"golang.org/x/term"
)

func main() {
	var (
		helpFlag        = flag.Bool("help", false, "Display usage information")
		jsonFlag        = flag.Bool("json", false, "Output as JSON")
		verboseFlag     = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag     = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag     *string
		serialFlag      = flag.String("serial", "", "Device serial number")
		idFlag          = flag.Int("id", 0, "Device ID")
		confirmFlag     = flag.Bool("y", false, "Skip confirmation prompt")
		passwordFlag    = flag.String("password", "", "New DWS password")
		oldPasswordFlag = flag.String("old-password", "", "Current DWS password (empty if no current password)")
		enableOnlyFlag  = flag.Bool("enable-only", false, "Only enable local DWS, don't set password")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Set up password flags with aliases
	flag.StringVar(passwordFlag, "p", "", "New DWS password [alias for --password]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to enable local Diagnostic Web Server (DWS) and set its password on BrightSign devices.\n")
		fmt.Fprintf(os.Stderr, "This tool performs two operations in sequence:\n")
		fmt.Fprintf(os.Stderr, "  1. Enable local DWS (PUT /control/local-dws/)\n")
		fmt.Fprintf(os.Stderr, "  2. Set DWS password (PUT /control/dws-password/)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Enable DWS and set password (interactive):\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Enable DWS and set password (command line):\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 -p mypassword\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Enable DWS and change password from existing:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 -p newpass --old-password oldpass\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Only enable DWS without setting password:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --enable-only\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Remove DWS password (set to blank):\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 -p \"\" --old-password currentpass\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 -p mypass -y --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Validate input
	if *serialFlag == "" && *idFlag == 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: Must specify either --serial or --id\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *serialFlag != "" && *idFlag != 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: Cannot specify both --serial and --id\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}

	// Add network if specified
	if *networkFlag != "" {
		opts = append(opts, bscontrol.WithNetwork(*networkFlag))
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

	// Authenticate
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

	// Handle network selection
	if err := handleNetworkSelection(ctx, client, *networkFlag, *verboseFlag, *jsonFlag); err != nil {
		log.Fatalf("❌ Network selection failed: %v", err)
	}

	// Get device info for display
	var deviceInfo string
	if *serialFlag != "" {
		deviceInfo = fmt.Sprintf("serial %s", *serialFlag)
	} else {
		deviceInfo = fmt.Sprintf("ID %d", *idFlag)
	}

	// Confirmation prompt unless -y flag is used
	if !*confirmFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "⚠️  This operation will:\n")
		fmt.Fprintf(os.Stderr, "   1. Enable local DWS on device with %s\n", deviceInfo)
		if !*enableOnlyFlag {
			fmt.Fprintf(os.Stderr, "   2. Set/change the DWS password\n")
		}
		fmt.Fprintf(os.Stderr, "   The device may reboot during this process.\n\n")

		fmt.Fprint(os.Stderr, "Are you sure you want to continue? (y/N): ")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			fmt.Fprintf(os.Stderr, "❌ Aborted\n")
			os.Exit(1)
		}

		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response != "y" && response != "yes" {
			fmt.Fprintf(os.Stderr, "❌ Operation cancelled\n")
			os.Exit(0)
		}
	}

	// Step 1: Enable local DWS
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔧 Step 1: Enabling local DWS for device with %s\n", deviceInfo)
	}
	enableCmd := handleEnableDWS(ctx, client, *serialFlag, *idFlag, *verboseFlag, *jsonFlag)

	if *enableOnlyFlag {
		if *jsonFlag {
			result := map[string]interface{}{
				"success":          true,
				"operationMode":    "enable-only",
				"enableDWSCommand": enableCmd,
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(result); err != nil {
				log.Fatalf("Failed to encode JSON: %v", err)
			}
			return
		}
		fmt.Fprintf(os.Stderr, "✅ Local DWS enabled successfully!\n")
		fmt.Fprintf(os.Stderr, "💡 You can now access the diagnostic web server locally on the device.\n")
		return
	}

	// Step 2: Set DWS password
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\n🔐 Step 2: Setting DWS password for device with %s\n", deviceInfo)
	}

	// Get password if not provided
	password := *passwordFlag
	if password == "" && !*jsonFlag {
		password = promptForPassword("Enter new DWS password (or press Enter for no password): ")
	}

	// Get old password if not provided
	oldPassword := *oldPasswordFlag
	if oldPassword == "" && !*confirmFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "💡 If the device currently has a DWS password, enter it below.\n")
		fmt.Fprintf(os.Stderr, "   If there's no current password, just press Enter.\n")
		oldPassword = promptForPassword("Enter current DWS password (or press Enter if none): ")
	}

	passwordCmd := handleSetDWSPassword(ctx, client, *serialFlag, *idFlag, password, oldPassword, *verboseFlag, *jsonFlag)

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success":            true,
			"enableDWSCommand":   enableCmd,
			"setPasswordCommand": passwordCmd,
			"passwordProtection": password != "",
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "\n✅ Local DWS setup completed successfully!\n")
	fmt.Fprintf(os.Stderr, "🔐 DWS is now enabled with %s\n",
		func() string {
			if password == "" {
				return "no password protection"
			}
			return "password protection"
		}())
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, verbose, jsonMode bool) error {
	// Check if network is already set
	if client.IsNetworkSet() {
		if current, err := client.GetCurrentNetwork(ctx); err == nil {
			if !jsonMode {
				fmt.Fprintf(os.Stderr, "📡 Using network: %s (ID: %d)\n", current.Name, current.ID)
			}
			return nil
		}
	}

	// If no network flag was provided, check BS_NETWORK environment variable
	if requestedNetwork == "" {
		if envNetwork := os.Getenv("BS_NETWORK"); envNetwork != "" {
			requestedNetwork = envNetwork
			if !jsonMode {
				fmt.Fprintf(os.Stderr, "Using network from BS_NETWORK environment variable\n")
			}
		}
	}

	// Get available networks
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "📡 Getting available networks...\n")
	}

	networks, err := client.GetNetworks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get networks: %w", err)
	}

	if len(networks) == 0 {
		return fmt.Errorf("no networks available")
	}

	// If a specific network was requested, try to find it
	if requestedNetwork != "" {
		for _, network := range networks {
			if strings.EqualFold(network.Name, requestedNetwork) {
				if !jsonMode {
					fmt.Fprintf(os.Stderr, "📡 Using requested network: %s (ID: %d)\n", network.Name, network.ID)
				}
				return client.SetNetworkByID(ctx, network.ID)
			}
		}

		// Network not found - show error and fall back to interactive selection
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "❌ Network '%s' not found. Available networks:\n", requestedNetwork)
			for i, network := range networks {
				fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
			}
			fmt.Fprintf(os.Stderr, "\n")
		} else {
			return fmt.Errorf("network '%s' not found", requestedNetwork)
		}
	}

	// If only one network and no specific network requested, use it automatically
	if len(networks) == 1 && requestedNetwork == "" {
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "📡 Using network: %s (ID: %d)\n", networks[0].Name, networks[0].ID)
		}
		return client.SetNetworkByID(ctx, networks[0].ID)
	}

	// In JSON mode, cannot do interactive selection
	if jsonMode {
		return fmt.Errorf("network selection required: use --network flag or BS_NETWORK environment variable")
	}

	// Show available networks and let user choose
	if requestedNetwork == "" {
		fmt.Fprintf(os.Stderr, "📡 Available networks:\n")
		for i, network := range networks {
			fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
			if verbose {
				fmt.Fprintf(os.Stderr, "     Created: %s, Modified: %s\n",
					network.CreationDate.Format("2006-01-02"),
					network.LastModifiedDate.Format("2006-01-02"))
			}
		}
	}

	// Get user selection
	fmt.Fprint(os.Stderr, "Select network (1-"+strconv.Itoa(len(networks))+"): ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	selection, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || selection < 1 || selection > len(networks) {
		return fmt.Errorf("invalid selection: must be between 1 and %d", len(networks))
	}

	selectedNetwork := networks[selection-1]
	fmt.Fprintf(os.Stderr, "📡 Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)

	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}

func promptForPassword(prompt string) string {
	fmt.Print(prompt)

	// Try to read password without echo
	if term.IsTerminal(int(syscall.Stdin)) {
		password, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintf(os.Stderr, "\n") // Add newline after password input
		if err != nil {
			log.Fatalf("❌ Failed to read password: %v", err)
		}
		return string(password)
	}

	// Fallback to regular input if not a terminal
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		log.Fatalf("❌ Failed to read input")
	}
	return strings.TrimSpace(scanner.Text())
}

func handleEnableDWS(ctx context.Context, client *bscontrol.Client, serial string, deviceID int, verbose, jsonMode bool) string {
	config := client.Config()

	var destinationName string
	if serial != "" {
		destinationName = serial
	} else {
		destinationName = fmt.Sprintf("%d", deviceID)
	}

	curlCommand := fmt.Sprintf("curl -X PUT '%s/control/local-dws/?destinationType=player&destinationName=%s' \\\n",
		config.RDWSBaseURL, destinationName) +
		"  -H \"Authorization: Bearer $BSN_ACCESS_TOKEN\" \\\n" +
		"  -H 'Accept: application/json' \\\n" +
		"  -H 'Content-Type: application/json' \\\n" +
		"  -d '{\n" +
		"    \"data\": {\n" +
		"      \"enable\": true\n" +
		"    }\n" +
		"  }'\n"

	if !jsonMode {
		fmt.Fprintf(os.Stderr, "📋 Equivalent curl command to enable local DWS:\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "%s", curlCommand)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "💡 Make sure to run 'source ./.token' first to set the BSN_ACCESS_TOKEN environment variable.\n")
		fmt.Fprintf(os.Stderr, "🎯 Expected Response: Local DWS will be enabled and the device may reboot.\n")
	}

	return curlCommand
}

func handleSetDWSPassword(ctx context.Context, client *bscontrol.Client, serial string, deviceID int, password, oldPassword string, verbose, jsonMode bool) string {
	config := client.Config()

	var destinationName string
	if serial != "" {
		destinationName = serial
	} else {
		destinationName = fmt.Sprintf("%d", deviceID)
	}

	curlCommand := fmt.Sprintf("curl -X PUT '%s/control/dws-password/?destinationType=player&destinationName=%s' \\\n",
		config.RDWSBaseURL, destinationName) +
		"  -H \"Authorization: Bearer $BSN_ACCESS_TOKEN\" \\\n" +
		"  -H 'Accept: application/json' \\\n" +
		"  -H 'Content-Type: application/json' \\\n" +
		fmt.Sprintf("  -d '{\n    \"data\": {\n      \"password\": \"%s\",\n      \"previous_password\": \"%s\"\n    }\n  }'\n",
			password, oldPassword)

	if !jsonMode {
		fmt.Fprintf(os.Stderr, "📋 Equivalent curl command to set DWS password:\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "%s", curlCommand)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "💡 Make sure to run 'source ./.token' first to set the BSN_ACCESS_TOKEN environment variable.\n")

		if password == "" {
			fmt.Fprintf(os.Stderr, "🎯 Expected Response: DWS password will be removed (no password required).\n")
		} else {
			fmt.Fprintf(os.Stderr, "🎯 Expected Response: DWS password will be set and the device may reboot.\n")
		}
	}

	return curlCommand
}
