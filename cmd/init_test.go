package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/schema"
)

func TestInitCmd_FlagRegistration(t *testing.T) {
	flags := []string{"from-code", "dir", "output", "force"}
	for _, f := range flags {
		if initCmd.Flags().Lookup(f) == nil {
			t.Errorf("expected initCmd to have flag %s", f)
		}
	}
}

func TestInitCmd_FromCodeExecution(t *testing.T) {
	tempDir := t.TempDir()

	goCode := `package main
import "os"
func run() {
	_ = os.Getenv("PORT")
	_ = os.Getenv("DATABASE_URL")
	_ = os.Getenv("ENABLE_METRICS")
}
`
	srcFile := filepath.Join(tempDir, "server.go")
	if err := os.WriteFile(srcFile, []byte(goCode), 0644); err != nil {
		t.Fatalf("failed to write test code: %v", err)
	}

	outFile := filepath.Join(tempDir, "test.env.example")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"init", "--from-code", "-d", tempDir, "-o", outFile, "-f"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init --from-code failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read generated output file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "PORT=8080") {
		t.Errorf("expected PORT=8080 in generated file, got:\n%s", content)
	}
	if !strings.Contains(content, "@type port") {
		t.Errorf("expected @type port tag, got:\n%s", content)
	}
	if !strings.Contains(content, "@type url") {
		t.Errorf("expected @type url tag, got:\n%s", content)
	}
	if !strings.Contains(content, "@type bool") {
		t.Errorf("expected @type bool tag, got:\n%s", content)
	}

	// Verify generated file parses cleanly with schema parser
	envVars, err := schema.ParseEnvWithMetadataReader(strings.NewReader(content))
	if err != nil {
		t.Fatalf("generated file failed schema parsing: %v", err)
	}
	if len(envVars) != 3 {
		t.Errorf("expected 3 variables parsed, got %d", len(envVars))
	}
}
