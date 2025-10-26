package x402

import "math/big"

// Helper functions for common client payment options

// AcceptUSDCBase creates a client payment option for USDC on Base mainnet
func AcceptUSDCBase() ClientPaymentOption {
	return ClientPaymentOption{
		PaymentRequirement: PaymentRequirement{
			Scheme:  "exact",
			Network: "base",
			Asset:   "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
			Extra: map[string]string{
				"name":    "USD Coin",
				"version": "2",
			},
		},
		Priority: 1,                // Default high priority for Base (cheap & fast)
		ChainID:  big.NewInt(8453), // Base mainnet chain ID
	}
}

// AcceptUSDCBaseSepolia creates a client payment option for USDC on Base Sepolia testnet
func AcceptUSDCBaseSepolia() ClientPaymentOption {
	return ClientPaymentOption{
		PaymentRequirement: PaymentRequirement{
			Scheme:  "exact",
			Network: "base-sepolia",
			Asset:   "0x036CbD53842c5426634e7929541eC2318f3dCF7e", // USDC on Base Sepolia
			Extra: map[string]string{
				"name":    "USDC",
				"version": "2",
			},
		},
		Priority: 1,
		ChainID:  big.NewInt(84532), // Base Sepolia chain ID
	}
}

// Fluent API for customization

// WithPriority sets the priority for this payment option
func (opt ClientPaymentOption) WithPriority(p int) ClientPaymentOption {
	opt.Priority = p
	return opt
}

// WithMaxAmount sets the maximum amount the client is willing to pay with this option
func (opt ClientPaymentOption) WithMaxAmount(amount string) ClientPaymentOption {
	opt.MaxAmount = amount
	return opt
}

// WithMinBalance sets the minimum balance to maintain (won't use if balance would fall below)
func (opt ClientPaymentOption) WithMinBalance(amount string) ClientPaymentOption {
	opt.MinBalance = amount
	return opt
}

// AcceptUSDCPolygon creates a client payment option for USDC on Polygon mainnet
func AcceptUSDCPolygon() ClientPaymentOption {
	return ClientPaymentOption{
		PaymentRequirement: PaymentRequirement{
			Scheme:  "exact",
			Network: "polygon",
			Asset:   "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", // USDC on Polygon
			Extra: map[string]string{
				"name":    "USD Coin",
				"version": "2",
			},
		},
		Priority: 2,               // Medium priority
		ChainID:  big.NewInt(137), // Polygon mainnet chain ID
	}
}

// AcceptUSDCPolygonAmoy creates a client payment option for USDC on Polygon Amoy testnet
func AcceptUSDCPolygonAmoy() ClientPaymentOption {
	return ClientPaymentOption{
		PaymentRequirement: PaymentRequirement{
			Scheme:  "exact",
			Network: "polygon-amoy",
			Asset:   "0x41E94Eb019C0762f9Bfcf9Fb1E58725BfB0e7582", // USDC on Polygon Amoy
			Extra: map[string]string{
				"name":    "USDC",
				"version": "2",
			},
		},
		Priority: 2,
		ChainID:  big.NewInt(80002), // Polygon Amoy testnet chain ID
	}
}

// AcceptUSDCAvalanche creates a client payment option for USDC on Avalanche C-Chain mainnet
func AcceptUSDCAvalanche() ClientPaymentOption {
	return ClientPaymentOption{
		PaymentRequirement: PaymentRequirement{
			Scheme:  "exact",
			Network: "avalanche",
			Asset:   "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E", // USDC on Avalanche C-Chain
			Extra: map[string]string{
				"name":    "USD Coin",
				"version": "2",
			},
		},
		Priority: 2,                 // Medium priority
		ChainID:  big.NewInt(43114), // Avalanche C-Chain mainnet chain ID
	}
}

// AcceptUSDCAvalancheFuji creates a client payment option for USDC on Avalanche Fuji testnet
func AcceptUSDCAvalancheFuji() ClientPaymentOption {
	return ClientPaymentOption{
		PaymentRequirement: PaymentRequirement{
			Scheme:  "exact",
			Network: "avalanche-fuji",
			Asset:   "0x5425890298aed601595a70AB815c96711a31Bc65", // USDC on Avalanche Fuji
			Extra: map[string]string{
				"name":    "USDC",
				"version": "2",
			},
		},
		Priority: 2,
		ChainID:  big.NewInt(43113), // Avalanche Fuji testnet chain ID
	}
}

// AcceptUSDCSolana creates a client payment option for USDC on Solana mainnet
func AcceptUSDCSolana() ClientPaymentOption {
	return ClientPaymentOption{
		PaymentRequirement: PaymentRequirement{
			Scheme:  "exact",
			Network: "solana",
			Asset:   "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", // USDC mint
			Extra: map[string]string{
				"name":     "USD Coin",
				"decimals": "6",
			},
		},
		Priority:  2,
		NetworkID: "mainnet-beta",
	}
}

// AcceptUSDCSolanaDevnet creates a client payment option for USDC on Solana devnet
func AcceptUSDCSolanaDevnet() ClientPaymentOption {
	return ClientPaymentOption{
		PaymentRequirement: PaymentRequirement{
			Scheme:  "exact",
			Network: "solana-devnet",
			Asset:   "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU", // Devnet USDC mint
			Extra: map[string]string{
				"name":     "USDC (Devnet)",
				"decimals": "6",
			},
		},
		Priority:  2,
		NetworkID: "devnet",
	}
}
