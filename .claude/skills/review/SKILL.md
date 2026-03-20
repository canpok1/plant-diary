---
name: review
description: コード品質レビューとドキュメント整合性チェックを統合的に実施するスキル。/simplify でコード品質をレビューし、ドキュメント検証サブエージェントでドキュメントの最新性を検証する。
allowed-tools: Skill, Agent, Bash, Read, Grep, Glob, Write, Edit
user-invocable: true
---

自己レビュー（コード品質 + ドキュメント整合性チェック）を行います。

## 手順

1. `/simplify` スキルを呼び出して、変更コードの再利用性・品質・効率のレビューと改善を行う
2. 修正があった場合はコミットする
