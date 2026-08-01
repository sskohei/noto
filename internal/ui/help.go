package ui

import tea "github.com/charmbracelet/bubbletea"

// helpText mirrors docs/keybindings.md.
const helpText = `ヘルプ

メイン画面(共通)
  1 / /         検索パネルにフォーカス
  2 / t         タグパネルにフォーカス
  3             メモ一覧パネルにフォーカス
  n             新規メモを作成(タイトル入力 → $EDITOR 起動)
  dd            選択中のメモを削除(確認プロンプトあり)
  ?             ヘルプ(このキーバインド一覧)を表示
  q / Ctrl+C    終了(Ctrl+C はどこからでも終了)

メモ一覧パネル
  j / ↓         カーソルを1つ下へ
  k / ↑         カーソルを1つ上へ
  Enter / e     選択中のメモを $EDITOR で編集

検索パネル
  文字入力       クエリに追加し、即座に一覧を再フィルタ
  Backspace     クエリを1文字削除
  Enter         クエリを保持したままメモ一覧にフォーカスを戻す
  Esc           クエリをクリアし、全件表示にしてメモ一覧にフォーカスを戻す

タグパネル
  j / k         タグ間を移動
  Enter / Space タグの選択・解除(複数選択でAND絞り込み)
  Esc           メモ一覧にフォーカスを戻す

削除確認プロンプト
  y             削除を確定
  n / Esc       キャンセル

ヘルプ表示中
  ? / Esc / q   ヘルプを閉じてメイン画面に戻る
`

// updateHelp handles key input for modeHelp.
func (m Model) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "?", "esc", "q":
		m.mode = modeMain
		return m, nil
	}

	return m, nil
}

func (m Model) viewHelp() string {
	return helpText
}
