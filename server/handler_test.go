package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockMCPHandler simulates an MCP handler
type mockMCPHandler struct {
	called   bool
	response string
}

func (m *mockMCPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.called = true
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(m.response))
}

func TestX402Handler_NoPaymentRequired(t *testing.T) {
	mockHandler := &mockMCPHandler{
		response: `{"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"success"}]},"id":1}`,
	}

	config := &Config{
		FacilitatorURL: "http://mock",
		PaymentTools:   make(map[string][]PaymentRequirement),
	}

	handler := NewX402Handler(mockHandler, config)

	// Request for non-paid tool
	reqBody := `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"free-tool"},"id":1}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}

	if !mockHandler.called {
		t.Error("MCP handler should have been called")
	}
}

func TestX402Handler_PaymentRequired(t *testing.T) {
	mockHandler := &mockMCPHandler{
		response: `{"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"success"}]},"id":1}`,
	}

	config := &Config{
		FacilitatorURL: "http://mock",
		PaymentTools: map[string][]PaymentRequirement{
			"paid-tool": {
				{
					Scheme:            "exact",
					Network:           "test",
					MaxAmountRequired: "1000",
					Asset:             "0xusdc",
					PayTo:             "0xrecipient",
					MaxTimeoutSeconds: 60,
				},
			},
		},
	}

	handler := NewX402Handler(mockHandler, config)

	// Request for paid tool without payment
	reqBody := `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"paid-tool"},"id":1}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Errorf("Expected 402, got %d", rr.Code)
	}

	// Check response
	var resp PaymentRequirements402Response
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if len(resp.Accepts) != 1 {
		t.Error("Expected payment requirements")
	}

	if resp.Accepts[0].MaxAmountRequired != "1000" {
		t.Errorf("Wrong amount: %s", resp.Accepts[0].MaxAmountRequired)
	}

	if !mockHandler.called {
		// Correct - handler should NOT be called without payment
	} else {
		t.Error("MCP handler should NOT have been called without payment")
	}
}

func TestX402Handler_WithValidPayment(t *testing.T) {
	mockHandler := &mockMCPHandler{
		response: `{"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"success"}]},"id":1}`,
	}

	// Mock facilitator
	mockFacilitator := &MockFacilitator{
		verifyResponse: &VerifyResponse{
			IsValid: true,
			Payer:   "0xpayer",
		},
		settleResponse: &SettleResponse{
			Success:     true,
			Transaction: "0xtx",
			Network:     "test",
		},
	}

	config := &Config{
		FacilitatorURL: "http://mock",
		PaymentTools: map[string][]PaymentRequirement{
			"paid-tool": {
				{
					Scheme:            "exact",
					Network:           "test",
					MaxAmountRequired: "1000",
					Asset:             "0xusdc",
					PayTo:             "0xrecipient",
					MaxTimeoutSeconds: 60,
				},
			},
		},
	}

	handler := NewX402Handler(mockHandler, config)
	handler.facilitator = mockFacilitator

	// Create payment
	payment := &PaymentPayload{
		X402Version: 1,
		Scheme:      "exact",
		Network:     "test",
	}
	payment.Payload.Signature = "0xsig"
	payment.Payload.Authorization.From = "0xpayer"
	payment.Payload.Authorization.To = "0xusdc" // Asset address in EIP-3009
	payment.Payload.Authorization.Value = "1000"

	paymentJSON, _ := json.Marshal(payment)
	paymentHeader := base64.StdEncoding.EncodeToString(paymentJSON)

	// Request with payment
	reqBody := `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"paid-tool"},"id":1}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PAYMENT", paymentHeader)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		body, _ := io.ReadAll(rr.Body)
		t.Errorf("Expected 200, got %d. Body: %s", rr.Code, string(body))
	}

	if !mockHandler.called {
		t.Error("MCP handler should have been called with valid payment")
	}

	// Check for payment response header
	if rr.Header().Get("X-PAYMENT-RESPONSE") == "" {
		t.Error("Expected X-PAYMENT-RESPONSE header")
	}
}

// MockFacilitator for testing
type MockFacilitator struct {
	verifyResponse *VerifyResponse
	settleResponse *SettleResponse
	verifyCalled   bool
	settleCalled   bool
}

func (m *MockFacilitator) Verify(ctx context.Context, payment *PaymentPayload, requirement *PaymentRequirement) (*VerifyResponse, error) {
	m.verifyCalled = true
	return m.verifyResponse, nil
}

func (m *MockFacilitator) Settle(ctx context.Context, payment *PaymentPayload, requirement *PaymentRequirement) (*SettleResponse, error) {
	m.settleCalled = true
	return m.settleResponse, nil
}

func (m *MockFacilitator) GetSupported(ctx context.Context) ([]SupportedKind, error) {
	return []SupportedKind{{X402Version: 1, Scheme: "exact", Network: "test"}}, nil
}

func TestX402Handler_MultiplePaymentOptions(t *testing.T) {
	mockHandler := &mockMCPHandler{
		response: `{"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"success"}]},"id":1}`,
	}

	config := &Config{
		FacilitatorURL: "http://mock",
		PaymentTools: map[string][]PaymentRequirement{
			"multi-pay-tool": {
				{
					Scheme:            "eip3009",
					Network:           "ethereum-mainnet",
					MaxAmountRequired: "1000000", // 1 USDC
					Asset:             "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
					PayTo:             "0xrecipient",
					Description:       "Pay with USDC on Ethereum",
				},
				{
					Scheme:            "eip3009",
					Network:           "polygon-mainnet",
					MaxAmountRequired: "1000000", // 1 USDC
					Asset:             "0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359",
					PayTo:             "0xrecipient",
					Description:       "Pay with USDC on Polygon",
				},
				{
					Scheme:            "eip3009",
					Network:           "base-mainnet",
					MaxAmountRequired: "500000", // 0.5 USDC (discount)
					Asset:             "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
					PayTo:             "0xrecipient",
					Description:       "Pay with USDC on Base (50% discount)",
				},
			},
		},
	}

	handler := NewX402Handler(mockHandler, config)

	// Request without payment should return all options
	reqBody := `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"multi-pay-tool"},"id":1}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Errorf("Expected 402, got %d", rr.Code)
	}

	// Check response contains all payment options
	var resp PaymentRequirements402Response
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if len(resp.Accepts) != 3 {
		t.Errorf("Expected 3 payment options, got %d", len(resp.Accepts))
	}

	// Verify each option
	if resp.Accepts[0].Network != "ethereum-mainnet" {
		t.Errorf("First option should be ethereum-mainnet, got %s", resp.Accepts[0].Network)
	}
	if resp.Accepts[1].Network != "polygon-mainnet" {
		t.Errorf("Second option should be polygon-mainnet, got %s", resp.Accepts[1].Network)
	}
	if resp.Accepts[2].Network != "base-mainnet" {
		t.Errorf("Third option should be base-mainnet, got %s", resp.Accepts[2].Network)
	}
	if resp.Accepts[2].MaxAmountRequired != "500000" {
		t.Errorf("Base option should have discount price, got %s", resp.Accepts[2].MaxAmountRequired)
	}
}

func TestX402Handler_PaymentMatching(t *testing.T) {
	mockHandler := &mockMCPHandler{
		response: `{"jsonrpc":"2.0","result":{"content":[{"type":"text","text":"success"}]},"id":1}`,
	}

	// Mock facilitator
	mockFacilitator := &MockFacilitator{
		verifyResponse: &VerifyResponse{
			IsValid: true,
			Payer:   "0xpayer",
		},
		settleResponse: &SettleResponse{
			Success:     true,
			Transaction: "0xtx",
			Network:     "base-mainnet",
		},
	}

	config := &Config{
		FacilitatorURL: "http://mock",
		PaymentTools: map[string][]PaymentRequirement{
			"multi-pay-tool": {
				{
					Scheme:            "eip3009",
					Network:           "ethereum-mainnet",
					MaxAmountRequired: "1000000",
					Asset:             "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
					PayTo:             "0xrecipient",
				},
				{
					Scheme:            "eip3009",
					Network:           "base-mainnet",
					MaxAmountRequired: "500000",
					Asset:             "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913",
					PayTo:             "0xrecipient",
				},
			},
		},
	}

	handler := NewX402Handler(mockHandler, config)
	handler.facilitator = mockFacilitator

	// Create payment for Base network (cheaper option)
	payment := &PaymentPayload{
		X402Version: 1,
		Scheme:      "eip3009",
		Network:     "base-mainnet",
	}
	payment.Payload.Signature = "0xsig"
	payment.Payload.Authorization.From = "0xpayer"
	payment.Payload.Authorization.To = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913" // Base USDC
	payment.Payload.Authorization.Value = "500000"

	paymentJSON, _ := json.Marshal(payment)
	paymentHeader := base64.StdEncoding.EncodeToString(paymentJSON)

	// Request with payment
	reqBody := `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"multi-pay-tool"},"id":1}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PAYMENT", paymentHeader)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		body, _ := io.ReadAll(rr.Body)
		t.Errorf("Expected 200, got %d. Body: %s", rr.Code, string(body))
	}

	if !mockHandler.called {
		t.Error("MCP handler should have been called with valid payment")
	}

	if !mockFacilitator.verifyCalled {
		t.Error("Facilitator verify should have been called")
	}
}
