package server

import (
	"sync"
)

// Helper functions for common payment requirements with USDC on Base network

var (
	// supportedPaymentsCache stores supported payment info by network
	supportedPaymentsCache      = make(map[string]SupportedKind)
	supportedPaymentsCacheMutex sync.RWMutex
)

// SetSupportedPayments caches the supported payment methods from the facilitator
// This is called automatically when the server initializes
func SetSupportedPayments(supported []SupportedKind) {
	supportedPaymentsCacheMutex.Lock()
	defer supportedPaymentsCacheMutex.Unlock()

	for _, kind := range supported {
		supportedPaymentsCache[kind.Network] = kind
	}
}

// getExtraForNetwork returns the Extra fields for a network from cached supported payments
func getExtraForNetwork(network string) map[string]string {
	supportedPaymentsCacheMutex.RLock()
	defer supportedPaymentsCacheMutex.RUnlock()

	if kind, ok := supportedPaymentsCache[network]; ok {
		return kind.Extra
	}
	return nil
}

// RequireUSDCBase creates a payment requirement for USDC on Base mainnet
func RequireUSDCBase(payTo, amount, description string) PaymentRequirement {
	return PaymentRequirement{
		Scheme:            "exact",
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
		Scheme:            "exact",
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

// RequireUSDCSolana creates a payment requirement for USDC on Solana mainnet
// The feePayer is automatically populated from the facilitator's /supported endpoint
func RequireUSDCSolana(payTo, amount, description string) PaymentRequirement {
	extra := map[string]string{
		"decimals": "6",
		"name":     "USD Coin",
	}

	// Merge in any extra fields from facilitator's supported payments (including feePayer)
	if facilitatorExtra := getExtraForNetwork("solana"); facilitatorExtra != nil {
		for k, v := range facilitatorExtra {
			extra[k] = v
		}
	}

	return PaymentRequirement{
		Scheme:            "exact",
		Network:           "solana",
		MaxAmountRequired: amount,
		Asset:             "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", // USDC mint
		PayTo:             payTo,
		Description:       description,
		MimeType:          "application/json",
		MaxTimeoutSeconds: 60,
		Extra:             extra,
	}
}

// RequireUSDCSolanaDevnet creates a payment requirement for USDC on Solana devnet
// The feePayer is automatically populated from the facilitator's /supported endpoint
func RequireUSDCSolanaDevnet(payTo, amount, description string) PaymentRequirement {
	extra := map[string]string{
		"decimals": "6",
		"name":     "USDC (Devnet)",
	}

	// Merge in any extra fields from facilitator's supported payments (including feePayer)
	if facilitatorExtra := getExtraForNetwork("solana-devnet"); facilitatorExtra != nil {
		for k, v := range facilitatorExtra {
			extra[k] = v
		}
	}

	return PaymentRequirement{
		Scheme:            "exact",
		Network:           "solana-devnet",
		MaxAmountRequired: amount,
		Asset:             "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU", // Devnet USDC
		PayTo:             payTo,
		Description:       description,
		MimeType:          "application/json",
		MaxTimeoutSeconds: 60,
		Extra:             extra,
	}
}
