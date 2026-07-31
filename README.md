# noto

手軽にメモを書き、素早く検索できることを目指したTUI(ターミナル)メモ帳です。

> **Status**: 仕様確定フェーズです。実装コードはまだ存在しません。現在はドキュメント(`CLAUDE.md` / `docs/`)で仕様を固めている段階です。

## コンセプト

- フォルダ階層を作らず、タグだけで整理する
- 一覧画面に文字を打つだけでインクリメンタルに全文検索できる
- 本文の編集は使い慣れた `$EDITOR`(vim/nano等)に任せる

## 技術スタック(予定)

- 言語: Go
- TUI: [Bubble Tea](https://github.com/charmbracelet/bubbletea)(+ Bubbles / Lipgloss)
- メモ本体: Markdownファイル(YAML frontmatter付き)
- 検索索引: SQLite(FTS5)

## ドキュメント

詳細な仕様や設計は `docs/` にまとまっています。

- [docs/spec.md](docs/spec.md) — 機能仕様・ユースケース
- [docs/architecture.md](docs/architecture.md) — レイヤ構成と依存方向
- [docs/data-model.md](docs/data-model.md) — ファイル形式・DBスキーマ
- [docs/keybindings.md](docs/keybindings.md) — キーバインド一覧
- [docs/development.md](docs/development.md) — 開発環境・テスト方針
- [docs/non-goals.md](docs/non-goals.md) — v1でやらないこと

Claude Codeでの開発規約(開発コマンド・ブランチ命名規則など)は [CLAUDE.md](CLAUDE.md) を参照してください。

## ライセンス

[MIT License](LICENSE)
