# SVM Client Example

Example x402 MCP client using Solana (SVM) payments.

## Prerequisites

1. Solana private key (base58 format) or Solana CLI keypair file
2. USDC tokens on Solana devnet for testing
3. Running MCP server that accepts Solana payments

## Setup

### Using Solana CLI keypair file

```bash
# Generate new keypair if needed
solana-keygen new --outfile ~/.config/solana/test-keypair.json

# Run client with keypair
go run main.go -keypair ~/.config/solana/test-keypair.json -v
```

### Using base58 private key

```bash
# Set environment variable
export SOLANA_PRIVATE_KEY="your_base58_private_key_here"

# Run client
go run main.go -v
```

## Usage

```bash
# Connect to local devnet server
go run main.go -server http://localhost:8080 -network devnet

# Connect to mainnet server
go run main.go -network mainnet -key <base58_key>

# Use keypair file with verbose output
go run main.go -keypair ~/.config/solana/id.json -v
```

## Flags

- `-key` - Solana private key in base58 format (or set SOLANA_PRIVATE_KEY env var)
- `-keypair` - Path to Solana CLI keypair file (alternative to -key)
- `-server` - MCP server URL (default: http://localhost:8080)
- `-network` - Network to use: devnet or mainnet (default: devnet)
- `-v` - Enable verbose output

## Getting Devnet USDC

1. Get devnet SOL from faucet: https://faucet.solana.com/
2. Get devnet USDC tokens (requires SOL for transaction fees)
