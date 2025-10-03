package x402

import (
	"encoding/json"
	"io"
	"net/http"
)

// create402JSONRPCResponse creates a JSON-RPC 402 error response
func create402JSONRPCResponse(requestID any, requirement PaymentRequirement) map[string]any {
	paymentReqData := PaymentRequirementsResponse{
		X402Version: 1,
		Error:       "Payment required",
		Accepts:     []PaymentRequirement{requirement},
	}
	data, _ := json.Marshal(paymentReqData)

	return map[string]any{
		"jsonrpc": "2.0",
		"id":      requestID,
		"error": map[string]any{
			"code":    402,
			"message": "Payment required",
			"data":    json.RawMessage(data),
		},
	}
}

// createSuccessResponse creates a successful JSON-RPC response with payment settlement in _meta
func createSuccessResponse(requestID any, resultData any, payer string) map[string]any {
	result := map[string]any{
		"data": resultData,
		"_meta": map[string]any{
			"x402/payment-response": map[string]any{
				"success":     true,
				"transaction": "0x123",
				"network":     "base-sepolia",
				"payer":       payer,
			},
		},
	}
	resultBytes, _ := json.Marshal(result)

	return map[string]any{
		"jsonrpc": "2.0",
		"id":      requestID,
		"result":  json.RawMessage(resultBytes),
	}
}

// hasPaymentInMeta checks if the request has payment in _meta field
func hasPaymentInMeta(r *http.Request) bool {
	bodyBytes, _ := io.ReadAll(r.Body)
	var req map[string]any
	json.Unmarshal(bodyBytes, &req)

	params, ok := req["params"].(map[string]any)
	if !ok {
		return false
	}

	meta, ok := params["_meta"].(map[string]any)
	if !ok {
		return false
	}

	_, hasPayment := meta["x402/payment"]
	return hasPayment
}
