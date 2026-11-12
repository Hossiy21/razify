package mcp_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/mcp"
	"github.com/Hossiy21/razify/internal/schema"
)

func skipIfSymlinkFails(t *testing.T, err error) {
	if err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("skipping symlink test on Windows due to OS privilege limitations: %v", err)
		}
		t.Fatalf("failed to create symlink: %v", err)
	}
}

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

	reqJSON := `{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"schema_inspect","arguments":{"example_file":".env.example"}}}` + "\n"
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
				"example_file": ".env.example",
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
				"env_file":     ".env",
				"example_file": ".env.example",
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
				"env_file":     ".env",
				"example_file": ".env.example",
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
	server := mcp.NewServer(t.TempDir(), in, &out, &errBuf, "1.0.5")
	_ = server.Serve()

	var resp mcp.Response
	_ = json.Unmarshal(out.Bytes(), &resp)

	if resp.Error == nil || resp.Error.Code != mcp.CodeMethodNotFound {
		t.Fatalf("expected MethodNotFound error (-32601), got: %+v", resp.Error)
	}
}

// Item 1, 2, 7: Fail-closed path confinement tests across all path edge cases
func TestMCP_PathConfinement_FailClosedCases(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a subdirectory that is a directory inside workspace
	subDir := filepath.Join(tmpDir, "sub")
	_ = os.MkdirAll(subDir, 0755)

	// Create a regular directory named .env (must be rejected as not a regular file)
	dirAsEnv := filepath.Join(tmpDir, "directory.env")
	_ = os.MkdirAll(dirAsEnv, 0755)

	testCases := []struct {
		name       string
		pathArg    string
		wantErrSub string
	}{
		// Windows rooted paths
		{name: "rooted path with forward slash", pathArg: "/etc/passwd", wantErrSub: "rooted path"},
		{name: "rooted path with backslash", pathArg: `\windows\win.ini`, wantErrSub: "rooted path"},

		// UNC paths
		{name: "UNC path with backslashes", pathArg: `\\server\share\.env`, wantErrSub: "UNC path"},
		{name: "UNC path with forward slashes", pathArg: `//server/share/.env`, wantErrSub: "UNC path"},

		// Windows drive-relative and absolute drive paths
		{name: "drive-relative path", pathArg: "C:.env", wantErrSub: "':' is prohibited"},
		{name: "absolute drive path with backslash", pathArg: `C:\secrets\.env`, wantErrSub: "':' is prohibited"},
		{name: "absolute drive path with forward slash", pathArg: "D:/.env", wantErrSub: "':' is prohibited"},

		// Alternate data streams (reject ':' anywhere)
		{name: "alternate data stream .env:stream", pathArg: ".env:stream", wantErrSub: "':' is prohibited"},
		{name: "alternate data stream DATA", pathArg: ".env::$DATA", wantErrSub: "':' is prohibited"},

		// Windows reserved device names
		{name: "reserved name NUL", pathArg: "NUL", wantErrSub: "reserved Windows device name"},
		{name: "reserved name nul lowercase", pathArg: "nul", wantErrSub: "reserved Windows device name"},
		{name: "reserved name CON", pathArg: "CON", wantErrSub: "reserved Windows device name"},
		{name: "reserved name AUX", pathArg: "AUX", wantErrSub: "reserved Windows device name"},
		{name: "reserved name COM1", pathArg: "COM1", wantErrSub: "reserved Windows device name"},
		{name: "reserved name LPT1", pathArg: "LPT1", wantErrSub: "reserved Windows device name"},
		{name: "reserved name with env extension", pathArg: "NUL.env", wantErrSub: "reserved Windows device name"},

		// Null bytes
		{name: "path containing null byte", pathArg: ".env\x00secret", wantErrSub: "null bytes"},

		// Directory traversal
		{name: "parent traversal", pathArg: "../outside.env", wantErrSub: "directory traversal"},
		{name: "nested parent traversal", pathArg: "sub/../../outside.env", wantErrSub: "directory traversal"},
		{name: "exact ..", pathArg: "..", wantErrSub: "directory traversal"},

		// Basename restrictions
		{name: "disallowed basename id_rsa", pathArg: "id_rsa", wantErrSub: "file basename"},
		{name: "disallowed basename secrets.json", pathArg: "secrets.json", wantErrSub: "file basename"},
		{name: "disallowed basename config.yaml", pathArg: "config.yaml", wantErrSub: "file basename"},
		{name: "disallowed basename foo.txt", pathArg: "foo.txt", wantErrSub: "file basename"},
		{name: "disallowed basename .env.", pathArg: ".env.", wantErrSub: "file basename"},

		// Not a regular file
		{name: "target is a directory", pathArg: "directory.env", wantErrSub: "not a regular file"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mcp.ResolveConfinedPath(tmpDir, tc.pathArg, ".env.example")
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.pathArg)
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Errorf("expected error containing %q, got: %v", tc.wantErrSub, err)
			}
		})
	}

	// Allowed basenames test (including ..env)
	validFiles := []string{".env", ".env.example", ".env.local", ".env.production", "local.env", "staging.env", "..env"}
	for _, vf := range validFiles {
		t.Run("allowed_"+vf, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, vf)
			_ = os.WriteFile(filePath, []byte("TEST=1\n"), 0644)
			resolved, err := mcp.ResolveConfinedPath(tmpDir, vf, ".env.example")
			if err != nil {
				t.Fatalf("expected allowed basename %q to succeed, got error: %v", vf, err)
			}
			if !strings.HasSuffix(filepath.ToSlash(resolved), vf) {
				t.Errorf("expected resolved path to end with %s, got %s", vf, resolved)
			}
		})
	}

	// Non-existent target resolving longest existing ancestor
	nonExistentTarget := "sub/nonexistent.env"
	resolved, err := mcp.ResolveConfinedPath(tmpDir, nonExistentTarget, ".env.example")
	if err != nil {
		t.Fatalf("expected non-existent target with valid ancestor to resolve, got error: %v", err)
	}
	expectedSuffix := filepath.Join("sub", "nonexistent.env")
	if !strings.HasSuffix(resolved, expectedSuffix) {
		t.Errorf("expected resolved path to end with %s, got %s", expectedSuffix, resolved)
	}
}

// Item 1 & 7: Symlink bypasses, chains, in-workspace .git/config target, and dangling symlinks
func TestMCP_Symlinks_BypassesAndChains(t *testing.T) {
	tmpDir := t.TempDir()
	outsideDir := t.TempDir()

	outsideSecret := filepath.Join(outsideDir, "secret_outside.env")
	_ = os.WriteFile(outsideSecret, []byte("OUTSIDE=secret\n"), 0600)

	// In-workspace sensitive non-env file
	gitDir := filepath.Join(tmpDir, ".git")
	_ = os.MkdirAll(gitDir, 0755)
	gitConfig := filepath.Join(gitDir, "config")
	_ = os.WriteFile(gitConfig, []byte("[core]\n\trepositoryformatversion = 0\n"), 0644)

	// 1. Symlink named .env pointing to .git/config (CRITICAL BYPASS TEST)
	symlinkToGitConfig := filepath.Join(tmpDir, "symlink_git.env")
	err := os.Symlink(gitConfig, symlinkToGitConfig)
	skipIfSymlinkFails(t, err)

	_, err = mcp.ResolveConfinedPath(tmpDir, "symlink_git.env", ".env.example")
	if err == nil {
		t.Fatalf("SECURITY VIOLATION: symlink pointing to .git/config was allowed!")
	}
	if !strings.Contains(err.Error(), "symlink") && !strings.Contains(err.Error(), "basename") {
		t.Errorf("expected symlink or basename rejection, got: %v", err)
	}

	// 2. Symlink pointing outside workspace
	symlinkEscape := filepath.Join(tmpDir, "symlink_escape.env")
	_ = os.Symlink(outsideSecret, symlinkEscape)
	_, err = mcp.ResolveConfinedPath(tmpDir, "symlink_escape.env", ".env.example")
	if err == nil {
		t.Fatalf("expected symlink escape to be rejected, but it succeeded")
	}

	// 3. Symlink chain: link1.env -> link2.env -> outsideSecret
	link2 := filepath.Join(tmpDir, "link2.env")
	_ = os.Symlink(outsideSecret, link2)
	link1 := filepath.Join(tmpDir, "link1.env")
	_ = os.Symlink(link2, link1)
	_, err = mcp.ResolveConfinedPath(tmpDir, "link1.env", ".env.example")
	if err == nil {
		t.Fatalf("expected symlink chain to outside to be rejected, but it succeeded")
	}

	// 4. Dangling symlink
	danglingLink := filepath.Join(tmpDir, "dangling.env")
	_ = os.Symlink(filepath.Join(tmpDir, "does_not_exist.env"), danglingLink)
	_, err = mcp.ResolveConfinedPath(tmpDir, "dangling.env", ".env.example")
	if err == nil {
		t.Fatalf("expected dangling symlink to be rejected, but it succeeded")
	}

	// 5. Symlinked parent directory pointing outside with non-existent child
	symlinkedParent := filepath.Join(tmpDir, "parent_link")
	_ = os.Symlink(outsideDir, symlinkedParent)
	_, err = mcp.ResolveConfinedPath(tmpDir, "parent_link/child.env", ".env.example")
	if err == nil {
		t.Fatalf("expected symlinked parent directory with child to be rejected, but it succeeded")
	}
}

// Item 2, 4, 7: Workspace validation tests (refuse /, drive roots, home directory, ancestor of home)
func TestMCP_WorkspaceValidation(t *testing.T) {
	// 1. Refuse empty workspace
	if _, err := mcp.ValidateWorkspaceDir(""); err == nil {
		t.Errorf("expected empty workspace to be rejected")
	}
	if _, err := mcp.ValidateWorkspaceDir("   "); err == nil {
		t.Errorf("expected whitespace workspace to be rejected")
	}

	// 2. Refuse root directory
	if _, err := mcp.ValidateWorkspaceDir("/"); err == nil {
		t.Errorf("expected / root directory to be rejected")
	}

	// 3. Refuse Windows drive roots
	if _, err := mcp.ValidateWorkspaceDir(`C:\`); err == nil {
		t.Errorf("expected C:\\ drive root to be rejected")
	}
	if _, err := mcp.ValidateWorkspaceDir("C:"); err == nil {
		t.Errorf("expected C: drive root to be rejected")
	}

	// 4. Refuse user home directory and ancestor of home
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		if _, err := mcp.ValidateWorkspaceDir(homeDir); err == nil {
			t.Errorf("expected user home directory to be rejected")
		}

		homeParent := filepath.Dir(homeDir)
		if homeParent != "" && homeParent != homeDir {
			if _, err := mcp.ValidateWorkspaceDir(homeParent); err == nil {
				t.Errorf("expected home directory ancestor %q to be rejected", homeParent)
			}
		}
	}

	// 5. Valid temporary directory succeeds
	tmpDir := t.TempDir()
	validWs, err := mcp.ValidateWorkspaceDir(tmpDir)
	if err != nil {
		t.Fatalf("expected valid tmp directory to succeed, got: %v", err)
	}
	if validWs == "" {
		t.Errorf("expected non-empty canonical workspace path")
	}
}

// Item 3: File open handle & 1 MB cap enforcement
func TestMCP_OpenConfinedEnvFile_1MBCap(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Regular valid file under 1MB opens and stats cleanly
	validFile := filepath.Join(tmpDir, ".env")
	_ = os.WriteFile(validFile, []byte("TEST=1\n"), 0644)
	f, path, err := mcp.OpenConfinedEnvFile(tmpDir, ".env", ".env.example")
	if err != nil {
		t.Fatalf("expected OpenConfinedEnvFile to succeed, got: %v", err)
	}
	f.Close()
	if path == "" {
		t.Errorf("expected non-empty resolved path")
	}

	// 2. Oversized file (> 1MB) must be rejected
	largeFile := filepath.Join(tmpDir, "large.env")
	largeData := bytes.Repeat([]byte("A=1\n"), 300000) // ~1.2 MB
	_ = os.WriteFile(largeFile, largeData, 0644)

	_, _, err = mcp.OpenConfinedEnvFile(tmpDir, "large.env", ".env.example")
	if err == nil {
		t.Fatalf("expected file > 1MB to be rejected, but it succeeded")
	}
	if !strings.Contains(err.Error(), "exceeds maximum allowed size") {
		t.Errorf("expected 1MB cap error, got: %v", err)
	}
}

// Item 7: /work vs /work-evil prefix bypass test
func TestMCP_WorkVsWorkEvil(t *testing.T) {
	parentDir := t.TempDir()
	workDir := filepath.Join(parentDir, "work")
	workEvilDir := filepath.Join(parentDir, "work-evil")
	_ = os.MkdirAll(workDir, 0755)
	_ = os.MkdirAll(workEvilDir, 0755)

	evilFile := filepath.Join(workEvilDir, ".env")
	_ = os.WriteFile(evilFile, []byte("EVIL=true\n"), 0644)

	// Attempt to access /work-evil/.env with workspace set to /work
	relPath, _ := filepath.Rel(workDir, evilFile)
	_, err := mcp.ResolveConfinedPath(workDir, relPath, ".env.example")
	if err == nil {
		t.Fatalf("SECURITY VIOLATION: /work-evil was treated as within /work!")
	}
}

// Item 3: Parser structured ParseError & SanitizeParseError canary tests
func TestMCP_ParserAndSanitizeParseError_Canaries(t *testing.T) {
	canarySecret := "CANARY_SECRET_RAW_TOKEN_99999"

	// 1. Error without "=" (missing equals on non-empty non-comment line)
	inputWithoutEq := "PORT=8080\n" + canarySecret + "\nNODE_ENV=dev\n"
	_, err := schema.ParseEnvWithMetadataReader(strings.NewReader(inputWithoutEq))
	if err == nil {
		t.Fatalf("expected parse error for line without equals")
	}
	sanitized := mcp.SanitizeParseError(err)
	if strings.Contains(sanitized, canarySecret) {
		t.Fatalf("SECURITY VIOLATION: canary secret leaked in sanitized error (without =): %s", sanitized)
	}
	if !strings.Contains(sanitized, "MISSING_EQUALS") {
		t.Errorf("expected structured MISSING_EQUALS code in sanitized error, got: %s", sanitized)
	}

	// 2. Error with "=" (unterminated quote on line with equals)
	canaryQuoted := "CANARY_SECRET_QUOTED_VAL_12345"
	inputWithEq := `PORT=8080` + "\n" + `API_KEY="` + canaryQuoted + "\n" + `OTHER=val`
	_, err = schema.ParseEnvWithMetadataReader(strings.NewReader(inputWithEq))
	if err == nil {
		t.Fatalf("expected parse error for unterminated quote")
	}
	sanitized = mcp.SanitizeParseError(err)
	if strings.Contains(sanitized, canaryQuoted) {
		t.Fatalf("SECURITY VIOLATION: canary secret leaked in sanitized error (with =): %s", sanitized)
	}
	if strings.Contains(sanitized, "=") {
		t.Fatalf("SECURITY VIOLATION: equals sign leaked in sanitized error: %s", sanitized)
	}
	if !strings.Contains(sanitized, "UNTERMINATED_QUOTE") {
		t.Errorf("expected UNTERMINATED_QUOTE code in sanitized error, got: %s", sanitized)
	}

	// 3. Multi-line unterminated quote containing canary
	canaryMultiline := "CANARY_MULTILINE_SECRET_abc123"
	inputMultiline := "PORT=8080\nMULTI_LINE=\"start\n" + canaryMultiline + "\nstill_no_closing_quote\n"
	_, err = schema.ParseEnvWithMetadataReader(strings.NewReader(inputMultiline))
	if err == nil {
		t.Fatalf("expected parse error for multi-line unterminated quote")
	}
	sanitized = mcp.SanitizeParseError(err)
	if strings.Contains(sanitized, canaryMultiline) {
		t.Fatalf("SECURITY VIOLATION: canary secret leaked in multi-line error: %s", sanitized)
	}
	if !strings.Contains(sanitized, "UNTERMINATED_QUOTE") {
		t.Errorf("expected UNTERMINATED_QUOTE in multi-line error, got: %s", sanitized)
	}

	// 4. Unknown error maps to fixed constant message
	canaryUnknown := "CANARY_RAW_UNKNOWN_DRIVER_ERROR_5555"
	unknownErr := errors.New("raw database error: " + canaryUnknown)
	sanitized = mcp.SanitizeParseError(unknownErr)
	if strings.Contains(sanitized, canaryUnknown) {
		t.Fatalf("SECURITY VIOLATION: canary secret leaked from unknown error: %s", sanitized)
	}
	if sanitized != mcp.UnknownParseErrorMessage {
		t.Errorf("expected fixed constant message %q, got: %q", mcp.UnknownParseErrorMessage, sanitized)
	}
}

// Item 4, 6: IsSecretPlaceholder per-charset entropy, URL-userinfo, and extended key names
func TestMCP_IsSecretPlaceholder_EntropyAndHeuristics(t *testing.T) {
	// 1. Hex per-charset entropy (len >= 32, entropy >= 3.0)
	highEntropyHex := "d41d8cd98f00b204e9800998ecf8427e" // 32 hex chars, entropy ~3.5
	if !mcp.IsSecretPlaceholder("CONFIG_VAL", highEntropyHex, nil) {
		t.Errorf("expected high-entropy hex (32 chars) to be flagged as secret placeholder")
	}
	lowEntropyHex := "00000000000000000000000000000000" // 32 chars of 0, entropy 0.0
	if mcp.IsSecretPlaceholder("CONFIG_VAL", lowEntropyHex, nil) {
		t.Errorf("expected low-entropy hex to NOT be flagged as secret")
	}
	repeatingHex := "01010101010101010101010101010101" // entropy 1.0 < 3.0
	if mcp.IsSecretPlaceholder("CONFIG_VAL", repeatingHex, nil) {
		t.Errorf("expected repeating hex with entropy < 3.0 to NOT be flagged as secret")
	}
	shortHex := "d41d8cd98f00b204" // 16 chars < 32
	if mcp.IsSecretPlaceholder("CONFIG_VAL", shortHex, nil) {
		t.Errorf("expected short hex (<32 chars) to NOT be flagged under hex entropy rule")
	}

	// 2. Base64-like per-charset entropy (len >= 32 entropy >= 4.0; 20 <= len < 32 entropy >= 4.3)
	highEntropyBase64Long := "xK9mQ4vL2pW8zN7bT1jR5aC3dE6gH0yU" // 32 chars, entropy > 4.5
	if !mcp.IsSecretPlaceholder("CONFIG_VAL", highEntropyBase64Long, nil) {
		t.Errorf("expected high-entropy base64 (32 chars) to be flagged as secret")
	}
	highEntropyBase64Short := "xK9mQ4vL2pW8zN7bT1jR5aC3" // 24 chars, high entropy > 4.3
	if !mcp.IsSecretPlaceholder("CONFIG_VAL", highEntropyBase64Short, nil) {
		t.Errorf("expected high-entropy base64 (24 chars) to be flagged as secret")
	}
	regularIdentifier := "my_long_normal_variable_name" // 29 chars, English words, entropy < 3.8
	if mcp.IsSecretPlaceholder("CONFIG_VAL", regularIdentifier, nil) {
		t.Errorf("expected regular English identifier to NOT be flagged as secret: %s", regularIdentifier)
	}

	// 3. URL userinfo pattern
	postgresURL := "postgres://admin:super_secret_pw@localhost:5432/mydb"
	if !mcp.IsSecretPlaceholder("DB_URL", postgresURL, nil) {
		t.Errorf("expected URL with userinfo credentials to be flagged as secret")
	}
	redisURL := "redis://:mypassword@cache.internal:6379"
	if !mcp.IsSecretPlaceholder("CACHE_URL", redisURL, nil) {
		t.Errorf("expected Redis URL with userinfo to be flagged as secret")
	}
	publicURL := "https://api.razify.dev/v1/health"
	if mcp.IsSecretPlaceholder("HEALTH_URL", publicURL, nil) {
		t.Errorf("expected public URL without userinfo to NOT be flagged as secret")
	}

	// 4. Extended sensitive key names
	extendedKeys := []string{
		"DB_DSN", "APP_CONNECTION_STRING", "PASSWORD_SALT", "TOKEN_SIGNING",
		"AUTH_BEARER", "SESSION_COOKIE", "USER_SESSION", "AUTH_JWT",
		"SLACK_WEBHOOK", "SERVER_CERT",
	}
	for _, key := range extendedKeys {
		if !mcp.IsSecretPlaceholder(key, "arbitrary_val", nil) {
			t.Errorf("expected extended key name %q to be recognized as sensitive", key)
		}
	}
}

// Item 4 & 5: Deny-by-default output formatting
func TestMCP_DenyByDefault_OutputFormatting(t *testing.T) {
	testCases := []struct {
		name       string
		key        string
		val        string
		tags       map[string]string
		wantOutput string
	}{
		// Allowed types with valid values
		{name: "valid port", key: "PORT", val: "8080", tags: map[string]string{"type": "port"}, wantOutput: "8080"},
		{name: "valid bool true", key: "DEBUG", val: "true", tags: map[string]string{"type": "bool"}, wantOutput: "true"},
		{name: "valid bool 1", key: "ENABLE_FEATURE", val: "1", tags: map[string]string{"type": "bool"}, wantOutput: "1"},
		{name: "valid int", key: "MAX_CONNS", val: "100", tags: map[string]string{"type": "int"}, wantOutput: "100"},
		{name: "valid enum", key: "ENVIRONMENT", val: "staging", tags: map[string]string{"enum": "dev,staging,prod"}, wantOutput: "staging"},
		{name: "valid range", key: "WORKERS", val: "16", tags: map[string]string{"range": "1-64"}, wantOutput: "16"},

		// Allowed types with invalid values -> must be masked
		{name: "invalid port string", key: "PORT", val: "not_a_port", tags: map[string]string{"type": "port"}, wantOutput: "[masked]"},
		{name: "port out of range", key: "PORT", val: "99999", tags: map[string]string{"type": "port"}, wantOutput: "[masked]"},
		{name: "invalid int string", key: "MAX_CONNS", val: "abc", tags: map[string]string{"type": "int"}, wantOutput: "[masked]"},
		{name: "enum mismatch", key: "ENVIRONMENT", val: "other", tags: map[string]string{"enum": "dev,staging,prod"}, wantOutput: "[masked]"},

		// Sensitive keys even with allowed types -> must be masked
		{name: "sensitive key with int type", key: "SECRET_PIN", val: "1234", tags: map[string]string{"type": "int"}, wantOutput: "[masked]"},

		// Denied by default (types other than port/bool/int/enum/range)
		{name: "string type denied by default", key: "APP_NAME", val: "razify-backend", tags: map[string]string{"type": "string"}, wantOutput: "[masked]"},
		{name: "no tags denied by default", key: "CUSTOM_VAL", val: "public_value", tags: nil, wantOutput: "[masked]"},
		{name: "secret type denied by default", key: "API_KEY", val: "sk_test_12345", tags: map[string]string{"type": "secret"}, wantOutput: "[masked]"},
		{name: "url with userinfo denied by default", key: "DATABASE_URL", val: "postgres://user:pass@localhost:5432/db", tags: map[string]string{"type": "url"}, wantOutput: "[masked]"},

		// Empty string
		{name: "empty value preserved as empty", key: "EMPTY_VAL", val: "", tags: map[string]string{"type": "string"}, wantOutput: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := mcp.SafeMCPValue(tc.key, tc.val, tc.tags)
			if got != tc.wantOutput {
				t.Errorf("SafeMCPValue(%q, %q, %v): expected %q, got %q", tc.key, tc.val, tc.tags, tc.wantOutput, got)
			}
		})
	}
}

// Item 5: Canary secrets absent from every tool response AND assert each call succeeded
func TestMCP_CanarySecretsAbsentFromAllToolResponses(t *testing.T) {
	tmpDir := t.TempDir()

	canaries := []string{
		"CANARY_SCHEMA_SECRET_9876543210",
		"CANARY_CONFIG_SECRET_1122334455",
		"CANARY_DIFF_SECRET_5566778899",
		"CANARY_AUDIT_SECRET_7788990011",
		"CANARY_URL_USERINFO_PASSWORD_667788",
	}

	exampleFile := filepath.Join(tmpDir, ".env.example")
	envFile := filepath.Join(tmpDir, ".env")

	exampleContent := `# Safe public variables
PORT=8080 # @type=port @required=true
DEBUG=true # @type=bool
NODE_ENV=production # @enum(development,staging,production)

# Sensitive and default-masked variables
DATABASE_URL=postgres://admin:` + canaries[4] + `@db:5432/mydb # @type=url
APP_SECRET=` + canaries[0] + ` # @type=secret
UNTRACKED_VAR=` + canaries[2] + `
AUDIT_TARGET=placeholder # @required=true
`
	_ = os.WriteFile(exampleFile, []byte(exampleContent), 0644)

	envContent := `PORT=8080
DEBUG=true
NODE_ENV=production
DATABASE_URL=postgres://admin:` + canaries[4] + `@db:5432/mydb
APP_SECRET=` + canaries[0] + `
UNTRACKED_VAR=` + canaries[2] + `
AUDIT_TARGET=` + "sk_live_" + "000000000000000000000000" + `
`
	_ = os.WriteFile(envFile, []byte(envContent), 0600)

	assertNoCanaries := func(toolName, responseText string) {
		for _, c := range canaries {
			if strings.Contains(responseText, c) {
				t.Fatalf("SECURITY VIOLATION in %s: canary secret %q was found in tool response:\n%s", toolName, c, responseText)
			}
		}
	}

	// 1. Test schema_inspect
	inspectRes, err := mcp.HandleSchemaInspect(json.RawMessage(`{"example_file":".env.example"}`), tmpDir)
	if err != nil {
		t.Fatalf("HandleSchemaInspect failed: %v", err)
	}
	if inspectRes.IsError {
		t.Fatalf("HandleSchemaInspect reported error: %+v", inspectRes)
	}
	if len(inspectRes.Content) == 0 {
		t.Fatalf("HandleSchemaInspect returned empty content")
	}
	assertNoCanaries("schema_inspect", inspectRes.Content[0].Text)
	// Verify allowed types are present while others are masked
	if !strings.Contains(inspectRes.Content[0].Text, "8080") {
		t.Errorf("schema_inspect missing valid port 8080")
	}
	if !strings.Contains(inspectRes.Content[0].Text, "[masked]") {
		t.Errorf("schema_inspect expected deny-by-default [masked] for secrets/strings")
	}

	// 2. Test env_validate_config with candidate config containing canary secret
	validateArgs := map[string]any{
		"example_file": ".env.example",
		"config": map[string]string{
			"PORT":         "8080",
			"DEBUG":        "true",
			"NODE_ENV":     "production",
			"AUDIT_TARGET": canaries[1], // Candidate secret value
		},
	}
	validateJSON, _ := json.Marshal(validateArgs)
	validateRes, err := mcp.HandleEnvValidateConfig(validateJSON, tmpDir)
	if err != nil {
		t.Fatalf("HandleEnvValidateConfig failed: %v", err)
	}
	if validateRes.IsError {
		t.Fatalf("HandleEnvValidateConfig reported error: %+v", validateRes)
	}
	if len(validateRes.Content) == 0 {
		t.Fatalf("HandleEnvValidateConfig returned empty content")
	}
	assertNoCanaries("env_validate_config", validateRes.Content[0].Text)

	// 3. Test env_diff_keys
	diffRes, err := mcp.HandleEnvDiffKeys(json.RawMessage(`{"env_file":".env","example_file":".env.example"}`), tmpDir)
	if err != nil {
		t.Fatalf("HandleEnvDiffKeys failed: %v", err)
	}
	if diffRes.IsError {
		t.Fatalf("HandleEnvDiffKeys reported error: %+v", diffRes)
	}
	if len(diffRes.Content) == 0 {
		t.Fatalf("HandleEnvDiffKeys returned empty content")
	}
	assertNoCanaries("env_diff_keys", diffRes.Content[0].Text)

	// 4. Test env_health_audit
	auditRes, err := mcp.HandleEnvHealthAudit(json.RawMessage(`{"env_file":".env","example_file":".env.example"}`), tmpDir)
	if err != nil {
		t.Fatalf("HandleEnvHealthAudit failed: %v", err)
	}
	if auditRes.IsError {
		t.Fatalf("HandleEnvHealthAudit reported error: %+v", auditRes)
	}
	if len(auditRes.Content) == 0 {
		t.Fatalf("HandleEnvHealthAudit returned empty content")
	}
	assertNoCanaries("env_health_audit", auditRes.Content[0].Text)
}
