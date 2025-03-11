package keys

import (
	"github.com/charmbracelet/bubbles/key"
)

type KeyName int

const (
	KeyUp KeyName = iota
	KeyDown
	KeyEnter
	KeyNew
	KeyKill
	KeyQuit
	KeyReview
)

// GlobalKeyStringsMap is a global, immutable map string to keybinding.
var GlobalKeyStringsMap = map[string]KeyName{
	"up":    KeyUp,
	"k":     KeyUp,
	"down":  KeyDown,
	"j":     KeyDown,
	"enter": KeyEnter,
	"n":     KeyNew,
	"d":     KeyKill,
	"q":     KeyQuit,
	"r":     KeyReview,
}

// GlobalkeyBindings is a global, immutable map of KeyName tot keybinding.
var GlobalkeyBindings = map[KeyName]key.Binding{
	KeyUp: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	KeyDown: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	KeyEnter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("↵/enter", "open"),
	),
	KeyNew: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	KeyKill: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "kill"),
	),
	KeyQuit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	KeyReview: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "review"),
	),
}
