package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"

	"github.com/wicanr2/fd2_re/remake/internal/figani"
)

// ---- 衍生 strike 與指令 24 ----

func (e *exporter) strikeCases(figaniPath string) {
	r := e.rng
	for i := 0; i < 200; i++ {
		count := r.IntN(17)
		frames := make([][]int, count)
		a := &figani.Animation{Frames: make([]figani.Frame, count)}
		for j := range frames {
			delay := []int{-1, 0, 1, 2, 2, 6, 255, 256}[r.IntN(8)]
			raw4 := 0
			if r.IntN(5) == 0 {
				raw4 = 1
			}
			raw5 := r.IntN(5)
			frames[j] = ints(delay, raw4, raw5)
			a.Frames[j] = figani.Frame{Delay: delay, RawByte4: byte(raw4), RawByte5: byte(raw5)}
		}
		a.HeaderByte1, a.HeaderByte2, a.HeaderByte4 = byte(r.IntN(3)), byte(r.IntN(count+2)), byte(r.IntN(8))
		result := e.builders(a)
		e.add("strike_random", "strike", obj{"frames": frames, "h1": int(a.HeaderByte1), "h2": int(a.HeaderByte2), "h4": int(a.HeaderByte4)},
			obj{"strike": result["strike"], "c24": result["c24"]})
	}
	base, err := figani.DecodeResource(figaniPath, 98)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	run := func(input obj, mutate func(*figani.Animation)) {
		a := *base
		a.Frames = append([]figani.Frame(nil), base.Frames...)
		mutate(&a)
		c24, err := figani.BuildNativeCommand24Schedule(&a)
		var result any = c24JSON(c24)
		if err != nil {
			result = e.errObj(err)
		}
		e.add("strike_c24_mutated", "c24_mutated", input, result)
	}
	run(obj{"frame": -1, "field": "none"}, func(*figani.Animation) {})
	for frame := range base.Frames {
		for _, field := range []string{"raw4", "raw5", "delay", "raw7", "delay0", "delay256"} {
			run(obj{"frame": frame, "field": field}, func(a *figani.Animation) {
				f := &a.Frames[frame]
				switch field {
				case "raw4":
					f.RawByte4++
				case "raw5":
					f.RawByte5++
				case "delay":
					f.Delay++
				case "raw7":
					f.RawByte7++
				case "delay0":
					f.Delay = 0
				case "delay256":
					f.Delay = 256
				}
			})
		}
	}
	for _, field := range []string{"h1", "h2", "h4", "drop"} {
		run(obj{"frame": -1, "field": field}, func(a *figani.Animation) {
			switch field {
			case "h1":
				a.HeaderByte1++
			case "h2":
				a.HeaderByte2++
			case "h4":
				a.HeaderByte4++
			case "drop":
				a.Frames = a.Frames[:len(a.Frames)-1]
			}
		})
	}
}

// ---- 指令契約表 ----

func (e *exporter) tableResult(path string) any {
	table, err := figani.LoadNativeCommandIndexedEntryTable(path)
	if err != nil {
		return e.errObj(err)
	}
	schedules := []any{}
	for id := -1; id <= 10; id++ {
		entry, err := table.Schedule(id)
		if err != nil {
			schedules = append(schedules, e.errObj(err))
			continue
		}
		markers := [][]any{}
		for _, m := range entry.SampleMarkers {
			var fbn any
			if m.FrameBaseNonzero != nil {
				fbn = *m.FrameBaseNonzero
			}
			markers = append(markers, []any{m.Mode, m.Channel, m.Counter, m.CounterPhase, fbn, m.Callee})
		}
		modes := map[string]int{}
		for k, v := range entry.ModeReturns {
			modes[k] = v
		}
		strs := func(v []string) []string {
			if v == nil {
				return []string{}
			}
			return v
		}
		schedules = append(schedules, obj{"id": entry.CommandID, "entry": entry.Entry, "modes": modes,
			"init": entry.InitChannels, "draw": entry.DrawChannels, "draw_modes": orEmpty(entry.DrawModes),
			"shift": entry.RawSideZeroXShift, "offsets": strs(entry.OffsetTables), "ranges": strs(entry.StateRanges),
			"rng": entry.UsesRNG, "markers": markers})
	}
	s := table.Source
	return obj{"schema": table.SchemaVersion, "source": []any{s.File, s.Size, s.MD5, s.SHA256, s.IDAVersion, s.AddressSpace, s.EntryTable, s.Consumer},
		"entries": len(table.Entries), "schedules": schedules}
}

// tableCase 把 JSON 文字寫進工作目錄後載入；input 以文字保存（都是合法 UTF-8）。
func (e *exporter) tableCase(kind string, text []byte) {
	path := filepath.Join(e.work, "entries.json")
	if err := os.MkdirAll(e.work, 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, text, 0o644); err != nil {
		panic(err)
	}
	e.add(kind, "table", obj{"text": string(text)}, e.tableResult(path))
}

// leaves 列出 JSON 樹中每個節點的路徑。
func leaves(v any, path []any, out *[][]any) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			leaves(t[k], append(append([]any(nil), path...), k), out)
		}
	case []any:
		for i := range t {
			leaves(t[i], append(append([]any(nil), path...), i), out)
		}
	}
	*out = append(*out, path)
}

func cloneJSON(v any) any {
	raw, _ := json.Marshal(v)
	var out any
	_ = json.Unmarshal(raw, &out)
	return out
}

// edit 在 path 指向的節點上套用 change（回傳新值；回傳 deleteMark 表示刪掉這個鍵／元素）。
var deleteMark = &struct{}{}

func edit(root any, path []any, change func(any) any) any {
	if len(path) == 0 {
		return change(root)
	}
	switch t := root.(type) {
	case map[string]any:
		k := path[0].(string)
		next := edit(t[k], path[1:], change)
		if next == deleteMark {
			delete(t, k)
		} else {
			t[k] = next
		}
	case []any:
		i := path[0].(int)
		next := edit(t[i], path[1:], change)
		if next == deleteMark {
			return append(t[:i:i], t[i+1:]...)
		}
		t[i] = next
	}
	return root
}

func (e *exporter) indexedTableCases(entriesPath string) {
	original, err := os.ReadFile(entriesPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	e.tableCase("table_original", original)
	var tree any
	if err := json.Unmarshal(original, &tree); err != nil {
		panic(err)
	}
	var paths [][]any
	leaves(tree, nil, &paths)
	// 每個節點都做一次同型別的變動（合法 JSON），另外刪除一次；契約錯誤要逐字相同。
	for _, path := range paths {
		if len(path) == 0 {
			continue
		}
		for _, change := range []func(any) any{
			func(v any) any {
				switch t := v.(type) {
				case float64:
					return t + 1
				case string:
					return t + "x"
				case bool:
					return !t
				case []any:
					if len(t) == 0 {
						return []any{nil}
					}
					return t[:len(t)-1]
				case map[string]any:
					t["zz_extra"] = 1
					return t
				}
				return 0
			},
			func(any) any { return deleteMark },
		} {
			mutated := edit(cloneJSON(tree), path, change)
			raw, _ := json.Marshal(mutated)
			e.tableCase("table_mutated", raw)
		}
	}
	// 型別不符與 null（合法 JSON；型別錯誤屬畸形，只比接受／拒收）。
	typed := []struct {
		path  []any
		value any
	}{
		{[]any{"schema_version"}, "1"}, {[]any{"schema_version"}, 1.5}, {[]any{"schema_version"}, nil},
		{[]any{"source"}, "x"}, {[]any{"source", "size"}, "357074"}, {[]any{"source", "file"}, 7},
		{[]any{"entries"}, map[string]any{}}, {[]any{"entries"}, nil}, {[]any{"entries", 0}, nil},
		{[]any{"entries", 0, "mode_returns"}, []any{}}, {[]any{"entries", 0, "mode_returns", "0"}, nil},
		{[]any{"entries", 0, "mode_returns", "0"}, "0"}, {[]any{"entries", 0, "draw_modes"}, nil},
		{[]any{"entries", 0, "draw_modes", 0}, "4"}, {[]any{"entries", 2, "uses_rng"}, 0},
		{[]any{"entries", 2, "sample_markers", 0, "frame_base_nonzero"}, nil},
		{[]any{"entries", 2, "sample_markers", 0, "frame_base_nonzero"}, 1},
		{[]any{"entries", 1, "offset_tables"}, nil}, {[]any{"entries", 1, "state_ranges", 0}, nil},
	}
	for _, t := range typed {
		mutated := edit(cloneJSON(tree), t.path, func(any) any { return t.value })
		raw, _ := json.Marshal(mutated)
		e.tableCase("table_typed", raw)
	}
	for _, text := range []string{"null", "{}", "[]", "", "{", `{"schema_version":1}`, "{} x"} {
		e.tableCase("table_malformed", []byte(text))
	}
	for i := 0; i < 20; i++ {
		e.tableCase("table_malformed", original[:e.rng.IntN(len(original))])
	}
	// 重複 entry。
	dup := cloneJSON(tree).(map[string]any)
	list := dup["entries"].([]any)
	list[8] = list[0]
	raw, _ := json.Marshal(dup)
	e.tableCase("table_mutated", raw)
	// 檔案不存在。
	e.add("table_missing", "table_path", obj{"path": "$ROOT/missing.json"}, e.tableResult(filepath.Join(e.work, "missing.json")))
}

// ---- 合成分離素材樹 ----

type tree map[string][]byte

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func palette(n int, alpha bool) color.Palette {
	p := make(color.Palette, n)
	for i := range p {
		a := uint8(255)
		if alpha && i == 0 {
			a = 0
		}
		p[i] = color.NRGBA{uint8(i * 7), uint8(i * 3), uint8(255 - i), a}
	}
	return p
}

type genFrame struct {
	w, h    int
	pixels  []byte
	mask    []byte
	palSize int
	x, y    int
	delay   int
	raw     [3]int
}

func (e *exporter) genFrame() genFrame {
	r := e.rng
	f := genFrame{w: 1 + r.IntN(6), h: 1 + r.IntN(5), palSize: []int{2, 4, 16, 256}[r.IntN(4)],
		x: r.IntN(300) - 20, y: r.IntN(200) - 20, delay: r.IntN(9), raw: [3]int{r.IntN(3), r.IntN(5), r.IntN(2)}}
	f.pixels = make([]byte, f.w*f.h)
	f.mask = make([]byte, f.w*f.h)
	for i := range f.pixels {
		f.pixels[i] = byte(r.IntN(f.palSize))
		f.mask[i] = []byte{0, 1, 255, 128}[r.IntN(4)]
	}
	return f
}

func (f genFrame) pixelPNG() []byte {
	img := image.NewPaletted(image.Rect(0, 0, f.w, f.h), palette(f.palSize, f.palSize == 16))
	copy(img.Pix, f.pixels)
	return encodePNG(img)
}

func (f genFrame) maskPNG() []byte {
	img := image.NewGray(image.Rect(0, 0, f.w, f.h))
	copy(img.Pix, f.mask)
	return encodePNG(img)
}

// animationTree 產生一個合法的 FIGANI_NNN／FDOTHER_NNN 目錄，回傳可再修改的後設資料。
func (e *exporter) animationTree(t tree, prefix string, resource, frames int) (map[string]any, []genFrame) {
	dir := fmt.Sprintf("%s_%03d", prefix, resource)
	file := map[string]string{"FIGANI": "FIGANI.DAT", "FDOTHER": "FDOTHER.DAT"}[prefix]
	id := map[string]string{"FIGANI": "figani", "FDOTHER": "fdother"}[prefix]
	list := []any{}
	gens := []genFrame{}
	for i := 0; i < frames; i++ {
		f := e.genFrame()
		gens = append(gens, f)
		pix, mask := fmt.Sprintf("%s_f%02d.png", dir, i), fmt.Sprintf("%s_f%02d_mask.png", dir, i)
		t[dir+"/"+pix] = f.pixelPNG()
		t[dir+"/"+mask] = f.maskPNG()
		list = append(list, map[string]any{"frame_id": fmt.Sprintf("frame/%03d", i), "path": pix, "mask_path": mask,
			"delay_native": f.delay, "x": f.x, "y": f.y, "width": f.w, "height": f.h,
			"raw_byte_4": f.raw[0], "raw_byte_5": f.raw[1], "raw_byte_7": f.raw[2], "extensions": map[string]any{}})
	}
	meta := map[string]any{"schema_version": 1, "kind": "animation", "animation_id": fmt.Sprintf("animation/%s_%03d", id, resource),
		"source": map[string]any{"file": file, "resource": resource}, "document_id": "x",
		"native_header": map[string]any{"byte_1": e.rng.IntN(2), "byte_2": e.rng.IntN(4), "byte_4": e.rng.IntN(7)}, "frames": list}
	return meta, gens
}

func statusDoc(resource int, zero bool) map[string]any {
	status, reason, word := "decoded", "decoded", any(4)
	if zero {
		status, reason, word = "empty_header_zero", "zero_header_word", 0
	}
	return map[string]any{"schema_version": 1, "kind": "animation_resource_status",
		"resource_id": fmt.Sprintf("animation-resource/figani_%03d", resource),
		"source":      map[string]any{"file": "FIGANI.DAT", "resource": resource}, "raw_size": 3,
		"header_word_le": word, "status": status, "reason_code": reason, "evidence": "confirmed", "extensions": map[string]any{}}
}

func putJSON(t tree, path string, v any) {
	raw, _ := json.Marshal(v)
	t[path] = raw
}

type call struct {
	Fn       string `json:"fn"`
	Root     string `json:"root"`
	Source   string `json:"source"`
	Resource int    `json:"resource"`
}

// treeCase 寫出整棵樹，依序執行呼叫並記錄結果；Rust 端在自己的暫存目錄重建同一棵樹。
func (e *exporter) treeCase(kind string, t tree, calls []call) {
	if err := os.RemoveAll(e.work); err != nil {
		panic(err)
	}
	names := make([]string, 0, len(t))
	for name := range t {
		names = append(names, name)
	}
	sort.Strings(names)
	files := obj{}
	for _, name := range names {
		path := filepath.Join(e.work, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			panic(err)
		}
		if t[name] == nil {
			// nil 代表目錄（例如把目錄放在 PNG 路徑上）。
			if err := os.MkdirAll(path, 0o755); err != nil {
				panic(err)
			}
			files[name] = nil
			continue
		}
		if err := os.WriteFile(path, t[name], 0o644); err != nil {
			panic(err)
		}
		files[name] = hex.EncodeToString(t[name])
	}
	if err := os.MkdirAll(e.work, 0o755); err != nil {
		panic(err)
	}
	results := []any{}
	for _, c := range calls {
		root := e.expand(c.Root)
		switch c.Fn {
		case "load":
			a, err := figani.LoadSeparatedArchiveResource(root, c.Source, c.Resource)
			if err != nil {
				results = append(results, e.errObj(err))
			} else {
				results = append(results, obj{"anim": animJSON(a)})
			}
		case "fallback":
			a, source, err := figani.LoadSeparatedResourceWithZeroHeaderFallbackSource(root, c.Resource)
			if err != nil {
				results = append(results, obj{"err": e.errText(err), "source": source})
			} else {
				results = append(results, obj{"anim": animJSON(a), "source": source})
			}
		}
	}
	e.add(kind, "sep_tree", obj{"files": files, "calls": calls}, results)
}

func load(resource int) call {
	return call{Fn: "load", Root: "$ROOT", Source: "FIGANI.DAT", Resource: resource}
}
func fallback(resource int) call {
	return call{Fn: "fallback", Root: "$ROOT", Source: "FIGANI.DAT", Resource: resource}
}

func (e *exporter) separatedTreeCases() {
	r := e.rng
	// 合法樹：多種調色盤深度與遮罩值，FIGANI 與 FDOTHER 兩個家族。
	for i := 0; i < 40; i++ {
		t := tree{}
		resource := r.IntN(1000)
		prefix := []string{"FIGANI", "FDOTHER"}[i%2]
		meta, _ := e.animationTree(t, prefix, resource, 1+r.IntN(3))
		putJSON(t, fmt.Sprintf("%s_%03d/animation.json", prefix, resource), meta)
		e.treeCase("sep_tree_valid", t, []call{{Fn: "load", Root: "$ROOT", Source: prefix + ".DAT", Resource: resource}})
	}
	// 合法 JSON、違反 figani 契約：錯誤文字要逐字相同。
	contract := []struct {
		name   string
		mutate func(meta map[string]any, frames []any, t tree)
	}{
		{"schema", func(m map[string]any, _ []any, _ tree) { m["schema_version"] = 2 }},
		{"kind", func(m map[string]any, _ []any, _ tree) { m["kind"] = "animations" }},
		{"id", func(m map[string]any, _ []any, _ tree) { m["animation_id"] = "animation/figani_005" }},
		{"source_file", func(m map[string]any, _ []any, _ tree) { m["source"].(map[string]any)["file"] = "FDOTHER.DAT" }},
		{"source_resource", func(m map[string]any, _ []any, _ tree) { m["source"].(map[string]any)["resource"] = 5 }},
		{"no_source", func(m map[string]any, _ []any, _ tree) { delete(m, "source") }},
		{"frames_empty", func(m map[string]any, _ []any, _ tree) { m["frames"] = []any{} }},
		{"frames_null", func(m map[string]any, _ []any, _ tree) { m["frames"] = nil }},
		{"frame_null", func(_ map[string]any, f []any, _ tree) { f[0] = nil }},
		{"frame_id", func(_ map[string]any, f []any, _ tree) { f[0].(map[string]any)["frame_id"] = "frame/1" }},
		{"path_slash", func(_ map[string]any, f []any, _ tree) { f[0].(map[string]any)["path"] = "x/a.png" }},
		{"path_empty", func(_ map[string]any, f []any, _ tree) { f[0].(map[string]any)["path"] = "" }},
		{"path_trailing_slash", func(_ map[string]any, f []any, _ tree) { f[0].(map[string]any)["path"] = "a.png/" }},
		{"mask_slash", func(_ map[string]any, f []any, _ tree) { f[0].(map[string]any)["mask_path"] = "../m.png" }},
		{"width_zero", func(_ map[string]any, f []any, _ tree) { f[0].(map[string]any)["width"] = 0 }},
		{"height_negative", func(_ map[string]any, f []any, _ tree) { f[0].(map[string]any)["height"] = -1 }},
		{"delay_negative", func(_ map[string]any, f []any, _ tree) { f[0].(map[string]any)["delay_native"] = -1 }},
		{"delay_256", func(_ map[string]any, f []any, _ tree) { f[0].(map[string]any)["delay_native"] = 256 }},
		{"delay_255", func(_ map[string]any, f []any, _ tree) { f[0].(map[string]any)["delay_native"] = 255 }},
		{"x_null", func(_ map[string]any, f []any, _ tree) { f[0].(map[string]any)["x"] = nil }},
		{"header_255", func(m map[string]any, _ []any, _ tree) {
			m["native_header"] = map[string]any{"byte_1": 255, "byte_2": 255, "byte_4": 255}
		}},
		{"width_mismatch", func(_ map[string]any, f []any, _ tree) {
			fm := f[0].(map[string]any)
			fm["width"] = fm["width"].(int) + 1
		}},
		{"pixels_gray", func(_ map[string]any, f []any, t tree) {
			fm := f[0].(map[string]any)
			t["FIGANI_004/"+fm["path"].(string)] = t["FIGANI_004/"+fm["mask_path"].(string)]
		}},
		{"mask_paletted", func(_ map[string]any, f []any, t tree) {
			fm := f[0].(map[string]any)
			t["FIGANI_004/"+fm["mask_path"].(string)] = t["FIGANI_004/"+fm["path"].(string)]
		}},
		{"mask_gray16", func(_ map[string]any, f []any, t tree) {
			fm := f[0].(map[string]any)
			t["FIGANI_004/"+fm["mask_path"].(string)] = encodePNG(image.NewGray16(image.Rect(0, 0, fm["width"].(int), fm["height"].(int))))
		}},
		{"pixels_rgba", func(_ map[string]any, f []any, t tree) {
			fm := f[0].(map[string]any)
			t["FIGANI_004/"+fm["path"].(string)] = encodePNG(image.NewRGBA(image.Rect(0, 0, fm["width"].(int), fm["height"].(int))))
		}},
		{"mask_nrgba", func(_ map[string]any, f []any, t tree) {
			fm := f[0].(map[string]any)
			img := image.NewNRGBA(image.Rect(0, 0, fm["width"].(int), fm["height"].(int)))
			img.Pix[3] = 7
			t["FIGANI_004/"+fm["mask_path"].(string)] = encodePNG(img)
		}},
	}
	for _, c := range contract {
		t := tree{}
		meta, _ := e.animationTree(t, "FIGANI", 4, 2)
		c.mutate(meta, meta["frames"].([]any), t)
		putJSON(t, "FIGANI_004/animation.json", meta)
		e.treeCase("sep_tree_contract", t, []call{load(4)})
	}
	// 請求驗證與來源名稱。
	{
		t := tree{}
		meta, _ := e.animationTree(t, "FIGANI", 4, 1)
		putJSON(t, "FIGANI_004/animation.json", meta)
		calls := []call{load(-1), load(1000), load(999), {Fn: "load", Root: "", Source: "FIGANI.DAT", Resource: 4},
			{Fn: "load", Root: "$ROOT", Source: "BG.DAT", Resource: 4}, {Fn: "load", Root: "$ROOT/", Source: "FIGANI.DAT", Resource: 4},
			fallback(-1), fallback(1000), {Fn: "fallback", Root: "", Resource: 4}, fallback(4)}
		e.treeCase("sep_tree_request", t, calls)
	}
	// 零標頭回退：狀態文件的每個條件。
	statusMutations := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"confirmed", func(map[string]any) {}},
		{"schema", func(s map[string]any) { s["schema_version"] = 2 }},
		{"kind", func(s map[string]any) { s["kind"] = "animation" }},
		{"id", func(s map[string]any) { s["resource_id"] = "animation-resource/figani_003" }},
		{"file", func(s map[string]any) { s["source"].(map[string]any)["file"] = "FDOTHER.DAT" }},
		{"resource", func(s map[string]any) { s["source"].(map[string]any)["resource"] = 3 }},
		{"raw_size", func(s map[string]any) { s["raw_size"] = 1 }},
		{"status", func(s map[string]any) { s["status"] = "decoded" }},
		{"reason", func(s map[string]any) { s["reason_code"] = "decoded" }},
		{"evidence", func(s map[string]any) { s["evidence"] = "hypothesis" }},
		{"word_null", func(s map[string]any) { s["header_word_le"] = nil }},
		{"word_missing", func(s map[string]any) { delete(s, "header_word_le") }},
		{"word_one", func(s map[string]any) { s["header_word_le"] = 1 }},
	}
	for _, m := range statusMutations {
		for _, previous := range []bool{true, false} {
			t := tree{}
			if previous {
				meta, _ := e.animationTree(t, "FIGANI", 1, 1)
				putJSON(t, "FIGANI_001/animation.json", meta)
			}
			status := statusDoc(2, true)
			m.mutate(status)
			putJSON(t, "FIGANI_002/resource.json", status)
			e.treeCase("sep_tree_fallback", t, []call{fallback(2)})
		}
	}
	{
		// 資源 0 的零標頭不可回退；主資源存在時直接採用。
		t := tree{}
		putJSON(t, "FIGANI_000/resource.json", statusDoc(0, true))
		meta, _ := e.animationTree(t, "FIGANI", 5, 1)
		putJSON(t, "FIGANI_005/animation.json", meta)
		putJSON(t, "FIGANI_005/resource.json", statusDoc(5, true))
		e.treeCase("sep_tree_fallback", t, []call{fallback(0), fallback(5), fallback(6), fallback(7)})
	}
	// 畸形 JSON、PNG 與檔案系統：只比接受／拒收。
	malformed := []struct {
		name   string
		mutate func(t tree, frame map[string]any)
	}{
		{"json_truncated", func(t tree, _ map[string]any) {
			raw := t["FIGANI_004/animation.json"]
			t["FIGANI_004/animation.json"] = raw[:len(raw)/2]
		}},
		{"json_trailing", func(t tree, _ map[string]any) {
			t["FIGANI_004/animation.json"] = append(t["FIGANI_004/animation.json"], 'x')
		}},
		{"json_array", func(t tree, _ map[string]any) { t["FIGANI_004/animation.json"] = []byte("[]") }},
		{"json_empty", func(t tree, _ map[string]any) { t["FIGANI_004/animation.json"] = []byte{} }},
		{"json_null", func(t tree, _ map[string]any) { t["FIGANI_004/animation.json"] = []byte("null") }},
		{"json_missing", func(t tree, _ map[string]any) { delete(t, "FIGANI_004/animation.json") }},
		{"json_directory", func(t tree, _ map[string]any) { t["FIGANI_004/animation.json"] = nil }},
		{"png_missing", func(t tree, f map[string]any) { delete(t, "FIGANI_004/"+f["path"].(string)) }},
		{"png_directory", func(t tree, f map[string]any) { t["FIGANI_004/"+f["mask_path"].(string)] = nil }},
		{"png_not_png", func(t tree, f map[string]any) { t["FIGANI_004/"+f["path"].(string)] = []byte("GIF89a......") }},
		{"png_empty", func(t tree, f map[string]any) { t["FIGANI_004/"+f["mask_path"].(string)] = []byte{} }},
		{"png_truncated", func(t tree, f map[string]any) {
			raw := t["FIGANI_004/"+f["path"].(string)]
			t["FIGANI_004/"+f["path"].(string)] = raw[:len(raw)-20]
		}},
		{"png_header_only", func(t tree, f map[string]any) {
			t["FIGANI_004/"+f["path"].(string)] = t["FIGANI_004/"+f["path"].(string)][:33]
		}},
		{"png_bad_ihdr_crc", func(t tree, f map[string]any) {
			raw := append([]byte(nil), t["FIGANI_004/"+f["path"].(string)]...)
			raw[29] ^= 0xff
			t["FIGANI_004/"+f["path"].(string)] = raw
		}},
	}
	for _, m := range malformed {
		t := tree{}
		meta, _ := e.animationTree(t, "FIGANI", 4, 1)
		putJSON(t, "FIGANI_004/animation.json", meta)
		m.mutate(t, meta["frames"].([]any)[0].(map[string]any))
		e.treeCase("sep_tree_malformed", t, []call{load(4)})
	}
	typed := []struct {
		key   []string
		value any
	}{
		{[]string{"schema_version"}, "1"}, {[]string{"schema_version"}, 1.5}, {[]string{"kind"}, 3},
		{[]string{"source"}, "FIGANI.DAT"}, {[]string{"frames"}, map[string]any{}}, {[]string{"native_header", "byte_1"}, 256},
		{[]string{"native_header", "byte_2"}, -1}, {[]string{"frames", "0", "width"}, 2.5}, {[]string{"frames", "0", "raw_byte_4"}, 300},
		{[]string{"frames", "0", "path"}, 1}, {[]string{"frames", "0"}, "frame"},
	}
	for _, ty := range typed {
		t := tree{}
		meta, _ := e.animationTree(t, "FIGANI", 4, 1)
		var target any = meta
		for i, k := range ty.key {
			last := i == len(ty.key)-1
			switch node := target.(type) {
			case map[string]any:
				if last {
					node[k] = ty.value
				} else {
					target = node[k]
				}
			case []any:
				if last {
					node[0] = ty.value
				} else {
					target = node[0]
				}
			}
		}
		putJSON(t, "FIGANI_004/animation.json", meta)
		e.treeCase("sep_tree_malformed", t, []call{load(4)})
	}
	for _, text := range []string{"{", "[]", "", `{"schema_version":"1"}`, `{"source":5}`, "nul"} {
		t := tree{}
		meta, _ := e.animationTree(t, "FIGANI", 1, 1)
		putJSON(t, "FIGANI_001/animation.json", meta)
		t["FIGANI_002/resource.json"] = []byte(text)
		e.treeCase("sep_tree_malformed", t, []call{fallback(2)})
	}
}
