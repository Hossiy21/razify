package cmd

import (
	"os"
	"testing"
)

func TestRunDiff(t *testing.T) {
	content1 := `
SAME=value
CHANGED=old
REMOVED=exists
`
	content2 := `
SAME=value
CHANGED=new
ADDED=newvar
`
	file1, _ := os.CreateTemp("", "file1.env")
	file2, _ := os.CreateTemp("", "file2.env")
	
	defer os.Remove(file1.Name())
	defer os.Remove(file2.Name())

	os.WriteFile(file1.Name(), []byte(content1), 0644)
	os.WriteFile(file2.Name(), []byte(content2), 0644)

	diffCount, results, err := RunDiff(file1.Name(), file2.Name())
	if err != nil {
		t.Fatalf("RunDiff failed: %v", err)
	}

	// 1 removed, 1 added, 1 changed = 3 diffs
	if diffCount != 3 {
		t.Errorf("Expected 3 differences, got %d", diffCount)
	}

	foundRemoved := false
	foundAdded := false
	foundChanged := false

	for _, r := range results {
		if r.Key == "REMOVED" && r.Type == "MISSING" {
			foundRemoved = true
		}
		if r.Key == "ADDED" && r.Type == "ADDED" {
			foundAdded = true
		}
		if r.Key == "CHANGED" && r.Type == "CHANGED" && r.OldValue == "old" && r.NewValue == "new" {
			foundChanged = true
		}
	}

	if !foundRemoved {
		t.Error("Did not detect REMOVED variable")
	}
	if !foundAdded {
		t.Error("Did not detect ADDED variable")
	}
	if !foundChanged {
		t.Error("Did not detect CHANGED variable")
	}
}
