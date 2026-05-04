package cmd

import (
	"os"
	"testing"
)

func TestValidate(t *testing.T) {
	exampleContent := `
REQUIRED_VAR=
PLACEHOLDER_VAR=your-value-here
OPTIONAL_VAR=
`
	envContent := `
REQUIRED_VAR=actual-value
PLACEHOLDER_VAR=your-value-here
# OPTIONAL_VAR is missing
`
	exampleFile, _ := os.CreateTemp("", "example.env")
	envFile, _ := os.CreateTemp("", "actual.env")
	
	defer os.Remove(exampleFile.Name())
	defer os.Remove(envFile.Name())

	os.WriteFile(exampleFile.Name(), []byte(exampleContent), 0644)
	os.WriteFile(envFile.Name(), []byte(envContent), 0644)

	missing, placeholder, empty, okCount, _, err := RunValidate(envFile.Name(), exampleFile.Name())
	if err != nil {
		t.Fatalf("RunValidate failed: %v", err)
	}

	if missing != 1 {
		t.Errorf("Expected 1 missing variable (OPTIONAL_VAR), got %d", missing)
	}
	if placeholder != 1 {
		t.Errorf("Expected 1 placeholder variable (PLACEHOLDER_VAR), got %d", placeholder)
	}
	if okCount != 1 {
		t.Errorf("Expected 1 OK variable (REQUIRED_VAR), got %d", okCount)
	}
	if empty != 0 {
		t.Errorf("Expected 0 empty variables, got %d", empty)
	}
}
