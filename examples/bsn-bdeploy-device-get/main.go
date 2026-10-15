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

func main() {
	var (
		helpFlag    = flag.Bool("help", false, "Display usage information")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
		verboseFlag = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		serialFlag  = flag.String("serial", "", "Device serial number to lookup")
		networkFlag *string
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to get B-Deploy device setup record by serial number from BSN.cloud.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get device by serial:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABCD00000001 --network \"My Network\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get device with verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABCD00000001 --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABCD00000001 --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// Check for required serial parameter
	if *serialFlag == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: --serial parameter is required\n")
		fmt.Fprintf(os.Stderr, "Use --help for usage information\n")
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
			log.Fatalf("❌ Authentication error: %v", err)
		}
		log.Fatalf("❌ Failed to authenticate: %v", err)
	}

	// Get network name from command line or environment
	networkName := *networkFlag
	if networkName == "" {
		networkName = os.Getenv("BS_NETWORK")
	}

	// Set network context if provided
	if networkName != "" {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🌐 Setting network context to: %s\n", networkName)
		}
		err = client.BDeploy.SetNetworkContext(ctx, networkName)
		if err != nil {
			log.Fatalf("❌ Failed to set network context: %v", err)
		}
	}

	// Get the device by serial number
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔍 Looking up device with serial: %s\n", *serialFlag)
	}

	response, err := client.BDeploy.GetDeviceBySerial(ctx, *serialFlag)
	if err != nil {
		log.Fatalf("❌ Failed to get device: %v", err)
	}

	// Check for API error
	if response.Error != nil {
		log.Fatalf("❌ API Error: %v", response.Error)
	}

	// Check if any devices were found
	if len(response.Result.Players) == 0 {
		if *jsonFlag {
			result := map[string]interface{}{
				"found":  false,
				"serial": *serialFlag,
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(result); err != nil {
				log.Fatalf("Failed to encode JSON: %v", err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "❌ No device found with serial: %s\n", *serialFlag)
		}
		os.Exit(1)
	}

	// Get the device information
	device := response.Result.Players[0]

	// Output as JSON if requested
	if *jsonFlag {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(device); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Display the device information
	fmt.Fprintf(os.Stderr, "\n📱 Device Information:\n")
	fmt.Fprintf(os.Stderr, "   Username:    %s\n", device.Username)
	fmt.Fprintf(os.Stderr, "   Name:        %s\n", device.Name)
	fmt.Fprintf(os.Stderr, "   Model:       %s\n", device.Model)
	fmt.Fprintf(os.Stderr, "   Setup Name:  %s\n", device.SetupName)

	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "\n🔍 Detailed Information:\n")
		fmt.Fprintf(os.Stderr, "   ID:          %s\n", device.ID)
		fmt.Fprintf(os.Stderr, "   Serial:      %s\n", device.Serial)
		fmt.Fprintf(os.Stderr, "   Network:     %s\n", device.NetworkName)
		fmt.Fprintf(os.Stderr, "   Client:      %s\n", device.Client)
		fmt.Fprintf(os.Stderr, "   Description: %s\n", device.Desc)
		fmt.Fprintf(os.Stderr, "   Created:     %s\n", device.CreatedAt.Format(time.RFC3339))
		fmt.Fprintf(os.Stderr, "   Updated:     %s\n", device.UpdatedAt.Format(time.RFC3339))
		fmt.Fprintf(os.Stderr, "   Version:     %d\n", device.Version)
	}

	fmt.Fprintf(os.Stderr, "\n✅ Device lookup completed successfully!\n")
}
