---
name: assign-issues
description: open状態のIssueを優先度順に評価し、指定件数（デフォルト2件）に assign-to-claude ラベルを付与する。
context: fork
agent: issue-assigner
disable-model-invocation: false
user-invocable: true
allowed-tools: Skill
argument-hint: "<付与する件数（省略時は2）>"
---

## コンテキスト
- repo-owner: !`gh repo view --json owner --jq .owner.login`
- repo-name: !`gh repo view --json name --jq .name`

## 手順

1. `base-tools:monologue` スキルを使用してアサイン開始を宣言する
2. Issue一覧を取得し、既に `assign-to-claude` または `in-progress-by-claude` ラベルが付いているIssueを除外する
3. issue-assigner エージェントの優先度ルールに従い、各Issueを評価して並び替える
4. 優先度が高い上位N件に `assign-to-claude` ラベルを付与する（引数指定がある場合はその数値をN、ない場合はデフォルト2件）。対象がN件未満の場合は存在する分だけ付与（0件なら何もしない）
5. ラベルを付与したIssue番号・タイトル・判定理由を出力する
6. `base-tools:monologue` スキルを使用してアサイン完了を宣言する

## コマンド

**Issue一覧の取得**
```sh
gh issue list --repo {repo-owner}/{repo-name} --state open --label "ready" --json number,title,labels,body,createdAt --limit 100
```

**ラベルの付与**
```sh
gh issue edit --repo {repo-owner}/{repo-name} {number} --add-label "assign-to-claude"
```
