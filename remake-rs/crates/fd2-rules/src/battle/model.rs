//! Go 來源：`remake/internal/battle/model.go`（1901 行）。
//!
//! Package battle — 炎龍騎士團2 重製的戰棋核心資料模型(M1)。
//!
//! 設計:遊戲差異全在資料(units.json 由 tools/export_units.py 從原版產生),
//! 引擎只認穩定的 JSON。Unit 用 HP/OnField/Acted 投影原版單位狀態。

use super::*;

pub(crate) const NATIVE_TURN_EVENT_CATALOG_SHA256: &str =
    "127c894c523f74905048524d02fdaf142c24610fdb3534694d1ccdfef8dfcd34";

/// Camp 陣營。
///
/// Go：`model.go:22`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Camp(pub i64);

/// 我方(玩家)
/// Go 原式：`iota`
pub const OWN: Camp = Camp(0);

/// 友軍 NPC
pub const ALLY: Camp = Camp(1);

/// 敵方
pub const ENEMY: Camp = Camp(2);

impl fmt::Display for Camp {
    /// Go：`model.go:30`，`func (c Camp) String() string`
    /// Rust 以 `Display` 實作 Go 的 `String()`。
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        f.write_str(match *self {
            OWN => "OWN",
            ALLY => "ALLY",
            _ => "ENEMY",
        })
    }
}

/// Unit 戰場單位。數值來自 EXE 表(doc 03); raw byte fields are optional
/// provenance and normalized fields remain engine projections.
///
/// Go：`model.go:43`
#[derive(Debug, Clone, Default, PartialEq)]
pub struct Unit {
    /// Go 標籤：`json:"enemy_balance_base,omitempty"`
    pub enemy_balance_base: Option<BalanceStats>,
    /// Go 標籤：`json:"enemy_balance_percent,omitempty"`
    pub enemy_balance_percent: i64,
    pub camp: Camp,
    /// 角色名(characters.json;敵方多為職業名)
    pub name: String,
    /// 職業名(中文,M2 TTF 才顯示)
    pub cls_name: String,
    /// 原版職業 table index；商店裝備相容以此對 class_equip_types 判定
    pub class_id: i64,
    pub lv: i64,
    pub hp: i64,
    pub max_hp: i64,
    pub mp: i64,
    pub max_mp: i64,
    pub ap: i64,
    pub dp: i64,
    /// 命中/閃避基礎值(doc02 §2;doc03:EXE 內為「衍生值」非表格原始欄位,
    /// 敵/友單位 10B 表無此欄,export_units.py 暫用固定近似值,見該檔頭註解)
    pub hit: i64,
    /// Go 與 `hit` 同列宣告，註解見該欄位。
    pub ev: i64,
    /// 暴擊率；原版表在 0x5239B(每職業一個 byte，索引為職業 ID 減一)，
    /// 由 resist_crit.json 承載，逐職業比對見 native_combat_tables_test.go。
    /// 原版 sub_29F72 的暴擊門檻還要加上武器 record +10 的加成，那一段尚未接。
    pub crit_pct: i64,
    /// 移動力
    pub mv: i64,
    /// 近戰攻擊距離下限(曼哈頓距離;0 視為預設 1,doc32 weapon_range.json 依武器 type 決定)
    pub atk_min: i64,
    /// 近戰攻擊距離上限(0 視為預設 1;例:騎士槍type3=2,doc32)
    pub atk_max: i64,
    pub portrait: i64,
    /// 地圖 FDICON selector approximation; native source is unit+2.
    pub fig: i64,
    /// MapSelectorSlot is native runtime unit+2 only when HasMapSelectorSlot is
    /// true. It is a process-global FDICON cache slot, never a character identity.
    pub map_selector_slot: i64,
    pub has_map_selector_slot: bool,
    /// MapSelectorKey is the raw byte passed to 0x11019 before it returns the
    /// cache slot above. It is optional because legacy map JSON does not carry
    /// its native source provenance. It must never be inferred from Fig.
    pub map_selector_key: i64,
    pub has_map_selector_key: bool,
    /// NativeMapPresentation is the materialized battle-local runtime
    /// +0/+1/+3/+4 subset. It is valid only after the native selector
    /// construction path succeeds; normalized X/Y/Dir/OffX/OffY are not aliases.
    pub native_map_presentation: NativeMapPresentationState,
    pub has_native_map_presentation: bool,
    /// BattleFig is the separately sourced native unit+7 selector for FIGANI.
    /// FDFIELD roster b1 supplies it; missing older JSON keeps the Fig fallback.
    pub battle_fig: i64,
    pub has_battle_fig: bool,
    /// NativeIdentity is the persistent-record +0x08 key consumed by native
    /// post-handler 0x11506. It is optional provenance and must never be inferred
    /// from Fig, BattleFig, portrait, or map-selector fields.
    pub native_identity: i64,
    pub has_native_identity: bool,
    /// NativeRecordByte8 preserves runtime/persistent record +8 without
    /// assigning a universal identity meaning. Player persistent records use
    /// this byte as NativeIdentity, while scripted runtime construction copies
    /// its visual selector into both +7 and +8.
    pub native_record_byte8: u8,
    pub has_native_record_byte8: bool,
    /// NativePositionRecord preserves the six-byte FDFIELD 3N+2 record paired
    /// with a scripted constructor row. The constructor reads XWord/YWord low
    /// bytes and may relocate them through its nearest-free-cell search.
    pub native_position_record: NativePositionRecord,
    pub has_native_position_record: bool,
    /// NativeRecordRace/Class are the independently proven raw bytes at
    /// persistent/runtime +0x1f/+0x20. They remain separate from normalized
    /// class labels and from the immutable constructor-table provenance because
    /// class change rewrites only +0x20.
    pub native_record_race: u8,
    pub has_native_record_race: bool,
    pub native_record_class: u8,
    pub has_native_record_class: bool,
    /// NativeConstructor preserves the proven EXE static-table provenance for
    /// unit+0x1f/+0x20 without assigning any gameplay meaning to raw bytes.
    /// Go 標籤：`json:"native_constructor,omitempty"`
    pub native_constructor: Option<NativeConstructorTable>,
    /// nativeConstructorPending 標記由 Load 建出、尚未經 0x10C50＋0x1B750 完整
    /// 物化的 FDFIELD 單位；BindNativeFutureItemRows 會補齊並清掉。
    pub(crate) native_constructor_pending: bool,
    pub x: i64,
    pub y: i64,
    /// engine projection; native bit7 is raw provenance only
    pub acted: bool,
    /// 出場波次(原版 FDFIELD b21;事件按 group 放出,doc 25/29)
    pub group: i64,
    /// 是否已登場(事件進場機制:false=待命,尚未出現在戰場,doc 25)
    pub on_field: bool,
    /// normalized/editable spell IDs; not a raw unit+0x22 bitfield
    pub spells: Vec<i64>,
    /// NativeCommandMask is the runtime 40-bit command inventory enumerated by
    /// 0x1c269.  FDFIELD b13..b16 initializes bytes 0..3; byte 4 begins zero and
    /// can be OR-mutated by 0x1d7fb.  It is deliberately separate from Spells:
    /// command ID effects and target contracts are not inferred from this mask.
    /// Go 標籤：`json:"native_command_mask,omitempty"`
    pub native_command_mask: [u8; 5],
    /// NativeTransient is the exact six-byte runtime interval unit+0x22..+0x27.
    /// It is distinct from the legacy normalized Buff*/Poison*/Seal*/Paralyze*
    /// approximation below: native command IDs 17..27 read/write individual raw
    /// bytes, and 0x1A866 decrements them separately once per camp phase.
    /// No gameplay name is inferred for an index from this storage alone.
    /// Go 標籤：`json:"native_transient"`
    pub native_transient: [u8; 6],
    /// NativeRecordByte5/6 preserve the two raw gate bytes consumed by
    /// 0x1A866. Their gameplay meaning is intentionally unknown; neither is
    /// inferred from Acted, OnField, Alive, or Camp.
    /// Go 標籤：`json:"native_record_byte5,omitempty"`
    pub native_record_byte5: u8,
    /// Go 標籤：`json:"has_native_record_byte5,omitempty"`
    pub has_native_record_byte5: bool,
    /// presentation-only HP transition, never serialized
    pub(crate) native_death_flag_before: u8,
    pub(crate) native_death_flag_valid: bool,
    /// Go 標籤：`json:"native_record_byte6,omitempty"`
    pub native_record_byte6: u8,
    /// Go 標籤：`json:"has_native_record_byte6,omitempty"`
    pub has_native_record_byte6: bool,
    /// FDFIELD b17..b19 are copied verbatim by 0x10fb6..0x10fc5 to runtime
    /// +0x34..+0x36. Byte34's low nibble dispatches 0x13a9f, while its high
    /// bits have separate consumers; keep the complete bytes and provenance.
    /// Go 標籤：`json:"native_record_byte34,omitempty"`
    pub native_record_byte34: u8,
    /// Go 標籤：`json:"has_native_record_byte34,omitempty"`
    pub has_native_record_byte34: bool,
    /// Go 標籤：`json:"native_record_byte35,omitempty"`
    pub native_record_byte35: u8,
    /// Go 標籤：`json:"has_native_record_byte35,omitempty"`
    pub has_native_record_byte35: bool,
    /// Go 標籤：`json:"native_record_byte36,omitempty"`
    pub native_record_byte36: u8,
    /// Go 標籤：`json:"has_native_record_byte36,omitempty"`
    pub has_native_record_byte36: bool,
    /// NativeJoinPersistentPending：sub_112A5 建構之後、還沒經過戰後 0x11506 整筆抄回的
    /// 持續記錄。這期間持續槽裡 +0..+4、+0x34..+0x36、+0x3d 仍是建構前的殘值，寫回槽時
    /// 不從 Unit 覆蓋；戰後同步（syncPartyFromBattle）清掉。
    /// Go 標籤：`json:"native_join_persistent_pending,omitempty"`
    pub native_join_persistent_pending: bool,
    /// NativeRecordByte3D is copied from FDFIELD b2 by 0x10fc8. It is distinct
    /// from constructor-table race at runtime +0x1f.
    pub native_record_byte3_d: u8,
    pub has_native_record_byte3_d: bool,
    /// NativeRecordDeathEffect is exact runtime +0x31..+0x33 from b22..b24.
    pub native_record_death_effect: [u8; 3],
    pub has_native_record_death_effect: bool,
    /// These source bytes are not read by 0x10c50; preserve them without names.
    pub native_fdfield_source_byte3: u8,
    pub native_fdfield_source_byte20: u8,
    pub native_fdfield_source_byte25: u8,
    /// NativeRecordWord42 is the optional raw u16 at runtime record +0x42.
    /// ch15_post compares this word directly. When present on a loaded scripted
    /// roster it is also the proven constructor source for initial HP/MaxHP.
    /// Go 標籤：`json:"native_record_word42,omitempty"`
    pub native_record_word42: u16,
    /// Go 標籤：`json:"has_native_record_word42,omitempty"`
    pub has_native_record_word42: bool,
    /// NativeRecordWord46 is the optional constructor-produced max-MP word at
    /// runtime +0x46. It is not inferred from normalized MaxMP.
    /// Go 標籤：`json:"native_record_word46,omitempty"`
    pub native_record_word46: u16,
    /// Go 標籤：`json:"has_native_record_word46,omitempty"`
    pub has_native_record_word46: bool,
    /// 角色物品欄 item IDs；原版 unit+0x0a 起 8×2B
    pub inventory: Vec<i64>,
    /// 與 Inventory 對齊；true 表示該欄位目前已裝備
    pub equipped: Vec<bool>,
    /// 原始 8 個 source bytes；0xff 保留空槽位置
    pub inventory_slots: Vec<i64>,
    /// NativeInventoryFlags is the constructor's raw eight-cell flag view. It is
    /// present only when InventorySlots came from the proven source layout.
    pub native_inventory_flags: Vec<i64>,
    /// Base* are the persistent pre-remake equipment values. Existing scenario
    /// data stores effective values, so EquipmentBaseSet is true for those
    /// records and newly purchased equipment is added without double counting.
    pub base_ap: i64,
    /// Go 與 `base_ap` 同列宣告，註解見該欄位。
    pub base_dp: i64,
    /// Go 與 `base_ap` 同列宣告，註解見該欄位。
    pub base_hit: i64,
    /// Go 與 `base_ap` 同列宣告，註解見該欄位。
    pub base_ev: i64,
    /// Go 與 `base_ap` 同列宣告，註解見該欄位。
    pub base_mv: i64,
    pub base_atk_min: i64,
    pub base_atk_max: i64,
    pub equipment_base_set: bool,
    /// FDFIELD b22 + b23..24；0=item、1=gold，2/3 特殊效果先原值保留
    pub death_effect: Option<DeathEffect>,
    /// 可執行死亡獎勵；type2 已知 handler 由 exporter lower 成 item/gold
    pub death_reward: Option<DeathEffect>,
    /// 朝向:0下 1左 2上 3右(原版 Z2,FDICON 方向幀)
    pub dir: i64,
    /// 行軍/移動的像素位移(顯示用;0=正在格上)
    pub off_x: f64,
    /// 進場時從邊緣滑入,漸減到 0
    pub off_y: f64,
    /// ---- 輔助法術暫時狀態(doc02 §6.4;施放邏輯見 magic.go CastArea/applySpell)----
    /// BuffAPPct／BuffDPPct 服務的是**沒有原版 raw record 的舊可編輯劇本**。
    /// 走原版那條的單位不經過這裡：command 17／18／19 的 handler 直接改 record
    /// 的 +0x48／+0x4A（native_raw_word_step.go），換裝備或狀態到期時由
    /// nativeEquipmentTotals 依 +0x22..+0x24 重算。見
    /// docs/knowledge-base/107-transient-status-semantics-20260910.md。
    /// 魔刃術:AP 加成百分比
    pub buff_ap_pct: i64,
    /// 魔鎧術:DP 加成百分比
    pub buff_dp_pct: i64,
    /// 風行術:HIT 加成
    pub buff_hit: i64,
    /// 風行術:EV 加成
    pub buff_ev: i64,
    /// 上述加成共用剩餘回合數(原版三招各自可疊加回合,重製簡化成單一計時器)
    pub buff_turns: i64,
    /// 封咒術:禁止施法
    pub sealed: bool,
    pub seal_turns: i64,
    /// 毒擊術:每回合扣 MaxHP 的 10%(doc02 §6.4)
    pub poisoned: bool,
    pub poison_turns: i64,
    /// 麻痺術:無法行動(是否擋下行動由呼叫端 UI/AI 檢查此欄位)
    pub paralyzed: bool,
    pub paralyze_turns: i64,
    /// ---- 經驗值/升級(doc02 §4.5/§4.6;doc03 0x43;實作見 growth.go)----
    /// typed +0x3e projection；0x1145a/0x2efb7 以它為 HIT/EV 的共用 base；不等同直接 raw dump。
    /// FDFIELD export 的 generic enemy HIT/EV 仍可能是近似值；evidence-backed
    /// player scenarios必須顯式保存DX，shop preview/class growth不可由derived值猜回。
    /// 尚未實際影響命中/閃避。
    pub dx: i64,
    /// 目前經驗值(滿 100 升級,doc03 0x43「EX 經驗」);用 float64 累加,避免
    /// 攻擊/法術公式算出的小數經驗(如 40/施法者等級)逐次相加時被提早捨去。
    pub exp: f64,
    /// 本單位每級可給出的經驗值(doc02 §4.5「守方每級經驗」;來源 EXE
    pub exp_per_level: i64,
    // 敵/友單位表 EX 欄,docs/data/exe_tables/unit.json,由 export_units.py 依 (race,cls)
    // 帶入 units.json 的 "ex" 欄;舊版(尚未重新匯出的)units.json 無此欄則為 0,
    // 該次攻擊經驗值算出 0,見 growth.go AttackExp 註解)
}

/// NativeConstructorTable is an editable, raw-only view of the constructor's
/// portrait-selected EXE tables. Record is 10 bytes for high_class and 24
/// bytes for lower_class; AuxRecord is the paired 11-byte 0x620a1 record for
/// the lower branch. It is never used as a portrait/class substitute.
///
/// Go：`model.go:227`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeConstructorTable {
    /// Go 標籤：`json:"branch"`
    pub branch: String,
    /// Go 標籤：`json:"index"`
    pub index: i64,
    /// Go 標籤：`json:"record"`
    pub record: Vec<u8>,
    /// Go 標籤：`json:"aux_record,omitempty"`
    pub aux_record: Vec<u8>,
}

impl NativeConstructorTable {
    /// Go：`model.go:234`，`func (t *NativeConstructorTable) validate() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn validate(&self) -> Result<()> {
        todo!("battle/model.go:NativeConstructorTable.validate")
    }
}

/// MaterializeNativeMapSelectorSlots applies 0x11019's first-seen cache rule
/// to one already-proven construction order. Callers must supply exactly the
/// native source order within one battle construction session (player persistent
/// roster followed by scripted FDFIELD spawn groups); this function deliberately does not choose
/// that order or fall back to Fig/Portrait. It makes no partial mutation when
/// a key is absent or invalid.
///
/// Go：`model.go:259`，`func MaterializeNativeMapSelectorSlots(units []*Unit, cache *fdicon.NativeSelectorCache) error`
pub fn materialize_native_map_selector_slots(
    units: &mut [Unit],
    cache: Option<&mut fdicon::NativeSelectorCache>,
) -> Result<()> {
    todo!("battle/model.go:MaterializeNativeMapSelectorSlots")
}

/// initialEquipmentFlags mirrors the original spawn constructor: its first
/// two source inventory bytes are placed with flag 0x40; later bytes start
/// held/unequipped. The editable exports omit 0xff empty bytes, so the
/// compacted list is the available representation here.
///
/// Go：`model.go:295`，`func initialEquipmentFlags(n int) []bool`
pub(crate) fn initial_equipment_flags(n: i64) -> Vec<bool> {
    todo!("battle/model.go:initialEquipmentFlags")
}

/// Go：`model.go:303`，`func materializeInventory(source, compact []int) ([]int, []bool, []int)`
pub(crate) fn materialize_inventory(source: &[i64], compact: &[i64]) -> (Vec<i64>, Vec<bool>, Vec<i64>) {
    todo!("battle/model.go:materializeInventory")
}

impl Unit {
    /// Go：`model.go:334`，`func (u *Unit) normalizeInventorySlots()`
    pub(crate) fn normalize_inventory_slots(&mut self) {
        todo!("battle/model.go:Unit.normalizeInventorySlots")
    }

    /// AddInventoryItem preserves the original fixed eight slot layout while
    /// keeping the compact Inventory view used by editable scripts.
    ///
    /// Go：`model.go:352`，`func (u *Unit) AddInventoryItem(id int, equipped bool) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn add_inventory_item(&mut self, id: i64, equipped: bool) -> bool {
        todo!("battle/model.go:Unit.AddInventoryItem")
    }

    /// Go：`model.go:376`，`func (u *Unit) RemoveInventoryIndex(index int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn remove_inventory_index(&mut self, index: i64) -> bool {
        todo!("battle/model.go:Unit.RemoveInventoryIndex")
    }

    /// Alive is the remake's normalized HP projection. It is intentionally not a
    /// native byte+5 decoder; callers with raw provenance must use the raw field.
    ///
    /// Go：`model.go:406`，`func (u *Unit) Alive() bool`
    pub fn alive(&self) -> bool {
        self.hp > 0
    }

    /// ApplyHPDamage updates the engine HP projection and, when raw provenance is
    /// present, mirrors the native death writer's byte+5 bit0.  It intentionally
    /// does not fabricate raw provenance for legacy/normalized units.
    ///
    /// Go：`model.go:411`，`func (u *Unit) ApplyHPDamage(damage int)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn apply_hp_damage(&mut self, damage: i64) {
        todo!("battle/model.go:Unit.ApplyHPDamage")
    }

    /// Go：`model.go:428`，`func (u *Unit) NativeDeathPresentationFlag() (byte, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_death_presentation_flag(&self) -> (u8, bool) {
        todo!("battle/model.go:Unit.NativeDeathPresentationFlag")
    }

    /// Go：`model.go:435`，`func (u *Unit) ClearNativeDeathPresentationFlag()`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn clear_native_death_presentation_flag(&mut self) {
        todo!("battle/model.go:Unit.ClearNativeDeathPresentationFlag")
    }

    /// RestoreNativeHP mirrors the proven revive writer: restore current HP and
    /// clear only raw byte+5 bit0 when the unit carries native provenance.
    ///
    /// Go：`model.go:443`，`func (u *Unit) RestoreNativeHP()`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn restore_native_hp(&mut self) {
        todo!("battle/model.go:Unit.RestoreNativeHP")
    }

    /// EffectiveAP/EffectiveDP 套用魔刃術/魔鎧術暫時加成後的攻防值。地形% 修正另外在
    /// combat.go AttackWithRNG 套用(取決於單位當下座標,不是固定加成,不適合放在這裡)。
    ///
    /// Go：`model.go:456`，`func (u *Unit) EffectiveAP() int`
    pub fn effective_ap(&self) -> i64 {
        self.ap.wrapping_add(self.ap.wrapping_mul(self.buff_ap_pct) / 100)
    }

    /// Go：`model.go:457`，`func (u *Unit) EffectiveDP() int`
    pub fn effective_dp(&self) -> i64 {
        self.dp.wrapping_add(self.dp.wrapping_mul(self.buff_dp_pct) / 100)
    }

    /// EffectiveHIT/EffectiveEV 套用風行術(HIT+15,EV+15)暫時加成後的命中/閃避值。
    /// 這一層同樣只給舊可編輯劇本用；原版那條在 nativeEquipmentTotals 裡，
    /// unit+0x24 非零時對基礎值加 15，命中與迴避從同一個基礎繼承（107）。
    ///
    /// Go：`model.go:462`，`func (u *Unit) EffectiveHIT() int`
    pub fn effective_hit(&self) -> i64 {
        self.hit.wrapping_add(self.buff_hit)
    }

    /// Go：`model.go:463`，`func (u *Unit) EffectiveEV() int`
    pub fn effective_ev(&self) -> i64 {
        self.ev.wrapping_add(self.buff_ev)
    }

    /// TickStatus 回合結束時呼叫一次:遞減暫時狀態剩餘回合、套用毒擊持續傷害、到期清除。
    /// (doc02 §6.4:毒擊每回合 -10% HP;各項加成/異常 2–4 回合到期消失。)
    ///
    /// Go：`model.go:467`，`func (u *Unit) TickStatus()`
    pub fn tick_status(&mut self) {
        todo!("battle/model.go:Unit.TickStatus")
    }
}

/// State 一場戰鬥的狀態。
///
/// Go：`model.go:505`
#[derive(Debug, Clone, Default)]
pub struct State {
    pub balance: Option<BalanceOptions>,
    pub w: i64,
    pub h: i64,
    /// 所有權：本欄擁有單位本體；其他地方以 `UnitIdx` 指向這個陣列（見 `battle` 模組說明）。
    pub units: Vec<Unit>,
    /// NativeMapSelectorCache is the single process-global raw-key cache accumulated
    /// in native construction order. It is intentionally populated only through
    /// AppendNativeMapSelectorBatch; legacy callers may keep using Units without
    /// claiming an indexed-native selector state.
    pub native_map_selector_cache: Option<fdicon::NativeSelectorCache>,
    /// NativeMapSelectorError records a rejected native construction batch.  The
    /// battle continues with legacy Fig rendering, but NativeMapSpriteKey remains
    /// fail-closed for the entire state rather than mixing native and guessed
    /// selectors after a malformed editable source.
    pub native_map_selector_error: Option<BattleError>,
    /// NativeMapCycleState is the battle-session ownership of native globals
    /// 0x53c0b/0x53c07/0x53c0f. It becomes valid only when the first native
    /// selector construction batch succeeds.
    pub native_map_cycle_state: fdicon::NativeMapSpriteCycleState,
    pub has_native_map_cycle_state: bool,
    /// NativeTerrainPhaseState owns 0x11eee's independent 20-phase terrain
    /// selector and last BIOS tick for this battle session.
    pub native_terrain_phase_state: fdother::NativeTerrainPhaseState,
    pub has_native_terrain_phase_state: bool,
    /// NativeTerrainFlipState and NativeUnitPixelShiftState are the independent
    /// 0x53a40/0x53a00 and 0x53a04/0x53a08 BIOS-word latches.
    pub native_terrain_flip_state: fdicon::NativeBinaryTickState,
    pub native_unit_pixel_shift_state: fdicon::NativeBinaryTickState,
    pub has_native_map_binary_timing_state: bool,
    /// NativeMapHUDState owns the two raw 0x1acf3 display bytes and persistent
    /// anchor. Unlike timing globals, it is not fabricated by selector
    /// construction because gate A can be restored from native save state.
    pub native_map_hud_state: NativeMapHUDRuntimeState,
    pub has_native_map_hud_state: bool,
    pub native_map_view_state: NativeMapViewState,
    pub has_native_map_view_state: bool,
    /// NativeMapRangeMode is raw [0x51a83]. Despite the retained field name it
    /// is an overlay/selection selector, not a bounded GUI range enum:
    /// 0x122dc draws only 1..5 and mutates the field for 6, while 0x115b6 still
    /// consumes values above 6 in target validation. Battle setup briefly uses
    /// zero for its opening frame at 0x10483, then returns with interactive
    /// selector one at 0x1060c.
    pub native_map_range_mode: i64,
    pub has_native_map_range_mode_state: bool,
    /// Roster is the unmaterialized FDFIELD source used by scenarios which
    /// preserve the original constructor semantics. Units is then the canonical
    /// runtime array: party/initial groups are appended in event order, and later
    /// SPAWN calls append their group without reserving slots ahead of time.
    /// 所有權：本欄擁有單位本體；其他地方以 `UnitIdx` 指向這個陣列（見 `battle` 模組說明）。
    pub roster: Vec<Unit>,
    pub pending_groups: BTreeMap<i64, bool>,
    /// nativeFutureItemRows is the immutable, explicitly bounded 0x4e56c row
    /// prefix used by 0x10c50→0x1b750. It is bound by the application layer and
    /// never inferred from normalized item statistics.
    pub(crate) native_future_item_rows: Vec<u8>,
    /// nativeAIScan 是敵方階段 0x1D80B／0x1D8BA／0x1D988 三遍掃描的游標（combat.go）。
    pub(crate) native_ai_scan: NativeAIScanState,
    /// 我方可部署格
    pub own_deploy: Vec<Cell>,
    /// 回合數(無上限,doc 27;只由劇本事件限制)
    pub turn: i64,
    /// NativeRoundCounter preserves executable global [0x53bef], incremented at
    /// the native turn-advance boundary (0x1a5b9), apart from normalized Turn.
    /// Go 標籤：`json:"native_round_counter,omitempty"`
    pub native_round_counter: i64,
    /// 事件旗標(跨事件/跨關劇情狀態,doc 29)
    pub flags: BTreeMap<String, bool>,
    /// raw [0x53ad5] battle-local state table; unnamed indices
    pub native_event_state: [u8; 32],
    /// per-tile 移動成本(len==W*H;index=y*W+x;nil=尚無地形資料,MoveCost 全回 1)
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    pub cost: Option<Vec<i64>>,
    /// immutable FDFIELD composition cell +2 source; each caller rebuilds its own mutable low5/live flags
    pub native_composition_event_bytes: Vec<u8>,
    /// NativeMapEventGrid is the mutable 0x53a51-shaped map buffer used by the
    /// native AI/event helpers.  It retains the four-byte header followed by
    /// one four-byte FDFIELD tile/event word per cell; it is never substituted
    /// for NativeCompositionEventBytes when a caller rebuilds a fresh grid.
    pub native_map_event_grid: Vec<u8>,
    pub has_native_map_event_grid: bool,
    /// NativeFieldControlRaw is the exact live current-save image rooted at
    /// [0x53a55]. It is distinct from composition and from the original chapter
    /// resource because battle handlers rewrite turn/chest bytes.
    pub native_field_control_raw: Vec<u8>,
    pub native_turn_event_controls: [NativeTurnEventControl; 16],
    /// HasNativeTurnEventControlState 只有從精確 map provenance 或 CONTINUE
    /// snapshot 載入全部16筆 raw row 時為真；不代表每筆休眠列都已有 runtime
    /// scenario consumer。
    pub has_native_turn_event_control_state: bool,
    pub native_chest_controls: [NativeChestControl; 16],
    /// HasNativeChestControlState 表示 NativeChestControls 的 16 筆是有出處的
    /// FDFIELD 控制列（0x53+3*slot 的 type／value）：從城鎮進戰場時由地圖的
    /// `chests`（tools/parse_field.py 從同一段 bytes 取出）建立，從 CONTINUE
    /// 進來時由存檔的 raw 控制影像建立。
    pub has_native_chest_control_state: bool,
    pub native_field_unit_controls: Vec<NativeFieldUnitControl>,
    pub has_native_field_control_state: bool,
    /// NativeRuntimeRecords preserves the exact saved current-unit array and
    /// CONTINUE-rebuilt selector slots before a typed Unit projection exists.
    /// Units remains the normalized/gameplay array and is never guessed from
    /// these bytes by State itself.
    pub native_runtime_records: Vec<NativeRuntimeRecordState>,
    /// HasNativeRuntimeUnitProjection is set only after the complete saved
    /// runtime record array has been atomically projected to Units in original
    /// record order. It does not imply that timing or the battle driver is ready.
    pub has_native_runtime_unit_projection: bool,
    /// HasNativePendingGroupBinding 表示 CONTINUE 已把仍待執行、且 live
    /// turn/event 列與可編輯 scenario 完全相符的 FDFIELD rows 深複製到
    /// Roster。它不代表動態改寫排程或 Game controller handoff 已完成。
    pub has_native_pending_group_binding: bool,
    /// row-major -1/0..15；0x13a44 的 1-based low5 已正規化
    pub native_field_event_slots: Vec<i64>,
    /// FDFIELD control 16×2 raw event-id/selector table
    pub native_field_events: Vec<NativeFieldEvent>,
    /// 已由 handler 閉合、仍保留 selector timing 的 editable rules
    pub native_field_event_rules: Vec<NativeFieldEventRule>,
    /// live FDFIELD entry byte+3; exact export admits it, then 0x4dbfc/0x14818 own mutation
    pub native_tile_blit_modes: Vec<u8>,
    /// raw FDSHAP four-byte terrain records; nil unless exact renderer export exists
    pub native_terrain_control: Vec<u8>,
    /// FDSHAP control byte+1 selected by each FDFIELD tile; nil unless the complete exact export validates
    pub native_terrain_move_codes: Vec<u8>,
    /// nativeMovementCostRows is the detached 0x4e555 table used by the
    /// original AI/path helpers. It is bound from the versioned asset export;
    /// normalized Cost must never be substituted for this raw table.
    pub(crate) native_movement_cost_rows: Vec<Vec<u8>>,
    /// scenario-injected spell table; AI command mapping remains data-only
    pub spell_book: Vec<Spell>,
    /// verified raw IDs 0..35; distinct from normalized SpellBook
    pub native_command_book: Vec<NativeCommandRecord>,
    /// verified class raw multiplier; nil means native command effects stay closed
    pub native_command_resistances: BTreeMap<i64, i64>,
    /// growth-row byte10 learn_idx -> native level-up command pairs
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    pub command_learn: Option<BTreeMap<i64, Vec<CommandLearnEntry>>>,
    /// native unit+7 selector -> growth-row byte10 learn_idx
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    pub command_learn_selectors: Option<BTreeMap<i64, i64>>,
    /// native unit+7 selector -> 0x4E4D1 升級成長列
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    pub native_growth_rows: Option<BTreeMap<i64, GrowthRow>>,
    /// NativeGrowthRollObserver 在每一次會消耗原版 RNG 的升級成長擲骰（0x1E54A）之前
    /// 被呼叫，回傳要用的 RNG 狀態。原版的升級訊息等待會依時序吃掉不定數量的亂數，
    /// 對拍重播用它把 eip-trace 的 0x1E54A 收據抄過來；nil 就照目前狀態擲。
    pub native_growth_roll_observer: Option<Callback<dyn Fn(&Unit, u16) -> u16 + Send + Sync>>,
    /// NativeDeathExpCancel 回答擊倒這個單位會不會把本次行動的經驗清零（事件 30 的
    /// [0x53EC8]=0）。由持有劇本的介面層綁定；nil 表示沒有這種死亡程式。
    /// Go 標籤：`json:"-"`
    pub native_death_exp_cancel: Option<Callback<dyn Fn(&Unit) -> bool + Send + Sync>>,
    /// editable item command byte -> spell id; AI ranking remains separate
    pub ai_command_spell: BTreeMap<i64, i64>,
    /// FDFIELD composition 地形旗標+slot 與 control chest table 的 join
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    pub treasures: Option<BTreeMap<Cell, Treasure>>,
    pub native_treasure_event_rules: BTreeMap<i64, NativeTreasureEventRule>,
    /// NativeExperienceAccumulator 投影戰鬥內 [0x53EC8]；滿級／死亡早退保留，
    /// 正常收下才清零。原版不序列化這個暫態全域（58 的 2026-10-01 勘誤）。
    pub native_experience_accumulator: i64,
    /// OpenedTreasure 僅投影 NativeEventState，供既有工具讀取；領取判定只讀
    /// 原生事件表。玩家 0x1924B／0x194F3 與敵方 mode 5 共用 [0x53AD5]。
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    pub opened_treasure: Option<BTreeMap<i64, bool>>,
    // 來源:tools/export_engine_assets.py 依地形控制表(doc01 §5)換算,由 Load 讀同目錄
    // map.json 的 "cost" 陣列自動接上(worklist 第 8 輪「地形屬性接線」)。
}

impl State {
    /// BindNativeFutureItemRows supplies the raw item-effect row prefix required
    /// by future-group construction. A private copy prevents UI lifecycle buffers
    /// or callers from changing an in-progress constructor transaction.
    ///
    /// Go：`model.go:645`，`func (s *State) BindNativeFutureItemRows(rows []byte) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn bind_native_future_item_rows(&mut self, rows: &[u8]) -> Result<()> {
        todo!("battle/model.go:State.BindNativeFutureItemRows")
    }

    /// BindNativeMovementCostRows supplies the versioned 0x4e555 movement table
    /// used by native AI/path helpers. The table is immutable after binding and is
    /// copied so callers cannot mutate an in-progress decision.
    ///
    /// Go：`model.go:676`，`func (s *State) BindNativeMovementCostRows(rows [][]byte) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn bind_native_movement_cost_rows(&mut self, rows: &[Vec<u8>]) -> Result<()> {
        todo!("battle/model.go:State.BindNativeMovementCostRows")
    }
}

/// Cell 格子座標。
///
/// Go：`model.go:692`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct Cell {
    pub x: i64,
    pub y: i64,
}

/// DeathEffect 原樣保存 FDFIELD 單位記錄 b22 與 b23..24 的 u16。Type 0/1 是死亡時掉物/金錢；
/// 2/3 的特殊事件語意尚未完全解明，runtime 在釘死前不得猜測執行。
///
/// Go：`model.go:696`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct DeathEffect {
    /// Go 標籤：`json:"type"`
    pub r#type: i64,
    /// Go 標籤：`json:"value"`
    pub value: i64,
}

/// Treasure 是一個可編輯的戰場寶物節點。Slot 對應 composition word 低5bit與
/// control table 16筆 reward；Hidden 只控制視覺，取得規則相同。
///
/// Go：`model.go:703`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Treasure {
    pub slot: i64,
    pub kind: String,
    pub native_type: u8,
    pub value: i64,
    pub hidden: bool,
}

/// NativeTreasureEventRule 是已由全域事件 handler 閉合的可編輯特殊寶物規則。
/// ItemBySlot 以原始 chest slot 索引；OpenSlots 保存同一事件成功後共同關閉的槽。
///
/// Go：`model.go:713`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeTreasureEventRule {
    /// Go 標籤：`json:"event_id"`
    pub event_id: i64,
    /// Go 標籤：`json:"item_by_slot"`
    pub item_by_slot: Vec<i64>,
    /// Go 標籤：`json:"open_slots"`
    pub open_slots: Vec<i64>,
}

/// NativeFieldEvent 是 FDFIELD 控制段的原始兩位元組格子事件列。
/// 0x13a44 只把 EventID 寫入全域 selector；高階玩法名稱仍由 handler 證據決定。
///
/// Go：`model.go:721`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeFieldEvent {
    /// Go 標籤：`json:"event_id"`
    pub event_id: u8,
    /// Go 標籤：`json:"selector"`
    pub selector: u8,
}

/// NativeTurnEventControl 保存一筆三位元組 FDFIELD live 回合列。
/// Turn 0xff 保留為原始休眠證據，不解讀成第255回合；RawCamp 仍是 selector
/// byte，不推成正規化 battle.Camp。
///
/// Go：`model.go:729`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeTurnEventControl {
    /// Go 標籤：`json:"turn"`
    pub turn: u8,
    /// Go 標籤：`json:"event_id"`
    pub event_id: u8,
    /// Go 標籤：`json:"raw_camp"`
    pub raw_camp: u8,
}

/// NativeChestControl preserves one live three-byte FDFIELD chest row.
///
/// Go：`model.go:736`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeChestControl {
    pub raw_type: u8,
    pub value: u16,
}

/// NativeFieldUnitControl preserves one live 26-byte FDFIELD constructor row.
/// It is not a current runtime unit; 0x10b4e consumes matching rows only when a
/// future group is appended through 0x10c50.
///
/// Go：`model.go:744`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeFieldUnitControl {
    pub raw: [u8; 26],
}

/// NativePositionRecord is one exact FDFIELD 3N+2 six-byte record.
///
/// Go：`model.go:749`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativePositionRecord {
    pub x_word: u16,
    pub y_word: u16,
    pub raw_key: u16,
}

/// NativeRuntimeRecordState is one exact saved 0x50-byte current-unit record
/// plus the selector result rebuilt from raw +7 in original list order.
///
/// Go：`model.go:755`
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct NativeRuntimeRecordState {
    pub raw: [u8; 80],
    pub selector_key: u8,
    pub selector_slot: u8,
}

impl Default for NativeRuntimeRecordState {
    fn default() -> Self {
        Self {
            raw: std::array::from_fn(|_| Default::default()),
            selector_key: Default::default(),
            selector_slot: Default::default(),
        }
    }
}

/// Go：`model.go:761`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeFieldModeRange {
    /// Go 標籤：`json:"start"`
    pub start: i64,
    /// Go 標籤：`json:"end"`
    pub end: i64,
    /// Go 標籤：`json:"mode"`
    pub mode: u8,
}

/// Go：`model.go:767`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeFieldTextIndices {
    /// Go 標籤：`json:"missing_item"`
    pub missing_item: i64,
    /// Go 標籤：`json:"success"`
    pub success: i64,
    /// Go 標籤：`json:"final"`
    pub r#final: i64,
}

/// Go：`model.go:773`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeFieldPresentation {
    /// Go 標籤：`json:"archive"`
    pub archive: String,
    /// Go 標籤：`json:"resource"`
    pub resource: i64,
    /// Go 標籤：`json:"frames"`
    pub frames: i64,
    /// Go 標籤：`json:"helper"`
    pub helper: String,
    /// Go 標籤：`json:"destination_offset"`
    pub destination_offset: i64,
    /// Go 標籤：`json:"stride"`
    pub stride: i64,
    /// Go 標籤：`json:"transparent"`
    pub transparent: i64,
    /// Go 標籤：`json:"delay_helper"`
    pub delay_helper: String,
    /// Go 標籤：`json:"delay_ticks"`
    pub delay_ticks: i64,
}

/// NativeTurnActivation 描述一個已證實的格子 handler 對 live FDFIELD 回合列
/// 的改寫。它辨識改寫前的原始列，不是正規化 scenario 排程，也不把 RawCamp
/// 解讀成 battle.Camp。
///
/// Go：`model.go:788`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeTurnActivation {
    /// Go 標籤：`json:"slot"`
    pub slot: i64,
    /// Go 標籤：`json:"event_id"`
    pub event_id: i64,
    /// Go 標籤：`json:"raw_camp"`
    pub raw_camp: u8,
    /// Go 標籤：`json:"turn_delta"`
    pub turn_delta: i64,
}

/// NativeEventStateWrite 保存 [0x53AD5] 指向之 0x20-byte table 的單一 byte write。
/// Index 刻意保留 raw 形式，其語意只能由所屬 field-event rule 的 caller provenance
/// 決定。
///
/// Go：`model.go:798`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeEventStateWrite {
    /// Go 標籤：`json:"index"`
    pub index: i64,
    /// Go 標籤：`json:"value"`
    pub value: u8,
}

/// NativeFieldTurnChain 是啟動 dormant live turn rows 的可編輯 field handler。
/// 它不替 event-state bytes 或 RawCamp selectors 猜測玩法名稱。
///
/// Go：`model.go:805`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeFieldTurnChain {
    /// Go 標籤：`json:"handler"`
    pub handler: String,
    /// Go 標籤：`json:"trigger_record_byte8"`
    pub trigger_record_byte8: u8,
    /// Go 標籤：`json:"mismatch_text_index"`
    pub mismatch_text_index: i64,
    /// Go 標籤：`json:"success_text_index"`
    pub success_text_index: i64,
    /// Go 標籤：`json:"state_writes"`
    pub state_writes: Vec<NativeEventStateWrite>,
    /// Go 標籤：`json:"turn_activations"`
    pub turn_activations: Vec<NativeTurnActivation>,
}

/// NativeFieldEventRule 保存已閉合 handler 的資料，不自行決定 selector 的呼叫時機。
///
/// Go：`model.go:815`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeFieldEventRule {
    /// Go 標籤：`json:"event_id"`
    pub event_id: i64,
    /// Go 標籤：`json:"handler,omitempty"`
    pub handler: String,
    /// Go 標籤：`json:"selector"`
    pub selector: u8,
    /// Go 標籤：`json:"trigger_gate,omitempty"`
    pub trigger_gate: String,
    /// Go 標籤：`json:"set_mode_ranges,omitempty"`
    pub set_mode_ranges: Vec<NativeFieldModeRange>,
    /// Go 標籤：`json:"set_state_index,omitempty"`
    pub set_state_index: Option<i64>,
    /// Go 標籤：`json:"set_state_value,omitempty"`
    pub set_state_value: Option<i64>,
    /// Go 標籤：`json:"once_state_index,omitempty"`
    pub once_state: Option<i64>,
    /// Go 標籤：`json:"required_nonzero_state_index,omitempty"`
    pub required_nonzero_state: Option<i64>,
    /// Go 標籤：`json:"required_item,omitempty"`
    pub required_item: Option<i64>,
    /// Go 標籤：`json:"consume_item,omitempty"`
    pub consume_item: bool,
    /// Go 標籤：`json:"spawn_group,omitempty"`
    pub spawn_group: Option<i64>,
    /// Go 標籤：`json:"join_character,omitempty"`
    pub join_character: Option<i64>,
    /// Go 標籤：`json:"text_indices,omitempty"`
    pub text_indices: Option<NativeFieldTextIndices>,
    /// Go 標籤：`json:"presentation,omitempty"`
    pub presentation: Option<NativeFieldPresentation>,
    /// Go 標籤：`json:"turn_activation,omitempty"`
    pub turn_activation: Option<NativeTurnActivation>,
    /// Go 標籤：`json:"turn_chain,omitempty"`
    pub turn_chain: Option<NativeFieldTurnChain>,
}

impl State {
    /// TreasureAt 查詢尚未取得的寶物格。
    ///
    /// Go：`model.go:836`，`func (s *State) TreasureAt(x, y int) (Treasure, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn treasure_at(&self, x: i64, y: i64) -> (Treasure, bool) {
        todo!("battle/model.go:State.TreasureAt")
    }

    /// ClaimTreasure 投影原版 0x190ac：只有站在該格的 active unit 可取；物品放進
    /// 該單位8格 inventory，滿背包時不開箱；金錢由 caller 加到 campaign bank。
    /// 原版沒有 camp 限制，因此 Enemy 也能取得並標記 opened。
    ///
    /// Go：`model.go:847`，`func (s *State) ClaimTreasure(u *Unit, x, y int) (Treasure, bool)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn claim_treasure(&mut self, u: Option<UnitIdx>, x: i64, y: i64) -> (Treasure, bool) {
        self.claim_treasure_inner(u, x, y, true)
    }

    /// ClaimTreasureBeforeClosing 對應物品 YES：取得物品並設事件狀態，0x19268
    /// 的地圖更新留到關框後由 CompleteTreasureMapUpdate 執行。
    ///
    /// Go：`model.go:853`，`func (s *State) ClaimTreasureBeforeClosing(u *Unit, x, y int) (Treasure, bool)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub fn claim_treasure_before_closing(&mut self, u: Option<UnitIdx>, x: i64, y: i64) -> (Treasure, bool) {
        self.claim_treasure_inner(u, x, y, false)
    }

    /// Go：`model.go:857`，`func (s *State) claimTreasure(u *Unit, x, y int, updateMap bool) (Treasure, bool)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn claim_treasure_inner(
        &mut self,
        u: Option<UnitIdx>,
        x: i64,
        y: i64,
        update_map: bool,
    ) -> (Treasure, bool) {
        todo!("battle/model.go:State.claimTreasure")
    }

    /// Go：`model.go:894`，`func (s *State) markTreasureEvent(slot int)`
    pub(crate) fn mark_treasure_event(&mut self, slot: i64) {
        todo!("battle/model.go:State.markTreasureEvent")
    }

    /// CompleteTreasureMapUpdate 共用玩家 0x19268 與敵方 mode 5 的 0x12263。
    /// 沒有原生緩衝的可編輯測試場景不需更新；已材料化但損壞的緩衝拒絕寫入。
    ///
    /// Go：`model.go:904`，`func (s *State) CompleteTreasureMapUpdate() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn complete_treasure_map_update(&mut self) -> bool {
        todo!("battle/model.go:State.CompleteTreasureMapUpdate")
    }

    /// Go：`model.go:914`，`func (s *State) claimNativeTreasureEvent(u *Unit, t Treasure) (Treasure, bool)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn claim_native_treasure_event(&mut self, u: UnitIdx, t: Treasure) -> (Treasure, bool) {
        todo!("battle/model.go:State.claimNativeTreasureEvent")
    }

    /// OnFieldUnit 回傳該格上「已登場且存活」的單位(無則 nil)。
    ///
    /// Go：`model.go:942`，`func (s *State) UnitAt(x, y int) *Unit`
    /// 所有權：回傳的單位是 `self.units` 的索引。
    pub fn unit_at(&self, x: i64, y: i64) -> Option<UnitIdx> {
        todo!("battle/model.go:State.UnitAt")
    }

    /// AliveCount 各陣營「已登場且存活」數(用於勝敗判定)。
    /// 注意:待命(未登場)單位不計入 → 敵方援軍未出時不會誤判全滅。
    ///
    /// Go：`model.go:953`，`func (s *State) AliveCount(c Camp) int`
    pub fn alive_count(&self, c: Camp) -> i64 {
        todo!("battle/model.go:State.AliveCount")
    }

    /// PendingCount 某陣營尚未登場(待命)的單位數。只供戰前資訊與測試觀測；勝負判定
    /// （Result）照原版 0x205b4 只看已登場的記錄，不等待命援軍。
    ///
    /// 原始記錄 +5 bit0 是「這一列不在場上」——寫入端與消費端都用它篩掉不啟用的列
    /// （占位格網 0x145CD 的 `record[5]&1`、AI 目標掃描 0x12C60 同一個判準）。那種列
    /// 不是「還沒登場的援軍」，是 FDFIELD 裡本來就不啟用的資料：第 1 章 group1+group2
    /// 共八列，其中一列帶 bit0，所以原版戰前資訊顯示的是 ENEMY·07（收據見
    /// docs/knowledge-base/46-ch1-opening-timeline.md §405–410s）。把它算成待命，敵方
    /// 全滅之後永遠等不到勝利。
    ///
    /// Go：`model.go:972`，`func (s *State) PendingCount(c Camp) int`
    pub fn pending_count(&self, c: Camp) -> i64 {
        todo!("battle/model.go:State.PendingCount")
    }

    // ---- 載入(units.json,tools/export_units.py 產生)----
}

/// Go：`model.go:992`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct UnitsFile {
    /// Go 標籤：`json:"map"`
    pub map: i64,
    /// Go 標籤：`json:"w"`
    pub w: i64,
    /// Go 標籤：`json:"h"`
    pub h: i64,
    /// Go 標籤：`json:"own_deploy"`
    pub own_deploy: Vec<Cell>,
    /// Optional raw global snapshot; absent legacy/editable files must not
    /// fabricate provenance for native round predicates.
    /// Go 標籤：`json:"native_round_counter,omitempty"`
    pub native_round_counter: Option<i64>,
    /// Go 標籤：`json:"chests,omitempty"`
    pub chests: Vec<UnitsFileChest>,
    /// Go 標籤：`json:"native_treasure_event_rules,omitempty"`
    pub native_treasure_event_rules: Vec<NativeTreasureEventRule>,
    /// Go 標籤：`json:"units"`
    pub units: Vec<UnitsFileUnit>,
}

/// Go：`model.go:1064`，`func campFrom(s string) Camp`
pub(crate) fn camp_from(s: &str) -> Camp {
    todo!("battle/model.go:campFrom")
}

/// Load 從 units.json 建出戰鬥初始狀態。我方(own)依序放到部署格。
///
/// Go：`model.go:1076`，`func Load(path string) (*State, error)`
pub fn load(path: &str) -> Result<State> {
    todo!("battle/model.go:Load")
}

/// mapCostFile map.json 裡跟地形成本相關的欄位(其餘欄位 main.go 的 MapData 自己讀,這裡只挑 cost 用)。
///
/// Go：`model.go:1288`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct MapCostFile {
    /// Go 標籤：`json:"w"`
    pub w: i64,
    /// Go 標籤：`json:"h"`
    pub h: i64,
    /// Go 標籤：`json:"cost"`
    pub cost: Vec<i64>,
    /// Go 標籤：`json:"native_composition_event_bytes"`
    pub native_composition_event_bytes: Vec<u8>,
    /// Go 標籤：`json:"native_field_event_slots"`
    pub native_field_event_slots: Vec<i64>,
    /// Go 標籤：`json:"native_field_events"`
    pub native_field_events: Vec<NativeFieldEvent>,
    /// Go 標籤：`json:"native_field_event_rules"`
    pub native_field_event_rules: Vec<NativeFieldEventRule>,
    /// Go 標籤：`json:"native_tile_blit_modes"`
    pub native_tile_blit_modes: Vec<u8>,
    /// Go 標籤：`json:"native_terrain_control"`
    pub native_terrain_control: Vec<u8>,
    /// Go 標籤：`json:"tiles"`
    pub tiles: Vec<i64>,
    /// Go 標籤：`json:"treasure_slots"`
    pub treasure_slots: Vec<i64>,
    /// Go 標籤：`json:"treasure_hidden"`
    pub treasure_hidden: Vec<bool>,
}

/// loadNativeTurnEventControls 只接受完整16筆 raw table；0xff 列保持可見，
/// 不提升成第255回合的 scenario event。
///
/// Go：`model.go:1305`，`func loadNativeTurnEventControls( mapJSONPath string, w, h int, ) ([16]NativeTurnEventControl, int, bool)`
pub(crate) fn load_native_turn_event_controls(
    map_json_path: &str,
    w: i64,
    h: i64,
) -> ([NativeTurnEventControl; 16], i64, bool) {
    todo!("battle/model.go:loadNativeTurnEventControls")
}

/// Go：`model.go:1382`，`func loadTreasures(mapJSONPath string, w, h int, chests []struct { Slot int 'json:"slot"' Kind string 'json:"type"' NativeType byte 'json:"native_type"' Value int 'json:"value"' }) map[Cell]Treasure`
pub(crate) fn load_treasures(
    map_json_path: &str,
    w: i64,
    h: i64,
    chests: &[UnitsFileChest],
) -> BTreeMap<Cell, Treasure> {
    todo!("battle/model.go:loadTreasures")
}

/// loadTerrainCost 嘗試讀 units.json 同目錄的 map.json,取其 "cost" 陣列(tools/
/// export_engine_assets.py 產生;doc01 §5 地形控制表換算)。檔案不存在、沒有 cost 欄位、
/// 或尺寸對不上 units.json 的 w/h,一律回 nil(MoveCost 退回全平地=1,不 fail Load)。
///
/// Go：`model.go:1422`，`func loadTerrainCost(mapJSONPath string, w, h int) []int`
pub(crate) fn load_terrain_cost(map_json_path: &str, w: i64, h: i64) -> Vec<i64> {
    todo!("battle/model.go:loadTerrainCost")
}

/// loadNativeCompositionEventBytes accepts only an exact FDFIELD composition
/// export. These bytes are immutable source provenance, not the live +2 flags:
/// 0x4dbfc first masks each byte with 0x1f and caller-specific writers may then
/// add 0x40/0x80.
///
/// Go：`model.go:1441`，`func loadNativeCompositionEventBytes(mapJSONPath string, w, h int) []byte`
pub(crate) fn load_native_composition_event_bytes(map_json_path: &str, w: i64, h: i64) -> Vec<u8> {
    todo!("battle/model.go:loadNativeCompositionEventBytes")
}

/// loadNativeMapEventGrid rebuilds the exact four-byte cell prefix consumed by
/// 0x12e38/0x15df3 from the same FDFIELD export that produced map.json.  The
/// first four bytes are the native width/height header; each cell then keeps
/// tile word (+0/+1) and event word (+2/+3).  Callers still own the mutable
/// 0x4dbfc reset and later event writers.
///
/// Go：`model.go:1459`，`func loadNativeMapEventGrid(mapJSONPath string, w, h int) ([]byte, bool)`
pub(crate) fn load_native_map_event_grid(map_json_path: &str, w: i64, h: i64) -> (Vec<u8>, bool) {
    todo!("battle/model.go:loadNativeMapEventGrid")
}

/// resetNativeMapEventGrid reproduces 0x4dbfc's constructor prefix.  It is
/// intentionally in-place because the original buffer is subsequently owned
/// by event/AI writers; malformed dimensions are rejected by the loader.
///
/// Go：`model.go:1494`，`func resetNativeMapEventGrid(grid []byte)`
pub(crate) fn reset_native_map_event_grid(grid: &mut [u8]) {
    todo!("battle/model.go:resetNativeMapEventGrid")
}

impl State {
    /// ResetNativeMapEventGrid applies 0x4DBFC atomically to the materialized raw
    /// four-byte cell buffer. It is exported only for the evidence-bound ch22
    /// post-battle resource transaction; callers cannot supply normalized cells.
    ///
    /// Go：`model.go:1513`，`func (s *State) ResetNativeMapEventGrid() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn reset_native_map_event_grid(&mut self) -> bool {
        todo!("battle/model.go:State.ResetNativeMapEventGrid")
    }
}

/// Go：`model.go:1525`，`func loadNativeFieldEvents( mapJSONPath string, w, h int, ) ([]int, []NativeFieldEvent, []NativeFieldEventRule)`
pub(crate) fn load_native_field_events(
    map_json_path: &str,
    w: i64,
    h: i64,
) -> (Vec<i64>, Vec<NativeFieldEvent>, Vec<NativeFieldEventRule>) {
    todo!("battle/model.go:loadNativeFieldEvents")
}

/// loadNativeTerrainRendererInputs accepts only a complete map export. It
/// deliberately keeps malformed data nil rather than repurposing Cost.
///
/// Go：`model.go:1581`，`func loadNativeTerrainRendererInputs(mapJSONPath string, w, h int) ([]byte, []byte, []byte)`
pub(crate) fn load_native_terrain_renderer_inputs(map_json_path: &str, w: i64, h: i64) -> (Vec<u8>, Vec<u8>, Vec<u8>) {
    todo!("battle/model.go:loadNativeTerrainRendererInputs")
}

impl State {
    /// AddUnit 把一個單位加入戰場(事件 spawn / 主角隊進場用)。 Legacy callers
    /// do not materialize a native selector slot implicitly.
    ///
    /// Go：`model.go:1604`，`func (s *State) AddUnit(u *Unit)`
    /// 所有權：`u` 的單位本體移入 State（Go 把指標 append 進 `Units`／`Roster`）。
    pub fn add_unit(&mut self, u: Unit) {
        todo!("battle/model.go:State.AddUnit")
    }

    /// AppendNativeMapSelectorBatch atomically appends one proven construction
    /// batch and assigns its unit+2 slots using the State's process-global cache.
    /// Callers must preserve the native order (party first, then scripted groups)
    /// and must not call this for legacy units without explicit raw keys. A failed
    /// batch changes neither Units nor the cache.
    ///
    /// Go：`model.go:1611`，`func (s *State) AppendNativeMapSelectorBatch(units []*Unit) error`
    /// 所有權：`units` 的單位本體移入 State（Go 把指標 append 進 `Units`／`Roster`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn append_native_map_selector_batch(&mut self, units: Vec<Unit>) -> Result<()> {
        todo!("battle/model.go:State.AppendNativeMapSelectorBatch")
    }

    /// AdvanceNativeMapPresentationCycles applies one proven 0x1297d call to the
    /// battle-local raw globals. The signed timer value must already be the low
    /// BIOS word observed by the caller; legacy states fail closed.
    ///
    /// Go：`model.go:1643`，`func (s *State) AdvanceNativeMapPresentationCycles(rawTimerTick int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_map_presentation_cycles(&mut self, raw_timer_tick: i64) -> bool {
        todo!("battle/model.go:State.AdvanceNativeMapPresentationCycles")
    }

    /// AdvanceNativeTerrainPhase applies one proven 0x11eee phase-selection call.
    /// override is raw [0x51a93]: -1 for the BIOS-timed path or 0..19 for the
    /// explicit selector path.
    ///
    /// Go：`model.go:1654`，`func (s *State) AdvanceNativeTerrainPhase(rawTimerTick, override int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_terrain_phase(&mut self, raw_timer_tick: i64, r#override: i64) -> bool {
        todo!("battle/model.go:State.AdvanceNativeTerrainPhase")
    }

    /// Go：`model.go:1666`，`func (s *State) AdvanceNativeTerrainFlip(rawTimerTick int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_terrain_flip(&mut self, raw_timer_tick: i64) -> bool {
        todo!("battle/model.go:State.AdvanceNativeTerrainFlip")
    }

    /// Go：`model.go:1678`，`func (s *State) AdvanceNativeUnitPixelShift(rawTimerTick int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_unit_pixel_shift(&mut self, raw_timer_tick: i64) -> bool {
        todo!("battle/model.go:State.AdvanceNativeUnitPixelShift")
    }

    /// AppendNativeMapSelectorBatchOrLegacy attempts the evidenced constructor
    /// contract, but preserves playable legacy construction if editable source data
    /// is malformed.  The failure is retained on State so renderers cannot quietly
    /// claim a partial native selector state.
    ///
    /// Go：`model.go:1694`，`func (s *State) AppendNativeMapSelectorBatchOrLegacy(units []*Unit)`
    /// 所有權：`units` 的單位本體移入 State（Go 把指標 append 進 `Units`／`Roster`）。
    pub fn append_native_map_selector_batch_or_legacy(&mut self, units: Vec<Unit>) {
        todo!("battle/model.go:State.AppendNativeMapSelectorBatchOrLegacy")
    }

    /// NativeMapSpriteKey resolves an explicitly materialized unit+2 slot back to
    /// its FDICON raw key.  It returns false unless the whole State remains in the
    /// proven native construction mode; callers must then retain their legacy Fig
    /// presentation rather than guessing a raw selector.
    ///
    /// Go：`model.go:1705`，`func (s *State) NativeMapSpriteKey(u *Unit) (int, bool)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_map_sprite_key(&self, u: Option<UnitIdx>) -> (i64, bool) {
        todo!("battle/model.go:State.NativeMapSpriteKey")
    }

    /// AppendGroup is the normalized compatibility append used by existing event
    /// playback. It preserves FDFIELD row order and selector-cache order, but it
    /// does not execute 0x10c50's table projection, [0x53afa] placement branch,
    /// inventory initialization, or 0x1b750 recomputation. Callers must not cite
    /// this helper as the native constructor.
    ///
    /// Go：`model.go:1722`，`func (s *State) AppendGroup(group int) int`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn append_group(&mut self, group: i64) -> i64 {
        todo!("battle/model.go:State.AppendGroup")
    }

    /// AppendGroupWithNativePlacement atomically applies the proven
    /// 0x10B4E→0x10C50 transaction represented by the typed remake: row order,
    /// per-call [0x53AFA] placement, table-derived base fields, constructor
    /// inventory cells, 0x1B750 effective-stat recomputation and 0x11019 selector
    /// allocation. Fields without a typed consumer remain raw provenance; this is
    /// not a claim of byte-identical 0x50-byte record storage.
    ///
    /// Go：`model.go:1755`，`func (s *State) AppendGroupWithNativePlacement(group int, rawGate byte) (int, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn append_group_with_native_placement(&mut self, group: i64, raw_gate: u8) -> Result<i64> {
        todo!("battle/model.go:State.AppendGroupWithNativePlacement")
    }

    /// SpawnGroup 讓既有重製劇本的 group 登場，可另行覆寫陣營與本回合行動狀態。
    /// 它保留舊的滑入與環狀錯位呈現，不等同原版 0x10b4e/0x10c50 constructor。
    ///
    /// Go：`model.go:1843`，`func (s *State) SpawnGroup(group int, camp Camp, changeCamp, act bool) int`
    pub fn spawn_group(&mut self, group: i64, camp: Camp, change_camp: bool, act: bool) -> i64 {
        todo!("battle/model.go:State.SpawnGroup")
    }

    /// nearestFree 是舊重製呈現使用的有界環狀搜尋；它不是原版
    /// 0x10cc6..0x10d4c 的全圖 row-major Manhattan 規則。
    ///
    /// Go：`model.go:1883`，`func (s *State) nearestFree(x, y int) (Cell, bool)`
    pub(crate) fn nearest_free(&self, x: i64, y: i64) -> (Cell, bool) {
        todo!("battle/model.go:State.nearestFree")
    }
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct UnitsFileChest {
    /// Go 標籤：`json:"slot"`
    pub slot: i64,
    /// Go 標籤：`json:"type"`
    pub kind: String,
    /// Go 標籤：`json:"native_type"`
    pub native_type: u8,
    /// Go 標籤：`json:"value"`
    pub value: i64,
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct UnitsFileUnit {
    /// Go 標籤：`json:"camp"`
    pub camp: String,
    /// Go 標籤：`json:"cls"`
    pub class_id: i64,
    /// Go 標籤：`json:"name"`
    pub name: String,
    /// Go 標籤：`json:"cls_name"`
    pub cls_name: String,
    /// Go 標籤：`json:"lv"`
    pub lv: i64,
    /// Go 標籤：`json:"hp"`
    pub hp: i64,
    /// Go 標籤：`json:"mp"`
    pub mp: i64,
    /// Go 標籤：`json:"spells"`
    pub spells: Vec<i64>,
    /// Go 標籤：`json:"initial_command_mask,omitempty"`
    pub initial_command_mask: Vec<u8>,
    /// Go 標籤：`json:"inventory,omitempty"`
    pub inventory: Vec<i64>,
    /// Go 標籤：`json:"inventory_slots,omitempty"`
    pub inventory_slots: Vec<i64>,
    /// Go 標籤：`json:"death_effect,omitempty"`
    pub death_effect: Option<DeathEffect>,
    /// Go 標籤：`json:"death_reward,omitempty"`
    pub death_reward: Option<DeathEffect>,
    /// Go 標籤：`json:"ap"`
    pub ap: i64,
    /// Go 標籤：`json:"dp"`
    pub dp: i64,
    /// Go 標籤：`json:"hit"`
    pub hit: i64,
    /// Go 標籤：`json:"ev"`
    pub ev: i64,
    /// Go 標籤：`json:"crit"`
    pub crit: i64,
    /// Go 標籤：`json:"mv"`
    pub mv: i64,
    /// Go 標籤：`json:"atk_min"`
    pub atk_min: i64,
    /// Go 標籤：`json:"atk_max"`
    pub atk_max: i64,
    /// Go 標籤：`json:"ex"`
    pub ex: i64,
    /// Go 標籤：`json:"portrait"`
    pub portrait: i64,
    /// Go 標籤：`json:"fig"`
    pub fig: i64,
    /// Go 標籤：`json:"battle_fig,omitempty"`
    pub battle_fig: Option<i64>,
    /// Go 標籤：`json:"native_identity,omitempty"`
    pub native_identity: Option<i64>,
    /// Go 標籤：`json:"native_record_byte8,omitempty"`
    pub native_record_byte8: Option<u8>,
    /// Go 標籤：`json:"native_position_record,omitempty"`
    pub native_position_record: Option<UnitsFileUnitNativePositionRecord>,
    /// Go 標籤：`json:"native_record_race,omitempty"`
    pub native_record_race: Option<u8>,
    /// Go 標籤：`json:"native_record_class,omitempty"`
    pub native_record_class: Option<u8>,
    /// Go 標籤：`json:"map_selector_slot,omitempty"`
    pub map_selector_slot: Option<i64>,
    /// Go 標籤：`json:"map_selector_key,omitempty"`
    pub map_selector_key: Option<i64>,
    /// Go 標籤：`json:"native_record_byte5,omitempty"`
    pub native_record_byte5: Option<u8>,
    /// Go 標籤：`json:"native_record_byte6,omitempty"`
    pub native_record_byte6: Option<u8>,
    /// Go 標籤：`json:"native_record_byte34,omitempty"`
    pub native_record_byte34: Option<u8>,
    /// Go 標籤：`json:"native_record_byte35,omitempty"`
    pub native_record_byte35: Option<u8>,
    /// Go 標籤：`json:"native_record_byte36,omitempty"`
    pub native_record_byte36: Option<u8>,
    /// Go 標籤：`json:"native_record_byte3d,omitempty"`
    pub native_record_byte3_d: Option<u8>,
    /// Go 標籤：`json:"native_record_death_effect,omitempty"`
    pub native_record_death_effect: Vec<u8>,
    /// Go 標籤：`json:"native_source_byte3,omitempty"`
    pub native_source_byte3: Option<u8>,
    /// Go 標籤：`json:"native_source_byte20,omitempty"`
    pub native_source_byte20: Option<u8>,
    /// Go 標籤：`json:"native_source_byte25,omitempty"`
    pub native_source_byte25: Option<u8>,
    /// Go 標籤：`json:"native_record_word42,omitempty"`
    pub native_record_word42: Option<u16>,
    /// Go 標籤：`json:"native_record_word46,omitempty"`
    pub native_record_word46: Option<u16>,
    /// Go 標籤：`json:"group"`
    pub group: i64,
    /// Go 標籤：`json:"native_constructor,omitempty"`
    pub native_constructor: Option<NativeConstructorTable>,
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct UnitsFileUnitNativePositionRecord {
    /// Go 標籤：`json:"x_word"`
    pub x_word: u16,
    /// Go 標籤：`json:"y_word"`
    pub y_word: u16,
    /// Go 標籤：`json:"raw_key"`
    pub raw_key: u16,
}
