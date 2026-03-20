#!/usr/bin/env bash
set -euo pipefail

# オプション解析
USE_PRINT_MODE=false
ASSIGN_COUNT=1

while getopts "pc:" opt; do
  case "$opt" in
    p) USE_PRINT_MODE=true ;;
    c)
      if ! [[ "${OPTARG}" =~ ^[0-9]+$ ]]; then
        echo "Error: -c requires a numeric value" >&2
        exit 1
      fi
      ASSIGN_COUNT="${OPTARG}"
      ;;
    *) echo "Usage: $0 [-p] [-c count]" >&2; exit 1 ;;
  esac
done

SCRIPT_DIR=$(dirname "$0")

# リモートURLからowner/repoを取得
source "$SCRIPT_DIR/lib/detect-repo.sh"

# 現在のGitHubユーザーを取得
source "$SCRIPT_DIR/lib/current-user.sh"

# assign-to-claudeラベルが付いていないopen Issueを取得（古い順、ASSIGN_COUNT件）
issue_numbers=$(gh issue list \
  --repo "$REPO" \
  --author "$CURRENT_USER" \
  --state open \
  --search "sort:created-asc" \
  --json number,labels \
  --jq '.[] | select(.labels | map(.name) | contains(["assign-to-claude"]) | not) | .number' \
  | head -n "$ASSIGN_COUNT")

if [ -z "$issue_numbers" ]; then
  echo "割り当て可能なIssueがありません"
  exit 0
fi

while IFS= read -r issue_number; do
  if "${USE_PRINT_MODE}"; then
    echo "[dry-run] Issue #${issue_number} に assign-to-claude ラベルを付与します"
  else
    echo "Issue #${issue_number} に assign-to-claude ラベルを付与します..."
    gh issue edit --repo "$REPO" "$issue_number" --add-label "assign-to-claude"
    echo "Issue #${issue_number} をキューに追加しました"
  fi
done <<< "$issue_numbers"
