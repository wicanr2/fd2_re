package main

// 範圍：階段 0 清單的 565 個函式、它們與 Game 可達的型別、函式本體用到的常數與套件層變數。

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"sort"
	"strings"
)

type fnInfo struct {
	decl    *ast.FuncDecl
	obj     *types.Func
	sig     *types.Signature
	file    *goFile
	key     string // Go 名稱：func 或 Recv.method
	recvTN  *types.TypeName
	ptrRecv bool
	sel     bool // 在階段 0 清單內
	class   string
	cover   float64
	calls   map[*fnInfo]bool
	// 產生時決定
	rustName string
	selfKind string // "", "&self", "&mut self"
	status   string
}

type scope struct {
	pd        *pkgData
	cov       *coverageDoc
	fns       []*fnInfo
	byObj     map[*types.Func]*fnInfo
	selected  []*fnInfo
	ambiguous []string
	missing   []string
	// 型別閉包
	types      map[*types.TypeName]bool
	hdKept     map[*types.Var]bool // HD 型別保留的欄位
	usedFields map[*types.Var]bool
	consts     map[*types.Const]bool
	vars       map[*types.Var]bool
	extTypes   map[string]int
	game       *types.TypeName
}

// coverageLines：階段 0 清單只記函式名稱；同一檔有兩個同名方法時，以階段 0 執行時
// go tool cover -func 的行號指定是哪一個（work/r4-stage0-20261114/cover-func.txt，
// 執行到的那個覆蓋率非 0，另一個是 0.0%）。
var coverageLines = map[string]int{
	"modern_hd_compound_map_source.go:digest":   64,  // modernHDCompoundMapScene.digest 50.0%（103 行 modernHDCompoundMapSource.digest 0.0%）
	"modern_hd_compound_map_source.go:seal":     95,  // 75.0%（115 行 0.0%）
	"modern_hd_compound_map_source.go:validate": 155, // 47.8%（122 行 0.0%）
	"title_menu_state.go:Step":                  32,  // 69.2%（74 行 0.0%）
}

func isHDFile(name string) bool {
	return strings.HasPrefix(name, "modern_") || strings.HasPrefix(name, "player_poster") || strings.HasPrefix(name, "player_toolbar")
}

func recvTypeName(sig *types.Signature) (*types.TypeName, bool) {
	r := sig.Recv()
	if r == nil {
		return nil, false
	}
	t := r.Type()
	ptr := false
	if p, ok := t.(*types.Pointer); ok {
		t = p.Elem()
		ptr = true
	}
	if n, ok := t.(*types.Named); ok {
		return n.Obj(), ptr
	}
	return nil, ptr
}

func newScope(pd *pkgData, cov *coverageDoc) *scope {
	s := &scope{pd: pd, cov: cov, byObj: map[*types.Func]*fnInfo{}, types: map[*types.TypeName]bool{},
		hdKept: map[*types.Var]bool{}, usedFields: map[*types.Var]bool{}, consts: map[*types.Const]bool{}, vars: map[*types.Var]bool{}, extTypes: map[string]int{}}
	byFileName := map[string][]*fnInfo{}
	for _, f := range pd.files {
		for _, d := range f.AST.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			obj := pd.info.Defs[fd.Name].(*types.Func)
			fi := &fnInfo{decl: fd, obj: obj, sig: obj.Type().(*types.Signature), file: f, calls: map[*fnInfo]bool{}}
			fi.recvTN, fi.ptrRecv = recvTypeName(fi.sig)
			fi.key = fd.Name.Name
			if fi.recvTN != nil {
				fi.key = fi.recvTN.Name() + "." + fd.Name.Name
			}
			s.fns = append(s.fns, fi)
			s.byObj[obj] = fi
			byFileName[f.Name+":"+fd.Name.Name] = append(byFileName[f.Name+":"+fd.Name.Name], fi)
		}
	}
	files := make([]string, 0, len(cov.Files))
	for f := range cov.Files {
		files = append(files, f)
	}
	sort.Strings(files)
	for _, f := range files {
		for _, c := range cov.Files[f] {
			cands := byFileName[f+":"+c.Func]
			if len(cands) == 0 {
				s.missing = append(s.missing, f+":"+c.Func)
				continue
			}
			if len(cands) > 1 {
				s.ambiguous = append(s.ambiguous, f+":"+c.Func)
				line, ok := coverageLines[f+":"+c.Func]
				if !ok {
					panic("階段 0 清單的 " + f + ":" + c.Func + " 對到多個 Go 宣告，coverageLines 沒有指定行號")
				}
				var keep []*fnInfo
				for _, fi := range cands {
					if pd.line(fi.decl.Pos()) == line {
						keep = append(keep, fi)
					}
				}
				if len(keep) != 1 {
					panic("coverageLines 的行號對不到 " + f + ":" + c.Func)
				}
				cands = keep
			}
			for _, fi := range cands {
				fi.sel = true
				fi.class = c.Class
				fi.cover = c.Cover
			}
		}
	}
	for _, fi := range s.fns {
		if fi.sel {
			s.selected = append(s.selected, fi)
		}
	}
	s.callGraph()
	g := pd.pkg.Scope().Lookup("Game")
	s.game = g.(*types.TypeName)
	s.closure()
	return s
}

func (s *scope) callGraph() {
	for _, fi := range s.fns {
		if fi.decl.Body == nil {
			continue
		}
		ast.Inspect(fi.decl.Body, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			if fn, ok := s.pd.info.Uses[id].(*types.Func); ok {
				if callee := s.byObj[fn.Origin()]; callee != nil {
					fi.calls[callee] = true
				}
			}
			return true
		})
	}
}

// closure 計算型別、常數、套件層變數的範圍。
func (s *scope) closure() {
	pd := s.pd
	// 1. 選定函式本體用到的欄位、常數、變數
	for _, fi := range s.selected {
		s.markBodyUses(fi.decl)
	}
	// 套件層變數的初始式也會用到常數與其他變數（遞移）
	for changed := true; changed; {
		changed = false
		for v := range s.vars {
			for _, f := range pd.files {
				for _, d := range f.AST.Decls {
					gd, ok := d.(*ast.GenDecl)
					if !ok {
						continue
					}
					for _, sp := range gd.Specs {
						vs, ok := sp.(*ast.ValueSpec)
						if !ok {
							continue
						}
						for _, nm := range vs.Names {
							if pd.info.Defs[nm] == v {
								before := len(s.vars) + len(s.consts)
								for _, val := range vs.Values {
									s.markExprUses(val)
								}
								if vs.Type != nil {
									s.markExprUses(vs.Type)
								}
								if len(s.vars)+len(s.consts) != before {
									changed = true
								}
							}
						}
					}
				}
			}
		}
	}
	// 2. 型別閉包
	var walk func(t types.Type)
	seen := map[types.Type]bool{}
	walk = func(t types.Type) {
		if t == nil || seen[t] {
			return
		}
		seen[t] = true
		switch u := t.(type) {
		case *types.Alias:
			walk(types.Unalias(u))
		case *types.Named:
			if u.Obj().Pkg() != pd.pkg {
				if u.Obj().Pkg() != nil {
					s.extTypes[u.Obj().Pkg().Path()+"."+u.Obj().Name()]++
				}
				return
			}
			s.types[u.Obj()] = true
			hd := isHDFile(pd.fileOf(u.Obj().Pos()).Name)
			if st, ok := u.Underlying().(*types.Struct); ok {
				for i := 0; i < st.NumFields(); i++ {
					fv := st.Field(i)
					if hd && !s.usedFields[fv] {
						continue
					}
					if hd {
						s.hdKept[fv] = true
					}
					walk(fv.Type())
				}
			} else {
				walk(u.Underlying())
			}
			// 方法集合裡被選定的方法簽名
		case *types.Pointer:
			walk(u.Elem())
		case *types.Slice:
			walk(u.Elem())
		case *types.Array:
			walk(u.Elem())
		case *types.Map:
			walk(u.Key())
			walk(u.Elem())
		case *types.Chan:
			walk(u.Elem())
		case *types.Signature:
			for i := 0; i < u.Params().Len(); i++ {
				walk(u.Params().At(i).Type())
			}
			for i := 0; i < u.Results().Len(); i++ {
				walk(u.Results().At(i).Type())
			}
		case *types.Struct:
			for i := 0; i < u.NumFields(); i++ {
				walk(u.Field(i).Type())
			}
		case *types.Interface:
			for i := 0; i < u.NumMethods(); i++ {
				walk(u.Method(i).Type())
			}
		}
	}
	walk(s.game.Type())
	for _, fi := range s.selected {
		walk(fi.sig)
		if fi.sig.Recv() != nil {
			walk(fi.sig.Recv().Type())
		}
	}
	for v := range s.vars {
		walk(v.Type())
	}
	for c := range s.consts {
		if n, ok := c.Type().(*types.Named); ok {
			walk(n)
		}
	}
}

func (s *scope) markBodyUses(fd *ast.FuncDecl) {
	if fd.Body != nil {
		s.markExprUses(fd.Body)
	}
}

func (s *scope) markExprUses(n ast.Node) {
	pd := s.pd
	ast.Inspect(n, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if sel := pd.info.Selections[x]; sel != nil {
				// 經內嵌欄位的提升：路徑上的每個欄位都算用到
				t := sel.Recv()
				for _, idx := range sel.Index() {
					st := structOf(t)
					if st == nil || idx >= st.NumFields() {
						break
					}
					fv := st.Field(idx)
					if sel.Kind() == types.FieldVal || fv.Embedded() {
						s.usedFields[fv] = true
					}
					t = fv.Type()
				}
			}
		case *ast.CompositeLit:
			if tv, ok := pd.info.Types[x]; ok {
				if st := structOf(tv.Type); st != nil {
					for i, el := range x.Elts {
						if kv, ok := el.(*ast.KeyValueExpr); ok {
							if id, ok := kv.Key.(*ast.Ident); ok {
								if v, ok := pd.info.Uses[id].(*types.Var); ok {
									s.usedFields[v] = true
								}
							}
						} else if i < st.NumFields() {
							s.usedFields[st.Field(i)] = true
						}
					}
				}
			}
		case *ast.Ident:
			switch o := pd.info.Uses[x].(type) {
			case *types.Const:
				if o.Pkg() == pd.pkg && o.Parent() == pd.pkg.Scope() {
					s.consts[o] = true
				}
			case *types.Var:
				if o.Pkg() == pd.pkg && o.Parent() == pd.pkg.Scope() {
					s.vars[o] = true
				}
			}
		}
		return true
	})
}

func structOf(t types.Type) *types.Struct {
	for {
		switch u := t.(type) {
		case *types.Pointer:
			t = u.Elem()
			continue
		}
		break
	}
	st, _ := t.Underlying().(*types.Struct)
	return st
}

func (s *scope) printStats(detail bool) {
	w := os.Stderr
	fmt.Fprintf(w, "STAT funcs=%d selected=%d ambiguous=%v missing=%v\n", len(s.fns), len(s.selected), s.ambiguous, s.missing)
	nHD := 0
	for tn := range s.types {
		if isHDFile(s.pd.fileOf(tn.Pos()).Name) {
			nHD++
		}
	}
	fmt.Fprintf(w, "STAT types=%d hd=%d consts=%d vars=%d\n", len(s.types), nHD, len(s.consts), len(s.vars))
	if !detail {
		return
	}
	for _, k := range sortedKeys(s.extTypes) {
		fmt.Fprintf(w, "EXT %s %d\n", k, s.extTypes[k])
	}
	for tn := range s.types {
		switch u := tn.Type().Underlying().(type) {
		case *types.Interface:
			fmt.Fprintf(w, "IFACE %s %s\n", tn.Name(), u)
		case *types.Signature:
			fmt.Fprintf(w, "FUNCTYPE %s %s\n", tn.Name(), u)
		case *types.Basic:
			fmt.Fprintf(w, "BASIC %s %s\n", tn.Name(), u)
		case *types.Struct:
			for i := 0; i < u.NumFields(); i++ {
				if sg, ok := u.Field(i).Type().Underlying().(*types.Signature); ok && (!isHDFile(s.pd.fileOf(tn.Pos()).Name) || s.hdKept[u.Field(i)]) {
					fmt.Fprintf(w, "FUNCFIELD %s.%s %s\n", tn.Name(), u.Field(i).Name(), sg)
				}
				if _, ok := u.Field(i).Type().Underlying().(*types.Interface); ok {
					fmt.Fprintf(w, "IFACEFIELD %s.%s %s\n", tn.Name(), u.Field(i).Name(), u.Field(i).Type())
				}
			}
		default:
			fmt.Fprintf(w, "OTHER %s %T\n", tn.Name(), u)
		}
		if n, ok := tn.Type().(*types.Named); ok && n.TypeParams().Len() > 0 {
			fmt.Fprintf(w, "GENERIC %s\n", tn.Name())
		}
	}
	for v := range s.vars {
		fmt.Fprintf(w, "VAR %s %s %s\n", s.pd.fileOf(v.Pos()).Name, v.Name(), v.Type())
	}
	for _, fi := range s.selected {
		if fi.sig.TypeParams().Len() > 0 {
			fmt.Fprintf(w, "GENFUNC %s\n", fi.key)
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
