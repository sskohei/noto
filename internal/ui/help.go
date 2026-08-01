package ui

import tea "github.com/charmbracelet/bubbletea"

// helpText mirrors docs/keybindings.md.
const helpText = `ヘルプ

一覧画面
  j / ↓         カーソルを1つ下へ
  k / ↑         カーソルを1つ上へ
  n             新規メモを作成(タイトル入力 → $EDITOR 起動)
  Enter / e     選択中のメモを $EDITOR で編集
  dd            選択中のメモを削除(確認プロンプトあり)
  /             検索欄にフォーカス。入力ごとにインクリメンタル検索
  Esc           検索/絞り込みをクリア(検索欄フォーカス中) / 確認プロンプトのキャンセル
  t             タグ一覧を開き、タグで絞り込み
  ?             ヘルプ(このキーバインド一覧)を表示
  q / Ctrl+C    終了

検索欄フォーカス中
  文字入力       クエリに追加し、即座に一覧を再フィルタ
  Backspace     クエリを1文字削除
  Enter         クエリを保持したまま一覧にフォーカスを戻す
  Esc           クエリをクリアし、全件表示の一覧にフォーカスを戻す

タグ一覧画面
  j / k         タグ間を移動
  Enter / Space タグの選択・解除(複数選択でAND絞り込み)
  Esc           一覧画面に戻る

削除確認プロンプト
  y             削除を確定
  n / Esc       キャンセル

ヘルプ表示中
  ? / Esc / q   ヘルプを閉じて一覧に戻る
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
