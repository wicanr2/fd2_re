package main

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/afm"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func recoveredCh20SkyKeySpec(t *testing.T) campaign.NativeCh20SkyKeySequence {
	t.Helper()
	beats, issues := campaign.CompileHandlerScript(&campaign.HandlerScript{Beats: []campaign.HandlerBeat{{
		Op: "native_call", NativeTarget: "0x24336",
		NativeSemantic: "ch20 天空之鑰固定合成演出序列", NativeConfidence: "已證實",
		NativeEvidence: []string{"docs/data/ida/fd2_ch20_sky_key_sequence_ida.txt"},
		Source:         campaign.HandlerSource{Addr: "0x242c9", Target: "0x24336"},
	}}}, campaign.HandlerBindings{})
	if len(issues) != 0 || len(beats) != 1 || beats[0].NativeCh20SkyKey == nil {
		t.Fatalf("compile recovered 0x24336 spec beats=%#v issues=%#v", beats, issues)
	}
	return *beats[0].NativeCh20SkyKey
}

func TestNativeCh20SkyKeyPlayerAssetsMatchRecoveredContract(t *testing.T) {
	fdPath, aniPath := nativeFDOTHERPath(), nativeANIPath()
	if fdPath == "" || aniPath == "" {
		t.Skip("player-provided FDOTHER.DAT or ANI.DAT is absent")
	}
	if _, err := os.Stat(fdPath); err != nil {
		t.Skipf("FDOTHER.DAT unavailable: %v", err)
	}
	if _, err := os.Stat(aniPath); err != nil {
		t.Skipf("ANI.DAT unavailable: %v", err)
	}
	spec := recoveredCh20SkyKeySpec(t)
	frames, err := fdother.DecodeResource(fdPath, spec.FDOTHERResource)
	if err != nil {
		t.Fatal(err)
	}
	clip, err := afm.DecodeResource(aniPath, spec.ANIResource)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateNativeCh20SkyKeyAssets(spec, frames, clip); err != nil {
		t.Fatal(err)
	}
}

func TestNativeCh20SkyKeyAssetValidationRejectsIncompleteANI(t *testing.T) {
	spec := recoveredCh20SkyKeySpec(t)
	frames := make([]fdother.Frame, spec.FDOTHERFrameCount)
	for index := range frames {
		frames[index] = fdother.Frame{Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, 1}}
	}
	if err := validateNativeCh20SkyKeyAssets(spec, frames, &afm.Clip{}); err == nil {
		t.Fatal("incomplete ANI snapshots were accepted")
	}
}

func TestNativeCh20SkyKeyBeatRejectsEditedPayloadBeforeMutation(t *testing.T) {
	spec := recoveredCh20SkyKeySpec(t)
	spec.TailFrameEnd = 99
	g := newBeatTestGame(t, []campaign.Beat{{
		Op: "native_ch20_sky_key_sequence", Source: "0x242c9", NativeCh20SkyKey: &spec,
	}})
	g.beatAdvance()
	if g.loadErr == "" || g.nativeCh20SkyKey != nil || g.beatIdx != 0 {
		t.Fatalf("edited payload err=%q job=%#v beat=%d", g.loadErr, g.nativeCh20SkyKey, g.beatIdx)
	}
}

func TestNativeCh20SkyKeyFailureRestoresPublishedMapTransaction(t *testing.T) {
	state := &battle.State{W: 41, H: 40}
	if err := state.MaterializeNativeMapViewState(battle.NativeMapViewState{
		CameraX: 2, CameraY: 3, CursorX: 7, CursorY: 9,
		VisibleCursorX: 5, VisibleCursorY: 6,
	}); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		st: state, curX: 7, curY: 9, camX: 48, camY: 72,
		nativeMapWork: make([]byte, 32), nativeMapVGA: make([]byte, 64),
		nativeMapDAC: make([]byte, 256*3), nativeFDOTHERPalettePhase: 7,
	}
	for index := range g.nativeMapWork {
		g.nativeMapWork[index] = byte(index + 1)
	}
	for index := range g.nativeMapVGA {
		g.nativeMapVGA[index] = byte(index + 2)
	}
	for index := range g.nativeMapDAC {
		g.nativeMapDAC[index] = byte(index % 64)
	}
	wantState := *state
	wantWork := append([]byte(nil), g.nativeMapWork...)
	wantVGA := append([]byte(nil), g.nativeMapVGA...)
	wantDAC := append([]byte(nil), g.nativeMapDAC...)
	wantClock := g.nativeMapClock
	wantCurX, wantCurY, wantCamX, wantCamY := g.curX, g.curY, g.camX, g.camY

	rollback := snapshotNativeCh20SkyKeyState(g)
	state.NativeMapViewState.CameraX = 14
	state.NativeMapViewState.CursorX = 19
	g.curX, g.curY, g.camX, g.camY = 19, 18, 336, 192
	g.nativeMapWork[0], g.nativeMapVGA[0], g.nativeMapDAC[0] = 0xff, 0xfe, 0x3f
	g.nativeFDOTHERPalettePhase = 12
	g.nativeMapClock.elapsedTicks = 123
	g.nativeCh20SkyKey = &nativeCh20SkyKeyJob{rollback: rollback}
	g.failNativeCh20SkyKey(errors.New("synthetic renderer failure"))

	if g.nativeCh20SkyKey != nil || g.loadErr == "" || !reflect.DeepEqual(*state, wantState) {
		t.Fatalf("failed 0x24336 did not restore state: job=%#v err=%q state=%#v want=%#v", g.nativeCh20SkyKey, g.loadErr, *state, wantState)
	}
	if g.curX != wantCurX || g.curY != wantCurY || g.camX != wantCamX || g.camY != wantCamY ||
		!reflect.DeepEqual(g.nativeMapWork, wantWork) || !reflect.DeepEqual(g.nativeMapVGA, wantVGA) ||
		!reflect.DeepEqual(g.nativeMapDAC, wantDAC) || g.nativeMapClock != wantClock ||
		g.nativeFDOTHERPalettePhase != 7 {
		t.Fatalf("failed 0x24336 leaked presentation state: cursor=(%d,%d) camera=(%v,%v) phase=%d", g.curX, g.curY, g.camX, g.camY, g.nativeFDOTHERPalettePhase)
	}
}

func rosterHasItem(roster map[int]battle.Unit, itemID int) bool {
	for _, unit := range roster {
		for _, held := range unit.Inventory {
			if held == itemID {
				return true
			}
		}
	}
	return false
}

func TestChapterTwentyOneSkyKeyBattleResultReachesTownAndSaveBoundary(t *testing.T) {
	fdPath, aniPath := nativeFDOTHERPath(), nativeANIPath()
	if fdPath == "" || aniPath == "" {
		t.Skip("第21戰完整流程需要玩家提供 FDOTHER.DAT 與 ANI.DAT")
	}
	if _, err := os.Stat(fdPath); err != nil {
		t.Skipf("FDOTHER.DAT unavailable: %v", err)
	}
	if _, err := os.Stat(aniPath); err != nil {
		t.Skipf("ANI.DAT unavailable: %v", err)
	}

	// 使用本關可編輯 scenario 的實際出戰順序建立測試用持續隊伍投影。
	// 這只提供既有 JOIN／部署的資料形狀，不證明上一個整備節點或一般玩家
	// 路徑已執行。戰後新增的 24、23 仍只能由正式 join beat 建立。
	scenario, err := battle.LoadScenario(assetPath("assets/scenarios/ch21.json"))
	if err != nil {
		t.Fatal(err)
	}
	order := make([]int, len(scenario.Party))
	members := make(map[int]bool, len(order))
	for index, unit := range scenario.Party {
		order[index] = unit.Fig
		members[unit.Fig] = true
	}
	g := &Game{
		partyMembers:   members,
		partyJoinOrder: append([]int(nil), order...),
		partyDeploy:    make(map[int]bool, 15),
	}
	if len(order) < 16 {
		t.Fatalf("第21戰永久名冊=%d，無法建立原版16名出戰前沿", len(order))
	}
	for _, id := range order[1:16] {
		g.partyDeploy[id] = true
	}
	// 戰前未出戰的永久成員也必須留在名冊；先用同一份可編輯 scenario 的
	// constructor 建立23筆持續紀錄，再由 partyDeploy 投影出本戰16人。
	if err := g.seedPersistentPartyFromLoadCH(order, scenario.PartyUnits(nil)); err != nil {
		t.Fatal(err)
	}
	if err := g.loadMap("assets/maps/map20"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map20/map20_units.json", "assets/scenarios/ch21.json")
	if g.loadErr != "" || g.st == nil || g.sc == nil || len(g.st.Units) != 75 {
		t.Fatalf("第21戰初始化 err=%q state=%v scenario=%v slots=%d", g.loadErr, g.st != nil, g.sc != nil, g.handlerUnitCount())
	}
	if !g.sc.RuntimeAppendGroups || len(g.st.PendingGroups) != 4 ||
		!g.st.PendingGroups[1] || !g.st.PendingGroups[2] || !g.st.PendingGroups[3] || !g.st.PendingGroups[4] {
		t.Fatalf("第21戰 runtime 群組契約 runtime=%v pending=%v", g.sc.RuntimeAppendGroups, g.st.PendingGroups)
	}
	for slot, id := range order[:16] {
		if unit := g.st.Units[slot]; unit == nil || !unit.HasNativeIdentity || unit.NativeIdentity != id {
			t.Fatalf("第21戰出戰 slot%d=%#v，want native identity %d", slot, unit, id)
		}
	}

	// 只改變勝利當下的物品欄，代表玩家在前面關卡已蒐集六件材料。先清掉
	// fixture 可能已有的同編號物品，確保原版「恰好六組配對」規則可被精確驗證。
	const firstIngredient = 0xd1
	for _, unit := range g.st.Units[:16] {
		for index := len(unit.Inventory) - 1; index >= 0; index-- {
			if unit.Inventory[index] >= firstIngredient && unit.Inventory[index] <= 0xd6 {
				unit.RemoveInventoryIndex(index)
			}
		}
	}
	for offset := 0; offset < 6; offset++ {
		if !g.st.Units[offset].AddInventoryItem(firstIngredient+offset, false) {
			t.Fatalf("第21戰 slot%d 無法放入材料 0x%x", offset, firstIngredient+offset)
		}
	}

	// 0x24336 會先以 0x135DD 把原生攝影機移到 (14,8)，因此保留
	// visible cursor=(0,0) 的可追溯入口，讓最後絕對游標自然落在 (14,8)。
	g.curX, g.curY = 0, 0
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{}); err != nil ||
		!g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatalf("第21戰原生視圖初始化失敗: %v", err)
	}
	if _, ok := g.nativeMapHUDInput(); !ok {
		t.Fatal("第21戰原生索引地圖輸入不完整")
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}

	full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	full.Start = "battle_ch21"
	g.camp = campaign.NewRunner(full)
	g.nativeFDOTHERPalettePhase = 5 // 驗證相對 68 次循環，不宣稱原版程序初值固定。
	g.result = "win"
	if !g.confirmBattleResult() || g.result != "" || g.camp.NodeID() != "story_ch21_post_sky_key_intro" {
		t.Fatalf("第21戰勝利邊界 node=%q result=%q err=%q", g.camp.NodeID(), g.result, g.loadErr)
	}
	if got := g.st.Units[0]; got.X != 15 || got.Y != 14 || got.Dir != 2 || !got.HasNativeMapPresentation || got.NativeMapPresentation.Pose != 2 {
		t.Fatalf("第21戰戰後 layout slot0=(%d,%d,pose%d)，want (15,14,2)", got.X, got.Y, got.Dir)
	}
	if got := g.st.Units[25]; got.X != 23 || got.Y != 14 || got.Dir != 1 || !got.HasNativeMapPresentation || got.NativeMapPresentation.Pose != 1 || g.camX != 336 || g.camY != 240 {
		t.Fatalf("第21戰戰後 layout slot25=(%d,%d,pose%d) camera=(%.0f,%.0f)", got.X, got.Y, got.Dir, g.camX, g.camY)
	}

	seen := make(map[nativeCh20SkyKeyPhase]bool)
	seenAct63, seenAct64 := false, false
	skyPaletteStart := -1
	type nativeDialogueKey struct{ stringIndex, utterance int }
	seenNativeDialogue := make(map[nativeDialogueKey]bool, 26)
	screen := ebiten.NewImage(logicalW, logicalH)
	evidenceOut := os.Getenv("FD2_SKY_KEY_EVIDENCE_OUT")
	evidenceTargets := map[nativeCh20SkyKeyPhase]int{
		nativeCh20SkyKeyFirstFrames: 34,
		nativeCh20SkyKeyANI:         48,
		nativeCh20SkyKeyTailFrames:  84,
	}
	evidenceFrames := make(map[nativeCh20SkyKeyPhase]*image.Paletted, len(evidenceTargets))
	for frame := 0; frame < 30000 && g.camp.NodeID() != "town_ch22"; frame++ {
		if job := g.actJob; job != nil {
			seenAct63 = seenAct63 || len(job.acting) == 4
			seenAct64 = seenAct64 || len(job.acting) == 5
		}
		if job := g.nativeCh20SkyKey; job != nil {
			if skyPaletteStart < 0 {
				skyPaletteStart = g.nativeFDOTHERPalettePhase
			}
			seen[job.phase] = true
			if !g.drawNativeCh20SkyKey(screen) {
				t.Fatalf("天空之鑰 phase=%d frame=%d 無法呈現", job.phase, job.frame)
			}
			if target, ok := evidenceTargets[job.phase]; evidenceOut != "" && ok &&
				job.frame == target && evidenceFrames[job.phase] == nil {
				palette := append(color.Palette(nil), job.palette...)
				indexed := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
				copy(indexed.Pix, job.vga)
				evidenceFrames[job.phase] = indexed
			}
			// 測試仍經過正式呈現確認與 stepper，只壓縮 DOS／主機等待時間。
			job.ticks = 0
			g.stepNativeCh20SkyKey()
		}
		if len(g.dialog) != 0 {
			current := g.dialog[len(g.dialog)-1]
			if current.NativeDialogue == nil || current.Upper == nil ||
				current.NativeDialogue.SourceDAT != "FDTXT_021" ||
				len(g.nativeDialogueOpening) != 5 || len(g.nativeDialogueClosing) < 5 {
				t.Fatalf("第21戰戰後對話遺失原生生命週期: %#v", current)
			}
			seenNativeDialogue[nativeDialogueKey{
				stringIndex: current.NativeDialogue.StringIndex,
				utterance:   current.NativeDialogue.Utterance,
			}] = true
			if g.nativeStoryDialogueAtInputWait() &&
				!g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
				t.Fatal("第21戰戰後正式故事輸入遭拒")
			}
			if err := g.Update(); err != nil {
				t.Fatal(err)
			}
			continue
		}
		g.tick(1)
		if g.loadErr != "" {
			t.Fatalf("第21戰戰後流程在 node=%q beat=%d/%d 停止: %s", g.camp.NodeID(), g.beatIdx, len(g.beats), g.loadErr)
		}
	}
	if g.camp.NodeID() != "town_ch22" || g.st != nil || g.handlerChapter != 21 {
		t.Fatalf("天空之鑰流程邊界 node=%q battle=%v chapter=%d", g.camp.NodeID(), g.st != nil, g.handlerChapter)
	}
	if !seenAct63 || !seenAct64 {
		t.Fatalf("天空之鑰相鄰 ACTING 未完整消費: act63=%v act64=%v", seenAct63, seenAct64)
	}
	if len(seenNativeDialogue) != 26 {
		t.Fatalf("第21戰天空之鑰成功臂原生對話=%d，want 26", len(seenNativeDialogue))
	}
	for phase := nativeCh20SkyKeyPan; phase <= nativeCh20SkyKeyTailFrames; phase++ {
		if !seen[phase] {
			t.Errorf("天空之鑰演出未經過 phase=%d", phase)
		}
	}
	if skyPaletteStart < 0 || g.nativeFDOTHERPalettePhase != (skyPaletteStart+68)&15 {
		t.Errorf("0x4DFCC 相對循環 start=%d got=%d，want %d", skyPaletteStart, g.nativeFDOTHERPalettePhase, (skyPaletteStart+68)&15)
	}
	if !g.partyMembers[24] || !g.partyMembers[23] || len(g.partyJoinOrder) < 2 ||
		g.partyJoinOrder[len(g.partyJoinOrder)-2] != 24 || g.partyJoinOrder[len(g.partyJoinOrder)-1] != 23 {
		t.Fatalf("天空之鑰戰後 JOIN chronology=%v members24/23=%v/%v", g.partyJoinOrder, g.partyMembers[24], g.partyMembers[23])
	}
	if !rosterHasItem(g.partyRoster, 0x64) {
		t.Fatal("天空之鑰沒有經 sync_party 進入持續隊伍")
	}
	if evidenceOut != "" {
		order := []nativeCh20SkyKeyPhase{
			nativeCh20SkyKeyFirstFrames,
			nativeCh20SkyKeyANI,
			nativeCh20SkyKeyTailFrames,
		}
		contact := image.NewRGBA(image.Rect(0, 0, 320*len(order), 200))
		for index, phase := range order {
			frame := evidenceFrames[phase]
			if frame == nil {
				t.Fatalf("天空之鑰證據圖缺少 phase=%d frame=%d", phase, evidenceTargets[phase])
			}
			draw.Draw(contact, image.Rect(index*320, 0, (index+1)*320, 200), frame, image.Point{}, draw.Src)
		}
		out, err := os.Create(evidenceOut)
		if err != nil {
			t.Fatalf("建立天空之鑰證據圖: %v", err)
		}
		if err := png.Encode(out, contact); err != nil {
			out.Close()
			t.Fatalf("編碼天空之鑰證據圖: %v", err)
		}
		if err := out.Close(); err != nil {
			t.Fatalf("關閉天空之鑰證據圖: %v", err)
		}
		if info, err := os.Stat(evidenceOut); err != nil || info.Size() == 0 {
			t.Fatalf("天空之鑰證據圖未寫出: size=%v err=%v", func() int64 {
				if info == nil {
					return 0
				}
				return info.Size()
			}(), err)
		}
	}

	// 多數原版關卡在戰後進城鎮；只在這個 node boundary 存檔，證實動畫、
	// JOIN 與配方獎勵沒有被直接接往下一場戰鬥而遺失。
	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Cleanup(func() { userDataDirCached = oldCache })
	g.saveGameToSlot(0)
	if _, err := os.Stat(saveSlotPath(0)); err != nil {
		t.Fatalf("town_ch22 存檔未寫入: %v", err)
	}
	wantOrder := append([]int(nil), g.partyJoinOrder...)
	wantMembers, wantRoster := g.partyMembers, g.partyRoster
	g.partyMembers, g.partyJoinOrder, g.partyRoster = nil, nil, nil
	g.handlerChapter = 0
	g.loadGameFromSlot(0)
	if g.loadErr != "" || g.camp.NodeID() != "town_ch22" || g.st != nil || g.handlerChapter != 21 ||
		!reflect.DeepEqual(g.partyMembers, wantMembers) || !reflect.DeepEqual(g.partyJoinOrder, wantOrder) ||
		!reflect.DeepEqual(g.partyRoster, wantRoster) || !rosterHasItem(g.partyRoster, 0x64) {
		t.Fatalf("town_ch22 存讀檔 node=%q chapter=%d order=%v members=%v roster=%d err=%q", g.camp.NodeID(), g.handlerChapter, g.partyJoinOrder, g.partyMembers, len(g.partyRoster), g.loadErr)
	}
}

func TestChapterTwentyOneSkyKeyInsufficientBranchUsesNativeDialogueAndTownSave(t *testing.T) {
	fdPath, datoPath := nativeFDOTHERPath(), nativeDATOPath()
	if fdPath == "" || datoPath == "" {
		t.Skip("第21戰材料不足原生對話需要玩家提供 FDOTHER.DAT 與 DATO.DAT")
	}
	if _, err := os.Stat(fdPath); err != nil {
		t.Skipf("FDOTHER.DAT unavailable: %v", err)
	}
	if _, err := os.Stat(datoPath); err != nil {
		t.Skipf("DATO.DAT unavailable: %v", err)
	}

	scenario, err := battle.LoadScenario(assetPath("assets/scenarios/ch21.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(scenario.Party) < 16 {
		t.Fatalf("第21戰永久名冊=%d，無法建立16名出戰前沿", len(scenario.Party))
	}
	order := make([]int, len(scenario.Party))
	members := make(map[int]bool, len(order))
	for index, unit := range scenario.Party {
		order[index] = unit.Fig
		members[unit.Fig] = true
	}
	g := &Game{
		partyMembers:   members,
		partyJoinOrder: append([]int(nil), order...),
		partyDeploy:    make(map[int]bool, 15),
	}
	for _, id := range order[1:16] {
		g.partyDeploy[id] = true
	}
	if err := g.seedPersistentPartyFromLoadCH(order, scenario.PartyUnits(nil)); err != nil {
		t.Fatal(err)
	}
	if err := g.loadMap("assets/maps/map20"); err != nil {
		t.Fatal(err)
	}
	g.resetBattle("assets/maps/map20/map20_units.json", "assets/scenarios/ch21.json")
	if g.loadErr != "" || g.st == nil || len(g.st.Units) != 75 {
		t.Fatalf("第21戰材料不足入口 err=%q state=%v slots=%d", g.loadErr, g.st != nil, g.handlerUnitCount())
	}
	// 清除所有六素材，證明這條路徑由正式配方判定進入不足臂，而不是直接節點測試。
	for _, unit := range g.st.Units[:16] {
		for index := len(unit.Inventory) - 1; index >= 0; index-- {
			if unit.Inventory[index] >= 0xd1 && unit.Inventory[index] <= 0xd6 {
				unit.RemoveInventoryIndex(index)
			}
		}
	}
	g.curX, g.curY = 0, 0
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{}); err != nil ||
		!g.st.MaterializeNativeMapHUDState(1, 1, 1) || !g.st.MaterializeNativeMapRangeMode(1) {
		t.Fatalf("第21戰材料不足原生視圖初始化失敗: %v", err)
	}
	if err := g.composeNativeMapFrame(); err != nil {
		t.Fatal(err)
	}
	full, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	full.Start = "battle_ch21"
	g.camp = campaign.NewRunner(full)
	g.result = "win"
	if !g.confirmBattleResult() || g.camp.NodeID() != "story_ch21_post_sky_key_intro" {
		t.Fatalf("第21戰材料不足勝利邊界 node=%q err=%q", g.camp.NodeID(), g.loadErr)
	}

	type nativeDialogueKey struct{ stringIndex, utterance int }
	seen := make(map[nativeDialogueKey]bool, 14)
	seenAct, seenSkyKey := false, false
	for frame := 0; frame < 20000 && g.camp.NodeID() != "town_ch22"; frame++ {
		seenAct = seenAct || g.actJob != nil
		seenSkyKey = seenSkyKey || g.nativeCh20SkyKey != nil
		if len(g.dialog) != 0 {
			current := g.dialog[len(g.dialog)-1]
			if current.NativeDialogue == nil || current.Upper == nil ||
				current.NativeDialogue.SourceDAT != "FDTXT_021" ||
				(current.NativeDialogue.StringIndex != 5 && current.NativeDialogue.StringIndex != 6) ||
				len(g.nativeDialogueOpening) != 5 || len(g.nativeDialogueClosing) < 5 {
				t.Fatalf("第21戰材料不足對話生命週期漂移: %#v", current)
			}
			seen[nativeDialogueKey{current.NativeDialogue.StringIndex, current.NativeDialogue.Utterance}] = true
			if g.nativeStoryDialogueAtInputWait() &&
				!g.handleNativeStoryInput(g.camp.Node(), nativeStoryInput{enter: true}) {
				t.Fatal("第21戰材料不足正式故事輸入遭拒")
			}
			if err := g.Update(); err != nil {
				t.Fatal(err)
			}
			continue
		}
		g.tick(1)
		if g.loadErr != "" {
			t.Fatalf("第21戰材料不足流程 node=%q beat=%d/%d: %s", g.camp.NodeID(), g.beatIdx, len(g.beats), g.loadErr)
		}
	}
	if g.camp.NodeID() != "town_ch22" || g.st != nil || g.handlerChapter != 21 {
		t.Fatalf("第21戰材料不足邊界 node=%q battle=%v chapter=%d", g.camp.NodeID(), g.st != nil, g.handlerChapter)
	}
	if len(seen) != 14 || seenAct || seenSkyKey {
		t.Fatalf("第21戰材料不足對話=%d act=%v sky-key=%v，want 14/false/false", len(seen), seenAct, seenSkyKey)
	}
	if rosterHasItem(g.partyRoster, 0x64) {
		t.Fatal("材料不足分支錯誤授予天空之鑰")
	}
	if !g.partyMembers[24] || !g.partyMembers[23] ||
		g.partyJoinOrder[len(g.partyJoinOrder)-2] != 24 || g.partyJoinOrder[len(g.partyJoinOrder)-1] != 23 {
		t.Fatalf("材料不足分支遺失共同JOIN24／23: %v", g.partyJoinOrder)
	}

	oldCache := userDataDirCached
	userDataDirCached = ""
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	t.Cleanup(func() { userDataDirCached = oldCache })
	g.saveGameToSlot(0)
	wantOrder := append([]int(nil), g.partyJoinOrder...)
	g.partyMembers, g.partyJoinOrder, g.partyRoster = nil, nil, nil
	g.handlerChapter = 0
	g.loadGameFromSlot(0)
	if g.loadErr != "" || g.camp.NodeID() != "town_ch22" || g.handlerChapter != 21 ||
		!reflect.DeepEqual(g.partyJoinOrder, wantOrder) || rosterHasItem(g.partyRoster, 0x64) {
		t.Fatalf("材料不足town_ch22存讀檔 node=%q chapter=%d order=%v err=%q", g.camp.NodeID(), g.handlerChapter, g.partyJoinOrder, g.loadErr)
	}
}
