# Go Style Guide

このプロジェクトのGo言語コーディングスタイルガイドです。

## 前提

- **gofmt**: すべてのコードはgofmtでフォーマット済みであること
- **golangci-lint**: linterのエラーがないこと

このガイドでは、linterでチェックできない設計パターンや考え方を説明します。

## インターフェース設計

### 単一責任の原則

1つのインターフェースは1つの責務に集中させます。

```go
// 良い例: 責務が明確
type EntryStorage interface {
    CreateEntry(ctx context.Context, entry *Entry) error
    ReadEntry(ctx context.Context, id string) (*Entry, error)
    UpdateEntry(ctx context.Context, entry *Entry) error
    DeleteEntry(ctx context.Context, id string) error
}

type EntryValidator interface {
    Validate(entry *Entry) error
}

// 悪い例: 複数の責務が混在
type EntryManager interface {
    CreateEntry(ctx context.Context, entry *Entry) error
    ReadEntry(ctx context.Context, id string) (*Entry, error)
    Validate(entry *Entry) error
    SendNotification(entry *Entry) error  // 異なる責務
}
```

### 小さいインターフェース

インターフェースは小さく保ちます（理想は1-3メソッド）。

```go
// 良い例
type Reader interface {
    Read(ctx context.Context, id string) (*Entry, error)
}

type Writer interface {
    Write(ctx context.Context, entry *Entry) error
}

// 必要に応じて組み合わせる
type ReadWriter interface {
    Reader
    Writer
}
```

### 命名規則

インターフェースは `-er` サフィックスを使用します。

```go
type Storage interface { ... }  // Storer でも可
type Validator interface { ... }
type Notifier interface { ... }
```

### context.Context

context.Contextは常に最初の引数に配置します。

```go
// 良い例
func CreateEntry(ctx context.Context, entry *Entry) error

// 悪い例
func CreateEntry(entry *Entry, ctx context.Context) error
```

## エラーハンドリング

### エラーのラッピング

エラーにコンテキスト情報を追加するため、`fmt.Errorf` と `%w` を使用します。

```go
// 良い例
if err := storage.CreateEntry(ctx, entry); err != nil {
    return fmt.Errorf("failed to create entry: %w", err)
}

// 悪い例
if err != nil {
    return err  // コンテキスト情報なし
}
```

### エラーメッセージ

- 小文字で開始
- 簡潔かつ具体的
- 句読点不要（末尾のピリオド等）

```go
// 良い例
errors.New("entry not found")
fmt.Errorf("invalid entry ID: %s", id)

// 悪い例
errors.New("Entry not found.")  // 大文字開始、ピリオド付き
errors.New("error")  // 情報不足
```

### カスタムエラー

よく使うエラーは定義します。

```go
var (
    ErrNotFound     = errors.New("entry not found")
    ErrInvalidInput = errors.New("invalid input")
    ErrUnauthorized = errors.New("unauthorized")
)
```

## テスト戦略

### テーブル駆動テスト

複数のケースを効率的にテストします。

```go
func TestValidateEntry(t *testing.T) {
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
        {
            name:    "空コンテンツ",
            entry:   &Entry{Title: "Test", Content: ""},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEntry(tt.entry)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEntry() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### テストカバレッジ

- 主要な機能パスをカバーする
- エッジケースを含める
- エラーパスもテストする

```bash
# カバレッジ確認
go test -cover ./...

# 詳細レポート
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### モックの使用

外部依存はモック化します。

```go
type MockStorage struct {
    entries map[string]*Entry
}

func (m *MockStorage) ReadEntry(ctx context.Context, id string) (*Entry, error) {
    entry, ok := m.entries[id]
    if !ok {
        return nil, ErrNotFound
    }
    return entry, nil
}
```

## ドキュメンテーション

### 公開APIへのコメント

エクスポートされた型・関数には必ずコメントを付けます。

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

### Exampleテスト

複雑な機能にはExampleテストを追加します。

```go
func ExampleCreateEntry() {
    entry := &Entry{
        Title:   "My First Entry",
        Content: "Today was a great day!",
    }

    if err := CreateEntry(context.Background(), entry); err != nil {
        log.Fatal(err)
    }

    fmt.Println("Entry created successfully")
    // Output: Entry created successfully
}
```

## セキュリティ

### 入力検証

ユーザー入力は常に検証します。

```go
func ValidateEntry(entry *Entry) error {
    if entry == nil {
        return ErrInvalidInput
    }
    if strings.TrimSpace(entry.Title) == "" {
        return errors.New("title is required")
    }
    if len(entry.Title) > 200 {
        return errors.New("title too long")
    }
    return nil
}
```

### パストラバーサル対策

ファイルパスは必ず検証します。

```go
func readFile(userPath string) error {
    cleanPath := filepath.Clean(userPath)
    if !strings.HasPrefix(cleanPath, "/safe/directory/") {
        return errors.New("invalid path")
    }
    // ファイル読み取り処理
    return nil
}
```

### 機密情報の扱い

- ハードコード禁止
- 環境変数を使用
- `.env` をgitignoreに追加

```go
// 良い例
apiKey := os.Getenv("API_KEY")
if apiKey == "" {
    return errors.New("API_KEY not set")
}

// 悪い例
const apiKey = "sk_live_xxxxx"  // NG
```

## パフォーマンス

### context.Context のタイムアウト

長時間実行される処理にはタイムアウトを設定します。

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := storage.CreateEntry(ctx, entry); err != nil {
    return fmt.Errorf("failed to create entry: %w", err)
}
```

### ゴルーチンの適切な終了

ゴルーチンリークを防ぐため、適切に終了処理を実装します。

```go
func process(ctx context.Context) {
    done := make(chan struct{})

    go func() {
        defer close(done)
        // 処理
    }()

    select {
    case <-done:
        return
    case <-ctx.Done():
        return
    }
}
```

### N+1問題の回避

データベースクエリのN+1問題に注意します。

```go
// 悪い例: N+1問題
entries := getEntries()
for _, entry := range entries {
    author := getAuthor(entry.AuthorID)  // N回のクエリ
}

// 良い例: バッチ取得
entries := getEntries()
authorIDs := extractAuthorIDs(entries)
authors := getAuthorsByIDs(authorIDs)  // 1回のクエリ
```

## 依存関係管理

### go.mod の管理

```bash
# 依存関係の追加
go get github.com/example/package

# 不要な依存の削除
go mod tidy

# ベンダリング（必要に応じて）
go mod vendor
```

### バージョン固定

本番環境では依存関係のバージョンを固定します。

```bash
# 特定バージョンを指定
go get github.com/example/package@v1.2.3
```

## 参考資料

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
