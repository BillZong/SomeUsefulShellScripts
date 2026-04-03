package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

type gitStatusSubdirScriptRepositoryResult struct {
	Path      string   `json:"path"`
	Branch    string   `json:"branch"`
	IsClean   bool     `json:"isClean"`
	Porcelain []string `json:"porcelain"`
}

type gitStatusSubdirScriptResult struct {
	OK           bool                                    `json:"ok"`
	Directory    string                                  `json:"directory"`
	Depth        int                                     `json:"depth"`
	Repositories []gitStatusSubdirScriptRepositoryResult `json:"repositories"`
}

func repoRootForScriptTests(t *testing.T) string {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}

	root := filepath.Clean(filepath.Join(cwd, "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "shell", "git-status-subdir.sh")); err != nil {
		t.Fatalf("locate git-status-subdir.sh: %v", err)
	}

	return root
}

func runCommand(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()

	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %v failed: %v\n%s", name, args, err, string(output))
	}

	return string(output)
}

func runGitStatusSubdirScript(t *testing.T, directory string, depth int) gitStatusSubdirScriptResult {
	t.Helper()

	scriptPath := filepath.Join(repoRootForScriptTests(t), "shell", "git-status-subdir.sh")
	cmd := exec.Command("bash", scriptPath, "--json", "--directory", directory, "--depth", strconv.Itoa(depth))

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git-status-subdir.sh failed: %v\n%s", err, string(output))
	}

	var result gitStatusSubdirScriptResult
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode script output: %v\n%s", err, string(output))
	}

	return result
}

func commitFixtureFile(t *testing.T, repoDir string, filename string, content string) {
	t.Helper()

	runCommand(t, repoDir, "git", "config", "user.name", "tester")
	runCommand(t, repoDir, "git", "config", "user.email", "tester@example.com")

	if err := os.WriteFile(filepath.Join(repoDir, filename), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	runCommand(t, repoDir, "git", "add", filename)
	runCommand(t, repoDir, "git", "commit", "-m", "init")
}

func TestGitStatusSubdirScriptFindsRepoAtExactDepth(t *testing.T) {
	tempDir := t.TempDir()
	rootDir := filepath.Join(tempDir, "root")
	repoDir := filepath.Join(rootDir, "level1", "level2")

	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatalf("mkdir repo dir: %v", err)
	}

	runCommand(t, repoDir, "git", "init")
	commitFixtureFile(t, repoDir, "file.txt", "exact depth\n")

	result := runGitStatusSubdirScript(t, rootDir, 2)
	if len(result.Repositories) != 1 {
		t.Fatalf("expected one repository, got %#v", result.Repositories)
	}

	if result.Repositories[0].Path != repoDir {
		t.Fatalf("unexpected repository path: %s", result.Repositories[0].Path)
	}
}

func TestGitStatusSubdirScriptFindsWorktreeGitFile(t *testing.T) {
	tempDir := t.TempDir()
	rootDir := filepath.Join(tempDir, "root")
	baseRepoDir := filepath.Join(rootDir, "base")
	worktreeDir := filepath.Join(rootDir, "wt")

	if err := os.MkdirAll(baseRepoDir, 0o755); err != nil {
		t.Fatalf("mkdir base repo dir: %v", err)
	}

	runCommand(t, baseRepoDir, "git", "init")
	commitFixtureFile(t, baseRepoDir, "file.txt", "base repo\n")
	runCommand(t, baseRepoDir, "git", "worktree", "add", worktreeDir, "-b", "feature")

	result := runGitStatusSubdirScript(t, rootDir, 1)
	if len(result.Repositories) != 2 {
		t.Fatalf("expected two repositories, got %#v", result.Repositories)
	}

	paths := map[string]bool{}
	for _, repository := range result.Repositories {
		paths[repository.Path] = true
	}

	if !paths[baseRepoDir] {
		t.Fatalf("base repository missing from result: %#v", result.Repositories)
	}
	if !paths[worktreeDir] {
		t.Fatalf("worktree repository missing from result: %#v", result.Repositories)
	}
}
