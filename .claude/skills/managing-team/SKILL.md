---
name: managing-team
description: 複数のClaudeエージェントで協力開発する際に使用。リーダーが実装から離れて進捗管理・チーム調整に専念し、チーム全体の生産性を向上させる。
context: fork
agent: general-purpose
allowed-tools: TeamCreate, TeamDelete, TaskCreate, TaskList, TaskUpdate, TaskGet, TaskOutput, TaskStop, SendMessage, Bash, Read, Grep, Glob, WebSearch, Skill
disable-model-invocation: false
user-invocable: true
argument-hint: "[team-name] [task-description]"
---

# Managing Team スキル

## 使用すべき場合

以下の条件を満たす場合、agent teamの使用を検討してください：

- **複数の独立したタスクを並行で実行できる場合**
  - 各タスクが明確に分割でき、相互依存が少ない
  - 並行実行により開発速度を大幅に向上できる

- **各タスクが明確に分割でき、役割分担が可能な場合**
  - 機能やレイヤーごとに明確な責任範囲を定義できる
  - 各メンバーが独立して作業を完結できる

- **機能のまとまりごとにコミットを作成したい場合**
  - 変更履歴を機能単位で追跡しやすくしたい
  - レビューやロールバックを容易にしたい

## 役割

リーダーとして以下の責務を担います：

1. **進捗管理**: タスク状況の監視とブロッカー解消
2. **チーム調整**: メンバー間の調整とコミュニケーション促進

**重要**: リーダーは実装作業（コード編集、コミット、プッシュ）を行いません。マネジメントに専念してください。

### チーム構成の例

- **リーダー（team-lead）**
  - 全体設計のレビュー
  - 各メンバーの成果物の統合確認
  - タスクの割り当てと進捗管理
  - 最終的なPR作成とマージ

- **メンバー（例）**
  - **project-setup**: プロジェクト構造とビルド設定の実装
  - **data-layer**: データモデルとストレージレイヤーの実装
  - **ai-layer**: AI連携インターフェースの実装

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

**重要**: 振り返りは**チーム全員で実施**してください。リーダーだけで振り返らないこと。

1. `SendMessage(type: "broadcast")` で振り返り開始を全員に通知
2. 各メンバーに質問を送信（`SendMessage(type: "message")`）
   - 担当タスクの概要と学び
   - 遭遇した課題と解決方法
   - チーム連携の良かった点と改善点
   - プロジェクト資産への改善提案
3. 全員からの回答を収集
4. リーダー自身も分析（タスク設計、依存関係管理、コミュニケーション）
5. 収集した情報をもとに `base-tools:retro` スキルを呼び出して改善提案をIssue化

### Phase 5: クリーンアップ

1. `SendMessage(type: "shutdown_request")` で各メンバーにシャットダウン要求
2. `shutdown_response` の承認を待機
3. 全メンバー終了後、`TeamDelete` でクリーンアップ
4. ユーザーに成果物を報告

## コミット方針

メンバーへの指示に含めるコミット方針：

- **各メンバーが独立した機能ごとにコミット**
  - 1つの機能 = 1つのコミット
  - 責任範囲を明確にし、変更履歴を追跡しやすくする

- **コミットメッセージの形式**
  - 基本形式: `{機能名}を{動詞}`
  - 例: "プロジェクト構造を作成"、"データモデルを実装"

- **Issue番号の含め方**
  - 関連するIssueがある場合: `#{Issue番号} {コミットメッセージ}`
  - 例: `#11 プロジェクト構造を作成`

## worktreeとの組み合わせ使用時の注意

- **agent teamを使用する場合、worktreeの使用は推奨しません**
  - チームメンバーは元のリポジトリで作業するため、リーダーがworktreeで作業すると作業ディレクトリが不一致になります
  - ファイルを手動でコピーする必要があり、非効率です

- **推奨される方法**:
  1. **agent teamを使用する場合**: 元のリポジトリのブランチで作業する（worktreeを使用しない）
  2. **worktreeを使用する場合**: agent teamを使用せず、単独で作業する

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

### 実装例（Issue #11より）

```text
チーム構成:
- team-lead（リーダー）
- project-setup（プロジェクト構造担当）
- data-layer（データレイヤー担当）
- ai-layer（AIレイヤー担当）

タスク構成:
1. プロジェクト構造を作成（project-setup）
2. データモデルを実装（data-layer）
3. AI連携インターフェースを定義（ai-layer）
4. 統合確認とドキュメント作成（team-lead） ← タスク1-3に依存

結果:
- 3つの独立した機能を並行開発
- 各機能ごとに意味のあるコミットを作成
- 効率的にタスクを完了
```

## 連携スキル

- **quality-check**: 成果物の品質チェック
- **base-tools:retro**: セッション振り返りと改善提案のIssue化
