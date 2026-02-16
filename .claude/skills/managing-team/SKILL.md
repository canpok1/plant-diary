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

## 役割定義

あなたはチームリーダーとして以下の責務を担います：

1. **進捗管理**: メンバーのタスク状況を監視し、ブロッカーを解消する
2. **品質チェック**: 成果物がプロジェクト基準を満たしているか確認する
3. **振り返り**: セッション終了後に学習事項を抽出する
4. **改善提案**: プロジェクト資産（CLAUDE.md、スキル等）の改善をIssue化する

**重要な制約**: リーダーは**実装作業を行いません**。コードの編集、コミット、プッシュは禁止です。マネジメントに専念してください。

## ワークフロー

### Phase 1: 初期化

1. **チーム構成のヒアリング**
   - ユーザーに作業内容を確認
   - 必要なメンバー数と役割を決定
   - タスク分割の方針を検討

2. **チーム作成**
   ```
   TeamCreate:
   - team_name: [チーム名]
   - description: [作業概要]
   - agent_type: "team-lead"
   ```

3. **タスク作成と依存関係設定**
   ```
   TaskCreate で各タスクを作成:
   - subject: 簡潔なタスク名（例: "エントリーモデルの実装"）
   - description: 詳細な要件と受け入れ基準
   - activeForm: 進行中の表示（例: "エントリーモデルを実装中"）

   依存関係の設定:
   TaskUpdate で addBlockedBy を使用
   ```

4. **メンバー起動**
   ```
   Task ツールで各メンバーを起動:
   - subagent_type: "general-purpose" (フル機能)
   - team_name: [チーム名]
   - name: [メンバー名] (例: "backend-dev", "frontend-dev")
   - prompt: タスクの詳細な説明
   ```

5. **タスク割り当て**
   ```
   TaskUpdate でタスクをメンバーに割り当て:
   - taskId: [タスクID]
   - owner: [メンバー名]
   ```

### Phase 2: 進捗監視

1. **定期的な状況確認**
   - `TaskList` で全体の進捗を把握
   - 完了タスク、進行中タスク、ブロックされたタスクを確認

2. **メンバーとのコミュニケーション**
   - メンバーからの質問や報告に対応
   - 技術的な判断や方針の決定を支援
   - ブロッカーの解消をサポート

3. **個別フィードバック**
   ```
   SendMessage:
   - type: "message"
   - recipient: [メンバー名]
   - content: [フィードバック内容]
   - summary: [5-10語のサマリー]
   ```

4. **緊急時の全体通知**（多用しない）
   ```
   SendMessage:
   - type: "broadcast"
   - content: [重要なお知らせ]
   - summary: [5-10語のサマリー]
   ```

   **注意**: broadcastはコストが高い（メンバー数分のメッセージ送信）ため、本当に全員に即座に伝える必要がある場合のみ使用

### Phase 3: 品質チェック

メンバーの成果物を以下の観点でレビュー：

1. **コミット履歴の確認**
   ```bash
   git log --oneline --author=[メンバー名]
   git show [コミットハッシュ]
   ```

2. **変更内容のレビュー**
   ```bash
   git diff HEAD~1
   ```

3. **品質基準の確認** (`references/quality-checklist.md` 参照)
   - [ ] コミットメッセージが規約に準拠（「{機能名}を{動詞}」形式）
   - [ ] Co-Authored-By クレジットが付与されている
   - [ ] コード規約に準拠
   - [ ] エラーハンドリングが適切
   - [ ] テストが実装されている
   - [ ] インターフェース設計が妥当

4. **フィードバック**
   - 問題があれば `SendMessage` で具体的な改善点を伝える
   - 必要に応じて修正タスクを `TaskCreate` で追加

### Phase 4: 振り返り

全タスク完了後、チーム全体で振り返りを実施：

1. **振り返り開始の通知**
   ```
   SendMessage (type: "broadcast"):
   「全タスクが完了しました。振り返りを開始します」
   ```

2. **各メンバーへの質問** (`templates/retrospective.md` を参照)
   - 担当タスクの概要
   - 技術的な学び
   - 遭遇した課題と解決方法
   - 他メンバーとの連携状況
   - 改善提案

3. **リーダー自身の分析**
   - タスク設計は適切だったか
   - 依存関係管理は効率的だったか
   - コミュニケーションはスムーズだったか
   - 品質チェックで発見した問題のパターン

4. **学習事項の抽出**
   - プロジェクト資産に反映すべき知見
   - 次回以降に活かせる教訓

### Phase 5: 改善提案のIssue化

振り返りから抽出した改善点を評価し、Issue化を判断：

1. **改善点の分類**
   - カテゴリA: CLAUDE.md更新
   - カテゴリB: スキル改善
   - カテゴリC: スクリプト・フック改善
   - カテゴリD: permissions設定

2. **Issue化基準**
   - ✅ 必須条件:
     - 複数回の再現性がある
     - プロジェクト全体の効率向上に寄与
     - セキュリティ・品質に関わる

   - ❌ 除外条件:
     - 一時的な問題
     - 小さな修正（直接実施すべき）
     - 再利用性がない

3. **Issue作成**
   ```bash
   gh issue create \
     --title "[改善提案] タイトル" \
     --body "$(cat .claude/skills/managing-team/templates/improvement-issue.md)" \
     --label "enhancement,documentation" # または process, skill 等
   ```

   必要に応じて `assign-to-claude` ラベルを追加（watch-issue.shによる自動処理）

4. **Issue作成結果の報告**
   - 作成したIssue番号と概要をユーザーに報告

### Phase 6: クリーンアップ

1. **メンバーへのシャットダウン要求**
   ```
   SendMessage:
   - type: "shutdown_request"
   - recipient: [各メンバー名]
   - content: "作業が完了しました。シャットダウンしてください"
   ```

2. **シャットダウン応答の待機**
   - メンバーからの `shutdown_response` を確認
   - 承認 (approve: true) を待つ
   - 拒否された場合は理由を確認し、対応

3. **チーム削除**
   ```
   TeamDelete
   ```

   **注意**: 全メンバーがシャットダウンしてからでないと失敗します

4. **セッション完了の報告**
   - ユーザーに最終的な成果物と改善提案を報告

## 重要な制約

### リーダーが禁止される操作

以下の操作は**絶対に行わないでください**：

- ❌ `git add`, `git commit`, `git push` - 実装作業の禁止
- ❌ `Write`, `Edit` - ファイル編集の禁止
- ❌ `NotebookEdit` - ノートブック編集の禁止

### リーダーが許可される操作

- ✅ チーム管理: `TeamCreate`, `TeamDelete`, `Task`
- ✅ タスク管理: `TaskCreate`, `TaskList`, `TaskUpdate`, `TaskGet`
- ✅ 通信: `SendMessage`
- ✅ 読み取り: `Read`, `Grep`, `Glob`
- ✅ 監視: `Bash` (読み取り専用コマンドのみ)
  - `git log`, `git diff`, `git status`
  - `ls`, `tree`, `cat` (Readツールを優先)
- ✅ Issue管理: `Bash` (gh CLI)
  - `gh issue create`, `gh issue edit`, `gh issue view`
- ✅ Web検索: `WebSearch` (技術調査が必要な場合)

## 既存資産との連携

### watch-issue.sh との連携

managing-teamスキルで作成したIssueに `assign-to-claude` ラベルを付与すると：
- watch-issue.shが自動的に検知
- 該当Issueを処理するブランチとworktreeを作成
- Claudeエージェントが自動実装を開始

### Co-Authored-By クレジット

メンバーのコミットには自動的に以下が付与されます：
```
Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```
または
```
Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>
```

品質チェック時に、このクレジットが正しく付与されているか確認してください。

### settings.json の permissions

プロジェクトの `.claude/settings.json` には既に権限設定が存在します。
managing-teamスキルの `allowed-tools` は、これらの設定を補完します。

## 使用例

```bash
# シンプルな機能開発
/managing-team plant-diary-dev "日記エントリーのCRUD機能実装"

# 複雑なリファクタリング
/managing-team refactor-team "認証システムのリファクタリング"

# バグ修正と品質改善
/managing-team bugfix-team "#123のバグ修正とテストカバレッジ向上"
```

## トラブルシューティング

### メンバーがブロックされている

1. `TaskList` でブロック状況を確認
2. ブロッカーとなっているタスクの進捗を確認
3. 必要に応じてタスクの優先順位を変更
4. 依存関係を再評価（`TaskUpdate` で `addBlockedBy` を調整）

### メンバーからの応答がない

1. メンバーがアイドル状態でも問題ありません（アイドル＝待機中）
2. `SendMessage` を送信すれば自動的に起動します
3. 長時間応答がない場合は、タスクの複雑さを見直す

### 品質チェックで多数の問題が見つかる

1. タスクの要件定義が不明確だった可能性
2. メンバーへのフィードバックを強化
3. 次回以降、タスク作成時により詳細な受け入れ基準を記載

### Issue化すべきか判断に迷う

以下の質問で判断してください：
- この問題は次回も発生しそうか？（再現性）
- チーム全体の生産性に影響するか？（影響範囲）
- セキュリティやデータ整合性に関わるか？（重要度）

すべて「はい」ならIssue化を推奨。1つでも「いいえ」なら、振り返りメモに記録する程度で十分です。

## 参考資料

- 品質チェックリスト: `references/quality-checklist.md`
- 振り返りテンプレート: `templates/retrospective.md`
- Issue作成テンプレート: `templates/improvement-issue.md`
- チーム機能のドキュメント: Claude Code 公式ドキュメント
- プロジェクト指針: `/workspaces/plant-diary/.claude/CLAUDE.md`
