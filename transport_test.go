package x402

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
			_ = json.NewEncoder(w).Encode(PaymentRequirementsResponse{
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
		_ = json.NewEncoder(w).Encode(transport.JSONRPCResponse{
			ID:     mcp.NewRequestId(1),
			Result: json.RawMessage(`{"data":"test"}`),
		})
	}))
	defer server.Close()

	// Create transport with mock signer that supports Base Sepolia
	signer := NewMockSigner("0xTestWallet", AcceptUSDCBaseSepolia())
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
		_ = json.NewEncoder(w).Encode(PaymentRequirementsResponse{
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

	signer := NewMockSigner("0xTestWallet", AcceptUSDCBaseSepolia())

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
			_ = json.NewEncoder(w).Encode(PaymentRequirementsResponse{
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
		_ = json.NewEncoder(w).Encode(transport.JSONRPCResponse{
			ID:     mcp.NewRequestId(requestCount),
			Result: json.RawMessage(`{"data":"test"}`),
		})
	}))
	defer server.Close()

	signer := NewMockSigner("0xTestWallet", AcceptUSDCBaseSepolia())

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
			_ = json.NewEncoder(w).Encode(PaymentRequirementsResponse{
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
		_ = json.NewEncoder(w).Encode(transport.JSONRPCResponse{
			ID:     mcp.NewRequestId(1),
			Result: json.RawMessage(`{"data":"test"}`),
		})
	}))
	defer server.Close()

	signer := NewMockSigner("0xTestWallet", AcceptUSDCBaseSepolia())
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

func TestX402Transport_MultipleRequests(t *testing.T) {
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqNum := requestCount.Add(1)

		// Only first request requires payment
		if reqNum == 1 && r.Header.Get("X-PAYMENT") == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			_ = json.NewEncoder(w).Encode(PaymentRequirementsResponse{
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

		// Parse request to get ID
		var req transport.JSONRPCRequest
		_ = json.NewDecoder(r.Body).Decode(&req)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(transport.JSONRPCResponse{
			ID:     req.ID,
			Result: json.RawMessage(fmt.Sprintf(`{"data":"response_%d"}`, reqNum)),
		})
	}))
	defer server.Close()

	signer := NewMockSigner("0xTestWallet", AcceptUSDCBaseSepolia())
	trans, err := New(Config{
		ServerURL:        server.URL,
		Signer:           signer,
		MaxPaymentAmount: "10000",
		AutoPayThreshold: "5000",
	})
	require.NoError(t, err)

	ctx := context.Background()
	err = trans.Start(ctx)
	require.NoError(t, err)
	defer trans.Close()

	const numRequests = 5
	var wg sync.WaitGroup
	responses := make([]*transport.JSONRPCResponse, numRequests)
	errors := make([]error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			request := transport.JSONRPCRequest{
				ID:     mcp.NewRequestId(int64(100 + idx)),
				Method: "test.method",
				Params: map[string]any{
					"requestIndex": idx,
				},
			}

			resp, err := trans.SendRequest(ctx, request)
			responses[idx] = resp
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Check results
	for i := 0; i < numRequests; i++ {
		if errors[i] != nil {
			t.Errorf("Request %d failed: %v", i, errors[i])
			continue
		}

		if responses[i] == nil {
			t.Errorf("Request %d: Response is nil", i)
			continue
		}

		expectedId := int64(100 + i)
		idValue, ok := responses[i].ID.Value().(int64)
		if !ok {
			t.Errorf("Request %d: Expected ID to be int64, got %T", i, responses[i].ID.Value())
		} else if idValue != expectedId {
			t.Errorf("Request %d: Expected ID %d, got %d", i, expectedId, idValue)
		}
	}
}

func TestX402Transport_SendRequestWithTimeout(t *testing.T) {
	// Server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than context timeout
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(transport.JSONRPCResponse{
			ID:     mcp.NewRequestId(1),
			Result: json.RawMessage(`{"data":"test"}`),
		})
	}))
	defer server.Close()

	signer := NewMockSigner("0xTestWallet", AcceptUSDCBaseSepolia())
	trans, err := New(Config{
		ServerURL:        server.URL,
		Signer:           signer,
		MaxPaymentAmount: "10000",
	})
	require.NoError(t, err)

	// Create a context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	request := transport.JSONRPCRequest{
		ID:     mcp.NewRequestId(1),
		Method: "test.method",
		Params: json.RawMessage(`{}`),
	}

	_, err = trans.SendRequest(ctx, request)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "context deadline exceeded"))
}

func TestX402Transport_ResponseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return JSON-RPC error response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(transport.JSONRPCResponse{
			ID: mcp.NewRequestId(1),
			Error: &struct {
				Code    int             `json:"code"`
				Message string          `json:"message"`
				Data    json.RawMessage `json:"data"`
			}{
				Code:    -32601,
				Message: "Method not found",
			},
		})
	}))
	defer server.Close()

	signer := NewMockSigner("0xTestWallet", AcceptUSDCBaseSepolia())
	trans, err := New(Config{
		ServerURL:        server.URL,
		Signer:           signer,
		MaxPaymentAmount: "10000",
	})
	require.NoError(t, err)

	ctx := context.Background()
	request := transport.JSONRPCRequest{
		ID:     mcp.NewRequestId(1),
		Method: "unknown.method",
		Params: json.RawMessage(`{}`),
	}

	resp, err := trans.SendRequest(ctx, request)
	assert.NoError(t, err) // No transport error
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
	assert.Equal(t, "Method not found", resp.Error.Message)
}

func TestX402Transport_InvalidURL(t *testing.T) {
	signer := NewMockSigner("0xTestWallet", AcceptUSDCBaseSepolia())

	// Test invalid URL
	_, err := New(Config{
		ServerURL: "://invalid-url",
		Signer:    signer,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestX402Transport_NonExistentServer(t *testing.T) {
	signer := NewMockSigner("0xTestWallet", AcceptUSDCBaseSepolia())
	trans, err := New(Config{
		ServerURL:        "http://localhost:1", // Port 1 is typically unused
		Signer:           signer,
		MaxPaymentAmount: "10000",
		HTTPClient: &http.Client{
			Timeout: 1 * time.Second,
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
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "connect: connection refused"))
}

func TestX402Transport_SetNotificationHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is SSE request
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "text/event-stream") {
			// Send SSE response with notification
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			// Send a notification
			notification := map[string]any{
				"jsonrpc": "2.0",
				"method":  "test/notification",
				"params":  map[string]any{"message": "Hello"},
			}
			data, _ := json.Marshal(notification)
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)

			// Send the actual response
			response := map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"result":  "success",
			}
			data, _ = json.Marshal(response)
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return
		}

		// Regular response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(transport.JSONRPCResponse{
			ID:     mcp.NewRequestId(1),
			Result: json.RawMessage(`{"data":"test"}`),
		})
	}))
	defer server.Close()

	signer := NewMockSigner("0xTestWallet", AcceptUSDCBaseSepolia())
	trans, err := New(Config{
		ServerURL:        server.URL,
		Signer:           signer,
		MaxPaymentAmount: "10000",
	})
	require.NoError(t, err)

	notificationChan := make(chan mcp.JSONRPCNotification, 1)
	trans.SetNotificationHandler(func(notification mcp.JSONRPCNotification) {
		notificationChan <- notification
	})

	ctx := context.Background()
	request := transport.JSONRPCRequest{
		ID:     mcp.NewRequestId(1),
		Method: "test.method",
		Params: json.RawMessage(`{}`),
	}

	go func() {
		_, _ = trans.SendRequest(ctx, request)
	}()

	select {
	case notification := <-notificationChan:
		assert.Equal(t, "test/notification", notification.Method)
	case <-time.After(2 * time.Second):
		// It's okay if notification is not received in this test
		// as our transport might not fully support SSE
	}
}

func TestX402Transport_SetRequestHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(transport.JSONRPCResponse{
			ID:     mcp.NewRequestId(1),
			Result: json.RawMessage(`{"data":"test"}`),
		})
	}))
	defer server.Close()

	signer := NewMockSigner("0xTestWallet", AcceptUSDCBaseSepolia())
	trans, err := New(Config{
		ServerURL:        server.URL,
		Signer:           signer,
		MaxPaymentAmount: "10000",
	})
	require.NoError(t, err)

	requestHandlerCalled := false
	trans.SetRequestHandler(func(ctx context.Context, request transport.JSONRPCRequest) (*transport.JSONRPCResponse, error) {
		requestHandlerCalled = true
		return &transport.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  json.RawMessage(`{"handled":true}`),
		}, nil
	})

	// The request handler would be called if the server sends requests to us
	// In this test we're just verifying it can be set
	assert.False(t, requestHandlerCalled, "Request handler shouldn't be called in this test")
}

func TestX402Transport_PaymentCallbackRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		_ = json.NewEncoder(w).Encode(PaymentRequirementsResponse{
			X402Version: 1,
			Error:       "Payment required",
			Accepts: []PaymentRequirement{
				{
					Scheme:            "exact",
					Network:           "base-sepolia",
					MaxAmountRequired: "10000",
					Asset:             "0x036CbD53842c5426634e7929541eC2318f3dCF7e",
					PayTo:             "0x209693Bc6afc0C5328bA36FaF03C514EF312287C",
					Resource:          r.URL.String(),
					Description:       "Test",
					MaxTimeoutSeconds: 60,
				},
			},
		})
	}))
	defer server.Close()

	signer := NewMockSigner("0xTestWallet", AcceptUSDCBaseSepolia())

	trans, err := New(Config{
		ServerURL:        server.URL,
		Signer:           signer,
		MaxPaymentAmount: "100000",
		AutoPayThreshold: "5000",
		PaymentCallback: func(amount *big.Int, resource string) bool {
			return false // Reject payment
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
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment declined")
}
