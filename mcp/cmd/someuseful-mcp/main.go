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

type scriptResult struct {
	OK              bool     `json:"ok"`
	Packages        []string `json:"packages"`
	IncludeStdlib   bool     `json:"include_stdlib"`
	TestImportDepth int      `json:"test_import_depth"`
	Dependencies    []string `json:"dependencies"`
}

type toolStructuredResult struct {
	OK              bool     `json:"ok"`
	Packages        []string `json:"packages"`
	IncludeStdlib   bool     `json:"includeStdlib"`
	TestImportDepth int      `json:"testImportDepth"`
	Dependencies    []string `json:"dependencies"`
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

	s.initialized = true

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
		"instructions": "Prefer the read-only tool go_list_dep for Go dependency inspection. High-risk shell utilities are intentionally not exposed yet.",
	}

	return marshalResponse(responseEnvelope{
		JSONRPC: "2.0",
		ID:      rawIDToValue(req.ID),
		Result:  result,
	})
}

func (s *server) handleToolsList(req requestEnvelope) ([]byte, bool) {
	tool := map[string]interface{}{
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
				"ok": map[string]interface{}{
					"type": "boolean",
				},
				"packages": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"includeStdlib": map[string]interface{}{
					"type": "boolean",
				},
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
	}

	return marshalResponse(responseEnvelope{
		JSONRPC: "2.0",
		ID:      rawIDToValue(req.ID),
		Result: map[string]interface{}{
			"tools": []interface{}{tool},
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
	default:
		return marshalResponse(responseEnvelope{
			JSONRPC: "2.0",
			ID:      rawIDToValue(req.ID),
			Error:   &rpcError{Code: -32601, Message: "tool not found", Data: params.Name},
		})
	}
}

func runGoListDep(arguments map[string]interface{}) (toolStructuredResult, error) {
	options, err := parseGoListDepOptions(arguments)
	if err != nil {
		return toolStructuredResult{}, err
	}

	scriptPath, err := resolveGoListDepScript()
	if err != nil {
		return toolStructuredResult{}, err
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
			return toolStructuredResult{}, fmt.Errorf("go_list_dep failed: %s", stderrText)
		}
		return toolStructuredResult{}, fmt.Errorf("go_list_dep execution failed: %w", err)
	}

	var parsed scriptResult
	if err := json.Unmarshal(output, &parsed); err != nil {
		return toolStructuredResult{}, fmt.Errorf("invalid JSON from go-list-dep script: %w", err)
	}

	return toolStructuredResult{
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

func resolveGoListDepScript() (string, error) {
	if explicitPath := os.Getenv("SUSS_GO_LIST_DEP_SCRIPT"); explicitPath != "" {
		return ensureFile(explicitPath)
	}

	candidates := make([]string, 0, 6)

	if repoRoot := os.Getenv("SUSS_REPO_ROOT"); repoRoot != "" {
		candidates = append(candidates, filepath.Join(repoRoot, "shell", "go-list-dep.sh"))
	}

	if repoRoot, err := gitTopLevel(); err == nil && repoRoot != "" {
		candidates = append(candidates, filepath.Join(repoRoot, "shell", "go-list-dep.sh"))
	}

	cwd, err := os.Getwd()
	if err == nil {
		candidates = append(candidates,
			filepath.Join(cwd, "shell", "go-list-dep.sh"),
			filepath.Join(cwd, "..", "shell", "go-list-dep.sh"),
			filepath.Join(cwd, "..", "..", "shell", "go-list-dep.sh"),
		)
	}

	for _, candidate := range candidates {
		resolved, err := ensureFile(candidate)
		if err == nil {
			return resolved, nil
		}
	}

	return "", errors.New("unable to resolve shell/go-list-dep.sh; set SUSS_REPO_ROOT or SUSS_GO_LIST_DEP_SCRIPT")
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
