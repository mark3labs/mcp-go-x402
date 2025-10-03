package server

import (
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// X402Server wraps an MCP server with x402 payment support
type X402Server struct {
	mcpServer *server.MCPServer
	config    *Config
}

// Option configures an X402Server
type Option func(*X402Server)

// WithFacilitator sets a custom facilitator
func WithFacilitator(f Facilitator) Option {
	return func(s *X402Server) {
		// Note: With middleware approach, facilitator is set during server creation
		// This option is kept for API compatibility but has no effect
	}
}

// NewX402Server creates a new x402-enabled MCP server
func NewX402Server(name, version string, config *Config, opts ...Option) *X402Server {
	// Create facilitator
	facilitator := NewHTTPFacilitator(config.FacilitatorURL)
	facilitator.SetVerbose(config.Verbose)

	// Create base MCP server with payment middleware
	mcpServer := server.NewMCPServer(name, version,
		server.WithToolHandlerMiddleware(newPaymentMiddleware(config, facilitator)),
	)

	srv := &X402Server{
		mcpServer: mcpServer,
		config:    config,
	}

	// Apply options
	for _, opt := range opts {
		opt(srv)
	}

	return srv
}

// AddTool adds a regular (non-paid) tool to the server
func (s *X402Server) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
	s.mcpServer.AddTool(tool, handler)
}

// AddPayableTool adds a tool that requires payment with one or more payment options
func (s *X402Server) AddPayableTool(
	tool mcp.Tool,
	handler server.ToolHandlerFunc,
	requirements ...PaymentRequirement,
) {
	// Add tool to MCP server
	s.mcpServer.AddTool(tool, handler)

	// Validate we have at least one requirement
	if len(requirements) == 0 {
		panic(fmt.Sprintf("tool %s requires at least one payment requirement", tool.Name))
	}

	// Register payment requirements
	if s.config.PaymentTools == nil {
		s.config.PaymentTools = make(map[string][]PaymentRequirement)
	}
	s.config.PaymentTools[tool.Name] = requirements
}

// Handler returns the http.Handler for the x402 server
func (s *X402Server) Handler() http.Handler {
	// Use the standard MCP HTTP server directly
	// No need for HTTP middleware wrapper anymore
	httpServer := server.NewStreamableHTTPServer(s.mcpServer)
	return httpServer
}

// Start starts the x402 server on the specified address
func (s *X402Server) Start(addr string) error {
	fmt.Printf("Starting X402 MCP Server on %s\n", addr)
	fmt.Printf("MCP endpoint: http://localhost%s\n", addr)

	return http.ListenAndServe(addr, s.Handler())
}
