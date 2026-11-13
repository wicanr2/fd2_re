package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// realArchives 解碼兩個原版容器的每一筆資源（含 -1 與 count 兩個越界序號）。
func (e *exporter) realArchives(figaniPath, fdotherPath string) {
	for _, item := range []struct{ name, path string }{{"figani", figaniPath}, {"fdother", fdotherPath}} {
		count, err := fdother.ArchiveResourceCount(item.path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for resource := -1; resource <= count; resource++ {
			a, err := figani.DecodeResource(item.path, resource)
			var result any
			if err != nil {
				result = e.errObj(err)
			} else {
				result = e.animationResult(a, true)
			}
			e.add("real_"+item.name+"_dat", "decode_resource", obj{"file": item.name, "resource": resource}, result)
		}
	}
}

var separatedDir = regexp.MustCompile(`^(FIGANI|FDOTHER)_([0-9]{3})$`)

func (e *exporter) sepLoad(kind, root, source string, resource int) {
	a, err := figani.LoadSeparatedArchiveResource(e.expand(root), source, resource)
	var result any
	if err != nil {
		result = e.errObj(err)
	} else {
		result = obj{"anim": animJSON(a)}
	}
	e.add(kind, "sep_load", obj{"root": root, "source": source, "resource": resource}, result)
}

func (e *exporter) sepFallback(kind, root string, resource int) {
	a, source, err := figani.LoadSeparatedResourceWithZeroHeaderFallbackSource(e.expand(root), resource)
	var result any
	if err != nil {
		result = obj{"err": e.errText(err), "source": source}
	} else {
		result = obj{"anim": animJSON(a), "source": source}
	}
	// 包一層不回傳來源的版本，確認兩個入口一致。
	_, plainErr := figani.LoadSeparatedResourceWithZeroHeaderFallback(e.expand(root), resource)
	plain := any(nil)
	if plainErr != nil {
		plain = e.errText(plainErr)
	}
	e.add(kind, "sep_fallback", obj{"root": root, "resource": resource}, obj{"source_result": result, "plain_err": plain})
}

// realSeparated 載入分離素材包裡每一個 FIGANI_NNN／FDOTHER_NNN 目錄（排序後的目錄清單），
// 並對 FIGANI 0..409 全數走零標頭回退入口。
func (e *exporter) realSeparated(animations string) {
	e.pack = animations
	entries, err := os.ReadDir(animations)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, entry := range entries {
		m := separatedDir.FindStringSubmatch(entry.Name())
		if m == nil || !entry.IsDir() {
			continue
		}
		resource, _ := strconv.Atoi(m[2])
		e.sepLoad("real_separated", "$PACK", m[1]+".DAT", resource)
	}
	for resource := -1; resource <= 410; resource++ {
		e.sepFallback("real_separated_fallback", "$PACK", resource)
	}
	for _, resource := range []int{998, 999, 1000} {
		e.sepFallback("real_separated_fallback", "$PACK", resource)
	}
	for resource := 0; resource <= 100; resource++ {
		e.sepLoad("real_separated_fdother_scan", "$PACK", "FDOTHER.DAT", resource)
	}
	for _, source := range []string{"BG.DAT", "", "figani.dat", "FIGANI.DAT\n", "中文.DAT", "FDOTHER.DAT\x00"} {
		e.sepLoad("real_separated_source", "$PACK", source, 4)
	}
	e.sepLoad("real_separated_source", "", "FIGANI.DAT", 4)
}

// ---- FIGANI 原始位元組 ----

type rleFrame struct {
	x, y        int
	w, h        int
	raw4, raw5  byte
	delay, raw7 byte
	pad         byte
	rle         []byte
}

func (e *exporter) randomRLE(w, h int) []byte {
	var out []byte
	r := e.rng
	for y := 0; y < h; y++ {
		x := 0
		for x < w {
			remaining := w - x
			mode := r.IntN(4)
			if mode == 1 && remaining < 2 {
				mode = 0
			}
			limit := remaining
			if mode == 1 {
				limit = remaining / 2
			}
			count := 1 + r.IntN(min(64, limit))
			out = append(out, byte(mode<<6|(count-1)))
			switch mode {
			case 0, 1:
				out = append(out, byte(r.Uint32()))
			case 2:
				for i := 0; i < count; i++ {
					out = append(out, byte(r.Uint32()))
				}
			}
			if mode == 1 {
				count *= 2
			}
			x += count
		}
	}
	return out
}

func (e *exporter) randomFrame(maxW, maxH int) rleFrame {
	r := e.rng
	w, h := 1+r.IntN(maxW), 1+r.IntN(maxH)
	return rleFrame{
		x: r.IntN(400) - 60, y: r.IntN(260) - 40, w: w, h: h,
		raw4: byte(r.IntN(3)), raw5: byte(r.IntN(5)), delay: byte(r.IntN(8)), raw7: byte(r.IntN(3)), pad: byte(r.Uint32()),
		rle: e.randomRLE(w, h),
	}
}

func buildAnimation(header [8]byte, frames []rleFrame) []byte {
	n := len(frames)
	out := append([]byte(nil), header[:]...)
	out[0] = byte(n)
	table := make([]byte, 4*n)
	out = append(out, table...)
	for i, f := range frames {
		binary.LittleEndian.PutUint32(out[8+4*i:], uint32(len(out)))
		var head [13]byte
		binary.LittleEndian.PutUint16(head[0:], uint16(int16(f.x)))
		binary.LittleEndian.PutUint16(head[2:], uint16(int16(f.y)))
		head[4], head[5], head[6], head[7], head[8] = f.raw4, f.raw5, f.delay, f.raw7, f.pad
		binary.LittleEndian.PutUint16(head[9:], uint16(f.w))
		binary.LittleEndian.PutUint16(head[11:], uint16(f.h))
		out = append(out, head[:]...)
		out = append(out, f.rle...)
	}
	return out
}

func (e *exporter) randomAnimation(maxFrames, maxW, maxH int) []byte {
	r := e.rng
	frames := make([]rleFrame, 1+r.IntN(maxFrames))
	for i := range frames {
		frames[i] = e.randomFrame(maxW, maxH)
	}
	header := [8]byte{0, byte(r.IntN(2)), byte(r.IntN(len(frames) + 1)), byte(r.Uint32()), byte(r.IntN(8)), byte(r.Uint32()), byte(r.Uint32()), byte(r.Uint32())}
	return buildAnimation(header, frames)
}

func (e *exporter) parseCase(kind string, raw []byte) {
	a, err := figani.Parse(raw)
	var result any
	if err != nil {
		result = e.errObj(err)
	} else {
		result = e.animationResult(a, true)
	}
	e.add(kind, "parse", obj{"hex": hex.EncodeToString(raw)}, result)
}

func (e *exporter) mutate(raw []byte) []byte {
	r := e.rng
	out := append([]byte(nil), raw...)
	for k := 0; k < 1+r.IntN(3); k++ {
		if len(out) == 0 {
			break
		}
		switch r.IntN(9) {
		case 0:
			out = out[:r.IntN(len(out))]
		case 1:
			out[r.IntN(len(out))] ^= byte(1 + r.IntN(255))
		case 2:
			out[0] = 0
		case 3:
			out[0] = byte(1 + r.IntN(255))
		case 4:
			if len(out) > 2 {
				out[1], out[2] = 1, byte(int(out[0])+1+r.IntN(3))
			}
		case 5:
			if n := int(out[0]); n > 0 && 8+4*n <= len(out) {
				binary.LittleEndian.PutUint32(out[8+4*r.IntN(n):], uint32(r.IntN(len(out)+20)))
			}
		case 6:
			if n := int(out[0]); n > 0 && 8+4*n <= len(out) {
				off := int(binary.LittleEndian.Uint32(out[8:]))
				if off+13 <= len(out) {
					binary.LittleEndian.PutUint16(out[off+9+2*r.IntN(2):], uint16([]int{0, 1025, 1024, 0xffff}[r.IntN(4)]))
				}
			}
		case 7:
			for i := 0; i < 1+r.IntN(8); i++ {
				out = append(out, byte(r.Uint32()))
			}
		case 8:
			if n := int(out[0]); n > 1 && 8+4*n <= len(out) {
				i := r.IntN(n - 1)
				a := binary.LittleEndian.Uint32(out[8+4*i:])
				copy(out[8+4*i:8+4*i+4], out[8+4*(i+1):8+4*(i+1)+4])
				binary.LittleEndian.PutUint32(out[8+4*(i+1):], a)
			}
		}
	}
	return out
}

func (e *exporter) parseCases() {
	one := func(w, h int, rle ...byte) []byte {
		return buildAnimation([8]byte{}, []rleFrame{{x: 2, y: 3, w: w, h: h, delay: 2, rle: rle}})
	}
	structured := [][]byte{
		nil, {1}, make([]byte, 11), make([]byte, 12), {1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		one(4, 1, 0x00, 7, 0x40, 9, 0xc0), // Go 測試的 run／dither／透明
		one(4, 1),                         // RLE ends in row 0
		one(2, 2, 0xc1),                   // 第二列沒有資料
		one(2, 1, 0x02, 7),                // overrun（3 > 2）
		one(4, 1, 0x40),                   // dither 缺值
		one(1, 1, 0x00),                   // colour 缺值
		one(3, 1, 0x82, 1, 2),             // literal 超出
		one(3, 1, 0x82, 1, 2, 3),          // literal 剛好
		one(2, 1, 0x40, 5),                // dither 剛好填滿
		one(3, 1, 0x40, 5),                // dither 不足一列（剩 1）
		one(64, 1, 0xff),                  // 64 像素透明
		one(1024, 1, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff),
		one(1025, 1, 0xff), // 寬度上限外
		one(0, 1),          // 寬度 0
	}
	// 前奏旗標與幀數邊界。
	prelude := one(4, 1, 0x00, 7, 0x40, 9, 0xc0)
	for _, b := range [][2]byte{{1, 1}, {1, 2}, {0, 9}, {255, 0}} {
		p := append([]byte(nil), prelude...)
		p[1], p[2] = b[0], b[1]
		structured = append(structured, p)
	}
	// 幀表剛好占滿與越界。
	tableOnly := []byte{2, 0, 0, 0, 0, 0, 0, 0, 16, 0, 0, 0, 16, 0, 0, 0}
	structured = append(structured, tableOnly, append(append([]byte(nil), tableOnly...), make([]byte, 13)...))
	for _, raw := range structured {
		e.parseCase("parse_structured", raw)
	}
	for i := 0; i < 150; i++ {
		e.parseCase("parse_valid", e.randomAnimation(6, 24, 16))
	}
	for i := 0; i < 300; i++ {
		e.parseCase("parse_mutated", e.mutate(e.randomAnimation(4, 12, 8)))
	}
	for i := 0; i < 150; i++ {
		raw := make([]byte, e.rng.IntN(90))
		for j := range raw {
			raw[j] = byte(e.rng.Uint32())
		}
		if len(raw) > 0 {
			raw[0] = byte(e.rng.IntN(4))
		}
		e.parseCase("parse_random", raw)
	}
}

// ---- 排程器 ----

func (e *exporter) schedulerCases() {
	r := e.rng
	for i := 0; i < 120; i++ {
		count := r.IntN(6)
		delays := make([]int, count)
		for j := range delays {
			delays[j] = r.IntN(6) - 1
		}
		isNil := i%17 == 3
		state := []int{r.IntN(count+3) - 1, r.IntN(5) - 1}
		if i%23 == 7 {
			state[1] = math.MaxInt
		}
		advance := make([]bool, 14)
		for j := range advance {
			advance[j] = r.IntN(5) != 0
		}
		var a *figani.Animation
		if !isNil {
			a = &figani.Animation{Frames: make([]figani.Frame, count)}
			for j, d := range delays {
				a.Frames[j].Delay = d
			}
		}
		s := figani.NativeScheduler{FrameIndex: state[0], Subframe: state[1]}
		trace := [][]any{}
		for _, adv := range advance {
			index, rendered, err := s.Step(a, adv)
			if err != nil {
				trace = append(trace, []any{e.errText(err)})
				continue
			}
			trace = append(trace, []any{index, rendered, s.FrameIndex, s.Subframe})
		}
		e.add("native_scheduler", "native_sched", obj{"delays": delays, "nil": isNil, "state": state, "advance": advance}, trace)
	}
	for i := 0; i < 120; i++ {
		count := r.IntN(7)
		delays := make([]int, count)
		for j := range delays {
			delays[j] = r.IntN(7) - 1
			if r.IntN(12) == 0 {
				delays[j] = 1 << 40
			}
		}
		ticks := r.IntN(5) - 1
		if i%19 == 4 {
			ticks = 1 << 30
		}
		e.add("display_scheduler", "display_sched", obj{"delays": delays, "ticks": ticks}, e.displayJSON(delays, ticks, 300))
	}
}

// ---- 繪製 ----

func (e *exporter) blitCase(kind string, f figani.Frame, length, stride, base, dx, dy int, fill *byte) {
	run := func(draw func([]byte) error) any {
		dst := make([]byte, length)
		for i := range dst {
			dst[i] = 9
		}
		before := append([]byte(nil), dst...)
		err := draw(dst)
		if err != nil {
			return obj{"err": e.errText(err), "kept": string(dst) == string(before)}
		}
		return obj{"dst": hex.EncodeToString(dst)}
	}
	result := obj{
		"at":   run(func(dst []byte) error { return f.BlitAt(dst, stride) }),
		"base": run(func(dst []byte) error { return f.BlitAtBase(dst, stride, base) }),
	}
	// BlitTranslated 在寬高乘積溢位時 Go 會 index panic，只對小幾何呼叫。
	if f.Width < 1<<20 && f.Height < 1<<20 {
		result["translated"] = run(func(dst []byte) error { return f.BlitTranslated(dst, stride, dx, dy, fill) })
	}
	input := obj{
		"x": f.X, "y": f.Y, "w": f.Width, "h": f.Height, "pix": hex.EncodeToString(f.Pixels), "mask": hex.EncodeToString(f.Mask),
		"len": length, "stride": stride, "base": base, "dx": dx, "dy": dy, "fill": nil,
	}
	if fill != nil {
		input["fill"] = int(*fill)
	}
	e.add(kind, "blit", input, result)
}

func (e *exporter) blitCases() {
	r := e.rng
	for i := 0; i < 400; i++ {
		w, h := r.IntN(6), r.IntN(5)
		n := w * h
		if r.IntN(8) == 0 {
			n += r.IntN(3) - 1
		}
		if n < 0 {
			n = 0
		}
		f := figani.Frame{X: r.IntN(16) - 3, Y: r.IntN(12) - 3, Width: w, Height: h, Pixels: make([]byte, n), Mask: make([]byte, n)}
		for j := range f.Pixels {
			f.Pixels[j] = byte(r.Uint32())
			f.Mask[j] = byte(r.IntN(2))
		}
		if r.IntN(10) == 0 && n > 0 {
			f.Mask = f.Mask[:n-1]
		}
		var fill *byte
		if r.IntN(2) == 0 {
			fill = ptr(byte(r.Uint32()))
		}
		e.blitCase("blit_random", f, r.IntN(130), r.IntN(22)-2, r.IntN(60)-3, r.IntN(13)-6, r.IntN(13)-6, fill)
	}
	// Go 測試的極值：maxInt 的 Y、寬、高與跨距。
	valid := figani.Frame{X: 1, Width: 2, Height: 2, Pixels: []byte{2, 3, 4, 5}, Mask: []byte{1, 1, 1, 1}}
	for _, change := range []func(*figani.Frame){
		func(f *figani.Frame) { f.Y = math.MaxInt }, func(f *figani.Frame) { f.Width = math.MaxInt },
		func(f *figani.Frame) { f.Height = math.MaxInt }, func(f *figani.Frame) { f.X = math.MaxInt },
		func(f *figani.Frame) { f.X = math.MinInt }, func(*figani.Frame) {},
	} {
		f := valid
		change(&f)
		e.blitCase("blit_extreme", f, 16, 8, 4, 0, 0, nil)
	}
	e.blitCase("blit_extreme", valid, 16, math.MaxInt, 4, 0, 0, nil)
	e.blitCase("blit_extreme", valid, 16, 8, math.MaxInt, 0, 0, nil)
	e.blitCase("blit_extreme", valid, 16, 8, 4, math.MaxInt-1, math.MinInt+1, ptr(1))
}
