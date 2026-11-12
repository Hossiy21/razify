package cmd

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSecretsNeverExposed proves that raw secret values are never printed in plaintext
// across --json, --sarif, or standard scan outputs.
func TestSecretsNeverExposed(t *testing.T) {
	binPath := filepath.Join(t.TempDir(), "razify_sec_test")
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
	_ = jsonCmd.Run()

	jsonOutput := jsonStdout.String() + jsonStderr.String()
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
	_ = sarifCmd.Run()

	sarifOutput := sarifStdout.String() + sarifStderr.String()
	if strings.Contains(sarifOutput, rawSecret) {
		t.Fatalf("SECURITY VIOLATION: Raw secret %q leaked in --sarif output!\n%s", rawSecret, sarifOutput)
	}

	// 3. Test terminal scan output
	scanCmd := exec.Command(binPath, "scan", secretFixture)
	var scanStdout, scanStderr bytes.Buffer
	scanCmd.Stdout = &scanStdout
	scanCmd.Stderr = &scanStderr
	_ = scanCmd.Run()

	scanOutput := scanStdout.String() + scanStderr.String()
	if strings.Contains(scanOutput, rawSecret) {
		t.Fatalf("SECURITY VIOLATION: Raw secret %q leaked in scan output!\n%s", rawSecret, scanOutput)
	}
	if !strings.Contains(scanOutput, "AK*****************LE") {
		t.Errorf("Expected masked value in scan output, got:\n%s", scanOutput)
	}
}

