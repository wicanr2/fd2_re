//! Go 來源：`remake/internal/campaign/campaign.go`（1017 行）。
//!
//! Package campaign — 劇本節點圖系統(doc 19):把「固定線性流程」變成可分支的有向圖。
//! 節點 = 一個遊戲段落(story/battle/town/preparation/choice/event/ending),轉場依結果(win/lose/next/optN)
//! 與旗標決定下一節點;敗北可走敗北路線而非 game over。

use super::*;

/// Line 一句對話(story 節點內嵌)。Speaker 是靜態或稽核用 DATO 頭像 id；
/// SpeakerSlot 保存原版 FFED/FFEC 的 runtime unit direct index，執行時必須
/// 從該 unit 的 Portrait 解析，不能把 slot 數字誤當全域角色 id。
///
/// Go：`campaign.go:15`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Line {
    /// Go 標籤：`json:"line_id,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub line_id: String,
    /// Go 標籤：`json:"speaker"`
    pub speaker: i64,
    /// Go 標籤：`json:"speaker_slot,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub speaker_slot: Option<i64>,
    /// Go 標籤：`json:"upper,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub upper: Option<bool>,
    /// Go 標籤：`json:"text"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub text: String,
}

/// Option choice 節點的選項;If 非空時需旗標為真才顯示。
///
/// Go：`campaign.go:24`
/// Rust 名稱改為 `ChoiceOption`：Go 名稱 `Option` 與 Rust prelude 相撞。
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct ChoiceOption {
    /// Go 標籤：`json:"label"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub label: String,
    /// Go 標籤：`json:"to"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub to: String,
    /// Go 標籤：`json:"if,omitempty"`
    #[serde(
        rename = "if",
        deserialize_with = "fd2_rules::battle::null_as_default",
        skip_serializing_if = "String::is_empty"
    )]
    pub r#if: String,
}

/// NativeTownSecretGate is the editable form of the 0x2cde0..0x2cef7
/// three-byte town-record gate. Hidden selection 5 is entered only when the
/// current visible selection and BIOS function-key scan both match.
///
/// Go：`campaign.go:33`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeTownSecretGate {
    /// Go 標籤：`json:"selection"`
    pub selection: i64,
    /// Go 標籤：`json:"scan_code"`
    pub scan_code: i64,
    /// Go 標籤：`json:"to"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub to: String,
}

/// Good 商店商品(名稱/價格;id 對映 EXE item.json)。
///
/// Go：`campaign.go:40`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Good {
    /// Go 標籤：`json:"id"`
    pub id: i64,
    /// Go 標籤：`json:"name"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub name: String,
    /// Go 標籤：`json:"price"`
    pub price: i64,
}

/// Actor story+Map 場景背景上的靜態角色(cutscene NPC/主角,無 AI/戰鬥邏輯,純擺位;
/// Fig=地圖 sprite 組 id(同 battle.Unit.Fig,恆等於角色 id,doc31);Dir:0下1左2上3右。
/// 座標/肖像多可直接取自 FDFIELD 該地圖出場位置段(見 tools/parse_field.py positions),
/// 見 doc23 §4 補述(map32 王座廳:國王portrait48@(7,5)/王后portrait66@(10,5))。
///
/// FromX/FromY/WalkFrames(doc46 §5.3,走位動畫):若 WalkFrames>0 且 From 與 X/Y 不同,
/// 進節點時該角色從 (FromX,FromY) 走位到 (X,Y),耗時 WalkFrames 幀(60fps);
/// 例:map31 密林「發現悠妮與蓋亞」一幕,索爾/亞雷斯從比劍點走到發現點(FDFIELD 出場位置證實
/// 兩組座標相距 14 格,非同格瞬移,見 doc46 §2)。
///
/// Go：`campaign.go:55`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Actor {
    /// Go 標籤：`json:"fig"`
    pub fig: i64,
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
    /// Go 標籤：`json:"dir,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub dir: i64,
    /// Go 標籤：`json:"from_x,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub from_x: i64,
    /// Go 標籤：`json:"from_y,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub from_y: i64,
    /// Go 標籤：`json:"walk_frames,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub walk_frames: i64,
}

/// ActorWalk 節點「退場」走位動畫(doc46 §5.3):對白播完、換場前,已在場的 actor 先走一段路
/// 再淡出(例:王座廳索爾對白說完沿紅毯走下場,~1.5s)。Fig 指定 Node.Actors 裡哪一個角色。
///
/// Go：`campaign.go:67`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct ActorWalk {
    /// Go 標籤：`json:"fig"`
    pub fig: i64,
    /// Go 標籤：`json:"to_x"`
    pub to_x: i64,
    /// Go 標籤：`json:"to_y"`
    pub to_y: i64,
    /// Go 標籤：`json:"frames"`
    pub frames: i64,
    /// 走完後面向(指標,nil=保留走位末向;指定則覆蓋,如索爾走到亞雷斯旁定住面右)
    /// Go 標籤：`json:"dir,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub dir: Option<i64>,
}

/// ActingUnit is one original acting frame target.  Slot is the original
/// FDFIELD/unit-array index and is the authoritative identity for decoded
/// 0x1366a data: a roster can contain many guards with the same Fig.  Fig is a
/// legacy fallback for hand-authored scenes that have no original roster; an
/// editable transcription of a decoded handler must use Slot instead.  Pose
/// follows the original direction encoding: 0 down, 1 left, 2 up, 3 right.
///
/// Go：`campaign.go:81`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct ActingUnit {
    /// Go 標籤：`json:"slot,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub slot: Option<i64>,
    /// Go 標籤：`json:"fig,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub fig: i64,
    /// Go 標籤：`json:"pose"`
    pub pose: i64,
}

/// ActingFrame 是原版 0x1366a 資源的一幀行為轉錄，不包含原始 bytes。
/// Special=false = bit7=0 正常模式：每個 Beat 沿 Pose 移動一格；Special=true = bit7=1：
/// 原地顯示／姿態，Beat 只表示停留節奏。規則見 doc50 §1.2。
///
/// Go：`campaign.go:90`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct ActingFrame {
    /// Go 標籤：`json:"beats"`
    pub beats: i64,
    /// Go 標籤：`json:"special,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub special: bool,
    /// Go 標籤：`json:"units"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub units: Vec<ActingUnit>,
}

/// LoadCHState is the editable remake state selected by an original LOADCH
/// call.  The original routine loads all three together: FDFIELD map, FDFIELD
/// roster, and FDTXT (doc23 §4).  Keeping those paths in one value makes an
/// incomplete reconstruction impossible to mistake for a harmless map change.
/// Paths are asset-root relative (for example assets/maps/map5/map5_units.json)
/// rather than relative to the handler binding file.
///
/// Go：`campaign.go:102`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct LoadCHState {
    /// Go 標籤：`json:"chapter"`
    pub chapter: i64,
    /// Go 標籤：`json:"map"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub map: String,
    /// Go 標籤：`json:"roster"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub roster: String,
    /// Go 標籤：`json:"slot_count"`
    pub slot_count: i64,
    /// selector30的模板列數與部署容量各自驗證。
    /// Go 標籤：`json:"source_slot_count,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub source_slot_count: Option<i64>,
    /// Go 標籤：`json:"script"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub script: String,
    /// persistent party constructed before FDFIELD groups
    /// Go 標籤：`json:"party_scenario,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub party_scenario: String,
    /// original JOIN chronology; direct-replay fallback and runtime assertion
    /// Go 標籤：`json:"party_order,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub party_order: Vec<i64>,
    /// selector30讀全部持續名冊，不沿用19人選擇。
    /// Go 標籤：`json:"all_persistent_party,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub all_persistent_party: bool,
    /// caller在LOADCH後、首次redraw前的已證實六全域。
    /// Go 標籤：`json:"post_load_view,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub post_load_view: Option<NativeMapViewConfig>,
    /// Go 標籤：`json:"cam_x,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub cam_x: i64,
    /// Go 標籤：`json:"cam_y,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub cam_y: i64,
    /// Go 標籤：`json:"cam_max_y,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub cam_max_y: i64,
}

/// BeatCondition is the runtime form of a proven handler predicate.
///
/// Go：`campaign.go:119`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct BeatCondition {
    /// Go 標籤：`json:"op"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub op: String,
    /// Go 標籤：`json:"unit_slots,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub unit_slots: Vec<i64>,
    /// Go 標籤：`json:"native_inventory_item_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_inventory_item_id: Option<i64>,
    /// Go 標籤：`json:"native_persistent_identity,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_persistent_identity: Option<i64>,
    /// Go 標籤：`json:"threshold,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub threshold: Option<i64>,
    /// Go 標籤：`json:"native_round,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_round: Option<i64>,
    /// Go 標籤：`json:"native_record_word_offset,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_word_offset: Option<i64>,
    /// Go 標籤：`json:"native_record_word_value,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_record_word_value: Option<i64>,
    /// Go 標籤：`json:"unit_slot,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unit_slot: Option<i64>,
    /// Go 標籤：`json:"any,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub any: Vec<BeatCondition>,
    /// Go 標籤：`json:"char_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub char_id: Option<i64>,
    /// Go 標籤：`json:"event_state_index,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub event_state_index: Option<i64>,
    /// Go 標籤：`json:"event_state_value,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub event_state_value: Option<i64>,
    /// Go 標籤：`json:"required_slot_count,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub required_slot_count: Option<i64>,
}

/// HandlerUnitLayout is one absolute runtime-slot placement recovered from a
/// native post-battle layout routine. Coordinates remain original map tiles;
/// CamX/CamY below are the verified remake pixel origin.
///
/// Go：`campaign.go:139`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerUnitLayout {
    /// Go 標籤：`json:"slot"`
    pub slot: i64,
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
    /// Go 標籤：`json:"pose"`
    pub pose: i64,
}

/// Go：`campaign.go:146`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerLayout {
    /// Go 標籤：`json:"units"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub units: Vec<HandlerUnitLayout>,
    /// Go 標籤：`json:"cam_x"`
    pub cam_x: i64,
    /// Go 標籤：`json:"cam_y"`
    pub cam_y: i64,
}

/// NativeMapViewConfig is an explicitly sourced original 13x8 tactical view.
/// Values are map tiles, not remake pixels.
///
/// Go：`campaign.go:154`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeMapViewConfig {
    /// Go 標籤：`json:"camera_x"`
    pub camera_x: i64,
    /// Go 標籤：`json:"camera_y"`
    pub camera_y: i64,
    /// Go 標籤：`json:"cursor_x"`
    pub cursor_x: i64,
    /// Go 標籤：`json:"cursor_y"`
    pub cursor_y: i64,
    /// Go 標籤：`json:"visible_cursor_x"`
    pub visible_cursor_x: i64,
    /// Go 標籤：`json:"visible_cursor_y"`
    pub visible_cursor_y: i64,
    /// Go 標籤：`json:"range_mode"`
    pub range_mode: Option<i64>,
}

/// NativeMapHUDConfig carries only the persistent 0x1acf3 raw globals. Gate
/// bytes remain byte-valued because native code tests nonzero, not boolean one.
///
/// Go：`campaign.go:166`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeMapHUDConfig {
    /// Go 標籤：`json:"display_gate_a"`
    pub display_gate_a: i64,
    /// Go 標籤：`json:"display_gate_b"`
    pub display_gate_b: i64,
    /// Go 標籤：`json:"anchor_x"`
    pub anchor_x: i64,
}

/// NativeMapHUDInheritedConfig declares a battle-controller gate B while gate A
/// and the anchor are inherited from their separately proven persistent owner.
/// The original currently proves only controller entry value 1.
///
/// Go：`campaign.go:175`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeMapHUDInheritedConfig {
    /// Go 標籤：`json:"display_gate_b"`
    pub display_gate_b: i64,
}

/// HandlerIndexedTransition records the recovered 0x24618 double-buffer
/// choreography. Tile geometry, radial radius progression, and row bounds are
/// kept explicit because this is not a generic fade; the PNG renderer may only
/// execute it once an indexed descriptor adapter is available.
///
/// Go：`campaign.go:183`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerIndexedTransition {
    /// Go 標籤：`json:"tile_x"`
    pub tile_x: i64,
    /// Go 標籤：`json:"tile_y"`
    pub tile_y: i64,
    /// CursorSource optionally resolves the first two native 0x24618 arguments
    /// from the battle's raw relative cursor globals.  The only accepted source
    /// is native_relative_cursor, which preserves [0x53ab9]/[0x53abd] rather than
    /// replacing them with a guessed absolute map coordinate.
    /// Go 標籤：`json:"cursor_source,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub cursor_source: String,
    /// Go 標籤：`json:"cursor_x_offset,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub cursor_x_offset: i64,
    /// Go 標籤：`json:"cursor_y_offset,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub cursor_y_offset: i64,
    /// RadialRadius and RadialRadiusStep are the third/fourth native arguments.
    /// 0x24618 advances the radius on every pass; 0x22046 passes it to the two
    /// 0x219ad radial LUT-remap calls and derives its final rect radius from it.
    /// They are not source-buffer coordinates or the radial scale (which is
    /// fixed to 16 by 0x22046).
    /// Go 標籤：`json:"radial_radius"`
    pub radial_radius: i64,
    /// Go 標籤：`json:"radial_radius_step"`
    pub radial_radius_step: i64,
    /// StartY/EndY are the row range passed to 0x22046; this routine reads its
    /// indexed buffers from globals, so these are not source-image coordinates
    /// or a blit width. EndY is exclusive.
    /// Go 標籤：`json:"start_y"`
    pub start_y: i64,
    /// Go 標籤：`json:"end_y"`
    pub end_y: i64,
    /// Go 標籤：`json:"clip_width"`
    pub clip_width: i64,
    /// Go 標籤：`json:"clip_height"`
    pub clip_height: i64,
    /// Go 標籤：`json:"frames"`
    pub frames: i64,
    /// Go 標籤：`json:"frame_delay_ms"`
    pub frame_delay_ms: i64,
    /// Go 標籤：`json:"tail_delay_ms"`
    pub tail_delay_ms: i64,
    /// Go 標籤：`json:"palette_range_start"`
    pub palette_range_start: i64,
    /// Go 標籤：`json:"palette_range_end"`
    pub palette_range_end: i64,
    /// Go 標籤：`json:"palette_delta_start"`
    pub palette_delta_start: i64,
    /// Go 標籤：`json:"palette_delta_end"`
    pub palette_delta_end: i64,
    /// Go 標籤：`json:"palette_delta_step"`
    pub palette_delta_step: i64,
    /// Go 標籤：`json:"palette_delay_ms"`
    pub palette_delay_ms: i64,
}

/// NativePaletteFadeOut is the exact 0x1f882 DAC-darkening schedule.  It is
/// separate from palette_update (0x11df2) and generic story fades: an indexed
/// DAC adapter must execute every inclusive step before a handler can play it.
///
/// Go：`campaign.go:221`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativePaletteFadeOut {
    /// Go 標籤：`json:"start"`
    pub start: i64,
    /// Go 標籤：`json:"end"`
    pub end: i64,
    /// Go 標籤：`json:"delay_ms"`
    pub delay_ms: i64,
}

/// NativePaletteFadeIn is the exact 0x1f525 whole-DAC schedule. The helper
/// writes baseline-minus-delta for inclusive deltas 64..0, waiting 2ms after
/// every write. It is not the remake's generic RGBA story fade.
///
/// Go：`campaign.go:230`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativePaletteFadeIn {
    /// Go 標籤：`json:"start"`
    pub start: i64,
    /// Go 標籤：`json:"end"`
    pub end: i64,
    /// Go 標籤：`json:"delay_ms"`
    pub delay_ms: i64,
}

/// HandlerRawUnitBytePatch keeps an original record offset visible instead of
/// assigning a gameplay name from one handler write. Runtime currently accepts
/// only offsets with an independently preserved typed storage location.
///
/// Go：`campaign.go:239`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerRawUnitBytePatch {
    /// Go 標籤：`json:"offset"`
    pub offset: i64,
    /// Go 標籤：`json:"value"`
    pub value: i64,
}

/// HandlerUnitRecordPatch is a sparse direct-write transcription. Pointer
/// fields distinguish an explicit zero from an omitted write; untouched record
/// bytes remain untouched.
///
/// Go：`campaign.go:247`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerUnitRecordPatch {
    /// Go 標籤：`json:"slot"`
    pub slot: i64,
    /// Go 標籤：`json:"x,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub x: Option<i64>,
    /// Go 標籤：`json:"y,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub y: Option<i64>,
    /// Go 標籤：`json:"pose,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub pose: Option<i64>,
    /// Go 標籤：`json:"raw_bytes,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub raw_bytes: Vec<HandlerRawUnitBytePatch>,
}

/// HandlerDirectRecordPatch represents hard-coded writes that occur between
/// calls and were therefore invisible to the call-only handler exporter. View
/// uses the already recovered raw camera/cursor globals; values stay additive
/// evidence tied to the source address in Beat.Source.
///
/// Go：`campaign.go:259`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerDirectRecordPatch {
    /// Go 標籤：`json:"units"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub units: Vec<HandlerUnitRecordPatch>,
    /// Go 標籤：`json:"view,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub view: Option<NativeMapViewConfig>,
}

/// NativePalettePulse is the exact whole-DAC pulse performed by 0x35e5a.
/// It is deliberately not represented as a generic fade: the first and
/// second ramps have different inclusive bounds and are separated by a hold.
/// An indexed DAC adapter must execute this schedule before a handler can play
/// it.
///
/// Go：`campaign.go:269`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativePalettePulse {
    /// Go 標籤：`json:"rise_start"`
    pub rise_start: i64,
    /// Go 標籤：`json:"rise_end"`
    pub rise_end: i64,
    /// Go 標籤：`json:"rise_delay_ms"`
    pub rise_delay_ms: i64,
    /// Go 標籤：`json:"hold_ms"`
    pub hold_ms: i64,
    /// Go 標籤：`json:"fall_start"`
    pub fall_start: i64,
    /// Go 標籤：`json:"fall_end"`
    pub fall_end: i64,
    /// Go 標籤：`json:"fall_delay_ms"`
    pub fall_delay_ms: i64,
}

/// NativePaletteBlackout is the exact ch07 post-battle terminal operation:
/// 0x11d40 subtracts 64 from every six-bit DAC component in entries 0..255,
/// which clamps the complete palette to black, and the caller immediately
/// clears all 0xFA00 mode-13h framebuffer bytes.  Keep the framebuffer clear
/// extent explicit so this cannot become a generic or guessed fade.
///
/// Go：`campaign.go:284`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativePaletteBlackout {
    /// Go 標籤：`json:"start"`
    pub start: i64,
    /// Go 標籤：`json:"end"`
    pub end: i64,
    /// Go 標籤：`json:"delta"`
    pub delta: i64,
    /// Go 標籤：`json:"clear_bytes"`
    pub clear_bytes: i64,
}

/// NativeCh20SkyKeySequence 是 raw ch20 post 的固定合成演出契約。它保留
/// 0x24336 內部的鏡頭格、FDOTHER 幀界線、AFM 資源與兩段色盤停留，不把
/// 這段 caller-specific choreography 泛化成任意動畫播放器。
///
/// Go：`campaign.go:294`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeCh20SkyKeySequence {
    /// Go 標籤：`json:"pan_grid_x"`
    pub pan_grid_x: i64,
    /// Go 標籤：`json:"pan_grid_y"`
    pub pan_grid_y: i64,
    /// Go 標籤：`json:"fdother_resource"`
    pub fdother_resource: i64,
    /// Go 標籤：`json:"fdother_frame_count"`
    pub fdother_frame_count: i64,
    /// Go 標籤：`json:"base_frame"`
    pub base_frame: i64,
    /// Go 標籤：`json:"first_frame_start"`
    pub first_frame_start: i64,
    /// Go 標籤：`json:"first_frame_end"`
    pub first_frame_end: i64,
    /// Go 標籤：`json:"frame_wait_bios_ticks"`
    pub frame_wait_bios_ticks: i64,
    /// Go 標籤：`json:"palette_cycle_first"`
    pub palette_cycle_first: bool,
    /// Go 標籤：`json:"ani_resource"`
    pub ani_resource: i64,
    /// Go 標籤：`json:"ani_frame_count"`
    pub ani_frame_count: i64,
    /// Go 標籤：`json:"ani_frame_delay_ms"`
    pub ani_frame_delay_ms: i64,
    /// Go 標籤：`json:"ani_skippable"`
    pub ani_skippable: bool,
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
    /// Go 標籤：`json:"restore_hold_ms"`
    pub restore_hold_ms: i64,
    /// Go 標籤：`json:"tail_frame_start"`
    pub tail_frame_start: i64,
    /// Go 標籤：`json:"tail_frame_end"`
    pub tail_frame_end: i64,
}

impl NativeCh20SkyKeySequence {
    /// IsRecoveredContract 限制正式執行器只接受 IDA／Capstone 已共同固定的
    /// 0x24336 payload。可編輯 JSON 仍完整顯示每個值，但改動後必須先補新證據，
    /// 不會悄悄取得原版語意。
    ///
    /// Go：`campaign.go:321`，`func (s *NativeCh20SkyKeySequence) IsRecoveredContract() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn is_recovered_contract(&self) -> bool {
        let s = self;
        s.pan_grid_x == 14
            && s.pan_grid_y == 8
            && s.fdother_resource == 34
            && s.fdother_frame_count == 101
            && s.base_frame == 0
            && s.first_frame_start == 1
            && s.first_frame_end == 68
            && s.frame_wait_bios_ticks == 3
            && s.palette_cycle_first
            && s.ani_resource == 0
            && s.ani_frame_count == 96
            && s.ani_frame_delay_ms == 15
            && !s.ani_skippable
            && s.palette_start == 0
            && s.palette_end == 255
            && s.flash_delta == 63
            && s.flash_hold_ms == 100
            && s.restore_delta == 0
            && s.restore_hold_ms == 500
            && s.tail_frame_start == 69
            && s.tail_frame_end == 100
    }
}

/// NativeStagingPresent preserves the exact 0x33f78 wrapper ABI. It focuses
/// (X,Y), then invokes 0x22253 with (Slot,X,Y,X,Y).  Focus coordinates are not
/// duplicated: the old FocusX=Slot/FocusY=X projection was disproved by the
/// wrapper's original stack offsets.
///
/// Go：`campaign.go:339`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeStagingPresent {
    /// Go 標籤：`json:"slot"`
    pub slot: i64,
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
}

/// NativeUnitPresent preserves 0x22253's five-argument ABI. LastRuntimeSlot is
/// source-specific evidence for ch28 post's [0x53BEB]-1 caller; it is not a
/// general script selector and is accepted only for call site 0x25535.
///
/// Go：`campaign.go:348`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeUnitPresent {
    /// Go 標籤：`json:"slot,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub slot: i64,
    /// Go 標籤：`json:"last_runtime_slot,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub last_runtime_slot: bool,
    /// Go 標籤：`json:"new_x"`
    pub new_x: i64,
    /// Go 標籤：`json:"new_y"`
    pub new_y: i64,
    /// Go 標籤：`json:"visual_x"`
    pub visual_x: i64,
    /// Go 標籤：`json:"visual_y"`
    pub visual_y: i64,
}

/// NativeCh28PostPresent preserves sub_35BBA(20) plus its mandatory sub_1DB65
/// consumer. Fields are raw archive/index/buffer contracts, not inferred visual
/// or audio names.
///
/// Go：`campaign.go:360`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeCh28PostPresent {
    /// Go 標籤：`json:"start_slot"`
    pub start_slot: i64,
    /// Go 標籤：`json:"pose_frames"`
    pub pose_frames: i64,
    /// Go 標籤：`json:"fdother_resource"`
    pub fdother_resource: i64,
    /// Go 標籤：`json:"entry_first"`
    pub entry_first: i64,
    /// Go 標籤：`json:"entry_last"`
    pub entry_last: i64,
    /// Go 標籤：`json:"sfx_resource"`
    pub sfx_resource: i64,
    /// Go 標籤：`json:"sfx_index"`
    pub sfx_index: i64,
    /// Go 標籤：`json:"sfx_arg"`
    pub sfx_arg: i64,
    /// Go 標籤：`json:"work_size"`
    pub work_size: i64,
    /// Go 標籤：`json:"work_stride"`
    pub work_stride: i64,
    /// Go 標籤：`json:"view_base"`
    pub view_base: i64,
    /// Go 標籤：`json:"visible_columns"`
    pub visible_columns: i64,
    /// Go 標籤：`json:"visible_rows"`
    pub visible_rows: i64,
}

impl NativeCh28PostPresent {
    /// Go：`campaign.go:376`，`func (p NativeCh28PostPresent) IsRecoveredContract() bool`
    pub fn is_recovered_contract(self) -> bool {
        let p = self;
        p.start_slot == 20
            && p.pose_frames == 13
            && p.fdother_resource == 5
            && p.entry_first == 0x44
            && p.entry_last == 0x4f
            && p.sfx_resource == 31
            && p.sfx_index == 3
            && p.sfx_arg == 1
            && p.work_size == 0x25680
            && p.work_stride == 0x1c8
            && p.view_base == 0x8088
            && p.visible_columns == 13
            && p.visible_rows == 8
    }
}

/// HandlerUnitPresent retains the formerly recovered metadata shape for native
/// 0x22253. It is not currently compilable: later direct trace found 11+6+10
/// presentation phases, so this six-frame schema is deliberately rejected
/// until a full indexed choreography representation exists.
///
/// Go：`campaign.go:388`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct HandlerUnitPresent {
    /// Go 標籤：`json:"slot"`
    pub slot: i64,
    /// Go 標籤：`json:"x"`
    pub x: i64,
    /// Go 標籤：`json:"y"`
    pub y: i64,
    /// Go 標籤：`json:"frames"`
    pub frames: i64,
    /// Go 標籤：`json:"frame_delay_ms"`
    pub frame_delay_ms: i64,
    /// Go 標籤：`json:"tail_ticks"`
    pub tail_ticks: i64,
}

/// Beat 過場原語(doc 50 §1/§2):cutscene 節點的 beats 通常依序執行；if 會在 runtime
/// 選一條 structured arm 插入目前拍之後，再回到共同 continuation。
/// 它可逐拍承接已有直接證據的原版 EXE handler 呼叫序列
/// (LOADCH/PAN/TXT/ACT/SPAWN/JOIN/BGM/FADE/DELAY)；單有 Beat 型別不代表特定 handler
/// 已完整閉合或達到玩家 E2。
/// 每個 op 只用到自己相關的欄位,其餘留零值即可(同 Node 的稀疏欄位風格)。
///
/// Go：`campaign.go:403`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Beat {
    /// loadch/pan/walk/dialog/act/spawn/spawn_intro/deactivate_unit/reactivate_nonzero_hp/reset_pose/redraw/...
    /// Go 標籤：`json:"op"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub op: String,
    /// original handler call-site; empty for authored-only beats
    /// Go 標籤：`json:"source,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub source: String,
    /// Go 標籤：`json:"condition,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub condition: Option<BeatCondition>,
    /// Go 標籤：`json:"then,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub then: Vec<Beat>,
    /// Go 標籤：`json:"else,omitempty"`
    #[serde(
        rename = "else",
        deserialize_with = "fd2_rules::battle::null_as_default",
        skip_serializing_if = "Vec::is_empty"
    )]
    pub r#else: Vec<Beat>,
    /// Go 標籤：`json:"runtime_context,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub runtime_context: Option<HandlerRuntimeContext>,
    /// Go 標籤：`json:"layout,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub layout: Option<HandlerLayout>,
    /// Go 標籤：`json:"indexed_transition,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub indexed_transition: Option<HandlerIndexedTransition>,
    /// Go 標籤：`json:"native_palette_fade_out,omitempty"`
    #[serde(rename = "native_palette_fade_out", skip_serializing_if = "Option::is_none")]
    pub native_palette_fade: Option<NativePaletteFadeOut>,
    /// Go 標籤：`json:"native_palette_fade_in,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_palette_fade_in: Option<NativePaletteFadeIn>,
    /// Go 標籤：`json:"native_palette_pulse,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_palette_pulse: Option<NativePalettePulse>,
    /// Go 標籤：`json:"native_palette_blackout,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_palette_blackout: Option<NativePaletteBlackout>,
    /// Go 標籤：`json:"native_ch20_sky_key_sequence,omitempty"`
    #[serde(rename = "native_ch20_sky_key_sequence", skip_serializing_if = "Option::is_none")]
    pub native_ch20_sky_key: Option<NativeCh20SkyKeySequence>,
    /// Go 標籤：`json:"native_staging_present,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_staging_present: Option<NativeStagingPresent>,
    /// Go 標籤：`json:"native_unit_present,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_unit_present: Option<NativeUnitPresent>,
    /// Go 標籤：`json:"native_ch28_post_present,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_ch28_post_present: Option<NativeCh28PostPresent>,
    /// Go 標籤：`json:"native_ch23_loop,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_ch23_loop: Option<NativeCh23Loop>,
    /// Go 標籤：`json:"native_2189a_loop,omitempty"`
    #[serde(rename = "native_2189a_loop", skip_serializing_if = "Option::is_none")]
    pub native2189_a_loop: Option<Native2189ALoop>,
    /// Go 標籤：`json:"unit_present,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub unit_present: Option<HandlerUnitPresent>,
    /// Go 標籤：`json:"direct_record_patch,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub direct_record_patch: Option<HandlerDirectRecordPatch>,
    /// loadch: atomically replace the active map, FDFIELD roster and FDTXT
    /// story context.  It is deliberately a nested required state object so a
    /// handler cannot compile a map-only imitation of original 0x205da.
    /// Go 標籤：`json:"loadch,omitempty"`
    #[serde(rename = "loadch", skip_serializing_if = "Option::is_none")]
    pub load_ch: Option<LoadCHState>,
    /// pan/walk 共用:目標格(walk 用 X/Y 當終點);pan 的 X/Y 沿用 Node.CamX/CamY 語意——
    /// 已由畫面回饋校準的「像素座標」,不是 doc47 §3 原始 grid(col,row)值(grid→px 未逐點驗證,
    /// 不自行換算,見 rulebook 62)。
    /// Go 標籤：`json:"x,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub x: i64,
    /// Go 標籤：`json:"y,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub y: i64,
    /// walk 起點;省略=沿用該角色目前座標(接續上一拍)
    /// Go 標籤：`json:"from_x,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub from_x: i64,
    /// Go 標籤：`json:"from_y,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub from_y: i64,
    /// walk/act:對應 Node.Actors 裡的角色(依 Fig 尋找,同 ActorWalk)
    /// Go 標籤：`json:"fig,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub fig: i64,
    /// original runtime unit-array slot; identity-critical handler primitives only
    /// Go 標籤：`json:"slot,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub slot: Option<i64>,
    /// pan/walk/fade 位移或漸變幀數;delay 用幀數(見 Ms)
    /// Go 標籤：`json:"frames,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub frames: i64,
    /// pan:0x135dd 每 tick 先 X 後 Y 移一個 tile
    /// Go 標籤：`json:"tile_step,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub tile_step: bool,
    /// walk:走位期間鏡頭鎖定跟隨(doc47 §9,同 Node.FollowWalk 機制)
    /// Go 標籤：`json:"follow,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub follow: bool,
    /// walk:走完後面向(指標,nil=保留走位末向;指定則面向它,如索爾走前面轉身面向亞雷斯)
    /// Go 標籤：`json:"dir,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub dir: Option<i64>,
    /// scroll_step:原版 0x13185 的重複上移格數
    /// Go 標籤：`json:"steps,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub steps: i64,
    /// palette_update: one-shot VGA DAC range update recovered from 0x11df2.
    /// Go 標籤：`json:"palette_start,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub palette_start: i64,
    /// Go 標籤：`json:"palette_end,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub palette_end: i64,
    /// Go 標籤：`json:"palette_delta,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub palette_delta: i64,
    /// transition_reveal: native 0x24b4d alternating-buffer present loop.
    /// Go 標籤：`json:"reveal_frames,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub reveal_frames: i64,
    /// Go 標籤：`json:"reveal_delay_ms,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub reveal_delay_ms: i64,
    /// act:Acting 非空時播放原版 acting frame 的行為轉錄：正常 frame 每 Beat 依 Pose 搬一格，
    /// special frame 只原地換姿態(doc50 §1.2)。Poses/PoseFrames 是舊的原地姿態近似欄位，
    /// 為舊場景相容而保留；不可與 Acting 混用。
    /// Go 標籤：`json:"acting_frames,omitempty"`
    #[serde(
        rename = "acting_frames",
        deserialize_with = "fd2_rules::battle::null_as_default",
        skip_serializing_if = "Vec::is_empty"
    )]
    pub acting: Vec<ActingFrame>,
    /// Go 標籤：`json:"poses,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub poses: Vec<i64>,
    /// 每個 pose 停留幀數(預設見 main.go)
    /// Go 標籤：`json:"pose_frames,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub pose_frames: i64,
    /// dialog:章文本第 Line 條起連續 Count 句(Count 省略=1)。Line 對應目前節點 Script+Scene
    /// 載入的那份 lines(同 Node.Scene 語意),不是 FDTXT 原始 idx(譯文精校版常把一條原文拆成
    /// 多句對白,見 doc47 §7 教訓:機制懂了但內容沒逐句對齊前不假裝一一對應)。
    /// Go 標籤：`json:"line,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub line: i64,
    /// Go 標籤：`json:"count,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub count: i64,
    /// handler compiler context; empty=Node.Script
    /// Go 標籤：`json:"script,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub script: String,
    /// handler compiler context; empty can mean unlabeled scene
    /// Go 標籤：`json:"scene,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub scene: String,
    /// authoritative for unlabeled/reused scene labels
    /// Go 標籤：`json:"scene_index,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub scene_index: Option<i64>,
    /// Upper:dialog 對話框上下位置覆蓋(指標,nil=沿用預設規則「說話者 id>=32 走上框」)。
    /// 草地撞見幕實測(doc55 截圖 18-03-10):亞雷斯(id4,<32)進場那句仍走上框——原版並非單純按
    /// id 分上下框,推測與進場/位置有關,尚未逆得通則;先開這個 per-beat 覆蓋做最小修正,別動全域規則。
    /// Go 標籤：`json:"upper,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub upper: Option<bool>,
    /// NativeDialogue 保存具 caller 證據的 FFFE／FFFD 頁面；未提供時，既有編輯場景
    /// 繼續使用正規化 renderer。
    /// Go 標籤：`json:"native_dialogue,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_dialogue: Option<NativeDialogueLayout>,
    /// NativeDialogues 對應同一分組 dialog beat 的逐句原生版面。陣列長度必須
    /// 嚴格等於有效 Count，且不可與單句 NativeDialogue 混用。
    /// Go 標籤：`json:"native_dialogues,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub native_dialogues: Vec<NativeDialogueLayout>,
    /// spawn:原版 FDFIELD 群組編號
    /// Go 標籤：`json:"group,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub group: i64,
    /// RawPlacementGate retains the exact per-call [0x53AFA] byte. Handler
    /// compiler output always supplies it; nil is reserved for authored legacy
    /// campaigns which intentionally use the normalized compatibility append.
    /// Go 標籤：`json:"raw_placement_gate,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub raw_placement_gate: Option<i64>,
    /// CharID is JOIN's permanent-player identity.  It intentionally remains
    /// separate from a scene actor's Fig/portrait: JOIN accepts only the
    /// original 0..31 player roster, while a cutscene may contain arbitrary
    /// NPC portraits (for example map31's shop-clerk portrait 75).
    /// Go 標籤：`json:"char_id,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub char_id: i64,
    /// bgm:曲目 id(對映 assets/bgm)
    /// Go 標籤：`json:"track,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub track: String,
    /// fade:true=淡出 false=淡入(重用 storyFade,doc46 §5.2)
    /// Go 標籤：`json:"out,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub out: bool,
    /// delay:毫秒(原版 0x375b2 語意);換算成 60fps 幀數,Frames 優先
    /// Go 標籤：`json:"ms,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub ms: i64,
    /// set_chapter: original [0x53c03] campaign/resource chapter assignment.
    /// Pointer form preserves chapter zero as an explicit editable value.
    /// Go 標籤：`json:"chapter,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub chapter: Option<i64>,
    /// grant_item: original unsigned-byte item identity.
    /// Go 標籤：`json:"item_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub item_id: Option<i64>,
    /// Go 標籤：`json:"resource_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub resource_id: Option<i64>,
    /// ResourceArchive/Owner preserve 0x111BA's filename and old/new handle
    /// owner when a bare numeric archive index would be ambiguous.
    /// Go 標籤：`json:"resource_archive,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub resource_archive: String,
    /// Go 標籤：`json:"resource_owner,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub resource_owner: String,
    /// Go 標籤：`json:"sfx_index,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub sfx_index: Option<i64>,
}

/// NativeEndingPrefixConfig 是已證實原版結局前綴的狹義戰役入口。它保留精確
/// timeline／handler，不會把 ending decoder 變成泛用終局場景。來源約束 E1
/// 模式允許正常戰役播放已驗證資源與近似時鐘，但不宣稱 DOS 逐像素／逐音訊 E2。
///
/// Go：`campaign.go:512`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeEndingPrefixConfig {
    /// Go 標籤：`json:"timeline"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub timeline: String,
    /// Go 標籤：`json:"handler"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub handler: String,
    /// Go 標籤：`json:"chapter"`
    pub chapter: i64,
    /// Go 標籤：`json:"mode"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub mode: String,
}

/// NativeDefeatReturnTitleConfig 僅接已閉合的 pending=1 外層返回契約。
/// 原始 bytes／caller／consumer 見 fd2_pending_code1_return_title_20261001.json。
///
/// Go：`campaign.go:521`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeDefeatReturnTitleConfig {
    /// Go 標籤：`json:"handler"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub handler: String,
    /// Go 標籤：`json:"resource"`
    pub resource: i64,
    /// Go 標籤：`json:"return"`
    #[serde(rename = "return", deserialize_with = "fd2_rules::battle::null_as_default")]
    pub r#return: String,
}

impl NativeDefeatReturnTitleConfig {
    /// Go：`campaign.go:527`，`func (c *NativeDefeatReturnTitleConfig) IsRecoveredContract() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn is_recovered_contract(&self) -> bool {
        self.handler == "0x22e5c" && self.resource == 79 && self.r#return == "full_title_sequence"
    }
}

pub const NATIVE_ENDING_PREFIX_SOURCE_BOUND_E1: &str = "source_bound_e1_terminal_hold";

impl NativeEndingPrefixConfig {
    /// IsRecoveredPrefixContract 即使呼叫端在記憶體內建構 Campaign、略過 JSON
    /// 載入驗證，仍將唯一支援的結局前綴限制在已證實合約內。
    ///
    /// Go：`campaign.go:535`，`func (c *NativeEndingPrefixConfig) IsRecoveredPrefixContract() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn is_recovered_prefix_contract(&self) -> bool {
        self.timeline == "assets/endings/native_2bce5.json"
            && self.handler == "0x2bce5"
            && (self.chapter == 26 || self.chapter == 29 || self.chapter == 30)
            && self.mode == NATIVE_ENDING_PREFIX_SOURCE_BOUND_E1
    }
}

/// Node 節點。Type: story / cutscene / battle / town / preparation / church / choice /
/// inventory_gate / inventory_recipe / event / shop / ending。
/// cutscene(doc 50):story 的 beats 驅動版——用 Beats 承接已有證據的原版章 handler 原語序列，
/// 對白與走位/演出天然交錯(平面序列,非「一幕一段」)。Map/Actors/BGM/ExitWalk(s) 等欄位與
/// story 共用同一套場景設置(進節點時的初始擺位、退場走位、淡出轉場),Beats 只負責節點「進行中」
/// 的编排;story 節點型別保留相容,兩者可並存於同一 campaign(逐步遷移,doc50 §2)。
///
/// Go：`campaign.go:547`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Node {
    /// Go 標籤：`json:"type"`
    #[serde(rename = "type", deserialize_with = "fd2_rules::battle::null_as_default")]
    pub r#type: String,
    /// story:對白(內嵌;Script 有檔時被覆蓋)
    /// Go 標籤：`json:"lines,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub lines: Vec<Line>,
    /// story:本機劇情文本檔(assets/story/chNN.json,不入庫)
    /// Go 標籤：`json:"script,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub script: String,
    /// battle:戰場事件劇本檔
    /// Go 標籤：`json:"scenario,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub scenario: String,
    /// battle:戰場資產目錄;story:場景背景圖(doc23 §4:
    /// 原版序幕王城/草地背景是 FDFIELD map32 複合場景,與戰場同一渲染器非另開圖片系統;
    /// story 填同一 assets/maps/mapN 目錄即可換場景背景;battle 空=沿用當前)
    /// Go 標籤：`json:"map,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub map: String,
    /// battle:單位配置檔
    /// Go 標籤：`json:"units,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub units: String,
    /// story+Map:固定鏡頭像素座標(場景不跟游標走,取代預設 focusOnParty)
    /// Go 標籤：`json:"cam_x,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub cam_x: i64,
    /// Go 標籤：`json:"cam_y,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub cam_y: i64,
    /// Go 標籤：`json:"native_map_view,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_map_view: Option<NativeMapViewConfig>,
    /// Go 標籤：`json:"native_map_hud,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_map_hud: Option<NativeMapHUDConfig>,
    /// Go 標籤：`json:"native_map_hud_inherited,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_map_hud_inherited: Option<NativeMapHUDInheritedConfig>,
    /// story+Map:場景背景上的靜態角色擺位
    /// Go 標籤：`json:"actors,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub actors: Vec<Actor>,
    /// story+Script:只取 Script 檔裡 label 對映的那個 scene(doc46 §5.2;
    /// 空=舊行為,整份 Script 攤平全部 scenes 成一條對白隊列——別讓一個節點播完整份劇本)
    /// Go 標籤：`json:"scene,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub scene: String,
    /// story:對白播完、換場前先走一段路再淡出(doc46 §5.3;單一角色)
    /// Go 標籤：`json:"exit_walk,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub exit_walk: Option<ActorWalk>,
    /// 多角色同時退場(全部走完才轉場)。
    /// ⚠ 更正(doc55 逐幀量測 2026-07-05):早前「草地幕兩人一起走離」是錯的——實測=亞雷斯對話中先走近、
    /// 索爾對話後才單獨走到亞雷斯旁、隨即淡出(無一起走離畫面)。草地幕已改「亞雷斯進場走位+索爾單人 ExitWalk」;
    /// 本欄保留供其他真有「多人同時退場」的幕使用;並行多角色不在 Beat.walk 單角色設計內,收尾沿用本欄不重造輪子)
    /// Go 標籤：`json:"exit_walks,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub exit_walks: Vec<ActorWalk>,
    /// cutscene:過場原語序列(doc 50);Beats 跑完後走 ExitWalk(s)+淡出+Advance,同 story 節點收尾
    /// Go 標籤：`json:"beats,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub beats: Vec<Beat>,
    /// cutscene:editable handler binding; runtime must reject unresolved compile issues
    /// Go 標籤：`json:"handler_binding,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub handler_binding: String,
    /// story:無對白/Script 時,進節點後幾幀自動轉場(doc46 行軍蒙太奇)
    /// Go 標籤：`json:"auto_advance,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub auto_advance: i64,
    /// story:進場走位全走完才顯示對白(2-1:王座廳索爾沿紅毯走到王座前對話框才出現)
    /// Go 標籤：`json:"walk_first,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub walk_first: bool,
    /// story:走位期間鏡頭鎖定跟隨走位者(原版 13×8 格視野長廊運鏡,doc25 0x11eee)
    /// Go 標籤：`json:"follow_walk,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub follow_walk: bool,
    /// story:鏡頭 Y 上限(px;0=不限)。王座廳=808 擋住 map32 底部草地段
    /// Go 標籤：`json:"cam_max_y,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub cam_max_y: i64,
    /// (原版第一幕畫面無草地,索爾從畫面外沿紅毯走入,使用者回饋 2026-07-04 #1)
    /// Go 標籤：`json:"bgm,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub bgm: String,
    /// story/event
    /// Go 標籤：`json:"next,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub next: String,
    /// hotel：服務 0（0x2FFA5 打聽消息）要進的 story 節點
    /// Go 標籤：`json:"rumor,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub rumor: String,
    /// battle
    /// Go 標籤：`json:"on_win,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub on_win: String,
    /// battle(敗北路線;空=game over)
    /// Go 標籤：`json:"on_lose,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub on_lose: String,
    /// Go 標籤：`json:"native_defeat_return_title,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_defeat_return_title: Option<NativeDefeatReturnTitleConfig>,
    /// Proven 0x321C8 calls after required checks and before final confirmation.
    /// Go 標籤：`json:"preparation_front_identities,omitempty"`
    #[serde(
        rename = "preparation_front_identities",
        deserialize_with = "fd2_rules::battle::null_as_default",
        skip_serializing_if = "Vec::is_empty"
    )]
    pub party_front_identities: Vec<i64>,
    /// EndingPartySnapshotOnWin 是重製終局資料邊界；只允許勝利直接進 ending
    /// 的 battle。它保存最後隊伍供回顧，不冒稱原版 FD2.SAV ABI。
    /// Go 標籤：`json:"ending_party_snapshot_on_win,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub ending_party_snapshot_on_win: bool,
    /// battle:保護目標；空值沿用主角索爾
    /// Go 標籤：`json:"protect,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub protect: String,
    /// inventory_gate:原版 unsigned-byte item identity
    /// Go 標籤：`json:"item_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub item_id: Option<i64>,
    /// inventory_gate:全隊任一角色持有 ItemID
    /// Go 標籤：`json:"if_present,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub if_present: String,
    /// inventory_gate:全隊皆未持有 ItemID
    /// Go 標籤：`json:"if_missing,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub if_missing: String,
    /// inventory_recipe:逐 item×runtime slot 計數／移除
    /// Go 標籤：`json:"item_ids,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub item_ids: Vec<i64>,
    /// inventory_recipe:只掃前 N 個 runtime records
    /// Go 標籤：`json:"slot_count,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub slot_count: i64,
    /// inventory_recipe:原版要求的精確命中組合數
    /// Go 標籤：`json:"required_matches,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub required_matches: i64,
    /// inventory_recipe:成功後 grant 的 item
    /// Go 標籤：`json:"reward_item_id,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub reward_item_id: Option<i64>,
    /// inventory_recipe:成功 arm
    /// Go 標籤：`json:"if_crafted,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub if_crafted: String,
    /// inventory_recipe:命中數不符 arm
    /// Go 標籤：`json:"if_insufficient,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub if_insufficient: String,
    /// choice/preparation
    /// Go 標籤：`json:"prompt,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub prompt: String,
    /// preparation: original 0x318ad selection cap (15, late route 19)
    /// Go 標籤：`json:"party_limit,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub party_limit: i64,
    /// preparation: selected records' native +8, checked before final confirmation
    /// Go 標籤：`json:"required_party_identities,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub required_party_identities: Vec<i64>,
    /// preparation: native confirmation/selection cancellation target
    /// Go 標籤：`json:"cancel,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub cancel: String,
    /// town:原版戰後城鎮/營地名稱(可編輯、可存檔的整備 hub)
    /// Go 標籤：`json:"town,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub town: String,
    /// town:0/1/2→FDOTHER#11/#61/#62
    /// Go 標籤：`json:"native_town_variant,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_town_variant: Option<i64>,
    /// town:selection+BIOS scan→reveal selection5
    /// Go 標籤：`json:"native_secret_gate,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_secret_gate: Option<NativeTownSecretGate>,
    /// choice
    /// Go 標籤：`json:"options,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub options: Vec<ChoiceOption>,
    /// Go 標籤：`json:"set_flags,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "BTreeMap::is_empty")]
    pub set_flags: BTreeMap<String, bool>,
    /// ending:結語
    /// Go 標籤：`json:"text,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub text: String,
    /// Go 標籤：`json:"native_ending_prefix,omitempty"`
    #[serde(skip_serializing_if = "Option::is_none")]
    pub native_ending_prefix: Option<NativeEndingPrefixConfig>,
    /// shop:商品
    /// Go 標籤：`json:"goods,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub goods: Vec<Good>,
    /// shop:0=custom/generic; original owner uses 1 weapon, 3 item, 5 secret
    /// Go 標籤：`json:"native_hub_variant,omitempty"`
    #[serde(skip_serializing_if = "fd2_rules::battle::go_is_zero")]
    pub native_hub_variant: i64,
    /// shop:legacy authored conditional goods
    /// Go 標籤：`json:"secret,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "Vec::is_empty")]
    pub secret: Vec<Good>,
    /// shop:legacy authored flag; not the proven native hidden-entry gate
    /// Go 標籤：`json:"secret_if,omitempty"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default", skip_serializing_if = "String::is_empty")]
    pub secret_if: String,
}

/// Campaign 整張節點圖。
///
/// Go：`campaign.go:616`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct Campaign {
    /// Go 標籤：`json:"title"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub title: String,
    /// Go 標籤：`json:"start"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub start: String,
    /// Go 標籤：`json:"flags"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub flags: BTreeMap<String, bool>,
    /// Go 標籤：`json:"nodes"`
    #[serde(deserialize_with = "fd2_rules::battle::null_as_default")]
    pub nodes: BTreeMap<String, Node>,
}

/// ExpandNativeDialogueGroups 將可編輯的分組對話投影成執行期的一句一拍。
/// 原生 opening／mouth／closing player 只接受單句，故這個展開必須在進入
/// cutscene 時完成；原始 Node.Beats 保持分組形狀，方便審查 FDTXT tuple。
///
/// Go：`campaign.go:626`，`func ExpandNativeDialogueGroups(beats []Beat) ([]Beat, error)`
pub fn expand_native_dialogue_groups(beats: &[Beat]) -> Result<Vec<Beat>> {
    let mut result = Vec::with_capacity(beats.len());
    for (index, source) in beats.iter().enumerate() {
        let mut beat = source.clone();
        beat.then = expand_native_dialogue_groups(&source.then)?;
        beat.r#else = expand_native_dialogue_groups(&source.r#else)?;
        if beat.op != "dialog" || beat.native_dialogues.is_empty() {
            result.push(beat);
            continue;
        }
        let mut count = beat.count;
        if count <= 0 {
            count = 1;
        }
        if beat.native_dialogue.is_some() || beat.native_dialogues.len() as i64 != count {
            return Err(campaign_err!("dialog beat {index} 的原生版面群組無效"));
        }
        // Go `campaign.go:651` 另檢查 `layout == nil`（「原生版面 %d 為空」）；Rust 的
        // `native_dialogues` 元素不會是 nil（JSON `null` 元素在解碼時就被拒收），該分支不移植。
        for (utterance, layout) in beat.native_dialogues.iter().enumerate() {
            let mut single = beat.clone();
            single.line = beat.line.wrapping_add(utterance as i64);
            single.count = 1;
            single.native_dialogue = Some(layout.clone());
            single.native_dialogues = Vec::new();
            let upper = layout.control == "FFED" || layout.control == "FFEF";
            if let Some(want) = beat.upper
                && want != upper
            {
                return Err(campaign_err!(
                    "dialog beat {index} 的 upper 與原生版面 {utterance} control={} 不一致",
                    layout.control
                ));
            }
            single.upper = Some(upper);
            result.push(single);
        }
    }
    Ok(result)
}

/// Load 讀 campaign.json 並驗證轉場目標都存在。
///
/// Go：`campaign.go:672`，`func Load(path string) (*Campaign, error)`
pub fn load(path: &str) -> Result<Campaign> {
    let raw = battle::read_file(path)?;
    decode(&raw)
}

/// Decode 走與 Load 相同的驗證，但來源是位元組。編輯器的 canonical 文件由
/// compiler 編成同一份 JSON 之後，要能走完全一樣的那道關卡——分開兩條驗證會讓
/// 「編輯器存得起來但遊戲讀不動」這種差異藏在兩份程式碼的縫裡。
///
/// Go：`campaign.go:683`，`func Decode(raw []byte) (*Campaign, error)`
pub fn decode(raw: &[u8]) -> Result<Campaign> {
    let c: Campaign = battle::decode_json(raw)?;
    if !c.nodes.contains_key(&c.start) {
        return Err(campaign_err!("start 節點 {} 不存在", battle::go_quote(&c.start)));
    }
    let check = |from: &str, to: &str| -> Result<()> {
        if to.is_empty() {
            return Ok(());
        }
        if !c.nodes.contains_key(to) {
            return Err(campaign_err!("節點 {} 的轉場目標 {} 不存在", battle::go_quote(from), battle::go_quote(to)));
        }
        Ok(())
    };
    fn validate_beats(node_id: &str, beats: &[Beat]) -> Result<()> {
        for (i, beat) in beats.iter().enumerate() {
            if beat.op == "dialog" {
                let mut count = beat.count;
                if count <= 0 {
                    count = 1;
                }
                let q = battle::go_quote(node_id);
                if beat.native_dialogue.is_some() && !beat.native_dialogues.is_empty() {
                    return Err(campaign_err!("節點 {q} beat {i} 不可混用 native_dialogue 與 native_dialogues"));
                }
                if !beat.native_dialogues.is_empty() {
                    if beat.native_dialogues.len() as i64 != count {
                        return Err(campaign_err!(
                            "節點 {q} beat {i} 的 native_dialogues={}，必須等於 count={count}",
                            beat.native_dialogues.len()
                        ));
                    }
                    // Go `campaign.go:715` 的 `layout == nil` 分支見 `expand_native_dialogue_groups`。
                    for (j, layout) in beat.native_dialogues.iter().enumerate() {
                        if let Err(err) = layout.validate() {
                            return Err(err.wrap(format!("節點 {q} beat {i} 的 native_dialogues[{j}] 無效：")));
                        }
                    }
                }
            }
            validate_beats(node_id, &beat.then)?;
            validate_beats(node_id, &beat.r#else)?;
        }
        Ok(())
    }
    // Go 以 map 迭代順序檢查節點（`campaign.go:735`）；只影響多個節點同時不合法時先報哪一個。
    for (id, n) in &c.nodes {
        let q = battle::go_quote(id);
        if let Some(defeat) = &n.native_defeat_return_title
            && (n.r#type != "battle" || !n.on_lose.is_empty() || !defeat.is_recovered_contract())
        {
            return Err(campaign_err!("節點 {q} 的原生敗北返回標題契約無效"));
        }
        validate_beats(id, &n.beats)?;
        for to in
            [&n.next, &n.cancel, &n.on_win, &n.on_lose, &n.if_present, &n.if_missing, &n.if_crafted, &n.if_insufficient]
        {
            check(id, to)?;
        }
        if n.r#type == "inventory_gate" {
            if !n.item_id.is_some_and(|v| (0..=255).contains(&v)) {
                return Err(campaign_err!("inventory_gate 節點 {q} 的 item_id 必須是 0..255"));
            }
            if n.if_present.is_empty() || n.if_missing.is_empty() {
                return Err(campaign_err!("inventory_gate 節點 {q} 必須同時定義 if_present / if_missing"));
            }
        }
        if n.r#type == "inventory_recipe" {
            if n.item_ids.is_empty() || n.slot_count <= 0 || n.required_matches <= 0 {
                return Err(campaign_err!(
                    "inventory_recipe 節點 {q} 必須定義 item_ids / slot_count / required_matches"
                ));
            }
            for &item_id in &n.item_ids {
                if !(0..=255).contains(&item_id) {
                    return Err(campaign_err!("inventory_recipe 節點 {q} 的 item_ids 必須是 0..255"));
                }
            }
            if !n.reward_item_id.is_some_and(|v| (0..=255).contains(&v)) {
                return Err(campaign_err!("inventory_recipe 節點 {q} 的 reward_item_id 必須是 0..255"));
            }
            if n.if_crafted.is_empty() || n.if_insufficient.is_empty() {
                return Err(campaign_err!("inventory_recipe 節點 {q} 必須同時定義 if_crafted / if_insufficient"));
            }
        }
        if !n.required_party_identities.is_empty() {
            let mut seen = BTreeSet::new();
            for &identity in &n.required_party_identities {
                if n.r#type != "preparation" || !(1..=31).contains(&identity) || seen.contains(&identity) {
                    return Err(campaign_err!("preparation node {q} has invalid required_party_identities"));
                }
                seen.insert(identity);
            }
        }
        if !n.party_front_identities.is_empty() {
            let mut seen = BTreeSet::new();
            for &identity in &n.party_front_identities {
                let required = n.required_party_identities.contains(&identity);
                if n.r#type != "preparation" || !required || !(1..=31).contains(&identity) || seen.contains(&identity) {
                    return Err(campaign_err!("preparation node {q} has invalid preparation_front_identities"));
                }
                seen.insert(identity);
            }
        }
        if n.r#type == "shop"
            && n.native_hub_variant != 0
            && n.native_hub_variant != 1
            && n.native_hub_variant != 3
            && n.native_hub_variant != 5
        {
            return Err(campaign_err!("shop 節點 {q} 的 native_hub_variant 必須是 1、3 或 5"));
        }
        if let Some(gate) = &n.native_secret_gate
            && (n.r#type != "town"
                || gate.selection < 0
                || gate.selection > 4
                || gate.scan_code < 0x54
                || gate.scan_code > 0x71
                || gate.to.is_empty()
                || n.native_town_variant.is_none()
                || check(id, &gate.to).is_err())
        {
            return Err(campaign_err!("town 節點 {q} 的 native_secret_gate 無效"));
        }
        if let Some(variant) = n.native_town_variant
            && (n.r#type != "town" || !(0..=2).contains(&variant))
        {
            return Err(campaign_err!("town 節點 {q} 的 native_town_variant 無效"));
        }
        if let Some(prefix) = &n.native_ending_prefix
            && (n.r#type != "ending" || !prefix.is_recovered_prefix_contract())
        {
            return Err(campaign_err!(
                "ending 節點 {q} 的 native_ending_prefix 不是已證實的 0x2bce5 recovered-only 合約"
            ));
        }
        if n.ending_party_snapshot_on_win {
            let ending = c.nodes.get(&n.on_win);
            if n.r#type != "battle" || n.on_win.is_empty() || !ending.is_some_and(|e| e.r#type == "ending") {
                return Err(campaign_err!(
                    "battle 節點 {q} 的 ending_party_snapshot_on_win 只能用於直接進入 ending 的勝利邊"
                ));
            }
        }
        if n.native_map_hud.is_some() && n.native_map_view.is_none() {
            return Err(campaign_err!("battle 節點 {q} 不可在缺少 native_map_view 時單獨定義 native_map_hud"));
        }
        if let Some(inherited) = &n.native_map_hud_inherited {
            if n.native_map_view.is_none() {
                return Err(campaign_err!("battle 節點 {q} 不可在缺少 native_map_view 時繼承 native HUD"));
            }
            if n.native_map_hud.is_some() {
                return Err(campaign_err!("battle 節點 {q} 不可同時定義固定與繼承 native HUD"));
            }
            if inherited.display_gate_b != 1 {
                return Err(campaign_err!(
                    "battle 節點 {q} 的 native_map_hud_inherited.display_gate_b 目前只允許已證實的 controller entry 1"
                ));
            }
        }
        if let Some(view) = &n.native_map_view {
            // Campaign JSON currently owns two directly proven entry states:
            // 0 is the no-overlay state retained by ch26_pre on return, while 1
            // is the CONTINUE/post-bootstrap state at 0x1060c. Runtime command
            // and item writers own every other selector transition.
            if !matches!(view.range_mode, Some(0) | Some(1)) {
                return Err(campaign_err!(
                    "battle 節點 {q} 的 native_map_view.range_mode 目前只允許已證實的 entry selector 0/1"
                ));
            }
        }
        if let Some(hud) = &n.native_map_hud
            && (!(0..=0xff).contains(&hud.display_gate_a)
                || !(0..=0xff).contains(&hud.display_gate_b)
                || !(0..=0xfb).contains(&hud.anchor_x))
        {
            return Err(campaign_err!("battle 節點 {q} 的 native_map_hud 超出原版 raw 範圍"));
        }
        for o in &n.options {
            check(id, &o.to)?;
        }
    }
    Ok(c)
}

/// Runner 執行狀態:目前節點 + 旗標。
///
/// Go：`campaign.go:881`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct Runner {
    pub c: Option<Campaign>,
    pub cur: String,
    pub flags: BTreeMap<String, bool>,
}

/// NewRunner 從起點開跑(複製初始旗標)。
///
/// Go：`campaign.go:888`，`func NewRunner(c *Campaign) *Runner`
pub fn new_runner(c: &Campaign) -> Runner {
    Runner { c: Some(c.clone()), cur: c.start.clone(), flags: c.flags.clone() }
}

impl Runner {
    /// Node 目前節點。
    ///
    /// Go：`campaign.go:897`，`func (r *Runner) Node() *Node`
    /// Rust：Go 回傳 `r.C.Nodes[r.Cur]`，鍵不存在時是 nil；Rust 回傳借用。
    pub fn node(&self) -> Option<&Node> {
        self.c.as_ref().and_then(|c| c.nodes.get(&self.cur))
    }

    /// NodeID exposes the editable node key for runtime data fallback decisions.
    /// Keeping this separate from Node avoids making callers infer identity from
    /// the node payload (which is intentionally allowed to be identical across
    /// multiple story segments).
    ///
    /// Go：`campaign.go:903`，`func (r *Runner) NodeID() string`
    pub fn node_id(&self) -> String {
        self.cur.clone()
    }

    /// Visible 回傳 choice 節點依旗標過濾後的選項。
    ///
    /// Go：`campaign.go:906`，`func (r *Runner) Visible() []Option`
    pub fn visible(&self) -> Vec<ChoiceOption> {
        // Go `campaign.go:907` 對 nil 節點取 `n.Options` 會 panic；Rust 同樣 panic。
        let n = self.node().expect("campaign: Runner.Visible 目前節點不存在");
        n.options
            .iter()
            .filter(|o| o.r#if.is_empty() || self.flags.get(&o.r#if).copied().unwrap_or(false))
            .cloned()
            .collect()
    }

    /// ShopGoods returns editable goods and the legacy authored conditional list.
    /// SecretIf is a remake schema feature; it does not prove the native hidden
    /// selection-5 keyboard/modifier gate.
    ///
    /// Go：`campaign.go:920`，`func (r *Runner) ShopGoods() []Good`
    pub fn shop_goods(&self) -> Vec<Good> {
        // Go `campaign.go:921` 對 nil 節點取 `n.Goods` 會 panic；Rust 同樣 panic。
        let n = self.node().expect("campaign: Runner.ShopGoods 目前節點不存在");
        let mut out = n.goods.clone();
        if !n.secret_if.is_empty() && self.flags.get(&n.secret_if).copied().unwrap_or(false) {
            out.extend(n.secret.iter().cloned());
        }
        out
    }

    /// MatchNativeTownSecret reports whether the current raw selection and BIOS
    /// scan reveal native selection 5. 0x2cde0..0x2cef7 does not dispatch here:
    /// the town owner redraws selection 5 and waits for a later confirmation.
    ///
    /// Go：`campaign.go:932`，`func (r *Runner) MatchNativeTownSecret(selection, scanCode int) bool`
    pub fn match_native_town_secret(&self, selection: i64, scan_code: i64) -> bool {
        let Some(n) = self.node() else {
            return false;
        };
        if n.r#type != "town" {
            return false;
        }
        let Some(gate) = &n.native_secret_gate else {
            return false;
        };
        if selection != gate.selection
            || scan_code != gate.scan_code
            || !self.c.as_ref().is_some_and(|c| c.nodes.contains_key(&gate.to))
        {
            return false;
        }
        true
    }

    /// ConfirmNativeTownSecret dispatches the already revealed native selection 5.
    /// It is deliberately separate from MatchNativeTownSecret so the original
    /// reveal/redraw/confirm lifecycle cannot collapse into a one-key transition.
    ///
    /// Go：`campaign.go:948`，`func (r *Runner) ConfirmNativeTownSecret(selection int) bool`
    pub fn confirm_native_town_secret(&mut self, selection: i64) -> bool {
        let Some(n) = self.node() else {
            return false;
        };
        let Some(gate) = &n.native_secret_gate else {
            return false;
        };
        if n.r#type != "town" || selection != 5 || !self.c.as_ref().is_some_and(|c| c.nodes.contains_key(&gate.to)) {
            return false;
        }
        self.cur = gate.to.clone();
        true
    }

    /// Advance 依結果離開目前節點:套用 set_flags,回傳下一節點 id(""=流程結束/game over)。
    /// outcome:story/event→忽略;battle→"win"/"lose";choice→"optN"(過濾後 index)。
    ///
    /// Go：`campaign.go:961`，`func (r *Runner) Advance(outcome string) string`
    pub fn advance(&mut self, outcome: &str) -> String {
        let Some(n) = self.node().cloned() else {
            return String::new();
        };
        for (k, v) in &n.set_flags {
            self.flags.insert(k.clone(), *v);
        }
        let next = match n.r#type.as_str() {
            "battle" => {
                if outcome == "win" {
                    n.on_win.clone()
                } else {
                    n.on_lose.clone()
                }
            }
            "choice" | "town" => {
                let mut next = String::new();
                if let Some(i) = go_sscanf_opt(outcome) {
                    let vis = self.visible();
                    if i >= 0 && (i as u64) < vis.len() as u64 {
                        next = vis[i as usize].to.clone();
                    }
                }
                next
            }
            "inventory_gate" => match outcome {
                "present" => n.if_present.clone(),
                "missing" => n.if_missing.clone(),
                _ => String::new(),
            },
            "inventory_recipe" => match outcome {
                "crafted" => n.if_crafted.clone(),
                "insufficient" => n.if_insufficient.clone(),
                _ => String::new(),
            },
            "preparation" => {
                if outcome == "cancel" {
                    n.cancel.clone()
                } else {
                    n.next.clone()
                }
            }
            // 0x2FC85 的四格：0 傳聞、1 存檔、2 讀檔、3 離開。傳聞是另一個 story 節點；
            // 其餘都回城鎮（原版讀檔／存檔完留在酒店選單，重製端由酒店 UI 自己接）。
            "hotel" => {
                if outcome == "rumor" && !n.rumor.is_empty() {
                    n.rumor.clone()
                } else {
                    n.next.clone()
                }
            }
            "ending" => String::new(),
            // story / event / shop
            _ => n.next.clone(),
        };
        self.cur = next.clone();
        next
    }
}

/// Go `fmt.Sscanf(outcome, "opt%d", &i)`（`campaign.go:979`）成功時的 `i`。
///
/// 照 Go `fmt` 掃描 `%d` 的實際接受範圍手寫（不用 `str::parse` 整串剖析）：字面 `opt` 必須完全
/// 相符；`%d` 先略過空白（換行即失敗，`\r\n` 視同換行），接受一個正負號與至少一位十進位數字，
/// 以 `strconv.ParseInt(tok, 10, 64)` 解析（溢位即失敗）。數字之後的其他字元不影響成功。
pub(crate) fn go_sscanf_opt(outcome: &str) -> Option<i64> {
    let rest = outcome.strip_prefix("opt")?;
    let mut start = rest.len();
    for (at, c) in rest.char_indices() {
        if c == '\n' || (c == '\r' && rest[at + 1..].starts_with('\n')) {
            return None;
        }
        if !c.is_whitespace() {
            start = at;
            break;
        }
    }
    let body = &rest.as_bytes()[start..];
    let mut end = 0;
    if end < body.len() && (body[end] == b'+' || body[end] == b'-') {
        end += 1;
    }
    let digits_from = end;
    while end < body.len() && body[end].is_ascii_digit() {
        end += 1;
    }
    if end == digits_from {
        return None;
    }
    rest[start..start + end].parse::<i64>().ok()
}

// 測試：Go `remake/internal/campaign/campaign_test.go` 逐條移植。
#[cfg(test)]
mod campaign_test;

// 測試：Go `remake/internal/campaign/story_asset_test.go` 逐條移植。
#[cfg(test)]
mod story_asset_test;
