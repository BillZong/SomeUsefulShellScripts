package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	serverName        = "someuseful-shell-scripts"
	serverTitle       = "SomeUsefulShellScripts MCP"
	serverVersion     = "0.1.0"
	latestProtocolVer = "2025-11-25"
)

var supportedProtocolVersions = []string{
	latestProtocolVer,
	"2025-06-18",
	"2025-03-26",
	"2024-11-05",
}

type requestEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type responseEnvelope struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type toolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type goListDepOptions struct {
	Packages         []string
	IncludeStdlib    bool
	TestImportDepth  int
	WorkingDirectory string
}

type gitCountLineOptions struct {
	BeginDate        string
	EndDate          string
	Directory        string
	AuthorName       string
	WorkingDirectory string
}

type gitFindLargeFilesOptions struct {
	Directory        string
	Limit            int
	WorkingDirectory string
}

type gitStatusSubdirsOptions struct {
	Directory        string
	Depth            int
	WorkingDirectory string
}

type goListDepScriptResult struct {
	OK              bool     `json:"ok"`
	Packages        []string `json:"packages"`
	IncludeStdlib   bool     `json:"include_stdlib"`
	TestImportDepth int      `json:"test_import_depth"`
	Dependencies    []string `json:"dependencies"`
}

type goListDepStructuredResult struct {
	OK              bool     `json:"ok"`
	Packages        []string `json:"packages"`
	IncludeStdlib   bool     `json:"includeStdlib"`
	TestImportDepth int      `json:"testImportDepth"`
	Dependencies    []string `json:"dependencies"`
}

type gitCountLineScriptResult struct {
	OK           bool   `json:"ok"`
	BeginDate    string `json:"begin_date"`
	EndDate      string `json:"end_date"`
	Directory    string `json:"directory"`
	AuthorName   string `json:"author_name"`
	AddedLines   int    `json:"added_lines"`
	RemovedLines int    `json:"removed_lines"`
	TotalLines   int    `json:"total_lines"`
}

type gitCountLineStructuredResult struct {
	OK           bool   `json:"ok"`
	BeginDate    string `json:"beginDate"`
	EndDate      string `json:"endDate"`
	Directory    string `json:"directory"`
	AuthorName   string `json:"authorName"`
	AddedLines   int    `json:"addedLines"`
	RemovedLines int    `json:"removedLines"`
	TotalLines   int    `json:"totalLines"`
}

type gitFindLargeFilesScriptFile struct {
	ObjectID  string `json:"object_id"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SizeHuman string `json:"size_human"`
}

type gitFindLargeFilesScriptResult struct {
	OK            bool                          `json:"ok"`
	Directory     string                        `json:"directory"`
	Limit         int                           `json:"limit"`
	TotalCount    int                           `json:"total_count"`
	ReturnedCount int                           `json:"returned_count"`
	Truncated     bool                          `json:"truncated"`
	Files         []gitFindLargeFilesScriptFile `json:"files"`
}

type gitFindLargeFilesStructuredFile struct {
	ObjectID  string `json:"objectId"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	SizeHuman string `json:"sizeHuman"`
}

type gitFindLargeFilesStructuredResult struct {
	OK            bool                              `json:"ok"`
	Directory     string                            `json:"directory"`
	Limit         int                               `json:"limit"`
	TotalCount    int                               `json:"totalCount"`
	ReturnedCount int                               `json:"returnedCount"`
	Truncated     bool                              `json:"truncated"`
	Files         []gitFindLargeFilesStructuredFile `json:"files"`
}

type gitStatusSubdirsScriptRepository struct {
	Path      string   `json:"path"`
	Branch    string   `json:"branch"`
	IsClean   bool     `json:"isClean"`
	Porcelain []string `json:"porcelain"`
}

type gitStatusSubdirsScriptResult struct {
	OK           bool                               `json:"ok"`
	Directory    string                             `json:"directory"`
	Depth        int                                `json:"depth"`
	Repositories []gitStatusSubdirsScriptRepository `json:"repositories"`
}

type gitStatusSubdirsStructuredRepository struct {
	Path      string   `json:"path"`
	Branch    string   `json:"branch"`
	IsClean   bool     `json:"isClean"`
	Porcelain []string `json:"porcelain"`
}

type gitStatusSubdirsStructuredResult struct {
	OK           bool                                   `json:"ok"`
	Directory    string                                 `json:"directory"`
	Depth        int                                    `json:"depth"`
	Repositories []gitStatusSubdirsStructuredRepository `json:"repositories"`
}

type server struct {
	out         *bufio.Writer
	errOut      io.Writer
	initialized bool
}

func main() {
	srv := &server{
		out:    bufio.NewWriter(os.Stdout),
		errOut: os.Stderr,
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		response, ok := srv.handleMessage(line)
		if !ok {
			continue
		}

		if _, err := srv.out.Write(response); err != nil {
			fmt.Fprintf(srv.errOut, "write response: %v\n", err)
			os.Exit(1)
		}
		if err := srv.out.WriteByte('\n'); err != nil {
			fmt.Fprintf(srv.errOut, "write newline: %v\n", err)
			os.Exit(1)
		}
		if err := srv.out.Flush(); err != nil {
			fmt.Fprintf(srv.errOut, "flush stdout: %v\n", err)
			os.Exit(1)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(srv.errOut, "read stdin: %v\n", err)
		os.Exit(1)
	}
}

func (s *server) handleMessage(line []byte) ([]byte, bool) {
	if bytes.HasPrefix(bytes.TrimSpace(line), []byte("[")) {
		return s.handleBatch(line)
	}
	return s.handleSingle(line)
}

func (s *server) handleBatch(line []byte) ([]byte, bool) {
	var rawBatch []json.RawMessage
	if err := json.Unmarshal(line, &rawBatch); err != nil {
		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      nil,
			Error:   &rpcError{Code: -32700, Message: "parse error", Data: err.Error()},
		})
	}

	responses := make([]json.RawMessage, 0, len(rawBatch))
	for _, raw := range rawBatch {
		response, ok := s.handleSingle(raw)
		if ok {
			responses = append(responses, response)
		}
	}

	if len(responses) == 0 {
		return nil, false
	}

	encoded, err := json.Marshal(responses)
	if err != nil {
		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      nil,
			Error:   &rpcError{Code: -32603, Message: "internal error", Data: err.Error()},
		})
	}
	return encoded, true
}

func (s *server) handleSingle(line []byte) ([]byte, bool) {
	var req requestEnvelope
	if err := json.Unmarshal(line, &req); err != nil {
		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      nil,
			Error:   &rpcError{Code: -32700, Message: "parse error", Data: err.Error()},
		})
	}

	if req.JSONRPC != "2.0" {
		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      rawIDToValue(req.ID),
			Error:   &rpcError{Code: -32600, Message: "invalid request", Data: "jsonrpc must be 2.0"},
		})
	}

	if req.Method == "" {
		return nil, false
	}

	if !s.initialized && req.Method != "initialize" && req.Method != "notifications/initialized" {
		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      rawIDToValue(req.ID),
			Error:   &rpcError{Code: -32002, Message: "server not initialized"},
		})
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		s.initialized = true
		return nil, false
	case "ping":
		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      rawIDToValue(req.ID),
			Result:  map[string]interface{}{},
		})
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	case "shutdown":
		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      rawIDToValue(req.ID),
			Result:  map[string]interface{}{},
		})
	default:
		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      rawIDToValue(req.ID),
			Error:   &rpcError{Code: -32601, Message: "method not found", Data: req.Method},
		})
	}
}

func (s *server) handleInitialize(req requestEnvelope) ([]byte, bool) {
	var params initializeParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return marshalResponse(responseEnvelope{
				JSONRPC: "2.0",
				ID:      rawIDToValue(req.ID),
				Error:   &rpcError{Code: -32602, Message: "invalid initialize params", Data: err.Error()},
			})
		}
	}

	result := map[string]interface{}{
		"protocolVersion": negotiateProtocolVersion(params.ProtocolVersion),
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":        serverName,
			"title":       serverTitle,
			"version":     serverVersion,
			"description": "Minimal stdio MCP server for curated local automation tools.",
		},
		"instructions": "Prefer the read-only tools go_list_dep, git_count_line, git_find_large_files, and git_status_subdirs for structured repository inspection. High-risk shell utilities are intentionally not exposed yet.",
	}

	return marshalResponse(responseEnvelope{
		JSONRPC: "2.0",
		ID:      rawIDToValue(req.ID),
		Result:  result,
	})
}

func (s *server) handleToolsList(req requestEnvelope) ([]byte, bool) {
	tools := []interface{}{
		map[string]interface{}{
			"name":        "go_list_dep",
			"title":       "List Go Package Dependencies",
			"description": "Run the repository's go-list-dep CLI and return Go package dependencies as structured data.",
			"inputSchema": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"packages": map[string]interface{}{
						"type":        "array",
						"description": "Go packages or import paths to inspect. Defaults to ['.'].",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"includeStdlib": map[string]interface{}{
						"type":        "boolean",
						"description": "When true, keep standard library packages in the result.",
					},
					"testImportDepth": map[string]interface{}{
						"type":        "integer",
						"description": "How many recursive TestImports levels to follow. Defaults to 1.",
					},
					"workingDirectory": map[string]interface{}{
						"type":        "string",
						"description": "Optional working directory for the underlying go list command.",
					},
				},
			},
			"outputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ok": map[string]interface{}{"type": "boolean"},
					"packages": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
					"includeStdlib": map[string]interface{}{"type": "boolean"},
					"testImportDepth": map[string]interface{}{
						"type": "integer",
					},
					"dependencies": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"ok", "packages", "includeStdlib", "testImportDepth", "dependencies"},
			},
			"annotations": map[string]interface{}{
				"title":           "Go dependency list",
				"readOnlyHint":    true,
				"destructiveHint": false,
				"idempotentHint":  true,
				"openWorldHint":   false,
			},
		},
		map[string]interface{}{
			"name":        "git_count_line",
			"title":       "Count Git Lines by Author",
			"description": "Run the repository's git-count-line CLI and return added and removed line totals for an author within a date range.",
			"inputSchema": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"beginDate": map[string]interface{}{
						"type":        "string",
						"description": "Inclusive begin date, for example 2024-01-01.",
					},
					"endDate": map[string]interface{}{
						"type":        "string",
						"description": "Inclusive end date, for example 2026-01-01.",
					},
					"directory": map[string]interface{}{
						"type":        "string",
						"description": "Repository directory to inspect. Defaults to the current directory.",
					},
					"authorName": map[string]interface{}{
						"type":        "string",
						"description": "Author name to match. Defaults to git config user.name.",
					},
					"workingDirectory": map[string]interface{}{
						"type":        "string",
						"description": "Optional working directory for launching the underlying script.",
					},
				},
				"required": []string{"beginDate", "endDate"},
			},
			"outputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ok":           map[string]interface{}{"type": "boolean"},
					"beginDate":    map[string]interface{}{"type": "string"},
					"endDate":      map[string]interface{}{"type": "string"},
					"directory":    map[string]interface{}{"type": "string"},
					"authorName":   map[string]interface{}{"type": "string"},
					"addedLines":   map[string]interface{}{"type": "integer"},
					"removedLines": map[string]interface{}{"type": "integer"},
					"totalLines":   map[string]interface{}{"type": "integer"},
				},
				"required": []string{"ok", "beginDate", "endDate", "directory", "authorName", "addedLines", "removedLines", "totalLines"},
			},
			"annotations": map[string]interface{}{
				"title":           "Git line count",
				"readOnlyHint":    true,
				"destructiveHint": false,
				"idempotentHint":  true,
				"openWorldHint":   false,
			},
		},
		map[string]interface{}{
			"name":        "git_find_large_files",
			"title":       "Find Large Git Blob Objects",
			"description": "Run the repository's git-find-large-files CLI and return tracked blob objects ordered by size.",
			"inputSchema": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"directory": map[string]interface{}{
						"type":        "string",
						"description": "Repository directory to inspect. Defaults to the current directory.",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of results to return. Defaults to 0 for no limit.",
					},
					"workingDirectory": map[string]interface{}{
						"type":        "string",
						"description": "Optional working directory for launching the underlying script.",
					},
				},
			},
			"outputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ok":            map[string]interface{}{"type": "boolean"},
					"directory":     map[string]interface{}{"type": "string"},
					"limit":         map[string]interface{}{"type": "integer"},
					"totalCount":    map[string]interface{}{"type": "integer"},
					"returnedCount": map[string]interface{}{"type": "integer"},
					"truncated":     map[string]interface{}{"type": "boolean"},
					"files": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"objectId":  map[string]interface{}{"type": "string"},
								"path":      map[string]interface{}{"type": "string"},
								"sizeBytes": map[string]interface{}{"type": "integer"},
								"sizeHuman": map[string]interface{}{"type": "string"},
							},
							"required": []string{"objectId", "path", "sizeBytes", "sizeHuman"},
						},
					},
				},
				"required": []string{"ok", "directory", "limit", "totalCount", "returnedCount", "truncated", "files"},
			},
			"annotations": map[string]interface{}{
				"title":           "Git large files",
				"readOnlyHint":    true,
				"destructiveHint": false,
				"idempotentHint":  true,
				"openWorldHint":   false,
			},
		},
		map[string]interface{}{
			"name":        "git_status_subdirs",
			"title":       "Inspect Git Repositories Under a Directory",
			"description": "Run the repository's git-status-subdir CLI and return child repository branches and porcelain status.",
			"inputSchema": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": false,
				"properties": map[string]interface{}{
					"directory": map[string]interface{}{
						"type":        "string",
						"description": "Root directory to scan. Defaults to the current directory.",
					},
					"depth": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum directory depth to scan for child repositories. Defaults to 2.",
					},
					"workingDirectory": map[string]interface{}{
						"type":        "string",
						"description": "Optional working directory for launching the underlying script.",
					},
				},
			},
			"outputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ok":        map[string]interface{}{"type": "boolean"},
					"directory": map[string]interface{}{"type": "string"},
					"depth":     map[string]interface{}{"type": "integer"},
					"repositories": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"path":      map[string]interface{}{"type": "string"},
								"branch":    map[string]interface{}{"type": "string"},
								"isClean":   map[string]interface{}{"type": "boolean"},
								"porcelain": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
							},
							"required": []string{"path", "branch", "isClean", "porcelain"},
						},
					},
				},
				"required": []string{"ok", "directory", "depth", "repositories"},
			},
			"annotations": map[string]interface{}{
				"title":           "Git subdir status",
				"readOnlyHint":    true,
				"destructiveHint": false,
				"idempotentHint":  true,
				"openWorldHint":   false,
			},
		},
	}

	return marshalResponse(responseEnvelope{
		JSONRPC: "2.0",
		ID:      rawIDToValue(req.ID),
		Result: map[string]interface{}{
			"tools": tools,
		},
	})
}

func (s *server) handleToolsCall(req requestEnvelope) ([]byte, bool) {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      rawIDToValue(req.ID),
			Error:   &rpcError{Code: -32602, Message: "invalid tool call params", Data: err.Error()},
		})
	}

	switch params.Name {
	case "go_list_dep":
		result, err := runGoListDep(params.Arguments)
		if err != nil {
			return marshalResponse(responseEnvelope{
				JSONRPC: "2.0",
				ID:      rawIDToValue(req.ID),
				Result: map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": err.Error(),
						},
					},
					"isError": true,
				},
			})
		}

		pretty, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return marshalResponse(responseEnvelope{
				JSONRPC: "2.0",
				ID:      rawIDToValue(req.ID),
				Error:   &rpcError{Code: -32603, Message: "failed to encode tool result", Data: err.Error()},
			})
		}

		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      rawIDToValue(req.ID),
			Result: map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": string(pretty),
					},
				},
				"structuredContent": result,
				"isError":           false,
			},
		})
	case "git_count_line":
		result, err := runGitCountLine(params.Arguments)
		if err != nil {
			return marshalResponse(responseEnvelope{
				JSONRPC: "2.0",
				ID:      rawIDToValue(req.ID),
				Result: map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": err.Error(),
						},
					},
					"isError": true,
				},
			})
		}

		pretty, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return marshalResponse(responseEnvelope{
				JSONRPC: "2.0",
				ID:      rawIDToValue(req.ID),
				Error:   &rpcError{Code: -32603, Message: "failed to encode tool result", Data: err.Error()},
			})
		}

		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      rawIDToValue(req.ID),
			Result: map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": string(pretty),
					},
				},
				"structuredContent": result,
				"isError":           false,
			},
		})
	case "git_find_large_files":
		result, err := runGitFindLargeFiles(params.Arguments)
		if err != nil {
			return marshalResponse(responseEnvelope{
				JSONRPC: "2.0",
				ID:      rawIDToValue(req.ID),
				Result: map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": err.Error(),
						},
					},
					"isError": true,
				},
			})
		}

		pretty, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return marshalResponse(responseEnvelope{
				JSONRPC: "2.0",
				ID:      rawIDToValue(req.ID),
				Error:   &rpcError{Code: -32603, Message: "failed to encode tool result", Data: err.Error()},
			})
		}

		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      rawIDToValue(req.ID),
			Result: map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": string(pretty),
					},
				},
				"structuredContent": result,
				"isError":           false,
			},
		})
	case "git_status_subdirs":
		result, err := runGitStatusSubdirs(params.Arguments)
		if err != nil {
			return marshalResponse(responseEnvelope{
				JSONRPC: "2.0",
				ID:      rawIDToValue(req.ID),
				Result: map[string]interface{}{
					"content": []interface{}{
						map[string]interface{}{
							"type": "text",
							"text": err.Error(),
						},
					},
					"isError": true,
				},
			})
		}

		pretty, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return marshalResponse(responseEnvelope{
				JSONRPC: "2.0",
				ID:      rawIDToValue(req.ID),
				Error:   &rpcError{Code: -32603, Message: "failed to encode tool result", Data: err.Error()},
			})
		}

		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      rawIDToValue(req.ID),
			Result: map[string]interface{}{
				"content": []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": string(pretty),
					},
				},
				"structuredContent": result,
				"isError":           false,
			},
		})
	default:
		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      rawIDToValue(req.ID),
			Error:   &rpcError{Code: -32601, Message: "tool not found", Data: params.Name},
		})
	}
}

func runGoListDep(arguments map[string]interface{}) (goListDepStructuredResult, error) {
	options, err := parseGoListDepOptions(arguments)
	if err != nil {
		return goListDepStructuredResult{}, err
	}

	scriptPath, err := resolveShellScript("SUSS_GO_LIST_DEP_SCRIPT", "go-list-dep.sh")
	if err != nil {
		return goListDepStructuredResult{}, err
	}

	args := []string{scriptPath, "--json", "--test-import-depth", strconv.Itoa(options.TestImportDepth)}
	if options.IncludeStdlib {
		args = append(args, "--include-stdlib")
	}
	args = append(args, options.Packages...)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", args...)
	if options.WorkingDirectory != "" {
		cmd.Dir = options.WorkingDirectory
	}

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderrText := strings.TrimSpace(string(exitErr.Stderr))
			if stderrText == "" {
				stderrText = exitErr.Error()
			}
			return goListDepStructuredResult{}, fmt.Errorf("go_list_dep failed: %s", stderrText)
		}
		return goListDepStructuredResult{}, fmt.Errorf("go_list_dep execution failed: %w", err)
	}

	var parsed goListDepScriptResult
	if err := json.Unmarshal(output, &parsed); err != nil {
		return goListDepStructuredResult{}, fmt.Errorf("invalid JSON from go-list-dep script: %w", err)
	}

	return goListDepStructuredResult{
		OK:              parsed.OK,
		Packages:        parsed.Packages,
		IncludeStdlib:   parsed.IncludeStdlib,
		TestImportDepth: parsed.TestImportDepth,
		Dependencies:    parsed.Dependencies,
	}, nil
}

func parseGoListDepOptions(arguments map[string]interface{}) (goListDepOptions, error) {
	options := goListDepOptions{
		Packages:        []string{"."},
		IncludeStdlib:   false,
		TestImportDepth: 1,
	}

	if len(arguments) == 0 {
		return options, nil
	}

	if value, ok := arguments["packages"]; ok {
		packages, err := toStringSlice(value)
		if err != nil {
			return options, fmt.Errorf("packages must be a string or array of strings")
		}
		if len(packages) > 0 {
			options.Packages = packages
		}
	}

	if value, ok := firstValue(arguments, "includeStdlib", "include_stdlib"); ok {
		booleanValue, ok := value.(bool)
		if !ok {
			return options, fmt.Errorf("includeStdlib must be a boolean")
		}
		options.IncludeStdlib = booleanValue
	}

	if value, ok := firstValue(arguments, "testImportDepth", "test_import_depth"); ok {
		intValue, err := toInt(value)
		if err != nil {
			return options, fmt.Errorf("testImportDepth must be an integer")
		}
		options.TestImportDepth = intValue
	}

	if value, ok := firstValue(arguments, "workingDirectory", "working_directory"); ok {
		stringValue, ok := value.(string)
		if !ok {
			return options, fmt.Errorf("workingDirectory must be a string")
		}
		options.WorkingDirectory = stringValue
	}

	return options, nil
}

func runGitCountLine(arguments map[string]interface{}) (gitCountLineStructuredResult, error) {
	options, err := parseGitCountLineOptions(arguments)
	if err != nil {
		return gitCountLineStructuredResult{}, err
	}

	scriptPath, err := resolveShellScript("SUSS_GIT_COUNT_LINE_SCRIPT", "git-count-line.sh")
	if err != nil {
		return gitCountLineStructuredResult{}, err
	}

	args := []string{
		scriptPath,
		"--json",
		"--begin-date", options.BeginDate,
		"--end-date", options.EndDate,
		"--directory", options.Directory,
	}
	if options.AuthorName != "" {
		args = append(args, "--author", options.AuthorName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", args...)
	if options.WorkingDirectory != "" {
		cmd.Dir = options.WorkingDirectory
	}

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderrText := strings.TrimSpace(string(exitErr.Stderr))
			if stderrText == "" {
				stderrText = exitErr.Error()
			}
			return gitCountLineStructuredResult{}, fmt.Errorf("git_count_line failed: %s", stderrText)
		}
		return gitCountLineStructuredResult{}, fmt.Errorf("git_count_line execution failed: %w", err)
	}

	var parsed gitCountLineScriptResult
	if err := json.Unmarshal(output, &parsed); err != nil {
		return gitCountLineStructuredResult{}, fmt.Errorf("invalid JSON from git-count-line script: %w", err)
	}

	return gitCountLineStructuredResult{
		OK:           parsed.OK,
		BeginDate:    parsed.BeginDate,
		EndDate:      parsed.EndDate,
		Directory:    parsed.Directory,
		AuthorName:   parsed.AuthorName,
		AddedLines:   parsed.AddedLines,
		RemovedLines: parsed.RemovedLines,
		TotalLines:   parsed.TotalLines,
	}, nil
}

func parseGitCountLineOptions(arguments map[string]interface{}) (gitCountLineOptions, error) {
	options := gitCountLineOptions{
		Directory: ".",
	}

	if len(arguments) == 0 {
		return options, fmt.Errorf("beginDate is required")
	}

	if value, ok := firstValue(arguments, "beginDate", "begin_date"); ok {
		stringValue, ok := value.(string)
		if !ok || stringValue == "" {
			return options, fmt.Errorf("beginDate must be a non-empty string")
		}
		options.BeginDate = stringValue
	}
	if value, ok := firstValue(arguments, "endDate", "end_date"); ok {
		stringValue, ok := value.(string)
		if !ok || stringValue == "" {
			return options, fmt.Errorf("endDate must be a non-empty string")
		}
		options.EndDate = stringValue
	}
	if value, ok := firstValue(arguments, "directory"); ok {
		stringValue, ok := value.(string)
		if !ok || stringValue == "" {
			return options, fmt.Errorf("directory must be a non-empty string")
		}
		options.Directory = stringValue
	}
	if value, ok := firstValue(arguments, "authorName", "author_name"); ok {
		stringValue, ok := value.(string)
		if !ok {
			return options, fmt.Errorf("authorName must be a string")
		}
		options.AuthorName = stringValue
	}
	if value, ok := firstValue(arguments, "workingDirectory", "working_directory"); ok {
		stringValue, ok := value.(string)
		if !ok {
			return options, fmt.Errorf("workingDirectory must be a string")
		}
		options.WorkingDirectory = stringValue
	}

	if options.BeginDate == "" {
		return options, fmt.Errorf("beginDate is required")
	}
	if options.EndDate == "" {
		return options, fmt.Errorf("endDate is required")
	}

	return options, nil
}

func runGitFindLargeFiles(arguments map[string]interface{}) (gitFindLargeFilesStructuredResult, error) {
	options, err := parseGitFindLargeFilesOptions(arguments)
	if err != nil {
		return gitFindLargeFilesStructuredResult{}, err
	}

	scriptPath, err := resolveShellScript("SUSS_GIT_FIND_LARGE_FILES_SCRIPT", "git-find-large-files.sh")
	if err != nil {
		return gitFindLargeFilesStructuredResult{}, err
	}

	args := []string{
		scriptPath,
		"--json",
		"--directory", options.Directory,
		"--limit", strconv.Itoa(options.Limit),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", args...)
	if options.WorkingDirectory != "" {
		cmd.Dir = options.WorkingDirectory
	}

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderrText := strings.TrimSpace(string(exitErr.Stderr))
			if stderrText == "" {
				stderrText = exitErr.Error()
			}
			return gitFindLargeFilesStructuredResult{}, fmt.Errorf("git_find_large_files failed: %s", stderrText)
		}
		return gitFindLargeFilesStructuredResult{}, fmt.Errorf("git_find_large_files execution failed: %w", err)
	}

	var parsed gitFindLargeFilesScriptResult
	if err := json.Unmarshal(output, &parsed); err != nil {
		return gitFindLargeFilesStructuredResult{}, fmt.Errorf("invalid JSON from git-find-large-files script: %w", err)
	}

	files := make([]gitFindLargeFilesStructuredFile, 0, len(parsed.Files))
	for _, file := range parsed.Files {
		files = append(files, gitFindLargeFilesStructuredFile{
			ObjectID:  file.ObjectID,
			Path:      file.Path,
			SizeBytes: file.SizeBytes,
			SizeHuman: file.SizeHuman,
		})
	}

	return gitFindLargeFilesStructuredResult{
		OK:            parsed.OK,
		Directory:     parsed.Directory,
		Limit:         parsed.Limit,
		TotalCount:    parsed.TotalCount,
		ReturnedCount: parsed.ReturnedCount,
		Truncated:     parsed.Truncated,
		Files:         files,
	}, nil
}

func parseGitFindLargeFilesOptions(arguments map[string]interface{}) (gitFindLargeFilesOptions, error) {
	options := gitFindLargeFilesOptions{
		Directory: ".",
		Limit:     0,
	}

	if len(arguments) == 0 {
		return options, nil
	}

	if value, ok := firstValue(arguments, "directory"); ok {
		stringValue, ok := value.(string)
		if !ok || stringValue == "" {
			return options, fmt.Errorf("directory must be a non-empty string")
		}
		options.Directory = stringValue
	}
	if value, ok := firstValue(arguments, "limit"); ok {
		intValue, err := toInt(value)
		if err != nil || intValue < 0 {
			return options, fmt.Errorf("limit must be a non-negative integer")
		}
		options.Limit = intValue
	}
	if value, ok := firstValue(arguments, "workingDirectory", "working_directory"); ok {
		stringValue, ok := value.(string)
		if !ok {
			return options, fmt.Errorf("workingDirectory must be a string")
		}
		options.WorkingDirectory = stringValue
	}

	return options, nil
}

func runGitStatusSubdirs(arguments map[string]interface{}) (gitStatusSubdirsStructuredResult, error) {
	options, err := parseGitStatusSubdirsOptions(arguments)
	if err != nil {
		return gitStatusSubdirsStructuredResult{}, err
	}

	scriptPath, err := resolveShellScript("SUSS_GIT_STATUS_SUBDIR_SCRIPT", "git-status-subdir.sh")
	if err != nil {
		return gitStatusSubdirsStructuredResult{}, err
	}

	args := []string{
		scriptPath,
		"--json",
		"--directory", options.Directory,
		"--depth", strconv.Itoa(options.Depth),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", args...)
	if options.WorkingDirectory != "" {
		cmd.Dir = options.WorkingDirectory
	}

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderrText := strings.TrimSpace(string(exitErr.Stderr))
			if stderrText == "" {
				stderrText = exitErr.Error()
			}
			return gitStatusSubdirsStructuredResult{}, fmt.Errorf("git_status_subdirs failed: %s", stderrText)
		}
		return gitStatusSubdirsStructuredResult{}, fmt.Errorf("git_status_subdirs execution failed: %w", err)
	}

	var parsed gitStatusSubdirsScriptResult
	if err := json.Unmarshal(output, &parsed); err != nil {
		return gitStatusSubdirsStructuredResult{}, fmt.Errorf("invalid JSON from git-status-subdir script: %w", err)
	}

	repositories := make([]gitStatusSubdirsStructuredRepository, 0, len(parsed.Repositories))
	for _, repository := range parsed.Repositories {
		repositories = append(repositories, gitStatusSubdirsStructuredRepository{
			Path:      repository.Path,
			Branch:    repository.Branch,
			IsClean:   repository.IsClean,
			Porcelain: repository.Porcelain,
		})
	}

	return gitStatusSubdirsStructuredResult{
		OK:           parsed.OK,
		Directory:    parsed.Directory,
		Depth:        parsed.Depth,
		Repositories: repositories,
	}, nil
}

func parseGitStatusSubdirsOptions(arguments map[string]interface{}) (gitStatusSubdirsOptions, error) {
	options := gitStatusSubdirsOptions{
		Directory: ".",
		Depth:     2,
	}

	if len(arguments) == 0 {
		return options, nil
	}

	if value, ok := firstValue(arguments, "directory"); ok {
		stringValue, ok := value.(string)
		if !ok || stringValue == "" {
			return options, fmt.Errorf("directory must be a non-empty string")
		}
		options.Directory = stringValue
	}
	if value, ok := firstValue(arguments, "depth"); ok {
		intValue, err := toInt(value)
		if err != nil || intValue < 0 {
			return options, fmt.Errorf("depth must be a non-negative integer")
		}
		options.Depth = intValue
	}
	if value, ok := firstValue(arguments, "workingDirectory", "working_directory"); ok {
		stringValue, ok := value.(string)
		if !ok {
			return options, fmt.Errorf("workingDirectory must be a string")
		}
		options.WorkingDirectory = stringValue
	}

	return options, nil
}

func toStringSlice(value interface{}) ([]string, error) {
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return nil, nil
		}
		return []string{typed}, nil
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			stringItem, ok := item.(string)
			if !ok {
				return nil, errors.New("non-string value in array")
			}
			out = append(out, stringItem)
		}
		return out, nil
	default:
		return nil, errors.New("unsupported packages type")
	}
}

func toInt(value interface{}) (int, error) {
	switch typed := value.(type) {
	case float64:
		return int(typed), nil
	case int:
		return typed, nil
	case string:
		return strconv.Atoi(typed)
	default:
		return 0, errors.New("unsupported integer type")
	}
}

func firstValue(arguments map[string]interface{}, keys ...string) (interface{}, bool) {
	for _, key := range keys {
		value, ok := arguments[key]
		if ok {
			return value, true
		}
	}
	return nil, false
}

func resolveShellScript(explicitEnvName, scriptName string) (string, error) {
	if explicitPath := os.Getenv(explicitEnvName); explicitPath != "" {
		return ensureFile(explicitPath)
	}

	candidates := make([]string, 0, 6)

	if repoRoot := os.Getenv("SUSS_REPO_ROOT"); repoRoot != "" {
		candidates = append(candidates, filepath.Join(repoRoot, "shell", scriptName))
	}

	if repoRoot, err := gitTopLevel(); err == nil && repoRoot != "" {
		candidates = append(candidates, filepath.Join(repoRoot, "shell", scriptName))
	}

	cwd, err := os.Getwd()
	if err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "shell", scriptName),
			filepath.Join(cwd, "..", "shell", scriptName),
			filepath.Join(cwd, "..", "..", "shell", scriptName),
		)
	}

	for _, candidate := range candidates {
		resolved, err := ensureFile(candidate)
		if err == nil {
			return resolved, nil
		}
	}

	return "", fmt.Errorf("unable to resolve shell/%s; set SUSS_REPO_ROOT or %s", scriptName, explicitEnvName)
}

func gitTopLevel() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func ensureFile(path string) (string, error) {
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", resolved)
	}
	return resolved, nil
}

func negotiateProtocolVersion(requested string) string {
	if slices.Contains(supportedProtocolVersions, requested) {
		return requested
	}
	return latestProtocolVer
}

func rawIDToValue(id json.RawMessage) interface{} {
	if len(id) == 0 {
		return nil
	}

	var value interface{}
	if err := json.Unmarshal(id, &value); err != nil {
		return nil
	}
	return value
}

func marshalResponse(response responseEnvelope) ([]byte, bool) {
	encoded, err := json.Marshal(response)
	if err != nil {
		fallback, _ := json.Marshal(responseEnvelope{
			JSONRPC: "2.0",
			ID:      nil,
			Error:   &rpcError{Code: -32603, Message: "internal error", Data: err.Error()},
		})
		return fallback, true
	}
	return encoded, true
}
