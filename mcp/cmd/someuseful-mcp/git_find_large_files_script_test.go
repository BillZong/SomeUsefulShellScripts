package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type gitFindLargeFilesScriptTestFileResult struct {
	ObjectID  string `json:"object_id"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SizeHuman string `json:"size_human"`
}

type gitFindLargeFilesScriptTestResult struct {
	OK            bool                                    `json:"ok"`
	Directory     string                                  `json:"directory"`
	Limit         int                                     `json:"limit"`
	TotalCount    int                                     `json:"total_count"`
	ReturnedCount int                                     `json:"returned_count"`
	Truncated     bool                                    `json:"truncated"`
	Files         []gitFindLargeFilesScriptTestFileResult `json:"files"`
}

func runGitFindLargeFilesScript(t *testing.T, directory string) gitFindLargeFilesScriptTestResult {
	t.Helper()

	scriptPath := filepath.Join(repoRootForScriptTests(t), "shell", "git-find-large-files.sh")
	cmd := exec.Command("bash", scriptPath, "--json", "--directory", directory)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git-find-large-files.sh failed: %v\n%s", err, string(output))
	}

	var result gitFindLargeFilesScriptTestResult
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode script output: %v\n%s", err, string(output))
	}

	return result
}

func TestGitFindLargeFilesScriptPreservesSpecialPaths(t *testing.T) {
	tempDir := t.TempDir()

	runCommand(t, tempDir, "git", "init")
	runCommand(t, tempDir, "git", "config", "user.name", "tester")
	runCommand(t, tempDir, "git", "config", "user.email", "tester@example.com")

	specialNames := []string{
		"plain.txt",
		"space name.txt",
		"tab\tname.txt",
		"odd\nname.txt",
	}

	for index, name := range specialNames {
		content := []byte("payload-" + string(rune('a'+index)) + "\n")
		if err := os.WriteFile(filepath.Join(tempDir, name), content, 0o644); err != nil {
			t.Fatalf("write special file %q: %v", name, err)
		}
	}

	runCommand(t, tempDir, "git", "add", ".")
	runCommand(t, tempDir, "git", "-c", "commit.gpgsign=false", "commit", "-m", "init")

	result := runGitFindLargeFilesScript(t, tempDir)

	paths := map[string]bool{}
	for _, file := range result.Files {
		paths[file.Path] = true
	}

	for _, name := range specialNames {
		if !paths[name] {
			t.Fatalf("expected path %q in result, got %#v", name, result.Files)
		}
	}
}
