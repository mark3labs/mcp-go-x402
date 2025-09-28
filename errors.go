package x402

import (
	"errors"
	"fmt"
)

var (
	// Payment errors
	ErrPaymentRequired     = errors.New("payment required")
	ErrNoAcceptablePayment = errors.New("no acceptable payment method found")
	ErrSigningFailed       = errors.New("failed to sign payment")
	ErrInvalidPaymentReqs  = errors.New("invalid payment requirements")

	// Network errors
	ErrUnsupportedNetwork = errors.New("unsupported network")
	ErrUnsupportedAsset   = errors.New("unsupported asset")

	// Signer errors
	ErrInvalidPrivateKey = errors.New("invalid private key")
	ErrInvalidMnemonic   = errors.New("invalid mnemonic phrase")
	ErrInvalidKeystore   = errors.New("invalid keystore file")
	ErrWrongPassword     = errors.New("wrong keystore password")
)

// PaymentError provides detailed payment error information
type PaymentError struct {
	Code     string
	Message  string
	Resource string
	Amount   string
	Network  string
	Wrapped  error
}

func (e *PaymentError) Error() string {
	if e.Wrapped != nil {
		return fmt.Sprintf("%s: %s (resource: %s, amount: %s, network: %s): %v",
			e.Code, e.Message, e.Resource, e.Amount, e.Network, e.Wrapped)
	}
	return fmt.Sprintf("%s: %s (resource: %s, amount: %s, network: %s)",
		e.Code, e.Message, e.Resource, e.Amount, e.Network)
}

func (e *PaymentError) Unwrap() error {
	return e.Wrapped
}

// NewPaymentError creates a new PaymentError
func NewPaymentError(code, message, resource, amount, network string, wrapped error) *PaymentError {
	return &PaymentError{
		Code:     code,
		Message:  message,
		Resource: resource,
		Amount:   amount,
		Network:  network,
		Wrapped:  wrapped,
	}
}
