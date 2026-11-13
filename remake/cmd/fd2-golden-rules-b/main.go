// fd2-golden-rules-b 輸出 internal/battle「B 組」（指令、目標與死亡程式）的黃金差分資料，
// 供 Rust 改寫 remake-rs/crates/fd2-rules 逐欄比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 battle 的公開 API，不改任何遊戲行為。
//   - 真實輸入：remake/assets/maps/map*/map*_units.json 全部以 Load＋BindNativeFutureItemRows 載入
//     （路徑排序後保存清單與 SHA-256），每張地圖輸出兩份戰場底稿：原樣單位（legacy）與補齊原版
//     provenance 的單位（raw）。spells.json、native_command_resistances.json、
//     native_movement_cost_rows.json、native_item_effect_rows.json 也記路徑與 SHA-256。
//   - 產生輸入：以固定種子的 PCG 產生合成戰場、單位修改、指令參數、原版亂數字組與 math/rand 種子，
//     連同結果寫進同一份 JSON；Rust 端不需重現產生器，只重做同一串呼叫。
//
// 戰場一律由「底稿＋修改」組成：底稿單位是 json.Marshal(battle.Unit) 的物件，修改是逐單位合併的
// JSON 鍵（Go 欄位名）。Go 與 Rust 都把合併後的物件解碼成 Unit，因此兩邊看到相同的單位，未匯出
// 欄位兩邊都是零值。結果裡的 *Unit 一律寫成 State.Units 的索引（nil 為 -1），結構以 Go 欄位名
// 輸出（未匯出欄位省略）。錯誤欄位是 Go err.Error() 原文。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	mrand "math/rand"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type obj = map[string]any

type caseJSON struct {
	Index  int    `json:"index"`
	Op     string `json:"op"`
	Input  obj    `json:"input"`
	Result obj    `json:"result"`
}

type fixture struct {
	Name   string
	W, H   int
	Events []byte
	Cost   []int
	Units  []obj
}

type exporter struct {
	rng      *rand.Rand
	tmp      string
	cases    []caseJSON
	fixtures []fixture
	book     []battle.NativeCommandRecord
	resist   map[int]int
	costRows [][]byte
	itemRows []byte
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

// rangeIn 回傳 [lo, hi] 的整數。
func (e *exporter) rangeIn(lo, hi int) int { return lo + e.rng.IntN(hi-lo+1) }

func (e *exporter) chance(pct int) bool { return e.rng.IntN(100) < pct }

func (e *exporter) bytesN(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(e.rng.IntN(256))
	}
	return b
}

func (e *exporter) u16() int { return e.rng.IntN(0x10000) }

func (e *exporter) seed() int64 { return int64(e.rng.Uint64()>>1) - int64(1<<61) }

func ints(b []byte) []int {
	out := make([]int, len(b))
	for i, v := range b {
		out[i] = int(v)
	}
	return out
}

// ---- 結果轉換：*Unit → 索引，結構 → Go 欄位名物件 ----

var unitPtrType = reflect.TypeOf((*battle.Unit)(nil))

func conv(units []*battle.Unit, v any) any { return convValue(units, reflect.ValueOf(v)) }

func convValue(units []*battle.Unit, v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	if v.Type() == unitPtrType {
		if v.IsNil() {
			return -1
		}
		p := v.Interface().(*battle.Unit)
		for i, u := range units {
			if u == p {
				return i
			}
		}
		return -2
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return convValue(units, v.Elem())
	case reflect.Struct:
		out := obj{}
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			out[f.Name] = convValue(units, v.Field(i))
		}
		return out
	case reflect.Slice, reflect.Array:
		arr := make([]any, v.Len())
		for i := range arr {
			arr[i] = convValue(units, v.Index(i))
		}
		return arr
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.String:
		return v.String()
	}
	panic("conv: 未支援的型別 " + v.Type().String())
}

// ---- 底稿與戰場組裝 ----

func decodeObj(data []byte) obj {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var out obj
	if err := dec.Decode(&out); err != nil {
		panic(err)
	}
	return out
}

func unitObj(u *battle.Unit) obj {
	data, err := json.Marshal(u)
	if err != nil {
		panic(err)
	}
	return decodeObj(data)
}

func cloneObj(o obj) obj {
	data, err := json.Marshal(o)
	if err != nil {
		panic(err)
	}
	return decodeObj(data)
}

func num(v any) int {
	switch x := v.(type) {
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			panic(err)
		}
		return int(n)
	case int:
		return x
	case float64:
		return int(x)
	}
	panic(fmt.Sprintf("num: %T", v))
}

func boolOf(v any) bool {
	b, _ := v.(bool)
	return b
}

func (e *exporter) buildUnits(sp obj) []*battle.Unit {
	fx := e.fixtures[sp["fixture"].(int)]
	maps := make([]obj, len(fx.Units))
	for i, u := range fx.Units {
		maps[i] = cloneObj(u)
	}
	edits, _ := sp["edits"].([]obj)
	for _, ed := range edits {
		idx := ed["unit"].(int)
		for key, value := range ed["set"].(obj) {
			maps[idx][key] = value
		}
	}
	units := make([]*battle.Unit, len(maps))
	for i, m := range maps {
		data, err := json.Marshal(m)
		if err != nil {
			panic(err)
		}
		var u battle.Unit
		if err := json.Unmarshal(data, &u); err != nil {
			panic(err)
		}
		units[i] = &u
	}
	return units
}

func balanceFrom(v any) *battle.BalanceOptions {
	b, ok := v.(obj)
	if !ok {
		return nil
	}
	g := b["growth"].([]int)
	opts := &battle.BalanceOptions{
		Growth:       battle.GrowthRates{HP: g[0], MP: g[1], AP: g[2], DP: g[3], DX: g[4]},
		EnemyPercent: b["enemy_percent"].(int),
	}
	for _, row := range b["characters"].([][]int) {
		if opts.Characters == nil {
			opts.Characters = map[int]battle.GrowthRates{}
		}
		opts.Characters[row[0]] = battle.GrowthRates{HP: row[1], MP: row[2], AP: row[3], DP: row[4], DX: row[5]}
	}
	for _, row := range b["skill_hit"].([][]int) {
		if opts.SkillHit == nil {
			opts.SkillHit = map[int]map[int]int{}
		}
		if opts.SkillHit[row[0]] == nil {
			opts.SkillHit[row[0]] = map[int]int{}
		}
		opts.SkillHit[row[0]][row[1]] = row[2]
	}
	return opts
}

func (e *exporter) buildBook(sp obj) []battle.NativeCommandRecord {
	book := append([]battle.NativeCommandRecord(nil), e.book...)
	if n, ok := sp["book_len"].(int); ok {
		for len(book) < n {
			book = append(book, battle.NativeCommandRecord{ID: len(book)})
		}
		book = book[:n]
	}
	if edits, ok := sp["book_edits"].([][3]any); ok {
		for _, ed := range edits {
			id, field, value := ed[0].(int), ed[1].(string), ed[2].(int)
			if id >= len(book) {
				continue
			}
			r := &book[id]
			switch field {
			case "ID":
				r.ID = value
			case "Damage":
				r.Damage = value
			case "Hit":
				r.Hit = value
			case "SelectionMode":
				r.SelectionMode = value
			case "EffectMode":
				r.EffectMode = value
			case "MPCost":
				r.MPCost = value
			case "TargetCode":
				r.TargetCode = value
			}
		}
	}
	return book
}

func (e *exporter) buildState(sp obj) *battle.State {
	fx := e.fixtures[sp["fixture"].(int)]
	st := &battle.State{
		W: fx.W, H: fx.H, Units: e.buildUnits(sp),
		NativeCompositionEventBytes: append([]byte(nil), fx.Events...),
		NativeCommandBook:           e.buildBook(sp),
		NativeCommandResistances:    map[int]int{},
		Balance:                     balanceFrom(sp["balance"]),
	}
	if fx.Cost != nil {
		st.Cost = append([]int(nil), fx.Cost...)
	}
	for k, v := range e.resist {
		st.NativeCommandResistances[k] = v
	}
	if drop, ok := sp["resist_drop"].([]int); ok {
		for _, k := range drop {
			delete(st.NativeCommandResistances, k)
		}
	}
	if set, ok := sp["resist_set"].([][2]int); ok {
		for _, kv := range set {
			st.NativeCommandResistances[kv[0]] = kv[1]
		}
	}
	return st
}

func snapshot(units []*battle.Unit) [][]byte {
	out := make([][]byte, len(units))
	for i, u := range units {
		data, err := json.Marshal(u)
		if err != nil {
			panic(err)
		}
		out[i] = data
	}
	return out
}

// changed 回傳與 before 不同的單位（鍵是索引字串，值是 Go JSON 物件）。
func changed(before [][]byte, units []*battle.Unit) obj {
	out := obj{}
	for i, u := range units {
		data, err := json.Marshal(u)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(data, before[i]) {
			out[fmt.Sprint(i)] = json.RawMessage(data)
		}
	}
	return out
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
	for _, p := range []string{"assets/spells.json", "assets/data/native_command_resistances.json",
		"assets/data/native_movement_cost_rows.json", "assets/data/native_item_effect_rows.json",
		"assets/maps/native_turn_event_controls.json"} {
		record(p)
	}
	if e.book, err = battle.LoadNativeCommandRecords("assets/spells.json"); err != nil {
		panic(err)
	}
	if e.resist, err = battle.LoadNativeCommandResistances("assets/data/native_command_resistances.json"); err != nil {
		panic(err)
	}
	if e.costRows, err = battle.LoadNativeMovementCostRows("assets/data/native_movement_cost_rows.json"); err != nil {
		panic(err)
	}
	if e.itemRows, err = battle.LoadNativeItemEffectRowPrefix("assets/data/native_item_effect_rows.json"); err != nil {
		panic(err)
	}
	for _, path := range paths {
		record(path)
		mapPath := filepath.Join(filepath.Dir(path), "map.json")
		if _, err := os.Stat(mapPath); err == nil {
			record(mapPath)
		}
		st, err := battle.Load(path)
		if err != nil {
			panic(fmt.Sprintf("%s: %v", path, err))
		}
		_ = st.BindNativeFutureItemRows(e.itemRows)
		legacy := fixture{Name: path + "#legacy", W: st.W, H: st.H, Events: st.NativeCompositionEventBytes, Cost: st.Cost}
		raw := fixture{Name: path + "#raw", W: st.W, H: st.H, Events: st.NativeCompositionEventBytes, Cost: st.Cost}
		for _, u := range st.Units {
			legacy.Units = append(legacy.Units, unitObj(u))
			raw.Units = append(raw.Units, e.rawify(unitObj(u)))
		}
		e.fixtures = append(e.fixtures, legacy, raw)
	}
	resist := [][2]int{}
	for k, v := range e.resist {
		resist = append(resist, [2]int{k, v})
	}
	sort.Slice(resist, func(i, j int) bool { return resist[i][0] < resist[j][0] })
	rows := []string{}
	for _, r := range e.costRows {
		rows = append(rows, hexs(r))
	}
	return obj{"files": files, "maps": paths, "book": conv(nil, e.book), "resistances": resist,
		"cost_rows": rows, "item_rows": hexs(e.itemRows)}
}

// rawify 補齊目標判定所需的原版 provenance：+5、+6、+0x1F/+0x20 職業、+7 選擇子與地圖呈現座標。
func (e *exporter) rawify(u obj) obj {
	camp := num(u["Camp"])
	hp := num(u["HP"])
	onField := boolOf(u["OnField"])
	b5 := 0
	if !onField || hp <= 0 {
		b5 = 1
	}
	if e.chance(10) {
		b5 |= 0x80
	}
	u["native_record_byte5"], u["has_native_record_byte5"] = b5, true
	u["native_record_byte6"], u["has_native_record_byte6"] = []int{2, 1, 0}[camp%3], true
	u["NativeMapPresentation"] = obj{"X": num(u["X"]) & 0xff, "Y": num(u["Y"]) & 0xff, "Pose": 0, "Motion": 0}
	u["HasNativeMapPresentation"] = true
	cls := num(u["ClassID"]) & 0xff
	if e.chance(15) {
		cls = 19
	}
	u["NativeRecordClass"], u["HasNativeRecordClass"] = cls, true
	u["BattleFig"], u["HasBattleFig"] = []int{4, 5, 6, 7, 20, 3, 30}[e.intn(7)], true
	for _, key := range []string{"native_record_byte34", "native_record_byte35", "native_record_byte36"} {
		if !boolOf(u["has_"+key]) {
			u[key] = e.intn(16)
		}
		u["has_"+key] = true
	}
	u["native_record_word42"], u["has_native_record_word42"] = num(u["MaxHP"])&0xffff, true
	u["native_record_word46"], u["has_native_record_word46"] = num(u["MaxMP"])&0xffff, true
	return u
}

// ---- 合成戰場 ----

func (e *exporter) genSyntheticFixtures(n int) {
	for k := 0; k < n; k++ {
		w, h := e.rangeIn(2, 12), e.rangeIn(1, 10)
		if k%5 == 0 {
			w, h = e.rangeIn(13, 16), e.rangeIn(8, 11)
		}
		fx := fixture{Name: fmt.Sprintf("synthetic-%d", k), W: w, H: h, Events: e.bytesN(w * h)}
		if e.chance(50) {
			fx.Cost = make([]int, w*h)
			for i := range fx.Cost {
				fx.Cost[i] = []int{1, 1, 1, 2, 3, 99}[e.intn(6)]
			}
		}
		rawProb := []int{0, 50, 100, 100}[k%4]
		count := e.rangeIn(2, 9)
		for i := 0; i < count; i++ {
			u := &battle.Unit{
				Camp: battle.Camp(e.intn(3)), ClassID: []int{1, 5, 12, 19, 25, 26, 30, 99}[e.intn(8)],
				Lv: e.rangeIn(1, 40), MaxHP: e.rangeIn(1, 300), MP: e.rangeIn(0, 60), MaxMP: 60,
				AP: e.rangeIn(0, 300), DP: e.rangeIn(0, 300), HIT: e.rangeIn(0, 200), EV: e.rangeIn(0, 200),
				DX: e.rangeIn(0, 100), MV: e.rangeIn(1, 6), AtkMin: e.intn(3), AtkMax: e.intn(3),
				X: e.intn(w), Y: e.intn(h), OnField: e.chance(85), Acted: e.chance(15),
			}
			u.HP = e.rangeIn(0, u.MaxHP)
			if e.chance(5) {
				u.HP = u.MaxHP + 5
			}
			if e.chance(30) {
				u.NativeTransient[e.intn(6)] = byte(e.rangeIn(1, 5))
			}
			if e.chance(rawProb) {
				u.NativeRecordByte5, u.HasNativeRecordByte5 = byte(e.intn(2)), true
				if e.chance(20) {
					u.NativeRecordByte5 |= 0x80
				}
				u.NativeRecordByte6, u.HasNativeRecordByte6 = byte([]int{2, 1, 0}[int(u.Camp)]), true
				u.NativeMapPresentation = battle.NativeMapPresentationState{X: byte(u.X), Y: byte(u.Y)}
				u.HasNativeMapPresentation = true
				u.NativeRecordClass, u.HasNativeRecordClass = byte(u.ClassID), true
				u.BattleFig, u.HasBattleFig = []int{4, 5, 6, 7, 20, 3}[e.intn(6)], true
				u.NativeRecordByte34, u.HasNativeRecordByte34 = byte(e.intn(16)), true
				u.NativeRecordByte35, u.HasNativeRecordByte35 = byte(e.intn(16)), true
				u.NativeRecordByte36, u.HasNativeRecordByte36 = byte(e.intn(16)), true
				u.NativeRecordWord42, u.HasNativeRecordWord42 = uint16(u.MaxHP), true
				u.NativeRecordWord46, u.HasNativeRecordWord46 = uint16(u.MaxMP), true
			}
			if u.Camp == battle.Own && e.chance(50) {
				u.NativeIdentity, u.HasNativeIdentity = e.intn(34), true
			}
			fx.Units = append(fx.Units, unitObj(u))
		}
		e.fixtures = append(e.fixtures, fx)
	}
}

func (e *exporter) fixtureList() []obj {
	out := []obj{}
	for _, fx := range e.fixtures {
		var cost any
		if fx.Cost != nil {
			cost = fx.Cost
		}
		out = append(out, obj{"name": fx.Name, "w": fx.W, "h": fx.H, "events": hexs(fx.Events), "cost": cost, "units": fx.Units})
	}
	return out
}

// ---- 修改產生器 ----

// randomEdit 對一個單位改一兩個欄位。
func (e *exporter) randomEdit(n int) obj {
	set := obj{}
	switch e.intn(9) {
	case 0:
		set["HP"] = e.rangeIn(0, 120)
	case 1:
		set["MP"] = e.rangeIn(0, 80)
	case 2:
		set["Acted"] = e.chance(50)
	case 3:
		set["OnField"] = e.chance(70)
	case 4:
		set["native_transient"] = []int{e.intn(3), e.intn(3), e.intn(3), e.intn(4), e.intn(4), e.intn(4)}
	case 5:
		set["native_record_byte5"], set["has_native_record_byte5"] = e.intn(2), e.chance(90)
	case 6:
		set["Lv"] = []int{0, 5, 99, 255, 256, -1}[e.intn(6)]
	case 7:
		set["AP"] = []int{0, 10, 500, 32767, 40000, -40000}[e.intn(6)]
	default:
		set["ClassID"], set["NativeRecordClass"] = []int{1, 25, 26, 19, 99}[e.intn(5)], e.intn(40)
	}
	return obj{"unit": e.intn(n), "set": set}
}

func (e *exporter) baseSpec(k int) obj {
	n := len(e.fixtures[k].Units)
	edits := []obj{}
	for i := e.intn(3); i > 0; i-- {
		edits = append(edits, e.randomEdit(n))
	}
	sp := obj{"fixture": k, "edits": edits}
	if e.chance(8) {
		sp["book_len"] = []int{0, 10, 35, 37}[e.intn(4)]
	}
	if e.chance(12) {
		fields := []string{"ID", "Damage", "Hit", "SelectionMode", "EffectMode", "MPCost", "TargetCode"}
		edits := [][3]any{}
		for i := e.rangeIn(1, 3); i > 0; i-- {
			edits = append(edits, [3]any{e.intn(36), fields[e.intn(len(fields))], []int{-1, 0, 1, 2, 5, 16, 18, 101, 300}[e.intn(9)]})
		}
		sp["book_edits"] = edits
	}
	if e.chance(10) {
		sp["resist_drop"] = []int{e.rangeIn(1, 30), e.rangeIn(1, 30)}
	}
	if e.chance(6) {
		sp["resist_set"] = [][2]int{{e.rangeIn(0, 40), []int{-1, 0, 5, 10, 11}[e.intn(5)]}}
	}
	return sp
}

func (e *exporter) randomBalance() obj {
	growth := []int{100, 100, 100, 100, 100}
	if e.chance(40) {
		for i := range growth {
			growth[i] = []int{0, 50, 100, 250, 500, 501, -1}[e.intn(7)]
		}
	}
	chars := [][]int{}
	for i := e.intn(3); i > 0; i-- {
		chars = append(chars, []int{[]int{0, 3, 31, 32, -1}[e.intn(5)], 100, e.rangeIn(0, 600), 100, 100, 100})
	}
	hits := [][]int{}
	bad := false
	for i := e.intn(4); i > 0; i-- {
		id, cmd, n := e.intn(32), []int{0, 5, 12, 22, 26, 27, 32}[e.intn(7)], e.rangeIn(0, 100)
		if !bad && e.chance(20) {
			// 只放一筆不合法的列：Go 以 map 迭代檢查，多筆不合法時錯誤訊息取決於順序。
			bad = true
			switch e.intn(3) {
			case 0:
				id = []int{-1, 32, 40}[e.intn(3)]
			case 1:
				cmd = []int{13, 20, 35}[e.intn(3)]
			default:
				n = []int{-1, 101}[e.intn(2)]
			}
		}
		hits = append(hits, []int{id, cmd, n})
	}
	return obj{"growth": growth, "characters": chars, "skill_hit": hits,
		"enemy_percent": []int{10, 50, 100, 150, 500, 9, 501}[e.intn(7)]}
}

// ---- 純函式 ----

func cellsOf(m map[battle.Cell]bool) [][2]int {
	out := [][2]int{}
	for c, ok := range m {
		if ok {
			out = append(out, [2]int{c.X, c.Y})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][1] != out[j][1] {
			return out[i][1] < out[j][1]
		}
		return out[i][0] < out[j][0]
	})
	return out
}

func (e *exporter) randomFlags(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		switch e.intn(6) {
		case 0:
			b[i] = battle.NativeCommandGridBlocked
		case 1:
			b[i] = battle.NativeCommandGridZeroBudget
		default:
			b[i] = byte(e.intn(32))
		}
	}
	return b
}

func (e *exporter) genGeometry() {
	for k := 0; k < 30; k++ {
		w, h := e.rangeIn(-1, 6), e.rangeIn(-1, 6)
		n := 0
		if w > 0 && h > 0 {
			n = w * h
		}
		if e.chance(15) {
			n++
		}
		events := e.bytesN(n)
		flags, err := battle.NativeCompositionBaseFlags(w, h, events)
		e.add("composition_flags", obj{"w": w, "h": h, "events": hexs(events)}, obj{"flags": hexs(flags), "err": errv(err)})
	}
	for k := 0; k < 200; k++ {
		w, h := e.rangeIn(1, 10), e.rangeIn(1, 10)
		origin := battle.Cell{X: e.rangeIn(-1, w), Y: e.rangeIn(-1, h)}
		if e.chance(85) {
			origin = battle.Cell{X: e.intn(w), Y: e.intn(h)}
		}
		dist := e.rangeIn(-1, 0x15)
		n := w * h
		if e.chance(5) {
			n++
		}
		flags := e.randomFlags(n)
		inner := e.rangeIn(-1, 3)
		cells, err := battle.NativeCommandTargetCells(w, h, origin, dist, flags)
		field, ferr := battle.NativeCommandTargetFieldBytes(w, h, origin, dist, inner, flags)
		e.add("target_cells", obj{"w": w, "h": h, "origin": [2]int{origin.X, origin.Y}, "dist": dist,
			"inner": inner, "flags": hexs(flags)},
			obj{"cells": cellsOf(cells), "err": errv(err), "field": hexs(field), "field_err": errv(ferr)})
	}
	matches := [][]bool{}
	for code := -1; code <= 6; code++ {
		row := []bool{}
		for camp := 0; camp < 3; camp++ {
			row = append(row, battle.NativeCommandTargetMatches(code, battle.Camp(camp)))
		}
		matches = append(matches, row)
	}
	e.add("target_matches", obj{"codes": [2]int{-1, 6}}, obj{"matches": matches})
	plans := []any{}
	for id := -1; id <= 40; id++ {
		steps, ok := battle.NativeCompoundCommandPlan(id)
		plans = append(plans, obj{"steps": conv(nil, steps), "ok": ok})
	}
	e.add("compound_plan", obj{"ids": [2]int{-1, 40}}, obj{"plans": plans})
	overlay := []int{}
	for b := 0; b < 256; b++ {
		overlay = append(overlay, battle.NativeMapOverlaySelectorFromRecordByte(byte(b)))
	}
	e.add("overlay_selector", obj{}, obj{"values": overlay})
}

func (e *exporter) genDamage() {
	for k := 0; k < 250; k++ {
		damage := []int{-1, 0, 1, 50, 440, 999, 30000}[e.intn(7)]
		hit := []int{-1, 0, 50, 90, 100, 101}[e.intn(6)]
		resist := []int{-1, 0, 4, 7, 10, 11}[e.intn(6)]
		state := e.u16()
		r, next, err := battle.ResolveNativeCommandDamage(damage, hit, resist, uint16(state))
		e.add("resolve_damage", obj{"damage": damage, "hit": hit, "resist": resist, "rng": state},
			obj{"result": conv(nil, r), "rng": next, "err": errv(err)})
	}
	for k := 0; k < 80; k++ {
		fk := e.intn(len(e.fixtures))
		sp := e.baseSpec(fk)
		units := e.buildUnits(sp)
		target := e.intn(len(units))
		if e.chance(5) {
			target = -1
		}
		damage, hit, resist, state := e.rangeIn(0, 500), e.rangeIn(0, 101), e.rangeIn(0, 10), e.u16()
		before := snapshot(units)
		var t *battle.Unit
		if target >= 0 {
			t = units[target]
		}
		r, next, err := battle.ApplyNativeCommandDamage(t, damage, hit, resist, uint16(state))
		e.add("apply_damage", obj{"state": sp, "target": target, "damage": damage, "hit": hit, "resist": resist, "rng": state},
			obj{"result": conv(nil, r), "rng": next, "err": errv(err), "changed": changed(before, units)})
	}
	for k := 0; k < 100; k++ {
		fk := e.intn(len(e.fixtures))
		sp := e.baseSpec(fk)
		units := e.buildUnits(sp)
		target := e.intn(len(units))
		if e.chance(5) {
			target = -1
		}
		amount := []int{-1, 0, 30, 70, 800, 2000}[e.intn(6)]
		native := e.chance(50)
		nilRNG := !native && e.chance(8)
		seed, state := e.seed(), e.u16()
		before := snapshot(units)
		var t *battle.Unit
		if target >= 0 {
			t = units[target]
		}
		in := obj{"state": sp, "target": target, "amount": amount, "native": native, "nil_rng": nilRNG, "seed": seed, "rng": state}
		var res obj
		if native {
			r, next, err := battle.ApplyNativeCommandRestoreNative(t, amount, uint16(state))
			res = obj{"result": conv(nil, r), "rng": next, "err": errv(err)}
		} else {
			var rng *mrand.Rand
			if !nilRNG {
				rng = mrand.New(mrand.NewSource(seed))
			}
			r, err := battle.ApplyNativeCommandRestore(t, amount, rng)
			res = obj{"result": conv(nil, r), "err": errv(err)}
		}
		res["changed"] = changed(before, units)
		e.add("restore", in, res)
	}
	for k := 0; k < 100; k++ {
		n := e.rangeIn(1, 4)
		records := e.bytesN(n * 0x50)
		for i := 0; i < n; i++ {
			base := i * 0x50
			for j := 0x22; j < 0x28; j++ {
				records[base+j] = byte(e.intn(4))
			}
		}
		indices := make([]byte, n)
		for i := range indices {
			indices[i] = byte(i)
		}
		if e.chance(10) {
			indices = append(indices, byte(n+e.intn(3)))
		}
		cmd := e.rangeIn(16, 20)
		state := e.u16()
		before := hexs(records)
		r, err := battle.ApplyNativeCommandModifier(records, indices, cmd, uint16(state))
		e.add("modifier_raw", obj{"records": before, "indices": ints(indices), "command": cmd, "rng": state},
			obj{"result": conv(nil, r), "err": errv(err), "records": hexs(records)})
	}
	// 抗性表載入：真實檔與竄改過的複本（本組自己的拒收訊息逐字比對）。
	raw, err := os.ReadFile("assets/data/native_command_resistances.json")
	if err != nil {
		panic(err)
	}
	texts := []string{string(raw), `[]`, `[{"cls": 0, "resist_raw": 5}]`, `[{"cls": 3, "resist_raw": 11}]`,
		`[{"cls": 3, "resist_raw": -1}]`, `[{"cls": 3, "resist_raw": 5}, {"cls": 3, "resist_raw": 6}]`,
		`[{"cls": 1, "resist_raw": 10}, {"cls": 2}]`, `{"cls": 1}`, `[{"cls": "x"}]`,
		strings.Replace(string(raw), `"resist_raw": 7`, `"resist_raw": 70`, 1)}
	for i, text := range texts {
		path := filepath.Join(e.tmp, fmt.Sprintf("resist-%d.json", i))
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			panic(err)
		}
		table, err := battle.LoadNativeCommandResistances(path)
		rows := [][2]int{}
		for k, v := range table {
			rows = append(rows, [2]int{k, v})
		}
		sort.Slice(rows, func(a, b int) bool { return rows[a][0] < rows[b][0] })
		// 第 0 筆是真實檔原文；Rust 端改讀 FD2_GOLDEN_REPO 底下的同一份並核對雜湊。
		input := obj{"text": text}
		if i == 0 {
			input = obj{"path": "assets/data/native_command_resistances.json"}
		}
		e.add("load_resistances", input, obj{"table": rows, "err": errv(err), "json_err": err != nil && i >= 7})
	}
}

// ---- 移動、路徑與 AI 幾何 ----

func (e *exporter) costRow() []byte {
	if e.chance(5) {
		return e.bytesN(e.rangeIn(19, 21))
	}
	return append([]byte(nil), e.costRows[e.intn(len(e.costRows))]...)
}

func (e *exporter) terrain(n, limit int, row []byte, positive bool) []byte {
	b := make([]byte, n)
	for i := range b {
		for {
			b[i] = byte(e.intn(limit))
			if !positive || int(b[i]) >= len(row) || row[b[i]] > 0 {
				break
			}
		}
	}
	return b
}

func (e *exporter) genPaths() {
	for k := 0; k < 80; k++ {
		cands := [][2]int{}
		cells := []battle.Cell{}
		for i := e.intn(9); i > 0; i-- {
			c := battle.Cell{X: e.intn(16), Y: e.intn(16)}
			cands = append(cands, [2]int{c.X, c.Y})
			cells = append(cells, c)
		}
		intended := battle.Cell{X: e.intn(16), Y: e.intn(16)}
		best, ok := battle.SelectNativeMovementDestination(cells, intended)
		e.add("select_destination", obj{"candidates": cands, "intended": [2]int{intended.X, intended.Y}},
			obj{"best": [2]int{best.X, best.Y}, "found": ok})
	}
	for k := 0; k < 80; k++ {
		count := e.rangeIn(0, 6)
		records := make([]byte, count*0x50)
		for i := 0; i < count; i++ {
			records[i*0x50], records[i*0x50+1] = byte(e.intn(12)), byte(e.intn(12))
			records[i*0x50+6] = byte(e.intn(3))
		}
		if e.chance(10) {
			count++
		}
		actor := e.rangeIn(-1, count)
		selector := e.intn(3)
		best, ok, err := battle.SelectNativeNearestOppositeCoordinate(records, count, actor, selector)
		e.add("nearest_opposite", obj{"records": hexs(records), "count": count, "actor": actor, "selector": selector},
			obj{"best": [2]int{best.X, best.Y}, "found": ok, "err": errv(err)})
	}
	for k := 0; k < 120; k++ {
		w, h := e.rangeIn(1, 7), e.rangeIn(1, 7)
		row := e.costRow()
		n := w * h
		if e.chance(5) {
			n--
		}
		limit := 20
		if e.chance(5) {
			limit = 24
		}
		flags := e.randomFlags(n)
		mode := []int{0, 1, 2, 0, 1}[e.intn(5)]
		terrain := e.terrain(n, limit, row, mode == 1)
		start := battle.Cell{X: e.intn(w), Y: e.intn(h)}
		if e.chance(5) {
			start.X = w
		}
		dest := battle.Cell{X: e.intn(w), Y: e.intn(h)}
		budget := []int{-1, 0, 1, 3, 5, 8, 256}[e.intn(7)]
		if e.chance(70) {
			budget = e.rangeIn(0, 8)
		}
		in := obj{"w": w, "h": h, "start": [2]int{start.X, start.Y}, "budget": budget, "flags": hexs(flags),
			"terrain": hexs(terrain), "cost_row": hexs(row)}
		if k%2 == 0 {
			c, ok, err := battle.NativePathBlockedCoordinate(w, h, start, budget, flags, terrain, row)
			e.add("blocked_coordinate", in, obj{"cell": [2]int{c.X, c.Y}, "found": ok, "err": errv(err)})
		} else {
			in["dest"], in["mode"] = [2]int{dest.X, dest.Y}, mode
			path, ok, err := battle.NativePathDirections(w, h, start, dest, budget, mode, flags, terrain, row)
			e.add("path_directions", in, obj{"path": ints(path), "found": ok, "err": errv(err)})
		}
	}
	for k := 0; k < 100; k++ {
		w, h := e.rangeIn(1, 8), e.rangeIn(1, 8)
		count := e.rangeIn(1, 6)
		records := make([]byte, count*0x50)
		for i := 0; i < count; i++ {
			records[i*0x50], records[i*0x50+1] = byte(e.intn(w)), byte(e.intn(h))
			records[i*0x50+5] = byte(e.intn(2) * e.intn(2))
			records[i*0x50+6] = byte(e.intn(3))
		}
		if e.chance(5) {
			records[0] = byte(w + 1)
		}
		actor := e.intn(count)
		if e.chance(5) {
			actor = count
		}
		selector := []int{0, 1, 0, 1, 2}[e.intn(5)]
		command := battle.NativeCommandRecord{ID: 0, SelectionMode: e.rangeIn(0, 6), EffectMode: []int{0, 1, 2, 3, 0x10, 0x11, 0x12}[e.intn(7)],
			TargetCode: []int{0, 1, 2, 3, 4, 256}[e.intn(6)]}
		if e.chance(40) {
			command = e.book[e.intn(len(e.book))]
		}
		row := e.costRow()
		n := w * h
		if e.chance(4) {
			n++
		}
		flags := e.randomFlags(n)
		terrain := e.terrain(n, 20, row, false)
		groups, err := battle.NativeAIScoredCommandCandidateGroups(w, h, records, count, actor, selector, command, flags, terrain, row)
		e.add("scored_groups", obj{"w": w, "h": h, "records": hexs(records), "count": count, "actor": actor,
			"selector": selector, "command": conv(nil, command), "flags": hexs(flags), "terrain": hexs(terrain), "cost_row": hexs(row)},
			obj{"groups": conv(nil, groups), "err": errv(err)})
	}
}

// ---- 地圖視圖 ----

func viewArr(v battle.NativeMapViewState) [6]int {
	return [6]int{v.CameraX, v.CameraY, v.CursorX, v.CursorY, v.VisibleCursorX, v.VisibleCursorY}
}

func viewFrom(a [6]int) battle.NativeMapViewState {
	return battle.NativeMapViewState{CameraX: a[0], CameraY: a[1], CursorX: a[2], CursorY: a[3], VisibleCursorX: a[4], VisibleCursorY: a[5]}
}

func (e *exporter) randomView(w, h int) [6]int {
	cx, cy := e.rangeIn(0, max(0, w-13)), e.rangeIn(0, max(0, h-8))
	vx, vy := e.intn(13), e.intn(8)
	v := [6]int{cx, cy, cx + vx, cy + vy, vx, vy}
	if e.chance(15) {
		v[e.intn(6)] += []int{-2, -1, 1, 2, 9}[e.intn(5)]
	}
	return v
}

func (e *exporter) genMapView() {
	for k := 0; k < 120; k++ {
		w, h := e.rangeIn(13, 30), e.rangeIn(8, 24)
		if e.chance(8) {
			w, h = e.rangeIn(10, 13), e.rangeIn(6, 8)
		}
		st := &battle.State{W: w, H: h}
		ops := []obj{}
		outs := []any{}
		if e.chance(85) {
			v := e.randomView(w, h)
			ops = append(ops, obj{"op": "materialize", "view": v})
		}
		for i := e.rangeIn(3, 25); i > 0; i-- {
			switch e.intn(12) {
			case 0, 1, 2, 3:
				d := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {0, 0}}[e.intn(6)]
				if e.chance(85) {
					d = [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}[e.intn(4)]
				}
				ops = append(ops, obj{"op": "move", "dx": d[0], "dy": d[1]})
			case 4:
				ops = append(ops, obj{"op": "focus", "x": e.rangeIn(-1, w), "y": e.rangeIn(-1, h)})
			case 5:
				ops = append(ops, obj{"op": "focus_steps", "x": e.intn(w), "y": e.intn(h)})
			case 6, 7:
				d := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {2, 0}}[e.intn(5)]
				ops = append(ops, obj{"op": "walk", "ux": e.intn(w), "uy": e.intn(h), "dx": d[0], "dy": d[1]})
			case 8:
				d := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {0, 2}}[e.intn(5)]
				ops = append(ops, obj{"op": "walk_state", "view": e.randomView(w, h), "w": w, "h": h, "ux": e.intn(w), "uy": e.intn(h), "dx": d[0], "dy": d[1]})
			case 9:
				ops = append(ops, obj{"op": "range_mode", "mode": []int{-1, 0, 1, 6, 0x101, 0x102}[e.intn(6)]})
			case 10:
				ops = append(ops, obj{"op": "redraws", "before": e.randomView(w, h), "after": e.randomView(w, h)})
			default:
				ops = append(ops, obj{"op": "materialize", "view": e.randomView(w, h)})
			}
		}
		for _, op := range ops {
			switch op["op"] {
			case "materialize":
				err := st.MaterializeNativeMapViewState(viewFrom(op["view"].([6]int)))
				outs = append(outs, obj{"err": errv(err)})
			case "move":
				moved, ok := st.MoveNativeMapCursor(op["dx"].(int), op["dy"].(int))
				outs = append(outs, obj{"moved": moved, "ok": ok})
			case "focus":
				outs = append(outs, obj{"ok": st.FocusNativeMapCursor(op["x"].(int), op["y"].(int))})
			case "focus_steps":
				steps := []any{}
				ok := st.FocusNativeMapCursorSteps(op["x"].(int), op["y"].(int), func(before, after battle.NativeMapViewState) {
					steps = append(steps, obj{"before": viewArr(before), "after": viewArr(after), "redraws": st.NativeMapCursorStepRedraws(before, after)})
				})
				outs = append(outs, obj{"ok": ok, "steps": steps})
			case "walk":
				outs = append(outs, obj{"ok": st.AdvanceNativeMapWalkStepView(op["ux"].(int), op["uy"].(int), op["dx"].(int), op["dy"].(int))})
			case "walk_state":
				v := viewFrom(op["view"].([6]int))
				nv, ok := battle.AdvanceNativeMapWalkStepViewState(v, op["w"].(int), op["h"].(int), op["ux"].(int), op["uy"].(int), op["dx"].(int), op["dy"].(int))
				outs = append(outs, obj{"view": viewArr(nv), "ok": ok, "in_field": v.CursorInField(op["w"].(int), op["h"].(int)), "in_viewport": v.VisibleCursorInViewport()})
			case "range_mode":
				outs = append(outs, obj{"ok": st.MaterializeNativeMapRangeMode(op["mode"].(int))})
			case "redraws":
				outs = append(outs, obj{"redraws": st.NativeMapCursorStepRedraws(viewFrom(op["before"].([6]int)), viewFrom(op["after"].([6]int)))})
			}
		}
		e.add("map_view", obj{"w": w, "h": h, "ops": ops},
			obj{"outs": outs, "view": viewArr(st.NativeMapViewState), "has_view": st.HasNativeMapViewState,
				"range_mode": st.NativeMapRangeMode, "has_range_mode": st.HasNativeMapRangeModeState})
	}
}

// ---- 平衡設定 ----

func (e *exporter) genBalance() {
	for k := 0; k < 120; k++ {
		b := e.randomBalance()
		o := balanceFrom(b)
		e.add("balance_validate", obj{"balance": b}, obj{"err": errv(o.Validate()), "original": o.IsOriginal()})
	}
	rows := []any{}
	for id := -2; id <= 40; id++ {
		rows = append(rows, battle.NativeCommandHasHitRoll(id))
	}
	e.add("hit_roll", obj{"ids": [2]int{-2, 40}}, obj{"rows": rows, "original": battle.NewBalanceOptions().IsOriginal(),
		"rates": conv(nil, battle.OriginalGrowthRates())})
	for k := 0; k < 120; k++ {
		fk := e.intn(len(e.fixtures))
		sp := e.baseSpec(fk)
		n := len(e.fixtures[fk].Units)
		actor := e.intn(n)
		set := obj{"Camp": e.intn(3)}
		if e.chance(70) {
			set["Camp"] = 0
			set["NativeIdentity"], set["HasNativeIdentity"] = []int{-1, 0, 5, 31, 32}[e.intn(5)], e.chance(85)
		}
		sp["edits"] = append(sp["edits"].([]obj), obj{"unit": actor, "set": set})
		if e.chance(80) {
			sp["balance"] = e.randomBalance()
		}
		st := e.buildState(sp)
		id, ok := battle.BalanceIdentity(st.Units[actor])
		hits := []int{}
		for _, cmd := range []int{0, 5, 12, 13, 22, 26, 32, 33} {
			hits = append(hits, st.CommandHit(st.Units[actor], cmd, 77))
		}
		e.add("balance_identity", obj{"state": sp, "actor": actor}, obj{"id": id, "ok": ok, "hits": hits,
			"nil_hit": st.CommandHit(nil, 0, 77)})
	}
	for k := 0; k < 100; k++ {
		fk := e.intn(len(e.fixtures))
		sp := e.baseSpec(fk)
		n := len(e.fixtures[fk].Units)
		unit := e.intn(n)
		set := obj{}
		if e.chance(50) {
			set["EquipmentBaseSet"], set["BaseAP"], set["BaseDP"] = true, e.rangeIn(0, 200), e.rangeIn(0, 200)
		}
		if e.chance(40) {
			set["native_record_word42"], set["has_native_record_word42"] = e.rangeIn(0, 500), true
			set["native_record_word46"], set["has_native_record_word46"] = e.rangeIn(0, 500), true
		}
		if e.chance(10) {
			set["AP"], set["HIT"] = 32700, 32760
		}
		sp["edits"] = append(sp["edits"].([]obj), obj{"unit": unit, "set": set})
		units := e.buildUnits(sp)
		u := units[unit]
		pick := func() int { return []int{-1, 0, 1, 50, 300, 32767, 32768}[e.intn(7)] }
		v := battle.BalanceStats{HP: pick(), MP: pick(), AP: pick(), DP: pick(), DX: pick()}
		if e.chance(60) {
			v = battle.BalanceStats{HP: e.rangeIn(1, 400), MP: e.rangeIn(0, 100), AP: e.rangeIn(0, 300), DP: e.rangeIn(0, 300), DX: e.rangeIn(0, 100)}
		}
		before := conv(nil, u.BalanceStats())
		err := u.SetBalanceStats(v)
		e.add("balance_stats", obj{"state": sp, "unit": unit, "stats": conv(nil, v)},
			obj{"before": before, "err": errv(err), "unit_after": json.RawMessage(snapshot([]*battle.Unit{u})[0])})
	}
	for k := 0; k < 80; k++ {
		fk := e.intn(len(e.fixtures))
		sp := e.baseSpec(fk)
		n := len(e.fixtures[fk].Units)
		for i := e.intn(3); i > 0; i-- {
			sp["edits"] = append(sp["edits"].([]obj), obj{"unit": e.intn(n), "set": obj{
				"enemy_balance_base":    obj{"HP": e.rangeIn(0, 400), "MP": e.rangeIn(0, 60), "AP": e.rangeIn(0, 300), "DP": e.rangeIn(0, 300), "DX": e.rangeIn(0, 100)},
				"enemy_balance_percent": []int{0, 100, 150}[e.intn(3)]}})
		}
		pcts := []any{}
		for i := e.rangeIn(1, 3); i > 0; i-- {
			if e.chance(15) {
				pcts = append(pcts, nil)
			} else {
				pcts = append(pcts, []int{10, 50, 100, 150, 300, 500}[e.intn(6)])
			}
		}
		st := e.buildState(sp)
		before := snapshot(st.Units)
		for _, p := range pcts {
			if p == nil {
				st.Balance = nil
			} else {
				st.Balance = &battle.BalanceOptions{EnemyPercent: p.(int)}
			}
			st.ApplyEnemyStrength()
		}
		e.add("enemy_strength", obj{"state": sp, "percents": pcts}, obj{"changed": changed(before, st.Units)})
	}
}

// ---- 移動 ----

func (e *exporter) genMove() {
	for k := 0; k < 160; k++ {
		fk := e.intn(len(e.fixtures))
		sp := e.baseSpec(fk)
		n := len(e.fixtures[fk].Units)
		unit := e.intn(n)
		set := obj{"MV": e.rangeIn(0, 7)}
		if e.chance(40) {
			set["AtkMin"], set["AtkMax"] = e.intn(3), e.intn(4)
		}
		sp["edits"] = append(sp["edits"].([]obj), obj{"unit": unit, "set": set})
		st := e.buildState(sp)
		u := st.Units[unit]
		reach := st.Reachable(u)
		probes := [][2]int{}
		for i := 0; i < 6; i++ {
			probes = append(probes, [2]int{e.rangeIn(-1, st.W), e.rangeIn(-1, st.H)})
		}
		cells := cellsOf(reach)
		if len(cells) > 0 {
			probes = append(probes, cells[e.intn(len(cells))])
		}
		costs, ranges, paths := []int{}, []bool{}, []any{}
		for _, p := range probes {
			costs = append(costs, st.MoveCost(p[0], p[1]))
			ranges = append(ranges, st.InAttackRange(u, p[0], p[1]))
			path := st.Path(u, p[0], p[1])
			arr := [][2]int{}
			for _, c := range path {
				arr = append(arr, [2]int{c.X, c.Y})
			}
			if path == nil {
				paths = append(paths, nil)
			} else {
				paths = append(paths, arr)
			}
		}
		e.add("move", obj{"state": sp, "unit": unit, "probes": probes},
			obj{"reachable": cells, "costs": costs, "in_range": ranges, "paths": paths})
	}
}

// ---- 死亡程式 ----

func (e *exporter) randomWhen() obj {
	w := obj{}
	if e.chance(40) {
		w["state_eq"] = [2]int{e.rangeIn(-1, 33), e.intn(3)}
	}
	if e.chance(30) {
		w["state_ne"] = [2]int{e.rangeIn(-1, 33), e.intn(3)}
	}
	if e.chance(30) {
		w["round_lt"] = e.rangeIn(0, 8)
	}
	if e.chance(30) {
		// Go 的 Match 對負的起點會索引越界 panic，黃金只放非負起點。
		w["any_active"] = [2]int{e.rangeIn(0, 6), e.rangeIn(0, 10)}
	}
	return w
}

func whenFrom(v any) *battle.NativeActionWhen {
	m, ok := v.(obj)
	if !ok {
		return nil
	}
	w := &battle.NativeActionWhen{}
	if p, ok := m["state_eq"].([2]int); ok {
		w.StateEq = &p
	}
	if p, ok := m["state_ne"].([2]int); ok {
		w.StateNe = &p
	}
	if p, ok := m["round_lt"].(int); ok {
		w.RoundLt = &p
	}
	if p, ok := m["any_active"].([2]int); ok {
		w.AnyActive = &p
	}
	return w
}

func (e *exporter) randomDeathOp(n int) obj {
	ops := []string{"ai_mode_range", "ai_byte_set_range", "ai_byte_and_range", "record_bytes", "control_turn",
		"state_set", "state_inc", "mark_inactive", "range_one", "range_zero", "exp_cancel", "bogus"}
	op := obj{"op": ops[e.intn(len(ops))], "unit": e.rangeIn(-1, n+1), "first": e.rangeIn(-2, n), "last": e.rangeIn(-1, n+2),
		"mode": e.rangeIn(-1, 16), "value": e.rangeIn(0, 300), "mask": e.intn(256), "slot": e.rangeIn(-1, 16),
		"delta": []int{-5, 0, 1, 3, 300}[e.intn(5)], "index": e.rangeIn(-1, 32), "x": 0, "y": 0, "group": 0, "kind": 0}
	writes := [][3]int{}
	for i := e.intn(4); i > 0; i-- {
		offset := []int{5, 6, 7, 8, 0x31, 0x34, 0x35, 0x36, 0x40, 0x41, 9}[e.intn(11)]
		width := 1
		if offset == 0x40 || e.chance(10) {
			width = 2
		}
		value := e.rangeIn(0, 300)
		if offset == 6 {
			value = e.rangeIn(0, 3)
		}
		if offset == 0x31 && e.chance(40) {
			value = 0xff
		}
		if e.chance(5) {
			value = []int{-1, 0x10000}[e.intn(2)]
		}
		writes = append(writes, [3]int{offset, value, width})
	}
	op["writes"] = writes
	return op
}

func deathOpFrom(m obj) battle.NativeDeathOp {
	return battle.NativeDeathOp{Op: m["op"].(string), Unit: m["unit"].(int), First: m["first"].(int), Last: m["last"].(int),
		Mode: m["mode"].(int), Value: m["value"].(int), Mask: m["mask"].(int), Slot: m["slot"].(int), Delta: m["delta"].(int),
		Index: m["index"].(int), X: m["x"].(int), Y: m["y"].(int), Group: m["group"].(int), Kind: m["kind"].(int),
		Writes: m["writes"].([][3]int)}
}

func (e *exporter) deathState(sp obj) (*battle.State, obj) {
	st := e.buildState(sp)
	extra := obj{"event_state": ints(e.bytesN(32)), "round": e.rangeIn(-1, 9), "has_turn": e.chance(80),
		"has_field": e.chance(70), "has_proj": e.chance(60)}
	controls := [][3]int{}
	for i := 0; i < 16; i++ {
		controls = append(controls, [3]int{e.intn(256), e.intn(256), e.intn(3)})
	}
	extra["turn_controls"] = controls
	extra["field_raw"] = hexs(e.bytesN(e.rangeIn(0, 60)))
	records := []string{}
	for i := e.rangeIn(0, len(st.Units)+1); i > 0; i-- {
		records = append(records, hexs(e.bytesN(0x50)))
	}
	extra["runtime_records"] = records
	applyDeathExtra(st, extra)
	return st, extra
}

func applyDeathExtra(st *battle.State, extra obj) {
	for i, v := range extra["event_state"].([]int) {
		st.NativeEventState[i] = byte(v)
	}
	st.NativeRoundCounter = extra["round"].(int)
	st.HasNativeTurnEventControlState = extra["has_turn"].(bool)
	for i, c := range extra["turn_controls"].([][3]int) {
		st.NativeTurnEventControls[i] = battle.NativeTurnEventControl{Turn: byte(c[0]), EventID: byte(c[1]), RawCamp: byte(c[2])}
	}
	st.HasNativeFieldControlState = extra["has_field"].(bool)
	st.NativeFieldControlRaw, _ = hex.DecodeString(extra["field_raw"].(string))
	st.HasNativeRuntimeUnitProjection = extra["has_proj"].(bool)
	for _, r := range extra["runtime_records"].([]string) {
		raw, _ := hex.DecodeString(r)
		var rec battle.NativeRuntimeRecordState
		copy(rec.Raw[:], raw)
		st.NativeRuntimeRecords = append(st.NativeRuntimeRecords, rec)
	}
}

func deathSummary(st *battle.State, before [][]byte) obj {
	controls := [][3]int{}
	for _, c := range st.NativeTurnEventControls {
		controls = append(controls, [3]int{int(c.Turn), int(c.EventID), int(c.RawCamp)})
	}
	records := []string{}
	for _, r := range st.NativeRuntimeRecords {
		records = append(records, hexs(r.Raw[:]))
	}
	return obj{"event_state": ints(st.NativeEventState[:]), "turn_controls": controls, "field_raw": hexs(st.NativeFieldControlRaw),
		"runtime_records": records, "range_mode": st.NativeMapRangeMode, "has_range_mode": st.HasNativeMapRangeModeState,
		"changed": changed(before, st.Units)}
}

func (e *exporter) genDeath() {
	for k := 0; k < 80; k++ {
		fk := e.intn(len(e.fixtures))
		sp := e.baseSpec(fk)
		n := len(e.fixtures[fk].Units)
		unit := e.intn(n)
		set := obj{}
		switch e.intn(3) {
		case 0:
			set["NativeRecordDeathEffect"], set["HasNativeRecordDeathEffect"] = []int{[]int{0, 1, 2, 3, 0xff}[e.intn(5)], e.intn(40), e.intn(3)}, true
		case 1:
			set["DeathEffect"] = obj{"type": e.intn(4), "value": e.intn(40)}
		}
		sp["edits"] = append(sp["edits"].([]obj), obj{"unit": unit, "set": set})
		units := e.buildUnits(sp)
		kind, value, ok := battle.NativeDeathEffectOf(units[unit])
		programs := []obj{}
		sc := &battle.Scenario{NativeDeathPrograms: map[string][]battle.Action{}}
		for i := e.intn(4); i > 0; i-- {
			key := fmt.Sprintf("%d:%d", e.rangeIn(1, 3), e.intn(40))
			if ok && e.chance(50) {
				key = fmt.Sprintf("%d:%d", kind, value)
			}
			actions := []obj{}
			var list []battle.Action
			for j := e.intn(4); j > 0; j-- {
				a := obj{"op": []any{nil, "exp_cancel", "state_set"}[e.intn(3)], "when": e.chance(30)}
				actions = append(actions, a)
				act := battle.Action{Type: "native_death_op"}
				if op, isOp := a["op"].(string); isOp {
					act.NativeDeathOp = &battle.NativeDeathOp{Op: op}
				}
				if a["when"].(bool) {
					act.NativeWhen = &battle.NativeActionWhen{}
				}
				list = append(list, act)
			}
			programs = append(programs, obj{"key": key, "actions": actions})
			sc.NativeDeathPrograms[key] = list
		}
		prog, key, found := sc.NativeDeathProgram(units[unit])
		e.add("death_program", obj{"state": sp, "unit": unit, "programs": programs},
			obj{"kind": kind, "value": value, "ok": ok, "actions": len(prog), "key": key, "found": found,
				"cancels": sc.NativeDeathCancelsExp(units[unit]), "inactive": battle.NativeRecordInactive(units[unit])})
	}
	for k := 0; k < 100; k++ {
		fk := e.intn(len(e.fixtures))
		sp := e.baseSpec(fk)
		st, extra := e.deathState(sp)
		whens := []any{}
		outs := []any{}
		for i := e.rangeIn(1, 4); i > 0; i-- {
			w := e.randomWhen()
			whens = append(whens, w)
			ok, err := whenFrom(w).Match(st)
			outs = append(outs, obj{"ok": ok, "err": errv(err)})
		}
		var nilState *battle.State
		nilOK, nilErr := whenFrom(obj{}).Match(nilState)
		e.add("when_match", obj{"state": sp, "extra": extra, "whens": whens},
			obj{"outs": outs, "nil_state": obj{"ok": nilOK, "err": errv(nilErr)}})
	}
	for k := 0; k < 140; k++ {
		fk := e.intn(len(e.fixtures))
		sp := e.baseSpec(fk)
		st, extra := e.deathState(sp)
		before := snapshot(st.Units)
		ops := []obj{}
		errs := []any{}
		for i := e.rangeIn(1, 6); i > 0; i-- {
			op := e.randomDeathOp(len(st.Units))
			ops = append(ops, op)
			errs = append(errs, errv(st.ApplyNativeDeathOp(deathOpFrom(op))))
		}
		var cleared any
		clearFrom := -1
		if e.chance(30) {
			clearFrom = e.rangeIn(-1, len(st.Units))
			cleared = conv(st.Units, st.ClearNativeHPFrom(clearFrom))
		}
		res := deathSummary(st, before)
		res["errs"], res["cleared"] = errs, cleared
		e.add("death_ops", obj{"state": sp, "extra": extra, "ops": ops, "clear_from": clearFrom}, res)
	}
}

// ---- 目標選擇（戰場底稿）----

func (e *exporter) randomCell(st *battle.State, oob int) [2]int {
	if e.chance(oob) {
		return [2]int{e.rangeIn(-1, st.W), e.rangeIn(-1, st.H)}
	}
	if len(st.Units) > 0 && e.chance(60) {
		u := st.Units[e.intn(len(st.Units))]
		return [2]int{min(max(u.X+e.rangeIn(-1, 1), 0), st.W-1), min(max(u.Y+e.rangeIn(-1, 1), 0), st.H-1)}
	}
	return [2]int{e.intn(st.W), e.intn(st.H)}
}

func cellFrom(a [2]int) battle.Cell { return battle.Cell{X: a[0], Y: a[1]} }

func unitOrNil(st *battle.State, i int) *battle.Unit {
	if i < 0 || i >= len(st.Units) {
		return nil
	}
	return st.Units[i]
}

// flagsFor 產生目標判定用的旗標格：0 = 0x4dbfc 基底、1/2 = 加上 0x145cd(0)/(1)、3 = 亂數。
func (e *exporter) flagsFor(st *battle.State, kind int) []byte {
	switch kind {
	case 0:
		f, _ := st.NativeCommandBaseFlags()
		return f
	case 1, 2:
		f, err := battle.NativeCommandRuntimeFlags(st.W, st.H, st.NativeCompositionEventBytes, st.Units, byte(kind-1))
		if err != nil {
			f, _ = st.NativeCommandBaseFlags()
		}
		return f
	}
	return e.randomFlags(st.W * st.H)
}

func (e *exporter) genTargets(k int) {
	for c := 0; c < 14; c++ {
		sp := e.baseSpec(k)
		st := e.buildState(sp)
		flagsKind := e.intn(4)
		flags := e.flagsFor(st, flagsKind)
		in := obj{"state": sp, "flags": hexs(flags)}
		res := obj{}
		op := ""
		switch c % 11 {
		case 0:
			op = "runtime_flags"
			sel := e.intn(3)
			f, err := battle.NativeCommandRuntimeFlags(st.W, st.H, st.NativeCompositionEventBytes, st.Units, byte(sel))
			in["selector"] = sel
			res = obj{"flags": hexs(f), "err": errv(err)}
			base, berr := st.NativeCommandBaseFlags()
			res["base"], res["base_err"] = hexs(base), errv(berr)
		case 1:
			op = "future_placement"
			pos := battle.NativePositionRecord{XWord: uint16(e.rangeIn(0, st.W+2) | e.intn(3)<<8), YWord: uint16(e.rangeIn(0, st.H+2))}
			gate := []int{0, 0, 1, 0x80}[e.intn(4)]
			cell, err := battle.NativeFutureGroupPlacement(st.W, st.H, st.NativeCompositionEventBytes, st.Units, pos, byte(gate))
			in["pos"], in["gate"] = [2]int{int(pos.XWord), int(pos.YWord)}, gate
			res = obj{"cell": [2]int{cell.X, cell.Y}, "err": errv(err)}
		case 2:
			op = "cursor_confirm"
			cursor := e.randomCell(st, 5)
			field := e.intn(256)
			if e.chance(25) {
				field = 0xff
			}
			overlay, code := []int{-1, 0, 1, 2, 3, 6, 0x101, 0x102}[e.intn(8)], e.rangeIn(-1, 7)
			ok, err := battle.NativeCursorConfirmationAllowed(cellFrom(cursor), byte(field), overlay, code, st.Units)
			in["cursor"], in["field"], in["overlay"], in["code"] = cursor, field, overlay, code
			res = obj{"ok": ok, "err": errv(err)}
		case 3:
			op = "command_targets"
			origin := e.randomCell(st, 5)
			dist, code := []int{-1, 0, 1, 2, 3, 5, 0x10, 0x11, 0x13}[e.intn(9)], e.rangeIn(-1, 4)
			t, err := battle.NativeCommandTargets(st.W, st.H, cellFrom(origin), dist, code, flags, st.Units)
			in["origin"], in["dist"], in["code"] = origin, dist, code
			res = obj{"targets": conv(st.Units, t), "err": errv(err)}
		case 4:
			op = "attack_candidates"
			origin := e.randomCell(st, 5)
			mode, inner, code := []int{0, 1, 2, 3, 5, 0x10, 0x12}[e.intn(7)], e.rangeIn(-1, 2), e.rangeIn(0, 4)
			t, err := battle.NativeAttackCandidates(st.W, st.H, cellFrom(origin), mode, inner, code, flags, st.Units)
			in["origin"], in["mode"], in["inner"], in["code"] = origin, mode, inner, code
			res = obj{"targets": conv(st.Units, t), "err": errv(err)}
		case 5:
			op = "effect_targets"
			sel, eff, code := []int{0, 1, 3, 5, 0x11}[e.intn(5)], []int{0, 1, 2, 0x10}[e.intn(4)], e.rangeIn(0, 4)
			actor := e.actorWith(st, sel, code, flags)
			confirmed := e.confirmedFor(st, actor, sel, code, flags)
			t, err := battle.NativeCommandEffectTargets(st.W, st.H, unitOrNil(st, actor), unitOrNil(st, confirmed), sel, eff, code, flags, st.Units)
			in["actor"], in["confirmed"], in["selection"], in["effect"], in["code"] = actor, confirmed, sel, eff, code
			res = obj{"targets": conv(st.Units, t), "err": errv(err)}
		case 6:
			op = "effect_targets_cursor"
			sel, eff, code := []int{0, 1, 3, 5, 0x11}[e.intn(5)], []int{0, 1, 2, 0x10, 0x100}[e.intn(5)], e.rangeIn(0, 5)
			actor := e.actorWith(st, sel, code, flags)
			cursor := e.randomCell(st, 5)
			if actor >= 0 && e.chance(75) {
				cursor = e.fieldCell(st, st.Units[actor], sel, flags)
			}
			t, err := battle.NativeCommandEffectTargetsAtCursor(st.W, st.H, unitOrNil(st, actor), cellFrom(cursor), sel, eff, code, flags, st.Units)
			in["actor"], in["cursor"], in["selection"], in["effect"], in["code"] = actor, cursor, sel, eff, code
			res = obj{"targets": conv(st.Units, t), "err": errv(err)}
		case 7:
			op = "self_centered"
			eff := []int{0, 1, 2, 3, 0x10, 0x12}[e.intn(6)]
			actor := e.actorWith(st, eff, 0, flags)
			cursor := e.randomCell(st, 5)
			if actor >= 0 && e.chance(75) {
				cursor = e.fieldCell(st, st.Units[actor], eff, flags)
			}
			t, err := battle.NativeCommandSelfCenteredTargets(st.W, st.H, unitOrNil(st, actor), cellFrom(cursor), eff, flags, st.Units)
			in["actor"], in["cursor"], in["effect"] = actor, cursor, eff
			res = obj{"targets": conv(st.Units, t), "err": errv(err)}
		case 8:
			op = "command30"
			saved, conf := e.randomCell(st, 5), e.randomCell(st, 5)
			steps := e.rangeIn(-1, 6)
			t, err := battle.NativeCommand30Targets(st.W, st.H, cellFrom(saved), cellFrom(conf), steps, st.Units)
			in["saved"], in["confirmed_cursor"], in["steps"] = saved, conf, steps
			res = obj{"targets": conv(st.Units, t), "err": errv(err)}
		case 9:
			op = "item_effect_targets"
			var row []byte
			rows := len(e.itemRows) / battle.NativeItemEffectRowSize
			if e.chance(70) && rows > 0 {
				r := e.intn(rows)
				row = append(row, e.itemRows[r*battle.NativeItemEffectRowSize:(r+1)*battle.NativeItemEffectRowSize]...)
			} else {
				row = e.bytesN(battle.NativeItemEffectRowSize)
				row[0x0d] = []byte{0x17, 1, 0x17}[e.intn(3)]
				row[0x10], row[0x12], row[0x15] = byte(e.intn(5)), byte([]int{0, 1, 2, 0x10}[e.intn(4)]), byte(e.intn(5))
			}
			if e.chance(5) {
				row = row[:len(row)-1]
			}
			plan, perr := battle.NativeItemTargetPlanFromRow(row)
			actor := e.actorWith(st, plan.SelectionMode, plan.TargetCode, flags)
			confirmed := e.confirmedFor(st, actor, plan.SelectionMode, plan.TargetCode, flags)
			in["row"], in["actor"], in["confirmed"] = hexs(row), actor, confirmed
			res = obj{"plan": conv(nil, plan), "plan_err": errv(perr)}
			if perr == nil {
				t, err := battle.NativeItemEffectTargets(st.W, st.H, unitOrNil(st, actor), unitOrNil(st, confirmed), plan, flags, st.Units)
				res["targets"], res["err"] = conv(st.Units, t), errv(err)
			}
		default:
			op = "base_flags"
			f, err := st.NativeCommandBaseFlags()
			res = obj{"flags": hexs(f), "err": errv(err)}
		}
		e.add(op, in, res)
	}
}

// confirmedFor 通常挑第一段候選裡的一個單位，讓第二段真的執行；偶爾挑任意單位。
func (e *exporter) confirmedFor(st *battle.State, actor, selection, code int, flags []byte) int {
	n := len(st.Units)
	if actor < 0 || actor >= n || e.chance(10) {
		if e.chance(5) {
			return -1
		}
		return e.intn(n)
	}
	a := st.Units[actor]
	cands, err := battle.NativeCommandTargets(st.W, st.H, battle.Cell{X: a.X, Y: a.Y}, selection, code, flags, st.Units)
	if err != nil || len(cands) == 0 {
		return e.intn(n)
	}
	pick := cands[e.intn(len(cands))]
	for i, u := range st.Units {
		if u == pick {
			return i
		}
	}
	return e.intn(n)
}

// actorWith 多半挑在 (mode, code) 下第一段候選不空的單位；偶爾任意挑或給 nil。
func (e *exporter) actorWith(st *battle.State, mode, code int, flags []byte) int {
	n := len(st.Units)
	if e.chance(4) {
		return -1
	}
	good := []int{}
	for i, u := range st.Units {
		cands, err := battle.NativeCommandTargets(st.W, st.H, battle.Cell{X: u.X, Y: u.Y}, mode, code, flags, st.Units)
		if err == nil && len(cands) > 0 {
			good = append(good, i)
		}
	}
	if len(good) == 0 || e.chance(15) {
		return e.intn(n)
	}
	return good[e.intn(len(good))]
}

// pickActor 多半挑「第一段候選不空」的單位當施放者，讓指令真的走到後段。
func (e *exporter) pickActor(st *battle.State, kind string, cmd int) int {
	n := len(st.Units)
	if e.chance(4) {
		return -1
	}
	if e.chance(15) || cmd < 0 || cmd >= len(st.NativeCommandBook) || strings.Contains(kind, "ai") {
		return e.intn(n)
	}
	rec := st.NativeCommandBook[cmd]
	flags, err := st.NativeCommandBaseFlags()
	if err != nil {
		return e.intn(n)
	}
	good := []int{}
	for i, u := range st.Units {
		cands, err := battle.NativeCommandTargets(st.W, st.H, battle.Cell{X: u.X, Y: u.Y}, rec.SelectionMode, rec.TargetCode, flags, st.Units)
		if err == nil && len(cands) > 0 {
			good = append(good, i)
		}
	}
	if len(good) == 0 {
		return e.intn(n)
	}
	return good[e.intn(len(good))]
}

// fieldCell 挑施放者第一段範圍內（field byte 不是 0xff）的一格，給游標確認用。
func (e *exporter) fieldCell(st *battle.State, a *battle.Unit, mode int, flags []byte) [2]int {
	field, err := battle.NativeCommandTargetFieldBytes(st.W, st.H, battle.Cell{X: a.X, Y: a.Y}, mode, 0, flags)
	if err != nil {
		return e.randomCell(st, 5)
	}
	cells := [][2]int{}
	for i, b := range field {
		if b != 0xff {
			cells = append(cells, [2]int{i % st.W, i / st.W})
		}
	}
	if len(cells) == 0 {
		return e.randomCell(st, 5)
	}
	return cells[e.intn(len(cells))]
}

// ---- 指令（戰場底稿）----

var cmdKinds = []string{
	"plan_damage", "execute_damage", "bound0_execute", "bound0_plan", "plan_damage_cursor", "plan_damage6_cursor",
	"plan_ai_damage_single", "plan_ai_damage",
	"heal_targets", "heal_exec", "heal_exec_native", "ai_heal_targets", "ai_heal_exec", "ai_heal_exec_native",
	"modifier_targets", "modifier_exec", "ai_modifier_targets", "ai_modifier_exec", "ai_modifier_plan", "modifier_validate",
	"clear_restore_targets", "clear_restore_exec", "ai2022_targets", "ai2022_plan", "player2022_plan",
	"cmd25_targets", "cmd25_exec", "app_targets", "app_exec",
	"c32_plan", "c32_exec", "c33_plan", "c33_exec", "c34_plan", "c34_exec", "c35_plan", "c35_exec",
}

func (e *exporter) commandFor(kind string) int {
	pick := func(ids ...int) int {
		if e.chance(5) {
			return []int{-1, 13, 21, 36}[e.intn(4)]
		}
		return ids[e.intn(len(ids))]
	}
	switch {
	case kind == "plan_damage" || kind == "execute_damage":
		return pick(0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12)
	case kind == "plan_damage_cursor":
		return pick(6, 10, 11, 12)
	case strings.HasPrefix(kind, "plan_ai_damage"):
		return pick(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)
	case strings.Contains(kind, "heal"):
		return pick(13, 14, 15, 16)
	case strings.Contains(kind, "modifier"):
		return pick(17, 18, 19)
	case strings.HasPrefix(kind, "clear_restore"):
		return pick(20, 21)
	case kind == "ai2022_targets" || kind == "ai2022_plan":
		return pick(20, 21, 22, 26, 27)
	case kind == "player2022_plan":
		return pick(20, 21, 22, 25, 26, 27)
	case strings.HasPrefix(kind, "app_"):
		return pick(22, 26, 27)
	case strings.HasPrefix(kind, "cmd25"):
		return 25
	case strings.HasPrefix(kind, "c3"):
		return int(kind[1]-'0')*10 + int(kind[2]-'0')
	}
	return 0
}

// tamper 在 script 中途改動一個單位，用來觸發交易邊界檢查。
func applyTamper(st *battle.State, op obj) {
	u := st.Units[op["unit"].(int)]
	v := op["value"].(int)
	switch op["field"] {
	case "HP":
		u.HP += v
	case "MP":
		u.MP += v
	case "Acted":
		u.Acted = !u.Acted
	case "AP":
		u.AP += v
	case "T":
		u.NativeTransient[(v&0xff)%6] ^= 1
	case "B5":
		u.NativeRecordByte5 ^= byte(v)
	}
}

func (e *exporter) tamperOp(st *battle.State, focus []int) obj {
	unit := e.intn(len(st.Units))
	if len(focus) > 0 && e.chance(80) {
		unit = focus[e.intn(len(focus))]
	}
	fields := []string{"HP", "MP", "Acted", "AP", "T", "B5"}
	return obj{"op": "tamper", "unit": unit, "field": fields[e.intn(len(fields))], "value": []int{-1, 1, 3, 0x80}[e.intn(4)]}
}

// perturb 以小機率插入竄改、重複或刪除一步。
func (e *exporter) perturb(st *battle.State, script []obj, focus []int, extra func() obj) []obj {
	if !e.chance(35) || len(script) == 0 {
		return script
	}
	pos := e.intn(len(script) + 1)
	var ins obj
	switch e.intn(4) {
	case 0, 1:
		ins = e.tamperOp(st, focus)
	case 2:
		ins = extra()
	default:
		if len(script) > 1 {
			return append(append([]obj(nil), script[:pos/2]...), script[pos/2+1:]...)
		}
		ins = extra()
	}
	out := append([]obj(nil), script[:pos]...)
	out = append(out, ins)
	return append(out, script[pos:]...)
}

func idxOf(units []*battle.Unit, p *battle.Unit) int {
	for i, u := range units {
		if u == p {
			return i
		}
	}
	return -1
}

func (e *exporter) walkSpec() obj {
	if e.chance(70) {
		return nil
	}
	return obj{"kind": []string{"extra", "reverse", "skip_last", "double"}[e.intn(4)], "extra": e.intn(4)}
}

// makeWalk 重現 Rust 端同一份演出亂數走法：每個目標前多走 extra 步、命中後再走一步。
func makeWalk(spec obj, start uint16) battle.NativeCommandDamageRNGWalk {
	if spec == nil {
		return nil
	}
	kind, extra := spec["kind"].(string), spec["extra"].(int)
	return func(count int, resolve func(index int, rng uint16) (uint16, bool, error)) (uint16, error) {
		order := []int{}
		for i := 0; i < count; i++ {
			order = append(order, i)
		}
		switch kind {
		case "reverse":
			for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
				order[i], order[j] = order[j], order[i]
			}
		case "skip_last":
			if len(order) > 0 {
				order = order[:len(order)-1]
			}
		case "double":
			if len(order) > 0 {
				order = append(order, order[0])
			}
		}
		rng := start
		for _, index := range order {
			for k := 0; k < extra; k++ {
				rng = fdother.NativeRNGStep(rng)
			}
			next, hit, err := resolve(index, rng)
			if err != nil {
				return rng, err
			}
			rng = next
			if hit {
				rng = fdother.NativeRNGStep(rng)
			}
		}
		return rng, nil
	}
}

func (e *exporter) genCommands(k int, perKind int) {
	for _, kind := range cmdKinds {
		fk := k
		if strings.Contains(kind, "ai") && strings.HasSuffix(e.fixtures[k].Name, "#legacy") && e.chance(85) {
			fk = k + 1
		}
		for c := 0; c < perKind; c++ {
			e.genCommand(fk, kind)
		}
	}
}

func (e *exporter) genCommand(k int, kind string) {
	fx := e.fixtures[k]
	n := len(fx.Units)
	if n == 0 {
		return
	}
	sp := e.baseSpec(k)
	cmd := e.commandFor(kind)
	aiKind := strings.Contains(kind, "ai")
	actor := e.pickActor(e.buildState(sp), kind, cmd)
	if actor >= 0 {
		set := obj{}
		if e.chance(85) {
			set["MP"], set["Acted"] = []int{255, 200, 60}[e.intn(3)], false
		}
		if e.chance(85) {
			set["OnField"], set["HP"] = true, e.rangeIn(1, 200)
			set["native_record_byte5"] = 0
		}
		if aiKind && e.chance(90) {
			set["Camp"] = 2
			set["native_record_byte6"], set["has_native_record_byte6"] = []int{0, 1, 0, 1, 0, 1, 0, 1, 2}[e.intn(9)], true
			set["HasNativeMapPresentation"] = e.chance(97)
		}
		if strings.HasPrefix(kind, "c3") && e.chance(90) {
			set["NativeRecordClass"], set["HasNativeRecordClass"] = 19, e.chance(97)
			set["BattleFig"], set["HasBattleFig"] = []int{4, 5, 6, 7, 20, 4, 5, 3}[e.intn(8)], e.chance(97)
		}
		if strings.HasPrefix(kind, "plan_damage_cursor") || kind == "plan_damage6_cursor" {
			set["Camp"] = []int{0, 1, 2, 0, 0, 1}[e.intn(6)]
		}
		if e.chance(12) {
			set["Camp"], set["NativeIdentity"], set["HasNativeIdentity"] = 0, e.intn(32), true
			sp["balance"] = e.randomBalance()
		}
		sp["edits"] = append(sp["edits"].([]obj), obj{"unit": actor, "set": set})
	}
	st := e.buildState(sp)
	a := unitOrNil(st, actor)
	flags, _ := st.NativeCommandBaseFlags()
	confirmed := e.intn(n)
	var rec battle.NativeCommandRecord
	if cmd >= 0 && cmd < len(st.NativeCommandBook) {
		rec = st.NativeCommandBook[cmd]
		confirmed = e.confirmedFor(st, actor, rec.SelectionMode, rec.TargetCode, flags)
	} else if e.chance(5) {
		confirmed = -1
	}
	if kind == "plan_ai_damage_single" && e.chance(85) {
		others := []int{}
		for i, u := range st.Units {
			if u.Camp != battle.Enemy && u.OnField && u.HP > 0 {
				others = append(others, i)
			}
		}
		if len(others) > 0 {
			confirmed = others[e.intn(len(others))]
		}
	}
	cf := unitOrNil(st, confirmed)
	cursor := e.randomCell(st, 5)
	if a != nil && e.chance(70) {
		cursor = e.fieldCell(st, a, rec.SelectionMode, flags)
	} else if cf != nil && e.chance(50) {
		cursor = [2]int{cf.X, cf.Y}
	}
	origin := e.randomCell(st, 5)
	if aiKind && len(st.Units) > 0 && e.chance(60) {
		u := st.Units[e.intn(len(st.Units))]
		origin = [2]int{u.X, u.Y}
	}
	rngState := uint16(e.u16())
	seed := e.seed()
	nilRNG := e.chance(5)
	var rng *mrand.Rand
	if !nilRNG {
		rng = mrand.New(mrand.NewSource(seed))
	}
	walk := e.walkSpec()
	in := obj{"state": sp, "kind": kind, "actor": actor, "confirmed": confirmed, "command": cmd, "cursor": cursor,
		"origin": origin, "rng": int(rngState), "seed": seed, "nil_rng": nilRNG, "walk": walk}
	before := snapshot(st.Units)
	res := obj{}
	units := st.Units
	setRes := func(v any, err error) { res["result"], res["err"] = conv(units, v), errv(err) }
	focusOf := func(targets []*battle.Unit) []int {
		out := []int{}
		if a != nil {
			out = append(out, actor)
		}
		for _, t := range targets {
			if i := idxOf(units, t); i >= 0 {
				out = append(out, i)
			}
		}
		return out
	}
	var script []obj
	runScript := func(apply func(op obj) error) {
		errs := []any{}
		for _, op := range script {
			if op["op"] == "tamper" {
				applyTamper(st, op)
				errs = append(errs, nil)
				continue
			}
			errs = append(errs, errv(apply(op)))
		}
		res["script_errs"] = errs
	}
	cursorCell, originCell := cellFrom(cursor), cellFrom(origin)
	switch kind {
	case "plan_damage", "plan_damage_cursor", "plan_damage6_cursor", "plan_ai_damage_single", "plan_ai_damage", "bound0_plan":
		var plan *battle.NativeCommandDamagePlan
		var err error
		w := makeWalk(walk, rngState)
		switch kind {
		case "plan_damage":
			plan, err = st.PlanNativeCommandDamageWalk(a, cf, cmd, st.NativeCommandResistances, rngState, w)
		case "plan_damage_cursor":
			plan, err = st.PlanNativeCommandDamageAtCursor(a, cursorCell, cmd, st.NativeCommandResistances, rngState, w)
		case "plan_damage6_cursor":
			plan, err = st.PlanNativeCommand6DamageAtCursor(a, cursorCell, st.NativeCommandResistances, rngState, w)
		case "plan_ai_damage_single":
			plan, err = st.PlanNativeAICommandDamageSingleTarget(a, cf, cmd, st.NativeCommandResistances, rngState)
		case "plan_ai_damage":
			plan, err = st.PlanNativeAICommandDamageWalk(a, originCell, cmd, st.NativeCommandResistances, rngState, w)
		default:
			plan, err = st.PlanBoundNativeCommand0(a, cf, rngState)
		}
		setRes(plan, err)
		if plan != nil {
			targets := []*battle.Unit{}
			script = append(script, obj{"op": "mp"})
			for i, r := range plan.Results {
				targets = append(targets, r.Target)
				if e.chance(30) {
					script = append(script, obj{"op": "stage", "index": i, "stage": plan.DamageStages})
				} else {
					for s := 1; s <= plan.DamageStages; s++ {
						script = append(script, obj{"op": "stage", "index": i, "stage": s})
					}
				}
			}
			script = append(script, obj{"op": "complete"})
			script = e.perturb(st, script, focusOf(targets), func() obj {
				return obj{"op": "stage", "index": e.rangeIn(-1, len(plan.Results)), "stage": e.rangeIn(0, plan.DamageStages+1)}
			})
			runScript(func(op obj) error {
				switch op["op"] {
				case "mp":
					return battle.ApplyNativeCommandDamageMP(plan)
				case "stage":
					return battle.ApplyNativeCommandDamageStage(plan, op["index"].(int), op["stage"].(int))
				}
				return battle.CompleteNativeCommandDamage(plan)
			})
			res["plan_after"] = conv(units, plan)
		}
	case "execute_damage":
		r, next, err := st.ExecuteNativeCommandDamage(a, cf, cmd, st.NativeCommandResistances, rngState)
		setRes(r, err)
		res["rng"] = next
	case "bound0_execute":
		r, next, err := st.ExecuteBoundNativeCommand0(a, cf, rngState)
		setRes(r, err)
		res["rng"] = next
	case "heal_targets":
		r, err := st.NativeCommandHealTargets(a, cf, cmd)
		setRes(r, err)
	case "heal_exec":
		r, err := st.ExecuteNativeCommandHeal(a, cf, cmd, rng)
		setRes(r, err)
	case "heal_exec_native":
		r, next, err := st.ExecuteNativeCommandHealNative(a, cf, cmd, rngState)
		setRes(r, err)
		res["rng"] = next
	case "ai_heal_targets":
		r, err := st.NativeAICommandHealTargets(a, originCell, cmd)
		setRes(r, err)
	case "ai_heal_exec":
		r, err := st.ExecuteNativeAICommandHeal(a, originCell, cmd, rng)
		setRes(r, err)
	case "ai_heal_exec_native":
		r, next, err := st.ExecuteNativeAICommandHealNative(a, originCell, cmd, rngState)
		setRes(r, err)
		res["rng"] = next
	case "modifier_targets":
		r, err := st.NativeCommandModifierTargets(a, cf, cmd)
		setRes(r, err)
	case "modifier_exec":
		r, err := st.ExecuteNativeCommandModifier(a, cf, cmd, rngState)
		setRes(r, err)
	case "ai_modifier_targets":
		r, err := st.NativeAICommandModifierTargets(a, originCell, cmd)
		setRes(r, err)
	case "ai_modifier_exec":
		r, err := st.ExecuteNativeAICommandModifier(a, originCell, cmd, rngState)
		setRes(r, err)
	case "modifier_validate":
		idx := []int{}
		for i := e.intn(4); i > 0; i-- {
			idx = append(idx, e.intn(n))
		}
		in["targets"] = idx
		targets := []*battle.Unit{}
		for _, i := range idx {
			targets = append(targets, units[i])
		}
		res["err"] = errv(st.ValidateNativeCommandModifierTargets(a, targets, cmd))
	case "ai_modifier_plan":
		plan, err := st.PlanNativeAICommandModifier(a, originCell, cmd, rngState)
		setRes(plan, err)
		if plan != nil {
			script = []obj{{"op": "publish"}, {"op": "complete"}}
			if e.chance(30) {
				script = []obj{{"op": "publish"}, {"op": "abort"}, {"op": "publish"}, {"op": "complete"}, {"op": "abort"}}
			}
			script = e.perturb(st, script, focusOf(plan.Targets), func() obj { return obj{"op": []string{"publish", "complete", "abort"}[e.intn(3)]} })
			runScript(func(op obj) error {
				switch op["op"] {
				case "publish":
					return battle.PublishNativeAICommandModifier(plan)
				case "complete":
					return battle.CompleteNativeAICommandModifier(plan)
				}
				return battle.AbortNativeAICommandModifier(plan)
			})
		}
	case "clear_restore_targets":
		t, off, err := st.NativeCommandClearRestoreTargets(a, cf, cmd)
		setRes(t, err)
		res["offset"] = off
	case "clear_restore_exec":
		r, err := st.ExecuteNativeCommandClearRestore(a, cf, cmd, rng)
		setRes(r, err)
	case "ai2022_targets":
		r, err := st.NativeAICommand2022Targets(a, originCell, cmd)
		setRes(r, err)
	case "ai2022_plan", "player2022_plan":
		var plan *battle.NativeAICommand2022Plan
		var err error
		if kind == "ai2022_plan" {
			plan, err = st.PlanNativeAICommand2022(a, originCell, cmd, rngState)
		} else {
			plan, err = st.PlanNativePlayerCommand2022(a, cf, cmd, rngState)
		}
		setRes(plan, err)
		if plan != nil {
			script = []obj{{"op": "publish"}, {"op": "complete"}}
			if e.chance(30) {
				script = []obj{{"op": "abort"}, {"op": "publish"}, {"op": "abort"}, {"op": "publish"}, {"op": "complete"}}
			}
			script = e.perturb(st, script, focusOf(plan.Targets), func() obj { return obj{"op": []string{"publish", "complete", "abort"}[e.intn(3)]} })
			runScript(func(op obj) error {
				switch op["op"] {
				case "publish":
					return battle.PublishNativeAICommand2022(plan)
				case "complete":
					return battle.CompleteNativeAICommand2022(plan)
				}
				return battle.AbortNativeAICommand2022(plan)
			})
		}
	case "cmd25_targets":
		r, err := st.NativeCommand25Targets(a, cf)
		setRes(r, err)
	case "cmd25_exec":
		r, err := st.ExecuteNativeCommand25(a, cf)
		setRes(r, err)
	case "app_targets":
		t, off, err := st.NativeCommandApplicationTargets(a, cf, cmd)
		setRes(t, err)
		res["offset"] = off
	case "app_exec":
		r, err := st.ExecuteNativeCommandApplication(a, cf, cmd, rng)
		setRes(r, err)
	case "c32_plan":
		plan, err := st.PlanNativeCompoundCommand32(a, cf, rngState)
		setRes(plan, err)
		if plan != nil {
			targets := []*battle.Unit{}
			for i, t := range plan.Result.Targets {
				targets = append(targets, t.Target)
				script = append(script, obj{"op": "apply", "index": i})
			}
			script = append(script, obj{"op": "complete"})
			if e.chance(25) {
				script = append(script[:len(script)-1], obj{"op": "abort"}, obj{"op": "complete"})
			}
			script = e.perturb(st, script, focusOf(targets), func() obj {
				return obj{"op": "apply", "index": e.rangeIn(-1, len(plan.Result.Targets))}
			})
			runScript(func(op obj) error {
				switch op["op"] {
				case "apply":
					return battle.ApplyNativeCompoundCommand32Target(plan, op["index"].(int))
				case "complete":
					return battle.CompleteNativeCompoundCommand32(plan)
				}
				return battle.AbortNativeCompoundCommand32(plan)
			})
		}
	case "c32_exec":
		r, err := st.ExecuteNativeCompoundCommand32(a, cf, rngState)
		setRes(r, err)
	case "c33_plan":
		plan, err := st.PlanNativeCompoundCommand33(a, cf, rngState)
		setRes(plan, err)
		if plan != nil {
			targets := []*battle.Unit{}
			for _, t := range plan.Result.Targets {
				targets = append(targets, t.Target)
			}
			script = []obj{{"op": "publish"}, {"op": "complete"}}
			if e.chance(30) {
				script = []obj{{"op": "abort"}, {"op": "publish"}, {"op": "abort"}, {"op": "publish"}, {"op": "complete"}}
			}
			script = e.perturb(st, script, focusOf(targets), func() obj { return obj{"op": []string{"publish", "complete", "abort"}[e.intn(3)]} })
			runScript(func(op obj) error {
				switch op["op"] {
				case "publish":
					return battle.PublishNativeCompoundCommand33(plan)
				case "complete":
					return battle.CompleteNativeCompoundCommand33(plan)
				}
				return battle.AbortNativeCompoundCommand33(plan)
			})
		}
	case "c33_exec":
		r, err := st.ExecuteNativeCompoundCommand33(a, cf, rngState)
		setRes(r, err)
	case "c34_plan", "c35_plan":
		var publish func(int) error
		var complete, abort func() error
		var targets []*battle.Unit
		if kind == "c34_plan" {
			plan, err := st.PlanNativeCompoundCommand34(a, cf, rngState)
			setRes(plan, err)
			if plan != nil {
				publish = func(s int) error { return battle.PublishNativeCompoundCommand34Stage(plan, s) }
				complete = func() error { return battle.CompleteNativeCompoundCommand34(plan) }
				abort = func() error { return battle.AbortNativeCompoundCommand34(plan) }
				for _, s := range plan.Result.StageStates[0] {
					targets = append(targets, s.Target)
				}
			}
		} else {
			plan, err := st.PlanNativeCompoundCommand35(a, cf, rngState)
			setRes(plan, err)
			if plan != nil {
				publish = func(s int) error { return battle.PublishNativeCompoundCommand35Stage(plan, s) }
				complete = func() error { return battle.CompleteNativeCompoundCommand35(plan) }
				abort = func() error { return battle.AbortNativeCompoundCommand35(plan) }
				for _, s := range plan.Result.StageStates[0] {
					targets = append(targets, s.Target)
				}
			}
		}
		if publish != nil {
			script = []obj{{"op": "publish", "stage": 0}, {"op": "publish", "stage": 1}, {"op": "publish", "stage": 2}, {"op": "complete"}}
			if e.chance(30) {
				script = []obj{{"op": "publish", "stage": 0}, {"op": "publish", "stage": 1}, {"op": "abort"}, {"op": "publish", "stage": 0}, {"op": "complete"}, {"op": "abort"}}
			}
			script = e.perturb(st, script, focusOf(targets), func() obj { return obj{"op": "publish", "stage": e.rangeIn(-1, 3)} })
			runScript(func(op obj) error {
				switch op["op"] {
				case "publish":
					return publish(op["stage"].(int))
				case "complete":
					return complete()
				}
				return abort()
			})
		}
	case "c34_exec":
		r, err := st.ExecuteNativeCompoundCommand34(a, cf, rngState)
		setRes(r, err)
	case "c35_exec":
		r, err := st.ExecuteNativeCompoundCommand35(a, cf, rngState)
		setRes(r, err)
	default:
		panic(kind)
	}
	if script != nil {
		in["script"] = script
	}
	res["changed"] = changed(before, st.Units)
	e.add("command", in, res)
}

// genNilPlans 對每個交易邊界函式傳 nil plan。
func (e *exporter) genNilPlans() {
	errs := obj{
		"damage_mp":       errv(battle.ApplyNativeCommandDamageMP(nil)),
		"damage_stage":    errv(battle.ApplyNativeCommandDamageStage(nil, 0, 1)),
		"damage_complete": errv(battle.CompleteNativeCommandDamage(nil)),
		"mod_publish":     errv(battle.PublishNativeAICommandModifier(nil)),
		"mod_complete":    errv(battle.CompleteNativeAICommandModifier(nil)),
		"mod_abort":       errv(battle.AbortNativeAICommandModifier(nil)),
		"t2022_publish":   errv(battle.PublishNativeAICommand2022(nil)),
		"t2022_complete":  errv(battle.CompleteNativeAICommand2022(nil)),
		"t2022_abort":     errv(battle.AbortNativeAICommand2022(nil)),
		"c32_apply":       errv(battle.ApplyNativeCompoundCommand32Target(nil, 0)),
		"c32_complete":    errv(battle.CompleteNativeCompoundCommand32(nil)),
		"c32_abort":       errv(battle.AbortNativeCompoundCommand32(nil)),
		"c33_publish":     errv(battle.PublishNativeCompoundCommand33(nil)),
		"c33_complete":    errv(battle.CompleteNativeCompoundCommand33(nil)),
		"c33_abort":       errv(battle.AbortNativeCompoundCommand33(nil)),
		"c34_publish":     errv(battle.PublishNativeCompoundCommand34Stage(nil, 0)),
		"c34_complete":    errv(battle.CompleteNativeCompoundCommand34(nil)),
		"c34_abort":       errv(battle.AbortNativeCompoundCommand34(nil)),
		"c35_publish":     errv(battle.PublishNativeCompoundCommand35Stage(nil, 0)),
		"c35_complete":    errv(battle.CompleteNativeCompoundCommand35(nil)),
		"c35_abort":       errv(battle.AbortNativeCompoundCommand35(nil)),
	}
	_, rerr := battle.ApplyNativeCommandRestore(nil, 1, mrand.New(mrand.NewSource(1)))
	errs["restore_nil"] = errv(rerr)
	e.add("nil_plans", obj{}, obj{"errs": errs})
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "需要 -out")
		os.Exit(2)
	}
	tmp, err := os.MkdirTemp("", "fd2-golden-rules-b-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x52554c42)), tmp: tmp}
	real := e.genReal()
	realCount := len(e.fixtures)
	e.genSyntheticFixtures(24)
	e.genGeometry()
	e.genDamage()
	e.genPaths()
	e.genMapView()
	e.genBalance()
	e.genMove()
	e.genDeath()
	for k := range e.fixtures {
		e.genTargets(k)
		per := 1
		if k >= realCount {
			per = 3
		}
		e.genCommands(k, per)
	}
	e.genNilPlans()
	doc := obj{"schema": "fd2-golden-rules-b/1", "seed": *seed, "real": real, "fixtures": e.fixtureList(), "cases": e.cases}
	data, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		panic(err)
	}
	ops := map[string][2]int{}
	for _, c := range e.cases {
		key := c.Op
		if c.Op == "command" {
			key = "command/" + c.Input["kind"].(string)
		}
		v := ops[key]
		v[0]++
		if errStr, isErr := c.Result["err"].(string); !isErr || errStr == "" {
			v[1]++
		}
		ops[key] = v
	}
	names := []string{}
	for name := range ops {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("%s=%d(ok %d) ", name, ops[name][0], ops[name][1])
	}
	fmt.Printf("total=%d\n", len(e.cases))
}
