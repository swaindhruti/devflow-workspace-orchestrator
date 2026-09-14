// Package banner renders short words as large block-letter art using a
// small 5x7 dot-matrix bitmap font, for DevFlow's splash screen.
//
// The font only defines the glyphs DevFlow's own banner text needs — D,
// E, V, F, L, O, W, and a blank space — rather than the full alphabet:
// each glyph was hand-verified by actually rendering it and reading the
// result before being committed here, and extending the set to letters
// nothing currently uses would mean adding unverified shapes on faith.
// Add a glyph (and verify it renders correctly) before calling Render
// with a word that needs one this package doesn't yet have.
package banner

import "strings"

// glyphHeight is the number of rows every glyph occupies.
const glyphHeight = 7

// glyphWidth is the number of columns every glyph occupies.
const glyphWidth = 5

// blankGlyph is substituted for any rune Render is asked to draw that
// isn't in glyphs, so an unsupported character degrades to a rendering
// gap instead of a panic.
var blankGlyph = [glyphHeight]string{
	"00000", "00000", "00000", "00000", "00000", "00000", "00000",
}

// glyphs maps each supported rune to its bitmap: glyphHeight strings of
// glyphWidth characters each, where '1' marks a filled cell and '0'
// marks an empty one.
var glyphs = map[rune][glyphHeight]string{
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'V': {"10001", "10001", "10001", "10001", "10001", "01010", "00100"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'L': {"10000", "10000", "10000", "10000", "10000", "10000", "11111"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'W': {"10001", "10001", "10001", "10101", "10101", "11011", "10001"},
	' ': blankGlyph,
}

// Render draws word as multi-line block-letter art, one glyph per rune,
// separated by a single-cell gap.
//
// Parameters:
//   - word: the text to render, using only runes present in glyphs (see
//     the package doc comment). Any other rune renders as blankGlyph
//     rather than panicking.
//   - filled: the string drawn for each filled cell (e.g. "██" for a
//     bold look, "█" for a more compact one).
//   - empty: the string drawn for each empty cell and for the one-cell
//     gap between glyphs. Should visually match filled's width so rows
//     stay aligned (e.g. pair "██" with "  ").
//
// Returns the rendered block art as a single string, glyphHeight lines
// joined by "\n", every line the same length.
func Render(word, filled, empty string) string {
	runes := []rune(word)

	rows := make([]strings.Builder, glyphHeight)
	for i, r := range runes {
		g, ok := glyphs[r]
		if !ok {
			g = blankGlyph
		}

		for row := 0; row < glyphHeight; row++ {
			for _, bit := range g[row] {
				if bit == '1' {
					rows[row].WriteString(filled)
				} else {
					rows[row].WriteString(empty)
				}
			}
			if i != len(runes)-1 {
				rows[row].WriteString(empty)
			}
		}
	}

	lines := make([]string, glyphHeight)
	for i := range rows {
		lines[i] = rows[i].String()
	}

	return strings.Join(lines, "\n")
}
