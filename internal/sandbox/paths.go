package sandbox

import (
	"fmt"
	"path/filepath"
)

// ToDockerMountPath normalizes a host path for use in Docker CLI -v volume mount flags.
// Converts Windows backslashes to forward slashes.
func ToDockerMountPath(hostPath string) string {
	cleaned := filepath.Clean(hostPath)
	return filepath.ToSlash(cleaned)
}

// FormatMountArg constructs a Docker -v volume mount argument.
func FormatMountArg(hostPath, containerPath, mode string) string {
	normalizedHost := ToDockerMountPath(hostPath)
	if mode != "" {
		return fmt.Sprintf("%s:%s:%s", normalizedHost, containerPath, mode)
	}
	return fmt.Sprintf("%s:%s", normalizedHost, containerPath)
}

