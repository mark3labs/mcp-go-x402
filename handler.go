package x402

import (
	"context"
	"fmt"
	"math/big"
	"sort"
)

// PaymentHandler handles x402 payment operations
type PaymentHandler struct {
	signer PaymentSigner
	config *HandlerConfig
}

// HandlerConfig configures the payment handler
type HandlerConfig struct {
	PaymentCallback func(amount *big.Int, resource string) bool
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(signer PaymentSigner, config *HandlerConfig) (*PaymentHandler, error) {
	if signer == nil {
		return nil, fmt.Errorf("signer cannot be nil")
	}

	if config == nil {
		config = &HandlerConfig{}
	}

	return &PaymentHandler{
		signer: signer,
		config: config,
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

	// Use callback if provided
	if h.config.PaymentCallback != nil {
		return h.config.PaymentCallback(amount, req.Resource), nil
	}

	// Default: approve payment
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

	return payment, nil
}

// selectPaymentMethod selects the best payment method from available options
func (h *PaymentHandler) selectPaymentMethod(accepts []PaymentRequirement) (*PaymentRequirement, error) {
	if len(accepts) == 0 {
		return nil, ErrNoAcceptablePayment
	}

	type candidate struct {
		req      PaymentRequirement
		priority int
		amount   *big.Int
	}

	var candidates []candidate

	for _, req := range accepts {
		// Check if we support this network and asset
		option := h.signer.GetPaymentOption(req.Network, req.Asset)
		if option == nil {
			continue
		}

		// Check scheme matches
		if option.Scheme != req.Scheme {
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

		// Check if within client's max amount for this option
		if option.MaxAmount != "" {
			maxAmount := new(big.Int)
			if _, ok := maxAmount.SetString(option.MaxAmount, 10); ok {
				if amount.Cmp(maxAmount) > 0 {
					// Required amount exceeds client's max for this option
					continue
				}
			}
		}

		candidates = append(candidates, candidate{
			req:      req,
			priority: option.Priority,
			amount:   amount,
		})
	}

	if len(candidates) == 0 {
		return nil, ErrNoAcceptablePayment
	}

	// Sort by priority first, then by amount
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].priority != candidates[j].priority {
			return candidates[i].priority < candidates[j].priority
		}
		return candidates[i].amount.Cmp(candidates[j].amount) < 0
	})

	return &candidates[0].req, nil
}
