# 脚本清单与分级

本文档用于给当前仓库中的脚本做一次可迁移、可治理的盘点，目标不是“评判脚本好坏”，而是回答三个问题：

1. 这个脚本值不值得继续维护。
2. 它更适合沉淀为 `skill`、`MCP tool`，还是仅保留本地使用。
3. 在迁移前是否必须先重写。

## 判定标准

- `风险`
  - `L`：只读或低破坏性。
  - `M`：会修改本地环境、仓库状态或远端资源，但通常可控。
  - `H`：删除、重写历史、改集群状态、涉及密钥等高风险操作。
- `现状`
  - `keep`：可继续保留。
  - `rewrite`：保留思路，但建议先重写。
  - `archive`：更适合归档或仅作为历史参考。
- `建议去向`
  - `MCP`：适合暴露给 OpenClaw 或其它 Agent 的结构化工具。
  - `Skill`：适合保留为流程编排、经验和操作准则。
  - `Local-only`：仅本机手工使用，不建议开放给通用 Agent。
  - `Archive`：归档，不再继续演进。

## 总体判断

- 当前仓库更像“个人运维脚本盒”，还不是稳定工具产品。
- `git` 脚本数量最多，也是最值得优先整理的一组。
- 高风险脚本不适合直接变成 MCP；应先收口为本地受控命令，必要时再由 skill 进行强确认包装。
- 真正适合先做 MCP 的，是一批只读、结构化、输出明确的脚本。

## 支撑文件

- `js/keystore-to-privatekey/main.js` 是 `shell/eth-keystore-2-privatekey.sh` 的依赖实现。
- 这个 JS 子目录属于“配套工具”，不建议单独暴露；应连同外层脚本一起重构和控权。

## Inventory

| 脚本 | 领域 | 当前作用 | 主要依赖 | 风险 | 现状 | 建议去向 | 备注 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `shell/curl-time.sh` | shell/curl | 向 rc 文件追加 `timecurl` alias | `grep`, `curl` | M | rewrite | Skill | 更像环境初始化片段，不像长期工具 |
| `shell/docker-change-brew-mirror.sh` | docker/homebrew | 切换 Docker Homebrew 镜像 | `git`, `brew` 语义 | M | archive | Archive | 镜像源已过时，脚本自己也提示 deprecated |
| `shell/docker-rm-dangling-images.sh` | docker | 删除 dangling images | `docker` | H | keep | Local-only | 破坏性明确，可做受控本地命令 |
| `shell/docker-rm-exited-containers.sh` | docker | 删除 exited containers | `docker` | H | keep | Local-only | 同上 |
| `shell/docker-show-containers-command.sh` | docker | 查看容器启动命令 | `docker` | L | rewrite | MCP | 适合改成结构化只读工具 |
| `shell/docker-show-images-arch.sh` | docker | 查看镜像架构 | `docker` | L | rewrite | MCP | 非常适合先做 MCP |
| `shell/du-dir.sh` | filesystem | 查看目录体积 | `du`, `sort` | L | rewrite | MCP | 需要修复 glob/空目录兼容性 |
| `shell/eth-keystore-2-privatekey.sh` | ethereum | keystore 导出私钥 | `node`, `npm` | H | rewrite | Local-only | 涉及私钥，不建议暴露给通用 Agent |
| `shell/gh-delete-oudate-actions.sh` | GitHub Actions | 删除过期 workflow runs | `gh` | H | keep | Skill | 已完成分页、dry-run、显式确认加固；适合作为 guarded skill 的底层命令 |
| `shell/gh-post-release-example.sh` | GitHub release | tag push 后构建并上传 release | `make`, `gh-upload-release.sh` | M | archive | Archive | 更像历史示例，不像通用工具 |
| `shell/gh-upload-release.sh` | GitHub release | 创建 release 并上传产物 | `git`, `github-release`, `go` | H | rewrite | Skill | 逻辑有价值，但实现老旧且有 `eval` 风险 |
| `shell/git-clean-after-rm.sh` | git maintenance | 清理 refs/reflog/gc | `git` | H | keep | Local-only | 可保留为手工维护命令 |
| `shell/git-clean-cursor-worktrees.sh` | git/cursor | 清理 Cursor worktrees 与分支 | `find`, `git`, `rm` | H | rewrite | Skill | 强个人环境假设，不适合直接 MCP |
| `shell/git-count-line.sh` | git analytics | 按时间和作者统计增删行数 | `git`, `awk` | L | rewrite | MCP | 很适合作为只读统计工具 |
| `shell/git-delete-merge-branch.sh` | git branch | 删除已 gone 的本地分支 | `git`, `grep`, `awk` | H | keep | Local-only | 可加 dry-run，但不建议先开放 |
| `shell/git-find-large-files.sh` | git analysis | 查找仓库大文件 | `git`, `sed`, `sort`, `numfmt` | L | rewrite | MCP | 值得优先整理 |
| `shell/git-lean.sh` | git maintenance | git repack/gc/fsck | `git` | H | keep | Local-only | 本地维护价值大，通用 Agent 风险高 |
| `shell/git-pull-all-subdir.sh` | git batch ops | 批量拉取子目录仓库 | `find`, `git` | M | rewrite | Skill | 适合流程型能力，不适合裸暴露 |
| `shell/git-push-hook-wrap.sh` | git workflow | push 前后触发自定义 hook | `git` | M | keep | Skill | 更适合作为 workflow 片段 |
| `shell/git-push-new-branch.sh` | git workflow | 推送当前分支并设置 upstream | `git` | M | keep | Skill | 太薄，不必单独做 MCP |
| `shell/git-rm-object.sh` | git history rewrite | 从历史移除文件对象 | `git filter-branch` | H | rewrite | Local-only | 高风险且实现老旧，应谨慎替换为更现代方案 |
| `shell/git-status-subdir.sh` | git batch ops | 批量查看子目录仓库状态 | `find`, `git` | L | rewrite | MCP | 适合产出 JSON 的只读工具 |
| `shell/git-tag-with-logs.sh` | git release | 创建 tag，并带提交日志 | `git` | H | rewrite | Skill | 很像 release workflow 的核心步骤 |
| `shell/git-untrack-remote-branch.sh` | git maintenance | 删除 remote tracking branches | `git`, `grep`, `xargs` | H | keep | Local-only | 建议后续补 dry-run，但不先开放 |
| `shell/go-list-dep.sh` | golang | 列出依赖与测试依赖 | `go`, `xargs` | L | keep | MCP | 已经很接近一个成熟只读工具 |
| `shell/k8s-delete-stuck-ns.sh` | kubernetes | 强制删除卡住 namespace | `kubectl`, `sed` | H | keep | Local-only | 破坏性极强，不建议直接给 Agent |
| `shell/k8s-merge-config.sh` | kubernetes | 合并 kubeconfig | `kubectl` | M | rewrite | Skill | 涉及写配置文件，更适合受控流程 |
| `shell/mv-cpp-2-mm.sh` | filesystem | 批量改后缀名 | `ls`, `mv` | M | archive | Archive | 太窄、实现脆弱、价值不高 |
| `shell/pandoc-insert-images-demo.sh` | pandoc/demo | ProGit demo 文本替换 | `perl` | M | archive | Archive | 更像示例，不像长期资产 |
| `shell/pandoc-md-2-epub.sh` | pandoc | 合并 markdown 生成 epub | `pandoc`, `ls` | M | rewrite | Skill | 可做文档 workflow，但不必优先 |
| `shell/pandoc-md-2-epub3.sh` | pandoc | 合并 markdown 生成 epub3 | `pandoc`, `ls` | M | rewrite | Skill | 同上 |
| `shell/pull-k8s-docker-images.sh` | kubernetes/images | 生成镜像拉取、tag、push 指令 | `kubeadm`, `docker` | M | rewrite | Skill | 更适合“离线拉镜像操作指南” skill |
| `shell/rm-bash-comments.sh` | text | 删除 bash 注释 | `grep` | L | rewrite | Archive | 价值低，且当前参数判断有 bug |
| `shell/rm-mac-empty-dir.sh` | filesystem | 删除 `.DS_Store` 和空目录 | `find` | H | keep | Local-only | 本地清理有用，但不宜开放 |
| `shell/watch-prog-memory.sh` | observability | 周期记录程序内存占用 | `pidstat`, `awk` | L | rewrite | MCP | 适合改造成结构化观测工具 |

## 首批迁移优先级

优先迁移这 6 个：

1. `shell/go-list-dep.sh`
2. `shell/git-count-line.sh`
3. `shell/git-find-large-files.sh`
4. `shell/git-status-subdir.sh`
5. `shell/docker-show-images-arch.sh`
6. `shell/watch-prog-memory.sh`

原因：

- 都偏只读。
- 输入输出容易结构化。
- 对 OpenClaw / Claude Code / 其它 Agent 的复用价值较高。
- 风险相对可控，适合作为第一批 MCP 工具。

## 不建议先开放的脚本

下面这些即使重写，也不建议第一阶段开放为 MCP：

- `shell/eth-keystore-2-privatekey.sh`
- `shell/k8s-delete-stuck-ns.sh`
- `shell/git-rm-object.sh`
- `shell/git-delete-merge-branch.sh`
- `shell/git-clean-after-rm.sh`
- `shell/git-lean.sh`
- `shell/git-untrack-remote-branch.sh`
- `shell/docker-rm-dangling-images.sh`
- `shell/docker-rm-exited-containers.sh`

它们可以继续作为：

- 本地手工命令；
- 或者仅被带强确认的 skill 间接调用。
