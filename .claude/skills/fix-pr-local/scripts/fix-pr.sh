#!/bin/bash
# ブランチ最新化・auto-merge有効化・CI待機・PRマージを行うスクリプト
#
# 使用方法:
#   ./.claude/skills/fix-pr-local/scripts/fix-pr.sh <PR番号>
#
# 終了コード:
#   0:  マージ完了
#   1:  スクリプトエラー（コンフリクト含む）
#   10: CI失敗（失敗チェック一覧を stderr に出力）
#   20: 未解決レビュースレッドが存在
#   30: CI通過・スレッド全解決済みだが未承認

set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <PR番号>" >&2
    exit 1
fi

PR_NUMBER="$1"

REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
OWNER=$(echo "$REPO" | cut -d/ -f1)
REPO_NAME=$(echo "$REPO" | cut -d/ -f2)

# ========================================
# Step 1: マージ済み確認
# ========================================
echo "マージ済み確認中..."

PR_STATE=$(gh pr view --repo "$REPO" "$PR_NUMBER" --json state --jq '.state')
if [[ "$PR_STATE" == "MERGED" ]]; then
    echo "PRはマージ済みです。"
    exit 0
fi

# ========================================
# Step 2: ブランチ最新化
# ========================================
echo "ブランチ最新化チェック中..."

git fetch origin main 2>/dev/null || true

MERGE_BASE=$(git merge-base HEAD origin/main 2>/dev/null || echo "")
REMOTE_MAIN=$(git rev-parse origin/main 2>/dev/null || echo "")

if [[ "$MERGE_BASE" != "$REMOTE_MAIN" ]]; then
    echo "ブランチがmainより遅れています。マージします..."
    if ! git merge origin/main 2>/dev/null; then
        echo "conflict: コンフリクトが発生しました。マージを中断します。" >&2
        git merge --abort 2>/dev/null || true
        exit 1
    fi
    echo "mainをマージしました。プッシュします..."
    if ! git push origin HEAD; then
        echo "プッシュに失敗しました。" >&2
        exit 1
    fi
else
    echo "ブランチは最新です。"
fi

# ========================================
# Step 3: リポジトリ設定取得（1回のAPI呼び出しで全設定を取得）
# ========================================
echo "リポジトリ設定を確認中..."

REPO_CONFIG=$(gh api "repos/$REPO" --jq '{
  allow_auto_merge: (.allow_auto_merge // false),
  allow_squash_merge: (.allow_squash_merge // false),
  allow_merge_commit: (.allow_merge_commit // false),
  allow_rebase_merge: (.allow_rebase_merge // false)
}')

# マージ方式を決定
MERGE_OPTION=""
if echo "$REPO_CONFIG" | jq -e '.allow_squash_merge' > /dev/null; then
    MERGE_OPTION="--squash"
elif echo "$REPO_CONFIG" | jq -e '.allow_merge_commit' > /dev/null; then
    MERGE_OPTION="--merge"
elif echo "$REPO_CONFIG" | jq -e '.allow_rebase_merge' > /dev/null; then
    MERGE_OPTION="--rebase"
else
    echo "許可されたマージ方式がありません。リポジトリ設定を確認してください。" >&2
    exit 1
fi

echo "マージ方式: $MERGE_OPTION"

# ========================================
# Step 4: auto-merge 有効化（冪等・失敗は無視）
# ========================================
ALLOW_AUTO_MERGE=$(echo "$REPO_CONFIG" | jq -r '.allow_auto_merge')
if [[ "$ALLOW_AUTO_MERGE" == "true" ]]; then
    echo "auto-mergeを有効化します..."
    # shellcheck disable=SC2086
    gh pr merge --repo "$REPO" "$PR_NUMBER" --auto $MERGE_OPTION 2>/dev/null || true
else
    echo "このリポジトリはauto-mergeが無効です。スキップします。"
fi

# ========================================
# Step 5: CI完了待機
# ========================================
echo "CI完了を待機中..."

if ! gh pr checks --repo "$REPO" "$PR_NUMBER" --watch --interval 15; then
    echo "CIに失敗したチェックがあります。" >&2
fi

# ========================================
# Step 6: マージ済み再確認（auto-mergeで完了した場合）
# ========================================
echo "マージ済み再確認中..."

PR_STATE=$(gh pr view --repo "$REPO" "$PR_NUMBER" --json state --jq '.state')
if [[ "$PR_STATE" == "MERGED" ]]; then
    echo "PRはauto-mergeで完了しました。"
    exit 0
fi

# ========================================
# Step 7: ブロッカー判定
# ========================================
echo "ブロッカーを確認中..."

# CI失敗チェック
FAILED_CHECKS=$(gh pr view --repo "$REPO" "$PR_NUMBER" --json statusCheckRollup \
    --jq '[.statusCheckRollup[] | select((.state // .status // .conclusion) | test("FAILURE|ERROR|CANCELLED|failed|error|cancelled"; "i")) | (.context // .name)] | join(", ")')

if [[ -n "$FAILED_CHECKS" ]]; then
    echo "CI失敗: $FAILED_CHECKS" >&2
    exit 10
fi

# 未解決スレッドチェック
# shellcheck disable=SC2016
UNRESOLVED_THREADS=$(gh api graphql -f query='
  query($owner: String!, $repo: String!, $pr: Int!) {
    repository(owner: $owner, name: $repo) {
      pullRequest(number: $pr) {
        reviewThreads(first: 100) {
          nodes {
            isResolved
          }
        }
      }
    }
  }
' -f owner="$OWNER" -f repo="$REPO_NAME" -F pr="$PR_NUMBER" \
    --jq '.data.repository.pullRequest.reviewThreads.nodes | map(select(.isResolved == false)) | length')

if [[ "$UNRESOLVED_THREADS" -gt 0 ]]; then
    echo "未解決レビュースレッドが ${UNRESOLVED_THREADS} 件あります。" >&2
    exit 20
fi

# 承認状態チェック
REVIEW_DECISION=$(gh pr view --repo "$REPO" "$PR_NUMBER" --json reviewDecision --jq '.reviewDecision // ""')
if [[ "$REVIEW_DECISION" == "CHANGES_REQUESTED" || "$REVIEW_DECISION" == "REVIEW_REQUIRED" ]]; then
    echo "未承認: reviewDecision=${REVIEW_DECISION}" >&2
    exit 30
fi

# ========================================
# Step 8: マージ試行
# ========================================
echo "PRマージを試行中..."

MERGE_OUTPUT=""
MERGE_EXIT=0
# shellcheck disable=SC2086
MERGE_OUTPUT=$(gh pr merge --repo "$REPO" "$PR_NUMBER" $MERGE_OPTION 2>&1) || MERGE_EXIT=$?

if [[ "$MERGE_EXIT" -eq 0 ]]; then
    echo "PRをマージしました。"
    exit 0
fi

echo "マージに失敗しました。" >&2
echo "$MERGE_OUTPUT" >&2
exit 1
