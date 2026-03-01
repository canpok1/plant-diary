#!/usr/bin/env bash
set -euo pipefail

INTERVAL_SECONDS=60

# Ctrl-C（SIGINT）で正常終了するためのトラップ
trap 'echo "Stopping watch-issue.sh..."; exit 0' INT

while true; do
  # assign-to-claudeラベル付き、かつin-progress-by-claudeラベルが付いていないissueを1件取得（古い順）
  issue_number=$(gh issue list \
    --label "assign-to-claude" \
    --sort created \
    --order asc \
    --json number,labels \
    --jq '.[] | select(.labels | map(.name) | contains(["in-progress-by-claude"]) | not) | .number' \
    | head -n 1)

  # 対象issueが存在しない場合
  if [ -z "$issue_number" ]; then
    echo "No issues to process"
  else
    echo "Processing issue #$issue_number"

    # in-progress-by-claudeラベルを付与
    gh issue edit "$issue_number" --add-label "in-progress-by-claude"

    # エラー時にラベルを削除するトラップを設定
    trap "gh issue edit \"$issue_number\" --remove-label \"in-progress-by-claude\" || true" ERR

    # solve-issue.shを実行
    "$(dirname "$0")/solve-issue.sh" -p "$issue_number"

    # ERRトラップを解除
    trap - ERR
  fi

  # 一定時間待機
  echo "Waiting ${INTERVAL_SECONDS} seconds..."
  sleep "$INTERVAL_SECONDS"
done
