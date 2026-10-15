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
		debugFlag   = flag.Bool("debug", false, "Enable debug mode (shows all HTTP requests/responses)")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag *string
		serialFlag  = flag.String("serial", "", "Get info for device with serial number")
		idFlag      = flag.Int("id", 0, "Get info for device with ID")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for getting real-time BrightSign device information.\n")
		fmt.Fprintf(os.Stderr, "This queries the player directly via rDWS API for live hardware and system data.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get device info by serial:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial A1B2C3D4E5F6\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Verbose output (includes hardware/API features):\n")
		fmt.Fprintf(os.Stderr, "    %s --serial A1B2C3D4E5F6 --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  JSON output (full RDWSInfo object):\n")
		fmt.Fprintf(os.Stderr, "    %s --serial A1B2C3D4E5F6 --json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Debug mode (show HTTP requests/responses):\n")
		fmt.Fprintf(os.Stderr, "    %s --serial A1B2C3D4E5F6 --debug\n", os.Args[0])
	}

	flag.Parse()

	// Show help if requested
	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// Validate required arguments
	if *serialFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --serial must be specified\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *idFlag != 0 {
		fmt.Fprintf(os.Stderr, "Error: --id is not supported by rDWS API. Use --serial instead.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create client with options
	opts := []bscontrol.Option{bscontrol.WithEnv()}

	if *timeoutFlag != 30 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}

	if *networkFlag != "" {
		opts = append(opts, bscontrol.WithNetwork(*networkFlag))
	}

	if *debugFlag {
		opts = append(opts, bscontrol.WithDebug(true))
	}

	// Create client
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

	// Get device information via rDWS
	if *idFlag != 0 {
		log.Fatalf("Error: --id is not supported with rDWS API. Please use --serial instead.")
	}

	if !*jsonFlag {
		fmt.Printf("Getting real-time device information from player: %s\n", *serialFlag)
	}

	info, err := client.RDWS.GetInfo(ctx, *serialFlag)
	if err != nil {
		log.Fatalf("Failed to get device info: %v", err)
	}

	// Display device information
	if *jsonFlag {
		// Output raw JSON - the entire RDWSInfo object
		jsonData, err := json.MarshalIndent(info, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		// Display formatted output
		fmt.Println("\n=== Real-Time Device Information ===")
		printPlayerInfo(info, *verboseFlag)
	}
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, verbose, jsonMode bool) error {
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

	// If a specific network was requested (via flag or env var), try to find it
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
		fmt.Fprintf(os.Stderr, "Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)
	}

	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}

func printPlayerInfo(info *bscontrol.RDWSInfo, verbose bool) {
	// Basic player information
	fmt.Fprintf(os.Stderr, "  Serial:              %s\n", info.Serial)
	fmt.Fprintf(os.Stderr, "  Model:               %s\n", info.Model)
	fmt.Fprintf(os.Stderr, "  Family:              %s\n", info.Family)
	fmt.Fprintf(os.Stderr, "  Firmware Version:    %s\n", info.FWVersion)
	fmt.Fprintf(os.Stderr, "  Boot Version:        %s\n", info.BootVersion)
	fmt.Fprintf(os.Stderr, "  Uptime:              %s (%d seconds)\n", info.UpTime, info.UpTimeSeconds)
	fmt.Fprintf(os.Stderr, "  BSN CE:              %v\n", info.BSNCE)

	// Power status
	if info.Power != nil && info.Power.Result != nil {
		fmt.Fprintf(os.Stderr, "\n=== Power Status ===\n")
		if powerMap, ok := info.Power.Result.(map[string]interface{}); ok {
			if battery, ok := powerMap["battery"]; ok {
				fmt.Fprintf(os.Stderr, "  Battery:             %v\n", battery)
			}
			if source, ok := powerMap["source"]; ok {
				fmt.Fprintf(os.Stderr, "  Power Source:        %v\n", source)
			}
			if switchMode, ok := powerMap["switch_mode"]; ok {
				fmt.Fprintf(os.Stderr, "  Switch Mode:         %v\n", switchMode)
			}
		}
	}

	// Extensions
	if info.Extensions != nil && info.Extensions.Result != nil {
		fmt.Fprintf(os.Stderr, "\n=== Firmware Extensions ===\n")
		if extMap, ok := info.Extensions.Result.(map[string]interface{}); ok {
			if extensions, ok := extMap["extensions"].([]interface{}); ok {
				if len(extensions) > 0 {
					for _, ext := range extensions {
						fmt.Fprintf(os.Stderr, "  - %v\n", ext)
					}
				} else {
					fmt.Fprintf(os.Stderr, "  (none)\n")
				}
			}
		}
	}

	// Blessings (codec licenses)
	if info.Blessings != nil && info.Blessings.Result != nil {
		fmt.Fprintf(os.Stderr, "\n=== Codec Licenses (Blessings) ===\n")
		if blessMap, ok := info.Blessings.Result.(map[string]interface{}); ok {
			if ac3, ok := blessMap["ac3"]; ok {
				fmt.Fprintf(os.Stderr, "  AC3:                 %v\n", ac3)
			}
			if eac3, ok := blessMap["eac3"]; ok {
				fmt.Fprintf(os.Stderr, "  EAC3:                %v\n", eac3)
			}
		}
	}

	// Network interfaces
	if len(info.Ethernet) > 0 || len(info.Wireless) > 0 {
		fmt.Fprintf(os.Stderr, "\n=== Network Interfaces ===\n")

		if len(info.Ethernet) > 0 {
			fmt.Fprintf(os.Stderr, "  Ethernet:\n")
			for _, eth := range info.Ethernet {
				fmt.Fprintf(os.Stderr, "    Interface:         %s (%s)\n", eth.InterfaceName, eth.InterfaceType)

				if len(eth.IPv4) > 0 {
					fmt.Fprintf(os.Stderr, "    IPv4:\n")
					for _, ip := range eth.IPv4 {
						fmt.Fprintf(os.Stderr, "      Address:         %s\n", ip.Address)
						if verbose {
							if ip.Netmask != "" {
								fmt.Fprintf(os.Stderr, "      Netmask:         %s\n", ip.Netmask)
							}
							if ip.CIDR != "" {
								fmt.Fprintf(os.Stderr, "      CIDR:            %s\n", ip.CIDR)
							}
							if ip.MAC != "" {
								fmt.Fprintf(os.Stderr, "      MAC:             %s\n", ip.MAC)
							}
						}
					}
				}

				if len(eth.IPv6) > 0 {
					fmt.Fprintf(os.Stderr, "    IPv6:\n")
					for _, ip := range eth.IPv6 {
						fmt.Fprintf(os.Stderr, "      Address:         %s\n", ip.Address)
						if verbose && ip.CIDR != "" {
							fmt.Fprintf(os.Stderr, "      CIDR:            %s\n", ip.CIDR)
						}
					}
				}
			}
		}

		if len(info.Wireless) > 0 {
			fmt.Fprintf(os.Stderr, "  Wireless:\n")
			for _, wifi := range info.Wireless {
				fmt.Fprintf(os.Stderr, "    Interface:         %s (%s)\n", wifi.InterfaceName, wifi.InterfaceType)

				if len(wifi.IPv4) > 0 {
					fmt.Fprintf(os.Stderr, "    IPv4:\n")
					for _, ip := range wifi.IPv4 {
						fmt.Fprintf(os.Stderr, "      Address:         %s\n", ip.Address)
						if verbose {
							if ip.Netmask != "" {
								fmt.Fprintf(os.Stderr, "      Netmask:         %s\n", ip.Netmask)
							}
							if ip.CIDR != "" {
								fmt.Fprintf(os.Stderr, "      CIDR:            %s\n", ip.CIDR)
							}
							if ip.MAC != "" {
								fmt.Fprintf(os.Stderr, "      MAC:             %s\n", ip.MAC)
							}
						}
					}
				}

				if len(wifi.IPv6) > 0 {
					fmt.Fprintf(os.Stderr, "    IPv6:\n")
					for _, ip := range wifi.IPv6 {
						fmt.Fprintf(os.Stderr, "      Address:         %s\n", ip.Address)
						if verbose && ip.CIDR != "" {
							fmt.Fprintf(os.Stderr, "      CIDR:            %s\n", ip.CIDR)
						}
					}
				}
			}
		}
	}

	// Hardware features (verbose mode)
	if verbose && len(info.HardwareFeatures) > 0 {
		fmt.Fprintf(os.Stderr, "\n=== Hardware Features ===\n")
		for feature, value := range info.HardwareFeatures {
			fmt.Fprintf(os.Stderr, "  %-20s %v\n", feature+":", value)
		}
	}

	// API features (verbose mode)
	if verbose && len(info.APIFeatures) > 0 {
		fmt.Fprintf(os.Stderr, "\n=== API Features ===\n")
		for feature, value := range info.APIFeatures {
			fmt.Fprintf(os.Stderr, "  %-20s %v\n", feature+":", value)
		}
	}

	// Active features (verbose mode)
	if verbose && len(info.ActiveFeatures) > 0 {
		fmt.Fprintf(os.Stderr, "\n=== Active Features ===\n")
		for feature, value := range info.ActiveFeatures {
			fmt.Fprintf(os.Stderr, "  %-20s %v\n", feature+":", value)
		}
	}

	fmt.Fprintf(os.Stderr, "\n")
}
