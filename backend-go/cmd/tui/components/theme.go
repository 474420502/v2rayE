package components

import "github.com/gdamore/tcell/v2"

// Theme 定义应用配色方案
type Theme struct {
	Primary       tcell.Color
	Secondary     tcell.Color
	Accent        tcell.Color
	Success       tcell.Color
	Error         tcell.Color
	Warning       tcell.Color
	Background    tcell.Color
	Surface       tcell.Color
	Text          tcell.Color
	TextMuted     tcell.Color
	Border        tcell.Color
}

// DefaultTheme 默认主题
var DefaultTheme = Theme{
	Primary:       tcell.ColorTeal,
	Secondary:     tcell.ColorDarkBlue,
	Accent:        tcell.ColorYellow,
	Success:       tcell.ColorGreen,
	Error:         tcell.ColorRed,
	Warning:       tcell.ColorOrange,
	Background:    tcell.ColorBlack,
	Surface:       tcell.ColorDarkGray,
	Text:          tcell.ColorWhite,
	TextMuted:     tcell.ColorDarkGray,
	Border:        tcell.ColorTeal,
}

// ButtonColors 按钮颜色配置
type ButtonColors struct {
	NormalBg       tcell.Color
	NormalFg       tcell.Color
	SelectedBg     tcell.Color
	SelectedFg     tcell.Color
}

// DefaultButtonColors 默认按钮配色
var DefaultButtonColors = ButtonColors{
	NormalBg:    tcell.ColorDarkBlue,
	NormalFg:    tcell.ColorWhite,
	SelectedBg:   tcell.ColorGreen,
	SelectedFg:  tcell.ColorBlack,
}

// NavItemColors 导航项颜色
type NavItemColors struct {
	NormalBg    tcell.Color
	NormalFg    tcell.Color
	SelectedBg  tcell.Color
	SelectedFg  tcell.Color
	HoverBg     tcell.Color
	HoverFg     tcell.Color
}

// DefaultNavItemColors 默认导航配色
var DefaultNavItemColors = NavItemColors{
	NormalBg:   tcell.ColorBlack,
	NormalFg:   tcell.ColorWhite,
	SelectedBg: tcell.ColorTeal,
	SelectedFg: tcell.ColorBlack,
	HoverBg:    tcell.ColorDarkGray,
	HoverFg:    tcell.ColorWhite,
}

// GetColors 返回当前主题配色
func (t *Theme) GetColors() (primary, secondary, accent, bg, fg tcell.Color) {
	return t.Primary, t.Secondary, t.Accent, t.Background, t.Text
}