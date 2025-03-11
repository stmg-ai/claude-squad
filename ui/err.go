package ui

import (
	"github.com/charmbracelet/lipgloss"
)

type ErrBox struct {
	height, width int
	err           error
}

func NewErrBox() *ErrBox {
	return &ErrBox{}
}

func (e *ErrBox) SetError(err error) {
	e.err = err
}

func (e *ErrBox) Clear() {
	e.err = nil
}

func (e *ErrBox) SetSize(width, height int) {
	e.width = width
	e.height = height
}

func (e *ErrBox) String() string {
	var err string
	if e.err != nil {
		err = e.err.Error()
	}
	return lipgloss.Place(e.width, e.height, lipgloss.Center, lipgloss.Center, descStyle.Render(err))
}
