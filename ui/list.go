package ui

import (
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

//s.DimmedTitle = lipgloss.NewStyle().
//Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}).
//Padding(0, 0, 0, 2) //nolint:mnd
//
//s.DimmedDesc = s.DimmedTitle.
//Foreground(lipgloss.AdaptiveColor{Light: "#C2B8C2", Dark: "#4D4D4D"})
//
//s.FilterMatch = lipgloss.NewStyle().Underline(true)

type Status int

const (
	Running Status = iota
	Stopped
)

type Instance struct {
	title  string
	status Status
	height int
	width  int
}

func (i *Instance) String(selected bool, width int) string {
	if selected {
		return lipgloss.Place(width, 1, 0.95, lipgloss.Center, selectedStyle.Render(i.title))
	}
	return lipgloss.Place(width, 1, 0.95, lipgloss.Center, titleStyle.Render(i.title))
}

type List struct {
	items         []*Instance
	selectedIdx   int
	height, width int
}

// SetSize sets the height and width of the list.
func (l *List) SetSize(width, height int) {
	l.width = width
	l.height = height
}

func NewList() *List {
	return &List{
		items: []*Instance{
			{title: "test 1", status: Running},
			{title: "test 2", status: Running},
			{title: "test 3", status: Running},
			{title: "test 4", status: Running},
			{title: "test 5", status: Running},
			{title: "test 6", status: Running},
		},
	}
}

func (l *List) String() string {
	if len(l.items) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(lipgloss.Place(
		l.width, 1, lipgloss.Center, lipgloss.Top, mainTitle.Render("  claude squad beta  ")))
	b.WriteString("\n")
	b.WriteString("\n")
	for i, item := range l.items {
		b.WriteString(item.String(i == l.selectedIdx, l.width))
		if i != len(l.items)-1 {
			b.WriteString("\n\n")
		}
	}
	return lipgloss.Place(l.width, l.height, lipgloss.Left, lipgloss.Top, b.String())

	// If there aren't enough items to fill up this page (always the last page)
	// then we need to add some newlines to fill up the space where items would
	// have been.
	//itemsOnPage := m.Paginator.ItemsOnPage(len(items))
	//if itemsOnPage < m.Paginator.PerPage {
	//	n := (m.Paginator.PerPage - itemsOnPage) * (m.delegate.Height() + m.delegate.Spacing())
	//	if len(items) == 0 {
	//		n -= m.delegate.Height() - 1
	//	}
	//	fmt.Fprint(&b, strings.Repeat("\n", n))
	//}

}
func (*List) Add() {
}

// Down selects the next item in the list.
func (l *List) Down() {
	l.selectedIdx = (l.selectedIdx + 1) % len(l.items)
}

// Up selects the prev item in the list.
func (l *List) Up() {
	if l.selectedIdx == 0 {
		l.selectedIdx = len(l.items) - 1
		return
	}
	l.selectedIdx = l.selectedIdx - 1
}
