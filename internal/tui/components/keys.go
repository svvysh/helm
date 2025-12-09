package components

import tea "github.com/charmbracelet/bubbletea"

// NormalizeKey reconciles common legacy/terminal variations so downstream
// update logic can rely on a single set of key identifiers.
func NormalizeKey(msg tea.KeyMsg) tea.KeyMsg {
	switch msg.String() {
	case "shift+tab":
		msg.Type = tea.KeyShiftTab
	case "ctrl+i", "tab":
		msg.Type = tea.KeyTab
	case "ctrl+h":
		msg.Type = tea.KeyBackspace
	case "backspace2":
		msg.Type = tea.KeyBackspace
	case "delete":
		msg.Type = tea.KeyDelete
	case "enter", "ctrl+m":
		msg.Type = tea.KeyEnter
	}
	return msg
}
