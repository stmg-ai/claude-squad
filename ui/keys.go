package ui

import (
	"github.com/charmbracelet/bubbles/key"
)

type keyName int

const (
	Up keyName = iota
	Down
	Enter
	Newt
	Kill
	Quit
	Review
)

// GlobalKeyStringsMap is a global, immutable map string to keybinding.
var GlobalKeyStringsMap = map[string]keyName{
	"up":    Up,
	"k":     Up,
	"down":  Down,
	"j":     Down,
	"enter": Enter,
	"n":     Newt,
	"d":     Kill,
	"q":     Quit,
	"r":     Review,
}

// GlobalkeyBindings is a global, immutable map of keyName tot keybinding.
var GlobalkeyBindings = map[keyName]key.Binding{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵/enter", "open"),
	),
	Newt: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Kill: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "kill"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	Review: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "review"),
	),
}
