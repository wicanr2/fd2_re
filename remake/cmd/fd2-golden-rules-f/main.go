// fd2-golden-rules-f 輸出 internal/battle「F 組」（劇本事件、回合與快照）的黃金差分資料，
// 供 Rust 改寫 remake-rs/crates/fd2-rules 逐欄比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 battle 的公開 API，不改任何遊戲行為。
//   - 真實輸入：remake/assets 的 30 份劇本 scenarios/ch*.json 與 33 張地圖 maps/map*/map*_units.json
//     （路徑排序後保存清單、大小與 SHA-256），另記 native_item_effect_rows.json 與
//     native_turn_event_controls.json。
//   - 劇本腳本：每份劇本載入對應地圖，綁定未來道具列後 SetupChecked，再以固定種子的 PCG 推進 14
//     回合：殺掉隨機單位、改寫戰場狀態表、觸發回合事件並逐一執行動作、陣亡事件、勝負判定、
//     回合末回復、格子事件、event76／79。每一步的輸入（op 與 args）和結果都寫進 JSON，
//     Rust 端不需重現產生器，只依序重做同一串操作。
//   - 地圖腳本：33 張地圖不配劇本直接載入，掃格子事件表、勝負與回復，並對格子事件做隨機操作。
//   - 接手腳本：runtime_append_groups 的劇本模擬 handler 已建好隊伍，再 AdoptHandlerBattleState。
//   - 快照：腳本裡的 snapshot 步驟保存 Go json.Marshal(StateSnapshot) 的原文；Rust 端讀入再寫出、
//     以及自己算出的快照都要與它相同（正規化後逐欄）。Rust 把自己寫出的快照存成檔案後，
//     本工具的 -verify 模式再讀回、RestoreSnapshot、重新 Snapshot，與黃金比對，完成雙向檢查。
//   - 合成案例：劇本 JSON 的隨機變異（LoadScenario 接受或拒收與錯誤文字）、快照的不合法變異
//     （RestoreSnapshot 錯誤文字）、RestoreFiredEvents 的錯誤文字。
//
//   - 系統群行軍：只放不經 D 組尋路的 CommitNativeSystemGroupMarchStep（合成單位與路徑）。
//
// 不放進黃金的路徑：PlanNativeSystemGroupMarch（呼叫 D 組 AI 尋路）。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
)

type obj = map[string]any

type opJSON struct {
	Op     string `json:"op"`
	Args   obj    `json:"args,omitempty"`
	Result obj    `json:"result"`
}

type scriptJSON struct {
	Name     string   `json:"name"`
	Kind     string   `json:"kind"`
	Scenario string   `json:"scenario,omitempty"`
	Map      string   `json:"map"`
	Ops      []opJSON `json:"ops"`
}

type caseJSON struct {
	Index  int    `json:"index"`
	Op     string `json:"op"`
	Input  obj    `json:"input"`
	Result obj    `json:"result"`
}

const itemRowsPath = "assets/data/native_item_effect_rows.json"

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

func mustJSON(v any) json.RawMessage {
	raw, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return raw
}

func ints(b []byte) []int {
	out := make([]int, len(b))
	for i, v := range b {
		out[i] = int(v)
	}
	return out
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// stateDigest 是每一步之後比對的戰場摘要（Rust 端以同一組欄位計算）。
func stateDigest(st *battle.State) obj {
	units := make([][]int, 0, len(st.Units))
	for _, u := range st.Units {
		row := []int{u.X, u.Y, u.HP, u.MaxHP, u.Group, int(u.Camp), b2i(u.OnField), b2i(u.Acted),
			int(u.NativeRecordByte5), b2i(u.HasNativeRecordByte5), int(u.NativeRecordByte6),
			b2i(u.HasNativeRecordByte6), int(u.NativeRecordByte34), u.Fig, u.MapSelectorSlot,
			b2i(u.HasMapSelectorSlot), len(u.Inventory)}
		row = append(row, u.Inventory...)
		units = append(units, row)
	}
	roster := make([]int, 0, len(st.Roster))
	for _, u := range st.Roster {
		roster = append(roster, u.Group)
	}
	controls := make([][]int, 0, 16)
	for _, row := range st.NativeTurnEventControls {
		controls = append(controls, []int{int(row.Turn), int(row.EventID), int(row.RawCamp)})
	}
	flags := []string{}
	for k, v := range st.Flags {
		if v {
			flags = append(flags, k)
		}
	}
	sort.Strings(flags)
	pending := []int{}
	for g, v := range st.PendingGroups {
		if v {
			pending = append(pending, g)
		}
	}
	sort.Ints(pending)
	runtime5 := make([]int, 0, len(st.NativeRuntimeRecords))
	for _, r := range st.NativeRuntimeRecords {
		runtime5 = append(runtime5, int(r.Raw[5]), int(r.Raw[0x34]))
	}
	selectorErr := ""
	if st.NativeMapSelectorError != nil {
		selectorErr = st.NativeMapSelectorError.Error()
	}
	return obj{
		"units": units, "roster": roster, "turn": st.Turn, "round": st.NativeRoundCounter,
		"event_state": ints(st.NativeEventState[:]), "controls": controls, "flags": flags,
		"pending": pending, "field_raw": hex.EncodeToString(st.NativeFieldControlRaw),
		"runtime": runtime5, "selector_error": selectorErr,
	}
}

func dialogues(lines []battle.DialogLine) [][]any {
	out := make([][]any, 0, len(lines))
	for _, l := range lines {
		out = append(out, []any{l.Speaker, l.Text})
	}
	return out
}

type runner struct {
	rng     *rand.Rand
	st      *battle.State
	sc      *battle.Scenario
	pending []battle.Action
	ops     []opJSON
	rows    []byte
}

func (r *runner) intn(n int) int { return r.rng.IntN(n) }

func (r *runner) chance(pct int) bool { return r.rng.IntN(100) < pct }

func (r *runner) emit(op string, args obj, result obj) {
	if result == nil {
		result = obj{}
	}
	r.ops = append(r.ops, opJSON{Op: op, Args: args, Result: result})
}

func (r *runner) load(mapPath, scenarioPath string) bool {
	st, err := battle.Load(mapPath)
	if err != nil {
		r.emit("load", nil, obj{"err": err.Error()})
		return false
	}
	bindErr := st.BindNativeFutureItemRows(r.rows)
	r.st = st
	result := obj{"err": nil, "bind_err": errv(bindErr)}
	if scenarioPath != "" {
		sc, err := battle.LoadScenario(scenarioPath)
		if err != nil {
			result["err"] = err.Error()
			r.emit("load", nil, result)
			return false
		}
		r.sc = sc
		selectors := make([]int, len(sc.Events))
		for i := range sc.Events {
			selectors[i] = sc.Events[i].NativeTurnPhaseSelector()
		}
		result["scenario"] = mustJSON(sc)
		result["phase_selectors"] = selectors
	}
	result["digest"] = stateDigest(st)
	r.emit("load", nil, result)
	return true
}

func (r *runner) snapshot() {
	snap, err := r.st.Snapshot()
	if err != nil {
		r.emit("snapshot", nil, obj{"err": err.Error()})
		return
	}
	r.emit("snapshot", nil, obj{"err": nil, "snapshot": mustJSON(snap)})
}

func (r *runner) digestOp() { r.emit("digest", nil, obj{"digest": stateDigest(r.st)}) }

func (r *runner) execPending() {
	for i, action := range r.pending {
		result := obj{}
		if action.NativeWhen != nil {
			active, err := action.NativeWhen.Match(r.st)
			if err != nil {
				result["when_err"] = err.Error()
				r.emit("exec", obj{"index": i}, result)
				continue
			}
			if !active {
				result["skipped"] = true
				r.emit("exec", obj{"index": i}, result)
				continue
			}
		}
		if action.Type == "native_death_op" && action.NativeDeathOp != nil {
			result["death_err"] = errv(r.st.ApplyNativeDeathOp(*action.NativeDeathOp))
		} else {
			dl, isDialogue, err := r.sc.ExecuteActionChecked(r.st, action)
			result["speaker"], result["text"], result["is_dialogue"], result["err"] = dl.Speaker, dl.Text, isDialogue, errv(err)
		}
		r.emit("exec", obj{"index": i}, result)
	}
	r.pending = nil
}

func (r *runner) fieldCells() [][3]int {
	var cells [][3]int
	for y := 0; y < r.st.H; y++ {
		for x := 0; x < r.st.W; x++ {
			for sel := 0; sel < 3; sel++ {
				if id, ok := battle.NativeFieldEventIDAt(r.st, x, y, byte(sel)); ok {
					cells = append(cells, [3]int{x, y, sel*256 + int(id)})
				}
			}
		}
	}
	return cells
}

// fieldOp 對一個格子事件做一次操作；觸發者是 st.Units 裡的隨機單位。
func (r *runner) fieldOp(cells [][3]int) {
	if len(r.st.Units) == 0 {
		return
	}
	var x, y, sel, id int
	if len(cells) > 0 && r.chance(85) {
		c := cells[r.intn(len(cells))]
		x, y, sel, id = c[0], c[1], c[2]/256, c[2]%256
	} else {
		x, y, sel = r.intn(r.st.W+2)-1, r.intn(r.st.H+2)-1, r.intn(3)
	}
	kinds := []string{"id", "mode", "activation", "ch28", "e61", "e75"}
	kind := kinds[r.intn(len(kinds))]
	if r.chance(70) {
		switch id {
		case 61:
			kind = "e61"
		case 75:
			kind = "e75"
		case 62:
			kind = "activation"
		case 65, 69:
			kind = "ch28"
		default:
			kind = "mode"
		}
	}
	unit := r.intn(len(r.st.Units))
	if kind == "e61" || kind == "e75" {
		if kind == "e61" && r.chance(60) {
			items := []int{0xD0, 0x20}
			if r.chance(30) {
				items = []int{0x20}
			}
			r.setInventory(unit, items)
		}
		r.st.Units[unit].X, r.st.Units[unit].Y = x, y
		r.emit("move", obj{"unit": unit, "x": x, "y": y}, nil)
	}
	args := obj{"kind": kind, "x": x, "y": y, "selector": sel, "unit": unit}
	trigger := r.st.Units[unit]
	result := obj{}
	switch kind {
	case "id":
		got, ok := battle.NativeFieldEventIDAt(r.st, x, y, byte(sel))
		result["id"], result["ok"] = int(got), ok
	case "mode":
		got, ok := battle.ApplyNativeFieldModeEvent(r.st, trigger, x, y, byte(sel))
		result["id"], result["ok"] = int(got), ok
	case "activation":
		got, err := battle.ApplyNativeFieldTurnActivationEvent(r.st, x, y, byte(sel))
		result["id"], result["err"] = int(got), errv(err)
	case "ch28":
		got, err := battle.ApplyNativeChapter28FieldTurnActivation(r.st, trigger, x, y, byte(sel))
		result["id"], result["err"] = int(got), errv(err)
	case "e61":
		plan, err := battle.PlanNativeFieldEvent61(r.st, trigger, x, y)
		result["err"] = errv(err)
		if err == nil {
			result["plan"] = obj{"event_id": int(plan.EventID), "missing_item": plan.MissingItem,
				"text_index": plan.TextIndex, "final_text": plan.FinalText,
				"presentation": mustJSON(plan.Presentation), "join_character": plan.JoinCharacter}
			if !plan.MissingItem {
				frames := plan.Presentation.Frames
				if r.chance(20) {
					frames--
				}
				args["frames"] = frames
				joined, err := battle.CommitNativeFieldEvent61(r.st, plan, frames)
				result["joined"], result["commit_err"] = joined, errv(err)
			}
		}
	case "e75":
		plan, err := battle.PlanNativeFieldEvent75(r.st, trigger, x, y)
		result["err"] = errv(err)
		if err == nil {
			result["plan"] = obj{"event_id": int(plan.EventID), "text_index": plan.TextIndex,
				"activate": plan.Activate, "noop": plan.Noop}
			if plan.Activate {
				result["commit_err"] = errv(battle.CommitNativeFieldEvent75(r.st, plan))
			}
		}
	}
	r.emit("field", args, result)
}

func (r *runner) setInventory(unit int, items []int) {
	u := r.st.Units[unit]
	u.Inventory = append([]int(nil), items...)
	u.Equipped = make([]bool, len(items))
	u.InventorySlots = make([]int, 8)
	u.NativeInventoryFlags = make([]int, 8)
	for i := range u.InventorySlots {
		u.InventorySlots[i] = 0xff
		u.NativeInventoryFlags[i] = 0x80
	}
	for i, item := range items {
		u.InventorySlots[i] = item
		u.NativeInventoryFlags[i] = 0
	}
	r.emit("set_inventory", obj{"unit": unit, "items": items}, nil)
}

func (r *runner) resultOp() {
	out := obj{}
	code, err := battle.NativeDefaultResultCode(r.st)
	out["default"] = obj{"code": code, "err": errv(err)}
	code, err = battle.NativeInitialResultCode(r.st, r.sc)
	out["initial"] = obj{"code": code, "err": errv(err)}
	rules := []obj{}
	if r.sc != nil {
		for _, rule := range r.sc.NativeResultRules {
			matched, err := rule.Match(r.st)
			rules = append(rules, obj{"matched": matched, "err": errv(err)})
		}
		out["code1"] = r.sc.NativeResultCode1(r.st)
	}
	out["rules"] = rules
	r.emit("result", nil, out)
}

func recoveryList(st *battle.State, plan []battle.NativeEndTurnRecovery) [][]int {
	out := [][]int{}
	for _, rec := range plan {
		index := -1
		for i, u := range st.Units {
			if u == rec.Unit {
				index = i
			}
		}
		out = append(out, []int{index, rec.Before, rec.After, rec.MaxHP})
	}
	return out
}

func (r *runner) recovery() {
	plan := r.st.PlanNativeEndTurnRecovery()
	validate := r.st.ValidateNativeEndTurnRecovery(plan)
	applied := r.st.ApplyNativeEndTurnRecovery()
	r.emit("recovery", nil, obj{"plan": recoveryList(r.st, plan), "validate_err": errv(validate),
		"applied": recoveryList(r.st, applied)})
}

func (r *runner) markDead() {
	marked := r.st.MarkNativeDeadRecords()
	out := []int{}
	for _, m := range marked {
		for i, u := range r.st.Units {
			if u == m {
				out = append(out, i)
			}
		}
	}
	r.emit("mark_dead", nil, obj{"marked": out})
}

func (r *runner) turnEvents(camp int) {
	result := obj{}
	events, err := r.sc.NativeTurnEventsAt(r.st, byte(camp))
	ids := []int{}
	for _, e := range events {
		ids = append(ids, e.EventID)
	}
	result["events"], result["err"] = ids, errv(err)
	actionEvents := r.sc.NativeTurnActionEventsAt(r.st, byte(camp))
	ids = []int{}
	r.pending = nil
	for _, e := range actionEvents {
		ids = append(ids, e.EventID)
		r.pending = append(r.pending, e.Actions...)
	}
	result["action_events"], result["actions"] = ids, mustJSON(r.pending)
	r.emit("turn_events", obj{"camp": camp}, result)
	r.execPending()
}

func (r *runner) trigger(trigger, ctx string) {
	r.pending = r.sc.TriggerActions(r.st, trigger, ctx)
	r.emit("trigger", obj{"trigger": trigger, "ctx": ctx}, obj{"actions": mustJSON(r.pending)})
	r.execPending()
}

func (r *runner) nativeTurnProgressions() {
	for i, event := range r.sc.NativeTurnEvents {
		if event.Progression == nil && event.PairMutation == nil {
			continue
		}
		var slot int
		var eventID, camp byte
		if event.Progression != nil {
			slot, eventID, camp = event.Progression.ControlSlot, byte(event.EventID), byte(event.RawCamp)
		} else {
			slot, eventID, camp = event.PairMutation.ControlSlot, byte(event.EventID), byte(event.RawCamp)
		}
		if r.chance(60) && slot >= 0 && slot < 16 && r.st.NativeRoundCounter > 0 && r.st.NativeRoundCounter <= 0xfe {
			row := battle.NativeTurnEventControl{Turn: byte(r.st.NativeRoundCounter), EventID: eventID, RawCamp: camp}
			if r.chance(10) {
				row.Turn++
			}
			r.st.NativeTurnEventControls[slot] = row
			r.emit("set_control", obj{"slot": slot, "turn": int(row.Turn), "event_id": int(row.EventID), "raw_camp": int(row.RawCamp)}, nil)
		}
		if event.Progression != nil {
			plan, err := battle.PlanNativeTurnEvent76(r.st, event)
			result := obj{"final": plan.Final, "err": errv(err)}
			if err == nil && !plan.Final {
				result["commit_err"] = errv(battle.CommitNativeTurnEvent76Repeat(r.st, plan))
			}
			r.emit("e76", obj{"event": i}, result)
		} else {
			rng := r.intn(0x10000)
			next, targets, err := battle.ApplyNativeTurnEvent79(r.st, event, uint16(rng))
			r.emit("e79", obj{"event": i, "rng": rng}, obj{"next": int(next), "targets": targets[:], "err": errv(err)})
		}
	}
}

func (r *runner) killSome() []string {
	var names []string
	n := r.intn(3)
	for k := 0; k < n && len(r.st.Units) > 0; k++ {
		unit := r.intn(len(r.st.Units))
		u := r.st.Units[unit]
		if u.HP == 0 || !u.OnField {
			continue
		}
		u.HP = 0
		names = append(names, u.Name)
		r.emit("kill", obj{"unit": unit}, obj{"name": u.Name})
	}
	// 回合末回復（0x1A30B）只對未滿血、未行動的我方生效：另外隨機扣血與清掉行動位。
	for k := r.intn(4); k > 0 && len(r.st.Units) > 0; k-- {
		unit := r.intn(len(r.st.Units))
		u := r.st.Units[unit]
		if u.MaxHP <= 1 || u.HP == 0 {
			continue
		}
		hp := 1 + r.intn(u.MaxHP-1)
		clear := r.chance(50)
		u.HP = hp
		if clear {
			u.Acted = false
			u.NativeRecordByte5 &^= 0x80
		}
		r.emit("damage", obj{"unit": unit, "hp": hp, "clear_acted": clear}, nil)
	}
	return names
}

// liveNativeTurnRows 隨機把劇本 native_turn_events 對應的控制列排到本回合，讓 NativeTurnEventsAt／
// NativeTurnActionEventsAt 與 event76／79 走到可分派的分支。
func (r *runner) liveNativeTurnRows() {
	if r.st.NativeRoundCounter <= 0 || r.st.NativeRoundCounter > 0xfe {
		return
	}
	for _, event := range r.sc.NativeTurnEvents {
		if !r.chance(35) {
			continue
		}
		for slot, row := range r.st.NativeTurnEventControls {
			if int(row.EventID) != event.EventID || int(row.RawCamp) != event.RawCamp {
				continue
			}
			row.Turn = byte(r.st.NativeRoundCounter)
			r.st.NativeTurnEventControls[slot] = row
			r.emit("set_control", obj{"slot": slot, "turn": int(row.Turn), "event_id": int(row.EventID), "raw_camp": int(row.RawCamp)}, nil)
			break
		}
		if event.PairMutation != nil && r.chance(70) {
			// group1 原版由 event76 的 final 分支（GUI 擁有）登場；這裡以 AppendGroup 先放出來，
			// 讓 event79 能走到成功分支。
			if r.chance(50) {
				n := r.st.AppendGroup(event.PairMutation.Group)
				r.emit("append_group", obj{"group": event.PairMutation.Group}, obj{"appended": n})
			}
			base := -1
			for i, u := range r.st.Units {
				if u.Group == event.PairMutation.Group {
					base = i
					break
				}
			}
			if base >= 0 && base <= 0xff {
				index := event.PairMutation.BaseStateIndex
				r.st.NativeEventState[index] = byte(base)
				r.emit("set_state", obj{"index": index, "value": base}, nil)
			}
		}
	}
}

func (r *runner) firedOp() {
	refs := r.sc.FiredEvents()
	if refs == nil {
		refs = []battle.FiredEventRef{}
	}
	r.emit("fired", nil, obj{"refs": mustJSON(refs), "pending_joins": r.sc.PendingJoinCount()})
}

func (r *runner) scenarioScript(name, scenarioPath, mapPath string) scriptJSON {
	if !r.load(mapPath, scenarioPath) {
		return scriptJSON{Name: name, Kind: "scenario", Scenario: scenarioPath, Map: mapPath, Ops: r.ops}
	}
	dl, err := r.sc.SetupChecked(r.st)
	r.emit("setup", nil, obj{"dialogues": dialogues(dl), "err": errv(err), "digest": stateDigest(r.st)})
	r.snapshot()
	r.firedOp()
	cells := r.fieldCells()
	r.emit("field_cells", nil, obj{"cells": cells})
	for turn := 1; turn <= 14; turn++ {
		r.st.Turn, r.st.NativeRoundCounter = turn, turn
		r.emit("turn", obj{"turn": turn}, nil)
		killed := r.killSome()
		r.markDead()
		if r.chance(25) {
			index, value := 16+r.intn(8), r.intn(6)
			r.st.NativeEventState[index] = byte(value)
			r.emit("set_state", obj{"index": index, "value": value}, nil)
		}
		r.liveNativeTurnRows()
		for camp := 0; camp < 3; camp++ {
			r.turnEvents(camp)
		}
		r.trigger("on_turn_end", "")
		for _, name := range killed {
			if r.chance(50) {
				r.trigger("on_unit_death", name)
			} else {
				dl := r.sc.Fire(r.st, "on_unit_death", name)
				r.emit("fire", obj{"trigger": "on_unit_death", "ctx": name}, obj{"dialogues": dialogues(dl)})
			}
		}
		r.nativeTurnProgressions()
		for k := r.intn(3); k > 0; k-- {
			r.fieldOp(cells)
		}
		r.resultOp()
		r.recovery()
		r.emit("joins", nil, obj{"joins": r.sc.TakePartyJoins()})
		r.digestOp()
		if turn == 7 {
			r.snapshot()
		}
	}
	r.firedOp()
	r.snapshot()
	return scriptJSON{Name: name, Kind: "scenario", Scenario: scenarioPath, Map: mapPath, Ops: r.ops}
}

func (r *runner) mapScript(name, mapPath string) scriptJSON {
	if !r.load(mapPath, "") {
		return scriptJSON{Name: name, Kind: "map", Map: mapPath, Ops: r.ops}
	}
	r.snapshot()
	cells := r.fieldCells()
	r.emit("field_cells", nil, obj{"cells": cells})
	r.resultOp()
	r.recovery()
	for k := 0; k < 8; k++ {
		r.st.NativeRoundCounter = 1 + r.intn(10)
		r.emit("set_round", obj{"round": r.st.NativeRoundCounter}, nil)
		if r.chance(30) {
			index, value := 16+r.intn(8), r.intn(3)
			r.st.NativeEventState[index] = byte(value)
			r.emit("set_state", obj{"index": index, "value": value}, nil)
		}
		r.fieldOp(cells)
		r.markDead()
	}
	r.digestOp()
	r.snapshot()
	return scriptJSON{Name: name, Kind: "map", Map: mapPath, Ops: r.ops}
}

func (r *runner) adoptScript(name, scenarioPath, mapPath string) scriptJSON {
	if !r.load(mapPath, scenarioPath) {
		return scriptJSON{Name: name, Kind: "adopt", Scenario: scenarioPath, Map: mapPath, Ops: r.ops}
	}
	r.st.Roster = r.st.Units
	r.st.Units = nil
	r.st.AppendNativeMapSelectorBatchOrLegacy(r.sc.PartyUnits(r.st.OwnDeploy))
	r.emit("handler_prepare", nil, obj{"digest": stateDigest(r.st)})
	err := r.sc.AdoptHandlerBattleState(r.st)
	r.emit("adopt", nil, obj{"err": errv(err), "digest": stateDigest(r.st)})
	dl := r.sc.Fire(r.st, "on_battle_start", "")
	r.emit("fire", obj{"trigger": "on_battle_start", "ctx": ""}, obj{"dialogues": dialogues(dl)})
	r.firedOp()
	for turn := 1; turn <= 4; turn++ {
		r.st.Turn, r.st.NativeRoundCounter = turn, turn
		r.emit("turn", obj{"turn": turn}, nil)
		r.trigger("on_turn_end", "")
		r.digestOp()
	}
	r.snapshot()
	return scriptJSON{Name: name, Kind: "adopt", Scenario: scenarioPath, Map: mapPath, Ops: r.ops}
}

// ---- 合成案例 ----

type exporter struct {
	rng   *rand.Rand
	cases []caseJSON
}

func (e *exporter) add(op string, input, result obj) {
	e.cases = append(e.cases, caseJSON{Index: len(e.cases), Op: op, Input: input, Result: result})
}

// mutateJSON 在 JSON 樹上隨機挑一個節點：數字改成隨機整數、字串改成另一個字串、刪掉物件的鍵、
// 或刪掉陣列元素。不寫入 null 純量（Go 讀 null 到 int 不報錯、Rust 會拒收，列在 KNOWN_DIFFERENCES）。
func (e *exporter) mutateJSON(v any, depth int) any {
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 0 {
			return t
		}
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		k := keys[e.rng.IntN(len(keys))]
		if e.rng.IntN(100) < 15 {
			delete(t, k)
			return t
		}
		t[k] = e.mutateJSON(t[k], depth+1)
		return t
	case []any:
		if len(t) == 0 {
			return t
		}
		i := e.rng.IntN(len(t))
		if e.rng.IntN(100) < 10 {
			return append(t[:i:i], t[i+1:]...)
		}
		t[i] = e.mutateJSON(t[i], depth+1)
		return t
	case float64:
		choices := []float64{-1, 0, 1, 2, 3, 15, 16, 31, 32, 89, 90, 105, 106, 254, 255, 256}
		return choices[e.rng.IntN(len(choices))]
	case string:
		choices := []string{"", "spawn_group", "spawn_group_with_intro", "native_acting", "set_native_event_state",
			"dialogue", "0x20b72", "0x20bf5", "x", "record_byte6_nonzero"}
		return choices[e.rng.IntN(len(choices))]
	case bool:
		return !t
	}
	return v
}

func (e *exporter) genLoadScenario(paths []string, tmp string) {
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		for k := 0; k < 12; k++ {
			var tree any
			if err := json.Unmarshal(raw, &tree); err != nil {
				panic(err)
			}
			steps := 1 + e.rng.IntN(5)
			for s := 0; s < steps; s++ {
				tree = e.mutateJSON(tree, 0)
			}
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			enc.SetEscapeHTML(false)
			if err := enc.Encode(tree); err != nil {
				panic(err)
			}
			text := buf.String()
			var probe battle.Scenario
			decodeErr := json.Unmarshal([]byte(text), &probe) != nil
			file := filepath.Join(tmp, "scenario.json")
			if err := os.WriteFile(file, []byte(text), 0o644); err != nil {
				panic(err)
			}
			sc, err := battle.LoadScenario(file)
			result := obj{"decode_error": decodeErr, "err": errv(err)}
			if err == nil {
				result["scenario"] = mustJSON(sc)
			}
			e.add("load_scenario", obj{"source": path, "json": text}, result)
		}
	}
}

func (e *exporter) genRestore() {
	base := &battle.State{W: 3, H: 2, Turn: 2, NativeRoundCounter: 2,
		Units:  []*battle.Unit{{Name: "a", HP: 3, X: 1}, {Name: "b", HP: 4, Y: 1}},
		Roster: []*battle.Unit{{Name: "r", Group: 2}},
		Cost:   []int{1, 1, 1, 2, 2, 2}, Flags: map[string]bool{"f": true},
		Treasures:      map[battle.Cell]battle.Treasure{{X: 2, Y: 1}: {Slot: 1, Kind: "item", Value: 7}, {X: 0, Y: 0}: {Slot: 0}},
		OpenedTreasure: map[int]bool{},
	}
	snap, err := base.Snapshot()
	if err != nil {
		panic(err)
	}
	good := mustJSON(snap)
	mutations := []func(m map[string]any){
		func(m map[string]any) {},
		func(m map[string]any) { m["version"] = float64(e.rng.IntN(5) + 2) },
		func(m map[string]any) { m["version"] = float64(0) },
		func(m map[string]any) { m["w"] = float64(0) },
		func(m map[string]any) { m["h"] = float64(-1) },
		func(m map[string]any) { m["cost"] = []any{float64(1), float64(2)} },
		func(m map[string]any) { m["cost"] = []any{} },
		func(m map[string]any) { m["cost"] = nil },
		func(m map[string]any) { m["units_constructor_pending"] = []any{true} },
		func(m map[string]any) { m["roster_constructor_pending"] = []any{false, true} },
		func(m map[string]any) {
			m["has_native_map_selector_cache"] = true
			m["native_map_selector_keys"] = "AQE="
		},
		func(m map[string]any) {
			m["has_native_map_selector_cache"] = true
			m["native_map_selector_keys"] = "BwMJ"
		},
		func(m map[string]any) {
			list := m["treasures"].([]any)
			m["treasures"] = append(list, list[0])
		},
		func(m map[string]any) { m["treasures"] = nil },
		func(m map[string]any) { m["treasures"] = []any{} },
		func(m map[string]any) { m["native_map_selector_error"] = "boom" },
	}
	for i, mutate := range mutations {
		var m map[string]any
		if err := json.Unmarshal(good, &m); err != nil {
			panic(err)
		}
		mutate(m)
		text := string(mustJSON(m))
		var decoded battle.StateSnapshot
		if err := json.Unmarshal([]byte(text), &decoded); err != nil {
			panic(err)
		}
		dst := &battle.State{Turn: 99, W: 5}
		err := dst.RestoreSnapshot(decoded)
		result := obj{"err": errv(err), "digest": stateDigest(dst)}
		if err == nil {
			again, err := dst.Snapshot()
			if err != nil {
				panic(err)
			}
			result["snapshot"] = mustJSON(again)
		}
		e.add("restore", obj{"mutation": i, "snapshot": text}, result)
	}
}

func (e *exporter) genRestoreFired(paths []string) {
	for _, path := range paths {
		sc, err := battle.LoadScenario(path)
		if err != nil {
			panic(err)
		}
		for k := 0; k < 3; k++ {
			refs := []battle.FiredEventRef{}
			for i := range sc.Events {
				if e.rng.IntN(100) < 30 {
					refs = append(refs, battle.FiredEventRef{Index: i, ID: sc.Events[i].ID})
				}
			}
			switch e.rng.IntN(4) {
			case 1:
				refs = append(refs, battle.FiredEventRef{Index: len(sc.Events) + e.rng.IntN(3), ID: "x"})
			case 2:
				if len(refs) > 0 {
					refs = append(refs, refs[0])
				}
			case 3:
				if len(refs) > 0 {
					refs[0].ID += "\"q\"\n"
				}
			}
			fresh, _ := battle.LoadScenario(path)
			err := fresh.RestoreFiredEvents(refs)
			after := fresh.FiredEvents()
			if after == nil {
				after = []battle.FiredEventRef{}
			}
			e.add("restore_fired", obj{"scenario": path, "refs": mustJSON(refs)},
				obj{"err": errv(err), "fired": mustJSON(after)})
		}
	}
}

// genMarchCommit 以合成單位測 CommitNativeSystemGroupMarchStep（不經 D 組尋路的發布半段）。
func (e *exporter) genMarchCommit(n int) {
	for k := 0; k < n; k++ {
		units := []obj{}
		st := &battle.State{W: 6, H: 6}
		for u := 0; u < 1+e.rng.IntN(3); u++ {
			x, y := e.rng.IntN(6), e.rng.IntN(6)
			unit := &battle.Unit{X: x, Y: y,
				NativeMapPresentation:    battle.NativeMapPresentationState{X: byte(x), Y: byte(y), Pose: byte(e.rng.IntN(4))},
				HasNativeMapPresentation: e.rng.IntN(100) < 90,
				NativeRecordByte5:        []byte{0, 0, 0, 0x80, 0x01, 0x04}[e.rng.IntN(6)],
				HasNativeRecordByte5:     e.rng.IntN(100) < 90,
				NativeRecordByte6:        []byte{2, 2, 2, 0, 1}[e.rng.IntN(5)],
				HasNativeRecordByte6:     e.rng.IntN(100) < 90,
			}
			if e.rng.IntN(100) < 20 {
				unit.NativeMapPresentation.X++
			}
			st.Units = append(st.Units, unit)
			units = append(units, obj{"x": unit.X, "y": unit.Y, "px": int(unit.NativeMapPresentation.X),
				"py": int(unit.NativeMapPresentation.Y), "pose": int(unit.NativeMapPresentation.Pose),
				"has_presentation": unit.HasNativeMapPresentation, "byte5": int(unit.NativeRecordByte5),
				"has5": unit.HasNativeRecordByte5, "byte6": int(unit.NativeRecordByte6), "has6": unit.HasNativeRecordByte6})
		}
		index := e.rng.IntN(len(st.Units)+1) - e.rng.IntN(2)
		path := []battle.Cell{}
		pathJSON := [][]int{}
		steps := e.rng.IntN(4)
		cur := battle.Cell{X: e.rng.IntN(6), Y: e.rng.IntN(6)}
		if index >= 0 && index < len(st.Units) && e.rng.IntN(100) < 70 {
			cur = battle.Cell{X: st.Units[index].X, Y: st.Units[index].Y}
		}
		for s := 0; s < steps; s++ {
			path = append(path, cur)
			pathJSON = append(pathJSON, []int{cur.X, cur.Y})
			cur = battle.Cell{X: e.rng.IntN(6), Y: e.rng.IntN(6)}
		}
		err := st.CommitNativeSystemGroupMarchStep(battle.NativeSystemGroupMarchStep{UnitIndex: index, Path: path})
		after := []obj{}
		for _, u := range st.Units {
			after = append(after, obj{"x": u.X, "y": u.Y, "px": int(u.NativeMapPresentation.X),
				"py": int(u.NativeMapPresentation.Y), "pose": int(u.NativeMapPresentation.Pose),
				"byte5": int(u.NativeRecordByte5), "acted": u.Acted})
		}
		e.add("march_commit", obj{"units": units, "index": index, "path": pathJSON},
			obj{"err": errv(err), "units": after})
	}
}

// ---- 輸入清單 ----

func recordFiles(paths []string) []obj {
	files := []obj{}
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		files = append(files, obj{"path": path, "size": len(raw), "sha256": digest(raw)})
	}
	return files
}

// normalize 與 Rust 端相同：空陣列、空物件、空字串都視為 null；整數值的浮點數視為整數。
func normalize(v any) any {
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 0 {
			return nil
		}
		out := map[string]any{}
		for k, x := range t {
			out[k] = normalize(x)
		}
		return out
	case []any:
		if len(t) == 0 {
			return nil
		}
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = normalize(x)
		}
		return out
	case string:
		if t == "" {
			return nil
		}
	}
	return v
}

func parseNormalized(raw []byte) any {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		panic(err)
	}
	return normalize(v)
}

// verify 讀 Rust 寫出的快照：每份都 Unmarshal 成 StateSnapshot、RestoreSnapshot 到新的 State、
// 再 Snapshot 與 Marshal，正規化後要與黃金裡同一步的 Go 快照相同。
func verify(goldenPath, rustPath string) {
	var golden struct {
		Scripts []scriptJSON `json:"scripts"`
	}
	raw, err := os.ReadFile(goldenPath)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(raw, &golden); err != nil {
		panic(err)
	}
	var rust struct {
		Snapshots []struct {
			Script int             `json:"script"`
			Op     int             `json:"op"`
			Read   json.RawMessage `json:"reread"`
			Own    json.RawMessage `json:"own"`
		} `json:"snapshots"`
	}
	raw, err = os.ReadFile(rustPath)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(raw, &rust); err != nil {
		panic(err)
	}
	fails := 0
	for _, s := range rust.Snapshots {
		want := golden.Scripts[s.Script].Ops[s.Op].Result["snapshot"]
		wantRaw := mustJSON(want)
		for label, text := range map[string]json.RawMessage{"reread": s.Read, "own": s.Own} {
			var snap battle.StateSnapshot
			if err := json.Unmarshal(text, &snap); err != nil {
				fmt.Printf("script %d op %d %s: Go 讀不進 Rust 快照：%v\n", s.Script, s.Op, label, err)
				fails++
				continue
			}
			st := &battle.State{}
			if err := st.RestoreSnapshot(snap); err != nil {
				fmt.Printf("script %d op %d %s: RestoreSnapshot：%v\n", s.Script, s.Op, label, err)
				fails++
				continue
			}
			again, err := st.Snapshot()
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(parseNormalized(mustJSON(again)), parseNormalized(wantRaw)) {
				fmt.Printf("script %d op %d %s: Go 還原 Rust 快照後與黃金不同\n", s.Script, s.Op, label)
				fails++
			}
		}
	}
	fmt.Printf("verify: Rust 快照 %d 份，失敗 %d\n", len(rust.Snapshots), fails)
	if fails != 0 || len(rust.Snapshots) == 0 {
		os.Exit(1)
	}
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	verifyRust := flag.String("verify-rust", "", "Rust 寫出的快照檔（-verify 模式）")
	golden := flag.String("golden", "", "-verify 模式要比對的黃金檔")
	flag.Parse()
	if *verifyRust != "" {
		verify(*golden, *verifyRust)
		return
	}
	if *out == "" {
		fmt.Fprintln(os.Stderr, "需要 -out")
		os.Exit(2)
	}
	tmp, err := os.MkdirTemp("", "fd2-golden-rules-f-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	rows, err := battle.LoadNativeItemEffectRowPrefix(itemRowsPath)
	if err != nil {
		panic(err)
	}
	scenarios, err := filepath.Glob("assets/scenarios/ch*.json")
	if err != nil || len(scenarios) == 0 {
		panic("找不到 assets/scenarios/ch*.json")
	}
	sort.Strings(scenarios)
	maps, err := filepath.Glob("assets/maps/map*/map*_units.json")
	if err != nil || len(maps) == 0 {
		panic("找不到 assets/maps/map*/map*_units.json")
	}
	sort.Strings(maps)
	inputs := append([]string{itemRowsPath, "assets/maps/native_turn_event_controls.json"}, scenarios...)
	for _, path := range maps {
		inputs = append(inputs, path)
		if _, err := os.Stat(filepath.Join(filepath.Dir(path), "map.json")); err == nil {
			inputs = append(inputs, filepath.Join(filepath.Dir(path), "map.json"))
		}
	}
	rng := rand.New(rand.NewPCG(*seed, 0x52554c46))
	scripts := []scriptJSON{}
	newRunner := func() *runner { return &runner{rng: rng, rows: rows} }
	for _, path := range scenarios {
		probe, err := battle.LoadScenario(path)
		if err != nil {
			panic(fmt.Sprintf("%s: %v", path, err))
		}
		mapPath := fmt.Sprintf("assets/maps/map%d/map%d_units.json", probe.Map, probe.Map)
		name := strings.TrimSuffix(filepath.Base(path), ".json")
		scripts = append(scripts, newRunner().scenarioScript(name, path, mapPath))
		if probe.RuntimeAppendGroups {
			scripts = append(scripts, newRunner().adoptScript(name+"-adopt", path, mapPath))
		}
	}
	for _, path := range maps {
		name := strings.TrimSuffix(filepath.Base(path), "_units.json")
		scripts = append(scripts, newRunner().mapScript(name, path))
	}
	e := &exporter{rng: rng}
	e.genLoadScenario(scenarios, tmp)
	e.genRestore()
	e.genRestoreFired(scenarios)
	e.genMarchCommit(60)
	doc := obj{"schema": "fd2-golden-rules-f/1", "seed": *seed, "inputs": recordFiles(inputs),
		"scripts": scripts, "cases": e.cases}
	data, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		panic(err)
	}
	counts := map[string]int{}
	for _, s := range scripts {
		for _, op := range s.Ops {
			counts["op/"+op.Op]++
		}
	}
	for _, c := range e.cases {
		counts["case/"+c.Op]++
	}
	names := []string{}
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	total := 0
	for _, name := range names {
		fmt.Printf("%s=%d ", name, counts[name])
		total += counts[name]
	}
	fmt.Printf("scripts=%d total=%d\n", len(scripts), total)
}
