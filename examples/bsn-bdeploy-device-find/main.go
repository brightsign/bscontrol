package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/brightsign/bscontrol"
)

func main() {
	var (
		helpFlag   = flag.Bool("help", false, "Display usage information")
		jsonFlag   = flag.Bool("json", false, "Output as JSON")
		serialFlag = flag.String("serial", "", "Device serial number to search for (required)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Search for a B-Deploy device across all networks.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s --serial D7E915000581\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --serial D7E915000581 --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	if *serialFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --serial is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔧 Creating BSN.cloud client...\n")
	}
	client, err := bscontrol.New(bscontrol.WithEnv())
	if err != nil {
		log.Fatalf("❌ Failed to create client: %v", err)
	}

	ctx := context.Background()

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔐 Authenticating with BSN.cloud...\n")
	}
	if err := client.Authenticate(ctx); err != nil {
		log.Fatalf("❌ Authentication failed: %v", err)
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Authentication successful!\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Get all available networks
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📡 Getting available networks...\n")
	}
	networks, err := client.GetNetworks(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to get networks: %v", err)
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Found %d network(s)\n\n", len(networks))
	}

	found := false
	var foundDevice *bscontrol.BDeployDevice
	var foundNetwork string

	// Search each network
	for _, network := range networks {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔍 Searching network: %s (ID: %d)...\n", network.Name, network.ID)
		}

		// Set network context
		if err := client.BDeploy.SetNetworkContext(ctx, network.Name); err != nil {
			if !*jsonFlag {
				fmt.Fprintf(os.Stderr, "   ⚠️  Failed to set network context: %v\n", err)
			}
			continue
		}

		// Try to get device by serial
		deviceResp, err := client.BDeploy.GetDeviceBySerial(ctx, *serialFlag)
		if err == nil && deviceResp.Result.Matched > 0 && len(deviceResp.Result.Players) > 0 {
			foundDevice = &deviceResp.Result.Players[0]
			foundNetwork = network.Name
			found = true
			if !*jsonFlag {
				fmt.Fprintf(os.Stderr, "   ✅ FOUND in this network!\n")
			}
			break
		}

		// Also check device list
		allDevices, err := client.BDeploy.GetAllDevices(ctx)
		if err == nil {
			for _, device := range allDevices.Players {
				if device.Serial == *serialFlag {
					foundDevice = &device
					foundNetwork = network.Name
					found = true
					if !*jsonFlag {
						fmt.Fprintf(os.Stderr, "   ✅ FOUND in device list!\n")
					}
					break
				}
			}
			if found {
				break
			}
		}

		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "   ❌ Not found in this network\n")
		}
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"found":            found,
			"serial":           *serialFlag,
			"searchedNetworks": len(networks),
		}
		if found {
			result["network"] = foundNetwork
			result["device"] = foundDevice
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Normal output
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("=", 70))

	if found {
		fmt.Fprintf(os.Stderr, "✅ DEVICE FOUND!\n")
		fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("=", 70))
		fmt.Fprintf(os.Stderr, "Network:     %s\n", foundNetwork)
		fmt.Fprintf(os.Stderr, "Serial:      %s\n", foundDevice.Serial)
		fmt.Fprintf(os.Stderr, "Device ID:   %s\n", foundDevice.ID)
		fmt.Fprintf(os.Stderr, "Name:        %s\n", foundDevice.Name)
		fmt.Fprintf(os.Stderr, "Model:       %s\n", foundDevice.Model)
		fmt.Fprintf(os.Stderr, "Setup ID:    %s\n", foundDevice.SetupID)
		fmt.Fprintf(os.Stderr, "Setup Name:  %s\n", foundDevice.SetupName)
		if foundDevice.Desc != "" {
			fmt.Fprintf(os.Stderr, "Description: %s\n", foundDevice.Desc)
		}
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "💡 To associate this device with a setup:\n")
		fmt.Fprintf(os.Stderr, "   ./bin/bsn-bdeploy-device-associate --serial %s --setup-name <SETUP_NAME> --network \"%s\"\n", *serialFlag, foundNetwork)
	} else {
		fmt.Fprintf(os.Stderr, "❌ DEVICE NOT FOUND\n")
		fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("=", 70))
		fmt.Fprintf(os.Stderr, "Serial number '%s' was not found in any of your %d network(s).\n", *serialFlag, len(networks))
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Possible reasons:\n")
		fmt.Fprintf(os.Stderr, "  1. The device has never been registered in B-Deploy\n")
		fmt.Fprintf(os.Stderr, "  2. The serial number is incorrect\n")
		fmt.Fprintf(os.Stderr, "  3. You don't have access to the network containing this device\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "💡 To create a new device record:\n")
		fmt.Fprintf(os.Stderr, "   ./bin/bsn-bdeploy-device-associate --serial %s --setup-name <SETUP_NAME> --network <NETWORK> --create\n", *serialFlag)
	}
}
