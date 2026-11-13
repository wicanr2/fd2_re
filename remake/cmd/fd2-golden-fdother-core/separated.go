package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"image"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

// ---- PNG 產生（依 PNG 規範，與 fd2-golden-fdicon 相同的作法） ----

func chunk(typ string, data []byte) []byte {
	out := binary.BigEndian.AppendUint32(nil, uint32(len(data)))
	out = append(out, typ...)
	out = append(out, data...)
	return binary.BigEndian.AppendUint32(out, crc32.ChecksumIEEE(append([]byte(typ), data...)))
}

type pngSpec struct {
	w, h         int
	depth, ctype byte
	interlace    byte
	sample       func(x, y int) []int
	plte         int
	trns         []byte
}

func channels(ctype byte) int { return map[byte]int{0: 1, 2: 3, 3: 1, 4: 2, 6: 4}[ctype] }

func packRow(s pngSpec, xs []int, y int) []byte {
	bits := int(s.depth) * channels(s.ctype)
	row := make([]byte, (bits*len(xs)+7)/8)
	bit := 0
	for _, x := range xs {
		for _, v := range s.sample(x, y) {
			switch s.depth {
			case 16:
				row[bit/8], row[bit/8+1] = byte(v>>8), byte(v)
			case 8:
				row[bit/8] = byte(v)
			default:
				shift := 8 - int(s.depth) - bit%8
				row[bit/8] |= byte(v&(1<<s.depth-1)) << shift
			}
			bit += int(s.depth)
		}
	}
	return row
}

var adam7 = [7][4]int{{8, 8, 0, 0}, {8, 8, 4, 0}, {4, 8, 0, 4}, {4, 4, 2, 0}, {2, 4, 0, 2}, {2, 2, 1, 0}, {1, 2, 0, 1}}

func scanlines(s pngSpec) []byte {
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
	for _, p := range passes {
		for _, y := range p.ys {
			out = append(out, 0) // 過濾類型 None
			out = append(out, packRow(s, p.xs, y)...)
		}
	}
	return out
}

func buildPNG(s pngSpec) []byte {
	out := []byte("\x89PNG\r\n\x1a\n")
	ihdr := binary.BigEndian.AppendUint32(nil, uint32(s.w))
	ihdr = binary.BigEndian.AppendUint32(ihdr, uint32(s.h))
	ihdr = append(ihdr, s.depth, s.ctype, 0, 0, s.interlace)
	out = append(out, chunk("IHDR", ihdr)...)
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
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	w.Write(scanlines(s))
	w.Close()
	out = append(out, chunk("IDAT", z.Bytes())...)
	return append(out, chunk("IEND", nil)...)
}

// buildPNGWithIDATTail 在 IDAT 的 zlib 串流之後多放 tail（規範外）。
func buildPNGWithIDATTail(s pngSpec, tail []byte) []byte {
	out := []byte("\x89PNG\r\n\x1a\n")
	ihdr := binary.BigEndian.AppendUint32(nil, uint32(s.w))
	ihdr = binary.BigEndian.AppendUint32(ihdr, uint32(s.h))
	ihdr = append(ihdr, s.depth, s.ctype, 0, 0, s.interlace)
	out = append(out, chunk("IHDR", ihdr)...)
	pal := make([]byte, 3*s.plte)
	for i := range pal {
		pal[i] = byte(i * 7)
	}
	out = append(out, chunk("PLTE", pal)...)
	var z bytes.Buffer
	w := zlib.NewWriter(&z)
	w.Write(scanlines(s))
	w.Close()
	out = append(out, chunk("IDAT", append(z.Bytes(), tail...))...)
	return append(out, chunk("IEND", nil)...)
}

func palSample(mod int) func(x, y int) []int {
	return func(x, y int) []int { return []int{(x*37 + y*11 + x*y) % mod} }
}

func maskSample(x, y int) []int {
	if (x*3+y*5)%7 < 3 {
		return []int{255}
	}
	return []int{0}
}

// pngVariants 依尺寸產生一組 PNG；第二個值為真表示格式錯誤（accept_reject）。
func pngVariants(w, h int) []struct {
	name      string
	data      []byte
	malformed bool
} {
	type v = struct {
		name      string
		data      []byte
		malformed bool
	}
	pal := func(depth byte, plte, mod int, il byte) []byte {
		return buildPNG(pngSpec{w: w, h: h, depth: depth, ctype: 3, plte: plte, interlace: il, sample: palSample(mod)})
	}
	gray := func(depth byte, sample func(x, y int) []int, il byte, trns []byte) []byte {
		return buildPNG(pngSpec{w: w, h: h, depth: depth, ctype: 0, interlace: il, sample: sample, trns: trns})
	}
	idx255 := func(x, y int) []int {
		if x == 0 && y == 0 {
			return []int{255}
		}
		return []int{(x + y) % 16}
	}
	low := func(levels int) func(x, y int) []int {
		return func(x, y int) []int {
			if maskSample(x, y)[0] != 0 {
				return []int{levels - 1}
			}
			return []int{0}
		}
	}
	good := pal(8, 256, 256, 0)
	out := []v{
		{"png-pal8", good, false},
		{"png-pal8-plte16", pal(8, 16, 16, 0), false},
		{"png-pal8-plte16-idx255", buildPNG(pngSpec{w: w, h: h, depth: 8, ctype: 3, plte: 16, sample: idx255}), false},
		{"png-pal8-plte255", pal(8, 255, 255, 0), false},
		{"png-pal4", pal(4, 16, 16, 0), false},
		{"png-pal8-interlaced", pal(8, 256, 256, 1), false},
		{"png-pal8-trns", buildPNG(pngSpec{w: w, h: h, depth: 8, ctype: 3, plte: 256, sample: palSample(256), trns: []byte{0, 1, 2}}), false},
		{"png-gray8", gray(8, maskSample, 0, nil), false},
		{"png-gray8-nonbinary", gray(8, func(x, y int) []int { return []int{(x + y) % 3 * 7} }, 0, nil), false},
		{"png-gray8-interlaced", gray(8, maskSample, 1, nil), false},
		{"png-gray1", gray(1, low(2), 0, nil), false},
		{"png-gray2", gray(2, low(4), 0, nil), false},
		{"png-gray2-mid", gray(2, func(x, y int) []int { return []int{(x + y) % 4} }, 0, nil), false},
		{"png-gray8-trns", gray(8, maskSample, 0, []byte{0, 0}), false},
		{"png-gray16", buildPNG(pngSpec{w: w, h: h, depth: 16, ctype: 0, sample: func(x, y int) []int { return []int{maskSample(x, y)[0] * 257} }}), false},
		{"png-rgb8", buildPNG(pngSpec{w: w, h: h, depth: 8, ctype: 2, sample: func(x, y int) []int { return []int{x, y, 0} }}), false},
		{"png-rgba8", buildPNG(pngSpec{w: w, h: h, depth: 8, ctype: 6, sample: func(x, y int) []int { return []int{x, y, 0, 255} }}), false},
		{"png-pal8-wide", buildPNG(pngSpec{w: w + 1, h: h, depth: 8, ctype: 3, plte: 256, sample: palSample(256)}), false},
		{"png-gray8-short", buildPNG(pngSpec{w: w, h: max(1, h-1), depth: 8, ctype: 0, sample: maskSample}), false},
		{"png-empty", []byte{}, true},
		{"png-not-png", []byte("this is not a PNG file at all"), true},
		{"png-truncated", good[:len(good)/2], true},
		{"png-no-iend", good[:len(good)-12], true},
		{"png-trailing", append(append([]byte(nil), good...), []byte("junk after IEND")...), true},
	}
	bad := append([]byte(nil), good...)
	bad[len(bad)-12-5] ^= 0xff // IDAT 的 CRC
	out = append(out, v{"png-bad-crc", bad, true})
	return out
}

// ---- 暫存目錄樹 ----

func (e *exporter) blob(content []byte) string {
	key := digest(content)
	if _, ok := e.blobs[key]; !ok {
		e.blobs[key] = hexs(content)
	}
	return key
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func (e *exporter) mirror(realRel, treeRel, dir string) {
	src := filepath.Join(e.assets, realRel)
	dst := filepath.Join(dir, treeRel)
	info, err := os.Stat(src)
	must(err)
	if !info.IsDir() {
		must(os.MkdirAll(filepath.Dir(dst), 0o755))
		must(os.Symlink(src, dst))
		return
	}
	must(filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		out := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		return os.Symlink(p, out)
	}))
}

func (e *exporter) materialize(t treeSpec, dir string) {
	must(os.MkdirAll(dir, 0o755))
	for _, l := range t.Links {
		e.mirror(l[0], l[1], dir)
	}
	for _, p := range t.Remove {
		must(os.RemoveAll(filepath.Join(dir, p)))
	}
	for _, p := range t.Dirs {
		must(os.MkdirAll(filepath.Join(dir, p), 0o755))
	}
	for _, f := range t.Files {
		path := filepath.Join(dir, f[0])
		os.Remove(path)
		must(os.MkdirAll(filepath.Dir(path), 0o755))
		must(os.WriteFile(path, unhex(e.blobs[f[1]]), 0o644))
	}
}

// ---- 分離載入器 ----

func frameMap[T any](m map[int]T, info func(T) any) obj {
	out := obj{}
	for k, v := range m {
		out[strconv.Itoa(k)] = info(v)
	}
	return out
}

func framesInfo(frames []fdother.Frame) []obj {
	out := make([]obj, len(frames))
	for i, f := range frames {
		out[i] = frameInfo(f)
	}
	return out
}

func runLoader(op, root string, args obj) (obj, error) {
	ia := func(k string) int { return args[k].(int) }
	res := obj{}
	var err error
	switch op {
	case "load_single_frame":
		var f fdother.Frame
		f, err = fdother.LoadSeparatedSingleFrame(root, args["source"].(string), ia("resource"))
		if err == nil {
			res["frame"] = frameInfo(f)
		}
	case "load_nested_single_frame":
		var f fdother.Frame
		f, err = fdother.LoadSeparatedNestedSingleFrame(root, ia("resource"), ia("nested"))
		if err == nil {
			res["frame"] = frameInfo(f)
		}
	case "load_palette":
		var dac []byte
		var pal []byte
		d, p, perr := fdother.LoadSeparatedFDOTHERPalette(root, ia("resource"))
		err, dac = perr, d
		if err == nil {
			for _, c := range p {
				r, g, b, a := c.RGBA()
				pal = binary.LittleEndian.AppendUint32(pal, r)
				pal = binary.LittleEndian.AppendUint32(pal, g)
				pal = binary.LittleEndian.AppendUint32(pal, b)
				pal = binary.LittleEndian.AppendUint32(pal, a)
			}
			res["dac"], res["palette"] = digest(dac), digest(pal)
		}
	case "load_item_panel":
		var en fdother.SeparatedItemPanelEntries
		en, err = fdother.LoadSeparatedItemPanelEntries(root)
		if err == nil {
			res["opaque"] = frameMap(en.Opaque, func(v fdother.LMI1Entry) any { return lmiInfo(v) })
			res["raw"] = frameMap(en.Raw, func(v fdother.RawCell) any { return rawInfo(v) })
			res["frames"] = frameMap(en.Frames, func(v fdother.Frame) any { return frameInfo(v) })
		}
	case "load_system_info":
		var panels [4]fdother.Frame
		panels, err = fdother.LoadSeparatedSystemInfoPanels(root)
		if err == nil {
			res["panels"] = framesInfo(panels[:])
		}
	case "load_lmi1_bank":
		var entries []fdother.LMI1Entry
		entries, err = fdother.LoadSeparatedLMI1Bank(root, ia("resource"))
		if err == nil {
			infos := make([][]any, len(entries))
			for i, en := range entries {
				infos[i] = lmiInfo(en)
			}
			res["entries"] = infos
		}
	case "load_load_slots":
		var en fdother.LMI1Entry
		en, err = fdother.LoadSeparatedLoadSlotsFrame(root)
		if err == nil {
			res["entry"] = lmiInfo(en)
		}
	case "load_action_cells":
		var cells []fdother.RawCell
		cells, err = fdother.LoadSeparatedActionCells(root)
		if err == nil {
			infos := make([][]any, len(cells))
			for i, c := range cells {
				infos[i] = rawInfo(c)
			}
			res["cells"] = infos
		}
	case "load_church":
		var a fdother.SeparatedChurchUIAssets
		a, err = fdother.LoadSeparatedChurchUIAssets(root)
		if err == nil {
			entries := make([][]any, len(a.Entries))
			for i, en := range a.Entries {
				entries[i] = lmiInfo(en)
			}
			res["background"], res["entries"], res["price"], res["revive"] = frameInfo(a.Background), entries, rawInfo(a.PriceCell), framesInfo(a.ReviveFX)
		}
	case "load_chapter_aux", "load_chapter_aux_resource":
		var s *fdother.NativeChapterAuxSurface
		if op == "load_chapter_aux" {
			s, err = fdother.LoadSeparatedChapterAuxSurface(root)
		} else {
			c := args["contract"].([]any)
			s, err = fdother.LoadSeparatedChapterAuxSurfaceResource(root, fdother.NativeChapterAuxSurfaceContract{
				Resource: c[0].(int), RawMD5: c[1].(string), RawSHA256: c[2].(string)})
		}
		if err == nil {
			res["pixels"] = digest(s.Pixels)
		}
	case "load_sound":
		var b fdother.SeparatedSoundBank
		b, err = fdother.LoadSeparatedSoundBank(root, ia("resource"))
		if err == nil {
			res["resource"], res["encoded"] = b.Resource, frameMap(b.Encoded, func(v []byte) any { return digest(v) })
		}
	case "load_lut_bank":
		var luts [][]byte
		luts, err = fdother.LoadSeparatedLUTBank(root)
		if err == nil {
			ds := make([]string, len(luts))
			for i, l := range luts {
				ds[i] = digest(l)
			}
			res["luts"] = ds
		}
	case "load_range_overlay":
		var b *fdicon.Bank
		b, err = fdother.LoadSeparatedRangeOverlayBank(root)
		if err == nil {
			res["bank"] = bankInfo(b)
		}
	case "load_frame_bank":
		var frames []fdother.Frame
		switch args["which"].(string) {
		case "event61":
			frames, err = fdother.LoadSeparatedEvent61Frames(root)
		case "ending_prefix":
			frames, err = fdother.LoadSeparatedEndingPrefixFrames(root)
		case "ch20_sky_key":
			frames, err = fdother.LoadSeparatedCh20SkyKeyFrames(root)
		default:
			frames, err = fdother.LoadSeparatedPendingCode1Frames(root)
		}
		if err == nil {
			res["frames"] = framesInfo(frames)
		}
	case "load_parallax":
		var s *fdother.NativeMapParallaxSurface
		s, err = fdother.LoadSeparatedNativeMapParallax(root)
		if err == nil {
			res["pixels"] = digest(s.Pixels)
		}
	default:
		panic(op)
	}
	return res, err
}

func (e *exporter) loadReal(kind, op, rel string, args obj) {
	in := copyObj(args)
	in["root"] = obj{"real": rel}
	root := filepath.Join(e.assets, rel)
	if rel == "<empty>" {
		in["root"], root = obj{"literal": ""}, ""
	}
	res, err := protectErr(func() (obj, error) { return runLoader(op, root, args) })
	e.add(kind, op, in, res, err)
}

func (e *exporter) loadTree(kind, op string, t treeSpec, args obj) {
	e.treeCount++
	name := fmt.Sprintf("t%04d", e.treeCount)
	e.trees[name] = t
	dir := filepath.Join(e.tmp, name)
	e.materialize(t, dir)
	in := copyObj(args)
	in["root"] = obj{"tree": name}
	res, err := protectErr(func() (obj, error) { return runLoader(op, dir, args) })
	if s, ok := res["err"].(string); ok {
		res["err"] = strings.ReplaceAll(s, dir, "$ROOT")
	}
	e.add(kind, op, in, res, err)
	must(os.RemoveAll(dir))
}

func (e *exporter) genRealPack() {
	k := "real-pack"
	for _, src := range []string{"BG.DAT", "TAI.DAT"} {
		for r := -1; r <= 57; r++ {
			e.loadReal(k, "load_single_frame", "surfaces", obj{"source": src, "resource": r})
		}
	}
	for r := 0; r <= 103; r++ {
		e.loadReal(k, "load_single_frame", "surfaces", obj{"source": "FDOTHER.DAT", "resource": r})
	}
	e.loadReal(k, "load_single_frame", "surfaces", obj{"source": "X.DAT", "resource": 0})
	e.loadReal(k, "load_single_frame", "<empty>", obj{"source": "BG.DAT", "resource": 0})
	for r := 6; r <= 8; r++ {
		for n := -1; n <= 8; n++ {
			e.loadReal(k, "load_nested_single_frame", "surfaces", obj{"resource": r, "nested": n})
		}
	}
	e.loadReal(k, "load_nested_single_frame", "<empty>", obj{"resource": 7, "nested": 0})
	for r := -1; r <= 103; r++ {
		e.loadReal(k, "load_palette", "palette", obj{"resource": r})
	}
	e.loadReal(k, "load_palette", "<empty>", obj{"resource": 0})
	for _, rel := range []string{"ui", "surfaces"} {
		e.loadReal(k, "load_item_panel", rel, obj{})
		e.loadReal(k, "load_system_info", rel, obj{})
		e.loadReal(k, "load_load_slots", rel, obj{})
		e.loadReal(k, "load_action_cells", rel, obj{})
		e.loadReal(k, "load_church", rel, obj{})
	}
	lmiRoots := map[int]string{5: "ui/fdother_005_lmi1_opaque", 6: "effects/fdother_006_lmi1_opaque", 9: "animations/fdother_009_spawn_intro"}
	for _, r := range []int{4, 5, 6, 9, 10} {
		for _, rr := range []int{5, 6, 9} {
			e.loadReal(k, "load_lmi1_bank", lmiRoots[rr], obj{"resource": r})
		}
	}
	e.loadReal(k, "load_chapter_aux", "surfaces", obj{})
	e.loadReal(k, "load_chapter_aux", "ui", obj{})
	for _, c := range fdother.NativeChapterAuxSurfaceContracts() {
		for _, res := range []int{15, 55, 16} {
			e.loadReal(k, "load_chapter_aux_resource", "surfaces", obj{"contract": []any{res, c.RawMD5, c.RawSHA256}})
		}
	}
	for r := -1; r <= 100; r++ {
		e.loadReal(k, "load_sound", "sfx", obj{"resource": r})
	}
	e.loadReal(k, "load_sound", "<empty>", obj{"resource": 82})
	e.loadReal(k, "load_lut_bank", "palette", obj{})
	e.loadReal(k, "load_lut_bank", "ui", obj{})
	e.loadReal(k, "load_range_overlay", "sprites/fdother_001_range_overlay", obj{})
	e.loadReal(k, "load_range_overlay", "ui", obj{})
	banks := map[string]string{"event61": "animations/fdother_045_event61", "ending_prefix": "animations/fdother_054_ending_prefix",
		"ch20_sky_key": "animations/fdother_034_ch20_sky_key", "pending_code1": "animations/fdother_079_pending_code1"}
	for _, which := range []string{"event61", "ending_prefix", "ch20_sky_key", "pending_code1"} {
		e.loadReal(k, "load_frame_bank", banks[which], obj{"which": which})
		e.loadReal(k, "load_frame_bank", banks["event61"], obj{"which": which})
	}
	e.loadReal(k, "load_parallax", "surfaces", obj{})
	e.loadReal(k, "load_parallax", "ui", obj{})
}

// ---- 變異樹 ----

type loaderSpec struct {
	op    string
	args  obj
	links [][2]string
	json  string   // 樹內 metadata 路徑
	pngs  []string // 樹內要換掉的 PNG
}

var loaderSpecs = []loaderSpec{
	{"load_single_frame", obj{"source": "BG.DAT", "resource": 3}, [][2]string{{"surfaces/BG_003", "BG_003"}}, "BG_003/resource.json", []string{"BG_003/frame.png", "BG_003/mask.png"}},
	{"load_single_frame", obj{"source": "TAI.DAT", "resource": 1}, [][2]string{{"surfaces/TAI_001", "TAI_001"}}, "TAI_001/resource.json", nil},
	{"load_single_frame", obj{"source": "FDOTHER.DAT", "resource": 56}, [][2]string{{"surfaces/FDOTHER_056", "FDOTHER_056"}}, "FDOTHER_056/resource.json", nil},
	{"load_nested_single_frame", obj{"resource": 7, "nested": 2}, [][2]string{{"surfaces/FDOTHER_007/entry_002", "FDOTHER_007/entry_002"}}, "FDOTHER_007/entry_002/resource.json", []string{"FDOTHER_007/entry_002/frame.png"}},
	{"load_palette", obj{"resource": 57}, [][2]string{{"palette/fdother_057.json", "fdother_057.json"}}, "fdother_057.json", nil},
	{"load_item_panel", obj{}, [][2]string{{"ui/fdother_005_item_panel", "fdother_005_item_panel"}}, "fdother_005_item_panel/resource.json",
		[]string{"fdother_005_item_panel/entry_020/frame.png", "fdother_005_item_panel/entry_000/frame.png", "fdother_005_item_panel/entry_031/frame.png", "fdother_005_item_panel/entry_031/mask.png"}},
	{"load_system_info", obj{}, [][2]string{{"ui/fdother_005_item_panel", "fdother_005_item_panel"}}, "fdother_005_item_panel/resource.json", nil},
	{"load_lmi1_bank", obj{"resource": 9}, [][2]string{{"animations/fdother_009_spawn_intro", ""}}, "bank.json", []string{"entry_000/frame.png"}},
	{"load_load_slots", obj{}, [][2]string{{"ui/fdother_013_load_slots", "fdother_013_load_slots"}}, "fdother_013_load_slots/resource.json", []string{"fdother_013_load_slots/entry_016/frame.png"}},
	{"load_action_cells", obj{}, [][2]string{{"ui/action_cells", "action_cells"}}, "action_cells/resource.json", []string{"action_cells/cell_000.png"}},
	{"load_church", obj{}, [][2]string{{"ui/fdother_014_church", "fdother_014_church"}}, "fdother_014_church/resource.json",
		[]string{"fdother_014_church/entry_001/frame.png", "fdother_014_church/entry_003/frame.png", "fdother_014_church/entry_000/mask.png", "fdother_014_church/entry_023/frame.png"}},
	{"load_chapter_aux", obj{}, [][2]string{{"surfaces/FDOTHER_055", "FDOTHER_055"}}, "FDOTHER_055/resource.json", []string{"FDOTHER_055/frame.png"}},
	{"load_sound", obj{"resource": 82}, [][2]string{{"sfx/FDOTHER_082", "FDOTHER_082"}}, "FDOTHER_082/resource.json", nil},
	{"load_lut_bank", obj{}, [][2]string{{"palette/fdother_003_luts.json", "fdother_003_luts.json"}}, "fdother_003_luts.json", nil},
	{"load_range_overlay", obj{}, [][2]string{{"sprites/fdother_001_range_overlay", ""}}, "bank.json",
		[]string{"sprite_0000/frame.png", "sprite_0000/mask.png", "sprite_0003/remap_mask.png"}},
	{"load_frame_bank", obj{"which": "event61"}, [][2]string{{"animations/fdother_045_event61", ""}}, "bank.json", []string{"frame_000/frame.png", "frame_000/mask.png"}},
	{"load_frame_bank", obj{"which": "pending_code1"}, [][2]string{{"animations/fdother_079_pending_code1", ""}}, "bank.json", nil},
	{"load_parallax", obj{}, [][2]string{{"surfaces/FDOTHER_016", "FDOTHER_016"}, {"surfaces/FDOTHER_017", "FDOTHER_017"}}, "FDOTHER_017/resource.json", nil},
}

func marshal(v any) []byte {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	must(enc.Encode(v))
	return b.Bytes()
}

// jsonMutations 列出對一份 metadata 的變異：每個純量欄位（頂層、巢狀物件、陣列首末元素）
// 改值、改成 null、刪除、改型別；陣列整體改成 null、空陣列、去尾、重複首項、交換前兩項。
func jsonMutations(raw []byte) []struct {
	name string
	doc  []byte
} {
	type mut = struct {
		name string
		doc  []byte
	}
	var out []mut
	parse := func() any {
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.UseNumber()
		var v any
		must(dec.Decode(&v))
		return v
	}
	scalar := func(path string, get func(root any) (map[string]any, []any, string, int)) {
		apply := func(tag string, f func(old any) (any, bool)) {
			root := parse()
			m, a, key, idx := get(root)
			if m != nil {
				nv, keep := f(m[key])
				if keep {
					m[key] = nv
				} else {
					delete(m, key)
				}
			} else {
				nv, _ := f(a[idx])
				a[idx] = nv
			}
			out = append(out, mut{path + ":" + tag, marshal(root)})
		}
		root := parse()
		m, a, key, idx := get(root)
		var old any
		if m != nil {
			old = m[key]
		} else {
			old = a[idx]
		}
		switch v := old.(type) {
		case json.Number:
			n, _ := v.Int64()
			apply("inc", func(any) (any, bool) { return json.Number(strconv.FormatInt(n+1, 10)), true })
			apply("zero", func(any) (any, bool) { return json.Number("0"), true })
			apply("neg", func(any) (any, bool) { return json.Number("-1"), true })
			apply("float", func(any) (any, bool) { return json.Number("1.5"), true })
			apply("string", func(any) (any, bool) { return "7", true })
		case string:
			apply("suffix", func(any) (any, bool) { return v + "x", true })
			apply("empty", func(any) (any, bool) { return "", true })
			apply("number", func(any) (any, bool) { return json.Number("7"), true })
		default:
			return
		}
		apply("null", func(any) (any, bool) { return nil, true })
		if m != nil {
			apply("delete", func(any) (any, bool) { return nil, false })
		}
	}
	type getter = func(root any) any
	type setter = func(root, v any)
	var walk func(path string, get getter, set setter)
	walk = func(path string, get getter, set setter) {
		node := get(parse())
		switch v := node.(type) {
		case map[string]any:
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				k := k
				child := func(root any) any { return get(root).(map[string]any)[k] }
				childSet := func(root, nv any) { get(root).(map[string]any)[k] = nv }
				switch v[k].(type) {
				case map[string]any, []any:
					walk(path+"."+k, child, childSet)
					for _, tag := range []string{"null", "delete", "scalar"} {
						root := parse()
						m := get(root).(map[string]any)
						switch tag {
						case "null":
							m[k] = nil
						case "delete":
							delete(m, k)
						default:
							m[k] = json.Number("3")
						}
						out = append(out, mut{path + "." + k + ":" + tag, marshal(root)})
					}
				default:
					scalar(path+"."+k, func(root any) (map[string]any, []any, string, int) {
						return get(root).(map[string]any), nil, k, 0
					})
				}
			}
		case []any:
			if len(v) == 0 {
				return
			}
			indices := []int{0}
			if len(v) > 1 {
				indices = append(indices, len(v)-1)
			}
			for _, i := range indices {
				i := i
				elem := func(root any) any { return get(root).([]any)[i] }
				elemSet := func(root, nv any) { get(root).([]any)[i] = nv }
				switch v[i].(type) {
				case map[string]any:
					walk(fmt.Sprintf("%s[%d]", path, i), elem, elemSet)
				default:
					scalar(fmt.Sprintf("%s[%d]", path, i), func(root any) (map[string]any, []any, string, int) {
						return nil, get(root).([]any), "", i
					})
				}
			}
			setArr := func(tag string, f func(a []any) []any) {
				root := parse()
				set(root, f(get(root).([]any)))
				out = append(out, mut{path + ":" + tag, marshal(root)})
			}
			setArr("empty", func([]any) []any { return []any{} })
			setArr("drop-last", func(a []any) []any { return a[:len(a)-1] })
			setArr("dup-first", func(a []any) []any { return append(a[:len(a)-1:len(a)-1], a[0]) })
			setArr("null-first", func(a []any) []any { a[0] = nil; return a })
			if len(v) > 1 {
				setArr("swap", func(a []any) []any { a[0], a[1] = a[1], a[0]; return a })
			}
		}
	}
	walk("$", func(root any) any { return root }, func(any, any) { panic("頂層不是陣列") })
	for _, t := range []struct{ name, doc string }{
		{"top-null", "null"}, {"top-array", "[]"}, {"top-empty", "{}"}, {"top-number", "7"},
		{"syntax-empty", ""}, {"syntax-garbage", "{not json"}, {"syntax-trailing", string(raw) + "x"},
	} {
		out = append(out, mut{t.name, []byte(t.doc)})
	}
	root := parse()
	if m, ok := root.(map[string]any); ok {
		m["unknown_extra_field"] = []any{1, "x", nil}
		out = append(out, mut{"unknown-field", marshal(root)})
	}
	return out
}

func (e *exporter) genMutations() {
	for _, spec := range loaderSpecs {
		kind := "tree-" + spec.op
		base := treeSpec{Links: spec.links}
		e.loadTree(kind+"-base", spec.op, base, spec.args)
		raw, err := os.ReadFile(filepath.Join(e.assets, realOf(spec, spec.json)))
		must(err)
		for _, m := range jsonMutations(raw) {
			t := base
			t.Files = [][2]string{{spec.json, e.blob(m.doc)}}
			e.loadTree(kind+"-json-"+m.name, spec.op, t, spec.args)
		}
		t := base
		t.Remove = []string{spec.json}
		e.loadTree(kind+"-fs-no-metadata", spec.op, t, spec.args)
		t = base
		t.Remove, t.Dirs = []string{spec.json}, []string{spec.json}
		e.loadTree(kind+"-fs-metadata-dir", spec.op, t, spec.args)
		for _, p := range spec.pngs {
			t = base
			t.Remove = []string{p}
			e.loadTree(kind+"-fs-no-png", spec.op, t, spec.args)
			t = base
			t.Remove, t.Dirs = []string{p}, []string{p}
			e.loadTree(kind+"-fs-png-dir", spec.op, t, spec.args)
			f, err := os.Open(filepath.Join(e.assets, realOf(spec, p)))
			must(err)
			cfg, _, err := image.DecodeConfig(f)
			f.Close()
			must(err)
			for _, v := range pngVariants(cfg.Width, cfg.Height) {
				t = base
				t.Files = [][2]string{{p, e.blob(v.data)}}
				e.malformed = v.malformed
				e.loadTree(kind+"-"+v.name, spec.op, t, spec.args)
				e.malformed = false
			}
		}
	}
	e.genTargeted()
	// 格式錯誤的 JSON／PNG：Go 標準函式庫寬鬆、Rust 拒收的已知類別（只比接受與否）。
	slots := loaderSpecs[8]
	raw, err := os.ReadFile(filepath.Join(e.assets, realOf(slots, slots.json)))
	must(err)
	text := string(raw)
	open := strings.Index(text, "{") + 1
	malformedJSON := []struct{ name, doc string }{
		{"json-fold", strings.Replace(text, `"kind"`, `"KIND"`, 1)},
		{"json-dup", text[:open] + `"schema_version": 2,` + text[open:]},
		{"json-utf8", text[:open] + "\"zz_unknown\": \"\xff\"," + text[open:]},
		{"json-neg-zero", strings.Replace(text, `"schema_version": 1`, `"schema_version": -0`, 1)},
	}
	e.malformed = true
	for _, m := range malformedJSON {
		t := treeSpec{Links: slots.links, Files: [][2]string{{slots.json, e.blob([]byte(m.doc))}}}
		e.loadTree("tree-load_load_slots-"+m.name, slots.op, t, slots.args)
	}
	data := buildPNGWithIDATTail(pngSpec{w: 310, h: 86, depth: 8, ctype: 3, plte: 256, sample: palSample(256)}, []byte("junk"))
	t := treeSpec{Links: slots.links, Files: [][2]string{{slots.pngs[0], e.blob(data)}}}
	e.loadTree("tree-load_load_slots-png-zlib-trailing", slots.op, t, slots.args)
	e.malformed = false
	// 音效：OGG 檔頭錯誤。
	spec := loaderSpecs[12]
	for _, c := range []struct{ name, data string }{{"ogg-bad-magic", "OggX1234"}, {"ogg-short", "Og"}, {"ogg-magic-only", "OggS"}} {
		t := treeSpec{Links: spec.links, Files: [][2]string{{"FDOTHER_082/sample_001.ogg", e.blob([]byte(c.data))}}}
		e.loadTree("tree-load_sound-"+c.name, "load_sound", t, spec.args)
	}
}

// realOf 把樹內路徑換回真實素材包路徑。
func realOf(spec loaderSpec, treeRel string) string {
	for _, l := range spec.links {
		if l[1] == "" {
			return filepath.Join(l[0], treeRel)
		}
		if treeRel == l[1] || strings.HasPrefix(treeRel, l[1]+"/") {
			return filepath.Join(l[0], strings.TrimPrefix(strings.TrimPrefix(treeRel, l[1]), "/"))
		}
	}
	panic(treeRel)
}

func palPNG(w, h int) []byte {
	return buildPNG(pngSpec{w: w, h: h, depth: 8, ctype: 3, plte: 256, sample: palSample(256)})
}

func maskPNG(w, h int) []byte {
	return buildPNG(pngSpec{w: w, h: h, depth: 8, ctype: 0, sample: maskSample})
}

// entryEdit 改寫 metadata 陣列中 index 相符的項目，並可另外放入檔案（例如尺寸相符的 PNG）。
func (e *exporter) entryEdit(name string, spec loaderSpec, arrayKey string, index int, set obj, files map[string][]byte) {
	raw, err := os.ReadFile(filepath.Join(e.assets, realOf(spec, spec.json)))
	must(err)
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var doc map[string]any
	must(dec.Decode(&doc))
	found := false
	for _, item := range doc[arrayKey].([]any) {
		m := item.(map[string]any)
		if n, _ := m["index"].(json.Number).Int64(); int(n) == index {
			for k, v := range set {
				m[k] = v
			}
			found = true
		}
	}
	if !found {
		panic(name)
	}
	t := treeSpec{Links: spec.links, Files: [][2]string{{spec.json, e.blob(marshal(doc))}}}
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		t.Files = append(t.Files, [2]string{k, e.blob(files[k])})
	}
	e.loadTree("tree-"+spec.op+"-target-"+name, spec.op, t, spec.args)
}

// genTargeted 補上泛用變異碰不到、但 Go 版確實可達的契約分支。
func (e *exporter) genTargeted() {
	item, sysinfo, lmi9, church := loaderSpecs[5], loaderSpecs[6], loaderSpecs[7], loaderSpecs[10]
	e.entryEdit("opaque-mask", item, "entries", 20, obj{"mask": "entry_020/mask.png"}, nil)
	e.entryEdit("raw-mask", item, "entries", 0, obj{"mask": "x"}, nil)
	e.entryEdit("frame-no-mask", item, "entries", 31, obj{"mask": ""}, nil)
	e.entryEdit("frame-resized", item, "entries", 31, obj{"width": 9, "height": 9},
		map[string][]byte{"fdother_005_item_panel/entry_031/frame.png": palPNG(9, 9), "fdother_005_item_panel/entry_031/mask.png": maskPNG(9, 9)})
	e.entryEdit("panel-0x85-resized", sysinfo, "entries", 0x85, obj{"width": 103},
		map[string][]byte{"fdother_005_item_panel/entry_133/frame.png": palPNG(103, 17), "fdother_005_item_panel/entry_133/mask.png": maskPNG(103, 17)})
	e.entryEdit("panel-0x88-resized", sysinfo, "entries", 0x88, obj{"height": 14},
		map[string][]byte{"fdother_005_item_panel/entry_136/frame.png": palPNG(63, 14), "fdother_005_item_panel/entry_136/mask.png": maskPNG(63, 14)})
	e.entryEdit("spawn-geometry", lmi9, "entries", 0, obj{"width": 43}, map[string][]byte{"entry_000/frame.png": palPNG(43, 24)})
	e.entryEdit("spawn-geometry-last", lmi9, "entries", 11, obj{"height": 25}, map[string][]byte{"entry_011/frame.png": palPNG(66, 25)})
	e.entryEdit("opaque-mask", church, "entries", 1, obj{"mask": "x"}, nil)
	e.entryEdit("service-width", church, "entries", 3, obj{"width": 25}, map[string][]byte{"fdother_014_church/entry_003/frame.png": palPNG(25, 20)})
	e.entryEdit("service-height", church, "entries", 10, obj{"height": 21}, map[string][]byte{"fdother_014_church/entry_010/frame.png": palPNG(24, 21)})
	e.entryEdit("price-mask", church, "entries", 15, obj{"mask": "x"}, nil)
	e.entryEdit("frame-no-mask", church, "entries", 23, obj{"mask": "entry_023/frame.png"}, nil)
	e.entryEdit("background-321", church, "entries", 0, obj{"width": 321},
		map[string][]byte{"fdother_014_church/entry_000/frame.png": palPNG(321, 200), "fdother_014_church/entry_000/mask.png": maskPNG(321, 200)})
	// 範圍覆蓋層：描述子 7 的圖格像素長度錯誤時，fdicon 的錯誤被包進 descriptor 前綴。
	for _, c := range [][3]int{{4, 3, 3}, {5, 4, 4}, {4, 0, 0}, {3, 5, 5}} {
		dst := pattern(0x1c8*240, 1)
		res, err := protectErr(func() (obj, error) {
			err := fdother.BlitNativeRangeOverlay(e.bankObjs["range-short-pixels"], dst, 10, 20, 13, 8, c[0], 10+c[1], 20+c[2])
			return obj{"dst": digest(dst)}, err
		})
		e.add("range-blit-target", "range_blit", obj{"bank": "range-short-pixels", "dst_len": 0x1c8 * 240, "seed": 1,
			"camera_x": 10, "camera_y": 20, "visible_width": 13, "visible_height": 8, "mode": c[0], "cx": 10 + c[1], "cy": 20 + c[2]}, res, err)
	}
}
