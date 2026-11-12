package sandbox_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Hossiy21/razify/internal/sandbox"
)

func TestFindEnvFilesToMask_Monorepo(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Files that MUST be masked
	filesToMask := []string{
		".env",
		".env.local",
		".env.production",
		"packages/api/.env",
		"packages/api/.env.test",
		"packages/web/.env.staging",
		"services/worker/app.env",
	}

	// 2. Files that MUST NOT be masked (templates/samples/examples)
	filesToExclude := []string{
		".env.example",
		".env.sample",
		".env.template",
		"packages/api/.env.example",
		"packages/web/.env.sample",
		"services/worker/app.env.template",
	}

	// 3. Files in ignored directories (node_modules, .git, vendor) that MUST NOT be traversed
	filesInIgnoredDirs := []string{
		"node_modules/somelib/.env",
		".git/hooks/.env",
		"vendor/pkg/.env",
		".venv/lib/.env",
	}

	for _, rel := range append(append(filesToMask, filesToExclude...), filesInIgnoredDirs...) {
		fullPath := filepath.Join(tmpDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte("SECRET=123\n"), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	discovered, err := sandbox.FindEnvFilesToMask(tmpDir)
	if err != nil {
		t.Fatalf("FindEnvFilesToMask failed: %v", err)
	}

	discoveredSet := make(map[string]bool)
	for _, f := range discovered {
		discoveredSet[f] = true
	}

	// Verify all expected files are present
	for _, exp := range filesToMask {
		if !discoveredSet[exp] {
			t.Errorf("expected %q to be in masked list, but it was missing! Found: %+v", exp, discovered)
		}
	}

	// Verify excluded template/sample files are NOT in masked list
	for _, excl := range filesToExclude {
		if discoveredSet[excl] {
			t.Errorf("CRITICAL: template/example file %q was incorrectly included in mask list!", excl)
		}
	}

	// Verify ignored directories are NOT in masked list
	for _, ign := range filesInIgnoredDirs {
		if discoveredSet[ign] {
			t.Errorf("CRITICAL: file in ignored directory %q was included in mask list!", ign)
		}
	}
}

func TestResolveUser(t *testing.T) {
	// Explicit user passed
	if got := sandbox.ResolveUser("2000:2000"); got != "2000:2000" {
		t.Errorf("expected 2000:2000, got %s", got)
	}

	// Default fallback on all platforms MUST be fixed non-root 1000:1000
	if got := sandbox.ResolveUser(""); got != "1000:1000" {
		t.Errorf("expected default non-root user 1000:1000, got %s", got)
	}
}
