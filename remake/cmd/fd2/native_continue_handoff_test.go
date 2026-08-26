package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

func TestNativeContinueBattlePublicationFromRealCurrentSnapshot(t *testing.T) {
	const (
		unitsPath    = "assets/maps/map0/map0_units.json"
		scenarioPath = "assets/scenarios/ch01.json"
	)
	savePath := filepath.Join(
		"../../../org_game/炎龍騎士團/FLAME2", "FD2.SAV",
	)
	stored, err := os.ReadFile(savePath)
	if err != nil {
		t.Skipf("player-provided FD2.SAV is absent: %v", err)
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := fdsave.InspectCurrentSnapshot(plain)
	if err != nil {
		t.Fatal(err)
	}
	assetState, err := battle.Load(assetPath(unitsPath))
	if err != nil {
		t.Fatal(err)
	}
	scenario, err := battle.LoadScenario(assetPath(scenarioPath))
	if err != nil {
		t.Fatal(err)
	}
	input, err := fdsave.BuildContinueRuntimeInput(
		snapshot,
		fdsave.ContinueRuntimeContext{
			Chapter:            int(snapshot.Header.Chapter),
			FieldWidth:         assetState.W,
			FieldHeight:        assetState.H,
			SelectorGroupCount: 0x100,
			// The signed BIOS tick is deliberately supplied by the caller;
			// this deterministic contract test uses a valid zero seed and does
			// not claim title-clock or pixel-timing E2.
			TitleTimerTick: 0, HasTitleTimerTick: true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	state := &battle.State{
		W: assetState.W, H: assetState.H,
		OwnDeploy: append([]battle.Cell(nil), assetState.OwnDeploy...),
		Cost:      append([]int(nil), assetState.Cost...),
		NativeCompositionEventBytes: append(
			[]byte(nil), assetState.NativeCompositionEventBytes...,
		),
		NativeMapEventGrid:    append([]byte(nil), assetState.NativeMapEventGrid...),
		HasNativeMapEventGrid: assetState.HasNativeMapEventGrid,
		NativeFieldEventSlots: append(
			[]int(nil), assetState.NativeFieldEventSlots...,
		),
		NativeFieldEvents: append(
			[]battle.NativeFieldEvent(nil), assetState.NativeFieldEvents...,
		),
		NativeFieldEventRules: append(
			[]battle.NativeFieldEventRule(nil), assetState.NativeFieldEventRules...,
		),
		NativeTileBlitModes:  append([]byte(nil), assetState.NativeTileBlitModes...),
		NativeTerrainControl: append([]byte(nil), assetState.NativeTerrainControl...),
		NativeTerrainMoveCodes: append(
			[]byte(nil), assetState.NativeTerrainMoveCodes...,
		),
	}
	if err := campaign.MaterializeNativeContinueFieldBoundary(
		state, input, int(snapshot.Header.Chapter),
	); err != nil {
		t.Fatal(err)
	}
	catalog, err := campaign.LoadNativeCharacterCatalog(
		assetPath("assets/data/native_character_catalog.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinueRuntimeUnits(
		state, input, catalog,
	); err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinueMapTiming(state, input); err != nil {
		t.Fatal(err)
	}
	itemRows, err := battle.LoadNativeItemEffectRowPrefix(
		assetPath("assets/data/native_item_effect_rows.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinuePendingGroups(
		state, input, int(snapshot.Header.Chapter), assetState, scenario, itemRows,
	); err != nil {
		t.Fatal(err)
	}
	if err := campaign.MaterializeNativeContinueInteractiveBoundary(state, input); err != nil {
		t.Fatal(err)
	}

	graph := &campaign.Campaign{
		Start: "battle_ch01",
		Nodes: map[string]*campaign.Node{
			"battle_ch01": {
				Type: "battle", Units: unitsPath, Scenario: scenarioPath,
			},
		},
	}
	g := &Game{
		camp: campaign.NewRunner(graph),
		m: &MapData{
			W: assetState.W, H: assetState.H, TileW: 24, TileH: 24,
		},
		dialog: []battle.DialogLine{{Text: "stale"}},
	}
	if err := g.publishNativeContinueBattle(
		input, state, scenario, "battle_ch01", unitsPath, scenarioPath,
	); err != nil {
		t.Fatal(err)
	}
	if g.camp.NodeID() != "battle_ch01" || g.st != state || g.sc != scenario ||
		g.titlePhase != "" || len(g.dialog) != 0 || g.sel != nil ||
		g.curX != int(snapshot.Header.CursorX) ||
		g.curY != int(snapshot.Header.CursorY) {
		t.Fatalf("published CONTINUE state node=%q st=%p sc=%p title=%q cursor=(%d,%d)",
			g.camp.NodeID(), g.st, g.sc, g.titlePhase, g.curX, g.curY)
	}
}

func TestNativeContinueTitleCallerPublishesRealCurrentSnapshot(t *testing.T) {
	savePath := filepath.Join(
		"../../../org_game/炎龍騎士團/FLAME2", "FD2.SAV",
	)
	if _, err := os.Stat(savePath); err != nil {
		t.Skipf("player-provided FD2.SAV is absent: %v", err)
	}
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_NATIVE_TITLE_TICK", "")
	g := &Game{
		camp:       campaign.NewRunner(graph),
		titlePhase: "menu",
	}
	if err := g.loadNativeContinueFromCurrentSnapshot(savePath); err != nil {
		t.Fatal(err)
	}
	if g.camp == nil || g.camp.NodeID() != "battle_ch01" ||
		g.st == nil || g.sc == nil || g.titlePhase != "" ||
		g.st.NativeRoundCounter != 1 || g.curX != 8 || g.curY != 17 ||
		g.st.NativeMapViewState.CameraX != 1 || g.st.NativeMapViewState.CameraY != 13 {
		t.Fatalf(
			"title CONTINUE publication node=%v title=%q state=%p scenario=%p round=%d cursor=(%d,%d) view=%+v",
			g.camp.NodeID(), g.titlePhase, g.st, g.sc, g.st.NativeRoundCounter,
			g.curX, g.curY, g.st.NativeMapViewState,
		)
	}
}

func TestNativeContinueTitleCallerPublishesLateBattleCandidate(t *testing.T) {
	savePath := os.Getenv("FD2_LATE_NATIVE_SAVE")
	if savePath == "" {
		t.Skip("未提供外部晚期 FD2.SAV 候選")
	}
	stored, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := fdsave.Decode(stored)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := fdsave.InspectCurrentSnapshot(plain)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Header.Chapter != 0x1d || snapshot.Header.RuntimeCount != 0x21 ||
		snapshot.Header.PersistentCount != 0x1f {
		t.Fatalf("晚期候選 header=%+v，不是已審查的第30戰來源", snapshot.Header)
	}
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_NATIVE_TITLE_TICK", "")
	g := &Game{camp: campaign.NewRunner(graph), titlePhase: "menu"}
	if err := g.loadNativeContinueFromCurrentSnapshot(savePath); err != nil {
		t.Fatal(err)
	}
	if g.camp == nil || g.camp.NodeID() != "battle_ch30" || g.st == nil || g.sc == nil ||
		g.titlePhase != "" || g.st.NativeRoundCounter != 0x0c ||
		len(g.st.Units) != 0x21 || len(g.partyJoinOrder) != 0x1f ||
		g.curX != 0x15 || g.curY != 0x14 ||
		g.st.NativeMapViewState.CameraX != 0x10 || g.st.NativeMapViewState.CameraY != 0x10 {
		t.Fatalf("晚期 CONTINUE 發布不完整：node=%q title=%q units=%d party=%d round=%d cursor=(%d,%d) view=%+v",
			g.camp.NodeID(), g.titlePhase, len(g.st.Units), len(g.partyJoinOrder),
			g.st.NativeRoundCounter, g.curX, g.curY, g.st.NativeMapViewState)
	}
}

func TestNativeContinueLateBattleIndexedStages(t *testing.T) {
	savePath := os.Getenv("FD2_LATE_NATIVE_SAVE")
	outputDir := os.Getenv("FD2_LATE_NATIVE_STAGE_DIR")
	if savePath == "" || outputDir == "" {
		t.Skip("未提供外部晚期 FD2.SAV 與明確的分階段輸出目錄")
	}
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FD2_NATIVE_TITLE_TICK", "0")
	t.Setenv("FD2_TITLE", "0")
	t.Setenv("FD2_CAMPAIGN", "")
	originalBase := "../../../org_game/炎龍騎士團/FLAME2"
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(originalBase, "FDOTHER.DAT"))
	g := loadGame()
	if g.loadErr != "" {
		t.Fatal(g.loadErr)
	}
	g.camp, g.titlePhase = campaign.NewRunner(graph), "menu"
	if err := g.loadNativeContinueFromCurrentSnapshot(savePath); err != nil {
		t.Fatal(err)
	}
	if _, err := loadNativeMapAssets(assetPath("assets/maps/map29")); err != nil {
		t.Fatalf("map29 原生資產載入錯誤：%v", err)
	}
	if g.camp.NodeID() != "battle_ch30" || g.st == nil || g.m == nil ||
		!nativeMapAssetsAvailable(g.nativeMapAssets) {
		t.Fatalf(
			"晚期 CONTINUE 未建立完整第30戰 indexed frame input: node=%q state=%v map=%v assets=%v",
			g.camp.NodeID(), g.st != nil, g.m != nil, nativeMapAssetsAvailable(g.nativeMapAssets),
		)
	}
	hud, ok := g.nativeMapHUDInput()
	if !ok {
		t.Fatal("晚期 CONTINUE 缺少原生 HUD input")
	}
	in, err := buildNativeMapFrameInput(
		g.nativeMapAssets, g.m, g.st, nativeMapFrameRuntime{HUD: hud},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	const (
		workStride = fdicon.NativeMapStride
		workBase   = 0x8088
		viewWidth  = 320
		viewHeight = 200
	)
	stageNames := map[indexedmap.FrameStage]string{
		indexedmap.FrameStageTerrain:    "terrain",
		indexedmap.FrameStageRange:      "range",
		indexedmap.FrameStageUnits:      "units",
		indexedmap.FrameStageForeground: "foreground",
		indexedmap.FrameStageHUD:        "hud",
		indexedmap.FrameStageViewport:   "viewport",
	}
	type stageResult struct {
		Name           string `json:"name"`
		WorkSHA256     string `json:"work_sha256"`
		ViewportSHA256 string `json:"viewport_sha256"`
		Image          string `json:"image"`
	}
	type unitResult struct {
		Index  int  `json:"index"`
		X      int  `json:"x"`
		Y      int  `json:"y"`
		HP     int  `json:"hp"`
		Byte5  byte `json:"native_record_byte5"`
		Camp   int  `json:"camp"`
		Active bool `json:"active"`
	}
	results := make([]stageResult, 0, len(stageNames))
	stageWork := make(map[indexedmap.FrameStage][]byte)
	writeStage := func(stage indexedmap.FrameStage, work []byte) error {
		name, exists := stageNames[stage]
		if !exists {
			return errors.New("未知 indexed frame stage")
		}
		viewport := make([]byte, viewWidth*viewHeight)
		if err := fdicon.CopyNativeIndexedRegion(
			viewport[4*viewWidth+4:], viewWidth,
			work[workBase:], workStride, 312, 192,
		); err != nil {
			return err
		}
		img := image.NewPaletted(image.Rect(0, 0, viewWidth, viewHeight), g.nativeMapAssets.Palette)
		copy(img.Pix, viewport)
		imageName := name + ".png"
		file, err := os.Create(filepath.Join(outputDir, imageName))
		if err != nil {
			return err
		}
		encodeErr := png.Encode(file, img)
		closeErr := file.Close()
		if encodeErr != nil {
			return encodeErr
		}
		if closeErr != nil {
			return closeErr
		}
		workHash, viewportHash := sha256.Sum256(work), sha256.Sum256(viewport)
		results = append(results, stageResult{
			Name: name, WorkSHA256: hex.EncodeToString(workHash[:]),
			ViewportSHA256: hex.EncodeToString(viewportHash[:]), Image: imageName,
		})
		stageWork[stage] = append([]byte(nil), work...)
		return nil
	}
	work := make([]byte, indexedmap.NativeUnitPresentWorkSize)
	vga := make([]byte, indexedmap.NativeMapVGASize)
	if err := indexedmap.ComposeNativeFrameObserved(
		work, vga, in,
		func(stage indexedmap.FrameStage, observedWork, _ []byte) error {
			return writeStage(stage, observedWork)
		},
	); err != nil {
		t.Fatal(err)
	}
	// The original candidate frame may have been captured at any of
	// sub_4EB90's sixteen raw phases. Emit every phase from the same typed
	// state so comparison tooling can select a time-aligned oracle without
	// changing production state or conflating this cycle with terrain LUTs.
	for phase := 0; phase < 16; phase++ {
		phaseInput := in
		phaseInput.Frame.ChapterAuxPhase = phase
		phaseWork := make([]byte, indexedmap.NativeUnitPresentWorkSize)
		phaseVGA := make([]byte, indexedmap.NativeMapVGASize)
		if err := indexedmap.ComposeNativeFrame(phaseWork, phaseVGA, phaseInput); err != nil {
			t.Fatalf("auxiliary phase %d: %v", phase, err)
		}
		img := image.NewPaletted(image.Rect(0, 0, viewWidth, viewHeight), g.nativeMapAssets.Palette)
		copy(img.Pix, phaseVGA)
		file, err := os.Create(filepath.Join(outputDir, fmt.Sprintf("aux-phase-%02d.png", phase)))
		if err != nil {
			t.Fatal(err)
		}
		encodeErr := png.Encode(file, img)
		closeErr := file.Close()
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	if oraclePath := os.Getenv("FD2_LATE_NATIVE_ORACLE"); oraclePath != "" {
		oracleFile, err := os.Open(oraclePath)
		if err != nil {
			t.Fatal(err)
		}
		oracle, _, decodeErr := image.Decode(oracleFile)
		closeErr := oracleFile.Close()
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		if oracle.Bounds() != image.Rect(0, 0, viewWidth, viewHeight) {
			t.Fatalf("原版候選尺寸=%v want 320x200", oracle.Bounds())
		}
		type phaseMatch struct {
			AE           int `json:"ae"`
			Aux          int `json:"aux_phase"`
			Idle         int `json:"idle_cycle"`
			Moving       int `json:"moving_cycle"`
			Flip         int `json:"terrain_flip"`
			PixelShift   int `json:"unit_pixel_shift"`
			Palette      int `json:"palette_phase"`
			Combinations int `json:"combinations"`
		}
		best := phaseMatch{AE: viewWidth * viewHeight, Combinations: 16 * 4 * 4 * 2 * 2}
		var bestVGA []byte
		for aux := 0; aux < 16; aux++ {
			for idle := 0; idle < 4; idle++ {
				for moving := 0; moving < 4; moving++ {
					for flip := 0; flip < 2; flip++ {
						for shift := 0; shift < 2; shift++ {
							candidate := in
							candidate.Frame.ChapterAuxPhase = aux
							candidate.Frame.IdleCycle = idle
							candidate.Frame.MovingCycle = moving
							candidate.Frame.Flip = flip
							candidate.Frame.PixelShift = shift
							candidateWork := make([]byte, indexedmap.NativeUnitPresentWorkSize)
							candidateVGA := make([]byte, indexedmap.NativeMapVGASize)
							if err := indexedmap.ComposeNativeFrame(candidateWork, candidateVGA, candidate); err != nil {
								t.Fatalf("相位組合aux=%d idle=%d moving=%d flip=%d shift=%d: %v", aux, idle, moving, flip, shift, err)
							}
							ae := 0
							for y := 0; y < viewHeight; y++ {
								for x := 0; x < viewWidth; x++ {
									gotR, gotG, gotB, _ := g.nativeMapAssets.Palette[candidateVGA[y*viewWidth+x]].RGBA()
									wantR, wantG, wantB, _ := oracle.At(x, y).RGBA()
									if gotR != wantR || gotG != wantG || gotB != wantB {
										ae++
									}
								}
							}
							if ae < best.AE {
								best.AE, best.Aux, best.Idle, best.Moving, best.Flip, best.PixelShift = ae, aux, idle, moving, flip, shift
								bestVGA = append(bestVGA[:0], candidateVGA...)
							}
						}
					}
				}
			}
		}
		geometryBest := best
		best.AE = viewWidth * viewHeight
		var bestPalette color.Palette
		for palettePhase := 0; palettePhase < 16; palettePhase++ {
			dac := append([]byte(nil), g.nativeMapAssets.PaletteDAC...)
			if err := fdother.ApplyNativeDACPaletteCycleE0EF(dac, palettePhase); err != nil {
				t.Fatal(err)
			}
			palette, err := fdother.VGAPaletteFromDAC(dac)
			if err != nil {
				t.Fatal(err)
			}
			ae := 0
			for y := 0; y < viewHeight; y++ {
				for x := 0; x < viewWidth; x++ {
					gotR, gotG, gotB, _ := palette[bestVGA[y*viewWidth+x]].RGBA()
					wantR, wantG, wantB, _ := oracle.At(x, y).RGBA()
					if gotR != wantR || gotG != wantG || gotB != wantB {
						ae++
					}
				}
			}
			if ae < best.AE {
				best = geometryBest
				best.AE, best.Palette = ae, palettePhase
				bestPalette = palette
			}
		}
		if best.AE != 0 || best.Aux != 10 || best.Idle != 0 || best.Moving != 0 ||
			best.Flip != 0 || best.PixelShift != 0 || best.Palette != 0 {
			t.Fatalf("第30戰相位配對未閉合：%+v", best)
		}
		bestJSON, err := json.MarshalIndent(best, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(outputDir, "animation-phase-best.json"), append(bestJSON, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		bestImage := image.NewPaletted(image.Rect(0, 0, viewWidth, viewHeight), bestPalette)
		copy(bestImage.Pix, bestVGA)
		bestFile, err := os.Create(filepath.Join(outputDir, "animation-phase-best.png"))
		if err != nil {
			t.Fatal(err)
		}
		encodeErr := png.Encode(bestFile, bestImage)
		closeErr = bestFile.Close()
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	rangeWork, unitsWork := stageWork[indexedmap.FrameStageRange], stageWork[indexedmap.FrameStageUnits]
	foregroundWork, hudWork := stageWork[indexedmap.FrameStageForeground], stageWork[indexedmap.FrameStageHUD]
	if len(rangeWork) == 0 || len(unitsWork) == 0 || len(foregroundWork) == 0 || len(hudWork) == 0 {
		t.Fatal("indexed observer 未回傳完整中間階段")
	}
	unitWrites, foregroundOverwrites, hudOverwrites := 0, 0, 0
	for index := range unitsWork {
		if unitsWork[index] == rangeWork[index] {
			continue
		}
		unitWrites++
		if foregroundWork[index] != unitsWork[index] {
			foregroundOverwrites++
		}
		if hudWork[index] != foregroundWork[index] {
			hudOverwrites++
		}
	}
	active, admitted := 0, 0
	units := make([]unitResult, 0, len(g.st.Units))
	view := g.st.NativeMapViewState
	for index, unit := range in.Frame.Units {
		runtimeUnit := g.st.Units[index]
		units = append(units, unitResult{
			Index: index, X: unit.X, Y: unit.Y, HP: runtimeUnit.HP,
			Byte5: unit.Flags, Camp: int(runtimeUnit.Camp), Active: !unit.Inactive,
		})
		if unit.Inactive {
			continue
		}
		active++
		if unit.X >= view.CameraX-1 && unit.X <= view.CameraX+12 &&
			unit.Y >= view.CameraY-1 && unit.Y <= view.CameraY+8 {
			admitted++
		}
	}
	summary := struct {
		CampaignNode         string        `json:"campaign_node"`
		RuntimeCount         int           `json:"runtime_count"`
		ActiveCount          int           `json:"active_count"`
		CameraAdmittedCount  int           `json:"camera_admitted_count"`
		UnitWrites           int           `json:"unit_stage_written_pixels"`
		ForegroundOverwrites int           `json:"foreground_overwritten_unit_pixels"`
		HUDOverwrites        int           `json:"hud_overwritten_remaining_unit_pixels"`
		Stages               []stageResult `json:"stages"`
		Units                []unitResult  `json:"units"`
	}{
		CampaignNode: g.camp.NodeID(), RuntimeCount: len(g.st.Units),
		ActiveCount: active, CameraAdmittedCount: admitted,
		UnitWrites: unitWrites, ForegroundOverwrites: foregroundOverwrites,
		HUDOverwrites: hudOverwrites, Stages: results, Units: units,
	}
	raw, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(outputDir, "stages.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if active != 19 || admitted != 18 || unitWrites == 0 {
		t.Fatalf("晚期 stage admission=%d/%d unit writes=%d", admitted, active, unitWrites)
	}
}

func TestNativeCurrentLoadPreparesPrivateCandidate(t *testing.T) {
	savePath := filepath.Join(
		"../../../org_game/炎龍騎士團/FLAME2", "FD2.SAV",
	)
	if _, err := os.Stat(savePath); err != nil {
		t.Skipf("player-provided FD2.SAV is absent: %v", err)
	}
	graph, err := campaign.Load(assetPath("assets/scenarios/campaign_full.json"))
	if err != nil {
		t.Fatal(err)
	}
	g := &Game{camp: campaign.NewRunner(graph), titlePhase: "unchanged"}
	candidate, err := g.prepareNativeContinueFromCurrentSnapshot(savePath, 0)
	if err != nil {
		t.Fatal(err)
	}
	if g.titlePhase != "unchanged" || g.st != nil {
		t.Fatal("private current LOAD preflight mutated the live Game")
	}
	if candidate.st == nil || candidate.sc == nil || candidate.camp.NodeID() != "battle_ch01" ||
		len(candidate.nativeCurrentSavePlain) != fdsave.FileSize {
		t.Fatalf("current LOAD candidate is incomplete: node=%q state=%p scenario=%p raw=%d",
			candidate.camp.NodeID(), candidate.st, candidate.sc, len(candidate.nativeCurrentSavePlain))
	}
}

func TestNativeContinueTitleTimerSeedUsesRuntimeClockAndRejectsBadOverride(t *testing.T) {
	t.Setenv("FD2_NATIVE_TITLE_TICK", "")
	g := &Game{}
	if tick, err := g.nativeContinueTitleTimerSeed(); err != nil || tick != 0 {
		t.Fatalf("runtime title seed=(%d,%v), want (0,nil)", tick, err)
	}
	if !g.nativeTitleClock.Seed(123, time.Unix(400, 0)) {
		t.Fatal("title clock seed rejected")
	}
	if tick, err := g.nativeContinueTitleTimerSeed(); err != nil || tick != 123 {
		t.Fatalf("current title seed=(%d,%v), want (123,nil)", tick, err)
	}
	t.Setenv("FD2_NATIVE_TITLE_TICK", "32768")
	if _, err := g.nativeContinueTitleTimerSeed(); err == nil {
		t.Fatal("out-of-range signed BIOS tick was accepted")
	}
	t.Setenv("FD2_NATIVE_TITLE_TICK", "-321")
	if tick, err := g.nativeContinueTitleTimerSeed(); err != nil || tick != -321 {
		t.Fatalf("explicit title override=(%d,%v), want (-321,nil)", tick, err)
	}
}

func nativeSystemOverlayTestCells() []*ebiten.Image {
	cells := make([]*ebiten.Image, nativeActionOverlayCellCount)
	for _, index := range []int{12, 15, 18, 21} {
		cells[index] = ebiten.NewImage(1, 1)
	}
	return cells
}

func TestNativeContinueOpeningConfirmHandsOffToSharedSystemOverlay(t *testing.T) {
	g := &Game{
		m: &MapData{W: 24, H: 24, TileW: 24, TileH: 24},
		st: &battle.State{
			W: 24, H: 24,
			HasNativeMapViewState: true,
			NativeMapViewState:    battle.NativeMapViewState{CursorX: 8, CursorY: 17},
		},
		curX: 8, curY: 17,
		nativeContinueOpeningConfirm: true,
		nativeActionCells:            nativeSystemOverlayTestCells(),
	}
	g.confirm()
	if g.sel != nil || !g.nativeSystemCursorOverlay || !g.ring ||
		g.actionOverlayPhase != actionOverlayOpening || g.ringSel != 0 ||
		g.reach != nil || g.nativeContinueOpeningConfirm {
		t.Fatalf("first native CONTINUE confirm did not open action overlay: %+v", g)
	}

	// 一次性 CONTINUE 旗標已消費，但 0x117E7 的共用空游標 owner 仍會
	// 在下一次確認重新開啟同一面板。
	g.resetActionOverlayLifecycle()
	g.moved, g.reach = false, nil
	g.confirm()
	if !g.ring || g.sel != nil || g.moved || g.nativeContinueOpeningConfirm || !g.nativeSystemCursorOverlay {
		t.Fatalf("shared empty-cursor owner was not reusable after CONTINUE bridge: %+v", g)
	}
}

func TestNativeContinueOpeningConfirmRejectsMovedCursor(t *testing.T) {
	unit := &battle.Unit{Camp: battle.Own, OnField: true, HP: 10, X: 7, Y: 17}
	g := &Game{
		m: &MapData{W: 24, H: 24, TileW: 24, TileH: 24},
		st: &battle.State{
			W: 24, H: 24, Units: []*battle.Unit{unit},
			HasNativeMapViewState: true,
			NativeMapViewState:    battle.NativeMapViewState{CursorX: 8, CursorY: 17},
		},
		curX: 7, curY: 17,
		nativeContinueOpeningConfirm: true,
		nativeActionCells:            nativeSystemOverlayTestCells(),
	}
	g.confirm()
	if g.ring || g.sel != unit || g.moved || g.nativeContinueOpeningConfirm || g.nativeSystemCursorOverlay {
		t.Fatalf("moved cursor must stay on normal selection path: %+v", g)
	}
}

func TestNativeSystemDownEndYesEntersAndCompletesEnemyPhase(t *testing.T) {
	state := &battle.State{W: 24, H: 24, Units: []*battle.Unit{
		{Camp: battle.Own, OnField: true, HP: 10, MaxHP: 10},
		{Camp: battle.Enemy, OnField: false, HP: 0, MaxHP: 10},
	}}
	g := &Game{
		st: state, sc: &battle.Scenario{}, ring: true, ringSel: 3,
		nativeSystemCursorOverlay: true,
	}
	base := "../../../org_game/炎龍騎士團/FLAME2"
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	var err error
	g.nativePreparationUI, err = loadNativePreparationUIAssets()
	if err != nil {
		t.Skipf("player-provided original UI assets are absent: %v", err)
	}
	g.nativeClassUI, err = loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeMapVGA = make([]byte, 320*200)
	if !g.beginNativeSystemEndTurn() {
		t.Fatal("verified Down→END route was rejected")
	}
	if g.aiBusy || g.nativeSystemEndTurnConfirm {
		t.Fatal("enemy phase or confirmation started before four close presents")
	}
	for present := 0; present < 4; present++ {
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	if !g.nativeSystemEndTurnConfirm || g.nativeSystemCursorOverlay || g.ring ||
		g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 10 {
		t.Fatalf("END confirmation handoff = confirm:%v overlay:%v ring:%v",
			g.nativeSystemEndTurnConfirm, g.nativeSystemCursorOverlay, g.ring)
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}

	g.confirmNativeSystemEndTurn()
	if g.aiBusy || g.nativeSystemEndTurnDelay != 0 || g.nativeClassUIJob == nil ||
		len(g.nativeClassUIJob.frames) != 4 {
		t.Fatalf("YES choice-close boundary: ai=%v delay=%d job=%#v",
			g.aiBusy, g.nativeSystemEndTurnDelay, g.nativeClassUIJob)
	}
	choiceJob := g.nativeClassUIJob
	for g.nativeClassUIJob == choiceJob {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) == 0 {
		t.Fatal("YES choice close did not start progressive response")
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if g.nativeSystemEndTurnDelay != nativeSystemEndTurnDelayFrames {
		t.Fatalf("YES response delay=%d", g.nativeSystemEndTurnDelay)
	}
	for frame := 1; frame < nativeSystemEndTurnDelayFrames; frame++ {
		g.stepNativeSystemEndTurn()
		if g.aiBusy {
			t.Fatalf("enemy phase started before 0xC8 ms boundary at frame %d", frame)
		}
	}
	g.stepNativeSystemEndTurn()
	if g.aiBusy || g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 5 ||
		len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatalf("dialogue close boundary: ai=%v job=%#v", g.aiBusy, g.nativeClassUIJob)
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if !g.aiBusy || g.banner != "ENEMY PHASE" || g.nativeSystemEndTurnUI != nil {
		t.Fatalf("restored YES did not enter enemy phase: ai=%v banner=%q ui=%#v",
			g.aiBusy, g.banner, g.nativeSystemEndTurnUI)
	}
	g.aiStep()
	if g.aiBusy || state.Turn != 1 || g.banner != "PLAYER PHASE" {
		t.Fatalf("enemy phase did not return to player: ai=%v turn=%d banner=%q",
			g.aiBusy, state.Turn, g.banner)
	}
}

func TestNativeSystemEndTurnRemainsNarrowAndCancelable(t *testing.T) {
	state := &battle.State{W: 24, H: 24}
	for direction := 0; direction < 3; direction++ {
		g := &Game{st: state, ring: true, ringSel: direction, nativeSystemCursorOverlay: true}
		if g.beginNativeSystemEndTurn() {
			t.Fatalf("unverified direction %d opened END confirmation", direction)
		}
	}

	g := &Game{st: state, nativeSystemEndTurnConfirm: true}
	g.cancelNativeSystemEndTurn()
	if g.nativeSystemEndTurnConfirm || g.aiBusy || state.Turn != 0 {
		t.Fatalf("cancel mutated turn: confirm=%v ai=%v turn=%d",
			g.nativeSystemEndTurnConfirm, g.aiBusy, state.Turn)
	}
}

func TestNativeSystemEndTurnFailsClosedBeforeOverlayMutation(t *testing.T) {
	g := &Game{
		st: &battle.State{W: 24, H: 24}, ring: true, ringSel: 3,
		nativeSystemCursorOverlay: true,
	}
	if g.beginNativeSystemEndTurn() || !g.ring || !g.nativeSystemCursorOverlay ||
		g.actionOverlayPhase != "" || g.nativeSystemEndTurnUI != nil {
		t.Fatalf("missing indexed assets mutated END overlay: %+v", g)
	}
}

func TestNativeSystemEndTurnNoClosesAndRestoresWithoutTurnMutation(t *testing.T) {
	state := &battle.State{W: 24, H: 24}
	g := &Game{
		st: state, sc: &battle.Scenario{}, ring: true, ringSel: 3,
		nativeSystemCursorOverlay: true,
	}
	base := "../../../org_game/炎龍騎士團/FLAME2"
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	var err error
	g.nativePreparationUI, err = loadNativePreparationUIAssets()
	if err != nil {
		t.Skipf("player-provided original UI assets are absent: %v", err)
	}
	g.nativeClassUI, err = loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeMapVGA = make([]byte, 320*200)
	if !g.beginNativeSystemEndTurn() {
		t.Fatal("verified END route was rejected")
	}
	for present := 0; present < 4; present++ {
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	g.cancelNativeSystemEndTurn()
	if g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 4 {
		t.Fatalf("NO did not start four choice-close frames: %#v", g.nativeClassUIJob)
	}
	choiceJob := g.nativeClassUIJob
	for g.nativeClassUIJob == choiceJob {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) == 0 {
		t.Fatal("NO choice close did not start progressive response")
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	for g.nativeSystemEndTurnDelay > 0 {
		g.stepNativeSystemEndTurn()
	}
	if g.nativeClassUIJob == nil || len(g.nativeClassUIJob.frames) != 5 ||
		len(g.nativeClassUIJob.restore) != 320*200 {
		t.Fatalf("NO did not start dialogue close and restore: %#v", g.nativeClassUIJob)
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if g.nativeSystemEndTurnUI != nil || g.aiBusy || state.Turn != 0 {
		t.Fatalf("NO mutated turn: ui=%#v ai=%v turn=%d", g.nativeSystemEndTurnUI, g.aiBusy, state.Turn)
	}
}

func TestNativeNestedSystemExitTerminatesOnlyAfterAcceptedClose(t *testing.T) {
	state := &battle.State{W: 24, H: 24}
	g := &Game{
		st: state, sc: &battle.Scenario{}, ring: true, ringSel: 3,
		nativeSystemCursorOverlay: true, nativeSystemNestedOpen: true,
		bgmCur: "battle.xmi", nativeSystemBGMTrack: "battle.xmi",
	}
	base := "../../../org_game/炎龍騎士團/FLAME2"
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	var err error
	g.nativePreparationUI, err = loadNativePreparationUIAssets()
	if err != nil {
		t.Skipf("player-provided original UI assets are absent: %v", err)
	}
	g.nativeClassUI, err = loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeMapVGA = make([]byte, 320*200)
	if !g.beginNativeNestedSystemExit() {
		t.Fatal("verified nested selector3 exit route was rejected")
	}
	if g.nativeSystemExitRequested || !g.nativeSystemNestedOpen {
		t.Fatal("exit was published before nested overlay close")
	}
	for present := 0; present < 4; present++ {
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	if !g.nativeSystemEndTurnConfirm || g.nativeSystemNestedOpen || g.ring ||
		g.nativeSystemEndTurnUI == nil || !g.nativeSystemEndTurnUI.exitProgram {
		t.Fatalf("nested exit confirmation handoff failed: %+v", g)
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	g.confirmNativeSystemEndTurn()
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if g.nativeSystemEndTurnDelay != nativeSystemEndTurnDelayFrames ||
		g.bgmCur != "" || g.nativeSystemBGMTrack != "" || g.nativeSystemExitRequested {
		t.Fatalf("accepted response boundary: delay=%d bgm=%q saved=%q exit=%v",
			g.nativeSystemEndTurnDelay, g.bgmCur, g.nativeSystemBGMTrack,
			g.nativeSystemExitRequested)
	}
	for g.nativeSystemEndTurnDelay > 0 {
		g.stepNativeSystemEndTurn()
	}
	if g.nativeClassUIJob == nil || g.nativeSystemExitRequested {
		t.Fatal("exit was published before dialogue closing frames")
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if !g.nativeSystemExitRequested || g.aiBusy || state.Turn != 0 {
		t.Fatalf("completed exit boundary: exit=%v ai=%v turn=%d",
			g.nativeSystemExitRequested, g.aiBusy, state.Turn)
	}
}

func TestNativeNestedSystemExitCancelAndMissingAssetsStayInGame(t *testing.T) {
	missing := &Game{
		st: &battle.State{W: 24, H: 24}, ring: true, ringSel: 3,
		nativeSystemCursorOverlay: true, nativeSystemNestedOpen: true,
	}
	if missing.beginNativeNestedSystemExit() || !missing.ring ||
		!missing.nativeSystemNestedOpen || missing.actionOverlayPhase != "" ||
		missing.nativeSystemExitRequested {
		t.Fatalf("missing assets mutated nested exit owner: %+v", missing)
	}

	state := &battle.State{W: 24, H: 24}
	g := &Game{
		st: state, sc: &battle.Scenario{}, ring: true, ringSel: 3,
		nativeSystemCursorOverlay: true, nativeSystemNestedOpen: true,
	}
	base := "../../../org_game/炎龍騎士團/FLAME2"
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	var err error
	g.nativePreparationUI, err = loadNativePreparationUIAssets()
	if err != nil {
		t.Skipf("player-provided original UI assets are absent: %v", err)
	}
	g.nativeClassUI, err = loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeMapVGA = make([]byte, 320*200)
	if !g.beginNativeNestedSystemExit() {
		t.Fatal("verified nested selector3 exit route was rejected")
	}
	for present := 0; present < 4; present++ {
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	g.cancelNativeSystemEndTurn()
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	for g.nativeSystemEndTurnDelay > 0 {
		g.stepNativeSystemEndTurn()
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if g.nativeSystemExitRequested || g.aiBusy || state.Turn != 0 || g.nativeSystemEndTurnUI != nil {
		t.Fatalf("cancel mutated game state: exit=%v ai=%v turn=%d ui=%#v",
			g.nativeSystemExitRequested, g.aiBusy, state.Turn, g.nativeSystemEndTurnUI)
	}
}

func TestNativeNestedSystemExitPublishesEbitenTermination(t *testing.T) {
	g := &Game{nativeSystemExitRequested: true}
	if err := g.Update(); !errors.Is(err, ebiten.Termination) {
		t.Fatalf("Update error=%v want ebiten.Termination", err)
	}
}

func nativeSystemGroupMarchUnit(x, y int) *battle.Unit {
	return &battle.Unit{
		Camp: battle.Own, X: x, Y: y, OnField: true, HP: 20, MaxHP: 20,
		MP: 4, MaxMP: 4, AP: 20, DP: 1, MV: 2,
		BattleFig: 1, HasBattleFig: true,
		NativeRecordByte5: 0, HasNativeRecordByte5: true,
		NativeRecordByte6: 2, HasNativeRecordByte6: true,
		NativeRecordByte34: 0, HasNativeRecordByte34: true,
		NativeRecordByte35: 0, HasNativeRecordByte35: true,
		NativeRecordByte36: 0, HasNativeRecordByte36: true,
		NativeRecordByte8: 1, HasNativeRecordByte8: true,
		NativeRecordRace: 0, HasNativeRecordRace: true,
		NativeRecordClass: 0, HasNativeRecordClass: true,
		NativeRecordWord42: 20, HasNativeRecordWord42: true,
		NativeRecordWord46: 4, HasNativeRecordWord46: true,
		NativeMapPresentation:    battle.NativeMapPresentationState{X: byte(x), Y: byte(y)},
		HasNativeMapPresentation: true,
		InventorySlots:           []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		NativeInventoryFlags:     []int{0, 0, 0, 0, 0, 0, 0, 0},
	}
}

func TestNativeSystemGroupMarchRunsAfterConfirmationAndEndsTurn(t *testing.T) {
	unit := nativeSystemGroupMarchUnit(0, 0)
	state := &battle.State{
		W: 4, H: 1, Units: []*battle.Unit{unit},
		NativeCompositionEventBytes: make([]byte, 4),
		NativeTerrainMoveCodes:      make([]byte, 4),
		NativeFieldEventSlots:       []int{-1, -1, -1, -1},
		NativeFieldEvents:           make([]battle.NativeFieldEvent, 16),
	}
	rows, err := battle.LoadNativeMovementCostRows(assetPath("assets/data/native_movement_cost_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(rows); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		st: state, sc: &battle.Scenario{}, m: &MapData{W: 4, H: 1, TileW: 24, TileH: 24},
		ring: true, ringSel: 1, nativeSystemCursorOverlay: true, curX: 3, curY: 0,
	}
	base := "../../../org_game/炎龍騎士團/FLAME2"
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	g.nativePreparationUI, err = loadNativePreparationUIAssets()
	if err != nil {
		t.Skipf("player-provided original UI assets are absent: %v", err)
	}
	g.nativeClassUI, err = loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeMapVGA = make([]byte, 320*200)
	if !g.activateNativeSystemDirectionOne() {
		t.Fatal("verified outer selector1 route was rejected")
	}
	if unit.X != 0 || unit.Acted || g.nativeSystemEndTurnConfirm {
		t.Fatal("group march mutated before overlay close")
	}
	for present := 0; present < 4; present++ {
		g.markActionOverlayDrawn()
		g.stepActionOverlayLifecycle()
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	g.confirmNativeSystemEndTurn()
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	for g.nativeSystemEndTurnDelay > 0 {
		g.stepNativeSystemEndTurn()
	}
	for g.nativeClassUIJob != nil {
		g.nativeClassUIJob.drawn = true
		g.stepNativeClassUILifecycle(time.Time{})
	}
	if g.walk == nil || unit.Acted {
		t.Fatalf("group march did not defer transaction to walk: walk=%#v acted=%v", g.walk, unit.Acted)
	}
	for g.walk != nil {
		g.stepBattleWalk()
	}
	if !unit.Acted || unit.NativeRecordByte5&0x80 == 0 || unit.X == 0 ||
		!g.aiBusy || g.nativeSystemGroupMarch != nil {
		t.Fatalf("group march finish: unit=%+v ai=%v plan=%#v", unit, g.aiBusy, g.nativeSystemGroupMarch)
	}
}

func TestNativeSystemGroupMarchUnknownEventFailsBeforeOverlayMutation(t *testing.T) {
	unit := nativeSystemGroupMarchUnit(1, 0)
	state := &battle.State{
		W: 2, H: 1, Units: []*battle.Unit{unit},
		NativeCompositionEventBytes: make([]byte, 2),
		NativeTerrainMoveCodes:      make([]byte, 2),
		NativeFieldEventSlots:       []int{0, -1},
		NativeFieldEvents:           make([]battle.NativeFieldEvent, 16),
	}
	state.NativeFieldEvents[0] = battle.NativeFieldEvent{EventID: 82, Selector: 1}
	rows, err := battle.LoadNativeMovementCostRows(assetPath("assets/data/native_movement_cost_rows.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := state.BindNativeMovementCostRows(rows); err != nil {
		t.Fatal(err)
	}
	g := &Game{
		st: state, m: &MapData{W: 2, H: 1, TileW: 24, TileH: 24},
		ring: true, ringSel: 1, nativeSystemCursorOverlay: true, curX: 0, curY: 0,
	}
	base := "../../../org_game/炎龍騎士團/FLAME2"
	t.Setenv("FD2_ORIGINAL_FDOTHER", filepath.Join(base, "FDOTHER.DAT"))
	g.nativePreparationUI, err = loadNativePreparationUIAssets()
	if err != nil {
		t.Skipf("player-provided original UI assets are absent: %v", err)
	}
	g.nativeClassUI, err = loadNativeClassUIAssets()
	if err != nil {
		t.Fatal(err)
	}
	g.nativeMapVGA = make([]byte, 320*200)
	if !g.activateNativeSystemDirectionOne() || g.actionOverlayPhase != "" ||
		!g.ring || !g.nativeSystemCursorOverlay ||
		unit.X != 1 || unit.Acted {
		t.Fatalf("unknown event mutated group march owner: %+v unit=%+v", g, unit)
	}
}

func TestNestedDirectionOneCurrentRuntimeSaveFailsClosedWithoutBaseline(t *testing.T) {
	unit := nativeSystemGroupMarchUnit(0, 0)
	g := &Game{
		st:   &battle.State{Units: []*battle.Unit{unit}},
		ring: true, ringSel: 1,
		nativeSystemCursorOverlay: true,
		nativeSystemNestedOpen:    true,
	}
	if !g.activateNativeSystemDirectionOne() {
		t.Fatal("nested direction1 was not consumed")
	}
	if g.nativeSystemGroupMarch != nil || g.actionOverlayPhase != "" ||
		!g.nativeSystemNestedOpen ||
		g.msg != "原版目前戰況存檔來源或資產不完整，未寫入檔案" {
		t.Fatalf("nested save fell into outer group march: %+v", g)
	}
}

func TestNativeSystemGroupMarchPausesForEvent75AndResumesAfterDialogue(t *testing.T) {
	state, err := battle.Load(assetPath("assets/maps/map28/map28_units.json"))
	if err != nil {
		t.Fatal(err)
	}
	sc, err := battle.LoadScenario(assetPath("assets/scenarios/ch29.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sc.SetupChecked(state); err != nil {
		t.Fatal(err)
	}
	actor := nativeSystemGroupMarchUnit(16, 21)
	actor.NativeRecordByte8 = 9
	actor.HasNativeRecordByte8 = true
	actor.BattleFig = 23
	actor.HasBattleFig = true
	state.Units = []*battle.Unit{actor}
	state.NativeRoundCounter = 8
	step := battle.NativeSystemGroupMarchStep{
		UnitIndex: 0,
		Path:      []battle.Cell{{X: 16, Y: 21}, {X: 15, Y: 21}},
		Events: []battle.NativeSystemGroupMarchEvent{{
			PathIndex: 1, EventID: 75, TextIndex: 1,
		}},
	}
	plan := battle.NativeSystemGroupMarchPlan{
		Destination: battle.Cell{X: 15, Y: 21}, Steps: []battle.NativeSystemGroupMarchStep{step},
	}
	g := &Game{
		st: state, sc: sc, m: &MapData{W: state.W, H: state.H, TileW: 24, TileH: 24},
		nativeSystemGroupMarch: &plan,
	}
	g.startNextNativeSystemGroupMarchStep()
	for tick := 0; tick < 7; tick++ {
		g.stepBattleWalk()
	}
	if g.walk == nil || !g.walk.nativeGroupMarchPaused || g.battleEvent == nil ||
		state.NativeEventState[16] != 0 || actor.Acted {
		t.Fatalf("event75 was not paused before dialogue commit: walk=%#v event=%v state=%v acted=%v", g.walk, g.battleEvent != nil, state.NativeEventState[16:18], actor.Acted)
	}
	for index := 0; index < 5; index++ {
		g.dialog = nil
		g.advanceBattleEvent()
	}
	if g.walk == nil || g.walk.nativeGroupMarchPaused || state.NativeEventState[16] != 4 ||
		state.NativeEventState[17] != 1 {
		t.Fatalf("event75 did not resume after commit: walk=%#v state=%v", g.walk, state.NativeEventState[16:18])
	}
	g.stepBattleWalk()
	if g.walk != nil || g.nativeSystemGroupMarch != nil || !actor.Acted ||
		actor.NativeRecordByte5&0x80 == 0 {
		t.Fatalf("group march did not finish after event75: walk=%#v plan=%#v actor=%+v ai=%v", g.walk, g.nativeSystemGroupMarch, actor, g.aiBusy)
	}
}
