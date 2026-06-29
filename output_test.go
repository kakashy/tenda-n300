package main

import (
	"testing"
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
