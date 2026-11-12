package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Hossiy21/razify/internal/scanner"
	"github.com/Hossiy21/razify/internal/schema"
)

// ListTools returns the catalog of safe MCP tools exposed to AI coding agents.
func ListTools() []Tool {
	return []Tool{
		{
			Name: "schema_inspect",
			Description: "Inspect the project environment schema from .env.example, including required keys, types (@type), enums (@enum), ranges (@range), and descriptions. Strictly schema metadata; never accesses raw .env secret files.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]PropertyDef{
					"example_file": {
						Type:        "string",
						Description: "Optional path to the template schema file (default: .env.example).",
					},
				},
			},
		},
		{
			Name: "env_validate_config",
			Description: "Validate a suggested candidate configuration (key-value map) against project schema rules without writing to disk. Returns type checks, enum/range validation, and missing dependencies.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]PropertyDef{
					"config": {
						Type:        "object",
						Description: "JSON object mapping environment variable keys to candidate string values.",
					},
					"example_file": {
						Type:        "string",
						Description: "Optional path to the template schema file (default: .env.example).",
					},
				},
				Required: []string{"config"},
			},
		},
		{
			Name: "env_diff_keys",
			Description: "Compare environment variable names between .env and .env.example to identify missing or untracked keys. Strictly compares key names only, never exposing values.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]PropertyDef{
					"env_file": {
						Type:        "string",
						Description: "Optional path to the .env file (default: .env).",
					},
					"example_file": {
						Type:        "string",
						Description: "Optional path to the template schema file (default: .env.example).",
					},
				},
			},
		},
		{
			Name: "env_health_audit",
			Description: "Perform an audit of environment configuration health (score 0-100, missing keys, and schema violations). Any secret findings have values masked or fingerprinted.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]PropertyDef{
					"env_file": {
						Type:        "string",
						Description: "Optional path to the .env file (default: .env).",
					},
					"example_file": {
						Type:        "string",
						Description: "Optional path to the template schema file (default: .env.example).",
					},
				},
			},
		},
	}
}

// Tool Handlers

type SchemaItem struct {
	Key         string            `json:"key"`
	Type        string            `json:"type"`
	Required    bool              `json:"required"`
	Category    string            `json:"category,omitempty"`
	Comment     string            `json:"comment,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Placeholder string            `json:"placeholder,omitempty"`
}

type SchemaInspectResult struct {
	ExampleFile string       `json:"example_file"`
	TotalKeys   int          `json:"total_keys"`
	Variables   []SchemaItem `json:"variables"`
}

func HandleSchemaInspect(argsRaw json.RawMessage, workspaceDir string) (CallToolResult, error) {
	var args struct {
		ExampleFile string `json:"example_file"`
	}
	if len(argsRaw) > 0 {
		_ = json.Unmarshal(argsRaw, &args)
	}

	exampleFile, err := ResolveConfinedPath(workspaceDir, args.ExampleFile, ".env.example")
	if err != nil {
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		}, nil
	}

	vars, err := schema.ParseEnvWithMetadata(exampleFile)
	if err != nil {
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error reading schema file %s: %s", filepath.Base(exampleFile), SanitizeParseError(err))}},
			IsError: true,
		}, nil
	}

	items := make([]SchemaItem, 0, len(vars))
	for _, v := range vars {
		inferredType := "string"
		if t, ok := v.Tags["type"]; ok && t != "" {
			inferredType = t
		}

		placeholder := v.Value
		// If placeholder looks like a real or synthetic secret, redact it
		if IsSecretPlaceholder(v.Key, placeholder, v.Tags) {
			placeholder = "[REDACTED_SECRET_PLACEHOLDER]"
		}

		items = append(items, SchemaItem{
			Key:         v.Key,
			Type:        inferredType,
			Required:    v.Required,
			Category:    v.Category,
			Comment:     v.Comment,
			Tags:        v.Tags,
			Placeholder: placeholder,
		})
	}

	result := SchemaInspectResult{
		ExampleFile: filepath.ToSlash(exampleFile),
		TotalKeys:   len(items),
		Variables:   items,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return CallToolResult{IsError: true}, err
	}

	return CallToolResult{
		Content: []ToolContent{{Type: "text", Text: string(data)}},
		IsError: false,
	}, nil
}

type ConfigValidationFinding struct {
	Key     string `json:"key"`
	Status  string `json:"status"` // "INVALID", "MISSING", "SECRET_LEAK"
	Message string `json:"message"`
}

type ConfigValidationResult struct {
	Valid        bool                      `json:"valid"`
	PassedCount  int                       `json:"passed_count"`
	InvalidCount int                       `json:"invalid_count"`
	Findings     []ConfigValidationFinding `json:"findings"`
}

func HandleEnvValidateConfig(argsRaw json.RawMessage, workspaceDir string) (CallToolResult, error) {
	var args struct {
		Config      map[string]string `json:"config"`
		ExampleFile string            `json:"example_file"`
	}
	if err := json.Unmarshal(argsRaw, &args); err != nil {
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Invalid arguments: %v", err)}},
			IsError: true,
		}, nil
	}

	exampleFile, err := ResolveConfinedPath(workspaceDir, args.ExampleFile, ".env.example")
	if err != nil {
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		}, nil
	}

	exampleVars, err := schema.ParseEnvWithMetadata(exampleFile)
	if err != nil {
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error reading schema file %s: %s", filepath.Base(exampleFile), SanitizeParseError(err))}},
			IsError: true,
		}, nil
	}

	findings := make([]ConfigValidationFinding, 0)
	passedCount := 0

	// 1. Check required variables
	for _, ev := range exampleVars {
		val, exists := args.Config[ev.Key]
		if !exists || strings.TrimSpace(val) == "" {
			if ev.Required {
				findings = append(findings, ConfigValidationFinding{
					Key:     ev.Key,
					Status:  "MISSING",
					Message: fmt.Sprintf("Required variable %s is missing from config", ev.Key),
				})
			}
			continue
		}

		// 2. Validate value against tags
		if errMsg := schema.ValidateValue(val, ev.Tags, args.Config); errMsg != "" {
			findings = append(findings, ConfigValidationFinding{
				Key:     ev.Key,
				Status:  "INVALID",
				Message: errMsg,
			})
			continue
		}

		// 3. Secret leak scanning on submitted candidate value
		isSecret := false
		for _, r := range scanner.SecretValuePatterns {
			if r.Pattern.MatchString(val) {
				findings = append(findings, ConfigValidationFinding{
					Key:     ev.Key,
					Status:  "SECRET_LEAK",
					Message: fmt.Sprintf("Submitted candidate value matches known secret pattern (%s). Masked value: %s", r.Reason, scanner.MaskValue(val)),
				})
				isSecret = true
				break
			}
		}

		if !isSecret {
			passedCount++
		}
	}

	result := ConfigValidationResult{
		Valid:        len(findings) == 0,
		PassedCount:  passedCount,
		InvalidCount: len(findings),
		Findings:     findings,
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	return CallToolResult{
		Content: []ToolContent{{Type: "text", Text: string(data)}},
		IsError: false,
	}, nil
}

type DiffKeysResult struct {
	EnvFile          string   `json:"env_file"`
	ExampleFile      string   `json:"example_file"`
	MissingKeys      []string `json:"missing_keys"`
	UntrackedKeys    []string `json:"untracked_keys"`
	SynchronizedKeys []string `json:"synchronized_keys"`
}

func HandleEnvDiffKeys(argsRaw json.RawMessage, workspaceDir string) (CallToolResult, error) {
	var args struct {
		EnvFile     string `json:"env_file"`
		ExampleFile string `json:"example_file"`
	}
	if len(argsRaw) > 0 {
		_ = json.Unmarshal(argsRaw, &args)
	}

	envFile := args.EnvFile
	envFile, err := ResolveConfinedPath(workspaceDir, args.EnvFile, ".env")
	if err != nil {
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		}, nil
	}

	exampleFile, err := ResolveConfinedPath(workspaceDir, args.ExampleFile, ".env.example")
	if err != nil {
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		}, nil
	}

	envMap, err := schema.ParseEnvFile(envFile)
	if err != nil && !os.IsNotExist(err) {
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error reading env file %s: %s", filepath.Base(envFile), SanitizeParseError(err))}},
			IsError: true,
		}, nil
	}

	exampleVars, err := schema.ParseEnvWithMetadata(exampleFile)
	if err != nil {
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error reading schema file %s: %s", filepath.Base(exampleFile), SanitizeParseError(err))}},
			IsError: true,
		}, nil
	}

	exampleMap := make(map[string]bool)
	for _, ev := range exampleVars {
		exampleMap[ev.Key] = true
	}

	var missingKeys []string
	var synchronizedKeys []string
	var untrackedKeys []string

	for k := range exampleMap {
		if _, exists := envMap[k]; exists {
			synchronizedKeys = append(synchronizedKeys, k)
		} else {
			missingKeys = append(missingKeys, k)
		}
	}

	for k := range envMap {
		if !exampleMap[k] {
			untrackedKeys = append(untrackedKeys, k)
		}
	}

	sort.Strings(missingKeys)
	sort.Strings(untrackedKeys)
	sort.Strings(synchronizedKeys)

	diffResult := DiffKeysResult{
		EnvFile:          filepath.ToSlash(envFile),
		ExampleFile:      filepath.ToSlash(exampleFile),
		MissingKeys:      missingKeys,
		UntrackedKeys:    untrackedKeys,
		SynchronizedKeys: synchronizedKeys,
	}

	data, _ := json.MarshalIndent(diffResult, "", "  ")
	return CallToolResult{
		Content: []ToolContent{{Type: "text", Text: string(data)}},
		IsError: false,
	}, nil
}

type HealthAuditSecretFinding struct {
	Key         string `json:"key"`
	Reason      string `json:"reason"`
	Risk        string `json:"risk"`
	MaskedValue string `json:"masked_value"`
	Fingerprint string `json:"fingerprint_sha256"`
}

type HealthAuditResult struct {
	HealthScore    int                        `json:"health_score"`
	Status         string                     `json:"status"` // "HEALTHY", "DEGRADED", "UNHEALTHY"
	TotalKeys      int                        `json:"total_keys"`
	MissingKeys    []string                   `json:"missing_keys"`
	SecretFindings []HealthAuditSecretFinding `json:"secret_findings"`
}

func HandleEnvHealthAudit(argsRaw json.RawMessage, workspaceDir string) (CallToolResult, error) {
	var args struct {
		EnvFile     string `json:"env_file"`
		ExampleFile string `json:"example_file"`
	}
	if len(argsRaw) > 0 {
		_ = json.Unmarshal(argsRaw, &args)
	}

	envFile, err := ResolveConfinedPath(workspaceDir, args.EnvFile, ".env")
	if err != nil {
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		}, nil
	}

	exampleFile, err := ResolveConfinedPath(workspaceDir, args.ExampleFile, ".env.example")
	if err != nil {
		return CallToolResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		}, nil
	}

	envMap, _ := schema.ParseEnvFile(envFile)
	exampleVars, _ := schema.ParseEnvWithMetadata(exampleFile)

	var missingKeys []string
	for _, ev := range exampleVars {
		if ev.Required {
			if _, exists := envMap[ev.Key]; !exists {
				missingKeys = append(missingKeys, ev.Key)
			}
		}
	}

	var secretFindings []HealthAuditSecretFinding
	for k, v := range envMap {
		for _, r := range scanner.SecretValuePatterns {
			if r.Pattern.MatchString(v) {
				hash := sha256.Sum256([]byte(v))
				secretFindings = append(secretFindings, HealthAuditSecretFinding{
					Key:         k,
					Reason:      r.Reason,
					Risk:        r.Risk,
					MaskedValue: scanner.MaskValue(v),
					Fingerprint: hex.EncodeToString(hash[:8]),
				})
				break
			}
		}
	}

	// Deterministic score calculation
	score := 100
	score -= len(missingKeys) * 15
	score -= len(secretFindings) * 25
	if score < 0 {
		score = 0
	}

	status := "HEALTHY"
	if score < 70 {
		status = "UNHEALTHY"
	} else if score < 90 {
		status = "DEGRADED"
	}

	auditResult := HealthAuditResult{
		HealthScore:    score,
		Status:         status,
		TotalKeys:      len(envMap),
		MissingKeys:    missingKeys,
		SecretFindings: secretFindings,
	}

	data, _ := json.MarshalIndent(auditResult, "", "  ")
	return CallToolResult{
		Content: []ToolContent{{Type: "text", Text: string(data)}},
		IsError: false,
	}, nil
}
