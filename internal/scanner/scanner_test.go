package scanner_test

import (
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/scanner"
	"github.com/Hossiy21/razify/internal/schema"
)

var mockStripeKey = "sk_live_" + "000000000000000000000000"

func TestMaskValue(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "********"},
		{"a", "********"},
		{"abcd", "********"},
		{"secret", "********"},
		{"password123", "***********"},
		{mockStripeKey, "sk****************************00"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := scanner.MaskValue(tt.input)
			if got != tt.want {
				t.Errorf("MaskValue(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestScanVars(t *testing.T) {
	vars := []schema.EnvVar{
		{LineNum: 1, Key: "STRIPE_KEY", Value: mockStripeKey},
		{LineNum: 2, Key: "DB_PASSWORD", Value: "password"},
		{LineNum: 3, Key: "SAFE_VAR", Value: "hello_world"},
		{LineNum: 4, Key: "IGNORED_SECRET", Value: mockStripeKey, Ignored: true},
	}

	results, summary := scanner.ScanVars(vars)

	if summary.Critical != 2 {
		t.Errorf("expected 2 critical findings, got %d", summary.Critical)
	}
	if summary.Total != 2 {
		t.Errorf("expected 2 total findings, got %d", summary.Total)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Key != "STRIPE_KEY" || results[0].Risk != "CRITICAL" {
		t.Errorf("expected STRIPE_KEY critical, got %+v", results[0])
	}
	if results[1].Key != "DB_PASSWORD" || results[1].Risk != "CRITICAL" {
		t.Errorf("expected DB_PASSWORD critical, got %+v", results[1])
	}
}

func TestScanVars_ShannonEntropyLivePath(t *testing.T) {
	// High-entropy token with an innocent key that does NOT match any regex in SecretValuePatterns or DangerPatterns
	highEntropyToken := "xK9mQ4vL2pW8zN7bT1jR5aC3dE6gH0yU"
	lowEntropyValue := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	vars := []schema.EnvVar{
		{LineNum: 1, Key: "DATA_BLOB", Value: highEntropyToken},
		{LineNum: 2, Key: "REPEATED_CHARS", Value: lowEntropyValue},
	}

	results, _ := scanner.ScanVars(vars)

	foundEntropy := false
	for _, r := range results {
		if r.Key == "DATA_BLOB" && strings.Contains(r.Reason, "High entropy detected") {
			foundEntropy = true
		}
		if r.Key == "REPEATED_CHARS" {
			t.Errorf("low entropy value was unexpectedly flagged: %+v", r)
		}
	}

	if !foundEntropy {
		t.Fatalf("ShannonEntropy on live scan path failed to flag high-entropy token! Results: %+v", results)
	}
}

func BenchmarkScan(b *testing.B) {
	vars := []schema.EnvVar{
		{LineNum: 1, Key: "PORT", Value: "3000"},
		{LineNum: 2, Key: "NODE_ENV", Value: "production"},
		{LineNum: 3, Key: "STRIPE_KEY", Value: mockStripeKey},
		{LineNum: 4, Key: "AWS_ACCESS_KEY", Value: "AKIA0000000000000000"},
		{LineNum: 5, Key: "DATABASE_URL", Value: "postgres://user:pass@localhost:5432/db"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = scanner.ScanVars(vars)
	}
}
