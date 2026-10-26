package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPerformanceCPUProfileBoundAndExclusiveOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.pprof")
	profile, err := startPerformanceCPUProfile(path, 30*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer profile.Close()
	if _, err := startPerformanceCPUProfile(path, time.Second); err == nil {
		t.Fatal("existing profile accepted")
	}
	select {
	case <-profile.done:
	case <-time.After(3 * time.Second):
		t.Fatal("profile timer did not stop the writer")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		t.Fatalf("profile writer did not finish: %v, %d bytes", err, len(data))
	}
	if err := profile.Close(); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if string(data) != string(after) {
		t.Fatal("repeated close modified the completed profile")
	}
}

func TestPerformanceCPUProfileEarlyClose(t *testing.T) {
	profile, err := startPerformanceCPUProfile(filepath.Join(t.TempDir(), "cpu.pprof"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := profile.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-profile.done:
	default:
		t.Fatal("early close left the profile active")
	}
}
