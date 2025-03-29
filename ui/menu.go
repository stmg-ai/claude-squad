package ui

import (
	"claude-squad/keys"
	"strings"

	"claude-squad/session"

	"github.com/charmbracelet/lipgloss"
)

var keyStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#655F5F",
	Dark:  "#7F7A7A",
})

var descStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#7A7474",
	Dark:  "#9C9494",
})

var sepStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
	Light: "#DDDADA",
	Dark:  "#3C3C3C",
})

var actionGroupStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

var separator = " • "
var verticalSeparator = " │ "

var menuStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("205"))

// MenuState represents different states the menu can be in
type MenuState int

const (
	StateDefault MenuState = iota
	StateNewInstance
	StatePrompt
)

type Menu struct {
	options       []keys.KeyName
	height, width int
	state         MenuState
	instance      *session.Instance
	isInDiffTab   bool

	// keyDown is the key which is pressed. The default is -1.
	keyDown keys.KeyName
}

var defaultMenuOptions = []keys.KeyName{keys.KeyNew, keys.KeyPrompt, keys.KeyQuit}
var newInstanceMenuOptions = []keys.KeyName{keys.KeySubmitName}
var promptMenuOptions = []keys.KeyName{keys.KeyEnter}

func NewMenu() *Menu {
	return &Menu{
		options:     defaultMenuOptions,
		state:       StateDefault,
		isInDiffTab: false,
		keyDown:     -1,
	}
}

func (m *Menu) Keydown(name keys.KeyName) {
	m.keyDown = name
}

func (m *Menu) ClearKeydown() {
	m.keyDown = -1
}

// SetState updates the menu state and options accordingly
func (m *Menu) SetState(state MenuState) {
	m.state = state
	m.updateOptions()
}

// SetInstance updates the current instance and refreshes menu options
func (m *Menu) SetInstance(instance *session.Instance) {
	m.instance = instance
	if m.state == StateDefault {
		m.updateOptions()
	}
}

// SetInDiffTab updates whether we're currently in the diff tab
func (m *Menu) SetInDiffTab(inDiffTab bool) {
	m.isInDiffTab = inDiffTab
	if m.state == StateDefault {
		m.updateOptions()
	}
}

// updateOptions updates the menu options based on current state and instance
func (m *Menu) updateOptions() {
	switch m.state {
	case StateDefault:
		m.options = defaultMenuOptions
		if m.instance != nil {
			m.addInstanceOptions()
		}
	case StateNewInstance:
		m.options = newInstanceMenuOptions
	case StatePrompt:
		m.options = promptMenuOptions
	}
}

func (m *Menu) addInstanceOptions() {
	// Instance management group
	options := []keys.KeyName{keys.KeyNew, keys.KeyKill}

	// Action group
	actionGroup := []keys.KeyName{keys.KeyEnter, keys.KeySubmit}
	if m.instance.Status == session.Paused {
		actionGroup = append(actionGroup, keys.KeyResume)
	} else {
		actionGroup = append(actionGroup, keys.KeyCheckout)
	}

	// Navigation group (when in diff tab)
	if m.isInDiffTab {
		actionGroup = append(actionGroup, keys.KeyShiftUp)
	}

	// System group
	systemGroup := []keys.KeyName{keys.KeyTab, keys.KeyQuit}

	// Combine all groups
	options = append(options, actionGroup...)
	options = append(options, systemGroup...)

	m.options = options
}

// SetSize sets the width of the window. The menu will be centered horizontally within this width.
func (m *Menu) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *Menu) String() string {

	// Define group boundaries
	groups := []struct {
		start int
		end   int
	}{
		{0, 2}, // Instance management group (n, d)
		{2, 5}, // Action group (enter, submit, pause/resume)
		{5, 7}, // System group (tab, q)
	}

	if m.isInDiffTab {
		groups = append(groups, struct{ start, end int }{7, 9}) // Navigation group (scroll up/down)
	}

	var actionKeys []keys.KeyName
	var nonActionKeys []keys.KeyName
	for i, k := range m.options {
		inActionGroup := i >= groups[1].start && i < groups[1].end
		if inActionGroup {
			actionKeys = append(actionKeys, k)
		} else {
			nonActionKeys = append(nonActionKeys, k)
		}
	}

	var actionLine strings.Builder
	for i, k := range actionKeys {
		binding := keys.GlobalkeyBindings[k]
		localActionStyle := actionGroupStyle
		if m.keyDown == k {
			localActionStyle = localActionStyle.Underline(true)
		}
		actionLine.WriteString(localActionStyle.Render(binding.Help().Key))
		actionLine.WriteString(" ")
		actionLine.WriteString(localActionStyle.Render(binding.Help().Desc))
		if i != len(actionKeys)-1 {
			actionLine.WriteString(sepStyle.Render(separator))
		}
	}
	var nonActionLine strings.Builder
	for i, k := range nonActionKeys {
		binding := keys.GlobalkeyBindings[k]
		localKeyStyle := keyStyle
		localDescStyle := descStyle
		if m.keyDown == k {
			localKeyStyle = localKeyStyle.Underline(true)
			localDescStyle = localDescStyle.Underline(true)
		}
		nonActionLine.WriteString(localKeyStyle.Render(binding.Help().Key))
		nonActionLine.WriteString(" ")
		nonActionLine.WriteString(localDescStyle.Render(binding.Help().Desc))
		if i != len(nonActionKeys)-1 {
			nonActionLine.WriteString(sepStyle.Render(separator))
		}
	}

	//var s strings.Builder
	//for i, k := range m.options {
	//	binding := keys.GlobalkeyBindings[k]
	//
	//	var (
	//		localActionStyle = actionGroupStyle
	//		localKeyStyle    = keyStyle
	//		localDescStyle   = descStyle
	//	)
	//	if m.keyDown == k {
	//		localActionStyle = localActionStyle.Underline(true)
	//		localKeyStyle = localKeyStyle.Underline(true)
	//		localDescStyle = localDescStyle.Underline(true)
	//	}
	//
	//	// Check if we're in the action group (middle group)
	//	inActionGroup := i >= groups[1].start && i < groups[1].end
	//
	//	if inActionGroup {
	//		s.WriteString(localActionStyle.Render(binding.Help().Key))
	//		s.WriteString(" ")
	//		s.WriteString(localActionStyle.Render(binding.Help().Desc))
	//	} else {
	//		s.WriteString(localKeyStyle.Render(binding.Help().Key))
	//		s.WriteString(" ")
	//		s.WriteString(localDescStyle.Render(binding.Help().Desc))
	//	}
	//
	//	// Add appropriate separator
	//	if i != len(m.options)-1 {
	//		isGroupEnd := false
	//		for _, group := range groups {
	//			if i == group.end-1 {
	//				s.WriteString(sepStyle.Render(verticalSeparator))
	//				isGroupEnd = true
	//				break
	//			}
	//		}
	//		if !isGroupEnd {
	//			s.WriteString(sepStyle.Render(separator))
	//		}
	//	}
	//}

	//centeredMenuText := menuStyle.Render(s.String())
	menuContent := lipgloss.JoinVertical(lipgloss.Center, actionLine.String(), "", nonActionLine.String())
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, menuContent)
}
