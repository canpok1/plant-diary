---
name: quality-check
description: コミットやPRの品質チェックを実施。コミットメッセージ、コード規約、テスト、エラーハンドリング等の基準に照らして成果物をレビューし、フィードバックを提供する。
context: fork
agent: general-purpose
allowed-tools: Bash Read Grep Glob SendMessage
disable-model-invocation: false
user-invocable: true
argument-hint: "[target-commit-or-pr]"
---

# Quality Check スキル

## 役割

成果物の品質を統一された基準でチェックし、具体的な改善点をフィードバックします。

## チェック対象

`references/quality-checklist.md` の基準に基づいて以下を確認：

### 1. コミットメッセージ
- 規約準拠（「{機能名}を{動詞}」形式）
- Co-Authored-By クレジットの存在
- 簡潔性と明確性

### 2. コード品質
- コード規約準拠（Go: golangci-lint等）
- エラーハンドリングの適切性
- テストの有無と網羅性

### 3. 設計
- インターフェース設計の妥当性
- 単一責任の原則
- 依存関係管理

### 4. セキュリティ
- 機密情報のハードコードなし
- 入力検証の適切性
- パストラバーサル対策

### 5. ドキュメント
- 公開APIへのコメント
- godoc形式の準拠
- パッケージコメントの存在

## チェック手順

1. **コミット履歴の確認**
   ```bash
   git log --oneline --author=[対象]
   git log --pretty=full  # Co-Authored-By確認
   ```

2. **変更内容の確認**
   ```bash
   git diff [対象範囲]
   git show [コミットハッシュ]
   ```

3. **品質基準の照合**
   `references/quality-checklist.md` の各項目を確認

4. **フィードバック作成**
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

# 特定メンバーのコミットをチェック
/quality-check --author=backend-dev
```

## 参考

詳細な品質基準: `references/quality-checklist.md`
