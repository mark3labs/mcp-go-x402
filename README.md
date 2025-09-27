# mcp-go-x402

x402 payment protocol support for [MCP-Go](https://github.com/mark3labs/mcp-go) clients and servers.

This library provides:
- **Client Transport**: Automatic x402 payment handling for MCP clients
- **Server Wrapper**: Payment collection support for MCP servers

## Features

### Client Features
- 🔌 **Drop-in replacement**: Fully compatible with mcp-go transport interface
- 💰 **Automatic payments**: Handles 402 responses transparently
- 🔐 **Multiple signers**: Support for private keys, mnemonics, keystores
- 📊 **Budget management**: Configurable spending limits and rate limiting
- 🎯 **Smart payment**: Automatic for small amounts, callbacks for large
- 🧪 **Testing support**: Mock signers and payment recorders for easy testing

### Server Features
- 💳 **Payment collection**: Require payments for specific MCP tools
- 🔒 **Payment verification**: Automatic verification via x402 facilitator
- ⛓️ **On-chain settlement**: Automatic settlement of verified payments
- 🎛️ **Flexible pricing**: Set different prices for different tools
- 🔄 **Mixed mode**: Support both free and paid tools on same server

## Installation

```bash
go get github.com/mark3labs/mcp-go-x402
```

## Quick Start

### Client Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/mcp"
    x402 "github.com/mark3labs/mcp-go-x402"
)

func main() {
    // Create signer with your private key and explicit payment options
    signer, err := x402.NewPrivateKeySigner(
        "YOUR_PRIVATE_KEY_HEX",
        x402.AcceptUSDCBase(), // Accept USDC on Base mainnet
    )
    if err != nil {
        log.Fatal(err)
    }
    
    // Create x402 transport
    transport, err := x402.New(x402.Config{
        ServerURL:        "https://paid-mcp-server.example.com",
        Signer:           signer,
        MaxPaymentAmount: "100000",  // 0.1 USDC max per request
        AutoPayThreshold: "10000",   // Auto-pay up to 0.01 USDC
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Create MCP client with x402 transport
    mcpClient := client.NewClient(transport)
    
    ctx := context.Background()
    if err := mcpClient.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer mcpClient.Close()
    
    // Initialize MCP session
    _, err = mcpClient.Initialize(ctx, mcp.InitializeRequest{
        Params: mcp.InitializeParams{
            ProtocolVersion: "1.0.0",
            ClientInfo: mcp.Implementation{
                Name:    "x402-client",
                Version: "1.0.0",
            },
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Use MCP client normally - payments handled automatically!
    tools, _ := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
    log.Printf("Found %d tools", len(tools.Tools))
}
```

### Server Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/mark3labs/mcp-go/mcp"
    "github.com/mark3labs/mcp-go/server"
    x402server "github.com/mark3labs/mcp-go-x402/server"
)

func main() {
    // Configure x402 server
    config := &x402server.Config{
        FacilitatorURL:  "https://facilitator.x402.rs",
        VerifyOnly:      false, // Set to true for testing without settlement
    }
    
    // Create x402 server
    srv := x402server.NewX402Server("my-server", "1.0.0", config)
    
    // Add a free tool
    srv.AddTool(
        mcp.NewTool("free-tool", 
            mcp.WithDescription("This tool is free")),
        freeToolHandler,
    )
    
    // Add a paid tool with multiple payment options
    srv.AddPayableTool(
        mcp.NewTool("premium-tool",
            mcp.WithDescription("Premium feature"),
            mcp.WithString("input", mcp.Required())),
        premiumToolHandler,
        // Option 1: Pay with USDC on Ethereum
        x402server.PaymentRequirement{
            Scheme:            "exact",
            Network:           "ethereum-mainnet",
            Asset:             "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
            PayTo:             "0xYourWallet",
            MaxAmountRequired: "10000", // 0.01 USDC
            Description:       "Premium feature via Ethereum",
            MaxTimeoutSeconds: 60,
        },
        // Option 2: Pay with USDC on Base (discounted)
        x402server.PaymentRequirement{
            Scheme:            "exact",
            Network:           "base-mainnet",
            Asset:             "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
            PayTo:             "0xYourWallet",
            MaxAmountRequired: "5000", // 0.005 USDC (50% discount)
            Description:       "Premium feature via Base (50% off)",
            MaxTimeoutSeconds: 60,
        },
    )
    
    // Start server
    log.Println("Starting x402 MCP server on :8080")
    if err := srv.Start(":8080"); err != nil {
        log.Fatal(err)
    }
}

func freeToolHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    return &mcp.CallToolResult{
        Content: []mcp.Content{mcp.NewTextContent("Free response")},
    }, nil
}

func premiumToolHandler(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    input := req.GetString("input", "")
    // Process premium request
    return &mcp.CallToolResult{
        Content: []mcp.Content{mcp.NewTextContent("Premium response for: " + input)},
    }, nil
}
```

## Client Configuration Options

### Basic Configuration

```go
// Create signer with explicit payment options
signer, err := x402.NewPrivateKeySigner(
    privateKey,
    x402.AcceptUSDCBase(),       // Accept USDC on Base
    x402.AcceptUSDCBaseSepolia(), // Accept USDC on Base Sepolia (testnet)
)

config := x402.Config{
    ServerURL:        "https://server.example.com",
    Signer:           signer,
    MaxPaymentAmount: "1000000",  // Maximum 1 USDC per request
    AutoPayThreshold: "100000",   // Auto-pay up to 0.1 USDC
}
```

### With Rate Limiting

```go
signer, err := x402.NewPrivateKeySigner(
    privateKey,
    x402.AcceptUSDCBase(),
)

config := x402.Config{
    ServerURL:        "https://server.example.com",
    Signer:           signer,
    MaxPaymentAmount: "1000000",
    RateLimits: &x402.RateLimits{
        MaxPaymentsPerMinute: 60,
        MaxAmountPerHour:     "10000000", // 10 USDC per hour
    },
}
```

### With Payment Approval Callback

```go
config := x402.Config{
    ServerURL:        "https://server.example.com",
    Signer:           signer,
    MaxPaymentAmount: "1000000",
    AutoPayThreshold: "10000",
    PaymentCallback: func(amount *big.Int, resource string) bool {
        // Custom logic to approve/decline payments
        fmt.Printf("Approve payment of %s for %s? ", amount, resource)
        return getUserApproval()
    },
}
```

### With Event Callbacks

```go
config := x402.Config{
    ServerURL:        "https://server.example.com",
    Signer:           signer,
    MaxPaymentAmount: "1000000",
    OnPaymentAttempt: func(event x402.PaymentEvent) {
        log.Printf("Attempting payment: %s to %s", event.Amount, event.Recipient)
    },
    OnPaymentSuccess: func(event x402.PaymentEvent) {
        log.Printf("Payment successful: tx %s", event.Transaction)
    },
    OnPaymentFailure: func(event x402.PaymentEvent, err error) {
        log.Printf("Payment failed: %v", err)
    },
}
```

## Server Configuration Options

### Basic Server Configuration

```go
config := &x402server.Config{
    FacilitatorURL:  "https://facilitator.x402.rs",
    VerifyOnly:      false, // Set to true for testing without settlement
}
```

### Multiple Payment Options

Servers can now offer multiple payment options per tool, allowing clients to choose their preferred network or take advantage of discounts:

```go
srv.AddPayableTool(
    mcp.NewTool("analytics",
        mcp.WithDescription("Advanced analytics"),
        mcp.WithString("query", mcp.Required())),
    analyticsHandler,
    // Ethereum mainnet - standard price
    x402server.PaymentRequirement{
        Scheme:            "eip3009",
        Network:           "ethereum-mainnet",
        Asset:             "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", // USDC
        PayTo:             "0xYourWallet",
        MaxAmountRequired: "100000", // 0.1 USDC
        Description:       "Analytics via Ethereum",
    },
    // Polygon - same price, lower gas fees
    x402server.PaymentRequirement{
        Scheme:            "eip3009",
        Network:           "polygon-mainnet",
        Asset:             "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", // USDC on Polygon
        PayTo:             "0xYourWallet",
        MaxAmountRequired: "100000", // 0.1 USDC
        Description:       "Analytics via Polygon (lower fees)",
    },
    // Base - discounted price
    x402server.PaymentRequirement{
        Scheme:            "eip3009",
        Network:           "base-mainnet",
        Asset:             "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
        PayTo:             "0xYourWallet",
        MaxAmountRequired: "50000", // 0.05 USDC (50% discount)
        Description:       "Analytics via Base (50% discount)",
    },
)
```

When a client requests a paid tool without payment, they receive all available payment options and can choose the one that works best for them based on:
- Network preference (gas fees, speed)
- Available balance on different chains
- Price differences (discounts for certain networks)
```

### Using with Existing MCP Server

```go
// If you already have an MCP server, wrap it with x402
mcpServer := server.NewMCPServer("existing", "1.0")
httpServer := server.NewStreamableHTTPServer(mcpServer)

// Wrap with x402 handler
x402Handler := x402server.NewX402Handler(httpServer, config)

// Use as http.Handler
http.Handle("/", x402Handler)
http.ListenAndServe(":8080", nil)
```

## Signer Options (Client)

### Private Key

```go
signer, err := x402.NewPrivateKeySigner(
    "0xYourPrivateKeyHex",
    x402.AcceptUSDCBase(),       // Must specify at least one payment option
)
```

### Mnemonic (BIP-39)

```go
signer, err := x402.NewMnemonicSigner(
    "your twelve word mnemonic phrase here ...",
    "m/44'/60'/0'/0/0", // Optional: derivation path
    x402.AcceptUSDCBase(),
)
```

### Keystore File

```go
keystoreJSON, _ := os.ReadFile("keystore.json")
signer, err := x402.NewKeystoreSigner(
    keystoreJSON,
    "password",
    x402.AcceptUSDCBase(),
)
```

### Multiple Payment Options with Priorities

```go
signer, err := x402.NewPrivateKeySigner(
    privateKey,
    // Priority 1: Prefer Base (cheap & fast)
    x402.AcceptUSDCBase().WithPriority(1),
    
    // Priority 2: Fallback to Base Sepolia (testnet)
    x402.AcceptUSDCBaseSepolia().WithPriority(2),
)
```

### With Custom Limits

```go
signer, err := x402.NewPrivateKeySigner(
    privateKey,
    x402.AcceptUSDCBase()
        .WithMaxAmount("100000")    // Max 0.1 USDC per payment
        .WithMinBalance("1000000"), // Keep 1 USDC reserve
)
```

### Custom Signer

```go
type MyCustomSigner struct {
    paymentOptions []x402.ClientPaymentOption
}

func NewMyCustomSigner() *MyCustomSigner {
    return &MyCustomSigner{
        paymentOptions: []x402.ClientPaymentOption{
            x402.AcceptUSDCBase(),
            x402.AcceptUSDCBaseSepolia(),
        },
    }
}

func (s *MyCustomSigner) SignPayment(ctx context.Context, req x402.PaymentRequirement) (*x402.PaymentPayload, error) {
    // Your custom signing logic
    // Could use hardware wallet, remote signer, etc.
}

func (s *MyCustomSigner) GetAddress() string {
    return "0xYourAddress"
}

func (s *MyCustomSigner) SupportsNetwork(network string) bool {
    for _, opt := range s.paymentOptions {
        if opt.Network == network {
            return true
        }
    }
    return false
}

func (s *MyCustomSigner) HasAsset(asset, network string) bool {
    for _, opt := range s.paymentOptions {
        if opt.Network == network && opt.Asset == asset {
            return true
        }
    }
    return false
}

func (s *MyCustomSigner) GetPaymentOption(network, asset string) *x402.ClientPaymentOption {
    for _, opt := range s.paymentOptions {
        if opt.Network == network && opt.Asset == asset {
            optCopy := opt
            return &optCopy
        }
    }
    return nil
}
```

## Testing

### Using Mock Signer

```go
func TestMyMCPClient(t *testing.T) {
    // No real wallet needed for tests
    signer := x402.NewMockSigner(
        "0xTestWallet",
        x402.AcceptUSDCBaseSepolia(), // Mock signer for testing
    )
    
    transport, _ := x402.New(x402.Config{
        ServerURL:        "https://test-server.example.com",
        Signer:           signer,
        MaxPaymentAmount: "1000000",
    })
    
    // Test your MCP client
    client := client.NewClient(transport)
    // ...
}
```

### Recording Payments

```go
func TestPaymentFlow(t *testing.T) {
    // Create mock signer and recorder
    signer := x402.NewMockSigner(
        "0xTestWallet",
        x402.AcceptUSDCBaseSepolia(),
    )
    recorder := x402.NewPaymentRecorder()
    
    transport, _ := x402.New(x402.Config{
        ServerURL:        testServer.URL,
        Signer:           signer,
        MaxPaymentAmount: "10000",
        AutoPayThreshold: "5000",
    })
    
    // Attach the recorder using the helper function
    x402.WithPaymentRecorder(recorder)(transport)
    
    // Make requests...
    
    // Verify payments
    assert.Equal(t, 2, recorder.PaymentCount()) // Attempt + Success events
    lastPayment := recorder.LastPayment()
    assert.Equal(t, x402.PaymentEventSuccess, lastPayment.Type)
    assert.Equal(t, "20000", recorder.TotalAmount())
}
```

## Supported Networks

- `base` - Base Mainnet
- `base-sepolia` - Base Sepolia Testnet
- `avalanche` - Avalanche C-Chain
- `avalanche-fuji` - Avalanche Fuji Testnet
- `ethereum` - Ethereum Mainnet
- `sepolia` - Ethereum Sepolia Testnet

## Security Considerations

- **Private keys**: Never hardcode private keys. Use environment variables or secure key management.
- **Spending limits**: Always set reasonable `MaxPaymentAmount` and `RateLimits`.
- **Payment approval**: Use `PaymentCallback` for amounts above your comfort threshold.
- **Network verification**: The library verifies network and asset compatibility before signing.

## Examples

See the [examples](./examples) directory for more detailed examples:

- [Client](./examples/client/main.go) - Simple client that can pay for tool use
- [Server](./examples/server/main.go) - Server that collects payments for tool use

## Architecture

### Client Flow
1. Client makes MCP request through x402 transport
2. If server returns 402 Payment Required, transport extracts payment requirements
3. Transport uses configured signer to create payment authorization
4. Transport retries request with X-PAYMENT header
5. Server verifies and settles payment, then returns response

### Server Flow
1. Server receives MCP request
2. Checks if requested tool requires payment
3. If no payment provided, returns 402 with payment requirements
4. If payment provided, verifies with facilitator service
5. Settles payment on-chain (unless in verify-only mode)
6. Executes tool and returns response with payment confirmation

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- [MCP-Go](https://github.com/mark3labs/mcp-go) for the excellent MCP implementation
- [x402 Protocol](https://github.com/coinbase/x402) for the payment specification
- [go-ethereum](https://github.com/ethereum/go-ethereum) for crypto utilities
