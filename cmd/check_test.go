package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunCheck_Success(t *testing.T) {
	dir := t.TempDir()

	envPath := filepath.Join(dir, ".env")
	examplePath := filepath.Join(dir, ".env.example")

	os.WriteFile(envPath, []byte("PORT=3000\nAPP_NAME=RazifyApp\n"), 0644)
	os.WriteFile(examplePath, []byte("PORT=3000\nAPP_NAME=RazifyApp\n"), 0644)

	output, err := RunCheck(envPath, examplePath)
	if err != nil {
		t.Fatalf("unexpected error running check: %v", err)
	}

	if output.Status != "PASSED" {
		t.Errorf("expected status PASSED, got %s", output.Status)
	}
	if output.ValidationSummary.Passed != 2 {
		t.Errorf("expected 2 passed validation keys, got %d", output.ValidationSummary.Passed)
	}
}

func TestRunCheck_MissingKeys(t *testing.T) {
	dir := t.TempDir()

	envPath := filepath.Join(dir, ".env")
	examplePath := filepath.Join(dir, ".env.example")

	os.WriteFile(envPath, []byte("PORT=3000\n"), 0644)
	os.WriteFile(examplePath, []byte("# @required=true\nPORT=3000\nSTRIPE_KEY=\n"), 0644)

	output, err := RunCheck(envPath, examplePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Status != "FAILED" {
		t.Errorf("expected status FAILED due to missing key, got %s", output.Status)
	}
	if output.ValidationSummary.Missing != 1 {
		t.Errorf("expected 1 missing key, got %d", output.ValidationSummary.Missing)
	}
	if len(output.RemediationSteps) == 0 {
		t.Errorf("expected remediation steps to be present")
	}
}

func TestRunCheck_SecretLeak(t *testing.T) {
	dir := t.TempDir()

	envPath := filepath.Join(dir, ".env")
	examplePath := filepath.Join(dir, ".env.example")

	mockKey := "sk_live_" + "000000000000000000000000"
	os.WriteFile(envPath, []byte("STRIPE_KEY="+mockKey+"\n"), 0644)
	os.WriteFile(examplePath, []byte("STRIPE_KEY=\n"), 0644)

	output, err := RunCheck(envPath, examplePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.Status != "FAILED" {
		t.Errorf("expected status FAILED due to secret leak, got %s", output.Status)
	}
	if output.SecretScanSummary.Critical != 1 {
		t.Errorf("expected 1 critical secret leak, got %d", output.SecretScanSummary.Critical)
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.env")

	if fileExists(filePath) {
		t.Errorf("expected fileExists to be false for non-existent file")
	}

	os.WriteFile(filePath, []byte("FOO=BAR"), 0644)
	if !fileExists(filePath) {
		t.Errorf("expected fileExists to be true for existing file")
	}
}
