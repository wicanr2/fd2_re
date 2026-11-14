package main

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

// 玩家用傷害指令 0 打中高階敵人：0x1C81F 累加經驗，0x1196D 發給行動者並排入
// 經驗／升級對話。設定沿用 TestPlayerNativeCommand0RunsIndexedPresentationThroughCursorConfirm，
// 走正式 confirm 與 indexed 演出直到結果回呼。
func TestPlayerNativeCommand0AwardsCommandExperience(t *testing.T) {
	assets, field, state := completeNativeMapFrameFixture(t)
	base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
	pack := filepath.Clean("../../generated-assets/fd2-original-b97caf22")
	if !fileExists(filepath.Join(pack, "surfaces", "BG_000", "resource.json")) {
		t.Skip("separated command0 asset pack unavailable")
	}
	t.Setenv("FD2_ASSET_PACK", pack)
	for _, name := range []string{"FDOTHER.DAT", "FDTXT.DAT"} {
		if !fileExists(filepath.Join(base, name)) {
			t.Skip("player-provided command0 archives unavailable")
		}
	}
	t.Setenv("FD2_ORIGINAL_FIGANI", filepath.Join(t.TempDir(), "missing-FIGANI.DAT"))
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	t.Setenv("FD2_ORIGINAL_BG", filepath.Join(t.TempDir(), "missing-BG.DAT"))
	t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
	t.Setenv("FD2_ORIGINAL_TAI", filepath.Join(t.TempDir(), "missing-TAI.DAT"))
	t.Setenv("FD2_MUTE", "1")
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id] = battle.NativeCommandRecord{ID: id}
	}
	book[0] = battle.NativeCommandRecord{
		ID: 0, Damage: 50, Hit: 100, SelectionMode: 5, EffectMode: 0,
		MPCost: 2, TargetCode: 0,
	}
	actor := state.Units[0]
	actor.Camp, actor.OnField, actor.HP, actor.MaxHP, actor.MP = battle.Own, true, 28, 28, 8
	actor.MaxMP, actor.NativeRecordByte8, actor.HasNativeRecordByte8 = 8, 0, true
	actor.InventorySlots, actor.NativeInventoryFlags = make([]int, 8), make([]int, 8)
	actor.Lv, actor.Exp = 5, 0
	actor.NativeRecordByte5, actor.HasNativeRecordByte5 = 0, true
	actor.NativeRecordByte6, actor.HasNativeRecordByte6 = 2, true
	actor.BattleFig, actor.HasBattleFig = 4, true
	row := make([]byte, 10)
	row[9] = 12
	target := &battle.Unit{
		Camp: battle.Enemy, OnField: true, X: 1, Y: 0, Lv: 5,
		HP: 100, MaxHP: 100, ClassID: 5,
		NativeRecordByte5: 0, HasNativeRecordByte5: true,
		NativeRecordByte6: 1, HasNativeRecordByte6: true,
		BattleFig: 0x44, HasBattleFig: true,
		NativeConstructor: &battle.NativeConstructorTable{Branch: "high_class", Index: 0, Record: row},
		NativeRecordByte8: 1, HasNativeRecordByte8: true,
		NativeRecordRace: 1, HasNativeRecordRace: true,
		NativeRecordClass: 1, HasNativeRecordClass: true,
		InventorySlots: make([]int, 8), NativeInventoryFlags: make([]int, 8),
	}
	state.Units = append(state.Units, target)
	state.NativeCommandBook = book
	state.NativeCommandResistances = map[int]int{1: 10}
	state.NativeCompositionEventBytes = make([]byte, state.W*state.H)
	state.NativeTileBlitModes[target.Y*state.W+target.X] = 0
	state.MaterializeNativeMapRangeMode(3)
	scene, err := battle.LoadNativeCommandSceneTable("../../assets/data/native_command_scene.json")
	if err != nil {
		t.Fatal(err)
	}
	flash, err := battle.LoadNativeCommandPaletteFlashTable("../../assets/data/native_command_palette_flash.json")
	if err != nil {
		t.Fatal(err)
	}
	for len(assets.LUTs) <= 14 {
		lut := make([]byte, 256)
		for i := range lut {
			lut[i] = byte(i)
		}
		assets.LUTs = append(assets.LUTs, lut)
	}
	g := &Game{
		m: field, st: state, sel: actor, curX: target.X, curY: target.Y,
		nativeCommand0Targeting: true, nativeCommandTargetID: 0,
		nativeRNGState: 1, nativeMapAssets: assets, nativeUIPalette: loadNativeUIPalette(),
		nativeCommandScene: scene, nativeCommandPaletteFlash: flash,
	}
	attachOfficialLocale(t, g)
	// 0x118EE：選到單位時清零；這裡模擬上一個行動留下的殘值。
	state.NativeExperienceAccumulator = 0

	g.confirm()
	if g.nativeCmd0Presentation == nil {
		t.Fatalf("command0 presentation did not start: %q", g.msg)
	}
	screen := ebiten.NewImage(640, 400)
	for steps := 0; g.nativeCmd0Presentation != nil && steps < 512; steps++ {
		if !g.drawNativeCommand0Presentation(screen) {
			t.Fatalf("command0 draw failed at step %d", steps)
		}
		g.stepNativeCommand0Presentation()
	}
	if g.nativeCmd0Presentation != nil || !actor.Acted || target.HP >= 100 || target.HP == 0 {
		t.Fatalf("command0 result actor=%#v target HP=%d msg=%q", actor, target.HP, g.msg)
	}
	// 存活：列表[9] 12 × 守方等級 5 × 擲出傷害 / 最大 HP 100。基數 50 的擲出值不會
	// 超過剩餘 HP，所以擲出值等於實際扣血。
	rolled := 100 - target.HP
	want := 12 * 5 * rolled / 100
	if want == 0 || actor.Exp != float64(want) {
		t.Fatalf("command0 EXP=%v want %d (rolled %d) loadErr=%q", actor.Exp, want, rolled, g.loadErr)
	}
	if state.NativeExperienceAccumulator != 0 {
		t.Fatalf("accumulator=%d after accepted award, want 0x1E51A reset", state.NativeExperienceAccumulator)
	}
	// 夾具沒有整備介面的 FDTXT_000 字串：finishSuccessfulUnitAction 的
	// runPendingNativeLevelUpDialogues 取出這筆經驗後，開框以固定訊息拒收。
	// 這個訊息只會在佇列裡有經驗時出現，證明經驗已交給 0x1E292 的對話擁有者。
	const dialogueOwnerRefusal = "native level-up dialogue: FDTXT_000 strings are unavailable"
	if g.nativeLevelUpDialogue == nil && len(g.pendingNativeLevelUps) == 0 && g.loadErr != dialogueOwnerRefusal {
		t.Fatalf("command0 EXP was not handed to the 0x1E292 dialogue owner (loadErr=%q)", g.loadErr)
	}
}
