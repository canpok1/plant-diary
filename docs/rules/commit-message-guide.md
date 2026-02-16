# Commit Message Guide

このプロジェクトでは [Conventional Commits](https://www.conventionalcommits.org/ja/v1.0.0/) のルールに従ってコミットメッセージを作成します。

## 基本形式

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

## Type（必須）

- **feat**: 新機能の追加
- **fix**: バグ修正
- **docs**: ドキュメントのみの変更
- **style**: コードの動作に影響しない変更（フォーマット、セミコロン追加など）
- **refactor**: バグ修正や機能追加を伴わないコード変更
- **perf**: パフォーマンス改善
- **test**: テストの追加や修正
- **chore**: ビルドプロセスやツールの変更
- **ci**: CI設定ファイルやスクリプトの変更

## 例

### 新機能の追加
```
feat(diary): エントリーのCRUD機能を実装

- CreateEntry, ReadEntry, UpdateEntry, DeleteEntry を追加
- インメモリストレージを使用
- エラーハンドリングを実装
```

### バグ修正
```
fix(storage): エントリー削除時のnil参照を修正
```

### ドキュメント更新
```
docs: READMEにセットアップ手順を追加
```

### リファクタリング
```
refactor(storage): インターフェース分離の原則を適用
```

### BREAKING CHANGE
```
feat(api)!: エントリーIDの型をintからstringに変更

BREAKING CHANGE: Entry.ID の型が int から string に変更されました。
既存のコードを更新する必要があります。
```

## このプロジェクトでのルール

1. **descriptionは日本語でも英語でもOK**（チームで統一）
2. **1行目は50文字以内を目安に**
3. **bodyで詳細を説明する**（複雑な変更の場合）
4. **BREAKING CHANGEは必ず明記**

## Co-Authored-By（Claude使用時）

Claudeが作業に関与した場合は、フッターに Co-Authored-By を追加：

```
feat(diary): エントリー検索機能を追加

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
```

## 参考

- [Conventional Commits 公式サイト（日本語）](https://www.conventionalcommits.org/ja/v1.0.0/)
