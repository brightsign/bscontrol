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
		helpFlag    = flag.Bool("help", false, "Display usage information")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
		verboseFlag = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag *string
		serialFlag  = flag.String("serial", "", "Delete device with serial number")
		idFlag      = flag.Int("id", 0, "Delete device with ID")
		forceFlag   = flag.Bool("force", false, "Skip confirmation prompt")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")
	flag.BoolVar(forceFlag, "y", false, "Skip confirmation prompt [alias for --force]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for deleting BrightSign devices from BSN.cloud.\n\n")
		fmt.Fprintf(os.Stderr, "⚠️  WARNING: This permanently removes devices from your network.\n")
		fmt.Fprintf(os.Stderr, "           Deleted devices must be re-provisioned to rejoin.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Delete by serial number (with confirmation):\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Delete by ID without confirmation:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 --force\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Delete with verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Delete with specific network:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 -n \"MyNetwork\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --force --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Validate input
	if *serialFlag == "" && *idFlag == 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: Must specify either --serial or --id\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *serialFlag != "" && *idFlag != 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: Cannot specify both --serial and --id\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create client
	opts := []bscontrol.Option{
		bscontrol.WithEnv(),
		bscontrol.WithTimeout(time.Duration(*timeoutFlag) * time.Second),
	}

	if *networkFlag != "" {
		opts = append(opts, bscontrol.WithNetwork(*networkFlag))
	}

	client, err := bscontrol.New(opts...)
	if err != nil {
		if bscontrol.IsConfigurationError(err) {
			log.Fatalf("❌ Configuration error: %v", err)
		}
		log.Fatalf("❌ Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Step 1: Authenticate
	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔐 Authenticating with BSN.cloud...\n")
	}
	if err := client.Authenticate(ctx); err != nil {
		if bscontrol.IsAuthenticationError(err) {
			log.Fatalf("❌ Authentication failed: %v", err)
		}
		log.Fatalf("❌ Authentication error: %v", err)
	}
	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Authentication successful\n")
	}

	// Step 2: Set network context
	if err := client.EnsureReady(ctx); err != nil {
		log.Fatalf("❌ Failed to set network context: %v", err)
	}

	currentNetwork, err := client.GetCurrentNetwork(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to get current network: %v", err)
	}

	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📡 Using network: %s\n", currentNetwork.Name)
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Step 3: Get device information
	var device *bscontrol.Device
	if *serialFlag != "" {
		if *verboseFlag && !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔍 Looking up device with serial: %s\n", *serialFlag)
		}
		device, err = client.Devices.Get(ctx, *serialFlag)
		if err != nil {
			log.Fatalf("❌ Failed to find device: %v", err)
		}
	} else {
		if *verboseFlag && !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔍 Looking up device with ID: %d\n", *idFlag)
		}
		device, err = client.Devices.GetByID(ctx, *idFlag)
		if err != nil {
			log.Fatalf("❌ Failed to find device: %v", err)
		}
	}

	// Display device information
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\n%s\n", strings.Repeat("=", 70))
		fmt.Fprintf(os.Stderr, "Device Information\n")
		fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("=", 70))
		if device.Settings != nil {
			fmt.Fprintf(os.Stderr, "Name:         %s\n", device.Settings.Name)
		}
		fmt.Fprintf(os.Stderr, "Serial:       %s\n", device.Serial)
		fmt.Fprintf(os.Stderr, "ID:           %d\n", device.ID)
		fmt.Fprintf(os.Stderr, "Model:        %s\n", device.Model)
		if device.Settings != nil && device.Settings.Description != "" {
			fmt.Fprintf(os.Stderr, "Description:  %s\n", device.Settings.Description)
		}
		if device.Settings != nil && device.Settings.Group != nil {
			fmt.Fprintf(os.Stderr, "Group:        %s\n", device.Settings.Group.Name)
		}
		fmt.Fprintf(os.Stderr, "%s\n\n", strings.Repeat("=", 70))
	}

	deviceName := "Unknown"
	if device.Settings != nil && device.Settings.Name != "" {
		deviceName = device.Settings.Name
	}

	// Step 4: Confirmation prompt
	if !*forceFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "⚠️  WARNING: This will permanently delete device '%s' from BSN.cloud!\n", deviceName)
		fmt.Fprintf(os.Stderr, "           Serial number: %s\n", device.Serial)
		fmt.Fprintf(os.Stderr, "           The device will need to be re-provisioned to rejoin the network.\n\n")
		fmt.Fprintf(os.Stderr, "Are you sure you want to delete this device? [yes/no]: ")

		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("❌ Error reading input: %v", err)
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" && response != "y" {
			fmt.Fprintf(os.Stderr, "\n❌ Deletion cancelled\n")
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Step 5: Delete device
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🗑️  Deleting device '%s' (Serial: %s)...\n", deviceName, device.Serial)
	}

	var deleteErr error
	if *serialFlag != "" {
		deleteErr = client.Devices.DeleteBySerial(ctx, *serialFlag)
	} else {
		deleteErr = client.Devices.Delete(ctx, *idFlag)
	}

	if deleteErr != nil {
		log.Fatalf("❌ Failed to delete device: %v", deleteErr)
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success":     true,
			"deviceName":  deviceName,
			"serial":      device.Serial,
			"deviceID":    device.ID,
			"networkName": currentNetwork.Name,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Success
	fmt.Fprintf(os.Stderr, "✅ Device deleted successfully\n")
	fmt.Fprintf(os.Stderr, "\n%s\n", strings.Repeat("=", 70))
	fmt.Fprintf(os.Stderr, "Device '%s' has been removed from network '%s'\n", deviceName, currentNetwork.Name)
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("=", 70))
	fmt.Fprintf(os.Stderr, "\n💡 Note: The physical device will need to be re-provisioned to rejoin BSN.cloud\n")
}
