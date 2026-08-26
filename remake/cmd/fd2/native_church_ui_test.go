package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestNativeChurchShotStateIsStrictAndStableMenuOnly(t *testing.T) {
	for _, tc := range []struct {
		spec                   string
		selection, pulse, gold int
		ok                     bool
	}{
		{spec: "0,0,0", ok: true},
		{spec: "3,3,99999999", selection: 3, pulse: 3, gold: 99999999, ok: true},
		{spec: "-1,0,0"}, {spec: "4,0,0"},
		{spec: "0,-1,0"}, {spec: "0,4,0"},
		{spec: "0,0,-1"}, {spec: "0,0,100000000"},
		{spec: "1"}, {spec: "1,2"}, {spec: "1,2,3,4"}, {spec: "x,0,0"},
	} {
		selection, pulse, gold, ok := parseNativeChurchShotState(tc.spec)
		if ok != tc.ok || selection != tc.selection || pulse != tc.pulse || gold != tc.gold {
			t.Fatalf("parseNativeChurchShotState(%q)=(%d,%d,%d,%v), want (%d,%d,%d,%v)",
				tc.spec, selection, pulse, gold, ok,
				tc.selection, tc.pulse, tc.gold, tc.ok)
		}
	}

	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided original resources are absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{
		camp: campaign.NewRunner(&campaign.Campaign{
			Start: "church", Nodes: map[string]*campaign.Node{"church": {Type: "church"}},
		}),
		nativeClassUI: assets,
		churchMode:    "menu", churchSel: 2, nativeChurchUIPulse: 0,
		nativeChurchTextIndex: 585, nativeChurchUIJob: &nativeChurchUIJob{},
		gold: 7, nativeChurchUIHasTick: true, nativeChurchUILastTick: 123,
	}
	if !g.setNativeChurchShotState(0, 2, 1000) {
		t.Fatal("native church screenshot state rejected")
	}
	if g.churchSel != 0 || g.nativeChurchUIPulse != 2 || g.gold != 1000 ||
		g.nativeChurchUIJob != nil || g.nativeChurchUIHasTick || g.nativeChurchUILastTick != 0 ||
		!g.nativeChurchUIShotHold {
		t.Fatalf("church shot state=(selection %d pulse %d gold %d job %v last %d has %v)",
			g.churchSel, g.nativeChurchUIPulse, g.gold, g.nativeChurchUIJob != nil,
			g.nativeChurchUILastTick, g.nativeChurchUIHasTick)
	}
	g.stepNativeChurchUILifecycle(time.Unix(100, 0))
	g.stepNativeChurchUILifecycle(time.Unix(101, 0))
	if g.nativeChurchUIPulse != 2 {
		t.Fatalf("stable church shot pulse advanced=%d", g.nativeChurchUIPulse)
	}
	if frame, ok := g.composeNativeChurchMenuFrame(); !ok || len(frame) != 320*200 {
		t.Fatal("native church screenshot state did not reach production compositor")
	}

	g.churchMode = "revive"
	if g.setNativeChurchShotState(0, 0, 0) {
		t.Fatal("non-menu church screenshot state accepted")
	}
	g.churchMode = "menu"
	g.camp.C.Nodes["church"] = &campaign.Node{Type: "town"}
	if g.setNativeChurchShotState(0, 0, 0) {
		t.Fatal("non-church screenshot state accepted")
	}
}

func TestNativeChurchUILifecyclePresentsOpeningAndClosingRestore(t *testing.T) {
	g := &Game{nativeChurchUIJob: &nativeChurchUIJob{
		frames: [][]byte{{0}, {1}, {2}, {3}},
	}}
	for want := 0; want < 4; want++ {
		job := g.nativeChurchUIJob
		if job == nil || job.frame != want {
			t.Fatalf("opening present%d job=%#v", want, job)
		}
		job.drawn = true
		g.stepNativeChurchUILifecycle(time.Time{})
	}
	if g.nativeChurchUIJob != nil {
		t.Fatal("opening did not settle after four acknowledged presents")
	}

	completed := false
	g.nativeChurchUIJob = &nativeChurchUIJob{
		frames:  [][]byte{{0}, {1}, {2}, {3}},
		restore: make([]byte, 320*200),
		after:   func() { completed = true },
	}
	for want := 0; want < 4; want++ {
		job := g.nativeChurchUIJob
		if completed || job == nil || job.frame != want {
			t.Fatalf("closing present%d completed=%v job=%#v", want, completed, job)
		}
		job.drawn = true
		g.stepNativeChurchUILifecycle(time.Time{})
	}
	if completed || g.nativeChurchUIJob == nil || g.nativeChurchUIJob.frame != 4 {
		t.Fatalf("closing ran callback before source restore: completed=%v job=%#v", completed, g.nativeChurchUIJob)
	}
	g.nativeChurchUIJob.drawn = true
	g.stepNativeChurchUILifecycle(time.Time{})
	if !completed || g.nativeChurchUIJob != nil {
		t.Fatalf("restore did not hand off: completed=%v job=%#v", completed, g.nativeChurchUIJob)
	}
}

func TestNativeChurchUILifecycleCannotSkipUndrawnFrame(t *testing.T) {
	g := &Game{nativeChurchUIJob: &nativeChurchUIJob{
		frames: [][]byte{{0}, {1}, {2}, {3}},
	}}
	g.stepNativeChurchUILifecycle(time.Time{})
	g.stepNativeChurchUILifecycle(time.Time{})
	if g.nativeChurchUIJob.frame != 0 {
		t.Fatalf("undrawn frame advanced to %d", g.nativeChurchUIJob.frame)
	}
}

func TestNativeChurchUIPulseStartsSelectedAndUsesTwoTicks(t *testing.T) {
	g := &Game{}
	g.resetNativeChurchUIPulse()
	if g.nativeChurchUIPulse/2 != 1 {
		t.Fatalf("initial selected variant=%d", g.nativeChurchUIPulse/2)
	}
	g.stepNativeChurchUIPulseTick(10)
	g.stepNativeChurchUIPulseTick(11)
	if g.nativeChurchUIPulse != 2 {
		t.Fatalf("one-tick delta advanced pulse=%d", g.nativeChurchUIPulse)
	}
	g.stepNativeChurchUIPulseTick(12)
	if g.nativeChurchUIPulse != 3 || g.nativeChurchUIPulse/2 != 1 {
		t.Fatalf("two-tick pulse=%d variant=%d", g.nativeChurchUIPulse, g.nativeChurchUIPulse/2)
	}
	g.stepNativeChurchUIPulseTick(14)
	if g.nativeChurchUIPulse != 0 || g.nativeChurchUIPulse/2 != 0 {
		t.Fatalf("wrapped pulse=%d variant=%d", g.nativeChurchUIPulse, g.nativeChurchUIPulse/2)
	}
}

func TestNativeChurchRuntimeUsesPlayerOriginalSceneAssets(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided original resources are absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{
		nativeClassUI:         assets,
		churchMode:            "menu",
		churchSel:             2,
		nativeChurchTextIndex: 585,
		gold:                  12345678,
	}
	screen := ebiten.NewImage(640, 400)
	if !g.drawNativeChurchMenu(screen) {
		t.Fatal("native steady church menu unexpectedly fell back")
	}
	if !g.beginNativeChurchMenuOpening() || len(g.nativeChurchUIJob.frames) != 4 {
		t.Fatal("native church opening unexpectedly fell back")
	}
	g.nativeChurchUIJob = nil
	if !g.beginNativeChurchMenuClosing(nil) ||
		len(g.nativeChurchUIJob.frames) != 4 ||
		len(g.nativeChurchUIJob.restore) != 320*200 {
		t.Fatal("native church closing/restore unexpectedly fell back")
	}
}

func TestNativeChurchMenuTypedInputDispatchesAllFourServices(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided original resources are absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(base, "DATO.DAT"))
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}

	for selection, wantMode := range []string{"status_roster", "transfer_source", "revive_empty", "class"} {
		g := &Game{
			nativeClassUI: assets,
			churchMode:    "menu", churchSel: selection,
			nativeChurchTextIndex: 585,
			partyJoinOrder:        []int{9},
			partyRoster: map[int]battle.Unit{9: {
				Name: "悠妮", Portrait: 9, HP: 10, MaxHP: 10,
				Inventory: []int{1}, Equipped: []bool{false},
			}},
		}
		if !g.handleNativeChurchMenuInput(nativeChurchMenuInput{enter: true}) {
			t.Fatalf("selection %d was not consumed", selection)
		}
		if g.churchMode != "menu" || g.nativeChurchUIJob == nil {
			t.Fatalf("selection %d published before menu closing: mode=%q job=%#v",
				selection, g.churchMode, g.nativeChurchUIJob)
		}
		for steps := 0; g.nativeChurchUIJob != nil; steps++ {
			if steps >= 8 {
				t.Fatalf("selection %d closing did not settle", selection)
			}
			g.nativeChurchUIJob.drawn = true
			g.stepNativeChurchUILifecycle(time.Time{})
		}
		if g.churchMode != wantMode {
			t.Fatalf("selection %d mode=%q want=%q", selection, g.churchMode, wantMode)
		}
		if selection < 2 && len(g.churchIDs) != 1 {
			t.Fatalf("selection %d roster=%v", selection, g.churchIDs)
		}
	}
}
