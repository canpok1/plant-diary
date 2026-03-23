---
name: fix-pr-local
description: PR のCI待機・レビュー対応・マージを行うスキル。auto-merge を中心とした設計で、CI完了→マージ確認→ブロッカー対応のループを実行する。solve-issue のステップ7を分離したもの。
context: fork
agent: general-purpose
allowed-tools: Skill, Bash, Read, Grep, Glob, Write, Edit
disable-model-invocation: false
user-invocable: true
argument-hint: "<PR番号>"
---

PR $ARGUMENTS に対して、CI待機・レビュー対応・マージを行います。

このチェックリストをコピーし、進行状況の追跡に使用してください：

- [ ] ステップ1：fix-pr.sh を実行
- [ ] ステップ2：終了コードに基づいて対応

## フロー

以下のループを繰り返す。ループ先頭に戻る場合は必ず `sleep 10` を挟んでからステップ1を再実行すること（push直後にCIが起動前の状態で `gh pr checks --watch` が即完了してしまうのを防ぐため）。

### ステップ1: fix-pr.sh を実行

```bash
./.claude/skills/fix-pr-local/scripts/fix-pr.sh <PR番号>
```

### ステップ2: 終了コードに基づいて対応

| 終了コード | 意味 | アクション |
|---|---|---|
| `0` | マージ完了 | 完了。ユーザーに結果を報告 |
| `1` | スクリプトエラー | ステップ3（エラー対応）へ進む |
| `10` | CI失敗 | ステップ4（CI失敗対応）へ進む |
| `20` | 未解決スレッド | ステップ5（未解決スレッド対応）へ進む |
| `30` | 承認待ち | ステップ6（承認待ち対応）へ進む |

### ステップ3: エラー対応（exit 1の場合）

stderrを分析する:

- **"conflict" を含む場合**: コンフリクト解消 → コミット・プッシュ → `sleep 10` → ステップ1へ戻る
  - コミットメッセージ例: `fix: コンフリクトを解消`
- **それ以外**: エラー内容をユーザーに通知して中断する

### ステップ4: CI失敗対応（exit 10の場合）

1. `gh run list --pr <PR番号>` で最新の失敗 run を取得する
2. `gh run view <RUN_ID> --log-failed` でログを取得・分析する
3. コードを修正してコミット・プッシュする
4. `sleep 10` してステップ1へ戻る

### ステップ5: 未解決スレッド対応（exit 20の場合）

1. GraphQL で `isResolved == false` のスレッドを全取得する（**CodeRabbit 限定しない**）

```bash
gh api graphql -f query='
  query($owner: String!, $repo: String!, $pr: Int!) {
    repository(owner: $owner, name: $repo) {
      pullRequest(number: $pr) {
        reviewThreads(first: 100) {
          nodes {
            isResolved
            path
            line
            comments(first: 1) {
              nodes {
                fullDatabaseId
                author { login }
                body
              }
            }
          }
        }
      }
    }
  }
' -f owner=<OWNER> -f repo=<REPO> -F pr=<PR番号> \
  --jq '.data.repository.pullRequest.reviewThreads.nodes
        | map(select(.isResolved == false))
        | map({id: .comments.nodes[0].fullDatabaseId, author: .comments.nodes[0].author.login, path: .path, line: .line, body: .comments.nodes[0].body})'
```

2. 各スレッドを分析して対応する:
   - 修正が必要 → コードを修正
   - 返信のみで対応可 → コメントで返信

3. コードを修正した場合はコミット・プッシュする

4. 各スレッドに返信する:

```bash
# レビュワーへのメンション付きで返信
gh api repos/<OWNER>/<REPO>/pulls/comments/<COMMENT_ID>/replies \
  -X POST -f body="@<author> 修正しました。ご確認ください。

---
🤖 Generated with [Claude Code](https://claude.ai/claude-code)"
```

5. `sleep 10` してステップ1へ戻る

### ステップ6: 承認待ち対応（exit 30の場合）

PRのコメントとレビューを取得して以下の順で判定する:

```bash
# コメント・レビューを取得
gh pr view --repo <REPO> <PR番号> --json comments,reviews
```

#### 判定順序

1. **CodeRabbit の rate limit コメントがある**

   rate limit コメントが最新の CodeRabbit コメント/レビューに含まれているか確認する。
   含まれている場合:
   - コメント本文から待機時間を自然言語として解釈し、秒数に換算する
   - 残り待機秒数 = `(comment_created_at の UNIX 時刻 + 待機秒数) - 現在の UNIX 時刻`
   - 残り待機秒数が 0 以下なら `0` を渡す
   - `handle-coderabbit-rate-limit.sh` を実行する:
     ```bash
     ./.claude/skills/fix-pr-local/scripts/handle-coderabbit-rate-limit.sh <PR番号> <残り待機秒数>
     ```
   - 終了コードに応じて対応:
     - exit 0（正常完了）: `sleep 10` してステップ1へ戻る
     - exit 1（タイムアウト）: ユーザーにタイムアウトした旨を報告し、そのままステップ1へ戻る

2. **対応すべきコメントがある（rate limit以外）**

   ステップ5と同様にコメントに対応する。

3. **対応不要なのに未承認**

   各レビュアー宛に approve 依頼コメントを投稿し、`sleep 10` してステップ1へ戻る:

   ```bash
   gh pr comment --repo <REPO> <PR番号> --body "@<レビュアー名> CIが通過し、未解決のコメントもありません。レビューとapproveをお願いします。

   ---
   🤖 Generated with [Claude Code](https://claude.ai/claude-code)"
   ```

## 注意事項

- コンフリクト解消時はコミットメッセージにIssue番号を含めること
- マージ完了後のクリーンアップ（ブランチ削除等）はこのスキルの責務外（GitHub側の自動削除設定に任せる）
- ループ先頭（ステップ1の再実行）の際は必ず `sleep 10` を挟むこと（push直後のCI起動前に `gh pr checks --watch` が即完了するタイミング問題を回避するため）
