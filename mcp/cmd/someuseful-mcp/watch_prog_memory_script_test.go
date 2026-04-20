package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type watchProgramMemoryScriptTestProcess struct {
	PID        int     `json:"pid"`
	CPUPercent float64 `json:"cpuPercent"`
	RSSKb      int64   `json:"rssKb"`
	VSZKb      int64   `json:"vszKb"`
}

type watchProgramMemoryScriptTestResult struct {
	OK           bool                                  `json:"ok"`
	Timestamp    string                                `json:"timestamp"`
	ProcessName  string                                `json:"processName"`
	MatchedCount int                                   `json:"matchedCount"`
	Processes    []watchProgramMemoryScriptTestProcess `json:"processes"`
}

func runWatchProgramMemoryScript(t *testing.T, processName string, env []string) (watchProgramMemoryScriptTestResult, string, error) {
	t.Helper()

	scriptPath := filepath.Join(repoRootForScriptTests(t), "shell", "watch-prog-memory.sh")
	cmd := exec.Command("bash", scriptPath, "--json", "--process-name", processName)
	cmd.Env = append(os.Environ(), env...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return watchProgramMemoryScriptTestResult{}, string(output), err
	}

	var result watchProgramMemoryScriptTestResult
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("decode watch-prog-memory output: %v\n%s", err, string(output))
	}

	return result, string(output), nil
}

func TestWatchProgMemoryScriptSamplesMultipleProcesses(t *testing.T) {
	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}

	pgrepScript := "#!/usr/bin/env bash\nset -euo pipefail\nprintf '101\\n202\\n'\n"
	pidstatScript := "#!/usr/bin/env bash\nset -euo pipefail\nif [ \"$1\" = -u ]; then\ncat <<'OUT'\nLinux demo\nAverage: 1000 101 1.25 0.10 0.00 0.00 1.35 0 demo\nAverage: 1000 202 2.00 0.50 0.00 0.00 2.50 1 demo\nOUT\nexit 0\nfi\nif [ \"$1\" = -r ]; then\ncat <<'OUT'\nLinux demo\nAverage: 1000 101 0.00 0.00 4096 1024 0.10 demo\nAverage: 1000 202 0.00 0.00 8192 2048 0.20 demo\nOUT\nexit 0\nfi\nprintf 'unexpected args: %s\\n' \"$*\" >&2\nexit 1\n"

	if err := os.WriteFile(filepath.Join(binDir, "pgrep"), []byte(pgrepScript), 0o755); err != nil {
		t.Fatalf("write fake pgrep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "pidstat"), []byte(pidstatScript), 0o755); err != nil {
		t.Fatalf("write fake pidstat: %v", err)
	}

	result, _, err := runWatchProgramMemoryScript(t, "demo", []string{"PATH=" + binDir + ":" + os.Getenv("PATH")})
	if err != nil {
		t.Fatalf("watch-prog-memory script failed: %v", err)
	}

	if result.ProcessName != "demo" {
		t.Fatalf("unexpected process name: %s", result.ProcessName)
	}
	if result.MatchedCount != 2 {
		t.Fatalf("unexpected matched count: %d", result.MatchedCount)
	}
	if len(result.Processes) != 2 {
		t.Fatalf("unexpected processes: %#v", result.Processes)
	}
}

func TestWatchProgMemoryScriptFailsWhenNoProcessMatches(t *testing.T) {
	tempDir := t.TempDir()
	binDir := filepath.Join(tempDir, "bin")

	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin dir: %v", err)
	}

	pgrepScript := "#!/usr/bin/env bash\nset -euo pipefail\nexit 1\n"
	pidstatScript := "#!/usr/bin/env bash\nset -euo pipefail\nprintf 'pidstat should not run\\n' >&2\nexit 1\n"

	if err := os.WriteFile(filepath.Join(binDir, "pgrep"), []byte(pgrepScript), 0o755); err != nil {
		t.Fatalf("write fake pgrep: %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "pidstat"), []byte(pidstatScript), 0o755); err != nil {
		t.Fatalf("write fake pidstat: %v", err)
	}

	_, output, err := runWatchProgramMemoryScript(t, "demo", []string{"PATH=" + binDir + ":" + os.Getenv("PATH")})
	if err == nil {
		t.Fatalf("expected watch-prog-memory script to fail")
	}
	if !strings.Contains(output, "no running process matched: demo") {
		t.Fatalf("unexpected output: %s", output)
	}
}
