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
		PaymentTools:   make(map[string]*PaymentRequirement),
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
		PaymentTools: map[string]*PaymentRequirement{
			"paid-tool": {
				Scheme:            "exact",
				Network:           "test",
				MaxAmountRequired: "1000",
				Asset:             "0xusdc",
				PayTo:             "0xrecipient",
				MaxTimeoutSeconds: 60,
			},
		},
		DefaultNetwork: "test",
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
		PaymentTools: map[string]*PaymentRequirement{
			"paid-tool": {
				Scheme:            "exact",
				Network:           "test",
				MaxAmountRequired: "1000",
				Asset:             "0xusdc",
				PayTo:             "0xrecipient",
				MaxTimeoutSeconds: 60,
			},
		},
		DefaultNetwork: "test",
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
	payment.Payload.Authorization.To = "0xrecipient"
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
