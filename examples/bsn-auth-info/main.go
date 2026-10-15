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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	var (
		helpFlag    = flag.Bool("help", false, "Display usage information")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		showFlag    = flag.Bool("show", false, "Show current token from .token file without re-authenticating")
	)

	// Custom usage output
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to authenticate with BSN.cloud and display curl command examples.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Get authentication info and curl examples:\n")
		fmt.Fprintf(os.Stderr, "    %s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Show current token from .token file:\n")
		fmt.Fprintf(os.Stderr, "    %s --show\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --json\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag {
		flag.Usage()
		return
	}

	// Handle show flag
	if *showFlag {
		tokenFile := "./.token"
		if content, err := os.ReadFile(tokenFile); err == nil {
			if *jsonFlag {
				result := map[string]interface{}{
					"tokenFile": tokenFile,
					"content":   string(content),
				}
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				if err := encoder.Encode(result); err != nil {
					log.Fatalf("Failed to encode JSON: %v", err)
				}
				return
			}
			fmt.Fprintf(os.Stderr, "🔐 Current token file content (%s):\n", tokenFile)
			fmt.Fprint(os.Stderr, string(content))
			fmt.Fprintf(os.Stderr, "\n")
			fmt.Fprintf(os.Stderr, "💡 To use this token:\n")
			fmt.Fprintf(os.Stderr, "   source %s\n", tokenFile)
			fmt.Fprintf(os.Stderr, "   echo $BSN_ACCESS_TOKEN\n")
		} else {
			if !*jsonFlag {
				fmt.Fprintf(os.Stderr, "❌ No token file found at %s\n", tokenFile)
				fmt.Fprintf(os.Stderr, "💡 Run '%s' (without --show) to authenticate and create the token file.\n", os.Args[0])
			}
		}
		return
	}

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔧 Creating BSN.cloud client...\n")
	}

	client, err := bscontrol.New(opts...)
	if err != nil {
		if bscontrol.IsConfigurationError(err) {
			log.Fatalf("❌ Configuration error: %v", err)
		}
		log.Fatalf("❌ Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Authenticate
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔐 Authenticating with BSN.cloud...\n")
	}

	if err := client.Authenticate(ctx); err != nil {
		if bscontrol.IsAuthenticationError(err) {
			log.Fatalf("❌ Authentication failed: %v", err)
		}
		log.Fatalf("❌ Authentication error: %v", err)
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Authentication successful!\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Get configuration to show API details
	config := client.Config()

	// Print authentication information
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔑 Authentication Information:\n")
		fmt.Fprintf(os.Stderr, "  BSN Base URL:     %s\n", config.BSNBaseURL)
		fmt.Fprintf(os.Stderr, "  API Version:      %s\n", config.APIVersion)
		fmt.Fprintf(os.Stderr, "  Token Endpoint:   %s\n", config.TokenEndpoint)
		fmt.Fprintf(os.Stderr, "  Client ID:        %s\n", config.ClientID)
		fmt.Fprintf(os.Stderr, "  Client Secret:    %s (hidden)\n", "***")
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Get the access token
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🔑 Getting access token...\n")
	}
	accessToken, err := client.GetAccessToken()
	if err != nil {
		log.Fatalf("❌ Failed to get access token: %v", err)
	}

	// Validate token format (should be a JWT or similar)
	if len(accessToken) < 20 {
		log.Fatalf("❌ Access token appears to be invalid (too short): %d characters", len(accessToken))
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Access token obtained (%d characters)\n", len(accessToken))
		if len(accessToken) > 50 {
			fmt.Fprintf(os.Stderr, "   Preview: %s...%s\n", accessToken[:20], accessToken[len(accessToken)-10:])
		}
	}

	// Save token to .token file
	tokenFile := "./.token"
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "💾 Saving access token to %s...\n", tokenFile)
	}

	tokenContent := fmt.Sprintf("export BSN_ACCESS_TOKEN=%s\n", accessToken)
	err = os.WriteFile(tokenFile, []byte(tokenContent), 0600)
	if err != nil {
		log.Fatalf("❌ Failed to write token file: %v", err)
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "✅ Token saved! You can now use:\n")
		fmt.Fprintf(os.Stderr, "   source %s\n", tokenFile)
		fmt.Fprintf(os.Stderr, "   echo $BSN_ACCESS_TOKEN\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Test the token by making an API call
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "🧪 Testing token validity...\n")
	}
	networks, err := client.GetNetworks(ctx)
	var tokenValid bool
	if err != nil {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "⚠️  Token test failed: %v\n", err)
			fmt.Fprintf(os.Stderr, "   This might indicate an authentication issue.\n")
			fmt.Fprintf(os.Stderr, "   The token has been saved, but may not work correctly.\n")
			fmt.Fprintf(os.Stderr, "   You can still try the curl commands - the issue might be temporary.\n")
			fmt.Fprintf(os.Stderr, "\n")
		}
		tokenValid = false
		// Create a dummy network for examples
		networks = []bscontrol.Network{{ID: 12345, Name: "ExampleNetwork"}}
	} else {
		if !*jsonFlag {
			fmt.Fprintf(os.Stderr, "✅ Token validation successful! Found %d networks.\n", len(networks))
			fmt.Fprintf(os.Stderr, "\n")
		}
		tokenValid = true
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"accessToken":   accessToken,
			"tokenFile":     tokenFile,
			"bsnBaseURL":    config.BSNBaseURL,
			"apiVersion":    config.APIVersion,
			"tokenEndpoint": config.TokenEndpoint,
			"clientID":      config.ClientID,
			"tokenValid":    tokenValid,
			"networks":      networks,
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			log.Fatalf("Failed to encode JSON: %v", err)
		}
		return
	}

	fmt.Fprintf(os.Stderr, "📋 curl Command Examples:\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "First, set your credentials:\n")
	fmt.Fprintf(os.Stderr, "export BS_CLIENT_ID=%s\n", config.ClientID)
	fmt.Fprintf(os.Stderr, "export BS_SECRET=YOUR_CLIENT_SECRET\n")
	fmt.Fprintf(os.Stderr, "\n")

	fmt.Fprintf(os.Stderr, "# Get Access Token (OAuth2 Client Credentials)\n")
	fmt.Fprintf(os.Stderr, "curl -X POST '%s' \\\n", config.TokenEndpoint)
	fmt.Fprintf(os.Stderr, "  -H 'Content-Type: application/x-www-form-urlencoded' \\\n")
	fmt.Fprintf(os.Stderr, "  -d 'grant_type=client_credentials' \\\n")
	fmt.Fprintf(os.Stderr, "  -u \"$BS_CLIENT_ID:$BS_SECRET\"\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "# Then extract the access_token from the JSON response and set it:\n")
	fmt.Fprintf(os.Stderr, "export BSN_ACCESS_TOKEN=<access_token_from_response>\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "# OR use the token file created by this tool:\n")
	fmt.Fprintf(os.Stderr, "source %s\n", tokenFile)
	fmt.Fprintf(os.Stderr, "\n")

	fmt.Fprintf(os.Stderr, "# Get Available Networks\n")
	fmt.Fprintf(os.Stderr, "curl -X GET '%s/%s/Self/Networks' \\\n", config.BSNBaseURL, config.APIVersion)
	fmt.Fprintf(os.Stderr, "  -H \"Authorization: Bearer $BSN_ACCESS_TOKEN\" \\\n")
	fmt.Fprintf(os.Stderr, "  -H 'Accept: application/json'\n")
	fmt.Fprintf(os.Stderr, "\n")

	if tokenValid {
		fmt.Fprintf(os.Stderr, "📊 Found %d networks:\n", len(networks))
		for i, network := range networks {
			fmt.Fprintf(os.Stderr, "  %d. %s (ID: %d)\n", i+1, network.Name, network.ID)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	fmt.Fprintf(os.Stderr, "✅ Authentication completed and token saved!\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "📝 Next steps:\n")
	fmt.Fprintf(os.Stderr, "  1. Source the token file: source %s\n", tokenFile)
	fmt.Fprintf(os.Stderr, "  2. Verify the token is set: echo $BSN_ACCESS_TOKEN\n")
	fmt.Fprintf(os.Stderr, "  3. Copy and run any of the curl commands above\n")
	fmt.Fprintf(os.Stderr, "  4. The token will expire - rerun this tool to get a fresh token\n")
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "🔐 Token saved to: %s (expires in ~1 hour)\n", tokenFile)

	if !tokenValid {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "🔧 Troubleshooting token issues:\n")
		fmt.Fprintf(os.Stderr, "  • Make sure BS_CLIENT_ID and BS_SECRET are correct\n")
		fmt.Fprintf(os.Stderr, "  • Check if your BSN.cloud account has proper permissions\n")
		fmt.Fprintf(os.Stderr, "  • Try running this tool again to get a fresh token\n")
		fmt.Fprintf(os.Stderr, "  • If curl commands fail with 401, the token may have expired\n")
		fmt.Fprintf(os.Stderr, "  • Token preview: %s...\n", accessToken[:min(50, len(accessToken))])
	}
}
