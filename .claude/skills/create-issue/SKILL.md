---
name: create-issue
description: ユーザーが `/create-issue` で手動実行した場合のみ使用。ユーザーとの会話で仕様を整理し、GitHub Issueを作成する。実装は行わない。
context: fork
agent: general-purpose
allowed-tools: Bash, Read, Grep, Glob, WebSearch, WebFetch, AskUserQuestion
disable-model-invocation: false
user-invocable: true
argument-hint: "[topic]"
---

# Create Issue スキル

## 役割

ユーザーと会話しながら仕様を整理し、GitHub Issueを作成します。
実装は行いません。Issue作成のみに専念します。

## ワークフロー

### 1. ヒアリング

引数にトピックが指定されていれば、それを起点に会話を始める。
指定がなければ、何について Issue を作成したいか確認する。

`AskUserQuestion` を使って以下を確認：
- 何を実現したいか（目的・ゴール）
- 背景・動機（なぜ必要か）
- 制約・考慮事項（あれば）

### 2. 調査

必要に応じてコードベースを調査し、実現可能性や影響範囲を把握する。

```bash
# 関連ファイルの検索
# Glob, Grep, Read で調査
```

#### コメント投稿に関するIssueを作成する際の追加調査

コメント投稿処理（`gh pr comment`、`gh issue comment`、`gh api .../replies` 等）に関連するIssueの場合、`SKILL.md` のコマンドテンプレートだけでなく、同一スキルディレクトリ配下の `scripts/*.sh` も必ず確認する。

```bash
# スキルディレクトリ配下のシェルスクリプトにコメント投稿コマンドが含まれていないか確認
grep -r "gh pr comment\|gh issue comment\|gh api.*replies" .claude/skills/ --include="*.sh"
```

- コマンドが含まれている場合は、フッター付与の有無も確認する
- 修正が必要な箇所をすべてIssueに記載することで、修正漏れを防ぐ

### 3. 仕様整理

会話の内容を Issue の形にまとめる。以下の構成を基本とする：

```markdown
## 概要
（何をするか、1〜2文で）

## 背景
（なぜ必要か）

## やりたいこと
（具体的な要件・受け入れ条件）

## 実装方針（任意）
（技術的なアプローチや注意点があれば）
```

### 4. 確認

`AskUserQuestion` でユーザーにドラフト内容を提示し、承認を得る。
修正依頼があれば内容を調整して再度確認する。

### 5. Issue 作成

承認を得たら `gh issue create` で Issue を作成する。

```bash
gh issue create \
  --title "タイトル" \
  --body "$(cat <<'EOF'
## 概要
...
EOF
)"
```

作成後、Issue の URL をユーザーに報告する。

## 重要な制約

### 禁止操作
- ❌ ファイルの作成・編集（`Write`, `Edit`, `NotebookEdit`）
- ❌ `git add`, `git commit`, `git push`
- ❌ 実装作業全般

### 許可操作
- ✅ 読み取り: `Read`, `Grep`, `Glob`（調査用）
- ✅ `Bash`: `gh issue create` および読み取り専用コマンド（`git log`, `git diff`, `git status` 等）
- ✅ `WebSearch`, `WebFetch`（技術調査用）
- ✅ `AskUserQuestion`（ユーザーへの確認）

## 使用例

```bash
# トピックを指定して起動
/create-issue APIドキュメント生成機能

# トピックなしで起動（会話から始める）
/create-issue
```
