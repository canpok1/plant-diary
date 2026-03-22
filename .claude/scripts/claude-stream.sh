#!/usr/bin/env bash
set -euo pipefail

# claude-stream.sh - stream-json形式でClaudeを呼び出すラッパースクリプト
#
# 使い方: claude-stream.sh [claudeのオプション/プロンプト]
#
# --output-format stream-json --verbose --include-partial-messages を付与して
# Claudeを呼び出し、jqでテキストデルタとresultのみを表示する。

claude "$@" \
  --output-format stream-json --verbose --include-partial-messages | \
  jq -rj 'if .type == "stream_event" and .event.delta.type? == "text_delta" then .event.delta.text elif .type == "result" then .result else empty end'
echo
