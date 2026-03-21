#!/bin/bash
# CodeRabbitのレビュー到着を待機するスクリプト
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
#
# 終了コード:
#   0: 正常完了（rate limitなし）
#   1: エラー（タイムアウト等）
#   2: rate limit検出（stdoutにJSONを出力済み）

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

REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)

# CodeRabbitのコメント+レビューの合計数を1回のAPI呼び出しで取得
get_coderabbit_total_count() {
    gh pr view --repo "$REPO" "$PR_NUMBER" --json comments,reviews \
        --jq '([.comments[] | select(.author.login=="coderabbitai")] | length)
            + ([.reviews[]  | select(.author.login=="coderabbitai")] | length)'
}

# CodeRabbitのコメント/レビューが既知の数より増えているかチェック
has_new_coderabbit_response() {
    local total
    total=$(get_coderabbit_total_count)
    [[ "$total" -gt "$KNOWN_REVIEW_COUNT" ]]
}

# CodeRabbitのコメントからrate limit情報をチェック（最新コメント/レビューのみ対象、1回のAPI呼び出しで取得）
# rate limit検出時: JSONをstdoutに出力してexit 2で終了
# rate limitなし: return 1
check_rate_limit() {
    local pr_data
    pr_data=$(gh pr view --repo "$REPO" "$PR_NUMBER" --json comments,reviews)

    local latest_comment_body latest_comment_created_at
    latest_comment_body=$(echo "$pr_data" | jq -r '[.comments[] | select(.author.login=="coderabbitai")] | last | .body // ""')
    latest_comment_created_at=$(echo "$pr_data" | jq -r '[.comments[] | select(.author.login=="coderabbitai")] | last | .createdAt // ""')

    if echo "$latest_comment_body" | grep -qi "rate limit"; then
        echo "{\"rate_limit\": true, \"comment_body\": $(echo "$latest_comment_body" | jq -Rs .), \"comment_created_at\": \"$latest_comment_created_at\"}"
        exit 2
    fi

    local latest_review_body latest_review_created_at
    latest_review_body=$(echo "$pr_data" | jq -r '[.reviews[] | select(.author.login=="coderabbitai")] | last | .body // ""')
    latest_review_created_at=$(echo "$pr_data" | jq -r '[.reviews[] | select(.author.login=="coderabbitai")] | last | .submittedAt // ""')

    if echo "$latest_review_body" | grep -qi "rate limit"; then
        echo "{\"rate_limit\": true, \"comment_body\": $(echo "$latest_review_body" | jq -Rs .), \"comment_created_at\": \"$latest_review_created_at\"}"
        exit 2
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

        # rate limitチェック（rate limit検出時はJSONをstdoutに出力してexit 2）
        check_rate_limit || true

        exit 0
    fi

    if [[ "$i" -lt "$MAX_POLLS" ]]; then
        echo "ポーリング $i/$MAX_POLLS: レビュー未到着、${POLL_INTERVAL}秒待機..."
        sleep "$POLL_INTERVAL"
    fi
done

echo "エラー: CodeRabbitのレビューがタイムアウトしました（${MAX_POLLS}回ポーリング）" >&2
exit 1
