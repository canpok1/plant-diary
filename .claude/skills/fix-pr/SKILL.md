---
name: fix-pr
description: PR のCI待機・レビュー対応・マージを行うスキル。solve-issue のステップ7を分離したもの。
context: fork
agent: general-purpose
allowed-tools: Skill, Bash, Read, Grep, Glob, Write, Edit
disable-model-invocation: false
user-invocable: true
argument-hint: "<PR番号>"
---

PR $ARGUMENTS に対して、CI待機・レビュー対応・マージを行います。

このチェックリストをコピーし、進行状況の追跡に使用してください：

- [ ] ステップ1：wait-coderabbit.sh を実行
- [ ] ステップ2：fix-pr.sh を実行
- [ ] ステップ3：終了コードに基づいて対応
  - [ ] ステップ3a：CodeRabbitのapprove漏れチェック（exit 2の場合）
  - [ ] ステップ3b：CodeRabbitの指摘を修正（exit 2の場合）

## フロー

以下のフローを繰り返す。フロー先頭に戻る場合は必ずステップ1から再開する。

### ステップ1: wait-coderabbit.sh を実行

```bash
./.claude/skills/fix-pr/scripts/wait-coderabbit.sh <PR番号> [既知のレビュー数]
```

- 第2引数（既知のレビュー数）: 省略可。この数を超えるレビューが来るまで待機する（デフォルト0）
  - 再レビュー依頼後など、すでにレビューが存在する状態で新しいレビューを待つ場合に指定する

| 終了コード | アクション |
|---|---|
| `0` | ステップ2へ進む |
| `1` | AIがエラー内容を分析し対応を試行。解決不可ならユーザーに通知して中断 |
| `2` | **rate limit検出**。stdoutのJSONを読み取り、`comment_body`から待機時間を解釈し、`comment_created_at`と現在時刻から残り待機秒数を算出して待機。待機後 `@coderabbitai review` をコメントし、ステップ1先頭に戻る（既知レビュー数を引き継ぐこと） |

#### exit 2 時のAIによる残り待機時間の算出手順

1. `comment_body` に記載された待機時間を自然言語として解釈し、秒数に換算する
2. 残り待機秒数 = `(comment_created_at の UNIX 時刻 + 待機秒数) - 現在の UNIX 時刻`
3. 残り待機秒数が 0 以下なら即座に進む、正なら `sleep <秒数>` する

### ステップ2: fix-pr.sh を実行

```bash
./.claude/skills/fix-pr/scripts/fix-pr.sh <PR番号>
```

### ステップ3: 終了コードに基づいて対応

| 終了コード | 意味 | アクション |
|---|---|---|
| `0` | マージ完了 | 完了。ユーザーに結果を報告 |
| `1` | コンフリクト要解消 | AIがコンフリクトを解消 → コミット・プッシュ → フロー先頭（ステップ1）に戻る |
| `2` | 未解決レビュー/CHANGES_REQUESTEDが原因 | ステップ3aへ進む |
| `3` | その他エラー | AIがstderrのエラー内容を分析・対応を試行 → 解決ならフロー先頭（ステップ1）に戻る → 解決不可ならユーザーに通知して中断 |

### ステップ3a: CodeRabbitのapprove漏れチェック（exit 2の場合）

exit 2の場合、レビューコメント対応の**前に**、CodeRabbitがレビュー完了済みなのにapproveし忘れていないかをチェックする。

#### 情報収集

以下のコマンドでCodeRabbitのレビュー状態を取得する:

```bash
# CodeRabbitの最新レビュー状態を取得
gh pr view --repo <REPO> <PR番号> --json reviews --jq '[.reviews[] | select(.author.login=="coderabbitai")] | last'

# CodeRabbitのレビューコメント（サマリー）を取得
gh pr view --repo <REPO> <PR番号> --json comments --jq '[.comments[] | select(.author.login=="coderabbitai")] | last | .body'

# CodeRabbitのcommit status checkを取得
gh pr view --repo <REPO> <PR番号> --json statusCheckRollup --jq '[.statusCheckRollup[] | select((.context // .name) | test("coderabbit"; "i"))]'

# 全CIチェックの状態を取得
gh pr view --repo <REPO> <PR番号> --json statusCheckRollup --jq '[.statusCheckRollup[] | {name: (.context // .name), state: (.state // .status // .conclusion)}]'
```

#### AI判断

取得した情報をもとに、以下の基準で判断する:

- **approve漏れと判断する条件**（以下のいずれかのケースに該当する場合）:
  - **ケース1: formal reviewが存在するがAPPROVEDでない場合**（以下をすべて満たす）:
    - CodeRabbitのレビューが存在するが、状態が `APPROVED` ではない
    - レビューコメントの内容から、指摘事項がない（または全て解決済み）と読み取れる
    - 実質的にレビュー完了しているにもかかわらず、approveアクションだけが行われていない
  - **ケース2: formal reviewが存在しないがcommit statusがSUCCESSの場合**（以下をすべて満たす）:
    - CodeRabbitのレビューが存在しない（`.reviews[]` が空）
    - CodeRabbitのcommit status checkが `SUCCESS`
    - CIが全て通過している

- **approve漏れではないと判断する条件**（以下のいずれかに該当する場合）:
  - 未解決の指摘事項が残っている
  - レビュー状態が `CHANGES_REQUESTED` で、実際に対応すべき変更要求がある
  - レビューがまだ進行中である

#### アクション

- **approve漏れの場合**: PRに `@coderabbitai approve` コメントを投稿 → フロー先頭（ステップ1）に戻る
- **approve漏れではない場合**: ステップ3bへ進む

### ステップ3b: CodeRabbitの指摘を修正（exit 2の場合）

CodeRabbitの指摘内容を取得・分析し、修正を実装する。

#### 1. 指摘内容の取得

```bash
# CodeRabbitのレビューコメント（インラインコメント含む）を取得
gh pr view --repo <REPO> <PR番号> --json reviews,comments --jq '
  {
    reviews: [.reviews[] | select(.author.login=="coderabbitai")],
    comments: [.comments[] | select(.author.login=="coderabbitai")]
  }
'

# 未解決のインラインレビューコメント（差分上のコメント）をGraphQLで取得
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
        | map(select(.isResolved == false
                     and (.comments.nodes[0].author.login == "coderabbitai")))
        | map({id: .comments.nodes[0].fullDatabaseId, path: .path, line: .line, body: .comments.nodes[0].body})'
```
#### 2. 指摘内容の修正

取得したレビューコメントを分析し、AIが判断して修正を実装する。

- ファイルパスと行番号からコードを特定し、問題を修正する
- コードの品質・スタイル指摘: 実際にコードを修正する
- ロジックの問題: 修正内容を判断して実装する
- ドキュメント・コメントの指摘: テキストを修正する

#### 3. コミット・プッシュ

プッシュ**前**に現在のCodeRabbitレビュー数を記録する（次のステップ1で使用するため）:

```bash
# プッシュ前にレビュー数を取得（1回のAPI呼び出しで取得）
KNOWN_TOTAL=$(gh pr view --repo <REPO> <PR番号> --json comments,reviews \
    --jq '([.comments[] | select(.author.login=="coderabbitai")] | length) + ([.reviews[] | select(.author.login=="coderabbitai")] | length)')
```

その後コミット・プッシュする:

```bash
git add <修正したファイルパス...>
git commit -m "fix: CodeRabbitの指摘を修正"
git push origin HEAD
```

#### 4. 返信

修正をプッシュ後、上記（手順1）で取得した各レビューコメントのスレッドに返信する。

**スレッド返信のルール**:
- レビューコメントへの返信は、**PRコメントではなくスレッドに対して行う**こと
  - PRコメント（誤った方法）: `gh pr comment --repo <REPO> <PR番号> --body "..."`
  - スレッド返信（正しい方法）: `gh api repos/<OWNER>/<REPO>/pulls/comments/<COMMENT_ID>/replies -X POST -f body="..."`
- `<COMMENT_ID>` は上記（手順1）のGraphQL取得結果の `id` フィールドを使用する
- 返信には必ず**レビュワーへのメンション**を付与すること（例: `@coderabbitai`）
- 複数のコメントがある場合は、各コメントのIDに対してそれぞれ返信を行う

```bash
# 各コメントに対してループで実行する
gh api repos/<OWNER>/<REPO>/pulls/comments/<COMMENT_ID>/replies \
  -X POST -f body="@coderabbitai 修正しました。ご確認ください。

---
🤖 Generated with [Claude Code](https://claude.ai/claude-code)"
```

→ フロー先頭（ステップ1）に戻る（ステップ1の `wait-coderabbit.sh` には、上記で記録した `KNOWN_TOTAL` を第2引数として渡すこと）

## 注意事項

- コンフリクト解消時はコミットメッセージにIssue番号を含めること
- マージ完了後のクリーンアップ（ブランチ削除等）はこのスキルの責務外（GitHub側の自動削除設定に任せる）
