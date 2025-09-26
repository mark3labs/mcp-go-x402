package main

import (
	"context"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	x402 "github.com/yourusername/mcp-go-x402"
)

func main() {
	// Get private key from environment
	privateKey := os.Getenv("WALLET_PRIVATE_KEY")
	if privateKey == "" {
		log.Fatal("WALLET_PRIVATE_KEY environment variable is required")
	}

	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		serverURL = "https://paid-mcp-server.example.com"
	}

	// Create signer with your private key
	signer, err := x402.NewPrivateKeySigner(privateKey)
	if err != nil {
		log.Fatal("Failed to create signer:", err)
	}

	log.Printf("Using wallet address: %s", signer.GetAddress())

	// Create x402 transport
	transport, err := x402.New(x402.Config{
		ServerURL:        serverURL,
		Signer:           signer,
		MaxPaymentAmount: "1000000", // 1 USDC max per request
		AutoPayThreshold: "100000",  // Auto-pay up to 0.1 USDC
		OnPaymentAttempt: func(event x402.PaymentEvent) {
			log.Printf("Attempting payment of %s %s to %s",
				event.Amount, event.Asset, event.Recipient)
		},
		OnPaymentSuccess: func(event x402.PaymentEvent) {
			log.Printf("Payment successful! Transaction: %s", event.Transaction)
		},
		OnPaymentFailure: func(event x402.PaymentEvent, err error) {
			log.Printf("Payment failed: %v", err)
		},
	})
	if err != nil {
		log.Fatal("Failed to create transport:", err)
	}

	// Create MCP client with x402 transport
	mcpClient := client.NewClient(transport)

	ctx := context.Background()
	if err := mcpClient.Start(ctx); err != nil {
		log.Fatal("Failed to start client:", err)
	}
	defer mcpClient.Close()

	// Initialize MCP session
	initResp, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "1.0.0",
			ClientInfo: mcp.Implementation{
				Name:    "x402-example",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		log.Fatal("Failed to initialize:", err)
	}

	log.Printf("Connected to server: %s v%s",
		initResp.ServerInfo.Name, initResp.ServerInfo.Version)

	// List available tools
	toolsResp, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Printf("Failed to list tools: %v", err)
	} else {
		log.Printf("Found %d tools:", len(toolsResp.Tools))
		for _, tool := range toolsResp.Tools {
			log.Printf("  - %s: %s", tool.Name, tool.Description)
		}
	}

	// List available resources
	resourcesResp, err := mcpClient.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		log.Printf("Failed to list resources: %v", err)
	} else {
		log.Printf("Found %d resources:", len(resourcesResp.Resources))
		for _, resource := range resourcesResp.Resources {
			log.Printf("  - %s: %s", resource.URI, resource.Name)
		}
	}

	// Get payment metrics
	metrics := transport.GetMetrics()
	log.Printf("Payment metrics:")
	log.Printf("  Total spent: %s", metrics.TotalSpent)
	log.Printf("  Hourly spent: %s", metrics.HourlySpent)
	log.Printf("  Payment count: %d", metrics.PaymentCount)
}
