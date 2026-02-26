#!/bin/bash
set -euo pipefail

# 植物観察日記 - 明るさ自動調整付き撮影スクリプト
# USBカメラで植物の写真を撮影し、明るさが適正範囲に収まるよう露出を自動調整する。
# 撮影画像は data/photos/ に保存される。
# crontab で定期実行することを想定。
#
# === 使い方 ===
# 基本: ./scripts/capture_auto.sh
#
# === 前提条件 ===
# - fswebcam がインストールされていること
# - ImageMagick（convert コマンド）がインストールされていること
# - カメラが Auto Exposure / Exposure Time, Absolute をサポートしていること

# === 設定 ===
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

DATA_DIR="${PROJECT_DIR}/data/photos"
LOG_FILE="${PROJECT_DIR}/data/capture.log"
RESOLUTION="1280x720"
JPEG_QUALITY="95"
DELAY="1"  # カメラ安定のための遅延（秒）

# 明るさ自動調整パラメータ
BRIGHTNESS_MIN="0.30"
BRIGHTNESS_MAX="0.65"
DEFAULT_EXPOSURE=250
EXPOSURE_MIN=10
EXPOSURE_MAX=5000
MAX_ADJUST_RETRIES=5
EXPOSURE_FILE="${PROJECT_DIR}/data/last_exposure.txt"

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
    if [ -f "${EXPOSURE_FILE}" ]; then
        local val
        val=$(cat "${EXPOSURE_FILE}")
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
    echo "$diff" | sed 's/^-//'
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

# ファイル名の生成（YYYYMMDD_HHMM_UTC.jpg）
DATE=$(date -u +%Y%m%d_%H%M_UTC)
FINAL_OUTPUT="${DATA_DIR}/${DATE}.jpg"

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
TARGET_BRIGHTNESS="0.475"

for ((i = 1; i <= MAX_ADJUST_RETRIES; i++)); do
    # 一時ファイルに撮影
    TMP_OUTPUT="${DATA_DIR}/_tmp_${i}_${DATE}.jpg"
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
