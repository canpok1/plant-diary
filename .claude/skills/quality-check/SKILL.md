---
name: quality-check
description: コードの設計・実装パターンを品質チェック。インターフェース設計、エラーハンドリング、テスト、セキュリティ等の基準に照らしてレビューし、フィードバックを提供する。
context: fork
agent: general-purpose
allowed-tools: Bash Read Grep Glob SendMessage
disable-model-invocation: false
user-invocable: true
argument-hint: "[target-commit-or-pr]"
---

# Quality Check スキル

## 役割

成果物の設計・実装パターンを統一された基準でチェックし、具体的な改善点をフィードバックします。

**前提**: linterでチェック可能な項目（gofmt, golangci-lint等）は自動チェックに任せます。このスキルは人間の判断が必要な項目に集中します。

**注意**: コミットメッセージは品質チェックの対象外です（開発者の責任範囲）。

## チェック手順

1. **変更内容の確認**
   ```bash
   git diff [対象範囲]
   git show [コミットハッシュ]
   ```

2. **linterチェック（自動）**
   ```bash
   golangci-lint run
   ```

3. **テスト実行**
   ```bash
   go test ./...
   go test -cover ./...
   ```

4. **品質基準の照合**
   `references/quality-checklist.md` の各項目を確認

5. **フィードバック作成**
   - 問題があれば具体的な改善点を記載
   - 良い点も積極的にフィードバック
   - 必要に応じて修正例を提示

## フィードバック方法

### チームメンバーへ
`SendMessage(type: "message")` で個別にフィードバック

### ユーザーへ
チェック結果をまとめて報告：
- ✅ 合格項目
- ❌ 改善が必要な項目
- 💡 推奨改善（任意）

## 使用例

```bash
# 最新コミットをチェック
/quality-check HEAD

# 特定のコミットをチェック
/quality-check abc123

# PRをチェック
/quality-check #123
```

## 参考

- `references/quality-checklist.md`: 品質チェックリスト（詳細な項目）
- `docs/conventions/go-style-guide.md`: Go言語スタイルガイド（設計パターン）
