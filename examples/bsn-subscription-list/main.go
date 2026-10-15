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
		pageSizeFlag = flag.Int("page-size", 100, "Number of items per page")
		filterFlag   = flag.String("filter", "", "Filter expression (e.g., \"status eq 'active'\")")
		sortFlag     = flag.String("sort", "", "Sort expression (e.g., \"startDate desc\")")
		allFlag      = flag.Bool("all", false, "Retrieve all subscriptions (paginate through all results)")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for listing device subscriptions on a BSN.cloud network.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  List all subscriptions:\n")
		fmt.Fprintf(os.Stderr, "    %s --all\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  List active subscriptions only:\n")
		fmt.Fprintf(os.Stderr, "    %s --filter \"status eq 'active'\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  List subscriptions sorted by start date:\n")
		fmt.Fprintf(os.Stderr, "    %s --sort \"startDate desc\" --all\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --json\n", os.Args[0])
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
	if err := handleNetworkSelection(ctx, client, *networkFlag, *verboseFlag, *jsonFlag); err != nil {
		log.Fatalf("Network selection failed: %v", err)
	}

	// List subscriptions
	if err := listSubscriptions(ctx, client, *jsonFlag, *verboseFlag, *pageSizeFlag, *filterFlag, *sortFlag, *allFlag); err != nil {
		log.Fatalf("Failed to list subscriptions: %v", err)
	}
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, verbose bool, jsonMode bool) error {
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

func listSubscriptions(ctx context.Context, client *bscontrol.Client, jsonMode bool, verbose bool, pageSize int, filter string, sort string, allPages bool) error {
	if !jsonMode {
		fmt.Println("\nRetrieving device subscriptions...")
	}

	// Build list options
	var listOpts []bscontrol.ListOption
	if pageSize > 0 {
		listOpts = append(listOpts, bscontrol.WithPageSize(pageSize))
	}
	if filter != "" {
		listOpts = append(listOpts, bscontrol.WithFilter(filter))
	}
	if sort != "" {
		listOpts = append(listOpts, bscontrol.WithSort(sort))
	}

	var allSubscriptions []bscontrol.Subscription
	var totalRetrieved int

	for {
		result, err := client.Subscriptions.List(ctx, listOpts...)
		if err != nil {
			return err
		}

		allSubscriptions = append(allSubscriptions, result.Items...)
		totalRetrieved += len(result.Items)

		if !jsonMode && !allPages {
			fmt.Printf("Retrieved %d subscriptions\n", len(result.Items))
		}

		// If not fetching all pages or no more pages, break
		if !allPages || !result.IsTruncated {
			break
		}

		// Update marker for next page
		if !jsonMode {
			fmt.Printf("Retrieved %d subscriptions so far, fetching next page...\n", totalRetrieved)
		}
		listOpts = updateMarker(listOpts, result.NextMarker)
	}

	if jsonMode {
		// Output all subscriptions as JSON
		output := bscontrol.SubscriptionList{
			Items:       allSubscriptions,
			TotalCount:  len(allSubscriptions),
			IsTruncated: false,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Display subscriptions
	fmt.Println("\nDevice Subscriptions:")
	fmt.Println(strings.Repeat("=", 80))

	if len(allSubscriptions) == 0 {
		fmt.Println("No subscriptions found.")
		return nil
	}

	for i, sub := range allSubscriptions {
		fmt.Printf("\n[%d] Subscription ID: %d\n", i+1, sub.ID)
		fmt.Printf("    Type: %s\n", sub.Type)
		fmt.Printf("    Status: %s\n", sub.Status)

		if sub.DeviceSerial != "" {
			fmt.Printf("    Device Serial: %s\n", sub.DeviceSerial)
		}
		if sub.DeviceID > 0 {
			fmt.Printf("    Device ID: %d\n", sub.DeviceID)
		}

		if !sub.StartDate.IsZero() {
			fmt.Printf("    Start Date: %s\n", sub.StartDate.Format("2006-01-02"))
		}

		if sub.EndDate != nil && !sub.EndDate.IsZero() {
			fmt.Printf("    End Date: %s\n", sub.EndDate.Format("2006-01-02"))
		}

		if sub.AutoRenew {
			fmt.Printf("    Auto-Renew: Yes\n")
		}

		if verbose {
			if !sub.CreationDate.IsZero() {
				fmt.Printf("    Created: %s\n", sub.CreationDate.Format("2006-01-02 15:04:05"))
			}
			if !sub.LastModifiedDate.IsZero() {
				fmt.Printf("    Modified: %s\n", sub.LastModifiedDate.Format("2006-01-02 15:04:05"))
			}
		}
	}

	fmt.Printf("\nTotal subscriptions: %d\n", len(allSubscriptions))

	return nil
}

func updateMarker(opts []bscontrol.ListOption, newMarker string) []bscontrol.ListOption {
	// Remove old marker if present and add new one
	var newOpts []bscontrol.ListOption
	for _, opt := range opts {
		newOpts = append(newOpts, opt)
	}
	newOpts = append(newOpts, bscontrol.WithMarker(newMarker))
	return newOpts
}
