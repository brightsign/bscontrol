package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/brightsign/bscontrol"
	"github.com/brightsign/bscontrol/internal/types"
)

func main() {
	var (
		helpFlag    = flag.Bool("help", false, "Display usage information")
		verboseFlag = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag = flag.Int("timeout", 60, "Request timeout in seconds (snapshots can take time)")
		networkFlag *string
		serialFlag  = flag.String("serial", "", "Device serial number")
		idFlag      = flag.Int("id", 0, "Device ID")
		formatFlag  = flag.String("format", "png", "Image format: png, jpeg")
		qualityFlag = flag.Int("quality", 90, "JPEG quality (1-100, only for jpeg format)")
		outputFlag  = flag.String("output", ".", "Output directory for saved snapshots")
		displayFlag = flag.Int("display", 0, "Display ID (0 for primary display)")
		widthFlag   = flag.Int("width", 0, "Crop width (0 for full width)")
		heightFlag  = flag.Int("height", 0, "Crop height (0 for full height)")
		xFlag       = flag.Int("x", 0, "Crop X coordinate")
		yFlag       = flag.Int("y", 0, "Crop Y coordinate")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to capture screenshots from BrightSign devices via rDWS API.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Basic screenshot:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  JPEG screenshot with custom quality:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --format jpeg --quality 85\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Cropped screenshot of specific region:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --x 100 --y 100 --width 800 --height 600\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Screenshot to specific folder:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --output ./screenshots\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Screenshot with JSON output (includes base64 image data):\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Validate input
	if *serialFlag == "" && *idFlag == 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: Must specify either --serial or --id\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *serialFlag != "" && *idFlag != 0 {
		fmt.Fprintf(os.Stderr, "❌ Error: Cannot specify both --serial and --id\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate format
	if *formatFlag != "png" && *formatFlag != "jpeg" {
		fmt.Fprintf(os.Stderr, "❌ Error: Format must be 'png' or 'jpeg'\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Validate quality
	if *qualityFlag < 1 || *qualityFlag > 100 {
		fmt.Fprintf(os.Stderr, "❌ Error: Quality must be between 1 and 100\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputFlag, 0755); err != nil {
		log.Fatalf("❌ Failed to create output directory: %v", err)
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

	// Handle network selection
	if err := handleNetworkSelection(ctx, client, *networkFlag, *verboseFlag, *jsonFlag); err != nil {
		log.Fatalf("❌ Network selection failed: %v", err)
	}

	// Get device info for display
	var deviceInfo string
	var deviceSerial string
	if *serialFlag != "" {
		deviceInfo = fmt.Sprintf("serial %s", *serialFlag)
		deviceSerial = *serialFlag
	} else {
		deviceInfo = fmt.Sprintf("ID %d", *idFlag)
		deviceSerial = fmt.Sprintf("%d", *idFlag)
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "📸 Capturing screenshot from device with %s\n", deviceInfo)
	}

	// Build filename
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("snapshot-%s-%s.%s", deviceSerial, timestamp, *formatFlag)
	filepath := filepath.Join(*outputFlag, filename)

	// Display parameters
	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "  Format: %s\n", *formatFlag)
		if *formatFlag == "jpeg" {
			fmt.Fprintf(os.Stderr, "  Quality: %d\n", *qualityFlag)
		}
		fmt.Fprintf(os.Stderr, "  Display ID: %d\n", *displayFlag)
		if *widthFlag > 0 && *heightFlag > 0 {
			fmt.Fprintf(os.Stderr, "  Region: %dx%d at (%d,%d)\n", *widthFlag, *heightFlag, *xFlag, *yFlag)
		}
		fmt.Fprintf(os.Stderr, "  Output: %s\n", filepath)
	}

	// Capture screenshot
	success, savedPath, imageData, err := captureSnapshot(
		ctx, client, deviceSerial, filepath,
		*formatFlag, *qualityFlag, *displayFlag,
		*xFlag, *yFlag, *widthFlag, *heightFlag, *verboseFlag, *jsonFlag)

	if err != nil {
		log.Fatalf("❌ Failed to capture snapshot: %v", err)
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success": success,
			"serial":  deviceSerial,
			"format":  *formatFlag,
		}
		if savedPath != "" {
			result["file_path"] = savedPath
			if fileInfo, err := os.Stat(savedPath); err == nil {
				result["file_size"] = fileInfo.Size()
			}
		}
		if imageData != "" {
			result["image_data"] = imageData
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	if success && savedPath != "" {
		fmt.Fprintf(os.Stderr, "✅ Screenshot saved to: %s\n", savedPath)

		// Show file info
		if fileInfo, err := os.Stat(savedPath); err == nil {
			fmt.Fprintf(os.Stderr, "📊 File size: %d bytes\n", fileInfo.Size())
		}
	} else if success {
		fmt.Fprintf(os.Stderr, "⏳ Screenshot request submitted but image data not available\n")
		fmt.Fprintf(os.Stderr, "💡 The device may be processing the request asynchronously\n")
	} else {
		fmt.Fprintf(os.Stderr, "❌ Screenshot capture failed\n")
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
	fmt.Fprintf(os.Stderr, "📡 Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)

	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}

func captureSnapshot(ctx context.Context, client *bscontrol.Client, deviceSerial, filepath, format string, quality, displayID, x, y, width, height int, verbose bool, jsonMode bool) (bool, string, string, error) {
	// Build snapshot request
	request := &types.SnapshotRequest{
		Format:          format,
		Quality:         quality,
		IncludeMetadata: true,
		Output:          "base64",
		Compression:     "medium",
		DisplayID:       displayID,
	}

	// Add region if specified
	if width > 0 && height > 0 {
		request.Region = &types.Region{
			X:      x,
			Y:      y,
			Width:  width,
			Height: height,
		}
	}

	if verbose && !jsonMode {
		fmt.Fprintf(os.Stderr, "📋 Making snapshot API call...\n")
		if request.Region != nil {
			fmt.Fprintf(os.Stderr, "  Region: %dx%d at (%d,%d)\n", request.Region.Width, request.Region.Height, request.Region.X, request.Region.Y)
		}
	}

	// Make the API call using the device service
	response, err := client.Devices.TakeSnapshotBySerial(ctx, deviceSerial, request)
	if err != nil {
		return false, "", "", fmt.Errorf("failed to capture snapshot: %w", err)
	}

	if verbose && !jsonMode {
		fmt.Fprintf(os.Stderr, "✅ API call successful\n")
		if response.Timestamp != "" {
			fmt.Fprintf(os.Stderr, "  Captured at: %s\n", response.Timestamp)
		}
		if response.Width > 0 && response.Height > 0 {
			fmt.Fprintf(os.Stderr, "  Dimensions: %dx%d\n", response.Width, response.Height)
		}
	}

	// Extract image data
	var imageData string
	if response.Data != "" {
		imageData = response.Data
	} else if response.RemoteSnapshotThumbnail != "" {
		// Extract base64 data from data URI format
		imageData = extractBase64FromDataURI(response.RemoteSnapshotThumbnail)
	}

	if imageData == "" {
		return false, "", "", fmt.Errorf("no image data found in response")
	}

	if verbose && !jsonMode {
		fmt.Fprintf(os.Stderr, "📊 Image data size: %d characters\n", len(imageData))
	}

	// Save image to file (unless in JSON mode where we return the data)
	if !jsonMode {
		err = saveBase64Image(imageData, filepath)
		if err != nil {
			return false, "", "", fmt.Errorf("failed to save image: %w", err)
		}
		return true, filepath, "", nil
	}

	// In JSON mode, return the base64 data instead of saving to file
	return true, "", imageData, nil
}

// saveBase64Image saves base64 image data to a file
func saveBase64Image(base64Data, filename string) error {
	// Handle data URL format or pure base64
	var imageData string
	if strings.HasPrefix(base64Data, "data:") {
		parts := strings.Split(base64Data, ",")
		if len(parts) != 2 {
			return fmt.Errorf("invalid data URL format")
		}
		imageData = parts[1]
	} else {
		imageData = base64Data
	}

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(imageData)
	if err != nil {
		return fmt.Errorf("failed to decode base64: %w", err)
	}

	// Write to file
	err = os.WriteFile(filename, decoded, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// extractBase64FromDataURI extracts base64 data from a data URI
func extractBase64FromDataURI(dataURI string) string {
	if strings.HasPrefix(dataURI, "data:") {
		parts := strings.Split(dataURI, ",")
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return dataURI
}
