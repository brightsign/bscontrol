package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/brightsign/bscontrol"
)

func main() {
	var (
		helpFlag    = flag.Bool("help", false, "Display usage information")
		jsonFlag    = flag.Bool("json", false, "Output as JSON")
		verboseFlag = flag.Bool("verbose", false, "Show detailed information")
		timeoutFlag = flag.Int("timeout", 30, "Request timeout in seconds")
		serialFlag  = flag.String("serial", "", "Device serial number (required)")
		pathFlag    = flag.String("path", "/sd", "Destination path on player (default: /sd)")
		fileFlag    = flag.String("file", "", "Local file to upload (required)")
		nameFlag    = flag.String("name", "", "Destination filename (defaults to source filename)")
		networkFlag *string
	)

	networkFlag = flag.String("network", "", "Network name to use (overrides BS_NETWORK)")
	flag.StringVar(networkFlag, "n", "", "Network name to use (overrides BS_NETWORK) [alias for --network]")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Upload a file to a BrightSign player via rDWS.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
		fmt.Fprintf(os.Stderr, "  BS_CLIENT_ID        BSN.cloud API client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_SECRET          BSN.cloud API client secret (required)\n")
		fmt.Fprintf(os.Stderr, "  BS_NETWORK         BSN.cloud network name (optional)\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  Upload text file:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --file test.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Upload with custom name:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --file test.txt --name myfile.txt\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Upload to specific path:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --file test.txt --path /sd/subfolder\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  Output as JSON:\n")
		fmt.Fprintf(os.Stderr, "    %s --serial USD3A8000375 --file test.txt --json\n", os.Args[0])
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

	if *fileFlag == "" {
		fmt.Fprintf(os.Stderr, "Error: --file is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Read the file
	fileData, err := ioutil.ReadFile(*fileFlag)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Determine filename
	fileName := *nameFlag
	if fileName == "" {
		fileName = filepath.Base(*fileFlag)
	}

	// Determine file type and content format
	fileType := getFileType(*fileFlag)
	fileContents := ""

	if isTextFile(fileType) {
		// Text files can be sent as plain text
		fileContents = string(fileData)
	} else {
		// Binary files need to be base64 encoded as Data URL
		encoded := base64.StdEncoding.EncodeToString(fileData)
		fileContents = fmt.Sprintf("data:%s;base64,%s", fileType, encoded)
	}

	// Create client
	opts := []bscontrol.Option{bscontrol.WithEnv()}
	if *timeoutFlag > 0 {
		opts = append(opts, bscontrol.WithTimeout(time.Duration(*timeoutFlag)*time.Second))
	}
	if *networkFlag != "" {
		opts = append(opts, bscontrol.WithNetwork(*networkFlag))
	}

	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Creating BSN.cloud client...\n")
	}

	client, err := bscontrol.New(opts...)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Authenticate
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Authenticating...\n")
	}

	if err := client.Authenticate(ctx); err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	// Upload file
	fileSize := len(fileData)
	if !*jsonFlag {
		fmt.Fprintf(os.Stderr, "Uploading %s (%d bytes) to %s on device %s...\n", fileName, fileSize, *pathFlag, *serialFlag)
	}

	success, err := client.RDWS.UploadFile(ctx, *serialFlag, *pathFlag, fileName, fileContents, fileType)
	if err != nil {
		log.Fatalf("Failed to upload file: %v", err)
	}

	// Output as JSON if requested
	if *jsonFlag {
		result := map[string]interface{}{
			"success":         success,
			"serial":          *serialFlag,
			"destinationPath": *pathFlag,
			"filename":        fileName,
			"sizeBytes":       fileSize,
			"mimeType":        fileType,
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
		fmt.Fprintf(os.Stderr, "✅ File uploaded successfully: %s%s\n", *pathFlag, fileName)
		if *verboseFlag {
			fmt.Fprintf(os.Stderr, "   Size: %d bytes\n", fileSize)
			fmt.Fprintf(os.Stderr, "   Type: %s\n", fileType)
		}
	} else {
		fmt.Fprintf(os.Stderr, "❌ Upload failed\n")
		os.Exit(1)
	}
}

func getFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".txt":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	case ".js":
		return "application/javascript"
	case ".brs":
		return "text/plain"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".zip":
		return "application/zip"
	default:
		return "application/octet-stream"
	}
}

func isTextFile(mimeType string) bool {
	return strings.HasPrefix(mimeType, "text/") ||
		mimeType == "application/json" ||
		mimeType == "application/xml" ||
		mimeType == "application/javascript"
}
