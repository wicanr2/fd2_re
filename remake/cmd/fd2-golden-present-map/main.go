// fd2-golden-present-map 輸出 internal/indexedmap 的黃金差分資料，供 Rust 改寫 remake-rs/crates/fd2-campaign
// 逐位元組比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md，R3 present 組）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 indexedmap 的公開 API，不改任何遊戲行為。
//   - 真實輸入：remake/assets/maps/mapN/map.json（33 張地圖的 tiles、native_tile_blit_modes、
//     native_terrain_control），分離素材包的 FDSHAP 地形、FDICON 單位、FDOTHER 範圍圖示、LUT、HUD、
//     spawn-intro、指令尾段 LMI、章節輔助表面與山景。檔案清單排序後連同大小與 SHA-256 寫進 JSON。
//   - 局面：每張地圖以固定種子產生數個局面（鏡頭、翻轉、週期、單位層、前景層、選擇器快取、範圍模式、
//     游標、HUD 輸入），各 op 從同一份以 xorshift32 填滿的工作／VGA 緩衝開始（只記 salt）。
//   - 結果：每個 op 之後工作緩衝與 VGA 的 SHA-256、觀察者各階段的 SHA-256、錯誤文字與回傳值。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/indexedmap"
)

type obj = map[string]any

type caseJSON struct {
	Index  int    `json:"index"`
	Op     string `json:"op"`
	Input  obj    `json:"input"`
	Result obj    `json:"result"`
}

const (
	workSize = indexedmap.NativeUnitPresentWorkSize
	vgaSize  = indexedmap.NativeMapVGASize
)

type mapAssets struct {
	index      int
	w, h       int
	tiles      []int
	modes      []byte
	terrain    *fdicon.Bank
	controls   []byte
	chapterAux *fdother.NativeChapterAuxSurface
	parallax   *fdother.NativeMapParallaxSurface
}

type exporter struct {
	rng       *rand.Rand
	assets    string
	pack      string
	cases     []caseJSON
	files     []obj
	units     *fdicon.Bank
	rangeBank *fdicon.Bank
	luts      [][]byte
	frames    indexedmap.NativeMapHUDFrames
	spawn     []fdother.LMI1Entry
	digits    []fdother.LMI1Entry
	fdother6  []fdother.LMI1Entry
	maps      []*mapAssets
	keys      []int
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func errv(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

// fill 以 xorshift32 產生 n 位元組；Rust 端同一個公式。
func fill(n, salt int) []byte {
	b := make([]byte, n)
	x := uint32(salt*2654435761 + 1)
	for i := range b {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		b[i] = byte(x)
	}
	return b
}

func (e *exporter) add(op string, input, result obj) {
	e.cases = append(e.cases, caseJSON{Index: len(e.cases), Op: op, Input: input, Result: result})
}

func (e *exporter) intn(n int) int          { return e.rng.IntN(n) }
func (e *exporter) rangeInt(lo, hi int) int { return lo + e.rng.IntN(hi-lo+1) }
func (e *exporter) chance(p int) bool       { return e.rng.IntN(100) < p }

func (e *exporter) record(label, path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	e.files = append(e.files, obj{"path": label, "size": len(raw), "sha256": digest(raw)})
}

func (e *exporter) recordTree(prefix, root, rel string) {
	var paths []string
	err := filepath.WalkDir(filepath.Join(root, rel), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil || len(paths) == 0 {
		panic(fmt.Sprintf("找不到 %s/%s", root, rel))
	}
	sort.Strings(paths)
	for _, path := range paths {
		r, err := filepath.Rel(root, path)
		if err != nil {
			panic(err)
		}
		e.record(prefix+"/"+filepath.ToSlash(r), path)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func (e *exporter) load() {
	p := func(rel string) string { return filepath.Join(e.pack, rel) }
	e.units = must(fdicon.LoadSeparatedBank(p("sprites/fdicon")))
	e.rangeBank = must(fdother.LoadSeparatedRangeOverlayBank(p("sprites/fdother_001_range_overlay")))
	e.luts = must(fdother.LoadSeparatedLUTBank(p("palette")))
	e.frames = must(indexedmap.LoadSeparatedNativeMapHUDFrames(p("ui")))
	e.spawn = must(fdother.LoadSeparatedLMI1Bank(p("animations/fdother_009_spawn_intro"), fdother.NativeSpawnIntroFrameResource))
	e.digits = must(fdother.LoadSeparatedLMI1Bank(p("ui/fdother_005_lmi1_opaque"), fdother.NativeCommandHealTailDigitResource))
	e.fdother6 = must(fdother.LoadSeparatedLMI1Bank(p("effects/fdother_006_lmi1_opaque"), fdother.NativeCommandHealTailEffectResource))
	for key := 0; key < fdicon.NativeSourceGroupCount; key++ {
		if fdicon.NativeSourceGroupHasPixels(key) {
			e.keys = append(e.keys, key)
		}
	}
	for m := 0; m < fdicon.SeparatedFDSHAPMapCount; m++ {
		path := filepath.Join(e.assets, "maps", fmt.Sprintf("map%d", m), "map.json")
		e.record(fmt.Sprintf("maps/map%d/map.json", m), path)
		raw := must(os.ReadFile(path))
		var field struct {
			W       int    `json:"w"`
			H       int    `json:"h"`
			Tiles   []int  `json:"tiles"`
			Modes   []int  `json:"native_tile_blit_modes"`
			Control []byte `json:"-"`
		}
		if err := json.Unmarshal(raw, &field); err != nil {
			panic(err)
		}
		modes := make([]byte, len(field.Modes))
		for i, v := range field.Modes {
			modes[i] = byte(v)
		}
		terrain, controls := must2(fdicon.LoadSeparatedFDSHAPBank(p("tilesets/fdshap"), m))
		a := &mapAssets{index: m, w: field.W, h: field.H, tiles: field.Tiles, modes: modes, terrain: terrain, controls: controls}
		if contract, ok := fdother.NativeChapterAuxSurfaceFor(m); ok {
			a.chapterAux = must(fdother.LoadSeparatedChapterAuxSurfaceResource(p("surfaces"), contract))
		}
		if m == 17 {
			a.parallax = must(fdother.LoadSeparatedNativeMapParallax(p("surfaces")))
		}
		e.maps = append(e.maps, a)
	}
}

func must2[A, B any](a A, b B, err error) (A, B) {
	if err != nil {
		panic(err)
	}
	return a, b
}

// ---- 局面 ----

type scenario struct {
	m          *mapAssets
	cells      []fdicon.NativeTerrainCell
	cache      *fdicon.NativeSelectorCache
	keys       []int
	frame      indexedmap.FrameInput
	hud        indexedmap.NativeMapHUDInput
	lutIndex   int
	json       obj
	workSalt   int
	vgaSalt    int
	useAux     bool
	useParalax bool
}

func unitJSON(u fdicon.NativeUnitLayerEntry) obj {
	return obj{"x": u.X, "y": u.Y, "slot": u.Slot, "pose": u.Pose, "motion": u.MotionOffset, "flags": u.Flags,
		"force": u.ForceBase, "inactive": u.Inactive}
}

func fgJSON(u fdicon.NativeForegroundLayerEntry) obj {
	return obj{"x": u.X, "y": u.Y, "pose": u.Pose, "motion": u.MotionOffset, "inactive": u.Inactive,
		"unit7": u.Unit7, "race": u.Race, "class": u.Class}
}

func (e *exporter) newScenario(m *mapAssets, salt int) *scenario {
	cells, err := indexedmap.BuildNativeTerrainCells(m.tiles, m.modes)
	if err != nil {
		panic(err)
	}
	s := &scenario{m: m, cells: cells, workSalt: salt*2 + 1, vgaSalt: salt*2 + 2}
	nkeys := 1 + e.intn(6)
	cache := &fdicon.NativeSelectorCache{}
	for i := 0; i < nkeys; i++ {
		key := e.keys[e.intn(len(e.keys))]
		if _, err := cache.SlotFor(key); err != nil {
			panic(err)
		}
	}
	s.cache = cache
	for _, k := range cache.Keys() {
		s.keys = append(s.keys, int(k))
	}
	camX := 0
	if m.w > 13 {
		camX = e.intn(m.w - 13 + 1)
	}
	camY := 0
	if m.h > 8 {
		camY = e.intn(m.h - 8 + 1)
	}
	phase := e.intn(20)
	s.lutIndex = must(fdother.NativeTerrainLUTIndex(phase))
	var units []fdicon.NativeUnitLayerEntry
	var fg []fdicon.NativeForegroundLayerEntry
	for i := 0; i < e.intn(9); i++ {
		x := camX + e.rangeInt(-1, 12)
		y := camY + e.rangeInt(-1, 8)
		slot := e.intn(len(s.keys))
		if e.chance(3) {
			slot = len(s.keys) + 1
		}
		pose := e.intn(4)
		units = append(units, fdicon.NativeUnitLayerEntry{X: x, Y: y, Slot: slot, Pose: pose,
			MotionOffset: e.intn(6), Flags: byte(e.intn(256)), ForceBase: e.chance(30), Inactive: e.chance(15)})
		fg = append(fg, fdicon.NativeForegroundLayerEntry{X: x, Y: y, Pose: byte(pose), MotionOffset: e.intn(6),
			Inactive: e.chance(15), Unit7: byte(e.intn(256)), Race: byte(e.intn(4)), Class: byte(e.intn(30))})
	}
	rangeMode := e.intn(7)
	curX, curY := camX+e.intn(13), camY+e.intn(8)
	if e.chance(5) {
		curX = m.w + 2
	}
	s.useAux = m.chapterAux != nil && e.chance(70)
	s.useParalax = m.parallax != nil && e.chance(80)
	s.frame = indexedmap.FrameInput{
		TerrainBank: m.terrain, RangeBank: e.rangeBank, UnitBank: e.units, ForegroundBank: m.terrain,
		SelectorCache: cache, Cells: cells, Controls: m.controls, LUT: e.luts[s.lutIndex],
		MapWidth: m.w, CameraX: camX, CameraY: camY,
		Flip: e.intn(2), TerrainCycle: e.intn(3), IdleCycle: e.intn(3), MovingCycle: e.intn(3), PixelShift: e.intn(2),
		RangeMode: rangeMode, CursorX: curX, CursorY: curY, Units: units, ForegroundUnits: fg,
		ParallaxScrollX: e.rangeInt(-6, 6), ParallaxScrollY: e.rangeInt(-6, 6), ChapterAuxPhase: e.intn(16),
	}
	if s.useAux {
		s.frame.ChapterAux = m.chapterAux
	}
	if s.useParalax {
		s.frame.Parallax = m.parallax
	}
	tile := 0
	control := byte(0)
	if curX >= 0 && curX < m.w && curY >= 0 && curY < m.h {
		tile = int(cells[curY*m.w+curX].Tile & 0x3ff)
		if tile*4+1 < len(m.controls) {
			control = m.controls[tile*4+1]
		}
	}
	if e.chance(5) {
		tile = 0x400
	}
	if e.chance(5) {
		control = 6
	}
	s.hud = indexedmap.NativeMapHUDInput{DisplayGateA: e.chance(90), DisplayGateB: e.chance(90),
		AnchorX: []int{1, 0xf2, 136, 251, 252}[e.intn(5)], TerrainDescriptor: tile, TerrainControl: control}
	if e.chance(70) && len(s.keys) > 0 {
		cur := uint16(e.intn(1200))
		max := cur
		if e.chance(50) {
			max = uint16(e.intn(1200))
		}
		s.hud.OptionalUnit = &indexedmap.NativeMapHUDOptionalUnit{SelectorSlot: e.intn(len(s.keys)), RawState: e.intn(5), Current: cur, Maximum: max}
	}
	uj := make([]any, len(units))
	for i, u := range units {
		uj[i] = unitJSON(u)
	}
	fj := make([]any, len(fg))
	for i, u := range fg {
		fj[i] = fgJSON(u)
	}
	hud := obj{"gate_a": s.hud.DisplayGateA, "gate_b": s.hud.DisplayGateB, "anchor": s.hud.AnchorX,
		"descriptor": s.hud.TerrainDescriptor, "control": s.hud.TerrainControl, "unit": nil}
	if u := s.hud.OptionalUnit; u != nil {
		hud["unit"] = obj{"slot": u.SelectorSlot, "state": u.RawState, "current": u.Current, "maximum": u.Maximum}
	}
	f := s.frame
	s.json = obj{"map": m.index, "keys": s.keys, "lut": s.lutIndex, "camera_x": f.CameraX, "camera_y": f.CameraY,
		"flip": f.Flip, "terrain_cycle": f.TerrainCycle, "idle": f.IdleCycle, "moving": f.MovingCycle,
		"pixel_shift": f.PixelShift, "range_mode": f.RangeMode, "cursor_x": f.CursorX, "cursor_y": f.CursorY,
		"units": uj, "fg": fj, "scroll_x": f.ParallaxScrollX, "scroll_y": f.ParallaxScrollY,
		"aux_phase": f.ChapterAuxPhase, "aux": s.useAux, "parallax": s.useParalax, "hud": hud,
		"work_salt": s.workSalt, "vga_salt": s.vgaSalt}
	return s
}

func (s *scenario) transition() indexedmap.NativeTransitionFrameInput {
	f := s.frame
	return indexedmap.NativeTransitionFrameInput{
		Parallax: f.Parallax, ParallaxScrollX: f.ParallaxScrollX, ParallaxScrollY: f.ParallaxScrollY,
		TerrainBank: f.TerrainBank, UnitBank: f.UnitBank, ForegroundBank: f.ForegroundBank, SelectorCache: f.SelectorCache,
		Cells: append([]fdicon.NativeTerrainCell(nil), s.cells...), Controls: f.Controls, TerrainLUT: f.LUT,
		MapWidth: f.MapWidth, CameraX: f.CameraX, CameraY: f.CameraY, Flip: f.Flip, TerrainCycle: f.TerrainCycle,
		IdleCycle: f.IdleCycle, MovingCycle: f.MovingCycle, PixelShift: f.PixelShift,
		Units: f.Units, ForegroundUnits: f.ForegroundUnits,
	}
}

func (s *scenario) buffers() ([]byte, []byte) {
	return fill(workSize, s.workSalt), fill(vgaSize, s.vgaSalt)
}

func cellsDigest(cells []fdicon.NativeTerrainCell) string {
	b := make([]byte, 0, len(cells)*3)
	for _, c := range cells {
		b = append(b, byte(c.Tile), byte(c.Tile>>8), c.BlitMode)
	}
	return digest(b)
}

type stageLog struct{ stages []any }

func (l *stageLog) observer(stage indexedmap.FrameStage, work, vga []byte) error {
	l.stages = append(l.stages, []any{int(stage), digest(work), digest(vga)})
	return nil
}

func (e *exporter) opFrame(s *scenario) {
	work, vga := s.buffers()
	in := indexedmap.NativeFrameInput{Frame: s.frame, HUD: s.hud, Frames: e.frames, HUDTerrain: s.m.terrain, HUDUnits: e.units, HUDCache: s.cache}
	in.Frame.Cells = append([]fdicon.NativeTerrainCell(nil), s.cells...)
	log := &stageLog{}
	err := indexedmap.ComposeNativeFrameObserved(work, vga, in, log.observer)
	e.add("frame", obj{"scenario": s.json}, obj{"error": errv(err), "work": digest(work), "vga": digest(vga),
		"cells": cellsDigest(in.Frame.Cells), "stages": log.stages})
}

func (e *exporter) opStep(s *scenario) {
	work, vga := s.buffers()
	log := &stageLog{}
	err := indexedmap.ComposeNativeStepFrameObserved(work, vga, s.frame, log.observer)
	e.add("step", obj{"scenario": s.json}, obj{"error": errv(err), "work": digest(work), "vga": digest(vga), "stages": log.stages})
	unitY, motion := s.frame.CameraY+e.rangeInt(-1, 8), e.rangeInt(0, 7)
	in := s.frame
	in.Parallax, in.ChapterAux = nil, nil
	if e.chance(10) {
		in = s.frame
	}
	work, vga = s.buffers()
	log = &stageLog{}
	err = indexedmap.ComposeNativeUpStepFrameObserved(work, vga, in, unitY, motion, log.observer)
	tb, tcy, rows, vb, lerr := indexedmap.NativeUpStepLayout(in, unitY, motion)
	e.add("upstep", obj{"scenario": s.json, "unit_y": unitY, "motion": motion, "strip_layers": in.Parallax == nil && in.ChapterAux == nil},
		obj{"error": errv(err), "work": digest(work), "vga": digest(vga), "stages": log.stages,
			"layout": []any{tb, tcy, rows, vb}, "layout_error": errv(lerr)})
}

func (e *exporter) opTransition(s *scenario) {
	work, vga := s.buffers()
	cx, cy, radius := e.rangeInt(0, 312), e.rangeInt(0, 192), e.rangeInt(0, 200)
	start, end := e.rangeInt(0, 100), e.rangeInt(100, 192)
	lut := 1 + e.intn(9)
	pass, perr := fdother.BuildNativeIndexedTransitionPass(cx, cy, radius, start, end)
	res := obj{"pass_error": errv(perr)}
	if perr == nil {
		err := indexedmap.ComposeNativeTransitionFrame(work, vga, s.transition(), pass, e.luts[lut])
		res["error"], res["work"], res["vga"] = errv(err), digest(work), digest(vga)
	}
	e.add("transition", obj{"scenario": s.json, "pass": []int{cx, cy, radius, start, end}, "lut2": lut}, res)
}

func (e *exporter) opUnitPresent(s *scenario) {
	in := s.transition()
	var steps []any
	step := func(name string, err error, bufs ...[]byte) {
		row := []any{name, errv(err)}
		for _, b := range bufs {
			row = append(row, digest(b))
		}
		steps = append(steps, row)
	}
	work, vga := s.buffers()
	snapshot := fill(workSize, s.workSalt+1000)
	step("terrain_snapshot", indexedmap.ComposeNativeUnitPresentTerrainSnapshot(snapshot, in), snapshot)
	staging := fill(workSize, s.workSalt+2000)
	step("24b4d", indexedmap.ComposeNative24B4DStaging(staging, in), staging)
	redraw := fill(workSize, s.workSalt+3000)
	step("redraw", indexedmap.RedrawNativeUnitPresentObjects(redraw, in), redraw)
	mapX, mapY := in.CameraX+e.rangeInt(-1, 13), in.CameraY+e.rangeInt(-1, 8)
	entry := 0x72 + e.intn(11)
	step("intro", indexedmap.ComposeNativeUnitPresentIntroFrame(work, vga, snapshot, in, e.fdother6[entry], mapX, mapY), work, vga)
	lutSnap := fill(workSize, s.workSalt+4000)
	step("lut_snapshot", indexedmap.ComposeNativeUnitPresentLUTSnapshot(lutSnap, snapshot, in, e.fdother6[0x7c], mapX, mapY), lutSnap)
	raw9, rawD := mapX-in.CameraX, mapY-in.CameraY
	frames, ferr := fdother.NativeUnitPresentLUTFrames(raw9, rawD)
	step("lut_frames", ferr)
	lutIdx := 1 + e.intn(9)
	frameIdx := -1
	if ferr == nil {
		frameIdx = e.intn(len(frames))
		step("lut_frame", indexedmap.ComposeNativeUnitPresentLUTFrame(work, vga, lutSnap, e.luts[lutIdx], in, frames[frameIdx]), work, vga)
	}
	var rows []any
	bridgeLUT := 1 + e.intn(9)
	err := indexedmap.ComposeNativeUnitPresentStripBridge(work, vga, lutSnap, e.luts[bridgeLUT], in, mapX, mapY, raw9, rawD,
		func(row int) error { rows = append(rows, row); return nil })
	step("bridge", err, work, vga)
	layout := indexedmap.NativeUnitPresentStripLayoutFor(mapX, mapY, in.CameraX, in.CameraY)
	e.add("unit_present", obj{"scenario": s.json, "map_x": mapX, "map_y": mapY, "entry": entry, "lut": lutIdx,
		"frame": frameIdx, "bridge_lut": bridgeLUT},
		obj{"steps": steps, "rows": rows, "layout": []any{layout.WorkOffset, layout.VGAOffset, layout.Rows}})
}

func (e *exporter) opCommand10(s *scenario) {
	work, vga := s.buffers()
	xo, yo, inc := e.rangeInt(-0x2000, 0x2000), e.rangeInt(-0x2000, 0x2000), e.pickInc()
	err := indexedmap.ComposeNativeCommand10To12Surface(work, vga, s.transition(), xo, yo, inc)
	e.add("command10", obj{"scenario": s.json, "x": xo, "y": yo, "inc": inc}, obj{"error": errv(err), "work": digest(work), "vga": digest(vga)})
}

func (e *exporter) pickInc() int {
	return []int{128, 96, 160, 64, 200, 1, 0, -5}[e.intn(8)]
}

func (e *exporter) healTargets(s *scenario) ([]indexedmap.NativeCommandHealTailTarget, []any) {
	var targets []indexedmap.NativeCommandHealTailTarget
	var tj []any
	for i, u := range s.frame.Units {
		if e.chance(60) {
			t := indexedmap.NativeCommandHealTailTarget{RecordIndex: i, X: u.X, Y: u.Y, SelectorSlot: u.Slot}
			targets = append(targets, t)
			tj = append(tj, []int{t.RecordIndex, t.X, t.Y, t.SelectorSlot})
		}
	}
	return targets, tj
}

func layersJSON(layers []indexedmap.NativeCommandTailLayer) []any {
	out := make([]any, len(layers))
	for i, l := range layers {
		out[i] = []int{l.Resource, l.Entry, l.X, l.Y}
	}
	return out
}

func (e *exporter) opHeal(s *scenario) {
	targets, tj := e.healTargets(s)
	cam := []int{s.frame.CameraX, s.frame.CameraY}
	snapshot := fill(workSize, s.workSalt+5000)
	work, vga := s.buffers()
	entry := 0x39 + e.intn(7)
	layers, err := indexedmap.ComposeNativeCommandHealEffectFrameWithSources(work, vga, snapshot, e.fdother6[entry], targets, cam[0], cam[1], 6, entry)
	e.add("heal_effect", obj{"scenario": s.json, "targets": tj, "entry": entry},
		obj{"error": errv(err), "work": digest(work), "vga": digest(vga), "layers": layersJSON(layers)})
	work, vga = s.buffers()
	cycle, idx := e.rangeInt(0, 4), byte(e.intn(256))
	err = indexedmap.ComposeNativeCommandHealMaskFrame(work, vga, snapshot, e.units, s.cache, targets, cam[0], cam[1], cycle, idx)
	e.add("heal_mask", obj{"scenario": s.json, "targets": tj, "cycle": cycle, "index": idx},
		obj{"error": errv(err), "work": digest(work), "vga": digest(vga)})
	work, vga = s.buffers()
	idle, blend, base := e.rangeInt(0, 4), e.rangeInt(0, 8), byte(e.intn(256))
	err = indexedmap.ComposeNativeAIItemDamageBlendFrame(work, vga, snapshot, e.units, s.cache, targets, cam[0], cam[1], idle, blend, base)
	e.add("heal_blend", obj{"scenario": s.json, "targets": tj, "idle": idle, "blend": blend, "base": base},
		obj{"error": errv(err), "work": digest(work), "vga": digest(vga)})
	var queue []battle.NativePresentationDigit
	var qj []any
	for i := 0; i < e.intn(12); i++ {
		target := 0
		if len(targets) > 0 {
			target = targets[e.intn(len(targets))].RecordIndex
		}
		if e.chance(5) {
			target = 99
		}
		digit := []int{0, 0x69 + e.intn(10), 0x74, 0x75, 0x76, e.intn(len(e.digits) + 3)}[e.intn(6)]
		d := battle.NativePresentationDigit{PositionCode: []int{2, 8, 12, 17}[e.intn(4)], Target: target, Digit: digit}
		queue = append(queue, d)
		qj = append(qj, []int{d.PositionCode, d.Target, d.Digit})
	}
	var vertical [25]int
	for i := range vertical {
		vertical[i] = e.intn(16)
	}
	frameIndex := e.rangeInt(-1, 22)
	work, vga = s.buffers()
	dl, err := indexedmap.ComposeNativeCommandHealDigitFrameWithSources(work, vga, snapshot, e.digits, queue, targets, cam[0], cam[1], vertical, frameIndex)
	e.add("heal_digit", obj{"scenario": s.json, "targets": tj, "queue": qj, "vertical": vertical, "frame": frameIndex},
		obj{"error": errv(err), "work": digest(work), "vga": digest(vga), "layers": layersJSON(dl)})
}

func (e *exporter) opSpawn(s *scenario) {
	in := indexedmap.NativeSpawnIntroFrameInput{Frame: s.transition(), OldUnitCount: e.rangeInt(0, len(s.frame.Units))}
	if e.chance(5) {
		in.OldUnitCount = len(s.frame.Units) + 1
	}
	work, vga := s.buffers()
	snapshot := must(indexedmap.SeedNativeSpawnIntroSnapshot(fill(workSize, s.workSalt+6000)))
	var steps []any
	for _, step := range fdother.NativeSpawnIntroSchedule() {
		entry := e.spawn[step.LMIEntry%len(e.spawn)]
		err := indexedmap.ComposeNativeSpawnIntroPass(work, vga, snapshot, in, entry, step)
		steps = append(steps, []any{step.Pass, errv(err), digest(work), digest(vga), digest(snapshot)})
	}
	bad := fdother.NativeSpawnIntroSchedule()[3]
	bad.DelayTicks++
	err := indexedmap.ComposeNativeSpawnIntroPass(work, vga, snapshot, in, e.spawn[0], bad)
	steps = append(steps, []any{-1, errv(err), digest(work), digest(vga), digest(snapshot)})
	e.add("spawn", obj{"scenario": s.json, "old": in.OldUnitCount}, obj{"steps": steps})
}

func (e *exporter) opHUD(s *scenario) {
	var rows []any
	for i := 0; i < 4; i++ {
		hud := s.hud
		if i > 0 {
			hud.AnchorX = []int{1, 0xf2, 136, 251, 252, 0}[e.intn(6)]
			hud.TerrainControl = byte(e.intn(7))
		}
		dst := fill(fdicon.NativeMapStride*200, s.workSalt+7000+i)
		err := indexedmap.BlitNativeMapHUD(e.frames, s.m.terrain, e.units, s.cache, dst, hud)
		rows = append(rows, []any{hud.AnchorX, hud.TerrainControl, errv(err), digest(dst)})
	}
	for _, v := range []int{-100, -99, -12, -1, 0, 5, 9, 10, 99, 100} {
		origin := e.rangeInt(-1, fdicon.NativeMapStride*30)
		dst := fill(fdicon.NativeMapStride*30, s.workSalt+v)
		err := indexedmap.BlitNativeMapHUDTwoDigitNumber(e.frames, dst, origin, v)
		rows = append(rows, []any{"two", v, origin, errv(err), digest(dst)})
	}
	for _, c := range [][2]uint16{{7, 7}, {7, 8}, {999, 999}, {1000, 1000}, {1000, 1001}, {0, 1}, {65535, 0}} {
		dst := fill(fdicon.NativeMapStride*200, s.workSalt+int(c[0]))
		anchor := []int{1, 0xf2, 300}[e.intn(3)]
		err := indexedmap.BlitNativeMapHUDHP(e.frames, dst, anchor, c[0], c[1])
		rows = append(rows, []any{"hp", c[0], c[1], anchor, errv(err), digest(dst)})
	}
	e.add("hud", obj{"scenario": s.json}, obj{"rows": rows})
}

func (e *exporter) genMosaic() {
	for i := 0; i < 6; i++ {
		src := fill(vgaSize, 9000+i)
		var rows []any
		for _, block := range []int{0, 1, 2, 3, 4, 8, 13, 16, 17, 312, 313} {
			dst := fill(vgaSize, 9100+i)
			err := indexedmap.MosaicNativeMapViewport(dst, src, block)
			rows = append(rows, []any{block, errv(err), digest(dst)})
		}
		e.add("mosaic", obj{"salt": 9000 + i}, obj{"rows": rows})
	}
	work := fill(workSize, 9200)
	staging := fill(312*192, 9201)
	err := indexedmap.SeedNativeCh23Staging(work, staging)
	e.add("ch23", obj{}, obj{"error": errv(err), "work": digest(work)})
}

func (e *exporter) genScalars() {
	var banner []any
	for step := -2; step <= 35; step++ {
		banner = append(banner, []int{step, indexedmap.PhaseBannerBlock(step), indexedmap.PhaseBannerDim(step),
			indexedmap.PhaseBannerSlideInOffset(step), indexedmap.PhaseBannerSlideOutOffset(step)})
	}
	consts := []any{indexedmap.PhaseBannerSteps, indexedmap.PhaseBannerEnterSteps, indexedmap.PhaseBannerExitSteps,
		indexedmap.PhaseBannerSlideInSteps, indexedmap.PhaseBannerSlideOutSteps, indexedmap.PhaseBannerSlideOriginX,
		indexedmap.PhaseBannerDimFirstIndex, indexedmap.PhaseBannerDimLastIndex, indexedmap.NativeMapViewportX0,
		indexedmap.NativeMapViewportY0, indexedmap.NativeMapViewportX1, indexedmap.NativeMapViewportY1,
		indexedmap.NativeMapVGASize, indexedmap.NativeUnitPresentWorkSize}
	var anchors, states, cycles, eligible, strips []any
	for i := 0; i < 200; i++ {
		a, d, n := []int{1, 0xf2, 7}[e.intn(3)], e.rangeInt(0, 12), e.rangeInt(0, 14)
		anchors = append(anchors, []int{a, d, n, indexedmap.AdvanceNativeMapHUDAnchor(a, d, n)})
		st, tick, last := e.intn(4), e.rangeInt(-40000, 40000), e.rangeInt(-40000, 40000)
		if e.chance(50) {
			tick = last + e.rangeInt(-3, 8)
		}
		ns, nl := indexedmap.AdvanceNativeMapHUDState(st, tick, last)
		states = append(states, []int{st, tick, last, ns, nl})
		mv := e.intn(4)
		ci, cm, cl := indexedmap.AdvanceNativeMapSpriteCycles(st, mv, tick, last)
		cycles = append(cycles, []int{st, mv, tick, last, ci, cm, cl})
		b7, b1f, b6 := byte([]int{0x79, 0, e.intn(256)}[e.intn(3)]), byte([]int{0x0a, e.intn(256)}[e.intn(2)]), byte(e.intn(3))
		eligible = append(eligible, []any{b7, b1f, b6, indexedmap.NativeMapHUDOptionalUnitEligible(b7, b1f, b6)})
		mx, my, cx, cy := e.rangeInt(-2, 40), e.rangeInt(-2, 40), e.rangeInt(0, 30), e.rangeInt(0, 30)
		l := indexedmap.NativeUnitPresentStripLayoutFor(mx, my, cx, cy)
		strips = append(strips, []int{mx, my, cx, cy, l.WorkOffset, l.VGAOffset, l.Rows})
	}
	var apdp []any
	for c := 0; c < 8; c++ {
		ap, dp, err := indexedmap.NativeMapHUDTerrainAPDP(byte(c))
		apdp = append(apdp, []any{c, ap, dp, errv(err)})
	}
	var cells []any
	for _, m := range e.maps {
		c, err := indexedmap.BuildNativeTerrainCells(m.tiles, m.modes)
		cells = append(cells, []any{m.index, errv(err), cellsDigest(c)})
	}
	for _, tc := range []struct {
		tiles []int
		modes []byte
	}{{nil, nil}, {[]int{1}, nil}, {[]int{0x400}, []byte{0}}, {[]int{-1}, []byte{0}}, {[]int{0x3ff, 0}, []byte{1, 2}}} {
		c, err := indexedmap.BuildNativeTerrainCells(tc.tiles, tc.modes)
		modes := make([]int, len(tc.modes)) // []byte 會被 encoding/json 寫成 base64，改寫成數字陣列
		for i, v := range tc.modes {
			modes[i] = int(v)
		}
		cells = append(cells, []any{tc.tiles, modes, errv(err), cellsDigest(c)})
	}
	e.add("scalars", obj{}, obj{"banner": banner, "consts": consts, "millis": indexedmap.PhaseBannerStepMillis,
		"tick_hz": indexedmap.PhaseBannerStepTickHz, "anchors": anchors, "states": states, "cycles": cycles,
		"eligible": eligible, "strips": strips, "apdp": apdp, "cells": cells})
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	assets := flag.String("assets", "", "remake/assets 目錄")
	pack := flag.String("pack", "", "分離素材包根目錄（fd2-original-b97caf22）")
	flag.Parse()
	if *out == "" || *assets == "" || *pack == "" {
		fmt.Fprintln(os.Stderr, "需要 -out、-assets、-pack")
		os.Exit(2)
	}
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x50524d41)), assets: *assets, pack: *pack}
	for _, rel := range []string{"sprites/fdicon", "sprites/fdother_001_range_overlay", "palette", "ui/fdother_005_item_panel",
		"animations/fdother_009_spawn_intro", "ui/fdother_005_lmi1_opaque", "effects/fdother_006_lmi1_opaque", "tilesets/fdshap", "surfaces"} {
		e.recordTree("pack", *pack, rel)
	}
	e.load()
	salt := 0
	for _, m := range e.maps {
		for k := 0; k < 3; k++ {
			salt++
			s := e.newScenario(m, salt)
			e.opFrame(s)
			e.opStep(s)
			e.opTransition(s)
			e.opUnitPresent(s)
			e.opCommand10(s)
			e.opHeal(s)
			e.opSpawn(s)
			e.opHUD(s)
		}
	}
	e.genMosaic()
	e.genScalars()
	raw, err := json.MarshalIndent(obj{"schema": "fd2-golden-present-map-v1", "seed": *seed, "files": e.files, "cases": e.cases}, "", " ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, append(raw, '\n'), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("%d 案例\n", len(e.cases))
}
