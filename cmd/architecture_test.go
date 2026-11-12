package cmd

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoNetHttpInCorePackages ensures core internal packages remain strictly offline-first
// with zero dependencies on net/http.
func TestNoNetHttpInCorePackages(t *testing.T) {
	corePackages := []string{
		filepath.Join("..", "internal", "schema"),
		filepath.Join("..", "internal", "scanner"),
		filepath.Join("..", "internal", "reporter"),
		filepath.Join("..", "internal", "sandbox"),
		filepath.Join("..", "internal", "doctor"),
		filepath.Join("..", "internal", "mcp"),
	}

	for _, pkgDir := range corePackages {
		t.Run(pkgDir, func(t *testing.T) {
			entries, err := os.ReadDir(pkgDir)
			if err != nil {
				t.Fatalf("failed to read package dir %s: %v", pkgDir, err)
			}

			fset := token.NewFileSet()
			for _, entry := range entries {
				if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
					continue
				}

				filePath := filepath.Join(pkgDir, entry.Name())
				node, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
				if err != nil {
					t.Fatalf("failed to parse file %s: %v", filePath, err)
				}

				for _, imp := range node.Imports {
					importPath := strings.Trim(imp.Path.Value, `"`)
					if importPath == "net/http" || strings.HasPrefix(importPath, "net/http/") {
						t.Errorf("VIOLATION: File %s imports forbidden package %q. Core packages must be offline-first.", filePath, importPath)
					}
				}
			}
		})
	}
}

