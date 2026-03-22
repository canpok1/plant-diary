#!/usr/bin/env bash
set -euo pipefail

# オプション解析
USE_PRINT_MODE=false
ASSIGN_COUNT=2
while getopts "pc:" opt; do
  case "$opt" in
    p) USE_PRINT_MODE=true ;;
    c) ASSIGN_COUNT="$OPTARG" ;;
    *) echo "Usage: $0 [-p] [-c assign_count]" >&2; exit 1 ;;
  esac
done

if ! [[ "${ASSIGN_COUNT}" =~ ^[0-9]+$ ]]; then
  echo "Error: assign_count must be numeric" >&2
  exit 1
fi

echo "Issue自動選定を開始します"

CURRENT_USER=$(gh api user --jq .login)
REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)

# readyラベル付きかつ割り当て済みラベルなしのIssue数を確認し、0件ならスキップ
ASSIGNABLE_COUNT=$(gh issue list --repo "$REPO" --state open --label "ready" --author "$CURRENT_USER" --json number,labels \
  --jq '[.[] | select(.labels | map(.name) | all(. != "assign-to-claude" and . != "in-progress-by-claude"))] | length')
if [ "$ASSIGNABLE_COUNT" -eq 0 ]; then
  echo "割り当て可能なIssueがないため、スキップします"
  exit 0
fi

SCRIPT_DIR=$(dirname "$0")

# Claudeでissueを選定・ラベル付与（コード変更不要のため--worktreeは不使用）
SKILL_ARGS="/assign-issues ${ASSIGN_COUNT}"
if "${USE_PRINT_MODE}"; then
  "$SCRIPT_DIR/claude-stream.sh" --dangerously-skip-permissions -p "$SKILL_ARGS"
else
  claude --dangerously-skip-permissions "$SKILL_ARGS"
fi
