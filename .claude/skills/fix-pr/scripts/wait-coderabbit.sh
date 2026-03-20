#!/bin/bash
# CodeRabbitのレビュー到着を待機し、rate limitがあれば解消後にreviewを投稿するスクリプト
#
# 使用方法:
#   ./.claude/skills/fix-pr/scripts/wait-coderabbit.sh <PR番号> [既知のレビュー数]
#
# 引数:
#   PR番号        : 必須。対象のPR番号
#   既知のレビュー数: 省略可。この数を超えるレビューが来るまで待機する（デフォルト0）
#
# 環境変数（テスト用にオーバーライド可能）:
#   POLL_INTERVAL: ポーリング間隔（秒）デフォルト30
#   MAX_POLLS: 最大ポーリング回数 デフォルト10
#   RATE_LIMIT_WAIT: rate limit時のデフォルト待機時間（秒）デフォルト600
#
# 終了コード:
#   0: 正常完了（rate limitなし、またはrate limit解消後に再レビュー投稿済み）
#   1: エラー

set -euo pipefail

if [[ $# -lt 1 ]]; then
    echo "Usage: $0 <PR番号> [既知のレビュー数]" >&2
    exit 1
fi

PR_NUMBER="$1"
KNOWN_REVIEW_COUNT="${2:-0}"

# 第2引数が数値かどうかを検証
if ! [[ "$KNOWN_REVIEW_COUNT" =~ ^[0-9]+$ ]]; then
    echo "エラー: 既知のレビュー数は0以上の整数で指定してください: $KNOWN_REVIEW_COUNT" >&2
    exit 1
fi

POLL_INTERVAL="${POLL_INTERVAL:-30}"
MAX_POLLS="${MAX_POLLS:-10}"
RATE_LIMIT_WAIT="${RATE_LIMIT_WAIT:-600}"

REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)

# CodeRabbitのコメント数を取得
get_coderabbit_comment_count() {
    gh pr view --repo "$REPO" "$PR_NUMBER" --json comments --jq '[.comments[] | select(.author.login=="coderabbitai")] | length'
}

# CodeRabbitのレビュー数を取得
get_coderabbit_review_count() {
    gh pr view --repo "$REPO" "$PR_NUMBER" --json reviews --jq '[.reviews[] | select(.author.login=="coderabbitai")] | length'
}

# CodeRabbitのコメント/レビューが既知の数より増えているかチェック
has_new_coderabbit_response() {
    local comment_count
    local review_count
    comment_count=$(get_coderabbit_comment_count)
    review_count=$(get_coderabbit_review_count)
    local total=$(( comment_count + review_count ))
    [[ "$total" -gt "$KNOWN_REVIEW_COUNT" ]]
}

# CodeRabbitのコメントからrate limit情報をチェック（最新コメント/レビューのみ対象、1回のAPI呼び出しで取得）
check_rate_limit() {
    local pr_data
    pr_data=$(gh pr view --repo "$REPO" "$PR_NUMBER" --json comments,reviews)
    local latest_comment
    latest_comment=$(echo "$pr_data" | jq -r '[.comments[] | select(.author.login=="coderabbitai")] | last | .body // ""')
    if echo "$latest_comment" | grep -qi "rate limit"; then
        return 0  # rate limitあり
    fi
    local latest_review
    latest_review=$(echo "$pr_data" | jq -r '[.reviews[] | select(.author.login=="coderabbitai")] | last | .body // ""')
    if echo "$latest_review" | grep -qi "rate limit"; then
        return 0  # rate limitあり
    fi
    return 1  # rate limitなし
}

# CodeRabbitのレビュー到着をポーリングで待機
if [[ "$KNOWN_REVIEW_COUNT" -gt 0 ]]; then
    echo "CodeRabbitの新しいレビュー到着を待機中（既知のレビュー数: ${KNOWN_REVIEW_COUNT}）..."
else
    echo "CodeRabbitのレビュー到着を待機中..."
fi
for i in $(seq 1 "$MAX_POLLS"); do
    if has_new_coderabbit_response; then
        echo "CodeRabbitのレビューを検出しました。"

        # rate limitチェック
        if check_rate_limit; then
            echo "rate limitが検出されました。${RATE_LIMIT_WAIT}秒待機します..."
            sleep "$RATE_LIMIT_WAIT"

            # reviewを投稿
            echo "@coderabbitai review を投稿します..."
            gh pr comment --repo "$REPO" "$PR_NUMBER" --body "@coderabbitai review"
            echo "再レビューを要求しました。"
        fi

        exit 0
    fi

    if [[ "$i" -lt "$MAX_POLLS" ]]; then
        echo "ポーリング $i/$MAX_POLLS: レビュー未到着、${POLL_INTERVAL}秒待機..."
        sleep "$POLL_INTERVAL"
    fi
done

echo "エラー: CodeRabbitのレビューがタイムアウトしました（${MAX_POLLS}回ポーリング）" >&2
exit 1
