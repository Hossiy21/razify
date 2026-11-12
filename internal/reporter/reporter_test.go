package reporter_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Hossiy21/razify/internal/reporter"
	"github.com/Hossiy21/razify/internal/scanner"
	"github.com/Hossiy21/razify/internal/schema"
)

func TestExitHandler(t *testing.T) {
	// Default handler (no override)
	h := reporter.NewExitHandler(-1)
	if code := h.ResolveViolationCode(); code != reporter.ExitViolation {
		t.Errorf("expected default code %d, got %d", reporter.ExitViolation, code)
	}

	// Override handler
	hOverride := reporter.NewExitHandler(42)
	if code := hOverride.ResolveViolationCode(); code != 42 {
		t.Errorf("expected override code 42, got %d", code)
	}
}

func TestRenderSarif(t *testing.T) {
	res := reporter.Result{
		TargetFile:   ".env",
		TemplateFile: ".env.example",
		Status:       "FAILED",
		Diagnostics: []reporter.Diagnostic{
			{
				RuleID:   "razify/secret-leak",
				Message:  "Stripe Live Secret Key",
				Severity: reporter.SeverityError,
				File:     ".env",
				Line:     12,
				Key:      "STRIPE_KEY",
			},
			{
				RuleID:   "razify/missing-key",
				Message:  "Required key not found",
				Severity: reporter.SeverityError,
				File:     ".env",
				Line:     1,
				Key:      "APP_SECRET",
			},
		},
		ValidationSummary: schema.ValidateSummary{Missing: 1},
		SecretSummary:     scanner.ScanSummary{Critical: 1, Total: 1},
	}

	sarifJSON, err := reporter.RenderSarif(res, "v1.0.0")
	if err != nil {
		t.Fatalf("RenderSarif failed: %v", err)
	}

	if !strings.Contains(sarifJSON, `"version": "2.1.0"`) {
		t.Errorf("SARIF missing version 2.1.0")
	}
	if !strings.Contains(sarifJSON, `"name": "razify"`) {
		t.Errorf("SARIF missing tool name razify")
	}

	var parsed reporter.SarifReport
	if err := json.Unmarshal([]byte(sarifJSON), &parsed); err != nil {
		t.Fatalf("failed to unmarshal SARIF: %v", err)
	}

	if len(parsed.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(parsed.Runs))
	}
	if len(parsed.Runs[0].Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(parsed.Runs[0].Results))
	}
	if parsed.Runs[0].Results[0].RuleID != "razify/secret-leak" {
		t.Errorf("expected rule razify/secret-leak, got %s", parsed.Runs[0].Results[0].RuleID)
	}
}

func BenchmarkSarifRender(b *testing.B) {
	res := reporter.Result{
		TargetFile:   ".env",
		TemplateFile: ".env.example",
		Status:       "FAILED",
		Diagnostics: []reporter.Diagnostic{
			{
				RuleID:   "razify/secret-leak",
				Message:  "Stripe Live Secret Key",
				Severity: reporter.SeverityError,
				File:     ".env",
				Line:     12,
				Key:      "STRIPE_KEY",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reporter.RenderSarif(res, "v1.0.0")
	}
}

