---
paths:
  - ".github/workflows/**"
---

# GitHub Actions ルール

詳細は [`docs/rules/github-actions-guide.md`](../../docs/rules/github-actions-guide.md) を参照してください。

## セキュリティ考慮事項

- セキュリティが重要な場合は、タグではなくコミット SHA を使用することを検討
  - 例: `actions/checkout@8e5e7e5ab8b370d6c329ec480221332ada57f0ab`
  - タグが移動されるリスクを防げる

## バージョン管理

- Dependabot や Renovate などの自動更新ツールの活用を推奨
  - GitHub Actions のバージョンを最新の状態に保つことができる
