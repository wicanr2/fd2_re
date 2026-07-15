package campaign

import (
	"path/filepath"
	"strconv"
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
		var checkSources func(string, []HandlerBeat)
		checkSources = func(location string, beats []HandlerBeat) {
			for i, beat := range beats {
				at := location + "[" + strconv.Itoa(i) + "]"
				if beat.Source.Addr == "" {
					t.Errorf("%s %s lacks source address", path, at)
				}
				checkSources(at+".then", beat.Then)
				checkSources(at+".else", beat.Else)
			}
		}
		checkSources("beats", script.Beats)
	}
}

func TestChapter1PostPreservesInactiveDiamond(t *testing.T) {
	script, err := LoadHandlerScript("../../assets/cutscenes/handlers/ch01_post.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(script.Beats) == 0 || script.Beats[0].Op != "if" {
		t.Fatalf("first beat = %#v, want structured if", script.Beats)
	}
	branch := script.Beats[0]
	if branch.Source.Addr != "0x22f71" || branch.Condition == nil || branch.Condition.Op != "any_unit_inactive" {
		t.Fatalf("branch identity = %#v", branch)
	}
	wantSlots := []int{5, 6, 7, 8, 9, 10}
	if len(branch.Condition.UnitSlots) != len(wantSlots) {
		t.Fatalf("inactive slots = %#v", branch.Condition.UnitSlots)
	}
	for i, want := range wantSlots {
		if branch.Condition.UnitSlots[i] != want {
			t.Fatalf("inactive slots = %#v", branch.Condition.UnitSlots)
		}
	}
	if len(branch.Then) != 1 || branch.Then[0].TextIndex != float64(7) {
		t.Fatalf("inactive arm = %#v", branch.Then)
	}
	if len(branch.Else) != 2 || branch.Else[0].TextIndex != float64(6) || branch.Else[1].ItemID == nil || *branch.Else[1].ItemID != 0xc6 {
		t.Fatalf("all-active arm = %#v", branch.Else)
	}
	if len(script.Beats) < 2 || script.Beats[1].Op != "pan" {
		t.Fatalf("common continuation = %#v", script.Beats)
	}
}

func TestChapter0PreHandlerPreservesReclassifiedNativeOperations(t *testing.T) {
	script, err := LoadHandlerScript("../../assets/cutscenes/handlers/ch00_pre.json")
	if err != nil {
		t.Fatal(err)
	}
	if script.Handler != "0x3231b" || script.Phase != "pre" || script.Chapter != 0 {
		t.Fatalf("chapter 0 pre identity = %#v", script)
	}
	if got := script.Diagnostics["unknown_ops"]; got != 0 {
		t.Fatalf("chapter 0 unknown operations = %d, want 0 after native body classification", got)
	}
	want := []string{"loadch", "pan", "act", "scroll_step", "dialog", "scroll_step", "dialog", "bgm"}
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
	if script.Beats[3].UnitSlot == nil || *script.Beats[3].UnitSlot != 2 || script.Beats[3].Repeat == nil || *script.Beats[3].Repeat != 15 {
		t.Errorf("first scroll step = slot %v repeat %v, want slot 2 repeat 15", script.Beats[3].UnitSlot, script.Beats[3].Repeat)
	}
	wantTail := []string{"deactivate_unit", "redraw", "delay", "dialog", "reset_pose", "focus_unit"}
	for i, op := range wantTail {
		beat := script.Beats[len(script.Beats)-len(wantTail)+i]
		if beat.Op != op {
			t.Errorf("tail beat %d op=%q, want %q", i, beat.Op, op)
		}
	}
}
