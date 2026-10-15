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
		jsonFlag    = flag.Bool("json", false, "Output raw JSON response")
		timeoutFlag = flag.Int("timeout", 60, "Request timeout in seconds (default: 60, network scan may take time)")
		networkFlag *string
		serialFlag  = flag.String("serial", "", "Device serial number (required)")
	)

	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for discovering network neighborhood from players via rDWS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Scan network:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
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

	if !*jsonFlag {
		fmt.Printf("Scanning network neighborhood from device %s (this may take a minute)...\n", *serialFlag)
	}

	result, err := client.RDWS.GetNetworkNeighborhood(ctx, *serialFlag)
	if err != nil {
		log.Fatalf("Failed to get network neighborhood: %v", err)
	}

	if *jsonFlag {
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		displayNeighborhood(result)
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
					fmt.Printf("Using network: %s (ID: %d)\n", network.Name, network.ID)
				}
				return client.SetNetworkByID(ctx, network.ID)
			}
		}
	}

	if len(networks) == 1 {
		if !jsonMode {
			fmt.Printf("Using network: %s (ID: %d)\n", networks[0].Name, networks[0].ID)
		}
		return client.SetNetworkByID(ctx, networks[0].ID)
	}

	fmt.Fprintf(os.Stderr, "Available networks:\n")
	for i, network := range networks {
		fmt.Printf("  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
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
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)
	}
	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}

func displayNeighborhood(result *bscontrol.RDWSNetworkNeighborhoodResult) {
	fmt.Fprintf(os.Stderr, "\n=== Network Neighborhood ===\n")
	if result.Success {
		if len(result.Neighbors) > 0 {
			fmt.Fprintf(os.Stderr, "Found %d device(s) on the network:\n\n", len(result.Neighbors))
			for i, neighbor := range result.Neighbors {
				fmt.Fprintf(os.Stderr, "%d. IP Address:  %s\n", i+1, neighbor.IPAddress)
				if neighbor.MACAddress != "" {
					fmt.Fprintf(os.Stderr, "   MAC Address: %s\n", neighbor.MACAddress)
				}
				if neighbor.Hostname != "" {
					fmt.Fprintf(os.Stderr, "   Hostname:    %s\n", neighbor.Hostname)
				}
				fmt.Fprintf(os.Stderr, "\n")
			}
		} else {
			fmt.Fprintf(os.Stderr, "No neighbors found on the network\n")
		}
	} else {
		fmt.Fprintf(os.Stderr, "✗ Network scan failed\n")
	}
}
