package components

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NavItem 导航项
type NavItem struct {
	Key      string
	Label    string
	Shortcut rune
}

// Sidebar 侧边栏导航组件
type Sidebar struct {
	*tview.Flex
	items      []NavItem
	selected   int
	onSelect   func(string)
	buttons    []*tview.Button
	theme      *Theme
	compact    bool
	visible    bool   // 是否可见
}

// NewSidebar 创建新的侧边栏
func NewSidebar(items []NavItem, onSelect func(string), theme *Theme) *Sidebar {
	if theme == nil {
		theme = &DefaultTheme
	}
	s := &Sidebar{
		Flex:     tview.NewFlex().SetDirection(tview.FlexRow),
		items:    items,
		selected: 0,
		onSelect: onSelect,
		theme:    theme,
	}

	s.SetBackgroundColor(theme.Surface)
	s.build()

	return s
}

// SetItems 更新导航项并重建侧边栏内容。
func (s *Sidebar) SetItems(items []NavItem) {
	s.items = append(s.items[:0], items...)
	if s.items == nil {
		s.items = make([]NavItem, 0)
	}
	if s.selected >= len(s.items) {
		s.selected = 0
	}
	s.build()
}

// SetCompact 设置紧凑模式。
func (s *Sidebar) SetCompact(compact bool) {
	if s.compact != compact {
		s.compact = compact
		s.build()
	}
}

// SetVisible 设置侧边栏是否可见
func (s *Sidebar) SetVisible(visible bool) {
	if s.visible != visible {
		s.visible = visible
		s.build()
	}
}

// IsVisible 获取侧边栏是否可见
func (s *Sidebar) IsVisible() bool {
	return s.visible
}

// build 构建侧边栏内容
func (s *Sidebar) build() {
	s.Clear()

	// 如果不可见，只保留一个占位符
	if !s.visible {
		s.SetBorder(false)
		s.AddItem(tview.NewBox(), 0, 1, false)
		return
	}

	s.SetBorder(true)

	// 根据是否为紧凑模式设置标题
	if s.compact {
		s.SetTitle(" ≡ ")
	} else {
		s.SetTitle(" Menu ")
	}
	s.SetBorderColor(s.theme.Border)

	// 添加标题
	title := tview.NewTextView()
	if s.compact {
		title.SetText("[::b]E")
	} else {
		title.SetText("[::b]v2rayE")
	}
	title.SetTextAlign(tview.AlignCenter)
	title.SetBackgroundColor(s.theme.Primary)
	title.SetTextColor(tcell.ColorBlack)
	title.SetBorderPadding(0, 0, 1, 1)
	s.AddItem(title, 1, 0, false)

	// 分隔线
	s.AddItem(tview.NewBox().SetBackgroundColor(s.theme.Border), 1, 0, false)

	// 添加导航项
	s.buttons = make([]*tview.Button, len(s.items))
	for i, item := range s.items {
		item := item
		var btnLabel string
		if s.compact {
			// 紧凑模式只显示快捷键
			btnLabel = fmt.Sprintf("%c", item.Shortcut)
		} else {
			btnLabel = fmt.Sprintf("%c %s", item.Shortcut, item.Label)
		}
		btn := tview.NewButton(btnLabel)
		btn.SetSelectedFunc(func() {
			s.Select(i)
			if s.onSelect != nil {
				s.onSelect(item.Key)
			}
		})

		if i == s.selected {
			btn.SetBackgroundColor(s.theme.Primary)
			btn.SetLabelColor(s.theme.Background)
		} else {
			btn.SetBackgroundColor(s.theme.Background)
			btn.SetLabelColor(s.theme.Text)
		}

		btn.SetBorderPadding(0, 0, 0, 0)
		s.buttons[i] = btn
		s.AddItem(btn, 1, 0, false)
	}

	// 添加底部空白
	s.AddItem(tview.NewBox(), 0, 1, false)
}

// Select 选择指定索引的导航项
func (s *Sidebar) Select(index int) {
	if index < 0 || index >= len(s.items) {
		return
	}

	oldSelected := s.selected
	s.selected = index

	// 更新按钮样式
	if oldSelected < len(s.buttons) {
		s.buttons[oldSelected].SetBackgroundColor(s.theme.Background)
		s.buttons[oldSelected].SetLabelColor(s.theme.Text)
	}

	if index < len(s.buttons) {
		s.buttons[index].SetBackgroundColor(s.theme.Primary)
		s.buttons[index].SetLabelColor(s.theme.Background)
	}
}

// SetSelectedKey 通过 key 设置选中的导航项
func (s *Sidebar) SetSelectedKey(key string) {
	for i, item := range s.items {
		if item.Key == key {
			s.Select(i)
			return
		}
	}
}

// GetSelectedKey 获取当前选中的 key
func (s *Sidebar) GetSelectedKey() string {
	if s.selected >= 0 && s.selected < len(s.items) {
		return s.items[s.selected].Key
	}
	return ""
}

// GetSelectedIndex 获取当前选中的索引
func (s *Sidebar) GetSelectedIndex() int {
	return s.selected
}

// SetOnSelect 设置选择回调
func (s *Sidebar) SetOnSelect(onSelect func(string)) {
	s.onSelect = onSelect
}

// GetAllButtons 获取所有按钮用于焦点管理
func (s *Sidebar) GetAllButtons() []*tview.Button {
	return s.buttons
}

// GetFocusables 获取所有可聚焦元素
func (s *Sidebar) GetFocusables() []tview.Primitive {
	result := make([]tview.Primitive, 0, len(s.buttons))
	for _, btn := range s.buttons {
		if btn != nil {
			result = append(result, btn)
		}
	}
	return result
}
