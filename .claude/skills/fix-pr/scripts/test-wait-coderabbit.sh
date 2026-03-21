#!/bin/bash
# wait-coderabbit.sh のテストスクリプト
#
# テストリスト:
# TODO: 正常系: 第2引数なしの場合、コメント/レビューが1件以上で検出（後方互換性）
# TODO: 正常系: 第2引数=0の場合、コメント/レビューが1件以上で検出
# TODO: 正常系: 第2引数=2の場合、既存2件では待機し続け、3件目で検出
# TODO: 異常系: 第1引数なしの場合、Usage表示してexit 1
# TODO: 異常系: 第2引数が数値でない場合、エラーでexit 1
# DONE: rate limit検出時にexit 2で終了する
# DONE: rate limit検出時にstdoutにrate_limit: trueを含むJSONを出力する
# DONE: rate limit検出時のJSONにcomment_bodyフィールドが含まれる
# DONE: rate limit検出時のJSONにcomment_created_atフィールドが含まれる

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="$SCRIPT_DIR/wait-coderabbit.sh"

pass=0
fail=0

# モック用の一時ディレクトリを作成
MOCK_DIR=$(mktemp -d)
trap 'rm -rf "$MOCK_DIR"' EXIT

run_test() {
    local name="$1"
    local expected_exit="$2"
    shift 2
    local actual_exit=0
    "$@" 2>/dev/null || actual_exit=$?
    if [[ "$actual_exit" -eq "$expected_exit" ]]; then
        echo "PASS: $name"
        ((pass++)) || true
    else
        echo "FAIL: $name (expected exit=$expected_exit, got exit=$actual_exit)"
        ((fail++)) || true
    fi
}

# stdoutの内容を検証するrun_test（出力内容も確認する）
run_test_with_output() {
    local name="$1"
    local expected_exit="$2"
    local expected_pattern="$3"
    shift 3
    local actual_exit=0
    local actual_output
    actual_output=$("$@" 2>/dev/null) || actual_exit=$?
    local exit_ok=false
    local output_ok=false
    [[ "$actual_exit" -eq "$expected_exit" ]] && exit_ok=true
    echo "$actual_output" | grep -q "$expected_pattern" && output_ok=true
    if $exit_ok && $output_ok; then
        echo "PASS: $name"
        ((pass++)) || true
    else
        if ! $exit_ok; then
            echo "FAIL: $name (expected exit=$expected_exit, got exit=$actual_exit)"
        fi
        if ! $output_ok; then
            echo "FAIL: $name (expected pattern='$expected_pattern' not found in output: $actual_output)"
        fi
        ((fail++)) || true
    fi
}

# ghコマンドのモックを作成する
# $1: モックのgh prコマンドが返すJSONファイルのパス
create_gh_mock() {
    local pr_json="$1"
    cat > "$MOCK_DIR/gh" << MOCK_EOF
#!/bin/bash
# ghモック
if [[ "\$1" == "repo" && "\$2" == "view" ]]; then
    echo "owner/repo"
    exit 0
fi
if [[ "\$1" == "pr" && "\$2" == "view" ]]; then
    # --jq フラグがある場合はjqで処理
    jq_filter=""
    args=("\$@")
    for ((i=0; i<\${#args[@]}; i++)); do
        if [[ "\${args[i]}" == "--jq" ]]; then
            jq_filter="\${args[i+1]}"
            break
        fi
    done
    if [[ -n "\$jq_filter" ]]; then
        jq -r "\$jq_filter" "$pr_json"
    else
        cat "$pr_json"
    fi
    exit 0
fi
exit 0
MOCK_EOF
    chmod +x "$MOCK_DIR/gh"
}

# ---
# テスト: 第1引数なしの場合、exit 1
# ---
run_test "第1引数なしの場合exit1" 1 \
    bash "$TARGET"

# ---
# テスト: 第2引数が数値でない場合、exit 1
# ---
run_test "第2引数が数値でない場合exit1" 1 \
    env POLL_INTERVAL=0 MAX_POLLS=1 bash "$TARGET" 123 "abc"

# ---
# rate limitテスト用のPRデータを作成
# ---
RATE_LIMIT_PR_JSON="$MOCK_DIR/pr_rate_limit.json"
cat > "$RATE_LIMIT_PR_JSON" << 'JSON_EOF'
{
  "comments": [
    {
      "author": {"login": "coderabbitai"},
      "body": "I'm currently rate limited. Please wait **21 minutes and 47 seconds** before requesting another review.",
      "createdAt": "2026-03-21T12:34:56Z"
    }
  ],
  "reviews": []
}
JSON_EOF

create_gh_mock "$RATE_LIMIT_PR_JSON"

# ---
# テスト: rate limit検出時にexit 2で終了する
# ---
run_test "rate limit検出時にexit2" 2 \
    env PATH="$MOCK_DIR:$PATH" POLL_INTERVAL=0 MAX_POLLS=1 bash "$TARGET" 123

# ---
# テスト: rate limit検出時にstdoutにrate_limit: trueを含むJSONを出力する
# ---
run_test_with_output "rate limit検出時にstdoutにrate_limit:trueを含むJSON出力" 2 '"rate_limit": true' \
    env PATH="$MOCK_DIR:$PATH" POLL_INTERVAL=0 MAX_POLLS=1 bash "$TARGET" 123

# ---
# テスト: rate limit検出時のJSONにcomment_bodyフィールドが含まれる
# ---
run_test_with_output "rate limit検出時のJSONにcomment_bodyフィールド" 2 '"comment_body":' \
    env PATH="$MOCK_DIR:$PATH" POLL_INTERVAL=0 MAX_POLLS=1 bash "$TARGET" 123

# ---
# テスト: rate limit検出時のJSONにcomment_created_atフィールドが含まれる
# ---
run_test_with_output "rate limit検出時のJSONにcomment_created_atフィールド" 2 '"comment_created_at":' \
    env PATH="$MOCK_DIR:$PATH" POLL_INTERVAL=0 MAX_POLLS=1 bash "$TARGET" 123

echo ""
echo "Results: $pass passed, $fail failed"
if [[ "$fail" -gt 0 ]]; then
    exit 1
fi
