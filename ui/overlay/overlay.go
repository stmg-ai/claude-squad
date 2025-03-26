package overlay

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/ansi"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/termenv"
)

// Most of this code is modified from https://github.com/charmbracelet/lipgloss/pull/102

// WhitespaceOption sets a styling rule for rendering whitespace.
type WhitespaceOption func(*whitespace)

// Split a string into lines, additionally returning the size of the widest
// line.
func getLines(s string) (lines []string, widest int) {
	lines = strings.Split(s, "\n")

	for _, l := range lines {
		w := ansi.PrintableRuneWidth(l)
		if widest < w {
			widest = w
		}
	}

	return lines, widest
}

func CalculateCenterCoordinates(foregroundLines []string, backgroundLines []string, foregroundWidth, backgroundWidth int) (int, int) {
	// Calculate the x-coordinate to horizontally center the foreground text.
	x := (backgroundWidth - foregroundWidth) / 2

	// Calculate the y-coordinate to vertically center the foreground text.
	y := (len(backgroundLines) - len(foregroundLines)) / 2

	return x, y
}

// PlaceOverlay places fg on top of bg with an optional shadow effect.
// If center is true, the foreground is centered on the background; otherwise, the provided x and y are used.
func PlaceOverlay(
	x, y int,
	fg, bg string,
	shadow bool,
	center bool,
	opts ...WhitespaceOption,
) string {
	fgLines, fgWidth := getLines(fg)
	bgLines, bgWidth := getLines(bg)
	bgHeight := len(bgLines)
	fgHeight := len(fgLines)

	// Apply a fade effect to the background by directly modifying each line
	// Create a new array of background lines with the fade effect applied
	fadedBgLines := make([]string, len(bgLines))

	// Compile regular expressions for ANSI color codes
	// Match background color codes like \x1b[48;2;R;G;Bm or \x1b[48;5;Nm
	bgColorRegex := regexp.MustCompile(`\x1b\[48;[25];[0-9;]+m`)

	// Match foreground color codes like \x1b[38;2;R;G;Bm or \x1b[38;5;Nm
	fgColorRegex := regexp.MustCompile(`\x1b\[38;[25];[0-9;]+m`)

	// Match simple color codes like \x1b[31m
	simpleColorRegex := regexp.MustCompile(`\x1b\[[0-9]+m`)

	for i, line := range bgLines {
		// Replace background color codes with a faded version
		content := bgColorRegex.ReplaceAllString(line, "\x1b[48;5;236m") // Dark gray background

		// Replace foreground color codes with a faded version
		content = fgColorRegex.ReplaceAllString(content, "\x1b[38;5;240m") // Medium gray foreground

		// Replace simple color codes with a faded version
		content = simpleColorRegex.ReplaceAllStringFunc(content, func(match string) string {
			// Skip reset codes
			if match == "\x1b[0m" {
				return match
			}
			// Replace with dimmed color
			return "\x1b[38;5;240m" // Medium gray
		})

		fadedBgLines[i] = content
	}

	// Replace the original background with the faded version
	bgLines = fadedBgLines

	// Determine placement coordinates
	placeX, placeY := x, y
	if center {
		placeX, placeY = CalculateCenterCoordinates(fgLines, bgLines, fgWidth, bgWidth)
	}

	// Handle shadow if enabled
	if shadow {
		// Define shadow style and character
		shadowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
		shadowChar := shadowStyle.Render("░")

		// Create shadow string with same dimensions as foreground
		shadowLines := make([]string, fgHeight)
		for i := 0; i < fgHeight; i++ {
			shadowLines[i] = strings.Repeat(shadowChar, fgWidth)
		}
		shadowStr := strings.Join(shadowLines, "\n")

		// Place shadow on background at an offset (e.g., +1, +1)
		const shadowOffsetX, shadowOffsetY = 1, 1
		_ = PlaceOverlay(placeX+shadowOffsetX, placeY+shadowOffsetY, shadowStr, bg, false, false, opts...)
	}

	// Check if foreground exceeds background size
	if fgWidth >= bgWidth && fgHeight >= bgHeight {
		return fg // Return foreground if it's larger than background
	}

	// Clamp coordinates to ensure foreground fits within background
	placeX = clamp(placeX, 0, bgWidth-fgWidth)
	placeY = clamp(placeY, 0, bgHeight-fgHeight)

	// Apply whitespace options
	ws := &whitespace{}
	for _, opt := range opts {
		opt(ws)
	}

	// Build the output string
	var b strings.Builder
	for i, bgLine := range bgLines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i < placeY || i >= placeY+fgHeight {
			b.WriteString(bgLine)
			continue
		}

		pos := 0
		if placeX > 0 {
			left := truncate.String(bgLine, uint(placeX))
			pos = ansi.PrintableRuneWidth(left)
			b.WriteString(left)
			if pos < placeX {
				b.WriteString(ws.render(placeX - pos))
				pos = placeX
			}
		}

		fgLine := fgLines[i-placeY]
		b.WriteString(fgLine)
		pos += ansi.PrintableRuneWidth(fgLine)

		right := cutLeft(bgLine, pos)
		bgLineWidth := ansi.PrintableRuneWidth(bgLine)
		rightWidth := ansi.PrintableRuneWidth(right)
		if rightWidth <= bgLineWidth-pos {
			b.WriteString(ws.render(bgLineWidth - rightWidth - pos))
		}
		b.WriteString(right)
	}

	return b.String()
}

func cutLeft(s string, cutWidth int) string {
	var (
		pos    int
		isAnsi bool
		ab     bytes.Buffer
		b      bytes.Buffer
	)
	for _, c := range s {
		var w int
		if c == ansi.Marker || isAnsi {
			isAnsi = true
			ab.WriteRune(c)
			if ansi.IsTerminator(c) {
				isAnsi = false
				if bytes.HasSuffix(ab.Bytes(), []byte("[0m")) {
					ab.Reset()
				}
			}
		} else {
			w = runewidth.RuneWidth(c)
		}

		if pos >= cutWidth {
			if b.Len() == 0 {
				if ab.Len() > 0 {
					b.Write(ab.Bytes())
				}
				if pos-cutWidth > 1 {
					b.WriteByte(' ')
					continue
				}
			}
			b.WriteRune(c)
		}
		pos += w
	}
	return b.String()
}

func clamp(v, lower, upper int) int {
	return min(max(v, lower), upper)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type whitespace struct {
	style termenv.Style
	chars string
}

// Render whitespaces.
func (w whitespace) render(width int) string {
	if w.chars == "" {
		w.chars = " "
	}

	r := []rune(w.chars)
	j := 0
	b := strings.Builder{}

	// Cycle through runes and print them into the whitespace.
	for i := 0; i < width; {
		b.WriteRune(r[j])
		j++
		if j >= len(r) {
			j = 0
		}
		i += ansi.PrintableRuneWidth(string(r[j]))
	}

	// Fill any extra gaps white spaces. This might be necessary if any runes
	// are more than one cell wide, which could leave a one-rune gap.
	short := width - ansi.PrintableRuneWidth(b.String())
	if short > 0 {
		b.WriteString(strings.Repeat(" ", short))
	}

	return w.style.Styled(b.String())
}
