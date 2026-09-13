package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// 這組測試只驗證索引 UI 與道具交易；測試環境的公開最小素材包刻意不含
// FDOTHER 音效。FD2_MUTE 下只豁免這一個已知、與畫面／交易無關的啟動錯誤，
// 其餘來源缺口仍照正式 loadGame 失敗即關閉。
func loadChapterOneDeathRewardBattle(t *testing.T) *Game {
	t.Helper()
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_CAMP_NODE", journeyBattleNode)
	t.Setenv("FD2_ASSET_PACK", assetPackRootWithLocales(
		t, "../../generated-assets/fd2-original-b97caf22",
	))
	g := loadGame()
	if strings.HasPrefix(g.loadErr, "separated UI sounds:") {
		g.loadErr = ""
	}
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if g.st == nil || g.sc == nil {
		t.Fatal("第一關戰場或劇本沒有建立")
	}
	if !pump(t, g, 20000, func() bool {
		if len(g.dialog) > 0 && g.battleEvent == nil {
			g.dlgAdvance()
		}
		return deathBattleIdle(g)
	}) {
		t.Fatalf("第一關的開場工作沒有收掉\n阻塞：%s", ch01Blockers(g))
	}
	return g
}

type nativeDeathRewardInventorySnapshot struct {
	inventory []int
	equipped  []bool
	slots     []int
	flags     []int
}

func makeNativeDeathRewardKiller(t *testing.T, g *Game) *battle.Unit {
	t.Helper()
	for _, unit := range g.st.Units {
		if unit == nil || unit.Camp != battle.Own || !unit.Alive() || !unit.OnField || !unit.HasBattleFig {
			continue
		}
		unit.Inventory = []int{0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17}
		unit.Equipped = []bool{true, false, true, false, false, true, false, true}
		unit.InventorySlots = append([]int(nil), unit.Inventory...)
		unit.NativeInventoryFlags = []int{0x40, 0, 0x40, 0, 0, 0x40, 0, 0x40}
		if err := battle.ValidateNativeInventoryProjection(unit); err != nil {
			t.Fatalf("完整原生道具欄 fixture 無效：%v", err)
		}
		return unit
	}
	t.Fatal("第一關找不到具原生頭像的我方擊殺者")
	return nil
}

func snapshotNativeDeathRewardInventory(unit *battle.Unit) nativeDeathRewardInventorySnapshot {
	return nativeDeathRewardInventorySnapshot{
		inventory: slices.Clone(unit.Inventory),
		equipped:  slices.Clone(unit.Equipped),
		slots:     slices.Clone(unit.InventorySlots),
		flags:     slices.Clone(unit.NativeInventoryFlags),
	}
}

func assertNativeDeathRewardInventory(t *testing.T, unit *battle.Unit, want nativeDeathRewardInventorySnapshot) {
	t.Helper()
	if !slices.Equal(unit.Inventory, want.inventory) ||
		!slices.Equal(unit.Equipped, want.equipped) ||
		!slices.Equal(unit.InventorySlots, want.slots) ||
		!slices.Equal(unit.NativeInventoryFlags, want.flags) {
		t.Fatalf("道具交易非原子：inventory=%#v equipped=%v slots=%#v flags=%#v",
			unit.Inventory, unit.Equipped, unit.InventorySlots, unit.NativeInventoryFlags)
	}
}

// driveNativeDeathRewardUI 讓正式 Draw 發布每張索引畫面，再由正式 Update
// 推進生命週期。輸入決策仍由各測試在已穩定的 owner 邊界送入。
func driveNativeDeathRewardUI(t *testing.T, g *Game, screen *ebiten.Image, done func() bool) {
	t.Helper()
	for frame := 0; frame < 1024 && !done(); frame++ {
		job := g.nativeClassUIJob
		g.Draw(screen)
		if job != nil && !job.drawn {
			t.Fatalf("第 %d 幀未由正式 Draw 確認原生 UI 發布", frame)
		}
		if err := g.Update(); err != nil {
			t.Fatalf("第 %d 幀正式 Update：%v", frame, err)
		}
		if g.loadErr != "" {
			t.Fatalf("第 %d 幀原生獎勵 UI 失敗：%s", frame, g.loadErr)
		}
	}
	if !done() {
		t.Fatal("原生死亡獎勵 UI 超過 1024 幀仍未抵達預期邊界")
	}
}

func beginFullNativeDeathRewardPrompt(t *testing.T, g *Game, killer *battle.Unit, rewards ...int) *ebiten.Image {
	t.Helper()
	for _, reward := range rewards {
		g.grantNativeDeathReward(0, reward, killer)
	}
	if len(g.pendingNativeDeathRewards) != len(rewards) {
		t.Fatalf("完整道具欄未依序排入獎勵：got=%d want=%d", len(g.pendingNativeDeathRewards), len(rewards))
	}
	g.finishSuccessfulUnitAction(killer, nil)
	if g.nativeDeathRewardUI == nil || !g.nativeSystemEndTurnConfirm || g.nativeClassUIJob == nil || killer.Acted {
		t.Fatalf("未停在丟棄提示：reward=%#v confirm=%v job=%v acted=%v err=%q",
			g.nativeDeathRewardUI, g.nativeSystemEndTurnConfirm, g.nativeClassUIJob != nil, killer.Acted, g.loadErr)
	}
	screen := ebiten.NewImage(logicalW, logicalH)
	driveNativeDeathRewardUI(t, g, screen, func() bool {
		return g.nativeClassUIJob == nil && g.nativeSystemEndTurnConfirm
	})
	return screen
}

func TestNativeDeathRewardFullInventoryNoAndEscapeRemainAtomic(t *testing.T) {
	for _, route := range []string{"NO", "Escape"} {
		t.Run(route, func(t *testing.T) {
			g := loadChapterOneDeathRewardBattle(t)
			killer := makeNativeDeathRewardKiller(t, g)
			before := snapshotNativeDeathRewardInventory(killer)
			screen := beginFullNativeDeathRewardPrompt(t, g, killer, 0xd3)

			// Escape 與選到 NO 都由 ringInput 匯入同一個原版取消分支。
			g.cancelNativeSystemEndTurn()
			assertNativeDeathRewardInventory(t, killer, before)
			driveNativeDeathRewardUI(t, g, screen, func() bool {
				return g.nativeDeathRewardUI == nil && g.nativeSystemEndTurnUI == nil
			})
			assertNativeDeathRewardInventory(t, killer, before)
			if !killer.Acted || len(g.pendingNativeDeathRewards) != 0 {
				t.Fatalf("取消後未回到行動收尾：acted=%v pending=%d", killer.Acted, len(g.pendingNativeDeathRewards))
			}
		})
	}
}

func TestNativeDeathRewardDiscardCommitsOnlyAfterItemPanelClose(t *testing.T) {
	g := loadChapterOneDeathRewardBattle(t)
	killer := makeNativeDeathRewardKiller(t, g)
	before := snapshotNativeDeathRewardInventory(killer)
	screen := beginFullNativeDeathRewardPrompt(t, g, killer, 0xd3)

	g.confirmNativeSystemEndTurn()
	driveNativeDeathRewardUI(t, g, screen, func() bool {
		return g.itemOpen && !g.itemClosing && g.itemAnimStep == 11
	})
	if g.sel != killer || g.nativeItemPanel == nil || !g.drawNativeItemPanel(screen) {
		t.Fatal("YES 未進入擊殺者的實際八格原生道具面板")
	}
	assertNativeDeathRewardInventory(t, killer, before)

	for step := 0; step < 3; step++ {
		if !g.handleNativeDeathRewardItemInput(nativeDeathRewardInput{down: true}) {
			t.Fatalf("第 %d 次 Down 未被獎勵選擇器擁有", step)
		}
	}
	if g.itemSel != 3 || !g.handleNativeDeathRewardItemInput(nativeDeathRewardInput{confirm: true}) {
		t.Fatalf("選擇器未停在原始索引 3：itemSel=%d", g.itemSel)
	}
	assertNativeDeathRewardInventory(t, killer, before)
	driveNativeDeathRewardUI(t, g, screen, func() bool { return g.nativeDeathRewardUI == nil })

	wantItems := []int{0x10, 0x11, 0x12, 0x14, 0x15, 0x16, 0x17, 0xd3}
	wantEquipped := []bool{true, false, true, false, true, false, true, false}
	if !slices.Equal(killer.Inventory, wantItems) || !slices.Equal(killer.InventorySlots, wantItems) ||
		!slices.Equal(killer.Equipped, wantEquipped) || killer.NativeInventoryFlags[7] != 0 {
		t.Fatalf("丟棄索引 3 後結果錯誤：inventory=%#v slots=%#v equipped=%v flags=%#v",
			killer.Inventory, killer.InventorySlots, killer.Equipped, killer.NativeInventoryFlags)
	}
	if !killer.Acted || len(g.pendingNativeDeathRewards) != 0 {
		t.Fatalf("交易後未回到行動收尾：acted=%v pending=%d", killer.Acted, len(g.pendingNativeDeathRewards))
	}
}

func TestNativeDeathRewardSelectorCancelAndSequentialQueue(t *testing.T) {
	g := loadChapterOneDeathRewardBattle(t)
	killer := makeNativeDeathRewardKiller(t, g)
	before := snapshotNativeDeathRewardInventory(killer)
	screen := beginFullNativeDeathRewardPrompt(t, g, killer, 0xd3, 0xd4)

	// 第一筆 YES 後在道具選擇器按 Escape，應顯示 0x1B2 並零變更，接著
	// 由同一個 action continuation 開啟第二筆提示。
	g.confirmNativeSystemEndTurn()
	driveNativeDeathRewardUI(t, g, screen, func() bool {
		return g.itemOpen && !g.itemClosing && g.itemAnimStep == 11
	})
	if !g.handleNativeDeathRewardItemInput(nativeDeathRewardInput{cancel: true}) {
		t.Fatal("道具選擇器 Escape 未被擁有")
	}
	driveNativeDeathRewardUI(t, g, screen, func() bool {
		return g.nativeSystemEndTurnConfirm && g.nativeClassUIJob == nil && len(g.pendingNativeDeathRewards) == 0
	})
	assertNativeDeathRewardInventory(t, killer, before)
	if killer.Acted || g.nativeDeathRewardUI == nil || g.nativeDeathRewardUI.item != 0xd4 {
		t.Fatalf("第二筆獎勵未循序接手：acted=%v reward=%#v", killer.Acted, g.nativeDeathRewardUI)
	}

	// 第二筆選 NO，兩筆都放棄後才真正結束行動。
	g.cancelNativeSystemEndTurn()
	driveNativeDeathRewardUI(t, g, screen, func() bool { return g.nativeDeathRewardUI == nil })
	assertNativeDeathRewardInventory(t, killer, before)
	if !killer.Acted {
		t.Fatal("序列獎勵全部結束後仍未完成擊殺者行動")
	}
}

func TestNativeDeathRewardMissingProvenanceFailsClosedWithoutMutation(t *testing.T) {
	killer := &battle.Unit{
		Camp: battle.Own, Inventory: []int{1, 2, 3, 4, 5, 6, 7, 8},
		Equipped:             make([]bool, 8),
		InventorySlots:       []int{1, 2, 3, 4, 5, 6, 7, 8},
		NativeInventoryFlags: make([]int, 8),
	}
	before := snapshotNativeDeathRewardInventory(killer)
	continued := false
	g := &Game{pendingNativeDeathRewards: []pendingNativeDeathReward{{killer: killer, item: 0xd3}}}
	if !g.runPendingNativeDeathRewards(func() { continued = true }) {
		t.Fatal("缺來源的待處理獎勵未被失敗即關閉閘門擁有")
	}
	if g.loadErr == "" || continued || g.nativeDeathRewardUI != nil {
		t.Fatalf("缺來源未停住：err=%q continued=%v ui=%#v", g.loadErr, continued, g.nativeDeathRewardUI)
	}
	assertNativeDeathRewardInventory(t, killer, before)
}
