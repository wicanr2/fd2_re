// fd2-golden-rules-c 輸出 internal/battle「C 組」（物品效果與面板繪製）的黃金差分資料，
// 供 Rust 改寫 remake-rs/crates/fd2-rules 逐欄比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 battle 的公開 API，不改任何遊戲行為。
//   - 真實輸入：remake/assets 的 native_item_effect_rows.json、spells.json、native_command_scene.json、
//     native_command_palette_flash.json；原版 FLAME2 的 FDOTHER.DAT／FDTXT.DAT；分離素材包的
//     ui/fdother_005_item_panel、text/FDTXT_000、fonts 與用到的 portraits。檔案清單排序後連同大小與
//     SHA-256 寫進 JSON，Rust 端讀同一批檔案前先核對。
//   - 產生輸入：以固定種子的 PCG 產生道具效果步驟、單位投影、選擇器、命令格、呈現佇列、物理場景、
//     ch28 呈現與面板繪製的輸入，連同結果寫進同一份 JSON，Rust 端不需重現亂數。
//
// 繪製結果記 320×200 目的緩衝的 SHA-256。錯誤欄位是 Go err.Error() 原文；Go 標準函式庫產生的錯誤
// （讀檔、JSON 解碼、hex 解碼）只比接受或拒收，見 Rust 端。
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
)

type obj = map[string]any

type caseJSON struct {
	Index  int    `json:"index"`
	Op     string `json:"op"`
	Input  obj    `json:"input"`
	Result obj    `json:"result"`
}

const panelBytes = 320 * 200

type exporter struct {
	rng      *rand.Rand
	tmp      string
	flame    string
	pack     string
	cases    []caseJSON
	files    []obj
	itemRows []byte
	book     []battle.NativeCommandRecord
	assets   map[string]battle.NativeItemPanelDataAssets
}

func hexs(b []byte) string { return hex.EncodeToString(b) }

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

// gj 把 Go 值以 encoding/json 轉成通用 JSON（沒有標籤的結構以 Go 欄位名為鍵）。
func gj(v any) any {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		panic(err)
	}
	return out
}

func (e *exporter) add(op string, input, result obj) {
	e.cases = append(e.cases, caseJSON{Index: len(e.cases), Op: op, Input: input, Result: result})
}

func (e *exporter) intn(n int) int { return e.rng.IntN(n) }

func (e *exporter) rangeInt(lo, hi int) int { return lo + e.rng.IntN(hi-lo+1) }

func (e *exporter) chance(pct int) bool { return e.rng.IntN(100) < pct }

func (e *exporter) pick(values ...int) int { return values[e.rng.IntN(len(values))] }

// validOr 有 85% 機會回傳 0..n-1，否則回傳 lo..hi 之間的任意值（含越界值）。
func (e *exporter) validOr(n, lo, hi int) int {
	if e.chance(85) {
		return e.intn(n)
	}
	return e.rangeInt(lo, hi)
}

func (e *exporter) bytesN(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(e.rng.IntN(256))
	}
	return b
}

func (e *exporter) record(label, path string) {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	e.files = append(e.files, obj{"path": label, "size": len(raw), "sha256": digest(raw)})
}

// recordTree 記錄 root/rel 底下全部檔案（路徑排序）。
func (e *exporter) recordTree(prefix, root, rel string) {
	var paths []string
	err := filepath.WalkDir(filepath.Join(root, rel), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil || len(paths) == 0 {
		panic(fmt.Sprintf("找不到 %s/%s", root, rel))
	}
	sort.Strings(paths)
	for _, path := range paths {
		r, err := filepath.Rel(root, path)
		if err != nil {
			panic(err)
		}
		e.record(prefix+"/"+filepath.ToSlash(r), path)
	}
}

// ---- 原版記錄產生 ----

// genRecords 產生 n 筆 0x50 位元組記錄；標記位元組（+0x22..+0x27）與物品欄有機會是乾淨的值，
// 讓各條閘門的兩邊都走得到。
func (e *exporter) genRecords(n int) []byte {
	records := e.bytesN(n * 0x50)
	for u := 0; u < n; u++ {
		r := records[u*0x50 : (u+1)*0x50]
		for off := 0x22; off <= 0x27; off++ {
			if e.chance(60) {
				r[off] = 0
			}
		}
		if e.chance(60) {
			for slot := 0; slot < 8; slot++ {
				if e.chance(70) {
					r[0x0a+slot*2] = byte(e.pick(0, 0x40))
				} else {
					r[0x0a+slot*2] = 0x80
				}
			}
		}
		for _, off := range []int{0x40, 0x42, 0x44, 0x46, 0x48, 0x4a, 0x4c, 0x4e} {
			if e.chance(60) {
				binary.LittleEndian.PutUint16(r[off:], uint16(e.intn(300)))
			}
		}
		if e.chance(50) {
			r[0x20] = byte(e.intn(30))
		}
	}
	return records
}

func (e *exporter) genIndices(n int) []byte {
	out := make([]byte, e.pick(0, 1, 1, 2, 2, 3, 4))
	for i := range out {
		if e.chance(4) {
			out[i] = byte(n + e.intn(3))
		} else {
			out[i] = byte(e.intn(n))
		}
	}
	return out
}

func ints(b []byte) []int {
	out := make([]int, len(b))
	for i, v := range b {
		out[i] = int(v)
	}
	return out
}

// ---- 道具路由 ----

func (e *exporter) genRoutes() {
	for itemType := 0; itemType < 32; itemType++ {
		for _, word := range []int{0, 1, 2, 20, 23, 35, 36, 0x1234, 0xffff} {
			typ, w := byte(itemType), uint16(word)
			res := obj{}
			put := func(name string, route any, ok bool) { res[name] = obj{"ok": ok, "route": gj(route)} }
			r1, ok := battle.NativeItemEffectRouteForType(typ, w)
			put("dispatch", r1, ok)
			r2, ok := battle.NativeItemAPDPStepRouteForType(typ)
			put("apdp", r2, ok)
			r3, ok := battle.NativeItemHITEVStepRouteForType(typ)
			put("hitev", r3, ok)
			r4, ok := battle.NativeItemCapacityStepRouteForType(typ, w)
			put("capacity", r4, ok)
			r5, ok := battle.NativeItemHPRestoreRouteForType(typ, w)
			put("hp", r5, ok)
			r6, ok := battle.NativeItemMPRestoreRouteForType(typ, w)
			put("mp", r6, ok)
			r7, ok := battle.NativeItemMarkerApplicationRouteForType(typ)
			put("marker", r7, ok)
			r8, ok := battle.NativeItemMarkerClearRestoreRouteForType(typ)
			put("marker_clear", r8, ok)
			r9, ok := battle.NativeItemCommandDamageRouteForType(typ, w)
			put("command_damage", r9, ok)
			r10, ok := battle.NativeItemRelocationRouteForType(typ, w)
			put("relocation", r10, ok)
			e.add("routes", obj{"type": itemType, "word": word}, res)
		}
	}
}

// tamper 有 15% 機會改掉路由的一個欄位，走路由驗證失敗與 %#v 錯誤訊息。
func (e *exporter) tamper() int {
	if e.chance(15) {
		return 1 + e.intn(5)
	}
	return 0
}

func (e *exporter) genItemSteps() {
	kinds := []string{"apdp", "hitev", "capacity", "hp", "mp", "marker", "marker_clear", "relocation", "command23"}
	for _, kind := range kinds {
		for k := 0; k < 60; k++ {
			n := 1 + e.intn(4)
			records := e.genRecords(n)
			in := obj{"kind": kind, "records": hexs(records)}
			res := obj{}
			rng := uint16(e.intn(0x10000))
			in["rng"] = int(rng)
			indices := e.genIndices(n)
			in["indices"] = ints(indices)
			sourceUnit, sourceSlot := e.validOr(n, -1, n), e.validOr(8, -1, 8)
			in["source_unit"], in["source_slot"] = sourceUnit, sourceSlot
			itemType := byte(e.intn(26))
			in["item_type"] = int(itemType)
			amount := uint16(e.pick(0, 1, 20, 40, 80, 200, 999, 0xfff0, e.intn(0x10000)))
			in["amount"] = int(amount)
			t := e.tamper()
			in["tamper"] = t
			switch kind {
			case "apdp":
				if e.chance(70) {
					itemType = byte(e.pick(15, 16))
					in["item_type"] = int(itemType)
				}
				route, _ := battle.NativeItemAPDPStepRouteForType(itemType)
				switch t {
				case 1:
					route.MarkerOffset++
				case 2:
					route.WordOffset++
				case 3:
					route.ConsumesSource = true
				case 4:
					route.DerivedStat = "hp"
				case 5:
					route.ItemType++
				}
				results, next, score, err := battle.ApplyNativeItemAPDPStep(records, indices, route, rng)
				res["results"], res["rng"], res["score"], res["err"] = gj(results), int(next), score, errv(err)
			case "hitev":
				if e.chance(70) {
					itemType = 12
					in["item_type"] = int(itemType)
				}
				route, _ := battle.NativeItemHITEVStepRouteForType(itemType)
				switch t {
				case 1, 4:
					route.Increment++
				case 2, 5:
					route.ConsumesSource = true
				case 3:
					route.ItemType++
				}
				results, next, score, err := battle.ApplyNativeItemHITEVStep(records, indices, route, rng)
				res["results"], res["rng"], res["score"], res["err"] = gj(results), int(next), score, errv(err)
			case "capacity":
				if e.chance(70) {
					itemType = byte(e.pick(17, 18, 19))
					in["item_type"] = int(itemType)
				}
				route, _ := battle.NativeItemCapacityStepRouteForType(itemType, amount)
				switch t {
				case 1:
					route.FieldOffset++
				case 2:
					route.Amount++
				case 3:
					route.PresentationCode++
				case 4:
					route.Stat = "mv"
				case 5:
					route.ConsumesSource = false
				}
				target := e.validOr(n, -1, n)
				in["target"] = target
				got, err := battle.ApplyNativeItemCapacityStep(records, target, route, sourceUnit, sourceSlot)
				res["value"], res["err"] = int(got), errv(err)
			case "hp":
				if e.chance(70) {
					itemType = byte(e.pick(5, 13))
					in["item_type"] = int(itemType)
				}
				route, _ := battle.NativeItemHPRestoreRouteForType(itemType, amount)
				switch t {
				case 1, 4:
					route.Amount++
				case 2, 5:
					route.ConsumesSource = !route.ConsumesSource
				case 3:
					route.ItemType = 6
				}
				batch, err := battle.ApplyNativeItemHPRestore(records, indices, route, rng, sourceUnit, sourceSlot)
				res["batch"], res["err"] = gj(batch), errv(err)
			case "mp":
				if e.chance(70) {
					itemType = 11
					in["item_type"] = int(itemType)
				}
				route, _ := battle.NativeItemMPRestoreRouteForType(itemType, amount)
				switch t {
				case 1, 3, 5:
					route.Amount++
				case 2, 4:
					route.ItemType = 12
				}
				batch, err := battle.ApplyNativeItemMPRestore(records, indices, route, rng, sourceUnit, sourceSlot)
				res["batch"], res["err"] = gj(batch), errv(err)
			case "marker":
				if e.chance(70) {
					itemType = byte(e.pick(14, 22))
					in["item_type"] = int(itemType)
				}
				route, _ := battle.NativeItemMarkerApplicationRouteForType(itemType)
				switch t {
				case 1:
					route.PresentationCode++
				case 2:
					route.MarkerOffset++
				case 3:
					route.HPDamageBase++
				case 4:
					route.ConsumesSource = true
				case 5:
					route.ItemType = 15
				}
				results, next, score, err := battle.ApplyNativeItemMarkerApplication(records, indices, route, rng)
				res["results"], res["rng"], res["score"], res["err"] = gj(results), int(next), score, errv(err)
			case "marker_clear":
				if e.chance(70) {
					itemType = byte(e.pick(6, 7))
					in["item_type"] = int(itemType)
				}
				route, _ := battle.NativeItemMarkerClearRestoreRouteForType(itemType)
				switch t {
				case 1:
					route.PresentationCode++
				case 2:
					route.MarkerOffset++
				case 3:
					route.HPRestoreBase++
				case 4:
					route.ConsumesSource = false
				case 5:
					route.ItemType = 8
				}
				results, next, score, err := battle.ApplyNativeItemMarkerClearRestore(records, indices, route, rng, sourceUnit, sourceSlot)
				res["results"], res["rng"], res["score"], res["err"] = gj(results), int(next), score, errv(err)
			case "relocation", "command23":
				book := e.bookVariant(in)
				actor := e.validOr(n, -1, n)
				in["actor"] = actor
				actorBase := actor * 0x50
				if actor >= 0 && actor < n && e.chance(75) {
					records[actorBase+0x08] = byte(e.pick(24, 24, 24, 24, 23, 7))
					binary.LittleEndian.PutUint16(records[actorBase+0x44:], uint16(e.intn(60)))
					binary.LittleEndian.PutUint16(records[actorBase+0x46:], uint16(e.pick(19, 20, 20, 40, 40, e.intn(0x10000))))
					in["records"] = hexs(records)
				}
				dx, dy := byte(e.intn(256)), byte(e.intn(256))
				in["dx"], in["dy"] = int(dx), int(dy)
				var got battle.NativeItemRelocationResult
				var err error
				if kind == "relocation" {
					if e.chance(70) {
						itemType = 23
						in["item_type"] = int(itemType)
					}
					route, _ := battle.NativeItemRelocationRouteForType(itemType, amount)
					switch t {
					case 1:
						route.CommandID++
					case 2:
						route.RequiredIdentity++
					case 3:
						route.RequiredMaxMP++
					case 4:
						route.Presentation++
					case 5:
						route.ConsumesSource = true
					}
					got, err = battle.ApplyNativeItemRelocation(records, actor, indices, dx, dy, route, book)
				} else {
					got, err = battle.ApplyNativeCommand23Relocation(records, actor, indices, dx, dy, book)
				}
				res["result"], res["err"] = gj(got), errv(err)
			}
			res["records"] = hexs(records)
			e.add("item_step", in, res)
		}
	}
}

// bookVariant 回傳真實 spells.json 的命令書，或改過 command23 MP 消耗／長度的變體（寫進 input）。
func (e *exporter) bookVariant(in obj) []battle.NativeCommandRecord {
	book := append([]battle.NativeCommandRecord(nil), e.book...)
	variant := 0
	switch {
	case e.chance(10):
		variant = 1
		book[23].MPCost = e.pick(-1, 0, 5, 255, 256, 300)
		in["mp_cost"] = book[23].MPCost
	case e.chance(4):
		variant = 2
		book = book[:35]
	case e.chance(4):
		variant = 3
		book[23].ID = 22
	}
	in["book_variant"] = variant
	return book
}

// ---- 單位投影 ----

func (e *exporter) genUnit() *battle.Unit {
	u := &battle.Unit{
		HP: e.intn(300), MaxHP: e.pick(0, 100, 32767, -32768, e.rangeInt(-40000, 70000)),
		MP: e.intn(100), MaxMP: e.pick(0, 50, 32760, e.rangeInt(-40000, 70000)),
		MV: e.pick(0, 5, 254, 255, 256, -1), Exp: float64(e.intn(100)),
		EquipmentBaseSet: e.chance(60), BaseMV: e.intn(10),
		BaseAP: e.pick(0, 10, 32766, -32768, e.rangeInt(-40000, 70000)), BaseDP: e.intn(200),
		DX: e.pick(0, 30, 32767, e.rangeInt(-40000, 70000)), BaseHIT: e.intn(100), BaseEV: e.intn(100),
	}
	slots := make([]int, 8)
	flags := make([]int, 8)
	inventory := []int{}
	equipped := []bool{}
	for slot := 0; slot < 8; slot++ {
		if e.chance(55) {
			item := e.intn(0xd7)
			slots[slot] = item
			if e.chance(30) {
				flags[slot] = 0x40
			}
			inventory = append(inventory, item)
			equipped = append(equipped, flags[slot] == 0x40)
		} else {
			slots[slot], flags[slot] = 0xff, 0x80
		}
	}
	switch {
	case e.chance(8):
		slots = slots[:7]
	case e.chance(8):
		flags[e.intn(8)] = 0x80
	case e.chance(8) && len(inventory) > 0:
		inventory = inventory[:len(inventory)-1]
		equipped = equipped[:len(equipped)-1]
	}
	u.InventorySlots, u.NativeInventoryFlags, u.Inventory, u.Equipped = slots, flags, inventory, equipped
	return u
}

func unitsJSON(units []*battle.Unit) []any {
	out := make([]any, len(units))
	for i, u := range units {
		out[i] = gj(u)
	}
	return out
}

func (e *exporter) genUnitItems() {
	for k := 0; k < 160; k++ {
		units := []*battle.Unit{e.genUnit(), e.genUnit()}
		in := obj{"units": unitsJSON(units)}
		target, source := 0, e.pick(0, 0, 1)
		if e.chance(5) {
			target = -1
		}
		if e.chance(5) {
			source = -1
		}
		ptr := func(i int) *battle.Unit {
			if i < 0 {
				return nil
			}
			return units[i]
		}
		slot := e.rangeInt(-1, 8)
		if src := ptr(source); src != nil && e.chance(75) {
			occupied := []int{}
			for i, v := range src.InventorySlots {
				if v != 0xff {
					occupied = append(occupied, i)
				}
			}
			if len(occupied) > 0 {
				slot = occupied[e.intn(len(occupied))]
			}
		}
		in["target"], in["source"], in["slot"] = target, source, slot
		amount := uint16(e.pick(1, 3, 9, 20, 0xffff, e.intn(0x10000)))
		in["amount"] = int(amount)
		t := e.tamper()
		in["tamper"] = t
		res := obj{}
		if k%2 == 0 {
			itemType := byte(e.pick(17, 18, 19, 19, 16))
			in["kind"], in["item_type"] = "capacity", int(itemType)
			route, _ := battle.NativeItemCapacityStepRouteForType(itemType, amount)
			if t != 0 {
				route.FieldOffset++
			}
			got, err := battle.ApplyNativeItemCapacityToUnit(ptr(target), ptr(source), route, slot)
			res["value"], res["err"] = got, errv(err)
		} else {
			itemType := e.pick(8, 9, 10, 10, 11)
			in["kind"], in["item_type"] = "base_stat", itemType
			route, _ := battle.NativeItemWordDeltaRouteForType(itemType)
			if t != 0 {
				route.PresentationCode++
			}
			got, err := battle.ApplyNativeItemBaseStatDeltaToUnit(ptr(target), ptr(source), route, amount, slot)
			res["value"], res["err"] = got, errv(err)
		}
		res["units"] = unitsJSON(units)
		e.add("unit_item", in, res)
	}
}

// ---- 載入器 ----

func (e *exporter) loader(op, text string) obj {
	path := filepath.Join(e.tmp, fmt.Sprintf("%s-%d.json", op, len(e.cases)))
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		panic(err)
	}
	return obj{"text": text}
}

func (e *exporter) genLoaders() {
	table, err := battle.LoadNativeItemEffectRowPrefix("assets/data/native_item_effect_rows.json")
	e.add("item_table_real", obj{"path": "remake/assets/data/native_item_effect_rows.json"},
		obj{"err": errv(err), "len": len(table), "sha256": digest(table)})
	row := strings.Repeat("00", battle.NativeItemEffectRowSize)
	many := make([]string, 0x101)
	for i := range many {
		many[i] = fmt.Sprintf(`{"id":%d,"raw":"%s"}`, i, row)
	}
	for _, text := range []string{
		`[{"id":0,"raw":"` + row + `"},{"id":1,"raw":"` + strings.Repeat("ab", battle.NativeItemEffectRowSize) + `"}]`,
		`[{"id":1,"raw":"` + row + `"}]`,
		`[{"id":0,"raw":"` + row[:44] + `"}]`,
		`[{"id":0,"raw":"` + row[:45] + `"}]`,
		`[{"id":0,"raw":"zz` + row[2:] + `"}]`,
		`[{"id":0,"raw":"` + row + `00"}]`,
		`[{"id":0}]`,
		`[]`,
		`null`,
		`{}`,
		`[` + strings.Join(many, ",") + `]`,
		`[` + strings.Join(many[:0x100], ",") + `]`,
		`[{"id":0,"raw":"` + strings.ToUpper(strings.Repeat("ab", battle.NativeItemEffectRowSize)) + `"}]`,
	} {
		in := e.loader("item_table", text)
		path := filepath.Join(e.tmp, fmt.Sprintf("item_table-%d.json", len(e.cases)))
		table, err := battle.LoadNativeItemEffectRowPrefix(path)
		e.add("item_table", in, obj{"err": errv(err), "table": hexs(table)})
	}

	scene, err := battle.LoadNativeCommandSceneTable("assets/data/native_command_scene.json")
	res := obj{"err": errv(err)}
	if err == nil {
		bg := []any{}
		for chapter := -1; chapter <= 31; chapter++ {
			v, err := scene.InitialBackground(chapter)
			bg = append(bg, []any{int(v), errv(err)})
		}
		res["table"], res["backgrounds"] = gj(scene), bg
	}
	e.add("scene_real", obj{"path": "remake/assets/data/native_command_scene.json"}, res)
	flash, err := battle.LoadNativeCommandPaletteFlashTable("assets/data/native_command_palette_flash.json")
	res = obj{"err": errv(err)}
	if err == nil {
		phases := []any{}
		for id := -1; id <= 37; id++ {
			p, err := flash.NativeCommandPaletteFlashPhases(id)
			phases = append(phases, []any{gj(p), errv(err)})
		}
		res["table"], res["phases"] = gj(flash), phases
	}
	e.add("flash_real", obj{"path": "remake/assets/data/native_command_palette_flash.json"}, res)

	sceneRaw, err := os.ReadFile("assets/data/native_command_scene.json")
	if err != nil {
		panic(err)
	}
	flashRaw, err := os.ReadFile("assets/data/native_command_palette_flash.json")
	if err != nil {
		panic(err)
	}
	mutate := func(raw []byte, edit func(m map[string]any)) string {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			panic(err)
		}
		edit(m)
		data, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}
		return string(data)
	}
	src := func(m map[string]any) map[string]any { return m["source"].(map[string]any) }
	sceneEdits := []func(map[string]any){
		func(m map[string]any) {},
		func(m map[string]any) { m["schema_version"] = 2 },
		func(m map[string]any) { m["chapter_initial_background"] = m["chapter_initial_background"].([]any)[:29] },
		func(m map[string]any) { m["chapter_initial_background"].([]any)[5] = 4 },
		func(m map[string]any) { m["chapter_initial_background"].([]any)[29] = 3 },
		func(m map[string]any) { src(m)["md5"] = "00" },
		func(m map[string]any) { src(m)["raw_gate"] = "sub_1F183" },
		func(m map[string]any) { src(m)["initial_selector_table"] = "" },
		func(m map[string]any) { src(m)["size"] = 1 },
		func(m map[string]any) { m["chapter_initial_background"] = "AwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMAAAADAAA=" },
		func(m map[string]any) { delete(m, "source") },
	}
	for _, edit := range sceneEdits {
		in := e.loader("scene", mutate(sceneRaw, edit))
		path := filepath.Join(e.tmp, fmt.Sprintf("scene-%d.json", len(e.cases)))
		table, err := battle.LoadNativeCommandSceneTable(path)
		res := obj{"err": errv(err)}
		if err == nil {
			res["table"] = gj(table)
		}
		e.add("scene", in, res)
	}
	flashEdits := []func(map[string]any){
		func(m map[string]any) {},
		func(m map[string]any) { m["cycles"] = 3 },
		func(m map[string]any) { m["ticks_per_phase"] = 2 },
		func(m map[string]any) { m["sfx_resource"] = 87 },
		func(m map[string]any) { m["entries"] = m["entries"].([]any)[:35] },
		func(m map[string]any) { m["entries"].([]any)[3].([]any)[1] = 0x40 },
		func(m map[string]any) { m["entries"].([]any)[35].([]any)[2] = 0x3f },
		func(m map[string]any) { src(m)["file"] = "FD2.exe" },
		func(m map[string]any) { src(m)["address_space"] = "file" },
		func(m map[string]any) { src(m)["sha256"] = "" },
	}
	for _, edit := range flashEdits {
		in := e.loader("flash", mutate(flashRaw, edit))
		path := filepath.Join(e.tmp, fmt.Sprintf("flash-%d.json", len(e.cases)))
		table, err := battle.LoadNativeCommandPaletteFlashTable(path)
		res := obj{"err": errv(err)}
		if err == nil {
			res["table"] = gj(table)
		}
		e.add("flash", in, res)
	}
}

// ---- 選擇器、命令格、呈現佇列 ----

func (e *exporter) genSelector() {
	scans := []int{0, 1, 28, 57, 72, 75, 77, 80, 99}
	results := []any{}
	for selection := -1; selection <= 9; selection++ {
		for count := -1; count <= 9; count++ {
			for _, scan := range scans {
				for _, battleUse := range []bool{false, true} {
					for _, effect := range []byte{0, 5} {
						got, r, err := battle.AdvanceNativeItemSelector(selection, count, scan, battleUse, effect)
						results = append(results, []any{got, int(r), errv(err)})
					}
				}
			}
		}
	}
	e.add("selector_advance", obj{"scans": scans}, obj{"results": results})

	for k := 0; k < 120; k++ {
		n := 1 + e.intn(3)
		records := e.genRecords(n)
		for u := 0; u < n; u++ {
			for slot := 0; slot < 8; slot++ {
				if e.chance(85) {
					records[u*0x50+0x0b+slot*2] = byte(e.intn(0xd7))
				}
			}
		}
		rows := e.itemRows
		if e.chance(10) {
			rows = rows[:e.intn(len(rows))]
		}
		unit := e.rangeInt(-1, n)
		selected := e.rangeInt(-2, 8)
		cells, err := battle.NativeItemSelectorCells(records, unit, selected, rows)
		e.add("selector_cells", obj{"records": hexs(records), "unit": unit, "selected": selected, "rows_len": len(rows)},
			obj{"cells": gj(cells), "err": errv(err)})
	}

	frames := []any{}
	for frame := -1; frame <= 12; frame++ {
		f, err := battle.NativeItemPanelFrameFor(frame)
		frames = append(frames, []any{gj(f), errv(err)})
	}
	opening, err1 := battle.NativeItemPanelSchedule(true)
	closing, err2 := battle.NativeItemPanelSchedule(false)
	e.add("panel_layout", obj{}, obj{
		"frames": frames, "opening": gj(opening), "closing": gj(closing), "errs": []any{errv(err1), errv(err2)},
		"base": gj(battle.NativeItemPanelBaseLayoutFor()), "plan": gj(battle.NativeItemPanelDataPlanFor()),
	})

	for k := 0; k < 40; k++ {
		ids := make([]int, e.intn(11))
		for i := range ids {
			ids[i] = e.rangeInt(-1, 40)
		}
		selected := e.rangeInt(-2, len(ids)+1)
		e.add("grid", obj{"ids": ids, "selected": selected}, obj{"cells": gj(battle.NativeCommandGrid(ids, selected))})
	}
	moves := []any{}
	for index := -1; index <= 9; index++ {
		for count := -1; count <= 9; count++ {
			for dir := -1; dir <= 4; dir++ {
				moves = append(moves, battle.NativeCommandGridMove(index, count, dir))
			}
		}
	}
	e.add("grid_move", obj{}, obj{"moves": moves})

	for k := 0; k < 160; k++ {
		queue := make([]battle.NativePresentationDigit, e.intn(3))
		for i := range queue {
			queue[i] = battle.NativePresentationDigit{PositionCode: e.intn(20), Target: e.intn(30), Digit: e.intn(200)}
		}
		value := e.pick(0, 7, 12, 99, 1234, 12345, 999999, -5, -1234, e.rangeInt(-100000, 100000))
		bias := e.pick(0, 48, 5, 0xff, -1, 256, e.intn(256))
		target := e.pick(0, 9, 0xff, -1, 256, e.intn(256))
		inCamera := e.chance(80)
		in := obj{"queue": gj(queue), "value": value, "bias": bias, "target": target, "in_camera": inCamera}
		res := obj{}
		if k%4 == 0 {
			res["queue"] = gj(battle.AppendNativePresentationMiss(queue, target))
			in["miss"] = true
		} else {
			out, err := battle.AppendNativePresentationDigits(queue, value, bias, target, inCamera)
			res["queue"], res["err"] = gj(out), errv(err)
			in["miss"] = false
		}
		e.add("presentation", in, res)
	}
}

// ---- 物理場景與命令背景閘門 ----

func (e *exporter) sceneUnit() *battle.Unit {
	if e.chance(4) {
		return nil
	}
	return &battle.Unit{
		HasNativeRecordByte6: e.chance(97), NativeRecordByte6: byte(e.pick(0, 0, 1, 2, 255)),
		HasBattleFig: e.chance(97), BattleFig: e.pick(1, 4, 27, 28, 29, 102, 255, 256, -1, e.intn(256)),
		HasNativeRecordRace: e.chance(97), NativeRecordRace: byte(e.pick(0, 1, 3, 4, 5, 6, 8)),
		HasNativeRecordClass: e.chance(97), NativeRecordClass: byte(e.pick(0, 2, 3, 18, 19, 25)),
	}
}

func optUnit(u *battle.Unit) any {
	if u == nil {
		return nil
	}
	return gj(u)
}

func (e *exporter) genPhysicalScene() {
	for k := 0; k < 240; k++ {
		actor, target := e.sceneUnit(), e.sceneUnit()
		var ac, tc [4]byte
		copy(ac[:], e.bytesN(4))
		copy(tc[:], e.bytesN(4))
		initial := byte(e.pick(0, 0, 3, 3, 1, 2, 4))
		header := byte(e.pick(0, 0, 1, 2))
		got, err := battle.SelectNativePhysicalScene(initial, actor, target, ac, tc, header)
		gate, gateErr := battle.NativeCommandBackgroundGate(actor)
		e.add("physical_scene", obj{
			"actor": optUnit(actor), "target": optUnit(target), "actor_control": ints(ac[:]),
			"target_control": ints(tc[:]), "initial": int(initial), "header": int(header),
		}, obj{"selection": gj(got), "err": errv(err), "gate": gate, "gate_err": errv(gateErr)})
	}
}

// ---- ch28 戰後 ----

func ch28Summary(st *battle.State) obj {
	units := make([]any, len(st.Units))
	for i, u := range st.Units {
		units[i] = []any{u.HP, u.BattleFig, u.HasBattleFig, int(u.NativeRecordByte8), u.HasNativeRecordByte8,
			int(u.NativeMapPresentation.X), int(u.NativeMapPresentation.Y), int(u.NativeMapPresentation.Pose),
			u.Dir, int(u.NativeRecordByte5)}
	}
	records := make([]any, len(st.NativeRuntimeRecords))
	for i, r := range st.NativeRuntimeRecords {
		records[i] = hexs(r.Raw[:])
	}
	return obj{"units": units, "records": records}
}

type ch28Unit struct {
	Group, HP, X, Y, Pose, Byte5 int
	HasMap, HasByte5             bool
}

func (e *exporter) ch28State(units []ch28Unit, roster []int, projection int) (*battle.State, obj) {
	st := &battle.State{}
	for _, u := range units {
		st.Units = append(st.Units, &battle.Unit{
			Group: u.Group, HP: u.HP,
			NativeMapPresentation:    battle.NativeMapPresentationState{X: byte(u.X), Y: byte(u.Y), Pose: byte(u.Pose)},
			HasNativeMapPresentation: u.HasMap, NativeRecordByte5: byte(u.Byte5), HasNativeRecordByte5: u.HasByte5,
		})
	}
	for _, g := range roster {
		st.Roster = append(st.Roster, &battle.Unit{Group: g})
	}
	// projection：0 無投影；1 完整投影；2 投影少一筆；3 投影有一筆不一致；4 沒有旗標卻有記錄。
	disagree := -1
	if projection != 0 {
		st.HasNativeRuntimeUnitProjection = projection != 4
		for i, u := range st.Units {
			var r battle.NativeRuntimeRecordState
			r.Raw[0], r.Raw[1], r.Raw[3] = u.NativeMapPresentation.X, u.NativeMapPresentation.Y, u.NativeMapPresentation.Pose
			r.Raw[5] = u.NativeRecordByte5
			binary.LittleEndian.PutUint16(r.Raw[0x40:], uint16(u.HP))
			r.Raw[7], r.Raw[8], r.Raw[0x10] = byte(i), byte(e.intn(256)), byte(e.intn(256))
			st.NativeRuntimeRecords = append(st.NativeRuntimeRecords, r)
		}
		if projection == 2 && len(st.NativeRuntimeRecords) > 0 {
			st.NativeRuntimeRecords = st.NativeRuntimeRecords[:len(st.NativeRuntimeRecords)-1]
		}
		if projection == 3 && len(st.NativeRuntimeRecords) > 0 {
			disagree = e.intn(len(st.NativeRuntimeRecords))
			st.NativeRuntimeRecords[disagree].Raw[e.pick(0, 1, 3, 5, 0x40, 0x41)]++
		}
	}
	records := make([]any, len(st.NativeRuntimeRecords))
	for i, r := range st.NativeRuntimeRecords {
		records[i] = hexs(r.Raw[:])
	}
	unitIn := make([]any, len(units))
	for i, u := range units {
		unitIn[i] = []any{u.Group, u.HP, u.X, u.Y, u.Pose, u.Byte5, u.HasMap, u.HasByte5}
	}
	return st, obj{"units": unitIn, "roster": roster, "projection": st.HasNativeRuntimeUnitProjection, "records": records}
}

func (e *exporter) genCh28() {
	mainTail := []int{4, 4, 5, 5, 6, 6, 7, 7, 1, 1, 1}
	captain := []int{2, 2, 2, 3, 3, 3}
	for k := 0; k < 90; k++ {
		// 約一半是原版可產生的前綴；另一半各帶一種損壞，走各條拒收分支。
		corrupt := 0
		if e.chance(50) {
			corrupt = 1 + e.intn(9)
		}
		length := e.pick(0, 2, 4, 6, 8, 11)
		if corrupt == 1 {
			length = e.pick(1, 3, 5, 7, 9, 10)
		}
		tail := append([]int(nil), mainTail[:length]...)
		if e.chance(40) || corrupt == 2 {
			at := e.pick(0, 2, 4, 6, 8, 11)
			if corrupt == 2 {
				at = e.pick(1, 3, 5)
			}
			if at > len(tail) {
				at = len(tail)
			}
			chain := append([]int(nil), captain...)
			if corrupt == 3 {
				chain = chain[:e.rangeInt(1, 5)]
			}
			if corrupt == 4 {
				chain[e.intn(6)] = e.pick(2, 3)
			}
			tail = append(tail[:at], append(chain, tail[at:]...)...)
		}
		if corrupt == 5 && len(tail) > 0 {
			tail[e.intn(len(tail))] = e.intn(10)
		}
		prefix := 76
		if corrupt == 6 {
			prefix = e.pick(74, 75)
		}
		units := []ch28Unit{}
		for index := 0; index < prefix+len(tail); index++ {
			group := 0
			if index >= 20 && index < prefix {
				group = 8
			} else if index >= prefix {
				group = tail[index-prefix]
			}
			units = append(units, ch28Unit{Group: group, HP: e.pick(0, 1, 0x1234, 0xffff), X: e.intn(40), Y: e.intn(40),
				Pose: e.intn(4), Byte5: e.pick(0, 0, 1), HasMap: true, HasByte5: true})
		}
		if corrupt == 7 {
			u := &units[e.intn(len(units))]
			switch e.intn(3) {
			case 0:
				u.HasMap = false
			case 1:
				u.HasByte5 = false
			default:
				u.HP = e.pick(-1, 0x10000)
			}
		}
		roster := []int{2, 3, 9}
		if corrupt == 8 {
			roster = [][]int{{2, 3}, {9, 9}, {}}[e.intn(3)]
		}
		projection := e.pick(0, 1, 1)
		if corrupt == 9 {
			projection = e.pick(2, 3, 4)
		}
		st, in := e.ch28State(units, roster, projection)
		in["corrupt"] = corrupt
		res := obj{"validate": errv(battle.ValidateNativeCh28PostFrontier(st))}
		res["prelude"] = errv(battle.ApplyNativeCh28PostRawPrelude(st))
		res["after"] = ch28Summary(st)
		e.add("ch28_prelude", in, res)
	}
	for k := 0; k < 120; k++ {
		n := e.intn(9)
		cameraX, cameraY := e.intn(30), e.intn(30)
		units := make([]ch28Unit, n)
		for i := range units {
			units[i] = ch28Unit{HP: e.pick(0, 0, 1, 0xffff), X: cameraX + e.rangeInt(-3, 16), Y: cameraY + e.rangeInt(-3, 12),
				Pose: e.intn(4), Byte5: e.pick(0, 0, 1, 2, 3), HasMap: !e.chance(1), HasByte5: !e.chance(1)}
			if units[i].X < 0 {
				units[i].X = 0
			}
			if units[i].Y < 0 {
				units[i].Y = 0
			}
		}
		st, in := e.ch28State(units, nil, e.pick(0, 0, 0, 1, 1, 1, 2, 3, 4))
		frame := e.rangeInt(-1, 14)
		in["camera_x"], in["camera_y"], in["frame"] = cameraX, cameraY, frame
		plan, err := battle.PlanNativeCh28PostPresentation(st, cameraX, cameraY)
		res := obj{"plan": gj(plan), "plan_err": errv(err)}
		res["pose_err"] = errv(battle.ApplyNativeCh28PostPoseFrame(st, frame))
		res["after_pose"] = ch28Summary(st)
		res["inactive_err"] = errv(battle.ApplyNativeCh28PostInactiveMark(st))
		res["after_inactive"] = ch28Summary(st)
		e.add("ch28_present", in, res)
	}
}

// ---- 面板繪製 ----

// syntheticAssets 與 Go 測試 nativeItemPanelTestAssets 相同：raw cell 與 frame 的像素等於索引。
func syntheticAssets(textWord uint16) battle.NativeItemPanelDataAssets {
	raw := make(map[int]fdother.RawCell)
	for index := 0; index <= 92; index++ {
		raw[index] = fdother.RawCell{Width: 1, Height: 1, Pixels: []byte{byte(index)}}
	}
	frames := make(map[int]fdother.Frame)
	for index := 0; index <= 140; index++ {
		frames[index] = fdother.Frame{Width: 1, Height: 1, Pixels: []byte{1, 0, 1, 0, 0, byte(index)}}
	}
	const count = 500
	data := make([]byte, count*2+count*4)
	for index := 0; index < count; index++ {
		offset := count*2 + index*4
		binary.LittleEndian.PutUint16(data[index*2:], uint16(offset))
		binary.LittleEndian.PutUint16(data[offset:], textWord)
		binary.LittleEndian.PutUint16(data[offset+2:], fdtxt.StringEnd)
	}
	strs, err := fdtxt.Parse(data)
	if err != nil {
		panic(err)
	}
	fontData := make([]byte, fdtxt.GlyphBytes)
	fontData[0] = 0x80
	font, err := fdtxt.ParseFont(fontData)
	if err != nil {
		panic(err)
	}
	panel := fdother.LMI1Entry{Width: 149, Height: 42, Pixels: make([]byte, 149*42)}
	for i := range panel.Pixels {
		panel.Pixels[i] = byte(i % 7)
	}
	return battle.NativeItemPanelDataAssets{RawCells: raw, Frames: frames, Strings: strs, Font: font, BattlePanel: panel}
}

func (e *exporter) loadAssets() {
	fdotherPath := filepath.Join(e.flame, "FDOTHER.DAT")
	fdtxtPath := filepath.Join(e.flame, "FDTXT.DAT")
	e.assets = map[string]battle.NativeItemPanelDataAssets{
		"synthetic":      syntheticAssets(0),
		"synthetic_ctrl": syntheticAssets(0xfffe),
	}
	var err error
	must := func(name string, a battle.NativeItemPanelDataAssets, err error) {
		if err != nil {
			panic(fmt.Sprintf("%s: %v", name, err))
		}
		e.assets[name] = a
	}
	a, err := battle.LoadNativeItemPanelDataAssets(e.pack)
	must("separated", a, err)
	a, err = battle.LoadNativeItemPanelDataAssetsArchive(fdotherPath, fdtxtPath)
	must("archive", a, err)
	a, err = battle.LoadNativeBattlePanelValueAssets(e.pack)
	must("values", a, err)
	a, err = battle.LoadNativeBattlePanelValueAssetsArchive(fdotherPath)
	must("values_archive", a, err)
	noText := syntheticAssets(0)
	noText.Strings, noText.Font = nil, nil
	e.assets["synthetic_no_text"] = noText
	_ = err
}

// assetChoice 依名稱取素材；"nil_maps" 是 RawCells／Frames 為 nil 的合成素材。
func (e *exporter) assetChoice(names ...string) (string, battle.NativeItemPanelDataAssets) {
	name := names[e.intn(len(names))]
	if e.chance(65) {
		// 多數案例用完整素材，讓繪製走到成功路徑；其餘從呼叫端列出的清單挑，走缺件分支。
		name = []string{"separated", "archive", "synthetic"}[e.intn(3)]
	}
	if name == "nil_maps" {
		a := syntheticAssets(0)
		a.RawCells, a.Frames = nil, nil
		return name, a
	}
	return name, e.assets[name]
}

func (e *exporter) dst() ([]byte, int) {
	fill := e.pick(0, 0, 0x44, 0x99, e.intn(256))
	dst := make([]byte, panelBytes)
	for i := range dst {
		dst[i] = byte(fill)
	}
	return dst, fill
}

func (e *exporter) panelRecord() []byte {
	record := e.bytesN(0x50)
	if e.chance(80) {
		record[8] = byte(e.intn(140))
		record[31] = byte(e.intn(10))
		record[32] = byte(e.intn(10))
		for _, off := range []int{64, 66, 68, 70, 72, 74, 76, 78, 62} {
			binary.LittleEndian.PutUint16(record[off:], uint16(e.pick(0, 1, 50, 99, 100, 999, 1000, e.intn(0x10000))))
		}
		binary.LittleEndian.PutUint16(record[66:], binary.LittleEndian.Uint16(record[64:])+uint16(e.pick(0, 0, 5)))
	}
	if e.chance(70) {
		for slot := 0; slot < 8; slot++ {
			if e.chance(65) {
				record[0x0a+slot*2], record[0x0b+slot*2] = byte(e.pick(0, 0x40)), byte(e.intn(0xd7))
			} else {
				record[0x0a+slot*2] = 0x80
			}
		}
	}
	return record
}

func (e *exporter) rawCell() (fdother.RawCell, obj) {
	w, h := 1+e.intn(8), 1+e.intn(6)
	n := w * h
	if e.chance(10) {
		n--
	}
	c := fdother.RawCell{Width: w, Height: h, Pixels: e.bytesN(n)}
	return c, obj{"width": w, "height": h, "pixels": hexs(c.Pixels)}
}

func (e *exporter) genRenders() {
	all := []string{"separated", "archive", "synthetic", "synthetic_ctrl", "values", "synthetic_no_text", "nil_maps"}
	full := []string{"separated", "archive", "synthetic", "synthetic_ctrl"}
	for k := 0; k < 80; k++ {
		name, assets := e.assetChoice(all...)
		record := e.panelRecord()
		if e.chance(3) {
			record = record[:0x4f]
		}
		dst, fill := e.dst()
		err := battle.RenderNativeItemPanelData(assets, record, dst)
		e.add("render_panel_data", obj{"assets": name, "record": hexs(record), "fill": fill},
			obj{"err": errv(err), "sha256": digest(dst)})
	}
	for k := 0; k < 80; k++ {
		name, assets := e.assetChoice(append(full, "synthetic_no_text", "nil_maps")...)
		record := e.panelRecord()
		selected := e.rangeInt(-2, 8)
		rows := e.itemRows
		if e.chance(5) {
			rows = rows[:e.intn(len(rows))]
		}
		dst, fill := e.dst()
		err := battle.RenderNativeItemPanelRows(assets, record, selected, rows, dst)
		e.add("render_panel_rows", obj{"assets": name, "record": hexs(record), "fill": fill, "selected": selected,
			"rows_len": len(rows)}, obj{"err": errv(err), "sha256": digest(dst)})
	}
	fdotherPath := filepath.Join(e.flame, "FDOTHER.DAT")
	fdtxtPath := filepath.Join(e.flame, "FDTXT.DAT")
	portraits := filepath.Join(e.pack, "portraits")
	used := map[int]bool{}
	for k := 0; k < 12; k++ {
		record := e.panelRecord()
		record[7] = byte(e.pick(0, 1, 5, 17, 60, 135, 136, 200, e.intn(136)))
		used[int(record[7])] = true
		dst, fill := e.dst()
		oracle := append([]byte(nil), dst...)
		err := battle.RenderNativeItemPanelResources(e.pack, portraits, record, dst)
		errArc := battle.RenderNativeItemPanelResourcesArchive(fdotherPath, fdtxtPath, portraits, record, oracle)
		e.add("render_panel_resources", obj{"record": hexs(record), "fill": fill},
			obj{"err": errv(err), "sha256": digest(dst), "archive_err": errv(errArc), "archive_sha256": digest(oracle)})
	}
	for k := 0; k < 10; k++ {
		portrait := e.pick(0, 2, 9, 30, 135, 136, -1, e.intn(136))
		if portrait >= 0 {
			used[portrait] = true
		}
		dst, fill := e.dst()
		oracle := append([]byte(nil), dst...)
		err := battle.RenderNativeItemPanelBaseResources(e.pack, portraits, portrait, dst)
		errArc := battle.RenderNativeItemPanelBaseResourcesArchive(fdotherPath, portraits, portrait, oracle)
		e.add("render_panel_base", obj{"portrait": portrait, "fill": fill},
			obj{"err": errv(err), "sha256": digest(dst), "archive_err": errv(errArc), "archive_sha256": digest(oracle)})
	}
	for p := range used {
		for m := 0; m < 4; m++ {
			path := filepath.Join(portraits, fmt.Sprintf("DATO_%03d_m%d.png", p, m))
			if _, err := os.Stat(path); err == nil {
				e.record(fmt.Sprintf("pack/portraits/DATO_%03d_m%d.png", p, m), path)
			}
		}
	}
	for k := 0; k < 80; k++ {
		name, assets := e.assetChoice(all...)
		record := e.panelRecord()
		record[6] = byte(e.pick(0, 0, 1, 2))
		unitIndex := e.pick(0, 3, 17, 17, -1, e.intn(40))
		chapter := e.pick(0, 7, 23, 24, 24, 255, 256, -1)
		withName := e.chance(60)
		dst, fill := e.dst()
		var err error
		if withName {
			err = battle.RenderNativeBattlePanel(assets, record, dst, unitIndex, chapter)
		} else {
			err = battle.RenderNativeBattlePanelWithoutName(assets, record, dst, unitIndex, chapter)
		}
		x, y, originErr := battle.NativeBattlePanelOrigin(record, unitIndex, chapter)
		e.add("render_battle_panel", obj{"assets": name, "record": hexs(record), "fill": fill, "unit": unitIndex,
			"chapter": chapter, "name": withName}, obj{"err": errv(err), "sha256": digest(dst),
			"origin": []any{x, y, errv(originErr)}})
	}
	for k := 0; k < 60; k++ {
		name, assets := e.assetChoice(all...)
		values := battle.NativeBattlePanelValues{
			Level: e.pick(0, 1, 12, 99, 100, e.intn(300)), HP: e.pick(0, 8, 28, 999, 1000, -3), MaxHP: e.pick(0, 28, 100, 999),
			MP: e.pick(0, 5, 40), MaxMP: e.pick(0, 40, 41),
		}
		x, y := e.pick(0, 171, 171, 172, -1, e.intn(200)), e.pick(0, 4, 154, 158, 159, -1)
		dst, fill := e.dst()
		err := battle.RenderNativeBattlePanelValuesAt(assets, dst, x, y, values)
		e.add("render_values_at", obj{"assets": name, "fill": fill, "x": x, "y": y, "values": gj(values)},
			obj{"err": errv(err), "sha256": digest(dst)})
	}
	for k := 0; k < 50; k++ {
		name, assets := e.assetChoice(all...)
		record := e.panelRecord()
		stride := e.pick(148, 149, 320, 0x1c8, 0x1c8, 0x1c8)
		rows := e.pick(46, 47, 60, 60, 60)
		cursor := e.rangeInt(0, 13)
		dst := make([]byte, stride*rows)
		fill := e.intn(256)
		for i := range dst {
			dst[i] = byte(fill)
		}
		err := battle.RenderNativeMapUnitWindow(assets, record, dst, stride, cursor)
		ox, oy := battle.NativeMapUnitWindowOrigin(cursor)
		e.add("render_map_unit_window", obj{"assets": name, "record": hexs(record), "fill": fill, "stride": stride,
			"rows": rows, "cursor": cursor}, obj{"err": errv(err), "sha256": digest(dst), "origin": []any{ox, oy}})
	}
	for k := 0; k < 50; k++ {
		name, assets := e.assetChoice(all...)
		cell, cellIn := e.rawCell()
		// 先產生合法的列，再以 40% 機會套用一種損壞，走各條拒收分支。
		corrupt := 0
		if e.chance(40) {
			corrupt = 1 + e.intn(4)
		}
		count := e.pick(1, 2, 3, 3)
		if corrupt == 1 {
			count = e.pick(0, 4)
		}
		rows := make([]battle.NativeReviveRow, count)
		rowIn := make([]any, len(rows))
		bad := e.intn(max(count, 1))
		for i := range rows {
			size := fdicon.NativeSize * fdicon.NativeSize
			pixels, mask := e.bytesN(size), make([]byte, size)
			for j := range mask {
				if e.chance(50) {
					mask[j] = 1
				}
			}
			if corrupt == 2 && i == bad {
				mask = mask[:size-1]
			}
			rows[i] = battle.NativeReviveRow{
				Sprite:        fdicon.Sprite{Pixels: pixels, Mask: mask},
				NameTextIndex: e.intn(140), RaceTextIndex: e.pick(140, 141, 145), ClassTextIndex: e.pick(150, 155, 160),
				Fee: e.pick(0, 28, 321, 65535, e.intn(65536)),
			}
			if corrupt == 3 && i == bad {
				rows[i].RaceTextIndex = e.pick(600, -1)
			}
			if corrupt == 4 && i == bad {
				rows[i].Fee = e.pick(65536, 99999, -1)
			}
			rowIn[i] = obj{"pixels": hexs(pixels), "mask": hexs(mask), "name": rows[i].NameTextIndex,
				"race": rows[i].RaceTextIndex, "class": rows[i].ClassTextIndex, "fee": rows[i].Fee}
		}
		selected := e.validOr(max(len(rows), 1), -1, len(rows))
		dst, fill := e.dst()
		err := battle.RenderNativeReviveRows(assets, cell, rows, selected, dst)
		e.add("render_revive", obj{"assets": name, "cell": cellIn, "rows": rowIn, "selected": selected, "fill": fill},
			obj{"err": errv(err), "sha256": digest(dst)})
	}
	for k := 0; k < 80; k++ {
		name, assets := e.assetChoice(all...)
		cell, cellIn := e.rawCell()
		ids := make([]int, e.pick(0, 1, 3, 5, 6, 7, 8))
		for i := range ids {
			ids[i] = e.intn(0xd7)
			if e.chance(2) {
				ids[i] = e.pick(-1, 0xd7, 0x100)
			}
		}
		start := e.pick(0, 0, 0, 0, 2, 4, 1, -2)
		selected := start + e.validOr(max(min(6, len(ids)-start), 1), -1, 6)
		mode := battle.NativeFacilityPriceMode(e.pick(0, 1, 1, 1, 2))
		variant := e.pick(0, 1, 2)
		dst, fill := e.dst()
		var err error
		switch variant {
		case 0:
			err = battle.RenderNativeTransferItemRows(assets, cell, ids, start, selected, e.itemRows, dst)
		case 1:
			err = battle.RenderNativeFacilityItemRows(assets, cell, ids, start, selected, e.itemRows, mode, dst)
		case 2:
			err = battle.RenderNativeFacilityItemRowsWithoutNames(assets, cell, ids, start, selected, e.itemRows, mode, dst)
		}
		e.add("render_transfer", obj{"assets": name, "cell": cellIn, "ids": ids, "start": start, "selected": selected,
			"mode": int(mode), "variant": variant, "fill": fill}, obj{"err": errv(err), "sha256": digest(dst)})
	}
	for k := 0; k < 60; k++ {
		name, assets := e.assetChoice(all...)
		x, y := e.pick(0, 10, 300, 319, -1), e.pick(0, 20, 190, 199)
		value := e.pick(0, 7, 99, 100, 999, 1000, 12345, 12345678, 123456789, -5)
		base := e.pick(31, 42, 119, 130, 0)
		width := e.pick(0, 1, 2, 3, 5, 8, 4)
		dst, fill := e.dst()
		err := battle.RenderNativeFacilityNumber(assets, dst, x, y, value, base, width)
		e.add("render_number", obj{"assets": name, "fill": fill, "x": x, "y": y, "value": value, "base": base,
			"width": width}, obj{"err": errv(err), "sha256": digest(dst)})
	}
	for k := 0; k < 60; k++ {
		name, assets := e.assetChoice(all...)
		ids := make([]int, e.pick(0, 1, 3, 5, 8, 12, 33))
		for i := range ids {
			ids[i] = e.intn(36)
			if e.chance(3) {
				ids[i] = e.pick(-1, 36)
			}
		}
		selected := e.validOr(max(len(ids), 1), -2, len(ids))
		in := obj{"assets": name, "ids": ids, "selected": selected}
		book := e.bookVariant(in)
		dst, fill := e.dst()
		in["fill"] = fill
		err := battle.RenderNativeCommandOverlay(assets, ids, book, selected, dst)
		e.add("render_command_overlay", in, obj{"err": errv(err), "sha256": digest(dst)})
	}
	prices := []any{}
	for _, item := range []int{-1, 0, 1, 57, 100, 0xd6, 0xd7, 0x100} {
		for mode := 0; mode <= 2; mode++ {
			p, err := battle.NativeFacilityItemListPrice(e.itemRows, item, battle.NativeFacilityPriceMode(mode))
			prices = append(prices, []any{p, errv(err)})
		}
	}
	e.add("facility_price", obj{}, obj{"prices": prices})
	for k := 0; k < 60; k++ {
		u := &battle.Unit{
			Lv: e.pick(0, 4, 99, 255, 255, 256, -1), HP: e.pick(5, 5, 32767, 32768, -32768, -32769), MaxHP: 10, MP: 0, MaxMP: 8,
			NativeRecordByte6: byte(e.intn(3)), HasNativeRecordByte6: e.chance(90),
			NativeRecordByte8: byte(e.intn(256)), HasNativeRecordByte8: e.chance(80),
			NativeIdentity: e.pick(0, 5, 255, 256, -1), HasNativeIdentity: e.chance(50),
		}
		var target *battle.Unit = u
		if e.chance(3) {
			target = nil
		}
		record, err := battle.NativeBattlePanelRecordForUnit(target)
		e.add("battle_panel_record", obj{"unit": optUnit(target)}, obj{"record": hexs(record), "err": errv(err)})
	}
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	flame := flag.String("flame2", "", "原版 FLAME2 目錄")
	pack := flag.String("pack", "", "分離素材包根目錄（fd2-original-b97caf22）")
	flag.Parse()
	if *out == "" || *flame == "" || *pack == "" {
		fmt.Fprintln(os.Stderr, "需要 -out、-flame2、-pack")
		os.Exit(2)
	}
	tmp, err := os.MkdirTemp("", "fd2-golden-rules-c-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x52554c43)), tmp: tmp, flame: *flame, pack: *pack}
	for _, p := range []string{"assets/data/native_item_effect_rows.json", "assets/spells.json",
		"assets/data/native_command_scene.json", "assets/data/native_command_palette_flash.json"} {
		e.record("remake/"+p, p)
	}
	e.record("flame2/FDOTHER.DAT", filepath.Join(*flame, "FDOTHER.DAT"))
	e.record("flame2/FDTXT.DAT", filepath.Join(*flame, "FDTXT.DAT"))
	e.recordTree("pack", *pack, "ui/fdother_005_item_panel")
	e.recordTree("pack", *pack, "text/FDTXT_000")
	e.recordTree("pack", *pack, "fonts")
	e.itemRows, err = battle.LoadNativeItemEffectRowPrefix("assets/data/native_item_effect_rows.json")
	if err != nil {
		panic(err)
	}
	e.book, err = battle.LoadNativeCommandRecords("assets/spells.json")
	if err != nil {
		panic(err)
	}
	e.loadAssets()
	e.genRoutes()
	e.genItemSteps()
	e.genUnitItems()
	e.genLoaders()
	e.genSelector()
	e.genPhysicalScene()
	e.genCh28()
	e.genRenders()
	sort.Slice(e.files, func(i, j int) bool { return e.files[i]["path"].(string) < e.files[j]["path"].(string) })
	doc := obj{"schema": "fd2-golden-rules-c/1", "seed": *seed, "files": e.files, "cases": e.cases}
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
	names := []string{}
	for name := range ops {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("%s=%d ", name, ops[name])
	}
	fmt.Printf("total=%d\n", len(e.cases))
}
