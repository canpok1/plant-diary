#!/usr/bin/env bash
set -euo pipefail

# assign-to-claudeラベル付き、かつin-progress-by-claudeラベルが付いていないissueを取得
issues=$(gh issue list \
  --label "assign-to-claude" \
  --json number,labels \
  --jq '.[] | select(.labels | map(.name) | contains(["in-progress-by-claude"]) | not) | .number')

# 対象issueが存在しない場合は終了
if [ -z "$issues" ]; then
  echo "No issues to process"
  exit 0
fi

# 各issueを処理
for issue_number in $issues; do
  echo "Processing issue #$issue_number"

  # in-progress-by-claudeラベルを付与
  gh issue edit "$issue_number" --add-label "in-progress-by-claude"

  # claudeコマンドを実行
  claude -p "/solve-issue $issue_number"
done
