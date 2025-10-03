package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type paymentError struct {
	details *mcp.JSONRPCErrorDetails
}

func (e *paymentError) Error() string {
	return e.details.Message
}

func (e *paymentError) JSONRPCErrorDetails() *mcp.JSONRPCErrorDetails {
	return e.details
}

func newPaymentError(code int, message string, data any) error {
	return &paymentError{
		details: &mcp.JSONRPCErrorDetails{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

type hasJSONRPCErrorDetails interface {
	JSONRPCErrorDetails() *mcp.JSONRPCErrorDetails
}

type contextKey string

const paymentErrorKey contextKey = "x402_payment_error"

func newPaymentMiddleware(config *Config, facilitator Facilitator) server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			requirements, needsPayment := config.PaymentTools[req.Params.Name]
			if !needsPayment {
				return next(ctx, req)
			}

			if config.Verbose {
				fmt.Printf("[X402] Tool '%s' requires payment, checking for payment in _meta\n", req.Params.Name)
			}

			for i := range requirements {
				requirements[i].Resource = fmt.Sprintf("mcp://tools/%s", req.Params.Name)
				if requirements[i].MimeType == "" {
					requirements[i].MimeType = "application/json"
				}
			}

			var paymentData any
			if req.Params.Meta != nil && req.Params.Meta.AdditionalFields != nil {
				paymentData = req.Params.Meta.AdditionalFields["x402/payment"]
			}

			if paymentData == nil {
				if config.Verbose {
					fmt.Printf("[X402] No payment found in _meta, returning 402 error\n")
				}
				// Store error details in error so interceptor can extract them
				return nil, newPaymentError(402, "Payment required", PaymentRequirements402Response{
					X402Version: 1,
					Error:       "Payment required to access this resource",
					Accepts:     requirements,
				})
			}

			if config.Verbose {
				fmt.Printf("[X402] Payment found in _meta, verifying...\n")
			}

			paymentBytes, err := json.Marshal(paymentData)
			if err != nil {
				return nil, newPaymentError(mcp.INVALID_PARAMS, "Invalid payment format", nil)
			}

			var payment PaymentPayload
			if err := json.Unmarshal(paymentBytes, &payment); err != nil {
				return nil, newPaymentError(mcp.INVALID_PARAMS, "Failed to parse payment data", nil)
			}

			requirement, err := findMatchingRequirement(&payment, requirements)
			if err != nil {
				if config.Verbose {
					fmt.Printf("[X402] Payment matching failed: %v\n", err)
				}
				return nil, newPaymentError(mcp.INVALID_PARAMS, fmt.Sprintf("Payment does not match requirements: %v", err), nil)
			}

			verifyResp, err := facilitator.Verify(ctx, &payment, requirement)
			if err != nil {
				if config.Verbose {
					fmt.Printf("[X402] Facilitator verification error: %v\n", err)
				}
				return nil, newPaymentError(mcp.INVALID_PARAMS, "Payment verification failed", nil)
			}

			if !verifyResp.IsValid {
				errorMsg := "Payment verification failed"
				if verifyResp.InvalidReason != "" {
					errorMsg = verifyResp.InvalidReason
				}
				if config.Verbose {
					fmt.Printf("[X402] Facilitator rejected payment: %s\n", errorMsg)
				}
				return nil, newPaymentError(mcp.INVALID_PARAMS, errorMsg, nil)
			}

			if config.Verbose {
				fmt.Printf("[X402] Payment verified successfully, payer: %s\n", verifyResp.Payer)
			}

			var settleResp *SettleResponse
			if !config.VerifyOnly {
				if config.Verbose {
					fmt.Printf("[X402] Settling payment on-chain...\n")
				}
				settleResp, err = facilitator.Settle(ctx, &payment, requirement)
				if err != nil || !settleResp.Success {
					errorMsg := "Payment settlement failed"
					if settleResp != nil && settleResp.ErrorReason != "" {
						errorMsg = settleResp.ErrorReason
					}
					if config.Verbose {
						fmt.Printf("[X402] Settlement failed: %s\n", errorMsg)
					}
					return nil, newPaymentError(mcp.INTERNAL_ERROR, errorMsg, nil)
				}
				if config.Verbose {
					fmt.Printf("[X402] Payment settled successfully, tx: %s\n", settleResp.Transaction)
				}
			} else {
				if config.Verbose {
					fmt.Printf("[X402] Verify-only mode, skipping settlement\n")
				}
				settleResp = &SettleResponse{
					Success:     true,
					Transaction: "verify-only-mode",
					Network:     payment.Network,
					Payer:       verifyResp.Payer,
				}
			}

			result, err := next(ctx, req)
			if err != nil {
				return nil, err
			}

			if result != nil {
				if result.Meta == nil {
					result.Meta = &mcp.Meta{
						AdditionalFields: make(map[string]any),
					}
				}
				if result.Meta.AdditionalFields == nil {
					result.Meta.AdditionalFields = make(map[string]any)
				}

				result.Meta.AdditionalFields["x402/payment-response"] = SettlementResponse{
					Success:     settleResp.Success,
					Transaction: settleResp.Transaction,
					Network:     settleResp.Network,
					Payer:       settleResp.Payer,
				}
			}

			return result, nil
		}
	}
}

func findMatchingRequirement(payment *PaymentPayload, requirements []PaymentRequirement) (*PaymentRequirement, error) {
	for i := range requirements {
		req := &requirements[i]

		if req.Network != "" && req.Network != payment.Network {
			continue
		}

		if req.Scheme != "" && req.Scheme != payment.Scheme {
			continue
		}

		return req, nil
	}

	return nil, fmt.Errorf("no matching payment requirement found for network=%s, scheme=%s",
		payment.Network, payment.Scheme)
}
