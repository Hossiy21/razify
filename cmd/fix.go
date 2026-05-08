package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// fileLocks tracks locks per file to prevent concurrent modifications
var fileLocks = make(map[string]*sync.Mutex)
var locksMutex = &sync.Mutex{}

// getFileLock returns a mutex for a specific file (creates one if it doesn't exist)
func getFileLock(filename string) *sync.Mutex {
	locksMutex.Lock()
	defer locksMutex.Unlock()

	if _, exists := fileLocks[filename]; !exists {
		fileLocks[filename] = &sync.Mutex{}
	}
	return fileLocks[filename]
}

// acquireFileLock attempts to acquire a lock on a file with retries
func acquireFileLock(filename string, maxRetries int) error {
	lockFile := filename + ".lock"
	retries := 0

	for retries < maxRetries {
		// Try to create lock file exclusively (atomic operation)
		f, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err == nil {
			f.Close()
			return nil
		}

		// Check if lock file is stale (older than 30 seconds)
		if os.IsExist(err) {
			if info, err := os.Stat(lockFile); err == nil {
				if time.Since(info.ModTime()) > 30*time.Second {
					// Lock is stale, remove it
					os.Remove(lockFile)
					retries = 0
					continue
				}
			}
			retries++
			if retries < maxRetries {
				time.Sleep(100 * time.Millisecond)
			}
			continue
		}

		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	return fmt.Errorf("could not acquire lock on %s after %d attempts", filename, maxRetries)
}

// releaseFileLock removes the lock file for a file
func releaseFileLock(filename string) error {
	lockFile := filename + ".lock"
	if err := os.Remove(lockFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}

// atomicWriteFile writes content to a file atomically (write to temp, then rename)
func atomicWriteFile(filename string, content []byte) error {
	// Create temp file in same directory to ensure same filesystem
	dir := filepath.Dir(filename)
	if dir == "" {
		dir = "."
	}

	tmpFile, err := os.CreateTemp(dir, ".tmp_razify_")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up if something fails

	// Write content to temp file
	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Set proper permissions (0600 = owner read/write only)
	if err := tmpFile.Chmod(0600); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename (cross-platform)
	if err := os.Rename(tmpPath, filename); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

var fixCmd = &cobra.Command{
	Use:   "fix [env-file] [example-file]",
	Short: "Sync missing keys from your .env.example to your .env file",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		envFile := args[0]
		exampleFile := args[1]
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		noBackup, _ := cmd.Flags().GetBool("no-backup")

		fmt.Printf("\nFixing %s using template %s...\n\n", envFile, exampleFile)

		added, backup, err := RunFix(envFile, exampleFile, dryRun, noBackup)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		success := color.New(color.FgGreen, color.Bold)
		if added > 0 {
			if dryRun {
				fmt.Printf("\nDry run: %d keys would be added.\n", added)
			} else {
				success.Printf("\n✔ Successfully added %d missing keys to %s!\n", added, envFile)
				if backup != "" {
					fmt.Printf("  Backup saved to: %s\n", backup)
				}
			}
		} else {
			success.Println("\n✔ No missing keys found. Your .env is already in sync!")
		}
	},
}

func RunFix(envFile, exampleFile string, dryRun, noBackup bool) (int, string, error) {
	// 1. Get existing keys in .env
	envVars, err := parseEnvFile(envFile)
	if err != nil && !os.IsNotExist(err) {
		return 0, "", err
	}
	if os.IsNotExist(err) {
		envVars = make(map[string]string)
	}

	// 2. Get keys from example
	exampleVars, err := ParseEnvWithMetadata(exampleFile)
	if err != nil {
		return 0, "", err
	}

	// 3. Find missing keys
	var toAdd []EnvVar
	for _, ev := range exampleVars {
		if _, exists := envVars[ev.Key]; !exists {
			toAdd = append(toAdd, ev)
		}
	}

	if len(toAdd) == 0 {
		return 0, "", nil
	}

	if dryRun {
		for _, ev := range toAdd {
			fmt.Printf("  + Would add: %s (default: %s)\n", ev.Key, ev.Value)
		}
		return len(toAdd), "", nil
	}

	// 4. Acquire file lock to prevent concurrent modifications
	if err := acquireFileLock(envFile, 5); err != nil {
		return 0, "", fmt.Errorf("cannot acquire lock on %s: %v", envFile, err)
	}
	defer releaseFileLock(envFile)

	// 5. Create backup before modifying
	backupPath := ""
	if !noBackup && fileExists(envFile) {
		backupPath, err = createBackup(envFile)
		if err != nil {
			return 0, "", fmt.Errorf("failed to create backup: %v", err)
		}
	}

	// 6. Read current content and build updated content
	var currentContent []byte
	if fileExists(envFile) {
		var readErr error
		currentContent, readErr = os.ReadFile(envFile)
		if readErr != nil {
			return 0, backupPath, fmt.Errorf("failed to read %s: %w", envFile, readErr)
		}
	}

	// 7. Build new content
	newContent := currentContent
	if len(newContent) > 0 && !bytes.HasSuffix(newContent, []byte("\n")) {
		newContent = append(newContent, '\n')
	}

	if len(newContent) > 0 {
		newContent = append(newContent, []byte("# --- Added by Razify Fix ---\n")...)
	}

	for _, ev := range toAdd {
		line := fmt.Sprintf("%s=%s\n", ev.Key, ev.Value)
		newContent = append(newContent, []byte(line)...)
		fmt.Printf("  + Added: %s\n", ev.Key)
	}

	// 8. Write atomically (temp file + rename)
	if err := atomicWriteFile(envFile, newContent); err != nil {
		return 0, backupPath, err
	}

	return len(toAdd), backupPath, nil
}

func init() {
	rootCmd.AddCommand(fixCmd)
	fixCmd.Flags().BoolP("dry-run", "d", false, "Show what would be added without modifying the file")
	fixCmd.Flags().Bool("no-backup", false, "Disable automatic backup before fixing")
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// createBackup creates a timestamped backup of the given file with secure permissions
func createBackup(filename string) (string, error) {
	// Read original file
	source, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer source.Close()

	// Create backup filename with timestamp
	dir := filepath.Dir(filename)
	base := filepath.Base(filename)
	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(dir, fmt.Sprintf("%s.backup_%s", base, timestamp))

	// Create backup file with secure permissions (0600 = owner read/write only)
	dest, err := os.OpenFile(backupPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	defer dest.Close()

	// Copy content
	if _, err := io.Copy(dest, source); err != nil {
		return "", err
	}

	return backupPath, nil
}
