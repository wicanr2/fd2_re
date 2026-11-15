// rust_r5_draw_inventory：R5 階段 0（docs/goal/121）的繪圖函式盤點。
//
// 用法（由 tools/rust_r5_golden_frames.sh 在 Docker 內呼叫）：
//
//	rust_r5_draw_inventory -deps <go list -export -deps -json ./cmd/fd2 輸出> -repo <儲存庫根>
//	  -cover <名稱>=<-covermode=count 的 cover.out> [-cover ...] -out <輸出目錄>
//
// 做法：
//  1. 以 go/types 檢查 remake/cmd/fd2 的正式程式（不含 _test.go），建靜態呼叫邊：函式本體（含閉包）
//     內每個解析到本套件 *types.Func 的識別字都算一條邊，包含方法值與函式值（保守）。介面方法呼叫
//     無法靜態解析，另計數。
//  2. 從 (*Game).Draw 出發走這張圖，只經過「覆蓋率顯示有執行」的函式。HD 指 modern_*、
//     player_poster*、player_toolbar、player_theme_loader、player_hd_quality、player_interface*、
//     player_speed_controls、ebiten_screen_filter 等檔，以及名稱含 modernHD／ModernHD 的函式；HD 節點
//     照樣往下走（原版模式下多半只跑守門判斷），另列一組，不計入非 HD 統計。
//  3. 每個函式記下檔案、行範圍、各覆蓋率檔的進入次數（函式本體第一個區塊的 count）、直接使用的
//     Ebiten API，以及舊後備路徑（TTF text、ColorM、DebugPrint、vector、色塊後備 drawUnit、半透明
//     Fill、ColorScale）的呼叫點與該行所在覆蓋區塊的 count。
//
// 輸出：executed-draw-functions.json（逐函式）、draw-inventory.json（分組與 Ebiten 語意）。
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const mainImportPath = "github.com/wicanr2/fd2_re/remake/cmd/fd2"
const ebitenPrefix = "github.com/hajimehoshi/ebiten/v2"

type coverFlags []string

func (c *coverFlags) String() string     { return strings.Join(*c, ",") }
func (c *coverFlags) Set(v string) error { *c = append(*c, v); return nil }

var (
	flagDeps  = flag.String("deps", "", "go list -export -deps -json ./cmd/fd2 的輸出檔")
	flagRepo  = flag.String("repo", "", "儲存庫根目錄（只用來寫相對路徑）")
	flagOut   = flag.String("out", "", "輸出目錄")
	flagCover coverFlags
)

type listPkg struct {
	ImportPath string
	Dir        string
	Export     string
	GoFiles    []string
}

type fn struct {
	obj       *types.Func
	Name      string
	File      string
	Start     int
	End       int
	decl      *ast.FuncDecl
	callees   map[*types.Func]bool
	ops       map[string]int
	sites     []site
	hd        bool
	ifaceCall int
	dynCalls  map[string]int // 經 func 型別變數／欄位的呼叫（靜態圖解析不到目標）
}

type site struct {
	Kind   string         `json:"kind"`
	Line   int            `json:"line"`
	Detail string         `json:"detail,omitempty"`
	Counts map[string]int `json:"counts"`
	pos    token.Position
}

type block struct {
	sl, sc, el, ec, count int
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	flag.Var(&flagCover, "cover", "名稱=cover.out（-covermode=count），可重複")
	flag.Parse()
	if *flagDeps == "" || *flagOut == "" || len(flagCover) == 0 {
		fmt.Fprintln(os.Stderr, "需要 -deps、-out 與至少一個 -cover")
		os.Exit(2)
	}
	fset, files, pkg, info := load(*flagDeps)
	globalFset = fset
	funcs := collect(fset, files, pkg, info)
	covers := map[string]map[string][]block{}
	coverNames := []string{}
	for _, c := range flagCover {
		name, path, ok := strings.Cut(c, "=")
		if !ok {
			panic("-cover 要寫成 名稱=路徑：" + c)
		}
		covers[name] = loadCover(path)
		coverNames = append(coverNames, name)
	}
	emitWith(funcs, covers, coverNames)
}

func load(depsPath string) (*token.FileSet, []*ast.File, *types.Package, *types.Info) {
	f, err := os.Open(depsPath)
	must(err)
	defer f.Close()
	dec := json.NewDecoder(f)
	exports := map[string]string{}
	var main *listPkg
	for {
		var p listPkg
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else {
			must(err)
		}
		if p.Export != "" {
			exports[p.ImportPath] = p.Export
		}
		if p.ImportPath == mainImportPath {
			pp := p
			main = &pp
		}
	}
	if main == nil {
		panic("go list 輸出沒有 " + mainImportPath)
	}
	fset := token.NewFileSet()
	names := append([]string{}, main.GoFiles...)
	sort.Strings(names)
	var files []*ast.File
	for _, n := range names {
		af, err := parser.ParseFile(fset, filepath.Join(main.Dir, n), nil, parser.ParseComments)
		must(err)
		files = append(files, af)
	}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	lookup := func(path string) (io.ReadCloser, error) {
		e, ok := exports[path]
		if !ok {
			return nil, fmt.Errorf("沒有 %s 的匯出資料", path)
		}
		return os.Open(e)
	}
	conf := types.Config{Importer: importer.ForCompiler(fset, "gc", lookup)}
	pkg, err := conf.Check(mainImportPath, fset, files, info)
	must(err)
	return fset, files, pkg, info
}

var hdFile = regexp.MustCompile(`^(modern_|player_poster|player_toolbar|player_theme_loader|player_hd_quality|player_interface|player_speed_controls|ebiten_screen_filter)`)

func funcKey(obj *types.Func) string {
	sig := obj.Type().(*types.Signature)
	if recv := sig.Recv(); recv != nil {
		t := recv.Type()
		ptr := ""
		if p, ok := t.(*types.Pointer); ok {
			t, ptr = p.Elem(), "*"
		}
		name := t.String()
		if n, ok := t.(*types.Named); ok {
			name = n.Obj().Name()
		}
		return "(" + ptr + name + ")." + obj.Name()
	}
	return obj.Name()
}

func ebitenOp(obj types.Object) string {
	if obj == nil || obj.Pkg() == nil || !strings.HasPrefix(obj.Pkg().Path(), ebitenPrefix) {
		return ""
	}
	short := strings.TrimPrefix(obj.Pkg().Path(), ebitenPrefix+"/")
	if obj.Pkg().Path() == ebitenPrefix {
		short = "ebiten"
	}
	switch o := obj.(type) {
	case *types.Func:
		if recv := o.Type().(*types.Signature).Recv(); recv != nil {
			t := recv.Type()
			if p, ok := t.(*types.Pointer); ok {
				t = p.Elem()
			}
			if n, ok := t.(*types.Named); ok {
				return short + "." + n.Obj().Name() + "." + o.Name()
			}
		}
		return short + "." + o.Name()
	case *types.Var:
		if o.IsField() {
			return short + ".field." + o.Name()
		}
	case *types.Const:
		return short + "." + o.Name()
	}
	return ""
}

func collect(fset *token.FileSet, files []*ast.File, pkg *types.Package, info *types.Info) map[*types.Func]*fn {
	funcs := map[*types.Func]*fn{}
	for _, af := range files {
		base := filepath.Base(fset.Position(af.Pos()).Filename)
		for _, d := range af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			obj, _ := info.Defs[fd.Name].(*types.Func)
			if obj == nil {
				continue
			}
			f := &fn{obj: obj, Name: funcKey(obj), File: base, decl: fd,
				Start: fset.Position(fd.Pos()).Line, End: fset.Position(fd.End()).Line,
				callees: map[*types.Func]bool{}, ops: map[string]int{}, dynCalls: map[string]int{}}
			f.hd = hdFile.MatchString(base) || strings.Contains(strings.ToLower(obj.Name()), "modernhd")
			funcs[obj] = f
		}
	}
	for _, f := range funcs {
		ast.Inspect(f.decl.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.Ident:
				obj := info.Uses[x]
				if callee, ok := obj.(*types.Func); ok {
					if callee.Pkg() == pkg {
						if o := callee.Origin(); o != nil {
							callee = o
						}
						if _, known := funcs[callee]; known {
							f.callees[callee] = true
						} else if sig, ok := callee.Type().(*types.Signature); ok && sig.Recv() != nil {
							if _, isIface := sig.Recv().Type().Underlying().(*types.Interface); isIface {
								f.ifaceCall++
							}
						}
					}
				}
				if op := ebitenOp(obj); op != "" {
					f.ops[op]++
				}
			case *ast.CallExpr:
				classifySite(fset, info, f, x)
				if v, ok := calleeObj(info, x).(*types.Var); ok {
					if _, isFunc := v.Type().Underlying().(*types.Signature); isFunc && v.Pkg() == pkg {
						f.dynCalls[v.Name()]++
					}
				}
			case *ast.SelectorExpr:
				if x.Sel.Name == "ColorM" {
					if sel := info.Selections[x]; sel != nil && sel.Kind() == types.FieldVal {
						if op := ebitenOp(sel.Obj()); op != "" {
							f.sites = append(f.sites, site{Kind: "ColorM", pos: fset.Position(x.Pos())})
						}
					}
				}
			}
			return true
		})
	}
	return funcs
}

func calleeObj(info *types.Info, call *ast.CallExpr) types.Object {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return info.Uses[fun]
	case *ast.SelectorExpr:
		return info.Uses[fun.Sel]
	}
	return nil
}

// classifySite 記下舊後備與 RGBA 路徑的呼叫點。
func classifySite(fset *token.FileSet, info *types.Info, f *fn, call *ast.CallExpr) {
	obj := calleeObj(info, call)
	if obj == nil {
		return
	}
	op := ebitenOp(obj)
	kind, detail := "", ""
	switch {
	case strings.HasPrefix(op, "text."):
		kind = "TTF text（ebiten/v2/text）"
	case strings.HasPrefix(op, "ebitenutil.Debug"):
		kind = "ebitenutil.DebugPrint"
	case strings.HasPrefix(op, "vector."):
		kind = "vector"
	case op == "ebiten.ColorScale.Scale" || op == "ebiten.ColorScale.ScaleWithColor" || op == "ebiten.ColorScale.ScaleAlpha":
		kind = "ColorScale"
	case op == "ebiten.Image.Fill":
		kind = "Fill"
		if len(call.Args) == 1 {
			if a, ok := literalAlpha(info, call.Args[0]); ok {
				detail = "alpha=" + a
				if a == "255" {
					kind = "Fill（不透明）"
				} else {
					kind = "Fill（半透明）"
				}
			} else {
				detail = "alpha 非常數"
			}
		}
	}
	if kind == "" {
		if fobj, ok := obj.(*types.Func); ok && fobj.Pkg() != nil && fobj.Pkg().Path() == mainImportPath {
			switch funcKey(fobj) {
			case "(*Font).Draw":
				kind = "(*Font).Draw（TTF 後備文字）"
			case "drawUnit":
				kind = "drawUnit（色塊後備）"
			case "titlePaletteBlend":
				kind = "titlePaletteBlend（ColorM 後備）"
			}
		}
	}
	if kind != "" {
		f.sites = append(f.sites, site{Kind: kind, Detail: detail, pos: fset.Position(call.Pos())})
	}
}

// literalAlpha 取 color.RGBA{...} 字面量的 A（第 4 個值或 A: 欄位）。
func literalAlpha(info *types.Info, e ast.Expr) (string, bool) {
	if u, ok := e.(*ast.UnaryExpr); ok && u.Op == token.AND {
		e = u.X
	}
	if sel, ok := e.(*ast.SelectorExpr); ok {
		if obj := info.Uses[sel.Sel]; obj != nil && obj.Pkg() != nil && (obj.Pkg().Path() == "image/color" || obj.Pkg().Path() == "image") {
			switch obj.Name() {
			case "Black", "White", "Opaque":
				return "255", true
			case "Transparent":
				return "0", true
			}
		}
	}
	cl, ok := e.(*ast.CompositeLit)
	if !ok || len(cl.Elts) == 0 {
		return "", false
	}
	var ae ast.Expr
	for i, el := range cl.Elts {
		if kv, ok := el.(*ast.KeyValueExpr); ok {
			if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "A" {
				ae = kv.Value
			}
		} else if i == 3 {
			ae = el
		}
	}
	if ae == nil {
		return "", false
	}
	tv, ok := info.Types[ae]
	if !ok || tv.Value == nil {
		return "", false
	}
	v, ok := constant.Int64Val(constant.ToInt(tv.Value))
	if !ok {
		return "", false
	}
	return strconv.FormatInt(v, 10), true
}

func loadCover(path string) map[string][]block {
	f, err := os.Open(path)
	must(err)
	defer f.Close()
	out := map[string][]block{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	first := true
	for sc.Scan() {
		line := sc.Text()
		if first {
			first = false
			if line != "mode: count" && line != "mode: atomic" {
				panic(path + " 不是 -covermode=count／atomic：" + line)
			}
			continue
		}
		colon := strings.LastIndex(line, ":")
		file := line[:colon]
		var b block
		var stmts int
		if _, err := fmt.Sscanf(line[colon+1:], "%d.%d,%d.%d %d %d", &b.sl, &b.sc, &b.el, &b.ec, &stmts, &b.count); err != nil {
			panic(err)
		}
		if !strings.HasPrefix(file, mainImportPath+"/") {
			continue
		}
		base := strings.TrimPrefix(file, mainImportPath+"/")
		// 同一區塊在多個測試二進位出現時會重複列出；count 相加。
		out[base] = append(out[base], b)
	}
	must(sc.Err())
	return out
}

func before(l1, c1, l2, c2 int) bool { return l1 < l2 || (l1 == l2 && c1 <= c2) }

// entryCount：函式本體內最早開始的覆蓋區塊之 count（＝進入次數）；executed：任一區塊 count>0。
func funcCoverage(blocks []block, f *fn, fset *token.FileSet) (entry int, executed bool, covered, total int) {
	lb := fset.Position(f.decl.Body.Lbrace)
	rb := fset.Position(f.decl.Body.Rbrace)
	best := -1
	for i, b := range blocks {
		if !before(lb.Line, lb.Column, b.sl, b.sc) || !before(b.el, b.ec, rb.Line, rb.Column+1) {
			continue
		}
		total++
		if b.count > 0 {
			executed = true
			covered++
		}
		if best < 0 || before(b.sl, b.sc, blocks[best].sl, blocks[best].sc) && !(b.sl == blocks[best].sl && b.sc == blocks[best].sc) {
			best = i
		}
	}
	if best >= 0 {
		// 同位置的重複區塊（多個 profile 合併）取最大。
		for _, b := range blocks {
			if b.sl == blocks[best].sl && b.sc == blocks[best].sc && b.count > entry {
				entry = b.count
			}
		}
	}
	return
}

func lineCount(blocks []block, pos token.Position) int {
	n := 0
	for _, b := range blocks {
		if before(b.sl, b.sc, pos.Line, pos.Column) && before(pos.Line, pos.Column, b.el, b.ec) && b.count > n {
			n = b.count
		}
	}
	return n
}

type groupRule struct {
	name  string
	files *regexp.Regexp
	names *regexp.Regexp
}

var groups = []groupRule{
	{"素材路徑與時鐘（非繪圖）", regexp.MustCompile(`^(assets|save|separated_asset_path_native|native_bios_clock|player_game_clock)\.go$`), regexp.MustCompile(`^$`)},
	{"轉場與調色盤", regexp.MustCompile(`^(native_indexed_transition|native_palette_|native_24b4d|native_transient_|native_turn_staging)`),
		regexp.MustCompile(`(?i)transition|palette|fade|reveal|flash|ramp|pulse|dac`)},
	{"標題與開場", regexp.MustCompile(`^(title|title_menu_state|native_load_slots_ui|native_continue_handoff)\.go$`),
		regexp.MustCompile(`(?i)title|opening|cutscene|logo|publisher`)},
	{"結局", regexp.MustCompile(`^ending_preview\.go$`), regexp.MustCompile(`(?i)ending`)},
	{"城鎮與設施", regexp.MustCompile(`^(native_town_|native_shop_|native_church_|native_hotel_ui|native_preparation_ui|native_class_ui|native_item_panel_ui)`),
		regexp.MustCompile(`(?i)town|shop|church|prep|hotel|class|facility|departure|campaign`)},
	{"HUD 與選單", regexp.MustCompile(`^(action_overlay_runtime|native_system_info_|native_battle_panel_|native_map_unit_window|native_player_|native_battle_name|native_command_locale)`),
		regexp.MustCompile(`(?i)ring|menu|panel|hud|status|systeminfo|grid|banner|overlay|window|cursor|endturn|transientbattlemessage|command`)},
	{"戰鬥演出", regexp.MustCompile(`^(native_physical_|native_attack_assets|native_command.*presentation|native_action_death_present|native_battle_staging_present|native_staging_present|native_ai_|battle_scene_layers|native_defeat|native_death_|native_ch28_post_present)`),
		regexp.MustCompile(`(?i)attack|atk|physical|death|cmd\d|spell|heal|modifier|defeat|strike|battlescene|weapon|postpresent`)},
	{"對話與文字", regexp.MustCompile(`^(native_story_dialogue|native_event_dialogue|native_level_up_dialogue|native_confirmation_mouth|font|locale)`),
		regexp.MustCompile(`(?i)dialog|text|portrait|mouth|font|glyph|story`)},
	{"地圖與單位", regexp.MustCompile(`^(native_map_|native_spawn_intro|native_story_map|native_unit_present|battle_snapshot|native_ch23_post)`),
		regexp.MustCompile(`(?i)map|terrain|unit|sprite|tile|spawn|camera|stage`)},
	{"播放器外框", regexp.MustCompile(`^player_`), regexp.MustCompile(`(?i)player|help`)},
}

const hdGroup = "HD 檔（原版模式下有執行，多為守門）"

func groupOf(f *fn) string {
	if f.hd {
		return hdGroup
	}
	if f.Name == "(*Game).Draw" {
		return "Game.Draw 本體（分派、正規化地圖層、舊對話框與 fade）"
	}
	if f.File != "main.go" {
		for _, g := range groups {
			if g.files.MatchString(f.File) {
				return g.name
			}
		}
	}
	for _, g := range groups {
		if g.names.MatchString(f.obj.Name()) {
			return g.name
		}
	}
	return "共用與其他"
}

type funcOut struct {
	Func     string         `json:"func"`
	File     string         `json:"file"`
	Lines    []int          `json:"lines"`
	LineSpan int            `json:"line_count"`
	Group    string         `json:"group"`
	Calls    map[string]int `json:"entry_counts"`
	Covered  string         `json:"covered_blocks"`
	Ebiten   []string       `json:"ebiten_api,omitempty"`
	Sites    []site         `json:"legacy_or_rgba_sites,omitempty"`
	Dynamic  map[string]int `json:"func_value_calls,omitempty"`
}

// globalFset 是 load 用的 FileSet；覆蓋率比對要用函式本體的行列。
var globalFset *token.FileSet

func emitWith(funcs map[*types.Func]*fn, covers map[string]map[string][]block, names []string) {
	fset := globalFset
	type cov struct {
		entry    map[string]int
		executed bool
		covered  int
		total    int
	}
	covOf := map[*fn]*cov{}
	for _, f := range funcs {
		c := &cov{entry: map[string]int{}}
		for _, name := range names {
			e, ex, cv, tot := funcCoverage(covers[name][f.File], f, fset)
			c.entry[name] = e
			if ex {
				c.executed = true
			}
			if cv > c.covered {
				c.covered = cv
			}
			c.total = tot
		}
		covOf[f] = c
	}
	var root *fn
	for _, f := range funcs {
		if f.Name == "(*Game).Draw" {
			root = f
		}
	}
	if root == nil {
		panic("找不到 (*Game).Draw")
	}
	reached := map[*fn]bool{}
	hdReached := map[*fn]bool{}
	unexecutedEdges := map[*fn]bool{}
	queue := []*fn{root}
	reached[root] = true
	for len(queue) > 0 {
		f := queue[0]
		queue = queue[1:]
		for callee := range f.callees {
			c := funcs[callee]
			if c == nil || reached[c] || hdReached[c] {
				continue
			}
			if !covOf[c].executed {
				unexecutedEdges[c] = true
				continue
			}
			if c.hd {
				// HD 檔的函式在原版模式下多半只跑守門判斷；照樣往下走，另列一組。
				hdReached[c] = true
			} else {
				reached[c] = true
			}
			queue = append(queue, c)
		}
	}
	outs := []funcOut{}
	groupLines := map[string]int{}
	groupFuncs := map[string]int{}
	groupOps := map[string]map[string]int{}
	allOps := map[string]int{}
	type legacy struct {
		Kind     string         `json:"kind"`
		Func     string         `json:"func"`
		File     string         `json:"file"`
		Line     int            `json:"line"`
		Detail   string         `json:"detail,omitempty"`
		Counts   map[string]int `json:"counts"`
		Executed bool           `json:"executed"`
	}
	legacySites := []legacy{}
	ifaceCalls, dynTotal := 0, 0
	all := []*fn{}
	for f := range reached {
		all = append(all, f)
	}
	for f := range hdReached {
		all = append(all, f)
	}
	for _, f := range all {
		c := covOf[f]
		g := groupOf(f)
		o := funcOut{Func: f.Name, File: f.File, Lines: []int{f.Start, f.End}, LineSpan: f.End - f.Start + 1,
			Group: g, Calls: c.entry, Covered: fmt.Sprintf("%d/%d", c.covered, c.total)}
		ifaceCalls += f.ifaceCall
		if len(f.dynCalls) > 0 {
			o.Dynamic = f.dynCalls
			dynTotal += len(f.dynCalls)
		}
		for op := range f.ops {
			o.Ebiten = append(o.Ebiten, op)
			allOps[op] += 1
			if groupOps[g] == nil {
				groupOps[g] = map[string]int{}
			}
			groupOps[g][op]++
		}
		sort.Strings(o.Ebiten)
		for _, s := range f.sites {
			s.Line = s.pos.Line
			s.Counts = map[string]int{}
			executed := false
			for _, name := range names {
				n := lineCount(covers[name][f.File], s.pos)
				s.Counts[name] = n
				if n > 0 {
					executed = true
				}
			}
			o.Sites = append(o.Sites, s)
			legacySites = append(legacySites, legacy{Kind: s.Kind, Func: f.Name, File: f.File, Line: s.Line, Detail: s.Detail, Counts: s.Counts, Executed: executed})
		}
		groupLines[g] += o.LineSpan
		groupFuncs[g]++
		outs = append(outs, o)
	}
	sort.Slice(outs, func(i, j int) bool {
		if outs[i].File != outs[j].File {
			return outs[i].File < outs[j].File
		}
		return outs[i].Lines[0] < outs[j].Lines[0]
	})
	sort.Slice(legacySites, func(i, j int) bool {
		if legacySites[i].File != legacySites[j].File {
			return legacySites[i].File < legacySites[j].File
		}
		return legacySites[i].Line < legacySites[j].Line
	})
	hdList := []string{}
	for f := range hdReached {
		hdList = append(hdList, f.File+":"+f.Name)
	}
	sort.Strings(hdList)
	notRun := []string{}
	for f := range unexecutedEdges {
		if !reached[f] {
			notRun = append(notRun, f.File+":"+f.Name)
		}
	}
	sort.Strings(notRun)
	totalLines, nonHDFuncs, nonHDLines := 0, 0, 0
	for _, o := range outs {
		totalLines += o.LineSpan
		if o.Group != hdGroup {
			nonHDFuncs++
			nonHDLines += o.LineSpan
		}
	}
	write := func(name string, v any) {
		raw, err := json.MarshalIndent(v, "", " ")
		must(err)
		must(os.WriteFile(filepath.Join(*flagOut, name), append(raw, '\n'), 0o644))
	}
	method := "go/types 靜態呼叫邊（含函式值，介面呼叫不解析）從 (*Game).Draw 出發，只經過覆蓋率有執行的函式（HD 檔另列一組）；進入次數取函式本體第一個覆蓋區塊的 count"
	write("executed-draw-functions.json", map[string]any{
		"schema":                "fd2_rust_r5_executed_draw_functions/1",
		"plan":                  "docs/goal/121-goal-rust-r5-original-art-shell-20261115.md 階段 0",
		"producer":              "tools/rust_r5_draw_inventory（tools/rust_r5_golden_frames.sh）",
		"method":                method,
		"coverage_profiles":     names,
		"functions":             outs,
		"function_count":        len(outs),
		"line_count":            totalLines,
		"non_hd_functions":      nonHDFuncs,
		"non_hd_lines":          nonHDLines,
		"interface_calls":       ifaceCalls,
		"func_value_call_sites": dynTotal,
		"hd_executed_reached":   hdList,
		"draw_tree_not_run":     notRun,
		"legacy_or_rgba_sites":  legacySites,
	})
	groupNames := []string{}
	for g := range groupLines {
		groupNames = append(groupNames, g)
	}
	sort.Slice(groupNames, func(i, j int) bool { return groupLines[groupNames[i]] > groupLines[groupNames[j]] })
	type groupOut struct {
		Group     string         `json:"group"`
		Functions int            `json:"functions"`
		Lines     int            `json:"lines"`
		Files     map[string]int `json:"files"`
		Ebiten    map[string]int `json:"ebiten_api_functions"`
	}
	gouts := []groupOut{}
	for _, g := range groupNames {
		files := map[string]int{}
		for _, o := range outs {
			if o.Group == g {
				files[o.File]++
			}
		}
		gouts = append(gouts, groupOut{Group: g, Functions: groupFuncs[g], Lines: groupLines[g], Files: files, Ebiten: groupOps[g]})
	}
	write("draw-inventory.json", map[string]any{
		"schema":            "fd2_rust_r5_draw_inventory/1",
		"method":            method,
		"grouping":          "非 main.go 的函式依檔名歸組，main.go 與未命中的檔依函式名關鍵字；見 tools/rust_r5_draw_inventory/main.go 的 groups",
		"groups":            gouts,
		"ebiten_api_totals": allOps,
		"function_count":    len(outs),
		"line_count":        totalLines,
		"non_hd_functions":  nonHDFuncs,
		"non_hd_lines":      nonHDLines,
	})
	fmt.Printf("STAT draw-tree functions=%d lines=%d non_hd_functions=%d non_hd_lines=%d hd_reached=%d not_run_edges=%d interface_calls=%d func_value_calls=%d\n",
		len(outs), totalLines, nonHDFuncs, nonHDLines, len(hdList), len(notRun), ifaceCalls, dynTotal)
	for _, g := range gouts {
		fmt.Printf("GROUP %s functions=%d lines=%d\n", g.Group, g.Functions, g.Lines)
	}
	kindTotal, kindRun := map[string]int{}, map[string]int{}
	kinds := []string{}
	for _, s := range legacySites {
		if kindTotal[s.Kind] == 0 {
			kinds = append(kinds, s.Kind)
		}
		kindTotal[s.Kind]++
		if s.Executed {
			kindRun[s.Kind]++
			fmt.Printf("LEGACY 執行 %s %s:%d %s %s %v\n", s.Kind, s.File, s.Line, s.Func, s.Detail, s.Counts)
		}
	}
	sort.Strings(kinds)
	for _, k := range kinds {
		fmt.Printf("LEGACY 小計 %s 呼叫點=%d 有執行=%d\n", k, kindTotal[k], kindRun[k])
	}
}
