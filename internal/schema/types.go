package schema

// EnvVar represents an environment variable with associated comments, tags, and metadata.
type EnvVar struct {
	Key      string            `json:"key"`
	Value    string            `json:"value"`
	Comment  string            `json:"comment"`
	Required bool              `json:"required"`
	Category string            `json:"category"`
	Tags     map[string]string `json:"tags"`
	LineNum  int               `json:"line_num"`
	Ignored  bool              `json:"ignored"`
}

// ValidationResult represents a diagnostic result from validating a single environment variable.
type ValidationResult struct {
	Key     string `json:"key"`
	Status  string `json:"status"` // "OK", "MISSING", "EMPTY", "PLACEHOLDER", "INVALID"
	Message string `json:"message"`
}

// ValidateSummary contains aggregated counts for validation findings.
type ValidateSummary struct {
	Passed      int `json:"passed"`
	Missing     int `json:"missing"`
	Empty       int `json:"empty"`
	Placeholder int `json:"placeholder"`
}

