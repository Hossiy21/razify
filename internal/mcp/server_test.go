package mcp_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/mcp"
)

func TestMCPServer_InitializeAndToolsList(t *testing.T) {
	tmpDir := t.TempDir()

	in := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}` + "\n" +
		`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n")

	var out, errBuf bytes.Buffer
	server := mcp.NewServer(tmpDir, in, &out, &errBuf, "1.0.5")

	if err := server.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 responses (initialize and tools/list), got %d: %s", len(lines), out.String())
	}

	// 1. Validate initialize response
	var initResp mcp.Response
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("failed to parse initialize response: %v", err)
	}
	if initResp.ID != float64(1) {
		t.Errorf("expected id 1, got %v", initResp.ID)
	}

	// 2. Validate tools/list response
	var listResp mcp.Response
	if err := json.Unmarshal([]byte(lines[1]), &listResp); err != nil {
		t.Fatalf("failed to parse tools/list response: %v", err)
	}
	if listResp.ID != float64(2) {
		t.Errorf("expected id 2, got %v", listResp.ID)
	}

	resultMap, ok := listResp.Result.(map[string]any)
	if !ok {
		t.Fatalf("tools/list result is not an object: %v", listResp.Result)
	}
	toolsList, ok := resultMap["tools"].([]any)
	if !ok || len(toolsList) != 4 {
		t.Fatalf("expected 4 tools declared, got %d", len(toolsList))
	}
}

func TestMCPServer_SchemaInspectTool(t *testing.T) {
	tmpDir := t.TempDir()
	exampleFile := filepath.Join(tmpDir, ".env.example")
	exampleContent := `# Application Settings
PORT=8080 # @type=port @required=true
NODE_ENV=production # @enum(development,staging,production)

# Database
DATABASE_URL=postgres://localhost:5432/db # @type=url
`
	_ = os.WriteFile(exampleFile, []byte(exampleContent), 0644)

	reqJSON := `{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"schema_inspect","arguments":{"example_file":"` + filepath.ToSlash(exampleFile) + `"}}}` + "\n"
	in := bytes.NewBufferString(reqJSON)

	var out, errBuf bytes.Buffer
	server := mcp.NewServer(tmpDir, in, &out, &errBuf, "1.0.5")
	if err := server.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp mcp.Response
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("tool call returned error: %v", resp.Error)
	}

	resMap, _ := resp.Result.(map[string]any)
	contentList, _ := resMap["content"].([]any)
	firstContent, _ := contentList[0].(map[string]any)
	text := firstContent["text"].(string)

	if !strings.Contains(text, "PORT") || !strings.Contains(text, "NODE_ENV") || !strings.Contains(text, "DATABASE_URL") {
		t.Errorf("schema inspect output missing variables: %s", text)
	}
	if !strings.Contains(text, "port") {
		t.Errorf("schema inspect output missing @type=port: %s", text)
	}
}

func TestMCPServer_EnvValidateConfigTool(t *testing.T) {
	tmpDir := t.TempDir()
	exampleFile := filepath.Join(tmpDir, ".env.example")
	exampleContent := `PORT=8080 # @type=port @required=true
APP_SECRET=placeholder # @type=secret
`
	_ = os.WriteFile(exampleFile, []byte(exampleContent), 0644)

	// Candidate config with invalid port and a simulated mock secret
	mockSecret := "sk_live_" + "000000000000000000000000"
	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      20,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "env_validate_config",
			"arguments": map[string]any{
				"example_file": filepath.ToSlash(exampleFile),
				"config": map[string]string{
					"PORT":       "999999", // Invalid port
					"APP_SECRET": mockSecret,
				},
			},
		},
	}
	reqBytes, _ := json.Marshal(req)
	in := bytes.NewBuffer(append(reqBytes, '\n'))

	var out, errBuf bytes.Buffer
	server := mcp.NewServer(tmpDir, in, &out, &errBuf, "1.0.5")
	if err := server.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp mcp.Response
	_ = json.Unmarshal(out.Bytes(), &resp)

	resMap, _ := resp.Result.(map[string]any)
	contentList, _ := resMap["content"].([]any)
	firstContent, _ := contentList[0].(map[string]any)
	text := firstContent["text"].(string)

	if !strings.Contains(text, "valid network port") {
		t.Errorf("expected port out of range validation error: %s", text)
	}
	if !strings.Contains(text, "SECRET_LEAK") {
		t.Errorf("expected secret leak detection in submitted candidate config: %s", text)
	}
	// Verify raw secret is MASKED in response
	if strings.Contains(text, mockSecret) {
		t.Fatalf("SECURITY VIOLATION: submitted secret was not masked in validation response:\n%s", text)
	}
}

func TestMCPServer_EnvDiffKeysTool(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	exampleFile := filepath.Join(tmpDir, ".env.example")

	_ = os.WriteFile(envFile, []byte("PORT=3000\nDATABASE_PASSWORD=real_secret_value\nUNTRACKED_KEY=val\n"), 0600)
	_ = os.WriteFile(exampleFile, []byte("PORT=3000\nMISSING_SCHEMA_KEY=val\n"), 0644)

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      30,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "env_diff_keys",
			"arguments": map[string]any{
				"env_file":     filepath.ToSlash(envFile),
				"example_file": filepath.ToSlash(exampleFile),
			},
		},
	}
	reqBytes, _ := json.Marshal(req)
	in := bytes.NewBuffer(append(reqBytes, '\n'))

	var out, errBuf bytes.Buffer
	server := mcp.NewServer(tmpDir, in, &out, &errBuf, "1.0.5")
	if err := server.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp mcp.Response
	_ = json.Unmarshal(out.Bytes(), &resp)

	resMap, _ := resp.Result.(map[string]any)
	contentList, _ := resMap["content"].([]any)
	firstContent, _ := contentList[0].(map[string]any)
	text := firstContent["text"].(string)

	// Verify key diffing
	if !strings.Contains(text, "MISSING_SCHEMA_KEY") {
		t.Errorf("missing MISSING_SCHEMA_KEY in diff keys: %s", text)
	}
	if !strings.Contains(text, "UNTRACKED_KEY") {
		t.Errorf("missing UNTRACKED_KEY in diff keys: %s", text)
	}
	// STRICT CHECK: Values must NEVER appear in diff keys
	if strings.Contains(text, "real_secret_value") {
		t.Fatalf("SECURITY VIOLATION: secret value leaked in env_diff_keys output:\n%s", text)
	}
}

func TestMCPServer_EnvHealthAuditTool(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	exampleFile := filepath.Join(tmpDir, ".env.example")

	mockSecret := "sk_live_" + "000000000000000000000000"
	_ = os.WriteFile(envFile, []byte("PORT=8080\nAPI_KEY="+mockSecret+"\n"), 0600)
	_ = os.WriteFile(exampleFile, []byte("PORT=8080 # @required=true\nAPI_KEY= # @required=true\nMISSING_KEY= # @required=true\n"), 0644)

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      40,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "env_health_audit",
			"arguments": map[string]any{
				"env_file":     filepath.ToSlash(envFile),
				"example_file": filepath.ToSlash(exampleFile),
			},
		},
	}
	reqBytes, _ := json.Marshal(req)
	in := bytes.NewBuffer(append(reqBytes, '\n'))

	var out, errBuf bytes.Buffer
	server := mcp.NewServer(tmpDir, in, &out, &errBuf, "1.0.5")
	if err := server.Serve(); err != nil {
		t.Fatalf("Serve failed: %v", err)
	}

	var resp mcp.Response
	_ = json.Unmarshal(out.Bytes(), &resp)

	resMap, _ := resp.Result.(map[string]any)
	contentList, _ := resMap["content"].([]any)
	firstContent, _ := contentList[0].(map[string]any)
	text := firstContent["text"].(string)

	if !strings.Contains(text, "MISSING_KEY") {
		t.Errorf("missing MISSING_KEY in audit: %s", text)
	}
	if !strings.Contains(text, "masked_value") {
		t.Errorf("missing masked_value in secret findings: %s", text)
	}
	if strings.Contains(text, mockSecret) {
		t.Fatalf("SECURITY LEAK: raw secret leaked in audit response:\n%s", text)
	}
}

func TestMCPServer_MethodNotFound(t *testing.T) {
	in := bytes.NewBufferString(`{"jsonrpc":"2.0","id":99,"method":"nonexistent_method"}` + "\n")
	var out, errBuf bytes.Buffer
	server := mcp.NewServer("", in, &out, &errBuf, "1.0.5")
	_ = server.Serve()

	var resp mcp.Response
	_ = json.Unmarshal(out.Bytes(), &resp)

	if resp.Error == nil || resp.Error.Code != mcp.CodeMethodNotFound {
		t.Fatalf("expected MethodNotFound error (-32601), got: %+v", resp.Error)
	}
}
