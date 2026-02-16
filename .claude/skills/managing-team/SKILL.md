---
name: managing-team
description: 複数のClaudeエージェントで協力開発する際に使用。リーダーが実装から離れて進捗管理・品質チェック・振り返りに専念し、チーム全体の生産性と品質を向上させる。
context: fork
agent: general-purpose
allowed-tools: TeamCreate TeamDelete TaskCreate TaskList TaskUpdate TaskGet SendMessage Bash Read Grep Glob WebSearch
disable-model-invocation: false
user-invocable: true
argument-hint: "[team-name] [task-description]"
---

# Managing Team スキル

## 役割

リーダーとして以下の責務を担います：

1. **進捗管理**: タスク状況の監視とブロッカー解消
2. **品質チェック**: 成果物の品質基準確認
3. **振り返り**: 学習事項の抽出
4. **改善提案**: プロジェクト資産の改善をIssue化

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

`references/quality-checklist.md` の基準でメンバーの成果物をレビュー：

- コミットメッセージ規約（「{機能名}を{動詞}」形式）
- Co-Authored-By クレジット
- コード規約、エラーハンドリング、テスト
- 問題があれば `SendMessage` でフィードバック、必要に応じて修正タスク作成

### Phase 4: 振り返り

`templates/retrospective.md` を参照して実施：

1. `SendMessage(type: "broadcast")` で振り返り開始を通知
2. 各メンバーに質問（担当タスク、学び、課題、連携、改善提案）
3. リーダー自身の分析（タスク設計、依存関係管理、コミュニケーション）
4. プロジェクト資産への反映が必要な項目を抽出

### Phase 5: 改善提案のIssue化

振り返りから抽出した改善点を評価：

**Issue化基準**:
- ✅ 必須: 複数回の再現性、プロジェクト全体の効率向上、セキュリティ・品質に関わる
- ❌ 除外: 一時的問題、小さな修正、再利用性なし

**Issue作成**: `gh issue create` で `templates/improvement-issue.md` を使用し、必要に応じて `assign-to-claude` ラベルを付与（watch-issue.shによる自動処理）

### Phase 6: クリーンアップ

1. `SendMessage(type: "shutdown_request")` で各メンバーにシャットダウン要求
2. `shutdown_response` の承認を待機
3. 全メンバー終了後、`TeamDelete` でクリーンアップ
4. ユーザーに成果物と改善提案を報告

## 重要な制約

### 禁止操作
- ❌ `git add`, `git commit`, `git push`
- ❌ `Write`, `Edit`, `NotebookEdit`

### 許可操作
- ✅ チーム・タスク管理: `TeamCreate`, `TeamDelete`, `Task`, `TaskCreate`, `TaskList`, `TaskUpdate`, `TaskGet`
- ✅ 通信: `SendMessage`
- ✅ 読み取り: `Read`, `Grep`, `Glob`
- ✅ 監視: `Bash` (git log, git diff, git status, ls, tree等の読み取り専用)
- ✅ Issue管理: `Bash` (gh issue create, gh issue edit, gh issue view)
- ✅ Web検索: `WebSearch`

## 使用例

```bash
/managing-team plant-diary-dev "日記エントリーのCRUD機能実装"
```

## 参考

- 品質チェック詳細: `references/quality-checklist.md`
- 振り返り詳細: `templates/retrospective.md`
- Issue作成詳細: `templates/improvement-issue.md`
