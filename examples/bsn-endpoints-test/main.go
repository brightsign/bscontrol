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
		os.Exit(1)
	}

	token := strings.TrimPrefix(tokenLine, "export BSN_ACCESS_TOKEN=")

	// Test different API version paths
	type Endpoint struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}

	endpoints := []Endpoint{
		{"Networks (2022/06/REST)", "https://api.bsn.cloud/2022/06/REST/Self/Networks"},
		{"Networks (2024-02)", "https://api.bsn.cloud/2024-02/Self/Networks"},
		{"User Info (2022/06/REST)", "https://api.bsn.cloud/2022/06/REST/Self/Info"},
		{"User Profile (2022/06/REST)", "https://api.bsn.cloud/2022/06/REST/Self/Profile"},
		{"Token Info (2022/06/REST)", "https://api.bsn.cloud/2022/06/REST/Self/Token"},
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"tokenFile": tokenFile,
			"token":     token,
			"endpoints": endpoints,
			"expectedResults": map[string]string{
				"200": "Success",
				"401": "Invalid token",
				"403": "Valid token, insufficient permissions",
				"404": "Endpoint doesn't exist",
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

	fmt.Fprintf(os.Stderr, "🧪 Testing different BSN API endpoints with your token:\n\n")

	for i, endpoint := range endpoints {
		fmt.Fprintf(os.Stderr, "# %d. Test %s\n", i+1, endpoint.Name)
		fmt.Fprintf(os.Stderr, "curl -X GET '%s' \\\n", endpoint.URL)
		fmt.Fprintf(os.Stderr, "  -H 'Authorization: Bearer %s' \\\n", token)
		fmt.Fprintf(os.Stderr, "  -H 'Accept: application/json'\n")
		fmt.Fprintf(os.Stderr, "echo \"Status: $?, Response above for %s\"\n", endpoint.Name)
		fmt.Fprintf(os.Stderr, "\n")
	}

	fmt.Fprintf(os.Stderr, "💡 Try these commands one by one to see which endpoints work.\n")
	fmt.Fprintf(os.Stderr, "Expected results:\n")
	fmt.Fprintf(os.Stderr, "  • 200 OK = Success\n")
	fmt.Fprintf(os.Stderr, "  • 401 Unauthorized = Invalid token\n")
	fmt.Fprintf(os.Stderr, "  • 403 Forbidden = Valid token, insufficient permissions\n")
	fmt.Fprintf(os.Stderr, "  • 404 Not Found = Endpoint doesn't exist\n")
}