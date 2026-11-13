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
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash, serde::Serialize, serde::Deserialize)]
#[serde(transparent)]
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
#[derive(Debug, Clone, Default, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Unit {
    /// Go 標籤：`json:"enemy_balance_base,omitempty"`
    #[serde(rename = "enemy_balance_base", with = "go_balance_stats", skip_serializing_if = "Option::is_none")]
    pub enemy_balance_base: Option<BalanceStats>,
    /// Go 標籤：`json:"enemy_balance_percent,omitempty"`
    #[serde(rename = "enemy_balance_percent", skip_serializing_if = "go_is_zero")]
    pub enemy_balance_percent: i64,
    #[serde(rename = "Camp")]
    pub camp: Camp,
    /// 角色名(characters.json;敵方多為職業名)
    #[serde(rename = "Name", deserialize_with = "null_as_default")]
    pub name: String,
    /// 職業名(中文,M2 TTF 才顯示)
    #[serde(rename = "ClsName", deserialize_with = "null_as_default")]
    pub cls_name: String,
    /// 原版職業 table index；商店裝備相容以此對 class_equip_types 判定
    #[serde(rename = "ClassID")]
    pub class_id: i64,
    #[serde(rename = "Lv")]
    pub lv: i64,
    #[serde(rename = "HP")]
    pub hp: i64,
    #[serde(rename = "MaxHP")]
    pub max_hp: i64,
    #[serde(rename = "MP")]
    pub mp: i64,
    #[serde(rename = "MaxMP")]
    pub max_mp: i64,
    #[serde(rename = "AP")]
    pub ap: i64,
    #[serde(rename = "DP")]
    pub dp: i64,
    /// 命中/閃避基礎值(doc02 §2;doc03:EXE 內為「衍生值」非表格原始欄位,
    /// 敵/友單位 10B 表無此欄,export_units.py 暫用固定近似值,見該檔頭註解)
    #[serde(rename = "HIT")]
    pub hit: i64,
    /// Go 與 `hit` 同列宣告，註解見該欄位。
    #[serde(rename = "EV")]
    pub ev: i64,
    /// 暴擊率；原版表在 0x5239B(每職業一個 byte，索引為職業 ID 減一)，
    /// 由 resist_crit.json 承載，逐職業比對見 native_combat_tables_test.go。
    /// 原版 sub_29F72 的暴擊門檻還要加上武器 record +10 的加成，那一段尚未接。
    #[serde(rename = "CritPct")]
    pub crit_pct: i64,
    /// 移動力
    #[serde(rename = "MV")]
    pub mv: i64,
    /// 近戰攻擊距離下限(曼哈頓距離;0 視為預設 1,doc32 weapon_range.json 依武器 type 決定)
    #[serde(rename = "AtkMin")]
    pub atk_min: i64,
    /// 近戰攻擊距離上限(0 視為預設 1;例:騎士槍type3=2,doc32)
    #[serde(rename = "AtkMax")]
    pub atk_max: i64,
    #[serde(rename = "Portrait")]
    pub portrait: i64,
    /// 地圖 FDICON selector approximation; native source is unit+2.
    #[serde(rename = "Fig")]
    pub fig: i64,
    /// MapSelectorSlot is native runtime unit+2 only when HasMapSelectorSlot is
    /// true. It is a process-global FDICON cache slot, never a character identity.
    #[serde(rename = "MapSelectorSlot")]
    pub map_selector_slot: i64,
    #[serde(rename = "HasMapSelectorSlot")]
    pub has_map_selector_slot: bool,
    /// MapSelectorKey is the raw byte passed to 0x11019 before it returns the
    /// cache slot above. It is optional because legacy map JSON does not carry
    /// its native source provenance. It must never be inferred from Fig.
    #[serde(rename = "MapSelectorKey")]
    pub map_selector_key: i64,
    #[serde(rename = "HasMapSelectorKey")]
    pub has_map_selector_key: bool,
    /// NativeMapPresentation is the materialized battle-local runtime
    /// +0/+1/+3/+4 subset. It is valid only after the native selector
    /// construction path succeeds; normalized X/Y/Dir/OffX/OffY are not aliases.
    #[serde(rename = "NativeMapPresentation")]
    pub native_map_presentation: NativeMapPresentationState,
    #[serde(rename = "HasNativeMapPresentation")]
    pub has_native_map_presentation: bool,
    /// BattleFig is the separately sourced native unit+7 selector for FIGANI.
    /// FDFIELD roster b1 supplies it; missing older JSON keeps the Fig fallback.
    #[serde(rename = "BattleFig")]
    pub battle_fig: i64,
    #[serde(rename = "HasBattleFig")]
    pub has_battle_fig: bool,
    /// NativeIdentity is the persistent-record +0x08 key consumed by native
    /// post-handler 0x11506. It is optional provenance and must never be inferred
    /// from Fig, BattleFig, portrait, or map-selector fields.
    #[serde(rename = "NativeIdentity")]
    pub native_identity: i64,
    #[serde(rename = "HasNativeIdentity")]
    pub has_native_identity: bool,
    /// NativeRecordByte8 preserves runtime/persistent record +8 without
    /// assigning a universal identity meaning. Player persistent records use
    /// this byte as NativeIdentity, while scripted runtime construction copies
    /// its visual selector into both +7 and +8.
    #[serde(rename = "NativeRecordByte8")]
    pub native_record_byte8: u8,
    #[serde(rename = "HasNativeRecordByte8")]
    pub has_native_record_byte8: bool,
    /// NativePositionRecord preserves the six-byte FDFIELD 3N+2 record paired
    /// with a scripted constructor row. The constructor reads XWord/YWord low
    /// bytes and may relocate them through its nearest-free-cell search.
    #[serde(rename = "NativePositionRecord")]
    pub native_position_record: NativePositionRecord,
    #[serde(rename = "HasNativePositionRecord")]
    pub has_native_position_record: bool,
    /// NativeRecordRace/Class are the independently proven raw bytes at
    /// persistent/runtime +0x1f/+0x20. They remain separate from normalized
    /// class labels and from the immutable constructor-table provenance because
    /// class change rewrites only +0x20.
    #[serde(rename = "NativeRecordRace")]
    pub native_record_race: u8,
    #[serde(rename = "HasNativeRecordRace")]
    pub has_native_record_race: bool,
    #[serde(rename = "NativeRecordClass")]
    pub native_record_class: u8,
    #[serde(rename = "HasNativeRecordClass")]
    pub has_native_record_class: bool,
    /// NativeConstructor preserves the proven EXE static-table provenance for
    /// unit+0x1f/+0x20 without assigning any gameplay meaning to raw bytes.
    /// Go 標籤：`json:"native_constructor,omitempty"`
    #[serde(rename = "native_constructor", skip_serializing_if = "Option::is_none")]
    pub native_constructor: Option<NativeConstructorTable>,
    /// nativeConstructorPending 標記由 Load 建出、尚未經 0x10C50＋0x1B750 完整
    /// 物化的 FDFIELD 單位；BindNativeFutureItemRows 會補齊並清掉。
    #[serde(skip)]
    pub(crate) native_constructor_pending: bool,
    #[serde(rename = "X")]
    pub x: i64,
    #[serde(rename = "Y")]
    pub y: i64,
    /// engine projection; native bit7 is raw provenance only
    #[serde(rename = "Acted")]
    pub acted: bool,
    /// 出場波次(原版 FDFIELD b21;事件按 group 放出,doc 25/29)
    #[serde(rename = "Group")]
    pub group: i64,
    /// 是否已登場(事件進場機制:false=待命,尚未出現在戰場,doc 25)
    #[serde(rename = "OnField")]
    pub on_field: bool,
    /// normalized/editable spell IDs; not a raw unit+0x22 bitfield
    #[serde(rename = "Spells", deserialize_with = "null_as_default")]
    pub spells: Vec<i64>,
    /// NativeCommandMask is the runtime 40-bit command inventory enumerated by
    /// 0x1c269.  FDFIELD b13..b16 initializes bytes 0..3; byte 4 begins zero and
    /// can be OR-mutated by 0x1d7fb.  It is deliberately separate from Spells:
    /// command ID effects and target contracts are not inferred from this mask.
    /// Go 標籤：`json:"native_command_mask,omitempty"`
    #[serde(rename = "native_command_mask")]
    pub native_command_mask: [u8; 5],
    /// NativeTransient is the exact six-byte runtime interval unit+0x22..+0x27.
    /// It is distinct from the legacy normalized Buff*/Poison*/Seal*/Paralyze*
    /// approximation below: native command IDs 17..27 read/write individual raw
    /// bytes, and 0x1A866 decrements them separately once per camp phase.
    /// No gameplay name is inferred for an index from this storage alone.
    /// Go 標籤：`json:"native_transient"`
    #[serde(rename = "native_transient")]
    pub native_transient: [u8; 6],
    /// NativeRecordByte5/6 preserve the two raw gate bytes consumed by
    /// 0x1A866. Their gameplay meaning is intentionally unknown; neither is
    /// inferred from Acted, OnField, Alive, or Camp.
    /// Go 標籤：`json:"native_record_byte5,omitempty"`
    #[serde(rename = "native_record_byte5", skip_serializing_if = "go_is_zero")]
    pub native_record_byte5: u8,
    /// Go 標籤：`json:"has_native_record_byte5,omitempty"`
    #[serde(rename = "has_native_record_byte5", skip_serializing_if = "std::ops::Not::not")]
    pub has_native_record_byte5: bool,
    /// presentation-only HP transition, never serialized
    #[serde(skip)]
    pub(crate) native_death_flag_before: u8,
    #[serde(skip)]
    pub(crate) native_death_flag_valid: bool,
    /// Go 標籤：`json:"native_record_byte6,omitempty"`
    #[serde(rename = "native_record_byte6", skip_serializing_if = "go_is_zero")]
    pub native_record_byte6: u8,
    /// Go 標籤：`json:"has_native_record_byte6,omitempty"`
    #[serde(rename = "has_native_record_byte6", skip_serializing_if = "std::ops::Not::not")]
    pub has_native_record_byte6: bool,
    /// FDFIELD b17..b19 are copied verbatim by 0x10fb6..0x10fc5 to runtime
    /// +0x34..+0x36. Byte34's low nibble dispatches 0x13a9f, while its high
    /// bits have separate consumers; keep the complete bytes and provenance.
    /// Go 標籤：`json:"native_record_byte34,omitempty"`
    #[serde(rename = "native_record_byte34", skip_serializing_if = "go_is_zero")]
    pub native_record_byte34: u8,
    /// Go 標籤：`json:"has_native_record_byte34,omitempty"`
    #[serde(rename = "has_native_record_byte34", skip_serializing_if = "std::ops::Not::not")]
    pub has_native_record_byte34: bool,
    /// Go 標籤：`json:"native_record_byte35,omitempty"`
    #[serde(rename = "native_record_byte35", skip_serializing_if = "go_is_zero")]
    pub native_record_byte35: u8,
    /// Go 標籤：`json:"has_native_record_byte35,omitempty"`
    #[serde(rename = "has_native_record_byte35", skip_serializing_if = "std::ops::Not::not")]
    pub has_native_record_byte35: bool,
    /// Go 標籤：`json:"native_record_byte36,omitempty"`
    #[serde(rename = "native_record_byte36", skip_serializing_if = "go_is_zero")]
    pub native_record_byte36: u8,
    /// Go 標籤：`json:"has_native_record_byte36,omitempty"`
    #[serde(rename = "has_native_record_byte36", skip_serializing_if = "std::ops::Not::not")]
    pub has_native_record_byte36: bool,
    /// NativeJoinPersistentPending：sub_112A5 建構之後、還沒經過戰後 0x11506 整筆抄回的
    /// 持續記錄。這期間持續槽裡 +0..+4、+0x34..+0x36、+0x3d 仍是建構前的殘值，寫回槽時
    /// 不從 Unit 覆蓋；戰後同步（syncPartyFromBattle）清掉。
    /// Go 標籤：`json:"native_join_persistent_pending,omitempty"`
    #[serde(rename = "native_join_persistent_pending", skip_serializing_if = "std::ops::Not::not")]
    pub native_join_persistent_pending: bool,
    /// NativeRecordByte3D is copied from FDFIELD b2 by 0x10fc8. It is distinct
    /// from constructor-table race at runtime +0x1f.
    #[serde(rename = "NativeRecordByte3D")]
    pub native_record_byte3_d: u8,
    #[serde(rename = "HasNativeRecordByte3D")]
    pub has_native_record_byte3_d: bool,
    /// NativeRecordDeathEffect is exact runtime +0x31..+0x33 from b22..b24.
    #[serde(rename = "NativeRecordDeathEffect")]
    pub native_record_death_effect: [u8; 3],
    #[serde(rename = "HasNativeRecordDeathEffect")]
    pub has_native_record_death_effect: bool,
    /// These source bytes are not read by 0x10c50; preserve them without names.
    #[serde(rename = "NativeFDFIELDSourceByte3")]
    pub native_fdfield_source_byte3: u8,
    #[serde(rename = "NativeFDFIELDSourceByte20")]
    pub native_fdfield_source_byte20: u8,
    #[serde(rename = "NativeFDFIELDSourceByte25")]
    pub native_fdfield_source_byte25: u8,
    /// NativeRecordWord42 is the optional raw u16 at runtime record +0x42.
    /// ch15_post compares this word directly. When present on a loaded scripted
    /// roster it is also the proven constructor source for initial HP/MaxHP.
    /// Go 標籤：`json:"native_record_word42,omitempty"`
    #[serde(rename = "native_record_word42", skip_serializing_if = "go_is_zero")]
    pub native_record_word42: u16,
    /// Go 標籤：`json:"has_native_record_word42,omitempty"`
    #[serde(rename = "has_native_record_word42", skip_serializing_if = "std::ops::Not::not")]
    pub has_native_record_word42: bool,
    /// NativeRecordWord46 is the optional constructor-produced max-MP word at
    /// runtime +0x46. It is not inferred from normalized MaxMP.
    /// Go 標籤：`json:"native_record_word46,omitempty"`
    #[serde(rename = "native_record_word46", skip_serializing_if = "go_is_zero")]
    pub native_record_word46: u16,
    /// Go 標籤：`json:"has_native_record_word46,omitempty"`
    #[serde(rename = "has_native_record_word46", skip_serializing_if = "std::ops::Not::not")]
    pub has_native_record_word46: bool,
    /// 角色物品欄 item IDs；原版 unit+0x0a 起 8×2B
    #[serde(rename = "Inventory", deserialize_with = "null_as_default")]
    pub inventory: Vec<i64>,
    /// 與 Inventory 對齊；true 表示該欄位目前已裝備
    #[serde(rename = "Equipped", deserialize_with = "null_as_default")]
    pub equipped: Vec<bool>,
    /// 原始 8 個 source bytes；0xff 保留空槽位置
    #[serde(rename = "InventorySlots", deserialize_with = "null_as_default")]
    pub inventory_slots: Vec<i64>,
    /// NativeInventoryFlags is the constructor's raw eight-cell flag view. It is
    /// present only when InventorySlots came from the proven source layout.
    #[serde(rename = "NativeInventoryFlags", deserialize_with = "null_as_default")]
    pub native_inventory_flags: Vec<i64>,
    /// Base* are the persistent pre-remake equipment values. Existing scenario
    /// data stores effective values, so EquipmentBaseSet is true for those
    /// records and newly purchased equipment is added without double counting.
    #[serde(rename = "BaseAP")]
    pub base_ap: i64,
    /// Go 與 `base_ap` 同列宣告，註解見該欄位。
    #[serde(rename = "BaseDP")]
    pub base_dp: i64,
    /// Go 與 `base_ap` 同列宣告，註解見該欄位。
    #[serde(rename = "BaseHIT")]
    pub base_hit: i64,
    /// Go 與 `base_ap` 同列宣告，註解見該欄位。
    #[serde(rename = "BaseEV")]
    pub base_ev: i64,
    /// Go 與 `base_ap` 同列宣告，註解見該欄位。
    #[serde(rename = "BaseMV")]
    pub base_mv: i64,
    #[serde(rename = "BaseAtkMin")]
    pub base_atk_min: i64,
    #[serde(rename = "BaseAtkMax")]
    pub base_atk_max: i64,
    #[serde(rename = "EquipmentBaseSet")]
    pub equipment_base_set: bool,
    /// FDFIELD b22 + b23..24；0=item、1=gold，2/3 特殊效果先原值保留
    #[serde(rename = "DeathEffect")]
    pub death_effect: Option<DeathEffect>,
    /// 可執行死亡獎勵；type2 已知 handler 由 exporter lower 成 item/gold
    #[serde(rename = "DeathReward")]
    pub death_reward: Option<DeathEffect>,
    /// 朝向:0下 1左 2上 3右(原版 Z2,FDICON 方向幀)
    #[serde(rename = "Dir")]
    pub dir: i64,
    /// 行軍/移動的像素位移(顯示用;0=正在格上)
    #[serde(rename = "OffX")]
    pub off_x: f64,
    /// 進場時從邊緣滑入,漸減到 0
    #[serde(rename = "OffY")]
    pub off_y: f64,
    /// ---- 輔助法術暫時狀態(doc02 §6.4;施放邏輯見 magic.go CastArea/applySpell)----
    /// BuffAPPct／BuffDPPct 服務的是**沒有原版 raw record 的舊可編輯劇本**。
    /// 走原版那條的單位不經過這裡：command 17／18／19 的 handler 直接改 record
    /// 的 +0x48／+0x4A（native_raw_word_step.go），換裝備或狀態到期時由
    /// nativeEquipmentTotals 依 +0x22..+0x24 重算。見
    /// docs/knowledge-base/107-transient-status-semantics-20260910.md。
    /// 魔刃術:AP 加成百分比
    #[serde(rename = "BuffAPPct")]
    pub buff_ap_pct: i64,
    /// 魔鎧術:DP 加成百分比
    #[serde(rename = "BuffDPPct")]
    pub buff_dp_pct: i64,
    /// 風行術:HIT 加成
    #[serde(rename = "BuffHit")]
    pub buff_hit: i64,
    /// 風行術:EV 加成
    #[serde(rename = "BuffEV")]
    pub buff_ev: i64,
    /// 上述加成共用剩餘回合數(原版三招各自可疊加回合,重製簡化成單一計時器)
    #[serde(rename = "BuffTurns")]
    pub buff_turns: i64,
    /// 封咒術:禁止施法
    #[serde(rename = "Sealed")]
    pub sealed: bool,
    #[serde(rename = "SealTurns")]
    pub seal_turns: i64,
    /// 毒擊術:每回合扣 MaxHP 的 10%(doc02 §6.4)
    #[serde(rename = "Poisoned")]
    pub poisoned: bool,
    #[serde(rename = "PoisonTurns")]
    pub poison_turns: i64,
    /// 麻痺術:無法行動(是否擋下行動由呼叫端 UI/AI 檢查此欄位)
    #[serde(rename = "Paralyzed")]
    pub paralyzed: bool,
    #[serde(rename = "ParalyzeTurns")]
    pub paralyze_turns: i64,
    /// ---- 經驗值/升級(doc02 §4.5/§4.6;doc03 0x43;實作見 growth.go)----
    /// typed +0x3e projection；0x1145a/0x2efb7 以它為 HIT/EV 的共用 base；不等同直接 raw dump。
    /// FDFIELD export 的 generic enemy HIT/EV 仍可能是近似值；evidence-backed
    /// player scenarios必須顯式保存DX，shop preview/class growth不可由derived值猜回。
    /// 尚未實際影響命中/閃避。
    #[serde(rename = "DX")]
    pub dx: i64,
    /// 目前經驗值(滿 100 升級,doc03 0x43「EX 經驗」);用 float64 累加,避免
    /// 攻擊/法術公式算出的小數經驗(如 40/施法者等級)逐次相加時被提早捨去。
    #[serde(rename = "Exp")]
    pub exp: f64,
    /// 本單位每級可給出的經驗值(doc02 §4.5「守方每級經驗」;來源 EXE
    #[serde(rename = "ExpPerLevel")]
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
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeConstructorTable {
    /// Go 標籤：`json:"branch"`
    #[serde(deserialize_with = "null_as_default")]
    pub branch: String,
    /// Go 標籤：`json:"index"`
    pub index: i64,
    /// Go 標籤：`json:"record"`
    #[serde(with = "go_bytes")]
    pub record: Vec<u8>,
    /// Go 標籤：`json:"aux_record,omitempty"`
    #[serde(with = "go_bytes", skip_serializing_if = "Vec::is_empty")]
    pub aux_record: Vec<u8>,
}

impl NativeConstructorTable {
    /// Go：`model.go:234`，`func (t *NativeConstructorTable) validate() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn validate(&self) -> Result<()> {
        match self.branch.as_str() {
            "high_class" => {
                if self.index < 0 || self.index >= 68 || self.record.len() != 10 || !self.aux_record.is_empty() {
                    return Err(battle_err!("invalid high_class table record"));
                }
            }
            "lower_class" => {
                if self.index < 0 || self.index >= 32 || self.record.len() != 24 || self.aux_record.len() != 11 {
                    return Err(battle_err!("invalid lower_class table records"));
                }
            }
            _ => return Err(battle_err!("unknown constructor table branch {}", go_quote(&self.branch))),
        }
        Ok(())
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
    let Some(cache) = cache else {
        return Err(battle_err!("native map selector: nil cache"));
    };
    for (i, u) in units.iter().enumerate() {
        if !u.has_map_selector_key {
            return Err(battle_err!("native map selector: unit {i} has no explicit raw key"));
        }
        if !(0..=0xff).contains(&u.map_selector_key) {
            return Err(battle_err!("native map selector: unit {i} has invalid raw key"));
        }
        if !(0..=0xff).contains(&u.x) || !(0..=0xff).contains(&u.y) {
            return Err(battle_err!("native map selector: unit {i} coordinate outside byte range"));
        }
    }
    let mut slots = Vec::with_capacity(units.len());
    for (i, u) in units.iter().enumerate() {
        let slot = cache
            .slot_for(u.map_selector_key)
            .map_err(|e| BattleError::from(e).wrap(format!("native map selector: unit {i}: ")))?;
        slots.push(slot);
    }
    for (i, u) in units.iter_mut().enumerate() {
        (u.map_selector_slot, u.has_map_selector_slot) = (slots[i], true);
        u.materialize_native_map_presentation()
            .map_err(|e| e.wrap(format!("native map selector: unit {i} presentation: ")))?;
    }
    Ok(())
}

/// initialEquipmentFlags mirrors the original spawn constructor: its first
/// two source inventory bytes are placed with flag 0x40; later bytes start
/// held/unequipped. The editable exports omit 0xff empty bytes, so the
/// compacted list is the available representation here.
///
/// Go：`model.go:295`，`func initialEquipmentFlags(n int) []bool`
pub(crate) fn initial_equipment_flags(n: i64) -> Vec<bool> {
    let mut flags = vec![false; n as usize];
    for flag in flags.iter_mut().take(2) {
        *flag = true;
    }
    flags
}

/// Go：`model.go:303`，`func materializeInventory(source, compact []int) ([]int, []bool, []int)`
pub(crate) fn materialize_inventory(source: &[i64], compact: &[i64]) -> (Vec<i64>, Vec<bool>, Vec<i64>) {
    if source.len() != 8 {
        return (compact.to_vec(), initial_equipment_flags(compact.len() as i64), compact.to_vec());
    }
    let mut runtime = vec![0xffi64; 8];
    let mut flags = [false; 8];
    if source[0] != 0xff {
        (runtime[0], flags[0]) = (source[0], true);
        if source[1] != 0xff {
            (runtime[1], flags[1]) = (source[1], true);
        }
    } else if source[1] != 0xff {
        (runtime[0], flags[0]) = (source[1], true);
    }
    runtime[2..8].copy_from_slice(&source[2..8]);
    let mut ids = Vec::with_capacity(8);
    let mut used_flags = Vec::with_capacity(8);
    for (i, &id) in runtime.iter().enumerate() {
        if id != 0xff {
            ids.push(id);
            used_flags.push(flags[i]);
        }
    }
    (ids, used_flags, runtime)
}

impl Unit {
    /// Go：`model.go:334`，`func (u *Unit) normalizeInventorySlots()`
    pub(crate) fn normalize_inventory_slots(&mut self) {
        if self.inventory_slots.len() == 8 {
            return;
        }
        self.inventory_slots = vec![0xff; 8];
        for (i, &id) in self.inventory.iter().enumerate().take(8) {
            self.inventory_slots[i] = id;
        }
    }

    /// AddInventoryItem preserves the original fixed eight slot layout while
    /// keeping the compact Inventory view used by editable scripts.
    ///
    /// Go：`model.go:352`，`func (u *Unit) AddInventoryItem(id int, equipped bool) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn add_inventory_item(&mut self, id: i64, equipped: bool) -> bool {
        if self.inventory.len() >= 8 {
            return false;
        }
        self.normalize_inventory_slots();
        let Some(slot) = (0..self.inventory_slots.len()).find(|&i| !self.inventory_cell_occupied(i as i64)) else {
            return false;
        };
        self.inventory_slots[slot] = id;
        if self.native_inventory_flags.len() as i64 == NATIVE_INVENTORY_CELLS {
            self.native_inventory_flags[slot] = 0;
        }
        self.inventory.push(id);
        self.equipped.push(equipped);
        true
    }

    /// Go：`model.go:376`，`func (u *Unit) RemoveInventoryIndex(index int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn remove_inventory_index(&mut self, index: i64) -> bool {
        if index < 0 || index >= self.inventory.len() as i64 {
            return false;
        }
        self.normalize_inventory_slots();
        let (mut seen, mut slot) = (0, -1i64);
        for i in 0..self.inventory_slots.len() as i64 {
            if self.inventory_cell_occupied(i) {
                if seen == index {
                    slot = i;
                    break;
                }
                seen += 1;
            }
        }
        if slot >= 0 {
            self.inventory_slots[slot as usize] = 0xff;
            if self.native_inventory_flags.len() as i64 == NATIVE_INVENTORY_CELLS {
                self.native_inventory_flags[slot as usize] = 0x80;
            }
        }
        self.inventory.remove(index as usize);
        if (index as usize) < self.equipped.len() {
            self.equipped.remove(index as usize);
        }
        true
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
        if damage <= 0 {
            return;
        }
        let (before_hp, before_flag) = (self.hp, self.native_record_byte5);
        self.hp = self.hp.wrapping_sub(damage);
        if self.hp < 0 {
            self.hp = 0;
        }
        if self.hp == 0 && self.has_native_record_byte5 {
            if before_hp > 0 && before_flag & 1 == 0 {
                (self.native_death_flag_before, self.native_death_flag_valid) = (before_flag, true);
            }
            self.native_record_byte5 = 1;
        }
    }

    /// Go：`model.go:428`，`func (u *Unit) NativeDeathPresentationFlag() (byte, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_death_presentation_flag(&self) -> (u8, bool) {
        if self.hp != 0 {
            return (0, false);
        }
        (self.native_death_flag_before, self.native_death_flag_valid)
    }

    /// Go：`model.go:435`，`func (u *Unit) ClearNativeDeathPresentationFlag()`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn clear_native_death_presentation_flag(&mut self) {
        (self.native_death_flag_before, self.native_death_flag_valid) = (0, false);
    }

    /// RestoreNativeHP mirrors the proven revive writer: restore current HP and
    /// clear only raw byte+5 bit0 when the unit carries native provenance.
    ///
    /// Go：`model.go:443`，`func (u *Unit) RestoreNativeHP()`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn restore_native_hp(&mut self) {
        self.hp = self.max_hp;
        self.clear_native_death_presentation_flag();
        if self.has_native_record_byte5 {
            self.native_record_byte5 &= !1;
        }
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
        if self.buff_turns > 0 {
            self.buff_turns -= 1;
            if self.buff_turns == 0 {
                (self.buff_ap_pct, self.buff_dp_pct, self.buff_hit, self.buff_ev) = (0, 0, 0, 0);
            }
        }
        if self.seal_turns > 0 {
            self.seal_turns -= 1;
            if self.seal_turns == 0 {
                self.sealed = false;
            }
        }
        if self.poisoned {
            let dmg = (self.max_hp / 10).max(1);
            self.hp = self.hp.wrapping_sub(dmg);
            if self.hp < 0 {
                self.hp = 0;
            }
        }
        if self.poison_turns > 0 {
            self.poison_turns -= 1;
            if self.poison_turns == 0 {
                self.poisoned = false;
            }
        }
        if self.paralyze_turns > 0 {
            self.paralyze_turns -= 1;
            if self.paralyze_turns == 0 {
                self.paralyzed = false;
            }
        }
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
        let row = NATIVE_ITEM_EFFECT_ROW_SIZE as usize;
        if rows.is_empty() || !rows.len().is_multiple_of(row) || rows.len() / row > 0x100 {
            return Err(battle_err!("native future item rows: invalid byte length {}", rows.len()));
        }
        self.native_future_item_rows = rows.to_vec();
        // 開局在場的 FDFIELD 單位在原版也是由 0x10C50 建構器加 0x1B750 裝備重算寫出
        // +0x48..+0x4E 與 HP／MP；Load 沒有物品表，所以延到這裡補齊。第四章原版收據
        // （work/parity-slot-ch04/sample-r7 checkpoint-0636 的 raw record）劍士 lv5 是
        // AP 55／DP 13，authored JSON 只有 20／12，敵方 AI 的 0x14237 評分
        // （actor+0x48 − target+0x4A > 2）因此永遠不接戰。
        for index in 0..self.units.len() {
            if !self.units[index].native_constructor_pending {
                continue;
            }
            materialize_native_future_constructor(Some(&mut self.units[index]), &self.native_future_item_rows)
                .map_err(|e| e.wrap(format!("native constructor unit {index}: ")))?;
            self.units[index].native_constructor_pending = false;
        }
        for unit in &mut self.roster {
            unit.native_constructor_pending = false;
        }
        Ok(())
    }

    /// BindNativeMovementCostRows supplies the versioned 0x4e555 movement table
    /// used by native AI/path helpers. The table is immutable after binding and is
    /// copied so callers cannot mutate an in-progress decision.
    ///
    /// Go：`model.go:676`，`func (s *State) BindNativeMovementCostRows(rows [][]byte) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn bind_native_movement_cost_rows(&mut self, rows: &[Vec<u8>]) -> Result<()> {
        if rows.len() as i64 != NATIVE_MOVEMENT_COST_ROW_COUNT {
            return Err(battle_err!("native movement rows: invalid row count"));
        }
        for (index, row) in rows.iter().enumerate() {
            if row.len() as i64 != NATIVE_MOVEMENT_COST_ROW_SIZE {
                return Err(battle_err!("native movement selector={index}: invalid row length {}", row.len()));
            }
        }
        self.native_movement_cost_rows = rows.to_vec();
        Ok(())
    }
}

/// Cell 格子座標。
///
/// Go：`model.go:692`
///
/// JSON：Go 沒有標籤，寫出 `X`／`Y`；讀取時 Go 欄位名不分大小寫，`units.json` 的 `own_deploy`
/// 實際寫成 `x`／`y`，所以兩種都收（其餘大小寫組合不收，見 `KNOWN_DIFFERENCES`）。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Cell {
    #[serde(rename = "X", alias = "x")]
    pub x: i64,
    #[serde(rename = "Y", alias = "y")]
    pub y: i64,
}

/// DeathEffect 原樣保存 FDFIELD 單位記錄 b22 與 b23..24 的 u16。Type 0/1 是死亡時掉物/金錢；
/// 2/3 的特殊事件語意尚未完全解明，runtime 在釘死前不得猜測執行。
///
/// Go：`model.go:696`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeTreasureEventRule {
    /// Go 標籤：`json:"event_id"`
    pub event_id: i64,
    /// Go 標籤：`json:"item_by_slot"`
    #[serde(deserialize_with = "null_as_default")]
    pub item_by_slot: Vec<i64>,
    /// Go 標籤：`json:"open_slots"`
    #[serde(deserialize_with = "null_as_default")]
    pub open_slots: Vec<i64>,
}

/// NativeFieldEvent 是 FDFIELD 控制段的原始兩位元組格子事件列。
/// 0x13a44 只把 EventID 寫入全域 selector；高階玩法名稱仍由 handler 證據決定。
///
/// Go：`model.go:721`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
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
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
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
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativePositionRecord {
    #[serde(rename = "XWord")]
    pub x_word: u16,
    #[serde(rename = "YWord")]
    pub y_word: u16,
    #[serde(rename = "RawKey")]
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
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeFieldModeRange {
    /// Go 標籤：`json:"start"`
    pub start: i64,
    /// Go 標籤：`json:"end"`
    pub end: i64,
    /// Go 標籤：`json:"mode"`
    pub mode: u8,
}

/// Go：`model.go:767`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeFieldTextIndices {
    /// Go 標籤：`json:"missing_item"`
    pub missing_item: i64,
    /// Go 標籤：`json:"success"`
    pub success: i64,
    /// Go 標籤：`json:"final"`
    pub r#final: i64,
}

/// Go：`model.go:773`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeFieldPresentation {
    /// Go 標籤：`json:"archive"`
    #[serde(deserialize_with = "null_as_default")]
    pub archive: String,
    /// Go 標籤：`json:"resource"`
    pub resource: i64,
    /// Go 標籤：`json:"frames"`
    pub frames: i64,
    /// Go 標籤：`json:"helper"`
    #[serde(deserialize_with = "null_as_default")]
    pub helper: String,
    /// Go 標籤：`json:"destination_offset"`
    pub destination_offset: i64,
    /// Go 標籤：`json:"stride"`
    pub stride: i64,
    /// Go 標籤：`json:"transparent"`
    pub transparent: i64,
    /// Go 標籤：`json:"delay_helper"`
    #[serde(deserialize_with = "null_as_default")]
    pub delay_helper: String,
    /// Go 標籤：`json:"delay_ticks"`
    pub delay_ticks: i64,
}

/// NativeTurnActivation 描述一個已證實的格子 handler 對 live FDFIELD 回合列
/// 的改寫。它辨識改寫前的原始列，不是正規化 scenario 排程，也不把 RawCamp
/// 解讀成 battle.Camp。
///
/// Go：`model.go:788`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
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
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeFieldTurnChain {
    /// Go 標籤：`json:"handler"`
    #[serde(deserialize_with = "null_as_default")]
    pub handler: String,
    /// Go 標籤：`json:"trigger_record_byte8"`
    pub trigger_record_byte8: u8,
    /// Go 標籤：`json:"mismatch_text_index"`
    pub mismatch_text_index: i64,
    /// Go 標籤：`json:"success_text_index"`
    pub success_text_index: i64,
    /// Go 標籤：`json:"state_writes"`
    #[serde(deserialize_with = "null_as_default")]
    pub state_writes: Vec<NativeEventStateWrite>,
    /// Go 標籤：`json:"turn_activations"`
    #[serde(deserialize_with = "null_as_default")]
    pub turn_activations: Vec<NativeTurnActivation>,
}

/// NativeFieldEventRule 保存已閉合 handler 的資料，不自行決定 selector 的呼叫時機。
///
/// Go：`model.go:815`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeFieldEventRule {
    /// Go 標籤：`json:"event_id"`
    pub event_id: i64,
    /// Go 標籤：`json:"handler,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "String::is_empty")]
    pub handler: String,
    /// Go 標籤：`json:"selector"`
    pub selector: u8,
    /// Go 標籤：`json:"trigger_gate,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "String::is_empty")]
    pub trigger_gate: String,
    /// Go 標籤：`json:"set_mode_ranges,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub set_mode_ranges: Vec<NativeFieldModeRange>,
    /// Go 標籤：`json:"set_state_index,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub set_state_index: Option<i64>,
    /// Go 標籤：`json:"set_state_value,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub set_state_value: Option<i64>,
    /// Go 標籤：`json:"once_state_index,omitempty"`
    #[serde(rename = "once_state_index", skip_serializing_if = "Option::is_none")]
    pub once_state: Option<i64>,
    /// Go 標籤：`json:"required_nonzero_state_index,omitempty"`
    #[serde(rename = "required_nonzero_state_index", skip_serializing_if = "Option::is_none")]
    pub required_nonzero_state: Option<i64>,
    /// Go 標籤：`json:"required_item,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub required_item: Option<i64>,
    /// Go 標籤：`json:"consume_item,omitempty"`
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    pub consume_item: bool,
    /// Go 標籤：`json:"spawn_group,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub spawn_group: Option<i64>,
    /// Go 標籤：`json:"join_character,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub join_character: Option<i64>,
    /// Go 標籤：`json:"text_indices,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub text_indices: Option<NativeFieldTextIndices>,
    /// Go 標籤：`json:"presentation,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub presentation: Option<NativeFieldPresentation>,
    /// Go 標籤：`json:"turn_activation,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub turn_activation: Option<NativeTurnActivation>,
    /// Go 標籤：`json:"turn_chain,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub turn_chain: Option<NativeFieldTurnChain>,
}

impl State {
    /// TreasureAt 查詢尚未取得的寶物格。
    ///
    /// Go：`model.go:836`，`func (s *State) TreasureAt(x, y int) (Treasure, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn treasure_at(&self, x: i64, y: i64) -> (Treasure, bool) {
        let Some(t) = self.treasures.as_ref().and_then(|m| m.get(&Cell { x, y })) else {
            return (Treasure::default(), false);
        };
        let open = t.slot >= 0
            && t.slot < self.native_event_state.len() as i64
            && self.native_event_state[t.slot as usize] == 0;
        (t.clone(), open)
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
        let fail = || (Treasure::default(), false);
        let Some(ui) = u else {
            return fail();
        };
        {
            let unit = &self.units[ui];
            if !unit.on_field || !unit.alive() || unit.x != x || unit.y != y {
                return fail();
            }
        }
        let (mut t, ok) = self.treasure_at(x, y);
        if !ok {
            return fail();
        }
        if self.has_native_map_event_grid && self.validate_native_ai_mode5_event_grid().is_err() {
            return fail();
        }
        match t.kind.as_str() {
            "item" => {
                if self.units[ui].inventory.len() >= 8 {
                    return fail();
                }
                if !self.units[ui].add_inventory_item(t.value, false) {
                    return fail();
                }
            }
            // Game owns the campaign bank; returning the reward lets it add atomically.
            "gold" => {}
            "event" => {
                let claimed;
                (t, claimed) = self.claim_native_treasure_event(ui, t);
                if !claimed {
                    return fail();
                }
            }
            _ => return fail(),
        }
        self.mark_treasure_event(t.slot);
        if update_map && !self.complete_treasure_map_update() {
            return fail(); // 完整緩衝已於交易前驗證，正常路徑不會抵達此處。
        }
        (t, true)
    }

    /// Go：`model.go:894`，`func (s *State) markTreasureEvent(slot int)`
    pub(crate) fn mark_treasure_event(&mut self, slot: i64) {
        self.native_event_state[slot as usize] = 1;
        self.opened_treasure.get_or_insert_with(BTreeMap::new).insert(slot, true);
    }

    /// CompleteTreasureMapUpdate 共用玩家 0x19268 與敵方 mode 5 的 0x12263。
    /// 沒有原生緩衝的可編輯測試場景不需更新；已材料化但損壞的緩衝拒絕寫入。
    ///
    /// Go：`model.go:904`，`func (s *State) CompleteTreasureMapUpdate() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn complete_treasure_map_update(&mut self) -> bool {
        if !self.has_native_map_event_grid {
            return true;
        }
        self.validate_native_ai_mode5_event_grid().is_ok() && self.advance_native_ai_mode5_event_grid().is_ok()
    }

    /// Go：`model.go:914`，`func (s *State) claimNativeTreasureEvent(u *Unit, t Treasure) (Treasure, bool)`
    /// 所有權：`u` 是 `self.units` 的索引（Go 傳 `*Unit`）。
    pub(crate) fn claim_native_treasure_event(&mut self, u: UnitIdx, t: Treasure) -> (Treasure, bool) {
        let fail = || (Treasure::default(), false);
        let Some(rule) = self.native_treasure_event_rules.get(&t.value).cloned() else {
            return fail();
        };
        if t.slot < 0 || t.slot >= rule.item_by_slot.len() as i64 || rule.open_slots.is_empty() {
            return fail();
        }
        let item = rule.item_by_slot[t.slot as usize];
        if item <= 0 || item > 0xff || self.units[u].inventory.len() >= 8 {
            return fail();
        }
        let mut seen = BTreeSet::new();
        for &slot in &rule.open_slots {
            if !(0..16).contains(&slot) || !seen.insert(slot) {
                return fail();
            }
        }
        if !self.units[u].add_inventory_item(item, false) {
            return fail();
        }
        // Go 以 map 走訪 seen；每格各自寫 1，順序不影響結果。
        for slot in seen {
            self.mark_treasure_event(slot);
        }
        let mut t = t;
        t.kind = "item".to_string();
        t.value = item;
        (t, true)
    }

    /// OnFieldUnit 回傳該格上「已登場且存活」的單位(無則 nil)。
    ///
    /// Go：`model.go:942`，`func (s *State) UnitAt(x, y int) *Unit`
    /// 所有權：回傳的單位是 `self.units` 的索引。
    pub fn unit_at(&self, x: i64, y: i64) -> Option<UnitIdx> {
        self.units.iter().position(|u| u.on_field && u.alive() && u.x == x && u.y == y)
    }

    /// AliveCount 各陣營「已登場且存活」數(用於勝敗判定)。
    /// 注意:待命(未登場)單位不計入 → 敵方援軍未出時不會誤判全滅。
    ///
    /// Go：`model.go:953`，`func (s *State) AliveCount(c Camp) int`
    pub fn alive_count(&self, c: Camp) -> i64 {
        self.units.iter().filter(|u| u.on_field && u.alive() && u.camp == c).count() as i64
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
        let mut n = 0;
        for u in &self.units {
            if u.has_native_record_byte5 && u.native_record_byte5 & 1 != 0 {
                continue;
            }
            if !u.on_field && u.alive() && u.camp == c {
                n += 1;
            }
        }
        for u in &self.roster {
            if self.pending_groups.get(&u.group).copied().unwrap_or(false) && u.alive() && u.camp == c {
                n += 1;
            }
        }
        n
    }

    // ---- 載入(units.json,tools/export_units.py 產生)----
}

/// Go：`model.go:992`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct UnitsFile {
    /// Go 標籤：`json:"map"`
    pub map: i64,
    /// Go 標籤：`json:"w"`
    pub w: i64,
    /// Go 標籤：`json:"h"`
    pub h: i64,
    /// Go 標籤：`json:"own_deploy"`
    #[serde(deserialize_with = "null_as_default")]
    pub own_deploy: Vec<Cell>,
    /// Optional raw global snapshot; absent legacy/editable files must not
    /// fabricate provenance for native round predicates.
    /// Go 標籤：`json:"native_round_counter,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_round_counter: Option<i64>,
    /// Go 標籤：`json:"chests,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub chests: Vec<UnitsFileChest>,
    /// Go 標籤：`json:"native_treasure_event_rules,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub native_treasure_event_rules: Vec<NativeTreasureEventRule>,
    /// Go 標籤：`json:"units"`
    #[serde(deserialize_with = "null_as_default")]
    pub units: Vec<UnitsFileUnit>,
}

/// Go：`model.go:1064`，`func campFrom(s string) Camp`
pub(crate) fn camp_from(s: &str) -> Camp {
    match s {
        "own" => OWN,
        "ally" => ALLY,
        _ => ENEMY,
    }
}

/// Load 從 units.json 建出戰鬥初始狀態。我方(own)依序放到部署格。
///
/// Go：`model.go:1076`，`func Load(path string) (*State, error)`
pub fn load(path: &str) -> Result<State> {
    let raw = read_file(path)?;
    let f: UnitsFile = decode_json(&raw)?;
    let mut st = State {
        w: f.w,
        h: f.h,
        own_deploy: f.own_deploy.clone(),
        turn: 1,
        flags: BTreeMap::new(),
        treasures: Some(BTreeMap::new()),
        opened_treasure: Some(BTreeMap::new()),
        native_treasure_event_rules: BTreeMap::new(),
        ..Default::default()
    };
    for rule in &f.native_treasure_event_rules {
        if rule.event_id < 0 || rule.event_id >= 90 || rule.item_by_slot.is_empty() || rule.open_slots.is_empty() {
            return Err(battle_err!("battle: invalid native treasure event rule {}", rule.event_id));
        }
        if st.native_treasure_event_rules.contains_key(&rule.event_id) {
            return Err(battle_err!("battle: duplicate native treasure event rule {}", rule.event_id));
        }
        st.native_treasure_event_rules.insert(rule.event_id, rule.clone());
    }
    if let Some(counter) = f.native_round_counter.filter(|&c| c > 0) {
        st.native_round_counter = counter;
    }
    for u in &f.units {
        if let Some(table) = &u.native_constructor {
            table.validate().map_err(|e| e.wrap("battle: native_constructor: "))?;
        }
        let camp = camp_from(&u.camp);
        let (inventory, equipped, runtime_slots) = materialize_inventory(&u.inventory_slots, &u.inventory);
        let mut nu = Unit {
            camp,
            name: u.name.clone(),
            cls_name: u.cls_name.clone(),
            class_id: u.class_id,
            lv: u.lv,
            hp: u.hp,
            max_hp: u.hp,
            mp: u.mp,
            max_mp: u.mp,
            ap: u.ap,
            dp: u.dp,
            mv: u.mv,
            hit: u.hit,
            ev: u.ev,
            crit_pct: u.crit,
            exp_per_level: u.ex,
            atk_min: u.atk_min,
            atk_max: u.atk_max,
            portrait: u.portrait,
            fig: u.fig,
            x: u.x,
            y: u.y,
            spells: u.spells.clone(),
            inventory,
            equipped,
            inventory_slots: runtime_slots,
            death_effect: u.death_effect,
            death_reward: u.death_reward,
            native_constructor: u.native_constructor.clone(),
            group: u.group,
            on_field: true, // 預設登場;Scenario 會把待命 group 設 false
            ..Default::default()
        };
        // 0x10eed initializes a freshly constructed record's byte +5 to zero.
        // Do not fabricate a raw value for an already-zero-HP input record: its
        // death writer/provenance is not present in this JSON boundary.
        if nu.hp > 0 {
            nu.native_record_byte5 = 0;
            nu.has_native_record_byte5 = true;
        }
        if let Some(v) = u.native_record_byte5 {
            (nu.native_record_byte5, nu.has_native_record_byte5) = (v, true);
        }
        if let Ok(flags) = native_inventory_flags_from_source(&u.inventory_slots) {
            nu.native_inventory_flags = flags;
        }
        // Older generated JSON lacks battle_fig; preserve its compatibility
        // behavior, while new exports carry the direct FDFIELD-b1 value.
        nu.battle_fig = u.fig;
        if let Some(v) = u.battle_fig {
            (nu.battle_fig, nu.has_battle_fig) = (v, true);
        }
        if let Some(identity) = u.native_identity {
            if !(0..=0xff).contains(&identity) {
                return Err(battle_err!(
                    "battle: unit {} native_identity {} outside byte range",
                    st.units.len(),
                    identity
                ));
            }
            (nu.native_identity, nu.has_native_identity) = (identity, true);
        }
        if let Some(v) = u.native_record_byte8 {
            (nu.native_record_byte8, nu.has_native_record_byte8) = (v, true);
        } else if nu.has_native_identity {
            // Compatibility for verified player-persistent assets created
            // before raw +8 was represented separately.
            nu.native_record_byte8 = nu.native_identity as u8;
            nu.has_native_record_byte8 = true;
        }
        if let Some(p) = u.native_position_record {
            nu.native_position_record = NativePositionRecord { x_word: p.x_word, y_word: p.y_word, raw_key: p.raw_key };
            nu.has_native_position_record = true;
        }
        if let Some(v) = u.native_record_race {
            (nu.native_record_race, nu.has_native_record_race) = (v, true);
        }
        if let Some(v) = u.native_record_class {
            (nu.native_record_class, nu.has_native_record_class) = (v, true);
        }
        if let Some(table) = &u.native_constructor {
            if !nu.has_native_record_race {
                (nu.native_record_race, nu.has_native_record_race) = (table.record[0], true);
            }
            if !nu.has_native_record_class {
                (nu.native_record_class, nu.has_native_record_class) = (table.record[1], true);
            }
        }
        if let Some(v) = u.map_selector_slot {
            (nu.map_selector_slot, nu.has_map_selector_slot) = (v, true);
        }
        if let Some(key) = u.map_selector_key {
            (nu.map_selector_key, nu.has_map_selector_key) = (key, true);
            // FDFIELD b0 is copied directly to native record +6 by the
            // constructor; map_selector_key is retained as separate provenance.
            (nu.native_record_byte6, nu.has_native_record_byte6) = (key as u8, true);
        }
        if let Some(v) = u.native_record_byte6 {
            (nu.native_record_byte6, nu.has_native_record_byte6) = (v, true);
        }
        if let Some(v) = u.native_record_byte34 {
            (nu.native_record_byte34, nu.has_native_record_byte34) = (v, true);
        }
        if let Some(v) = u.native_record_byte35 {
            (nu.native_record_byte35, nu.has_native_record_byte35) = (v, true);
        }
        if let Some(v) = u.native_record_byte36 {
            (nu.native_record_byte36, nu.has_native_record_byte36) = (v, true);
        }
        if let Some(v) = u.native_record_byte3_d {
            (nu.native_record_byte3_d, nu.has_native_record_byte3_d) = (v, true);
        }
        if !u.native_record_death_effect.is_empty() {
            if u.native_record_death_effect.len() != 3 {
                return Err(battle_err!("battle: unit {} native_record_death_effect needs 3 bytes", st.units.len()));
            }
            nu.native_record_death_effect.copy_from_slice(&u.native_record_death_effect);
            nu.has_native_record_death_effect = true;
        }
        if let Some(v) = u.native_source_byte3 {
            nu.native_fdfield_source_byte3 = v;
        }
        if let Some(v) = u.native_source_byte20 {
            nu.native_fdfield_source_byte20 = v;
        }
        if let Some(v) = u.native_source_byte25 {
            nu.native_fdfield_source_byte25 = v;
        }
        if let Some(v) = u.native_record_word42 {
            (nu.native_record_word42, nu.has_native_record_word42) = (v, true);
            (nu.hp, nu.max_hp) = (v as i64, v as i64);
        }
        if let Some(v) = u.native_record_word46 {
            (nu.native_record_word46, nu.has_native_record_word46) = (v, true);
            (nu.mp, nu.max_mp) = (v as i64, v as i64);
        }
        if let Some(table) = &u.native_constructor {
            // 0x10C50 建構器把表列的移動力寫進 runtime +0x3b（high_class record[8]、
            // lower_class record[7]）。authored `mv` 是早期由職業表猜的值；第四章
            // dosgolem 收據證實 map3 劍士 +0x3b=4、弓兵／法師=3，JSON 卻全是 6，敵方
            // 回合一走就分岔。增援已由 AppendGroupWithNativePlacement 走同一份表，
            // 開局群組在這裡補齊。
            let base = decode_native_future_constructor_base(Some(table), u.lv as u8)
                .map_err(|e| e.wrap(format!("battle: unit {} native_constructor: ", st.units.len())))?;
            (nu.mv, nu.base_mv) = (base.mobility as i64, base.mobility as i64);
            // 其餘數值（AP／DP／HIT／EV／HP／MP）要物品表才能重算，
            // BindNativeFutureItemRows 時再由建構器整批補齊。
            nu.native_constructor_pending = true;
        }
        nu.set_initial_command_mask(&u.initial_command_mask)
            .map_err(|e| e.wrap(format!("battle: unit {} initial_command_mask: ", st.units.len())))?;
        // 註:不再自動把 own 塞部署格 — 部署格保留給 scenario 主角隊(spawn_party);
        // FDFIELD 的 own(如哈諾/哈瓦特)用自己的出場座標(房子位置),由事件按回合放出。
        st.units.push(nu);
    }
    let map_path = go_join(&go_dir(path), "map.json");
    let cost = load_terrain_cost(&map_path, f.w, f.h);
    st.cost = if cost.is_empty() { None } else { Some(cost) };
    st.native_composition_event_bytes = load_native_composition_event_bytes(&map_path, f.w, f.h);
    (st.native_map_event_grid, st.has_native_map_event_grid) = load_native_map_event_grid(&map_path, f.w, f.h);
    if st.has_native_map_event_grid {
        reset_native_map_event_grid(&mut st.native_map_event_grid);
    }
    let native_round_seed;
    (st.native_turn_event_controls, native_round_seed, st.has_native_turn_event_control_state) =
        load_native_turn_event_controls(&map_path, f.w, f.h);
    if st.has_native_turn_event_control_state && st.native_round_counter == 0 {
        st.native_round_counter = native_round_seed;
    }
    (st.native_field_event_slots, st.native_field_events, st.native_field_event_rules) =
        load_native_field_events(&map_path, f.w, f.h);
    (st.native_tile_blit_modes, st.native_terrain_control, st.native_terrain_move_codes) =
        load_native_terrain_renderer_inputs(&map_path, f.w, f.h);
    // 0x4dbfc is the runtime constructor for composition byte+3: it replaces
    // every serialized value with 0xff before 0x14818/0x122dc mutate the grid.
    // Keep the exported field only as exact map provenance, never as live state.
    st.native_tile_blit_modes.fill(0xff);
    st.treasures = Some(load_treasures(&map_path, f.w, f.h, &f.chests));
    // 0x53+3*slot 的寶箱控制列：mode 5 的事件尾段直接讀它（type 0 給物品、
    // type 1 給金錢、其餘不給）。地圖 JSON 的 chests 就是這段 bytes 的具型別
    // 版本，所以從城鎮進戰場也有出處，不必等 CONTINUE 的 raw 影像。
    for chest in &f.chests {
        if chest.slot < 0
            || chest.slot >= st.native_chest_controls.len() as i64
            || chest.value < 0
            || chest.value > 0xffff
        {
            continue;
        }
        st.native_chest_controls[chest.slot as usize] =
            NativeChestControl { raw_type: chest.native_type, value: chest.value as u16 };
        st.has_native_chest_control_state = true;
    }
    Ok(st)
}

/// mapCostFile map.json 裡跟地形成本相關的欄位(其餘欄位 main.go 的 MapData 自己讀,這裡只挑 cost 用)。
///
/// Go：`model.go:1288`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub(crate) struct MapCostFile {
    /// Go 標籤：`json:"w"`
    pub w: i64,
    /// Go 標籤：`json:"h"`
    pub h: i64,
    /// Go 標籤：`json:"cost"`
    #[serde(deserialize_with = "null_as_default")]
    pub cost: Vec<i64>,
    /// Go 標籤：`json:"native_composition_event_bytes"`
    #[serde(with = "go_bytes")]
    pub native_composition_event_bytes: Vec<u8>,
    /// Go 標籤：`json:"native_field_event_slots"`
    #[serde(deserialize_with = "null_as_default")]
    pub native_field_event_slots: Vec<i64>,
    /// Go 標籤：`json:"native_field_events"`
    #[serde(deserialize_with = "null_as_default")]
    pub native_field_events: Vec<NativeFieldEvent>,
    /// Go 標籤：`json:"native_field_event_rules"`
    #[serde(deserialize_with = "null_as_default")]
    pub native_field_event_rules: Vec<NativeFieldEventRule>,
    /// Go 標籤：`json:"native_tile_blit_modes"`
    #[serde(with = "go_bytes")]
    pub native_tile_blit_modes: Vec<u8>,
    /// Go 標籤：`json:"native_terrain_control"`
    #[serde(with = "go_bytes")]
    pub native_terrain_control: Vec<u8>,
    /// Go 標籤：`json:"tiles"`
    #[serde(deserialize_with = "null_as_default")]
    pub tiles: Vec<i64>,
    /// Go 標籤：`json:"treasure_slots"`
    #[serde(deserialize_with = "null_as_default")]
    pub treasure_slots: Vec<i64>,
    /// Go 標籤：`json:"treasure_hidden"`
    #[serde(deserialize_with = "null_as_default")]
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
    let out = [NativeTurnEventControl::default(); 16];
    if w <= 0 || h <= 0 {
        return (out, 0, false);
    }
    let map_dir = go_dir(map_json_path);
    let Some(map_index) = go_scan_map_index(go_base(&map_dir)).filter(|&m| m >= 0) else {
        return (out, 0, false);
    };
    let catalog_path = go_join(&go_dir(&map_dir), "native_turn_event_controls.json");
    let Ok(raw) = std::fs::read(&catalog_path) else {
        return (out, 0, false);
    };
    if sha256_hex(&raw) != NATIVE_TURN_EVENT_CATALOG_SHA256 {
        return (out, 0, false);
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct SourceRef {
        file: String,
        size: i64,
        md5: String,
        sha256: String,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct RoundSeed {
        value: i64,
        writer: String,
        source: SourceRef,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct MapEntry {
        map: i64,
        control_resource: String,
        control_sha256: String,
        #[serde(deserialize_with = "null_as_default")]
        controls: Vec<NativeTurnEventControl>,
    }
    #[derive(serde::Deserialize, Default)]
    #[serde(default)]
    struct Catalog {
        schema_version: i64,
        source: SourceRef,
        round_seed: RoundSeed,
        #[serde(deserialize_with = "null_as_default")]
        maps: Vec<MapEntry>,
    }
    let Ok(catalog) = serde_json::from_slice::<Catalog>(&raw) else {
        return (out, 0, false);
    };
    if catalog.schema_version != 1
        || catalog.source.file != "FDFIELD.DAT"
        || catalog.source.size != 243169
        || catalog.source.md5 != "ecdb0436d26adfe5d107f2713fa7e9a2"
        || catalog.source.sha256 != "b0cf75d94f58603f091c7462c0494f0e83bd6edfb04c1acbf83ed4d938c7a513"
        || catalog.round_seed.value != 1
        || catalog.round_seed.writer != "0x2066e"
        || catalog.round_seed.source.file != "FD2.EXE"
        || catalog.round_seed.source.size != 357074
        || catalog.round_seed.source.md5 != "b97caf2239a27a896069d03549d96e1e"
        || catalog.round_seed.source.sha256 != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
        || catalog.maps.len() != 33
    {
        return (out, 0, false);
    }
    let mut seen = [false; 33];
    for entry in &catalog.maps {
        if entry.map < 0
            || entry.map >= seen.len() as i64
            || seen[entry.map as usize]
            || entry.control_resource != format!("FDFIELD_{:03}.bin", entry.map * 3 + 1)
            || entry.control_sha256.len() != 64
            || entry.controls.len() != out.len()
        {
            return (out, 0, false);
        }
        seen[entry.map as usize] = true;
    }
    // Go 的 `seen[mapIndex]` 在 mapIndex >= 33 時越界 panic；Rust 照樣直接索引。
    if !seen[map_index as usize] {
        return (out, 0, false);
    }
    for entry in &catalog.maps {
        if entry.map == map_index {
            let mut controls = out;
            controls.copy_from_slice(&entry.controls);
            return (controls, catalog.round_seed.value, true);
        }
    }
    (out, 0, false)
}

/// Go：`model.go:1382`，`func loadTreasures(mapJSONPath string, w, h int, chests []struct { Slot int 'json:"slot"' Kind string 'json:"type"' NativeType byte 'json:"native_type"' Value int 'json:"value"' }) map[Cell]Treasure`
pub(crate) fn load_treasures(
    map_json_path: &str,
    w: i64,
    h: i64,
    chests: &[UnitsFileChest],
) -> BTreeMap<Cell, Treasure> {
    let mut out = BTreeMap::new();
    let Ok(raw) = std::fs::read(map_json_path) else {
        return out;
    };
    let Ok(m) = serde_json::from_slice::<MapCostFile>(&raw) else {
        return out;
    };
    if m.w != w || m.h != h || m.treasure_slots.len() as i64 != w.wrapping_mul(h) {
        return out;
    }
    let mut defs = BTreeMap::new();
    for c in chests {
        if c.slot >= 0 && c.slot < 16 && matches!(c.kind.as_str(), "item" | "gold" | "event") && c.value > 0 {
            defs.insert(
                c.slot,
                Treasure {
                    slot: c.slot,
                    kind: c.kind.clone(),
                    native_type: c.native_type,
                    value: c.value,
                    hidden: false,
                },
            );
        }
    }
    for (i, &slot) in m.treasure_slots.iter().enumerate() {
        if slot < 0 {
            continue;
        }
        if let Some(t) = defs.get(&slot) {
            let mut t = t.clone();
            t.hidden = m.treasure_hidden.len() as i64 == w * h && m.treasure_hidden[i];
            out.insert(Cell { x: i as i64 % w, y: i as i64 / w }, t);
        }
    }
    out
}

/// loadTerrainCost 嘗試讀 units.json 同目錄的 map.json,取其 "cost" 陣列(tools/
/// export_engine_assets.py 產生;doc01 §5 地形控制表換算)。檔案不存在、沒有 cost 欄位、
/// 或尺寸對不上 units.json 的 w/h,一律回 nil(MoveCost 退回全平地=1,不 fail Load)。
///
/// Go：`model.go:1422`，`func loadTerrainCost(mapJSONPath string, w, h int) []int`
pub(crate) fn load_terrain_cost(map_json_path: &str, w: i64, h: i64) -> Vec<i64> {
    let Ok(raw) = std::fs::read(map_json_path) else {
        return Vec::new();
    };
    let Ok(m) = serde_json::from_slice::<MapCostFile>(&raw) else {
        return Vec::new();
    };
    if m.cost.is_empty() || m.w != w || m.h != h || m.cost.len() as i64 != w.wrapping_mul(h) {
        return Vec::new();
    }
    m.cost
}

/// loadNativeCompositionEventBytes accepts only an exact FDFIELD composition
/// export. These bytes are immutable source provenance, not the live +2 flags:
/// 0x4dbfc first masks each byte with 0x1f and caller-specific writers may then
/// add 0x40/0x80.
///
/// Go：`model.go:1441`，`func loadNativeCompositionEventBytes(mapJSONPath string, w, h int) []byte`
pub(crate) fn load_native_composition_event_bytes(map_json_path: &str, w: i64, h: i64) -> Vec<u8> {
    let Ok(raw) = std::fs::read(map_json_path) else {
        return Vec::new();
    };
    match serde_json::from_slice::<MapCostFile>(&raw) {
        Ok(m) if m.w == w && m.h == h && m.native_composition_event_bytes.len() as i64 == w.wrapping_mul(h) => {
            m.native_composition_event_bytes
        }
        _ => Vec::new(),
    }
}

/// loadNativeMapEventGrid rebuilds the exact four-byte cell prefix consumed by
/// 0x12e38/0x15df3 from the same FDFIELD export that produced map.json.  The
/// first four bytes are the native width/height header; each cell then keeps
/// tile word (+0/+1) and event word (+2/+3).  Callers still own the mutable
/// 0x4dbfc reset and later event writers.
///
/// Go：`model.go:1459`，`func loadNativeMapEventGrid(mapJSONPath string, w, h int) ([]byte, bool)`
pub(crate) fn load_native_map_event_grid(map_json_path: &str, w: i64, h: i64) -> (Vec<u8>, bool) {
    if w <= 0 || h <= 0 || w > 0xff || h > 0xff {
        return (Vec::new(), false);
    }
    let Ok(raw) = std::fs::read(map_json_path) else {
        return (Vec::new(), false);
    };
    let Ok(m) = serde_json::from_slice::<MapCostFile>(&raw) else {
        return (Vec::new(), false);
    };
    let cells = (w * h) as usize;
    if m.w != w
        || m.h != h
        || m.tiles.len() != cells
        || m.native_composition_event_bytes.len() != cells
        || m.native_tile_blit_modes.len() != cells
    {
        return (Vec::new(), false);
    }
    let mut grid = vec![0u8; 4 + 4 * cells];
    (grid[0], grid[2]) = (w as u8, h as u8);
    for (index, &tile) in m.tiles.iter().enumerate() {
        if !(0..=0xffff).contains(&tile) {
            return (Vec::new(), false);
        }
        let offset = 4 + 4 * index;
        grid[offset..offset + 2].copy_from_slice(&(tile as u16).to_le_bytes());
        let event_word = m.native_composition_event_bytes[index] as u16 | (m.native_tile_blit_modes[index] as u16) << 8;
        grid[offset + 2..offset + 4].copy_from_slice(&event_word.to_le_bytes());
    }
    (grid, true)
}

/// resetNativeMapEventGrid reproduces 0x4dbfc's constructor prefix.  It is
/// intentionally in-place because the original buffer is subsequently owned
/// by event/AI writers; malformed dimensions are rejected by the loader.
///
/// Go：`model.go:1494`，`func resetNativeMapEventGrid(grid []byte)`
pub(crate) fn reset_native_map_event_grid(grid: &mut [u8]) {
    if grid.len() < 4 {
        return;
    }
    let count = grid[0] as usize * grid[2] as usize;
    if grid.len() != 4 + 4 * count {
        return;
    }
    for index in 0..count {
        let offset = 4 + 4 * index;
        grid[offset + 1] &= 0x03;
        grid[offset + 2] &= 0x1f;
        grid[offset + 3] = 0xff;
    }
}

impl State {
    /// ResetNativeMapEventGrid applies 0x4DBFC atomically to the materialized raw
    /// four-byte cell buffer. It is exported only for the evidence-bound ch22
    /// post-battle resource transaction; callers cannot supply normalized cells.
    ///
    /// Go：`model.go:1513`，`func (s *State) ResetNativeMapEventGrid() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn reset_native_map_event_grid(&mut self) -> bool {
        if !self.has_native_map_event_grid
            || self.w <= 0
            || self.h <= 0
            || self.native_map_event_grid.len() as i64
                != 4i64.wrapping_add(4i64.wrapping_mul(self.w).wrapping_mul(self.h))
            || self.native_map_event_grid[0] as i64 != self.w
            || self.native_map_event_grid[2] as i64 != self.h
        {
            return false;
        }
        let mut next = self.native_map_event_grid.clone();
        reset_native_map_event_grid(&mut next);
        self.native_map_event_grid = next;
        true
    }
}

/// Go：`model.go:1525`，`func loadNativeFieldEvents( mapJSONPath string, w, h int, ) ([]int, []NativeFieldEvent, []NativeFieldEventRule)`
pub(crate) fn load_native_field_events(
    map_json_path: &str,
    w: i64,
    h: i64,
) -> (Vec<i64>, Vec<NativeFieldEvent>, Vec<NativeFieldEventRule>) {
    let empty = || (Vec::new(), Vec::new(), Vec::new());
    let Ok(raw) = std::fs::read(map_json_path) else {
        return empty();
    };
    let Ok(m) = serde_json::from_slice::<MapCostFile>(&raw) else {
        return empty();
    };
    if m.w != w
        || m.h != h
        || m.native_field_event_slots.len() as i64 != w.wrapping_mul(h)
        || m.native_field_events.len() != 16
    {
        return empty();
    }
    for &slot in &m.native_field_event_slots {
        if slot < -1 || slot >= m.native_field_events.len() as i64 {
            return empty();
        }
    }
    let mut seen = BTreeSet::new();
    for rule in &m.native_field_event_rules {
        if rule.event_id < 0 || rule.event_id >= 90 || seen.contains(&rule.event_id) {
            return empty();
        }
        if rule.set_state_index.is_none() != rule.set_state_value.is_none() {
            return empty();
        }
        if let (Some(index), Some(value)) = (rule.set_state_index, rule.set_state_value)
            && (!(0..0x20).contains(&index) || !(0..=0xff).contains(&value))
        {
            return empty();
        }
        if rule.once_state.is_some_and(|s| !(0..0x20).contains(&s)) {
            return empty();
        }
        if let Some(a) = &rule.turn_activation
            && (!(0..16).contains(&a.slot) || !(0..90).contains(&a.event_id) || !(0..=0xff).contains(&a.turn_delta))
        {
            return empty();
        }
        seen.insert(rule.event_id);
        for mode_range in &rule.set_mode_ranges {
            if mode_range.start < 0 || mode_range.end < mode_range.start || mode_range.mode > 0x0f {
                return empty();
            }
        }
    }
    (m.native_field_event_slots, m.native_field_events, m.native_field_event_rules)
}

/// loadNativeTerrainRendererInputs accepts only a complete map export. It
/// deliberately keeps malformed data nil rather than repurposing Cost.
///
/// Go：`model.go:1581`，`func loadNativeTerrainRendererInputs(mapJSONPath string, w, h int) ([]byte, []byte, []byte)`
pub(crate) fn load_native_terrain_renderer_inputs(map_json_path: &str, w: i64, h: i64) -> (Vec<u8>, Vec<u8>, Vec<u8>) {
    let empty = || (Vec::new(), Vec::new(), Vec::new());
    let Ok(raw) = std::fs::read(map_json_path) else {
        return empty();
    };
    let Ok(m) = serde_json::from_slice::<MapCostFile>(&raw) else {
        return empty();
    };
    let cells = w.wrapping_mul(h);
    if m.w != w
        || m.h != h
        || m.tiles.len() as i64 != cells
        || m.native_tile_blit_modes.len() as i64 != cells
        || m.native_terrain_control.is_empty()
        || m.native_terrain_control.len() % 4 != 0
    {
        return empty();
    }
    for &tile in &m.tiles {
        if tile < 0 || tile & !0x3ff != 0 || tile >= m.native_terrain_control.len() as i64 / 4 {
            return empty();
        }
    }
    let move_codes = m.tiles.iter().map(|&tile| m.native_terrain_control[tile as usize * 4 + 1]).collect();
    (m.native_tile_blit_modes, m.native_terrain_control, move_codes)
}

impl State {
    /// AddUnit 把一個單位加入戰場(事件 spawn / 主角隊進場用)。 Legacy callers
    /// do not materialize a native selector slot implicitly.
    ///
    /// Go：`model.go:1604`，`func (s *State) AddUnit(u *Unit)`
    /// 所有權：`u` 的單位本體移入 State（Go 把指標 append 進 `Units`／`Roster`）。
    pub fn add_unit(&mut self, u: Unit) {
        self.units.push(u);
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
        let mut units = units;
        self.append_native_map_selector_batch_keep(&mut units)
    }

    /// AdvanceNativeMapPresentationCycles applies one proven 0x1297d call to the
    /// battle-local raw globals. The signed timer value must already be the low
    /// BIOS word observed by the caller; legacy states fail closed.
    ///
    /// Go：`model.go:1643`，`func (s *State) AdvanceNativeMapPresentationCycles(rawTimerTick int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_map_presentation_cycles(&mut self, raw_timer_tick: i64) -> bool {
        if !self.has_native_map_cycle_state || !(-0x8000..=0x7fff).contains(&raw_timer_tick) {
            return false;
        }
        self.native_map_cycle_state =
            fdicon::advance_native_map_sprite_cycles(self.native_map_cycle_state, raw_timer_tick);
        true
    }

    /// AdvanceNativeTerrainPhase applies one proven 0x11eee phase-selection call.
    /// override is raw [0x51a93]: -1 for the BIOS-timed path or 0..19 for the
    /// explicit selector path.
    ///
    /// Go：`model.go:1654`，`func (s *State) AdvanceNativeTerrainPhase(rawTimerTick, override int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_terrain_phase(&mut self, raw_timer_tick: i64, r#override: i64) -> bool {
        if !self.has_native_terrain_phase_state {
            return false;
        }
        match fdother::advance_native_terrain_phase(self.native_terrain_phase_state, raw_timer_tick, r#override) {
            Ok(next) => {
                self.native_terrain_phase_state = next;
                true
            }
            Err(_) => false,
        }
    }

    /// Go：`model.go:1666`，`func (s *State) AdvanceNativeTerrainFlip(rawTimerTick int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_terrain_flip(&mut self, raw_timer_tick: i64) -> bool {
        if !self.has_native_map_binary_timing_state {
            return false;
        }
        match fdicon::advance_native_binary_tick(self.native_terrain_flip_state, raw_timer_tick) {
            Ok(next) => {
                self.native_terrain_flip_state = next;
                true
            }
            Err(_) => false,
        }
    }

    /// Go：`model.go:1678`，`func (s *State) AdvanceNativeUnitPixelShift(rawTimerTick int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_unit_pixel_shift(&mut self, raw_timer_tick: i64) -> bool {
        if !self.has_native_map_binary_timing_state {
            return false;
        }
        match fdicon::advance_native_binary_tick(self.native_unit_pixel_shift_state, raw_timer_tick) {
            Ok(next) => {
                self.native_unit_pixel_shift_state = next;
                true
            }
            Err(_) => false,
        }
    }

    /// AppendNativeMapSelectorBatchOrLegacy attempts the evidenced constructor
    /// contract, but preserves playable legacy construction if editable source data
    /// is malformed.  The failure is retained on State so renderers cannot quietly
    /// claim a partial native selector state.
    ///
    /// Go：`model.go:1694`，`func (s *State) AppendNativeMapSelectorBatchOrLegacy(units []*Unit)`
    /// 所有權：`units` 的單位本體移入 State（Go 把指標 append 進 `Units`／`Roster`）。
    pub fn append_native_map_selector_batch_or_legacy(&mut self, units: Vec<Unit>) {
        let mut units = units;
        if let Err(err) = self.append_native_map_selector_batch_keep(&mut units) {
            self.native_map_selector_error = Some(err);
            self.units.append(&mut units);
        }
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
        let Some(cache) = self.native_map_selector_cache.as_ref() else {
            return (0, false);
        };
        let Some(u) = u.map(|i| &self.units[i]) else {
            return (0, false);
        };
        if self.native_map_selector_error.is_some() || !u.has_map_selector_slot {
            return (0, false);
        }
        match cache.key_for_slot(u.map_selector_slot) {
            Ok(key) => (key, true),
            Err(_) => (0, false),
        }
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
        if self.roster.is_empty() {
            return 0;
        }
        let (mut batch, remaining): (Vec<Unit>, Vec<Unit>) =
            std::mem::take(&mut self.roster).into_iter().partition(|u| u.group == group);
        for u in &mut batch {
            u.on_field = true;
            (u.off_x, u.off_y) = (0.0, 0.0);
        }
        self.roster = remaining;
        if batch.is_empty() {
            return 0;
        }
        let n = batch.len() as i64;
        if self.native_map_selector_cache.is_some() && self.native_map_selector_error.is_none() {
            self.append_native_map_selector_batch_or_legacy(batch);
        } else {
            self.units.append(&mut batch);
        }
        n
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
        if self.roster.is_empty() {
            return Err(battle_err!("native future group {group}: runtime roster unavailable"));
        }
        if self.native_future_item_rows.is_empty() {
            return Err(battle_err!("native future group {group}: item effect rows unavailable"));
        }
        if self.native_map_selector_error.is_some()
            || (!self.units.is_empty() && self.native_map_selector_cache.is_none())
        {
            return Err(battle_err!("native future group {group}: selector state unavailable"));
        }
        let batch: Vec<&Unit> = self.roster.iter().filter(|u| u.group == group).collect();
        if batch.is_empty() {
            return Err(battle_err!("native future group {group}: no matching FDFIELD rows"));
        }

        // Go 的 prospective 與 shadow 共用指標；Rust 以複本試算，整批成功才寫回（見 battle 模組說明第 5 點）。
        let mut prospective = self.units.clone();
        let mut prepared = Vec::with_capacity(batch.len());
        for (i, unit) in batch.into_iter().enumerate() {
            if !unit.has_native_position_record {
                return Err(battle_err!("native future group {group} row {i}: six-byte position record unavailable"));
            }
            let cell = native_future_group_placement(
                self.w,
                self.h,
                &self.native_composition_event_bytes,
                &prospective,
                unit.native_position_record,
                raw_gate,
            )
            .map_err(|e| e.wrap(format!("native future group {group} row {i}: ")))?;
            let mut shadow = unit.clone();
            materialize_native_future_constructor(Some(&mut shadow), &self.native_future_item_rows)
                .map_err(|e| e.wrap(format!("native future group {group} row {i}: ")))?;
            if !shadow.set_map_placement(cell.x, cell.y, 0) {
                return Err(battle_err!("native future group {group} row {i}: invalid placement"));
            }
            // The next row observes this just-constructed runtime record through
            // 0x145CD. Materialize only the local shadow during preflight; the real
            // unit remains untouched until the whole batch succeeds.
            shadow
                .materialize_native_map_presentation()
                .map_err(|e| e.wrap(format!("native future group {group} row {i}: ")))?;
            shadow.on_field = true;
            prospective.push(shadow.clone());
            prepared.push(shadow);
        }

        let mut candidate = self.clone();
        candidate.roster = Vec::new();
        let appended = prepared.len() as i64;
        candidate
            .append_native_map_selector_batch(prepared)
            .map_err(|e| e.wrap(format!("native future group {group} selector commit: ")))?;

        self.units = candidate.units;
        self.roster.retain(|u| u.group != group);
        self.native_map_selector_cache = candidate.native_map_selector_cache;
        self.native_map_selector_error = None;
        self.native_map_cycle_state = candidate.native_map_cycle_state;
        self.has_native_map_cycle_state = candidate.has_native_map_cycle_state;
        self.native_terrain_phase_state = candidate.native_terrain_phase_state;
        self.has_native_terrain_phase_state = candidate.has_native_terrain_phase_state;
        self.native_terrain_flip_state = candidate.native_terrain_flip_state;
        self.native_unit_pixel_shift_state = candidate.native_unit_pixel_shift_state;
        self.has_native_map_binary_timing_state = candidate.has_native_map_binary_timing_state;
        Ok(appended)
    }

    /// SpawnGroup 讓既有重製劇本的 group 登場，可另行覆寫陣營與本回合行動狀態。
    /// 它保留舊的滑入與環狀錯位呈現，不等同原版 0x10b4e/0x10c50 constructor。
    ///
    /// Go：`model.go:1843`，`func (s *State) SpawnGroup(group int, camp Camp, changeCamp, act bool) int`
    pub fn spawn_group(&mut self, group: i64, camp: Camp, change_camp: bool, act: bool) -> i64 {
        let mut n = 0;
        let before = self.units.len();
        if self.append_group(group) > 0 {
            for i in before..self.units.len() {
                self.spawn_group_place(i, camp, change_camp, act);
                n += 1;
            }
            return n;
        }
        for i in 0..self.units.len() {
            if self.units[i].group == group && !self.units[i].on_field {
                self.units[i].on_field = true;
                // 增援進場:從上方滑入(spawn_march)
                self.spawn_group_place(i, camp, change_camp, act);
                n += 1;
            }
        }
        n
    }

    /// nearestFree 是舊重製呈現使用的有界環狀搜尋；它不是原版
    /// 0x10cc6..0x10d4c 的全圖 row-major Manhattan 規則。
    ///
    /// Go：`model.go:1883`，`func (s *State) nearestFree(x, y int) (Cell, bool)`
    pub(crate) fn nearest_free(&self, x: i64, y: i64) -> (Cell, bool) {
        for r in 1..8i64 {
            for dy in -r..=r {
                for dx in -r..=r {
                    if dx > -r && dx < r && dy > -r && dy < r {
                        continue; // 只看環邊
                    }
                    let (nx, ny) = (x + dx, y + dy);
                    if nx < 0 || ny < 0 || nx >= self.w || ny >= self.h {
                        continue;
                    }
                    if self.unit_at(nx, ny).is_none() {
                        return (Cell { x: nx, y: ny }, true);
                    }
                }
            }
        }
        (Cell::default(), false)
    }
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct UnitsFileChest {
    /// Go 標籤：`json:"slot"`
    pub slot: i64,
    /// Go 標籤：`json:"type"`
    #[serde(rename = "type", deserialize_with = "null_as_default")]
    pub kind: String,
    /// Go 標籤：`json:"native_type"`
    pub native_type: u8,
    /// Go 標籤：`json:"value"`
    pub value: i64,
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct UnitsFileUnit {
    /// Go 標籤：`json:"camp"`
    #[serde(deserialize_with = "null_as_default")]
    pub camp: String,
    /// Go 標籤：`json:"cls"`
    #[serde(rename = "cls")]
    pub class_id: i64,
    /// Go 標籤：`json:"name"`
    #[serde(deserialize_with = "null_as_default")]
    pub name: String,
    /// Go 標籤：`json:"cls_name"`
    #[serde(deserialize_with = "null_as_default")]
    pub cls_name: String,
    /// Go 標籤：`json:"lv"`
    pub lv: i64,
    /// Go 標籤：`json:"hp"`
    pub hp: i64,
    /// Go 標籤：`json:"mp"`
    pub mp: i64,
    /// Go 標籤：`json:"spells"`
    #[serde(deserialize_with = "null_as_default")]
    pub spells: Vec<i64>,
    /// Go 標籤：`json:"initial_command_mask,omitempty"`
    #[serde(with = "go_bytes", skip_serializing_if = "Vec::is_empty")]
    pub initial_command_mask: Vec<u8>,
    /// Go 標籤：`json:"inventory,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub inventory: Vec<i64>,
    /// Go 標籤：`json:"inventory_slots,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub inventory_slots: Vec<i64>,
    /// Go 標籤：`json:"death_effect,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub death_effect: Option<DeathEffect>,
    /// Go 標籤：`json:"death_reward,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
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
    #[serde(skip_serializing_if = "Option::is_none")]
    pub battle_fig: Option<i64>,
    /// Go 標籤：`json:"native_identity,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_identity: Option<i64>,
    /// Go 標籤：`json:"native_record_byte8,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_byte8: Option<u8>,
    /// Go 標籤：`json:"native_position_record,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_position_record: Option<UnitsFileUnitNativePositionRecord>,
    /// Go 標籤：`json:"native_record_race,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_race: Option<u8>,
    /// Go 標籤：`json:"native_record_class,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_class: Option<u8>,
    /// Go 標籤：`json:"map_selector_slot,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub map_selector_slot: Option<i64>,
    /// Go 標籤：`json:"map_selector_key,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub map_selector_key: Option<i64>,
    /// Go 標籤：`json:"native_record_byte5,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_byte5: Option<u8>,
    /// Go 標籤：`json:"native_record_byte6,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_byte6: Option<u8>,
    /// Go 標籤：`json:"native_record_byte34,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_byte34: Option<u8>,
    /// Go 標籤：`json:"native_record_byte35,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_byte35: Option<u8>,
    /// Go 標籤：`json:"native_record_byte36,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_byte36: Option<u8>,
    /// Go 標籤：`json:"native_record_byte3d,omitempty"`
    #[serde(rename = "native_record_byte3d", skip_serializing_if = "Option::is_none")]
    pub native_record_byte3_d: Option<u8>,
    /// Go 標籤：`json:"native_record_death_effect,omitempty"`
    #[serde(with = "go_bytes", skip_serializing_if = "Vec::is_empty")]
    pub native_record_death_effect: Vec<u8>,
    /// Go 標籤：`json:"native_source_byte3,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_source_byte3: Option<u8>,
    /// Go 標籤：`json:"native_source_byte20,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_source_byte20: Option<u8>,
    /// Go 標籤：`json:"native_source_byte25,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_source_byte25: Option<u8>,
    /// Go 標籤：`json:"native_record_word42,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_word42: Option<u16>,
    /// Go 標籤：`json:"native_record_word46,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_word46: Option<u16>,
    /// Go 標籤：`json:"group"`
    pub group: i64,
    /// Go 標籤：`json:"native_constructor,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_constructor: Option<NativeConstructorTable>,
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
}

/// Go 版此處是匿名 struct（field）；Rust 需要具名型別，名稱由骨架產生。
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct UnitsFileUnitNativePositionRecord {
    /// Go 標籤：`json:"x_word"`
    pub x_word: u16,
    /// Go 標籤：`json:"y_word"`
    pub y_word: u16,
    /// Go 標籤：`json:"raw_key"`
    pub raw_key: u16,
}

impl State {
    /// `AppendNativeMapSelectorBatch` 的本體。Go 以指標切片傳入，失敗時呼叫端（`OrLegacy`）仍握有
    /// 那批單位；Rust 改借用 `Vec`，成功才把單位移進 `self.units`，失敗時原樣留給呼叫端。
    ///
    /// Go 的快取是指標：已有快取時就地累積（`SlotFor` 中途失敗也保留已配置的槽位），沒有快取時
    /// 用新快取，失敗就丟棄。這裡照同樣的規則處理。
    fn append_native_map_selector_batch_keep(&mut self, units: &mut Vec<Unit>) -> Result<()> {
        let had_cache = self.native_map_selector_cache.is_some();
        let cache = self.native_map_selector_cache.get_or_insert_with(Default::default);
        if let Err(err) = materialize_native_map_selector_slots(units, Some(cache)) {
            if !had_cache {
                self.native_map_selector_cache = None;
            }
            return Err(err);
        }
        if !self.has_native_map_cycle_state {
            self.native_map_cycle_state = Default::default();
            self.has_native_map_cycle_state = true;
        }
        if !self.has_native_terrain_phase_state {
            self.native_terrain_phase_state = Default::default();
            self.has_native_terrain_phase_state = true;
        }
        if !self.has_native_map_binary_timing_state {
            self.native_terrain_flip_state = Default::default();
            self.native_unit_pixel_shift_state = Default::default();
            self.has_native_map_binary_timing_state = true;
        }
        self.units.append(units);
        Ok(())
    }

    /// `SpawnGroup` 兩個迴圈共用的單位處理（Go 寫成兩段相同的程式）。
    fn spawn_group_place(&mut self, i: UnitIdx, camp: Camp, change_camp: bool, act: bool) {
        let u = &mut self.units[i];
        u.off_y = -56.0;
        if change_camp {
            u.camp = camp;
        }
        u.acted = !act; // act=true → 可行動;否則標記已行動(下回合才動)
        let (x, y, dir) = (u.x, u.y, u.dir);
        if self.unit_at(x, y).is_some_and(|occ| occ != i) {
            let (c, ok) = self.nearest_free(x, y);
            if ok {
                self.units[i].set_map_placement(c.x, c.y, dir);
            }
        }
    }
}

/// Go `filepath.Dir`（只處理 `battle` 用到的相對／絕對路徑；`Dir("x")` 是 `"."`）。
fn go_dir(path: &str) -> String {
    match std::path::Path::new(path).parent() {
        Some(p) if !p.as_os_str().is_empty() => p.to_string_lossy().into_owned(),
        Some(_) => ".".to_string(),
        None => path.to_string(),
    }
}

/// Go `filepath.Base`。
fn go_base(path: &str) -> &str {
    std::path::Path::new(path).file_name().and_then(|s| s.to_str()).unwrap_or(".")
}

/// Go `filepath.Join(dir, name)`（`dir` 為 `"."` 時只回 `name`）。
fn go_join(dir: &str, name: &str) -> String {
    if dir == "." { name.to_string() } else { std::path::Path::new(dir).join(name).to_string_lossy().into_owned() }
}

/// Go `fmt.Sscanf(name, "map%d", &mapIndex)` 成功時的值：字面 `map` 之後可有空白與正負號，再接至少
/// 一位十進位數字，之後的字元忽略。Go `fmt` 掃描的其餘細節（底線、`0x` 前綴等）不移植。
fn go_scan_map_index(name: &str) -> Option<i64> {
    let rest = name.strip_prefix("map")?.trim_start_matches([' ', '\t']);
    let (neg, digits) = match rest.as_bytes().first() {
        Some(b'-') => (true, &rest[1..]),
        Some(b'+') => (false, &rest[1..]),
        _ => (false, rest),
    };
    let end = digits.bytes().take_while(u8::is_ascii_digit).count();
    let value: i64 = digits[..end].parse().ok()?;
    Some(if neg { -value } else { value })
}

/// 小寫十六進位 SHA-256（Go `fmt.Sprintf("%x", sha256.Sum256(raw))`）。
fn sha256_hex(raw: &[u8]) -> String {
    use sha2::Digest as _;
    sha2::Sha256::digest(raw).iter().map(|b| format!("{b:02x}")).collect()
}

/// `Unit::enemy_balance_base` 的 JSON：Go `*BalanceStats` 沒有標籤，寫成
/// `{"HP":..,"MP":..,"AP":..,"DP":..,"DX":..}`。`BalanceStats` 在 B 組的 `balance.rs`，A 組不改
/// 該檔，所以在這裡用鏡像結構轉換。
mod go_balance_stats {
    use super::BalanceStats;
    use serde::{Deserialize, Deserializer, Serialize, Serializer};

    #[derive(Serialize, Deserialize, Default)]
    #[serde(default)]
    struct Mirror {
        #[serde(rename = "HP")]
        hp: i64,
        #[serde(rename = "MP")]
        mp: i64,
        #[serde(rename = "AP")]
        ap: i64,
        #[serde(rename = "DP")]
        dp: i64,
        #[serde(rename = "DX")]
        dx: i64,
    }

    pub(super) fn serialize<S: Serializer>(v: &Option<BalanceStats>, s: S) -> Result<S::Ok, S::Error> {
        v.map(|b| Mirror { hp: b.hp, mp: b.mp, ap: b.ap, dp: b.dp, dx: b.dx }).serialize(s)
    }

    pub(super) fn deserialize<'de, D: Deserializer<'de>>(d: D) -> Result<Option<BalanceStats>, D::Error> {
        Ok(Option::<Mirror>::deserialize(d)?.map(|m| BalanceStats { hp: m.hp, mp: m.mp, ap: m.ap, dp: m.dp, dx: m.dx }))
    }
}

#[cfg(test)]
mod model_test;

#[cfg(test)]
mod native_death_presentation_flag_test;

#[cfg(test)]
mod treasure_test;
