// fd2-golden-fdother-native 輸出 internal/fdother B 組（原版流程邏輯，沒有套件內相依的
// 檔案）的黃金差分資料，供 Rust 改寫逐欄比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 fdother 的公開 API，不改任何遊戲行為。
// 狀態與輸入值域小的函式（phase、按鍵、selector、計數器、mode 布林組合）逐一列舉，
// 另含範圍內外的邊界值；其餘以固定種子的 PCG 產生隨機案例。輸入與結果寫進同一份 JSON，
// Rust 端不需重現亂數。
// 會被 64 位元繞回騙過範圍檢查而在執行期 panic 的輸入以 protect 記錄 panicked，
// 並保留 panic 之前已寫入的緩衝區內容與 callback 呼叫紀錄。
// 本組不讀原版檔：FDOTHER #80 的原版資源檢查留在單元測試（FD2_FLAME2_DIR）。
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type obj = map[string]any

type caseJSON struct {
	Index  int    `json:"index"`
	Kind   string `json:"kind"`
	Op     string `json:"op"`
	Input  obj    `json:"input"`
	Result obj    `json:"result"`
}

type exporter struct {
	rng   *rand.Rand
	cases []caseJSON
}

func hexs(b []byte) string { return hex.EncodeToString(b) }

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

// protect 回報 Go 執行期 panic（例如 64 位元繞回騙過範圍檢查後的越界索引）。
func protect(f func() error) (err error, panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	err = f()
	return
}

func (e *exporter) add(kind, op string, input, result obj) {
	e.cases = append(e.cases, caseJSON{Index: len(e.cases), Kind: kind, Op: op, Input: input, Result: result})
}

func (e *exporter) bytesN(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(e.rng.Uint32())
	}
	return b
}

func (e *exporter) pickInt(values []int) int { return values[e.rng.IntN(len(values))] }

func (e *exporter) pickByte(values []byte) byte { return values[e.rng.IntN(len(values))] }

var extremes = []int{math.MaxInt64, math.MinInt64, math.MaxInt64 - 1, math.MinInt64 + 1, 1 << 62, -(1 << 62),
	1 << 60, 0x7fffffff, -0x80000000, 1 << 32}

// ---- battle_entry_step ----

func (e *exporter) battleEntry(kind string, raw []byte) {
	input := obj{"raw": hexs(raw)}
	next, eligible, changed, err := fdother.NativeBattleEntryStep(raw)
	e.add(kind, "battle_entry_step", input, obj{"err": errv(err), "next": next, "eligible": eligible,
		"changed": changed, "raw": hexs(raw)})
}

func battleRecord(current, maximum uint16) []byte {
	raw := make([]byte, 0x50)
	raw[6] = 2
	binary.LittleEndian.PutUint16(raw[0x40:], current)
	binary.LittleEndian.PutUint16(raw[0x42:], maximum)
	return raw
}

func (e *exporter) genBattleEntry() {
	for _, offset := range []int{6, 5, 0x25, 0x26} {
		for v := 0; v < 256; v++ {
			raw := battleRecord(30, 100)
			raw[offset] = byte(v)
			e.battleEntry("enum-gate", raw)
		}
	}
	words := []uint16{0, 1, 4, 5, 6, 9, 10, 99, 100, 101, 0x7fff, 0x8000, 0xcccc, 0xccce, 0xfff0, 0xfffe, 0xffff}
	for _, c := range words {
		for _, m := range words {
			e.battleEntry("edge-word", battleRecord(c, m))
		}
	}
	for i := 0; i < 1500; i++ {
		raw := battleRecord(uint16(e.rng.Uint32()), uint16(e.rng.Uint32()))
		if e.rng.IntN(4) == 0 {
			raw[5] = e.pickByte([]byte{0, 1, 0x80, 0x7e, 0x81})
		}
		e.battleEntry("random", raw)
	}
	for n := 0; n <= 0x46; n++ {
		raw := e.bytesN(n)
		if n > 6 {
			raw[6] = 2
			raw[5] &^= 0x81
		}
		if n > 0x26 {
			raw[0x25], raw[0x26] = 0, 0
		}
		e.battleEntry("edge-len", raw)
	}
}

// ---- indexed_transition_buffer ----

func (e *exporter) genTransitionBuffer() {
	for d := 0; d <= 10; d++ {
		for s := 0; s <= 10; s++ {
			dst, src := e.bytesN(d), e.bytesN(s)
			input := obj{"dst": hexs(dst), "src": hexs(src)}
			err := fdother.CopyNativeTransitionBuffer(dst, src)
			e.add("enum-len", "copy_buffer", input, obj{"err": errv(err), "dst": hexs(dst)})
		}
	}
	viewport := func(kind string, dstLen, dstStride, srcLen, srcOffset, srcStride, width, height int) {
		dst, src := e.bytesN(dstLen), e.bytesN(srcLen)
		input := obj{"dst": hexs(dst), "dst_stride": dstStride, "src": hexs(src), "src_offset": srcOffset,
			"src_stride": srcStride, "width": width, "height": height}
		err, panicked := protect(func() error {
			return fdother.CopyNativeTransitionViewport(dst, dstStride, src, srcOffset, srcStride, width, height)
		})
		e.add(kind, "copy_viewport", input, obj{"err": errv(err), "panicked": panicked, "dst": hexs(dst)})
	}
	for i := 0; i < 3000; i++ {
		viewport("random", e.rng.IntN(61), e.rng.IntN(10)-1, e.rng.IntN(61), e.rng.IntN(33)-1,
			e.rng.IntN(10)-1, e.rng.IntN(10)-1, e.rng.IntN(7)-1)
	}
	// 64 位元繞回：height*stride 或 (height-1)*stride 繞成小值，騙過範圍檢查後在複製時 panic。
	viewport("edge-wrap", 32, 4, 32, 0, 4, 4, 1<<62)
	viewport("edge-wrap", 32, 4, 32, 2, 8, 4, 1<<61)
	viewport("edge-wrap", 32, 8, 64, 0, 4, 4, 1<<62)
	viewport("edge-wrap", 8, math.MaxInt64, 8, 0, math.MaxInt64, 2, 1)
	viewport("edge-wrap", 8, math.MaxInt64, 8, math.MaxInt64-1, math.MaxInt64, math.MaxInt64, 1)
	viewport("edge-wrap", 8, 4, 8, math.MaxInt64, 4, 2, 2)
	viewport("edge-wrap", 312*192, 312, 456*192+32904, 32904, 456, 312, 192)
}

// ---- indexed_transition_lut / schedule ----

func (e *exporter) genTransitionLUT() {
	for b := 0; b < 14; b++ {
		n := b
		if b >= 12 {
			n = 10
		}
		bank := make([][]byte, n)
		rows := make([]string, n)
		for i := range bank {
			size := 256
			if e.rng.IntN(5) == 0 {
				size = e.pickInt([]int{0, 1, 255, 257})
			}
			bank[i] = e.bytesN(size)
			rows[i] = hexs(bank[i])
		}
		for d := -2; d <= 12; d++ {
			got, err := fdother.NativeIndexedTransitionLUT(bank, d)
			res := obj{"err": errv(err)}
			if err == nil {
				res["lut"] = hexs(got)
			}
			e.add("enum-descriptor", "transition_lut", obj{"bank": rows, "descriptor": d}, res)
		}
		for _, d := range extremes {
			_, err := fdother.NativeIndexedTransitionLUT(bank, d)
			e.add("edge-descriptor", "transition_lut", obj{"bank": rows, "descriptor": d}, obj{"err": errv(err)})
		}
	}
}

func scheduleFrames(frames []fdother.NativeIndexedTransitionFrame) [][4]int {
	out := make([][4]int, len(frames))
	for i, f := range frames {
		out[i] = [4]int{f.Pass, f.LUTIndex, f.Radius, f.FrameDelayMs}
	}
	return out
}

func (e *exporter) transitionSchedule(kind string, start, step int) {
	s, err := fdother.BuildNativeIndexedTransitionSchedule(start, step)
	res := obj{"err": errv(err)}
	if err == nil {
		res["frames"] = scheduleFrames(s.Frames)
		res["tail_delay_ms"] = s.TailDelayMs
		res["palette_deltas"] = s.PaletteDeltas
		res["palette_delay_ms"] = s.PaletteDelayMs
	}
	e.add(kind, "transition_schedule", obj{"start": start, "step": step}, res)
}

func (e *exporter) genTransitionSchedule() {
	starts := append([]int{-1, 0, 1, 10, 0x7ffe, 0x7fff, 0x8000}, extremes...)
	steps := append([]int{-0x8000, -0x7fff, -0x7ffe, -1, 0, 1, 8, 0x7ffe, 0x7fff, 0x8000}, extremes...)
	for _, a := range starts {
		for _, b := range steps {
			e.transitionSchedule("edge", a, b)
		}
	}
	for i := 0; i < 200; i++ {
		e.transitionSchedule("random", e.rng.IntN(0x10002)-1, e.rng.IntN(0x10003)-0x8001)
	}
}

// ---- map_binding ----

func (e *exporter) mapIndex(kind, path string) {
	index, err := fdother.MapIndexFromAssetPath(path)
	e.add(kind, "map_index", obj{"path": path}, obj{"err": errv(err), "index": index})
}

func (e *exporter) genMapBinding() {
	fixed := []string{"", ".", "..", "/", "//", "assets", "assets/", "./assets", "/assets", "assets/.", "assets/x/..",
		"map", "map0", "map00", "map01", "map32", "map-0", "map-1", "map+5", "map 5", "map5 ", "map0x10", "map1_0",
		"map9223372036854775807", "map9223372036854775808", "map-9223372036854775808", "Map3", "MAP3", "xmap3",
		"maps/map7/", "maps/map7/.", "maps/map7/..", "maps/map7/../map8", "map7/../..", "/../map3", "../map3",
		"../../map3/", "a//b//map4", "map4/./", "maps\\map5", "assets/maps/map", "assets/maps/map01x",
		"assets/maps/map-1", "assets/maps/other", "assets/maps/map0", "./map12", "map١", "map３", "map\x00",
		"maps/assets", "maps/assets/..", "map3/x/..", "/map3/../..", "map+", "map-", "map++1", "map+-1"}
	for _, p := range fixed {
		e.mapIndex("fixed", p)
	}
	tokens := []string{"map", "map0", "map7", "map42", "map-3", "map+4", "map007", "mapx", "assets", "a", "maps",
		".", "..", "", "map99999999999999999999"}
	for i := 0; i < 1200; i++ {
		parts := make([]string, 1+e.rng.IntN(4))
		for j := range parts {
			parts[j] = tokens[e.rng.IntN(len(tokens))]
		}
		p := strings.Join(parts, "/")
		if e.rng.IntN(6) == 0 {
			p = "/" + p
		}
		if e.rng.IntN(6) == 0 {
			p += "/"
		}
		e.mapIndex("random", p)
	}
}

// ---- schedules keyed by command id ----

func commandIDs() []int {
	ids := []int{}
	for id := -5; id <= 300; id++ {
		ids = append(ids, id)
	}
	return append(ids, extremes...)
}

func (e *exporter) genCommandSchedules() {
	for _, id := range commandIDs() {
		s, err := fdother.BuildNativeAIItemDamageTailSchedule(id)
		res := obj{"err": errv(err)}
		if err == nil {
			res["schedule"] = obj{"effect_resource": s.EffectResource, "effect_start": s.EffectStart,
				"effect_frames": s.EffectFrames, "effect_sample": s.EffectSample,
				"effect_frame_delay_ticks": s.EffectFrameDelayTicks, "blend_frames": s.BlendFrames,
				"raw_base": s.RawBase, "blend_delay_ticks": s.BlendDelayTicks, "blend": s.Blend,
				"digit_resource": s.DigitResource, "digit_bias": s.DigitBias, "digit_frames": s.DigitFrames,
				"digit_frame_delay_ticks": s.DigitFrameDelayTicks, "digit_hold_ms": s.DigitHoldMs,
				"digit_vertical": s.DigitVertical}
		}
		e.add("enum-id", "ai_item_damage", obj{"id": id}, res)
	}
	for _, id := range commandIDs() {
		s, err := fdother.BuildNativeCommand10To12Schedule(id)
		res := obj{"err": errv(err)}
		if err == nil {
			p := s.Prelude
			res["schedule"] = obj{"command_id": s.CommandID, "sound_resource": s.SoundResource,
				"prelude":   []any{p.Enabled, p.InitialRadius, p.RadiusStep, p.Repeat, p.Sample},
				"x_origins": s.XOrigins, "y_origins": s.YOrigins, "sampling_increments": s.SamplingIncrements,
				"surface_cycle": s.SurfaceCycle, "frames": s.Frames, "delay_ms": s.DelayMS,
				"main_sample": s.MainSample, "main_sample_frames": s.MainSampleFrames,
				"result_resource": s.ResultResource, "result_digit_bias": s.ResultDigitBias,
				"result_miss_descriptors": s.ResultMissDescriptors, "result_frames": s.ResultFrames,
				"result_hold_ms": s.ResultHoldMS, "result_vertical": s.ResultVertical}
		}
		e.add("enum-id", "command10_12", obj{"id": id}, res)
	}
	for _, id := range commandIDs() {
		s, err := fdother.BuildNativeCommandHealPresentationSchedule(id)
		res := obj{"err": errv(err)}
		if err == nil {
			frames := make([][3]int, len(s.Frames))
			for i, f := range s.Frames {
				frames[i] = [3]int{f.LUTIndex, f.Radius, f.FrameDelayMs}
			}
			res["schedule"] = obj{"frames": frames, "mid_frame": s.MidFrame, "mid_delay_ms": s.MidDelayMs,
				"tail_delay_ms": s.TailDelayMs, "sample_index": s.SampleIndex}
		}
		e.add("enum-id", "heal_presentation", obj{"id": id}, res)
	}
	for _, id := range commandIDs() {
		s, err := fdother.BuildNativeCommandHealTailSchedule(id)
		res := obj{"err": errv(err)}
		if err == nil {
			res["schedule"] = obj{"effect_resource": s.EffectResource, "effect_start": s.EffectStart,
				"effect_frames": s.EffectFrames, "effect_sample": s.EffectSample,
				"effect_frame_delay_ticks": s.EffectFrameDelayTicks, "mask_sample": s.MaskSample,
				"mask_index": s.MaskIndex, "mask_pairs": s.MaskPairs, "digit_resource": s.DigitResource,
				"digit_bias": s.DigitBias, "digit_frames": s.DigitFrames, "digit_frame_delay_ms": s.DigitFrameDelayMs,
				"digit_hold_ms": s.DigitHoldMs, "digit_vertical": s.DigitVertical}
		}
		e.add("enum-id", "heal_tail", obj{"id": id}, res)
	}
	rows, err := fdother.BuildNativeCommand34TailSchedule()
	res := obj{"err": errv(err)}
	if err == nil {
		out := make([]obj, len(rows))
		for i, r := range rows {
			extra := r.ExtraSampleFrameIndices
			if extra == nil {
				extra = []int{}
			}
			out[i] = obj{"command_id": r.CommandID, "effect_start": r.EffectStart, "effect_frames": r.EffectFrames,
				"effect_sample": r.EffectSample, "extra_sample_frame_indices": extra, "mask_index": r.MaskIndex,
				"mask_pairs": r.MaskPairs, "digit_bias": r.DigitBias, "digit_frames": r.DigitFrames,
				"digit_hold_milliseconds": r.DigitHoldMilliseconds, "digit_vertical": r.DigitVertical}
		}
		res["rows"] = out
	}
	e.add("fixed", "command34", obj{}, res)
}

// ---- native_compound_palette ----

func (e *exporter) compoundDAC(kind string, baseline []byte, start, end, delta int, raw [3]byte) {
	input := obj{"baseline": hexs(baseline), "start": start, "end": end, "delta": delta, "raw": hexs(raw[:])}
	out, err := fdother.InterpolateNativeCompoundDAC(baseline, start, end, delta, raw)
	res := obj{"err": errv(err)}
	if err == nil {
		res["out"] = hexs(out)
	}
	e.add(kind, "compound_dac", input, res)
}

func (e *exporter) rawDAC() [3]byte {
	var raw [3]byte
	copy(raw[:], e.bytesN(3))
	return raw
}

func (e *exporter) genCompoundDAC() {
	for delta := -2; delta <= 42; delta++ {
		e.compoundDAC("enum-delta", e.bytesN(768), 0, 256, delta, e.rawDAC())
	}
	for _, start := range []int{math.MinInt64, -1, 0, 1, 254, 255, 256} {
		for _, end := range []int{-1, 0, 1, 2, 255, 256, 257, math.MaxInt64} {
			for _, delta := range []int{0, 20, 40} {
				e.compoundDAC("edge-range", e.bytesN(768), start, end, delta, e.rawDAC())
			}
		}
	}
	for _, n := range []int{0, 1, 767, 769, 1536} {
		e.compoundDAC("edge-len", e.bytesN(n), 0, 256, 20, e.rawDAC())
	}
	for i := 0; i < 150; i++ {
		start := e.rng.IntN(256)
		end := start + 1 + e.rng.IntN(256-start)
		e.compoundDAC("random", e.bytesN(768), start, end, e.rng.IntN(41), e.rawDAC())
	}
}

// ---- routes ----

func (e *exporter) genRoutes() {
	for s := 0; s < 256; s++ {
		route, ok := fdother.ResolveNativeHotelServiceRoute(byte(s))
		res := obj{"ok": ok}
		if ok {
			res["route"] = []any{route.Selector, route.ResourceID, route.Primary, route.Secondary}
		} else if route != (fdother.NativeHotelServiceRoute{}) {
			panic("hotel route: non-zero rejected route")
		}
		e.add("enum-selector", "hotel_route", obj{"selector": s}, res)
	}
	for v := 0; v < 256; v++ {
		routes := make([]any, 256)
		for s := 0; s < 256; s++ {
			route, ok := fdother.ResolveNativeShopServiceRoute(byte(v), byte(s))
			if ok {
				routes[s] = []any{route.HubVariant, route.ResourceID, route.Selector, route.Callee, string(route.Kind)}
			} else if route != (fdother.NativeShopServiceRoute{}) {
				panic("shop route: non-zero rejected route")
			}
		}
		e.add("enum-variant-selector", "shop_route", obj{"hub_variant": v}, obj{"routes": routes})
	}
	tables := [][]byte{nil, {0}, {1}, make([]byte, 30), e.bytesN(30), e.bytesN(40)}
	tables[3][22], tables[3][27] = 1, 1
	for i := 0; i < 6; i++ {
		t := make([]byte, 1+e.rng.IntN(40))
		for j := range t {
			if e.rng.IntN(3) == 0 {
				t[j] = byte(1 + e.rng.IntN(255))
			}
		}
		tables = append(tables, t)
	}
	for _, table := range tables {
		chapters := []int{math.MinInt64, -2, -1}
		for c := 0; c <= len(table)+1; c++ {
			chapters = append(chapters, c)
		}
		chapters = append(chapters, math.MaxInt64)
		for _, chapter := range chapters {
			routes := make([]any, 256)
			for s := 0; s < 256; s++ {
				route, ok := fdother.ResolveNativePostbattleRoute(chapter, table, byte(s))
				if ok {
					routes[s] = []any{route.Selector, route.Callee, route.Preparation, route.DirectPreparation}
				} else if route != (fdother.NativePostbattleRoute{}) {
					panic("postbattle route: non-zero rejected route")
				}
			}
			e.add("enum-chapter-selector", "postbattle_route", obj{"chapter": chapter, "table": hexs(table)},
				obj{"routes": routes})
		}
	}
	selectors := []int{0, 1, 2, 3, 4, 5, 6, 7, 128, 255}
	callees := []uint32{0, 0x318ad, 0x2fc85, 0x2e341, 0x3072f, 1, 0xffffffff}
	results := []int{0, 1, -1, 7, math.MaxInt64, math.MinInt64}
	for _, s := range selectors {
		for _, callee := range callees {
			for flags := 0; flags < 4; flags++ {
				for _, r := range results {
					route := fdother.NativePostbattleRoute{Selector: byte(s), Callee: callee,
						Preparation: flags&1 != 0, DirectPreparation: flags&2 != 0}
					out, ok := fdother.ResolveNativePostbattleOutcome(route, r)
					res := obj{"ok": ok}
					if ok {
						res["outcome"] = []any{out.Repeat, out.ReturnValue}
					} else if out != (fdother.NativePostbattleOutcome{}) {
						panic("postbattle outcome: non-zero rejected outcome")
					}
					e.add("enum-route", "postbattle_outcome", obj{"selector": s, "callee": callee,
						"preparation": route.Preparation, "direct_preparation": route.DirectPreparation,
						"callee_result": r}, res)
				}
			}
		}
	}
}

// ---- preparation ----

func (e *exporter) genPreparationSmall() {
	for index := -300; index <= 300; index++ {
		e.add("enum-index", "party_limit", obj{"index": index},
			obj{"limit": fdother.NativePreparationPartyLimit(index)})
	}
	for _, index := range extremes {
		e.add("edge-index", "party_limit", obj{"index": index},
			obj{"limit": fdother.NativePreparationPartyLimit(index)})
	}
	for d := 0; d < 256; d++ {
		keys := make([]byte, 256)
		for k := 0; k < 256; k++ {
			keys[k] = fdother.NormalizeNativePreparationKey(byte(d), byte(k))
		}
		e.add("enum-key", "preparation_key", obj{"raw53a8d": d}, obj{"keys": hexs(keys)})
	}
	states := []any{nil, -1, 0, 1, 2, math.MaxInt64}
	for scan := 0; scan < 256; scan++ {
		out := make([][2]any, len(states))
		for i, s := range states {
			if s == nil {
				out[i] = [2]any{int(fdother.ApplyNativePreparationInput(byte(scan), nil)), nil}
				continue
			}
			state := s.(int)
			got := fdother.ApplyNativePreparationInput(byte(scan), &state)
			out[i] = [2]any{int(got), state}
		}
		e.add("enum-scan-state", "preparation_input", obj{"scan": scan, "states": states}, obj{"results": out})
	}
}

func (e *exporter) prepDispatch(kind string, table []byte, gate3, selector byte) {
	matches, err := fdother.FindNativePreparationDispatch(table, gate3, selector)
	res := obj{"err": errv(err)}
	if err == nil {
		out := make([][3]int, len(matches))
		for i, m := range matches {
			out[i] = [3]int{m.Slot, m.EntryStart, int(m.FunctionIndex)}
		}
		res["matches"] = out
	}
	e.add(kind, "preparation_dispatch", obj{"table": hexs(table), "gate3": gate3, "selector": selector}, res)
}

func (e *exporter) genPreparationDispatch() {
	alphabet := []byte{0, 1, 2, 0x2a}
	for t := 0; t < 30; t++ {
		n := 49 + e.rng.IntN(8)
		table := make([]byte, n)
		for i := range table {
			table[i] = e.pickByte(alphabet)
		}
		for _, g := range alphabet {
			for _, s := range alphabet {
				e.prepDispatch("enum-gate-selector", table, g, s)
			}
		}
	}
	for n := 0; n <= 52; n++ {
		e.prepDispatch("edge-len", make([]byte, n), 0, 0)
	}
}

func (e *exporter) prepRecord(kind string, raw []byte, selector int) {
	record, err := fdother.ParseNativePreparationRecord(raw)
	res := obj{"err": errv(err)}
	if err == nil {
		adjusted, divisor := fdother.NativePreparationAdjustedWord40(record)
		res["record"] = []any{record.Raw25, record.Raw5, record.Raw6, record.Raw7, record.Word40, record.Word42}
		res["eligible"] = fdother.NativePreparationEligible(record, selector)
		res["adjusted"] = adjusted
		res["divisor"] = divisor
	}
	e.add(kind, "preparation_record", obj{"raw": hexs(raw), "selector": selector}, res)
}

func (e *exporter) genPreparationRecord() {
	for _, raw5 := range []byte{0, 1, 2, 3, 0xfe, 0xff} {
		for _, raw25 := range []byte{0, 1, 0xff} {
			for _, raw6 := range []byte{0, 7, 0xff} {
				for _, selector := range []int{math.MinInt64, -1, 0, 7, 8, 255, 256} {
					raw := e.bytesN(0x50)
					raw[5], raw[0x25], raw[6] = raw5, raw25, raw6
					e.prepRecord("enum-gate", raw, selector)
				}
			}
		}
	}
	words := []uint16{0, 1, 2, 9, 10, 11, 99, 100, 0x7fff, 0x8000, 0xfff6, 0xffff}
	for _, w40 := range words {
		for _, w42 := range words {
			raw := e.bytesN(0x44)
			binary.LittleEndian.PutUint16(raw[0x40:], w40)
			binary.LittleEndian.PutUint16(raw[0x42:], w42)
			e.prepRecord("edge-word", raw, int(raw[6]))
		}
	}
	for i := 0; i < 600; i++ {
		e.prepRecord("random", e.bytesN(0x44+e.rng.IntN(13)), e.rng.IntN(258)-1)
	}
	for n := 0; n <= 0x45; n++ {
		e.prepRecord("edge-len", e.bytesN(n), 0)
	}
}

func (e *exporter) prepTimer(kind string, records []byte, count, selector int) {
	input := obj{"records": hexs(records), "count": count, "selector": selector}
	var expired []fdother.NativePreparationTimerExpiry
	err, panicked := protect(func() error {
		var err error
		expired, err = fdother.TickNativePreparationTimers(records, count, selector)
		return err
	})
	res := obj{"err": errv(err), "panicked": panicked, "records": hexs(records)}
	if err == nil && !panicked {
		out := make([][4]int, len(expired))
		for i, x := range expired {
			out[i] = [4]int{x.RecordIndex, x.CounterIndex, x.CounterOffset, x.VisualSourceOffset}
		}
		res["expired"] = out
	}
	e.add(kind, "preparation_timer", input, res)
}

func (e *exporter) timerRecords(n int) []byte {
	records := make([]byte, n*0x50)
	for r := 0; r < n; r++ {
		base := r * 0x50
		records[base+5] = e.pickByte([]byte{0, 0, 1, 2, 3})
		records[base+6] = e.pickByte([]byte{0, 1, 4, 4, 255})
		for c := 0; c < 8; c++ {
			records[base+0x21+c] = e.pickByte([]byte{0, 1, 1, 2, 255})
		}
	}
	return records
}

func (e *exporter) genPreparationTimer() {
	for _, selector := range []int{math.MinInt64, -1, 0, 1, 4, 255, 256, math.MaxInt64} {
		for count := -1; count <= 4; count++ {
			e.prepTimer("enum-count-selector", e.timerRecords(4), count, selector)
		}
	}
	for i := 0; i < 500; i++ {
		n := e.rng.IntN(5)
		records := e.timerRecords(n)
		if e.rng.IntN(6) == 0 && len(records) > 0 {
			records = records[:len(records)-1-e.rng.IntN(len(records))]
		}
		e.prepTimer("random", records, e.rng.IntN(6)-1, e.pickInt([]int{0, 1, 4, 255}))
	}
	// 64 位元繞回：1<<60 筆 × 0x50 繞成 0，騙過長度檢查後逐筆處理到越界 panic。
	e.prepTimer("edge-wrap", e.timerRecords(2), 1<<60, 4)
	e.prepTimer("edge-wrap", e.timerRecords(1), (1<<60)+1, 4)
	e.prepTimer("edge-wrap", e.timerRecords(0), 1<<62, 0)
}

// ---- post_resolution_commands ----

func (e *exporter) postResolution(kind string, raw []byte, count int) {
	input := obj{"raw": hexs(raw), "count": count}
	var commands []fdother.NativePostResolutionCommand
	err, panicked := protect(func() error {
		var err error
		commands, err = fdother.ParseNativePostResolutionCommands(raw, count)
		return err
	})
	res := obj{"err": errv(err), "panicked": panicked}
	if err == nil && !panicked {
		out := make([][2]int, len(commands))
		for i, c := range commands {
			out[i] = [2]int{int(c.Kind), int(c.Payload)}
		}
		res["commands"] = out
	}
	e.add(kind, "post_resolution", input, res)
}

func (e *exporter) genPostResolution() {
	for n := 0; n <= 13; n++ {
		raw := e.bytesN(n)
		for count := -2; count <= 5; count++ {
			e.postResolution("enum-len-count", raw, count)
		}
	}
	for i := 0; i < 200; i++ {
		e.postResolution("random", e.bytesN(e.rng.IntN(64)), e.rng.IntN(24))
	}
	// count*3 繞成負值騙過長度檢查；Go 在 make 時 panic。
	e.postResolution("edge-wrap", e.bytesN(9), (1<<62)+1)
	e.postResolution("edge-wrap", e.bytesN(9), math.MaxInt64)
	e.postResolution("edge-wrap", e.bytesN(0), math.MinInt64)
}

// ---- native_unit_mode ----

func (e *exporter) unitMode(kind string, record []byte, unit int, callerArg byte) {
	plan, err := fdother.PlanNativeUnitMode(record, unit, callerArg)
	res := obj{"err": errv(err), "plan": nil}
	if plan != nil {
		res["plan"] = []any{plan.UnitIndex, plan.CallerArg, plan.Mode, plan.RawArg35, plan.RawArg36, plan.RawCounter}
	}
	e.add(kind, "unit_mode", obj{"record": hexs(record), "unit_index": unit, "caller_arg": callerArg}, res)
}

func (e *exporter) genUnitMode() {
	for _, offset := range []int{5, 0x34} {
		for v := 0; v < 256; v++ {
			record := e.bytesN(0x3e)
			record[5] &^= 5
			record[offset] = byte(v)
			e.unitMode("enum-byte", record, e.rng.IntN(40), byte(e.rng.Uint32()))
		}
	}
	for n := 0; n <= 0x40; n++ {
		e.unitMode("edge-len", e.bytesN(n), 3, 1)
	}
	for _, unit := range extremes {
		e.unitMode("edge-unit", make([]byte, 0x50), unit, 0xff)
	}
	for i := 0; i < 300; i++ {
		e.unitMode("random", e.bytesN(0x3e+e.rng.IntN(0x20)), e.rng.IntN(1000)-500, byte(e.rng.Uint32()))
	}
}

// ---- native_unit_mode_branch ----

func actionsOf(actions []fdother.NativeUnitModeAction) []int {
	out := make([]int, len(actions))
	for i, a := range actions {
		out[i] = int(a)
	}
	return out
}

func bit(n, i int) bool { return n>>i&1 != 0 }

func (e *exporter) branch(mode int, args []any, actions []fdother.NativeUnitModeAction) {
	e.add("enum-branch", "unit_mode_branch", obj{"mode": mode, "args": args}, obj{"actions": actionsOf(actions)})
}

func (e *exporter) genUnitModeBranch() {
	targets := []int{math.MinInt64, -2, -1, 0, 1, 4, math.MaxInt64}
	for n := 0; n < 8; n++ {
		a, b, c := bit(n, 0), bit(n, 1), bit(n, 2)
		e.branch(0, []any{a, b, c}, fdother.PlanNativeUnitMode0(a, b, c))
	}
	for n := 0; n < 4; n++ {
		a, b := bit(n, 0), bit(n, 1)
		e.branch(1, []any{a, b}, fdother.PlanNativeUnitMode1(a, b))
		e.branch(7, []any{a, b}, fdother.PlanNativeUnitMode7(a, b))
		e.branch(10, []any{a, b}, fdother.PlanNativeUnitMode10(a, b))
	}
	for n := 0; n < 2; n++ {
		a := bit(n, 0)
		e.branch(2, []any{a}, fdother.PlanNativeUnitMode2(a))
		e.branch(4, []any{a}, fdother.PlanNativeUnitMode4(a))
	}
	for _, t := range targets {
		for n := 0; n < 16; n++ {
			a, b, c, d := bit(n, 0), bit(n, 1), bit(n, 2), bit(n, 3)
			e.branch(3, []any{a, t, b, c, d}, fdother.PlanNativeUnitMode3(a, t, b, c, d))
			e.branch(9, []any{t, a, b, c, d}, fdother.PlanNativeUnitMode9(t, a, b, c, d))
		}
	}
	for _, state := range []byte{0, 1, 0xff} {
		for _, typ := range []byte{0, 1, 2, 3, 0xff} {
			for n := 0; n < 64; n++ {
				in := fdother.NativeUnitMode5Inputs{CandidateDispatchSucceeded: bit(n, 0), EventStateByte: state,
					EventLookupSucceeded: bit(n, 1), MovementSucceeded: bit(n, 2), ArrivedAtEventCoordinate: bit(n, 3),
					EventRecordType: typ, BlockedSearchSucceeded: bit(n, 4), NearestCoordinateSearchSucceeded: bit(n, 5)}
				e.branch(5, []any{in.CandidateDispatchSucceeded, in.EventStateByte, in.EventLookupSucceeded,
					in.MovementSucceeded, in.ArrivedAtEventCoordinate, in.EventRecordType, in.BlockedSearchSucceeded,
					in.NearestCoordinateSearchSucceeded}, fdother.PlanNativeUnitMode5(in))
			}
		}
	}
	scores := []int32{math.MinInt32, -1, 0, 5, 6, 7, math.MaxInt32}
	for _, cs := range scores {
		for _, pp := range scores {
			for n := 0; n < 2; n++ {
				e.branch(11, []any{cs, pp, bit(n, 0)}, fdother.PlanNativeUnitMode11(cs, pp, bit(n, 0)))
			}
		}
	}
}

// ---- native_phase_dispatch ----

func (e *exporter) phaseRecords(n int) []byte {
	records := make([]byte, n*0x50)
	for r := 0; r < n; r++ {
		base := r * 0x50
		records[base+5] = e.pickByte([]byte{0, 0, 0, 0x80, 1, 0x81, 0x40})
		records[base+6] = e.pickByte([]byte{0, 0, 1, 1, 2})
		records[base+0x26] = e.pickByte([]byte{0, 0, 0, 1})
	}
	return records
}

func candidatesOf(c []fdother.NativePhaseDispatchCandidate) [][2]int {
	out := make([][2]int, len(c))
	for i, x := range c {
		out[i] = [2]int{x.UnitIndex, int(x.SelectorArg)}
	}
	return out
}

func (e *exporter) phaseCount(n int) int {
	switch e.rng.IntN(12) {
	case 0:
		return -1 - e.rng.IntN(2)
	case 1:
		return n + 1
	case 2:
		if n > 0 {
			return n - 1
		}
	}
	return n
}

func (e *exporter) phaseFind(kind string, records []byte, count int) {
	var got []fdother.NativePhaseDispatchCandidate
	err, panicked := protect(func() error {
		var err error
		got, err = fdother.FindNativePhaseDispatchCandidates(records, count)
		return err
	})
	res := obj{"err": errv(err), "panicked": panicked}
	if err == nil && !panicked {
		res["candidates"] = candidatesOf(got)
	}
	e.add(kind, "phase_find", obj{"records": hexs(records), "count": count}, res)
}

func (e *exporter) phasePlan(kind string, records []byte, count int, scores []fdother.NativePhasePreselectionScores) {
	rows := make([][2]int32, len(scores))
	for i, s := range scores {
		rows[i] = [2]int32{s.SpellCandidateScore53C23, s.ItemCommandScore53C33}
	}
	var plan fdother.NativePhaseUnitScanPlan
	err, panicked := protect(func() error {
		var err error
		plan, err = fdother.PlanNativePhaseUnitScans(records, count, scores)
		return err
	})
	res := obj{"err": errv(err), "panicked": panicked}
	if err == nil && !panicked {
		pre := make([][3]any, len(plan.ZeroPreselection))
		for i, p := range plan.ZeroPreselection {
			pre[i] = [3]any{p.UnitIndex, p.SelectorArg, p.RunUnitAction}
		}
		res["selector_one"] = candidatesOf(plan.SelectorOne)
		res["zero_preselection"] = pre
		res["zero_second_pass"] = candidatesOf(plan.ZeroSecondPass)
	}
	e.add(kind, "phase_plan", obj{"records": hexs(records), "count": count, "scores": rows}, res)
}

func (e *exporter) score() int32 {
	return int32(e.pickInt([]int{math.MinInt32, -1, 0, 5, 6, 7, math.MaxInt32}))
}

func (e *exporter) genPhaseFindPlan() {
	for i := 0; i < 600; i++ {
		n := e.rng.IntN(7)
		records := e.phaseRecords(n)
		count := e.phaseCount(n)
		e.phaseFind("random", records, count)
		scores := make([]fdother.NativePhasePreselectionScores, e.rng.IntN(n+2))
		for j := range scores {
			scores[j] = fdother.NativePhasePreselectionScores{SpellCandidateScore53C23: e.score(),
				ItemCommandScore53C33: e.score()}
		}
		e.phasePlan("random", records, count, scores)
	}
	for _, sel := range []byte{0, 1, 2, 0xff} {
		for _, flag := range []byte{0, 1, 0x40, 0x7e, 0x80, 0x81, 0xff} {
			for _, b26 := range []byte{0, 1, 0xff} {
				records := make([]byte, 0x50)
				records[5], records[6], records[0x26] = flag, sel, b26
				e.phaseFind("enum-gate", records, 1)
				e.phasePlan("enum-gate", records, 1, []fdother.NativePhasePreselectionScores{{SpellCandidateScore53C23: 6}})
			}
		}
	}
	e.phaseFind("edge-wrap", make([]byte, 2*0x50), 1<<60)
	e.phaseFind("edge-wrap", make([]byte, 0x50), 1<<62)
	e.phasePlan("edge-wrap", make([]byte, 2*0x50), 1<<60, nil)
	e.phaseFind("edge-count", nil, 0)
	e.phaseFind("edge-count", nil, math.MinInt64)
	e.phasePlan("edge-count", nil, math.MinInt64, nil)
}

type phaseScript struct {
	Nil          [6]bool  `json:"nil"`
	Preselect    [][3]any `json:"preselect"`
	Action       [][4]any `json:"action"`
	Global       []any    `json:"global"`
	Chapter      [][2]any `json:"chapter"`
	ChapterEvent []any    `json:"chapter_event"`
	Pending      [][2]any `json:"pending"`
}

func scriptErr(v any) error {
	if v == nil {
		return nil
	}
	return errors.New(v.(string))
}

func (e *exporter) maybeErr(tag string, n int) any {
	if e.rng.IntN(25) == 0 {
		return fmt.Sprintf("script %s error %d", tag, n)
	}
	return nil
}

func (e *exporter) genScript(n int) phaseScript {
	var s phaseScript
	if e.rng.IntN(10) == 0 {
		s.Nil[e.rng.IntN(6)] = true
	}
	length := func() int { return e.rng.IntN(10) }
	s.Preselect = make([][3]any, length())
	for i := range s.Preselect {
		s.Preselect[i] = [3]any{e.score(), e.score(), e.maybeErr("preselect", i)}
	}
	s.Action = make([][4]any, length())
	for i := range s.Action {
		event := e.pickInt([]int{0xff, 0xff, 0xff, 0, 7, 89, 90, -1, 300})
		offset, value := -1, 0
		if n > 0 && e.rng.IntN(3) == 0 {
			offset = e.rng.IntN(n)*0x50 + e.pickInt([]int{5, 6, 0x26})
			value = int(e.pickByte([]byte{0, 1, 0x80, 2}))
		}
		s.Action[i] = [4]any{event, e.maybeErr("action", i), offset, value}
	}
	s.Global = make([]any, length())
	for i := range s.Global {
		s.Global[i] = e.maybeErr("global", i)
	}
	s.Chapter = make([][2]any, length())
	for i := range s.Chapter {
		s.Chapter[i] = [2]any{e.pickInt([]int{3, 3, 3, 0, 29, 30, -1}), e.maybeErr("chapter", i)}
	}
	s.ChapterEvent = make([]any, length())
	for i := range s.ChapterEvent {
		s.ChapterEvent[i] = e.maybeErr("chapter-event", i)
	}
	s.Pending = make([][2]any, length()*2)
	for i := range s.Pending {
		s.Pending[i] = [2]any{e.pickInt([]int{0, 0, 0, 0, 0, 0, 1, 2, -3}), e.maybeErr("pending", i)}
	}
	return s
}

// callbacks 依腳本依序取值；腳本用完後回到預設值。
func scriptCallbacks(s phaseScript, records []byte, calls *[]string) fdother.NativePhaseScanCallbacks {
	var ip, ia, ig, ic, ie, iq int
	cb := fdother.NativePhaseScanCallbacks{
		Preselect: func(unit int) (fdother.NativePhasePreselectionScores, error) {
			*calls = append(*calls, fmt.Sprintf("preselect:%d", unit))
			if ip >= len(s.Preselect) {
				return fdother.NativePhasePreselectionScores{}, nil
			}
			row := s.Preselect[ip]
			ip++
			return fdother.NativePhasePreselectionScores{SpellCandidateScore53C23: row[0].(int32),
				ItemCommandScore53C33: row[1].(int32)}, scriptErr(row[2])
		},
		RunUnitAction: func(unit int, selector byte) (int, error) {
			*calls = append(*calls, fmt.Sprintf("action:%d:%d", unit, selector))
			if ia >= len(s.Action) {
				return 0xff, nil
			}
			row := s.Action[ia]
			ia++
			if offset := row[2].(int); offset >= 0 {
				records[offset] = byte(row[3].(int))
			}
			return row[0].(int), scriptErr(row[1])
		},
		RunGlobalEvent: func(eventID, unit int) error {
			*calls = append(*calls, fmt.Sprintf("global:%d:%d", eventID, unit))
			if ig >= len(s.Global) {
				return nil
			}
			ig++
			return scriptErr(s.Global[ig-1])
		},
		ChapterIndex: func() (int, error) {
			*calls = append(*calls, "chapter-index")
			if ic >= len(s.Chapter) {
				return 3, nil
			}
			ic++
			return s.Chapter[ic-1][0].(int), scriptErr(s.Chapter[ic-1][1])
		},
		RunChapterEvent: func(chapter, unit int) error {
			*calls = append(*calls, fmt.Sprintf("chapter:%d:%d", chapter, unit))
			if ie >= len(s.ChapterEvent) {
				return nil
			}
			ie++
			return scriptErr(s.ChapterEvent[ie-1])
		},
		PendingCode: func() (int, error) {
			*calls = append(*calls, "pending")
			if iq >= len(s.Pending) {
				return 0, nil
			}
			iq++
			return s.Pending[iq-1][0].(int), scriptErr(s.Pending[iq-1][1])
		},
	}
	if s.Nil[0] {
		cb.Preselect = nil
	}
	if s.Nil[1] {
		cb.RunUnitAction = nil
	}
	if s.Nil[2] {
		cb.RunGlobalEvent = nil
	}
	if s.Nil[3] {
		cb.ChapterIndex = nil
	}
	if s.Nil[4] {
		cb.RunChapterEvent = nil
	}
	if s.Nil[5] {
		cb.PendingCode = nil
	}
	return cb
}

func (e *exporter) phaseExecute(kind string, records []byte, count int, s phaseScript) {
	input := obj{"records": hexs(records), "count": count, "script": s}
	calls := []string{}
	var result fdother.NativePhaseScanResult
	err, panicked := protect(func() error {
		var err error
		result, err = fdother.ExecuteNativePhaseUnitScans(records, count, scriptCallbacks(s, records, &calls))
		return err
	})
	res := obj{"err": errv(err), "panicked": panicked, "calls": calls, "records": hexs(records)}
	if err == nil && !panicked {
		res["result"] = []any{result.Stopped, result.Pass, result.UnitIndex, result.PendingCode}
	}
	e.add(kind, "phase_execute", input, res)
}

func (e *exporter) genPhaseExecute() {
	for i := 0; i < 1500; i++ {
		n := e.rng.IntN(5)
		e.phaseExecute("random", e.phaseRecords(n), e.phaseCount(n), e.genScript(n))
	}
	for mask := 0; mask < 64; mask++ {
		var s phaseScript
		for i := range s.Nil {
			s.Nil[i] = bit(mask, i)
		}
		e.phaseExecute("enum-nil", e.phaseRecords(1), 1, s)
	}
	e.phaseExecute("edge-wrap", e.phaseRecords(2), 1<<60, phaseScript{})
	e.phaseExecute("edge-count", nil, 0, phaseScript{})
	e.phaseExecute("edge-count", nil, math.MinInt64, phaseScript{})
}

// ---- system_info_transition ----

func (e *exporter) genSystemInfo() {
	phases := []string{"steady", "opening", "closing", "", "other", "Opening", "steady ", "close"}
	for _, phase := range phases {
		frames := []int{math.MinInt64}
		for f := -3; f <= 15; f++ {
			frames = append(frames, f)
		}
		frames = append(frames, math.MaxInt64)
		for _, frame := range frames {
			regions, err := fdother.NativeSystemInfoRegions(phase, frame)
			res := obj{"err": errv(err)}
			if err == nil {
				out := make([][6]int, len(regions))
				for i, r := range regions {
					out[i] = [6]int{r.Source.Min.X, r.Source.Min.Y, r.Source.Max.X, r.Source.Max.Y,
						r.Destination.X, r.Destination.Y}
				}
				res["regions"] = out
			}
			e.add("enum-phase-frame", "sysinfo_regions", obj{"phase": phase, "frame": frame}, res)
		}
	}
	surface := func(n int, random bool) ([]byte, any) {
		if random {
			b := e.bytesN(n)
			return b, hexs(b)
		}
		return make([]byte, n), nil
	}
	const full = fdother.NativeSystemInfoBytes
	shapes := [][2]int{{full, full}, {full, full}, {full, full}, {full - 1, full}, {full, full + 1}, {0, 0}, {full, 0}}
	for i, shape := range shapes {
		random := shape[0] == full && shape[1] == full
		baseline, bhex := surface(shape[0], random)
		information, ihex := surface(shape[1], random)
		input := obj{"baseline": bhex, "baseline_len": shape[0], "information": ihex, "information_len": shape[1]}
		opening, closing, err := fdother.NativeSystemInfoTransitionFrames(baseline, information)
		res := obj{"err": errv(err)}
		if err == nil {
			digests := func(frames [][]byte) []string {
				out := make([]string, len(frames))
				for j, f := range frames {
					out[j] = digest(f)
				}
				return out
			}
			res["opening"], res["closing"] = digests(opening), digests(closing)
		}
		e.add(fmt.Sprintf("shape-%d", i), "sysinfo_frames", input, res)
		steady, err := fdother.NativeSystemInfoSteadyFrame(baseline, information)
		res = obj{"err": errv(err)}
		if err == nil {
			res["frame"] = digest(steady)
		}
		e.add(fmt.Sprintf("shape-%d", i), "sysinfo_steady", input, res)
	}
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "需要 -out")
		os.Exit(2)
	}
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x46444f4e))}
	e.genBattleEntry()
	e.genTransitionBuffer()
	e.genTransitionLUT()
	e.genTransitionSchedule()
	e.genMapBinding()
	e.genCommandSchedules()
	e.genCompoundDAC()
	e.genRoutes()
	e.genPreparationSmall()
	e.genPreparationDispatch()
	e.genPreparationRecord()
	e.genPreparationTimer()
	e.genPostResolution()
	e.genUnitMode()
	e.genUnitModeBranch()
	e.genPhaseFindPlan()
	e.genPhaseExecute()
	e.genSystemInfo()
	doc := obj{"schema": "fd2-golden-fdother-native/1", "seed": *seed, "cases": e.cases}
	data, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		panic(err)
	}
	ops := map[string]int{}
	for _, c := range e.cases {
		ops[c.Op]++
	}
	fmt.Printf("fd2-golden-fdother-native: %d cases sha256=%s ops=%v\n", len(e.cases), digest(data), ops)
}
