package cmd

import (
	"os"
	"testing"
)

func TestRunAudit(t *testing.T) {
	exampleContent := `
DB_PASSWORD=
AWS_ACCESS_KEY=
`
	envContent := `
DB_PASSWORD=password
AWS_ACCESS_KEY=AKIA1234567890
`
	exampleFile, _ := os.CreateTemp("", "example.env")
	envFile, _ := os.CreateTemp("", "actual.env")

	defer os.Remove(exampleFile.Name())
	defer os.Remove(envFile.Name())

	os.WriteFile(exampleFile.Name(), []byte(exampleContent), 0644)
	os.WriteFile(envFile.Name(), []byte(envContent), 0644)

	report, err := RunAudit(envFile.Name(), exampleFile.Name())
	if err != nil {
		t.Fatalf("RunAudit failed: %v", err)
	}

	// 2 critical issues: 1 weak password + 1 AWS key
	if report.ScanCritical != 2 {
		t.Errorf("Expected 2 critical scan issues, got %d", report.ScanCritical)
	}

	// Scoring: 100 - (2 * 30) = 40
	if report.Score != 40 {
		t.Errorf("Expected score 40, got %d", report.Score)
	}
}
