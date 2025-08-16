package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// MCP Server implementation
type MCPServer struct {
	name    string
	version string
	tools   map[string]Tool
}

type Tool struct {
	name        string
	description string
	schema      map[string]interface{}
	handler     func(map[string]interface{}) map[string]interface{}
}

func NewMCPServer(name, version string) *MCPServer {
	return &MCPServer{
		name:    name,
		version: version,
		tools:   make(map[string]Tool),
	}
}

func (s *MCPServer) AddTool(name, description string, schema map[string]interface{}, handler func(map[string]interface{}) map[string]interface{}) {
	s.tools[name] = Tool{
		name:        name,
		description: description,
		schema:      schema,
		handler:     handler,
	}
}

func (s *MCPServer) Serve() error {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var request map[string]interface{}
		if err := json.Unmarshal([]byte(line), &request); err != nil {
			s.sendError(-32700, "Parse error", nil)
			continue
		}

		s.handleRequest(request)
	}
	return scanner.Err()
}

func (s *MCPServer) handleRequest(request map[string]interface{}) {
	method, ok := request["method"].(string)
	if !ok {
		s.sendError(-32600, "Invalid Request", request["id"])
		return
	}

	switch method {
	case "initialize":
		s.handleInitialize(request)
	case "tools/list":
		s.handleToolsList(request)
	case "tools/call":
		s.handleToolsCall(request)
	default:
		s.sendError(-32601, "Method not found", request["id"])
	}
}

func (s *MCPServer) handleInitialize(request map[string]interface{}) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      request["id"],
		"result": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    s.name,
				"version": s.version,
			},
		},
	}
	s.sendResponse(response)
}

func (s *MCPServer) handleToolsList(request map[string]interface{}) {
	var tools []map[string]interface{}
	for _, tool := range s.tools {
		tools = append(tools, map[string]interface{}{
			"name":        tool.name,
			"description": tool.description,
			"inputSchema": tool.schema,
		})
	}

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      request["id"],
		"result": map[string]interface{}{
			"tools": tools,
		},
	}
	s.sendResponse(response)
}

func (s *MCPServer) handleToolsCall(request map[string]interface{}) {
	params, ok := request["params"].(map[string]interface{})
	if !ok {
		s.sendError(-32602, "Invalid params", request["id"])
		return
	}

	name, ok := params["name"].(string)
	if !ok {
		s.sendError(-32602, "Tool name required", request["id"])
		return
	}

	tool, exists := s.tools[name]
	if !exists {
		s.sendError(-32602, "Tool not found", request["id"])
		return
	}

	args, _ := params["arguments"].(map[string]interface{})
	result := tool.handler(args)

	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      request["id"],
		"result":  result,
	}
	s.sendResponse(response)
}

func (s *MCPServer) sendResponse(response map[string]interface{}) {
	data, err := json.Marshal(response)
	if err != nil {
		logger.Printf("Failed to marshal response: %s", err)
		fmt.Println(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"}}`)
		return
	}
	fmt.Println(string(data))
}

func (s *MCPServer) sendError(code int, message string, id interface{}) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	s.sendResponse(response)
}
