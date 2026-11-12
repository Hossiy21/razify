package mcp

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Hossiy21/razify/internal/scanner"
)

var sensitiveKeyRegex = regexp.MustCompile(`(?i)(secret|password|passwd|pwd|token|auth|credential|api_key|apikey|private_key|privatekey|_key$|access_key)`)

// ResolveConfinedPath resolves reqPath strictly within workspaceDir.
// It rejects absolute paths, directory traversal attempts (../), and external symlinks.
func ResolveConfinedPath(workspaceDir, reqPath, defaultFile string) (string, error) {
	if workspaceDir == "" {
		workspaceDir = "."
	}

	if reqPath == "" {
		reqPath = defaultFile
	}

	// Reject absolute paths
	if filepath.IsAbs(reqPath) {
		return "", fmt.Errorf("security violation: absolute path %q is prohibited in MCP tools", reqPath)
	}

	cleanReq := filepath.Clean(reqPath)

	// Reject traversal indicators
	if cleanReq == ".." || strings.HasPrefix(cleanReq, ".."+string(filepath.Separator)) || strings.HasPrefix(cleanReq, "../") {
		return "", fmt.Errorf("security violation: directory traversal in %q is prohibited in MCP tools", reqPath)
	}

	absWorkspace, err := filepath.Abs(workspaceDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace directory: %w", err)
	}

	targetPath := filepath.Join(absWorkspace, cleanReq)

	// Lexical containment verification
	rel, err := filepath.Rel(absWorkspace, targetPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("security violation: path %q escapes workspace directory", reqPath)
	}

	// Symlink escape verification (if target exists)
	if realTarget, err := filepath.EvalSymlinks(targetPath); err == nil {
		if realWorkspace, err := filepath.EvalSymlinks(absWorkspace); err == nil {
			symRel, err := filepath.Rel(realWorkspace, realTarget)
			if err != nil || strings.HasPrefix(symRel, "..") || filepath.IsAbs(symRel) {
				return "", fmt.Errorf("security violation: symlink %q escapes workspace directory", reqPath)
			}
		}
	}

	return targetPath, nil
}

// SanitizeParseError sanitizes parser errors so raw secret lines are never echoed in tool responses.
func SanitizeParseError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()

	// If error message contains equals sign or raw line dump, redact the value portion
	if strings.Contains(msg, "=") {
		parts := strings.Split(msg, "=")
		return parts[0] + "=[REDACTED]"
	}

	return msg
}

// IsSecretPlaceholder returns true if a key name, default value, or schema tag indicates a secret.
func IsSecretPlaceholder(key, val string, tags map[string]string) bool {
	if val == "" {
		return false
	}

	// 1. Schema tags
	if tags != nil {
		typeTag := strings.ToLower(tags["type"])
		if typeTag == "secret" || typeTag == "password" || typeTag == "key" || typeTag == "token" {
			return true
		}
		if _, ok := tags["secret"]; ok {
			return true
		}
	}

	// 2. Sensitive key name heuristics
	if sensitiveKeyRegex.MatchString(key) {
		upperKey := strings.ToUpper(key)
		if upperKey != "KEY_COUNT" && upperKey != "PRIMARY_KEY" {
			return true
		}
	}

	// 3. Known secret value patterns
	for _, r := range scanner.SecretValuePatterns {
		if r.Pattern.MatchString(val) {
			return true
		}
	}

	// 4. High Shannon entropy (true random secrets)
	if len(val) >= 16 && scanner.ShannonEntropy(val) >= 4.0 {
		return true
	}

	return false
}
