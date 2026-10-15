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
		verboseFlag = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag *string
		serialFlag  = flag.String("serial", "", "Device serial number")
		idFlag      = flag.Int("id", 0, "Device ID")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
		statusFlag  = flag.String("status", "", "Filter by status (pending, downloading, complete, error)")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for viewing content downloads on BrightSign devices.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get downloads by serial:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get downloads by ID:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Filter by status:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --status downloading\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  JSON output:\n")
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

	// Step 3: Get downloads
	var downloadList *bscontrol.DeviceDownloadList
	if *serialFlag != "" {
		if *verboseFlag {
			fmt.Printf("📥 Retrieving downloads for device: %s\n", *serialFlag)
		}
		downloadList, err = client.Devices.GetDownloadsBySerial(ctx, *serialFlag)
	} else {
		if *verboseFlag {
			fmt.Printf("📥 Retrieving downloads for device ID: %d\n", *idFlag)
		}
		downloadList, err = client.Devices.GetDownloads(ctx, *idFlag)
	}

	if err != nil {
		log.Fatalf("❌ Failed to get downloads: %v", err)
	}

	// Filter by status if specified
	var filteredDownloads []bscontrol.DeviceDownload
	if *statusFlag != "" {
		for _, download := range downloadList.Items {
			if strings.EqualFold(download.Status, *statusFlag) {
				filteredDownloads = append(filteredDownloads, download)
			}
		}
	} else {
		filteredDownloads = downloadList.Items
	}

	// Output
	if *jsonFlag {
		// JSON output
		output := struct {
			Items      []bscontrol.DeviceDownload `json:"items"`
			TotalCount int                       `json:"totalCount"`
		}{
			Items:      filteredDownloads,
			TotalCount: len(filteredDownloads),
		}
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			log.Fatalf("❌ Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		// Human-readable output
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Println(strings.Repeat("═", 100))
		fmt.Println("Content Downloads")
		fmt.Println(strings.Repeat("═", 100))

		if len(filteredDownloads) == 0 {
			fmt.Println("No downloads found")
			if *statusFlag != "" {
				fmt.Printf("(filtered by status: %s)\n", *statusFlag)
			}
		} else {
			// Calculate totals
			var totalSize, totalDownloaded int64
			for _, download := range filteredDownloads {
				totalSize += download.FileSize
				totalDownloaded += download.DownloadedBytes
			}

			fmt.Printf("Total Downloads: %d", len(filteredDownloads))
			if *statusFlag != "" {
				fmt.Printf(" (status: %s)", *statusFlag)
			}
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Printf("Total Size: %s\n", formatBytes(totalSize))
			if totalSize > 0 {
				percentage := float64(totalDownloaded) / float64(totalSize) * 100
				fmt.Printf("Progress: %s / %s (%.1f%%)\n",
					formatBytes(totalDownloaded), formatBytes(totalSize), percentage)
			}
			fmt.Println(strings.Repeat("─", 100))

			// Display each download
			for i, download := range filteredDownloads {
				if i > 0 {
					fmt.Println(strings.Repeat("─", 100))
				}
				fmt.Printf("File:     %s\n", download.FileName)
				fmt.Printf("Status:   %s\n", formatStatus(download.Status))
				fmt.Printf("Size:     %s\n", formatBytes(download.FileSize))

				// Progress bar
				if download.FileSize > 0 {
					percentage := float64(download.DownloadedBytes) / float64(download.FileSize) * 100
					fmt.Printf("Progress: %s / %s (%.1f%%)\n",
						formatBytes(download.DownloadedBytes), formatBytes(download.FileSize), percentage)
					fmt.Printf("          %s\n", progressBar(percentage, 50))
				}

				fmt.Printf("Started:  %s\n", download.StartTime.Format("2006-01-02 15:04:05"))
				if !download.EndTime.IsZero() {
					fmt.Printf("Ended:    %s\n", download.EndTime.Format("2006-01-02 15:04:05"))
					duration := download.EndTime.Sub(download.StartTime)
					fmt.Printf("Duration: %s\n", duration.Round(time.Second))
				}
				if download.Error != "" {
					fmt.Printf("Error:    %s\n", download.Error)
				}
			}
		}

		fmt.Println(strings.Repeat("═", 100))
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatStatus(status string) string {
	switch strings.ToLower(status) {
	case "pending":
		return "⏳ Pending"
	case "downloading":
		return "⬇️  Downloading"
	case "complete":
		return "✅ Complete"
	case "error":
		return "❌ Error"
	default:
		return status
	}
}

func progressBar(percentage float64, width int) string {
	filled := int(percentage / 100 * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
}
