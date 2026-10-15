package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/brightsign/bscontrol"
)

func main() {
	var (
		helpFlag     = flag.Bool("help", false, "Display usage information")
		jsonFlag     = flag.Bool("json", false, "Output as JSON")
		quietFlag    = flag.Bool("quiet", false, "Suppress non-essential output")
		verboseFlag  = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag  = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag  *string
		pageSizeFlag = flag.Int("page-size", 20, "Number of devices to show per page")
		filterFlag   = flag.String("filter", "", "Filter expression for device listing")
		sortFlag     = flag.String("sort", "", "Sort expression for device listing")
		serialFlag   = flag.String("serial", "", "Get specific device by serial number")
		idFlag       = flag.Int("id", 0, "Get specific device by ID")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for testing BSN.cloud device management.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  List all devices:\n")
		fmt.Fprintf(os.Stderr, "    %s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  List devices with smaller page size:\n")
		fmt.Fprintf(os.Stderr, "    %s --page-size 10\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get specific device:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial A1B2C3D4E5F6\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Filter and sort:\n")
		fmt.Fprintf(os.Stderr, "    %s --filter 'model=XD1033' --sort 'registrationDate'\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --json\n", os.Args[0])
	}

	flag.Parse()

	// Show help if requested
	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// Create client with options
	opts := []bscontrol.Option{bscontrol.WithEnv()}

	// Add timeout if specified
	if *timeoutFlag != 30 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}

	// Add network if specified
	if *networkFlag != "" {
		opts = append(opts, bscontrol.WithNetwork(*networkFlag))
	}

	// Create client
	if !*quietFlag && !*jsonFlag {
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
	if !*quietFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔐 Authenticating with BSN.cloud...\n")
	}

	if err := client.Authenticate(ctx); err != nil {
		if bscontrol.IsAuthenticationError(err) {
			log.Fatalf("❌ Authentication failed: %v", err)
		}
		log.Fatalf("❌ Authentication error: %v", err)
	}

	if !*quietFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Authentication successful!\n")
	}

	// Handle network selection
	if err := handleNetworkSelection(ctx, client, *networkFlag, *quietFlag, *verboseFlag, *jsonFlag); err != nil {
		log.Fatalf("❌ Network selection failed: %v", err)
	}

	// Handle specific device requests
	if *serialFlag != "" {
		if err := showDeviceBySerial(ctx, client, *serialFlag, *verboseFlag, *jsonFlag); err != nil {
			log.Fatalf("❌ Failed to get device: %v", err)
		}
		return
	}

	if *idFlag > 0 {
		if err := showDeviceByID(ctx, client, *idFlag, *verboseFlag, *jsonFlag); err != nil {
			log.Fatalf("❌ Failed to get device: %v", err)
		}
		return
	}

	// List devices
	if err := listDevices(ctx, client, *pageSizeFlag, *filterFlag, *sortFlag, *quietFlag, *verboseFlag, *jsonFlag); err != nil {
		log.Fatalf("❌ Failed to list devices: %v", err)
	}
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, quiet, verbose, jsonMode bool) error {
	// Check if network is already set
	if client.IsNetworkSet() {
		if current, err := client.GetCurrentNetwork(ctx); err == nil {
			if !quiet && !jsonMode {
				fmt.Fprintf(os.Stderr, "📡 Using network: %s (ID: %d)\n", current.Name, current.ID)
			}
			return nil
		}
	}

	// If no network flag was provided, check BS_NETWORK environment variable
	if requestedNetwork == "" {
		if envNetwork := os.Getenv("BS_NETWORK"); envNetwork != "" {
			requestedNetwork = envNetwork
			if !quiet && !jsonMode {
				fmt.Fprintf(os.Stderr, "📡 Using network from BS_NETWORK environment variable\n")
			}
		}
	}

	// Get available networks
	if !quiet && !jsonMode {
		fmt.Fprintf(os.Stderr, "📡 Getting available networks...\n")
	}

	networks, err := client.GetNetworks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get networks: %w", err)
	}

	if len(networks) == 0 {
		return fmt.Errorf("no networks available")
	}

	// If a specific network was requested (via flag or env var), try to find it
	if requestedNetwork != "" {
		for _, network := range networks {
			if strings.EqualFold(network.Name, requestedNetwork) {
				if !quiet && !jsonMode {
					fmt.Fprintf(os.Stderr, "📡 Using requested network: %s (ID: %d)\n", network.Name, network.ID)
				}
				return client.SetNetworkByID(ctx, network.ID)
			}
		}

		// Network not found - show error and fall back to interactive selection
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "❌ Network '%s' not found. Available networks:\n", requestedNetwork)
			for i, network := range networks {
				fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
			}
			fmt.Fprintf(os.Stderr, "\n")
		} else {
			return fmt.Errorf("network '%s' not found", requestedNetwork)
		}
	}

	// If only one network and no specific network requested, use it automatically
	if len(networks) == 1 && requestedNetwork == "" {
		if !quiet && !jsonMode {
			fmt.Fprintf(os.Stderr, "📡 Using network: %s (ID: %d)\n", networks[0].Name, networks[0].ID)
		}
		return client.SetNetworkByID(ctx, networks[0].ID)
	}

	// In JSON mode, cannot do interactive selection
	if jsonMode {
		return fmt.Errorf("network selection required: use --network flag or BS_NETWORK environment variable")
	}

	// Show available networks and let user choose
	if requestedNetwork == "" {
		fmt.Fprintf(os.Stderr, "📡 Available networks:\n")
		for i, network := range networks {
			fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
			if verbose {
				fmt.Fprintf(os.Stderr, "     Created: %s, Modified: %s\n",
					network.CreationDate.Format("2006-01-02"),
					network.LastModifiedDate.Format("2006-01-02"))
			}
		}
	}

	// Get user selection
	fmt.Fprint(os.Stderr, "Select network (1-"+strconv.Itoa(len(networks))+"): ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	selection, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || selection < 1 || selection > len(networks) {
		return fmt.Errorf("invalid selection: must be between 1 and %d", len(networks))
	}

	selectedNetwork := networks[selection-1]
	if !quiet {
		fmt.Fprintf(os.Stderr, "📡 Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)
	}

	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}

func showDeviceBySerial(ctx context.Context, client *bscontrol.Client, serial string, verbose, jsonMode bool) error {
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "🔍 Getting device with serial: %s\n", serial)
	}

	device, err := client.Devices.Get(ctx, serial)
	if err != nil {
		return err
	}

	if jsonMode {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(device); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
		return nil
	}

	printDevice(device, verbose)
	return nil
}

func showDeviceByID(ctx context.Context, client *bscontrol.Client, id int, verbose, jsonMode bool) error {
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "🔍 Getting device with ID: %d\n", id)
	}

	device, err := client.Devices.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if jsonMode {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(device); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
		return nil
	}

	printDevice(device, verbose)
	return nil
}

func listDevices(ctx context.Context, client *bscontrol.Client, pageSize int, filter, sort string, quiet, verbose, jsonMode bool) error {
	// Build list options
	var opts []bscontrol.ListOption

	if pageSize > 0 {
		opts = append(opts, bscontrol.WithPageSize(pageSize))
	}

	if filter != "" {
		opts = append(opts, bscontrol.WithFilter(filter))
		if !quiet && !jsonMode {
			fmt.Fprintf(os.Stderr, "🔍 Using filter: %s\n", filter)
		}
	}

	if sort != "" {
		opts = append(opts, bscontrol.WithSort(sort))
		if !quiet && !jsonMode {
			fmt.Fprintf(os.Stderr, "📊 Using sort: %s\n", sort)
		}
	}

	if !quiet && !jsonMode {
		fmt.Fprintf(os.Stderr, "📱 Getting devices...\n")
	}

	deviceList, err := client.Devices.List(ctx, opts...)
	if err != nil {
		return err
	}

	if jsonMode {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(deviceList); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
		return nil
	}

	if len(deviceList.Items) == 0 {
		fmt.Fprintf(os.Stderr, "📱 No devices found in the network\n")
		return nil
	}

	// Show summary
	fmt.Fprintf(os.Stderr, "📱 Found %d device(s)", len(deviceList.Items))
	if deviceList.TotalCount > 0 {
		fmt.Fprintf(os.Stderr, " (total: %d)", deviceList.TotalCount)
	}
	if deviceList.IsTruncated {
		fmt.Fprintf(os.Stderr, " - more available")
	}
	fmt.Fprintf(os.Stderr, "\n")

	if verbose {
		fmt.Fprintf(os.Stderr, "   Pagination: truncated=%v, nextMarker=%s\n",
			deviceList.IsTruncated, deviceList.NextMarker)
	}

	fmt.Fprintf(os.Stderr, "\n")

	// Show devices
	for i, device := range deviceList.Items {
		fmt.Fprintf(os.Stderr, "Device %d:\n", i+1)
		printDevice(&device, verbose)
		if i < len(deviceList.Items)-1 {
			fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("-", 50))
		}
	}

	// Handle pagination
	if deviceList.IsTruncated && deviceList.NextMarker != "" {
		fmt.Fprintf(os.Stderr, "\n📄 More devices available. Use --marker '%s' to get next page\n", deviceList.NextMarker)
	}

	return nil
}

func printDevice(device *bscontrol.Device, verbose bool) {
	fmt.Fprintf(os.Stderr, "  ID:           %d\n", device.ID)
	fmt.Fprintf(os.Stderr, "  Serial:       %s\n", device.Serial)
	fmt.Fprintf(os.Stderr, "  Model:        %s\n", device.Model)
	fmt.Fprintf(os.Stderr, "  Family:       %s\n", device.Family)
	fmt.Fprintf(os.Stderr, "  Registered:   %s\n", device.RegistrationDate.Format("2006-01-02 15:04:05"))

	if verbose {
		fmt.Fprintf(os.Stderr, "  Modified:     %s\n", device.LastModifiedDate.Format("2006-01-02 15:04:05"))

		if device.Settings != nil {
			fmt.Fprintf(os.Stderr, "  Name:         %s\n", device.Settings.Name)
			if device.Settings.Description != "" {
				fmt.Fprintf(os.Stderr, "  Description:  %s\n", device.Settings.Description)
			}
			fmt.Fprintf(os.Stderr, "  Setup Type:   %s\n", device.Settings.SetupType)
			fmt.Fprintf(os.Stderr, "  Timezone:     %s\n", device.Settings.Timezone)

			if device.Settings.Group != nil {
				fmt.Fprintf(os.Stderr, "  Group:        %s (ID: %d)\n", device.Settings.Group.Name, device.Settings.Group.ID)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\n")
}
