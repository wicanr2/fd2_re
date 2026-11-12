package main

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// battleSnapshotTestDir 讓 F5／F9 寫入暫存目錄，不碰玩家的資料夾。
func battleSnapshotTestDir(t *testing.T) {
	t.Helper()
	old := userDataDirCached
	userDataDirCached = t.TempDir()
	t.Cleanup(func() { userDataDirCached = old })
}

// loadChapterOneStoryBattle 沿正常劇情快進到第一關，敵方 AI 真的執行；直接跳節點
// 的戰場缺少原版紀錄出處，AI 會拒絕。
func loadChapterOneStoryBattle(t *testing.T) *Game {
	t.Helper()
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", defaultPlayerCampaign)
	t.Setenv("FD2_SHOT_AI", "1")
	t.Setenv("FD2_SEED", "20260911")
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	if err := g.fastForwardShotCampaign(); err != nil {
		t.Fatal(err)
	}
	if !pump(t, g, ch01FrameBudget*4, func() bool {
		if len(g.dialog) > 0 && g.battleEvent == nil {
			g.dlgAdvance()
		}
		return deathBattleIdle(g) && g.battleSnapshotBlocker() == ""
	}) {
		t.Fatalf("第一關沒有到達可操作狀態\n阻塞：%s（%s）", ch01Blockers(g), g.battleSnapshotBlocker())
	}
	return g
}

// playOneOwnUnit 以正式游標／確認流程讓第一名待命我方行動，推回可操作狀態。
func playOneOwnUnit(t *testing.T, g *Game) {
	t.Helper()
	pending := pendingOwn(g)
	if len(pending) < 2 {
		t.Fatalf("待命我方只剩 %d 名", len(pending))
	}
	actor := pending[0]
	playUnit(t, g, actor, &ch01Recorder{})
	if !pump(t, g, ch01FrameBudget*4, func() bool {
		return g.result != "" || (deathBattleIdle(g) && g.battleSnapshotBlocker() == "")
	}) {
		t.Fatalf("單位行動後沒有回到可操作\n阻塞：%s（%s）", ch01Blockers(g), g.battleSnapshotBlocker())
	}
	if !actor.Acted {
		t.Fatal("行動的單位沒有標記已行動")
	}
}

// advanceToNextOwnTurn 走正式 END、敵方回合，推到我方再次可操作。
func advanceToNextOwnTurn(t *testing.T, g *Game) {
	t.Helper()
	turn := g.st.Turn
	// 正式 END 回復以當下的原生地圖影格為來源；離屏測試要先畫一次。
	if ch23JourneyScreen == nil {
		ch23JourneyScreen = ebiten.NewImage(640, 400)
	}
	g.Draw(ch23JourneyScreen)
	g.endTurn()
	if g.loadErr != "" {
		t.Fatalf("END：%s", g.loadErr)
	}
	if !pump(t, g, ch01FrameBudget*4, func() bool {
		return g.result != "" || (g.st.Turn != turn && deathBattleIdle(g) && g.battleSnapshotBlocker() == "")
	}) {
		t.Fatalf("回合 %d 沒有回到我方可操作\n阻塞：%s", turn, ch01Blockers(g))
	}
	if g.result != "" {
		t.Fatalf("回合 %d 結束時判定勝負：%q", turn, g.result)
	}
}

// stateFieldDiffs 以反射比對 battle.State 每個欄位；函式欄位不可比較而略過，
// 錯誤只比字串。回傳不同的欄位名稱。
func stateFieldDiffs(a, b *battle.State) []string {
	va, vb := reflect.ValueOf(a).Elem(), reflect.ValueOf(b).Elem()
	typ := va.Type()
	var diffs []string
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		x, y := va.Field(i), vb.Field(i)
		switch {
		case f.Type.Kind() == reflect.Func:
			continue
		case f.Name == "NativeMapSelectorError":
			if (a.NativeMapSelectorError == nil) != (b.NativeMapSelectorError == nil) ||
				a.NativeMapSelectorError != nil && a.NativeMapSelectorError.Error() != b.NativeMapSelectorError.Error() {
				diffs = append(diffs, f.Name)
			}
			continue
		}
		if !reflect.DeepEqual(reflect.NewAt(f.Type, x.Addr().UnsafePointer()).Elem().Interface(),
			reflect.NewAt(f.Type, y.Addr().UnsafePointer()).Elem().Interface()) {
			diffs = append(diffs, f.Name)
		}
	}
	return diffs
}

func scenarioFired(sc *battle.Scenario) string {
	raw, _ := json.Marshal(sc.FiredEvents())
	return string(raw)
}

func TestBattleSnapshotQuickSaveRestoresSameTurn(t *testing.T) {
	battleSnapshotTestDir(t)
	played := loadChapterOneStoryBattle(t)
	if reason := played.battleSnapshotBlocker(); reason != "" {
		t.Fatalf("第一關開局待機不能存快照：%s\n阻塞：%s", reason, ch01Blockers(played))
	}
	playOneOwnUnit(t, played)
	advanceToNextOwnTurn(t, played)
	if played.st.Turn < 2 {
		t.Fatalf("turn = %d", played.st.Turn)
	}
	// 回合中段存檔：一名我方已行動，其餘待命。這是玩家按 F5 最常見的時刻。
	playOneOwnUnit(t, played)
	if err := played.saveGameToSlot(0); err != nil {
		t.Fatalf("F5：%v（%s）", err, played.msg)
	}
	raw, err := os.ReadFile(saveSlotPath(0))
	if err != nil {
		t.Fatal(err)
	}
	var stored saveData
	if err := json.Unmarshal(raw, &stored); err != nil || stored.BattleSnapshot == nil {
		t.Fatalf("存檔沒有戰場快照：%v", err)
	}

	restored := loadChapterOneStoryBattle(t)
	restored.loadGameFromSlot(0)
	if restored.loadErr != "" {
		t.Fatal(restored.loadErr)
	}
	if strings.Contains(restored.msg, "無法還原") || restored.st == nil {
		t.Fatalf("快照沒有還原：%q", restored.msg)
	}
	if diffs := stateFieldDiffs(played.st, restored.st); len(diffs) != 0 {
		t.Fatalf("讀檔後 battle.State 不同：%v", diffs)
	}
	if scenarioFired(played.sc) != scenarioFired(restored.sc) {
		t.Fatalf("劇本已觸發事件不同：%s vs %s", scenarioFired(played.sc), scenarioFired(restored.sc))
	}
	if played.nativeRNGState != restored.nativeRNGState || played.curX != restored.curX || played.curY != restored.curY {
		t.Fatalf("RNG／游標不同：%d,%d,%d vs %d,%d,%d", played.nativeRNGState, played.curX, played.curY,
			restored.nativeRNGState, restored.curX, restored.curY)
	}
	if played.gold != restored.gold || !reflect.DeepEqual(played.items, restored.items) ||
		!reflect.DeepEqual(played.partyRoster, restored.partyRoster) {
		t.Fatal("金幣、道具或持續隊伍不同")
	}
	// Game 層只允許下列已核對的殘值或呈現暫態不同：城鎮／商店選單索引在讀檔時重設；
	// ringSel、selOrigX/Y、nativeMovePanelX 在下次開環、選取、移動時重寫；DAC／VGA／
	// work 由下一次 Draw 重組；調色盤相位、frame 由主機時間與程序生命週期驅動。
	allowed := map[string]bool{"banner": true, "frame": true, "msg": true,
		"churchClassID": true, "churchReviveID": true, "churchStatusID": true,
		"churchTransferDest": true, "churchTransferItem": true, "churchTransferSource": true,
		"nativeShopTransferDest": true, "nativeShopTransferItem": true, "nativeShopTransferSource": true,
		"nativeChurchUIPulse": true, "nativeShopUIPulse": true, "nativeFDOTHERPalettePhase": true,
		"nativeFDOTHERPaletteTick": true,
		"nativeMapDAC": true, "nativeMapVGA": true, "nativeMapWork": true, "nativeMovePanelX": true,
		"nativePlayerFocusStarted": true, "ringSel": true, "selOrigX": true, "selOrigY": true}
	for _, name := range gameFieldDiffs(played, restored) {
		if !allowed[name] {
			t.Errorf("讀檔後 Game.%s 與存檔前不同，需判斷是否要存進快照", name)
		}
	}

	// 續玩決定性：兩局各自再過一個敵方回合，戰場與 RNG 必須相同。
	advanceToNextOwnTurn(t, played)
	advanceToNextOwnTurn(t, restored)
	// 四個地圖動畫計時由主機時間驅動（advanceNativeMapClock），兩局在不同的實際
	// 時間執行，續玩後本來就不同；它們只決定待機動畫與地形相位，不影響戰況。
	// 讀檔當下的逐欄相等已在上面驗過。
	hostTimed := map[string]bool{"NativeMapCycleState": true, "NativeTerrainPhaseState": true,
		"NativeTerrainFlipState": true, "NativeUnitPixelShiftState": true}
	var gameplay []string
	for _, name := range stateFieldDiffs(played.st, restored.st) {
		if !hostTimed[name] {
			gameplay = append(gameplay, name)
		}
	}
	if len(gameplay) != 0 {
		t.Fatalf("續玩一回合後 battle.State 分歧：%v", gameplay)
	}
	if played.nativeRNGState != restored.nativeRNGState {
		t.Fatalf("續玩一回合後 RNG 分歧：%d vs %d", played.nativeRNGState, restored.nativeRNGState)
	}
}

func TestBattleSnapshotRejectsBusyBattle(t *testing.T) {
	battleSnapshotTestDir(t)
	g := loadChapterOneDeathBattle(t)
	g.aiBusy = true
	if err := g.saveGameToSlot(0); err == nil {
		t.Fatal("敵方回合中 F5 應拒絕")
	}
	if g.msg == "" {
		t.Fatal("拒絕時沒有玩家可見訊息")
	}
	if _, err := os.Stat(saveSlotPath(0)); !os.IsNotExist(err) {
		t.Fatalf("拒絕時仍寫了存檔：%v", err)
	}
}

func TestBattleSnapshotRejectsMismatchedNode(t *testing.T) {
	battleSnapshotTestDir(t)
	g := loadChapterOneDeathBattle(t)
	if err := g.saveGameToSlot(0); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(saveSlotPath(0))
	if err != nil {
		t.Fatal(err)
	}
	var d saveData
	if err := json.Unmarshal(raw, &d); err != nil || d.BattleSnapshot == nil {
		t.Fatal("沒有快照")
	}
	d.BattleSnapshot.Units = "assets/maps/other_units.json"
	raw, _ = json.Marshal(d)
	if err := os.WriteFile(saveSlotPath(0), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	before := g.camp.Cur
	if g.loadJSONGameFromSlot(0, false) {
		t.Fatal("資產不符的快照應拒絕")
	}
	if g.camp.Cur != before || g.msg == "" {
		t.Fatalf("拒絕後狀態被改或沒有訊息：node=%q msg=%q", g.camp.Cur, g.msg)
	}
}

// gameFieldDiffs 列出兩局 Game 不同的純資料欄位，用來找出還沒存的戰鬥狀態。
// 圖片、音訊、函式與介面欄位不比。
func gameFieldDiffs(a, b *Game) []string {
	va, vb := reflect.ValueOf(a).Elem(), reflect.ValueOf(b).Elem()
	typ := va.Type()
	var diffs []string
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if !plainDataType(f.Type, 0) {
			continue
		}
		x := reflect.NewAt(f.Type, va.Field(i).Addr().UnsafePointer()).Elem().Interface()
		y := reflect.NewAt(f.Type, vb.Field(i).Addr().UnsafePointer()).Elem().Interface()
		if !reflect.DeepEqual(x, y) {
			diffs = append(diffs, f.Name)
		}
	}
	sort.Strings(diffs)
	return diffs
}

func plainDataType(t reflect.Type, depth int) bool {
	if depth > 3 {
		return false
	}
	switch t.Kind() {
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.String:
		return true
	case reflect.Slice, reflect.Array:
		return plainDataType(t.Elem(), depth+1)
	case reflect.Map:
		return plainDataType(t.Key(), depth+1) && plainDataType(t.Elem(), depth+1)
	case reflect.Struct:
		if strings.HasPrefix(t.PkgPath(), "github.com/hajimehoshi") {
			return false
		}
		for i := 0; i < t.NumField(); i++ {
			if !plainDataType(t.Field(i).Type, depth+1) {
				return false
			}
		}
		return true
	}
	return false
}

func TestBattleSystemMenuSaveFallsBackToSnapshot(t *testing.T) {
	battleSnapshotTestDir(t)
	t.Setenv("FD2_NATIVE_SAVE", "")
	g := loadChapterOneStoryBattle(t)
	turn := g.st.Turn
	// 游標空格開啟的系統選單 → 巢狀選單 selector1（存檔）。
	g.nativeSystemCursorOverlay, g.nativeSystemNestedOpen, g.ring, g.ringSel = true, true, true, 1
	if !g.activateNativeSystemDirectionOne() {
		t.Fatal("系統選單存檔入口沒有受理")
	}
	raw, err := os.ReadFile(saveSlotPath(0))
	if err != nil {
		t.Fatalf("系統選單存檔沒有寫檔（%s）：%v", g.msg, err)
	}
	var d saveData
	if err := json.Unmarshal(raw, &d); err != nil || d.BattleSnapshot == nil {
		t.Fatal("系統選單存檔沒有戰場快照")
	}
	if !strings.Contains(g.msg, "已存檔") {
		t.Fatalf("沒有成功訊息：%q", g.msg)
	}
	restored := loadChapterOneStoryBattle(t)
	restored.loadGameFromSlot(0)
	if restored.st == nil || restored.st.Turn != turn || restored.ring || restored.nativeSystemCursorOverlay {
		t.Fatalf("讀回後狀態不對：turn=%v ring=%v overlay=%v msg=%q", restored.st != nil && restored.st.Turn == turn,
			restored.ring, restored.nativeSystemCursorOverlay, restored.msg)
	}
	if diffs := stateFieldDiffs(g.st, restored.st); len(diffs) != 0 {
		t.Fatalf("讀檔後 battle.State 不同：%v", diffs)
	}
}
