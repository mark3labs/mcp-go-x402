package x402

import (
	"math/big"
	"sync"
)

// PaymentRecorder records payment events for testing
type PaymentRecorder struct {
	mu     sync.RWMutex
	events []PaymentEvent
}

// NewPaymentRecorder creates a new payment recorder
func NewPaymentRecorder() *PaymentRecorder {
	return &PaymentRecorder{
		events: make([]PaymentEvent, 0),
	}
}

// Record records a payment event
func (r *PaymentRecorder) Record(event PaymentEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

// PaymentCount returns the number of recorded payments
func (r *PaymentRecorder) PaymentCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.events)
}

// LastPayment returns the most recent payment event
func (r *PaymentRecorder) LastPayment() *PaymentEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.events) == 0 {
		return nil
	}
	return &r.events[len(r.events)-1]
}

// GetEvents returns all recorded events
func (r *PaymentRecorder) GetEvents() []PaymentEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	events := make([]PaymentEvent, len(r.events))
	copy(events, r.events)
	return events
}

// Clear clears all recorded events
func (r *PaymentRecorder) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = make([]PaymentEvent, 0)
}

// SuccessfulPayments returns only successful payment events
func (r *PaymentRecorder) SuccessfulPayments() []PaymentEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var successful []PaymentEvent
	for _, event := range r.events {
		if event.Type == PaymentEventSuccess {
			successful = append(successful, event)
		}
	}
	return successful
}

// FailedPayments returns only failed payment events
func (r *PaymentRecorder) FailedPayments() []PaymentEvent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var failed []PaymentEvent
	for _, event := range r.events {
		if event.Type == PaymentEventFailure {
			failed = append(failed, event)
		}
	}
	return failed
}

// TotalAmount returns the total amount of all successful payments
func (r *PaymentRecorder) TotalAmount() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	total := big.NewInt(0)
	for _, event := range r.events {
		if event.Type == PaymentEventSuccess && event.Amount != nil {
			total.Add(total, event.Amount)
		}
	}
	return total.String()
}
