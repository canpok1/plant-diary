---
paths:
  - "internal/adapter/handler/**"
---

# photo 系ハンドラ実装ルール

## multipart/form-data 処理

- `ParseMultipartForm` の前に `http.MaxBytesReader(w, r.Body, 32<<20)` でボディサイズを制限すること
  - 超過時は 413 Request Entity Too Large を返す

## 時刻変数の命名

- サーバー受信時刻は `receivedAt := time.Now().UTC()` とする
- フォームから受け取る撮影時刻は `capturedAt` とする（両者を混同しない）
- カメラ更新（`UpdateCameraAfter*`）には `receivedAt` を渡す

## ファイル保存

- ファイル名に UUID サフィックスを付与して同一秒の衝突を回避する
  - 例: `capturedAt.Format("20060102_150405") + "_" + uuid + "_UTC.jpg"`
- `os.Create` の代わりに `os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)` を使用する
