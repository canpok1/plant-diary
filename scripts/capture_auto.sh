#!/bin/bash
set -euo pipefail

# 植物観察日記 - 明るさ自動調整付き撮影スクリプト
# USBカメラで植物の写真を撮影し、明るさが適正範囲に収まるよう露出を自動調整する。
# crontab で定期実行することを想定（推奨: 5分おき）。
#
# サーバーの GET /api/script-config から以下のフラグを取得して動作を決定する:
#   should_test_capture    ... true のとき POST /api/test-photo にアップロード
#   should_schedule_capture ... true のとき POST /api/scheduled-photo にアップロード
# 両方 false の場合は即終了（負荷ほぼゼロ）。
# 新フラグが存在しない旧サーバーの場合は POST /api/photos への後方互換モードで動作する。
#
# === 使い方 ===
# ./scripts/capture_auto.sh --api-url http://192.168.1.10:8080 --script-key <32文字キー>
#
#   --api-url: APIのベースURL（必須）
#   --script-key: スクリプトキー（32文字）（必須）
#
# === 前提条件 ===
# - fswebcam がインストールされていること
# - ImageMagick（convert コマンド）がインストールされていること
# - curl がインストールされていること
# - カメラが Auto Exposure / Exposure Time, Absolute をサポートしていること

# === 設定 ===
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

DATA_DIR="${PROJECT_DIR}/data/photos"
LOG_FILE="${PROJECT_DIR}/data/capture.log"
RESOLUTION="1280x720"
JPEG_QUALITY="95"
DELAY="1"  # カメラ安定のための遅延（秒）

# 露出調整パラメータ
DEFAULT_EXPOSURE=250
EXPOSURE_MIN=10
EXPOSURE_MAX=5000
EXPOSURE_FILE="${PROJECT_DIR}/data/last_exposure.txt"

# 引数のパース
DIARY_API_URL=""
DIARY_SCRIPT_KEY=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --api-url)
      if [[ -z "${2:-}" || "$2" == --* ]]; then
        echo "ERROR: --api-url には値が必要です。" >&2
        exit 1
      fi
      DIARY_API_URL="$2"; shift 2 ;;
    --script-key)
      if [[ -z "${2:-}" || "$2" == --* ]]; then
        echo "ERROR: --script-key には値が必要です。" >&2
        exit 1
      fi
      DIARY_SCRIPT_KEY="$2"; shift 2 ;;
    *)
      echo "ERROR: 不明な引数: $1" >&2
      exit 1 ;;
  esac
done

# 必須引数のバリデーション
if [ -z "${DIARY_API_URL}" ]; then
    echo "ERROR: --api-url は必須です。" >&2
    exit 1
fi
if [ -z "${DIARY_SCRIPT_KEY}" ]; then
    echo "ERROR: --script-key は必須です。" >&2
    exit 1
fi

# 一時ファイル管理
TMP_FILES=()

# === 関数 ===
log_message() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') $1" >> "${LOG_FILE}"
}

cleanup() {
    for f in "${TMP_FILES[@]}"; do
        if [ -f "$f" ]; then
            rm -f "$f"
        fi
    done
}

trap cleanup EXIT INT TERM

get_brightness() {
    local image="$1"
    convert "$image" -colorspace Gray -format "%[fx:mean]" info:
}

load_exposure() {
    local val
    if [ -r "${EXPOSURE_FILE}" ] && val=$(<"${EXPOSURE_FILE}" 2>/dev/null); then
        if [[ "$val" =~ ^[0-9]+$ ]] && [ "$val" -ge "${EXPOSURE_MIN}" ] && [ "$val" -le "${EXPOSURE_MAX}" ]; then
            echo "$val"
            return
        fi
    fi
    echo "${DEFAULT_EXPOSURE}"
}

save_exposure() {
    local val="$1"
    echo "$val" > "${EXPOSURE_FILE}"
}

capture_with_exposure() {
    local exposure="$1"
    local output="$2"
    fswebcam -r "${RESOLUTION}" --jpeg "${JPEG_QUALITY}" -D "${DELAY}" --no-banner \
        --set "Auto Exposure=Manual Mode" \
        --set "Exposure Time, Absolute=${exposure}" \
        "${output}" 2>> "${LOG_FILE}"
}

# bc を使った浮動小数点比較（第1引数 < 第2引数 なら 0 を返す）
float_lt() {
    [ "$(echo "$1 < $2" | bc -l)" -eq 1 ]
}

# bc を使った浮動小数点比較（第1引数 > 第2引数 なら 0 を返す）
float_gt() {
    [ "$(echo "$1 > $2" | bc -l)" -eq 1 ]
}

# 2つの浮動小数点の差の絶対値
float_abs_diff() {
    local diff
    diff=$(echo "$1 - $2" | bc -l)
    echo "${diff#-}"
}

# 新フラグ（should_test_capture/should_schedule_capture）が存在する場合は新エンドポイントを使用。
# 存在しない場合は後方互換として upload_key + POST /api/photos を使用。
upload_photo() {
    local photo_file="$1"
    local captured_at
    captured_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)

    if [ "${HAS_NEW_FLAGS}" = "true" ]; then
        if [ "${SHOULD_TEST_CAPTURE}" = "true" ]; then
            curl -fsS -X POST \
              -H "Authorization: Bearer ${DIARY_SCRIPT_KEY}" \
              -F "photo=@${photo_file}" \
              "${DIARY_API_URL}/api/test-photo" || log_message "WARN: test-photo upload failed (photo saved locally)"
        fi
        if [ "${SHOULD_SCHEDULE_CAPTURE}" = "true" ]; then
            curl -fsS -X POST \
              -H "Authorization: Bearer ${DIARY_SCRIPT_KEY}" \
              -F "photo=@${photo_file}" \
              -F "captured_at=${captured_at}" \
              "${DIARY_API_URL}/api/scheduled-photo" || log_message "WARN: scheduled-photo upload failed (photo saved locally)"
        fi
    else
        # 後方互換: 旧サーバー向けに POST /api/photos を使用
        curl -fsS -X POST \
          -F "photo=@${photo_file}" \
          -F "upload_key=${DIARY_UPLOAD_KEY}" \
          "${DIARY_API_URL}/api/photos" || log_message "WARN: API upload failed (photo saved locally)"
    fi
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

# ImageMagick（convert）の存在確認
if ! command -v convert &> /dev/null; then
    log_message "ERROR: ImageMagick（convert コマンド）が見つかりません。sudo apt install imagemagick でインストールしてください。"
    echo "ERROR: ImageMagick（convert コマンド）が見つかりません。" >&2
    exit 1
fi

# bc の存在確認
if ! command -v bc &> /dev/null; then
    log_message "ERROR: bc が見つかりません。sudo apt install bc でインストールしてください。"
    echo "ERROR: bc が見つかりません。" >&2
    exit 1
fi

# curl の存在確認
if ! command -v curl &> /dev/null; then
    log_message "ERROR: curl が見つかりません。sudo apt install curl でインストールしてください。"
    echo "ERROR: curl が見つかりません。" >&2
    exit 1
fi

# python3 の存在確認
if ! command -v python3 &> /dev/null; then
    log_message "ERROR: python3 が見つかりません。sudo apt install python3 でインストールしてください。"
    echo "ERROR: python3 が見つかりません。" >&2
    exit 1
fi

# サーバーからスクリプト設定を取得
log_message "INFO: サーバーから設定を取得中..."
SCRIPT_CONFIG_JSON=$(curl -s -f \
    -H "Authorization: Bearer ${DIARY_SCRIPT_KEY}" \
    "${DIARY_API_URL}/api/script-config") || {
    log_message "ERROR: 設定の取得に失敗しました。サーバーURLやスクリプトキーを確認してください。"
    echo "ERROR: 設定の取得に失敗しました。" >&2
    exit 1
}

# JSON から各設定値を取得（python3 を1回呼び出してまとめて解析）
{
    IFS= read -r TARGET_BRIGHTNESS
    IFS= read -r BRIGHTNESS_TOLERANCE
    IFS= read -r MAX_ADJUST_RETRIES
    IFS= read -r DIARY_UPLOAD_KEY
    IFS= read -r SHOULD_TEST_CAPTURE
    IFS= read -r SHOULD_SCHEDULE_CAPTURE
    IFS= read -r HAS_NEW_FLAGS
} < <(echo "${SCRIPT_CONFIG_JSON}" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(d['target_brightness'])
print(d['brightness_tolerance'])
print(d['max_adjust_retries'])
print(d.get('upload_key', ''))
print('true' if d.get('should_test_capture', False) else 'false')
print('true' if d.get('should_schedule_capture', False) else 'false')
print('true' if ('should_test_capture' in d or 'should_schedule_capture' in d) else 'false')
") || {
    log_message "ERROR: レスポンスの解析に失敗しました。"
    echo "ERROR: レスポンスの解析に失敗しました。" >&2
    exit 1
}

log_message "INFO: 設定を取得しました（target_brightness=${TARGET_BRIGHTNESS}, brightness_tolerance=${BRIGHTNESS_TOLERANCE}, max_adjust_retries=${MAX_ADJUST_RETRIES}, should_test_capture=${SHOULD_TEST_CAPTURE}, should_schedule_capture=${SHOULD_SCHEDULE_CAPTURE}）"

# 新しい撮影フラグが存在し、両方 false の場合は撮影をスキップ
if [ "${HAS_NEW_FLAGS}" = "true" ] && [ "${SHOULD_TEST_CAPTURE}" = "false" ] && [ "${SHOULD_SCHEDULE_CAPTURE}" = "false" ]; then
    log_message "INFO: should_test_capture=false, should_schedule_capture=false のため撮影をスキップします"
    exit 0
fi

# 適正輝度範囲の算出
BRIGHTNESS_MIN=$(echo "${TARGET_BRIGHTNESS} - ${BRIGHTNESS_TOLERANCE}" | bc -l)
BRIGHTNESS_MAX=$(echo "${TARGET_BRIGHTNESS} + ${BRIGHTNESS_TOLERANCE}" | bc -l)

# ファイル名の生成（YYYYMMDD_HHMM_UTC.jpg）
DATE=$(date -u +%Y%m%d_%H%M_UTC)
RUN_ID="$$"
FINAL_OUTPUT="${DATA_DIR}/${DATE}.jpg"
if [ -e "${FINAL_OUTPUT}" ]; then
    FINAL_OUTPUT="${DATA_DIR}/${DATE}_${RUN_ID}.jpg"
fi

# 前回の露出値を読み込み
EXPOSURE=$(load_exposure)
EXPOSURE_SOURCE="デフォルト値"
if [ -f "${EXPOSURE_FILE}" ]; then
    EXPOSURE_SOURCE="前回保存値"
fi

log_message "INFO: 明るさ自動調整を開始（初期露出: ${EXPOSURE}、${EXPOSURE_SOURCE}を使用）"

# 各試行の結果を保存する配列
declare -a TRIAL_FILES=()
declare -a TRIAL_BRIGHTNESSES=()
declare -a TRIAL_EXPOSURES=()

BEST_INDEX=-1

for ((i = 1; i <= MAX_ADJUST_RETRIES; i++)); do
    # 一時ファイルに撮影
    TMP_OUTPUT="${DATA_DIR}/_tmp_${RUN_ID}_${i}_${DATE}.jpg"
    TMP_FILES+=("${TMP_OUTPUT}")

    if ! capture_with_exposure "${EXPOSURE}" "${TMP_OUTPUT}"; then
        log_message "ERROR: 試行 ${i}/${MAX_ADJUST_RETRIES} - 撮影に失敗しました（露出: ${EXPOSURE}）"
        echo "ERROR: 撮影に失敗しました" >&2
        exit 1
    fi

    # 平均輝度を算出
    BRIGHTNESS=$(get_brightness "${TMP_OUTPUT}")

    TRIAL_FILES+=("${TMP_OUTPUT}")
    TRIAL_BRIGHTNESSES+=("${BRIGHTNESS}")
    TRIAL_EXPOSURES+=("${EXPOSURE}")

    # 適正範囲の判定
    if ! float_lt "${BRIGHTNESS}" "${BRIGHTNESS_MIN}" && ! float_gt "${BRIGHTNESS}" "${BRIGHTNESS_MAX}"; then
        # 適正範囲内
        log_message "INFO: 試行 ${i}/${MAX_ADJUST_RETRIES} - 露出: ${EXPOSURE}, 平均輝度: ${BRIGHTNESS} (適正範囲内)"
        log_message "INFO: 明るさ自動調整完了 - 最終露出: ${EXPOSURE}, 最終輝度: ${BRIGHTNESS}"

        # 最終画像として保存
        mv "${TMP_OUTPUT}" "${FINAL_OUTPUT}"

        # 一時ファイルリストから除外（移動済み）
        TMP_FILES=("${TMP_FILES[@]/${TMP_OUTPUT}/}")

        # 露出値を保存
        save_exposure "${EXPOSURE}"
        log_message "INFO: 露出値を保存: ${EXPOSURE}"
        log_message "INFO: Captured ${FINAL_OUTPUT}"
        echo "撮影成功: ${FINAL_OUTPUT}"

        # API登録
        upload_photo "${FINAL_OUTPUT}"

        exit 0
    fi

    # 範囲外の場合
    if float_lt "${BRIGHTNESS}" "${BRIGHTNESS_MIN}"; then
        log_message "INFO: 試行 ${i}/${MAX_ADJUST_RETRIES} - 露出: ${EXPOSURE}, 平均輝度: ${BRIGHTNESS} (暗すぎ)"
        # 露出を2倍に増加
        EXPOSURE=$((EXPOSURE * 2))
    else
        log_message "INFO: 試行 ${i}/${MAX_ADJUST_RETRIES} - 露出: ${EXPOSURE}, 平均輝度: ${BRIGHTNESS} (明るすぎ)"
        # 露出を半分に減少
        EXPOSURE=$((EXPOSURE / 2))
    fi

    # 露出値の上下限をクランプ
    if [ "${EXPOSURE}" -lt "${EXPOSURE_MIN}" ]; then
        EXPOSURE="${EXPOSURE_MIN}"
    fi
    if [ "${EXPOSURE}" -gt "${EXPOSURE_MAX}" ]; then
        EXPOSURE="${EXPOSURE_MAX}"
    fi
done

# 最大リトライ回数に達した場合：最も適正に近い画像を採用
log_message "INFO: 最大リトライ回数（${MAX_ADJUST_RETRIES}）に達しました。最も適正に近い画像を採用します。"

BEST_INDEX=0
BEST_DIFF=$(float_abs_diff "${TRIAL_BRIGHTNESSES[0]}" "${TARGET_BRIGHTNESS}")

for ((j = 1; j < ${#TRIAL_BRIGHTNESSES[@]}; j++)); do
    DIFF=$(float_abs_diff "${TRIAL_BRIGHTNESSES[$j]}" "${TARGET_BRIGHTNESS}")
    if float_lt "${DIFF}" "${BEST_DIFF}"; then
        BEST_INDEX=$j
        BEST_DIFF="${DIFF}"
    fi
done

BEST_FILE="${TRIAL_FILES[${BEST_INDEX}]}"
BEST_BRIGHTNESS="${TRIAL_BRIGHTNESSES[${BEST_INDEX}]}"
BEST_EXPOSURE="${TRIAL_EXPOSURES[${BEST_INDEX}]}"

log_message "INFO: 最適画像を選択 - 試行 $((BEST_INDEX + 1)), 露出: ${BEST_EXPOSURE}, 平均輝度: ${BEST_BRIGHTNESS}"

# 最適画像を最終出力先に移動
mv "${BEST_FILE}" "${FINAL_OUTPUT}"

# 一時ファイルリストから除外（移動済み）
TMP_FILES=("${TMP_FILES[@]/${BEST_FILE}/}")

# 露出値を保存
save_exposure "${BEST_EXPOSURE}"
log_message "INFO: 露出値を保存: ${BEST_EXPOSURE}"
log_message "INFO: Captured ${FINAL_OUTPUT}"
echo "撮影成功: ${FINAL_OUTPUT}"

# API登録
upload_photo "${FINAL_OUTPUT}"
