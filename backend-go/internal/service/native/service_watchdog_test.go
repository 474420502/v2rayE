package native

import "testing"

func TestWatchdogRestartPlanRecognizesDegradedCore(t *testing.T) {
	t.Parallel()

	svc := &Service{running: true, degraded: true}
	reason, restartWhileRunning := svc.watchdogRestartPlanLocked(true)
	if reason != "core running in degraded mode" {
		t.Fatalf("watchdogRestartPlanLocked() reason = %q, want degraded restart", reason)
	}
	if !restartWhileRunning {
		t.Fatalf("watchdogRestartPlanLocked() restartWhileRunning = false, want true")
	}
}

func TestWatchdogRestartPlanRecognizesUnexpectedExit(t *testing.T) {
	t.Parallel()

	svc := &Service{}
	reason, restartWhileRunning := svc.watchdogRestartPlanLocked(true)
	if reason != "core exited unexpectedly" {
		t.Fatalf("watchdogRestartPlanLocked() reason = %q, want unexpected exit", reason)
	}
	if restartWhileRunning {
		t.Fatalf("watchdogRestartPlanLocked() restartWhileRunning = true, want false")
	}
}
