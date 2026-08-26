package main

import (
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

func TestNativePreparationCycleUses1297DBoundariesAndVisibleSequence(t *testing.T) {
	g := &Game{}
	tests := []struct {
		tick, state, visible int
	}{
		{tick: 0, state: 0, visible: 0},
		{tick: 4, state: 0, visible: 0},
		{tick: 5, state: 1, visible: 1},
		{tick: 9, state: 1, visible: 1},
		{tick: 10, state: 2, visible: 2},
		{tick: 15, state: 3, visible: 1},
		{tick: 20, state: 0, visible: 0},
	}
	for _, test := range tests {
		g.stepNativePreparationCycleTick(test.tick)
		visible, err := fdicon.NativeFrameIndex(
			0, false, g.prepIdleCycle, 0,
		)
		if err != nil || g.prepIdleCycle != test.state || visible != test.visible {
			t.Fatalf(
				"tick=%d state=%d visible=%d err=%v, want state=%d visible=%d",
				test.tick, g.prepIdleCycle, visible, err, test.state, test.visible,
			)
		}
	}
}

func TestNativePreparationCycleAdvancesOnSignedBIOSWrap(t *testing.T) {
	g := &Game{prepIdleCycle: 2, prepLastTick: 0x7fff}
	g.stepNativePreparationCycleTick(-0x8000)
	if g.prepIdleCycle != 3 || g.prepLastTick != -0x8000 {
		t.Fatalf("wrapped state=%d last=%d", g.prepIdleCycle, g.prepLastTick)
	}
}

func TestNativePreparationLifecycleRunsOnlyForActiveSelection(t *testing.T) {
	g := &Game{
		camp: campaign.NewRunner(&campaign.Campaign{
			Start: "prep",
			Nodes: map[string]*campaign.Node{
				"prep": {Type: "preparation"},
			},
		}),
		nativePreparationUI: &nativePreparationUIAssets{},
	}
	start := time.Unix(1, 0)
	g.stepNativePreparationUILifecycle(start)
	g.stepNativePreparationUILifecycle(start.Add(5 * nativeBIOSTickPeriod))
	if g.prepIdleCycle != 0 {
		t.Fatalf("inactive preparation advanced to %d", g.prepIdleCycle)
	}
	g.prepSelecting = true
	g.stepNativePreparationUILifecycle(start.Add(10 * nativeBIOSTickPeriod))
	g.stepNativePreparationUILifecycle(start.Add(15 * nativeBIOSTickPeriod))
	if g.prepIdleCycle != 1 || g.prepLastTick != 5 {
		t.Fatalf("active preparation state=%d last=%d", g.prepIdleCycle, g.prepLastTick)
	}
	g.camp.Cur = "missing"
	g.stepNativePreparationUILifecycle(start.Add(20 * nativeBIOSTickPeriod))
	if g.prepIdleCycle != 1 {
		t.Fatalf("non-preparation node advanced to %d", g.prepIdleCycle)
	}
}

func TestNativePreparationFixedHashShotFreezesRequestedCycle(t *testing.T) {
	g := &Game{
		camp: campaign.NewRunner(&campaign.Campaign{
			Start: "prep",
			Nodes: map[string]*campaign.Node{
				"prep": {Type: "preparation"},
			},
		}),
		nativePreparationUI: &nativePreparationUIAssets{},
		prepSelecting:       true,
		prepIdleCycle:       2,
		prepShotCycleFrozen: true,
	}
	g.stepNativePreparationUILifecycle(time.Unix(100, 0))
	g.stepNativePreparationUILifecycle(time.Unix(200, 0))
	if g.prepIdleCycle != 2 {
		t.Fatalf("固定雜湊截圖相位被時鐘推進為 %d", g.prepIdleCycle)
	}
}

func TestNativePreparationConfirmationUses19953PulseCadence(t *testing.T) {
	g := &Game{
		camp: campaign.NewRunner(&campaign.Campaign{
			Start: "prep",
			Nodes: map[string]*campaign.Node{
				"prep": {Type: "preparation"},
			},
		}),
		prepConfirm: true,
	}
	start := time.Unix(10, 0)
	g.stepNativeClassUILifecycle(start)
	g.stepNativeClassUILifecycle(start.Add(nativeBIOSTickPeriod))
	if g.nativeClassUIPulse != 0 {
		t.Fatalf("one BIOS tick advanced confirmation pulse to %d", g.nativeClassUIPulse)
	}
	g.stepNativeClassUILifecycle(start.Add(2 * nativeBIOSTickPeriod))
	if g.nativeClassUIPulse != 1 {
		t.Fatalf("two BIOS ticks left confirmation pulse at %d", g.nativeClassUIPulse)
	}
	g.prepConfirm = false
	g.prepSelecting = true
	g.stepNativeClassUILifecycle(start.Add(4 * nativeBIOSTickPeriod))
	if g.nativeClassUIPulse != 1 {
		t.Fatalf("inactive preparation confirmation advanced pulse to %d", g.nativeClassUIPulse)
	}
}

func TestNativePreparationPromptUses19953PulseCadence(t *testing.T) {
	g := &Game{
		camp: campaign.NewRunner(&campaign.Campaign{
			Start: "prep",
			Nodes: map[string]*campaign.Node{
				"prep": {Type: "preparation"},
			},
		}),
	}
	start := time.Unix(20, 0)
	g.stepNativeClassUILifecycle(start)
	g.stepNativeClassUILifecycle(start.Add(2 * nativeBIOSTickPeriod))
	if g.nativeClassUIPulse != 1 {
		t.Fatalf("preparation prompt pulse=%d want 1", g.nativeClassUIPulse)
	}
	g.prepSelecting = true
	g.stepNativeClassUILifecycle(start.Add(4 * nativeBIOSTickPeriod))
	if g.nativeClassUIPulse != 1 {
		t.Fatalf("inactive prompt advanced pulse to %d", g.nativeClassUIPulse)
	}
}
