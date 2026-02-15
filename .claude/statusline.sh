#!/bin/bash

# 標準入力からJSON形式でセッション情報を受け取る
input=$(cat)

# ヘルパー関数
get_model_name() { echo "$input" | jq -r '.model.display_name // "Unknown"'; }
get_context_window_size() { echo "$input" | jq -r '.context_window.context_window_size // 0'; }
get_current_usage() { echo "$input" | jq '.context_window.current_usage'; }
get_cost() { echo "$input" | jq -r '.cost.total_cost_usd // empty'; }

# コンテキスト使用率を算出
calc_context_percent() {
  local context_size usage
  context_size=$(get_context_window_size)
  usage=$(get_current_usage)

  if [ "$usage" != "null" ] && [[ "$context_size" =~ ^[1-9][0-9]*$ ]]; then
    echo "$usage" | jq -r "(.input_tokens + .cache_creation_input_tokens + .cache_read_input_tokens) * 100 / $context_size | floor"
  else
    echo "0"
  fi
}

# 各値を取得
MODEL=$(get_model_name)
CONTEXT_PERCENT=$(calc_context_percent)
COST=$(get_cost)

# ステータスラインを出力
if [ -n "$COST" ]; then
  printf "Model: %s | Context: %s%% | Cost: \$%.2f" "$MODEL" "$CONTEXT_PERCENT" "$COST"
else
  printf "Model: %s | Context: %s%%" "$MODEL" "$CONTEXT_PERCENT"
fi
