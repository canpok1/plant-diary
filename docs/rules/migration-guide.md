# マイグレーション管理ルール

- **適用済みマイグレーションファイルは変更しない**
  - `migrations/` 配下の既存ファイルは原則変更禁止
  - golang-migrateは適用済みマイグレーションを再実行しないため、既存環境に変更が反映されない
  - 既存環境と新規環境でスキーマの不整合が生じるリスクがある
- **スキーマ変更やデータ修正は新しいマイグレーションファイルを追加して対応する**
  - 過去のマイグレーションで作成したデータの削除・修正も、新しいマイグレーションで行う
  - 新規環境では全マイグレーションが順番に実行されるため、最終状態は同じになる
- **ファイル命名規則**: `{次の連番}_{説明}.{up,down}.sql`（連番は6桁ゼロパディング）

## down.sqlの作成ルール

### カラム削除時の再作成

SQLiteはカラム削除のALTER TABLE DROP COLUMNをサポートしているが、down.sqlでその逆操作（元の状態に戻す）を行う場合は、テーブルの再作成が必要になることが多い。

**カラムを追加するup.sqlに対するdown.sqlを作成する場合:**

1. **down.sql作成前に、up.sqlが対象とするテーブルの既存スキーマを確認する**
   - `migrations/` 配下の過去のup.sqlファイルを参照し、テーブルの完全なスキーマを把握する
2. **元のテーブルを完全なスキーマ定義で再作成する**
   - PRIMARY KEY、AUTOINCREMENT、NOT NULL、DEFAULT値、UNIQUE制約、CHECK制約、FOREIGN KEY等を省略しない
   - 以下のような手順でテーブルを再作成する:

```sql
-- 1. データを一時テーブルに退避
CREATE TABLE {table}_backup AS SELECT {必要なカラム} FROM {table};

-- 2. 元のテーブルを削除
DROP TABLE {table};

-- 3. 完全なスキーマ定義で元のテーブルを再作成
CREATE TABLE {table} (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    -- 元のスキーマを完全に再現する
    ...
);

-- 4. データを復元
INSERT INTO {table} SELECT * FROM {table}_backup;

-- 5. 一時テーブルを削除
DROP TABLE {table}_backup;
```

### 禁止事項

- **本テーブルの再作成に `CREATE TABLE AS SELECT` を使用することは禁止**
  - この構文はデータはコピーされるが、PRIMARY KEY、NOT NULL、DEFAULT値、UNIQUE制約、CHECK制約、FOREIGN KEY等の制約が失われる
  - 一時退避テーブル（`_backup` 等）の作成は例外として許可（制約が不要なため）
  - 最終的に残るテーブルの再作成では必ず完全なスキーマ定義を記述すること
