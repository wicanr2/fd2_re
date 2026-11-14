// fd2-golden-present-loc 輸出 internal/localization 與 internal/musiccatalog 的黃金差分資料，供 Rust 改寫
// remake-rs/crates/fd2-campaign 逐欄比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md，R3 present 組）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫兩個套件的公開 API，不改任何遊戲行為。
//   - 真實輸入：-assets 指向的 remake/assets（locales/ 全部語系檔與 music_catalog.json，加上未入庫的
//     music_fm／music_mt32 render）。檔案清單排序後連同大小與 SHA-256 寫進 JSON，Rust 端讀同一批檔案前先核對。
//   - 官方語系：四個語系的 LoadOfficial 結果、每個鍵以固定種子產生參數的 Format 結果（含型別錯誤與個數錯誤）、
//     LoadOfficialContent 每筆 string_id 的 StoryText 結果、LoadOfficialEntities 各 ID 範圍的查詢結果。
//   - 變造輸入：以操作描述（JSON 路徑上的 set／delete、序列化後附加位元組）改寫真實檔案或合成小檔，記錄
//     Load* 的錯誤文字。Rust 端對同一份 JSON 樹套用同樣的操作再讀。
//   - 樣式與佔位符：以固定種子產生字串，經公開 API（LoadOfficial、Load、LoadOfficialContent）間接檢驗
//     locale／key／font path／pack id／SHA-256 樣式與兩種佔位符掃描；佔位符清單由本工具逐字複製的掃描函式
//     算出，並以真實 Load 驗證（不成立就 panic），Rust 端再以 Load 成立與「多一個變數就失敗」確認相等。
//   - Format 的動詞與旗標：合成官方語系包，一筆一個佔位符，記錄各種參數型別的輸出。
//
// 錯誤欄位是 Go err.Error() 原文；暫存目錄前綴換成 <tmp>。Go 標準函式庫產生的錯誤（讀檔、JSON 解碼）
// 只比本套件自己的前綴，見 Rust 端。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/localization"
	"github.com/wicanr2/fd2_re/remake/internal/musiccatalog"
)

type obj = map[string]any

type caseJSON struct {
	Index  int    `json:"index"`
	Op     string `json:"op"`
	Input  obj    `json:"input"`
	Result obj    `json:"result"`
}

var locales = []string{"en", "ja", "zh-Hans", "zh-Hant"}

type exporter struct {
	rng    *rand.Rand
	tmp    string
	assets string
	cases  []caseJSON
	files  []obj
	serial int
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func short(s string) string { return digest([]byte(s))[:16] }

func (e *exporter) errv(err error) any {
	if err == nil {
		return nil
	}
	return strings.ReplaceAll(err.Error(), e.tmp, "<tmp>")
}

func (e *exporter) add(op string, input, result obj) {
	e.cases = append(e.cases, caseJSON{Index: len(e.cases), Op: op, Input: input, Result: result})
}

func (e *exporter) intn(n int) int   { return e.rng.IntN(n) }
func (e *exporter) chance(p int) bool { return e.rng.IntN(100) < p }

func (e *exporter) record(label, path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	e.files = append(e.files, obj{"path": label, "size": len(raw), "sha256": digest(raw)})
}

func (e *exporter) recordTree(rel string) {
	var paths []string
	err := filepath.WalkDir(filepath.Join(e.assets, rel), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil || len(paths) == 0 {
		panic(fmt.Sprintf("找不到 %s/%s", e.assets, rel))
	}
	sort.Strings(paths)
	for _, path := range paths {
		r, err := filepath.Rel(e.assets, path)
		if err != nil {
			panic(err)
		}
		e.record(filepath.ToSlash(r), path)
	}
}

// newDir 建立 <tmp>/<序號>，回傳完整路徑與目錄名（寫進案例輸入，Rust 端在自己的暫存根目錄下用同名目錄）。
func (e *exporter) newDir() (string, string) {
	e.serial++
	name := fmt.Sprintf("d%04d", e.serial)
	dir := filepath.Join(e.tmp, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	return dir, name
}

func readTree(path string) any {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		panic(err)
	}
	return v
}

// ---- 變造操作 ----

// mutation 是一個對 JSON 樹的操作：op 為 set（設值）、delete（刪除鍵）、rename（改鍵名）；
// path 的每段是物件鍵（字串）或陣列索引（數字）。
type mutation struct {
	Op    string `json:"op"`
	Path  []any  `json:"path"`
	Value any    `json:"value"`
	To    string `json:"to,omitempty"`
}

func clone(v any) any {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	return out
}

func apply(tree any, m mutation) {
	parent := tree
	for _, seg := range m.Path[:len(m.Path)-1] {
		switch s := seg.(type) {
		case string:
			parent = parent.(map[string]any)[s]
		case int:
			parent = parent.([]any)[s]
		}
	}
	last := m.Path[len(m.Path)-1]
	switch m.Op {
	case "set":
		switch s := last.(type) {
		case string:
			parent.(map[string]any)[s] = clone(m.Value)
		case int:
			parent.([]any)[s] = clone(m.Value)
		}
	case "delete":
		delete(parent.(map[string]any), last.(string))
	case "rename":
		mp := parent.(map[string]any)
		v := mp[last.(string)]
		delete(mp, last.(string))
		mp[m.To] = v
	default:
		panic(m.Op)
	}
}

func set(value any, path ...any) mutation { return mutation{Op: "set", Path: path, Value: value} }
func del(path ...any) mutation             { return mutation{Op: "delete", Path: path} }
func rename(to string, path ...any) mutation {
	return mutation{Op: "rename", Path: path, To: to}
}

// writeMutated 把 base 樹套用 ops 後序列化、附加 suffix，寫到 path。
func writeMutated(path string, base any, ops []mutation, suffix string) {
	tree := clone(base)
	for _, m := range ops {
		apply(tree, m)
	}
	raw, err := json.Marshal(tree)
	if err != nil {
		panic(err)
	}
	raw = append(raw, suffix...)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		panic(err)
	}
}

// ---- 官方語系 ----

func catalogJSON(c *localization.Catalog) obj {
	entries := obj{}
	for k, v := range c.Entries {
		entries[k] = obj{"text": v.Text, "variables": v.Variables, "source_string_id": v.SourceStringID}
	}
	return obj{"locale": c.Locale, "kind": c.Kind, "base_locale": c.BaseLocale, "layout_profile": c.LayoutProfile,
		"font": obj{"profile": c.Font.Profile, "path": c.Font.Path}, "entries": entries}
}

type fmtArg struct {
	T string `json:"t"`
	V any    `json:"v"`
}

func goArgs(args []fmtArg) []any {
	out := make([]any, len(args))
	for i, a := range args {
		switch a.T {
		case "i":
			out[i] = a.V.(int)
		case "s":
			out[i] = a.V.(string)
		case "f":
			out[i] = a.V.(float64)
		}
	}
	return out
}

var sampleStrings = []string{"", "A", "悠妮", "Lv.12", "a b", "%d", "\"q\"", "tab\t", "ÄÖÜ", "長長的名字"}

func (e *exporter) randomArg(kind byte) fmtArg {
	switch kind {
	case 's':
		return fmtArg{"s", sampleStrings[e.intn(len(sampleStrings))]}
	case 'f':
		vals := []float64{0, 0.5, 1.5, 2.5, -0.4, 99.9, 12.49, 1e6 + 0.5, -3.5}
		if e.chance(50) {
			return fmtArg{"f", vals[e.intn(len(vals))]}
		}
		return fmtArg{"f", float64(e.intn(20000)-10000) / 8}
	default:
		vals := []int{0, 1, 9, 10, 99, 100, 255, 256, 999, 1000, -1, -255, 4096, 65535}
		if e.chance(50) {
			return fmtArg{"i", vals[e.intn(len(vals))]}
		}
		return fmtArg{"i", e.intn(2000) - 1000}
	}
}

func (e *exporter) argsFor(vars []string) []fmtArg {
	out := make([]fmtArg, len(vars))
	for i, v := range vars {
		verb := v[len(v)-1]
		switch verb {
		case 's', 'q':
			out[i] = e.randomArg('s')
		case 'f', 'F':
			out[i] = e.randomArg('f')
		default:
			out[i] = e.randomArg('i')
		}
	}
	return out
}

func (e *exporter) genOfficialPacks() {
	root := filepath.Join(e.assets, "locales")
	for _, locale := range locales {
		catalog, err := localization.LoadOfficial(root, locale)
		res := obj{"error": e.errv(err)}
		if err == nil {
			res["catalog"] = catalogJSON(catalog)
		}
		e.add("load_official", obj{"locale": locale}, res)
		if err != nil {
			continue
		}
		keys := make([]string, 0, len(catalog.Entries))
		for k := range catalog.Entries {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, key := range keys {
			vars := catalog.Entries[key].Variables
			variants := [][]fmtArg{e.argsFor(vars)}
			if len(vars) > 0 && i%3 == 0 {
				wrong := e.argsFor(vars)
				j := e.intn(len(wrong))
				if wrong[j].T == "s" {
					wrong[j] = e.randomArg('i')
				} else {
					wrong[j] = e.randomArg('s')
				}
				variants = append(variants, wrong)
			}
			if i%7 == 0 {
				variants = append(variants, append(e.argsFor(vars), e.randomArg('i')))
			}
			for _, args := range variants {
				got, err := catalog.Format(key, goArgs(args)...)
				e.add("format", obj{"locale": locale, "key": key, "args": args}, obj{"text": got, "error": e.errv(err)})
			}
		}
		_, err = catalog.Format("missing.key")
		e.add("format", obj{"locale": locale, "key": "missing.key", "args": []fmtArg{}}, obj{"text": "", "error": e.errv(err)})
	}
}

func (e *exporter) genOfficialContent() {
	root := filepath.Join(e.assets, "locales")
	for _, locale := range locales {
		catalog, err := localization.LoadOfficialContent(root, locale)
		res := obj{"error": e.errv(err)}
		if err != nil {
			e.add("content", obj{"locale": locale}, res)
			continue
		}
		res["locale"], res["inventory_sha256"] = catalog.Locale, catalog.InventorySHA256
		res["glossary_sha256"], res["translation_engine"] = catalog.GlossarySHA256, catalog.TranslationEngine
		tree := readTree(filepath.Join(root, locale, "content.json")).(map[string]any)
		var lines []any
		for _, raw := range tree["entries"].([]any) {
			id := raw.(map[string]any)["string_id"].(string)
			lineID := strings.TrimSuffix(id, "/text")
			text, err := catalog.StoryText(lineID)
			if err != nil {
				lines = append(lines, []any{lineID, nil, e.errv(err)})
			} else {
				lines = append(lines, []any{lineID, short(text), nil})
			}
		}
		for _, id := range []string{"missing", "", "/text", "legacy/line/7ecb566a60db/scenes/0/lines/0/text"} {
			text, err := catalog.StoryText(id)
			if err != nil {
				lines = append(lines, []any{id, nil, e.errv(err)})
			} else {
				lines = append(lines, []any{id, short(text), nil})
			}
		}
		res["lines"] = lines
		e.add("content", obj{"locale": locale}, res)
	}
}

func (e *exporter) genOfficialEntities() {
	root := filepath.Join(e.assets, "locales")
	for _, locale := range locales {
		catalog, err := localization.LoadOfficialEntities(root, locale)
		res := obj{"error": e.errv(err)}
		if err == nil {
			res["locale"] = catalog.Locale
			res["lookups"] = e.entityLookups(catalog)
		}
		e.add("entities", obj{"locale": locale}, res)
	}
}

func (e *exporter) entityLookups(c *localization.EntityCatalog) obj {
	table := func(lo, hi int, f func(int) (string, error)) []any {
		var out []any
		for id := lo; id <= hi; id++ {
			name, err := f(id)
			out = append(out, []any{id, name, e.errv(err)})
		}
		return out
	}
	return obj{
		"item":      table(-2, 260, c.ItemName),
		"character": table(-1, 33, c.CharacterName),
		"battle":    table(-1, 140, c.BattleName),
		"command":   table(-1, 37, c.CommandName),
		"class":     table(-1, 30, c.ClassName),
	}
}

// ---- 變造：pack.json ----

const validPack = `{"schema_version":1,"pack_id":"test-en","locale":"en","kind":"official","layout_profile":"dos-8x8-latin","font":{"profile":"fd2-latin-8x8","path":"fonts/atlas.png"},"entries":{"battle.attack.hit":{"text":"%s deals %d damage","variables":["%s","%d"],"source_string_id":"native-1"}}}`

func (e *exporter) runPack(label string, base any, ops []mutation, suffix string, official bool, locale string) {
	dir, name := e.newDir()
	input := obj{"label": label, "ops": ops, "suffix": suffix, "official": official, "locale": locale, "dir": name}
	var err error
	var catalog *localization.Catalog
	if official {
		writeMutated(filepath.Join(dir, locale, "pack.json"), base, ops, suffix)
		catalog, err = localization.LoadOfficial(dir, locale)
	} else {
		writeMutated(filepath.Join(dir, "pack.json"), base, ops, suffix)
		catalog, err = localization.Load(filepath.Join(dir, "pack.json"))
	}
	res := obj{"error": e.errv(err)}
	if err == nil {
		res["catalog"] = catalogJSON(catalog)
	}
	e.add("pack", input, res)
}

func (e *exporter) genPackMutations() {
	var base any
	if err := json.Unmarshal([]byte(validPack), &base); err != nil {
		panic(err)
	}
	k := "battle.attack.hit"
	cases := []struct {
		label  string
		ops    []mutation
		suffix string
	}{
		{"valid", nil, ""},
		{"schema2", []mutation{set(2, "schema_version")}, ""},
		{"schema-null", []mutation{set(nil, "schema_version")}, ""},
		{"packid-short", []mutation{set("a", "pack_id")}, ""},
		{"packid-upper", []mutation{set("Test", "pack_id")}, ""},
		{"packid-dot", []mutation{set("a.b_c-d", "pack_id")}, ""},
		{"locale-upper", []mutation{set("EN", "locale")}, ""},
		{"kind-x", []mutation{set("x", "kind")}, ""},
		{"official-base", []mutation{set("zh-Hant", "base_locale")}, ""},
		{"community-nobase", []mutation{set("community", "kind")}, ""},
		{"community-same", []mutation{set("community", "kind"), set("en", "base_locale")}, ""},
		{"community-ok", []mutation{set("community", "kind"), set("ja", "base_locale")}, ""},
		{"layout", []mutation{set("pc98-16x16-cjk", "layout_profile")}, ""},
		{"layout-bad", []mutation{set("vga", "layout_profile")}, ""},
		{"font-profile", []mutation{set("fd2-cjk-16x16", "font", "profile")}, ""},
		{"font-profile-bad", []mutation{set("x", "font", "profile")}, ""},
		{"font-null", []mutation{set(nil, "font")}, ""},
		{"entries-empty", []mutation{set(obj{}, "entries")}, ""},
		{"entries-null", []mutation{set(nil, "entries")}, ""},
		{"text-empty", []mutation{set("", "entries", k, "text")}, ""},
		{"source-empty", []mutation{set("", "entries", k, "source_string_id")}, ""},
		{"vars-null", []mutation{set(nil, "entries", k, "variables")}, ""},
		{"vars-mismatch", []mutation{set([]any{"%d", "%s"}, "entries", k, "variables")}, ""},
		{"unknown-top", []mutation{set(true, "future")}, ""},
		{"unknown-font", []mutation{set(true, "font", "future")}, ""},
		{"unknown-entry", []mutation{set(true, "entries", k, "future")}, ""},
		{"trailing-object", nil, "{}"},
		{"trailing-space", nil, " \n\t"},
		{"trailing-garbage", nil, "x"},
		{"trailing-truncated", nil, "{"},
		{"missing-locale", []mutation{del("locale")}, ""},
		{"missing-kind", []mutation{del("kind")}, ""},
		{"wrong-type", []mutation{set("1", "schema_version")}, ""},
	}
	for _, c := range cases {
		e.runPack(c.label, base, c.ops, c.suffix, false, "")
	}
	for _, key := range []string{"legacy.x.y", "Battle.x", "a", "a.", ".a", "a.b_c", "a1.b", "a.B", "a..b", "a.b.c_d.e9",
		"1a.b", "a-b.c", "ab.c-d", "x.y z"} {
		e.runPack("key:"+key, base, []mutation{rename(key, "entries", k)}, "", false, "")
	}
	for _, path := range []string{"", "fonts/a.png", "fonts/../../x.png", "fonts/a/../b.png", "fonts/../x.png", "/abs.png",
		"fonts\\a.png", "fonts/A.png", "fonts/a.txt", "fonts/.png", "fonts/a.json", "fonts/a.bin", "fonts/a_b-c.d/e.png",
		"fonts/a/../../..", "fonts/png", "fonts/a.png/", "fonts/a\x00.png", "fonts/.././a.png", "fonts//a.png", "Fonts/a.png"} {
		e.runPack("font:"+path, base, []mutation{set(path, "font", "path")}, "", false, "")
	}
	// LoadOfficial：路徑與身分
	for _, loc := range []string{"en", "ja"} {
		e.runPack("official:"+loc, base, nil, "", true, loc)
	}
	e.runPack("official-kind", base, []mutation{set("community", "kind"), set("ja", "base_locale")}, "", true, "en")
	// 隨機組合
	ops := []func() mutation{
		func() mutation { return set(e.intn(3), "schema_version") },
		func() mutation { return set([]string{"official", "community", "x"}[e.intn(3)], "kind") },
		func() mutation { return set([]string{"", "zh-Hant", "en", "ja-JP"}[e.intn(4)], "base_locale") },
		func() mutation { return set([]string{"", "x", "%d %s", "%s deals %d damage"}[e.intn(4)], "entries", k, "text") },
		func() mutation {
			return set([][]any{{}, {"%s"}, {"%s", "%d"}, {"%d"}}[e.intn(4)], "entries", k, "variables")
		},
		func() mutation { return set([]string{"en", "ja", "zh-Hans", "EN", "e"}[e.intn(5)], "locale") },
	}
	for i := 0; i < 60; i++ {
		var chosen []mutation
		for j := 0; j < 1+e.intn(3); j++ {
			chosen = append(chosen, ops[e.intn(len(ops))]())
		}
		e.runPack(fmt.Sprintf("random:%d", i), base, chosen, "", false, "")
	}
	// 真實 en pack.json 的變造
	real := readTree(filepath.Join(e.assets, "locales", "en", "pack.json"))
	realKeys := sortedKeys(real.(map[string]any)["entries"].(map[string]any))
	for i := 0; i < 30; i++ {
		key := realKeys[e.intn(len(realKeys))]
		var m mutation
		switch e.intn(5) {
		case 0:
			m = set("", "entries", key, "text")
		case 1:
			m = set([]any{"%q"}, "entries", key, "variables")
		case 2:
			m = rename("legacy."+key, "entries", key)
		case 3:
			m = set("", "entries", key, "source_string_id")
		default:
			m = set(true, "entries", key, "extra")
		}
		e.runPack(fmt.Sprintf("real:%d", i), real, []mutation{m}, "", true, "en")
	}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ---- 樣式與佔位符 ----

// formatPlaceholdersCopy 逐字複製 localization/locale.go 的 formatPlaceholders（私有函式）；結果以真實 Load
// 驗證，不成立就 panic。
func formatPlaceholdersCopy(text string) []string {
	var out []string
	for i := 0; i < len(text); i++ {
		if text[i] != '%' {
			continue
		}
		if i+1 < len(text) && text[i+1] == '%' {
			i++
			continue
		}
		start := i
		i++
		for i < len(text) && strings.ContainsRune("+-# 0", rune(text[i])) {
			i++
		}
		for i < len(text) && text[i] >= '0' && text[i] <= '9' {
			i++
		}
		if i < len(text) && text[i] == '.' {
			i++
			for i < len(text) && text[i] >= '0' && text[i] <= '9' {
				i++
			}
		}
		if i < len(text) && strings.ContainsRune("vTtbcdoOqxXUeEfFgGsxp", rune(text[i])) {
			out = append(out, text[start:i+1])
		}
	}
	return out
}

// contentPlaceholderCopy 逐字複製 localization/content.go 的 contentPlaceholderPattern 與 contentPlaceholders。
var contentPlaceholderCopy = regexp.MustCompile(`%(?:%|[-+0-9.#]*[a-zA-Z])`)

func contentPlaceholdersCopy(text string) []string {
	matches := contentPlaceholderCopy.FindAllString(text, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if m != "%%" {
			out = append(out, m)
		}
	}
	return out
}

func (e *exporter) randomText() string {
	alphabet := []string{"%", "%", "%", "d", "s", "v", "x", "X", "f", ".", "0", "2", "5", "-", "+", "#", " ", "q",
		"z", "爾", "a", "T", "%%", "e", "g", "[", "*", "é"}
	n := 1 + e.intn(9)
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString(alphabet[e.intn(len(alphabet))])
	}
	return b.String()
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func (e *exporter) genPlaceholders() {
	var base any
	if err := json.Unmarshal([]byte(validPack), &base); err != nil {
		panic(err)
	}
	contentBase := e.syntheticContent(3)
	texts := []string{"%", "%%", "%%%", "%d%", "% d", "%-5.2f", "%+#x", "%5", "%.", "%.d", "%.5s", "100%", "%T", "%U",
		"%z", "%[1]d", "%*d", "%-+# 0d", "%é", "a%%b%dc", "%2.3.4d", "%%d", "%-%d", "%05.2fzz%q"}
	for i := 0; i < 150; i++ {
		texts = append(texts, e.randomText())
	}
	for _, text := range texts {
		vars := nonNil(formatPlaceholdersCopy(text))
		dir, _ := e.newDir()
		writeMutated(filepath.Join(dir, "pack.json"), base, []mutation{
			set(text, "entries", "battle.attack.hit", "text"), set(vars, "entries", "battle.attack.hit", "variables")}, "")
		if _, err := localization.Load(filepath.Join(dir, "pack.json")); err != nil && text != "" {
			panic(fmt.Sprintf("formatPlaceholders 複本與 Load 不一致：%q %v", text, err))
		}
		cvars := contentPlaceholdersCopy(text)
		cdir, _ := e.newDir()
		writeMutated(filepath.Join(cdir, "en", "content.json"), contentBase, []mutation{
			set(text, "entries", 0, "text"), set(cvars, "entries", 0, "variables")}, "")
		if _, err := localization.LoadOfficialContent(cdir, "en"); err != nil && text != "" {
			panic(fmt.Sprintf("contentPlaceholders 複本與 LoadOfficialContent 不一致：%q %v", text, err))
		}
		e.add("placeholders", obj{"text": text}, obj{"format": vars, "content": cvars})
	}
	// locale 樣式：LoadOfficial 在讀檔前先檢查 locale。
	cands := []string{"en", "ja", "zh-Hant", "zh-Hans", "zh-Hant-TW", "zh-TW", "es-419", "abc", "abcd", "a", "EN",
		"zh-hant", "zh-HANT", "zh-Hant-tw", "zh-Hant-x-test", "zh-", "zh-Ha", "zh-Hantt", "en-US-x", "en-1234", "en-12",
		"en-US ", " en", "en\n", "zh-Hant-419", "zh-419-TW", "sr-Latn-RS", "xyz-Abcd-AB", "en_US", ""}
	letters := "aZ-0zA9_"
	for i := 0; i < 60; i++ {
		n := 1 + e.intn(11)
		b := make([]byte, n)
		for j := range b {
			b[j] = letters[e.intn(len(letters))]
		}
		cands = append(cands, string(b))
	}
	empty, name := e.newDir()
	for _, c := range cands {
		_, err := localization.LoadOfficial(empty, c)
		_, err2 := localization.LoadOfficialContent(empty, c)
		_, err3 := localization.LoadOfficialEntities(empty, c)
		e.add("locale_pattern", obj{"locale": c, "dir": name}, obj{"pack": e.errv(err), "content": e.errv(err2), "entities": e.errv(err3)})
	}
}

// ---- 變造：content.json（合成小檔）與 entities.json ----

// syntheticContent 取真實 en content.json 的前 n 筆，entry_count 改成 n。
func (e *exporter) syntheticContent(n int) any {
	tree := readTree(filepath.Join(e.assets, "locales", "en", "content.json")).(map[string]any)
	entries := tree["entries"].([]any)
	tree["entries"] = clone(entries[:n])
	tree["entry_count"] = n
	return tree
}

func (e *exporter) runContent(label string, base any, ops []mutation, suffix, locale string) {
	dir, name := e.newDir()
	writeMutated(filepath.Join(dir, "en", "content.json"), base, ops, suffix)
	catalog, err := localization.LoadOfficialContent(dir, locale)
	res := obj{"error": e.errv(err)}
	if err == nil {
		res["locale"], res["inventory_sha256"] = catalog.Locale, catalog.InventorySHA256
		res["glossary_sha256"], res["translation_engine"] = catalog.GlossarySHA256, catalog.TranslationEngine
		tree := clone(base).(map[string]any)
		for _, m := range ops {
			apply(tree, m)
		}
		var lines []any
		for _, raw := range tree["entries"].([]any) {
			id, _ := raw.(map[string]any)["string_id"].(string)
			lineID := strings.TrimSuffix(id, "/text")
			text, err := catalog.StoryText(lineID)
			lines = append(lines, []any{lineID, text, e.errv(err)})
		}
		res["lines"] = lines
	}
	e.add("content_mut", obj{"label": label, "ops": ops, "suffix": suffix, "locale": locale, "dir": name}, res)
}

func (e *exporter) genContentMutations() {
	base := e.syntheticContent(12)
	cases := []struct {
		label  string
		ops    []mutation
		suffix string
	}{
		{"valid", nil, ""},
		{"schema", []mutation{set(2, "schema_version")}, ""},
		{"kind", []mutation{set("x", "kind")}, ""},
		{"locale", []mutation{set("ja", "locale")}, ""},
		{"source-locale", []mutation{set("zh-Hans", "source_locale")}, ""},
		{"count", []mutation{set(11, "entry_count")}, ""},
		{"count-zero", []mutation{set(0, "entry_count"), set([]any{}, "entries")}, ""},
		{"inventory-upper", []mutation{set(strings.Repeat("A", 64), "inventory_sha256")}, ""},
		{"inventory-short", []mutation{set(strings.Repeat("a", 63), "inventory_sha256")}, ""},
		{"glossary", []mutation{set("", "glossary_sha256")}, ""},
		{"pivot-null", []mutation{set(nil, "pivot_content_sha256")}, ""},
		{"pivot-missing", []mutation{del("pivot_content_sha256")}, ""},
		{"pivot-ok", []mutation{set(strings.Repeat("b", 64), "pivot_content_sha256")}, ""},
		{"pivot-bad", []mutation{set("zz", "pivot_content_sha256")}, ""},
		{"engine", []mutation{set("", "translation_engine")}, ""},
		{"empty-id", []mutation{set("", "entries", 3, "string_id")}, ""},
		{"empty-role", []mutation{set("", "entries", 4, "role")}, ""},
		{"empty-text", []mutation{set("", "entries", 5, "text")}, ""},
		{"empty-idstatus", []mutation{set("", "entries", 6, "id_status")}, ""},
		{"dup", []mutation{set(nil, "entries", 7, "string_id")}, ""},
		{"status", []mutation{set("final", "entries", 2, "status")}, ""},
		{"status-blocked", []mutation{set("blocked", "entries", 0, "status")}, ""},
		{"vars", []mutation{set([]any{"%d"}, "entries", 1, "variables")}, ""},
		{"role", []mutation{set("narration", "entries", 0, "role")}, ""},
		{"idstatus", []mutation{set("provisional_legacy_path", "entries", 0, "id_status")}, ""},
		{"source-null", []mutation{set(nil, "entries", 0, "source")}, ""},
		{"unknown-top", []mutation{set(1, "future")}, ""},
		{"unknown-entry", []mutation{set(1, "entries", 0, "future")}, ""},
		{"trailing-object", nil, "{}"},
		{"trailing-garbage", nil, "]"},
		{"bad-locale-arg", nil, ""},
	}
	tree := base.(map[string]any)
	first := tree["entries"].([]any)[0].(map[string]any)["string_id"]
	for _, c := range cases {
		ops := c.ops
		if c.label == "dup" {
			ops = []mutation{set(first, "entries", 7, "string_id")}
		}
		locale := "en"
		if c.label == "bad-locale-arg" {
			locale = "EN"
		}
		e.runContent(c.label, base, ops, c.suffix, locale)
	}
	for i := 0; i < 40; i++ {
		idx := e.intn(12)
		var m mutation
		switch e.intn(6) {
		case 0:
			m = set([]string{"source", "machine_draft", "reviewed", "blocked", "x"}[e.intn(5)], "entries", idx, "status")
		case 1:
			m = set([]string{"stable_canonical", "provisional_legacy_path", ""}[e.intn(3)], "entries", idx, "id_status")
		case 2:
			m = set([]string{"dialogue", "menu", ""}[e.intn(3)], "entries", idx, "role")
		case 3:
			m = set(e.randomText(), "entries", idx, "text")
		case 4:
			m = set([]any{}, "entries", idx, "variables")
		default:
			m = set(fmt.Sprintf("x/%d/text", e.intn(4)), "entries", idx, "string_id")
		}
		e.runContent(fmt.Sprintf("random:%d", i), base, []mutation{m}, "", "en")
	}
}

func (e *exporter) runEntities(label string, base any, ops []mutation, suffix string) {
	dir, name := e.newDir()
	writeMutated(filepath.Join(dir, "en", "entities.json"), base, ops, suffix)
	catalog, err := localization.LoadOfficialEntities(dir, "en")
	res := obj{"error": e.errv(err)}
	if err == nil {
		res["lookups"] = e.entityLookups(catalog)
	}
	e.add("entities_mut", obj{"label": label, "ops": ops, "suffix": suffix, "dir": name}, res)
}

func (e *exporter) genEntityMutations() {
	base := readTree(filepath.Join(e.assets, "locales", "en", "entities.json"))
	tree := base.(map[string]any)
	itemKeys := sortedKeys(tree["items"].(map[string]any))
	cases := []struct {
		label  string
		ops    []mutation
		suffix string
	}{
		{"valid", nil, ""},
		{"schema", []mutation{set(0, "schema_version")}, ""},
		{"kind", []mutation{set("x", "kind")}, ""},
		{"item-count", []mutation{set(1, "item_count")}, ""},
		{"char-count", []mutation{set(31, "character_count")}, ""},
		{"battle-count", []mutation{set(93, "battle_name_count")}, ""},
		{"command-count", []mutation{set(34, "command_name_count")}, ""},
		{"class-count", []mutation{set(28, "class_name_count")}, ""},
		{"item-key", []mutation{rename("abc", "items", itemKeys[0])}, ""},
		{"item-key-256", []mutation{rename("256", "items", itemKeys[0])}, ""},
		{"item-key-neg", []mutation{rename("-1", "items", itemKeys[0])}, ""},
		{"item-key-plus", []mutation{rename("+300", "items", itemKeys[0])}, ""},
		{"item-name", []mutation{set("", "items", itemKeys[1], "name")}, ""},
		{"item-sources", []mutation{set([]any{}, "items", itemKeys[2], "source_string_ids")}, ""},
		{"item-source-empty", []mutation{set([]any{"x", ""}, "items", itemKeys[3], "source_string_ids")}, ""},
		{"item-status", []mutation{set("original_placeholder", "items", itemKeys[4], "status")}, ""},
		{"item-status-missing", []mutation{del("items", itemKeys[5], "status")}, ""},
		{"char-key", []mutation{rename("32", "characters", "0")}, ""},
		{"char-sources", []mutation{set([]any{"x"}, "characters", "1", "source_string_ids")}, ""},
		{"char-status", []mutation{set("machine_draft", "characters", "2", "status")}, ""},
		{"battle-key", []mutation{rename("139", "battle_names", sortedKeys(tree["battle_names"].(map[string]any))[0])}, ""},
		{"battle-status", []mutation{set("machine_draft", "battle_names", sortedKeys(tree["battle_names"].(map[string]any))[1], "status")}, ""},
		{"command-31", []mutation{rename("31", "command_names", sortedKeys(tree["command_names"].(map[string]any))[0])}, ""},
		{"command-status", []mutation{set("x", "command_names", sortedKeys(tree["command_names"].(map[string]any))[2], "status")}, ""},
		{"class-placeholder", []mutation{set("original_confirmed", "class_names", "26", "status")}, ""},
		{"class-status", []mutation{set("original_placeholder", "class_names", "3", "status")}, ""},
		{"class-key", []mutation{rename("29", "class_names", "0")}, ""},
		{"unknown-top", []mutation{set(true, "future")}, ""},
		{"unknown-entry", []mutation{set(true, "items", itemKeys[6], "future")}, ""},
		{"trailing-object", nil, "[]"},
		{"trailing-garbage", nil, "}"},
	}
	for _, c := range cases {
		e.runEntities(c.label, base, c.ops, c.suffix)
	}
}

// ---- musiccatalog ----

func (e *exporter) musicResult(c *musiccatalog.Catalog, err error) obj {
	res := obj{"error": e.errv(err)}
	if err != nil {
		return res
	}
	tracks := []string{"FDMUS_999", ""}
	for _, t := range c.Tracks {
		tracks = append(tracks, t.TrackID)
	}
	var resolves []any
	for _, profile := range []string{"fm", "mt32", "legacy", ""} {
		for _, track := range tracks {
			path, err := c.Resolve(profile, track)
			resolves = append(resolves, []any{profile, track, strings.ReplaceAll(path, e.tmp, "<tmp>"), e.errv(err)})
		}
	}
	res["catalog"] = obj{"schema": c.Schema, "schema_version": c.SchemaVersion, "source": c.Source, "loop": c.Loop,
		"profiles": c.Profiles, "tracks": c.Tracks}
	res["resolves"] = resolves
	return res
}

func (e *exporter) runMusic(label string, base any, ops []mutation, suffix string, renders bool) {
	dir, name := e.newDir()
	writeMutated(filepath.Join(dir, "music_catalog.json"), base, ops, suffix)
	if renders {
		for _, d := range []string{"music_fm", "music_mt32"} {
			if err := os.Symlink(filepath.Join(e.assets, d), filepath.Join(dir, d)); err != nil {
				panic(err)
			}
		}
	}
	c, err := musiccatalog.Load(dir)
	e.add("music_mut", obj{"label": label, "ops": ops, "suffix": suffix, "renders": renders, "dir": name}, e.musicResult(c, err))
}

func (e *exporter) genMusic() {
	c, err := musiccatalog.Load(e.assets)
	e.add("music_load", obj{}, e.musicResult(c, err))
	base := readTree(filepath.Join(e.assets, "music_catalog.json"))
	cases := []struct {
		label   string
		ops     []mutation
		suffix  string
		renders bool
	}{
		{"valid", nil, "", true},
		{"no-renders", nil, "", false},
		{"schema", []mutation{set("x", "schema")}, "", true},
		{"version", []mutation{set(2, "schema_version")}, "", true},
		{"source-size", []mutation{set(1, "source", "size")}, "", true},
		{"source-md5", []mutation{set("", "source", "md5")}, "", true},
		{"loop-mode", []mutation{set("x", "loop", "mode")}, "", true},
		{"loop-counts", []mutation{set([]any{0}, "loop", "accepted_counts")}, "", true},
		{"loop-counts-order", []mutation{set([]any{1, 0}, "loop", "accepted_counts")}, "", true},
		{"loop-seam", []mutation{set("known", "loop", "seam_evidence")}, "", true},
		{"loop-level", []mutation{set("e2", "loop", "evidence_level")}, "", true},
		{"profiles-extra", []mutation{set(obj{"render_pipeline": "x", "provenance_status": "incomplete_legacy_render", "rights_note": "y"}, "profiles", "opl")}, "", true},
		{"profile-missing", []mutation{rename("opl", "profiles", "mt32")}, "", true},
		{"profile-pipeline", []mutation{set("", "profiles", "fm", "render_pipeline")}, "", true},
		{"profile-status", []mutation{set("complete", "profiles", "mt32", "provenance_status")}, "", true},
		{"profile-rights", []mutation{set("", "profiles", "fm", "rights_note")}, "", true},
		{"tracks-short", []mutation{set(clone(base.(map[string]any)["tracks"].([]any)[:14]), "tracks")}, "", true},
		{"track-id", []mutation{set("FDMUS_002", "tracks", 0, "track_id")}, "", true},
		{"track-resource", []mutation{set(2, "tracks", 1, "resource_index")}, "", true},
		{"track-renders", []mutation{del("tracks", 2, "renders", "mt32")}, "", true},
		{"track-renders-rename", []mutation{rename("opl", "tracks", 2, "renders", "mt32")}, "", true},
		{"render-path", []mutation{set("music_fm/../music_fm/FDMUS_001.ogg", "tracks", 0, "renders", "fm", "path")}, "", true},
		{"render-path-abs", []mutation{set("/music_fm/FDMUS_001.ogg", "tracks", 0, "renders", "fm", "path")}, "", true},
		{"render-bytes", []mutation{set(1, "tracks", 3, "renders", "fm", "bytes")}, "", true},
		{"render-sha", []mutation{set(strings.Repeat("0", 64), "tracks", 4, "renders", "mt32", "sha256")}, "", true},
		{"render-codec", []mutation{set("opus", "tracks", 5, "renders", "fm", "codec")}, "", true},
		{"render-channels", []mutation{set(1, "tracks", 6, "renders", "fm", "channels")}, "", true},
		{"render-rate", []mutation{set(44100, "tracks", 7, "renders", "mt32", "sample_rate")}, "", true},
		{"render-samples", []mutation{set(1, "tracks", 8, "renders", "fm", "pcm_samples")}, "", true},
		{"render-duration", []mutation{set(1, "tracks", 9, "renders", "mt32", "duration_ms")}, "", true},
		{"render-negative", []mutation{set(-1, "tracks", 9, "renders", "mt32", "duration_ms")}, "", true},
		{"unknown-top", []mutation{set(1, "future")}, "", true},
		{"unknown-render", []mutation{set(1, "tracks", 0, "renders", "fm", "future")}, "", true},
		{"unknown-byid", []mutation{set(obj{}, "byID")}, "", true},
		{"trailing-object", nil, "{}", true},
		{"trailing-space", nil, "\n", true},
		{"trailing-garbage", nil, "x", true},
	}
	for _, c := range cases {
		e.runMusic(c.label, base, c.ops, c.suffix, c.renders)
	}
	// 變造的 OGG：拿第一個 render 改位元組，catalog 的 bytes／sha256 跟著改成新檔，迫使 inspectVorbis 判斷。
	tracks := base.(map[string]any)["tracks"].([]any)
	first := tracks[0].(map[string]any)["renders"].(map[string]any)["fm"].(map[string]any)
	orig, err := os.ReadFile(filepath.Join(e.assets, first["path"].(string)))
	if err != nil {
		panic(err)
	}
	for i := 0; i < 24; i++ {
		data := append([]byte(nil), orig...)
		truncate := -1
		var patch [][2]int
		var extra []int
		label := ""
		poke := func(at int, v byte) {
			data[at] = v
			patch = append(patch, [2]int{at, int(v)})
		}
		switch i % 6 {
		case 0:
			at := e.intn(64)
			poke(at, data[at]^byte(1+e.intn(255)))
			label = fmt.Sprintf("ogg-head-%d", at)
		case 1:
			truncate = e.intn(len(data))
			data = data[:truncate]
			label = fmt.Sprintf("ogg-cut-%d", truncate)
		case 2:
			v := e.intn(256)
			data = append(data, byte(v))
			extra = append(extra, v)
			label = "ogg-append"
		case 3:
			// 識別封包的 channels（頁首 27+段表之後第 11 個位元組）
			at := 27 + int(data[26]) + 11
			poke(at, byte(1+e.intn(4)))
			label = fmt.Sprintf("ogg-channels-%d", data[at])
		case 4:
			poke(27+int(data[26])+7, 1)
			label = "ogg-version"
		default:
			at := 27 + int(data[26]) + 12
			poke(at, 0x44)
			poke(at+1, 0xac)
			poke(at+2, 0)
			poke(at+3, 0)
			label = "ogg-rate"
		}
		dir, name := e.newDir()
		for _, d := range []string{"music_fm", "music_mt32"} {
			if err := os.MkdirAll(filepath.Join(dir, d), 0o755); err != nil {
				panic(err)
			}
			entries, err := os.ReadDir(filepath.Join(e.assets, d))
			if err != nil {
				panic(err)
			}
			for _, entry := range entries {
				if err := os.Symlink(filepath.Join(e.assets, d, entry.Name()), filepath.Join(dir, d, entry.Name())); err != nil {
					panic(err)
				}
			}
		}
		target := filepath.Join(dir, first["path"].(string))
		if err := os.Remove(target); err != nil {
			panic(err)
		}
		if err := os.WriteFile(target, data, 0o644); err != nil {
			panic(err)
		}
		ops := []mutation{set(len(data), "tracks", 0, "renders", "fm", "bytes"), set(digest(data), "tracks", 0, "renders", "fm", "sha256")}
		writeMutated(filepath.Join(dir, "music_catalog.json"), base, ops, "")
		c, err := musiccatalog.Load(dir)
		e.add("music_ogg", obj{"label": label, "ops": ops, "data_sha256": digest(data), "truncate": truncate,
			"patch": patch, "append": extra, "orig": first["path"], "dir": name}, e.musicResult(c, err))
	}
}

// ---- Format 的動詞與旗標 ----

func (e *exporter) genFormatSpecs() {
	var base any
	if err := json.Unmarshal([]byte(validPack), &base); err != nil {
		panic(err)
	}
	flags := []string{"", "-", "+", "#", " ", "0", "-0", "+0", "#0", "0-", "+ ", " #"}
	widths := []string{"", "1", "3", "5", "8"}
	precs := []string{"", ".", ".0", ".2", ".5"}
	verbs := []string{"d", "v", "x", "X", "c", "s", "q", "f", "F", "e", "g", "b", "o", "t", "T", "U", "p"}
	for i := 0; i < 400; i++ {
		spec := "%" + flags[e.intn(len(flags))] + widths[e.intn(len(widths))] + precs[e.intn(len(precs))] + verbs[e.intn(len(verbs))]
		text := "<" + spec + ">"
		vars := nonNil(formatPlaceholdersCopy(text))
		if len(vars) != 1 {
			continue
		}
		dir, _ := e.newDir()
		writeMutated(filepath.Join(dir, "pack.json"), base, []mutation{
			set(text, "entries", "battle.attack.hit", "text"), set(vars, "entries", "battle.attack.hit", "variables")}, "")
		catalog, err := localization.Load(filepath.Join(dir, "pack.json"))
		if err != nil {
			panic(err)
		}
		for _, kind := range []byte{'i', 's', 'f'} {
			arg := e.randomArg(kind)
			got, err := catalog.Format("battle.attack.hit", goArgs([]fmtArg{arg})...)
			e.add("format_spec", obj{"text": text, "verb": spec[len(spec)-1:], "args": []fmtArg{arg}},
				obj{"text": got, "error": e.errv(err)})
		}
	}
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	assets := flag.String("assets", "", "remake/assets 目錄（含未入庫的 music render）")
	flag.Parse()
	if *out == "" || *assets == "" {
		fmt.Fprintln(os.Stderr, "需要 -out、-assets")
		os.Exit(2)
	}
	tmp, err := os.MkdirTemp("", "fd2-golden-present-loc-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x50524c43)), tmp: tmp, assets: *assets}
	e.recordTree("locales")
	e.record("music_catalog.json", filepath.Join(*assets, "music_catalog.json"))
	e.recordTree("music_fm")
	e.recordTree("music_mt32")
	e.genOfficialPacks()
	e.genOfficialContent()
	e.genOfficialEntities()
	e.genPackMutations()
	e.genPlaceholders()
	e.genContentMutations()
	e.genEntityMutations()
	e.genMusic()
	e.genFormatSpecs()
	raw, err := json.MarshalIndent(obj{
		"schema": "fd2-golden-present-loc-v1", "seed": *seed, "files": e.files, "cases": e.cases,
	}, "", " ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, append(raw, '\n'), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("%d 案例\n", len(e.cases))
}
