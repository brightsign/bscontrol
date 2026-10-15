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
		debugFlag   = flag.Bool("debug", false, "Enable debug logging of HTTP requests/responses")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag *string
		serialFlag  = flag.String("serial", "", "Device serial number")
		idFlag      = flag.Int("id", 0, "Device ID")
		deviceFlag  = flag.String("device", "sd", "Storage device to reformat (sd, ssd, usb)")
		confirmFlag = flag.Bool("y", false, "Skip confirmation prompt")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for remotely reformatting storage devices on BrightSign players.\n\n")
		fmt.Fprintf(os.Stderr, "⚠️  WARNING: This operation will ERASE ALL DATA on the specified storage device!\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Storage Devices:\n")
		fmt.Fprintf(os.Stderr, "  sd                  SD card (default)\n")
		fmt.Fprintf(os.Stderr, "  ssd                 Internal SSD\n")
		fmt.Fprintf(os.Stderr, "  usb                 USB storage\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Reformat SD card:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --device sd\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Reformat SSD with confirmation skip:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --device ssd -y\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Reformat USB storage with specific network:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 -n \"MyNetwork\" --device usb\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get reformat result as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --device sd -y --json\n", os.Args[0])
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
		fmt.Fprintf(os.Stderr, "Error: Must specify either --serial or --id\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *serialFlag != "" && *idFlag != 0 {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify both --serial and --id\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate device name
	validDevices := map[string]bool{"sd": true, "ssd": true, "usb": true}
	deviceName := strings.ToLower(*deviceFlag)
	if !validDevices[deviceName] {
		fmt.Fprintf(os.Stderr, "Error: Invalid storage device '%s'. Valid devices: sd, ssd, usb\n\n", *deviceFlag)
		flag.Usage()
		os.Exit(1)
	}

	// Get network name from flag or environment variable
	networkName := *networkFlag
	if networkName == "" {
		networkName = os.Getenv("BS_NETWORK")
	}

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}

	// Add network if specified
	if networkName != "" {
		opts = append(opts, bscontrol.WithNetwork(networkName))
	}

	// Enable debug logging if requested
	if *debugFlag {
		opts = append(opts, bscontrol.WithDebug(true))
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Creating BSN.cloud client...\n")
	}

	client, err := bscontrol.New(opts...)
	if err != nil {
		if bscontrol.IsConfigurationError(err) {
			log.Fatalf("Configuration error: %v", err)
		}
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Authenticate
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Authenticating with BSN.cloud...\n")
	}

	if err := client.Authenticate(ctx); err != nil {
		if bscontrol.IsAuthenticationError(err) {
			log.Fatalf("Authentication failed: %v", err)
		}
		log.Fatalf("Authentication error: %v", err)
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Authentication successful!\n")
	}

	// Handle network selection
	if err := handleNetworkSelection(ctx, client, networkName, *verboseFlag, *jsonFlag); err != nil {
		log.Fatalf("Network selection failed: %v", err)
	}

	// Get device serial number
	var serial string
	if *serialFlag != "" {
		serial = *serialFlag
	} else {
		// Get device by ID to retrieve serial number
		device, err := client.Devices.GetByID(ctx, *idFlag)
		if err != nil {
			log.Fatalf("Failed to get device with ID %d: %v", *idFlag, err)
		}
		serial = device.Serial
	}

	// Get device info for confirmation
	var deviceInfo string
	if *serialFlag != "" {
		deviceInfo = fmt.Sprintf("serial number '%s'", serial)
	} else {
		deviceInfo = fmt.Sprintf("ID %d (serial: %s)", *idFlag, serial)
	}

	// Confirmation prompt (unless -y flag is used)
	if !*confirmFlag {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Println(strings.Repeat("=", 70))
		fmt.Printf("  WARNING: DATA DESTRUCTION OPERATION\n")
		fmt.Println(strings.Repeat("=", 70))
		fmt.Printf("\nThis operation will REFORMAT the %s storage device on the player with %s.\n\n", strings.ToUpper(deviceName), deviceInfo)
		fmt.Printf("ALL DATA on the %s storage will be PERMANENTLY ERASED!\n\n", strings.ToUpper(deviceName))
		fmt.Printf("This action CANNOT be undone.\n\n")
		fmt.Print("Type 'REFORMAT' to confirm (case-sensitive): ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			log.Fatalf("Failed to read confirmation")
		}

		confirmation := scanner.Text()
		if confirmation != "REFORMAT" {
			fmt.Println("\nOperation cancelled.")
			os.Exit(0)
		}
	}

	// Perform the reformat
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\nReformatting %s storage on device %s...\n", strings.ToUpper(deviceName), serial)
	}

	success, err := client.RDWS.ReformatStorage(ctx, serial, deviceName)
	if err != nil {
		log.Fatalf("Failed to reformat storage: %v", err)
	}

	if !success {
		if *debugFlag {
			log.Fatalf("Storage reformat operation returned unsuccessful status - check debug output above for details")
		}
		log.Fatalf("Storage reformat operation returned unsuccessful status - add --debug to see detailed HTTP request/response logs")
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success": true,
			"serial":  serial,
			"device":  deviceName,
			"message": fmt.Sprintf("Storage device %s successfully reformatted", strings.ToUpper(deviceName)),
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "\nStorage device %s successfully reformatted!\n", strings.ToUpper(deviceName))
	fmt.Fprintf(os.Stderr, "Note: The device may need to reboot to complete the reformatting process.\n")
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, verbose bool, jsonMode bool) error {
	// Check if network is already set
	if client.IsNetworkSet() {
		if current, err := client.GetCurrentNetwork(ctx); err == nil {
			if !jsonMode {
				fmt.Fprintf(os.Stderr, "Using network: %s (ID: %d)\n", current.Name, current.ID)
			}
			return nil
		}
	}

	// Get available networks
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "Getting available networks...\n")
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
					fmt.Fprintf(os.Stderr, "Using requested network: %s (ID: %d)\n", network.Name, network.ID)
				}
				return client.SetNetworkByID(ctx, network.ID)
			}
		}

		// Network not found - show error and fall back to interactive selection
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "Network '%s' not found. Available networks:\n", requestedNetwork)
			for i, network := range networks {
				fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
	}

	// If only one network and no specific network requested, use it automatically
	if len(networks) == 1 && requestedNetwork == "" {
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "Using network: %s (ID: %d)\n", networks[0].Name, networks[0].ID)
		}
		return client.SetNetworkByID(ctx, networks[0].ID)
	}

	// In JSON mode, cannot do interactive selection
	if jsonMode {
		return fmt.Errorf("network selection required: use --network flag or BS_NETWORK environment variable")
	}

	// Show available networks and let user choose
	if requestedNetwork == "" {
		fmt.Fprintf(os.Stderr, "Available networks:\n")
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
	fmt.Fprintf(os.Stderr, "Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)

	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}
