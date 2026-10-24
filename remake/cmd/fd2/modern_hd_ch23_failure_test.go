package main

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestModernHDCh23LateNativeFailureRestoresBothThemes(t *testing.T) {
	g, control := stage42TestGame(t, true), stage42TestGame(t, false)
	// Deliberately reproduce the compact fixture's missing later phases.
	// This is a fault injection, never a production lookup policy.
	for _, v := range []*Game{g, control} {
		v.nativeMapAssets.LUTs = v.nativeMapAssets.LUTs[:10]
	}
	before, otherBefore := stage42NativeState(g), stage42NativeState(control)
	now := time.Unix(1000, 0)
	for _, v := range []*Game{g, control} {
		v.nativeMapFrozenNow = func() time.Time { return now }
	}
	a, b := 0, 0
	if err := g.startNativeCh23Loop(nativeCh23LoopForTest("initial"), func() { a++ }); err != nil {
		t.Fatal(err)
	}
	if err := control.startNativeCh23Loop(nativeCh23LoopForTest("initial"), func() { b++ }); err != nil {
		t.Fatal(err)
	}
	count := 0
	for g.nativeCh23Loop != nil && control.nativeCh23Loop != nil && count < 240 {
		g.nativeCh23Loop.drawn = true
		control.nativeCh23Loop.drawn = true
		for tick := 0; tick < 3; tick++ {
			now = now.Add(time.Second / 60)
			g.stepNativeCh23LoopAt(now)
			control.stepNativeCh23LoopAt(now)
		}
		count++
	}
	if count != 33 || g.nativeCh23Loop != nil || control.nativeCh23Loop != nil || a != 0 || b != 0 ||
		!strings.Contains(g.loadErr, "terrain phase LUT") || !strings.Contains(control.loadErr, "terrain phase LUT") ||
		!reflect.DeepEqual(before, stage42NativeState(g)) || !reflect.DeepEqual(otherBefore, stage42NativeState(control)) {
		t.Fatal("late native fault diverged or failed rollback", count, g.loadErr, control.loadErr)
	}
	t.Logf("controlled ten-LUT fault at draw %d: HD=%s; native-theme=%s; both fully rolled back", count, g.loadErr, control.loadErr)
}
