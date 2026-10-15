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
		serialFlag  = flag.String("serial", "", "Device serial number")
		idFlag      = flag.Int("id", 0, "Device ID")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
		saveFlag    = flag.String("save", "", "Save crash dumps to directory")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for retrieving BrightSign player crash dumps.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get crash dumps:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get crash dumps as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Save crash dumps to directory:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --save ./dumps\n", os.Args[0])
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

	// Get crash dumps
	if err := getCrashDumps(ctx, client, serial, *jsonFlag, *verboseFlag, *saveFlag); err != nil {
		log.Fatalf("Failed to get crash dumps: %v", err)
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

func getCrashDumps(ctx context.Context, client *bscontrol.Client, serial string, jsonMode bool, verbose bool, saveDir string) error {
	if !jsonMode {
		fmt.Printf("\nGetting crash dumps from device %s...\n", serial)
	}

	crashDump, err := client.RDWS.GetCrashDump(ctx, serial)
	if err != nil {
		return err
	}

	if jsonMode {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(crashDump)
	}

	fmt.Println("\nPlayer Crash Dumps:")
	fmt.Println(strings.Repeat("=", 70))

	if len(crashDump.Files) == 0 {
		fmt.Println("No crash dump files found on device.")
		return nil
	}

	// Display crash dumps
	for i, dumpFile := range crashDump.Files {
		fmt.Printf("\n[%d] %s\n", i+1, dumpFile.Name)
		fmt.Printf("    Timestamp: %s\n", dumpFile.Timestamp)
		fmt.Printf("    Size: %d bytes\n", dumpFile.Size)

		if verbose && dumpFile.Content != "" {
			fmt.Println("    Content preview:")
			lines := strings.Split(dumpFile.Content, "\n")
			displayLines := 10
			if len(lines) < displayLines {
				displayLines = len(lines)
			}
			for j := 0; j < displayLines; j++ {
				fmt.Printf("      %s\n", lines[j])
			}
			if len(lines) > displayLines {
				fmt.Printf("      ... (%d more lines)\n", len(lines)-displayLines)
			}
		}
	}

	fmt.Printf("\nTotal crash dump files: %d\n", len(crashDump.Files))

	// Save crash dumps to directory if requested
	if saveDir != "" {
		if err := saveCrashDumps(crashDump, saveDir, serial); err != nil {
			return fmt.Errorf("failed to save crash dumps: %w", err)
		}
		fmt.Printf("\nCrash dumps saved to: %s\n", saveDir)
	}

	return nil
}

func saveCrashDumps(crashDump *bscontrol.RDWSCrashDump, saveDir string, serial string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create a subdirectory for this device
	deviceDir := fmt.Sprintf("%s/%s_%s", saveDir, serial, time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(deviceDir, 0755); err != nil {
		return fmt.Errorf("failed to create device directory: %w", err)
	}

	// Save each crash dump file
	for _, dumpFile := range crashDump.Files {
		filePath := fmt.Sprintf("%s/%s", deviceDir, dumpFile.Name)
		if err := os.WriteFile(filePath, []byte(dumpFile.Content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", dumpFile.Name, err)
		}
	}

	return nil
}
