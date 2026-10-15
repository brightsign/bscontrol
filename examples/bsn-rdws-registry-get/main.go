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
		verboseFlag  = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag  = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag  *string
		serialFlag   = flag.String("serial", "", "Device serial number")
		idFlag       = flag.Int("id", 0, "Device ID")
		fullFlag     = flag.Bool("full", false, "Get full registry dump")
		sectionFlag  = flag.String("section", "", "Registry section name")
		keyFlag      = flag.String("key", "", "Registry key name")
		recoveryFlag = flag.Bool("recovery-url", false, "Get recovery URL from registry")
		jsonFlag     = flag.Bool("json", false, "Output as JSON")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for reading BrightSign player registry values.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Operations:\n")
		fmt.Fprintf(os.Stderr, "  --full              Get complete registry dump\n")
		fmt.Fprintf(os.Stderr, "  --section & --key   Get specific registry value\n")
		fmt.Fprintf(os.Stderr, "  --recovery-url      Get recovery URL from registry\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get full registry:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --full\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get specific value:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --section networking --key hostname\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get recovery URL:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --recovery-url\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get full registry as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --full --json\n", os.Args[0])
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

	// Validate operation flags
	operationCount := 0
	if *fullFlag {
		operationCount++
	}
	if *sectionFlag != "" || *keyFlag != "" {
		operationCount++
	}
	if *recoveryFlag {
		operationCount++
	}

	if operationCount == 0 {
		fmt.Fprintf(os.Stderr, "Error: Must specify an operation: --full, --section/--key, or --recovery-url\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if operationCount > 1 {
		fmt.Fprintf(os.Stderr, "Error: Can only specify one operation at a time\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate section/key combination
	if (*sectionFlag != "" && *keyFlag == "") || (*sectionFlag == "" && *keyFlag != "") {
		fmt.Fprintf(os.Stderr, "Error: Both --section and --key must be specified together\n\n")
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
		fmt.Println("Creating BSN.cloud client...")
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
		fmt.Println("Authentication successful!")
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

	// Perform requested operation
	if *fullFlag {
		if err := getFullRegistry(ctx, client, serial, *jsonFlag, *verboseFlag); err != nil {
			log.Fatalf("Failed to get registry: %v", err)
		}
	} else if *sectionFlag != "" {
		if err := getRegistryValue(ctx, client, serial, *sectionFlag, *keyFlag, *jsonFlag); err != nil {
			log.Fatalf("Failed to get registry value: %v", err)
		}
	} else if *recoveryFlag {
		if err := getRecoveryURL(ctx, client, serial, *jsonFlag); err != nil {
			log.Fatalf("Failed to get recovery URL: %v", err)
		}
	}
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
		fmt.Printf("Network '%s' not found. Available networks:\n", requestedNetwork)
		for i, network := range networks {
			fmt.Printf("  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// If only one network and no specific network requested, use it automatically
	if len(networks) == 1 && requestedNetwork == "" {
		if !jsonMode {
			fmt.Printf("Using network: %s (ID: %d)\n", networks[0].Name, networks[0].ID)
		}
		return client.SetNetworkByID(ctx, networks[0].ID)
	}

	// Show available networks and let user choose
	if requestedNetwork == "" {
		fmt.Fprintf(os.Stderr, "Available networks:\n")
		for i, network := range networks {
			fmt.Printf("  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
			if verbose {
				fmt.Printf("     Created: %s, Modified: %s\n",
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
	if !jsonMode {
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)
		}
	}

	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}

func getFullRegistry(ctx context.Context, client *bscontrol.Client, serial string, jsonMode bool, verbose bool) error {
	if !jsonMode {
		fmt.Printf("\nGetting full registry from device %s...\n", serial)
	}

	registry, err := client.RDWS.GetRegistry(ctx, serial)
	if err != nil {
		return err
	}

	if jsonMode {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(registry)
	}

	fmt.Println("\nPlayer Registry:")
	fmt.Println(strings.Repeat("=", 70))

	for section, keys := range registry.Sections {
		fmt.Printf("\n[%s]\n", section)
		for key, value := range keys {
			if verbose {
				fmt.Printf("  %s = %s\n", key, value)
			} else {
				// Truncate long values
				displayValue := value
				if len(value) > 60 {
					displayValue = value[:57] + "..."
				}
				fmt.Printf("  %s = %s\n", key, displayValue)
			}
		}
	}

	fmt.Printf("\nTotal sections: %d\n", len(registry.Sections))

	return nil
}

func getRegistryValue(ctx context.Context, client *bscontrol.Client, serial string, section string, key string, jsonMode bool) error {
	if !jsonMode {
		fmt.Printf("\nGetting registry value %s/%s from device %s...\n", section, key, serial)
	}

	regValue, err := client.RDWS.GetRegistryValue(ctx, serial, section, key)
	if err != nil {
		return err
	}

	if jsonMode {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(regValue)
	}

	fmt.Println("\nRegistry Value:")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Section: %s\n", regValue.Section)
	fmt.Printf("Key:     %s\n", regValue.Key)
	fmt.Printf("Value:   %s\n", regValue.Value)

	return nil
}

func getRecoveryURL(ctx context.Context, client *bscontrol.Client, serial string, jsonMode bool) error {
	if !jsonMode {
		fmt.Printf("\nGetting recovery URL from device %s...\n", serial)
	}

	recoveryURL, err := client.RDWS.GetRecoveryURL(ctx, serial)
	if err != nil {
		return err
	}

	if jsonMode {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(recoveryURL)
	}

	fmt.Println("\nRecovery URL:")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("URL: %s\n", recoveryURL.URL)

	return nil
}
