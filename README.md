# noto

手軽にメモを書き、素早く検索できることを目指したTUI(ターミナル)メモ帳です。

## 特長

- フォルダ階層を作らず、タグと全文検索だけで整理・発見する
- `n` で新規メモを作成 → 使い慣れた `$EDITOR`(vim/nano等)で本文を編集
- 一覧画面に文字を打つだけでインクリメンタルに全文検索(タイトル・本文・タグが対象)
- `t` でタグ一覧を開き、複数タグのAND絞り込み(検索語との併用も可能)
- `dd` → 確認プロンプトでメモを削除(Markdownファイルと索引の両方を削除)
- `?` でキーバインド一覧をオーバーレイ表示

## インストール・実行

### Goがある場合

Go 1.26 以降が必要です。

```sh
go install github.com/sskohei/noto/cmd/noto@latest
```

`$(go env GOPATH)/bin` にPATHが通っていれば、`noto` コマンドとして実行できます。

### Goが無い場合(ビルド済みバイナリ)

[GitHub Releases](https://github.com/sskohei/noto/releases) から自分のOS・アーキテクチャに合ったアーカイブをダウンロードし、展開してPATHの通った場所に置いてください。

```sh
# 例: Linux (amd64)
curl -L -o noto.tar.gz https://github.com/sskohei/noto/releases/latest/download/noto_linux_amd64.tar.gz
tar xzf noto.tar.gz
sudo mv noto /usr/local/bin/
```

### ソースからビルドする場合

```sh
git clone https://github.com/sskohei/noto.git
cd noto
go build -o noto ./cmd/noto
./noto
```

インストールせずその場で試す場合は `go run ./cmd/noto` でも起動できます。

## 使い方

`noto` を実行するとメモ一覧画面が開きます。一覧画面での主なキーバインドは以下の通りです。

| キー | 動作 |
|---|---|
| `j` / `k`(または `↓` / `↑`) | カーソル移動 |
| `n` | 新規メモを作成 |
| `Enter` / `e` | 選択中のメモを編集 |
| `/` | 検索欄にフォーカスしインクリメンタル検索 |
| `t` | タグ一覧を開いてタグで絞り込み |
| `dd` | 選択中のメモを削除(確認プロンプトあり) |
| `?` | ヘルプを表示 |
| `q` / `Ctrl+C` | 終了 |

全画面分のキーバインドは [docs/keybindings.md](docs/keybindings.md) を参照してください。

## 設定

設定ファイルは `~/.config/noto/config.toml`(`$XDG_CONFIG_HOME` で上書き可能)です。存在しない場合はすべてデフォルト値で動作します。

```toml
notes_dir = "~/.local/share/noto/notes"
editor = ""          # 空ならOS標準の $EDITOR / $VISUAL を使用
```

### エディターの変更方法

メモ編集(`n` で新規作成、`Enter` / `e` で既存メモ編集)時に起動するエディターは、以下の優先順位で決まります。

1. `config.toml` の `editor`
2. 環境変数 `$VISUAL`
3. 環境変数 `$EDITOR`
4. いずれも未設定なら `vi`

**環境変数で変更する場合**

シェルの設定ファイル(`~/.bashrc` や `~/.zshrc` など)に追記します。noto以外のコマンドラインツールにも適用されます。

```sh
export EDITOR=nvim
```

**`config.toml` で変更する場合(noto専用に固定したい場合)**

```toml
editor = "nvim"
```

VS Codeのように、エディターの終了を待ってからnoto側の処理を再開したいGUIエディターを使う場合は、待機用のオプションを付けて指定します。

```toml
editor = "code --wait"
```

`editor` はスペース区切りでコマンドと引数をそのまま指定でき、シェルを経由せずに実行されます。そのため `|` や `&&` のようなシェル演算子は使えません。

## データの保存場所

メモは平文Markdownファイル(1メモ = 1ファイル、YAML frontmatter付き)として保存され、これが常に正(source of truth)です。検索用のSQLite索引はいつでも再構築可能なキャッシュに過ぎません。

| 用途 | パス(既定値) | 環境変数での上書き |
|---|---|---|
| メモ本体 | `~/.local/share/noto/notes/*.md` | `$XDG_DATA_HOME/noto/notes/` |
| 検索索引DB | `~/.local/share/noto/index.db` | `$XDG_DATA_HOME/noto/index.db` |
| 設定ファイル | `~/.config/noto/config.toml` | `$XDG_CONFIG_HOME/noto/config.toml` |

詳細なファイル形式・DBスキーマは [docs/data-model.md](docs/data-model.md) を参照してください。

## 技術スタック

- 言語: Go
- TUIフレームワーク: [Bubble Tea](https://github.com/charmbracelet/bubbletea)(+ Bubbles / Lipgloss)
- メモ本体: 平文Markdownファイル(YAML frontmatter付き)
- 検索索引: SQLite(FTS5拡張による全文検索)

## ドキュメント

より詳細な仕様や設計は `docs/` にまとまっています。

- [docs/spec.md](docs/spec.md) — 機能仕様・ユースケース
- [docs/architecture.md](docs/architecture.md) — レイヤ構成と依存方向
- [docs/data-model.md](docs/data-model.md) — ファイル形式・DBスキーマ
- [docs/keybindings.md](docs/keybindings.md) — キーバインド一覧
- [docs/development.md](docs/development.md) — 開発環境・テスト方針
- [docs/non-goals.md](docs/non-goals.md) — v1でやらないこと

## 開発

```sh
go build ./...     # ビルド
go run ./cmd/noto   # 実行
go test ./...       # テスト
go vet ./...        # 静的解析
gofmt -l .           # フォーマットチェック
```

Claude Codeでの開発規約(ブランチ命名規則など)は [CLAUDE.md](CLAUDE.md) を参照してください。

## ライセンス

[MIT License](LICENSE)
