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
		verboseFlag = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag = flag.String("network", "", "Network name (env: BS_NETWORK)")
	)

	flag.StringVar(networkFlag, "n", "", "Network name (env: BS_NETWORK) [alias for --network]")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate a BSN.cloud device registration token and output the\n")
		fmt.Fprintf(os.Stderr, "bsnDeviceRegistrationTokenEntity JSON object to stdout.\n\n")
		fmt.Fprintf(os.Stderr, "The output is ready to paste into a setup JSON file for use with\n")
		fmt.Fprintf(os.Stderr, "AddSetupRecordRaw or bdeploy-add-setup-raw.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID   BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET      BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK     BSN.cloud network name\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Generate token entity JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --network \"Production\"\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Save to file:\n")
		fmt.Fprintf(os.Stderr, "    %s --network \"Production\" > token.json\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Inject into setup JSON with jq:\n")
		fmt.Fprintf(os.Stderr, "    jq --slurpfile tok token.json '.bsnDeviceRegistrationTokenEntity = $tok[0]' setup.json\n")
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Resolve network from flag or environment
	networkName := *networkFlag
	if networkName == "" {
		networkName = os.Getenv("BS_NETWORK")
	}
	if networkName == "" {
		fmt.Fprintf(os.Stderr, "Error: network is required (use --network or set BS_NETWORK)\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}

	fmt.Fprintf(os.Stderr, "Creating BSN.cloud client...\n")
	client, err := bscontrol.New(opts...)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Authenticate
	fmt.Fprintf(os.Stderr, "Authenticating with BSN.cloud...\n")
	if err := client.Authenticate(ctx); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Authentication successful.\n")

	// Set network context
	fmt.Fprintf(os.Stderr, "Setting network context to: %s\n", networkName)
	if err := client.BDeploy.SetNetworkContext(ctx, networkName); err != nil {
		log.Fatalf("Failed to set network context: %v", err)
	}

	// Generate device registration token
	fmt.Fprintf(os.Stderr, "Generating device registration token...\n")
	deviceToken, err := client.Provisioning.GenerateDeviceToken(ctx)
	if err != nil {
		log.Fatalf("Failed to generate device token: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Token generated successfully.\n")

	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "  Scope:      %s\n", deviceToken.Scope)
		fmt.Fprintf(os.Stderr, "  Valid from: %s\n", deviceToken.ValidFrom)
		fmt.Fprintf(os.Stderr, "  Valid to:   %s\n", deviceToken.ValidTo)
	}

	// Output the token entity JSON to stdout
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(deviceToken); err != nil {
		log.Fatalf("Failed to encode JSON: %v", err)
	}
}
