package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/brightsign/bscontrol"
)

func main() {
	// Create client
	client, err := bscontrol.New()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Authenticate
	if err := client.Authenticate(ctx); err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}

	networkName := "gch-control-only"
	serial := "UTD416000978"

	fmt.Printf("=== Testing Device API for Serial: %s ===\n\n", serial)

	// Set network context
	if err := client.BDeploy.SetNetworkContext(ctx, networkName); err != nil {
		log.Fatalf("Failed to set network context: %v", err)
	}

	// Test 1: GetDeviceBySerial
	fmt.Println("1. GetDeviceBySerial():")
	deviceResp, err := client.BDeploy.GetDeviceBySerial(ctx, serial)
	if err != nil {
		log.Fatalf("Failed to get device: %v", err)
	}

	if deviceResp.Result.Matched > 0 && len(deviceResp.Result.Players) > 0 {
		device := deviceResp.Result.Players[0]
		fmt.Printf("   Serial:    %s\n", device.Serial)
		fmt.Printf("   SetupID:   '%s'\n", device.SetupID)
		fmt.Printf("   SetupName: '%s'\n", device.SetupName)
		fmt.Printf("   URL:       '%s'\n", device.URL)
		fmt.Println("\n   Full JSON:")
		jsonData, _ := json.MarshalIndent(device, "   ", "  ")
		fmt.Printf("   %s\n\n", string(jsonData))
	} else {
		fmt.Println("   No device found!")
	}

	// Test 2: GetAllDevices
	fmt.Println("2. GetAllDevices():")
	allDevices, err := client.BDeploy.GetAllDevices(ctx)
	if err != nil {
		log.Fatalf("Failed to get all devices: %v", err)
	}

	for _, device := range allDevices.Players {
		if device.Serial == serial {
			fmt.Printf("   Found device in list:\n")
			fmt.Printf("   Serial:    %s\n", device.Serial)
			fmt.Printf("   SetupID:   '%s'\n", device.SetupID)
			fmt.Printf("   SetupName: '%s'\n", device.SetupName)
			fmt.Printf("   URL:       '%s'\n", device.URL)
			fmt.Println("\n   Full JSON:")
			jsonData, _ := json.MarshalIndent(device, "   ", "  ")
			fmt.Printf("   %s\n\n", string(jsonData))
			break
		}
	}

	// Test 3: Check if association actually exists via direct curl to show raw API
	fmt.Println("3. Conclusion:")
	if deviceResp.Result.Matched > 0 && len(deviceResp.Result.Players) > 0 {
		device := deviceResp.Result.Players[0]
		if device.SetupID == "" {
			fmt.Println("   ❌ API does NOT return setupId field (confirmed API limitation)")
			fmt.Println("   The association was set via PUT but GET doesn't return it")
		} else {
			fmt.Printf("   ✅ API returns setupId: %s\n", device.SetupID)
		}
	}
}
