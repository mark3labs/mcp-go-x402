# X402 MCP Server Example

This example demonstrates how to create an MCP server that requires x402 payments for tool invocations.

## Features

- **Payment-Required Tools**: The server hosts a `search` tool that requires 0.01 USDC per query
- **x402 Protocol Support**: Full implementation of the x402 HTTP micropayment protocol
- **Facilitator Integration**: Connects to an x402 facilitator service for payment verification

## Running the Server

### Prerequisites

1. An x402 facilitator service (can be local or remote)
2. A wallet address to receive payments
3. Network and asset configuration (defaults to Base Sepolia with USDC)

### Environment Variables

```bash
# Facilitator URL (defaults to https://facilitator.x402.rs)
export X402_FACILITATOR_URL=https://facilitator.x402.rs

# Wallet to receive payments
export X402_PAY_TO=0xYourWalletAddress

# Token contract address (defaults to USDC on Base mainnet)
export X402_ASSET=0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913

# Network (defaults to base)
export X402_NETWORK=base

# Server port (defaults to 8080)
export PORT=8080
```

### Start the Server

```bash
cd examples/server
go run main.go
```

The server will start on port 8080 (or the specified PORT). 
The MCP endpoint is available at the root URL (e.g., `http://localhost:8080`)
The server internally handles `/mcp` and other MCP protocol routes.

## Testing the Server

### 1. Without Payment (Returns 402)

```bash
# Note: The server internally routes to /mcp
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tools/call",
    "params": {
      "name": "search",
      "arguments": {
        "query": "test",
        "max_results": 5
      }
    },
    "id": 1
  }'
```

This will return a 402 Payment Required response with payment requirements.

### 2. With x402 Client

Use the x402 MCP client from the basic example to make paid requests:

```bash
# From the examples/basic directory
export WALLET_PRIVATE_KEY=your_private_key
export MCP_SERVER_URL=http://localhost:8080
go run main.go
```

Note: The client should connect to the root URL (e.g., `http://localhost:8080`), not `/mcp`. 
The server handles MCP routing internally.

## How It Works

1. **Request Interception**: The x402 handler intercepts incoming MCP requests
2. **Payment Check**: For tools marked as payable, it checks for the `X-PAYMENT` header
3. **Payment Verification**: If payment is provided, it's verified with the facilitator
4. **Settlement**: The payment is settled on-chain (unless in verify-only mode)
5. **Tool Execution**: After successful payment, the tool is executed
6. **Response**: The response includes an `X-PAYMENT-RESPONSE` header confirming the transaction

## Customization

### Adding More Tools

```go
// Add a free tool
srv.AddTool(
    mcp.NewTool("free-tool", 
        mcp.WithDescription("A free tool")),
    freeToolHandler,
)

// Add another paid tool
srv.AddPayableTool(
    mcp.NewTool("premium-tool",
        mcp.WithDescription("Premium feature")),
    premiumHandler,
    "50000", // 0.05 USDC
    "Premium feature access",
)
```

### Custom Payment Requirements

```go
customReq := &x402server.PaymentRequirement{
    Scheme:            "exact",
    Network:           "base-sepolia",
    MaxAmountRequired: "100000", // 0.1 USDC
    Asset:             "0xUSDC",
    PayTo:             "0xYourWallet",
    Description:       "Advanced processing",
    MaxTimeoutSeconds: 120,
}

srv.AddPayableToolWithRequirement(
    tool, handler, customReq,
)
```

## Architecture

The x402 server implementation uses a wrapper pattern:

```
HTTP Request
    ↓
X402Handler (payment verification)
    ↓
MCP StreamableHTTPServer
    ↓
MCP Server (tool execution)
    ↓
HTTP Response (with payment confirmation)
```

## Security Notes

- Always use HTTPS in production
- Keep your facilitator URL secure
- Validate payment amounts before processing
- Use environment variables for sensitive configuration
- Consider implementing rate limiting for payment requests

## Troubleshooting

### "Payment verification failed"
- Check facilitator is running and accessible
- Verify the payment signature is valid
- Ensure the payment amount matches requirements

### "Invalid session ID"
- Make sure MCP client properly initializes session
- Check that requests include proper session headers

### "Payment settlement failed"
- Verify blockchain network connectivity
- Check wallet has sufficient gas
- Ensure facilitator has settlement permissions