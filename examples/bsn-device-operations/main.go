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
		statusFlag  = flag.String("status", "", "Filter by status (pending, in_progress, completed, failed)")
		typeFlag    = flag.String("type", "", "Filter by operation type (reboot, reprovision, update, etc.)")
	)

	// Set up network flags to point to the same variable
	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for viewing operations on BrightSign devices.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get operations by serial:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Get operations by ID:\n")
		fmt.Fprintf(os.Stderr, "    %s --id 12345\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Filter by status:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --status completed\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Filter by type:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --type reboot\n", os.Args[0])
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
		fmt.Fprintf(os.Stderr, "Error: Must specify either --serial or --id\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *serialFlag != "" && *idFlag != 0 {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify both --serial and --id\n\n")
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
			log.Fatalf("Configuration error: %v", err)
		}
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Step 1: Authenticate
	if *verboseFlag {
		fmt.Println("Authenticating with BSN.cloud...")
	}
	if err := client.Authenticate(ctx); err != nil {
		if bscontrol.IsAuthenticationError(err) {
			log.Fatalf("Authentication failed: %v", err)
		}
		log.Fatalf("Authentication error: %v", err)
	}
	if *verboseFlag {
		fmt.Println("Authentication successful")
	}

	// Step 2: Set network context
	if err := client.EnsureReady(ctx); err != nil {
		log.Fatalf("Failed to set network context: %v", err)
	}

	currentNetwork, err := client.GetCurrentNetwork(ctx)
	if err != nil {
		log.Fatalf("Failed to get current network: %v", err)
	}

	if *verboseFlag {
		fmt.Printf("Using network: %s\n", currentNetwork.Name)
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Step 3: Get operations
	var operationList *bscontrol.DeviceOperationList
	if *serialFlag != "" {
		if *verboseFlag {
			fmt.Printf("Retrieving operations for device: %s\n", *serialFlag)
		}
		operationList, err = client.Devices.GetOperationsBySerial(ctx, *serialFlag)
	} else {
		if *verboseFlag {
			fmt.Printf("Retrieving operations for device ID: %d\n", *idFlag)
		}
		operationList, err = client.Devices.GetOperations(ctx, *idFlag)
	}

	if err != nil {
		log.Fatalf("Failed to get operations: %v", err)
	}

	// Filter by status and/or type if specified
	var filteredOperations []bscontrol.DeviceOperation
	for _, operation := range operationList.Items {
		// Check status filter
		if *statusFlag != "" && !strings.EqualFold(operation.Status, *statusFlag) {
			continue
		}
		// Check type filter
		if *typeFlag != "" && !strings.EqualFold(operation.OperationType, *typeFlag) {
			continue
		}
		filteredOperations = append(filteredOperations, operation)
	}

	// Output
	if *jsonFlag {
		// JSON output
		output := struct {
			Items      []bscontrol.DeviceOperation `json:"items"`
			TotalCount int                        `json:"totalCount"`
		}{
			Items:      filteredOperations,
			TotalCount: len(filteredOperations),
		}
		jsonData, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			log.Fatalf("Failed to marshal JSON: %v", err)
		}
		fmt.Println(string(jsonData))
	} else {
		// Human-readable output
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Println(strings.Repeat("═", 100))
		fmt.Println("Device Operations")
		fmt.Println(strings.Repeat("═", 100))

		if len(filteredOperations) == 0 {
			fmt.Println("No operations found")
			if *statusFlag != "" || *typeFlag != "" {
				fmt.Print("(filtered by")
				if *statusFlag != "" {
					fmt.Printf(" status: %s", *statusFlag)
				}
				if *typeFlag != "" {
					fmt.Printf(" type: %s", *typeFlag)
				}
				fmt.Println(")")
			}
		} else {
			fmt.Printf("Total Operations: %d", len(filteredOperations))
			if *statusFlag != "" || *typeFlag != "" {
				fmt.Print(" (filtered by")
				if *statusFlag != "" {
					fmt.Printf(" status: %s", *statusFlag)
				}
				if *typeFlag != "" {
					fmt.Printf(" type: %s", *typeFlag)
				}
				fmt.Print(")")
			}
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Println(strings.Repeat("─", 100))

			// Display each operation
			for i, operation := range filteredOperations {
				if i > 0 {
					fmt.Println(strings.Repeat("─", 100))
				}
				fmt.Printf("ID:        %d\n", operation.ID)
				fmt.Printf("Type:      %s\n", formatOperationType(operation.OperationType))
				fmt.Printf("Status:    %s\n", formatStatus(operation.Status))
				fmt.Printf("Created:   %s by %s\n",
					operation.CreatedAt.Format("2006-01-02 15:04:05"), operation.CreatedBy)

				if !operation.StartedAt.IsZero() {
					fmt.Printf("Started:   %s\n", operation.StartedAt.Format("2006-01-02 15:04:05"))
				}

				if !operation.CompletedAt.IsZero() {
					fmt.Printf("Completed: %s\n", operation.CompletedAt.Format("2006-01-02 15:04:05"))
					duration := operation.CompletedAt.Sub(operation.StartedAt)
					fmt.Printf("Duration:  %s\n", duration.Round(time.Second))
				}

				if operation.Progress > 0 {
					fmt.Printf("Progress:  %d%%\n", operation.Progress)
					fmt.Printf("           %s\n", progressBar(operation.Progress, 50))
				}

				if operation.Error != "" {
					fmt.Printf("Error:     %s\n", operation.Error)
				}
			}
		}

		fmt.Println(strings.Repeat("═", 100))
	}
}

func formatOperationType(opType string) string {
	switch strings.ToLower(opType) {
	case "reboot":
		return "🔄 Reboot"
	case "reprovision":
		return "🔧 Reprovision"
	case "update":
		return "⬆️  Update"
	case "screenshot":
		return "📷 Screenshot"
	case "sync":
		return "🔄 Sync"
	default:
		return opType
	}
}

func formatStatus(status string) string {
	switch strings.ToLower(status) {
	case "pending":
		return "⏳ Pending"
	case "in_progress":
		return "▶️  In Progress"
	case "completed":
		return "✅ Completed"
	case "failed":
		return "❌ Failed"
	default:
		return status
	}
}

func progressBar(percentage int, width int) string {
	filled := int(float64(percentage) / 100 * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", empty) + "]"
}
