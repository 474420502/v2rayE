package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const narrowLayoutBreakpoint = 130

type textWidget struct {
	*tview.TextView
}

func newTextWidget(content string) *textWidget {
	widget := &textWidget{TextView: tview.NewTextView()}
	widget.SetDynamicColors(true)
	widget.SetScrollable(true)
	widget.SetWrap(true)
	widget.TextView.SetText(content)
	widget.SetBorderPadding(0, 0, 1, 1)
	widget.SetTextColor(tcell.ColorWhite)
	return widget
}

func (w *textWidget) SetText(value string, _ *tview.Application) {
	w.TextView.SetText(value)
}

type inputWidget struct {
	*tview.InputField
}

func newInputWidget(label string, onChange func(string)) *inputWidget {
	widget := &inputWidget{InputField: tview.NewInputField()}
	widget.SetLabel(label)
	widget.SetFieldWidth(0)
	widget.SetLabelColor(tcell.ColorLightGray)
	widget.SetFieldTextColor(tcell.ColorWhite)
	widget.SetFieldBackgroundColor(tcell.ColorBlack)
	widget.SetPlaceholderTextColor(tcell.ColorDarkGray)
	widget.SetChangedFunc(onChange)
	return widget
}

func (w *inputWidget) SetText(value string, _ *tview.Application) {
	w.InputField.SetText(value)
}

func (w *inputWidget) Text() string {
	return w.GetText()
}

type textSetter interface {
	SetText(string, *tview.Application)
}

type builtPage struct {
	root       tview.Primitive
	focusables []tview.Primitive
}

func readOnlyEditor(content string) *textWidget {
	return newTextWidget(content)
}

func newMutedText(content string) *tview.TextView {
	widget := tview.NewTextView()
	widget.SetDynamicColors(true)
	widget.SetTextColor(tcell.ColorDarkGray)
	widget.SetText(content)
	return widget
}

func newListWidget() *tview.List {
	list := tview.NewList()
	list.ShowSecondaryText(false)
	list.SetMainTextColor(tcell.ColorWhite)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.ColorYellow)
	list.SetHighlightFullLine(true)
	return list
}

func verticalSpacer(size int) tview.Primitive {
	return tview.NewBox().SetBackgroundColor(tcell.ColorBlack)
}

func horizontalSpacer(size int) tview.Primitive {
	return tview.NewBox().SetBackgroundColor(tcell.ColorBlack)
}

func wrapPanel(title string, primitive tview.Primitive) *tview.Flex {
	panel := tview.NewFlex().SetDirection(tview.FlexRow)
	panel.AddItem(primitive, 0, 1, false)
	panel.SetBorder(true)
	panel.SetTitle(" " + title + " ")
	return panel
}

func buttonWidth(label string) int {
	return len([]rune(label)) + 4
}

func buttonRow(buttons ...*tview.Button) *tview.Flex {
	row := tview.NewFlex().SetDirection(tview.FlexColumn)
	for idx, btn := range buttons {
		row.AddItem(btn, buttonWidth(btn.GetLabel()), 0, false)
		if idx != len(buttons)-1 {
			row.AddItem(horizontalSpacer(1), 1, 0, false)
		}
	}
	row.AddItem(horizontalSpacer(1), 0, 1, false)
	return row
}

func buttonColumn(buttons ...*tview.Button) *tview.Flex {
	col := tview.NewFlex().SetDirection(tview.FlexRow)
	for idx, btn := range buttons {
		col.AddItem(btn, 1, 0, false)
		if idx != len(buttons)-1 {
			col.AddItem(verticalSpacer(1), 1, 0, false)
		}
	}
	return col
}

func inputRow(left, right tview.Primitive, stacked bool, leftWeight, rightWeight int) tview.Primitive {
	if stacked {
		col := tview.NewFlex().SetDirection(tview.FlexRow)
		col.AddItem(left, 1, 0, false)
		col.AddItem(verticalSpacer(1), 1, 0, false)
		col.AddItem(right, 1, 0, false)
		return col
	}
	row := tview.NewFlex().SetDirection(tview.FlexColumn)
	row.AddItem(left, 0, leftWeight, false)
	row.AddItem(horizontalSpacer(1), 1, 0, false)
	row.AddItem(right, 0, rightWeight, false)
	return row
}

func splitContent(stacked bool, first, second tview.Primitive, firstWeight, secondWeight int) tview.Primitive {
	if stacked {
		col := tview.NewFlex().SetDirection(tview.FlexRow)
		col.AddItem(first, 0, firstWeight, false)
		col.AddItem(verticalSpacer(1), 1, 0, false)
		col.AddItem(second, 0, secondWeight, false)
		return col
	}
	row := tview.NewFlex().SetDirection(tview.FlexColumn)
	row.AddItem(first, 0, firstWeight, false)
	row.AddItem(horizontalSpacer(1), 1, 0, false)
	row.AddItem(second, 0, secondWeight, false)
	return row
}

func actionBlockHeight(stacked bool, buttonCount int) int {
	if !stacked {
		return 1
	}
	if buttonCount <= 0 {
		return 1
	}
	// Stacked buttons are rendered as button + spacer (+ button...), so reserve exact rows.
	return buttonCount*2 - 1
}

func dualItemRowHeight(stacked bool) int {
	if stacked {
		return 3
	}
	return 1
}

func panelHeight(contentHeight int) int {
	if contentHeight < 1 {
		contentHeight = 1
	}
	// Account for top and bottom borders.
	return contentHeight + 2
}

func buildPageLayout(headerTitle string, header tview.Primitive, headerContentHeight int, body tview.Primitive) tview.Primitive {
	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.AddItem(wrapPanel(headerTitle, header), panelHeight(headerContentHeight), 0, false)
	root.AddItem(verticalSpacer(1), 1, 0, false)
	root.AddItem(body, 0, 1, false)
	return root
}

func joinFocusables(groups ...[]tview.Primitive) []tview.Primitive {
	var out []tview.Primitive
	for _, group := range groups {
		out = append(out, group...)
	}
	return out
}

func buttonsToFocusables(buttons ...*tview.Button) []tview.Primitive {
	items := make([]tview.Primitive, 0, len(buttons))
	for _, btn := range buttons {
		items = append(items, btn)
	}
	return items
}

func primitivesToFocusables(primitives ...tview.Primitive) []tview.Primitive {
	items := make([]tview.Primitive, 0, len(primitives))
	for _, primitive := range primitives {
		if primitive != nil {
			items = append(items, primitive)
		}
	}
	return items
}

func (a *tuiApp) withSuspendedFieldTracking(fn func()) {
	a.suspendFieldTracking.Store(true)
	defer a.suspendFieldTracking.Store(false)
	fn()
}

func (a *tuiApp) fieldTrackingSuspended() bool {
	return a.suspendFieldTracking.Load()
}

func (a *tuiApp) focusIsInput() bool {
	if a.app == nil {
		return false
	}
	_, ok := a.app.GetFocus().(*tview.InputField)
	return ok
}

func appendBounded(lines []string, next string, max int) []string {
	lines = append(lines, next)
	if len(lines) <= max {
		return lines
	}
	return append([]string(nil), lines[len(lines)-max:]...)
}

func humanBytes(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%dB", value)
	}
	div, exp := int64(unit), 0
	for n := value / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(value)/float64(div), "KMGTPE"[exp])
}

func humanDurationSeconds(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}
	duration := time.Duration(seconds) * time.Second
	if duration < time.Minute {
		return duration.String()
	}
	hours := int(duration / time.Hour)
	minutes := int((duration % time.Hour) / time.Minute)
	secs := int((duration % time.Minute) / time.Second)
	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, secs)
	}
	return fmt.Sprintf("%dm%ds", minutes, secs)
}

func sleepContext(ctx context.Context, delay time.Duration) bool {
	if delay <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func nextBackoffDelay(delay time.Duration) time.Duration {
	if delay <= 0 {
		return time.Second
	}
	delay *= 2
	if delay > 30*time.Second {
		return 30 * time.Second
	}
	return delay
}

func stringValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	if raw, ok := values[key]; ok {
		switch typed := raw.(type) {
		case string:
			return typed
		case fmt.Stringer:
			return typed.String()
		}
	}
	return ""
}

func intValue(values map[string]any, key string) int {
	if values == nil {
		return 0
	}
	if raw, ok := values[key]; ok {
		switch typed := raw.(type) {
		case int:
			return typed
		case int64:
			return int(typed)
		case float64:
			return int(typed)
		case jsonNumber:
			parsed, _ := strconv.Atoi(string(typed))
			return parsed
		case string:
			parsed, _ := strconv.Atoi(strings.TrimSpace(typed))
			return parsed
		}
	}
	return 0
}

type jsonNumber string

func boolValue(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	if raw, ok := values[key]; ok {
		switch typed := raw.(type) {
		case bool:
			return typed
		case string:
			return strings.EqualFold(typed, "true")
		}
	}
	return false
}

func mustAtoiDefault(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func formatEvent(msg EventMessage) string {
	return fmt.Sprintf("%s %v", msg.Event, msg.Data)
}

func errorString(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

func appendBoundedLogLines(lines []LogLine, next LogLine, max int) []LogLine {
	lines = append(lines, next)
	if len(lines) <= max {
		return lines
	}
	return append([]LogLine(nil), lines[len(lines)-max:]...)
}

func formatHighlightedLogLine(line LogLine, query string) string {
	level := logLevelLabel(line.Level)
	source := logSourceLabel(line.Source)
	message := highlightLogMessage(line.Message, query)
	return fmt.Sprintf("%s [%s] [%s] %s", line.Timestamp, level, source, message)
}

func logLevelLabel(level string) string {
	switch normalizeLogLevel(level) {
	case "error":
		return "ERR"
	case "warning":
		return "WRN"
	case "debug":
		return "DBG"
	default:
		return "INF"
	}
}

func logSourceLabel(source string) string {
	switch normalizeLogSource(source) {
	case "app":
		return "APP"
	case "xray-core":
		return "CORE"
	default:
		return "UNKN"
	}
}

func highlightLogMessage(message, query string) string {
	keywords := []string{"error", "failed", "timeout", "refused", "connect", "start", "stop"}
	out := message
	for _, keyword := range keywords {
		out = emphasizeKeyword(out, keyword)
	}
	if strings.TrimSpace(query) != "" {
		out = emphasizeKeyword(out, query)
	}
	return out
}

func emphasizeKeyword(message, keyword string) string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return message
	}
	lowerMessage := strings.ToLower(message)
	lowerKeyword := strings.ToLower(keyword)
	var builder strings.Builder
	index := 0
	for {
		match := strings.Index(lowerMessage[index:], lowerKeyword)
		if match == -1 {
			builder.WriteString(message[index:])
			break
		}
		match += index
		builder.WriteString(message[index:match])
		builder.WriteString("[[")
		builder.WriteString(message[match : match+len(keyword)])
		builder.WriteString("]]")
		index = match + len(keyword)
	}
	return builder.String()
}

func (a *tuiApp) useStackedLayout() bool {
	widthBreakpoint := narrowLayoutBreakpoint
	heightBreakpoint := 38
	if a.compactMode {
		widthBreakpoint -= 8
		heightBreakpoint -= 2
	} else {
		widthBreakpoint += 8
		heightBreakpoint += 2
	}
	if a.viewportCols > 0 && a.viewportCols < widthBreakpoint {
		return true
	}
	return a.viewportRows > 0 && a.viewportRows < heightBreakpoint
}

func (a *tuiApp) compactModeLabel() string {
	if a.compactMode {
		return "compact"
	}
	return "comfortable"
}

func (a *tuiApp) viewportWarning() string {
	if a.viewportCols > 0 && a.viewportCols < 90 {
		return "viewport narrow"
	}
	if a.viewportRows > 0 && a.viewportRows < 26 {
		return "viewport short"
	}
	return ""
}

func parseBoolText(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func splitCSV(value string) []any {
	parts := strings.Split(value, ",")
	items := make([]any, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		items = append(items, trimmed)
	}
	return items
}

func splitCommaStrings(value string) []string {
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		items = append(items, trimmed)
	}
	return items
}

func toStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			s, ok := item.(string)
			if !ok {
				continue
			}
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
}

func fitSingleLineToWidth(text string, cols int) string {
	if cols <= 0 {
		return text
	}
	runes := []rune(text)
	if len(runes) > cols {
		return string(runes[:cols])
	}
	if len(runes) < cols {
		return text + strings.Repeat(" ", cols-len(runes))
	}
	return text
}
