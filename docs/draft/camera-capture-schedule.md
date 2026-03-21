# カメラ撮影スケジュール ＆ テスト撮影機能 仕様書

## 概要

カメラ設定画面からテスト撮影を実行できるようにする。あわせて、撮影タイミングをカメラごとに設定できるスケジュール機能を追加する。

## アーキテクチャ方針

- ポーリング方式（cronで5分おき）で実現
- `should_capture` の判定はサーバー側で行い、スクリプトは判定ロジックを持たない
- テスト撮影と通常（スケジュール）撮影でエンドポイント・保存先を分離
- スケジュールはカメラ単位で設定
- 時刻指定はUIでJST入力、DB保存はUTC（暗黙的JST保存はしない）

## エンドポイント設計

| 用途 | エンドポイント | 認証 | 処理 |
|------|--------------|------|------|
| テストモード有効化 | `PATCH /cameras/{id}` | cookie（requireLogin） | `test_capture_requested = 1` に更新 |
| テスト写真アップロード | `POST /api/test-photo` | script_key | ファイル保存＋camera更新＋フラグクリア |
| スケジュール写真アップロード | `POST /api/scheduled-photo` | script_key | script_key→camera→book解決＋写真保存＋`last_scheduled_capture_at`更新 |
| テスト写真表示 | `GET /cameras/{id}/test-photo` | cookie（requireLogin） | ファイルをサーブ |
| 設定取得（スクリプト用） | `GET /api/script-config` | script_key | `should_test_capture`, `should_schedule_capture` を追加（`upload_key`は移行期間中は維持し、全スクリプト移行後に廃止） |
| （既存）写真アップロード | `POST /api/photos` | upload_key | 既存のまま維持（後方互換） |

## スクリプトフロー

```text
[cron 5分おき] ./capture_auto.sh --api-url ... --script-key ...

  GET /api/script-config
    → { should_test_capture, should_schedule_capture, upload_key, ... }
    ※ upload_key は後方互換のため移行期間中は引き続き返却

  if should_test_capture == false AND should_schedule_capture == false:
    exit 0（何もしない）

  [カメラで撮影、露出自動調整（既存処理）]

  if should_test_capture == true:
    POST /api/test-photo (Authorization: Bearer <script_key>)
    → サーバー: ファイル保存 + last_test_photo_path/captured_at更新 + test_capture_requested=0

  if should_schedule_capture == true:
    POST /api/scheduled-photo (Authorization: Bearer <script_key>)
    → サーバー: 写真保存（日記エントリ作成）+ last_scheduled_capture_at更新

※ 両方 true の場合は同じ写真を両エンドポイントにアップロード
```

### cronの推奨設定

```bash
# 5分おきに実行（should_capture=falseのときは即終了するため負荷はほぼゼロ）
*/5 * * * * /path/to/capture_auto.sh --api-url http://... --script-key <key>
```

## DBスキーマ変更

### camerasテーブルへの追加カラム（マイグレーション 000011）

```sql
-- テスト撮影
ALTER TABLE cameras ADD COLUMN test_capture_requested INTEGER NOT NULL DEFAULT 0;
ALTER TABLE cameras ADD COLUMN last_test_photo_path TEXT;
ALTER TABLE cameras ADD COLUMN last_test_photo_captured_at DATETIME;  -- UTC

-- スケジュール撮影
-- 複数時刻をカンマ区切りで保存（UTC HH:MM形式）
-- 例: "03:00,09:00" = JST 12:00, 18:00
-- NULL はスケジュールなし
ALTER TABLE cameras ADD COLUMN capture_times_utc TEXT;
ALTER TABLE cameras ADD COLUMN last_scheduled_capture_at DATETIME;  -- UTC、重複防止用
```

## API仕様

### GET /api/script-config

既存フィールドに `should_test_capture` / `should_schedule_capture` を追加する。`upload_key` は既存スクリプト互換のため移行期間中は返却を継続し、全スクリプト移行後に削除する（廃止予定日を別途定義）。

```json
{
  "target_brightness": 0.475,
  "brightness_tolerance": 0.175,
  "max_adjust_retries": 5,
  "upload_key": "<key>",
  "should_test_capture": false,
  "should_schedule_capture": false
}
```

`should_test_capture` と `should_schedule_capture` は独立したbool値。

#### upload_key の移行方針

- **移行期間中**: `GET /api/script-config` は `upload_key` を引き続き返却する。`scripts/capture_auto.sh` は `upload_key` を使った既存の `POST /api/photos` を継続利用可。
- **切替条件**: 全カメラのスクリプトが `script_key` 認証による新エンドポイント（`POST /api/scheduled-photo`）に移行完了した後。
- **廃止**: 切替完了後、`upload_key` フィールドをレスポンスから削除する（廃止予定日はスクリプト移行完了時点で別途決定）。
- **ロールバック**: 廃止前であれば `upload_key` を再度有効化することで即時ロールバック可能。

`should_capture` という統合フラグは提供しない。スクリプトはサーバーが返す `should_test_capture` と `should_schedule_capture` をそれぞれ参照して動作を決定する（判定ロジックはサーバー側のみ）。

#### should_schedule_capture の判定ロジック

1. `capture_times_utc` が NULL または空 → false
2. いずれかの登録時刻 T について:
   - `now > T` かつ
   - `last_scheduled_capture_at IS NULL` または `last_scheduled_capture_at < T`
   → true

### PATCH /cameras/{id}

**認証**: cookie（ログイン必須）
**Content-Type**: `application/json`
**Request Body**:
```json
{"test_capture_requested": true}
```

**Response**: `200 OK`
```json
{"status": "ok"}
```

### POST /api/test-photo

**認証**: `Authorization: Bearer <script_key>`
**Content-Type**: `multipart/form-data`
**フォームフィールド**: `photo`（画像ファイル）

**処理**:
1. ファイルを `data/test-photos/{camera_id}_latest.jpg` に保存（上書き）
2. `cameras` テーブルを更新:
   - `test_capture_requested = 0`
   - `last_test_photo_path = <path>`
   - `last_test_photo_captured_at = <now UTC>`

**Response**: `200 OK`

### POST /api/scheduled-photo

**認証**: `Authorization: Bearer <script_key>`
**Content-Type**: `multipart/form-data`
**フォームフィールド**:
- `photo`（画像ファイル）
- `captured_at`（RFC3339形式、省略時はサーバー受信時刻）

**処理**:
1. script_key → camera → book を解決
2. 既存の `POST /api/photos` と同様に写真を保存（日記エントリ作成）
3. `cameras.last_scheduled_capture_at = now()` に更新

**Response**: `200 OK`

### GET /cameras/{id}/test-photo

**認証**: cookie（ログイン必須）
最新のテスト写真ファイルをサーブ。未撮影の場合は `404 Not Found`。

## UI仕様

### カメラ設定画面 (`/cameras/{id}/settings`)

#### 撮影スケジュールセクション（輝度設定の下に追加）

```text
撮影スケジュール
┌──────────────────────────────────────────────────┐
│ 撮影時刻（JST）                                  │
│ ※1行に1つ、HH:MM形式で入力してください          │
│ ┌────────────────────────────────────────────┐   │
│ │ 12:00                                      │   │
│ │ 18:00                                      │   │
│ └────────────────────────────────────────────┘   │
│ 空欄にするとスケジュール撮影を無効化します        │
└──────────────────────────────────────────────────┘
```

- `<textarea>` で複数時刻を1行ずつ入力（JST "HH:MM" 形式）
- `<label>` に "(JST)" を明記
- 保存時: ハンドラ側でJST→UTC変換してDB保存
- 表示時: UTC→JST変換して表示

#### テスト撮影セクション（danger-zone の前）

- 「テスト撮影をリクエスト」ボタン
  - `TestCaptureRequested = true` のとき → disabled「リクエスト中...（5分以内に実行）」
  - クリック時: JavaScript fetch で `PATCH /cameras/{id}` → `{"test_capture_requested": true}`
- テスト撮影結果エリア:
  - `LastTestPhotoPath.Valid = true` のとき:
    - `<img src="/cameras/{id}/test-photo">` を表示
    - 撮影日時を JST で表示（例: `撮影日時: 2026-03-21 12:00 JST`）
  - `LastTestPhotoPath.Valid = false` のとき: 「まだテスト撮影がありません」

## 将来の「N分おき」スケジュール対応

`capture_times_utc` は TEXT 型のためフォーマットを将来拡張可能：

- 現在: `"03:00,09:00"` （UTC時刻リスト）
- 将来: `"interval:30"` （30分おき）など

`computeCaptureDecision` 関数でフォーマット判定を分岐させるだけで対応可能。
