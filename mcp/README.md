# SomeUsefulShellScripts MCP

这是当前仓库的最小 MCP 骨架，目标是先把低风险、结构化、可组合的能力挂出来，再逐步扩展更多工具。

当前暴露五个 tool：

- `go_list_dep`
- `git_count_line`
- `git_find_large_files`
- `git_status_subdirs`
- `docker_show_images_arch`

## 特点

- 使用 `stdio` 传输，适合本地被 Agent 进程拉起。
- 不依赖第三方 SDK，便于在当前仓库中快速验证和维护。
- 当前通过调用 `shell/` 下的 Bash CLI 提供能力，后续可以逐步把底层脚本替换成更稳定的实现。

## 运行

```bash
cd mcp
go run ./cmd/someuseful-mcp
```

## 接入示例

### Claude Code

如果你希望把这个 MCP 作为当前项目的共享配置写入 `.mcp.json`，可以在仓库根目录执行：

```bash
cd /absolute/path/to/SomeUsefulShellScripts

claude mcp add --transport stdio --scope project \
  -e SUSS_REPO_ROOT="$PWD" \
  someuseful-shell-scripts -- \
  go run "$PWD/mcp/cmd/someuseful-mcp"
```

执行后可以用下面的命令确认是否已注册：

```bash
claude mcp list
```

进入 Claude Code 后，也可以通过 `/mcp` 查看服务器状态。

如果你更希望手工维护项目级 `.mcp.json`，可以参考下面的最小示例：

```json
{
  "mcpServers": {
    "someuseful-shell-scripts": {
      "command": "go",
      "args": [
        "run",
        "/absolute/path/to/SomeUsefulShellScripts/mcp/cmd/someuseful-mcp"
      ],
      "env": {
        "SUSS_REPO_ROOT": "/absolute/path/to/SomeUsefulShellScripts"
      }
    }
  }
}
```

示例对话：

```text
Use the MCP tool `go_list_dep` from `someuseful-shell-scripts`
to inspect packages ["fmt"] with includeStdlib=true and testImportDepth=0.
```

### OpenClaw

OpenClaw 自身的 CLI fallback backend 不直接消费这个仓库的 MCP 配置；更实用的接法是：

1. 先按上面的 Claude Code 示例把这个 MCP 注册到本机 `claude` CLI。
2. 再让 OpenClaw 使用内置的 `claude-cli/...` backend。

如果你的 OpenClaw Gateway 运行环境找不到 `claude`，可以在 OpenClaw 配置里显式指定命令路径：

```json5
{
  agents: {
    defaults: {
      cliBackends: {
        "claude-cli": {
          command: "/opt/homebrew/bin/claude"
        }
      }
    }
  }
}
```

然后用 `claude-cli/...` 模型发起请求：

```bash
openclaw agent \
  --model claude-cli/opus-4.6 \
  --message 'Use the configured MCP server `someuseful-shell-scripts` and call `go_list_dep` for packages ["fmt"].'
```

这里真正执行 MCP tool 的仍然是 `claude` CLI；OpenClaw 负责把会话路由到 `claude-cli` backend。

## 可选环境变量

- `SUSS_REPO_ROOT`
  - 显式指定仓库根目录。
- `SUSS_GO_LIST_DEP_SCRIPT`
  - 显式指定 `go-list-dep` 脚本路径。
- `SUSS_GIT_COUNT_LINE_SCRIPT`
  - 显式指定 `git-count-line.sh` 脚本路径。
- `SUSS_GIT_FIND_LARGE_FILES_SCRIPT`
  - 显式指定 `git-find-large-files.sh` 脚本路径。
- `SUSS_GIT_STATUS_SUBDIR_SCRIPT`
  - 显式指定 `git-status-subdir.sh` 脚本路径。
- `SUSS_DOCKER_SHOW_IMAGES_ARCH_SCRIPT`
  - 显式指定 `docker-show-images-arch.sh` 脚本路径。

如果这些变量都不传，服务会优先尝试：

1. 当前目录所在 git 仓库根目录下的 `shell/<script>.sh`
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

## Tool: `git_count_line`

输入参数：

- `beginDate`
  - `string`，必填，例如 `2024-01-01`
- `endDate`
  - `string`，必填，例如 `2026-01-01`
- `directory`
  - `string`，可选，默认 `"."`
- `authorName`
  - `string`，可选，默认取 `git config user.name`
- `workingDirectory`
  - `string`，可选，用于指定底层脚本的启动目录

输出：

- `ok`
- `beginDate`
- `endDate`
- `directory`
- `authorName`
- `addedLines`
- `removedLines`
- `totalLines`

## Tool: `git_find_large_files`

输入参数：

- `directory`
  - `string`，可选，默认 `"."`
- `limit`
  - `integer`，可选，默认 `0`，表示不限制返回数量
- `workingDirectory`
  - `string`，可选，用于指定底层脚本的启动目录

输出：

- `ok`
- `directory`
- `limit`
- `totalCount`
- `returnedCount`
- `truncated`
- `files`
  - `objectId`
  - `path`
  - `sizeBytes`
  - `sizeHuman`

## Tool: `git_status_subdirs`

输入参数：

- `directory`
  - `string`，可选，默认 `"."`
- `depth`
  - `integer`，可选，默认 `2`
- `workingDirectory`
  - `string`，可选，用于指定底层脚本的启动目录

输出：

- `ok`
- `directory`
- `depth`
- `repositories`
  - `path`
  - `branch`
  - `isClean`
  - `porcelain`

## Tool: `docker_show_images_arch`

输入参数：

- `workingDirectory`
  - `string`，可选，用于指定底层脚本的启动目录

输出：

- `ok`
- `images`
  - `id`
  - `repoTags`
  - `architecture`

## 设计取舍

- 当前先不引入 tasks、resources、prompts。
- 当前先不做批量高风险操作，只暴露只读工具。
- 当前先保留 Bash 作为执行层，后续如果某个工具复杂度上来，再换成 Go 实现。
