package server

import (
	"context"
	"log"
)

// Helper functions for common payment requirements with USDC on Base network

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
// The feePayer is automatically fetched from the facilitator's /supported endpoint
func RequireUSDCSolana(facilitatorURL, payTo, amount, description string) PaymentRequirement {
	feePayer := fetchFeePayerFromFacilitator(facilitatorURL, "solana")
	return PaymentRequirement{
		Scheme:            "exact",
		Network:           "solana",
		MaxAmountRequired: amount,
		Asset:             "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v", // USDC mint
		PayTo:             payTo,
		Description:       description,
		MimeType:          "application/json",
		MaxTimeoutSeconds: 60,
		Extra: map[string]string{
			"feePayer": feePayer,
			"decimals": "6",
			"name":     "USD Coin",
		},
	}
}

// RequireUSDCSolanaDevnet creates a payment requirement for USDC on Solana devnet
// The feePayer is automatically fetched from the facilitator's /supported endpoint
func RequireUSDCSolanaDevnet(facilitatorURL, payTo, amount, description string) PaymentRequirement {
	feePayer := fetchFeePayerFromFacilitator(facilitatorURL, "solana-devnet")
	return PaymentRequirement{
		Scheme:            "exact",
		Network:           "solana-devnet",
		MaxAmountRequired: amount,
		Asset:             "4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU", // Devnet USDC
		PayTo:             payTo,
		Description:       description,
		MimeType:          "application/json",
		MaxTimeoutSeconds: 60,
		Extra: map[string]string{
			"feePayer": feePayer,
			"decimals": "6",
			"name":     "USDC (Devnet)",
		},
	}
}

// fetchFeePayerFromFacilitator fetches the feePayer address for a given network from the facilitator
func fetchFeePayerFromFacilitator(facilitatorURL, network string) string {
	facilitator := NewHTTPFacilitator(facilitatorURL)
	ctx := context.Background()

	supported, err := facilitator.GetSupported(ctx)
	if err != nil {
		log.Printf("Warning: failed to fetch supported payments from facilitator: %v", err)
		return ""
	}

	for _, kind := range supported {
		if kind.Network == network && kind.Extra != nil {
			if feePayer, ok := kind.Extra["feePayer"]; ok {
				return feePayer
			}
		}
	}

	log.Printf("Warning: feePayer not found for network %s in facilitator's supported list", network)
	return ""
}
