// fd2-golden-fdicon 輸出 internal/fdicon 的黃金差分資料，供 Rust 改寫逐欄比對
// （docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 fdicon 的公開 API，不改任何遊戲行為。
// 隨機與結構化輸入以固定種子的 PCG 產生，連同結果寫進同一份 JSON，Rust 端不需重現亂數。
// 分離素材載入器的輸入是暫存目錄樹：樹的檔案內容以十六進位存進 blobs（依 SHA-256 去重），
// Rust 端照樣重建。錯誤訊息中的暫存路徑換成 $ROOT／$TMP，真實素材根目錄換成 $REAL。
// 案例的 kind 決定 Rust 端的比對方式：格式錯誤的 JSON／PNG／檔案系統輸入只比接受或拒收
// （Rust 刻意不照做 Go 標準函式庫對這類輸入的行為），其餘逐欄比對。
// 真實原版檔不嵌入 JSON，只記路徑與 SHA-256（分離素材包另記全部檔案的清單摘要）。
package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"hash/crc32"
	"io/fs"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

const ns = fdicon.NativeSize

type obj = map[string]any

type treeSpec struct {
	Base     string      `json:"base,omitempty"`
	Remove   []string    `json:"remove,omitempty"`
	Dirs     []string    `json:"dirs,omitempty"`
	Files    [][2]string `json:"files,omitempty"`
	Chmod000 []string    `json:"chmod000,omitempty"`
}

type caseJSON struct {
	Index  int    `json:"index"`
	Kind   string `json:"kind"`
	Op     string `json:"op"`
	Input  obj    `json:"input"`
	Result obj    `json:"result"`
}

type exporter struct {
	rng       *rand.Rand
	tmp       string
	flame2    string
	assets    string
	blobs     map[string]string
	trees     map[string]treeSpec
	banks     map[string][][3]string
	bankObjs  map[string]*fdicon.Bank
	cases     []caseJSON
	treeCount int
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hexs(b []byte) string { return hex.EncodeToString(b) }

func unhex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func errv(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

func spriteDigest(s fdicon.Sprite) string {
	h := sha256.New()
	h.Write(s.Pixels)
	h.Write(s.Mask)
	h.Write(s.RemapMask)
	return hex.EncodeToString(h.Sum(nil))
}

func bankResult(b *fdicon.Bank, err error) obj {
	out := obj{"err": errv(err)}
	if err == nil {
		ds := make([]string, len(b.Sprites))
		for i, s := range b.Sprites {
			ds[i] = spriteDigest(s)
		}
		out["digests"] = ds
	}
	return out
}

func diff(before, after []byte) [][2]int {
	out := make([][2]int, 0)
	for i := range after {
		if before[i] != after[i] {
			out = append(out, [2]int{i, int(after[i])})
		}
	}
	return out
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

var extremes = []int{0, -1, 1, math.MaxInt64, math.MinInt64, math.MaxInt64 - 10, math.MinInt64 + 10,
	1 << 62, -(1 << 62), 0x7fffffff, 1 << 32, math.MaxInt64 / 12, math.MaxInt64/12 + 1, 0x1555555555555556}

func (e *exporter) pick(lo, hi int) int {
	if e.rng.IntN(25) == 0 {
		return extremes[e.rng.IntN(len(extremes))]
	}
	return lo + e.rng.IntN(hi-lo+1)
}

func (e *exporter) bytesN(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(e.rng.Uint32())
	}
	return b
}

func (e *exporter) layer(density int, odd bool) []byte {
	n := ns * ns
	if odd && e.rng.IntN(10) == 0 {
		n = []int{0, ns*ns - 1, ns*ns + 1}[e.rng.IntN(3)]
	}
	b := make([]byte, n)
	for i := range b {
		if e.rng.IntN(100) < density {
			b[i] = 1
			if e.rng.IntN(40) == 0 {
				b[i] = byte(2 + e.rng.IntN(250))
			}
		}
	}
	return b
}

func (e *exporter) sprite(density int, odd bool) fdicon.Sprite {
	n := ns * ns
	if odd && e.rng.IntN(10) == 0 {
		n = []int{0, ns*ns - 1, ns*ns + 1}[e.rng.IntN(3)]
	}
	return fdicon.Sprite{Pixels: e.bytesN(n), Mask: e.layer(density, odd), RemapMask: e.layer(density/2, odd)}
}

func spriteJSON(s fdicon.Sprite) [3]string {
	return [3]string{hexs(s.Pixels), hexs(s.Mask), hexs(s.RemapMask)}
}

func (e *exporter) defineBank(name string, count, density int, odd bool) {
	b := &fdicon.Bank{Sprites: make([]fdicon.Sprite, count)}
	rows := make([][3]string, count)
	for i := range b.Sprites {
		b.Sprites[i] = e.sprite(density, odd)
		rows[i] = spriteJSON(b.Sprites[i])
	}
	e.banks[name] = rows
	e.bankObjs[name] = b
}

// identBank 的第 i 張圖格以像素 [i&0xff, i>>8] 辨識，Rust 端由 bank_len 自行重建。
func identBank(n int) *fdicon.Bank {
	b := &fdicon.Bank{Sprites: make([]fdicon.Sprite, n)}
	for i := range b.Sprites {
		b.Sprites[i] = fdicon.Sprite{Pixels: []byte{byte(i), byte(i >> 8)}}
	}
	return b
}

func spriteIdent(s fdicon.Sprite) int { return int(s.Pixels[0]) | int(s.Pixels[1])<<8 }

func lutOrNil(lut []byte) any {
	if lut == nil {
		return nil
	}
	return hexs(lut)
}

func (e *exporter) lut() []byte {
	switch e.rng.IntN(20) {
	case 0:
		return nil
	case 1:
		return e.bytesN(255)
	case 2:
		return e.bytesN(257)
	}
	return e.bytesN(256)
}

// ---- 純函式 ----

func (e *exporter) genScalars() {
	for i := 0; i < 160; i++ {
		motion := []int{0, 0, 1, -1, 7, e.pick(-3, 6)}[e.rng.IntN(6)]
		force := e.rng.IntN(3) == 0
		idle, moving := e.pick(-1, 4), e.pick(-1, 4)
		v, err := fdicon.NativeFrameIndex(motion, force, idle, moving)
		e.add("random", "frame_index", obj{"motion": motion, "force": force, "idle": idle, "moving": moving},
			obj{"err": errv(err), "v": v})
	}
	for i := 0; i < 200; i++ {
		x, y, cx, cy := e.pick(-3, 40), e.pick(-3, 30), e.pick(-3, 30), e.pick(-3, 20)
		pose, motion, shift := e.pick(-1, 4), e.pick(-6, 6), e.pick(-1, 2)
		force := e.rng.IntN(2) == 0
		v, err := fdicon.NativePlacementOffset(x, y, cx, cy, pose, motion, shift, force)
		e.add("random", "placement", obj{"x": x, "y": y, "cx": cx, "cy": cy, "pose": pose, "motion": motion,
			"shift": shift, "force": force}, obj{"err": errv(err), "v": v})
	}
	for i := 0; i < 120; i++ {
		st := fdicon.NativeBinaryTickState{Value: e.pick(-1, 2), LastTimerTick: e.pick(-0x8002, 0x8001)}
		raw := e.pick(-0x8002, 0x8001)
		if e.rng.IntN(3) == 0 {
			raw = st.LastTimerTick
		}
		got, err := fdicon.AdvanceNativeBinaryTick(st, raw)
		e.add("random", "binary_tick", obj{"value": st.Value, "last": st.LastTimerTick, "raw": raw},
			obj{"err": errv(err), "value": got.Value, "last": got.LastTimerTick})
	}
	for i := 0; i < 120; i++ {
		st := fdicon.NativeMapSpriteCycleState{Idle: e.pick(-5, 5), Moving: e.pick(-5, 5), LastTimerTick: e.pick(-0x8000, 0x7fff)}
		raw := st.LastTimerTick + e.pick(-3, 8)
		got := fdicon.AdvanceNativeMapSpriteCycles(st, raw)
		e.add("random", "sprite_cycles", obj{"idle": st.Idle, "moving": st.Moving, "last": st.LastTimerTick, "raw": raw},
			obj{"idle": got.Idle, "moving": got.Moving, "last": got.LastTimerTick})
	}
	for i := 0; i < 100; i++ {
		tileWord, eventWord := uint16(e.rng.Uint32()), uint16(e.rng.Uint32())
		if e.rng.IntN(2) == 0 {
			tileWord = uint16(e.rng.IntN(0x40)) | uint16(e.rng.IntN(64))<<10
		}
		controls := e.bytesN(e.rng.IntN(0x40 * 4))
		info, err := fdicon.NativeTerrainCursorInfoForCell(tileWord, eventWord, controls)
		e.add("random", "cursor_info", obj{"tile_word": tileWord, "event_word": eventWord, "controls": hexs(controls)},
			obj{"err": errv(err), "tile": info.Tile, "event_low": info.EventLow, "control": hexs(info.Control[:])})
	}
	for i := 0; i < 100; i++ {
		anchor, stride := e.pick(-2, 260), fdicon.NativeMapStride
		if e.rng.IntN(6) == 0 {
			stride = e.pick(0, 500)
		}
		l, err := fdicon.NativeMapHUDLayoutFor(anchor, stride)
		e.add("random", "hud_layout", obj{"anchor": anchor, "stride": stride}, obj{"err": errv(err),
			"layout": []int{l.Frame, l.Terrain, l.AP, l.DP, l.Unit, l.HP}})
	}
	for i := 0; i < 80; i++ {
		group, state := e.pick(-2, 150), e.pick(-1, 4)
		v, err := fdicon.NativeMapHUDUnitFrameIndex(group, state)
		e.add("random", "hud_unit_frame", obj{"group": group, "state": state}, obj{"err": errv(err), "v": v})
	}
	for i := 0; i < 80; i++ {
		initial := byte(e.rng.Uint32())
		if e.rng.IntN(3) == 0 {
			initial = 0
		}
		n := e.rng.IntN(6)
		targets := make([]fdicon.NativeCommandBackgroundTarget, n)
		rows := make([][]any, n)
		for j := range targets {
			c := [4]byte{byte(e.rng.Uint32()), byte(e.rng.Uint32()), byte(e.rng.IntN(4) * 0x11), byte(e.rng.Uint32())}
			targets[j] = fdicon.NativeCommandBackgroundTarget{Gate: e.rng.IntN(2) == 0, Control: c}
			rows[j] = []any{targets[j].Gate, hexs(c[:])}
		}
		e.add("random", "command_bg", obj{"initial": initial, "targets": rows},
			obj{"v": fdicon.NativeCommandBackgroundSelector(initial, targets)})
	}
	for i := 0; i < 80; i++ {
		inactive := e.rng.IntN(3) == 0
		unit7 := []byte{0, 0x1c, byte(e.rng.Uint32())}[e.rng.IntN(3)]
		race := []byte{0, 4, 5, byte(e.rng.IntN(8))}[e.rng.IntN(4)]
		class := []byte{0, 0x13, byte(e.rng.IntN(0x20))}[e.rng.IntN(3)]
		e.add("random", "fg_eligible", obj{"inactive": inactive, "unit7": unit7, "race": race, "class": class},
			obj{"v": fdicon.NativeForegroundRedrawEligible(inactive, unit7, race, class)})
	}
	for i := 0; i < 80; i++ {
		x, y := e.pick(-2, 40), e.pick(-2, 40)
		pose := []byte{0, 1, 2, 3, byte(e.rng.Uint32())}[e.rng.IntN(5)]
		motion := []int{0, 1, -1, e.pick(-6, 6)}[e.rng.IntN(4)]
		cells, n := fdicon.NativeForegroundRedrawCells(x, y, pose, motion)
		e.add("random", "fg_cells", obj{"x": x, "y": y, "pose": pose, "motion": motion}, obj{"n": n,
			"cells": [][2]int{{cells[0].X, cells[0].Y}, {cells[1].X, cells[1].Y}, {cells[2].X, cells[2].Y}}})
	}
	for i := 0; i < 160; i++ {
		tile, flags, flip, cycle := e.pick(-2, 0x401), byte(e.rng.Uint32()), e.pick(-1, 2), e.pick(-12, 12)
		v, err := fdicon.NativeTerrainFrameIndex(tile, flags, flip, cycle)
		e.add("random", "terrain_frame", obj{"tile": tile, "flags": flags, "flip": flip, "cycle": cycle},
			obj{"err": errv(err), "v": v})
		fv, present, ferr := fdicon.NativeForegroundFrameIndex(tile, flags, flip)
		e.add("random", "fg_frame", obj{"tile": tile, "flags": flags, "flip": flip},
			obj{"err": errv(ferr), "v": fv, "present": present})
	}
	groups := make([]bool, 0)
	for g := -3; g <= fdicon.NativeSourceGroupCount+3; g++ {
		groups = append(groups, fdicon.NativeSourceGroupHasPixels(g))
	}
	e.add("structured", "source_groups", obj{"from": -3, "to": fdicon.NativeSourceGroupCount + 3}, obj{"v": groups})
	e.add("structured", "constants", obj{}, obj{"v": []int{fdicon.NativeSize, fdicon.NativeMapStride,
		fdicon.NativeUnitOriginBytes, fdicon.NativeMotionPixels, fdicon.SeparatedSpriteCount,
		fdicon.NativeSourceGroupCount, fdicon.SeparatedFDSHAPMapCount}})
}

func (e *exporter) genSelector() {
	for i := 0; i < 60; i++ {
		var cache fdicon.NativeSelectorCache
		var clone *fdicon.NativeSelectorCache
		ops := make([][]any, 0)
		res := make([]obj, 0)
		for j := 0; j < 24; j++ {
			key := e.pick(-2, 12)
			if e.rng.IntN(15) == 0 {
				key = 0x100
			}
			slot := e.pick(-1, 8)
			switch k := e.rng.IntN(8); {
			case k < 3:
				v, err := cache.SlotFor(key)
				ops, res = append(ops, []any{"slot", key}), append(res, obj{"v": v, "err": errv(err)})
			case k == 3:
				v, err := cache.KeyForSlot(slot)
				ops, res = append(ops, []any{"key", slot}), append(res, obj{"v": v, "err": errv(err)})
			case k == 4:
				clone = cache.Clone()
				ops, res = append(ops, []any{"clone"}), append(res, obj{})
			case k == 5 && clone != nil:
				v, err := clone.SlotFor(key)
				ops, res = append(ops, []any{"cslot", key}), append(res, obj{"v": v, "err": errv(err)})
			case k == 6 && clone != nil:
				ops, res = append(ops, []any{"ckeys"}), append(res, obj{"v": hexs(clone.Keys())})
			default:
				ops, res = append(ops, []any{"keys"}), append(res, obj{"v": hexs(cache.Keys())})
			}
		}
		keys := cache.Keys()
		switch i % 3 {
		case 1:
			if len(keys) > 0 {
				keys = append(keys, keys[e.rng.IntN(len(keys))])
			}
		case 2:
			keys = e.bytesN(e.rng.IntN(8))
		}
		rebuilt, err := fdicon.NewNativeSelectorCacheFromKeys(keys)
		fk := obj{"err": errv(err)}
		if err == nil {
			fk["keys"] = hexs(rebuilt.Keys())
		}
		ops, res = append(ops, []any{"from_keys", hexs(keys)}), append(res, fk)
		e.add("random", "selector_cache", obj{"ops": ops}, obj{"results": res})
	}
	for i := 0; i < 120; i++ {
		n := e.rng.IntN(40)
		b := identBank(n)
		group, pose, cycle := e.pick(-1, 4), e.pick(-1, 4), e.pick(-1, 3)
		s, err := b.SpriteFor(group, pose, cycle)
		r := obj{"err": errv(err)}
		if err == nil {
			r["sprite"] = spriteIdent(s)
		}
		e.add("random", "sprite_for", obj{"bank_len": n, "group": group, "pose": pose, "cycle": cycle}, r)
	}
	for _, tc := range [][3]int{{math.MaxInt64/12 + 1, 0, 0}, {0x1555555555555556, 0, 0}, {0x1555555555555556, 1, 2}} {
		b := identBank(40)
		s, err := b.SpriteFor(tc[0], tc[1], tc[2])
		r := obj{"err": errv(err)}
		if err == nil {
			r["sprite"] = spriteIdent(s)
		}
		e.add("structured-overflow", "sprite_for", obj{"bank_len": 40, "group": tc[0], "pose": tc[1], "cycle": tc[2]}, r)
	}
	for i := 0; i < 60; i++ {
		n := e.rng.IntN(40)
		b := identBank(n)
		keys := e.bytesN(e.rng.IntN(5))
		for j := range keys {
			keys[j] %= 5
		}
		cache, err := fdicon.NewNativeSelectorCacheFromKeys(keys)
		if err != nil {
			keys = nil
			cache = &fdicon.NativeSelectorCache{}
		}
		slot, pose, cycle := e.pick(-1, 4), e.pick(-1, 4), e.pick(-1, 3)
		s, err := b.SpriteForNativeSlot(cache, slot, pose, cycle)
		r := obj{"err": errv(err)}
		if err == nil {
			r["sprite"] = spriteIdent(s)
		}
		e.add("random", "sprite_for_native_slot", obj{"bank_len": n, "keys": hexs(keys), "slot": slot, "pose": pose,
			"cycle": cycle}, r)
	}
}

func (e *exporter) genBlits() {
	variants := []string{"at", "band", "flags", "flags_offset", "lut", "lut_t", "lut_t_offset", "const", "damage"}
	for i := 0; i < 360; i++ {
		variant := variants[i%len(variants)]
		s := e.sprite(10+e.rng.IntN(60), true)
		stride := 24 + e.rng.IntN(12)
		if e.rng.IntN(12) == 0 {
			stride = e.pick(-2, 30)
		}
		rows := 24 + e.rng.IntN(8)
		n := 24*30 + e.rng.IntN(400)
		if stride > 0 && stride < 64 {
			n = stride*rows - e.rng.IntN(3)
		}
		dst := e.bytesN(n)
		x, y := e.pick(-2, 14), e.pick(-2, 8)
		hi := 50
		if stride > 0 && stride < 64 {
			hi = max(n-(23*stride+24)+3, -2)
		}
		offset := e.pick(-3, hi)
		flags := byte(e.rng.Uint32())
		lut := e.lut()
		index := byte(e.rng.Uint32())
		blend := e.pick(-1, 8)
		rawBase := byte(e.rng.Uint32())
		before := append([]byte(nil), dst...)
		err, panicked := protect(func() error {
			switch variant {
			case "at":
				return s.BlitAt(dst, stride, x, y)
			case "band":
				return s.BlitPaletteBand(dst, stride, x, y)
			case "flags":
				return s.BlitForNativeFlags(dst, stride, x, y, flags)
			case "flags_offset":
				return s.BlitForNativeFlagsAtOffset(dst, stride, offset, flags)
			case "lut":
				return s.BlitLUT(dst, stride, x, y, lut)
			case "lut_t":
				return s.BlitLUTTransparent(dst, stride, x, y, lut)
			case "lut_t_offset":
				return s.BlitLUTTransparentAtOffset(dst, stride, offset, lut)
			case "const":
				return s.BlitConstantMaskAt(dst, stride, x, y, index)
			default:
				return s.BlitNativeDamageBlendAt(dst, stride, x, y, blend, rawBase)
			}
		})
		e.add("random", "sprite_blit", obj{"variant": variant, "sprite": spriteJSON(s), "dst": hexs(before),
			"stride": stride, "x": x, "y": y, "offset": offset, "flags": flags, "lut": lutOrNil(lut), "index": index,
			"blend": blend, "raw_base": rawBase}, obj{"err": errv(err), "panic": panicked, "diff": diff(before, dst)})
	}
	for i := 0; i < 80; i++ {
		dstStride, srcStride := e.pick(-1, 12), e.pick(-1, 12)
		width, height := e.pick(-1, 10), e.pick(-1, 6)
		dst := e.bytesN(e.rng.IntN(70))
		src := e.bytesN(e.rng.IntN(70))
		before := append([]byte(nil), dst...)
		err, panicked := protect(func() error {
			return fdicon.CopyNativeIndexedRegion(dst, dstStride, src, srcStride, width, height)
		})
		e.add("random", "copy_region", obj{"dst": hexs(before), "dst_stride": dstStride, "src": hexs(src),
			"src_stride": srcStride, "width": width, "height": height},
			obj{"err": errv(err), "panic": panicked, "diff": diff(before, dst)})
	}
}

func cellsJSON(cells []fdicon.NativeTerrainCell) [][2]int {
	out := make([][2]int, len(cells))
	for i, c := range cells {
		out[i] = [2]int{int(c.Tile), int(c.BlitMode)}
	}
	return out
}

func (e *exporter) genCells(n, maxTile int) []fdicon.NativeTerrainCell {
	cells := make([]fdicon.NativeTerrainCell, n)
	for i := range cells {
		tile := uint16(e.rng.IntN(maxTile + 1))
		if e.rng.IntN(4) == 0 {
			tile |= uint16(e.rng.IntN(64)) << 10
		}
		mode := byte(0xff)
		if e.rng.IntN(2) == 0 {
			mode = byte(e.rng.Uint32())
		}
		cells[i] = fdicon.NativeTerrainCell{Tile: tile, BlitMode: mode}
	}
	return cells
}

func (e *exporter) controls(tiles int) []byte {
	c := e.bytesN(tiles * 4)
	for i := 0; i < len(c); i += 4 {
		c[i] &= 0x9c
	}
	if e.rng.IntN(10) == 0 && len(c) > 4 {
		c = c[:len(c)-4*(1+e.rng.IntN(len(c)/4))]
	}
	return c
}

func (e *exporter) genTerrain() {
	e.defineBank("terrain48", 48, 25, false)
	e.defineBank("terrain-odd", 24, 25, true)
	for i := 0; i < 120; i++ {
		bankName := []string{"terrain48", "terrain48", "terrain-odd"}[e.rng.IntN(3)]
		b := e.bankObjs[bankName]
		stride := 24 + e.rng.IntN(20)
		dst := e.bytesN(stride * (24 + e.rng.IntN(6)))
		x, y := e.pick(-1, stride-24), e.pick(-1, 4)
		tile, flags, mode := e.pick(-1, 50), byte(e.rng.Uint32())&0x9c, byte(0xff)
		if e.rng.IntN(2) == 0 {
			mode = byte(e.rng.Uint32())
		}
		flip, cycle := e.pick(-1, 2), e.pick(-6, 12)
		lut := e.lut()
		before := append([]byte(nil), dst...)
		fg := i%2 == 1
		if fg {
			flags |= byte(e.rng.IntN(2)) << 7
		}
		err, panicked := protect(func() error {
			if fg {
				return b.BlitNativeForegroundCell(dst, stride, x, y, tile, flags, mode, flip, lut)
			}
			return b.BlitNativeTerrainCell(dst, stride, x, y, tile, flags, mode, flip, cycle, lut)
		})
		op := "terrain_cell"
		if fg {
			op = "fg_cell"
		}
		e.add("random", op, obj{"bank": bankName, "dst": hexs(before), "stride": stride, "x": x, "y": y, "tile": tile,
			"flags": flags, "blit_mode": mode, "flip": flip, "cycle": cycle, "lut": lutOrNil(lut)},
			obj{"err": errv(err), "panic": panicked, "diff": diff(before, dst)})
	}
	for i := 0; i < 80; i++ {
		bankName := []string{"terrain48", "terrain48", "terrain-odd"}[e.rng.IntN(3)]
		b := e.bankObjs[bankName]
		mapWidth := e.pick(-1, 5)
		mapHeight := 1 + e.rng.IntN(4)
		cellCount := max(mapWidth, 1)*mapHeight + []int{0, 0, 0, 1}[e.rng.IntN(4)]
		if mapWidth > 100 || mapWidth < -100 {
			cellCount = 4
		}
		cells := e.genCells(cellCount, 30)
		controls := e.controls(34)
		mapX, mapY := e.pick(-1, 3), e.pick(-1, 3)
		width, height := e.pick(-1, 4), e.pick(-1, 3)
		stride := 24*4 + e.rng.IntN(10)
		dst := e.bytesN(stride * 24 * 3)
		dstX, dstY := e.pick(-1, 30), e.pick(-1, 10)
		flip, cycle := e.pick(-1, 2), e.pick(-4, 8)
		lut := e.lut()
		before := append([]byte(nil), dst...)
		err, panicked := protect(func() error {
			return b.BlitNativeTerrainRegion(dst, stride, dstX, dstY, mapWidth, cells, controls, mapX, mapY, width,
				height, flip, cycle, lut)
		})
		e.add("random", "terrain_region", obj{"bank": bankName, "dst": hexs(before), "stride": stride, "dst_x": dstX,
			"dst_y": dstY, "map_width": mapWidth, "cells": cellsJSON(cells), "controls": hexs(controls), "map_x": mapX,
			"map_y": mapY, "width": width, "height": height, "flip": flip, "cycle": cycle, "lut": lutOrNil(lut)},
			obj{"err": errv(err), "panic": panicked, "diff": diff(before, dst)})
	}
}

func (e *exporter) genLayers() {
	e.defineBank("layer36", 36, 6, false)
	e.defineBank("layer-odd", 24, 6, true)
	for i := 0; i < 80; i++ {
		bankName := []string{"layer36", "layer36", "layer-odd"}[e.rng.IntN(3)]
		b := e.bankObjs[bankName]
		keys := []byte{0, 1, 2}
		if e.rng.IntN(3) == 0 {
			keys = []byte{2, 0}
		}
		cache, _ := fdicon.NewNativeSelectorCacheFromKeys(keys)
		stride := fdicon.NativeMapStride
		if e.rng.IntN(15) == 0 {
			stride = 320
		}
		rows := 70 + e.rng.IntN(60)
		dst := make([]byte, fdicon.NativeMapStride*rows)
		n := e.rng.IntN(7)
		units := make([]fdicon.NativeUnitLayerEntry, n)
		rowsJSON := make([][]any, n)
		cx, cy := e.pick(-1, 4), e.pick(-1, 4)
		for j := range units {
			u := fdicon.NativeUnitLayerEntry{X: cx + e.pick(-2, 13), Y: cy + e.pick(-2, 9), Slot: e.pick(-1, 3),
				Pose: e.pick(-1, 4), MotionOffset: []int{0, 0, 1, 3, 5, -2}[e.rng.IntN(6)], Flags: byte(e.rng.Uint32()),
				ForceBase: e.rng.IntN(4) == 0, Inactive: e.rng.IntN(5) == 0}
			units[j] = u
			rowsJSON[j] = []any{u.X, u.Y, u.Slot, u.Pose, u.MotionOffset, u.Flags, u.ForceBase, u.Inactive}
		}
		vx, vy := e.pick(-1, 13), e.pick(-1, 8)
		idle, moving, shift := e.pick(-1, 4), e.pick(0, 3), e.pick(-1, 2)
		baseShift := []int{0, 0, -8 * fdicon.NativeMapStride, 5, -0x8000, e.pick(-100, 100)}[e.rng.IntN(6)]
		before := append([]byte(nil), dst...)
		err, panicked := protect(func() error {
			return b.BlitNativeUnitLayerWithBaseShift(dst, stride, cache, units, cx, cy, vx, vy, idle, moving, shift,
				baseShift)
		})
		e.add("random", "unit_layer", obj{"bank": bankName, "keys": hexs(keys), "dst_len": len(dst), "stride": stride,
			"units": rowsJSON, "cx": cx, "cy": cy, "vx": vx, "vy": vy, "idle": idle, "moving": moving, "shift": shift,
			"base_shift": baseShift}, obj{"err": errv(err), "panic": panicked, "diff": diff(before, dst)})
	}
	e.defineBank("fg48", 48, 6, false)
	for i := 0; i < 80; i++ {
		bankName := []string{"fg48", "fg48", "layer-odd"}[e.rng.IntN(3)]
		b := e.bankObjs[bankName]
		stride := fdicon.NativeMapStride
		if e.rng.IntN(15) == 0 {
			stride = 320
		}
		mapWidth := e.pick(-1, 8)
		mapHeight := 1 + e.rng.IntN(8)
		cellCount := max(mapWidth, 1)*mapHeight + []int{0, 0, 0, 1}[e.rng.IntN(4)]
		if mapWidth > 100 || mapWidth < -100 {
			cellCount = 8
		}
		cells := e.genCells(cellCount, 40)
		controls := e.controls(42)
		for j := 0; j < len(controls); j += 4 {
			controls[j] |= byte(e.rng.IntN(2)) << 7
		}
		cx, cy := e.pick(-1, 3), e.pick(-1, 3)
		n := e.rng.IntN(6)
		units := make([]fdicon.NativeForegroundLayerEntry, n)
		rowsJSON := make([][]any, n)
		for j := range units {
			u := fdicon.NativeForegroundLayerEntry{X: e.pick(-2, 9), Y: e.pick(-2, 9),
				Pose: []byte{0, 1, 2, 3, byte(e.rng.Uint32())}[e.rng.IntN(5)], MotionOffset: []int{0, 0, 1, -1}[e.rng.IntN(4)],
				Inactive: e.rng.IntN(6) == 0, Unit7: []byte{0, 0x1c}[e.rng.IntN(2)], Race: byte(e.rng.IntN(7)),
				Class: []byte{0, 0x13, 3}[e.rng.IntN(3)]}
			units[j] = u
			rowsJSON[j] = []any{u.X, u.Y, u.Pose, u.MotionOffset, u.Inactive, u.Unit7, u.Race, u.Class}
		}
		vx, vy, flip := e.pick(-1, 13), e.pick(-1, 8), e.pick(-1, 2)
		lut := e.lut()
		rows := 60 + e.rng.IntN(60)
		dst := make([]byte, fdicon.NativeMapStride*rows)
		before := append([]byte(nil), dst...)
		err, panicked := protect(func() error {
			return b.BlitNativeForegroundLayer(dst, stride, units, mapWidth, cells, controls, cx, cy, vx, vy, flip, lut)
		})
		e.add("random", "fg_layer", obj{"bank": bankName, "dst_len": len(dst), "stride": stride, "units": rowsJSON,
			"map_width": mapWidth, "cells": cellsJSON(cells), "controls": hexs(controls), "cx": cx, "cy": cy, "vx": vx,
			"vy": vy, "flip": flip, "lut": lutOrNil(lut)}, obj{"err": errv(err), "panic": panicked, "diff": diff(before, dst)})
	}
}

// ---- FDICON.B24 解析 ----

func (e *exporter) rleRow() []byte {
	var out []byte
	for x := 0; x < ns; {
		rem := ns - x
		mode := e.rng.IntN(4)
		if mode == 1 && rem < 2 {
			mode = 0
		}
		limit := rem
		if mode == 1 {
			limit = rem / 2
		}
		count := 1 + e.rng.IntN(min(limit, 64))
		out = append(out, byte(mode<<6|(count-1)))
		switch mode {
		case 0, 1:
			out = append(out, byte(e.rng.Uint32()))
		case 2:
			out = append(out, e.bytesN(count)...)
		}
		if mode == 1 {
			x += 2 * count
		} else {
			x += count
		}
	}
	return out
}

func (e *exporter) b24(n int) ([]byte, []int) {
	var blobs [][]byte
	for i := 0; i < n; i++ {
		var s []byte
		for r := 0; r < ns; r++ {
			s = append(s, e.rleRow()...)
		}
		blobs = append(blobs, s)
	}
	raw := make([]byte, 6+4*n)
	binary.LittleEndian.PutUint16(raw, ns)
	binary.LittleEndian.PutUint16(raw[2:], ns)
	binary.LittleEndian.PutUint16(raw[4:], uint16(n))
	offs := make([]int, n)
	for i, s := range blobs {
		offs[i] = len(raw)
		binary.LittleEndian.PutUint32(raw[6+4*i:], uint32(len(raw)))
		raw = append(raw, s...)
	}
	return raw, offs
}

func (e *exporter) genParse() {
	kinds := []string{"valid", "valid-junk", "short", "bad-width", "bad-height", "zero-count", "table-overflow",
		"offset-into-table", "dup-offset", "offset-past-end", "truncate", "overrun", "random-body", "random"}
	for i := 0; i < 280; i++ {
		kind := kinds[i%len(kinds)]
		n := 1 + e.rng.IntN(5)
		raw, offs := e.b24(n)
		switch kind {
		case "valid-junk":
			raw = append(raw, e.bytesN(1+e.rng.IntN(20))...)
		case "short":
			raw = raw[:e.rng.IntN(6)]
		case "bad-width":
			binary.LittleEndian.PutUint16(raw, uint16([]int{0, 23, 25, 48}[e.rng.IntN(4)]))
		case "bad-height":
			binary.LittleEndian.PutUint16(raw[2:], uint16([]int{0, 23, 25}[e.rng.IntN(3)]))
		case "zero-count":
			binary.LittleEndian.PutUint16(raw[4:], 0)
		case "table-overflow":
			binary.LittleEndian.PutUint16(raw[4:], uint16(len(raw)/4+e.rng.IntN(100)))
		case "offset-into-table":
			binary.LittleEndian.PutUint32(raw[6+4*e.rng.IntN(n):], uint32(e.rng.IntN(6+4*n)))
		case "dup-offset":
			if n > 1 {
				j := e.rng.IntN(n - 1)
				binary.LittleEndian.PutUint32(raw[6+4*(j+1):], uint32(offs[j]))
			} else {
				binary.LittleEndian.PutUint32(raw[6:], uint32(len(raw)))
			}
		case "offset-past-end":
			j := e.rng.IntN(n)
			binary.LittleEndian.PutUint32(raw[6+4*j:], uint32(len(raw)+e.rng.IntN(5)))
		case "truncate":
			last := offs[n-1]
			raw = raw[:last+e.rng.IntN(len(raw)-last)]
		case "overrun":
			j := e.rng.IntN(n)
			raw[offs[j]] = byte(e.rng.IntN(4)<<6 | 0x3f)
		case "random-body":
			for k := 6 + 4*n; k < len(raw); k++ {
				if e.rng.IntN(30) == 0 {
					raw[k] = byte(e.rng.Uint32())
				}
			}
		case "random":
			raw = e.bytesN(e.rng.IntN(200))
		}
		b, err := fdicon.Parse(raw)
		e.add(kind, "parse", obj{"raw": hexs(raw)}, bankResult(b, err))
	}
}

// ---- PNG 產生器 ----

func chunk(typ string, data []byte) []byte {
	out := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
	out = append(out, typ...)
	out = append(out, data...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(append([]byte(typ), data...)))
}

func rawChunk(length uint32, typ string, data []byte, crc uint32) []byte {
	out := binary.BigEndian.AppendUint32(nil, length)
	out = append(out, typ...)
	out = append(out, data...)
	return binary.BigEndian.AppendUint32(out, crc)
}

type pngSpec struct {
	w, h         int
	depth, ctype byte
	interlace    byte
	sample       func(x, y int) []int
	plte         int
	trns         []byte
	filter       func(row int) byte
	level        int
	ihdr         []byte
	pre          [][]byte // IHDR 之後、PLTE 之前
	mid          [][]byte // PLTE／tRNS 之後、IDAT 之前
	post         [][]byte // IDAT 之後
	tail         []byte   // IEND 之後
	split        []int
	rawMutate    func([]byte) []byte
	zMutate      func([]byte) []byte
	idatMutate   func([][]byte) [][]byte
	noIEND       bool
	noIDATCRC    bool
}

func channels(ctype byte) int {
	return map[byte]int{0: 1, 2: 3, 3: 1, 4: 2, 6: 4}[ctype]
}

func paeth(a, b, c int) int {
	p := a + b - c
	pa, pb, pc := abs(p-a), abs(p-b), abs(p-c)
	if pa <= pb && pa <= pc {
		return a
	}
	if pb <= pc {
		return b
	}
	return c
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func packRow(s pngSpec, xs []int, y int) []byte {
	bits := int(s.depth) * channels(s.ctype)
	row := make([]byte, (bits*len(xs)+7)/8)
	bit := 0
	for _, x := range xs {
		for _, v := range s.sample(x, y) {
			if s.depth == 16 {
				row[bit/8], row[bit/8+1] = byte(v>>8), byte(v)
			} else if s.depth == 8 {
				row[bit/8] = byte(v)
			} else {
				shift := 8 - int(s.depth) - bit%8
				row[bit/8] |= byte(v&(1<<s.depth-1)) << shift
			}
			bit += int(s.depth)
		}
	}
	return row
}

func filterRow(ft byte, cur, prev []byte, bpp int) []byte {
	out := make([]byte, len(cur))
	for i := range cur {
		a, b, c := 0, int(prev[i]), 0
		if i >= bpp {
			a, c = int(cur[i-bpp]), int(prev[i-bpp])
		}
		switch ft {
		case 1:
			out[i] = cur[i] - byte(a)
		case 2:
			out[i] = cur[i] - byte(b)
		case 3:
			out[i] = cur[i] - byte((a+b)/2)
		case 4:
			out[i] = cur[i] - byte(paeth(a, b, c))
		default:
			out[i] = cur[i]
		}
	}
	return append([]byte{ft}, out...)
}

var adam7 = [7][4]int{{8, 8, 0, 0}, {8, 8, 4, 0}, {4, 8, 0, 4}, {4, 4, 2, 0}, {2, 4, 0, 2}, {2, 2, 1, 0}, {1, 2, 0, 1}}

func scanlines(s pngSpec) []byte {
	bpp := max(1, int(s.depth)*channels(s.ctype)/8)
	type pass struct{ xs, ys []int }
	var passes []pass
	if s.interlace == 0 {
		p := pass{}
		for x := 0; x < s.w; x++ {
			p.xs = append(p.xs, x)
		}
		for y := 0; y < s.h; y++ {
			p.ys = append(p.ys, y)
		}
		passes = append(passes, p)
	} else {
		for _, a := range adam7 {
			p := pass{}
			for x := a[2]; x < s.w; x += a[0] {
				p.xs = append(p.xs, x)
			}
			for y := a[3]; y < s.h; y += a[1] {
				p.ys = append(p.ys, y)
			}
			if len(p.xs) > 0 && len(p.ys) > 0 {
				passes = append(passes, p)
			}
		}
	}
	var out []byte
	row := 0
	for _, p := range passes {
		var prev []byte
		for _, y := range p.ys {
			cur := packRow(s, p.xs, y)
			if prev == nil {
				prev = make([]byte, len(cur))
			}
			ft := byte(0)
			if s.filter != nil {
				ft = s.filter(row)
			}
			out = append(out, filterRow(ft, cur, prev, bpp)...)
			prev = cur
			row++
		}
	}
	return out
}

func zlibBytes(raw []byte, level int) []byte {
	var b bytes.Buffer
	w, err := zlib.NewWriterLevel(&b, level)
	if err != nil {
		panic(err)
	}
	w.Write(raw)
	w.Close()
	return b.Bytes()
}

func buildPNG(s pngSpec) []byte {
	out := []byte("\x89PNG\r\n\x1a\n")
	ihdr := s.ihdr
	if ihdr == nil {
		ihdr = binary.BigEndian.AppendUint32(nil, uint32(s.w))
		ihdr = binary.BigEndian.AppendUint32(ihdr, uint32(s.h))
		ihdr = append(ihdr, s.depth, s.ctype, 0, 0, s.interlace)
	}
	out = append(out, chunk("IHDR", ihdr)...)
	for _, c := range s.pre {
		out = append(out, c...)
	}
	if s.plte > 0 {
		pal := make([]byte, 3*s.plte)
		for i := range pal {
			pal[i] = byte(i * 7)
		}
		out = append(out, chunk("PLTE", pal)...)
	}
	if s.trns != nil {
		out = append(out, chunk("tRNS", s.trns)...)
	}
	for _, c := range s.mid {
		out = append(out, c...)
	}
	raw := scanlines(s)
	if s.rawMutate != nil {
		raw = s.rawMutate(raw)
	}
	level := s.level
	if level == 0 {
		level = zlib.DefaultCompression
	}
	if level == 100 {
		level = zlib.NoCompression
	}
	z := zlibBytes(raw, level)
	if s.zMutate != nil {
		z = s.zMutate(z)
	}
	parts := [][]byte{z}
	if s.split != nil {
		parts = nil
		rest := z
		for _, n := range s.split {
			n = min(n, len(rest))
			parts = append(parts, rest[:n])
			rest = rest[n:]
		}
		for len(rest) > 0 {
			n := min(s.split[len(s.split)-1], len(rest))
			if n == 0 {
				n = len(rest)
			}
			parts = append(parts, rest[:n])
			rest = rest[n:]
		}
	}
	if s.idatMutate != nil {
		parts = s.idatMutate(parts)
	}
	for _, p := range parts {
		c := chunk("IDAT", p)
		if s.noIDATCRC {
			c = c[:len(c)-4]
		}
		out = append(out, c...)
		if s.noIDATCRC {
			return out
		}
	}
	for _, c := range s.post {
		out = append(out, c...)
	}
	if !s.noIEND {
		out = append(out, chunk("IEND", nil)...)
	}
	return append(out, s.tail...)
}

func framePattern(x, y int) []int { return []int{(x*37 + y*11 + x*y) & 0xff} }
func maskPattern(x, y int) []int {
	if (x*3+y*5)%7 < 3 {
		return []int{255}
	}
	return []int{0}
}
func remapPattern(x, y int) []int {
	if (x+2*y)%5 == 0 {
		return []int{255}
	}
	return []int{0}
}

func baseFrame() pngSpec {
	return pngSpec{w: ns, h: ns, depth: 8, ctype: 3, plte: 256, sample: framePattern}
}
func baseMask() pngSpec {
	return pngSpec{w: ns, h: ns, depth: 8, ctype: 0, sample: maskPattern}
}

// ---- 暫存目錄樹 ----

func (e *exporter) blob(content []byte) string {
	key := digest(content)
	if _, ok := e.blobs[key]; !ok {
		e.blobs[key] = hexs(content)
	}
	return key
}

func (e *exporter) blobContent(key string) []byte { return unhex(e.blobs[key]) }

func (e *exporter) materialize(name, dir string) {
	t := e.trees[name]
	if t.Base != "" {
		e.materialize(t.Base, dir)
	}
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	for _, p := range t.Remove {
		must(os.RemoveAll(filepath.Join(dir, p)))
	}
	for _, p := range t.Dirs {
		must(os.MkdirAll(filepath.Join(dir, p), 0o755))
	}
	for _, f := range t.Files {
		path := filepath.Join(dir, f[0])
		must(os.MkdirAll(filepath.Dir(path), 0o755))
		must(os.WriteFile(path, e.blobContent(f[1]), 0o644))
	}
	for _, p := range t.Chmod000 {
		must(os.Chmod(filepath.Join(dir, p), 0))
	}
}

// withTree 建立一棵樹、執行 f，結果中的樹根路徑換成 $ROOT，之後刪除樹。
func (e *exporter) withTree(name string, f func(root string) obj) obj {
	e.treeCount++
	dir := filepath.Join(e.tmp, fmt.Sprintf("t%04d", e.treeCount))
	e.materialize(name, dir)
	res := f(dir)
	if err, ok := res["err"].(string); ok {
		res["err"] = strings.ReplaceAll(err, dir, "$ROOT")
	}
	filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil {
			os.Chmod(p, 0o755)
		}
		return nil
	})
	if err := os.RemoveAll(dir); err != nil {
		panic(err)
	}
	return res
}

type spriteEntry struct {
	Index     any `json:"index"`
	Width     any `json:"width"`
	Height    any `json:"height"`
	Frame     any `json:"frame"`
	Mask      any `json:"mask"`
	RemapMask any `json:"remap_mask"`
}

func entries(prefix string, n int) []spriteEntry {
	out := make([]spriteEntry, n)
	for i := range out {
		p := fmt.Sprintf("%s_%04d/", prefix, i)
		out[i] = spriteEntry{i, ns, ns, p + "frame.png", p + "mask.png", p + "remap_mask.png"}
	}
	return out
}

func marshal(v any) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic(err)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// ---- 分離 FDSHAP ----

const fdshapMap = 5

func fdshapSource() string {
	return `{"file":"FDSHAP.DAT","resource":10,"size":3557794,"md5":"9b0d356074f57cc27aebf3bb89aae247","sha256":"901b70ea82d5d977192759fad510921ffe16a0ab6af6ab7c32757de03e30aa3c","raw_size":3557794}`
}

func fdshapDoc(sprites string) string {
	return `{"schema_version":1,"kind":"fdshap_terrain_bank","asset_id":"tilesets/fdshap/map_05","status":"decoded","evidence":"confirmed","source":` +
		fdshapSource() + `,"map_index":5,"image_resource":10,"control_resource":11,"controls":[0,0,4,0,128,8,0,1,16,0,2,0],"sprites":` + sprites + `}`
}

func spriteObj(i int) string { return marshal(entries("tile", 3)[i]) }

func (e *exporter) fdshapBase() {
	files := [][2]string{}
	frame, mask := buildPNG(baseFrame()), buildPNG(baseMask())
	remap := buildPNG(pngSpec{w: ns, h: ns, depth: 8, ctype: 0, sample: remapPattern})
	frame2 := buildPNG(pngSpec{w: ns, h: ns, depth: 8, ctype: 3, plte: 256, sample: func(x, y int) []int {
		return []int{(x ^ y*5) & 0xff}
	}, filter: func(r int) byte { return byte(r % 5) }})
	for i := 0; i < 3; i++ {
		d := fmt.Sprintf("map_05/tile_%04d/", i)
		f := frame
		if i == 1 {
			f = frame2
		}
		files = append(files, [2]string{d + "frame.png", e.blob(f)}, [2]string{d + "mask.png", e.blob(mask)},
			[2]string{d + "remap_mask.png", e.blob(remap)})
	}
	files = append(files, [2]string{"map_05/bank.json", e.blob([]byte(fdshapDoc(marshal(entries("tile", 3)))))})
	e.trees["fdshap-base"] = treeSpec{Files: files}
}

func (e *exporter) loadFdshapCase(kind string, tree string, suffix string, mapIndex int) {
	in := obj{"tree": tree, "suffix": suffix, "map_index": mapIndex}
	res := e.withTree(tree, func(root string) obj {
		b, controls, err := fdicon.LoadSeparatedFDSHAPBank(root+suffix, mapIndex)
		r := bankResult(b, err)
		if err == nil {
			r["controls"] = hexs(controls)
		}
		return r
	})
	e.add(kind, "load_fdshap", in, res)
}

func (e *exporter) fdshapJSONCase(kind, doc string) {
	name := fmt.Sprintf("fdshap-json-%d", len(e.trees))
	e.trees[name] = treeSpec{Base: "fdshap-base", Files: [][2]string{{"map_05/bank.json", e.blob([]byte(doc))}}}
	e.loadFdshapCase(kind, name, "", fdshapMap)
}

func (e *exporter) fdshapFileCase(kind, path string, content []byte) {
	name := fmt.Sprintf("fdshap-file-%d", len(e.trees))
	e.trees[name] = treeSpec{Base: "fdshap-base", Files: [][2]string{{"map_05/" + path, e.blob(content)}}}
	e.loadFdshapCase(kind, name, "", fdshapMap)
}

func (e *exporter) genFdshapJSON() {
	doc := fdshapDoc(marshal(entries("tile", 3)))
	rep := func(old, new string) string {
		if !strings.Contains(doc, old) {
			panic("missing " + old)
		}
		return strings.Replace(doc, old, new, 1)
	}
	s0, s1, s2 := spriteObj(0), spriteObj(1), spriteObj(2)
	deep := strings.Repeat("[", 10001) + strings.Repeat("]", 10001)
	ok300 := strings.Repeat("[", 300) + strings.Repeat("]", 300)
	cases := []struct{ kind, doc string }{
		{"json-valid", doc},
		{"json-valid-ws", " \n\t" + doc + "\r\n "},
		{"json-syntax", ""},
		{"json-syntax", "{"},
		{"json-syntax", doc[:len(doc)/2]},
		{"json-syntax", doc + "x"},
		{"json-syntax", doc + ","},
		{"json-syntax", rep(`"kind":`, `'kind':`)},
		{"json-syntax", rep(`"schema_version":1`, `"schema_version":01`)},
		{"json-syntax", rep(`"decoded"`, "\"dec\x01oded\"")},
		{"json-syntax", rep(`"decoded"`, `"dec\qoded"`)},
		{"json-syntax", "\xef\xbb\xbf" + doc},
		{"json-syntax", rep(`"map_index":5`, `"map_index":-`)},
		{"json-syntax", rep(`"map_index":5`, `"map_index":5.`)},
		{"json-syntax", rep(`"size":3557794`, `"size":3557794e`)},
		{"json-syntax", rep(`[0,0,4`, `[0,,0,4`)},
		{"json-syntax", strings.TrimSuffix(doc, "}")},
		{"json-syntax", rep(`"md5":"9b`, `"md5":"\u12G4`)},
		{"json-syntax", rep(`"status":`, `"x":`+deep+`,"status":`)},
		{"json-syntax", rep(`"status":`, `"x":tru,"status":`)},
		{"json-syntax", rep(`"status":`, `"x":nul,"status":`)},
		{"json-syntax", rep(`"status":`, `"x":fals,"status":`)},
		{"json-syntax", rep(`"status":`, `"x":[1 2],"status":`)},
		{"json-syntax", rep(`"status":`, `"x":{"a" 1},"status":`)},
		{"json-syntax", rep(`"status":`, `"x":{,},"status":`)},
		{"json-syntax", rep(`"status":`, "\"x\":\"\xa0\x85\x7f\",\"status\":")},
		{"json-syntax", "\xa0" + doc},
		{"json-syntax", "\x85" + doc},
		{"json-syntax", "\xe4" + doc},
		{"json-syntax", "\x7f" + doc},
		{"json-syntax", "\\" + doc},
		{"json-unknown", rep(`"status":`, `"x":`+ok300+`,"status":`)},
		{"json-unknown", rep(`"status":`, "\"x\":\"\xff\xfe\",\"y\":{\"z\":[true,false,null,-0.5e+3]},\"status\":")},
		{"json-top", "[1]"},
		{"json-top", "null"},
		{"json-top", `"str"`},
		{"json-top", "123"},
		{"json-top", "true"},
		{"json-type", rep(`"schema_version":1`, `"schema_version":"1"`)},
		{"json-type", rep(`"schema_version":1`, `"schema_version":1.0`)},
		{"json-type", rep(`"schema_version":1`, `"schema_version":1e0`)},
		{"json-type", rep(`"schema_version":1`, `"schema_version":-0`)},
		{"json-type", rep(`"schema_version":1`, `"schema_version":99999999999999999999`)},
		{"json-type", rep(`"schema_version":1`, `"schema_version":-9223372036854775808`)},
		{"json-type", rep(`"schema_version":1`, `"schema_version":-9223372036854775809`)},
		{"json-type", rep(`"schema_version":1`, `"schema_version":true`)},
		{"json-type", rep(`"schema_version":1`, `"schema_version":[]`)},
		{"json-type", rep(`"schema_version":1`, `"schema_version":{}`)},
		{"json-type", rep(`"schema_version":1`, `"schema_version":null`)},
		{"json-type", rep(`"kind":"fdshap_terrain_bank"`, `"kind":5`)},
		{"json-type", rep(`"kind":"fdshap_terrain_bank"`, `"kind":null`)},
		{"json-type", rep(`"kind":"fdshap_terrain_bank"`, `"kind":[]`)},
		{"json-type", rep(`"kind":"fdshap_terrain_bank"`, `"kind":{"a":1}`)},
		{"json-type", rep(`"kind":"fdshap_terrain_bank"`, `"kind":false`)},
		{"json-type", rep(`"source":`+fdshapSource(), `"source":"x"`)},
		{"json-type", rep(`"source":`+fdshapSource(), `"source":[1]`)},
		{"json-type", rep(`"source":`+fdshapSource(), `"source":null`)},
		{"json-type", rep(`"source":`+fdshapSource(), `"source":7`)},
		{"json-type", rep(`"source":`+fdshapSource(), `"source":{}`)},
		{"json-type", rep(`"size":3557794,"md5"`, `"size":"3557794","md5"`)},
		{"json-type", rep(`"file":"FDSHAP.DAT"`, `"file":["FDSHAP.DAT"]`)},
		{"json-type", rep(`"sprites":[`, `"sprites":{},"x":[`)},
		{"json-type", rep(`"sprites":[`, `"sprites":"x","x":[`)},
		{"json-type", rep(`"sprites":[`, `"sprites":[1],"x":[`)},
		{"json-type", rep(`"sprites":[`, `"sprites":[[]],"x":[`)},
		{"json-type", rep(`"sprites":[`, `"sprites":null,"x":[`)},
		{"json-type", rep(`"sprites":[`, `"sprites":true,"x":[`)},
		{"json-type", rep(`{"index":1,`, `{"index":"1",`)},
		{"json-type", rep(`{"index":1,`, `{"index":1.5,`)},
		{"json-type", rep(`"frame":"tile_0001/frame.png"`, `"frame":1`)},
		{"json-type", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":["a",0,4,0,128,8,0,1,16,0,2,0]`)},
		{"json-type", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[1.5,0,4,0,128,8,0,1,16,0,2,0]`)},
		{"json-type", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":{}`)},
		{"json-type", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":5`)},
		{"json-type", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":"abcd"`)},
		{"json-type", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[[1],0,4,0]`)},
		{"json-type", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[99999999999999999999,0,4,0]`)},
		{"json-type-first", rep(`"schema_version":1,"kind":"fdshap_terrain_bank"`, `"schema_version":"1","kind":5`)},
		{"json-type-first", rep(`"evidence":"confirmed"`, `"evidence":"confirmed","x":[{"index":"q"}],"sprites":[1]`)},
		{"json-null", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[null,null,null,null,null,null,null,null,null,null,null,null]`)},
		{"json-null", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":null`)},
		{"json-null", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[]`)},
		{"json-null", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[0,0,4,0,128,8,0,1,16,0,2]`)},
		{"json-null", rep(`{"index":1,`, `{"index":null,`)},
		{"json-range", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[0,0,4,0,256,8,0,1,16,0,2,0]`)},
		{"json-range", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[0,0,4,0,128,8,0,-1,16,0,2,0]`)},
		{"json-range", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[255,255,255,255]`)},
		{"json-fold", rep(`"schema_version"`, `"Schema_Version"`)},
		{"json-fold", rep(`"kind"`, `"KIND"`)},
		{"json-fold", rep(`"kind"`, `"Kind"`)},
		{"json-fold", rep(`"kind"`, "\"\xe2\x84\xaaIND\"")},
		{"json-fold", rep(`"status"`, "\"\xc5\xbftatus\"")},
		{"json-fold", rep(`"status"`, `"ſtatus"`)},
		{"json-fold", rep(`"status"`, `"status"`)},
		{"json-fold", rep(`"kind"`, "\"k\xc4\xb1nd\"")},
		{"json-fold", rep(`"asset_id"`, `"Asset_ID"`)},
		{"json-fold", rep(`"asset_id"`, `"assetid"`)},
		{"json-fold", rep(`"source":{"file"`, `"SOURCE":{"FILE"`)},
		{"json-fold", rep(`"sprites"`, `"Sprites"`)},
		{"json-fold", rep(`{"index":0,`, `{"INDEX":0,`)},
		{"json-fold", rep(`"remap_mask":"tile_0002/remap_mask.png"`, `"Remap_Mask":"tile_0002/remap_mask.png"`)},
		{"json-dup", rep(`"kind":"fdshap_terrain_bank"`, `"kind":"x","KIND":"fdshap_terrain_bank"`)},
		{"json-dup", rep(`"kind":"fdshap_terrain_bank"`, `"kind":"fdshap_terrain_bank","Kind":"x"`)},
		{"json-dup", rep(`"kind":"fdshap_terrain_bank"`, `"kind":"fdshap_terrain_bank","kind":null`)},
		{"json-dup", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[1,2,3,4,5,6,7,8],"controls":[9,9,9,9]`)},
		{"json-dup", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[1,2,3,4,5,6,7,8],"controls":[9],"controls":[null,null,null,null,null,null,null,null]`)},
		{"json-dup", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[1,2,3,4,5],"controls":[7],"controls":[null,null,null,null,null,null,null,null,null,null,null,null]`)},
		{"json-dup", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[1,2,3,4],"controls":[],"controls":[null,null,null,null]`)},
		{"json-dup", rep(`"controls":[0,0,4,0,128,8,0,1,16,0,2,0]`, `"controls":[1,2,3,4],"controls":null,"controls":[null,null,null,null]`)},
		{"json-dup", rep(`"sprites":[`, `"sprites":[`+s0+`,`+s1+`,`+s2+`],"sprites":[{"index":2},{"index":1},{"index":0}],"x":[`)},
		{"json-dup", rep(`"sprites":[`, `"sprites":[`+s0+`,`+s1+`,`+s2+`],"sprites":[`+s0+`],"sprites":[{},{},{}],"x":[`)},
		{"json-dup", rep(`"sprites":[`, `"sprites":[`+s0+`,`+s1+`,`+s2+`,`+s2+`],"sprites":[`+s0+`],"sprites":[{},{},{}],"x":[`)},
		{"json-dup", rep(`"sprites":[`, `"sprites":[`+s0+`,`+s1+`,`+s2+`],"sprites":[`+s0+`],"sprites":[{},{},{},{}],"x":[`)},
		{"json-dup", rep(`"sprites":[`, `"sprites":[`+s0+`,`+s1+`,`+s2+`],"sprites":[],"sprites":[{},{},{}],"x":[`)},
		{"json-dup", rep(`"source":`+fdshapSource(), `"source":`+fdshapSource()+`,"source":{"file":"x"}`)},
		{"json-dup", rep(`"source":`+fdshapSource(), `"source":`+fdshapSource()+`,"source":{"resource":3}`)},
		{"json-escape", rep(`"decoded"`, `"decoded"`)},
		{"json-escape", rep(`"decoded"`, `"dec\/oded"`)},
		{"json-escape", rep(`"tile_0001/frame.png"`, `"tile_0001\/frame.png"`)},
		{"json-escape", rep(`"decoded"`, "\"dec\xffoded\"")},
		{"json-escape", rep(`"decoded"`, `"decoded\ud800"`)},
		{"json-escape", rep(`"decoded"`, `"decoded😀"`)},
		{"json-contract", rep(`"map_index":5`, `"map_index":6`)},
		{"json-contract", rep(`"image_resource":10`, `"image_resource":11`)},
		{"json-contract", rep(`"control_resource":11`, `"control_resource":10`)},
		{"json-contract", rep(`"asset_id":"tilesets/fdshap/map_05"`, `"asset_id":"tilesets/fdshap/map_5"`)},
		{"json-contract", rep(`"md5":"9b`, `"md5":"9B`)},
		{"json-contract", rep(`"evidence":"confirmed"`, `"evidence":"strong"`)},
		{"json-contract", rep(`"sprites":[`, `"sprites":[],"x":[`)},
		{"json-sprite", rep(`{"index":1,`, `{"index":-1,`)},
		{"json-sprite", rep(`{"index":1,`, `{"index":3,`)},
		{"json-sprite", rep(`{"index":1,`, `{"index":99999999999,`)},
		{"json-sprite", rep(`{"index":1,`, `{"index":0,`)},
		{"json-sprite", rep(`"index":2,"width":24`, `"index":2,"width":23`)},
		{"json-sprite", rep(`"index":2,"width":24,"height":24`, `"index":2,"width":24,"height":0`)},
		{"json-sprite", rep(`"tile_0001/frame.png"`, `"tile_0001/frame.PNG"`)},
		{"json-sprite", rep(`"tile_0002/mask.png"`, `"tile_0001/mask.png"`)},
		{"json-sprite", rep(`"tile_0000/remap_mask.png"`, `"tile_0000/remap.png"`)},
		{"json-sprite", rep(`"tile_0001/frame.png"`, `"tile_0001//frame.png"`)},
		{"json-sprite-order", fdshapDoc(`[` + s2 + `,` + s0 + `,` + s1 + `]`)},
	}
	for _, tc := range cases {
		e.fdshapJSONCase(tc.kind, tc.doc)
	}
	for i := 0; i < 90; i++ {
		b := []byte(doc)
		switch i % 3 {
		case 0:
			b[e.rng.IntN(len(b))] = byte(e.rng.Uint32())
		case 1:
			k := e.rng.IntN(len(b))
			b = append(b[:k], b[k+1:]...)
		default:
			k := e.rng.IntN(len(b) + 1)
			ins := []byte(`{}[],:"\0-1e. tnu`)
			b = append(b[:k], append([]byte{ins[e.rng.IntN(len(ins))]}, b[k:]...)...)
		}
		e.fdshapJSONCase("json-fuzz", string(b))
	}
}

func idatCRCFix(png []byte) []byte {
	out := append([]byte(nil), png[:8]...)
	for p := 8; p+12 <= len(png); {
		n := int(binary.BigEndian.Uint32(png[p:]))
		if p+12+n > len(png) {
			return append(out, png[p:]...)
		}
		c := png[p : p+12+n]
		fixed := append([]byte(nil), c...)
		binary.BigEndian.PutUint32(fixed[8+n:], crc32.ChecksumIEEE(c[4:8+n]))
		out = append(out, fixed...)
		p += 12 + n
	}
	return out
}

func (e *exporter) genFdshapPNG() {
	fr := func(mod func(*pngSpec)) []byte {
		s := baseFrame()
		mod(&s)
		return buildPNG(s)
	}
	mk := func(mod func(*pngSpec)) []byte {
		s := baseMask()
		mod(&s)
		return buildPNG(s)
	}
	hdr := func(w, h uint32, depth, ctype, comp, filt, inter byte) []byte {
		b := binary.BigEndian.AppendUint32(nil, w)
		b = binary.BigEndian.AppendUint32(b, h)
		return append(b, depth, ctype, comp, filt, inter)
	}
	idx := func(v int) func(x, y int) []int {
		return func(x, y int) []int {
			if x == 5 && y == 7 {
				return []int{v}
			}
			return []int{(x + y) % 4}
		}
	}
	baseZ := zlibBytes(scanlines(baseFrame()), zlib.DefaultCompression)
	frames := []struct {
		kind string
		data []byte
	}{
		{"png-frame-valid", fr(func(s *pngSpec) {})},
		{"png-frame-filters", fr(func(s *pngSpec) { s.filter = func(r int) byte { return byte(r % 5) } })},
		{"png-frame-interlaced", fr(func(s *pngSpec) { s.interlace = 1 })},
		{"png-frame-interlaced-filters", fr(func(s *pngSpec) { s.interlace = 1; s.filter = func(r int) byte { return byte((r * 3) % 5) } })},
		{"png-frame-plte16-idx255", fr(func(s *pngSpec) { s.plte = 16; s.sample = idx(255) })},
		{"png-frame-plte16-idx254", fr(func(s *pngSpec) { s.plte = 16; s.sample = idx(254) })},
		{"png-frame-plte16-trns256", fr(func(s *pngSpec) { s.plte = 16; s.sample = idx(3); s.trns = bytes.Repeat([]byte{0x80}, 256) })},
		{"png-frame-plte16-trns255", fr(func(s *pngSpec) { s.plte = 16; s.sample = idx(3); s.trns = bytes.Repeat([]byte{0x80}, 255) })},
		{"png-frame-plte200", fr(func(s *pngSpec) { s.plte = 200; s.sample = idx(100) })},
		{"png-frame-interlaced-idx255", fr(func(s *pngSpec) {
			s.plte = 10
			s.interlace = 1
			s.sample = func(x, y int) []int {
				if x == 23 && y == 23 {
					return []int{255}
				}
				return []int{x % 10}
			}
		})},
		{"png-frame-depth4", fr(func(s *pngSpec) { s.depth = 4; s.plte = 16; s.sample = idx(15) })},
		{"png-frame-depth2-trns256", fr(func(s *pngSpec) { s.depth = 2; s.plte = 4; s.sample = idx(3); s.trns = make([]byte, 256) })},
		{"png-frame-depth1-trns256", fr(func(s *pngSpec) {
			s.depth = 1
			s.plte = 2
			s.sample = func(x, y int) []int { return []int{(x + y) % 2} }
			s.trns = make([]byte, 256)
		})},
		{"png-frame-depth1-trns256-interlaced", fr(func(s *pngSpec) {
			s.depth = 1
			s.plte = 2
			s.interlace = 1
			s.sample = func(x, y int) []int { return []int{(x * y) % 2} }
			s.trns = make([]byte, 256)
		})},
		{"png-frame-size25", fr(func(s *pngSpec) { s.h = 25 })},
		{"png-frame-size23", fr(func(s *pngSpec) { s.w = 23 })},
		{"png-frame-rgba", fr(func(s *pngSpec) {
			s.ctype = 6
			s.plte = 0
			s.sample = func(x, y int) []int { return []int{x, y, 1, 255} }
		})},
		{"png-frame-rgb", fr(func(s *pngSpec) { s.ctype = 2; s.plte = 0; s.sample = func(x, y int) []int { return []int{x, y, 1} } })},
		{"png-frame-rgb-plte", fr(func(s *pngSpec) { s.ctype = 2; s.plte = 7; s.sample = func(x, y int) []int { return []int{x, y, 1} } })},
		{"png-frame-gray", fr(func(s *pngSpec) { s.ctype = 0; s.plte = 0 })},
		{"png-frame-ga16", fr(func(s *pngSpec) {
			s.ctype = 4
			s.depth = 16
			s.plte = 0
			s.sample = func(x, y int) []int { return []int{x * 999, 65535} }
		})},
		{"png-frame-rgb16-trns", fr(func(s *pngSpec) {
			s.ctype = 2
			s.depth = 16
			s.plte = 0
			s.sample = func(x, y int) []int { return []int{x, y, 3} }
			s.trns = make([]byte, 6)
		})},
		{"png-frame-gray16", fr(func(s *pngSpec) {
			s.ctype = 0
			s.depth = 16
			s.plte = 0
			s.sample = func(x, y int) []int { return []int{x * y * 100} }
		})},
		{"png-frame-rgba16-interlaced", fr(func(s *pngSpec) {
			s.ctype = 6
			s.depth = 16
			s.plte = 0
			s.interlace = 1
			s.sample = func(x, y int) []int { return []int{x, y, 3, 65535} }
			s.filter = func(r int) byte { return byte(r % 5) }
		})},
		{"png-frame-not-png", []byte("GIF89a this is not a png at all")},
		{"png-frame-short", []byte{0x89, 'P', 'N', 'G', '\r'}},
		{"png-frame-empty", nil},
		{"png-frame-sig-only", []byte("\x89PNG\r\n\x1a\n")},
		{"png-frame-trunc-ihdr", fr(func(s *pngSpec) {})[:20]},
		{"png-frame-trunc-chunk-header", fr(func(s *pngSpec) {})[:12]},
		{"png-frame-ihdr-crc", func() []byte { b := fr(func(s *pngSpec) {}); b[29] ^= 1; return b }()},
		{"png-frame-ihdr-len", fr(func(s *pngSpec) { s.ihdr = hdr(24, 24, 8, 3, 0, 0, 0)[:12] })},
		{"png-frame-depth3", fr(func(s *pngSpec) { s.ihdr = hdr(24, 24, 3, 3, 0, 0, 0) })},
		{"png-frame-ctype5", fr(func(s *pngSpec) { s.ihdr = hdr(24, 24, 8, 5, 0, 0, 0) })},
		{"png-frame-depth16-paletted", fr(func(s *pngSpec) { s.ihdr = hdr(24, 24, 16, 3, 0, 0, 0) })},
		{"png-frame-comp1", fr(func(s *pngSpec) { s.ihdr = hdr(24, 24, 8, 3, 1, 0, 0) })},
		{"png-frame-filtmeth1", fr(func(s *pngSpec) { s.ihdr = hdr(24, 24, 8, 3, 0, 1, 0) })},
		{"png-frame-interlace2", fr(func(s *pngSpec) { s.ihdr = hdr(24, 24, 8, 3, 0, 0, 2) })},
		{"png-frame-zero-w", fr(func(s *pngSpec) { s.ihdr = hdr(0, 24, 8, 3, 0, 0, 0) })},
		{"png-frame-neg-h", fr(func(s *pngSpec) { s.ihdr = hdr(24, 0x80000000, 8, 3, 0, 0, 0) })},
		{"png-frame-dim-overflow", fr(func(s *pngSpec) { s.ihdr = hdr(0x7fffffff, 0x7fffffff, 8, 3, 0, 0, 0) })},
		{"png-frame-huge-dim-short-data", fr(func(s *pngSpec) { s.ihdr = hdr(24, 30000, 8, 3, 0, 0, 0) })},
		{"png-frame-plte-first", func() []byte {
			b := []byte("\x89PNG\r\n\x1a\n")
			b = append(b, chunk("PLTE", make([]byte, 768))...)
			return append(b, chunk("IHDR", hdr(24, 24, 8, 3, 0, 0, 0))...)
		}()},
		{"png-frame-no-plte", fr(func(s *pngSpec) { s.plte = 0 })},
		{"png-frame-plte-len", fr(func(s *pngSpec) { s.plte = 0; s.mid = [][]byte{chunk("PLTE", make([]byte, 10))} })},
		{"png-frame-plte-257", fr(func(s *pngSpec) { s.plte = 0; s.mid = [][]byte{chunk("PLTE", make([]byte, 771))} })},
		{"png-frame-plte-empty", fr(func(s *pngSpec) { s.plte = 0; s.mid = [][]byte{chunk("PLTE", nil)} })},
		{"png-frame-plte-depth", fr(func(s *pngSpec) { s.depth = 4; s.plte = 17; s.sample = idx(1) })},
		{"png-frame-gray-plte", fr(func(s *pngSpec) { s.ctype = 0; s.plte = 4 })},
		{"png-frame-dup-plte", fr(func(s *pngSpec) { s.mid = [][]byte{chunk("PLTE", make([]byte, 768))} })},
		{"png-frame-trns-before-plte", fr(func(s *pngSpec) { s.pre = [][]byte{chunk("tRNS", []byte{1})} })},
		{"png-frame-trns-long", fr(func(s *pngSpec) { s.trns = make([]byte, 257) })},
		{"png-frame-trns-dup", fr(func(s *pngSpec) { s.trns = []byte{1}; s.mid = [][]byte{chunk("tRNS", []byte{2})} })},
		{"png-frame-ga-trns", fr(func(s *pngSpec) {
			s.ctype = 4
			s.plte = 0
			s.sample = func(x, y int) []int { return []int{x, 255} }
			s.trns = []byte{0, 0}
		})},
		{"png-frame-rgb-trns-len", fr(func(s *pngSpec) {
			s.ctype = 2
			s.plte = 0
			s.sample = func(x, y int) []int { return []int{1, 2, 3} }
			s.trns = []byte{0, 0}
		})},
		{"png-frame-dup-ihdr", fr(func(s *pngSpec) { s.pre = [][]byte{chunk("IHDR", hdr(24, 24, 8, 3, 0, 0, 0))} })},
		{"png-frame-zlib-hdr", fr(func(s *pngSpec) { s.zMutate = func(z []byte) []byte { z[0] = 0x79; return z } })},
		{"png-frame-zlib-window", fr(func(s *pngSpec) { s.zMutate = func(z []byte) []byte { z[0], z[1] = 0x88, 0x98; return z } })},
		{"png-frame-zlib-fcheck", fr(func(s *pngSpec) { s.zMutate = func(z []byte) []byte { z[1]++; return z } })},
		{"png-frame-fdict-1", fr(func(s *pngSpec) {
			s.zMutate = func(z []byte) []byte { return append([]byte{0x78, 0x20, 0, 0, 0, 1}, z[2:]...) }
		})},
		{"png-frame-fdict-2", fr(func(s *pngSpec) {
			s.zMutate = func(z []byte) []byte { return append([]byte{0x78, 0x20, 0, 0, 0, 2}, z[2:]...) }
		})},
		{"png-frame-fdict-short", fr(func(s *pngSpec) { s.zMutate = func(z []byte) []byte { return []byte{0x78, 0x20, 0, 0} } })},
		{"png-frame-zlib-1byte", fr(func(s *pngSpec) { s.zMutate = func(z []byte) []byte { return z[:1] } })},
		{"png-frame-zlib-empty", fr(func(s *pngSpec) { s.zMutate = func(z []byte) []byte { return nil } })},
		{"png-frame-adler", fr(func(s *pngSpec) { s.zMutate = func(z []byte) []byte { z[len(z)-1] ^= 0x55; return z } })},
		{"png-frame-extra-row", fr(func(s *pngSpec) { s.rawMutate = func(r []byte) []byte { return append(r, make([]byte, 25)...) } })},
		{"png-frame-extra-byte", fr(func(s *pngSpec) { s.rawMutate = func(r []byte) []byte { return append(r, 0) } })},
		{"png-frame-extra-row-bad-adler", fr(func(s *pngSpec) {
			s.rawMutate = func(r []byte) []byte { return append(r, make([]byte, 25)...) }
			s.zMutate = func(z []byte) []byte { z[len(z)-1] ^= 1; return z }
		})},
		{"png-frame-missing-row", fr(func(s *pngSpec) { s.rawMutate = func(r []byte) []byte { return r[:len(r)-25] } })},
		{"png-frame-missing-row-bad-adler", fr(func(s *pngSpec) {
			s.rawMutate = func(r []byte) []byte { return r[:len(r)-25] }
			s.zMutate = func(z []byte) []byte { z[len(z)-2] ^= 1; return z }
		})},
		{"png-frame-trunc-zlib", fr(func(s *pngSpec) { s.zMutate = func(z []byte) []byte { return z[:len(z)/2] } })},
		{"png-frame-trunc-adler", fr(func(s *pngSpec) { s.zMutate = func(z []byte) []byte { return z[:len(z)-2] } })},
		{"png-frame-trunc-adler-full", fr(func(s *pngSpec) { s.zMutate = func(z []byte) []byte { return z[:len(z)-4] } })},
		{"png-frame-trunc-adler-eof", fr(func(s *pngSpec) { s.zMutate = func(z []byte) []byte { return z[:len(z)-2] }; s.noIDATCRC = true })},
		{"png-frame-trunc-file-in-idat", fr(func(s *pngSpec) {})[:len(fr(func(s *pngSpec) {}))-30]},
		{"png-frame-bad-filter", fr(func(s *pngSpec) { s.filter = func(r int) byte { return map[bool]byte{true: 5, false: 0}[r == 3] } })},
		{"png-frame-bad-filter-last", fr(func(s *pngSpec) { s.filter = func(r int) byte { return map[bool]byte{true: 255, false: 1}[r == 23] } })},
		{"png-frame-split-1", fr(func(s *pngSpec) { s.split = []int{1} })},
		{"png-frame-split-7", fr(func(s *pngSpec) { s.split = []int{7} })},
		{"png-frame-split-zero", fr(func(s *pngSpec) {
			s.split = []int{0, 3, 0, 0, 5, 0, 1000}
		})},
		{"png-frame-split-badcrc", fr(func(s *pngSpec) {
			s.split = []int{10}
			s.idatMutate = func(p [][]byte) [][]byte { return p }
		})},
		{"png-frame-junk-small", fr(func(s *pngSpec) {
			s.zMutate = func(z []byte) []byte { return append(z, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10) }
		})},
		{"png-frame-junk-big", fr(func(s *pngSpec) { s.zMutate = func(z []byte) []byte { return append(z, make([]byte, 5000)...) } })},
		{"png-frame-junk-exact-4096", fr(func(s *pngSpec) {
			s.zMutate = func(z []byte) []byte { return append(z, make([]byte, 4096-len(z))...) }
		})},
		{"png-frame-junk-4097", fr(func(s *pngSpec) {
			s.zMutate = func(z []byte) []byte { return append(z, make([]byte, 4097-len(z))...) }
		})},
		{"png-frame-junk-split", fr(func(s *pngSpec) {
			s.split = []int{len(baseZ), 5000}
			s.zMutate = func(z []byte) []byte { return append(z, make([]byte, 5000)...) }
		})},
		{"png-frame-junk-chunk", fr(func(s *pngSpec) { s.post = [][]byte{chunk("IDAT", []byte("garbage"))} })},
		{"png-frame-junk-chunk-badcrc", fr(func(s *pngSpec) { s.post = [][]byte{rawChunk(3, "IDAT", []byte("abc"), 7)} })},
		{"png-frame-unknown-crit", fr(func(s *pngSpec) { s.mid = [][]byte{chunk("ABCD", []byte("xyz"))} })},
		{"png-frame-unknown-anc-post", fr(func(s *pngSpec) { s.post = [][]byte{chunk("tEXt", []byte("k\x00v"))} })},
		{"png-frame-unknown-badcrc", fr(func(s *pngSpec) { s.mid = [][]byte{rawChunk(2, "zzzz", []byte("ab"), 1)} })},
		{"png-frame-bad-chunk-len", fr(func(s *pngSpec) { s.mid = [][]byte{rawChunk(0x80000000, "zzzz", nil, 0)} })},
		{"png-frame-big-unknown", fr(func(s *pngSpec) { s.mid = [][]byte{chunk("bIGc", make([]byte, 9000))} })},
		{"png-frame-trunc-unknown", func() []byte {
			b := []byte("\x89PNG\r\n\x1a\n")
			b = append(b, chunk("IHDR", hdr(24, 24, 8, 3, 0, 0, 0))...)
			return append(b, rawChunk(100, "zzzz", make([]byte, 40), 0)[:48]...)
		}()},
		{"png-frame-no-iend", fr(func(s *pngSpec) { s.noIEND = true })},
		{"png-frame-iend-len", fr(func(s *pngSpec) { s.noIEND = true; s.post = [][]byte{chunk("IEND", []byte{0})} })},
		{"png-frame-iend-badcrc", fr(func(s *pngSpec) { s.noIEND = true; s.post = [][]byte{rawChunk(0, "IEND", nil, 0)} })},
		{"png-frame-after-iend", fr(func(s *pngSpec) { s.tail = []byte("trailing bytes after IEND") })},
		{"png-frame-plte-after-idat", fr(func(s *pngSpec) { s.post = [][]byte{chunk("PLTE", make([]byte, 768))} })},
		{"png-frame-trns-after-idat", fr(func(s *pngSpec) { s.post = [][]byte{chunk("tRNS", []byte{1})} })},
		{"png-frame-ihdr-after-idat", fr(func(s *pngSpec) { s.post = [][]byte{chunk("IHDR", hdr(24, 24, 8, 3, 0, 0, 0))} })},
		{"png-frame-iend-before-idat", fr(func(s *pngSpec) { s.mid = [][]byte{chunk("IEND", nil)} })},
		{"png-frame-idat-crc", func() []byte {
			b := fr(func(s *pngSpec) {})
			b[len(b)-12-1] ^= 0xff
			return b
		}()},
		{"png-frame-stored", fr(func(s *pngSpec) { s.level = 100 })},
		{"png-frame-level1", fr(func(s *pngSpec) { s.level = 1 })},
		{"png-frame-level9-filters", fr(func(s *pngSpec) { s.level = 9; s.filter = func(r int) byte { return 4 } })},
		{"png-frame-huffman-only", fr(func(s *pngSpec) { s.level = zlib.HuffmanOnly })},
		{"png-frame-1x1-interlaced", fr(func(s *pngSpec) { s.w, s.h, s.interlace = 1, 1, 1 })},
		{"png-frame-3x2-interlaced", fr(func(s *pngSpec) { s.w, s.h, s.interlace = 3, 2, 1 })},
		{"png-frame-stored-junk-small", fr(func(s *pngSpec) {
			s.level = 100
			s.zMutate = func(z []byte) []byte { return append(z, 1, 2, 3) }
		})},
		{"png-frame-stored-junk-4097", fr(func(s *pngSpec) {
			s.level = 100
			s.zMutate = func(z []byte) []byte { return append(z, make([]byte, 4097-len(z))...) }
		})},
		{"png-frame-stored-junk-big", fr(func(s *pngSpec) {
			s.level = 100
			s.zMutate = func(z []byte) []byte { return append(z, make([]byte, 9000)...) }
		})},
	}
	// 分段後中間那一段的 CRC 破壞：在已產生的檔案中找第二個 IDAT。
	for i := range frames {
		if frames[i].kind == "png-frame-split-badcrc" {
			b := frames[i].data
			count := 0
			for p := 8; p+12 <= len(b); {
				n := int(binary.BigEndian.Uint32(b[p:]))
				if string(b[p+4:p+8]) == "IDAT" {
					count++
					if count == 2 {
						b[p+8+n] ^= 0xff
						break
					}
				}
				p += 12 + n
			}
		}
	}
	for _, f := range frames {
		e.fdshapFileCase(f.kind, "tile_0001/frame.png", f.data)
	}
	masks := []struct {
		kind string
		data []byte
	}{
		{"png-mask-gray1", mk(func(s *pngSpec) { s.depth = 1; s.sample = func(x, y int) []int { return []int{(x * y) % 2} } })},
		{"png-mask-gray2-binary", mk(func(s *pngSpec) { s.depth = 2; s.sample = func(x, y int) []int { return []int{3 * ((x + y) % 2)} } })},
		{"png-mask-gray2-mid", mk(func(s *pngSpec) { s.depth = 2; s.sample = func(x, y int) []int { return []int{(x + y) % 3} } })},
		{"png-mask-gray4", mk(func(s *pngSpec) { s.depth = 4; s.sample = func(x, y int) []int { return []int{15 * ((x ^ y) & 1)} } })},
		{"png-mask-gray4-mid", mk(func(s *pngSpec) { s.depth = 4; s.sample = func(x, y int) []int { return []int{(x ^ y) & 15} } })},
		{"png-mask-gray8-0x80", mk(func(s *pngSpec) {
			s.sample = func(x, y int) []int {
				if x == 20 && y == 20 {
					return []int{0x80}
				}
				return maskPattern(x, y)
			}
		})},
		{"png-mask-gray8-1", mk(func(s *pngSpec) { s.sample = func(x, y int) []int { return []int{x % 2} } })},
		{"png-mask-gray8-trns", mk(func(s *pngSpec) { s.trns = []byte{0, 0} })},
		{"png-mask-gray1-trns", mk(func(s *pngSpec) {
			s.depth = 1
			s.sample = func(x, y int) []int { return []int{x % 2} }
			s.trns = []byte{0, 1}
		})},
		{"png-mask-gray16", mk(func(s *pngSpec) { s.depth = 16; s.sample = func(x, y int) []int { return []int{65535 * (x % 2)} } })},
		{"png-mask-paletted", mk(func(s *pngSpec) { s.ctype = 3; s.plte = 256 })},
		{"png-mask-interlaced", mk(func(s *pngSpec) { s.interlace = 1; s.filter = func(r int) byte { return byte(r % 5) } })},
		{"png-mask-interlaced-gray2", mk(func(s *pngSpec) {
			s.interlace = 1
			s.depth = 2
			s.sample = func(x, y int) []int { return []int{3 * ((x * y) % 2)} }
			s.filter = func(r int) byte { return byte((r + 2) % 5) }
		})},
		{"png-mask-size", mk(func(s *pngSpec) { s.w = 25 })},
		{"png-mask-ga8", mk(func(s *pngSpec) { s.ctype = 4; s.sample = func(x, y int) []int { return []int{255, 255} } })},
	}
	for _, m := range masks {
		e.fdshapFileCase(m.kind, "tile_0002/mask.png", m.data)
	}
	e.fdshapFileCase("png-remap-rgb", "tile_0000/remap_mask.png", buildPNG(pngSpec{w: ns, h: ns, depth: 8, ctype: 2,
		sample: func(x, y int) []int { return []int{0, 0, 0} }}))
	e.fdshapFileCase("png-remap-nonbinary", "tile_0000/remap_mask.png", mk(func(s *pngSpec) {
		s.sample = func(x, y int) []int { return []int{y} }
	}))
	// 隨機單一位元組突變：一半修正 CRC（測到解壓縮與濾波），一半不修正。
	baseFramePNG := buildPNG(pngSpec{w: ns, h: ns, depth: 8, ctype: 3, plte: 256, sample: framePattern,
		filter: func(r int) byte { return byte(r % 5) }})
	baseMaskPNG := buildPNG(baseMask())
	for i := 0; i < 120; i++ {
		src, path := baseFramePNG, "tile_0001/frame.png"
		if i%2 == 1 {
			src, path = baseMaskPNG, "tile_0002/mask.png"
		}
		b := append([]byte(nil), src...)
		start := 8
		if i%4 >= 2 {
			// 集中在 IDAT 資料區。
			start = len(b) - 12 - 40
		}
		k := start + e.rng.IntN(len(b)-start)
		b[k] ^= byte(1 + e.rng.IntN(255))
		kind := "png-fuzz"
		if (i/4)%2 == 1 {
			b = idatCRCFix(b)
			kind = "png-fuzz-crcfix"
		}
		e.fdshapFileCase(kind, path, b)
	}
	// 檔案系統情況。
	e.trees["fdshap-missing-frame"] = treeSpec{Base: "fdshap-base", Remove: []string{"map_05/tile_0001/frame.png"}}
	e.loadFdshapCase("fs-missing", "fdshap-missing-frame", "", fdshapMap)
	e.trees["fdshap-dir-mask"] = treeSpec{Base: "fdshap-base", Remove: []string{"map_05/tile_0002/mask.png"},
		Dirs: []string{"map_05/tile_0002/mask.png"}}
	e.loadFdshapCase("fs-dir", "fdshap-dir-mask", "", fdshapMap)
	e.trees["fdshap-chmod-remap"] = treeSpec{Base: "fdshap-base", Chmod000: []string{"map_05/tile_0000/remap_mask.png"}}
	e.loadFdshapCase("fs-permission", "fdshap-chmod-remap", "", fdshapMap)
	e.trees["fdshap-chmod-bank"] = treeSpec{Base: "fdshap-base", Chmod000: []string{"map_05/bank.json"}}
	e.loadFdshapCase("fs-permission", "fdshap-chmod-bank", "", fdshapMap)
	e.trees["fdshap-dir-bank"] = treeSpec{Base: "fdshap-base", Remove: []string{"map_05/bank.json"},
		Dirs: []string{"map_05/bank.json"}}
	e.loadFdshapCase("fs-dir", "fdshap-dir-bank", "", fdshapMap)
	e.trees["fdshap-root-file"] = treeSpec{Files: [][2]string{{"notdir", e.blob([]byte("x"))}}}
	e.loadFdshapCase("fs-notdir", "fdshap-root-file", "/notdir", fdshapMap)
	for _, suffix := range []string{"/", "//", "/./", "/map_05/..", "/missing"} {
		e.loadFdshapCase("fs-suffix", "fdshap-base", suffix, fdshapMap)
	}
	for _, m := range []int{4, -1, fdicon.SeparatedFDSHAPMapCount, 0} {
		e.loadFdshapCase("fs-map-index", "fdshap-base", "", m)
	}
	e.add("structured", "load_fdshap_empty_root", obj{"map_index": 5},
		func() obj { _, _, err := fdicon.LoadSeparatedFDSHAPBank("", 5); return obj{"err": errv(err)} }())
}

// ---- 分離 FDICON ----

func fdiconSource() string {
	return `{"file":"FDICON.B24","resource":0,"size":624010,"md5":"46f793540209a063ea73a5373ca14bf4","sha256":"7efb4448d05f19c1e17ebd53f3e3afead235f5c008d5167548d834c3686b1e44","raw_size":624010}`
}

func fdiconDoc(sprites string) string {
	return `{"schema_version":1,"kind":"fdicon_sprite_bank","asset_id":"sprites/fdicon","status":"decoded","evidence":"confirmed","source":` +
		fdiconSource() + `,"sprites":` + sprites + `}`
}

func (e *exporter) genFdicon() {
	frames := [][]byte{buildPNG(baseFrame()), buildPNG(pngSpec{w: ns, h: ns, depth: 8, ctype: 3, plte: 256,
		sample: func(x, y int) []int { return []int{(x * y) & 0xff} }})}
	mask, remap := buildPNG(baseMask()), buildPNG(pngSpec{w: ns, h: ns, depth: 8, ctype: 0, sample: remapPattern})
	full := [][2]string{}
	head := [][2]string{}
	for i := 0; i < fdicon.SeparatedSpriteCount; i++ {
		d := fmt.Sprintf("sprite_%04d/", i)
		files := [][2]string{{d + "frame.png", e.blob(frames[i%2])}, {d + "mask.png", e.blob(mask)},
			{d + "remap_mask.png", e.blob(remap)}}
		full = append(full, files...)
		if i < 32 {
			head = append(head, files...)
		}
	}
	all := entries("sprite", fdicon.SeparatedSpriteCount)
	doc := fdiconDoc(marshal(all))
	bank := [2]string{"bank.json", e.blob([]byte(doc))}
	e.trees["fdicon-full"] = treeSpec{Files: append(full, bank)}
	e.trees["fdicon-head"] = treeSpec{Files: append(head, bank)}
	run := func(kind, tree, suffix string) {
		res := e.withTree(tree, func(root string) obj { return bankResult(fdicon.LoadSeparatedBank(root + suffix)) })
		e.add(kind, "load_bank", obj{"tree": tree, "suffix": suffix}, res)
	}
	variant := func(kind, base string, docText string, extra treeSpec) {
		name := fmt.Sprintf("fdicon-v%d", len(e.trees))
		extra.Base = base
		if docText != "" {
			extra.Files = append(extra.Files, [2]string{"bank.json", e.blob([]byte(docText))})
		}
		e.trees[name] = extra
		run(kind, name, "")
	}
	run("fdicon-valid", "fdicon-full", "")
	run("fdicon-suffix", "fdicon-full", "//")
	shuffled := append([]spriteEntry(nil), all...)
	e.rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	variant("fdicon-shuffled", "fdicon-full", fdiconDoc(marshal(shuffled)), treeSpec{})
	edit := func(f func([]spriteEntry) []spriteEntry) string {
		return fdiconDoc(marshal(f(append([]spriteEntry(nil), all...))))
	}
	variant("fdicon-contract", "fdicon-head", edit(func(s []spriteEntry) []spriteEntry { return s[:1679] }), treeSpec{})
	variant("fdicon-contract", "fdicon-head", edit(func(s []spriteEntry) []spriteEntry { return append(s, s[0]) }), treeSpec{})
	variant("fdicon-contract", "fdicon-head", strings.Replace(doc, `"md5":"46`, `"md5":"47`, 1), treeSpec{})
	variant("fdicon-contract", "fdicon-head", strings.Replace(doc, `"fdicon_sprite_bank"`, `"Fdicon_sprite_bank"`, 1), treeSpec{})
	variant("fdicon-contract", "fdicon-head", strings.Replace(doc, `"sprites/fdicon"`, `"sprites/FDICON"`, 1), treeSpec{})
	variant("fdicon-contract", "fdicon-head", strings.Replace(doc, `"schema_version":1`, `"schema_version":2`, 1), treeSpec{})
	variant("fdicon-json", "fdicon-head", strings.Replace(doc, `"size":624010`, `"size":"624010"`, 1), treeSpec{})
	variant("fdicon-contract", "fdicon-head", strings.Replace(doc, `"file":"FDICON.B24"`, `"file":"FDICON.b24"`, 1), treeSpec{})
	variant("fdicon-json", "fdicon-head", strings.Replace(doc, `"sprites":[{`, `"sprites":[{,`, 1), treeSpec{})
	variant("fdicon-json", "fdicon-head", strings.Replace(doc, `{"index":3,`, `{"index":[3],`, 1), treeSpec{})
	variant("fdicon-json", "fdicon-head", strings.Replace(doc, `"source":{"file"`, `"Source":{"File"`, 1), treeSpec{})
	variant("fdicon-sprite", "fdicon-head", edit(func(s []spriteEntry) []spriteEntry { s[10].Index = 1680; return s }), treeSpec{})
	variant("fdicon-sprite", "fdicon-head", edit(func(s []spriteEntry) []spriteEntry { s[5].Index = 4; return s }), treeSpec{})
	variant("fdicon-sprite", "fdicon-head", edit(func(s []spriteEntry) []spriteEntry { s[7].Width = 23; return s }), treeSpec{})
	variant("fdicon-sprite", "fdicon-head", edit(func(s []spriteEntry) []spriteEntry { s[8].Height = -24; return s }), treeSpec{})
	variant("fdicon-sprite", "fdicon-head", edit(func(s []spriteEntry) []spriteEntry { s[0].Index = -5; return s }), treeSpec{})
	variant("fdicon-sprite", "fdicon-head", edit(func(s []spriteEntry) []spriteEntry { s[9].Frame = "sprite_0009/frame.PNG"; return s }), treeSpec{})
	variant("fdicon-sprite", "fdicon-head", edit(func(s []spriteEntry) []spriteEntry { s[0].Mask = "sprite_0001/mask.png"; return s }), treeSpec{})
	variant("fdicon-sprite", "fdicon-head", edit(func(s []spriteEntry) []spriteEntry { s[3].RemapMask = "./sprite_0003/remap_mask.png"; return s }), treeSpec{})
	variant("fdicon-file", "fdicon-head", "", treeSpec{Remove: []string{"sprite_0007/frame.png"}})
	variant("fdicon-file", "fdicon-head", "", treeSpec{Files: [][2]string{{"sprite_0011/mask.png", e.blob(buildPNG(pngSpec{w: ns, h: ns,
		depth: 8, ctype: 0, sample: func(x, y int) []int { return []int{x * 10} }}))}}})
	variant("fdicon-file", "fdicon-full", "", treeSpec{Files: [][2]string{{"sprite_1679/remap_mask.png", e.blob(buildPNG(pngSpec{w: ns,
		h: ns, depth: 16, ctype: 0, sample: func(x, y int) []int { return []int{0} }}))}}})
	variant("fdicon-file", "fdicon-head", "", treeSpec{Files: [][2]string{{"sprite_0003/frame.png", e.blob(buildPNG(pngSpec{w: ns, h: ns,
		depth: 4, ctype: 3, plte: 16, sample: func(x, y int) []int { return []int{x % 16} }}))}}})
	variant("fdicon-file", "fdicon-head", "", treeSpec{Remove: []string{"sprite_0002/frame.png"}, Dirs: []string{"sprite_0002/frame.png"}})
	variant("fdicon-file", "fdicon-head", "", treeSpec{Chmod000: []string{"sprite_0004/mask.png"}})
	variant("fdicon-fs", "fdicon-head", "", treeSpec{Remove: []string{"bank.json"}})
	variant("fdicon-fs", "fdicon-head", "", treeSpec{Remove: []string{"bank.json"}, Dirs: []string{"bank.json"}})
	e.trees["fdicon-root-file"] = treeSpec{Files: [][2]string{{"notdir", e.blob([]byte("x"))}}}
	run("fdicon-fs", "fdicon-root-file", "/notdir")
	run("fs-lexical-dotdot", "fdicon-head", "/./missing/..")
	e.add("structured", "load_bank_empty_root", obj{}, func() obj { _, err := fdicon.LoadSeparatedBank(""); return obj{"err": errv(err)} }())
}

// ---- 真實原版檔 ----

func fileDigest(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return digest(b)
}

// treeDigest 是目錄下全部一般檔案（依路徑排序）的 "相對路徑\x00SHA-256\n" 串接後的 SHA-256。
func treeDigest(root string) (string, int) {
	var lines []string
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			panic(err)
		}
		if d.Type().IsRegular() {
			rel, _ := filepath.Rel(root, p)
			lines = append(lines, rel+"\x00"+fileDigest(p)+"\n")
		}
		return nil
	})
	sort.Strings(lines)
	return digest([]byte(strings.Join(lines, ""))), len(lines)
}

func (e *exporter) genReal() obj {
	real := obj{}
	clean := func(r obj, root string) obj {
		if err, ok := r["err"].(string); ok {
			r["err"] = strings.ReplaceAll(err, root, "$REAL")
		}
		return r
	}
	if e.flame2 != "" {
		path := filepath.Join(e.flame2, "FDICON.B24")
		real["fdicon_b24"] = obj{"path": "FDICON.B24", "sha256": fileDigest(path)}
		b, err := fdicon.DecodeFile(path)
		e.add("real-file", "decode_file", obj{"real": "FDICON.B24"}, clean(bankResult(b, err), e.flame2))
		if err == nil {
			for _, tc := range [][3]int{{4, 2, 1}, {4, 4, 0}, {139, 3, 2}, {140, 0, 0}} {
				s, err := b.SpriteFor(tc[0], tc[1], tc[2])
				r := obj{"err": errv(err)}
				if err == nil {
					r["digest"] = spriteDigest(s)
				}
				e.add("real-file", "real_sprite_for", obj{"group": tc[0], "pose": tc[1], "cycle": tc[2]}, r)
			}
		}
	}
	if e.assets != "" {
		bankRoot := filepath.Join(e.assets, "sprites", "fdicon")
		d, n := treeDigest(bankRoot)
		real["fdicon_pack"] = obj{"path": "sprites/fdicon", "files": n, "tree_sha256": d,
			"bank_json_sha256": fileDigest(filepath.Join(bankRoot, "bank.json"))}
		b, err := fdicon.LoadSeparatedBank(bankRoot)
		e.add("real-file", "load_bank", obj{"real": "sprites/fdicon"}, clean(bankResult(b, err), e.assets))
		shapRoot := filepath.Join(e.assets, "tilesets", "fdshap")
		d, n = treeDigest(shapRoot)
		real["fdshap_pack"] = obj{"path": "tilesets/fdshap", "files": n, "tree_sha256": d}
		for m := -1; m <= fdicon.SeparatedFDSHAPMapCount; m++ {
			b, controls, err := fdicon.LoadSeparatedFDSHAPBank(shapRoot, m)
			r := bankResult(b, err)
			if err == nil {
				r["controls"] = hexs(controls)
			}
			e.add("real-file", "load_fdshap", obj{"real": "tilesets/fdshap", "map_index": m}, clean(r, e.assets))
		}
	}
	// 不存在與目錄路徑。
	missing := filepath.Join(e.tmp, "nope", "FDICON.B24")
	_, err := fdicon.DecodeFile(missing)
	e.add("fs-decode", "decode_file", obj{"tmp": "nope/FDICON.B24"},
		obj{"err": strings.ReplaceAll(errv(err).(string), e.tmp, "$TMP")})
	_, err = fdicon.DecodeFile(e.tmp)
	e.add("fs-decode", "decode_file", obj{"tmp": ""}, obj{"err": strings.ReplaceAll(errv(err).(string), e.tmp, "$TMP")})
	return real
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	flame2 := flag.String("flame2", "", "含 FDICON.B24 的原版目錄（唯讀）")
	assets := flag.String("assets", "", "分離素材包根目錄（唯讀）")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "需要 -out")
		os.Exit(2)
	}
	tmp, err := os.MkdirTemp("", "fd2-golden-fdicon-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x46444943)), tmp: tmp, flame2: *flame2, assets: *assets,
		blobs: map[string]string{}, trees: map[string]treeSpec{}, banks: map[string][][3]string{},
		bankObjs: map[string]*fdicon.Bank{}}
	e.genScalars()
	e.genSelector()
	e.genBlits()
	e.genTerrain()
	e.genLayers()
	e.genParse()
	e.fdshapBase()
	e.genFdshapJSON()
	e.genFdshapPNG()
	e.genFdicon()
	real := e.genReal()
	doc := obj{"schema": "fd2-golden-fdicon/1", "seed": *seed, "real": real, "blobs": e.blobs, "trees": e.trees,
		"banks": e.banks, "cases": e.cases}
	data, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		panic(err)
	}
	kinds := map[string]int{}
	for _, c := range e.cases {
		kinds[c.Op]++
	}
	fmt.Printf("fd2-golden-fdicon: %d cases sha256=%s ops=%v\n", len(e.cases), digest(data), kinds)
}
