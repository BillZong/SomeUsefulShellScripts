# SomeUsefulShellScripts MCP

这是当前仓库的最小 MCP 骨架，目标是先把低风险、结构化、可组合的能力挂出来，再逐步扩展更多工具。

当前仅暴露一个 tool：

- `go_list_dep`

## 特点

- 使用 `stdio` 传输，适合本地被 Agent 进程拉起。
- 不依赖第三方 SDK，便于在当前仓库中快速验证和维护。
- 当前通过调用 `shell/go-list-dep.sh` 提供能力，后续可以逐步把底层脚本替换成更稳定的实现。

## 运行

```bash
cd mcp
go run ./cmd/someuseful-mcp
```

## 可选环境变量

- `SUSS_REPO_ROOT`
  - 显式指定仓库根目录。
- `SUSS_GO_LIST_DEP_SCRIPT`
  - 显式指定 `go-list-dep` 脚本路径。

如果两者都不传，服务会优先尝试：

1. 当前目录所在 git 仓库根目录下的 `shell/go-list-dep.sh`
2. 当前工作目录附近的常见相对路径

## Tool: `go_list_dep`

输入参数：

- `packages`
  - `string[]`，可选，默认 `["."]`
- `includeStdlib`
  - `boolean`，可选，默认 `false`
- `testImportDepth`
  - `integer`，可选，默认 `1`
- `workingDirectory`
  - `string`，可选，用于指定执行 `go list` 时的工作目录

输出：

- `ok`
- `packages`
- `includeStdlib`
- `testImportDepth`
- `dependencies`

## 设计取舍

- 当前先不引入 tasks、resources、prompts。
- 当前先不做批量高风险操作，只暴露只读工具。
- 当前先保留 Bash 作为执行层，后续如果某个工具复杂度上来，再换成 Go 实现。

