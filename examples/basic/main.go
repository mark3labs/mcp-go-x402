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
		serverURL = "https://mcpay.tech/mcp/a9ad1af3-f91a-468c-96e4-28ebdfdd36c3"
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

	// Call the search tool
	log.Println("Searching for 'x402'...")
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
