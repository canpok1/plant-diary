---
paths:
  - ".github/workflows/**"
---

# GitHub Actions ルール

## バージョン指定

- **`@master` や `@latest` は使用しない**
  - 再現性のない動作を防ぐため、必ず特定のバージョンタグを指定すること
  - 例: `uses: ludeeus/action-shellcheck@2.0.0`
- **`v` プレフィックスの扱い**
  - サードパーティ製アクションは `v` プレフィックスなしで記述すること
    - 例: `@2.0.0`（`@v2.0.0` ではなく）
  - ただし `actions/` や `github/` などの公式アクションは `v` プレフィックスが慣例のため、それに従うこと
    - 例: `actions/checkout@v4`、`actions/setup-go@v5`
