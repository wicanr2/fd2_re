// fd2-golden-rules-a 輸出 internal/battle「A 組」（單位、記錄與物品欄基礎）的黃金差分資料，
// 供 Rust 改寫 remake-rs/crates/fd2-rules 逐欄比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 battle 的公開 API，不改任何遊戲行為。
//   - 真實輸入：remake/assets/maps/map*/map*_units.json 全部載入（路徑排序後保存清單），
//     地圖 JSON、寶箱、回合控制列、建構器與 BindNativeFutureItemRows 後的單位；選擇器批次與
//     SpawnGroup；spells.json、native_movement_cost_rows.json、native_item_effect_rows.json 與
//     docs/data/exe_tables 的 command_learn／growth 載入器。真實檔只記路徑與 SHA-256。
//   - 產生輸入：以固定種子的 PCG 產生物品欄、原版記錄位元操作、裝備重算、建構器、指令可用性、
//     移動目的地、面板記錄、暫態計時與地圖呈現的輸入，連同結果寫進同一份 JSON，Rust 端不需重現亂數。
//
// 單位一律以 json.Marshal(battle.Unit) 傳遞（Go 欄位名；Rust 端以同名 serde 讀寫）。錯誤欄位是
// Go err.Error() 原文；Go 標準函式庫產生的錯誤（讀檔、JSON 解碼）只比接受或拒收，見 Rust 端。
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
)

type obj = map[string]any

type caseJSON struct {
	Index  int    `json:"index"`
	Op     string `json:"op"`
	Input  obj    `json:"input"`
	Result obj    `json:"result"`
}

type exporter struct {
	rng      *rand.Rand
	tmp      string
	cases    []caseJSON
	itemRows []byte
	costRows [][]byte
	book     []battle.NativeCommandRecord
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

func (e *exporter) add(op string, input, result obj) {
	e.cases = append(e.cases, caseJSON{Index: len(e.cases), Op: op, Input: input, Result: result})
}

func (e *exporter) intn(n int) int { return e.rng.IntN(n) }

func (e *exporter) chance(pct int) bool { return e.rng.IntN(100) < pct }

func (e *exporter) bytesN(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(e.rng.IntN(256))
	}
	return b
}

// unitJSON 以 Go 的 JSON 欄位名輸出單位（未匯出欄位不在其中）。
func unitJSON(u *battle.Unit) json.RawMessage {
	data, err := json.Marshal(u)
	if err != nil {
		panic(err)
	}
	return data
}

func unitsJSON(units []*battle.Unit) []json.RawMessage {
	out := make([]json.RawMessage, len(units))
	for i, u := range units {
		out[i] = unitJSON(u)
	}
	return out
}

func cloneUnit(u *battle.Unit) *battle.Unit {
	var c battle.Unit
	if err := json.Unmarshal(unitJSON(u), &c); err != nil {
		panic(err)
	}
	return &c
}

func ints(b []byte) []int {
	out := make([]int, len(b))
	for i, v := range b {
		out[i] = int(v)
	}
	return out
}

// stateSummary 只輸出 A 組 Load／地圖載入寫入的欄位；State 含 func 欄位，不能整個 json.Marshal。
func stateSummary(st *battle.State) obj {
	deploy := make([][2]int, len(st.OwnDeploy))
	for i, c := range st.OwnDeploy {
		deploy[i] = [2]int{c.X, c.Y}
	}
	var cost any
	if st.Cost != nil {
		cost = st.Cost
	}
	controls := make([][3]int, 16)
	for i, c := range st.NativeTurnEventControls {
		controls[i] = [3]int{int(c.Turn), int(c.EventID), int(c.RawCamp)}
	}
	chests := make([][2]int, 16)
	for i, c := range st.NativeChestControls {
		chests[i] = [2]int{int(c.RawType), int(c.Value)}
	}
	treasures := make([]obj, 0, len(st.Treasures))
	cells := make([]battle.Cell, 0, len(st.Treasures))
	for c := range st.Treasures {
		cells = append(cells, c)
	}
	sort.Slice(cells, func(i, j int) bool {
		if cells[i].Y != cells[j].Y {
			return cells[i].Y < cells[j].Y
		}
		return cells[i].X < cells[j].X
	})
	for _, c := range cells {
		t := st.Treasures[c]
		treasures = append(treasures, obj{"x": c.X, "y": c.Y, "slot": t.Slot, "kind": t.Kind,
			"native_type": int(t.NativeType), "value": t.Value, "hidden": t.Hidden})
	}
	events := make([][2]int, len(st.NativeFieldEvents))
	for i, ev := range st.NativeFieldEvents {
		events[i] = [2]int{int(ev.EventID), int(ev.Selector)}
	}
	alive, pending := []int{}, []int{}
	for _, c := range []battle.Camp{battle.Own, battle.Ally, battle.Enemy} {
		alive = append(alive, st.AliveCount(c))
		pending = append(pending, st.PendingCount(c))
	}
	return obj{
		"w": st.W, "h": st.H, "turn": st.Turn, "native_round_counter": st.NativeRoundCounter,
		"own_deploy": deploy, "cost": cost,
		"composition":    hexs(st.NativeCompositionEventBytes),
		"has_event_grid": st.HasNativeMapEventGrid, "event_grid": hexs(st.NativeMapEventGrid),
		"has_turn_controls": st.HasNativeTurnEventControlState, "turn_controls": controls,
		"field_event_slots": st.NativeFieldEventSlots, "field_events": events,
		"field_event_rules": st.NativeFieldEventRules,
		"tile_blit_modes":   hexs(st.NativeTileBlitModes), "terrain_control": hexs(st.NativeTerrainControl),
		"terrain_move_codes": hexs(st.NativeTerrainMoveCodes),
		"treasures":          treasures, "treasure_rules": st.NativeTreasureEventRules,
		"has_chests": st.HasNativeChestControlState, "chests": chests,
		"alive": alive, "pending": pending,
		"units": unitsJSON(st.Units),
	}
}

// ---- 真實輸入 ----

func (e *exporter) genReal() obj {
	paths, err := filepath.Glob("assets/maps/map*/map*_units.json")
	if err != nil || len(paths) == 0 {
		panic("找不到 assets/maps/map*/map*_units.json")
	}
	sort.Strings(paths)
	files := []obj{}
	record := func(path string) {
		raw, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		files = append(files, obj{"path": path, "size": len(raw), "sha256": digest(raw)})
	}
	for _, p := range []string{"assets/data/native_item_effect_rows.json", "assets/data/native_movement_cost_rows.json",
		"assets/spells.json", "assets/maps/native_turn_event_controls.json",
		"../docs/data/exe_tables/command_learn.json", "../docs/data/exe_tables/growth.json"} {
		record(p)
	}
	rows, err := battle.LoadNativeItemEffectRowPrefix("assets/data/native_item_effect_rows.json")
	if err != nil {
		panic(err)
	}
	e.itemRows = rows
	e.costRows, err = battle.LoadNativeMovementCostRows("assets/data/native_movement_cost_rows.json")
	if err != nil {
		panic(err)
	}
	e.book, err = battle.LoadNativeCommandRecords("assets/spells.json")
	if err != nil {
		panic(err)
	}
	for _, path := range paths {
		record(path)
		mapPath := filepath.Join(filepath.Dir(path), "map.json")
		if _, err := os.Stat(mapPath); err == nil {
			record(mapPath)
		}
		e.genRealMap(path)
	}
	e.genLoaders()
	return obj{"files": files, "maps": paths, "item_rows": hexs(e.itemRows)}
}

func (e *exporter) genRealMap(path string) {
	st, err := battle.Load(path)
	result := obj{"err": errv(err)}
	if err == nil {
		result["state"] = stateSummary(st)
		bindErr := st.BindNativeFutureItemRows(e.itemRows)
		result["bind_err"] = errv(bindErr)
		result["units_after_bind"] = unitsJSON(st.Units)
	}
	e.add("load_map", obj{"path": path}, result)
	if err != nil {
		return
	}
	// 選擇器批次：新 State 接上整份單位（原版建構順序之一）。
	fresh, _ := battle.Load(path)
	batchState := &battle.State{W: fresh.W, H: fresh.H}
	batchState.AppendNativeMapSelectorBatchOrLegacy(fresh.Units)
	frame, frameErr := batchState.NativeMapFrameRoster()
	batch := obj{"error": errv(batchState.NativeMapSelectorError), "units": unitsJSON(batchState.Units),
		"has_cycle": batchState.HasNativeMapCycleState, "frame_err": errv(frameErr)}
	if frameErr == nil {
		batch["frame_units"] = frame.Units
		batch["frame_foreground"] = frame.Foreground
	}
	keys := []int{}
	for i := range batchState.Units {
		key, ok := batchState.NativeMapSpriteKey(batchState.Units[i])
		if !ok {
			key = -1
		}
		keys = append(keys, key)
	}
	batch["sprite_keys"] = keys
	e.add("selector_batch", obj{"path": path}, batch)

	// SpawnGroup：最小 group 先以原版選擇器批次登場，其餘 group 依序 SpawnGroup。
	spawn, _ := battle.Load(path)
	groups := map[int]bool{}
	minGroup := 1 << 30
	for _, u := range spawn.Units {
		groups[u.Group] = true
		if u.Group < minGroup {
			minGroup = u.Group
		}
	}
	order := []int{}
	for g := range groups {
		order = append(order, g)
	}
	sort.Ints(order)
	st2 := &battle.State{W: spawn.W, H: spawn.H, PendingGroups: map[int]bool{}}
	var first []*battle.Unit
	for _, u := range spawn.Units {
		if u.Group == minGroup {
			first = append(first, u)
		} else {
			u.OnField = false
			st2.Roster = append(st2.Roster, u)
			st2.PendingGroups[u.Group] = true
		}
	}
	nativeAttempt := e.chance(50)
	if nativeAttempt {
		st2.AppendNativeMapSelectorBatchOrLegacy(first)
	} else {
		for _, u := range first {
			st2.AddUnit(u)
		}
	}
	steps := []obj{}
	for i, g := range order {
		if g == minGroup {
			continue
		}
		camp := battle.Camp(i % 3)
		change, act := i%2 == 0, i%3 != 0
		pendingBefore := []int{st2.PendingCount(battle.Own), st2.PendingCount(battle.Ally), st2.PendingCount(battle.Enemy)}
		n := st2.SpawnGroup(g, camp, change, act)
		delete(st2.PendingGroups, g)
		steps = append(steps, obj{"group": g, "camp": int(camp), "change": change, "act": act, "n": n,
			"pending_before": pendingBefore, "roster_left": len(st2.Roster)})
	}
	e.add("spawn_groups", obj{"path": path, "min_group": minGroup, "native_attempt": nativeAttempt},
		obj{"steps": steps, "error": errv(st2.NativeMapSelectorError), "units": unitsJSON(st2.Units)})
}

func (e *exporter) genLoaders() {
	type loader struct {
		op, path string
	}
	for _, l := range []loader{{"load_command_records", "assets/spells.json"},
		{"load_movement_rows", "assets/data/native_movement_cost_rows.json"},
		{"load_command_learn", "../docs/data/exe_tables/command_learn.json"},
		{"load_command_learn_selectors", "../docs/data/exe_tables/growth.json"}} {
		raw, err := os.ReadFile(l.path)
		if err != nil {
			panic(err)
		}
		e.add(l.op, obj{"path": l.path}, e.runLoader(l.op, l.path))
		// 竄改過的複本：改一個欄位值或刪掉一列，Go 自己的拒收訊息必須逐字相同。
		for k := 0; k < 6; k++ {
			mutated := e.mutateJSONText(string(raw))
			path := filepath.Join(e.tmp, fmt.Sprintf("%s-%d.json", l.op, k))
			if err := os.WriteFile(path, []byte(mutated), 0o600); err != nil {
				panic(err)
			}
			e.add(l.op, obj{"text": mutated}, e.runLoader(l.op, path))
		}
	}
}

func (e *exporter) runLoader(op, path string) obj {
	switch op {
	case "load_command_records":
		r, err := battle.LoadNativeCommandRecords(path)
		return obj{"err": errv(err), "records": r}
	case "load_movement_rows":
		r, err := battle.LoadNativeMovementCostRows(path)
		rows := []string{}
		for _, row := range r {
			rows = append(rows, hexs(row))
		}
		return obj{"err": errv(err), "rows": rows}
	case "load_command_learn":
		r, err := battle.LoadCommandLearn(path)
		return obj{"err": errv(err), "table": r}
	default:
		r, err := battle.LoadCommandLearnSelectors(path)
		return obj{"err": errv(err), "selectors": r}
	}
}

// mutateJSONText 以字串層級改一個數字、改一個十六進位字元或刪掉陣列的最後一列。
func (e *exporter) mutateJSONText(text string) string {
	switch e.intn(4) {
	case 0: // 改第 k 個 "idx"／"id"／"selector" 的值
		keys := []string{`"idx": `, `"id": `, `"selector": `, `"learn_idx": `, `"command_id": `, `"required_level": `, `"dmg": `, `"mp": `}
		key := keys[e.intn(len(keys))]
		count := strings.Count(text, key)
		if count == 0 {
			return text
		}
		k := e.intn(count)
		pos := 0
		for i := 0; i <= k; i++ {
			pos += strings.Index(text[pos:], key) + len(key)
		}
		end := pos
		for end < len(text) && (text[end] >= '0' && text[end] <= '9') {
			end++
		}
		repl := []string{"0", "1", "5", "40", "255", "256", "999", "-1"}[e.intn(8)]
		return text[:pos] + repl + text[end:]
	case 1: // 改 raw 的一個十六進位字元或長度
		count := strings.Count(text, `"raw": "`)
		if count == 0 {
			return text
		}
		k := e.intn(count)
		pos := 0
		for i := 0; i <= k; i++ {
			pos += strings.Index(text[pos:], `"raw": "`) + len(`"raw": "`)
		}
		switch e.intn(3) {
		case 0:
			return text[:pos] + "zz" + text[pos+2:]
		case 1:
			return text[:pos] + "00" + text[pos:]
		default:
			return text[:pos] + text[pos+1:]
		}
	case 2: // 刪掉最後一列
		last := strings.LastIndex(text, "\n },")
		if last < 0 {
			return text
		}
		closing := strings.LastIndex(text, "\n }")
		return text[:last+3] + text[closing+3:]
	default: // 空表
		return "[]"
	}
}

// ---- 產生輸入：單位物品欄 ----

func (e *exporter) randomInventoryUnit() *battle.Unit {
	u := &battle.Unit{}
	if e.chance(80) {
		source := make([]int, 8)
		for i := range source {
			if e.chance(45) {
				source[i] = 0xff
			} else {
				source[i] = e.intn(0x100)
			}
		}
		u.InventorySlots = source
		flags, _ := battle.NativeInventoryFlagsFromSource(source)
		u.NativeInventoryFlags = flags
		for i, item := range source {
			if flags[i]&0x80 == 0 {
				// 第一格保留 0xff 槽時建構器把第二格收進第一格；這裡照 Load 的 compact 視圖重建。
				_ = item
			}
		}
		for i := range source {
			if flags[i]&0x80 == 0 {
				u.Inventory = append(u.Inventory, source[i])
				u.Equipped = append(u.Equipped, flags[i]&0x40 != 0)
			}
		}
		if e.chance(5) && len(u.Inventory) > 0 { // 偶爾讓投影不一致
			u.Inventory[e.intn(len(u.Inventory))] ^= 1
		}
	} else {
		for n := e.intn(9); n > 0; n-- {
			u.Inventory = append(u.Inventory, e.intn(0x100))
			u.Equipped = append(u.Equipped, e.chance(30))
		}
		if e.chance(40) {
			u.InventorySlots = make([]int, 8)
			for i := range u.InventorySlots {
				u.InventorySlots[i] = 0xff
			}
			for i, id := range u.Inventory {
				u.InventorySlots[i] = id
			}
		}
	}
	return u
}

func (e *exporter) genInventory() {
	for k := 0; k < 400; k++ {
		u := e.randomInventoryUnit()
		other := e.randomInventoryUnit()
		input := obj{"unit": unitJSON(u), "other": unitJSON(other)}
		ops := []obj{}
		for step := 0; step < 6; step++ {
			var op obj
			switch e.intn(8) {
			case 0:
				id, eq := e.intn(0x100), e.chance(50)
				ok := u.AddInventoryItem(id, eq)
				op = obj{"op": "add", "id": id, "equipped": eq, "ok": ok}
			case 1:
				i := e.intn(10) - 1
				ok := u.RemoveInventoryIndex(i)
				op = obj{"op": "remove_index", "i": i, "ok": ok}
			case 2:
				i := e.intn(10) - 1
				err := battle.RemoveNativeCompactInventory(u, i)
				op = obj{"op": "remove_compact", "i": i, "err": errv(err)}
			case 3:
				sel, item := e.intn(10)-1, e.intn(0x102)-1
				err := battle.ReplaceNativeFullInventoryReward(u, sel, item)
				op = obj{"op": "replace_reward", "sel": sel, "item": item, "err": errv(err)}
			case 4:
				op = obj{"op": "validate", "err": errv(battle.ValidateNativeInventoryProjection(u))}
			case 5:
				records, err := battle.NativeInventoryRecords([]*battle.Unit{u}, 1)
				op = obj{"op": "records", "err": errv(err), "buf": hexs(records)}
			case 6:
				i := e.intn(10) - 1
				err := battle.TransferNativeInventoryItem(u, i, other)
				op = obj{"op": "transfer", "i": i, "err": errv(err)}
			default:
				i := e.intn(10) - 1
				err := battle.TransferNativeInventoryItem(u, i, u)
				op = obj{"op": "transfer_self", "i": i, "err": errv(err)}
			}
			op["unit"] = unitJSON(u)
			op["other"] = unitJSON(other)
			ops = append(ops, op)
		}
		e.add("unit_inventory", input, obj{"ops": ops})
	}
	for k := 0; k < 300; k++ {
		n := 8
		if e.chance(10) {
			n = e.intn(10)
		}
		source, flags, slots := make([]int, n), make([]int, n), make([]int, n)
		for i := 0; i < n; i++ {
			pick := func() int {
				switch {
				case e.chance(1):
					return 0x100 + e.intn(4)
				case e.chance(1):
					return -1 - e.intn(3)
				case e.chance(40):
					return 0xff
				default:
					return e.intn(0x100)
				}
			}
			source[i], slots[i] = pick(), pick()
			flags[i] = []int{0, 0x40, 0x80, 0xc0, e.intn(0x100)}[e.intn(5)]
			if e.chance(1) {
				flags[i] = 0x100
			}
		}
		idx, a2 := e.intn(10)-1, e.intn(3)
		fromSource, errSource := battle.NativeInventoryFlagsFromSource(source)
		eligible, errEligible := battle.NativeInventoryCompactEligible(flags, slots, idx)
		count, errCount := battle.NativeInventoryAvailableCount(flags)
		slot, errSlot := battle.NativeEquippedInventorySlot(flags, slots, a2)
		e.add("inventory_pure", obj{"source": source, "flags": flags, "slots": slots, "idx": idx, "a2": a2},
			obj{"from_source": fromSource, "from_source_err": errv(errSource),
				"eligible": eligible, "eligible_err": errv(errEligible),
				"count": count, "count_err": errv(errCount), "slot": slot, "slot_err": errv(errSlot)})
	}
}

// ---- 產生輸入：原版記錄位元操作 ----

func (e *exporter) recordBuffer() []byte {
	count := 1 + e.intn(4)
	if e.chance(15) {
		count = 16 + e.intn(4)
	}
	buf := e.bytesN(count * 0x50)
	if e.chance(10) {
		buf = buf[:len(buf)-1-e.intn(0x30)]
	}
	return buf
}

func (e *exporter) smallInt(limit int) int {
	if e.chance(8) {
		return []int{-1, -0x50, 0x50, 0x100, 1 << 20}[e.intn(5)]
	}
	return e.intn(limit)
}

func (e *exporter) indexBytes(max int) []byte {
	out := make([]byte, e.intn(5))
	for i := range out {
		out[i] = byte(e.intn(max + 2))
	}
	return out
}

func (e *exporter) genRecords() {
	ops := []string{"assign_reserved", "remove_slot", "find_item", "find_item_in_unit", "occupied_count",
		"set_coordinates", "rewrite_marker", "set_byte5", "byte5_bit0", "set_bit7", "clear_bit7",
		"clear_unit_byte", "word_delta_remove", "word_subtract", "hp_damage", "hp_restore", "mp_restore",
		"hp_restore_list", "application", "flag_restore", "pair_step", "word_step", "persistent_copy",
		"persistent_identity", "relocation_allowed"}
	for k := 0; k < 2500; k++ {
		op := ops[k%len(ops)]
		buf := e.recordBuffer()
		count := len(buf) / 0x50
		in := obj{"op": op, "buf": hexs(buf)}
		res := obj{}
		rng := uint16(e.intn(0x10000))
		switch op {
		case "assign_reserved":
			unit, item := e.smallInt(count+1), byte(e.intn(256))
			ok, err := battle.AssignNativeReservedItem(buf, unit, item)
			in["unit"], in["item"] = unit, int(item)
			res["ok"], res["err"] = ok, errv(err)
		case "remove_slot":
			unit, slot := e.smallInt(count+1), e.smallInt(9)
			err := battle.RemoveNativeInventorySlot(buf, unit, slot)
			in["unit"], in["slot"] = unit, slot
			res["err"] = errv(err)
		case "find_item":
			item := byte(e.intn(256))
			u, s, err := battle.FindNativeInventoryItem(buf, item)
			in["item"] = int(item)
			res["unit"], res["slot"], res["err"] = u, s, errv(err)
		case "find_item_in_unit":
			unit, item := e.smallInt(count+1), byte(e.intn(256))
			s, err := battle.FindNativeInventoryItemInUnit(buf, unit, item)
			in["unit"], in["item"] = unit, int(item)
			res["slot"], res["err"] = s, errv(err)
		case "occupied_count":
			unit := e.smallInt(count + 1)
			n, err := battle.NativeInventoryOccupiedCount(buf, unit)
			in["unit"] = unit
			res["count"], res["err"] = n, errv(err)
		case "set_coordinates":
			unit, x, y := e.smallInt(count+1), byte(e.intn(256)), byte(e.intn(256))
			err := battle.SetNativeUnitCoordinateBytes(buf, unit, x, y)
			in["unit"], in["x"], in["y"] = unit, int(x), int(y)
			res["err"] = errv(err)
		case "rewrite_marker":
			n := e.smallInt(count + 2)
			err := battle.RewriteNativeMarker1F(buf, n)
			in["count"] = n
			res["err"] = errv(err)
		case "set_byte5":
			i := e.smallInt(count + 1)
			in["index"], res["err"] = i, errv(battle.SetNativeRecordByte5One(buf, i))
		case "byte5_bit0":
			i := e.smallInt(count + 1)
			v, err := battle.NativeRecordByte5Bit0(buf, i)
			in["index"], res["value"], res["err"] = i, int(v), errv(err)
		case "set_bit7":
			i := e.smallInt(count + 1)
			in["index"], res["err"] = i, errv(battle.SetNativeRecordBit7(buf, i))
		case "clear_bit7":
			n := e.smallInt(count + 2)
			in["count"], res["err"] = n, errv(battle.ClearNativeRecordBit7All(buf, n))
		case "clear_unit_byte":
			idx, off := e.indexBytes(count), e.smallInt(0x50)
			r, total, err := battle.ClearNativeUnitByte(buf, idx, off)
			in["indices"], in["offset"] = ints(idx), off
			res["results"], res["total"], res["err"] = r, total, errv(err)
		case "word_delta_remove":
			target, off, delta := e.smallInt(count+1), e.smallInt(0x50), e.intn(0x20000)-0x10000
			ru, rs := e.smallInt(count+1), e.smallInt(9)
			v, err := battle.ApplyNativeWordDeltaAndRemove(buf, target, off, delta, ru, rs)
			in["target"], in["offset"], in["delta"], in["removal_unit"], in["removal_slot"] = target, off, delta, ru, rs
			res["value"], res["err"] = int(v), errv(err)
		case "word_subtract":
			unit, off, amount := e.smallInt(count+1), e.smallInt(0x50), e.smallInt(0x100)
			v, err := battle.ApplyNativeRawWordSubtract(buf, unit, off, amount)
			in["unit"], in["offset"], in["amount"] = unit, off, amount
			res["value"], res["err"] = int(v), errv(err)
		case "hp_damage", "hp_restore", "mp_restore":
			unit, amount := e.smallInt(count+1), e.smallInt(500)
			in["unit"], in["amount"], in["rng"] = unit, amount, int(rng)
			var r any
			var err error
			switch op {
			case "hp_damage":
				r, err = battle.ApplyNativeRawHPDamage(buf, unit, amount, rng)
			case "hp_restore":
				r, err = battle.ApplyNativeRawHPRestore(buf, unit, amount, rng)
			default:
				r, err = battle.ApplyNativeRawMPRestore(buf, unit, amount, rng)
			}
			res["result"], res["err"] = r, errv(err)
		case "hp_restore_list":
			idx, amount := e.indexBytes(count), e.smallInt(500)
			r, err := battle.ApplyNativeRawHPRestoreList(buf, idx, amount, rng)
			in["indices"], in["amount"], in["rng"] = ints(idx), amount, int(rng)
			res["result"], res["err"] = r, errv(err)
		case "application", "flag_restore":
			idx, off := e.indexBytes(count), e.smallInt(0x50)
			in["indices"], in["offset"], in["rng"] = ints(idx), off, int(rng)
			if op == "application" {
				r, state, total, err := battle.ApplyNativeRawApplication(buf, idx, off, rng)
				res["results"], res["rng"], res["total"], res["err"] = r, int(state), total, errv(err)
			} else {
				r, state, total, err := battle.ApplyNativeRawFlagRestore(buf, idx, off, rng)
				res["results"], res["rng"], res["total"], res["err"] = r, int(state), total, errv(err)
			}
		case "pair_step":
			idx := e.indexBytes(count)
			r, state, total, err := battle.ApplyNativeRawPairStep(buf, idx, rng)
			in["indices"], in["rng"] = ints(idx), int(rng)
			res["results"], res["rng"], res["total"], res["err"] = r, int(state), total, errv(err)
		case "word_step":
			idx, marker, word := e.indexBytes(count), e.smallInt(0x50), e.smallInt(0x50)
			r, state, total, err := battle.ApplyNativeRawWordStepAtOffsets(buf, idx, rng, marker, word)
			in["indices"], in["rng"], in["marker"], in["word"] = ints(idx), int(rng), marker, word
			res["results"], res["rng"], res["total"], res["err"] = r, int(state), total, errv(err)
		case "persistent_copy":
			persistent := e.recordBuffer()
			ri, pi := e.smallInt(count+1), e.smallInt(len(persistent)/0x50+1)
			err := battle.ApplyNativePersistentRecordCopy(buf, persistent, ri, pi)
			in["persistent"], in["runtime_index"], in["persistent_index"] = hexs(persistent), ri, pi
			res["err"], res["persistent"] = errv(err), hexs(persistent)
		case "persistent_identity":
			n, identity := e.smallInt(count+2), byte(e.intn(256))
			if e.chance(50) && count > 0 {
				identity = buf[e.intn(count)*0x50+8]
			}
			i, ok, err := battle.FindNativePersistentIdentity(buf, n, identity)
			in["count"], in["identity"] = n, int(identity)
			res["index"], res["ok"], res["err"] = i, ok, errv(err)
		case "relocation_allowed":
			n, target := e.smallInt(count+1), e.smallInt(count+1)
			x, y, terrain := byte(e.intn(8)), byte(e.intn(8)), e.smallInt(20)
			rows := e.costRows
			if e.chance(10) {
				rows = rows[:28]
			}
			for i := 0; i+0x50 <= len(buf); i += 0x50 { // 讓座標容易撞在一起
				buf[i], buf[i+1] = byte(e.intn(8)), byte(e.intn(8))
				buf[i+7] = []byte{0x1c, buf[i+7]}[e.intn(2)]
				buf[i+0x20] = byte(e.intn(29))
			}
			in["buf"] = hexs(buf)
			ok, err := battle.NativeRelocationDestinationAllowed(buf, n, target, x, y, terrain, rows)
			in["count"], in["target"], in["x"], in["y"], in["terrain"], in["short_rows"] = n, target, int(x), int(y), terrain, len(rows) != 29
			res["ok"], res["err"] = ok, errv(err)
		}
		res["buf"] = hexs(buf)
		e.add("records", in, res)
	}
}

// ---- 產生輸入：裝備重算與建構器 ----

func (e *exporter) genEquipment() {
	for k := 0; k < 400; k++ {
		size := 0x50
		if e.chance(5) {
			size = e.intn(0x50)
		}
		record := e.bytesN(size)
		for slot := 0; slot < 8 && size >= 0x1a; slot++ {
			record[0x0a+slot*2] &= 0xc0
			if e.chance(60) {
				record[0x0b+slot*2] %= 40
			}
		}
		if size > 0x24 && e.chance(50) {
			record[0x22], record[0x23], record[0x24] = 0, 0, 0
		}
		rowCount := 0x100
		if e.chance(40) {
			rowCount = 1 + e.intn(48)
		}
		rows := e.bytesN(rowCount * battle.NativeItemEffectRowSize)
		if e.chance(5) {
			rows = nil
		}
		runtime := e.chance(50)
		in := obj{"record": hexs(record), "table": hexs(rows), "runtime": runtime}
		var err error
		if runtime {
			err = battle.ApplyNativeRuntimeEquipmentRecalc(record, rows)
		} else {
			err = battle.ApplyNativeEquipmentRecalc(record, rows)
		}
		e.add("equipment", in, obj{"err": errv(err), "record": hexs(record)})
	}
	for k := 0; k < 400; k++ {
		t := &battle.NativeConstructorTable{Branch: []string{"high_class", "lower_class", "high_class", "lower_class", "other"}[e.intn(5)]}
		if t.Branch == "high_class" {
			t.Index = e.intn(70) - 1
		} else {
			t.Index = e.intn(34) - 1
		}
		if t.Branch == "high_class" {
			t.Record = e.bytesN([]int{10, 10, 10, 9}[e.intn(4)])
			if e.chance(5) {
				t.AuxRecord = e.bytesN(1)
			}
		} else {
			t.Record = e.bytesN([]int{24, 24, 24, 10}[e.intn(4)])
			t.AuxRecord = e.bytesN([]int{11, 11, 11, 0}[e.intn(4)])
		}
		level := byte(e.intn(256))
		var table *battle.NativeConstructorTable
		if !e.chance(3) {
			table = t
		}
		base, err := battle.DecodeNativeFutureConstructorBase(table, level)
		tj := any(nil)
		if table != nil {
			tj = table
		}
		e.add("constructor_base", obj{"table": tj, "level": int(level)}, obj{"base": base, "err": errv(err)})

		u := e.randomInventoryUnit()
		u.Lv = e.intn(40)
		if e.chance(5) {
			u.Lv = []int{-1, 256, 300}[e.intn(3)]
		}
		u.NativeRecordByte6, u.HasNativeRecordByte6 = byte(e.intn(4)), e.chance(90)
		if e.chance(90) {
			u.NativeConstructor = t
		}
		u.NativeTransient = [6]byte{byte(e.intn(4)), 0, byte(e.intn(4)), 1, 2, 3}
		u.AP, u.DP, u.Exp = e.intn(100), e.intn(100), float64(e.intn(300))
		rows := e.itemRows
		if e.chance(20) {
			rows = rows[:battle.NativeItemEffectRowSize*(1+e.intn(8))]
		}
		in := obj{"unit": unitJSON(u), "rows_len": len(rows)}
		err = battle.MaterializeNativeFutureConstructor(u, rows)
		e.add("materialize_constructor", in, obj{"err": errv(err), "unit": unitJSON(u)})
	}
}

// ---- 產生輸入：指令可用性與遮罩 ----

func (e *exporter) genCommands() {
	for k := 0; k < 400; k++ {
		u := &battle.Unit{MP: e.intn(60) - 5}
		for i := range u.NativeCommandMask {
			if e.chance(60) {
				u.NativeCommandMask[i] = byte(e.intn(256))
			}
		}
		if e.chance(30) {
			u.NativeTransient[5] = byte(1 + e.intn(3))
		}
		book := append([]battle.NativeCommandRecord(nil), e.book...)
		bookKind := "real"
		switch {
		case e.chance(10):
			book, bookKind = book[:35], "short"
		case e.chance(10):
			book[e.intn(36)].ID = 99
			bookKind = "bad_id"
		case e.chance(10):
			book[e.intn(36)].MPCost = -1
			bookKind = "negative_cost"
		}
		in := obj{"unit": unitJSON(u), "book": book, "book_kind": bookKind}
		available := []bool{}
		for id := -1; id <= 40; id++ {
			available = append(available, battle.NativeCommandAvailable(u, book, id))
		}
		res := obj{"available": available,
			"ids":    battle.NativeAvailableCommandIDs(u, book),
			"ai_ids": battle.NativeAvailableAICommandIDs(u, book),
			"scored": battle.NativeAvailableAIScoredCommandIDs(u, book),
			"native": u.NativeCommandIDs()}
		cost := e.intn(0x110) - 5
		res["spend_cost"], res["spend_ok"] = cost, battle.SpendNativeCommandMP(u, cost)
		res["mp_after"] = u.MP
		raw := e.bytesN([]int{0, 4, 4, 4, 3, 5}[e.intn(6)])
		res["mask_raw"], res["mask_err"] = hexs(raw), errv(u.SetInitialCommandMask(raw))
		enabled := []obj{}
		for i := 0; i < 4; i++ {
			id := e.intn(46) - 3
			enabled = append(enabled, obj{"id": id, "ok": u.EnableNativeCommand(id)})
		}
		res["enabled"], res["mask_after"], res["ids_after"] = enabled, ints(u.NativeCommandMask[:]), u.NativeCommandIDs()
		e.add("commands", in, res)
	}
	for k := 0; k < 120; k++ {
		classID, item := e.intn(0x104)-2, e.intn(0x104)-2
		row := e.bytesN([]int{7, 6, 5, 0}[e.intn(4)])
		probe := byte(e.intn(256))
		if len(row) > 0 && e.chance(50) {
			probe = row[e.intn(len(row))]
		}
		off, errOff := battle.NativeClassCompatibilityRowOffset(classID)
		compat, errCompat := battle.NativeClassItemCompatible(probe, row)
		rowOff, errRow := battle.NativeItemEffectRowOffset(item)
		route, ok := battle.NativeItemWordDeltaRouteForType(e.intn(14) - 1)
		e.add("tables", obj{"class": classID, "item": item, "row": hexs(row), "probe": int(probe), "type": route.ItemType},
			obj{"class_off": off, "class_err": errv(errOff), "compat": compat, "compat_err": errv(errCompat),
				"row_off": rowOff, "row_err": errv(errRow), "route": route, "route_ok": ok})
	}
}

// ---- 產生輸入：移動目的地 ----

func (e *exporter) genDestinations() {
	for k := 0; k < 400; k++ {
		w, h := 1+e.intn(8), 1+e.intn(8)
		if e.chance(3) {
			w = 0
		}
		cells := w * h
		count := 1 + e.intn(4)
		records := make([]byte, count*0x50)
		for i := 0; i < count; i++ {
			records[i*0x50], records[i*0x50+1] = byte(e.intn(max(w, 1))), byte(e.intn(h))
			if e.chance(5) {
				records[i*0x50] = byte(w)
			}
			records[i*0x50+5] = []byte{0, 0, 1, 0x80}[e.intn(4)]
			records[i*0x50+6] = []byte{0, 0, 1, 2}[e.intn(4)]
		}
		actor, selector, budget := e.intn(count), e.intn(3), e.intn(12)
		if e.chance(5) {
			actor = count
		}
		if e.chance(5) {
			budget = 0xff
		}
		flags, codes := make([]byte, cells), make([]byte, cells)
		for i := range codes {
			codes[i] = byte(e.intn(6))
			if e.chance(10) {
				flags[i] = []byte{0x40, 0x80}[e.intn(2)]
			}
		}
		if e.chance(3) && cells > 0 {
			codes[0] = 20
		}
		costRow := e.costRows[e.intn(len(e.costRows))]
		if e.chance(5) {
			costRow = costRow[:19]
		}
		in := obj{"w": w, "h": h, "records": hexs(records), "count": count, "actor": actor, "selector": selector,
			"budget": budget, "flags": hexs(flags), "codes": hexs(codes), "cost_row": hexs(costRow)}
		got, err := battle.NativeAIPhysicalDestinations(w, h, records, count, actor, selector, budget, flags, codes, costRow)
		cellsOut := [][2]int{}
		for _, c := range got {
			cellsOut = append(cellsOut, [2]int{c.X, c.Y})
		}
		e.add("destinations", in, obj{"cells": cellsOut, "err": errv(err)})
	}
}

// ---- 產生輸入：面板記錄、暫態、地圖呈現 ----

func (e *exporter) randomRawUnit() *battle.Unit {
	u := e.randomInventoryUnit()
	if len(u.NativeInventoryFlags) != 8 && e.chance(80) {
		u.InventorySlots = []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
		u.NativeInventoryFlags = []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
		u.Inventory, u.Equipped = nil, nil
	}
	u.MapSelectorSlot, u.HasMapSelectorSlot = e.intn(20), e.chance(80)
	u.BattleFig, u.HasBattleFig = e.intn(0x100), e.chance(95)
	if e.chance(3) {
		u.BattleFig = []int{-1, 0x100}[e.intn(2)]
	}
	u.NativeIdentity, u.HasNativeIdentity = e.intn(0x104)-2, e.chance(40)
	u.NativeRecordByte8, u.HasNativeRecordByte8 = byte(e.intn(256)), e.chance(60)
	u.NativeRecordByte5, u.HasNativeRecordByte5 = []byte{0, 0, 1, 0x80, 0x81}[e.intn(5)], e.chance(90)
	u.NativeRecordByte6, u.HasNativeRecordByte6 = byte(e.intn(3)), e.chance(97)
	u.NativeRecordRace, u.HasNativeRecordRace = byte(e.intn(8)), e.chance(97)
	u.NativeRecordClass, u.HasNativeRecordClass = byte(e.intn(30)), e.chance(97)
	u.NativeRecordByte34, u.HasNativeRecordByte34 = byte(e.intn(256)), e.chance(97)
	u.NativeRecordByte35, u.HasNativeRecordByte35 = byte(e.intn(256)), e.chance(97)
	u.NativeRecordByte36, u.HasNativeRecordByte36 = byte(e.intn(256)), e.chance(97)
	u.NativeRecordWord42, u.HasNativeRecordWord42 = uint16(e.intn(0x10000)), e.chance(97)
	u.NativeRecordWord46, u.HasNativeRecordWord46 = uint16(e.intn(0x10000)), e.chance(97)
	u.Lv, u.MV, u.DX = e.intn(0x100), e.intn(0x100), e.intn(0x9000)-0x4800
	u.Exp = float64(e.intn(0x100))
	if e.chance(3) {
		u.Lv, u.Exp = []int{-1, 0x100}[e.intn(2)], float64(e.intn(0x104)-2)
	}
	if e.chance(5) {
		u.Exp += 0.5
	}
	u.HP, u.MaxHP, u.MP, u.MaxMP = e.intn(400), e.intn(400), e.intn(100), e.intn(100)
	u.AP, u.DP, u.HIT, u.EV = e.intn(300), e.intn(300), e.intn(300), e.intn(300)
	if e.chance(3) {
		u.AP = 40000
	}
	for i := range u.NativeTransient {
		if e.chance(40) {
			u.NativeTransient[i] = byte(e.intn(4))
		}
	}
	for i := range u.NativeCommandMask {
		u.NativeCommandMask[i] = byte(e.intn(256))
	}
	u.X, u.Y = e.intn(0x104)-2, e.intn(40)
	if e.chance(85) {
		u.NativeMapPresentation = battle.NativeMapPresentationState{X: byte(e.intn(40)), Y: byte(e.intn(40)),
			Pose: byte(e.intn(4)), Motion: byte(e.intn(7))}
		u.HasNativeMapPresentation = true
	}
	u.MapSelectorKey, u.HasMapSelectorKey = e.intn(0x104)-2, e.chance(90)
	return u
}

func (e *exporter) genPanels() {
	for k := 0; k < 300; k++ {
		n := 1 + e.intn(3)
		units := make([]*battle.Unit, n)
		for i := range units {
			units[i] = e.randomRawUnit()
		}
		record, errRecord := battle.NativeItemPanelRecordForUnit(units[0])
		scoring, errScoring := battle.NativeAIScoringRecords(units)
		e.add("panel_records", obj{"units": unitsJSON(units)},
			obj{"record": hexs(record), "record_err": errv(errRecord), "scoring": hexs(scoring), "scoring_err": errv(errScoring)})
	}
	for k := 0; k < 300; k++ {
		n := 1 + e.intn(5)
		units := make([]*battle.Unit, n)
		records := make([]battle.NativeRuntimeRecordState, n)
		for i := range units {
			u := &battle.Unit{HP: e.intn(60), MaxHP: e.intn(120)}
			if e.chance(3) {
				u.HP = 0x10000
			}
			u.NativeRecordByte5, u.HasNativeRecordByte5 = []byte{0, 0, 0, 1, 0x80}[e.intn(5)], e.chance(95)
			u.NativeRecordByte6, u.HasNativeRecordByte6 = byte(e.intn(3)), e.chance(95)
			for j := range u.NativeTransient {
				if e.chance(50) {
					u.NativeTransient[j] = byte(e.intn(4))
				}
			}
			units[i] = u
			raw := &records[i].Raw
			copy(raw[:], e.bytesN(0x50))
			raw[5], raw[6] = u.NativeRecordByte5, u.NativeRecordByte6
			copy(raw[0x22:0x28], u.NativeTransient[:])
			raw[0x40], raw[0x41] = byte(u.HP), byte(u.HP>>8)
			raw[0x42], raw[0x43] = byte(u.MaxHP), byte(u.MaxHP>>8)
			if e.chance(5) {
				raw[0x22+e.intn(6)] ^= 1
			}
			records[i].SelectorKey, records[i].SelectorSlot = byte(e.intn(256)), byte(e.intn(256))
		}
		mode := []string{"tick", "typed", "raw", "raw"}[e.intn(4)]
		selector := byte(e.intn(3))
		hasProjection := e.chance(90)
		rawHex := []string{}
		for _, r := range records {
			rawHex = append(rawHex, hexs(r.Raw[:]))
		}
		in := obj{"units": unitsJSON(units), "raw": rawHex, "mode": mode, "selector": int(selector),
			"has_projection": hasProjection}
		st := &battle.State{Units: units, NativeRuntimeRecords: records, HasNativeRuntimeUnitProjection: hasProjection}
		res := obj{}
		expiry := func(list []battle.NativeTransientExpiry) [][2]int {
			out := [][2]int{}
			for _, x := range list {
				index := -1
				for i, u := range st.Units {
					if u == x.Unit {
						index = i
					}
				}
				out = append(out, [2]int{index, x.Offset})
			}
			return out
		}
		switch mode {
		case "tick":
			res["expired"] = expiry(st.TickNativeTransientsRaw(selector))
		default:
			var r battle.NativeTransientPhaseResult
			var err error
			if mode == "typed" {
				r, err = st.AdvanceNativeTransientPhaseTyped(selector)
			} else {
				r, err = st.AdvanceNativeTransientPhaseRaw(selector)
			}
			damage := [][4]int{}
			for _, d := range r.Damage {
				damage = append(damage, [4]int{d.Slot, d.Amount, d.Before, d.After})
			}
			res["err"], res["damage"], res["inactive"], res["expired"] = errv(err), damage, r.InactiveSlots, expiry(r.Expired)
		}
		rawAfter := []string{}
		for _, r := range st.NativeRuntimeRecords {
			rawAfter = append(rawAfter, hexs(r.Raw[:]))
		}
		res["units"], res["raw"] = unitsJSON(st.Units), rawAfter
		e.add("transient", in, res)
	}
	for k := 0; k < 300; k++ {
		u := e.randomRawUnit()
		in := obj{"unit": unitJSON(u)}
		ops := []obj{}
		for step := 0; step < 8; step++ {
			var op obj
			a, b, c := e.intn(0x106)-3, e.intn(0x106)-3, e.intn(9)-1
			switch e.intn(7) {
			case 0:
				op = obj{"op": "materialize", "err": errv(u.MaterializeNativeMapPresentation())}
			case 1:
				op = obj{"op": "pose", "a": c, "ok": u.SetMapPose(c)}
			case 2:
				op = obj{"op": "placement", "a": a, "b": b, "c": c, "ok": u.SetMapPlacement(a, b, c)}
			case 3:
				op = obj{"op": "coords_raw", "a": a, "b": b, "ok": u.SetNativeMapCoordinatesRaw(a, b)}
			case 4:
				op = obj{"op": "grid_motion", "a": c, "b": c + e.intn(3), "ok": false}
				op["ok"] = u.SetNativeMapGridMotion(op["a"].(int), op["b"].(int))
			case 5:
				op = obj{"op": "finish_step", "a": c, "b": a, "c": b, "ok": u.FinishNativeMapGridStep(c, a, b)}
			default:
				entry, ok := u.NativeUnitLayerEntry()
				fg, okFg := u.NativeForegroundLayerEntry()
				op = obj{"op": "layers", "entry": entry, "ok": ok, "fg": fg, "ok_fg": okFg}
			}
			op["unit"] = unitJSON(u)
			ops = append(ops, op)
		}
		e.add("map_presentation", in, obj{"ops": ops})
	}
	for k := 0; k < 200; k++ {
		gateA, gateB, anchor := byte(e.intn(3)), byte(e.intn(3)), []int{1, 0xf2, 0, 252, 300, -1}[e.intn(6)]
		st := &battle.State{}
		res := obj{"advance_before": st.AdvanceNativeMapHUDAnchor(2, 6)}
		res["materialized"] = st.MaterializeNativeMapHUDState(gateA, gateB, anchor)
		steps := []obj{}
		for i := 0; i < 6; i++ {
			x, y := e.intn(14)-1, e.intn(10)
			ok := st.AdvanceNativeMapHUDAnchor(x, y)
			steps = append(steps, obj{"x": x, "y": y, "ok": ok, "anchor": st.NativeMapHUDState.AnchorX})
		}
		p := battle.InitialNativeMapHUDPersistentState()
		if e.chance(30) {
			p = battle.NativeMapHUDPersistentState{DisplayGateA: gateA, HasDisplayGateA: e.chance(50), AnchorX: anchor, HasAnchorX: e.chance(50)}
		}
		res["persistent_before"] = p
		captured := p.CaptureNativeMapHUD(st.NativeMapHUDState)
		restored := p.RestoreSavedGateA(gateB)
		runtime, ok := p.MaterializeRuntime(gateA)
		res["steps"], res["captured"], res["restored"], res["persistent"] = steps, captured, restored, p
		res["runtime"], res["runtime_ok"], res["state"] = runtime, ok, st.NativeMapHUDState
		e.add("hud", obj{"gate_a": int(gateA), "gate_b": int(gateB), "anchor": anchor}, res)
	}
}

// ---- 產生輸入：合成的 units.json／map.json ----

func (e *exporter) genSyntheticLoads() {
	for k := 0; k < 250; k++ {
		w, h := 1+e.intn(4), 1+e.intn(3)
		units := []obj{}
		for i := e.intn(4); i > 0; i-- {
			u := obj{"camp": []string{"own", "ally", "enemy", "x"}[e.intn(4)], "hp": e.intn(30), "mp": e.intn(10),
				"lv": e.intn(10), "x": e.intn(w), "y": e.intn(h), "group": e.intn(3), "fig": e.intn(40)}
			if e.chance(50) {
				u["inventory"] = []int{e.intn(256), e.intn(256)}
			}
			if e.chance(40) {
				slots := []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
				for j := range slots {
					if e.chance(40) {
						slots[j] = e.intn(256)
					}
				}
				if e.chance(10) {
					slots = slots[:7]
				}
				u["inventory_slots"] = slots
			}
			optional := map[string]func() any{
				"battle_fig":                 func() any { return e.intn(60) },
				"native_identity":            func() any { return []int{3, 68, 12, 40, 7, 99, 256, -1}[e.intn(8)] },
				"native_record_byte8":        func() any { return e.intn(256) },
				"map_selector_key":           func() any { return e.intn(256) },
				"map_selector_slot":          func() any { return e.intn(4) },
				"native_record_byte5":        func() any { return []int{0, 1, 128}[e.intn(3)] },
				"native_record_byte6":        func() any { return e.intn(3) },
				"native_record_byte34":       func() any { return e.intn(256) },
				"native_record_word42":       func() any { return e.intn(500) },
				"native_record_word46":       func() any { return e.intn(500) },
				"native_record_death_effect": func() any { return []int{1, 2, 3}[:[]int{3, 3, 3, 3, 3, 2}[e.intn(6)]] },
				"initial_command_mask":       func() any { return []int{1, 0, 0, 128}[:[]int{0, 4, 4, 4, 4, 3}[e.intn(6)]] },
				"native_position_record":     func() any { return obj{"x_word": e.intn(300), "y_word": e.intn(300), "raw_key": e.intn(256)} },
				"death_effect":               func() any { return obj{"type": e.intn(4), "value": e.intn(100)} },
				"native_constructor": func() any {
					if e.chance(50) {
						return obj{"branch": "high_class", "index": e.intn(69), "record": ints(e.bytesN([]int{10, 10, 10, 10, 9}[e.intn(5)]))}
					}
					return obj{"branch": "lower_class", "index": e.intn(33), "record": ints(e.bytesN(24)), "aux_record": ints(e.bytesN([]int{11, 11, 11, 11, 10}[e.intn(5)]))}
				},
			}
			names := []string{}
			for name := range optional {
				names = append(names, name)
			}
			sort.Strings(names)
			for _, name := range names {
				if e.chance(20) {
					u[name] = optional[name]()
				}
			}
			units = append(units, u)
		}
		doc := obj{"map": e.intn(33), "w": w, "h": h, "units": units,
			"own_deploy": []obj{{"x": e.intn(w), "y": e.intn(h)}}}
		if e.chance(40) {
			doc["native_round_counter"] = e.intn(4) - 1
		}
		if e.chance(50) {
			chests := []obj{}
			for i := e.intn(3); i >= 0; i-- {
				slot := e.intn(4)
				if e.chance(10) {
					slot = []int{-1, 16, 17}[e.intn(3)]
				}
				kind, value := []string{"item", "item", "gold", "event", "x"}[e.intn(5)], e.intn(300)-10
				if kind == "event" && e.chance(70) {
					value = []int{58, 3}[e.intn(2)]
				}
				chests = append(chests, obj{"slot": slot, "type": kind, "native_type": e.intn(3), "value": value})
			}
			doc["chests"] = chests
		}
		if e.chance(25) {
			rules := []obj{}
			for i := e.intn(3); i >= 0; i-- {
				rules = append(rules, obj{"event_id": []int{58, 58, 3, 95}[e.intn(4)], "item_by_slot": []int{29, 30, 0, 300}[:[]int{0, 2, 4, 4}[e.intn(4)]],
					"open_slots": []int{0, 1, 2, 2}[:[]int{0, 2, 3, 3, 4}[e.intn(5)]]})
			}
			doc["native_treasure_event_rules"] = rules
		}
		unitsText, _ := json.Marshal(doc)
		var mapText []byte
		if e.chance(70) {
			cells := w * h
			mw, mh := w, h
			if e.chance(10) {
				mw++
			}
			m := obj{"w": mw, "h": mh}
			fill := func(gen func(int) any) []any {
				out := make([]any, cells)
				for i := range out {
					out[i] = gen(i)
				}
				if e.chance(5) {
					out = out[:len(out)-1]
				}
				return out
			}
			m["cost"] = fill(func(int) any { return 1 + e.intn(3) })
			m["tiles"] = fill(func(int) any { return e.intn(4) })
			m["native_composition_event_bytes"] = fill(func(int) any { return e.intn(256) })
			m["native_tile_blit_modes"] = fill(func(int) any { return e.intn(256) })
			m["treasure_slots"] = fill(func(int) any { return e.intn(5) - 1 })
			m["treasure_hidden"] = fill(func(int) any { return e.chance(50) })
			m["native_field_event_slots"] = fill(func(int) any { return e.intn(4) - 1 })
			events := []obj{}
			for i := 0; i < 16; i++ {
				events = append(events, obj{"event_id": e.intn(90), "selector": e.intn(3)})
			}
			m["native_field_events"] = events
			if e.chance(40) {
				m["native_field_event_rules"] = []obj{{"event_id": e.intn(95), "selector": 1, "once_state_index": e.intn(0x22),
					"set_mode_ranges": []obj{{"start": e.intn(4) - 1, "end": e.intn(4), "mode": e.intn(18)}}}}
			}
			control := make([]int, 4*(1+e.intn(4)))
			for i := range control {
				control[i] = e.intn(256)
			}
			m["native_terrain_control"] = control
			if e.chance(5) {
				m["cost"] = "bad"
			}
			mapText, _ = json.Marshal(m)
		}
		if e.chance(5) {
			unitsText = unitsText[:len(unitsText)/2]
		}
		dir := filepath.Join(e.tmp, fmt.Sprintf("load-%d", k))
		if err := os.MkdirAll(dir, 0o700); err != nil {
			panic(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "units.json"), unitsText, 0o600); err != nil {
			panic(err)
		}
		if mapText != nil {
			if err := os.WriteFile(filepath.Join(dir, "map.json"), mapText, 0o600); err != nil {
				panic(err)
			}
		}
		st, err := battle.Load(filepath.Join(dir, "units.json"))
		res := obj{"err": errv(err)}
		if err == nil {
			res["state"] = stateSummary(st)
			res["bind_err"] = errv(st.BindNativeFutureItemRows(e.itemRows))
			res["units_after_bind"] = unitsJSON(st.Units)
			// 寶箱：每格以新單位嘗試領取。事件格網的驗證與推進屬 D 組（native_ai_mode5），
			// A 組差分先關掉格網，只比 A 組的領取、物品欄與事件狀態。
			st.HasNativeMapEventGrid = false
			claims := []obj{}
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					if _, ok := st.TreasureAt(x, y); !ok {
						continue
					}
					claimer := &battle.Unit{HP: 1, MaxHP: 1, OnField: true, X: x, Y: y}
					full := e.chance(30)
					if full {
						claimer.Inventory = []int{1, 2, 3, 4, 5, 6, 7}
					}
					st.AddUnit(claimer)
					t, ok := st.ClaimTreasure(claimer, x, y)
					claims = append(claims, obj{"x": x, "y": y, "full": full, "ok": ok,
						"slot": t.Slot, "kind": t.Kind, "value": t.Value, "inventory": claimer.Inventory,
						"event_state": ints(st.NativeEventState[:])})
				}
			}
			res["claims"] = claims
		}
		var mapAny any
		if mapText != nil {
			mapAny = string(mapText)
		}
		e.add("load_synthetic", obj{"units_json": string(unitsText), "map_json": mapAny}, res)
	}
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "需要 -out")
		os.Exit(2)
	}
	tmp, err := os.MkdirTemp("", "fd2-golden-rules-a-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x52554c45)), tmp: tmp}
	real := e.genReal()
	e.genInventory()
	e.genRecords()
	e.genEquipment()
	e.genCommands()
	e.genDestinations()
	e.genPanels()
	e.genSyntheticLoads()
	_ = fdicon.NativeMapStride
	doc := obj{"schema": "fd2-golden-rules-a/1", "seed": *seed, "real": real, "cases": e.cases}
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
