package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type ghDeleteOutdateActionsMatchedRun struct {
	ID         int64  `json:"id"`
	CreatedAt  string `json:"createdAt"`
	Name       string `json:"name"`
	HeadBranch string `json:"headBranch"`
}

type ghDeleteOutdateActionsResult struct {
	OK              bool                                `json:"ok"`
	Repository      string                              `json:"repository"`
	Owner           string                              `json:"owner"`
	Repo            string                              `json:"repo"`
	CutoffEpoch     int64                               `json:"cutoffEpoch"`
	DryRun          bool                                `json:"dryRun"`
	Mode            string                              `json:"mode"`
	Confirmed       bool                                `json:"confirmed"`
	MatchedRunCount int                                 `json:"matchedRunCount"`
	DeletedRunCount int                                 `json:"deletedRunCount"`
	MatchedRuns     []ghDeleteOutdateActionsMatchedRun  `json:"matchedRuns"`
	DeletedRunIDs   []int64                             `json:"deletedRunIds"`
}

func runGhDeleteOutdateActionsScript(t *testing.T, env []string, args ...string) (string, error) {
	t.Helper()

	scriptPath := filepath.Join(repoRootForScriptTests(t), "shell", "gh-delete-oudate-actions.sh")
	cmd := exec.Command("bash", append([]string{scriptPath}, args...)...)
	cmd.Env = append(os.Environ(), env...)

	output, err := cmd.CombinedOutput()
	return string(output), err
}

func decodeGhDeleteOutdateActionsResult(t *testing.T, output string) ghDeleteOutdateActionsResult {
	t.Helper()

	var result ghDeleteOutdateActionsResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode gh-delete-oudate-actions output: %v\n%s", err, output)
	}

	return result
}

func writeFakeGhFixture(t *testing.T, path string, lines ...string) {
	t.Helper()

	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}

func setupFakeGh(t *testing.T) ([]string, string, string) {
	t.Helper()

	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")
	fixtureDir := filepath.Join(tempDir, "fixtures")
	logPath := filepath.Join(tempDir, "gh.log")
	deleteLogPath := filepath.Join(tempDir, "gh-deletes.log")

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir fake gh bin dir: %v", err)
	}
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatalf("mkdir fake gh fixture dir: %v", err)
	}

	writeFakeGhFixture(t, filepath.Join(fixtureDir, "total_count.txt"), "101")
	writeFakeGhFixture(t, filepath.Join(fixtureDir, "page-1.txt"),
		"101|1700000000|2023-11-14T22:13:20Z|old-main|main",
		"102|1900000000|2030-03-17T17:46:40Z|new-main|main",
	)
	writeFakeGhFixture(t, filepath.Join(fixtureDir, "page-2.txt"),
		"103|1600000000|2020-09-13T12:26:40Z|legacy-cleanup|stale",
		"104|||unknown-created-at|mystery",
	)

	fakeGhScript := `#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$*" >> "$FAKE_GH_LOG"

[ "$#" -ge 1 ] || exit 1
[ "$1" = "api" ] || exit 1
shift

method="GET"
path=""
query=""
page="1"

while [ "$#" -gt 0 ]; do
  case "$1" in
    -X)
      method=$2
      shift 2
      ;;
    -q)
      query=$2
      shift 2
      ;;
    -F)
      case "$2" in
        page=*)
          page=${2#page=}
          ;;
      esac
      shift 2
      ;;
    -H)
      shift 2
      ;;
    /*)
      path=$1
      shift
      ;;
    *)
      shift
      ;;
  esac
done

if [ "$method" = "DELETE" ]; then
  printf '%s\n' "${path##*/}" >> "$FAKE_GH_DELETE_LOG"
  exit 0
fi

if [ "$query" = ".total_count" ]; then
  cat "$FAKE_GH_FIXTURE_DIR/total_count.txt"
  exit 0
fi

cutoff=$(printf '%s' "$query" | sed -n 's/.*fromdateiso8601) < \([0-9][0-9]*\).*/\1/p')
fixture="$FAKE_GH_FIXTURE_DIR/page-$page.txt"
[ -f "$fixture" ] || exit 0

while IFS='|' read -r id created_epoch created_at name branch; do
  [ -n "$id" ] || continue

  if [ -n "$created_epoch" ] && [ "$created_epoch" -lt "$cutoff" ]; then
    printf '%s\t%s\t%s\t%s\n' "$id" "$created_at" "$name" "$branch"
  fi
done < "$fixture"
`

	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte(fakeGhScript), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}

	env := []string{
		"PATH=" + binDir + ":" + os.Getenv("PATH"),
		"FAKE_GH_FIXTURE_DIR=" + fixtureDir,
		"FAKE_GH_LOG=" + logPath,
		"FAKE_GH_DELETE_LOG=" + deleteLogPath,
	}

	return env, logPath, deleteLogPath
}

func readLinesOrEmpty(t *testing.T, path string) []string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read %s: %v", path, err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return nil
	}

	return strings.Split(content, "\n")
}

func TestGhDeleteOutdateActionsScriptHelp(t *testing.T) {
	output, err := runGhDeleteOutdateActionsScript(t, nil, "--help")
	if err != nil {
		t.Fatalf("help command failed: %v\n%s", err, output)
	}

	expected := []string{
		"--json",
		"--dry-run",
		"--execute",
		"--yes",
		"--owner",
		"--repo",
		"--cutoff-epoch <unix-seconds>",
	}
	for _, fragment := range expected {
		if !strings.Contains(output, fragment) {
			t.Fatalf("help output missing %q:\n%s", fragment, output)
		}
	}
}

func TestGhDeleteOutdateActionsScriptValidatesRequiredArguments(t *testing.T) {
	testCases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "missing owner",
			args: []string{"--dry-run", "--repo", "demo", "--cutoff-epoch", "1700000000"},
			want: "missing required --owner",
		},
		{
			name: "missing repo",
			args: []string{"--dry-run", "--owner", "acme", "--cutoff-epoch", "1700000000"},
			want: "missing required --repo",
		},
		{
			name: "missing cutoff",
			args: []string{"--dry-run", "--owner", "acme", "--repo", "demo"},
			want: "missing required --cutoff-epoch",
		},
		{
			name: "invalid cutoff",
			args: []string{"--dry-run", "--owner", "acme", "--repo", "demo", "--cutoff-epoch", "not-a-number"},
			want: "--cutoff-epoch must be a non-negative integer",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := runGhDeleteOutdateActionsScript(t, nil, tc.args...)
			if err == nil {
				t.Fatalf("expected command to fail:\n%s", output)
			}
			if !strings.Contains(output, tc.want) {
				t.Fatalf("unexpected output, want %q:\n%s", tc.want, output)
			}
		})
	}
}

func TestGhDeleteOutdateActionsScriptDryRunFindsCandidatesWithoutDeleting(t *testing.T) {
	env, _, deleteLogPath := setupFakeGh(t)

	output, err := runGhDeleteOutdateActionsScript(
		t,
		env,
		"--json",
		"--dry-run",
		"--owner", "acme",
		"--repo", "widget",
		"--cutoff-epoch", "1800000000",
	)
	if err != nil {
		t.Fatalf("dry-run command failed: %v\n%s", err, output)
	}

	result := decodeGhDeleteOutdateActionsResult(t, output)
	if !result.OK {
		t.Fatalf("expected ok result: %#v", result)
	}
	if !result.DryRun || result.Mode != "dry-run" {
		t.Fatalf("unexpected mode fields: %#v", result)
	}
	if result.Confirmed {
		t.Fatalf("dry-run should not be confirmed: %#v", result)
	}
	if result.MatchedRunCount != 2 {
		t.Fatalf("unexpected matched count: %#v", result)
	}
	if result.DeletedRunCount != 0 {
		t.Fatalf("unexpected deleted count: %#v", result)
	}

	gotIDs := []int64{result.MatchedRuns[0].ID, result.MatchedRuns[1].ID}
	if gotIDs[0] != 101 || gotIDs[1] != 103 {
		t.Fatalf("unexpected matched ids: %#v", gotIDs)
	}

	if deleted := readLinesOrEmpty(t, deleteLogPath); len(deleted) != 0 {
		t.Fatalf("dry-run should not delete anything: %#v", deleted)
	}
}

func TestGhDeleteOutdateActionsScriptRefusesDefaultDestructiveMode(t *testing.T) {
	env, logPath, deleteLogPath := setupFakeGh(t)

	output, err := runGhDeleteOutdateActionsScript(
		t,
		env,
		"--owner", "acme",
		"--repo", "widget",
		"--cutoff-epoch", "1800000000",
	)
	if err == nil {
		t.Fatalf("expected refusal without --dry-run or --execute --yes:\n%s", output)
	}
	if !strings.Contains(output, "refusing destructive execution") {
		t.Fatalf("unexpected output: %s", output)
	}
	if logs := readLinesOrEmpty(t, logPath); len(logs) != 0 {
		t.Fatalf("default refusal should not call gh: %#v", logs)
	}
	if deletes := readLinesOrEmpty(t, deleteLogPath); len(deletes) != 0 {
		t.Fatalf("default refusal should not delete: %#v", deletes)
	}
}

func TestGhDeleteOutdateActionsScriptPaginatesAndFiltersCandidates(t *testing.T) {
	env, logPath, _ := setupFakeGh(t)

	output, err := runGhDeleteOutdateActionsScript(
		t,
		env,
		"--json",
		"--dry-run",
		"--owner", "acme",
		"--repo", "widget",
		"--cutoff-epoch", "1800000000",
	)
	if err != nil {
		t.Fatalf("dry-run command failed: %v\n%s", err, output)
	}

	result := decodeGhDeleteOutdateActionsResult(t, output)
	if result.MatchedRunCount != 2 {
		t.Fatalf("unexpected matched count: %#v", result)
	}
	for _, matchedRun := range result.MatchedRuns {
		if matchedRun.ID == 104 {
			t.Fatalf("null created_at run should not match cutoff deletion: %#v", result.MatchedRuns)
		}
	}

	logs := readLinesOrEmpty(t, logPath)
	var sawPageTwo bool
	for _, line := range logs {
		if strings.Contains(line, "page=2") {
			sawPageTwo = true
			break
		}
	}
	if !sawPageTwo {
		t.Fatalf("expected pagination to reach page 2: %#v", logs)
	}
}

func TestGhDeleteOutdateActionsScriptDeletesExpectedRunsInExecuteMode(t *testing.T) {
	env, _, deleteLogPath := setupFakeGh(t)

	output, err := runGhDeleteOutdateActionsScript(
		t,
		env,
		"--json",
		"--execute",
		"--yes",
		"--owner", "acme",
		"--repo", "widget",
		"--cutoff-epoch", "1800000000",
	)
	if err != nil {
		t.Fatalf("execute command failed: %v\n%s", err, output)
	}

	result := decodeGhDeleteOutdateActionsResult(t, output)
	if result.Mode != "execute" || !result.Confirmed || result.DryRun {
		t.Fatalf("unexpected execution mode: %#v", result)
	}
	if result.DeletedRunCount != 2 {
		t.Fatalf("unexpected deleted count: %#v", result)
	}

	deletes := readLinesOrEmpty(t, deleteLogPath)
	if len(deletes) != 2 {
		t.Fatalf("unexpected delete calls: %#v", deletes)
	}
	if deletes[0] != "101" || deletes[1] != "103" {
		t.Fatalf("unexpected deleted ids: %#v", deletes)
	}
	for _, deletedID := range deletes {
		if deletedID == "104" {
			t.Fatalf("null created_at run should not be deleted: %#v", deletes)
		}
	}
}

func TestGhDeleteOutdateActionsScriptJsonOutputShape(t *testing.T) {
	env, _, _ := setupFakeGh(t)

	output, err := runGhDeleteOutdateActionsScript(
		t,
		env,
		"--json",
		"--dry-run",
		"--owner", "acme",
		"--repo", "widget",
		"--cutoff-epoch", strconv.FormatInt(1800000000, 10),
	)
	if err != nil {
		t.Fatalf("json mode command failed: %v\n%s", err, output)
	}

	result := decodeGhDeleteOutdateActionsResult(t, output)
	if result.Repository != "acme/widget" || result.Owner != "acme" || result.Repo != "widget" {
		t.Fatalf("unexpected repository fields: %#v", result)
	}
	if result.CutoffEpoch != 1800000000 {
		t.Fatalf("unexpected cutoff epoch: %#v", result)
	}
	if result.Mode == "" {
		t.Fatalf("missing mode: %#v", result)
	}
}
