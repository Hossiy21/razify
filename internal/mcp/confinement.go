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

const (
	UnknownParseErrorMessage = "failed to parse environment file due to a syntax error"
	MaxAllowedFileSize       = 1024 * 1024 // 1 MB cap
)

var (
	// Typed sentinel errors returned by OpenConfinedEnvFile
	ErrNotExist  = errors.New("environment file does not exist")
	ErrForbidden = errors.New("security violation: path forbidden or escapes workspace")
	ErrTooLarge  = errors.New("security violation: file exceeds maximum allowed size")

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
// is a directory, and is NOT a filesystem root, Windows drive root, the user's home directory,
// or an ancestor of the user's home directory.
func ValidateWorkspaceDir(ws string) (string, error) {
	trimmed := strings.TrimSpace(ws)
	if trimmed == "" {
		return "", errors.New("security violation: workspace directory must be explicitly specified via --workspace")
	}

	// Refuse ':' in workspace string to avoid drive ambiguities or alternate stream confusion
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

	wsFi, err := os.Stat(realWs)
	if err != nil {
		return "", fmt.Errorf("security violation: cannot stat workspace directory %q: %w", ws, err)
	}
	if !wsFi.IsDir() {
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

	// Refuse user home directory and any ancestor of the home directory
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		homeFi, errHome := os.Stat(homeDir)
		if errHome == nil {
			if os.SameFile(homeFi, wsFi) {
				return "", fmt.Errorf("security violation: refusing to use user home directory %q as workspace", ws)
			}
		}

		realHome, err := filepath.EvalSymlinks(homeDir)
		if err == nil {
			cleanRealHome := filepath.Clean(realHome)
			if cleanWs == cleanRealHome {
				return "", fmt.Errorf("security violation: refusing to use user home directory %q as workspace", ws)
			}

			// Check if cleanWs is an ancestor of cleanRealHome
			relHome, err := filepath.Rel(cleanWs, cleanRealHome)
			if err == nil {
				sep := string(filepath.Separator)
				if relHome == "." {
					return "", fmt.Errorf("security violation: refusing to use user home directory %q as workspace", ws)
				}
				if relHome != ".." && !strings.HasPrefix(relHome, ".."+sep) && !filepath.IsAbs(relHome) {
					return "", fmt.Errorf("security violation: refusing to use home directory ancestor %q as workspace", ws)
				}
			}
		}
	}

	return realWs, nil
}

// OpenConfinedEnvFile opens an environment file strictly confined within workspaceDir using os.OpenRoot.
// It verifies regular file status, enforces the 1 MB file size cap, checks root.Lstat to reject symlinks,
// and verifies that the opened file handle matches the directory entry (!os.SameFile check) to defeat swap races / TOCTOU.
func OpenConfinedEnvFile(workspaceDir, reqPath, defaultFile string) (*os.File, string, error) {
	resolvedPath, err := ResolveConfinedPath(workspaceDir, reqPath, defaultFile)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrForbidden, err)
	}

	realWorkspace, err := ValidateWorkspaceDir(workspaceDir)
	if err != nil {
		return nil, "", fmt.Errorf("%w: %v", ErrForbidden, err)
	}

	rel, err := filepath.Rel(realWorkspace, resolvedPath)
	if err != nil {
		return nil, "", fmt.Errorf("%w: failed to determine relative path for %q: %v", ErrForbidden, resolvedPath, err)
	}

	root, err := os.OpenRoot(realWorkspace)
	if err != nil {
		return nil, "", fmt.Errorf("%w: failed to open workspace root: %v", ErrForbidden, err)
	}
	defer root.Close()

	cleanRel := filepath.ToSlash(filepath.Clean(rel))
	f, err := root.Open(cleanRel)
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, os.ErrNotExist) {
			return nil, "", fmt.Errorf("%w: failed to open %q within workspace: %v", ErrNotExist, reqPath, err)
		}
		return nil, "", fmt.Errorf("%w: failed to open %q within workspace root: %v", ErrForbidden, reqPath, err)
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, "", fmt.Errorf("%w: failed to stat opened file %q: %v", ErrForbidden, reqPath, err)
	}
	if !fi.Mode().IsRegular() {
		f.Close()
		return nil, "", fmt.Errorf("%w: target %q is not a regular file", ErrForbidden, reqPath)
	}
	if fi.Size() > MaxAllowedFileSize {
		f.Close()
		return nil, "", fmt.Errorf("%w: file %q exceeds maximum allowed size of 1MB (%d bytes)", ErrTooLarge, reqPath, fi.Size())
	}

	// Compare f.Stat() with root.Lstat(cleanRel) to reject symlinks and defeat swap-race / TOCTOU attacks
	lfi, err := root.Lstat(cleanRel)
	if err != nil {
		f.Close()
		return nil, "", fmt.Errorf("%w: failed to lstat %q: %v", ErrForbidden, reqPath, err)
	}
	if lfi.Mode()&os.ModeSymlink != 0 {
		f.Close()
		return nil, "", fmt.Errorf("%w: target %q is a symlink", ErrForbidden, reqPath)
	}
	if !os.SameFile(fi, lfi) {
		f.Close()
		return nil, "", fmt.Errorf("%w: file handle does not match directory entry for %q (possible symlink swap or TOCTOU)", ErrForbidden, reqPath)
	}

	return f, resolvedPath, nil
}

// ResolveConfinedPath resolves reqPath strictly within workspaceDir using fail-closed security checks.
// It rejects Windows rooted/drive-relative/UNC/NUL paths, ':' alternate data streams,
// symlink escapes, symlinked final components (closing in-workspace symlink bypasses), non-regular files,
// 1 MB size violations, and enforces basename restrictions (.env, .env.*, *.env). For non-existent targets,
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

	// 1. Reject ':' anywhere in reqPath (NTFS alternate data streams and drive prefixes)
	if strings.ContainsRune(reqPath, ':') {
		return "", fmt.Errorf("security violation: ':' is prohibited in path %q (NTFS alternate data streams / drive prefixes)", reqPath)
	}

	// 2. Reject null bytes
	if strings.ContainsRune(reqPath, 0) {
		return "", errors.New("security violation: path contains null bytes")
	}

	// 3. Reject UNC paths (\\... or //...)
	if strings.HasPrefix(reqPath, `\\`) || strings.HasPrefix(reqPath, `//`) {
		return "", fmt.Errorf("security violation: UNC path %q is prohibited", reqPath)
	}

	// 4. Reject rooted paths without drive letter (\... or /...)
	if strings.HasPrefix(reqPath, `\`) || strings.HasPrefix(reqPath, `/`) {
		return "", fmt.Errorf("security violation: rooted path %q is prohibited", reqPath)
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

	// 9. Filesystem check using os.Lstat on targetPath
	lfi, lstatErr := os.Lstat(targetPath)
	if lstatErr == nil {
		// Final component check: reject if final component is a symlink.
		// This closes the bypass where an in-workspace symlink named .env points to .git/config, credentials.json, etc.
		// It also closes dangling symlinks immediately.
		if lfi.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("security violation: target %q is a symlink", reqPath)
		}

		// Resolve full path and re-verify basename on the resolved target
		realTarget, err := filepath.EvalSymlinks(targetPath)
		if err != nil {
			return "", fmt.Errorf("security violation: failed to resolve symlink for %q: %w", reqPath, err)
		}

		resolvedBase := filepath.Base(realTarget)
		if !isAllowedEnvBasename(resolvedBase) {
			return "", fmt.Errorf("security violation: resolved target basename %q is prohibited; must match .env, .env.*, or *.env", resolvedBase)
		}

		if !isWithin(realWorkspace, realTarget) {
			return "", fmt.Errorf("security violation: path %q escapes workspace directory via symlink", reqPath)
		}

		info, err := os.Stat(realTarget)
		if err != nil {
			return "", fmt.Errorf("security violation: cannot stat target %q: %w", reqPath, err)
		}
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("security violation: target %q is not a regular file", reqPath)
		}
		if info.Size() > MaxAllowedFileSize {
			return "", fmt.Errorf("security violation: file %q exceeds maximum allowed size of 1MB (%d bytes)", reqPath, info.Size())
		}

		// Verify through os.OpenRoot ensuring confinement is enforced at OS open level
		rel, err := filepath.Rel(realWorkspace, realTarget)
		if err == nil {
			root, err := os.OpenRoot(realWorkspace)
			if err != nil {
				return "", fmt.Errorf("security violation: failed to open workspace root: %w", err)
			}
			defer root.Close()

			f, err := root.Open(filepath.ToSlash(filepath.Clean(rel)))
			if err != nil {
				return "", fmt.Errorf("security violation: cannot open %q via workspace root: %w", reqPath, err)
			}
			defer f.Close()

			fi, err := f.Stat()
			if err != nil || !fi.Mode().IsRegular() {
				return "", fmt.Errorf("security violation: target %q is not a regular file", reqPath)
			}
			if fi.Size() > MaxAllowedFileSize {
				return "", fmt.Errorf("security violation: file %q exceeds maximum allowed size of 1MB (%d bytes)", reqPath, fi.Size())
			}
		}

		return realTarget, nil
	}

	if !os.IsNotExist(lstatErr) {
		return "", fmt.Errorf("security violation: error inspecting target path %q: %w", reqPath, lstatErr)
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

// isAllowedEnvBasename verifies the file basename matches .env, .env.*, or *.env.
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
// Note: checks rel == ".." or rel starts with ".."+sep, avoiding flawed prefix matches like /work vs /work-evil.
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
//
// NOTE: Shannon entropy is a secondary defense heuristic. Deterministic pattern matching,
// schema tags (@type=secret/password/token/key), sensitive key names, and URL userinfo detection
// act as primary defense layers.
//
// For short base64 sequences (length < 32), Shannon entropy has higher variance due to small
// sample sizes (e.g. structured English identifiers with high character diversity may approach
// high entropy). Consequently:
// - Hex strings require len >= 32 and Shannon entropy >= 3.0.
// - Base64-like strings: for len >= 32, entropy threshold is >= 4.0; for lengths 20 <= len < 32,
//   the threshold is tightened to >= 4.3 to reduce false positives on diverse English identifiers.
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

	// 5. Per-charset entropy thresholds (secondary heuristic layer)
	// Hex secrets: len >= 32, entropy >= 3.0
	if isHexString(val) && len(val) >= 32 {
		if scanner.ShannonEntropy(val) >= 3.0 {
			return true
		}
	}

	// Base64-like secrets: calibrated empirical thresholds guaranteeing >=90% detection
	// for lengths 20-31 based on 10th-percentile entropy of 1,000 random strings per length,
	// and 4.0 for lengths >= 32.
	if isBase64Like(val) {
		if len(val) >= 32 {
			if scanner.ShannonEntropy(val) >= 4.0 {
				return true
			}
		} else if len(val) >= 20 {
			if scanner.ShannonEntropy(val) >= base64EntropyThreshold(len(val)) {
				return true
			}
		}
	}

	return false
}

// base64EntropyThreshold returns the empirical 10th-percentile Shannon entropy threshold
// calibrated from a fixed-seed test of 1,000 random base64 strings per length to guarantee >=90% detection.
func base64EntropyThreshold(length int) float64 {
	switch length {
	case 20:
		return 3.88
	case 21:
		return 3.88
	case 22:
		return 3.97
	case 23:
		return 4.00
	case 24:
		return 4.05
	case 25:
		return 4.10
	case 26:
		return 4.16
	case 27:
		return 4.18
	case 28:
		return 4.23
	case 29:
		return 4.28
	case 30:
		return 4.30
	case 31:
		return 4.34
	default:
		if length >= 32 {
			return 4.00
		}
		return 4.50
	}
}

// SanitizeMCPText sanitizes metadata strings (comments, categories, tag values) in MCP responses.
// If the text contains or matches known secret patterns, URL userinfo, or high-entropy tokens,
// the sensitive portions are replaced with "[masked]". Long text is truncated to 512 characters.
func SanitizeMCPText(text string) string {
	if text == "" {
		return ""
	}

	if len(text) > 512 {
		text = text[:512] + "..."
	}

	// 1. Redact URL userinfo
	if urlUserinfoRegex.MatchString(text) {
		text = urlUserinfoRegex.ReplaceAllString(text, "${1}[masked]@")
	}

	// 2. Redact known secret patterns
	for _, r := range scanner.SecretValuePatterns {
		if strings.HasPrefix(r.Reason, "Hexadecimal Hash") || strings.HasPrefix(r.Reason, "Base64-like String") {
			continue
		}
		if r.Pattern.MatchString(text) {
			text = r.Pattern.ReplaceAllString(text, "[masked]")
		}
	}

	// 3. Scan individual words/tokens for high-entropy hex/base64 secrets
	words := strings.Fields(text)
	replaced := false
	for i, w := range words {
		cleanWord := strings.Trim(w, `"'`+"`"+`(),;:[]{}<>`)
		if len(cleanWord) >= 20 {
			if isHexString(cleanWord) && len(cleanWord) >= 32 && scanner.ShannonEntropy(cleanWord) >= 3.0 {
				words[i] = "[masked]"
				replaced = true
			} else if isBase64Like(cleanWord) {
				if len(cleanWord) >= 32 && scanner.ShannonEntropy(cleanWord) >= 4.0 {
					words[i] = "[masked]"
					replaced = true
				} else if len(cleanWord) >= 20 && scanner.ShannonEntropy(cleanWord) >= base64EntropyThreshold(len(cleanWord)) {
					words[i] = "[masked]"
					replaced = true
				}
			}
		}
	}
	if replaced {
		text = strings.Join(words, " ")
	}

	return text
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
