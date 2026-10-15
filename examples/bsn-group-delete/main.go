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
		idFlag      = flag.Int("id", 0, "Delete group with ID")
		forceFlag   = flag.Bool("force", false, "Skip confirmation prompt")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")
	flag.BoolVar(forceFlag, "y", false, "Skip confirmation prompt [alias for --force]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for deleting BrightSign groups from BSN.cloud.\n\n")
		fmt.Fprintf(os.Stderr, "⚠️  WARNING: This permanently removes groups from your network.\n")
		fmt.Fprintf(os.Stderr, "           Devices in the group are NOT deleted but will be ungrouped.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Delete group with confirmation:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Delete without confirmation:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 --force\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Delete with verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Delete with specific network:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 -n \"MyNetwork\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 --force --json\n", os.Args[0])
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

	// Step 3: Get group information
	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔍 Looking up group with ID: %d\n", *idFlag)
	}

	group, err := client.Devices.GetGroup(ctx, *idFlag)
	if err != nil {
		log.Fatalf("❌ Failed to find group: %v", err)
	}

	// Display group information
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\n%s\n", strings.Repeat("═", 70))
		fmt.Fprintf(os.Stderr, "Group Information\n")
		fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("═", 70))
		fmt.Fprintf(os.Stderr, "ID:           %d\n", group.ID)
		fmt.Fprintf(os.Stderr, "Name:         %s\n", group.Name)
		if group.Link != "" {
			fmt.Fprintf(os.Stderr, "Link:         %s\n", group.Link)
		}
		fmt.Fprintf(os.Stderr, "%s\n\n", strings.Repeat("═", 70))
	}

	// Step 4: Confirmation prompt
	if !*forceFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "⚠️  WARNING: This will permanently delete group '%s' from BSN.cloud!\n", group.Name)
		fmt.Fprintf(os.Stderr, "           Group ID: %d\n", group.ID)
		fmt.Fprintf(os.Stderr, "           Devices in this group will NOT be deleted but will be ungrouped.\n\n")
		fmt.Fprintf(os.Stderr, "Are you sure you want to delete this group? [yes/no]: ")

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

	// Step 5: Delete group
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🗑️  Deleting group '%s' (ID: %d)...\n", group.Name, group.ID)
	}

	err = client.Devices.DeleteGroup(ctx, *idFlag)
	if err != nil {
		log.Fatalf("❌ Failed to delete group: %v", err)
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success":     true,
			"groupName":   group.Name,
			"groupID":     group.ID,
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
	fmt.Fprintf(os.Stderr, "✅ Group deleted successfully\n")
	fmt.Fprintf(os.Stderr, "\n%s\n", strings.Repeat("═", 70))
	fmt.Fprintf(os.Stderr, "Group '%s' has been removed from network '%s'\n", group.Name, currentNetwork.Name)
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("═", 70))
	fmt.Fprintf(os.Stderr, "\n💡 Note: Devices that were in this group are still in your network but are now ungrouped\n")
}
