package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestNativeTownAssetsDoNotRequireFDOTHERArchive(t *testing.T) {
	pack, err := filepath.Abs("../../generated-assets/fd2-original-b97caf22")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(pack, "ui", "fdother_010_town_label", "resource.json")); err != nil {
		t.Skip("generated separated town pack is absent")
	}
	t.Setenv("FD2_ASSET_PACK", pack)
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(t.TempDir(), "missing-FDOTHER.DAT"))
	assets, err := loadNativeTownUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	if assets.scene == nil {
		t.Fatal("separated town scene was not preflighted")
	}
}

func TestNativeTownProductionOwnerUsesEditableVariantAndHiddenSelection(t *testing.T) {
	base := filepath.Join("..", "..", "..", "org_game", "炎龍騎士團", "FLAME2")
	if _, err := os.Stat(filepath.Join(base, "FDOTHER.DAT")); err != nil {
		t.Skip("player-provided original resources are absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_DATO", filepath.Join(base, "DATO.DAT"))
	shared, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	town, err := loadNativeTownUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	variant := 2
	c := &campaign.Campaign{
		Start: "town",
		Nodes: map[string]*campaign.Node{
			"town": {
				Type:              "town",
				NativeTownVariant: &variant,
				NativeSecretGate: &campaign.NativeTownSecretGate{
					Selection: 1, ScanCode: 0x5f, To: "secret",
				},
			},
			"secret": {Type: "shop", NativeHubVariant: 5},
		},
	}
	g := &Game{
		camp:          campaign.NewRunner(c),
		campSel:       1,
		nativeClassUI: shared,
		nativeTownUI:  town,
	}
	visible, ok := g.composeNativeTownFrame()
	if !ok || len(visible) != 320*200 {
		t.Fatalf("visible native town frame=%d ok=%v", len(visible), ok)
	}
	if !g.camp.MatchNativeTownSecret(g.campSel, 0x5f) ||
		g.camp.NodeID() != "town" {
		t.Fatal("native secret chord did not remain in the town owner")
	}
	g.campSel = 5
	hidden, ok := g.composeNativeTownFrame()
	if !ok || bytes.Equal(visible, hidden) {
		t.Fatal("hidden selection did not redraw the native town frame")
	}
	if !g.camp.ConfirmNativeTownSecret(g.campSel) ||
		g.camp.NodeID() != "secret" {
		t.Fatal("confirmed hidden selection did not dispatch variant 5")
	}
}

func TestNativeTownPulseUsesFourTickSignedDelta(t *testing.T) {
	g := &Game{}
	for _, tc := range []struct {
		tick int
		want int
	}{
		{0x7ffe, 0},
		{0x7fff, 0},
		{-0x7ffe, 1},
		{-0x7ffa, 2},
		{-0x7fF6, 3},
		{-0x7fF2, 0},
	} {
		g.stepNativeTownUIPulseTick(tc.tick)
		if g.nativeTownUIPulse != tc.want {
			t.Fatalf(
				"tick %#x pulse=%d want %d",
				tc.tick, g.nativeTownUIPulse, tc.want,
			)
		}
	}
}

func TestNativeTownSelectionUsesOriginalLeftRightWrap(t *testing.T) {
	for _, tc := range []struct {
		selection int
		delta     int
		want      int
	}{
		{0, -1, 4},
		{4, 1, 0},
		{2, -1, 1},
		{2, 1, 3},
		{5, -1, 4},
		{5, 1, 0},
	} {
		got, ok := nativeTownMoveSelection(tc.selection, tc.delta)
		if !ok || got != tc.want {
			t.Fatalf(
				"selection=%d delta=%d got=%d ok=%v want=%d",
				tc.selection, tc.delta, got, ok, tc.want,
			)
		}
	}
	for _, invalid := range [][2]int{{-1, 1}, {6, -1}, {0, 0}} {
		if _, ok := nativeTownMoveSelection(invalid[0], invalid[1]); ok {
			t.Fatalf("invalid move %#v was accepted", invalid)
		}
	}

	g := &Game{
		campSel:              0,
		nativeTownUIPulse:    2,
		nativeTownUILastTick: 321,
		nativeTownUIHasTick:  true,
	}
	if !g.moveNativeTownSelection(1) || g.campSel != 1 {
		t.Fatalf("runtime move selection=%d", g.campSel)
	}
	if g.nativeTownUIPulse != 2 || g.nativeTownUILastTick != 321 ||
		!g.nativeTownUIHasTick {
		t.Fatalf(
			"selection move reset pulse: pulse=%d last=%d has=%v",
			g.nativeTownUIPulse, g.nativeTownUILastTick,
			g.nativeTownUIHasTick,
		)
	}
}

func TestNativeTownShotStateIsStrictAndTownOnly(t *testing.T) {
	for _, tc := range []struct {
		spec      string
		selection int
		pulse     int
		ok        bool
	}{
		{spec: "0,0", selection: 0, pulse: 0, ok: true},
		{spec: "5,3", selection: 5, pulse: 3, ok: true},
		{spec: "6,0"},
		{spec: "0,4"},
		{spec: "1"},
		{spec: "1,2,3"},
		{spec: "x,0"},
	} {
		selection, pulse, ok := parseNativeTownShotState(tc.spec)
		if ok != tc.ok || selection != tc.selection || pulse != tc.pulse {
			t.Fatalf(
				"parseNativeTownShotState(%q)=(%d,%d,%v), want (%d,%d,%v)",
				tc.spec, selection, pulse, ok,
				tc.selection, tc.pulse, tc.ok,
			)
		}
	}

	variant := 0
	g := &Game{
		nativeTownUI: &nativeTownUIAssets{},
		camp: &campaign.Runner{
			Cur: "town",
			C: &campaign.Campaign{
				Nodes: map[string]*campaign.Node{
					"town": {Type: "town", NativeTownVariant: &variant},
				},
			},
		},
		nativeTownUIHasTick:  true,
		nativeTownUILastTick: 123,
	}
	if !g.setNativeTownShotState(1, 2) {
		t.Fatal("native town screenshot state rejected")
	}
	if g.campSel != 1 || g.nativeTownUIPulse != 2 ||
		g.nativeTownUIHasTick || g.nativeTownUILastTick != 0 {
		t.Fatalf(
			"shot state=(selection=%d pulse=%d hasTick=%v last=%d)",
			g.campSel, g.nativeTownUIPulse,
			g.nativeTownUIHasTick, g.nativeTownUILastTick,
		)
	}

	g.camp.C.Nodes["town"] = &campaign.Node{Type: "choice"}
	if g.setNativeTownShotState(0, 0) {
		t.Fatal("non-town screenshot state accepted")
	}
}

func TestNativeTownSecretRevealPreservesPulseState(t *testing.T) {
	variant := 0
	g := &Game{
		campSel: 0,
		camp: &campaign.Runner{
			Cur: "town",
			C: &campaign.Campaign{
				Nodes: map[string]*campaign.Node{
					"town": {
						Type:              "town",
						NativeTownVariant: &variant,
						NativeSecretGate: &campaign.NativeTownSecretGate{
							Selection: 0,
							ScanCode:  0x54,
							To:        "secret",
						},
					},
					"secret": {Type: "shop"},
				},
			},
		},
		nativeTownUIPulse:    2,
		nativeTownUILastTick: 456,
		nativeTownUIHasTick:  true,
	}
	if !g.revealNativeTownSecret(0x54) || g.campSel != 5 {
		t.Fatalf("secret reveal selection=%d", g.campSel)
	}
	if g.nativeTownUIPulse != 2 || g.nativeTownUILastTick != 456 ||
		!g.nativeTownUIHasTick {
		t.Fatalf(
			"secret reveal reset pulse: pulse=%d last=%d has=%v",
			g.nativeTownUIPulse, g.nativeTownUILastTick,
			g.nativeTownUIHasTick,
		)
	}
	if g.revealNativeTownSecret(0x54) {
		t.Fatal("hidden selection accepted the normal-selection chord again")
	}
}
