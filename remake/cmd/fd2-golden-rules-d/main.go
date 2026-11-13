// fd2-golden-rules-d 輸出 internal/battle「D 組」（敵我 AI）的黃金差分資料，供 Rust 改寫
// remake-rs/crates/fd2-rules 逐欄比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 battle 的公開 API，不改任何遊戲行為。
//   - 真實輸入：remake/assets/maps/map*/map*_units.json 全部以 Load＋BindNativeFutureItemRows 載入
//     （路徑排序後保存清單與 SHA-256），補齊原版 provenance 後成為戰場底稿；spells.json、
//     native_movement_cost_rows.json、native_item_effect_rows.json 與各地圖 map.json 也記路徑與 SHA-256。
//   - 產生輸入：以固定種子的 PCG 產生單位修改、指令參數、原始記錄、事件格網與 math/rand 種子，
//     連同結果寫進同一份 JSON；Rust 端不需重現產生器，只重做同一串呼叫。
//
// 涵蓋 D 組不經過 E 組（戰鬥、成長與法術）的公開函式：0x14ef0 派發、0x1598a／0x1567e 評分、
// 指令群組與法術評分、物理候選與評分、物品來源、閒置恢復、mode5／mode11、模式記錄與寫入、
// 階段診斷、玩家移動與 command24 家族。經由 NextAIPlan 或地形 AP/DP 表的 runtime 規劃整合後才能涵蓋。
//
// 戰場一律由「底稿＋修改」組成：底稿單位是 json.Marshal(battle.Unit) 的物件，修改是逐單位合併的
// JSON 鍵（Go 欄位名）。Go 與 Rust 都把合併後的物件解碼成 Unit，因此兩邊看到相同的單位，未匯出
// 欄位兩邊都是零值。結果裡的 *Unit 一律寫成 State.Units 的索引（nil 為 -1），結構以 Go 欄位名
// 輸出（未匯出欄位省略）。錯誤欄位是 Go err.Error() 原文。
//
// Go map 迭代順序會影響兩處錯誤訊息（ScoreNativePhysicalAttackCandidate／Select 的多欄越界、
// BuildNativeAIPhaseDiagnosticPlan 的多筆不合格輸入）；每個案例最多放一個不合法欄位或輸入。
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
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

	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

type obj = map[string]any

const recSize = 0x50
const rowSize = battle.NativeItemEffectRowSize

type caseJSON struct {
	Index  int    `json:"index"`
	Op     string `json:"op"`
	Input  obj    `json:"input"`
	Result obj    `json:"result"`
}

// fixture 是一份戰場底稿：地圖大小、地形與事件緩衝，以及單位物件。
type fixture struct {
	Name    string
	W, H    int
	Events  []byte
	Terrain []byte
	Control []byte
	Grid    []byte
	HasGrid bool
	Chests  [16]battle.NativeChestControl
	HasChst bool
	Units   []obj
	Records []byte // 補齊 provenance 後 NativeAIScoringRecords 的結果；nil 表示無法產生
}

type exporter struct {
	rng      *rand.Rand
	cases    []caseJSON
	fixtures []fixture
	book     []battle.NativeCommandRecord
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

func (e *exporter) rangeIn(lo, hi int) int { return lo + e.rng.IntN(hi-lo+1) }

func (e *exporter) chance(pct int) bool { return e.rng.IntN(100) < pct }

func (e *exporter) bytesN(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(e.rng.IntN(256))
	}
	return b
}

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

// ---- 單位物件與戰場組裝 ----

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

func unitFromObj(m obj) *battle.Unit {
	data, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	var u battle.Unit
	if err := json.Unmarshal(data, &u); err != nil {
		panic(err)
	}
	return &u
}

// units 以底稿加上 edits（[]{unit, set}）組出單位。
func (e *exporter) units(k int, edits []obj) []*battle.Unit {
	fx := e.fixtures[k]
	maps := make([]obj, len(fx.Units))
	for i, u := range fx.Units {
		maps[i] = cloneObj(u)
	}
	for _, ed := range edits {
		idx := ed["unit"].(int)
		for key, value := range ed["set"].(obj) {
			maps[idx][key] = value
		}
	}
	out := make([]*battle.Unit, len(maps))
	for i, m := range maps {
		out[i] = unitFromObj(m)
	}
	return out
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

// stateSpec 是 State 的可變欄位；Rust 端以同一份規格建出同一個 State。
type stateSpec struct {
	fixture  int
	edits    []obj
	eventSt  map[int]int // NativeEventState 覆寫
	fieldRaw []byte      // nil 表示 HasNativeFieldControlState=false
	grid     []byte      // nil 表示沿用底稿
	noGrid   bool
	mapView  bool
	book     []battle.NativeCommandRecord
	withRows bool // 綁定移動成本列與物品效果列
}

func (sp stateSpec) json() obj {
	out := obj{"fixture": sp.fixture, "edits": sp.edits, "map_view": sp.mapView, "rows": sp.withRows, "no_grid": sp.noGrid}
	ev := [][2]int{}
	for k, v := range sp.eventSt {
		ev = append(ev, [2]int{k, v})
	}
	sort.Slice(ev, func(i, j int) bool { return ev[i][0] < ev[j][0] })
	out["event_state"] = ev
	if sp.fieldRaw != nil {
		out["field_raw"] = hexs(sp.fieldRaw)
	}
	if sp.grid != nil {
		out["grid"] = hexs(sp.grid)
	}
	if sp.book != nil {
		out["book"] = conv(nil, sp.book)
	}
	return out
}

func (e *exporter) state(sp stateSpec) *battle.State {
	fx := e.fixtures[sp.fixture]
	st := &battle.State{
		W: fx.W, H: fx.H, Units: e.units(sp.fixture, sp.edits),
		NativeCompositionEventBytes: append([]byte(nil), fx.Events...),
		NativeTerrainMoveCodes:      append([]byte(nil), fx.Terrain...),
		NativeTerrainControl:        append([]byte(nil), fx.Control...),
		NativeChestControls:         fx.Chests,
		HasNativeChestControlState:  fx.HasChst,
		NativeCommandBook:           append([]battle.NativeCommandRecord(nil), e.book...),
	}
	if sp.book != nil {
		st.NativeCommandBook = append([]battle.NativeCommandRecord(nil), sp.book...)
	}
	if !sp.noGrid {
		st.NativeMapEventGrid, st.HasNativeMapEventGrid = append([]byte(nil), fx.Grid...), fx.HasGrid
		if sp.grid != nil {
			st.NativeMapEventGrid, st.HasNativeMapEventGrid = append([]byte(nil), sp.grid...), true
		}
	}
	for k, v := range sp.eventSt {
		st.NativeEventState[k] = byte(v)
	}
	if sp.fieldRaw != nil {
		st.NativeFieldControlRaw, st.HasNativeFieldControlState = append([]byte(nil), sp.fieldRaw...), true
	}
	st.HasNativeMapViewState = sp.mapView
	if sp.withRows {
		if err := st.BindNativeMovementCostRows(e.costRows); err != nil {
			panic(err)
		}
		// 不經 BindNativeFutureItemRows 的建構器補值：底稿單位已經材料化過，直接存表。
		if err := st.BindNativeFutureItemRows(e.itemRows); err != nil {
			panic(err)
		}
	}
	return st
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
	for _, p := range []string{"assets/spells.json", "assets/data/native_movement_cost_rows.json",
		"assets/data/native_item_effect_rows.json", "assets/maps/native_turn_event_controls.json"} {
		record(p)
	}
	if e.book, err = battle.LoadNativeCommandRecords("assets/spells.json"); err != nil {
		panic(err)
	}
	if e.costRows, err = battle.LoadNativeMovementCostRows("assets/data/native_movement_cost_rows.json"); err != nil {
		panic(err)
	}
	if e.itemRows, err = battle.LoadNativeItemEffectRowPrefix("assets/data/native_item_effect_rows.json"); err != nil {
		panic(err)
	}
	withRecords := 0
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
		fx := fixture{
			Name: path, W: st.W, H: st.H, Events: st.NativeCompositionEventBytes,
			Terrain: st.NativeTerrainMoveCodes, Control: st.NativeTerrainControl,
			Grid: st.NativeMapEventGrid, HasGrid: st.HasNativeMapEventGrid,
			Chests: st.NativeChestControls, HasChst: st.HasNativeChestControlState,
		}
		for _, u := range st.Units {
			_ = u.MaterializeNativeMapPresentation()
			fx.Units = append(fx.Units, e.rawify(unitObj(u), st.W, st.H))
		}
		units := make([]*battle.Unit, len(fx.Units))
		for i, m := range fx.Units {
			units[i] = unitFromObj(m)
		}
		if records, err := battle.NativeAIScoringRecords(units); err == nil {
			fx.Records = records
			withRecords++
		}
		e.fixtures = append(e.fixtures, fx)
	}
	rows := []string{}
	for _, r := range e.costRows {
		rows = append(rows, hexs(r))
	}
	fmt.Fprintf(os.Stderr, "地圖 %d 張，可產生原始記錄 %d 張\n", len(paths), withRecords)
	return obj{"files": files, "maps": paths, "book": conv(nil, e.book), "cost_rows": rows, "item_rows": hexs(e.itemRows)}
}

// rawify 補齊 NativeAIScoringRecords 需要的原版 provenance（地圖呈現座標、+5/+6、模式位元組、
// 戰鬥圖、HP/MP 字），已有的欄位不動。
func (e *exporter) rawify(u obj, w, h int) obj {
	camp := num(u["Camp"])
	if !boolOf(u["HasNativeMapPresentation"]) {
		u["NativeMapPresentation"] = obj{"X": num(u["X"]) & 0xff, "Y": num(u["Y"]) & 0xff, "Pose": 0, "Motion": 0}
		u["HasNativeMapPresentation"] = true
	}
	if !boolOf(u["has_native_record_byte5"]) {
		b5 := 0
		if !boolOf(u["OnField"]) || num(u["HP"]) <= 0 {
			b5 = 1
		}
		u["native_record_byte5"], u["has_native_record_byte5"] = b5, true
	}
	if !boolOf(u["has_native_record_byte6"]) {
		u["native_record_byte6"], u["has_native_record_byte6"] = []int{2, 1, 0}[camp%3], true
	}
	if !boolOf(u["HasBattleFig"]) {
		u["BattleFig"], u["HasBattleFig"] = []int{4, 5, 6, 7, 20, 3}[e.intn(6)], true
	}
	for _, key := range []string{"native_record_byte34", "native_record_byte35", "native_record_byte36"} {
		if !boolOf(u["has_"+key]) {
			u[key] = e.intn(16)
		}
		u["has_"+key] = true
	}
	if !boolOf(u["has_native_record_word42"]) {
		u["native_record_word42"], u["has_native_record_word42"] = num(u["MaxHP"])&0xffff, true
	}
	if !boolOf(u["has_native_record_word46"]) {
		u["native_record_word46"], u["has_native_record_word46"] = num(u["MaxMP"])&0xffff, true
	}
	return u
}

func (e *exporter) fixtureList() []obj {
	out := []obj{}
	for _, fx := range e.fixtures {
		chests := [][2]int{}
		for _, c := range fx.Chests {
			chests = append(chests, [2]int{int(c.RawType), int(c.Value)})
		}
		var records any
		if fx.Records != nil {
			records = hexs(fx.Records)
		}
		out = append(out, obj{
			"name": fx.Name, "w": fx.W, "h": fx.H, "events": hexs(fx.Events), "terrain": hexs(fx.Terrain),
			"control": hexs(fx.Control), "grid": hexs(fx.Grid), "has_grid": fx.HasGrid,
			"chests": chests, "has_chest": fx.HasChst, "units": fx.Units, "records": records,
		})
	}
	return out
}

// ---- 合成戰場（command24 家族、小地圖 mode5） ----

func (e *exporter) genSyntheticFixtures(n int) {
	for k := 0; k < n; k++ {
		w, h := e.rangeIn(2, 9), e.rangeIn(1, 7)
		fx := fixture{Name: fmt.Sprintf("synthetic-%d", k), W: w, H: h, Events: make([]byte, w*h), Terrain: make([]byte, w*h)}
		for i := range fx.Events {
			if e.chance(10) {
				fx.Events[i] = byte(e.intn(256))
			}
			fx.Terrain[i] = byte(e.intn(4))
		}
		fx.Control = make([]byte, 4*4)
		for t := 0; t < 4; t++ {
			fx.Control[4*t] = []byte{0x20, 0x00, 0x40, 0x60, 0x21}[e.intn(5)]
		}
		fx.Grid, fx.HasGrid = e.randomGrid(w, h, 4), true
		for slot := range fx.Chests {
			fx.Chests[slot] = battle.NativeChestControl{RawType: byte(e.intn(4)), Value: uint16(e.intn(0x10000))}
		}
		fx.HasChst = e.chance(70)
		count := e.rangeIn(2, 7)
		for i := 0; i < count; i++ {
			camp := battle.Camp(e.intn(3))
			u := &battle.Unit{
				Camp: camp, ClassID: []int{1, 5, 12, 19, 25}[e.intn(5)], Lv: e.rangeIn(1, 40),
				MaxHP: e.rangeIn(1, 300), MP: e.rangeIn(0, 60), MaxMP: 60, AP: e.rangeIn(0, 300), DP: e.rangeIn(0, 300),
				MV: e.rangeIn(1, 5), X: e.intn(w), Y: e.intn(h), OnField: e.chance(90), Acted: e.chance(10),
				InventorySlots:       []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
				NativeInventoryFlags: []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80},
			}
			u.HP = e.rangeIn(0, u.MaxHP)
			u.NativeRecordByte5, u.HasNativeRecordByte5 = byte([]int{0, 0, 0, 0, 0, 1}[e.intn(6)]), e.chance(95)
			if u.HP == 0 {
				u.NativeRecordByte5 = 1
			}
			u.NativeRecordByte6, u.HasNativeRecordByte6 = byte([]int{2, 1, 0}[int(camp)]), true
			u.NativeMapPresentation = battle.NativeMapPresentationState{X: byte(u.X), Y: byte(u.Y)}
			u.HasNativeMapPresentation = true
			u.BattleFig, u.HasBattleFig = []int{19, 20, 4, 5}[e.intn(4)], true
			u.NativeRecordByte34, u.HasNativeRecordByte34 = byte(e.intn(16)), true
			u.NativeRecordByte35, u.HasNativeRecordByte35 = byte(e.intn(16)), true
			u.NativeRecordByte36, u.HasNativeRecordByte36 = byte(e.intn(16)), true
			u.NativeRecordWord42, u.HasNativeRecordWord42 = uint16(u.MaxHP), true
			u.NativeRecordWord46, u.HasNativeRecordWord46 = uint16(u.MaxMP), true
			u.NativeRecordByte3D, u.HasNativeRecordByte3D = byte(e.intn(4)), e.chance(80)
			u.HasNativeRecordDeathEffect = e.chance(85)
			fx.Units = append(fx.Units, unitObj(u))
		}
		units := make([]*battle.Unit, len(fx.Units))
		for i, m := range fx.Units {
			units[i] = unitFromObj(m)
		}
		if records, err := battle.NativeAIScoringRecords(units); err == nil {
			fx.Records = records
		}
		e.fixtures = append(e.fixtures, fx)
	}
}

// randomGrid 產生 0x53A51 格式的事件格網：表頭 W/H、每格 tile word（低 10 bit 落在 tiles 之內）與事件 byte。
func (e *exporter) randomGrid(w, h, tiles int) []byte {
	grid := make([]byte, 4+4*w*h)
	grid[0], grid[2] = byte(w), byte(h)
	for i := 0; i < w*h; i++ {
		off := 4 + 4*i
		binary.LittleEndian.PutUint16(grid[off:], uint16(e.intn(tiles)))
		if e.chance(25) {
			grid[off+2] = byte(e.intn(6))
		}
	}
	return grid
}

// ---- 純函式案例 ----

func (e *exporter) genDispatch() {
	for n := 0; n < 600; n++ {
		score := func() int32 { return int32([]int{-2, 0, 3, 5, 6, 7, 8, 9, 12, 18}[e.intn(10)]) }
		in := battle.NativeAI14EF0Input{
			ScoreC4F: score(), ScoreC23: score(), ScoreC33: score(),
			CommandID: int32(e.rangeIn(-1, 36)), CommandWord: uint16(e.intn(0x10000)),
			Record34: byte(e.intn(256)), ActorWord48: uint16(e.intn(300)), TargetWord4A: uint16(e.intn(300)),
			HasRawScoreC4F: e.chance(95), HasRawScoreC23: e.chance(95), HasRawScoreC33: e.chance(95),
			HasRawCommandID: e.chance(90), HasRawCommandWord: e.chance(90), HasRawRecord34: e.chance(95),
			HasRawActorWord48: e.chance(95), HasRawTargetWord4A: e.chance(95),
		}
		if e.chance(30) {
			in.ScoreC23 = in.ScoreC4F
		}
		if e.chance(20) {
			in.ScoreC33 = in.ScoreC4F
		}
		tail, err := battle.SelectNativeAI14EF0Tail(in)
		decision, derr := battle.PlanNativeAI14EF0Dispatch(in)
		e.add("dispatch14ef0", obj{"in": conv(nil, in)}, obj{
			"tail": int(tail), "err": errv(err), "decision": conv(nil, decision), "decision_err": errv(derr),
		})
	}
}

func (e *exporter) genSpellCandidate() {
	for n := 0; n < 200; n++ {
		count := e.intn(6)
		cands := []battle.NativeAISpellCandidate{}
		for i := 0; i < count; i++ {
			cands = append(cands, battle.NativeAISpellCandidate{
				CommandID: e.intn(36), CommandValue: e.intn(5), X: e.intn(20), Y: e.intn(20),
				Score: e.intn(4), TargetIndices: e.bytesN(e.intn(4)),
			})
		}
		best, ok := battle.SelectNativeAISpellCandidate(cands)
		e.add("spell_candidate", obj{"candidates": conv(nil, cands)}, obj{"best": conv(nil, best), "ok": ok})
	}
}

// randomRecords 產生 n 筆原始記錄，HP、+5／+6、+8 與旗標位元組隨機。
func (e *exporter) randomRecords(n int) []byte {
	records := make([]byte, n*recSize)
	for i := 0; i < n; i++ {
		r := records[i*recSize : (i+1)*recSize]
		r[0], r[1] = byte(e.intn(8)), byte(e.intn(6))
		r[5] = byte([]int{0, 0, 0, 1, 0x80}[e.intn(5)])
		r[6] = byte(e.intn(3))
		r[7] = byte([]int{0, 3, 0x1c}[e.intn(3)])
		r[8] = byte(e.intn(3))
		for _, off := range []int{0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27} {
			if e.chance(30) {
				r[off] = byte(e.intn(256))
			}
		}
		r[0x1f], r[0x20] = byte(e.intn(7)), byte([]int{0, 2, 0x13}[e.intn(3)])
		r[0x34] = byte(e.intn(256))
		maxHP := e.rangeIn(1, 400)
		binary.LittleEndian.PutUint16(r[0x40:], uint16(e.intn(maxHP+1)))
		binary.LittleEndian.PutUint16(r[0x42:], uint16(maxHP))
		binary.LittleEndian.PutUint16(r[0x48:], uint16(e.intn(300)))
		binary.LittleEndian.PutUint16(r[0x4a:], uint16(e.intn(300)))
	}
	return records
}

func (e *exporter) randomTargets(n int) []byte {
	t := make([]byte, e.intn(5))
	for i := range t {
		t[i] = byte(e.intn(n))
		if e.chance(3) {
			t[i] = byte(n + e.intn(3))
		}
	}
	return t
}

// skipGate 是黃金工具自己的確定性閘門（原版 0x1F183 判定屬未匯出函式）：mod=0 表示 nil。
func skipGate(mod int) func(int) bool {
	if mod == 0 {
		return nil
	}
	return func(i int) bool { return i%mod == 0 }
}

func (e *exporter) genScores() {
	for n := 0; n < 300; n++ {
		count := e.rangeIn(1, 6)
		records := e.randomRecords(count)
		if e.chance(3) {
			records = records[:len(records)-1]
		}
		targets := e.randomTargets(count)
		itemType := []int{0, 5, 0x0d, 0x14, 0x15, 0x18, 7, 0x100, -1}[e.intn(9)]
		itemWord := []int{0, 2, 20, 35, 36, 0x10000, -1, e.intn(400)}[e.intn(8)]
		var book []battle.NativeCommandRecord
		if e.chance(80) {
			book = e.book
		}
		got, err := battle.ScoreNativeAIItemCommandTargets(records, targets, itemType, itemWord, book)
		e.add("item_score", obj{"records": hexs(records), "targets": ints(targets), "item_type": itemType,
			"item_word": itemWord, "book": book != nil}, obj{"score": got, "err": errv(err)})
	}
	for n := 0; n < 300; n++ {
		count := e.rangeIn(1, 6)
		records := e.randomRecords(count)
		if e.chance(3) {
			records = records[:len(records)-1]
		}
		commandID := e.rangeIn(-1, 36)
		value := []int{-1, 0, 15, 50, 200}[e.intn(5)]
		groups := []battle.NativeAIScoredCommandCandidateGroup{}
		for g := 0; g < e.intn(4); g++ {
			groups = append(groups, battle.NativeAIScoredCommandCandidateGroup{
				Destination: battle.Cell{X: e.intn(8), Y: e.intn(6)}, TargetIndices: e.randomTargets(count),
			})
		}
		mod := []int{0, 1, 2, 3}[e.intn(4)]
		got, err := battle.ScoreNativeAIScoredCommandGroups(records, commandID, value, groups, skipGate(mod))
		e.add("group_score", obj{"records": hexs(records), "command_id": commandID, "value": value,
			"groups": conv(nil, groups), "skip_mod": mod}, obj{"scores": conv(nil, got), "err": errv(err)})
	}
	for n := 0; n < 300; n++ {
		count := e.rangeIn(1, 6)
		records := e.randomRecords(count)
		if e.chance(3) {
			records = records[:len(records)-1]
		}
		targets := e.randomTargets(count)
		fn := []string{"attack", "recovery", "flag", "zero_flag", "22"}[e.intn(5)]
		id := e.rangeIn(-1, 30)
		value := []int{-1, 0, 15, 100}[e.intn(4)]
		mod := []int{0, 1, 2}[e.intn(3)]
		var got int
		var err error
		switch fn {
		case "attack":
			got, err = battle.ScoreNativeAISpellAttack(records, targets, id, value, skipGate(mod))
		case "recovery":
			got, err = battle.ScoreNativeAISpellRecovery(records, targets, id)
		case "flag":
			got, err = battle.ScoreNativeAISpellFlag(records, targets, id)
		case "zero_flag":
			got, err = battle.ScoreNativeAISpellZeroFlag(records, targets, id)
		default:
			got, err = battle.ScoreNativeAISpell22(records, targets)
		}
		e.add("spell_score", obj{"fn": fn, "records": hexs(records), "targets": ints(targets), "id": id,
			"value": value, "skip_mod": mod}, obj{"score": got, "err": errv(err)})
	}
}

func (e *exporter) genIdle() {
	for n := 0; n < 300; n++ {
		count := e.rangeIn(1, 4)
		records := e.randomRecords(count)
		for i := 0; i < count; i++ {
			r := records[i*recSize:]
			if e.chance(60) {
				r[0x25], r[0x26] = 0, 0
			}
			if e.chance(10) {
				copy(r[0x40:0x42], r[0x42:0x44])
			}
		}
		if e.chance(3) {
			records = records[:len(records)-1]
		}
		cnt := count
		if e.chance(10) {
			cnt = []int{-1, 0, count + 1}[e.intn(3)]
		}
		unit := e.rangeIn(-1, count)
		if e.chance(80) {
			unit = e.intn(count)
		}
		mode := []string{"plan", "apply", "present_ok", "present_err", "present_nil"}[e.intn(5)]
		in := obj{"records": hexs(records), "count": cnt, "unit": unit, "mode": mode}
		res := obj{}
		switch mode {
		case "plan":
			d, err := battle.PlanNativeAIIdleRecovery(records, cnt, unit)
			res["decision"], res["err"] = conv(nil, d), errv(err)
			if err == nil {
				p, perr := battle.BuildNativeAIIdleRecoveryPresentation(d)
				res["presentation"], res["presentation_err"] = conv(nil, p), errv(perr)
			}
		case "apply":
			ok, err := battle.ApplyNativeAIIdleRecovery(records, cnt, unit)
			res["ok"], res["err"] = ok, errv(err)
		default:
			var present func(battle.NativeAIIdleRecoveryPresentation) error
			calls := 0
			switch mode {
			case "present_ok":
				present = func(battle.NativeAIIdleRecoveryPresentation) error { calls++; return nil }
			case "present_err":
				present = func(battle.NativeAIIdleRecoveryPresentation) error {
					calls++
					return fmt.Errorf("renderer unavailable")
				}
			}
			ok, err := battle.ApplyNativeAIIdleRecoveryWithPresentation(records, cnt, unit, present)
			res["ok"], res["err"], res["calls"] = ok, errv(err), calls
		}
		res["records"] = hexs(records)
		e.add("idle", in, res)
	}
}

func (e *exporter) genMode11() {
	for n := 0; n < 200; n++ {
		c23, c4f := int32(e.rangeIn(-1, 9)), int32(e.rangeIn(-1, 9))
		h23, h4f := e.chance(90), e.chance(90)
		tx, err := battle.SelectNativeAIMode11Transaction(c23, c4f, h23, h4f)
		if e.chance(30) {
			tx = battle.NativeAIMode11Transaction{FirstRoute: battle.NativeAIMode11Route(e.intn(5)), SecondRoute: battle.NativeAIMode11Route(e.intn(5))}
		}
		stages, serr := tx.Stages()
		failAt := e.intn(3)
		calls := []int{}
		var execute func(battle.NativeAIMode11Stage) error
		if e.chance(90) {
			execute = func(stage battle.NativeAIMode11Stage) error {
				calls = append(calls, stage.Ordinal*10+int(stage.Route))
				if stage.Ordinal == failAt {
					return fmt.Errorf("presentation owner unavailable")
				}
				return nil
			}
		}
		xerr := battle.ExecuteNativeAIMode11Transaction(tx, execute)
		e.add("mode11", obj{"c23": c23, "c4f": c4f, "h23": h23, "h4f": h4f, "tx": conv(nil, tx), "fail_at": failAt,
			"execute": execute != nil}, obj{"select": conv(nil, mustTx(battle.SelectNativeAIMode11Transaction(c23, c4f, h23, h4f))),
			"select_err": errv(err), "stages": conv(nil, stages), "stages_err": errv(serr), "calls": calls, "exec_err": errv(xerr)})
	}
}

func mustTx(tx battle.NativeAIMode11Transaction, _ error) battle.NativeAIMode11Transaction { return tx }

func (e *exporter) genModeRecordWrite() {
	for n := 0; n < 150; n++ {
		u := &battle.Unit{
			NativeRecordByte5: byte(e.intn(256)), HasNativeRecordByte5: e.chance(90),
			NativeRecordByte34: byte(e.intn(256)), HasNativeRecordByte34: e.chance(90),
			NativeRecordByte35: byte(e.intn(256)), HasNativeRecordByte35: e.chance(90),
			NativeRecordByte36: byte(e.intn(256)), HasNativeRecordByte36: e.chance(90),
			NativeRecordRace: byte(e.intn(256)), HasNativeRecordRace: e.chance(90),
		}
		var p *battle.Unit
		if e.chance(95) {
			p = u
		}
		record, err := battle.NativeAIModeRecordForUnit(p)
		e.add("mode_record", obj{"nil": p == nil, "unit": unitObj(u)}, obj{"record": hexs(record), "err": errv(err)})
	}
	for n := 0; n < 200; n++ {
		count := e.rangeIn(0, 5)
		records := e.bytesN(count * recSize)
		if e.chance(5) {
			records = records[:len(records)/2]
		}
		cnt := []int{count, count, count, count, count, count, -1, count + 1}[e.intn(8)]
		byteMode := e.chance(40)
		first, last := e.rangeIn(-1, count), e.rangeIn(-1, count)
		if count > 0 && e.chance(70) {
			first = e.intn(count)
			last = e.rangeIn(first, count-1)
		}
		mode := byte([]int{0, 2, 5, 7, 11, 15, 16, 0xff}[e.intn(8)])
		var err error
		if byteMode {
			err = battle.SetNativeAIModeByte(records, cnt, first, mode)
		} else {
			err = battle.SetNativeAIModeRange(records, cnt, first, last, mode)
		}
		in := obj{"records": hexs(records), "count": cnt, "byte": byteMode, "first": first, "last": last, "mode": int(mode)}
		e.add("mode_write", in, obj{"err": errv(err), "records": hexs(records)})
	}
}

func (e *exporter) randomScoreInput(bad bool) battle.NativePhysicalAttackScoreInput {
	w := func() int { return e.intn(400) }
	in := battle.NativePhysicalAttackScoreInput{
		ActorWord48: w(), ActorWord4A: w(), TargetWord48: w(), TargetWord4A: w(), TargetWord40: w(),
		RawTargetByte8: byte(e.intn(3)), RawHelper1DEBEResult: []int{-1, 1, 0}[e.intn(3)],
	}
	if bad {
		v := []int{-1, 0x10000}[e.intn(2)]
		switch e.intn(5) {
		case 0:
			in.ActorWord48 = v
		case 1:
			in.ActorWord4A = v
		case 2:
			in.TargetWord48 = v
		case 3:
			in.TargetWord4A = v
		default:
			in.TargetWord40 = v
		}
	}
	return in
}

func (e *exporter) genPhysicalScore() {
	for n := 0; n < 300; n++ {
		in := e.randomScoreInput(e.chance(10))
		s, ok, err := battle.ScoreNativePhysicalAttackCandidate(in)
		e.add("physical_score", obj{"in": conv(nil, in)}, obj{"score": conv(nil, s), "ok": ok, "err": errv(err)})
	}
	for n := 0; n < 200; n++ {
		cands := []battle.NativePhysicalAttackCandidate{}
		badAt := -1
		if e.chance(15) {
			badAt = e.intn(5)
		}
		for i := 0; i < e.intn(6); i++ {
			c := battle.NativePhysicalAttackCandidate{DestinationX: e.intn(40), DestinationY: e.intn(40), TargetIndex: e.intn(60),
				Inputs: e.randomScoreInput(i == badAt && e.chance(50))}
			if i == badAt {
				v := []int{-1, 0x100}[e.intn(2)]
				switch e.intn(3) {
				case 0:
					c.DestinationX = v
				case 1:
					c.DestinationY = v
				default:
					c.TargetIndex = v
				}
			}
			cands = append(cands, c)
		}
		sel, ok, err := battle.SelectNativePhysicalAttackCandidate(cands)
		e.add("physical_select", obj{"candidates": conv(nil, cands)}, obj{"selection": conv(nil, sel), "ok": ok, "err": errv(err)})
	}
}

// ---- 以戰場底稿為輸入的案例 ----

func (e *exporter) fixtureIndices(needRecords bool) []int {
	out := []int{}
	for k, fx := range e.fixtures {
		if !needRecords || fx.Records != nil {
			out = append(out, k)
		}
	}
	return out
}

func le16(b []byte, off int) int { return int(binary.LittleEndian.Uint16(b[off:])) }

// resolver 是黃金工具自己的確定性 score-input 來源（原版的地形修正屬 E 組）：
// 直接取兩份原始記錄的字，helper 依座標與索引的奇偶決定；failIndex 命中時回錯誤。
func resolver(failIndex int) battle.NativeAIPhysicalAttackScoreResolver {
	return func(raw battle.NativeAIPhysicalAttackRawCandidate) (battle.NativePhysicalAttackScoreInput, error) {
		if raw.TargetIndex == failIndex {
			return battle.NativePhysicalAttackScoreInput{}, fmt.Errorf("resolver refused %d", raw.TargetIndex)
		}
		helper := -1
		if (raw.Destination.X+raw.Destination.Y+raw.TargetIndex)%2 == 0 {
			helper = 1
		}
		return battle.NativePhysicalAttackScoreInput{
			ActorWord48: le16(raw.ActorRecord, 0x48), ActorWord4A: le16(raw.ActorRecord, 0x4a),
			TargetWord48: le16(raw.TargetRecord, 0x48), TargetWord4A: le16(raw.TargetRecord, 0x4a),
			TargetWord40: le16(raw.TargetRecord, 0x40), RawTargetByte8: raw.TargetRecord[8],
			RawHelper1DEBEResult: helper,
		}, nil
	}
}

func (e *exporter) genFixtureScores() {
	for _, k := range e.fixtureIndices(true) {
		fx := e.fixtures[k]
		count := len(fx.Units)
		base, err := battle.NativeCompositionBaseFlags(fx.W, fx.H, fx.Events)
		if err != nil {
			panic(err)
		}
		perMap := 6
		if fx.W*fx.H < 100 {
			perMap = 3
		}
		for n := 0; n < perMap; n++ {
			actor := e.intn(count)
			selector := e.intn(2)
			if e.chance(5) {
				selector = 2
			}
			mask := []int{e.intn(256), e.intn(256), e.intn(256), e.intn(256), e.intn(256)}
			mp := []int{0, 10, 40, 255}[e.intn(4)]
			edits := []obj{{"unit": actor, "set": obj{"NativeCommandMask": mask, "MP": mp}}}
			units := e.units(k, edits)
			records, rerr := battle.NativeAIScoringRecords(units)
			if rerr != nil {
				panic(rerr)
			}
			row := e.intn(len(e.costRows))
			mod := []int{0, 2, 3}[e.intn(3)]
			got, err := battle.ScoreNativeAI1598A(fx.W, fx.H, records, count, actor, selector, units[actor], e.book,
				base, fx.Terrain, e.costRows[row], skipGate(mod))
			e.add("score1598a", obj{"fixture": k, "edits": edits, "actor": actor, "selector": selector, "row": row, "skip_mod": mod},
				obj{"result": conv(nil, got), "err": errv(err)})
		}
		for n := 0; n < perMap; n++ {
			actor := e.intn(count)
			selector := e.intn(2)
			slots := []int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
			flags := []int{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}
			for s := 0; s < e.intn(4); s++ {
				slots[s] = []int{79, 194, 1, 0x1f, 120, 150, e.intn(256)}[e.intn(7)]
				flags[s] = []int{0, 0x40}[e.intn(2)]
			}
			edits := []obj{{"unit": actor, "set": obj{"InventorySlots": slots, "NativeInventoryFlags": flags}}}
			units := e.units(k, edits)
			records, rerr := battle.NativeAIScoringRecords(units)
			if rerr != nil {
				panic(rerr)
			}
			got, err := battle.ScoreNativeAI1567E(fx.W, fx.H, records, count, actor, selector, e.itemRows, e.book, base)
			e.add("score1567e", obj{"fixture": k, "edits": edits, "actor": actor, "selector": selector},
				obj{"result": conv(nil, got), "err": errv(err)})
			src, found, serr := battle.ResolveNativeAIPhysicalItemSource(records[actor*recSize:], e.itemRows)
			e.add("item_source", obj{"fixture": k, "edits": edits, "actor": actor},
				obj{"source": conv(nil, src), "found": found, "err": errv(serr)})
		}
		for n := 0; n < perMap; n++ {
			actor := e.intn(count)
			selector := e.intn(2)
			budget := []int{0, 2, 4, 6, 0x1c}[e.intn(5)]
			geom := battle.NativeAIPhysicalAttackGeometry{TargetMode: []int{0, 1, 2, 3, 0x11, 0x12}[e.intn(6)],
				TargetInnerMark: []int{0, 1, 2}[e.intn(3)], TargetCode: e.intn(4)}
			if e.chance(5) {
				geom.TargetCode = 4
			}
			units := e.units(k, nil)
			records := fx.Records
			runtime := e.chance(60)
			movement := base
			if runtime {
				if movement, err = battle.NativeCommandRuntimeFlags(fx.W, fx.H, fx.Events, units, byte(selector)); err != nil {
					panic(err)
				}
			}
			row := e.intn(len(e.costRows))
			failIndex := -1
			if e.chance(10) {
				failIndex = e.intn(count)
			}
			var resolve battle.NativeAIPhysicalAttackScoreResolver
			if e.chance(97) {
				resolve = resolver(failIndex)
			}
			cands, err := battle.BuildNativeAIPhysicalAttackCandidates(fx.W, fx.H, records, count, actor, selector, budget, geom,
				movement, base, fx.Terrain, e.costRows[row], resolve)
			res := obj{"candidates": conv(nil, cands), "err": errv(err)}
			if err == nil {
				sel, ok, serr := battle.SelectNativePhysicalAttackCandidate(cands)
				res["selection"], res["ok"], res["select_err"] = conv(nil, sel), ok, errv(serr)
			}
			e.add("physical_build", obj{"fixture": k, "actor": actor, "selector": selector, "budget": budget,
				"geometry": conv(nil, geom), "runtime": runtime, "row": row, "fail_index": failIndex, "resolver": resolve != nil}, res)
		}
	}
}

func (e *exporter) genPhase() {
	for _, k := range e.fixtureIndices(true) {
		fx := e.fixtures[k]
		count := len(fx.Units)
		if count == 0 || count > 0x100 {
			continue
		}
		for n := 0; n < 2; n++ {
			edits := []obj{}
			for i := 0; i < 3; i++ {
				edits = append(edits, obj{"unit": e.intn(count), "set": obj{
					"NativeCommandMask": []int{e.intn(256), 0, e.intn(256), 0, e.intn(256)}, "MP": e.intn(80),
				}})
			}
			sp := stateSpec{fixture: k, edits: edits}
			st := e.state(sp)
			records, err := battle.NativeAIScoringRecords(st.Units)
			if err != nil {
				panic(err)
			}
			admission, err := fdother.PlanNativePhaseUnitScans(records, count, make([]fdother.NativePhasePreselectionScores, count))
			if err != nil {
				panic(err)
			}
			inputs := []obj{}
			for _, c := range admission.ZeroPreselection {
				inputs = append(inputs, obj{"unit": c.UnitIndex, "row": e.intn(len(e.costRows)), "skip_mod": []int{0, 2, 3}[e.intn(3)]})
			}
			switch e.intn(8) {
			case 0:
				if len(inputs) > 0 {
					inputs = inputs[1:]
				}
			case 1:
				if len(inputs) > 0 {
					inputs = append(inputs, inputs[0])
				}
			case 2:
				// 至多一筆不合格輸入：Go 迭代 map 報錯，多筆時報哪一筆不定。
				for idx := 0; idx < count; idx++ {
					if records[idx*recSize+6] != 0 {
						inputs = append(inputs, obj{"unit": idx, "row": 0, "skip_mod": 0})
						break
					}
				}
			case 3:
				inputs = append(inputs, obj{"unit": []int{-1, count}[e.intn(2)], "row": 0, "skip_mod": 0})
			}
			goInputs := []battle.NativeAIPhaseZeroInput{}
			for _, in := range inputs {
				goInputs = append(goInputs, battle.NativeAIPhaseZeroInput{UnitIndex: in["unit"].(int),
					CostRow: e.costRows[in["row"].(int)], Skip: skipGate(in["skip_mod"].(int))})
			}
			var itemRows []byte = e.itemRows
			plan, diags, err := battle.BuildNativeAIPhaseDiagnosticPlan(st, e.book, itemRows, goInputs)
			e.add("phase", obj{"state": sp.json(), "inputs": inputs}, obj{"plan": conv(nil, plan), "diagnostics": conv(nil, diags), "err": errv(err)})
		}
	}
}

// eventCells 列出每個事件號碼在格網中列優先的第一個命中格（地形控制 byte 0 的 &0x60==0x20，
// 即 0x15DF3 會回傳的格子），以及其餘帶事件 byte 的格子（不是命中格，用來產生「沒走到」的案例）。
func eventCells(grid, control []byte, w, h int) (hits, others [][3]int) {
	if len(grid) != 4+4*w*h {
		return nil, nil
	}
	seen := map[int]bool{}
	for i := 0; i < w*h; i++ {
		off := 4 + 4*i
		ev := int(grid[off+2] & 0x1f)
		tile := int(binary.LittleEndian.Uint16(grid[off:]) & 0x3ff)
		cell := [3]int{i % w, i / w, ev}
		if !seen[ev] && tile < len(control)/4 && control[tile*4]&0x60 == 0x20 {
			seen[ev] = true
			hits = append(hits, cell)
		} else if ev != 0 {
			others = append(others, cell)
		}
	}
	return hits, others
}

func (e *exporter) genMode5() {
	for k := range e.fixtures {
		fx := e.fixtures[k]
		if len(fx.Units) == 0 {
			continue
		}
		hits, others := eventCells(fx.Grid, fx.Control, fx.W, fx.H)
		perMap := 4
		for n := 0; n < perMap; n++ {
			unit := e.intn(len(fx.Units))
			eventID, dest := e.intn(6), battle.Cell{X: e.intn(fx.W), Y: e.intn(fx.H)}
			if len(hits) > 0 && e.chance(80) {
				c := hits[e.intn(len(hits))]
				eventID, dest = c[2], battle.Cell{X: c[0], Y: c[1]}
			} else if len(others) > 0 && e.chance(50) {
				c := others[e.intn(len(others))]
				eventID, dest = c[2], battle.Cell{X: c[0], Y: c[1]}
			}
			set := obj{"NativeRecordByte3D": eventID, "HasNativeRecordByte3D": e.chance(95),
				"NativeRecordDeathEffect": []int{0xff, 0, 0}, "HasNativeRecordDeathEffect": e.chance(90)}
			if e.chance(85) {
				set["X"], set["Y"] = dest.X, dest.Y
			}
			if e.chance(20) {
				set["Inventory"] = []int{1, 2, 3, 4, 5, 6, 7, 8}
			}
			edits := []obj{{"unit": unit, "set": set}}
			sp := stateSpec{fixture: k, edits: edits, eventSt: map[int]int{}}
			if e.chance(15) {
				sp.eventSt[eventID%0x20] = 1
			}
			if e.chance(5) {
				sp.eventSt[e.intn(0x20)] = 1
			}
			if e.chance(40) {
				raw := e.bytesN(0x56 + 3*16)
				if e.chance(10) {
					raw = raw[:0x56]
				}
				sp.fieldRaw = raw
			}
			if e.chance(5) {
				sp.noGrid = true
			}
			if e.chance(5) && len(fx.Grid) > 8 {
				g := append([]byte(nil), fx.Grid...)
				binary.LittleEndian.PutUint16(g[4+4*e.intn(fx.W*fx.H):], 0x3ff)
				sp.grid = g
			}
			st := e.state(sp)
			res := obj{}
			lookups := []obj{}
			for _, id := range []int{eventID, e.intn(0x20)} {
				c, err := st.NativeAIMode5EventCell(byte(id))
				lookups = append(lookups, obj{"id": id, "cell": [2]int{c.X, c.Y}, "err": errv(err)})
			}
			res["lookups"] = lookups
			tiles, ok := st.NativeMapDrawTiles()
			res["draw_before"], res["draw_ok"] = tiles, ok
			before := snapshot(st.Units)
			cues := []obj{}
			var err error
			var target *battle.Unit
			if e.chance(97) {
				target = st.Units[unit]
			}
			withAudio := e.chance(50)
			if withAudio {
				err = st.ApplyNativeAIMode5EventWithAudioCue(target, byte(eventID), dest, func(cue battle.NativeAIMode5AudioCue) {
					cues = append(cues, conv(nil, cue).(obj))
				})
			} else {
				err = st.ApplyNativeAIMode5Event(target, byte(eventID), dest)
			}
			res["err"], res["cues"], res["changed"] = errv(err), cues, changed(before, st.Units)
			res["event_state"] = ints(st.NativeEventState[:])
			res["grid"] = hexs(st.NativeMapEventGrid)
			treasure := [][2]int{}
			for slot, v := range st.OpenedTreasure {
				b := 0
				if v {
					b = 1
				}
				treasure = append(treasure, [2]int{slot, b})
			}
			sort.Slice(treasure, func(i, j int) bool { return treasure[i][0] < treasure[j][0] })
			res["opened"] = treasure
			tiles, ok = st.NativeMapDrawTiles()
			res["draw_after"], res["draw_after_ok"] = tiles, ok
			at := []obj{}
			query := [][2]int{}
			for q := 0; q < 3; q++ {
				x, y := e.rangeIn(-1, fx.W), e.rangeIn(-1, fx.H)
				tile, ok := st.NativeMapTileAt(x, y)
				at = append(at, obj{"x": x, "y": y, "tile": tile, "ok": ok})
				query = append(query, [2]int{x, y})
			}
			res["tile_at"] = at
			e.add("mode5", obj{"state": sp.json(), "unit": unit, "nil": target == nil, "event": eventID,
				"dest": [2]int{dest.X, dest.Y}, "audio": withAudio, "lookup2": lookups[1]["id"], "tile_at_query": query}, res)
		}
	}
}

func (e *exporter) genPlayerMovement() {
	for _, k := range e.fixtureIndices(true) {
		fx := e.fixtures[k]
		count := len(fx.Units)
		for n := 0; n < 3; n++ {
			actor := e.intn(count)
			set := obj{}
			if e.chance(80) {
				set["native_record_byte6"] = 2
			}
			if e.chance(20) {
				set["NativeRecordClass"] = []int{19, 0x13, 3}[e.intn(3)]
			}
			edits := []obj{{"unit": actor, "set": set}}
			sp := stateSpec{fixture: k, edits: edits, mapView: e.chance(95), withRows: e.chance(95)}
			st := e.state(sp)
			var u *battle.Unit
			if e.chance(97) {
				u = st.Units[actor]
			}
			p, err := st.PlanNativePlayerMovement(u)
			res := obj{"err": errv(err)}
			paths := []obj{}
			query := [][2]int{}
			if err == nil {
				reach := [][2]int{}
				for c := range p.Reach {
					reach = append(reach, [2]int{c.X, c.Y})
				}
				sort.Slice(reach, func(i, j int) bool {
					if reach[i][1] != reach[j][1] {
						return reach[i][1] < reach[j][1]
					}
					return reach[i][0] < reach[j][0]
				})
				res["field"], res["reach"] = hexs(p.Field), reach
				for q := 0; q < 4; q++ {
					dest := battle.Cell{X: e.intn(fx.W), Y: e.intn(fx.H)}
					if len(reach) > 0 && e.chance(70) {
						r := reach[e.intn(len(reach))]
						dest = battle.Cell{X: r[0], Y: r[1]}
					}
					query = append(query, [2]int{dest.X, dest.Y})
					path, perr := p.Path(dest)
					cells := [][2]int{}
					for _, c := range path {
						cells = append(cells, [2]int{c.X, c.Y})
					}
					paths = append(paths, obj{"dest": [2]int{dest.X, dest.Y}, "path": cells, "err": errv(perr)})
				}
			}
			res["paths"] = paths
			e.add("player_move", obj{"state": sp.json(), "actor": actor, "nil": u == nil, "path_query": query}, res)
		}
	}
}

// ---- command24 家族 ----

func (e *exporter) cmdBook() []battle.NativeCommandRecord {
	book := append([]battle.NativeCommandRecord(nil), e.book...)
	for _, id := range []int{24, 28, 29, 30, 31} {
		if e.chance(40) {
			book[id] = battle.NativeCommandRecord{ID: id, SelectionMode: []int{0, 1, 5, 0x14, 0x12}[e.intn(5)],
				EffectMode: e.intn(3), MPCost: []int{0, 22, 26, 24, 300, -1}[e.intn(6)], TargetCode: e.intn(4)}
		}
	}
	if e.chance(5) {
		book[e.intn(36)].ID = 99
	}
	return book
}

func (e *exporter) genCommand24() {
	for n := 0; n < 300; n++ {
		ap, dp := e.rangeIn(-50, 400), e.rangeIn(-50, 400)
		mult := []int{15, 20, 12, 18, 0, -1}[e.intn(6)]
		seed := e.seed()
		var rng *mrand.Rand
		if e.chance(95) {
			rng = mrand.New(mrand.NewSource(seed))
		}
		amount, damage, err := battle.ResolveNativeCommandDerivedStrikeDamage(ap, dp, mult, rng)
		e.add("derived_damage", obj{"ap": ap, "dp": dp, "mult": mult, "seed": seed, "nil_rng": rng == nil},
			obj{"amount": amount, "damage": damage, "err": errv(err)})
	}
	start := len(e.fixtures) - 40 // 合成戰場
	for k := start; k < len(e.fixtures); k++ {
		fx := e.fixtures[k]
		count := len(fx.Units)
		for n := 0; n < 25; n++ {
			book := e.cmdBook()
			edits := []obj{}
			actor := e.intn(count)
			edits = append(edits, obj{"unit": actor, "set": obj{"MP": []int{0, 25, 40, 80}[e.intn(4)], "Acted": e.chance(10)}})
			sp := stateSpec{fixture: k, edits: edits, book: book}
			st := e.state(sp)
			seed := e.seed()
			rng := mrand.New(mrand.NewSource(seed))
			kind := []string{"plan", "execute", "plan29", "plan30", "execute30", "targets29", "targets30"}[e.intn(7)]
			id := []int{24, 28, 29, 31, 30, 0, -1, 36}[e.intn(8)]
			confirmed := e.rangeIn(-1, count-1)
			cursor := battle.Cell{X: e.rangeIn(-1, fx.W), Y: e.rangeIn(-1, fx.H)}
			saved := battle.Cell{X: e.intn(fx.W), Y: e.intn(fx.H)}
			// 多半挑原版會接受的游標／確認單位：0x14818 在 actor 格以該指令的 selection mode 掃出的候選。
			flags, ferr := st.NativeCommandBaseFlags()
			if ferr != nil {
				panic(ferr)
			}
			recID := id
			switch kind {
			case "plan29", "targets29":
				recID = 29
			case "plan30", "execute30", "targets30":
				recID = 30
			}
			picked := false
			if recID >= 0 && recID < len(book) && e.chance(75) {
				rec := book[recID]
				a := st.Units[actor]
				if cands, cerr := battle.NativeCommandTargets(fx.W, fx.H, battle.Cell{X: a.X, Y: a.Y}, rec.SelectionMode, rec.TargetCode, flags, st.Units); cerr == nil && len(cands) > 0 {
					pick := cands[e.intn(len(cands))]
					for i, u := range st.Units {
						if u == pick {
							confirmed = i
						}
					}
					cursor = battle.Cell{X: pick.X, Y: pick.Y}
					picked = true
				}
			}
			if (kind == "plan29" || kind == "targets29") && e.chance(50) {
				a := st.Units[actor]
				cursor = battle.Cell{X: a.X + e.rangeIn(-1, 1), Y: a.Y + e.rangeIn(-1, 1)}
			}
			if !picked && e.chance(40) && confirmed >= 0 {
				cursor = battle.Cell{X: st.Units[confirmed].X, Y: st.Units[confirmed].Y}
			}
			var actorP, confirmedP *battle.Unit
			if e.chance(97) {
				actorP = st.Units[actor]
			}
			if confirmed >= 0 {
				confirmedP = st.Units[confirmed]
			}
			script := []obj{}
			for s := 0; s < e.intn(12); s++ {
				script = append(script, obj{"op": []string{"mp", "target", "stage", "complete", "restore30"}[e.intn(5)],
					"index": e.rangeIn(-1, 2), "stage": e.rangeIn(0, 9)})
			}
			if e.chance(50) {
				// 正常的發佈順序：MP → 每個目標（或 28 的八段）→ 完成。
				script = []obj{{"op": "mp", "index": 0, "stage": 0}}
				for i := 0; i < 3; i++ {
					if id == 28 {
						for s := 1; s <= 8; s++ {
							script = append(script, obj{"op": "stage", "index": i, "stage": s})
						}
					} else {
						script = append(script, obj{"op": "target", "index": i, "stage": 0})
					}
				}
				script = append(script, obj{"op": "complete", "index": 0, "stage": 0})
				if e.chance(30) {
					script = append(script, obj{"op": "restore30", "index": 0, "stage": 0})
				}
			}
			before := snapshot(st.Units)
			res := obj{}
			var plan *battle.NativeCommandDerivedStrikePlan
			var err error
			switch kind {
			case "plan":
				plan, err = st.PlanNativeCommandDerivedStrike(actorP, confirmedP, id, rng)
			case "plan29":
				plan, err = st.PlanNativeCommand29AtCursor(actorP, cursor, rng)
			case "plan30":
				plan, err = st.PlanNativeCommand30(actorP, saved, cursor, rng)
			case "execute":
				results, xerr := st.ExecuteNativeCommandDerivedStrike(actorP, confirmedP, id, rng)
				res["results"], err = conv(st.Units, results), xerr
			case "execute30":
				results, xerr := st.ExecuteNativeCommand30(actorP, saved, cursor, rng)
				res["results"], err = conv(st.Units, results), xerr
			case "targets29":
				targets, terr := st.NativeCommand29EffectTargets(actorP, cursor)
				res["targets"], err = conv(st.Units, targets), terr
			case "targets30":
				targets, terr := st.NativeCommand30EffectTargets(actorP, saved, cursor)
				res["targets"], err = conv(st.Units, targets), terr
			}
			res["err"] = errv(err)
			res["plan"] = conv(st.Units, plan)
			res["plan_nil"] = plan == nil
			res["rng_after"] = rng.Int63()
			res["changed_plan"] = changed(before, st.Units)
			steps := []obj{}
			if plan != nil {
				for _, step := range script {
					idx, stage := step["index"].(int), step["stage"].(int)
					var serr error
					switch step["op"] {
					case "mp":
						serr = battle.ApplyNativeCommandDerivedStrikeMP(plan)
					case "target":
						serr = battle.ApplyNativeCommandDerivedStrikeTarget(plan, idx)
					case "stage":
						serr = battle.ApplyNativeCommand28TargetStage(plan, idx, stage)
					case "complete":
						serr = battle.CompleteNativeCommandDerivedStrike(plan)
					case "restore30":
						battle.RestoreNativeCommand30Targets(plan)
					}
					var proj any
					if step["op"] == "stage" {
						r, perr := battle.NativeCommand28StageResult(plan, idx, stage)
						proj = obj{"result": conv(st.Units, r), "err": errv(perr)}
					}
					steps = append(steps, obj{"err": errv(serr), "projection": proj})
				}
			}
			res["steps"] = steps
			res["changed"] = changed(before, st.Units)
			e.add("command24", obj{"state": sp.json(), "kind": kind, "id": id, "actor": actor, "actor_nil": actorP == nil,
				"confirmed": confirmed, "cursor": [2]int{cursor.X, cursor.Y}, "saved": [2]int{saved.X, saved.Y},
				"seed": seed, "script": script}, res)
		}
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
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x52554c44))}
	real := e.genReal()
	e.genSyntheticFixtures(40)
	e.genDispatch()
	e.genSpellCandidate()
	e.genScores()
	e.genIdle()
	e.genMode11()
	e.genModeRecordWrite()
	e.genPhysicalScore()
	e.genFixtureScores()
	e.genPhase()
	e.genMode5()
	e.genPlayerMovement()
	e.genCommand24()
	doc := obj{"schema": "fd2-golden-rules-d/1", "seed": *seed, "real": real, "fixtures": e.fixtureList(), "cases": e.cases}
	data, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		panic(err)
	}
	ops := map[string][2]int{}
	for _, c := range e.cases {
		v := ops[c.Op]
		v[0]++
		if errStr, isErr := c.Result["err"].(string); !isErr || errStr == "" {
			v[1]++
		}
		ops[c.Op] = v
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
