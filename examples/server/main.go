package main

import (
	"context"
	"fmt"
	"log"
	"os"

	x402server "github.com/mark3labs/mcp-go-x402/server"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	// Configure x402 server
	// You can set the facilitator URL via environment variable or config
	facilitatorURL := os.Getenv("X402_FACILITATOR_URL")
	if facilitatorURL == "" {
		facilitatorURL = "https://facilitator.x402.rs" // Production facilitator
	}

	// Get wallet configuration from environment
	payTo := os.Getenv("X402_PAY_TO")
	if payTo == "" {
		payTo = "0x209693Bc6afc0C5328bA36FaF03C514EF312287C" // Default test wallet
	}

	// Check if we should verify only (not settle)
	verifyOnly := os.Getenv("X402_VERIFY_ONLY") == "true"

	config := &x402server.Config{
		FacilitatorURL: facilitatorURL,
		VerifyOnly:     verifyOnly,
	}

	// Create x402 server
	srv := x402server.NewX402Server("x402-search-server", "1.0.0", config)

	// Add a paid search tool with multiple payment options
	srv.AddPayableTool(
		mcp.NewTool("search",
			mcp.WithDescription("Search for information on any topic"),
			mcp.WithString("query", mcp.Required(), mcp.Description("The search query")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results to return")),
		),
		searchHandler,
		// Option 1: USDC on Ethereum mainnet
		x402server.PaymentRequirement{
			Scheme:            "eip3009",
			Network:           "ethereum-mainnet",
			Asset:             "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", // USDC on Ethereum
			PayTo:             payTo,
			MaxAmountRequired: "10000", // 0.01 USDC (6 decimals)
			Description:       "Premium search via USDC on Ethereum",
			MaxTimeoutSeconds: 60,
			Extra: map[string]string{
				"name":    "USD Coin",
				"version": "2",
			},
		},
		// Option 2: USDC on Polygon (same price, lower fees)
		x402server.PaymentRequirement{
			Scheme:            "eip3009",
			Network:           "polygon-mainnet",
			Asset:             "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", // USDC on Polygon
			PayTo:             payTo,
			MaxAmountRequired: "10000", // 0.01 USDC
			Description:       "Premium search via USDC on Polygon (lower fees)",
			MaxTimeoutSeconds: 60,
			Extra: map[string]string{
				"name":    "USD Coin",
				"version": "2",
			},
		},
		// Option 3: USDC on Base (discounted price)
		x402server.PaymentRequirement{
			Scheme:            "eip3009",
			Network:           "base-mainnet",
			Asset:             "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
			PayTo:             payTo,
			MaxAmountRequired: "5000", // 0.005 USDC (50% discount)
			Description:       "Premium search via USDC on Base (50% discount)",
			MaxTimeoutSeconds: 60,
			Extra: map[string]string{
				"name":    "USD Coin",
				"version": "2",
			},
		},
	)

	// Add a free echo tool to demonstrate non-paid tools
	srv.AddTool(
		mcp.NewTool("echo",
			mcp.WithDescription("Simple echo tool that returns the input message"),
			mcp.WithString("message", mcp.Required(), mcp.Description("The message to echo back")),
		),
		echoHandler,
	)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting x402 MCP server on :%s", port)
	log.Printf("Server URL: http://localhost:%s", port)
	log.Printf("Facilitator URL: %s", facilitatorURL)
	log.Printf("Payment recipient: %s", payTo)
	log.Printf("Verify Only Mode: %v", verifyOnly)
	log.Println("Tools:")
	log.Println("  - search (paid - multiple payment options):")
	log.Println("    • 0.01 USDC on Ethereum mainnet")
	log.Println("    • 0.01 USDC on Polygon (lower fees)")
	log.Println("    • 0.005 USDC on Base (50% discount)")
	log.Println("  - echo (free)")
	log.Println("")
	log.Println("Connect with client using:")
	log.Printf("  export MCP_SERVER_URL=http://localhost:%s", port)
	if verifyOnly {
		log.Println("  (Running in verify-only mode - payments will be verified but not settled)")
	}

	if err := srv.Start(":" + port); err != nil {
		log.Fatal(err)
	}
}

func searchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := request.GetString("query", "")
	maxResults := request.GetFloat("max_results", 5)

	if query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	// Simulate a search operation
	// In a real implementation, this would query an actual search API
	results := fmt.Sprintf(`Search Results for "%s":

1. Understanding %s: A comprehensive guide
   Learn everything you need to know about %s, from basics to advanced concepts.

2. %s in Practice: Real-world applications
   Discover how %s is being used in various industries and applications.

3. The Future of %s: Trends and predictions
   Expert analysis on where %s is heading in the next 5 years.

4. %s FAQ: Common questions answered
   Get answers to the most frequently asked questions about %s.

5. %s Resources: Tools and references
   A curated list of the best resources for learning and working with %s.

Showing top %.0f results`,
		query, query, query, query, query, query, query, query, query, query, query, maxResults)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(results),
		},
	}, nil
}

func echoHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	message := request.GetString("message", "")

	if message == "" {
		return nil, fmt.Errorf("message parameter is required")
	}

	// Simply echo back the message
	response := fmt.Sprintf("Echo: %s", message)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(response),
		},
	}, nil
}
