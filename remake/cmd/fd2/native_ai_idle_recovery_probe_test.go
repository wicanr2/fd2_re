package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
)

// E1 fixture: import an observed raw state after normal LOAD/departure. This
// probe never reads original pixels or draws a screenshot into production.
func TestNativeAIIdleRecoverySameSourceProbe(t *testing.T) {
	slot, checkpointPath, prefix := os.Getenv("FD2_AI_IDLE_SLOT"), os.Getenv("FD2_AI_IDLE_CHECKPOINT"), os.Getenv("FD2_AI_IDLE_PREFIX")
	if slot == "" || checkpointPath == "" || prefix == "" {
		t.Skip("需要 #163 固定槽、原版停點與輸出路徑")
	}
	actorIndex, err := strconv.Atoi(os.Getenv("FD2_AI_IDLE_ACTOR"))
	if err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(slot)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(stored)) != "768a561e8a713cd4f7f6fa6f36e553c7330bd60cc122bca03d8654a037d723a0" {
		t.Fatal("固定 ch08 槽不符", err)
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
		t.Fatal("正常 LOAD", g.loadErr)
	}
	r := &parityReplay{t: t, g: g, battle: "battle_ch08"}
	r.settleTown()
	for g.campSel != 2 {
		if !g.moveNativeTownSelection(1) {
			t.Fatal("城鎮出口")
		}
	}
	r.enterTownOption(2)
	if !pump(t, g, 600, func() bool { return !g.nativeClassUIBlocksInput() }) || !g.handleNativePreparationInput(nativePreparationInput{enter: true}) {
		t.Fatal("正常出戰 YES")
	}
	if !pump(t, g, 600, func() bool { return g.camp.Node().Type != "preparation" }) {
		t.Fatal("出戰提示未收尾")
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
		t.Fatal("正常戰場控制", g.loadErr)
	}
	var checkpoint struct {
		EXE        string `json:"exe_sha256"`
		Injections []any  `json:"state_injections"`
		Units      []struct {
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
	raw, err := os.ReadFile(checkpointPath)
	if err != nil || json.Unmarshal(raw, &checkpoint) != nil || checkpoint.EXE != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" || len(checkpoint.Injections) != 0 {
		t.Fatal("正常原版停點不符", err)
	}
	units := make([]*battle.Unit, len(checkpoint.Units))
	for i, row := range checkpoint.Units {
		b, err := hex.DecodeString(row.Raw)
		if err != nil || len(b) != 80 {
			t.Fatal("raw record", i, err)
		}
		var record fdsave.PersistentRecord
		copy(record.Raw[:], b)
		v := record.View()
		word := func(offset int) int { return int(binary.LittleEndian.Uint16(b[offset : offset+2])) }
		u := &battle.Unit{
			X: int(b[0]), Y: int(b[1]), Dir: int(b[3]), HP: word(0x40), MaxHP: word(0x42), MP: word(0x44), MaxMP: word(0x46),
			AP: word(0x48), DP: word(0x4a), HIT: word(0x4c), EV: word(0x4e), DX: word(0x3e), MV: int(b[0x3b]), Lv: int(b[0x21]),
			OnField: b[5]&1 == 0, Acted: b[5]&0x80 != 0, NativeRecordByte5: b[5], HasNativeRecordByte5: true,
			NativeRecordByte6: b[6], HasNativeRecordByte6: true, NativeRecordByte8: b[8], HasNativeRecordByte8: true,
			NativeRecordByte34: b[0x34], HasNativeRecordByte34: true, NativeRecordByte35: b[0x35], HasNativeRecordByte35: true,
			NativeRecordByte36: b[0x36], HasNativeRecordByte36: true, NativeRecordWord42: uint16(word(0x42)), HasNativeRecordWord42: true,
			NativeRecordWord46: uint16(word(0x46)), HasNativeRecordWord46: true, NativeTransient: v.Transient,
			MapSelectorKey: int(b[7]), HasMapSelectorKey: true, BattleFig: int(b[7]), HasBattleFig: true,
			NativeRecordRace: v.Race, HasNativeRecordRace: true, NativeRecordClass: v.Class, HasNativeRecordClass: true,
			NativeMapPresentation: battle.NativeMapPresentationState{X: b[0], Y: b[1], Pose: b[3], Motion: b[4]}, HasNativeMapPresentation: true,
			InventorySlots: make([]int, 8), NativeInventoryFlags: make([]int, 8),
		}
		copy(u.NativeCommandMask[:], b[0x1a:0x1f])
		for j := 0; j < 8; j++ {
			u.NativeInventoryFlags[j], u.InventorySlots[j] = int(b[0x0a+2*j]), int(b[0x0b+2*j])
		}
		switch b[6] {
		case 0:
			u.Camp = battle.Enemy
		case 1:
			u.Camp = battle.Ally
		case 2:
			u.Camp = battle.Own
		default:
			t.Fatal("raw camp", b[6])
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
	if actorIndex < 0 || actorIndex >= len(units) {
		t.Fatal("actor index")
	}
	g.st.Units, g.st.NativeMapSelectorCache = units, cache
	v := checkpoint.View
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: v.CameraX, CameraY: v.CameraY, CursorX: v.CursorX, CursorY: v.CursorY, VisibleCursorX: v.VisibleX, VisibleCursorY: v.VisibleY}); err != nil {
		t.Fatal(err)
	}
	g.camX, g.camY = float64(v.CameraX*24), float64(v.CameraY*24)
	g.curX, g.curY = v.CursorX, v.CursorY
	g.st.MaterializeNativeMapRangeMode(0)
	g.st.NativeMapHUDState.DisplayGateB = 0
	frozen := time.Unix(1000, 0)
	g.nativeMapFrozenNow = func() time.Time { return frozen }
	records, err := battle.NativeAIScoringRecords(units)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := battle.PlanNativeAIIdleRecovery(records, len(units), actorIndex)
	if err != nil || !decision.Accepted {
		t.Fatal("原版候選", decision, err)
	}
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
	actor := units[actorIndex]
	for terrain := 0; terrain < 4; terrain++ {
		for idle := 0; idle < 3; idle++ {
			g.nativeMapClock.Reset()
			g.st.NativeMapCycleState.Idle, g.st.NativeMapCycleState.Moving = idle, 0
			g.st.NativeTerrainPhaseState.Phase = terrain
			if err := g.beginNativeAIIdleRecovery(actor, decision, nil); err != nil {
				t.Fatal(err)
			}
			for frame, pixels := range g.nativeAIIdleRecovery.frames {
				dump(fmt.Sprintf("t%d-i%d-f%d", terrain, idle, frame), pixels)
			}
			g.nativeAIIdleRecovery = nil
		}
	}
	finished := false
	if err := g.beginNativeAIIdleRecovery(actor, decision, func() { finished = true }); err != nil {
		t.Fatal(err)
	}
	screen := ebiten.NewImage(640, 400)
	for stage := 0; stage < 3; stage++ {
		if actor.HP != int(decision.CurrentHP) || finished {
			t.Fatal("HP 提前提交")
		}
		g.Draw(screen)
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if actor.HP != int(decision.NextHP) || !finished || g.loadErr != "" {
		t.Fatal("最後HP交易", actor.HP, finished, g.loadErr)
	}
	receipt := map[string]any{"kind": "AI idle recovery same-source E1 fixture", "status": "passed", "slot_sha256": fmt.Sprintf("%x", sha256.Sum256(stored)), "checkpoint_sha256": fmt.Sprintf("%x", sha256.Sum256(raw)), "actor": actorIndex, "before_hp": decision.CurrentHP, "after_hp": decision.NextHP, "phase_method": "terrain0..3 × idle0..2；未同步時間，原版像素只供比較", "three_real_draws_before_commit": true, "limit": "正常LOAD／出戰後匯入原版raw停點的E1，不宣稱整章PLAYER-E2"}
	out, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prefix+".json", append(out, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
