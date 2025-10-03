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

// NewX402Server creates a new x402-enabled MCP server
func NewX402Server(name, version string, config *Config) *X402Server {
	// Create base MCP server
	mcpServer := server.NewMCPServer(name, version)

	srv := &X402Server{
		mcpServer: mcpServer,
		config:    config,
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
	// Wrap MCP HTTP server with x402 payment handler
	httpServer := server.NewStreamableHTTPServer(s.mcpServer)
	return NewX402Handler(httpServer, s.config)
}

// Start starts the x402 server on the specified address
func (s *X402Server) Start(addr string) error {
	fmt.Printf("Starting X402 MCP Server on %s\n", addr)
	fmt.Printf("MCP endpoint: http://localhost%s\n", addr)

	return http.ListenAndServe(addr, s.Handler())
}
