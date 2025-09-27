package server

// Helper functions for common payment requirements with USDC on Base network

// RequireUSDCBase creates a payment requirement for USDC on Base mainnet
func RequireUSDCBase(payTo, amount, description string) PaymentRequirement {
	return PaymentRequirement{
		Scheme:            "eip3009",
		Network:           "base",
		Asset:             "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", // USDC on Base
		PayTo:             payTo,
		MaxAmountRequired: amount,
		Description:       description,
		MimeType:          "application/json",
		MaxTimeoutSeconds: 60,
		Extra: map[string]string{
			"name":    "USD Coin",
			"version": "2",
		},
	}
}

// RequireUSDCBaseSepolia creates a payment requirement for USDC on Base Sepolia testnet
func RequireUSDCBaseSepolia(payTo, amount, description string) PaymentRequirement {
	return PaymentRequirement{
		Scheme:            "eip3009",
		Network:           "base-sepolia",
		Asset:             "0x036CbD53842c5426634e7929541eC2318f3dCF7e", // USDC on Base Sepolia
		PayTo:             payTo,
		MaxAmountRequired: amount,
		Description:       description,
		MimeType:          "application/json",
		MaxTimeoutSeconds: 60,
		Extra: map[string]string{
			"name":    "USDC", // Exact name for Base Sepolia USDC
			"version": "2",
		},
	}
}
