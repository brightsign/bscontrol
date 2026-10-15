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
		serialFlag  = flag.String("serial", "", "Device serial number (required)")
		groupFlag   = flag.String("group", "", "Group name to assign device to (required)")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
	)

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to change a device's group assignment in BSN.cloud.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (required)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Change device to existing group:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial \"ABC123456789\" --group \"Retail Displays\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Change device to new group (will prompt for confirmation):\n")
		fmt.Fprintf(os.Stderr, "    %s --serial \"ABC123456789\" --group \"NewGroup\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial \"ABC123456789\" --group \"Retail Displays\" --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Validate required parameters
	if *serialFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --serial is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *groupFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --group is required\n\n")
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

	// Get the device
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔍 Looking up device: %s\n", *serialFlag)
	}

	device, err := client.Devices.Get(ctx, *serialFlag)
	if err != nil {
		log.Fatalf("❌ Failed to get device: %v", err)
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Found device: %s (%s)\n", device.Settings.Name, device.Serial)

		// Show current group
		if device.Settings.Group != nil {
			fmt.Fprintf(os.Stderr, "📋 Current group: %s (ID: %d)\n", device.Settings.Group.Name, device.Settings.Group.ID)
		} else {
			fmt.Fprintf(os.Stderr, "📋 Current group: (none)\n")
		}
	}

	// List all groups to check if the target group exists
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📋 Fetching groups...\n")
	}

	groups, err := client.Devices.ListGroups(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to list groups: %v", err)
	}

	// Find the target group
	var targetGroup *bscontrol.Group
	for i := range groups.Items {
		if groups.Items[i].Name == *groupFlag {
			targetGroup = &groups.Items[i]
			break
		}
	}

	// If group doesn't exist, prompt to create it
	if targetGroup == nil {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "\n⚠️  Group '%s' does not exist.\n", *groupFlag)
			fmt.Fprintf(os.Stderr, "Available groups:\n")
			for _, g := range groups.Items {
				fmt.Fprintf(os.Stderr, "  - %s (ID: %d)\n", g.Name, g.ID)
			}
			fmt.Fprintf(os.Stderr, "\n")

			if !confirmCreate(*groupFlag) {
				fmt.Fprintf(os.Stderr, "Operation cancelled.\n")
				return
			}
		} else {
			log.Fatalf("❌ Group '%s' does not exist", *groupFlag)
		}

		// Create the new group
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🆕 Creating new group: %s\n", *groupFlag)
		}
		createdGroup, err := client.Devices.CreateGroup(ctx, *groupFlag)
		if err != nil {
			log.Fatalf("❌ Failed to create group: %v", err)
		}

		targetGroup = createdGroup
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "✅ Group created: %s (ID: %d)\n", targetGroup.Name, targetGroup.ID)
		}
	} else {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "✅ Found group: %s (ID: %d)\n", targetGroup.Name, targetGroup.ID)
		}
	}

	// Check if device is already in the target group
	if device.Settings.Group != nil && device.Settings.Group.ID == targetGroup.ID {
		if *jsonFlag {
			result := map[string]interface{}{
				"changed": false,
				"message": "Device is already in target group",
				"device":  device,
				"group":   targetGroup,
			}
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(result); err != nil {
				log.Fatalf("Failed to encode JSON: %v", err)
			}
			return
		}
		fmt.Fprintf(os.Stderr, "\n✅ Device is already in group '%s'\n", targetGroup.Name)
		return
	}

	// Update the device's group
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔄 Changing device group to: %s\n", targetGroup.Name)
	}

	// Update the device's settings with the new group
	device.Settings.Group = targetGroup

	updatedDevice, err := client.Devices.UpdateBySerial(ctx, *serialFlag, device)
	if err != nil {
		log.Fatalf("❌ Failed to update device: %v", err)
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"changed": true,
			"device":  updatedDevice,
			"group":   targetGroup,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Verify the change
	if updatedDevice.Settings.Group != nil && updatedDevice.Settings.Group.ID == targetGroup.ID {
		fmt.Fprintf(os.Stderr, "✅ Successfully changed device group to: %s\n", updatedDevice.Settings.Group.Name)
	} else {
		fmt.Fprintf(os.Stderr, "⚠️  Device was updated but group change could not be verified\n")
	}
}

func confirmCreate(groupName string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Create new group '%s'? (y/N): ", groupName)

	response, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("❌ Failed to read input: %v", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
