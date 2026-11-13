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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Scenario {
    /// Go 標籤：`json:"chapter"`
    pub chapter: i64,
    /// Go 標籤：`json:"name"`
    pub name: String,
    /// Go 標籤：`json:"map"`
    pub map: i64,
    /// party first; FDFIELD groups append only when constructed
    /// Go 標籤：`json:"runtime_append_groups,omitempty"`
    pub runtime_append_groups: bool,
    /// asset-root-relative decoded 0x1366A resource set for native battle events
    /// Go 標籤：`json:"native_acting_resources,omitempty"`
    pub native_acting_resources: String,
    /// 開局即在場的 unit group;其餘待命
    /// Go 標籤：`json:"initial_groups"`
    pub initial_groups: Vec<i64>,
    /// native pre-handler conditional FDFIELD group
    /// Go 標籤：`json:"initial_groups_if_party_absent,omitempty"`
    pub initial_groups_absent: Vec<InitialGroupAbsent>,
    /// 主角隊(不在 FDFIELD roster,on_battle_start 進場)
    /// Go 標籤：`json:"party"`
    pub party: Vec<PartyMember>,
    /// 主角隊進場目標格
    /// Go 標籤：`json:"deploy_cells"`
    pub deploy_cells: Vec<[i64; 2]>,
    /// Go 標籤：`json:"events"`
    pub events: Vec<Event>,
    /// Go 標籤：`json:"native_field_event_rules,omitempty"`
    pub native_field_event_rules: Vec<NativeFieldEventRule>,
    /// Go 標籤：`json:"native_turn_events,omitempty"`
    pub native_turn_events: Vec<NativeTurnEvent>,
    /// NativeDeathPrograms 是死亡效果型態 2／3 降成的動作清單，鍵是「型態:值」。
    /// 型態2亦是全域0x51B91事件程式；已審查的selector0行動收尾共用它，不加死亡delay。
    /// 來源與逐動作的原始位址見 native_death_program.go。
    /// Go 標籤：`json:"native_death_programs,omitempty"`
    pub native_death_programs: BTreeMap<String, Vec<Action>>,
    /// NativeResultHandler／NativeResultCode1Records 是非 default 的章節勝負 handler：先跑
    /// 0x205BE 的三值規則，再對列出的記錄查 raw +5 bit0（0x3453E），任一筆成立就寫
    /// [0x53ECC]=1——與 default「記錄 0 倒下」同一個碼。第十章 0x20707 查記錄 50、51
    /// （docs/knowledge-base/26-per-chapter-event-handlers.md）。
    /// Go 標籤：`json:"native_result_handler,omitempty"`
    pub native_result_handler: String,
    /// Go 標籤：`json:"native_result_initial_code,omitempty"`
    pub native_result_initial_code: Option<i64>,
    /// Go 標籤：`json:"native_result_auto_advance,omitempty"`
    pub native_result_auto_advance: bool,
    /// Go 標籤：`json:"native_result_code1_records,omitempty"`
    pub native_result_code1_records: Vec<i64>,
    /// Go 標籤：`json:"native_result_rules,omitempty"`
    pub native_result_rules: Vec<NativeResultRule>,
    pub(crate) pending_joins: Vec<i64>,
}

impl Scenario {
    /// NativeResultCode1 回報章節 handler 列出的記錄是否有任一筆 raw +5 bit0 成立。還沒建立的
    /// 記錄（索引超出記錄表）原版讀到的是未使用槽，不算。
    ///
    /// Go：`event.go:46`，`func (sc *Scenario) NativeResultCode1(st *State) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_result_code1(&self, st: Option<&State>) -> bool {
        todo!("battle/event.go:Scenario.NativeResultCode1")
    }
}

/// NativeTurnEvent preserves one live FDFIELD three-byte row consumer without
/// forcing it into the normalized on_turn_end trigger. RawCamp is deliberately
/// not renamed to a faction: sub_1A813 compares the byte at row+5, and each
/// caller owns a different phase boundary.
///
/// Go：`event.go:65`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeTurnEvent {
    /// Go 標籤：`json:"event_id"`
    pub event_id: i64,
    /// Go 標籤：`json:"raw_camp"`
    pub raw_camp: i64,
    /// Go 標籤：`json:"handler"`
    pub handler: String,
    /// Go 標籤：`json:"staging"`
    pub staging: NativeTurnStaging,
    /// Go 標籤：`json:"dynamic_group,omitempty"`
    pub dynamic_group: Option<NativeTurnDynamicGroup>,
    /// Go 標籤：`json:"progression,omitempty"`
    pub progression: Option<NativeTurnProgression>,
    /// Go 標籤：`json:"pair_mutation,omitempty"`
    pub pair_mutation: Option<NativeTurnPairMutation>,
    /// 原始state索引、已證實下限及上限。
    /// Go 標籤：`json:"state_range,omitempty"`
    pub state_range: Option<[i64; 3]>,
    /// Actions 是控制列事件轉寫後降階的劇本動作（tools/sync_native_turn_events.py 由
    /// native_death_events.json 產生），依 0x1A813 的 selector 在該階段的可編輯事件之後執行。
    /// 用於由其他處理器 control_turn 動態啟用、不在靜態 turn_events 裡的列（第九章事件 31）。
    /// Go 標籤：`json:"actions,omitempty"`
    pub actions: Vec<Action>,
}

/// NativeTurnDynamicGroup 從 raw event-state table 解析一個 0x35822 group 參數，
/// 再套用該 caller 已證實的 live-row／state writes；它與 event63 這類固定 staging
/// calls 分開表示。
///
/// Go：`event.go:83`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeTurnProgression {
    /// Go 標籤：`json:"state_index"`
    pub state_index: i64,
    /// Go 標籤：`json:"repeat_until"`
    pub repeat_until: i64,
    /// Go 標籤：`json:"mark_unit_index"`
    pub mark_unit_index: i64,
    /// Go 標籤：`json:"mark_helper"`
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
    pub pulse_handler: String,
    /// Go 標籤：`json:"pulse_count"`
    pub pulse_count: i64,
    /// Go 標籤：`json:"extra_delay_ms"`
    pub extra_delay_ms: i64,
    /// Go 標籤：`json:"tail_text_indices"`
    pub tail_text_indices: Vec<i64>,
}

/// NativeTurnPairMutation 保存 event79 由一個 raw base 與單一步 RNG 選出兩個
/// 循環相鄰 runtime slots 的規則，不替 bit7 賦予 caller 之外的玩法名稱。
///
/// Go：`event.go:118`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeTurnStaging {
    /// Go 標籤：`json:"helper"`
    pub helper: String,
    /// Go 標籤：`json:"pan_helper"`
    pub pan_helper: String,
    /// Go 標籤：`json:"spawn_helper"`
    pub spawn_helper: String,
    /// Go 標籤：`json:"delay_before_flash_ms"`
    pub delay_before_flash_ms: i64,
    /// Go 標籤：`json:"palette_helper"`
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
    pub redraw_helper: String,
    /// Go 標籤：`json:"raw_placement_gate"`
    pub raw_placement_gate: i64,
    /// Go 標籤：`json:"calls"`
    pub calls: Vec<NativeTurnStagingCall>,
}

/// Go：`event.go:148`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeTurnStagingCall {
    /// Go 標籤：`json:"group"`
    pub group: i64,
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
    /// Go 標籤：`json:"source"`
    pub source: String,
}

/// InitialGroupAbsent is an evidence-backed pre-battle condition: materialize
/// its FDFIELD group only when the permanent party lacks CharID. It is not a
/// generic event expression and does not claim runtime-slot append identity.
///
/// Go：`event.go:158`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct InitialGroupAbsent {
    /// Go 標籤：`json:"char_id"`
    pub char_id: i64,
    /// Go 標籤：`json:"group"`
    pub group: i64,
}

/// PartyMember 主角隊成員(數值來自 characters.json / EXE 表)。
///
/// Go：`event.go:164`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct PartyMember {
    /// Go 標籤：`json:"name"`
    pub name: String,
    /// Go 標籤：`json:"cls"`
    pub cls: String,
    /// Fig is the stable JOIN/roster identity used by the remake's persistent
    /// party map. For a fresh native JOIN record it also seeds raw +7, but it is
    /// not the mutable map/battle selector after class change.
    /// Go 標籤：`json:"fig"`
    pub fig: i64,
    /// NativeIdentity is the optional persistent-record +0x08 key used by
    /// native 0x11506. It is intentionally separate from Fig/+7 selectors.
    /// Go 標籤：`json:"native_identity,omitempty"`
    pub native_identity: Option<i64>,
    /// Raw JOIN-constructor bytes written to persistent +0x1f/+0x20.
    /// Go 標籤：`json:"native_record_race,omitempty"`
    pub native_record_race: Option<u8>,
    /// Go 標籤：`json:"native_record_class,omitempty"`
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
    pub spells: Vec<i64>,
    /// InitialCommandMask is the exact four-byte constructor source for
    /// unit+0x1a..+0x1d. It is deliberately not derived from Spells.
    /// Go 標籤：`json:"initial_command_mask,omitempty"`
    pub initial_command_mask: Vec<u8>,
    /// Go 標籤：`json:"inventory,omitempty"`
    pub inventory: Vec<i64>,
    /// Go 標籤：`json:"inventory_slots,omitempty"`
    pub inventory_slots: Vec<i64>,
}

/// Event 一條事件規則。
///
/// Go：`event.go:199`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Event {
    /// Go 標籤：`json:"id"`
    pub id: String,
    /// on_battle_start / on_turn_end / on_unit_death
    /// Go 標籤：`json:"trigger"`
    pub trigger: String,
    /// Go 標籤：`json:"when,omitempty"`
    pub when: Option<When>,
    /// Go 標籤：`json:"do"`
    pub r#do: Vec<Action>,
    /// Go 標籤：`json:"once"`
    pub once: bool,
    pub(crate) fired: bool,
}

/// When 條件(可擴充:加欄位 + Match 加判斷)。
///
/// Go：`event.go:209`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct When {
    /// turn == N(0=不限)
    /// Go 標籤：`json:"turn,omitempty"`
    pub turn: i64,
    /// 某角色陣亡
    /// Go 標籤：`json:"unit_dead,omitempty"`
    pub unit_dead: String,
    /// 原版 runtime slot 已登場且存活
    /// Go 標籤：`json:"unit_slot_active,omitempty"`
    pub unit_slot_active: Option<i64>,
    /// battle-local raw byte index
    /// Go 標籤：`json:"native_event_state_index,omitempty"`
    pub native_event_state_index: Option<i64>,
    /// exact raw byte value
    /// Go 標籤：`json:"native_event_state_value,omitempty"`
    pub native_event_state_value: Option<i64>,
}

/// Action 動作(可擴充:加 type + execAction 加 case)。
///
/// Go：`event.go:218`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Action {
    /// Go 標籤：`json:"type"`
    pub r#type: String,
    /// spawn_group 的波次
    /// Go 標籤：`json:"groups,omitempty"`
    pub groups: Vec<i64>,
    /// turn_events 的原版事件編號
    /// Go 標籤：`json:"native_event_id,omitempty"`
    pub native_event_id: Option<i64>,
    /// 原版逐呼叫配置資料
    /// Go 標籤：`json:"native_spawns,omitempty"`
    pub native_spawns: Vec<NativeSpawnCall>,
    /// 增援陣營(改為)
    /// Go 標籤：`json:"camp,omitempty"`
    pub camp: String,
    /// 增援當回合可動(青衫「立即行動」)
    /// Go 標籤：`json:"act_immediately,omitempty"`
    pub act_immediately: bool,
    /// dialogue 說話者(DATO 肖像 id;-1=旁白)
    /// Go 標籤：`json:"speaker"`
    pub speaker: i64,
    /// dialogue 文本
    /// Go 標籤：`json:"text,omitempty"`
    pub text: String,
    /// set_flag
    /// Go 標籤：`json:"flag,omitempty"`
    pub flag: String,
    /// set_ai 目標
    /// Go 標籤：`json:"unit,omitempty"`
    pub unit: String,
    /// set_ai 模式(berserk…)
    /// Go 標籤：`json:"mode,omitempty"`
    pub mode: String,
    /// join_party: permanent player identity
    /// Go 標籤：`json:"char_id,omitempty"`
    pub char_id: i64,
    /// pan:原版 camera grid(col,row)，runtime 依地圖 tile 尺寸換 pixel
    /// Go 標籤：`json:"grid,omitempty"`
    pub grid: Option<[i64; 2]>,
    /// delay:原版毫秒數
    /// Go 標籤：`json:"ms,omitempty"`
    pub ms: i64,
    /// NativeActing 保存戰鬥事件呼叫 0x1366A 的資源與原始呼叫點。
    /// 它只描述已解碼的行為，不以位址作為重製端執行鍵。
    /// Go 標籤：`json:"native_acting,omitempty"`
    pub native_acting: Option<NativeFollowingActing>,
    /// Go 標籤：`json:"native_staging_present,omitempty"`
    pub native_staging_present: Option<NativeStagingPlacement>,
    /// NativeSource 保留無獨立結構可承載的原始寫入／對話呼叫點。
    /// Go 標籤：`json:"native_source,omitempty"`
    pub native_source: String,
    /// NativeTextIndex 是一次原版 0x15F84 呼叫的 FDTXT 索引；多句
    /// editable dialogue 可共用同一索引，但不能由文字內容反推。
    /// Go 標籤：`json:"native_text_index,omitempty"`
    pub native_text_index: Option<i64>,
    /// NativeDialogueRef 將戰鬥事件接到同一份故事翻譯與原始逐句版面。
    /// Go 標籤：`json:"native_dialogue_ref,omitempty"`
    pub native_dialogue_ref: Option<NativeEventDialogue>,
    /// EventStateIndex/Value 僅供已證實的 battle-local raw byte 寫入。
    /// Go 標籤：`json:"event_state_index,omitempty"`
    pub event_state_index: Option<i64>,
    /// Go 標籤：`json:"event_state_value,omitempty"`
    pub event_state_value: Option<i64>,
    /// NativeWhen 是原版處理器裡夾在動作之間的條件，執行到這個動作時才判斷。
    /// Go 標籤：`json:"native_when,omitempty"`
    pub native_when: Option<NativeActionWhen>,
    /// NativeDeathOp 承載死亡事件處理器裡沒有既有動作種類可表達的原語。
    /// Go 標籤：`json:"native_death_op,omitempty"`
    pub native_death_op: Option<NativeDeathOp>,
}

/// NativeStagingPlacement 保存33F78的slot/x/y，consumer仍走12CEA及22253。
///
/// Go：`event.go:254`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
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
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeEventDialogue {
    /// Go 標籤：`json:"script"`
    pub script: String,
    /// Go 標籤：`json:"scene_index"`
    pub scene_index: i64,
    /// Go 標籤：`json:"line"`
    pub line: i64,
    /// Go 標籤：`json:"source_dat"`
    pub source_dat: String,
    /// Go 標籤：`json:"string_index"`
    pub string_index: i64,
    /// Go 標籤：`json:"utterance"`
    pub utterance: i64,
    /// Go 標籤：`json:"control"`
    pub control: String,
    /// Go 標籤：`json:"operand"`
    pub operand: i64,
    /// Go 標籤：`json:"pages"`
    pub pages: Vec<Vec<String>>,
    /// Go 標籤：`json:"glyph_pages,omitempty"`
    pub glyph_pages: Vec<Vec<Vec<String>>>,
    /// Go 標籤：`json:"glyph_ids,omitempty"`
    pub glyph_ids: Vec<Vec<Vec<i64>>>,
}

/// NativeSpawnCall 保存全域事件處理器的一個確切呼叫點。Group 是該排程回合
/// 解析出的值；Source、Via 與 RawPlacementGate 直接來自 EXE，不從陣營或群組推測。
///
/// Go：`event.go:277`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeSpawnCall {
    /// Go 標籤：`json:"group"`
    pub group: i64,
    /// Go 標籤：`json:"via"`
    pub via: String,
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"raw_placement_gate"`
    pub raw_placement_gate: Option<i64>,
    /// Go 標籤：`json:"following_acting,omitempty"`
    pub following_acting: Option<NativeFollowingActing>,
    /// GroupFromState 非 nil 時 group 在執行當下讀戰場狀態表 [0x53AD5]+index（第九章事件 31
    /// `0x34B67 movzx eax, byte [eax+0x10]; push eax; call 0x10B4E`），Group 不使用。
    /// Go 標籤：`json:"group_from_state,omitempty"`
    pub group_from_state: Option<i64>,
}

/// NativeFollowingActing 保存 wrapper 返回後、由同一呼叫端明確執行的 ACTING。
/// 它不是由 group 推導的，也不是 0x32999 函式本體的一部分。
///
/// Go：`event.go:290`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeFollowingActing {
    /// Go 標籤：`json:"resource"`
    pub resource: i64,
    /// Go 標籤：`json:"source"`
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
    todo!("battle/event.go:LoadScenario")
}

impl Scenario {
    /// Setup 是既有測試與無原生欄位規則劇本的相容入口。正式執行路徑必須使用
    /// SetupChecked，避免把資料綁定錯誤誤當成沒有開場對話。
    ///
    /// Go：`event.go:541`，`func (sc *Scenario) Setup(st *State) []DialogLine`
    pub fn setup(&mut self, st: Option<&mut State>) -> Vec<DialogLine> {
        todo!("battle/event.go:Scenario.Setup")
    }

    /// SetupChecked 套用劇本初始狀態並回報原生規則綁定錯誤；正式執行期以此
    /// 入口維持失敗即關閉。
    ///
    /// Go：`event.go:548`，`func (sc *Scenario) SetupChecked(st *State) ([]DialogLine, error)`
    pub fn setup_checked(&mut self, st: Option<&mut State>) -> Result<Vec<DialogLine>> {
        todo!("battle/event.go:Scenario.SetupChecked")
    }

    /// Go：`event.go:588`，`func (sc *Scenario) bindNativeFieldEventRules(st *State) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn bind_native_field_event_rules_inner(&self, st: Option<&mut State>) -> Result<()> {
        todo!("battle/event.go:Scenario.bindNativeFieldEventRules")
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
        todo!("battle/event.go:Scenario.materializePendingGroups")
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
        todo!("battle/event.go:Scenario.NativeTurnActionEventsAt")
    }

    /// Go：`event.go:665`，`func (sc *Scenario) NativeTurnEventsAt(st *State, rawCamp byte) ([]NativeTurnEvent, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_turn_events_at(&self, st: Option<&State>, raw_camp: u8) -> Result<Vec<NativeTurnEvent>> {
        todo!("battle/event.go:Scenario.NativeTurnEventsAt")
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
        todo!("battle/event.go:Scenario.AdoptHandlerBattleState")
    }

    /// Fire 對某 trigger 評估所有事件,執行符合者的動作。回傳要播的對話(含說話者)。
    /// ctxUnit:on_unit_death 時傳陣亡者名。
    ///
    /// Go：`event.go:735`，`func (sc *Scenario) Fire(st *State, trigger, ctxUnit string) []DialogLine`
    pub fn fire(&mut self, st: Option<&mut State>, trigger: &str, ctx_unit: &str) -> Vec<DialogLine> {
        todo!("battle/event.go:Scenario.Fire")
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
        todo!("battle/event.go:Scenario.TriggerActionsWhere")
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
        todo!("battle/event.go:Event.NativeTurnPhaseSelector")
    }
}
impl When {
    /// match 條件判斷(可擴充)。nil = 無條件,恆真。
    ///
    /// Go：`event.go:797`，`func (w *When) match(st *State, ctxUnit string) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub(crate) fn r#match(&self, st: Option<&State>, ctx_unit: &str) -> bool {
        todo!("battle/event.go:When.match")
    }
}
impl Scenario {
    /// ExecuteAction 執行單一狀態動作。pan/delay 由 UI runner 阻塞處理；
    /// 回傳 (對話, true) 表示 runner 應停下並播放這句。
    ///
    /// Go：`event.go:829`，`func (sc *Scenario) ExecuteAction(st *State, a Action) (DialogLine, bool)`
    pub fn execute_action(&mut self, st: Option<&mut State>, a: &Action) -> (DialogLine, bool) {
        todo!("battle/event.go:Scenario.ExecuteAction")
    }

    /// ExecuteActionChecked 是正式執行路徑的錯誤回報邊界。上方相容包裝保留給舊的
    /// 同步呼叫者；介面執行器使用本函式，避免原版增援資料錯誤被靜默忽略。
    ///
    /// Go：`event.go:836`，`func (sc *Scenario) ExecuteActionChecked(st *State, a Action) (DialogLine, bool, error)`
    pub fn execute_action_checked(&mut self, st: Option<&mut State>, a: &Action) -> Result<(DialogLine, bool)> {
        todo!("battle/event.go:Scenario.ExecuteActionChecked")
    }

    /// TakePartyJoins transfers JOIN effects from battle-script execution to the
    /// campaign-owned persistent roster. Scenario stays independent from the UI /
    /// save layer while preserving the original ordering (JOIN before SPAWN).
    ///
    /// Go：`event.go:925`，`func (sc *Scenario) TakePartyJoins() []int`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn take_party_joins(&mut self) -> Vec<i64> {
        todo!("battle/event.go:Scenario.TakePartyJoins")
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
        todo!("battle/event.go:Scenario.PartyUnits")
    }
}
