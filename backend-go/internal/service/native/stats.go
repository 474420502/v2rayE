package native

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	statsService "github.com/xtls/xray-core/app/stats/command"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"v2raye/backend-go/internal/domain"
)

// statsTracker periodically queries xray's stats API and exposes the results.
type statsTracker struct {
	mu        sync.RWMutex
	current   domain.StatsResult
	outbound  map[string]domain.RoutingOutboundHit
	prevTime  time.Time
	updatedAt time.Time
	statsPort int
	stop      chan struct{}
}

func newStatsTracker(statsPort int) *statsTracker {
	return &statsTracker{
		statsPort: statsPort,
		prevTime:  time.Now(),
		outbound:  make(map[string]domain.RoutingOutboundHit),
	}
}

// start begins periodic polling every second.
func (t *statsTracker) start() {
	t.stop = make(chan struct{})
	go t.loop()
}

// shutdown stops the polling goroutine.
func (t *statsTracker) shutdown() {
	if t.stop != nil {
		close(t.stop)
	}
}

func (t *statsTracker) loop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.stop:
			return
		case <-ticker.C:
			t.poll()
		}
	}
}

func (t *statsTracker) poll() {
	deltas := t.queryXrayOutboundDeltas()

	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	dt := now.Sub(t.prevTime).Seconds()
	if dt < 0.1 {
		dt = 1
	}
	for key, item := range t.outbound {
		item.UpSpeed = 0
		item.DownSpeed = 0
		t.outbound[key] = item
	}

	for outbound, delta := range deltas {
		item := t.outbound[outbound]
		item.Outbound = outbound
		item.UpBytes += delta.up
		item.DownBytes += delta.down
		item.UpSpeed = int64(float64(delta.up) / dt)
		item.DownSpeed = int64(float64(delta.down) / dt)
		if item.UpSpeed < 0 {
			item.UpSpeed = 0
		}
		if item.DownSpeed < 0 {
			item.DownSpeed = 0
		}
		t.outbound[outbound] = item
	}

	proxy := t.outbound["proxy"]

	t.current = domain.StatsResult{
		UpBytes:   proxy.UpBytes,
		DownBytes: proxy.DownBytes,
		UpSpeed:   proxy.UpSpeed,
		DownSpeed: proxy.DownSpeed,
	}
	t.prevTime = now
	t.updatedAt = now
}

// get returns the latest stats snapshot.
func (t *statsTracker) get() domain.StatsResult {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.current
}

func (t *statsTracker) getRoutingHitStats() domain.RoutingHitStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	items := make([]domain.RoutingOutboundHit, 0, len(t.outbound))
	for _, item := range t.outbound {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Outbound < items[j].Outbound
	})
	updatedAt := ""
	if !t.updatedAt.IsZero() {
		updatedAt = t.updatedAt.UTC().Format(time.RFC3339)
	}
	return domain.RoutingHitStats{
		UpdatedAt: updatedAt,
		Items:     items,
		Note:      "统计维度为 outbound 命中（proxy/direct/block 等），不是单条规则逐条命中计数。",
	}
}

// reset zeroes accumulated counters (called when core restarts).
func (t *statsTracker) reset() {
	t.mu.Lock()
	t.current = domain.StatsResult{}
	t.outbound = make(map[string]domain.RoutingOutboundHit)
	t.prevTime = time.Now()
	t.updatedAt = time.Time{}
	t.mu.Unlock()
}

type trafficDelta struct {
	up   int64
	down int64
}

// queryXrayOutboundDeltas calls xray-core stats API and returns per-outbound
// byte deltas since last poll.
func (t *statsTracker) queryXrayOutboundDeltas() map[string]trafficDelta {
	server := fmt.Sprintf("127.0.0.1:%d", t.statsPort)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, server, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return map[string]trafficDelta{}
	}
	defer conn.Close()

	client := statsService.NewStatsServiceClient(conn)
	resp, err := client.QueryStats(ctx, &statsService.QueryStatsRequest{
		Pattern: "outbound>>>",
		Reset_:  true,
	})
	if err != nil {
		return map[string]trafficDelta{}
	}
	return parseOutboundDeltas(resp)
}

func parseOutboundDeltas(resp *statsService.QueryStatsResponse) map[string]trafficDelta {
	out := make(map[string]trafficDelta)
	if resp == nil {
		return out
	}
	for _, s := range resp.Stat {
		parts := strings.Split(s.Name, ">>>")
		if len(parts) < 4 || parts[0] != "outbound" {
			continue
		}
		outbound := parts[1]
		dir := parts[len(parts)-1]
		delta := out[outbound]
		if dir == "uplink" {
			delta.up += s.GetValue()
		} else if dir == "downlink" {
			delta.down += s.GetValue()
		}
		out[outbound] = delta
	}
	return out
}
