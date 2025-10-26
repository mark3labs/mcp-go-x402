package x402

import (
	"context"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientPaymentOptions(t *testing.T) {
	t.Run("RequiresAtLeastOneOption", func(t *testing.T) {
		_, err := NewPrivateKeySigner("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least one payment option")
	})

	t.Run("HelperFunctionsIncludeChainID", func(t *testing.T) {
		baseOption := AcceptUSDCBase()
		assert.Equal(t, big.NewInt(8453), baseOption.ChainID, "Base mainnet should have chain ID 8453")

		sepoliaOption := AcceptUSDCBaseSepolia()
		assert.Equal(t, big.NewInt(84532), sepoliaOption.ChainID, "Base Sepolia should have chain ID 84532")

		// Test new chains
		polygonOption := AcceptUSDCPolygon()
		assert.Equal(t, big.NewInt(137), polygonOption.ChainID, "Polygon mainnet should have chain ID 137")
		assert.Equal(t, "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", polygonOption.Asset)
		assert.Equal(t, "polygon", polygonOption.Network)

		polygonAmoyOption := AcceptUSDCPolygonAmoy()
		assert.Equal(t, big.NewInt(80002), polygonAmoyOption.ChainID, "Polygon Amoy should have chain ID 80002")
		assert.Equal(t, "0x41E94Eb019C0762f9Bfcf9Fb1E58725BfB0e7582", polygonAmoyOption.Asset)
		assert.Equal(t, "polygon-amoy", polygonAmoyOption.Network)

		avalancheOption := AcceptUSDCAvalanche()
		assert.Equal(t, big.NewInt(43114), avalancheOption.ChainID, "Avalanche mainnet should have chain ID 43114")
		assert.Equal(t, "0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E", avalancheOption.Asset)
		assert.Equal(t, "avalanche", avalancheOption.Network)

		avalancheFujiOption := AcceptUSDCAvalancheFuji()
		assert.Equal(t, big.NewInt(43113), avalancheFujiOption.ChainID, "Avalanche Fuji should have chain ID 43113")
		assert.Equal(t, "0x5425890298aed601595a70AB815c96711a31Bc65", avalancheFujiOption.Asset)
		assert.Equal(t, "avalanche-fuji", avalancheFujiOption.Network)
	})

	t.Run("AcceptsMultipleOptions", func(t *testing.T) {
		signer, err := NewPrivateKeySigner(
			"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			AcceptUSDCBase(),
			AcceptUSDCBaseSepolia(),
		)
		require.NoError(t, err)

		// Check Base support
		assert.True(t, signer.SupportsNetwork("base"))
		assert.True(t, signer.HasAsset("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", "base"))

		// Check Base Sepolia support
		assert.True(t, signer.SupportsNetwork("base-sepolia"))
		assert.True(t, signer.HasAsset("0x036CbD53842c5426634e7929541eC2318f3dCF7e", "base-sepolia"))

		// Check unsupported network
		assert.False(t, signer.SupportsNetwork("ethereum"))
		assert.False(t, signer.HasAsset("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", "ethereum"))
	})

	t.Run("AcceptsMultipleChainsIncludingNewOnes", func(t *testing.T) {
		signer, err := NewPrivateKeySigner(
			"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			AcceptUSDCBase().WithPriority(1),
			AcceptUSDCPolygon().WithPriority(2),
			AcceptUSDCAvalanche().WithPriority(3),
			AcceptUSDCPolygonAmoy().WithPriority(4),
			AcceptUSDCAvalancheFuji().WithPriority(5),
		)
		require.NoError(t, err)

		// Check all mainnet chains
		assert.True(t, signer.SupportsNetwork("base"))
		assert.True(t, signer.SupportsNetwork("polygon"))
		assert.True(t, signer.SupportsNetwork("avalanche"))

		// Check all testnet chains
		assert.True(t, signer.SupportsNetwork("polygon-amoy"))
		assert.True(t, signer.SupportsNetwork("avalanche-fuji"))

		// Check correct assets
		assert.True(t, signer.HasAsset("0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359", "polygon"))
		assert.True(t, signer.HasAsset("0xB97EF9Ef8734C71904D8002F8b6Bc66Dd9c48a6E", "avalanche"))
		assert.True(t, signer.HasAsset("0x41E94Eb019C0762f9Bfcf9Fb1E58725BfB0e7582", "polygon-amoy"))
		assert.True(t, signer.HasAsset("0x5425890298aed601595a70AB815c96711a31Bc65", "avalanche-fuji"))
	})

	t.Run("FluentAPI", func(t *testing.T) {
		option := AcceptUSDCBase().
			WithPriority(2).
			WithMaxAmount("100000").
			WithMinBalance("50000")

		assert.Equal(t, 2, option.Priority)
		assert.Equal(t, "100000", option.MaxAmount)
		assert.Equal(t, "50000", option.MinBalance)
		assert.Equal(t, "base", option.Network)
		assert.Equal(t, big.NewInt(8453), option.ChainID, "Chain ID should be preserved through fluent API")
	})

	t.Run("GetPaymentOption", func(t *testing.T) {
		signer, err := NewPrivateKeySigner(
			"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			AcceptUSDCBase().WithPriority(1),
			AcceptUSDCBaseSepolia().WithPriority(2),
		)
		require.NoError(t, err)

		// Get Base option
		baseOpt := signer.GetPaymentOption("base", "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")
		require.NotNil(t, baseOpt)
		assert.Equal(t, 1, baseOpt.Priority)
		assert.Equal(t, "base", baseOpt.Network)
		assert.Equal(t, big.NewInt(8453), baseOpt.ChainID)

		// Get Base Sepolia option
		sepoliaOpt := signer.GetPaymentOption("base-sepolia", "0x036CbD53842c5426634e7929541eC2318f3dCF7e")
		require.NotNil(t, sepoliaOpt)
		assert.Equal(t, 2, sepoliaOpt.Priority)
		assert.Equal(t, "base-sepolia", sepoliaOpt.Network)
		assert.Equal(t, big.NewInt(84532), sepoliaOpt.ChainID)

		// Try non-existent option
		nilOpt := signer.GetPaymentOption("ethereum", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
		assert.Nil(t, nilOpt)
	})

	t.Run("CustomPaymentOptionWithChainID", func(t *testing.T) {
		// Create a custom payment option for Ethereum mainnet
		customOption := ClientPaymentOption{
			PaymentRequirement: PaymentRequirement{
				Scheme:  "exact",
				Network: "ethereum",
				Asset:   "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", // USDC on Ethereum
				Extra: map[string]string{
					"name":    "USD Coin",
					"version": "2",
				},
			},
			Priority: 1,
			ChainID:  big.NewInt(1), // Ethereum mainnet
		}

		signer, err := NewPrivateKeySigner(
			"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			customOption,
		)
		require.NoError(t, err)

		// Verify the option is stored correctly
		option := signer.GetPaymentOption("ethereum", "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
		require.NotNil(t, option)
		assert.Equal(t, big.NewInt(1), option.ChainID)
	})
}

func TestPaymentSelection(t *testing.T) {
	t.Run("SelectsByPriority", func(t *testing.T) {
		signer, err := NewPrivateKeySigner(
			"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			AcceptUSDCBaseSepolia().WithPriority(2), // Lower priority
			AcceptUSDCBase().WithPriority(1),        // Higher priority
		)
		require.NoError(t, err)

		handler, err := NewPaymentHandler(signer, &HandlerConfig{})
		require.NoError(t, err)

		// Server accepts both options
		accepts := []PaymentRequirement{
			{
				Scheme:            "exact",
				Network:           "base",
				Asset:             "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
				MaxAmountRequired: "5000",
			},
			{
				Scheme:            "exact",
				Network:           "base-sepolia",
				Asset:             "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
				MaxAmountRequired: "5000",
			},
		}

		selected, err := handler.selectPaymentMethod(accepts)
		require.NoError(t, err)
		assert.Equal(t, "base", selected.Network) // Should select base (priority 1)
	})

	t.Run("SelectsCheaperWithSamePriority", func(t *testing.T) {
		signer, err := NewPrivateKeySigner(
			"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			AcceptUSDCBase().WithPriority(1),
			AcceptUSDCBaseSepolia().WithPriority(1), // Same priority
		)
		require.NoError(t, err)

		handler, err := NewPaymentHandler(signer, &HandlerConfig{})
		require.NoError(t, err)

		// Server accepts both with different prices
		accepts := []PaymentRequirement{
			{
				Scheme:            "exact",
				Network:           "base",
				Asset:             "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
				MaxAmountRequired: "10000", // More expensive
			},
			{
				Scheme:            "exact",
				Network:           "base-sepolia",
				Asset:             "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
				MaxAmountRequired: "5000", // Cheaper
			},
		}

		selected, err := handler.selectPaymentMethod(accepts)
		require.NoError(t, err)
		assert.Equal(t, "base-sepolia", selected.Network) // Should select cheaper option
	})

	t.Run("RespectsMaxAmount", func(t *testing.T) {
		signer, err := NewPrivateKeySigner(
			"0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			AcceptUSDCBase().WithMaxAmount("1000"), // Low max on Base
			AcceptUSDCBaseSepolia(),                // No limit on Sepolia
		)
		require.NoError(t, err)

		handler, err := NewPaymentHandler(signer, &HandlerConfig{})
		require.NoError(t, err)

		// Server requires more than Base max
		accepts := []PaymentRequirement{
			{
				Scheme:            "exact",
				Network:           "base",
				Asset:             "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
				MaxAmountRequired: "5000", // Exceeds client's Base max
			},
			{
				Scheme:            "exact",
				Network:           "base-sepolia",
				Asset:             "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
				MaxAmountRequired: "5000",
			},
		}

		selected, err := handler.selectPaymentMethod(accepts)
		require.NoError(t, err)
		assert.Equal(t, "base-sepolia", selected.Network) // Should skip Base due to max limit
	})
}

func TestMockSigner(t *testing.T) {
	t.Run("DefaultsToBaseSepolia", func(t *testing.T) {
		signer := NewMockSigner("0xTestWallet")
		assert.True(t, signer.SupportsNetwork("base-sepolia"))
		assert.True(t, signer.HasAsset("0x036CbD53842c5426634e7929541eC2318f3dCF7e", "base-sepolia"))
	})

	t.Run("AcceptsCustomOptions", func(t *testing.T) {
		signer := NewMockSigner("0xTestWallet", AcceptUSDCBase())
		assert.True(t, signer.SupportsNetwork("base"))
		assert.False(t, signer.SupportsNetwork("base-sepolia"))
	})

	t.Run("SignsWithCorrectScheme", func(t *testing.T) {
		signer := NewMockSigner("0xTestWallet", AcceptUSDCBase())

		payment, err := signer.SignPayment(context.Background(), PaymentRequirement{
			Scheme:            "exact",
			Network:           "base",
			Asset:             "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
			PayTo:             "0xRecipient",
			MaxAmountRequired: "1000",
			MaxTimeoutSeconds: 60,
		})

		require.NoError(t, err)
		assert.Equal(t, "exact", payment.Scheme)
		assert.Equal(t, "base", payment.Network)
	})
}
