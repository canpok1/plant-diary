---
name: retro
description: 作業セッションを振り返り、改善点を分析してGitHub Issueを作成します。会話セッション終了時に使用してください。
context: fork
agent: general-purpose
allowed-tools: Read, Grep, Glob, Bash(gh issue create), Bash(gh issue list)
disable-model-invocation: false
user-invocable: true
---

振り返りを行います。

1. 会話セッションでの作業内容を振り返る
  - 実施タスク
  - 使用したスキル/エージェント
  - スムーズだった点
  - 問題点
2. 改善案をまとめる
  - 改善することがなければ改善案なしとする
3. 既存のgithub issueを確認し、改善案が既にissue化されていないか確認する
4. issue化されていない改善案をgithub issueとして作成する
5. 振り返りの内容をユーザーに報告する
