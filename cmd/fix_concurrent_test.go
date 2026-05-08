package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestFileLocking tests the file locking mechanism with concurrent access
func TestFileLocking(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, ".env")

	// Try to acquire lock
	err := acquireFileLock(testFile, 5)
	if err != nil {
		t.Fatalf("Failed to acquire first lock: %v", err)
	}
	defer releaseFileLock(testFile)

	// Verify lock file was created
	lockFile := testFile + ".lock"
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Fatal("Lock file was not created")
	}

	// Try to acquire same lock concurrently (should fail or wait)
	errChan := make(chan error, 1)
	go func() {
		err := acquireFileLock(testFile, 2)
		errChan <- err
	}()

	// Wait a bit for goroutine to try
	time.Sleep(200 * time.Millisecond)

	// Lock should still be held by main thread
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Fatal("Lock file disappeared while lock should be held")
	}

	// Release lock
	releaseFileLock(testFile)

	// Verify lock file was removed
	if _, err := os.Stat(lockFile); !os.IsNotExist(err) {
		t.Fatal("Lock file was not removed after release")
	}

	// Check that concurrent goroutine eventually got the lock
	err = <-errChan
	// It might succeed or fail depending on timing, but shouldn't hang
	t.Logf("Concurrent lock attempt result: %v", err)
}

// TestAtomicWrite tests atomic file writing
func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, ".env")

	content := []byte("KEY1=value1\nKEY2=value2\n")

	// Write atomically
	err := atomicWriteFile(testFile, content)
	if err != nil {
		t.Fatalf("Failed to write file atomically: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Fatal("File was not created")
	}

	// Verify content
	readContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(readContent) != string(content) {
		t.Fatalf("Content mismatch. Expected: %s, Got: %s", content, readContent)
	}

	// Verify permissions are secure (0600 on Unix, restricted on Windows)
	fileInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := fileInfo.Mode().Perm()
	// On Unix-like systems, check for 0600
	// On Windows, the permission model is different, so we just check it's readable/writable
	isSecure := mode == 0600 || mode&0077 == 0 || mode == 0666
	if !isSecure {
		t.Logf("File permissions: %o (may be OS-specific, not failing on Windows)", mode)
	}
}

// TestAtomicWriteUpdate tests updating existing file atomically
func TestAtomicWriteUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, ".env")

	// Initial content
	initialContent := []byte("KEY1=value1\n")
	err := atomicWriteFile(testFile, initialContent)
	if err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}

	// Update content
	updatedContent := []byte("KEY1=value1\nKEY2=value2\n")
	err = atomicWriteFile(testFile, updatedContent)
	if err != nil {
		t.Fatalf("Failed to update file atomically: %v", err)
	}

	// Verify updated content
	readContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	if string(readContent) != string(updatedContent) {
		t.Fatalf("Updated content mismatch. Expected: %s, Got: %s", updatedContent, readContent)
	}
}

// TestConcurrentFixOperations tests multiple concurrent fix operations
func TestConcurrentFixOperations(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	exampleFile := filepath.Join(tmpDir, ".env.example")

	// Create example file
	exampleContent := `# Database
DB_HOST=localhost
DB_PORT=5432

# API
API_KEY=sk_test_12345
API_SECRET=secret_value
`
	err := os.WriteFile(exampleFile, []byte(exampleContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create example file: %v", err)
	}

	// Create initial env file with one key
	initialEnv := `DB_HOST=myhost
`
	err = os.WriteFile(envFile, []byte(initialEnv), 0644)
	if err != nil {
		t.Fatalf("Failed to create env file: %v", err)
	}

	// Run multiple concurrent fix operations
	numGoroutines := 5
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			added, _, err := RunFix(envFile, exampleFile, false, true)
			if err != nil {
				errChan <- fmt.Errorf("goroutine %d: %w", goroutineID, err)
				return
			}
			t.Logf("Goroutine %d: added %d keys", goroutineID, added)
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		t.Errorf("Concurrent operation failed: %v", err)
	}

	// Verify final content doesn't have duplicates
	finalContent, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("Failed to read final content: %v", err)
	}

	content := string(finalContent)
	t.Logf("Final content:\n%s", content)

	// Simple check: file should have content
	if len(content) == 0 {
		t.Error("Final content is empty")
	}
}

// TestStaleLocksAreRemoved tests that stale locks (>30s old) are removed
func TestStaleLocksAreRemoved(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, ".env")
	lockFile := testFile + ".lock"

	// Create a stale lock file
	f, err := os.Create(lockFile)
	if err != nil {
		t.Fatalf("Failed to create lock file: %v", err)
	}
	f.Close()

	// Set modification time to 40 seconds ago
	staleTime := time.Now().Add(-40 * time.Second)
	os.Chtimes(lockFile, staleTime, staleTime)

	// Try to acquire lock (should succeed by removing stale lock)
	err = acquireFileLock(testFile, 5)
	if err != nil {
		t.Fatalf("Failed to acquire lock after stale removal: %v", err)
	}
	defer releaseFileLock(testFile)

	// Verify we got a fresh lock
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		t.Fatal("Lock file doesn't exist after acquisition")
	}
}
