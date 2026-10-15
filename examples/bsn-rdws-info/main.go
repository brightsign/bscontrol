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
		jsonFlag    = flag.Bool("json", false, "Output raw JSON response")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag *string
		serialFlag  = flag.String("serial", "", "Device serial number (required)")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for retrieving player information via rDWS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get player information:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get information with verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get information as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --json\n", os.Args[0])
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

	// Get player information
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Getting player information for device %s...\n", *serialFlag)
	}

	info, err := client.RDWS.GetInfo(ctx, *serialFlag)
	if err != nil {
		log.Fatalf("Failed to get player info: %v", err)
	}

	// Display results
	if *jsonFlag {
		jsonData, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		displayPlayerInfo(info, *verboseFlag)
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
		fmt.Fprintf(os.Stderr, "Network '%s' not found. Available networks:\n", requestedNetwork)
		for i, network := range networks {
			fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// If only one network and no specific network requested, use it automatically
	if len(networks) == 1 && requestedNetwork == "" {
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "Using network: %s (ID: %d)\n", networks[0].Name, networks[0].ID)
		}
		return client.SetNetworkByID(ctx, networks[0].ID)
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
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)
	}

	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}

func displayPlayerInfo(info *bscontrol.RDWSInfo, verbose bool) {
	fmt.Fprintf(os.Stderr, "\n=== Player Information ===\n")
	fmt.Fprintf(os.Stderr, "Serial:          %s\n", info.Serial)
	fmt.Fprintf(os.Stderr, "Model:           %s\n", info.Model)
	fmt.Fprintf(os.Stderr, "Family:          %s\n", info.Family)
	fmt.Fprintf(os.Stderr, "Firmware:        %s\n", info.FWVersion)
	fmt.Fprintf(os.Stderr, "Boot Version:    %s\n", info.BootVersion)
	fmt.Fprintf(os.Stderr, "Is Player:       %v\n", info.IsPlayer)
	fmt.Fprintf(os.Stderr, "Uptime:          %s (%d seconds)\n", info.UpTime, info.UpTimeSeconds)
	fmt.Fprintf(os.Stderr, "Connection Type: %s\n", info.ConnectionType)
	fmt.Fprintf(os.Stderr, "BSN CE:          %v\n", info.BSNCE)

	// Network interfaces
	displayNetworkInterfaces("Ethernet", info.Ethernet)
	displayNetworkInterfaces("Wireless", info.Wireless)
	displayNetworkInterfaces("Other Interfaces", info.Interfaces)

	// Sub-result sections
	displaySubResult("Power", info.Power)
	displaySubResult("POE", info.POE)
	displaySubResult("Extensions", info.Extensions)
	displaySubResult("Blessings", info.Blessings)
	displaySubResult("Networking", info.Networking)
	displaySubResult("BVN Pipelines", info.BVNPipelines)
	displaySubResult("BVN Components", info.BVNComponents)

	// Feature maps
	displayFeatureMap("Hardware Features", info.HardwareFeatures)
	displayFeatureMap("API Features", info.APIFeatures)
	displayFeatureMap("Active Features", info.ActiveFeatures)

	fmt.Fprintf(os.Stderr, "\n")
}

func displayNetworkInterfaces(title string, interfaces []bscontrol.RDWSNetworkInterface) {
	if len(interfaces) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "\n=== %s ===\n", title)
	for _, iface := range interfaces {
		fmt.Fprintf(os.Stderr, "\nInterface: %s (%s)\n", iface.InterfaceName, iface.InterfaceType)
		if len(iface.IPv4) > 0 {
			fmt.Fprintf(os.Stderr, "  IPv4:\n")
			for _, ip := range iface.IPv4 {
				fmt.Fprintf(os.Stderr, "    Address: %s\n", ip.Address)
				fmt.Fprintf(os.Stderr, "    CIDR:    %s\n", ip.CIDR)
				fmt.Fprintf(os.Stderr, "    Netmask: %s\n", ip.Netmask)
				fmt.Fprintf(os.Stderr, "    MAC:     %s\n", ip.MAC)
			}
		}
		if len(iface.IPv6) > 0 {
			fmt.Fprintf(os.Stderr, "  IPv6:\n")
			for _, ip := range iface.IPv6 {
				fmt.Fprintf(os.Stderr, "    Address:  %s\n", ip.Address)
				fmt.Fprintf(os.Stderr, "    CIDR:     %s\n", ip.CIDR)
				if ip.ScopeID != 0 {
					fmt.Fprintf(os.Stderr, "    Scope ID: %d\n", ip.ScopeID)
				}
			}
		}
	}
}

func displaySubResult(title string, sub *bscontrol.RDWSInfoSubResult) {
	if sub == nil || sub.Result == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "\n=== %s ===\n", title)
	jsonData, err := json.MarshalIndent(sub.Result, "  ", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "  %v\n", sub.Result)
		return
	}
	fmt.Fprintf(os.Stderr, "  %s\n", string(jsonData))
}

func displayFeatureMap(title string, features map[string]interface{}) {
	if len(features) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "\n=== %s ===\n", title)
	for feature, value := range features {
		fmt.Fprintf(os.Stderr, "  %-30s %v\n", feature+":", value)
	}
}
