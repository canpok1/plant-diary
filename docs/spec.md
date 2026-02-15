# 植物観察日記システム 仕様書

## 1. システム概要

USBカメラで撮影した植物の画像を元に、Gemini APIを用いて日記を自動生成し、Webサイトとして閲覧可能にするセルフホスト型システム。

## 2. システム構成

### 2.1 実行環境

* **ホストOS**: Linux (Ubuntu/Debian等)
* **コンテナ基盤**: Docker / Docker Compose
* **バックアップ先**: 自宅NAS (NFS/SMBマウント想定)

### 2.2 技術スタック

* **言語**: Go (Golang)
* **データベース**: SQLite 3
* **AIエンジン**: Google Gemini 2.5 Flash (API)
* **カメラ制御**: fswebcam (ホスト側Cron実行)
* **DBマイグレーション**: golang-migrate/migrate

---

## 3. ディレクトリ構造

```text
.
├── docker-compose.yml
├── Dockerfile
├── .env                    # Gemini APIキー（.gitignore対象）
├── .env.example            # 環境変数テンプレート
├── app/                    # Go ソースコード
│   ├── main.go             # エントリーポイント
│   ├── server.go           # Webサーバー
│   ├── worker.go           # 日記生成Worker
│   ├── gemini.go           # Gemini API連携
│   ├── db.go               # SQLite操作
│   ├── migrations/         # DBマイグレーションファイル
│   │   ├── 000001_create_diary_table.up.sql
│   │   └── 000001_create_diary_table.down.sql
│   ├── templates/
│   │   ├── index.html      # 一覧ページ
│   │   └── detail.html     # 詳細ページ
│   └── go.mod              # Go依存関係
├── data/                   # 永続データ（.gitignore対象）
│   ├── photos/             # 植物画像ファイル (.jpg)
│   └── plant_log.db        # SQLite データベース
├── scripts/
│   └── capture.sh          # 撮影用スクリプト (ホスト側)
└── docs/
    └── spec.md             # 仕様書
```

---

## 4. 機能仕様

### 4.1 画像撮影プロセス (Host側)

1. **トリガー**: Cronによる定期実行（例：毎日 12:00）。
2. **処理**:
   * `fswebcam` 等を用いてUSBカメラから撮影。
   * `data/photos/YYYYMMDD_HHMM.jpg` 形式で保存。
   * 撮影成功後、Dockerコンテナ内のWorkerがポーリングで自動検知。

#### スクリプト例 (`scripts/capture.sh`)

```bash
#!/bin/bash
set -e

DATA_DIR="/path/to/plant-diary/data/photos"
DATE=$(date +%Y%m%d_%H%M)
OUTPUT="${DATA_DIR}/${DATE}.jpg"

fswebcam -r 1280x720 --jpeg 95 -D 1 "${OUTPUT}"

echo "$(date): Captured ${OUTPUT}" >> /var/log/plant-diary-capture.log
```

#### crontab設定例

```cron
0 12 * * * /path/to/plant-diary/scripts/capture.sh
```



### 4.2 日記生成プロセス (Worker側)

1. **新着検知**: `data/photos/` に未処理の画像があるか確認。
2. **AI解析**:
* 画像を Gemini API (Vision機能) に送信。
* 「成長の様子」や「変化」にフォーカスした日記を生成。


3. **DB保存**:
* 画像パス、日記本文、生成日時をSQLiteに記録。



### 4.3 Web公開機能 (Server側)

1. **一覧表示**: 過去の日記をカレンダー逆順（新着順）でリスト表示。
2. **詳細表示**: 高解像度画像と日記本文の閲覧。
3. **レスポンシブ対応**: スマートフォンからの閲覧を考慮した簡易デザイン。

---

## 5. データモデル (SQLite)

### Table: `diary`

| カラム名 | 型 | 説明 |
| --- | --- | --- |
| `id` | INTEGER | プライマリキー（連番） |
| `image_path` | TEXT | 保存された画像の相対パス |
| `content` | TEXT | Geminiが生成した日記本文 |
| `created_at` | DATETIME | レコード作成日時 |

**注記**: スキーマは `golang-migrate/migrate` を用いたマイグレーションファイルで管理。詳細は「## 11. DBマイグレーション」を参照。

---

## 6. 運用・バックアップ

* **DBバックアップ**: SQLiteファイルが単一のため、`data/` ディレクトリを丸ごとNASへ `rsync` または `cp` するだけで完了。
* **環境変数の管理**: Gemini APIキーなどの機密情報は `.env` ファイルで管理。

---

## 7. Webサーバー仕様

### 7.1 基本設定

* **ポート**: 8080
* **公開範囲**: ローカルネットワークのみ（認証なし）
* **テンプレートエンジン**: Go標準の `html/template`

### 7.2 エンドポイント

| パス | メソッド | 説明 |
| --- | --- | --- |
| `/` | GET | 日記一覧ページ（新着順） |
| `/diary/:id` | GET | 日記詳細ページ |
| `/photos/:filename` | GET | 画像ファイル配信 |

### 7.3 UI/UX

* **デザイン**: シンプルな白背景、読みやすいフォント
* **レスポンシブ**: メディアクエリでスマートフォン対応
* **CSS**: 素のCSS（フレームワーク不使用）

#### 一覧ページ
* タイトル: "植物観察日記"
* 表示項目: 撮影日時、画像（CSS縮小）、本文の冒頭50文字
* ソート: 新着順（`created_at DESC`）
* レイアウト: カード形式

#### 詳細ページ
* 画像: 元サイズ表示（`max-width: 100%`）
* 日記本文: 全文表示
* 戻るリンク: 一覧へ

---

## 8. Gemini API仕様

### 8.1 基本設定

* **モデル**: `gemini-2.5-flash`
* **APIキー**: 環境変数 `GEMINI_API_KEY` で管理
* **タイムアウト**: 30秒
* **リトライ**: 最大3回（指数バックオフ: 1秒、2秒、4秒）

### 8.2 プロンプト設計

```
この植物の写真を見て、成長の様子や変化を観察してください。
親しみやすい口調で、200文字程度の観察日記を書いてください。
```

### 8.3 エラーハンドリング

* **API失敗時**: 3回リトライ後、エラーログを出力してスキップ
* **該当画像**: DBに記録せず、次回ポーリング時に再試行

---

## 9. Worker仕様

### 9.1 実行方式

* **トリガー**: Goroutineで1分ごとにポーリング
* **未処理判定**:
  1. `data/photos/*.jpg` のファイル一覧を取得
  2. DBの `image_path` と照合
  3. DBに存在しないファイル = 未処理

### 9.2 処理フロー

1. 未処理画像を検出
2. 画像ファイルを読み込み
3. Gemini APIに送信して日記を生成
4. DBに保存（`image_path`, `content`, `created_at`）
5. 成功ログを出力

### 9.3 並行処理

* 1枚ずつ順次処理（並行処理なし）

---

## 10. Docker構成

### 10.1 コンテナ構成

* **コンテナ数**: 1（Web + Worker統合）
* **ベースイメージ**: `golang:1.23-alpine`

### 10.2 docker-compose.yml

```yaml
services:
  plant-diary:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    env_file:
      - .env
    restart: unless-stopped
```

### 10.3 Dockerfile

```dockerfile
FROM golang:1.23-alpine
WORKDIR /app
COPY app/ .
RUN go build -o plant-diary .
CMD ["./plant-diary"]
```

### 10.4 環境変数

`.env` ファイルで管理：

```bash
GEMINI_API_KEY=your_api_key_here
```

---

## 11. DBマイグレーション

### 11.1 ツール

* **ライブラリ**: [golang-migrate/migrate](https://github.com/golang-migrate/migrate)
* **実行タイミング**: サーバー起動時に自動実行

### 11.2 マイグレーションファイル

`app/migrations/` ディレクトリに配置：

#### `000001_create_diary_table.up.sql`

```sql
CREATE TABLE IF NOT EXISTS diary (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    image_path TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_created_at ON diary(created_at DESC);
```

#### `000001_create_diary_table.down.sql`

```sql
DROP INDEX IF EXISTS idx_created_at;
DROP TABLE IF EXISTS diary;
```

### 11.3 実装方針

* サーバー起動時に `migrate.Up()` を実行
* マイグレーション失敗時はアプリケーション起動を中止

---

## 12. ログ・エラーハンドリング

### 12.1 ログ設定

* **ログレベル**: INFO（本番）、DEBUG（開発時は環境変数で切替）
* **出力先**: 標準出力（Dockerログとして収集）
* **フォーマット**: テキスト形式
  * 例: `2024-01-15 12:00:00 INFO: Processing image...`

### 12.2 エラー処理方針

| エラー種別 | 対応 |
| --- | --- |
| Gemini API失敗 | 3回リトライ後、ログ出力してスキップ |
| DB接続失敗 | アプリケーション起動失敗（Dockerが再起動） |
| 画像読み込み失敗 | ログ出力してスキップ |

