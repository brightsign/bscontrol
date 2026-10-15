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
		helpFlag     = flag.Bool("help", false, "Display usage information")
		jsonFlag     = flag.Bool("json", false, "Output as JSON")
		verboseFlag  = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag  = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag  *string
		serialFlag   = flag.String("serial", "", "Device serial number")
		idFlag       = flag.Int("id", 0, "Device ID")
		firmwareFlag = flag.String("firmware-url", "", "File URL to download")
		confirmFlag  = flag.Bool("y", false, "Skip confirmation prompt")
		noRebootFlag = flag.Bool("no-reboot", false, "Do not automatically reboot after download")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for remotely downloading files to BrightSign players.\n")
		fmt.Fprintf(os.Stderr, "Commonly used for firmware updates, but can download any file type.\n\n")
		fmt.Fprintf(os.Stderr, "⚠️  WARNING: By default, the player will reboot automatically after download completes!\n")
		fmt.Fprintf(os.Stderr, "           Use --no-reboot to prevent automatic reboot.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "File URL Format:\n")
		fmt.Fprintf(os.Stderr, "  The file URL must be publicly accessible (http:// or https://)\n")
		fmt.Fprintf(os.Stderr, "  Examples:\n")
		fmt.Fprintf(os.Stderr, "    - Firmware: https://example.com/firmware/brightsign-8.5.25.bsfw\n")
		fmt.Fprintf(os.Stderr, "    - Config:   https://example.com/config/autorun.brs\n")
		fmt.Fprintf(os.Stderr, "    - Content:  https://example.com/media/video.mp4\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Download firmware with auto-reboot:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --firmware-url https://example.com/firmware.bsfw\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Download file without auto-reboot:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --firmware-url https://example.com/config.brs --no-reboot\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Download with confirmation skip:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --firmware-url https://example.com/firmware.bsfw -y\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Download with specific network:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 -n \"Production\" --firmware-url https://example.com/media/video.mp4 --no-reboot\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --firmware-url https://example.com/firmware.bsfw -y --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
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

	if *firmwareFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: Must specify --firmware-url\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate firmware URL format
	if !strings.HasPrefix(*firmwareFlag, "http://") && !strings.HasPrefix(*firmwareFlag, "https://") {
		fmt.Fprintf(os.Stderr, "Error: Firmware URL must start with http:// or https://\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *jsonFlag && !*confirmFlag {
		fmt.Fprintf(os.Stderr, "Error: -y flag is required when using --json (cannot prompt for confirmation)\n\n")
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
	if err := handleNetworkSelection(ctx, client, *networkFlag, *verboseFlag, *jsonFlag); err != nil {
		log.Fatalf("Network selection failed: %v", err)
	}

	// Get device serial number
	var serial string
	var deviceName string
	if *serialFlag != "" {
		serial = *serialFlag
		deviceName = fmt.Sprintf("serial '%s'", serial)
	} else {
		// Get device by ID to retrieve serial number
		device, err := client.Devices.GetByID(ctx, *idFlag)
		if err != nil {
			log.Fatalf("Failed to get device with ID %d: %v", *idFlag, err)
		}
		serial = device.Serial
		deviceName = fmt.Sprintf("ID %d (serial: %s)", *idFlag, serial)
	}

	// Confirmation prompt (unless -y flag is used)
	if !*confirmFlag {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Println(strings.Repeat("=", 70))
		fmt.Printf("  WARNING: FILE DOWNLOAD OPERATION\n")
		fmt.Println(strings.Repeat("=", 70))
		fmt.Printf("\nThis will download a file to the player with %s.\n\n", deviceName)
		fmt.Printf("File URL: %s\n", *firmwareFlag)

		if *noRebootFlag {
			fmt.Printf("Auto-reboot: DISABLED (player will NOT reboot automatically)\n\n")
			fmt.Println("ℹ️  The player will download the file but will NOT automatically reboot.")
			fmt.Println("ℹ️  You may need to manually reboot the player after the download.")
		} else {
			fmt.Printf("Auto-reboot: ENABLED (player WILL reboot automatically)\n\n")
			fmt.Println("⚠️  The player will AUTOMATICALLY REBOOT after the file is downloaded!")
			fmt.Println("⚠️  Do NOT power off the player during the download and reboot process!")
			fmt.Println("⚠️  The player will be unavailable during the reboot.")
		}
		fmt.Fprintf(os.Stderr, "\n")

		if *verboseFlag {
			fmt.Println("Process:")
			fmt.Println("  1. Player downloads file from the specified URL")
			if !*noRebootFlag {
				fmt.Println("  2. Player validates the downloaded file")
				fmt.Println("  3. Player applies the update")
				fmt.Println("  4. Player reboots automatically")
				fmt.Println("  5. Player comes back online")
			} else {
				fmt.Println("  2. Player saves the file")
				fmt.Println("  3. Player continues normal operation (no automatic reboot)")
			}
			fmt.Fprintf(os.Stderr, "\n")
		}

		fmt.Print("Type 'DOWNLOAD' to confirm (case-sensitive): ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			log.Fatalf("Failed to read confirmation")
		}

		confirmation := scanner.Text()
		if confirmation != "DOWNLOAD" {
			fmt.Println("\nOperation cancelled.")
			os.Exit(0)
		}
	}

	// Initiate file download
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\nInitiating file download on device %s...\n", serial)
		if *verboseFlag {
			fmt.Fprintf(os.Stderr, "File URL: %s\n", *firmwareFlag)
			if *noRebootFlag {
				fmt.Fprintf(os.Stderr, "Auto-reboot: false\n")
			} else {
				fmt.Fprintf(os.Stderr, "Auto-reboot: true (default)\n")
			}
		}
	}

	// Prepare autoReboot parameter
	var autoReboot *bool
	if *noRebootFlag {
		reboot := false
		autoReboot = &reboot
	}
	// If noRebootFlag is false, autoReboot stays nil (uses server default, which is true)

	success, err := client.RDWS.DownloadFirmware(ctx, serial, *firmwareFlag, autoReboot)
	if err != nil {
		log.Fatalf("Failed to initiate file download: %v", err)
	}

	if !success {
		log.Fatalf("File download operation returned unsuccessful status")
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success":    true,
			"serial":     serial,
			"fileURL":    *firmwareFlag,
			"autoReboot": !*noRebootFlag,
			"note":       "Download initiated successfully. Player is now downloading the file.",
		}
		if *noRebootFlag {
			result["warning"] = "Auto-reboot is disabled. You may need to manually reboot the player."
		} else {
			result["warning"] = "Player will automatically reboot when download completes. Do NOT power off during this process."
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "\nFile download successfully initiated on device %s!\n", serial)
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "Next steps:\n")
	fmt.Fprintf(os.Stderr, "  1. The player is now downloading the file\n")

	if *noRebootFlag {
		fmt.Fprintf(os.Stderr, "  2. The player will save the file (no automatic reboot)\n")
		fmt.Fprintf(os.Stderr, "  3. You may need to manually reboot the player if required\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Note: The download process may take several minutes.\n")
	} else {
		fmt.Fprintf(os.Stderr, "  2. The player will automatically reboot when download completes\n")
		fmt.Fprintf(os.Stderr, "  3. Do NOT power off the player during this process\n")
		fmt.Fprintf(os.Stderr, "  4. Wait for the player to come back online\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Note: The update process may take several minutes.\n")
		fmt.Fprintf(os.Stderr, "      You can check device status after it comes back online.\n")
	}
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, verbose, jsonMode bool) error {
	// Check if network is already set
	if client.IsNetworkSet() {
		if current, err := client.GetCurrentNetwork(ctx); err == nil {
			if !jsonMode {
				fmt.Fprintf(os.Stderr, "Using network: %s (ID: %d)\n", current.Name, current.ID)
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
		} else {
			return fmt.Errorf("network '%s' not found", requestedNetwork)
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
