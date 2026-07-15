// beatrunner_test.go — BeatRunner(doc50)純邏輯測試:不碰 ebiten 顯示/輸入,
// 只驗證 beatStart/beatAdvance 與 stepCamPan/stepStoryWalks/stepActJob/stepFocusUnit/stepFade
// 這幾個「逐幀推進」method 的狀態機是否照 op 表正確銜接。
package main

import (
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
		g.stepStoryWalks()
		g.stepActJob()
		g.stepFocusUnit()
		g.stepFade()
		g.stepCamPan()
		if g.beatDelay > 0 {
			g.beatDelay--
			if g.beatDelay == 0 {
				g.beatAdvance()
			}
		}
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
	}}}})
	g.beatAdvance()
	g.tick(2)
	if g.actJob == nil {
		t.Fatal("zero-special frame advanced before original delay(1)+delay(2)")
	}
	g.tick(1)
	if g.actJob != nil {
		t.Fatal("zero-special frame did not advance after three ticks")
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

func TestBeatSyncPartyPersistsProgressAndClearsBattleState(t *testing.T) {
	chapter := 1
	g := newBeatTestGame(t, []campaign.Beat{{Op: "sync_party"}, {Op: "set_chapter", Chapter: &chapter}})
	g.partyMembers = map[int]bool{0: true, 9: true}
	g.st = &battle.State{Units: []*battle.Unit{
		{Camp: battle.Own, Fig: 0, Name: "索爾", Lv: 3, HP: 7, MaxHP: 50, MP: 2, MaxMP: 12, AP: 18, Exp: 42, Acted: true, Poisoned: true, PoisonTurns: 3, Spells: []int{1, 2}},
		{Camp: battle.Own, Fig: 9, Name: "悠妮", Lv: 2, HP: 0, MaxHP: 31, MP: 0, MaxMP: 19, Paralyzed: true, ParalyzeTurns: 2},
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
	if sol.HP != 7 || sol.MP != 12 || sol.Lv != 3 || sol.Exp != 42 || sol.Acted || sol.Poisoned || len(sol.Spells) != 2 {
		t.Fatalf("survivor snapshot = %#v", sol)
	}
	yuni := g.partyRoster[9]
	if yuni.HP != 31 || yuni.MP != 19 || yuni.Paralyzed {
		t.Fatalf("defeated member was not revived/cleared: %#v", yuni)
	}

	fresh := &battle.State{Units: []*battle.Unit{{Camp: battle.Own, Fig: 0, X: 11, Y: 22, Group: 4, OnField: true, HP: 99, MaxHP: 99}}}
	g.applyPersistentParty(fresh)
	got := fresh.Units[0]
	if got.Lv != 3 || got.HP != 7 || got.MP != 12 || got.X != 11 || got.Y != 22 || got.Group != 4 || !got.OnField {
		t.Fatalf("persistent overlay lost progression or deployment: %#v", got)
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

func TestBeatAnyUnitAliveChoosesOneArmAndKeepsCommonTail(t *testing.T) {
	itemID := 0xc6
	condition := &campaign.BeatCondition{Op: "any_unit_alive", UnitSlots: []int{5, 6, 7, 8, 9, 10}}
	branch := campaign.Beat{
		Op: "if", Condition: condition,
		Then: []campaign.Beat{{Op: "join", CharID: 4}},
		Else: []campaign.Beat{{Op: "grant_item", ItemID: &itemID}},
	}
	common := campaign.Beat{Op: "join", CharID: 9}

	alive := newBeatTestGame(t, []campaign.Beat{branch, common})
	alive.st = &battle.State{Units: make([]*battle.Unit, 12)}
	alive.st.Units[0] = &battle.Unit{Camp: battle.Own, Inventory: []int{1}}
	alive.st.Units[10] = &battle.Unit{HP: 1}
	alive.beatAdvance()
	if alive.loadErr != "" || !alive.partyMembers[4] || !alive.partyMembers[9] {
		t.Fatalf("alive arm did not run: err=%q party=%#v", alive.loadErr, alive.partyMembers)
	}
	if got := alive.st.Units[0].Inventory; len(got) != 1 {
		t.Fatalf("alive arm incorrectly granted item: %#v", got)
	}
	if len(alive.beats) != 3 || alive.beats[2].Op != "join" {
		t.Fatalf("selected arm/common tail splice = %#v", alive.beats)
	}

	dead := newBeatTestGame(t, []campaign.Beat{branch, common})
	dead.st = &battle.State{Units: make([]*battle.Unit, 12)}
	dead.st.Units[0] = &battle.Unit{Camp: battle.Own}
	dead.st.Units[4] = &battle.Unit{HP: 1}
	dead.st.Units[11] = &battle.Unit{HP: 1}
	dead.beatAdvance()
	if dead.loadErr != "" || dead.partyMembers[4] || !dead.partyMembers[9] {
		t.Fatalf("dead arm selection = err=%q party=%#v", dead.loadErr, dead.partyMembers)
	}
	if got := dead.st.Units[0].Inventory; len(got) != 1 || got[0] != 0xc6 {
		t.Fatalf("slots outside 5..10 affected condition; reward=%#v", got)
	}
}

func TestBeatAnyUnitAliveFailsClosedWithoutCompleteRoster(t *testing.T) {
	condition := &campaign.BeatCondition{Op: "any_unit_alive", UnitSlots: []int{5, 6, 7, 8, 9, 10}}
	beats := []campaign.Beat{{
		Op: "if", Condition: condition,
		Else: []campaign.Beat{{Op: "join", CharID: 9}},
	}}
	short := &battle.State{Units: make([]*battle.Unit, 10)}
	short.Units[5] = &battle.Unit{HP: 1} // must still reject before selecting the alive arm.
	for _, st := range []*battle.State{nil, short} {
		g := newBeatTestGame(t, beats)
		g.st = st
		g.beatAdvance()
		if g.loadErr == "" || g.partyMembers[9] {
			t.Fatalf("incomplete runtime state did not fail closed: state=%#v err=%q", st, g.loadErr)
		}
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

func TestActivateResetAndRedrawPreserveHandlerBoundaries(t *testing.T) {
	slot := 0
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "activate_unit", Slot: &slot},
		{Op: "reset_pose", Ms: 20},
		{Op: "redraw", Frames: 1},
	})
	g.storyActors[0].OnField = false
	g.storyActors[0].Dir = 3
	g.beatAdvance()
	if !g.storyActors[0].OnField || g.storyActors[0].Dir != 0 || g.beatDelay != 1 {
		t.Fatalf("activate/reset state = onField:%v dir:%d delay:%d", g.storyActors[0].OnField, g.storyActors[0].Dir, g.beatDelay)
	}
	g.tick(2)
	if g.beatDelay != 0 || g.beatIdx < 3 {
		t.Fatalf("reset/redraw boundaries did not advance: beat=%d delay=%d", g.beatIdx, g.beatDelay)
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
	g.tick(168) // ACTs=144 ticks plus two original 12-step spawn-intro loops.
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
