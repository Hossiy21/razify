package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Server implements an MCP (Model Context Protocol) JSON-RPC 2.0 server over standard I/O.
type Server struct {
	WorkspaceDir string
	In           io.Reader
	Out          io.Writer
	Err          io.Writer
	Version      string
}

// NewServer creates a new MCP server.
func NewServer(workspaceDir string, in io.Reader, out io.Writer, errLog io.Writer, version string) *Server {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	if errLog == nil {
		errLog = os.Stderr
	}
	if version == "" {
		version = "1.0.5"
	}
	return &Server{
		WorkspaceDir: workspaceDir,
		In:           in,
		Out:          out,
		Err:          errLog,
		Version:      version,
	}
}

// Serve reads JSON-RPC 2.0 messages from In, handles them, and writes responses to Out.
func (s *Server) Serve() error {
	scanner := bufio.NewScanner(s.In)
	// Buffer up to 10MB to accommodate large tool payloads
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.sendError(nil, CodeParseError, "Parse error: invalid JSON", err.Error())
			continue
		}

		// Handle notifications (no ID or null ID, starts with notifications/ or no response expected)
		if req.ID == nil && strings.HasPrefix(req.Method, "notifications/") {
			// Acknowledge notification internally without sending a response
			continue
		}

		s.handleRequest(req)
	}

	return scanner.Err()
}

func (s *Server) handleRequest(req Request) {
	switch req.Method {
	case "initialize":
		res := InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCapabilities{
				Tools: &ToolsCapability{ListChanged: false},
			},
			ServerInfo: ServerInfo{
				Name:    "razify-mcp",
				Version: s.Version,
			},
		}
		s.sendResult(req.ID, res)

	case "ping":
		s.sendResult(req.ID, map[string]any{})

	case "tools/list":
		s.sendResult(req.ID, map[string]any{
			"tools": ListTools(),
		})

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.sendError(req.ID, CodeInvalidParams, "Invalid params for tools/call", err.Error())
			return
		}

		result, err := s.dispatchTool(params.Name, params.Arguments)
		if err != nil {
			s.sendError(req.ID, CodeInternalError, "Tool execution error", err.Error())
			return
		}

		s.sendResult(req.ID, result)

	default:
		s.sendError(req.ID, CodeMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), nil)
	}
}

func (s *Server) dispatchTool(name string, args json.RawMessage) (CallToolResult, error) {
	switch name {
	case "schema_inspect":
		return HandleSchemaInspect(args, s.WorkspaceDir)
	case "env_validate_config":
		return HandleEnvValidateConfig(args, s.WorkspaceDir)
	case "env_diff_keys":
		return HandleEnvDiffKeys(args, s.WorkspaceDir)
	case "env_health_audit":
		return HandleEnvHealthAudit(args, s.WorkspaceDir)
	default:
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Unknown tool: %s", name)}},
			IsError: true,
		}, nil
	}
}

func (s *Server) sendResult(id any, result any) {
	resp := Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Result:  result,
	}
	s.writeResponse(resp)
}

func (s *Server) sendError(id any, code int, message string, data any) {
	resp := Response{
		JSONRPC: JSONRPCVersion,
		ID:      id,
		Error:   NewRPCError(code, message, data),
	}
	s.writeResponse(resp)
}

func (s *Server) writeResponse(resp Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(s.Err, "Error encoding response: %v\n", err)
		return
	}
	_, _ = s.Out.Write(data)
	_, _ = s.Out.Write([]byte("\n"))
}
