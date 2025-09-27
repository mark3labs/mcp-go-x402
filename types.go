package x402

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
)

// PaymentRequirement represents a payment method from the server
type PaymentRequirement struct {
	Scheme            string            `json:"scheme"`
	Network           string            `json:"network"`
	MaxAmountRequired string            `json:"maxAmountRequired"`
	Asset             string            `json:"asset"`
	PayTo             string            `json:"payTo"`
	Resource          string            `json:"resource"`
	Description       string            `json:"description"`
	MimeType          string            `json:"mimeType,omitempty"`
	OutputSchema      interface{}       `json:"outputSchema,omitempty"`
	MaxTimeoutSeconds int               `json:"maxTimeoutSeconds"`
	Extra             map[string]string `json:"extra,omitempty"`
}

// PaymentRequirementsResponse is the 402 response body
type PaymentRequirementsResponse struct {
	X402Version int                  `json:"x402Version"`
	Error       string               `json:"error"`
	Accepts     []PaymentRequirement `json:"accepts"`
}

// PaymentPayload is the signed payment sent in X-PAYMENT header
type PaymentPayload struct {
	X402Version int                `json:"x402Version"`
	Scheme      string             `json:"scheme"`
	Network     string             `json:"network"`
	Payload     PaymentPayloadData `json:"payload"`
}

// PaymentPayloadData contains the signature and authorization
type PaymentPayloadData struct {
	Signature     string               `json:"signature"`
	Authorization PaymentAuthorization `json:"authorization"`
}

// PaymentAuthorization contains EIP-3009 authorization data
type PaymentAuthorization struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	ValidAfter  string `json:"validAfter"`
	ValidBefore string `json:"validBefore"`
	Nonce       string `json:"nonce"`
}

// Encode encodes the payment payload as base64 for the X-PAYMENT header
func (p *PaymentPayload) Encode() string {
	data, _ := json.Marshal(p)
	return base64.StdEncoding.EncodeToString(data)
}

// SettlementResponse represents the X-PAYMENT-RESPONSE header content
type SettlementResponse struct {
	Success     bool   `json:"success"`
	Transaction string `json:"transaction"`
	Network     string `json:"network"`
	Payer       string `json:"payer"`
	ErrorReason string `json:"errorReason,omitempty"`
}

// PaymentEvent represents a payment lifecycle event
type PaymentEvent struct {
	Type        PaymentEventType
	Resource    string
	Method      string
	Amount      *big.Int
	Network     string
	Asset       string
	Recipient   string
	Transaction string
	Error       error
	Timestamp   int64
}

// PaymentEventType represents types of payment events
type PaymentEventType string

const (
	PaymentEventAttempt PaymentEventType = "attempt"
	PaymentEventSuccess PaymentEventType = "success"
	PaymentEventFailure PaymentEventType = "failure"
)

// NetworkChainIDs maps network names to chain IDs
var NetworkChainIDs = map[string]*big.Int{
	"base-sepolia":   big.NewInt(84532),
	"base":           big.NewInt(8453),
	"avalanche-fuji": big.NewInt(43113),
	"avalanche":      big.NewInt(43114),
	"ethereum":       big.NewInt(1),
	"sepolia":        big.NewInt(11155111),
}

// GetChainID returns the chain ID for a network name
func GetChainID(network string) *big.Int {
	if chainID, ok := NetworkChainIDs[network]; ok {
		return chainID
	}
	return big.NewInt(1) // Default to mainnet
}

// ClientPaymentOption represents a payment method the client accepts
type ClientPaymentOption struct {
	PaymentRequirement

	// Client-specific fields
	Priority   int    `json:"-"` // Lower number = higher priority
	MaxAmount  string `json:"-"` // Client's max willing to pay with this option
	MinBalance string `json:"-"` // Don't use if balance falls below this
}
