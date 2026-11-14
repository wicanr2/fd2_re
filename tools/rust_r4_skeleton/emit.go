package main

// 每個 Go 檔一個 Rust 子模組：型別、常數、套件層變數與選定函式的簽名。

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type fnManifest struct {
	Name   string   `json:"name"`
	Recv   string   `json:"recv,omitempty"`
	Rust   string   `json:"rust,omitempty"`
	Status string   `json:"status"`
	Line   int      `json:"go_line"`
	Class  string   `json:"class,omitempty"`
	Cover  *float64 `json:"cover,omitempty"`
	Group  string   `json:"group,omitempty"`
	Calls  []string `json:"calls,omitempty"`
	Notes  []string `json:"notes,omitempty"`
}

type itemManifest struct {
	Kind string `json:"kind"`
	Go   string `json:"go"`
	Rust string `json:"rust"`
	Line int    `json:"go_line"`
	Note string `json:"note,omitempty"`
}

type fileManifest struct {
	Go        string         `json:"go"`
	Rust      *string        `json:"rust"`
	GoLines   int            `json:"go_lines"`
	HD        bool           `json:"hd,omitempty"`
	Items     []itemManifest `json:"items"`
	Functions []*fnManifest  `json:"functions"`
	DependsOn []string       `json:"depends_on"`
}

type emitter struct {
	g        *gen
	a        *analyzer
	s        *scope
	pd       *pkgData
	f        *goFile
	b        strings.Builder
	openImpl string
	manifest *fileManifest
}

// 全域命名：Go 物件 → Rust 名稱
type naming struct {
	fn     map[*fnInfo]string
	consts map[*types.Const]string
	vars   map[*types.Var]string
	todo   map[*fnInfo]string
}

var nm naming

func (e *emitter) w(format string, args ...any) { fmt.Fprintf(&e.b, format, args...) }

var wsRe = regexp.MustCompile(`\s+`)

func collapse(s string) string { return strings.TrimSpace(wsRe.ReplaceAllString(s, " ")) }

func (e *emitter) src(from, to token.Pos) string {
	a := e.pd.fset.Position(from).Offset
	b := e.pd.fset.Position(to).Offset
	return string(e.f.Src[a:b])
}

func commentLines(cg *ast.CommentGroup) []string {
	if cg == nil {
		return nil
	}
	var lines []string
	for _, c := range cg.List {
		t := c.Text
		if strings.HasPrefix(t, "//") {
			t = strings.TrimPrefix(t, "//")
			if strings.HasPrefix(t, " ") {
				t = t[1:]
			}
			if strings.HasPrefix(t, "go:") || strings.HasPrefix(t, "nolint") {
				continue
			}
			lines = append(lines, strings.TrimRight(t, " \t"))
		} else {
			t = strings.TrimSuffix(strings.TrimPrefix(t, "/*"), "*/")
			for _, l := range strings.Split(t, "\n") {
				lines = append(lines, strings.TrimRight(strings.TrimPrefix(strings.TrimSpace(l), "* "), " "))
			}
		}
	}
	// 去掉首尾空行
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// doc 輸出多個註解群組（中間空一行）。
func (e *emitter) doc(indent, prefix string, groups ...*ast.CommentGroup) bool {
	any := false
	for _, cg := range groups {
		ls := commentLines(cg)
		if len(ls) == 0 {
			continue
		}
		if any {
			e.w("%s%s\n", indent, strings.TrimRight(prefix, " "))
		}
		any = true
		for _, l := range ls {
			l = strings.ReplaceAll(l, "\t", "    ")
			if strings.TrimSpace(l) == "" {
				e.w("%s%s\n", indent, strings.TrimRight(prefix, " "))
			} else {
				e.w("%s%s%s\n", indent, prefix, l)
			}
		}
	}
	return any
}

func (e *emitter) docText(indent string, lines ...string) {
	for _, l := range lines {
		if l == "" {
			e.w("%s///\n", indent)
		} else {
			e.w("%s/// %s\n", indent, l)
		}
	}
}

func (e *emitter) goLoc(pos token.Pos) string {
	return fmt.Sprintf("%s:%d", e.f.Name, e.pd.line(pos))
}

func (e *emitter) closeImpl() {
	if e.openImpl != "" {
		e.w("}\n")
		e.openImpl = ""
	}
}

// ---------------- file ----------------

func (e *emitter) emitFile(selFns, totalFns int) string {
	f := e.f
	e.w("//! Go 來源：`remake/cmd/fd2/%s`（%d 行）。\n", f.Name, f.Lines)
	var firstDecl token.Pos = f.AST.End()
	if len(f.AST.Decls) > 0 {
		firstDecl = f.AST.Decls[0].Pos()
	}
	var header []*ast.CommentGroup
	for _, cg := range f.AST.Comments {
		if cg.End() < firstDecl && !declDoc(f.AST, cg) && cg.Pos() > f.AST.Package {
			header = append(header, cg)
		}
	}
	if f.AST.Doc != nil {
		header = append([]*ast.CommentGroup{f.AST.Doc}, header...)
	}
	if len(header) > 0 {
		e.w("//!\n")
		e.doc("", "//! ", header...)
	}
	e.w("//!\n//! 骨架：Go 檔的 %d 個函式中產生 %d 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。\n", totalFns, selFns)
	if isHDFile(f.Name) {
		e.w("//! HD 檔：型別只保留第一章忠實模式執行到的欄位，其餘欄位與函式在 R6。\n")
	}
	e.w("\nuse super::*;\n")
	for _, d := range f.AST.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			e.genDecl(d)
		case *ast.FuncDecl:
			e.funcDecl(d)
		}
	}
	e.closeImpl()
	return e.b.String()
}

func declDoc(f *ast.File, cg *ast.CommentGroup) bool {
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			if d.Doc == cg {
				return true
			}
		case *ast.FuncDecl:
			if d.Doc == cg {
				return true
			}
		}
	}
	return false
}

// ---------------- gen decls ----------------

func (e *emitter) genDecl(d *ast.GenDecl) {
	single := len(d.Specs) == 1 && !d.Lparen.IsValid()
	for _, sp := range d.Specs {
		switch s := sp.(type) {
		case *ast.TypeSpec:
			doc := s.Doc
			if doc == nil && single {
				doc = d.Doc
			}
			tn := e.pd.info.Defs[s.Name].(*types.TypeName)
			if !e.s.types[tn] {
				continue
			}
			e.closeImpl()
			e.typeSpec(s, tn, doc)
		case *ast.ValueSpec:
			doc := s.Doc
			if doc == nil && single {
				doc = d.Doc
			}
			if d.Tok == token.CONST {
				e.constSpec(s, doc)
			} else {
				e.varSpec(s, doc)
			}
		}
	}
}

func (e *emitter) typeSpec(ts *ast.TypeSpec, tn *types.TypeName, doc *ast.CommentGroup) {
	g := e.g
	rn := g.typeNames[tn]
	e.w("\n")
	if e.doc("", "/// ", doc, ts.Comment) {
		e.w("///\n")
	}
	e.docText("", "Go：`"+e.goLoc(ts.Pos())+"`")
	if rn != camel(tn.Name()) {
		e.docText("", "Rust 名稱改為 `"+rn+"`：與其他名稱相撞。")
	}
	item := itemManifest{Kind: "type", Go: tn.Name(), Rust: rn, Line: e.pd.line(ts.Pos())}
	defer func() { e.manifest.Items = append(e.manifest.Items, item) }()
	vis := "pub(crate)"
	if tn == e.s.game {
		vis = "pub"
	}
	if ts.Assign.IsValid() {
		e.w("%s type %s = %s;\n", vis, rn, g.fieldType(types.Unalias(tn.Type()), tn.Name()))
		item.Kind = "alias"
		return
	}
	named := tn.Type().(*types.Named)
	switch u := named.Underlying().(type) {
	case *types.Struct:
		g.curStruct = tn
		n, kept := e.structBody(vis, rn, tn, u, ts.Type.(*ast.StructType))
		g.curStruct = nil
		if n != kept {
			item.Note = fmt.Sprintf("HD：保留 %d／%d 個欄位", kept, n)
		}
		item.Kind = "struct"
	case *types.Basic:
		inner := basicRust(u)
		if inner == "String" {
			inner = "String"
		}
		tr := g.traitMemo[tn]
		e.w("#[derive(%s)]\n", strings.Join(deriveList(tr, true), ", "))
		e.w("%s struct %s(pub(crate) %s);\n", vis, rn, inner)
		item.Kind = "newtype"
	case *types.Interface:
		e.docText("", "Go 介面（`IsPlaying`、`Close`）只由 `*audio.Player` 實作；Rust 直接用 [`AudioVoice`]。")
		e.w("%s type %s = AudioVoice;\n", vis, rn)
		item.Kind = "alias"
	default:
		e.w("%s type %s = %s;\n", vis, rn, g.fieldType(u, tn.Name()))
		item.Kind = "alias"
	}
}

func (e *emitter) structBody(vis, rn string, tn *types.TypeName, u *types.Struct, st *ast.StructType) (int, int) {
	g := e.g
	tr := g.traitMemo[tn]
	manual := g.manualDefault[tn]
	derives := deriveList(tr, g.keyTypes[tn])
	if manual {
		var d []string
		for _, x := range derives {
			if x != "Default" {
				d = append(d, x)
			}
		}
		derives = d
	}
	hd := isHDFile(e.pd.fileOf(tn.Pos()).Name)
	if hd {
		e.docText("", "", "HD 型別：只保留第一章忠實模式下執行到的函式用到的欄位（120「階段 1 骨架的對應規則」），其餘欄位在 R6 補上。")
	}
	if tr&tDebug == 0 || tr&tClone == 0 {
		e.docText("", "", "不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。")
	}
	if len(derives) > 0 {
		e.w("#[derive(%s)]\n", strings.Join(derives, ", "))
	}
	e.w("%s struct %s {\n", vis, rn)
	total, kept := 0, 0
	seen := map[string]string{}
	idx := 0
	var keptFields []struct {
		name string
		t    types.Type
	}
	for _, fld := range st.Fields.List {
		names := fld.Names
		embedded := len(names) == 0
		if embedded {
			fv := u.Field(idx)
			names = []*ast.Ident{{Name: fv.Name(), NamePos: fld.Pos()}}
		}
		for ni, nmID := range names {
			fv := u.Field(idx)
			idx++
			total++
			if hd && !e.s.hdKept[fv] {
				continue
			}
			kept++
			rf := escapeIdent(snake(nmID.Name))
			if prev, dup := seen[rf]; dup {
				g.reviewf("FIELD COLLISION %s.%s vs %s", tn.Name(), nmID.Name, prev)
				rf += "_2"
				e.docText("    ", fmt.Sprintf("Rust 名稱改為 `%s`：Go 欄位 `%s` 與 `%s` 轉成 snake_case 後相同。", rf, nmID.Name, prev))
			}
			seen[rf] = nmID.Name
			wrote := false
			if ni == 0 {
				wrote = e.doc("    ", "/// ", fld.Doc, fld.Comment)
			} else if fld.Doc != nil || fld.Comment != nil {
				e.docText("    ", fmt.Sprintf("Go 與 `%s` 同列宣告，註解見該欄位。", names[0].Name))
				wrote = true
			}
			if embedded {
				if wrote {
					e.docText("    ", "")
				}
				e.docText("    ", fmt.Sprintf("Go 內嵌欄位 `%s`；Go 的欄位提升在 Rust 要寫出 `.%s`。", fv.Name(), rf))
				wrote = true
			}
			if fld.Tag != nil {
				tag, _ := strconv.Unquote(fld.Tag.Value)
				if wrote {
					e.docText("    ", "")
				}
				e.docText("    ", "Go 標籤：`"+tag+"`")
			}
			key := tn.Name() + "." + fv.Name()
			rt := g.fieldType(fv.Type(), key)
			e.w("    pub(crate) %s: %s,\n", rf, rt)
			keptFields = append(keptFields, struct {
				name string
				t    types.Type
			}{rf, fv.Type()})
		}
	}
	if tn == e.s.game {
		e.docText("    ", "Rust 新增（沒有 Go 對應）：本邏輯幀的輸入。取代 Go 直接查詢 `inpututil.IsKeyJustPressed`、",
			"`ebiten.IsKeyPressed`；驅動端（無畫面重播或 `fd2-app`）在每次 `update` 前設定。")
		e.w("    pub(crate) tick_input: TickInput,\n")
		e.docText("    ", "Rust 新增（沒有 Go 對應）：虛擬時鐘目前的時間。取代 Go `gameNow()` 的牆鐘乘遊戲速度；",
			"驅動端每個邏輯幀前進固定步長。")
		e.w("    pub(crate) virtual_now: GameInstant,\n")
		for _, n := range []string{"tick_input", "virtual_now"} {
			if prev, dup := seen[n]; dup {
				panic("Game 的 Rust 新增欄位與 Go 欄位相撞：" + prev)
			}
		}
		keptFields = append(keptFields, struct {
			name string
			t    types.Type
		}{"tick_input", nil}, struct {
			name string
			t    types.Type
		}{"virtual_now", nil})
	}
	e.w("}\n")
	if manual {
		e.w("\nimpl Default for %s {\n    fn default() -> Self {\n        Self {\n", rn)
		for _, kf := range keptFields {
			e.w("            %s: %s,\n", kf.name, defaultExpr(kf.t))
		}
		e.w("        }\n    }\n}\n")
	}
	return total, kept
}

func defaultExpr(t types.Type) string {
	if t == nil {
		return "Default::default()"
	}
	if a, ok := t.(*types.Array); ok {
		inner := defaultExpr(a.Elem())
		if a.Len() > 32 || inner != "Default::default()" {
			return "std::array::from_fn(|_| " + inner + ")"
		}
	}
	return "Default::default()"
}

// ---------------- const ----------------

func (e *emitter) constSpec(vs *ast.ValueSpec, doc *ast.CommentGroup) {
	for i, id := range vs.Names {
		c, ok := e.pd.info.Defs[id].(*types.Const)
		if !ok || !e.s.consts[c] {
			continue
		}
		e.closeImpl()
		name := nm.consts[c]
		typ, val := e.g.constRust(c, vs, i)
		e.w("\n")
		wrote := e.doc("", "/// ", doc, vs.Comment)
		if i < len(vs.Values) {
			srcExpr := collapse(e.src(vs.Values[i].Pos(), vs.Values[i].End()))
			if _, lit := vs.Values[i].(*ast.BasicLit); !lit {
				if wrote {
					e.w("///\n")
				}
				e.docText("", "Go 原式：`"+srcExpr+"`")
			}
		} else if len(vs.Values) == 0 {
			if wrote {
				e.w("///\n")
			}
			e.docText("", "Go：沿用上一個常數的式子（`iota` 序列）。")
		}
		e.w("pub(crate) const %s: %s = %s;\n", name, typ, val)
		e.manifest.Items = append(e.manifest.Items, itemManifest{Kind: "const", Go: c.Name(), Rust: name, Line: e.pd.line(id.Pos())})
	}
}

func rustStr(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 || r == 0x7f {
				fmt.Fprintf(&b, `\u{%x}`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

func floatLit(f float64) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") && !strings.Contains(s, "Inf") && !strings.Contains(s, "NaN") {
		s += ".0"
	}
	if strings.Contains(s, "e") && !strings.Contains(s, ".") {
		// 1e+06 → 1.0e6
		parts := strings.SplitN(s, "e", 2)
		exp := strings.TrimPrefix(parts[1], "+")
		s = parts[0] + ".0e" + exp
	}
	return s
}

// basicValue：常數值在 Rust 基本型別 rt 下的字面值。
func basicValue(v constant.Value, rt string, srcHex string) string {
	switch rt {
	case "f64", "f32":
		f, _ := constant.Float64Val(constant.ToFloat(v))
		return floatLit(f)
	case "bool":
		return strconv.FormatBool(constant.BoolVal(v))
	case "String", "&str", "&'static str":
		return rustStr(constant.StringVal(v))
	}
	iv := constant.ToInt(v)
	if srcHex != "" {
		if u, ok := constant.Uint64Val(iv); ok && constant.Sign(iv) >= 0 {
			return fmt.Sprintf("0x%x", u)
		}
	}
	if i, ok := constant.Int64Val(iv); ok {
		return strconv.FormatInt(i, 10)
	}
	if u, ok := constant.Uint64Val(iv); ok {
		return strconv.FormatUint(u, 10)
	}
	return iv.ExactString()
}

func (g *gen) constRust(c *types.Const, vs *ast.ValueSpec, i int) (string, string) {
	hex := ""
	if i < len(vs.Values) {
		if bl, ok := vs.Values[i].(*ast.BasicLit); ok && bl.Kind == token.INT && strings.HasPrefix(strings.ToLower(bl.Value), "0x") {
			hex = bl.Value
		}
	}
	return g.typedValue(c.Type(), c.Val(), hex)
}

// typedValue：常數值在 Go 型別 t 下的 Rust 型別與字面值（const 位置）。
func (g *gen) typedValue(t types.Type, v constant.Value, hex string) (string, string) {
	switch u := t.(type) {
	case *types.Basic:
		rt := basicRust(u)
		if u.Kind() == types.UntypedRune {
			rt = "i32"
		}
		if rt == "String" {
			return "&str", basicValue(v, "&str", "")
		}
		return rt, basicValue(v, rt, hex)
	case *types.Named:
		if u.Obj().Pkg() == g.pd.pkg {
			rn := g.typeNames[u.Obj()]
			b := u.Underlying().(*types.Basic)
			inner := basicRust(b)
			return rn, rn + "(" + basicValue(v, inner, hex) + ")"
		}
		if nmx, ok := stdNamed(u); ok {
			switch nmx {
			case "GoDuration":
				return "GoDuration", "GoDuration(" + basicValue(v, "i64", "") + ")"
			case "Key":
				return "Key", "Key::" + keyVariantForValue(v)
			}
		}
		pp := u.Obj().Pkg().Path()
		last := pp[strings.LastIndex(pp, "/")+1:]
		p := extPath(g, u)
		b, _ := u.Underlying().(*types.Basic)
		inner := "i64"
		if b != nil {
			inner = basicRust(b)
		}
		if rt := rustScan.lookupType(last, u.Obj().Name()); rt != nil && rt.Kind == "newtype" {
			return p, p + "(" + basicValue(v, inner, hex) + ")"
		}
		return p, basicValue(v, inner, hex)
	}
	panic("const type " + t.String())
}

// ---------------- var ----------------

func (e *emitter) varSpec(vs *ast.ValueSpec, doc *ast.CommentGroup) {
	for i, id := range vs.Names {
		v, ok := e.pd.info.Defs[id].(*types.Var)
		if !ok || !e.s.vars[v] {
			continue
		}
		e.closeImpl()
		name := nm.vars[v]
		e.w("\n")
		wrote := e.doc("", "/// ", doc, vs.Comment)
		if wrote {
			e.w("///\n")
		}
		e.docText("", "Go：`"+e.goLoc(id.Pos())+"`")
		var init ast.Expr
		if i < len(vs.Values) {
			init = vs.Values[i]
		}
		written := e.g.varWritten[v]
		rt := e.g.fieldType(v.Type(), v.Name())
		item := itemManifest{Kind: "var", Go: v.Name(), Rust: name, Line: e.pd.line(id.Pos())}
		// errors.New 的哨兵錯誤
		if init != nil && !written && isErrorType(v.Type()) {
			if msg, ok := errorsNewArg(e.pd, init); ok {
				e.docText("", "Go `errors.New` 的哨兵錯誤；`errors.Is` 以 [`GameError::is`] 判斷。")
				e.w("pub(crate) static %s: GameError = GameError::sentinel(%s);\n", name, rustStr(msg))
				item.Kind = "sentinel"
				e.manifest.Items = append(e.manifest.Items, item)
				continue
			}
		}
		val := "Default::default()"
		translated := true
		if embed := goEmbed(doc); embed != "" {
			// go:embed 的素材不在 remake-rs 底下（Rust 容器只掛 remake-rs），骨架不嵌入
			val = fmt.Sprintf("todo!(%s)", rustStr(e.f.Name+":"+v.Name()+" go:embed "+embed))
			e.docText("", "", "Go 以 `//go:embed "+embed+"` 嵌入 `remake/cmd/fd2/"+embed+"`；素材放哪裡由擁有本檔的組決定（`include_bytes!` 或執行期讀檔）。")
			item.Note = "go:embed " + embed
			e.g.reviewf("VAR GO:EMBED %s:%s %s", e.f.Name, v.Name(), embed)
		} else if init != nil {
			val, translated = e.g.expr(init, v.Type())
			if !translated {
				val = fmt.Sprintf("todo!(%s)", rustStr(e.f.Name+":"+v.Name()+" 初始式"))
				e.docText("", "", "初始式還沒翻譯，階段 2 由擁有本檔的組補上。")
				item.Note = "初始式 todo"
				e.g.reviewf("VAR INIT TODO %s:%s", e.f.Name, v.Name())
			}
		}
		if written {
			e.docText("", "", "Go 會在執行期改寫這個套件層變數，Rust 以 `RwLock` 包住。")
			e.w("pub(crate) static %s: LazyLock<RwLock<%s>> = LazyLock::new(|| RwLock::new(%s));\n", name, rt, val)
			item.Note = strings.TrimSpace(item.Note + " 可改寫")
		} else if val == "Default::default()" {
			e.w("pub(crate) static %s: LazyLock<%s> = LazyLock::new(Default::default);\n", name, rt)
		} else {
			e.w("pub(crate) static %s: LazyLock<%s> = LazyLock::new(|| %s);\n", name, rt, val)
		}
		e.manifest.Items = append(e.manifest.Items, item)
	}
}

func goEmbed(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	for _, c := range cg.List {
		if strings.HasPrefix(c.Text, "//go:embed ") {
			return strings.TrimSpace(strings.TrimPrefix(c.Text, "//go:embed "))
		}
	}
	return ""
}

func errorsNewArg(pd *pkgData, e ast.Expr) (string, bool) {
	c, ok := e.(*ast.CallExpr)
	if !ok || len(c.Args) != 1 {
		return "", false
	}
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	fn, ok := pd.info.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "errors" || fn.Name() != "New" {
		return "", false
	}
	tv := pd.info.Types[c.Args[0]]
	if tv.Value == nil {
		return "", false
	}
	return constant.StringVal(tv.Value), true
}

// expr：把套件層變數的初始式翻成 Rust 運算式（只處理字面值與複合字面值）。
func (g *gen) expr(x ast.Expr, want types.Type) (string, bool) {
	info := g.pd.info
	x = ast.Unparen(x)
	if tv, ok := info.Types[x]; ok && tv.Value != nil {
		t := want
		if t == nil {
			t = tv.Type
		}
		return g.ownedValue(t, tv.Value)
	}
	switch v := x.(type) {
	case *ast.Ident:
		if v.Name == "nil" {
			return "Default::default()", true
		}
	case *ast.UnaryExpr:
		if v.Op == token.AND {
			if cl, ok := v.X.(*ast.CompositeLit); ok {
				s, ok := g.expr(cl, nil)
				if !ok {
					return "", false
				}
				return "Some(" + s + ")", true
			}
		}
	case *ast.CompositeLit:
		t := info.TypeOf(v)
		return g.compositeLit(v, t)
	}
	return "", false
}

func (g *gen) ownedValue(t types.Type, v constant.Value) (string, bool) {
	switch u := t.(type) {
	case *types.Basic:
		rt := basicRust(u)
		if rt == "String" {
			return "String::from(" + rustStr(constant.StringVal(v)) + ")", true
		}
		return basicValue(v, rt, ""), true
	case *types.Named:
		if u.Obj().Pkg() == g.pd.pkg {
			if b, ok := u.Underlying().(*types.Basic); ok {
				inner := basicRust(b)
				if inner == "String" {
					return g.typeNames[u.Obj()] + "(String::from(" + rustStr(constant.StringVal(v)) + "))", true
				}
				return g.typeNames[u.Obj()] + "(" + basicValue(v, inner, "") + ")", true
			}
		}
		_, s := g.typedValue(u, v, "")
		return s, true
	}
	return "", false
}

func (g *gen) compositeLit(cl *ast.CompositeLit, t types.Type) (string, bool) {
	var parts []string
	switch u := t.Underlying().(type) {
	case *types.Slice:
		for _, el := range cl.Elts {
			if _, ok := el.(*ast.KeyValueExpr); ok {
				return "", false
			}
			s, ok := g.elemExpr(el, u.Elem())
			if !ok {
				return "", false
			}
			parts = append(parts, s)
		}
		if len(parts) == 0 {
			return "Vec::new()", true
		}
		return "vec![" + strings.Join(parts, ", ") + "]", true
	case *types.Array:
		if int64(len(cl.Elts)) != u.Len() {
			return "", false
		}
		for _, el := range cl.Elts {
			if _, ok := el.(*ast.KeyValueExpr); ok {
				return "", false
			}
			s, ok := g.elemExpr(el, u.Elem())
			if !ok {
				return "", false
			}
			parts = append(parts, s)
		}
		return "[" + strings.Join(parts, ", ") + "]", true
	case *types.Map:
		for _, el := range cl.Elts {
			kv, ok := el.(*ast.KeyValueExpr)
			if !ok {
				return "", false
			}
			k, ok1 := g.elemExpr(kv.Key, u.Key())
			val, ok2 := g.elemExpr(kv.Value, u.Elem())
			if !ok1 || !ok2 {
				return "", false
			}
			parts = append(parts, "("+k+", "+val+")")
		}
		if len(parts) == 0 {
			return "BTreeMap::new()", true
		}
		return "BTreeMap::from([" + strings.Join(parts, ", ") + "])", true
	case *types.Struct:
		n, ok := t.(*types.Named)
		if !ok || n.Obj().Pkg() != g.pd.pkg {
			return "", false
		}
		rn := g.typeNames[n.Obj()]
		given := 0
		for i, el := range cl.Elts {
			var fv *types.Var
			val := el
			if kv, ok := el.(*ast.KeyValueExpr); ok {
				id, _ := kv.Key.(*ast.Ident)
				if id == nil {
					return "", false
				}
				fv, _ = g.pd.info.Uses[id].(*types.Var)
				val = kv.Value
			} else if i < u.NumFields() {
				fv = u.Field(i)
			}
			if fv == nil {
				return "", false
			}
			s, ok := g.expr(val, fv.Type())
			if !ok {
				return "", false
			}
			parts = append(parts, escapeIdent(snake(fv.Name()))+": "+s)
			given++
		}
		if given < len(g.structFields(n.Obj(), u)) {
			if g.traitMemo[n.Obj()]&tDefault == 0 && !g.manualDefault[n.Obj()] {
				return "", false
			}
			parts = append(parts, "..Default::default()")
		}
		return rn + " { " + strings.Join(parts, ", ") + " }", true
	}
	return "", false
}

func (g *gen) elemExpr(x ast.Expr, elem types.Type) (string, bool) {
	if p, ok := elem.(*types.Pointer); ok {
		if u, ok := ast.Unparen(x).(*ast.UnaryExpr); ok && u.Op == token.AND {
			x = u.X
		}
		elem = p.Elem()
	}
	return g.expr(x, elem)
}

// ---------------- functions ----------------

func (e *emitter) funcDecl(d *ast.FuncDecl) {
	obj := e.pd.info.Defs[d.Name].(*types.Func)
	fi := e.s.byObj[obj]
	if fi == nil || !fi.sel {
		return
	}
	g := e.g
	fa := e.a.fa[fi]
	recvRust := ""
	if fi.recvTN != nil {
		recvRust = g.typeNames[fi.recvTN]
	}
	if e.openImpl != recvRust {
		e.closeImpl()
		if recvRust != "" {
			e.w("\nimpl %s {\n", recvRust)
			e.openImpl = recvRust
		}
	}
	indent := ""
	if recvRust != "" {
		indent = "    "
	}
	sig, notes := e.signature(fi, fa)
	e.w("\n")
	if e.doc(indent, "/// ", d.Doc) {
		e.w("%s///\n", indent)
	}
	goSig := collapse(e.src(d.Pos(), d.Type.End()))
	e.docText(indent, "Go：`"+e.goLoc(d.Pos())+"`，`"+goSig+"`")
	for _, n := range notes {
		e.docText(indent, n)
	}
	e.w("%s%s {\n%s    todo!(%s)\n%s}\n", indent, sig, indent, rustStr(nm.todo[fi]), indent)
}

type rparam struct {
	name string
	typ  string
	ref  bool // 參考型別（借用）
	mut  bool
	note string
}

func (e *emitter) signature(fi *fnInfo, fa *fnAnalysis) (string, []string) {
	g := e.g
	var notes []string
	var ps []string
	selfKind := ""
	refParams := 0
	var refOnly *rparam
	if fi.sig.Recv() != nil {
		if fi.ptrRecv && fa.written[fa.params[0]] {
			selfKind = "&mut self"
		} else {
			selfKind = "&self"
		}
		if !fi.ptrRecv {
			notes = append(notes, "Go 值接收者：本體對接收者的改寫只影響副本。")
		}
		if fa.recvNil {
			notes = append(notes, "Go 檢查接收者是否為 nil；Rust 的 `self` 不會是 null，該分支不移植。")
		}
		ps = append(ps, selfKind)
	}
	fi.selfKind = selfKind
	unitIdx := fi.recvTN == e.s.game
	for i := 0; i < fi.sig.Params().Len(); i++ {
		if isStatePtr(fi.sig.Params().At(i).Type()) {
			unitIdx = true
		}
	}
	used := map[string]bool{}
	start := 0
	if fi.sig.Recv() != nil {
		start = 1
	}
	var rps []*rparam
	for i := 0; i < fi.sig.Params().Len(); i++ {
		p := fi.sig.Params().At(i)
		pn := escapeIdent(snake(p.Name()))
		if p.Name() == "" || p.Name() == "_" {
			pn = fmt.Sprintf("_arg%d", i)
		}
		for used[pn] {
			pn += "_2"
		}
		used[pn] = true
		variadic := fi.sig.Variadic() && i == fi.sig.Params().Len()-1
		rp := e.paramType(fi, fa, fa.params[start+i], variadic, unitIdx)
		rp.typ = strings.ReplaceAll(rp.typ, "<'static>", "<'_>")
		rp.name = pn
		if rp.ref {
			refParams++
			refOnly = rp
		}
		rps = append(rps, rp)
		if rp.note != "" {
			notes = append(notes, rp.note)
		}
		ps = append(ps, pn+": "+rp.typ)
	}
	// 指標回傳值：借用或新建
	borrow := map[int]string{}
	for i, from := range fa.borrowFrom {
		t := fi.sig.Results().At(i).Type()
		if isUnitPtr(t) || isImagePtr(t) {
			continue
		}
		if inner := g.elemType(t.(*types.Pointer).Elem(), ""); arcShared[inner] {
			continue
		}
		if selfKind != "" && from == fa.params[0] {
			if selfKind == "&mut self" {
				borrow[i] = "&mut "
			} else {
				borrow[i] = "&"
			}
			continue
		}
		if selfKind == "" && refParams == 1 && refOnly != nil {
			if refOnly.mut {
				borrow[i] = "&mut "
			} else {
				borrow[i] = "&"
			}
			continue
		}
		notes = append(notes, fmt.Sprintf("Go 第 %d 個回傳值是指向參數 `%s` 的指標；骨架回傳擁有的值，移植時再決定借用方式。", i+1, from.Name()))
	}
	ret := g.resultList(fi.sig.Results(), borrow)
	for i := range borrow {
		notes = append(notes, fmt.Sprintf("Go 第 %d 個回傳值是指向既有狀態的指標，Rust 回傳借用。", i+1))
	}
	sort.Strings(notes)
	name := nm.fn[fi]
	s := "pub(crate) fn " + name + "(" + strings.Join(ps, ", ") + ")"
	if ret != "()" {
		s += " -> " + ret
	}
	return s, notes
}

func (e *emitter) paramType(fi *fnInfo, fa *fnAnalysis, p *types.Var, variadic, unitIdx bool) *rparam {
	g := e.g
	t := p.Type()
	w := fa.written[p]
	nilc := fa.nilc[p]
	wrap := func(s string) string {
		if nilc {
			return "Option<" + s + ">"
		}
		return s
	}
	ref := func(s string) *rparam {
		m := "&"
		if w {
			m = "&mut "
		}
		return &rparam{typ: wrap(m + s), ref: true, mut: w}
	}
	if variadic {
		el := t.(*types.Slice).Elem()
		return &rparam{typ: "&[" + g.elemType(el, "") + "]", ref: true}
	}
	switch u := t.(type) {
	case *types.Pointer:
		if isUnitPtr(u) {
			if unitIdx {
				return &rparam{typ: wrap("UnitIdx")}
			}
			return ref("battle::Unit")
		}
		if isRandPtr(u) {
			return &rparam{typ: wrap("&mut dyn GoRand"), ref: true, mut: true}
		}
		inner := g.elemType(u.Elem(), "")
		if arcShared[inner] {
			if w {
				return &rparam{typ: wrap("&mut " + inner), ref: true, mut: true}
			}
			return &rparam{typ: wrap("&Arc<" + inner + ">"), ref: true}
		}
		return ref(inner)
	case *types.Slice:
		el := g.elemType(u.Elem(), "")
		if w {
			return &rparam{typ: "&mut [" + el + "]", ref: true, mut: true}
		}
		return &rparam{typ: "&[" + el + "]", ref: true}
	case *types.Map:
		return ref(g.fieldType(t, ""))
	case *types.Signature:
		if fa.stored[p] {
			// 存進工作結構的延續：與欄位同型（120 的 func 欄位規則），nil 合法
			return &rparam{typ: "Option<" + g.storedCallback(u, "param."+p.Name()) + ">", note: "Go 的 func 參數 `" + p.Name() + "` 會被存起來（工作結構的延續），Rust 收擁有的 `Box`。"}
		}
		return &rparam{typ: wrap("&mut " + g.dynFn(u)), ref: true, mut: true}
	case *types.Basic:
		if u.Kind() == types.String {
			return &rparam{typ: "&str", ref: true}
		}
		return &rparam{typ: basicRust(u)}
	case *types.Interface:
		if u.Empty() {
			return &rparam{typ: "Option<OwnerToken>"}
		}
		if u.NumMethods() == 1 && u.Method(0).Name() == "Words" {
			// 匿名介面 interface{ Words(int) ([]uint16, error) }：正式程式只傳 *fdtxt.Strings
			return &rparam{typ: "&fdtxt::Strings", ref: true, note: "Go 參數是匿名介面 `interface{ Words(int) ([]uint16, error) }`，正式程式只傳 `*fdtxt.Strings`。"}
		}
		panic("anonymous interface param " + u.String())
	case *types.Named:
		if isErrorType(u) {
			return &rparam{typ: "Option<&GameError>", ref: true}
		}
		if _, ok := u.Underlying().(*types.Interface); ok {
			if nmx, ok := stdNamed(u); ok && arcShared[nmx] {
				return &rparam{typ: "Option<&Arc<" + nmx + ">>", ref: true}
			}
			return &rparam{typ: "Option<&" + g.namedRust(u) + ">", ref: true}
		}
	case *types.Array:
	}
	ft := g.fieldType(t, "")
	if g.traits(t)&tCopy != 0 {
		return &rparam{typ: ft}
	}
	return &rparam{typ: "&" + ft, ref: true}
}
