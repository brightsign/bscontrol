package main

import (
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
		idFlag      = flag.Int("id", 0, "Group ID to update")
		nameFlag    = flag.String("name", "", "New group name")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for updating BrightSign group information in BSN.cloud.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Update group name:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 --name \"New Group Name\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Update with verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 --name \"Updated Name\" --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Update with specific network:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 --name \"New Name\" -n \"MyNetwork\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 --name \"Updated Name\" --json\n", os.Args[0])
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

	if *nameFlag == "" {
		fmt.Fprintf(os.Stderr, "❌ Error: Must specify --name with new group name\n\n")
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

	// Step 3: Get current group information
	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔍 Retrieving current group information (ID: %d)\n", *idFlag)
	}

	currentGroup, err := client.Devices.GetGroup(ctx, *idFlag)
	if err != nil {
		log.Fatalf("❌ Failed to get group: %v", err)
	}

	// Display current information
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("═", 70))
		fmt.Fprintf(os.Stderr, "Current Group Information\n")
		fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("═", 70))
		fmt.Fprintf(os.Stderr, "ID:           %d\n", currentGroup.ID)
		fmt.Fprintf(os.Stderr, "Name:         %s\n", currentGroup.Name)
		fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("═", 70))
		fmt.Fprintf(os.Stderr, "\n")

		// Display proposed changes
		fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("═", 70))
		fmt.Fprintf(os.Stderr, "Proposed Changes\n")
		fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("═", 70))
		fmt.Fprintf(os.Stderr, "Name:         %s → %s\n", currentGroup.Name, *nameFlag)
		fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("═", 70))
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Step 4: Update the group
	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📝 Updating group '%s' (ID: %d)...\n", currentGroup.Name, *idFlag)
	}

	// Create updated group object
	updatedGroup := &bscontrol.Group{
		ID:   currentGroup.ID,
		Name: *nameFlag,
		Link: currentGroup.Link,
	}

	result, err := client.Devices.UpdateGroup(ctx, *idFlag, updatedGroup)
	if err != nil {
		log.Fatalf("❌ Failed to update group: %v", err)
	}

	// Output as JSON if requested
	if *jsonFlag {
		output := map[string]interface{}{
			"success":      true,
			"previousName": currentGroup.Name,
			"updatedGroup": result,
			"networkName":  currentNetwork.Name,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Display result
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("═", 70))
	fmt.Fprintf(os.Stderr, "✅ Group Updated Successfully\n")
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("═", 70))
	fmt.Fprintf(os.Stderr, "ID:           %d\n", result.ID)
	fmt.Fprintf(os.Stderr, "Name:         %s\n", result.Name)
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("═", 70))
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "💡 Group '%s' has been updated in network '%s'\n", result.Name, currentNetwork.Name)
}
