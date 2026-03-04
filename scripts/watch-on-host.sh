#!/bin/bash
set -euo pipefail

# ホスト（macOS）から devcontainer CLI 経由で watch-issue.sh を実行するスクリプト
#
# === 使い方 ===
#   ./scripts/watch-on-host.sh
#
# === 前提条件 ===
# - devcontainer CLI がインストールされていること
#   インストール方法: npm install -g @devcontainers/cli
# - Docker が起動していること
#
# === 動作フロー ===
# 1. devcontainer up でコンテナを起動（既存コンテナがある場合は再利用）
# 2. devcontainer exec でコンテナ内の watch-issue.sh を実行

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
WORKSPACE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=== コンテナを起動中... ==="
devcontainer up --workspace-folder "${WORKSPACE_DIR}"

echo ""
echo "=== watch-issue.sh を実行中... ==="
devcontainer exec --workspace-folder "${WORKSPACE_DIR}" \
  bash -c ".claude/scripts/watch-issue.sh"
