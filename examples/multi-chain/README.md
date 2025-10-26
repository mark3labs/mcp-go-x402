# Multi-Chain Payment Client Example

This example demonstrates how to configure an x402 client with support for multiple blockchain networks, including Polygon, Avalanche, and Base chains.

## Features

- Support for multiple EVM chains (Base, Polygon, Avalanche)
- Automatic fallback between mainnet and testnet
- Chain priority configuration
- Payment event callbacks

## Supported Chains

### Mainnet
- **Base** (Chain ID: 8453) - Low fees, fast transactions
- **Polygon** (Chain ID: 137) - Moderate fees, widely supported
- **Avalanche C-Chain** (Chain ID: 43114) - Fast finality

### Testnet
- **Base Sepolia** (Chain ID: 84532)
- **Polygon Amoy** (Chain ID: 80002)
- **Avalanche Fuji** (Chain ID: 43113)

## Usage

```bash
# Install dependencies
go mod download

# Run with default configuration (Base as primary chain)
go run main.go -server http://localhost:8080 -key YOUR_PRIVATE_KEY_HEX

# Use Polygon as primary chain
go run main.go -server http://localhost:8080 -key YOUR_PRIVATE_KEY_HEX -chain polygon

# Use Avalanche as primary chain
go run main.go -server http://localhost:8080 -key YOUR_PRIVATE_KEY_HEX -chain avalanche

# Or use environment variable for private key
export PRIVATE_KEY=YOUR_PRIVATE_KEY_HEX
go run main.go -server http://localhost:8080
```

## Configuration Options

### Command Line Flags
- `-server`: MCP server URL (default: `http://localhost:8080`)
- `-key`: Private key in hex format
- `-chain`: Preferred chain - `base`, `polygon`, or `avalanche` (default: `base`)

### Environment Variables
- `PRIVATE_KEY`: Alternative to `-key` flag

## How It Works

1. **Chain Selection**: Based on the `-chain` flag, the client configures payment options with different priorities:
   - The preferred chain gets priority 1 (tried first)
   - The testnet version of that chain gets priority 2 (fallback)
   - Alternative chains get lower priorities

2. **Automatic Fallback**: If a payment fails on the primary chain (e.g., insufficient balance), the client automatically tries the next chain in priority order.

3. **Payment Events**: The client logs payment attempts, successes, and failures with helpful emojis:
   - 📤 Payment attempt
   - ✅ Payment success
   - ❌ Payment failure

## Example Output

```
Configuring for Polygon as primary chain...
Payment signer configured with 4 payment options
Supported networks:
  - polygon (priority 1)
  - polygon-amoy (priority 2)
  - base (priority 3)
  - avalanche (priority 4)

Found 3 tools:
  - search: Search for information
  - calculate: Perform calculations
  - weather: Get weather information

Calling tool: search
📤 Attempting payment on polygon: 100000 to 0xRecipient...
✅ Payment successful on polygon: tx 0x123...
Tool result: {Content: "Search results..."}

✨ Multi-chain client example completed successfully!
```

## USDC Addresses

The example uses the official USDC contract addresses from Circle:

### Mainnet USDC
- Base: `0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913`
- Polygon: `0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359`
- Avalanche: `0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E`

### Testnet USDC
- Base Sepolia: `0x036CbD53842c5426634e7929541eC2318f3dCF7e`
- Polygon Amoy: `0x41E94Eb019C0762f9Bfcf9Fb1E58725BfB0e7582`
- Avalanche Fuji: `0x5425890298aed601595a70AB815c96711a31Bc65`

## Tips

1. **Test First**: Always test with testnets (Sepolia, Amoy, Fuji) before using mainnet
2. **Check Balances**: Ensure you have USDC on your preferred chains
3. **Gas Fees**: Keep native tokens (ETH, MATIC, AVAX) for gas fees
4. **Priority Matters**: Set lower priority numbers for preferred chains