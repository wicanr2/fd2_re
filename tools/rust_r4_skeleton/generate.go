package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

//go:embed templates/*
var templates embed.FS

func tmpl(name string) string {
	data, err := templates.ReadFile("templates/" + name)
	must(err)
	return string(data)
}

func writeOut(out, rel, content string) {
	p := filepath.Join(out, rel)
	must(os.MkdirAll(filepath.Dir(p), 0o755))
	must(os.WriteFile(p, []byte(content), 0o644))
}

// ---------- 命名 ----------

func computeNames(g *gen, s *scope) {
	nm = naming{fn: map[*fnInfo]string{}, consts: map[*types.Const]string{}, vars: map[*types.Var]string{}, todo: map[*fnInfo]string{}}
	// 函式：同一個命名空間（自由函式一組、每個接收者型別一組），匯出名稱優先
	spaces := map[string][]*fnInfo{}
	for _, fi := range s.fns {
		k := ""
		if fi.recvTN != nil {
			k = fi.recvTN.Name()
		}
		spaces[k] = append(spaces[k], fi)
	}
	keys := make([]string, 0, len(spaces))
	for k := range spaces {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		list := spaces[k]
		sort.SliceStable(list, func(i, j int) bool {
			ei, ej := isExported(list[i].decl.Name.Name), isExported(list[j].decl.Name.Name)
			if ei != ej {
				return ei
			}
			return less(list[i], list[j])
		})
		used := map[string]*fnInfo{}
		for _, fi := range list {
			name := fi.decl.Name.Name
			rn := snake(name)
			if name == "init" && fi.recvTN == nil {
				rn = "init_" + strings.TrimSuffix(fi.file.Name, ".go")
			}
			rn = escapeIdent(rn)
			if prev, dup := used[rn]; dup {
				g.reviewf("FN NAME COLLISION %s vs %s", prev.key, fi.key)
				if !isExported(name) && isExported(prev.decl.Name.Name) {
					rn += "_inner"
				}
				for used[rn] != nil {
					rn += "_2"
				}
			}
			used[rn] = fi
			nm.fn[fi] = rn
		}
	}
	// todo! 訊息：<檔>.go:<函式名>；同檔有同名方法時加接收者
	perFile := map[string]int{}
	for _, fi := range s.fns {
		perFile[fi.file.Name+":"+fi.decl.Name.Name]++
	}
	for _, fi := range s.fns {
		n := fi.decl.Name.Name
		if perFile[fi.file.Name+":"+n] > 1 && fi.recvTN != nil {
			n = fi.recvTN.Name() + "." + n
		}
		nm.todo[fi] = fi.file.Name + ":" + n
	}
	// 常數與套件層變數：同一個命名空間（SCREAMING_SNAKE）
	type val struct {
		name string
		obj  types.Object
		pos  token.Pos
	}
	var vals []val
	scope := g.pd.pkg.Scope()
	for _, n := range scope.Names() {
		switch o := scope.Lookup(n).(type) {
		case *types.Const, *types.Var:
			vals = append(vals, val{n, o, o.Pos()})
		}
	}
	sort.SliceStable(vals, func(i, j int) bool {
		ei, ej := isExported(vals[i].name), isExported(vals[j].name)
		if ei != ej {
			return ei
		}
		return vals[i].name < vals[j].name
	})
	used := map[string]string{}
	for _, v := range vals {
		rn := screaming(v.name)
		if prev, dup := used[rn]; dup {
			g.reviewf("VALUE NAME COLLISION %s vs %s", prev, v.name)
			for used[rn] != "" {
				rn += "_2"
			}
		}
		used[rn] = v.name
		switch o := v.obj.(type) {
		case *types.Const:
			nm.consts[o] = rn
		case *types.Var:
			nm.vars[o] = rn
		}
	}
}

// ---------- 套件層變數是否被改寫 ----------

func computeVarWritten(g *gen) {
	pd := g.pd
	a := &analyzer{pd: pd}
	isPkgVar := func(o types.Object) *types.Var {
		v, ok := o.(*types.Var)
		if ok && v.Pkg() == pd.pkg && v.Parent() == pd.pkg.Scope() {
			return v
		}
		return nil
	}
	root := func(e ast.Expr) *types.Var {
		o, _ := a.rootOf(e)
		if o == nil {
			return nil
		}
		return isPkgVar(o)
	}
	for _, f := range pd.files {
		for _, d := range f.AST.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.AssignStmt:
					for _, l := range x.Lhs {
						if v := root(l); v != nil {
							g.varWritten[v] = true
						}
					}
				case *ast.IncDecStmt:
					if v := root(x.X); v != nil {
						g.varWritten[v] = true
					}
				case *ast.UnaryExpr:
					if x.Op == token.AND {
						if v := root(x.X); v != nil {
							g.varWritten[v] = true
						}
					}
				case *ast.CallExpr:
					if id, ok := ast.Unparen(x.Fun).(*ast.Ident); ok {
						if b, ok := pd.info.Uses[id].(*types.Builtin); ok && len(x.Args) > 0 {
							switch b.Name() {
							case "append", "copy", "delete", "clear":
								if v := root(x.Args[0]); v != nil {
									g.varWritten[v] = true
								}
							}
						}
					}
					if sel, ok := ast.Unparen(x.Fun).(*ast.SelectorExpr); ok {
						if s := pd.info.Selections[sel]; s != nil && s.Kind() == types.MethodVal {
							if _, ptr := s.Obj().Type().(*types.Signature).Recv().Type().(*types.Pointer); ptr {
								if v := root(sel.X); v != nil {
									g.varWritten[v] = true
								}
							}
						}
					}
				}
				return true
			})
		}
	}
}

// ---------- map 鍵型別 ----------

func computeKeyTypes(g *gen) {
	for _, f := range g.pd.files {
		ast.Inspect(f.AST, func(n ast.Node) bool {
			if mt, ok := n.(*ast.MapType); ok {
				if tv, ok := g.pd.info.Types[mt]; ok {
					g.markKeyType(tv.Type.(*types.Map).Key())
				}
			}
			return true
		})
	}
	// 結構欄位與選定函式簽名裡的 map 型別（含沒有寫出 map 字樣的具名型別）
	for tn := range g.s.types {
		if st, ok := tn.Type().Underlying().(*types.Struct); ok {
			for i := 0; i < st.NumFields(); i++ {
				markMapKeys(g, st.Field(i).Type())
			}
		}
	}
}

func markMapKeys(g *gen, t types.Type) {
	switch u := t.(type) {
	case *types.Map:
		g.markKeyType(u.Key())
		markMapKeys(g, u.Elem())
	case *types.Slice:
		markMapKeys(g, u.Elem())
	case *types.Array:
		markMapKeys(g, u.Elem())
	case *types.Pointer:
		markMapKeys(g, u.Elem())
	}
}

// ---------- Ebiten 鍵 ----------

var keyVariants []string
var keyByValue = map[string]string{}

func computeKeys(pd *pkgData) {
	seen := map[string]bool{}
	for id, o := range pd.info.Uses {
		c, ok := o.(*types.Const)
		if !ok || c.Pkg() == nil || c.Pkg().Path() != "github.com/hajimehoshi/ebiten/v2" || !strings.HasPrefix(c.Name(), "Key") {
			continue
		}
		if !namedIs(c.Type(), "github.com/hajimehoshi/ebiten/v2", "Key") {
			continue
		}
		_ = id
		v := strings.TrimPrefix(c.Name(), "Key")
		if !seen[v] {
			seen[v] = true
			keyVariants = append(keyVariants, v)
		}
		if prev, ok := keyByValue[c.Val().ExactString()]; !ok || v < prev {
			keyByValue[c.Val().ExactString()] = v
		}
	}
	sort.Strings(keyVariants)
}

func keyVariantForValue(v constant.Value) string {
	if k, ok := keyByValue[v.ExactString()]; ok {
		return k
	}
	panic("ebiten.Key 常數值沒有對應的鍵：" + v.ExactString())
}

// ---------- 產生 ----------

type porting struct {
	Schema         int                 `json:"schema"`
	Crate          string              `json:"crate"`
	Generator      string              `json:"generator"`
	GoPackage      string              `json:"go_package"`
	GoSourceCommit string              `json:"go_source_commit_stage0"`
	Coverage       map[string]string   `json:"coverage_input"`
	RustModule     string              `json:"rust_module"`
	Totals         map[string]int      `json:"totals"`
	ExcludedFiles  []string            `json:"excluded_go_files"`
	TypeRules      []map[string]string `json:"type_rules"`
	Callbacks      map[string]string   `json:"callback_fields"`
	MissingExt     []string            `json:"missing_external_types"`
	LifetimeExt    []string            `json:"borrowing_external_types"`
	Review         []string            `json:"review"`
	Stage2         map[string]any      `json:"stage2"`
	Files          []*fileManifest     `json:"files"`
}

var typeRules = []map[string]string{
	{"go": "*ebiten.Image（欄位、容器元素、回傳值）", "rust": "Option<Arc<Picture>>"},
	{"go": "*ebiten.Image（參數）", "rust": "&mut Picture（會畫上去）／&Arc<Picture>（只讀）"},
	{"go": "*image.NRGBA、image.Image", "rust": "Option<Arc<Picture>>"},
	{"go": "*image.Paletted", "rust": "Option<Arc<IndexedFrame>>"},
	{"go": "*image.RGBA", "rust": "Option<Rgba>（fd2_campaign::campaign）"},
	{"go": "image.Point、image.Rectangle", "rust": "Point、Rectangle（fd2_campaign::campaign）"},
	{"go": "color.RGBA、color.NRGBA、color.Palette", "rust": "ColorRgba、ColorNrgba、ColorPalette"},
	{"go": "ebiten.Key", "rust": "Key"},
	{"go": "time.Time", "rust": "GameInstant"},
	{"go": "time.Duration", "rust": "GoDuration"},
	{"go": "*audio.Player、sfxVoice", "rust": "Option<AudioVoice>"},
	{"go": "*audio.Context", "rust": "Option<AudioContext>"},
	{"go": "font.Face、*sfnt.Font", "rust": "Option<Arc<FontFace>>、Option<Arc<SfntFont>>"},
	{"go": "*rand.Rand", "rust": "欄位 Option<battle::GoMathRand>；參數 &mut dyn GoRand"},
	{"go": "*os.File", "rust": "Option<std::fs::File>"},
	{"go": "any", "rust": "Option<OwnerToken>"},
	{"go": "error", "rust": "欄位 Option<GameError>；回傳 Result<T>"},
	{"go": "*battle.Unit", "rust": "欄位與容器 UnitIdx（Option<UnitIdx>）；Game 方法與帶 *battle.State 的函式的參數 UnitIdx；其他 &battle::Unit"},
	{"go": "func 欄位", "rust": "Option<Box<dyn FnOnce/FnMut(&mut Game, ..) + Send>>"},
	{"go": "func 參數", "rust": "&mut dyn FnMut(..)"},
	{"go": "*T 欄位", "rust": "Option<T>（遞迴時 Option<Box<T>>）"},
	{"go": "*T 參數", "rust": "&T／&mut T（Go 檢查 nil 時 Option<&T>）"},
	{"go": "[]T、map[K]V、map[K]struct{}", "rust": "Vec<T>、BTreeMap<K, V>、BTreeSet<K>"},
	{"go": "int、uint8、float64、string", "rust": "i64、u8、f64、String（參數 &str）"},
}

func generate(s *scope, out string) {
	g := newGen(s)
	a := newAnalyzer(s)
	computeKeys(s.pd)
	computeKeyTypes(g)
	computeVarWritten(g)
	g.computeTraits()
	computeNames(g, s)
	grp, groups := assignGroups(s)

	// 每個 Go 檔
	var manifests []*fileManifest
	var mods []string
	totalTypes, totalConsts, totalVars := 0, 0, 0
	for _, f := range s.pd.files {
		stem := strings.TrimSuffix(f.Name, ".go")
		m := &fileManifest{Go: "remake/cmd/fd2/" + f.Name, GoLines: f.Lines, HD: isHDFile(f.Name), Items: []itemManifest{}, Functions: []*fnManifest{}}
		total, sel := 0, 0
		var list []*fnInfo
		for _, fi := range s.fns {
			if fi.file == f {
				list = append(list, fi)
			}
		}
		sort.Slice(list, func(i, j int) bool { return list[i].decl.Pos() < list[j].decl.Pos() })
		for _, fi := range list {
			total++
			fm := &fnManifest{Name: fi.decl.Name.Name, Line: s.pd.line(fi.decl.Pos()), Rust: rustPath(fi)}
			if fi.recvTN != nil {
				fm.Recv = fi.recvTN.Name()
			}
			if fi.sel {
				sel++
				fm.Status = "todo"
				fm.Class = fi.class
				c := fi.cover
				fm.Cover = &c
				fm.Group = grp[fi]
				for c := range fi.calls {
					if c.sel {
						fm.Calls = append(fm.Calls, c.file.Name+":"+c.key)
					}
				}
				sort.Strings(fm.Calls)
			} else {
				fm.Status = "r8"
			}
			m.Functions = append(m.Functions, fm)
		}
		e := &emitter{g: g, a: a, s: s, pd: s.pd, f: f, manifest: m}
		src := e.emitFile(sel, total)
		hasItems := len(m.Items) > 0 || sel > 0
		if hasItems {
			rel := "src/game/" + stem + ".rs"
			r := "remake-rs/crates/fd2-game/" + rel
			m.Rust = &r
			writeOut(out, rel, src)
			mods = append(mods, stem)
		}
		for _, it := range m.Items {
			switch it.Kind {
			case "struct", "newtype", "alias":
				totalTypes++
			case "const":
				totalConsts++
			case "var", "sentinel":
				totalVars++
			}
		}
		// 選定函式在本檔呼叫到的其他檔
		deps := map[string]bool{}
		for _, fi := range list {
			if !fi.sel {
				continue
			}
			for c := range fi.calls {
				if c.sel && c.file != f {
					deps[c.file.Name] = true
				}
			}
		}
		m.DependsOn = sortedKeys(deps)
		if m.DependsOn == nil {
			m.DependsOn = []string{}
		}
		manifests = append(manifests, m)
	}
	for tn := range s.types {
		found := false
		for _, m := range manifests {
			for _, it := range m.Items {
				if it.Go == tn.Name() && (it.Kind == "struct" || it.Kind == "newtype" || it.Kind == "alias") {
					found = true
				}
			}
		}
		if !found {
			g.reviewf("TYPE NOT EMITTED %s", tn.Name())
		}
	}

	// game.rs
	var b strings.Builder
	head := tmpl("game_head.rs")
	head = strings.ReplaceAll(head, "{{SELECTED}}", strconv.Itoa(len(s.selected)))
	head = strings.ReplaceAll(head, "{{GAME_FIELDS}}", strconv.Itoa(s.game.Type().Underlying().(*types.Struct).NumFields()))
	b.WriteString(head)
	b.WriteString("\n// ---- 每個 Go 檔一個子模組（產生器輸出，依檔名排序）----\n")
	for _, m := range mods {
		if m == "game" {
			b.WriteString("#[allow(clippy::module_inception)]\n")
		}
		fmt.Fprintf(&b, "mod %s;\n", escapeIdent(m))
	}
	b.WriteString("\n")
	for _, m := range mods {
		fmt.Fprintf(&b, "pub(crate) use %s::*;\n", escapeIdent(m))
	}
	b.WriteString("\n// crate 外使用的入口\npub use main::Game;\n")
	writeOut(out, "src/game.rs", b.String())

	// support.rs
	var kb strings.Builder
	for i, k := range keyVariants {
		if i == 0 {
			kb.WriteString("    #[default]\n")
		}
		fmt.Fprintf(&kb, "    /// Go `ebiten.Key%s`。\n    %s,\n", k, k)
	}
	writeOut(out, "src/game/support.rs", strings.Replace(tmpl("support.rs"), "{{KEYS}}\n", kb.String(), 1))
	writeOut(out, "src/lib.rs", tmpl("lib.rs"))
	writeOut(out, "Cargo.toml", tmpl("Cargo.toml"))

	// PORTING.json
	tot := map[string]int{"go_files": len(s.pd.files), "go_files_with_module": len(mods), "types": totalTypes,
		"consts": totalConsts, "vars": totalVars, "functions": len(s.fns), "functions_todo": len(s.selected),
		"functions_r8": len(s.fns) - len(s.selected), "game_fields": s.game.Type().Underlying().(*types.Struct).NumFields()}
	goLines := 0
	for _, f := range s.pd.files {
		goLines += f.Lines
	}
	tot["go_lines"] = goLines
	hdTypes := 0
	for tn := range s.types {
		if isHDFile(s.pd.fileOf(tn.Pos()).Name) {
			hdTypes++
		}
	}
	tot["types_hd"] = hdTypes
	// 套件全體（含只有 r8 函式用到的）：骨架只產生選定函式用到的常數與變數
	pkgScope := s.pd.pkg.Scope()
	for _, n := range pkgScope.Names() {
		switch o := pkgScope.Lookup(n).(type) {
		case *types.Const:
			tot["package_consts"]++
		case *types.Var:
			tot["package_vars"]++
		case *types.TypeName:
			tot["package_types"]++
			_ = o
		}
	}
	covData, err := os.ReadFile(filepath.Join(*flagRepo, "docs/data/rust-r4-ch01-coverage-20261114.json"))
	must(err)
	sum := sha256.Sum256(covData)
	missing := sortedKeys(g.missing)
	if missing == nil {
		missing = []string{}
	}
	review := append([]string{}, g.review...)
	sort.Strings(review)
	if review == nil {
		review = []string{}
	}
	p := porting{Schema: 1, Crate: "fd2-game", Generator: "tools/rust_r4_skeleton（Go go/types 分析；tools/rust_r4_skeleton.sh 重跑）",
		GoPackage: "remake/cmd/fd2", GoSourceCommit: s.cov.GoSourceCommit,
		Coverage:   map[string]string{"file": "docs/data/rust-r4-ch01-coverage-20261114.json", "sha256": hex.EncodeToString(sum[:])},
		RustModule: "fd2_game::game", Totals: tot, ExcludedFiles: s.pd.ignored, TypeRules: typeRules,
		Callbacks: g.callbackFields, MissingExt: missing, LifetimeExt: sortedKeys(g.lifetimeExt), Review: review,
		Stage2: map[string]any{"note": "分組依檔名規則與呼叫關係產生，理由與相依見 PORTING.md。", "groups": groups},
		Files:  manifests}
	data, err := json.MarshalIndent(p, "", "  ")
	must(err)
	writeOut(out, "PORTING.json", string(data)+"\n")
	for _, r := range review {
		fmt.Fprintln(os.Stderr, "REVIEW", r)
	}
	for _, m := range missing {
		fmt.Fprintln(os.Stderr, "MISSING", m)
	}
}

func rustPath(fi *fnInfo) string {
	if fi.recvTN != nil {
		return "fd2_game::game::" + camel(fi.recvTN.Name()) + "::" + nm.fn[fi]
	}
	return "fd2_game::game::" + nm.fn[fi]
}
