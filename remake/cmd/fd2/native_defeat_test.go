package main

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func TestNativeDefeatBattleResultWithoutGame(t *testing.T) {
	var g *Game
	if g.confirmBattleResult() {
		t.Fatal("沒有遊戲狀態仍確認戰鬥結果")
	}
}

func TestNativePlayerRecordSelectionClearsExperienceBeforeActionGate(t *testing.T) {
	for _, tc := range []struct {
		name           string
		camp           battle.Camp
		paralyzed, raw bool
		want           int
	}{
		{"acted own", battle.Own, false, true, 0},
		{"paralyzed own", battle.Own, true, true, 0},
		{"enemy inspection", battle.Enemy, false, true, 0},
		{"authored compatibility", battle.Own, false, false, 15},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := &battle.Unit{X: 1, Y: 1, HP: 10, MaxHP: 10, Camp: tc.camp, Acted: true, Paralyzed: tc.paralyzed, OnField: true, HasNativeRecordByte6: tc.raw}
			g := &Game{st: &battle.State{W: 4, H: 4, Units: []*battle.Unit{u}, NativeExperienceAccumulator: 15}, curX: 1, curY: 1}
			g.confirm()
			if g.st.NativeExperienceAccumulator != tc.want {
				t.Fatalf("選取 gate 未清累計：%d", g.st.NativeExperienceAccumulator)
			}
		})
	}
	g := &Game{st: &battle.State{W: 4, H: 4, NativeExperienceAccumulator: 15}, curX: 1, curY: 1}
	g.confirm()
	if g.st.NativeExperienceAccumulator != 15 {
		t.Fatal("空地系統選單誤清累計")
	}
}

func TestNativeDefeatRequiresDrawAndFixedHolds(t *testing.T) {
	baseline := bytes.Repeat([]byte{63}, 256*3)
	palette, err := fdother.VGAPaletteFromDAC(baseline)
	if err != nil {
		t.Fatal(err)
	}
	j := &nativeDefeatJob{plan: fdother.NativePendingCode1PresentationPlan(), pos: 1,
		vga: bytes.Repeat([]byte{7}, indexedmap.NativeMapVGASize), dac: append([]byte(nil), baseline...), baseline: baseline, palette: palette,
		frames: []fdother.Frame{
			{X: 1, Y: 1, Width: 1, Height: 1, Indexed: []byte{11}, Mask: []byte{255}},
			{X: 2, Y: 1, Width: 1, Height: 1, Indexed: []byte{12}, Mask: []byte{255}},
		}}
	g := &Game{nativeDefeat: j}
	now := time.Unix(1, 0)
	g.stepNativeDefeat(now.Add(time.Hour))
	if j.pos != 1 || !j.waitStart.IsZero() {
		t.Fatal("未呈現畫面便消耗停留")
	}
	screen := ebiten.NewImage(640, 400)
	g.Draw(screen)
	g.stepNativeDefeat(now)
	g.stepNativeDefeat(now.Add(nativeBIOSTickPeriod - time.Nanosecond))
	if j.pos != 1 {
		t.Fatal("初始1 tick提前結束")
	}
	now = now.Add(nativeBIOSTickPeriod)
	g.stepNativeDefeat(now)
	for k := 0; k < 64+65; k++ {
		if j.ramp == nil {
			t.Fatalf("色盤第%d步提早結束", k)
		}
		g.Draw(screen)
		g.stepNativeDefeat(now)
	}
	if j.pos != 6 || j.ramp != nil || !bytes.Equal(j.dac, baseline) || j.vga[321] != 11 || j.vga[0] != 0 {
		t.Fatalf("第0幀/色盤/清底錯誤：pos=%d", j.pos)
	}
	g.Draw(screen)
	g.stepNativeDefeat(now)
	g.stepNativeDefeat(now.Add(9*nativeBIOSTickPeriod - time.Nanosecond))
	if j.vga[322] != 0 {
		t.Fatal("第1幀在9 ticks前疊入")
	}
	now = now.Add(9 * nativeBIOSTickPeriod)
	g.stepNativeDefeat(now)
	if j.pos != 8 || j.vga[321] != 11 || j.vga[322] != 12 {
		t.Fatal("第1幀未保留第0幀")
	}
	g.stepNativeDefeat(now.Add(time.Hour))
	if !j.waitStart.IsZero() {
		t.Fatal("第1幀未呈現便開始36 ticks")
	}
	g.Draw(screen)
	g.stepNativeDefeat(now)
	g.stepNativeDefeat(now.Add(36*nativeBIOSTickPeriod - time.Nanosecond))
	if g.nativeDefeat == nil {
		t.Fatal("第1幀36 ticks提前結束")
	}
}

func TestNativeDefeatMissingAssetsCannotConfirmRetreat(t *testing.T) {
	c := &campaign.Campaign{Start: "battle", Nodes: map[string]*campaign.Node{"battle": {
		Type: "battle", NativeDefeatReturnTitle: &campaign.NativeDefeatReturnTitleConfig{Handler: "0x22e5c", Resource: 79, Return: "full_title_sequence"},
	}}}
	g := &Game{camp: campaign.NewRunner(c), result: "lose"}
	g.beginNativeDefeatIfConfigured()
	if g.loadErr == "" || !g.startupBlocked || g.nativeDefeat != nil || g.confirmBattleResult() || g.camp.NodeID() != "battle" {
		t.Fatal("缺少原生素材仍能發布呈現或確認敗北捷徑")
	}
}

// 此測試為重製 E1：直接進章12並明示死亡 bit0，不能充當正常原版 PLAYER-E2。
// 正常章內動作與兩側受控 RNG 由 TestChapterParityReplay 的敗北收據另驗。
func TestNativeDefeatChapter12ReturnsTitleAndStartResetsCampaign(t *testing.T) {
	pack := filepath.Clean("../../generated-assets/fd2-original-b97caf22")
	if _, err := os.Stat(filepath.Join(pack, "animations/fdother_079_pending_code1/bank.json")); err != nil {
		t.Skip(err)
	}
	t.Setenv("FD2_ASSET_PACK", pack)
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_SEED", "4")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_CAMP_NODE", "battle_ch12")
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	save := filepath.Join(t.TempDir(), "FD2.SAV")
	if err := os.WriteFile(save, []byte("敗北不得改寫既有存檔"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_NATIVE_SAVE", save)
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if len(g.st.Units) <= 14 || g.st.Units[14].Fig != 17 {
		t.Fatal("第十二章建構槽14身分不符")
	}
	g.st.Units[14].HP = 0
	g.st.Units[14].NativeRecordByte5 |= 1
	g.checkResult()
	if g.nativeDefeat == nil || g.result != "lose" || g.loadErr != "" {
		t.Fatalf("正式 checkResult 未啟動敗北：%s", g.loadErr)
	}
	if g.confirmBattleResult() {
		t.Fatal("Enter略過原生提示")
	}
	screen := ebiten.NewImage(640, 400)
	now := time.Unix(1, 0)
	for k := 0; g.nativeDefeat != nil && k < 500; k++ {
		g.Draw(screen)
		g.stepNativeDefeat(now)
		now = now.Add(nativeBIOSTickPeriod)
	}
	if g.nativeDefeat != nil || g.titlePhase != "cutscene" || g.cutIdx != 0 || g.titleAssets == nil || len(g.titleAssets.aniClips) == 0 || g.st != nil || g.aiBusy {
		t.Fatal("未返回完整標題或遺留舊戰鬥")
	}
	raw, err := os.ReadFile(save)
	if err != nil || sha256.Sum256(raw) != sha256.Sum256([]byte("敗北不得改寫既有存檔")) {
		t.Fatal("敗北改寫存檔")
	}
	// Draw 的黑 DAC 延後遮罩不得覆蓋返回標題。
	g.enterTitleMenu()
	g.nativeMapDAC = make([]byte, 768)
	g.Draw(screen)
	g.gold = 999
	if !g.applyTitleMenuEvent(TitleMenuConfirm) {
		t.Fatal("START確認未消費")
	}
	for i := 0; i < 24; i++ {
		g.applyTitleMenuEvent(TitleMenuTick)
	}
	if g.titlePhase != "" || g.camp.NodeID() != g.camp.C.Start || g.gold != 0 || g.handlerChapter != 0 || g.titleNeedsNewCampaign || g.loadErr != "" {
		t.Fatalf("START 沿用敗北戰役：node=%s gold=%d err=%s", g.camp.NodeID(), g.gold, g.loadErr)
	}
}
