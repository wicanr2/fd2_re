// fd2-golden-campaign-a 輸出 internal/campaign「A 組」（劇本資料與處理器編譯）與 internal/editorcanonical
// 的黃金差分資料，供 Rust 改寫 remake-rs/crates/fd2-campaign 逐欄比對
// （docs/goal/119-goal-rust-bevy-rewrite-20261113.md 的 R3 完成條件「讀入全部 remake/assets 劇本」）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫公開 API，不改任何遊戲行為。輸入全部是真實檔案，不抽樣：
//   - assets/scenarios/*.json：campaign.Load 全部；載入成功的戰役圖再對每個節點跑
//     ExpandNativeDialogueGroups、Runner（每個節點 × 固定結果清單）與城鎮暗門判定。
//   - assets/cutscenes/handlers/**.json：LoadHandlerScript 全部，成功者再以空 HandlerBindings 編譯。
//   - assets/cutscenes/bindings/**.json：LoadHandlerBinding 與 CompileHandlerBinding 全部，編譯結果再跑
//     ExpandNativeDialogueGroups 與每個原生對白版面的逐字時鐘／可見字形。
//   - assets/cutscenes/acting/**.json：LoadActingResourceSet 全部。
//   - assets/cutscenes/dialogue-index/*.json：LoadStoryIndexMap 全部，並對每個映射逐一 Lookup。
//   - assets/editor-canonical：ValidateBundle、CompileCampaignJSON（輸出原文與 SHA-256）與 Decode。
//   - 合成的 canonical 戰役文件（原文存進案例）：驗 CompileCampaignJSON 的 HTML 跳脫、壓縮與嚴格解碼。
//
// 讀到的每個檔案依相對路徑排序，連同大小與 SHA-256 存進 inputs。
//
// 結構以 json.Marshal 輸出（Go 標籤名；Rust 端以同名 serde 讀寫）。錯誤欄位是 Go err.Error() 原文；
// Go 標準函式庫產生的錯誤（讀檔、JSON 解碼）只比接受或拒收，見 Rust 端。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/editorcanonical"
)

type obj = map[string]any

type caseJSON struct {
	Index  int    `json:"index"`
	Op     string `json:"op"`
	Input  obj    `json:"input"`
	Result obj    `json:"result"`
}

type inputFile struct {
	Path   string `json:"path"`
	Size   int    `json:"size"`
	SHA256 string `json:"sha256"`
}

type exporter struct {
	root   string
	cases  []caseJSON
	inputs map[string]inputFile
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func errv(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}

func (e *exporter) add(op string, input, result obj) {
	e.cases = append(e.cases, caseJSON{Index: len(e.cases), Op: op, Input: input, Result: result})
}

// record 記下一個讀到的真實輸入（相對於 assets 根目錄）。
func (e *exporter) record(path string) {
	rel, err := filepath.Rel(e.root, path)
	if err != nil {
		panic(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return
	}
	e.inputs[filepath.ToSlash(rel)] = inputFile{Path: filepath.ToSlash(rel), Size: len(raw), SHA256: digest(raw)}
}

// files 列出 dir 底下（含子目錄）全部 .json，依路徑排序。
func files(dir string) []string {
	var out []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".json") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	sort.Strings(out)
	return out
}

func issuesJSON(issues []campaign.HandlerCompileIssue) []obj {
	out := make([]obj, 0, len(issues))
	for _, issue := range issues {
		out = append(out, obj{"Beat": issue.Beat, "Op": issue.Op, "Source": issue.Source, "Reason": issue.Reason})
	}
	return out
}

// layoutsOf 依深度優先順序收集 beats 內（含 then／else）全部原生對白版面。
func layoutsOf(beats []campaign.Beat, out *[]*campaign.NativeDialogueLayout) {
	for i := range beats {
		if beats[i].NativeDialogue != nil {
			*out = append(*out, beats[i].NativeDialogue)
		}
		for _, layout := range beats[i].NativeDialogues {
			*out = append(*out, layout)
		}
		layoutsOf(beats[i].Then, out)
		layoutsOf(beats[i].Else, out)
	}
}

// glyphSummary 對一個版面的每頁記逐字時鐘、全部可見字形的雜湊，以及超出時鐘一格的錯誤。
func glyphSummary(layout *campaign.NativeDialogueLayout) obj {
	x, y, width, step, rows, gerr := campaign.NativeStoryDialogueTextGeometry(layout.Control)
	pages := []obj{}
	for page := 0; page <= len(layout.Pages); page++ {
		steps, err := campaign.NativeStoryDialogueGlyphSteps(layout, page)
		entry := obj{"page": page, "error": errv(err)}
		if err == nil {
			var clock strings.Builder
			for _, s := range steps {
				if s {
					clock.WriteByte('1')
				} else {
					clock.WriteByte('0')
				}
			}
			entry["steps"] = clock.String()
			for _, progress := range []int{0, (len(steps) - 1) / 2, len(steps) - 1, len(steps)} {
				glyphs, err := campaign.NativeStoryDialogueVisibleGlyphs(layout, page, progress)
				var text strings.Builder
				for _, g := range glyphs {
					fmt.Fprintf(&text, "%s|%d|%d|%d\n", g.Token, g.X, g.Y, g.GlyphID)
				}
				entry[fmt.Sprintf("visible_%d", progress)] = obj{"error": errv(err), "count": len(glyphs), "sha256": digest([]byte(text.String()))}
			}
		}
		pages = append(pages, entry)
	}
	return obj{"geometry": []int{x, y, width, step, rows}, "geometry_error": errv(gerr), "validate": errv(layout.Validate()), "pages": pages}
}

func expandJSON(beats []campaign.Beat) obj {
	expanded, err := campaign.ExpandNativeDialogueGroups(beats)
	result := obj{"error": errv(err)}
	if err == nil {
		result["beats"] = expanded
		var layouts []*campaign.NativeDialogueLayout
		layoutsOf(expanded, &layouts)
		summaries := make([]obj, 0, len(layouts))
		for _, layout := range layouts {
			summaries = append(summaries, glyphSummary(layout))
		}
		result["glyphs"] = summaries
	}
	return result
}

var outcomes = []string{"", "win", "lose", "next", "opt0", "opt1", "opt2", "opt3", "opt-1", "opt 1", "opt01x", "optx",
	"present", "missing", "crafted", "insufficient", "cancel", "rumor"}

// runnerJSON 對每個節點以全新 Runner 逐一套用固定結果清單，並記可見選項、商品與城鎮暗門判定。
func runnerJSON(c *campaign.Campaign) obj {
	ids := make([]string, 0, len(c.Nodes))
	for id := range c.Nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	nodes := obj{}
	for _, id := range ids {
		node := c.Nodes[id]
		advance := obj{}
		for _, outcome := range outcomes {
			r := campaign.NewRunner(c)
			r.Cur = id
			next := r.Advance(outcome)
			advance[outcome] = obj{"next": next, "cur": r.Cur, "flags": r.Flags}
		}
		r := campaign.NewRunner(c)
		r.Cur = id
		entry := obj{"advance": advance, "shop_goods": r.ShopGoods(), "node_id": r.NodeID()}
		if node.Type == "choice" || node.Type == "town" {
			entry["visible"] = r.Visible()
		}
		scans := []int{0x3b, 0x54, 0x71}
		if node.NativeSecretGate != nil {
			scans = append(scans, node.NativeSecretGate.ScanCode)
		}
		match := []obj{}
		for selection := -1; selection <= 5; selection++ {
			for _, scan := range scans {
				if r.MatchNativeTownSecret(selection, scan) {
					match = append(match, obj{"selection": selection, "scan": scan})
				}
			}
		}
		entry["secret_match"] = match
		confirm := obj{}
		for _, selection := range []int{4, 5} {
			r := campaign.NewRunner(c)
			r.Cur = id
			ok := r.ConfirmNativeTownSecret(selection)
			confirm[fmt.Sprint(selection)] = obj{"ok": ok, "cur": r.Cur}
		}
		entry["secret_confirm"] = confirm
		nodes[id] = entry
	}
	r := campaign.NewRunner(c)
	return obj{"start": r.Cur, "flags": r.Flags, "nodes": nodes}
}

func (e *exporter) genScenarios() {
	for _, path := range files(filepath.Join(e.root, "scenarios")) {
		e.record(path)
		c, err := campaign.Load(path)
		result := obj{"error": errv(err)}
		if err == nil {
			result["campaign"] = c
			ids := make([]string, 0, len(c.Nodes))
			for id := range c.Nodes {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			expanded := obj{}
			for _, id := range ids {
				if len(c.Nodes[id].Beats) > 0 {
					expanded[id] = expandJSON(c.Nodes[id].Beats)
				}
			}
			result["expanded"] = expanded
			result["runner"] = runnerJSON(c)
		}
		e.add("campaign_load", obj{"path": path}, result)
	}
}

func (e *exporter) genHandlers() {
	for _, path := range files(filepath.Join(e.root, "cutscenes", "handlers")) {
		e.record(path)
		script, err := campaign.LoadHandlerScript(path)
		result := obj{"error": errv(err)}
		if err == nil {
			result["script"] = script
			beats, issues := campaign.CompileHandlerScript(script, campaign.HandlerBindings{})
			result["beats"] = beats
			result["issues"] = issuesJSON(issues)
		}
		e.add("handler_script", obj{"path": path}, result)
	}
}

func (e *exporter) genBindings() {
	for _, path := range files(filepath.Join(e.root, "cutscenes", "bindings")) {
		e.record(path)
		binding, err := campaign.LoadHandlerBinding(path)
		result := obj{"load_error": errv(err)}
		if err == nil {
			result["binding"] = binding
			if binding.HandlerScript != "" {
				e.record(filepath.Join(filepath.Dir(path), binding.HandlerScript))
			}
		}
		beats, issues, err := campaign.CompileHandlerBinding(path)
		result["error"] = errv(err)
		if err == nil {
			result["beats"] = beats
			result["issues"] = issuesJSON(issues)
			result["expanded"] = expandJSON(beats)
		}
		e.add("handler_binding", obj{"path": path}, result)
	}
}

func (e *exporter) genActing() {
	for _, path := range files(filepath.Join(e.root, "cutscenes", "acting")) {
		e.record(path)
		resources, err := campaign.LoadActingResourceSet(path)
		result := obj{"error": errv(err)}
		if err == nil {
			result["resources"] = resources
		}
		e.add("acting_set", obj{"path": path}, result)
	}
}

func (e *exporter) genStoryIndex() {
	for _, path := range files(filepath.Join(e.root, "cutscenes", "dialogue-index")) {
		e.record(path)
		index, err := campaign.LoadStoryIndexMap(path)
		result := obj{"error": errv(err)}
		if err == nil {
			result["index"] = index
			lookups := []obj{}
			for _, resource := range index.Resources {
				for _, mapping := range resource.ScriptMappings {
					for i := -1; i <= len(mapping.Mappings); i++ {
						targets, ok := index.Lookup(resource.SourceDAT, mapping.Script, i)
						lookups = append(lookups, obj{"source_dat": resource.SourceDAT, "script": mapping.Script, "index": i, "ok": ok, "targets": targets})
					}
					targets, ok := index.Lookup(resource.SourceDAT, mapping.Script+".missing", 0)
					lookups = append(lookups, obj{"source_dat": resource.SourceDAT, "script": mapping.Script + ".missing", "index": 0, "ok": ok, "targets": targets})
				}
			}
			result["lookups"] = lookups
		}
		e.add("story_index", obj{"path": path}, result)
	}
}

func (e *exporter) genEditorCanonical() {
	root := filepath.Join(e.root, "editor-canonical")
	for _, path := range files(root) {
		e.record(path)
	}
	summary, err := editorcanonical.ValidateBundle(root)
	result := obj{"validate_error": errv(err)}
	if err == nil {
		result["summary"] = summary
	}
	compiled, err := editorcanonical.CompileCampaignJSON(root)
	result["compile_error"] = errv(err)
	if err == nil {
		result["compiled"] = string(compiled)
		result["compiled_sha256"] = digest(compiled)
		c, err := campaign.Decode(compiled)
		result["decode_error"] = errv(err)
		if err == nil {
			result["campaign"] = c
		}
	}
	e.add("editor_canonical", obj{"root": root}, result)
}

// syntheticCanonical 是只用來驗 CompileCampaignJSON 位元組輸出與嚴格解碼的合成文件：legacy 值含字串內
// 空白、`<`／`>`／`&`、U+2028／U+2029、未排序的鍵、各種數字字面值；轉場含可解析、查不到、非字串與
// null 四種（非字串另成一個變體）。真實 bundle 沒有這些字元，HTML 跳脫只有這裡驗得到。
const syntheticCanonical = `{ "schema_version": 1, "document_id": "campaign/synthetic", "kind": "campaign",
  "source": {"path": "assets/scenarios/synthetic.json", "importer_version": "v"},
  "extensions": {"legacy": {"title": "A <b> & c` + " " + `d` + " " + `", "start": "k1",
    "flags": { "z": true , "a": false }, "nodes": {"stale": 1}, "esc": "<\t\"q\"\\"}},
  "nodes": [
    {"node_id": "n1", "type": "story<1>&", "next": "n2", "on_win": "n9", "on_lose": null,
     "asset_ids": [ "x" , "y" ], "map_id": "", "scenario_id": "", "story_id": "",
     "extensions": {"legacy_key": "k1", "legacy": {"text": "line\n <&> ", "beats": [ {"op": "pan", "x": 1, "w": 1.50, "e": 1e3, "z": -0} ],
       "type": "ignored", "b": {"y": 2, "a": [1, 2 , {}]}}}},
    {"node_id": "n2", "type": "ending", "next": "n1", "on_win": "n1",
     "extensions": {"legacy_key": "k&2", "legacy": {}}},
    {"node_id": "n3", "type": "battle", "on_lose": "n3", "asset_ids": [],
     "extensions": {"legacy_key": "k3", "legacy": null}}
  ]
}`

func (e *exporter) genEditorCanonicalSynthetic() {
	tmp, err := os.MkdirTemp("", "fd2-golden-campaign-a-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	variants := []struct{ name, text string }{
		{"valid", syntheticCanonical},
		{"non_string_target", strings.Replace(syntheticCanonical, `"on_win": "n9"`, `"on_win": 5`, 1)},
		{"trailing_value", syntheticCanonical + "\n{}"},
		{"trailing_space", syntheticCanonical + " \n\t "},
		{"trailing_garbage", syntheticCanonical + " x"},
		{"unknown_top", strings.Replace(syntheticCanonical, `"kind": "campaign",`, `"kind": "campaign", "extra": 1,`, 1)},
		{"unknown_node", strings.Replace(syntheticCanonical, `"node_id": "n2",`, `"node_id": "n2", "extra": 1,`, 1)},
		{"unknown_source", strings.Replace(syntheticCanonical, `"importer_version": "v"`, `"importer_version": "v", "x": 0`, 1)},
		{"duplicate_legacy_key", strings.Replace(syntheticCanonical, `"legacy_key": "k3"`, `"legacy_key": "k1"`, 1)},
		{"empty_node_id", strings.Replace(syntheticCanonical, `"node_id": "n3",`, `"node_id": "",`, 1)},
		{"missing_legacy_key", strings.Replace(syntheticCanonical, `"legacy_key": "k3", `, ``, 1)},
		{"wrong_kind", strings.Replace(syntheticCanonical, `"kind": "campaign",`, `"kind": "scenario",`, 1)},
		{"no_nodes", `{"schema_version": 1, "kind": "campaign", "nodes": []}`},
		{"empty_file", ``},
	}
	for i, variant := range variants {
		root := filepath.Join(tmp, fmt.Sprint(i))
		if err := os.MkdirAll(filepath.Join(root, "campaign"), 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(filepath.Join(root, "campaign", "campaign_full.json"), []byte(variant.text), 0o644); err != nil {
			panic(err)
		}
		compiled, err := editorcanonical.CompileCampaignJSON(root)
		result := obj{"compile_error": errv(err)}
		if err == nil {
			result["compiled"] = string(compiled)
			result["compiled_sha256"] = digest(compiled)
			_, err := campaign.Decode(compiled)
			result["decode_error"] = errv(err)
		}
		e.add("editor_canonical_synthetic", obj{"name": variant.name, "document": variant.text}, result)
	}
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	assets := flag.String("assets", "", "remake/assets 根目錄")
	flag.Parse()
	if *out == "" || *assets == "" {
		fmt.Fprintln(os.Stderr, "需要 -out 與 -assets")
		os.Exit(2)
	}
	e := &exporter{root: *assets, inputs: map[string]inputFile{}}
	e.genScenarios()
	e.genHandlers()
	e.genBindings()
	e.genActing()
	e.genStoryIndex()
	e.genEditorCanonical()
	e.genEditorCanonicalSynthetic()
	names := make([]string, 0, len(e.inputs))
	for name := range e.inputs {
		names = append(names, name)
	}
	sort.Strings(names)
	inputs := make([]inputFile, 0, len(names))
	for _, name := range names {
		inputs = append(inputs, e.inputs[name])
	}
	doc := obj{"schema": "fd2-golden-campaign-a/1", "assets": *assets, "inputs": inputs, "cases": e.cases}
	data, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		panic(err)
	}
	ops := map[string]int{}
	for _, c := range e.cases {
		ops[c.Op]++
	}
	keys := make([]string, 0, len(ops))
	for name := range ops {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		fmt.Printf("%s=%d ", name, ops[name])
	}
	fmt.Printf("inputs=%d total=%d\n", len(inputs), len(e.cases))
}
