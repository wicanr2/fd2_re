package main

// Go 型別 → Rust 型別（docs/goal/120「階段 1 骨架的對應規則」）與可 derive 的 trait。

import (
	"fmt"
	"go/types"
	"os"
	"sort"
	"strings"
)

type trait uint

const (
	tClone trait = 1 << iota
	tCopy
	tDebug
	tDefault
	tPartialEq
	tEq
	tOrd
	tHash
)

const allTraits = tClone | tCopy | tDebug | tDefault | tPartialEq | tEq | tOrd | tHash

// 值型別（非 Copy）常見的 trait 組合。
const valueTraits = tClone | tDebug | tDefault | tPartialEq | tEq

type gen struct {
	s         *scope
	pd        *pkgData
	typeNames map[*types.TypeName]string
	traitMemo map[*types.TypeName]trait
	keyTypes  map[*types.TypeName]bool
	review    []string
	missing   map[string]bool
	curStruct *types.TypeName
	// 欄位型別的 callback 種類（FnOnce／FnMut），PORTING.json 記錄
	callbackFields map[string]string
	// 只因超過 32 元素的陣列而不能 derive Default 的結構：手寫 impl Default
	manualDefault map[*types.TypeName]bool
	// 套件層變數是否在執行期被改寫
	varWritten map[*types.Var]bool
	// 帶生命週期參數的外部型別
	lifetimeExt map[string]bool
}

func (g *gen) reviewf(format string, args ...any) {
	g.review = append(g.review, fmt.Sprintf(format, args...))
}

// Rust 型別名稱（type namespace）中已被 support.rs 或匯入佔用的名稱。
var reservedTypeNames = map[string]bool{
	"Game": false, "Picture": true, "IndexedFrame": true, "TickInput": true, "Key": true, "GameInstant": true,
	"GoDuration": true, "AudioVoice": true, "AudioContext": true, "GameError": true, "Result": true,
	"OwnerToken": true, "FontFace": true, "SfntFont": true, "ColorRgba": true, "ColorNrgba": true,
	"ColorPalette": true, "Point": true, "Rectangle": true, "Rgba": true, "Option": true, "Vec": true,
	"String": true, "Box": true, "Arc": true, "BTreeMap": true, "BTreeSet": true, "RwLock": true, "LazyLock": true,
	"GameThen": true,
}

func newGen(s *scope) *gen {
	g := &gen{s: s, pd: s.pd, typeNames: map[*types.TypeName]string{}, traitMemo: map[*types.TypeName]trait{},
		keyTypes: map[*types.TypeName]bool{}, missing: map[string]bool{}, callbackFields: map[string]string{},
		manualDefault: map[*types.TypeName]bool{}, varWritten: map[*types.Var]bool{}, lifetimeExt: map[string]bool{}}
	used := map[string]string{}
	var names []*types.TypeName
	for tn := range s.types {
		names = append(names, tn)
	}
	sort.Slice(names, func(i, j int) bool { return names[i].Name() < names[j].Name() })
	// 先排 Go 匯出名稱（大寫開頭），撞名時未匯出的那個加後綴
	sort.SliceStable(names, func(i, j int) bool {
		return isExported(names[i].Name()) && !isExported(names[j].Name())
	})
	for _, tn := range names {
		rn := camel(tn.Name())
		if reservedTypeNames[rn] {
			g.reviewf("TYPE NAME RESERVED %s -> %sGo", tn.Name(), rn)
			rn += "Go"
		}
		if prev, dup := used[rn]; dup {
			g.reviewf("TYPE NAME COLLISION %s vs %s -> %sInner", prev, tn.Name(), rn)
			rn += "Inner"
		}
		used[rn] = tn.Name()
		g.typeNames[tn] = rn
	}
	return g
}

func isExported(s string) bool { return s != "" && s[0] >= 'A' && s[0] <= 'Z' }

// ---------- 判斷特定型別 ----------

func namedIs(t types.Type, pkgSuffix, name string) bool {
	if a, ok := t.(*types.Alias); ok {
		t = types.Unalias(a)
	}
	n, ok := t.(*types.Named)
	return ok && n.Obj().Pkg() != nil && n.Obj().Name() == name &&
		(n.Obj().Pkg().Path() == pkgSuffix || strings.HasSuffix(n.Obj().Pkg().Path(), "/"+pkgSuffix))
}

func ptrTo(t types.Type, pkg, name string) bool {
	p, ok := t.(*types.Pointer)
	return ok && namedIs(p.Elem(), pkg, name)
}

func isUnitPtr(t types.Type) bool  { return ptrTo(t, "battle", "Unit") }
func isStatePtr(t types.Type) bool { return ptrTo(t, "battle", "State") }
func isImagePtr(t types.Type) bool { return ptrTo(t, "github.com/hajimehoshi/ebiten/v2", "Image") }
func isRandPtr(t types.Type) bool  { return ptrTo(t, "math/rand", "Rand") }

func isErrorType(t types.Type) bool {
	return types.Identical(t, types.Universe.Lookup("error").Type())
}

func isEmptyInterface(t types.Type) bool {
	i, ok := t.Underlying().(*types.Interface)
	return ok && i.Empty() && !isErrorType(t)
}

func isEmptyStruct(t types.Type) bool {
	s, ok := t.Underlying().(*types.Struct)
	return ok && s.NumFields() == 0
}

func basicRust(b *types.Basic) string {
	switch b.Kind() {
	case types.Int, types.Int64, types.UntypedInt:
		return "i64"
	case types.Int32, types.UntypedRune:
		return "i32"
	case types.Int16:
		return "i16"
	case types.Int8:
		return "i8"
	case types.Uint, types.Uint64:
		return "u64"
	case types.Uint32:
		return "u32"
	case types.Uint16:
		return "u16"
	case types.Uint8:
		return "u8"
	case types.Uintptr:
		return "usize"
	case types.Float64, types.UntypedFloat:
		return "f64"
	case types.Float32:
		return "f32"
	case types.Bool, types.UntypedBool:
		return "bool"
	case types.String, types.UntypedString:
		return "String"
	}
	panic("basic " + b.String())
}

// stdNamed：Go 標準函式庫與 Ebiten 的具名型別（以值出現時）。ok=false 表示不是這一類。
func stdNamed(n *types.Named) (string, bool) {
	p := n.Obj().Pkg().Path()
	nm := n.Obj().Name()
	switch p + "." + nm {
	case "time.Time":
		return "GameInstant", true
	case "time.Duration":
		return "GoDuration", true
	case "github.com/hajimehoshi/ebiten/v2.Key":
		return "Key", true
	case "image.Point":
		return "Point", true
	case "image.Rectangle":
		return "Rectangle", true
	case "image.RGBA":
		return "Rgba", true
	case "image.NRGBA", "image.Image":
		return "Picture", true
	case "image.Paletted":
		return "IndexedFrame", true
	case "image/color.RGBA":
		return "ColorRgba", true
	case "image/color.NRGBA":
		return "ColorNrgba", true
	case "image/color.Palette":
		return "ColorPalette", true
	case "github.com/hajimehoshi/ebiten/v2.Image":
		return "Picture", true
	case "github.com/hajimehoshi/ebiten/v2/audio.Player":
		return "AudioVoice", true
	case "github.com/hajimehoshi/ebiten/v2/audio.Context":
		return "AudioContext", true
	case "math/rand.Rand":
		return "battle::GoMathRand", true
	case "os.File":
		return "std::fs::File", true
	case "golang.org/x/image/font.Face":
		return "FontFace", true
	case "golang.org/x/image/font/sfnt.Font":
		return "SfntFont", true
	}
	return "", false
}

// stdTraits：上面這些型別的 trait。
var stdTraits = map[string]trait{
	"GameInstant":        allTraits,
	"GoDuration":         allTraits,
	"Key":                allTraits,
	"Point":              allTraits,
	"Rectangle":          allTraits,
	"Rgba":               valueTraits,
	"Picture":            valueTraits,
	"IndexedFrame":       valueTraits,
	"ColorRgba":          allTraits,
	"ColorNrgba":         allTraits,
	"ColorPalette":       valueTraits,
	"AudioVoice":         valueTraits,
	"AudioContext":       valueTraits,
	"battle::GoMathRand": tClone | tDebug,
	"std::fs::File":      tDebug,
	"FontFace":           tDebug,
	"SfntFont":           tDebug,
	"OwnerToken":         allTraits,
	"GameError":          valueTraits,
}

// 以 Arc 共享的大型唯讀資源（Go 以指標共用、不改寫內容）。
var arcShared = map[string]bool{"Picture": true, "IndexedFrame": true, "FontFace": true, "SfntFont": true}

func extPath(g *gen, n *types.Named) string {
	if s, ok := stdNamed(n); ok {
		return s
	}
	p := n.Obj().Pkg().Path()
	last := p[strings.LastIndex(p, "/")+1:]
	mod := rustModFor(last)
	if mod == "" {
		panic("external type " + p + "." + n.Obj().Name())
	}
	if t := rustScan.lookupType(last, n.Obj().Name()); t != nil {
		if t.Lifetime {
			// 外部型別借用資源；存進狀態要擁有的版本（缺口，見 PORTING.md）。骨架暫用 'static，參數位置改成 '_
			g.lifetimeExt[mod+"::"+t.Name] = true
			return mod + "::" + t.Name + "<'static>"
		}
		return mod + "::" + t.Name
	}
	g.missing[last+"."+n.Obj().Name()] = true
	return mod + "::" + camel(n.Obj().Name())
}

func (g *gen) namedRust(n *types.Named) string {
	if n.Obj().Pkg() == nil {
		if n.Obj().Name() == "error" {
			return "GameError"
		}
		panic("universe named " + n.String())
	}
	if n.Obj().Pkg() == g.pd.pkg {
		if rn, ok := g.typeNames[n.Obj()]; ok {
			return rn
		}
		panic("type outside closure " + n.Obj().Name())
	}
	return extPath(g, n)
}

// fieldType：結構欄位、套件層變數、回傳值等「擁有」位置的型別。
func (g *gen) fieldType(t types.Type, fieldKey string) string {
	switch t := t.(type) {
	case *types.Basic:
		return basicRust(t)
	case *types.Alias:
		return g.fieldType(types.Unalias(t), fieldKey)
	case *types.Named:
		if isErrorType(t) {
			return "Option<GameError>"
		}
		if t.Obj().Pkg() != g.pd.pkg {
			if nm, ok := stdNamed(t); ok && arcShared[nm] {
				// image.Image、font.Face 等介面值：nil 合法
				return "Option<Arc<" + nm + ">>"
			}
		}
		return g.namedRust(t)
	case *types.Pointer:
		if isUnitPtr(t) {
			return "Option<UnitIdx>"
		}
		inner := g.elemType(t.Elem(), fieldKey)
		if arcShared[inner] {
			return "Option<Arc<" + inner + ">>"
		}
		if g.curStruct != nil && g.inlineReaches(t.Elem(), g.curStruct, map[*types.TypeName]bool{}) {
			return "Option<Box<" + inner + ">>"
		}
		return "Option<" + inner + ">"
	case *types.Slice:
		return "Vec<" + g.elemType(t.Elem(), fieldKey) + ">"
	case *types.Array:
		return fmt.Sprintf("[%s; %d]", g.elemType(t.Elem(), fieldKey), t.Len())
	case *types.Map:
		k := g.elemType(t.Key(), fieldKey)
		if isEmptyStruct(t.Elem()) {
			return "BTreeSet<" + k + ">"
		}
		return "BTreeMap<" + k + ", " + g.elemType(t.Elem(), fieldKey) + ">"
	case *types.Signature:
		return "Option<" + g.storedCallback(t, fieldKey) + ">"
	case *types.Chan:
		// Go 以 close(done) 通知背景工作結束（player_theme_loader.go）；Rust 用 mpsc 的接收端。
		el := "()"
		if !isEmptyStruct(t.Elem()) {
			el = g.elemType(t.Elem(), fieldKey)
		}
		return "Option<std::sync::mpsc::Receiver<" + el + ">>"
	case *types.Interface:
		if isErrorType(t) {
			return "Option<GameError>"
		}
		if t.Empty() {
			return "Option<OwnerToken>"
		}
	case *types.Struct:
		if t.NumFields() == 0 {
			return "()"
		}
	}
	panic(fmt.Sprintf("fieldType %T %v (%s)", t, t, fieldKey))
}

// elemType：容器元素。指標元素是擁有的值（R2、R3 的規則），*battle.Unit 是索引，
// *ebiten.Image 照 120 仍是 Option<Arc<Picture>>（Go 的畫格快取常留 nil）。
func (g *gen) elemType(t types.Type, fieldKey string) string {
	if p, ok := t.(*types.Pointer); ok {
		if isUnitPtr(p) {
			return "UnitIdx"
		}
		inner := g.elemType(p.Elem(), fieldKey)
		if arcShared[inner] {
			return "Option<Arc<" + inner + ">>"
		}
		return inner
	}
	if isErrorType(t) {
		return "GameError"
	}
	if n, ok := t.(*types.Named); ok && n.Obj().Pkg() != nil && n.Obj().Pkg() != g.pd.pkg {
		if nm, ok := stdNamed(n); ok {
			if arcShared[nm] && !types.IsInterface(n) {
				return nm
			}
			if arcShared[nm] {
				return "Option<Arc<" + nm + ">>"
			}
			return nm
		}
	}
	return g.fieldType(t, fieldKey)
}

// storedCallback：結構欄位裡的 Go func。120 的規則：閉包捕捉 g，所以 Rust 第一個參數是
// &mut Game；工作結束時只呼叫一次的延續（then／after／rollback 等）是 FnOnce，其餘 FnMut。
func (g *gen) storedCallback(sig *types.Signature, fieldKey string) string {
	kind := "FnMut"
	name := fieldKey[strings.LastIndex(fieldKey, ".")+1:]
	ln := strings.ToLower(name)
	if strings.HasPrefix(ln, "then") || strings.HasSuffix(ln, "then") || strings.HasSuffix(ln, "after") ||
		strings.HasPrefix(ln, "after") || ln == "rollback" {
		kind = "FnOnce"
	}
	if fieldKey != "" && !strings.HasPrefix(fieldKey, "param.") {
		g.callbackFields[fieldKey] = kind
	}
	ps := []string{"&mut Game"}
	for i := 0; i < sig.Params().Len(); i++ {
		ps = append(ps, g.callbackParam(sig.Params().At(i).Type()))
	}
	r := g.resultList(sig.Results(), nil)
	out := "Box<dyn " + kind + "(" + strings.Join(ps, ", ") + ")"
	if r != "()" {
		out += " -> " + r
	}
	return out + " + Send>"
}

// callbackParam：callback 的參數。指標收借用（*battle.Unit 收索引），其餘收值。
func (g *gen) callbackParam(t types.Type) string {
	switch u := t.(type) {
	case *types.Pointer:
		if isUnitPtr(u) {
			return "UnitIdx"
		}
		return "&" + g.elemType(u.Elem(), "")
	case *types.Signature:
		return "&mut " + g.dynFn(u)
	}
	return g.fieldType(t, "")
}

// dynFn：函式參數裡的 Go func → dyn FnMut(..)。
func (g *gen) dynFn(sig *types.Signature) string {
	var ps []string
	for i := 0; i < sig.Params().Len(); i++ {
		ps = append(ps, g.callbackParam(sig.Params().At(i).Type()))
	}
	r := g.resultList(sig.Results(), nil)
	out := "dyn FnMut(" + strings.Join(ps, ", ") + ")"
	if r != "()" {
		out += " -> " + r
	}
	return out
}

// resultList：回傳值；最後一個是 error 時包成 Result。borrow 指定第 i 個指標回傳值改為借用。
func (g *gen) resultList(res *types.Tuple, borrow map[int]string) string {
	var outs []string
	hasErr := false
	for i := 0; i < res.Len(); i++ {
		t := res.At(i).Type()
		if i == res.Len()-1 && isErrorType(t) {
			hasErr = true
			continue
		}
		if b, ok := borrow[i]; ok {
			outs = append(outs, "Option<"+b+g.elemType(t.(*types.Pointer).Elem(), "")+">")
			continue
		}
		outs = append(outs, g.fieldType(t, ""))
	}
	inner := tuple(outs)
	if hasErr {
		return "Result<" + inner + ">"
	}
	return inner
}

func tuple(xs []string) string {
	switch len(xs) {
	case 0:
		return "()"
	case 1:
		return xs[0]
	}
	return "(" + strings.Join(xs, ", ") + ")"
}

// ---------- trait ----------

func (g *gen) traits(t types.Type) trait {
	switch t := t.(type) {
	case *types.Basic:
		switch t.Kind() {
		case types.Float32, types.Float64, types.UntypedFloat:
			return tClone | tCopy | tDebug | tDefault | tPartialEq
		case types.String, types.UntypedString:
			return allTraits &^ tCopy
		}
		return allTraits
	case *types.Alias:
		return g.traits(types.Unalias(t))
	case *types.Named:
		if isErrorType(t) {
			return valueTraits
		}
		if t.Obj().Pkg() != g.pd.pkg {
			if nm, ok := stdNamed(t); ok {
				tr := stdTraits[nm]
				if arcShared[nm] {
					// Option<Arc<T>>
					tr = (tr | tClone | tDefault) &^ tCopy
				}
				return tr
			}
			pp := t.Obj().Pkg().Path()
			if rt := rustScan.lookupType(pp[strings.LastIndex(pp, "/")+1:], t.Obj().Name()); rt != nil {
				return rt.Traits
			}
			fmt.Fprintf(os.Stderr, "WARN unknown external traits %s\n", t)
			return tClone
		}
		if v, ok := g.traitMemo[t.Obj()]; ok {
			return v
		}
		return g.namedTraitsOnce(t)
	case *types.Pointer:
		if isUnitPtr(t) {
			return allTraits
		}
		et := g.elemTraits(t.Elem())
		if n, ok := t.Elem().(*types.Named); ok && n.Obj().Pkg() == g.pd.pkg && g.inlineReaches(n.Underlying(), n.Obj(), map[*types.TypeName]bool{}) {
			et &^= tCopy
		}
		return et | tDefault
	case *types.Slice:
		return (g.elemTraits(t.Elem()) &^ tCopy) | tDefault
	case *types.Array:
		e := g.elemTraits(t.Elem())
		if t.Len() > 32 {
			e &^= tDefault
		}
		return e
	case *types.Map:
		k := g.elemTraits(t.Key())
		v := g.elemTraits(t.Elem())
		return (k & v &^ tCopy) | tDefault
	case *types.Signature:
		// Option<Box<dyn Fn..>>
		return tDefault
	case *types.Chan:
		return tDebug | tDefault
	case *types.Struct:
		v := allTraits
		for i := 0; i < t.NumFields(); i++ {
			v &= g.traits(t.Field(i).Type())
		}
		return v
	case *types.Interface:
		return allTraits
	}
	panic(fmt.Sprintf("traits %T", t))
}

func (g *gen) elemTraits(t types.Type) trait {
	if p, ok := t.(*types.Pointer); ok {
		if isUnitPtr(p) {
			return allTraits
		}
		if n, ok := p.Elem().(*types.Named); ok && n.Obj().Pkg() != nil && n.Obj().Pkg() != g.pd.pkg {
			if nm, ok := stdNamed(n); ok && arcShared[nm] {
				return (stdTraits[nm] | tClone | tDefault) &^ tCopy
			}
		}
		return g.traits(p.Elem())
	}
	return g.traits(t)
}

func (g *gen) structFields(tn *types.TypeName, st *types.Struct) []*types.Var {
	hd := isHDFile(g.pd.fileOf(tn.Pos()).Name)
	var out []*types.Var
	for i := 0; i < st.NumFields(); i++ {
		fv := st.Field(i)
		if hd && !g.s.hdKept[fv] {
			continue
		}
		out = append(out, fv)
	}
	return out
}

func (g *gen) namedTraitsOnce(t *types.Named) trait {
	switch u := t.Underlying().(type) {
	case *types.Struct:
		v := allTraits
		bigArray := false
		for _, fv := range g.structFields(t.Obj(), u) {
			ft := g.traits(fv.Type())
			if hasBigArray(fv.Type()) && g.arrayElemDefault(fv.Type()) {
				ft |= tDefault
				bigArray = true
			}
			v &= ft
		}
		g.manualDefault[t.Obj()] = bigArray && v&tDefault != 0
		return v
	case *types.Basic:
		if u.Kind() == types.Float64 || u.Kind() == types.Float32 {
			return tClone | tCopy | tDebug | tDefault | tPartialEq
		}
		return allTraits
	case *types.Interface:
		// sfxVoice → AudioVoice
		return stdTraits["AudioVoice"]
	default:
		return g.traits(u)
	}
}

// computeTraits：對套件內具名型別做最大不動點。
func (g *gen) computeTraits() {
	var named []*types.Named
	for tn := range g.typeNames {
		named = append(named, tn.Type().(*types.Named))
		g.traitMemo[tn] = allTraits
	}
	sort.Slice(named, func(i, j int) bool { return named[i].Obj().Name() < named[j].Obj().Name() })
	for {
		changed := false
		for _, n := range named {
			v := g.namedTraitsOnce(n)
			if v != g.traitMemo[n.Obj()] {
				g.traitMemo[n.Obj()] = v
				changed = true
			}
		}
		if !changed {
			return
		}
	}
}

// inlineReaches：t 是否以值（含 Option<T> 的指標）包含 target。
func (g *gen) inlineReaches(t types.Type, target *types.TypeName, seen map[*types.TypeName]bool) bool {
	switch u := t.(type) {
	case *types.Named:
		if u.Obj() == target {
			return true
		}
		if u.Obj().Pkg() != g.pd.pkg || seen[u.Obj()] {
			return false
		}
		seen[u.Obj()] = true
		if st, ok := u.Underlying().(*types.Struct); ok {
			for _, fv := range g.structFields(u.Obj(), st) {
				if g.inlineReaches(fv.Type(), target, seen) {
					return true
				}
			}
			return false
		}
		return g.inlineReaches(u.Underlying(), target, seen)
	case *types.Pointer:
		if isUnitPtr(u) {
			return false
		}
		return g.inlineReaches(u.Elem(), target, seen)
	case *types.Array:
		return g.inlineReaches(u.Elem(), target, seen)
	case *types.Struct:
		for i := 0; i < u.NumFields(); i++ {
			if g.inlineReaches(u.Field(i).Type(), target, seen) {
				return true
			}
		}
	}
	return false
}

func (g *gen) markKeyType(t types.Type) {
	switch t := t.(type) {
	case *types.Named:
		if t.Obj().Pkg() != g.pd.pkg || g.keyTypes[t.Obj()] {
			return
		}
		g.keyTypes[t.Obj()] = true
		if s, ok := t.Underlying().(*types.Struct); ok {
			for i := 0; i < s.NumFields(); i++ {
				g.markKeyType(s.Field(i).Type())
			}
		}
	case *types.Array:
		g.markKeyType(t.Elem())
	case *types.Slice:
		g.markKeyType(t.Elem())
	case *types.Struct:
		for i := 0; i < t.NumFields(); i++ {
			g.markKeyType(t.Field(i).Type())
		}
	}
}

func deriveList(tr trait, key bool) []string {
	var d []string
	add := func(b trait, s string) {
		if tr&b != 0 {
			d = append(d, s)
		}
	}
	add(tDebug, "Debug")
	add(tClone, "Clone")
	if tr&tClone != 0 {
		add(tCopy, "Copy")
	}
	add(tDefault, "Default")
	add(tPartialEq, "PartialEq")
	if tr&tPartialEq != 0 {
		add(tEq, "Eq")
	}
	if key && tr&tEq != 0 {
		add(tOrd, "PartialOrd")
		add(tOrd, "Ord")
		add(tHash, "Hash")
	}
	return d
}

// hasBigArray：型別本身（不經指標）是超過 32 元素的陣列，或巢狀陣列裡有。
func hasBigArray(t types.Type) bool {
	a, ok := t.(*types.Array)
	if !ok {
		return false
	}
	return a.Len() > 32 || hasBigArray(a.Elem())
}

func (g *gen) arrayElemDefault(t types.Type) bool {
	for {
		a, ok := t.(*types.Array)
		if !ok {
			return g.traits(t)&tDefault != 0
		}
		t = a.Elem()
	}
}
