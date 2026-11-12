package cmd

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestArchitecture_StrictNetworkBoundaries verifies that:
// 1. net/http is strictly forbidden in all packages outside cmd/version.go.
// 2. net is strictly forbidden outside internal/doctor and internal/schema (where only net.ParseIP is permitted).
func TestArchitecture_StrictNetworkBoundaries(t *testing.T) {
	root := filepath.Join("..")

	allowedNetHttpFile := filepath.Clean(filepath.Join(root, "cmd", "version.go"))
	allowedNetPackages := map[string]bool{
		filepath.Clean(filepath.Join(root, "internal", "doctor")): true,
		filepath.Clean(filepath.Join(root, "internal", "schema")): true,
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == "scratch" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		cleanPath := filepath.Clean(path)
		dirPath := filepath.Dir(cleanPath)

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, cleanPath, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("failed to parse file %s: %v", cleanPath, err)
		}

		for _, imp := range node.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			// 1. Verify net/http is strictly forbidden outside cmd/version.go
			if importPath == "net/http" || strings.HasPrefix(importPath, "net/http/") {
				if cleanPath != allowedNetHttpFile {
					t.Errorf("VIOLATION: File %s imports forbidden package %q. Outbound HTTP is only permitted in cmd/version.go for opt-in update checks.", cleanPath, importPath)
				}
			}

			// 2. Verify net is forbidden outside internal/doctor and internal/schema
			if importPath == "net" {
				if !allowedNetPackages[dirPath] {
					t.Errorf("VIOLATION: File %s imports forbidden package %q. Package 'net' is only permitted in internal/doctor and internal/schema.", cleanPath, importPath)
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}
}
