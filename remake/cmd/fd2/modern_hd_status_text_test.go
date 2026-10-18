package main

import (
	"bytes"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

func newModernHDStatusGame(t *testing.T) (*Game, *ebiten.Image, *battle.Unit) {
	t.Helper()
	a, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	u := nativeItemPanelTestUnit()
	u.NativeIdentity, u.HasNativeIdentity = 9, true
	u.MapSelectorKey, u.HasMapSelectorKey = 9, true
	u.NativeCommandMask = [5]byte{1}
	g := &Game{nativeClassUI: a, nativeUIPalette: a.palette,
		nativeChurchTextIndex: 585, partyRoster: map[int]battle.Unit{9: *u},
		nativeCommandBook: []battle.NativeCommandRecord{{ID: 0, MPCost: 2}}}
	return g, ebiten.NewImage(logicalW, logicalH), u
}

func TestModernHDStatusSnapshotMatchesLegacyComposition(t *testing.T) {
	g, screen, u := newModernHDStatusGame(t)
	d, status, commands, ok := g.prepareNativeUnitStatusData(u)
	if !ok {
		t.Fatal("status snapshot unavailable")
	}
	base := make([]byte, 64000)
	if err := battle.RenderNativeItemPanelResources(separatedAssetPath(""), separatedAssetPath("portraits"), d.record, base); err != nil {
		t.Fatal(err)
	}
	for _, command := range []bool{false, true} {
		legacy := append([]byte(nil), base...)
		var err error
		want := status
		if command {
			err = battle.RenderNativeCommandOverlay(d.assets, d.commandIDs, g.nativeCommandBook, -1, legacy)
			want = commands
		} else {
			err = battle.RenderNativeItemPanelRows(d.assets, d.record, -1, d.itemRows, legacy)
		}
		if err != nil || !bytes.Equal(legacy, want) {
			t.Fatalf("legacy panel differs, commands=%v err=%v", command, err)
		}
	}
	defer prepareHDChurchText(t, g, screen)()
	record, art := append([]byte(nil), d.record...), append([]byte(nil), d.base...)
	u.HP, u.NativeCommandMask = 0, [5]byte{}
	g.nativeCommandBook[0].MPCost = 99
	font := d.assets.Font
	for _, command := range []bool{false, true} {
		if !g.drawModernHDUnitStatus(screen, make([]byte, 64000), d, command, g.nativeUIPalette) {
			t.Fatal("HD status composition failed")
		}
		numbers, glyphs := 0, 0
		for _, r := range g.modernHDText.runs {
			if r.clip.Empty() || !r.clip.In(screen.Bounds()) {
				t.Fatal("text escaped the original panel")
			}
			if r.inkTop {
				numbers++
			} else {
				glyphs++
			}
		}
		if numbers == 0 || glyphs == 0 || g.font.indexedHD != nil || d.assets.Font != font {
			t.Fatal("missing glyphs/digits or leaked adapter")
		}
		after, ok := d.compose(d.assets, command)
		want := status
		if command {
			want = commands
		}
		if !ok || !bytes.Equal(want, after) || !bytes.Equal(record, d.record) || !bytes.Equal(art, d.base) {
			t.Fatal("HD draw or live changes mutated the open snapshot")
		}
	}
}

func TestModernHDStatusChurchAndPlayerKeepIndexedTransitions(t *testing.T) {
	g, screen, _ := newModernHDStatusGame(t)
	defer prepareHDChurchText(t, g, screen)()
	if !g.beginNativeChurchStatus(9) {
		t.Fatal("church status failed to open")
	}
	openingHasText := false
	for n := 0; g.nativeClassUIJob != nil; n++ {
		if n > 32 {
			t.Fatal("opening did not settle")
		}
		g.drawNativeClassUIJob(screen)
		openingHasText = openingHasText || len(g.modernHDText.runs) != 0
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if !openingHasText {
		t.Fatal("opening has no HD text")
	}
	if !g.drawNativeChurchStatus(screen) || len(g.modernHDText.runs) == 0 {
		t.Fatal("steady church panel has no HD text")
	}
	d := g.churchStatusData
	status, commands, ok := g.prepareNativeChurchStatus(9)
	if !ok {
		t.Fatal("player fixture unavailable")
	}
	steady, err := nativeChurchPanelFrame(make([]byte, 64000), status, 0)
	if err != nil {
		t.Fatal(err)
	}
	g.nativePlayerStatus = &nativePlayerStatusState{data: d, source: make([]byte, 64000),
		status: status, commands: commands, steady: steady, phase: "status"}
	g.drawNativePlayerStatus(screen)
	if len(g.modernHDText.runs) == 0 {
		t.Fatal("steady player panel has no HD text")
	}
	g.advanceNativePlayerStatus()
	transitionHasText := false
	for n := 0; n < 14; n++ {
		g.drawNativePlayerStatus(screen)
		transitionHasText = transitionHasText || len(g.modernHDText.runs) != 0
		g.stepNativePlayerStatus()
	}
	if !transitionHasText {
		t.Fatal("command transition has no HD text")
	}
	g.drawNativePlayerStatus(screen)
	if g.nativePlayerStatus.phase != "commands" || len(g.modernHDText.runs) == 0 {
		t.Fatal("player command page has no HD text")
	}
	g.returnToNativeStatusRoster()
	if g.churchStatusData != nil {
		t.Fatal("closed church snapshot was retained")
	}
}
