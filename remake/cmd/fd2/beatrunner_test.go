// beatrunner_test.go — BeatRunner(doc50)純邏輯測試:不碰 ebiten 顯示/輸入,
// 只驗證 beatStart/beatAdvance 與 stepCamPan/stepStoryWalks/stepActJob/stepFade
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
		g.stepFade()
		g.stepCamPan()
	}
}

func TestBeatPanMovesCamera(t *testing.T) {
	g := newBeatTestGame(t, []campaign.Beat{
		{Op: "pan", X: 100, Y: 200, Frames: 10},
	})
	g.beatAdvance() // 啟動第 0 拍
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
	// Exact decoded 0x66 from ch00_pre call-site 0x32461.  Slots 13,16,17
	// are all fig69 guards; a Fig-only lookup used to animate slot13 twice.
	slot16, slot17 := 16, 17
	frames := []campaign.ActingFrame{
		{Beats: 8, Special: true, Units: []campaign.ActingUnit{{Slot: &slot16, Pose: 2}, {Slot: &slot17, Pose: 2}}},
		{Beats: 8, Special: true, Units: []campaign.ActingUnit{{Slot: &slot16, Pose: 0}, {Slot: &slot17, Pose: 0}}},
		{Beats: 8, Special: true, Units: []campaign.ActingUnit{{Slot: &slot16, Pose: 2}, {Slot: &slot17, Pose: 2}}},
		{Beats: 8, Special: true, Units: []campaign.ActingUnit{{Slot: &slot16, Pose: 3}, {Slot: &slot17, Pose: 3}}},
		{Beats: 8, Special: true, Units: []campaign.ActingUnit{{Slot: &slot16, Pose: 1}, {Slot: &slot17, Pose: 1}}},
		{Beats: 0, Special: true, Units: []campaign.ActingUnit{{Slot: &slot16, Pose: 1}, {Slot: &slot17, Pose: 1}}},
	}
	g := newBeatTestGame(t, []campaign.Beat{{Op: "act", Acting: frames}})
	g.storyActors = make([]battle.Unit, 18)
	for i := range g.storyActors {
		g.storyActors[i] = battle.Unit{Fig: 69, OnField: true, Dir: 3}
	}
	g.beatAdvance()
	g.tick(41) // five 8-tick special frames plus the zero-beat terminator
	if g.storyActors[13].Dir != 3 {
		t.Fatalf("duplicate fig fallback touched slot13: dir=%d", g.storyActors[13].Dir)
	}
	if g.storyActors[16].Dir != 1 || g.storyActors[17].Dir != 1 {
		t.Fatalf("decoded slots 16/17 were not targeted: dirs=(%d,%d)", g.storyActors[16].Dir, g.storyActors[17].Dir)
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
		{Op: "spawn", Group: 1},         // 非阻塞 stub
		{Op: "join", Fig: 0},            // 非阻塞 stub
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
