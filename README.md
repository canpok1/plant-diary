# 植物観察日記システム

USBカメラで撮影した植物の画像を元に、Google Gemini APIを使って観察日記を自動生成し、Webブラウザから閲覧できるセルフホスト型システムです。

## 特徴

- USBカメラで定期的に植物を撮影（cron連携）
- Gemini APIが画像から観察日記を自動生成
- Webブラウザでスマートフォンからも閲覧可能
- Docker Composeで簡単にデプロイ
- SQLiteでデータ管理（バックアップが容易）

## システム構成

```
┌──────────────┐    ┌─────────────────────────────┐
│  USBカメラ    │    │  Docker コンテナ              │
│  + cron      │───>│  ┌─────────┐  ┌───────────┐ │
│  (ホスト側)   │    │  │ Worker  │  │ Webサーバー │ │
│              │    │  │(Gemini) │  │ (:8080)   │ │
└──────────────┘    │  └────┬────┘  └─────┬─────┘ │
                    │       │             │       │
                    │  ┌────▼─────────────▼────┐  │
                    │  │      SQLite DB        │  │
                    │  └───────────────────────┘  │
                    └─────────────────────────────┘
```

## 必要な環境

- Linux (Ubuntu/Debian推奨)
- Docker / Docker Compose
- USBカメラ
- fswebcam（ホスト側での撮影用）
- Google Gemini API キー

## セットアップ

### 1. リポジトリをクローン

```bash
git clone https://github.com/your-username/plant-diary.git
cd plant-diary
```

### 2. 環境変数を設定

```bash
cp .env.example .env
```

`.env` ファイルを編集し、Gemini API キーを設定します。

```bash
# .env
GEMINI_API_KEY=your_actual_api_key_here
```

API キーは [Google AI Studio](https://aistudio.google.com/apikey) から取得できます。

### 3. データディレクトリを作成

```bash
mkdir -p data/photos
```

### 4. fswebcam をインストール（ホスト側）

```bash
sudo apt update
sudo apt install -y fswebcam
```

USBカメラが認識されているか確認します。

```bash
ls /dev/video*
```

### 5. 撮影スクリプトに実行権限を付与

```bash
chmod +x scripts/capture.sh
```

### 6. Docker Compose で起動

```bash
docker compose up -d
```

起動後、ブラウザで `http://localhost:8080` にアクセスすると日記一覧ページが表示されます。

## 使い方

### 手動で撮影する

```bash
./scripts/capture.sh
```

撮影した画像は `data/photos/YYYYMMDD_HHMM.jpg` として保存されます。Worker が1分以内に画像を検出し、Gemini API で日記を自動生成します。

### crontab で定期撮影を設定する

毎日12時に自動撮影する例：

```bash
crontab -e
```

以下の行を追加します。

```cron
0 12 * * * /path/to/plant-diary/scripts/capture.sh
```

複数回撮影したい場合の例（朝8時と夕方17時）：

```cron
0 8 * * * /path/to/plant-diary/scripts/capture.sh
0 17 * * * /path/to/plant-diary/scripts/capture.sh
```

### 日記を閲覧する

ブラウザで以下のURLにアクセスします。

| ページ | URL |
| --- | --- |
| 日記一覧 | `http://localhost:8080/` |
| 日記詳細 | `http://localhost:8080/diary/{id}` |

## ディレクトリ構造

```
.
├── docker-compose.yml      # Docker Compose設定
├── Dockerfile              # コンテナビルド定義
├── .env                    # 環境変数（Git管理外）
├── .env.example            # 環境変数テンプレート
├── app/                    # Go ソースコード
│   ├── main.go             # エントリーポイント
│   ├── server.go           # Webサーバー
│   ├── worker.go           # 日記生成Worker
│   ├── gemini.go           # Gemini API連携
│   ├── db.go               # SQLite操作
│   ├── migrations/         # DBマイグレーション
│   └── templates/          # HTMLテンプレート
├── data/                   # 永続データ（Git管理外）
│   ├── photos/             # 撮影画像
│   └── plant_log.db        # SQLiteデータベース
├── scripts/
│   └── capture.sh          # 撮影スクリプト
└── docs/
    └── spec.md             # 仕様書
```

## バックアップ

SQLiteは単一ファイルのため、`data/` ディレクトリをコピーするだけでバックアップできます。

### rsync でNASにバックアップする例

```bash
rsync -av /path/to/plant-diary/data/ /mnt/nas/plant-diary-backup/
```

### crontab で毎日バックアップする例

```cron
0 2 * * * rsync -av /path/to/plant-diary/data/ /mnt/nas/plant-diary-backup/
```

## トラブルシューティング

### カメラが認識されない

```bash
# デバイスの確認
ls /dev/video*

# USBデバイスの確認
lsusb
```

デバイスが表示されない場合、カメラが正しく接続されているか確認してください。

### 撮影スクリプトがエラーになる

```bash
# fswebcam がインストールされているか確認
which fswebcam

# 手動で撮影テスト
fswebcam -r 1280x720 --jpeg 95 test.jpg
```

権限エラーの場合、ユーザーを `video` グループに追加してください。

```bash
sudo usermod -aG video $USER
# ログアウトして再ログイン
```

### Docker コンテナが起動しない

```bash
# ログを確認
docker compose logs

# コンテナの状態を確認
docker compose ps
```

### 日記が生成されない

- `.env` ファイルに正しい `GEMINI_API_KEY` が設定されているか確認してください
- `data/photos/` ディレクトリに `.jpg` ファイルが存在するか確認してください
- Docker コンテナのログで Worker のエラーを確認してください

```bash
docker compose logs -f
```

## 技術スタック

| 要素 | 技術 |
| --- | --- |
| 言語 | Go |
| データベース | SQLite 3 |
| AIエンジン | Google Gemini 2.5 Flash |
| カメラ制御 | fswebcam |
| コンテナ | Docker / Docker Compose |
| DBマイグレーション | golang-migrate/migrate |
