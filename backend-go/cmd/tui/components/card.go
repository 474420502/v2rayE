package components

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Card 带标题和边框的卡片容器
type Card struct {
	*tview.Flex
	title    string
	content  tview.Primitive
	theme    *Theme
}

// NewCard 创建新的卡片
func NewCard(title string, content tview.Primitive, theme *Theme) *Card {
	if theme == nil {
		theme = &DefaultTheme
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	flex.SetBorder(true)
	flex.SetBorderColor(theme.Border)
	flex.SetBackgroundColor(theme.Background)
	flex.SetTitle(" " + title + " ")
	flex.SetTitleColor(theme.Text)

	card := &Card{
		Flex:    flex,
		title:   title,
		content: content,
		theme:   theme,
	}

	if content != nil {
		flex.AddItem(content, 0, 1, false)
	}

	return card
}

// SetContent 设置卡片内容
func (c *Card) SetContent(content tview.Primitive) {
	c.content = content
	c.Clear()
	if content != nil {
		c.AddItem(content, 0, 1, false)
	}
}

// GetContent 获取卡片内容
func (c *Card) GetContent() tview.Primitive {
	return c.content
}

// SetTitle 设置卡片标题
func (c *Card) SetTitle(title string) {
	c.title = title
	c.Flex.SetTitle(" " + title + " ")
}

// GetTitle 获取卡片标题
func (c *Card) GetTitle() string {
	return c.title
}

// SetBorderColor 设置边框颜色
func (c *Card) SetBorderColor(color tcell.Color) {
	c.Flex.SetBorderColor(color)
}

// SetTitleColor 设置标题颜色
func (c *Card) SetTitleColor(color tcell.Color) {
	c.Flex.SetTitleColor(color)
}

// CardWithContent 创建带内容的卡片
func CardWithContent(title string, content tview.Primitive, theme *Theme) *Card {
	return NewCard(title, content, theme)
}

// SimpleCard 创建简单的卡片（仅标题）
func SimpleCard(title string, theme *Theme) *Card {
	return NewCard(title, nil, theme)
}

// CardGrid 卡片网格容器
type CardGrid struct {
	*tview.Grid
	cards    []*Card
	theme    *Theme
}

// NewCardGrid 创建新的卡片网格
func NewCardGrid(rows, cols int, theme *Theme) *CardGrid {
	if theme == nil {
		theme = &DefaultTheme
	}

	grid := tview.NewGrid().
		SetBorders(false)

	return &CardGrid{
		Grid:  grid,
		cards: make([]*Card, 0),
		theme: theme,
	}
}

// SetRows 设置网格行
func (g *CardGrid) SetRows(rows ...int) *CardGrid {
	g.Grid.SetRows(rows...)
	return g
}

// SetCols 设置网格列
func (g *CardGrid) SetCols(cols ...int) *CardGrid {
	g.Grid.SetColumns(cols...)
	return g
}

// AddCard 添加卡片到网格
func (g *CardGrid) AddCard(card *Card, row, col, rowSpan, colSpan int) {
	g.cards = append(g.cards, card)
	g.Grid.AddItem(card, row, col, rowSpan, colSpan, 0, 0, false)
}

// AddCardWithSizes 添加卡片并指定大小
func (g *CardGrid) AddCardWithSizes(card *Card, row, col int, minHeight, minWidth int) {
	g.cards = append(g.cards, card)
	g.Grid.AddItem(card, row, col, 1, 1, minHeight, minWidth, false)
}

// GetCards 获取所有卡片
func (g *CardGrid) GetCards() []*Card {
	return g.cards
}

// GetCard 获取指定索引的卡片
func (g *CardGrid) GetCard(index int) *Card {
	if index >= 0 && index < len(g.cards) {
		return g.cards[index]
	}
	return nil
}