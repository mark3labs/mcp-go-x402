package server

import (
	"fmt"
	"net/http"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// X402Server wraps an MCP server with x402 payment support
type X402Server struct {
	mcpServer   *server.MCPServer
	httpServer  *server.StreamableHTTPServer
	x402Handler *X402Handler
	config      *Config
}

// Option configures an X402Server
type Option func(*X402Server)

// WithFacilitator sets a custom facilitator
func WithFacilitator(f Facilitator) Option {
	return func(s *X402Server) {
		s.x402Handler.facilitator = f
	}
}

// NewX402Server creates a new x402-enabled MCP server
func NewX402Server(name, version string, config *Config, opts ...Option) *X402Server {
	// Create base MCP server
	mcpServer := server.NewMCPServer(name, version)

	// Create HTTP server
	httpServer := server.NewStreamableHTTPServer(mcpServer)

	// Create x402 handler wrapper
	x402Handler := NewX402Handler(httpServer, config)

	srv := &X402Server{
		mcpServer:   mcpServer,
		httpServer:  httpServer,
		x402Handler: x402Handler,
		config:      config,
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
	return s.x402Handler
}

// Start starts the x402 server on the specified address
func (s *X402Server) Start(addr string) error {
	fmt.Printf("Starting X402 MCP Server on %s\n", addr)
	fmt.Printf("MCP endpoint: http://localhost%s\n", addr)

	// The httpServer (StreamableHTTPServer) handles routing internally
	// and expects to be served at the root. It adds /mcp and other routes itself.
	return http.ListenAndServe(addr, s.x402Handler)
}
