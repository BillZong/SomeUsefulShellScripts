#!/bin/bash
set -e

echo "🧹 Cursor Agent Composer Worktree Cleanup"
echo "----------------------------------------"

# Step 1: 遍历所有项目 worktrees 目录
BASE="$HOME/.cursor/worktrees"
if [ ! -d "$BASE" ]; then
  echo "No ~/.cursor/worktrees found. Nothing to clean."
  exit 0
fi

find "$BASE" -mindepth 1 -maxdepth 1 -type d | while read -r repo_dir; do
  echo "🔍 Checking repo: $repo_dir"

  # 推测原始仓库路径（Cursor 会复制结构，如 ~/.cursor/worktrees/backend/xxx）
  repo_name=$(basename "$repo_dir")

  # 推测原始仓库路径（常见在 ~/work/$repo_name 或 ~/Projects/$repo_name）
  # 你可以修改成你自己的主仓库根路径前缀
  for prefix in "$HOME/work" "$HOME/Projects" "$HOME/dev" "$HOME"; do
    main_repo="$prefix/$repo_name"
    if [ -d "$main_repo/.git" ]; then
      cd "$main_repo"
      echo "→ Found Git repo at $main_repo"
      echo "  Cleaning .git/worktrees/ ..."
      for d in .git/worktrees/*; do
        [ -d "$d" ] || continue
        wt_name=$(basename "$d")
        wt_path="$repo_dir/$wt_name"
        if [ ! -d "$wt_path" ]; then
          echo "  🗑️  Removing orphan record: $d"
          rm -rf "$d"
        fi
      done

      echo "  Removing Cursor auto branches..."
      git branch | grep -E '^[[:space:]]*[0-9]{4}-[0-9]{2}-[0-9]{2}-' | xargs -r git branch -D || true
      echo "✅ Repo $repo_name cleaned."
      echo
    fi
  done
done

echo "✨ All done! Cursor worktree cleanup complete."
