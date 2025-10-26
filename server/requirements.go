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

// cloneStringMap creates a deep copy of a string map
func cloneStringMap(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// getExtraForNetwork retrieves cached extra data for a network
func getExtraForNetwork(network string) map[string]string {
	supportedPaymentsCacheMutex.RLock()
	defer supportedPaymentsCacheMutex.RUnlock()

	if kind, ok := supportedPaymentsCache[network]; ok {
		return cloneStringMap(kind.Extra)
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

// RequireUSDCPolygon creates a payment requirement for USDC on Polygon mainnet
func RequireUSDCPolygon(payTo, amount, description string) PaymentRequirement {
	return PaymentRequirement{
		Scheme:            "exact",
		Network:           "polygon",
		Asset:             "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", // USDC on Polygon
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

// RequireUSDCPolygonAmoy creates a payment requirement for USDC on Polygon Amoy testnet
func RequireUSDCPolygonAmoy(payTo, amount, description string) PaymentRequirement {
	return PaymentRequirement{
		Scheme:            "exact",
		Network:           "polygon-amoy",
		Asset:             "0x41E94Eb019C0762f9Bfcf9Fb1E58725BfB0e7582", // USDC on Polygon Amoy
		PayTo:             payTo,
		MaxAmountRequired: amount,
		Description:       description,
		MimeType:          "application/json",
		MaxTimeoutSeconds: 60,
		Extra: map[string]string{
			"name":    "USDC",
			"version": "2",
		},
	}
}

// RequireUSDCAvalanche creates a payment requirement for USDC on Avalanche C-Chain mainnet
func RequireUSDCAvalanche(payTo, amount, description string) PaymentRequirement {
	return PaymentRequirement{
		Scheme:            "exact",
		Network:           "avalanche",
		Asset:             "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E", // USDC on Avalanche C-Chain
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

// RequireUSDCAvalancheFuji creates a payment requirement for USDC on Avalanche Fuji testnet
func RequireUSDCAvalancheFuji(payTo, amount, description string) PaymentRequirement {
	return PaymentRequirement{
		Scheme:            "exact",
		Network:           "avalanche-fuji",
		Asset:             "0x5425890298aed601595a70AB815c96711a31Bc65", // USDC on Avalanche Fuji
		PayTo:             payTo,
		MaxAmountRequired: amount,
		Description:       description,
		MimeType:          "application/json",
		MaxTimeoutSeconds: 60,
		Extra: map[string]string{
			"name":    "USDC",
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
