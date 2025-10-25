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
	var (
		privateKeyFlag = flag.String("key", "", "Solana private key base58 (or set SOLANA_PRIVATE_KEY env var)")
		keypairFlag    = flag.String("keypair", "", "Path to Solana CLI keypair file (alternative to -key)")
		serverURL      = flag.String("server", "http://localhost:8080", "MCP server URL")
		network        = flag.String("network", "devnet", "Network to use: devnet or mainnet")
		verbose        = flag.Bool("v", false, "Verbose output")
	)
	flag.Parse()

	var signer x402.PaymentSigner
	var err error

	if *keypairFlag != "" {
		log.Printf("Loading keypair from: %s", *keypairFlag)
		if *network == "mainnet" {
			signer, err = x402.NewSolanaPrivateKeySignerFromFile(
				*keypairFlag,
				x402.AcceptUSDCSolana().
					WithPriority(1).
					WithMaxAmount("500000").
					WithMinBalance("1000000"),
			)
		} else {
			signer, err = x402.NewSolanaPrivateKeySignerFromFile(
				*keypairFlag,
				x402.AcceptUSDCSolanaDevnet(),
			)
		}
	} else {
		privateKey := *privateKeyFlag
		if privateKey == "" {
			privateKey = os.Getenv("SOLANA_PRIVATE_KEY")
			if privateKey == "" {
				log.Fatal("Private key required: use -key flag, -keypair flag, or set SOLANA_PRIVATE_KEY environment variable")
			}
		}

		if *network == "mainnet" {
			log.Println("Configuring for Solana mainnet...")
			signer, err = x402.NewSolanaPrivateKeySigner(
				privateKey,
				x402.AcceptUSDCSolana().
					WithPriority(1).
					WithMaxAmount("500000").
					WithMinBalance("1000000"),
			)
		} else {
			log.Println("Configuring for Solana devnet...")
			signer, err = x402.NewSolanaPrivateKeySigner(
				privateKey,
				x402.AcceptUSDCSolanaDevnet(),
			)
		}
	}

	if err != nil {
		log.Fatal("Failed to create signer:", err)
	}

	log.Printf("Using Solana address: %s", signer.GetAddress())
	if *verbose {
		log.Printf("Payment options configured:")
		if signer.SupportsNetwork("solana") {
			log.Printf("  - Solana mainnet: USDC")
		}
		if signer.SupportsNetwork("solana-devnet") {
			log.Printf("  - Solana devnet: USDC (testnet)")
		}
	}

	config := x402.Config{
		ServerURL: *serverURL,
		Signers:   []x402.PaymentSigner{signer},
	}

	if *verbose {
		config.OnPaymentAttempt = func(event x402.PaymentEvent) {
			log.Printf("Attempting payment of %s %s to %s",
				event.Amount, event.Asset, event.Recipient)
		}
		config.OnPaymentSuccess = func(event x402.PaymentEvent) {
			log.Printf("Payment successful! Transaction: %s", event.Transaction)
		}
		config.OnPaymentFailure = func(event x402.PaymentEvent, err error) {
			log.Printf("Payment failed: %v", err)
		}
	}

	x402transport, err := x402.New(config)
	if err != nil {
		log.Fatal("Failed to create transport:", err)
	}

	mcpClient := client.NewClient(x402transport)

	ctx := context.Background()
	if err := mcpClient.Start(ctx); err != nil {
		log.Fatal("Failed to start client:", err)
	}
	defer mcpClient.Close()

	initResp, err := mcpClient.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "1.0.0",
			ClientInfo: mcp.Implementation{
				Name:    "x402-svm-example",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		log.Fatal("Failed to initialize:", err)
	}

	log.Printf("Connected to server: %s v%s",
		initResp.ServerInfo.Name, initResp.ServerInfo.Version)

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

	log.Println("\nCalling echo tool (free)...")
	echoResp, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "echo",
			Arguments: map[string]any{
				"message": "Hello from x402 SVM client!",
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

	log.Println("\nSearching for 'Solana' (paid tool)...")
	searchResp, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "search",
			Arguments: map[string]any{
				"query":       "Solana",
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
