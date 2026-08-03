# データモデル

## ファイル配置(XDG準拠)

| 用途 | パス(既定値) | 環境変数での上書き |
|---|---|---|
| メモ本体 | `~/.local/share/noto/notes/*.md` | `$XDG_DATA_HOME/noto/notes/` |
| todo本体 | `~/.local/share/noto/todos/*.md` | `$XDG_DATA_HOME/noto/todos/` |
| 検索索引DB | `~/.local/share/noto/index.db` | `$XDG_DATA_HOME/noto/index.db` |
| 設定ファイル | `~/.config/noto/config.toml` | `$XDG_CONFIG_HOME/noto/config.toml` |

`XDG_DATA_HOME` / `XDG_CONFIG_HOME` が未設定の場合はそれぞれ `~/.local/share` / `~/.config` にフォールバックする。

## メモファイル形式

1メモ = 1 Markdownファイル。YAML frontmatter + 本文。

```markdown
---
id: 018f2e4a-6b3a-7c9e-9a1a-2f7e1a9c4b21
title: 買い物リスト
tags: [life, shopping]
created_at: 2026-08-01T09:12:00+09:00
updated_at: 2026-08-01T09:20:31+09:00
---

- 牛乳
- コーヒー豆
```

| フィールド | 型 | 説明 |
|---|---|---|
| `id` | string (UUIDv7) | メモの一意識別子。ファイル名変更やタイトル変更後も不変 |
| `title` | string | 一覧表示に使うタイトル。空なら本文1行目を代替表示に使う |
| `tags` | string配列 | フラットなタグ一覧。フォルダ階層は持たない |
| `created_at` | RFC3339 | 作成日時 |
| `updated_at` | RFC3339 | 最終更新日時。保存の都度更新 |

### ファイル名規則

```
<created_at をYYYYMMDDTHHMMSS形式にしたもの>-<titleのslug>.md
例: 20260801T091200-買い物リスト.md
```

`title` が空、またはslug化した結果が空文字になる場合は `id` の先頭8文字を使う。ファイル名はあくまで人間が `ls` した時に分かりやすくするためのものであり、メモの同一性は `id` で判定する(ファイル名だけで一意性を保証しない)。

## todoファイル形式

1 todo = 1 Markdownファイル。メモと同様にYAML frontmatter + 本文(本文は任意の詳細メモ、空でもよい)。

```markdown
---
id: 018f2e4a-7a1e-7b3f-9c2a-4d8e6f1a0b33
title: 経費精算を提出する
done: false
created_at: 2026-08-01T09:12:00+09:00
updated_at: 2026-08-01T09:12:00+09:00
---
```

| フィールド | 型 | 説明 |
|---|---|---|
| `id` | string (UUIDv7) | todoの一意識別子。ファイル名変更やタイトル変更後も不変 |
| `title` | string | 一覧表示に使うタイトル |
| `done` | bool | 完了状態。`Space` で切り替える |
| `created_at` | RFC3339 | 作成日時 |
| `updated_at` | RFC3339 | 最終更新日時(完了状態の切り替えを含め、保存の都度更新) |

タグは持たない(整理はdone/未doneのみ)。ファイル名規則はメモと同じパターンを `todos/` 配下に適用する(`<created_at>-<titleのslug>.md`)。

## SQLite索引スキーマ

索引はキャッシュ。破損・削除時はメモファイル群から全再構築できる。

```sql
-- メモのメタデータキャッシュ
CREATE TABLE notes (
    id          TEXT PRIMARY KEY,
    path        TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    mtime       INTEGER NOT NULL -- ファイルのmtime(差分検知用)
);

-- タグの正規化テーブル(タグ一覧表示・絞り込み用)
CREATE TABLE note_tags (
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    tag     TEXT NOT NULL,
    PRIMARY KEY (note_id, tag)
);

-- 全文検索用の仮想テーブル(FTS5)
CREATE VIRTUAL TABLE notes_fts USING fts5(
    id UNINDEXED,
    title,
    body,
    tags,
    tokenize = 'unicode61 remove_diacritics 2'
);

-- todoのメタデータキャッシュ(全文検索の対象外)
CREATE TABLE todos (
    id          TEXT PRIMARY KEY,
    path        TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    done        INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    mtime       INTEGER NOT NULL -- ファイルのmtime(差分検知用)
);
```

- `notes` / `note_tags` はタグ一覧・絞り込みクエリ用の通常テーブル
- `notes_fts` はタイトル・本文・タグを対象にしたインクリメンタル検索用のFTS5テーブル
- `todos` はtodoリストパネルの一覧表示・完了状態の管理用テーブル。検索パネル・FTS5の対象にはしない
- `mtime` を使い、起動時にファイルシステムをスキャンして差分のみ `notes` / `note_tags` / `notes_fts` / `todos` を更新する

## 設定ファイル(`config.toml`)

```toml
notes_dir = "~/.local/share/noto/notes"
editor = ""          # 空ならOS標準の $EDITOR / $VISUAL を使用
```
