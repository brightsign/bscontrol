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
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag *string
		verboseFlag = flag.Bool("verbose", false, "Show detailed information")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to generate a device registration token for BSN.cloud.\n\n")
		fmt.Fprintf(os.Stderr, "Device registration tokens allow BrightSign players to register themselves\n")
		fmt.Fprintf(os.Stderr, "with BSN.cloud. The token has 'cert' scope and is valid for 2 years by default.\n")
		fmt.Fprintf(os.Stderr, "Multiple devices can use the same token.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Generate a registration token:\n")
		fmt.Fprintf(os.Stderr, "    %s --network \"My Network\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Generate token with verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --network \"My Network\" --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON only:\n")
		fmt.Fprintf(os.Stderr, "    %s --network \"My Network\" --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
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
			log.Fatalf("❌ Authentication failed: %v", err)
		}
		log.Fatalf("❌ Authentication error: %v", err)
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Authentication successful!\n")
	}

	// Determine network to use
	networkName := getNetworkName(*networkFlag, client, ctx, *jsonFlag)

	// Set network context
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📡 Setting network context to: %s\n", networkName)
	}
	if err := client.BDeploy.SetNetworkContext(ctx, networkName); err != nil {
		log.Fatalf("❌ Failed to set network context: %v", err)
	}
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Network context set successfully!\n")
	}

	// Generate device registration token
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔑 Generating device registration token...\n")
	}

	deviceToken, err := client.Provisioning.GenerateDeviceToken(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to generate device registration token: %v", err)
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Device registration token generated successfully!\n")
	}

	// Output results
	if *jsonFlag {
		// JSON output only
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(deviceToken); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Human-friendly output
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(os.Stderr, "🎫 Device Registration Token\n")
	fmt.Fprintf(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Fprintf(os.Stderr, "\n")

	// Display token details
	fmt.Fprintf(os.Stderr, "📋 Token Details:\n")
	fmt.Fprintf(os.Stderr, "  Network:      %s\n", networkName)
	fmt.Fprintf(os.Stderr, "  Scope:        %s\n", deviceToken.Scope)
	fmt.Fprintf(os.Stderr, "  Valid From:   %s\n", deviceToken.ValidFrom)
	fmt.Fprintf(os.Stderr, "  Valid To:     %s\n", deviceToken.ValidTo)
	fmt.Fprintf(os.Stderr, "\n")

	// Display the token
	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "🔐 Full Token:\n")
		fmt.Fprintf(os.Stderr, "%s\n", deviceToken.Token)
	} else {
		tokenLength := len(deviceToken.Token)
		if tokenLength > 60 {
			fmt.Fprintf(os.Stderr, "🔐 Token Preview:\n")
			fmt.Fprintf(os.Stderr, "  %s...%s\n", deviceToken.Token[:30], deviceToken.Token[tokenLength-20:])
			fmt.Fprintf(os.Stderr, "  (Use --verbose to see full token)\n")
		} else {
			fmt.Fprintf(os.Stderr, "🔐 Token:\n")
			fmt.Fprintf(os.Stderr, "  %s\n", deviceToken.Token)
		}
	}
	fmt.Fprintf(os.Stderr, "\n")

	// Usage information
	fmt.Fprintf(os.Stderr, "📖 Usage:\n")
	fmt.Fprintf(os.Stderr, "  This token can be embedded in B-Deploy setup records to enable\n")
	fmt.Fprintf(os.Stderr, "  player registration during provisioning.\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  The token allows multiple devices to register using the same token.\n")
	fmt.Fprintf(os.Stderr, "\n")

	// API endpoint information
	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "🔍 API Details:\n")
		fmt.Fprintf(os.Stderr, "  Endpoint:     POST https://api.bsn.cloud/2020/10/REST/Provisioning/Setups/Tokens/\n")
		fmt.Fprintf(os.Stderr, "  Scope:        bsn.api.main.devices.setups.token.create\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	// curl example
	fmt.Fprintf(os.Stderr, "📋 curl Command to Generate Token:\n")
	accessToken, err := client.GetAccessToken()
	if err == nil && accessToken != "" {
		fmt.Fprintf(os.Stderr, "# First, get an access token (replace with your credentials)\n")
		config := client.Config()
		fmt.Fprintf(os.Stderr, "curl -X POST '%s' \\\n", config.TokenEndpoint)
		fmt.Fprintf(os.Stderr, "  -H 'Content-Type: application/x-www-form-urlencoded' \\\n")
		fmt.Fprintf(os.Stderr, "  -d 'grant_type=client_credentials' \\\n")
		fmt.Fprintf(os.Stderr, "  -u \"$BS_CLIENT_ID:$BS_SECRET\"\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "# Then generate a device registration token\n")
		fmt.Fprintf(os.Stderr, "curl -X POST 'https://api.bsn.cloud/2020/10/REST/Provisioning/Setups/Tokens/' \\\n")
		fmt.Fprintf(os.Stderr, "  -H 'Authorization: Bearer <access_token>' \\\n")
		fmt.Fprintf(os.Stderr, "  -H 'Accept: application/json'\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Next steps
	fmt.Fprintf(os.Stderr, "📱 Next Steps:\n")
	fmt.Fprintf(os.Stderr, "  1. Copy the token and use it in your B-Deploy setup configuration\n")
	fmt.Fprintf(os.Stderr, "  2. Players provisioned with this setup will use the token to register\n")
	fmt.Fprintf(os.Stderr, "  3. See bdeploy-add-setup example for creating complete setup records\n")
	fmt.Fprintf(os.Stderr, "\n")

	// Save token to file option
	fmt.Fprintf(os.Stderr, "💡 Tip: Use --json flag to get machine-readable output:\n")
	fmt.Fprintf(os.Stderr, "  %s --network \"%s\" --json > regtoken.json\n", os.Args[0], networkName)
	fmt.Fprintf(os.Stderr, "\n")
}

func getNetworkName(requestedNetwork string, client *bscontrol.Client, ctx context.Context, jsonOutput bool) string {
	// If network was specified via flag, use it
	if requestedNetwork != "" {
		return requestedNetwork
	}

	// Check if network is already set in client
	if client.IsNetworkSet() {
		if current, err := client.GetCurrentNetwork(ctx); err == nil {
			return current.Name
		}
	}

	// Check environment variable
	if envNetwork := os.Getenv("BS_NETWORK"); envNetwork != "" {
		return envNetwork
	}

	// Need to select a network
	if !jsonOutput {
		fmt.Fprintf(os.Stderr, "📡 Getting available networks...\n")
	}

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
		if !jsonOutput {
			fmt.Fprintf(os.Stderr, "✅ Using only available network: %s\n", networkName)
		}
		return networkName
	}

	// Multiple networks - need user to specify
	if !jsonOutput {
		fmt.Fprintf(os.Stderr, "❌ Multiple networks available. Please specify --network or set BS_NETWORK:\n")
		for i, network := range networks {
			fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
		}
	}
	os.Exit(1)
	return ""
}
