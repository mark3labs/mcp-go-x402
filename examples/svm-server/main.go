package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	x402server "github.com/mark3labs/mcp-go-x402/server"
	"github.com/mark3labs/mcp-go/mcp"
)

func main() {
	var (
		port           = flag.String("port", "8080", "Port to listen on")
		facilitatorURL = flag.String("facilitator", "https://facilitator.payai.network", "x402 facilitator URL")
		payTo          = flag.String("pay-to", "", "Payment recipient Solana address (required)")
		verifyOnly     = flag.Bool("verify-only", false, "Only verify payments, don't settle on-chain")
		devnet         = flag.Bool("devnet", false, "Use Solana devnet instead of mainnet")
		verbose        = flag.Bool("v", false, "Verbose output (show requests and payment processing)")
	)
	flag.Parse()

	if *payTo == "" {
		log.Fatal("Error: -pay-to flag is required. Please provide a Solana address to receive payments.")
	}

	config := &x402server.Config{
		FacilitatorURL: *facilitatorURL,
		VerifyOnly:     *verifyOnly,
		Verbose:        *verbose,
	}

	srv := x402server.NewX402Server("x402-svm-search-server", "1.0.0", config)

	var searchPaymentReq x402server.PaymentRequirement
	if *devnet {
		searchPaymentReq = x402server.RequireUSDCSolanaDevnet(
			*payTo,
			"10000",
			"Premium search service - 0.01 USDC on Solana devnet",
		)
	} else {
		searchPaymentReq = x402server.RequireUSDCSolana(
			*payTo,
			"10000",
			"Premium search service - 0.01 USDC on Solana",
		)
	}

	srv.AddPayableTool(
		mcp.NewTool("search",
			mcp.WithDescription("Search for information on any topic"),
			mcp.WithString("query", mcp.Required(), mcp.Description("The search query")),
			mcp.WithNumber("max_results", mcp.Description("Maximum number of results to return")),
		),
		searchHandler,
		searchPaymentReq,
	)

	srv.AddTool(
		mcp.NewTool("echo",
			mcp.WithDescription("Simple echo tool that returns the input message"),
			mcp.WithString("message", mcp.Required(), mcp.Description("The message to echo back")),
		),
		echoHandler,
	)

	if *devnet {
		srv.AddPayableTool(
			mcp.NewTool("test-feature",
				mcp.WithDescription("Test feature for development"),
				mcp.WithString("input", mcp.Required(), mcp.Description("Test input")),
			),
			testFeatureHandler,
			x402server.RequireUSDCSolanaDevnet(
				*payTo,
				"1000",
				"Test payment on Solana devnet - 0.001 USDC",
			),
		)
	}

	networkName := "mainnet"
	if *devnet {
		networkName = "devnet"
	}

	log.Printf("Starting x402 SVM MCP server on :%s", *port)
	log.Printf("Server URL: http://localhost:%s", *port)
	log.Printf("Facilitator URL: %s", *facilitatorURL)
	log.Printf("Solana Network: %s", networkName)
	log.Printf("Payment recipient: %s", *payTo)
	log.Printf("Verify Only Mode: %v", *verifyOnly)
	if *verbose {
		log.Printf("Verbose Mode: ENABLED")
	}
	log.Println("Tools:")
	if *devnet {
		log.Println("  - search (paid): 0.01 USDC on Solana devnet")
		log.Println("  - test-feature (devnet): 0.001 USDC on Solana devnet")
	} else {
		log.Println("  - search (paid): 0.01 USDC on Solana mainnet")
	}
	log.Println("  - echo (free)")
	log.Println("")
	log.Println("Connect with SVM client using:")
	if *devnet {
		log.Printf("  ./svm-client -server http://localhost:%s -network devnet", *port)
	} else {
		log.Printf("  ./svm-client -server http://localhost:%s -network mainnet", *port)
	}
	if *verifyOnly {
		log.Println("  (Running in verify-only mode - payments will be verified but not settled)")
	}

	if err := srv.Start(":" + *port); err != nil {
		log.Fatal(err)
	}
}

func searchHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := request.GetString("query", "")
	maxResults := request.GetFloat("max_results", 5)

	if query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

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

	response := fmt.Sprintf("Echo: %s", message)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(response),
		},
	}, nil
}

func testFeatureHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	input := request.GetString("input", "")

	if input == "" {
		return nil, fmt.Errorf("input parameter is required")
	}

	response := fmt.Sprintf("Test feature processed: %s", input)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(response),
		},
	}, nil
}
