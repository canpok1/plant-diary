#!/usr/bin/env bash
set -euo pipefail

# オプション解析
USE_PRINT_MODE=false
while getopts "p" opt; do
  case "$opt" in
    p) USE_PRINT_MODE=true ;;
    *) echo "Usage: $0 [-p] <issue_number>" >&2; exit 1 ;;
  esac
done
shift $((OPTIND - 1))

# 引数チェック
if [ $# -ne 1 ]; then
  echo "Usage: $0 [-p] <issue_number>" >&2
  exit 1
fi

ISSUE_NUMBER="$1"
if ! [[ "${ISSUE_NUMBER}" =~ ^[0-9]+$ ]]; then
  echo "Error: issue_number must be numeric" >&2
  exit 1
fi

SCRIPT_DIR=$(dirname "$0")
WORKSPACE_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# ロックファイル用ディレクトリの準備
LOCK_DIR="$WORKSPACE_DIR/.tmp/locks"
mkdir -p "$LOCK_DIR"

REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)

# ロックを取得（取得できない場合は別プロセスが処理中）
lock_file="$LOCK_DIR/issue-${ISSUE_NUMBER}.lock"
exec 9>"$lock_file"
if ! flock -n 9; then
  echo "Error: Issue #${ISSUE_NUMBER} is already being processed by another process." >&2
  exit 1
fi

echo "Issue #${ISSUE_NUMBER} の処理を開始します"

# 処理完了後（成否問わず）にラベルを除去
trap 'gh issue edit --repo "$REPO" "$ISSUE_NUMBER" --remove-label "in-progress-by-claude" || true' EXIT

# in-progress-by-claudeラベルを付与
gh issue edit --repo "$REPO" "$ISSUE_NUMBER" --add-label "in-progress-by-claude"

# mainブランチに切り替えて最新化
git checkout main
git pull origin main

# Claudeでissueを解決（--worktreeで自動的にブランチとワークツリーを作成）
if "${USE_PRINT_MODE}"; then
  "$SCRIPT_DIR/claude-stream.sh" --worktree "issue-${ISSUE_NUMBER}" --dangerously-skip-permissions -p "/solve-issue ${ISSUE_NUMBER}"
else
  claude --worktree "issue-${ISSUE_NUMBER}" --dangerously-skip-permissions "/solve-issue ${ISSUE_NUMBER}"
fi
