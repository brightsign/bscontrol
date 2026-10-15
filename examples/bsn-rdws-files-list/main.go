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
		verboseFlag = flag.Bool("verbose", false, "Show detailed file information")
		jsonFlag    = flag.Bool("json", false, "Output raw JSON response")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		serialFlag  = flag.String("serial", "", "Device serial number (required)")
		pathFlag    = flag.String("path", "sd", "Path to list files from (default: sd)")
		networkFlag *string
	)

	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "List files and directories on a BrightSign player via rDWS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  List files in /sd directory:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --path sd\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  List with verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --path sd --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --path sd --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	if *serialFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --serial is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}
	if *networkFlag != "" {
		opts = append(opts, bscontrol.WithNetwork(*networkFlag))
	}

	if !*jsonFlag {
		fmt.Println("Creating BSN.cloud client...")
	}

	client, err := bscontrol.New(opts...)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Authenticate
	if !*jsonFlag {
		fmt.Println("Authenticating...")
	}

	if err := client.Authenticate(ctx); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	// List files
	if !*jsonFlag {
		fmt.Printf("Listing files at /%s on device %s...\n", *pathFlag, *serialFlag)
	}

	response, err := client.RDWS.ListFiles(ctx, *serialFlag, *pathFlag)
	if err != nil {
		log.Fatalf("Failed to list files: %v", err)
	}

	// Display results
	if *jsonFlag {
		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		displayFileList(&response.Data.Result, *verboseFlag)
	}
}

func displayFileList(result *bscontrol.RDWSFileListResult, verbose bool) {
	fmt.Fprintf(os.Stderr, "\n=== File Listing ===\n")

	if result.Type != "" {
		fmt.Fprintf(os.Stderr, "Type: %s\n", result.Type)
	}
	if result.Path != "" {
		fmt.Fprintf(os.Stderr, "Path: /%s\n", result.Path)
	}

	// Display storage info if available
	if result.StorageInfo != nil {
		fmt.Fprintf(os.Stderr, "\n=== Storage Information ===\n")
		if result.StorageInfo.FileSystemType != "" {
			fmt.Fprintf(os.Stderr, "Filesystem: %s\n", result.StorageInfo.FileSystemType)
		}
		if result.StorageInfo.MountedOn != "" {
			fmt.Fprintf(os.Stderr, "Mounted on: %s\n", result.StorageInfo.MountedOn)
		}
		if result.StorageInfo.Stats != nil {
			stats := result.StorageInfo.Stats
			totalGB := float64(stats.SizeBytes) / (1024 * 1024 * 1024)
			freeGB := float64(stats.BytesFree) / (1024 * 1024 * 1024)
			usedGB := totalGB - freeGB
			usedPercent := (usedGB / totalGB) * 100

			fmt.Fprintf(os.Stderr, "Total: %.2f GB\n", totalGB)
			fmt.Fprintf(os.Stderr, "Used:  %.2f GB (%.1f%%)\n", usedGB, usedPercent)
			fmt.Fprintf(os.Stderr, "Free:  %.2f GB\n", freeGB)
			if stats.IsReadOnly {
				fmt.Fprintf(os.Stderr, "Read-only: Yes\n")
			}
		}
	}

	// Display files
	files := result.Files
	if len(files) == 0 {
		files = result.Contents
	}

	if len(files) > 0 {
		fmt.Fprintf(os.Stderr, "\n=== Files and Directories ===\n")
		for _, file := range files {
			displayFile(&file, verbose, 0)
		}
	} else {
		fmt.Fprintf(os.Stderr, "\n(No files found)\n")
	}

	fmt.Fprintf(os.Stderr, "\n")
}

func displayFile(file *bscontrol.RDWSFileInfo, verbose bool, indent int) {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	typeIcon := "📄"
	if file.Type == "dir" {
		typeIcon = "📁"
	}

	fmt.Fprintf(os.Stderr, "%s%s %s\n", prefix, typeIcon, file.Name)

	if verbose && file.Stat != nil {
		sizeStr := ""
		if file.Type == "file" {
			size := file.Stat.Size
			if size < 1024 {
				sizeStr = fmt.Sprintf("%d B", size)
			} else if size < 1024*1024 {
				sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
			} else if size < 1024*1024*1024 {
				sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
			} else {
				sizeStr = fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
			}
			fmt.Fprintf(os.Stderr, "%s  Size: %s\n", prefix, sizeStr)
		}

		if file.Mime != "" {
			fmt.Fprintf(os.Stderr, "%s  Type: %s\n", prefix, file.Mime)
		}

		if file.Stat.Mtime != "" {
			fmt.Fprintf(os.Stderr, "%s  Modified: %s\n", prefix, file.Stat.Mtime)
		}
	}

	// Display children recursively if present
	if len(file.Children) > 0 {
		for _, child := range file.Children {
			displayFile(&child, verbose, indent+1)
		}
	}
}
