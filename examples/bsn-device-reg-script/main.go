package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/brightsign/bscontrol"
)

// The five "opt-out" registration keys live in the networking registry section.
// Three are static for the safe Standalone join; the other two are the target
// network name and a freshly minted registration token. Writing these and
// rebooting makes the player self-register over its existing BSN.cloud WebSocket
// without a wipe or content re-sync.
const (
	regKeyBSNCloudEnabled = "bsnce" // static: enable BSN.cloud access
	regKeySetupType       = "sut"   // static: Standalone avoids a wrong default group
	regKeyInheritNetwork  = "inp"   // static: register using existing network config (non-destructive)
	regKeyNetworkName     = "a"     // the network (account) to join
	regKeyRegToken        = "bsnrt" // one-time-use registration token (consumed on success)
	valBSNCloudEnabled    = "true"
	valSetupType          = "Standalone"
	valInheritNetwork     = "yes"
)

func main() {
	var (
		helpFlag    = flag.Bool("help", false, "Display usage information")
		typeFlag    = flag.String("type", "", "Script type to generate: brs (BrightScript) or sh (shell)")
		fileFlag    = flag.String("file", "", "Write the script to this file instead of stdout")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		networkFlag *string
	)

	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s --network <name> --type <brs|sh> [--file <path>]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generate a self-contained player registration script that joins a BrightSign\n")
		fmt.Fprintf(os.Stderr, "player to a BSN.cloud network by writing the five 'opt-out' registry keys and\n")
		fmt.Fprintf(os.Stderr, "rebooting. A fresh registration token for the network is minted and embedded.\n\n")
		fmt.Fprintf(os.Stderr, "The script is written to STDOUT (or to --file); all progress and diagnostics\n")
		fmt.Fprintf(os.Stderr, "go to STDERR.\n\n")
		fmt.Fprintf(os.Stderr, "Script types:\n")
		fmt.Fprintf(os.Stderr, "  brs  BrightScript (roRegistrySection) - run on the player via autorun/debugger\n")
		fmt.Fprintf(os.Stderr, "  sh   Shell script using the 'registry' CLI - for INSECURE players, run from\n")
		fmt.Fprintf(os.Stderr, "       the player shell, e.g. copy to the SD card and run:\n")
		fmt.Fprintf(os.Stderr, "         bash /storage/sd/register-player.sh\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  --network, -n <name>  Network name to join (required; overrides BS_NETWORK)\n")
		fmt.Fprintf(os.Stderr, "  --type <brs|sh>       Script type to generate (required)\n")
		fmt.Fprintf(os.Stderr, "  --file <path>         Write the script to this file instead of stdout\n")
		fmt.Fprintf(os.Stderr, "  --timeout <seconds>   Request timeout in seconds (default 30)\n")
		fmt.Fprintf(os.Stderr, "  --help                Display usage information\n")
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET           BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK          BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s --network \"My Network\" --type brs --file register-player.brs\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --network \"My Network\" --type sh  > register-player.sh\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Precondition: a valid script type is required; there is no sensible default.
	scriptType := strings.ToLower(*typeFlag)
	if scriptType != "brs" && scriptType != "sh" {
		fmt.Fprintf(os.Stderr, "❌ --type is required and must be 'brs' or 'sh' (got %q)\n\n", *typeFlag)
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

	fmt.Fprintf(os.Stderr, "🔧 Creating BSN.cloud client...\n")
	client, err := bscontrol.New(opts...)
	if err != nil {
		if bscontrol.IsConfigurationError(err) {
			log.Fatalf("❌ Configuration error: %v", err)
		}
		log.Fatalf("❌ Failed to create client: %v", err)
	}

	ctx := context.Background()

	fmt.Fprintf(os.Stderr, "🔐 Authenticating with BSN.cloud...\n")
	if err := client.Authenticate(ctx); err != nil {
		if bscontrol.IsAuthenticationError(err) {
			log.Fatalf("❌ Authentication failed: %v", err)
		}
		log.Fatalf("❌ Authentication error: %v", err)
	}
	fmt.Fprintf(os.Stderr, "✅ Authentication successful!\n")

	networkName := getNetworkName(*networkFlag, client, ctx)

	// Precondition: the network name is embedded into a shell/BrightScript string
	// literal, so reject characters that would break that literal rather than
	// emitting a script that silently misbehaves.
	assertEmbeddable("network name", networkName)

	fmt.Fprintf(os.Stderr, "📡 Setting network context to: %s\n", networkName)
	if err := client.BDeploy.SetNetworkContext(ctx, networkName); err != nil {
		log.Fatalf("❌ Failed to set network context: %v", err)
	}

	fmt.Fprintf(os.Stderr, "🔑 Generating device registration token...\n")
	deviceToken, err := client.Provisioning.GenerateDeviceToken(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to generate device registration token: %v", err)
	}
	assertEmbeddable("registration token", deviceToken.Token)
	fmt.Fprintf(os.Stderr, "✅ Token generated (valid %s to %s)\n", deviceToken.ValidFrom, deviceToken.ValidTo)

	var script string
	switch scriptType {
	case "brs":
		script = renderBrightScript(networkName, deviceToken.Token, deviceToken.ValidTo)
	case "sh":
		script = renderShellScript(networkName, deviceToken.Token, deviceToken.ValidTo)
	}

	// With --file the script goes to that file; otherwise it is the only thing on
	// stdout so the caller can redirect it.
	if *fileFlag != "" {
		if err := os.WriteFile(*fileFlag, []byte(script), 0o644); err != nil {
			log.Fatalf("❌ Failed to write script to %s: %v", *fileFlag, err)
		}
		fmt.Fprintf(os.Stderr, "✅ Wrote %s registration script to %s\n", scriptType, *fileFlag)
		return
	}
	fmt.Print(script)
	fmt.Fprintf(os.Stderr, "✅ Wrote %s registration script to stdout.\n", scriptType)
}

// renderBrightScript emits a BrightScript that writes the five networking keys and
// reboots. The entry point is Sub Main(), the BrightSign autorun convention.
func renderBrightScript(networkName, token, validTo string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "' Auto-generated by bsn-device-reg-script\n")
	fmt.Fprintf(&b, "' Joins this player to BSN.cloud network %q by writing registration keys.\n", networkName)
	fmt.Fprintf(&b, "' Token valid until %s. bsnrt is one-time-use and is consumed on registration.\n", validTo)
	fmt.Fprintf(&b, "Sub Main()\n")
	fmt.Fprintf(&b, "    reg = CreateObject(\"roRegistrySection\", \"networking\")\n")
	fmt.Fprintf(&b, "    reg.Write(%q, %q)\n", regKeyBSNCloudEnabled, valBSNCloudEnabled)
	fmt.Fprintf(&b, "    reg.Write(%q, %q)\n", regKeyNetworkName, networkName)
	fmt.Fprintf(&b, "    reg.Write(%q, %q)\n", regKeyRegToken, token)
	fmt.Fprintf(&b, "    reg.Write(%q, %q)\n", regKeySetupType, valSetupType)
	fmt.Fprintf(&b, "    reg.Write(%q, %q)\n", regKeyInheritNetwork, valInheritNetwork)
	fmt.Fprintf(&b, "    reg.Flush()\n")
	fmt.Fprintf(&b, "    RebootSystem()\n")
	fmt.Fprintf(&b, "End Sub\n")
	return b.String()
}

// renderShellScript emits a bash script using the on-player 'registry' CLI. set -e
// aborts before the reboot if any write fails, so a half-configured player is never
// rebooted. Intended for insecure players run from the player shell.
func renderShellScript(networkName, token, validTo string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "#!/bin/bash\n")
	fmt.Fprintf(&b, "# Auto-generated by bsn-device-reg-script\n")
	fmt.Fprintf(&b, "# Joins this player to BSN.cloud network %q by writing registration keys.\n", networkName)
	fmt.Fprintf(&b, "# Token valid until %s. bsnrt is one-time-use and is consumed on registration.\n", validTo)
	fmt.Fprintf(&b, "# Run on an INSECURE player from the shell, e.g.:\n")
	fmt.Fprintf(&b, "#   bash /storage/sd/register-player.sh\n")
	fmt.Fprintf(&b, "set -e\n")
	fmt.Fprintf(&b, "registry networking %s %s\n", regKeyBSNCloudEnabled, valBSNCloudEnabled)
	fmt.Fprintf(&b, "registry networking %s %q\n", regKeyNetworkName, networkName)
	fmt.Fprintf(&b, "registry networking %s %q\n", regKeyRegToken, token)
	fmt.Fprintf(&b, "registry networking %s %s\n", regKeySetupType, valSetupType)
	fmt.Fprintf(&b, "registry networking %s %s\n", regKeyInheritNetwork, valInheritNetwork)
	fmt.Fprintf(&b, "reboot\n")
	return b.String()
}

// assertEmbeddable fails fast if a value cannot be safely embedded in a shell or
// BrightScript double-quoted string literal. These characters would otherwise break
// the generated script or change its meaning.
func assertEmbeddable(label, value string) {
	const unsafe = "\"`$\\\n\r"
	if strings.ContainsAny(value, unsafe) {
		log.Fatalf("❌ %s %q contains characters that cannot be embedded in a generated script (one of %q)", label, value, unsafe)
	}
}

func getNetworkName(requestedNetwork string, client *bscontrol.Client, ctx context.Context) string {
	if requestedNetwork != "" {
		return requestedNetwork
	}

	if client.IsNetworkSet() {
		if current, err := client.GetCurrentNetwork(ctx); err == nil {
			return current.Name
		}
	}

	if envNetwork := os.Getenv("BS_NETWORK"); envNetwork != "" {
		return envNetwork
	}

	fmt.Fprintf(os.Stderr, "📡 Getting available networks...\n")
	networks, err := client.GetNetworks(ctx)
	if err != nil {
		log.Fatalf("❌ Failed to get networks: %v", err)
	}
	if len(networks) == 0 {
		log.Fatalf("❌ No networks available")
	}
	if len(networks) == 1 {
		fmt.Fprintf(os.Stderr, "✅ Using only available network: %s\n", networks[0].Name)
		return networks[0].Name
	}

	fmt.Fprintf(os.Stderr, "❌ Multiple networks available. Please specify --network or set BS_NETWORK:\n")
	for i, network := range networks {
		fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
	}
	os.Exit(1)
	return ""
}
