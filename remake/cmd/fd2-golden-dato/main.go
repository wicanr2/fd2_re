// fd2-golden-dato 輸出 internal/dato 的黃金差分資料，供 Rust 改寫逐欄比對
// （docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 dato 的公開 API，輸入另用 fdother.ReadResource 取原始
// 資源位元組，不改任何遊戲行為。隨機輸入以固定種子的 PCG 產生，輸入與結果寫進同一份 JSON；
// 真實原版檔與分離頭像只記路徑與 SHA-256，Rust 端自行讀同一批檔案。
package main

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// errJSON 記錄完整錯誤文字；若錯誤以 %w 包住內層，另記去掉內層文字後的前綴，
// 讓 Rust 端對 I/O 與 zlib 串流這類實作相依的內層只比前綴。
func errJSON(err error) map[string]any {
	if err == nil {
		return map[string]any{"err": nil}
	}
	out := map[string]any{"err": err.Error()}
	if inner := errors.Unwrap(err); inner != nil {
		out["err_prefix"] = strings.TrimSuffix(err.Error(), inner.Error())
	}
	return out
}

func framesJSON(frames []dato.Frame) []map[string]any {
	out := make([]map[string]any, len(frames))
	for i, f := range frames {
		row := map[string]any{"w": f.Width, "h": f.Height, "len": len(f.Pixels), "sha256": digest(f.Pixels)}
		if len(f.Pixels) <= 256 {
			row["hex"] = hex.EncodeToString(f.Pixels)
		}
		out[i] = row
	}
	return out
}

func framesResult(frames []dato.Frame, err error) map[string]any {
	out := errJSON(err)
	out["nil"] = frames == nil
	if err == nil {
		out["frames"] = framesJSON(frames)
	}
	return out
}

// ---- ParseResource ----

func putU32(b []byte, v int) { binary.LittleEndian.PutUint32(b, uint32(v)) }

// rleStream 產生一幀的 0x4e916 游程串流：字面像素 0..0xc0 或 (0xc1..0xff, 值)。
func rleStream(rng *rand.Rand, want int) []byte {
	var out []byte
	for n := 0; n < want; {
		if rng.IntN(3) == 0 {
			run := 1 + rng.IntN(63)
			out = append(out, byte(0xc0+run), byte(rng.IntN(256)))
			n += run
		} else {
			out = append(out, byte(rng.IntN(0xc1)))
			n++
		}
	}
	return out
}

func shapedResource(rng *rand.Rand, kind string) []byte {
	var frames [][]byte
	for i := 0; i < 4; i++ {
		w, h := 1+rng.IntN(12), 1+rng.IntN(12)
		switch rng.IntN(12) {
		case 0:
			w = 0
		case 1:
			h = 257
		case 2:
			w, h = 256, 1
		}
		head := make([]byte, 4)
		binary.LittleEndian.PutUint16(head, uint16(w))
		binary.LittleEndian.PutUint16(head[2:], uint16(h))
		body := rleStream(rng, w*h)
		if kind == "shaped-trailing" {
			body = append(body, byte(rng.IntN(256)), byte(rng.IntN(256)))
		}
		frames = append(frames, append(head, body...))
	}
	data := make([]byte, 16)
	at := 16
	for i, f := range frames {
		putU32(data[i*4:], at)
		at += len(f)
	}
	for _, f := range frames {
		data = append(data, f...)
	}
	switch kind {
	case "shaped-truncated":
		data = data[:16+rng.IntN(len(data)-15)]
	case "shaped-flip":
		data[rng.IntN(len(data))] ^= byte(1 << rng.IntN(8))
	case "shaped-offsets":
		i := rng.IntN(4)
		switch rng.IntN(4) {
		case 0:
			putU32(data[i*4:], rng.IntN(20))
		case 1:
			putU32(data[i*4:], len(data)+rng.IntN(3))
		case 2:
			if i > 0 {
				copy(data[i*4:i*4+4], data[(i-1)*4:i*4])
			}
		default:
			putU32(data[i*4:], int(binary.LittleEndian.Uint32(data[i*4:]))+1+rng.IntN(4))
		}
	}
	return data
}

func parseCases(rng *rand.Rand, count int) []map[string]any {
	var rows []map[string]any
	add := func(kind string, data []byte) {
		frames, err := dato.ParseResource(data)
		rows = append(rows, map[string]any{
			"index": len(rows), "kind": kind, "data_hex": hex.EncodeToString(data), "result": framesResult(frames, err),
		})
	}
	// 結構化邊界。
	add("structured-empty", nil)
	add("structured-15", make([]byte, 15))
	add("structured-16", make([]byte, 16))
	same := make([]byte, 24)
	for i := 0; i < 4; i++ {
		putU32(same[i*4:], 16)
	}
	add("structured-equal-offsets", same)
	lit := []byte{}
	for i := 0; i < 4; i++ {
		lit = append(lit, 1, 0, 1, 0, 0xc0)
	}
	lit16 := make([]byte, 16)
	for i := 0; i < 4; i++ {
		putU32(lit16[i*4:], 16+5*i)
	}
	add("structured-literal-c0", append(lit16, lit...))
	clamp := append([]byte(nil), lit16...)
	for i := 0; i < 4; i++ {
		clamp = append(clamp, 1, 0, 1, 0, 0xff)
	}
	add("structured-repeat-lacks-pixel", clamp)
	clamp2 := append([]byte(nil), lit16...)
	for i := 0; i < 4; i++ {
		clamp2 = append(clamp2, 2, 0, 1, 0, 0xff)
	}
	// 每幀 2 像素但只有 0xff 一個碼：最後一幀缺值，前幾幀的值是下一幀的標頭位元組。
	add("structured-run-clamped", clamp2)
	short := append([]byte(nil), lit16...)
	for i := 0; i < 4; i++ {
		short = append(short, 3, 0, 1, 0, 7)
	}
	add("structured-stream-ends", short)
	kinds := []string{"random", "shaped", "shaped-trailing", "shaped-truncated", "shaped-flip", "shaped-offsets"}
	for i := 0; i < count; i++ {
		kind := kinds[i%len(kinds)]
		if kind == "random" {
			data := make([]byte, rng.IntN(64))
			for j := range data {
				data[j] = byte(rng.Uint32())
			}
			if len(data) >= 16 && rng.IntN(2) == 0 {
				for j := 0; j < 4; j++ {
					putU32(data[j*4:], 16+rng.IntN(len(data)-15))
				}
			}
			add(kind, data)
		} else {
			add(kind, shapedResource(rng, kind))
		}
	}
	return rows
}

// ---- Frame 繪製 ----

type blitInput struct {
	Op     string `json:"op"`
	W      int    `json:"w"`
	H      int    `json:"h"`
	PixHex string `json:"pixels_hex"`
	DstHex string `json:"dst_hex"`
	Stride int    `json:"stride"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Offset int    `json:"offset"`
}

func runBlit(in blitInput) (out map[string]any) {
	pix, _ := hex.DecodeString(in.PixHex)
	dst, _ := hex.DecodeString(in.DstHex)
	f := dato.Frame{Width: in.W, Height: in.H, Pixels: pix}
	defer func() {
		if r := recover(); r != nil {
			out = map[string]any{"err": nil, "panic": true, "dst_hex": hex.EncodeToString(dst)}
		}
	}()
	var err error
	switch in.Op {
	case "at":
		err = f.BlitAt(dst, in.Stride, in.X, in.Y)
	case "offset":
		err = f.BlitAtOffset(dst, in.Stride, in.Offset)
	default:
		err = f.BlitRightToLeftAtOffset(dst, in.Stride, in.Offset)
	}
	out = errJSON(err)
	out["panic"] = false
	out["dst_hex"] = hex.EncodeToString(dst)
	return out
}

func blitCases(rng *rand.Rand, count int) []map[string]any {
	var rows []map[string]any
	add := func(kind string, in blitInput) {
		rows = append(rows, map[string]any{"index": len(rows), "kind": kind, "input": in, "result": runBlit(in)})
	}
	fill := func(n int) string {
		b := make([]byte, n)
		for i := range b {
			b[i] = byte(0xe0 + i%16)
		}
		return hex.EncodeToString(b)
	}
	seq := func(n int) string {
		b := make([]byte, max(n, 0))
		for i := range b {
			b[i] = byte(i + 1)
		}
		return hex.EncodeToString(b)
	}
	// 結構化：整數溢位使檢查通過後 panic、除以零、負 stride 與 4*2^62 環繞成 0。
	big := math.MaxInt64
	add("structured-x-overflow", blitInput{Op: "at", W: 1, H: 1, PixHex: seq(1), DstHex: fill(16), Stride: 4, X: big})
	add("structured-stride-zero", blitInput{Op: "at", W: 1, H: 1, PixHex: seq(1), DstHex: fill(16), Stride: 0, X: big})
	add("structured-stride-negative", blitInput{Op: "at", W: 1, H: 1, PixHex: seq(1), DstHex: fill(16), Stride: -3, X: big})
	add("structured-offset-width-overflow", blitInput{Op: "offset", W: big, H: 1, PixHex: seq(1), DstHex: fill(16), Stride: 4, Offset: 5})
	add("structured-rtl-wrapped-area", blitInput{Op: "rtl", W: 4, H: 1 << 62, PixHex: "", DstHex: fill(16), Stride: 4, Offset: 3})
	add("structured-at-exact-fit", blitInput{Op: "at", W: 2, H: 2, PixHex: seq(4), DstHex: fill(8), Stride: 4, X: 2, Y: 0})
	add("structured-at-y-at-end", blitInput{Op: "at", W: 1, H: 1, PixHex: seq(1), DstHex: fill(8), Stride: 4, X: 0, Y: 2})
	add("structured-rtl-exact-fit", blitInput{Op: "rtl", W: 3, H: 2, PixHex: seq(6), DstHex: fill(8), Stride: 4, Offset: 2})
	ops := []string{"at", "offset", "rtl"}
	for i := 0; i < count; i++ {
		w, h := rng.IntN(7)-1, rng.IntN(7)-1
		n := max(w, 0) * max(h, 0)
		if rng.IntN(8) == 0 {
			n += rng.IntN(3) - 1
		}
		in := blitInput{
			Op: ops[i%len(ops)], W: w, H: h, PixHex: seq(n), DstHex: fill(rng.IntN(80)),
			Stride: rng.IntN(22) - 2, X: rng.IntN(22) - 2, Y: rng.IntN(22) - 2, Offset: rng.IntN(125) - 5,
		}
		add("random", in)
	}
	return rows
}

// ---- 嘴型狀態機 ----

func mouthCases(rng *rand.Rand, count int) []map[string]any {
	var rows []map[string]any
	stepMouth := func(s dato.MouthState, sample int) (dato.MouthState, map[string]any) {
		next, err := s.Tick(sample)
		out := errJSON(err)
		out["open"], out["countdown"], out["frame"] = next.Open, next.Countdown, next.FrameIndex()
		return next, out
	}
	stepConf := func(s dato.ConfirmationMouthState, sample int) (dato.ConfirmationMouthState, map[string]any) {
		next, err := s.Tick(sample)
		out := errJSON(err)
		out["open"], out["countdown"], out["frame"] = next.Open, next.Countdown, next.FrameIndex()
		return next, out
	}
	addMouth := func(kind string, start dato.MouthState, samples []int) {
		var steps []map[string]any
		s := start
		for _, sample := range samples {
			var step map[string]any
			s, step = stepMouth(s, sample)
			steps = append(steps, step)
		}
		rows = append(rows, map[string]any{"index": len(rows), "kind": kind, "machine": "mouth",
			"start": map[string]any{"open": start.Open, "countdown": start.Countdown}, "samples": samples, "steps": steps})
	}
	addConf := func(kind string, start *dato.ConfirmationMouthState, newSample int, samples []int) {
		var steps []map[string]any
		var s dato.ConfirmationMouthState
		row := map[string]any{"index": len(rows), "kind": kind, "machine": "confirmation", "samples": samples}
		if start == nil {
			var err error
			s, err = dato.NewConfirmationMouthState(newSample)
			created := errJSON(err)
			created["open"], created["countdown"], created["frame"] = s.Open, s.Countdown, s.FrameIndex()
			row["new_sample"], row["new"] = newSample, created
		} else {
			s = *start
			row["start"] = map[string]any{"open": start.Open, "countdown": start.Countdown}
		}
		for _, sample := range samples {
			var step map[string]any
			s, step = stepConf(s, sample)
			steps = append(steps, step)
		}
		row["steps"] = steps
		rows = append(rows, row)
	}
	// 結構化：確認嘴型從每個初始值跑完整一輪。
	for sample := -1; sample <= 30; sample++ {
		cycle := make([]int, sample+4)
		cycle[len(cycle)-1] = (sample + 7) % 30
		addConf("structured-cycle", nil, sample, cycle)
	}
	for _, c := range []int{-2, -1, 0, 39, 40, 1000, math.MinInt64} {
		for _, open := range []bool{false, true} {
			st := dato.ConfirmationMouthState{Open: open, Countdown: c}
			addConf("structured-state", &st, 0, []int{0, 5, -1, 30})
			addMouth("structured-state", dato.MouthState{Open: open, Countdown: c}, []int{0, 29, 30, -1, 3})
		}
	}
	for i := 0; i < count; i++ {
		samples := make([]int, 12)
		for j := range samples {
			samples[j] = rng.IntN(35) - 2
		}
		open, countdown := rng.IntN(2) == 0, rng.IntN(49)-3
		if i%2 == 0 {
			addMouth("random", dato.MouthState{Open: open, Countdown: countdown}, samples)
		} else if i%4 == 1 {
			st := dato.ConfirmationMouthState{Open: open, Countdown: countdown}
			addConf("random", &st, 0, samples)
		} else {
			addConf("random-new", nil, rng.IntN(35)-2, samples)
		}
	}
	return rows
}

// ---- 分離頭像 PNG ----

type pngChunk struct {
	typ    string
	data   []byte
	badCRC bool
}

func buildPNG(chunks []pngChunk) []byte {
	out := []byte("\x89PNG\r\n\x1a\n")
	for _, c := range chunks {
		var l [4]byte
		binary.BigEndian.PutUint32(l[:], uint32(len(c.data)))
		out = append(out, l[:]...)
		body := append([]byte(c.typ), c.data...)
		out = append(out, body...)
		sum := crc32.ChecksumIEEE(body)
		if c.badCRC {
			sum ^= 1
		}
		binary.BigEndian.PutUint32(l[:], sum)
		out = append(out, l[:]...)
	}
	return out
}

func ihdr(w, h uint32, depth, ct, comp, filter, interlace byte) []byte {
	b := make([]byte, 13)
	binary.BigEndian.PutUint32(b, w)
	binary.BigEndian.PutUint32(b[4:], h)
	b[8], b[9], b[10], b[11], b[12] = depth, ct, comp, filter, interlace
	return b
}

func zlibBytes(raw []byte, level int) []byte {
	var buf bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&buf, level)
	zw.Write(raw)
	zw.Close()
	return buf.Bytes()
}

// packRow 把一列索引依位元深度打包（高位在前）。
func packRow(idx []byte, depth int) []byte {
	out := make([]byte, (len(idx)*depth+7)/8)
	for x, v := range idx {
		bit := x * depth
		out[bit/8] |= (v & byte(1<<depth-1)) << (8 - depth - bit%8)
	}
	return out
}

var adam7 = [7][4]int{{8, 8, 0, 0}, {8, 8, 4, 0}, {4, 8, 0, 4}, {4, 4, 2, 0}, {2, 4, 0, 2}, {2, 2, 1, 0}, {1, 2, 0, 1}}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// rawRows 產生 IDAT 解壓前的位元組：每列一個濾波位元組加打包後的索引；可選 Adam7。
// filters 依列循環套用 PNG 的五種濾波（Go 的 png 編碼器對 indexed 影像不加濾波，
// 這裡自己做，才能讓 Rust 端的濾波還原在 indexed 像素上被逐位元組比對）。
func rawRows(w, h int, idx []byte, depth int, interlace bool, filters []byte) []byte {
	var out []byte
	emit := func(rows [][]byte) {
		var prev []byte
		for y, row := range rows {
			packed := packRow(row, depth)
			f := byte(0)
			if len(filters) > 0 {
				f = filters[y%len(filters)]
			}
			bpp := max(depth/8, 1)
			enc := make([]byte, len(packed))
			for i := range packed {
				var left, up, upLeft int
				if i >= bpp {
					left = int(packed[i-bpp])
				}
				if prev != nil {
					up = int(prev[i])
					if i >= bpp {
						upLeft = int(prev[i-bpp])
					}
				}
				var pred int
				switch f {
				case 1:
					pred = left
				case 2:
					pred = up
				case 3:
					pred = (left + up) / 2
				case 4:
					p := left + up - upLeft
					pa, pb, pc := abs(p-left), abs(p-up), abs(p-upLeft)
					switch {
					case pa <= pb && pa <= pc:
						pred = left
					case pb <= pc:
						pred = up
					default:
						pred = upLeft
					}
				}
				enc[i] = packed[i] - byte(pred)
			}
			out = append(out, f)
			out = append(out, enc...)
			prev = packed
		}
	}
	if !interlace {
		var rows [][]byte
		for y := 0; y < h; y++ {
			rows = append(rows, idx[y*w:(y+1)*w])
		}
		emit(rows)
		return out
	}
	for _, p := range adam7 {
		pw := (w - p[2] + p[0] - 1) / p[0]
		ph := (h - p[3] + p[1] - 1) / p[1]
		if pw <= 0 || ph <= 0 {
			continue
		}
		var rows [][]byte
		for y := 0; y < ph; y++ {
			row := make([]byte, pw)
			for x := 0; x < pw; x++ {
				row[x] = idx[(y*p[1]+p[3])*w+x*p[0]+p[2]]
			}
			rows = append(rows, row)
		}
		emit(rows)
	}
	return out
}

func palette(n int) []byte {
	out := make([]byte, 3*n)
	for i := range out {
		out[i] = byte(i * 37)
	}
	return out
}

// indexedPNG 用自製區塊組出 indexed PNG；回傳完整檔與 zlib 串流供變形。
func indexedPNG(w, h, depth, plte int, idx []byte, interlace bool, filters []byte, level int) ([]byte, []byte) {
	il := byte(0)
	if interlace {
		il = 1
	}
	z := zlibBytes(rawRows(w, h, idx, depth, interlace, filters), level)
	return buildPNG([]pngChunk{
		{typ: "IHDR", data: ihdr(uint32(w), uint32(h), byte(depth), 3, 0, 0, il)},
		{typ: "PLTE", data: palette(plte)},
		{typ: "IDAT", data: z},
		{typ: "IEND"},
	}), z
}

func encodeImage(img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func randomIndices(rng *rand.Rand, n, limit int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(rng.IntN(limit))
	}
	return out
}

type sepFile struct {
	Name string `json:"name"`
	Hex  string `json:"hex,omitempty"`
	Dir  bool   `json:"dir,omitempty"`
}

// 案例命名決定 Rust 端的比對規則（tests/golden_dato.rs）：input-／ok-／open-／type-／nonpng-
// 與 random-clean、stream-clean 是合法輸入，逐欄比對；其餘（lenient-、crc-、garbage- 等）
// 是刻意畸形的 PNG，只比接受或拒收。
//
// separatedCase 在暫存目錄寫出檔案後呼叫 LoadSeparatedResource。root 是相對暫存目錄的
// 寫法（可含 `..`）；空字串代表直接傳空根目錄。
func separatedCase(tmp string, index int, kind string, files []sepFile, root string, resource int) map[string]any {
	dir := filepath.Join(tmp, fmt.Sprintf("case%04d", index))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	for _, f := range files {
		path := filepath.Join(dir, f.Name)
		if f.Dir {
			if err := os.MkdirAll(path, 0o755); err != nil {
				panic(err)
			}
			continue
		}
		data, _ := hex.DecodeString(f.Hex)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			panic(err)
		}
	}
	callRoot := ""
	if root != "" {
		callRoot = dir + "/" + root
	}
	frames, err := dato.LoadSeparatedResource(callRoot, resource)
	return map[string]any{
		"index": index, "kind": kind, "files": files, "root": root, "resource": resource,
		"result": framesResult(frames, err),
	}
}

func fourSame(resource int, data []byte) []sepFile {
	var files []sepFile
	for m := 0; m < 4; m++ {
		files = append(files, sepFile{Name: fmt.Sprintf("DATO_%03d_m%d.png", resource, m), Hex: hex.EncodeToString(data)})
	}
	return files
}

func withMouth(files []sepFile, mouth int, data []byte) []sepFile {
	out := append([]sepFile(nil), files...)
	out[mouth].Hex = hex.EncodeToString(data)
	return out
}

func separatedCases(rng *rand.Rand, tmp string, count int) []map[string]any {
	var rows []map[string]any
	add := func(kind string, files []sepFile, root string, resource int) {
		rows = append(rows, separatedCase(tmp, len(rows), kind, files, root, resource))
	}
	good, goodZ := indexedPNG(5, 3, 8, 16, randomIndices(rng, 15, 16), false, nil, zlib.DefaultCompression)
	base := fourSame(7, good)

	// 呼叫參數與檔案系統。
	add("input-empty-root", base, "", 7)
	add("input-negative", base, ".", -1)
	add("ok-baseline", base, ".", 7)
	add("ok-root-dotdot", base, "missing/../.", 7)
	add("ok-four-digit", fourSame(1000, good), ".", 1000)
	add("open-missing-m2", base[:2], ".", 7)
	add("open-root-is-file", []sepFile{{Name: "plain", Hex: "00"}}, "plain", 7)
	add("nonpng-directory", append(base[:3:3], sepFile{Name: "DATO_007_m3.png", Dir: true}), ".", 7)
	add("nonpng-empty-file", withMouth(base, 1, nil), ".", 7)
	add("nonpng-seven-bytes", withMouth(base, 1, []byte("\x89PNG\r\n\x1a")), ".", 7)
	add("nonpng-gif", withMouth(base, 1, []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;")), ".", 7)
	add("trunc-signature-only", withMouth(base, 1, []byte("\x89PNG\r\n\x1a\n")), ".", 7)

	// Go 編碼器產生的各種色彩型態。
	gray := image.NewGray(image.Rect(0, 0, 3, 2))
	add("type-gray", withMouth(base, 0, encodeImage(gray)), ".", 7)
	add("type-gray16", withMouth(base, 0, encodeImage(image.NewGray16(image.Rect(0, 0, 3, 2)))), ".", 7)
	add("type-rgba", withMouth(base, 0, encodeImage(image.NewRGBA(image.Rect(0, 0, 3, 2)))), ".", 7)
	add("type-nrgba", withMouth(base, 0, encodeImage(image.NewNRGBA(image.Rect(0, 0, 3, 2)))), ".", 7)
	add("type-rgba64", withMouth(base, 0, encodeImage(image.NewRGBA64(image.Rect(0, 0, 3, 2)))), ".", 7)
	for _, n := range []int{1, 2, 3, 4, 5, 16, 17, 256} {
		pal := make(color.Palette, n)
		for i := range pal {
			pal[i] = color.NRGBA{byte(i), byte(i * 3), byte(i * 7), byte(255 - i%2*128)}
		}
		img := image.NewPaletted(image.Rect(0, 0, 7, 3), pal)
		for i := range img.Pix {
			img.Pix[i] = byte(rng.IntN(n))
		}
		add(fmt.Sprintf("type-paletted-%d", n), withMouth(base, 2, encodeImage(img)), ".", 7)
	}

	// 自製區塊：位元深度、交錯、濾波、調色盤與 tRNS。
	for _, depth := range []int{1, 2, 4, 8} {
		for _, il := range []bool{false, true} {
			for _, dims := range [][2]int{{1, 1}, {3, 5}, {9, 9}} {
				w, h := dims[0], dims[1]
				data, _ := indexedPNG(w, h, depth, 1<<depth, randomIndices(rng, w*h, 1<<depth), il, []byte{0, 1, 2, 3, 4}, 6)
				add(fmt.Sprintf("ok-depth%d-interlace%v-%dx%d", depth, il, w, h), fourSame(8, data), ".", 8)
			}
		}
	}
	idx := randomIndices(rng, 15, 16)
	raw := rawRows(5, 3, idx, 8, false, nil)
	z := zlibBytes(raw, 6)
	hdr := pngChunk{typ: "IHDR", data: ihdr(5, 3, 8, 3, 0, 0, 0)}
	plte := pngChunk{typ: "PLTE", data: palette(16)}
	iend := pngChunk{typ: "IEND"}
	idat := pngChunk{typ: "IDAT", data: z}
	one := func(kind string, chunks []pngChunk) {
		add(kind, withMouth(base, 3, buildPNG(chunks)), ".", 7)
	}
	one("lenient-index-beyond-plte", []pngChunk{hdr, {typ: "PLTE", data: palette(3)}, idat, iend})
	one("ok-trns", []pngChunk{hdr, plte, {typ: "tRNS", data: []byte{0, 128}}, idat, iend})
	one("lenient-unknown-before-ihdr", []pngChunk{{typ: "abcd", data: []byte{1}}, hdr, plte, idat, iend})
	one("ok-ancillary", []pngChunk{hdr, {typ: "tEXt", data: []byte("k\x00v")}, plte, idat, {typ: "tIME", data: []byte{0x07, 0xcb, 7, 12, 0, 0, 0}}, iend})
	one("lenient-after-iend", append(append([]pngChunk{}, hdr, plte, idat, iend), pngChunk{typ: "junk", data: []byte{1}}))
	one("lenient-trailing-idat", []pngChunk{hdr, plte, idat, {typ: "IDAT", data: []byte("garbage")}, iend})
	one("crc-trailing-idat", []pngChunk{hdr, plte, idat, {typ: "IDAT", data: []byte("garbage"), badCRC: true}, iend})
	one("ok-split-1", []pngChunk{hdr, plte, {typ: "IDAT", data: z[:1]}, {typ: "IDAT", data: z[1:]}, iend})
	one("ok-split-last", []pngChunk{hdr, plte, {typ: "IDAT", data: z[:len(z)-1]}, {typ: "IDAT", data: z[len(z)-1:]}, iend})
	one("ok-zero-idat-first", []pngChunk{hdr, plte, {typ: "IDAT"}, idat, iend})
	one("ok-stored", []pngChunk{hdr, plte, {typ: "IDAT", data: zlibBytes(raw, zlib.NoCompression)}, iend})
	one("split-text-between", []pngChunk{hdr, plte, {typ: "IDAT", data: z[:3]}, {typ: "tEXt", data: []byte("a\x00b")}, {typ: "IDAT", data: z[3:]}, iend})
	one("lenient-text-after-complete", []pngChunk{hdr, plte, idat, {typ: "tEXt", data: []byte("a\x00b")}, {typ: "IDAT", data: []byte{9}}, iend})
	one("crc-idat-first-of-split", []pngChunk{hdr, plte, {typ: "IDAT", data: z[:3], badCRC: true}, {typ: "IDAT", data: z[3:]}, iend})
	one("crc-ihdr", []pngChunk{{typ: "IHDR", data: hdr.data, badCRC: true}, plte, idat, iend})
	one("crc-plte", []pngChunk{hdr, {typ: "PLTE", data: palette(16), badCRC: true}, idat, iend})
	one("crc-idat", []pngChunk{hdr, plte, {typ: "IDAT", data: z, badCRC: true}, iend})
	one("crc-iend", []pngChunk{hdr, plte, idat, {typ: "IEND", badCRC: true}})
	one("crc-unknown", []pngChunk{hdr, {typ: "tEXt", data: []byte("a"), badCRC: true}, plte, idat, iend})
	one("missing-iend", []pngChunk{hdr, plte, idat})
	one("iend-length", []pngChunk{hdr, plte, idat, {typ: "IEND", data: []byte{0}}})
	one("missing-plte", []pngChunk{hdr, idat, iend})
	one("trns-before-plte", []pngChunk{hdr, {typ: "tRNS", data: []byte{0}}, plte, idat, iend})
	one("trns-too-long", []pngChunk{hdr, plte, {typ: "tRNS", data: make([]byte, 257)}, idat, iend})
	one("plte-twice", []pngChunk{hdr, plte, plte, idat, iend})
	one("ihdr-twice", []pngChunk{hdr, hdr, plte, idat, iend})
	one("idat-before-ihdr", []pngChunk{idat, hdr, plte, iend})
	one("iend-before-idat", []pngChunk{hdr, plte, iend})
	one("plte-length-0", []pngChunk{hdr, {typ: "PLTE"}, idat, iend})
	one("plte-length-4", []pngChunk{hdr, {typ: "PLTE", data: make([]byte, 4)}, idat, iend})
	one("plte-257", []pngChunk{hdr, {typ: "PLTE", data: palette(257)}, idat, iend})
	one("plte-over-depth", []pngChunk{{typ: "IHDR", data: ihdr(5, 3, 1, 3, 0, 0, 0)}, {typ: "PLTE", data: palette(3)}, {typ: "IDAT", data: zlibBytes(rawRows(5, 3, make([]byte, 15), 1, false, nil), 6)}, iend})
	one("ihdr-length", []pngChunk{{typ: "IHDR", data: hdr.data[:12]}, plte, idat, iend})
	one("ihdr-depth16-paletted", []pngChunk{{typ: "IHDR", data: ihdr(5, 3, 16, 3, 0, 0, 0)}, plte, idat, iend})
	one("ihdr-colortype5", []pngChunk{{typ: "IHDR", data: ihdr(5, 3, 8, 5, 0, 0, 0)}, plte, idat, iend})
	one("ihdr-compression", []pngChunk{{typ: "IHDR", data: ihdr(5, 3, 8, 3, 1, 0, 0)}, plte, idat, iend})
	one("ihdr-filter", []pngChunk{{typ: "IHDR", data: ihdr(5, 3, 8, 3, 0, 1, 0)}, plte, idat, iend})
	one("ihdr-interlace2", []pngChunk{{typ: "IHDR", data: ihdr(5, 3, 8, 3, 0, 0, 2)}, plte, idat, iend})
	one("ihdr-width0", []pngChunk{{typ: "IHDR", data: ihdr(0, 3, 8, 3, 0, 0, 0)}, plte, idat, iend})
	one("ihdr-width-negative", []pngChunk{{typ: "IHDR", data: ihdr(1<<31, 3, 8, 3, 0, 0, 0)}, plte, idat, iend})
	one("ihdr-dimension-overflow", []pngChunk{{typ: "IHDR", data: ihdr(1<<31-1, 1<<31-1, 8, 3, 0, 0, 0)}, plte, idat, iend})
	one("gray-with-plte", []pngChunk{{typ: "IHDR", data: ihdr(5, 3, 8, 0, 0, 0, 0)}, plte, idat, iend})
	one("rgb-with-plte", []pngChunk{{typ: "IHDR", data: ihdr(5, 1, 8, 2, 0, 0, 0)}, plte, {typ: "IDAT", data: zlibBytes(make([]byte, 16), 6)}, iend})
	one("trns-gray-alpha", []pngChunk{{typ: "IHDR", data: ihdr(5, 3, 8, 4, 0, 0, 0)}, {typ: "tRNS", data: []byte{0, 0}}, idat, iend})
	one("lenient-empty-unknown", []pngChunk{hdr, plte, idat, {typ: "zzzz", data: nil}, iend})
	// 把 zzzz 的長度欄改成 0x80000000。
	{
		data := buildPNG([]pngChunk{hdr, plte, idat, {typ: "zzzz"}, iend})
		at := bytes.Index(data, []byte("zzzz")) - 4
		binary.BigEndian.PutUint32(data[at:], 0x80000000)
		add("bad-chunk-length-huge", withMouth(base, 3, data), ".", 7)
	}
	bad := append([]byte(nil), raw...)
	bad[0] = 5
	one("filter-5", []pngChunk{hdr, plte, {typ: "IDAT", data: zlibBytes(bad, 6)}, iend})
	one("pixel-too-much", []pngChunk{hdr, plte, {typ: "IDAT", data: zlibBytes(append(append([]byte(nil), raw...), 0), 6)}, iend})
	one("pixel-not-enough", []pngChunk{hdr, plte, {typ: "IDAT", data: zlibBytes(raw[:len(raw)-1], 6)}, iend})
	adler := append([]byte(nil), z...)
	adler[len(adler)-1] ^= 1
	one("zlib-adler", []pngChunk{hdr, plte, {typ: "IDAT", data: adler}, iend})
	one("zlib-no-adler", []pngChunk{hdr, plte, {typ: "IDAT", data: z[:len(z)-4]}, iend})
	head := append([]byte(nil), z...)
	head[0] = 0x79
	one("zlib-header", []pngChunk{hdr, plte, {typ: "IDAT", data: head}, iend})
	dict := append([]byte{0x78, 0xbb}, z[2:]...)
	one("zlib-fdict", []pngChunk{hdr, plte, {typ: "IDAT", data: dict}, iend})
	corrupt := append([]byte(nil), z...)
	corrupt[2] = 0xff
	one("deflate-corrupt", []pngChunk{hdr, plte, {typ: "IDAT", data: corrupt}, iend})
	one("idat-one-byte", []pngChunk{hdr, plte, {typ: "IDAT", data: z[:1]}, iend})

	// zlib 串流之後夾帶垃圾：Go 只看得到還沒被兩層 bufio 拉進來的部分。
	noise, noiseZ := indexedPNG(80, 80, 8, 256, randomIndices(rng, 6400, 256), false, nil, zlib.BestCompression)
	_ = noise
	small := goodZ
	for _, stream := range []struct {
		name string
		z    []byte
		w, h int
	}{{"small", small, 5, 3}, {"noise", noiseZ, 80, 80}} {
		for _, extra := range []int{1, 10, 1000, 3000, 5000, 9000} {
			garbage := make([]byte, extra)
			for i := range garbage {
				garbage[i] = byte(rng.Uint32())
			}
			pl := 16
			if stream.name == "noise" {
				pl = 256
			}
			data := buildPNG([]pngChunk{
				{typ: "IHDR", data: ihdr(uint32(stream.w), uint32(stream.h), 8, 3, 0, 0, 0)},
				{typ: "PLTE", data: palette(pl)},
				{typ: "IDAT", data: append(append([]byte(nil), stream.z...), garbage...)},
				iend,
			})
			// 前置一個長度不一的 tEXt，讓 IDAT 起點相對 4096 位元組的 bufio 邊界移動。
			for _, pad := range []int{0, 2000} {
				shifted := data
				if pad > 0 {
					t := buildPNG([]pngChunk{{typ: "tEXt", data: append([]byte("p\x00"), make([]byte, pad)...)}})[8:]
					shifted = append(append(append([]byte(nil), data[:33]...), t...), data[33:]...)
				}
				add(fmt.Sprintf("garbage-%s-%d-pad%d", stream.name, extra, pad), fourSame(9, shifted), ".", 9)
			}
		}
	}
	add("ok-noise", fourSame(9, noise), ".", 9)

	// 隨機串流：較大的 indexed 影像、四種壓縮等級（含 stored 區塊）、隨機切成多個 IDAT、
	// 前置不等長的 tEXt 移動 4096 位元組邊界，再對 zlib 串流做變形。
	levels := []int{zlib.NoCompression, zlib.BestSpeed, zlib.DefaultCompression, zlib.BestCompression}
	for i := 0; i < count/2; i++ {
		w, h := 1+rng.IntN(100), 1+rng.IntN(100)
		depth := []int{1, 2, 4, 8}[rng.IntN(4)]
		idx := make([]byte, w*h)
		smooth := rng.IntN(2) == 0
		for j := range idx {
			if smooth {
				idx[j] = byte((j%w + j/w) / 7)
			} else {
				idx[j] = byte(rng.Uint32())
			}
			idx[j] &= byte(1<<depth - 1)
		}
		filters := make([]byte, 1+rng.IntN(5))
		for j := range filters {
			filters[j] = byte(rng.IntN(5))
		}
		il := rng.IntN(4) == 0
		ilByte := byte(0)
		if il {
			ilByte = 1
		}
		z := zlibBytes(rawRows(w, h, idx, depth, il, filters), levels[rng.IntN(len(levels))])
		kind := "stream-clean"
		switch rng.IntN(20) {
		case 0, 1, 2:
			z[2+rng.IntN(len(z)-2)] ^= byte(1 << rng.IntN(8))
			kind = "stream-bitflip"
		case 3, 4:
			z = z[:rng.IntN(len(z))]
			kind = "stream-truncated"
		case 5, 6, 7:
			garbage := make([]byte, rng.IntN(9000))
			for j := range garbage {
				garbage[j] = byte(rng.Uint32())
			}
			z = append(z, garbage...)
			kind = "stream-garbage"
		case 8:
			at := 2 + rng.IntN(len(z)-1)
			z = append(z[:at:at], append([]byte{byte(rng.Uint32())}, z[at:]...)...)
			kind = "stream-insert"
		case 9:
			z = z[:len(z)-4]
			kind = "stream-no-adler"
		}
		chunks := []pngChunk{
			{typ: "IHDR", data: ihdr(uint32(w), uint32(h), byte(depth), 3, 0, 0, ilByte)},
		}
		if pad := rng.IntN(6000); rng.IntN(2) == 0 {
			chunks = append(chunks, pngChunk{typ: "tEXt", data: append([]byte("p\x00"), make([]byte, pad)...)})
		}
		chunks = append(chunks, pngChunk{typ: "PLTE", data: palette(1 << depth)})
		for rest := z; ; {
			if len(rest) == 0 || rng.IntN(3) == 0 {
				chunks = append(chunks, pngChunk{typ: "IDAT", data: rest})
				break
			}
			cut := rng.IntN(len(rest) + 1)
			chunks = append(chunks, pngChunk{typ: "IDAT", data: rest[:cut]})
			rest = rest[cut:]
		}
		chunks = append(chunks, iend)
		add(kind, withMouth(base, 0, buildPNG(chunks)), ".", 7)
	}

	// 隨機：前三張合法，第 4 張（或隨機一張）是 Go 編碼後再變形的 PNG。
	for i := 0; i < count; i++ {
		w, h := 1+rng.IntN(12), 1+rng.IntN(12)
		var img image.Image
		switch rng.IntN(6) {
		case 0:
			g := image.NewGray(image.Rect(0, 0, w, h))
			for j := range g.Pix {
				g.Pix[j] = byte(rng.Uint32())
			}
			img = g
		case 1:
			c := image.NewNRGBA(image.Rect(0, 0, w, h))
			for j := range c.Pix {
				c.Pix[j] = byte(rng.Uint32())
			}
			img = c
		default:
			n := 1 + rng.IntN(256)
			pal := make(color.Palette, n)
			for j := range pal {
				pal[j] = color.RGBA{byte(rng.Uint32()), byte(rng.Uint32()), byte(rng.Uint32()), 255}
			}
			p := image.NewPaletted(image.Rect(0, 0, w, h), pal)
			for j := range p.Pix {
				p.Pix[j] = byte(rng.IntN(n))
			}
			img = p
		}
		data := encodeImage(img)
		kind := "random-clean"
		switch rng.IntN(10) {
		case 0, 1:
			data = data[:rng.IntN(len(data))]
			kind = "random-truncated"
		case 2, 3, 4:
			data[rng.IntN(len(data))] ^= byte(1 << rng.IntN(8))
			kind = "random-bitflip"
		case 5:
			at := rng.IntN(len(data) + 1)
			data = append(data[:at:at], append([]byte{byte(rng.Uint32())}, data[at:]...)...)
			kind = "random-insert"
		}
		mouth := 3
		if rng.IntN(3) == 0 {
			mouth = rng.IntN(4)
		}
		add(kind, withMouth(base, mouth, data), ".", 7)
	}
	return rows
}

// ---- 真實檔案 ----

func fileInfo(path string) map[string]any {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return map[string]any{"path": path, "name": filepath.Base(path), "size": len(data), "sha256": digest(data)}
}

func realDAT(path string) map[string]any {
	info := fileInfo(path)
	var rows []map[string]any
	for resource := -1; ; resource++ {
		frames, err := dato.DecodeResource(path, resource)
		row := map[string]any{"resource": resource, "result": framesResult(frames, err)}
		if resource >= 0 {
			if raw, rawErr := fdother.ReadResource(path, resource); rawErr == nil {
				row["entry_sha256"], row["entry_len"] = digest(raw), len(raw)
			}
		}
		rows = append(rows, row)
		if resource >= 0 && err != nil && err.Error() == "fdother: archive resource is out of range" {
			break
		}
	}
	info["resources"] = rows
	return info
}

var portraitName = regexp.MustCompile(`^DATO_(\d{3,})_m(\d)\.png$`)

func realSeparated(dir, datPath string) map[string]any {
	entries, err := os.ReadDir(dir)
	if err != nil {
		panic(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	var files []map[string]any
	ids := map[int]bool{}
	for _, name := range names {
		files = append(files, fileInfo(filepath.Join(dir, name)))
		if m := portraitName.FindStringSubmatch(name); m != nil {
			id, _ := strconv.Atoi(m[1])
			ids[id] = true
		}
	}
	var sorted []int
	for id := range ids {
		sorted = append(sorted, id)
	}
	sort.Ints(sorted)
	if len(sorted) > 0 {
		sorted = append(sorted, sorted[len(sorted)-1]+1)
	}
	var rows []map[string]any
	for _, id := range sorted {
		frames, err := dato.LoadSeparatedResource(dir, id)
		row := map[string]any{"resource": id, "result": framesResult(frames, err)}
		if err == nil {
			want, wantErr := dato.DecodeResource(datPath, id)
			equal := wantErr == nil && len(want) == len(frames)
			for i := 0; equal && i < len(want); i++ {
				equal = want[i].Width == frames[i].Width && want[i].Height == frames[i].Height &&
					bytes.Equal(want[i].Pixels, frames[i].Pixels)
			}
			row["equals_archive"] = equal
		}
		rows = append(rows, row)
	}
	return map[string]any{"dir": dir, "files": files, "resources": rows}
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	cases := flag.Int("cases", 240, "每一類的隨機案例數")
	datPath := flag.String("dat", "", "原版 DATO.DAT（必要）")
	portraits := flag.String("portraits", "", "分離頭像目錄（必要）")
	tmp := flag.String("tmp", "", "分離頭像案例的暫存目錄（必要，需為空目錄）")
	flag.Parse()
	if *out == "" || *datPath == "" || *portraits == "" || *tmp == "" {
		fmt.Fprintln(os.Stderr, "需要 -out、-dat、-portraits 與 -tmp")
		os.Exit(2)
	}
	rng := rand.New(rand.NewPCG(*seed, 0x4441544f))
	doc := map[string]any{
		"schema":    "fd2-golden-dato/1",
		"seed":      *seed,
		"parse":     parseCases(rng, *cases),
		"blit":      blitCases(rng, *cases),
		"mouth":     mouthCases(rng, *cases),
		"separated": separatedCases(rng, *tmp, *cases),
		"real_dat":  realDAT(*datPath),
	}
	doc["real_separated"] = realSeparated(*portraits, *datPath)
	data, err := json.Marshal(doc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("fd2-golden-dato: parse=%d blit=%d mouth=%d separated=%d sha256=%s\n",
		len(doc["parse"].([]map[string]any)), len(doc["blit"].([]map[string]any)),
		len(doc["mouth"].([]map[string]any)), len(doc["separated"].([]map[string]any)), digest(data))
}
