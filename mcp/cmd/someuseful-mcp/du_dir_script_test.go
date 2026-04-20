package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type duDirScriptTestEntry struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	SizeHuman string `json:"sizeHuman"`
}

type duDirScriptTestResult struct {
	OK        bool                   `json:"ok"`
	Directory string                 `json:"directory"`
	Entries   []duDirScriptTestEntry `json:"entries"`
}

func runDuDirScript(t *testing.T, directory string) duDirScriptTestResult {
	t.Helper()

	scriptPath := filepath.Join(repoRootForScriptTests(t), "shell", "du-dir.sh")
	cmd := exec.Command("bash", scriptPath, "--json", "--directory", directory)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("du-dir.sh failed: %v\n%s", err, string(output))
	}

	var result duDirScriptTestResult
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode du-dir output: %v\n%s", err, string(output))
	}

	return result
}

func TestDuDirScriptReturnsEmptyEntriesForEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	result := runDuDirScript(t, tempDir)

	if !result.OK {
		t.Fatalf("expected ok result")
	}
	if len(result.Entries) != 0 {
		t.Fatalf("expected no entries, got %#v", result.Entries)
	}
}

func TestDuDirScriptIncludesHiddenAndPlainEntries(t *testing.T) {
	tempDir := t.TempDir()
	rootDir := filepath.Join(tempDir, "root")

	if err := os.MkdirAll(filepath.Join(rootDir, "subdir"), 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, ".hidden.txt"), []byte("hidden\n"), 0o644); err != nil {
		t.Fatalf("write hidden file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "plain.txt"), []byte("plain\n"), 0o644); err != nil {
		t.Fatalf("write plain file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "subdir", "nested.txt"), []byte("nested\n"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	result := runDuDirScript(t, rootDir)

	paths := map[string]bool{}
	for _, entry := range result.Entries {
		paths[entry.Path] = true
		if entry.SizeBytes <= 0 {
			t.Fatalf("expected positive size for %#v", entry)
		}
	}

	expectedPaths := []string{
		filepath.Join(rootDir, ".hidden.txt"),
		filepath.Join(rootDir, "plain.txt"),
		filepath.Join(rootDir, "subdir"),
	}
	for _, expectedPath := range expectedPaths {
		if !paths[expectedPath] {
			t.Fatalf("missing path %q in result %#v", expectedPath, result.Entries)
		}
	}
}
