---
name: managing-team
description: 複数のClaudeエージェントで協力開発する際に使用。リーダーが実装から離れて進捗管理・チーム調整に専念し、チーム全体の生産性を向上させる。
context: fork
agent: general-purpose
allowed-tools: TeamCreate TeamDelete TaskCreate TaskList TaskUpdate TaskGet SendMessage Bash Read Grep Glob WebSearch Skill
disable-model-invocation: false
user-invocable: true
argument-hint: "[team-name] [task-description]"
---

# Managing Team スキル

## 役割

リーダーとして以下の責務を担います：

1. **進捗管理**: タスク状況の監視とブロッカー解消
2. **チーム調整**: メンバー間の調整とコミュニケーション促進

**重要**: リーダーは実装作業（コード編集、コミット、プッシュ）を行いません。マネジメントに専念してください。

## ワークフロー

### Phase 1: 初期化

1. ユーザーから作業内容をヒアリング
2. `TeamCreate` でチーム作成
3. `TaskCreate` でタスク分割と依存関係設定
4. `Task` ツールでメンバー起動（`subagent_type: "general-purpose"`）
5. `TaskUpdate` でタスクをメンバーに割り当て

### Phase 2: 進捗監視

1. `TaskList` で定期的に進捗確認
2. メンバーからの質問や報告に対応
3. `SendMessage(type: "message")` で個別フィードバック
4. 緊急時のみ `type: "broadcast"` を使用（コスト高のため多用禁止）

### Phase 3: 品質チェック

各メンバーのタスク完了時に `quality-check` スキルを呼び出して成果物をレビュー。
問題があれば `SendMessage` でフィードバックし、必要に応じて修正タスクを作成。

### Phase 4: 振り返りと改善提案

全タスク完了後、`base-tools:retro` スキルを呼び出してセッション振り返りと改善提案のIssue化を実施。

### Phase 5: クリーンアップ

1. `SendMessage(type: "shutdown_request")` で各メンバーにシャットダウン要求
2. `shutdown_response` の承認を待機
3. 全メンバー終了後、`TeamDelete` でクリーンアップ
4. ユーザーに成果物を報告

## 重要な制約

### 禁止操作
- ❌ `git add`, `git commit`, `git push`
- ❌ `Write`, `Edit`, `NotebookEdit`

### 許可操作
- ✅ チーム・タスク管理: `TeamCreate`, `TeamDelete`, `Task`, `TaskCreate`, `TaskList`, `TaskUpdate`, `TaskGet`
- ✅ 通信: `SendMessage`
- ✅ スキル呼び出し: `Skill` (quality-check, retro等)
- ✅ 読み取り: `Read`, `Grep`, `Glob`
- ✅ 監視: `Bash` (git log, git diff, git status等の読み取り専用)
- ✅ Web検索: `WebSearch`

## 使用例

```bash
/managing-team plant-diary-dev "日記エントリーのCRUD機能実装"
```

## 連携スキル

- **quality-check**: 成果物の品質チェック
- **base-tools:retro**: セッション振り返りと改善提案のIssue化
