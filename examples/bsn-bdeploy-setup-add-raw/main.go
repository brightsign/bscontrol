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
	"github.com/brightsign/bscontrol/examples/internal/setuptemplate"
)

func main() {
	var (
		helpFlag        = flag.Bool("help", false, "Display usage information")
		verboseFlag     = flag.Bool("verbose", false, "Show detailed information")
		jsonFlag        = flag.Bool("json", false, "Output as JSON")
		timeoutFlag     = flag.Int("timeout", 30, "Request timeout in seconds")
		templateFlag    = flag.String("template", setuptemplate.DefaultTemplate, "Path to setup template file")
		packageNameFlag = flag.String("package-name", "", "Package name for the setup (required in template mode, env: BS_PACKAGE_NAME)")
		deviceNameFlag  = flag.String("device-name", "", "Device name (optional, env: BS_DEVICE_NAME)")
		deviceDescFlag  = flag.String("device-description", "", "Device description (optional, env: BS_DEVICE_DESCRIPTION)")
		networkFlag     = flag.String("network", "", "Network name (env: BS_NETWORK)")
		usernameFlag    = flag.String("username", "", "BSN.cloud username (env: BS_CLIENT_ID)")
		groupFlag       = flag.String("group", "Default", "BSN group name")
		setupTypeFlag   = flag.String("setup-type", "bsn", "Setup type (bsn, standalone, lfn)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [config.json]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to create a B-Deploy setup record using raw JSON.\n")
		fmt.Fprintf(os.Stderr, "Posts the rendered template directly to the API, preserving all fields.\n\n")
		fmt.Fprintf(os.Stderr, "Two modes of operation:\n")
		fmt.Fprintf(os.Stderr, "  Template mode: Use --package-name (or BS_PACKAGE_NAME) with the setup template\n")
		fmt.Fprintf(os.Stderr, "  Config mode:   Pass a JSON configuration file as a positional argument\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID           BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET              BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK             BSN.cloud network name\n")
		fmt.Fprintf(os.Stderr, "  BS_PACKAGE_NAME        Setup package name\n")
		fmt.Fprintf(os.Stderr, "  BS_DEVICE_NAME         Device name\n")
		fmt.Fprintf(os.Stderr, "  BS_DEVICE_DESCRIPTION  Device description\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Create setup using template:\n")
		fmt.Fprintf(os.Stderr, "    %s --package-name \"retail-v1\" --network \"Production\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Create setup with full template (preserves firmware entries):\n")
		fmt.Fprintf(os.Stderr, "    %s --template DefaultSetupPackageTemplateMaster.json --package-name \"retail-v1\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Create setup from raw JSON file:\n")
		fmt.Fprintf(os.Stderr, "    %s config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --json --package-name \"retail-v1\"\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Resolve environment variables
	resolvedPackageName := setuptemplate.ResolveVar(*packageNameFlag, setuptemplate.EnvPackageName)
	resolvedDeviceName := setuptemplate.ResolveVar(*deviceNameFlag, setuptemplate.EnvDeviceName)
	resolvedDeviceDesc := setuptemplate.ResolveVar(*deviceDescFlag, setuptemplate.EnvDeviceDescription)
	resolvedNetwork := setuptemplate.ResolveVar(*networkFlag, "BS_NETWORK")
	resolvedUsername := setuptemplate.ResolveVar(*usernameFlag, "BS_CLIENT_ID")

	// Determine mode and get setup JSON
	var setupJSON string

	if flag.NArg() == 1 {
		// Config file mode: read raw JSON from file
		configFile := flag.Arg(0)
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "Loading raw JSON from: %s\n", configFile)
		}
		data, err := os.ReadFile(configFile)
		if err != nil {
			log.Fatalf("Failed to read config file: %v", err)
		}
		if !json.Valid(data) {
			log.Fatalf("Config file is not valid JSON: %s", configFile)
		}
		setupJSON = string(data)
	} else if flag.NArg() == 0 && resolvedPackageName != "" {
		// Template mode: render template to raw JSON string
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "Rendering setup from template: %s\n", *templateFlag)
		}

		vars := setuptemplate.TemplateVars{
			Username:          resolvedUsername,
			NetworkName:       resolvedNetwork,
			PackageName:       resolvedPackageName,
			SetupType:         *setupTypeFlag,
			GroupName:         *groupFlag,
			DeviceName:        resolvedDeviceName,
			DeviceDescription: resolvedDeviceDesc,
		}

		var err error
		setupJSON, err = setuptemplate.RenderRaw(*templateFlag, vars)
		if err != nil {
			log.Fatalf("Failed to render template: %v", err)
		}
	} else {
		fmt.Fprintf(os.Stderr, "Error: provide a config file or use --package-name (env: BS_PACKAGE_NAME)\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if *verboseFlag && !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Rendered JSON:\n%s\n\n", setupJSON)
	}

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Creating BSN.cloud client...\n")
	}
	client, err := bscontrol.New(opts...)
	if err != nil {
		if bscontrol.IsConfigurationError(err) {
			log.Fatalf("Configuration error: %v", err)
		}
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Authenticate
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Authenticating with BSN.cloud...\n")
	}
	if err := client.Authenticate(ctx); err != nil {
		if bscontrol.IsAuthenticationError(err) {
			log.Fatalf("Authentication failed: %v", err)
		}
		log.Fatalf("Authentication error: %v", err)
	}
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Authentication successful!\n")
	}

	// Set network context using the network from the JSON or flag
	networkName := resolvedNetwork
	if networkName == "" {
		// Try to extract from the JSON
		var parsed struct {
			BDeploy struct {
				NetworkName string `json:"networkName"`
			} `json:"bDeploy"`
		}
		if err := json.Unmarshal([]byte(setupJSON), &parsed); err == nil && parsed.BDeploy.NetworkName != "" {
			networkName = parsed.BDeploy.NetworkName
		}
	}

	if networkName != "" {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "Setting network context to: %s\n", networkName)
		}
		if err := client.BDeploy.SetNetworkContext(ctx, networkName); err != nil {
			log.Fatalf("Failed to set network context: %v", err)
		}
	}

	// Generate device registration token if the JSON has empty/placeholder token
	var parsed struct {
		BSNDeviceRegistrationTokenEntity *struct {
			Token string `json:"token"`
		} `json:"bsnDeviceRegistrationTokenEntity"`
	}
	if err := json.Unmarshal([]byte(setupJSON), &parsed); err == nil {
		needsToken := parsed.BSNDeviceRegistrationTokenEntity == nil ||
			parsed.BSNDeviceRegistrationTokenEntity.Token == "" ||
			strings.HasPrefix(parsed.BSNDeviceRegistrationTokenEntity.Token, "{{")

		if needsToken {
			if !*jsonFlag {
				fmt.Fprintf(os.Stderr, "Generating device registration token...\n")
			}
			deviceToken, err := client.Provisioning.GenerateDeviceToken(ctx)
			if err != nil {
				log.Fatalf("Failed to generate device token: %v", err)
			}

			// Inject the token into the raw JSON
			var rawMap map[string]interface{}
			if err := json.Unmarshal([]byte(setupJSON), &rawMap); err != nil {
				log.Fatalf("Failed to parse setup JSON for token injection: %v", err)
			}
			rawMap["bsnDeviceRegistrationTokenEntity"] = map[string]interface{}{
				"token":     deviceToken.Token,
				"scope":     deviceToken.Scope,
				"validFrom": deviceToken.ValidFrom,
				"validTo":   deviceToken.ValidTo,
			}
			injected, err := json.Marshal(rawMap)
			if err != nil {
				log.Fatalf("Failed to re-marshal setup JSON: %v", err)
			}
			setupJSON = string(injected)

			if !*jsonFlag {
				fmt.Fprintf(os.Stderr, "Device registration token generated!\n")
			}
			if *verboseFlag && !*jsonFlag {
				fmt.Fprintf(os.Stderr, "   Token: %s...\n", deviceToken.Token[:32])
				fmt.Fprintf(os.Stderr, "   Valid from: %s\n", deviceToken.ValidFrom)
				fmt.Fprintf(os.Stderr, "   Valid to: %s\n", deviceToken.ValidTo)
			}
		}
	}

	// Create B-Deploy setup record using raw JSON
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Creating B-Deploy setup record (raw)...\n")
	}

	response, err := client.BDeploy.AddSetupRecordRaw(ctx, setupJSON)
	if err != nil {
		log.Fatalf("Failed to create B-Deploy setup record: %v", err)
	}

	// Display results
	if *jsonFlag {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(response); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "B-Deploy setup record created successfully!\n")

	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "\nResponse Debug:\n")
		fmt.Fprintf(os.Stderr, "   ID: '%s'\n", response.ID)
		fmt.Fprintf(os.Stderr, "   Success: %v\n", response.Success)
		fmt.Fprintf(os.Stderr, "   Error: '%s'\n", response.Error)
	}

	if response.Error != "" {
		fmt.Fprintf(os.Stderr, "   Warning: API Error: %s\n", response.Error)
	}

	fmt.Fprintf(os.Stderr, "\n%s\n", strings.Repeat("=", 70))
	fmt.Fprintf(os.Stderr, "SETUP-ID: %s\n", response.ID)
	fmt.Fprintf(os.Stderr, "%s\n", strings.Repeat("=", 70))
	fmt.Fprintf(os.Stderr, "Save this setup-id - you'll need it to associate players with this setup\n")

	fmt.Fprintf(os.Stderr, "\nSetup URLs:\n")
	fmt.Fprintf(os.Stderr, "   Web: https://provision.bsn.cloud/setup/%s\n", response.ID)
	fmt.Fprintf(os.Stderr, "   API: https://provision.bsn.cloud/rest-setup/v3/setup/%s\n", response.ID)
	fmt.Fprintf(os.Stderr, "\n")
}
