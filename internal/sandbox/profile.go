package sandbox

import (
	"os"
	"path/filepath"
)

// ProfileConfig represents profile-specific settings.
type ProfileConfig struct {
	Name         string
	DefaultImage string
	EnvVars      map[string]string
	DefaultTmpfs []string
}

// DetectProfile inspects workspace files to determine the project profile.
func DetectProfile(workspaceDir string) string {
	if workspaceDir == "" {
		workspaceDir = "."
	}

	if fileExists(filepath.Join(workspaceDir, "go.mod")) {
		return "go"
	}
	if fileExists(filepath.Join(workspaceDir, "package.json")) {
		return "node"
	}
	if fileExists(filepath.Join(workspaceDir, "requirements.txt")) ||
		fileExists(filepath.Join(workspaceDir, "pyproject.toml")) {
		return "python"
	}

	return "generic"
}

// GetProfileConfig returns the profile configuration for a given profile name.
func GetProfileConfig(profileName string) ProfileConfig {
	switch profileName {
	case "go":
		return ProfileConfig{
			Name:         "go",
			DefaultImage: "golang:1.25-alpine",
			EnvVars: map[string]string{
				"GOCACHE":     "/cache/go/build",
				"GOMODCACHE":  "/cache/go/mod",
			},
			DefaultTmpfs: []string{
				"/tmp:rw,exec,nosuid,size=512m",
				"/cache:rw,exec,nosuid,size=1g",
				"/home/razify:rw,exec,nosuid,size=256m",
			},
		}
	case "node":
		return ProfileConfig{
			Name:         "node",
			DefaultImage: "node:22-alpine",
			EnvVars: map[string]string{
				"npm_config_cache":          "/cache/npm",
				"npm_config_update_notifier": "false",
			},
			DefaultTmpfs: []string{
				"/tmp:rw,exec,nosuid,size=512m",
				"/cache:rw,exec,nosuid,size=1g",
				"/home/razify:rw,exec,nosuid,size=256m",
			},
		}
	case "python":
		return ProfileConfig{
			Name:         "python",
			DefaultImage: "python:3.12-alpine",
			EnvVars: map[string]string{
				"PYTHONPYCACHEPREFIX": "/cache/python/pycache",
				"PIP_CACHE_DIR":       "/cache/python/pip",
			},
			DefaultTmpfs: []string{
				"/tmp:rw,exec,nosuid,size=512m",
				"/cache:rw,exec,nosuid,size=1g",
				"/home/razify:rw,exec,nosuid,size=256m",
			},
		}
	default:
		return ProfileConfig{
			Name:         "generic",
			DefaultImage: "alpine:latest",
			EnvVars:      map[string]string{},
			DefaultTmpfs: []string{
				"/tmp:rw,exec,nosuid,size=256m",
				"/home/razify:rw,exec,nosuid,size=256m",
			},
		}
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

