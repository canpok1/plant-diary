#!/bin/bash
# wait-coderabbit.sh のテストスクリプト
#
# テストリスト:
# TODO: 正常系: 第2引数なしの場合、コメント/レビューが1件以上で検出（後方互換性）
# TODO: 正常系: 第2引数=0の場合、コメント/レビューが1件以上で検出
# TODO: 正常系: 第2引数=2の場合、既存2件では待機し続け、3件目で検出
# TODO: 異常系: 第1引数なしの場合、Usage表示してexit 1
# TODO: 異常系: 第2引数が数値でない場合、エラーでexit 1

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="$SCRIPT_DIR/wait-coderabbit.sh"

pass=0
fail=0

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

echo ""
echo "Results: $pass passed, $fail failed"
if [[ "$fail" -gt 0 ]]; then
    exit 1
fi
