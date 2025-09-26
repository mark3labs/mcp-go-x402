package x402

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// X402Transport implements transport.Interface with x402 payment support
type X402Transport struct {
	serverURL  *url.URL
	httpClient *http.Client
	handler    *PaymentHandler

	// Session management (from StreamableHTTP)
	sessionID       atomic.Value
	protocolVersion atomic.Value
	initialized     chan struct{}
	initializedOnce sync.Once

	// Notification handling
	notificationHandler func(mcp.JSONRPCNotification)
	notifyMu            sync.RWMutex

	// Request handling
	requestHandler transport.RequestHandler
	requestMu      sync.RWMutex

	// Event callbacks
	onPaymentAttempt func(PaymentEvent)
	onPaymentSuccess func(PaymentEvent)
	onPaymentFailure func(PaymentEvent, error)

	// State
	closed chan struct{}
	wg     sync.WaitGroup

	// Testing support
	paymentRecorder *PaymentRecorder
}

// Config configures the X402Transport
type Config struct {
	ServerURL        string
	Signer           PaymentSigner
	MaxPaymentAmount string
	AutoPayThreshold string
	RateLimits       *RateLimits
	PaymentCallback  func(amount *big.Int, resource string) bool
	HTTPClient       *http.Client
	OnPaymentAttempt func(PaymentEvent)
	OnPaymentSuccess func(PaymentEvent)
	OnPaymentFailure func(PaymentEvent, error)
}

// New creates a new X402Transport
func New(config Config) (*X402Transport, error) {
	parsedURL, err := url.Parse(config.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	handlerConfig := &HandlerConfig{
		MaxPaymentAmount: config.MaxPaymentAmount,
		AutoPayThreshold: config.AutoPayThreshold,
		RateLimits:       config.RateLimits,
		PaymentCallback:  config.PaymentCallback,
	}

	handler, err := NewPaymentHandler(config.Signer, handlerConfig)
	if err != nil {
		return nil, err
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 2 * time.Minute,
		}
	}

	t := &X402Transport{
		serverURL:        parsedURL,
		httpClient:       httpClient,
		handler:          handler,
		closed:           make(chan struct{}),
		initialized:      make(chan struct{}),
		onPaymentAttempt: config.OnPaymentAttempt,
		onPaymentSuccess: config.OnPaymentSuccess,
		onPaymentFailure: config.OnPaymentFailure,
	}

	t.sessionID.Store("")
	t.protocolVersion.Store("")

	return t, nil
}

// Start implements transport.Interface
func (t *X402Transport) Start(ctx context.Context) error {
	// Similar to StreamableHTTP, we don't need persistent connection
	return nil
}

// Close implements transport.Interface
func (t *X402Transport) Close() error {
	select {
	case <-t.closed:
		return nil
	default:
	}

	close(t.closed)

	// Send session close if we have a session
	sessionID := t.sessionID.Load().(string)
	if sessionID != "" {
		t.sessionID.Store("")
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodDelete, t.serverURL.String(), nil)
			if err != nil {
				return
			}

			req.Header.Set("X-Session-Id", sessionID)
			if version := t.protocolVersion.Load().(string); version != "" {
				req.Header.Set("X-Protocol-Version", version)
			}

			t.httpClient.Do(req)
		}()
	}

	t.wg.Wait()
	return nil
}

// SetProtocolVersion implements transport.Interface
func (t *X402Transport) SetProtocolVersion(version string) {
	t.protocolVersion.Store(version)
}

// SendRequest implements transport.Interface with x402 payment handling
func (t *X402Transport) SendRequest(ctx context.Context, request transport.JSONRPCRequest) (*transport.JSONRPCResponse, error) {
	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Try request without payment first
	resp, err := t.sendHTTP(ctx, http.MethodPost, bytes.NewReader(requestBody), nil)
	if err != nil {
		return nil, err
	}

	// Check for 402 Payment Required
	if resp.StatusCode == http.StatusPaymentRequired {
		defer resp.Body.Close()

		// Parse payment requirements
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read 402 response: %w", err)
		}

		var requirements PaymentRequirementsResponse
		if err := json.Unmarshal(body, &requirements); err != nil {
			return nil, fmt.Errorf("failed to parse payment requirements: %w", err)
		}

		// Record payment attempt
		t.recordPaymentEvent(PaymentEventAttempt, request.Method, requirements)

		// Create and sign payment
		payment, err := t.handler.CreatePayment(ctx, requirements)
		if err != nil {
			t.recordPaymentError(PaymentEventFailure, request.Method, requirements, err)
			return nil, fmt.Errorf("failed to create payment: %w", err)
		}

		// Retry request with payment
		headers := map[string]string{
			"X-PAYMENT": payment.Encode(),
		}

		resp2, err := t.sendHTTP(ctx, http.MethodPost, bytes.NewReader(requestBody), headers)
		if err != nil {
			t.recordPaymentError(PaymentEventFailure, request.Method, requirements, err)
			return nil, err
		}

		resp = resp2

		// Check for payment response header
		if paymentResponse := resp.Header.Get("X-PAYMENT-RESPONSE"); paymentResponse != "" {
			// Payment was processed
			t.recordPaymentEvent(PaymentEventSuccess, request.Method, requirements)
		}
	}

	defer resp.Body.Close()

	// Handle response based on content type
	return t.handleResponse(ctx, resp)
}

// sendHTTP sends an HTTP request with standard headers
func (t *X402Transport) sendHTTP(ctx context.Context, method string, body io.Reader, extraHeaders map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, t.serverURL.String(), body)
	if err != nil {
		return nil, err
	}

	// Set standard headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	// Add session headers if available
	if sessionID := t.sessionID.Load().(string); sessionID != "" {
		req.Header.Set("X-Session-Id", sessionID)
	}

	if version := t.protocolVersion.Load().(string); version != "" {
		req.Header.Set("X-Protocol-Version", version)
	}

	// Add extra headers (like X-PAYMENT)
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}

	return t.httpClient.Do(req)
}

// handleResponse processes HTTP responses (JSON or SSE)
func (t *X402Transport) handleResponse(ctx context.Context, resp *http.Response) (*transport.JSONRPCResponse, error) {
	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)

		// Try to parse as JSON-RPC error
		var errResponse transport.JSONRPCResponse
		if err := json.Unmarshal(body, &errResponse); err == nil {
			return &errResponse, nil
		}

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, body)
	}

	// Handle based on content type
	mediaType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))

	switch mediaType {
	case "application/json":
		var response transport.JSONRPCResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &response, nil

	case "text/event-stream":
		return t.handleSSEResponse(ctx, resp.Body)

	default:
		return nil, fmt.Errorf("unexpected content type: %s", mediaType)
	}
}

// handleSSEResponse processes Server-Sent Events stream
func (t *X402Transport) handleSSEResponse(ctx context.Context, reader io.ReadCloser) (*transport.JSONRPCResponse, error) {
	defer reader.Close()

	responseChan := make(chan *transport.JSONRPCResponse, 1)

	go func() {
		defer close(responseChan)

		scanner := bufio.NewScanner(reader)
		var data string

		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "data:") {
				data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			} else if line == "" && data != "" {
				// Empty line means end of event
				var response transport.JSONRPCResponse
				if err := json.Unmarshal([]byte(data), &response); err == nil {
					if !response.ID.IsNil() {
						responseChan <- &response
						return
					}

					// Handle notification
					var notification mcp.JSONRPCNotification
					if err := json.Unmarshal([]byte(data), &notification); err == nil {
						t.notifyMu.RLock()
						if t.notificationHandler != nil {
							t.notificationHandler(notification)
						}
						t.notifyMu.RUnlock()
					}
				}
				data = ""
			}
		}
	}()

	select {
	case response := <-responseChan:
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// SendNotification implements transport.Interface
func (t *X402Transport) SendNotification(ctx context.Context, notification mcp.JSONRPCNotification) error {
	notificationBody, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	resp, err := t.sendHTTP(ctx, http.MethodPost, bytes.NewReader(notificationBody), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Handle 402 for notifications (unusual but possible)
	if resp.StatusCode == http.StatusPaymentRequired {
		// Handle payment and retry
		// Similar logic to SendRequest
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("notification failed with status %d: %s", resp.StatusCode, body)
	}

	return nil
}

// SetNotificationHandler implements transport.Interface
func (t *X402Transport) SetNotificationHandler(handler func(mcp.JSONRPCNotification)) {
	t.notifyMu.Lock()
	defer t.notifyMu.Unlock()
	t.notificationHandler = handler
}

// SetRequestHandler implements transport.Interface
func (t *X402Transport) SetRequestHandler(handler transport.RequestHandler) {
	t.requestMu.Lock()
	defer t.requestMu.Unlock()
	t.requestHandler = handler
}

// GetSessionId implements transport.Interface
func (t *X402Transport) GetSessionId() string {
	if sessionID := t.sessionID.Load(); sessionID != nil {
		return sessionID.(string)
	}
	return ""
}

// GetMetrics returns payment metrics
func (t *X402Transport) GetMetrics() BudgetMetrics {
	return t.handler.GetMetrics()
}

// Helper methods for event recording

func (t *X402Transport) recordPaymentEvent(eventType PaymentEventType, method string, reqs PaymentRequirementsResponse) {
	if len(reqs.Accepts) == 0 {
		return
	}

	req := reqs.Accepts[0]
	amount := new(big.Int)
	amount.SetString(req.MaxAmountRequired, 10)

	event := PaymentEvent{
		Type:      eventType,
		Resource:  req.Resource,
		Method:    method,
		Amount:    amount,
		Network:   req.Network,
		Asset:     req.Asset,
		Recipient: req.PayTo,
		Timestamp: time.Now().Unix(),
	}

	switch eventType {
	case PaymentEventAttempt:
		if t.onPaymentAttempt != nil {
			t.onPaymentAttempt(event)
		}
	case PaymentEventSuccess:
		if t.onPaymentSuccess != nil {
			t.onPaymentSuccess(event)
		}
	}

	if t.paymentRecorder != nil {
		t.paymentRecorder.Record(event)
	}
}

func (t *X402Transport) recordPaymentError(eventType PaymentEventType, method string, reqs PaymentRequirementsResponse, err error) {
	if len(reqs.Accepts) == 0 {
		return
	}

	req := reqs.Accepts[0]
	amount := new(big.Int)
	amount.SetString(req.MaxAmountRequired, 10)

	event := PaymentEvent{
		Type:      eventType,
		Resource:  req.Resource,
		Method:    method,
		Amount:    amount,
		Network:   req.Network,
		Asset:     req.Asset,
		Recipient: req.PayTo,
		Error:     err,
		Timestamp: time.Now().Unix(),
	}

	if t.onPaymentFailure != nil {
		t.onPaymentFailure(event, err)
	}

	if t.paymentRecorder != nil {
		t.paymentRecorder.Record(event)
	}
}

// WithPaymentRecorder adds a payment recorder for testing
func WithPaymentRecorder(recorder *PaymentRecorder) func(*X402Transport) {
	return func(t *X402Transport) {
		t.paymentRecorder = recorder
	}
}
