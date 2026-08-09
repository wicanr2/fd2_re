// beatrunner_test.go — BeatRunner(doc50)純邏輯測試:不碰 ebiten 顯示/輸入,
// 只驗證 beatStart/beatAdvance 與 stepCamPan/stepStoryWalks/stepActJob/stepFocusUnit/stepFade
// 這幾個「逐幀推進」method 的狀態機是否照 op 表正確銜接。
package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
)

// newBeatTestGame 建最小 Game:一張假地圖(供 storyWalkJob 的 tile 換算用)+ 一個
// cutscene 節點(帶 beats)轉場到一個空白 ending 節點(無 map/bgm,enterNode 不碰檔案系統)。
func newBeatTestGame(t *testing.T, beats []campaign.Beat) *Game {
	t.Helper()
	c := &campaign.Campaign{
		Start: "cs",
		Nodes: map[string]*campaign.Node{
			"cs": {
				Type:  "cutscene",
				Beats: beats,
				Actors: []campaign.Actor{
					{Fig: 0, X: 1, Y: 1},
					{Fig: 4, X: 2, Y: 2},
				},
				Next: "end",
			},
			"end": {Type: "ending", Text: "完"},
		},
	}
	g := &Game{
		m:    &MapData{W: 20, H: 20, TileW: 24, TileH: 24, Cols: 8, Tiles: make([]int, 400)},
		camp: campaign.NewRunner(c),
	}
	g.campLines = []campaign.Line{
		{Speaker: 0, Text: "第一句"},
		{Speaker: 4, Text: "第二句"},
		{Speaker: 0, Text: "第三句"},
	}
	// enterNode 的「story/cutscene」分支需要 storyActors 才能讓 walk/act 拍找得到 Fig;
	// 直接照 enterNode 的 Actors 初始化邏輯手動掛上(不呼叫完整 enterNode,避免觸發
	// loadMap/playBGM 等與本測試無關的 I/O)。
	for _, a := range c.Nodes["cs"].Actors {
		g.storyActors = append(g.storyActors, battle.Unit{Fig: a.Fig, X: a.X, Y: a.Y, OnField: true})
	}
	g.storyBG = true
	g.beats = beats
	g.beatIdx = -1
	return g
}

// tick 手動跑一輪「Update 會做的 BeatRunner 相關步驟」,次數可控,方便測試逐幀推進。
func (g *Game) tick(n int) {
	for i := 0; i < n; i++ {
		g.stepNativeSpawnIntro()
		g.stepStoryWalks()
		g.stepActJob()
		g.stepFocusUnit()
		g.stepFade()
		g.stepNativePaletteRamp()
		g.stepCamPan()
		if g.beatDelay > 0 {
			g.beatDelay--
			if g.beatDelay == 0 {
				g.beatAdvance()
			}
		}
	}
}

func TestBeatPaletteUpdatePresentsRecoveredFullDACWhiteFlash(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "palette_update", PaletteStart: 0, PaletteEnd: 255, PaletteDelta: 255},
		{Op: "delay", Ms: 200},
		{Op: "palette_update", PaletteStart: 0, PaletteEnd: 255, PaletteDelta: 0},
	})
	g.beatAdvance()
	if !g.nativeFullDACWhite || g.beatDelay != 12 || g.loadErr != "" {
		t.Fatalf("white flash start white=%v delay=%d err=%q", g.nativeFullDACWhite, g.beatDelay, g.loadErr)
	}
	g.tick(12)
	if g.nativeFullDACWhite || g.loadErr != "" {
		t.Fatalf("white flash restore white=%v err=%q", g.nativeFullDACWhite, g.loadErr)
	}

	rejected := newBeatTestGame(t, []campaign.Beat{{
		Op: "palette_update", PaletteStart: 1, PaletteEnd: 255, PaletteDelta: 255,
	}})
	rejected.beatAdvance()
	if rejected.loadErr == "" || rejected.nativeFullDACWhite {
		t.Fatalf("partial non-indexed update err=%q white=%v", rejected.loadErr, rejected.nativeFullDACWhite)
	}
}

func TestBeatPanMovesCamera(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "pan", X: 100, Y: 200, Frames: 10},
	})
	g.beatAdvance() // 啟動第 0 拍
	g.camMaxY = 50  // follow-only cap must not corrupt an explicit original PAN target.
	if g.camPan == nil {
		t.Fatal("pan 拍應設定 camPan")
	}
	g.tick(10)
	if g.camX != 100 || g.camY != 200 {
		t.Fatalf("pan 走完應到 (100,200),得 (%v,%v)", g.camX, g.camY)
	}
	if g.camPan != nil {
		t.Fatal("pan 走完應清除 camPan")
	}
}

func TestFastForwardShotCampaignCompletesBlockingBeats(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "pan", X: 100, Y: 200, Frames: 10},
		{Op: "dialog", Line: 0},
		{Op: "delay", Frames: 5},
	})
	g.camp.C.Nodes["end"].Type = "battle"
	g.beatAdvance()
	if err := g.fastForwardShotCampaign(); err != nil {
		t.Fatalf("screenshot fast-forward failed: %v", err)
	}
	if g.loadErr != "" || g.camp.Cur != "end" || g.camX != 100 || g.camY != 200 || len(g.dialog) != 0 {
		t.Fatalf("fast-forward state node=%q camera=(%v,%v) dialog=%d err=%q", g.camp.Cur, g.camX, g.camY, len(g.dialog), g.loadErr)
	}
}

func TestBeatPanTileStepMatchesOriginalXFirstRedrawOrder(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{{Op: "pan", X: 72, Y: 48, TileStep: true}})
	g.camX, g.camY = 0, 0
	g.beatAdvance()
	g.tick(2)
	if g.camX != 48 || g.camY != 0 || g.camPan == nil {
		t.Fatalf("0x135dd first two redraws camera=(%v,%v) job=%#v, want X-only (48,0)", g.camX, g.camY, g.camPan)
	}
	g.tick(1)
	if g.camX != 72 || g.camY != 0 {
		t.Fatalf("0x135dd must finish X before Y: (%v,%v)", g.camX, g.camY)
	}
	g.tick(1)
	if g.camX != 72 || g.camY != 24 {
		t.Fatalf("first Y redraw = (%v,%v), want (72,24)", g.camX, g.camY)
	}
	g.tick(1)
	if g.camX != 72 || g.camY != 48 || g.camPan != nil {
		t.Fatalf("tile-step pan finish = (%v,%v) job=%#v", g.camX, g.camY, g.camPan)
	}
}

func TestBeatWalkMovesActorAndAdvances(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "walk", Fig: 0, X: 5, Y: 1, Frames: 5, Follow: true},
		{Op: "dialog", Line: 0, Count: 1},
	})
	g.beatAdvance()
	if len(g.storyWalks) != 1 {
		t.Fatalf("walk 拍應建立 1 個 storyWalks job,得 %d", len(g.storyWalks))
	}
	if !g.followWalk {
		t.Fatal("walk 拍 Follow=true 應設定 g.followWalk")
	}
	g.tick(5)
	if len(g.storyWalks) != 0 {
		t.Fatal("走完應清空 storyWalks")
	}
	u := &g.storyActors[0]
	if u.X != 5 || u.Y != 1 {
		t.Fatalf("索爾應走到 (5,1),得 (%d,%d)", u.X, u.Y)
	}
	if len(g.dialog) != 1 { // 走完應自動接下一拍(dialog),推入第 0 句
		t.Fatalf("walk 完成應接 dialog 拍,g.dialog 應有 1 句,得 %d", len(g.dialog))
	}
}

func TestBeatDialogCountConsecutiveLines(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "dialog", Line: 0, Count: 2},
	})
	g.beatAdvance()
	if len(g.dialog) != 2 {
		t.Fatalf("count=2 應推入 2 句,得 %d", len(g.dialog))
	}
	if g.dialog[len(g.dialog)-1].Text != "第一句" { // 反序堆疊,末端=先顯示那句
		t.Fatalf("堆疊末端應是第一句,得 %q", g.dialog[len(g.dialog)-1].Text)
	}
	// 模擬玩家逐句 Enter(campInput cutscene 分支的邏輯)
	g.dialog = g.dialog[:len(g.dialog)-1]
	if len(g.dialog) != 1 {
		t.Fatal("pop 一次應剩 1 句")
	}
	g.dialog = g.dialog[:len(g.dialog)-1]
	if len(g.dialog) != 0 {
		t.Fatal("pop 兩次應清空")
	}
	g.beatAdvance() // 對白播完,序列只有 1 拍,應跑到收尾(進入淡出)
	if g.fade == nil {
		t.Fatal("beats 跑完應觸發 advanceStoryNode 的淡出轉場")
	}
}

func TestBeatDialogResolvesOriginalRuntimeSpeakerSlot(t *testing.T) {
	slot := 6
	upper := true
	g := newBeatTestGame(t, []campaign.Beat{{Op: "dialog", Line: 0}})
	g.campLines = []campaign.Line{{Speaker: 133, SpeakerSlot: &slot, Upper: &upper, Text: "謝謝各位的幫助"}}
	g.st = &battle.State{Units: make([]*battle.Unit, 7)}
	g.st.Units[6] = &battle.Unit{Portrait: 134}
	g.beatAdvance()
	if g.loadErr != "" || len(g.dialog) != 1 || g.dialog[0].Speaker != 134 || g.dialog[0].Upper == nil || !*g.dialog[0].Upper {
		t.Fatalf("runtime speaker slot did not resolve unit portrait/box: err=%q dialog=%#v", g.loadErr, g.dialog)
	}
	cutsceneSlot := 1
	cutscene := newBeatTestGame(t, []campaign.Beat{{Op: "dialog", Line: 0}})
	cutscene.campLines = []campaign.Line{{Speaker: 96, SpeakerSlot: &cutsceneSlot, Text: "場景單位"}}
	cutscene.storyActors[1].Portrait = 133
	cutscene.beatAdvance()
	if cutscene.loadErr != "" || len(cutscene.dialog) != 1 || cutscene.dialog[0].Speaker != 133 {
		t.Fatalf("cutscene speaker slot did not resolve materialized actor: err=%q dialog=%#v", cutscene.loadErr, cutscene.dialog)
	}

	bad := newBeatTestGame(t, []campaign.Beat{{Op: "dialog", Line: 0}})
	bad.campLines = []campaign.Line{{Speaker: 133, SpeakerSlot: &slot, Text: "不可猜頭像"}}
	bad.st = &battle.State{Units: make([]*battle.Unit, 6)}
	bad.beatAdvance()
	if bad.loadErr == "" || len(bad.dialog) != 0 {
		t.Fatalf("missing direct speaker slot must fail closed: err=%q dialog=%#v", bad.loadErr, bad.dialog)
	}
}

func TestChapter2StoryPreservesDirectSpeakerSlotsAndBoxSides(t *testing.T) {
	battleLines := loadStoryScript("assets/story/ch02.json", "戰鬥中,強盜兵分兩路")
	if len(battleLines) != 12 {
		t.Fatalf("ch02 battle lines = %d, want 12", len(battleLines))
	}
	for line, slot := range map[int]int{0: 21, 1: 22, 2: 23, 3: 8, 8: 7, 10: 6, 11: 6} {
		if battleLines[line].SpeakerSlot == nil || *battleLines[line].SpeakerSlot != slot || battleLines[line].Upper == nil || !*battleLines[line].Upper {
			t.Errorf("ch02 battle line %d direct speaker = %#v, want upper slot %d", line, battleLines[line], slot)
		}
	}
	postLines := loadStoryScript("assets/story/ch02.json", "希莉亞登場")
	if len(postLines) != 23 || postLines[0].Upper == nil || !*postLines[0].Upper || postLines[1].Upper == nil || *postLines[1].Upper {
		t.Fatalf("ch02 post line count/box sides = %#v", postLines)
	}
	casualties := loadStoryScript("assets/story/ch02.json", "戰鬥受創短句")
	if len(casualties) != 6 {
		t.Fatalf("ch02 casualty lines = %d, want 6", len(casualties))
	}
	for i, line := range casualties {
		if line.SpeakerSlot == nil || *line.SpeakerSlot != i+5 || line.Speaker != []int{134, 133, 134, 133, 134, 133}[i] {
			t.Errorf("casualty line %d = %#v", i, line)
		}
	}
}

func TestBeatActCyclesPosesThenAdvances(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "act", Fig: 4, Poses: []int{1, 2, 3}, PoseFrames: 3},
		{Op: "dialog", Line: 2, Count: 1},
	})
	g.beatAdvance()
	if g.actJob == nil {
		t.Fatal("act 拍應建立 actJob")
	}
	g.tick(3 * 3) // 3 個 pose × 3 幀
	if g.actJob != nil {
		t.Fatal("pose 序列跑完應清除 actJob")
	}
	u := &g.storyActors[1] // fig=4 對映 storyActors[1]
	if u.Dir != 3 {
		t.Fatalf("最後一個 pose 應停在 3,得 %d", u.Dir)
	}
	if len(g.dialog) != 1 || g.dialog[0].Text != "第三句" {
		t.Fatalf("act 完成應接下一拍(dialog line=2),得 %+v", g.dialog)
	}
}

func TestBeatActingNormalFrameMovesEachBeat(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "act", Acting: []campaign.ActingFrame{{
			Beats: 3,
			Units: []campaign.ActingUnit{{Fig: 0, Pose: 3}}, // 右三格
		}}},
		{Op: "dialog", Line: 1, Count: 1},
	})
	g.beatAdvance()
	if g.actJob == nil {
		t.Fatal("acting frame 應建立 actJob")
	}
	g.tick(6)
	if u := g.storyActors[0]; u.X != 1 || u.OffX <= 0 || u.Dir != 3 {
		t.Fatalf("第 6 tick 應仍在第一格內插，得 X=%d OffX=%v Dir=%d", u.X, u.OffX, u.Dir)
	}
	g.tick(15) // 3 格 × 每格 7 tick，合計 21
	if g.actJob != nil {
		t.Fatal("正常 acting 的全部 beat 後應結束")
	}
	u := g.storyActors[0]
	if u.X != 4 || u.Y != 1 || u.OffX != 0 || u.OffY != 0 || u.Dir != 3 {
		t.Fatalf("右三格後應為 (4,1) 且定格，得 (%d,%d) off=(%v,%v) dir=%d", u.X, u.Y, u.OffX, u.OffY, u.Dir)
	}
	if len(g.dialog) != 1 || g.dialog[0].Text != "第二句" {
		t.Fatalf("acting 結束應接下一 dialog，得 %+v", g.dialog)
	}
}

func TestBeatActingSpecialFrameStaysPut(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "act", Acting: []campaign.ActingFrame{{
			Beats:   5,
			Special: true,
			Units:   []campaign.ActingUnit{{Fig: 4, Pose: 1}},
		}}},
	})
	g.beatAdvance()
	g.tick(5)
	u := g.storyActors[1]
	if u.X != 2 || u.Y != 2 || u.OffX != 0 || u.OffY != 0 || u.Dir != 1 {
		t.Fatalf("special acting 必須原地面左，得 (%d,%d) off=(%v,%v) dir=%d", u.X, u.Y, u.OffX, u.OffY, u.Dir)
	}
}

func TestBeatActingUsesOriginalSlotBeforeDuplicateFig(t *testing.T) {
	// Synthetic duplicate-Fig regression: decoded resources are slot-addressed,
	// so a Fig-only lookup must never redirect either target to the first guard.
	slot1, slot2 := 1, 2
	frames := []campaign.ActingFrame{{
		Beats: 3, Special: true,
		Units: []campaign.ActingUnit{{Slot: &slot1, Pose: 1}, {Slot: &slot2, Pose: 2}},
	}}
	g := newBeatTestGame(t, []campaign.Beat{{Op: "act", Acting: frames}})
	g.storyActors = make([]battle.Unit, 3)
	for i := range g.storyActors {
		g.storyActors[i] = battle.Unit{Fig: 69, OnField: true, Dir: 3}
	}
	g.beatAdvance()
	g.tick(3)
	if g.storyActors[0].Dir != 3 {
		t.Fatalf("duplicate fig fallback touched slot0: dir=%d", g.storyActors[0].Dir)
	}
	if g.storyActors[1].Dir != 1 || g.storyActors[2].Dir != 2 {
		t.Fatalf("decoded slots 1/2 were not targeted: dirs=(%d,%d)", g.storyActors[1].Dir, g.storyActors[2].Dir)
	}
}

func TestBeatActingZeroSpecialPreservesOriginalThreeTickTransition(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{{Op: "act", Acting: []campaign.ActingFrame{{
		Beats: 0, Special: true,
		Units: []campaign.ActingUnit{{Fig: 0, Pose: 2}},
	}}}})
	g.beatAdvance()
	if g.storyActors[0].Dir != 2 {
		t.Fatalf("zero-special frame must apply native pose before delay, got dir=%d", g.storyActors[0].Dir)
	}
	g.tick(2)
	if g.actJob == nil {
		t.Fatal("zero-special frame advanced before original delay(1)+delay(2)")
	}
	g.tick(1)
	if g.actJob != nil {
		t.Fatal("zero-special frame did not advance after three ticks")
	}
}

func TestBeatNativePaletteFadeOutFailsClosedWithoutIndexedInput(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{{
		Op: "native_palette_fade_out", NativePaletteFade: &campaign.NativePaletteFadeOut{Start: 0, End: 63, DelayMs: 2},
	}})
	g.beatAdvance()
	if g.loadErr != "beat native_palette_fade_out: native indexed framebuffer or DAC baseline unavailable" {
		t.Fatalf("native palette fade error=%q", g.loadErr)
	}
}

func TestBeatNativePalettePulseFailsClosedWithoutIndexedDACAdapter(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{{
		Op: "native_palette_pulse", NativePalettePulse: &campaign.NativePalettePulse{
			RiseStart: 0, RiseEnd: 63, RiseDelayMs: 8, HoldMs: 400, FallStart: 62, FallEnd: 0, FallDelayMs: 8,
		},
	}})
	g.beatAdvance()
	if g.loadErr != "beat native_palette_pulse: native indexed DAC adapter未完成" {
		t.Fatalf("native palette pulse error=%q", g.loadErr)
	}
}

func TestBeatCh07PostBlackoutCoversWholeSurfaceAndRejectsOtherShapes(t *testing.T) {
	payload := &campaign.NativePaletteBlackout{Start: 0, End: 255, Delta: 64, ClearBytes: 0xFA00}
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "native_palette_blackout", Source: "0x23599", NativePaletteBlackout: payload},
		{Op: "delay", Frames: 1},
	})
	g.nativeMapVGA = make([]byte, 0xFA00)
	for i := range g.nativeMapVGA {
		g.nativeMapVGA[i] = 0x7f
	}
	g.beatAdvance()
	if !g.nativeFullDACBlack || g.beatDelay != 1 || g.loadErr != "" {
		t.Fatalf("blackout black=%v delay=%d err=%q", g.nativeFullDACBlack, g.beatDelay, g.loadErr)
	}
	for i, value := range g.nativeMapVGA {
		if value != 0 {
			t.Fatalf("blackout framebuffer[%d]=%d", i, value)
		}
	}

	rejected := newBeatTestGame(t, []campaign.Beat{{
		Op: "native_palette_blackout", Source: "0x23598", NativePaletteBlackout: payload,
	}})
	rejected.beatAdvance()
	if rejected.loadErr == "" || rejected.nativeFullDACBlack {
		t.Fatalf("non-proven blackout err=%q black=%v", rejected.loadErr, rejected.nativeFullDACBlack)
	}

	short := newBeatTestGame(t, []campaign.Beat{{
		Op: "native_palette_blackout", Source: "0x23599", NativePaletteBlackout: payload,
	}})
	short.nativeMapVGA = make([]byte, 0xF9FF)
	short.beatAdvance()
	if short.loadErr == "" || short.nativeFullDACBlack {
		t.Fatalf("short framebuffer err=%q black=%v", short.loadErr, short.nativeFullDACBlack)
	}
}

func TestBeatClearNativeRecordBit7PreservesOtherBits(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{{Op: "clear_native_record_bit7"}})
	g.st = &battle.State{Units: []*battle.Unit{
		{HasNativeRecordByte5: true, NativeRecordByte5: 0xff},
		{HasNativeRecordByte5: true, NativeRecordByte5: 0x81},
	}}
	g.beatAdvance()
	if g.loadErr != "" || g.st.Units[0].NativeRecordByte5 != 0x7f || g.st.Units[1].NativeRecordByte5 != 0x01 {
		t.Fatalf("raw bit7 clear mutated wrong bytes: err=%q bytes=%#x/%#x", g.loadErr, g.st.Units[0].NativeRecordByte5, g.st.Units[1].NativeRecordByte5)
	}
}

func TestBeatNativeStagingPresentFailsClosedWithoutRendererAdapter(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{{Op: "native_staging_present", NativeStagingPresent: &campaign.NativeStagingPresent{Slot: 22, X: 23, Y: 5, FocusX: 22, FocusY: 23}}})
	g.beatAdvance()
	if g.loadErr != "beat native_staging_present: native 0x22253 renderer adapter未完成" {
		t.Fatalf("native staging present error=%q", g.loadErr)
	}
}

func TestBeatNativeCh23LoopFailsClosedWithoutIndexedLatchAdapter(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{{Op: "native_ch23_loop", NativeCh23Loop: &campaign.NativeCh23Loop{Phase: "initial"}}})
	g.beatAdvance()
	if g.loadErr != "beat native_ch23_loop: native ch23 indexed/latch renderer adapter未完成" {
		t.Fatalf("native ch23 loop error=%q", g.loadErr)
	}
}

func TestBeatActingDecodedNormalSlotMovement(t *testing.T) {
	// Direct resource102 at ch00 source 0x32461: slot4 left×2, up×1,
	// left×1. A duplicate Fig at slot0 must remain untouched.
	resources, err := campaign.LoadActingResourceSet(assetPath("assets/cutscenes/acting/map32.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := newBeatTestGame(t, []campaign.Beat{{Op: "act", Acting: resources[102]}})
	g.storyActors = make([]battle.Unit, 5)
	g.storyActors[0] = battle.Unit{Fig: 4, X: 9, Y: 5, OnField: true}
	g.storyActors[4] = battle.Unit{Fig: 4, X: 10, Y: 10, OnField: true}
	g.beatAdvance()
	g.tick(28) // four normal grid beats × seven ticks
	if got := g.storyActors[0]; got.X != 9 || got.Y != 5 {
		t.Fatalf("same-Fig slot0 moved instead of slot4: (%d,%d)", got.X, got.Y)
	}
	if got := g.storyActors[4]; got.X != 7 || got.Y != 9 || got.Dir != 1 {
		t.Fatalf("slot4 decoded movement = (%d,%d) dir=%d, want (7,9) dir=1", got.X, got.Y, got.Dir)
	}
}

func TestChapter1PostRuntimeContextSpawnsAndActsOnCanonicalBattleSlots(t *testing.T) {
	st, err := battle.Load(assetPath("assets/maps/map1/map1_units.json"))
	if err != nil {
		t.Fatal(err)
	}
	sc, err := battle.LoadScenario(assetPath("assets/scenarios/ch02.json"))
	if err != nil {
		t.Fatal(err)
	}
	sc.Setup(st)
	st.SpawnGroup(3, battle.Ally, true, false)
	if len(st.Units) != 27 {
		t.Fatalf("pre-post runtime slots=%d, want 27", len(st.Units))
	}
	resources, err := campaign.LoadActingResourceSet(assetPath("assets/cutscenes/acting/map32.json"))
	if err != nil {
		t.Fatal(err)
	}
	context := &campaign.HandlerRuntimeContext{SlotCount: 27, SpawnGroups: map[int]int{4: 1}, StoryViewport: true}
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "runtime_context", RuntimeContext: context},
		{Op: "spawn", Group: 4, Source: "0x22fde"},
		{Op: "act", Acting: resources[14], Source: "0x22ff2"},
		{Op: "act", Acting: resources[15], Source: "0x2303a"},
		{Op: "act", Acting: resources[16], Source: "0x2309b"},
	})
	g.st = st
	g.storyActors = nil
	g.storyBG = false
	g.beatAdvance()
	if !g.storyBG || len(st.Units) != 28 || st.Units[27].Portrait != 8 {
		t.Fatalf("post context/spawn storyBG=%v slots=%d slot27=%#v", g.storyBG, len(st.Units), st.Units[27])
	}
	g.tick(14)
	if got := st.Units[27]; got.X != 22 || got.Y != 6 {
		t.Fatalf("ACT14 slot27 = (%d,%d), want (22,6)", got.X, got.Y)
	}
	g.tick(3)
	for slot := 0; slot < 5; slot++ {
		if st.Units[slot].Dir != 0 {
			t.Fatalf("ACT15 party slot%d dir=%d, want 0", slot, st.Units[slot].Dir)
		}
	}
	g.tick(91)
	if got := st.Units[27]; got.X != 15 || got.Y != 0 || got.Dir != 2 {
		t.Fatalf("ACT16 slot27 = (%d,%d) dir%d, want (15,0) dir2", got.X, got.Y, got.Dir)
	}
}

func TestBeatScrollStepSlot2MatchesCh00ACT99Followup(t *testing.T) {
	// ch00 handler 0x32351 calls 0x13185(slot2) 15 times immediately after
	// direct ACT99 has moved Sol from Y42 to Y36.  Each original grid step has
	// seven redraw ticks, so the complete scroll is exactly 105 ticks.
	slot2 := 2
	g := newBeatTestGame(t, []campaign.Beat{{
		Op: "scroll_step", Slot: &slot2, Steps: 15, Frames: 105, Follow: true,
	}})
	g.m = &MapData{W: 20, H: 60, TileW: 24, TileH: 24, Cols: 8, Tiles: make([]int, 1200)}
	g.storyActors = make([]battle.Unit, 3)
	g.storyActors[2] = battle.Unit{Fig: 0, X: 8, Y: 36, OnField: true}
	g.camY = 34 * 24
	g.beatAdvance()
	if len(g.storyWalks) != 1 || g.followWalk {
		t.Fatalf("scroll_step should use its original safe-band follow rather than centering, walks=%d follow=%v", len(g.storyWalks), g.followWalk)
	}
	g.tick(104)
	if got := g.storyActors[2]; got.Y != 21 || got.Dir != 2 || got.OffY == 0 {
		t.Fatalf("after 104/105 ticks slot2 should still interpolate toward Y21 facing up: %+v", got)
	}
	g.tick(1)
	if got := g.storyActors[2]; got.X != 8 || got.Y != 21 || got.Dir != 2 || got.OffX != 0 || got.OffY != 0 {
		t.Fatalf("15-step scroll should finish slot2 at (8,21), pose2, without offset: %+v", got)
	}
	if g.camY != 20*24 {
		t.Fatalf("0x13185 safe-band camera=%v, want original cam row 20", g.camY)
	}
}

func TestBeatDirectACT100MovesSlot2DownTenCells(t *testing.T) {
	resources, err := campaign.LoadActingResourceSet(assetPath("assets/cutscenes/acting/map32.json"))
	if err != nil {
		t.Fatal(err)
	}
	frames := resources[100]
	if len(frames) != 1 || frames[0].Beats != 10 || frames[0].Special || len(frames[0].Units) != 1 {
		t.Fatalf("ACT100 must retain its direct decoded one-frame down×10 data: %#v", frames)
	}
	slot2 := 2
	if frames[0].Units[0].Slot == nil || *frames[0].Units[0].Slot != slot2 || frames[0].Units[0].Pose != 0 {
		t.Fatalf("ACT100 target must be original slot2 pose0: %#v", frames[0])
	}
	g := newBeatTestGame(t, []campaign.Beat{{Op: "act", Source: "0x323f5", Acting: frames}})
	g.storyActors = make([]battle.Unit, 3)
	g.storyActors[2] = battle.Unit{Fig: 0, X: 8, Y: 8, OnField: true}
	g.beatAdvance()
	g.tick(69)
	if got := g.storyActors[2]; got.Y != 17 || got.Dir != 0 {
		t.Fatalf("ACT100 before its 70th tick should have completed nine down cells: %+v", got)
	}
	g.tick(1)
	if got := g.storyActors[2]; got.X != 8 || got.Y != 18 || got.Dir != 0 || got.OffX != 0 || got.OffY != 0 {
		t.Fatalf("ACT100 direct frame should move slot2 Y8→18 in 70 ticks, pose0: %+v", got)
	}
}

func TestBeatFocusUnitWalksCursorAndScrollsAtOriginalSafeBand(t *testing.T) {
	slot2 := 2
	g := newBeatTestGame(t, []campaign.Beat{{Op: "focus_unit", Slot: &slot2}})
	// Original 0x12cea uses a 13×8 viewport. It walks X first, then Y, and only
	// scrolls the map origin after screen cursor X>10 / Y>5; it never centers.
	g.m = &MapData{W: 40, H: 30, TileW: 24, TileH: 24, Cols: 8, Tiles: make([]int, 1200)}
	g.storyActors = make([]battle.Unit, 3)
	g.storyActors[2] = battle.Unit{Fig: 0, X: 20, Y: 15, OnField: true}
	g.beatAdvance()
	if g.focusJob == nil || g.curX != 0 || g.curY != 0 {
		t.Fatalf("focus_unit must start a blocking grid walk, job=%#v cursor=(%d,%d)", g.focusJob, g.curX, g.curY)
	}
	g.tick(20)
	if g.curX != 20 || g.curY != 0 || g.camX != 216 || g.camY != 0 || g.focusJob == nil {
		t.Fatalf("after X phase cursor=(%d,%d) camera=(%v,%v) job=%#v, want (20,0)/(216,0)/active", g.curX, g.curY, g.camX, g.camY, g.focusJob)
	}
	g.tick(14)
	if g.curX != 20 || g.curY != 14 || g.camX != 216 || g.camY != 192 || g.focusJob == nil {
		t.Fatalf("before final Y step cursor=(%d,%d) camera=(%v,%v) job=%#v", g.curX, g.curY, g.camX, g.camY, g.focusJob)
	}
	g.tick(1)
	if g.curX != 20 || g.curY != 15 || g.camX != 216 || g.camY != 216 || g.focusJob != nil {
		t.Fatalf("focus_unit finish cursor=(%d,%d) camera=(%v,%v) job=%#v, want target/(216,216)/nil", g.curX, g.curY, g.camX, g.camY, g.focusJob)
	}
}

func TestBeatActingFailsClosedWhenRuntimeSlotWasNotMaterialized(t *testing.T) {
	slot8 := 8
	g := newBeatTestGame(t, []campaign.Beat{{Op: "act", Source: "0x32657", Acting: []campaign.ActingFrame{{
		Beats: 1, Special: true, Units: []campaign.ActingUnit{{Slot: &slot8, Pose: 2}},
	}}}})
	g.storyActors = make([]battle.Unit, 5) // map31 after groups 1+3+5
	g.beatAdvance()
	if g.loadErr == "" {
		t.Fatal("decoded act targeting an unmaterialized runtime slot must fail closed")
	}
}

func TestBeatFadeBothDirectionsCallThen(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "fade", Out: true, Frames: 4},
		{Op: "fade", Out: false, Frames: 4},
		{Op: "dialog", Line: 0, Count: 1},
	})
	g.beatAdvance()
	if g.fade == nil || !g.fade.out {
		t.Fatal("第一拍應是淡出中")
	}
	g.tick(4)
	if g.fade == nil || g.fade.out {
		t.Fatal("淡出走完應接淡入拍")
	}
	g.tick(4)
	if g.fade != nil {
		t.Fatal("淡入走完應清除 fade")
	}
	if len(g.dialog) != 1 {
		t.Fatalf("淡入完成應接 dialog 拍,得 dialog=%v", g.dialog)
	}
}

func TestBeatDelayCountsDownThenAdvances(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "delay", Frames: 3},
		{Op: "dialog", Line: 0, Count: 1},
	})
	g.beatAdvance()
	if g.beatDelay != 3 {
		t.Fatalf("delay 拍應設 beatDelay=3,得 %d", g.beatDelay)
	}
	for i := 0; i < 3; i++ { // 模擬 Update 裡 beatDelay 倒數(見 Update 內該區塊)
		g.beatDelay--
		if g.beatDelay == 0 {
			g.beatAdvance()
		}
	}
	if len(g.dialog) != 1 {
		t.Fatal("delay 倒數完應接下一拍")
	}
}

func TestBeatSequenceEndTriggersNodeTransition(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "bgm", Track: "FDMUS_999"}, // 非阻塞拍:應立即連呼下一拍
		{Op: "spawn", Group: 1},         // 非阻塞：啟用既有 roster group
		{Op: "join", CharID: 0},         // 非阻塞：寫入永久 party membership
	})
	g.beatAdvance() // 三個非阻塞拍應在同一次呼叫內全部跑完,直接進入收尾
	if g.fade == nil {
		t.Fatal("非阻塞拍序列跑完應觸發收尾淡出")
	}
	g.tick(storyFadeFrames) // 淡出走完 → camp.Advance("cs"→"end") + enterNode("end")
	if g.camp.Cur != "end" {
		t.Fatalf("應轉場到 end,得 %s", g.camp.Cur)
	}
}

func TestBeatJoinPersistsOnlyPlayerCharacterIDs(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "join", CharID: 0}, {Op: "join", CharID: 9},
		{Op: "join", CharID: 4}, {Op: "join", CharID: 9},
	})
	g.beatAdvance()
	if !g.partyMembers[9] || len(g.partyMembers) != 3 {
		t.Fatalf("join did not persist party membership: %#v", g.partyMembers)
	}
	if len(g.partyJoinOrder) != 3 || g.partyJoinOrder[0] != 0 || g.partyJoinOrder[1] != 9 || g.partyJoinOrder[2] != 4 {
		t.Fatalf("JOIN chronology lost or duplicated: %#v", g.partyJoinOrder)
	}

	bad := newBeatTestGame(t, []campaign.Beat{{Op: "join", CharID: 75}})
	bad.beatAdvance()
	if bad.loadErr == "" || len(bad.partyMembers) != 0 {
		t.Fatalf("scene portrait join must fail closed: err=%q party=%#v", bad.loadErr, bad.partyMembers)
	}
}

func TestBeatJoinRejectsDuplicateNativeIdentityRecords(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{{Op: "join", CharID: 12, Source: "0x23389"}})
	g.st = &battle.State{Units: []*battle.Unit{
		{NativeRecordByte8: 12, HasNativeRecordByte8: true},
		{NativeRecordByte8: 12, HasNativeRecordByte8: true},
	}}
	g.beatAdvance()
	if g.loadErr == "" || len(g.partyMembers) != 0 || len(g.partyRoster) != 0 {
		t.Fatalf(
			"duplicate native identity joined: err=%q members=%#v roster=%#v",
			g.loadErr, g.partyMembers, g.partyRoster,
		)
	}
}

func TestBeatSyncPartyPersistsProgressAndClearsBattleState(t *testing.T) {
	chapter := 1
	g := newBeatTestGame(t, []campaign.Beat{{Op: "sync_party"}, {Op: "set_chapter", Chapter: &chapter}})
	g.partyMembers = map[int]bool{0: true, 9: true}
	g.st = &battle.State{Units: []*battle.Unit{
		{Camp: battle.Own, Fig: 0, Name: "索爾", Lv: 3, HP: 7, MaxHP: 50, MP: 2, MaxMP: 12, AP: 18, Exp: 42, Acted: true, Poisoned: true, PoisonTurns: 3, Spells: []int{1, 2}, OnField: true},
		{Camp: battle.Own, Fig: 9, Name: "悠妮", Lv: 2, HP: 0, MaxHP: 31, MP: 0, MaxMP: 19, Paralyzed: true, ParalyzeTurns: 2, OnField: true},
		{Camp: battle.Enemy, Fig: 20, HP: 1, MaxHP: 1},
	}}
	g.beatAdvance()
	if g.handlerChapter != 1 || g.fade == nil {
		t.Fatalf("immediate post beats did not finish: chapter=%d fade=%#v err=%q", g.handlerChapter, g.fade, g.loadErr)
	}
	if len(g.partyRoster) != 2 {
		t.Fatalf("party roster = %#v, want two JOIN members", g.partyRoster)
	}
	sol := g.partyRoster[0]
	if sol.HP != 50 || sol.MP != 12 || sol.Lv != 3 || sol.Exp != 42 || sol.Acted || sol.Poisoned || len(sol.Spells) != 2 {
		t.Fatalf("survivor snapshot = %#v", sol)
	}
	yuni := g.partyRoster[9]
	if yuni.HP != 0 || yuni.MP != 19 || yuni.Paralyzed {
		t.Fatalf("defeated member did not retain zero HP/clear transient state: %#v", yuni)
	}

	fresh := &battle.State{Units: []*battle.Unit{{Camp: battle.Own, Fig: 0, X: 11, Y: 22, Group: 4, OnField: true, HP: 99, MaxHP: 99}}}
	g.applyPersistentParty(fresh)
	got := fresh.Units[0]
	if got.Lv != 3 || got.HP != 50 || got.MP != 12 || got.X != 11 || got.Y != 22 || got.Group != 4 || !got.OnField {
		t.Fatalf("persistent overlay lost progression or deployment: %#v", got)
	}
}

func TestCampaignPersistenceStubKeepsInventoryAfterBattleStateClears(t *testing.T) {
	c, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	n := c.Nodes["postbattle_ch04_persist"]
	if n == nil || n.Type != "cutscene" {
		t.Fatalf("missing chapter four persistence node: %#v", n)
	}
	chapter := 4
	g := newBeatTestGame(t, []campaign.Beat{{Op: "sync_party"}, {Op: "set_chapter", Chapter: &chapter}})
	g.partyMembers = map[int]bool{0: true}
	units := make([]*battle.Unit, 50)
	for i := range units {
		units[i] = &battle.Unit{Camp: battle.Enemy, Fig: i, OnField: false}
	}
	units[0] = &battle.Unit{
		Camp: battle.Own, Fig: 0, Name: "索爾", Lv: 8,
		HP: 13, MaxHP: 55, MP: 4, MaxMP: 12, OnField: true,
		Inventory: []int{0xc0, 0xd2},
	}
	g.st = &battle.State{Units: units}
	g.beatAdvance()
	if g.loadErr != "" || g.handlerChapter != 4 {
		t.Fatalf("persistence beats failed: err=%q chapter=%d", g.loadErr, g.handlerChapter)
	}
	if got := g.partyRoster[0]; got.Lv != 8 || got.HP != 55 || !reflect.DeepEqual(got.Inventory, []int{0xc0, 0xd2}) {
		t.Fatalf("postbattle snapshot lost progression/inventory: %#v", got)
	}
	g.tick(storyFadeFrames)
	if g.st != nil {
		t.Fatal("intermission boundary did not clear completed battle state")
	}
	if got := g.partyRoster[0].Inventory; !reflect.DeepEqual(got, []int{0xc0, 0xd2}) {
		t.Fatalf("clearing battle state also cleared persistent inventory: %#v", got)
	}
}

func TestScenarioJoinPersistsRecruitedAllyThroughPostBattleSync(t *testing.T) {
	joinTable, err := campaign.LoadNativeJoinConstructorTable(assetPath("assets/data/native_join_constructor.json"))
	if err != nil {
		t.Fatal(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(assetPath("assets/data/native_item_effect_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	hanoRuntime, err := joinTable.MaterializePersistentUnit(1, battle.Unit{
		Camp: battle.Ally, Fig: 1, Name: "哈諾", OnField: true,
	}, itemRows)
	if err != nil {
		t.Fatal(err)
	}
	hanoRuntime.Camp = battle.Ally
	g := &Game{
		partyMembers:             map[int]bool{0: true},
		partyJoinOrder:           []int{0},
		nativeJoinConstructor:    joinTable,
		hasNativeJoinConstructor: true,
		sc:                       &battle.Scenario{},
		st: &battle.State{Units: []*battle.Unit{
			{Camp: battle.Own, Fig: 0, HP: 10, MaxHP: 10, OnField: true},
			&hanoRuntime,
		}},
	}
	// Model the scenario-owned JOIN effect separately from the spawned unit's
	// current camp; permanent membership, not camp colour, controls persistence.
	g.sc.Events = []battle.Event{{Trigger: "on_turn_end", Do: []battle.Action{{Type: "join_party", CharID: 1}}}}
	g.sc.Fire(g.st, "on_turn_end", "")
	g.applyScenarioPartyJoins()
	if !g.partyMembers[1] || len(g.partyJoinOrder) != 2 || g.partyJoinOrder[1] != 1 {
		t.Fatalf("scenario JOIN did not update campaign membership/order: %#v %#v", g.partyMembers, g.partyJoinOrder)
	}
	if err := g.syncPartyFromBattle(); err != nil {
		t.Fatal(err)
	}
	if hano, ok := g.partyRoster[1]; !ok || hano.HP != 56 || hano.NativeRecordWord42 != 56 || !hano.HasNativeRecordWord42 || hano.Lv != 3 {
		t.Fatalf("recruited allied Hano was not persisted: %#v", g.partyRoster)
	}
}

func TestSyncPartyUsesNativeIdentityWhenFigDiffers(t *testing.T) {
	g := &Game{
		partyMembers: map[int]bool{4: true},
		partyRoster:  map[int]battle.Unit{4: {Fig: 4, NativeIdentity: 0x2a, HasNativeIdentity: true, MaxHP: 30, MaxMP: 7}},
		st:           &battle.State{Units: []*battle.Unit{{Camp: battle.Own, Fig: 99, NativeIdentity: 0x2a, HasNativeIdentity: true, HP: 10, MaxHP: 30, MP: 2, MaxMP: 7, OnField: true}}},
	}
	if err := g.syncPartyFromBattle(); err != nil {
		t.Fatal(err)
	}
	if got := g.partyRoster[4]; got.HP != 30 || got.MP != 7 {
		t.Fatalf("raw identity did not update persistent member: %#v", got)
	}
	if _, ok := g.partyRoster[99]; ok {
		t.Fatal("Fig selector was incorrectly used as persistent key")
	}
}

func TestSyncPartySkipsUnknownNativeIdentity(t *testing.T) {
	g := &Game{
		partyMembers: map[int]bool{4: true},
		partyRoster:  map[int]battle.Unit{4: {Fig: 4, NativeIdentity: 0x2a, HasNativeIdentity: true, HP: 21, MaxHP: 30}},
		st:           &battle.State{Units: []*battle.Unit{{Camp: battle.Own, Fig: 4, NativeIdentity: 0x2b, HasNativeIdentity: true, HP: 1, MaxHP: 30, OnField: true}}},
	}
	if err := g.syncPartyFromBattle(); err != nil {
		t.Fatal(err)
	}
	if got := g.partyRoster[4].HP; got != 21 {
		t.Fatalf("unknown raw identity should fail closed, HP=%d", got)
	}
}

func TestBeatGrantItemUsesFirstPlayerInventoryWithRoom(t *testing.T) {
	itemID := 0xc6
	g := newBeatTestGame(t, []campaign.Beat{{Op: "grant_item", ItemID: &itemID}, {Op: "sync_party"}})
	g.partyMembers = map[int]bool{9: true}
	g.st = &battle.State{Units: []*battle.Unit{
		{Camp: battle.Own, Fig: 0, Inventory: []int{1, 2, 3, 4, 5, 6, 7, 8}},
		{Camp: battle.Enemy, Fig: 99},
		{Camp: battle.Own, Fig: 9, Inventory: []int{4}},
	}}
	g.beatAdvance()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if got := g.st.Units[0].Inventory; len(got) != 8 {
		t.Fatalf("full first player inventory changed: %#v", got)
	}
	if got := g.st.Units[1].Inventory; len(got) != 0 {
		t.Fatalf("enemy received reward: %#v", got)
	}
	if got := g.st.Units[2].Inventory; len(got) != 2 || got[1] != 0xc6 {
		t.Fatalf("second player reward inventory = %#v", got)
	}
	if got := g.partyRoster[9].Inventory; len(got) != 2 || got[1] != 0xc6 {
		t.Fatalf("reward did not persist through sync_party: %#v", got)
	}

	full := &Game{st: &battle.State{Units: []*battle.Unit{{Camp: battle.Own, Inventory: make([]int, 8)}}}}
	if full.grantItemToParty(0x64) || len(full.st.Units[0].Inventory) != 8 {
		t.Fatalf("all-full inventory should silently reject reward: %#v", full.st.Units[0].Inventory)
	}
}

func TestBeatAnyUnitInactiveChoosesOneArmAndKeepsCommonTail(t *testing.T) {
	itemID := 0xc6
	condition := &campaign.BeatCondition{Op: "any_unit_inactive", UnitSlots: []int{5, 6, 7, 8, 9, 10}}
	branch := campaign.Beat{
		Op: "if", Condition: condition,
		Then: []campaign.Beat{{Op: "join", CharID: 4}},
		Else: []campaign.Beat{{Op: "grant_item", ItemID: &itemID}},
	}
	common := campaign.Beat{Op: "join", CharID: 9}

	inactive := newBeatTestGame(t, []campaign.Beat{branch, common})
	inactive.st = &battle.State{Units: make([]*battle.Unit, 12)}
	for i := range inactive.st.Units {
		inactive.st.Units[i] = &battle.Unit{HP: 1, OnField: true}
	}
	inactive.st.Units[0] = &battle.Unit{Camp: battle.Own, Inventory: []int{1}, HP: 1, OnField: true}
	inactive.st.Units[10].HP = 0
	inactive.beatAdvance()
	if inactive.loadErr != "" || !inactive.partyMembers[4] || !inactive.partyMembers[9] {
		t.Fatalf("inactive arm did not run: err=%q party=%#v", inactive.loadErr, inactive.partyMembers)
	}
	if got := inactive.st.Units[0].Inventory; len(got) != 1 {
		t.Fatalf("inactive arm incorrectly granted item: %#v", got)
	}
	if len(inactive.beats) != 3 || inactive.beats[2].Op != "join" {
		t.Fatalf("selected arm/common tail splice = %#v", inactive.beats)
	}

	active := newBeatTestGame(t, []campaign.Beat{branch, common})
	active.st = &battle.State{Units: make([]*battle.Unit, 12)}
	for i := range active.st.Units {
		active.st.Units[i] = &battle.Unit{HP: 1, OnField: true}
	}
	active.st.Units[0] = &battle.Unit{Camp: battle.Own, HP: 1, OnField: true}
	active.beatAdvance()
	if active.loadErr != "" || active.partyMembers[4] || !active.partyMembers[9] {
		t.Fatalf("all-active arm selection = err=%q party=%#v", active.loadErr, active.partyMembers)
	}
	if got := active.st.Units[0].Inventory; len(got) != 1 || got[0] != 0xc6 {
		t.Fatalf("slots outside 5..10 affected condition; reward=%#v", got)
	}
}

func TestBeatAnyUnitInactiveFailsClosedWithoutCompleteRoster(t *testing.T) {
	condition := &campaign.BeatCondition{Op: "any_unit_inactive", UnitSlots: []int{5, 6, 7, 8, 9, 10}}
	beats := []campaign.Beat{{
		Op: "if", Condition: condition,
		Else: []campaign.Beat{{Op: "join", CharID: 9}},
	}}
	short := &battle.State{Units: make([]*battle.Unit, 10)}
	short.Units[5] = &battle.Unit{HP: 1, OnField: true} // reject before selecting either arm.
	for _, st := range []*battle.State{nil, short} {
		g := newBeatTestGame(t, beats)
		g.st = st
		g.beatAdvance()
		if g.loadErr == "" || g.partyMembers[9] {
			t.Fatalf("incomplete runtime state did not fail closed: state=%#v err=%q", st, g.loadErr)
		}
	}
}

func TestBeatAnyUnitInactivePrefersNativeByte5Predicate(t *testing.T) {
	itemID := 0xc6
	condition := &campaign.BeatCondition{Op: "any_unit_inactive", UnitSlots: []int{5}}
	branch := campaign.Beat{Op: "if", Condition: condition,
		Then: []campaign.Beat{{Op: "join", CharID: 4}},
		Else: []campaign.Beat{{Op: "grant_item", ItemID: &itemID}}}

	rawInactive := newBeatTestGame(t, []campaign.Beat{branch})
	rawInactive.st = &battle.State{Units: make([]*battle.Unit, 6)}
	for i := range rawInactive.st.Units {
		rawInactive.st.Units[i] = &battle.Unit{HP: 9, OnField: true, HasNativeRecordByte5: true}
	}
	rawInactive.st.Units[5].NativeRecordByte5 = 1
	rawInactive.beatAdvance()
	if rawInactive.loadErr != "" || !rawInactive.partyMembers[4] {
		t.Fatalf("raw bit0=1 did not select native branch: err=%q party=%#v", rawInactive.loadErr, rawInactive.partyMembers)
	}

	rawActive := newBeatTestGame(t, []campaign.Beat{branch})
	rawActive.st = &battle.State{Units: make([]*battle.Unit, 6)}
	for i := range rawActive.st.Units {
		rawActive.st.Units[i] = &battle.Unit{HP: 0, OnField: false, HasNativeRecordByte5: true, NativeRecordByte5: 0}
	}
	rawActive.beatAdvance()
	if rawActive.loadErr != "" || rawActive.partyMembers[4] || len(rawActive.st.Units[0].Inventory) != 1 || rawActive.st.Units[0].Inventory[0] != itemID {
		t.Fatalf("raw bit0=0 incorrectly used HP/OnField fallback: err=%q party=%#v inventory=%#v", rawActive.loadErr, rawActive.partyMembers, rawActive.st.Units[0].Inventory)
	}
}

func TestBeatRosterHasUsesPersistentPartyAndFailsClosedWithoutIt(t *testing.T) {
	charID := 12
	branch := campaign.Beat{
		Op: "if", Condition: &campaign.BeatCondition{Op: "roster_has", CharID: &charID},
		Then: []campaign.Beat{{Op: "join", CharID: 4}},
		Else: []campaign.Beat{{Op: "join", CharID: 9}},
	}
	for _, tc := range []struct {
		name    string
		members map[int]bool
		want    int
		fail    bool
	}{
		{name: "present", members: map[int]bool{12: true}, want: 4},
		{name: "absent", members: map[int]bool{0: true}, want: 9},
		{name: "missing roster", members: nil, fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newBeatTestGame(t, []campaign.Beat{branch})
			g.partyMembers = tc.members
			g.beatAdvance()
			if tc.fail {
				if g.loadErr == "" || len(g.partyMembers) != 0 {
					t.Fatalf("missing permanent roster did not fail closed: err=%q party=%#v", g.loadErr, g.partyMembers)
				}
				return
			}
			if g.loadErr != "" || !g.partyMembers[tc.want] {
				t.Fatalf("roster variant=%d err=%q party=%#v", tc.want, g.loadErr, g.partyMembers)
			}
		})
	}
}

func TestBeatNativeEventStateConditionSelectsOnlyProvenIndex(t *testing.T) {
	index := 12
	branch := campaign.Beat{Op: "if", Condition: &campaign.BeatCondition{Op: "native_event_state_nonzero", EventStateIndex: &index}, Then: []campaign.Beat{{Op: "join", CharID: 4}}, Else: []campaign.Beat{{Op: "join", CharID: 9}}}
	for _, tc := range []struct {
		value byte
		want  int
	}{{0, 9}, {1, 4}} {
		g := newBeatTestGame(t, []campaign.Beat{branch})
		g.st = &battle.State{}
		g.st.NativeEventState[12] = tc.value
		g.beatAdvance()
		if g.loadErr != "" || !g.partyMembers[tc.want] {
			t.Fatalf("event state %d chose party=%#v err=%q", tc.value, g.partyMembers, g.loadErr)
		}
	}
	bad := 32
	g := newBeatTestGame(t, []campaign.Beat{{Op: "if", Condition: &campaign.BeatCondition{Op: "native_event_state_nonzero", EventStateIndex: &bad}}})
	g.st = &battle.State{}
	g.beatAdvance()
	if g.loadErr == "" {
		t.Fatal("out-of-range native event state index did not fail closed")
	}
}

func TestBeatNativeEventStateEqualsRefinesExactRuntimeFrontier(t *testing.T) {
	index, value, slots := 17, 1, 44
	condition := &campaign.BeatCondition{
		Op: "native_event_state_eq", EventStateIndex: &index,
		EventStateValue: &value, RequiredSlotCount: &slots,
	}
	g := &Game{st: &battle.State{Units: make([]*battle.Unit, 34)}}
	matched, err := g.evalBeatCondition(condition)
	if err != nil || matched {
		t.Fatalf("clear state matched=%v err=%v", matched, err)
	}
	g.st.NativeEventState[17] = 1
	if _, err := g.evalBeatCondition(condition); err == nil {
		t.Fatal("matched event state accepted a 34-slot runtime")
	}
	g.st.Units = make([]*battle.Unit, 44)
	matched, err = g.evalBeatCondition(condition)
	if err != nil || !matched {
		t.Fatalf("exact event25 frontier matched=%v err=%v", matched, err)
	}
}

func TestBeatNativeInactiveCountGreaterThanUsesRawByteOnly(t *testing.T) {
	threshold := 4
	branch := campaign.Beat{Op: "if", Condition: &campaign.BeatCondition{Op: "native_inactive_count_gt", UnitSlots: []int{66, 67, 68, 69, 70, 71, 72, 73}, Threshold: &threshold}, Then: []campaign.Beat{{Op: "join", CharID: 18}}, Else: []campaign.Beat{{Op: "join", CharID: 19}}}
	for _, tc := range []struct {
		inactive int
		want     int
	}{
		{inactive: 5, want: 18},
		{inactive: 4, want: 19},
	} {
		g := newBeatTestGame(t, []campaign.Beat{branch})
		g.st = &battle.State{Units: make([]*battle.Unit, 74)}
		for i := 66; i <= 73; i++ {
			g.st.Units[i] = &battle.Unit{HasNativeRecordByte5: true}
		}
		for i := 66; i < 66+tc.inactive; i++ {
			g.st.Units[i].NativeRecordByte5 = 1
		}
		g.beatAdvance()
		if g.loadErr != "" || !g.partyMembers[tc.want] {
			t.Fatalf("inactive=%d party=%#v err=%q", tc.inactive, g.partyMembers, g.loadErr)
		}
	}
	g := newBeatTestGame(t, []campaign.Beat{branch})
	g.st = &battle.State{Units: make([]*battle.Unit, 74)}
	for i := 66; i <= 73; i++ {
		g.st.Units[i] = &battle.Unit{}
	}
	g.beatAdvance()
	if g.loadErr == "" {
		t.Fatal("missing raw byte5 did not fail closed")
	}
}

func TestBeatNativeCh15RawComparisonsFailClosedWithoutProvenance(t *testing.T) {
	round := 18
	wordOffset, wordValue, slot := 0x42, 0x140, 0
	for _, tc := range []struct {
		name      string
		condition *campaign.BeatCondition
		state     *battle.State
		want      bool
	}{
		{"round-hit", &campaign.BeatCondition{Op: "native_round_gt", NativeRound: &round}, &battle.State{NativeRoundCounter: 19}, true},
		{"round-miss", &campaign.BeatCondition{Op: "native_round_gt", NativeRound: &round}, &battle.State{NativeRoundCounter: 18}, false},
		{"word-hit", &campaign.BeatCondition{Op: "native_record_word_gte", UnitSlot: &slot, NativeRecordWordOffset: &wordOffset, NativeRecordWordValue: &wordValue}, &battle.State{Units: []*battle.Unit{{HasNativeRecordWord42: true, NativeRecordWord42: 0x140}}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newBeatTestGame(t, []campaign.Beat{{Op: "if", Condition: tc.condition, Then: []campaign.Beat{{Op: "join", CharID: 18}}, Else: []campaign.Beat{{Op: "join", CharID: 19}}}})
			g.st = tc.state
			g.beatAdvance()
			if g.loadErr != "" || !g.partyMembers[map[bool]int{true: 18, false: 19}[tc.want]] {
				t.Fatalf("want=%v party=%#v err=%q", tc.want, g.partyMembers, g.loadErr)
			}
		})
	}
	g := newBeatTestGame(t, []campaign.Beat{{Op: "if", Condition: &campaign.BeatCondition{Op: "native_record_word_gte", UnitSlot: &slot, NativeRecordWordOffset: &wordOffset, NativeRecordWordValue: &wordValue}}})
	g.st = &battle.State{Units: []*battle.Unit{{MaxHP: 999}}}
	g.beatAdvance()
	if g.loadErr == "" {
		t.Fatal("normalized MaxHP did not fail closed as raw +0x42")
	}
}

func TestBeatChapter22RawBranchConditionsUseOnlyProvenance(t *testing.T) {
	item, identity, round := 100, 18, 15
	branch := campaign.Beat{Op: "if", Condition: &campaign.BeatCondition{Op: "native_inventory_item_present", NativeInventoryItemID: &item}, Then: []campaign.Beat{{Op: "join", CharID: 22}}, Else: []campaign.Beat{{Op: "join", CharID: 19}}}
	g := newBeatTestGame(t, []campaign.Beat{branch})
	g.st = &battle.State{Units: make([]*battle.Unit, 16)}
	for i := range g.st.Units {
		g.st.Units[i] = &battle.Unit{InventorySlots: make([]int, 8), NativeInventoryFlags: make([]int, 8)}
	}
	g.st.Units[3].InventorySlots[2] = 100
	g.beatAdvance()
	if g.loadErr != "" || !g.partyMembers[22] {
		t.Fatalf("raw inventory item branch party=%#v err=%q", g.partyMembers, g.loadErr)
	}

	branch = campaign.Beat{Op: "if", Condition: &campaign.BeatCondition{Op: "native_persistent_identity_present", NativePersistentIdentity: &identity}, Then: []campaign.Beat{{Op: "join", CharID: 22}}, Else: []campaign.Beat{{Op: "join", CharID: 19}}}
	g = newBeatTestGame(t, []campaign.Beat{branch})
	g.partyRoster = map[int]battle.Unit{1: {HasNativeIdentity: true, NativeIdentity: 18}}
	g.beatAdvance()
	if g.loadErr != "" || !g.partyMembers[22] {
		t.Fatalf("raw persistent identity branch party=%#v err=%q", g.partyMembers, g.loadErr)
	}

	branch = campaign.Beat{Op: "if", Condition: &campaign.BeatCondition{Op: "native_round_lt", NativeRound: &round}, Then: []campaign.Beat{{Op: "join", CharID: 22}}, Else: []campaign.Beat{{Op: "join", CharID: 19}}}
	for _, tc := range []struct {
		counter int
		want    int
	}{{counter: 14, want: 22}, {counter: 15, want: 19}} {
		g = newBeatTestGame(t, []campaign.Beat{branch})
		g.st = &battle.State{NativeRoundCounter: tc.counter}
		g.beatAdvance()
		if g.loadErr != "" || !g.partyMembers[tc.want] {
			t.Fatalf("raw round=%d party=%#v err=%q", tc.counter, g.partyMembers, g.loadErr)
		}
	}

	g = newBeatTestGame(t, []campaign.Beat{{Op: "if", Condition: &campaign.BeatCondition{Op: "native_inventory_item_present", NativeInventoryItemID: &item}}})
	g.st = &battle.State{Units: make([]*battle.Unit, 16)}
	for i := range g.st.Units {
		g.st.Units[i] = &battle.Unit{}
	}
	g.beatAdvance()
	if g.loadErr == "" {
		t.Fatal("missing raw inventory slots did not fail closed")
	}
}

func TestBeatNativeAnyOfCombinesOnlyRawPredicates(t *testing.T) {
	round, threshold := 18, 4
	condition := &campaign.BeatCondition{Op: "native_any_of", Any: []campaign.BeatCondition{
		{Op: "native_round_gt", NativeRound: &round},
		{Op: "native_inactive_count_gt", UnitSlots: []int{66, 67, 68, 69, 70}, Threshold: &threshold},
	}}
	g := newBeatTestGame(t, []campaign.Beat{{Op: "if", Condition: condition, Then: []campaign.Beat{{Op: "join", CharID: 18}}, Else: []campaign.Beat{{Op: "join", CharID: 19}}}})
	g.st = &battle.State{NativeRoundCounter: 19, Units: make([]*battle.Unit, 68)}
	g.beatAdvance()
	if g.loadErr != "" || !g.partyMembers[18] {
		t.Fatalf("native_any_of round branch failed: party=%#v err=%q", g.partyMembers, g.loadErr)
	}
	g = newBeatTestGame(t, []campaign.Beat{{Op: "if", Condition: condition, Then: []campaign.Beat{{Op: "join", CharID: 18}}, Else: []campaign.Beat{{Op: "join", CharID: 19}}}})
	g.st = &battle.State{Units: make([]*battle.Unit, 71)}
	for _, slot := range []int{66, 67, 68, 69, 70} {
		g.st.Units[slot] = &battle.Unit{HasNativeRecordByte5: true, NativeRecordByte5: 1}
	}
	g.beatAdvance()
	if g.loadErr != "" || !g.partyMembers[18] {
		t.Fatalf("native_any_of count branch failed: party=%#v err=%q", g.partyMembers, g.loadErr)
	}
	g = newBeatTestGame(t, []campaign.Beat{{Op: "if", Condition: condition}})
	g.st = &battle.State{Units: make([]*battle.Unit, 71)}
	for _, slot := range []int{66, 67, 68, 69, 70} {
		g.st.Units[slot] = &battle.Unit{}
	}
	g.beatAdvance()
	if g.loadErr == "" {
		t.Fatal("native_any_of missing raw child provenance did not fail closed")
	}
}

func TestReorderScenarioPartyUsesOriginalJoinSlots(t *testing.T) {
	sc := &battle.Scenario{
		Party:       []battle.PartyMember{{Fig: 0}, {Fig: 4}, {Fig: 9}, {Fig: 30}},
		DeployCells: [][2]int{{7, 20}, {8, 22}, {10, 21}, {11, 23}},
	}
	if err := reorderScenarioParty(sc, []int{0, 9, 4, 30}); err != nil {
		t.Fatal(err)
	}
	for slot, want := range []struct{ id, x, y int }{
		{0, 7, 20}, {9, 10, 21}, {4, 8, 22}, {30, 11, 23},
	} {
		if sc.Party[slot].Fig != want.id || sc.DeployCells[slot] != [2]int{want.x, want.y} {
			t.Fatalf("party runtime slot %d = fig%d at %v, want fig%d at (%d,%d)", slot, sc.Party[slot].Fig, sc.DeployCells[slot], want.id, want.x, want.y)
		}
	}
}

func TestApplyLoadCHDirectReplayUsesBindingPartyOrder(t *testing.T) {
	g := &Game{}
	err := g.applyLoadCH(&campaign.LoadCHState{
		Chapter:       0,
		Map:           "assets/maps/map0",
		Roster:        "assets/maps/map0/map0_units.json",
		SlotCount:     30,
		Script:        "assets/story/ch01.json",
		PartyScenario: "assets/scenarios/ch01.json",
		PartyOrder:    []int{0, 9, 4, 30},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(g.storyActors) < 4 {
		t.Fatalf("LOADCH materialized only %d actors", len(g.storyActors))
	}
	for slot, want := range []struct{ id, x, y int }{
		{0, 7, 20}, {9, 10, 21}, {4, 8, 22}, {30, 11, 23},
	} {
		u := g.storyActors[slot]
		if u.Fig != want.id || u.X != want.x || u.Y != want.y {
			t.Fatalf("direct LOADCH slot %d = fig%d at (%d,%d), want fig%d at (%d,%d)", slot, u.Fig, u.X, u.Y, want.id, want.x, want.y)
		}
	}
	if len(g.partyRoster) != 0 {
		t.Fatalf("direct LOADCH replay invented persistent party: %#v", g.partyRoster)
	}
}

func TestApplyLoadCHSeedsJoinedPersistentPartyBeforeFirstBattleSync(t *testing.T) {
	stats, err := campaign.LoadItemStats(assetPath("assets/data/item.json"))
	if err != nil {
		t.Fatal(err)
	}
	joinTable, err := campaign.LoadNativeJoinConstructorTable(assetPath("assets/data/native_join_constructor.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{
		partyMembers: map[int]bool{
			0: true, 9: true, 4: true, 30: true,
		},
		partyJoinOrder:           []int{0, 9, 4, 30},
		shopItemStats:            stats,
		nativeJoinConstructor:    joinTable,
		hasNativeJoinConstructor: true,
	}
	err = g.applyLoadCH(&campaign.LoadCHState{
		Chapter:       0,
		Map:           "assets/maps/map0",
		Roster:        "assets/maps/map0/map0_units.json",
		SlotCount:     30,
		Script:        "assets/story/ch01.json",
		PartyScenario: "assets/scenarios/ch01.json",
		PartyOrder:    []int{0, 9, 4, 30},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(g.partyRoster) != 4 {
		t.Fatalf("joined LOADCH roster=%#v, want four persistent records", g.partyRoster)
	}
	for _, id := range g.partyJoinOrder {
		unit, ok := g.partyRoster[id]
		if !ok || unit.Fig != id || !unit.HasNativeIdentity ||
			!unit.HasNativeRecordWord42 || !unit.HasNativeRecordWord46 ||
			len(unit.InventorySlots) != 8 ||
			len(unit.NativeInventoryFlags) != 8 ||
			!unit.EquipmentBaseSet {
			t.Fatalf("joined LOADCH record %d lacks typed provenance: %#v", id, unit)
		}
	}
	full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	g.camp = campaign.NewRunner(full)
	g.camp.Cur = "shop_ch02_weapon"
	g.shopItemTypes, g.shopEquipTypes, err = campaign.LoadShopEligibility(
		assetPath("assets/data/item.json"),
		assetPath("assets/data/class_equip_types.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !g.setupNativeShopRecipients() ||
		g.nativeShopMode != "recipient_equipment" ||
		!reflect.DeepEqual(g.shopRecipients, []int{0, 9, 4}) {
		t.Fatalf(
			"normal campaign LOADCH roster did not reach equipment recipients: mode=%q ids=%v",
			g.nativeShopMode, g.shopRecipients,
		)
	}

	// The first post-battle sync must now find the persistent native identity
	// instead of skipping every freshly joined player record.
	sol := g.partyRoster[0]
	current := cloneNativeShopUnit(sol)
	current.HP, current.MaxHP, current.MP, current.MaxMP = 3, 40, 1, 8
	current.OnField = true
	g.st = &battle.State{Units: []*battle.Unit{&current}}
	if err := g.syncPartyFromBattle(); err != nil {
		t.Fatal(err)
	}
	if got := g.partyRoster[0]; got.HP != 40 || got.MP != 8 {
		t.Fatalf("first native-identity sync did not update seeded record: %#v", got)
	}
}

func TestCh00CompiledHandlerCarriesItsExactRuntimeRosterIntoChapterOne(t *testing.T) {
	const originalBase = "../../../org_game/炎龍騎士團/FLAME2"
	fdotherPath := filepath.Join(originalBase, "FDOTHER.DAT")
	if _, err := os.Stat(fdotherPath); err != nil {
		t.Skip("player-provided FDOTHER.DAT is absent")
	}
	t.Setenv("FD2_ORIGINAL_FDOTHER", fdotherPath)

	c, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{camp: campaign.NewRunner(c), sfxSpawnIntro: []byte{1}}
	g.enterNode()
	if g.loadErr != "" {
		t.Fatalf("enter ch00 handler: %s", g.loadErr)
	}

	// Drive the same blocking jobs that Update owns and dismiss each compiled
	// dialog beat as campInput would.  The bound keeps this a regression test:
	// an unresolved native op or a stalled handler must fail instead of being
	// silently skipped.
	spawnIntroFrames := 0
	for frame := 0; frame < 100000 && g.camp.NodeID() != "battle_ch01"; frame++ {
		if len(g.dialog) > 0 {
			g.dialog = nil
			g.beatAdvance()
		}
		g.tick(1)
		if g.loadErr != "" {
			t.Fatalf("compiled ch00 handler stopped at beat %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
		}
		if g.spawnIntroTransition != nil {
			spawnIntroFrames++
			g.spawnIntroTransition.drawn = true
		}
	}
	if g.camp.NodeID() != "battle_ch01" {
		t.Fatalf("compiled ch00 handler did not reach battle_ch01: node=%q beat=%d/%d", g.camp.NodeID(), g.beatIdx, len(g.beats))
	}
	if g.st == nil || g.sc == nil {
		t.Fatalf("battle handoff did not materialize state/scenario: st=%v sc=%v", g.st != nil, g.sc != nil)
	}
	if spawnIntroFrames != 24 {
		t.Fatalf("two native spawn intros presented %d frames, want 2*12", spawnIntroFrames)
	}
	if len(g.st.Units) != 12 {
		t.Fatalf("handler runtime frontier=%d, want 4 party + two four-record groups = 12", len(g.st.Units))
	}
	for slot, want := range []struct {
		fig, group, x, y int
		onField          bool
	}{
		{0, 0, 7, 14, true},
		{9, 0, 10, 15, true},
		{4, 0, 8, 16, true},
		{30, 0, 11, 17, true},
		{96, 1, 1, 4, true},
		{96, 1, 2, 2, true},
		{96, 1, 4, 2, true},
		{96, 1, 6, 1, true},
		{96, 2, 3, 18, true},
		{96, 2, 4, 23, false},
		{96, 2, 2, 18, true},
		{96, 2, 5, 19, true},
	} {
		u := g.st.Units[slot]
		if u == nil || u.Fig != want.fig || u.Group != want.group || u.X != want.x || u.Y != want.y || u.OnField != want.onField {
			t.Fatalf("battle slot%d = %#v, want fig%d group%d at (%d,%d) onField=%v", slot, u, want.fig, want.group, want.x, want.y, want.onField)
		}
	}
	if g.st.Units[9].NativeRecordByte5 != 1 {
		t.Fatalf("ch00 deactivate(slot9) was not carried across battle handoff: %#v", g.st.Units[9])
	}
	for _, group := range []int{3, 4, 5, 6, 7} {
		if !g.st.PendingGroups[group] {
			t.Fatalf("adopted battle lost pending spawn group %d: %#v", group, g.st.PendingGroups)
		}
	}

	// Continue through the real chapter-one join event, the compiled postbattle
	// sync, town, and its editable exit. This is the normal campaign path that
	// a direct FD2_CAMP_NODE jump deliberately does not reproduce.
	g.st.Turn = 3
	g.sc.Fire(g.st, "on_turn_end", "")
	g.applyScenarioPartyJoins()
	if !g.partyMembers[1] ||
		!reflect.DeepEqual(g.partyJoinOrder, []int{0, 9, 4, 30, 1}) {
		t.Fatalf("chapter-one join chronology=%v members=%#v", g.partyJoinOrder, g.partyMembers)
	}
	if got := g.camp.Advance("win"); got != "story_ch02" {
		t.Fatalf("battle_ch01 win=%q, want story_ch02", got)
	}
	g.enterNode()
	for frame := 0; frame < 10000 && g.camp.NodeID() != "town_ch02"; frame++ {
		if len(g.dialog) > 0 {
			g.dialog = nil
			g.beatAdvance()
		}
		g.tick(1)
		if g.loadErr != "" {
			t.Fatalf("compiled ch00 post stopped at beat %d/%d: %s", g.beatIdx, len(g.beats), g.loadErr)
		}
	}
	if g.camp.NodeID() != "town_ch02" {
		t.Fatalf("postbattle did not reach town_ch02: node=%q beat=%d/%d", g.camp.NodeID(), g.beatIdx, len(g.beats))
	}
	if len(g.partyRoster) != 5 {
		t.Fatalf("postbattle persistent roster=%#v, want five joined records", g.partyRoster)
	}
	if got := g.camp.Advance("opt2"); got != "preparation_ch02" {
		t.Fatalf("town exit=%q, want preparation_ch02", got)
	}
	g.enterNode()
	if !reflect.DeepEqual(g.prepIDs, []int{9, 4, 30, 1}) ||
		g.preparationSelected() != 0 || g.prepSelecting || g.prepConfirm {
		t.Fatalf(
			"normal preparation entry ids=%v selected=%d selecting=%v confirm=%v",
			g.prepIDs, g.preparationSelected(), g.prepSelecting, g.prepConfirm,
		)
	}
	if !g.acceptTownDeparturePrompt() {
		t.Fatal("fixed record0 plus four selectable records should skip 0x318ad after record confirmation")
	}
}

func TestChapter1PreLoadCHUsesFiveMemberJoinOrderAndSpawnFrontiers(t *testing.T) {
	beats, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch01_pre.json"))
	if err != nil || len(issues) != 0 || len(beats) == 0 || beats[0].LoadCH == nil {
		t.Fatalf("compile ch01_pre err=%v issues=%#v first=%#v", err, issues, beats)
	}
	g := &Game{
		partyMembers:   map[int]bool{0: true, 9: true, 4: true, 30: true, 1: true},
		partyJoinOrder: []int{0, 9, 4, 30, 1},
		partyRoster: map[int]battle.Unit{
			1: {Fig: 1, Name: "哈諾", Lv: 3, HP: 17, MaxHP: 36, MP: 0, MaxMP: 0},
		},
	}
	if err := g.applyLoadCH(beats[0].LoadCH); err != nil {
		t.Fatal(err)
	}
	if len(g.storyActors) != 5 {
		t.Fatalf("ch01_pre initial runtime slots=%d, want five party members", len(g.storyActors))
	}
	for slot, id := range []int{0, 9, 4, 30, 1} {
		if g.storyActors[slot].Fig != id {
			t.Fatalf("ch01_pre party slot%d fig=%d, want %d", slot, g.storyActors[slot].Fig, id)
		}
	}
	if g.storyActors[4].HP != 17 || g.storyActors[4].Lv != 3 {
		t.Fatalf("Hano persistent battle progress was not overlaid: %#v", g.storyActors[4])
	}
	g.materializeStoryGroup(1)
	if len(g.storyActors) != 11 {
		t.Fatalf("SPAWN1 frontier=%d, want 11", len(g.storyActors))
	}
	g.materializeStoryGroup(2)
	if len(g.storyActors) != 21 {
		t.Fatalf("SPAWN2 frontier=%d, want 21", len(g.storyActors))
	}
	for slot := 11; slot <= 16; slot++ {
		if g.storyActors[slot].X != 9 || g.storyActors[slot].Y != 20 {
			t.Fatalf("ACT12 source slot%d pre-position=(%d,%d), want stacked (9,20)", slot, g.storyActors[slot].X, g.storyActors[slot].Y)
		}
	}
}

func TestChapter2PreLoadCHUsesSixMemberJoinOrderAndGroupOneFrontier(t *testing.T) {
	beats, issues, err := campaign.CompileHandlerBinding(assetPath("assets/cutscenes/bindings/ch02_pre.json"))
	if err != nil || len(issues) != 0 || len(beats) == 0 || beats[0].LoadCH == nil {
		t.Fatalf("compile ch02_pre err=%v issues=%#v first=%#v", err, issues, beats)
	}
	g := &Game{}
	if err := g.applyLoadCH(beats[0].LoadCH); err != nil {
		t.Fatal(err)
	}
	if len(g.storyActors) != 6 {
		t.Fatalf("ch02_pre initial runtime slots=%d, want six party members", len(g.storyActors))
	}
	for slot, id := range []int{0, 9, 4, 30, 1, 8} {
		if g.storyActors[slot].Fig != id {
			t.Fatalf("ch02_pre party slot%d fig=%d, want %d", slot, g.storyActors[slot].Fig, id)
		}
	}
	g.materializeStoryGroup(1)
	if len(g.storyActors) != 15 {
		t.Fatalf("SPAWN1 frontier=%d, want 6 party + 9 FDFIELD records", len(g.storyActors))
	}
	if g.storyActors[6].Fig != 2 || g.storyActors[6].Camp != battle.Ally {
		t.Fatalf("slot6 after SPAWN1 = %#v, want Tino ally", g.storyActors[6])
	}
}

func TestCh15CandidateBindingCompilesForChapter16RuntimeButRemainsDataOnly(t *testing.T) {
	bindingPath := assetPath("assets/cutscenes/bindings/ch15_post_candidate.json")
	beats, issues, err := campaign.CompileHandlerBinding(bindingPath)
	if err != nil || len(issues) != 0 {
		t.Fatalf("ch15 candidate fixed-context compile err=%v issues=%#v", err, issues)
	}
	if len(beats) == 0 || beats[0].Op != "runtime_context" || beats[0].RuntimeContext == nil ||
		beats[0].RuntimeContext.SlotCount != 76 {
		t.Fatalf("candidate runtime context=%#v", beats)
	}
	var branch *campaign.Beat
	for _, beat := range beats {
		if beat.Op == "if" {
			branch = &beat
			break
		}
	}
	if branch == nil || branch.Condition == nil || branch.Condition.Op != "native_any_of" || len(branch.Condition.Any) != 2 || len(branch.Else) != 1 || branch.Else[0].Condition == nil || branch.Else[0].Condition.Op != "native_record_word_gte" {
		t.Fatalf("candidate raw CFG=%#v", beats)
	}
	nestedThen := branch.Else[0].Then
	if len(nestedThen) < 2 || nestedThen[0].Op != "dialog" ||
		nestedThen[len(nestedThen)-1].Op != "join" || nestedThen[len(nestedThen)-1].CharID != 18 {
		t.Fatalf("0x23b21 branch must own dialog4 and conditional JOIN18: %#v", branch.Else[0].Then)
	}
	for _, beat := range beats {
		if beat.Op == "join" {
			t.Fatalf("JOIN18 must not remain on the unconditional 0x23b52 tail: %#v", beats)
		}
	}
}

func TestFilterScenarioPartyUsesJoinMembership(t *testing.T) {
	sc := &battle.Scenario{
		Party:       []battle.PartyMember{{Fig: 0}, {Fig: 9}, {Fig: 30}, {Fig: 75}},
		DeployCells: [][2]int{{1, 10}, {2, 20}, {3, 30}, {4, 40}},
	}
	filterScenarioParty(sc, map[int]bool{0: true, 9: true})
	if len(sc.Party) != 2 || sc.Party[0].Fig != 0 || sc.Party[1].Fig != 9 {
		t.Fatalf("party filter ignored JOIN membership: %#v", sc.Party)
	}
	if len(sc.DeployCells) != 2 || sc.DeployCells[0] != [2]int{1, 10} || sc.DeployCells[1] != [2]int{2, 20} {
		t.Fatalf("party deploy cells drifted after membership filter: %#v", sc.DeployCells)
	}

	direct := &battle.Scenario{Party: []battle.PartyMember{{Fig: 0}, {Fig: 9}}}
	filterScenarioParty(direct, nil)
	if len(direct.Party) != 2 {
		t.Fatalf("direct scenario start must preserve authored party: %#v", direct.Party)
	}
}

func TestBeatSpawnActivatesOnlyItsRosterGroup(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{{Op: "spawn", Group: 3}})
	g.storyActors = []battle.Unit{
		{Group: 1, OnField: false},
		{Group: 3, OnField: false},
		{Group: 3, OnField: false},
		{Group: 5, OnField: false},
	}
	g.beatAdvance()
	if g.storyActors[0].OnField || !g.storyActors[1].OnField || !g.storyActors[2].OnField || g.storyActors[3].OnField {
		t.Fatalf("spawn group=3 activated wrong story slots: %#v", g.storyActors)
	}
}

func TestBeatSpawnCarriesRawPlacementGateIntoNativeGroupAppend(t *testing.T) {
	gate := 1
	g := newBeatTestGame(t, []campaign.Beat{{
		Op: "spawn", Group: 6, RawPlacementGate: &gate, Source: "0x34397",
	}})
	active := &battle.Unit{
		X: 1, Y: 1,
		MapSelectorKey:           2,
		HasMapSelectorKey:        true,
		NativeMapPresentation:    battle.NativeMapPresentationState{X: 1, Y: 1},
		HasNativeMapPresentation: true,
		NativeRecordByte5:        0,
		HasNativeRecordByte5:     true,
		NativeRecordByte6:        2,
		HasNativeRecordByte6:     true,
	}
	pending := &battle.Unit{
		Group: 6, Dir: 0, Lv: 2,
		MapSelectorKey:          3,
		HasMapSelectorKey:       true,
		NativeRecordByte5:       0,
		HasNativeRecordByte5:    true,
		NativeRecordByte6:       1,
		HasNativeRecordByte6:    true,
		NativePositionRecord:    battle.NativePositionRecord{XWord: 1, YWord: 1},
		HasNativePositionRecord: true,
		NativeConstructor: &battle.NativeConstructorTable{
			Branch: "high_class", Index: 0,
			Record: []byte{4, 5, 10, 0, 3, 6, 7, 8, 9, 0},
		},
		Inventory:            []int{0},
		Equipped:             []bool{true},
		InventorySlots:       []int{0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags: []int{0x40, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
	}
	g.st = &battle.State{
		W: 3, H: 3, Roster: []*battle.Unit{pending},
		NativeCompositionEventBytes: make([]byte, 9),
	}
	if err := g.st.BindNativeFutureItemRows(make([]byte, battle.NativeItemEffectRowSize)); err != nil {
		t.Fatal(err)
	}
	if err := g.st.AppendNativeMapSelectorBatch([]*battle.Unit{active}); err != nil {
		t.Fatal(err)
	}
	g.beatAdvance()
	if g.loadErr != "" {
		t.Fatalf("native gate spawn failed: %s", g.loadErr)
	}
	if len(g.st.Units) != 2 || g.st.Units[1].X != 1 || g.st.Units[1].Y != 1 {
		t.Fatalf("gate=1 runtime placement=%#v", g.st.Units)
	}
}

func TestBeatSpawnIntroWithoutNativeAssetsFailsClosedBeforeGroupAppend(t *testing.T) {
	gate := 0
	g := newBeatTestGame(t, []campaign.Beat{{
		Op: "spawn_intro", Group: 2, RawPlacementGate: &gate, Source: "0x3289b",
	}})
	active := &battle.Unit{
		X: 1, Y: 1,
		NativeMapPresentation:    battle.NativeMapPresentationState{X: 1, Y: 1},
		HasNativeMapPresentation: true,
		NativeRecordByte5:        0,
		HasNativeRecordByte5:     true,
		NativeRecordByte6:        2,
		HasNativeRecordByte6:     true,
	}
	pending := &battle.Unit{
		Group: 2, Dir: 0,
		MapSelectorKey:          4,
		HasMapSelectorKey:       true,
		NativeRecordByte5:       0,
		HasNativeRecordByte5:    true,
		NativeRecordByte6:       4,
		HasNativeRecordByte6:    true,
		NativePositionRecord:    battle.NativePositionRecord{XWord: 1, YWord: 1},
		HasNativePositionRecord: true,
	}
	g.st = &battle.State{
		W: 3, H: 3, Units: []*battle.Unit{active}, Roster: []*battle.Unit{pending},
		NativeCompositionEventBytes: make([]byte, 9),
	}
	g.beatAdvance()
	if g.loadErr == "" {
		t.Fatal("缺少原版 0x32999 視覺／音訊素材時未採失敗即關閉")
	}
	if len(g.st.Units) != 1 || g.st.Units[0] != active || g.beatDelay != 0 {
		t.Fatalf("失敗前已變更狀態：units=%#v delay=%d", g.st.Units, g.beatDelay)
	}
}

func TestBeatSpawnAppendsFDFIELDGroupInOriginalOrder(t *testing.T) {
	// Original 0x10b4e does not reveal preallocated units: it constructs every
	// matching FDFIELD record at unit_count, so the runtime slot identity is
	// the order groups were spawned. This is the map31 pattern (1, then 3, 5).
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "spawn", Group: 1}, {Op: "spawn", Group: 3}, {Op: "spawn", Group: 1},
	})
	g.storyActors = nil // LOADCH map31 has no group0 records.
	g.storyRoster = []battle.Unit{
		{Group: 1, Fig: 10}, {Group: 3, Fig: 30}, {Group: 1, Fig: 11}, {Group: 3, Fig: 9},
	}
	g.storySpawned = map[int]bool{0: true}
	g.beatAdvance()
	if got := len(g.storyActors); got != 4 {
		t.Fatalf("spawn constructed %d runtime units, want 4: %#v", got, g.storyActors)
	}
	for i, fig := range []int{10, 11, 30, 9} {
		if g.storyActors[i].Fig != fig || !g.storyActors[i].OnField {
			t.Fatalf("runtime slot %d = %#v, want on-field fig=%d", i, g.storyActors[i], fig)
		}
	}
}

func TestDeactivateResetAndRedrawPreserveHandlerBoundaries(t *testing.T) {
	slot := 0
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "deactivate_unit", Slot: &slot},
		{Op: "reset_pose", Ms: 20},
		{Op: "redraw", Frames: 1},
	})
	g.storyActors[0].OnField = true
	g.storyActors[0].Dir = 3
	g.beatAdvance()
	if g.storyActors[0].OnField || g.storyActors[0].Dir != 0 || g.beatDelay != 1 {
		t.Fatalf("deactivate/reset state = onField:%v dir:%d delay:%d", g.storyActors[0].OnField, g.storyActors[0].Dir, g.beatDelay)
	}
	g.tick(2)
	if g.beatDelay != 0 || g.beatIdx < 3 {
		t.Fatalf("reset/redraw boundaries did not advance: beat=%d delay=%d", g.beatIdx, g.beatDelay)
	}
}

func TestLayoutUnitsUsesCanonicalRuntimeSlotsAndCamera(t *testing.T) {
	layout := &campaign.HandlerLayout{
		Units: []campaign.HandlerUnitLayout{
			{Slot: 0, X: 8, Y: 3, Pose: 2},
			{Slot: 6, X: 8, Y: 1, Pose: 0},
		},
		CamX: 48,
		CamY: 0,
	}
	g := newBeatTestGame(t, []campaign.Beat{{Op: "layout_units", Layout: layout}})
	g.st = &battle.State{Units: make([]*battle.Unit, 7)}
	for i := range g.st.Units {
		g.st.Units[i] = &battle.Unit{X: 20 + i, Y: 30 + i, Dir: 3, OffX: 4, OffY: 5}
	}
	g.beatAdvance()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if got := g.st.Units[0]; got.X != 8 || got.Y != 3 || got.Dir != 2 || got.OffX != 0 || got.OffY != 0 {
		t.Fatalf("layout slot0 = %#v", got)
	}
	if got := g.st.Units[6]; got.X != 8 || got.Y != 1 || got.Dir != 0 {
		t.Fatalf("layout Tino slot6 = %#v", got)
	}
	if g.camX != 48 || g.camY != 0 {
		t.Fatalf("layout camera = (%v,%v), want (48,0)", g.camX, g.camY)
	}
}

func TestRuntimeContextAcceptsOnlyDeclaredPostBattleFrontiers(t *testing.T) {
	context := &campaign.HandlerRuntimeContext{SlotCounts: []int{15, 27}, StoryViewport: true}
	for _, count := range []int{15, 27} {
		g := newBeatTestGame(t, []campaign.Beat{{Op: "runtime_context", RuntimeContext: context}})
		g.st = &battle.State{Units: make([]*battle.Unit, count)}
		g.beatAdvance()
		if g.loadErr != "" || !g.storyBG {
			t.Fatalf("runtime count%d rejected: err=%q storyBG=%v", count, g.loadErr, g.storyBG)
		}
	}
	rejected := newBeatTestGame(t, []campaign.Beat{{Op: "runtime_context", RuntimeContext: context}})
	rejected.st = &battle.State{Units: make([]*battle.Unit, 16)}
	rejected.beatAdvance()
	if rejected.loadErr == "" {
		t.Fatal("undeclared 16-slot post-battle frontier was accepted")
	}
}

func TestMap0ActingUsesPartyThenSpawnedRuntimeSlots(t *testing.T) {
	resources, err := campaign.LoadActingResourceSet(assetPath("assets/cutscenes/acting/map32.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "act", Source: "0x3283a", Acting: resources[0]},
		{Op: "spawn_intro", Group: 1, Frames: 12},
		{Op: "act", Source: "0x328a5", Acting: resources[1]},
		{Op: "spawn_intro", Group: 2, Frames: 12},
		{Op: "act", Source: "0x328c5", Acting: resources[2]},
		{Op: "act", Source: "0x3290d", Acting: resources[5]},
	})
	// Original JOIN chronology fixes party slots as Sol, Yuni, Ares, Gaia.
	g.storyActors = []battle.Unit{
		{Fig: 0, X: 7, Y: 20, OnField: true},
		{Fig: 9, X: 10, Y: 21, OnField: true},
		{Fig: 4, X: 8, Y: 22, OnField: true},
		{Fig: 30, X: 11, Y: 23, OnField: true},
	}
	g.storyRoster = []battle.Unit{
		{Group: 1, Fig: 96, X: 1, Y: 3}, {Group: 1, Fig: 96, X: 2, Y: 1},
		{Group: 1, Fig: 96, X: 4, Y: 1}, {Group: 1, Fig: 96, X: 6, Y: 0},
		{Group: 2, Fig: 96, X: 1, Y: 21}, {Group: 2, Fig: 96, X: 2, Y: 22},
		{Group: 2, Fig: 96, X: 3, Y: 22}, {Group: 2, Fig: 96, X: 4, Y: 23},
	}
	g.storySpawned = map[int]bool{0: true}
	g.beatAdvance()
	g.tick(168) // legacy authored approximation: ACTs=144 ticks plus two 12-tick intro waits.
	if g.loadErr != "" || len(g.storyActors) != 12 {
		t.Fatalf("map0 acting sequence failed: err=%q actors=%d", g.loadErr, len(g.storyActors))
	}
	want := [][2]int{
		{7, 14}, {10, 15}, {8, 16}, {11, 17},
		{1, 4}, {2, 2}, {4, 2}, {6, 1},
		{3, 18}, {4, 23}, {2, 18}, {5, 19},
	}
	for slot, xy := range want {
		u := g.storyActors[slot]
		if u.X != xy[0] || u.Y != xy[1] {
			t.Fatalf("map0 runtime slot %d fig%d=(%d,%d), want (%d,%d)", slot, u.Fig, u.X, u.Y, xy[0], xy[1])
		}
	}
}

func TestEnterTownClearsCompletedBattlePresentation(t *testing.T) {
	c := &campaign.Campaign{
		Start: "town",
		Nodes: map[string]*campaign.Node{
			"town": {Type: "town", Town: "羅德鎮", Options: []campaign.Option{{Label: "出口", To: "prep"}}},
			"prep": {Type: "preparation"},
		},
	}
	g := &Game{
		camp:   campaign.NewRunner(c),
		st:     &battle.State{Units: []*battle.Unit{{Fig: 0, OnField: true}}},
		sel:    &battle.Unit{Fig: 0, OnField: true},
		dialog: []battle.DialogLine{{Speaker: 0, Text: "上一戰殘留"}},
	}
	g.enterNode()
	if g.st != nil || g.sel != nil || len(g.dialog) != 0 {
		t.Fatalf("town retained completed battle presentation: state=%#v sel=%#v dialog=%#v", g.st, g.sel, g.dialog)
	}
}
