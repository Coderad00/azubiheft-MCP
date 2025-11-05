package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// ToolHandler is a function that handles tool execution
type ToolHandler func(ctx context.Context, params map[string]interface{}) (string, error)

// Server represents an MCP server
type Server struct {
	name     string
	version  string
	tools    map[string]Tool
	handlers map[string]ToolHandler
	logger   *log.Logger
}

// NewServer creates a new MCP server
func NewServer(name, version string, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.New(os.Stderr, "[mcp] ", log.LstdFlags)
	}
	return &Server{
		name:     name,
		version:  version,
		tools:    make(map[string]Tool),
		handlers: make(map[string]ToolHandler),
		logger:   logger,
	}
}

// RegisterTool registers a tool with its handler
func (s *Server) RegisterTool(name, description string, inputSchema map[string]interface{}, handler ToolHandler) {
	tool := Tool{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
	}
	s.tools[name] = tool
	s.handlers[name] = handler
}

// Serve starts the server and handles stdio communication
func (s *Server) Serve() error {
	reader := bufio.NewReader(os.Stdin)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("error reading input: %w", err)
		}

		// Parse request
		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.logger.Printf("Error parsing request: %v", err)
			s.sendError(nil, -32700, "Parse error", nil)
			continue
		}

		// Handle request
		s.handleRequest(req)
	}
}

// handleRequest processes a JSON-RPC request
func (s *Server) handleRequest(req JSONRPCRequest) {
	ctx := context.Background()

	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(ctx, req)
	case "ping":
		s.sendResult(req.ID, map[string]interface{}{})
	default:
		s.sendError(req.ID, -32601, "Method not found", nil)
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req JSONRPCRequest) {
	result := InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: Capabilities{
			Tools: &ToolsCapability{},
		},
		ServerInfo: ServerInfo{
			Name:    s.name,
			Version: s.version,
		},
	}
	s.sendResult(req.ID, result)
}

// handleToolsList handles the tools/list request
func (s *Server) handleToolsList(req JSONRPCRequest) {
	tools := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}

	result := map[string]interface{}{
		"tools": tools,
	}
	s.sendResult(req.ID, result)
}

// handleToolsCall handles the tools/call request
func (s *Server) handleToolsCall(ctx context.Context, req JSONRPCRequest) {
	// Extract tool name and arguments
	toolName, ok := req.Params["name"].(string)
	if !ok {
		s.sendError(req.ID, -32602, "Invalid params: missing tool name", nil)
		return
	}

	args, ok := req.Params["arguments"].(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	// Find handler
	handler, exists := s.handlers[toolName]
	if !exists {
		s.sendError(req.ID, -32601, fmt.Sprintf("Tool not found: %s", toolName), nil)
		return
	}

	// Execute handler
	result, err := handler(ctx, args)
	if err != nil {
		s.logger.Printf("Tool execution error (%s): %v", toolName, err)
		s.sendResult(req.ID, ToolResult{
			Content: []ContentItem{
				{
					Type: "text",
					Text: fmt.Sprintf("Error: %v", err),
				},
			},
			IsError: true,
		})
		return
	}

	// Send success result
	s.sendResult(req.ID, ToolResult{
		Content: []ContentItem{
			{
				Type: "text",
				Text: result,
			},
		},
		IsError: false,
	})
}

// sendResult sends a successful JSON-RPC response
func (s *Server) sendResult(id interface{}, result interface{}) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.sendResponse(response)
}

// sendError sends an error JSON-RPC response
func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.sendResponse(response)
}

// sendResponse writes a JSON-RPC response to stdout
func (s *Server) sendResponse(response JSONRPCResponse) {
	data, err := json.Marshal(response)
	if err != nil {
		s.logger.Printf("Error marshaling response: %v", err)
		return
	}

	fmt.Println(string(data))
}
