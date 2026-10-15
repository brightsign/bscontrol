package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"text/template"
	"time"

	"github.com/brightsign/bscontrol"
	"github.com/brightsign/bscontrol/examples/internal/setuptemplate"
)

func main() {
	var (
		helpFlag        = flag.Bool("help", false, "Display usage information")
		verboseFlag     = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag     = flag.Int("timeout", 30, "Request timeout in seconds")
		usernameFlag    = flag.String("username", "", "BSN.cloud username / email for bDeploy.username (env: BS_USERNAME)")
		networkFlag     = flag.String("network", "", "Network name (env: BS_NETWORK)")
		packageNameFlag = flag.String("package-name", "", "Package name (required)")
		setupTypeFlag   = flag.String("setup-type", "", "Setup type: bsn, standalone, lfn (env: BS_SETUP_TYPE, default: bsn)")
		deviceNameFlag  = flag.String("device-name", "", "Device name")
		deviceDescFlag  = flag.String("device-description", "", "Device description")
		groupFlag       = flag.String("group", "", "BSN group name (env: BS_GROUP_NAME, default: Default)")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <template.json>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Render a Go-template setup JSON file by substituting template variables\n")
		fmt.Fprintf(os.Stderr, "from CLI flags / environment variables and generating a fresh device\n")
		fmt.Fprintf(os.Stderr, "registration token. Outputs the complete setup JSON to stdout.\n\n")
		fmt.Fprintf(os.Stderr, "The output is ready for use with bdeploy-add-setup-raw or AddSetupRecordRaw.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID   BSN.cloud API client ID (required, for auth)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET      BSN.cloud API client secret (required, for auth)\n")
		fmt.Fprintf(os.Stderr, "  BS_USERNAME    BSN.cloud username/email for bDeploy.username (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK     Network name (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SETUP_TYPE  Setup type (default: bsn)\n")
		fmt.Fprintf(os.Stderr, "  BS_GROUP_NAME  BSN group name (default: Default)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Render template to stdout:\n")
		fmt.Fprintf(os.Stderr, "    %s --package-name \"retail-v1\" --network \"Production\" DefaultSetupPackageTemplateMaster.json\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Save rendered setup to file:\n")
		fmt.Fprintf(os.Stderr, "    %s --package-name \"retail-v1\" template.json > setup.json\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Use with bdeploy-add-setup-raw:\n")
		fmt.Fprintf(os.Stderr, "    %s --package-name \"retail-v1\" template.json | bdeploy-add-setup-raw /dev/stdin\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "Error: exactly one template file argument is required\n\n")
		flag.Usage()
		os.Exit(1)
	}
	templateFile := flag.Arg(0)

	// Resolve variables: CLI flag -> env var (where applicable) -> default
	resolvedUsername := setuptemplate.ResolveVar(*usernameFlag, "BS_USERNAME")
	resolvedNetwork := setuptemplate.ResolveVar(*networkFlag, "BS_NETWORK")
	resolvedPackageName := *packageNameFlag
	resolvedSetupType := setuptemplate.ResolveVar(*setupTypeFlag, "BS_SETUP_TYPE")
	resolvedDeviceName := *deviceNameFlag
	resolvedDeviceDesc := *deviceDescFlag
	resolvedGroup := setuptemplate.ResolveVar(*groupFlag, "BS_GROUP_NAME")

	// Apply defaults for optional fields
	if resolvedSetupType == "" {
		resolvedSetupType = "bsn"
	}
	if resolvedGroup == "" {
		resolvedGroup = "Default"
	}

	// Validate required fields
	if resolvedNetwork == "" {
		fmt.Fprintf(os.Stderr, "Error: network is required (use --network or set BS_NETWORK)\n")
		os.Exit(1)
	}
	if resolvedPackageName == "" {
		fmt.Fprintf(os.Stderr, "Error: package-name is required (use --package-name or set BS_PACKAGE_NAME)\n")
		os.Exit(1)
	}
	if resolvedUsername == "" {
		fmt.Fprintf(os.Stderr, "Error: username is required (use --username or set BS_USERNAME)\n")
		os.Exit(1)
	}

	// Read the template file
	fmt.Fprintf(os.Stderr, "Reading template: %s\n", templateFile)
	templateData, err := os.ReadFile(templateFile)
	if err != nil {
		log.Fatalf("Failed to read template file: %v", err)
	}

	// Create client and authenticate
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}

	fmt.Fprintf(os.Stderr, "Authenticating with BSN.cloud...\n")
	client, err := bscontrol.New(opts...)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	if err := client.Authenticate(ctx); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Authentication successful.\n")

	// Set network context
	fmt.Fprintf(os.Stderr, "Setting network context to: %s\n", resolvedNetwork)
	if err := client.BDeploy.SetNetworkContext(ctx, resolvedNetwork); err != nil {
		log.Fatalf("Failed to set network context: %v", err)
	}

	// Generate device registration token
	fmt.Fprintf(os.Stderr, "Generating device registration token...\n")
	deviceToken, err := client.Provisioning.GenerateDeviceToken(ctx)
	if err != nil {
		log.Fatalf("Failed to generate device token: %v", err)
	}
	fmt.Fprintf(os.Stderr, "Token generated successfully.\n")

	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "  Token:      %s...%s\n", deviceToken.Token[:16], deviceToken.Token[len(deviceToken.Token)-16:])
		fmt.Fprintf(os.Stderr, "  Scope:      %s\n", deviceToken.Scope)
		fmt.Fprintf(os.Stderr, "  Valid from: %s\n", deviceToken.ValidFrom)
		fmt.Fprintf(os.Stderr, "  Valid to:   %s\n", deviceToken.ValidTo)
	}

	// Build template variables
	vars := setuptemplate.TemplateVars{
		Username:          resolvedUsername,
		NetworkName:       resolvedNetwork,
		PackageName:       resolvedPackageName,
		SetupType:         resolvedSetupType,
		RegistrationToken: deviceToken.Token,
		TokenValidFrom:    deviceToken.ValidFrom,
		TokenValidTo:      deviceToken.ValidTo,
		DeviceName:        resolvedDeviceName,
		DeviceDescription: resolvedDeviceDesc,
		GroupName:         resolvedGroup,
	}

	if *verboseFlag {
		fmt.Fprintf(os.Stderr, "Template variables:\n")
		fmt.Fprintf(os.Stderr, "  Username:          %s\n", vars.Username)
		fmt.Fprintf(os.Stderr, "  NetworkName:       %s\n", vars.NetworkName)
		fmt.Fprintf(os.Stderr, "  PackageName:       %s\n", vars.PackageName)
		fmt.Fprintf(os.Stderr, "  SetupType:         %s\n", vars.SetupType)
		fmt.Fprintf(os.Stderr, "  DeviceName:        %s\n", vars.DeviceName)
		fmt.Fprintf(os.Stderr, "  DeviceDescription: %s\n", vars.DeviceDescription)
		fmt.Fprintf(os.Stderr, "  GroupName:         %s\n", vars.GroupName)
	}

	// Execute the template
	fmt.Fprintf(os.Stderr, "Rendering template...\n")
	tmpl, err := template.New("setup").Parse(string(templateData))
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}

	// Validate the rendered output is valid JSON
	if !json.Valid(buf.Bytes()) {
		log.Fatalf("Rendered template is not valid JSON")
	}

	// Pretty-print the JSON to stdout
	var prettyBuf bytes.Buffer
	if err := json.Indent(&prettyBuf, buf.Bytes(), "", "  "); err != nil {
		log.Fatalf("Failed to format JSON: %v", err)
	}

	fmt.Fprintln(os.Stdout, prettyBuf.String())
	fmt.Fprintf(os.Stderr, "Setup JSON rendered successfully.\n")
}
