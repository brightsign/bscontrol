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
		verboseFlag = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag *string
		serialFlag  = flag.String("serial", "", "Reboot device with serial number")
		idFlag      = flag.Int("id", 0, "Reboot device with ID")
		confirmFlag *bool
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
		typeFlag    *string
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Set up confirm flags to point to the same variable
	confirmFlag = flag.Bool("y", false, "Skip confirmation prompt")
	flag.BoolVar(confirmFlag, "force", false, "Skip confirmation prompt [alias for -y]")

	// Set up type flags to point to the same variable
	typeFlag = flag.String("type", "normal", "Reboot type: normal, crash, factoryreset, disableautorun")
	flag.StringVar(typeFlag, "t", "normal", "Reboot type: normal, crash, factoryreset, disableautorun [alias for --type]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for remotely rebooting BrightSign devices.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Normal reboot by serial number:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Factory reset reboot without confirmation:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 -t factoryreset --force\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Crash report reboot with specific network:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 -n \"MyNetwork\" -t crash\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Disable autorun reboot:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --type disableautorun\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get reboot result as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 -y --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Require -y flag when using --json (cannot prompt for confirmation)
	if *jsonFlag && !*confirmFlag {
		fmt.Fprintf(os.Stderr, "Error: -y flag is required when using --json (cannot prompt for confirmation)\n\n")
		flag.Usage()
		os.Exit(1)
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

	// Validate reboot type
	var rebootType bscontrol.RebootType
	switch strings.ToLower(*typeFlag) {
	case "normal":
		rebootType = bscontrol.RebootTypeNormal
	case "crash":
		rebootType = bscontrol.RebootTypeCrash
	case "factoryreset":
		rebootType = bscontrol.RebootTypeFactoryReset
	case "disableautorun":
		rebootType = bscontrol.RebootTypeDisableAutorun
	default:
		fmt.Fprintf(os.Stderr, "❌ Error: Invalid reboot type '%s'. Valid types: normal, crash, factoryreset, disableautorun\n\n", *typeFlag)
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

	// Get device info and confirm reboot
	var deviceInfo string
	if *serialFlag != "" {
		deviceInfo = fmt.Sprintf("serial number '%s'", *serialFlag)
	} else {
		deviceInfo = fmt.Sprintf("ID %d", *idFlag)
	}

	// Confirmation prompt (unless -y flag is used)
	if !*confirmFlag {
		fmt.Printf("⚠️  WARNING: This will perform a %s reboot of the device with %s.\n", *typeFlag, deviceInfo)

		// Add specific warnings for destructive reboot types
		switch rebootType {
		case bscontrol.RebootTypeFactoryReset:
			fmt.Printf("🚨 FACTORY RESET will erase ALL settings, networking, security, and application data!\n")
		case bscontrol.RebootTypeCrash:
			fmt.Printf("📊 CRASH REPORT will generate diagnostic files and may take longer than normal reboot.\n")
		case bscontrol.RebootTypeDisableAutorun:
			fmt.Printf("🔧 DISABLE AUTORUN will prevent the current autorun script from starting after reboot.\n")
		}

		fmt.Printf("The device will be temporarily unavailable during the reboot process.\n\n")
		fmt.Print("Are you sure you want to continue? (y/N): ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			fmt.Println("❌ Aborted")
			os.Exit(1)
		}

		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response != "y" && response != "yes" {
			fmt.Println("❌ Reboot cancelled")
			os.Exit(0)
		}
	}

	// Perform reboot
	var rebootResponse *bscontrol.RebootResponse
	if *serialFlag != "" {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔄 Rebooting device with serial: %s (type: %s)\n", *serialFlag, *typeFlag)
		}
		rebootResponse, err = client.Devices.RebootBySerial(ctx, *serialFlag, rebootType)
		if err != nil {
			log.Fatalf("❌ Failed to reboot device: %v", err)
		}
	} else {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔄 Rebooting device with ID: %d (type: %s)\n", *idFlag, *typeFlag)
		}
		rebootResponse, err = client.Devices.Reboot(ctx, *idFlag, rebootType)
		if err != nil {
			log.Fatalf("❌ Failed to reboot device: %v", err)
		}
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success":      true,
			"status":       rebootResponse.Status,
			"device_id":    rebootResponse.DeviceID,
			"serial":       rebootResponse.Serial,
			"message":      rebootResponse.Message,
			"reboot_type":  *typeFlag,
			"timestamp":    rebootResponse.Timestamp,
			"reboot_time":  rebootResponse.RebootTime,
			"operation_id": rebootResponse.OperationID,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Display results
	fmt.Fprintf(os.Stderr, "\n🎯 Reboot Request Status:\n")
	printRebootResponse(rebootResponse, *verboseFlag)
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

func printRebootResponse(response *bscontrol.RebootResponse, verbose bool) {
	// Status indicator with emoji
	statusIcon := "✅"
	switch strings.ToLower(response.Status) {
	case "success":
		statusIcon = "✅"
	case "failed":
		statusIcon = "❌"
	case "pending":
		statusIcon = "⏳"
	default:
		statusIcon = "❓"
	}

	fmt.Printf("  Status:           %s %s\n", statusIcon, response.Status)

	if response.DeviceID != "" {
		fmt.Printf("  Device ID:        %s\n", response.DeviceID)
	}

	if response.Serial != "" {
		fmt.Printf("  Serial:           %s\n", response.Serial)
	}

	if response.Message != "" {
		fmt.Printf("  Message:          %s\n", response.Message)
	}

	if !response.Timestamp.IsZero() {
		fmt.Printf("  Request Time:     %s\n", response.Timestamp.Format("2006-01-02 15:04:05 MST"))
	}

	if !response.RebootTime.IsZero() {
		fmt.Printf("  Reboot Time:      %s\n", response.RebootTime.Format("2006-01-02 15:04:05 MST"))
	}

	if verbose {
		if response.OperationID != "" {
			fmt.Printf("  Operation ID:     %s\n", response.OperationID)
		}
	}

	fmt.Fprintf(os.Stderr, "\n")

	// Additional information
	switch strings.ToLower(response.Status) {
	case "success":
		fmt.Println("✅ Reboot command sent successfully!")
		fmt.Println("   The device should restart within a few moments.")
		fmt.Println("   It may take 1-2 minutes for the device to come back online.")
	case "failed":
		fmt.Println("❌ Reboot command failed!")
		if response.Message != "" {
			fmt.Printf("   Reason: %s\n", response.Message)
		}
	case "pending":
		fmt.Println("⏳ Reboot command is pending...")
		if response.OperationID != "" {
			fmt.Printf("   You can check the status using operation ID: %s\n", response.OperationID)
		}
	default:
		fmt.Printf("❓ Reboot status: %s\n", response.Status)
	}
}
