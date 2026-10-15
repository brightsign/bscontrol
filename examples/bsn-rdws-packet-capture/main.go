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
		jsonFlag     = flag.Bool("json", false, "Output raw JSON response")
		timeoutFlag  = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag  *string
		serialFlag   = flag.String("serial", "", "Device serial number (required)")
		statusFlag   = flag.Bool("status", false, "Get packet capture status")
		startFlag    = flag.Bool("start", false, "Start packet capture")
		stopFlag     = flag.Bool("stop", false, "Stop packet capture")
		ifaceFlag    = flag.String("interface", "eth0", "Network interface for capture (default: eth0)")
		durationFlag = flag.Int("duration", 60, "Capture duration in seconds (default: 60)")
		filterFlag   = flag.String("filter", "", "tcpdump filter expression (optional)")
	)

	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for packet capture on players via rDWS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Check capture status:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --status\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Start capture:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --start --interface eth0 --duration 60\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Start capture with filter:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --start --filter \"port 80\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Stop capture:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial UTD41X000009 --stop\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	if *serialFlag == "" || (!*statusFlag && !*startFlag && !*stopFlag) {
		fmt.Fprintf(os.Stderr, "Error: Must specify --serial and one of --status, --start, or --stop\n\n")
		flag.Usage()
		os.Exit(1)
	}

	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}
	if *networkFlag != "" {
		opts = append(opts, bscontrol.WithNetwork(*networkFlag))
	}

	client, err := bscontrol.New(opts...)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Authenticating with BSN.cloud...\n")
	}
	if err := client.Authenticate(ctx); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	if err := handleNetworkSelection(ctx, client, *networkFlag, *jsonFlag); err != nil {
		log.Fatalf("Network selection failed: %v", err)
	}

	if *statusFlag {
		if !*jsonFlag {
			fmt.Printf("Getting packet capture status for device %s...\n", *serialFlag)
		}

		status, err := client.RDWS.GetPacketCaptureStatus(ctx, *serialFlag)
		if err != nil {
			log.Fatalf("Failed to get packet capture status: %v", err)
		}

		if *jsonFlag {
			jsonData, err := json.MarshalIndent(status, "", "  ")
			if err != nil {
				log.Fatalf("Failed to marshal JSON: %v", err)
			}
			fmt.Println(string(jsonData))
		} else {
			displayCaptureStatus(status)
		}
	}

	if *startFlag {
		if !*jsonFlag {
			fmt.Printf("Starting packet capture on %s for %d seconds...\n", *ifaceFlag, *durationFlag)
		}

		request := &bscontrol.RDWSPacketCaptureStartRequest{}
		request.Data.Interface = *ifaceFlag
		request.Data.Duration = *durationFlag
		if *filterFlag != "" {
			request.Data.Filter = *filterFlag
		}

		filePath, err := client.RDWS.StartPacketCapture(ctx, *serialFlag, request)
		if err != nil {
			log.Fatalf("Failed to start packet capture: %v", err)
		}

		fmt.Printf("✓ Packet capture started\n")
		if filePath != "" {
			fmt.Printf("  Capture file: %s\n", filePath)
		}
	}

	if *stopFlag {
		if !*jsonFlag {
			fmt.Println("Stopping packet capture...")
		}

		filePath, err := client.RDWS.StopPacketCapture(ctx, *serialFlag)
		if err != nil {
			log.Fatalf("Failed to stop packet capture: %v", err)
		}

		fmt.Printf("✓ Packet capture stopped\n")
		if filePath != "" {
			fmt.Printf("  Capture file: %s\n", filePath)
		}
	}
}

func handleNetworkSelection(ctx context.Context, client *bscontrol.Client, requestedNetwork string, jsonMode bool) error {
	if client.IsNetworkSet() {
		if current, err := client.GetCurrentNetwork(ctx); err == nil {
			if !jsonMode {
				fmt.Fprintf(os.Stderr, "Using network: %s (ID: %d)\n", current.Name, current.ID)
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

	networks, err := client.GetNetworks(ctx)
	if err != nil {
		return fmt.Errorf("failed to get networks: %w", err)
	}

	if len(networks) == 0 {
		return fmt.Errorf("no networks available")
	}

	if requestedNetwork != "" {
		for _, network := range networks {
			if strings.EqualFold(network.Name, requestedNetwork) {
				if !jsonMode {
					fmt.Printf("Using network: %s (ID: %d)\n", network.Name, network.ID)
				}
				return client.SetNetworkByID(ctx, network.ID)
			}
		}
	}

	if len(networks) == 1 {
		if !jsonMode {
			fmt.Printf("Using network: %s (ID: %d)\n", networks[0].Name, networks[0].ID)
		}
		return client.SetNetworkByID(ctx, networks[0].ID)
	}

	fmt.Fprintf(os.Stderr, "Available networks:\n")
	for i, network := range networks {
		fmt.Printf("  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
	}

	fmt.Fprint(os.Stderr, "Select network (1-"+strconv.Itoa(len(networks))+"): ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	selection, err := strconv.Atoi(strings.TrimSpace(scanner.Text()))
	if err != nil || selection < 1 || selection > len(networks) {
		return fmt.Errorf("invalid selection")
	}

	selectedNetwork := networks[selection-1]
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "Selected network: %s (ID: %d)\n", selectedNetwork.Name, selectedNetwork.ID)
	}
	return client.SetNetworkByID(ctx, selectedNetwork.ID)
}

func displayCaptureStatus(status *bscontrol.RDWSPacketCaptureStatus) {
	fmt.Fprintf(os.Stderr, "\n=== Packet Capture Status ===\n")
	if status.Running {
		fmt.Fprintf(os.Stderr, "Status:      Running\n")
		fmt.Fprintf(os.Stderr, "Interface:   %s\n", status.Interface)
		fmt.Fprintf(os.Stderr, "Duration:    %d seconds\n", status.Duration)
		if status.StartTime != "" {
			fmt.Fprintf(os.Stderr, "Started:     %s\n", status.StartTime)
		}
		if status.ElapsedTime > 0 {
			fmt.Fprintf(os.Stderr, "Elapsed:     %d seconds\n", status.ElapsedTime)
		}
		if status.FilePath != "" {
			fmt.Fprintf(os.Stderr, "File:        %s\n", status.FilePath)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Status:      Not running\n")
	}
	fmt.Fprintf(os.Stderr, "\n")
}
