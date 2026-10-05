package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// READY：physical_owner_return_dac_spec。固定槽由正常LOAD進場，再同步一次輸入。
// 完整owner沿用既有work；原版pixels與DAC僅用於合成後比較。
func TestNativePhysicalOwnerMapHandoffFromOracle(t *testing.T) {
	cfg := physicalMapReturnAcceptance{
		battle:    "battle_ch08",
		slotSHA:   "768a561e8a713cd4f7f6fa6f36e553c7330bd60cc122bca03d8654a037d723a0",
		traceSHA:  "e2a043d7a6052c0aa68b8adf8cbcec5de3fd2393818fe815f7925bb64d93bbab",
		entryStep: 1324258027, unitCount: 31,
	}
	t.Helper()
	slot, run, prefix := os.Getenv("FD2_PHYSICAL_OWNER_MAP_SLOT"), os.Getenv("FD2_PHYSICAL_OWNER_MAP_ORIGINAL"), os.Getenv("FD2_PHYSICAL_OWNER_MAP_OUT")
	if slot == "" || run == "" || prefix == "" {
		t.Skip("需要#166固定槽、同時點來源與輸出路徑")
	}
	dacRun := os.Getenv("FD2_PHYSICAL_OWNER_DAC_ORIGINAL")
	fadeInRun := os.Getenv("FD2_PHYSICAL_OWNER_FADE_IN_ORIGINAL")
	if dacRun == "" || fadeInRun == "" {
		t.Skip("需要#176正常caller的64份DAC收據")
	}
	stored, err := os.ReadFile(slot)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(stored)) != cfg.slotSHA {
		t.Fatal("固定槽", err)
	}
	t.Setenv("FD2_TITLE", "1")
	t.Setenv("FD2_MUTE", "1")
	t.Setenv("FD2_CAMPAIGN", canonicalCampaignReference)
	t.Setenv("FD2_NATIVE_SAVE", slot)
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	userDataDirCached = ""
	t.Cleanup(func() { userDataDirCached = "" })
	g := loadGame()
	if g.loadErr != "" || !g.confirmTitleLoadSlot(0) {
		t.Fatal("正常 LOAD", g.loadErr)
	}
	r := &parityReplay{t: t, g: g, battle: cfg.battle}
	r.settleTown()
	for g.campSel != 2 {
		if !g.moveNativeTownSelection(1) {
			t.Fatal("城鎮出口")
		}
	}
	r.enterTownOption(2)
	if !pump(t, g, 600, func() bool { return !g.nativeClassUIBlocksInput() }) || !g.handleNativePreparationInput(nativePreparationInput{enter: true}) {
		t.Fatal("正常出戰 YES")
	}
	if !pump(t, g, 600, func() bool { return g.camp.Node().Type != "preparation" }) {
		t.Fatal("出戰提示未收尾")
	}
	driveStory(t, g, newJourneyTrace(), func() bool { return g.camp.NodeID() == r.battle })
	if !pump(t, g, journeyStoryFrames, func() bool {
		if r.playerHasControl() {
			return true
		}
		if len(g.dialog) > 0 && storyEnterReady(g) {
			g.handleBattleEventDialogueInput(true)
		}
		return false
	}) {
		t.Fatal("正常戰場控制", g.loadErr)
	}

	var entry struct {
		EIP   string
		Step  uint64
		Stack []string
		Units []struct {
			Raw string `json:"raw_hex"`
		}
		Valid      bool `json:"map_runtime_valid"`
		UnitsValid bool `json:"frame_units_valid"`
		View       struct {
			CameraX  int `json:"camera_x"`
			CameraY  int `json:"camera_y"`
			CursorX  int `json:"cursor_x"`
			CursorY  int `json:"cursor_y"`
			VisibleX int `json:"visible_x"`
			VisibleY int `json:"visible_y"`
			Overlay  int `json:"overlay_selector"`
			Aux      int `json:"aux_phase"`
		}
		Runtime struct {
			Globals []struct {
				Address string
				Width   int    `json:"width_bytes"`
				Raw     string `json:"raw_hex"`
				Value   uint32
			}
			DAC string `json:"palette_dac6_hex"`
		} `json:"map_runtime"`
	}
	raw, err := os.ReadFile(filepath.Join(run, "eip-trace.jsonl"))
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != cfg.traceSHA {
		t.Fatal("固定trace", err)
	}
	found := 0
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		var row struct {
			EIP   string
			Step  uint64
			Stack []string
		}
		if json.Unmarshal([]byte(line), &row) != nil {
			t.Fatal("trace JSON")
		}
		if row.EIP == "0x11CAC" && len(row.Stack) > 1 && row.Step == cfg.entryStep && row.Stack[0] == "0x12DD9" {
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				t.Fatal(err)
			}
			found++
		}
	}
	if found != 1 || entry.Step != cfg.entryStep || entry.Stack[1] != "0x0" || !entry.Valid || !entry.UnitsValid || len(entry.Units) != cfg.unitCount {
		t.Fatal("entry不符", found, entry.Step)
	}
	template, err := battle.Load(assetPath("assets/maps/map7/map7_units.json"))
	if err != nil {
		t.Fatal(err)
	}
	loadedUnits := g.st.Units
	units := make([]*battle.Unit, len(entry.Units))
	for i, row := range entry.Units {
		b, err := hex.DecodeString(row.Raw)
		if err != nil || len(b) != 80 {
			t.Fatal("raw record", i, err)
		}
		var record fdsave.PersistentRecord
		copy(record.Raw[:], b)
		v := record.View()
		word := func(offset int) int { return int(binary.LittleEndian.Uint16(b[offset : offset+2])) }
		u := &battle.Unit{
			X: int(b[0]), Y: int(b[1]), Dir: int(b[3]), HP: word(0x40), MaxHP: word(0x42), MP: word(0x44), MaxMP: word(0x46),
			AP: word(0x48), DP: word(0x4a), HIT: word(0x4c), EV: word(0x4e), DX: word(0x3e), MV: int(b[0x3b]), Lv: int(b[0x21]),
			OnField: b[5]&1 == 0, Acted: b[5]&0x80 != 0, NativeRecordByte5: b[5], HasNativeRecordByte5: true,
			NativeRecordByte6: b[6], HasNativeRecordByte6: true, NativeRecordByte8: b[8], HasNativeRecordByte8: true,
			NativeRecordByte34: b[0x34], HasNativeRecordByte34: true, NativeRecordByte35: b[0x35], HasNativeRecordByte35: true,
			NativeRecordByte36: b[0x36], HasNativeRecordByte36: true, NativeRecordWord42: uint16(word(0x42)), HasNativeRecordWord42: true,
			NativeRecordWord46: uint16(word(0x46)), HasNativeRecordWord46: true, NativeTransient: v.Transient,
			MapSelectorKey: int(b[7]), HasMapSelectorKey: true, BattleFig: int(b[7]), HasBattleFig: true,
			NativeRecordRace: v.Race, HasNativeRecordRace: true, NativeRecordClass: v.Class, HasNativeRecordClass: true,
			NativeMapPresentation: battle.NativeMapPresentationState{X: b[0], Y: b[1], Pose: b[3], Motion: b[4]}, HasNativeMapPresentation: true,
			InventorySlots: make([]int, 8), NativeInventoryFlags: make([]int, 8),
		}
		copy(u.NativeCommandMask[:], b[0x1a:0x1f])
		for j := 0; j < 8; j++ {
			u.NativeInventoryFlags[j], u.InventorySlots[j] = int(b[0x0a+2*j]), int(b[0x0b+2*j])
		}
		switch b[6] {
		case 0:
			u.Camp = battle.Enemy
		case 1:
			u.Camp = battle.Ally
		case 2:
			u.Camp = battle.Own
		default:
			t.Fatal("raw camp", b[6])
		}
		u.ClassID, u.Exp = int(b[0x20]), float64(b[0x3c])
		u.Inventory, u.Equipped = make([]int, 8), make([]bool, 8)
		for j := 0; j < 8; j++ {
			u.Inventory[j], u.Equipped[j] = u.InventorySlots[j], b[0x0a+2*j]&0x40 != 0
		}
		for _, source := range template.Units {
			if source.HasBattleFig && source.BattleFig == u.BattleFig {
				u.NativeConstructor = source.NativeConstructor
				u.Name, u.ClsName, u.AtkMin, u.AtkMax = source.Name, source.ClsName, source.AtkMin, source.AtkMax
				break
			}
		}
		for _, loaded := range loadedUnits {
			if ((loaded.HasNativeRecordByte8 && loaded.NativeRecordByte8 == b[8]) || (loaded.HasNativeIdentity && int(loaded.NativeIdentity) == int(b[8]))) && loaded.HasBattleFig && loaded.BattleFig == int(b[7]) {
				u.Name, u.ClsName = loaded.Name, loaded.ClsName
				u.NativeIdentity, u.HasNativeIdentity = loaded.NativeIdentity, loaded.HasNativeIdentity
				break
			}
		}
		units[i] = u
	}
	cache := &fdicon.NativeSelectorCache{}
	if err := battle.MaterializeNativeMapSelectorSlots(units, cache); err != nil {
		t.Fatal(err)
	}
	for i, u := range units {
		b, _ := hex.DecodeString(entry.Units[i].Raw)
		key, err := cache.KeyForSlot(u.MapSelectorSlot)
		if err != nil || u.MapSelectorSlot != int(b[2]) || key != int(b[7]) {
			t.Fatal("selector slot/key", i, err)
		}
		// 快取constructor會初始化pose/motion；同狀態匯入須在其後恢復raw。
		// READY契約：physical_own_map_raw_restore_spec。不可用輸出phase猜補。
		u.Dir = int(b[3])
		u.NativeMapPresentation = battle.NativeMapPresentationState{X: b[0], Y: b[1], Pose: b[3], Motion: b[4]}
		layer, ok := u.NativeUnitLayerEntry()
		if !ok || layer.X != int(b[0]) || layer.Y != int(b[1]) || layer.Slot != int(b[2]) ||
			layer.Pose != int(b[3]) || layer.MotionOffset != int(b[4]) || layer.Flags != b[5] || layer.ForceBase != (b[0x26] != 0) {
			t.Fatal("raw unit layer input", i)
		}
	}

	g.st.Units, g.st.NativeMapSelectorCache = units, cache
	v := entry.View
	if err := g.st.MaterializeNativeMapViewState(battle.NativeMapViewState{CameraX: v.CameraX, CameraY: v.CameraY, CursorX: v.CursorX, CursorY: v.CursorY, VisibleCursorX: v.VisibleX, VisibleCursorY: v.VisibleY}); err != nil {
		t.Fatal(err)
	}
	g.camX, g.camY = float64(v.CameraX*24), float64(v.CameraY*24)
	g.curX, g.curY = v.CursorX, v.CursorY
	if !g.st.MaterializeNativeMapRangeMode(v.Overlay) {
		t.Fatal("overlay")
	}
	fields := map[string]uint32{}
	for _, f := range entry.Runtime.Globals {
		b, err := hex.DecodeString(f.Raw)
		if err != nil || len(b) != f.Width {
			t.Fatal("global width", f.Address)
		}
		var value uint32
		for i, c := range b {
			value |= uint32(c) << uint(8*i)
		}
		if value != f.Value {
			t.Fatal("global bits", f.Address)
		}
		fields[f.Address] = value
	}
	if len(fields) != 16 || fields["0x51A93"] != 0xffffffff {
		t.Fatal("globals")
	}
	signed := func(a string) int { return int(int32(fields[a])) }
	g.st.NativeMapCycleState = fdicon.NativeMapSpriteCycleState{Idle: signed("0x53C0B"), Moving: signed("0x53C07"), LastTimerTick: signed("0x53C0F")}
	g.st.NativeTerrainPhaseState = fdother.NativeTerrainPhaseState{Phase: signed("0x53C1F"), LastTimerTick: signed("0x539F4")}
	g.st.NativeTerrainFlipState = fdicon.NativeBinaryTickState{Value: signed("0x53A40"), LastTimerTick: signed("0x53A00")}
	g.st.NativeUnitPixelShiftState = fdicon.NativeBinaryTickState{Value: signed("0x53A04"), LastTimerTick: signed("0x53A08")}
	g.st.HasNativeMapCycleState, g.st.HasNativeTerrainPhaseState, g.st.HasNativeMapBinaryTimingState = true, true, true
	if !g.st.MaterializeNativeMapHUDState(byte(fields["0x51AAB"]), byte(fields["0x51AAC"]), signed("0x51A0C")) {
		t.Fatal("HUD")
	}
	g.nativeMapDAC, err = hex.DecodeString(entry.Runtime.DAC)
	if err != nil || len(g.nativeMapDAC) != 768 {
		t.Fatal("DAC", err)
	}
	g.nativeFDOTHERPalettePhase, g.nativeFDOTHERPaletteTick = int(fields["0x60002"]), int(fields["0x60000"])
	aux := v.Aux
	g.nativeChapterAuxPhaseOverride = &aux
	frozen := time.Unix(1000, 0)
	g.nativeMapFrozenNow = func() time.Time { return frozen }
	if !g.nativeMapClock.Seed(int(int16(uint16(fields["0x46C"]))), frozen) {
		t.Fatal("BIOS")
	}

	g.sel, g.moved = g.st.Units[0], true
	g.nativeRNGState = 11065
	g.markNativePlayerAttackField()
	report := map[string]any{
		"state": "RUNTIME-E1", "scope": "固定建構槽完整owner與保留工作緩衝；非PLAYER-E2",
		"raw_import_step": entry.Step, "trace_sha256": cfg.traceSHA,
		"clock_method": "原版input BIOS2717→4145；沿既有nativeBIOSTickPeriod前進，不輸入output phase",
	}
	dacTrace, err := os.ReadFile(filepath.Join(fadeInRun, "eip-trace.jsonl"))
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(dacTrace)) != "3e0da879241dfed0d00fecd867a04dab715a72a1384a3dd6fac7d13ed7b1839a" {
		t.Fatal("固定DAC trace", err)
	}
	var originalDACs [][]byte
	var originalInDACs [][]byte
	for _, line := range strings.Split(strings.TrimSpace(string(dacTrace)), "\n") {
		var row struct {
			EIP     string
			Step    uint64
			EBX     string
			Stack   []string
			Runtime struct {
				DAC string `json:"palette_dac6_hex"`
			} `json:"map_runtime"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatal(err)
		}
		if row.EIP == "0x1F510" && row.Step > 1330198110 && row.Step < 1331402171 {
			if row.EBX != fmt.Sprintf("0x%X", len(originalDACs)) || len(row.Stack) < 5 || row.Stack[4] != "0x290AC" {
				t.Fatal("DAC caller或delta順序")
			}
			dac, err := hex.DecodeString(row.Runtime.DAC)
			if err != nil || len(dac) != 768 {
				t.Fatal("DAC shape", err)
			}
			originalDACs = append(originalDACs, dac)
		}
		if row.EIP == "0x1F544" && len(row.Stack) >= 5 && row.Stack[4] == "0x2910F" {
			if row.EBX != fmt.Sprintf("0x%X", 64-len(originalInDACs)) {
				t.Fatal("漸亮delta順序")
			}
			dac, err := hex.DecodeString(row.Runtime.DAC)
			if err != nil || len(dac) != 768 {
				t.Fatal("漸亮DAC shape", err)
			}
			originalInDACs = append(originalInDACs, dac)
		}
	}
	if len(originalDACs) != 64 {
		t.Fatal("DAC rows", len(originalDACs))
	}
	if len(originalInDACs) != 65 {
		t.Fatal("漸亮DAC rows", len(originalInDACs))
	}
	mapPNG, err := os.ReadFile(filepath.Join(fadeInRun, "frames/frame-000000.png"))
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(mapPNG)) != "bd373631baacbc874352c9a01dd40f2928272c477cb3a30c8280576a86115a36" {
		t.Fatal("固定漸亮map PNG", err)
	}
	mapDecoded, err := png.Decode(bytes.NewReader(mapPNG))
	if err != nil {
		t.Fatal(err)
	}
	mapOriginal, ok := mapDecoded.(*image.Paletted)
	if !ok || mapOriginal.Bounds() != image.Rect(0, 0, 320, 200) {
		t.Fatal("漸亮map shape")
	}
	fadePNG, err := os.ReadFile(filepath.Join(dacRun, "frames/frame-000001.png"))
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(fadePNG)) != "14d1ddf376a01d4319e53a949b14a1c03fe34133cf79ae517cee24247e58a4f7" {
		t.Fatal("固定fade PNG", err)
	}
	fadeDecoded, err := png.Decode(bytes.NewReader(fadePNG))
	if err != nil {
		t.Fatal(err)
	}
	fadeOriginal, ok := fadeDecoded.(*image.Paletted)
	if !ok || fadeOriginal.Bounds() != image.Rect(0, 0, 320, 200) {
		t.Fatal("fade PNG shape")
	}
	writeReport := func() {
		encoded, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(prefix+".json", append(encoded, '\n'), 0644); err != nil {
			t.Fatal(err)
		}
	}
	checkMap := func(index int, expectedSHA string, label string, expectedPalette ...color.Palette) (int, int, int) {
		palette, err := fdother.VGAPaletteFromDAC(g.nativeMapDAC)
		if err != nil {
			t.Fatal(err)
		}
		pic := image.NewPaletted(image.Rect(0, 0, 320, 200), palette)
		copy(pic.Pix, g.nativeMapVGA)
		originalPNG, err := os.ReadFile(filepath.Join(run, "frames", fmt.Sprintf("frame-%06d.png", index)))
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(originalPNG)) != expectedSHA {
			t.Fatal("original PNG", err)
		}
		decoded, err := png.Decode(bytes.NewReader(originalPNG))
		if err != nil {
			t.Fatal(err)
		}
		original, ok := decoded.(*image.Paletted)
		if !ok || original.Bounds() != pic.Bounds() || len(original.Palette) != 256 {
			t.Fatal("original shape")
		}
		if len(expectedPalette) > 0 {
			original.Palette = expectedPalette[0]
		} // 寫後trace的完整DAC；不改原PNG。
		idx, rgb, pal := 0, 0, 0
		for y := 0; y < 200; y++ {
			for x := 0; x < 320; x++ {
				if original.ColorIndexAt(x, y) != pic.ColorIndexAt(x, y) {
					idx++
				}
				a, b, c, d := original.At(x, y).RGBA()
				e, f, h, i := pic.At(x, y).RGBA()
				if a != e || b != f || c != h || d != i {
					rgb++
				}
			}
		}
		for n := 0; n < 256; n++ {
			a, b, c, d := original.Palette[n].RGBA()
			e, f, h, i := pic.Palette[n].RGBA()
			if a != e || b != f || c != h || d != i {
				pal++
			}
		}
		output, err := os.Create(prefix + "-" + label + ".png")
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(output, pic); err != nil {
			t.Fatal(err)
		}
		if err := output.Close(); err != nil {
			t.Fatal(err)
		}
		report[label] = map[string]any{"indexed_difference_pixels": idx, "rgb_difference_pixels": rgb, "palette_difference_entries": pal}
		writeReport()
		return idx, rgb, pal
	}
	if err := g.composeNativeMapFrame(); err != nil {
		report["initial_error"] = err.Error()
		writeReport()
		t.Fatal("initial map", err)
	}
	idx, rgb, pal := checkMap(91, "2dc391cbe92df441cd13650d5eb38759110373e53c772ac39044b679b371bcc8", "pre-map")
	if idx != 0 || rgb != 0 || pal != 0 {
		t.Fatalf("pre map differs: index%d RGB%d palette%d", idx, rgb, pal)
	}
	work := g.nativeMapWork
	report["before_work_bytes"] = len(work)
	report["before_work_sha256"] = fmt.Sprintf("%x", sha256.Sum256(work))
	g.confirm()
	if g.atk == nil || g.atk.nativeScene == nil || g.atk.nativeScene.body == nil || g.loadErr != "" {
		report["confirm_error"] = g.loadErr
		report["confirm_message"] = g.msg
		writeReport()
		t.Fatal("confirm", g.loadErr, g.msg)
	}
	report["resolved_rng"] = g.nativeRNGState
	report["resolved_hp"] = []int{g.st.Units[0].HP, g.st.Units[11].HP}
	if g.nativeRNGState != 44258 || g.st.Units[0].HP != 341 || g.st.Units[11].HP != 55 {
		writeReport()
		t.Fatal("fixed presentation inputs changed")
	}
	if &g.nativeMapWork[0] != &work[0] || fmt.Sprintf("%x", sha256.Sum256(g.nativeMapWork)) != report["before_work_sha256"] {
		t.Fatal("preflight published candidate work")
	}
	body := g.atk.nativeScene.body
	voice := &physicalOwnerReturnVoice{}
	g.nativePhysicalVoice = voice
	after := g.atk.after
	returned := false
	afterCalls := 0
	g.atk.after = func() {
		returned = true
		afterCalls++
		if g.nativePaletteRamp != nil || !bytes.Equal(g.nativeMapDAC, originalInDACs[64]) || voice.calls != 1 {
			t.Fatal("after早於最後DAC交接")
		}
		report["retained_work_storage"] = &work[0] == &g.nativeMapWork[0]
		report["return_work_sha256"] = fmt.Sprintf("%x", sha256.Sum256(g.nativeMapWork))
		palette, err := fdother.VGAPaletteFromDAC(originalInDACs[64])
		if err != nil {
			t.Fatal(err)
		}
		idx, rgb, pal = checkMap(126, "bd373631baacbc874352c9a01dd40f2928272c477cb3a30c8280576a86115a36", "return-map", palette)
		report["return_cycles"] = g.st.NativeMapCycleState
		report["return_palette_phase"] = g.nativeFDOTHERPalettePhase
		report["return_palette_tick"] = g.nativeFDOTHERPaletteTick
		writeReport()
		if after != nil {
			after()
		}
	}
	// clock只於正式返回前推至原版輸入刻度，不重設globals或state。
	frozen = frozen.Add(time.Duration(4145-2717) * nativeBIOSTickPeriod)
	screen := ebiten.NewImage(640, 400)
	defer screen.Dispose()
	fadeSteps := 0
	fadeInSteps := 0
	checkBlackReturn := func() {
		i, r, p := checkMap(126, "bd373631baacbc874352c9a01dd40f2928272c477cb3a30c8280576a86115a36", "black-return-map")
		if i != 0 || r != 0 || p != 0 {
			t.Fatal("黑map中間時點不同", i, r, p)
		}
	}
	if os.Getenv("FD2_PHYSICAL_OWNER_GPU") == "1" {
		probe := &physicalOwnerDACGPUProbe{g: g, pixels: fadeOriginal.Pix, dacs: originalDACs, mapPixels: mapOriginal.Pix, inDACs: originalInDACs, voice: voice, onMap: checkBlackReturn}
		ebiten.SetWindowSize(640, 400)
		if err := ebiten.RunGame(probe); err != nil {
			t.Fatal(err)
		}
		if probe.err != nil {
			t.Fatal(probe.err)
		}
		for _, seen := range probe.seen {
			if seen {
				fadeSteps++
			}
		}
		for _, seen := range probe.seenIn {
			if seen {
				fadeInSteps++
			}
		}
		report["gpu_fade_steps"] = fadeSteps
		report["gpu_fade_in_steps"] = fadeInSteps
	} else {
		for tick := 0; tick < 3000 && g.atk != nil; tick++ {
			if job := g.nativePaletteRamp; job != nil {
				expectedDACs, pixels, count := originalDACs, fadeOriginal.Pix, fadeSteps
				isIn := len(job.deltas) == 65
				if isIn {
					if job.step == 0 {
						checkBlackReturn()
					}
					expectedDACs, pixels, count = originalInDACs, mapOriginal.Pix, fadeInSteps
					if fadeSteps != 64 || voice.calls != 1 {
						t.Fatal("map漸亮早於漸暗或停音")
					}
				} else if voice.calls != 0 {
					t.Fatal("漸暗時已停音")
				}
				if job.step != count || job.step >= len(expectedDACs) {
					t.Fatal("fade順序", job.step, fadeSteps)
				}
				if !bytes.Equal(job.vga, pixels) || !bytes.Equal(job.dac, expectedDACs[job.step]) {
					t.Fatal("整張fade索引或DAC差異", job.step)
				}
				palette, err := fdother.VGAPaletteFromDAC(expectedDACs[job.step])
				if err != nil {
					t.Fatal(err)
				}
				for n := 0; n < 256; n++ {
					ar, ag, ab, aa := job.palette[n].RGBA()
					br, bg, bb, ba := palette[n].RGBA()
					if ar != br || ag != bg || ab != bb || aa != ba {
						t.Fatal("fade RGB差異", job.step, n)
					}
				}
				g.stepNativePaletteRamp()
				if g.nativePaletteRamp != job || job.step != count || returned {
					t.Fatal("未Draw便漸變或續行")
				}
				if isIn {
					fadeInSteps++
				} else {
					fadeSteps++
				}
			}
			g.Draw(screen)
			g.stepNativePaletteRamp()
			if err := g.stepAttackPresentationTick(); err != nil && err != errAttackPresentationYield {
				t.Fatal(err)
			}
		}
	}
	report["owner_finished"] = g.atk == nil
	report["after_called"] = returned
	report["body_jobs"] = len(body.jobs)
	report["fade_steps"] = fadeSteps
	report["fade_in_steps"] = fadeInSteps
	report["sound_stop_calls"] = voice.calls
	report["after_calls"] = afterCalls
	report["load_error"] = g.loadErr
	writeReport()
	if !returned || g.atk != nil || g.loadErr != "" || fadeSteps != 64 || fadeInSteps != 65 || afterCalls != 1 || voice.calls != 1 {
		t.Fatal("owner", returned, g.loadErr)
	}
	if idx != 0 || rgb != 0 || pal != 0 {
		t.Fatalf("owner return differs: index%d RGB%d palette%d", idx, rgb, pal)
	}
	t.Log("完整owner漸暗64＋漸亮65份DAC、最後body與前後地圖indexed／RGB皆零差異；保留work及Draw閘門")
}

// 正式Draw的GPU收據使用同一完整Game owner，不建立第二個演出入口。
type physicalOwnerDACGPUProbe struct {
	g         *Game
	pixels    []byte
	dacs      [][]byte
	mapPixels []byte
	inDACs    [][]byte
	voice     *physicalOwnerReturnVoice
	onMap     func()
	seen      [64]bool
	seenIn    [65]bool
	updates   int
	err       error
}

func (p *physicalOwnerDACGPUProbe) Layout(_, _ int) (int, int) { return 640, 400 }

func (p *physicalOwnerDACGPUProbe) Update() error {
	p.updates++
	if p.err != nil || p.g.atk == nil {
		return ebiten.Termination
	}
	if p.updates > 3000 {
		return fmt.Errorf("完整owner GPU逾時")
	}
	p.g.stepNativePaletteRamp()
	if err := p.g.stepAttackPresentationTick(); err != nil && err != errAttackPresentationYield {
		return err
	}
	return nil
}

func (p *physicalOwnerDACGPUProbe) Draw(screen *ebiten.Image) {
	if p.err != nil || p.g.atk == nil {
		return
	}
	job := p.g.nativePaletteRamp
	p.g.Draw(screen)
	if job == nil {
		return
	}
	step := job.step
	pixels, dacs := p.pixels, p.dacs
	isIn := len(job.deltas) == 65
	if isIn {
		if step == 0 && !p.seenIn[0] && p.onMap != nil {
			p.onMap()
		}
		pixels, dacs = p.mapPixels, p.inDACs
		if p.voice.calls != 1 {
			p.err = fmt.Errorf("漸亮前未停音")
			return
		}
	} else if p.voice.calls != 0 {
		p.err = fmt.Errorf("漸暗便停音")
		return
	}
	if step < 0 || step >= len(dacs) || !bytes.Equal(job.vga, pixels) || !bytes.Equal(job.dac, dacs[step]) {
		p.err = fmt.Errorf("GPU漸變索引或DAC不同，step%d", step)
		return
	}
	palette, err := fdother.VGAPaletteFromDAC(dacs[step])
	if err != nil {
		p.err = err
		return
	}
	rgba := make([]byte, 640*400*4)
	screen.ReadPixels(rgba)
	for y := 0; y < 400; y++ {
		for x := 0; x < 640; x++ {
			r, g, b, a := palette[pixels[(y/2)*320+x/2]].RGBA()
			i := (y*640 + x) * 4
			if rgba[i] != byte(r>>8) || rgba[i+1] != byte(g>>8) || rgba[i+2] != byte(b>>8) || rgba[i+3] != byte(a>>8) {
				p.err = fmt.Errorf("GPU整張RGB不同，step%d x%d y%d", step, x, y)
				return
			}
		}
	}
	if isIn {
		p.seenIn[step] = true
	} else {
		p.seen[step] = true
	}
}

type physicalOwnerReturnVoice struct{ calls int }

func (v *physicalOwnerReturnVoice) IsPlaying() bool { return v.calls == 0 }
func (v *physicalOwnerReturnVoice) Close() error    { v.calls++; return nil }
