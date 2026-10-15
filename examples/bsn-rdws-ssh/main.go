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
		debugFlag    = flag.Bool("debug", false, "Enable debug logging of HTTP requests/responses")
		timeoutFlag  = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag  *string
		serialFlag   = flag.String("serial", "", "Device serial number (required)")
		getFlag      = flag.Bool("get", false, "Get current SSH status")
		enableFlag   = flag.Bool("enable", false, "Enable SSH")
		disableFlag  = flag.Bool("disable", false, "Disable SSH")
		portFlag     = flag.Int("port", 22, "SSH port (default: 22)")
		passwordFlag = flag.String("password", "", "SSH password (optional, if not set existing password is not changed)")
	)

	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A command-line tool for managing SSH access on BrightSign players via Remote DWS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get SSH status:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --get\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Enable SSH with default port and password:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --enable --password mypassword\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Enable SSH on custom port:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --enable --port 2222 --password secure123\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Enable SSH without changing password:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --enable\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Disable SSH:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --disable\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Change SSH password without changing enabled status:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --enable --password newpassword\n", os.Args[0])
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

	if !*getFlag && !*enableFlag && !*disableFlag {
		fmt.Fprintf(os.Stderr, "Error: Must specify --get, --enable, or --disable\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *enableFlag && *disableFlag {
		fmt.Fprintf(os.Stderr, "Error: Cannot specify both --enable and --disable\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *getFlag && (*enableFlag || *disableFlag) {
		fmt.Fprintf(os.Stderr, "Error: Cannot use --get with --enable or --disable\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Get network name from flag or environment variable
	networkName := *networkFlag
	if networkName == "" {
		networkName = os.Getenv("BS_NETWORK")
	}

	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}
	if networkName != "" {
		opts = append(opts, bscontrol.WithNetwork(networkName))
	}
	if *debugFlag {
		opts = append(opts, bscontrol.WithDebug(true))
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

	if err := handleNetworkSelection(ctx, client, networkName, *jsonFlag); err != nil {
		log.Fatalf("Network selection failed: %v", err)
	}

	if *getFlag {
		if !*jsonFlag {
			fmt.Printf("Getting SSH status for device %s...\n", *serialFlag)
		}

		info, err := client.RDWS.GetSSHStatus(ctx, *serialFlag)
		if err != nil {
			log.Fatalf("Failed to get SSH status: %v", err)
		}

		if *jsonFlag {
			jsonData, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				log.Fatalf("Failed to marshal JSON: %v", err)
			}
			fmt.Println(string(jsonData))
		} else {
			displaySSHInfo(info)
		}
	}

	if *enableFlag || *disableFlag {
		enabled := *enableFlag

		if !*jsonFlag {
			if enabled {
				if *passwordFlag != "" {
					fmt.Printf("Enabling SSH on port %d with password...\n", *portFlag)
				} else {
					fmt.Printf("Enabling SSH on port %d (password unchanged)...\n", *portFlag)
				}
			} else {
				fmt.Println("Disabling SSH...")
			}
		}

		success, err := client.RDWS.SetSSHStatus(ctx, *serialFlag, enabled, *portFlag, *passwordFlag)
		if err != nil {
			log.Fatalf("Failed to set SSH status: %v", err)
		}

		if !*jsonFlag {
			if success {
				fmt.Println("✅ SSH setting updated successfully")
				if enabled && *passwordFlag != "" {
					fmt.Println("   SSH password has been set")
				}
			} else {
				if *debugFlag {
					fmt.Println("❌ SSH setting update failed - check debug output above for details")
				} else {
					fmt.Println("❌ SSH setting update failed - add --debug to see detailed HTTP request/response logs")
				}
			}
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

func displaySSHInfo(info *bscontrol.RDWSSSHInfo) {
	fmt.Fprintf(os.Stderr, "\n=== SSH Status ===\n")
	if info.Enabled {
		fmt.Fprintf(os.Stderr, "Status:  Enabled\n")
		if info.Port > 0 {
			fmt.Fprintf(os.Stderr, "Port:    %d\n", info.Port)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Status:  Disabled\n")
	}
	fmt.Fprintf(os.Stderr, "\n")
}
