package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/brightsign/bscontrol"
)

func main() {
	var (
		helpFlag    = flag.Bool("help", false, "Display usage information")
		verboseFlag = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag *string
		serialFlag  = flag.String("serial", "", "Device serial number")
		idFlag      = flag.Int("id", 0, "Device ID")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
		saveFlag    = flag.String("save", "", "Save logs to directory")
		logfileFlag = flag.String("logfile", "", "Write logs to specific file")
		stdoutFlag  = flag.Bool("stdout", false, "Output only log content to stdout (suppress all status messages)")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for retrieving BrightSign player log files.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get logs:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get logs as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Save logs to directory:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --save ./logs\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Write logs to specific file:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --logfile player.log\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output logs to stdout (for piping):\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --stdout > output.log\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Validate input
	if *serialFlag == "" && *idFlag == 0 {
		fmt.Fprintf(os.Stderr, "Error: Must specify either --serial or --id\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *serialFlag != "" && *idFlag != 0 {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify both --serial and --id\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate output mode flags are mutually exclusive
	outputModes := 0
	if *jsonFlag {
		outputModes++
	}
	if *saveFlag != "" {
		outputModes++
	}
	if *logfileFlag != "" {
		outputModes++
	}
	if *stdoutFlag {
		outputModes++
	}
	if outputModes > 1 {
		fmt.Fprintf(os.Stderr, "Error: --json, --save, --logfile, and --stdout are mutually exclusive\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Determine quiet mode (suppress status messages)
	quietMode := *stdoutFlag

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}

	// Add network if specified
	if *networkFlag != "" {
		opts = append(opts, bscontrol.WithNetwork(*networkFlag))
	}

	if !quietMode && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Creating BSN.cloud client...\n")
	}

	client, err := bscontrol.New(opts...)
	if err != nil {
		if bscontrol.IsConfigurationError(err) {
			fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Failed to create client: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Authenticate
	if !quietMode && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Authenticating with BSN.cloud...\n")
	}

	if err := client.Authenticate(ctx); err != nil {
		if bscontrol.IsAuthenticationError(err) {
			fmt.Fprintf(os.Stderr, "Authentication failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Authentication error: %v\n", err)
		os.Exit(1)
	}

	if !quietMode && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Authentication successful!\n")
	}

	// Handle network selection
	if err := handleNetworkSelection(ctx, client, *networkFlag, *verboseFlag, quietMode, *jsonFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Network selection failed: %v\n", err)
		os.Exit(1)
	}

	// Get device serial number
	var serial string
	if *serialFlag != "" {
		serial = *serialFlag
	} else {
		// Get device by ID to retrieve serial number
		device, err := client.Devices.GetByID(ctx, *idFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get device with ID %d: %v\n", *idFlag, err)
			os.Exit(1)
		}
		serial = device.Serial
	}

	// Get logs
	if err := getLogs(ctx, client, serial, *jsonFlag, *verboseFlag, *saveFlag, *logfileFlag, quietMode); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get logs: %v\n", err)
		os.Exit(1)
	}
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, verbose bool, quietMode bool, jsonMode bool) error {
	// Check if network is already set
	if client.IsNetworkSet() {
		if current, err := client.GetCurrentNetwork(ctx); err == nil {
			if !quietMode && !jsonMode {
				fmt.Fprintf(os.Stderr, "Using network: %s (ID: %d)\n", current.Name, current.ID)
			}
			return nil
		}
	}

	// If no network flag was provided, check BS_NETWORK environment variable
	if requestedNetwork == "" {
		if envNetwork := os.Getenv("BS_NETWORK"); envNetwork != "" {
			requestedNetwork = envNetwork
			if !quietMode && !jsonMode {
				fmt.Fprintf(os.Stderr, "Using network from BS_NETWORK environment variable\n")
			}
		}
	}

	// Get available networks
	if !quietMode && !jsonMode {
		fmt.Fprintf(os.Stderr, "Getting available networks...\n")
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
				if !quietMode && !jsonMode {
					fmt.Fprintf(os.Stderr, "Using requested network: %s (ID: %d)\n", network.Name, network.ID)
				}
				return client.SetNetworkByID(ctx, network.ID)
			}
		}

		// Network not found - show error and fall back to interactive selection
		fmt.Fprintf(os.Stderr, "Network '%s' not found. Available networks:\n", requestedNetwork)
		for i, network := range networks {
			fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// If only one network and no specific network requested, use it automatically
	if len(networks) == 1 && requestedNetwork == "" {
		if !quietMode && !jsonMode {
			fmt.Fprintf(os.Stderr, "Using network: %s (ID: %d)\n", networks[0].Name, networks[0].ID)
		}
		return client.SetNetworkByID(ctx, networks[0].ID)
	}

	// Show available networks and let user choose
	if requestedNetwork == "" {
		fmt.Fprintf(os.Stderr, "Available networks:\n")
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
	if !quietMode && !jsonMode {
		fmt.Fprintf(os.Stderr, "Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)
	}

	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}

func getLogs(ctx context.Context, client *bscontrol.Client, serial string, jsonMode bool, verbose bool, saveDir string, logfile string, quietMode bool) error {
	if !quietMode && !jsonMode {
		fmt.Fprintf(os.Stderr, "\nGetting logs from device %s...\n", serial)
	}

	logs, err := client.RDWS.GetLogs(ctx, serial)
	if err != nil {
		return err
	}

	// Handle --json mode
	if jsonMode {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(logs)
	}

	// Handle --stdout mode (output only log content)
	if quietMode {
		for i, logFile := range logs.Files {
			if len(logs.Files) > 1 {
				// Add separator between multiple log files
				if i > 0 {
					os.Stdout.WriteString("\n" + strings.Repeat("-", 70) + "\n")
				}
				os.Stdout.WriteString(fmt.Sprintf("# %s (%d bytes)\n", logFile.Name, logFile.Size))
				os.Stdout.WriteString(strings.Repeat("-", 70) + "\n")
			}
			os.Stdout.WriteString(logFile.Content)
		}
		return nil
	}

	// Handle --logfile mode (write to specific file)
	if logfile != "" {
		var content strings.Builder
		for i, logFile := range logs.Files {
			if len(logs.Files) > 1 {
				if i > 0 {
					content.WriteString("\n" + strings.Repeat("=", 70) + "\n")
				}
				content.WriteString(fmt.Sprintf("# Log File: %s (%d bytes)\n", logFile.Name, logFile.Size))
				content.WriteString(strings.Repeat("=", 70) + "\n")
			}
			content.WriteString(logFile.Content)
		}

		if err := os.WriteFile(logfile, []byte(content.String()), 0644); err != nil {
			return fmt.Errorf("failed to write logfile: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Logs written to: %s\n", logfile)
		return nil
	}

	// Default display mode
	fmt.Fprintf(os.Stderr, "\nPlayer Logs:\n")
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("=", 70))

	if len(logs.Files) == 0 {
		fmt.Fprintf(os.Stderr, "No log files found on device.\n")
		return nil
	}

	// Display logs
	for i, logFile := range logs.Files {
		fmt.Fprintf(os.Stderr, "\n[%d] %s\n", i+1, logFile.Name)
		fmt.Fprintf(os.Stderr, "    Size: %d bytes\n", logFile.Size)

		if verbose && logFile.Content != "" {
			fmt.Fprintf(os.Stderr, "    Content preview:\n")
			lines := strings.Split(logFile.Content, "\n")
			displayLines := 10
			if len(lines) < displayLines {
				displayLines = len(lines)
			}
			for j := 0; j < displayLines; j++ {
				fmt.Fprintf(os.Stderr, "      %s\n", lines[j])
			}
			if len(lines) > displayLines {
				fmt.Fprintf(os.Stderr, "      ... (%d more lines)\n", len(lines)-displayLines)
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\nTotal log files: %d\n", len(logs.Files))

	// Save logs to directory if requested
	if saveDir != "" {
		if err := saveLogs(logs, saveDir, serial); err != nil {
			return fmt.Errorf("failed to save logs: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\nLogs saved to: %s\n", saveDir)
	}

	return nil
}

func saveLogs(logs *bscontrol.RDWSLogs, saveDir string, serial string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create a subdirectory for this device
	deviceDir := fmt.Sprintf("%s/%s_%s", saveDir, serial, time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(deviceDir, 0755); err != nil {
		return fmt.Errorf("failed to create device directory: %w", err)
	}

	// Save each log file
	for _, logFile := range logs.Files {
		filePath := fmt.Sprintf("%s/%s", deviceDir, logFile.Name)
		if err := os.WriteFile(filePath, []byte(logFile.Content), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", logFile.Name, err)
		}
	}

	return nil
}
