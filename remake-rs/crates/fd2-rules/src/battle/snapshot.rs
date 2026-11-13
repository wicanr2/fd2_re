//! Go 來源：`remake/internal/battle/snapshot.go`（448 行）。
//!
//! snapshot.go — 戰場 JSON 快照（重製端自有存檔擴充，56「2026-11-13 #303 戰場JSON快照」）。
//!
//! 快照不是原版 FD2.SAV ABI。State 每個欄位都必須歸入 stateSnapshotFieldClass 的
//! 三類之一，snapshot_test 以反射列舉欄位；新增欄位沒分類就失敗，避免新狀態漏存後
//! 讀檔靜默偏差。
//!
//! # JSON 格式（與 Go `encoding/json` 互通）
//!
//! `StateSnapshot` 的 serde 輸出要讓 Go 讀得進去，Go 寫出的快照 Rust 也要讀得進去：
//!
//! - 欄位名是 Go 標籤；Go 型別沒有標籤時用 Go 欄位名（`Spell` 的 `Name`、`Treasure` 的 `Slot` 等）。
//! - Go nil 切片與 nil map 寫成 `null`。Rust 的 `Vec`／`BTreeMap` 不分 nil 與空值，空的一律寫 `null`
//!   （`go_slice`、`go_map`）；Go 寫出的 `[]`、`{}` 讀成空值。`State` 以 `Option` 區分 nil 的三個欄位
//!   （`cost`、`treasures`、`opened_treasure`）在快照裡也是 `Option`，`None` 寫 `null`、`Some` 照寫。
//! - Go `[]byte` 是標準 base64（`go_bytes`，空的寫 `null`）；Go 固定長度陣列 `[N]byte` 是數字陣列。
//! - `battle` 裡沒有 serde 的別組型別（`Spell`、`Treasure`、`NativeChestControl`、
//!   `NativeFieldUnitControl`、`NativeRuntimeRecordState`、`NativeMapHUDRuntimeState`、`NativeMapViewState`）
//!   在本檔以鏡像結構轉換，不改那些組的檔案。
//! - Go 的 `[]*Unit` 可以含 nil 元素（寫成 `null`）；Rust 的 `Vec<Unit>` 沒有 nil，讀到 `null` 元素會拒收。
//!   正式戰場不會出現 nil 單位，列入 `KNOWN_DIFFERENCES`。

use serde::{Deserialize, Deserializer, Serialize, Serializer};

use super::*;

/// StateSnapshotVersion 只在快照欄位語意改變時遞增；讀到不同版本一律拒絕。
pub const STATE_SNAPSHOT_VERSION: i64 = 1;

/// 存入快照
#[cfg_attr(not(test), allow(dead_code))] // Go 也只在 snapshot_test 經分類表用到
pub(crate) const SNAPSHOT_SAVED: &str = "saved";

/// 讀檔時由應用層從資產與劇本重新綁定
#[cfg_attr(not(test), allow(dead_code))] // Go 也只在 snapshot_test 經分類表用到
pub(crate) const SNAPSHOT_BOUND: &str = "bound";

/// 受理存檔時必須是零值
#[cfg_attr(not(test), allow(dead_code))] // Go 也只在 snapshot_test 經分類表用到
pub(crate) const SNAPSHOT_TRANSIENT: &str = "transient";

/// Go：`snapshot.go:26`
#[cfg_attr(not(test), allow(dead_code))] // Go 也只在 snapshot_test 用到
pub(crate) static STATE_SNAPSHOT_FIELD_CLASS: LazyLock<BTreeMap<&str, &str>> = LazyLock::new(|| {
    BTreeMap::from([
        ("Balance", SNAPSHOT_BOUND),
        ("W", SNAPSHOT_SAVED),
        ("H", SNAPSHOT_SAVED),
        ("Units", SNAPSHOT_SAVED),
        ("NativeMapSelectorCache", SNAPSHOT_SAVED),
        ("NativeMapSelectorError", SNAPSHOT_SAVED),
        ("NativeMapCycleState", SNAPSHOT_SAVED),
        ("HasNativeMapCycleState", SNAPSHOT_SAVED),
        ("NativeTerrainPhaseState", SNAPSHOT_SAVED),
        ("HasNativeTerrainPhaseState", SNAPSHOT_SAVED),
        ("NativeTerrainFlipState", SNAPSHOT_SAVED),
        ("NativeUnitPixelShiftState", SNAPSHOT_SAVED),
        ("HasNativeMapBinaryTimingState", SNAPSHOT_SAVED),
        ("NativeMapHUDState", SNAPSHOT_SAVED),
        ("HasNativeMapHUDState", SNAPSHOT_SAVED),
        ("NativeMapViewState", SNAPSHOT_SAVED),
        ("HasNativeMapViewState", SNAPSHOT_SAVED),
        ("NativeMapRangeMode", SNAPSHOT_SAVED),
        ("HasNativeMapRangeModeState", SNAPSHOT_SAVED),
        ("Roster", SNAPSHOT_SAVED),
        ("PendingGroups", SNAPSHOT_SAVED),
        ("nativeFutureItemRows", SNAPSHOT_BOUND),
        ("nativeAIScan", SNAPSHOT_TRANSIENT),
        ("OwnDeploy", SNAPSHOT_SAVED),
        ("Turn", SNAPSHOT_SAVED),
        ("NativeRoundCounter", SNAPSHOT_SAVED),
        ("Flags", SNAPSHOT_SAVED),
        ("NativeEventState", SNAPSHOT_SAVED),
        ("Cost", SNAPSHOT_SAVED),
        ("NativeCompositionEventBytes", SNAPSHOT_SAVED),
        ("NativeMapEventGrid", SNAPSHOT_SAVED),
        ("HasNativeMapEventGrid", SNAPSHOT_SAVED),
        ("NativeFieldControlRaw", SNAPSHOT_SAVED),
        ("NativeTurnEventControls", SNAPSHOT_SAVED),
        ("HasNativeTurnEventControlState", SNAPSHOT_SAVED),
        ("NativeChestControls", SNAPSHOT_SAVED),
        ("HasNativeChestControlState", SNAPSHOT_SAVED),
        ("NativeFieldUnitControls", SNAPSHOT_SAVED),
        ("HasNativeFieldControlState", SNAPSHOT_SAVED),
        ("NativeRuntimeRecords", SNAPSHOT_SAVED),
        ("HasNativeRuntimeUnitProjection", SNAPSHOT_SAVED),
        ("HasNativePendingGroupBinding", SNAPSHOT_SAVED),
        ("NativeFieldEventSlots", SNAPSHOT_SAVED),
        ("NativeFieldEvents", SNAPSHOT_SAVED),
        ("NativeFieldEventRules", SNAPSHOT_BOUND),
        ("NativeTileBlitModes", SNAPSHOT_SAVED),
        ("NativeTerrainControl", SNAPSHOT_SAVED),
        ("NativeTerrainMoveCodes", SNAPSHOT_SAVED),
        ("nativeMovementCostRows", SNAPSHOT_BOUND),
        ("SpellBook", SNAPSHOT_SAVED),
        ("NativeCommandBook", SNAPSHOT_BOUND),
        ("NativeCommandResistances", SNAPSHOT_BOUND),
        ("CommandLearn", SNAPSHOT_BOUND),
        ("CommandLearnSelectors", SNAPSHOT_BOUND),
        ("NativeGrowthRows", SNAPSHOT_BOUND),
        ("NativeGrowthRollObserver", SNAPSHOT_BOUND),
        ("NativeDeathExpCancel", SNAPSHOT_BOUND),
        ("AICommandSpell", SNAPSHOT_SAVED),
        ("Treasures", SNAPSHOT_SAVED),
        ("NativeTreasureEventRules", SNAPSHOT_BOUND),
        ("NativeExperienceAccumulator", SNAPSHOT_TRANSIENT),
        ("OpenedTreasure", SNAPSHOT_SAVED),
    ])
});

/// SnapshotTreasure 是 Treasures 的 JSON 形式；map 的鍵 Cell 不是字串。
///
/// Go：`snapshot.go:92`
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(default)]
pub struct SnapshotTreasure {
    /// Go 標籤：`json:"cell"`
    pub cell: Cell,
    /// Go 標籤：`json:"treasure"`
    #[serde(
        serialize_with = "ser_via::<Treasure, TreasureJson, _>",
        deserialize_with = "de_via::<Treasure, TreasureJson, _>"
    )]
    pub treasure: Treasure,
}

/// StateSnapshot 是 State 的 saved 類欄位。指標陣列的 aliasing 不保存：Units 與
/// Roster 是兩個獨立陣列，快照前先驗證沒有共用同一個 *Unit。
///
/// Go：`snapshot.go:99`
#[derive(Debug, Clone, Default, PartialEq, Serialize, Deserialize)]
#[serde(default)]
pub struct StateSnapshot {
    /// Go 標籤：`json:"version"`
    pub version: i64,
    /// Go 標籤：`json:"w"`
    pub w: i64,
    /// Go 標籤：`json:"h"`
    pub h: i64,
    /// Go 標籤：`json:"units"`
    /// 所有權：本欄擁有單位本體；其他地方以 `UnitIdx` 指向這個陣列（見 `battle` 模組說明）。
    #[serde(with = "go_slice")]
    pub units: Vec<Unit>,
    /// UnitsConstructorPending／RosterConstructorPending 保存未匯出的
    /// nativeConstructorPending；Roster 的待命單位登場時仍須跑原生 constructor。
    /// Go 標籤：`json:"units_constructor_pending"`
    #[serde(with = "go_slice")]
    pub units_constructor_pending: Vec<bool>,
    /// Go 標籤：`json:"roster_constructor_pending"`
    #[serde(with = "go_slice")]
    pub roster_constructor_pending: Vec<bool>,
    /// Go 標籤：`json:"has_native_map_selector_cache"`
    pub has_native_map_selector_cache: bool,
    /// Go 標籤：`json:"native_map_selector_keys"`
    #[serde(with = "go_bytes")]
    pub native_map_selector_keys: Vec<u8>,
    /// Go 標籤：`json:"native_map_selector_error"`
    #[serde(rename = "native_map_selector_error", deserialize_with = "null_as_default")]
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
    #[serde(
        serialize_with = "ser_via::<NativeMapHUDRuntimeState, HudJson, _>",
        deserialize_with = "de_via::<NativeMapHUDRuntimeState, HudJson, _>"
    )]
    pub native_map_hud_state: NativeMapHUDRuntimeState,
    /// Go 標籤：`json:"has_native_map_hud_state"`
    pub has_native_map_hud_state: bool,
    /// Go 標籤：`json:"native_map_view_state"`
    #[serde(
        serialize_with = "ser_via::<NativeMapViewState, ViewJson, _>",
        deserialize_with = "de_via::<NativeMapViewState, ViewJson, _>"
    )]
    pub native_map_view_state: NativeMapViewState,
    /// Go 標籤：`json:"has_native_map_view_state"`
    pub has_native_map_view_state: bool,
    /// Go 標籤：`json:"native_map_range_mode"`
    pub native_map_range_mode: i64,
    /// Go 標籤：`json:"has_native_map_range_mode_state"`
    pub has_native_map_range_mode_state: bool,
    /// Go 標籤：`json:"roster"`
    /// 所有權：本欄擁有單位本體；其他地方以 `UnitIdx` 指向這個陣列（見 `battle` 模組說明）。
    #[serde(with = "go_slice")]
    pub roster: Vec<Unit>,
    /// Go 標籤：`json:"pending_groups"`
    #[serde(with = "go_map")]
    pub pending_groups: BTreeMap<i64, bool>,
    /// Go 標籤：`json:"own_deploy"`
    #[serde(with = "go_slice")]
    pub own_deploy: Vec<Cell>,
    /// Go 標籤：`json:"turn"`
    pub turn: i64,
    /// Go 標籤：`json:"native_round_counter"`
    pub native_round_counter: i64,
    /// Go 標籤：`json:"flags"`
    #[serde(with = "go_map")]
    pub flags: BTreeMap<String, bool>,
    /// Go 標籤：`json:"native_event_state"`
    pub native_event_state: [u8; 32],
    /// Go 標籤：`json:"cost"`
    /// Rust：`State::cost` 以 `Option` 區分 nil 與空值，快照照樣保存。
    pub cost: Option<Vec<i64>>,
    /// Go 標籤：`json:"native_composition_event_bytes"`
    #[serde(with = "go_bytes")]
    pub native_composition_event_bytes: Vec<u8>,
    /// Go 標籤：`json:"native_map_event_grid"`
    #[serde(with = "go_bytes")]
    pub native_map_event_grid: Vec<u8>,
    /// Go 標籤：`json:"has_native_map_event_grid"`
    pub has_native_map_event_grid: bool,
    /// Go 標籤：`json:"native_field_control_raw"`
    #[serde(with = "go_bytes")]
    pub native_field_control_raw: Vec<u8>,
    /// Go 標籤：`json:"native_turn_event_controls"`
    pub native_turn_event_controls: [NativeTurnEventControl; 16],
    /// Go 標籤：`json:"has_native_turn_event_control_state"`
    pub has_native_turn_event_control_state: bool,
    /// Go 標籤：`json:"native_chest_controls"`
    #[serde(
        serialize_with = "ser_array_via::<NativeChestControl, ChestJson, _>",
        deserialize_with = "de_array_via::<NativeChestControl, ChestJson, 16, _>"
    )]
    pub native_chest_controls: [NativeChestControl; 16],
    /// Go 標籤：`json:"has_native_chest_control_state"`
    pub has_native_chest_control_state: bool,
    /// Go 標籤：`json:"native_field_unit_controls"`
    #[serde(
        serialize_with = "ser_vec_via::<NativeFieldUnitControl, FieldUnitJson, _>",
        deserialize_with = "de_vec_via::<NativeFieldUnitControl, FieldUnitJson, _>"
    )]
    pub native_field_unit_controls: Vec<NativeFieldUnitControl>,
    /// Go 標籤：`json:"has_native_field_control_state"`
    pub has_native_field_control_state: bool,
    /// Go 標籤：`json:"native_runtime_records"`
    #[serde(
        serialize_with = "ser_vec_via::<NativeRuntimeRecordState, RuntimeRecordJson, _>",
        deserialize_with = "de_vec_via::<NativeRuntimeRecordState, RuntimeRecordJson, _>"
    )]
    pub native_runtime_records: Vec<NativeRuntimeRecordState>,
    /// Go 標籤：`json:"has_native_runtime_unit_projection"`
    pub has_native_runtime_unit_projection: bool,
    /// Go 標籤：`json:"has_native_pending_group_binding"`
    pub has_native_pending_group_binding: bool,
    /// Go 標籤：`json:"native_field_event_slots"`
    #[serde(with = "go_slice")]
    pub native_field_event_slots: Vec<i64>,
    /// Go 標籤：`json:"native_field_events"`
    #[serde(with = "go_slice")]
    pub native_field_events: Vec<NativeFieldEvent>,
    /// Go 標籤：`json:"native_tile_blit_modes"`
    #[serde(with = "go_bytes")]
    pub native_tile_blit_modes: Vec<u8>,
    /// Go 標籤：`json:"native_terrain_control"`
    #[serde(with = "go_bytes")]
    pub native_terrain_control: Vec<u8>,
    /// Go 標籤：`json:"native_terrain_move_codes"`
    #[serde(with = "go_bytes")]
    pub native_terrain_move_codes: Vec<u8>,
    /// Go 標籤：`json:"spell_book"`
    #[serde(
        serialize_with = "ser_vec_via::<Spell, SpellJson, _>",
        deserialize_with = "de_vec_via::<Spell, SpellJson, _>"
    )]
    pub spell_book: Vec<Spell>,
    /// Go 標籤：`json:"ai_command_spell"`
    #[serde(with = "go_map")]
    pub ai_command_spell: BTreeMap<i64, i64>,
    /// Go 標籤：`json:"treasures"`
    /// Rust：Go 程式以 `== nil` 區分這個欄位的 nil 與空值，所以包成 `Option`。
    pub treasures: Option<Vec<SnapshotTreasure>>,
    /// Go 標籤：`json:"opened_treasure"`
    /// Rust：`State::opened_treasure` 以 `Option` 區分 nil 與空值，快照照樣保存。
    pub opened_treasure: Option<BTreeMap<i64, bool>>,
}

impl State {
    /// SnapshotReady 回報 transient 類欄位是否都是零值；不是就不能存快照。
    ///
    /// Go：`snapshot.go:165`，`func (s *State) SnapshotReady() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn snapshot_ready(&self) -> Result<()> {
        if self.native_ai_scan != NativeAIScanState::default() {
            return Err(battle_err!("battle snapshot: enemy AI scan is active"));
        }
        if self.native_experience_accumulator != 0 {
            return Err(battle_err!("battle snapshot: action experience is pending"));
        }
        Ok(())
    }

    /// Snapshot 深複製 saved 類欄位。輸出與 State 不共用任何可變記憶體。
    ///
    /// Go：`snapshot.go:179`，`func (s *State) Snapshot() (StateSnapshot, error)`
    /// Go 先檢查 Units 與 Roster 沒有共用同一個 `*Unit`；Rust 的兩個 `Vec<Unit>` 各自擁有單位，
    /// 由型別保證（`battle` 模組說明第 1 點），該檢查不移植。
    pub fn snapshot(&self) -> Result<StateSnapshot> {
        self.snapshot_ready()?;
        let mut out = StateSnapshot {
            version: STATE_SNAPSHOT_VERSION,
            w: self.w,
            h: self.h,
            native_map_cycle_state: self.native_map_cycle_state,
            has_native_map_cycle_state: self.has_native_map_cycle_state,
            native_terrain_phase_state: self.native_terrain_phase_state,
            has_native_terrain_phase_state: self.has_native_terrain_phase_state,
            native_terrain_flip_state: self.native_terrain_flip_state,
            native_unit_pixel_shift_state: self.native_unit_pixel_shift_state,
            has_native_map_binary_timing_state: self.has_native_map_binary_timing_state,
            native_map_hud_state: self.native_map_hud_state,
            has_native_map_hud_state: self.has_native_map_hud_state,
            native_map_view_state: self.native_map_view_state,
            has_native_map_view_state: self.has_native_map_view_state,
            native_map_range_mode: self.native_map_range_mode,
            has_native_map_range_mode_state: self.has_native_map_range_mode_state,
            turn: self.turn,
            native_round_counter: self.native_round_counter,
            native_event_state: self.native_event_state,
            has_native_map_event_grid: self.has_native_map_event_grid,
            native_turn_event_controls: self.native_turn_event_controls,
            has_native_turn_event_control_state: self.has_native_turn_event_control_state,
            native_chest_controls: self.native_chest_controls,
            has_native_chest_control_state: self.has_native_chest_control_state,
            has_native_field_control_state: self.has_native_field_control_state,
            has_native_runtime_unit_projection: self.has_native_runtime_unit_projection,
            has_native_pending_group_binding: self.has_native_pending_group_binding,
            ..Default::default()
        };
        (out.units, out.units_constructor_pending) = snapshot_units(&self.units);
        (out.roster, out.roster_constructor_pending) = snapshot_units(&self.roster);
        if let Some(cache) = &self.native_map_selector_cache {
            out.has_native_map_selector_cache = true;
            out.native_map_selector_keys = cache.keys();
        }
        if let Some(err) = &self.native_map_selector_error {
            out.native_map_selector_error_text = err.to_string();
            if out.native_map_selector_error_text.is_empty() {
                out.native_map_selector_error_text = "native map selector rejected".into();
            }
        }
        out.pending_groups = clone_map(&self.pending_groups);
        out.own_deploy = clone_slice(&self.own_deploy);
        out.flags = clone_map(&self.flags);
        out.cost = self.cost.as_deref().map(clone_slice);
        out.native_composition_event_bytes = clone_slice(&self.native_composition_event_bytes);
        out.native_map_event_grid = clone_slice(&self.native_map_event_grid);
        out.native_field_control_raw = clone_slice(&self.native_field_control_raw);
        out.native_field_unit_controls = clone_slice(&self.native_field_unit_controls);
        out.native_runtime_records = clone_slice(&self.native_runtime_records);
        out.native_field_event_slots = clone_slice(&self.native_field_event_slots);
        out.native_field_events = clone_slice(&self.native_field_events);
        out.native_tile_blit_modes = clone_slice(&self.native_tile_blit_modes);
        out.native_terrain_control = clone_slice(&self.native_terrain_control);
        out.native_terrain_move_codes = clone_slice(&self.native_terrain_move_codes);
        out.spell_book = clone_slice(&self.spell_book);
        out.ai_command_spell = clone_map(&self.ai_command_spell);
        out.opened_treasure = self.opened_treasure.as_ref().map(clone_map);
        // Go 迭代 map 後依 (Y, X) 排序；Cell 不重複，排序結果唯一，與 map 迭代順序無關。
        out.treasures = self.treasures.as_ref().map(|treasures| {
            let mut list: Vec<SnapshotTreasure> = treasures
                .iter()
                .map(|(cell, treasure)| SnapshotTreasure { cell: *cell, treasure: treasure.clone() })
                .collect();
            list.sort_by_key(|t| (t.cell.y, t.cell.x));
            list
        });
        Ok(out)
    }

    /// RestoreSnapshot 把 saved 類欄位寫到一份由同一章資產載入、且已完成應用層綁定的
    /// State。驗證全部通過才寫入；失敗時 s 不變。
    ///
    /// Go：`snapshot.go:258`，`func (s *State) RestoreSnapshot(snap StateSnapshot) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn restore_snapshot(&mut self, snap: &StateSnapshot) -> Result<()> {
        if snap.version != STATE_SNAPSHOT_VERSION {
            return Err(battle_err!("battle snapshot: version {}, want {}", snap.version, STATE_SNAPSHOT_VERSION));
        }
        if snap.w <= 0 || snap.h <= 0 {
            return Err(battle_err!("battle snapshot: invalid field size"));
        }
        if let Some(cost) = &snap.cost
            && !cost.is_empty()
            && cost.len() as i64 != snap.w.wrapping_mul(snap.h)
        {
            return Err(battle_err!("battle snapshot: cost size mismatch"));
        }
        if !snap.units_constructor_pending.is_empty() && snap.units_constructor_pending.len() != snap.units.len()
            || !snap.roster_constructor_pending.is_empty() && snap.roster_constructor_pending.len() != snap.roster.len()
        {
            return Err(battle_err!("battle snapshot: constructor flag count mismatch"));
        }
        let mut cache = None;
        if snap.has_native_map_selector_cache {
            match fdicon::NativeSelectorCache::from_keys(&snap.native_map_selector_keys) {
                Ok(c) => cache = Some(c),
                Err(err) => return Err(battle_err!("battle snapshot: {err}")),
            }
        }
        let mut treasures = BTreeMap::new();
        for t in snap.treasures.iter().flatten() {
            if treasures.contains_key(&t.cell) {
                return Err(battle_err!("battle snapshot: duplicate treasure cell"));
            }
            treasures.insert(t.cell, t.treasure.clone());
        }
        let treasures = snap.treasures.as_ref().map(|_| treasures);

        (self.w, self.h) = (snap.w, snap.h);
        self.units = restore_units(&snap.units, &snap.units_constructor_pending);
        self.roster = restore_units(&snap.roster, &snap.roster_constructor_pending);
        self.native_map_selector_cache = cache;
        self.native_map_selector_error = None;
        if !snap.native_map_selector_error_text.is_empty() {
            self.native_map_selector_error = Some(BattleError::new(snap.native_map_selector_error_text.clone()));
        }
        (self.native_map_cycle_state, self.has_native_map_cycle_state) =
            (snap.native_map_cycle_state, snap.has_native_map_cycle_state);
        (self.native_terrain_phase_state, self.has_native_terrain_phase_state) =
            (snap.native_terrain_phase_state, snap.has_native_terrain_phase_state);
        (self.native_terrain_flip_state, self.native_unit_pixel_shift_state) =
            (snap.native_terrain_flip_state, snap.native_unit_pixel_shift_state);
        self.has_native_map_binary_timing_state = snap.has_native_map_binary_timing_state;
        (self.native_map_hud_state, self.has_native_map_hud_state) =
            (snap.native_map_hud_state, snap.has_native_map_hud_state);
        (self.native_map_view_state, self.has_native_map_view_state) =
            (snap.native_map_view_state, snap.has_native_map_view_state);
        (self.native_map_range_mode, self.has_native_map_range_mode_state) =
            (snap.native_map_range_mode, snap.has_native_map_range_mode_state);
        self.pending_groups = clone_map(&snap.pending_groups);
        self.own_deploy = clone_slice(&snap.own_deploy);
        (self.turn, self.native_round_counter) = (snap.turn, snap.native_round_counter);
        self.flags = clone_map(&snap.flags);
        self.native_event_state = snap.native_event_state;
        self.cost = snap.cost.as_deref().map(clone_slice);
        self.native_composition_event_bytes = clone_slice(&snap.native_composition_event_bytes);
        self.native_map_event_grid = clone_slice(&snap.native_map_event_grid);
        self.has_native_map_event_grid = snap.has_native_map_event_grid;
        self.native_field_control_raw = clone_slice(&snap.native_field_control_raw);
        (self.native_turn_event_controls, self.has_native_turn_event_control_state) =
            (snap.native_turn_event_controls, snap.has_native_turn_event_control_state);
        (self.native_chest_controls, self.has_native_chest_control_state) =
            (snap.native_chest_controls, snap.has_native_chest_control_state);
        self.native_field_unit_controls = clone_slice(&snap.native_field_unit_controls);
        self.has_native_field_control_state = snap.has_native_field_control_state;
        self.native_runtime_records = clone_slice(&snap.native_runtime_records);
        self.has_native_runtime_unit_projection = snap.has_native_runtime_unit_projection;
        self.has_native_pending_group_binding = snap.has_native_pending_group_binding;
        self.native_field_event_slots = clone_slice(&snap.native_field_event_slots);
        self.native_field_events = clone_slice(&snap.native_field_events);
        self.native_tile_blit_modes = clone_slice(&snap.native_tile_blit_modes);
        self.native_terrain_control = clone_slice(&snap.native_terrain_control);
        self.native_terrain_move_codes = clone_slice(&snap.native_terrain_move_codes);
        self.spell_book = clone_slice(&snap.spell_book);
        self.ai_command_spell = clone_map(&snap.ai_command_spell);
        self.treasures = treasures;
        self.opened_treasure = snap.opened_treasure.as_ref().map(clone_map);
        self.native_ai_scan = NativeAIScanState::default();
        self.native_experience_accumulator = 0;
        Ok(())
    }
}

/// Go：`snapshot.go:339`，`func snapshotUnits(list []*Unit) ([]*Unit, []bool)`
/// 所有權：回傳逐一複製的單位本體。Go 在沒有待建構單位時回傳 nil 的 `pending`，Rust 回傳空 `Vec`；Go 會略過 nil 元素，Rust 的 `&[Unit]` 沒有 nil。
pub(crate) fn snapshot_units(list: &[Unit]) -> (Vec<Unit>, Vec<bool>) {
    let out = list.to_vec();
    let mut pending = Vec::new();
    for (i, u) in list.iter().enumerate() {
        if u.native_constructor_pending {
            if pending.is_empty() {
                pending = vec![false; list.len()];
            }
            pending[i] = true;
        }
    }
    (out, pending)
}

/// Go：`snapshot.go:361`，`func restoreUnits(list []*Unit, pending []bool) []*Unit`
/// 所有權：回傳逐一複製的單位本體（Go `c := *u; out[i] = &c`）。
pub(crate) fn restore_units(list: &[Unit], pending: &[bool]) -> Vec<Unit> {
    list.iter()
        .enumerate()
        .map(|(i, u)| {
            let mut c = u.clone();
            c.native_constructor_pending = pending.get(i).copied().unwrap_or(false);
            c
        })
        .collect()
}

/// Go：`snapshot.go:377`，`func cloneMap[K comparable, V any](m map[K]V) map[K]V`
/// Rust：Go 泛型函式；`BTreeMap::clone` 已逐項複製。Go 的 nil 與空 map 差別由呼叫端以 `Option` 表達。
pub(crate) fn clone_map<K: Ord + Clone, V: Clone>(m: &BTreeMap<K, V>) -> BTreeMap<K, V> {
    m.clone()
}

/// cloneSlice 深複製並保留 nil 與空切片的差別；往返後要與原值逐欄相等。
///
/// Go：`snapshot.go:389`，`func cloneSlice[T any](s []T) []T`
/// Rust：Go 泛型函式；Go 註解要求保留 nil 與空切片的差別，Rust 的欄位沒有 nil 切片，這個差別在快照格式層處理。
pub(crate) fn clone_slice<T: Clone>(s: &[T]) -> Vec<T> {
    s.to_vec()
}

/// FiredEventRef 記錄一個已觸發事件的劇本索引與 ID；兩者都要相符才還原。
///
/// Go：`snapshot.go:397`
#[derive(Debug, Clone, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct FiredEventRef {
    /// Go 標籤：`json:"index"`
    pub index: i64,
    /// Go 標籤：`json:"id"`
    #[serde(deserialize_with = "null_as_default")]
    pub id: String,
}

impl Scenario {
    /// FiredEvents 回傳已觸發的事件（依劇本順序）。未匯出的 fired 旗標是劇本的
    /// 執行期狀態，快照需要它才不會在讀檔後重播事件。
    ///
    /// Go：`snapshot.go:404`，`func (sc *Scenario) FiredEvents() []FiredEventRef`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn fired_events(&self) -> Vec<FiredEventRef> {
        self.events
            .iter()
            .enumerate()
            .filter(|(_, e)| e.fired)
            .map(|(i, e)| FiredEventRef { index: i as i64, id: e.id.clone() })
            .collect()
    }

    /// PendingJoinCount 回報尚未發布的 JOIN；快照受理時必須為零。
    ///
    /// Go：`snapshot.go:418`，`func (sc *Scenario) PendingJoinCount() int`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn pending_join_count(&self) -> i64 {
        self.pending_joins.len() as i64
    }

    /// RestoreFiredEvents 還原 fired；索引越界、ID 不符或重複時拒絕且不改變劇本。
    ///
    /// Go：`snapshot.go:426`，`func (sc *Scenario) RestoreFiredEvents(refs []FiredEventRef) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn restore_fired_events(&mut self, refs: &[FiredEventRef]) -> Result<()> {
        let mut want = vec![false; self.events.len()];
        for r in refs {
            if r.index < 0 || r.index >= self.events.len() as i64 || self.events[r.index as usize].id != r.id {
                return Err(battle_err!(
                    "battle snapshot: scenario event {} {} does not match",
                    r.index,
                    go_quote(&r.id)
                ));
            }
            if want[r.index as usize] {
                return Err(battle_err!("battle snapshot: repeated scenario event {}", r.index));
            }
            want[r.index as usize] = true;
        }
        for (event, fired) in self.events.iter_mut().zip(want) {
            event.fired = fired;
        }
        self.pending_joins.clear();
        Ok(())
    }
}

// ---- 快照 JSON 的 serde 輔助（Go `encoding/json` 互通，見模組說明）----

/// Go nil 切片：空的寫 `null`；讀取接受 `null` 與陣列。
mod go_slice {
    use super::*;

    pub(super) fn serialize<T: Serialize, S: Serializer>(v: &[T], s: S) -> std::result::Result<S::Ok, S::Error> {
        if v.is_empty() { s.serialize_none() } else { s.collect_seq(v) }
    }

    pub(super) fn deserialize<'de, T: Deserialize<'de>, D: Deserializer<'de>>(
        d: D,
    ) -> std::result::Result<Vec<T>, D::Error> {
        Ok(Option::<Vec<T>>::deserialize(d)?.unwrap_or_default())
    }
}

/// Go nil map：空的寫 `null`；讀取接受 `null` 與物件。整數鍵由 serde_json 寫成字串，與 Go 相同。
mod go_map {
    use super::*;

    pub(super) fn serialize<K: Serialize, V: Serialize, S: Serializer>(
        v: &BTreeMap<K, V>,
        s: S,
    ) -> std::result::Result<S::Ok, S::Error> {
        if v.is_empty() { s.serialize_none() } else { s.collect_map(v) }
    }

    pub(super) fn deserialize<'de, K: Deserialize<'de> + Ord, V: Deserialize<'de>, D: Deserializer<'de>>(
        d: D,
    ) -> std::result::Result<BTreeMap<K, V>, D::Error> {
        Ok(Option::<BTreeMap<K, V>>::deserialize(d)?.unwrap_or_default())
    }
}

/// 以鏡像結構 `M` 寫出 `T`。
fn ser_via<T, M, S>(v: &T, s: S) -> std::result::Result<S::Ok, S::Error>
where
    M: Serialize + for<'a> From<&'a T>,
    S: Serializer,
{
    M::from(v).serialize(s)
}

/// 以鏡像結構 `M` 讀入 `T`；`null` 讀成零值（Go 讀 `null` 到結構時不改動）。
fn de_via<'de, T, M, D>(d: D) -> std::result::Result<T, D::Error>
where
    T: From<M> + Default,
    M: Deserialize<'de>,
    D: Deserializer<'de>,
{
    Ok(Option::<M>::deserialize(d)?.map(T::from).unwrap_or_default())
}

/// 以鏡像結構寫出切片；空的寫 `null`（Go nil 切片）。
fn ser_vec_via<T, M, S>(v: &[T], s: S) -> std::result::Result<S::Ok, S::Error>
where
    M: Serialize + for<'a> From<&'a T>,
    S: Serializer,
{
    if v.is_empty() { s.serialize_none() } else { s.collect_seq(v.iter().map(M::from)) }
}

/// 以鏡像結構讀入切片；`null` 讀成空切片。
fn de_vec_via<'de, T, M, D>(d: D) -> std::result::Result<Vec<T>, D::Error>
where
    T: From<M>,
    M: Deserialize<'de>,
    D: Deserializer<'de>,
{
    Ok(Option::<Vec<M>>::deserialize(d)?.unwrap_or_default().into_iter().map(T::from).collect())
}

/// 以鏡像結構寫出 Go 固定長度陣列（一律寫成 JSON 陣列）。
fn ser_array_via<T, M, S>(v: &[T], s: S) -> std::result::Result<S::Ok, S::Error>
where
    M: Serialize + for<'a> From<&'a T>,
    S: Serializer,
{
    s.collect_seq(v.iter().map(M::from))
}

/// 以鏡像結構讀入 Go 固定長度陣列：Go 讀較短的陣列時其餘元素是零值、較長的多餘元素丟棄，
/// `null` 不改動（零值）。
fn de_array_via<'de, T, M, const N: usize, D>(d: D) -> std::result::Result<[T; N], D::Error>
where
    T: From<M> + Default + Copy,
    M: Deserialize<'de>,
    D: Deserializer<'de>,
{
    let mut out = [T::default(); N];
    for (slot, m) in out.iter_mut().zip(Option::<Vec<M>>::deserialize(d)?.unwrap_or_default()) {
        *slot = T::from(m);
    }
    Ok(out)
}

/// Go `[N]byte` 欄位的讀取：同 `de_array_via` 的長度規則。
fn fixed_bytes<const N: usize>(raw: &[u8]) -> [u8; N] {
    let mut out = [0u8; N];
    for (slot, b) in out.iter_mut().zip(raw) {
        *slot = *b;
    }
    out
}

/// `Spell` 的 JSON 形式（Go 標籤 `id`…`target`，`Name` 沒有標籤）。
#[derive(Default, Serialize, Deserialize)]
#[serde(default)]
struct SpellJson {
    id: i64,
    dmg: i64,
    hit: i64,
    dist: i64,
    range: i64,
    mp: i64,
    target: i64,
    #[serde(rename = "Name", deserialize_with = "null_as_default")]
    name: String,
}

impl From<&Spell> for SpellJson {
    fn from(v: &Spell) -> Self {
        Self {
            id: v.id,
            dmg: v.dmg,
            hit: v.hit,
            dist: v.dist,
            range: v.range,
            mp: v.mp,
            target: v.target,
            name: v.name.clone(),
        }
    }
}

impl From<SpellJson> for Spell {
    fn from(v: SpellJson) -> Self {
        Self {
            id: v.id,
            dmg: v.dmg,
            hit: v.hit,
            dist: v.dist,
            range: v.range,
            mp: v.mp,
            target: v.target,
            name: v.name,
        }
    }
}

/// `Treasure` 的 JSON 形式（Go 沒有標籤）。
#[derive(Default, Serialize, Deserialize)]
#[serde(default, rename_all = "PascalCase")]
struct TreasureJson {
    slot: i64,
    #[serde(deserialize_with = "null_as_default")]
    kind: String,
    native_type: u8,
    value: i64,
    hidden: bool,
}

impl From<&Treasure> for TreasureJson {
    fn from(v: &Treasure) -> Self {
        Self { slot: v.slot, kind: v.kind.clone(), native_type: v.native_type, value: v.value, hidden: v.hidden }
    }
}

impl From<TreasureJson> for Treasure {
    fn from(v: TreasureJson) -> Self {
        Self { slot: v.slot, kind: v.kind, native_type: v.native_type, value: v.value, hidden: v.hidden }
    }
}

/// `NativeChestControl` 的 JSON 形式（Go 沒有標籤）。
#[derive(Default, Serialize, Deserialize)]
#[serde(default, rename_all = "PascalCase")]
struct ChestJson {
    raw_type: u8,
    value: u16,
}

impl From<&NativeChestControl> for ChestJson {
    fn from(v: &NativeChestControl) -> Self {
        Self { raw_type: v.raw_type, value: v.value }
    }
}

impl From<ChestJson> for NativeChestControl {
    fn from(v: ChestJson) -> Self {
        Self { raw_type: v.raw_type, value: v.value }
    }
}

/// `NativeFieldUnitControl` 的 JSON 形式：Go `Raw [0x1a]byte` 是數字陣列。
#[derive(Default, Serialize, Deserialize)]
#[serde(default, rename_all = "PascalCase")]
struct FieldUnitJson {
    #[serde(deserialize_with = "null_as_default")]
    raw: Vec<u8>,
}

impl From<&NativeFieldUnitControl> for FieldUnitJson {
    fn from(v: &NativeFieldUnitControl) -> Self {
        Self { raw: v.raw.to_vec() }
    }
}

impl From<FieldUnitJson> for NativeFieldUnitControl {
    fn from(v: FieldUnitJson) -> Self {
        Self { raw: fixed_bytes(&v.raw) }
    }
}

/// `NativeRuntimeRecordState` 的 JSON 形式：Go `Raw [0x50]byte` 是數字陣列。
#[derive(Default, Serialize, Deserialize)]
#[serde(default, rename_all = "PascalCase")]
struct RuntimeRecordJson {
    #[serde(deserialize_with = "null_as_default")]
    raw: Vec<u8>,
    selector_key: u8,
    selector_slot: u8,
}

impl From<&NativeRuntimeRecordState> for RuntimeRecordJson {
    fn from(v: &NativeRuntimeRecordState) -> Self {
        Self { raw: v.raw.to_vec(), selector_key: v.selector_key, selector_slot: v.selector_slot }
    }
}

impl From<RuntimeRecordJson> for NativeRuntimeRecordState {
    fn from(v: RuntimeRecordJson) -> Self {
        Self { raw: fixed_bytes(&v.raw), selector_key: v.selector_key, selector_slot: v.selector_slot }
    }
}

/// `NativeMapHUDRuntimeState` 的 JSON 形式（Go 沒有標籤）。
#[derive(Default, Serialize, Deserialize)]
#[serde(default, rename_all = "PascalCase")]
struct HudJson {
    display_gate_a: u8,
    display_gate_b: u8,
    #[serde(rename = "AnchorX")]
    anchor_x: i64,
}

impl From<&NativeMapHUDRuntimeState> for HudJson {
    fn from(v: &NativeMapHUDRuntimeState) -> Self {
        Self { display_gate_a: v.display_gate_a, display_gate_b: v.display_gate_b, anchor_x: v.anchor_x }
    }
}

impl From<HudJson> for NativeMapHUDRuntimeState {
    fn from(v: HudJson) -> Self {
        Self { display_gate_a: v.display_gate_a, display_gate_b: v.display_gate_b, anchor_x: v.anchor_x }
    }
}

/// `NativeMapViewState` 的 JSON 形式（Go 沒有標籤）。
#[derive(Default, Serialize, Deserialize)]
#[serde(default, rename_all = "PascalCase")]
struct ViewJson {
    #[serde(rename = "CameraX")]
    camera_x: i64,
    #[serde(rename = "CameraY")]
    camera_y: i64,
    #[serde(rename = "CursorX")]
    cursor_x: i64,
    #[serde(rename = "CursorY")]
    cursor_y: i64,
    #[serde(rename = "VisibleCursorX")]
    visible_cursor_x: i64,
    #[serde(rename = "VisibleCursorY")]
    visible_cursor_y: i64,
}

impl From<&NativeMapViewState> for ViewJson {
    fn from(v: &NativeMapViewState) -> Self {
        Self {
            camera_x: v.camera_x,
            camera_y: v.camera_y,
            cursor_x: v.cursor_x,
            cursor_y: v.cursor_y,
            visible_cursor_x: v.visible_cursor_x,
            visible_cursor_y: v.visible_cursor_y,
        }
    }
}

impl From<ViewJson> for NativeMapViewState {
    fn from(v: ViewJson) -> Self {
        Self {
            camera_x: v.camera_x,
            camera_y: v.camera_y,
            cursor_x: v.cursor_x,
            cursor_y: v.cursor_y,
            visible_cursor_x: v.visible_cursor_x,
            visible_cursor_y: v.visible_cursor_y,
        }
    }
}

#[cfg(test)]
mod snapshot_test;
