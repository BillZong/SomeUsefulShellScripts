package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGoListDepOptionsDefaults(t *testing.T) {
	options, err := parseGoListDepOptions(nil)
	if err != nil {
		t.Fatalf("parseGoListDepOptions returned error: %v", err)
	}

	if len(options.Packages) != 1 || options.Packages[0] != "." {
		t.Fatalf("unexpected default packages: %#v", options.Packages)
	}
	if options.IncludeStdlib {
		t.Fatalf("expected include stdlib to default to false")
	}
	if options.TestImportDepth != 1 {
		t.Fatalf("unexpected default test import depth: %d", options.TestImportDepth)
	}
}

func TestParseGoListDepOptionsAliases(t *testing.T) {
	options, err := parseGoListDepOptions(map[string]interface{}{
		"packages":          []interface{}{"fmt", "net/http"},
		"include_stdlib":    true,
		"test_import_depth": float64(0),
		"working_directory": "/tmp/demo",
	})
	if err != nil {
		t.Fatalf("parseGoListDepOptions returned error: %v", err)
	}

	if len(options.Packages) != 2 || options.Packages[0] != "fmt" || options.Packages[1] != "net/http" {
		t.Fatalf("unexpected packages: %#v", options.Packages)
	}
	if !options.IncludeStdlib {
		t.Fatalf("expected include stdlib to be true")
	}
	if options.TestImportDepth != 0 {
		t.Fatalf("unexpected test import depth: %d", options.TestImportDepth)
	}
	if options.WorkingDirectory != "/tmp/demo" {
		t.Fatalf("unexpected working directory: %s", options.WorkingDirectory)
	}
}

func TestParseGitCountLineOptionsDefaults(t *testing.T) {
	options, err := parseGitCountLineOptions(map[string]interface{}{
		"beginDate": "2024-01-01",
		"endDate":   "2024-12-31",
	})
	if err != nil {
		t.Fatalf("parseGitCountLineOptions returned error: %v", err)
	}

	if options.BeginDate != "2024-01-01" || options.EndDate != "2024-12-31" {
		t.Fatalf("unexpected date range: %#v", options)
	}
	if options.Directory != "." {
		t.Fatalf("unexpected default directory: %s", options.Directory)
	}
	if options.AuthorName != "" {
		t.Fatalf("expected empty default author name, got: %s", options.AuthorName)
	}
}

func TestParseGitCountLineOptionsAliases(t *testing.T) {
	options, err := parseGitCountLineOptions(map[string]interface{}{
		"begin_date":        "2024-01-01",
		"end_date":          "2024-12-31",
		"directory":         "/tmp/repo",
		"author_name":       "BillZong",
		"working_directory": "/tmp",
	})
	if err != nil {
		t.Fatalf("parseGitCountLineOptions returned error: %v", err)
	}

	if options.BeginDate != "2024-01-01" || options.EndDate != "2024-12-31" {
		t.Fatalf("unexpected date range: %#v", options)
	}
	if options.Directory != "/tmp/repo" {
		t.Fatalf("unexpected directory: %s", options.Directory)
	}
	if options.AuthorName != "BillZong" {
		t.Fatalf("unexpected author name: %s", options.AuthorName)
	}
	if options.WorkingDirectory != "/tmp" {
		t.Fatalf("unexpected working directory: %s", options.WorkingDirectory)
	}
}

func TestParseGitFindLargeFilesOptionsDefaults(t *testing.T) {
	options, err := parseGitFindLargeFilesOptions(nil)
	if err != nil {
		t.Fatalf("parseGitFindLargeFilesOptions returned error: %v", err)
	}

	if options.Directory != "." {
		t.Fatalf("unexpected default directory: %s", options.Directory)
	}
	if options.Limit != 0 {
		t.Fatalf("unexpected default limit: %d", options.Limit)
	}
	if options.WorkingDirectory != "" {
		t.Fatalf("expected empty default working directory, got: %s", options.WorkingDirectory)
	}
}

func TestParseGitFindLargeFilesOptionsAliases(t *testing.T) {
	options, err := parseGitFindLargeFilesOptions(map[string]interface{}{
		"directory":         "/tmp/repo",
		"limit":             float64(25),
		"working_directory": "/tmp",
	})
	if err != nil {
		t.Fatalf("parseGitFindLargeFilesOptions returned error: %v", err)
	}

	if options.Directory != "/tmp/repo" {
		t.Fatalf("unexpected directory: %s", options.Directory)
	}
	if options.Limit != 25 {
		t.Fatalf("unexpected limit: %d", options.Limit)
	}
	if options.WorkingDirectory != "/tmp" {
		t.Fatalf("unexpected working directory: %s", options.WorkingDirectory)
	}
}

func TestParseGitStatusSubdirsOptionsDefaults(t *testing.T) {
	options, err := parseGitStatusSubdirsOptions(nil)
	if err != nil {
		t.Fatalf("parseGitStatusSubdirsOptions returned error: %v", err)
	}

	if options.Directory != "." {
		t.Fatalf("unexpected default directory: %s", options.Directory)
	}
	if options.Depth != 2 {
		t.Fatalf("unexpected default depth: %d", options.Depth)
	}
	if options.WorkingDirectory != "" {
		t.Fatalf("expected empty default working directory, got: %s", options.WorkingDirectory)
	}
}

func TestParseGitStatusSubdirsOptionsAliases(t *testing.T) {
	options, err := parseGitStatusSubdirsOptions(map[string]interface{}{
		"directory":         "/tmp/workspace",
		"depth":             float64(4),
		"working_directory": "/tmp",
	})
	if err != nil {
		t.Fatalf("parseGitStatusSubdirsOptions returned error: %v", err)
	}

	if options.Directory != "/tmp/workspace" {
		t.Fatalf("unexpected directory: %s", options.Directory)
	}
	if options.Depth != 4 {
		t.Fatalf("unexpected depth: %d", options.Depth)
	}
	if options.WorkingDirectory != "/tmp" {
		t.Fatalf("unexpected working directory: %s", options.WorkingDirectory)
	}
}

func TestHandleToolsListIncludesGitStatusSubdirs(t *testing.T) {
	srv := &server{}

	response, ok := srv.handleToolsList(requestEnvelope{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
	})
	if !ok {
		t.Fatalf("expected handleToolsList to return a response")
	}

	var decoded struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &decoded); err != nil {
		t.Fatalf("failed to decode tools/list response: %v", err)
	}

	for _, tool := range decoded.Result.Tools {
		if tool.Name == "git_status_subdirs" {
			return
		}
	}

	t.Fatalf("git_status_subdirs not found in tools/list response")
}

func TestParseDockerShowImagesArchOptionsDefaults(t *testing.T) {
	options, err := parseDockerShowImagesArchOptions(nil)
	if err != nil {
		t.Fatalf("parseDockerShowImagesArchOptions returned error: %v", err)
	}

	if options.WorkingDirectory != "" {
		t.Fatalf("expected empty default working directory, got: %s", options.WorkingDirectory)
	}
}

func TestParseDockerShowImagesArchOptionsAliases(t *testing.T) {
	options, err := parseDockerShowImagesArchOptions(map[string]interface{}{
		"working_directory": "/tmp/docker",
	})
	if err != nil {
		t.Fatalf("parseDockerShowImagesArchOptions returned error: %v", err)
	}

	if options.WorkingDirectory != "/tmp/docker" {
		t.Fatalf("unexpected working directory: %s", options.WorkingDirectory)
	}
}

func TestHandleToolsListIncludesDockerShowImagesArch(t *testing.T) {
	srv := &server{}

	response, ok := srv.handleToolsList(requestEnvelope{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
	})
	if !ok {
		t.Fatalf("expected handleToolsList to return a response")
	}

	var decoded struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &decoded); err != nil {
		t.Fatalf("failed to decode tools/list response: %v", err)
	}

	for _, tool := range decoded.Result.Tools {
		if tool.Name == "docker_show_images_arch" {
			return
		}
	}

	t.Fatalf("docker_show_images_arch not found in tools/list response")
}

func TestParseWatchProgramMemoryOptionsDefaults(t *testing.T) {
	options, err := parseWatchProgramMemoryOptions(map[string]interface{}{
		"processName": "postgres",
	})
	if err != nil {
		t.Fatalf("parseWatchProgramMemoryOptions returned error: %v", err)
	}

	if options.ProcessName != "postgres" {
		t.Fatalf("unexpected process name: %s", options.ProcessName)
	}
	if options.WorkingDirectory != "" {
		t.Fatalf("expected empty default working directory, got: %s", options.WorkingDirectory)
	}
}

func TestParseWatchProgramMemoryOptionsAliases(t *testing.T) {
	options, err := parseWatchProgramMemoryOptions(map[string]interface{}{
		"process_name":      "node",
		"working_directory": "/tmp/runtime",
	})
	if err != nil {
		t.Fatalf("parseWatchProgramMemoryOptions returned error: %v", err)
	}

	if options.ProcessName != "node" {
		t.Fatalf("unexpected process name: %s", options.ProcessName)
	}
	if options.WorkingDirectory != "/tmp/runtime" {
		t.Fatalf("unexpected working directory: %s", options.WorkingDirectory)
	}
}

func TestHandleToolsListIncludesWatchProgramMemory(t *testing.T) {
	srv := &server{}

	response, ok := srv.handleToolsList(requestEnvelope{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
	})
	if !ok {
		t.Fatalf("expected handleToolsList to return a response")
	}

	var decoded struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &decoded); err != nil {
		t.Fatalf("failed to decode tools/list response: %v", err)
	}

	for _, tool := range decoded.Result.Tools {
		if tool.Name == "watch_program_memory" {
			return
		}
	}

	t.Fatalf("watch_program_memory not found in tools/list response")
}

func TestRunWatchProgramMemoryReturnsScriptFailure(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "watch-prog-memory.sh")

	if err := os.WriteFile(scriptPath, []byte("#!/usr/bin/env bash\nprintf 'boom\\n' >&2\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("write fake script: %v", err)
	}

	t.Setenv("SUSS_WATCH_PROG_MEMORY_SCRIPT", scriptPath)

	_, err := runWatchProgramMemory(map[string]interface{}{
		"processName": "demo",
	})
	if err == nil {
		t.Fatalf("expected runWatchProgramMemory to fail")
	}
	if !strings.Contains(err.Error(), "watch_program_memory failed: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleToolsCallWatchProgramMemorySuccess(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "watch-prog-memory.sh")
	script := "#!/usr/bin/env bash\ncat <<'EOF'\n{\"ok\":true,\"timestamp\":\"2026-04-20T12:00:00+0800\",\"processName\":\"demo\",\"matchedCount\":1,\"processes\":[{\"pid\":123,\"cpuPercent\":1.5,\"rssKb\":2048,\"vszKb\":4096}]}\nEOF\n"

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake script: %v", err)
	}

	t.Setenv("SUSS_WATCH_PROG_MEMORY_SCRIPT", scriptPath)

	requestParams, err := json.Marshal(toolCallParams{
		Name: "watch_program_memory",
		Arguments: map[string]interface{}{
			"processName": "demo",
		},
	})
	if err != nil {
		t.Fatalf("marshal tool call params: %v", err)
	}

	srv := &server{}
	response, ok := srv.handleToolsCall(requestEnvelope{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Params:  requestParams,
	})
	if !ok {
		t.Fatalf("expected handleToolsCall to return a response")
	}

	var decoded struct {
		Result struct {
			IsError           bool `json:"isError"`
			StructuredContent struct {
				ProcessName  string `json:"processName"`
				MatchedCount int    `json:"matchedCount"`
				Processes    []struct {
					PID        int     `json:"pid"`
					CPUPercent float64 `json:"cpuPercent"`
					RSSKb      int64   `json:"rssKb"`
					VSZKb      int64   `json:"vszKb"`
				} `json:"processes"`
			} `json:"structuredContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &decoded); err != nil {
		t.Fatalf("decode tool call response: %v", err)
	}

	if decoded.Result.IsError {
		t.Fatalf("expected successful tool response")
	}
	if decoded.Result.StructuredContent.ProcessName != "demo" {
		t.Fatalf("unexpected process name: %s", decoded.Result.StructuredContent.ProcessName)
	}
	if decoded.Result.StructuredContent.MatchedCount != 1 {
		t.Fatalf("unexpected matched count: %d", decoded.Result.StructuredContent.MatchedCount)
	}
	if len(decoded.Result.StructuredContent.Processes) != 1 {
		t.Fatalf("unexpected processes: %#v", decoded.Result.StructuredContent.Processes)
	}
	process := decoded.Result.StructuredContent.Processes[0]
	if process.PID != 123 || process.CPUPercent != 1.5 || process.RSSKb != 2048 || process.VSZKb != 4096 {
		t.Fatalf("unexpected process mapping: %#v", process)
	}
}

func TestParseDuDirectoryOptionsDefaults(t *testing.T) {
	options, err := parseDuDirectoryOptions(nil)
	if err != nil {
		t.Fatalf("parseDuDirectoryOptions returned error: %v", err)
	}

	if options.Directory != "." {
		t.Fatalf("unexpected directory: %s", options.Directory)
	}
	if options.WorkingDirectory != "" {
		t.Fatalf("expected empty working directory, got: %s", options.WorkingDirectory)
	}
}

func TestParseDuDirectoryOptionsAliases(t *testing.T) {
	options, err := parseDuDirectoryOptions(map[string]interface{}{
		"directory":         "/tmp/workspace",
		"working_directory": "/tmp",
	})
	if err != nil {
		t.Fatalf("parseDuDirectoryOptions returned error: %v", err)
	}

	if options.Directory != "/tmp/workspace" {
		t.Fatalf("unexpected directory: %s", options.Directory)
	}
	if options.WorkingDirectory != "/tmp" {
		t.Fatalf("unexpected working directory: %s", options.WorkingDirectory)
	}
}

func TestHandleToolsListIncludesDuDirectory(t *testing.T) {
	srv := &server{}

	response, ok := srv.handleToolsList(requestEnvelope{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
	})
	if !ok {
		t.Fatalf("expected handleToolsList to return a response")
	}

	var decoded struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(response, &decoded); err != nil {
		t.Fatalf("failed to decode tools/list response: %v", err)
	}

	for _, tool := range decoded.Result.Tools {
		if tool.Name == "du_directory" {
			return
		}
	}

	t.Fatalf("du_directory not found in tools/list response")
}
func TestNegotiateProtocolVersion(t *testing.T) {
	if got := negotiateProtocolVersion("2024-11-05"); got != "2024-11-05" {
		t.Fatalf("expected requested supported protocol version, got %s", got)
	}
	if got := negotiateProtocolVersion("2099-01-01"); got != latestProtocolVer {
		t.Fatalf("expected fallback to latest protocol version, got %s", got)
	}
}
