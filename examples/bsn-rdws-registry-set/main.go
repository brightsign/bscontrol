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
		sectionFlag  = flag.String("section", "", "Registry section name")
		keyFlag      = flag.String("key", "", "Registry key name")
		valueFlag    = flag.String("value", "", "Registry value to set")
		deleteFlag   = flag.Bool("delete", false, "Delete registry key")
		flushFlag    = flag.Bool("flush", false, "Flush registry to disk")
		recoveryFlag = flag.String("recovery-url", "", "Set recovery URL")
		confirmFlag  *bool
		jsonFlag     = flag.Bool("json", false, "Output as JSON")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Set up confirm flags to point to the same variable
	confirmFlag = flag.Bool("y", false, "Skip confirmation prompt")
	flag.BoolVar(confirmFlag, "force", false, "Skip confirmation prompt [alias for -y]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for writing BrightSign player registry values.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Operations:\n")
		fmt.Fprintf(os.Stderr, "  --section, --key, --value  Set registry value\n")
		fmt.Fprintf(os.Stderr, "  --section, --key, --delete Delete registry key\n")
		fmt.Fprintf(os.Stderr, "  --flush                    Flush registry to disk\n")
		fmt.Fprintf(os.Stderr, "  --recovery-url URL         Set recovery URL\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Set registry value:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --section networking --key hostname --value player1\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Delete registry key without confirmation:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --section networking --key hostname --delete --force\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Flush registry:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --flush\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Set recovery URL:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --recovery-url https://example.com/recovery\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Set registry value with JSON output:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --section networking --key hostname --value player1 -y --json\n", os.Args[0])
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

	// Validate operation flags
	operationCount := 0
	if *valueFlag != "" || *deleteFlag {
		operationCount++
	}
	if *flushFlag {
		operationCount++
	}
	if *recoveryFlag != "" {
		operationCount++
	}

	if operationCount == 0 {
		fmt.Fprintf(os.Stderr, "Error: Must specify an operation: --value, --delete, --flush, or --recovery-url\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if operationCount > 1 {
		fmt.Fprintf(os.Stderr, "Error: Can only specify one operation at a time\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate set/delete operations
	if (*valueFlag != "" || *deleteFlag) && (*sectionFlag == "" || *keyFlag == "") {
		fmt.Fprintf(os.Stderr, "Error: --section and --key must be specified for set/delete operations\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *valueFlag != "" && *deleteFlag {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify both --value and --delete\n\n")
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

	// Confirmation prompt for write operations (unless -y flag is used)
	if !*confirmFlag {
		var operationDesc string
		if *valueFlag != "" {
			operationDesc = fmt.Sprintf("set registry value %s/%s = '%s'", *sectionFlag, *keyFlag, *valueFlag)
		} else if *deleteFlag {
			operationDesc = fmt.Sprintf("delete registry key %s/%s", *sectionFlag, *keyFlag)
		} else if *flushFlag {
			operationDesc = "flush registry to disk"
		} else if *recoveryFlag != "" {
			operationDesc = fmt.Sprintf("set recovery URL to '%s'", *recoveryFlag)
		}

		fmt.Printf("\nThis will %s on device with %s.\n", operationDesc, deviceName)
		fmt.Print("Proceed? (yes/no): ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			log.Fatalf("Failed to read confirmation")
		}

		confirmation := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if confirmation != "yes" && confirmation != "y" {
			fmt.Println("\nOperation cancelled.")
			os.Exit(0)
		}
	}

	// Perform requested operation
	if *valueFlag != "" {
		if err := setRegistryValue(ctx, client, serial, *sectionFlag, *keyFlag, *valueFlag, *jsonFlag); err != nil {
			log.Fatalf("Failed to set registry value: %v", err)
		}
	} else if *deleteFlag {
		if err := deleteRegistryValue(ctx, client, serial, *sectionFlag, *keyFlag, *jsonFlag); err != nil {
			log.Fatalf("Failed to delete registry value: %v", err)
		}
	} else if *flushFlag {
		if err := flushRegistry(ctx, client, serial, *jsonFlag); err != nil {
			log.Fatalf("Failed to flush registry: %v", err)
		}
	} else if *recoveryFlag != "" {
		if err := setRecoveryURL(ctx, client, serial, *recoveryFlag, *jsonFlag); err != nil {
			log.Fatalf("Failed to set recovery URL: %v", err)
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

func setRegistryValue(ctx context.Context, client *bscontrol.Client, serial string, section string, key string, value string, jsonMode bool) error {
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "\nSetting registry value %s/%s on device %s...\n", section, key, serial)
	}

	success, err := client.RDWS.SetRegistryValue(ctx, serial, section, key, value)
	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("set registry value operation returned unsuccessful status")
	}

	if jsonMode {
		result := map[string]interface{}{
			"success": true,
			"serial":  serial,
			"section": section,
			"key":     key,
			"value":   value,
			"message": "Registry value successfully set",
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
		return nil
	}

	fmt.Fprintf(os.Stderr, "\nRegistry value %s/%s successfully set to '%s'\n", section, key, value)
	fmt.Fprintf(os.Stderr, "Note: Changes may require registry flush or device reboot to persist.\n")

	return nil
}

func deleteRegistryValue(ctx context.Context, client *bscontrol.Client, serial string, section string, key string, jsonMode bool) error {
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "\nDeleting registry value %s/%s from device %s...\n", section, key, serial)
	}

	success, err := client.RDWS.DeleteRegistryValue(ctx, serial, section, key)
	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("delete registry value operation returned unsuccessful status")
	}

	if jsonMode {
		result := map[string]interface{}{
			"success": true,
			"serial":  serial,
			"section": section,
			"key":     key,
			"message": "Registry value successfully deleted",
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
		return nil
	}

	fmt.Fprintf(os.Stderr, "\nRegistry value %s/%s successfully deleted\n", section, key)
	fmt.Fprintf(os.Stderr, "Note: Changes may require registry flush or device reboot to persist.\n")

	return nil
}

func flushRegistry(ctx context.Context, client *bscontrol.Client, serial string, jsonMode bool) error {
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "\nFlushing registry on device %s...\n", serial)
	}

	success, err := client.RDWS.FlushRegistry(ctx, serial)
	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("flush registry operation returned unsuccessful status")
	}

	if jsonMode {
		result := map[string]interface{}{
			"success": true,
			"serial":  serial,
			"message": "Registry successfully flushed to disk",
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
		return nil
	}

	fmt.Fprintf(os.Stderr, "\nRegistry successfully flushed to disk on device %s\n", serial)
	fmt.Fprintf(os.Stderr, "All pending registry changes have been written to persistent storage.\n")

	return nil
}

func setRecoveryURL(ctx context.Context, client *bscontrol.Client, serial string, recoveryURL string, jsonMode bool) error {
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "\nSetting recovery URL on device %s...\n", serial)
	}

	success, err := client.RDWS.SetRecoveryURL(ctx, serial, recoveryURL)
	if err != nil {
		return err
	}

	if !success {
		return fmt.Errorf("set recovery URL operation returned unsuccessful status")
	}

	if jsonMode {
		result := map[string]interface{}{
			"success":      true,
			"serial":       serial,
			"recovery_url": recoveryURL,
			"message":      "Recovery URL successfully set",
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
		return nil
	}

	fmt.Fprintf(os.Stderr, "\nRecovery URL successfully set to '%s'\n", recoveryURL)
	fmt.Fprintf(os.Stderr, "The player will use this URL if it needs to recover from a failure.\n")

	return nil
}
