package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// X402Handler wraps an MCP HTTP handler with x402 payment support
type X402Handler struct {
	mcpHandler  http.Handler
	config      *Config
	facilitator Facilitator
}

// NewX402Handler creates a new x402 handler wrapper
func NewX402Handler(mcpHandler http.Handler, config *Config) *X402Handler {
	return &X402Handler{
		mcpHandler:  mcpHandler,
		config:      config,
		facilitator: NewHTTPFacilitator(config.FacilitatorURL),
	}
}

func (h *X402Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only intercept POST requests (MCP tool calls)
	if r.Method != http.MethodPost {
		h.mcpHandler.ServeHTTP(w, r)
		return
	}

	// Read and buffer the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Parse to check if it's a tool call
	var mcpReq struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Name      string `json:"name"`
			Arguments any    `json:"arguments"`
		} `json:"params"`
		ID any `json:"id"`
	}

	if err := json.Unmarshal(body, &mcpReq); err != nil {
		// Not valid JSON, let MCP handler deal with it
		h.mcpHandler.ServeHTTP(w, r)
		return
	}

	// Check if this is a tool call (JSON-RPC method)
	if mcpReq.Method != "tools/call" {
		h.mcpHandler.ServeHTTP(w, r)
		return
	}

	// Check if this tool requires payment
	requirement, needsPayment := h.config.PaymentTools[mcpReq.Params.Name]
	if !needsPayment {
		h.mcpHandler.ServeHTTP(w, r)
		return
	}

	// Make a copy of the requirement and ensure all required fields are set
	reqCopy := *requirement
	reqCopy.Resource = fmt.Sprintf("mcp://tool/%s", mcpReq.Params.Name)
	if reqCopy.MimeType == "" {
		reqCopy.MimeType = "application/json"
	}
	requirement = &reqCopy

	// Extract payment from _meta field
	payment, err := extractPaymentFromMeta(body)
	if err != nil {
		fmt.Printf("Failed to extract payment from _meta: %v\n", err)
		h.sendJSONRPCError(w, mcpReq.ID, -32602, "Invalid payment in _meta field")
		return
	}

	if payment == nil {
		// No payment provided - send 402 error
		h.send402JSONRPCError(w, requirement, mcpReq.Params.Name, mcpReq.ID)
		return
	}

	// Verify payment with facilitator
	ctx := r.Context()
	verifyResp, err := h.facilitator.Verify(ctx, payment, requirement)
	if err != nil {
		h.sendJSONRPCError(w, mcpReq.ID, -32603, fmt.Sprintf("Payment verification failed: %v", err))
		return
	}
	if !verifyResp.IsValid {
		errorMsg := "Payment verification failed"
		if verifyResp.InvalidReason != "" {
			errorMsg = verifyResp.InvalidReason
		}
		h.sendJSONRPCError(w, mcpReq.ID, -32602, errorMsg)
		return
	}

	// Settle payment if not in verify-only mode
	var settleResp *SettleResponse
	if !h.config.VerifyOnly {
		settleResp, err = h.facilitator.Settle(ctx, payment, requirement)
		if err != nil || !settleResp.Success {
			// Convert to SettlementResponse for error response
			settlement := &SettlementResponse{
				Success:     false,
				Transaction: "",
				Network:     h.config.DefaultNetwork,
				Payer:       verifyResp.Payer,
			}
			if settleResp != nil {
				settlement.ErrorReason = settleResp.ErrorReason
				settlement.Transaction = settleResp.Transaction
			}
			// Send 402 error with payment-response in error.data
			h.send402ErrorWithSettlement(w, requirement, mcpReq.Params.Name, mcpReq.ID, settlement)
			return
		}
	} else {
		// In verify-only mode, create a mock settle response
		settleResp = &SettleResponse{
			Success:     true,
			Transaction: "verify-only-mode",
			Network:     h.config.DefaultNetwork,
			Payer:       verifyResp.Payer,
		}
	}

	// Convert SettleResponse to SettlementResponse for forwarding
	settlement := &SettlementResponse{
		Success:     settleResp.Success,
		Transaction: settleResp.Transaction,
		Network:     settleResp.Network,
		Payer:       settleResp.Payer,
	}

	// Forward request to MCP handler and inject settlement into response
	h.forwardWithPaymentResponse(w, r, settlement)
}

// send402JSONRPCError sends a JSON-RPC error response with code 402
func (h *X402Handler) send402JSONRPCError(w http.ResponseWriter, requirement *PaymentRequirement, toolName string, requestID any) {
	responseBytes, err := createJSONRPC402Error(requestID, requirement, toolName)
	if err != nil {
		http.Error(w, "Failed to create error response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors use HTTP 200
	_, _ = w.Write(responseBytes)
}

// sendJSONRPCError sends a generic JSON-RPC error response
func (h *X402Handler) sendJSONRPCError(w http.ResponseWriter, requestID any, code int, message string) {
	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      requestID,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// send402ErrorWithSettlement sends 402 error with settlement info in error.data
func (h *X402Handler) send402ErrorWithSettlement(w http.ResponseWriter, requirement *PaymentRequirement, toolName string, requestID any, settlement *SettlementResponse) {
	// Create payment requirements
	reqCopy := *requirement
	reqCopy.Resource = fmt.Sprintf("mcp://tool/%s", toolName)

	paymentReqData := PaymentRequirements402Response{
		X402Version: 1,
		Error:       "Payment settlement failed",
		Accepts:     []PaymentRequirement{reqCopy},
	}

	// Create error data with both payment requirements and settlement response
	errorData := map[string]any{
		"x402Version": paymentReqData.X402Version,
		"error":       paymentReqData.Error,
		"accepts":     paymentReqData.Accepts,
	}

	if settlement != nil {
		errorData["x402/payment-response"] = map[string]any{
			"success":     settlement.Success,
			"errorReason": settlement.ErrorReason,
			"transaction": settlement.Transaction,
			"network":     settlement.Network,
			"payer":       settlement.Payer,
		}
	}

	errorDataBytes, _ := json.Marshal(errorData)

	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      requestID,
		"error": map[string]any{
			"code":    402,
			"message": "Payment settlement failed",
			"data":    json.RawMessage(errorDataBytes),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

func (h *X402Handler) forwardWithPaymentResponse(w http.ResponseWriter, r *http.Request, settlement *SettlementResponse) {
	// Create a custom ResponseWriter to capture the response
	captureWriter := &responseCapture{
		ResponseWriter: w,
		statusCode:     200,
		headers:        make(http.Header),
	}

	// Forward to MCP handler
	h.mcpHandler.ServeHTTP(captureWriter, r)

	// Inject settlement into response _meta if successful
	if captureWriter.statusCode == 200 {
		settlementResp := SettlementResponse{
			Success:     settlement.Success,
			Transaction: settlement.Transaction,
			Network:     settlement.Network,
			Payer:       settlement.Payer,
		}

		// Inject into response body
		modifiedBody, err := injectSettlementIntoResponse(captureWriter.body.Bytes(), settlementResp)
		if err != nil {
			// If we can't inject, just return the original response
			fmt.Printf("Warning: failed to inject settlement into response: %v\n", err)
			modifiedBody = captureWriter.body.Bytes()
		}

		// Write modified response
		for k, v := range captureWriter.headers {
			w.Header()[k] = v
		}
		w.WriteHeader(captureWriter.statusCode)
		_, _ = w.Write(modifiedBody)
	} else {
		// Non-200 response, forward as-is
		for k, v := range captureWriter.headers {
			w.Header()[k] = v
		}
		w.WriteHeader(captureWriter.statusCode)
		_, _ = w.Write(captureWriter.body.Bytes())
	}
}

// responseCapture captures HTTP response for modification
type responseCapture struct {
	http.ResponseWriter
	statusCode int
	headers    http.Header
	body       bytes.Buffer
}

func (c *responseCapture) Header() http.Header {
	return c.headers
}

func (c *responseCapture) Write(b []byte) (int, error) {
	return c.body.Write(b)
}

func (c *responseCapture) WriteHeader(statusCode int) {
	c.statusCode = statusCode
}
