package ui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

var titleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

var selectedStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderForeground(lipgloss.AdaptiveColor{Light: "#F793FF", Dark: "#AD58B4"}).
	Foreground(lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"})

var mainTitle = lipgloss.NewStyle().
	Background(lipgloss.Color("62")).
	Foreground(lipgloss.Color("230"))

const GlobalInstanceLimit = 16

type Status int

const (
	Running Status = iota
	Ready
	Loading
)

type Instance struct {
	title  string
	status Status
	height int
	width  int
}

type List struct {
	items         []*Instance
	selectedIdx   int
	height, width int
}

func NewList() *List {
	return &List{
		items: []*Instance{
			{title: "asdf", status: Running},
			{title: "banana", status: Running},
			{title: "apple banana", status: Running},
			{title: "peach apple", status: Running},
			{title: "peach banana", status: Running},
			{title: "test 6", status: Running},
			{title: "asdf", status: Running},
			{title: "banana", status: Running},
			{title: "apple banana", status: Running},
			{title: "peach apple", status: Running},
			{title: "peach banana", status: Running},
			{title: "test 6", status: Running},
			{title: "asdf", status: Running},
			{title: "banana", status: Running},
			{title: "apple banana", status: Running},
			{title: "peach apple", status: Running},
		},
	}
}

// SetSize sets the height and width of the list.
func (l *List) SetSize(width, height int) {
	l.width = width
	l.height = height
}

func (l *List) NumInstances() int {
	return len(l.items)
}

func (l *List) RenderInstance(idx int, selected bool, text string) string {
	if selected {
		return selectedStyle.Render(fmt.Sprintf(" %d. %s ", idx, text))
	}
	return titleStyle.Render(fmt.Sprintf(" %d. %s ", idx, text))
}

func (l *List) String() string {
	// Write the title.
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(lipgloss.Place(
		l.width, 1, lipgloss.Center, lipgloss.Top, mainTitle.Render("  claude squad beta  ")))
	b.WriteString("\n")
	b.WriteString("\n")

	// Render the list.
	for i, item := range l.items {
		b.WriteString(l.RenderInstance(i+1, i == l.selectedIdx, item.title))
		if i != len(l.items)-1 {
			b.WriteString("\n\n")
		}
	}
	return lipgloss.Place(l.width, l.height, lipgloss.Left, lipgloss.Top, b.String())
}

// Down selects the next item in the list.
func (l *List) Down() {
	if len(l.items) == 0 {
		return
	}
	l.selectedIdx = (l.selectedIdx + 1) % len(l.items)
}

// Kill selects the next item in the list.
func (l *List) Kill() {
	if len(l.items) == 0 {
		return
	}
	// If you delete the last one in the list, select the previous one.
	if l.selectedIdx == len(l.items)-1 {
		defer l.Up()
	}
	// Since there's items after this, the selectedIdx can stay the same.
	l.items = append(l.items[:l.selectedIdx], l.items[l.selectedIdx+1:]...)
}

// Up selects the prev item in the list.
func (l *List) Up() {
	if len(l.items) == 0 {
		return
	}

	if l.selectedIdx == 0 {
		l.selectedIdx = len(l.items) - 1
		return
	}
	l.selectedIdx = l.selectedIdx - 1
}
