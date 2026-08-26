package main

import (
	"os"
	"testing"
	"time"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func TestNativeChurchPanelLifecycleMatches17AEDSchedules(t *testing.T) {
	source := make([]byte, 320*200)
	panel := make([]byte, 320*200)
	for y := 7; y < 196; y++ {
		for x := 5; x < 315; x++ {
			panel[y*320+x] = 9
		}
	}
	opening, err := nativeChurchPanelFrames(source, panel, true)
	if err != nil || len(opening) != 12 {
		t.Fatalf("opening frames=%d err=%v", len(opening), err)
	}
	closing, err := nativeChurchPanelFrames(source, panel, false)
	if err != nil || len(closing) != 12 {
		t.Fatalf("closing frames=%d err=%v", len(closing), err)
	}
	if opening[0][7*320+5] != 9 || opening[11][7*320+5] != 9 {
		t.Fatal("left panel was not composed in opening schedule")
	}
	transition, err := nativeChurchBottomPanelFrame(source, panel, 6)
	if err != nil {
		t.Fatal(err)
	}
	if transition[190*320+5] != 9 || transition[189*320+5] != 0 {
		t.Fatal("bottom step6 did not land at y190")
	}
}

func TestNativeChurchStatusUsesPlayerOriginalAssets(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2/"
	for _, name := range []string{"FDOTHER.DAT", "FDTXT.DAT", "DATO.DAT"} {
		if _, err := os.Stat(base + name); err != nil {
			t.Skip("player-provided original archives are absent")
		}
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", base+"FDOTHER.DAT")
	t.Setenv("FD2_ORIGINAL_FDTXT", base+"FDTXT.DAT")
	t.Setenv("FD2_ORIGINAL_DATO", base+"DATO.DAT")
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	unit := *nativeItemPanelTestUnit()
	unit.NativeIdentity, unit.HasNativeIdentity = 9, true
	unit.MapSelectorKey, unit.HasMapSelectorKey = 9, true
	unit.NativeCommandMask = [5]byte{1}
	g := &Game{
		nativeClassUI:         assets,
		nativeChurchTextIndex: 585,
		partyRoster:           map[int]battle.Unit{9: unit},
		churchMode:            "status_roster",
		churchIDs:             []int{9},
		churchStatusID:        -1,
		nativeCommandBook:     []battle.NativeCommandRecord{{ID: 0, MPCost: 2}},
	}
	status, commands, ok := g.prepareNativeChurchStatus(9)
	if !ok || len(status) != 320*200 || len(commands) != 320*200 {
		t.Fatalf("status=%d commands=%d ok=%v", len(status), len(commands), ok)
	}
	if !g.beginNativeChurchStatus(9) || g.churchMode != "status_view" ||
		len(g.nativeClassUIJob.frames) != 12 {
		t.Fatal("native status twelve-frame opening did not start")
	}
	g.nativeClassUIJob = nil
	if !g.beginNativeChurchStatusCommandTransition() ||
		len(g.nativeClassUIJob.frames) != 14 {
		t.Fatal("native status-to-command fourteen-frame transition did not start")
	}
	g.nativeClassUIJob = nil
	g.churchMode = "status_commands"
	g.closeNativeChurchStatus(g.churchCommandPanel)
	if g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 12 ||
		len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatal("native command panel twelve-frame closing did not start")
	}
}

func TestNativeChurchStatusTypedInputRoundTrip(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2/"
	for _, name := range []string{"FDOTHER.DAT", "FDTXT.DAT", "DATO.DAT"} {
		if _, err := os.Stat(base + name); err != nil {
			t.Skip("player-provided original archives are absent")
		}
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", base+"FDOTHER.DAT")
	t.Setenv("FD2_ORIGINAL_FDTXT", base+"FDTXT.DAT")
	t.Setenv("FD2_ORIGINAL_DATO", base+"DATO.DAT")
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	unit := *nativeItemPanelTestUnit()
	unit.NativeIdentity, unit.HasNativeIdentity = 9, true
	unit.MapSelectorKey, unit.HasMapSelectorKey = 9, true
	unit.NativeCommandMask = [5]byte{1}
	g := &Game{
		nativeClassUI: assets, nativeChurchTextIndex: 585,
		partyRoster: map[int]battle.Unit{9: unit}, partyJoinOrder: []int{9},
		churchMode: "status_roster", churchIDs: []int{9}, churchStatusID: -1,
		nativeCommandBook: []battle.NativeCommandRecord{{ID: 0, MPCost: 2}},
	}
	drain := func(label string) {
		for steps := 0; g.nativeClassUIJob != nil; steps++ {
			if steps >= 64 {
				t.Fatalf("%s did not settle: mode=%q job=%#v", label, g.churchMode, g.nativeClassUIJob)
			}
			g.nativeClassUIJob.drawn = true
			g.stepNativeClassUILifecycle(time.Time{})
		}
	}

	if !g.handleNativeChurchStatusInput(nativeChurchStatusInput{enter: true}) {
		t.Fatal("status roster confirmation was not consumed")
	}
	if g.churchMode != "status_roster" {
		t.Fatalf("status published before roster closing: %q", g.churchMode)
	}
	drain("status opening")
	if g.churchMode != "status_view" || g.churchStatusID != 9 {
		t.Fatalf("status mode=%q id=%d", g.churchMode, g.churchStatusID)
	}
	if !g.handleNativeChurchStatusInput(nativeChurchStatusInput{enter: true}) {
		t.Fatal("status-to-command confirmation was not consumed")
	}
	drain("status-to-command")
	if g.churchMode != "status_commands" {
		t.Fatalf("command mode=%q", g.churchMode)
	}
	if !g.handleNativeChurchStatusInput(nativeChurchStatusInput{escape: true}) {
		t.Fatal("command panel escape was not consumed")
	}
	drain("command return")
	if g.churchMode != "status_roster" || g.churchStatusID != -1 || len(g.churchIDs) != 1 {
		t.Fatalf("return mode=%q id=%d roster=%v", g.churchMode, g.churchStatusID, g.churchIDs)
	}
}

func TestNativeChurchStatusPublishesRawTransientIndicators(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2/"
	for _, name := range []string{"FDOTHER.DAT", "FDTXT.DAT", "DATO.DAT"} {
		if _, err := os.Stat(base + name); err != nil {
			t.Skip("player-provided original archives are absent")
		}
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", base+"FDOTHER.DAT")
	t.Setenv("FD2_ORIGINAL_FDTXT", base+"FDTXT.DAT")
	t.Setenv("FD2_ORIGINAL_DATO", base+"DATO.DAT")
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	unit := *nativeItemPanelTestUnit()
	unit.NativeIdentity, unit.HasNativeIdentity = 9, true
	unit.MapSelectorKey, unit.HasMapSelectorKey = 9, true
	g := &Game{
		nativeClassUI: assets,
		partyRoster:   map[int]battle.Unit{9: unit},
	}
	plain, _, ok := g.prepareNativeChurchStatus(9)
	if !ok {
		t.Fatal("plain native status panel was unavailable")
	}
	unit.NativeTransient = [6]byte{1, 1, 1, 1, 1, 1}
	g.partyRoster[9] = unit
	flagged, _, ok := g.prepareNativeChurchStatus(9)
	if !ok {
		t.Fatal("flagged native status panel was unavailable")
	}
	for _, region := range [][4]int{
		{157, 67, 18, 9}, {157, 79, 18, 9},
		{117, 67, 18, 9}, {117, 79, 18, 9},
		{194, 68, 30, 20}, {229, 68, 30, 20}, {264, 68, 30, 20},
	} {
		if !nativeStatusRegionDiffers(plain, flagged, region) {
			t.Fatalf("raw transient indicator region %v did not change", region)
		}
	}
}

func nativeStatusRegionDiffers(a, b []byte, region [4]int) bool {
	for y := region[1]; y < region[1]+region[3]; y++ {
		for x := region[0]; x < region[0]+region[2]; x++ {
			offset := y*320 + x
			if a[offset] != b[offset] {
				return true
			}
		}
	}
	return false
}
