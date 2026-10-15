package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
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
		serialFlag   = flag.String("serial", "", "Device serial number to delete")
		deviceIDFlag = flag.String("device-id", "", "Device ID to delete (alternative to --serial)")
		forceFlag    = flag.Bool("force", false, "Skip confirmation prompt")
		networkFlag  *string
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to completely remove a BrightSign device from the B-Deploy system.\n\n")
		fmt.Fprintf(os.Stderr, "⚠️  WARNING: This permanently removes the device record from B-Deploy!\n")
		fmt.Fprintf(os.Stderr, "The device will need to be re-added to B-Deploy to provision again.\n\n")
		fmt.Fprintf(os.Stderr, "NOTE: To keep the device but remove its setup association, use:\n")
		fmt.Fprintf(os.Stderr, "      bdeploy-associate --serial XXX --dissociate\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Delete device by serial number:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABC123456789\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Delete device by device ID:\n")
		fmt.Fprintf(os.Stderr, "    %s --device-id 658f1dbef1d46c829f60a14f\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Specify network explicitly:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABC123456789 --network \"Production\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Force delete without confirmation:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABC123456789 --force\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABC123456789 --json --force\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Validate that either serial or device-id is provided
	if *serialFlag == "" && *deviceIDFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: Either --serial or --device-id must be specified\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *serialFlag != "" && *deviceIDFlag != "" {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify both --serial and --device-id\n\n")
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
		if bscontrol.IsAuthenticationError(err) {
			log.Fatalf("❌ Authentication failed: %v", err)
		}
		log.Fatalf("❌ Authentication error: %v", err)
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Authentication successful!\n")
	}

	// Determine network to use
	networkName := getNetworkName(*networkFlag, client, ctx, *verboseFlag)

	// Set network context for B-Deploy operations
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📡 Setting network context to: %s\n", networkName)
	}
	if err := client.BDeploy.SetNetworkContext(ctx, networkName); err != nil {
		log.Fatalf("❌ Failed to set network context: %v", err)
	}

	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Network context set successfully!\n")
	}

	// Get device info for confirmation (if serial provided)
	var deviceInfo string
	if *serialFlag != "" {
		deviceInfo = fmt.Sprintf("Serial: %s", *serialFlag)

		// Try to fetch device details for better confirmation
		if *verboseFlag && !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔍 Looking up device: %s\n", *serialFlag)
		}

		device, err := client.BDeploy.GetDeviceBySerial(ctx, *serialFlag)
		if err == nil && device.Result.Matched > 0 && len(device.Result.Players) > 0 {
			player := device.Result.Players[0]
			deviceInfo = fmt.Sprintf("Serial: %s\n    Name: %s\n    Network: %s\n    Description: %s",
				player.Serial, player.Name, player.NetworkName, player.Desc)
		}
	} else {
		deviceInfo = fmt.Sprintf("Device ID: %s", *deviceIDFlag)
	}

	// Confirm deletion unless --force is used
	if !*forceFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "⚠️  WARNING: This will PERMANENTLY DELETE the device from B-Deploy!\n")
		fmt.Fprintf(os.Stderr, "    The device will be completely removed from the provisioning system.\n")
		fmt.Fprintf(os.Stderr, "    To provision this device again, you will need to re-add it.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Device to delete:\n")
		fmt.Fprintf(os.Stderr, "    %s\n", deviceInfo)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "💡 TIP: To keep the device but remove its setup association, use:\n")
		fmt.Fprintf(os.Stderr, "         bdeploy-associate --serial XXX --dissociate\n")
		fmt.Fprintf(os.Stderr, "\n")

		confirmed := confirmDeletion()
		if !confirmed {
			fmt.Fprintf(os.Stderr, "❌ Operation cancelled.\n")
			return
		}
	}

	// Delete the device
	identifier := *serialFlag
	if identifier == "" {
		identifier = *deviceIDFlag
	}
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🗑️  Deleting device from B-Deploy: %s\n", identifier)
	}

	err = client.BDeploy.DeleteDevice(ctx, *deviceIDFlag, *serialFlag)
	if err != nil {
		log.Fatalf("❌ Failed to delete device: %v", err)
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success":  true,
			"message":  "Device successfully deleted from B-Deploy",
			"serial":   *serialFlag,
			"deviceId": *deviceIDFlag,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Display results
	fmt.Fprintf(os.Stderr, "✅ Device successfully deleted from B-Deploy!\n")

	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "📋 The device has been removed from the provisioning system.\n")
		fmt.Fprintf(os.Stderr, "   If the device boots and contacts B-Deploy, it will no longer be recognized.\n")
		fmt.Fprintf(os.Stderr, "   To provision this device again, you must re-add it using bdeploy-associate.\n")
	}
}

func confirmDeletion() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Type 'DELETE' to confirm (or anything else to cancel): ")

	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("❌ Failed to read input: %v", err)
	}

	response = strings.TrimSpace(response)
	return response == "DELETE"
}

func getNetworkName(requestedNetwork string, client *bscontrol.Client, ctx context.Context, verbose bool) string {
	// If network was specified via flag, use it
	if requestedNetwork != "" {
		return requestedNetwork
	}

	// Check if network is already set in client
	if client.IsNetworkSet() {
		if current, err := client.GetCurrentNetwork(ctx); err == nil {
			if verbose {
				fmt.Printf("📡 Using current network: %s (ID: %d)\n", current.Name, current.ID)
			}
			return current.Name
		}
	}

	// Check environment variable
	if envNetwork := os.Getenv("BS_NETWORK"); envNetwork != "" {
		if verbose {
			fmt.Printf("📡 Using network from BS_NETWORK: %s\n", envNetwork)
		}
		return envNetwork
	}

	// Need to select a network
	fmt.Println("📡 Getting available networks...")

	networks, err := client.GetNetworks(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to get networks: %v", err)
	}

	if len(networks) == 0 {
		log.Fatalf("❌ No networks available")
	}

	// If only one network, use it automatically
	if len(networks) == 1 {
		networkName := networks[0].Name
		if verbose {
			fmt.Printf("📡 Using network: %s (ID: %d)\n", networkName, networks[0].ID)
		}
		return networkName
	}

	// Multiple networks - need user to specify
	fmt.Println("❌ Multiple networks available. Please specify --network or set BS_NETWORK:")
	for i, network := range networks {
		fmt.Printf("  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
	}
	os.Exit(1)
	return ""
}
