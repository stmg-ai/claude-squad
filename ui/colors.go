package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color definitions for the UI
var (
	// Main brand/highlight colors
	Purple = lipgloss.AdaptiveColor{Light: "#E3D8F1", Dark: "#DABECA"}

	// Status colors
	Green  = lipgloss.AdaptiveColor{Light: "#51bd73", Dark: "#51bd73"}
	Red    = lipgloss.AdaptiveColor{Light: "#de613e", Dark: "#de613e"}
	Blue   = lipgloss.AdaptiveColor{Light: "#0ea5e9", Dark: "#0ea5e9"}
	Yellow = lipgloss.AdaptiveColor{Light: "#eab308", Dark: "#eab308"}

	// Diff colors (using more vibrant colors for better readability)
	DiffGreen = lipgloss.Color("#22c55e")
	DiffRed   = lipgloss.Color("#ef4444")
	DiffBlue  = lipgloss.Color("#0ea5e9")

	// Text colors
	TextPrimary   = lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}
	TextSecondary = lipgloss.AdaptiveColor{Light: "#7A7474", Dark: "#9C9494"}
	TextMuted     = lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}
	TextGray      = lipgloss.AdaptiveColor{Light: "#888888", Dark: "#888888"}

	// Background colors
	BgSelection = lipgloss.Color("#E3D8F1")
	BgTitle     = lipgloss.Color("#DABECA")

	// Misc colors
	TitleText     = lipgloss.Color("#BF8B85")
	ActionGroup   = lipgloss.Color("#AD8A64")
	MenuText      = lipgloss.Color("#BF8B85")
	SeparatorLine = lipgloss.AdaptiveColor{Light: "#DABECA", Dark: "#AD8A64"}
)
