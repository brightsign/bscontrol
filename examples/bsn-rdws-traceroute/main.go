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
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag *string
		serialFlag  = flag.String("serial", "", "Device serial number (required)")
		hostFlag    = flag.String("host", "", "Host to trace route to (required, e.g., 8.8.8.8 or google.com)")
	)

	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for performing trace route from players via rDWS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Trace route to an IP:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --host 8.8.8.8\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	if *serialFlag == "" || *hostFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: Must specify --serial and --host\n\n")
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
		fmt.Printf("Tracing route to %s from device (this may take a minute)...\n", *hostFlag)
	}

	result, err := client.RDWS.TraceRoute(ctx, *serialFlag, *hostFlag)
	if err != nil {
		log.Fatalf("Failed to trace route: %v", err)
	}

	if *jsonFlag {
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		displayTraceRouteResult(result)
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

func displayTraceRouteResult(result *bscontrol.RDWSTraceRouteResult) {
	fmt.Fprintf(os.Stderr, "\n=== Trace Route Result ===\n")
	if result.Success {
		fmt.Fprintf(os.Stderr, "✓ Success\n")
		fmt.Fprintf(os.Stderr, "Host: %s\n\n", result.Host)

		if len(result.Hops) > 0 {
			fmt.Fprintf(os.Stderr, "Hops:\n")
			for _, hop := range result.Hops {
				if hop.Timeout {
					fmt.Fprintf(os.Stderr, "  %2d.  * * * (timeout)\n", hop.Hop)
				} else {
					fmt.Fprintf(os.Stderr, "  %2d.  %-15s  %.2f ms\n", hop.Hop, hop.Address, hop.RTT)
				}
			}
		}

		if result.Output != "" {
			fmt.Fprintf(os.Stderr, "\nRaw Output:\n%s\n", result.Output)
		}
	} else {
		fmt.Fprintf(os.Stderr, "✗ Failed\n")
		if result.Error != "" {
			fmt.Fprintf(os.Stderr, "Error: %s\n", result.Error)
		}
	}
	fmt.Fprintf(os.Stderr, "\n")
}
