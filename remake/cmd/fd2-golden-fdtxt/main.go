// fd2-golden-fdtxt 輸出 internal/fdtxt 的黃金差分資料，供 Rust 改寫逐欄比對
// （docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 fdtxt 的公開 API（另以 fdother.ReadResource 取得原版資源，
// 以標準函式庫 image/png 對 PNG 解碼結果做直接對照），不改任何遊戲行為。
// 合成輸入以固定種子的 PCG 產生，位元組與結果寫進同一份 JSON；真實原版檔只記路徑與 SHA-256，
// Rust 端以環境變數讀同一批檔案。
//
// 每個分離素材與 PNG 案例帶 compare 欄位：exact 表示格式正確的輸入，Rust 端逐欄比對；
// accept_reject 表示壞 JSON、壞 PNG 或檔案系統邊界，Rust 端只比對接受或拒收。Rust 版改用
// serde_json、png crate 與 std::path，刻意不照搬 Go 標準函式庫對壞輸入的寬鬆處理。
package main

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/adler32"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

// ---------------------------------------------------------------------------
// 共用輸出
// ---------------------------------------------------------------------------

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func errText(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

// errInfo 記錄錯誤。包裝（%w）的錯誤另記外層前綴與內層種類；內層為檔案系統錯誤時，文字含暫存路徑
// 且 errno 文字依實作而異，只比對前綴與種類。
func errInfo(err error) map[string]any {
	if err == nil {
		return map[string]any{"ok": true}
	}
	msg := err.Error()
	kind, stable := "none", msg
	if inner := errors.Unwrap(err); inner != nil {
		stable = strings.TrimSuffix(msg, inner.Error())
		var se *json.SyntaxError
		var te *json.UnmarshalTypeError
		var pe *fs.PathError
		switch {
		case errors.As(err, &se):
			kind = "json_syntax"
		case errors.As(err, &te):
			kind = "json_type"
		case errors.Is(err, fs.ErrNotExist):
			kind = "not_exist"
		case errors.As(err, &pe):
			kind = "io"
		default:
			kind = "wrapped"
		}
	}
	out := map[string]any{"ok": false, "kind": kind, "stable": stable, "err": msg}
	if kind == "not_exist" || kind == "io" {
		out["err"] = nil
	}
	return out
}

func ints16(w []uint16) []int {
	out := make([]int, len(w))
	for i, v := range w {
		out[i] = int(v)
	}
	return out
}

func stringsSummary(s *fdtxt.Strings) map[string]any {
	words := make([][]int, s.Count())
	for i := range words {
		w, err := s.Words(i)
		if err != nil {
			panic(err)
		}
		words[i] = ints16(w)
	}
	_, negErr := s.Words(-1)
	_, endErr := s.Words(s.Count())
	locate := [][]int{}
	var locEnd error
	for i := 0; ; i++ {
		loc, err := s.LocateLogicalUtterance(i)
		if err != nil {
			locEnd = err
			break
		}
		locate = append(locate, []int{loc.RawStringIndex, loc.RawUtteranceIndex})
	}
	_, locNeg := s.LocateLogicalUtterance(-1)
	return map[string]any{
		"count": s.Count(), "words": words, "words_neg_err": errText(negErr), "words_end_err": errText(endErr),
		"locate": locate, "locate_end_err": errText(locEnd), "locate_neg_err": errText(locNeg),
	}
}

// parseSummary 記錄 Parse 的結果。最後一個字串的範圍結束在奇數位元組時，Go 版的
// binary.LittleEndian.Uint16 會越界 panic；這裡如實記成 panic，不跳過該輸入。
func parseSummary(data []byte) (out map[string]any) {
	defer func() {
		if r := recover(); r != nil {
			out = map[string]any{"panic": true}
		}
	}()
	s, err := fdtxt.Parse(data)
	out = errInfo(err)
	if err == nil {
		out["strings"] = stringsSummary(s)
	}
	return out
}

func fontBits(f *fdtxt.Font) []byte {
	bits := make([]byte, f.GlyphCount()*fdtxt.GlyphBytes)
	for g := 0; g < f.GlyphCount(); g++ {
		for y := 0; y < fdtxt.GlyphHeight; y++ {
			for x := 0; x < fdtxt.GlyphWidth; x++ {
				set, err := f.GlyphBit(g, x, y)
				if err != nil {
					panic(err)
				}
				if set {
					bits[g*fdtxt.GlyphBytes+y*2+x/8] |= 0x80 >> (x % 8)
				}
			}
		}
	}
	return bits
}

func fontSummary(f *fdtxt.Font, withHex bool) map[string]any {
	n := f.GlyphCount()
	probes := []any{}
	for _, p := range [][3]int{{-1, 0, 0}, {n, 0, 0}, {0, -1, 0}, {0, 16, 0}, {0, 0, -1}, {0, 0, 16}} {
		_, err := f.GlyphBit(p[0], p[1], p[2])
		probes = append(probes, errText(err))
	}
	bits := fontBits(f)
	out := map[string]any{"glyph_count": n, "bits_sha256": digest(bits), "probe_errs": probes}
	if withHex {
		out["bits_hex"] = hex.EncodeToString(bits)
	}
	return out
}

// ---------------------------------------------------------------------------
// FDTXT 位移表案例
// ---------------------------------------------------------------------------

func randWord(rng *rand.Rand) uint16 {
	switch rng.IntN(10) {
	case 0:
		return uint16(0xff00 + rng.IntN(0xff))
	case 1:
		return 0xffff
	case 2:
		return fdtxt.OpeningGlyph
	case 3:
		return uint16(rng.Uint32())
	default:
		return uint16(rng.IntN(1824))
	}
}

func utteranceWords(rng *rand.Rand) []uint16 {
	var w []uint16
	for n := rng.IntN(6); n > 0; n-- {
		switch rng.IntN(5) {
		case 0:
			w = append(w, uint16(0xff00+rng.IntN(0xff)))
		case 1:
			w = append(w, fdtxt.OpeningGlyph)
		default:
			w = append(w, uint16(rng.IntN(600)), fdtxt.OpeningGlyph)
			for k := rng.IntN(5); k > 0; k-- {
				w = append(w, uint16(rng.IntN(1824)))
			}
		}
	}
	return w
}

func buildTable(stringsWords [][]uint16, terminate bool) []byte {
	n := len(stringsWords)
	out := make([]byte, 2*n)
	at := 2 * n
	var body []byte
	for i, w := range stringsWords {
		binary.LittleEndian.PutUint16(out[2*i:], uint16(at))
		for _, v := range w {
			body = binary.LittleEndian.AppendUint16(body, v)
		}
		if terminate {
			body = binary.LittleEndian.AppendUint16(body, 0xffff)
		}
		at = 2*n + len(body)
	}
	return append(out, body...)
}

func generateParse(index int, rng *rand.Rand) (string, []byte) {
	kinds := []string{"random", "table", "table-mutated", "utterance"}
	kind := kinds[index%len(kinds)]
	switch kind {
	case "random":
		b := make([]byte, rng.IntN(64))
		for i := range b {
			b[i] = byte(rng.Uint32())
		}
		if len(b) >= 2 && rng.IntN(2) == 0 {
			b[0], b[1] = byte(2*rng.IntN(len(b)/2+1)), 0
		}
		return kind, b
	case "utterance":
		ws := make([][]uint16, 1+rng.IntN(5))
		for i := range ws {
			ws[i] = utteranceWords(rng)
		}
		return kind, buildTable(ws, rng.IntN(4) != 0)
	}
	ws := make([][]uint16, 1+rng.IntN(8))
	for i := range ws {
		w := make([]uint16, rng.IntN(8))
		for k := range w {
			w[k] = randWord(rng)
		}
		ws[i] = w
	}
	b := buildTable(ws, rng.IntN(3) != 0)
	if rng.IntN(4) == 0 {
		b = append(b, byte(rng.Uint32()))
	}
	if kind == "table-mutated" {
		switch rng.IntN(6) {
		case 0:
			b[rng.IntN(2*len(ws))] ^= byte(1 << rng.IntN(8))
		case 1:
			b = b[:rng.IntN(len(b)+1)]
		case 2:
			if len(ws) > 1 {
				i := 1 + rng.IntN(len(ws)-1)
				binary.LittleEndian.PutUint16(b[2*i:], binary.LittleEndian.Uint16(b[2*(i-1):])-2)
			}
		case 3:
			i := rng.IntN(len(ws))
			binary.LittleEndian.PutUint16(b[2*i:], uint16(len(b)+2))
		case 4:
			i := rng.IntN(len(ws))
			binary.LittleEndian.PutUint16(b[2*i:], binary.LittleEndian.Uint16(b[2*i:])|1)
		case 5:
			b[0], b[1] = 0, 0
		}
	}
	return kind, b
}

var fixedParse = [][]byte{
	{}, {0}, {0, 0}, {1, 0}, {2, 0}, {2, 0, 0}, {3, 0, 0, 0}, {4, 0}, {4, 0, 4, 0}, {2, 0, 0xff, 0xff},
	{2, 0, 0x2d, 0x02}, {2, 0, 0x2d, 0x02, 0x01}, {4, 0, 6, 0, 1, 0, 2, 0}, {4, 0, 2, 0, 0, 0},
}

// ---------------------------------------------------------------------------
// 字型與繪字案例
// ---------------------------------------------------------------------------

type blitOp struct {
	Op        string         `json:"op"`
	DstHex    string         `json:"dst_hex"`
	Stride    int            `json:"stride"`
	Base      int            `json:"base"`
	Index     int            `json:"index"`
	Color     int            `json:"color"`
	Style     [3]int         `json:"style"`
	Presenter string         `json:"presenter"`
	Result    map[string]any `json:"result"`
}

func runBlit(f *fdtxt.Font, op *blitOp) {
	dst, _ := hex.DecodeString(op.DstHex)
	calls := [][]int{}
	font := f
	present := func(stride, base, index int, s fdtxt.NativeGlyphStyle) bool {
		calls = append(calls, []int{stride, base, index, int(s.Foreground), int(s.Shadow), int(s.Background)})
		switch op.Presenter {
		case "true":
			return true
		case "parity":
			return (stride+base+index)%2 == 0
		}
		return false
	}
	switch op.Presenter {
	case "none":
	case "nil":
		font = f.WithNativePresenter(nil)
	default:
		font = f.WithNativePresenter(present)
	}
	var err error
	style := fdtxt.NativeGlyphStyle{Foreground: byte(op.Style[0]), Shadow: byte(op.Style[1]), Background: byte(op.Style[2])}
	if op.Op == "blit" {
		err = font.BlitGlyph(dst, op.Stride, op.Base, op.Index, byte(op.Color))
	} else {
		err = font.BlitNativeGlyph(dst, op.Stride, op.Base, op.Index, style)
	}
	op.Result = map[string]any{"err": errText(err), "dst_sha256": digest(dst), "calls": calls}
}

func randomBlit(rng *rand.Rand, glyphs int) blitOp {
	op := blitOp{Op: []string{"blit", "native"}[rng.IntN(2)]}
	op.Stride = 16 + rng.IntN(20)
	if rng.IntN(10) == 0 {
		op.Stride = rng.IntN(18) - 2
	}
	op.Base = rng.IntN(40)
	if rng.IntN(10) == 0 {
		op.Base = rng.IntN(4) - 2
	}
	op.Index = rng.IntN(glyphs+2) - 1
	need := op.Base + 16*op.Stride + 16
	if op.Stride < 0 {
		need = 64
	}
	n := need - 8 + rng.IntN(24)
	if rng.IntN(6) == 0 {
		n = rng.IntN(need + 1)
	}
	if n < 0 {
		n = 0
	}
	dst := make([]byte, n)
	for i := range dst {
		dst[i] = byte(rng.IntN(4))
	}
	op.DstHex = hex.EncodeToString(dst)
	op.Color = rng.IntN(256)
	op.Style = [3]int{rng.IntN(256), rng.IntN(256), []int{0, rng.IntN(256)}[rng.IntN(2)]}
	op.Presenter = []string{"none", "nil", "true", "false", "parity"}[rng.IntN(5)]
	return op
}

func generateFont(index int, rng *rand.Rand) (string, []byte) {
	lengths := []int{0, 1, 31, 33, 64 + 1}
	if index < len(lengths) {
		return "bad-length", make([]byte, lengths[index])
	}
	n := 1 + rng.IntN(6)
	data := make([]byte, n*fdtxt.GlyphBytes)
	for i := range data {
		if rng.IntN(3) == 0 {
			data[i] = byte(rng.Uint32())
		}
	}
	if index%7 == 0 {
		for i := range data {
			data[i] = 0xff
		}
	}
	return "random", data
}

// ---------------------------------------------------------------------------
// 分離字表案例
// ---------------------------------------------------------------------------

type genSource struct {
	File     string `json:"file"`
	Resource int    `json:"resource"`
	Size     int    `json:"size"`
	MD5      string `json:"md5"`
	SHA256   string `json:"sha256"`
	RawSize  int    `json:"raw_size"`
}

type genToken struct {
	Kind       string `json:"kind"`
	GlyphIndex *int   `json:"glyph_index,omitempty"`
	Control    string `json:"control,omitempty"`
	Text       string `json:"text,omitempty"`
}

type genString struct {
	StringID    string     `json:"string_id"`
	SourceIndex int        `json:"source_index"`
	Text        string     `json:"text"`
	Tokens      []genToken `json:"tokens"`
}

type genResource struct {
	SchemaVersion int         `json:"schema_version"`
	Kind          string      `json:"kind"`
	AssetID       string      `json:"asset_id"`
	Status        string      `json:"status"`
	Evidence      string      `json:"evidence"`
	ReasonCode    string      `json:"reason_code,omitempty"`
	Source        genSource   `json:"source"`
	Strings       []genString `json:"strings,omitempty"`
}

func textSource(resource int) genSource {
	return genSource{File: "FDTXT.DAT", Resource: resource, Size: 120502, MD5: "fe5c487ce4313485f1da9d48d35b05f9",
		SHA256: "a4555f8a0e61e884b4f504d56a8bdde11672583bbbbc6506281ae10dcdfb1f69", RawSize: 64}
}

var glyphTexts = []string{"a", "Z", "0", "『", "』", "中", "龍", "é", "\U0001f600", "\\", "\"", " ", "<&>"}

func genDoc(rng *rand.Rand, resource int) genResource {
	doc := genResource{SchemaVersion: 1, Kind: "fdtxt_word_table", AssetID: fmt.Sprintf("text/FDTXT_%03d", resource),
		Status: "decoded", Evidence: "confirmed", Source: textSource(resource)}
	for i := 0; i < 1+rng.IntN(4); i++ {
		s := genString{StringID: fmt.Sprintf("FDTXT_%03d/string_%04d", resource, i), SourceIndex: i, Tokens: []genToken{}}
		for k := rng.IntN(6); k > 0; k-- {
			if rng.IntN(4) == 0 {
				c := fmt.Sprintf("%04X", 0xff00+rng.IntN(0xff))
				if rng.IntN(3) == 0 {
					c = strings.ToLower(c)
				}
				s.Tokens = append(s.Tokens, genToken{Kind: "control", Control: c})
				continue
			}
			g := rng.IntN(1824)
			if rng.IntN(10) == 0 {
				g = 0xfeff
			}
			t := glyphTexts[rng.IntN(len(glyphTexts))]
			s.Tokens = append(s.Tokens, genToken{Kind: "glyph", GlyphIndex: &g, Text: t})
			s.Text += t
		}
		doc.Strings = append(doc.Strings, s)
	}
	return doc
}

func marshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func intp(v int) *int { return &v }

// 結構化變體：每個都只改一處，涵蓋每一條檢查與 encoding/json 的特殊語意。
// textVariant 的 exact 為 true 時，輸入是格式正確的 JSON（json.Marshal 或等價寫法），Rust 端逐欄
// 比對；否則只比對接受或拒收：Go 標準函式庫對壞輸入的寬鬆處理不是原版行為，Rust 版刻意不照做。
type textVariant struct {
	kind  string
	raw   []byte
	exact bool
}

func textVariants(rng *rand.Rand) []textVariant {
	var out []textVariant
	add := func(kind string, raw []byte) { out = append(out, textVariant{kind, raw, false}) }
	addExact := func(kind string, raw []byte) { out = append(out, textVariant{kind, raw, true}) }
	mut := func(kind string, f func(d *genResource)) {
		d := genDoc(rng, 7)
		f(&d)
		addExact(kind, marshal(d))
	}
	mut("valid", func(d *genResource) {})
	mut("schema-version", func(d *genResource) { d.SchemaVersion = 2 })
	mut("kind", func(d *genResource) { d.Kind = "fdtxt" })
	mut("asset-id", func(d *genResource) { d.AssetID = "text/FDTXT_7" })
	mut("status", func(d *genResource) { d.Status = "pending" })
	mut("evidence", func(d *genResource) { d.Evidence = "strong" })
	mut("reason-code", func(d *genResource) { d.ReasonCode = "any" })
	mut("source-file", func(d *genResource) { d.Source.File = "FDTXT.dat" })
	mut("source-resource", func(d *genResource) { d.Source.Resource = 8 })
	mut("source-size", func(d *genResource) { d.Source.Size++ })
	mut("source-md5", func(d *genResource) { d.Source.MD5 = strings.ToUpper(d.Source.MD5) })
	mut("source-sha256", func(d *genResource) { d.Source.SHA256 = d.Source.SHA256[1:] + "0" })
	mut("raw-size-zero", func(d *genResource) { d.Source.RawSize = 0 })
	mut("raw-size-negative", func(d *genResource) { d.Source.RawSize = -5 })
	mut("strings-empty", func(d *genResource) { d.Strings = nil })
	mut("string-id", func(d *genResource) { d.Strings[0].StringID = "FDTXT_007/string_0001" })
	mut("source-index", func(d *genResource) { d.Strings[len(d.Strings)-1].SourceIndex = 9 })
	mut("tokens-nil", func(d *genResource) { d.Strings[0].Tokens = nil })
	mut("tokens-empty", func(d *genResource) { d.Strings[0].Tokens = []genToken{}; d.Strings[0].Text = "" })
	mut("tokens-empty-text", func(d *genResource) { d.Strings[0].Tokens = []genToken{} })
	tok := func(kind string, t genToken, text string) {
		mut(kind, func(d *genResource) {
			d.Strings[0].Tokens = append(d.Strings[0].Tokens, t)
			d.Strings[0].Text += text
		})
	}
	tok("glyph-nil-index", genToken{Kind: "glyph", Text: "x"}, "x")
	tok("glyph-negative", genToken{Kind: "glyph", GlyphIndex: intp(-1), Text: "x"}, "x")
	tok("glyph-control-min", genToken{Kind: "glyph", GlyphIndex: intp(0xff00), Text: "x"}, "x")
	tok("glyph-max", genToken{Kind: "glyph", GlyphIndex: intp(0xfeff), Text: "x"}, "x")
	tok("glyph-zero", genToken{Kind: "glyph", GlyphIndex: intp(0), Text: "x"}, "x")
	tok("glyph-with-control", genToken{Kind: "glyph", GlyphIndex: intp(1), Control: "FF00", Text: "x"}, "x")
	tok("glyph-empty-text", genToken{Kind: "glyph", GlyphIndex: intp(1)}, "")
	tok("control-with-index", genToken{Kind: "control", GlyphIndex: intp(0), Control: "FF00"}, "")
	tok("control-with-text", genToken{Kind: "control", Control: "FF00", Text: "x"}, "x")
	for _, c := range []string{"FF0", "FF000", "", "FFFF", "FEFF", "ff00", "FFFE", "fffe", "+fff", "0xFF", "FF_0",
		"gggg", "ＦＦ", "FFé", " FF0", "-FF0", "FFFe"} {
		tok("control-"+c, genToken{Kind: "control", Control: c}, "")
	}
	tok("kind-unknown", genToken{Kind: "Glyph", GlyphIndex: intp(1), Text: "x"}, "x")
	tok("kind-empty", genToken{Kind: "", GlyphIndex: intp(1), Text: "x"}, "x")
	mut("projection", func(d *genResource) { d.Strings[0].Text += "!" })
	mut("projection-lead", func(d *genResource) {
		g := 3
		d.Strings[0].Tokens = append(d.Strings[0].Tokens, genToken{Kind: "glyph", GlyphIndex: &g, Text: "ab"})
		d.Strings[0].Text = "ba" + d.Strings[0].Text
	})

	// encoding/json 層面的語意。
	base := marshal(genDoc(rng, 7))
	rep := func(kind, old, new string) { add(kind, bytes.Replace(base, []byte(old), []byte(new), 1)) }
	repAll := func(kind, old, new string) { add(kind, bytes.ReplaceAll(base, []byte(old), []byte(new))) }
	repExact := func(kind, old, new string) { addExact(kind, bytes.Replace(base, []byte(old), []byte(new), 1)) }
	rep("fold-upper", `"schema_version"`, `"SCHEMA_VERSION"`)
	rep("fold-long-s", `"schema_version"`, `"\u017fchema_version"`)
	rep("fold-long-s-raw", `"schema_version"`, "\"ſchema_version\"")
	rep("fold-kelvin", `"kind":"fdtxt_word_table"`, "\"Kind\":\"fdtxt_word_table\"")
	rep("fold-mixed", `"asset_id"`, `"Asset_ID"`)
	rep("fold-escape", `"status"`, `"st\u0061tus"`)
	rep("fold-wrong", `"status"`, `"status_"`)
	repAll("fold-tokens", `"tokens"`, `"TOKENS"`)
	repAll("fold-glyph-index", `"glyph_index"`, `"Glyph_Index"`)
	rep("number-float", `"schema_version":1`, `"schema_version":1.0`)
	rep("number-exp", `"schema_version":1`, `"schema_version":1e0`)
	rep("number-neg-zero", `"raw_size":64`, `"raw_size":-0`)
	rep("number-leading-zero", `"schema_version":1`, `"schema_version":01`)
	rep("number-huge", `"raw_size":64`, `"raw_size":99999999999999999999`)
	repExact("number-int64-max", `"raw_size":64`, `"raw_size":9223372036854775807`)
	rep("number-int64-min", `"raw_size":64`, `"raw_size":-9223372036854775808`)
	rep("number-string", `"schema_version":1`, `"schema_version":"1"`)
	rep("number-bool", `"schema_version":1`, `"schema_version":true`)
	rep("number-null", `"schema_version":1`, `"schema_version":null`)
	rep("number-array", `"schema_version":1`, `"schema_version":[1]`)
	rep("number-object", `"schema_version":1`, `"schema_version":{}`)
	rep("string-number", `"kind":"fdtxt_word_table"`, `"kind":5`)
	rep("string-null", `"status":"decoded"`, `"status":null`)
	rep("string-null-dup", `"status":"decoded"`, `"status":"decoded","status":null`)
	rep("string-bool", `"status":"decoded"`, `"status":false`)
	rep("string-object", `"status":"decoded"`, `"status":{"a":1}`)
	rep("source-null", `"source":{`, `"source":null,"x":{`)
	rep("source-array", `"source":{`, `"source":[],"x":{`)
	rep("source-string", `"source":{`, `"source":"s","x":{`)
	rep("strings-null", `"strings":[`, `"strings":null,"x":[`)
	rep("strings-object", `"strings":[`, `"strings":{},"x":[`)
	rep("strings-string", `"strings":[`, `"strings":"s","x":[`)
	rep("strings-elem-number", `"strings":[`, `"strings":[1,`)
	rep("strings-elem-null", `"strings":[`, `"strings":[null,`)
	rep("tokens-object", `"tokens":[`, `"tokens":{},"x":[`)
	rep("tokens-elem-string", `"tokens":[`, `"tokens":["t",`)
	rep("glyph-index-string", `"glyph_index":`, `"glyph_index":"1","y":`)
	rep("glyph-index-float", `"glyph_index":`, `"glyph_index":1.5,"y":`)
	rep("glyph-index-null", `"glyph_index":`, `"glyph_index":null,"y":`)
	rep("glyph-index-bool", `"glyph_index":`, `"glyph_index":true,"y":`)
	rep("glyph-index-array", `"glyph_index":`, `"glyph_index":[],"y":`)
	repExact("unknown-fields", `"kind":`, `"zz":{"a":[1,{"b":null}],"c":"\u0000"},"kind":`)
	rep("dup-later-wins", `"kind":"fdtxt_word_table"`, `"kind":"bad","kind":"fdtxt_word_table"`)
	rep("dup-later-loses", `"kind":"fdtxt_word_table"`, `"kind":"fdtxt_word_table","kind":"bad"`)
	rep("dup-source-merge", `"source":{`, `"source":{"file":"FDTXT.DAT","md5":"x"},"source":{`)
	rep("dup-source-merge-keeps", `"source":{"file":"FDTXT.DAT",`, `"source":{"file":"FDTXT.DAT"},"source":{`)
	addExact("whitespace", bytes.ReplaceAll(base, []byte(`,`), []byte(" ,\n\t\r ")))
	add("top-null", []byte(`null`))
	add("top-array", []byte(`[1,2]`))
	add("top-string", []byte(`"x"`))
	add("top-number", []byte(` 12 `))
	addExact("top-empty-object", []byte(`{}`))
	add("empty", []byte(``))
	add("spaces", []byte("  \n"))
	add("trailing-garbage", append(append([]byte{}, base...), []byte(" x")...))
	add("trailing-object", append(append([]byte{}, base...), []byte("{}")...))
	addExact("trailing-space", append(append([]byte{}, base...), []byte(" \n")...))
	add("bom", append([]byte("\xef\xbb\xbf"), base...))
	add("truncated", base[:len(base)/2])
	add("deep-10000", bytes.Replace(base, []byte(`"kind":`),
		[]byte(`"deep":`+strings.Repeat("[", 9999)+strings.Repeat("]", 9999)+`,"kind":`), 1))
	add("deep-10001", bytes.Replace(base, []byte(`"kind":`),
		[]byte(`"deep":`+strings.Repeat("[", 10000)+strings.Repeat("]", 10000)+`,"kind":`), 1))
	for _, bad := range []string{`\x`, `\u12`, `\u12G4`, "\x01", "\x1f", `\'`} {
		rep("bad-string-"+hex.EncodeToString([]byte(bad)), `"status":"decoded"`, `"status":"dec`+bad+`oded"`)
	}
	for _, lit := range []string{"tru", "nul", "fals", "truE", "-", "1.", "1e", "1e+", ".5", "+1", "0x1", "NaN"} {
		rep("bad-literal-"+lit, `"schema_version":1`, `"schema_version":`+lit)
	}
	for _, bad := range []string{`{"a" 1}`, `{"a":1 "b":2}`, `[1 2]`, `{1:2}`, `{"a":1,}`, `[1,]`, `{,}`, `]`, `}`, `:`} {
		add("bad-structure-"+bad, []byte(bad))
	}
	// 文字中的跳脫、無效 UTF-8 與代理項：glyph text 與字串 text 走同樣的替換，投影仍相等。
	exactPairs := map[string]bool{"text-escape-equal": true, "text-escape-slash": true, "text-escapes": true,
		"text-pair": true, "text-mismatch": true}
	pair := func(kind, tokText, strText string) {
		raw := fmt.Sprintf(`{"schema_version":1,"kind":"fdtxt_word_table","asset_id":"text/FDTXT_007","status":"decoded",`+
			`"evidence":"confirmed","source":{"file":"FDTXT.DAT","resource":7,"size":120502,`+
			`"md5":"fe5c487ce4313485f1da9d48d35b05f9","sha256":"a4555f8a0e61e884b4f504d56a8bdde11672583bbbbc6506281ae10dcdfb1f69",`+
			`"raw_size":4},"strings":[{"string_id":"FDTXT_007/string_0000","source_index":0,"text":"%s",`+
			`"tokens":[{"kind":"glyph","glyph_index":5,"text":"%s"}]}]}`, strText, tokText)
		if exactPairs[kind] {
			addExact(kind, []byte(raw))
		} else {
			add(kind, []byte(raw))
		}
	}
	pair("text-escape-equal", `\u3010`, "【")
	pair("text-escape-slash", `\/`, "/")
	pair("text-escapes", `\b\f\n\r\t\"\\`, `\u0008\u000c\u000a\u000d\u0009\u0022\u005c`)
	pair("text-pair", `\ud83d\ude00`, "\U0001f600")
	pair("text-lone-high", `\ud83d`, "�")
	pair("text-lone-high-then-u", `\ud83d\u0041`, "\ufffdA")
	pair("text-lone-low", `\ude00x`, "�x")
	pair("text-high-high", `\ud83d\ud83d\ude00`, "\ufffd\U0001f600")
	pair("text-invalid-utf8", "\xe4\xb8", "��")
	pair("text-invalid-utf8-one", "\xe4\xb8", "�")
	pair("text-surrogate-utf8", "\xed\xa0\x80", "���")
	pair("text-overlong", "\xc0\xaf", "��")
	pair("text-f4-90", "\xf4\x90\x80\x80", "����")
	pair("text-cont-only", "\x80", "�")
	pair("text-raw-vs-escape-ffd", "\xff", `\ufffd`)
	pair("text-null-char", `\u0000`, "\x00")
	pair("text-mismatch", "a", "b")
	// 重複的 strings／tokens 鍵：encoding/json 會把第二個陣列解進既有元素。
	stale := func(kind, stringsA, stringsB string) {
		raw := fmt.Sprintf(`{"schema_version":1,"kind":"fdtxt_word_table","asset_id":"text/FDTXT_007","status":"decoded",`+
			`"evidence":"confirmed","source":{"file":"FDTXT.DAT","resource":7,"size":120502,`+
			`"md5":"fe5c487ce4313485f1da9d48d35b05f9","sha256":"a4555f8a0e61e884b4f504d56a8bdde11672583bbbbc6506281ae10dcdfb1f69",`+
			`"raw_size":4},"strings":%s,"strings":%s}`, stringsA, stringsB)
		add(kind, []byte(raw))
	}
	s0 := `{"string_id":"FDTXT_007/string_0000","source_index":0,"text":"a","tokens":[{"kind":"glyph","glyph_index":1,"text":"a"}]}`
	s1 := `{"string_id":"FDTXT_007/string_0001","source_index":1,"text":"b","tokens":[{"kind":"glyph","glyph_index":2,"text":"b"}]}`
	stale("stale-merge-fields", "["+s0+"]", `[{"text":"a"}]`)
	stale("stale-merge-tokens", "["+s0+"]", `[{"tokens":[{"glyph_index":9}]}]`)
	stale("stale-truncate", "["+s0+","+s1+"]", "["+s0+"]")
	stale("stale-null-elem", "["+s0+"]", `[null]`)
	stale("stale-empty-resets", "["+s0+"]", `[]`)
	stale("stale-null-resets", "["+s0+"]", `null`)
	stale("stale-regrow", "["+s0+","+s1+"],\"strings\":["+s0+"]", `[null,null]`)
	stale("stale-regrow-empty", "["+s0+","+s1+"],\"strings\":[]", `[null,null]`)
	stale("stale-tokens-null", "["+s0+"]", `[{"tokens":null}]`)
	stale("stale-tokens-empty", "["+s0+"]", `[{"tokens":[],"text":""}]`)
	stale("stale-glyph-null", "["+s0+"]", `[{"tokens":[{"glyph_index":null}]}]`)
	stale("stale-type-error", "["+s0+"]", `[{"tokens":[{"glyph_index":"x"}]}]`)
	return out
}

func mutateJSON(rng *rand.Rand, base []byte) []byte {
	b := append([]byte{}, base...)
	charset := []byte(`{}[]",:0123456789-.eE \\uabcdfnrt`)
	for n := 1 + rng.IntN(3); n > 0; n-- {
		pos := rng.IntN(len(b) + 1)
		c := charset[rng.IntN(len(charset))]
		switch rng.IntN(3) {
		case 0:
			if pos < len(b) {
				b[pos] = c
			}
		case 1:
			b = append(b[:pos], append([]byte{c}, b[pos:]...)...)
		case 2:
			if pos < len(b) {
				b = append(b[:pos], b[pos+1:]...)
			}
		}
	}
	return b
}

type fileJSON struct {
	Path string `json:"path"`
	Hex  string `json:"hex"`
}

type sepCase struct {
	Index    int            `json:"index"`
	Kind     string         `json:"kind"`
	Root     string         `json:"root"` // "tmp"、"empty"，或接在暫存目錄後的字尾
	Resource int            `json:"resource"`
	Compare  string         `json:"compare"` // "exact" 或 "accept_reject"
	Files    []fileJSON     `json:"files"`
	Dirs     []string       `json:"dirs"`
	Result   map[string]any `json:"result"`
}

func materialize(c *sepCase) (root string, cleanup func()) {
	dir, err := os.MkdirTemp("", "fd2-golden-fdtxt-")
	if err != nil {
		panic(err)
	}
	for _, d := range c.Dirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
			panic(err)
		}
	}
	for _, f := range c.Files {
		data, _ := hex.DecodeString(f.Hex)
		path := filepath.Join(dir, f.Path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			panic(err)
		}
	}
	switch c.Root {
	case "empty":
		root = ""
	case "tmp":
		root = dir
	default:
		root = dir + c.Root
	}
	return root, func() { os.RemoveAll(dir) }
}

func compareMode(exact bool) string {
	if exact {
		return "exact"
	}
	return "accept_reject"
}

func runText(c *sepCase) {
	if c.Compare == "" {
		c.Compare = "accept_reject"
	}
	root, cleanup := materialize(c)
	defer cleanup()
	s, err := fdtxt.LoadSeparatedResource(root, c.Resource)
	c.Result = errInfo(err)
	if err == nil {
		c.Result["strings"] = stringsSummary(s)
	}
}

func textCases(rng *rand.Rand) []sepCase {
	var out []sepCase
	add := func(kind string, resource int, raw []byte, exact bool) {
		out = append(out, sepCase{Kind: kind, Root: "tmp", Resource: resource, Compare: compareMode(exact),
			Files: []fileJSON{{Path: fmt.Sprintf("FDTXT_%03d/resource.json", resource), Hex: hex.EncodeToString(raw)}}})
	}
	for _, v := range textVariants(rng) {
		add(v.kind, 7, v.raw, v.exact)
	}
	for i := 0; i < 160; i++ {
		r := rng.IntN(40)
		add("random-valid", r, marshal(genDoc(rng, r)), true)
	}
	small := marshal(genDoc(rand.New(rand.NewPCG(7, 7)), 3))
	for i := 0; i < 400; i++ {
		add("random-mutated", 3, mutateJSON(rng, small), false)
	}
	valid := marshal(genDoc(rng, 12))
	file := func(p string) []fileJSON { return []fileJSON{{Path: p, Hex: hex.EncodeToString(valid)}} }
	out = append(out,
		sepCase{Kind: "fs-missing-root", Root: "/missing", Resource: 12},
		sepCase{Kind: "fs-missing-file", Root: "tmp", Resource: 12, Dirs: []string{"FDTXT_012"}},
		sepCase{Kind: "fs-file-is-dir", Root: "tmp", Resource: 12, Dirs: []string{"FDTXT_012/resource.json"}},
		sepCase{Kind: "fs-empty-root", Root: "empty", Resource: 12, Files: file("FDTXT_012/resource.json")},
		sepCase{Kind: "fs-negative", Root: "tmp", Resource: -1, Files: file("FDTXT_012/resource.json")},
		sepCase{Kind: "fs-wrong-resource", Root: "tmp", Resource: 13, Files: file("FDTXT_012/resource.json")},
		sepCase{Kind: "fs-clean-dotdot", Root: "/missing/..", Resource: 12, Files: file("FDTXT_012/resource.json")},
		sepCase{Kind: "fs-clean-dot", Root: "/./", Resource: 12, Files: file("FDTXT_012/resource.json")},
		sepCase{Kind: "fs-clean-deep", Root: "/a/b/../../", Resource: 12, Files: file("FDTXT_012/resource.json")},
		sepCase{Kind: "fs-1000", Root: "tmp", Resource: 1000, Compare: "exact",
			Files: []fileJSON{{Path: "FDTXT_1000/resource.json", Hex: hex.EncodeToString(marshal(genDoc(rng, 1000)))}}},
	)
	return out
}

// ---------------------------------------------------------------------------
// PNG：Go 編碼器輸出與手工區塊
// ---------------------------------------------------------------------------

type pngChunk struct {
	typ    string
	data   []byte
	badCRC bool
}

func writePNG(chunks []pngChunk) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	for _, c := range chunks {
		var l [4]byte
		binary.BigEndian.PutUint32(l[:], uint32(len(c.data)))
		b.Write(l[:])
		b.WriteString(c.typ)
		b.Write(c.data)
		crc := crc32.ChecksumIEEE(append([]byte(c.typ), c.data...))
		if c.badCRC {
			crc ^= 1
		}
		binary.BigEndian.PutUint32(l[:], crc)
		b.Write(l[:])
	}
	return b.Bytes()
}

func ihdr(w, h int, depth, ctype, interlace byte) pngChunk {
	d := make([]byte, 13)
	binary.BigEndian.PutUint32(d[0:], uint32(w))
	binary.BigEndian.PutUint32(d[4:], uint32(h))
	d[8], d[9], d[12] = depth, ctype, interlace
	return pngChunk{typ: "IHDR", data: d}
}

func channels(ctype int) int {
	return map[int]int{0: 1, 2: 3, 3: 1, 4: 2, 6: 4}[ctype]
}

var adam7 = [7][4]int{{8, 8, 0, 0}, {8, 8, 4, 0}, {4, 8, 0, 4}, {4, 4, 2, 0}, {2, 4, 0, 2}, {2, 2, 1, 0}, {1, 2, 0, 1}}

// scanlines 產生濾波前的掃描列。sample 為 nil 時內容與濾波類型都隨機；否則以濾波 0 寫入
// sample(x, y, channel) 的取樣值。
func scanlines(rng *rand.Rand, w, h, depth, ctype, interlace int, badFilter bool, sample func(x, y, c int) int) []byte {
	type pass struct{ w, h, xf, yf, xo, yo int }
	passes := []pass{{w, h, 1, 1, 0, 0}}
	if interlace == 1 {
		passes = nil
		for _, p := range adam7 {
			pw, ph := (w-p[2]+p[0]-1)/p[0], (h-p[3]+p[1]-1)/p[1]
			if pw > 0 && ph > 0 {
				passes = append(passes, pass{pw, ph, p[0], p[1], p[2], p[3]})
			}
		}
	}
	ch := channels(ctype)
	var out []byte
	for _, p := range passes {
		rowBytes := (depth*ch*p.w + 7) / 8
		for y := 0; y < p.h; y++ {
			row := make([]byte, 1+rowBytes)
			if sample == nil {
				row[0] = byte(rng.IntN(5))
				if badFilter && rng.IntN(3) == 0 {
					row[0] = byte(5 + rng.IntN(250))
				}
				for i := 1; i < len(row); i++ {
					row[i] = byte(rng.Uint32())
				}
			} else {
				bit := 0
				for x := 0; x < p.w; x++ {
					for c := 0; c < ch; c++ {
						v := sample(x*p.xf+p.xo, y*p.yf+p.yo, c)
						if depth == 16 {
							row[1+bit/8] = byte(v >> 8)
							row[2+bit/8] = byte(v)
						} else {
							row[1+bit/8] |= byte(v<<(8-depth)) >> (bit % 8)
						}
						bit += depth
					}
				}
			}
			out = append(out, row...)
		}
	}
	return out
}

func zlibStream(data []byte, level int) []byte {
	var b bytes.Buffer
	zw, err := zlib.NewWriterLevel(&b, level)
	if err != nil {
		panic(err)
	}
	zw.Write(data)
	zw.Close()
	return b.Bytes()
}

// zlibDict 以 FDICT 標頭包裝原始 deflate；Go 解碼端以 nil 字典比對 adler32（=1）。
func zlibDict(data []byte, dictID uint32) []byte {
	var raw bytes.Buffer
	fw, _ := flate.NewWriter(&raw, 6)
	fw.Write(data)
	fw.Close()
	cmf := byte(0x78)
	flg := byte(0x20)
	flg |= byte((31 - (int(cmf)*256+int(flg))%31) % 31)
	out := []byte{cmf, flg}
	out = binary.BigEndian.AppendUint32(out, dictID)
	out = append(out, raw.Bytes()...)
	return binary.BigEndian.AppendUint32(out, adler32.Checksum(data))
}

func splitIDAT(rng *rand.Rand, z []byte, mode int) []pngChunk {
	var out []pngChunk
	switch mode {
	case 0:
		return []pngChunk{{typ: "IDAT", data: z}}
	case 1: // 隨機切塊，含零長度
		for len(z) > 0 {
			n := rng.IntN(min(len(z), 300) + 1)
			out = append(out, pngChunk{typ: "IDAT", data: z[:n]})
			z = z[n:]
		}
		return out
	default: // 一位元組一塊
		for _, c := range z {
			out = append(out, pngChunk{typ: "IDAT", data: []byte{c}})
		}
		return out
	}
}

type pngCase struct {
	Index   int            `json:"index"`
	Kind    string         `json:"kind"`
	Compare string         `json:"compare"`
	PNGHex  string         `json:"png_hex"`
	Result  map[string]any `json:"result"`
}

func pngSummary(data []byte) map[string]any {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return map[string]any{"ok": false, "err": err.Error()}
	}
	b := img.Bounds()
	var px []byte
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			px = binary.BigEndian.AppendUint16(px, uint16(r))
			px = binary.BigEndian.AppendUint16(px, uint16(g))
			px = binary.BigEndian.AppendUint16(px, uint16(bl))
			px = binary.BigEndian.AppendUint16(px, uint16(a))
		}
	}
	return map[string]any{"ok": true, "type": fmt.Sprintf("%T", img), "width": b.Dx(), "height": b.Dy(),
		"rgba_sha256": digest(px)}
}

func randomImage(rng *rand.Rand, w, h int) (string, image.Image) {
	r := image.Rect(0, 0, w, h)
	fill := func(set func(x, y int)) {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				set(x, y)
			}
		}
	}
	u8 := func() uint8 { return uint8(rng.Uint32()) }
	u16 := func() uint16 { return uint16(rng.Uint32()) }
	switch rng.IntN(9) {
	case 0:
		m := image.NewGray(r)
		fill(func(x, y int) { m.SetGray(x, y, color.Gray{u8()}) })
		return "gray", m
	case 1:
		m := image.NewGray16(r)
		fill(func(x, y int) { m.SetGray16(x, y, color.Gray16{u16()}) })
		return "gray16", m
	case 2:
		m := image.NewRGBA(r)
		fill(func(x, y int) { m.SetRGBA(x, y, color.RGBA{u8(), u8(), u8(), 0xff}) })
		return "rgba-opaque", m
	case 3:
		m := image.NewNRGBA(r)
		fill(func(x, y int) {
			m.SetNRGBA(x, y, color.NRGBA{u8(), u8(), u8(), []uint8{0, 0x80, 0xff, u8()}[rng.IntN(4)]})
		})
		return "nrgba", m
	case 4:
		m := image.NewNRGBA64(r)
		fill(func(x, y int) {
			m.SetNRGBA64(x, y, color.NRGBA64{u16(), u16(), u16(), []uint16{0, 0xffff, u16()}[rng.IntN(3)]})
		})
		return "nrgba64", m
	case 5:
		m := image.NewRGBA64(r)
		fill(func(x, y int) { m.SetRGBA64(x, y, color.RGBA64{u16(), u16(), u16(), 0xffff}) })
		return "rgba64-opaque", m
	default:
		n := []int{2, 4, 16, 256}[rng.IntN(4)]
		pal := make(color.Palette, n)
		for i := range pal {
			if rng.IntN(3) == 0 {
				pal[i] = color.NRGBA{u8(), u8(), u8(), u8()}
			} else {
				pal[i] = color.RGBA{u8(), u8(), u8(), 0xff}
			}
		}
		m := image.NewPaletted(r, pal)
		fill(func(x, y int) { m.SetColorIndex(x, y, uint8(rng.IntN(n))) })
		return fmt.Sprintf("paletted%d", n), m
	}
}

var colorCombos = [][2]int{{1, 0}, {2, 0}, {4, 0}, {8, 0}, {16, 0}, {8, 2}, {16, 2}, {1, 3}, {2, 3}, {4, 3}, {8, 3},
	{8, 4}, {16, 4}, {8, 6}, {16, 6}}

// craftedPNG 以手工區塊組出 PNG。第三個回傳值表示是否符合 PNG 規格：任一破壞選項或規格外的
// 樣本（調色盤索引超出 PLTE、tRNS 長度或取值不合色彩型別）都會讓它變成 false，並在 kind 加上
// 對應標記；Rust 端只對符合規格者逐欄比對。
func craftedPNG(rng *rand.Rand) (string, []byte, bool) {
	wf := true
	bad := func(tag string, kind *string) {
		wf = false
		*kind += tag
	}
	combo := colorCombos[rng.IntN(len(colorCombos))]
	depth, ctype := combo[0], combo[1]
	w, h := 1+rng.IntN(24), 1+rng.IntN(24)
	interlace := rng.IntN(2)
	kind := fmt.Sprintf("crafted-d%d-c%d-i%d", depth, ctype, interlace)
	chunks := []pngChunk{ihdr(w, h, byte(depth), byte(ctype), byte(interlace))}
	np := 0
	if ctype == 3 || (ctype == 2 || ctype == 6) && rng.IntN(4) == 0 {
		np = 1 + rng.IntN(1<<min(depth, 8))
		if np > 256 {
			np = 256
		}
		pal := make([]byte, 3*np)
		for i := range pal {
			pal[i] = byte(rng.Uint32())
		}
		chunks = append(chunks, pngChunk{typ: "PLTE", data: pal})
		kind += "-plte"
		if ctype == 3 && np < 1<<depth {
			bad("-palidx", &kind) // 隨機索引會超出 PLTE 筆數
		}
	}
	if rng.IntN(3) == 0 {
		var t []byte
		switch ctype {
		case 0:
			t = []byte{byte(rng.IntN(2)), byte(rng.IntN(4))}
			if rng.IntN(2) == 0 {
				t[1] = byte(rng.Uint32())
			}
		case 2:
			t = make([]byte, 6)
			for i := range t {
				t[i] = byte(rng.IntN(3))
			}
		case 3:
			t = make([]byte, rng.IntN(300))
			for i := range t {
				t[i] = byte(rng.Uint32())
			}
		default:
			t = []byte{1, 2}
		}
		chunks = append(chunks, pngChunk{typ: "tRNS", data: t})
		kind += "-trns"
		switch {
		case ctype == 0 && depth < 16 && int(t[0])<<8|int(t[1]) >= 1<<depth,
			ctype == 2 && depth < 16 && (t[0] != 0 || t[2] != 0 || t[4] != 0),
			ctype == 3 && len(t) > np,
			ctype == 4 || ctype == 6:
			bad("-badtrns", &kind)
		}
	}
	if rng.IntN(4) == 0 {
		chunks = append(chunks, pngChunk{typ: []string{"tEXt", "abCD", "zzZZ"}[rng.IntN(3)], data: []byte("hello")})
		kind += "-extra"
	}
	badFilter := rng.IntN(6) == 0
	raw := scanlines(rng, w, h, depth, ctype, interlace, badFilter, nil)
	if badFilter {
		bad("-badfilter", &kind)
	}
	if rng.IntN(8) == 0 {
		raw = raw[:rng.IntN(len(raw))]
		bad("-short", &kind)
	} else if rng.IntN(8) == 0 {
		raw = append(raw, byte(rng.Uint32()))
		bad("-long", &kind)
	}
	var z []byte
	switch rng.IntN(5) {
	case 0:
		z = zlibStream(raw, zlib.NoCompression)
		kind += "-stored"
	case 1:
		z = zlibStream(raw, zlib.HuffmanOnly)
	case 2:
		z = zlibDict(raw, []uint32{1, 0, 2}[rng.IntN(3)])
		bad("-fdict", &kind) // PNG 不允許預設字典
	default:
		z = zlibStream(raw, []int{1, 6, 9}[rng.IntN(3)])
	}
	switch rng.IntN(10) {
	case 0:
		z[len(z)-1-rng.IntN(4)] ^= 1
		bad("-adler", &kind)
	case 1:
		z[0] = []byte{0x79, 0x88, 0x78}[rng.IntN(3)]
		bad("-zhdr", &kind)
	case 2:
		z = append(z, make([]byte, 1+rng.IntN(5))...)
		bad("-ztrail", &kind)
	case 3:
		if len(z) > 6 {
			z[2+rng.IntN(len(z)-6)] ^= byte(1 << rng.IntN(8))
			bad("-zflip", &kind)
		}
	}
	chunks = append(chunks, splitIDAT(rng, z, rng.IntN(3))...)
	if rng.IntN(5) == 0 {
		chunks = append(chunks, pngChunk{typ: "IDAT", data: []byte{}})
		kind += "-idat0"
	}
	if rng.IntN(5) == 0 {
		chunks = append(chunks, pngChunk{typ: "IDAT", data: []byte{1, 2, 3}})
		bad("-idatx", &kind)
	}
	if rng.IntN(10) != 0 {
		chunks = append(chunks, pngChunk{typ: "IEND"})
	} else {
		bad("-noiend", &kind)
	}
	if rng.IntN(12) == 0 {
		chunks[rng.IntN(len(chunks))].badCRC = true
		bad("-crc", &kind)
	}
	if rng.IntN(15) == 0 {
		i, j := rng.IntN(len(chunks)), rng.IntN(len(chunks))
		chunks[i], chunks[j] = chunks[j], chunks[i]
		bad("-swap", &kind)
	}
	b := writePNG(chunks)
	if rng.IntN(10) == 0 {
		b = append(b, []byte("trailing")...)
		bad("-trailing", &kind)
	}
	if rng.IntN(10) == 0 {
		b = b[:rng.IntN(len(b))]
		bad("-trunc", &kind)
	}
	if rng.IntN(10) == 0 {
		b[rng.IntN(len(b))] ^= byte(1 << rng.IntN(8))
		bad("-flip", &kind)
	}
	return kind, b, wf
}

// bufioWindow 測 IDAT 尾端多餘位元組：Go 只放過已被 bufio 預讀的部分。
func bufioWindowCases(rng *rand.Rand) []pngCase {
	var out []pngCase
	for _, level := range []int{zlib.NoCompression, zlib.BestCompression} {
		w, h := 64, 70
		raw := scanlines(rng, w, h, 8, 6, 0, false, nil)
		for i := 0; i < len(raw); i += 1 + 4*w {
			raw[i] = 0
		}
		z := zlibStream(raw, level)
		rem := len(z) % 4096
		for _, extra := range []int{1, 2, 4096 - rem - 1, 4096 - rem, 4096 - rem + 1, 4096 - rem + 4096, 5000} {
			if extra <= 0 {
				continue
			}
			zz := append(append([]byte{}, z...), make([]byte, extra)...)
			for _, split := range []int{0, 4096, 10000} {
				var idat []pngChunk
				if split == 0 || split >= len(zz) {
					idat = []pngChunk{{typ: "IDAT", data: zz}}
				} else {
					idat = []pngChunk{{typ: "IDAT", data: zz[:split]}, {typ: "IDAT", data: zz[split:]}}
				}
				chunks := append([]pngChunk{ihdr(w, h, 8, 6, 0)}, idat...)
				chunks = append(chunks, pngChunk{typ: "IEND"})
				out = append(out, pngCase{Kind: fmt.Sprintf("bufio-l%d-x%d-s%d", level, extra, split), PNGHex: hex.EncodeToString(writePNG(chunks))})
			}
		}
	}
	return out
}

func pngCases(rng *rand.Rand) []pngCase {
	var out []pngCase
	for i := 0; i < 160; i++ {
		kind, img := randomImage(rng, 1+rng.IntN(32), 1+rng.IntN(32))
		var b bytes.Buffer
		enc := png.Encoder{CompressionLevel: []png.CompressionLevel{png.DefaultCompression, png.NoCompression,
			png.BestSpeed, png.BestCompression}[rng.IntN(4)]}
		if err := enc.Encode(&b, img); err != nil {
			panic(err)
		}
		out = append(out, pngCase{Kind: "encoder-" + kind, Compare: "exact", PNGHex: hex.EncodeToString(b.Bytes())})
	}
	for i := 0; i < 700; i++ {
		kind, b, wf := craftedPNG(rng)
		out = append(out, pngCase{Kind: kind, Compare: compareMode(wf), PNGHex: hex.EncodeToString(b)})
	}
	out = append(out, bufioWindowCases(rng)...)
	fixed := map[string][]byte{
		"empty":          {},
		"sig-only":       []byte("\x89PNG\r\n\x1a\n"),
		"sig-partial":    []byte("\x89PNG"),
		"not-png":        []byte("GIF89a......"),
		"ihdr-len":       writePNG([]pngChunk{{typ: "IHDR", data: make([]byte, 12)}}),
		"ihdr-zero-w":    writePNG([]pngChunk{ihdr(0, 1, 8, 0, 0)}),
		"ihdr-neg":       writePNG([]pngChunk{ihdr(-1, 1, 8, 0, 0)}),
		"ihdr-depth":     writePNG([]pngChunk{ihdr(1, 1, 3, 0, 0)}),
		"ihdr-overflow":  writePNG([]pngChunk{ihdr(0x7fffffff, 0x7fffffff, 8, 0, 0)}),
		"ihdr-p16":       writePNG([]pngChunk{ihdr(1, 1, 16, 3, 0)}),
		"ihdr-interlace": writePNG([]pngChunk{ihdr(1, 1, 8, 0, 2)}),
		"ihdr-compress": func() []byte {
			c := ihdr(1, 1, 8, 0, 0)
			c.data[10] = 1
			return writePNG([]pngChunk{c})
		}(),
		"ihdr-filter": func() []byte {
			c := ihdr(1, 1, 8, 0, 0)
			c.data[11] = 1
			return writePNG([]pngChunk{c})
		}(),
		"two-ihdr":      writePNG([]pngChunk{ihdr(1, 1, 8, 0, 0), ihdr(1, 1, 8, 0, 0)}),
		"plte-gray":     writePNG([]pngChunk{ihdr(1, 1, 8, 0, 0), {typ: "PLTE", data: []byte{1, 2, 3}}}),
		"plte-len":      writePNG([]pngChunk{ihdr(1, 1, 8, 3, 0), {typ: "PLTE", data: []byte{1, 2}}}),
		"plte-too-many": writePNG([]pngChunk{ihdr(1, 1, 1, 3, 0), {typ: "PLTE", data: make([]byte, 9)}}),
		"paletted-no-plte": writePNG([]pngChunk{ihdr(1, 1, 8, 3, 0),
			{typ: "IDAT", data: zlibStream([]byte{0, 0}, 6)}, {typ: "IEND"}}),
		"trns-before-plte": writePNG([]pngChunk{ihdr(1, 1, 8, 3, 0), {typ: "tRNS", data: []byte{1}}}),
		"trns-ga":          writePNG([]pngChunk{ihdr(1, 1, 8, 4, 0), {typ: "tRNS", data: []byte{1, 2}}}),
		"trns-gray-len":    writePNG([]pngChunk{ihdr(1, 1, 8, 0, 0), {typ: "tRNS", data: []byte{1}}}),
		"idat-before-ihdr": writePNG([]pngChunk{{typ: "IDAT", data: []byte{1}}}),
		"iend-before-idat": writePNG([]pngChunk{ihdr(1, 1, 8, 0, 0), {typ: "IEND"}}),
		"iend-len": writePNG([]pngChunk{ihdr(1, 1, 8, 0, 0), {typ: "IDAT", data: zlibStream([]byte{0, 7}, 6)},
			{typ: "IEND", data: []byte{0}}}),
		"no-iend":   writePNG([]pngChunk{ihdr(1, 1, 8, 0, 0), {typ: "IDAT", data: zlibStream([]byte{0, 7}, 6)}}),
		"big-chunk": append(writePNG([]pngChunk{ihdr(1, 1, 8, 0, 0)}), 0x80, 0, 0, 0, 'a', 'b', 'C', 'D'),
		"idat-then-text": writePNG([]pngChunk{ihdr(1, 1, 8, 0, 0), {typ: "IDAT", data: zlibStream([]byte{0}, 6)[:4]},
			{typ: "tEXt", data: []byte("x")}, {typ: "IEND"}}),
	}
	keys := make([]string, 0, len(fixed))
	for k := range fixed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, pngCase{Kind: "fixed-" + k, PNGHex: hex.EncodeToString(fixed[k])})
	}
	return out
}

// ---------------------------------------------------------------------------
// 分離字型案例
// ---------------------------------------------------------------------------

type genFont struct {
	SchemaVersion int       `json:"schema_version"`
	Kind          string    `json:"kind"`
	AssetID       string    `json:"asset_id"`
	Status        string    `json:"status"`
	Evidence      string    `json:"evidence"`
	GlyphCount    int       `json:"glyph_count"`
	CellWidth     int       `json:"cell_width"`
	CellHeight    int       `json:"cell_height"`
	Columns       int       `json:"columns"`
	Rows          int       `json:"rows"`
	Atlas         string    `json:"atlas"`
	Source        genSource `json:"source"`
}

func validFontDoc() genFont {
	return genFont{SchemaVersion: 1, Kind: "bitmap_font", AssetID: "font/FDOTHER_004", Status: "decoded",
		Evidence: "confirmed", GlyphCount: 1824, CellWidth: 16, CellHeight: 16, Columns: 32, Rows: 57,
		Atlas: "atlas.png", Source: genSource{File: "FDOTHER.DAT", Resource: 4, Size: 3382481,
			MD5: "22f56e5027edc7c766ad34ca4e5aca93", SHA256: "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce",
			RawSize: 58368}}
}

// atlasBits 隨機挑幾個字格填入位元，其餘為黑，方便壓縮。
func atlasBits(rng *rand.Rand) func(x, y int) bool {
	set := map[[2]int]bool{}
	for n := 20 + rng.IntN(30); n > 0; n-- {
		g := rng.IntN(1824)
		for k := rng.IntN(80); k > 0; k-- {
			set[[2]int{(g%32)*16 + rng.IntN(16), (g/32)*16 + rng.IntN(16)}] = true
		}
	}
	return func(x, y int) bool { return set[[2]int{x, y}] }
}

func encodeImage(img image.Image) []byte {
	var b bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&b, img); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func atlasVariants(rng *rand.Rand) map[string][]byte {
	out := map[string][]byte{}
	r := image.Rect(0, 0, 512, 912)
	bit := atlasBits(rng)
	each := func(set func(x, y int, on bool)) {
		for y := 0; y < 912; y++ {
			for x := 0; x < 512; x++ {
				set(x, y, bit(x, y))
			}
		}
	}
	gray := image.NewGray(r)
	each(func(x, y int, on bool) {
		if on {
			gray.SetGray(x, y, color.Gray{255})
		}
	})
	out["gray"] = encodeImage(gray)
	pal := image.NewPaletted(r, color.Palette{color.Black, color.White})
	each(func(x, y int, on bool) {
		if on {
			pal.SetColorIndex(x, y, 1)
		}
	})
	out["paletted1"] = encodeImage(pal)
	pal16 := image.NewPaletted(r, color.Palette{color.RGBA{0, 0, 0, 255}, color.RGBA{9, 9, 9, 255}, color.White,
		color.NRGBA{255, 255, 255, 0}, color.NRGBA{255, 255, 255, 128}})
	each(func(x, y int, on bool) {
		switch {
		case on:
			pal16.SetColorIndex(x, y, 2)
		case (x+y)%97 == 0:
			pal16.SetColorIndex(x, y, 3) // alpha 0 白：乘上 alpha 後成黑
		}
	})
	out["paletted-alpha0"] = encodeImage(pal16)
	pal16.SetColorIndex(5, 5, 4)
	out["paletted-semi"] = encodeImage(pal16)
	rgba := image.NewRGBA(r)
	each(func(x, y int, on bool) {
		c := color.RGBA{0, 0, 0, 255}
		if on {
			c = color.RGBA{255, 255, 255, 255}
		}
		rgba.SetRGBA(x, y, c)
	})
	out["rgba"] = encodeImage(rgba)
	nrgba := image.NewNRGBA(r)
	each(func(x, y int, on bool) {
		switch {
		case on:
			nrgba.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
		case (x*7+y)%13 == 0:
			nrgba.SetNRGBA(x, y, color.NRGBA{200, 17, 3, 0})
		default:
			nrgba.SetNRGBA(x, y, color.NRGBA{0, 0, 0, 255})
		}
	})
	out["nrgba-alpha0"] = encodeImage(nrgba)
	nrgba.SetNRGBA(100, 100, color.NRGBA{255, 255, 255, 254})
	out["nrgba-semi"] = encodeImage(nrgba)
	g16 := image.NewGray16(r)
	each(func(x, y int, on bool) {
		if on {
			g16.SetGray16(x, y, color.Gray16{0xffff})
		}
	})
	out["gray16"] = encodeImage(g16)
	g16.SetGray16(3, 900, color.Gray16{0xfffe})
	out["gray16-nonbinary"] = encodeImage(g16)
	rgba64 := image.NewRGBA64(r)
	each(func(x, y int, on bool) {
		c := color.RGBA64{0, 0, 0, 0xffff}
		if on {
			c = color.RGBA64{0xffff, 0xffff, 0xffff, 0xffff}
		}
		rgba64.SetRGBA64(x, y, c)
	})
	out["rgba64"] = encodeImage(rgba64)
	gray.SetGray(511, 911, color.Gray{128})
	out["gray-nonbinary"] = encodeImage(gray)
	gray.SetGray(511, 911, color.Gray{0})
	rgba.SetRGBA(0, 0, color.RGBA{255, 255, 0, 255})
	out["rgba-yellow"] = encodeImage(rgba)
	out["small"] = encodeImage(image.NewGray(image.Rect(0, 0, 512, 896)))
	out["wide"] = encodeImage(image.NewGray(image.Rect(0, 0, 513, 912)))
	// 手工：1 位元灰階 Adam7 交錯、灰階 tRNS、截斷與 CRC。
	sample := func(x, y, c int) int {
		if bit(x, y) {
			return 1
		}
		return 0
	}
	raw := scanlines(rng, 512, 912, 1, 0, 1, false, sample)
	out["g1-interlaced"] = writePNG([]pngChunk{ihdr(512, 912, 1, 0, 1), {typ: "IDAT", data: zlibStream(raw, 6)}, {typ: "IEND"}})
	raw8 := scanlines(rng, 512, 912, 8, 0, 0, false, func(x, y, c int) int { return 255 * sample(x, y, c) })
	out["g8-trns-white"] = writePNG([]pngChunk{ihdr(512, 912, 8, 0, 0), {typ: "tRNS", data: []byte{0, 255}},
		{typ: "IDAT", data: zlibStream(raw8, 6)}, {typ: "IEND"}})
	out["g8-trns-black"] = writePNG([]pngChunk{ihdr(512, 912, 8, 0, 0), {typ: "tRNS", data: []byte{7, 0}},
		{typ: "IDAT", data: zlibStream(raw8, 6)}, {typ: "IEND"}})
	g := out["gray"]
	out["truncated"] = g[:len(g)/2]
	bad := append([]byte{}, g...)
	bad[len(bad)-20] ^= 0x40
	out["corrupt"] = bad
	out["garbage"] = []byte("not a png at all")
	return out
}

func runFont(c *sepCase) {
	if c.Compare == "" {
		c.Compare = "accept_reject"
	}
	root, cleanup := materialize(c)
	defer cleanup()
	f, err := fdtxt.LoadSeparatedFont(root)
	c.Result = errInfo(err)
	if err == nil {
		c.Result["font"] = fontSummary(f, false)
	}
}

func fontCases(rng *rand.Rand) []sepCase {
	var out []sepCase
	doc := marshal(validFontDoc())
	atlases := atlasVariants(rng)
	names := make([]string, 0, len(atlases))
	for k := range atlases {
		names = append(names, k)
	}
	sort.Strings(names)
	files := func(docRaw, atlas []byte) []fileJSON {
		fs := []fileJSON{{Path: "fdother_004/font.json", Hex: hex.EncodeToString(docRaw)}}
		if atlas != nil {
			fs = append(fs, fileJSON{Path: "fdother_004/atlas.png", Hex: hex.EncodeToString(atlas)})
		}
		return fs
	}
	for _, name := range names {
		malformed := name == "truncated" || name == "corrupt" || name == "garbage"
		out = append(out, sepCase{Kind: "atlas-" + name, Root: "tmp", Compare: compareMode(!malformed),
			Files: files(doc, atlases[name])})
	}
	mut := func(kind string, f func(d *genFont)) {
		d := validFontDoc()
		f(&d)
		out = append(out, sepCase{Kind: kind, Root: "tmp", Compare: "exact", Files: files(marshal(d), atlases["paletted1"])})
	}
	mut("doc-schema", func(d *genFont) { d.SchemaVersion = 0 })
	mut("doc-kind", func(d *genFont) { d.Kind = "font" })
	mut("doc-asset", func(d *genFont) { d.AssetID = "font/FDOTHER_4" })
	mut("doc-status", func(d *genFont) { d.Status = "" })
	mut("doc-evidence", func(d *genFont) { d.Evidence = "Confirmed" })
	mut("doc-glyphs", func(d *genFont) { d.GlyphCount = 1823 })
	mut("doc-cell-w", func(d *genFont) { d.CellWidth = 8 })
	mut("doc-cell-h", func(d *genFont) { d.CellHeight = 15 })
	mut("doc-columns", func(d *genFont) { d.Columns = 16 })
	mut("doc-rows", func(d *genFont) { d.Rows = 58 })
	mut("doc-atlas", func(d *genFont) { d.Atlas = "./atlas.png" })
	mut("doc-src-file", func(d *genFont) { d.Source.File = "FDTXT.DAT" })
	mut("doc-src-resource", func(d *genFont) { d.Source.Resource = 5 })
	mut("doc-src-size", func(d *genFont) { d.Source.Size = 0 })
	mut("doc-src-md5", func(d *genFont) { d.Source.MD5 = "" })
	mut("doc-src-sha", func(d *genFont) { d.Source.SHA256 = "x" })
	mut("doc-src-raw", func(d *genFont) { d.Source.RawSize = 58367 })
	raw := func(kind string, docRaw []byte) {
		out = append(out, sepCase{Kind: kind, Root: "tmp", Files: files(docRaw, atlases["paletted1"])})
	}
	raw("doc-fold", bytes.Replace(doc, []byte(`"glyph_count"`), []byte(`"GLYPH_COUNT"`), 1))
	raw("doc-dup", bytes.Replace(doc, []byte(`"rows":57`), []byte(`"rows":1,"rows":57`), 1))
	raw("doc-null-rows", bytes.Replace(doc, []byte(`"rows":57`), []byte(`"rows":null`), 1))
	raw("doc-type", bytes.Replace(doc, []byte(`"rows":57`), []byte(`"rows":"57"`), 1))
	raw("doc-source-type", bytes.Replace(doc, []byte(`"source":{`), []byte(`"source":7,"x":{`), 1))
	raw("doc-syntax", doc[:len(doc)-1])
	raw("doc-empty", []byte{})
	for i := 0; i < 60; i++ {
		raw("doc-mutated", mutateJSON(rng, doc))
	}
	pal := atlases["paletted1"]
	out = append(out,
		sepCase{Kind: "fs-no-atlas", Root: "tmp", Files: files(doc, nil)},
		sepCase{Kind: "fs-atlas-dir", Root: "tmp", Files: files(doc, nil), Dirs: []string{"fdother_004/atlas.png"}},
		sepCase{Kind: "fs-no-json", Root: "tmp", Dirs: []string{"fdother_004"}},
		sepCase{Kind: "fs-json-dir", Root: "tmp", Dirs: []string{"fdother_004/font.json"}},
		sepCase{Kind: "fs-no-dir", Root: "tmp"},
		sepCase{Kind: "fs-empty-root", Root: "empty", Files: files(doc, pal)},
		sepCase{Kind: "fs-clean-dotdot", Root: "/missing/..", Files: files(doc, pal)},
	)
	return out
}

// ---------------------------------------------------------------------------
// 真實原版檔
// ---------------------------------------------------------------------------

func fileSHA(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return digest(data)
}

func realFont(f *fdtxt.Font) map[string]any {
	out := fontSummary(f, true)
	blits := make([]string, f.GlyphCount())
	for g := range blits {
		a := make([]byte, 20*18+4)
		b := make([]byte, 20*18+4)
		for i := range a {
			a[i], b[i] = byte(i%5), byte(i%7)
		}
		errA := f.BlitGlyph(a, 20, 2, g, 0x2a)
		errB := f.BlitNativeGlyph(b, 20, 21, g, fdtxt.NativeGlyphStyle{Foreground: 0xcd, Shadow: 0x4c, Background: byte(g % 3)})
		if errA != nil || errB != nil {
			panic(fmt.Sprint(errA, errB))
		}
		blits[g] = digest(append(a, b...))[:16]
	}
	out["blit_digests"] = blits
	return out
}

func realSection(flame2, separated string) map[string]any {
	out := map[string]any{}
	if flame2 != "" {
		path := filepath.Join(flame2, "FDTXT.DAT")
		count, err := fdother.ArchiveResourceCount(path)
		if err != nil {
			panic(err)
		}
		var resources []map[string]any
		for r := 0; r < count; r++ {
			data, err := fdother.ReadResource(path, r)
			if err != nil {
				panic(err)
			}
			resources = append(resources, map[string]any{"resource": r, "data_sha256": digest(data), "result": parseSummary(data)})
		}
		out["fdtxt_dat"] = map[string]any{"path": "FDTXT.DAT", "sha256": fileSHA(path), "resources": resources}
		fpath := filepath.Join(flame2, "FDOTHER.DAT")
		data, err := fdother.ReadResource(fpath, 4)
		if err != nil {
			panic(err)
		}
		font, err := fdtxt.ParseFont(data)
		if err != nil {
			panic(err)
		}
		out["fdother_font"] = map[string]any{"path": "FDOTHER.DAT", "sha256": fileSHA(fpath), "resource": 4,
			"data_sha256": digest(data), "result": realFont(font)}
	}
	if separated != "" {
		textRoot := filepath.Join(separated, "text")
		var texts []map[string]any
		for r := 0; r <= 35; r++ {
			rel := fmt.Sprintf("text/FDTXT_%03d/resource.json", r)
			row := map[string]any{"resource": r, "path": rel}
			if data, err := os.ReadFile(filepath.Join(separated, rel)); err == nil {
				row["sha256"] = digest(data)
			}
			s, err := fdtxt.LoadSeparatedResource(textRoot, r)
			row["result"] = errInfo(err)
			if err == nil {
				row["result"].(map[string]any)["strings"] = stringsSummary(s)
			}
			texts = append(texts, row)
		}
		out["separated_text"] = texts
		font, err := fdtxt.LoadSeparatedFont(filepath.Join(separated, "fonts"))
		fontRow := map[string]any{"json_path": "fonts/fdother_004/font.json", "atlas_path": "fonts/fdother_004/atlas.png",
			"json_sha256":  fileSHA(filepath.Join(separated, "fonts/fdother_004/font.json")),
			"atlas_sha256": fileSHA(filepath.Join(separated, "fonts/fdother_004/atlas.png")), "result": errInfo(err)}
		if err == nil {
			fontRow["result"].(map[string]any)["font"] = realFont(font)
		}
		out["separated_font"] = fontRow
		var pngs []map[string]any
		for _, rel := range []string{"fonts/fdother_004/atlas.png", "fonts/atlas.png"} {
			data, err := os.ReadFile(filepath.Join(separated, rel))
			if err != nil {
				panic(err)
			}
			pngs = append(pngs, map[string]any{"path": rel, "sha256": digest(data), "result": pngSummary(data)})
		}
		out["png"] = pngs
	}
	return out
}

// ---------------------------------------------------------------------------

type parseCase struct {
	Index   int            `json:"index"`
	Kind    string         `json:"kind"`
	DataHex string         `json:"data_hex"`
	Result  map[string]any `json:"result"`
}

type fontCase struct {
	Index   int            `json:"index"`
	Kind    string         `json:"kind"`
	DataHex string         `json:"data_hex"`
	Result  map[string]any `json:"result"`
	Blits   []blitOp       `json:"blits"`
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	flame2 := flag.String("flame2", "", "原版 FLAME2 目錄（FDTXT.DAT、FDOTHER.DAT）")
	separated := flag.String("separated", "", "分離素材根目錄（generated-assets/fd2-original-b97caf22）")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "需要 -out")
		os.Exit(2)
	}
	rng := rand.New(rand.NewPCG(*seed, 0x46445458))

	var parses []parseCase
	for _, b := range fixedParse {
		parses = append(parses, parseCase{Kind: "fixed", DataHex: hex.EncodeToString(b)})
	}
	for i := 0; i < 600; i++ {
		kind, b := generateParse(i, rng)
		parses = append(parses, parseCase{Kind: kind, DataHex: hex.EncodeToString(b)})
	}
	for i := range parses {
		parses[i].Index = i
		data, _ := hex.DecodeString(parses[i].DataHex)
		parses[i].Result = parseSummary(data)
	}

	var fonts []fontCase
	for i := 0; i < 160; i++ {
		kind, data := generateFont(i, rng)
		c := fontCase{Index: i, Kind: kind, DataHex: hex.EncodeToString(data), Blits: []blitOp{}}
		f, err := fdtxt.ParseFont(data)
		c.Result = errInfo(err)
		if err == nil {
			c.Result["font"] = fontSummary(f, true)
			for k := 0; k < 8; k++ {
				op := randomBlit(rng, f.GlyphCount())
				runBlit(f, &op)
				c.Blits = append(c.Blits, op)
			}
		}
		fonts = append(fonts, c)
	}

	texts := textCases(rng)
	for i := range texts {
		texts[i].Index = i
		runText(&texts[i])
	}
	pngs := pngCases(rng)
	for i := range pngs {
		pngs[i].Index = i
		if pngs[i].Compare == "" {
			pngs[i].Compare = "accept_reject"
		}
		data, _ := hex.DecodeString(pngs[i].PNGHex)
		pngs[i].Result = pngSummary(data)
	}
	sepFonts := fontCases(rng)
	for i := range sepFonts {
		sepFonts[i].Index = i
		runFont(&sepFonts[i])
	}

	doc := map[string]any{
		"schema": "fd2-golden-fdtxt/1", "seed": *seed, "go_version": "go1.22.12",
		"parse_cases": parses, "font_cases": fonts, "separated_text_cases": texts,
		"separated_font_cases": sepFonts, "png_cases": pngs, "real": realSection(*flame2, *separated),
	}
	data, err := json.Marshal(doc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("fd2-golden-fdtxt: parse=%d font=%d text=%d sepfont=%d png=%d sha256=%s\n",
		len(parses), len(fonts), len(texts), len(sepFonts), len(pngs), digest(data))
}
