#!/bin/bash
if [ -z "${WORKSPACE_DIR:-}" ]; then
    echo "[ERROR] 環境変数 'WORKSPACE_DIR' が設定されていません。"
    exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
    echo "[ERROR] python3 が見つかりません。"
    exit 1
fi
python3 - "$1" "$2" <<'PY' > "${WORKSPACE_DIR}/.devcontainer/host-notifier.json"
import json, sys
msg = sys.argv[1] if len(sys.argv) > 1 and sys.argv[1] else "Done"
title = sys.argv[2] if len(sys.argv) > 2 and sys.argv[2] else "Dev Container"
print(json.dumps({"message": msg, "title": title}, ensure_ascii=False))
PY
