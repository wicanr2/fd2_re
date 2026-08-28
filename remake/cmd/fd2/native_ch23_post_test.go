package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func nativeCh23LoopForTest(phase string) campaign.NativeCh23Loop {
	loop := campaign.NativeCh23Loop{Phase: phase, StageLatchSource: "byte_0x51a10", TickCounterSource: "[0x46c]"}
	if phase == "initial" {
		loop.Repeat = 30
		loop.StageValues = []int{2, 3, 4, 5, 6, 7, 8, 9}
		return loop
	}
	loop.Repeat = 12
	loop.StageValues = []int{10, 11, 12, 13, 14}
	loop.Palette = &campaign.HandlerNativeCall{}
	loop.PaletteTableSource = "0x60003"
	return loop
}

func completeNativeCh23Game(t *testing.T) (*Game, time.Time) {
	t.Helper()
	fdotherPath := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2/FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is unavailable")
	}
	assetPack := filepath.Clean("../../generated-assets/fd2-original-b97caf22")
	if _, err := os.Stat(filepath.Join(assetPack, "surfaces", "FDOTHER_042", "resource.json")); err != nil {
		t.Skip("separated FDOTHER #42 is unavailable")
	}
	t.Setenv("FD2_ASSET_PACK", assetPack)
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	assets, field, state := completeNativeMapFrameFixture(t)
	assets.LUTs = make([][]byte, 32)
	for i := range assets.LUTs {
		assets.LUTs[i] = make([]byte, 256)
	}
	for i := range assets.PaletteDAC {
		assets.PaletteDAC[i] = 63
	}
	g := &Game{nativeMapAssets: assets, m: field, st: state}
	start := time.Now().Add(-time.Second)
	if err := g.composeNativeMapFrameAt(start); err != nil {
		t.Fatal(err)
	}
	g.nativeMapDAC = append([]byte(nil), assets.PaletteDAC...)
	return g, start
}

func runNativeCh23Job(t *testing.T, g *Game, now *time.Time) {
	t.Helper()
	for steps := 0; g.nativeCh23Loop != nil && steps < 400; steps++ {
		g.nativeCh23Loop.drawn = true
		g.nativeCh23Loop.waitFrames = 0
		*now = (*now).Add(nativeBIOSTickPeriod)
		g.stepNativeCh23LoopAt(*now)
	}
	if g.nativeCh23Loop != nil {
		t.Fatal("native ch23 job exceeded bounded draw count")
	}
}

func TestNativeCh23LoopsPreserveSetterTickPaletteAndContinuation(t *testing.T) {
	g, now := completeNativeCh23Game(t)
	continued := 0
	if err := g.startNativeCh23Loop(nativeCh23LoopForTest("initial"), func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	if g.nativeCh23State == nil || g.nativeCh23State.latch != 2 || g.nativeCh23Loop.draw != 0 {
		t.Fatalf("first draw did not consume setter first: state=%+v job=%+v", g.nativeCh23State, g.nativeCh23Loop)
	}
	if continued != 0 {
		t.Fatal("initial continuation ran before all 240 draws")
	}
	now = g.nativeMapClock.last
	runNativeCh23Job(t, g, &now)
	if continued != 1 || g.nativeCh23State == nil || !g.nativeCh23State.initialComplete || g.nativeCh23State.latch != 9 {
		t.Fatalf("initial completion continuation=%d state=%+v err=%q", continued, g.nativeCh23State, g.loadErr)
	}

	phaseBefore := g.nativeFDOTHERPalettePhase
	if err := g.startNativeCh23Loop(nativeCh23LoopForTest("palette"), func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	if g.nativeCh23State.latch != 10 {
		t.Fatalf("palette first draw latch=%d, want 10", g.nativeCh23State.latch)
	}
	now = g.nativeMapClock.last
	runNativeCh23Job(t, g, &now)
	if continued != 2 || g.nativeCh23State.latch != 14 {
		t.Fatalf("palette completion continuation=%d state=%+v", continued, g.nativeCh23State)
	}
	// The first palette draw sees the 240-tick initial-loop gap; the remaining
	// calls advance on every second BIOS tick: 30 relative phase updates total.
	if got, want := g.nativeFDOTHERPalettePhase, (phaseBefore+30)&15; got != want {
		t.Fatalf("palette phase=%d want=%d", got, want)
	}
	// rawESI=59 is the final full-DAC subtraction. 0x11CAC(0) then owns
	// 0x4DFCC: indices outside E0..EF remain baseline-minus-59 (=4), while the
	// dynamic range retains the most recent raw palette-cycle window.
	wantDAC := make([]byte, 256*3)
	for i := range wantDAC {
		wantDAC[i] = 4
	}
	if err := fdother.ApplyNativeDACPaletteCycleE0EF(wantDAC, g.nativeFDOTHERPalettePhase); err != nil {
		t.Fatal(err)
	}
	for i, component := range g.nativeMapDAC {
		if component != wantDAC[i] {
			t.Fatalf("final DAC[%d]=%d, want %d", i, component, wantDAC[i])
		}
	}
}

func TestNativeCh23FailureRollsBackFirstVisibleDraw(t *testing.T) {
	g, now := completeNativeCh23Game(t)
	beforeState := *g.st
	beforeWork := append([]byte(nil), g.nativeMapWork...)
	beforeVGA := append([]byte(nil), g.nativeMapVGA...)
	beforeDAC := append([]byte(nil), g.nativeMapDAC...)
	beforeClock := g.nativeMapClock
	if err := g.startNativeCh23Loop(nativeCh23LoopForTest("initial"), nil); err != nil {
		t.Fatal(err)
	}
	g.nativeMapAssets.Terrain = nil
	g.nativeCh23Loop.drawn = true
	g.nativeCh23Loop.waitFrames = 0
	now = now.Add(nativeBIOSTickPeriod)
	g.stepNativeCh23LoopAt(now)
	if g.nativeCh23Loop != nil || g.nativeCh23State != nil || g.loadErr == "" {
		t.Fatalf("failed job was not closed: job=%v state=%v err=%q", g.nativeCh23Loop, g.nativeCh23State, g.loadErr)
	}
	if !reflect.DeepEqual(*g.st, beforeState) || g.nativeMapClock != beforeClock ||
		!bytes.Equal(g.nativeMapWork, beforeWork) || !bytes.Equal(g.nativeMapVGA, beforeVGA) ||
		!bytes.Equal(g.nativeMapDAC, beforeDAC) {
		t.Fatal("failed ch23 draw did not restore state/work/VGA/DAC/clock atomically")
	}
}

func TestNativeCh23RejectsMissingRawFrameBeforeMutation(t *testing.T) {
	g, _ := completeNativeCh23Game(t)
	g.nativeMapWork = g.nativeMapWork[:indexedmap.NativeUnitPresentWorkSize-1]
	before := append([]byte(nil), g.nativeMapWork...)
	if err := g.startNativeCh23Loop(nativeCh23LoopForTest("initial"), nil); err == nil {
		t.Fatal("short native work buffer was accepted")
	}
	if g.nativeCh23Loop != nil || g.nativeCh23State != nil || !bytes.Equal(g.nativeMapWork, before) {
		t.Fatal("rejected ch23 start partially mutated runtime")
	}
}

func TestNativeCh23RejectsMissingSeparatedStageBeforeMutation(t *testing.T) {
	g, _ := completeNativeCh23Game(t)
	t.Setenv("FD2_ASSET_PACK", t.TempDir())
	if err := g.startNativeCh23Loop(nativeCh23LoopForTest("initial"), nil); err == nil {
		t.Fatal("missing separated FDOTHER #42 was accepted")
	}
	if g.nativeCh23State != nil || g.nativeCh23Loop != nil {
		t.Fatal("missing separated FDOTHER #42 partially published loop state")
	}
}
