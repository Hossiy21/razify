package sandbox_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Hossiy21/razify/internal/sandbox"
)

func TestDetectProfile(t *testing.T) {
	tmpDir := t.TempDir()

	// Generic initially
	if got := sandbox.DetectProfile(tmpDir); got != "generic" {
		t.Errorf("expected generic profile, got %s", got)
	}

	// Go project
	_ = os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)
	if got := sandbox.DetectProfile(tmpDir); got != "go" {
		t.Errorf("expected go profile, got %s", got)
	}
	_ = os.Remove(filepath.Join(tmpDir, "go.mod"))

	// Node project
	_ = os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte("{}"), 0644)
	if got := sandbox.DetectProfile(tmpDir); got != "node" {
		t.Errorf("expected node profile, got %s", got)
	}
	_ = os.Remove(filepath.Join(tmpDir, "package.json"))

	// Python project
	_ = os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte("requests"), 0644)
	if got := sandbox.DetectProfile(tmpDir); got != "python" {
		t.Errorf("expected python profile, got %s", got)
	}
}

func TestGetProfileConfig(t *testing.T) {
	goProf := sandbox.GetProfileConfig("go")
	if goProf.DefaultImage != "golang:1.25-alpine" {
		t.Errorf("unexpected go default image: %s", goProf.DefaultImage)
	}
	if goProf.EnvVars["GOCACHE"] != "/cache/go/build" {
		t.Errorf("unexpected GOCACHE: %s", goProf.EnvVars["GOCACHE"])
	}

	nodeProf := sandbox.GetProfileConfig("node")
	if nodeProf.DefaultImage != "node:22-alpine" {
		t.Errorf("unexpected node default image: %s", nodeProf.DefaultImage)
	}

	pyProf := sandbox.GetProfileConfig("python")
	if pyProf.DefaultImage != "python:3.12-alpine" {
		t.Errorf("unexpected python default image: %s", pyProf.DefaultImage)
	}
}

