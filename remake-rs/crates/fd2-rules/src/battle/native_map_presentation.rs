//! Go 來源：`remake/internal/battle/native_map_presentation.go`（289 行）。

use super::*;

/// NativeMapPresentationState is the battle-local runtime record subset used
/// by 0x127e0. It is intentionally independent of normalized X/Y/Dir/OffX/OffY:
/// native grid movement keeps X/Y on the source cell while Motion advances
/// 1..6, then commits the destination and clears Motion at tick seven.
///
/// Go：`native_map_presentation.go:13`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeMapPresentationState {
    pub x: u8,
    pub y: u8,
    pub pose: u8,
    pub motion: u8,
}

/// NativeMapHUDRuntimeState is the raw persistent subset consumed by 0x1acf3.
/// Display bytes retain byte identity; only their zero/nonzero tests are known.
///
/// Go：`native_map_presentation.go:21`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeMapHUDRuntimeState {
    pub display_gate_a: u8,
    pub display_gate_b: u8,
    pub anchor_x: i64,
}

/// NativeMapHUDPersistentState separates the two values whose lifetimes cross
/// battle nodes. DisplayGateA is also present in native chapter saves and the
/// current-battle snapshot; AnchorX is process-local and is never serialized by
/// the original. The provenance bits prevent a default Go zero from becoming a
/// fabricated raw value.
///
/// Go：`native_map_presentation.go:31`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub struct NativeMapHUDPersistentState {
    pub display_gate_a: u8,
    pub anchor_x: i64,
    pub has_display_gate_a: bool,
    pub has_anchor_x: bool,
}

/// InitialNativeMapHUDPersistentState returns the two data-image seeds at
/// [0x51AAB] and [0x51A0C]. Gate B is deliberately absent: it is a transient
/// redraw gate with its own controller writers.
///
/// Go：`native_map_presentation.go:41`，`func InitialNativeMapHUDPersistentState() NativeMapHUDPersistentState`
pub fn initial_native_map_hud_persistent_state() -> NativeMapHUDPersistentState {
    todo!("battle/native_map_presentation.go:InitialNativeMapHUDPersistentState")
}

impl NativeMapHUDPersistentState {
    /// CaptureNativeMapHUD records only the two cross-node values from a complete
    /// runtime HUD state. It never carries the transient display gate B forward.
    ///
    /// Go：`native_map_presentation.go:49`，`func (p *NativeMapHUDPersistentState) CaptureNativeMapHUD(runtime NativeMapHUDRuntimeState) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn capture_native_map_hud(&mut self, runtime: NativeMapHUDRuntimeState) -> bool {
        todo!("battle/native_map_presentation.go:NativeMapHUDPersistentState.CaptureNativeMapHUD")
    }

    /// RestoreSavedGateA applies the only HUD byte stored by both native save
    /// formats. AnchorX intentionally remains process-local.
    ///
    /// Go：`native_map_presentation.go:65`，`func (p *NativeMapHUDPersistentState) RestoreSavedGateA(gateA byte) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn restore_saved_gate_a(&mut self, gate_a: u8) -> bool {
        todo!("battle/native_map_presentation.go:NativeMapHUDPersistentState.RestoreSavedGateA")
    }

    /// MaterializeRuntime combines the persistent pair with an explicitly sourced
    /// controller gate B. Missing provenance fails closed.
    ///
    /// Go：`native_map_presentation.go:76`，`func (p NativeMapHUDPersistentState) MaterializeRuntime(gateB byte) (NativeMapHUDRuntimeState, bool)`
    pub fn materialize_runtime(self, gate_b: u8) -> (NativeMapHUDRuntimeState, bool) {
        todo!("battle/native_map_presentation.go:NativeMapHUDPersistentState.MaterializeRuntime")
    }
}
impl State {
    /// Go：`native_map_presentation.go:88`，`func (s *State) MaterializeNativeMapHUDState(gateA, gateB byte, anchorX int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn materialize_native_map_hud_state(&mut self, gate_a: u8, gate_b: u8, anchor_x: i64) -> bool {
        todo!("battle/native_map_presentation.go:State.MaterializeNativeMapHUDState")
    }

    /// AdvanceNativeMapHUDAnchor applies the only two proven writes in 0x1acf3.
    /// rawVisibleX/Y are [0x53ab9]/[0x53abd], not dialogue-box dimensions.
    ///
    /// Go：`native_map_presentation.go:104`，`func (s *State) AdvanceNativeMapHUDAnchor(rawVisibleX, rawVisibleY int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_map_hud_anchor(&mut self, raw_visible_x: i64, raw_visible_y: i64) -> bool {
        todo!("battle/native_map_presentation.go:State.AdvanceNativeMapHUDAnchor")
    }
}

/// NativeMapFrameRoster is the atomically materialized roster subset consumed
/// by the steady unit and foreground layers. Cycles are the battle-session
/// globals selected by those same entries.
///
/// Go：`native_map_presentation.go:133`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct NativeMapFrameRoster {
    pub units: Vec<fdicon::NativeUnitLayerEntry>,
    pub foreground: Vec<fdicon::NativeForegroundLayerEntry>,
    pub cycles: fdicon::NativeMapSpriteCycleState,
    pub terrain_phase: i64,
    pub terrain_flip: i64,
    pub unit_pixel_shift: i64,
}

impl Unit {
    /// MaterializeNativeMapPresentation reproduces the constructor's initial
    /// +0/+1 coordinates and +3/+4=0/0. Callers must establish the native selector
    /// slot separately; this method performs no identity/Fig fallback.
    ///
    /// Go：`native_map_presentation.go:145`，`func (u *Unit) MaterializeNativeMapPresentation() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn materialize_native_map_presentation(&mut self) -> Result<()> {
        todo!("battle/native_map_presentation.go:Unit.MaterializeNativeMapPresentation")
    }

    /// SetMapPose updates the normalized direction and, when materialized, raw
    /// unit+3. It never fabricates native provenance for a legacy unit.
    ///
    /// Go：`native_map_presentation.go:158`，`func (u *Unit) SetMapPose(pose int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn set_map_pose(&mut self, pose: i64) -> bool {
        todo!("battle/native_map_presentation.go:Unit.SetMapPose")
    }

    /// SetMapPlacement updates an out-of-motion placement writer. Materialized
    /// native state receives the same byte coordinates and a zero motion offset;
    /// legacy units retain only the normalized projection.
    ///
    /// Go：`native_map_presentation.go:172`，`func (u *Unit) SetMapPlacement(x, y, pose int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn set_map_placement(&mut self, x: i64, y: i64, pose: i64) -> bool {
        todo!("battle/native_map_presentation.go:Unit.SetMapPlacement")
    }

    /// SetNativeMapCoordinatesRaw mirrors a direct write to runtime bytes +0/+1.
    /// Unlike SetMapPlacement it deliberately preserves +3 pose and +4 motion:
    /// hard-coded handlers may update only the two coordinate bytes, and clearing
    /// adjacent presentation state would invent a write that is absent in the EXE.
    ///
    /// Go：`native_map_presentation.go:190`，`func (u *Unit) SetNativeMapCoordinatesRaw(x, y int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn set_native_map_coordinates_raw(&mut self, x: i64, y: i64) -> bool {
        todo!("battle/native_map_presentation.go:Unit.SetNativeMapCoordinatesRaw")
    }

    /// SetNativeMapGridMotion writes one native in-cell step. X/Y deliberately
    /// remain on the source cell. motion is the exact unit+4 range 1..6.
    ///
    /// Go：`native_map_presentation.go:202`，`func (u *Unit) SetNativeMapGridMotion(pose, motion int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn set_native_map_grid_motion(&mut self, pose: i64, motion: i64) -> bool {
        todo!("battle/native_map_presentation.go:Unit.SetNativeMapGridMotion")
    }

    /// FinishNativeMapGridStep reproduces the seventh-tick commit: destination
    /// bytes replace +0/+1, +4 becomes zero, and +3 retains the movement pose.
    ///
    /// Go：`native_map_presentation.go:214`，`func (u *Unit) FinishNativeMapGridStep(pose, x, y int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn finish_native_map_grid_step(&mut self, pose: i64, x: i64, y: i64) -> bool {
        todo!("battle/native_map_presentation.go:Unit.FinishNativeMapGridStep")
    }

    /// NativeUnitLayerEntry returns the exact raw subset required by 0x127e0 and
    /// its preceding inactive gate. Missing selector/record/presentation
    /// provenance fails closed rather than projecting normalized fields.
    ///
    /// Go：`native_map_presentation.go:230`，`func (u *Unit) NativeUnitLayerEntry() (fdicon.NativeUnitLayerEntry, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_unit_layer_entry(&self) -> (fdicon::NativeUnitLayerEntry, bool) {
        todo!("battle/native_map_presentation.go:Unit.NativeUnitLayerEntry")
    }

    /// NativeForegroundLayerEntry returns the exact raw subset consumed by
    /// 0x129ec. BattleFig's legacy Fig fallback is rejected: unit+7, race and class
    /// must each carry explicit native provenance.
    ///
    /// Go：`native_map_presentation.go:246`，`func (u *Unit) NativeForegroundLayerEntry() (fdicon.NativeForegroundLayerEntry, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_foreground_layer_entry(&self) -> (fdicon::NativeForegroundLayerEntry, bool) {
        todo!("battle/native_map_presentation.go:Unit.NativeForegroundLayerEntry")
    }
}
impl State {
    /// NativeMapFrameRoster materializes one complete compositor input snapshot.
    /// The native loop has no nil or guessed records, so any missing provenance
    /// rejects the entire roster instead of producing a partially native frame.
    ///
    /// Go：`native_map_presentation.go:263`，`func (s *State) NativeMapFrameRoster() (NativeMapFrameRoster, error)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_map_frame_roster(&self) -> Result<NativeMapFrameRoster> {
        todo!("battle/native_map_presentation.go:State.NativeMapFrameRoster")
    }
}
