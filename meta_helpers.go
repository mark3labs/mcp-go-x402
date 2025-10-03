package x402

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// injectPaymentIntoMeta injects payment data into the request's _meta field
func injectPaymentIntoMeta(requestBody []byte, payment *PaymentPayload) ([]byte, error) {
	var req transport.JSONRPCRequest
	if err := json.Unmarshal(requestBody, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	// Convert params to map for manipulation
	var paramsMap map[string]any
	if req.Params != nil {
		// Try to convert params to map
		paramsBytes, err := json.Marshal(req.Params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		if err := json.Unmarshal(paramsBytes, &paramsMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal params to map: %w", err)
		}
	} else {
		paramsMap = make(map[string]any)
	}

	// Get or create _meta field
	var metaMap map[string]any
	if metaField, ok := paramsMap["_meta"].(map[string]any); ok {
		metaMap = metaField
	} else {
		metaMap = make(map[string]any)
	}

	// Convert payment to map for injection
	paymentBytes, err := json.Marshal(payment)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment: %w", err)
	}
	var paymentMap map[string]any
	if err := json.Unmarshal(paymentBytes, &paymentMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment to map: %w", err)
	}

	// Inject payment into _meta
	metaMap["x402/payment"] = paymentMap
	paramsMap["_meta"] = metaMap

	// Reconstruct request
	req.Params = paramsMap
	return json.Marshal(req)
}

// extractPaymentRequirementsFromError extracts payment requirements from JSON-RPC error response
func extractPaymentRequirementsFromError(body []byte) (*PaymentRequirementsResponse, error) {
	var jsonrpcResp transport.JSONRPCResponse
	if err := json.Unmarshal(body, &jsonrpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON-RPC response: %w", err)
	}

	// Check if it's a 402 error
	if jsonrpcResp.Error == nil {
		return nil, fmt.Errorf("expected error in JSON-RPC response")
	}

	if jsonrpcResp.Error.Code != 402 {
		return nil, fmt.Errorf("expected error code 402, got %d", jsonrpcResp.Error.Code)
	}

	// Extract payment requirements from error.data
	var requirements PaymentRequirementsResponse
	if err := json.Unmarshal(jsonrpcResp.Error.Data, &requirements); err != nil {
		return nil, fmt.Errorf("failed to parse payment requirements from error.data: %w", err)
	}

	return &requirements, nil
}

// extractSettlementFromMeta extracts settlement response from result's _meta field
func extractSettlementFromMeta(response *transport.JSONRPCResponse) (*SettlementResponse, error) {
	if response.Result == nil {
		return nil, nil
	}

	// Parse result to extract _meta
	var result struct {
		Meta *mcp.Meta `json:"_meta,omitempty"`
	}

	if err := json.Unmarshal(response.Result, &result); err != nil {
		// Not a fatal error - result might not have _meta
		return nil, nil
	}

	if result.Meta == nil || result.Meta.AdditionalFields == nil {
		return nil, nil
	}

	// Look for x402/payment-response in additional fields
	paymentRespField, ok := result.Meta.AdditionalFields["x402/payment-response"]
	if !ok {
		return nil, nil
	}

	// Convert to SettlementResponse
	settlementBytes, err := json.Marshal(paymentRespField)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settlement response: %w", err)
	}

	var settlement SettlementResponse
	if err := json.Unmarshal(settlementBytes, &settlement); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settlement response: %w", err)
	}

	return &settlement, nil
}
