---
name: solve-issue
description: ユーザーが `/solve-issue` で手動実行した場合のみ使用。GitHub Issueの対応を行うスキル。実装、自己レビュー、PR作成、マージ、振り返りまで一連の流れを一気に行う。
disable-model-invocation: true
user-invocable: true
argument-hint: "[issue-number]"
---

GitHub Issue $ARGUMENTS を対応します。

1. Issue の内容を理解する
2. `/tdd` スキルで実装する
3. `/simplify` スキルで自己レビューしてコードを改善する
4. `commit-push-pr` スキルでPRを作成する
5. CIの終了を待機する
  - コマンド: `gh pr checks {PR番号} --watch`
6. `/pr-comments` スキルでレビューコメントを取得し、必要に応じてコードを修正する
  - コードを修正した場合はコミット・プッシュを行いレビューコメントに返信して、手順5に戻る
  - レビューコメントへの返信時は、レビュースレッド内の全レビュワーに対してメンションすること
7. PRをマージする
  - マージできない場合は、原因を確認して必要に応じてコードを修正し、手順5に戻る
8. `/retro` スキルで振り返りを行う
