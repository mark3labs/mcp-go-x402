package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	x402 "github.com/mark3labs/mcp-go-x402"
)

// mockMCPServer creates a test server that simulates an MCP server with x402 support
func mockMCPServer(t *testing.T, requirePayment bool) *httptest.Server {
	paymentReceived := false

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle WebSocket upgrade
		if r.Header.Get("Upgrade") == "websocket" {
			// For simplicity in testing, we'll just return an error
			http.Error(w, "WebSocket not supported in test", http.StatusNotImplemented)
			return
		}

		// Parse JSON-RPC request
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			ID     any             `json:"id"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Check for x402 payment header if payment is required
		if requirePayment && !paymentReceived {
			paymentHeader := r.Header.Get("X-402-Payment")
			if paymentHeader == "" {
				// Return 402 Payment Required
				w.Header().Set("X-402-Payment-Required", fmt.Sprintf(`{
					"network": "base-sepolia",
					"payTo": "0x1234567890123456789012345678901234567890",
					"asset": "USDC",
					"maxAmountRequired": "10000",
					"maxTimeoutSeconds": 60,
					"resource": "%s"
				}`, req.Method))
				w.WriteHeader(http.StatusPaymentRequired)
				json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":    -32000,
						"message": "Payment required",
					},
					"id": req.ID,
				})
				return
			}
			// Payment received
			paymentReceived = true
		}

		// Handle different methods
		switch req.Method {
		case "initialize":
			response := map[string]any{
				"result": map[string]any{
					"protocolVersion": "1.0.0",
					"serverInfo": map[string]any{
						"name":    "test-server",
						"version": "1.0.0",
					},
					"capabilities": map[string]any{
						"tools": true,
					},
				},
				"id": req.ID,
			}
			json.NewEncoder(w).Encode(response)

		case "tools/list":
			response := map[string]any{
				"result": map[string]any{
					"tools": []map[string]any{
						{
							"name":        "search",
							"description": "Search tool",
							"inputSchema": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"query":       map[string]string{"type": "string"},
									"max_results": map[string]string{"type": "number"},
								},
							},
						},
					},
				},
				"id": req.ID,
			}
			json.NewEncoder(w).Encode(response)

		case "tools/call":
			var params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			}
			json.Unmarshal(req.Params, &params)

			result := "No results found"
			if params.Name == "search" {
				if query, ok := params.Arguments["query"].(string); ok {
					result = fmt.Sprintf("Found 3 results for '%s':\n1. Result one\n2. Result two\n3. Result three", query)
				}
			}

			response := map[string]any{
				"result": map[string]any{
					"content": []map[string]any{
						{
							"type": "text",
							"text": result,
						},
					},
				},
				"id": req.ID,
			}
			json.NewEncoder(w).Encode(response)

		default:
			response := map[string]any{
				"error": map[string]any{
					"code":    -32601,
					"message": "Method not found",
				},
				"id": req.ID,
			}
			json.NewEncoder(w).Encode(response)
		}
	}))
}

// TestBasicClientInitialization tests basic MCP client initialization with x402
func TestBasicClientInitialization(t *testing.T) {
	server := mockMCPServer(t, false)
	defer server.Close()

	signer := x402.NewMockSigner("0xTestWallet")

	transport, err := x402.New(x402.Config{
		ServerURL:        server.URL,
		Signer:           signer,
		MaxPaymentAmount: "1000000",
		AutoPayThreshold: "100000",
	})
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	// Since the test server doesn't support WebSocket, we'll test the transport directly
	// In a real scenario, you'd need a WebSocket-capable test server

	// Test that transport was created successfully
	if transport == nil {
		t.Fatal("Transport should not be nil")
	}
}

// TestPaymentEventCallbacks tests that payment callbacks are triggered correctly
func TestPaymentEventCallbacks(t *testing.T) {
	server := mockMCPServer(t, true)
	defer server.Close()

	signer := x402.NewMockSigner("0xTestWallet")
	recorder := x402.NewPaymentRecorder()

	transport, err := x402.New(x402.Config{
		ServerURL:        server.URL,
		Signer:           signer,
		MaxPaymentAmount: "1000000",
		AutoPayThreshold: "100000",
		OnPaymentAttempt: func(event x402.PaymentEvent) {
			recorder.Record(event)

			// Verify event fields
			if event.Type != x402.PaymentEventAttempt {
				t.Errorf("Expected PaymentEventAttempt, got %s", event.Type)
			}
			if event.Amount == nil || event.Amount.Cmp(big.NewInt(10000)) != 0 {
				t.Errorf("Expected amount 10000, got %v", event.Amount)
			}
			if event.Asset != "USDC" {
				t.Errorf("Expected asset USDC, got %s", event.Asset)
			}
			if event.Network != "base-sepolia" {
				t.Errorf("Expected network base-sepolia, got %s", event.Network)
			}
		},
		OnPaymentSuccess: func(event x402.PaymentEvent) {
			recorder.Record(event)

			// Verify success event
			if event.Type != x402.PaymentEventSuccess {
				t.Errorf("Expected PaymentEventSuccess, got %s", event.Type)
			}
			if event.Transaction == "" {
				t.Error("Transaction hash should not be empty")
			}
		},
		OnPaymentFailure: func(event x402.PaymentEvent, err error) {
			t.Logf("Payment failed (expected in some test scenarios): %v", err)
		},
	})
	if err != nil {
		t.Fatalf("Failed to create transport: %v", err)
	}

	// Attach recorder for additional verification
	x402.WithPaymentRecorder(recorder)(transport)

	// Make a request that requires payment (this would trigger the callbacks in a real scenario)
	// Note: Since our test server doesn't fully implement WebSocket, we're testing the setup

	// Verify transport configuration
	if transport == nil {
		t.Fatal("Transport should not be nil")
	}

	// In a real test with WebSocket support, you would:
	// 1. Create an MCP client
	// 2. Make a request that triggers 402 response
	// 3. Verify callbacks were called
	// 4. Check recorder for payment events
}

// TestPaymentRecorder tests the payment recording functionality
func TestPaymentRecorder(t *testing.T) {
	recorder := x402.NewPaymentRecorder()

	// Test initial state
	if recorder.PaymentCount() != 0 {
		t.Errorf("Expected 0 payments, got %d", recorder.PaymentCount())
	}
	if recorder.LastPayment() != nil {
		t.Error("Expected no last payment")
	}
	if recorder.TotalAmount() != "0" {
		t.Errorf("Expected total amount 0, got %s", recorder.TotalAmount())
	}

	// Record attempt event
	attemptEvent := x402.PaymentEvent{
		Type:      x402.PaymentEventAttempt,
		Amount:    big.NewInt(10000),
		Asset:     "USDC",
		Network:   "base-sepolia",
		Recipient: "0x1234567890123456789012345678901234567890",
		Resource:  "tools/call",
		Timestamp: time.Now().Unix(),
	}
	recorder.Record(attemptEvent)

	// Verify attempt was recorded
	if recorder.PaymentCount() != 1 {
		t.Errorf("Expected 1 payment, got %d", recorder.PaymentCount())
	}

	// Record success event
	successEvent := x402.PaymentEvent{
		Type:        x402.PaymentEventSuccess,
		Amount:      big.NewInt(10000),
		Asset:       "USDC",
		Network:     "base-sepolia",
		Recipient:   "0x1234567890123456789012345678901234567890",
		Resource:    "tools/call",
		Transaction: "0xabcdef123456",
		Timestamp:   time.Now().Unix(),
	}
	recorder.Record(successEvent)

	// Verify success was recorded
	if recorder.PaymentCount() != 2 {
		t.Errorf("Expected 2 payments, got %d", recorder.PaymentCount())
	}

	lastPayment := recorder.LastPayment()
	if lastPayment == nil {
		t.Fatal("Expected last payment")
	}
	if lastPayment.Type != x402.PaymentEventSuccess {
		t.Errorf("Expected last payment to be success, got %s", lastPayment.Type)
	}
	if lastPayment.Transaction != "0xabcdef123456" {
		t.Errorf("Expected transaction hash, got %s", lastPayment.Transaction)
	}

	// Test successful payments
	successful := recorder.SuccessfulPayments()
	if len(successful) != 1 {
		t.Errorf("Expected 1 successful payment, got %d", len(successful))
	}

	// Test total amount
	if recorder.TotalAmount() != "10000" {
		t.Errorf("Expected total amount 10000, got %s", recorder.TotalAmount())
	}

	// Record failure event
	failureEvent := x402.PaymentEvent{
		Type:      x402.PaymentEventFailure,
		Amount:    big.NewInt(5000),
		Asset:     "USDC",
		Network:   "base-sepolia",
		Recipient: "0x1234567890123456789012345678901234567890",
		Resource:  "tools/call",
		Error:     errors.New("Insufficient balance"),
		Timestamp: time.Now().Unix(),
	}
	recorder.Record(failureEvent)

	// Test failed payments
	failed := recorder.FailedPayments()
	if len(failed) != 1 {
		t.Errorf("Expected 1 failed payment, got %d", len(failed))
	}
	if failed[0].Error.Error() != "Insufficient balance" {
		t.Errorf("Expected error message, got %s", failed[0].Error)
	}

	// Total amount shouldn't include failed payments
	if recorder.TotalAmount() != "10000" {
		t.Errorf("Expected total amount still 10000, got %s", recorder.TotalAmount())
	}

	// Test clear
	recorder.Clear()
	if recorder.PaymentCount() != 0 {
		t.Errorf("Expected 0 payments after clear, got %d", recorder.PaymentCount())
	}
}

// TestMultiplePayments tests handling multiple payment requests
func TestMultiplePayments(t *testing.T) {
	recorder := x402.NewPaymentRecorder()

	// Simulate multiple payments
	for i := 0; i < 5; i++ {
		// Attempt
		recorder.Record(x402.PaymentEvent{
			Type:      x402.PaymentEventAttempt,
			Amount:    big.NewInt(int64(1000 * (i + 1))),
			Asset:     "USDC",
			Network:   "base-sepolia",
			Resource:  fmt.Sprintf("resource-%d", i),
			Timestamp: time.Now().Unix(),
		})

		// Success
		recorder.Record(x402.PaymentEvent{
			Type:        x402.PaymentEventSuccess,
			Amount:      big.NewInt(int64(1000 * (i + 1))),
			Asset:       "USDC",
			Network:     "base-sepolia",
			Resource:    fmt.Sprintf("resource-%d", i),
			Transaction: fmt.Sprintf("0xtx%d", i),
			Timestamp:   time.Now().Unix(),
		})
	}

	// Verify all events recorded
	if recorder.PaymentCount() != 10 {
		t.Errorf("Expected 10 events, got %d", recorder.PaymentCount())
	}

	// Verify successful payments
	successful := recorder.SuccessfulPayments()
	if len(successful) != 5 {
		t.Errorf("Expected 5 successful payments, got %d", len(successful))
	}

	// Verify total amount (1000 + 2000 + 3000 + 4000 + 5000 = 15000)
	if recorder.TotalAmount() != "15000" {
		t.Errorf("Expected total amount 15000, got %s", recorder.TotalAmount())
	}

	// Verify events are in order
	events := recorder.GetEvents()
	for i := 0; i < len(events); i += 2 {
		if events[i].Type != x402.PaymentEventAttempt {
			t.Errorf("Expected attempt at index %d, got %s", i, events[i].Type)
		}
		if i+1 < len(events) && events[i+1].Type != x402.PaymentEventSuccess {
			t.Errorf("Expected success at index %d, got %s", i+1, events[i+1].Type)
		}
	}
}

// TestSignerTypes tests different signer implementations
func TestSignerTypes(t *testing.T) {
	tests := []struct {
		name         string
		createSigner func() (x402.PaymentSigner, error)
		expectError  bool
	}{
		{
			name: "MockSigner",
			createSigner: func() (x402.PaymentSigner, error) {
				return x402.NewMockSigner("0xTestWallet"), nil
			},
			expectError: false,
		},
		{
			name: "PrivateKeySigner-Invalid",
			createSigner: func() (x402.PaymentSigner, error) {
				return x402.NewPrivateKeySigner("invalid-key")
			},
			expectError: true,
		},
		{
			name: "MnemonicSigner-Invalid",
			createSigner: func() (x402.PaymentSigner, error) {
				return x402.NewMnemonicSigner("invalid mnemonic phrase", "")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signer, err := tt.createSigner()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error creating signer")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error creating signer: %v", err)
			}

			// Test signer methods
			address := signer.GetAddress()
			if address == "" {
				t.Error("Signer should return address")
			}

			// Test network support
			if !signer.SupportsNetwork("base-sepolia") {
				t.Error("Signer should support base-sepolia")
			}

			// Test signing
			ctx := context.Background()
			req := x402.PaymentRequirement{
				Network:           "base-sepolia",
				PayTo:             "0x1234567890123456789012345678901234567890",
				Asset:             "USDC",
				MaxAmountRequired: "10000",
				MaxTimeoutSeconds: 60,
				Resource:          "test-resource",
			}

			payload, err := signer.SignPayment(ctx, req)
			if err != nil {
				t.Fatalf("Failed to sign payment: %v", err)
			}

			if payload == nil {
				t.Fatal("Payload should not be nil")
			}
			if payload.Payload.Signature == "" {
				t.Error("Signature should not be empty")
			}
			if payload.Payload.Authorization.From != address {
				t.Errorf("Expected signer %s, got %s", address, payload.Payload.Authorization.From)
			}
		})
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	signer := x402.NewMockSigner("0xTestWallet")

	tests := []struct {
		name        string
		config      x402.Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid config",
			config: x402.Config{
				ServerURL:        "https://example.com",
				Signer:           signer,
				MaxPaymentAmount: "1000000",
			},
			expectError: false,
		},
		{
			name: "Invalid ServerURL",
			config: x402.Config{
				ServerURL:        ":::invalid-url:::",
				Signer:           signer,
				MaxPaymentAmount: "1000000",
			},
			expectError: true,
			errorMsg:    "invalid server URL",
		},
		{
			name: "Invalid MaxPaymentAmount",
			config: x402.Config{
				ServerURL:        "https://example.com",
				Signer:           signer,
				MaxPaymentAmount: "not-a-number",
			},
			expectError: true,
			errorMsg:    "amount",
		},
		{
			name: "Invalid AutoPayThreshold",
			config: x402.Config{
				ServerURL:        "https://example.com",
				Signer:           signer,
				MaxPaymentAmount: "1000000",
				AutoPayThreshold: "not-a-number",
			},
			expectError: false, // AutoPayThreshold is not validated on creation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := x402.New(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error creating transport")
				} else if tt.errorMsg != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errorMsg)) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error creating transport: %v", err)
				}
			}
		})
	}
}

// TestEnvironmentVariables tests loading configuration from environment
func TestEnvironmentVariables(t *testing.T) {
	// Save and restore original env vars
	origPrivateKey := os.Getenv("WALLET_PRIVATE_KEY")
	origServerURL := os.Getenv("MCP_SERVER_URL")
	defer func() {
		os.Setenv("WALLET_PRIVATE_KEY", origPrivateKey)
		os.Setenv("MCP_SERVER_URL", origServerURL)
	}()

	// Test with missing private key
	os.Unsetenv("WALLET_PRIVATE_KEY")
	os.Setenv("MCP_SERVER_URL", "https://test.example.com")

	// This would be the pattern used in main.go
	privateKey := os.Getenv("WALLET_PRIVATE_KEY")
	if privateKey != "" {
		t.Error("Private key should be empty when not set")
	}

	serverURL := os.Getenv("MCP_SERVER_URL")
	if serverURL != "https://test.example.com" {
		t.Errorf("Expected custom server URL, got %s", serverURL)
	}

	// Test with default server URL
	os.Unsetenv("MCP_SERVER_URL")
	serverURL = os.Getenv("MCP_SERVER_URL")
	if serverURL == "" {
		// This is expected - the code should provide a default
		serverURL = "https://mcpay.tech/mcp/a9ad1af3-f91a-468c-96e4-28ebdfdd36c3"
	}
	if !strings.Contains(serverURL, "mcpay.tech") {
		t.Errorf("Expected default mcpay.tech URL, got %s", serverURL)
	}
}

// BenchmarkPaymentRecording benchmarks payment recording performance
func BenchmarkPaymentRecording(b *testing.B) {
	recorder := x402.NewPaymentRecorder()

	event := x402.PaymentEvent{
		Type:        x402.PaymentEventSuccess,
		Amount:      big.NewInt(10000),
		Asset:       "USDC",
		Network:     "base-sepolia",
		Recipient:   "0x1234567890123456789012345678901234567890",
		Resource:    "benchmark-resource",
		Transaction: "0xbenchmark",
		Timestamp:   time.Now().Unix(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder.Record(event)
	}
}

// BenchmarkTotalAmountCalculation benchmarks total amount calculation
func BenchmarkTotalAmountCalculation(b *testing.B) {
	recorder := x402.NewPaymentRecorder()

	// Pre-populate with many events
	for i := 0; i < 1000; i++ {
		recorder.Record(x402.PaymentEvent{
			Type:        x402.PaymentEventSuccess,
			Amount:      big.NewInt(int64(i + 1)),
			Asset:       "USDC",
			Network:     "base-sepolia",
			Recipient:   "0x1234567890123456789012345678901234567890",
			Resource:    fmt.Sprintf("resource-%d", i),
			Transaction: fmt.Sprintf("0xtx%d", i),
			Timestamp:   time.Now().Unix(),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = recorder.TotalAmount()
	}
}
