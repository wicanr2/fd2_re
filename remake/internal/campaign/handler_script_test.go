package campaign

import (
	"path/filepath"
	"testing"
)

func TestLoadExportedHandlerScripts(t *testing.T) {
	paths, err := filepath.Glob("../../assets/cutscenes/handlers/ch??_*.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 60 {
		t.Fatalf("exported handler scripts = %d, want 60", len(paths))
	}
	for _, path := range paths {
		script, err := LoadHandlerScript(path)
		if err != nil {
			t.Fatalf("LoadHandlerScript(%q): %v", path, err)
		}
		for i, beat := range script.Beats {
			if beat.Source.Addr == "" {
				t.Errorf("%s beat %d lacks source address", path, i)
			}
		}
	}
}

func TestChapter0PreHandlerPreservesKnownAndUnknownOperations(t *testing.T) {
	script, err := LoadHandlerScript("../../assets/cutscenes/handlers/ch00_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	if script.Handler != "0x3231b" || script.Phase != "pre" || script.Chapter != 0 {
		t.Fatalf("chapter 0 pre identity = %#v", script)
	}
	if got := script.Diagnostics["unknown_ops"]; got != 2 {
		t.Fatalf("chapter 0 unknown operations = %d, want 2", got)
	}
	want := []string{"loadch", "pan", "act", "step", "dialog", "step", "dialog", "bgm"}
	if len(script.Beats) < len(want) {
		t.Fatalf("chapter 0 has only %d beats", len(script.Beats))
	}
	for i, op := range want {
		if got := script.Beats[i].Op; got != op {
			t.Errorf("beat %d op = %q, want %q", i, got, op)
		}
	}
	if script.Beats[0].Chapter == nil || *script.Beats[0].Chapter != 32 {
		t.Errorf("first loadch chapter = %v, want 32", script.Beats[0].Chapter)
	}
	if script.Beats[2].ActingID == nil || *script.Beats[2].ActingID != 99 {
		t.Errorf("first act id = %v, want 99", script.Beats[2].ActingID)
	}
	if script.Beats[3].Repeat == nil || *script.Beats[3].Repeat != 15 {
		t.Errorf("first step repeat = %v, want 15", script.Beats[3].Repeat)
	}
}
