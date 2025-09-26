package main

import (
	"context"
	"log"
	"os"

	x402 "github.com/mark3labs/mcp-go-x402"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	// Get private key from environment
	privateKey := os.Getenv("WALLET_PRIVATE_KEY")
	if privateKey == "" {
		log.Fatal("WALLET_PRIVATE_KEY environment variable is required")
	}

	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	// Create signer with your private key
	signer, err := x402.NewPrivateKeySigner(privateKey)
	if err != nil {
		log.Fatal("Failed to create signer:", err)
	}

	log.Printf("Using wallet address: %s", signer.GetAddress())

	// Create x402 transport
	x402transport, err := x402.New(x402.Config{
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
	mcpClient := client.NewClient(x402transport)

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

	// List all available tools
	log.Println("\nListing available tools...")
	toolsResp, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Printf("Failed to list tools: %v", err)
	} else {
		log.Println("Available tools:")
		for _, tool := range toolsResp.Tools {
			log.Printf("  - %s: %s", tool.Name, tool.Description)
		}
	}

	// Call the free echo tool first
	log.Println("\nCalling echo tool (free)...")
	echoResp, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "echo",
			Arguments: map[string]any{
				"message": "Hello from x402 client!",
			},
		},
	})

	if err != nil {
		log.Printf("Echo failed: %v", err)
	} else {
		log.Println("Echo response:")
		if len(echoResp.Content) > 0 {
			if textContent, ok := mcp.AsTextContent(echoResp.Content[0]); ok {
				log.Println(textContent.Text)
			}
		}
	}

	// Call the search tool (paid)
	log.Println("\nSearching for 'x402' (paid tool)...")
	searchResp, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "search",
			Arguments: map[string]any{
				"query":       "x402",
				"max_results": 5,
			},
		},
	})

	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	log.Println("Search results:")
	if len(searchResp.Content) > 0 {
		if textContent, ok := mcp.AsTextContent(searchResp.Content[0]); ok {
			log.Println(textContent.Text)
		}
	}
}
