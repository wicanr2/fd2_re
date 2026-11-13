package main

import (
	"encoding/binary"
	"errors"
	"image/color"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// ---- 隨機產生器 ----

// rle 產生 width/height 標頭加四模式串流；corrupt 時截斷或改壞一個位元組。
func (e *exporter) rle(w, h int, corrupt bool) []byte {
	out := binary.LittleEndian.AppendUint16(nil, uint16(w))
	out = binary.LittleEndian.AppendUint16(out, uint16(h))
	for y := 0; y < h; y++ {
		written := 0
		for written < w {
			mode := e.rng.IntN(4)
			remain := w - written
			if mode == 1 && remain < 2 {
				mode = 3
			}
			limit := remain
			if mode == 1 {
				limit = remain / 2
			}
			if limit > 64 {
				limit = 64
			}
			count := 1 + e.rng.IntN(limit)
			out = append(out, byte(mode<<6|(count-1)))
			switch mode {
			case 0, 1:
				out = append(out, byte(e.rng.IntN(256)))
			case 2:
				out = append(out, e.bytesN(count)...)
			}
			if mode == 1 {
				written += 2 * count
			} else {
				written += count
			}
		}
	}
	if corrupt && len(out) > 4 {
		switch e.rng.IntN(3) {
		case 0:
			out = out[:4+e.rng.IntN(len(out)-4)]
		case 1:
			out[4+e.rng.IntN(len(out)-4)] = byte(e.rng.IntN(256))
		default:
			out = append(out, e.bytesN(1+e.rng.IntN(4))...)
		}
	}
	return out
}

// randFrame 回傳原始或分離表示的 frame 與其 JSON 描述。
func (e *exporter) randFrame(maxW, maxH int) (fdother.Frame, obj) {
	w, h := 1+e.rng.IntN(maxW), 1+e.rng.IntN(maxH)
	x, y := e.rng.IntN(12), e.rng.IntN(12)
	if e.rng.IntN(15) == 0 {
		x, y = e.pick(-3, 12), e.pick(-3, 12)
	}
	if e.rng.IntN(3) == 0 {
		n := w * h
		if e.rng.IntN(12) == 0 {
			n += []int{-1, 1}[e.rng.IntN(2)]
		}
		ix := e.bytesN(n)
		mk := make([]byte, n)
		for i := range mk {
			if e.rng.IntN(2) == 0 {
				mk[i] = 255
			}
			if e.rng.IntN(150) == 0 {
				mk[i] = byte(1 + e.rng.IntN(254))
			}
		}
		f := fdother.Frame{X: x, Y: y, Width: w, Height: h, Indexed: ix, Mask: mk}
		desc := obj{"x": x, "y": y, "w": w, "h": h, "indexed": hexs(ix), "mask": hexs(mk)}
		if e.rng.IntN(25) == 0 {
			f.Pixels = []byte{1}
			desc["pixels"] = "01"
		}
		return f, desc
	}
	px := e.rle(w, h, e.rng.IntN(6) == 0)
	if e.rng.IntN(20) == 0 {
		w += []int{-1, 1}[e.rng.IntN(2)]
		if w == 0 {
			w = 2
		}
	}
	return fdother.Frame{X: x, Y: y, Width: w, Height: h, Pixels: px}, obj{"x": x, "y": y, "w": w, "h": h, "pixels": hexs(px)}
}

func (e *exporter) randBytesEntry(maxW, maxH int, zeroBias bool) (int, int, []byte) {
	w, h := 1+e.rng.IntN(maxW), 1+e.rng.IntN(maxH)
	n := w * h
	if e.rng.IntN(15) == 0 {
		n += []int{-1, 1}[e.rng.IntN(2)]
	}
	px := e.bytesN(n)
	if zeroBias {
		for i := range px {
			if e.rng.IntN(3) == 0 {
				px[i] = 0
			}
		}
	}
	if e.rng.IntN(25) == 0 {
		w = []int{0, -1}[e.rng.IntN(2)]
	}
	return w, h, px
}

// ---- 隨機容器 ----

func (e *exporter) frameContainer() []byte {
	n := 1 + e.rng.IntN(4)
	var frames [][]byte
	for i := 0; i < n; i++ {
		w, h := 1+e.rng.IntN(20), 1+e.rng.IntN(6)
		d := make([]byte, 9)
		binary.LittleEndian.PutUint16(d, uint16(e.rng.IntN(300)))
		binary.LittleEndian.PutUint16(d[2:], uint16(e.rng.IntN(200)))
		copy(d[4:], e.bytesN(5))
		body := e.rle(w, h, e.rng.IntN(8) == 0)
		frames = append(frames, append(d, body...))
	}
	out := make([]byte, 8+4*n)
	binary.LittleEndian.PutUint16(out, uint16(n))
	copy(out[2:8], e.bytesN(6))
	off := len(out)
	for i, f := range frames {
		binary.LittleEndian.PutUint32(out[8+4*i:], uint32(off))
		out = append(out, f...)
		off += len(f)
	}
	return e.mutate(out)
}

func (e *exporter) lmi1Container(kind int) []byte {
	n := 1 + e.rng.IntN(4)
	var cells [][]byte
	for i := 0; i < n; i++ {
		w, h := 1+e.rng.IntN(12), 1+e.rng.IntN(5)
		var body []byte
		switch kind {
		case 0: // 0x4e916 高位連續段
			body = binary.LittleEndian.AppendUint16(nil, uint16(w))
			body = binary.LittleEndian.AppendUint16(body, uint16(h))
			for filled := 0; filled < w*h; {
				if e.rng.IntN(3) == 0 {
					run := 1 + e.rng.IntN(8)
					body = append(body, byte(0xc0+run), byte(e.rng.IntN(256)))
					filled += run
				} else {
					body = append(body, byte(e.rng.IntN(0xc1)))
					filled++
				}
			}
		case 1: // 四模式 RLE
			body = e.rle(w, h, false)
		default: // 原始
			body = binary.LittleEndian.AppendUint16(nil, uint16(w))
			body = binary.LittleEndian.AppendUint16(body, uint16(h))
			body = append(body, e.bytesN(w*h)...)
		}
		cells = append(cells, body)
	}
	out := []byte("LMI1")
	out = binary.LittleEndian.AppendUint16(out, uint16(n))
	off := 6 + 4*n
	for _, c := range cells {
		out = binary.LittleEndian.AppendUint32(out, uint32(off))
		off += len(c)
	}
	for _, c := range cells {
		out = append(out, c...)
	}
	return e.mutate(out)
}

func (e *exporter) rawBank() []byte {
	n := 1 + e.rng.IntN(4)
	out := make([]byte, 4*n)
	var cells [][]byte
	for i := 0; i < n; i++ {
		w, h := 1+e.rng.IntN(10), 1+e.rng.IntN(5)
		c := binary.LittleEndian.AppendUint16(nil, uint16(w))
		c = binary.LittleEndian.AppendUint16(c, uint16(h))
		cells = append(cells, append(c, e.bytesN(w*h)...))
	}
	off := len(out)
	for i, c := range cells {
		binary.LittleEndian.PutUint32(out[4*i:], uint32(off))
		off += len(c)
	}
	for _, c := range cells {
		out = append(out, c...)
	}
	return e.mutate(out)
}

func (e *exporter) lutBank() []byte {
	n := 1 + e.rng.IntN(4)
	out := []byte("LMI1")
	out = binary.LittleEndian.AppendUint16(out, uint16(n))
	for i := 0; i < n; i++ {
		out = binary.LittleEndian.AppendUint32(out, uint32(6+4*n+256*i))
	}
	return e.mutate(append(out, e.bytesN(256*n)...))
}

// mutate 有時截斷、改一個位元組或在尾端加料。
func (e *exporter) mutate(b []byte) []byte {
	switch e.rng.IntN(6) {
	case 0:
		return b[:e.rng.IntN(len(b)+1)]
	case 1:
		if len(b) > 0 {
			b[e.rng.IntN(len(b))] = byte(e.rng.IntN(256))
		}
	case 2:
		return append(b, e.bytesN(1+e.rng.IntN(8))...)
	}
	return b
}

func frameDesc(f fdother.Frame) obj {
	d := obj{"x": f.X, "y": f.Y, "w": f.Width, "h": f.Height}
	if len(f.Pixels) > 0 {
		d["pixels"] = hexs(f.Pixels)
	}
	if len(f.Indexed) > 0 || len(f.Mask) > 0 {
		d["indexed"], d["mask"] = hexs(f.Indexed), hexs(f.Mask)
	}
	return d
}

// ---- 結構化與隨機案例 ----

func (e *exporter) genParsers() {
	e.parseAll("rand-empty", obj{"data": ""}, nil)
	gens := []struct {
		kind string
		gen  func() []byte
	}{
		{"rand-frames", e.frameContainer},
		{"rand-single", func() []byte { return e.mutate(e.rle(1+e.rng.IntN(16), 1+e.rng.IntN(4), false)) }},
		{"rand-lmi1-run", func() []byte { return e.lmi1Container(0) }},
		{"rand-lmi1-rle", func() []byte { return e.lmi1Container(1) }},
		{"rand-lmi1-raw", func() []byte { return e.lmi1Container(2) }},
		{"rand-rawbank", e.rawBank},
		{"rand-lutbank", e.lutBank},
		{"rand-palette", func() []byte {
			b := make([]byte, 768)
			for i := range b {
				b[i] = byte(e.rng.IntN(64))
			}
			if e.rng.IntN(3) == 0 {
				b[e.rng.IntN(768)] = byte(64 + e.rng.IntN(192))
			}
			return e.mutate(b)
		}},
		{"rand-bytes", func() []byte { return e.bytesN(e.rng.IntN(40)) }},
	}
	for _, g := range gens {
		for i := 0; i < 70; i++ {
			data := g.gen()
			e.parseAll(g.kind, obj{"data": hexs(data)}, data)
		}
	}
}

func (e *exporter) genFrameBlits() {
	for i := 0; i < 700; i++ {
		f, desc := e.randFrame(24, 6)
		stride := f.X + f.Width + e.rng.IntN(6) - 1
		base := e.rng.IntN(20)
		rows := f.Y + f.Height + e.rng.IntN(3) - 1
		dstLen := stride*rows + base + e.rng.IntN(4) - 1
		if e.rng.IntN(12) == 0 {
			stride, base = e.pick(-2, 40), e.pick(-2, 40)
		}
		if dstLen < 0 || dstLen > 1<<16 {
			dstLen = e.rng.IntN(200)
		}
		transparent := -1
		if e.rng.IntN(20) == 0 {
			transparent = e.pick(-2, 1)
		}
		seed := e.rng.IntN(256)
		in := obj{"frame": desc, "dst_len": dstLen, "seed": seed, "stride": stride, "base": base, "transparent": transparent}
		useLUT := e.rng.IntN(3) == 0
		lutLen, mul, add := 256, 1+e.rng.IntN(9), e.rng.IntN(256)
		if useLUT {
			if e.rng.IntN(12) == 0 {
				lutLen = []int{0, 255, 257}[e.rng.IntN(3)]
			}
			in["lut"] = []int{lutLen, mul, add}
		}
		dst := pattern(dstLen, seed)
		res, err := protectErr(func() (obj, error) {
			var err error
			if useLUT {
				err = f.BlitLUTAt(dst, stride, base, lutOf(lutLen, mul, add))
			} else {
				err = f.BlitAt(dst, stride, base, transparent)
			}
			return obj{"dst": digest(dst)}, err
		})
		e.add("rand-frame-blit", "frame_blit", in, res, err)
		if i%4 == 0 {
			e.add("rand-frame-info", "frame_info", obj{"frame": desc}, frameInfo(f), nil)
		}
	}
	// TextPlaceholder：驗證原圖後換成回呼。
	for i := 0; i < 150; i++ {
		f, desc := e.randFrame(30, 20)
		if e.rng.IntN(5) == 0 {
			f.X, f.Y = e.pick(0, 300), e.pick(0, 190)
			desc["x"], desc["y"] = f.X, f.Y
		}
		stride := e.rng.IntN(40) + 1
		base := e.rng.IntN(60)
		dstLen := e.rng.IntN(2000)
		seed := e.rng.IntN(256)
		in := obj{"frame": desc, "dst_len": dstLen, "seed": seed, "stride": stride, "base": base}
		var calls [][2]int
		res, err := protectErr(func() (obj, error) {
			p, err := f.TextPlaceholder(func(x, y int) { calls = append(calls, [2]int{x, y}) })
			if err != nil {
				return obj{}, err
			}
			out := obj{"info": frameInfo(p)}
			dst := pattern(dstLen, seed)
			berr := p.BlitAt(dst, stride, base, -1)
			out["blit_err"], out["dst"] = errv(berr), digest(dst)
			return out, nil
		})
		if calls == nil {
			calls = [][2]int{}
		}
		res["calls"] = calls
		e.add("rand-text-placeholder", "text_placeholder", in, res, err)
	}
}

func (e *exporter) genCellBlits() {
	for i := 0; i < 500; i++ {
		w, h, px := e.randBytesEntry(10, 5, true)
		variant := []string{"at", "opaque", "clipped"}[e.rng.IntN(3)]
		stride := e.rng.IntN(20) + 1
		x, y := e.rng.IntN(12)-2, e.rng.IntN(6)-2
		if variant == "clipped" {
			x, y = e.rng.IntN(30)-12, e.rng.IntN(16)-8
		}
		if e.rng.IntN(15) == 0 {
			stride, x, y = e.pick(-2, 20), e.pick(-3, 10), e.pick(-3, 10)
		}
		dstLen, seed, mirror := e.rng.IntN(200), e.rng.IntN(256), e.rng.IntN(2) == 0
		in := obj{"variant": variant, "entry": []any{w, h, hexs(px)}, "dst_len": dstLen, "seed": seed,
			"stride": stride, "x": x, "y": y, "mirror": mirror}
		entry := fdother.LMI1Entry{Width: w, Height: h, Pixels: px}
		dst := pattern(dstLen, seed)
		res, err := protectErr(func() (obj, error) {
			var err error
			switch variant {
			case "at":
				err = entry.BlitAt(dst, stride, x, y, mirror)
			case "opaque":
				err = entry.BlitOpaqueAt(dst, stride, x, y, mirror)
			default:
				err = entry.BlitAtClipped(dst, stride, x, y, mirror)
			}
			return obj{"dst": digest(dst)}, err
		})
		e.add("rand-lmi1-blit", "lmi1_blit", in, res, err)
	}
	for i := 0; i < 500; i++ {
		w, h, px := e.randBytesEntry(10, 5, true)
		variant := []string{"at", "clipped", "offset"}[e.rng.IntN(3)]
		stride := e.rng.IntN(20) + 1
		x, y, offset := e.rng.IntN(12)-2, e.rng.IntN(6)-2, e.rng.IntN(60)-2
		if variant == "clipped" {
			x, y = e.rng.IntN(30)-12, e.rng.IntN(16)-8
		}
		if e.rng.IntN(15) == 0 {
			stride, x, y, offset = e.pick(-2, 20), e.pick(-3, 10), e.pick(-3, 10), e.pick(-3, 60)
		}
		dstLen, seed := e.rng.IntN(200), e.rng.IntN(256)
		in := obj{"variant": variant, "cell": []any{w, h, hexs(px)}, "dst_len": dstLen, "seed": seed,
			"stride": stride, "x": x, "y": y, "offset": offset}
		cell := fdother.RawCell{Width: w, Height: h, Pixels: px}
		dst := pattern(dstLen, seed)
		res, err := protectErr(func() (obj, error) {
			var err error
			switch variant {
			case "at":
				err = cell.BlitAt(dst, stride, x, y)
			case "clipped":
				err = cell.BlitAtClipped(dst, stride, x, y)
			default:
				err = cell.BlitOpaqueAtOffset(dst, stride, offset)
			}
			return obj{"dst": digest(dst)}, err
		})
		e.add("rand-raw-blit", "raw_blit", in, res, err)
	}
}

func (e *exporter) genPalettes() {
	for i := 0; i < 120; i++ {
		w, h, px := e.randBytesEntry(6, 3, true)
		n := 256
		if e.rng.IntN(8) == 0 {
			n = []int{0, 255, 257, 300}[e.rng.IntN(4)]
		}
		pal := make(color.Palette, n)
		desc := make([][4]int, n)
		for j := range pal {
			c := color.NRGBA{R: byte(e.rng.IntN(256)), G: byte(e.rng.IntN(256)), B: byte(e.rng.IntN(256)), A: 0xff}
			if e.rng.IntN(4) == 0 {
				c.A = byte(e.rng.IntN(256))
			}
			pal[j], desc[j] = c, [4]int{int(c.R), int(c.G), int(c.B), int(c.A)}
		}
		in := obj{"cell": []any{w, h, hexs(px)}, "palette": desc}
		res, err := protectErr(func() (obj, error) {
			im, err := fdother.RawCell{Width: w, Height: h, Pixels: px}.Paletted(pal)
			if err != nil {
				return obj{}, err
			}
			out := make([][4]int, len(im.Palette))
			for j, c := range im.Palette {
				nc := c.(color.NRGBA)
				out[j] = [4]int{int(nc.R), int(nc.G), int(nc.B), int(nc.A)}
			}
			return obj{"pix": hexs(im.Pix), "stride": im.Stride, "rect": []int{im.Rect.Dx(), im.Rect.Dy()}, "palette": out}, nil
		})
		e.add("rand-paletted", "raw_paletted", in, res, err)
	}
	for i := 0; i < 300; i++ {
		dac := e.bytesN(768)
		base := make([]byte, 768)
		for j := range base {
			base[j] = byte(e.rng.IntN(64))
		}
		if e.rng.IntN(10) == 0 {
			dac = dac[:e.rng.IntN(770)]
		}
		if e.rng.IntN(10) == 0 {
			base = append(base, 1)
		}
		start, end, delta := e.rng.IntN(256), e.rng.IntN(256), e.rng.IntN(256)
		if start > end && e.rng.IntN(4) != 0 {
			start, end = end, start
		}
		if e.rng.IntN(10) == 0 {
			start, end, delta = e.pick(-2, 258), e.pick(-2, 258), e.pick(-2, 300)
		}
		sub := e.rng.IntN(2) == 0
		in := obj{"sub": sub, "dac": hexs(dac), "baseline": hexs(base), "start": start, "end": end, "delta": delta}
		res, err := protectErr(func() (obj, error) {
			var err error
			if sub {
				err = fdother.ApplyVGAPaletteSubtraction(dac, base, start, end, delta)
			} else {
				err = fdother.ApplyVGAPaletteDelta(dac, base, start, end, delta)
			}
			return obj{"dac": hexs(dac)}, err
		})
		e.add("rand-palette-apply", "palette_apply", in, res, err)
	}
	for i := 0; i < 60; i++ {
		lens := make([]int, e.rng.IntN(4))
		for j := range lens {
			lens[j] = 256
			if e.rng.IntN(5) == 0 {
				lens[j] = []int{0, 255, 257}[e.rng.IntN(3)]
			}
		}
		luts := make([][]byte, len(lens))
		for j, n := range lens {
			luts[j] = lutOf(n, 7+j, j)
		}
		res, err := protectErr(func() (obj, error) {
			b, err := fdother.NativeUnitPresentBridgeLUT(luts)
			return obj{"bridge": hexs(b)}, err
		})
		e.add("rand-bridge", "bridge_lut", obj{"lens": lens}, res, err)
	}
}

func (e *exporter) genRemap() {
	for i := 0; i < 400; i++ {
		radial := e.rng.IntN(2) == 0
		stride := 4 + e.rng.IntN(20)
		rows := 1 + e.rng.IntN(12)
		dstLen := stride*rows + e.rng.IntN(3) - 1
		clip := 1 + e.rng.IntN(stride)
		lutLen, mul, add := 256, 1+e.rng.IntN(9), e.rng.IntN(256)
		if e.rng.IntN(15) == 0 {
			lutLen = 255
		}
		spec := obj{"center_x": e.rng.IntN(clip + 1), "center_y": e.rng.IntN(rows + 4), "radius": e.rng.IntN(10),
			"scale": e.rng.IntN(25), "start_y": e.rng.IntN(rows + 1), "end_y": e.rng.IntN(rows + 2),
			"horizontal_radius": e.rng.IntN(stride+3) - 1, "clip_width": clip}
		if e.rng.IntN(12) == 0 {
			for _, k := range []string{"center_x", "center_y", "radius", "scale", "start_y", "horizontal_radius", "clip_width"} {
				if e.rng.IntN(3) == 0 {
					spec[k] = e.pick(-2, 0x8001)
				}
			}
		}
		if e.rng.IntN(15) == 0 {
			stride = e.pick(-2, 30)
		}
		seed := e.rng.IntN(256)
		in := obj{"radial": radial, "dst_len": dstLen, "seed": seed, "stride": stride, "lut": []int{lutLen, mul, add}, "spec": spec}
		dst := pattern(dstLen, seed)
		g := func(k string) int { return spec[k].(int) }
		res, err := protectErr(func() (obj, error) {
			var err error
			if radial {
				err = fdother.ApplyRadialLUTRemap(dst, stride, lutOf(lutLen, mul, add), fdother.RadialLUTRemap{
					CenterX: g("center_x"), CenterY: g("center_y"), Radius: g("radius"), Scale: g("scale"),
					StartY: g("start_y"), EndY: g("end_y"), ClipWidth: g("clip_width")})
			} else {
				err = fdother.ApplyCenteredRectLUTRemap(dst, stride, lutOf(lutLen, mul, add), fdother.CenteredRectLUTRemap{
					CenterX: g("center_x"), HorizontalRadius: g("horizontal_radius"),
					StartY: g("start_y"), EndY: g("end_y"), ClipWidth: g("clip_width")})
			}
			return obj{"dst": digest(dst)}, err
		})
		e.add("rand-remap", "remap", in, res, err)
	}
}

func (e *exporter) defineBank(name string, count int) {
	b := &fdicon.Bank{Sprites: make([]fdicon.Sprite, count)}
	rows := make([][3]string, count)
	for i := range b.Sprites {
		layer := func(density int) []byte {
			l := make([]byte, 24*24)
			for j := range l {
				if e.rng.IntN(100) < density {
					l[j] = 1
				}
			}
			return l
		}
		s := fdicon.Sprite{Pixels: e.bytesN(24 * 24), Mask: layer(40), RemapMask: layer(10)}
		if i == 7 && name == "range-short-pixels" {
			s.Pixels = s.Pixels[:100]
		}
		b.Sprites[i] = s
		rows[i] = [3]string{hexs(s.Pixels), hexs(s.Mask), hexs(s.RemapMask)}
	}
	e.banks[name], e.bankObjs[name] = rows, b
}

func (e *exporter) genRange() {
	for mode := -1; mode <= 8; mode++ {
		for _, c := range [][2]int{{10, 20}, {0, 0}, {-3, 7}, {1 << 62, 1 << 62}, {-(1 << 62), 5}} {
			res, err := protectErr(func() (obj, error) {
				ps, err := fdother.NativeRangeOverlayPlacements(mode, c[0], c[1])
				out := make([][3]int, len(ps))
				for i, p := range ps {
					out[i] = [3]int{p.X, p.Y, p.Descriptor}
				}
				return obj{"placements": out}, err
			})
			e.add("range-placements", "range_placements", obj{"mode": mode, "cx": c[0], "cy": c[1]}, res, err)
		}
	}
	for i := 0; i < 80; i++ {
		w, cx, cy := e.pick(-2, 40), e.pick(-5, 40), e.pick(-5, 40)
		res, err := protectErr(func() (obj, error) {
			v, err := fdother.NativeRangeOverlayMode6ByteAddress(w, cx, cy)
			return obj{"value": v}, err
		})
		e.add("range-mode6", "mode6_address", obj{"w": w, "cx": cx, "cy": cy}, res, err)
	}
	for i := 0; i < 150; i++ {
		w, h := e.rng.IntN(8)-1, e.rng.IntN(6)-1
		if e.rng.IntN(10) == 0 {
			w, h = []int{0x7fff, 0x8000, 0xffff}[e.rng.IntN(3)], e.rng.IntN(4)
		}
		n := 4 + 4*w*h + e.rng.IntN(5) - 2
		if n < 0 || n > 4096 {
			n = e.rng.IntN(40)
		}
		field := e.bytesN(n)
		if n >= 4 {
			binary.LittleEndian.PutUint16(field, uint16(w))
			binary.LittleEndian.PutUint16(field[2:], uint16(h))
		}
		cx, cy := e.rng.IntN(10)-1, e.rng.IntN(8)-1
		in := obj{"field": hexs(field), "cx": cx, "cy": cy}
		res, err := protectErr(func() (obj, error) {
			err := fdother.ClearNativeRangeOverlayMode6FieldByte(field, cx, cy)
			return obj{"field": hexs(field)}, err
		})
		e.add("range-mode6-clear", "mode6_clear", in, res, err)
	}
	e.defineBank("range", 20)
	e.defineBank("range-19", 19)
	e.defineBank("range-short-pixels", 20)
	for i := 0; i < 300; i++ {
		bank := "range"
		if e.rng.IntN(10) == 0 {
			bank = []string{"range-19", "range-short-pixels"}[e.rng.IntN(2)]
		}
		rows := 120 + e.rng.IntN(120)
		dstLen := 0x1c8*rows + []int{0, 0, 0, 1}[e.rng.IntN(4)]
		camX, camY := e.rng.IntN(20), e.rng.IntN(20)
		vw, vh := 1+e.rng.IntN(14), 1+e.rng.IntN(9)
		mode := e.rng.IntN(8) - 1
		cx, cy := camX+e.rng.IntN(16)-2, camY+e.rng.IntN(12)-2
		if e.rng.IntN(15) == 0 {
			vw, vh, camX, cx = e.pick(-1, 14), e.pick(-1, 9), e.pick(-2, 20), e.pick(-2, 20)
		}
		seed := e.rng.IntN(256)
		in := obj{"bank": bank, "dst_len": dstLen, "seed": seed, "camera_x": camX, "camera_y": camY,
			"visible_width": vw, "visible_height": vh, "mode": mode, "cx": cx, "cy": cy}
		dst := pattern(dstLen, seed)
		b := e.bankObjs[bank]
		res, err := protectErr(func() (obj, error) {
			err := fdother.BlitNativeRangeOverlay(b, dst, camX, camY, vw, vh, mode, cx, cy)
			return obj{"dst": digest(dst)}, err
		})
		e.add("range-blit", "range_blit", in, res, err)
	}
}

func (e *exporter) genSurfaces() {
	// 視差底面建構：大多數案例用 462x113 的不透明分離圖，少數改變幾何或遮罩。
	for i := 0; i < 40; i++ {
		var frames [2]fdother.Frame
		var descs [2]obj
		for k := range frames {
			w, h := fdother.NativeMapParallaxWidth, fdother.NativeMapParallaxHeight/2
			ix := e.bytesN(w * h)
			mk := make([]byte, w*h)
			for j := range mk {
				mk[j] = 255
			}
			f := fdother.Frame{Width: w, Height: h, Indexed: ix, Mask: mk}
			switch e.rng.IntN(8) {
			case 0:
				f.Mask[e.rng.IntN(len(mk))] = 0
			case 1:
				f.Height--
				f.Indexed, f.Mask = f.Indexed[:w*(h-1)], f.Mask[:w*(h-1)]
			case 2:
				f.X = 1
			case 3:
				f.Mask[e.rng.IntN(len(mk))] = 7
			}
			frames[k], descs[k] = f, frameDesc(f)
		}
		res, err := protectErr(func() (obj, error) {
			s, err := fdother.BuildNativeMapParallax(frames[0], frames[1])
			if err != nil {
				return obj{}, err
			}
			return obj{"pixels": digest(s.Pixels)}, nil
		})
		e.add("parallax-build", "parallax_build", obj{"top": descs[0], "bottom": descs[1]}, res, err)
	}
	for i := 0; i < 250; i++ {
		surfaceLen := 462 * 226
		if e.rng.IntN(20) == 0 {
			surfaceLen--
		}
		stride := 312 + e.rng.IntN(200)
		dstLen := 191*stride + 312 + e.rng.IntN(5) - 2
		cx, cy, sx, sy := e.rng.IntN(54)-1, e.rng.IntN(21)-1, e.rng.IntN(15)-7, e.rng.IntN(15)-7
		if e.rng.IntN(15) == 0 {
			stride, dstLen = e.pick(300, 600), e.rng.IntN(1000)
		}
		seed := e.rng.IntN(256)
		in := obj{"surface_len": surfaceLen, "dst_len": dstLen, "seed": seed, "stride": stride, "cx": cx, "cy": cy, "sx": sx, "sy": sy}
		dst := pattern(dstLen, seed)
		s := &fdother.NativeMapParallaxSurface{Pixels: pattern(surfaceLen, 5)}
		res, err := protectErr(func() (obj, error) {
			err := fdother.BlitNativeMapParallaxViewport(dst, stride, s, cx, cy, sx, sy)
			return obj{"dst": digest(dst)}, err
		})
		e.add("parallax-viewport", "parallax_viewport", in, res, err)
	}
	for chapter := -1; chapter <= 32; chapter++ {
		c, ok := fdother.NativeChapterAuxSurfaceFor(chapter)
		e.add("aux-for", "aux_for", obj{"chapter": chapter}, obj{"ok": ok, "contract": []any{c.Resource, c.RawMD5, c.RawSHA256}}, nil)
	}
	var contracts [][]any
	for _, c := range fdother.NativeChapterAuxSurfaceContracts() {
		contracts = append(contracts, []any{c.Resource, c.RawMD5, c.RawSHA256})
	}
	e.add("aux-contracts", "aux_contracts", obj{}, obj{"contracts": contracts}, nil)
	for i := 0; i < 150; i++ {
		surfaceLen := 320 * 200
		if e.rng.IntN(20) == 0 {
			surfaceLen++
		}
		stride := 310 + e.rng.IntN(40)
		dstLen := stride*192 + e.rng.IntN(5) - 3
		phase := e.rng.IntN(18) - 1
		if e.rng.IntN(20) == 0 {
			stride = e.pick(300, 400)
		}
		seed := e.rng.IntN(256)
		in := obj{"surface_len": surfaceLen, "dst_len": dstLen, "seed": seed, "stride": stride, "phase": phase}
		dst := pattern(dstLen, seed)
		s := &fdother.NativeChapterAuxSurface{Pixels: pattern(surfaceLen, 9)}
		res, err := protectErr(func() (obj, error) {
			err := fdother.BlitNativeChapterAuxViewport(dst, stride, s, phase)
			return obj{"dst": digest(dst)}, err
		})
		e.add("aux-viewport", "aux_viewport", in, res, err)
	}
}

var errPresent = errors.New("present failed")
var errRestore = errors.New("restore failed")

func (e *exporter) genFlow() {
	var steps [][5]int
	for _, s := range fdother.NativeBattleEntrySplitSlideSteps() {
		steps = append(steps, [5]int{s.Offset, s.LeftX, s.LeftY, s.RightX, s.RightY})
	}
	e.add("split-steps", "split_steps", obj{}, obj{"steps": steps}, nil)
	for i := 0; i < 120; i++ {
		w, h, px := e.randBytesEntry(80, 20, true)
		step := [5]int{e.rng.IntN(120), e.rng.IntN(500) - 100, e.rng.IntN(240) - 20, e.rng.IntN(500) - 100, e.rng.IntN(240) - 20}
		stride := 456
		if e.rng.IntN(8) == 0 {
			stride = e.pick(0, 500)
		}
		dstLen, seed := []int{456 * 200, 456 * 100, 456*200 + 7, 0}[e.rng.IntN(4)], e.rng.IntN(256)
		in := obj{"cell": []any{w, h, hexs(px)}, "step": step, "dst_len": dstLen, "seed": seed, "stride": stride}
		dst := pattern(dstLen, seed)
		cell := fdother.LMI1Entry{Width: w, Height: h, Pixels: px}
		res, err := protectErr(func() (obj, error) {
			err := fdother.BlitNativeBattleEntrySplitSlideCell(cell, fdother.NativeBattleEntrySplitSlideStep{
				Offset: step[0], LeftX: step[1], LeftY: step[2], RightX: step[3], RightY: step[4]}, dst, stride)
			return obj{"dst": digest(dst)}, err
		})
		e.add("split-cell", "split_cell", in, res, err)
	}
	for i := 0; i < 40; i++ {
		w, h, px := e.randBytesEntry(80, 20, true)
		failPresent, failRestore := e.rng.IntN(8)-1, e.rng.IntN(8)-1
		dstLen, seed := []int{456 * 200, 456 * 60}[e.rng.IntN(2)], e.rng.IntN(256)
		in := obj{"cell": []any{w, h, hexs(px)}, "dst_len": dstLen, "seed": seed, "fail_present": failPresent, "fail_restore": failRestore}
		dst := pattern(dstLen, seed)
		events := []string{}
		presents, restores := 0, 0
		res, err := protectErr(func() (obj, error) {
			err := fdother.RunNativeBattleEntrySplitSlide(fdother.LMI1Entry{Width: w, Height: h, Pixels: px}, dst,
				func() error {
					events = append(events, "present")
					presents++
					if presents == failPresent {
						return errPresent
					}
					return nil
				}, func() error {
					events = append(events, "restore")
					restores++
					if restores == failRestore {
						return errRestore
					}
					return nil
				})
			return obj{"dst": digest(dst)}, err
		})
		res["events"] = events
		e.add("split-run", "split_run", in, res, err)
	}
	for i := 0; i < 250; i++ {
		stride, x, y, cols, rows := 16+e.rng.IntN(400), e.rng.IntN(60)-2, e.rng.IntN(60)-2, e.rng.IntN(12), e.rng.IntN(12)
		if e.rng.IntN(15) == 0 {
			stride, x, y = e.pick(-1, 400), e.pick(-1, 60), e.pick(-1, 60)
		}
		in := obj{"stride": stride, "x": x, "y": y, "columns": cols, "rows": rows}
		res, err := protectErr(func() (obj, error) {
			ps, err := fdother.PlanNativeDialogueFrameGrid(stride, x, y, cols, rows)
			out := make([][2]int, len(ps))
			for j, p := range ps {
				out[j] = [2]int{p.ResourceIndex, p.DestinationByte}
			}
			return obj{"placements": out}, err
		})
		e.add("dialogue-grid", "dialogue_grid", in, res, err)
	}
	for phase := -2; phase <= 21; phase++ {
		res, err := protectErr(func() (obj, error) {
			v, err := fdother.NativeTerrainLUTIndex(phase)
			return obj{"value": v}, err
		})
		e.add("terrain-lut", "terrain_lut", obj{"phase": phase}, res, err)
	}
	for i := 0; i < 300; i++ {
		st := fdother.NativeTerrainPhaseState{Phase: e.rng.IntN(22) - 1, LastTimerTick: e.rng.IntN(0x10002) - 0x8001}
		tick := e.rng.IntN(0x10002) - 0x8001
		if e.rng.IntN(2) == 0 {
			tick = st.LastTimerTick + e.rng.IntN(7) - 2
		}
		override := -1
		if e.rng.IntN(4) == 0 {
			override = e.rng.IntN(24) - 2
		}
		in := obj{"phase": st.Phase, "last": st.LastTimerTick, "tick": tick, "override": override}
		res, err := protectErr(func() (obj, error) {
			got, err := fdother.AdvanceNativeTerrainPhase(st, tick, override)
			return obj{"state": []int{got.Phase, got.LastTimerTick}}, err
		})
		e.add("terrain-phase", "terrain_phase", in, res, err)
	}
}
