// fd2-golden-afm 輸出 internal/afm 的黃金差分資料，供 Rust 改寫逐欄比對
// （docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 afm 的公開 API（ArchiveResourceCount、
// ReadArchiveResource、DecodeResource、LoadSeparatedResource），不改任何遊戲行為。
//
// 每個案例是一棵暫存目錄下的檔案樹加上一次 API 呼叫。合成輸入以固定種子的 PCG 產生，
// 檔案內容以 hex 寫進 JSON；由原版檔衍生的輸入只記錄來源路徑與變形步驟，原版檔本身
// 只記路徑、大小與 SHA-256，Rust 端讀同一份檔案重建。錯誤字串內的暫存路徑換成 $ROOT，
// 原版目錄換成 $ORG／$ANIM，讓輸出與執行位置無關。
//
// 本工具刻意只匯入 image/png（afm 已註冊），不匯入其他影像解碼器，以免改變
// image.Decode 的格式清單。
package main

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"hash/adler32"
	"hash/crc32"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/afm"
)

type opJSON struct {
	Op  string `json:"op"`
	At  int    `json:"at"`
	N   int    `json:"n"`
	Val int    `json:"val"`
	Hex string `json:"hex,omitempty"`
}

type fromJSON struct {
	Root string   `json:"root"`
	Path string   `json:"path"`
	Ops  []opJSON `json:"ops"`
}

type fileJSON struct {
	Path string    `json:"path"`
	Hex  string    `json:"hex,omitempty"`
	Dir  bool      `json:"dir,omitempty"`
	From *fromJSON `json:"from,omitempty"`
}

type caseJSON struct {
	Index     int            `json:"index"`
	Kind      string         `json:"kind"`
	Note      string         `json:"note"`
	Call      string         `json:"call"`
	Path      string         `json:"path"`
	Resources []int          `json:"resources"`
	Files     []fileJSON     `json:"files"`
	Result    map[string]any `json:"result"`
}

type realFile struct {
	Path   string `json:"path"`
	Size   int    `json:"size"`
	SHA256 string `json:"sha256"`
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func die(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// ---------- 檔案變形（Rust 端以相同規則重建） ----------

func fixCRC(b []byte) {
	pos := 8
	for pos+8 <= len(b) {
		n := binary.BigEndian.Uint32(b[pos:])
		end := pos + 8 + int(n)
		if n > 0x7fffffff || end+4 > len(b) {
			return
		}
		binary.BigEndian.PutUint32(b[end:], crc32.ChecksumIEEE(b[pos+4:end]))
		pos = end + 4
	}
}

func extractResource(b []byte, res int) []byte {
	first := int(binary.LittleEndian.Uint32(b[6:]))
	n := (first - 6) / 4
	start := int(binary.LittleEndian.Uint32(b[6+4*res:]))
	end := len(b)
	if res+1 < n {
		end = int(binary.LittleEndian.Uint32(b[6+4*(res+1):]))
	}
	return append([]byte(nil), b[start:end]...)
}

func container(resources ...[]byte) []byte {
	out := []byte("LLLLLL")
	off := 6 + 4*len(resources)
	for _, r := range resources {
		out = binary.LittleEndian.AppendUint32(out, uint32(off))
		off += len(r)
	}
	for _, r := range resources {
		out = append(out, r...)
	}
	return out
}

func applyOps(data []byte, ops []opJSON) []byte {
	b := append([]byte(nil), data...)
	for _, op := range ops {
		switch op.Op {
		case "xor":
			b[op.At] ^= byte(op.Val)
		case "set16":
			binary.LittleEndian.PutUint16(b[op.At:], uint16(op.Val))
		case "truncate":
			b = b[:op.N]
		case "insert":
			ins, err := hex.DecodeString(op.Hex)
			if err != nil {
				die(err)
			}
			b = append(b[:op.At:op.At], append(ins, b[op.At:]...)...)
		case "delete":
			b = append(b[:op.At:op.At], b[op.At+op.N:]...)
		case "fixcrc":
			fixCRC(b)
		case "extract":
			b = extractResource(b, op.N)
		case "wrap":
			b = container(b)
		default:
			die(fmt.Errorf("未知變形 %q", op.Op))
		}
	}
	return b
}

// ---------- 案例執行 ----------

type runner struct {
	roots   map[string]string // org／anim → 實際目錄
	scratch string
}

func (r *runner) norm(base, s string) string {
	s = strings.ReplaceAll(s, base, "$ROOT")
	if d := r.roots["org"]; d != "" {
		s = strings.ReplaceAll(s, d, "$ORG")
	}
	if d := r.roots["anim"]; d != "" {
		s = strings.ReplaceAll(s, d, "$ANIM")
	}
	return s
}

func (r *runner) expand(base, p string) string {
	p = strings.ReplaceAll(p, "$ROOT", base)
	p = strings.ReplaceAll(p, "$ORG", r.roots["org"])
	return strings.ReplaceAll(p, "$ANIM", r.roots["anim"])
}

func (r *runner) fileBytes(f fileJSON) []byte {
	if f.From != nil {
		raw, err := os.ReadFile(filepath.Join(r.roots[f.From.Root], f.From.Path))
		if err != nil {
			die(err)
		}
		return applyOps(raw, f.From.Ops)
	}
	b, err := hex.DecodeString(f.Hex)
	if err != nil {
		die(err)
	}
	return b
}

func (r *runner) materialize(base string, files []fileJSON) {
	if err := os.MkdirAll(base, 0o755); err != nil {
		die(err)
	}
	for _, f := range files {
		full := filepath.Join(base, f.Path)
		if f.Dir {
			if err := os.MkdirAll(full, 0o755); err != nil {
				die(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			die(err)
		}
		if err := os.WriteFile(full, r.fileBytes(f), 0o644); err != nil {
			die(err)
		}
	}
}

func clipJSON(c *afm.Clip) map[string]any {
	frames := make([]map[string]any, len(c.Frames))
	for i, f := range c.Frames {
		b := f.Bounds()
		frames[i] = map[string]any{
			"x0": b.Min.X, "y0": b.Min.Y, "w": b.Dx(), "h": b.Dy(), "stride": f.Stride,
			"rgba": digest(f.Pix), "indexed": digest(c.IndexedFrames[i]), "palette": digest(c.Palettes[i]),
		}
	}
	return map[string]any{
		"title_hex": hex.EncodeToString([]byte(c.Title)), "header_frames": c.HeaderFrames,
		"n_frames": len(c.Frames), "n_indexed": len(c.IndexedFrames), "n_palettes": len(c.Palettes),
		"frames": frames,
	}
}

func (r *runner) errValue(base string, err error) any {
	if err == nil {
		return nil
	}
	return r.norm(base, err.Error())
}

// guarded 執行 f；Go 端 panic（例如 op9 越界切片）記為 panic=true。
func guarded(f func() map[string]any) (out map[string]any) {
	defer func() {
		if recover() != nil {
			out = map[string]any{"panic": true}
		}
	}()
	return f()
}

func (r *runner) run(index int, c *caseJSON) {
	base := filepath.Join(r.scratch, fmt.Sprintf("case-%04d", index))
	r.materialize(base, c.Files)
	path := r.expand(base, c.Path)
	result := map[string]any{}
	switch c.Call {
	case "archive":
		n, err := afm.ArchiveResourceCount(path)
		result["count"] = map[string]any{"n": n, "err": r.errValue(base, err)}
		var reads, decodes []map[string]any
		for _, res := range c.Resources {
			raw, err := afm.ReadArchiveResource(path, res)
			reads = append(reads, map[string]any{
				"res": res, "len": len(raw), "sha256": digest(raw), "err": r.errValue(base, err),
			})
			decodes = append(decodes, guarded(func() map[string]any {
				clip, err := afm.DecodeResource(path, res)
				row := map[string]any{"res": res, "err": r.errValue(base, err)}
				if err == nil {
					row["clip"] = clipJSON(clip)
				}
				return row
			}))
		}
		result["read"], result["decode"] = reads, decodes
	case "separated":
		var loads []map[string]any
		for _, res := range c.Resources {
			loads = append(loads, guarded(func() map[string]any {
				clip, err := afm.LoadSeparatedResource(path, res)
				row := map[string]any{"res": res, "err": r.errValue(base, err)}
				if err == nil {
					row["clip"] = clipJSON(clip)
				}
				return row
			}))
		}
		result["load"] = loads
	default:
		die(fmt.Errorf("未知呼叫 %q", c.Call))
	}
	c.Index = index
	c.Result = result
	if err := os.RemoveAll(base); err != nil {
		die(err)
	}
}

// ---------- AFM 合成 ----------

func u16(v int) []byte { return []byte{byte(v), byte(v >> 8)} }

func rleStream(rng *rand.Rand, count, runPercent, minRun int) []byte {
	var out []byte
	for written := 0; written < count; {
		if rng.IntN(100) < runPercent {
			n := minRun + rng.IntN(64-minRun)
			out = append(out, byte(0xc0|n), byte(rng.IntN(256)))
			written += n
		} else {
			out = append(out, byte(rng.IntN(0xc0)))
			written++
		}
	}
	return out
}

// afmCommand 產生一個指令；panicOK 允許 op9 偏移落在 64000 之後（Go 版會 panic）。
func afmCommand(rng *rand.Rand, op int, panicOK bool) []byte {
	switch op {
	case 0, 4:
		return []byte{byte(op), byte(rng.IntN(256))}
	case 1:
		b := []byte{1}
		for i := 0; i < 768; i++ {
			b = append(b, byte(rng.IntN(256)))
		}
		return b
	case 2:
		return append([]byte{2}, rleStream(rng, 768, 50, 0)...)
	case 3:
		n := rng.IntN(5)
		b := []byte{3, byte(n)}
		for k := 0; k < n; k++ {
			idx := rng.IntN(256)
			cnt := rng.IntN(257 - idx)
			if rng.IntN(20) == 0 {
				cnt = 257 - idx + rng.IntN(idx+1)
				if cnt > 255 {
					cnt = 255
				}
			}
			b = append(b, byte(idx), byte(cnt))
			for i := 0; i < cnt*3; i++ {
				b = append(b, byte(rng.IntN(64)))
			}
		}
		return b
	case 5:
		b := []byte{5}
		seedA, seedB := rng.IntN(256), rng.IntN(7)+1
		for i := 0; i < 64000; i++ {
			b = append(b, byte((i/320)*seedB+(i%320)^seedA))
		}
		return b
	case 6:
		return append([]byte{6}, rleStream(rng, 64000, 97, 40)...)
	case 7, 8, 9:
		n := rng.IntN(map[int]int{7: 21, 8: 11, 9: 7}[op])
		b := append([]byte{byte(op)}, u16(n)...)
		for k := 0; k < n; k++ {
			length := rng.IntN(256)
			var off int
			switch p := rng.IntN(100); {
			case op == 9 && p < 85:
				off = rng.IntN(64000 - length + 1)
			case op == 9 && (p < 95 || !panicOK):
				off = 64000 - length + rng.IntN(length+1)
			case op == 9:
				off = 64001 + rng.IntN(65536-64001)
			case p < 90:
				off = rng.IntN(64000)
			default:
				off = 64000 + rng.IntN(1536)
			}
			b = append(b, u16(off)...)
			switch op {
			case 7:
				b = append(b, byte(rng.IntN(256)))
			case 8:
				b = append(b, byte(length), byte(rng.IntN(256)))
			case 9:
				b = append(b, byte(length))
				for i := 0; i < length; i++ {
					b = append(b, byte(rng.IntN(256)))
				}
			}
		}
		return b
	default:
		return []byte{byte(op)}
	}
}

type afmFrame struct {
	script   []byte
	cmdCount int
	extra    [4]byte
}

// afmScript 產生一幀；fault 決定注入哪一種 script 層錯誤。
func afmScript(rng *rand.Rand, fault string, panicOK bool) afmFrame {
	ops := []int{0, 1, 2, 3, 4, 6, 7, 8, 9}
	k := 1 + rng.IntN(5)
	var script []byte
	usedBig := false
	for i := 0; i < k; i++ {
		op := ops[rng.IntN(len(ops))]
		if (op == 1 || op == 6) && usedBig {
			op = 0
		}
		if op == 1 || op == 6 {
			usedBig = true
		}
		script = append(script, afmCommand(rng, op, panicOK)...)
	}
	f := afmFrame{script: script, cmdCount: k}
	for i := range f.extra {
		f.extra[i] = byte(rng.IntN(256))
	}
	switch fault {
	case "more-cmds":
		f.cmdCount += 1 + rng.IntN(3)
	case "fewer-cmds":
		f.cmdCount--
	case "cut":
		if len(f.script) > 1 {
			f.script = f.script[:1+rng.IntN(len(f.script)-1)]
		}
	case "bad-op":
		f.script = append(f.script, byte(10+rng.IntN(246)))
		f.cmdCount++
	}
	return f
}

func afmResource(rng *rand.Rand, frames []afmFrame, headerFrames, w, h int) []byte {
	b := make([]byte, 0xad)
	for i := range b {
		b[i] = byte(rng.IntN(256))
	}
	for i := 0x51; i < 0xa1; i++ {
		switch rng.IntN(4) {
		case 0:
			b[i] = 0
		case 1:
			b[i] = ' '
		case 2:
			b[i] = byte(0x20 + rng.IntN(0x5f))
		}
	}
	binary.LittleEndian.PutUint16(b[0xa5:], uint16(headerFrames))
	binary.LittleEndian.PutUint16(b[0xa7:], uint16(w))
	binary.LittleEndian.PutUint16(b[0xa9:], uint16(h))
	for _, f := range frames {
		b = append(b, u16(len(f.script))...)
		b = append(b, u16(f.cmdCount)...)
		b = append(b, f.extra[:]...)
		b = append(b, f.script...)
	}
	return b
}

func hexFile(path string, data []byte) fileJSON {
	return fileJSON{Path: path, Hex: hex.EncodeToString(data)}
}

func archiveCase(kind, note string, data []byte, resources []int) caseJSON {
	return caseJSON{Kind: kind, Note: note, Call: "archive", Path: "$ROOT/x.dat", Resources: resources,
		Files: []fileJSON{hexFile("x.dat", data)}}
}

func span(lo, hi int) []int {
	var out []int
	for i := lo; i <= hi; i++ {
		out = append(out, i)
	}
	return out
}

func randomArchiveCase(rng *rand.Rand, i int) caseJSON {
	faults := []string{"", "", "", "", "more-cmds", "fewer-cmds", "cut", "bad-op"}
	nres := 1 + rng.IntN(3)
	var resources [][]byte
	var notes []string
	for r := 0; r < nres; r++ {
		nf := 1 + rng.IntN(4)
		var frames []afmFrame
		for f := 0; f < nf; f++ {
			fault := ""
			if rng.IntN(4) == 0 {
				fault = faults[rng.IntN(len(faults))]
			}
			frames = append(frames, afmScript(rng, fault, rng.IntN(25) == 0))
		}
		headerFrames, w, h := nf, 320, 200
		res := []byte(nil)
		switch rng.IntN(16) {
		case 0:
			headerFrames = nf + 1
			notes = append(notes, "header+1")
		case 1:
			headerFrames = 0
			notes = append(notes, "zero")
		case 2:
			w = 320 + rng.IntN(3) - 1
			h = 200 + rng.IntN(3) - 1
			notes = append(notes, "geometry")
		}
		res = afmResource(rng, frames, headerFrames, w, h)
		switch rng.IntN(16) {
		case 0:
			res = append(res, byte(rng.IntN(256)))
			notes = append(notes, "trailing")
		case 1:
			res = res[:len(res)-1-rng.IntN(10)]
			notes = append(notes, "short")
		case 2:
			res = res[:rng.IntN(0xad)]
			notes = append(notes, "tiny")
		}
		resources = append(resources, res)
	}
	data := container(resources...)
	switch rng.IntN(20) {
	case 0:
		data[rng.IntN(6)] ^= byte(1 + rng.IntN(255))
		notes = append(notes, "magic")
	case 1:
		data[6+rng.IntN(4*nres)] ^= byte(1 << rng.IntN(8))
		notes = append(notes, "directory")
	}
	return archiveCase("afm-random", strings.Join(notes, ","), data, span(-1, nres))
}

func frameOf(script []byte, cmdCount int) afmFrame {
	return afmFrame{script: script, cmdCount: cmdCount}
}

func singleFrameArchive(rng *rand.Rand, note string, script []byte, cmdCount int) caseJSON {
	res := afmResource(rng, []afmFrame{frameOf(script, cmdCount)}, 1, 320, 200)
	return archiveCase("afm-structured", note, container(res), span(-1, 1))
}

func structuredArchiveCases(rng *rand.Rand) []caseJSON {
	var out []caseJSON
	add := func(note string, data []byte, resources []int) {
		out = append(out, archiveCase("afm-structured", note, data, resources))
	}
	add("empty file", nil, span(-1, 1))
	add("5 bytes", []byte("LLLLL"), span(-1, 1))
	add("9 bytes", []byte("LLLLLL\x06\x00\x00"), span(-1, 1))
	add("no entries", []byte("LLLLLL\x06\x00\x00\x00"), span(-1, 1))
	add("first=5", []byte("LLLLLL\x05\x00\x00\x00xx"), span(-1, 1))
	add("first=7", []byte("LLLLLL\x07\x00\x00\x00xx"), span(-1, 1))
	add("first=11", []byte("LLLLLL\x0b\x00\x00\x00xxxxxxx"), span(-1, 1))
	add("first>len", []byte("LLLLLL\x0e\x00\x00\x00\x00\x00\x00\x00"), span(-1, 1))
	add("wrong magic", append([]byte("LLLLLX"), container([]byte("abc"))[6:]...), span(-1, 1))
	two := container(make([]byte, 0xad), make([]byte, 0xad))
	dec := append([]byte(nil), two...)
	binary.LittleEndian.PutUint32(dec[6:], uint32(len(dec)-1))
	add("offsets decreasing", dec, span(-1, 2))
	over := append([]byte(nil), two...)
	binary.LittleEndian.PutUint32(over[10:], uint32(len(over)+1))
	add("offset past end", over, span(-1, 2))
	add("zero-length entry", container(make([]byte, 0)), span(-1, 1))
	add("0xAC bytes", container(make([]byte, 0xac)), span(-1, 1))
	zero := make([]byte, 0xad)
	binary.LittleEndian.PutUint16(zero[0xa7:], 320)
	binary.LittleEndian.PutUint16(zero[0xa9:], 200)
	add("0xAD bytes zero frames", container(zero), span(-1, 1))
	add("record truncated", container(append(append([]byte(nil), zero...), 1, 2, 3)), span(-1, 1))

	frame := func(note string, script []byte, cmdCount int) {
		out = append(out, singleFrameArchive(rng, note, script, cmdCount))
	}
	frame("op0+op4", []byte{0, 63, 4, 255}, 2)
	frame("palette >63 then RGBA", append(append([]byte{1}, bytes.Repeat([]byte{0xff, 0x80, 0x41}, 256)...), 4, 7), 2)
	frame("rle zero-length runs", append([]byte{2}, append(bytes.Repeat([]byte{0xc0, 9}, 50), rleStream(rng, 768, 50, 0)...)...), 1)
	frame("rle cut at ctrl", []byte{2, 1, 2, 3}, 1)
	frame("rle cut at run value", []byte{2, 0xc5}, 1)
	frame("op1 short", append([]byte{1}, make([]byte, 767)...), 1)
	frame("op3 idx cut", []byte{3, 1}, 1)
	frame("op3 cnt cut", []byte{3, 1, 4}, 1)
	frame("op3 data cut", []byte{3, 1, 4, 2, 1, 2, 3}, 1)
	frame("op3 past palette", append([]byte{3, 1, 255, 2}, make([]byte, 6)...), 1)
	frame("op3 exact end", append([]byte{3, 1, 255, 1}, 9, 9, 9), 1)
	frame("op5 short", append([]byte{5}, make([]byte, 63999)...), 1)
	frame("op5 exact", afmCommand(rng, 5, false), 1)
	frame("op7 count cut", []byte{7, 1}, 1)
	frame("op7 entry cut", []byte{7, 1, 0, 0x10}, 1)
	frame("op7 off>=64000", []byte{7, 2, 0, 0x00, 0xfa, 5, 0xff, 0xff, 6}, 1)
	frame("op8 entry cut", []byte{8, 1, 0, 0x10, 0x10, 5}, 1)
	frame("op8 off>=64000 and clip", []byte{8, 2, 0, 0x00, 0xfa, 9, 5, 0xf0, 0xf9, 200, 6}, 1)
	frame("op9 entry cut", []byte{9, 1, 0, 0x10}, 1)
	frame("op9 data cut", []byte{9, 1, 0, 0x10, 0x10, 4, 1, 2}, 1)
	frame("op9 off=64000", append([]byte{9, 1, 0, 0x00, 0xfa, 3}, 1, 2, 3), 1)
	frame("op9 clip tail", append([]byte{9, 1, 0, 0xfe, 0xf9, 5}, 1, 2, 3, 4, 5), 1)
	frame("op9 off>64000 panics in Go", append([]byte{9, 1, 0, 0x01, 0xfa, 3}, 1, 2, 3), 1)
	frame("op9 off>64000 but data cut", []byte{9, 1, 0, 0x01, 0xfa, 3, 1}, 1)
	frame("unknown opcode", []byte{0, 1, 10}, 2)
	frame("unknown opcode 255", []byte{255}, 1)
	frame("cmd count beyond script", []byte{0, 1}, 3)
	frame("zero commands with bytes", []byte{4, 4, 4}, 0)
	frame("empty script", nil, 0)
	frame("op6 full screen", afmCommand(rng, 6, false), 1)

	multi := afmResource(rng, []afmFrame{afmScript(rng, "", false), afmScript(rng, "", false), afmScript(rng, "", false)}, 3, 320, 200)
	add("three resources incl. empty", container(multi, nil, multi[:0xad+3]), span(-1, 3))
	return out
}

// ---------- PNG 合成 ----------

type chunk struct {
	typ  string
	data []byte
}

func serialize(chunks []chunk) []byte {
	b := []byte("\x89PNG\r\n\x1a\n")
	for _, c := range chunks {
		b = binary.BigEndian.AppendUint32(b, uint32(len(c.data)))
		start := len(b)
		b = append(b, c.typ...)
		b = append(b, c.data...)
		b = binary.BigEndian.AppendUint32(b, crc32.ChecksumIEEE(b[start:]))
	}
	return b
}

type pngSpec struct {
	w, h, depth, ct, interlace int
	filter                     func(row int) int // -1 表示隨機
	level                      int
	split                      []int // IDAT 切塊大小循環；空表示一塊
	plte                       int   // 調色盤項數；0 表示依深度
	flushEvery                 int
	pattern                    int
	extraRows                  int // 多寫的原始位元組（too much pixel data）
	shortBytes                 int // 少寫的原始位元組
	badFilterRow               int // >0 時該列濾波器設為 5
}

func channels(ct int) int {
	return map[int]int{0: 1, 2: 3, 3: 1, 4: 2, 6: 4}[ct]
}

var adam7 = [7][4]int{{8, 8, 0, 0}, {8, 8, 4, 0}, {4, 8, 0, 4}, {4, 4, 2, 0}, {2, 4, 0, 2}, {2, 2, 1, 0}, {1, 2, 0, 1}}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
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

// rawScanlines 產生已加濾波器位元組的未壓縮影像資料。
func rawScanlines(rng *rand.Rand, s pngSpec) []byte {
	bpp := channels(s.ct) * s.depth
	bytesPP := (bpp + 7) / 8
	type pass struct{ w, h int }
	var passes []pass
	if s.interlace == 1 {
		for _, p := range adam7 {
			w := (s.w - p[2] + p[0] - 1) / p[0]
			h := (s.h - p[3] + p[1] - 1) / p[1]
			if w > 0 && h > 0 {
				passes = append(passes, pass{w, h})
			}
		}
	} else {
		passes = []pass{{s.w, s.h}}
	}
	var out []byte
	row := 0
	for _, p := range passes {
		rowBytes := (bpp*p.w + 7) / 8
		prev := make([]byte, rowBytes)
		for y := 0; y < p.h; y++ {
			cur := make([]byte, rowBytes)
			for i := range cur {
				v := (i/bytesPP*3 + y*5) ^ s.pattern
				switch s.pattern % 3 {
				case 1:
					v = (i / (bytesPP * 16)) + (y/16)*7
				case 2:
					v = i*y + s.pattern
				}
				if rng.IntN(60) == 0 {
					v = rng.IntN(256)
				}
				cur[i] = byte(v)
			}
			ft := 0
			if s.filter != nil {
				ft = s.filter(row)
			}
			if ft < 0 {
				ft = rng.IntN(5)
			}
			enc := make([]byte, rowBytes)
			for i := range cur {
				left, up, ul := 0, int(prev[i]), 0
				if i >= bytesPP {
					left, ul = int(cur[i-bytesPP]), int(prev[i-bytesPP])
				}
				pred := 0
				switch ft {
				case 1:
					pred = left
				case 2:
					pred = up
				case 3:
					pred = (left + up) / 2
				case 4:
					pred = paeth(left, up, ul)
				}
				enc[i] = cur[i] - byte(pred)
			}
			if s.badFilterRow > 0 && row == s.badFilterRow {
				ft = 5
			}
			out = append(out, byte(ft))
			out = append(out, enc...)
			prev = cur
			row++
		}
	}
	for i := 0; i < s.extraRows; i++ {
		out = append(out, byte(rng.IntN(256)))
	}
	if s.shortBytes > 0 && s.shortBytes <= len(out) {
		out = out[:len(out)-s.shortBytes]
	}
	return out
}

func zlibBytes(raw []byte, level, flushEvery int) []byte {
	var b bytes.Buffer
	w, err := zlib.NewWriterLevel(&b, level)
	if err != nil {
		die(err)
	}
	if flushEvery <= 0 {
		flushEvery = len(raw) + 1
	}
	for start := 0; start < len(raw); start += flushEvery {
		end := min(start+flushEvery, len(raw))
		if _, err := w.Write(raw[start:end]); err != nil {
			die(err)
		}
		if end < len(raw) {
			if err := w.Flush(); err != nil {
				die(err)
			}
		}
	}
	if err := w.Close(); err != nil {
		die(err)
	}
	return b.Bytes()
}

// rawZlib 以自訂標頭與 adler32 包一段 raw deflate。
func rawZlib(raw []byte, cmf, flgBase byte, dict []byte, adler uint32) []byte {
	flg := flgBase &^ 0x1f
	for (int(cmf)*256+int(flg))%31 != 0 {
		flg++
	}
	out := []byte{cmf, flg}
	out = append(out, dict...)
	var b bytes.Buffer
	w, _ := flate.NewWriter(&b, 6)
	w.Write(raw)
	w.Close()
	out = append(out, b.Bytes()...)
	return binary.BigEndian.AppendUint32(out, adler)
}

func ihdr(w, h, depth, ct, interlace int) chunk {
	d := binary.BigEndian.AppendUint32(nil, uint32(w))
	d = binary.BigEndian.AppendUint32(d, uint32(h))
	return chunk{"IHDR", append(d, byte(depth), byte(ct), 0, 0, byte(interlace))}
}

func plteChunk(n int) chunk {
	var d []byte
	for i := 0; i < n; i++ {
		d = append(d, byte(i), byte(255-i), byte(i*7))
	}
	return chunk{"PLTE", d}
}

func idatChunks(z []byte, split []int) []chunk {
	if len(split) == 0 {
		return []chunk{{"IDAT", z}}
	}
	var out []chunk
	for i := 0; len(z) > 0 || i == 0; i++ {
		n := split[i%len(split)]
		n = min(n, len(z))
		out = append(out, chunk{"IDAT", z[:n]})
		z = z[n:]
	}
	return out
}

// pngChunks 依規格產生完整區塊清單（IHDR、PLTE、IDAT…、IEND）。
func pngChunks(rng *rand.Rand, s pngSpec) []chunk {
	chunks := []chunk{ihdr(s.w, s.h, s.depth, s.ct, s.interlace)}
	if s.ct == 3 {
		n := s.plte
		if n == 0 {
			n = min(256, 1<<s.depth)
		}
		chunks = append(chunks, plteChunk(n))
	}
	z := zlibBytes(rawScanlines(rng, s), s.level, s.flushEvery)
	chunks = append(chunks, idatChunks(z, s.split)...)
	return append(chunks, chunk{"IEND", nil})
}

func p8(w, h int) pngSpec {
	return pngSpec{w: w, h: h, depth: 8, ct: 3, level: 6, pattern: 1}
}

func withChunks(chunks []chunk, f func([]chunk) []chunk) []chunk {
	return f(append([]chunk(nil), chunks...))
}

func insertAt(chunks []chunk, i int, c ...chunk) []chunk {
	return append(chunks[:i:i], append(c, chunks[i:]...)...)
}

func findChunk(chunks []chunk, typ string) int {
	for i, c := range chunks {
		if c.typ == typ {
			return i
		}
	}
	return -1
}

// ---------- 分離動畫 fixture ----------

const fixtureRes = 4

var expectedFrameCounts = []int{96, 51, 26, 28, 12, 35, 12, 17, 12}
var expectedRawSizes = []int{1002800, 635952, 97726, 35566, 36113, 411039, 43553, 137859, 36893}

type kv struct{ k, v string }

func q(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		die(err)
	}
	return string(b)
}

func obj(kvs []kv) string {
	parts := make([]string, len(kvs))
	for i, p := range kvs {
		parts[i] = p.k + ":" + p.v
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func arr(items []string) string { return "[" + strings.Join(items, ",") + "]" }

func frameObj(i int, frame, palette string) string {
	return obj([]kv{{q("frame_id"), q(fmt.Sprintf("frame/%03d", i))}, {q("frame"), q(frame)},
		{q("palette"), q(palette)}, {q("source_frame"), fmt.Sprint(i)}})
}

func framesArr(res int, frame, palette string) string {
	var items []string
	for i := 0; i < expectedFrameCounts[res]; i++ {
		items = append(items, frameObj(i, frame, palette))
	}
	return arr(items)
}

func sourceKVs(res int) []kv {
	return []kv{{q("file"), q("ANI.DAT")}, {q("resource"), fmt.Sprint(res)}, {q("size"), fmt.Sprint(afm.ANIFileSize)},
		{q("md5"), q(afm.ANIMD5)}, {q("sha256"), q(afm.ANISHA256)}, {q("raw_size"), fmt.Sprint(expectedRawSizes[res])},
		{q("container_entry_count"), "10"}, {q("empty_tail_index"), "9"}}
}

func animationKVs(res int, title string) []kv {
	return []kv{{q("schema_version"), "1"}, {q("kind"), q("afm_indexed_animation")},
		{q("asset_id"), q(fmt.Sprintf("animation/ANI_%03d", res))}, {q("status"), q("decoded")},
		{q("evidence"), q("confirmed")}, {q("codec"), q("fd2_afm_vm_v1")}, {q("title"), q(title)},
		{q("width"), "320"}, {q("height"), "200"}, {q("frame_count"), fmt.Sprint(expectedFrameCounts[res])},
		{q("source"), obj(sourceKVs(res))}, {q("frames"), framesArr(res, "v.png", "p.json")}}
}

func replaceKV(kvs []kv, key, value string) []kv {
	out := append([]kv(nil), kvs...)
	for i := range out {
		if out[i].k == q(key) {
			out[i].v = value
		}
	}
	return out
}

func renameKV(kvs []kv, key, raw string) []kv {
	out := append([]kv(nil), kvs...)
	for i := range out {
		if out[i].k == q(key) {
			out[i].k = raw
		}
	}
	return out
}

func dropKV(kvs []kv, key string) []kv {
	var out []kv
	for _, p := range kvs {
		if p.k != q(key) {
			out = append(out, p)
		}
	}
	return out
}

func paletteValues(seed int) []string {
	var out []string
	for i := 0; i < 768; i++ {
		out = append(out, fmt.Sprint((i*7+seed)%64))
	}
	return out
}

func paletteKVs(values []string) []kv {
	return []kv{{q("schema_version"), "1"}, {q("kind"), q("vga_dac_6bit_snapshot")}, {q("dac_6bit_components"), arr(values)}}
}

type fixture struct {
	res       int
	animation string
	png       []byte
	palette   string
	extra     []fileJSON
	root      string
	resources []int
	noMeta    bool
	emptyRoot bool
}

var validPNG []byte

func (f fixture) build(kind, note string) caseJSON {
	dir := fmt.Sprintf("ANI_%03d", f.res)
	var files []fileJSON
	if !f.noMeta {
		files = append(files, fileJSON{Path: dir + "/animation.json", Hex: hex.EncodeToString([]byte(f.animation))})
	}
	if f.png != nil {
		files = append(files, hexFile(dir+"/v.png", f.png))
	}
	if f.palette != "" {
		files = append(files, fileJSON{Path: dir + "/p.json", Hex: hex.EncodeToString([]byte(f.palette))})
	}
	files = append(files, f.extra...)
	root := f.root
	if root == "" {
		root = "$ROOT"
	}
	if f.emptyRoot {
		root = ""
	}
	resources := f.resources
	if resources == nil {
		resources = []int{f.res}
	}
	return caseJSON{Kind: kind, Note: note, Call: "separated", Path: root, Resources: resources, Files: files}
}

func baseFixture() fixture {
	return fixture{res: fixtureRes, animation: obj(animationKVs(fixtureRes, "synthetic title")), png: validPNG,
		palette: obj(paletteKVs(paletteValues(3)))}
}

func metadataCases() []caseJSON {
	var out []caseJSON
	add := func(note string, kvs []kv) {
		f := baseFixture()
		f.animation = obj(kvs)
		out = append(out, f.build("separated-metadata", note))
	}
	addRaw := func(note, text string) {
		f := baseFixture()
		f.animation = text
		out = append(out, f.build("separated-metadata", note))
	}
	base := animationKVs(fixtureRes, "synthetic title")
	add("valid", base)
	for _, title := range []string{"", "\x00 pad\x00  ", "中文標題", strings.Repeat("長", 300)} {
		add("title "+strconv.QuoteToASCII(title[:min(len(title), 12)]), animationKVs(fixtureRes, title))
	}
	for _, raw := range []string{`"é\n\t\"\\\/"`, `"😀"`, `"\ud800"`, `"\udc00x"`, `"\ud800A"`,
		`"\ud800\ud800"`, "\"\xff\xfe\"", "\"\xe4\xb8\"", "\"\xf0\x9f\x98\"", "\"\xed\xa0\x80\"", "\"\xc0\xaf\"", `"\u0000"`} {
		add("title raw "+strconv.QuoteToASCII(raw), replaceKV(base, "title", raw))
	}
	wrong := map[string]string{
		"schema_version": "2", "kind": q("x"), "asset_id": q("animation/ANI_005"), "status": q("draft"),
		"evidence": q("strong"), "codec": q("v2"), "width": "319", "height": "201", "frame_count": "11",
	}
	keys := make([]string, 0, len(wrong))
	for k := range wrong {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		add("wrong "+k, replaceKV(base, k, wrong[k]))
	}
	srcWrong := map[string]string{"file": q("ANI.DAT2"), "resource": "5", "size": "1", "md5": q("0"), "sha256": q("0"),
		"raw_size": "1", "container_entry_count": "9", "empty_tail_index": "10"}
	keys = keys[:0]
	for k := range srcWrong {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		add("wrong source."+k, replaceKV(base, "source", obj(replaceKV(sourceKVs(fixtureRes), k, srcWrong[k]))))
	}
	var eleven, thirteen []string
	for i := 0; i < 13; i++ {
		if i < 11 {
			eleven = append(eleven, frameObj(i, "v.png", "p.json"))
		}
		thirteen = append(thirteen, frameObj(i, "v.png", "p.json"))
	}
	add("11 frames", replaceKV(base, "frames", arr(eleven)))
	add("13 frames", replaceKV(base, "frames", arr(thirteen)))
	for _, v := range []string{`"1"`, "1.0", "1e0", "true", "null", "[1]", `{"a":1}`, "-0", "01x"} {
		add("schema_version="+v, replaceKV(base, "schema_version", v))
	}
	for _, v := range []string{"1", "true", "null", "[]", "{}"} {
		add("kind="+v, replaceKV(base, "kind", v))
	}
	for _, v := range []string{`"x"`, "[1]", "null", "{}", "5", "true"} {
		add("source="+v, replaceKV(base, "source", v))
	}
	for _, v := range []string{"{}", `"x"`, "null", "[1]", "[]", "true", "[null]", `[{"frame_id":5}]`} {
		add("frames="+v, replaceKV(base, "frames", v))
	}
	for _, v := range []string{"2437547.0", "99999999999999999999", "-9223372036854775809", "2.437547e6", `"2437547"`} {
		add("source.size="+v, replaceKV(base, "source", obj(replaceKV(sourceKVs(fixtureRes), "size", v))))
	}
	frame0 := func(k, v string) string {
		items := []string{obj(replaceKV([]kv{{q("frame_id"), q("frame/000")}, {q("frame"), q("v.png")},
			{q("palette"), q("p.json")}, {q("source_frame"), "0"}}, k, v))}
		for i := 1; i < 12; i++ {
			items = append(items, frameObj(i, "v.png", "p.json"))
		}
		return arr(items)
	}
	for _, c := range [][2]string{{"source_frame", "-0"}, {"source_frame", `"0"`}, {"source_frame", "0.0"},
		{"source_frame", "null"}, {"frame_id", "5"}, {"frame_id", "null"}, {"frame_id", q("frame/0")}} {
		add("frames[0]."+c[0]+"="+c[1], replaceKV(base, "frames", frame0(c[0], c[1])))
	}
	// 鍵名摺疊
	add("key SCHEMA_VERSION", renameKV(base, "schema_version", q("SCHEMA_VERSION")))
	add("key Kind", renameKV(base, "kind", q("Kind")))
	add("key long-s status", renameKV(base, "status", q("ſtatus")))
	add("key escaped long-s", renameKV(base, "status", `"ſtatus"`))
	add("key kelvin kind", renameKV(base, "kind", `"Kind"`))
	add("key dotted-I width", renameKV(base, "width", q("wİdth")))
	add("key dotless-i width", renameKV(base, "width", q("wıdth")))
	add("key escaped plain", renameKV(base, "codec", `"codec"`))
	add("key trailing space", renameKV(base, "codec", q("codec ")))
	// 重複鍵與合併
	add("dup kind bad then good", append([]kv{{q("kind"), q("bad")}}, base...))
	add("dup kind good then bad", append(append([]kv(nil), base...), kv{q("kind"), q("bad")}))
	add("dup kind folded later", append(append([]kv(nil), base...), kv{q("KIND"), q("bad")}))
	add("dup source partial merge", append(append([]kv(nil), base...), kv{q("source"), `{"size":2437547}`}))
	add("dup source empty", append(append([]kv(nil), base...), kv{q("source"), "{}"}))
	add("dup source null", append(append([]kv(nil), base...), kv{q("source"), "null"}))
	add("dup source overrides", append(append([]kv(nil), base...), kv{q("source"), `{"md5":"x"}`}))
	add("dup frames shorter", append(append([]kv(nil), base...), kv{q("frames"), arr([]string{frameObj(0, "v.png", "p.json")})}))
	var empties []string
	for i := 0; i < 12; i++ {
		empties = append(empties, "{}")
	}
	stale := append(append([]kv(nil), base...), kv{q("frames"), `[{"frame_id":"frame/000"}]`}, kv{q("frames"), arr(empties)})
	add("stale slice elements resurface", stale)
	add("frames reset by [] then empties", append(append([]kv(nil), base...), kv{q("frames"), "[]"}, kv{q("frames"), arr(empties)}))
	add("frames reset by null then empties", append(append([]kv(nil), base...), kv{q("frames"), "null"}, kv{q("frames"), arr(empties)}))
	nulls := strings.Repeat("null,", 11) + "null"
	add("frames of nulls after full", append(append([]kv(nil), base...), kv{q("frames"), "[" + nulls + "]"}))
	// 未知欄位與巢狀
	add("unknown fields", append(append([]kv(nil), base...), kv{q("extra"), `{"a":[1,2,{"b":null}],"c":"é"}`}, kv{q(""), "0"}))
	add("deep unknown 200", append(append([]kv(nil), base...), kv{q("deep"), strings.Repeat("[", 200) + strings.Repeat("]", 200)}))
	add("deep unknown 10001", append(append([]kv(nil), base...), kv{q("deep"), strings.Repeat("[", 10001) + strings.Repeat("]", 10001)}))
	add("deep unknown 9999 objects", append(append([]kv(nil), base...), kv{q("deep"), strings.Repeat(`{"a":`, 9999) + "1" + strings.Repeat("}", 9999)}))
	add("missing kind", dropKV(base, "kind"))
	// 語法
	full := obj(base)
	for _, cut := range []int{0, 1, 2, 10, 100, len(full) / 2, len(full) - 1} {
		addRaw(fmt.Sprintf("truncated at %d", cut), full[:cut])
	}
	for _, s := range []string{full + " x", full + "}", "\xef\xbb\xbf" + full, " \t\r\n" + full + "\n\n", "[" + full + "]",
		`"str"`, "42", "true", "null", "{'a':1}", "{/*c*/}", `{"a":NaN}`, `{"a":01}`, `{"a":+1}`, `{"a":1 "b":2}`,
		`{"a":1,}`, `{"a":"\x"}`, `{"a":"\u12G4"}`, "{\"a\":\"\x01\"}", `{"a":-}`, `{"a":1.}`, `{"a":1e}`, `{"a":tru}`,
		`{"a":fals}`, `{"a":nul}`, `{,}`, `{"a"}`, `[1,]`, "{\"a\":1}\x00", "\x80", "{\"\xff\":1}"} {
		addRaw("raw "+strconv.QuoteToASCII(s[:min(len(s), 24)]), s)
	}
	return out
}

func paletteCases() []caseJSON {
	var out []caseJSON
	add := func(note, text string) {
		f := baseFixture()
		f.palette = text
		out = append(out, f.build("separated-palette", note))
	}
	vals := paletteValues(5)
	add("valid", obj(paletteKVs(vals)))
	add("schema 2", obj(replaceKV(paletteKVs(vals), "schema_version", "2")))
	add("kind wrong", obj(replaceKV(paletteKVs(vals), "kind", q("vga"))))
	add("767", obj(paletteKVs(vals[:767])))
	add("769", obj(paletteKVs(append(append([]string(nil), vals...), "1"))))
	for _, v := range []string{"-1", "64", "63", "0", "1.5", `"1"`, "null", "true", "[]", "1e1", "-0"} {
		mod := append([]string(nil), vals...)
		mod[100] = v
		add("component="+v, obj(paletteKVs(mod)))
	}
	add("components null", obj(replaceKV(paletteKVs(vals), "dac_6bit_components", "null")))
	add("components object", obj(replaceKV(paletteKVs(vals), "dac_6bit_components", "{}")))
	add("components string", obj(replaceKV(paletteKVs(vals), "dac_6bit_components", `"x"`)))
	add("key folded", obj(renameKV(paletteKVs(vals), "dac_6bit_components", q("DAC_6bit_Components"))))
	allNull := strings.Repeat("null,", 767) + "null"
	add("stale components", obj(append(paletteKVs(vals), kv{q("dac_6bit_components"), "[5]"},
		kv{q("dac_6bit_components"), "[" + allNull + "]"})))
	add("dup kind", obj(append(paletteKVs(vals), kv{q("kind"), q("x")})))
	add("top-level array", arr(vals))
	add("syntax", `{"schema_version":1,`)
	add("empty", "")
	f := baseFixture()
	f.palette = ""
	out = append(out, f.build("separated-palette", "palette missing"))
	f = baseFixture()
	f.palette = ""
	f.extra = []fileJSON{{Path: "ANI_004/p.json", Dir: true}}
	out = append(out, f.build("separated-palette", "palette is a directory"))
	return out
}

func filesystemCases() []caseJSON {
	var out []caseJSON
	add := func(note string, f fixture) { out = append(out, f.build("separated-fs", note)) }
	f := baseFixture()
	f.resources = []int{-1, 0, 1, 4, 8, 9, 10, 100}
	add("resource sweep", f)
	f = baseFixture()
	f.emptyRoot = true
	f.resources = []int{0, 4}
	add("empty root", f)
	f = baseFixture()
	f.root = "$ROOT/sub/.."
	add("root with sub/..", f)
	f = baseFixture()
	f.root = "$ROOT/"
	add("root trailing slash", f)
	f = baseFixture()
	f.root = "$ROOT/./"
	add("root dot", f)
	f = baseFixture()
	f.root = "$ROOT/missing"
	add("root missing", f)
	f = baseFixture()
	f.root = "$ROOT/file.txt"
	f.extra = []fileJSON{hexFile("file.txt", []byte("x"))}
	add("root is a file", f)
	add("animation.json missing", fixture{res: 4, noMeta: true, extra: []fileJSON{{Path: "ANI_004", Dir: true}}})
	add("ANI_004 missing", fixture{res: 4, noMeta: true})
	add("ANI_004 missing, root a/b/../..", fixture{res: 4, noMeta: true, root: "$ROOT/a/b/../.."})
	add("ANI_004 missing, root ./a/../", fixture{res: 4, noMeta: true, root: "$ROOT/./a/..//"})
	add("animation.json is a directory", fixture{res: 4, noMeta: true, extra: []fileJSON{{Path: "ANI_004/animation.json", Dir: true}}})
	nameCase := func(note, frame, palette string, extra ...fileJSON) {
		f := baseFixture()
		f.animation = obj(replaceKV(animationKVs(fixtureRes, "t"), "frames", framesArr(fixtureRes, frame, palette)))
		f.extra = extra
		add(note, f)
	}
	for _, name := range []string{"", ".", "..", "/", "//", "a/b", "v.png/", "\\v.png", "v.png\x00", strings.Repeat("n", 300),
		"missing.png", " v.png", "ANI_004"} {
		nameCase("frame name "+q(name)[:min(len(q(name)), 24)], name, "p.json")
	}
	for _, name := range []string{"", ".", "..", "/", "a/b", "missing.json", "p.json\x00"} {
		nameCase("palette name "+q(name)[:min(len(q(name)), 24)], "v.png", name)
	}
	nameCase("frame is a directory", "d", "p.json", fileJSON{Path: "ANI_004/d", Dir: true})
	f = baseFixture()
	f.png = nil
	add("png missing", f)
	f = baseFixture()
	f.png = []byte{}
	add("png empty", f)
	return out
}

// pngVariants 產生結構化 PNG 案例；每個都以 v.png 放進 fixture。
func pngVariants(rng *rand.Rand) []caseJSON {
	var out []caseJSON
	addBytes := func(note string, png []byte) {
		f := baseFixture()
		f.png = png
		out = append(out, f.build("separated-png", note))
	}
	add := func(note string, chunks []chunk) { addBytes(note, serialize(chunks)) }
	spec := func(note string, s pngSpec) { add(note, pngChunks(rng, s)) }

	spec("p8 baseline", p8(320, 200))
	for ft := 0; ft < 5; ft++ {
		s := p8(320, 200)
		ft := ft
		s.filter = func(int) int { return ft }
		s.pattern = ft*11 + 2
		spec(fmt.Sprintf("p8 filter %d", ft), s)
	}
	s := p8(320, 200)
	s.filter = func(int) int { return -1 }
	spec("p8 random filters", s)
	for _, level := range []int{0, 1, 9, -2} {
		s := p8(320, 200)
		s.level = level
		s.pattern = 4
		spec(fmt.Sprintf("p8 level %d", level), s)
	}
	for _, split := range [][]int{{1}, {7}, {100, 0}, {4096}, {4097}, {4095, 1}, {8192}, {0, 3000}} {
		s := p8(320, 200)
		s.split = split
		spec(fmt.Sprintf("p8 split %v", split), s)
	}
	s = p8(320, 200)
	s.flushEvery = 1000
	spec("p8 sync flush blocks", s)
	s = p8(320, 200)
	s.interlace = 1
	s.filter = func(int) int { return -1 }
	spec("p8 adam7", s)
	for _, depth := range []int{1, 2, 4} {
		for _, il := range []int{0, 1} {
			s := pngSpec{w: 320, h: 200, depth: depth, ct: 3, interlace: il, level: 6, pattern: depth, filter: func(int) int { return -1 }}
			spec(fmt.Sprintf("p%d interlace %d", depth, il), s)
		}
	}
	s = p8(320, 200)
	s.plte = 3
	spec("p8 short palette, indices beyond", s)
	for _, ct := range [][2]int{{8, 0}, {1, 0}, {8, 2}, {8, 4}, {8, 6}, {16, 0}, {16, 2}, {16, 4}, {16, 6}, {4, 0}, {2, 0}} {
		s := pngSpec{w: 320, h: 200, depth: ct[0], ct: ct[1], level: 6, pattern: 2, filter: func(int) int { return -1 }}
		spec(fmt.Sprintf("non-paletted depth %d ct %d", ct[0], ct[1]), s)
	}
	s = pngSpec{w: 320, h: 200, depth: 8, ct: 2, level: 6, interlace: 1}
	spec("rgb adam7", s)
	for _, wh := range [][2]int{{319, 200}, {320, 201}, {1, 1}, {640, 400}, {200, 320}, {3, 9}} {
		spec(fmt.Sprintf("p8 %dx%d", wh[0], wh[1]), p8(wh[0], wh[1]))
	}
	s = pngSpec{w: 5, h: 3, depth: 8, ct: 3, interlace: 1, level: 6}
	spec("p8 5x3 adam7 empty passes", s)

	good := pngChunks(rng, p8(320, 200))
	gray := pngChunks(rng, pngSpec{w: 320, h: 200, depth: 8, ct: 0, level: 6})
	rgb := pngChunks(rng, pngSpec{w: 320, h: 200, depth: 8, ct: 2, level: 6})
	idat := findChunk(good, "IDAT")
	text := chunk{"tEXt", []byte("Comment\x00hello")}
	add("ancillary before PLTE", withChunks(good, func(c []chunk) []chunk { return insertAt(c, 1, text) }))
	add("ancillary after IDAT", withChunks(good, func(c []chunk) []chunk { return insertAt(c, idat+1, text) }))
	add("unknown critical chunk", withChunks(good, func(c []chunk) []chunk { return insertAt(c, 2, chunk{"ABCD", []byte{1}}) }))
	add("chunk before IHDR", withChunks(good, func(c []chunk) []chunk { return insertAt(c, 0, text) }))
	add("tRNS after PLTE", withChunks(good, func(c []chunk) []chunk { return insertAt(c, 2, chunk{"tRNS", []byte{0, 128, 255}}) }))
	add("tRNS 256", withChunks(good, func(c []chunk) []chunk { return insertAt(c, 2, chunk{"tRNS", make([]byte, 256)}) }))
	add("tRNS 257", withChunks(good, func(c []chunk) []chunk { return insertAt(c, 2, chunk{"tRNS", make([]byte, 257)}) }))
	add("tRNS before PLTE", withChunks(good, func(c []chunk) []chunk { return insertAt(c, 1, chunk{"tRNS", []byte{1}}) }))
	add("tRNS twice", withChunks(good, func(c []chunk) []chunk {
		return insertAt(c, 2, chunk{"tRNS", []byte{1}}, chunk{"tRNS", []byte{1}})
	}))
	add("gray tRNS 2", withChunks(gray, func(c []chunk) []chunk { return insertAt(c, 1, chunk{"tRNS", []byte{0, 7}}) }))
	add("gray tRNS 3", withChunks(gray, func(c []chunk) []chunk { return insertAt(c, 1, chunk{"tRNS", []byte{0, 7, 1}}) }))
	add("gray PLTE", withChunks(gray, func(c []chunk) []chunk { return insertAt(c, 1, plteChunk(4)) }))
	add("rgb PLTE ignored", withChunks(rgb, func(c []chunk) []chunk { return insertAt(c, 1, plteChunk(16)) }))
	add("rgb tRNS 6", withChunks(rgb, func(c []chunk) []chunk { return insertAt(c, 1, chunk{"tRNS", make([]byte, 6)}) }))
	add("rgb tRNS 5", withChunks(rgb, func(c []chunk) []chunk { return insertAt(c, 1, chunk{"tRNS", make([]byte, 5)}) }))
	ga := pngChunks(rng, pngSpec{w: 8, h: 8, depth: 8, ct: 4, level: 6})
	add("gray+alpha tRNS", withChunks(ga, func(c []chunk) []chunk { return insertAt(c, 1, chunk{"tRNS", []byte{0, 1}}) }))
	add("IDAT before PLTE", withChunks(good, func(c []chunk) []chunk {
		c[1], c[2] = c[2], c[1]
		return c
	}))
	add("no PLTE", withChunks(good, func(c []chunk) []chunk { return append(c[:1:1], c[2:]...) }))
	add("PLTE after IDAT", withChunks(good, func(c []chunk) []chunk { return insertAt(c, idat+1, plteChunk(4)) }))
	add("two IHDR", withChunks(good, func(c []chunk) []chunk { return insertAt(c, 1, c[0]) }))
	add("IEND first", withChunks(good, func(c []chunk) []chunk { return insertAt(c, 0, chunk{"IEND", nil}) }))
	add("no IEND", good[:len(good)-1])
	add("IEND length 1", withChunks(good, func(c []chunk) []chunk { c[len(c)-1] = chunk{"IEND", []byte{0}}; return c }))
	add("trailing IDAT garbage", withChunks(good, func(c []chunk) []chunk {
		return insertAt(c, idat+1, chunk{"IDAT", []byte("garbage")}, chunk{"IDAT", nil})
	}))
	add("data after IEND", nil)
	out[len(out)-1].Files[1] = hexFile("ANI_004/v.png", append(serialize(good), []byte("trailing junk")...))
	ihdrMod := func(note string, f func([]byte)) {
		add(note, withChunks(good, func(c []chunk) []chunk {
			d := append([]byte(nil), c[0].data...)
			f(d)
			c[0] = chunk{"IHDR", d}
			return c
		}))
	}
	ihdrMod("IHDR compression 1", func(d []byte) { d[10] = 1 })
	ihdrMod("IHDR filter 1", func(d []byte) { d[11] = 1 })
	ihdrMod("IHDR interlace 2", func(d []byte) { d[12] = 2 })
	ihdrMod("IHDR width 0", func(d []byte) { binary.BigEndian.PutUint32(d, 0) })
	ihdrMod("IHDR width negative", func(d []byte) { binary.BigEndian.PutUint32(d, 0x80000000) })
	ihdrMod("IHDR depth 3", func(d []byte) { d[8] = 3 })
	ihdrMod("IHDR color type 5", func(d []byte) { d[9] = 5 })
	ihdrMod("IHDR 16-bit paletted", func(d []byte) { d[8] = 16 })
	ihdrMod("IHDR claims 2 rows more", func(d []byte) { binary.BigEndian.PutUint32(d[4:], 202) })
	ihdrMod("IHDR claims fewer rows", func(d []byte) { binary.BigEndian.PutUint32(d[4:], 199) })
	ihdrMod("IHDR huge overflow", func(d []byte) {
		binary.BigEndian.PutUint32(d, 0x7fffffff)
		binary.BigEndian.PutUint32(d[4:], 0x7fffffff)
	})
	add("IHDR length 12", withChunks(good, func(c []chunk) []chunk { c[0] = chunk{"IHDR", c[0].data[:12]}; return c }))
	for _, n := range []int{0, 1, 2, 257} {
		n := n
		add(fmt.Sprintf("PLTE %d entries raw", n), withChunks(good, func(c []chunk) []chunk {
			if n == 1 || n == 2 {
				c[1] = chunk{"PLTE", make([]byte, n)}
			} else {
				c[1] = plteChunk(n)
			}
			return c
		}))
	}
	p1 := pngChunks(rng, pngSpec{w: 320, h: 200, depth: 1, ct: 3, level: 6})
	add("p1 PLTE 3 entries", withChunks(p1, func(c []chunk) []chunk { c[1] = plteChunk(3); return c }))
	// zlib 層
	raw := rawScanlines(rng, p8(320, 200))
	zl := func(note string, z []byte) {
		add(note, withChunks(good, func(c []chunk) []chunk {
			return append(c[:idat:idat], chunk{"IDAT", z}, chunk{"IEND", nil})
		}))
	}
	ad := adler32.Checksum(raw)
	zl("zlib FDICT dictid 1", rawZlib(raw, 0x78, 0x20, []byte{0, 0, 0, 1}, ad))
	zl("zlib FDICT dictid 2", rawZlib(raw, 0x78, 0x20, []byte{0, 0, 0, 2}, ad))
	zl("zlib FDICT cut", rawZlib(nil, 0x78, 0x20, nil, 0)[:2])
	zl("zlib CM 7", rawZlib(raw, 0x77, 0, nil, ad))
	zl("zlib CINFO 8", rawZlib(raw, 0x88, 0, nil, ad))
	zl("zlib FCHECK bad", append([]byte{0x78, 0x9d}, rawZlib(raw, 0x78, 0, nil, ad)[2:]...))
	zl("zlib adler wrong", rawZlib(raw, 0x78, 0, nil, ad^1))
	zl("zlib adler cut", rawZlib(raw, 0x78, 0, nil, ad)[:len(rawZlib(raw, 0x78, 0, nil, ad))-2])
	zl("zlib header only", []byte{0x78, 0x9c})
	zl("zlib one byte", []byte{0x78})
	zl("empty IDAT", nil)
	full := zlibBytes(raw, 6, 0)
	zl("zlib cut mid stream", full[:len(full)/2])
	zl("deflate block type 3", append([]byte{0x78, 0x9c, 0x07}, make([]byte, 10)...))
	zl("stored LEN/NLEN mismatch", []byte{0x78, 0x01, 0x01, 0x05, 0x00, 0xfb, 0xfe, 1, 2, 3, 4, 5})
	for _, extra := range []int{1, 7} {
		s := p8(320, 200)
		s.extraRows = extra
		spec(fmt.Sprintf("too much pixel data +%d", extra), s)
	}
	s = p8(320, 200)
	s.shortBytes = 1
	spec("not enough pixel data -1", s)
	s = p8(320, 200)
	s.badFilterRow = 17
	spec("bad filter type row 17", s)
	for _, junk := range []int{1, 2, 3, 4, 5, 100, 4095, 4096, 5000} {
		for _, split := range [][]int{nil, {3000}, {4096}} {
			s := p8(320, 200)
			s.split = split
			c := pngChunks(rng, s)
			last := findChunk(c, "IEND") - 1
			c[last] = chunk{"IDAT", append(append([]byte(nil), c[last].data...), bytes.Repeat([]byte{0xa5}, junk)...)}
			add(fmt.Sprintf("junk %d after zlib split %v", junk, split), c)
		}
	}
	// 位元組層
	gb := serialize(good)
	addBytes("not png", []byte("GIF89a......"))
	addBytes("7 bytes of signature", gb[:7])
	addBytes("signature only", gb[:8])
	addBytes("signature then garbage", append(append([]byte(nil), gb[:8]...), []byte("garbage!")...))
	for _, cut := range []int{9, 16, 20, 33, 40, 50, 900, len(gb) / 2, len(gb) - 13, len(gb) - 12, len(gb) - 1} {
		addBytes(fmt.Sprintf("truncated at %d", cut), gb[:cut])
	}
	for _, ci := range []int{0, 1, idat, len(good) - 1} {
		b := serialize(good)
		pos := 8
		for k := 0; k < ci; k++ {
			pos += 12 + len(good[k].data)
		}
		b[pos+8+len(good[ci].data)+3] ^= 0x40
		addBytes(fmt.Sprintf("bad CRC on %s", good[ci].typ), b)
	}
	unk := serialize(withChunks(good, func(c []chunk) []chunk { return insertAt(c, 1, chunk{"zzZZ", []byte{1, 2}}) }))
	binary.BigEndian.PutUint32(unk[33:], 0x80000000)
	addBytes("unknown chunk length > 2^31", unk)
	idatHuge := serialize(good)
	pos := 8
	for k := 0; k < idat; k++ {
		pos += 12 + len(good[k].data)
	}
	binary.BigEndian.PutUint32(idatHuge[pos:], 0xfffffff0)
	addBytes("IDAT length huge", idatHuge)
	return out
}

var pngPool [][]byte

func randomPNGCase(rng *rand.Rand) caseJSON {
	base := pngPool[rng.IntN(len(pngPool))]
	b := append([]byte(nil), base...)
	var notes []string
	for k := 0; k < 1+rng.IntN(3); k++ {
		switch op := rng.IntN(10); {
		case op < 7:
			at := 8 + rng.IntN(len(b)-8)
			b[at] ^= byte(1 + rng.IntN(255))
			notes = append(notes, fmt.Sprintf("xor@%d", at))
		case op < 8:
			n := 8 + rng.IntN(len(b)-8)
			b = b[:n]
			notes = append(notes, fmt.Sprintf("cut@%d", n))
		case op < 9:
			at := 8 + rng.IntN(len(b)-8)
			ins := make([]byte, 1+rng.IntN(8))
			for i := range ins {
				ins[i] = byte(rng.IntN(256))
			}
			b = append(b[:at:at], append(ins, b[at:]...)...)
			notes = append(notes, fmt.Sprintf("ins@%d", at))
		default:
			at := 8 + rng.IntN(len(b)-8)
			n := min(1+rng.IntN(16), len(b)-at)
			b = append(b[:at:at], b[at+n:]...)
			notes = append(notes, fmt.Sprintf("del@%d", at))
		}
	}
	if rng.IntN(10) < 7 {
		fixCRC(b)
		notes = append(notes, "fixcrc")
	}
	f := baseFixture()
	f.png = b
	return f.build("separated-png-random", strings.Join(notes, ","))
}

// ---------- 原版檔 ----------

func listReal(dir string, filter func(rel string) bool) []realFile {
	var out []realFile
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if !filter(rel) {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out = append(out, realFile{Path: rel, Size: len(b), SHA256: digest(b)})
		return nil
	})
	if err != nil {
		die(err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func derivedArchiveCases(rng *rand.Rand, ani []byte) []caseJSON {
	var out []caseJSON
	first := int(binary.LittleEndian.Uint32(ani[6:]))
	n := (first - 6) / 4
	size := func(res int) int { return len(extractResource(ani, res)) }
	add := func(note string, ops []opJSON) {
		out = append(out, caseJSON{Kind: "afm-derived", Note: note, Call: "archive", Path: "$ROOT/x.dat",
			Resources: []int{-1, 0, 1}, Files: []fileJSON{{Path: "x.dat", From: &fromJSON{Root: "org", Path: "ANI.DAT", Ops: ops}}}})
	}
	for res := 0; res < n; res++ {
		add(fmt.Sprintf("resource %d rewrapped", res), []opJSON{{Op: "extract", N: res}, {Op: "wrap"}})
	}
	for _, res := range []int{8, 4, 6} {
		sz := size(res)
		for _, cut := range []int{1, 7, 100, sz - 0xad, sz - 0xac} {
			add(fmt.Sprintf("resource %d minus %d", res, cut), []opJSON{{Op: "extract", N: res}, {Op: "truncate", N: sz - cut}, {Op: "wrap"}})
		}
		add(fmt.Sprintf("resource %d plus byte", res), []opJSON{{Op: "extract", N: res}, {Op: "insert", At: sz, Hex: "00"}, {Op: "wrap"}})
		add(fmt.Sprintf("resource %d frames-1", res), []opJSON{{Op: "extract", N: res}, {Op: "set16", At: 0xa5, Val: expectedFrameCounts[res] - 1}, {Op: "wrap"}})
		add(fmt.Sprintf("resource %d frames+1", res), []opJSON{{Op: "extract", N: res}, {Op: "set16", At: 0xa5, Val: expectedFrameCounts[res] + 1}, {Op: "wrap"}})
		add(fmt.Sprintf("resource %d width 321", res), []opJSON{{Op: "extract", N: res}, {Op: "set16", At: 0xa7, Val: 321}, {Op: "wrap"}})
		for k := 0; k < 6; k++ {
			at := 0xad + rng.IntN(sz-0xad)
			add(fmt.Sprintf("resource %d xor@%d", res, at), []opJSON{{Op: "extract", N: res}, {Op: "xor", At: at, Val: 1 + rng.IntN(255)}, {Op: "wrap"}})
		}
	}
	return out
}

func derivedSeparatedCases(rng *rand.Rand, anim []realFile) []caseJSON {
	var out []caseJSON
	dir := fmt.Sprintf("ANI_%03d", fixtureRes)
	var files []realFile
	for _, f := range anim {
		if strings.HasPrefix(f.Path, dir+"/") {
			files = append(files, f)
		}
	}
	copyDir := func(note string, mutate string, ops []opJSON) {
		var fl []fileJSON
		for _, f := range files {
			from := &fromJSON{Root: "anim", Path: f.Path, Ops: []opJSON{}}
			if f.Path == mutate {
				from.Ops = ops
			}
			fl = append(fl, fileJSON{Path: f.Path, From: from})
		}
		out = append(out, caseJSON{Kind: "separated-derived", Note: note, Call: "separated", Path: "$ROOT",
			Resources: []int{fixtureRes}, Files: fl})
	}
	copyDir("real ANI_004 copy", "", nil)
	sizeOf := func(path string) int {
		for _, f := range files {
			if f.Path == path {
				return f.Size
			}
		}
		die(fmt.Errorf("找不到 %s", path))
		return 0
	}
	for k := 0; k < 40; k++ {
		frame := fmt.Sprintf("%s/frame_%03d.png", dir, rng.IntN(12))
		sz := sizeOf(frame)
		var ops []opJSON
		switch rng.IntN(4) {
		case 0, 1:
			ops = []opJSON{{Op: "xor", At: 8 + rng.IntN(sz-8), Val: 1 + rng.IntN(255)}, {Op: "fixcrc"}}
		case 2:
			ops = []opJSON{{Op: "truncate", N: 8 + rng.IntN(sz-8)}}
		default:
			ops = []opJSON{{Op: "xor", At: 8 + rng.IntN(sz-8), Val: 1 + rng.IntN(255)}}
		}
		copyDir(fmt.Sprintf("real %s %v", frame, ops), frame, ops)
	}
	meta := dir + "/animation.json"
	msz := sizeOf(meta)
	for k := 0; k < 8; k++ {
		at := rng.IntN(msz)
		copyDir(fmt.Sprintf("real animation.json xor@%d", at), meta, []opJSON{{Op: "xor", At: at, Val: 1 + rng.IntN(255)}})
	}
	pal := dir + "/palette_007.json"
	psz := sizeOf(pal)
	for k := 0; k < 6; k++ {
		at := rng.IntN(psz)
		copyDir(fmt.Sprintf("real palette_007.json xor@%d", at), pal, []opJSON{{Op: "xor", At: at, Val: 1 + rng.IntN(255)}})
	}
	return out
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	orgDir := flag.String("org", "", "含 ANI.DAT 的原版目錄")
	animDir := flag.String("animations", "", "分離動畫目錄（含 ANI_000…ANI_008）")
	cases := flag.Int("cases", 320, "隨機 AFM 容器案例數")
	pngCases := flag.Int("png-cases", 320, "隨機 PNG 變形案例數")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	flag.Parse()
	if *out == "" || *orgDir == "" || *animDir == "" {
		fmt.Fprintln(os.Stderr, "需要 -out、-org 與 -animations")
		os.Exit(2)
	}
	scratch, err := os.MkdirTemp("", "fd2-golden-afm-")
	if err != nil {
		die(err)
	}
	defer os.RemoveAll(scratch)
	r := &runner{roots: map[string]string{"org": filepath.Clean(*orgDir), "anim": filepath.Clean(*animDir)}, scratch: scratch}
	rng := rand.New(rand.NewPCG(*seed, 0x41464d31))

	realOrg := listReal(r.roots["org"], func(rel string) bool { return rel == "ANI.DAT" })
	realAnim := listReal(r.roots["anim"], func(rel string) bool {
		return len(rel) > 8 && rel[:4] == "ANI_" && rel[7] == '/'
	})
	if len(realOrg) != 1 || len(realAnim) == 0 {
		die(fmt.Errorf("原版檔不完整：org=%d anim=%d", len(realOrg), len(realAnim)))
	}
	ani, err := os.ReadFile(filepath.Join(r.roots["org"], "ANI.DAT"))
	if err != nil {
		die(err)
	}

	validPNG = serialize(pngChunks(rng, p8(320, 200)))
	pngPool = [][]byte{
		validPNG,
		serialize(pngChunks(rng, pngSpec{w: 9, h: 7, depth: 8, ct: 3, level: 6, filter: func(int) int { return -1 }})),
		serialize(pngChunks(rng, pngSpec{w: 13, h: 5, depth: 1, ct: 3, interlace: 1, level: 6})),
		serialize(pngChunks(rng, pngSpec{w: 6, h: 6, depth: 8, ct: 2, level: 6, filter: func(int) int { return -1 }})),
		serialize(pngChunks(rng, pngSpec{w: 320, h: 200, depth: 4, ct: 3, level: 1, split: []int{900}})),
		serialize(pngChunks(rng, pngSpec{w: 320, h: 200, depth: 8, ct: 3, level: 0, split: []int{5000}})),
		serialize(pngChunks(rng, pngSpec{w: 17, h: 3, depth: 16, ct: 6, level: 9, interlace: 1})),
		serialize(withChunks(pngChunks(rng, p8(320, 200)), func(c []chunk) []chunk {
			return insertAt(c, 1, chunk{"tEXt", []byte("k\x00v")})
		})),
	}

	var rows []caseJSON
	for i := 0; i < *cases; i++ {
		rows = append(rows, randomArchiveCase(rng, i))
	}
	rows = append(rows, structuredArchiveCases(rng)...)
	rows = append(rows, caseJSON{Kind: "afm-real", Note: "ANI.DAT", Call: "archive", Path: "$ORG/ANI.DAT", Resources: span(-1, 10)})
	rows = append(rows, archiveCase("afm-structured", "missing file", nil, span(0, 0)))
	rows[len(rows)-1].Files, rows[len(rows)-1].Path = nil, "$ROOT/missing.dat"
	rows = append(rows, caseJSON{Kind: "afm-structured", Note: "path is a directory", Call: "archive", Path: "$ROOT/d",
		Resources: span(0, 0), Files: []fileJSON{{Path: "d", Dir: true}}})
	rows = append(rows, derivedArchiveCases(rng, ani)...)
	rows = append(rows, metadataCases()...)
	rows = append(rows, paletteCases()...)
	rows = append(rows, filesystemCases()...)
	rows = append(rows, pngVariants(rng)...)
	for i := 0; i < *pngCases; i++ {
		rows = append(rows, randomPNGCase(rng))
	}
	rows = append(rows, derivedSeparatedCases(rng, realAnim)...)
	rows = append(rows, caseJSON{Kind: "separated-real", Note: "ANIM", Call: "separated", Path: "$ANIM", Resources: span(-1, 10)})
	rows = append(rows, caseJSON{Kind: "separated-real", Note: "ANIM/", Call: "separated", Path: "$ANIM/", Resources: span(0, 0)})

	for i := range rows {
		if rows[i].Files == nil {
			rows[i].Files = []fileJSON{}
		}
		r.run(i, &rows[i])
	}
	doc := map[string]any{
		"schema": "fd2-golden-afm/1", "seed": *seed,
		"real":  map[string]any{"org": realOrg, "anim": realAnim},
		"cases": rows,
	}
	data, err := json.Marshal(doc)
	if err != nil {
		die(err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		die(err)
	}
	fmt.Printf("fd2-golden-afm: %d cases sha256=%s\n", len(rows), digest(data))
}
