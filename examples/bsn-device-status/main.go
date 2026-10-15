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
		serialFlag  = flag.String("serial", "", "Get status for device with serial number")
		idFlag      = flag.Int("id", 0, "Get status for device with ID")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for getting BrightSign device status information.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get status by serial:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial A1B2C3D4E5F6\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get status by device ID:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial A1B2C3D4E5F6 --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  JSON output:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial A1B2C3D4E5F6 --json\n", os.Args[0])
	}

	flag.Parse()

	// Show help if requested
	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// Validate required arguments
	if *serialFlag == "" && *idFlag == 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: Either --serial or --id must be specified\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *serialFlag != "" && *idFlag != 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: Cannot specify both --serial and --id\n\n")
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

	// Create client
	if !*jsonFlag {
		fmt.Println("🔧 Creating BSN.cloud client...")
	}

	client, err := bscontrol.New(opts...)
	if err != nil {
		if bscontrol.IsConfigurationError(err) {
			log.Fatalf("❌ Configuration error: %v", err)
		}
		log.Fatalf("❌ Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Authenticate
	if !*jsonFlag {
		fmt.Println("🔐 Authenticating with BSN.cloud...")
	}

	if err := client.Authenticate(ctx); err != nil {
		if bscontrol.IsAuthenticationError(err) {
			log.Fatalf("❌ Authentication failed: %v", err)
		}
		log.Fatalf("❌ Authentication error: %v", err)
	}

	if !*jsonFlag {
		fmt.Println("✅ Authentication successful!")
	}

	// Handle network selection
	if err := handleNetworkSelection(ctx, client, *networkFlag, *verboseFlag, *jsonFlag); err != nil {
		log.Fatalf("❌ Network selection failed: %v", err)
	}

	// Get device status
	var status *bscontrol.DeviceStatus

	if *serialFlag != "" {
		if !*jsonFlag {
			fmt.Printf("📊 Getting status for device with serial: %s\n", *serialFlag)
		}
		status, err = client.Devices.GetStatusBySerial(ctx, *serialFlag)
		if err != nil {
			log.Fatalf("❌ Failed to get device status: %v", err)
		}
	} else {
		if !*jsonFlag {
			fmt.Printf("📊 Getting status for device with ID: %d\n", *idFlag)
		}
		status, err = client.Devices.GetStatus(ctx, *idFlag)
		if err != nil {
			log.Fatalf("❌ Failed to get device status: %v", err)
		}
	}

	// Display status
	if *jsonFlag {
		// Output raw JSON
		jsonData, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			log.Fatalf("❌ Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		// Display formatted output
		fmt.Printf("\n🎯 Device Status:\n")
		printStatus(status, *verboseFlag)
	}
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, verbose, jsonMode bool) error {
	// Check if network is already set
	if client.IsNetworkSet() {
		if current, err := client.GetCurrentNetwork(ctx); err == nil {
			if !jsonMode {
				fmt.Printf("📡 Using network: %s (ID: %d)\n", current.Name, current.ID)
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
		fmt.Println("📡 Getting available networks...")
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
					fmt.Printf("📡 Using requested network: %s (ID: %d)\n", network.Name, network.ID)
				}
				return client.SetNetworkByID(ctx, network.ID)
			}
		}

		// Network not found - show error and fall back to interactive selection
		fmt.Printf("❌ Network '%s' not found. Available networks:\n", requestedNetwork)
		for i, network := range networks {
			fmt.Printf("  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// If only one network and no specific network requested, use it automatically
	if len(networks) == 1 && requestedNetwork == "" {
		if !jsonMode {
			fmt.Printf("📡 Using network: %s (ID: %d)\n", networks[0].Name, networks[0].ID)
		}
		return client.SetNetworkByID(ctx, networks[0].ID)
	}

	// Show available networks and let user choose
	if requestedNetwork == "" {
		fmt.Println("📡 Available networks:")
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
	fmt.Printf("📡 Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)

	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}

func printStatus(status *bscontrol.DeviceStatus, verbose bool) {
	fmt.Fprintf(os.Stderr, "  Device ID:        %s\n", status.DeviceID)
	fmt.Fprintf(os.Stderr, "  Serial:           %s\n", status.Serial)
	fmt.Fprintf(os.Stderr, "  Model:            %s\n", status.Model)
	fmt.Fprintf(os.Stderr, "  Firmware:         %s\n", status.FirmwareVersion)

	// Status indicator with emoji
	statusIcon := "🔴"
	if status.IsOnline {
		statusIcon = "🟢"
	}
	fmt.Fprintf(os.Stderr, "  Online Status:    %s %v\n", statusIcon, status.IsOnline)
	fmt.Fprintf(os.Stderr, "  Status:           %s\n", status.Status)

	// Health status with emoji
	healthIcon := "❓"
	switch status.HealthStatus {
	case "Healthy":
		healthIcon = "✅"
	case "Warning":
		healthIcon = "⚠️"
	case "Error":
		healthIcon = "❌"
	}
	fmt.Fprintf(os.Stderr, "  Health Status:    %s %s\n", healthIcon, status.HealthStatus)

	// Last seen time
	fmt.Fprintf(os.Stderr, "  Last Seen:        %s", status.LastSeen.Format("2006-01-02 15:04:05 MST"))
	timeSince := time.Since(status.LastSeen)
	if timeSince < time.Minute {
		fmt.Fprintf(os.Stderr, " (just now)")
	} else if timeSince < time.Hour {
		fmt.Fprintf(os.Stderr, " (%d minutes ago)", int(timeSince.Minutes()))
	} else if timeSince < 24*time.Hour {
		fmt.Fprintf(os.Stderr, " (%d hours ago)", int(timeSince.Hours()))
	} else {
		fmt.Fprintf(os.Stderr, " (%d days ago)", int(timeSince.Hours()/24))
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Uptime display
	if status.UptimeDisplay != "" {
		fmt.Fprintf(os.Stderr, "  Uptime:           %s\n", status.UptimeDisplay)
	} else if status.Uptime > 0 {
		uptime := time.Duration(status.Uptime) * time.Second
		fmt.Fprintf(os.Stderr, "  Uptime:           %v\n", uptime)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "  Last Health Check: %s\n", status.LastHealthCheck.Format("2006-01-02 15:04:05 MST"))

		if status.IPAddress != "" {
			fmt.Fprintf(os.Stderr, "  IP Address:       %s\n", status.IPAddress)
		}

		if status.ConnectionType != "" {
			connIcon := "🔌"
			if status.ConnectionType == "wifi" {
				connIcon = "📶"
			}
			fmt.Fprintf(os.Stderr, "  Connection:       %s %s\n", connIcon, status.ConnectionType)

			if status.SignalStrength > 0 && status.ConnectionType == "wifi" {
				signalIcon := "📶"
				if status.SignalStrength < 25 {
					signalIcon = "📶" // weak
				} else if status.SignalStrength < 50 {
					signalIcon = "📶" // fair
				} else if status.SignalStrength < 75 {
					signalIcon = "📶" // good
				} else {
					signalIcon = "📶" // excellent
				}
				fmt.Fprintf(os.Stderr, "  Signal Strength:  %s %d%%\n", signalIcon, status.SignalStrength)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\n")
}
