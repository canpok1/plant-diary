#!/usr/bin/env bash
set -euo pipefail

# 引数チェック
if [ $# -ne 1 ]; then
  echo "Usage: $0 <issue_number>" >&2
  exit 1
fi

ISSUE_NUMBER="$1"
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

# mainブランチに戻る
git checkout main
echo "mainブランチに戻りました"
