package x402

import (
	"fmt"
	"math/big"
	"sync"
	"time"
)

// RateLimits defines rate limiting configuration
type RateLimits struct {
	MaxPaymentsPerMinute int
	MaxAmountPerHour     string
}

// BudgetManager manages spending limits and tracking
type BudgetManager struct {
	mu               sync.RWMutex
	maxPaymentAmount *big.Int
	rateLimits       *RateLimits

	// Tracking
	payments        []paymentRecord
	hourlySpent     *big.Int
	hourlyResetTime time.Time
	minuteCount     int
	minuteResetTime time.Time
}

type paymentRecord struct {
	timestamp time.Time
	amount    *big.Int
	resource  string
}

// NewBudgetManager creates a new budget manager
func NewBudgetManager(maxPaymentAmount string, rateLimits *RateLimits) (*BudgetManager, error) {
	maxAmount := new(big.Int)
	if maxPaymentAmount != "" {
		if _, ok := maxAmount.SetString(maxPaymentAmount, 10); !ok {
			return nil, fmt.Errorf("invalid max payment amount: %s", maxPaymentAmount)
		}
		// Validate max amount is positive
		if maxAmount.Sign() <= 0 {
			return nil, fmt.Errorf("max payment amount must be positive: %s", maxPaymentAmount)
		}
	}

	// Validate rate limits
	if rateLimits != nil && rateLimits.MaxAmountPerHour != "" {
		hourlyMax := new(big.Int)
		if _, ok := hourlyMax.SetString(rateLimits.MaxAmountPerHour, 10); !ok {
			return nil, fmt.Errorf("invalid max hourly amount: %s", rateLimits.MaxAmountPerHour)
		}
		if hourlyMax.Sign() <= 0 {
			return nil, fmt.Errorf("max hourly amount must be positive: %s", rateLimits.MaxAmountPerHour)
		}
	}

	bm := &BudgetManager{
		maxPaymentAmount: maxAmount,
		rateLimits:       rateLimits,
		hourlySpent:      big.NewInt(0),
		hourlyResetTime:  time.Now().Add(time.Hour),
		minuteResetTime:  time.Now().Add(time.Minute),
		payments:         make([]paymentRecord, 0),
	}

	return bm, nil
}

// CanSpend checks if a payment is within budget limits
func (bm *BudgetManager) CanSpend(amount *big.Int, resource string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	now := time.Now()

	// Check max payment amount
	if bm.maxPaymentAmount != nil && bm.maxPaymentAmount.Cmp(big.NewInt(0)) > 0 {
		if amount.Cmp(bm.maxPaymentAmount) > 0 {
			return ErrAmountExceedsLimit
		}
	}

	if bm.rateLimits != nil {
		// Reset counters if needed (use >= for boundary condition)
		if now.After(bm.hourlyResetTime) || now.Equal(bm.hourlyResetTime) {
			bm.hourlySpent = big.NewInt(0)
			bm.hourlyResetTime = now.Add(time.Hour)
		}

		if now.After(bm.minuteResetTime) || now.Equal(bm.minuteResetTime) {
			bm.minuteCount = 0
			bm.minuteResetTime = now.Add(time.Minute)
		}

		// Check rate limits
		if bm.rateLimits.MaxPaymentsPerMinute > 0 {
			if bm.minuteCount >= bm.rateLimits.MaxPaymentsPerMinute {
				return ErrRateLimitExceeded
			}
		}

		if bm.rateLimits.MaxAmountPerHour != "" {
			maxHourly := new(big.Int)
			if _, ok := maxHourly.SetString(bm.rateLimits.MaxAmountPerHour, 10); !ok {
				// This should have been validated in NewBudgetManager, but check anyway
				return fmt.Errorf("invalid max hourly amount: %s", bm.rateLimits.MaxAmountPerHour)
			}

			newTotal := new(big.Int).Add(bm.hourlySpent, amount)
			if newTotal.Cmp(maxHourly) > 0 {
				return ErrBudgetExceeded
			}
		}
	}

	return nil
}

// RecordPayment records a successful payment
func (bm *BudgetManager) RecordPayment(amount *big.Int, resource string) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	now := time.Now()

	bm.payments = append(bm.payments, paymentRecord{
		timestamp: now,
		amount:    new(big.Int).Set(amount),
		resource:  resource,
	})

	if bm.rateLimits != nil {
		bm.minuteCount++
		bm.hourlySpent.Add(bm.hourlySpent, amount)
	}

	// Clean up old payment records (keep last 24 hours)
	cutoff := now.Add(-24 * time.Hour)
	for i, p := range bm.payments {
		if p.timestamp.After(cutoff) {
			bm.payments = bm.payments[i:]
			break
		}
	}
}

// GetMetrics returns current spending metrics
func (bm *BudgetManager) GetMetrics() BudgetMetrics {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	total := big.NewInt(0)
	for _, p := range bm.payments {
		total.Add(total, p.amount)
	}

	return BudgetMetrics{
		TotalSpent:   total.String(),
		HourlySpent:  bm.hourlySpent.String(),
		PaymentCount: len(bm.payments),
		MinuteCount:  bm.minuteCount,
	}
}

// BudgetMetrics contains spending metrics
type BudgetMetrics struct {
	TotalSpent   string
	HourlySpent  string
	PaymentCount int
	MinuteCount  int
}
