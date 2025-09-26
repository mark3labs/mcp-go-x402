package server

import (
	"bytes"
	"encoding/base64"
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
	// Based on mcp.CallToolRequest and JSONRPC structure
	var mcpReq struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Name      string `json:"name"`      // Tool name from CallToolParams
			Arguments any    `json:"arguments"` // Tool arguments
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

	// Check for payment header
	paymentHeader := r.Header.Get("X-PAYMENT")
	if paymentHeader == "" {
		h.send402Response(w, requirement, mcpReq.Params.Name)
		return
	}

	// Decode and verify payment
	payment, err := h.decodePaymentHeader(paymentHeader)
	if err != nil {
		http.Error(w, "Invalid payment header", http.StatusBadRequest)
		return
	}

	// Verify payment with facilitator
	ctx := r.Context()
	verifyResp, err := h.facilitator.Verify(ctx, payment, requirement)
	if err != nil || !verifyResp.IsValid {
		errorMsg := "Payment verification failed"
		if verifyResp != nil && verifyResp.InvalidReason != "" {
			errorMsg = verifyResp.InvalidReason
		}
		http.Error(w, errorMsg, http.StatusBadRequest)
		return
	}

	// Settle payment if not in verify-only mode
	var settleResp *SettleResponse
	if !h.config.VerifyOnly {
		settleResp, err = h.facilitator.Settle(ctx, payment, requirement)
		if err != nil || !settleResp.Success {
			errorMsg := "Payment settlement failed"
			if settleResp != nil && settleResp.ErrorReason != "" {
				errorMsg = settleResp.ErrorReason
			}
			http.Error(w, errorMsg, http.StatusBadRequest)
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

	// Forward request to MCP handler with payment confirmation
	h.forwardWithPaymentResponse(w, r, settleResp.Transaction, verifyResp.Payer)
}

func (h *X402Handler) send402Response(w http.ResponseWriter, requirement *PaymentRequirement, toolName string) {
	// Construct full resource URL
	requirement.Resource = fmt.Sprintf("mcp://tools/%s", toolName)

	response := PaymentRequirements402Response{
		X402Version: 1,
		Error:       "X-PAYMENT header is required",
		Accepts:     []PaymentRequirement{*requirement},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)
	json.NewEncoder(w).Encode(response)
}

func (h *X402Handler) decodePaymentHeader(header string) (*PaymentPayload, error) {
	decoded, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	var payment PaymentPayload
	if err := json.Unmarshal(decoded, &payment); err != nil {
		return nil, fmt.Errorf("unmarshal payment: %w", err)
	}

	if payment.X402Version != 1 {
		return nil, fmt.Errorf("unsupported x402 version: %d", payment.X402Version)
	}

	return &payment, nil
}

func (h *X402Handler) forwardWithPaymentResponse(w http.ResponseWriter, r *http.Request, transaction string, payer string) {
	// Create a custom ResponseWriter to capture the response
	captureWriter := &responseCapture{
		ResponseWriter: w,
		statusCode:     200,
		headers:        make(http.Header),
	}

	// Forward to MCP handler
	h.mcpHandler.ServeHTTP(captureWriter, r)

	// Add payment response header if successful
	if captureWriter.statusCode == 200 {
		paymentResp := SettlementResponse{
			Success:     true,
			Transaction: transaction,
			Network:     h.config.DefaultNetwork,
			Payer:       payer,
		}

		respJSON, _ := json.Marshal(paymentResp)
		encoded := base64.StdEncoding.EncodeToString(respJSON)
		w.Header().Set("X-PAYMENT-RESPONSE", encoded)
	}

	// Write captured headers and body
	for k, v := range captureWriter.headers {
		w.Header()[k] = v
	}
	w.WriteHeader(captureWriter.statusCode)
	w.Write(captureWriter.body.Bytes())
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
