package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFix_BackupCreationAndPermissions(t *testing.T) {
	tempDir := t.TempDir()

	envFile := filepath.Join(tempDir, ".env")
	exampleFile := filepath.Join(tempDir, ".env.example")
	gitignoreFile := filepath.Join(tempDir, ".gitignore")

	_ = os.WriteFile(envFile, []byte("EXISTING_KEY=secret_value_123\n"), 0600)
	_ = os.WriteFile(exampleFile, []byte("EXISTING_KEY=\nNEW_REQUIRED_KEY=default_val # @required\n"), 0644)
	_ = os.WriteFile(gitignoreFile, []byte("# Existing gitignore\nnode_modules/\n"), 0644)

	// 1. Run fix with backup enabled (default)
	added, backupPath, err := RunFix(envFile, exampleFile, false, false)
	if err != nil {
		t.Fatalf("RunFix failed: %v", err)
	}

	if added != 1 {
		t.Errorf("expected 1 key added, got %d", added)
	}

	if backupPath == "" {
		t.Fatalf("expected backupPath to be returned, got empty string")
	}

	// Verify backup file exists
	stat, err := os.Stat(backupPath)
	if err != nil {
		t.Fatalf("failed to stat backup file %s: %v", backupPath, err)
	}

	// Verify permissions (0600 owner read/write on Unix)
	if runtime.GOOS != "windows" {
		perm := stat.Mode().Perm()
		if perm != 0600 {
			t.Errorf("expected backup file permission 0600, got %o", perm)
		}
	}

	// Verify backup content matches pre-fix original .env
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}
	if !strings.Contains(string(backupContent), "EXISTING_KEY=secret_value_123") {
		t.Errorf("backup does not contain original content: %s", string(backupContent))
	}
	if strings.Contains(string(backupContent), "NEW_REQUIRED_KEY") {
		t.Errorf("backup unexpectedly contains new key: %s", string(backupContent))
	}

	// Verify .gitignore was updated with *.backup_* pattern
	gitignoreContent, err := os.ReadFile(gitignoreFile)
	if err != nil {
		t.Fatalf("failed to read .gitignore: %v", err)
	}
	if !strings.Contains(string(gitignoreContent), "*.backup_*") {
		t.Errorf(".gitignore was not updated to ignore backup files: %s", string(gitignoreContent))
	}

	// 2. Test noBackup flag
	_ = os.WriteFile(exampleFile, []byte("EXISTING_KEY=\nNEW_REQUIRED_KEY=\nANOTHER_KEY=val\n"), 0644)
	added2, backupPath2, err := RunFix(envFile, exampleFile, false, true)
	if err != nil {
		t.Fatalf("RunFix with noBackup failed: %v", err)
	}
	if added2 != 1 {
		t.Errorf("expected 1 key added, got %d", added2)
	}
	if backupPath2 != "" {
		t.Errorf("expected empty backupPath when noBackup=true, got %s", backupPath2)
	}
}
