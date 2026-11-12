package codegen

// Location represents a source file location where an environment variable is referenced.
type Location struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// ExtractedVar represents an environment variable extracted from source code.
type ExtractedVar struct {
	Name         string     `json:"name"`
	DefaultValue string     `json:"defaultValue,omitempty"`
	InferredType string     `json:"inferredType"` // "port", "url", "int", "bool", "email", "secret", "string"
	IsRequired   bool       `json:"isRequired"`
	Category     string     `json:"category"` // "Server", "Database", "Security", "Feature Flags", "General"
	Description  string     `json:"description,omitempty"`
	Occurrences  []Location `json:"occurrences"`
}

// ExtractOptions configures the code extractor.
type ExtractOptions struct {
	RootDir     string
	ExcludeDirs []string
}

// ExtractionReport contains the aggregated results from source code extraction.
type ExtractionReport struct {
	FilesScanned      int            `json:"filesScanned"`
	Variables         []ExtractedVar `json:"variables"`
	LanguagesDetected []string       `json:"languagesDetected"`
}
