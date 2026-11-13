//! Go 來源：`remake/internal/battle/native_map_presentation.go`（289 行）。

use super::*;

/// NativeMapPresentationState is the battle-local runtime record subset used
/// by 0x127e0. It is intentionally independent of normalized X/Y/Dir/OffX/OffY:
/// native grid movement keeps X/Y on the source cell while Motion advances
/// 1..6, then commits the destination and clears Motion at tick seven.
///
/// Go：`native_map_presentation.go:13`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(default)]
pub struct NativeMapPresentationState {
    #[serde(rename = "X")]
    pub x: u8,
    #[serde(rename = "Y")]
    pub y: u8,
    #[serde(rename = "Pose")]
    pub pose: u8,
    #[serde(rename = "Motion")]
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
    NativeMapHUDPersistentState { display_gate_a: 1, anchor_x: 1, has_display_gate_a: true, has_anchor_x: true }
}

impl NativeMapHUDPersistentState {
    /// CaptureNativeMapHUD records only the two cross-node values from a complete
    /// runtime HUD state. It never carries the transient display gate B forward.
    ///
    /// Go：`native_map_presentation.go:49`，`func (p *NativeMapHUDPersistentState) CaptureNativeMapHUD(runtime NativeMapHUDRuntimeState) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn capture_native_map_hud(&mut self, runtime: NativeMapHUDRuntimeState) -> bool {
        if fdicon::native_map_hud_layout_for(runtime.anchor_x, fdicon::NATIVE_MAP_STRIDE).is_err() {
            return false;
        }
        self.display_gate_a = runtime.display_gate_a;
        self.anchor_x = runtime.anchor_x;
        self.has_display_gate_a = true;
        self.has_anchor_x = true;
        true
    }

    /// RestoreSavedGateA applies the only HUD byte stored by both native save
    /// formats. AnchorX intentionally remains process-local.
    ///
    /// Go：`native_map_presentation.go:65`，`func (p *NativeMapHUDPersistentState) RestoreSavedGateA(gateA byte) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn restore_saved_gate_a(&mut self, gate_a: u8) -> bool {
        self.display_gate_a = gate_a;
        self.has_display_gate_a = true;
        true
    }

    /// MaterializeRuntime combines the persistent pair with an explicitly sourced
    /// controller gate B. Missing provenance fails closed.
    ///
    /// Go：`native_map_presentation.go:76`，`func (p NativeMapHUDPersistentState) MaterializeRuntime(gateB byte) (NativeMapHUDRuntimeState, bool)`
    pub fn materialize_runtime(self, gate_b: u8) -> (NativeMapHUDRuntimeState, bool) {
        if !self.has_display_gate_a || !self.has_anchor_x {
            return (NativeMapHUDRuntimeState::default(), false);
        }
        if fdicon::native_map_hud_layout_for(self.anchor_x, fdicon::NATIVE_MAP_STRIDE).is_err() {
            return (NativeMapHUDRuntimeState::default(), false);
        }
        (
            NativeMapHUDRuntimeState {
                display_gate_a: self.display_gate_a,
                display_gate_b: gate_b,
                anchor_x: self.anchor_x,
            },
            true,
        )
    }
}
impl State {
    /// Go：`native_map_presentation.go:88`，`func (s *State) MaterializeNativeMapHUDState(gateA, gateB byte, anchorX int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn materialize_native_map_hud_state(&mut self, gate_a: u8, gate_b: u8, anchor_x: i64) -> bool {
        if fdicon::native_map_hud_layout_for(anchor_x, fdicon::NATIVE_MAP_STRIDE).is_err() {
            return false;
        }
        self.native_map_hud_state =
            NativeMapHUDRuntimeState { display_gate_a: gate_a, display_gate_b: gate_b, anchor_x };
        self.has_native_map_hud_state = true;
        true
    }

    /// AdvanceNativeMapHUDAnchor applies the only two proven writes in 0x1acf3.
    /// rawVisibleX/Y are [0x53ab9]/[0x53abd], not dialogue-box dimensions.
    ///
    /// Go：`native_map_presentation.go:104`，`func (s *State) AdvanceNativeMapHUDAnchor(rawVisibleX, rawVisibleY int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn advance_native_map_hud_anchor(&mut self, raw_visible_x: i64, raw_visible_y: i64) -> bool {
        if !self.has_native_map_hud_state {
            return false;
        }
        // 0x1ACF3 先查兩個顯示閘門 [0x51AAB]／[0x51AAC]，任一為 0 就直接返回，0x1AD2A 的
        // anchor 分支根本不會跑。移動確認後游標走回單位（0x18A26→0x12CEA，每格 0x11CAC
        // 重繪）時閘門是關的，所以原版不會因為那幾格落在左下角而把小窗翻到右邊
        //（第六章 r3 seq 817：悠妮從 (7,23) 走到 (8,23)，可見游標 (2,6) 沒有翻）。
        if self.native_map_hud_state.display_gate_a == 0 || self.native_map_hud_state.display_gate_b == 0 {
            return false;
        }
        let mut next = self.native_map_hud_state.anchor_x;
        if raw_visible_y > 5 {
            if raw_visible_x < 3 {
                next = 0xf2;
            } else if raw_visible_x > 9 {
                next = 1;
            }
        }
        if fdicon::native_map_hud_layout_for(next, fdicon::NATIVE_MAP_STRIDE).is_err() {
            return false;
        }
        self.native_map_hud_state.anchor_x = next;
        true
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
        if !(0..=0xff).contains(&self.x) || !(0..=0xff).contains(&self.y) {
            return Err(battle_err!("battle: native map presentation coordinate outside byte range"));
        }
        self.native_map_presentation =
            NativeMapPresentationState { x: self.x as u8, y: self.y as u8, ..Default::default() };
        self.has_native_map_presentation = true;
        self.dir = 0;
        (self.off_x, self.off_y) = (0.0, 0.0);
        Ok(())
    }

    /// SetMapPose updates the normalized direction and, when materialized, raw
    /// unit+3. It never fabricates native provenance for a legacy unit.
    ///
    /// Go：`native_map_presentation.go:158`，`func (u *Unit) SetMapPose(pose int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn set_map_pose(&mut self, pose: i64) -> bool {
        if !(0..=3).contains(&pose) {
            return false;
        }
        self.dir = pose;
        if self.has_native_map_presentation {
            self.native_map_presentation.pose = pose as u8;
        }
        true
    }

    /// SetMapPlacement updates an out-of-motion placement writer. Materialized
    /// native state receives the same byte coordinates and a zero motion offset;
    /// legacy units retain only the normalized projection.
    ///
    /// Go：`native_map_presentation.go:172`，`func (u *Unit) SetMapPlacement(x, y, pose int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn set_map_placement(&mut self, x: i64, y: i64, pose: i64) -> bool {
        if !(0..=0xff).contains(&x) || !(0..=0xff).contains(&y) || !(0..=3).contains(&pose) {
            return false;
        }
        (self.x, self.y, self.dir) = (x, y, pose);
        (self.off_x, self.off_y) = (0.0, 0.0);
        if self.has_native_map_presentation {
            self.native_map_presentation =
                NativeMapPresentationState { x: x as u8, y: y as u8, pose: pose as u8, motion: 0 };
        }
        true
    }

    /// SetNativeMapCoordinatesRaw mirrors a direct write to runtime bytes +0/+1.
    /// Unlike SetMapPlacement it deliberately preserves +3 pose and +4 motion:
    /// hard-coded handlers may update only the two coordinate bytes, and clearing
    /// adjacent presentation state would invent a write that is absent in the EXE.
    ///
    /// Go：`native_map_presentation.go:190`，`func (u *Unit) SetNativeMapCoordinatesRaw(x, y int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn set_native_map_coordinates_raw(&mut self, x: i64, y: i64) -> bool {
        if !self.has_native_map_presentation || !(0..=0xff).contains(&x) || !(0..=0xff).contains(&y) {
            return false;
        }
        (self.x, self.y) = (x, y);
        self.native_map_presentation.x = x as u8;
        self.native_map_presentation.y = y as u8;
        true
    }

    /// SetNativeMapGridMotion writes one native in-cell step. X/Y deliberately
    /// remain on the source cell. motion is the exact unit+4 range 1..6.
    ///
    /// Go：`native_map_presentation.go:202`，`func (u *Unit) SetNativeMapGridMotion(pose, motion int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn set_native_map_grid_motion(&mut self, pose: i64, motion: i64) -> bool {
        if !self.has_native_map_presentation || !(0..=3).contains(&pose) || !(1..=6).contains(&motion) {
            return false;
        }
        self.dir = pose;
        self.native_map_presentation.pose = pose as u8;
        self.native_map_presentation.motion = motion as u8;
        true
    }

    /// FinishNativeMapGridStep reproduces the seventh-tick commit: destination
    /// bytes replace +0/+1, +4 becomes zero, and +3 retains the movement pose.
    ///
    /// Go：`native_map_presentation.go:214`，`func (u *Unit) FinishNativeMapGridStep(pose, x, y int) bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn finish_native_map_grid_step(&mut self, pose: i64, x: i64, y: i64) -> bool {
        if !self.has_native_map_presentation
            || !(0..=3).contains(&pose)
            || !(0..=0xff).contains(&x)
            || !(0..=0xff).contains(&y)
        {
            return false;
        }
        (self.x, self.y, self.dir) = (x, y, pose);
        (self.off_x, self.off_y) = (0.0, 0.0);
        self.native_map_presentation =
            NativeMapPresentationState { x: x as u8, y: y as u8, pose: pose as u8, motion: 0 };
        true
    }

    /// NativeUnitLayerEntry returns the exact raw subset required by 0x127e0 and
    /// its preceding inactive gate. Missing selector/record/presentation
    /// provenance fails closed rather than projecting normalized fields.
    ///
    /// Go：`native_map_presentation.go:230`，`func (u *Unit) NativeUnitLayerEntry() (fdicon.NativeUnitLayerEntry, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_unit_layer_entry(&self) -> (fdicon::NativeUnitLayerEntry, bool) {
        if !self.has_native_map_presentation || !self.has_map_selector_slot || !self.has_native_record_byte5 {
            return (fdicon::NativeUnitLayerEntry::default(), false);
        }
        let raw = self.native_map_presentation;
        (
            fdicon::NativeUnitLayerEntry {
                x: raw.x as i64,
                y: raw.y as i64,
                slot: self.map_selector_slot,
                pose: raw.pose as i64,
                motion_offset: raw.motion as i64,
                flags: self.native_record_byte5,
                force_base: self.native_transient[4] != 0,
                inactive: self.native_record_byte5 & 1 != 0,
            },
            true,
        )
    }

    /// NativeForegroundLayerEntry returns the exact raw subset consumed by
    /// 0x129ec. BattleFig's legacy Fig fallback is rejected: unit+7, race and class
    /// must each carry explicit native provenance.
    ///
    /// Go：`native_map_presentation.go:246`，`func (u *Unit) NativeForegroundLayerEntry() (fdicon.NativeForegroundLayerEntry, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn native_foreground_layer_entry(&self) -> (fdicon::NativeForegroundLayerEntry, bool) {
        if !self.has_native_map_presentation
            || !self.has_native_record_byte5
            || !self.has_battle_fig
            || !(0..=0xff).contains(&self.battle_fig)
            || !self.has_native_record_race
            || !self.has_native_record_class
        {
            return (fdicon::NativeForegroundLayerEntry::default(), false);
        }
        let raw = self.native_map_presentation;
        (
            fdicon::NativeForegroundLayerEntry {
                x: raw.x as i64,
                y: raw.y as i64,
                pose: raw.pose,
                motion_offset: raw.motion as i64,
                inactive: self.native_record_byte5 & 1 != 0,
                unit7: self.battle_fig as u8,
                race: self.native_record_race,
                class: self.native_record_class,
            },
            true,
        )
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
        if self.native_map_selector_error.is_some()
            || self.native_map_selector_cache.is_none()
            || !self.has_native_map_cycle_state
            || !self.has_native_terrain_phase_state
            || !self.has_native_map_binary_timing_state
        {
            return Err(battle_err!("battle: native map frame state is incomplete"));
        }
        let mut out = NativeMapFrameRoster {
            units: Vec::with_capacity(self.units.len()),
            foreground: Vec::with_capacity(self.units.len()),
            cycles: self.native_map_cycle_state,
            terrain_phase: self.native_terrain_phase_state.phase,
            terrain_flip: self.native_terrain_flip_state.value,
            unit_pixel_shift: self.native_unit_pixel_shift_state.value,
        };
        for (i, u) in self.units.iter().enumerate() {
            let (unit, ok) = u.native_unit_layer_entry();
            if !ok {
                return Err(battle_err!("battle: native map frame unit {i} lacks unit-layer provenance"));
            }
            let (foreground, ok) = u.native_foreground_layer_entry();
            if !ok {
                return Err(battle_err!("battle: native map frame unit {i} lacks foreground provenance"));
            }
            out.units.push(unit);
            out.foreground.push(foreground);
        }
        Ok(out)
    }
}

#[cfg(test)]
mod native_map_presentation_test;
