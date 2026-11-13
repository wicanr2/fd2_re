// fd2-golden-rules-e 輸出 internal/battle「E 組」（戰鬥、成長與法術）的黃金差分資料，
// 供 Rust 改寫 remake-rs/crates/fd2-rules 逐欄比對（docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 battle 的公開 API，不改任何遊戲行為。
//   - 真實輸入：remake/assets/maps/map*/map*_units.json 全部以 Load＋BindNativeFutureItemRows 載入
//     （路徑排序後保存清單與 SHA-256），每張地圖一份戰場底稿。spells.json、native_item_effect_rows.json、
//     class_change_growth.json、native_growth_slots.json、command_learn.json 也記路徑與 SHA-256。
//   - 產生輸入：以固定種子的 PCG 產生合成戰場、單位修改、原版亂數字組與 math/rand 種子，連同結果寫進
//     同一份 JSON；Rust 端不需重現產生器，只重做同一串呼叫。math/rand 種子另有 1～100 的固定迴圈。
//
// 戰場一律由「底稿＋修改」組成：底稿單位是 json.Marshal(battle.Unit) 的物件，修改是逐單位合併的
// JSON 鍵（Go 的 JSON 鍵）。Go 與 Rust 都把合併後的物件解碼成 Unit，再呼叫 BindNativeFutureItemRows。
// 結果裡的 *Unit 一律寫成 State.Units 的索引（nil 為 -1），結構以 Go 欄位名輸出（未匯出欄位省略）。
// 錯誤欄位是 Go err.Error() 原文。每個帶 math/rand 的案例另記呼叫後的下一個 Int63，用來比對亂數消耗量。
//
// 不放進黃金的路徑（會走到 D 組，或以時間播種）：NextAIPlan、NextAllyAIPlan、AITurn、Attack、Cast。
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
	Codes  []byte
	Cost   []int
	Units  []obj
}

type exporter struct {
	rng         *rand.Rand
	tmp         string
	cases       []caseJSON
	fixtures    []fixture
	itemRows    []byte
	spells      []battle.Spell
	growthRows  map[int]battle.GrowthRow
	growthSlots map[int]battle.GrowthRow
	learn       map[int][]battle.CommandLearnEntry
	selectors   map[int]int
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

// ---- Unit 的 JSON 鍵（有標籤用標籤名，沒有用 Go 欄位名）----

var unitKeys = func() map[string]string {
	out := map[string]string{}
	t := reflect.TypeOf(battle.Unit{})
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		key := f.Name
		if tag, ok := f.Tag.Lookup("json"); ok {
			if name := strings.Split(tag, ",")[0]; name != "" && name != "-" {
				key = name
			}
		}
		out[f.Name] = key
	}
	return out
}()

func ukey(field string) string {
	key, ok := unitKeys[field]
	if !ok {
		panic("Unit 沒有欄位 " + field)
	}
	return key
}

// set 以 Go 欄位名組成單位修改。
func set(kv ...any) obj {
	out := obj{}
	for i := 0; i < len(kv); i += 2 {
		out[ukey(kv[i].(string))] = kv[i+1]
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
	case reflect.Map:
		keys := v.MapKeys()
		sort.Slice(keys, func(i, j int) bool { return fmt.Sprint(keys[i]) < fmt.Sprint(keys[j]) })
		out := obj{}
		for _, k := range keys {
			out[fmt.Sprint(k.Interface())] = convValue(units, v.MapIndex(k))
		}
		return out
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
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

// buildState 依案例輸入組出戰場。可選鍵：growth（"rows"／"slots"）、learn、cancel、observer、
// balance、acc（NativeExperienceAccumulator）、spellbook（true 表示真實法術表）、ai_spell、no_items。
func (e *exporter) buildState(sp obj) *battle.State {
	fx := e.fixtures[sp["fixture"].(int)]
	st := &battle.State{
		W: fx.W, H: fx.H, Units: e.buildUnits(sp),
		NativeCompositionEventBytes: append([]byte(nil), fx.Events...),
		NativeTerrainMoveCodes:      append([]byte(nil), fx.Codes...),
	}
	if fx.Cost != nil {
		st.Cost = append([]int(nil), fx.Cost...)
	}
	if b, _ := sp["no_items"].(bool); !b {
		if err := st.BindNativeFutureItemRows(e.itemRows); err != nil {
			panic(err)
		}
	}
	switch sp["growth"] {
	case "rows":
		st.NativeGrowthRows = e.growthRows
	case "slots":
		st.NativeGrowthRows = e.growthSlots
	}
	if b, _ := sp["learn"].(bool); b {
		st.CommandLearn, st.CommandLearnSelectors = e.learn, e.selectors
	}
	if b, _ := sp["cancel"].(bool); b {
		// 以等級決定的可重現規則；Rust 端用同一條。
		st.NativeDeathExpCancel = func(u *battle.Unit) bool { return u.Lv%3 == 0 }
	}
	if b, _ := sp["observer"].(bool); b {
		st.NativeGrowthRollObserver = func(u *battle.Unit, s uint16) uint16 { return s + uint16(u.Lv) }
	}
	st.Balance = balanceFrom(sp["balance"])
	if acc, ok := sp["acc"].(int); ok {
		st.NativeExperienceAccumulator = acc
	}
	if b, _ := sp["spellbook"].(bool); b {
		st.SpellBook = append([]battle.Spell(nil), e.spells...)
	}
	if pairs, ok := sp["ai_spell"].([][2]int); ok {
		st.AICommandSpell = map[int]int{}
		for _, kv := range pairs {
			st.AICommandSpell[kv[0]] = kv[1]
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
	for _, p := range []string{"assets/spells.json", "assets/data/native_item_effect_rows.json",
		"assets/data/class_change_growth.json", "assets/data/native_growth_slots.json",
		"assets/data/command_learn.json"} {
		record(p)
	}
	if e.spells, err = battle.LoadSpells("assets/spells.json"); err != nil {
		panic(err)
	}
	if e.itemRows, err = battle.LoadNativeItemEffectRowPrefix("assets/data/native_item_effect_rows.json"); err != nil {
		panic(err)
	}
	if e.growthRows, err = battle.LoadNativeGrowthRows("assets/data/class_change_growth.json"); err != nil {
		panic(err)
	}
	if e.growthSlots, err = battle.LoadNativeGrowthSlots("assets/data/native_growth_slots.json"); err != nil {
		panic(err)
	}
	if e.learn, err = battle.LoadCommandLearn("assets/data/command_learn.json"); err != nil {
		panic(err)
	}
	if e.selectors, err = battle.LoadCommandLearnSelectors("assets/data/class_change_growth.json"); err != nil {
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
		if err := st.BindNativeFutureItemRows(e.itemRows); err != nil {
			panic(fmt.Sprintf("%s: %v", path, err))
		}
		fx := fixture{Name: path, W: st.W, H: st.H, Events: st.NativeCompositionEventBytes,
			Codes: st.NativeTerrainMoveCodes, Cost: st.Cost}
		for _, u := range st.Units {
			fx.Units = append(fx.Units, unitObj(u))
		}
		e.fixtures = append(e.fixtures, fx)
	}
	return obj{"files": files, "maps": paths, "item_rows": hexs(e.itemRows), "spells": conv(nil, e.spells)}
}

// ---- 合成戰場 ----

func (e *exporter) genSyntheticFixtures(n int) {
	for k := 0; k < n; k++ {
		w, h := e.rangeIn(2, 10), e.rangeIn(1, 8)
		fx := fixture{Name: fmt.Sprintf("synthetic-%d", k), W: w, H: h, Events: make([]byte, w*h)}
		if e.chance(70) {
			fx.Cost = make([]int, w*h)
			for i := range fx.Cost {
				fx.Cost[i] = []int{1, 1, 1, 2, 99, 0}[e.intn(6)]
			}
		}
		switch e.intn(3) {
		case 0: // 完整原版地形碼
			fx.Codes = make([]byte, w*h)
			for i := range fx.Codes {
				fx.Codes[i] = byte(e.intn(8))
			}
		case 1: // 不完整（長度不符），退回舊的 Cost 推法
			fx.Codes = e.bytesN(e.intn(w * h))
		}
		count := e.rangeIn(2, 7)
		for i := 0; i < count; i++ {
			u := &battle.Unit{
				Camp: battle.Camp(e.intn(3)), Name: e.legacyName(), Lv: e.rangeIn(1, 45), HP: e.rangeIn(0, 300),
				MP: e.rangeIn(0, 60), AP: e.rangeIn(0, 250), DP: e.rangeIn(0, 200), HIT: e.rangeIn(0, 130),
				EV: e.rangeIn(0, 60), CritPct: e.rangeIn(0, 40), MV: e.rangeIn(0, 6),
				X: e.intn(w), Y: e.intn(h), OnField: e.chance(85), ExpPerLevel: e.rangeIn(0, 120),
				ClassID: e.rangeIn(0, 26), DX: e.rangeIn(0, 20),
			}
			u.ClsName = e.legacyClass(u.Name)
			u.MaxHP = u.HP + e.rangeIn(0, 60)
			u.MaxMP = u.MP + e.rangeIn(0, 20)
			if e.chance(40) {
				u.Inventory = []int{e.intn(0x100), e.intn(0x100)}
				u.Equipped = []bool{e.chance(60), e.chance(60)}
			}
			fx.Units = append(fx.Units, unitObj(u))
		}
		e.fixtures = append(e.fixtures, fx)
	}
}

// legacyGrowth 是 growth.go growthTable 的名字與職業（未匯出，這裡列出部分作為輸入；另含查不到的名字）。
var legacyGrowth = [][2]string{
	{"索爾", "劍士"}, {"索爾", "英雄"}, {"鐵諾", "劍聖"}, {"亞雷斯", "騎士"}, {"亞雷斯", "龍騎士"},
	{"悠妮", "法師"}, {"悠妮", "召喚師"}, {"瑪琳", "僧侶"}, {"哈瓦特", "戰士"}, {"希莉亞", "弓兵"},
	{"凱麗", "武聖"}, {"蓋亞", "機兵"}, {"達克賽", "？？？"}, {"無名雜兵", "士兵"}, {"索爾", "不存在"},
}

func (e *exporter) legacyName() string {
	if e.chance(30) {
		return ""
	}
	return legacyGrowth[e.intn(len(legacyGrowth))][0]
}

func (e *exporter) legacyClass(name string) string {
	for _, p := range legacyGrowth {
		if p[0] == name && e.chance(60) {
			return p[1]
		}
	}
	return legacyGrowth[e.intn(len(legacyGrowth))][1]
}

func (e *exporter) fixtureList() []obj {
	out := make([]obj, len(e.fixtures))
	for i, fx := range e.fixtures {
		out[i] = obj{"name": fx.Name, "w": fx.W, "h": fx.H, "events": hexs(fx.Events), "codes": hexs(fx.Codes),
			"cost": fx.Cost, "units": fx.Units}
	}
	return out
}

// ---- 單位修改 ----

// pickUnits 從戰場挑兩個不同的單位（只有一個單位時兩者相同）。
func (e *exporter) pickUnits(fx int) (int, int) {
	n := len(e.fixtures[fx].Units)
	a := e.intn(n)
	if n == 1 {
		return a, a
	}
	d := e.intn(n - 1)
	if d >= a {
		d++
	}
	return a, d
}

func (e *exporter) baseOf(fx, unit int) obj { return e.fixtures[fx].Units[unit] }

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
	case nil:
		return 0
	}
	panic(fmt.Sprintf("num: %T", v))
}

// combatEdit 讓兩個單位進入可交戰狀態：都在場、活著，守方站到攻方旁邊或兩格外。
func (e *exporter) combatEdit(fxi, a, d int) []obj {
	fx := e.fixtures[fxi]
	ax, ay := num(e.baseOf(fxi, a)["X"]), num(e.baseOf(fxi, a)["Y"])
	if ax < 0 || ax >= fx.W || ay < 0 || ay >= fx.H {
		ax, ay = e.intn(fx.W), e.intn(fx.H)
	}
	dx, dy := ax, ay
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {2, 0}, {1, 1}}
	for _, k := range e.rng.Perm(len(dirs)) {
		if e.chance(80) && k >= 4 {
			continue
		}
		nx, ny := ax+dirs[k][0], ay+dirs[k][1]
		if nx >= 0 && nx < fx.W && ny >= 0 && ny < fx.H {
			dx, dy = nx, ny
			break
		}
	}
	as := set("X", ax, "Y", ay, "OnField", true, "Acted", e.chance(20))
	ds := set("X", dx, "Y", dy, "OnField", true)
	if num(e.baseOf(fxi, a)["HP"]) <= 0 {
		as[ukey("HP")] = e.rangeIn(1, 80)
	}
	if num(e.baseOf(fxi, d)["HP"]) <= 0 || e.chance(30) {
		ds[ukey("HP")] = e.rangeIn(1, 60)
	}
	e.statEdits(as, true)
	e.statEdits(ds, false)
	return []obj{{"unit": a, "set": as}, {"unit": d, "set": ds}}
}

// statEdits 隨機調整攻防數值與陣營，製造命中、暴擊、擊倒與各陣營分支。
func (e *exporter) statEdits(s obj, attacker bool) {
	if e.chance(40) {
		s[ukey("Camp")] = e.intn(3)
	}
	if e.chance(35) {
		s[ukey("AP")] = e.rangeIn(0, 300)
	}
	if e.chance(30) {
		s[ukey("DP")] = e.rangeIn(0, 150)
	}
	if e.chance(30) {
		s[ukey("HIT")] = e.rangeIn(0, 140)
	}
	if e.chance(30) {
		s[ukey("EV")] = e.rangeIn(0, 80)
	}
	if e.chance(25) {
		s[ukey("CritPct")] = e.rangeIn(0, 100)
	}
	if e.chance(25) {
		s[ukey("Lv")] = e.rangeIn(0, 45)
	}
	if e.chance(20) {
		s[ukey("ExpPerLevel")] = e.rangeIn(0, 150)
	}
	if e.chance(15) {
		s[ukey("BuffAPPct")], s[ukey("BuffDPPct")] = e.rangeIn(0, 30), e.rangeIn(0, 30)
		s[ukey("BuffHit")], s[ukey("BuffEV")] = e.rangeIn(0, 30), e.rangeIn(0, 30)
	}
	if e.chance(10) {
		s[ukey("NativeTransient")] = []int{e.intn(4), e.intn(4), e.intn(4), e.intn(4), e.intn(3), e.intn(3)}
	}
	if attacker && e.chance(30) {
		// 武器槽：原版物品 ID 小於 0x80 且旗標 0x40 才算已裝備。
		inv := []int{e.intn(0x80), e.intn(0x100), e.intn(0x80)}
		s[ukey("Inventory")] = inv
		if e.chance(70) {
			s[ukey("NativeInventoryFlags")] = []int{[]int{0x40, 0, 0x80}[e.intn(3)], 0x40, 0x40}
		} else {
			s[ukey("NativeInventoryFlags")] = nil
			s[ukey("Equipped")] = []bool{e.chance(50), e.chance(50), true}
		}
	}
	if e.chance(20) {
		s[ukey("HasNativeRecordClass")] = true
		s[ukey("NativeRecordClass")] = []int{0, 3, 7, 9, 19, 24, 25, 30}[e.intn(8)]
	}
	if e.chance(15) {
		s[ukey("HasNativeRecordRace")] = e.chance(70)
		s[ukey("NativeRecordRace")] = e.intn(7)
	}
	if e.chance(15) {
		s[ukey("HasBattleFig")] = true
		s[ukey("BattleFig")] = []int{28, 4, 9, 0x1e, 96, 102}[e.intn(6)]
	}
}

// nativeEdit 補齊 planNativePhysicalExperience 需要的原版來源；有時故意留缺。
func (e *exporter) nativeEdit(s obj, own bool) {
	if e.chance(90) {
		s[ukey("HasNativeRecordByte5")], s[ukey("NativeRecordByte5")] = true, []int{0, 0, 0, 1, 0x80}[e.intn(5)]
	}
	if e.chance(90) {
		s[ukey("HasNativeRecordByte6")] = true
		if own {
			s[ukey("NativeRecordByte6")] = 2
		} else {
			s[ukey("NativeRecordByte6")] = []int{0, 1}[e.intn(2)]
		}
	}
	if own {
		s[ukey("Camp")] = 0
		s[ukey("HasNativeRecordClass")] = e.chance(95)
		s[ukey("NativeRecordClass")] = e.rangeIn(1, 26)
		s[ukey("HasNativeRecordByte8")], s[ukey("NativeRecordByte8")] = true, []int{0, 5, 28}[e.intn(3)]
		s[ukey("Exp")] = []any{0, e.rangeIn(0, 99), 99, 99.5}[e.intn(4)]
		s[ukey("Lv")] = e.rangeIn(1, 41)
		s[ukey("HasBattleFig")], s[ukey("BattleFig")] = true, e.intn(68)
	}
}

func (e *exporter) balance() any {
	if !e.chance(25) {
		return nil
	}
	g := []int{e.rangeIn(0, 300), e.rangeIn(0, 300), e.rangeIn(0, 300), e.rangeIn(0, 300), e.rangeIn(0, 300)}
	if e.chance(30) {
		g = []int{100, 100, 100, 100, 100}
	}
	chars := [][]int{}
	if e.chance(40) {
		chars = append(chars, []int{e.intn(32), e.rangeIn(0, 300), e.rangeIn(0, 300), e.rangeIn(0, 300),
			e.rangeIn(0, 300), e.rangeIn(0, 300)})
	}
	skill := [][]int{}
	if e.chance(50) {
		skill = append(skill, []int{e.intn(32), []int{0, 4, 9, 22}[e.intn(4)], []int{0, 50, 100}[e.intn(3)]})
	}
	return obj{"growth": g, "enemy_percent": 100, "characters": chars, "skill_hit": skill}
}

// ---- 案例 ----

func rngNext(r *mrand.Rand) int64 { return r.Int63() }

func (e *exporter) genAttackRNG(fx int, n int, fixedSeeds bool) {
	for k := 0; k < n; k++ {
		a, d := e.pickUnits(fx)
		seed := e.seed()
		if fixedSeeds {
			seed = int64(k + 1)
		}
		sp := obj{"fixture": fx, "edits": e.combatEdit(fx, a, d), "cancel": e.chance(20), "growth": []any{nil, "rows", "slots"}[e.intn(3)],
			"learn": e.chance(30), "balance": e.balance()}
		st := e.buildState(sp)
		before := snapshot(st.Units)
		r := mrand.New(mrand.NewSource(seed))
		res := st.AttackWithRNG(st.Units[a], st.Units[d], r)
		sp["a"], sp["d"], sp["seed"] = a, d, seed
		e.add("attack_rng", sp, obj{"result": conv(st.Units, res), "changed": changed(before, st.Units), "rng_next": rngNext(r)})
	}
}

func (e *exporter) genAttackNativeExp(fx int, n int) {
	for k := 0; k < n; k++ {
		a, d := e.pickUnits(fx)
		edits := e.combatEdit(fx, a, d)
		e.nativeEdit(edits[0]["set"].(obj), e.chance(80))
		sp := obj{"fixture": fx, "edits": edits, "cancel": e.chance(20), "growth": []any{nil, "rows", "slots"}[e.intn(3)],
			"learn": e.chance(30)}
		st := e.buildState(sp)
		before := snapshot(st.Units)
		seed := e.seed()
		r := mrand.New(mrand.NewSource(seed))
		res, err := st.AttackWithNativeExperience(st.Units[a], st.Units[d], r)
		sp["a"], sp["d"], sp["seed"] = a, d, seed
		e.add("attack_native_exp", sp, obj{"result": conv(st.Units, res), "err": errv(err),
			"changed": changed(before, st.Units), "rng_next": rngNext(r)})
	}
}

func (e *exporter) genNativePhysical(fx int, n int) {
	for k := 0; k < n; k++ {
		a, d := e.pickUnits(fx)
		edits := e.combatEdit(fx, a, d)
		withExp := e.chance(70)
		if withExp {
			attackerOwn := e.chance(50)
			e.nativeEdit(edits[0]["set"].(obj), attackerOwn)
			e.nativeEdit(edits[1]["set"].(obj), !attackerOwn && e.chance(80))
			if e.chance(50) {
				// 讓守方帶武器射程 1 的已裝備槽，容易觸發反擊。
				edits[1]["set"].(obj)[ukey("Inventory")] = []int{e.intn(0x60)}
				edits[1]["set"].(obj)[ukey("NativeInventoryFlags")] = []int{0x40}
			}
		}
		sp := obj{"fixture": fx, "edits": edits, "cancel": e.chance(25), "growth": []any{nil, "rows", "slots"}[e.intn(3)],
			"learn": e.chance(30), "observer": e.chance(20), "acc": []int{0, 0, e.intn(120)}[e.intn(3)],
			"balance": e.balance()}
		st := e.buildState(sp)
		before := snapshot(st.Units)
		seed := e.seed()
		state := e.u16()
		r := mrand.New(mrand.NewSource(seed))
		var res battle.NativePhysicalAttackResult
		var err error
		if withExp {
			res, err = st.AttackNativePhysicalWithExperience(st.Units[a], st.Units[d], uint16(state), r)
		} else {
			res, err = st.AttackNativePhysical(st.Units[a], st.Units[d], uint16(state), r)
		}
		eligible := st.NativePhysicalCounterattackEligible(st.Units[a], st.Units[d])
		sp["a"], sp["d"], sp["seed"], sp["state"], sp["with_exp"] = a, d, seed, state, withExp
		e.add("attack_native_physical", sp, obj{"result": conv(st.Units, res), "err": errv(err),
			"changed": changed(before, st.Units), "acc": st.NativeExperienceAccumulator,
			"eligible_after": eligible, "rng_next": rngNext(r)})
	}
}

func (e *exporter) genCounterEligible(fx int, n int) {
	for k := 0; k < n; k++ {
		a, d := e.pickUnits(fx)
		edits := e.combatEdit(fx, a, d)
		if e.chance(50) {
			edits[1]["set"].(obj)[ukey("Inventory")] = []int{e.intn(0x80)}
			edits[1]["set"].(obj)[ukey("NativeInventoryFlags")] = []int{0x40}
		}
		sp := obj{"fixture": fx, "edits": edits, "no_items": e.chance(10)}
		st := e.buildState(sp)
		sp["a"], sp["d"] = a, d
		e.add("counter_eligible", sp, obj{"eligible": st.NativePhysicalCounterattackEligible(st.Units[a], st.Units[d])})
	}
}

func (e *exporter) genPlayerAttackField(fx int, n int) {
	for k := 0; k < n; k++ {
		u := e.intn(len(e.fixtures[fx].Units))
		edits := []obj{}
		if e.chance(60) {
			edits = append(edits, obj{"unit": u, "set": set("Inventory", []int{e.intn(0x100), e.intn(0x80)},
				"NativeInventoryFlags", []int{[]int{0x40, 0}[e.intn(2)], 0x40})})
		}
		sp := obj{"fixture": fx, "edits": edits, "no_items": e.chance(10)}
		st := e.buildState(sp)
		field, err := st.NativePlayerAttackTargetField(st.Units[u])
		sp["u"] = u
		e.add("player_attack_field", sp, obj{"field": hexs(field), "err": errv(err)})
	}
}

func (e *exporter) genTerrainCells(fx int) {
	f := e.fixtures[fx]
	st := e.buildState(obj{"fixture": fx, "no_items": true})
	cells := [][2]int{}
	for y := -1; y <= f.H; y++ {
		for x := -1; x <= f.W; x++ {
			ap, dp := st.TerrainAPDPPct(x, y)
			cells = append(cells, [2]int{ap, dp})
		}
	}
	e.add("terrain_cells", obj{"fixture": fx}, obj{"cells": cells})
}

func (e *exporter) genCast(fx int, n int) {
	for k := 0; k < n; k++ {
		c := e.intn(len(e.fixtures[fx].Units))
		s := set("OnField", true, "MP", e.rangeIn(0, 80), "Sealed", e.chance(10), "Lv", e.rangeIn(0, 40))
		if e.chance(40) {
			s[ukey("Camp")] = e.intn(3)
		}
		edits := []obj{{"unit": c, "set": s}}
		// 幾個目標改成殘血、在場，製造擊倒與治療上限。
		for t := 0; t < 3; t++ {
			u := e.intn(len(e.fixtures[fx].Units))
			if u == c {
				continue
			}
			ts := set("OnField", e.chance(85), "HP", e.rangeIn(0, 120), "MaxHP", e.rangeIn(80, 200))
			e.statEdits(ts, false)
			edits = append(edits, obj{"unit": u, "set": ts})
		}
		spell := e.spells[e.intn(len(e.spells))]
		if e.chance(15) {
			spell = battle.Spell{ID: e.rangeIn(0, 40), Dmg: e.rangeIn(0, 300), Hit: []int{0, 50, 90, 100}[e.intn(4)],
				Dist: e.rangeIn(0, 6), Range: e.rangeIn(0, 3), MP: e.rangeIn(0, 30), Target: []int{0, 1, 3}[e.intn(3)]}
		}
		f := e.fixtures[fx]
		cx, cy := num(e.baseOf(fx, c)["X"]), num(e.baseOf(fx, c)["Y"])
		if e.chance(70) {
			t := e.intn(len(f.Units))
			cx, cy = num(e.baseOf(fx, t)["X"]), num(e.baseOf(fx, t)["Y"])
		}
		sp := obj{"fixture": fx, "edits": edits, "cancel": e.chance(20), "growth": []any{nil, "rows", "slots"}[e.intn(3)],
			"learn": e.chance(30), "balance": e.balance()}
		st := e.buildState(sp)
		before := snapshot(st.Units)
		seed := e.seed()
		tx, ty := e.rangeIn(-1, f.W), e.rangeIn(-1, f.H)
		inRange := st.InCastRange(st.Units[c], spell, tx, ty)
		r := mrand.New(mrand.NewSource(seed))
		res := st.CastArea(st.Units[c], cx, cy, spell, r)
		sp["caster"], sp["cx"], sp["cy"], sp["seed"], sp["spell"], sp["tx"], sp["ty"] = c, cx, cy, seed, conv(nil, spell), tx, ty
		e.add("cast_area", sp, obj{"results": conv(st.Units, res), "nil": res == nil, "in_range": inRange,
			"changed": changed(before, st.Units), "rng_next": rngNext(r)})
	}
}

func (e *exporter) genAISpells(fx int, n int) {
	for k := 0; k < n; k++ {
		u := e.intn(len(e.fixtures[fx].Units))
		inv := []int{e.intn(0x100), e.intn(0x100), e.intn(0x100), 0x61}
		pairs := [][2]int{{0x61, e.intn(36)}, {inv[0], e.intn(40)}, {inv[1], e.intn(36)}}
		edits := []obj{{"unit": u, "set": set("Inventory", inv)}}
		for t := 0; t < 3; t++ {
			v := e.intn(len(e.fixtures[fx].Units))
			edits = append(edits, obj{"unit": v, "set": set("Poisoned", e.chance(30), "Paralyzed", e.chance(30),
				"Acted", e.chance(40), "HP", e.rangeIn(0, 60), "Camp", e.intn(3))})
		}
		sp := obj{"fixture": fx, "edits": edits, "spellbook": true, "ai_spell": pairs, "no_items": true}
		st := e.buildState(sp)
		spells := st.AIAvailableSpells(st.Units[u])
		spell := e.spells[e.intn(len(e.spells))]
		cands := st.AISpellCandidates(st.Units[u], spell)
		sp["u"], sp["spell"] = u, conv(nil, spell)
		e.add("ai_spells", sp, obj{"available": conv(st.Units, spells), "candidates": conv(st.Units, cands)})
	}
}

func (e *exporter) genResult(fx int) {
	f := e.fixtures[fx]
	names := []string{"", "不存在的名字"}
	for _, u := range f.Units {
		if n, _ := u["Name"].(string); n != "" {
			names = append(names, n)
		}
	}
	for k := 0; k < 3; k++ {
		edits := []obj{}
		for t := 0; t < 4; t++ {
			v := e.intn(len(f.Units))
			edits = append(edits, obj{"unit": v, "set": set("HP", []int{0, 0, 5}[e.intn(3)], "OnField", e.chance(70))})
		}
		protect := names[e.intn(len(names))]
		sp := obj{"fixture": fx, "edits": edits, "no_items": true}
		st := e.buildState(sp)
		sp["protect"] = protect
		e.add("result", sp, obj{"result": st.Result(protect)})
	}
}

func (e *exporter) genGrowth(fx int, n int) {
	for k := 0; k < n; k++ {
		u := e.intn(len(e.fixtures[fx].Units))
		s := set("Lv", []int{1, 5, 28, 29, 39, 40, 98, 99}[e.intn(8)], "Exp", []any{0, e.rangeIn(0, 99), 50.5}[e.intn(3)])
		if e.chance(50) {
			s[ukey("Camp")] = e.intn(3)
		}
		if e.chance(50) {
			s[ukey("HasBattleFig")], s[ukey("BattleFig")] = true, []int{e.intn(68), 0x1e, 0x1f, e.intn(256)}[e.intn(4)]
		}
		if e.chance(40) {
			s[ukey("HasNativeRecordByte5")], s[ukey("NativeRecordByte5")] = true, e.intn(2)
			s[ukey("HasNativeRecordByte6")], s[ukey("NativeRecordByte6")] = e.chance(80), e.intn(3)
		}
		if e.chance(40) {
			s[ukey("EquipmentBaseSet")] = true
			s[ukey("BaseAP")], s[ukey("BaseDP")] = e.rangeIn(0, 100), e.rangeIn(0, 100)
			s[ukey("BaseHIT")], s[ukey("BaseEV")] = e.rangeIn(0, 100), e.rangeIn(0, 100)
		}
		if e.chance(30) {
			s[ukey("HasNativeRecordWord42")], s[ukey("HasNativeRecordWord46")] = true, e.chance(50)
		}
		if e.chance(30) {
			name := e.legacyName()
			s[ukey("Name")], s[ukey("ClsName")] = name, e.legacyClass(name)
		}
		if e.chance(25) {
			s[ukey("HasNativeIdentity")], s[ukey("NativeIdentity")] = true, e.intn(34)
		}
		if e.chance(15) {
			s[ukey("AP")], s[ukey("MaxHP")] = 32760, 32765
		}
		sp := obj{"fixture": fx, "edits": []obj{{"unit": u, "set": s}}, "growth": []any{nil, "rows", "slots"}[e.intn(3)],
			"learn": e.chance(50), "observer": e.chance(30), "balance": e.balance(), "no_items": true}
		amount := []int{0, -5, e.rangeIn(1, 99), e.rangeIn(100, 450), 100}[e.intn(5)]
		famount := float64(amount) + 0.25*float64(e.intn(4))
		st := e.buildState(sp)
		before := snapshot(st.Units)
		seed := e.seed()
		state := e.u16()
		sp["u"], sp["amount"], sp["famount"], sp["seed"], sp["state"] = u, amount, famount, seed, state
		row, ok := st.NativeGrowthRowFor(st.Units[u])
		switch e.intn(4) {
		case 0:
			r := mrand.New(mrand.NewSource(seed))
			got, ups := st.AwardExp(st.Units[u], famount, r)
			e.add("award_exp", sp, obj{"got": got, "ups": conv(st.Units, ups), "row": conv(nil, row), "row_ok": ok,
				"changed": changed(before, st.Units), "rng_next": rngNext(r)})
		case 1:
			r := mrand.New(mrand.NewSource(seed))
			ups := st.GainExp(st.Units[u], float64(amount), r)
			e.add("state_gain_exp", sp, obj{"ups": conv(st.Units, ups), "row": conv(nil, row), "row_ok": ok,
				"changed": changed(before, st.Units), "rng_next": rngNext(r)})
		case 2:
			got, ups, next := st.AwardExpNative(st.Units[u], amount, uint16(state))
			e.add("award_exp_native", sp, obj{"got": got, "ups": conv(st.Units, ups), "next": next,
				"row": conv(nil, row), "row_ok": ok, "changed": changed(before, st.Units)})
		default:
			r := mrand.New(mrand.NewSource(seed))
			ups := battle.GainExp(st.Units[u], float64(amount), r)
			e.add("gain_exp", sp, obj{"ups": conv(st.Units, ups), "changed": changed(before, st.Units), "rng_next": rngNext(r)})
		}
	}
}

// ---- 純函式 ----

func (e *exporter) genPure() {
	for k := 0; k < 400; k++ {
		in := battle.NativePhysicalRoll{
			AttackerAP: e.rangeIn(-3, 300), DefenderDP: e.rangeIn(-3, 200), AttackerHit: e.rangeIn(0, 140),
			DefenderEV: e.rangeIn(0, 80), AttackerCritPct: e.rangeIn(0, 40),
			AttackerTerrainAPPct: []int{0, 5, -5, 10}[e.intn(4)], DefenderTerrainDPPct: []int{0, 5, -5, 10}[e.intn(4)],
			Weapon:   battle.NativePhysicalWeapon{Effect: e.intn(6), Param: e.rangeIn(0, 100), Reach: e.intn(3)},
			RNGState: uint16(e.u16()),
		}
		res, err := battle.RollNativePhysicalDamage(in)
		e.add("roll_physical", obj{"in": conv(nil, in)}, obj{"result": conv(nil, res), "err": errv(err)})
	}
	for k := 0; k < 300; k++ {
		d := battle.NativeCounterattackDefender{X: e.intn(6), Y: e.intn(6), Byte38: []int{0, 0, 0, 2}[e.intn(4)],
			HasWeapon: e.chance(70), WeaponReach: e.intn(3), AliveHP: e.rangeIn(-1, 5)}
		ax, ay := e.intn(6), e.intn(6)
		e.add("counter_raw", obj{"ax": ax, "ay": ay, "d": conv(nil, d)},
			obj{"eligible": battle.NativeCounterattackEligible(ax, ay, d)})
	}
	for code := 0; code < 256; code++ {
		ap, dp, ok := battle.NativeTerrainAPDPPct(byte(code))
		e.add("terrain_code", obj{"code": code}, obj{"ap": ap, "dp": dp, "ok": ok})
	}
	for k := 0; k < 300; k++ {
		args := []int{e.rangeIn(-2, 50), e.rangeIn(-2, 50), e.rangeIn(-5, 400), e.rangeIn(-2, 300), e.rangeIn(-2, 150)}
		e.add("exp_formula", obj{"args": args}, obj{
			"attack":   battle.AttackExp(args[0], args[1], args[2], args[3], args[4]),
			"teleport": battle.TeleportExp(args[0], args[1]),
			"action":   battle.ActionExp(args[0], args[1]),
		})
	}
	for k := 0; k < 120; k++ {
		count := []int{0, 1, 2, 5, 30, 0x100, 0x101}[e.intn(7)]
		size := count * 0x50
		if e.chance(15) {
			size += []int{-1, 1, 0x50}[e.intn(3)]
		}
		if size < 0 {
			size = 0
		}
		records := e.bytesN(size)
		for i := 0; i+6 < len(records); i += 0x50 {
			records[i+5] &= byte([]int{0xfe, 0xff, 0x81}[e.intn(3)])
			records[i+6] = byte([]int{0, 1, 2, 0}[e.intn(4)])
		}
		got, err := battle.NativeBattleResultCode205B4(records, count)
		e.add("battle_result", obj{"records": hexs(records), "count": count}, obj{"code": got, "err": errv(err)})
	}
}

func (e *exporter) genLoads() {
	real := []string{"assets/data/class_change_growth.json", "assets/data/native_growth_slots.json"}
	for _, p := range real {
		rows, err := battle.LoadNativeGrowthRows(p)
		slots, serr := battle.LoadNativeGrowthSlots(p)
		e.add("load_growth", obj{"path": p}, obj{"rows": conv(nil, rows), "err": errv(err),
			"slots": conv(nil, slots), "slots_err": errv(serr)})
	}
	// 合成檔：空表、idx 錯位、不合法範圍、負跨距、256 槽與非 256 槽。
	texts := []string{
		`[]`,
		`[{"idx":0,"ap":[1,2],"dp":[1,1],"dx":[0,0],"hp":[3,5],"mp":[0,0]},{"idx":2}]`,
		`[{"idx":0,"ap":[-1,2]}]`,
		`[{"idx":0,"ap":[5,3],"dp":[2,2],"dx":[1,1],"hp":[300,4],"mp":[0,9]}]`,
		`not json`,
		`[{"idx":0,"AP":[4,7],"DP":[2,3]}]`,
	}
	slots := make([]obj, 256)
	for i := range slots {
		slots[i] = obj{"idx": i, "ap": []int{e.intn(20), e.intn(20)}, "dp": []int{e.intn(20), e.intn(20)},
			"dx": []int{e.intn(5), e.intn(5)}, "hp": []int{e.intn(30), e.intn(30)}, "mp": []int{e.intn(9), e.intn(9)}}
	}
	data, _ := json.Marshal(slots)
	texts = append(texts, string(data))
	slots[200]["hp"] = []int{256, 3}
	data, _ = json.Marshal(slots)
	texts = append(texts, string(data))
	for k, text := range texts {
		name := fmt.Sprintf("growth-%d.json", k)
		path := filepath.Join(e.tmp, name)
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			panic(err)
		}
		rows, err := battle.LoadNativeGrowthRows(path)
		slotRows, serr := battle.LoadNativeGrowthSlots(path)
		e.add("load_growth_text", obj{"name": name, "text": text}, obj{"rows": conv(nil, rows), "err": errv(err),
			"slots": conv(nil, slotRows), "slots_err": errv(serr)})
	}
	spells, err := battle.LoadSpells("assets/spells.json")
	e.add("load_spells", obj{"path": "assets/spells.json"}, obj{"spells": conv(nil, spells), "err": errv(err)})
}

func main() {
	out := flag.String("out", "", "輸出 JSON 路徑")
	seed := flag.Uint64("seed", 20261114, "PCG 種子")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "需要 -out")
		os.Exit(2)
	}
	tmp, err := os.MkdirTemp("", "fd2-golden-rules-e-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x52554c45)), tmp: tmp}
	real := e.genReal()
	realCount := len(e.fixtures)
	e.genSyntheticFixtures(16)
	e.genPure()
	e.genLoads()
	for fx := range e.fixtures {
		per := 1
		if fx >= realCount {
			per = 2
		}
		e.genAttackRNG(fx, 6*per, false)
		e.genAttackNativeExp(fx, 6*per)
		e.genNativePhysical(fx, 12*per)
		e.genCounterEligible(fx, 4*per)
		e.genPlayerAttackField(fx, 3)
		e.genTerrainCells(fx)
		e.genCast(fx, 6*per)
		e.genAISpells(fx, 2)
		e.genResult(fx)
		e.genGrowth(fx, 10*per)
	}
	// 固定種子 1～100：同一組單位以 Go 測試相同的種子迴圈重跑（R2 完成條件的決定性核心）。
	for fx := 0; fx < realCount; fx += 8 {
		e.genAttackRNG(fx, 100, true)
	}
	doc := obj{"schema": "fd2-golden-rules-e/1", "seed": *seed, "real": real, "fixtures": e.fixtureList(), "cases": e.cases}
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
