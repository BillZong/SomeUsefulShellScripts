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

func TestNegotiateProtocolVersion(t *testing.T) {
	if got := negotiateProtocolVersion("2024-11-05"); got != "2024-11-05" {
		t.Fatalf("expected requested supported protocol version, got %s", got)
	}
	if got := negotiateProtocolVersion("2099-01-01"); got != latestProtocolVer {
		t.Fatalf("expected fallback to latest protocol version, got %s", got)
	}
}
