//! Go 來源：`remake/internal/battle/snapshot.go`（448 行）。
//!
//! snapshot.go — 戰場 JSON 快照（重製端自有存檔擴充，56「2026-11-13 #303 戰場JSON快照」）。
//!
//! 快照不是原版 FD2.SAV ABI。State 每個欄位都必須歸入 stateSnapshotFieldClass 的
//! 三類之一，snapshot_test 以反射列舉欄位；新增欄位沒分類就失敗，避免新狀態漏存後
//! 讀檔靜默偏差。

use super::*;

/// StateSnapshotVersion 只在快照欄位語意改變時遞增；讀到不同版本一律拒絕。
pub const STATE_SNAPSHOT_VERSION: i64 = 1;

/// 存入快照
pub(crate) const SNAPSHOT_SAVED: &str = "saved";

/// 讀檔時由應用層從資產與劇本重新綁定
pub(crate) const SNAPSHOT_BOUND: &str = "bound";

/// 受理存檔時必須是零值
pub(crate) const SNAPSHOT_TRANSIENT: &str = "transient";

/// Go：`snapshot.go:26`
pub(crate) static STATE_SNAPSHOT_FIELD_CLASS: LazyLock<BTreeMap<&str, &str>> = LazyLock::new(|| {
    BTreeMap::from([
        ("Balance", "bound"),
        ("W", "saved"),
        ("H", "saved"),
        ("Units", "saved"),
        ("NativeMapSelectorCache", "saved"),
        ("NativeMapSelectorError", "saved"),
        ("NativeMapCycleState", "saved"),
        ("HasNativeMapCycleState", "saved"),
        ("NativeTerrainPhaseState", "saved"),
        ("HasNativeTerrainPhaseState", "saved"),
        ("NativeTerrainFlipState", "saved"),
        ("NativeUnitPixelShiftState", "saved"),
        ("HasNativeMapBinaryTimingState", "saved"),
        ("NativeMapHUDState", "saved"),
        ("HasNativeMapHUDState", "saved"),
        ("NativeMapViewState", "saved"),
        ("HasNativeMapViewState", "saved"),
        ("NativeMapRangeMode", "saved"),
        ("HasNativeMapRangeModeState", "saved"),
        ("Roster", "saved"),
        ("PendingGroups", "saved"),
        ("nativeFutureItemRows", "bound"),
        ("nativeAIScan", "transient"),
        ("OwnDeploy", "saved"),
        ("Turn", "saved"),
        ("NativeRoundCounter", "saved"),
        ("Flags", "saved"),
        ("NativeEventState", "saved"),
        ("Cost", "saved"),
        ("NativeCompositionEventBytes", "saved"),
        ("NativeMapEventGrid", "saved"),
        ("HasNativeMapEventGrid", "saved"),
        ("NativeFieldControlRaw", "saved"),
        ("NativeTurnEventControls", "saved"),
        ("HasNativeTurnEventControlState", "saved"),
        ("NativeChestControls", "saved"),
        ("HasNativeChestControlState", "saved"),
        ("NativeFieldUnitControls", "saved"),
        ("HasNativeFieldControlState", "saved"),
        ("NativeRuntimeRecords", "saved"),
        ("HasNativeRuntimeUnitProjection", "saved"),
        ("HasNativePendingGroupBinding", "saved"),
        ("NativeFieldEventSlots", "saved"),
        ("NativeFieldEvents", "saved"),
        ("NativeFieldEventRules", "bound"),
        ("NativeTileBlitModes", "saved"),
        ("NativeTerrainControl", "saved"),
        ("NativeTerrainMoveCodes", "saved"),
        ("nativeMovementCostRows", "bound"),
        ("SpellBook", "saved"),
        ("NativeCommandBook", "bound"),
        ("NativeCommandResistances", "bound"),
        ("CommandLearn", "bound"),
        ("CommandLearnSelectors", "bound"),
        ("NativeGrowthRows", "bound"),
        ("NativeGrowthRollObserver", "bound"),
        ("NativeDeathExpCancel", "bound"),
        ("AICommandSpell", "saved"),
        ("Treasures", "saved"),
        ("NativeTreasureEventRules", "bound"),
        ("NativeExperienceAccumulator", "transient"),
        ("OpenedTreasure", "saved"),
    ])
});

/// SnapshotTreasure 是 Treasures 的 JSON 形式；map 的鍵 Cell 不是字串。
///
/// Go：`snapshot.go:92`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct SnapshotTreasure {
    /// Go 標籤：`json:"cell"`
    pub cell: Cell,
    /// Go 標籤：`json:"treasure"`
    pub treasure: Treasure,
}

/// StateSnapshot 是 State 的 saved 類欄位。指標陣列的 aliasing 不保存：Units 與
/// Roster 是兩個獨立陣列，快照前先驗證沒有共用同一個 *Unit。
///
/// Go：`snapshot.go:99`
#[derive(Debug, Clone, Default, PartialEq)]
pub struct StateSnapshot {
    /// Go 標籤：`json:"version"`
    pub version: i64,
    /// Go 標籤：`json:"w"`
    pub w: i64,
    /// Go 標籤：`json:"h"`
    pub h: i64,
    /// Go 標籤：`json:"units"`
    /// 所有權：本欄擁有單位本體；其他地方以 `UnitIdx` 指向這個陣列（見 `battle` 模組說明）。
    pub units: Vec<Unit>,
    /// UnitsConstructorPending／RosterConstructorPending 保存未匯出的
    /// nativeConstructorPending；Roster 的待命單位登場時仍須跑原生 constructor。
    /// Go 標籤：`json:"units_constructor_pending"`
    pub units_constructor_pending: Vec<bool>,
    /// Go 標籤：`json:"roster_constructor_pending"`
    pub roster_constructor_pending: Vec<bool>,
    /// Go 標籤：`json:"has_native_map_selector_cache"`
    pub has_native_map_selector_cache: bool,
    /// Go 標籤：`json:"native_map_selector_keys"`
    pub native_map_selector_keys: Vec<u8>,
    /// Go 標籤：`json:"native_map_selector_error"`
    pub native_map_selector_error_text: String,
    /// Go 標籤：`json:"native_map_cycle_state"`
    pub native_map_cycle_state: fdicon::NativeMapSpriteCycleState,
    /// Go 標籤：`json:"has_native_map_cycle_state"`
    pub has_native_map_cycle_state: bool,
    /// Go 標籤：`json:"native_terrain_phase_state"`
    pub native_terrain_phase_state: fdother::NativeTerrainPhaseState,
    /// Go 標籤：`json:"has_native_terrain_phase_state"`
    pub has_native_terrain_phase_state: bool,
    /// Go 標籤：`json:"native_terrain_flip_state"`
    pub native_terrain_flip_state: fdicon::NativeBinaryTickState,
    /// Go 標籤：`json:"native_unit_pixel_shift_state"`
    pub native_unit_pixel_shift_state: fdicon::NativeBinaryTickState,
    /// Go 標籤：`json:"has_native_map_binary_timing_state"`
    pub has_native_map_binary_timing_state: bool,
    /// Go 標籤：`json:"native_map_hud_state"`
    pub native_map_hud_state: NativeMapHUDRuntimeState,
    /// Go 標籤：`json:"has_native_map_hud_state"`
    pub has_native_map_hud_state: bool,
    /// Go 標籤：`json:"native_map_view_state"`
    pub native_map_view_state: NativeMapViewState,
    /// Go 標籤：`json:"has_native_map_view_state"`
    pub has_native_map_view_state: bool,
    /// Go 標籤：`json:"native_map_range_mode"`
    pub native_map_range_mode: i64,
    /// Go 標籤：`json:"has_native_map_range_mode_state"`
    pub has_native_map_range_mode_state: bool,
    /// Go 標籤：`json:"roster"`
    /// 所有權：本欄擁有單位本體；其他地方以 `UnitIdx` 指向這個陣列（見 `battle` 模組說明）。
    pub roster: Vec<Unit>,
    /// Go 標籤：`json:"pending_groups"`
    pub pending_groups: BTreeMap<i64, bool>,
    /// Go 標籤：`json:"own_deploy"`
    pub own_deploy: Vec<Cell>,
    /// Go 標籤：`json:"turn"`
    pub turn: i64,
    /// Go 標籤：`json:"native_round_counter"`
    pub native_round_counter: i64,
    /// Go 標籤：`json:"flags"`
    pub flags: BTreeMap<String, bool>,
    /// Go 標籤：`json:"native_event_state"`
    pub native_event_state: [u8; 32],
    /// Go 標籤：`json:"cost"`
    pub cost: Vec<i64>,
    /// Go 標籤：`json:"native_composition_event_bytes"`
    pub native_composition_event_bytes: Vec<u8>,
    /// Go 標籤：`json:"native_map_event_grid"`
    pub native_map_event_grid: Vec<u8>,
    /// Go 標籤：`json:"has_native_map_event_grid"`
    pub has_native_map_event_grid: bool,
    /// Go 標籤：`json:"native_field_control_raw"`
    pub native_field_control_raw: Vec<u8>,
    /// Go 標籤：`json:"native_turn_event_controls"`
    pub native_turn_event_controls: [NativeTurnEventControl; 16],
    /// Go 標籤：`json:"has_native_turn_event_control_state"`
    pub has_native_turn_event_control_state: bool,
    /// Go 標籤：`json:"native_chest_controls"`
    pub native_chest_controls: [NativeChestControl; 16],
    /// Go 標籤：`json:"has_native_chest_control_state"`
    pub has_native_chest_control_state: bool,
    /// Go 標籤：`json:"native_field_unit_controls"`
    pub native_field_unit_controls: Vec<NativeFieldUnitControl>,
    /// Go 標籤：`json:"has_native_field_control_state"`
    pub has_native_field_control_state: bool,
    /// Go 標籤：`json:"native_runtime_records"`
    pub native_runtime_records: Vec<NativeRuntimeRecordState>,
    /// Go 標籤：`json:"has_native_runtime_unit_projection"`
    pub has_native_runtime_unit_projection: bool,
    /// Go 標籤：`json:"has_native_pending_group_binding"`
    pub has_native_pending_group_binding: bool,
    /// Go 標籤：`json:"native_field_event_slots"`
    pub native_field_event_slots: Vec<i64>,
    /// Go 標籤：`json:"native_field_events"`
    pub native_field_events: Vec<NativeFieldEvent>,
    /// Go 標籤：`json:"native_tile_blit_modes"`
    pub native_tile_blit_modes: Vec<u8>,
    /// Go 標籤：`json:"native_terrain_control"`
    pub native_terrain_control: Vec<u8>,
    /// Go 標籤：`json:"native_terrain_move_codes"`
    pub native_terrain_move_codes: Vec<u8>,
    /// Go 標籤：`json:"spell_book"`
    pub spell_book: Vec<Spell>,
    /// Go 標籤：`json:"ai_command_spell"`
    pub ai_command_spell: BTreeMap<i64, i64>,
    /// Go 標籤：`json:"treasures"`
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    pub treasures: Option<Vec<SnapshotTreasure>>,
    /// Go 標籤：`json:"opened_treasure"`
    pub opened_treasure: BTreeMap<i64, bool>,
}

impl State {
    /// SnapshotReady 回報 transient 類欄位是否都是零值；不是就不能存快照。
    ///
    /// Go：`snapshot.go:165`，`func (s *State) SnapshotReady() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn snapshot_ready(&self) -> Result<()> {
        todo!("battle/snapshot.go:State.SnapshotReady")
    }

    /// Snapshot 深複製 saved 類欄位。輸出與 State 不共用任何可變記憶體。
    ///
    /// Go：`snapshot.go:179`，`func (s *State) Snapshot() (StateSnapshot, error)`
    pub fn snapshot(&self) -> Result<StateSnapshot> {
        todo!("battle/snapshot.go:State.Snapshot")
    }

    /// RestoreSnapshot 把 saved 類欄位寫到一份由同一章資產載入、且已完成應用層綁定的
    /// State。驗證全部通過才寫入；失敗時 s 不變。
    ///
    /// Go：`snapshot.go:258`，`func (s *State) RestoreSnapshot(snap StateSnapshot) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn restore_snapshot(&mut self, snap: &StateSnapshot) -> Result<()> {
        todo!("battle/snapshot.go:State.RestoreSnapshot")
    }
}

/// Go：`snapshot.go:339`，`func snapshotUnits(list []*Unit) ([]*Unit, []bool)`
/// 所有權：回傳的單位是 `list` 的索引。
/// 所有權：回傳逐一複製的單位本體。Go 在沒有待建構單位時回傳 nil 的 `pending`，Rust 回傳空 `Vec`；Go 會略過 nil 元素，Rust 的 `&[Unit]` 沒有 nil。
pub(crate) fn snapshot_units(list: &[Unit]) -> (Vec<Unit>, Vec<bool>) {
    todo!("battle/snapshot.go:snapshotUnits")
}

/// Go：`snapshot.go:361`，`func restoreUnits(list []*Unit, pending []bool) []*Unit`
/// 所有權：回傳的單位是 `list` 的索引。
/// 所有權：回傳逐一複製的單位本體（Go `c := *u; out[i] = &c`）。
pub(crate) fn restore_units(list: &[Unit], pending: &[bool]) -> Vec<Unit> {
    todo!("battle/snapshot.go:restoreUnits")
}

/// Go：`snapshot.go:377`，`func cloneMap[K comparable, V any](m map[K]V) map[K]V`
/// Rust：Go 泛型函式；`BTreeMap::clone` 已逐項複製。Go 的 nil 與空 map 差別由呼叫端以 `Option` 表達。
pub(crate) fn clone_map<K: Ord + Clone, V: Clone>(m: &BTreeMap<K, V>) -> BTreeMap<K, V> {
    todo!("battle/snapshot.go:cloneMap")
}

/// cloneSlice 深複製並保留 nil 與空切片的差別；往返後要與原值逐欄相等。
///
/// Go：`snapshot.go:389`，`func cloneSlice[T any](s []T) []T`
/// Rust：Go 泛型函式；Go 註解要求保留 nil 與空切片的差別，Rust 的欄位沒有 nil 切片，這個差別在快照格式層處理。
pub(crate) fn clone_slice<T: Clone>(s: &[T]) -> Vec<T> {
    todo!("battle/snapshot.go:cloneSlice")
}

/// FiredEventRef 記錄一個已觸發事件的劇本索引與 ID；兩者都要相符才還原。
///
/// Go：`snapshot.go:397`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FiredEventRef {
    /// Go 標籤：`json:"index"`
    pub index: i64,
    /// Go 標籤：`json:"id"`
    pub id: String,
}

impl Scenario {
    /// FiredEvents 回傳已觸發的事件（依劇本順序）。未匯出的 fired 旗標是劇本的
    /// 執行期狀態，快照需要它才不會在讀檔後重播事件。
    ///
    /// Go：`snapshot.go:404`，`func (sc *Scenario) FiredEvents() []FiredEventRef`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn fired_events(&self) -> Vec<FiredEventRef> {
        todo!("battle/snapshot.go:Scenario.FiredEvents")
    }

    /// PendingJoinCount 回報尚未發布的 JOIN；快照受理時必須為零。
    ///
    /// Go：`snapshot.go:418`，`func (sc *Scenario) PendingJoinCount() int`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn pending_join_count(&self) -> i64 {
        todo!("battle/snapshot.go:Scenario.PendingJoinCount")
    }

    /// RestoreFiredEvents 還原 fired；索引越界、ID 不符或重複時拒絕且不改變劇本。
    ///
    /// Go：`snapshot.go:426`，`func (sc *Scenario) RestoreFiredEvents(refs []FiredEventRef) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn restore_fired_events(&mut self, refs: &[FiredEventRef]) -> Result<()> {
        todo!("battle/snapshot.go:Scenario.RestoreFiredEvents")
    }
}
