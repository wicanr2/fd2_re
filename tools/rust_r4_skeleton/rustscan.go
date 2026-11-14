package main

// 掃描既有 Rust crate（fd2-rules、fd2-formats、fd2-campaign）的公開項目：型別的 derive／手寫 trait、
// 函式與方法的接收者及參數型別。產生器用它把 Go 外部型別換成 Rust 路徑、決定能 derive 什麼，
// 並判斷呼叫外部方法時接收者會不會被改寫（Rust 方法收 `&self` 就是只讀）。
// （改寫自 R3 骨架產生器的同名檔。）

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type rustType struct {
	Mod    string
	Name   string
	Traits trait
	Kind   string // struct / enum / type / newtype
	// 型別帶生命週期參數（例如 indexedmap::NativeFrameInput<'a> 借用呼叫端的資源）
	Lifetime bool
}

type rustParam struct {
	Name string
	Type string
}

type rustFn struct {
	Mod    string
	Owner  string
	Name   string
	Self   string // "", "self", "&self", "&mut self"
	Params []rustParam
	Ret    string
	Vis    string
	File   string
}

type rustScanResult struct {
	types  map[string]*rustType // mod.Name
	fns    map[string]*rustFn   // mod.Owner.name / mod.name
	consts map[string]string    // mod.NAME -> kind
	norm   map[string]string    // mod.normalized -> actual key（型別與常數名稱比對用）
	reexp  map[string]bool      // mod.name：以 pub use 再匯出的名稱
}

var rustScan *rustScanResult

// Go 套件 → Rust 模組（fd2-game 內的別名）與原始碼位置（相對於儲存庫根）。
var rustMods = []struct{ goPkg, mod, path, file, dir string }{
	{"battle", "battle", "fd2_rules::battle", "remake-rs/crates/fd2-rules/src/battle.rs", "remake-rs/crates/fd2-rules/src/battle"},
	{"fdother", "fdother", "fd2_formats::fdother", "remake-rs/crates/fd2-formats/src/fdother.rs", "remake-rs/crates/fd2-formats/src/fdother"},
	{"fdicon", "fdicon", "fd2_formats::fdicon", "remake-rs/crates/fd2-formats/src/fdicon.rs", "remake-rs/crates/fd2-formats/src/fdicon"},
	{"fdtxt", "fdtxt", "fd2_formats::fdtxt", "remake-rs/crates/fd2-formats/src/fdtxt.rs", "remake-rs/crates/fd2-formats/src/fdtxt"},
	{"dato", "dato", "fd2_formats::dato", "remake-rs/crates/fd2-formats/src/dato.rs", "remake-rs/crates/fd2-formats/src/dato"},
	{"fdsave", "fdsave", "fd2_formats::save", "remake-rs/crates/fd2-formats/src/save.rs", "remake-rs/crates/fd2-formats/src/save"},
	{"afm", "afm", "fd2_formats::afm", "remake-rs/crates/fd2-formats/src/afm.rs", "remake-rs/crates/fd2-formats/src/afm"},
	{"figani", "figani", "fd2_formats::figani", "remake-rs/crates/fd2-formats/src/figani.rs", "remake-rs/crates/fd2-formats/src/figani"},
	{"archive", "archive", "fd2_formats::archive", "remake-rs/crates/fd2-formats/src/archive.rs", ""},
	{"campaign", "campaign", "fd2_campaign::campaign", "remake-rs/crates/fd2-campaign/src/campaign.rs", "remake-rs/crates/fd2-campaign/src/campaign"},
	{"ending", "ending", "fd2_campaign::ending", "remake-rs/crates/fd2-campaign/src/ending.rs", "remake-rs/crates/fd2-campaign/src/ending"},
	{"indexedmap", "indexedmap", "fd2_campaign::indexedmap", "remake-rs/crates/fd2-campaign/src/indexedmap.rs", "remake-rs/crates/fd2-campaign/src/indexedmap"},
	{"battlepresent", "battlepresent", "fd2_campaign::battlepresent", "remake-rs/crates/fd2-campaign/src/battlepresent.rs", "remake-rs/crates/fd2-campaign/src/battlepresent"},
	{"localization", "localization", "fd2_campaign::localization", "remake-rs/crates/fd2-campaign/src/localization.rs", "remake-rs/crates/fd2-campaign/src/localization"},
	{"musiccatalog", "musiccatalog", "fd2_campaign::musiccatalog", "remake-rs/crates/fd2-campaign/src/musiccatalog.rs", "remake-rs/crates/fd2-campaign/src/musiccatalog"},
	{"editorcanonical", "editorcanonical", "fd2_campaign::editorcanonical", "remake-rs/crates/fd2-campaign/src/editorcanonical.rs", "remake-rs/crates/fd2-campaign/src/editorcanonical"},
}

func rustModFor(goPkg string) string {
	for _, m := range rustMods {
		if m.goPkg == goPkg {
			return m.mod
		}
	}
	return ""
}

func normName(s string) string { return strings.ToLower(strings.ReplaceAll(s, "_", "")) }

var (
	reDerive  = regexp.MustCompile(`^\s*#\[derive\(([^)]*)\)\]`)
	reType    = regexp.MustCompile(`^pub (struct|enum|type) ([A-Za-z0-9_]+)`)
	reNewtype = regexp.MustCompile(`^pub struct ([A-Za-z0-9_]+)\(`)
	reImpl    = regexp.MustCompile(`^impl(?:<[^>]*>)? (?:([A-Za-z0-9_:]+)(?:<[^>]*>)? for )?([A-Za-z0-9_]+)`)
	reFn      = regexp.MustCompile(`^\s*(pub(?:\([a-z]+\))?) fn ([A-Za-z0-9_]+)`)
	reConst   = regexp.MustCompile(`^pub (const|static) ([A-Z0-9_]+)\s*:`)
)

func scanRust(repo string) *rustScanResult {
	r := &rustScanResult{types: map[string]*rustType{}, fns: map[string]*rustFn{}, consts: map[string]string{}, norm: map[string]string{}, reexp: map[string]bool{}}
	for _, m := range rustMods {
		var files []string
		if _, err := os.Stat(filepath.Join(repo, m.file)); err == nil {
			files = append(files, filepath.Join(repo, m.file))
		}
		if m.dir != "" {
			filepath.Walk(filepath.Join(repo, m.dir), func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					if info.Name() == "tests" {
						return filepath.SkipDir
					}
					return nil
				}
				if strings.HasSuffix(p, ".rs") && info.Name() != "tests.rs" && !strings.HasSuffix(p, "_test.rs") {
					files = append(files, p)
				}
				return nil
			})
		}
		sort.Strings(files)
		for _, f := range files {
			r.scanFile(m.mod, f)
		}
	}
	return r
}

func traitOf(name string) trait {
	switch strings.TrimSpace(name) {
	case "Clone":
		return tClone
	case "Copy":
		return tCopy
	case "Debug", "fmt::Debug", "std::fmt::Debug":
		return tDebug
	case "Default":
		return tDefault
	case "PartialEq":
		return tPartialEq
	case "Eq":
		return tEq
	case "Ord":
		return tOrd
	case "Hash", "std::hash::Hash":
		return tHash
	}
	return 0
}

func (r *rustScanResult) scanFile(mod, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	var pendingDerive trait
	owner := ""
	inTests := false
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if strings.HasPrefix(l, "#[cfg(test)]") {
			inTests = true
			continue
		}
		if inTests {
			// 跳過 #[cfg(test)] 之後的單一項目（mod xxx; 或 mod tests { ... }）
			if strings.HasPrefix(l, "mod ") && strings.HasSuffix(strings.TrimSpace(l), ";") {
				inTests = false
				continue
			}
			if strings.HasPrefix(l, "}") {
				inTests = false
			}
			continue
		}
		if strings.HasPrefix(l, "pub use ") {
			stmt := l
			for j := i + 1; j < len(lines) && !strings.Contains(stmt, ";"); j++ {
				stmt += " " + strings.TrimSpace(lines[j])
				i = j
			}
			stmt = strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(stmt, "pub use ")), ";")
			if b := strings.Index(stmt, "{"); b >= 0 {
				stmt = strings.TrimSuffix(stmt[b+1:], "}")
			} else if c := strings.LastIndex(stmt, "::"); c >= 0 {
				stmt = stmt[c+2:]
			}
			for _, part := range strings.Split(stmt, ",") {
				part = strings.TrimSpace(part)
				if a := strings.Index(part, " as "); a >= 0 {
					part = strings.TrimSpace(part[a+4:])
				}
				if c := strings.LastIndex(part, "::"); c >= 0 {
					part = part[c+2:]
				}
				if part != "" && part != "*" {
					r.reexp[mod+"."+part] = true
				}
			}
			continue
		}
		if m := reDerive.FindStringSubmatch(l); m != nil {
			for _, d := range strings.Split(m[1], ",") {
				pendingDerive |= traitOf(d)
			}
			continue
		}
		if m := reType.FindStringSubmatch(l); m != nil {
			k := mod + "." + m[2]
			kind := m[1]
			if reNewtype.MatchString(l) {
				kind = "newtype"
			}
			t := r.types[k]
			if t == nil {
				t = &rustType{Mod: mod, Name: m[2]}
				r.types[k] = t
			}
			t.Traits |= pendingDerive
			t.Kind = kind
			t.Lifetime = strings.HasPrefix(l[len(m[0]):], "<'")
			if kind == "type" {
				t.Traits = allTraits // 型別別名：traits 取決於目標，視為不限制
			}
			r.norm[mod+"."+normName(m[2])] = k
			pendingDerive = 0
			continue
		}
		if !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t") && !strings.HasPrefix(l, "#") && !strings.HasPrefix(l, "//") {
			pendingDerive = 0
		}
		if m := reConst.FindStringSubmatch(l); m != nil {
			r.consts[mod+"."+m[2]] = m[1]
			continue
		}
		if m := reImpl.FindStringSubmatch(l); m != nil && strings.HasPrefix(l, "impl") {
			tr, ty := m[1], m[2]
			if tr != "" {
				if t := traitOf(tr); t != 0 {
					k := mod + "." + ty
					if r.types[k] == nil {
						r.types[k] = &rustType{Mod: mod, Name: ty}
					}
					r.types[k].Traits |= t
				}
				owner = "" // trait impl 內的方法不算公開 API
				if strings.HasSuffix(strings.TrimSpace(l), "{}") {
					continue
				}
				owner = "!trait"
			} else {
				owner = ty
			}
			continue
		}
		if l == "}" {
			owner = ""
			continue
		}
		if m := reFn.FindStringSubmatch(l); m != nil {
			if owner == "!trait" {
				continue
			}
			indented := strings.HasPrefix(l, " ")
			if indented && owner == "" {
				continue // 函式內的巢狀函式或 trait 定義
			}
			if !indented && owner != "" {
				owner = ""
			}
			// 收集簽名到 `{` 或 `;`
			sig := l
			for j := i + 1; j < len(lines) && !strings.Contains(sig, "{") && !strings.HasSuffix(strings.TrimSpace(sig), ";"); j++ {
				sig += " " + strings.TrimSpace(lines[j])
			}
			fn := parseRustSig(sig)
			fn.Mod = mod
			fn.Owner = owner
			fn.Name = m[2]
			fn.Vis = m[1]
			fn.File = path
			k := mod + "." + m[2]
			if owner != "" {
				k = mod + "." + owner + "." + m[2]
			}
			if _, dup := r.fns[k]; !dup {
				r.fns[k] = fn
			}
		}
	}
}

// parseRustSig 拆出參數清單與回傳型別（只處理單層泛型，足以判斷 &mut）。
func parseRustSig(sig string) *rustFn {
	fn := &rustFn{}
	open := strings.Index(sig, "(")
	if open < 0 {
		return fn
	}
	depth := 0
	end := -1
	for i := open; i < len(sig); i++ {
		switch sig[i] {
		case '(', '<', '[':
			depth++
		case ')', '>', ']':
			if sig[i] == '>' && i > 0 && sig[i-1] == '-' {
				continue
			}
			depth--
			if depth == 0 && sig[i] == ')' {
				end = i
			}
		}
		if end >= 0 {
			break
		}
	}
	if end < 0 {
		return fn
	}
	inner := sig[open+1 : end]
	var parts []string
	depth = 0
	start := 0
	for i := 0; i < len(inner); i++ {
		switch inner[i] {
		case '(', '<', '[':
			depth++
		case ')', ']':
			depth--
		case '>':
			if i > 0 && inner[i-1] == '-' {
				continue
			}
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, inner[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, inner[start:])
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "self" || p == "&self" || p == "&mut self" || p == "mut self" {
			fn.Self = p
			continue
		}
		c := strings.Index(p, ":")
		if c < 0 {
			continue
		}
		fn.Params = append(fn.Params, rustParam{Name: strings.TrimSpace(strings.TrimPrefix(p[:c], "mut ")), Type: strings.TrimSpace(p[c+1:])})
	}
	rest := sig[end+1:]
	if a := strings.Index(rest, "->"); a >= 0 {
		r := rest[a+2:]
		if b := strings.Index(r, "{"); b >= 0 {
			r = r[:b]
		}
		r = strings.TrimSuffix(strings.TrimSpace(r), ";")
		if w := strings.Index(r, " where "); w >= 0 {
			r = r[:w]
		}
		fn.Ret = strings.TrimSpace(r)
	}
	return fn
}

// lookupType 以 Go 套件與型別名稱找 Rust 型別。
func (r *rustScanResult) lookupType(goPkg, goName string) *rustType {
	mod := rustModFor(goPkg)
	if t := r.types[mod+"."+camel(goName)]; t != nil {
		return t
	}
	if k, ok := r.norm[mod+"."+normName(goName)]; ok {
		return r.types[k]
	}
	return nil
}

// lookupFn 以 Go 鍵（pkg.Owner.Name 或 pkg.Name）找 Rust 函式。
func (r *rustScanResult) lookupFn(goPkg, owner, name string) *rustFn {
	mod := rustModFor(goPkg)
	if owner != "" {
		ot := r.lookupType(goPkg, owner)
		on := camel(owner)
		if ot != nil {
			on = ot.Name
		}
		if f := r.fns[mod+"."+on+"."+snake(name)]; f != nil {
			return f
		}
		for k, f := range r.fns {
			if f.Mod == mod && f.Owner == on && normName(f.Name) == normName(name) {
				_ = k
				return f
			}
		}
		return nil
	}
	if f := r.fns[mod+"."+snake(name)]; f != nil {
		return f
	}
	for _, f := range r.fns {
		if f.Mod == mod && f.Owner == "" && normName(f.Name) == normName(name) {
			return f
		}
	}
	if r.reexp[mod+"."+snake(name)] {
		var keys []string
		for k := range r.fns {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if f := r.fns[k]; f.Owner == "" && f.Name == snake(name) {
				return f
			}
		}
	}
	return nil
}

func (r *rustScanResult) lookupConst(goPkg, goName string) string {
	mod := rustModFor(goPkg)
	up := strings.ToUpper(snake(goName))
	if _, ok := r.consts[mod+"."+up]; ok {
		return up
	}
	for k := range r.consts {
		if strings.HasPrefix(k, mod+".") && normName(strings.TrimPrefix(k, mod+".")) == normName(goName) {
			return strings.TrimPrefix(k, mod+".")
		}
	}
	return ""
}
