package mcp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Hossiy21/razify/internal/scanner"
	"github.com/Hossiy21/razify/internal/schema"
)

const UnknownParseErrorMessage = "failed to parse environment file due to a syntax error"

var (
	sensitiveKeyRegex = regexp.MustCompile(`(?i)(secret|password|passwd|pwd|token|auth|credential|api_key|apikey|private_key|privatekey|_key$|access_key|dsn|connection_string|salt|signing|bearer|cookie|session|jwt|webhook|cert)`)
	urlUserinfoRegex  = regexp.MustCompile(`(?i)[a-z][a-z0-9+.-]*://(?:[^:@\s]+:)?([^@\s]+)@`)

	windowsReservedNames = map[string]bool{
		"CON": true, "PRN": true, "AUX": true, "NUL": true,
		"COM1": true, "COM2": true, "COM3": true, "COM4": true, "COM5": true,
		"COM6": true, "COM7": true, "COM8": true, "COM9": true,
		"LPT1": true, "LPT2": true, "LPT3": true, "LPT4": true, "LPT5": true,
		"LPT6": true, "LPT7": true, "LPT8": true, "LPT9": true,
	}
)

// ValidateWorkspaceDir canonicalizes workspaceDir, ensuring it is non-empty, exists,
// is a directory, and is NOT a filesystem root, Windows drive root, or the user's home directory.
func ValidateWorkspaceDir(ws string) (string, error) {
	trimmed := strings.TrimSpace(ws)
	if trimmed == "" {
		return "", errors.New("security violation: workspace directory must be explicitly specified via --workspace")
	}

	// Refuse Windows drive root directly (e.g. C: or C:\ or C:/)
	if len(trimmed) == 2 && trimmed[1] == ':' {
		return "", fmt.Errorf("security violation: refusing to use drive root %q as workspace", ws)
	}
	if len(trimmed) == 3 && trimmed[1] == ':' && (trimmed[2] == '\\' || trimmed[2] == '/') {
		return "", fmt.Errorf("security violation: refusing to use drive root %q as workspace", ws)
	}

	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("security violation: invalid workspace path %q: %w", ws, err)
	}

	realWs, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("security violation: workspace directory %q does not exist or cannot be resolved: %w", ws, err)
	}

	fi, err := os.Stat(realWs)
	if err != nil {
		return "", fmt.Errorf("security violation: cannot stat workspace directory %q: %w", ws, err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("security violation: workspace %q is not a directory", ws)
	}

	cleanWs := filepath.Clean(realWs)

	// Refuse root directory (/ on Unix or \ on Windows)
	if cleanWs == "/" || cleanWs == `\` {
		return "", fmt.Errorf("security violation: refusing to use root directory %q as workspace", ws)
	}

	// Refuse Windows drive root (e.g. C:\ or C:)
	if vol := filepath.VolumeName(cleanWs); vol != "" {
		rest := strings.TrimPrefix(cleanWs, vol)
		if rest == "" || rest == `\` || rest == `/` {
			return "", fmt.Errorf("security violation: refusing to use drive root %q as workspace", ws)
		}
	}

	// Refuse filesystem root when Dir equals self
	if filepath.Dir(cleanWs) == cleanWs {
		return "", fmt.Errorf("security violation: refusing to use system root %q as workspace", ws)
	}

	// Refuse user home directory
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		cleanHome := filepath.Clean(homeDir)
		if realHome, err := filepath.EvalSymlinks(cleanHome); err == nil {
			if cleanWs == cleanHome || cleanWs == realHome || realWs == realHome {
				return "", fmt.Errorf("security violation: refusing to use user home directory %q as workspace", ws)
			}
		} else if cleanWs == cleanHome {
			return "", fmt.Errorf("security violation: refusing to use user home directory %q as workspace", ws)
		}
	}

	return realWs, nil
}

// ResolveConfinedPath resolves reqPath strictly within workspaceDir using fail-closed security checks.
// It rejects Windows rooted/drive-relative/UNC/NUL paths, symlink escapes, non-regular files,
// and enforces basename restrictions (.env, .env.*, *.env). For non-existent targets,
// it resolves the longest existing ancestor to verify workspace containment.
func ResolveConfinedPath(workspaceDir, reqPath, defaultFile string) (string, error) {
	realWorkspace, err := ValidateWorkspaceDir(workspaceDir)
	if err != nil {
		return "", err
	}

	if reqPath == "" {
		reqPath = defaultFile
	}
	if reqPath == "" {
		return "", errors.New("security violation: no path specified and no default file provided")
	}

	// 1. Reject null bytes
	if strings.ContainsRune(reqPath, 0) {
		return "", errors.New("security violation: path contains null bytes")
	}

	// 2. Reject UNC paths (\\... or //...)
	if strings.HasPrefix(reqPath, `\\`) || strings.HasPrefix(reqPath, `//`) {
		return "", fmt.Errorf("security violation: UNC path %q is prohibited", reqPath)
	}

	// 3. Reject rooted paths without drive letter (\... or /...)
	if strings.HasPrefix(reqPath, `\`) || strings.HasPrefix(reqPath, `/`) {
		return "", fmt.Errorf("security violation: rooted path %q is prohibited", reqPath)
	}

	// 4. Reject drive-relative or drive-absolute paths (C:... or D:/...)
	if len(reqPath) >= 2 && reqPath[1] == ':' {
		return "", fmt.Errorf("security violation: drive-relative or absolute drive path %q is prohibited", reqPath)
	}

	// 5. Reject general absolute paths
	if filepath.IsAbs(reqPath) {
		return "", fmt.Errorf("security violation: absolute path %q is prohibited", reqPath)
	}

	// 6. Reject Windows reserved device names (CON, PRN, AUX, NUL, COM1..9, LPT1..9)
	segments := strings.FieldsFunc(reqPath, func(r rune) bool { return r == '/' || r == '\\' })
	for _, seg := range segments {
		segUpper := strings.ToUpper(seg)
		nameOnly := segUpper
		if dot := strings.Index(segUpper, "."); dot != -1 {
			nameOnly = segUpper[:dot]
		}
		if windowsReservedNames[nameOnly] || windowsReservedNames[segUpper] {
			return "", fmt.Errorf("security violation: reserved Windows device name %q in path %q is prohibited", seg, reqPath)
		}
	}

	cleanReq := filepath.Clean(reqPath)

	// 7. Reject directory traversal indicators
	if cleanReq == ".." || strings.HasPrefix(cleanReq, ".."+string(filepath.Separator)) || strings.HasPrefix(cleanReq, "../") {
		return "", fmt.Errorf("security violation: directory traversal in %q is prohibited", reqPath)
	}

	// 8. Restrict basenames to .env, .env.*, or *.env
	base := filepath.Base(cleanReq)
	if !isAllowedEnvBasename(base) {
		return "", fmt.Errorf("security violation: file basename %q is prohibited; must match .env, .env.*, or *.env", base)
	}

	targetPath := filepath.Join(realWorkspace, cleanReq)

	// Lexical containment verification
	if !isWithin(realWorkspace, targetPath) {
		return "", fmt.Errorf("security violation: path %q escapes workspace directory", reqPath)
	}

	// 9. Filesystem check
	_, statErr := os.Stat(targetPath)
	if statErr == nil {
		// Target exists: resolve full symlink and verify regular file
		realTarget, err := filepath.EvalSymlinks(targetPath)
		if err != nil {
			return "", fmt.Errorf("security violation: failed to resolve symlink for %q: %w", reqPath, err)
		}

		if !isWithin(realWorkspace, realTarget) {
			return "", fmt.Errorf("security violation: symlink %q escapes workspace directory", reqPath)
		}

		info, err := os.Stat(realTarget)
		if err != nil {
			return "", fmt.Errorf("security violation: cannot stat target %q: %w", reqPath, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("security violation: target %q is not a regular file", reqPath)
		}

		return realTarget, nil
	}

	if !os.IsNotExist(statErr) {
		return "", fmt.Errorf("security violation: error inspecting target path %q: %w", reqPath, statErr)
	}

	// 10. Target does NOT exist: resolve longest existing ancestor
	cur := filepath.Dir(targetPath)
	missingSegments := []string{filepath.Base(targetPath)}
	var existingAncestor string

	for {
		if info, err := os.Stat(cur); err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("security violation: ancestor %q of %q is not a directory", cur, reqPath)
			}
			existingAncestor = cur
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		missingSegments = append([]string{filepath.Base(cur)}, missingSegments...)
		cur = parent
	}

	if existingAncestor == "" {
		return "", fmt.Errorf("security violation: no existing ancestor directory found for %q", reqPath)
	}

	realAncestor, err := filepath.EvalSymlinks(existingAncestor)
	if err != nil {
		return "", fmt.Errorf("security violation: failed to resolve ancestor symlinks for %q: %w", reqPath, err)
	}

	if !isAncestorWithinOrEqual(realWorkspace, realAncestor) {
		return "", fmt.Errorf("security violation: ancestor directory %q of %q escapes workspace directory", existingAncestor, reqPath)
	}

	resolvedPath := realAncestor
	for _, seg := range missingSegments {
		resolvedPath = filepath.Join(resolvedPath, seg)
	}

	if !isWithin(realWorkspace, resolvedPath) {
		return "", fmt.Errorf("security violation: resolved target path %q escapes workspace directory", reqPath)
	}

	return resolvedPath, nil
}

func isAllowedEnvBasename(base string) bool {
	if base == ".env" {
		return true
	}
	if strings.HasPrefix(base, ".env.") && len(base) > len(".env.") {
		return true
	}
	if strings.HasSuffix(base, ".env") && len(base) > len(".env") {
		return true
	}
	return false
}

// isWithin checks if target is strictly inside base (not base itself and does not escape with ..).
func isWithin(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	sep := string(filepath.Separator)
	if rel == ".." || strings.HasPrefix(rel, ".."+sep) || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	return true
}

// isAncestorWithinOrEqual checks if target is base or inside base.
func isAncestorWithinOrEqual(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	sep := string(filepath.Separator)
	if rel == ".." || strings.HasPrefix(rel, ".."+sep) || filepath.IsAbs(rel) {
		return false
	}
	return true
}

// SanitizeParseError sanitizes parser errors so raw secret lines are never echoed in tool responses.
// Structured schema.ParseError values with Line and Code are returned; unknown errors map to a fixed message.
func SanitizeParseError(err error) string {
	if err == nil {
		return ""
	}
	var parseErr *schema.ParseError
	if errors.As(err, &parseErr) {
		return parseErr.Error()
	}
	return UnknownParseErrorMessage
}

// IsSecretPlaceholder returns true if a key name, default value, or schema tag indicates a secret.
// It applies per-charset entropy thresholds, URL userinfo detection, and extended sensitive key names.
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

	// 2. Sensitive key name heuristics (including extended key names)
	if sensitiveKeyRegex.MatchString(key) {
		upperKey := strings.ToUpper(key)
		if upperKey != "KEY_COUNT" && upperKey != "PRIMARY_KEY" {
			return true
		}
	}

	// 3. URL-userinfo pattern (e.g. postgres://user:password@host:5432/db)
	if urlUserinfoRegex.MatchString(val) {
		return true
	}

	// 4. Known specific secret value patterns from scanner (excluding generic fallbacks)
	for _, r := range scanner.SecretValuePatterns {
		if strings.HasPrefix(r.Reason, "Hexadecimal Hash") || strings.HasPrefix(r.Reason, "Base64-like String") {
			continue
		}
		if r.Pattern.MatchString(val) {
			return true
		}
	}

	// 5. Per-charset entropy thresholds
	// Hex secrets: len >= 32, entropy >= 3.0
	if isHexString(val) && len(val) >= 32 {
		if scanner.ShannonEntropy(val) >= 3.0 {
			return true
		}
	}

	// Base64-like secrets: len >= 20, entropy >= 4.0
	if isBase64Like(val) && len(val) >= 20 {
		if scanner.ShannonEntropy(val) >= 4.0 {
			return true
		}
	}

	return false
}

// SafeMCPValue implements deny-by-default output formatting for MCP tools.
// A value is displayed ONLY when its @type (or tag) is port/bool/int/enum/range,
// it validates successfully against that type rule, and it does not match secret heuristics.
// All other values are masked with "[masked]".
func SafeMCPValue(key, val string, tags map[string]string) string {
	if val == "" {
		return ""
	}

	isAllowedType := false
	if tags != nil {
		t := strings.ToLower(tags["type"])
		if t == "port" || t == "bool" || t == "boolean" || t == "int" || t == "integer" || t == "enum" || t == "range" {
			isAllowedType = true
		}
		if _, ok := tags["enum"]; ok {
			isAllowedType = true
		}
		if _, ok := tags["range"]; ok {
			isAllowedType = true
		}
	}

	if isAllowedType {
		// Value must pass validation against tag constraints
		if schema.ValidateValue(val, tags, nil) == "" {
			// Value must not be a secret placeholder
			if !IsSecretPlaceholder(key, val, tags) {
				return val
			}
		}
	}

	return "[masked]"
}

func isHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func isBase64Like(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '+' || c == '/' || c == '=' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
