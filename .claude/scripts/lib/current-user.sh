#!/usr/bin/env bash
# 現在のGitHubユーザーを取得し、CURRENT_USER変数にセットする

CURRENT_USER=$(gh api user --jq .login)
if [ -z "$CURRENT_USER" ]; then
  echo "Error: GitHubユーザー情報を取得できませんでした" >&2
  exit 1
fi
