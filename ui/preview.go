package ui

type PreviewPane struct {
	width     int
	maxHeight int
}

func NewPreviewPane(width, maxHeight int) *PreviewPane {
	return &PreviewPane{width: width, maxHeight: maxHeight}
}
