---
paths:
  - ".github/workflows/**"
---

# GitHub Actions ルール

## バージョン指定

- **`@master` や `@latest` は使用しない**
  - 再現性のない動作を防ぐため、必ず特定のバージョンタグを指定すること
  - 例: `uses: ludeeus/action-shellcheck@v2.0.0`
- **`v` プレフィックスの扱い**
  - 公式・サードパーティ製を問わず、すべてのアクションは `v` プレフィックス付きで記述すること
    - 例: `@v2`、`@v1.0.0`、`@v2.3.4`
  - 例: `actions/checkout@v4`、`actions/setup-go@v5`、`ludeeus/action-shellcheck@v2.0.0`
