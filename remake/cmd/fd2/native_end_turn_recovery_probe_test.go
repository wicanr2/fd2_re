package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// This E1 fixture imports raw state from one normal oracle checkpoint. Original
// pixels are never read here. The map and two END frames use production code.
func TestNativeEndTurnRecoverySameSourceProbe(t *testing.T) {
	run, prefix, slot := os.Getenv("FD2_END_PROBE_ORACLE"), os.Getenv("FD2_END_PROBE_PREFIX"), os.Getenv("FD2_END_PROBE_SLOT")
	if run == "" || prefix == "" || slot == "" {
		t.Skip("需要#29固定原版收據與輸出路徑")
	}
	stored, err := os.ReadFile(slot)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(stored)) != "ee9e412c0df245672e1644689f922cecb01e1b3abfd97a3d3334081921a963e5" {
		t.Fatal("固定槽不符", err)
	}
	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", canonicalCampaignReference)
	t.Setenv("FD2_NATIVE_SAVE", slot)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	g := loadGame()
	if g.loadErr != "" || !g.confirmTitleLoadSlot(0) {
		t.Fatal("正式LOAD失敗", g.loadErr)
	}
	r := &parityReplay{t: t, g: g, battle: "battle_ch04"}
	r.settleTown()
	for g.campSel != 2 {
		if !g.moveNativeTownSelection(1) {
			t.Fatal("出口選單")
		}
	}
	r.enterTownOption(2)
	if !pump(t, g, 600, func() bool { return !g.nativeClassUIBlocksInput() }) || !g.handleNativePreparationInput(nativePreparationInput{enter: true}) {
		t.Fatal("正式整備YES失敗")
	}
	if !pump(t, g, 600, func() bool { return g.camp.Node().Type != "preparation" }) {
		t.Fatal("沒有離開整備")
	}
	driveStory(t, g, newJourneyTrace(), func() bool { return g.camp.NodeID() == r.battle })
	if !pump(t, g, journeyStoryFrames, func() bool {
		if r.playerHasControl() {
			return true
		}
		if len(g.dialog) > 0 && storyEnterReady(g) {
			g.handleBattleEventDialogueInput(true)
		}
		return false
	}) {
		t.Fatal("正常出戰未交出控制")
	}
	var checkpoint struct {
		EXE   string `json:"exe_sha256"`
		Units []struct {
			Raw string `json:"raw_hex"`
		} `json:"units"`
		View struct {
			CameraX  int `json:"camera_x"`
			CameraY  int `json:"camera_y"`
			CursorX  int `json:"cursor_x"`
			CursorY  int `json:"cursor_y"`
			VisibleX int `json:"visible_x"`
			VisibleY int `json:"visible_y"`
		} `json:"view"`
	}
	raw, err := os.ReadFile(filepath.Join(run, "checkpoint-0196.json"))
	if err != nil || json.Unmarshal(raw, &checkpoint) != nil || checkpoint.EXE != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" {
		t.Fatal("正常來源checkpoint不符", err)
	}
	units := make([]*battle.Unit, len(checkpoint.Units))
	for i, row := range checkpoint.Units {
		b, err := hex.DecodeString(row.Raw)
		if err != nil || len(b) != 80 {
			t.Fatal("record", i, err)
		}
		var record fdsave.PersistentRecord
		copy(record.Raw[:], b)
		v := record.View()
		u := &battle.Unit{X: int(b[0]), Y: int(b[1]), Dir: int(b[3]), HP: int(v.HP), MaxHP: int(v.MaxHP),
			OnField: b[5]&1 == 0, Acted: b[5]&0x80 != 0, NativeRecordByte5: b[5], HasNativeRecordByte5: true,
			NativeRecordByte6: b[6], HasNativeRecordByte6: true, NativeTransient: v.Transient,
			MapSelectorKey: int(b[7]), HasMapSelectorKey: true, BattleFig: int(b[7]), HasBattleFig: true,
			NativeRecordRace: v.Race, HasNativeRecordRace: true, NativeRecordClass: v.Class, HasNativeRecordClass: true,
			NativeRecordWord42: uint16(v.MaxHP), HasNativeRecordWord42: true,
			NativeMapPresentation: battle.NativeMapPresentationState{X: b[0], Y: b[1], Pose: b[3], Motion: b[4]}, HasNativeMapPresentation: true,
		}
		switch b[6] {
		case 0:
			u.Camp = battle.Enemy
		case 1:
			u.Camp = battle.Ally
		case 2:
			u.Camp = battle.Own
		default:
			t.Fatal("raw camp")
		}
		units[i] = u
	}
	cache := &fdicon.NativeSelectorCache{}
	if err := battle.MaterializeNativeMapSelectorSlots(units, cache); err != nil {
		t.Fatal(err)
	}
	for i, u := range units {
		b, _ := hex.DecodeString(checkpoint.Units[i].Raw)
		if u.MapSelectorSlot != int(b[2]) {
			t.Fatal("selector slot", i)
		}
	}
	g.st.Units, g.st.NativeMapSelectorCache = units, cache
	v := checkpoint.View
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: v.CameraX, CameraY: v.CameraY, CursorX: v.CursorX, CursorY: v.CursorY, VisibleCursorX: v.VisibleX, VisibleCursorY: v.VisibleY}); err != nil {
		t.Fatal(err)
	}
	g.camX, g.camY = float64(v.CameraX*24), float64(v.CameraY*24)
	g.curX, g.curY = v.CursorX, v.CursorY
	// END disables future HUD redraws. It retains the already rendered cursor
	// and HUD in work, as seen at the initial 17AA9(1) frame925.
	g.st.NativeMapHUDState.DisplayGateB = 1
	g.st.MaterializeNativeMapRangeMode(1)
	frozen := time.Unix(1000, 0)
	g.nativeMapFrozenNow = func() time.Time { return frozen }
	dump := func(name string, pixels []byte) {
		palette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
		if err != nil {
			t.Fatal(err)
		}
		pic := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
		copy(pic.Pix, pixels)
		f, err := os.Create(prefix + "-" + name + ".png")
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, pic)
		closeErr := f.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
	}
	// These globals are not captured by checkpoint JSON. Enumerate exactly
	// their closed legal states; the comparison receipt records the chosen pair.
	for terrain := 0; terrain < 4; terrain++ {
		for idle := 0; idle < 3; idle++ {
			g.st.NativeMapHUDState.DisplayGateB = 1
			g.nativeMapClock.Reset()
			g.st.NativeMapCycleState.Idle = idle
			g.st.NativeMapCycleState.Moving = 0
			g.st.NativeTerrainPhaseState.Phase = terrain
			if err := g.composeNativeMapFrameAt(frozen); err != nil {
				t.Fatal(err)
			}
			if err := g.beginNativeEndTurnRecovery(nil); err != nil {
				t.Fatal(err)
			}
			job := g.nativeEndTurnRecovery
			for frame, pixels := range job.frames {
				dump(fmt.Sprintf("t%d-i%d-f%d", terrain, idle, frame), pixels)
			}
			g.nativeEndTurnRecovery = nil
		}
	}
	// The same state must reach this owner through END/YES after its closing
	// frames; only the transaction fixture above supplies wounded unit records.
	g.st.MaterializeNativeMapRangeMode(1)
	if err := g.composeNativeMapFrameAt(frozen); err != nil {
		t.Fatal(err)
	}
	g.nativeSystemCursorOverlay = true
	g.beginActionOverlayOpen(3)
	screen := ebiten.NewImage(640, 400)
	pumpUI := func(limit int, done func() bool) bool {
		for frame := 0; frame < limit; frame++ {
			if done() {
				return true
			}
			g.Draw(screen)
			if err := g.Update(); err != nil {
				t.Fatal(err)
			}
			if g.loadErr != "" {
				t.Fatal(g.loadErr)
			}
		}
		return done()
	}
	if !pumpUI(20, func() bool { return !g.actionOverlayBlocksInput() }) || !g.beginNativeSystemEndTurn() {
		t.Fatalf("正式END入口：phase=%s ring=%v sel=%d overlay=%v prep=%v class=%v VGA=%d err=%s", g.actionOverlayPhase, g.ring, g.ringSel, g.nativeSystemCursorOverlay, g.nativePreparationUI != nil, g.nativeClassUI != nil, len(g.nativeMapVGA), g.loadErr)
	}
	if !pumpUI(100, func() bool { return g.nativeSystemEndTurnConfirm && g.nativeClassUIJob == nil }) {
		t.Fatal("END問題未完成")
	}
	g.confirmNativeSystemEndTurn()
	if !pumpUI(100, func() bool { return g.nativeEndTurnRecovery != nil }) {
		t.Fatal("YES未交給回復owner")
	}
	job := g.nativeEndTurnRecovery
	g.Draw(screen)
	g.stepNativeEndTurnRecovery(job.readyAt)
	g.Draw(screen)
	g.Update()
	if units[0].HP != 108 || units[4].HP != 110 || units[6].HP != 26 || job.cueCount != 1 {
		t.Fatal("正式YES數值/cue")
	}
	for _, i := range []int{0, 4, 6} {
		if units[i].NativeRecordByte5 != 0x80 || !units[i].Acted {
			t.Fatal("已行動投影", i)
		}
	}
	receipt := map[string]any{"status": "passed", "kind": "END same-source E1 fixture", "slot_sha256": fmt.Sprintf("%x", sha256.Sum256(stored)), "checkpoint_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "original_frame_indices": []int{925, 926, 927}, "phase_method": "terrain0..3 × idle0..2; 未同步時間，原版像素只作比較目標", "changes": [][3]int{{0, 87, 108}, {4, 89, 110}, {6, 14, 26}}, "cue_count": job.cueCount, "formal_end_yes_owner": true, "limit": "正常LOAD與出戰後匯入原版seq196 raw狀態的E1 fixture；不宣稱整段同狀態GUI E2"}
	out, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(prefix+".json", append(out, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
