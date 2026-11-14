// fd2-golden-campaign-c 輸出 internal/campaign「C 組」（存檔、CONTINUE、JOIN）與 internal/ending
// （結局演出）的黃金差分資料，供 Rust 改寫 remake-rs/crates/fd2-campaign 逐欄比對
// （docs/goal/119-goal-rust-bevy-rewrite-20261113.md）。
//
// Go 版凍結於 v.1.1.11；本工具只呼叫 campaign／ending 的公開 API，不改任何遊戲行為。
//   - 真實輸入：remake/assets 的 data、maps、scenarios、endings；原版 FLAME2 的 FD2.SAV（使用者合法存檔，
//     與 Go 測試 FD2_NATIVE_SAVE_FIXTURE 同一份）、FDOTHER.DAT、ANI.DAT、FDTXT.DAT、DATO.DAT；分離素材包的
//     surfaces、ui、animations、portraits、text、fonts、palette、fields、sprites。檔案清單排序後連同大小與
//     SHA-256 寫進 JSON，Rust 端讀同一批檔案前先核對。
//   - 產生輸入：以固定種子的 PCG 產生合成存檔槽、持續隊伍、JOIN 殘值與 CONTINUE 快照的變體，連同結果寫進
//     同一份 JSON，Rust 端不需重現亂數。
//   - 結局演出：以 assets/endings 的時間軸、蒙太奇與片尾跑固定步數，每步記 VGA 與調色盤的 SHA-256。
//
// 錯誤欄位是 Go err.Error() 原文；Go 標準函式庫產生的錯誤（讀檔、JSON 解碼）只比接受或拒收，見 Rust 端。
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
	"sort"
	"strconv"

	"github.com/wicanr2/fd2_re/remake/internal/afm"
	"github.com/wicanr2/fd2_re/remake/internal/battle"
	"github.com/wicanr2/fd2_re/remake/internal/campaign"
	"github.com/wicanr2/fd2_re/remake/internal/dato"
	"github.com/wicanr2/fd2_re/remake/internal/ending"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
	"github.com/wicanr2/fd2_re/remake/internal/fdsave"
	"github.com/wicanr2/fd2_re/remake/internal/fdtxt"
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
	flame    string
	pack     string
	cases    []caseJSON
	files    []obj
	catalog  *campaign.NativeCharacterCatalog
	gate     *campaign.NativeIntermissionGateTable
	join     campaign.NativeJoinConstructorTable
	itemRows []byte
	plain    []byte
	current  fdsave.CurrentSnapshot
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

// recordTree 記下 root/rel 底下所有檔案（排序在 main 最後統一做）。
func (e *exporter) recordTree(prefix, root, rel string) {
	base := filepath.Join(root, rel)
	if err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		r, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		e.record(prefix+"/"+filepath.ToSlash(r), path)
		return nil
	}); err != nil {
		panic(err)
	}
}

func (e *exporter) packPath(rel string) string { return filepath.Join(e.pack, rel) }

// ---- 合成存檔槽與持續隊伍 ----

func (e *exporter) itemTable(variant string) []byte {
	switch variant {
	case "full":
		return e.itemRows
	case "short":
		return e.itemRows[:battle.NativeItemEffectRowSize]
	case "empty":
		return nil
	}
	panic("item variant " + variant)
}

func (e *exporter) pickItemVariant() string {
	switch {
	case e.chance(85):
		return "full"
	case e.chance(50):
		return "short"
	}
	return "empty"
}

// genRecord 以真實持續紀錄為模板，改寫身分、職業與部分原始位元組。
func (e *exporter) genRecord() fdsave.PersistentRecord {
	var record fdsave.PersistentRecord
	if e.chance(70) {
		record = e.current.PersistentRecords[e.intn(4)]
	} else {
		copy(record.Raw[:], e.bytesN(fdsave.UnitSize))
	}
	if e.chance(97) {
		record.Raw[8] = byte(e.intn(32))
	} else {
		record.Raw[8] = byte(e.rangeInt(32, 255))
	}
	if e.chance(97) {
		record.Raw[0x20] = byte(e.intn(29))
	} else {
		record.Raw[0x20] = byte(e.rangeInt(29, 255))
	}
	for k := 0; k < e.intn(4); k++ {
		record.Raw[e.intn(fdsave.UnitSize)] = byte(e.intn(256))
	}
	return record
}

func slotInput(s fdsave.ChapterSlotSnapshot) obj {
	records := make([]byte, 0, fdsave.RosterSize)
	for _, r := range s.Records {
		records = append(records, r.Raw[:]...)
	}
	return obj{
		"slot": s.Slot, "chapter": s.Verified.Chapter, "roster_count": s.Verified.RosterCount,
		"currency": s.Verified.Currency, "hud_gate_a": s.Verified.HUDGateA, "raw_53af9": s.Verified.Raw53AF9,
		"raw_51e61": s.Verified.Raw51E61, "raw_51e62": s.Verified.Raw51E62,
		"metadata": hexs(s.Metadata[:]), "records": hexs(records), "source_sha256": s.SourceSaveSHA256,
	}
}

func (e *exporter) genSlot() fdsave.ChapterSlotSnapshot {
	var s fdsave.ChapterSlotSnapshot
	s.Slot = e.intn(4)
	if e.chance(5) {
		s.Slot = 4
	}
	s.Verified.Chapter = byte(e.rangeInt(0, 29))
	if e.chance(5) {
		s.Verified.Chapter = byte(e.rangeInt(30, 255))
	}
	s.Verified.RosterCount = byte(e.rangeInt(0, 6))
	if e.chance(4) {
		s.Verified.RosterCount = 33
	}
	s.Verified.Currency = e.rng.Uint32()
	s.Verified.HUDGateA, s.Verified.Raw53AF9 = byte(e.intn(256)), byte(e.intn(256))
	s.Verified.Raw51E61, s.Verified.Raw51E62 = byte(e.intn(256)), byte(e.intn(256))
	copy(s.Metadata[:], e.bytesN(fdsave.MetadataSize))
	used := map[byte]bool{}
	for i := range s.Records {
		s.Records[i] = e.genRecord()
		if i < int(s.Verified.RosterCount) && e.chance(90) {
			for used[s.Records[i].Raw[8]] {
				s.Records[i].Raw[8] = byte(e.intn(32))
			}
		}
		used[s.Records[i].Raw[8]] = true
	}
	if s.Verified.Chapter == 28 && e.chance(60) {
		s.SourceSaveSHA256 = []string{
			"f46d9c54d3037f84f05d72714569c282e63f39bf125251a9cf5cd9593ff3241f",
			"df50919111cccb37a00215dea248a069617aa222545d0f4493ab4b00dc5b819f",
		}[e.intn(2)]
	} else {
		s.SourceSaveSHA256 = digest(e.bytesN(8))
	}
	return s
}

// realSlot 以真實存檔的目前持續名冊組出一個章節槽（四個槽在這份存檔裡都是空槽）。
func (e *exporter) realSlot() fdsave.ChapterSlotSnapshot {
	var s fdsave.ChapterSlotSnapshot
	s.Verified.Chapter = e.current.Header.Chapter
	s.Verified.RosterCount = e.current.Header.PersistentCount
	s.Verified.Currency = e.current.Header.Currency
	s.Verified.HUDGateA = e.current.Header.HUDGateA
	s.Verified.Raw53AF9 = e.current.Header.Raw53AF9
	s.Verified.Raw51E61, s.Verified.Raw51E62 = e.current.Header.Raw51E61, e.current.Header.Raw51E62
	s.Records = e.current.PersistentRecords
	s.SourceSaveSHA256 = fmt.Sprintf("%x", sha256.Sum256(fdsave.XOREnvelope(e.plain)))
	return s
}

func (e *exporter) genGraph() ([][2]string, *campaign.Campaign) {
	var nodes [][2]string
	graph := &campaign.Campaign{Nodes: map[string]*campaign.Node{}}
	for chapter := 1; chapter <= 30; chapter++ {
		for _, kind := range []string{"town", "preparation"} {
			if e.chance(8) {
				continue
			}
			id := fmt.Sprintf("%s_ch%02d", kind, chapter)
			typ := kind
			if e.chance(3) {
				typ = "battle"
			}
			nodes = append(nodes, [2]string{id, typ})
			graph.Nodes[id] = &campaign.Node{Type: typ}
		}
	}
	return nodes, graph
}

func (e *exporter) genRestorePlans() {
	for n := 0; n < 80; n++ {
		snapshot := e.genSlot()
		if n == 0 {
			snapshot = e.realSlot()
		}
		nodes, graph := e.genGraph()
		var graphArg *campaign.Campaign = graph
		nilGraph := n%37 == 5
		if nilGraph {
			graphArg = nil
		}
		plan, err := campaign.BuildNativeChapterSlotRestorePlan(snapshot, e.catalog, e.gate, graphArg)
		result := obj{"err": errv(err)}
		if err == nil {
			result["plan"] = gj(plan)
		}
		input := slotInput(snapshot)
		input["nodes"], input["nil_graph"] = nodes, nilGraph
		e.add("restore_plan", input, result)
	}
}

// mutateUnit 對持續隊伍單位做會進存檔的改動（座標、能力、物品、旗標）。
func (e *exporter) mutateUnit(u *battle.Unit, faults bool) {
	fault := func() bool { return faults && e.chance(30) }
	if e.chance(50) {
		u.X, u.Y = e.rangeInt(0, 40), e.rangeInt(0, 40)
		u.NativeMapPresentation = battle.NativeMapPresentationState{X: byte(u.X), Y: byte(u.Y), Pose: byte(e.intn(4)), Motion: byte(e.intn(7))}
		u.HasNativeMapPresentation = e.chance(70)
	}
	if fault() {
		u.X = 300
	}
	if e.chance(40) {
		u.MapSelectorSlot, u.HasMapSelectorSlot = e.rangeInt(-1, 260), e.chance(80)
	}
	if e.chance(50) {
		u.HP = e.rangeInt(0, max(u.MaxHP, 0)+5)
		u.MP = e.rangeInt(0, max(u.MaxMP, 0)+5)
	}
	if e.chance(40) {
		u.Lv = e.rangeInt(1, 40)
		u.Exp = float64(e.intn(100))
	}
	if e.chance(40) && len(u.InventorySlots) == 8 {
		slot := e.intn(8)
		u.InventorySlots[slot] = e.intn(200)
		u.NativeInventoryFlags[slot] = []int{0, 0x40, 0x80}[e.intn(3)]
	}
	if e.chance(40) {
		u.BaseAP, u.BaseDP = e.rangeInt(-10, 300), e.rangeInt(-10, 300)
		u.EquipmentBaseSet = e.chance(80)
	}
	if fault() {
		u.BaseAP = 0x8000
	}
	if e.chance(20) {
		u.NativeRecordDeathEffect, u.HasNativeRecordDeathEffect = [3]byte{byte(e.intn(256)), byte(e.intn(256)), byte(e.intn(256))}, true
	}
	if e.chance(25) {
		u.NativeRecordByte34, u.HasNativeRecordByte34 = byte(e.intn(256)), e.chance(80)
		u.NativeRecordByte3D, u.HasNativeRecordByte3D = byte(e.intn(256)), e.chance(80)
	}
	if e.chance(20) {
		u.NativeRecordByte5 = byte(e.intn(256))
	}
	if fault() {
		u.HasNativeRecordByte5 = false
	}
	if e.chance(15) {
		u.NativeJoinPersistentPending = !u.NativeJoinPersistentPending
	}
	if e.chance(20) {
		u.HasBattleFig = !u.HasBattleFig
	}
	if fault() {
		u.NativeRecordByte8 = byte(e.intn(256))
	}
}

func unitList(roster map[int]battle.Unit) [][2]any {
	keys := make([]int, 0, len(roster))
	for k := range roster {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := make([][2]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, [2]any{k, gj(roster[k])})
	}
	return out
}

func (e *exporter) genSlotSaves() {
	for n := 0; n < 70; n++ {
		baseline := e.genSlot()
		if n == 0 {
			baseline = e.realSlot()
		}
		if baseline.Verified.RosterCount > 6 && e.chance(70) {
			baseline.Verified.RosterCount = 3
		}
		// 約四分之三的案例不注入錯誤，讓成功路徑佔多數；其餘案例逐項注入會被拒收的欄位。
		faults := e.chance(25)
		roster := map[int]battle.Unit{}
		var order []int
		count := int(baseline.Verified.RosterCount)
		if count > fdsave.RosterUnits {
			count = fdsave.RosterUnits
		}
		if !faults && n > 0 {
			for i := 0; i < count; i++ {
				baseline.Records[i].Raw[8] = byte(i * 5 % 32)
				baseline.Records[i].Raw[0x20] = byte(e.intn(29))
			}
		}
		for i := 0; i < count; i++ {
			unit, err := campaign.MaterializeNativePersistentPartyRecord(baseline.Records[i], e.catalog)
			if err != nil || (faults && e.chance(10)) {
				continue
			}
			if n > 0 {
				e.mutateUnit(&unit, faults)
			}
			key := unit.NativeIdentity
			if faults && e.chance(10) {
				unit.HasNativeIdentity = false
			}
			roster[key] = unit
			order = append(order, key)
		}
		for k := 0; k < e.intn(3); k++ {
			id := e.intn(34)
			if _, ok := roster[id]; ok {
				continue
			}
			var residual fdsave.PersistentRecord
			if e.chance(50) {
				copy(residual.Raw[:], e.bytesN(fdsave.UnitSize))
			}
			unit, err := e.join.MaterializePersistentUnitOn(residual, id, battle.Unit{}, e.itemRows)
			if err != nil {
				continue
			}
			if e.chance(40) {
				unit.NativeJoinPersistentPending = false
			}
			if e.chance(30) {
				e.mutateUnit(&unit, faults)
			}
			roster[id] = unit
			if !faults || e.chance(60) {
				order = append(order, id)
			}
		}
		if e.chance(10) && len(order) > 1 {
			order = append(order, order[0])
		}
		chapter, gold := byte(e.intn(30)), e.rng.Uint32()
		variant := e.pickItemVariant()
		slot, err := campaign.BuildNativeChapterSlot(baseline, roster, order, chapter, gold, e.join, e.itemTable(variant))
		result := obj{"err": errv(err)}
		if orderDependent(err, func() error {
			_, err := campaign.BuildNativeChapterSlot(baseline, roster, order, chapter, gold, e.join, e.itemTable(variant))
			return err
		}) {
			result["err"], result["err_order_dependent"] = "map-order", true
		}
		if err == nil {
			result["roster"], result["metadata"] = hexs(slot.Roster), hexs(slot.Metadata)
		}
		input := slotInput(baseline)
		input["roster"], input["join_order"], input["chapter"], input["gold"], input["items"] = unitList(roster), order, chapter, gold, variant
		e.add("slot_save", input, result)

		// 依鍵排序取第一名成員（Go map 迭代順序不固定，會讓黃金檔不可重現）。
		keys := make([]int, 0, len(roster))
		for key := range roster {
			keys = append(keys, key)
		}
		sort.Ints(keys)
		for _, key := range keys {
			unit := roster[key]
			baselineRecord := baseline.Records[e.intn(fdsave.RosterUnits)]
			if e.chance(60) {
				baselineRecord.Raw[8] = byte(unit.NativeIdentity)
			}
			record, err := campaign.EncodeNativePersistentRecord(baselineRecord, &unit)
			result := obj{"err": errv(err)}
			if err == nil {
				result["record"] = hexs(record.Raw[:])
			}
			e.add("encode_record", obj{"key": key, "baseline": hexs(baselineRecord.Raw[:]), "unit": gj(unit)}, result)
			break
		}
	}
	record, err := campaign.EncodeNativePersistentRecord(fdsave.PersistentRecord{}, nil)
	e.add("encode_record", obj{"baseline": hexs(make([]byte, fdsave.UnitSize)), "unit": nil},
		obj{"err": errv(err), "record": nilIfErr(err, hexs(record.Raw[:]))})
}

// orderDependent 對失敗的呼叫重做 64 次；錯誤文字不一致（Go map 迭代順序決定先報哪一個）時回傳 true。
// 只用在失敗時不改任何狀態的函式。
func orderDependent(err error, call func() error) bool {
	if err == nil {
		return false
	}
	for k := 0; k < 64; k++ {
		if again := call(); again == nil || again.Error() != err.Error() {
			return true
		}
	}
	return false
}

func nilIfErr(err error, v any) any {
	if err != nil {
		return nil
	}
	return v
}

func (e *exporter) genJoin() {
	base, err := campaign.LoadNativeJoinBaseTable("assets/data/native_join_base_units.json")
	if err != nil {
		panic(err)
	}
	for id := -1; id <= 33; id++ {
		unit, err := base.LoadBaseUnit(id)
		result := obj{"err": errv(err)}
		if err == nil {
			result["unit"] = gj(unit)
		}
		e.add("join_base", obj{"id": id}, result)
	}
	for n := 0; n < 90; n++ {
		id := e.rangeInt(-2, 34)
		if n < 32 {
			id = n
		}
		var residual fdsave.PersistentRecord
		if e.chance(60) {
			copy(residual.Raw[:], e.bytesN(fdsave.UnitSize))
		}
		variant := e.pickItemVariant()
		record, err := e.join.MaterializePersistentRecordOn(residual, id, e.itemTable(variant))
		e.add("join_record", obj{"id": id, "residual": hexs(residual.Raw[:]), "items": variant},
			obj{"err": errv(err), "record": nilIfErr(err, hexs(record.Raw[:]))})
		baseUnit := battle.Unit{}
		if loaded, err := base.LoadBaseUnit(id); err == nil && e.chance(70) {
			baseUnit = loaded
		} else if e.chance(50) {
			baseUnit.Name, baseUnit.AtkMin, baseUnit.AtkMax, baseUnit.Fig = "基底", e.intn(30), e.rangeInt(30, 60), e.intn(200)
			baseUnit.NativeCommandMask = [5]byte{1, 2, 3, 4, 5}
			baseUnit.Exp = float64(e.intn(99))
		}
		unit, err := e.join.MaterializePersistentUnitOn(residual, id, baseUnit, e.itemTable(variant))
		result := obj{"err": errv(err)}
		if err == nil {
			result["unit"] = gj(unit)
		}
		e.add("join_unit", obj{"id": id, "residual": hexs(residual.Raw[:]), "items": variant, "base": gj(baseUnit)}, result)
	}
}

func (e *exporter) genPartyRecords() {
	for n := 0; n < 60; n++ {
		record := e.genRecord()
		if n < 5 {
			record = e.current.PersistentRecords[n]
		}
		unit, err := campaign.MaterializeNativePersistentPartyRecord(record, e.catalog)
		result := obj{"err": errv(err)}
		if err == nil {
			result["unit"] = gj(unit)
		}
		e.add("party_record", obj{"record": hexs(record.Raw[:])}, result)
	}
	e.add("catalog", obj{}, obj{"catalog": gj(e.catalog)})
	e.add("gate", obj{}, obj{"gate": gj(e.gate)})
}

// ---- CONTINUE 五段物化 ----

type continueAsset struct {
	mapPath, scenarioPath string
	chapter               int
	state                 *battle.State
	scenario              *battle.Scenario
}

func (e *exporter) loadAsset(mapPath, scenarioPath string, chapter int) continueAsset {
	state, err := battle.Load(mapPath)
	if err != nil {
		panic(err)
	}
	scenario, err := battle.LoadScenario(scenarioPath)
	if err != nil {
		panic(err)
	}
	return continueAsset{mapPath, scenarioPath, chapter, state, scenario}
}

func stateFromAsset(asset *battle.State) *battle.State {
	return &battle.State{
		W: asset.W, H: asset.H,
		NativeCompositionEventBytes: append([]byte(nil), asset.NativeCompositionEventBytes...),
		NativeFieldEventSlots:       append([]int(nil), asset.NativeFieldEventSlots...),
		NativeFieldEvents:           append([]battle.NativeFieldEvent(nil), asset.NativeFieldEvents...),
		NativeFieldEventRules:       append([]battle.NativeFieldEventRule(nil), asset.NativeFieldEventRules...),
	}
}

// summary 是 CONTINUE 五個轉接器讀寫的 State 欄位（Go 欄位名）。
func summary(s *battle.State) obj {
	runtime := make([]any, len(s.NativeRuntimeRecords))
	for i, r := range s.NativeRuntimeRecords {
		runtime[i] = []any{hexs(r.Raw[:]), r.SelectorKey, r.SelectorSlot}
	}
	controls := make([]any, len(s.NativeFieldUnitControls))
	for i, c := range s.NativeFieldUnitControls {
		controls[i] = hexs(c.Raw[:])
	}
	turns := make([]any, len(s.NativeTurnEventControls))
	for i, t := range s.NativeTurnEventControls {
		turns[i] = []any{t.Turn, t.EventID, t.RawCamp}
	}
	fields := make([]any, len(s.NativeFieldEvents))
	for i, f := range s.NativeFieldEvents {
		fields[i] = []any{f.EventID, f.Selector}
	}
	chests := make([]any, len(s.NativeChestControls))
	for i, c := range s.NativeChestControls {
		chests[i] = []any{c.RawType, c.Value}
	}
	pending := make([]int, 0, len(s.PendingGroups))
	for g, ok := range s.PendingGroups {
		if ok {
			pending = append(pending, g)
		}
	}
	sort.Ints(pending)
	return obj{
		"W": s.W, "H": s.H, "Units": gj(s.Units), "Roster": gj(s.Roster), "PendingGroups": pending,
		"NativeRuntimeRecords": runtime, "NativeFieldUnitControls": controls, "NativeTurnEventControls": turns,
		"NativeFieldEvents": fields, "NativeChestControls": chests,
		"NativeFieldControlRaw": hexs(s.NativeFieldControlRaw), "NativeEventState": hexs(s.NativeEventState[:]),
		"NativeRoundCounter": s.NativeRoundCounter, "NativeMapViewState": gj(s.NativeMapViewState),
		"NativeMapHUDState": gj(s.NativeMapHUDState), "NativeMapRangeMode": s.NativeMapRangeMode,
		"NativeMapCycleState": gj(s.NativeMapCycleState), "NativeTerrainPhaseState": gj(s.NativeTerrainPhaseState),
		"NativeTerrainFlipState": gj(s.NativeTerrainFlipState), "NativeUnitPixelShiftState": gj(s.NativeUnitPixelShiftState),
		"HasSelectorCache": s.NativeMapSelectorCache != nil, "HasSelectorError": s.NativeMapSelectorError != nil,
		"flags": []bool{s.HasNativeFieldControlState, s.HasNativeTurnEventControlState, s.HasNativeChestControlState,
			s.HasNativeRuntimeUnitProjection, s.HasNativePendingGroupBinding, s.HasNativeMapCycleState,
			s.HasNativeTerrainPhaseState, s.HasNativeMapBinaryTimingState, s.HasNativeMapViewState,
			s.HasNativeMapHUDState, s.HasNativeMapRangeModeState},
	}
}

func snapshotInput(s fdsave.CurrentSnapshot) obj {
	persistent := make([]byte, 0, fdsave.RosterSize)
	for _, r := range s.PersistentRecords {
		persistent = append(persistent, r.Raw[:]...)
	}
	runtime := make([]string, len(s.RuntimeRecords))
	for i, r := range s.RuntimeRecords {
		runtime[i] = hexs(r.Raw[:])
	}
	return obj{"field_control": hexs(s.NativeFieldControl[:]), "header": gj(s.Header),
		"persistent": hexs(persistent), "runtime": runtime, "event_state": hexs(s.NativeEventState[:])}
}

func (e *exporter) mutateSnapshot(s *fdsave.CurrentSnapshot, asset continueAsset) {
	if e.chance(60) {
		s.Header.TurnCounter = byte(e.rangeInt(0, 8))
	}
	if e.chance(40) {
		row := e.intn(8)
		s.NativeFieldControl[3+row*3+e.intn(3)] = byte(e.intn(8))
	}
	if e.chance(30) {
		s.NativeEventState[e.intn(len(s.NativeEventState))] = byte(e.intn(3))
	}
	for i := range s.RuntimeRecords {
		if !e.chance(25) {
			continue
		}
		r := &s.RuntimeRecords[i].Raw
		switch e.intn(6) {
		case 0:
			r[0], r[1] = byte(e.intn(asset.state.W+2)), byte(e.intn(asset.state.H+2))
		case 1:
			r[3], r[4] = byte(e.intn(5)), byte(e.intn(8))
		case 2:
			r[5] = byte(e.intn(256))
		case 3:
			r[6] = byte(e.intn(4))
		case 4:
			r[0x20] = byte(e.intn(31))
		case 5:
			r[0x40], r[0x41] = byte(e.intn(256)), 0
		}
	}
	if e.chance(10) {
		s.Header.CameraX = byte(e.intn(40))
	}
}

func (e *exporter) genContinue() {
	map0 := e.loadAsset("assets/maps/map0/map0_units.json", "assets/scenarios/ch01.json", 0)
	map25 := e.loadAsset("assets/maps/map25/map25_units.json", "assets/scenarios/ch26.json", 25)
	for n := 0; n < 60; n++ {
		asset := map0
		snapshot := e.current
		snapshot.RuntimeRecords = append([]fdsave.PersistentRecord(nil), e.current.RuntimeRecords...)
		if n >= 45 {
			asset = map25
			snapshot = fdsave.CurrentSnapshot{Header: fdsave.CurrentRuntimeHeader{
				TurnCounter: 1, Chapter: 25, CursorX: 7, CursorY: 4, VisibleCursorX: 7, VisibleCursorY: 4,
			}}
			snapshot.NativeFieldControl[2] = byte(len(asset.state.Units))
			for index, turn := range []byte{2, 4, 6, 8, 10, 12, 15, 16, 17} {
				copy(snapshot.NativeFieldControl[3+index*3:], []byte{turn, 57, 0})
			}
			const fieldEventOffset = 3 + 16*3
			for index, event := range asset.state.NativeFieldEvents {
				snapshot.NativeFieldControl[fieldEventOffset+index*2] = event.EventID
				snapshot.NativeFieldControl[fieldEventOffset+index*2+1] = event.Selector
			}
			if e.chance(25) {
				snapshot.NativeFieldControl[fieldEventOffset+2*2+1] = 0
			}
			snapshot.NativeEventState[12] = byte(e.intn(2))
		}
		if n > 0 {
			e.mutateSnapshot(&snapshot, asset)
		}
		ctx := fdsave.ContinueRuntimeContext{
			Chapter: int(snapshot.Header.Chapter), FieldWidth: asset.state.W, FieldHeight: asset.state.H,
			SelectorGroupCount: 256, TitleTimerTick: e.rangeInt(-0x8000, 0x7fff), HasTitleTimerTick: true,
		}
		if n == 0 {
			ctx.TitleTimerTick = 0
		}
		if e.chance(5) {
			ctx.FieldWidth++
		}
		assetChapter := asset.chapter
		if e.chance(5) {
			assetChapter++
		}
		variant := e.pickItemVariant()
		input := snapshotInput(snapshot)
		input["context"], input["map"], input["scenario"] = gj(ctx), asset.mapPath, asset.scenarioPath
		input["asset_chapter"], input["items"] = assetChapter, variant
		result := obj{}
		cri, err := fdsave.BuildContinueRuntimeInput(snapshot, ctx)
		result["preflight_err"] = errv(err)
		if err != nil {
			e.add("continue", input, result)
			continue
		}
		state := stateFromAsset(asset.state)
		var stages []any
		stage := func(name string, err error) {
			stages = append(stages, obj{"stage": name, "err": errv(err), "state": summary(state)})
		}
		// pending groups 依 map 迭代 schedule；多列同時不合法時，Go 報哪一列不固定（失敗時不改 State，可重呼叫）。
		pendingCall := func() error {
			return campaign.MaterializeNativeContinuePendingGroups(state, cri, assetChapter, asset.state, asset.scenario, e.itemTable(variant))
		}
		stage("field_boundary", campaign.MaterializeNativeContinueFieldBoundary(state, cri, assetChapter))
		stage("runtime_units", campaign.MaterializeNativeContinueRuntimeUnits(state, cri, e.catalog))
		stage("map_timing", campaign.MaterializeNativeContinueMapTiming(state, cri))
		pendingErr := pendingCall()
		pendingOrder := orderDependent(pendingErr, pendingCall)
		stage("pending_groups", pendingErr)
		if pendingOrder {
			last := stages[len(stages)-1].(obj)
			last["err"], last["err_order_dependent"] = "map-order", true
		}
		stage("interactive", campaign.MaterializeNativeContinueInteractiveBoundary(state, cri))
		stage("handoff", campaign.ValidateNativeContinueBattleHandoff(cri, state, asset.scenario))
		result["stages"] = stages
		records, err := campaign.BuildNativeCurrentRuntimeRecords(state)
		hexRecords := make([]string, len(records))
		for i, r := range records {
			hexRecords[i] = hexs(r.Raw[:])
		}
		result["runtime_records"], result["runtime_err"] = hexRecords, errv(err)
		// 戰鬥中加入一名 JOIN 角色後的持續名冊（Go 測試的 TestBuildNativeCurrentPersistentRecords 形狀）。
		joinID := e.intn(34)
		if joined, err := e.join.MaterializePersistentUnit(joinID, battle.Unit{}, e.itemRows); err == nil {
			state.Units = append(state.Units, &joined)
			state.NativeRuntimeRecords = append(state.NativeRuntimeRecords, battle.NativeRuntimeRecordState{})
		}
		persistent, count, err := campaign.BuildNativeCurrentPersistentRecords(state, snapshot, e.join, e.itemTable(variant))
		all := make([]byte, 0, fdsave.RosterSize)
		for _, r := range persistent {
			all = append(all, r.Raw[:]...)
		}
		input["join_id"] = joinID
		result["persistent_records"], result["persistent_count"], result["persistent_err"] = nilIfErr(err, hexs(all)), nilIfErr(err, count), errv(err)
		e.add("continue", input, result)
	}
}

// ---- 結局 ----

func compositorSums(c *ending.IndexedCompositor) obj {
	return obj{"vga": digest(c.VGA), "palette": digest(c.Palette[:])}
}

func (e *exporter) genEndingLoaders() {
	timeline, err := ending.LoadTimeline("assets/endings/native_2bce5.json")
	if err != nil {
		panic(err)
	}
	phase, err := ending.LoadFinalePhase("assets/endings/native_2c405.json")
	if err != nil {
		panic(err)
	}
	montage, err := ending.LoadMontage("assets/endings/native_2c548.json")
	if err != nil {
		panic(err)
	}
	tail, err := ending.LoadMontageTail("assets/endings/native_2c194_tail.json")
	if err != nil {
		panic(err)
	}
	e.add("ending_loaders", obj{}, obj{"timeline": gj(timeline), "phase": gj(phase), "montage": gj(montage), "tail": gj(tail),
		"tail_plan": gj(mustPlan(tail.Plan())), "tail_visual": gj(mustVisual(tail.PlanVisualResources())),
		"fade": gj(mustFade(montage.PlanFigureFade(2))), "mirror": gj(mustMirror(montage.PlanMirrorFigureFade(0, 0))),
		"grid": gj(mustGrid(montage.PlanDialogueFrameGrid()))})
}

func mustPlan(v []ending.MontageTailEntry, err error) []ending.MontageTailEntry {
	if err != nil {
		panic(err)
	}
	return v
}

func mustVisual(v []ending.MontageTailVisualResources, err error) []ending.MontageTailVisualResources {
	if err != nil {
		panic(err)
	}
	return v
}

func mustFade(v []ending.FigureFadePass, err error) []ending.FigureFadePass {
	if err != nil {
		panic(err)
	}
	return v
}

func mustMirror(v []ending.MirrorFigureFadePass, err error) []ending.MirrorFigureFadePass {
	if err != nil {
		panic(err)
	}
	return v
}

func mustGrid(v []ending.DialogueFramePlacement, err error) []ending.DialogueFramePlacement {
	if err != nil {
		panic(err)
	}
	return v
}

// montageUnits 取真實存檔目前戰場的前四筆我方執行期紀錄（0x2c548 消費的 +6、+7、+8、+0x20）；
// 第一筆的 +6 清零以走 unit[+6]==0 的鏡像分支。
func (e *exporter) montageUnits() ([][]byte, []byte) {
	units := make([][]byte, 0, 4)
	groups := make([]byte, 0, 4)
	for i := 0; i < 4; i++ {
		raw := e.current.RuntimeRecords[i].Raw
		unit := append([]byte(nil), raw[:0x21]...)
		if i == 0 {
			unit[6] = 0
		}
		units = append(units, unit)
		groups = append(groups, unit[7])
	}
	return units, groups
}

func (e *exporter) genEnding() {
	timeline, _ := ending.LoadTimeline("assets/endings/native_2bce5.json")
	phase, _ := ending.LoadFinalePhase("assets/endings/native_2c405.json")
	montage, _ := ending.LoadMontage("assets/endings/native_2c548.json")
	tail, _ := ending.LoadMontageTail("assets/endings/native_2c194_tail.json")
	frames, err := fdother.DecodeResource(filepath.Join(e.flame, "FDOTHER.DAT"), timeline.Resource.Index)
	if err != nil {
		panic(err)
	}
	clip, err := afm.DecodeResource(filepath.Join(e.flame, "ANI.DAT"), 2)
	if err != nil {
		panic(err)
	}
	strings, err := fdtxt.LoadSeparatedResource(e.packPath("text"), 31)
	if err != nil {
		panic(err)
	}
	font, err := fdtxt.LoadSeparatedFont(e.packPath("fonts"))
	if err != nil {
		panic(err)
	}

	// 1. 終章前段 Player：固定步長序列，遇到文字閘門就記下兩種分支的頭像後繼續。
	c := ending.NewIndexedCompositor()
	player, err := ending.NewPlayer(*timeline, frames, clip, c)
	if err != nil {
		panic(err)
	}
	if err := player.EnableRecoveredPhase0(*phase, ending.Phase0Assets{Strings: strings, Font: font}); err != nil {
		panic(err)
	}
	pattern := []int{0, 1, 7, 16, 33, 100, 250, 3, 40}
	var steps []any
	var elapsed []int
	for k := 0; k < 4000; k++ {
		ms := pattern[k%len(pattern)]
		elapsed = append(elapsed, ms)
		state, err := player.Advance(ms)
		step := obj{"state": string(state), "err": errv(err), "segment": player.Segment, "wait": player.WaitMS}
		if player.Blocked != nil {
			step["blocked"] = []string{player.Blocked.Op, player.Blocked.Source}
		}
		for key, v := range compositorSums(c) {
			step[key] = v
		}
		resumed := false
		if state == ending.PlaybackBlocked {
			then, ok26 := player.BlockedDialogue(26)
			other, ok29 := player.BlockedDialogue(29)
			step["dialogue"] = []any{len(then), ok26, len(other), ok29}
			cue, ok := player.AudioCueAtBlockedBoundary()
			step["cue"] = []any{cue.Source, ok}
			resumed = player.ResumeBlockedDialogue()
			step["resumed"] = resumed
		}
		steps = append(steps, step)
		if err != nil || (state != ending.PlaybackRunning && !resumed) {
			break
		}
	}
	e.add("ending_player", obj{"elapsed": elapsed}, obj{"steps": steps, "rgba": digest(c.RGBA().Pix)})

	// 2. 0x2c548 隊伍蒙太奇：接在 Player 之後的同一個合成器（cmd/fd2/ending_preview.go 的順序）。
	units, groups := e.montageUnits()
	paths := ending.MontageArchivePaths{
		SurfaceRoot: e.packPath("surfaces"), ItemPanelRoot: e.packPath("ui"), AnimationRoot: e.packPath("animations"),
		PortraitRoot: e.packPath("portraits"), TextRoot: e.packPath("text"), FontRoot: e.packPath("fonts"),
	}
	assets, err := ending.LoadMontageCycleAssets(*montage, paths, units)
	if err != nil {
		panic(err)
	}
	for variant, skipAt := range []int{-1, 30} {
		cc := ending.NewIndexedCompositor()
		copy(cc.VGA, c.VGA)
		if err := cc.PresentANI(c.VGA, c.Baseline[:]); err != nil {
			panic(err)
		}
		copy(cc.Palette[:], c.Palette[:])
		cycle, err := ending.NewMontageCycle(*montage, assets, units, groups, cc)
		if err != nil {
			panic(err)
		}
		var randoms []int
		var cycleSteps []any
		portraitSteps := 0
		for k := 0; k < 6000 && !cycle.Ready(); k++ {
			random := 0
			if cycle.NeedsRandomByte() {
				random = e.intn(256)
			}
			randoms = append(randoms, random)
			err := cycle.Step(byte(random))
			observed := false
			if cycle.Phase == ending.MontagePhasePortrait {
				portraitSteps++
				if portraitSteps == skipAt {
					observed = cycle.ObserveInputChange()
				}
			}
			step := obj{"phase": string(cycle.Phase), "plan": cycle.PlanIndex, "ticks": cycle.DelayTicks, "ms": cycle.DelayMS,
				"err": errv(err), "observed": observed, "portrait": cycle.Portrait}
			for key, v := range compositorSums(cc) {
				step[key] = v
			}
			cycleSteps = append(cycleSteps, step)
			if err != nil {
				break
			}
		}
		e.add("montage_cycle", obj{"variant": variant, "skip_at": skipAt, "randoms": randoms, "units": hexList(units), "groups": hexs(groups)},
			obj{"steps": cycleSteps, "ready": cycle.Ready()})
	}

	// 3. 0x2c194 片尾：原版 FDOTHER 資源加分離素材的 20 組視覺交易。
	tailAssets, err := ending.LoadMontageTailAssetsArchive(*tail, filepath.Join(e.flame, "FDOTHER.DAT"))
	if err != nil {
		panic(err)
	}
	sets, err := ending.LoadMontageTailVisualSets(*tail, ending.MontageTailVisualPaths{
		SurfaceRoot: e.packPath("surfaces"), AnimationRoot: e.packPath("animations")})
	if err != nil {
		panic(err)
	}
	for _, source := range []string{"archive", "separated"} {
		ta := tailAssets
		if source == "separated" {
			ta, err = ending.LoadMontageTailAssets(*tail, ending.MontageTailAssetPaths{
				SurfaceRoot: e.packPath("surfaces"), PaletteRoot: e.packPath("palette"), AnimationRoot: e.packPath("animations")})
			if err != nil {
				panic(err)
			}
		}
		tc := ending.NewIndexedCompositor()
		tp, err := ending.NewMontageTailPlayer(*tail, ta, sets, tc)
		if err != nil {
			panic(err)
		}
		var tailSteps []any
		for k := 0; k < 8000 && !tp.Ready(); k++ {
			err := tp.Step()
			step := obj{"phase": string(tp.Phase), "segment": tp.Segment, "frame": tp.Frame, "inner": tp.Inner,
				"base": []int{tp.BaseFrame, tp.BaseInner}, "effect": tp.EffectStep, "sound": tp.SoundMarker,
				"ticks": tp.DelayTicks, "err": errv(err)}
			for key, v := range compositorSums(tc) {
				step[key] = v
			}
			tailSteps = append(tailSteps, step)
			if err != nil {
				break
			}
		}
		e.add("montage_tail", obj{"source": source}, obj{"steps": tailSteps, "ready": tp.Ready()})
	}

	// 4. 片尾的 0x1088d(0x1e) 載入基準：真實持續名冊與合成名冊。
	for n := 0; n < 12; n++ {
		count := int(e.current.Header.PersistentCount)
		records := append([]fdsave.PersistentRecord(nil), e.current.PersistentRecords[:count]...)
		if n > 0 {
			records = make([]fdsave.PersistentRecord, e.rangeInt(1, 32))
			for i := range records {
				records[i] = e.genRecord()
				records[i].Raw[7] = byte(e.intn(140))
			}
		}
		variant := e.pickItemVariant()
		baseline, err := ending.BuildMontageTailLoaderBaseline(*tail, records, ending.MontageTailLoaderPaths{
			FDFIELDRoot: e.packPath("fields/fdfield"), FDICONRoot: e.packPath("sprites/fdicon")}, e.itemTable(variant))
		var runtime []string
		for i := 0; i < baseline.RuntimeCount(); i++ {
			r, _ := baseline.RuntimeRecord(i)
			runtime = append(runtime, hexs(r[:]))
		}
		raw := make([]string, len(records))
		for i, r := range records {
			raw[i] = hexs(r.Raw[:])
		}
		e.add("tail_loader", obj{"records": raw, "items": variant}, obj{"err": errv(err), "runtime": runtime})
	}

	// 5. 0x2c7ed 頭像與五個文字欄位：分離素材的對話格網、DATO 頭像與兩張 FDTXT 表。
	for n := 0; n < 24; n++ {
		unit := append([]byte(nil), units[n%len(units)]...)
		if n >= 8 {
			unit[8] = byte(e.intn(32))
			unit[0x20] = byte(e.intn(29))
		}
		edi := []int{0, 1, 0xdb, 0xdc, 0x1b7}[e.intn(5)]
		frame := e.intn(5)
		frameOut, err := ending.ComposeMontagePortraitFrame(*montage, assets.Grid, unit, edi, frame,
			assets.Portraits[int(unit[7])], assets.Current, assets.Permanent, assets.Font)
		e.add("portrait_frame", obj{"unit": hexs(unit), "edi": edi, "frame": frame},
			obj{"err": errv(err), "sha": nilIfErr(err, digest(frameOut))})
	}

	// 6. 終章對話框：sub_1956B 底框、逐字發布頁面與張嘴頭像（開收框的職業清單轉場屬 B 組，另由 B 的黃金涵蓋）。
	entries, err := fdother.LoadSeparatedItemPanelEntries(e.packPath("ui"))
	if err != nil {
		panic(err)
	}
	maxCell := 0
	for index := range entries.Raw {
		if index > maxCell {
			maxCell = index
		}
	}
	cells := make([]fdother.RawCell, maxCell+1)
	for index, cell := range entries.Raw {
		cells[index] = cell
	}
	glyphs := loadGlyphs("../docs/data/glyph_map.json")
	background := make([]byte, ending.Bytes)
	for k := range background {
		background[k] = byte(k * 7)
	}
	for _, segment := range []int{9, 13} {
		blocks := append(append([]ending.DialogueBlock(nil), timeline.Segments[segment].ThenDialogue...), timeline.Segments[segment].ElseDialogue...)
		for _, block := range blocks {
			for u, utterance := range block.NativeUtterances {
				portraits, err := dato.LoadSeparatedResource(e.packPath("portraits"), utterance.Operand)
				if err != nil {
					e.add("dialogue", obj{"portrait": utterance.Operand}, obj{"err": "portrait unavailable"})
					continue
				}
				base, err := ending.ComposeNativeDialogueBase(background, cells, portraits[0])
				result := obj{"base_err": errv(err), "base": nilIfErr(err, digest(base))}
				used := map[string]int{}
				var pageSums []any
				if err == nil {
					for page := range utterance.Pages {
						for _, row := range utterance.Pages[page] {
							for _, r := range row {
								if index, ok := glyphs[string(r)]; ok {
									used[string(r)] = index
								}
							}
						}
						framesOut, err := ending.ComposeNativeDialogueProgressiveFrames(base, font, used, utterance.Pages, page)
						sums := make([]string, len(framesOut))
						for i, f := range framesOut {
							sums[i] = digest(f)
						}
						pageSums = append(pageSums, obj{"err": errv(err), "frames": sums})
					}
					mouth, err := ending.ComposeNativeDialogueMouthFrame(base, portraits[3])
					result["mouth_err"], result["mouth"] = errv(err), nilIfErr(err, digest(mouth))
				}
				result["pages"] = pageSums
				e.add("dialogue", obj{"portrait": utterance.Operand, "pages": utterance.Pages, "glyphs": used,
					"background": "k*7", "utterance": u}, result)
			}
		}
	}

	// 7. 0x2c405 第 0 段：獨立的 500 次捲動，基準調色盤由固定種子產生。
	for n := 0; n < 3; n++ {
		var baseline [768]byte
		for i := range baseline {
			baseline[i] = byte(e.intn(64))
		}
		pc := ending.NewIndexedCompositor()
		p0, err := ending.NewPhase0PlayerFromAssets(*phase, ending.Phase0Assets{Strings: strings, Font: font, Baseline: baseline}, pc)
		if err != nil {
			panic(err)
		}
		var steps []any
		var elapsed []int
		for k := 0; k < 700; k++ {
			ms := []int{0, 1, 2, 5, 1}[k%5]
			elapsed = append(elapsed, ms)
			done, err := p0.Advance(ms)
			step := obj{"done": done, "err": errv(err)}
			for key, v := range compositorSums(pc) {
				step[key] = v
			}
			steps = append(steps, step)
			if done || err != nil {
				break
			}
		}
		e.add("phase0", obj{"baseline": hexs(baseline[:]), "elapsed": elapsed}, obj{"steps": steps})
	}
}

func hexList(b [][]byte) []string {
	out := make([]string, len(b))
	for i, v := range b {
		out[i] = hexs(v)
	}
	return out
}

func loadGlyphs(path string) map[string]int {
	raw, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var encoded map[string]json.RawMessage
	if err := json.Unmarshal(raw, &encoded); err != nil {
		panic(err)
	}
	out := map[string]int{}
	keys := make([]string, 0, len(encoded))
	for key := range encoded {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		index, err := strconv.Atoi(key)
		if err != nil {
			continue
		}
		var text string
		if err := json.Unmarshal(encoded[key], &text); err != nil || len([]rune(text)) != 1 {
			continue
		}
		if prior, ok := out[text]; !ok || index < prior {
			out[text] = index
		}
	}
	return out
}

func main() {
	out := flag.String("out", "", "輸出 JSON")
	flame := flag.String("flame2", "", "原版 FLAME2 目錄")
	pack := flag.String("pack", "", "分離素材包")
	seed := flag.Uint64("seed", 0x43414d50, "PCG 種子")
	flag.Parse()
	if *out == "" || *flame == "" || *pack == "" {
		fmt.Fprintln(os.Stderr, "需要 -out、-flame2、-pack")
		os.Exit(2)
	}
	e := &exporter{rng: rand.New(rand.NewPCG(*seed, 0x43414d43)), flame: *flame, pack: *pack}
	for _, p := range []string{
		"assets/data/native_character_catalog.json", "assets/data/native_intermission_gate.json",
		"assets/data/native_join_constructor.json", "assets/data/native_join_base_units.json",
		"assets/data/native_item_effect_rows.json", "assets/maps/map0/map0_units.json",
		"assets/maps/map25/map25_units.json", "assets/maps/map17/map17_units.json", "assets/maps/map5/map5_units.json",
		"assets/scenarios/ch01.json", "assets/scenarios/ch26.json",
		"assets/endings/native_2bce5.json", "assets/endings/native_2c405.json",
		"assets/endings/native_2c548.json", "assets/endings/native_2c194_tail.json",
	} {
		e.record("remake/"+p, p)
	}
	e.record("docs/data/glyph_map.json", "../docs/data/glyph_map.json")
	for _, name := range []string{"FD2.SAV", "FDOTHER.DAT", "ANI.DAT"} {
		e.record("flame2/"+name, filepath.Join(*flame, name))
	}
	for _, rel := range []string{"surfaces", "ui/fdother_005_item_panel", "portraits", "text/FDTXT_000", "text/FDTXT_031",
		"fonts", "palette", "fields/fdfield/selector_30", "sprites/fdicon"} {
		e.recordTree("pack", *pack, rel)
	}
	// 動畫目錄很大：只記用到的 FIGANI 與 FDOTHER_058。
	units := []int{}
	var err error
	stored, err := os.ReadFile(filepath.Join(*flame, "FD2.SAV"))
	if err != nil {
		panic(err)
	}
	if e.plain, err = fdsave.Decode(stored); err != nil {
		panic(err)
	}
	if e.current, err = fdsave.InspectCurrentSnapshot(e.plain); err != nil {
		panic(err)
	}
	for i := 0; i < 4; i++ {
		units = append(units, int(e.current.RuntimeRecords[i].Raw[7]))
	}
	tail, err := ending.LoadMontageTail("assets/endings/native_2c194_tail.json")
	if err != nil {
		panic(err)
	}
	animations := map[string]bool{"FDOTHER_058": true}
	for _, g := range units {
		animations[fmt.Sprintf("FIGANI_%03d", g*3)] = true
		animations[fmt.Sprintf("FIGANI_%03d", g*3+1)] = true
	}
	visual, err := tail.PlanVisualResources()
	if err != nil {
		panic(err)
	}
	for _, v := range visual {
		for _, r := range []int{v.Record0FIGANIBase, v.Record0FIGANIAux, v.Record1FIGANIBase, v.Record1FIGANIAux} {
			animations[fmt.Sprintf("FIGANI_%03d", r)] = true
		}
	}
	names := make([]string, 0, len(animations))
	for name := range animations {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		e.recordTree("pack", *pack, "animations/"+name)
	}
	if e.catalog, err = campaign.LoadNativeCharacterCatalog("assets/data/native_character_catalog.json"); err != nil {
		panic(err)
	}
	if e.gate, err = campaign.LoadNativeIntermissionGateTable("assets/data/native_intermission_gate.json"); err != nil {
		panic(err)
	}
	if e.join, err = campaign.LoadNativeJoinConstructorTable("assets/data/native_join_constructor.json"); err != nil {
		panic(err)
	}
	if e.itemRows, err = battle.LoadNativeItemEffectRowPrefix("assets/data/native_item_effect_rows.json"); err != nil {
		panic(err)
	}
	e.genRestorePlans()
	e.genSlotSaves()
	e.genJoin()
	e.genPartyRecords()
	e.genContinue()
	e.genEndingLoaders()
	e.genEnding()
	sort.Slice(e.files, func(i, j int) bool { return e.files[i]["path"].(string) < e.files[j]["path"].(string) })
	doc := obj{"schema": "fd2-golden-campaign-c/1", "seed": *seed, "files": e.files, "cases": e.cases}
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
	opNames := []string{}
	for name := range ops {
		opNames = append(opNames, name)
	}
	sort.Strings(opNames)
	for _, name := range opNames {
		fmt.Printf("%s=%d ", name, ops[name])
	}
	fmt.Printf("total=%d\n", len(e.cases))
}
