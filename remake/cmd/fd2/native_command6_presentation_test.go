package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/battlepresent"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

func nativeCommand6PresentationTestPlan(t *testing.T) (*battle.NativeCommandDamagePlan, *battle.Unit, *battle.Unit) {
	t.Helper()
	book := make([]battle.NativeCommandRecord, battle.NativeCommandRecordCount)
	for id := range book {
		book[id].ID = id
	}
	book[6] = battle.NativeCommandRecord{
		ID: 6, Damage: 90, Hit: 100, SelectionMode: 1,
		EffectMode: 0, MPCost: 2, TargetCode: 0,
	}
	actor := &battle.Unit{Camp: battle.Own, X: 0, Y: 0, HP: 20, MP: 5, OnField: true}
	target := &battle.Unit{Camp: battle.Enemy, ClassID: 5, X: 1, Y: 0, HP: 103, OnField: true}
	state := &battle.State{
		W: 2, H: 1, Units: []*battle.Unit{actor, target},
		NativeCompositionEventBytes: make([]byte, 2), NativeCommandBook: book,
	}
	schedule := figani.NativeCommand6PresentationSchedule{EffectResource: 33, BaseByte: 90, DwordTable: [5]int{-10, -8, -3, 0, 0}}
	walk := func(count int, resolve func(int, uint16) (uint16, bool, error)) (uint16, error) {
		return figani.WalkNativeCommand6RNG(3, schedule, 0, count, resolve)
	}
	plan, err := state.PlanNativeCommandDamageWalk(actor, target, 6, map[int]int{5: 10}, 3, walk)
	if err != nil {
		t.Fatal(err)
	}
	return plan, actor, target
}

func TestNativeCommand6PresentationPublishesOnlyAfterDraw(t *testing.T) {
	plan, actor, target := nativeCommand6PresentationTestPlan(t)
	callback := 0
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd6Presentation = &nativeCommand6PresentationJob{
		actor: actor, plan: plan, prelude: make([]*ebiten.Image, 1),
		actorBlack: make([]*ebiten.Image, 1), actorPulse: make([]*ebiten.Image, 1),
		actorSpecs:    []battlepresent.NativeCommand0ActorFrame{{PublishMP: true}},
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted,
		targetHPBefore: []int{target.HP},
		then:           func(results []battle.NativeCommandDamageResult) { callback = len(results) },
	}
	for stage := 1; stage <= plan.DamageStages; stage++ {
		g.nativeCmd6Presentation.handler = append(g.nativeCmd6Presentation.handler,
			nativeCommand6HandlerFrame{targetIndex: 0, hpStage: stage})
	}

	g.stepNativeCommand6Presentation()
	if actor.MP != plan.MPBefore || actor.Acted || target.HP != plan.Results[0].HPBefore {
		t.Fatal("未繪製的第 6 指令改變了戰鬥狀態")
	}
	for g.nativeCmd6Presentation != nil {
		g.nativeCmd6Presentation.drawn = true
		g.stepNativeCommand6Presentation()
	}
	if actor.MP != plan.MPAfter || !actor.Acted || target.HP != plan.Results[0].HPAfter ||
		g.nativeRNGState != plan.RNGAfter || callback != 1 {
		t.Fatalf("第 6 指令交易未完成：mp=%d acted=%v hp=%d rng=%#x callback=%d",
			actor.MP, actor.Acted, target.HP, g.nativeRNGState, callback)
	}
}

func TestNativeCommand6PresentationRollsBackRuntimeFailure(t *testing.T) {
	plan, actor, target := nativeCommand6PresentationTestPlan(t)
	g := &Game{nativeRNGState: plan.RNGBefore}
	g.nativeCmd6Presentation = &nativeCommand6PresentationJob{
		actor: actor, plan: plan, phase: nativeCommand6Actor,
		actorBlack: make([]*ebiten.Image, 1), actorPulse: make([]*ebiten.Image, 1),
		actorSpecs:    []battlepresent.NativeCommand0ActorFrame{{PublishMP: true}},
		handler:       []nativeCommand6HandlerFrame{{targetIndex: 0, hpStage: 2}},
		actorMPBefore: actor.MP, actorActedBefore: actor.Acted,
		targetHPBefore: []int{target.HP}, drawn: true,
	}
	g.stepNativeCommand6Presentation()
	if actor.MP != plan.MPAfter {
		t.Fatalf("MP 標記未發布：%d", actor.MP)
	}
	g.nativeCmd6Presentation.drawn = true
	g.stepNativeCommand6Presentation()
	if g.nativeCmd6Presentation != nil || actor.MP != plan.MPBefore || actor.Acted ||
		target.HP != plan.Results[0].HPBefore || g.nativeRNGState != plan.RNGBefore || g.loadErr == "" {
		t.Fatalf("失敗回復不完整：mp=%d acted=%v hp=%d rng=%#x err=%q",
			actor.MP, actor.Acted, target.HP, g.nativeRNGState, g.loadErr)
	}
}

// #157 的正常原版收據記錄了各目標 entry、命中、傷害與施法末端 RNG。
// side0 有正常原版樣本；side1 在相同控制條件下驗證共用規則。
func TestNativeCommand6RNGMatchesOriginalChapter24Targets(t *testing.T) {
	book, err := battle.LoadNativeCommandRecords("../../assets/spells.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, side := range []byte{0, 1} {
		frames := make([]figani.Frame, figani.NativeCommand6EffectFrameCount)
		for i := range frames {
			frames[i] = figani.Frame{Width: 1, Height: 1, Pixels: []byte{1}, Mask: []byte{1}}
		}
		schedule, err := figani.BuildNativeCommand6PresentationSchedule(side, &figani.Animation{HeaderByte2: figani.NativeCommand6EffectFrameCount, Frames: frames})
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			name           string
			seed, final    uint16
			before         []uint16
			resist, damage []int
			hit            []bool
		}{
			{"original-actor20", 14047, 32891, []uint16{14047, 57339}, []int{10, 7}, []int{212, 0}, []bool{true, false}},
			{"original-actor21", 32891, 61045, []uint16{32891}, []int{10}, []int{215}, []bool{true}},
			{"original-actor25", 2762, 59025, []uint16{2762}, []int{8}, []int{158}, []bool{true}},
			// 以下是明示的受控規則樣本，驗證後續命中的12個 marker 與先落空。
			{"controlled-hit-miss-hit", 14047, 62539, []uint16{14047, 57339, 32891}, []int{10, 8, 7}, []int{212, 0, 150}, []bool{true, false, true}},
			{"controlled-miss-hit-miss", 63768, 57895, []uint16{63768, 18788, 27696}, []int{10, 8, 7}, []int{0, 165, 0}, []bool{false, true, false}},
		} {
			t.Run(fmt.Sprintf("side%d/%s", side, tc.name), func(t *testing.T) {
				calls := 0
				final, err := figani.WalkNativeCommand6RNG(tc.seed, schedule, side, len(tc.before), func(index int, rng uint16) (uint16, bool, error) {
					if index != calls || rng != tc.before[index] {
						t.Fatalf("target=%d entry=%d want=%d calls=%d", index, rng, tc.before[index], calls)
					}
					calls++
					result, next, err := battle.ResolveNativeCommandDamage(book[6].Damage, book[6].Hit, tc.resist[index], rng)
					if result.Damage != tc.damage[index] || result.Hit != tc.hit[index] {
						t.Fatalf("target=%d result=%+v want damage=%d hit=%v", index, result, tc.damage[index], tc.hit[index])
					}
					return next, result.Hit, err
				})
				if err != nil || final != tc.final || calls != len(tc.before) {
					t.Fatalf("final=%d want=%d calls=%d err=%v", final, tc.final, calls, err)
				}
			})
		}
	}
}

// #160 的 UI 回歸使用已證座標與固定 RNG；side0 是受控演出資料，
// 非零側仍依 #154 拒收，不能把本測試稱為原版完整影格對拍。
func TestPlayerNativeCommand6CursorCentersThroughConfirm(t *testing.T) {
	for _, side := range []byte{0, 1} {
		for _, empty := range []bool{false, true} {
			t.Run(fmt.Sprintf("side%d/empty%v", side, empty), func(t *testing.T) {
				assets, field, state := completeNativeMapFrameFixture(t)
				pack := filepath.Clean("../../generated-assets/fd2-original-b97caf22")
				if !fileExists(filepath.Join(pack, "animations", "FDOTHER_032", "animation.json")) {
					t.Skip("separated command6 pack unavailable")
				}
				t.Setenv("FD2_ASSET_PACK", pack)
				t.Setenv("FD2_MUTE", "1")
				base := filepath.Clean("../../../org_game/炎龍騎士團/FLAME2")
				t.Setenv("FD2_ORIGINAL_FDTXT", filepath.Join(base, "FDTXT.DAT"))
				field.W, field.H = 30, 30
				field.Tiles = make([]int, 900)
				state.W, state.H = 30, 30
				state.NativeTileBlitModes = make([]byte, 900)
				state.NativeCompositionEventBytes = make([]byte, 900)
				actor := state.Units[0]
				actor.Camp, actor.OnField, actor.X, actor.Y = battle.Own, true, 20, 22
				actor.HP, actor.MaxHP, actor.MP, actor.MaxMP = 331, 331, 1101, 1101
				actor.NativeRecordByte6 = side
				actor.NativeRecordByte8 = 0
				actor.HasNativeRecordByte8 = true
				actor.InventorySlots, actor.NativeInventoryFlags = make([]int, 8), make([]int, 8)
				actor.NativeCommandMask = [5]byte{0x40}
				center := *actor
				center.X, center.Y, center.HP, center.MaxHP = 22, 20, 877, 877
				target := &battle.Unit{Camp: battle.Enemy, X: 22, Y: 18, HP: 83, MaxHP: 83, ClassID: 4, OnField: true,
					BattleFig: 0, HasBattleFig: true, HasNativeRecordByte5: true, NativeRecordByte6: 1, HasNativeRecordByte6: true,
					NativeRecordByte8: 1, HasNativeRecordByte8: true, NativeRecordRace: 1, HasNativeRecordRace: true,
					NativeRecordClass: 4, HasNativeRecordClass: true, InventorySlots: make([]int, 8), NativeInventoryFlags: make([]int, 8)}
				state.Units = []*battle.Unit{actor, &center, target}
				book, err := battle.LoadNativeCommandRecords("../../assets/spells.json")
				if err != nil {
					t.Fatal(err)
				}
				state.NativeCommandBook = book
				state.NativeCommandResistances = map[int]int{4: 4}
				scene, err := battle.LoadNativeCommandSceneTable("../../assets/data/native_command_scene.json")
				if err != nil {
					t.Fatal(err)
				}
				flash, err := battle.LoadNativeCommandPaletteFlashTable("../../assets/data/native_command_palette_flash.json")
				if err != nil {
					t.Fatal(err)
				}
				for len(assets.LUTs) <= 15 {
					lut := make([]byte, 256)
					for i := range lut {
						lut[i] = byte(i)
					}
					assets.LUTs = append(assets.LUTs, lut)
				}
				g := &Game{m: field, st: state, sel: actor, curX: 22, curY: 20, nativeCommand0Targeting: true, nativeCommandTargetID: 6,
					nativeMapAssets: assets, nativeUIPalette: loadNativeUIPalette(), nativeCommandScene: scene, nativeCommandPaletteFlash: flash, nativeRNGState: 3473}
				attachOfficialLocale(t, g)
				if empty {
					g.curX, g.curY = 21, 19
				}
				if candidates, err := g.nativeCommandTargetUnitsFor(6); err != nil || len(candidates) != 0 {
					t.Fatalf("initial candidates=%v err=%v", candidates, err)
				}
				if err := g.materializeNativeCommandTargetField(book[6]); err != nil {
					t.Fatal(err)
				}
				g.confirm()
				if actor.MP != 1101 || actor.Acted || target.HP != 83 || center.HP != 877 || g.nativeRNGState != 3473 {
					t.Fatal("confirm crossed prebuild boundary")
				}
				if side != 0 {
					if g.nativeCmd6Presentation != nil || !strings.Contains(g.msg, "work frame bounds (141,-1 143x111)") {
						t.Fatalf("expected #154 refusal after valid center, msg=%q", g.msg)
					}
					if !g.nativeCommand0Targeting || g.sel != actor {
						t.Fatal("prebuild failure lost target modal")
					}
					return
				}
				if g.nativeCmd6Presentation == nil {
					t.Fatalf("valid cursor did not start command6: %s", g.msg)
				}
				if len(g.nativeCmd6Presentation.plan.Results) != 1 || g.nativeCmd6Presentation.plan.Results[0].Target != target {
					t.Fatal("cursor occupant became damage target")
				}
				g.stepNativeCommand6Presentation()
				if actor.MP != 1101 || target.HP != 83 {
					t.Fatal("undrawn frame published state")
				}
				screen := ebiten.NewImage(640, 400)
				for steps := 0; g.nativeCmd6Presentation != nil && steps < 256; steps++ {
					if !g.drawNativeCommand6Presentation(screen) {
						t.Fatal("command6 draw failed")
					}
					g.stepNativeCommand6Presentation()
				}
				if g.nativeCmd6Presentation != nil || actor.MP != 1071 || !actor.Acted || target.HP != 4 || center.HP != 877 || g.nativeRNGState != 33552 {
					t.Fatalf("transaction MP=%d HP=%d center=%d RNG=%d acted=%v err=%s", actor.MP, target.HP, center.HP, g.nativeRNGState, actor.Acted, g.loadErr)
				}
				if g.nativeCommand0Targeting || g.sel != nil || state.NativeMapRangeMode != 1 {
					t.Fatal("command6 did not restore player modal")
				}
			})
		}
	}
}

// 與 #154 正常原版使用同一固定玩家槽，經正式 CONTINUE owner 載入。
// 游標／modal 使用公開玩家處理器作窄測試；施法入口 RNG 明示固定為3473。
// 這是局部資料流比對，沒有全程鍵盤或影像一致的宣稱。
func TestNativeCommand6OriginalContinueCursorProbe(t *testing.T) {
	savePath := os.Getenv("FD2_COMMAND6_CURSOR_SAVE")
	if savePath == "" {
		t.Skip("未提供#154固定玩家槽")
	}
	stored, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(stored)
	hash := hex.EncodeToString(sum[:])
	if hash != "f46d9c54d3037f84f05d72714569c282e63f39bf125251a9cf5cd9593ff3241f" {
		t.Fatalf("unreviewed source SHA256=%s", hash)
	}
	t.Setenv("FD2_NATIVE_SAVE", savePath)
	t.Setenv("FD2_NATIVE_TITLE_TICK", "")
	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_MUTE", "1")
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	g.camp, g.titlePhase, g.titleSel = campaign.NewRunner(graph), "menu", 0
	if !g.applyTitleMenuEvent(TitleMenuDown) || !g.applyTitleMenuEvent(TitleMenuDown) || !g.applyTitleMenuEvent(TitleMenuConfirm) {
		t.Fatal("CONTINUE owner rejected")
	}
	for tick := 0; tick < 24; tick++ {
		if !g.applyTitleMenuEvent(TitleMenuTick) {
			t.Fatal("CONTINUE tick rejected")
		}
	}
	if g.camp.NodeID() != "battle_ch30" || g.st == nil || len(g.st.Units) != 27 || g.st.NativeRoundCounter != 4 {
		t.Fatalf("wrong CONTINUE state node=%s", g.camp.NodeID())
	}
	actor, center, target := g.st.Units[6], g.st.Units[18], g.st.Units[24]
	if actor.X != 20 || actor.Y != 22 || actor.MP != 1101 || target.X != 22 || target.Y != 18 || target.HP != 83 || center.X != 22 || center.Y != 20 {
		t.Fatalf("source records differ actor=%+v target=%+v center=%+v", actor, target, center)
	}
	if !g.positionScreenshotCursor(20, 22) {
		t.Fatal("normal cursor could not reach caster")
	}
	g.confirm()
	if g.sel != actor {
		t.Fatal("normal selection did not choose record6")
	}
	g.confirm()
	r := &parityReplay{t: t, g: g}
	r.settleActionOverlay(parityAction{Kind: "stay"})
	if !g.ring || g.actionOverlayBlocksInput() {
		t.Fatal("stationary ring did not settle")
	}

	if !nativeActionSelectable(g.actionOverlayAvailability(), 1) {
		t.Fatal("stationary spell action unavailable")
	}
	// 此窄探針以既有 owner 交接 modal，沒有新增狀態注入、移動單位或改槽。
	if !closeRing(t, g, func() { g.nativeCommandOpen = true; g.nativeCommandSel = 0 }) {
		t.Fatal("spell UI handoff did not complete")
	}

	if candidates, err := g.nativeCommandTargetUnitsFor(6); err != nil || len(candidates) != 0 {
		t.Fatalf("selection list=%v err=%v", candidates, err)
	}
	cursor := battle.Cell{X: 22, Y: 20}
	g.nativeCommandOpen = false
	g.nativeCommand0Targeting = true
	g.nativeCommandTargetID = 6
	if err := g.materializeNativeCommandTargetField(g.st.NativeCommandBook[6]); err != nil {
		t.Fatal(err)
	}
	if !g.positionScreenshotCursor(cursor.X, cursor.Y) {
		t.Fatal("normal target cursor rejected center")
	}
	effect, err := figani.LoadSeparatedArchiveResource(separatedAssetPath("animations"), "FDOTHER.DAT", 32)
	if err != nil {
		t.Fatal(err)
	}
	schedule, err := figani.BuildNativeCommand6PresentationSchedule(actor.NativeRecordByte6, effect)
	if err != nil {
		t.Fatal(err)
	}
	g.nativeRNGState = 3473
	walk := func(n int, resolve func(int, uint16) (uint16, bool, error)) (uint16, error) {
		return figani.WalkNativeCommand6RNG(3473, schedule, actor.NativeRecordByte6, n, resolve)
	}
	plan, err := g.st.PlanNativeCommand6DamageAtCursor(actor, cursor, g.st.NativeCommandResistances, 3473, walk)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Results) != 1 || plan.Results[0].Target != target || plan.Results[0].HPAfter != 4 || plan.MPAfter != 1071 || plan.RNGAfter != 33552 {
		t.Fatalf("same-source planner differs: plan=%+v results=%+v", plan, plan.Results)
	}
	if prefix := os.Getenv("FD2_COMMAND6_PROTOTYPE_PREFIX"); prefix != "" {
		command6SignedArenaPrototype(t, g, actor, plan, effect, schedule, prefix)
	}
	g.confirm()
	if !strings.Contains(g.msg, "work frame bounds (141,-1 143x111)") || g.nativeCmd6Presentation != nil || actor.MP != 1101 || target.HP != 83 || actor.Acted || g.nativeRNGState != 3473 || !g.nativeCommand0Targeting {
		t.Fatalf("expected #154 atomic refusal, msg=%s", g.msg)
	}
	report := map[string]interface{}{"source_save_sha256": hash, "source_classification": "既有固定第三方玩家槽；同源CONTINUE局部資料流，不是全程正常鍵盤或章E2",
		"actor_record": 6, "center": cursor, "center_record": 18, "effect_targets": []int{24}, "mp_before": 1101, "planned_mp_after": plan.MPAfter,
		"hp_before": 83, "planned_hp_after": plan.Results[0].HPAfter, "controlled_rng_at_cast": 3473, "planned_rng_after": plan.RNGAfter,
		"rng_method":    "執行施法規劃前固定原版0x1C75E entry3473，隔離中心與作用名單，不宣稱選單跨段骰序一致",
		"render_result": "#154負列原子拒收", "error": g.msg, "state_unchanged": true, "original_runner": "dosgolem apps/fd2/cmd/oracle 951cb55f7834311fccbce208fde6bc48d57f1c5a"}
	if out := os.Getenv("FD2_COMMAND6_CURSOR_OUT"); out != "" {
		raw, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(out, append(raw, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// #154 可撤回原型。額外一列只保存已觀測的負位移寫入，不映射原版其他 heap。
// 正式組圖仍拒收負列；此探針不更新 MP、HP、RNG 或玩家演出 owner。
func command6SignedArenaPrototype(t *testing.T, g *Game, actor *battle.Unit, plan *battle.NativeCommandDamagePlan, effect *figani.Animation, schedule figani.NativeCommand6PresentationSchedule, prefix string) {
	t.Helper()
	if len(plan.Results) != 1 {
		t.Fatal("prototype needs the fixed single target")
	}
	target := plan.Results[0].Target
	initial, err := g.nativeCommandScene.InitialBackground(g.handlerChapter)
	if err != nil {
		t.Fatal(err)
	}
	actorControl, err := nativeCommand0Control(g.m, actor)
	if err != nil {
		t.Fatal(err)
	}
	targetControl, err := nativeCommand0Control(g.m, target)
	if err != nil {
		t.Fatal(err)
	}
	actorGate, err := battle.NativeCommandBackgroundGate(actor)
	if err != nil {
		t.Fatal(err)
	}
	targetGate, err := battle.NativeCommandBackgroundGate(target)
	if err != nil {
		t.Fatal(err)
	}
	actorSelector := initial
	if !actorGate || actorSelector == 0 {
		actorSelector = actorControl[2]
	}
	targetSelector := fdicon.NativeCommandBackgroundSelector(initial, []fdicon.NativeCommandBackgroundTarget{{Gate: targetGate, Control: targetControl}})
	bgSelector, taiSelector := targetSelector, actorSelector
	if actor.NativeRecordByte6 == 0 {
		bgSelector, taiSelector = actorSelector, targetSelector
	}
	background, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "BG.DAT", int(bgSelector))
	if err != nil {
		t.Fatal(err)
	}
	platform, err := fdother.LoadSeparatedSingleFrame(separatedAssetPath("surfaces"), "TAI.DAT", int(taiSelector))
	if err != nil {
		t.Fatal(err)
	}
	panels, err := battle.LoadNativeItemPanelDataAssets(separatedAssetPath(""))
	if err != nil {
		t.Fatal(err)
	}
	actorIndex, err := nativeCommand24RuntimeUnitIndex(g.st, actor)
	if err != nil {
		t.Fatal(err)
	}
	after := *actor
	after.MP = plan.MPAfter
	actorRecord, err := battle.NativeBattlePanelRecordForUnit(&after)
	if err != nil {
		t.Fatal(err)
	}

	stageBases, err := g.nativeCommand6StageBases(background, platform, panels, actorRecord, actorIndex, plan)
	if err != nil {
		t.Fatal(err)
	}
	bases := stageBases[0]
	actorEffect, err := nativeCommand6ActorEffect(separatedAssetPath("animations"), actor.BattleFig)
	if err != nil {
		t.Fatal(err)
	}
	idle, err := figani.LoadSeparatedResource(separatedAssetPath("animations"), target.BattleFig*3)
	if err != nil {
		t.Fatal(err)
	}
	points := figani.NativeCommand6TargetCoordinates(figani.NativeCommand6Coordinates(36, schedule.BaseByte), 42)
	planned, err := figani.BuildNativeCommand6TargetSequence(schedule, points, actor.NativeRecordByte6)
	if err != nil {
		t.Fatal(err)
	}
	reports := make([]map[string]interface{}, 0, len(planned))
	hpStage, idleFrame, idleRepeat := 0, 0, 0
	shade, pose, jitter := 8, 3, -1
	_, numericRNG, err := battle.ResolveNativeCommandDamage(g.st.NativeCommandBook[6].Damage, g.st.NativeCommandBook[6].Hit, g.st.NativeCommandResistances[int(target.NativeRecordClass)], 3473)
	if err != nil {
		t.Fatal(err)
	}
	xPose, yPose := [4]int{6, 4, 2, 0}, [4]int{-3, -2, -1, 0}
	displays, err := figani.BuildNativeCommand6TargetDisplayFrames(figani.NewNativeCommand6DisplayState(), planned, true, numericRNG)
	if err != nil {
		t.Fatal(err)
	}
	formalCheck := os.Getenv("FD2_COMMAND6_FORMAL_TARGET_CHECK") == "1"

	for index, frame := range planned {
		// guard 是原型專用的獨立前置列，不是原版0x2A300配置的一部分。
		const guard, stride, height, viewport = 640, 640, 270, 19360
		arena := make([]byte, guard+stride*height)
		for i := 0; i < guard; i++ {
			arena[i] = 0xa5
		}
		for y := 0; y < 200; y++ {
			at := guard + viewport + y*stride
			copy(arena[at:at+320], bases[hpStage][y*320:(y+1)*320])
		}
		spillCount, firstSpill := 0, 0
		blit := func(sprite figani.Frame, dx, dy int) {
			x0, y0 := 160+sprite.X+dx, 30+sprite.Y+dy
			if sprite.Width <= 0 || sprite.Height <= 0 || len(sprite.Pixels) != sprite.Width*sprite.Height || len(sprite.Mask) != len(sprite.Pixels) ||
				x0 < 0 || x0+sprite.Width > stride || y0 < -1 || y0+sprite.Height > height {
				t.Fatalf("prototype bounds index=%d (%d,%d %dx%d)", index, x0, y0, sprite.Width, sprite.Height)
			}
			for y := 0; y < sprite.Height; y++ {
				for x := 0; x < sprite.Width; x++ {
					src := y*sprite.Width + x
					if sprite.Mask[src] == 0 {
						continue
					}
					logical := (y0+y)*stride + x0 + x
					if logical < 0 {
						if spillCount == 0 {
							firstSpill = logical
						}
						spillCount++
					}
					arena[guard+logical] = sprite.Pixels[src]
				}
			}
		}
		for _, layer := range frame.Mode4 {
			blit(effect.Frames[layer.Frame], layer.X, layer.Y)
		}
		blit(actorEffect.Frames[len(actorEffect.Frames)-1], 0, 0)
		shaded := idle.Frames[idleFrame]
		shaded.Pixels = append([]byte(nil), shaded.Pixels...)
		for i, c := range shaded.Pixels {
			shaded.Pixels[i] = byte((int(c)+shade)&7) + 0xb0
		}
		blit(shaded, xPose[pose]*jitter, yPose[pose])
		shade--
		if shade == 1 {
			shade = 8
		}
		if pose != 3 {
			pose++
		}
		for _, layer := range frame.Mode5 {
			blit(effect.Frames[layer.Frame], layer.X, layer.Y)
		}
		pixels := make([]byte, 320*200)
		for y := 0; y < 200; y++ {
			at := guard + viewport + y*stride
			copy(pixels[y*320:(y+1)*320], arena[at:at+320])
		}
		file := fmt.Sprintf("%s-frame-%02d.idx", prefix, index)
		if err := os.WriteFile(file, pixels, 0644); err != nil {
			t.Fatal(err)
		}

		var formal []byte
		var formalErr error
		if formalCheck {
			formal, formalErr = battlepresent.ComposeNativeCommand6TargetFrame(bases[hpStage], actorEffect.Frames[len(actorEffect.Frames)-1], idle.Frames[idleFrame], effect, frame, displays[index])
			if index == 7 {
				if formalErr == nil || formal != nil || !strings.Contains(formalErr.Error(), "work frame bounds (141,-1 143x111)") {
					t.Fatalf("formal guard changed frame%d err=%v", index, formalErr)
				}
			} else {
				if formalErr != nil {
					t.Fatal(formalErr)
				}
				if !bytes.Equal(formal, pixels) {
					t.Fatalf("formal caller differs from original-matched prototype frame%d", index)
				}
				if err := os.WriteFile(fmt.Sprintf("%s-formal-frame-%02d.idx", prefix, index), formal, 0644); err != nil {
					t.Fatal(err)
				}
			}
		}
		sum := sha256.Sum256(pixels)
		entry := map[string]interface{}{"index": index, "path": file, "indexed_sha256": hex.EncodeToString(sum[:]),
			"hp_stage": hpStage, "idle_frame": idleFrame, "negative_opaque_writes": spillCount, "first_negative_offset": firstSpill}

		if formalCheck {
			if formalErr != nil {
				entry["formal_result"] = "rejected #154"
				entry["formal_error"] = formalErr.Error()
			} else {
				entry["formal_result"] = "full-frame matched"
				entry["formal_indexed_differences"] = 0
				entry["formal_rgb_differences"] = 0
			}
			entry["display"] = displays[index]
		}
		pic := image.NewPaletted(image.Rect(0, 0, 320, 200), g.nativeUIPalette)
		copy(pic.Pix, pixels)
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, pic); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fmt.Sprintf("%s-frame-%02d.png", prefix, index), encoded.Bytes(), 0644); err != nil {
			t.Fatal(err)
		}
		if originalDir := os.Getenv("FD2_COMMAND6_PROTOTYPE_ORIGINAL"); originalDir != "" {
			originalPath := filepath.Join(originalDir, fmt.Sprintf("frame-%06d.png", index))
			originalRaw, err := os.ReadFile(originalPath)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := png.Decode(bytes.NewReader(originalRaw))
			if err != nil {
				t.Fatal(err)
			}
			original, ok := decoded.(*image.Paletted)
			if !ok || original.Bounds() != pic.Bounds() {
				t.Fatalf("original indexed shape invalid: %s", originalPath)
			}
			indexedDifferences, rgbDifferences := 0, 0
			for y := 0; y < 200; y++ {
				for x := 0; x < 320; x++ {
					if pic.ColorIndexAt(x, y) != original.ColorIndexAt(x, y) {
						indexedDifferences++
					}
					ar, ag, ab, aa := pic.At(x, y).RGBA()
					br, bg, bb, ba := original.At(x, y).RGBA()
					if ar != br || ag != bg || ab != bb || aa != ba {
						rgbDifferences++
					}
				}
			}
			originalSum := sha256.Sum256(originalRaw)
			entry["original_png"] = originalPath
			entry["original_png_sha256"] = hex.EncodeToString(originalSum[:])
			entry["indexed_differences"] = indexedDifferences
			entry["rgb_differences"] = rgbDifferences
			if indexedDifferences != 0 || rgbDifferences != 0 {
				t.Errorf("full frame%d indexed=%d RGB=%d; no masks", index, indexedDifferences, rgbDifferences)
			}
		}
		reports = append(reports, entry)
		if frame.HPStage != 0 {
			hpStage = frame.HPStage
		}
		if frame.NumericMarker {
			pose = 0
			numericRNG = fdother.NativeRNGStep(numericRNG)
			jitter = 1 - int(numericRNG%3)
		}
		idleRepeat++
		if idle.Frames[idleFrame].Delay <= 0 {
			t.Fatal("prototype idle delay missing")
		}
		if idleRepeat >= idle.Frames[idleFrame].Delay {
			idleRepeat = 0
			idleFrame = (idleFrame + 1) % len(idle.Frames)
		}
	}
	palette := make([]byte, 0, 256*3)
	for _, c := range g.nativeUIPalette {
		r, gg, b, _ := c.RGBA()
		palette = append(palette, byte(r>>8), byte(gg>>8), byte(b>>8))
	}
	if err := os.WriteFile(prefix+"-palette.rgb", palette, 0644); err != nil {
		t.Fatal(err)
	}
	report := map[string]interface{}{"status": "DRAFT test-only signed arena and #161 caller composition; not production or heap parity", "work_allocation_bytes": 640 * 270,
		"prototype_guard_bytes": 640, "background_selector": bgSelector, "platform_selector": taiSelector,
		"raw_side": actor.NativeRecordByte6, "actor_battle_fig": actor.BattleFig, "target_battle_fig": target.BattleFig, "frames": reports,
		"formal_caller_checked": formalCheck, "strict_frame7_rejected": formalCheck,
		"comparison_scope": "12 prototype frames / 11 formal accepted frames, frame7 strict rejected; not complete formal cast"}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prefix+".json", append(raw, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
