package main

import (
	"context"
	"flag"
	"log"
	"os"

	x402 "github.com/mark3labs/mcp-go-x402"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	serverURL := flag.String("server", "http://localhost:8080", "MCP server URL")
	privateKey := flag.String("key", "", "Private key (hex)")
	preferredChain := flag.String("chain", "base", "Preferred chain (base, polygon, avalanche)")
	flag.Parse()

	if *privateKey == "" {
		if envKey := os.Getenv("PRIVATE_KEY"); envKey != "" {
			*privateKey = envKey
		} else {
			log.Fatal("Private key required (use -key flag or PRIVATE_KEY env var)")
		}
	}

	// Configure payment options based on preferred chain
	var paymentOptions []x402.ClientPaymentOption

	switch *preferredChain {
	case "polygon":
		log.Println("Configuring for Polygon as primary chain...")
		paymentOptions = []x402.ClientPaymentOption{
			x402.AcceptUSDCPolygon().WithPriority(1),     // Primary: Polygon mainnet
			x402.AcceptUSDCPolygonAmoy().WithPriority(2), // Fallback: Polygon testnet
			x402.AcceptUSDCBase().WithPriority(3),        // Alternative: Base
			x402.AcceptUSDCAvalanche().WithPriority(4),   // Alternative: Avalanche
		}
	case "avalanche":
		log.Println("Configuring for Avalanche as primary chain...")
		paymentOptions = []x402.ClientPaymentOption{
			x402.AcceptUSDCAvalanche().WithPriority(1),     // Primary: Avalanche mainnet
			x402.AcceptUSDCAvalancheFuji().WithPriority(2), // Fallback: Avalanche testnet
			x402.AcceptUSDCBase().WithPriority(3),          // Alternative: Base
			x402.AcceptUSDCPolygon().WithPriority(4),       // Alternative: Polygon
		}
	default: // "base"
		log.Println("Configuring for Base as primary chain...")
		paymentOptions = []x402.ClientPaymentOption{
			x402.AcceptUSDCBase().WithPriority(1),        // Primary: Base mainnet
			x402.AcceptUSDCBaseSepolia().WithPriority(2), // Fallback: Base testnet
			x402.AcceptUSDCPolygon().WithPriority(3),     // Alternative: Polygon
			x402.AcceptUSDCAvalanche().WithPriority(4),   // Alternative: Avalanche
		}
	}

	// Create signer with multi-chain support
	signer, err := x402.NewPrivateKeySigner(*privateKey, paymentOptions...)
	if err != nil {
		log.Fatalf("Failed to create signer: %v", err)
	}

	log.Printf("Payment signer configured with %d payment options", len(paymentOptions))
	log.Println("Supported networks:")
	for _, opt := range paymentOptions {
		log.Printf("  - %s (priority %d)", opt.Network, opt.Priority)
	}

	// Create x402 transport
	transport, err := x402.New(x402.Config{
		ServerURL: *serverURL,
		Signers:   []x402.PaymentSigner{signer},
		OnPaymentAttempt: func(event x402.PaymentEvent) {
			log.Printf("📤 Attempting payment on %s: %s to %s", event.Network, event.Amount, event.Recipient)
		},
		OnPaymentSuccess: func(event x402.PaymentEvent) {
			log.Printf("✅ Payment successful on %s: tx %s", event.Network, event.Transaction)
		},
		OnPaymentFailure: func(event x402.PaymentEvent, err error) {
			log.Printf("❌ Payment failed on %s: %v", event.Network, err)
		},
	})
	if err != nil {
		log.Fatalf("Failed to create transport: %v", err)
	}

	// Create MCP client
	mcpClient := client.NewClient(transport)

	// Start the client
	ctx := context.Background()
	if err := mcpClient.Start(ctx); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}
	defer mcpClient.Close()

	// Initialize
	_, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "1.0.0",
			ClientInfo: mcp.Implementation{
				Name:    "multi-chain-client",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// List available tools
	tools, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Fatalf("Failed to list tools: %v", err)
	}

	log.Printf("\nFound %d tools:", len(tools.Tools))
	for _, tool := range tools.Tools {
		log.Printf("  - %s: %s", tool.Name, tool.Description)
	}

	// Try calling a tool (if any are available)
	if len(tools.Tools) > 0 {
		toolName := tools.Tools[0].Name
		log.Printf("\nCalling tool: %s", toolName)

		// Build simple test arguments
		args := map[string]interface{}{
			"query": "test query", // Common parameter for many tools
		}

		result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      toolName,
				Arguments: args,
			},
		})
		if err != nil {
			log.Printf("Tool call failed: %v", err)
		} else {
			log.Printf("Tool result: %+v", result)
		}
	}

	log.Println("\n✨ Multi-chain client example completed successfully!")
}
