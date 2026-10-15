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
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		verboseFlag = flag.Bool("verbose", false, "Show detailed information")
		tokenFlag   = flag.String("token", "", "Device registration token to revoke (required)")
		forceFlag   = flag.Bool("force", false, "Skip confirmation prompt")
		networkFlag *string
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Revoke a device registration token on BSN.cloud.\n\n")
		fmt.Fprintf(os.Stderr, "Once revoked, players can no longer use the token to register with\n")
		fmt.Fprintf(os.Stderr, "BSN.cloud. Tokens with 'cert' scope are shared by every device\n")
		fmt.Fprintf(os.Stderr, "provisioned with them, so revoking one breaks registration for all\n")
		fmt.Fprintf(os.Stderr, "of them. This cannot be undone.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Revoke a token (prompts for confirmation):\n")
		fmt.Fprintf(os.Stderr, "    %s --token \"<token>\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Revoke without confirmation:\n")
		fmt.Fprintf(os.Stderr, "    %s --token \"<token>\" --force\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Revoke a token from a saved regtoken.json:\n")
		fmt.Fprintf(os.Stderr, "    %s --token \"$(jq -r .token regtoken.json)\" --force\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --token \"<token>\" --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	if *tokenFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --token is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}
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

	// Show what is about to be destroyed, then confirm. JSON mode is
	// non-interactive, so it skips both the lookup and the prompt.
	interactive := !*forceFlag && !*jsonFlag
	if interactive {
		fmt.Fprintf(os.Stderr, "🔎 Looking up token...\n")

		details, err := client.Provisioning.ValidateDeviceToken(ctx, *tokenFlag)
		if err != nil {
			log.Fatalf("❌ Failed to validate token before revoking: %v", err)
		}

		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "📋 Token Details:\n")
		fmt.Fprintf(os.Stderr, "  Network:      %s\n", networkName)
		fmt.Fprintf(os.Stderr, "  Scope:        %s\n", details.Scope)
		fmt.Fprintf(os.Stderr, "  Valid From:   %s\n", details.ValidFrom)
		fmt.Fprintf(os.Stderr, "  Valid To:     %s\n", details.ValidTo)
		fmt.Fprintf(os.Stderr, "  Token:        %s\n", previewToken(*tokenFlag, *verboseFlag))
		fmt.Fprintf(os.Stderr, "\n")

		if details.Scope == "cert" {
			fmt.Fprintf(os.Stderr, "⚠️  This token has 'cert' scope - it may be shared by many players.\n")
			fmt.Fprintf(os.Stderr, "   Revoking it breaks registration for every one of them.\n\n")
		}

		if !confirmRevocation() {
			fmt.Fprintf(os.Stderr, "Aborted. Token was not revoked.\n")
			os.Exit(1)
		}
	}

	// Revoke the token
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🗑️  Revoking device registration token...\n")
	}

	if err := client.Provisioning.RevokeRegistrationToken(ctx, *tokenFlag); err != nil {
		log.Fatalf("❌ Failed to revoke device registration token: %v", err)
	}

	// Output results
	if *jsonFlag {
		result := map[string]interface{}{
			"revoked": true,
			"network": networkName,
			"token":   *tokenFlag,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "✅ Token revoked successfully!\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "  Network:      %s\n", networkName)
	fmt.Fprintf(os.Stderr, "  Token:        %s\n", previewToken(*tokenFlag, *verboseFlag))
	fmt.Fprintf(os.Stderr, "\n")

	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "🔍 API Details:\n")
		fmt.Fprintf(os.Stderr, "  Endpoint:     DELETE https://api.bsn.cloud/2022/06/REST/Provisioning/Setups/Tokens/{token}/\n")
		fmt.Fprintf(os.Stderr, "  Scope:        bsn.api.main.devices.setups.token.delete\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	fmt.Fprintf(os.Stderr, "📱 Next Steps:\n")
	fmt.Fprintf(os.Stderr, "  1. Generate a replacement token with main-get-regtoken\n")
	fmt.Fprintf(os.Stderr, "  2. Update any B-Deploy setup records that embedded the revoked token\n")
	fmt.Fprintf(os.Stderr, "  3. Re-provision affected players so they pick up the new token\n")
	fmt.Fprintf(os.Stderr, "\n")
}

// previewToken abbreviates long tokens so terminal output stays readable.
func previewToken(token string, verbose bool) string {
	if verbose || len(token) <= 60 {
		return token
	}
	return fmt.Sprintf("%s...%s (use --verbose for full token)", token[:30], token[len(token)-20:])
}

func confirmRevocation() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stderr, "Are you sure you want to revoke this token? (y/N): ")

	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("❌ Failed to read input: %v", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
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
