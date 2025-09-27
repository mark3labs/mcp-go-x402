package x402

// Helper functions for common client payment options

// AcceptUSDCBase creates a client payment option for USDC on Base mainnet
func AcceptUSDCBase() ClientPaymentOption {
	return ClientPaymentOption{
		PaymentRequirement: PaymentRequirement{
			Scheme:  "eip3009",
			Network: "base",
			Asset:   "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
			Extra: map[string]string{
				"name":    "USD Coin",
				"version": "2",
			},
		},
		Priority: 1, // Default high priority for Base (cheap & fast)
	}
}

// AcceptUSDCBaseSepolia creates a client payment option for USDC on Base Sepolia testnet
func AcceptUSDCBaseSepolia() ClientPaymentOption {
	return ClientPaymentOption{
		PaymentRequirement: PaymentRequirement{
			Scheme:  "eip3009",
			Network: "base-sepolia",
			Asset:   "0x036CbD53842c5426634e7929541eC2318f3dCF7e", // USDC on Base Sepolia
			Extra: map[string]string{
				"name":    "USDC",
				"version": "2",
			},
		},
		Priority: 1,
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
