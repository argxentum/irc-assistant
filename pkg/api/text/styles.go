package text

import "fmt"

const (
	StyleBold          = "\x02"
	StyleItalics       = "\x1D"
	StyleUnderline     = "\x1F"
	StyleStrikethrough = "\x1E"
	StyleMonospace     = "\x11"
	StyleColor         = "\x03"
	StyleReset         = "\x0F"
)

func Bold(s string) string {
	return StyleBold + s + StyleBold
}

func Italics(s string) string {
	return StyleItalics + s + StyleItalics
}

func Underline(s string) string {
	return StyleUnderline + s + StyleUnderline
}

func Strikethrough(s string) string {
	return StyleStrikethrough + s + StyleStrikethrough
}

func Monospace(s string) string {
	return StyleMonospace + s + StyleMonospace
}

func Color(s string, fg, bg int) string {
	return StyleColor + fmt.Sprintf("%02d,%02d", fg, bg) + s + StyleColor
}
