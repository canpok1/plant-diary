- 日本語で回答してください。

## CLAUDE.mdの方針

- このファイルはできるだけシンプルに保つ
- 詳細なルールは `.claude/rules/` に分離する（パス固有ルールで必要時のみ読み込まれる）
- スキル固有の情報は `.claude/skills/` のスキル定義に記載する（スキル呼び出し時のみ読み込まれる）

## スキルの管理範囲

### このリポジトリで管理しているスキル（編集可能）

- `managing-team`: 複数のClaudeエージェントで協力開発
- `quality-check`: コミットやPR作成後のレビュー
- `create-issue`: 仕様相談からGitHub Issueを作成

これらのスキルは `.claude/skills/` 配下で管理されており、改善提案がある場合はこのリポジトリにIssueを作成してください。

### 外部プラグインで管理しているスキル（編集不可）

- `base-tools:create-pr`: PR作成
- `base-tools:fix-pr`: PR修正・マージ
- `base-tools:teardown-worktree`: worktree削除
- `base-tools:retro`: 振り返り
- `base-tools:solve-issue`: Issue解決フロー
- `base-tools:setup-worktree`: worktree作成

これらのスキルは外部プラグイン（canpok1-plugins/base-tools）で管理されており、このリポジトリでは編集できません。改善提案がある場合は、https://github.com/canpok1/claude-code-plugins にIssueを作成してください。
