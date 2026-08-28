package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

func TestDrawNativeClassListUsesPlayerOriginalAssets(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	special := 0x34
	g := &Game{
		nativeClassUI: assets,
		churchMode:    "class",
		churchIDs:     []int{9},
		partyRoster: map[int]battle.Unit{
			9: {
				Name: "悠妮", Portrait: 9, ClassID: 5,
				NativeIdentity: 9, HasNativeIdentity: true,
				MapSelectorKey: 9, HasMapSelectorKey: true,
				NativeRecordRace: 1, HasNativeRecordRace: true,
				NativeRecordClass: 5, HasNativeRecordClass: true,
				NativeRecordByte5: 1, HasNativeRecordByte5: true,
				Lv: 4, HP: 0, MaxHP: 31,
				Inventory: []int{0x5a},
			},
			0: {
				Name: "索爾", Portrait: 0, ClassID: 0,
				NativeIdentity: 0, HasNativeIdentity: true,
				MapSelectorKey: 0, HasMapSelectorKey: true,
			},
		},
		classChangeTable: campaign.ClassChangeTable{
			Current: map[int]campaign.ClassChangeCurrent{9: {
				Portrait: 9, DefaultTarget: 0x29, SpecialItem: 0x5a, SpecialTarget: &special,
			}},
			Targets: map[int]campaign.ClassChangeTarget{
				0x29: {Portrait: 0x29, ClassID: 13},
				0x34: {Portrait: 0x34, ClassID: 21},
			},
		},
		nativeChurchTextIndex: 585,
		reviveFeeRates:        []int{0, 10, 20, 30, 40, 50},
		gold:                  1000,
	}
	screen := ebiten.NewImage(640, 400)
	if !g.drawNativeClassList(screen) {
		t.Fatal("native class list unexpectedly fell back")
	}
	if !g.beginNativeClassListOpening() || len(g.nativeClassUIJob.frames) != 6 {
		t.Fatal("native six-frame class list opening unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	if !g.beginNativeClassListClosing(nil) || len(g.nativeClassUIJob.frames) != 5 ||
		len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatal("native five-frame class list closing and source restore unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	g.churchMode, g.churchClassID, g.churchSel = "class_confirm", 9, 0
	if !g.drawNativeClassConfirmation(screen) {
		t.Fatal("native class confirmation unexpectedly fell back")
	}
	if !g.beginNativeClassConfirmationOpening() || len(g.nativeClassUIJob.frames) != 4 {
		t.Fatal("native four-frame confirmation opening unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	if !g.beginNativeClassConfirmationClosing(nil) || len(g.nativeClassUIJob.frames) != 9 ||
		len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatal("native confirmation choice/dialogue closing and source restore unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	g.churchMode, g.churchIDs, g.churchSel = "status_roster", []int{9}, 0
	if !g.drawNativeChurchRoster(screen) {
		t.Fatal("native two-column church roster unexpectedly fell back")
	}
	if !g.beginNativeChurchRosterOpening() || len(g.nativeClassUIJob.frames) != 6 {
		t.Fatal("native six-frame church roster opening unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	if !g.beginNativeChurchRosterClosing(nil) || len(g.nativeClassUIJob.frames) != 5 ||
		len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatal("native five-frame church roster closing and source restore unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	g.churchMode, g.churchSel = "transfer_item", 0
	g.churchTransferSource, g.churchTransferItems = 9, []int{0}
	g.nativeChurchTextIndex = 512
	if !g.drawNativeChurchTransferItem(screen) {
		t.Fatal("native transfer item list unexpectedly fell back")
	}
	if !g.beginNativeChurchTransferItemOpening() || len(g.nativeClassUIJob.frames) != 6 {
		t.Fatal("native transfer item six-frame opening unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	if !g.beginNativeChurchTransferItemClosing(nil) ||
		len(g.nativeClassUIJob.frames) != 5 || len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatal("native transfer item five-frame closing unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	g.churchMode, g.churchTransferDest = "transfer_full", 0
	g.nativeChurchTextIndex = 510
	if !g.drawNativeChurchTransferFull(screen) {
		t.Fatal("native transfer-full feedback unexpectedly fell back")
	}
	if !g.beginNativeChurchTransferFullOpening() || len(g.nativeClassUIJob.frames) != 6 {
		t.Fatal("native transfer-full six-frame dialogue opening unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	if !g.beginNativeChurchTransferFullClosing(nil) ||
		len(g.nativeClassUIJob.frames) != 5 || len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatal("native transfer-full five-frame dialogue closing unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	g.churchMode, g.churchIDs, g.churchSel = "revive", []int{9}, 0
	g.nativeChurchTextIndex = 589
	if !g.drawNativeChurchReviveList(screen) {
		t.Fatal("native revive candidate list unexpectedly fell back")
	}
	if !g.beginNativeChurchReviveListOpening() || len(g.nativeClassUIJob.frames) != 6 {
		t.Fatal("native revive candidate six-frame opening unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	if !g.beginNativeChurchReviveListClosing(nil) ||
		len(g.nativeClassUIJob.frames) != 5 || len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatal("native revive candidate five-frame closing unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	g.churchMode, g.churchReviveID, g.churchReviveFee, g.churchSel =
		"revive_confirm", 9, 200, 0
	if _, err := campaign.ComposeNativeReviveConfirmationQuestion(
		make([]byte, 320*200), assets.strings, assets.font, 10, 200,
	); err != nil {
		t.Fatalf("native revive question composition failed: %v", err)
	}
	if !g.drawNativeChurchReviveConfirmation(screen) {
		t.Fatal("native revive confirmation unexpectedly fell back")
	}
	if !g.beginNativeChurchReviveConfirmationOpening() || len(g.nativeClassUIJob.frames) != 4 {
		t.Fatal("native revive confirmation four-frame opening unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	if !g.beginNativeChurchReviveConfirmationClosing(nil) ||
		len(g.nativeClassUIJob.frames) != 9 || len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatal("native revive confirmation choice/dialogue closing unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	if !g.beginNativeChurchReviveChoiceClosing(nil) ||
		len(g.nativeClassUIJob.frames) != 4 || len(g.nativeClassUIJob.restore) != 0 {
		t.Fatal("native revive choice-only closing unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	if !g.beginNativeChurchReviveSuccess(nil) ||
		len(g.nativeClassUIJob.timeline) != 76 ||
		len(g.nativeClassUIJob.timeline[0].frame) != 320*200 ||
		len(g.nativeClassUIJob.timeline[75].frame) != 320*200 {
		t.Fatal("native revive success timeline unexpectedly fell back")
	}
	if g.nativeClassUIJob.timeline[9].palette[1] ==
		g.nativeClassUIJob.timeline[40].palette[1] {
		t.Fatal("native revive success palette rise did not change the DAC")
	}
	g.nativeClassUIJob = nil
	g.churchMode = "revive_insufficient"
	if !g.drawNativeChurchReviveMessage(screen) {
		t.Fatal("native insufficient-gold message unexpectedly fell back")
	}
	if !g.beginNativeChurchReviveMessageClosing(nil) ||
		len(g.nativeClassUIJob.frames) != 5 || len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatal("native insufficient-gold five-frame dialogue closing unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	g.churchMode, g.nativeChurchTextIndex = "revive_empty", 586
	if !g.drawNativeChurchReviveMessage(screen) {
		t.Fatal("native no-candidate message unexpectedly fell back")
	}
	g.openNativeChurchReviveEmpty()
	if g.churchMode != "revive_empty" || g.nativeClassUIJob == nil ||
		len(g.nativeClassUIJob.frames) != 6 {
		t.Fatal("native no-candidate six-frame dialogue opening unexpectedly fell back")
	}
	g.nativeClassUIJob = nil
	if !g.beginNativeChurchReviveMessageClosing(nil) ||
		len(g.nativeClassUIJob.frames) != 5 || len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatal("native no-candidate five-frame dialogue closing unexpectedly fell back")
	}
}

func TestLoadNativeClassUIUsesSeparatedFDTXT(t *testing.T) {
	const base = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(base, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(t.TempDir(), "missing-FDTXT.DAT"))
	t.Setenv("FD2_ASSET_PACK", "../../generated-assets/fd2-original-b97caf22")
	assets, err := loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	if assets.strings == nil || assets.strings.Count() != 661 {
		t.Fatalf("separated FDTXT_000 strings=%v", assets.strings)
	}
}
