package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	jsonFlag := flag.Bool("json", false, "Output as JSON")
	flag.Parse()

	tokenFile := "./.token"

	// Check if token file exists
	content, err := os.ReadFile(tokenFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ No token file found at %s\n", tokenFile)
		fmt.Fprintf(os.Stderr, "💡 Run './bin/auth-info' first to create the token file.\n")
		os.Exit(1)
	}

	// Parse the token
	tokenLine := strings.TrimSpace(string(content))
	if !strings.HasPrefix(tokenLine, "export BSN_ACCESS_TOKEN=") {
		fmt.Fprintf(os.Stderr, "❌ Invalid token file format. Expected: export BSN_ACCESS_TOKEN=<token>\n")
		fmt.Fprintf(os.Stderr, "Found: %s\n", tokenLine)
		os.Exit(1)
	}

	token := strings.TrimPrefix(tokenLine, "export BSN_ACCESS_TOKEN=")

	// Prepare validation results
	validation := map[string]interface{}{
		"lengthOK":    len(token) >= 20,
		"noSpaces":    !strings.Contains(token, " "),
		"isJWT":       strings.Count(token, ".") >= 2,
		"tokenLength": len(token),
	}

	var preview string
	if len(token) > 40 {
		preview = token[:20] + "..." + token[len(token)-10:]
	} else {
		preview = token
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"tokenFile":  tokenFile,
			"token":      token,
			"tokenType":  getTokenType(token),
			"length":     len(token),
			"preview":    preview,
			"validation": validation,
			"testCommands": []map[string]string{
				{
					"name":    "Test BSN API",
					"command": fmt.Sprintf("curl -X GET 'https://api.bsn.cloud/2024-02/Self/Networks' -H 'Authorization: Bearer %s' -H 'Accept: application/json'", token),
				},
				{
					"name":    "Test rDWS API",
					"command": fmt.Sprintf("curl -X PUT 'https://ws.bsn.cloud/rest/v1/control/reboot/?destinationType=player&destinationName=UTD41X000009' -H 'Authorization: Bearer %s' -H 'Accept: application/json'", token),
				},
			},
		}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to encode JSON: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Display token information
	fmt.Fprintf(os.Stderr, "🔍 Token Analysis:\n")
	fmt.Fprintf(os.Stderr, "  Length: %d characters\n", len(token))
	fmt.Fprintf(os.Stderr, "  Type: %s\n", getTokenType(token))

	if len(token) > 40 {
		fmt.Fprintf(os.Stderr, "  Preview: %s...%s\n", token[:20], token[len(token)-10:])
	} else {
		fmt.Fprintf(os.Stderr, "  Full token: %s\n", token)
	}

	// Basic validation
	fmt.Fprintf(os.Stderr, "\n🧪 Basic Validation:\n")
	if len(token) < 20 {
		fmt.Fprintf(os.Stderr, "  ⚠️  Token seems too short (< 20 chars)\n")
	} else {
		fmt.Fprintf(os.Stderr, "  ✅ Token length looks reasonable\n")
	}

	if strings.Contains(token, " ") {
		fmt.Fprintf(os.Stderr, "  ⚠️  Token contains spaces (might cause issues)\n")
	} else {
		fmt.Fprintf(os.Stderr, "  ✅ Token has no spaces\n")
	}

	// Check for common JWT format
	if strings.Count(token, ".") >= 2 {
		fmt.Fprintf(os.Stderr, "  ✅ Token appears to be JWT format\n")
	} else {
		fmt.Fprintf(os.Stderr, "  ⚠️  Token doesn't appear to be JWT format\n")
	}

	fmt.Fprintf(os.Stderr, "\n📋 Test Commands:\n")
	fmt.Fprintf(os.Stderr, "# Test BSN API (should work if token is valid)\n")
	fmt.Fprintf(os.Stderr, "curl -X GET 'https://api.bsn.cloud/2024-02/Self/Networks' \\\n")
	fmt.Fprintf(os.Stderr, "  -H 'Authorization: Bearer %s' \\\n", token)
	fmt.Fprintf(os.Stderr, "  -H 'Accept: application/json'\n")
	fmt.Fprintf(os.Stderr, "\n")

	fmt.Fprintf(os.Stderr, "# Test rDWS API (your failing command)\n")
	fmt.Fprintf(os.Stderr, "curl -X PUT 'https://ws.bsn.cloud/rest/v1/control/reboot/?destinationType=player&destinationName=UTD41X000009' \\\n")
	fmt.Fprintf(os.Stderr, "  -H 'Authorization: Bearer %s' \\\n", token)
	fmt.Fprintf(os.Stderr, "  -H 'Accept: application/json'\n")
	fmt.Fprintf(os.Stderr, "\n")

	fmt.Fprintf(os.Stderr, "💡 Next steps:\n")
	fmt.Fprintf(os.Stderr, "  1. Try the BSN API test command first\n")
	fmt.Fprintf(os.Stderr, "  2. If BSN API works but rDWS fails, there might be different auth requirements\n")
	fmt.Fprintf(os.Stderr, "  3. If both fail, run './bin/auth-info' to get a fresh token\n")
	fmt.Fprintf(os.Stderr, "  4. Check that UTD41X000009 is a valid device serial in your network\n")
}

func getTokenType(token string) string {
	if strings.Count(token, ".") >= 2 {
		return "JWT (JSON Web Token)"
	}
	if len(token) == 32 {
		return "MD5-like"
	}
	if len(token) == 40 {
		return "SHA1-like"
	}
	if len(token) == 64 {
		return "SHA256-like"
	}
	return "Unknown format"
}