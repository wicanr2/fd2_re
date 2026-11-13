//! Go 來源：`remake/internal/battle/event.go`（1011 行）。
//!
//! event.go — 可擴充事件系統(doc 29)。
//!
//! 設計三層:trigger(觸發時機)→ when(條件)→ do(動作序列)。原版 FD2 事件寫死在
//! FDFIELD.DAT turn_events + EXE handler(doc 25);remake 全資料化成 scenario JSON,
//! 引擎不為任何關卡寫死分支。新增條件/動作 = 在下方 switch 加一個 case。

use super::*;

/// Scenario 一關的劇本(對映原版 FDFIELD turn_events + 青衫 ground truth)。
///
/// Go：`event.go:15`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Scenario {
    /// Go 標籤：`json:"chapter"`
    pub chapter: i64,
    /// Go 標籤：`json:"name"`
    #[serde(deserialize_with = "null_as_default")]
    pub name: String,
    /// Go 標籤：`json:"map"`
    pub map: i64,
    /// party first; FDFIELD groups append only when constructed
    /// Go 標籤：`json:"runtime_append_groups,omitempty"`
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    pub runtime_append_groups: bool,
    /// asset-root-relative decoded 0x1366A resource set for native battle events
    /// Go 標籤：`json:"native_acting_resources,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "String::is_empty")]
    pub native_acting_resources: String,
    /// 開局即在場的 unit group;其餘待命
    /// Go 標籤：`json:"initial_groups"`
    #[serde(deserialize_with = "null_as_default")]
    pub initial_groups: Vec<i64>,
    /// native pre-handler conditional FDFIELD group
    /// Go 標籤：`json:"initial_groups_if_party_absent,omitempty"`
    #[serde(
        rename = "initial_groups_if_party_absent",
        deserialize_with = "null_as_default",
        skip_serializing_if = "Vec::is_empty"
    )]
    pub initial_groups_absent: Vec<InitialGroupAbsent>,
    /// 主角隊(不在 FDFIELD roster,on_battle_start 進場)
    /// Go 標籤：`json:"party"`
    #[serde(deserialize_with = "null_as_default")]
    pub party: Vec<PartyMember>,
    /// 主角隊進場目標格
    /// Go 標籤：`json:"deploy_cells"`
    #[serde(deserialize_with = "go_int_arrays")]
    pub deploy_cells: Vec<[i64; 2]>,
    /// Go 標籤：`json:"events"`
    #[serde(deserialize_with = "null_as_default")]
    pub events: Vec<Event>,
    /// Go 標籤：`json:"native_field_event_rules,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub native_field_event_rules: Vec<NativeFieldEventRule>,
    /// Go 標籤：`json:"native_turn_events,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub native_turn_events: Vec<NativeTurnEvent>,
    /// NativeDeathPrograms 是死亡效果型態 2／3 降成的動作清單，鍵是「型態:值」。
    /// 型態2亦是全域0x51B91事件程式；已審查的selector0行動收尾共用它，不加死亡delay。
    /// 來源與逐動作的原始位址見 native_death_program.go。
    /// Go 標籤：`json:"native_death_programs,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "BTreeMap::is_empty")]
    pub native_death_programs: BTreeMap<String, Vec<Action>>,
    /// NativeResultHandler／NativeResultCode1Records 是非 default 的章節勝負 handler：先跑
    /// 0x205BE 的三值規則，再對列出的記錄查 raw +5 bit0（0x3453E），任一筆成立就寫
    /// [0x53ECC]=1——與 default「記錄 0 倒下」同一個碼。第十章 0x20707 查記錄 50、51
    /// （docs/knowledge-base/26-per-chapter-event-handlers.md）。
    /// Go 標籤：`json:"native_result_handler,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "String::is_empty")]
    pub native_result_handler: String,
    /// Go 標籤：`json:"native_result_initial_code,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_result_initial_code: Option<i64>,
    /// Go 標籤：`json:"native_result_auto_advance,omitempty"`
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    pub native_result_auto_advance: bool,
    /// Go 標籤：`json:"native_result_code1_records,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub native_result_code1_records: Vec<i64>,
    /// Go 標籤：`json:"native_result_rules,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub native_result_rules: Vec<NativeResultRule>,
    #[serde(skip)]
    pub(crate) pending_joins: Vec<i64>,
}

impl Scenario {
    /// NativeResultCode1 回報章節 handler 列出的記錄是否有任一筆 raw +5 bit0 成立。還沒建立的
    /// 記錄（索引超出記錄表）原版讀到的是未使用槽，不算。
    ///
    /// Go：`event.go:46`，`func (sc *Scenario) NativeResultCode1(st *State) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_result_code1(&self, st: Option<&State>) -> bool {
        let Some(st) = st else {
            return false;
        };
        for &index in &self.native_result_code1_records {
            if index < 0 || index >= st.units.len() as i64 {
                continue;
            }
            let u = &st.units[index as usize];
            if u.has_native_record_byte5 && u.native_record_byte5 & 1 != 0 {
                return true;
            }
        }
        false
    }
}

/// NativeTurnEvent preserves one live FDFIELD three-byte row consumer without
/// forcing it into the normalized on_turn_end trigger. RawCamp is deliberately
/// not renamed to a faction: sub_1A813 compares the byte at row+5, and each
/// caller owns a different phase boundary.
///
/// Go：`event.go:65`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeTurnEvent {
    /// Go 標籤：`json:"event_id"`
    pub event_id: i64,
    /// Go 標籤：`json:"raw_camp"`
    pub raw_camp: i64,
    /// Go 標籤：`json:"handler"`
    #[serde(deserialize_with = "null_as_default")]
    pub handler: String,
    /// Go 標籤：`json:"staging"`
    pub staging: NativeTurnStaging,
    /// Go 標籤：`json:"dynamic_group,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub dynamic_group: Option<NativeTurnDynamicGroup>,
    /// Go 標籤：`json:"progression,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub progression: Option<NativeTurnProgression>,
    /// Go 標籤：`json:"pair_mutation,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub pair_mutation: Option<NativeTurnPairMutation>,
    /// 原始state索引、已證實下限及上限。
    /// Go 標籤：`json:"state_range,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none", deserialize_with = "go_int_array_opt")]
    pub state_range: Option<[i64; 3]>,
    /// Actions 是控制列事件轉寫後降階的劇本動作（tools/sync_native_turn_events.py 由
    /// native_death_events.json 產生），依 0x1A813 的 selector 在該階段的可編輯事件之後執行。
    /// 用於由其他處理器 control_turn 動態啟用、不在靜態 turn_events 裡的列（第九章事件 31）。
    /// Go 標籤：`json:"actions,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub actions: Vec<Action>,
}

/// NativeTurnDynamicGroup 從 raw event-state table 解析一個 0x35822 group 參數，
/// 再套用該 caller 已證實的 live-row／state writes；它與 event63 這類固定 staging
/// calls 分開表示。
///
/// Go：`event.go:83`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeTurnDynamicGroup {
    /// Go 標籤：`json:"state_index"`
    pub state_index: i64,
    /// Go 標籤：`json:"minimum"`
    pub minimum: i64,
    /// Go 標籤：`json:"maximum"`
    pub maximum: i64,
    /// Go 標籤：`json:"control_slot"`
    pub control_slot: i64,
    /// Go 標籤：`json:"reschedule_delta"`
    pub reschedule_delta: i64,
    /// Go 標籤：`json:"stop_value"`
    pub stop_value: i64,
    /// Go 標籤：`json:"increment"`
    pub increment: i64,
}

/// NativeTurnProgression 保存 event76 這類由 raw event-state byte 推進、最後才
/// materialize group 並啟動另一 live row 的可編輯規則；欄位名只描述運算，不替
/// state byte 或 RawCamp 猜測玩法語意。
///
/// Go：`event.go:96`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeTurnProgression {
    /// Go 標籤：`json:"state_index"`
    pub state_index: i64,
    /// Go 標籤：`json:"repeat_until"`
    pub repeat_until: i64,
    /// Go 標籤：`json:"mark_unit_index"`
    pub mark_unit_index: i64,
    /// Go 標籤：`json:"mark_helper"`
    #[serde(deserialize_with = "null_as_default")]
    pub mark_helper: String,
    /// Go 標籤：`json:"control_slot"`
    pub control_slot: i64,
    /// Go 標籤：`json:"reschedule_delta"`
    pub reschedule_delta: i64,
    /// Go 標籤：`json:"final_text_index"`
    pub final_text_index: i64,
    /// Go 標籤：`json:"spawn_group"`
    pub spawn_group: i64,
    /// Go 標籤：`json:"spawn_helper"`
    #[serde(deserialize_with = "null_as_default")]
    pub spawn_helper: String,
    /// Go 標籤：`json:"raw_placement_gate"`
    pub raw_placement_gate: i64,
    /// Go 標籤：`json:"spawn_count"`
    pub spawn_count: i64,
    /// Go 標籤：`json:"base_state_index"`
    pub base_state_index: i64,
    /// Go 標籤：`json:"final_activation"`
    pub final_activation: NativeTurnActivation,
    /// Go 標籤：`json:"pulse_handler"`
    #[serde(deserialize_with = "null_as_default")]
    pub pulse_handler: String,
    /// Go 標籤：`json:"pulse_count"`
    pub pulse_count: i64,
    /// Go 標籤：`json:"extra_delay_ms"`
    pub extra_delay_ms: i64,
    /// Go 標籤：`json:"tail_text_indices"`
    #[serde(deserialize_with = "null_as_default")]
    pub tail_text_indices: Vec<i64>,
}

/// NativeTurnPairMutation 保存 event79 由一個 raw base 與單一步 RNG 選出兩個
/// 循環相鄰 runtime slots 的規則，不替 bit7 賦予 caller 之外的玩法名稱。
///
/// Go：`event.go:118`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeTurnPairMutation {
    /// Go 標籤：`json:"base_state_index"`
    pub base_state_index: i64,
    /// Go 標籤：`json:"group"`
    pub group: i64,
    /// Go 標籤：`json:"count"`
    pub count: i64,
    /// Go 標籤：`json:"modulo"`
    pub modulo: i64,
    /// Go 標籤：`json:"second_offset"`
    pub second_offset: i64,
    /// Go 標籤：`json:"mark_helper"`
    #[serde(deserialize_with = "null_as_default")]
    pub mark_helper: String,
    /// Go 標籤：`json:"control_slot"`
    pub control_slot: i64,
    /// Go 標籤：`json:"reschedule_delta"`
    pub reschedule_delta: i64,
}

/// NativeTurnStaging is the editable form of the recovered 0x35822 helper.
/// Helper addresses remain provenance, not runtime dispatch keys: the UI
/// adapter validates the complete known signature before executing it.
///
/// Go：`event.go:132`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeTurnStaging {
    /// Go 標籤：`json:"helper"`
    #[serde(deserialize_with = "null_as_default")]
    pub helper: String,
    /// Go 標籤：`json:"pan_helper"`
    #[serde(deserialize_with = "null_as_default")]
    pub pan_helper: String,
    /// Go 標籤：`json:"spawn_helper"`
    #[serde(deserialize_with = "null_as_default")]
    pub spawn_helper: String,
    /// Go 標籤：`json:"delay_before_flash_ms"`
    pub delay_before_flash_ms: i64,
    /// Go 標籤：`json:"palette_helper"`
    #[serde(deserialize_with = "null_as_default")]
    pub palette_helper: String,
    /// Go 標籤：`json:"palette_start"`
    pub palette_start: i64,
    /// Go 標籤：`json:"palette_end"`
    pub palette_end: i64,
    /// Go 標籤：`json:"flash_delta"`
    pub flash_delta: i64,
    /// Go 標籤：`json:"flash_hold_ms"`
    pub flash_hold_ms: i64,
    /// Go 標籤：`json:"restore_delta"`
    pub restore_delta: i64,
    /// Go 標籤：`json:"redraw_helper"`
    #[serde(deserialize_with = "null_as_default")]
    pub redraw_helper: String,
    /// Go 標籤：`json:"raw_placement_gate"`
    pub raw_placement_gate: i64,
    /// Go 標籤：`json:"calls"`
    #[serde(deserialize_with = "null_as_default")]
    pub calls: Vec<NativeTurnStagingCall>,
}

/// Go：`event.go:148`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeTurnStagingCall {
    /// Go 標籤：`json:"group"`
    pub group: i64,
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
    /// Go 標籤：`json:"source"`
    #[serde(deserialize_with = "null_as_default")]
    pub source: String,
}

/// InitialGroupAbsent is an evidence-backed pre-battle condition: materialize
/// its FDFIELD group only when the permanent party lacks CharID. It is not a
/// generic event expression and does not claim runtime-slot append identity.
///
/// Go：`event.go:158`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct InitialGroupAbsent {
    /// Go 標籤：`json:"char_id"`
    pub char_id: i64,
    /// Go 標籤：`json:"group"`
    pub group: i64,
}

/// PartyMember 主角隊成員(數值來自 characters.json / EXE 表)。
///
/// Go：`event.go:164`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct PartyMember {
    /// Go 標籤：`json:"name"`
    #[serde(deserialize_with = "null_as_default")]
    pub name: String,
    /// Go 標籤：`json:"cls"`
    #[serde(deserialize_with = "null_as_default")]
    pub cls: String,
    /// Fig is the stable JOIN/roster identity used by the remake's persistent
    /// party map. For a fresh native JOIN record it also seeds raw +7, but it is
    /// not the mutable map/battle selector after class change.
    /// Go 標籤：`json:"fig"`
    pub fig: i64,
    /// NativeIdentity is the optional persistent-record +0x08 key used by
    /// native 0x11506. It is intentionally separate from Fig/+7 selectors.
    /// Go 標籤：`json:"native_identity,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_identity: Option<i64>,
    /// Raw JOIN-constructor bytes written to persistent +0x1f/+0x20.
    /// Go 標籤：`json:"native_record_race,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_race: Option<u8>,
    /// Go 標籤：`json:"native_record_class,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_class: Option<u8>,
    /// Go 標籤：`json:"portrait"`
    pub portrait: i64,
    /// Go 標籤：`json:"hp"`
    pub hp: i64,
    /// Go 標籤：`json:"mp"`
    pub mp: i64,
    /// Go 標籤：`json:"ap"`
    pub ap: i64,
    /// Go 標籤：`json:"dp"`
    pub dp: i64,
    /// projected into +0x3e; direct-raw provenance must be stated separately
    /// Go 標籤：`json:"dx"`
    pub dx: i64,
    /// 命中(doc32:DX+起始武器HIT增值,對照orig_07_unit_status.png逐位驗證)
    /// Go 標籤：`json:"hit"`
    pub hit: i64,
    /// 閃避(doc32:DX+起始防具EV增值;起始4件防具EV增值皆為0)
    /// Go 標籤：`json:"ev"`
    pub ev: i64,
    /// 暴擊率(resist_crit.json 依角色職業)
    /// Go 標籤：`json:"crit"`
    #[serde(rename = "crit")]
    pub crit_pct: i64,
    /// Go 標籤：`json:"mv"`
    pub mv: i64,
    /// 攻擊距離下限(0=預設1;doc32 weapon_range.json)
    /// Go 標籤：`json:"atk_min"`
    pub atk_min: i64,
    /// 攻擊距離上限(0=預設1;如亞雷斯騎士槍type3=2)
    /// Go 標籤：`json:"atk_max"`
    pub atk_max: i64,
    /// Go 標籤：`json:"lv"`
    pub lv: i64,
    /// 已習得法術 id(spell.json)
    /// Go 標籤：`json:"spells"`
    #[serde(deserialize_with = "null_as_default")]
    pub spells: Vec<i64>,
    /// InitialCommandMask is the exact four-byte constructor source for
    /// unit+0x1a..+0x1d. It is deliberately not derived from Spells.
    /// Go 標籤：`json:"initial_command_mask,omitempty"`
    #[serde(with = "go_bytes", skip_serializing_if = "Vec::is_empty")]
    pub initial_command_mask: Vec<u8>,
    /// Go 標籤：`json:"inventory,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub inventory: Vec<i64>,
    /// Go 標籤：`json:"inventory_slots,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub inventory_slots: Vec<i64>,
}

/// Event 一條事件規則。
///
/// Go：`event.go:199`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Event {
    /// Go 標籤：`json:"id"`
    #[serde(deserialize_with = "null_as_default")]
    pub id: String,
    /// on_battle_start / on_turn_end / on_unit_death
    /// Go 標籤：`json:"trigger"`
    #[serde(deserialize_with = "null_as_default")]
    pub trigger: String,
    /// Go 標籤：`json:"when,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub when: Option<When>,
    /// Go 標籤：`json:"do"`
    #[serde(deserialize_with = "null_as_default")]
    pub r#do: Vec<Action>,
    /// Go 標籤：`json:"once"`
    pub once: bool,
    #[serde(skip)]
    pub(crate) fired: bool,
}

/// When 條件(可擴充:加欄位 + Match 加判斷)。
///
/// Go：`event.go:209`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct When {
    /// turn == N(0=不限)
    /// Go 標籤：`json:"turn,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub turn: i64,
    /// 某角色陣亡
    /// Go 標籤：`json:"unit_dead,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "String::is_empty")]
    pub unit_dead: String,
    /// 原版 runtime slot 已登場且存活
    /// Go 標籤：`json:"unit_slot_active,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unit_slot_active: Option<i64>,
    /// battle-local raw byte index
    /// Go 標籤：`json:"native_event_state_index,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_event_state_index: Option<i64>,
    /// exact raw byte value
    /// Go 標籤：`json:"native_event_state_value,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_event_state_value: Option<i64>,
}

/// Action 動作(可擴充:加 type + execAction 加 case)。
///
/// Go：`event.go:218`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Action {
    /// Go 標籤：`json:"type"`
    #[serde(deserialize_with = "null_as_default")]
    pub r#type: String,
    /// spawn_group 的波次
    /// Go 標籤：`json:"groups,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub groups: Vec<i64>,
    /// turn_events 的原版事件編號
    /// Go 標籤：`json:"native_event_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_event_id: Option<i64>,
    /// 原版逐呼叫配置資料
    /// Go 標籤：`json:"native_spawns,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub native_spawns: Vec<NativeSpawnCall>,
    /// 增援陣營(改為)
    /// Go 標籤：`json:"camp,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "String::is_empty")]
    pub camp: String,
    /// 增援當回合可動(青衫「立即行動」)
    /// Go 標籤：`json:"act_immediately,omitempty"`
    #[serde(skip_serializing_if = "std::ops::Not::not")]
    pub act_immediately: bool,
    /// dialogue 說話者(DATO 肖像 id;-1=旁白)
    /// Go 標籤：`json:"speaker"`
    pub speaker: i64,
    /// dialogue 文本
    /// Go 標籤：`json:"text,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "String::is_empty")]
    pub text: String,
    /// set_flag
    /// Go 標籤：`json:"flag,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "String::is_empty")]
    pub flag: String,
    /// set_ai 目標
    /// Go 標籤：`json:"unit,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "String::is_empty")]
    pub unit: String,
    /// set_ai 模式(berserk…)
    /// Go 標籤：`json:"mode,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "String::is_empty")]
    pub mode: String,
    /// join_party: permanent player identity
    /// Go 標籤：`json:"char_id,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub char_id: i64,
    /// pan:原版 camera grid(col,row)，runtime 依地圖 tile 尺寸換 pixel
    /// Go 標籤：`json:"grid,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none", deserialize_with = "go_int_array_opt")]
    pub grid: Option<[i64; 2]>,
    /// delay:原版毫秒數
    /// Go 標籤：`json:"ms,omitempty"`
    #[serde(skip_serializing_if = "go_is_zero")]
    pub ms: i64,
    /// NativeActing 保存戰鬥事件呼叫 0x1366A 的資源與原始呼叫點。
    /// 它只描述已解碼的行為，不以位址作為重製端執行鍵。
    /// Go 標籤：`json:"native_acting,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_acting: Option<NativeFollowingActing>,
    /// Go 標籤：`json:"native_staging_present,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_staging_present: Option<NativeStagingPlacement>,
    /// NativeSource 保留無獨立結構可承載的原始寫入／對話呼叫點。
    /// Go 標籤：`json:"native_source,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "String::is_empty")]
    pub native_source: String,
    /// NativeTextIndex 是一次原版 0x15F84 呼叫的 FDTXT 索引；多句
    /// editable dialogue 可共用同一索引，但不能由文字內容反推。
    /// Go 標籤：`json:"native_text_index,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_text_index: Option<i64>,
    /// NativeDialogueRef 將戰鬥事件接到同一份故事翻譯與原始逐句版面。
    /// Go 標籤：`json:"native_dialogue_ref,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_dialogue_ref: Option<NativeEventDialogue>,
    /// EventStateIndex/Value 僅供已證實的 battle-local raw byte 寫入。
    /// Go 標籤：`json:"event_state_index,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub event_state_index: Option<i64>,
    /// Go 標籤：`json:"event_state_value,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub event_state_value: Option<i64>,
    /// NativeWhen 是原版處理器裡夾在動作之間的條件，執行到這個動作時才判斷。
    /// Go 標籤：`json:"native_when,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_when: Option<NativeActionWhen>,
    /// NativeDeathOp 承載死亡事件處理器裡沒有既有動作種類可表達的原語。
    /// Go 標籤：`json:"native_death_op,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_death_op: Option<NativeDeathOp>,
}

/// NativeStagingPlacement 保存33F78的slot/x/y，consumer仍走12CEA及22253。
///
/// Go：`event.go:254`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeStagingPlacement {
    /// Go 標籤：`json:"slot"`
    pub slot: i64,
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
}

/// NativeEventDialogue 僅描述資料來源，原生顯示與輸入由 GUI 事件擁有者執行。
///
/// Go：`event.go:261`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeEventDialogue {
    /// Go 標籤：`json:"script"`
    #[serde(deserialize_with = "null_as_default")]
    pub script: String,
    /// Go 標籤：`json:"scene_index"`
    pub scene_index: i64,
    /// Go 標籤：`json:"line"`
    pub line: i64,
    /// Go 標籤：`json:"source_dat"`
    #[serde(deserialize_with = "null_as_default")]
    pub source_dat: String,
    /// Go 標籤：`json:"string_index"`
    pub string_index: i64,
    /// Go 標籤：`json:"utterance"`
    pub utterance: i64,
    /// Go 標籤：`json:"control"`
    #[serde(deserialize_with = "null_as_default")]
    pub control: String,
    /// Go 標籤：`json:"operand"`
    pub operand: i64,
    /// Go 標籤：`json:"pages"`
    #[serde(deserialize_with = "null_as_default")]
    pub pages: Vec<Vec<String>>,
    /// Go 標籤：`json:"glyph_pages,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub glyph_pages: Vec<Vec<Vec<String>>>,
    /// Go 標籤：`json:"glyph_ids,omitempty"`
    #[serde(deserialize_with = "null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub glyph_ids: Vec<Vec<Vec<i64>>>,
}

/// NativeSpawnCall 保存全域事件處理器的一個確切呼叫點。Group 是該排程回合
/// 解析出的值；Source、Via 與 RawPlacementGate 直接來自 EXE，不從陣營或群組推測。
///
/// Go：`event.go:277`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeSpawnCall {
    /// Go 標籤：`json:"group"`
    pub group: i64,
    /// Go 標籤：`json:"via"`
    #[serde(deserialize_with = "null_as_default")]
    pub via: String,
    /// Go 標籤：`json:"source"`
    #[serde(deserialize_with = "null_as_default")]
    pub source: String,
    /// Go 標籤：`json:"raw_placement_gate"`
    pub raw_placement_gate: Option<i64>,
    /// Go 標籤：`json:"following_acting,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub following_acting: Option<NativeFollowingActing>,
    /// GroupFromState 非 nil 時 group 在執行當下讀戰場狀態表 [0x53AD5]+index（第九章事件 31
    /// `0x34B67 movzx eax, byte [eax+0x10]; push eax; call 0x10B4E`），Group 不使用。
    /// Go 標籤：`json:"group_from_state,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub group_from_state: Option<i64>,
}

/// NativeFollowingActing 保存 wrapper 返回後、由同一呼叫端明確執行的 ACTING。
/// 它不是由 group 推導的，也不是 0x32999 函式本體的一部分。
///
/// Go：`event.go:290`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeFollowingActing {
    /// Go 標籤：`json:"resource"`
    pub resource: i64,
    /// Go 標籤：`json:"source"`
    #[serde(deserialize_with = "null_as_default")]
    pub source: String,
}

/// DialogLine 一句對話(說話者肖像 + 文本),供 UI 畫頭像+嘴型+文字。
///
/// Go：`event.go:296`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct DialogLine {
    pub speaker: i64,
    pub text: String,
    /// 對話框上下位置覆蓋(nil=沿用預設「id>=32 走上框」規則;見 campaign.Beat.Upper)
    pub upper: Option<bool>,
    pub native_dialogue: Option<NativeDialogueLayout>,
}

/// NativeDialogueLayout 是 campaign 編輯資料中 FFFE／FFFD 頁面的執行期副本。
/// 型別留在 battle，避免泛用戰鬥套件反向依賴 campaign 編譯器。
///
/// Go：`event.go:305`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeDialogueLayout {
    pub source_dat: String,
    pub string_index: i64,
    pub utterance: i64,
    pub control: String,
    pub operand: i64,
    pub pages: Vec<Vec<String>>,
    pub glyph_pages: Vec<Vec<Vec<String>>>,
    pub glyph_ids: Vec<Vec<Vec<i64>>>,
    /// MotionTargetY 是 sub_15F84 var_20 的 caller-resolved runtime 值。
    /// 它不屬於可編輯 FDTXT bytes；HasMotionTarget=false 時原生收框必須失敗即關閉。
    pub motion_target_y: i64,
    pub has_motion_target_y: bool,
}

/// LoadScenario 讀 scenario JSON。
///
/// Go：`event.go:321`，`func LoadScenario(path string) (*Scenario, error)`
pub fn load_scenario(path: &str) -> Result<Scenario> {
    let raw = read_file(path)?;
    let sc: Scenario = decode_json(&raw)?;
    if let Some(code) = sc.native_result_initial_code
        && (code != 0
            || (sc.native_result_handler != "0x20b72" && sc.native_result_handler != "0x20bf5")
            || sc.native_result_rules.len() != 3)
    {
        return Err(battle_err!("原生結果初始值缺少已證實handler／三列規則"));
    }
    if sc.native_result_auto_advance && sc.native_result_initial_code.is_none() {
        return Err(battle_err!("原生自動戰後缺少已證實章結果來源"));
    }
    if !sc.native_result_rules.is_empty() {
        if sc.native_result_handler.is_empty() || !sc.native_result_code1_records.is_empty() {
            return Err(battle_err!("原生結果列缺少handler來源或混用舊記錄清單"));
        }
        let mut seen = BTreeSet::new();
        for (i, rule) in sc.native_result_rules.iter().enumerate() {
            if let Err(err) = rule.validate() {
                return Err(battle_err!("原生結果列{i}：{err}"));
            }
            if seen.contains(&rule.id) {
                return Err(battle_err!("原生結果列ID重複：{}", rule.id));
            }
            seen.insert(rule.id.clone());
        }
    }
    for (i, member) in sc.party.iter().enumerate() {
        // Validate at the editable boundary. PartyUnits has a legacy no-error
        // signature, so accepting malformed JSON there would otherwise silently
        // change the native command inventory to zero.
        if let Err(err) = Unit::default().set_initial_command_mask(&member.initial_command_mask) {
            return Err(battle_err!("scenario party member {i} ({}) initial_command_mask: {err}", member.name));
        }
        if let Some(identity) = member.native_identity
            && !(0..=0xff).contains(&identity)
        {
            return Err(battle_err!(
                "scenario party member {i} ({}) native_identity {identity} out of byte range",
                member.name
            ));
        }
    }
    for (event_index, event) in sc.events.iter().enumerate() {
        if let Some(when) = &event.when
            && (when.native_event_state_index.is_none() != when.native_event_state_value.is_none()
                || when.native_event_state_index.is_some_and(|index| {
                    let value = when.native_event_state_value.unwrap_or_default();
                    !(0..0x20).contains(&index) || !(0..=0xff).contains(&value)
                }))
        {
            return Err(battle_err!("scenario event {event_index} has invalid native event-state condition"));
        }
        for (action_index, action) in event.r#do.iter().enumerate() {
            if action.r#type == "native_acting"
                && (action
                    .native_acting
                    .as_ref()
                    .is_none_or(|acting| !(0..106).contains(&acting.resource) || acting.source.is_empty())
                    || action.native_event_id.is_none_or(|id| !(0..90).contains(&id))
                    || sc.native_acting_resources.is_empty())
            {
                return Err(battle_err!(
                    "scenario event {event_index} action {action_index} has invalid native acting provenance"
                ));
            }
            if action.r#type == "set_native_event_state"
                && (action.event_state_index.is_none_or(|index| !(0..0x20).contains(&index))
                    || action.event_state_value.is_none_or(|value| !(0..=0xff).contains(&value))
                    || action.native_source.is_empty()
                    || action.native_event_id.is_none_or(|id| !(0..90).contains(&id)))
            {
                return Err(battle_err!(
                    "scenario event {event_index} action {action_index} has invalid native event-state write"
                ));
            }
            if action.native_spawns.is_empty() {
                continue;
            }
            if action.r#type != "spawn_group" || action.native_spawns.len() != action.groups.len() {
                return Err(battle_err!(
                    "scenario event {event_index} action {action_index} has inconsistent native spawn calls"
                ));
            }
            if action.native_event_id.is_none_or(|id| !(0..90).contains(&id)) {
                return Err(battle_err!(
                    "scenario event {event_index} action {action_index} lacks native event provenance"
                ));
            }
            for (i, call) in action.native_spawns.iter().enumerate() {
                if call.group != action.groups[i]
                    || call.source.is_empty()
                    || (call.via != "spawn_group" && call.via != "spawn_group_with_intro")
                    || call.raw_placement_gate.is_none_or(|gate| !(0..=0xff).contains(&gate))
                {
                    return Err(battle_err!(
                        "scenario event {event_index} action {action_index} native spawn {i} is invalid"
                    ));
                }
                if call.via == "spawn_group_with_intro" {
                    if call
                        .following_acting
                        .as_ref()
                        .is_none_or(|acting| !(0..106).contains(&acting.resource) || acting.source.is_empty())
                        || sc.native_acting_resources.is_empty()
                    {
                        return Err(battle_err!(
                            "scenario event {event_index} action {action_index} native intro spawn {i} lacks following acting provenance/resources"
                        ));
                    }
                } else if call.following_acting.is_some() {
                    return Err(battle_err!(
                        "scenario event {event_index} action {action_index} ordinary native spawn {i} carries intro acting"
                    ));
                }
            }
        }
    }
    let mut seen_native_field = BTreeSet::new();
    for (rule_index, rule) in sc.native_field_event_rules.iter().enumerate() {
        let key = (rule.event_id, rule.selector as i64);
        let chain_invalid = rule.turn_chain.as_ref().is_none_or(|chain| {
            chain.handler.is_empty() || chain.state_writes.is_empty() || chain.turn_activations.is_empty()
        });
        if !(0..90).contains(&rule.event_id)
            || rule.selector > 2
            || seen_native_field.contains(&key)
            || rule.trigger_gate.is_empty()
            || chain_invalid
        {
            return Err(battle_err!("scenario native field rule {rule_index} is invalid"));
        }
        seen_native_field.insert(key);
        let chain = rule.turn_chain.as_ref().expect("上方已檢查 turn_chain");
        for write in &chain.state_writes {
            if !(0..0x20).contains(&write.index) {
                return Err(battle_err!("scenario native field rule {rule_index} state write is invalid"));
            }
        }
        for activation in &chain.turn_activations {
            // Go 的 RawCamp 是 byte，`< 0 || > 0xff` 恆為假；Rust 的 u8 同樣不需檢查。
            if !(0..16).contains(&activation.slot) || !(0..90).contains(&activation.event_id) {
                return Err(battle_err!("scenario native field rule {rule_index} turn activation is invalid"));
            }
        }
    }
    let mut seen_native_turn = BTreeSet::new();
    let initial_groups: BTreeSet<i64> = sc.initial_groups.iter().copied().collect();
    for (event_index, event) in sc.native_turn_events.iter().enumerate() {
        let key = (event.event_id, event.raw_camp);
        let staging = &event.staging;
        if !sc.runtime_append_groups
            || !(0..90).contains(&event.event_id)
            || !(0..=0xff).contains(&event.raw_camp)
            || event.handler.is_empty()
            || seen_native_turn.contains(&key)
        {
            return Err(battle_err!("scenario native turn event {event_index} is invalid"));
        }
        if !event.actions.is_empty() {
            if event.dynamic_group.is_some()
                || event.progression.is_some()
                || event.pair_mutation.is_some()
                || !staging.calls.is_empty()
            {
                return Err(battle_err!("scenario native turn event {event_index} mixes actions with staging"));
            }
            seen_native_turn.insert(key);
            if let Some(r) = event.state_range
                && (r[0] < 0 || r[0] >= 0x20 || r[1] < 0 || r[2] < r[1] || r[2] > 0xff)
            {
                return Err(battle_err!("scenario native turn event {event_index} state range is invalid"));
            }
            continue;
        }
        if event.progression.is_none()
            && event.pair_mutation.is_none()
            && (staging.helper.is_empty()
                || staging.pan_helper.is_empty()
                || staging.spawn_helper.is_empty()
                || staging.palette_helper.is_empty()
                || staging.redraw_helper.is_empty()
                || staging.delay_before_flash_ms < 0
                || staging.flash_hold_ms < 0
                || staging.palette_start < 0
                || staging.palette_end < staging.palette_start
                || staging.palette_end > 255
                || !(0..=255).contains(&staging.flash_delta)
                || !(0..=255).contains(&staging.restore_delta)
                || !(0..=0xff).contains(&staging.raw_placement_gate)
                || staging.calls.is_empty())
        {
            return Err(battle_err!("scenario native turn event {event_index} is invalid"));
        }
        seen_native_turn.insert(key);
        if let Some(dynamic) = &event.dynamic_group
            && (!(0..0x20).contains(&dynamic.state_index)
                || dynamic.minimum < 0
                || dynamic.maximum < dynamic.minimum
                || dynamic.maximum > 0xfe
                || !(0..16).contains(&dynamic.control_slot)
                || dynamic.reschedule_delta <= 0
                || dynamic.stop_value < dynamic.minimum
                || dynamic.stop_value > dynamic.maximum
                || dynamic.increment <= 0
                || staging.calls.len() != 1)
        {
            return Err(battle_err!("scenario native turn event {event_index} dynamic group is invalid"));
        }
        if let Some(progression) = &event.progression {
            let fa = &progression.final_activation;
            if event.dynamic_group.is_some()
                || !staging.calls.is_empty()
                || !(0..0x20).contains(&progression.state_index)
                || progression.repeat_until <= 0
                || progression.repeat_until > 0xff
                || progression.mark_unit_index < 0
                || progression.mark_helper.is_empty()
                || !(0..16).contains(&progression.control_slot)
                || progression.reschedule_delta <= 0
                || progression.final_text_index < 0
                || !(0..=0xff).contains(&progression.spawn_group)
                || progression.spawn_helper.is_empty()
                || !(0..=0xff).contains(&progression.raw_placement_gate)
                || progression.spawn_count <= 0
                || !(0..0x20).contains(&progression.base_state_index)
                || !(0..16).contains(&fa.slot)
                || !(0..90).contains(&fa.event_id)
                || progression.pulse_handler.is_empty()
                || progression.pulse_count <= 0
                || progression.extra_delay_ms < 0
                || progression.tail_text_indices.is_empty()
            {
                return Err(battle_err!("scenario native turn event {event_index} progression is invalid"));
            }
        }
        if let Some(pair) = &event.pair_mutation
            && (event.dynamic_group.is_some()
                || event.progression.is_some()
                || !staging.calls.is_empty()
                || !(0..0x20).contains(&pair.base_state_index)
                || !(0..=0xff).contains(&pair.group)
                || pair.count <= 1
                || pair.modulo <= 1
                || pair.second_offset <= 0
                || pair.mark_helper.is_empty()
                || !(0..16).contains(&pair.control_slot)
                || pair.reschedule_delta <= 0)
        {
            return Err(battle_err!("scenario native turn event {event_index} pair mutation is invalid"));
        }
        let mut seen_groups = BTreeSet::new();
        for (call_index, call) in staging.calls.iter().enumerate() {
            if (event.dynamic_group.is_none() && !(0..=0xff).contains(&call.group))
                || (event.dynamic_group.is_some() && call.group != -1)
                || call.x < 0
                || call.y < 0
                || call.source.is_empty()
                || seen_groups.contains(&call.group)
                || initial_groups.contains(&call.group)
            {
                return Err(battle_err!(
                    "scenario native turn event {event_index} staging call {call_index} is invalid"
                ));
            }
            seen_groups.insert(call.group);
        }
    }
    Ok(sc)
}

impl Scenario {
    /// Setup 是既有測試與無原生欄位規則劇本的相容入口。正式執行路徑必須使用
    /// SetupChecked，避免把資料綁定錯誤誤當成沒有開場對話。
    ///
    /// Go：`event.go:541`，`func (sc *Scenario) Setup(st *State) []DialogLine`
    pub fn setup(&mut self, st: Option<&mut State>) -> Vec<DialogLine> {
        self.setup_checked(st).unwrap_or_default()
    }

    /// SetupChecked 套用劇本初始狀態並回報原生規則綁定錯誤；正式執行期以此
    /// 入口維持失敗即關閉。
    ///
    /// Go：`event.go:548`，`func (sc *Scenario) SetupChecked(st *State) ([]DialogLine, error)`
    pub fn setup_checked(&mut self, st: Option<&mut State>) -> Result<Vec<DialogLine>> {
        let mut st = st;
        self.bind_native_field_event_rules_inner(st.as_deref_mut())?;
        let st = st.expect("bindNativeFieldEventRules 已拒絕 nil State");
        if self.runtime_append_groups {
            // Keep FDFIELD records as immutable-order constructor inputs. The opening
            // event materializes the player party first; initial FDFIELD groups follow,
            // exactly matching the runtime slots addressed by handler bytecode.
            st.roster = std::mem::take(&mut st.units);
            self.materialize_pending_groups(st);
            let dialogues = self.fire(Some(&mut *st), "on_battle_start", "");
            for &group in &self.initial_groups {
                st.append_group(group);
            }
            return Ok(dialogues);
        }
        if !self.initial_groups.is_empty() {
            let mut init: BTreeSet<i64> = self.initial_groups.iter().copied().collect();
            let present: BTreeSet<i64> = self.party.iter().map(|member| member.fig).collect();
            for conditional in &self.initial_groups_absent {
                if !present.contains(&conditional.char_id) {
                    init.insert(conditional.group);
                }
            }
            for u in &mut st.units {
                if !init.contains(&u.group) {
                    u.on_field = false; // 待命,等事件放出
                }
            }
        }
        Ok(self.fire(Some(st), "on_battle_start", ""))
    }

    /// Go：`event.go:588`，`func (sc *Scenario) bindNativeFieldEventRules(st *State) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn bind_native_field_event_rules_inner(&self, st: Option<&mut State>) -> Result<()> {
        let Some(st) = st else {
            return Err(battle_err!("scenario field rules: missing scenario or state"));
        };
        let mut seen: BTreeSet<(i64, u8)> =
            st.native_field_event_rules.iter().map(|rule| (rule.event_id, rule.selector)).collect();
        for rule in &self.native_field_event_rules {
            let key = (rule.event_id, rule.selector);
            if !(0..90).contains(&rule.event_id) || seen.contains(&key) {
                return Err(battle_err!(
                    "scenario field rule event{} selector{} is invalid or duplicated",
                    rule.event_id,
                    rule.selector
                ));
            }
            seen.insert(key);
            st.native_field_event_rules.push(rule.clone());
        }
        Ok(())
    }

    /// BindNativeFieldEventRules 只把劇本的可編輯格子規則綁到既有戰場狀態。
    /// 原版 CONTINUE 已保存 live runtime，不能呼叫 SetupChecked 重播開場或重建單位；
    /// 這個窄入口讓續戰交易仍使用和正常新戰鬥相同的驗證與重複鍵拒絕規則。
    ///
    /// Go：`event.go:610`，`func (sc *Scenario) BindNativeFieldEventRules(st *State) error`
    pub fn bind_native_field_event_rules(&self, st: Option<&mut State>) -> Result<()> {
        self.bind_native_field_event_rules_inner(st)
    }

    /// Go：`event.go:614`，`func (sc *Scenario) materializePendingGroups(st *State)`
    pub(crate) fn materialize_pending_groups(&self, st: &mut State) {
        st.pending_groups = BTreeMap::new();
        for event in &self.events {
            for action in &event.r#do {
                if action.r#type == "spawn_group" {
                    for &group in &action.groups {
                        st.pending_groups.insert(group, true);
                    }
                }
            }
        }
        for event in &self.native_turn_events {
            if let Some(dynamic) = &event.dynamic_group {
                for group in dynamic.minimum..=dynamic.maximum {
                    st.pending_groups.insert(group, true);
                }
                continue;
            }
            if let Some(progression) = &event.progression {
                st.pending_groups.insert(progression.spawn_group, true);
                continue;
            }
            for call in &event.staging.calls {
                st.pending_groups.insert(call.group, true);
            }
        }
    }

    /// NativeTurnEventsAt returns the editable handlers selected by sub_1A813's
    /// exact live-row predicate, preserving the sixteen-row order. A live row with
    /// no unique editable consumer is an error rather than a skipped event.
    /// NativeTurnActionEventsAt 回傳這一階段活的控制列裡、帶轉寫動作的事件，依 16 列順序。
    /// 沒有動作版本的列（由靜態 turn_events 降成的劇本事件，或舊的 staging 事件）不在這裡處理。
    ///
    /// Go：`event.go:647`，`func (sc *Scenario) NativeTurnActionEventsAt(st *State, rawCamp byte) []NativeTurnEvent`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_turn_action_events_at(&self, st: Option<&State>, raw_camp: u8) -> Vec<NativeTurnEvent> {
        let Some(st) = st.filter(|st| st.has_native_turn_event_control_state && st.native_round_counter > 0) else {
            return Vec::new();
        };
        let mut out = Vec::new();
        for row in &st.native_turn_event_controls {
            if row.turn as i64 != st.native_round_counter || row.raw_camp != raw_camp {
                continue;
            }
            for event in &self.native_turn_events {
                if !event.actions.is_empty()
                    && event.event_id == row.event_id as i64
                    && event.raw_camp == raw_camp as i64
                {
                    out.push(event.clone());
                }
            }
        }
        out
    }

    /// Go：`event.go:665`，`func (sc *Scenario) NativeTurnEventsAt(st *State, rawCamp byte) ([]NativeTurnEvent, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_turn_events_at(&self, st: Option<&State>, raw_camp: u8) -> Result<Vec<NativeTurnEvent>> {
        let Some(st) = st.filter(|st| {
            st.has_native_turn_event_control_state && st.native_round_counter > 0 && st.native_round_counter <= 0xfe
        }) else {
            return Err(battle_err!("native turn event: live control state unavailable"));
        };
        let mut out = Vec::new();
        for (slot, row) in st.native_turn_event_controls.iter().enumerate() {
            if row.turn as i64 != st.native_round_counter || row.raw_camp != raw_camp {
                continue;
            }
            let mut matches = 0;
            let mut matched = None;
            for event in &self.native_turn_events {
                if event.event_id == row.event_id as i64 && event.raw_camp == raw_camp as i64 {
                    matched = Some(event);
                    matches += 1;
                }
            }
            if matches != 1 {
                return Err(battle_err!(
                    "native turn event: slot {slot} event {} raw camp {raw_camp} has {matches} editable consumers",
                    row.event_id
                ));
            }
            let matched = matched.expect("matches == 1");
            if let Some(r) = matched.state_range
                && (r[0] < 0
                    || r[0] >= st.native_event_state.len() as i64
                    || (st.native_event_state[r[0] as usize] as i64) < r[1]
                    || st.native_event_state[r[0] as usize] as i64 > r[2])
            {
                return Err(battle_err!("native turn event: state outside recovered range"));
            }
            out.push(matched.clone());
        }
        Ok(out)
    }

    /// AdoptHandlerBattleState attaches turn/death events to a runtime roster that
    /// the immediately preceding native pre-handler already constructed and moved.
    /// It must not replay on_battle_start: LOADCH/SPAWN/ACT effects are already
    /// present in that roster. Callers are responsible for proving that handler and
    /// battle use the same roster/scenario sources.
    ///
    /// Go：`event.go:702`，`func (sc *Scenario) AdoptHandlerBattleState(st *State) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn adopt_handler_battle_state(&mut self, st: Option<&mut State>) -> Result<()> {
        let Some(st) = st.filter(|st| self.runtime_append_groups && !st.units.is_empty()) else {
            return Err(battle_err!("battle: handler state cannot satisfy runtime-append scenario"));
        };
        self.bind_native_field_event_rules_inner(Some(&mut *st))?;
        self.materialize_pending_groups(st);
        // LOADCH always constructs party first and may already have appended one or
        // more initial FDFIELD groups. The caller removes those source rows from
        // Roster before this boundary; append every declared initial group now so a
        // no-op preserves an already materialized group while remaining groups keep
        // the original selector-cache order.
        for &group in &self.initial_groups {
            st.append_group(group);
            if let Some(err) = &st.native_map_selector_error {
                return Err(battle_err!("battle: adopted initial group {group}: {err}"));
            }
        }
        for event in &mut self.events {
            if event.trigger == "on_battle_start" {
                if !event.once {
                    return Err(battle_err!("battle: repeatable on_battle_start cannot be adopted"));
                }
                event.fired = true;
            }
        }
        Ok(())
    }

    /// Fire 對某 trigger 評估所有事件,執行符合者的動作。回傳要播的對話(含說話者)。
    /// ctxUnit:on_unit_death 時傳陣亡者名。
    ///
    /// Go：`event.go:735`，`func (sc *Scenario) Fire(st *State, trigger, ctxUnit string) []DialogLine`
    pub fn fire(&mut self, st: Option<&mut State>, trigger: &str, ctx_unit: &str) -> Vec<DialogLine> {
        let mut st = st;
        let actions = self.trigger_actions(st.as_deref(), trigger, ctx_unit);
        let mut dialogues = Vec::new();
        for action in &actions {
            if let (dl, true) = self.execute_action(st.as_deref_mut(), action) {
                dialogues.push(dl);
            }
        }
        dialogues
    }

    /// TriggerActions evaluates one trigger and returns its ordered editable actions,
    /// marking matching once-events as fired without executing them. The UI runtime
    /// uses this to preserve blocking PAN/delay/dialogue order; Fire remains the
    /// synchronous compatibility path for setup, tests and triggers without staging.
    ///
    /// Go：`event.go:750`，`func (sc *Scenario) TriggerActions(st *State, trigger, ctxUnit string) []Action`
    pub fn trigger_actions(&mut self, st: Option<&State>, trigger: &str, ctx_unit: &str) -> Vec<Action> {
        self.trigger_actions_where(st, trigger, ctx_unit, None)
    }

    /// TriggerActionsWhere 與 TriggerActions 相同，但只收 accept 回 true 的事件；
    /// 沒被收的事件不標 fired，留給下一個觸發點。
    ///
    /// Go：`event.go:756`，`func (sc *Scenario) TriggerActionsWhere(st *State, trigger, ctxUnit string, accept func(*Event) bool) []Action`
    pub fn trigger_actions_where(
        &mut self,
        st: Option<&State>,
        trigger: &str,
        ctx_unit: &str,
        accept: Option<&mut dyn FnMut(&Event) -> bool>,
    ) -> Vec<Action> {
        let mut accept = accept;
        let mut actions = Vec::new();
        for e in &mut self.events {
            if e.trigger != trigger || (e.once && e.fired) {
                continue;
            }
            // Go 的 `(*When)(nil).match` 恆真。
            let matched = e.when.as_ref().is_none_or(|w| w.r#match(st, ctx_unit));
            if !matched || accept.as_mut().is_some_and(|accept| !accept(e)) {
                continue;
            }
            e.fired = true;
            actions.extend(e.r#do.iter().cloned());
        }
        actions
    }
}
impl Event {
    /// NativeTurnPhaseSelector 回報由 FDFIELD turn_events 轉出的回合事件對應的 0x1A813
    /// phase selector（該列第三個 byte，gen_campaign 以 camp 字串保存）：1 在 END 之後、
    /// 友軍 AI 之前；0 在敵方 AI 之前；2 在下一回合的玩家輸入之前（那時 [0x53BEF]
    /// 已經加一）。手寫、沒有 native_event_id 的事件回 -1，維持舊的回合末時機。
    ///
    /// Go：`event.go:776`，`func (e *Event) NativeTurnPhaseSelector() int`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_turn_phase_selector(&self) -> i64 {
        for a in &self.r#do {
            if a.native_event_id.is_none() {
                continue;
            }
            match a.camp.as_str() {
                "ally" => return 1,
                "enemy" => return 0,
                "special" => return 2,
                _ => {}
            }
        }
        -1
    }
}
impl When {
    /// match 條件判斷(可擴充)。nil = 無條件,恆真。
    ///
    /// Go：`event.go:797`，`func (w *When) match(st *State, ctxUnit string) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn r#match(&self, st: Option<&State>, ctx_unit: &str) -> bool {
        // Go 在 Turn、UnitSlotActive 兩個條件直接解參照 st（nil 會 panic）；Rust 照樣在這兩處要求 State。
        if self.turn != 0 && st.expect("When.match: Go 對 nil State 解參照").turn != self.turn {
            return false;
        }
        if !self.unit_dead.is_empty() && ctx_unit != self.unit_dead {
            return false;
        }
        if let Some(slot) = self.unit_slot_active {
            let st = st.expect("When.match: Go 對 nil State 解參照");
            if slot < 0
                || slot >= st.units.len() as i64
                || !st.units[slot as usize].on_field
                || !st.units[slot as usize].alive()
            {
                return false;
            }
        }
        if self.native_event_state_index.is_some() || self.native_event_state_value.is_some() {
            let (Some(index), Some(value)) = (self.native_event_state_index, self.native_event_state_value) else {
                return false;
            };
            let Some(st) = st else {
                return false;
            };
            if index < 0
                || index >= st.native_event_state.len() as i64
                || st.native_event_state[index as usize] as i64 != value
            {
                return false;
            }
        }
        true
    }
}
impl Scenario {
    /// ExecuteAction 執行單一狀態動作。pan/delay 由 UI runner 阻塞處理；
    /// 回傳 (對話, true) 表示 runner 應停下並播放這句。
    ///
    /// Go：`event.go:829`，`func (sc *Scenario) ExecuteAction(st *State, a Action) (DialogLine, bool)`
    pub fn execute_action(&mut self, st: Option<&mut State>, a: &Action) -> (DialogLine, bool) {
        let (dialogue, is_dialogue) = self.execute_action_checked(st, a).unwrap_or_default();
        (dialogue, is_dialogue)
    }

    /// ExecuteActionChecked 是正式執行路徑的錯誤回報邊界。上方相容包裝保留給舊的
    /// 同步呼叫者；介面執行器使用本函式，避免原版增援資料錯誤被靜默忽略。
    ///
    /// Go：`event.go:836`，`func (sc *Scenario) ExecuteActionChecked(st *State, a Action) (DialogLine, bool, error)`
    pub fn execute_action_checked(&mut self, st: Option<&mut State>, a: &Action) -> Result<(DialogLine, bool)> {
        // Go 在 spawn_party、一般化 spawn_group、set_flag、set_ai 直接解參照 st（nil 會 panic）。
        const NIL_STATE: &str = "ExecuteActionChecked: Go 對 nil State 解參照";
        let mut st = st;
        match a.r#type.as_str() {
            "spawn_party" => {
                // 主角隊從隊伍名冊進場到部署格(doc 25 雙來源)
                // The party constructor itself places members directly on the deployment
                // cells. Chapter 0 then immediately plays decoded ACT(0), which moves all
                // four runtime slots six cells upward; the old "no movement animation"
                // conclusion confused construction with the following handler operation.
                let st = st.expect(NIL_STATE);
                let units = self.party_units(&st.own_deploy);
                st.append_native_map_selector_batch_or_legacy(units);
            }
            "spawn_group" => {
                // 增援登場(原版 turn_events;doc 25)
                let camp = camp_from(&a.camp);
                if !a.native_spawns.is_empty() && self.runtime_append_groups {
                    let Some(st) = st.filter(|st| !st.roster.is_empty()) else {
                        return Err(battle_err!("native spawn requires a runtime roster"));
                    };
                    // 0x32999 不只是建構單位：它含 FDOTHER #9 的 12 次索引呈現，
                    // wrapper 返回後呼叫端還會執行另一個 ACTING。兩段尚未由正式介面
                    // 執行器承接前，必須在任何 roster 變更之前停止。
                    for call in &a.native_spawns {
                        if call.via == "spawn_group_with_intro" {
                            return Err(battle_err!(
                                "native intro spawn {} requires the 0x32999 transition and following acting adapter",
                                call.source
                            ));
                        }
                    }
                    for call in &a.native_spawns {
                        let Some(gate) = call.raw_placement_gate else {
                            return Err(battle_err!("native spawn {} lacks raw placement gate", call.source));
                        };
                        let before = st.units.len();
                        let mut group = call.group;
                        if let Some(index) = call.group_from_state {
                            if index < 0 || index >= st.native_event_state.len() as i64 {
                                return Err(battle_err!(
                                    "native spawn {} state index {index} outside table",
                                    call.source
                                ));
                            }
                            group = st.native_event_state[index as usize] as i64;
                        }
                        if let Err(err) = st.append_group_with_native_placement(group, gate as u8) {
                            return Err(battle_err!("native spawn {}: {err}", call.source));
                        }
                        // FDFIELD 回合事件列的第三個 byte 是 0x1A813 的 phase selector
                        // （1＝END 之後友軍 AI 前、0＝敵方 AI 前），不是登場單位的陣營；
                        // 陣營由該 group 的 FDFIELD record +6 決定。第四章 event 11
                        // （turn_events 標 ally）原版登場的是 camp 0 敵軍，收據見
                        // docs/data/ui-traces/parity-ch04.json。原生路徑因此不覆寫 camp。
                        // 0x32999 建構的 record +5 是 0：登場當下就能行動（第四章 event 11
                        // 的四名敵軍在同一個敵方回合就走了）。act_immediately 只屬正規化路徑。
                        for unit in &mut st.units[before..] {
                            unit.acted = false;
                        }
                    }
                } else {
                    // 尚未遷移到 runtime_append_groups 的情境仍走明確標示的正規化
                    // 相容路徑。資料即使已帶原版欄位，本分支也不能作為忠實度證據。
                    for &group in &a.groups {
                        st.as_deref_mut().expect(NIL_STATE).spawn_group(
                            group,
                            camp,
                            !a.camp.is_empty(),
                            a.act_immediately,
                        );
                    }
                }
            }
            "join_party" => self.pending_joins.push(a.char_id),
            "dialogue" => {
                if a.native_dialogue_ref.is_some() {
                    return Err(battle_err!("原生事件對話必須由 GUI 生命週期執行"));
                }
                return Ok((DialogLine { speaker: a.speaker, text: a.text.clone(), ..Default::default() }, true));
            }
            "set_native_event_state" => {
                let Some(st) = st.filter(|st| {
                    a.event_state_index.is_some_and(|index| index >= 0 && index < st.native_event_state.len() as i64)
                        && a.event_state_value.is_some_and(|value| (0..=0xff).contains(&value))
                        && !a.native_source.is_empty()
                        && a.native_event_id.is_some_and(|id| (0..90).contains(&id))
                }) else {
                    return Err(battle_err!("native event-state write lacks proven raw provenance"));
                };
                st.native_event_state[a.event_state_index.unwrap_or_default() as usize] =
                    a.event_state_value.unwrap_or_default() as u8;
            }
            "set_flag" => {
                st.expect(NIL_STATE).flags.insert(a.flag.clone(), true);
            }
            "set_ai" => {
                // Editable scenario marker only. No planner currently consumes this
                // string, and "berserk" has not been mapped to native record +0x34.
                // Keep the inert marker visible instead of claiming an AI transition.
                st.expect(NIL_STATE).flags.insert(format!("ai_{}_{}", a.unit, a.mode), true);
            }
            _ => {}
        }
        Ok((DialogLine::default(), false))
    }

    /// TakePartyJoins transfers JOIN effects from battle-script execution to the
    /// campaign-owned persistent roster. Scenario stays independent from the UI /
    /// save layer while preserving the original ordering (JOIN before SPAWN).
    ///
    /// Go：`event.go:925`，`func (sc *Scenario) TakePartyJoins() []int`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn take_party_joins(&mut self) -> Vec<i64> {
        std::mem::take(&mut self.pending_joins)
    }

    /// PartyUnits materializes the persistent player roster in scenario order.
    /// Original chapter 0 constructs these units before any FDFIELD spawn, so this
    /// order is also their authoritative acting-slot order (slots 0..3).  The same
    /// constructor is shared by battle setup and handler cutscenes to prevent the
    /// two paths from drifting in stats, deployment cells, or identity.
    ///
    /// Go：`event.go:939`，`func (sc *Scenario) PartyUnits(fallback []Cell) []*Unit`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    /// 所有權：回傳新建的單位本體（Go 每次 `&Unit{...}`），由呼叫端放進 `State::units`。
    pub fn party_units(&self, fallback: &[Cell]) -> Vec<Unit> {
        let mut units = Vec::with_capacity(self.party.len());
        for (i, pm) in self.party.iter().enumerate() {
            let (mut x, mut y) = (0, 0);
            if i < self.deploy_cells.len() {
                (x, y) = (self.deploy_cells[i][0], self.deploy_cells[i][1]);
            } else if i < fallback.len() {
                (x, y) = (fallback[i].x, fallback[i].y);
            }
            let (inventory, equipped, runtime_slots) = materialize_inventory(&pm.inventory_slots, &pm.inventory);
            let mut u = Unit {
                camp: OWN,
                name: pm.name.clone(),
                cls_name: pm.cls.clone(),
                lv: pm.lv,
                hp: pm.hp,
                max_hp: pm.hp,
                mp: pm.mp,
                max_mp: pm.mp,
                ap: pm.ap,
                dp: pm.dp,
                dx: pm.dx,
                mv: pm.mv,
                hit: pm.hit,
                ev: pm.ev,
                crit_pct: pm.crit_pct,
                atk_min: pm.atk_min,
                atk_max: pm.atk_max,
                portrait: pm.portrait,
                fig: pm.fig,
                // Fresh native JOIN writes join_id to persistent +7/+8. Fig is the
                // authored JOIN identity here; class change later writes raw +7
                // explicitly and must not alter Fig/+8 identity.
                battle_fig: pm.fig,
                has_battle_fig: true,
                map_selector_key: pm.fig,
                has_map_selector_key: true,
                x,
                y,
                on_field: true,
                spells: pm.spells.clone(),
                inventory,
                equipped,
                inventory_slots: runtime_slots,
                dir: 0,
                ..Default::default()
            };
            // The native constructor writes record byte +5 as zero for a newly
            // materialized unit. Keep that provenance explicit so handler predicates
            // can prefer the raw byte instead of inferring it from HP/OnField.
            if u.hp > 0 {
                u.native_record_byte5 = 0;
                u.has_native_record_byte5 = true;
            }
            // Party constructors use FDFIELD camp code 2 for native record +6.
            u.native_record_byte6 = 2;
            u.has_native_record_byte6 = true;
            // 原版單位建構子把最大 HP／MP 寫進記錄 +0x42／+0x46；持久名冊
            // (native_persistent_party)、JOIN(native_join_constructor) 與待登場
            // (native_future_constructor) 三個建構子都是同一條關係。主角隊由可編輯
            // 腳本物化時沿用同一份已授權的最大值，不另行猜測；超出 16 位元的資料
            // 不標記出處，讓 HUD 依失敗即關閉拒絕。
            if (0..=0xffff).contains(&u.max_hp) {
                (u.native_record_word42, u.has_native_record_word42) = (u.max_hp as u16, true);
            }
            if (0..=0xffff).contains(&u.max_mp) {
                (u.native_record_word46, u.has_native_record_word46) = (u.max_mp as u16, true);
            }
            if let Ok(flags) = native_inventory_flags_from_source(&pm.inventory_slots) {
                u.native_inventory_flags = flags;
            }
            if let Some(identity) = pm.native_identity
                && (0..=0xff).contains(&identity)
            {
                u.native_identity = identity;
                u.has_native_identity = true;
            }
            if let Some(race) = pm.native_record_race {
                (u.native_record_race, u.has_native_record_race) = (race, true);
            }
            if let Some(class) = pm.native_record_class {
                (u.native_record_class, u.has_native_record_class) = (class, true);
            }
            // Editable scenario AP/DP/HIT/EV are already effective values (doc32),
            // so preserve them as the base for later shop purchases.
            (u.base_ap, u.base_dp, u.base_hit, u.base_ev, u.base_mv) = (u.ap, u.dp, u.hit, u.ev, u.mv);
            (u.base_atk_min, u.base_atk_max, u.equipment_base_set) = (u.atk_min, u.atk_max, false);
            // LoadScenario validates authored masks before this materialization. Keep
            // direct in-memory Scenario construction backward compatible: an invalid
            // field is not partially copied into a different command inventory.
            let _ = u.set_initial_command_mask(&pm.initial_command_mask);
            units.push(u);
        }
        units
    }
}

/// Go 讀 JSON 陣列到 `[N]int`：較短時其餘元素是零值、較長時多餘元素丟棄、`null` 元素保留零值；
/// serde 的固定長度陣列要求長度完全相同，這裡照 Go 的寬鬆規則。
fn go_fixed_ints<const N: usize>(raw: Option<Vec<Option<i64>>>) -> [i64; N] {
    let mut out = [0i64; N];
    for (slot, v) in out.iter_mut().zip(raw.unwrap_or_default()) {
        *slot = v.unwrap_or_default();
    }
    out
}

/// serde 輔助：Go `*[N]int`（`null` 讀成 `None`，陣列長度依 Go 規則）。
fn go_int_array_opt<'de, const N: usize, D: serde::Deserializer<'de>>(
    d: D,
) -> std::result::Result<Option<[i64; N]>, D::Error> {
    use serde::Deserialize as _;
    Ok(Option::<Vec<Option<i64>>>::deserialize(d)?.map(|v| go_fixed_ints(Some(v))))
}

/// serde 輔助：Go `[][N]int`（`null` 讀成空切片，每個元素依 Go 規則）。
fn go_int_arrays<'de, const N: usize, D: serde::Deserializer<'de>>(
    d: D,
) -> std::result::Result<Vec<[i64; N]>, D::Error> {
    use serde::Deserialize as _;
    Ok(Option::<Vec<Option<Vec<Option<i64>>>>>::deserialize(d)?
        .unwrap_or_default()
        .into_iter()
        .map(go_fixed_ints)
        .collect())
}

#[cfg(test)]
mod event_test;
#[cfg(test)]
mod native_ch30_chain_test;
#[cfg(test)]
mod native_turn_event_test;
