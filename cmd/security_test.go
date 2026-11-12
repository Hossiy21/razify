package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSecretsNeverExposed proves that raw secret values are never printed in plaintext
// across --json, --sarif, or standard scan outputs.
func TestSecretsNeverExposed(t *testing.T) {
	binName := "razify_sec_test"
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	binPath := filepath.Join(t.TempDir(), binName)
	buildCmd := exec.Command("go", "build", "-o", binPath, "..")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build test binary: %v", err)
	}

	secretFixture := filepath.Join("..", "testdata", "fixtures", "secret.env")
	secretExample := filepath.Join("..", "testdata", "fixtures", "secret.env.example")
	rawSecret := "AKIA1234567890EXAMPLE"

	// 1. Test --json output
	jsonCmd := exec.Command(binPath, "check", secretFixture, secretExample, "--json")
	var jsonStdout, jsonStderr bytes.Buffer
	jsonCmd.Stdout = &jsonStdout
	jsonCmd.Stderr = &jsonStderr
	err := jsonCmd.Run()
	if err == nil {
		t.Fatalf("expected check command to exit with violation code 1 for secret leaks, got exit 0")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("check --json failed to run: %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1 for check --json, got %d", exitErr.ExitCode())
	}
	jsonOutput := jsonStdout.String()
	if len(strings.TrimSpace(jsonOutput)) == 0 {
		t.Fatalf("check --json produced empty output, cannot verify security masking")
	}
	if strings.Contains(jsonOutput, rawSecret) {
		t.Fatalf("SECURITY VIOLATION: Raw secret %q leaked in --json output!\n%s", rawSecret, jsonOutput)
	}
	if !strings.Contains(jsonOutput, "AK*****************LE") {
		t.Errorf("Expected masked value in --json output, got:\n%s", jsonOutput)
	}

	// 2. Test --sarif output
	sarifCmd := exec.Command(binPath, "check", secretFixture, secretExample, "--sarif")
	var sarifStdout, sarifStderr bytes.Buffer
	sarifCmd.Stdout = &sarifStdout
	sarifCmd.Stderr = &sarifStderr
	err = sarifCmd.Run()
	if err == nil {
		t.Fatalf("expected check --sarif to exit with violation code 1 for secret leaks, got exit 0")
	}
	exitErr, ok = err.(*exec.ExitError)
	if !ok {
		t.Fatalf("check --sarif failed to run: %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1 for check --sarif, got %d", exitErr.ExitCode())
	}
	sarifOutput := sarifStdout.String()
	if len(strings.TrimSpace(sarifOutput)) == 0 {
		t.Fatalf("check --sarif produced empty output, cannot verify security masking")
	}
	if strings.Contains(sarifOutput, rawSecret) {
		t.Fatalf("SECURITY VIOLATION: Raw secret %q leaked in --sarif output!\n%s", rawSecret, sarifOutput)
	}
	if !strings.Contains(sarifOutput, "razify/secret-leak") || !strings.Contains(sarifOutput, "AWS_ACCESS_KEY") {
		t.Errorf("Expected secret-leak rule and key in --sarif output, got:\n%s", sarifOutput)
	}

	// 3. Test terminal scan output
	scanCmd := exec.Command(binPath, "scan", secretFixture)
	var scanStdout, scanStderr bytes.Buffer
	scanCmd.Stdout = &scanStdout
	scanCmd.Stderr = &scanStderr
	err = scanCmd.Run()
	if err == nil {
		t.Fatalf("expected scan command to exit with violation code 1 for secret leaks, got exit 0")
	}
	exitErr, ok = err.(*exec.ExitError)
	if !ok {
		t.Fatalf("scan command failed to run: %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1 for scan, got %d", exitErr.ExitCode())
	}
	scanOutput := scanStdout.String()
	if len(strings.TrimSpace(scanOutput)) == 0 {
		t.Fatalf("scan produced empty output, cannot verify security masking")
	}
	if strings.Contains(scanOutput, rawSecret) {
		t.Fatalf("SECURITY VIOLATION: Raw secret %q leaked in scan output!\n%s", rawSecret, scanOutput)
	}
	if !strings.Contains(scanOutput, "AK*****************LE") {
		t.Errorf("Expected masked value in scan output, got:\n%s", scanOutput)
	}

	// 4. Test doctor command database credential masking
	docDir := t.TempDir()
	docEnv := filepath.Join(docDir, ".env")
	docExample := filepath.Join(docDir, ".env.example")
	dbPassword := "super_secret_db_pass_789"
	if err := os.WriteFile(docEnv, []byte("DATABASE_URL=postgres://appuser:"+dbPassword+"@localhost:5432/prod\n"), 0644); err != nil {
		t.Fatalf("failed to write doc test .env: %v", err)
	}
	if err := os.WriteFile(docExample, []byte("DATABASE_URL=postgres://localhost:5432/prod\n"), 0644); err != nil {
		t.Fatalf("failed to write doc test .env.example: %v", err)
	}

	docCmd := exec.Command(binPath, "doctor", docEnv, docExample, "--json", "--timeout", "100")
	var docStdout, docStderr bytes.Buffer
	docCmd.Stdout = &docStdout
	docCmd.Stderr = &docStderr
	_ = docCmd.Run() // Exit code depends on health score, but command must execute

	docOutput := docStdout.String()
	if len(strings.TrimSpace(docOutput)) == 0 {
		t.Fatalf("doctor --json produced empty output, stderr: %s", docStderr.String())
	}
	if strings.Contains(docOutput, dbPassword) {
		t.Fatalf("SECURITY VIOLATION: Database password %q leaked in doctor --json output!\n%s", dbPassword, docOutput)
	}
	if !strings.Contains(docOutput, "postgres://localhost:5432") {
		t.Errorf("Expected masked database endpoint 'postgres://localhost:5432' in doctor output, got:\n%s", docOutput)
	}
}

