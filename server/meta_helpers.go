package server

import (
	"encoding/json"
	"fmt"
)

// extractPaymentFromMeta extracts payment data from request params' _meta field
func extractPaymentFromMeta(requestBody []byte) (*PaymentPayload, error) {
	var mcpReq struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
		Params  struct {
			Name      string         `json:"name"`
			Arguments any            `json:"arguments"`
			Meta      map[string]any `json:"_meta,omitempty"`
		} `json:"params"`
		ID any `json:"id"`
	}

	if err := json.Unmarshal(requestBody, &mcpReq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	if mcpReq.Params.Meta == nil {
		return nil, nil
	}

	paymentField, ok := mcpReq.Params.Meta["x402/payment"]
	if !ok {
		return nil, nil
	}

	// Convert to PaymentPayload
	paymentBytes, err := json.Marshal(paymentField)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment field: %w", err)
	}

	var payment PaymentPayload
	if err := json.Unmarshal(paymentBytes, &payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment: %w", err)
	}

	return &payment, nil
}

// createJSONRPC402Error creates a JSON-RPC error response with code 402
func createJSONRPC402Error(requestID any, requirement *PaymentRequirement, toolName string) ([]byte, error) {
	// Create payment requirements
	reqCopy := *requirement
	reqCopy.Resource = fmt.Sprintf("mcp://tool/%s", toolName)
	if reqCopy.MimeType == "" {
		reqCopy.MimeType = "application/json"
	}

	paymentReqData := PaymentRequirements402Response{
		X402Version: 1,
		Error:       "Payment required",
		Accepts:     []PaymentRequirement{reqCopy},
	}

	data, err := json.Marshal(paymentReqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment requirements: %w", err)
	}

	response := map[string]any{
		"jsonrpc": "2.0",
		"id":      requestID,
		"error": map[string]any{
			"code":    402,
			"message": "Payment required",
			"data":    json.RawMessage(data),
		},
	}

	return json.Marshal(response)
}

// injectSettlementIntoResponse injects settlement response into result's _meta field
func injectSettlementIntoResponse(responseBody []byte, settlement SettlementResponse) ([]byte, error) {
	var resp map[string]any
	if err := json.Unmarshal(responseBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Get result field
	result, ok := resp["result"]
	if !ok {
		// No result field - might be an error response
		return responseBody, nil
	}

	// Convert result to map
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var resultMap map[string]any
	if err := json.Unmarshal(resultBytes, &resultMap); err != nil {
		// Result is not a map, can't inject _meta
		return responseBody, nil
	}

	// Get or create _meta field
	var metaMap map[string]any
	if metaField, ok := resultMap["_meta"].(map[string]any); ok {
		metaMap = metaField
	} else {
		metaMap = make(map[string]any)
	}

	// Inject settlement response
	settlementMap := map[string]any{
		"success":     settlement.Success,
		"transaction": settlement.Transaction,
		"network":     settlement.Network,
		"payer":       settlement.Payer,
	}
	if settlement.ErrorReason != "" {
		settlementMap["errorReason"] = settlement.ErrorReason
	}

	metaMap["x402/payment-response"] = settlementMap
	resultMap["_meta"] = metaMap
	resp["result"] = resultMap

	return json.Marshal(resp)
}
