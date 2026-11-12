package doctor

import "github.com/Hossiy21/razify/internal/reporter"

// CheckStatus represents the evaluation state of an individual diagnostic check.
type CheckStatus string

const (
	StatusPassed  CheckStatus = "PASSED"
	StatusFailed  CheckStatus = "FAILED"
	StatusWarning CheckStatus = "WARNING"
	StatusSkipped CheckStatus = "SKIPPED"
)

// CheckResult represents a single diagnostic finding or health check outcome.
type CheckResult struct {
	ID           string      `json:"id"`
	Category     string      `json:"category"` // "git", "env", "schema", "toolchain", "network", "service"
	Name         string      `json:"name"`
	Status       CheckStatus `json:"status"`
	Message      string      `json:"message"`
	Fix          string      `json:"fix,omitempty"`
	PointsEarned int         `json:"points_earned"`
	MaxPoints    int         `json:"max_points"`
	Target       string      `json:"target,omitempty"`
}

// DoctorReport aggregates all diagnostic check results and computes the overall health score.
type DoctorReport struct {
	HealthScore  int             `json:"health_score"`  // 0-100
	HealthStatus string          `json:"health_status"` // "HEALTHY", "DEGRADED", "UNHEALTHY"
	Checks       []CheckResult   `json:"checks"`
	Remediations []string        `json:"remediations"`
	PassedCount  int             `json:"passed_count"`
	FailedCount  int             `json:"failed_count"`
	WarningCount int             `json:"warning_count"`
	SkippedCount int             `json:"skipped_count"`
	Result       reporter.Result `json:"result"`
}

