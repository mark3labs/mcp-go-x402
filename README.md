# mcp-go-x402

X402 payment protocol support for [MCP-Go](https://github.com/mark3labs/mcp-go) clients and servers.

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
    // Create signer with your private key
    signer, err := x402.NewPrivateKeySigner("YOUR_PRIVATE_KEY_HEX")
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
        DefaultPayTo:    "0xYourWallet",
        DefaultAsset:    "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
        DefaultNetwork:  "base",
    }
    
    // Create x402 server
    srv := x402server.NewX402Server("my-server", "1.0.0", config)
    
    // Add a free tool
    srv.AddTool(
        mcp.NewTool("free-tool", 
            mcp.WithDescription("This tool is free")),
        freeToolHandler,
    )
    
    // Add a paid tool (0.01 USDC per call)
    srv.AddPayableTool(
        mcp.NewTool("premium-tool",
            mcp.WithDescription("Premium feature"),
            mcp.WithString("input", mcp.Required())),
        premiumToolHandler,
        "10000", // 0.01 USDC (6 decimals)
        "Access to premium feature",
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
config := x402.Config{
    ServerURL:        "https://server.example.com",
    Signer:           signer,
    MaxPaymentAmount: "1000000",  // Maximum 1 USDC per request
    AutoPayThreshold: "100000",   // Auto-pay up to 0.1 USDC
}
```

### With Rate Limiting

```go
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
    DefaultPayTo:    "0xYourWallet",
    DefaultAsset:    "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
    DefaultNetwork:  "base",
    VerifyOnly:      false, // Set to true for testing without settlement
}
```

### Custom Payment Requirements

```go
// Add tool with custom payment requirements
customReq := &x402server.PaymentRequirement{
    Scheme:            "exact",
    Network:           "base",
    MaxAmountRequired: "50000", // 0.05 USDC
    Asset:             "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
    PayTo:             "0xYourWallet",
    Description:       "Advanced processing",
    MimeType:          "application/json",
    MaxTimeoutSeconds: 120,
    Extra: map[string]string{
        "name":    "USD Coin",  // EIP-712 domain name for USDC
        "version": "2",
    },
}

srv.AddPayableToolWithRequirement(tool, handler, customReq)
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
signer, err := x402.NewPrivateKeySigner("0xYourPrivateKeyHex")
```

### Mnemonic (BIP-39)

```go
signer, err := x402.NewMnemonicSigner(
    "your twelve word mnemonic phrase here ...",
    "m/44'/60'/0'/0/0", // Optional: derivation path
)
```

### Keystore File

```go
keystoreJSON, _ := os.ReadFile("keystore.json")
signer, err := x402.NewKeystoreSigner(keystoreJSON, "password")
```

### Custom Signer

```go
type MyCustomSigner struct{}

func (s *MyCustomSigner) SignPayment(ctx context.Context, req x402.PaymentRequirement) (*x402.PaymentPayload, error) {
    // Your custom signing logic
    // Could use hardware wallet, remote signer, etc.
}

func (s *MyCustomSigner) GetAddress() string {
    return "0xYourAddress"
}

func (s *MyCustomSigner) SupportsNetwork(network string) bool {
    return network == "base" || network == "base-sepolia"
}

func (s *MyCustomSigner) HasAsset(asset, network string) bool {
    // Check if you have the required asset
    return true
}
```

## Testing

### Using Mock Signer

```go
func TestMyMCPClient(t *testing.T) {
    // No real wallet needed for tests
    signer := x402.NewMockSigner("0xTestWallet")
    
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
    signer := x402.NewMockSigner("0xTestWallet")
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

- [Basic Client](./examples/basic/main.go) - Simple client with automatic payments
- [X402 Server](./examples/server/main.go) - Server that collects payments for tools

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

## Roadmap

- ✅ **MCP Client** - Complete support for x402 payments in MCP clients
- ✅ **MCP Server** - Complete support for x402 payment collection in MCP servers
- 🔄 **Payment caching** - Cache successful payments for session reuse
- 🔄 **Multi-asset support** - Support for multiple payment tokens
- 🔄 **Subscription model** - Support for time-based access passes

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- [MCP-Go](https://github.com/mark3labs/mcp-go) for the excellent MCP implementation
- [x402 Protocol](https://github.com/coinbase/x402) for the payment specification
- [go-ethereum](https://github.com/ethereum/go-ethereum) for crypto utilities