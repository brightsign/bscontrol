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
	"github.com/brightsign/bscontrol/examples/internal/setuptemplate"
)

func main() {
	var (
		helpFlag       = flag.Bool("help", false, "Display usage information")
		jsonFlag       = flag.Bool("json", false, "Output as JSON")
		serialFlag     = flag.String("serial", "", "Device serial number (required)")
		setupIDFlag    = flag.String("setup-id", "", "Setup ID to associate with device (required unless --dissociate or --setup-name)")
		setupNameFlag  = flag.String("setup-name", "", "Setup package name to associate with device (alternative to --setup-id, env: BS_PACKAGE_NAME)")
		dissociateFlag = flag.Bool("dissociate", false, "Remove setup association (sets setupId to null)")
		networkFlag    = flag.String("network", "", "Network name (defaults to BS_NETWORK env var)")
		usernameFlag   = flag.String("username", "", "BSN.cloud username (defaults to BS_CLIENT_ID env var)")
		descFlag       = flag.String("description", "", "Device description (optional, env: BS_DEVICE_DESCRIPTION)")
		nameFlag       = flag.String("name", "", "Device name (optional, env: BS_DEVICE_NAME)")
		createFlag     = flag.Bool("create", false, "Create device if it doesn't exist")
		verboseFlag    = flag.Bool("verbose", false, "Show detailed information")
		debugFlag      = flag.Bool("debug", false, "Show raw API responses for debugging")
		timeoutFlag    = flag.Int("timeout", 30, "Request timeout in seconds")
	)

	// Register flag aliases
	flag.StringVar(setupNameFlag, "package-name", "", "Alias for --setup-name (env: BS_PACKAGE_NAME)")
	flag.StringVar(nameFlag, "device-name", "", "Alias for --name (env: BS_DEVICE_NAME)")
	flag.StringVar(descFlag, "device-description", "", "Alias for --description (env: BS_DEVICE_DESCRIPTION)")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Associates a BrightSign device with a B-Deploy setup package, or removes an association.\n\n")
		fmt.Fprintf(os.Stderr, "This tool links a device serial number to a setup ID. When the physical\n")
		fmt.Fprintf(os.Stderr, "device boots and contacts B-Deploy, it will receive the associated setup.\n")
		fmt.Fprintf(os.Stderr, "Use --dissociate to remove the setup association while keeping the device record.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (required if not specified via --network)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Associate existing device with setup by ID:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABC123456789 --setup-id 69024f69c178ba94c487c24d\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Associate existing device with setup by package name:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABC123456789 --setup-name \"my-retail-setup\"\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Create and associate new device:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABC123456789 --setup-name \"my-retail-setup\" --create --description \"Lobby Display\"\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Dissociate device from setup (remove association):\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABC123456789 --dissociate\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Specify network explicitly:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABC123456789 --setup-name \"my-retail-setup\" --network \"Production\"\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial ABC123456789 --setup-id 69024f69c178ba94c487c24d --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Resolve flags from environment variables
	if *setupNameFlag == "" {
		*setupNameFlag = setuptemplate.ResolveVar(*setupNameFlag, setuptemplate.EnvPackageName)
	}
	if *nameFlag == "" {
		*nameFlag = setuptemplate.ResolveVar(*nameFlag, setuptemplate.EnvDeviceName)
	}
	if *descFlag == "" {
		*descFlag = setuptemplate.ResolveVar(*descFlag, setuptemplate.EnvDeviceDescription)
	}

	// Validate required parameters
	if *serialFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --serial is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate setup-id or setup-name requirement based on mode
	if !*dissociateFlag && *setupIDFlag == "" && *setupNameFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: Either --setup-id or --setup-name is required (unless using --dissociate)\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate conflicting flags
	if *setupIDFlag != "" && *setupNameFlag != "" {
		fmt.Fprintf(os.Stderr, "Error: Cannot use both --setup-id and --setup-name\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *dissociateFlag && (*setupIDFlag != "" || *setupNameFlag != "") {
		fmt.Fprintf(os.Stderr, "Error: Cannot use --dissociate with --setup-id or --setup-name\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *dissociateFlag && *createFlag {
		fmt.Fprintf(os.Stderr, "Error: Cannot use --create with --dissociate (device must already exist)\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Get network name from flag or environment
	networkName := *networkFlag
	if networkName == "" {
		networkName = os.Getenv("BS_NETWORK")
		if networkName == "" {
			fmt.Fprintf(os.Stderr, "Error: Network name must be specified via --network or BS_NETWORK\n\n")
			flag.Usage()
			os.Exit(1)
		}
	}

	// Get username from flag or environment
	username := *usernameFlag
	if username == "" {
		username = os.Getenv("BS_CLIENT_ID")
		if username == "" {
			fmt.Fprintf(os.Stderr, "Error: Username must be specified via --username or BS_CLIENT_ID\n\n")
			flag.Usage()
			os.Exit(1)
		}
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

	// Set network context for B-Deploy
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📡 Setting network context to: %s\n", networkName)
	}
	if err := client.BDeploy.SetNetworkContext(ctx, networkName); err != nil {
		log.Fatalf("❌ Failed to set network context: %v", err)
	}
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Network context set!\n")
	}

	// Resolve setup ID and setup name (package name).
	// The B-Deploy device API needs both setupId and setupName to properly
	// index the association for later queries.
	var setupID string
	var setupName string
	if *setupNameFlag != "" {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔍 Looking up setup with package name: %s\n", *setupNameFlag)
		}

		// Query setup records by package name
		records, err := client.BDeploy.GetSetupRecords(ctx,
			bscontrol.WithNetworkName(networkName),
			bscontrol.WithPackageName(*setupNameFlag),
		)
		if err != nil {
			log.Fatalf("❌ Failed to search for setup records: %v", err)
		}

		// Filter for exact matches only (API does substring match)
		var exactMatches []bscontrol.BDeployRecord
		for _, record := range records.Items {
			if record.PackageName == *setupNameFlag {
				exactMatches = append(exactMatches, record)
			}
		}

		if len(exactMatches) == 0 {
			log.Fatalf("❌ No setup record found with exact package name '%s' on network '%s'", *setupNameFlag, networkName)
		}

		if len(exactMatches) > 1 {
			fmt.Printf("⚠️  Found %d setup records with package name '%s':\n", len(exactMatches), *setupNameFlag)
			for i, record := range exactMatches {
				fmt.Printf("  %d. ID: %s, Package: %s, Type: %s\n", i+1, record.ID, record.PackageName, record.SetupType)
			}
			log.Fatalf("❌ Multiple setups found. Please use --setup-id to specify which one")
		}

		// Found exactly one exact match
		setupID = exactMatches[0].ID
		setupName = exactMatches[0].PackageName
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "✅ Found setup: ID=%s, Package=%s\n", setupID, setupName)
		}
	} else {
		setupID = *setupIDFlag

		// Look up the package name for this setup ID so we can send setupName
		if !*dissociateFlag && setupID != "" {
			record, err := client.BDeploy.GetSetupRecord(ctx, setupID)
			if err == nil {
				setupName = record.BDeploy.PackageName
			}
		}
	}

	// Check if device exists - try serial lookup first
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔍 Looking for device with serial: %s\n", *serialFlag)
	}
	deviceResponse, err := client.BDeploy.GetDeviceBySerial(ctx, *serialFlag)

	var deviceID string
	var deviceExists bool

	if *verboseFlag && err != nil {
		fmt.Fprintf(os.Stderr, "   Debug: Serial lookup error: %v\n", err)
	}
	if *verboseFlag && deviceResponse != nil {
		fmt.Fprintf(os.Stderr, "   Debug: Serial lookup response - Total: %d, Matched: %d, Players: %d\n",
			deviceResponse.Result.Total, deviceResponse.Result.Matched, len(deviceResponse.Result.Players))
	}

	if err == nil && deviceResponse.Result.Matched > 0 && len(deviceResponse.Result.Players) > 0 {
		// Found via serial lookup
		device := deviceResponse.Result.Players[0]
		deviceID = device.ID
		deviceExists = true
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "✅ Found existing device: %s\n", device.Serial)
		}
		if device.SetupID != "" {
			fmt.Fprintf(os.Stderr, "   Current setup ID: %s\n", device.SetupID)
		} else {
			fmt.Fprintf(os.Stderr, "   No setup currently associated")
		}
	} else {
		// Serial lookup failed or returned no results - try listing all devices
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "   Device not found by serial, checking device list...\n")
		}
		allDevices, listErr := client.BDeploy.GetAllDevices(ctx)

		if *verboseFlag && listErr != nil {
			fmt.Printf("   Debug: Device list error: %v\n", listErr)
		}
		if *verboseFlag && allDevices != nil {
			fmt.Printf("   Debug: Device list response - Total: %d, Matched: %d, Players: %d\n",
				allDevices.Total, allDevices.Matched, len(allDevices.Players))
		}

		if listErr == nil {
			// Search through all devices for matching serial
			for _, dev := range allDevices.Players {
				if *verboseFlag {
					fmt.Fprintf(os.Stderr, "   Debug: Checking device %s (ID: %s)\n", dev.Serial, dev.ID)
				}
				if dev.Serial == *serialFlag {
					deviceID = dev.ID
					deviceExists = true
					if !*jsonFlag {
						fmt.Fprintf(os.Stderr, "✅ Found existing device in list: %s\n", dev.Serial)
					}
					if dev.SetupID != "" {
						fmt.Fprintf(os.Stderr, "   Current setup ID: %s\n", dev.SetupID)
					} else {
						fmt.Fprintf(os.Stderr, "   No setup currently associated")
					}
					break
				}
			}
		}

		if !deviceExists {
			if !*createFlag {
				log.Fatalf("❌ Device not found. Use --create to create it")
			}
			if !*jsonFlag {
				fmt.Fprintf(os.Stderr, "⚠️  Device not found - will create new device record\n")
			}
		}
	}

	// Create device if needed (only in associate mode)
	if !deviceExists {
		if *dissociateFlag {
			log.Fatalf("❌ Device not found. Cannot dissociate a non-existent device")
		}

		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "📝 Creating device record...\n")
		}

		deviceName := *nameFlag
		if deviceName == "" {
			deviceName = *serialFlag // Default to serial if no name provided
		}

		description := *descFlag
		if description == "" {
			description = fmt.Sprintf("Device %s", *serialFlag)
		}

		createRequest := &bscontrol.BDeployDeviceRequest{
			Username:    username,
			Serial:      *serialFlag,
			Name:        deviceName,
			NetworkName: networkName,
			Desc:        description,
			SetupID:     setupID,
			SetupName:   setupName,
		}

		deviceID, err = client.BDeploy.CreateDevice(ctx, createRequest)
		if err != nil {
			log.Fatalf("❌ Failed to create device: %v", err)
		}
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "✅ Device created with ID: %s\n", deviceID)
		}
	}

	// If we just created the device with setupID already set, skip the update
	// unless we're dissociating. The B-Deploy API may not persist setupId
	// from PUT updates — only from the initial POST creation.
	var updatedDevice *bscontrol.BDeployDevice
	if !deviceExists && !*dissociateFlag && setupID != "" {
		// Device was just created with setupID — construct the response
		deviceName := *nameFlag
		if deviceName == "" {
			deviceName = *serialFlag
		}
		description := *descFlag
		if description == "" {
			description = fmt.Sprintf("Device %s", *serialFlag)
		}
		updatedDevice = &bscontrol.BDeployDevice{
			ID:          deviceID,
			Serial:      *serialFlag,
			Name:        deviceName,
			NetworkName: networkName,
			Desc:        description,
			SetupID:     setupID,
			Username:    username,
		}
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔗 Device created with setup ID: %s\n", setupID)
		}
	} else {
		// Existing device — update it
		if *dissociateFlag {
			if !*jsonFlag {
				fmt.Fprintf(os.Stderr, "🔓 Removing setup association from device...\n")
			}
		} else {
			if !*jsonFlag {
				fmt.Fprintf(os.Stderr, "🔗 Associating device with setup ID: %s\n", setupID)
			}
		}

		deviceName := *nameFlag
		if deviceName == "" {
			deviceName = *serialFlag
		}

		description := *descFlag
		if description == "" {
			description = fmt.Sprintf("Device %s", *serialFlag)
		}

		// Prepare update request
		updateRequest := &bscontrol.BDeployDeviceRequest{
			Username:    username,
			Serial:      *serialFlag,
			Name:        deviceName,
			NetworkName: networkName,
			Desc:        description,
		}

		// Set setupID and setupName based on mode
		if *dissociateFlag {
			updateRequest.SetupID = ""
			updateRequest.SetupName = ""
		} else {
			updateRequest.SetupID = setupID
			updateRequest.SetupName = setupName
		}

		var err error
		updatedDevice, err = client.BDeploy.UpdateDevice(ctx, deviceID, updateRequest)
		if err != nil {
			if *dissociateFlag {
				log.Fatalf("❌ Failed to remove setup association: %v", err)
			} else {
				log.Fatalf("❌ Failed to associate device with setup: %v", err)
			}
		}
	}

	// Debug: Show what UpdateDevice returned
	if *debugFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "\n🔍 DEBUG: UpdateDevice response:\n")
		debugJSON, _ := json.MarshalIndent(updatedDevice, "   ", "  ")
		fmt.Fprintf(os.Stderr, "   %s\n\n", string(debugJSON))
		fmt.Fprintf(os.Stderr, "   SetupID field value: '%s'\n", updatedDevice.SetupID)
		fmt.Fprintf(os.Stderr, "   SetupID is empty: %v\n\n", updatedDevice.SetupID == "")
	}

	// Output as JSON if requested
	if *jsonFlag {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(updatedDevice); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Display results based on mode
	if *dissociateFlag {
		fmt.Fprintf(os.Stderr, "✅ Setup association removed successfully!")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "📋 Device Details:")
		fmt.Fprintf(os.Stderr, "  Serial:     %s\n", updatedDevice.Serial)
		fmt.Fprintf(os.Stderr, "  Device ID:  %s\n", updatedDevice.ID)
		fmt.Fprintf(os.Stderr, "  Setup ID:   <none>\n")
		fmt.Fprintf(os.Stderr, "  Network:    %s\n", updatedDevice.NetworkName)
		if updatedDevice.Name != "" {
			fmt.Fprintf(os.Stderr, "  Name:       %s\n", updatedDevice.Name)
		}
		if updatedDevice.Desc != "" {
			fmt.Fprintf(os.Stderr, "  Description: %s\n", updatedDevice.Desc)
		}

		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "🎉 Dissociation complete!")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "ℹ️  Note:")
		fmt.Fprintf(os.Stderr, "  - Device record remains in B-Deploy")
		fmt.Fprintf(os.Stderr, "  - Device will not auto-provision on next boot")
		fmt.Fprintf(os.Stderr, "  - To re-associate, run with --setup-id")
		fmt.Fprintf(os.Stderr, "  - To delete device entirely, use the B-Deploy device deletion API")
	} else {
		fmt.Fprintf(os.Stderr, "✅ Device successfully associated with setup!")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "📋 Device Details:")
		fmt.Fprintf(os.Stderr, "  Serial:     %s\n", updatedDevice.Serial)
		fmt.Fprintf(os.Stderr, "  Device ID:  %s\n", updatedDevice.ID)
		fmt.Fprintf(os.Stderr, "  Setup ID:   %s\n", updatedDevice.SetupID)
		fmt.Fprintf(os.Stderr, "  Network:    %s\n", updatedDevice.NetworkName)
		if updatedDevice.Name != "" {
			fmt.Fprintf(os.Stderr, "  Name:       %s\n", updatedDevice.Name)
		}
		if updatedDevice.Desc != "" {
			fmt.Fprintf(os.Stderr, "  Description: %s\n", updatedDevice.Desc)
		}

		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "🎉 Association complete!")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "📱 Next Steps:")
		fmt.Fprintf(os.Stderr, "  1. Insert blank storage (microSD) into the BrightSign device")
		fmt.Fprintf(os.Stderr, "  2. Power on the device")
		fmt.Fprintf(os.Stderr, "  3. Device will query B-Deploy with its serial number")
		fmt.Fprintf(os.Stderr, "  4. B-Deploy will deliver the associated setup configuration")
		fmt.Fprintf(os.Stderr, "  5. Device will automatically provision and download content")
	}
}
