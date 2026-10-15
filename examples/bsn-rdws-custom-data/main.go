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
		dataFlag    = flag.String("data", "", "Custom data string to send")
		fileFlag    = flag.String("file", "", "Read custom data from file")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for sending custom data to BrightSign players via UDP port 5000.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Data Input:\n")
		fmt.Fprintf(os.Stderr, "  --data STRING       Send specified string directly\n")
		fmt.Fprintf(os.Stderr, "  --file PATH         Read data from specified file\n")
		fmt.Fprintf(os.Stderr, "  (stdin)             Read data from stdin if neither --data nor --file is provided\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Send string directly:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --data \"start:playlist1\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Send data from file:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --file command.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Send data from stdin:\n")
		fmt.Fprintf(os.Stderr, "    echo \"custom command\" | %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  JSON command:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --data '{\"action\":\"play\",\"file\":\"video.mp4\"}'\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --data \"test\" --json\n", os.Args[0])
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

	// Get custom data from various sources
	var customData string
	var err error

	if *dataFlag != "" && *fileFlag != "" {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify both --data and --file\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *dataFlag != "" {
		// Data provided directly via flag
		customData = *dataFlag
	} else if *fileFlag != "" {
		// Read data from file
		fileContent, err := os.ReadFile(*fileFlag)
		if err != nil {
			log.Fatalf("Failed to read file '%s': %v", *fileFlag, err)
		}
		customData = string(fileContent)
	} else {
		// Read from stdin
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "Reading custom data from stdin (Ctrl+D or Ctrl+Z to end):\n")
		}
		scanner := bufio.NewScanner(os.Stdin)
		var lines []string
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			log.Fatalf("Error reading from stdin: %v", err)
		}
		customData = strings.Join(lines, "\n")
	}

	// Validate that we have data
	if strings.TrimSpace(customData) == "" {
		fmt.Fprintf(os.Stderr, "Error: Custom data cannot be empty\n\n")
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

	// Display what we're sending
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\nSending custom data to device %s...\n", serial)
		if *verboseFlag {
			fmt.Fprintf(os.Stderr, "\nData to send:\n")
			fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("-", 50))
			fmt.Fprintf(os.Stderr, "%s\n", customData)
			fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("-", 50))
			fmt.Fprintf(os.Stderr, "Data length: %d bytes\n\n", len(customData))
		}
	}

	// Send the custom data
	success, err := client.RDWS.SendCustomData(ctx, serial, customData)
	if err != nil {
		log.Fatalf("Failed to send custom data: %v", err)
	}

	if !success {
		log.Fatalf("Custom data send operation returned unsuccessful status")
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success":    true,
			"serial":     serial,
			"dataLength": len(customData),
			"data":       customData,
			"note":       "Player application must be listening on UDP port 5000",
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "\nCustom data successfully sent to device %s!\n", serial)
	fmt.Fprintf(os.Stderr, "\nNote: The player application must be listening on UDP port 5000 to receive this data.\n")
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
	if verbose {
		fmt.Fprintf(os.Stderr, "Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)
	}

	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}
