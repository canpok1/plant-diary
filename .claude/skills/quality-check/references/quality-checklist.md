# 品質チェックリスト

このチェックリストは、managing-teamスキルでメンバーの成果物をレビューする際に使用します。

## 1. コミットメッセージ規約

### 基本形式

```
{機能名}を{動詞}

詳細説明（オプション）

Co-Authored-By: Claude [Model] <noreply@anthropic.com>
```

### チェック項目

- [ ] **簡潔性**: 1行目は50文字以内
- [ ] **動詞の適切性**:
  - `追加` - 新機能の追加
  - `実装` - 機能の実装
  - `修正` - バグ修正
  - `更新` - 既存機能の改善
  - `削除` - 不要なコードの削除
  - `リファクタリング` - 構造改善
  - `ドキュメント化` - ドキュメント追加・更新
- [ ] **主語の明確性**: 何を変更したかが明確
- [ ] **英語の場合**: 命令形（Add, Fix, Update等）

### 良い例

```
日記エントリーのCRUD機能を実装

- CreateEntry, ReadEntry, UpdateEntry, DeleteEntry を追加
- インメモリストレージを使用
- エラーハンドリングを実装

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

### 悪い例

```
いろいろ修正  # 何を修正したか不明
Update code  # 抽象的すぎる
日記機能追加しました  # 「を」がない、敬語不要
```

## 2. Co-Authored-By クレジット

### チェック項目

- [ ] **存在確認**: すべてのコミットに Co-Authored-By が含まれている
- [ ] **正しいモデル名**:
  - `Claude Sonnet 4.5` (claude-sonnet-4-5-20250929)
  - `Claude Opus 4.6` (claude-opus-4-6)
  - `Claude Haiku 4.5` (claude-haiku-4-5-20251001)
- [ ] **メールアドレス**: `<noreply@anthropic.com>` を使用

### 確認コマンド

```bash
# 最新コミットのクレジット確認
git log -1 --pretty=full

# 特定作者のコミット確認
git log --author="backend-dev" --pretty=full

# Co-Authored-Byがないコミットを検索
git log --all --grep="Co-Authored-By" --invert-grep --oneline
```

## 3. Go言語コード規約（plant-diaryプロジェクト）

### 一般的な規約

- [ ] **gofmt**: コードがフォーマットされている
- [ ] **golangci-lint**: リンターエラーがない
- [ ] **命名規則**:
  - パッケージ名: 小文字、単語（例: `diary`, `storage`）
  - エクスポート: 大文字開始（例: `CreateEntry`）
  - 非エクスポート: 小文字開始（例: `validateEntry`）
- [ ] **エラー変数**: `Err` プレフィックス（例: `ErrNotFound`）

### インターフェース設計

- [ ] **単一責任**: 1つのインターフェースは1つの責務
- [ ] **小さいインターフェース**: メソッド数は少なく（理想は1-3）
- [ ] **命名**: `-er` サフィックス（例: `Reader`, `Writer`, `Storage`）
- [ ] **context.Context**: 最初の引数に配置

### 例（良い設計）

```go
// EntryStorage は日記エントリーの永続化を担当
type EntryStorage interface {
    CreateEntry(ctx context.Context, entry *Entry) error
    ReadEntry(ctx context.Context, id string) (*Entry, error)
    UpdateEntry(ctx context.Context, entry *Entry) error
    DeleteEntry(ctx context.Context, id string) error
}
```

## 4. エラーハンドリング

### チェック項目

- [ ] **エラーチェック**: すべてのエラーを適切に処理
- [ ] **エラーラッピング**: `fmt.Errorf` + `%w` を使用
- [ ] **カスタムエラー**: 必要に応じて定義
- [ ] **エラーメッセージ**: 小文字開始、簡潔、コンテキスト情報を含む

### 良い例

```go
if err := storage.CreateEntry(ctx, entry); err != nil {
    return fmt.Errorf("failed to create entry: %w", err)
}
```

### 悪い例

```go
storage.CreateEntry(ctx, entry)  // エラー無視
if err != nil {
    return err  // コンテキスト情報なし
}
if err != nil {
    return fmt.Errorf("Error: %s", err)  // ラッピングなし、大文字開始
}
```

## 5. テストの実装

### チェック項目

- [ ] **テストファイル**: `_test.go` サフィックス
- [ ] **テスト関数**: `Test` プレフィックス + 対象関数名
- [ ] **カバレッジ**: 主要な機能パスをカバー
- [ ] **テーブル駆動テスト**: 複数ケースを効率的にテスト
- [ ] **モック**: 外部依存をモック化

### テーブル駆動テストの例

```go
func TestCreateEntry(t *testing.T) {
    tests := []struct {
        name    string
        entry   *Entry
        wantErr bool
    }{
        {
            name:    "正常系",
            entry:   &Entry{Title: "Test", Content: "Content"},
            wantErr: false,
        },
        {
            name:    "空タイトル",
            entry:   &Entry{Title: "", Content: "Content"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := CreateEntry(context.Background(), tt.entry)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateEntry() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 確認コマンド

```bash
# テスト実行
go test ./...

# カバレッジ確認
go test -cover ./...

# 詳細なカバレッジレポート
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 6. ドキュメンテーション

### チェック項目

- [ ] **パッケージコメント**: `package` 文の前にパッケージ説明
- [ ] **公開API**: エクスポートされた型・関数にコメント
- [ ] **godoc形式**: コメントは対象名で始まる
- [ ] **例**: 複雑な機能には `Example` テストを追加

### 良い例

```go
// Package diary provides functionality for managing daily diary entries.
package diary

// Entry represents a single diary entry with metadata and content.
type Entry struct {
    ID        string
    Title     string
    Content   string
    CreatedAt time.Time
}

// CreateEntry creates a new diary entry in the storage.
// It returns an error if the entry is invalid or storage fails.
func CreateEntry(ctx context.Context, entry *Entry) error {
    // implementation
}
```

### 悪い例

```go
// diary package  # 対象名で始まっていない
package diary

type Entry struct {  // コメントなし
    ID string
}

// create entry  # 小文字開始、対象名なし
func CreateEntry(ctx context.Context, entry *Entry) error {
```

## 7. セキュリティ

### チェック項目

- [ ] **SQLインジェクション**: プリペアドステートメント使用
- [ ] **パストラバーサル**: ファイルパス検証
- [ ] **機密情報**: ハードコードされた認証情報なし
- [ ] **入力検証**: ユーザー入力を信頼しない
- [ ] **エラー情報**: 詳細すぎる情報を外部に漏らさない

### 確認例

```go
// 良い例: パストラバーサル対策
func readFile(userPath string) error {
    cleanPath := filepath.Clean(userPath)
    if !strings.HasPrefix(cleanPath, "/safe/directory/") {
        return errors.New("invalid path")
    }
    // read file
}

// 悪い例: 機密情報のハードコード
const apiKey = "sk_live_xxxxx"  // NG
```

## 8. パフォーマンス

### チェック項目

- [ ] **不要なアロケーション**: メモリ効率を意識
- [ ] **ゴルーチンリーク**: 適切な終了処理
- [ ] **context.Context**: タイムアウト・キャンセル処理
- [ ] **データベース**: N+1問題の回避

### 確認コマンド

```bash
# ベンチマーク実行
go test -bench=. -benchmem

# プロファイリング
go test -cpuprofile=cpu.prof -memprofile=mem.prof
go tool pprof cpu.prof
```

## 9. 依存関係管理

### チェック項目

- [ ] **go.mod**: 依存関係が適切に記録
- [ ] **go.sum**: チェックサムが更新されている
- [ ] **不要な依存**: 使用していない依存関係がない
- [ ] **バージョン固定**: 本番環境で使用する依存のバージョンを固定

### 確認コマンド

```bash
# 依存関係の整理
go mod tidy

# 使用されていない依存を確認
go mod why -m <module>

# 依存関係の更新確認
go list -u -m all
```

## 10. Git管理

### チェック項目

- [ ] **.gitignore**: ビルド成果物、機密情報を除外
- [ ] **コミット粒度**: 1コミット = 1つの論理的変更
- [ ] **ブランチ戦略**: feature/*, bugfix/* 等の命名規則
- [ ] **マージコミット**: 不要なマージコミットを避ける（rebase推奨）

### 除外すべきファイル例

```
# Go
*.exe
*.exe~
*.dll
*.so
*.dylib
*.test
*.out
coverage.txt
vendor/

# IDE
.idea/
.vscode/
*.swp
*.swo

# 機密情報
.env
*.key
*.pem
```

## レビュー手順のまとめ

1. **コミット履歴の確認**: `git log --oneline`
2. **コミットメッセージチェック**: `git log --pretty=full`
3. **変更内容のレビュー**: `git diff` または `git show`
4. **コード規約チェック**: `golangci-lint run`
5. **テスト実行**: `go test ./...`
6. **カバレッジ確認**: `go test -cover ./...`
7. **依存関係確認**: `go mod verify`

問題が見つかった場合は、`SendMessage` でメンバーに具体的なフィードバックを送り、必要に応じて修正タスクを作成してください。
