package native

import (
	"context"
	"fmt"
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
	prevUp    int64
	prevDown  int64
	prevTime  time.Time
	statsPort int
	stop      chan struct{}
}

func newStatsTracker(statsPort int) *statsTracker {
	return &statsTracker{
		statsPort: statsPort,
		prevTime:  time.Now(),
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
	up, down := t.queryXrayStats()

	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	dt := now.Sub(t.prevTime).Seconds()
	if dt < 0.1 {
		dt = 1
	}
	upSpeed := int64(float64(up-t.prevUp) / dt)
	downSpeed := int64(float64(down-t.prevDown) / dt)
	if upSpeed < 0 {
		upSpeed = 0
	}
	if downSpeed < 0 {
		downSpeed = 0
	}

	t.current = domain.StatsResult{
		UpBytes:   up,
		DownBytes: down,
		UpSpeed:   upSpeed,
		DownSpeed: downSpeed,
	}
	t.prevUp = up
	t.prevDown = down
	t.prevTime = now
}

// get returns the latest stats snapshot.
func (t *statsTracker) get() domain.StatsResult {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.current
}

// reset zeroes accumulated counters (called when core restarts).
func (t *statsTracker) reset() {
	t.mu.Lock()
	t.current = domain.StatsResult{}
	t.prevUp = 0
	t.prevDown = 0
	t.prevTime = time.Now()
	t.mu.Unlock()
}

// queryXrayStats calls the in-process xray-core gRPC API and returns cumulative
// up/down bytes for outbound[proxy]. Returns 0,0 on any error.
func (t *statsTracker) queryXrayStats() (up, down int64) {
	server := fmt.Sprintf("127.0.0.1:%d", t.statsPort)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, server, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return 0, 0
	}
	defer conn.Close()

	client := statsService.NewStatsServiceClient(conn)
	resp, err := client.QueryStats(ctx, &statsService.QueryStatsRequest{
		Pattern: "outbound>>>proxy>>>traffic",
		Reset_:  true,
	})
	if err != nil {
		return 0, 0
	}
	return parseStatsResponse(resp)
}

func parseStatsResponse(resp *statsService.QueryStatsResponse) (up, down int64) {
	if resp == nil {
		return 0, 0
	}
	for _, s := range resp.Stat {
		n := s.GetValue()
		if strings.HasSuffix(s.Name, "uplink") {
			up += n
		} else if strings.HasSuffix(s.Name, "downlink") {
			down += n
		}
	}
	return
}
