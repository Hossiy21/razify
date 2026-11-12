package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/Hossiy21/razify/internal/scanner"
	"github.com/Hossiy21/razify/internal/schema"
)

var sensitiveKeyRegex = regexp.MustCompile(`(?i)(secret|password|passwd|pwd|token|auth|credential|api_key|apikey|private_key|privatekey|_key$|access_key)`)

// IsSecretClassified determines whether an environment variable is a secret based on tags, name, and value.
func IsSecretClassified(key, val string, tags map[string]string) bool {
	// 1. Explicit schema annotations
	if tags != nil {
		typeTag := strings.ToLower(tags["type"])
		if typeTag == "secret" || typeTag == "password" || typeTag == "key" || typeTag == "token" {
			return true
		}
		if _, ok := tags["secret"]; ok {
			return true
		}
	}

	// 2. Sensitive key name heuristics
	if sensitiveKeyRegex.MatchString(key) {
		// Exceptions for known non-secret configuration variables
		upperKey := strings.ToUpper(key)
		if upperKey == "KEY_COUNT" || upperKey == "PRIMARY_KEY" {
			return false
		}
		return true
	}

	// 3. Known secret value provider patterns
	if val != "" {
		for _, rule := range scanner.SecretValuePatterns {
			if rule.Pattern.MatchString(val) {
				return true
			}
		}

		// 4. Shannon entropy analysis (for strings of reasonable length)
		if len(val) >= 16 && scanner.ShannonEntropy(val) >= 3.5 {
			return true
		}
	}

	return false
}

// GenerateFakeSecret generates a safe, deterministic synthetic secret value conforming to the prefix requirements.
func GenerateFakeSecret(key, val, secretType string) string {
	hasher := sha256.New()
	hasher.Write([]byte(key))
	if val != "" {
		hasher.Write([]byte(val))
	}
	hashStr := hex.EncodeToString(hasher.Sum(nil))[:8]

	cleanKey := strings.ToLower(key)
	cleanKey = regexp.MustCompile(`[^a-z0-9_]`).ReplaceAllString(cleanKey, "_")
	cleanKey = strings.Trim(cleanKey, "_")
	if cleanKey == "" {
		cleanKey = "secret"
	}

	return "razify_fake_" + cleanKey + "_" + hashStr
}

// ProcessEnvForSandbox scopes and classifies environment variables from the schema and .env file.
func ProcessEnvForSandbox(
	exampleVars []schema.EnvVar,
	envValues map[string]string,
	onlyKeys []string,
	injectSecrets bool,
) (map[string]string, []string) {
	onlyMap := make(map[string]bool)
	for _, k := range onlyKeys {
		trimmed := strings.TrimSpace(k)
		if trimmed != "" {
			onlyMap[trimmed] = true
		}
	}

	injectedEnv := make(map[string]string)
	var rawSecretsToMask []string

	// If exampleVars is empty but envValues has keys, fallback to envValues keys
	type keyCandidate struct {
		key   string
		value string
		tags  map[string]string
	}

	var candidates []keyCandidate
	if len(exampleVars) > 0 {
		for _, ev := range exampleVars {
			val := ev.Value
			if actual, exists := envValues[ev.Key]; exists {
				val = actual
			}
			candidates = append(candidates, keyCandidate{
				key:   ev.Key,
				value: val,
				tags:  ev.Tags,
			})
		}
	} else {
		for k, v := range envValues {
			candidates = append(candidates, keyCandidate{
				key:   k,
				value: v,
				tags:  nil,
			})
		}
	}

	for _, c := range candidates {
		if len(onlyMap) > 0 && !onlyMap[c.key] {
			continue
		}

		isSecret := IsSecretClassified(c.key, c.value, c.tags)

		if isSecret {
			if actualVal, exists := envValues[c.key]; exists && actualVal != "" {
				rawSecretsToMask = append(rawSecretsToMask, actualVal)
			}

			if injectSecrets {
				injectedEnv[c.key] = c.value
			} else {
				injectedEnv[c.key] = GenerateFakeSecret(c.key, c.value, "secret")
			}
		} else {
			injectedEnv[c.key] = c.value
		}
	}

	return injectedEnv, rawSecretsToMask
}

