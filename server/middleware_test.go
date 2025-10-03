package server

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestPaymentMiddleware_FreeTool(t *testing.T) {
	config := &Config{
		PaymentTools: make(map[string][]PaymentRequirement),
	}

	called := false
	nextHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		called = true
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.NewTextContent("success")},
		}, nil
	}

	middleware := newPaymentMiddleware(config, nil)
	wrappedHandler := middleware(nextHandler)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "free-tool",
		},
	}

	result, err := wrappedHandler(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, called)
}

func TestPaymentMiddleware_PaidToolWithoutPayment(t *testing.T) {
	config := &Config{
		PaymentTools: map[string][]PaymentRequirement{
			"paid-tool": {
				{
					Scheme:            "exact",
					Network:           "base-sepolia",
					Asset:             "0x123",
					PayTo:             "0x456",
					MaxAmountRequired: "10000",
					MaxTimeoutSeconds: 60,
				},
			},
		},
	}

	nextHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		t.Error("Handler should not be called without payment")
		return nil, nil
	}

	middleware := newPaymentMiddleware(config, nil)
	wrappedHandler := middleware(nextHandler)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "paid-tool",
		},
	}

	result, err := wrappedHandler(context.Background(), req)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "Payment required")
}

func TestPaymentMiddleware_PaidToolWithValidPayment(t *testing.T) {
	mockFacilitator := &MockFacilitator{
		verifyResponse: &VerifyResponse{
			IsValid: true,
			Payer:   "0xabc",
		},
		settleResponse: &SettleResponse{
			Success:     true,
			Transaction: "0xtx123",
			Network:     "base-sepolia",
			Payer:       "0xabc",
		},
	}

	config := &Config{
		PaymentTools: map[string][]PaymentRequirement{
			"paid-tool": {
				{
					Scheme:            "exact",
					Network:           "base-sepolia",
					Asset:             "0x123",
					PayTo:             "0x456",
					MaxAmountRequired: "10000",
					MaxTimeoutSeconds: 60,
				},
			},
		},
		VerifyOnly: false,
	}

	handlerCalled := false
	nextHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		handlerCalled = true
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.NewTextContent("paid result")},
		}, nil
	}

	middleware := newPaymentMiddleware(config, mockFacilitator)
	wrappedHandler := middleware(nextHandler)

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "paid-tool",
			Meta: &mcp.Meta{
				AdditionalFields: map[string]any{
					"x402/payment": PaymentPayload{
						X402Version: 1,
						Scheme:      "exact",
						Network:     "base-sepolia",
						Payload: struct {
							Signature     string `json:"signature"`
							Authorization struct {
								From        string `json:"from"`
								To          string `json:"to"`
								Value       string `json:"value"`
								ValidAfter  string `json:"validAfter"`
								ValidBefore string `json:"validBefore"`
								Nonce       string `json:"nonce"`
							} `json:"authorization"`
						}{
							Signature: "0xsig",
							Authorization: struct {
								From        string `json:"from"`
								To          string `json:"to"`
								Value       string `json:"value"`
								ValidAfter  string `json:"validAfter"`
								ValidBefore string `json:"validBefore"`
								Nonce       string `json:"nonce"`
							}{
								From:        "0xabc",
								To:          "0x456",
								Value:       "10000",
								ValidAfter:  "1000000",
								ValidBefore: "2000000",
								Nonce:       "0x123",
							},
						},
					},
				},
			},
		},
	}

	result, err := wrappedHandler(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, handlerCalled)

	assert.NotNil(t, result.Meta)
	assert.NotNil(t, result.Meta.AdditionalFields)
	paymentResp := result.Meta.AdditionalFields["x402/payment-response"]
	assert.NotNil(t, paymentResp)
}
