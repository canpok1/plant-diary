---
name: assign-issues
description: open状態のIssueを優先度順に評価し、指定件数（デフォルト2件）に assign-to-claude ラベルを付与する。
context: fork
agent: issue-assigner
disable-model-invocation: false
user-invocable: true
allowed-tools: Skill
---

1. /monologue コマンドを使用してアサイン開始を宣言してください。
2. open状態のIssueを確認し、優先度に基づいてラベルを付与してください。引数が指定された場合、その数値をアサイン件数として使用してください（デフォルト: 2件）。
3. /monologue コマンドを使用してアサイン完了を宣言してください。
