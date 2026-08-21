package main

import (
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func materializedNativeClockGame(t *testing.T) *Game {
	t.Helper()
	st := &battle.State{W: 13, H: 8}
	if err := st.AppendNativeMapSelectorBatch(nil); err != nil {
		t.Fatal(err)
	}
	if err := st.MaterializeNativeMapViewState(battle.NativeMapViewState{}); err != nil {
		t.Fatal(err)
	}
	return &Game{st: st}
}

func TestNativeBIOSClockUsesMonotonicBattleLocalLowWord(t *testing.T) {
	var clock nativeBIOSClock
	start := time.Unix(100, 0)
	if got := clock.Sample(start); got != 0 {
		t.Fatalf("initial tick=%d", got)
	}
	if got := clock.Sample(start.Add(5*nativeBIOSTickPeriod + nativeBIOSTickPeriod/2)); got != 5 {
		t.Fatalf("five ticks=%d", got)
	}
	if got := clock.Sample(start.Add(5 * nativeBIOSTickPeriod)); got != 5 {
		t.Fatalf("backward wall sample changed monotonic tick=%d", got)
	}
	clock.elapsedTicks = 0x7fff
	clock.remainder = 0
	clock.last = start
	if got := clock.Sample(start.Add(nativeBIOSTickPeriod)); got != -0x8000 {
		t.Fatalf("signed low-word wrap=%d", got)
	}
}

func TestNativeBIOSClockSeedsCapturedSignedLowWord(t *testing.T) {
	var clock nativeBIOSClock
	start := time.Unix(150, 0)
	if !clock.Seed(-123, start) {
		t.Fatal("valid signed title tick was rejected")
	}
	if got := clock.Sample(start); got != -123 {
		t.Fatalf("seed sample=%d", got)
	}
	if got := clock.Sample(start.Add(5 * nativeBIOSTickPeriod)); got != -118 {
		t.Fatalf("elapsed seeded sample=%d", got)
	}
	before := clock
	if clock.Seed(0x8000, start) || clock != before {
		t.Fatal("invalid seed changed clock")
	}
}

func TestNativeBIOSClockCurrentDoesNotAdvance(t *testing.T) {
	var clock nativeBIOSClock
	if _, ok := clock.Current(); ok {
		t.Fatal("unseeded clock exposed a current tick")
	}
	now := time.Unix(900, 0)
	if !clock.Seed(123, now) {
		t.Fatal("clock seed rejected")
	}
	before := clock
	if got, ok := clock.Current(); !ok || got != 123 {
		t.Fatalf("current=(%d,%v), want (123,true)", got, ok)
	}
	if clock != before {
		t.Fatal("Current advanced or mutated the clock")
	}
}

func TestGameAdvancesAllNativeMapFrameTimingFromOneBIOSSample(t *testing.T) {
	g := materializedNativeClockGame(t)
	start := time.Unix(200, 0)
	if !g.advanceNativeMapClock(start) {
		t.Fatal("initial native map clock sample rejected")
	}
	if got := g.st.NativeMapCycleState.Moving; got != 1 {
		t.Fatalf("moving cycle after first 0x1297d call=%d", got)
	}
	if !g.advanceNativeMapClock(start.Add(5 * nativeBIOSTickPeriod)) {
		t.Fatal("five-tick native map clock sample rejected")
	}
	if got := g.st.NativeMapCycleState; got.Idle != 1 || got.Moving != 2 || got.LastTimerTick != 5 {
		t.Fatalf("sprite cycles=%+v", got)
	}
	if got := g.st.NativeTerrainPhaseState; got.Phase != 1 || got.LastTimerTick != 5 {
		t.Fatalf("terrain phase=%+v", got)
	}
	if got := g.st.NativeTerrainFlipState; got.Value != 1 || got.LastTimerTick != 5 {
		t.Fatalf("terrain flip=%+v", got)
	}
	if got := g.st.NativeUnitPixelShiftState; got.Value != 1 || got.LastTimerTick != 5 {
		t.Fatalf("unit pixel shift=%+v", got)
	}
}

func TestGameNativeMapClockFailsClosedWithoutView(t *testing.T) {
	g := materializedNativeClockGame(t)
	g.st.HasNativeMapViewState = false
	before := g.st.NativeMapCycleState
	if g.advanceNativeMapClock(time.Unix(300, 0)) || g.st.NativeMapCycleState != before {
		t.Fatal("partial native runtime advanced")
	}
}
