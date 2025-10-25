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

// AcceptUSDCSolanaDevnet creates a ClientPaymentOption configured for USDC on Solana devnet.
// The option requires an exact payment to the devnet USDC mint, sets Priority to 2, and uses NetworkID "devnet".
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