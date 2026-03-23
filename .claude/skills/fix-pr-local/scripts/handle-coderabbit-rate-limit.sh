#!/bin/bash
# CodeRabbitのrate limit検出後に「待機→レビュー依頼→新規レビュー到着待機」を自動で行うスクリプト
#
# 使用方法:
#   ./.claude/skills/fix-pr-local/scripts/handle-coderabbit-rate-limit.sh <PR番号> <残り待機秒数>
#
# 引数:
#   PR番号      : 必須。対象のPR番号
#   残り待機秒数: 必須。AIが算出した残り待機秒（0以下なら即進む）
#
# 環境変数（テスト用にオーバーライド可能）:
#   POLL_INTERVAL: ポーリング間隔（秒）デフォルト30
#   MAX_POLLS    : 最大ポーリング回数 デフォルト30（=15分）
#
# 終了コード:
#   0: 正常完了（CodeRabbitの新規レビューを検出）
#   1: タイムアウト（15分待機しても新規レビューが来なかった）

set -euo pipefail

if [[ $# -lt 2 ]]; then
    echo "Usage: $0 <PR番号> <残り待機秒数>" >&2
    exit 1
fi

PR_NUMBER="$1"
WAIT_SECONDS="$2"

# 引数の型チェック
if ! [[ "$PR_NUMBER" =~ ^[0-9]+$ ]]; then
    echo "エラー: PR番号は正の整数で指定してください: $PR_NUMBER" >&2
    exit 1
fi

if ! [[ "$WAIT_SECONDS" =~ ^-?[0-9]+$ ]]; then
    echo "エラー: 残り待機秒数は整数で指定してください: $WAIT_SECONDS" >&2
    exit 1
fi

POLL_INTERVAL="${POLL_INTERVAL:-30}"
MAX_POLLS="${MAX_POLLS:-30}"

REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)

# CodeRabbitのコメント+レビューの合計数を取得
get_coderabbit_total_count() {
    gh pr view --repo "$REPO" "$PR_NUMBER" --json comments,reviews \
        --jq '([.comments[] | select(.author.login=="coderabbitai")] | length)
            + ([.reviews[]  | select(.author.login=="coderabbitai")] | length)'
}

# ========================================
# Step 1: rate limit 待機
# ========================================
if [[ "$WAIT_SECONDS" -gt 0 ]]; then
    echo "CodeRabbitのrate limit待機中（${WAIT_SECONDS}秒）..."
    sleep "$WAIT_SECONDS"
fi

# ========================================
# Step 2: レビュー依頼コメント投稿前のカウント記録
# ========================================
echo "現在のCodeRabbitアクティビティ数を記録中..."
KNOWN_COUNT=$(get_coderabbit_total_count)
echo "現在のカウント: ${KNOWN_COUNT}"

# ========================================
# Step 3: レビュー依頼コメント投稿
# ========================================
echo "CodeRabbitにレビューを依頼します..."
gh pr comment --repo "$REPO" "$PR_NUMBER" --body "@coderabbitai review

---
🤖 Generated with [Claude Code](https://claude.ai/claude-code)"

# ========================================
# Step 4: トリガー確認待機（60秒）
# ========================================
echo "CodeRabbitのトリガー確認を60秒待機中..."
sleep 60

# ========================================
# Step 5: 新規レビュー到着待機（ポーリング）
# ========================================
echo "CodeRabbitの新規レビューを待機中（最大 $((MAX_POLLS * POLL_INTERVAL / 60)) 分）..."

for i in $(seq 1 "$MAX_POLLS"); do
    CURRENT_COUNT=$(get_coderabbit_total_count)
    if [[ "$CURRENT_COUNT" -gt "$KNOWN_COUNT" ]]; then
        echo "CodeRabbitの新規レビューを検出しました（カウント: ${KNOWN_COUNT} → ${CURRENT_COUNT}）。"
        exit 0
    fi

    if [[ "$i" -lt "$MAX_POLLS" ]]; then
        echo "ポーリング ${i}/${MAX_POLLS}: レビュー未到着、${POLL_INTERVAL}秒待機..."
        sleep "$POLL_INTERVAL"
    fi
done

echo "タイムアウト: CodeRabbitの新規レビューが${MAX_POLLS}回ポーリングで確認できませんでした。" >&2
exit 1
