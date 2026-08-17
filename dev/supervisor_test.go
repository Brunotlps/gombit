package dev

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWaitAllTimesOutAfterKill(t *testing.T) {
	t.Parallel()

	var wg sync.WaitGroup
	wg.Add(1)

	start := time.Now()
	err := waitAll(&wg, &sync.Mutex{}, nil, 50*time.Millisecond)
	wg.Done()
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("waitAll() error = nil, want timeout after kill")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("waitAll() error = %q, want timed out", err)
	}
	if elapsed > 2*time.Second {
		t.Fatalf("waitAll() hung for %s, want ~100ms cap", elapsed)
	}
}

func TestWindowsTaskkillArgs(t *testing.T) {
	t.Parallel()

	got := windowsTaskkillArgs(4242, false)
	want := []string{"/T", "/PID", "4242"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("windowsTaskkillArgs(force=false) = %v, want %v", got, want)
	}

	got = windowsTaskkillArgs(4242, true)
	want = []string{"/T", "/F", "/PID", "4242"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("windowsTaskkillArgs(force=true) = %v, want %v", got, want)
	}
}
