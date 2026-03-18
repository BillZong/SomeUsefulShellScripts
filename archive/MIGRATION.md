# archive/ 迁移清单

本文档用于管理“从活跃脚本区迁移到归档区”的第一批动作。

目标不是一次性删光旧脚本，而是把这些脚本明确标记为：

- 仅保留历史参考价值；
- 不再作为主维护对象；
- 不参与后续 `MCP` / `Skill` 能力建设。

## 迁移规则

- 迁移后保留原始文件名，降低历史检索成本。
- 第一阶段只移动“示例性、过时性、过窄用途”的脚本。
- 迁移前不要顺手重写逻辑，避免归档动作和功能改造混在一起。
- 如脚本仍被其他脚本引用，先在本清单中标记 `defer`，暂不迁移。

## 建议目录

建议归档目标路径如下：

```text
archive/
  MIGRATION.md
  shell/
```

后续实际迁移时，将待归档脚本移动到 `archive/shell/`。

## 第一批建议归档

### Ready now

- [ ] `shell/docker-change-brew-mirror.sh` -> `archive/shell/docker-change-brew-mirror.sh`
  - 原因：镜像源逻辑已过时，脚本本身也提示 deprecated。
- [ ] `shell/gh-post-release-example.sh` -> `archive/shell/gh-post-release-example.sh`
  - 原因：更像历史示例，不是稳定工具。
- [ ] `shell/mv-cpp-2-mm.sh` -> `archive/shell/mv-cpp-2-mm.sh`
  - 原因：用途过窄，且实现方式脆弱。
- [ ] `shell/pandoc-insert-images-demo.sh` -> `archive/shell/pandoc-insert-images-demo.sh`
  - 原因：属于 demo 脚本，不适合作为活跃工具维护。

### Archive after confirmation

- [ ] `shell/rm-bash-comments.sh` -> `archive/shell/rm-bash-comments.sh`
  - 原因：价值较低，且当前实现存在参数判断 bug。
  - 说明：如果你还偶尔会用它，可先修好再归档；否则直接归档也可以。

## 暂不归档

下面这些虽然要重写，但不建议直接归档，因为它们仍有明显演进价值：

- `shell/go-list-dep.sh`
- `shell/git-count-line.sh`
- `shell/git-find-large-files.sh`
- `shell/git-status-subdir.sh`
- `shell/docker-show-images-arch.sh`
- `shell/watch-prog-memory.sh`
- `shell/git-tag-with-logs.sh`
- `shell/gh-upload-release.sh`
- `shell/gh-delete-oudate-actions.sh`

## 实施前检查

在真正执行迁移前，建议逐项确认：

- [ ] 当前分支不是主分支
- [ ] 工作区已清理或已分步提交
- [ ] 没有其它脚本直接 `source` 或调用待迁移文件
- [ ] `README` 或文档中不存在仍指向旧位置的说明

## 实施顺序

建议的迁移顺序：

1. 创建 `archive/shell/`
2. 迁移 `Ready now` 列表
3. 修正文档中的路径引用
4. 单独提交一次归档 commit

## 归档 commit message 建议

可使用类似 message：

```text
chore: archive deprecated and example shell scripts
```

