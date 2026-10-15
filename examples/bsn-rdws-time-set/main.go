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
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag *string
		serialFlag  = flag.String("serial", "", "Device serial number (required)")
		timeFlag    = flag.String("time", "", "Time to set (HH:MM:SS format, leave empty for current time)")
		dateFlag    = flag.String("date", "", "Date to set (YYYY-MM-DD format, leave empty for current date)")
		tzFlag      = flag.Bool("apply-timezone", true, "Apply player's timezone (true) or use UTC (false)")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
	)

	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for setting player time via rDWS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Set player time to current time:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Set specific time and date:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --time 14:30:00 --date 2025-11-11\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Set time with JSON output:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	if *serialFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: Must specify --serial\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}
	if *networkFlag != "" {
		opts = append(opts, bscontrol.WithNetwork(*networkFlag))
	}

	client, err := bscontrol.New(opts...)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Authenticating with BSN.cloud...\n")
	}
	if err := client.Authenticate(ctx); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	if err := handleNetworkSelection(ctx, client, *networkFlag, *jsonFlag); err != nil {
		log.Fatalf("Network selection failed: %v", err)
	}

	// Prepare time request
	now := time.Now()
	timeStr := *timeFlag
	dateStr := *dateFlag

	if timeStr == "" {
		timeStr = now.Format("15:04:05")
	}
	if dateStr == "" {
		dateStr = now.Format("2006-01-02")
	}

	request := &bscontrol.RDWSTimeSetRequest{
		Time:          timeStr,
		Date:          dateStr,
		ApplyTimezone: *tzFlag,
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Setting player time to %s %s...\n", dateStr, timeStr)
	}
	success, err := client.RDWS.SetTime(ctx, *serialFlag, request)
	if err != nil {
		log.Fatalf("Failed to set time: %v", err)
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success":        success,
			"serial":         *serialFlag,
			"time":           timeStr,
			"date":           dateStr,
			"apply_timezone": *tzFlag,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	if success {
		fmt.Fprintf(os.Stderr, "✓ Time set successfully\n")
	} else {
		fmt.Fprintf(os.Stderr, "✗ Time set failed\n")
	}
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, jsonMode bool) error {
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

	networks, err := client.GetNetworks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get networks: %w", err)
	}

	if len(networks) == 0 {
		return fmt.Errorf("no networks available")
	}

	if requestedNetwork != "" {
		for _, network := range networks {
			if strings.EqualFold(network.Name, requestedNetwork) {
				if !jsonMode {
					fmt.Fprintf(os.Stderr, "Using network: %s (ID: %d)\n", network.Name, network.ID)
				}
				return client.SetNetworkByID(ctx, network.ID)
			}
		}
	}

	if len(networks) == 1 {
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "Using network: %s (ID: %d)\n", networks[0].Name, networks[0].ID)
		}
		return client.SetNetworkByID(ctx, networks[0].ID)
	}

	// In JSON mode, cannot do interactive selection
	if jsonMode {
		return fmt.Errorf("network selection required: use --network flag or BS_NETWORK environment variable")
	}

	fmt.Fprintf(os.Stderr, "Available networks:\n")
	for i, network := range networks {
		fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
	}

	fmt.Fprint(os.Stderr, "Select network (1-"+strconv.Itoa(len(networks))+"): ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	selection, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || selection < 1 || selection > len(networks) {
		return fmt.Errorf("invalid selection")
	}

	selectedNetwork := networks[selection-1]
	fmt.Fprintf(os.Stderr, "Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)
	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}
