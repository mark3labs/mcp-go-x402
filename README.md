# mcp-go-x402

X402 payment protocol support for [MCP-Go](https://github.com/mark3labs/mcp-go) clients.

This library provides a transport implementation that adds [x402 protocol](https://github.com/coinbase/x402) support to MCP clients, enabling automatic micropayments for accessing paid MCP services.

## Features

- 🔌 **Drop-in replacement**: Fully compatible with mcp-go transport interface
- 💰 **Automatic payments**: Handles 402 responses transparently
- 🔐 **Multiple signers**: Support for private keys, mnemonics, keystores
- 📊 **Budget management**: Configurable spending limits and rate limiting
- 🎯 **Smart payment**: Automatic for small amounts, callbacks for large
- 🧪 **Testing support**: Mock signers and payment recorders for easy testing

## Installation

```bash
go get github.com/mark3labs/mcp-go-x402
```

## Quick Start

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

## Configuration Options

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

## Signer Options

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

- [Basic Usage](./examples/basic/main.go) - Simple payment setup

## Roadmap

- ✅ **MCP Client** - Complete support for x402 payments in MCP clients
- ⏳ **MCP Server** - TODO: Add x402 payment collection support for MCP servers

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- [MCP-Go](https://github.com/mark3labs/mcp-go) for the excellent MCP implementation
- [x402 Protocol](https://github.com/coinbase/x402) for the payment specification
- [go-ethereum](https://github.com/ethereum/go-ethereum) for crypto utilities