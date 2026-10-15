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
		debugFlag   = flag.Bool("debug", false, "Enable debug logging of HTTP requests/responses")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		serialFlag  = flag.String("serial", "", "Device serial number (required)")
		pathFlag    = flag.String("path", "", "Path to file to rename (e.g., 'sd/test.txt') (required)")
		nameFlag    = flag.String("name", "", "New filename (required)")
		networkFlag *string
	)

	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Rename a file on a BrightSign player via rDWS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Rename a file:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --path sd/test.txt --name renamed.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Rename with verbose output:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --path sd/old.txt --name new.txt --verbose\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --path sd/old.txt --name new.txt --json\n", os.Args[0])
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

	if *nameFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --name is required\n\n")
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
	if *debugFlag {
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

	// Rename file
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Renaming '%s' to '%s' on device %s...\n", *pathFlag, *nameFlag, *serialFlag)
	}

	success, err := client.RDWS.RenameFile(ctx, *serialFlag, *pathFlag, *nameFlag)
	if err != nil {
		log.Fatalf("Failed to rename file: %v", err)
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success":      success,
			"serial":       *serialFlag,
			"originalPath": *pathFlag,
			"newName":      *nameFlag,
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
		fmt.Fprintf(os.Stderr, "✅ File renamed successfully: %s\n", *nameFlag)
		if *verboseFlag {
			fmt.Fprintf(os.Stderr, "   Original: %s\n", *pathFlag)
			fmt.Fprintf(os.Stderr, "   New name: %s\n", *nameFlag)
		}
	} else {
		if *debugFlag {
			fmt.Fprintf(os.Stderr, "❌ Rename failed - check debug output above for details\n")
		} else {
			fmt.Fprintf(os.Stderr, "❌ Rename failed - add --debug to see detailed HTTP request/response logs\n")
		}
		os.Exit(1)
	}
}
