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
		helpFlag       = flag.Bool("help", false, "Display usage information")
		jsonFlag       = flag.Bool("json", false, "Output as JSON")
		verboseFlag    = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag    = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag    *string
		serialFlag     = flag.String("serial", "", "Get errors for device with serial number")
		idFlag         = flag.Int("id", 0, "Get errors for device with ID")
		pageSizeFlag   = flag.Int("page-size", 20, "Number of errors to show per page")
		filterFlag     = flag.String("filter", "", "Filter expression for error listing")
		sortFlag       = flag.String("sort", "", "Sort expression for error listing")
		severityFlag   = flag.String("severity", "", "Filter by severity: info, warning, error, critical")
		typeFlag       = flag.String("type", "", "Filter by error type: system, content, network, etc.")
		unresolvedFlag = flag.Bool("unresolved", false, "Show only unresolved errors")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for retrieving BrightSign device error logs.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get errors by serial:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial A1B2C3D4E5F6\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get critical errors only:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial A1B2C3D4E5F6 --severity critical\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get unresolved system errors:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345 --type system --unresolved\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Verbose output with pagination:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial A1B2C3D4E5F6 --verbose --page-size 10\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial A1B2C3D4E5F6 --json\n", os.Args[0])
	}

	flag.Parse()

	// Show help if requested
	if *helpFlag {
		flag.Usage()
		os.Exit(0)
	}

	// Validate required arguments
	if *serialFlag == "" && *idFlag == 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: Either --serial or --id must be specified\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *serialFlag != "" && *idFlag != 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: Cannot specify both --serial and --id\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create client with options
	opts := []bscontrol.Option{bscontrol.WithEnv()}

	if *timeoutFlag != 30 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}

	if *networkFlag != "" {
		opts = append(opts, bscontrol.WithNetwork(*networkFlag))
	}

	// Create client
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

	// Handle network selection
	if err := handleNetworkSelection(ctx, client, *networkFlag, *verboseFlag, *jsonFlag); err != nil {
		log.Fatalf("❌ Network selection failed: %v", err)
	}

	// Build filter expression
	var filterParts []string
	if *severityFlag != "" {
		filterParts = append(filterParts, fmt.Sprintf("severity=%s", *severityFlag))
	}
	if *typeFlag != "" {
		filterParts = append(filterParts, fmt.Sprintf("errorType=%s", *typeFlag))
	}
	if *unresolvedFlag {
		filterParts = append(filterParts, "resolved=false")
	}
	if *filterFlag != "" {
		filterParts = append(filterParts, *filterFlag)
	}

	combinedFilter := strings.Join(filterParts, " AND ")

	// Build list options
	var listOpts []bscontrol.ListOption
	if *pageSizeFlag > 0 {
		listOpts = append(listOpts, bscontrol.WithPageSize(*pageSizeFlag))
	}
	if combinedFilter != "" {
		listOpts = append(listOpts, bscontrol.WithFilter(combinedFilter))
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "🔍 Using filter: %s\n", combinedFilter)
		}
	}
	if *sortFlag != "" {
		listOpts = append(listOpts, bscontrol.WithSort(*sortFlag))
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "📊 Using sort: %s\n", *sortFlag)
		}
	}

	// Get device errors
	var errorList *bscontrol.DeviceErrorList

	if *serialFlag != "" {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "📋 Getting errors for device with serial: %s\n", *serialFlag)
		}
		errorList, err = client.Devices.GetErrorsBySerial(ctx, *serialFlag, listOpts...)
		if err != nil {
			log.Fatalf("❌ Failed to get device errors: %v", err)
		}
	} else {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "📋 Getting errors for device with ID: %d\n", *idFlag)
		}
		errorList, err = client.Devices.GetErrors(ctx, *idFlag, listOpts...)
		if err != nil {
			log.Fatalf("❌ Failed to get device errors: %v", err)
		}
	}

	// Output as JSON if requested
	if *jsonFlag {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(errorList); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	// Display results
	if len(errorList.Items) == 0 {
		fmt.Fprintf(os.Stderr, "📋 No errors found for this device\n")
		return
	}

	// Show summary
	fmt.Fprintf(os.Stderr, "\n📋 Found %d error(s)", len(errorList.Items))
	if errorList.TotalCount > 0 {
		fmt.Fprintf(os.Stderr, " (total: %d)", errorList.TotalCount)
	}
	if errorList.IsTruncated {
		fmt.Fprintf(os.Stderr, " - more available")
	}
	fmt.Fprintf(os.Stderr, "\n")

	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "   Pagination: truncated=%v, nextMarker=%s\n",
			errorList.IsTruncated, errorList.NextMarker)
	}

	fmt.Fprintf(os.Stderr, "\n")

	// Show errors
	for i, deviceError := range errorList.Items {
		fmt.Fprintf(os.Stderr, "Error %d:\n", i+1)
		printError(&deviceError, *verboseFlag)
		if i < len(errorList.Items)-1 {
			fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("-", 60))
		}
	}

	// Handle pagination
	if errorList.IsTruncated && errorList.NextMarker != "" {
		fmt.Fprintf(os.Stderr, "\n📄 More errors available. Use --marker '%s' to get next page\n", errorList.NextMarker)
	}
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, verbose bool, jsonMode bool) error {
	// Check if network is already set
	if client.IsNetworkSet() {
		if current, err := client.GetCurrentNetwork(ctx); err == nil {
			if !jsonMode {
				fmt.Fprintf(os.Stderr, "📡 Using network: %s (ID: %d)\n", current.Name, current.ID)
			}
			return nil
		}
	}

	// If no network flag was provided, check BS_NETWORK environment variable
	if requestedNetwork == "" {
		if envNetwork := os.Getenv("BS_NETWORK"); envNetwork != "" {
			requestedNetwork = envNetwork
			if !jsonMode {
				fmt.Fprintf(os.Stderr, "Using network from BS_NETWORK environment variable\n")
			}
		}
	}

	// Get available networks
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "📡 Getting available networks...\n")
	}

	networks, err := client.GetNetworks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get networks: %w", err)
	}

	if len(networks) == 0 {
		return fmt.Errorf("no networks available")
	}

	// If a specific network was requested, try to find it
	if requestedNetwork != "" {
		for _, network := range networks {
			if strings.EqualFold(network.Name, requestedNetwork) {
				if !jsonMode {
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
		}
	}

	// If only one network and no specific network requested, use it automatically
	if len(networks) == 1 && requestedNetwork == "" {
		if !jsonMode {
			fmt.Fprintf(os.Stderr, "📡 Using network: %s (ID: %d)\n", networks[0].Name, networks[0].ID)
		}
		return client.SetNetworkByID(ctx, networks[0].ID)
	}

	// Show available networks and let user choose
	if requestedNetwork == "" {
		fmt.Fprintf(os.Stderr, "📡 Available networks:\n")
		for i, network := range networks {
			fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
			if verbose && !jsonMode {
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
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "📡 Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)
	}

	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}

func printError(deviceError *bscontrol.DeviceError, verbose bool) {
	// Determine severity - try multiple fields
	severity := deviceError.Severity
	if severity == "" {
		severity = deviceError.Level
	}
	if severity == "" {
		severity = "unknown"
	}

	// Severity with emoji
	severityIcon := "ℹ️"
	switch strings.ToLower(severity) {
	case "info":
		severityIcon = "ℹ️"
	case "warning":
		severityIcon = "⚠️"
	case "error":
		severityIcon = "❌"
	case "critical":
		severityIcon = "🚨"
	}

	// Determine message - try multiple fields
	message := deviceError.Message
	if message == "" {
		message = deviceError.Description
	}
	if message == "" {
		message = deviceError.Name
	}

	// Determine error type/code
	errorType := deviceError.ErrorType
	if errorType == "" {
		errorType = deviceError.Type
	}

	errorCode := deviceError.ErrorCode
	if errorCode == "" {
		errorCode = deviceError.Code
	}

	// Status determination - try multiple fields
	statusIcon := "❓"
	status := "Open"
	if deviceError.Resolved {
		statusIcon = "✅"
		status = "Resolved"
	} else if deviceError.Status != "" {
		status = deviceError.Status
		if strings.ToLower(status) == "resolved" || strings.ToLower(status) == "closed" {
			statusIcon = "✅"
		}
	}

	// Determine timestamp - try multiple fields
	timestamp := deviceError.Timestamp
	if timestamp.IsZero() {
		timestamp = deviceError.CreationDate
	}
	if timestamp.IsZero() {
		timestamp = deviceError.LastModifiedDate
	}

	// Display basic fields
	if deviceError.ID > 0 {
		fmt.Printf("  ID:           %d\n", deviceError.ID)
	} else {
		fmt.Printf("  ID:           %s\n", "N/A")
	}

	fmt.Printf("  Severity:     %s %s\n", severityIcon, severity)
	fmt.Printf("  Type:         %s\n", errorType)
	fmt.Printf("  Code:         %s\n", errorCode)
	fmt.Printf("  Status:       %s %s\n", statusIcon, status)
	fmt.Printf("  Message:      %s\n", message)

	if deviceError.Details != "" && verbose {
		fmt.Printf("  Details:      %s\n", deviceError.Details)
	}

	if !timestamp.IsZero() {
		fmt.Printf("  Timestamp:    %s\n", timestamp.Format("2006-01-02 15:04:05 MST"))
	} else {
		fmt.Printf("  Timestamp:    %s\n", "N/A")
	}

	if verbose {
		// Show all populated fields in verbose mode
		if deviceError.DeviceID != "" {
			fmt.Printf("  Device ID:    %s\n", deviceError.DeviceID)
		}
		if deviceError.Serial != "" {
			fmt.Printf("  Serial:       %s\n", deviceError.Serial)
		}
		if deviceError.Source != "" {
			fmt.Printf("  Source:       %s\n", deviceError.Source)
		}
		if deviceError.Component != "" {
			fmt.Printf("  Component:    %s\n", deviceError.Component)
		}

		if deviceError.Resolved && deviceError.ResolvedAt != nil {
			fmt.Printf("  Resolved At:  %s\n", deviceError.ResolvedAt.Format("2006-01-02 15:04:05 MST"))
		}

		// Debug: Show raw structure in verbose mode to understand API response
		fmt.Printf("  [DEBUG] Raw error data:\n")
		fmt.Printf("    ID: %d, DeviceID: '%s', Serial: '%s'\n", deviceError.ID, deviceError.DeviceID, deviceError.Serial)
		fmt.Printf("    ErrorCode: '%s', ErrorType: '%s', Code: '%s', Type: '%s'\n",
			deviceError.ErrorCode, deviceError.ErrorType, deviceError.Code, deviceError.Type)
		fmt.Printf("    Severity: '%s', Level: '%s', Message: '%s'\n",
			deviceError.Severity, deviceError.Level, deviceError.Message)
		fmt.Printf("    Description: '%s', Name: '%s'\n", deviceError.Description, deviceError.Name)
		fmt.Printf("    Status: '%s', Resolved: %t\n", deviceError.Status, deviceError.Resolved)
		fmt.Printf("    Timestamps: Timestamp=%v, CreationDate=%v, LastModifiedDate=%v\n",
			deviceError.Timestamp, deviceError.CreationDate, deviceError.LastModifiedDate)
	}

	fmt.Fprintf(os.Stderr, "\n")
}
