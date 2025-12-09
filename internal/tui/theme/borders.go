package theme

import "github.com/charmbracelet/lipgloss"

// BorderVariant enumerates the reusable border shapes.
type BorderVariant string

const (
	BorderNormal       BorderVariant = "normal"
	BorderRounded      BorderVariant = "rounded"
	BorderThick        BorderVariant = "thick"
	BorderDouble       BorderVariant = "double"
	BorderBlock        BorderVariant = "block"
	BorderOuterHalf    BorderVariant = "outer-half"
	BorderInnerHalf    BorderVariant = "inner-half"
	BorderHidden       BorderVariant = "hidden"
	BorderMarkdown     BorderVariant = "markdown"
	BorderASCII        BorderVariant = "ascii"
	DefaultCardBorder                = BorderNormal
	DefaultModalBorder               = BorderRounded
)

// BorderFor returns the lipgloss border definition for the variant.
func BorderFor(variant BorderVariant) lipgloss.Border {
	switch variant {
	case BorderRounded:
		return lipgloss.RoundedBorder()
	case BorderThick:
		return lipgloss.Border{
			Top:         "━",
			Bottom:      "━",
			Left:        "┃",
			Right:       "┃",
			TopLeft:     "┏",
			TopRight:    "┓",
			BottomLeft:  "┗",
			BottomRight: "┛",
		}
	case BorderDouble:
		return lipgloss.Border{
			Top:         "═",
			Bottom:      "═",
			Left:        "║",
			Right:       "║",
			TopLeft:     "╔",
			TopRight:    "╗",
			BottomLeft:  "╚",
			BottomRight: "╝",
		}
	case BorderBlock:
		return lipgloss.Border{
			Top:         "█",
			Bottom:      "█",
			Left:        "█",
			Right:       "█",
			TopLeft:     "█",
			TopRight:    "█",
			BottomLeft:  "█",
			BottomRight: "█",
		}
	case BorderOuterHalf:
		return lipgloss.Border{
			Top:         "▀",
			Bottom:      "▄",
			Left:        "▌",
			Right:       "▐",
			TopLeft:     "▛",
			TopRight:    "▜",
			BottomLeft:  "▙",
			BottomRight: "▟",
		}
	case BorderInnerHalf:
		return lipgloss.Border{
			Top:         "▄",
			Bottom:      "▀",
			Left:        "▐",
			Right:       "▌",
			TopLeft:     "▗",
			TopRight:    "▖",
			BottomLeft:  "▝",
			BottomRight: "▘",
		}
	case BorderHidden:
		return lipgloss.HiddenBorder()
	case BorderMarkdown:
		return lipgloss.Border{
			Top:         "",
			Bottom:      "",
			Left:        "|",
			Right:       "|",
			TopLeft:     "|",
			TopRight:    "|",
			BottomLeft:  "|",
			BottomRight: "|",
		}
	case BorderASCII:
		return lipgloss.Border{
			Top:         "-",
			Bottom:      "-",
			Left:        "|",
			Right:       "|",
			TopLeft:     "+",
			TopRight:    "+",
			BottomLeft:  "+",
			BottomRight: "+",
		}
	default:
		return lipgloss.NormalBorder()
	}
}
