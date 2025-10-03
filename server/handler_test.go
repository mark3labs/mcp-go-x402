package server

import (
	"bytes"
	"context"
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

	// New spec: JSON-RPC errors use HTTP 200
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rr.Code)
	}

	// Check response is JSON-RPC error with code 402
	var resp map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}

	// Check error field exists
	errorField, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatal("Expected error field in response")
	}

	// Check error code is 402
	code, ok := errorField["code"].(float64)
	if !ok || int(code) != 402 {
		t.Errorf("Expected error code 402, got %v", errorField["code"])
	}

	// Check error data contains payment requirements
	dataBytes, _ := json.Marshal(errorField["data"])
	var paymentReq PaymentRequirements402Response
	if err := json.Unmarshal(dataBytes, &paymentReq); err != nil {
		t.Fatal(err)
	}

	if len(paymentReq.Accepts) != 1 {
		t.Error("Expected payment requirements")
	}

	if paymentReq.Accepts[0].MaxAmountRequired != "1000" {
		t.Errorf("Wrong amount: %s", paymentReq.Accepts[0].MaxAmountRequired)
	}

	if mockHandler.called {
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
			Payer:       "0xpayer",
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

	// Create payment in _meta field
	payment := map[string]any{
		"x402Version": 1,
		"scheme":      "exact",
		"network":     "test",
		"payload": map[string]any{
			"signature": "0xsig",
			"authorization": map[string]any{
				"from":        "0xpayer",
				"to":          "0xrecipient",
				"value":       "1000",
				"validAfter":  "0",
				"validBefore": "9999999999",
				"nonce":       "0x123",
			},
		},
	}

	// Request with payment in _meta
	reqBody := map[string]any{
		"jsonrpc": "2.0",
		"method":  "tools/call",
		"params": map[string]any{
			"name": "paid-tool",
			"_meta": map[string]any{
				"x402/payment": payment,
			},
		},
		"id": 1,
	}
	reqBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		body, _ := io.ReadAll(rr.Body)
		t.Errorf("Expected 200, got %d. Body: %s", rr.Code, string(body))
	}

	if !mockHandler.called {
		t.Error("MCP handler should have been called with valid payment")
	}

	// Check for payment response in result._meta
	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatal("Expected result field in response")
	}

	meta, ok := result["_meta"].(map[string]any)
	if !ok {
		t.Fatal("Expected _meta field in result")
	}

	paymentResp, ok := meta["x402/payment-response"].(map[string]any)
	if !ok {
		t.Error("Expected x402/payment-response in _meta")
	}

	if success, _ := paymentResp["success"].(bool); !success {
		t.Error("Expected payment response to indicate success")
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
