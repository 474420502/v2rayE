package launcher

import (
	"context"
	"testing"
	"time"

	"v2raye/backend-go/internal/domain"
)

type bootRestoreStub struct {
	network []domain.AvailabilityResult
	starts  []domain.CoreStatus

	networkCalls int
	startCalls   int
}

func (s *bootRestoreStub) NetworkAvailability() domain.AvailabilityResult {
	result := domain.AvailabilityResult{}
	if len(s.network) != 0 {
		index := s.networkCalls
		if index >= len(s.network) {
			index = len(s.network) - 1
		}
		result = s.network[index]
	}
	s.networkCalls++
	return result
}

func (s *bootRestoreStub) StartCore() domain.CoreStatus {
	result := domain.CoreStatus{}
	if len(s.starts) != 0 {
		index := s.startCalls
		if index >= len(s.starts) {
			index = len(s.starts) - 1
		}
		result = s.starts[index]
	}
	s.startCalls++
	return result
}

func TestWaitForBootNetworkReadySucceedsAfterRetries(t *testing.T) {
	t.Parallel()

	stub := &bootRestoreStub{
		network: []domain.AvailabilityResult{
			{Available: false, Message: "booting"},
			{Available: false, Message: "dhcp"},
			{Available: true},
		},
	}

	err := waitForBootNetworkReady(context.Background(), stub, 50*time.Millisecond, time.Millisecond)
	if err != nil {
		t.Fatalf("waitForBootNetworkReady() error = %v", err)
	}
	if stub.networkCalls < 3 {
		t.Fatalf("expected at least 3 probes, got %d", stub.networkCalls)
	}
}

func TestRestoreCoreOnBootRetriesUntilRunning(t *testing.T) {
	t.Parallel()

	stub := &bootRestoreStub{
		network: []domain.AvailabilityResult{{Available: true}},
		starts: []domain.CoreStatus{
			{Running: false, Error: "network warming up"},
			{Running: true, Degraded: true, State: "degraded", Error: "tun takeover failed"},
			{Running: true, CurrentProfileID: "p1"},
		},
	}

	restoreCoreOnBoot(context.Background(), stub, bootRestoreOptions{
		initialNetworkWait:  20 * time.Millisecond,
		networkPollInterval: time.Millisecond,
		maxStartAttempts:    4,
		retryBackoffBase:    time.Millisecond,
		maxRetryBackoff:     2 * time.Millisecond,
	})

	if stub.startCalls != 3 {
		t.Fatalf("expected 3 StartCore calls, got %d", stub.startCalls)
	}
}

func TestRestoreCoreOnBootRetriesWhenCoreStartsDegraded(t *testing.T) {
	t.Parallel()

	stub := &bootRestoreStub{
		network: []domain.AvailabilityResult{{Available: true}},
		starts: []domain.CoreStatus{
			{Running: true, Degraded: true, State: "degraded", Error: "tun takeover failed"},
			{Running: true, CurrentProfileID: "p1"},
		},
	}

	restoreCoreOnBoot(context.Background(), stub, bootRestoreOptions{
		initialNetworkWait:  20 * time.Millisecond,
		networkPollInterval: time.Millisecond,
		maxStartAttempts:    3,
		retryBackoffBase:    time.Millisecond,
		maxRetryBackoff:     time.Millisecond,
	})

	if stub.startCalls != 2 {
		t.Fatalf("expected degraded boot restore to retry, got %d StartCore calls", stub.startCalls)
	}
}

func TestRestoreCoreOnBootStopsAfterMaxAttempts(t *testing.T) {
	t.Parallel()

	stub := &bootRestoreStub{
		network: []domain.AvailabilityResult{{Available: true}},
		starts:  []domain.CoreStatus{{Running: false, Error: "still failing"}},
	}

	restoreCoreOnBoot(context.Background(), stub, bootRestoreOptions{
		initialNetworkWait:  20 * time.Millisecond,
		networkPollInterval: time.Millisecond,
		maxStartAttempts:    2,
		retryBackoffBase:    time.Millisecond,
		maxRetryBackoff:     time.Millisecond,
	})

	if stub.startCalls != 2 {
		t.Fatalf("expected 2 StartCore calls, got %d", stub.startCalls)
	}
}
