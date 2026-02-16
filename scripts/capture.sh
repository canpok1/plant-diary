#!/bin/bash
set -euo pipefail

# 植物観察日記 - 撮影スクリプト
# USBカメラで植物の写真を撮影し、data/photos/ に保存する。
# crontab で定期実行することを想定。

# === 設定 ===
# スクリプトの配置場所から plant-diary ルートを特定
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

DATA_DIR="${PROJECT_DIR}/data/photos"
LOG_FILE="${PROJECT_DIR}/data/capture.log"
RESOLUTION="1280x720"
JPEG_QUALITY="95"
DELAY="1"  # カメラ安定のための遅延（秒）

# 追加オプション（引数から取得）
EXTRA_OPTIONS=("$@")

# === 関数 ===
log_message() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') $1" >> "${LOG_FILE}"
}

# === メイン処理 ===

# 保存先ディレクトリの作成
if [ ! -d "${DATA_DIR}" ]; then
    mkdir -p "${DATA_DIR}"
    log_message "INFO: Created directory ${DATA_DIR}"
fi

# fswebcam の存在確認
if ! command -v fswebcam &> /dev/null; then
    log_message "ERROR: fswebcam が見つかりません。sudo apt install fswebcam でインストールしてください。"
    echo "ERROR: fswebcam が見つかりません。" >&2
    exit 1
fi

# ファイル名の生成（YYYYMMDD_HHMM.jpg）
DATE=$(date +%Y%m%d_%H%M)
OUTPUT="${DATA_DIR}/${DATE}.jpg"

# 撮影（追加オプション付き）
if fswebcam -r "${RESOLUTION}" --jpeg "${JPEG_QUALITY}" -D "${DELAY}" --no-banner ${EXTRA_OPTIONS[@]} "${OUTPUT}" 2>> "${LOG_FILE}"; then
    log_message "INFO: Captured ${OUTPUT}"
    echo "撮影成功: ${OUTPUT}"
else
    log_message "ERROR: 撮影に失敗しました"
    echo "ERROR: 撮影に失敗しました" >&2
    exit 1
fi
