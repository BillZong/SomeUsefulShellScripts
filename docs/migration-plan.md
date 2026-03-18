# Skills / MCP 迁移计划

## 结论先行

对这个仓库，建议采用：

`本地稳定 CLI -> MCP -> Skills`

而不是：

- 直接把 Bash 脚本塞进 skill；
- 或者直接把所有脚本包成 MCP。

原因：

- 你的脚本里混合了“只读工具”“危险动作”“环境初始化”“个人 workflow”四种形态。
- `MCP` 适合工具；`Skill` 适合流程、判断、注意事项和保守操作。
- 如果底层 CLI 没有先收敛，后面不管做 MCP 还是 skill，都会变成把脆弱性包装一层再暴露出去。

## 目标架构

建议演进为下面四层：

1. `bin/` 或 `cmd/`
   - 存放稳定 CLI。
   - 每个命令都要有一致参数风格、明确退出码、可机器解析输出。
2. `internal/` 或 `lib/`
   - 放共享逻辑，避免多个脚本重复拼 `git`、`docker`、`kubectl` 命令。
3. `mcp/`
   - 暴露只读或低风险工具给 OpenClaw / 其它 Agent。
4. `skills/`
   - 暴露“什么时候该调用什么工具、哪些操作必须确认、失败如何回退”。

## 工具设计约束

所有计划继续演进的脚本，建议统一到下面契约：

- 支持 `--help`
- 支持 `--json`
- 支持 `--dry-run`（凡是有副作用的命令都应支持）
- stdout 输出结果，stderr 输出诊断
- 非零退出码只表达失败，不混杂业务状态
- 不依赖隐式当前目录；尽量显式传参
- 不在运行时自动安装依赖
- 不使用 `eval`

## 分阶段计划

### Phase 0：整理与冻结

先做仓库卫生，而不是先上协议层。

建议动作：

- 建立脚本 inventory，并固定分级口径。
- 给仓库加一个最小目录规划：
  - `archive/`
  - `docs/`
  - `bin/`
  - `mcp/`
  - `skills/`
- 把明显过时或示例性质脚本移入 `archive/`。

这一阶段的目标是“减少噪音”。

### Phase 1：重写第一批稳定 CLI

建议首批只做只读、低风险脚本：

- `shell/go-list-dep.sh`
- `shell/git-count-line.sh`
- `shell/git-find-large-files.sh`
- `shell/git-status-subdir.sh`
- `shell/docker-show-images-arch.sh`
- `shell/watch-prog-memory.sh`

建议输出风格示例：

```json
{
  "ok": true,
  "data": []
}
```

这里不要求你一开始就把它们改成 Go；Bash 也行，只要契约稳定。
但如果某个命令逻辑已经比较复杂，优先用 Go 重写更稳。

### Phase 2：做只读 MCP Server

MCP 第一阶段只暴露以下类型：

- git 统计/查询
- docker 镜像与容器信息查询
- 进程资源观测
- 目录与文件体积分析

不要在第一阶段暴露：

- 删除分支
- 改 git 历史
- 删除容器/镜像
- 修改 kubeconfig
- 导出私钥
- 强制 finalize namespace

MCP tool 命名建议统一前缀：

- `git_count_lines`
- `git_find_large_files`
- `git_status_subdirs`
- `docker_show_images_arch`
- `watch_program_memory`
- `du_directory`

## Skills 设计建议

你的 skill 不应该只是“帮我调用脚本”，而应沉淀为：

- 哪些仓库适合执行；
- 执行前检查什么；
- 什么情况先 `dry-run`；
- 什么情况禁止自动执行；
- 失败后如何回退；
- 如何解释结果给用户。

建议首批做 3 个 skill：

1. `git-maintainer`
   - 负责仓库统计、分支清理建议、对象体积分析、状态巡检。
2. `github-release-ops`
   - 负责 tag、release、产物上传、Actions 清理。
3. `k8s-ops-guarded`
   - 负责 kubeconfig 合并、镜像搬运指令生成、危险操作确认。

## High-risk 列表

下面这些能力即使未来要开放，也应至少满足：

- `--dry-run`
- 二次确认
- 明确 impact 提示
- 日志留痕

对象如下：

- `shell/eth-keystore-2-privatekey.sh`
- `shell/k8s-delete-stuck-ns.sh`
- `shell/git-rm-object.sh`
- `shell/git-delete-merge-branch.sh`
- `shell/git-clean-after-rm.sh`
- `shell/git-lean.sh`
- `shell/git-untrack-remote-branch.sh`
- `shell/docker-rm-dangling-images.sh`
- `shell/docker-rm-exited-containers.sh`

在这之前，它们默认归类为 `Local-only`。

## 为什么不是“只做 Skills”

只做 skill 的问题是：

- 跨 Agent 可移植性差；
- 工具输出不结构化；
- 长期会把经验和执行耦合在一起；
- 一旦 Agent 换平台，复用价值会下降。

## 为什么也不是“只做 MCP”

只做 MCP 的问题是：

- 很多脚本其实是流程，不是工具；
- 高风险操作需要保守规则，不只是参数 schema；
- 你的个人环境假设很多，直接公开工具会放大误用概率。

## 对“大公司现成 skills 已经够用了”的判断

不是。

更准确的说法是：

- 通用技能，大厂生态已经越来越成熟；
- 私有 workflow，仍然要靠你自己沉淀。

你的仓库恰恰大多属于后者：

- 本地环境治理；
- Git/GitHub 维护习惯；
- K8s 运维捷径；
- 发布流程；
- 私有目录布局假设。

这些内容不会被 Anthropic、OpenAI 或其它平台替你完整抽象掉。

## 最近两周可执行版本

如果只做一个现实、克制的两周计划，建议是：

### Week 1

- 补齐 `docs/script-inventory.md`
- 归档过时脚本
- 统一首批 3 个只读脚本接口
  - `go-list-dep`
  - `git-count-line`
  - `git-find-large-files`

### Week 2

- 再统一 3 个只读脚本接口
  - `git-status-subdir`
  - `docker-show-images-arch`
  - `watch-prog-memory`
- 起一个最小 MCP server
- 把 `git-maintainer` skill 写出来，先只接只读工具

## 建议的仓库方向

如果你打算长期维护，建议最终把仓库从“脚本收集箱”改成“个人 Agent 工具底座”：

- `archive/` 放历史脚本
- `bin/` 放稳定 CLI
- `mcp/` 放跨 Agent 工具协议层
- `skills/` 放经验和流程
- `docs/` 放 inventory、风险约定、迁移计划

这样以后不管你接 OpenClaw、Claude Code、Codex CLI，还是别的 Agent，都能复用同一套底层能力。

