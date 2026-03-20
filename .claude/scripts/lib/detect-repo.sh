#!/usr/bin/env bash
# gh repo viewでカレントディレクトリのリポジトリからowner/repoを取得し、REPO変数にセットする

REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
if [ -z "$REPO" ]; then
  echo "Error: リポジトリ情報を取得できませんでした" >&2
  # source経由（シェル関数外）でも安全に失敗するため、
  # スクリプトとして直接実行された場合はexit、sourceされた場合はreturnを使う
  # shellcheck disable=SC2317
  return 1 2>/dev/null || exit 1
fi
