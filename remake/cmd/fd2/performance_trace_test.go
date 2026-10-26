package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPerformanceTraceWindowsAndExclusiveOutput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "performance.jsonl")
	trace, err := openPerformanceTrace(nil, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := openPerformanceTrace(nil, path); err == nil {
		t.Fatal("existing report accepted")
	}
	now := trace.started.Add(2 * time.Second)
	trace.updates, trace.draws, trace.finals = 120, 60, 60
	trace.scenes["physical"] = 60
	trace.sample(now, false)
	if trace.err != nil {
		t.Fatal(trace.err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var row struct {
		Interval float64        `json:"interval_seconds"`
		Updates  int            `json:"updates"`
		Draws    int            `json:"draws"`
		Finals   int            `json:"final_draws"`
		Scenes   map[string]int `json:"draw_scenes"`
	}
	if err := json.Unmarshal(data, &row); err != nil {
		t.Fatal(err)
	}
	if row.Interval != 2 || row.Updates != 120 || row.Draws != 60 || row.Finals != 60 || row.Scenes["physical"] != 60 {
		t.Fatalf("wrong window: %+v", row)
	}
	if trace.updates != 0 || trace.draws != 0 || trace.finals != 0 || len(trace.scenes) != 0 {
		t.Fatal("window counters accumulated across intervals")
	}
	trace.sample(now.Add(500*time.Millisecond), false)
	after, _ := os.ReadFile(path)
	if string(after) != string(data) {
		t.Fatal("sub-second window was published early")
	}
	trace.sample(now.Add(500*time.Millisecond), true)
	after, _ = os.ReadFile(path)
	if len(strings.Split(strings.TrimSpace(string(after)), "\n")) != 2 {
		t.Fatal("final partial interval missing")
	}
	if err := trace.file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPerformanceTraceBattleReuseCounters(t *testing.T) {
	player := &modernHDPresentation{Game: &Game{modernHDBattleGPU: &modernHDBattleGPU{rendered: 12, reused: 27}}}
	trace, err := openPerformanceTrace(player, filepath.Join(t.TempDir(), "reuse.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer trace.file.Close()
	trace.sample(trace.started.Add(time.Second), false)
	data, err := os.ReadFile(trace.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	var row struct {
		Rendered uint64 `json:"battle_gpu_rendered_total"`
		Reused   uint64 `json:"battle_gpu_reused_total"`
	}
	if err := json.Unmarshal(data, &row); err != nil || row.Rendered != 12 || row.Reused != 27 {
		t.Fatalf("render/reuse counters missing or converted to draw rate: %+v, %v", row, err)
	}
}

func TestPerformanceTraceWriteFailureStopsObservation(t *testing.T) {
	trace, err := openPerformanceTrace(nil, filepath.Join(t.TempDir(), "performance.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := trace.file.Close(); err != nil {
		t.Fatal(err)
	}
	trace.sample(trace.started.Add(2*time.Second), false)
	if trace.err == nil {
		t.Fatal("closed report accepted")
	}
	if err := trace.Update(); err == nil || !strings.Contains(err.Error(), "效能觀測寫入") {
		t.Fatalf("write failure did not surface before game update: %v", err)
	}
}
