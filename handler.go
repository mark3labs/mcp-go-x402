package x402

import (
	"context"
	"fmt"
	"math/big"
)

// PaymentHandler handles x402 payment operations
type PaymentHandler struct {
	signer        PaymentSigner
	budgetManager *BudgetManager
	config        *HandlerConfig
}

// HandlerConfig configures the payment handler
type HandlerConfig struct {
	MaxPaymentAmount string
	AutoPayThreshold string // Automatically pay if below this amount
	RateLimits       *RateLimits
	PaymentCallback  func(amount *big.Int, resource string) bool
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(signer PaymentSigner, config *HandlerConfig) (*PaymentHandler, error) {
	if signer == nil {
		return nil, fmt.Errorf("signer cannot be nil")
	}

	if config == nil {
		config = &HandlerConfig{
			MaxPaymentAmount: "1000000", // Default 1 USDC
			AutoPayThreshold: "100000",  // Default 0.1 USDC
		}
	}

	budgetManager, err := NewBudgetManager(config.MaxPaymentAmount, config.RateLimits)
	if err != nil {
		return nil, err
	}

	return &PaymentHandler{
		signer:        signer,
		budgetManager: budgetManager,
		config:        config,
	}, nil
}

// ShouldPay determines if a payment should be made
func (h *PaymentHandler) ShouldPay(req PaymentRequirement) (bool, error) {
	amount := new(big.Int)
	if _, ok := amount.SetString(req.MaxAmountRequired, 10); !ok {
		return false, fmt.Errorf("invalid payment amount: %s", req.MaxAmountRequired)
	}

	// Validate amount is positive
	if amount.Sign() <= 0 {
		return false, fmt.Errorf("payment amount must be positive: %s", req.MaxAmountRequired)
	}

	// Check budget limits
	if err := h.budgetManager.CanSpend(amount, req.Resource); err != nil {
		return false, err
	}

	// Check auto-pay threshold
	if h.config.AutoPayThreshold != "" {
		threshold := new(big.Int)
		if _, ok := threshold.SetString(h.config.AutoPayThreshold, 10); !ok {
			return false, fmt.Errorf("invalid auto-pay threshold: %s", h.config.AutoPayThreshold)
		}

		if amount.Cmp(threshold) <= 0 {
			return true, nil
		}
	}

	// Use callback if provided and amount exceeds auto-pay threshold
	if h.config.PaymentCallback != nil {
		return h.config.PaymentCallback(amount, req.Resource), nil
	}

	// Default: pay if within max amount
	return true, nil
}

// CreatePayment creates a signed payment for the given requirements
func (h *PaymentHandler) CreatePayment(ctx context.Context, reqs PaymentRequirementsResponse) (*PaymentPayload, error) {
	// Select best payment method
	selected, err := h.selectPaymentMethod(reqs.Accepts)
	if err != nil {
		return nil, err
	}

	// Check if we should pay
	shouldPay, err := h.ShouldPay(*selected)
	if err != nil {
		return nil, err
	}

	if !shouldPay {
		return nil, fmt.Errorf("payment declined by policy")
	}

	// Sign the payment
	payment, err := h.signer.SignPayment(ctx, *selected)
	if err != nil {
		return nil, fmt.Errorf("failed to sign payment: %w", err)
	}

	// Record the payment
	amount := new(big.Int)
	if _, ok := amount.SetString(selected.MaxAmountRequired, 10); !ok {
		return nil, fmt.Errorf("invalid payment amount for recording: %s", selected.MaxAmountRequired)
	}
	h.budgetManager.RecordPayment(amount, selected.Resource)

	return payment, nil
}

// selectPaymentMethod selects the best payment method from available options
func (h *PaymentHandler) selectPaymentMethod(accepts []PaymentRequirement) (*PaymentRequirement, error) {
	if len(accepts) == 0 {
		return nil, ErrNoAcceptablePayment
	}

	var best *PaymentRequirement
	var bestAmount *big.Int

	for _, req := range accepts {
		// Check if we support this network
		if !h.signer.SupportsNetwork(req.Network) {
			continue
		}

		// Check if we have this asset
		if !h.signer.HasAsset(req.Asset, req.Network) {
			continue
		}

		// Check scheme (only support "exact" for now)
		if req.Scheme != "exact" {
			continue
		}

		amount := new(big.Int)
		if _, ok := amount.SetString(req.MaxAmountRequired, 10); !ok {
			// Skip invalid amounts
			continue
		}

		// Skip zero or negative amounts
		if amount.Sign() <= 0 {
			continue
		}

		// Select the cheapest option
		if best == nil || amount.Cmp(bestAmount) < 0 {
			best = &req
			bestAmount = amount
		}
	}

	if best == nil {
		return nil, ErrNoAcceptablePayment
	}

	return best, nil
}

// GetMetrics returns budget metrics
func (h *PaymentHandler) GetMetrics() BudgetMetrics {
	return h.budgetManager.GetMetrics()
}
