package ui

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
)

var titleStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"}).
	Padding(0, 0, 0, 2)

var selectedStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder(), false, false, false, true).
	BorderForeground(lipgloss.AdaptiveColor{Light: "#F793FF", Dark: "#AD58B4"}).
	Foreground(lipgloss.AdaptiveColor{Light: "#EE6FF8", Dark: "#EE6FF8"}).
	Padding(0, 0, 0, 1)

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
}

func (i *Instance) String(selected bool) string {
	if selected {
		return selectedStyle.Render(i.title)
	}
	return titleStyle.Render(i.title)
}

type List struct {
	items       []*Instance
	selectedIdx int
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

	for i, item := range l.items {
		b.WriteString(item.String(i == l.selectedIdx))
		//if i != len(docs)-1 {
		//	fmt.Fprint(&b, strings.Repeat("\n", m.delegate.Spacing()+1))
		//}
	}

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

	return b.String()
}
func (*List) Add() {
}

// Down selects the next item in the list.
func (*List) Down() {
}

// Up selects the prev item in the list.
func (*List) Up() {

}
