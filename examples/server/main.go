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

	asset := os.Getenv("X402_ASSET")
	if asset == "" {
		asset = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913" // USDC on Base mainnet
	}

	network := os.Getenv("X402_NETWORK")
	if network == "" {
		network = "base" // Base mainnet
	}

	config := &x402server.Config{
		FacilitatorURL: facilitatorURL,
		DefaultPayTo:   payTo,
		DefaultAsset:   asset,
		DefaultNetwork: network,
	}

	// Create x402 server
	srv := x402server.NewX402Server("x402-search-server", "1.0.0", config)

	// Add a paid search tool (modeled after basic example)
	srv.AddPayableTool(
		mcp.NewTool("search",
			mcp.WithDescription("Search for information on any topic"),
			mcp.WithString("query", mcp.Required(), mcp.Description("The search query")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results to return")),
		),
		searchHandler,
		"10000", // 0.01 USDC (6 decimals)
		"Premium search service - provides high-quality search results",
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
	log.Printf("Asset: %s", asset)
	log.Printf("Network: %s", network)
	log.Println("Tool: search (0.01 USDC per query)")
	log.Println("")
	log.Println("Connect with client using:")
	log.Printf("  export MCP_SERVER_URL=http://localhost:%s", port)

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
