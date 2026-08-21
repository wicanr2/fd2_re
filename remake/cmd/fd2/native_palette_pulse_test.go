package main

import (
	"bytes"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func recoveredNativePalettePulse() campaign.NativePalettePulse {
	return campaign.NativePalettePulse{
		RiseStart: 0, RiseEnd: 63, RiseDelayMs: 8,
		HoldMs:    400,
		FallStart: 62, FallEnd: 0, FallDelayMs: 8,
	}
}

func TestNativePalettePulsePreservesAsymmetric127StepSchedule(t *testing.T) {
	g := completeNativeUnitPresentGame(t)
	beforeVGA := append([]byte(nil), g.nativeMapVGA...)
	continued := 0
	if err := g.startNativePalettePulse(recoveredNativePalettePulse(), func() { continued++ }); err != nil {
		t.Fatal(err)
	}
	j := g.nativePalettePulse
	if j == nil || len(j.deltas) != 127 || j.deltas[0] != 0 || j.deltas[63] != 63 ||
		j.deltas[64] != 62 || j.deltas[126] != 0 || j.waits[63] != 25 {
		t.Fatalf("pulse=%+v", j)
	}
	g.stepNativePalettePulse()
	if j.step != 0 {
		t.Fatal("pulse advanced without a draw acknowledgement")
	}
	for steps := 0; g.nativePalettePulse != nil && steps < 200; steps++ {
		g.nativePalettePulse.drawn = true
		g.nativePalettePulse.wait = 0
		g.stepNativePalettePulse()
	}
	if g.nativePalettePulse != nil || continued != 1 || !bytes.Equal(beforeVGA, g.nativeMapVGA) ||
		!bytes.Equal(g.nativeMapDAC, g.nativeMapAssets.PaletteDAC) {
		t.Fatalf("job=%v continued=%d", g.nativePalettePulse, continued)
	}
}

func TestNativePalettePulseRejectsScheduleDriftBeforePublication(t *testing.T) {
	g := completeNativeUnitPresentGame(t)
	beforeDAC, beforeVGA := append([]byte(nil), g.nativeMapDAC...), append([]byte(nil), g.nativeMapVGA...)
	spec := recoveredNativePalettePulse()
	spec.FallStart = 63
	if err := g.startNativePalettePulse(spec, nil); err == nil {
		t.Fatal("drifted 0x35E5A schedule was accepted")
	}
	if g.nativePalettePulse != nil || !bytes.Equal(beforeDAC, g.nativeMapDAC) || !bytes.Equal(beforeVGA, g.nativeMapVGA) {
		t.Fatal("failed pulse preflight changed indexed state")
	}
}
