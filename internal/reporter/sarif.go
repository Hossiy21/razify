package reporter

import (
	"encoding/json"
	"path/filepath"
)

// SarifReport represents a SARIF v2.1.0 root document.
type SarifReport struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []SarifRun `json:"runs"`
}

// SarifRun represents a single tool invocation run.
type SarifRun struct {
	Tool    SarifTool     `json:"tool"`
	Results []SarifResult `json:"results"`
}

// SarifTool defines the driver and rules for the tool.
type SarifTool struct {
	Driver SarifDriver `json:"driver"`
}

// SarifDriver contains the tool metadata and rule definitions.
type SarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []SarifRule `json:"rules"`
}

// SarifRule defines metadata for an individual static analysis rule.
type SarifRule struct {
	ID               string               `json:"id"`
	ShortDescription SarifMultiformatText `json:"shortDescription"`
	FullDescription  SarifMultiformatText `json:"fullDescription,omitempty"`
}

// SarifResult represents an individual finding / alert.
type SarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"` // "error", "warning", "note"
	Message   SarifMessage    `json:"message"`
	Locations []SarifLocation `json:"locations,omitempty"`
}

// SarifMessage holds text for a diagnostic result.
type SarifMessage struct {
	Text string `json:"text"`
}

// SarifMultiformatText holds description text.
type SarifMultiformatText struct {
	Text string `json:"text"`
}

// SarifLocation points to where an issue was found.
type SarifLocation struct {
	PhysicalLocation SarifPhysicalLocation `json:"physicalLocation"`
}

// SarifPhysicalLocation contains the file path and line region.
type SarifPhysicalLocation struct {
	ArtifactLocation SarifArtifactLocation `json:"artifactLocation"`
	Region           *SarifRegion          `json:"region,omitempty"`
}

// SarifArtifactLocation contains the URI of the file.
type SarifArtifactLocation struct {
	URI string `json:"uri"`
}

// SarifRegion specifies the line range.
type SarifRegion struct {
	StartLine int `json:"startLine"`
}

var standardRules = []SarifRule{
	{
		ID:               "razify/missing-key",
		ShortDescription: SarifMultiformatText{Text: "Required environment variable missing"},
		FullDescription:  SarifMultiformatText{Text: "A variable marked as required in the schema template is absent from the target environment file."},
	},
	{
		ID:               "razify/invalid-type",
		ShortDescription: SarifMultiformatText{Text: "Environment variable type or constraint violation"},
		FullDescription:  SarifMultiformatText{Text: "The variable value does not satisfy the specified @type, @enum, @range, or @requires constraint."},
	},
	{
		ID:               "razify/empty-value",
		ShortDescription: SarifMultiformatText{Text: "Environment variable is empty"},
		FullDescription:  SarifMultiformatText{Text: "The variable exists but has no value assigned."},
	},
	{
		ID:               "razify/placeholder-value",
		ShortDescription: SarifMultiformatText{Text: "Placeholder value detected in environment file"},
		FullDescription:  SarifMultiformatText{Text: "The variable value appears to be an unconfigured example placeholder."},
	},
	{
		ID:               "razify/secret-leak",
		ShortDescription: SarifMultiformatText{Text: "Sensitive credential or secret key detected"},
		FullDescription:  SarifMultiformatText{Text: "A high-confidence secret (e.g. AWS key, Stripe live key, private key) was detected in plaintext."},
	},
	{
		ID:               "razify/suspicious-secret",
		ShortDescription: SarifMultiformatText{Text: "Suspicious high-entropy or token value detected"},
		FullDescription:  SarifMultiformatText{Text: "A high-entropy string or key matching danger patterns was detected."},
	},
}

// ToSarif converts a unified Result into a SARIF v2.1.0 document.
func ToSarif(res Result, version string) SarifReport {
	var sarifResults []SarifResult

	for _, d := range res.Diagnostics {
		level := "warning"
		switch d.Severity {
		case SeverityError:
			level = "error"
		case SeverityNote:
			level = "note"
		}

		line := d.Line
		if line < 1 {
			line = 1
		}

		msg := d.Message
		if d.Key != "" {
			msg = d.Key + ": " + d.Message
		}

		sarifResults = append(sarifResults, SarifResult{
			RuleID:  d.RuleID,
			Level:   level,
			Message: SarifMessage{Text: msg},
			Locations: []SarifLocation{
				{
					PhysicalLocation: SarifPhysicalLocation{
						ArtifactLocation: SarifArtifactLocation{
							URI: filepath.ToSlash(d.File),
						},
						Region: &SarifRegion{
							StartLine: line,
						},
					},
				},
			},
		})
	}

	if sarifResults == nil {
		sarifResults = []SarifResult{}
	}

	return SarifReport{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []SarifRun{
			{
				Tool: SarifTool{
					Driver: SarifDriver{
						Name:           "razify",
						Version:        version,
						InformationURI: "https://github.com/Hossiy21/razify",
						Rules:          standardRules,
					},
				},
				Results: sarifResults,
			},
		},
	}
}

// RenderSarif serializes the Result into indented SARIF JSON.
func RenderSarif(res Result, version string) (string, error) {
	report := ToSarif(res, version)
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

