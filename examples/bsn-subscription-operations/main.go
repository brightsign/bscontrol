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
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for listing subscription operations and permissions on a BSN.cloud network.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get subscription operations:\n")
		fmt.Fprintf(os.Stderr, "    %s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Show detailed information:\n")
		fmt.Fprintf(os.Stderr, "    %s --verbose\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
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
		fmt.Println("Authenticating with BSN.cloud...")
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
	if err := handleNetworkSelection(ctx, client, *networkFlag, *jsonFlag); err != nil {
		log.Fatalf("Network selection failed: %v", err)
	}

	// Get subscription operations
	if err := getSubscriptionOperations(ctx, client, *jsonFlag, *verboseFlag); err != nil {
		log.Fatalf("Failed to get subscription operations: %v", err)
	}
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, jsonMode bool) error {
	// Check if network is already set
	if client.IsNetworkSet() {
		if current, err := client.GetCurrentNetwork(ctx); err == nil {
			if !jsonMode {
				fmt.Printf("Using network: %s (ID: %d)\n", current.Name, current.ID)
			}
			return nil
		}
	}

	// If no network flag was provided, check BS_NETWORK environment variable
	if requestedNetwork == "" {
		if envNetwork := os.Getenv("BS_NETWORK"); envNetwork != "" {
			requestedNetwork = envNetwork
			if !jsonMode {
				fmt.Printf("Using network from BS_NETWORK environment variable\n")
			}
		}
	}

	// Get available networks
	if !jsonMode {
		fmt.Println("Getting available networks...")
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
					fmt.Printf("Using requested network: %s (ID: %d)\n", network.Name, network.ID)
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

func getSubscriptionOperations(ctx context.Context, client *bscontrol.Client, jsonMode bool, verbose bool) error {
	if !jsonMode {
		fmt.Println("\nRetrieving subscription operations...")
	}

	result, err := client.Subscriptions.GetOperations(ctx)
	if err != nil {
		return err
	}

	if jsonMode {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	// Display operations
	fmt.Println("\nSubscription Operations:")
	fmt.Println(strings.Repeat("=", 80))

	if len(result.Operations) == 0 {
		fmt.Println("No operations found.")
		return nil
	}

	// Separate allowed and not allowed operations
	var allowedOps []bscontrol.SubscriptionOperation
	var notAllowedOps []bscontrol.SubscriptionOperation

	for _, op := range result.Operations {
		if op.Allowed {
			allowedOps = append(allowedOps, op)
		} else {
			notAllowedOps = append(notAllowedOps, op)
		}
	}

	// Display allowed operations
	if len(allowedOps) > 0 {
		fmt.Println("\nAllowed Operations:")
		for i, op := range allowedOps {
			fmt.Printf("  [%d] %s\n", i+1, op.Name)
			if verbose && op.Description != "" {
				fmt.Printf("      Description: %s\n", op.Description)
			}
		}
	}

	// Display not allowed operations
	if len(notAllowedOps) > 0 {
		fmt.Println("\nNot Allowed Operations:")
		for i, op := range notAllowedOps {
			fmt.Printf("  [%d] %s\n", i+1, op.Name)
			if verbose && op.Description != "" {
				fmt.Printf("      Description: %s\n", op.Description)
			}
		}
	}

	fmt.Printf("\nTotal operations: %d (Allowed: %d, Not Allowed: %d)\n",
		len(result.Operations), len(allowedOps), len(notAllowedOps))

	return nil
}
