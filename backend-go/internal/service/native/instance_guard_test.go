package native

import (
	"errors"
	"testing"
)

func TestIsBackendServerCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"systemd unit", []string{"/usr/lib/v2raye/v2raye", "--server", "--api-addr", "0.0.0.0:18000", "--data-dir", "/opt/v2rayE"}, true},
		{"dev server", []string{"/home/u/workspace/v2rayE/v2raye", "--server"}, true},
		{"backend-api binary", []string{"/usr/bin/backend-api"}, true},
		{"tui client", []string{"/usr/bin/v2raye"}, false},
		{"tui client with base url", []string{"/usr/bin/v2raye", "--base-url", "http://127.0.0.1:18000"}, false},
		{"unrelated process", []string{"/usr/sbin/nginx", "-g", "daemon off;"}, false},
		{"empty command line", nil, false},
	}
	for _, tc := range tests {
		if got := isBackendServerCommand(tc.args); got != tc.want {
			t.Errorf("%s: isBackendServerCommand(%q) = %t, want %t", tc.name, tc.args, got, tc.want)
		}
	}
}

func TestScanForOtherServerInstance(t *testing.T) {
	commands := map[int][]string{
		1:  {"/sbin/init"},
		42: {"/usr/lib/v2raye/v2raye", "--server", "--data-dir", "/opt/v2rayE"},
		43: {"/usr/bin/v2raye"}, // TUI client: must not count as an owner
		44: {"/usr/bin/backend-api"},
	}
	readCmdline := func(pid int) ([]string, error) {
		if args, ok := commands[pid]; ok {
			return args, nil
		}
		return nil, errors.New("no such process")
	}

	if !scanForOtherServerInstance([]int{1, 42}, 999, readCmdline) {
		t.Fatal("expected to detect a running server instance")
	}
	if !scanForOtherServerInstance([]int{44}, 999, readCmdline) {
		t.Fatal("expected to detect the backend-api server binary")
	}
	if scanForOtherServerInstance([]int{1, 43}, 999, readCmdline) {
		t.Fatal("a TUI client must not be treated as a server instance")
	}
	if scanForOtherServerInstance([]int{42}, 42, readCmdline) {
		t.Fatal("the current process must be ignored")
	}
	if scanForOtherServerInstance([]int{0, -1, 777}, 999, readCmdline) {
		t.Fatal("unreadable or invalid pids must be skipped")
	}
}
