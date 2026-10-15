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
		networkFlag *string
		idFlag      = flag.Int("id", 0, "Group ID to retrieve")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for retrieving BrightSign group information from BSN.cloud.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get group information:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get group with JSON output:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 --json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get group with verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get group with specific network:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 -n \"MyNetwork\"\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Validate input
	if *idFlag <= 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: Must specify a valid --id (greater than 0)\n\n")
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
	if *verboseFlag {
		fmt.Println("🔐 Authenticating with BSN.cloud...")
	}
	if err := client.Authenticate(ctx); err != nil {
		if bscontrol.IsAuthenticationError(err) {
			log.Fatalf("❌ Authentication failed: %v", err)
		}
		log.Fatalf("❌ Authentication error: %v", err)
	}
	if *verboseFlag {
		fmt.Println("✅ Authentication successful")
	}

	// Step 2: Set network context
	if err := client.EnsureReady(ctx); err != nil {
		log.Fatalf("❌ Failed to set network context: %v", err)
	}

	currentNetwork, err := client.GetCurrentNetwork(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to get current network: %v", err)
	}

	if *verboseFlag {
		fmt.Printf("📡 Using network: %s\n", currentNetwork.Name)
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Step 3: Get group information
	if *verboseFlag {
		fmt.Printf("🔍 Retrieving group with ID: %d\n", *idFlag)
	}

	group, err := client.Devices.GetGroup(ctx, *idFlag)
	if err != nil {
		log.Fatalf("❌ Failed to get group: %v", err)
	}

	// Output
	if *jsonFlag {
		// JSON output
		jsonData, err := json.MarshalIndent(group, "", "  ")
		if err != nil {
			log.Fatalf("❌ Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		// Human-readable output
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Println("═══════════════════════════════════════════════════════════════════")
		fmt.Println("Group Information")
		fmt.Println("═══════════════════════════════════════════════════════════════════")
		fmt.Printf("ID:           %d\n", group.ID)
		fmt.Printf("Name:         %s\n", group.Name)
		if group.Link != "" {
			fmt.Printf("Link:         %s\n", group.Link)
		}
		fmt.Println("═══════════════════════════════════════════════════════════════════")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Printf("✅ Group '%s' retrieved from network '%s'\n", group.Name, currentNetwork.Name)
	}
}
