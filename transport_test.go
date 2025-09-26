package x402

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestX402Transport_Basic(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for payment header
		if r.Header.Get("X-PAYMENT") == "" {
			// Return 402 with payment requirements
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			json.NewEncoder(w).Encode(PaymentRequirementsResponse{
				X402Version: 1,
				Error:       "Payment required",
				Accepts: []PaymentRequirement{
					{
						Scheme:            "exact",
						Network:           "base-sepolia",
						MaxAmountRequired: "1000",
						Asset:             "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
						PayTo:             "0x209693Bc6afc0C5328bA36FaF03C514EF312287C",
						Resource:          r.URL.String(),
						Description:       "Test payment",
						MaxTimeoutSeconds: 60,
						Extra: map[string]string{
							"name":    "USDC",
							"version": "2",
						},
					},
				},
			})
			return
		}

		// Payment provided, return success
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-PAYMENT-RESPONSE", `{"success":true,"transaction":"0x123"}`)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(transport.JSONRPCResponse{
			ID:     mcp.NewRequestId(1),
			Result: json.RawMessage(`{"data":"test"}`),
		})
	}))
	defer server.Close()

	// Create transport with mock signer
	signer := NewMockSigner("0xTestWallet")
	recorder := NewPaymentRecorder()

	trans, err := New(Config{
		ServerURL:        server.URL,
		Signer:           signer,
		MaxPaymentAmount: "10000",
		AutoPayThreshold: "5000",
	})
	require.NoError(t, err)

	trans.paymentRecorder = recorder

	ctx := context.Background()
	err = trans.Start(ctx)
	require.NoError(t, err)
	defer trans.Close()

	// Send request
	request := transport.JSONRPCRequest{
		ID:     mcp.NewRequestId(1),
		Method: "test.method",
		Params: json.RawMessage(`{}`),
	}

	response, err := trans.SendRequest(ctx, request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.False(t, response.ID.IsNil())

	// Check payment was made
	assert.Equal(t, 2, recorder.PaymentCount()) // Attempt + Success
	lastPayment := recorder.LastPayment()
	assert.Equal(t, PaymentEventSuccess, lastPayment.Type)
	assert.Equal(t, "1000", lastPayment.Amount.String())
}

func TestX402Transport_ExceedsLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		json.NewEncoder(w).Encode(PaymentRequirementsResponse{
			X402Version: 1,
			Error:       "Payment required",
			Accepts: []PaymentRequirement{
				{
					Scheme:            "exact",
					Network:           "base-sepolia",
					MaxAmountRequired: "1000000", // Exceeds limit
					Asset:             "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
					PayTo:             "0x209693Bc6afc0C5328bA36FaF03C514EF312287C",
					Resource:          r.URL.String(),
					Description:       "Expensive payment",
					MaxTimeoutSeconds: 60,
					Extra: map[string]string{
						"name":    "USDC",
						"version": "2",
					},
				},
			},
		})
	}))
	defer server.Close()

	signer := NewMockSigner("0xTestWallet")

	trans, err := New(Config{
		ServerURL:        server.URL,
		Signer:           signer,
		MaxPaymentAmount: "10000", // Limit is 10000
		AutoPayThreshold: "5000",
	})
	require.NoError(t, err)

	ctx := context.Background()
	request := transport.JSONRPCRequest{
		ID:     mcp.NewRequestId(1),
		Method: "test.method",
		Params: json.RawMessage(`{}`),
	}

	_, err = trans.SendRequest(ctx, request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestX402Transport_RateLimit(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if r.Header.Get("X-PAYMENT") == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			json.NewEncoder(w).Encode(PaymentRequirementsResponse{
				X402Version: 1,
				Error:       "Payment required",
				Accepts: []PaymentRequirement{
					{
						Scheme:            "exact",
						Network:           "base-sepolia",
						MaxAmountRequired: "100",
						Asset:             "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
						PayTo:             "0x209693Bc6afc0C5328bA36FaF03C514EF312287C",
						Resource:          r.URL.String(),
						Description:       "Test",
						MaxTimeoutSeconds: 60,
						Extra: map[string]string{
							"name":    "USDC",
							"version": "2",
						},
					},
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(transport.JSONRPCResponse{
			ID:     mcp.NewRequestId(requestCount),
			Result: json.RawMessage(`{"data":"test"}`),
		})
	}))
	defer server.Close()

	signer := NewMockSigner("0xTestWallet")

	trans, err := New(Config{
		ServerURL:        server.URL,
		Signer:           signer,
		MaxPaymentAmount: "10000",
		RateLimits: &RateLimits{
			MaxPaymentsPerMinute: 2,
		},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// First two requests should succeed
	for i := 0; i < 2; i++ {
		request := transport.JSONRPCRequest{
			ID:     mcp.NewRequestId(i + 1),
			Method: "test.method",
			Params: json.RawMessage(`{}`),
		}

		_, err := trans.SendRequest(ctx, request)
		assert.NoError(t, err)
	}

	// Third request should fail due to rate limit
	request := transport.JSONRPCRequest{
		ID:     mcp.NewRequestId(3),
		Method: "test.method",
		Params: json.RawMessage(`{}`),
	}

	_, err = trans.SendRequest(ctx, request)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit")
}

func TestX402Transport_PaymentCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-PAYMENT") == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			json.NewEncoder(w).Encode(PaymentRequirementsResponse{
				X402Version: 1,
				Error:       "Payment required",
				Accepts: []PaymentRequirement{
					{
						Scheme:            "exact",
						Network:           "base-sepolia",
						MaxAmountRequired: "10000", // Above auto-pay threshold
						Asset:             "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
						PayTo:             "0x209693Bc6afc0C5328bA36FaF03C514EF312287C",
						Resource:          r.URL.String(),
						Description:       "Test",
						MaxTimeoutSeconds: 60,
						Extra: map[string]string{
							"name":    "USDC",
							"version": "2",
						},
					},
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(transport.JSONRPCResponse{
			ID:     mcp.NewRequestId(1),
			Result: json.RawMessage(`{"data":"test"}`),
		})
	}))
	defer server.Close()

	signer := NewMockSigner("0xTestWallet")
	callbackCalled := false

	trans, err := New(Config{
		ServerURL:        server.URL,
		Signer:           signer,
		MaxPaymentAmount: "100000",
		AutoPayThreshold: "5000", // Auto-pay below 5000
		PaymentCallback: func(amount *big.Int, resource string) bool {
			callbackCalled = true
			assert.Equal(t, "10000", amount.String())
			return true // Approve payment
		},
	})
	require.NoError(t, err)

	ctx := context.Background()
	request := transport.JSONRPCRequest{
		ID:     mcp.NewRequestId(1),
		Method: "test.method",
		Params: json.RawMessage(`{}`),
	}

	_, err = trans.SendRequest(ctx, request)
	assert.NoError(t, err)
	assert.True(t, callbackCalled, "Payment callback should have been called")
}
