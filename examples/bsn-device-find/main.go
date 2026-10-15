package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brightsign/bscontrol"
)

// SearchResult represents the result of searching for a device
type SearchResult struct {
	Found         bool             `json:"found"`
	Serial        string           `json:"serial"`
	NetworkName   string           `json:"networkName,omitempty"`
	NetworkID     int              `json:"networkId,omitempty"`
	Device        *bscontrol.Device `json:"device,omitempty"`
	NetworksCount int              `json:"networksSearched"`
	Error         string           `json:"error,omitempty"`
}

func main() {
	var (
		helpFlag    = flag.Bool("help", false, "Display usage information")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		serialFlag  = flag.String("serial", "", "Device serial number to search for (required)")
		verboseFlag = flag.Bool("verbose", false, "Show detailed search progress")
	)

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to find a BrightSign player across all available BSN.cloud networks.\n\n")
		fmt.Fprintf(os.Stderr, "This tool searches through all networks you have access to and locates\n")
		fmt.Fprintf(os.Stderr, "the specified device by serial number.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Find a device by serial number:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Find with verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Validate required parameter
	serial := *serialFlag
	if serial == "" {
		fmt.Fprintf(os.Stderr, "Error: --serial is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔧 Creating BSN.cloud client...\n")
	}

	client, err := bscontrol.New(opts...)
	if err != nil {
		result := SearchResult{
			Found:  false,
			Serial: serial,
			Error:  fmt.Sprintf("Configuration error: %v", err),
		}
		outputResult(result, *jsonFlag)
		if bscontrol.IsConfigurationError(err) {
			log.Fatalf("❌ Configuration error: %v", err)
		}
		log.Fatalf("❌ Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Authenticate
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔐 Authenticating with BSN.cloud...\n")
	}

	if err := client.Authenticate(ctx); err != nil {
		result := SearchResult{
			Found:  false,
			Serial: serial,
			Error:  fmt.Sprintf("Authentication error: %v", err),
		}
		outputResult(result, *jsonFlag)
		if bscontrol.IsAuthenticationError(err) {
			log.Fatalf("❌ Authentication failed: %v", err)
		}
		log.Fatalf("❌ Authentication error: %v", err)
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Authentication successful!\n")
	}

	// Get all available networks
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📡 Retrieving available networks...\n")
	}

	networks, err := client.GetNetworks(ctx)
	if err != nil {
		result := SearchResult{
			Found:  false,
			Serial: serial,
			Error:  fmt.Sprintf("Failed to get networks: %v", err),
		}
		outputResult(result, *jsonFlag)
		log.Fatalf("❌ Failed to get networks: %v", err)
	}

	if len(networks) == 0 {
		result := SearchResult{
			Found:         false,
			Serial:        serial,
			NetworksCount: 0,
			Error:         "No networks available",
		}
		outputResult(result, *jsonFlag)
		log.Fatalf("❌ No networks available")
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Found %d network(s) to search\n", len(networks))
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "🔍 Searching for device: %s\n", serial)
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Search through each network
	var foundDevice *bscontrol.Device
	var foundNetwork *bscontrol.Network

	for i, network := range networks {
		if *verboseFlag && !*jsonFlag {
			fmt.Fprintf(os.Stderr, "  [%d/%d] Searching network: %s (ID: %d)\n",
				i+1, len(networks), network.Name, network.ID)
		}

		// Set network context
		if err := client.SetNetworkByID(ctx, network.ID); err != nil {
			if *verboseFlag && !*jsonFlag {
				fmt.Fprintf(os.Stderr, "        ⚠ Failed to set network context\n")
			}
			continue
		}

		// Get device from this network
		device, err := client.Devices.Get(ctx, serial)
		if err != nil {
			// Device not found in this network, continue to next
			if *verboseFlag && !*jsonFlag {
				fmt.Fprintf(os.Stderr, "        ⊘ Not found\n")
			}
			continue
		}

		// Found the device!
		foundDevice = device
		foundNetwork = &network
		if *verboseFlag && !*jsonFlag {
			fmt.Fprintf(os.Stderr, "        ✅ Found!\n")
		}
		break
	}

	// Prepare result
	result := SearchResult{
		Found:         foundDevice != nil,
		Serial:        serial,
		NetworksCount: len(networks),
	}

	if foundDevice != nil {
		result.Device = foundDevice
		result.NetworkName = foundNetwork.Name
		result.NetworkID = foundNetwork.ID
	}

	// Output result
	outputResult(result, *jsonFlag)

	// Exit with appropriate code
	if !result.Found {
		os.Exit(1)
	}
}

func outputResult(result SearchResult, jsonOutput bool) {
	if jsonOutput {
		// JSON output
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Human-friendly output
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	if result.Found {
		fmt.Fprintf(os.Stderr, "✅ Device Found!\n")
		fmt.Fprintf(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "📱 Device Information:\n")
		fmt.Fprintf(os.Stderr, "  Serial:       %s\n", result.Serial)
		fmt.Fprintf(os.Stderr, "  Network:      %s (ID: %d)\n", result.NetworkName, result.NetworkID)

		if result.Device != nil {
			if result.Device.Settings != nil {
				fmt.Fprintf(os.Stderr, "  Name:         %s\n", result.Device.Settings.Name)
				if result.Device.Settings.Description != "" {
					fmt.Fprintf(os.Stderr, "  Description:  %s\n", result.Device.Settings.Description)
				}
				if result.Device.Settings.Group != nil {
					fmt.Fprintf(os.Stderr, "  Group:        %s\n", result.Device.Settings.Group.Name)
				}
			}
			fmt.Fprintf(os.Stderr, "  Model:        %s\n", result.Device.Model)
			fmt.Fprintf(os.Stderr, "  Family:       %s\n", result.Device.Family)
		}

		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "📊 Search Statistics:\n")
		fmt.Fprintf(os.Stderr, "  Networks searched: %d\n", result.NetworksCount)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "💡 Tip: Use --json flag to get machine-readable output\n")
	} else {
		fmt.Fprintf(os.Stderr, "❌ Device Not Found\n")
		fmt.Fprintf(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "  Serial:            %s\n", result.Serial)
		fmt.Fprintf(os.Stderr, "  Networks searched: %d\n", result.NetworksCount)

		if result.Error != "" {
			fmt.Fprintf(os.Stderr, "  Error:             %s\n", result.Error)
		}

		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "💡 Troubleshooting:\n")
		fmt.Fprintf(os.Stderr, "  • Verify the serial number is correct\n")
		fmt.Fprintf(os.Stderr, "  • Ensure the device is registered in one of your networks\n")
		fmt.Fprintf(os.Stderr, "  • Check that you have access to the network containing the device\n")
		fmt.Fprintf(os.Stderr, "  • Use --verbose flag to see which networks were searched\n")
	}
	fmt.Fprintf(os.Stderr, "\n")
}
