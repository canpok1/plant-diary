#!/usr/bin/env bash
set -euo pipefail

cleanup() {
  git checkout main >/dev/null 2>&1 || true
}
trap cleanup EXIT

# 引数チェック
if [ $# -ne 1 ]; then
  echo "Usage: $0 <issue_number>" >&2
  exit 1
fi

ISSUE_NUMBER="$1"
if ! [[ "${ISSUE_NUMBER}" =~ ^[0-9]+$ ]]; then
  echo "Error: issue_number must be numeric" >&2
  exit 1
fi
BRANCH_NAME="feature/issue${ISSUE_NUMBER}"

echo "Issue #${ISSUE_NUMBER} の処理を開始します"

# mainブランチに切り替えて最新化
git checkout main
git pull origin main

# 作業ブランチを作成・切り替え
git checkout -b "${BRANCH_NAME}"
echo "ブランチ ${BRANCH_NAME} を作成しました"

# Claudeでissueを解決
claude --dangerously-skip-permissions "/solve-issue ${ISSUE_NUMBER}"
