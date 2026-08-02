# noto — TUIメモ帳

手軽にメモを書き、素早く検索できることを目的としたターミナル(TUI)メモ帳。

## 現在のステータス

v1相当の実装(新規作成・編集・検索・タグ絞り込み・削除・ヘルプ表示・スプラッシュ画面)は完了しており、通常の機能開発フェーズにある。`go.mod` / `cmd/noto` / `internal/{app,config,index,storage,ui}` に実装とテストが揃っている。仕様に迷ったら実装より先に `docs/` を更新すること。

## 技術スタック

- 言語: Go
- TUIフレームワーク: [Bubble Tea](https://github.com/charmbracelet/bubbletea)(+ Bubbles / Lipgloss)
- メモ本体: 平文Markdownファイル(1メモ = 1ファイル、YAML frontmatter付き)
- 検索索引: SQLite(FTS5拡張による全文検索)

詳細は [docs/architecture.md](docs/architecture.md) と [docs/data-model.md](docs/data-model.md) を参照。

## ドキュメント

第三者が開発に参入する際は、まず `docs/` 配下を読むこと。

| ファイル | 内容 |
|---|---|
| [docs/spec.md](docs/spec.md) | 機能仕様・ユースケース |
| [docs/architecture.md](docs/architecture.md) | レイヤ構成と依存方向 |
| [docs/data-model.md](docs/data-model.md) | ファイル形式・DBスキーマ |
| [docs/keybindings.md](docs/keybindings.md) | キーバインド一覧 |
| [docs/development.md](docs/development.md) | 開発環境・テスト方針 |
| [docs/non-goals.md](docs/non-goals.md) | v1でやらないこと |

## 開発コマンド

開発時によく使うコマンド:

```sh
go build ./...     # ビルド
go run ./cmd/noto   # 実行
go test ./...       # テスト
go vet ./...        # 静的解析
gofmt -l .           # フォーマットチェック
```

## ブランチ命名規則

`<種別>/<内容を表す短い説明>` の形式を使う(種別はkebab-case、説明は英語推奨)。

| 種別 | 用途 | 例 |
|---|---|---|
| `feature/` | 新機能の追加 | `feature/tag-filter` |
| `fix/` | バグ修正 | `fix/index-mtime-diff` |

## コーディング規約

- Go標準の `gofmt` / `go vet` に従う。lint例外を追加する場合は理由をコメントで残す。
- ストレージ層(Markdownファイル入出力)、索引層(SQLite/FTS5)、UI層(Bubble Tea)を明確に分離し、UI層がストレージ/索引の実装詳細(ファイルパス組み立てやSQL)に直接依存しないようにする。詳細は [docs/architecture.md](docs/architecture.md)。
- Markdownファイルが常に正(source of truth)。SQLite索引はいつでも再構築可能なキャッシュとして扱い、索引だけにしか存在しないデータを作らない。
- 新機能を追加する前に [docs/non-goals.md](docs/non-goals.md) を確認し、v1スコープ外なら別途相談する。
