package ui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

var titleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

var listDescStyle = titleStyle.Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})

var selectedTitleStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderForeground(lipgloss.AdaptiveColor{Light: "#F793FF", Dark: "#AD58B4"}).
	Foreground(lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"})

var selectedDescStyle = selectedTitleStyle.Foreground(lipgloss.AdaptiveColor{Light: "#F793FF", Dark: "#AD58B4"})

var mainTitle = lipgloss.NewStyle().
	Background(lipgloss.Color("62")).
	Foreground(lipgloss.Color("230"))

type Status int

const (
	Running Status = iota
	Ready
	Loading
)

type Instance struct {
	title  string
	path   string // workspace path?
	status Status
	height int
	width  int
}

type List struct {
	items         []*Instance
	selectedIdx   int
	height, width int

	// global spinner which is always spinning. we can choose to render it or not
	spinner *spinner.Model
}

func NewList(spinner *spinner.Model) *List {
	return &List{
		items: []*Instance{
			{title: "asdf", path: "../blah", status: Running},
			{title: "banana", path: "../blah", status: Running},
			{title: "apple banana", path: "../blah", status: Running},
			{title: "peach apple", path: "../blah", status: Running},
			{title: "peach banana", path: "../blah", status: Running},
			{title: "test 6", path: "../blah", status: Running},
			{title: "asdf", path: "../blah", status: Ready},
			{title: "banana", path: "../blah", status: Loading},
			{title: "apple banana", path: "../blah", status: Ready},
			{title: "peach apple", path: "../blah", status: Loading},
			//{title: "peach banana", path: "../blah", status: Running},
			//{title: "test 6", path: "../blah", status: Running},
			//{title: "asdf", path: "../blah", status: Running},
			//{title: "banana", path: "../blah", status: Running},
			//{title: "apple banana", path: "../blah", status: Running},
			//{title: "peach apple", path: "../blah", status: Running},
		},
		spinner: spinner,
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

func (i *Instance) Render(idx int, selected bool, spinner *spinner.Model) string {
	prefix := fmt.Sprintf(" %d. ", idx)
	titleS := selectedTitleStyle
	descS := selectedDescStyle
	if !selected {
		titleS = titleStyle
		descS = listDescStyle
	}

	title := titleS.Render(fmt.Sprintf("%s %s", prefix, i.title))

	// add spinner next to title if it's running
	if i.status == Running {
		title = lipgloss.JoinHorizontal(
			lipgloss.Left,
			title,
			" ",
			spinner.View(),
		)
	}

	// join title and subtitle
	text := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		descS.Render(fmt.Sprintf("%s %s", strings.Repeat(" ", len(prefix)), i.path)),
	)

	return text

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
		b.WriteString(item.Render(i+1, i == l.selectedIdx, l.spinner))
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
