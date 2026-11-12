package cmd

import (
	"github.com/Hossiy21/razify/internal/schema"
)

// EnvVar aliases schema.EnvVar for backward compatibility.
type EnvVar = schema.EnvVar

func parseEnvFile(filename string) (map[string]string, error) {
	return schema.ParseEnvFile(filename)
}

func ParseEnvWithMetadata(filename string) ([]EnvVar, error) {
	return schema.ParseEnvWithMetadata(filename)
}
