// Package battle — 炎龍騎士團2 重製的戰棋核心資料模型(M1)。
//
// 設計:遊戲差異全在資料(units.json 由 tools/export_units.py 從原版產生),
// 引擎只認穩定的 JSON。Unit 用 HP/OnField/Acted 投影原版單位狀態。
package battle

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wicanr2/fd2_re/remake/internal/fdicon"
	"github.com/wicanr2/fd2_re/remake/internal/fdother"
)

const nativeTurnEventCatalogSHA256 = "127c894c523f74905048524d02fdaf142c24610fdb3534694d1ccdfef8dfcd34"

// Camp 陣營。
type Camp int

const (
	Own   Camp = iota // 我方(玩家)
	Ally              // 友軍 NPC
	Enemy             // 敵方
)

func (c Camp) String() string {
	switch c {
	case Own:
		return "OWN"
	case Ally:
		return "ALLY"
	default:
		return "ENEMY"
	}
}

// Unit 戰場單位。數值來自 EXE 表(doc 03); raw byte fields are optional
// provenance and normalized fields remain engine projections.
type Unit struct {
	Camp      Camp
	Name      string // 角色名(characters.json;敵方多為職業名)
	ClsName   string // 職業名(中文,M2 TTF 才顯示)
	ClassID   int    // 原版職業 table index；商店裝備相容以此對 class_equip_types 判定
	Lv        int
	HP, MaxHP int
	MP, MaxMP int
	AP, DP    int
	HIT, EV   int // 命中/閃避基礎值(doc02 §2;doc03:EXE 內為「衍生值」非表格原始欄位,
	// 敵/友單位 10B 表無此欄,export_units.py 暫用固定近似值,見該檔頭註解)
	CritPct  int // 暴擊率(doc03 職業暴擊率表 0x5219B,resist_crit.json,依 class 已驗證吻合 doc02 §7.2)
	MV       int // 移動力
	AtkMin   int // 近戰攻擊距離下限(曼哈頓距離;0 視為預設 1,doc32 weapon_range.json 依武器 type 決定)
	AtkMax   int // 近戰攻擊距離上限(0 視為預設 1;例:騎士槍type3=2,doc32)
	Portrait int
	Fig      int // 地圖 FDICON selector approximation; native source is unit+2.
	// MapSelectorSlot is native runtime unit+2 only when HasMapSelectorSlot is
	// true. It is a process-global FDICON cache slot, never a character identity.
	MapSelectorSlot    int
	HasMapSelectorSlot bool
	// MapSelectorKey is the raw byte passed to 0x11019 before it returns the
	// cache slot above. It is optional because legacy map JSON does not carry
	// its native source provenance. It must never be inferred from Fig.
	MapSelectorKey    int
	HasMapSelectorKey bool
	// NativeMapPresentation is the materialized battle-local runtime
	// +0/+1/+3/+4 subset. It is valid only after the native selector
	// construction path succeeds; normalized X/Y/Dir/OffX/OffY are not aliases.
	NativeMapPresentation    NativeMapPresentationState
	HasNativeMapPresentation bool
	// BattleFig is the separately sourced native unit+7 selector for FIGANI.
	// FDFIELD roster b1 supplies it; missing older JSON keeps the Fig fallback.
	BattleFig    int
	HasBattleFig bool
	// NativeIdentity is the persistent-record +0x08 key consumed by native
	// post-handler 0x11506. It is optional provenance and must never be inferred
	// from Fig, BattleFig, portrait, or map-selector fields.
	NativeIdentity    int
	HasNativeIdentity bool
	// NativeRecordByte8 preserves runtime/persistent record +8 without
	// assigning a universal identity meaning. Player persistent records use
	// this byte as NativeIdentity, while scripted runtime construction copies
	// its visual selector into both +7 and +8.
	NativeRecordByte8    byte
	HasNativeRecordByte8 bool
	// NativePositionRecord preserves the six-byte FDFIELD 3N+2 record paired
	// with a scripted constructor row. The constructor reads XWord/YWord low
	// bytes and may relocate them through its nearest-free-cell search.
	NativePositionRecord    NativePositionRecord
	HasNativePositionRecord bool
	// NativeRecordRace/Class are the independently proven raw bytes at
	// persistent/runtime +0x1f/+0x20. They remain separate from normalized
	// class labels and from the immutable constructor-table provenance because
	// class change rewrites only +0x20.
	NativeRecordRace     byte
	HasNativeRecordRace  bool
	NativeRecordClass    byte
	HasNativeRecordClass bool
	// NativeConstructor preserves the proven EXE static-table provenance for
	// unit+0x1f/+0x20 without assigning any gameplay meaning to raw bytes.
	NativeConstructor *NativeConstructorTable `json:"native_constructor,omitempty"`
	X, Y              int
	Acted             bool  // engine projection; native bit7 is raw provenance only
	Group             int   // 出場波次(原版 FDFIELD b21;事件按 group 放出,doc 25/29)
	OnField           bool  // 是否已登場(事件進場機制:false=待命,尚未出現在戰場,doc 25)
	Spells            []int // normalized/editable spell IDs; not a raw unit+0x22 bitfield
	// NativeCommandMask is the runtime 40-bit command inventory enumerated by
	// 0x1c269.  FDFIELD b13..b16 initializes bytes 0..3; byte 4 begins zero and
	// can be OR-mutated by 0x1d7fb.  It is deliberately separate from Spells:
	// command ID effects and target contracts are not inferred from this mask.
	NativeCommandMask [5]byte `json:"native_command_mask,omitempty"`
	// NativeTransient is the exact six-byte runtime interval unit+0x22..+0x27.
	// It is distinct from the legacy normalized Buff*/Poison*/Seal*/Paralyze*
	// approximation below: native command IDs 17..27 read/write individual raw
	// bytes, and 0x1A866 decrements them separately once per camp phase.
	// No gameplay name is inferred for an index from this storage alone.
	NativeTransient [6]byte `json:"native_transient"`
	// NativeRecordByte5/6 preserve the two raw gate bytes consumed by
	// 0x1A866. Their gameplay meaning is intentionally unknown; neither is
	// inferred from Acted, OnField, Alive, or Camp.
	NativeRecordByte5    byte `json:"native_record_byte5,omitempty"`
	HasNativeRecordByte5 bool `json:"has_native_record_byte5,omitempty"`
	NativeRecordByte6    byte `json:"native_record_byte6,omitempty"`
	HasNativeRecordByte6 bool `json:"has_native_record_byte6,omitempty"`
	// FDFIELD b17..b19 are copied verbatim by 0x10fb6..0x10fc5 to runtime
	// +0x34..+0x36. Byte34's low nibble dispatches 0x13a9f, while its high
	// bits have separate consumers; keep the complete bytes and provenance.
	NativeRecordByte34    byte `json:"native_record_byte34,omitempty"`
	HasNativeRecordByte34 bool `json:"has_native_record_byte34,omitempty"`
	NativeRecordByte35    byte `json:"native_record_byte35,omitempty"`
	HasNativeRecordByte35 bool `json:"has_native_record_byte35,omitempty"`
	NativeRecordByte36    byte `json:"native_record_byte36,omitempty"`
	HasNativeRecordByte36 bool `json:"has_native_record_byte36,omitempty"`
	// NativeRecordByte3D is copied from FDFIELD b2 by 0x10fc8. It is distinct
	// from constructor-table race at runtime +0x1f.
	NativeRecordByte3D    byte
	HasNativeRecordByte3D bool
	// NativeRecordDeathEffect is exact runtime +0x31..+0x33 from b22..b24.
	NativeRecordDeathEffect    [3]byte
	HasNativeRecordDeathEffect bool
	// These source bytes are not read by 0x10c50; preserve them without names.
	NativeFDFIELDSourceByte3  byte
	NativeFDFIELDSourceByte20 byte
	NativeFDFIELDSourceByte25 byte
	// NativeRecordWord42 is the optional raw u16 at runtime record +0x42.
	// ch15_post compares this word directly. When present on a loaded scripted
	// roster it is also the proven constructor source for initial HP/MaxHP.
	NativeRecordWord42    uint16 `json:"native_record_word42,omitempty"`
	HasNativeRecordWord42 bool   `json:"has_native_record_word42,omitempty"`
	// NativeRecordWord46 is the optional constructor-produced max-MP word at
	// runtime +0x46. It is not inferred from normalized MaxMP.
	NativeRecordWord46    uint16 `json:"native_record_word46,omitempty"`
	HasNativeRecordWord46 bool   `json:"has_native_record_word46,omitempty"`
	Inventory             []int  // 角色物品欄 item IDs；原版 unit+0x0a 起 8×2B
	Equipped              []bool // 與 Inventory 對齊；true 表示該欄位目前已裝備
	InventorySlots        []int  // 原始 8 個 source bytes；0xff 保留空槽位置
	// NativeInventoryFlags is the constructor's raw eight-cell flag view. It is
	// present only when InventorySlots came from the proven source layout.
	NativeInventoryFlags []int
	// Base* are the persistent pre-remake equipment values. Existing scenario
	// data stores effective values, so EquipmentBaseSet is true for those
	// records and newly purchased equipment is added without double counting.
	BaseAP, BaseDP, BaseHIT, BaseEV, BaseMV int
	BaseAtkMin, BaseAtkMax                  int
	EquipmentBaseSet                        bool
	DeathEffect                             *DeathEffect // FDFIELD b22 + b23..24；0=item、1=gold，2/3 特殊效果先原值保留
	DeathReward                             *DeathEffect // 可執行死亡獎勵；type2 已知 handler 由 exporter lower 成 item/gold
	Dir                                     int          // 朝向:0下 1左 2上 3右(原版 Z2,FDICON 方向幀)
	OffX                                    float64      // 行軍/移動的像素位移(顯示用;0=正在格上)
	OffY                                    float64      // 進場時從邊緣滑入,漸減到 0

	// ---- 輔助法術暫時狀態(doc02 §6.4;施放邏輯見 magic.go CastArea/applySpell)----
	BuffAPPct int // 魔刃術:AP 加成百分比
	BuffDPPct int // 魔鎧術:DP 加成百分比
	BuffHit   int // 風行術:HIT 加成
	BuffEV    int // 風行術:EV 加成
	BuffTurns int // 上述加成共用剩餘回合數(原版三招各自可疊加回合,重製簡化成單一計時器)

	Sealed    bool // 封咒術:禁止施法
	SealTurns int

	Poisoned    bool // 毒擊術:每回合扣 MaxHP 的 10%(doc02 §6.4)
	PoisonTurns int

	Paralyzed     bool // 麻痺術:無法行動(是否擋下行動由呼叫端 UI/AI 檢查此欄位)
	ParalyzeTurns int

	// ---- 經驗值/升級(doc02 §4.5/§4.6;doc03 0x43;實作見 growth.go)----
	DX int // typed +0x3e projection；0x1145a/0x2efb7 以它為 HIT/EV 的共用 base；不等同直接 raw dump。
	// FDFIELD export 的 generic enemy HIT/EV 仍可能是近似值；evidence-backed
	// player scenarios必須顯式保存DX，shop preview/class growth不可由derived值猜回。
	// 尚未實際影響命中/閃避。
	Exp float64 // 目前經驗值(滿 100 升級,doc03 0x43「EX 經驗」);用 float64 累加,避免
	// 攻擊/法術公式算出的小數經驗(如 40/施法者等級)逐次相加時被提早捨去。
	ExpPerLevel int // 本單位每級可給出的經驗值(doc02 §4.5「守方每級經驗」;來源 EXE
	// 敵/友單位表 EX 欄,docs/data/exe_tables/unit.json,由 export_units.py 依 (race,cls)
	// 帶入 units.json 的 "ex" 欄;舊版(尚未重新匯出的)units.json 無此欄則為 0,
	// 該次攻擊經驗值算出 0,見 growth.go AttackExp 註解)
}

// NativeConstructorTable is an editable, raw-only view of the constructor's
// portrait-selected EXE tables. Record is 10 bytes for high_class and 24
// bytes for lower_class; AuxRecord is the paired 11-byte 0x620a1 record for
// the lower branch. It is never used as a portrait/class substitute.
type NativeConstructorTable struct {
	Branch    string `json:"branch"`
	Index     int    `json:"index"`
	Record    []byte `json:"record"`
	AuxRecord []byte `json:"aux_record,omitempty"`
}

func (t *NativeConstructorTable) validate() error {
	if t == nil {
		return nil
	}
	switch t.Branch {
	case "high_class":
		if t.Index < 0 || t.Index >= 68 || len(t.Record) != 10 || len(t.AuxRecord) != 0 {
			return fmt.Errorf("invalid high_class table record")
		}
	case "lower_class":
		if t.Index < 0 || t.Index >= 32 || len(t.Record) != 24 || len(t.AuxRecord) != 11 {
			return fmt.Errorf("invalid lower_class table records")
		}
	default:
		return fmt.Errorf("unknown constructor table branch %q", t.Branch)
	}
	return nil
}

// MaterializeNativeMapSelectorSlots applies 0x11019's first-seen cache rule
// to one already-proven construction order. Callers must supply exactly the
// native source order within one battle construction session (player persistent
// roster followed by scripted FDFIELD spawn groups); this function deliberately does not choose
// that order or fall back to Fig/Portrait. It makes no partial mutation when
// a key is absent or invalid.
func MaterializeNativeMapSelectorSlots(units []*Unit, cache *fdicon.NativeSelectorCache) error {
	if cache == nil {
		return fmt.Errorf("native map selector: nil cache")
	}
	for i, u := range units {
		if u == nil || !u.HasMapSelectorKey {
			return fmt.Errorf("native map selector: unit %d has no explicit raw key", i)
		}
		if u.MapSelectorKey < 0 || u.MapSelectorKey > 0xff {
			return fmt.Errorf("native map selector: unit %d has invalid raw key", i)
		}
		if u.X < 0 || u.X > 0xff || u.Y < 0 || u.Y > 0xff {
			return fmt.Errorf("native map selector: unit %d coordinate outside byte range", i)
		}
	}
	slots := make([]int, len(units))
	for i, u := range units {
		slot, err := cache.SlotFor(u.MapSelectorKey)
		if err != nil {
			return fmt.Errorf("native map selector: unit %d: %w", i, err)
		}
		slots[i] = slot
	}
	for i, u := range units {
		u.MapSelectorSlot, u.HasMapSelectorSlot = slots[i], true
		if err := u.MaterializeNativeMapPresentation(); err != nil {
			return fmt.Errorf("native map selector: unit %d presentation: %w", i, err)
		}
	}
	return nil
}

// initialEquipmentFlags mirrors the original spawn constructor: its first
// two source inventory bytes are placed with flag 0x40; later bytes start
// held/unequipped. The editable exports omit 0xff empty bytes, so the
// compacted list is the available representation here.
func initialEquipmentFlags(n int) []bool {
	flags := make([]bool, n)
	for i := 0; i < n && i < 2; i++ {
		flags[i] = true
	}
	return flags
}

func materializeInventory(source, compact []int) ([]int, []bool, []int) {
	if len(source) != 8 {
		return append([]int(nil), compact...), initialEquipmentFlags(len(compact)), append([]int(nil), compact...)
	}
	runtime := make([]int, 8)
	flags := make([]bool, 8)
	for i := range runtime {
		runtime[i] = 0xff
	}
	if source[0] != 0xff {
		runtime[0], flags[0] = source[0], true
		if source[1] != 0xff {
			runtime[1], flags[1] = source[1], true
		}
	} else if source[1] != 0xff {
		runtime[0], flags[0] = source[1], true
	}
	for i := 2; i < 8; i++ {
		runtime[i] = source[i]
	}
	ids := make([]int, 0, 8)
	usedFlags := make([]bool, 0, 8)
	for i, id := range runtime {
		if id != 0xff {
			ids = append(ids, id)
			usedFlags = append(usedFlags, flags[i])
		}
	}
	return ids, usedFlags, runtime
}

func (u *Unit) normalizeInventorySlots() {
	if len(u.InventorySlots) == 8 {
		return
	}
	u.InventorySlots = make([]int, 8)
	for i := range u.InventorySlots {
		u.InventorySlots[i] = 0xff
	}
	for i, id := range u.Inventory {
		if i >= 8 {
			break
		}
		u.InventorySlots[i] = id
	}
}

// AddInventoryItem preserves the original fixed eight slot layout while
// keeping the compact Inventory view used by editable scripts.
func (u *Unit) AddInventoryItem(id int, equipped bool) bool {
	if u == nil || len(u.Inventory) >= 8 {
		return false
	}
	u.normalizeInventorySlots()
	slot := -1
	for i, held := range u.InventorySlots {
		if held == 0xff {
			slot = i
			break
		}
	}
	if slot < 0 {
		return false
	}
	u.InventorySlots[slot] = id
	if len(u.NativeInventoryFlags) == nativeInventoryCells {
		u.NativeInventoryFlags[slot] = 0
	}
	u.Inventory = append(u.Inventory, id)
	u.Equipped = append(u.Equipped, equipped)
	return true
}

func (u *Unit) RemoveInventoryIndex(index int) bool {
	if u == nil || index < 0 || index >= len(u.Inventory) {
		return false
	}
	u.normalizeInventorySlots()
	seen, slot := 0, -1
	for i, id := range u.InventorySlots {
		if id != 0xff {
			if seen == index {
				slot = i
				break
			}
			seen++
		}
	}
	if slot >= 0 {
		u.InventorySlots[slot] = 0xff
		if len(u.NativeInventoryFlags) == nativeInventoryCells {
			u.NativeInventoryFlags[slot] = 0x80
		}
	}
	u.Inventory = append(u.Inventory[:index], u.Inventory[index+1:]...)
	if index < len(u.Equipped) {
		u.Equipped = append(u.Equipped[:index], u.Equipped[index+1:]...)
	}
	return true
}

// Alive is the remake's normalized HP projection. It is intentionally not a
// native byte+5 decoder; callers with raw provenance must use the raw field.
func (u *Unit) Alive() bool { return u.HP > 0 }

// ApplyHPDamage updates the engine HP projection and, when raw provenance is
// present, mirrors the native death writer's byte+5 bit0.  It intentionally
// does not fabricate raw provenance for legacy/normalized units.
func (u *Unit) ApplyHPDamage(damage int) {
	if u == nil || damage <= 0 {
		return
	}
	u.HP -= damage
	if u.HP < 0 {
		u.HP = 0
	}
	if u.HP == 0 && u.HasNativeRecordByte5 {
		u.NativeRecordByte5 = 1
	}
}

// RestoreNativeHP mirrors the proven revive writer: restore current HP and
// clear only raw byte+5 bit0 when the unit carries native provenance.
func (u *Unit) RestoreNativeHP() {
	if u == nil {
		return
	}
	u.HP = u.MaxHP
	if u.HasNativeRecordByte5 {
		u.NativeRecordByte5 &^= 1
	}
}

// EffectiveAP/EffectiveDP 套用魔刃術/魔鎧術暫時加成後的攻防值。地形% 修正另外在
// combat.go AttackWithRNG 套用(取決於單位當下座標,不是固定加成,不適合放在這裡)。
func (u *Unit) EffectiveAP() int { return u.AP + u.AP*u.BuffAPPct/100 }
func (u *Unit) EffectiveDP() int { return u.DP + u.DP*u.BuffDPPct/100 }

// EffectiveHIT/EffectiveEV 套用風行術(HIT+15,EV+15)暫時加成後的命中/閃避值(doc02 §6.4)。
func (u *Unit) EffectiveHIT() int { return u.HIT + u.BuffHit }
func (u *Unit) EffectiveEV() int  { return u.EV + u.BuffEV }

// TickStatus 回合結束時呼叫一次:遞減暫時狀態剩餘回合、套用毒擊持續傷害、到期清除。
// (doc02 §6.4:毒擊每回合 -10% HP;各項加成/異常 2–4 回合到期消失。)
func (u *Unit) TickStatus() {
	if u.BuffTurns > 0 {
		u.BuffTurns--
		if u.BuffTurns == 0 {
			u.BuffAPPct, u.BuffDPPct, u.BuffHit, u.BuffEV = 0, 0, 0, 0
		}
	}
	if u.SealTurns > 0 {
		u.SealTurns--
		if u.SealTurns == 0 {
			u.Sealed = false
		}
	}
	if u.Poisoned {
		dmg := u.MaxHP / 10
		if dmg < 1 {
			dmg = 1
		}
		u.HP -= dmg
		if u.HP < 0 {
			u.HP = 0
		}
	}
	if u.PoisonTurns > 0 {
		u.PoisonTurns--
		if u.PoisonTurns == 0 {
			u.Poisoned = false
		}
	}
	if u.ParalyzeTurns > 0 {
		u.ParalyzeTurns--
		if u.ParalyzeTurns == 0 {
			u.Paralyzed = false
		}
	}
}

// State 一場戰鬥的狀態。
type State struct {
	W, H  int
	Units []*Unit
	// NativeMapSelectorCache is the single process-global raw-key cache accumulated
	// in native construction order. It is intentionally populated only through
	// AppendNativeMapSelectorBatch; legacy callers may keep using Units without
	// claiming an indexed-native selector state.
	NativeMapSelectorCache *fdicon.NativeSelectorCache
	// NativeMapSelectorError records a rejected native construction batch.  The
	// battle continues with legacy Fig rendering, but NativeMapSpriteKey remains
	// fail-closed for the entire state rather than mixing native and guessed
	// selectors after a malformed editable source.
	NativeMapSelectorError error
	// NativeMapCycleState is the battle-session ownership of native globals
	// 0x53c0b/0x53c07/0x53c0f. It becomes valid only when the first native
	// selector construction batch succeeds.
	NativeMapCycleState    fdicon.NativeMapSpriteCycleState
	HasNativeMapCycleState bool
	// NativeTerrainPhaseState owns 0x11eee's independent 20-phase terrain
	// selector and last BIOS tick for this battle session.
	NativeTerrainPhaseState    fdother.NativeTerrainPhaseState
	HasNativeTerrainPhaseState bool
	// NativeTerrainFlipState and NativeUnitPixelShiftState are the independent
	// 0x53a40/0x53a00 and 0x53a04/0x53a08 BIOS-word latches.
	NativeTerrainFlipState        fdicon.NativeBinaryTickState
	NativeUnitPixelShiftState     fdicon.NativeBinaryTickState
	HasNativeMapBinaryTimingState bool
	// NativeMapHUDState owns the two raw 0x1acf3 display bytes and persistent
	// anchor. Unlike timing globals, it is not fabricated by selector
	// construction because gate A can be restored from native save state.
	NativeMapHUDState     NativeMapHUDRuntimeState
	HasNativeMapHUDState  bool
	NativeMapViewState    NativeMapViewState
	HasNativeMapViewState bool
	// NativeMapRangeMode is raw [0x51a83]. Despite the retained field name it
	// is an overlay/selection selector, not a bounded GUI range enum:
	// 0x122dc draws only 1..5 and mutates the field for 6, while 0x115b6 still
	// consumes values above 6 in target validation. Battle setup briefly uses
	// zero for its opening frame at 0x10483, then returns with interactive
	// selector one at 0x1060c.
	NativeMapRangeMode         int
	HasNativeMapRangeModeState bool
	// Roster is the unmaterialized FDFIELD source used by scenarios which
	// preserve the original constructor semantics. Units is then the canonical
	// runtime array: party/initial groups are appended in event order, and later
	// SPAWN calls append their group without reserving slots ahead of time.
	Roster        []*Unit
	PendingGroups map[int]bool
	// nativeFutureItemRows is the immutable, explicitly bounded 0x4e56c row
	// prefix used by 0x10c50→0x1b750. It is bound by the application layer and
	// never inferred from normalized item statistics.
	nativeFutureItemRows []byte
	OwnDeploy            []Cell // 我方可部署格
	Turn                 int    // 回合數(無上限,doc 27;只由劇本事件限制)
	// NativeRoundCounter preserves executable global [0x53bef], incremented at
	// the native turn-advance boundary (0x1a5b9), apart from normalized Turn.
	NativeRoundCounter          int             `json:"native_round_counter,omitempty"`
	Flags                       map[string]bool // 事件旗標(跨事件/跨關劇情狀態,doc 29)
	NativeEventState            [0x20]byte      // raw [0x53ad5] battle-local state table; unnamed indices
	Cost                        []int           // per-tile 移動成本(len==W*H;index=y*W+x;nil=尚無地形資料,MoveCost 全回 1)
	NativeCompositionEventBytes []byte          // immutable FDFIELD composition cell +2 source; each caller rebuilds its own mutable low5/live flags
	// NativeMapEventGrid is the mutable 0x53a51-shaped map buffer used by the
	// native AI/event helpers.  It retains the four-byte header followed by
	// one four-byte FDFIELD tile/event word per cell; it is never substituted
	// for NativeCompositionEventBytes when a caller rebuilds a fresh grid.
	NativeMapEventGrid    []byte
	HasNativeMapEventGrid bool
	// NativeFieldControlRaw is the exact live current-save image rooted at
	// [0x53a55]. It is distinct from composition and from the original chapter
	// resource because battle handlers rewrite turn/chest bytes.
	NativeFieldControlRaw   []byte
	NativeTurnEventControls [16]NativeTurnEventControl
	// HasNativeTurnEventControlState 只有從精確 map provenance 或 CONTINUE
	// snapshot 載入全部16筆 raw row 時為真；不代表每筆休眠列都已有 runtime
	// scenario consumer。
	HasNativeTurnEventControlState bool
	NativeChestControls            [16]NativeChestControl
	NativeFieldUnitControls        []NativeFieldUnitControl
	HasNativeFieldControlState     bool
	// NativeRuntimeRecords preserves the exact saved current-unit array and
	// CONTINUE-rebuilt selector slots before a typed Unit projection exists.
	// Units remains the normalized/gameplay array and is never guessed from
	// these bytes by State itself.
	NativeRuntimeRecords []NativeRuntimeRecordState
	// HasNativeRuntimeUnitProjection is set only after the complete saved
	// runtime record array has been atomically projected to Units in original
	// record order. It does not imply that timing or the battle driver is ready.
	HasNativeRuntimeUnitProjection bool
	// HasNativePendingGroupBinding 表示 CONTINUE 已把仍待執行、且 live
	// turn/event 列與可編輯 scenario 完全相符的 FDFIELD rows 深複製到
	// Roster。它不代表動態改寫排程或 Game controller handoff 已完成。
	HasNativePendingGroupBinding bool
	NativeFieldEventSlots        []int                  // row-major -1/0..15；0x13a44 的 1-based low5 已正規化
	NativeFieldEvents            []NativeFieldEvent     // FDFIELD control 16×2 raw event-id/selector table
	NativeFieldEventRules        []NativeFieldEventRule // 已由 handler 閉合、仍保留 selector timing 的 editable rules
	NativeTileBlitModes          []byte                 // live FDFIELD entry byte+3; exact export admits it, then 0x4dbfc/0x14818 own mutation
	NativeTerrainControl         []byte                 // raw FDSHAP four-byte terrain records; nil unless exact renderer export exists
	NativeTerrainMoveCodes       []byte                 // FDSHAP control byte+1 selected by each FDFIELD tile; nil unless the complete exact export validates
	// nativeMovementCostRows is the detached 0x4e555 table used by the
	// original AI/path helpers. It is bound from the versioned asset export;
	// normalized Cost must never be substituted for this raw table.
	nativeMovementCostRows   [][]byte
	SpellBook                []Spell                     // scenario-injected spell table; AI command mapping remains data-only
	NativeCommandBook        []NativeCommandRecord       // verified raw IDs 0..35; distinct from normalized SpellBook
	NativeCommandResistances map[int]int                 // verified class raw multiplier; nil means native command effects stay closed
	CommandLearn             map[int][]CommandLearnEntry // portrait/growth-row idx -> native level-up command pairs
	AICommandSpell           map[int]int                 // editable item command byte -> spell id; AI ranking remains separate
	Treasures                map[Cell]Treasure           // FDFIELD composition 地形旗標+slot 與 control chest table 的 join
	NativeTreasureEventRules map[int]NativeTreasureEventRule
	// OpenedTreasure is remake-owned state for editable treasure nodes.  It has
	// no asserted native-global address: native [0x53ad5] is a pointer to a
	// 0x20-byte battle-local state table (0x10322 copies it; 0x13d00 writes an
	// index). Ch25 reads entry #12 for dialogue, which does not prove a treasure
	// slot mapping.
	OpenedTreasure map[int]bool
	// 來源:tools/export_engine_assets.py 依地形控制表(doc01 §5)換算,由 Load 讀同目錄
	// map.json 的 "cost" 陣列自動接上(worklist 第 8 輪「地形屬性接線」)。
}

// BindNativeFutureItemRows supplies the raw item-effect row prefix required
// by future-group construction. A private copy prevents UI lifecycle buffers
// or callers from changing an in-progress constructor transaction.
func (s *State) BindNativeFutureItemRows(rows []byte) error {
	if s == nil || len(rows) == 0 || len(rows)%NativeItemEffectRowSize != 0 ||
		len(rows)/NativeItemEffectRowSize > 0x100 {
		return fmt.Errorf("native future item rows: invalid byte length %d", len(rows))
	}
	s.nativeFutureItemRows = append([]byte(nil), rows...)
	return nil
}

// BindNativeMovementCostRows supplies the versioned 0x4e555 movement table
// used by native AI/path helpers. The table is immutable after binding and is
// copied so callers cannot mutate an in-progress decision.
func (s *State) BindNativeMovementCostRows(rows [][]byte) error {
	if s == nil || len(rows) != NativeMovementCostRowCount {
		return fmt.Errorf("native movement rows: invalid row count")
	}
	copyRows := make([][]byte, len(rows))
	for index, row := range rows {
		if len(row) != NativeMovementCostRowSize {
			return fmt.Errorf("native movement selector=%d: invalid row length %d", index, len(row))
		}
		copyRows[index] = append([]byte(nil), row...)
	}
	s.nativeMovementCostRows = copyRows
	return nil
}

// Cell 格子座標。
type Cell struct{ X, Y int }

// DeathEffect 原樣保存 FDFIELD 單位記錄 b22 與 b23..24 的 u16。Type 0/1 是死亡時掉物/金錢；
// 2/3 的特殊事件語意尚未完全解明，runtime 在釘死前不得猜測執行。
type DeathEffect struct {
	Type  int `json:"type"`
	Value int `json:"value"`
}

// Treasure 是一個可編輯的戰場寶物節點。Slot 對應 composition word 低5bit與
// control table 16筆 reward；Hidden 只控制視覺，取得規則相同。
type Treasure struct {
	Slot       int
	Kind       string
	NativeType byte
	Value      int
	Hidden     bool
}

// NativeTreasureEventRule 是已由全域事件 handler 閉合的可編輯特殊寶物規則。
// ItemBySlot 以原始 chest slot 索引；OpenSlots 保存同一事件成功後共同關閉的槽。
type NativeTreasureEventRule struct {
	EventID    int   `json:"event_id"`
	ItemBySlot []int `json:"item_by_slot"`
	OpenSlots  []int `json:"open_slots"`
}

// NativeFieldEvent 是 FDFIELD 控制段的原始兩位元組格子事件列。
// 0x13a44 只把 EventID 寫入全域 selector；高階玩法名稱仍由 handler 證據決定。
type NativeFieldEvent struct {
	EventID  byte `json:"event_id"`
	Selector byte `json:"selector"`
}

// NativeTurnEventControl 保存一筆三位元組 FDFIELD live 回合列。
// Turn 0xff 保留為原始休眠證據，不解讀成第255回合；RawCamp 仍是 selector
// byte，不推成正規化 battle.Camp。
type NativeTurnEventControl struct {
	Turn    byte `json:"turn"`
	EventID byte `json:"event_id"`
	RawCamp byte `json:"raw_camp"`
}

// NativeChestControl preserves one live three-byte FDFIELD chest row.
type NativeChestControl struct {
	RawType byte
	Value   uint16
}

// NativeFieldUnitControl preserves one live 26-byte FDFIELD constructor row.
// It is not a current runtime unit; 0x10b4e consumes matching rows only when a
// future group is appended through 0x10c50.
type NativeFieldUnitControl struct {
	Raw [0x1a]byte
}

// NativePositionRecord is one exact FDFIELD 3N+2 six-byte record.
type NativePositionRecord struct {
	XWord, YWord, RawKey uint16
}

// NativeRuntimeRecordState is one exact saved 0x50-byte current-unit record
// plus the selector result rebuilt from raw +7 in original list order.
type NativeRuntimeRecordState struct {
	Raw          [0x50]byte
	SelectorKey  byte
	SelectorSlot byte
}

type NativeFieldModeRange struct {
	Start int  `json:"start"`
	End   int  `json:"end"`
	Mode  byte `json:"mode"`
}

type NativeFieldTextIndices struct {
	MissingItem int `json:"missing_item"`
	Success     int `json:"success"`
	Final       int `json:"final"`
}

type NativeFieldPresentation struct {
	Archive           string `json:"archive"`
	Resource          int    `json:"resource"`
	Frames            int    `json:"frames"`
	Helper            string `json:"helper"`
	DestinationOffset int    `json:"destination_offset"`
	Stride            int    `json:"stride"`
	Transparent       int    `json:"transparent"`
	DelayHelper       string `json:"delay_helper"`
	DelayTicks        int    `json:"delay_ticks"`
}

// NativeTurnActivation 描述一個已證實的格子 handler 對 live FDFIELD 回合列
// 的改寫。它辨識改寫前的原始列，不是正規化 scenario 排程，也不把 RawCamp
// 解讀成 battle.Camp。
type NativeTurnActivation struct {
	Slot      int  `json:"slot"`
	EventID   int  `json:"event_id"`
	RawCamp   byte `json:"raw_camp"`
	TurnDelta int  `json:"turn_delta"`
}

// NativeFieldEventRule 保存已閉合 handler 的資料，不自行決定 selector 的呼叫時機。
type NativeFieldEventRule struct {
	EventID        int                      `json:"event_id"`
	Selector       byte                     `json:"selector"`
	TriggerGate    string                   `json:"trigger_gate,omitempty"`
	SetModeRanges  []NativeFieldModeRange   `json:"set_mode_ranges,omitempty"`
	SetStateIndex  *int                     `json:"set_state_index,omitempty"`
	SetStateValue  *int                     `json:"set_state_value,omitempty"`
	OnceState      *int                     `json:"once_state_index,omitempty"`
	RequiredItem   *int                     `json:"required_item,omitempty"`
	ConsumeItem    bool                     `json:"consume_item,omitempty"`
	SpawnGroup     *int                     `json:"spawn_group,omitempty"`
	JoinCharacter  *int                     `json:"join_character,omitempty"`
	TextIndices    *NativeFieldTextIndices  `json:"text_indices,omitempty"`
	Presentation   *NativeFieldPresentation `json:"presentation,omitempty"`
	TurnActivation *NativeTurnActivation    `json:"turn_activation,omitempty"`
}

// TreasureAt 查詢尚未取得的寶物格。
func (s *State) TreasureAt(x, y int) (Treasure, bool) {
	if s == nil || s.OpenedTreasure == nil {
		return Treasure{}, false
	}
	t, ok := s.Treasures[Cell{X: x, Y: y}]
	return t, ok && !s.OpenedTreasure[t.Slot]
}

// ClaimTreasure 投影原版 0x190ac：只有站在該格的 active unit 可取；物品放進
// 該單位8格 inventory，滿背包時不開箱；金錢由 caller 加到 campaign bank。
// 原版沒有 camp 限制，因此 Enemy 也能取得並標記 opened。
func (s *State) ClaimTreasure(u *Unit, x, y int) (Treasure, bool) {
	if u == nil || !u.OnField || !u.Alive() || u.X != x || u.Y != y {
		return Treasure{}, false
	}
	t, ok := s.TreasureAt(x, y)
	if !ok {
		return Treasure{}, false
	}
	switch t.Kind {
	case "item":
		if len(u.Inventory) >= 8 {
			return Treasure{}, false
		}
		if !u.AddInventoryItem(t.Value, false) {
			return Treasure{}, false
		}
	case "gold":
		// Game owns the campaign bank; returning the reward lets it add atomically.
	case "event":
		return s.claimNativeTreasureEvent(u, t)
	default:
		return Treasure{}, false
	}
	s.OpenedTreasure[t.Slot] = true
	return t, true
}

func (s *State) claimNativeTreasureEvent(u *Unit, t Treasure) (Treasure, bool) {
	rule, ok := s.NativeTreasureEventRules[t.Value]
	if !ok || t.Slot < 0 || t.Slot >= len(rule.ItemBySlot) || len(rule.OpenSlots) == 0 {
		return Treasure{}, false
	}
	item := rule.ItemBySlot[t.Slot]
	if item <= 0 || item > 0xFF || len(u.Inventory) >= 8 {
		return Treasure{}, false
	}
	seen := map[int]bool{}
	for _, slot := range rule.OpenSlots {
		if slot < 0 || slot >= 16 || seen[slot] {
			return Treasure{}, false
		}
		seen[slot] = true
	}
	if !u.AddInventoryItem(item, false) {
		return Treasure{}, false
	}
	for slot := range seen {
		s.OpenedTreasure[slot] = true
	}
	t.Kind = "item"
	t.Value = item
	return t, true
}

// OnFieldUnit 回傳該格上「已登場且存活」的單位(無則 nil)。
func (s *State) UnitAt(x, y int) *Unit {
	for _, u := range s.Units {
		if u.OnField && u.Alive() && u.X == x && u.Y == y {
			return u
		}
	}
	return nil
}

// AliveCount 各陣營「已登場且存活」數(用於勝敗判定)。
// 注意:待命(未登場)單位不計入 → 敵方援軍未出時不會誤判全滅。
func (s *State) AliveCount(c Camp) int {
	n := 0
	for _, u := range s.Units {
		if u.OnField && u.Alive() && u.Camp == c {
			n++
		}
	}
	return n
}

// PendingCount 某陣營尚未登場(待命)的單位數;>0 表示還有援軍沒出,不該判全滅。
func (s *State) PendingCount(c Camp) int {
	n := 0
	for _, u := range s.Units {
		if !u.OnField && u.Alive() && u.Camp == c {
			n++
		}
	}
	for _, u := range s.Roster {
		if s.PendingGroups[u.Group] && u.Alive() && u.Camp == c {
			n++
		}
	}
	return n
}

// ---- 載入(units.json,tools/export_units.py 產生)----

type unitsFile struct {
	Map       int    `json:"map"`
	W         int    `json:"w"`
	H         int    `json:"h"`
	OwnDeploy []Cell `json:"own_deploy"`
	// Optional raw global snapshot; absent legacy/editable files must not
	// fabricate provenance for native round predicates.
	NativeRoundCounter *int `json:"native_round_counter,omitempty"`
	Chests             []struct {
		Slot       int    `json:"slot"`
		Kind       string `json:"type"`
		NativeType byte   `json:"native_type"`
		Value      int    `json:"value"`
	} `json:"chests,omitempty"`
	NativeTreasureEventRules []NativeTreasureEventRule `json:"native_treasure_event_rules,omitempty"`
	Units                    []struct {
		Camp               string       `json:"camp"`
		ClassID            int          `json:"cls"`
		Name               string       `json:"name"`
		ClsName            string       `json:"cls_name"`
		Lv                 int          `json:"lv"`
		HP                 int          `json:"hp"`
		MP                 int          `json:"mp"`
		Spells             []int        `json:"spells"`
		InitialCommandMask []byte       `json:"initial_command_mask,omitempty"`
		Inventory          []int        `json:"inventory,omitempty"`
		InventorySlots     []int        `json:"inventory_slots,omitempty"`
		DeathEffect        *DeathEffect `json:"death_effect,omitempty"`
		DeathReward        *DeathEffect `json:"death_reward,omitempty"`
		AP                 int          `json:"ap"`
		DP                 int          `json:"dp"`
		HIT                int          `json:"hit"`
		EV                 int          `json:"ev"`
		Crit               int          `json:"crit"`
		MV                 int          `json:"mv"`
		AtkMin             int          `json:"atk_min"` // 攻擊距離下限(0=預設1;沒此欄的舊版 units.json 一律 0,doc32)
		AtkMax             int          `json:"atk_max"` // 攻擊距離上限(0=預設1)
		Ex                 int          `json:"ex"`      // 每級經驗(doc02 §4.5「守方每級經驗」;export_units.py 新增欄,
		// 舊版 units.json 沒有此欄時 json.Unmarshal 留 0,見 Unit.ExpPerLevel 註解)
		Portrait             int   `json:"portrait"`
		Fig                  int   `json:"fig"`
		BattleFig            *int  `json:"battle_fig,omitempty"`
		NativeIdentity       *int  `json:"native_identity,omitempty"`
		NativeRecordByte8    *byte `json:"native_record_byte8,omitempty"`
		NativePositionRecord *struct {
			XWord  uint16 `json:"x_word"`
			YWord  uint16 `json:"y_word"`
			RawKey uint16 `json:"raw_key"`
		} `json:"native_position_record,omitempty"`
		NativeRecordRace        *byte                   `json:"native_record_race,omitempty"`
		NativeRecordClass       *byte                   `json:"native_record_class,omitempty"`
		MapSelectorSlot         *int                    `json:"map_selector_slot,omitempty"`
		MapSelectorKey          *int                    `json:"map_selector_key,omitempty"`
		NativeRecordByte5       *byte                   `json:"native_record_byte5,omitempty"`
		NativeRecordByte6       *byte                   `json:"native_record_byte6,omitempty"`
		NativeRecordByte34      *byte                   `json:"native_record_byte34,omitempty"`
		NativeRecordByte35      *byte                   `json:"native_record_byte35,omitempty"`
		NativeRecordByte36      *byte                   `json:"native_record_byte36,omitempty"`
		NativeRecordByte3D      *byte                   `json:"native_record_byte3d,omitempty"`
		NativeRecordDeathEffect []byte                  `json:"native_record_death_effect,omitempty"`
		NativeSourceByte3       *byte                   `json:"native_source_byte3,omitempty"`
		NativeSourceByte20      *byte                   `json:"native_source_byte20,omitempty"`
		NativeSourceByte25      *byte                   `json:"native_source_byte25,omitempty"`
		NativeRecordWord42      *uint16                 `json:"native_record_word42,omitempty"`
		NativeRecordWord46      *uint16                 `json:"native_record_word46,omitempty"`
		Group                   int                     `json:"group"`
		NativeConstructor       *NativeConstructorTable `json:"native_constructor,omitempty"`
		X                       int                     `json:"x"`
		Y                       int                     `json:"y"`
	} `json:"units"`
}

func campFrom(s string) Camp {
	switch s {
	case "own":
		return Own
	case "ally":
		return Ally
	default:
		return Enemy
	}
}

// Load 從 units.json 建出戰鬥初始狀態。我方(own)依序放到部署格。
func Load(path string) (*State, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f unitsFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, err
	}
	st := &State{W: f.W, H: f.H, OwnDeploy: f.OwnDeploy, Turn: 1, Flags: map[string]bool{},
		Treasures: map[Cell]Treasure{}, OpenedTreasure: map[int]bool{},
		NativeTreasureEventRules: map[int]NativeTreasureEventRule{}}
	for _, rule := range f.NativeTreasureEventRules {
		if rule.EventID < 0 || rule.EventID >= 90 ||
			len(rule.ItemBySlot) == 0 || len(rule.OpenSlots) == 0 {
			return nil, fmt.Errorf("battle: invalid native treasure event rule %d", rule.EventID)
		}
		if _, exists := st.NativeTreasureEventRules[rule.EventID]; exists {
			return nil, fmt.Errorf("battle: duplicate native treasure event rule %d", rule.EventID)
		}
		st.NativeTreasureEventRules[rule.EventID] = rule
	}
	if f.NativeRoundCounter != nil && *f.NativeRoundCounter > 0 {
		st.NativeRoundCounter = *f.NativeRoundCounter
	}
	for _, u := range f.Units {
		if err := u.NativeConstructor.validate(); err != nil {
			return nil, fmt.Errorf("battle: native_constructor: %w", err)
		}
		camp := campFrom(u.Camp)
		inventory, equipped, runtimeSlots := materializeInventory(u.InventorySlots, u.Inventory)
		nu := &Unit{
			Camp: camp, Name: u.Name, ClsName: u.ClsName, ClassID: u.ClassID, Lv: u.Lv,
			HP: u.HP, MaxHP: u.HP, MP: u.MP, MaxMP: u.MP, AP: u.AP, DP: u.DP, MV: u.MV,
			HIT: u.HIT, EV: u.EV, CritPct: u.Crit, ExpPerLevel: u.Ex,
			AtkMin: u.AtkMin, AtkMax: u.AtkMax,
			Portrait: u.Portrait, Fig: u.Fig, X: u.X, Y: u.Y,
			Spells: append([]int(nil), u.Spells...), Inventory: inventory, Equipped: equipped, InventorySlots: runtimeSlots,
			DeathEffect:       u.DeathEffect,
			DeathReward:       u.DeathReward,
			NativeConstructor: u.NativeConstructor,
			Group:             u.Group, OnField: true, // 預設登場;Scenario 會把待命 group 設 false
		}
		// 0x10eed initializes a freshly constructed record's byte +5 to zero.
		// Do not fabricate a raw value for an already-zero-HP input record: its
		// death writer/provenance is not present in this JSON boundary.
		if nu.HP > 0 {
			nu.NativeRecordByte5 = 0
			nu.HasNativeRecordByte5 = true
		}
		if u.NativeRecordByte5 != nil {
			nu.NativeRecordByte5 = *u.NativeRecordByte5
			nu.HasNativeRecordByte5 = true
		}
		if flags, flagErr := NativeInventoryFlagsFromSource(u.InventorySlots); flagErr == nil {
			nu.NativeInventoryFlags = flags
		}
		// Older generated JSON lacks battle_fig; preserve its compatibility
		// behavior, while new exports carry the direct FDFIELD-b1 value.
		nu.BattleFig = u.Fig
		if u.BattleFig != nil {
			nu.BattleFig, nu.HasBattleFig = *u.BattleFig, true
		}
		if u.NativeIdentity != nil {
			if *u.NativeIdentity < 0 || *u.NativeIdentity > 0xff {
				return nil, fmt.Errorf("battle: unit %d native_identity %d outside byte range", len(st.Units), *u.NativeIdentity)
			}
			nu.NativeIdentity, nu.HasNativeIdentity = *u.NativeIdentity, true
		}
		if u.NativeRecordByte8 != nil {
			nu.NativeRecordByte8, nu.HasNativeRecordByte8 = *u.NativeRecordByte8, true
		} else if nu.HasNativeIdentity {
			// Compatibility for verified player-persistent assets created
			// before raw +8 was represented separately.
			nu.NativeRecordByte8 = byte(nu.NativeIdentity)
			nu.HasNativeRecordByte8 = true
		}
		if u.NativePositionRecord != nil {
			nu.NativePositionRecord = NativePositionRecord{
				XWord:  u.NativePositionRecord.XWord,
				YWord:  u.NativePositionRecord.YWord,
				RawKey: u.NativePositionRecord.RawKey,
			}
			nu.HasNativePositionRecord = true
		}
		if u.NativeRecordRace != nil {
			nu.NativeRecordRace, nu.HasNativeRecordRace = *u.NativeRecordRace, true
		}
		if u.NativeRecordClass != nil {
			nu.NativeRecordClass, nu.HasNativeRecordClass = *u.NativeRecordClass, true
		}
		if u.NativeConstructor != nil {
			if !nu.HasNativeRecordRace {
				nu.NativeRecordRace, nu.HasNativeRecordRace = u.NativeConstructor.Record[0], true
			}
			if !nu.HasNativeRecordClass {
				nu.NativeRecordClass, nu.HasNativeRecordClass = u.NativeConstructor.Record[1], true
			}
		}
		if u.MapSelectorSlot != nil {
			nu.MapSelectorSlot, nu.HasMapSelectorSlot = *u.MapSelectorSlot, true
		}
		if u.MapSelectorKey != nil {
			nu.MapSelectorKey, nu.HasMapSelectorKey = *u.MapSelectorKey, true
			// FDFIELD b0 is copied directly to native record +6 by the
			// constructor; map_selector_key is retained as separate provenance.
			nu.NativeRecordByte6, nu.HasNativeRecordByte6 = byte(*u.MapSelectorKey), true
		}
		if u.NativeRecordByte6 != nil {
			nu.NativeRecordByte6, nu.HasNativeRecordByte6 = *u.NativeRecordByte6, true
		}
		if u.NativeRecordByte34 != nil {
			nu.NativeRecordByte34, nu.HasNativeRecordByte34 = *u.NativeRecordByte34, true
		}
		if u.NativeRecordByte35 != nil {
			nu.NativeRecordByte35, nu.HasNativeRecordByte35 = *u.NativeRecordByte35, true
		}
		if u.NativeRecordByte36 != nil {
			nu.NativeRecordByte36, nu.HasNativeRecordByte36 = *u.NativeRecordByte36, true
		}
		if u.NativeRecordByte3D != nil {
			nu.NativeRecordByte3D, nu.HasNativeRecordByte3D = *u.NativeRecordByte3D, true
		}
		if len(u.NativeRecordDeathEffect) != 0 {
			if len(u.NativeRecordDeathEffect) != 3 {
				return nil, fmt.Errorf(
					"battle: unit %d native_record_death_effect needs 3 bytes",
					len(st.Units),
				)
			}
			copy(nu.NativeRecordDeathEffect[:], u.NativeRecordDeathEffect)
			nu.HasNativeRecordDeathEffect = true
		}
		if u.NativeSourceByte3 != nil {
			nu.NativeFDFIELDSourceByte3 = *u.NativeSourceByte3
		}
		if u.NativeSourceByte20 != nil {
			nu.NativeFDFIELDSourceByte20 = *u.NativeSourceByte20
		}
		if u.NativeSourceByte25 != nil {
			nu.NativeFDFIELDSourceByte25 = *u.NativeSourceByte25
		}
		if u.NativeRecordWord42 != nil {
			nu.NativeRecordWord42, nu.HasNativeRecordWord42 = *u.NativeRecordWord42, true
			nu.HP, nu.MaxHP = int(*u.NativeRecordWord42), int(*u.NativeRecordWord42)
		}
		if u.NativeRecordWord46 != nil {
			nu.NativeRecordWord46, nu.HasNativeRecordWord46 = *u.NativeRecordWord46, true
			nu.MP, nu.MaxMP = int(*u.NativeRecordWord46), int(*u.NativeRecordWord46)
		}
		if err := nu.SetInitialCommandMask(u.InitialCommandMask); err != nil {
			return nil, fmt.Errorf("battle: unit %d initial_command_mask: %w", len(st.Units), err)
		}
		// 註:不再自動把 own 塞部署格 — 部署格保留給 scenario 主角隊(spawn_party);
		// FDFIELD 的 own(如哈諾/哈瓦特)用自己的出場座標(房子位置),由事件按回合放出。
		st.Units = append(st.Units, nu)
	}
	mapPath := filepath.Join(filepath.Dir(path), "map.json")
	st.Cost = loadTerrainCost(mapPath, f.W, f.H)
	st.NativeCompositionEventBytes = loadNativeCompositionEventBytes(mapPath, f.W, f.H)
	st.NativeMapEventGrid, st.HasNativeMapEventGrid = loadNativeMapEventGrid(mapPath, f.W, f.H)
	if st.HasNativeMapEventGrid {
		resetNativeMapEventGrid(st.NativeMapEventGrid)
	}
	var nativeRoundSeed int
	st.NativeTurnEventControls, nativeRoundSeed, st.HasNativeTurnEventControlState =
		loadNativeTurnEventControls(mapPath, f.W, f.H)
	if st.HasNativeTurnEventControlState && st.NativeRoundCounter == 0 {
		st.NativeRoundCounter = nativeRoundSeed
	}
	st.NativeFieldEventSlots, st.NativeFieldEvents, st.NativeFieldEventRules =
		loadNativeFieldEvents(mapPath, f.W, f.H)
	st.NativeTileBlitModes, st.NativeTerrainControl, st.NativeTerrainMoveCodes = loadNativeTerrainRendererInputs(mapPath, f.W, f.H)
	// 0x4dbfc is the runtime constructor for composition byte+3: it replaces
	// every serialized value with 0xff before 0x14818/0x122dc mutate the grid.
	// Keep the exported field only as exact map provenance, never as live state.
	for i := range st.NativeTileBlitModes {
		st.NativeTileBlitModes[i] = 0xff
	}
	st.Treasures = loadTreasures(filepath.Join(filepath.Dir(path), "map.json"), f.W, f.H, f.Chests)
	return st, nil
}

// mapCostFile map.json 裡跟地形成本相關的欄位(其餘欄位 main.go 的 MapData 自己讀,這裡只挑 cost 用)。
type mapCostFile struct {
	W                           int                    `json:"w"`
	H                           int                    `json:"h"`
	Cost                        []int                  `json:"cost"`
	NativeCompositionEventBytes []byte                 `json:"native_composition_event_bytes"`
	NativeFieldEventSlots       []int                  `json:"native_field_event_slots"`
	NativeFieldEvents           []NativeFieldEvent     `json:"native_field_events"`
	NativeFieldEventRules       []NativeFieldEventRule `json:"native_field_event_rules"`
	NativeTileBlitModes         []byte                 `json:"native_tile_blit_modes"`
	NativeTerrainControl        []byte                 `json:"native_terrain_control"`
	Tiles                       []int                  `json:"tiles"`
	TreasureSlots               []int                  `json:"treasure_slots"`
	TreasureHidden              []bool                 `json:"treasure_hidden"`
}

// loadNativeTurnEventControls 只接受完整16筆 raw table；0xff 列保持可見，
// 不提升成第255回合的 scenario event。
func loadNativeTurnEventControls(
	mapJSONPath string,
	w, h int,
) ([16]NativeTurnEventControl, int, bool) {
	var out [16]NativeTurnEventControl
	if w <= 0 || h <= 0 {
		return out, 0, false
	}
	var mapIndex int
	if n, err := fmt.Sscanf(filepath.Base(filepath.Dir(mapJSONPath)), "map%d", &mapIndex); err != nil || n != 1 || mapIndex < 0 {
		return out, 0, false
	}
	catalogPath := filepath.Join(filepath.Dir(filepath.Dir(mapJSONPath)), "native_turn_event_controls.json")
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		return out, 0, false
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != nativeTurnEventCatalogSHA256 {
		return out, 0, false
	}
	var catalog struct {
		SchemaVersion int `json:"schema_version"`
		Source        struct {
			File   string `json:"file"`
			Size   int    `json:"size"`
			MD5    string `json:"md5"`
			SHA256 string `json:"sha256"`
		} `json:"source"`
		RoundSeed struct {
			Value  int    `json:"value"`
			Writer string `json:"writer"`
			Source struct {
				File   string `json:"file"`
				Size   int    `json:"size"`
				MD5    string `json:"md5"`
				SHA256 string `json:"sha256"`
			} `json:"source"`
		} `json:"round_seed"`
		Maps []struct {
			Map             int                      `json:"map"`
			ControlResource string                   `json:"control_resource"`
			ControlSHA256   string                   `json:"control_sha256"`
			Controls        []NativeTurnEventControl `json:"controls"`
		} `json:"maps"`
	}
	if json.Unmarshal(raw, &catalog) != nil || catalog.SchemaVersion != 1 ||
		catalog.Source.File != "FDFIELD.DAT" || catalog.Source.Size != 243169 ||
		catalog.Source.MD5 != "ecdb0436d26adfe5d107f2713fa7e9a2" ||
		catalog.Source.SHA256 != "b0cf75d94f58603f091c7462c0494f0e83bd6edfb04c1acbf83ed4d938c7a513" ||
		catalog.RoundSeed.Value != 1 || catalog.RoundSeed.Writer != "0x2066e" ||
		catalog.RoundSeed.Source.File != "FD2.EXE" || catalog.RoundSeed.Source.Size != 357074 ||
		catalog.RoundSeed.Source.MD5 != "b97caf2239a27a896069d03549d96e1e" ||
		catalog.RoundSeed.Source.SHA256 != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f" ||
		len(catalog.Maps) != 33 {
		return out, 0, false
	}
	seen := [33]bool{}
	for _, entry := range catalog.Maps {
		if entry.Map < 0 || entry.Map >= len(seen) || seen[entry.Map] ||
			entry.ControlResource != fmt.Sprintf("FDFIELD_%03d.bin", entry.Map*3+1) ||
			len(entry.ControlSHA256) != 64 || len(entry.Controls) != len(out) {
			return out, 0, false
		}
		seen[entry.Map] = true
	}
	if !seen[mapIndex] {
		return out, 0, false
	}
	for _, entry := range catalog.Maps {
		if entry.Map == mapIndex {
			copy(out[:], entry.Controls)
			return out, catalog.RoundSeed.Value, true
		}
	}
	return out, 0, false
}

func loadTreasures(mapJSONPath string, w, h int, chests []struct {
	Slot       int    `json:"slot"`
	Kind       string `json:"type"`
	NativeType byte   `json:"native_type"`
	Value      int    `json:"value"`
}) map[Cell]Treasure {
	out := map[Cell]Treasure{}
	raw, err := os.ReadFile(mapJSONPath)
	if err != nil {
		return out
	}
	var m mapCostFile
	if json.Unmarshal(raw, &m) != nil || m.W != w || m.H != h || len(m.TreasureSlots) != w*h {
		return out
	}
	defs := make(map[int]Treasure, len(chests))
	for _, c := range chests {
		if c.Slot >= 0 && c.Slot < 16 &&
			(c.Kind == "item" || c.Kind == "gold" || c.Kind == "event") &&
			c.Value > 0 {
			defs[c.Slot] = Treasure{
				Slot: c.Slot, Kind: c.Kind, NativeType: c.NativeType, Value: c.Value,
			}
		}
	}
	for i, slot := range m.TreasureSlots {
		if slot < 0 {
			continue
		}
		if t, ok := defs[slot]; ok {
			t.Hidden = len(m.TreasureHidden) == w*h && m.TreasureHidden[i]
			out[Cell{X: i % w, Y: i / w}] = t
		}
	}
	return out
}

// loadTerrainCost 嘗試讀 units.json 同目錄的 map.json,取其 "cost" 陣列(tools/
// export_engine_assets.py 產生;doc01 §5 地形控制表換算)。檔案不存在、沒有 cost 欄位、
// 或尺寸對不上 units.json 的 w/h,一律回 nil(MoveCost 退回全平地=1,不 fail Load)。
func loadTerrainCost(mapJSONPath string, w, h int) []int {
	raw, err := os.ReadFile(mapJSONPath)
	if err != nil {
		return nil
	}
	var m mapCostFile
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	if len(m.Cost) == 0 || m.W != w || m.H != h || len(m.Cost) != w*h {
		return nil
	}
	return m.Cost
}

// loadNativeCompositionEventBytes accepts only an exact FDFIELD composition
// export. These bytes are immutable source provenance, not the live +2 flags:
// 0x4dbfc first masks each byte with 0x1f and caller-specific writers may then
// add 0x40/0x80.
func loadNativeCompositionEventBytes(mapJSONPath string, w, h int) []byte {
	raw, err := os.ReadFile(mapJSONPath)
	if err != nil {
		return nil
	}
	var m mapCostFile
	if json.Unmarshal(raw, &m) != nil || m.W != w || m.H != h ||
		len(m.NativeCompositionEventBytes) != w*h {
		return nil
	}
	return append([]byte(nil), m.NativeCompositionEventBytes...)
}

// loadNativeMapEventGrid rebuilds the exact four-byte cell prefix consumed by
// 0x12e38/0x15df3 from the same FDFIELD export that produced map.json.  The
// first four bytes are the native width/height header; each cell then keeps
// tile word (+0/+1) and event word (+2/+3).  Callers still own the mutable
// 0x4dbfc reset and later event writers.
func loadNativeMapEventGrid(mapJSONPath string, w, h int) ([]byte, bool) {
	if w <= 0 || h <= 0 || w > 0xff || h > 0xff {
		return nil, false
	}
	raw, err := os.ReadFile(mapJSONPath)
	if err != nil {
		return nil, false
	}
	var m mapCostFile
	if json.Unmarshal(raw, &m) != nil || m.W != w || m.H != h ||
		len(m.Tiles) != w*h || len(m.NativeCompositionEventBytes) != w*h ||
		len(m.NativeTileBlitModes) != w*h {
		return nil, false
	}
	grid := make([]byte, 4+4*w*h)
	grid[0], grid[2] = byte(w), byte(h)
	for index, tile := range m.Tiles {
		if tile < 0 || tile > 0xffff ||
			m.NativeCompositionEventBytes[index] < 0 ||
			m.NativeCompositionEventBytes[index] > 0xff ||
			m.NativeTileBlitModes[index] < 0 || m.NativeTileBlitModes[index] > 0xff {
			return nil, false
		}
		offset := 4 + 4*index
		binary.LittleEndian.PutUint16(grid[offset:offset+2], uint16(tile))
		eventWord := uint16(m.NativeCompositionEventBytes[index]) |
			uint16(m.NativeTileBlitModes[index])<<8
		binary.LittleEndian.PutUint16(grid[offset+2:offset+4], eventWord)
	}
	return grid, true
}

// resetNativeMapEventGrid reproduces 0x4dbfc's constructor prefix.  It is
// intentionally in-place because the original buffer is subsequently owned
// by event/AI writers; malformed dimensions are rejected by the loader.
func resetNativeMapEventGrid(grid []byte) {
	if len(grid) < 4 {
		return
	}
	count := int(grid[0]) * int(grid[2])
	if count < 0 || len(grid) != 4+4*count {
		return
	}
	for index := 0; index < count; index++ {
		offset := 4 + 4*index
		grid[offset+1] &= 0x03
		grid[offset+2] &= 0x1f
		grid[offset+3] = 0xff
	}
}

func loadNativeFieldEvents(
	mapJSONPath string,
	w, h int,
) ([]int, []NativeFieldEvent, []NativeFieldEventRule) {
	raw, err := os.ReadFile(mapJSONPath)
	if err != nil {
		return nil, nil, nil
	}
	var m mapCostFile
	if json.Unmarshal(raw, &m) != nil ||
		m.W != w ||
		m.H != h ||
		len(m.NativeFieldEventSlots) != w*h ||
		len(m.NativeFieldEvents) != 16 {
		return nil, nil, nil
	}
	for _, slot := range m.NativeFieldEventSlots {
		if slot < -1 || slot >= len(m.NativeFieldEvents) {
			return nil, nil, nil
		}
	}
	seen := map[int]bool{}
	for _, rule := range m.NativeFieldEventRules {
		if rule.EventID < 0 || rule.EventID >= 90 || seen[rule.EventID] {
			return nil, nil, nil
		}
		if (rule.SetStateIndex == nil) != (rule.SetStateValue == nil) ||
			(rule.SetStateIndex != nil &&
				(*rule.SetStateIndex < 0 || *rule.SetStateIndex >= 0x20 ||
					*rule.SetStateValue < 0 || *rule.SetStateValue > 0xff)) {
			return nil, nil, nil
		}
		if rule.OnceState != nil && (*rule.OnceState < 0 || *rule.OnceState >= 0x20) {
			return nil, nil, nil
		}
		if rule.TurnActivation != nil &&
			(rule.TurnActivation.Slot < 0 || rule.TurnActivation.Slot >= 16 ||
				rule.TurnActivation.EventID < 0 || rule.TurnActivation.EventID >= 90 ||
				rule.TurnActivation.TurnDelta < 0 || rule.TurnActivation.TurnDelta > 0xff) {
			return nil, nil, nil
		}
		seen[rule.EventID] = true
		for _, modeRange := range rule.SetModeRanges {
			if modeRange.Start < 0 || modeRange.End < modeRange.Start ||
				modeRange.Mode > 0x0F {
				return nil, nil, nil
			}
		}
	}
	return append([]int(nil), m.NativeFieldEventSlots...),
		append([]NativeFieldEvent(nil), m.NativeFieldEvents...),
		append([]NativeFieldEventRule(nil), m.NativeFieldEventRules...)
}

// loadNativeTerrainRendererInputs accepts only a complete map export. It
// deliberately keeps malformed data nil rather than repurposing Cost.
func loadNativeTerrainRendererInputs(mapJSONPath string, w, h int) ([]byte, []byte, []byte) {
	raw, err := os.ReadFile(mapJSONPath)
	if err != nil {
		return nil, nil, nil
	}
	var m mapCostFile
	if json.Unmarshal(raw, &m) != nil || m.W != w || m.H != h || len(m.Tiles) != w*h || len(m.NativeTileBlitModes) != w*h || len(m.NativeTerrainControl) == 0 || len(m.NativeTerrainControl)%4 != 0 {
		return nil, nil, nil
	}
	moveCodes := make([]byte, len(m.Tiles))
	for _, tile := range m.Tiles {
		if tile < 0 || tile&^0x3ff != 0 || tile >= len(m.NativeTerrainControl)/4 {
			return nil, nil, nil
		}
	}
	for i, tile := range m.Tiles {
		moveCodes[i] = m.NativeTerrainControl[tile*4+1]
	}
	return append([]byte(nil), m.NativeTileBlitModes...), append([]byte(nil), m.NativeTerrainControl...), moveCodes
}

// AddUnit 把一個單位加入戰場(事件 spawn / 主角隊進場用)。 Legacy callers
// do not materialize a native selector slot implicitly.
func (s *State) AddUnit(u *Unit) { s.Units = append(s.Units, u) }

// AppendNativeMapSelectorBatch atomically appends one proven construction
// batch and assigns its unit+2 slots using the State's process-global cache.
// Callers must preserve the native order (party first, then scripted groups)
// and must not call this for legacy units without explicit raw keys. A failed
// batch changes neither Units nor the cache.
func (s *State) AppendNativeMapSelectorBatch(units []*Unit) error {
	if s == nil {
		return fmt.Errorf("native map selector: nil state")
	}
	cache := s.NativeMapSelectorCache
	if cache == nil {
		cache = &fdicon.NativeSelectorCache{}
	}
	if err := MaterializeNativeMapSelectorSlots(units, cache); err != nil {
		return err
	}
	s.NativeMapSelectorCache = cache
	if !s.HasNativeMapCycleState {
		s.NativeMapCycleState = fdicon.NativeMapSpriteCycleState{}
		s.HasNativeMapCycleState = true
	}
	if !s.HasNativeTerrainPhaseState {
		s.NativeTerrainPhaseState = fdother.NativeTerrainPhaseState{}
		s.HasNativeTerrainPhaseState = true
	}
	if !s.HasNativeMapBinaryTimingState {
		s.NativeTerrainFlipState = fdicon.NativeBinaryTickState{}
		s.NativeUnitPixelShiftState = fdicon.NativeBinaryTickState{}
		s.HasNativeMapBinaryTimingState = true
	}
	s.Units = append(s.Units, units...)
	return nil
}

// AdvanceNativeMapPresentationCycles applies one proven 0x1297d call to the
// battle-local raw globals. The signed timer value must already be the low
// BIOS word observed by the caller; legacy states fail closed.
func (s *State) AdvanceNativeMapPresentationCycles(rawTimerTick int) bool {
	if s == nil || !s.HasNativeMapCycleState || rawTimerTick < -0x8000 || rawTimerTick > 0x7fff {
		return false
	}
	s.NativeMapCycleState = fdicon.AdvanceNativeMapSpriteCycles(s.NativeMapCycleState, rawTimerTick)
	return true
}

// AdvanceNativeTerrainPhase applies one proven 0x11eee phase-selection call.
// override is raw [0x51a93]: -1 for the BIOS-timed path or 0..19 for the
// explicit selector path.
func (s *State) AdvanceNativeTerrainPhase(rawTimerTick, override int) bool {
	if s == nil || !s.HasNativeTerrainPhaseState {
		return false
	}
	next, err := fdother.AdvanceNativeTerrainPhase(s.NativeTerrainPhaseState, rawTimerTick, override)
	if err != nil {
		return false
	}
	s.NativeTerrainPhaseState = next
	return true
}

func (s *State) AdvanceNativeTerrainFlip(rawTimerTick int) bool {
	if s == nil || !s.HasNativeMapBinaryTimingState {
		return false
	}
	next, err := fdicon.AdvanceNativeBinaryTick(s.NativeTerrainFlipState, rawTimerTick)
	if err != nil {
		return false
	}
	s.NativeTerrainFlipState = next
	return true
}

func (s *State) AdvanceNativeUnitPixelShift(rawTimerTick int) bool {
	if s == nil || !s.HasNativeMapBinaryTimingState {
		return false
	}
	next, err := fdicon.AdvanceNativeBinaryTick(s.NativeUnitPixelShiftState, rawTimerTick)
	if err != nil {
		return false
	}
	s.NativeUnitPixelShiftState = next
	return true
}

// AppendNativeMapSelectorBatchOrLegacy attempts the evidenced constructor
// contract, but preserves playable legacy construction if editable source data
// is malformed.  The failure is retained on State so renderers cannot quietly
// claim a partial native selector state.
func (s *State) AppendNativeMapSelectorBatchOrLegacy(units []*Unit) {
	if err := s.AppendNativeMapSelectorBatch(units); err != nil {
		s.NativeMapSelectorError = err
		s.Units = append(s.Units, units...)
	}
}

// NativeMapSpriteKey resolves an explicitly materialized unit+2 slot back to
// its FDICON raw key.  It returns false unless the whole State remains in the
// proven native construction mode; callers must then retain their legacy Fig
// presentation rather than guessing a raw selector.
func (s *State) NativeMapSpriteKey(u *Unit) (int, bool) {
	if s == nil || s.NativeMapSelectorCache == nil || s.NativeMapSelectorError != nil ||
		u == nil || !u.HasMapSelectorSlot {
		return 0, false
	}
	key, err := s.NativeMapSelectorCache.KeyForSlot(u.MapSelectorSlot)
	if err != nil {
		return 0, false
	}
	return key, true
}

// AppendGroup is the normalized compatibility append used by existing event
// playback. It preserves FDFIELD row order and selector-cache order, but it
// does not execute 0x10c50's table projection, [0x53afa] placement branch,
// inventory initialization, or 0x1b750 recomputation. Callers must not cite
// this helper as the native constructor.
func (s *State) AppendGroup(group int) int {
	if s == nil || len(s.Roster) == 0 {
		return 0
	}
	remaining := s.Roster[:0]
	batch := make([]*Unit, 0)
	for _, u := range s.Roster {
		if u.Group != group {
			remaining = append(remaining, u)
			continue
		}
		u.OnField = true
		u.OffX, u.OffY = 0, 0
		batch = append(batch, u)
	}
	s.Roster = remaining
	if len(batch) == 0 {
		return 0
	}
	if s.NativeMapSelectorCache != nil && s.NativeMapSelectorError == nil {
		s.AppendNativeMapSelectorBatchOrLegacy(batch)
	} else {
		s.Units = append(s.Units, batch...)
	}
	return len(batch)
}

// AppendGroupWithNativePlacement atomically applies the proven
// 0x10B4E→0x10C50 transaction represented by the typed remake: row order,
// per-call [0x53AFA] placement, table-derived base fields, constructor
// inventory cells, 0x1B750 effective-stat recomputation and 0x11019 selector
// allocation. Fields without a typed consumer remain raw provenance; this is
// not a claim of byte-identical 0x50-byte record storage.
func (s *State) AppendGroupWithNativePlacement(group int, rawGate byte) (int, error) {
	if s == nil || len(s.Roster) == 0 {
		return 0, fmt.Errorf("native future group %d: runtime roster unavailable", group)
	}
	if len(s.nativeFutureItemRows) == 0 {
		return 0, fmt.Errorf("native future group %d: item effect rows unavailable", group)
	}
	if s.NativeMapSelectorError != nil || (len(s.Units) != 0 && s.NativeMapSelectorCache == nil) {
		return 0, fmt.Errorf("native future group %d: selector state unavailable", group)
	}
	batch := make([]*Unit, 0)
	remaining := make([]*Unit, 0, len(s.Roster))
	for _, unit := range s.Roster {
		if unit != nil && unit.Group == group {
			batch = append(batch, unit)
		} else {
			remaining = append(remaining, unit)
		}
	}
	if len(batch) == 0 {
		return 0, fmt.Errorf("native future group %d: no matching FDFIELD rows", group)
	}

	prospective := append([]*Unit(nil), s.Units...)
	prepared := make([]*Unit, 0, len(batch))
	for i, unit := range batch {
		if !unit.HasNativePositionRecord {
			return 0, fmt.Errorf(
				"native future group %d row %d: six-byte position record unavailable",
				group, i,
			)
		}
		cell, err := NativeFutureGroupPlacement(
			s.W, s.H, s.NativeCompositionEventBytes, prospective,
			unit.NativePositionRecord, rawGate,
		)
		if err != nil {
			return 0, fmt.Errorf("native future group %d row %d: %w", group, i, err)
		}
		shadow := *unit
		shadow.Inventory = append([]int(nil), unit.Inventory...)
		shadow.Equipped = append([]bool(nil), unit.Equipped...)
		shadow.InventorySlots = append([]int(nil), unit.InventorySlots...)
		shadow.NativeInventoryFlags = append([]int(nil), unit.NativeInventoryFlags...)
		shadow.Spells = append([]int(nil), unit.Spells...)
		if err := MaterializeNativeFutureConstructor(&shadow, s.nativeFutureItemRows); err != nil {
			return 0, fmt.Errorf("native future group %d row %d: %w", group, i, err)
		}
		if !shadow.SetMapPlacement(cell.X, cell.Y, 0) {
			return 0, fmt.Errorf("native future group %d row %d: invalid placement", group, i)
		}
		// The next row observes this just-constructed runtime record through
		// 0x145CD. Materialize only the local shadow during preflight; the real
		// unit remains untouched until the whole batch succeeds.
		if err := shadow.MaterializeNativeMapPresentation(); err != nil {
			return 0, fmt.Errorf("native future group %d row %d: %w", group, i, err)
		}
		shadow.OnField = true
		prepared = append(prepared, &shadow)
		prospective = append(prospective, &shadow)
	}

	candidate := *s
	candidate.Units = append([]*Unit(nil), s.Units...)
	candidate.Roster = remaining
	if s.NativeMapSelectorCache != nil {
		candidate.NativeMapSelectorCache = s.NativeMapSelectorCache.Clone()
	}
	if err := candidate.AppendNativeMapSelectorBatch(prepared); err != nil {
		return 0, fmt.Errorf("native future group %d selector commit: %w", group, err)
	}

	s.Units = candidate.Units
	s.Roster = candidate.Roster
	s.NativeMapSelectorCache = candidate.NativeMapSelectorCache
	s.NativeMapSelectorError = nil
	s.NativeMapCycleState = candidate.NativeMapCycleState
	s.HasNativeMapCycleState = candidate.HasNativeMapCycleState
	s.NativeTerrainPhaseState = candidate.NativeTerrainPhaseState
	s.HasNativeTerrainPhaseState = candidate.HasNativeTerrainPhaseState
	s.NativeTerrainFlipState = candidate.NativeTerrainFlipState
	s.NativeUnitPixelShiftState = candidate.NativeUnitPixelShiftState
	s.HasNativeMapBinaryTimingState = candidate.HasNativeMapBinaryTimingState
	return len(prepared), nil
}

// SpawnGroup 讓既有重製劇本的 group 登場，可另行覆寫陣營與本回合行動狀態。
// 它保留舊的滑入與環狀錯位呈現，不等同原版 0x10b4e/0x10c50 constructor。
func (s *State) SpawnGroup(group int, camp Camp, changeCamp, act bool) int {
	n := 0
	before := len(s.Units)
	if appended := s.AppendGroup(group); appended > 0 {
		for _, u := range s.Units[before:] {
			u.OffY = -56
			if changeCamp {
				u.Camp = camp
			}
			u.Acted = !act
			if occ := s.UnitAt(u.X, u.Y); occ != nil && occ != u {
				if c, ok := s.nearestFree(u.X, u.Y); ok {
					u.SetMapPlacement(c.X, c.Y, u.Dir)
				}
			}
			n++
		}
		return n
	}
	for _, u := range s.Units {
		if u.Group == group && !u.OnField {
			u.OnField = true
			u.OffY = -56 // 增援進場:從上方滑入(spawn_march)
			if changeCamp {
				u.Camp = camp
			}
			u.Acted = !act // act=true → 可行動;否則標記已行動(下回合才動)
			if occ := s.UnitAt(u.X, u.Y); occ != nil && occ != u {
				if c, ok := s.nearestFree(u.X, u.Y); ok {
					u.SetMapPlacement(c.X, c.Y, u.Dir)
				}
			}
			n++
		}
	}
	return n
}

// nearestFree 是舊重製呈現使用的有界環狀搜尋；它不是原版
// 0x10cc6..0x10d4c 的全圖 row-major Manhattan 規則。
func (s *State) nearestFree(x, y int) (Cell, bool) {
	for r := 1; r < 8; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if dx > -r && dx < r && dy > -r && dy < r {
					continue // 只看環邊
				}
				nx, ny := x+dx, y+dy
				if nx < 0 || ny < 0 || nx >= s.W || ny >= s.H {
					continue
				}
				if s.UnitAt(nx, ny) == nil {
					return Cell{nx, ny}, true
				}
			}
		}
	}
	return Cell{}, false
}
