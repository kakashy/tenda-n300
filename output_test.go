package main

import (
	"os"
	"testing"
	"time"
)

func TestCountAccess(t *testing.T) {
	devices := []Device{
		{Hostname: "a", Access: true},
		{Hostname: "b", Access: false},
		{Hostname: "c", Access: true},
		{Hostname: "d", Access: false},
		{Hostname: "e", Access: true},
	}

	if got := countAccess(devices, true); got != 3 {
		t.Fatalf("expected 3 online, got %d", got)
	}
	if got := countAccess(devices, false); got != 2 {
		t.Fatalf("expected 2 blocked, got %d", got)
	}
}

func TestCountAccessEmpty(t *testing.T) {
	if got := countAccess(nil, true); got != 0 {
		t.Fatalf("expected 0 for nil slice, got %d", got)
	}
	if got := countAccess([]Device{}, true); got != 0 {
		t.Fatalf("expected 0 for empty slice, got %d", got)
	}
}

func TestCountAccessAllSame(t *testing.T) {
	devices := []Device{
		{Hostname: "a", Access: true},
		{Hostname: "b", Access: true},
	}
	if got := countAccess(devices, true); got != 2 {
		t.Fatalf("expected 2 online, got %d", got)
	}
	if got := countAccess(devices, false); got != 0 {
		t.Fatalf("expected 0 blocked, got %d", got)
	}
}

func TestStartStopSpinner(t *testing.T) {
	t.Cleanup(func() { stopSpinner() })

	startSpinner("testing")
	if !spinnerActive() {
		t.Fatal("spinner should be active after startSpinner")
	}

	stopSpinner()
	if spinnerActive() {
		t.Fatal("spinner should not be active after stopSpinner")
	}
}

func TestStopSpinnerWithoutStart(t *testing.T) {
	// Must not panic
	stopSpinner()
	stopSpinner()
	stopSpinner()
}

func TestRestartSpinner(t *testing.T) {
	t.Cleanup(func() { stopSpinner() })

	startSpinner("first")
	stopSpinner()

	startSpinner("second")
	stopSpinner()
}

func TestSpinnerGoroutineExitsOnStop(t *testing.T) {
	t.Cleanup(func() { stopSpinner() })

	startSpinner("leak test")
	stopped := make(chan struct{})
	go func() {
		// Wait a bit then signal the spinner is still our concern
		stopSpinner()
		close(stopped)
	}()

	select {
	case <-stopped:
		// goroutine stopped successfully
	case <-time.After(time.Second):
		t.Fatal("stopSpinner timed out — goroutine may not have exited")
	}
}

func TestHandleSignalsStopsSpinner(t *testing.T) {
	t.Cleanup(func() {
		stopSpinner()
		exitFunc = os.Exit
	})
	stopSpinner()

	exitResult := make(chan int, 1)
	exitFunc = func(code int) { exitResult <- code }

	startSpinner("handler test")
	sigCh := make(chan os.Signal, 1)
	go handleSignals(sigCh)
	sigCh <- os.Interrupt

	select {
	case code := <-exitResult:
		if code != 130 {
			t.Fatalf("expected exit code 130, got %d", code)
		}
	case <-time.After(time.Second):
		t.Fatal("handleSignals did not call exitFunc")
	}

	if spinnerActive() {
		t.Fatal("spinner should not be active after handleSignals processes signal")
	}
}

func TestHandleSignalsIdempotent(t *testing.T) {
	t.Cleanup(func() {
		stopSpinner()
		exitFunc = os.Exit
	})
	stopSpinner()

	exitResult := make(chan int, 2)
	exitFunc = func(code int) { exitResult <- code }

	sigCh := make(chan os.Signal, 1)
	go handleSignals(sigCh)

	sigCh <- os.Interrupt
	sigCh <- os.Interrupt

	select {
	case <-exitResult:
		// first signal handled
	case <-time.After(time.Second):
		t.Fatal("handleSignals did not respond to first signal")
	}

	select {
	case <-exitResult:
		t.Fatal("handleSignals should not respond to second signal (goroutine exited)")
	case <-time.After(200 * time.Millisecond):
		// expected — handler only processes one signal
	}
}
