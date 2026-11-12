package reporter

import (
	"github.com/Hossiy21/razify/internal/scanner"
	"github.com/Hossiy21/razify/internal/schema"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityNote    Severity = "note"
)

// Diagnostic represents a single finding in a file (validation issue, secret leak, weak password, etc.).
type Diagnostic struct {
	RuleID   string   `json:"rule_id"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Key      string   `json:"key"`
	Value    string   `json:"value,omitempty"`
}

// Result is the unified data model representing the outcome of checking or scanning files.
type Result struct {
	TargetFile        string                 `json:"target_file"`
	TemplateFile      string                 `json:"template_file,omitempty"`
	Status            string                 `json:"status"` // "PASSED" or "FAILED"
	Diagnostics       []Diagnostic           `json:"diagnostics"`
	ValidationSummary schema.ValidateSummary `json:"validation_summary"`
	SecretSummary     scanner.ScanSummary    `json:"secret_summary"`
	RemediationSteps  []string               `json:"remediation_steps,omitempty"`
}

// LegacyCheckOutput preserves exact field names and structure for existing JSON consumers (vscode-razify, CI).
type LegacyCheckOutput struct {
	EnvFile           string                    `json:"env_file"`
	ExampleFile       string                    `json:"example_file"`
	Status            string                    `json:"status"` // "PASSED" or "FAILED"
	ValidationResults []schema.ValidationResult `json:"validation_results"`
	ValidationSummary schema.ValidateSummary    `json:"validation_summary"`
	SecretScanResults []scanner.ScanResult      `json:"secret_scan_results"`
	SecretScanSummary scanner.ScanSummary       `json:"secret_scan_summary"`
	RemediationSteps  []string                  `json:"remediation_steps"`
}

// LegacyValidateOutput preserves validate command JSON output.
type LegacyValidateOutput struct {
	EnvFile     string                    `json:"env_file"`
	ExampleFile string                    `json:"example_file"`
	Results     []schema.ValidationResult `json:"results"`
	Summary     schema.ValidateSummary    `json:"summary"`
}

// LegacyScanOutput preserves scan command JSON output.
type LegacyScanOutput struct {
	File    string               `json:"file"`
	Results []scanner.ScanResult `json:"results"`
	Summary scanner.ScanSummary  `json:"summary"`
}

// BuildResult creates a unified Result from validation and scan data.
func BuildResult(envFile, exampleFile string, valSummary schema.ValidateSummary, valResults []schema.ValidationResult, scanSummary scanner.ScanSummary, scanResults []scanner.ScanResult, remediation []string) Result {
	status := "PASSED"
	if valSummary.Missing > 0 || scanSummary.Critical > 0 || scanSummary.High > 0 {
		status = "FAILED"
	}

	var diags []Diagnostic

	for _, vr := range valResults {
		if vr.Status == "OK" {
			continue
		}
		var sev Severity
		ruleID := "razify/validation"
		switch vr.Status {
		case "MISSING":
			sev = SeverityError
			ruleID = "razify/missing-key"
		case "INVALID":
			sev = SeverityError
			ruleID = "razify/invalid-type"
		case "EMPTY":
			sev = SeverityWarning
			ruleID = "razify/empty-value"
		case "PLACEHOLDER":
			sev = SeverityWarning
			ruleID = "razify/placeholder-value"
		default:
			sev = SeverityNote
		}

		diags = append(diags, Diagnostic{
			RuleID:   ruleID,
			Message:  vr.Message,
			Severity: sev,
			File:     envFile,
			Line:     1, // Validation doesn't always have exact line
			Key:      vr.Key,
		})
	}

	for _, sr := range scanResults {
		sev := SeverityWarning
		ruleID := "razify/suspicious-secret"
		if sr.Risk == "CRITICAL" || sr.Risk == "HIGH" {
			sev = SeverityError
			ruleID = "razify/secret-leak"
		}
		diags = append(diags, Diagnostic{
			RuleID:   ruleID,
			Message:  sr.Reason,
			Severity: sev,
			File:     envFile,
			Line:     sr.Line,
			Key:      sr.Key,
			Value:    sr.Value,
		})
	}

	return Result{
		TargetFile:        envFile,
		TemplateFile:      exampleFile,
		Status:            status,
		Diagnostics:       diags,
		ValidationSummary: valSummary,
		SecretSummary:     scanSummary,
		RemediationSteps:  remediation,
	}
}

