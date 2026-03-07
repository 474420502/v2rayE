package native

import (
	"bufio"
	"io"
	"strings"
	"sync"
	"time"

	"v2raye/backend-go/internal/domain"
)

const logBufSize = 500

// logBroker captures lines from an io.Reader and fans them out to subscribers.
type logBroker struct {
	mu      sync.RWMutex
	buf     []domain.LogLine // ring buffer of recent lines
	subs    map[int]chan domain.LogLine
	nextID  int
}

func newLogBroker() *logBroker {
	return &logBroker{
		buf:  make([]domain.LogLine, 0, logBufSize),
		subs: make(map[int]chan domain.LogLine),
	}
}

// ingest reads lines from r until EOF and distributes them.
func (b *logBroker) ingest(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 64*1024)
	for scanner.Scan() {
		line := parseLine(scanner.Text())
		b.dispatch(line)
	}
}

func (b *logBroker) dispatch(line domain.LogLine) {
	b.mu.Lock()
	if len(b.buf) >= logBufSize {
		b.buf = b.buf[1:]
	}
	b.buf = append(b.buf, line)
	subs := make([]chan domain.LogLine, 0, len(b.subs))
	for _, ch := range b.subs {
		subs = append(subs, ch)
	}
	b.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- line:
		default:
		}
	}
}

// subscribe returns a channel that receives new log lines and a cancel func.
func (b *logBroker) subscribe() (<-chan domain.LogLine, func()) {
	ch := make(chan domain.LogLine, 64)

	b.mu.Lock()
	id := b.nextID
	b.nextID++
	b.subs[id] = ch
	// send buffered lines immediately so the client sees recent history
	recent := make([]domain.LogLine, len(b.buf))
	copy(recent, b.buf)
	b.mu.Unlock()

	go func() {
		for _, l := range recent {
			select {
			case ch <- l:
			default:
			}
		}
	}()

	cancel := func() {
		b.mu.Lock()
		delete(b.subs, id)
		b.mu.Unlock()
		close(ch)
	}
	return ch, cancel
}

// clear resets the buffer (called when core stops).
func (b *logBroker) clear() {
	b.mu.Lock()
	b.buf = b.buf[:0]
	b.mu.Unlock()
}

// parseLine converts a raw xray log line into a structured LogLine.
// Xray log format: "2006/01/02 15:04:05 [warning] ..."
func parseLine(raw string) domain.LogLine {
	raw = strings.TrimRight(raw, "\r")
	ts := time.Now().UTC().Format(time.RFC3339)
	level := "info"
	msg := raw

	// Attempt to parse timestamp + level prefix.
	// e.g. "2006/01/02 15:04:05 [warning] core: ..."
	if len(raw) > 20 && raw[19] == ' ' {
		ts = raw[:19]
		rest := strings.TrimSpace(raw[20:])
		if strings.HasPrefix(rest, "[") {
			end := strings.Index(rest, "]")
			if end != -1 {
				level = strings.ToLower(rest[1:end])
				msg = strings.TrimSpace(rest[end+1:])
			}
		} else {
			msg = rest
		}
	}

	return domain.LogLine{Timestamp: ts, Level: level, Message: msg}
}
