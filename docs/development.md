# 開発ガイド

## 現状

このリポジトリはまだ実装前(仕様確定フェーズ)。`go.mod` やソースツリーは存在しない。本ドキュメントは実装開始時に採用する想定の構成・方針をまとめたもので、実装が進むにつれて実態に合わせて更新すること。

## 想定ディレクトリ構成

```
noto/
├── cmd/
│   └── noto/            # main パッケージ(エントリポイント)
├── internal/
│   ├── ui/               # Bubble Tea モデル・画面
│   ├── app/               # ユースケース(作成/一覧/検索/削除)
│   ├── storage/           # Markdownファイルの読み書き
│   ├── index/              # SQLite/FTS5 索引
│   └── config/             # 設定ファイルの読み込み
├── docs/                   # 設計ドキュメント(本フォルダ)
├── go.mod
└── CLAUDE.md
```

依存方向は [architecture.md](architecture.md) の通り、`ui → app → storage/index` の一方向。`internal/` 配下は外部パッケージから参照させない(Goの `internal` 慣習)。

## セットアップ(実装後の想定)

```sh
go mod tidy
go build ./...
go run ./cmd/noto
```

## テスト方針

- **ストレージ層・索引層・アプリケーション層**: 通常のGo単体テスト(`go test ./...`)。frontmatterのパース、ファイル名生成、FTS5クエリの結果などをテーブル駆動テストで検証する
- **UI層**: [`teatest`](https://github.com/charmbracelet/x/tree/main/exp/teatest)(Bubble Tea公式のテストヘルパー)を使い、キー入力シーケンスに対する画面出力をスナップショット的に検証する
- 索引の再構築ロジック(mtime差分検知)は、一時ディレクトリを使った結合テストで検証する

## Lint / フォーマット

```sh
gofmt -l .
go vet ./...
```

lintルールを追加する場合(golangci-lint等の導入を含む)は、このドキュメントと `CLAUDE.md` の開発コマンド一覧を合わせて更新すること。

## コントリビューションの進め方

1. [spec.md](spec.md) で対象機能のユースケースを確認する
2. [architecture.md](architecture.md) でどのレイヤに実装が属するかを確認する
3. [non-goals.md](non-goals.md) にある機能ではないか確認する
4. レイヤの依存方向(UI → app → storage/index)を崩さないように実装する
