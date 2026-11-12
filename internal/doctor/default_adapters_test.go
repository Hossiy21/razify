package doctor_test

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Hossiy21/razify/internal/doctor"
)

func TestDefaultDependencies_Initialization(t *testing.T) {
	deps := doctor.NewDefaultDependencies()
	if deps.Runner == nil || deps.Git == nil || deps.PortChecker == nil || deps.Prober == nil || deps.FS == nil {
		t.Fatalf("NewDefaultDependencies returned uninitialized adapter fields: %+v", deps)
	}
}

func TestOSToolchainRunner(t *testing.T) {
	runner := &doctor.OSToolchainRunner{}

	// Test LookPath
	path, err := runner.LookPath("go")
	if err != nil || path == "" {
		t.Fatalf("LookPath('go') failed: %v", err)
	}

	// Test Run
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := runner.Run(ctx, "go", "version")
	if err != nil {
		t.Fatalf("runner.Run('go version') failed: %v", err)
	}
	if !strings.Contains(out, "go version") {
		t.Errorf("expected 'go version' in output, got: %s", out)
	}

	// Test non-existent binary
	_, err = runner.Run(ctx, "non_existent_binary_12345")
	if err == nil {
		t.Errorf("expected error running non-existent binary, got nil")
	}
}

func TestNetPortChecker(t *testing.T) {
	checker := &doctor.NetPortChecker{}

	// Listen on dynamic loopback port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on dynamic port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	// CheckPort should report port is in use
	inUse, err := checker.CheckPort(port)
	if err != nil {
		t.Fatalf("CheckPort failed: %v", err)
	}
	if !inUse {
		t.Errorf("expected port %d to be in use, got inUse=false", port)
	}

	// Close listener
	_ = ln.Close()

	// Brief pause for OS to release socket
	time.Sleep(20 * time.Millisecond)

	// CheckPort should now report port is free
	inUse, err = checker.CheckPort(port)
	if err != nil {
		t.Fatalf("CheckPort after close failed: %v", err)
	}
	if inUse {
		t.Errorf("expected port %d to be available after close, got inUse=true", port)
	}
}

func TestNetServiceProber(t *testing.T) {
	prober := &doctor.NetServiceProber{}

	// Start local mock server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err == nil {
			_ = conn.Close()
		}
	}()

	addr := ln.Addr().String()

	// Probing open port should succeed
	err = prober.Probe("tcp", addr, 500*time.Millisecond)
	if err != nil {
		t.Errorf("expected Probe on active listener %s to succeed, got: %v", addr, err)
	}

	// Probing non-listening port should return error
	err = prober.Probe("tcp", "127.0.0.1:54321", 50*time.Millisecond)
	if err == nil {
		t.Errorf("expected Probe on closed port to fail, got nil")
	}
}

func TestOSGitChecker(t *testing.T) {
	tempGitDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tempGitDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Create .gitignore
	_ = os.WriteFile(filepath.Join(tempGitDir, ".gitignore"), []byte(".env\n.env.*\n"), 0644)

	// Create files
	_ = os.WriteFile(filepath.Join(tempGitDir, ".env"), []byte("SECRET=1\n"), 0600)
	trackedFile := filepath.Join(tempGitDir, "main.go")
	_ = os.WriteFile(trackedFile, []byte("package main\n"), 0644)

	addCmd := exec.Command("git", "add", "main.go")
	addCmd.Dir = tempGitDir
	_ = addCmd.Run()

	checker := &doctor.OSGitChecker{}
	ctx := context.Background()

	// Test from inside repo dir
	origDir, _ := os.Getwd()
	_ = os.Chdir(tempGitDir)
	defer os.Chdir(origDir)

	// CheckIgnore
	ignored, err := checker.CheckIgnore(ctx, ".env")
	if err != nil || !ignored {
		t.Errorf("expected .env to be ignored, got ignored=%v, err=%v", ignored, err)
	}

	notIgnored, err := checker.CheckIgnore(ctx, "main.go")
	if err != nil || notIgnored {
		t.Errorf("expected main.go not to be ignored, got ignored=%v, err=%v", notIgnored, err)
	}

	// IsTracked
	isTracked, err := checker.IsTracked(ctx, "main.go")
	if err != nil || !isTracked {
		t.Errorf("expected main.go to be tracked, got isTracked=%v, err=%v", isTracked, err)
	}

	notTracked, err := checker.IsTracked(ctx, "untracked_file.go")
	if err != nil || notTracked {
		t.Errorf("expected untracked_file.go not to be tracked, got isTracked=%v, err=%v", notTracked, err)
	}
}

func TestOSFileSystem(t *testing.T) {
	fs := &doctor.OSFileSystem{}
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	testData := []byte("doctor adapter test")
	_ = os.WriteFile(testFile, testData, 0644)

	// FileExists
	if !fs.FileExists(testFile) {
		t.Errorf("expected FileExists=true for %s", testFile)
	}
	if fs.FileExists(filepath.Join(tmpDir, "missing.txt")) {
		t.Errorf("expected FileExists=false for missing file")
	}

	// ReadFile
	readBytes, err := fs.ReadFile(testFile)
	if err != nil || string(readBytes) != string(testData) {
		t.Errorf("ReadFile failed: got %q, err %v", string(readBytes), err)
	}

	_, err = fs.ReadFile(filepath.Join(tmpDir, "missing.txt"))
	if err == nil {
		t.Errorf("expected error reading missing file, got nil")
	}
}
