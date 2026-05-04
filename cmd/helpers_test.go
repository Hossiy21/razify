package cmd

import (
	"os"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	// Create a temporary env file for testing
	content := `
# This is a comment
DB_HOST=localhost
DB_PORT=5432

EMPTY_VAR=
  SPACED_KEY  =  spaced_value  
`
	tmpfile, err := os.CreateTemp("", "test.env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Run the parser
	vars, err := parseEnvFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("parseEnvFile failed: %v", err)
	}

	// Verify results
	expected := map[string]string{
		"DB_HOST":    "localhost",
		"DB_PORT":    "5432",
		"EMPTY_VAR":   "",
		"SPACED_KEY": "spaced_value",
	}

	if len(vars) != len(expected) {
		t.Errorf("Expected %d variables, got %d", len(expected), len(vars))
	}

	for k, v := range expected {
		if vars[k] != v {
			t.Errorf("For key %s: expected %s, got %s", k, v, vars[k])
		}
	}
}

func TestParseEnvFileNotFound(t *testing.T) {
	_, err := parseEnvFile("non_existent_file.env")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}
