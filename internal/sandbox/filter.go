package sandbox

import (
	"strings"

	"github.com/Hossiy21/razify/internal/scanner"
)

// FilterEnv filters a given map of environment variables so only schema-approved keys are included.
func FilterEnv(sourceEnv map[string]string, approvedKeys []string) map[string]string {
	approvedMap := make(map[string]bool, len(approvedKeys))
	for _, k := range approvedKeys {
		approvedMap[k] = true
	}

	filtered := make(map[string]string)
	for k, v := range sourceEnv {
		if approvedMap[k] {
			filtered[k] = v
		}
	}
	return filtered
}

// MaskSecretsInEnv returns a copy of the environment map where secret values are masked for safe logging.
func MaskSecretsInEnv(env map[string]string, secretKeys []string) map[string]string {
	secretMap := make(map[string]bool, len(secretKeys))
	for _, k := range secretKeys {
		secretMap[strings.ToUpper(k)] = true
	}

	result := make(map[string]string, len(env))
	for k, v := range env {
		if secretMap[strings.ToUpper(k)] {
			result[k] = scanner.MaskValue(v)
		} else {
			result[k] = v
		}
	}
	return result
}

