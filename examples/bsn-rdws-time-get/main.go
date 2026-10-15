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
		jsonFlag     = flag.Bool("json", false, "Output raw JSON response")
		timeoutFlag  = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag  *string
		serialFlag   = flag.String("serial", "", "Device serial number (required)")
		actionFlag   = flag.String("action", "get", "Action: get or set")
		timeFlag     = flag.String("time", "", "Time to set (format: HH:MM:SS)")
		dateFlag     = flag.String("date", "", "Date to set (format: YYYY-MM-DD)")
		timezoneFlag = flag.Bool("apply-timezone", true, "Apply player's timezone (true) or UTC (false)")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for managing player time via rDWS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get player time:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Set player time:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --action set --time 14:30:00 --date 2025-11-07\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Set time with UTC (no timezone):\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --action set --time 14:30:00 --date 2025-11-07 --apply-timezone=false\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Validate input
	if *serialFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: Must specify --serial\n\n")
		flag.Usage()
		os.Exit(1)
	}

	action := strings.ToLower(*actionFlag)
	if action != "get" && action != "set" {
		fmt.Fprintf(os.Stderr, "Error: Invalid action '%s'. Valid actions: get, set\n\n", action)
		flag.Usage()
		os.Exit(1)
	}

	if action == "set" {
		if *timeFlag == "" || *dateFlag == "" {
			fmt.Fprintf(os.Stderr, "Error: Must specify both --time and --date when setting time\n\n")
			flag.Usage()
			os.Exit(1)
		}
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

	// Handle network selection
	if err := handleNetworkSelection(ctx, client, *networkFlag, *verboseFlag, *jsonFlag); err != nil {
		log.Fatalf("Network selection failed: %v", err)
	}

	// Execute action
	switch action {
	case "get":
		handleGetTime(ctx, client, *serialFlag, *jsonFlag, *verboseFlag)
	case "set":
		handleSetTime(ctx, client, *serialFlag, *timeFlag, *dateFlag, *timezoneFlag, *jsonFlag)
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
		fmt.Fprintf(os.Stderr, "Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)
	}

	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}

func handleGetTime(ctx context.Context, client *bscontrol.Client, serial string, jsonMode, verbose bool) {
	if !jsonMode {
		fmt.Printf("Getting player time for device %s...\n", serial)
	}

	timeInfo, err := client.RDWS.GetTime(ctx, serial)
	if err != nil {
		log.Fatalf("Failed to get player time: %v", err)
	}

	// Display results
	if jsonMode {
		jsonData, err := json.MarshalIndent(timeInfo, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		displayTimeInfo(timeInfo, verbose)
	}
}

func handleSetTime(ctx context.Context, client *bscontrol.Client, serial, timeStr, dateStr string, applyTimezone, jsonMode bool) {
	if !jsonMode {
		fmt.Printf("Setting player time for device %s...\n", serial)
		fmt.Printf("Time: %s\n", timeStr)
		fmt.Printf("Date: %s\n", dateStr)
		fmt.Printf("Apply Timezone: %v\n", applyTimezone)
	}

	request := &bscontrol.RDWSTimeSetRequest{
		Time:          timeStr,
		Date:          dateStr,
		ApplyTimezone: applyTimezone,
	}

	success, err := client.RDWS.SetTime(ctx, serial, request)
	if err != nil {
		log.Fatalf("Failed to set player time: %v", err)
	}

	if jsonMode {
		result := map[string]interface{}{
			"success": success,
			"time":    timeStr,
			"date":    dateStr,
		}
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		if success {
			fmt.Println("\nTime successfully set on player!")
		} else {
			fmt.Println("\nFailed to set time on player")
		}
	}
}

func displayTimeInfo(info *bscontrol.RDWSTimeInfo, verbose bool) {
	fmt.Fprintf(os.Stderr, "\n=== Player Time Information ===\n")
	fmt.Fprintf(os.Stderr, "Current Time:    %s\n", info.Time)
	fmt.Fprintf(os.Stderr, "Timezone:        %s (%s)\n", info.TimezoneName, info.TimezoneAbbr)

	if verbose {
		fmt.Fprintf(os.Stderr, "\nDetailed Time:\n")
		fmt.Fprintf(os.Stderr, "  Year:         %d\n", info.Year)
		fmt.Fprintf(os.Stderr, "  Month:        %d\n", info.Month)
		fmt.Fprintf(os.Stderr, "  Date:         %d\n", info.Date)
		fmt.Fprintf(os.Stderr, "  Hour:         %d\n", info.Hour)
		fmt.Fprintf(os.Stderr, "  Minute:       %d\n", info.Minute)
		fmt.Fprintf(os.Stderr, "  Second:       %d\n", info.Second)
		fmt.Fprintf(os.Stderr, "  Millisecond:  %d\n", info.Millisecond)

		if info.TimezoneMin != nil {
			fmt.Fprintf(os.Stderr, "  TZ Offset:    %d minutes\n", *info.TimezoneMin)
		}
	}

	fmt.Fprintf(os.Stderr, "\n")
}
