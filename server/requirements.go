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
// SetSupportedPayments populates the package-level supported payments cache with the provided
// SupportedKind slice, indexing entries by their Network field and replacing any existing entry
// for the same network. It is safe for concurrent use.
func SetSupportedPayments(supported []SupportedKind) {
	supportedPaymentsCacheMutex.Lock()
	defer supportedPaymentsCacheMutex.Unlock()

	for _, kind := range supported {
		supportedPaymentsCache[kind.Network] = kind
	}
}

// cloneStringMap returns a shallow copy of the provided map of strings.
// If the input is nil, cloneStringMap returns nil.
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

// getExtraForNetwork retrieves a shallow copy of the Extra fields for the specified network from the supportedPaymentsCache.
// The returned map is a new map that may be modified by the caller; returns nil if no entry exists for the network.
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

// RequireUSDCBaseSepolia creates a PaymentRequirement configured for USDC on the Base Sepolia testnet.
// The requirement uses the Base Sepolia USDC mint address, the "exact" scheme, a 60-second timeout, and Extra fields "name" and "version".
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
// RequireUSDCSolana creates a PaymentRequirement for USDC on the Solana mainnet.
// It sets network to "solana", the asset to the USDC mint, a 60-second timeout,
// and default Extra fields "decimals"="6" and "name"="USD Coin".
// If facilitator-provided Extra fields exist for "solana", they are merged into
// the Extra map, overwriting the defaults (commonly used to supply a feePayer).
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
// RequireUSDCSolanaDevnet constructs a PaymentRequirement for USDC on the Solana Devnet.
// The requirement uses the Devnet USDC mint, sets network to "solana-devnet", a 60-second timeout,
// and merges any facilitator-provided Extra fields (including a `feePayer` if present) into the returned Extra map.
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