package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gcla/gowid"
	"github.com/gcla/gowid/widgets/edit"
	"github.com/gcla/gowid/widgets/text"
)

func readOnlyEditor(content string) *edit.Widget {
	return edit.New(edit.Options{Text: content, ReadOnly: true})
}

func spacerCell() gowid.IContainerWidget {
	return &gowid.ContainerWidget{IWidget: text.New(" "), D: gowid.RenderFixed{}}
}

func buttonCell(widget gowid.IWidget) gowid.IContainerWidget {
	return &gowid.ContainerWidget{IWidget: widget, D: gowid.RenderFixed{}}
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
