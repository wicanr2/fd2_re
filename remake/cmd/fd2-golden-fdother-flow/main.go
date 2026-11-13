// fd2-golden-fdother-flow 輸出 internal/fdother 第三組（依賴核心與原版流程邏輯的檔案：action_overlay、
// ch28_post_present、indexed_transition*、native_ai_item_type21_tail、native_ch23、native_command*_tail、
// native_command9、pending_code1_interlude、preparation_roster、spawn_intro、unit_present_schedule）的
// 黃金差分資料，供 Rust 改寫逐欄比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 fdother、fdicon 的公開 API，不改任何遊戲行為。
// 值域小的輸入（方向、相位、選擇器、掃描碼、游標、latch、tick 組合）逐一列舉，另含範圍內外的
// 邊界與 64 位元極值；其餘以固定種子的 PCG 產生。輸入與結果寫進同一份 JSON，Rust 端不需重現亂數。
// 大緩衝不嵌入 JSON：內容由 pat(n,k) 公式產生，結果只記 SHA-256。
// 真實原版檔與分離素材包也不嵌入，只記大小與 SHA-256（素材包記子目錄檔案清單摘要）。
//
// 每個案例的 compare 欄決定 Rust 端的比對方式：
//   - exact：結果逐欄相同，錯誤文字逐字相同。
//   - err_prefix：Go 錯誤包了標準函式庫錯誤（os、encoding/json、image/png）；Rust 必須同樣拒收，
//     而且 Go 自己的前綴（prefix 欄）相同，內層文字不比。
//
// 會被 64 位元繞回騙過範圍檢查而在執行期 panic 的輸入以 protect 記錄 panic，並保留 panic 之前已
// 寫入的緩衝區內容摘要。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/fs"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type obj = map[string]any

type caseJSON struct {
	Index   int     `json:"index"`
	Kind    string  `json:"kind"`
	Op      string  `json:"op"`
	Compare string  `json:"compare"`
	Prefix  *string `json:"prefix,omitempty"`
	Input   obj     `json:"input"`
	Result  obj     `json:"result"`
}

type exporter struct {
	rng    *rand.Rand
	tmp    string
	flame2 string
	assets string
	cases  []caseJSON
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hexs(b []byte) string { return hex.EncodeToString(b) }

func errv(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

// pat 產生可在 Rust 端重現的緩衝內容。
func pat(n, k int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(uint64(i)*uint64(k) + uint64(i>>8)*3 + uint64(k))
	}
	return b
}

func isStdlib(e error) bool {
	var pathErr *fs.PathError
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(e, &pathErr) && e == error(pathErr):
		return true
	case errors.As(e, &syntaxErr) && e == error(syntaxErr):
		return true
	case errors.As(e, &typeErr) && e == error(typeErr):
		return true
	}
	switch e.(type) {
	case png.FormatError, png.UnsupportedError:
		return true
	}
	if e == image.ErrFormat || e == io.ErrUnexpectedEOF || e == io.EOF {
		return true
	}
	msg := e.Error()
	return strings.HasPrefix(msg, "zlib:") || strings.HasPrefix(msg, "flate:") || strings.HasPrefix(msg, "json:") ||
		strings.HasPrefix(msg, "png:") || strings.HasPrefix(msg, "unexpected EOF")
}

// goPrefix 回傳 Go 錯誤中屬於本套件的前綴；沒有包標準函式庫錯誤時回傳 ok=false。
func goPrefix(err error) (string, bool) {
	full := err.Error()
	for e := err; e != nil; e = errors.Unwrap(e) {
		if isStdlib(e) {
			return strings.TrimSuffix(full, e.Error()), true
		}
	}
	return "", false
}

// protect 回報 Go 執行期 panic（64 位元繞回騙過範圍檢查後的越界切片等）。
func protect(f func() error) (err error, panicked bool) {
	defer func() {
		if recover() != nil {
			panicked = true
		}
	}()
	err = f()
	return
}

func (e *exporter) scrub(s string) string {
	s = strings.ReplaceAll(s, e.tmp, "$TMP")
	s = strings.ReplaceAll(s, e.assets, "$REAL")
	return strings.ReplaceAll(s, e.flame2, "$FLAME2")
}

func (e *exporter) add(kind, op string, input, result obj, err error) {
	compare := "exact"
	var prefix *string
	if err != nil {
		if p, ok := goPrefix(err); ok {
			p = e.scrub(p)
			compare, prefix = "err_prefix", &p
		}
	}
	if result == nil {
		result = obj{}
	}
	if _, ok := result["err"]; !ok {
		result["err"] = errv(err)
	}
	if s, ok := result["err"].(string); ok {
		result["err"] = e.scrub(s)
	}
	e.cases = append(e.cases, caseJSON{Index: len(e.cases), Kind: kind, Op: op, Compare: compare, Prefix: prefix,
		Input: input, Result: result})
}

var extremes = []int{math.MaxInt64, math.MinInt64, math.MaxInt64 - 10, math.MinInt64 + 10, 1 << 62, -(1 << 62),
	0x7fffffff, -0x80000000, 1 << 32}

func (e *exporter) pick(lo, hi int) int {
	if e.rng.IntN(25) == 0 {
		return extremes[e.rng.IntN(len(extremes))]
	}
	return lo + e.rng.IntN(hi-lo+1)
}

// ---- LMI1／RawCell 規格：[w, h, len, k]，像素是 pat(len, k)。 ----

func lmi(spec [4]int) fdother.LMI1Entry {
	return fdother.LMI1Entry{Width: spec[0], Height: spec[1], Pixels: pat(spec[2], spec[3])}
}

func (e *exporter) lmiSpec() [4]int {
	w, h := e.rng.IntN(5), e.rng.IntN(4)
	n := w * h
	if e.rng.IntN(6) == 0 {
		n += e.rng.IntN(3) - 1
		if n < 0 {
			n = 0
		}
	}
	return [4]int{w, h, n, e.rng.IntN(250) + 1}
}

// 合成 LMI 庫：n 個 1×1 有效項目（像素 byte(index)），bad 列出 [index, w, h, len] 的改寫。
func lmiBank(n int, bad [][4]int) []fdother.LMI1Entry {
	out := make([]fdother.LMI1Entry, n)
	for i := range out {
		out[i] = fdother.LMI1Entry{Width: 1, Height: 1, Pixels: []byte{byte(i)}}
	}
	for _, b := range bad {
		if b[0] < n {
			out[b[0]] = fdother.LMI1Entry{Width: b[1], Height: b[2], Pixels: make([]byte, b[3])}
		}
	}
	return out
}

func bankInput(n int, bad [][4]int) obj {
	if bad == nil {
		bad = [][4]int{}
	}
	return obj{"n": n, "bad": bad}
}

// ---- action_overlay ----

func cellResult(v int, err error) []any { return []any{v, errv(err)} }

func (e *exporter) overlayState(kind string, s fdother.ActionOverlayState) {
	cells := [][]any{}
	for d := -1; d <= 4; d++ {
		cells = append(cells, cellResult(s.CellIndex(d)))
	}
	selected := [][]any{}
	for d := -1; d <= 4; d++ {
		for phase := -1; phase <= 2; phase++ {
			selected = append(selected, cellResult(s.SelectedCellIndex(d, phase)))
		}
	}
	e.add(kind, "overlay_state", obj{"avail": s.Availability, "dir": s.DirectionState},
		obj{"cells": cells, "selected": selected}, nil)
}

func stateOut(s fdother.ActionOverlayState) []int {
	return append(append([]int{}, s.Availability[:]...), s.DirectionState[:]...)
}

func (e *exporter) genOverlay() {
	values := []int{-1, 0, 1, 2}
	for _, a0 := range values {
		for _, a1 := range values {
			for _, a2 := range values {
				for _, a3 := range values {
					avail := [4]int{a0, a1, a2, a3}
					e.add("enum-avail", "overlay_ctor", obj{"ctor": "battle", "avail": avail},
						obj{"state": stateOut(fdother.BattleActionOverlayState(avail))}, nil)
					e.add("enum-avail", "overlay_initial", obj{"avail": avail}, e.initialResult(avail), nil)
				}
			}
		}
	}
	for _, gate := range []bool{false, true} {
		for _, absent := range []bool{false, true} {
			e.add("enum-bool", "overlay_ctor", obj{"ctor": "nested", "gate": gate, "absent": absent},
				obj{"state": stateOut(fdother.NativeNestedSystemActionOverlayState(gate, absent))}, nil)
		}
	}
	e.add("fixed", "overlay_ctor", obj{"ctor": "continue"},
		obj{"state": stateOut(fdother.NativeContinueActionOverlayState())}, nil)
	words := []int{-1, 0, 1, 2, 3, 7, 24, 0x7fff}
	for _, a := range []int{-1, 0, 1, 2} {
		for _, d := range words {
			e.overlayState("enum-word", fdother.ActionOverlayState{
				Availability: [4]int{a, 0, 1, a}, DirectionState: [4]int{d, d + 1, 0, 3}})
		}
	}
	for _, x := range extremes {
		e.overlayState("edge-wrap", fdother.ActionOverlayState{Availability: [4]int{x, 0, 1, x / 2},
			DirectionState: [4]int{x, x / 3, 1, 0}})
	}
	for i := 0; i < 300; i++ {
		var s fdother.ActionOverlayState
		for d := 0; d < 4; d++ {
			s.Availability[d] = e.pick(-1, 2)
			s.DirectionState[d] = e.pick(-1, 30)
		}
		e.overlayState("random", s)
	}
	// NativeSystemOptions：四個 raw 位元組在 {0,1,2,255} 中逐一列舉。
	raw := []byte{0, 1, 2, 255}
	for _, a := range raw {
		for _, b := range raw {
			for _, c := range raw {
				for _, d := range raw {
					o := fdother.NativeSystemOptions{Raw53AF9: a, Raw51AAB: b, Raw51E61: c, Raw51E62: d}
					res := obj{"validate": errv(o.Validate()), "flags": []bool{o.MusicEnabled(), o.SFXEnabled(),
						o.FullPresentationEnabled(), o.HUDEnabled()}}
					st, err := o.ActionOverlayState()
					res["state"] = []any{stateOut(st), errv(err)}
					toggles := [][]any{}
					for sel := -1; sel <= 4; sel++ {
						t, err := o.Toggle(sel)
						toggles = append(toggles, []any{[]int{int(t.Raw53AF9), int(t.Raw51AAB), int(t.Raw51E61), int(t.Raw51E62)}, errv(err)})
					}
					res["toggles"] = toggles
					e.add("enum-bytes", "system_options", obj{"raw": []int{int(a), int(b), int(c), int(d)}}, res, nil)
				}
			}
		}
	}
	def := fdother.DefaultNativeSystemOptions()
	e.add("fixed", "system_default", obj{}, obj{"raw": []int{int(def.Raw53AF9), int(def.Raw51AAB), int(def.Raw51E61), int(def.Raw51E62)}}, nil)
	for _, a := range []int{-1, 0, 1, 2, math.MaxInt64} {
		for _, b := range []int{-1, 0, 1} {
			avail := [4]int{a, b, b, a}
			e.add("enum-avail-wide", "overlay_initial", obj{"avail": avail}, e.initialResult(avail), nil)
		}
	}
	frames := []int{-2, -1, 0, 1, 2, 3, 4, 5, math.MaxInt64, math.MinInt64}
	for _, f := range frames {
		for _, closing := range []bool{false, true} {
			o, err := fdother.ActionOverlayFrameOffsets(f, closing)
			e.add("enum-frame", "overlay_offsets", obj{"frame": f, "closing": closing}, obj{"offsets": o}, err)
		}
	}
	origin := func(kind string, col, row int) {
		a, errA := fdother.ActionOverlayOrigin(col, row)
		b, errB := fdother.ActionOverlaySnapshotOrigin(col, row)
		e.add(kind, "overlay_origin", obj{"col": col, "row": row},
			obj{"origin": []any{a, errv(errA)}, "snapshot": []any{b, errv(errB)}}, nil)
	}
	for col := -2; col <= 14; col++ {
		for row := -2; row <= 9; row++ {
			origin("enum-cursor", col, row)
		}
	}
	for _, x := range extremes {
		origin("edge-extreme", x, 1)
		origin("edge-extreme", 1, x)
	}
	e.genSnapshot()
	e.genOverlayBlit()
	e.genBlink()
}

func (e *exporter) initialResult(avail [4]int) obj {
	accepts := []bool{}
	for d := -2; d <= 5; d++ {
		accepts = append(accepts, fdother.ActionOverlayAcceptsDirection(avail, d))
	}
	return obj{"initial": fdother.ActionOverlayInitialDirection(avail), "accepts": accepts}
}

func (e *exporter) snapshotCapture(kind string, n, k, stride, x, y int) {
	src := pat(n, k)
	var snap []byte
	err, panicked := protect(func() error {
		var err error
		snap, err = fdother.CaptureActionOverlaySnapshot(src, stride, x, y)
		return err
	})
	res := obj{"err": errv(err), "panic": panicked}
	if err == nil && !panicked {
		res["snapshot"] = digest(snap)
	}
	e.add(kind, "snapshot_capture", obj{"n": n, "k": k, "stride": stride, "x": x, "y": y}, res, err)
}

func (e *exporter) snapshotRestore(kind string, n, k, sn, sk, stride, x, y int) {
	dst, snap := pat(n, k), pat(sn, sk)
	err, panicked := protect(func() error { return fdother.RestoreActionOverlaySnapshot(dst, snap, stride, x, y) })
	e.add(kind, "snapshot_restore", obj{"n": n, "k": k, "sn": sn, "sk": sk, "stride": stride, "x": x, "y": y},
		obj{"err": errv(err), "panic": panicked, "dst": digest(dst)}, err)
}

func (e *exporter) genSnapshot() {
	for i := 0; i < 400; i++ {
		stride := e.pick(60, 110)
		rows := e.rng.IntN(90)
		n := 0
		if stride > 0 && stride < 1000 {
			n = stride*rows + e.rng.IntN(5)
		}
		if e.rng.IntN(15) == 0 {
			n = 0
		}
		x, y := e.pick(-2, 40), e.pick(-2, 20)
		e.snapshotCapture("random", n, e.rng.IntN(200)+1, stride, x, y)
		sn := 72 * 72
		if e.rng.IntN(8) == 0 {
			sn += e.rng.IntN(3) - 1
		}
		e.snapshotRestore("random", n, e.rng.IntN(200)+1, sn, e.rng.IntN(200)+1, stride, x, y)
	}
	// 64 位元繞回：x 或 y 接近 int 上限時 x+72／y+72 繞成負值而通過檢查，之後在切片時 panic。
	for _, x := range []int{math.MaxInt64, math.MaxInt64 - 10, math.MaxInt64 - 71, math.MaxInt64 - 72} {
		e.snapshotCapture("edge-wrap", 100*100, 3, 100, x, 0)
		e.snapshotCapture("edge-wrap", 100*100, 3, 100, 0, x)
		e.snapshotRestore("edge-wrap", 100*100, 3, 72*72, 5, 100, x, 0)
		e.snapshotRestore("edge-wrap", 100*100, 3, 72*72, 5, 100, 0, x)
	}
	e.snapshotCapture("edge-stride", 100*100, 3, math.MaxInt64, 0, 0)
	e.snapshotRestore("edge-stride", 100*100, 3, 72*72, 5, math.MaxInt64, 0, 0)
	e.snapshotCapture("edge-exact", 72*72, 9, 72, 0, 0)
	e.snapshotRestore("edge-exact", 72*72, 9, 72*72, 11, 72, 0, 0)
}

func (e *exporter) overlayBlit(kind string, cellSpecs [][4]int, s fdother.ActionOverlayState, n, k, stride, origin,
	frame int, closing bool) {
	cells := make([]fdother.RawCell, len(cellSpecs))
	for i, c := range cellSpecs {
		cells[i] = fdother.RawCell{Width: c[0], Height: c[1], Pixels: pat(c[2], c[3])}
	}
	dst := pat(n, k)
	err, panicked := protect(func() error {
		return fdother.BlitActionOverlayFrame(cells, s, dst, stride, origin, frame, closing)
	})
	e.add(kind, "overlay_blit", obj{"cells": cellSpecs, "avail": s.Availability, "dir": s.DirectionState, "n": n,
		"k": k, "stride": stride, "origin": origin, "frame": frame, "closing": closing},
		obj{"err": errv(err), "panic": panicked, "dst": digest(dst)}, err)
}

func (e *exporter) genOverlayBlit() {
	for i := 0; i < 400; i++ {
		cells := make([][4]int, e.rng.IntN(80))
		for j := range cells {
			cells[j] = e.lmiSpec()
		}
		var s fdother.ActionOverlayState
		for d := 0; d < 4; d++ {
			s.Availability[d] = e.rng.IntN(3) - e.rng.IntN(2)
			s.DirectionState[d] = e.rng.IntN(26)
		}
		stride := 0x1c8
		if e.rng.IntN(6) == 0 {
			stride = e.pick(-1, 600)
		}
		n := e.rng.IntN(0x6000)
		origin := e.pick(-10, n+10)
		if e.rng.IntN(2) == 0 {
			origin = 0x2400 + e.rng.IntN(0x800)
		}
		e.overlayBlit("random", cells, s, n, e.rng.IntN(250)+1, stride, origin, e.pick(-1, 4), e.rng.IntN(2) == 0)
	}
	// 每個方向都能找到格的成功案例：75 個 1×1 到 3×2 格。
	full := make([][4]int, 80)
	for j := range full {
		w, h := 1+j%3, 1+j%2
		full[j] = [4]int{w, h, w * h, j + 1}
	}
	for frame := 0; frame < 4; frame++ {
		for _, closing := range []bool{false, true} {
			e.overlayBlit("enum-frame", full, fdother.BattleActionOverlayState([4]int{0, 1, 0, 1}), 0x6000, 7, 0x1c8,
				0x2800, frame, closing)
			e.overlayBlit("enum-frame", full, fdother.NativeContinueActionOverlayState(), 0x6000, 7, 0x1c8,
				0x2800, frame, closing)
		}
	}
}

func (e *exporter) genBlink() {
	for i := 0; i < 300; i++ {
		b := fdother.ActionOverlaySelectionBlink{Phase: e.pickFrom([]int{0, 0, 0, 1, 2, -1, 5}),
			Tick: e.pick(-5, 40)}
		start := []int{b.Phase, b.Tick}
		ticks := make([]int, 1+e.rng.IntN(12))
		states := [][2]int{}
		for j := range ticks {
			ticks[j] = e.pick(-6, 60)
			b.Advance(ticks[j])
			states = append(states, [2]int{b.Phase, b.Tick})
		}
		e.add("random", "blink", obj{"start": start, "ticks": ticks}, obj{"states": states}, nil)
	}
	for _, t := range extremes {
		b := fdother.ActionOverlaySelectionBlink{Phase: 1, Tick: t}
		ticks := []int{-t, t, t + 3, t + 4, math.MinInt64}
		states := [][2]int{}
		for _, x := range ticks {
			b.Advance(x)
			states = append(states, [2]int{b.Phase, b.Tick})
		}
		e.add("edge-wrap", "blink", obj{"start": []int{1, t}, "ticks": ticks}, obj{"states": states}, nil)
	}
	b := fdother.ActionOverlaySelectionBlink{Phase: math.MaxInt64}
	b.Advance(10)
	e.add("edge-wrap", "blink", obj{"start": []int{math.MaxInt64, 0}, "ticks": []int{10}},
		obj{"states": [][2]int{{b.Phase, b.Tick}}}, nil)
}

func (e *exporter) pickFrom(values []int) int { return values[e.rng.IntN(len(values))] }

// ---- ch28_post_present ----

func (e *exporter) genCh28() {
	frames := [][]any{}
	for _, f := range fdother.NativeCh28PostOverlayFrames() {
		frames = append(frames, []any{f.Phase, f.Entry})
	}
	e.add("fixed", "ch28_frames", obj{}, obj{"frames": frames}, nil)
	origin := func(kind string, x, y, cx, cy int) {
		o, err := fdother.NativeCh28PostOverlayOrigin(x, y, cx, cy)
		e.add(kind, "ch28_origin", obj{"x": x, "y": y, "cx": cx, "cy": cy}, obj{"origin": o}, err)
	}
	for x := -1; x <= 16; x++ {
		for y := -1; y <= 12; y++ {
			origin("enum-grid", x, y, 0, 0)
		}
	}
	for i := 0; i < 300; i++ {
		origin("random", e.pick(-5, 40), e.pick(-5, 30), e.pick(-5, 30), e.pick(-5, 25))
	}
	for _, x := range extremes {
		origin("edge-wrap", x, 1, 0, 0)
		origin("edge-wrap", 1, x, 0, 0)
		origin("edge-wrap", 1, 1, x, x)
	}
	for i := 0; i < 200; i++ {
		spec := e.lmiSpec()
		n := fdother.NativeCh28PostWorkSize
		if e.rng.IntN(8) == 0 {
			n = e.rng.IntN(0x30000)
		}
		origin := e.pick(-3, 0x25680+3)
		work := pat(n, 5)
		err, panicked := protect(func() error { return fdother.BlitNativeCh28PostOverlay(lmi(spec), work, origin) })
		e.add("random", "ch28_blit", obj{"entry": spec, "n": n, "origin": origin},
			obj{"err": errv(err), "panic": panicked, "work": digest(work)}, err)
	}
	for i := 0; i < 200; i++ {
		n := fdother.NativeCh28PostLMILast + 1 + e.rng.IntN(4) - 1
		if e.rng.IntN(6) == 0 {
			n = e.rng.IntN(0x50)
		}
		specs := make([][4]int, n)
		for j := range specs {
			specs[j] = [4]int{1, 1, 1, j + 1}
			if e.rng.IntN(30) == 0 {
				specs[j] = e.lmiSpec()
			}
		}
		entries := make([]fdother.LMI1Entry, n)
		for j := range specs {
			entries[j] = lmi(specs[j])
		}
		e.add("random", "ch28_validate", obj{"entries": specs}, obj{}, fdother.ValidateNativeCh28PostLMI(entries))
	}
}

// ---- indexed_transition / pass ----

type redrawScript struct {
	Mode string `json:"mode"` // nil、ok、err、mutate
	Pos  int    `json:"pos"`
	Val  int    `json:"val"`
}

func (e *exporter) redrawScript() redrawScript {
	switch e.rng.IntN(6) {
	case 0:
		return redrawScript{Mode: "nil"}
	case 1:
		return redrawScript{Mode: "err"}
	case 2, 3:
		return redrawScript{Mode: "mutate", Pos: e.rng.IntN(64), Val: e.rng.IntN(256)}
	}
	return redrawScript{Mode: "ok"}
}

func (e *exporter) radial() fdother.RadialLUTRemap {
	r := fdother.RadialLUTRemap{CenterX: e.pick(-1, 12), CenterY: e.pick(-1, 14), Radius: e.pick(0, 8),
		Scale: e.pick(-1, 20), StartY: e.pick(-1, 6), EndY: e.pick(-1, 14), ClipWidth: e.pick(0, 14)}
	if e.rng.IntN(30) == 0 {
		r.Radius = 0x8000
	}
	return r
}

func (e *exporter) rect() fdother.CenteredRectLUTRemap {
	return fdother.CenteredRectLUTRemap{CenterX: e.pick(-1, 12), HorizontalRadius: e.pick(-1, 10),
		StartY: e.pick(-1, 6), EndY: e.pick(-1, 14), ClipWidth: e.pick(0, 14)}
}

func radialOut(r fdother.RadialLUTRemap) []int {
	return []int{r.CenterX, r.CenterY, r.Radius, r.Scale, r.StartY, r.EndY, r.ClipWidth}
}

func rectOut(r fdother.CenteredRectLUTRemap) []int {
	return []int{r.CenterX, r.HorizontalRadius, r.StartY, r.EndY, r.ClipWidth}
}

func passOut(p fdother.IndexedTransitionPass) [][]int {
	return [][]int{radialOut(p.FirstRadial), radialOut(p.SecondRadial), rectOut(p.FinalRect)}
}

func (e *exporter) genTransition() {
	for i := 0; i < 1500; i++ {
		stride := e.pick(1, 14)
		n := e.rng.IntN(15) * 12
		dstK, lutK, lutN := e.rng.IntN(200)+1, e.rng.IntN(200)+1, 256
		if e.rng.IntN(15) == 0 {
			lutN = e.pickFrom([]int{0, 255, 257})
		}
		pass := fdother.IndexedTransitionPass{FirstRadial: e.radial(), SecondRadial: e.radial(), FinalRect: e.rect()}
		if e.rng.IntN(2) == 0 {
			// 偏向有效幾何：三組共用同一個 clip 與合法中心。
			clip := 1 + e.rng.IntN(12)
			stride = clip + e.rng.IntN(3)
			rows := 1 + e.rng.IntN(12)
			n = stride*rows + e.rng.IntN(stride)
			for _, r := range []*fdother.RadialLUTRemap{&pass.FirstRadial, &pass.SecondRadial} {
				r.ClipWidth, r.CenterX = clip, e.rng.IntN(clip)
				r.CenterY, r.Radius, r.Scale = e.rng.IntN(rows+2), 1+e.rng.IntN(6), e.rng.IntN(20)
				r.StartY = e.rng.IntN(rows + 1)
				r.EndY = r.StartY + e.rng.IntN(rows-r.StartY+1)
			}
			pass.FinalRect.ClipWidth, pass.FinalRect.CenterX = clip, e.rng.IntN(clip)
			pass.FinalRect.HorizontalRadius = e.rng.IntN(8)
			pass.FinalRect.StartY = e.rng.IntN(rows + 1)
			pass.FinalRect.EndY = pass.FinalRect.StartY + e.rng.IntN(rows-pass.FinalRect.StartY+1)
		}
		script := e.redrawScript()
		dst, lut := pat(n, dstK), pat(lutN, lutK)
		var redraw func([]byte) error
		calls := 0
		switch script.Mode {
		case "ok":
			redraw = func([]byte) error { calls++; return nil }
		case "err":
			redraw = func([]byte) error { calls++; return errors.New("synthetic redraw failure") }
		case "mutate":
			redraw = func(b []byte) error {
				calls++
				if script.Pos < len(b) {
					b[script.Pos] = byte(script.Val)
				}
				return nil
			}
		}
		err, panicked := protect(func() error { return fdother.ApplyIndexedTransitionPass(dst, stride, lut, pass, redraw) })
		e.add("random", "transition_pass", obj{"n": n, "k": dstK, "stride": stride, "lut_n": lutN, "lut_k": lutK,
			"pass": passOut(pass), "redraw": script},
			obj{"err": errv(err), "panic": panicked, "dst": hexs(dst), "calls": calls}, err)
	}
	build := func(kind string, a [5]int) {
		p, err := fdother.BuildNativeIndexedTransitionPass(a[0], a[1], a[2], a[3], a[4])
		res := obj{}
		if err == nil {
			res["pass"] = passOut(p)
		}
		e.add(kind, "transition_build", obj{"args": a}, res, err)
	}
	for _, cx := range []int{-1, 0, 311, 312} {
		for _, cy := range []int{-1, 0, 191, 200} {
			for _, r := range []int{-1, 0, 1, 10} {
				for _, sy := range []int{-1, 0, 192, 193} {
					for _, ey := range []int{-1, 0, 192, 193} {
						build("enum-edge", [5]int{cx, cy, r, sy, ey})
					}
				}
			}
		}
	}
	for i := 0; i < 300; i++ {
		build("random", [5]int{e.pick(-2, 320), e.pick(-2, 200), e.pick(-1, 0x8000), e.pick(-2, 195), e.pick(-2, 195)})
	}
	for _, x := range extremes {
		build("edge-wrap", [5]int{6, 6, x, 0, 192})
	}
}

// ---- 呈現排程 builder ----

func (e *exporter) genSchedules() {
	vertical := func(v [25]int) []int { return v[:] }
	type21 := func(kind string, id int, effectN int, effectBad [][4]int, digitN int) {
		s, err := fdother.BuildNativeAIItemType21TailSchedule(id, lmiBank(effectN, effectBad), lmiBank(digitN, nil))
		e.add(kind, "type21", obj{"id": id, "effect": bankInput(effectN, effectBad), "digits": bankInput(digitN, nil)},
			obj{"s": []int{s.CommandID, s.EffectResource, s.EffectStart, s.EffectFrames, s.SoundResource, s.EffectSample,
				s.EffectDelayTicks, s.ToggleA, s.ToggleB, s.TogglePairs, s.ToggleDelayMS, s.DigitResource, s.DigitBias,
				s.DigitFrames, s.DigitDelayMS, s.DigitHoldMS}, "v": vertical(s.DigitVertical)}, err)
	}
	for id := -1; id <= 40; id++ {
		type21("enum-command", id, 0x4c, nil, 0x68)
	}
	for _, x := range []int{math.MaxInt64, math.MinInt64} {
		type21("edge-command", x, 0x4c, nil, 0x68)
	}
	for _, id := range []int{1, 6, 7} {
		for _, n := range []int{0, 0x39, 0x4a, 0x4b, 0x4c, 0x80} {
			type21("enum-len", id, n, nil, 0x68)
		}
		for _, n := range []int{0, 0x66, 0x67, 0x68} {
			type21("enum-len", id, 0x4c, nil, n)
		}
		for i := 0; i < 40; i++ {
			bad := [][4]int{{0x30 + e.rng.IntN(0x1c), e.rng.IntN(3), e.rng.IntN(3), e.rng.IntN(5)}}
			type21("random-bad", id, 0x4c, bad, 0x68)
		}
	}
	command32 := func(kind string, effectN int, effectBad [][4]int, digitN int) {
		s, err := fdother.BuildNativeCommand32TailSchedule(lmiBank(effectN, effectBad), lmiBank(digitN, nil))
		e.add(kind, "command32", obj{"effect": bankInput(effectN, effectBad), "digits": bankInput(digitN, nil)},
			obj{"s": []int{s.EffectResource, s.EffectStart, s.EffectFrames, s.SoundResource, s.EffectSample,
				s.EffectDelayTicks, s.ToggleA, s.ToggleB, s.TogglePairs, s.ToggleDelayMS, s.DigitResource, s.DigitBias,
				s.DigitFrames, s.DigitDelayMS, s.DigitHoldMS}, "v": vertical(s.DigitVertical)}, err)
	}
	for _, n := range []int{0, 0x40, 0x4b, 0x4c, 0x100} {
		for _, d := range []int{0, 0x76, 0x77, 0x100} {
			command32("enum-len", n, nil, d)
		}
	}
	for i := 0; i < 60; i++ {
		command32("random-bad", 0x4c, [][4]int{{0x3e + e.rng.IntN(0x10), e.rng.IntN(3), e.rng.IntN(3), e.rng.IntN(5)}}, 0x77)
	}
	command9 := func(kind string, effectN int, effectBad [][4]int, resultN int, resultBad [][4]int) {
		s, err := fdother.BuildNativeCommand9PlayerSchedule(lmiBank(effectN, effectBad), lmiBank(resultN, resultBad))
		e.add(kind, "command9", obj{"effect": bankInput(effectN, effectBad), "result": bankInput(resultN, resultBad)},
			obj{"s": []int{s.EffectResource, s.EffectStart, s.EffectFrames, s.SoundResource, s.InitialSample,
				s.RepeatSample, s.RepeatSampleFrames[0], s.RepeatSampleFrames[1], s.EffectFrameDelayTicks,
				s.ResultResource, s.ResultDigitBias, s.ResultMissDescriptors[0], s.ResultMissDescriptors[1],
				s.ResultMissDescriptors[2], s.ResultMissDescriptors[3], s.ResultFrames, s.ResultDelayMS,
				s.ResultHoldMS}, "v": vertical(s.ResultVertical)}, err)
	}
	for _, n := range []int{0, 113, 114, 230} {
		for _, r := range []int{0, 94, 104, 117, 118, 119, 138} {
			command9("enum-len", n, nil, r, nil)
		}
	}
	for i := 0; i < 80; i++ {
		bad := [][4]int{{80 + e.rng.IntN(40), e.rng.IntN(3), e.rng.IntN(3), e.rng.IntN(5)}}
		rbad := [][4]int{{e.pickFrom([]int{93, 94, 99, 103, 104, 115, 116, 117, 118}), e.rng.IntN(3), e.rng.IntN(3),
			e.rng.IntN(5)}}
		if e.rng.IntN(2) == 0 {
			command9("random-bad", 230, bad, 138, nil)
		} else {
			command9("random-bad", 230, nil, 138, rbad)
		}
	}
	stage34 := func(rows []fdother.NativeCommand34StageSchedule) [][]any {
		out := [][]any{}
		for _, r := range rows {
			extra := r.ExtraSampleFrameIndices
			if extra == nil {
				extra = []int{}
			}
			out = append(out, []any{r.CommandID, r.EffectStart, r.EffectFrames, r.EffectSample, extra, r.MaskIndex,
				r.MaskPairs, r.DigitBias, r.DigitFrames, r.DigitHoldMilliseconds, vertical(r.DigitVertical)})
		}
		return out
	}
	rows, err := fdother.BuildNativeCommand2022TailSchedule()
	e.add("fixed", "command2022", obj{}, obj{"rows": stage34(rows)}, err)
	heal, err := fdother.BuildNativeCommand33TailSchedule()
	e.add("fixed", "command33", obj{}, obj{"s": []int{heal.EffectResource, heal.EffectStart, heal.EffectFrames,
		heal.EffectSample, heal.EffectFrameDelayTicks, heal.MaskSample, heal.MaskIndex, heal.MaskPairs,
		heal.DigitResource, heal.DigitBias, heal.DigitFrames, heal.DigitFrameDelayMs, heal.DigitHoldMs},
		"v": vertical(heal.DigitVertical)}, err)
	rows35, err := fdother.BuildNativeCommand35TailSchedule()
	out35 := [][]any{}
	for _, r := range rows35 {
		extra := r.ExtraSampleFrameIndices
		if extra == nil {
			extra = []int{}
		}
		out35 = append(out35, []any{r.CommandID, r.EffectStart, r.EffectFrames, r.EffectSample, extra, r.ExtraSample,
			r.MaskIndex, r.MaskPairs, r.DigitBias, r.DigitFrames, r.DigitHoldMilliseconds, vertical(r.DigitVertical)})
	}
	e.add("fixed", "command35", obj{}, obj{"rows": out35}, err)
	plan := [][]any{}
	for _, s := range fdother.NativePendingCode1PresentationPlan() {
		plan = append(plan, []any{string(s.Kind), s.Frame, s.Count, s.DurationMS})
	}
	e.add("fixed", "pending_code1", obj{}, obj{"plan": plan}, nil)
}

// ---- spawn_intro ----

func spawnOut(steps []fdother.NativeSpawnIntroStep) [][]any {
	out := [][]any{}
	for _, s := range steps {
		out = append(out, []any{s.Pass, s.LMIEntry, s.SoundResource, s.DelayTicks, int(s.SnapshotMode), s.RedrawTerrain,
			s.NewUnitPixelLift})
	}
	return out
}

func (e *exporter) genSpawn() {
	e.add("fixed", "spawn_schedule", obj{}, obj{"steps": spawnOut(fdother.NativeSpawnIntroSchedule())}, nil)
	validate := func(kind string, steps []fdother.NativeSpawnIntroStep) {
		e.add(kind, "spawn_validate", obj{"steps": spawnOut(steps)}, obj{},
			fdother.ValidateNativeSpawnIntroSchedule(steps))
	}
	base := fdother.NativeSpawnIntroSchedule()
	validate("fixed", base)
	for n := 0; n <= 13; n++ {
		steps := make([]fdother.NativeSpawnIntroStep, 0, n)
		for i := 0; i < n; i++ {
			if i < len(base) {
				steps = append(steps, base[i])
			} else {
				steps = append(steps, base[len(base)-1])
			}
		}
		validate("enum-len", steps)
	}
	for i := range base {
		for field := 0; field < 7; field++ {
			steps := fdother.NativeSpawnIntroSchedule()
			s := &steps[i]
			switch field {
			case 0:
				s.Pass++
			case 1:
				s.LMIEntry--
			case 2:
				s.SoundResource = 95 - s.SoundResource
			case 3:
				s.DelayTicks = 0
			case 4:
				s.SnapshotMode = fdother.NativeSpawnIntroSnapshotMode((int(s.SnapshotMode) + 1 + e.rng.IntN(254)) % 256)
			case 5:
				s.RedrawTerrain = !s.RedrawTerrain
			case 6:
				s.NewUnitPixelLift -= 1 + e.rng.IntN(3)
			}
			validate("enum-field", steps)
		}
	}
	visible := func(kind string, a [6]int) {
		e.add(kind, "spawn_visible", obj{"a": a},
			obj{"v": fdother.NativeSpawnIntroVisible(a[0], a[1], a[2], a[3], a[4], a[5])}, nil)
	}
	for x := 6; x <= 25; x++ {
		for y := 17; y <= 30; y++ {
			visible("enum-grid", [6]int{x, y, 10, 20, 12, 7})
		}
	}
	for i := 0; i < 300; i++ {
		visible("random", [6]int{e.pick(-5, 40), e.pick(-5, 40), e.pick(-5, 30), e.pick(-5, 30), e.pick(-2, 14),
			e.pick(-2, 9)})
	}
	for _, x := range extremes {
		visible("edge-wrap", [6]int{x, 0, 1, 1, 12, 7})
		visible("edge-wrap", [6]int{0, 0, x, x, x, x})
		visible("edge-wrap", [6]int{x, x, x, x, 12, 7})
	}
	originCase := func(kind string, a [4]int) {
		e.add(kind, "spawn_origin", obj{"a": a}, obj{"o": fdother.NativeSpawnIntroLMIOrigin(a[0], a[1], a[2], a[3])}, nil)
	}
	for i := 0; i < 200; i++ {
		originCase("random", [4]int{e.pick(-5, 40), e.pick(-5, 40), e.pick(-5, 30), e.pick(-5, 30)})
	}
	for _, x := range extremes {
		originCase("edge-wrap", [4]int{x, 1, 0, 0})
		originCase("edge-wrap", [4]int{1, x, 0, 0})
	}
	for i := 0; i < 300; i++ {
		spec := e.lmiSpec()
		if e.rng.IntN(3) == 0 {
			w := 40 + e.rng.IntN(30)
			spec = [4]int{w, 24, w * 24, e.rng.IntN(200) + 1}
		}
		n := 0x25680
		if e.rng.IntN(6) == 0 {
			n = e.rng.IntN(0x30000)
		}
		a := [4]int{e.pick(-3, 30), e.pick(-3, 30), e.pick(-3, 20), e.pick(-3, 20)}
		work := pat(n, 9)
		err, panicked := protect(func() error {
			return fdother.BlitNativeSpawnIntroLMI(lmi(spec), work, a[0], a[1], a[2], a[3])
		})
		e.add("random", "spawn_blit", obj{"entry": spec, "n": n, "a": a},
			obj{"err": errv(err), "panic": panicked, "work": digest(work)}, err)
	}
}

// ---- unit_present_schedule ----

func unitStepsOut(steps []fdother.UnitPresentStep) [][]any {
	out := [][]any{}
	for _, s := range steps {
		out = append(out, []any{s.Phase, s.Resource, s.Entry, s.DelayMs, s.DelayTicks})
	}
	return out
}

func lutFrameOut(f fdother.UnitPresentLUTFrame) []any {
	return []any{f.Phase, f.LUTIndex, passOut(f.Pass), f.DelayMs, f.DelayTicks}
}

type tickScript struct {
	Mode string `json:"mode"` // nil、ok、err、mutate
	At   int    `json:"at"`   // err：第幾次呼叫回傳錯誤
	Pos  int    `json:"pos"`
	Val  int    `json:"val"`
}

func (e *exporter) tickScript(maxAt, maxPos int) tickScript {
	switch e.rng.IntN(6) {
	case 0:
		return tickScript{Mode: "nil"}
	case 1:
		return tickScript{Mode: "err", At: e.rng.IntN(maxAt)}
	case 2, 3:
		return tickScript{Mode: "mutate", Pos: e.rng.IntN(maxPos), Val: e.rng.IntN(256)}
	}
	return tickScript{Mode: "ok"}
}

func (e *exporter) genUnitPresent() {
	steps := fdother.NativeUnitPresentSchedule()
	e.add("fixed", "unit_schedule", obj{}, obj{"steps": unitStepsOut(steps)}, nil)
	validate := func(kind string, steps []fdother.UnitPresentStep) {
		e.add(kind, "unit_validate", obj{"steps": unitStepsOut(steps)}, obj{},
			fdother.ValidateNativeUnitPresentSchedule(steps))
	}
	validate("fixed", steps)
	for _, n := range []int{0, 6, 11, 16, 26, 28} {
		s := fdother.NativeUnitPresentSchedule()
		if n <= len(s) {
			s = s[:n]
		} else {
			s = append(s, s[len(s)-1])
		}
		validate("enum-len", s)
	}
	for i := range steps {
		for field := 0; field < 5; field++ {
			s := fdother.NativeUnitPresentSchedule()
			switch field {
			case 0:
				s[i].Phase = e.pickStr([]string{"", "lmi_intro", "lut_contract", "lut_release", "LUT_release"})
			case 1:
				s[i].Resource ^= 5
			case 2:
				s[i].Entry++
			case 3:
				s[i].DelayMs += 10
			case 4:
				s[i].DelayTicks = 1 - s[i].DelayTicks
			}
			validate("enum-field", s)
		}
	}
	for i := 0; i < 200; i++ {
		a := [4]int{e.pick(-5, 40), e.pick(-5, 40), e.pick(-5, 30), e.pick(-5, 30)}
		e.add("random", "unit_origin", obj{"a": a},
			obj{"o": fdother.NativeUnitPresentByteOrigin(a[0], a[1], a[2], a[3])}, nil)
	}
	for _, x := range extremes {
		for _, a := range [][4]int{{x, 0, 0, 0}, {0, x, 0, 0}, {0, 0, x, x}} {
			e.add("edge-wrap", "unit_origin", obj{"a": a},
				obj{"o": fdother.NativeUnitPresentByteOrigin(a[0], a[1], a[2], a[3])}, nil)
		}
	}
	startY := func(kind string, raw, lut int) {
		v, err := fdother.NativeUnitPresentContractStartY(raw, lut)
		e.add(kind, "unit_start_y", obj{"raw": raw, "lut": lut}, obj{"v": v}, err)
	}
	for raw := -3; raw <= 12; raw++ {
		for lut := -1; lut <= 6; lut++ {
			startY("enum-raw-lut", raw, lut)
		}
	}
	for _, x := range extremes {
		startY("edge-wrap", x, 5)
		startY("edge-wrap", x, 3)
	}
	lutPass := func(kind string, cx, cy, sy int) {
		p, err := fdother.NativeUnitPresentLUTPass(cx, cy, sy)
		res := obj{}
		if err == nil {
			res["pass"] = passOut(p)
		}
		e.add(kind, "unit_lut_pass", obj{"a": []int{cx, cy, sy}}, res, err)
	}
	for _, cx := range []int{-1, 0, 12, 311, 312} {
		for _, cy := range []int{-1, 0, 15, 191, 192, 193} {
			for _, sy := range []int{-1, 0, 15, 191, 192, 193} {
				lutPass("enum-edge", cx, cy, sy)
			}
		}
	}
	for i := 0; i < 200; i++ {
		lutPass("random", e.pick(-3, 320), e.pick(-3, 200), e.pick(-3, 200))
	}
	lutFrames := func(kind string, a, b int) {
		frames, err := fdother.NativeUnitPresentLUTFrames(a, b)
		out := [][]any{}
		for _, f := range frames {
			out = append(out, lutFrameOut(f))
		}
		e.add(kind, "unit_lut_frames", obj{"a": a, "b": b}, obj{"frames": out}, err)
	}
	for a := -2; a <= 14; a++ {
		for b := -2; b <= 9; b++ {
			lutFrames("enum-raw", a, b)
		}
	}
	for _, x := range extremes {
		lutFrames("edge-wrap", x, 2)
		lutFrames("edge-wrap", 3, x)
	}
	vals := []int{-1, 0, 0xff, 0x100}
	for _, a := range vals {
		for _, b := range vals {
			for _, c := range vals {
				for _, d := range vals {
					for _, f := range vals {
						call, err := fdother.PlanNativeUnitPresentCall(a, b, c, d, f)
						e.add("enum-byte", "unit_plan", obj{"a": []int{a, b, c, d, f}},
							obj{"call": []int{call.UnitSlot, int(call.NewX), int(call.NewY), int(call.VisualX),
								int(call.VisualY)}}, err)
					}
				}
			}
		}
	}
	for i := 0; i < 100; i++ {
		a := []int{e.pick(-1, 256), e.pick(-1, 256), e.pick(-1, 256), e.pick(-1, 256), e.pick(-1, 256)}
		call, err := fdother.PlanNativeUnitPresentCall(a[0], a[1], a[2], a[3], a[4])
		e.add("random", "unit_plan", obj{"a": a}, obj{"call": []int{call.UnitSlot, int(call.NewX), int(call.NewY),
			int(call.VisualX), int(call.VisualY)}}, err)
	}
	for i := 0; i < 300; i++ {
		spec := e.lmiSpec()
		n := 0x1c8 * (180 + e.rng.IntN(30))
		if e.rng.IntN(8) == 0 {
			n = e.rng.IntN(0x30000)
		}
		a := [4]int{e.pick(-3, 30), e.pick(-3, 30), e.pick(-3, 20), e.pick(-3, 20)}
		dst := pat(n, 13)
		err, panicked := protect(func() error {
			return fdother.BlitNativeUnitPresentLMI(lmi(spec), dst, a[0], a[1], a[2], a[3])
		})
		e.add("random", "unit_blit", obj{"entry": spec, "n": n, "a": a},
			obj{"err": errv(err), "panic": panicked, "dst": digest(dst)}, err)
	}
	e.genUnitIntro()
	e.genUnitLUTFrame()
}

func (e *exporter) pickStr(values []string) string { return values[e.rng.IntN(len(values))] }

func (e *exporter) unitIntro(kind string, specs [][4]int, n int, a [4]int, script tickScript, real []fdother.LMI1Entry,
	realName string) {
	entries := real
	if entries == nil {
		entries = make([]fdother.LMI1Entry, len(specs))
		for i := range specs {
			entries[i] = lmi(specs[i])
		}
	}
	dst := pat(n, 21)
	calls := 0
	var tick func() error
	switch script.Mode {
	case "ok":
		tick = func() error { calls++; return nil }
	case "err":
		tick = func() error {
			calls++
			if calls-1 == script.At {
				return errors.New("synthetic present failure")
			}
			return nil
		}
	case "mutate":
		tick = func() error {
			calls++
			if script.Pos < len(dst) {
				dst[script.Pos] = byte(script.Val + calls)
			}
			return nil
		}
	}
	err, panicked := protect(func() error {
		return fdother.RunNativeUnitPresentLMIIntro(entries, dst, a[0], a[1], a[2], a[3], tick)
	})
	input := obj{"entries": specs, "n": n, "a": a, "tick": script}
	if realName != "" {
		input = obj{"real": realName, "n": n, "a": a, "tick": script}
	}
	e.add(kind, "unit_lmi_intro", input, obj{"err": errv(err), "panic": panicked, "calls": calls, "dst": digest(dst)}, err)
}

func (e *exporter) genUnitIntro() {
	for i := 0; i < 200; i++ {
		n := 0x7d + e.rng.IntN(3) - 1
		if e.rng.IntN(8) == 0 {
			n = e.rng.IntN(0x7d)
		}
		specs := make([][4]int, n)
		for j := range specs {
			specs[j] = [4]int{1 + j%3, 1 + j%2, (1 + j%3) * (1 + j%2), j + 1}
			if e.rng.IntN(40) == 0 {
				specs[j] = e.lmiSpec()
			}
		}
		dstN := 0x1c8 * 200
		if e.rng.IntN(8) == 0 {
			dstN = e.rng.IntN(0x10000)
		}
		a := [4]int{e.pick(10, 26), e.pick(10, 26), e.pick(5, 20), e.pick(5, 20)}
		e.unitIntro("random", specs, dstN, a, e.tickScript(12, dstN+1), nil, "")
	}
}

func (e *exporter) unitLUTFrame(kind string, frame fdother.UnitPresentLUTFrame, workN, workK, snapN, snapK, lutN,
	lutK int, redrawS, presentS tickScript) {
	work, snap, lut := pat(workN, workK), pat(snapN, snapK), pat(lutN, lutK)
	calls := []string{}
	var redraw func([]byte) error
	var present func([]byte, fdother.UnitPresentLUTFrame) error
	mk := func(s tickScript, name string) func(b []byte) error {
		n := 0
		return func(b []byte) error {
			n++
			calls = append(calls, fmt.Sprintf("%s:%d:%d", name, len(b), b[0]))
			if s.Mode == "err" {
				return errors.New("synthetic " + name + " failure")
			}
			if s.Mode == "mutate" && s.Pos < len(b) {
				b[s.Pos] = byte(s.Val + n)
			}
			return nil
		}
	}
	if redrawS.Mode != "nil" {
		redraw = mk(redrawS, "redraw")
	}
	if presentS.Mode != "nil" {
		f := mk(presentS, "present")
		present = func(b []byte, fr fdother.UnitPresentLUTFrame) error {
			calls = append(calls, fmt.Sprintf("frame:%s:%d", fr.Phase, fr.LUTIndex))
			return f(b)
		}
	}
	err, panicked := protect(func() error {
		return fdother.RunNativeUnitPresentLUTFrame(work, snap, lut, frame, redraw, present)
	})
	e.add(kind, "unit_lut_frame", obj{"frame": lutFrameOut(frame), "work_n": workN, "work_k": workK, "snap_n": snapN,
		"snap_k": snapK, "lut_n": lutN, "lut_k": lutK, "redraw": redrawS, "present": presentS},
		obj{"err": errv(err), "panic": panicked, "calls": calls, "work": digest(work)}, err)
}

func (e *exporter) genUnitLUTFrame() {
	const full = 0x25680
	for i := 0; i < 120; i++ {
		a, b := e.rng.IntN(13), e.rng.IntN(8)
		frames, err := fdother.NativeUnitPresentLUTFrames(a, b)
		if err != nil {
			continue
		}
		frame := frames[e.rng.IntN(len(frames))]
		workN, snapN, lutN := full, full, 256
		switch e.rng.IntN(10) {
		case 0:
			workN = full - 1
		case 1:
			snapN = full + 1
		case 2:
			lutN = 255
		}
		r, p := e.tickScript(1, full), e.tickScript(1, full)
		if r.Mode == "err" && e.rng.IntN(2) == 0 {
			r.Mode = "mutate"
		}
		e.unitLUTFrame("random", frame, workN, e.rng.IntN(200)+1, snapN, e.rng.IntN(200)+1, lutN, e.rng.IntN(200)+1, r, p)
	}
	// 幾何不合法的幀：處理階段的錯誤會被包上 unit-present 前綴。
	bad := fdother.UnitPresentLUTFrame{Phase: "lut_contract", LUTIndex: 3, Pass: fdother.IndexedTransitionPass{
		FirstRadial:  fdother.RadialLUTRemap{CenterX: 400, CenterY: 10, Radius: 11, Scale: 16, EndY: 192, ClipWidth: 312},
		SecondRadial: fdother.RadialLUTRemap{CenterX: 10, CenterY: 10, Radius: 11, Scale: 16, EndY: 192, ClipWidth: 312},
		FinalRect:    fdother.CenteredRectLUTRemap{CenterX: 10, HorizontalRadius: 17, EndY: 10, ClipWidth: 312}}}
	e.unitLUTFrame("edge-geometry", bad, full, 3, full, 5, 256, 7, tickScript{Mode: "ok"}, tickScript{Mode: "ok"})
	bad.Pass.FirstRadial.CenterX = 10
	bad.Pass.FinalRect.EndY = 300
	e.unitLUTFrame("edge-geometry", bad, full, 3, full, 5, 256, 7, tickScript{Mode: "ok"}, tickScript{Mode: "ok"})
	bad.Pass.FinalRect.EndY = 10
	bad.Pass.SecondRadial.Radius = 0
	e.unitLUTFrame("edge-geometry", bad, full, 3, full, 5, 256, 7, tickScript{Mode: "ok"}, tickScript{Mode: "ok"})
}

// ---- native_ch23 ----

type frameSpec struct {
	X    int    `json:"x"`
	Y    int    `json:"y"`
	W    int    `json:"w"`
	H    int    `json:"h"`
	Raw  string `json:"raw,omitempty"` // 原始 0x4e63d 位元組（十六進位）
	SepK int    `json:"sep_k,omitempty"`
}

func (f frameSpec) frame() fdother.Frame {
	if f.SepK != 0 {
		n := f.W * f.H
		if n < 0 {
			n = 0
		}
		mask := pat(n, f.SepK+1)
		for i := range mask {
			if mask[i]&1 == 0 {
				mask[i] = 0
			} else {
				mask[i] = 255
			}
		}
		return fdother.Frame{X: f.X, Y: f.Y, Width: f.W, Height: f.H, Indexed: pat(n, f.SepK), Mask: mask}
	}
	raw, _ := hex.DecodeString(f.Raw)
	return fdother.Frame{X: f.X, Y: f.Y, Width: f.W, Height: f.H, Pixels: raw}
}

func (e *exporter) ch23Blit(kind string, f frameSpec, n, k int) {
	staging := pat(n, k)
	err, panicked := protect(func() error { return fdother.BlitNativeCh23Stage(f.frame(), staging) })
	e.add(kind, "ch23_blit", obj{"frame": f, "n": n, "k": k},
		obj{"err": errv(err), "panic": panicked, "staging": digest(staging)}, err)
}

type hookScript struct {
	Nil      [4]bool `json:"nil"`       // latch、palette、draw、tick
	FailHook int     `json:"fail_hook"` // -1 不失敗
	FailAt   int     `json:"fail_at"`
	Write    bool    `json:"write"` // latch 把 stage 寫進 staging[stage]
}

func (e *exporter) ch23Loop(kind string, spec fdother.NativeCh23LoopSpec, n, dacN int, s hookScript) {
	staging, dac := pat(n, 17), pat(dacN, 19)
	calls := []string{}
	count := [4]int{}
	fail := func(hook int) error {
		count[hook]++
		if s.FailHook == hook && count[hook]-1 == s.FailAt {
			return fmt.Errorf("synthetic hook %d failure", hook)
		}
		return nil
	}
	hooks := fdother.NativeCh23LoopHooks{
		Latch: func(stage int) error {
			calls = append(calls, fmt.Sprintf("L%d", stage))
			if s.Write && stage < len(staging) {
				staging[stage] = byte(stage + 100)
				if len(dac) > stage {
					dac[stage] = byte(stage + 50)
				}
			}
			return fail(0)
		},
		Palette: func(esi int) error { calls = append(calls, fmt.Sprintf("P%d", esi)); return fail(1) },
		Draw:    func() error { calls = append(calls, "D"); return fail(2) },
		Tick:    func() error { calls = append(calls, "T"); return fail(3) },
	}
	if s.Nil[0] {
		hooks.Latch = nil
	}
	if s.Nil[1] {
		hooks.Palette = nil
	}
	if s.Nil[2] {
		hooks.Draw = nil
	}
	if s.Nil[3] {
		hooks.Tick = nil
	}
	err := fdother.RunNativeCh23Loop(spec, staging, dac, hooks)
	stages := spec.StageValues
	if stages == nil {
		stages = []int{}
	}
	e.add(kind, "ch23_loop", obj{"phase": spec.Phase, "repeat": spec.Repeat, "stages": stages, "palette": spec.Palette,
		"n": n, "dac_n": dacN, "hooks": s},
		obj{"recovered": spec.IsRecoveredContract(), "calls_n": len(calls), "calls": digest([]byte(strings.Join(calls, ","))),
			"head": calls[:min(len(calls), 12)], "staging": digest(staging), "dac": digest(dac)}, err)
}

func rangeInts(a, b int) []int {
	out := []int{}
	for i := a; i <= b; i++ {
		out = append(out, i)
	}
	return out
}

func (e *exporter) genCh23() {
	const stage = 312 * 192
	for _, n := range []int{stage, stage + 7, stage - 1} {
		for latch := -2; latch <= 194; latch++ {
			if n != stage && latch%17 != 0 && latch != 192 && latch != 193 {
				continue
			}
			buf := pat(n, 23)
			err := fdother.RotateNativeCh23Rows(buf, latch)
			e.add("enum-latch", "ch23_rotate", obj{"n": n, "latch": latch}, obj{"buf": digest(buf)}, err)
		}
	}
	for _, x := range extremes {
		buf := pat(stage, 23)
		err := fdother.RotateNativeCh23Rows(buf, x)
		e.add("edge-extreme", "ch23_rotate", obj{"n": stage, "latch": x}, obj{"buf": digest(buf)}, err)
	}
	for _, n := range []int{768, 767, 769, 0} {
		for phase := -2; phase <= 17; phase++ {
			dac := pat(n, 29)
			err := fdother.ApplyNativeDACPaletteCycleE0EF(dac, phase)
			e.add("enum-phase", "ch23_dac_apply", obj{"n": n, "phase": phase}, obj{"dac": digest(dac)}, err)
		}
	}
	ticks := []int{-0x8001, -0x8000, -0x7fff, -1, 0, 1, 2, 3, 0x7ffe, 0x7fff, 0x8000}
	for _, n := range []int{768, 767} {
		for phase := -1; phase <= 16; phase++ {
			for _, last := range ticks {
				for _, now := range ticks {
					if n != 768 && (phase%5 != 0 || last != 0) {
						continue
					}
					dac := pat(n, 31)
					p, t, adv, err := fdother.AdvanceNativeDACPaletteCycleE0EF(dac, phase, last, now)
					e.add("enum-tick", "ch23_dac_advance", obj{"n": n, "phase": phase, "last": last, "now": now},
						obj{"phase": p, "tick": t, "advanced": adv, "dac": digest(dac)}, err)
				}
			}
		}
	}
	// RunNativeCh23Loop：合法與不合法的規格、各種 nil callback、各 callback 在各次呼叫失敗。
	specs := []fdother.NativeCh23LoopSpec{
		{Phase: "initial", Repeat: 30, StageValues: rangeInts(2, 9)},
		{Phase: "palette", Repeat: 12, StageValues: rangeInts(10, 14), Palette: true},
		{Phase: "initial", Repeat: 30, StageValues: rangeInts(2, 9), Palette: true},
		{Phase: "palette", Repeat: 12, StageValues: rangeInts(10, 14)},
		{Phase: "initial", Repeat: 29, StageValues: rangeInts(2, 9)},
		{Phase: "initial", Repeat: 30, StageValues: rangeInts(2, 8)},
		{Phase: "initial", Repeat: 30, StageValues: []int{2, 3, 4, 5, 6, 7, 9, 8}},
		{Phase: "palette", Repeat: 12, StageValues: rangeInts(10, 15), Palette: true},
		{Phase: "Initial", Repeat: 30, StageValues: rangeInts(2, 9)},
		{Phase: "", Repeat: 0},
		{Phase: "palette", Repeat: 12, Palette: true},
	}
	for _, spec := range specs {
		for mask := 0; mask < 16; mask++ {
			s := hookScript{FailHook: -1}
			for i := range s.Nil {
				s.Nil[i] = mask&(1<<i) != 0
			}
			e.ch23Loop("enum-nil", spec, stage, 768, s)
		}
		for _, n := range []int{stage - 1, stage + 1} {
			e.ch23Loop("enum-len", spec, n, 768, hookScript{FailHook: -1})
		}
		for _, dacN := range []int{0, 767} {
			e.ch23Loop("enum-len", spec, stage, dacN, hookScript{FailHook: -1})
		}
	}
	for i := 0; i < 160; i++ {
		spec := specs[e.rng.IntN(2)]
		hook := e.rng.IntN(4)
		e.ch23Loop("random-fail", spec, stage, 768, hookScript{FailHook: hook, FailAt: e.rng.IntN(70), Write: e.rng.IntN(2) == 0})
	}
	e.ch23Loop("write-ok", specs[0], stage, 768, hookScript{FailHook: -1, Write: true})
	e.ch23Loop("write-ok", specs[1], stage, 768, hookScript{FailHook: -1, Write: true})
	// 合成 ch23 幀：幾何錯、暫存面大小錯、RLE 壞、分離表示。
	for _, geom := range [][4]int{{0, 0, 312, 192}, {1, 0, 312, 192}, {0, 1, 312, 192}, {0, 0, 311, 192}, {0, 0, 312, 191}} {
		for _, n := range []int{stage, stage - 1, stage + 1} {
			e.ch23Blit("enum-geom", frameSpec{X: geom[0], Y: geom[1], W: geom[2], H: geom[3], SepK: 7}, n, 3)
		}
	}
	for i := 0; i < 60; i++ {
		raw := make([]byte, e.rng.IntN(40))
		for j := range raw {
			raw[j] = byte(e.rng.Uint32())
		}
		e.ch23Blit("random-rle", frameSpec{W: 312, H: 192, Raw: hexs(raw)}, stage, 5)
	}
	for i := 0; i < 20; i++ {
		e.ch23Blit("random-sep", frameSpec{W: 312, H: 192, SepK: 1 + e.rng.IntN(200)}, stage, 1+e.rng.IntN(200))
	}
}

// ---- preparation_roster ----

type spriteSpec struct {
	Index int    `json:"i"`
	Kind  string `json:"kind"` // solid、pattern、broken、nomask
	P     int    `json:"p"`
}

type bankSpec struct {
	N       int          `json:"n"`
	Sprites []spriteSpec `json:"sprites"`
}

func (b bankSpec) bank() *fdicon.Bank {
	out := &fdicon.Bank{Sprites: make([]fdicon.Sprite, b.N)}
	const n = fdicon.NativeSize * fdicon.NativeSize
	for _, s := range b.Sprites {
		if s.Index >= b.N {
			continue
		}
		var sp fdicon.Sprite
		switch s.Kind {
		case "solid":
			sp = fdicon.Sprite{Pixels: pat(n, 0), Mask: make([]byte, n), RemapMask: make([]byte, n)}
			for i := range sp.Pixels {
				sp.Pixels[i], sp.Mask[i] = byte(s.P), 1
			}
		case "pattern":
			sp = fdicon.Sprite{Pixels: pat(n, s.P), Mask: pat(n, s.P+3), RemapMask: make([]byte, n)}
			for i := range sp.Mask {
				sp.Mask[i] &= 1
			}
		case "nomask":
			sp = fdicon.Sprite{Pixels: pat(n, s.P)}
		}
		out.Sprites[s.Index] = sp
	}
	return out
}

func (e *exporter) bankSpec(groups int) bankSpec {
	b := bankSpec{N: groups * 12}
	for g := 0; g < groups; g++ {
		for cycle := 0; cycle < 3; cycle++ {
			kind := "solid"
			switch e.rng.IntN(30) {
			case 0:
				kind = "broken"
			case 1:
				kind = "nomask"
			case 2, 3, 4, 5, 6, 7:
				kind = "pattern"
			}
			b.Sprites = append(b.Sprites, spriteSpec{Index: g*12 + cycle, Kind: kind, P: g*8 + cycle + 1})
		}
	}
	return b
}

func (e *exporter) genPrepMove() {
	scans := []byte{0x4b, 0x4d, 0x48, 0x50, 0x00, 0x4c, 0xff}
	for count := -1; count <= 31; count++ {
		for _, scan := range scans {
			out := [][]any{}
			for cursor := -1; cursor <= 31; cursor++ {
				v, err := fdother.MoveNativePreparationRosterCursor(cursor, count, scan)
				out = append(out, []any{v, errv(err)})
			}
			e.add("enum-cursor", "prep_move", obj{"count": count, "scan": scan}, obj{"out": out}, nil)
		}
	}
	for index := -2; index <= 31; index++ {
		out := [][]any{}
		for _, sel := range []bool{false, true} {
			for _, cur := range []bool{false, true} {
				x, y, err := fdother.NativePreparationRosterPosition(index, sel, cur)
				out = append(out, []any{x, y, errv(err)})
			}
		}
		e.add("enum-index", "prep_position", obj{"index": index}, obj{"out": out}, nil)
	}
	for _, x := range extremes {
		x0, y0, err := fdother.NativePreparationRosterPosition(x, true, true)
		e.add("edge-extreme", "prep_position", obj{"index": x}, obj{"out": [][]any{{x0, y0, errv(err)}}}, nil)
		v, err := fdother.MoveNativePreparationRosterCursor(x, 16, 0x50)
		e.add("edge-extreme", "prep_move", obj{"count": 16, "scan": 0x50, "cursor": x}, obj{"out": [][]any{{v, errv(err)}}}, nil)
	}
}

func (e *exporter) genPrepRoster() {
	for i := 0; i < 400; i++ {
		groups := 1 + e.rng.IntN(5)
		units := e.bankSpec(groups)
		cursor := bankSpec{N: 1, Sprites: []spriteSpec{{Index: 0, Kind: "solid", P: 0x55}}}
		switch e.rng.IntN(15) {
		case 0:
			cursor = bankSpec{N: 0}
		case 1:
			cursor.Sprites[0].Kind = "nomask"
		case 2:
			cursor.Sprites[0].Kind = "pattern"
		}
		n := 1 + e.rng.IntN(30)
		if e.rng.IntN(15) == 0 {
			n = e.pickFrom([]int{0, 31})
		}
		keys, selected := make([]int, n), make([]bool, n)
		for j := range keys {
			keys[j] = e.rng.IntN(groups)
			if e.rng.IntN(40) == 0 {
				keys[j] = e.pickFrom([]int{-1, groups, 999})
			}
			selected[j] = e.rng.IntN(2) == 0
		}
		if e.rng.IntN(20) == 0 && n > 0 {
			selected = selected[:n-1]
		}
		cur, cycle := e.pick(-1, n), e.pick(-1, 3)
		if e.rng.IntN(3) != 0 && n > 0 {
			cur, cycle = e.rng.IntN(n), e.rng.IntN(3)
		}
		stride, rows := 320, 200
		switch e.rng.IntN(12) {
		case 0:
			stride = 319
		case 1:
			rows = 199
		case 2:
			stride = 330
		}
		dst := pat(stride*rows, 41)
		err, panicked := protect(func() error {
			return fdother.BlitNativePreparationRoster(dst, stride, units.bank(), cursor.bank(), keys, selected, cur, cycle)
		})
		e.add("random", "prep_roster", obj{"units": units, "cursor_bank": cursor, "keys": keys, "selected": selected,
			"cursor": cur, "cycle": cycle, "stride": stride, "rows": rows},
			obj{"err": errv(err), "panic": panicked, "dst": digest(dst)}, err)
	}
}

// 合成整備素材：面板與數字用分離表示的 Frame，單位與游標用 bankSpec。
type prepAssetsSpec struct {
	UpperRight [4]int      `json:"upper_right"`
	Lower      [4]int      `json:"lower"`
	UpperLeft  frameSpec   `json:"upper_left"`
	Quota      []frameSpec `json:"quota"`
	Remaining  []frameSpec `json:"remaining"`
	Cursor     bankSpec    `json:"cursor"`
	Units      bankSpec    `json:"units"`
}

func (s prepAssetsSpec) assets() *fdother.NativePreparationAssets {
	a := &fdother.NativePreparationAssets{UpperRight: lmi(s.UpperRight), Lower: lmi(s.Lower), UpperLeft: s.UpperLeft.frame(),
		Cursor: s.Cursor.bank(), Units: s.Units.bank()}
	for i := 0; i < 10; i++ {
		a.QuotaDigits[i] = s.Quota[i].frame()
		a.RemainingDigits[i] = s.Remaining[i].frame()
	}
	return a
}

type composeArgs struct {
	Fn        string `json:"fn"` // compose、background、without_units、without_cursor
	BgN       int    `json:"bg_n"`
	BgK       int    `json:"bg_k"`
	Keys      []int  `json:"keys"`
	Selected  []bool `json:"selected"`
	Cursor    int    `json:"cursor"`
	Cycle     int    `json:"cycle"`
	Limit     int    `json:"limit"`
	DrawUnits bool   `json:"draw_units"`
}

func compose(a *fdother.NativePreparationAssets, c composeArgs) ([]byte, error) {
	bg := pat(c.BgN, c.BgK)
	switch c.Fn {
	case "compose":
		return fdother.ComposeNativePreparationFrame(a, c.Keys, c.Selected, c.Cursor, c.Cycle, c.Limit)
	case "background":
		return fdother.ComposeNativePreparationFrameOnBackground(bg, a, c.Keys, c.Selected, c.Cursor, c.Cycle, c.Limit)
	case "without_units":
		return fdother.ComposeNativePreparationFrameWithoutRosterUnits(bg, a, c.Keys, c.Selected, c.Cursor, c.Cycle, c.Limit)
	}
	return fdother.ComposeNativePreparationFrameWithoutCursor(bg, a, c.Keys, c.Selected, c.Cursor, c.Cycle, c.Limit, c.DrawUnits)
}

func (e *exporter) composeArgs(groups int) composeArgs {
	c := composeArgs{Fn: e.pickStr([]string{"compose", "background", "without_units", "without_cursor"}),
		BgN: 64000, BgK: e.rng.IntN(200) + 1, DrawUnits: e.rng.IntN(2) == 0}
	if e.rng.IntN(15) == 0 {
		c.BgN = 63999
	}
	n := 1 + e.rng.IntN(30)
	c.Keys, c.Selected = make([]int, n), make([]bool, n)
	sel := 0
	for j := range c.Keys {
		c.Keys[j] = e.rng.IntN(groups)
		c.Selected[j] = e.rng.IntN(3) == 0
		if c.Selected[j] {
			sel++
		}
	}
	c.Cursor, c.Cycle = e.rng.IntN(n), e.rng.IntN(3)
	c.Limit = sel + e.rng.IntN(100-sel)
	switch e.rng.IntN(12) {
	case 0:
		c.Limit = sel - 1
	case 1:
		c.Limit = e.pickFrom([]int{-1, 100})
	case 2:
		c.Selected = c.Selected[:n-1]
	}
	return c
}

func (e *exporter) genPrepCompose() {
	digitFrame := func(w, h, k int) frameSpec { return frameSpec{W: w, H: h, SepK: k} }
	for i := 0; i < 150; i++ {
		groups := 1 + e.rng.IntN(4)
		s := prepAssetsSpec{
			UpperRight: [4]int{223, 86, 223 * 86, 3}, Lower: [4]int{310, 99, 310 * 99, 5},
			UpperLeft: frameSpec{W: 86, H: 86, SepK: 7},
			Cursor:    bankSpec{N: 1, Sprites: []spriteSpec{{Index: 0, Kind: "solid", P: 0x55}}},
			Units:     e.bankSpec(groups),
		}
		for d := 0; d < 10; d++ {
			s.Quota = append(s.Quota, digitFrame(5+d%2, 8, 11+d))
			s.Remaining = append(s.Remaining, digitFrame(6-d%2, 8, 31+d))
		}
		switch e.rng.IntN(10) {
		case 0:
			s.Quota[e.rng.IntN(10)] = digitFrame(e.pickFrom([]int{4, 7}), 8, 9)
		case 1:
			s.Remaining[e.rng.IntN(10)] = digitFrame(5, e.pickFrom([]int{7, 9}), 9)
		case 2:
			s.UpperRight = [4]int{223, 86, 10, 3}
		case 3:
			s.UpperLeft = frameSpec{X: 300, W: 86, H: 86, SepK: 7}
		}
		c := e.composeArgs(groups)
		var frame []byte
		err, panicked := protect(func() error {
			var err error
			frame, err = compose(s.assets(), c)
			return err
		})
		res := obj{"err": errv(err), "panic": panicked}
		if err == nil && !panicked {
			res["frame"] = digest(frame)
		}
		e.add("random", "prep_compose", obj{"assets": s, "args": c}, res, err)
	}
}

// ---- 真實原版檔與分離素材包 ----

func frameDigest(f fdother.Frame) []any {
	return []any{f.X, f.Y, f.Width, f.Height, digest(f.Pixels), digest(f.Indexed), digest(f.Mask)}
}

func lmiDigest(l fdother.LMI1Entry) []any { return []any{l.Width, l.Height, digest(l.Pixels)} }

func bankDigest(b *fdicon.Bank) []any {
	h := sha256.New()
	for _, s := range b.Sprites {
		for _, part := range [][]byte{s.Pixels, s.Mask, s.RemapMask} {
			fmt.Fprintf(h, "%d:", len(part))
			h.Write(part)
		}
	}
	return []any{len(b.Sprites), hex.EncodeToString(h.Sum(nil))}
}

func assetsDigest(a *fdother.NativePreparationAssets) obj {
	q, r := [][]any{}, [][]any{}
	for i := 0; i < 10; i++ {
		q = append(q, frameDigest(a.QuotaDigits[i]))
		r = append(r, frameDigest(a.RemainingDigits[i]))
	}
	return obj{"upper_right": lmiDigest(a.UpperRight), "lower": lmiDigest(a.Lower), "upper_left": frameDigest(a.UpperLeft),
		"quota": q, "remaining": r, "cursor": bankDigest(a.Cursor), "units": bankDigest(a.Units)}
}

func (e *exporter) genReal() obj {
	real := obj{}
	for _, name := range []string{"FDOTHER.DAT", "FDICON.B24"} {
		data, err := os.ReadFile(filepath.Join(e.flame2, name))
		must(err)
		real[name] = obj{"size": len(data), "sha256": digest(data)}
	}
	dat := filepath.Join(e.flame2, "FDOTHER.DAT")
	frame, err := fdother.DecodeNativeCh23Stage(dat)
	res := obj{}
	if err == nil {
		res["frame"] = frameDigest(frame)
		staging := pat(312*192, 3)
		err2 := fdother.BlitNativeCh23Stage(frame, staging)
		res["blit"] = []any{errv(err2), digest(staging)}
		indexed, mask, err3 := frame.IndexedLayers()
		res["layers"] = []any{errv(err3), digest(indexed), digest(mask)}
	}
	e.add("real", "ch23_decode_real", obj{}, res, err)

	entries, err := fdother.DecodeNativeSpawnIntroFrames(dat)
	out := [][]any{}
	for _, en := range entries {
		out = append(out, lmiDigest(en))
	}
	e.add("real", "spawn_decode_real", obj{}, obj{"entries": out}, err)

	r5, err := fdother.DecodeLMI1Resource(dat, 5)
	must(err)
	r6, err := fdother.DecodeLMI1Resource(dat, 6)
	must(err)
	for _, id := range rangeInts(0, 8) {
		s, err := fdother.BuildNativeAIItemType21TailSchedule(id, r6, r5)
		e.add("real", "type21_real", obj{"id": id}, obj{"start": s.EffectStart, "frames": s.EffectFrames,
			"sample": s.EffectSample}, err)
	}
	s32, err := fdother.BuildNativeCommand32TailSchedule(r6, r5)
	e.add("real", "command32_real", obj{}, obj{"start": s32.EffectStart, "bias": s32.DigitBias}, err)
	s9, err := fdother.BuildNativeCommand9PlayerSchedule(r6, r5)
	e.add("real", "command9_real", obj{}, obj{"start": s9.EffectStart, "bias": s9.ResultDigitBias}, err)
	e.add("real", "ch28_validate_real", obj{}, obj{}, fdother.ValidateNativeCh28PostLMI(r5))
	// 第 28 章的十二個真實格依原生排程寫進工作面。
	work := pat(fdother.NativeCh28PostWorkSize, 37)
	origin, err := fdother.NativeCh28PostOverlayOrigin(12, 9, 4, 3)
	must(err)
	errs := []any{}
	for _, f := range fdother.NativeCh28PostOverlayFrames() {
		errs = append(errs, errv(fdother.BlitNativeCh28PostOverlay(r5[f.Entry], work, origin)))
	}
	e.add("real", "ch28_blit_real", obj{"a": []int{12, 9, 4, 3}}, obj{"errs": errs, "work": digest(work)}, nil)
	for _, a := range [][4]int{{22, 23, 16, 19}, {16, 19, 16, 19}, {28, 26, 16, 19}, {0, 0, 1, 1}} {
		e.unitIntro("real", nil, 0x25680, a, tickScript{Mode: "mutate", Pos: 0x9000, Val: 3}, r6, "fdother#6")
	}
	spawnEntries, err := fdother.DecodeNativeSpawnIntroFrames(dat)
	must(err)
	work = pat(0x25680, 43)
	errs = []any{}
	for _, step := range fdother.NativeSpawnIntroSchedule() {
		errs = append(errs, errv(fdother.BlitNativeSpawnIntroLMI(spawnEntries[step.LMIEntry], work, 14, 23, 10, 20)))
	}
	e.add("real", "spawn_blit_real", obj{"a": []int{14, 23, 10, 20}}, obj{"errs": errs, "work": digest(work)}, nil)

	// 整備素材：封存解碼（明示與預設 FDICON 路徑）與分離素材包載入。
	archive, err := fdother.DecodeNativePreparationAssetsArchive(dat, filepath.Join(e.flame2, "FDICON.B24"))
	must(err)
	e.add("real", "prep_assets_real", obj{"source": "archive"}, assetsDigest(archive), nil)
	viaDefault, err := fdother.DecodeNativePreparationAssetsArchive(dat, "")
	must(err)
	e.add("real", "prep_assets_real", obj{"source": "archive_default"}, assetsDigest(viaDefault), nil)
	separated, err := fdother.LoadNativePreparationAssets(e.assets)
	must(err)
	e.add("real", "prep_assets_real", obj{"source": "separated"}, assetsDigest(separated), nil)
	groups := len(archive.Units.Sprites) / 12
	for i := 0; i < 80; i++ {
		c := e.composeArgs(groups)
		for _, src := range []string{"archive", "separated"} {
			a := archive
			if src == "separated" {
				a = separated
			}
			frame, err := compose(a, c)
			res := obj{}
			if err == nil {
				res["frame"] = digest(frame)
			}
			e.add("real", "prep_compose_real", obj{"source": src, "args": c}, res, err)
		}
	}
	return real
}

// ch23 分離面與整備素材包的變異樹：檔案內容以十六進位嵌入，Rust 端寫進暫存目錄重建。
func (e *exporter) genRealPack() {
	sep, err := fdother.LoadSeparatedNativeCh23Stage(filepath.Join(e.assets, "surfaces"))
	res := obj{}
	if err == nil {
		res["frame"] = frameDigest(sep)
	}
	e.add("real", "ch23_load_sep", obj{"tree": "real"}, res, err)
	dir := filepath.Join(e.assets, "surfaces", "FDOTHER_042")
	read := func(name string) []byte {
		data, err := os.ReadFile(filepath.Join(dir, name))
		must(err)
		return data
	}
	meta, framePNG, maskPNG := read("resource.json"), read("frame.png"), read("mask.png")
	edit := func(f func(source map[string]any)) []byte {
		var d map[string]any
		must(json.Unmarshal(meta, &d))
		f(d["source"].(map[string]any))
		out, err := json.Marshal(d)
		must(err)
		return out
	}
	variants := []struct {
		kind  string
		files map[string][]byte
	}{
		{"empty", map[string][]byte{}},
		{"copy", map[string][]byte{"resource.json": meta, "frame.png": framePNG, "mask.png": maskPNG}},
		{"json-only", map[string][]byte{"resource.json": meta}},
		{"no-mask", map[string][]byte{"resource.json": meta, "frame.png": framePNG}},
		{"swapped-png", map[string][]byte{"resource.json": meta, "frame.png": maskPNG, "mask.png": framePNG}},
		{"bad-size", map[string][]byte{"resource.json": edit(func(s map[string]any) { s["raw_size"] = 59413 })}},
		{"bad-md5", map[string][]byte{"resource.json": edit(func(s map[string]any) { s["raw_md5"] = "00" })}},
		{"bad-sha", map[string][]byte{"resource.json": edit(func(s map[string]any) { s["raw_sha256"] = "" })}},
		{"null-json", map[string][]byte{"resource.json": []byte("null")}},
		{"broken-json", map[string][]byte{"resource.json": []byte("{\"source\":")}},
		{"type-json", map[string][]byte{"resource.json": []byte("{\"source\":[]}")}},
	}
	for i, v := range variants {
		root := filepath.Join(e.tmp, fmt.Sprintf("ch23-%02d", i))
		must(os.MkdirAll(filepath.Join(root, "FDOTHER_042"), 0o755))
		files := obj{}
		names := []string{}
		for name := range v.files {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			must(os.WriteFile(filepath.Join(root, "FDOTHER_042", name), v.files[name], 0o644))
			files[name] = hexs(v.files[name])
		}
		sep, err := fdother.LoadSeparatedNativeCh23Stage(root)
		res := obj{}
		if err == nil {
			res["frame"] = frameDigest(sep)
		}
		e.add("tree-"+v.kind, "ch23_load_sep", obj{"tree": fmt.Sprintf("ch23-%02d", i), "files": files}, res, err)
	}
	_, err = fdother.LoadNativePreparationAssets(filepath.Join(e.tmp, "missing-pack"))
	e.add("tree-missing", "prep_load_missing", obj{"tree": "missing-pack"}, obj{}, err)
}

func treeDigest(root string) (string, int) {
	var lines []string
	must(filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(root, p)
			lines = append(lines, rel+"\x00"+digest(data)+"\n")
		}
		return nil
	}))
	sort.Strings(lines)
	return digest([]byte(strings.Join(lines, ""))), len(lines)
}

// 分離素材包中本工具讀到的子目錄。
var packDirs = []string{"surfaces/FDOTHER_042", "ui/fdother_005_item_panel", "sprites/fdicon",
	"sprites/fdother_001_range_overlay"}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	flame2 := flag.String("flame2", "", "原版 FLAME2 目錄（唯讀）")
	assets := flag.String("assets", "", "分離素材包根目錄（唯讀）")
	flag.Parse()
	if *out == "" || *flame2 == "" || *assets == "" {
		fmt.Fprintln(os.Stderr, "需要 -out、-flame2、-assets")
		os.Exit(2)
	}
	tmp, err := os.MkdirTemp("", "fd2-golden-fdother-flow-")
	must(err)
	defer os.RemoveAll(tmp)
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x46444f46)), tmp: tmp, flame2: *flame2, assets: *assets}
	e.genOverlay()
	e.genCh28()
	e.genTransition()
	e.genSchedules()
	e.genSpawn()
	e.genUnitPresent()
	e.genCh23()
	e.genPrepMove()
	e.genPrepRoster()
	e.genPrepCompose()
	real := e.genReal()
	pack := obj{}
	for _, d := range packDirs {
		sum, n := treeDigest(filepath.Join(*assets, d))
		pack[d] = obj{"files": n, "tree_sha256": sum}
	}
	real["pack"] = pack
	e.genRealPack()
	doc := obj{"schema": "fd2-golden-fdother-flow/1", "seed": *seed, "real": real, "cases": e.cases}
	data, err := json.Marshal(doc)
	must(err)
	must(os.WriteFile(*out, data, 0o644))
	ops, compares := map[string]int{}, map[string]int{}
	for _, c := range e.cases {
		ops[c.Op]++
		compares[c.Compare]++
	}
	fmt.Printf("fd2-golden-fdother-flow: %d cases sha256=%s\ncompare=%v\nops=%v\n", len(e.cases), digest(data), compares, ops)
}
