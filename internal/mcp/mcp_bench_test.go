package mcp_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Hossiy21/razify/internal/mcp"
)

func BenchmarkMCP_ToolsList(b *testing.B) {
	req := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		in := bytes.NewBufferString(req)
		var out bytes.Buffer
		server := mcp.NewServer("", in, &out, nil, "1.0.5")
		_ = server.Serve()
	}
}

func BenchmarkMCP_SchemaInspect(b *testing.B) {
	tmpDir := b.TempDir()
	exampleFile := filepath.Join(tmpDir, ".env.example")
	_ = os.WriteFile(exampleFile, []byte("PORT=8080 # @type=port\nDB_URL=postgres://localhost/db # @type=url\n"), 0644)

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "schema_inspect",
			"arguments": map[string]any{
				"example_file": filepath.ToSlash(exampleFile),
			},
		},
	}
	data, _ := json.Marshal(req)
	reqLine := string(append(data, '\n'))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		in := bytes.NewBufferString(reqLine)
		var out bytes.Buffer
		server := mcp.NewServer(tmpDir, in, &out, nil, "1.0.5")
		_ = server.Serve()
	}
}

func BenchmarkMCP_EnvValidateConfig(b *testing.B) {
	tmpDir := b.TempDir()
	exampleFile := filepath.Join(tmpDir, ".env.example")
	_ = os.WriteFile(exampleFile, []byte("PORT=8080 # @type=port\nNODE_ENV=production # @enum(dev,stage,production)\n"), 0644)

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "env_validate_config",
			"arguments": map[string]any{
				"example_file": filepath.ToSlash(exampleFile),
				"config": map[string]string{
					"PORT":     "3000",
					"NODE_ENV": "production",
				},
			},
		},
	}
	data, _ := json.Marshal(req)
	reqLine := string(append(data, '\n'))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		in := bytes.NewBufferString(reqLine)
		var out bytes.Buffer
		server := mcp.NewServer(tmpDir, in, &out, nil, "1.0.5")
		_ = server.Serve()
	}
}
