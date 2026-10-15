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
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
		verboseFlag = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		serialFlag  = flag.String("serial", "", "Device serial number (required)")
		pathFlag    = flag.String("path", "", "Path to file to delete (e.g., 'sd/test.txt') (required)")
		forceFlag   = flag.Bool("force", false, "Delete without confirmation")
		networkFlag *string
	)

	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Delete a file from a BrightSign player via rDWS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Delete a file:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --path sd/test.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Delete without confirmation:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --path sd/test.txt --force\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --path sd/test.txt --force --json\n", os.Args[0])
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

	if *pathFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --path is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *jsonFlag && !*forceFlag {
		fmt.Fprintf(os.Stderr, "Error: --force is required when using --json (cannot prompt for confirmation)\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Confirm deletion unless --force is used
	if !*forceFlag {
		fmt.Fprintf(os.Stderr, "Are you sure you want to delete '%s' from device %s? (y/N): ", *pathFlag, *serialFlag)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Fprintf(os.Stderr, "Deletion cancelled\n")
			return
		}
	}

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}
	if *networkFlag != "" {
		opts = append(opts, bscontrol.WithNetwork(*networkFlag))
	}
	if *verboseFlag {
		opts = append(opts, bscontrol.WithDebug(true))
	}

	if !*verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Creating BSN.cloud client...\n")
	}

	client, err := bscontrol.New(opts...)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Authenticate
	if !*verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Authenticating...\n")
	}

	if err := client.Authenticate(ctx); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	// Delete file
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Deleting '%s' from device %s...\n", *pathFlag, *serialFlag)
	}

	success, err := client.RDWS.DeleteFile(ctx, *serialFlag, *pathFlag)
	if err != nil {
		log.Fatalf("Failed to delete file: %v", err)
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success": success,
			"serial":  *serialFlag,
			"path":    *pathFlag,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		if !success {
			os.Exit(1)
		}
		return
	}

	if success {
		fmt.Fprintf(os.Stderr, "✅ File deleted successfully: %s\n", *pathFlag)
		if *verboseFlag {
			fmt.Fprintf(os.Stderr, "   Device: %s\n", *serialFlag)
		}
	} else {
		fmt.Fprintf(os.Stderr, "❌ Deletion failed\n")
		os.Exit(1)
	}
}
