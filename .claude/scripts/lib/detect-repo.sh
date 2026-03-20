#!/usr/bin/env bash
# gh repo viewでカレントディレクトリのリポジトリからowner/repoを取得し、REPO変数にセットする

REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
if [ -z "$REPO" ]; then
  echo "Error: リポジトリ情報を取得できませんでした" >&2
  exit 1
fi
