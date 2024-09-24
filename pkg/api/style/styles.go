package style

import "fmt"

const (
	StyleBold          = "\x02"
	StyleItalics       = "\x1D"
	StyleUnderline     = "\x1F"
	StyleStrikethrough = "\x1E"
	StyleMonospace     = "\x11"
	StyleColor         = "\x03"
	StyleReset         = "\x0F"

	ColorWhite   = 0
	ColorBlack   = 1
	ColorBlue    = 2
	ColorGreen   = 3
	ColorRed     = 4
	ColorBrown   = 5
	ColorPurple  = 6
	ColorOrange  = 7
	ColorYellow  = 8
	ColorLime    = 9
	ColorTeal    = 10
	ColorCyan    = 11
	ColorRoyal   = 12
	ColorPink    = 13
	ColorGrey    = 14
	ColorSilver  = 15
	ColorDefault = 99
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

func ColorForeground(s string, fg int) string {
	return StyleColor + fmt.Sprintf("%02d", fg) + s + StyleColor
}

func ColorForegroundBackground(s string, fg, bg int) string {
	return StyleColor + fmt.Sprintf("%02d,%02d", fg, bg) + s + StyleColor
}
