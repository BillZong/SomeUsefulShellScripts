package main

import "testing"

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

func TestNegotiateProtocolVersion(t *testing.T) {
	if got := negotiateProtocolVersion("2024-11-05"); got != "2024-11-05" {
		t.Fatalf("expected requested supported protocol version, got %s", got)
	}
	if got := negotiateProtocolVersion("2099-01-01"); got != latestProtocolVer {
		t.Fatalf("expected fallback to latest protocol version, got %s", got)
	}
}

func TestParseDockerShowImagesArchOptionsDefaults(t *testing.T) {
	options, err := parseDockerShowImagesArchOptions(nil)
	if err != nil {
		t.Fatalf("parseDockerShowImagesArchOptions returned error: %v", err)
	}
	if len(options.Images) != 0 {
		t.Fatalf("expected empty default images, got %#v", options.Images)
	}
}

func TestParseGitStatusSubdirOptionsDefaults(t *testing.T) {
	options, err := parseGitStatusSubdirOptions(nil)
	if err != nil {
		t.Fatalf("parseGitStatusSubdirOptions returned error: %v", err)
	}
	if options.Directory != "." {
		t.Fatalf("unexpected default directory: %s", options.Directory)
	}
	if options.Depth != 2 {
		t.Fatalf("unexpected default depth: %d", options.Depth)
	}
}

func TestParseGitStatusSubdirOptionsAliases(t *testing.T) {
	options, err := parseGitStatusSubdirOptions(map[string]interface{}{
		"directory":         "/tmp/workspace",
		"depth":             float64(3),
		"working_directory": "/tmp",
	})
	if err != nil {
		t.Fatalf("parseGitStatusSubdirOptions returned error: %v", err)
	}
	if options.Directory != "/tmp/workspace" {
		t.Fatalf("unexpected directory: %s", options.Directory)
	}
	if options.Depth != 3 {
		t.Fatalf("unexpected depth: %d", options.Depth)
	}
	if options.WorkingDirectory != "/tmp" {
		t.Fatalf("unexpected working directory: %s", options.WorkingDirectory)
	}
}

func TestParseWatchProgramMemoryOptionsRequired(t *testing.T) {
	if _, err := parseWatchProgramMemoryOptions(nil); err == nil {
		t.Fatalf("expected missing program to fail")
	}
}

func TestParseWatchProgramMemoryOptionsAliases(t *testing.T) {
	options, err := parseWatchProgramMemoryOptions(map[string]interface{}{
		"program":           "claude",
		"working_directory": "/tmp",
	})
	if err != nil {
		t.Fatalf("parseWatchProgramMemoryOptions returned error: %v", err)
	}
	if options.Program != "claude" {
		t.Fatalf("unexpected program: %s", options.Program)
	}
	if options.WorkingDirectory != "/tmp" {
		t.Fatalf("unexpected working directory: %s", options.WorkingDirectory)
	}
}
