# SVM Server Example

Example x402 MCP server that requires Solana (SVM) payments.

## Prerequisites

1. Solana recipient address to receive payments
2. Running x402 facilitator with SVM support

## Setup

```bash
# Run with devnet configuration
go run main.go \
  -pay-to YOUR_SOLANA_ADDRESS \
  -devnet \
  -v
```

## Usage

```bash
# Devnet server (for testing)
go run main.go \
  -pay-to 7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU \
  -devnet

# Mainnet server
go run main.go \
  -pay-to YOUR_MAINNET_ADDRESS

# Custom port and facilitator
go run main.go \
  -port 9090 \
  -facilitator https://facilitator.payai.network \
  -pay-to YOUR_ADDRESS \
  -devnet
```

## Flags

- `-port` - Port to listen on (default: 8080)
- `-facilitator` - x402 facilitator URL (default: https://facilitator.payai.network)
- `-pay-to` - Payment recipient Solana address (required)
- `-verify-only` - Only verify payments, don't settle on-chain
- `-devnet` - Use Solana devnet instead of mainnet
- `-v` - Enable verbose output

## Tools

- **search** (paid): Requires 0.01 USDC payment on Solana
- **echo** (free): No payment required
- **test-feature** (devnet only): Requires 0.001 USDC payment


