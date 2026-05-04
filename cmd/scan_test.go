package cmd

import (
	"os"
	"testing"
)

func TestRunScan(t *testing.T) {
	content := `
AWS_ACCESS_KEY=AKIA1234567890EXAMPLE
STRIPE_KEY=sk_test_12345
DB_PASSWORD=password
SAFE_VAR=just_a_value
`
	tmpfile, err := os.CreateTemp("", "scan_test.env")
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

	results, err := RunScan(tmpfile.Name())
	if err != nil {
		t.Fatalf("RunScan failed: %v", err)
	}

	// We expect 3 issues: AWS key, Stripe key (matches API_KEY pattern), and weak password
	expectedCount := 3
	if len(results) != expectedCount {
		t.Errorf("Expected %d issues, got %d", expectedCount, len(results))
	}

	foundAWS := false
	foundStripe := false
	foundWeak := false

	for _, r := range results {
		if r.Key == "AWS_ACCESS_KEY" && r.Risk == "CRITICAL" {
			foundAWS = true
		}
		if r.Key == "STRIPE_KEY" && r.Risk == "HIGH" {
			foundStripe = true
		}
		if r.Key == "DB_PASSWORD" && r.Risk == "CRITICAL" {
			foundWeak = true
		}
	}

	if !foundAWS {
		t.Error("Did not detect AWS_ACCESS_KEY as CRITICAL")
	}
	if !foundStripe {
		t.Error("Did not detect STRIPE_KEY as HIGH")
	}
	if !foundWeak {
		t.Error("Did not detect weak DB_PASSWORD as CRITICAL")
	}
}

func TestMaskValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"123", "****"},
		{"1234", "****"},
		{"secret", "se**et"},
		{"password123", "pa*******23"},
	}

	for _, tt := range tests {
		result := maskValue(tt.input)
		if result != tt.expected {
			t.Errorf("maskValue(%s): expected %s, got %s", tt.input, tt.expected, result)
		}
	}
}
