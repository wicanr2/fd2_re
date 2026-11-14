package main

// 函式分析：參數（含接收者）是否被改寫、是否檢查 nil、指標回傳值是新建還是借用。
// 全部是保守判斷：拿不準就當成改寫（&mut）。

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

type fnAnalysis struct {
	params  []*types.Var // 接收者（若有）在前
	written map[*types.Var]bool
	why     map[*types.Var]string
	nilc    map[*types.Var]bool
	// 第 i 個回傳值（指標）借用自哪個參數；nil 表示新建
	borrowFrom map[int]*types.Var
	recvNil    bool
	// func 參數被存起來（指定給欄位、放進複合字面值或閉包、回傳、交給會存起來的參數）
	stored map[*types.Var]bool
}

type analyzer struct {
	s       *scope
	pd      *pkgData
	fa      map[*fnInfo]*fnAnalysis
	callFun map[ast.Expr]bool
}

func isRefType(t types.Type) bool {
	switch u := t.Underlying().(type) {
	case *types.Pointer, *types.Slice, *types.Map, *types.Signature, *types.Chan:
		return true
	case *types.Interface:
		return !u.Empty()
	}
	return false
}

func (a *analyzer) obj(e ast.Expr) types.Object {
	if id, ok := e.(*ast.Ident); ok {
		if o := a.pd.info.Uses[id]; o != nil {
			return o
		}
		return a.pd.info.Defs[id]
	}
	return nil
}

// rootOf：剝掉欄位、索引、切片、解參考與括號，回傳最左邊的識別字物件；bare 表示本身就是識別字。
func (a *analyzer) rootOf(e ast.Expr) (types.Object, bool) {
	bare := true
	for {
		switch x := e.(type) {
		case *ast.ParenExpr:
			e = x.X
			continue
		case *ast.SelectorExpr:
			if sel := a.pd.info.Selections[x]; sel == nil {
				// 套件限定名稱（pkg.X）
				return nil, false
			}
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.SliceExpr:
			e = x.X
		case *ast.StarExpr:
			e = x.X
		case *ast.Ident:
			return a.obj(x), bare
		default:
			return nil, false
		}
		bare = false
	}
}

func newAnalyzer(s *scope) *analyzer {
	a := &analyzer{s: s, pd: s.pd, fa: map[*fnInfo]*fnAnalysis{}, callFun: map[ast.Expr]bool{}}
	for _, fi := range s.fns {
		fa := &fnAnalysis{written: map[*types.Var]bool{}, why: map[*types.Var]string{}, nilc: map[*types.Var]bool{}, borrowFrom: map[int]*types.Var{}, stored: map[*types.Var]bool{}}
		if fi.sig.Recv() != nil {
			fa.params = append(fa.params, fi.sig.Recv())
		}
		for i := 0; i < fi.sig.Params().Len(); i++ {
			fa.params = append(fa.params, fi.sig.Params().At(i))
		}
		a.fa[fi] = fa
		if fi.decl.Body != nil {
			ast.Inspect(fi.decl.Body, func(n ast.Node) bool {
				if c, ok := n.(*ast.CallExpr); ok {
					a.callFun[c.Fun] = true
				}
				return true
			})
		}
	}
	for _, fi := range s.fns {
		a.nilChecks(fi)
	}
	for changed := true; changed; {
		changed = false
		for _, fi := range s.fns {
			if a.writes(fi) {
				changed = true
			}
		}
	}
	for i := 0; i < 4; i++ {
		for _, fi := range s.fns {
			a.returns(fi)
		}
	}
	for changed := true; changed; {
		changed = false
		for _, fi := range s.fns {
			if a.storedFuncs(fi) {
				changed = true
			}
		}
	}
	return a
}

// storedFuncs：func 型別的參數是否被存起來（Rust 要收擁有的 Box，不能收 &mut dyn FnMut）。
func (a *analyzer) storedFuncs(fi *fnInfo) bool {
	fa := a.fa[fi]
	if fi.decl.Body == nil {
		return false
	}
	isFn := map[types.Object]*types.Var{}
	for _, p := range fa.params {
		if _, ok := p.Type().Underlying().(*types.Signature); ok && !fa.stored[p] {
			isFn[p] = p
		}
	}
	if len(isFn) == 0 {
		return false
	}
	changed := false
	mark := func(e ast.Expr) {
		if p := isFn[a.obj(ast.Unparen(e))]; p != nil && !fa.stored[p] {
			fa.stored[p] = true
			changed = true
		}
	}
	ast.Inspect(fi.decl.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			for _, r := range x.Rhs {
				mark(r)
			}
		case *ast.ValueSpec:
			for _, r := range x.Values {
				mark(r)
			}
		case *ast.CompositeLit:
			for _, el := range x.Elts {
				if kv, ok := el.(*ast.KeyValueExpr); ok {
					el = kv.Value
				}
				mark(el)
			}
		case *ast.ReturnStmt:
			for _, r := range x.Results {
				mark(r)
			}
		case *ast.FuncLit:
			ast.Inspect(x.Body, func(m ast.Node) bool {
				if id, ok := m.(*ast.Ident); ok {
					mark(id)
				}
				return true
			})
		case *ast.CallExpr:
			var fn *types.Func
			switch f := ast.Unparen(x.Fun).(type) {
			case *ast.Ident:
				fn, _ = a.pd.info.Uses[f].(*types.Func)
			case *ast.SelectorExpr:
				if sel := a.pd.info.Selections[f]; sel != nil && sel.Kind() == types.MethodVal {
					fn, _ = sel.Obj().(*types.Func)
				}
			}
			for i, arg := range x.Args {
				if isFn[a.obj(ast.Unparen(arg))] == nil {
					continue
				}
				cfi := (*fnInfo)(nil)
				if fn != nil {
					cfi = a.s.byObj[fn.Origin()]
				}
				if cfi == nil {
					mark(arg)
					continue
				}
				cfa := a.fa[cfi]
				idx := i
				if cfi.sig.Recv() != nil {
					idx++
				}
				if idx < len(cfa.params) && cfa.stored[cfa.params[idx]] {
					mark(arg)
				}
			}
		}
		return true
	})
	return changed
}

func (a *analyzer) nilChecks(fi *fnInfo) {
	if fi.decl.Body == nil {
		return
	}
	fa := a.fa[fi]
	isParam := map[types.Object]*types.Var{}
	for _, p := range fa.params {
		isParam[p] = p
	}
	ast.Inspect(fi.decl.Body, func(n ast.Node) bool {
		b, ok := n.(*ast.BinaryExpr)
		if !ok || (b.Op != token.EQL && b.Op != token.NEQ) {
			return true
		}
		for _, pair := range [][2]ast.Expr{{b.X, b.Y}, {b.Y, b.X}} {
			if id, ok := pair[1].(*ast.Ident); ok && id.Name == "nil" {
				if p := isParam[a.obj(pair[0])]; p != nil {
					if fi.sig.Recv() == p {
						fa.recvNil = true
					} else {
						fa.nilc[p] = true
					}
				}
			}
		}
		return true
	})
}

// 只讀的標準函式庫套件與函式（引數不會被改寫）。
var readOnlyStdPkgs = map[string]bool{
	"fmt": true, "strings": true, "strconv": true, "math": true, "errors": true, "unicode": true,
	"unicode/utf8": true, "path/filepath": true, "path": true, "encoding/hex": true, "reflect": true,
	"crypto/sha256": true, "encoding/json": false, "bytes": true, "log": true, "time": true, "os": true,
}

func (a *analyzer) calleeWrites(fn *types.Func, argIdx int, recvCall bool) (bool, string) {
	if fi := a.s.byObj[fn.Origin()]; fi != nil {
		fa := a.fa[fi]
		idx := argIdx
		if fi.sig.Recv() != nil && !recvCall {
			// 方法運算式 T.m(recv, ...)：不常見，保守處理
			return true, "method expression"
		}
		if recvCall {
			if !fi.ptrRecv {
				return false, ""
			}
			return fa.written[fa.params[0]], "local method " + fi.key
		}
		if fi.sig.Recv() != nil {
			idx++
		}
		if fi.sig.Variadic() && idx >= len(fa.params)-1 {
			idx = len(fa.params) - 1
		}
		if idx < len(fa.params) {
			return fa.written[fa.params[idx]], "local call " + fi.key
		}
		return true, "arg index"
	}
	pkg := fn.Pkg()
	if pkg == nil {
		return true, "universe " + fn.Name()
	}
	path := pkg.Path()
	sig := fn.Type().(*types.Signature)
	if recvCall {
		if _, ptr := sig.Recv().Type().(*types.Pointer); !ptr {
			return false, ""
		}
		owner, _ := recvTypeName(sig)
		last := path[strings.LastIndex(path, "/")+1:]
		if owner != nil && rustModFor(last) != "" {
			if rf := rustScan.lookupFn(last, owner.Name(), fn.Name()); rf != nil {
				return rf.Self == "&mut self", "ext method " + rf.Mod + "." + rf.Owner + "." + rf.Name + " " + rf.Self
			}
		}
		if path == "bytes" || path == "strings" {
			// bytes.Buffer／strings.Builder 的方法只改寫自己的區域變數
			return true, "std method " + fn.FullName()
		}
		return true, "ext method unknown " + fn.FullName()
	}
	if readOnlyStdPkgs[path] {
		return false, ""
	}
	last := path[strings.LastIndex(path, "/")+1:]
	if rustModFor(last) != "" {
		if rf := rustScan.lookupFn(last, "", fn.Name()); rf != nil {
			if argIdx < len(rf.Params) {
				return strings.HasPrefix(rf.Params[argIdx].Type, "&mut") || strings.Contains(rf.Params[argIdx].Type, "&mut "), "ext fn " + rf.Mod + "." + rf.Name
			}
			return true, "ext fn arg index"
		}
	}
	return true, "ext fn unknown " + fn.FullName()
}

// writes：更新 fi 的改寫集合，回傳是否有變化。
func (a *analyzer) writes(fi *fnInfo) bool {
	fa := a.fa[fi]
	if fi.decl.Body == nil {
		return false
	}
	alias := map[types.Object]*types.Var{}
	for _, p := range fa.params {
		if p.Name() == "" || p.Name() == "_" {
			continue
		}
		alias[p] = p
	}
	rootParam := func(e ast.Expr) (*types.Var, bool) {
		o, bare := a.rootOf(e)
		if o == nil {
			return nil, false
		}
		return alias[o], bare
	}
	// 別名傳遞：x := g.list、for _, u := range g.units（元素是參考型別）
	for pass := 0; pass < 3; pass++ {
		ast.Inspect(fi.decl.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				if len(x.Lhs) == len(x.Rhs) {
					for i, l := range x.Lhs {
						if p, _ := rootParam(x.Rhs[i]); p != nil {
							if t := a.pd.info.TypeOf(x.Rhs[i]); t != nil && isRefType(t) {
								if o := a.obj(l); o != nil {
									alias[o] = p
								}
							}
						}
					}
				}
			case *ast.RangeStmt:
				if p, _ := rootParam(x.X); p != nil && x.Value != nil {
					if t := a.pd.info.TypeOf(x.Value); t != nil && isRefType(t) {
						if o := a.obj(x.Value); o != nil {
							alias[o] = p
						}
					}
				}
			case *ast.ValueSpec:
				if len(x.Names) == len(x.Values) {
					for i, nm := range x.Names {
						if p, _ := rootParam(x.Values[i]); p != nil {
							if t := a.pd.info.TypeOf(x.Values[i]); t != nil && isRefType(t) {
								alias[a.pd.info.Defs[nm]] = p
							}
						}
					}
				}
			}
			return true
		})
	}
	changed := false
	mark := func(p *types.Var, why string) {
		if p != nil && !fa.written[p] {
			fa.written[p] = true
			fa.why[p] = why
			changed = true
		}
	}
	lhsWrite := func(e ast.Expr, why string) {
		if p, bare := rootParam(e); p != nil && !bare {
			mark(p, why)
		}
	}
	ast.Inspect(fi.decl.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.AssignStmt:
			if x.Tok != token.DEFINE {
				for _, l := range x.Lhs {
					lhsWrite(l, "assign")
				}
			}
		case *ast.IncDecStmt:
			lhsWrite(x.X, "incdec")
		case *ast.UnaryExpr:
			if x.Op == token.AND {
				if t := a.pd.info.TypeOf(x); t != nil && isUnitPtr(t) {
					// &st.Units[i] 在 Rust 是 UnitIdx，不借用
					break
				}
				if p, _ := rootParam(x.X); p != nil {
					mark(p, "address-of")
				}
			}
		case *ast.CompositeLit:
			for _, el := range x.Elts {
				if kv, ok := el.(*ast.KeyValueExpr); ok {
					el = kv.Value
				}
				if p, _ := rootParam(el); p != nil {
					if t := a.pd.info.TypeOf(el); t != nil && isRefType(t) {
						mark(p, "stored in composite literal")
					}
				}
			}
		case *ast.SelectorExpr:
			if sel := a.pd.info.Selections[x]; sel != nil && sel.Kind() == types.MethodVal && !a.callFun[x] {
				if p, _ := rootParam(x.X); p != nil {
					mark(p, "method value")
				}
			}
		case *ast.CallExpr:
			a.callWrites(x, rootParam, mark)
		}
		return true
	})
	return changed
}

func (a *analyzer) callWrites(c *ast.CallExpr, rootParam func(ast.Expr) (*types.Var, bool), mark func(*types.Var, string)) {
	info := a.pd.info
	// 型別轉換
	if tv, ok := info.Types[c.Fun]; ok && tv.IsType() {
		return
	}
	var fn *types.Func
	recvCall := false
	var recvExpr ast.Expr
	switch f := ast.Unparen(c.Fun).(type) {
	case *ast.Ident:
		if b, ok := info.Uses[f].(*types.Builtin); ok {
			switch b.Name() {
			case "append", "copy", "delete", "clear":
				if len(c.Args) > 0 {
					if p, _ := rootParam(c.Args[0]); p != nil {
						mark(p, "builtin "+b.Name())
					}
				}
			}
			return
		}
		fn, _ = info.Uses[f].(*types.Func)
	case *ast.SelectorExpr:
		if sel := info.Selections[f]; sel != nil {
			if sel.Kind() == types.MethodVal {
				fn, _ = sel.Obj().(*types.Func)
				recvCall = true
				recvExpr = f.X
			} else {
				// 欄位裡的 func 值
				fn = nil
			}
		} else {
			fn, _ = info.Uses[f.Sel].(*types.Func)
		}
	}
	if recvCall && fn != nil {
		if p, _ := rootParam(recvExpr); p != nil {
			if w, why := a.calleeWrites(fn, 0, true); w {
				mark(p, why)
			}
		}
	}
	for i, arg := range c.Args {
		p, _ := rootParam(arg)
		if p == nil {
			continue
		}
		t := info.TypeOf(arg)
		if t == nil || !isRefType(t) {
			continue
		}
		if fn == nil {
			mark(p, "passed to func value")
			continue
		}
		if w, why := a.calleeWrites(fn, i, false); w {
			mark(p, why)
		}
	}
}

// returns：判定指標回傳值借用自哪個參數。
func (a *analyzer) returns(fi *fnInfo) {
	fa := a.fa[fi]
	res := fi.sig.Results()
	if fi.decl.Body == nil {
		return
	}
	isParam := map[types.Object]*types.Var{}
	for _, p := range fa.params {
		isParam[p] = p
	}
	// 區域變數的所有指定來源
	assigns := map[types.Object][]ast.Expr{}
	ast.Inspect(fi.decl.Body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		if as, ok := n.(*ast.AssignStmt); ok && len(as.Lhs) == len(as.Rhs) {
			for i, l := range as.Lhs {
				if o := a.obj(l); o != nil {
					assigns[o] = append(assigns[o], as.Rhs[i])
				}
			}
		}
		if vs, ok := n.(*ast.ValueSpec); ok && len(vs.Names) == len(vs.Values) {
			for i, nm := range vs.Names {
				o := a.pd.info.Defs[nm]
				assigns[o] = append(assigns[o], vs.Values[i])
			}
		}
		return true
	})
	var classify func(e ast.Expr, depth int) *types.Var
	classify = func(e ast.Expr, depth int) *types.Var {
		if depth > 4 {
			return nil
		}
		e = ast.Unparen(e)
		switch x := e.(type) {
		case *ast.Ident:
			o := a.obj(x)
			if p := isParam[o]; p != nil {
				return p
			}
			for _, r := range assigns[o] {
				if p := classify(r, depth+1); p != nil {
					return p
				}
			}
			return nil
		case *ast.UnaryExpr:
			if x.Op == token.AND {
				if _, ok := ast.Unparen(x.X).(*ast.CompositeLit); ok {
					return nil
				}
				if o, _ := a.rootOf(x.X); o != nil {
					if p := isParam[o]; p != nil {
						return p
					}
				}
			}
			return nil
		case *ast.SelectorExpr, *ast.IndexExpr:
			if o, _ := a.rootOf(x); o != nil {
				if p := isParam[o]; p != nil {
					return p
				}
				for _, r := range assigns[o] {
					if p := classify(r, depth+1); p != nil {
						return p
					}
				}
			}
			return nil
		case *ast.CallExpr:
			var fn *types.Func
			var recv ast.Expr
			switch f := ast.Unparen(x.Fun).(type) {
			case *ast.Ident:
				fn, _ = a.pd.info.Uses[f].(*types.Func)
			case *ast.SelectorExpr:
				if sel := a.pd.info.Selections[f]; sel != nil && sel.Kind() == types.MethodVal {
					fn, _ = sel.Obj().(*types.Func)
					recv = f.X
				}
			}
			if fn == nil {
				return nil
			}
			cfi := a.s.byObj[fn.Origin()]
			if cfi == nil || recv == nil {
				return nil
			}
			cfa := a.fa[cfi]
			if from, ok := cfa.borrowFrom[0]; ok && from == cfi.sig.Recv() {
				if o, _ := a.rootOf(recv); o != nil {
					return isParam[o]
				}
			}
			return nil
		}
		return nil
	}
	for i := 0; i < res.Len(); i++ {
		if _, ok := res.At(i).Type().(*types.Pointer); !ok {
			continue
		}
		var from *types.Var
		ast.Inspect(fi.decl.Body, func(n ast.Node) bool {
			if _, ok := n.(*ast.FuncLit); ok {
				return false
			}
			r, ok := n.(*ast.ReturnStmt)
			if !ok || len(r.Results) != res.Len() {
				return true
			}
			if p := classify(r.Results[i], 0); p != nil && from == nil {
				from = p
			}
			return true
		})
		if from != nil {
			fa.borrowFrom[i] = from
		}
	}
}
