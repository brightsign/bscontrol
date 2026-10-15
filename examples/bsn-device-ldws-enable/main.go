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
	"time"

	"github.com/brightsign/bscontrol"
)

func main() {
	var (
		helpFlag    = flag.Bool("help", false, "Display usage information")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
		verboseFlag = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag *string
		serialFlag  = flag.String("serial", "", "Device serial number")
		idFlag      = flag.Int("id", 0, "Device ID")
		confirmFlag = flag.Bool("y", false, "Skip confirmation prompt")
		actionFlag  *string
		enableFlag  = flag.Bool("enable", false, "Enable local DWS")
		disableFlag = flag.Bool("disable", false, "Disable local DWS")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Set up action flags to point to the same variable
	actionFlag = flag.String("action", "", "Action: get, enable, disable")
	flag.StringVar(actionFlag, "a", "", "Action: get, enable, disable [alias for --action]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for managing the local Diagnostic Web Server (DWS) on BrightSign devices.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get current local DWS status:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --action get\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Enable local DWS:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --enable\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Disable local DWS:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --disable -y\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Using device ID instead of serial:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 --action get\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --action get --json\n", os.Args[0])
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

	// Determine action
	var action string
	if *enableFlag {
		action = "enable"
	} else if *disableFlag {
		action = "disable"
	} else if *actionFlag != "" {
		action = strings.ToLower(*actionFlag)
	} else {
		action = "get" // default action
	}

	// Validate action
	switch action {
	case "get", "enable", "disable":
		// valid actions
	default:
		fmt.Fprintf(os.Stderr, "❌ Error: Invalid action '%s'. Valid actions: get, enable, disable\n\n", action)
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

	// Handle different actions
	switch action {
	case "get":
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "📊 Getting local DWS status for device with %s\n", deviceInfo)
		}
		handleGetStatus(ctx, client, *serialFlag, *idFlag, *verboseFlag, *jsonFlag)

	case "enable":
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔧 Enabling local DWS for device with %s\n", deviceInfo)
		}
		if !*confirmFlag && !*jsonFlag {
			if !confirmAction("enable", deviceInfo) {
				fmt.Fprintf(os.Stderr, "❌ Operation cancelled\n")
				os.Exit(0)
			}
		}
		handleSetStatus(ctx, client, *serialFlag, *idFlag, true, *verboseFlag, *jsonFlag)

	case "disable":
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔧 Disabling local DWS for device with %s\n", deviceInfo)
		}
		if !*confirmFlag && !*jsonFlag {
			if !confirmAction("disable", deviceInfo) {
				fmt.Fprintf(os.Stderr, "❌ Operation cancelled\n")
				os.Exit(0)
			}
		}
		handleSetStatus(ctx, client, *serialFlag, *idFlag, false, *verboseFlag, *jsonFlag)
	}
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, verbose bool, jsonMode bool) error {
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

func confirmAction(action, deviceInfo string) bool {
	fmt.Printf("⚠️  WARNING: This will %s the local DWS on the device with %s.\n", action, deviceInfo)

	if action == "enable" {
		fmt.Println("🔧 ENABLE will allow direct access to the device's diagnostic web server.")
		fmt.Println("   This may expose diagnostic information on the local network.")
	} else if action == "disable" {
		fmt.Println("🔧 DISABLE will prevent direct access to the device's diagnostic web server.")
		fmt.Println("   You will need to use rDWS for diagnostics after this change.")
	}

	fmt.Println("The device may reboot to apply this change.")
	fmt.Printf("\nAre you sure you want to %s local DWS? (y/N): ", action)

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return false
	}

	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return response == "y" || response == "yes"
}

func handleGetStatus(ctx context.Context, client *bscontrol.Client, serial string, deviceID int, verbose bool, jsonMode bool) {
	// TODO: Implement GET /control/local-dws/ when it's available in the SDK
	// For now, we'll show what the equivalent curl command would be

	config := client.Config()

	var destinationName string
	if serial != "" {
		destinationName = serial
	} else {
		destinationName = fmt.Sprintf("%d", deviceID)
	}

	curlCommand := fmt.Sprintf("curl -X GET '%s/control/local-dws/?destinationType=player&destinationName=%s' \\\n",
		config.RDWSBaseURL, destinationName) +
		"  -H \"Authorization: Bearer $BSN_ACCESS_TOKEN\" \\\n" +
		"  -H 'Accept: application/json'\n"

	if jsonMode {
		result := map[string]interface{}{
			"action":          "get",
			"destinationType": "player",
			"destinationName": destinationName,
			"rdwsBaseURL":     config.RDWSBaseURL,
			"curlCommand":     curlCommand,
			"note":            "This endpoint is not yet implemented in the SDK. Use the curl command shown.",
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "📋 Equivalent curl command to get local DWS status:\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "%s", curlCommand)
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "💡 Make sure to run 'source ./.token' first to set the BSN_ACCESS_TOKEN environment variable.\n")
}

func handleSetStatus(ctx context.Context, client *bscontrol.Client, serial string, deviceID int, enable bool, verbose bool, jsonMode bool) {
	// TODO: Implement PUT /control/local-dws/ when it's available in the SDK
	// For now, we'll show what the equivalent curl command would be

	config := client.Config()

	action := "disable"
	if enable {
		action = "enable"
	}

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
		fmt.Sprintf("  -d '{\n    \"data\": {\n      \"enable\": %t\n    }\n  }'\n", enable)

	expectedResponse := fmt.Sprintf("The device will %s local DWS access", action)
	if enable {
		expectedResponse += " and may reboot"
	}
	expectedResponse += "."

	if jsonMode {
		result := map[string]interface{}{
			"action":           action,
			"destinationType":  "player",
			"destinationName":  destinationName,
			"enable":           enable,
			"rdwsBaseURL":      config.RDWSBaseURL,
			"curlCommand":      curlCommand,
			"expectedResponse": expectedResponse,
			"note":             "This endpoint is not yet implemented in the SDK. Use the curl command shown.",
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "📋 Equivalent curl command to %s local DWS:\n", action)
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "%s", curlCommand)
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "💡 Make sure to run 'source ./.token' first to set the BSN_ACCESS_TOKEN environment variable.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "🎯 Expected Response: %s\n", expectedResponse)
}
