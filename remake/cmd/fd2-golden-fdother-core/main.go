// fd2-golden-fdother-core 輸出 internal/fdother 核心與分離載入器（第一組）的黃金差分資料，
// 供 Rust 改寫逐欄比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 fdother、fdicon 的公開 API，不改任何遊戲行為。
// 隨機與結構化輸入以固定種子的 PCG 產生，連同結果寫進同一份 JSON，Rust 端不需重現亂數。
// 真實原版檔與分離素材包不嵌入 JSON，只記路徑、大小與 SHA-256；分離載入器的變異樹以
// 「連到真實素材包的符號連結＋改寫過的檔案」描述，改寫內容以十六進位存進 blobs。
//
// 每個案例的 compare 欄決定 Rust 端的比對方式：
//   - exact：結果逐欄相同，錯誤文字逐字相同。
//   - err_prefix：Go 錯誤包了標準函式庫錯誤（os、encoding/json、image/png）；Rust 必須同樣
//     拒收，而且 Go 自己的前綴（prefix 欄）相同，內層文字不比。
//   - accept_reject：格式錯誤的 PNG／JSON，只比接受或拒收；都接受時結果仍逐欄比對。
package main

import (
	"crypto/sha256"
	"encoding/binary"
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

type treeSpec struct {
	Links  [][2]string `json:"links,omitempty"`  // 真實素材包相對路徑 -> 樹內相對路徑（遞迴鏡像為符號連結）
	Remove []string    `json:"remove,omitempty"` // 樹內路徑
	Dirs   []string    `json:"dirs,omitempty"`
	Files  [][2]string `json:"files,omitempty"` // 樹內路徑 -> blob
}

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
	malformed bool // 目前產生的案例是否屬於格式錯誤輸入（accept_reject）
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

// 本套件自己的、不帶 fdother: 前綴的葉錯誤文字（由 %w 包進去）。
var ownLeaves = map[string]bool{
	"indexed PNG mode or geometry mismatch":              true,
	"mask PNG mode or geometry mismatch":                 true,
	"mask PNG is not binary":                             true,
	"indexed PNG must be 24x24 with 256 palette entries": true,
	"mask PNG must be 24x24 grayscale":                   true,
	"mask PNG contains a non-binary value":               true,
	"codec stream ends before dimensions are filled":     true,
	"repeat command lacks pixel value":                   true,
	"present failed":                                     true, // 本工具注入的回呼錯誤
	"restore failed":                                     true,
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
		if errors.Unwrap(e) == nil {
			msg := e.Error()
			if strings.HasPrefix(msg, "fdother:") || strings.HasPrefix(msg, "fdicon:") || ownLeaves[msg] {
				return "", false
			}
			return strings.TrimSuffix(full, msg), true
		}
	}
	return "", false
}

// protect 回報 Go 執行期 panic（越界索引、除以零等）。
func protect(f func() obj) (res obj) {
	defer func() {
		if recover() != nil {
			res = obj{"err": nil, "panic": true}
		}
	}()
	res = f()
	if res == nil {
		res = obj{}
	}
	res["panic"] = false
	return res
}

func (e *exporter) add(kind, op string, input obj, result obj, err error) {
	compare := "exact"
	var prefix *string
	if e.malformed {
		compare = "accept_reject"
	} else if err != nil {
		if p, ok := goPrefix(err); ok {
			compare, prefix = "err_prefix", &p
		}
	}
	if result == nil {
		result = obj{}
	}
	if s, ok := result["err"].(string); ok {
		s = strings.ReplaceAll(s, e.assets, "$REAL")
		s = strings.ReplaceAll(s, e.flame2, "$FLAME2")
		result["err"] = s
		if prefix != nil {
			p := strings.ReplaceAll(strings.ReplaceAll(*prefix, e.assets, "$REAL"), e.flame2, "$FLAME2")
			prefix = &p
		}
	}
	e.cases = append(e.cases, caseJSON{Index: len(e.cases), Kind: kind, Op: op, Compare: compare, Prefix: prefix, Input: input, Result: result})
}

var extremes = []int{0, -1, 1, math.MaxInt64, math.MinInt64, math.MaxInt64 - 10, math.MinInt64 + 10,
	1 << 62, -(1 << 62), 0x7fffffff, 1 << 32, -320, 320, 456, 0x7fff, 0x8000}

func (e *exporter) pick(lo, hi int) int {
	if e.rng.IntN(20) == 0 {
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

// pattern 是 Rust 端可重建的目的端初始內容。
func pattern(n, seed int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*13 + seed)
	}
	return b
}

func lutOf(n, mul, add int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i*mul + add)
	}
	return b
}

// ---- 結果摘要 ----

const layerLimit = 1 << 22

func frameInfo(f fdother.Frame) obj {
	out := obj{"x": f.X, "y": f.Y, "w": f.Width, "h": f.Height,
		"pixels": digest(f.Pixels), "indexed": digest(f.Indexed), "mask": digest(f.Mask)}
	if f.Width > 0 && f.Height > 0 && f.Width*f.Height <= layerLimit {
		r := protect(func() obj {
			ix, mk, err := f.IndexedLayers()
			o := obj{"err": errv(err)}
			if err == nil {
				o["sha"] = digest(append(append([]byte(nil), ix...), mk...))
			}
			return o
		})
		out["layers"] = r
	}
	return out
}

func lmiInfo(en fdother.LMI1Entry) []any { return []any{en.Width, en.Height, digest(en.Pixels)} }
func rawInfo(c fdother.RawCell) []any    { return []any{c.Width, c.Height, digest(c.Pixels)} }

func spriteDigest(s fdicon.Sprite) string {
	h := sha256.New()
	h.Write(s.Pixels)
	h.Write(s.Mask)
	h.Write(s.RemapMask)
	return hex.EncodeToString(h.Sum(nil))
}

func bankInfo(b *fdicon.Bank) []string {
	ds := make([]string, len(b.Sprites))
	for i, s := range b.Sprites {
		ds[i] = spriteDigest(s)
	}
	return ds
}

// protectErr 同 protect，另外傳回 Go 的錯誤值（決定 compare 方式用）。
func protectErr(f func() (obj, error)) (res obj, err error) {
	defer func() {
		if recover() != nil {
			res, err = obj{"err": nil, "panic": true}, nil
		}
	}()
	res, err = f()
	res["err"] = errv(err)
	res["panic"] = false
	return res, err
}

// ---- 原始資料解析（隨機與真實共用） ----

// parseOp 對一份原始資料執行指定的解析器。
func parseOp(op string, data []byte, index int) (obj, error) {
	var err error
	out := obj{}
	switch op {
	case "parse_frames":
		var frames []fdother.Frame
		frames, err = fdother.ParseFrames(data)
		if err == nil {
			infos := make([]obj, len(frames))
			for i, f := range frames {
				infos[i] = frameInfo(f)
			}
			out["frames"] = infos
		}
	case "parse_single_frame", "parse_lmi1_frame_entry":
		var f fdother.Frame
		if op == "parse_single_frame" {
			f, err = fdother.ParseSingleFrame(data)
		} else {
			f, err = fdother.ParseLMI1FrameEntry(data, index)
		}
		if err == nil {
			out["frame"] = frameInfo(f)
		}
	case "parse_lmi1":
		var entries []fdother.LMI1Entry
		entries, err = fdother.ParseLMI1(data)
		if err == nil {
			infos := make([][]any, len(entries))
			for i, en := range entries {
				infos[i] = lmiInfo(en)
			}
			out["entries"] = infos
		}
	case "parse_opaque_run_cell", "parse_lmi1_opaque_entry":
		var en fdother.LMI1Entry
		if op == "parse_opaque_run_cell" {
			en, err = fdother.ParseOpaqueRunCell(data)
		} else {
			en, err = fdother.ParseLMI1OpaqueEntry(data, index)
		}
		if err == nil {
			out["entry"] = lmiInfo(en)
		}
	case "parse_raw_cell", "parse_lmi1_raw_entry":
		var c fdother.RawCell
		if op == "parse_raw_cell" {
			c, err = fdother.ParseRawCell(data)
		} else {
			c, err = fdother.ParseLMI1RawEntry(data, index)
		}
		if err == nil {
			out["cell"] = rawInfo(c)
		}
	case "parse_raw_cell_bank":
		var cells []fdother.RawCell
		cells, err = fdother.ParseRawCellBank(data)
		if err == nil {
			infos := make([][]any, len(cells))
			for i, c := range cells {
				infos[i] = rawInfo(c)
			}
			out["cells"] = infos
		}
	case "parse_lut_bank":
		var luts [][]byte
		luts, err = fdother.ParseLUTBank(data)
		if err == nil {
			ds := make([]string, len(luts))
			for i, l := range luts {
				ds[i] = digest(l)
			}
			out["luts"] = ds
			if b, berr := fdother.NativeUnitPresentBridgeLUT(luts); berr == nil {
				out["bridge"] = digest(b)
			} else {
				out["bridge_err"] = berr.Error()
			}
		}
	case "parse_vga_palette":
		var p []byte
		pal, perr := fdother.ParseVGAPalette(data)
		err = perr
		if err == nil {
			for _, c := range pal {
				r, g, b, a := c.RGBA()
				p = binary.LittleEndian.AppendUint32(p, r)
				p = binary.LittleEndian.AppendUint32(p, g)
				p = binary.LittleEndian.AppendUint32(p, b)
				p = binary.LittleEndian.AppendUint32(p, a)
			}
			out["palette"] = digest(p)
		}
	default:
		panic(op)
	}
	out["err"] = errv(err)
	return out, err
}

// headerArea 回傳資料開頭 u16 寬高的乘積；用來避開會讓 Go 配置數 GB 的解析。
func headerArea(data []byte) int {
	if len(data) < 4 {
		return 0
	}
	return int(binary.LittleEndian.Uint16(data)) * int(binary.LittleEndian.Uint16(data[2:]))
}

func isLMI1(data []byte) bool { return len(data) >= 6 && string(data[:4]) == "LMI1" }

// lmiAreaOK：LMI1 第 index 筆的寬高乘積不超過上限（或根本讀不到標頭）。FDOTHER #3 是同樣
// 帶 LMI1 檔頭的 LUT 庫，把它當 UI 圖格解會讓 Go 先 make 數 GB 的容量；兩端都以同一規則略過。
func lmiAreaOK(data []byte, index int) bool {
	if !isLMI1(data) {
		return true
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	if index < 0 || index >= count || 6+4*count > len(data) {
		return true
	}
	off := int(binary.LittleEndian.Uint32(data[6+4*index:]))
	if off+4 > len(data) {
		return true
	}
	return headerArea(data[off:]) <= layerLimit
}

func lmiAllOK(data []byte) bool {
	if !isLMI1(data) {
		return true
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	for i := 0; i < count; i++ {
		if !lmiAreaOK(data, i) {
			return false
		}
	}
	return true
}

func (e *exporter) parseAll(kind string, input obj, data []byte) {
	for _, op := range []string{"parse_frames", "parse_single_frame", "parse_lmi1", "parse_raw_cell_bank", "parse_lut_bank", "parse_vga_palette"} {
		if op == "parse_lmi1" && !lmiAllOK(data) {
			continue
		}
		in := copyObj(input)
		in["parser"] = op
		res, err := protectErr(func() (obj, error) { return parseOp(op, data, 0) })
		e.add(kind, "parse", in, res, err)
	}
	if headerArea(data) <= layerLimit {
		for _, op := range []string{"parse_opaque_run_cell", "parse_raw_cell"} {
			in := copyObj(input)
			in["parser"] = op
			res, err := protectErr(func() (obj, error) { return parseOp(op, data, 0) })
			e.add(kind, "parse", in, res, err)
		}
	}
	if isLMI1(data) {
		count := int(binary.LittleEndian.Uint16(data[4:]))
		for index := -1; index <= count; index++ {
			for _, op := range []string{"parse_lmi1_frame_entry", "parse_lmi1_raw_entry", "parse_lmi1_opaque_entry"} {
				if op == "parse_lmi1_opaque_entry" && !lmiAreaOK(data, index) {
					continue
				}
				in := copyObj(input)
				in["parser"], in["index"] = op, index
				res, err := protectErr(func() (obj, error) { return parseOp(op, data, index) })
				e.add(kind, "parse", in, res, err)
			}
		}
	}
}

func copyObj(o obj) obj {
	out := obj{}
	for k, v := range o {
		out[k] = v
	}
	return out
}

// ---- 真實原版檔 ----

func (e *exporter) realFile(name string) []byte {
	b, err := os.ReadFile(filepath.Join(e.flame2, name))
	if err != nil {
		panic(err)
	}
	return b
}

func (e *exporter) genReal() obj {
	real := obj{}
	for _, name := range []string{"FDOTHER.DAT", "BG.DAT", "TAI.DAT", "FDSHAP.DAT"} {
		data := e.realFile(name)
		real[name] = obj{"size": len(data), "sha256": digest(data)}
		count, err := fdother.ArchiveDataResourceCount(data)
		if err != nil {
			panic(err)
		}
		for resource := 0; resource < count; resource++ {
			entry, err := fdother.ArchiveEntry(data, resource)
			if err != nil {
				panic(err)
			}
			if name == "FDSHAP.DAT" {
				continue // FDSHAP 走下面的圖庫路徑
			}
			e.parseAll("real-"+name, obj{"real": name, "resource": resource}, entry)
			if n, err := fdother.ArchiveDataResourceCount(entry); err == nil {
				for nested := 0; nested < n; nested++ {
					sub, _ := fdother.ArchiveEntry(entry, nested)
					e.parseAll("real-"+name+"-nested", obj{"real": name, "resource": resource, "nested": nested}, sub)
				}
			}
		}
		e.genRealPathOps(name, count)
	}
	return real
}

// genRealPathOps 走以路徑為參數的公開 API（含讀檔與目錄錯誤的順序）。
func (e *exporter) genRealPathOps(name string, count int) {
	kind := "real-path-" + name
	resources := []int{-1, 0, 1, 2, 3, 5, 6, 9, 13, 14, 34, 45, 54, 55, 56, 81, count - 1, count, 1 << 40}
	archive, _ := os.ReadFile(filepath.Join(e.flame2, name))
	pathOp := func(op string, file string, a, b int) {
		in := obj{"file": file, "a": a, "b": b}
		path := filepath.Join(e.flame2, file)
		if op == "decode_lmi1_resource" && file == name {
			if entry, err := fdother.ArchiveEntry(archive, a); err == nil && !lmiAllOK(entry) {
				return
			}
		}
		var err error
		res := obj{}
		switch op {
		case "decode_resource":
			var frames []fdother.Frame
			frames, err = fdother.DecodeResource(path, a)
			if err == nil {
				infos := make([]obj, len(frames))
				for i, f := range frames {
					infos[i] = frameInfo(f)
				}
				res["frames"] = infos
			}
		case "decode_archive_single_frame":
			var f fdother.Frame
			f, err = fdother.DecodeArchiveSingleFrame(path, a)
			if err == nil {
				res["frame"] = frameInfo(f)
			}
		case "decode_lmi1_resource":
			var entries []fdother.LMI1Entry
			entries, err = fdother.DecodeLMI1Resource(path, a)
			if err == nil {
				infos := make([][]any, len(entries))
				for i, en := range entries {
					infos[i] = lmiInfo(en)
				}
				res["entries"] = infos
			}
		case "decode_lmi1_frame_resource":
			var f fdother.Frame
			f, err = fdother.DecodeLMI1FrameResource(path, a, b)
			if err == nil {
				res["frame"] = frameInfo(f)
			}
		case "decode_lmi1_raw_entry":
			var c fdother.RawCell
			c, err = fdother.DecodeLMI1RawEntry(path, a, b)
			if err == nil {
				res["cell"] = rawInfo(c)
			}
		case "decode_raw_cell_resource":
			var cells []fdother.RawCell
			cells, err = fdother.DecodeRawCellResource(path, a)
			if err == nil {
				infos := make([][]any, len(cells))
				for i, c := range cells {
					infos[i] = rawInfo(c)
				}
				res["cells"] = infos
			}
		case "decode_lut_resource":
			var luts [][]byte
			luts, err = fdother.DecodeLUTResource(path, a)
			if err == nil {
				ds := make([]string, len(luts))
				for i, l := range luts {
					ds[i] = digest(l)
				}
				res["luts"] = ds
			}
		case "decode_sprite_bank_resource":
			var bank *fdicon.Bank
			bank, err = fdother.DecodeSpriteBankResource(path, a)
			if err == nil {
				res["bank"] = bankInfo(bank)
			}
		case "decode_map_terrain_resources":
			var bank *fdicon.Bank
			var controls []byte
			bank, controls, err = fdother.DecodeMapTerrainResources(path, a)
			if err == nil {
				res["bank"], res["controls"] = bankInfo(bank), digest(controls)
			}
		case "decode_native_range_overlay_bank":
			var bank *fdicon.Bank
			bank, err = fdother.DecodeNativeRangeOverlayBank(path)
			if err == nil {
				res["bank"] = bankInfo(bank)
			}
		case "decode_native_chapter_aux_surface":
			var s *fdother.NativeChapterAuxSurface
			s, err = fdother.DecodeNativeChapterAuxSurface(path)
			if err == nil {
				res["pixels"] = digest(s.Pixels)
			}
		case "read_resource":
			var raw []byte
			raw, err = fdother.ReadResource(path, a)
			if err == nil {
				res["raw"] = digest(raw)
			}
		case "read_nested_resource":
			var raw []byte
			raw, err = fdother.ReadNestedResource(path, a, b)
			if err == nil {
				res["raw"] = digest(raw)
			}
		default:
			panic(op)
		}
		res["err"] = errv(err)
		e.add(kind, op, in, res, err)
	}
	if name == "FDSHAP.DAT" {
		for r := 0; r <= count; r++ {
			pathOp("decode_sprite_bank_resource", name, r, 0)
		}
		for m := -1; m <= count/2+1; m++ {
			pathOp("decode_map_terrain_resources", name, m, 0)
		}
		pathOp("decode_map_terrain_resources", "MISSING.DAT", 0, 0)
		return
	}
	for _, r := range resources {
		for _, op := range []string{"decode_resource", "decode_archive_single_frame", "decode_lmi1_resource", "decode_raw_cell_resource", "decode_lut_resource", "read_resource"} {
			pathOp(op, name, r, 0)
		}
		for _, idx := range []int{-1, 0, 1, 0x52, 0x83, 1 << 20} {
			pathOp("decode_lmi1_frame_resource", name, r, idx)
			pathOp("decode_lmi1_raw_entry", name, r, idx)
		}
		// 巢狀索引只用非負值：負的 nested 在 Go 的錯誤訊息印成 -1，crate::archive 的
		// ArchiveError::Nested 以 usize 保存而印成 18446744073709551615（archive 模組的差異，另行回報）。
		for _, idx := range []int{0, 1, 2, 3} {
			pathOp("read_nested_resource", name, r, idx)
		}
	}
	for _, op := range []string{"decode_resource", "decode_lmi1_resource", "decode_lut_resource", "read_resource"} {
		pathOp(op, "MISSING.DAT", 0, 0)
	}
	if name == "FDOTHER.DAT" {
		pathOp("decode_native_range_overlay_bank", name, 0, 0)
		pathOp("decode_native_chapter_aux_surface", name, 0, 0)
		pathOp("decode_sprite_bank_resource", name, 1, 0)
	} else {
		pathOp("decode_native_range_overlay_bank", name, 0, 0)
		pathOp("decode_native_chapter_aux_surface", name, 0, 0)
	}
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
var packDirs = []string{"surfaces", "palette", "ui", "sfx", "effects/fdother_006_lmi1_opaque",
	"animations/fdother_009_spawn_intro", "animations/fdother_045_event61", "animations/fdother_054_ending_prefix",
	"animations/fdother_034_ch20_sky_key", "animations/fdother_079_pending_code1", "sprites/fdother_001_range_overlay"}

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
	tmp, err := os.MkdirTemp("", "fd2-golden-fdother-core-")
	must(err)
	defer os.RemoveAll(tmp)
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x46444f54)), tmp: tmp, flame2: *flame2, assets: *assets,
		blobs: map[string]string{}, trees: map[string]treeSpec{}, banks: map[string][][3]string{},
		bankObjs: map[string]*fdicon.Bank{}}
	e.genParsers()
	e.genFrameBlits()
	e.genCellBlits()
	e.genPalettes()
	e.genRemap()
	e.genRange()
	e.genSurfaces()
	e.genFlow()
	real := e.genReal()
	pack := obj{}
	for _, d := range packDirs {
		sum, n := treeDigest(filepath.Join(*assets, d))
		pack[d] = obj{"files": n, "tree_sha256": sum}
	}
	real["pack"] = pack
	e.genRealPack()
	e.genMutations()
	doc := obj{"schema": "fd2-golden-fdother-core/1", "seed": *seed, "real": real, "blobs": e.blobs, "trees": e.trees,
		"banks": e.banks, "cases": e.cases}
	data, err := json.Marshal(doc)
	must(err)
	must(os.WriteFile(*out, data, 0o644))
	ops, compares := map[string]int{}, map[string]int{}
	for _, c := range e.cases {
		ops[c.Op]++
		compares[c.Compare]++
	}
	fmt.Printf("fd2-golden-fdother-core: %d cases sha256=%s\ncompare=%v\nops=%v\n", len(e.cases), digest(data), compares, ops)
}
