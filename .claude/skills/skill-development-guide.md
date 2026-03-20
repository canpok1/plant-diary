# スキル開発ガイド

スキル（SKILL.md）を作成・修正する際のガイドライン。

## スキル名の正式表記

SKILL.md 内で他のスキルを参照する際は、**必ず正式な完全名**を使用すること。

### スキルの種類と正式名称の形式

| 種類 | 形式 | 例 |
|---|---|---|
| プラグインスキル | `{namespace}:{skill-name}` | `commit-commands:commit-push-pr` |
| ローカルスキル（`.claude/skills/` 配下） | スラッシュコマンド形式 `/` + スキル名 | `/monologue`, `/review`, `/tdd` |

### 正式名称の確認方法

- **プラグインスキル**: `settings.json` の `enabledPlugins` セクションでプラグイン名を確認し、`{plugin-name}:{command-name}` 形式で記述する
- **ローカルスキル**: `.claude/skills/{skill-name}/SKILL.md` の `name` フィールドで確認し、`/{name}` 形式で記述する

### 悪い例と良い例

```markdown
# ❌ 悪い例: 省略形
`commit-push-pr` スキルでPRを作成する

# ✅ 良い例: 正式な完全名
`commit-commands:commit-push-pr` スキルでPRを作成する
```

```markdown
# ❌ 悪い例: スラッシュコマンドなしでローカルスキルを参照
`monologue` スキルで通知する

# ✅ 良い例: スラッシュコマンド形式で参照
`/monologue` スキルで通知する
```

### 背景

省略形を使うと：
- レビュアー（CodeRabbit 等）が正式名称と一致しないと判断し、CHANGES_REQUESTED が発生する
- スキルの意図が伝わりにくくなる

## SKILL.md の構造

```markdown
---
name: {スキル名}
description: {スキルの説明}
allowed-tools: {使用するツール}
user-invocable: true/false（ユーザーが /skill-name で直接呼び出せる場合は true）
argument-hint: "[引数ヒント]"（引数が必要な場合）
---

{スキルの本文}
```
