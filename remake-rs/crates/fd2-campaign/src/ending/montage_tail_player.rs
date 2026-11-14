//! Go 來源：`remake/internal/ending/montage_tail_player.go`（357 行）。

use super::*;

/// MontageTailPhase describes the source-bound visual bridge for the proven
/// 20-entry 0x2c194 schedule. It implements the reachable header-byte1-zero
/// 0x2939d pair loop but does not claim exact call-time record continuity,
/// sound ownership, DOS timing or player-path E2.
///
/// Go：`montage_tail_player.go:14`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct MontageTailPhase(pub &'static str);

pub const MONTAGE_TAIL_PHASE_INTRO: MontageTailPhase = MontageTailPhase("intro");

pub const MONTAGE_TAIL_PHASE_RECORD0_AUX: MontageTailPhase = MontageTailPhase("record0_aux");

pub const MONTAGE_TAIL_PHASE_RECORD1_AUX: MontageTailPhase = MontageTailPhase("record1_aux");

pub const MONTAGE_TAIL_PHASE_BASE_HOLD: MontageTailPhase = MontageTailPhase("base_hold");

pub const MONTAGE_TAIL_PHASE_OVERLAY: MontageTailPhase = MontageTailPhase("fdother_58_overlay");

pub const MONTAGE_TAIL_PHASE_FINAL: MontageTailPhase = MontageTailPhase("final");

pub const MONTAGE_TAIL_PHASE_COMPLETED: MontageTailPhase = MontageTailPhase("completed");

pub(crate) const NATIVE_MONTAGE_TAIL_INTRO_WAIT_TICKS: i64 = 0x50;

/// Go：`montage_tail_player.go:28`
pub(crate) static NATIVE_MONTAGE_TAIL_HORIZONTAL_OFFSETS: [i64; 6] = [0, 4, 9, 14, 18, 14];

/// Go：`montage_tail_player.go:29`
pub(crate) static NATIVE_MONTAGE_TAIL_VERTICAL_OFFSETS: [i64; 6] = [0, 2, 4, 6, 8, 10];

/// MontageTailPlayer turns the source-proven resource schedule into a bounded,
/// deterministic indexed presentation. Campaign admission is owned by the
/// source-bound E1 caller; this player owns no campaign, party, battle or save
/// state and does not claim exact native 0x28a6c rendering.
///
/// Go：`montage_tail_player.go:36`
#[derive(Debug, Clone, Default)]
pub struct MontageTailPlayer {
    pub tail: MontageTail,
    pub assets: MontageTailAssets,
    pub visual_sets: Vec<MontageTailVisualSet>,
    pub entries: Vec<MontageTailEntry>,
    pub compositor: Option<SharedCompositor>,
    pub phase: MontageTailPhase,
    pub segment: i64,
    pub frame: i64,
    pub inner: i64,
    pub base_frame: i64,
    pub base_inner: i64,
    pub effect_step: i64,
    /// SoundMarker preserves raw frame +5 for tracing. The player deliberately
    /// does not emit audio while the two native sound-owner globals are unknown.
    pub sound_marker: u8,
    pub delay_ticks: i64,
}

/// NewMontageTailPlayer validates every source asset before exposing any
/// playable state. This all-or-nothing admission prevents a missing late entry
/// from producing a partially plausible ending.
///
/// Go：`montage_tail_player.go:58`，`func NewMontageTailPlayer(tail MontageTail, assets MontageTailAssets, sets []MontageTailVisualSet, compositor *IndexedCompositor) (*MontageTailPlayer, error)`
pub fn new_montage_tail_player(
    tail: &MontageTail,
    assets: &MontageTailAssets,
    sets: &[MontageTailVisualSet],
    compositor: Option<SharedCompositor>,
) -> Result<MontageTailPlayer> {
    todo!("ending/montage_tail_player.go:NewMontageTailPlayer")
}

/// Go：`montage_tail_player.go:112`，`func validateMontageTailPlayerAnimation(animation *figani.Animation) error`
pub(crate) fn validate_montage_tail_player_animation(animation: Option<&figani::Animation>) -> Result<()> {
    todo!("ending/montage_tail_player.go:validateMontageTailPlayerAnimation")
}

impl MontageTailPlayer {
    /// Ready reports that all 20 source transactions completed and the stable
    /// FDOTHER#59 frame was presented. Further Step calls are rejected so the
    /// caller can hold that image indefinitely.
    ///
    /// Go：`montage_tail_player.go:127`，`func (p *MontageTailPlayer) Ready() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn ready(&self) -> bool {
        self.phase == MONTAGE_TAIL_PHASE_COMPLETED
    }

    /// Step presents one source-bound native inner iteration and exposes the next
    /// raw-tick delay. Each auxiliary descriptor +6 owns that many presentations;
    /// the paired base scheduler advances after every one. Zero-delay descriptors
    /// publish no frame. The 20/78 waits around FDOTHER#58 remain caller-owned.
    ///
    /// Go：`montage_tail_player.go:135`，`func (p *MontageTailPlayer) Step() error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn step(&mut self) -> Result<()> {
        todo!("ending/montage_tail_player.go:MontageTailPlayer.Step")
    }

    /// Go：`montage_tail_player.go:216`，`func (p *MontageTailPlayer) resetPairState()`
    pub(crate) fn reset_pair_state(&mut self) {
        todo!("ending/montage_tail_player.go:MontageTailPlayer.resetPairState")
    }

    /// Go：`montage_tail_player.go:225`，`func (p *MontageTailPlayer) currentSet() (*MontageTailVisualSet, error)`
    pub(crate) fn current_set(&self) -> Result<&MontageTailVisualSet> {
        todo!("ending/montage_tail_player.go:MontageTailPlayer.currentSet")
    }

    /// Go：`montage_tail_player.go:232`，`func (p *MontageTailPlayer) stepNativePair(set MontageTailVisualSet, auxiliary, base *figani.Animation, record0, terminal bool) (presented, done bool, err error)`
    /// Go 具名回傳值依序為：presented、done、err。
    pub(crate) fn step_native_pair(
        &mut self,
        set: &MontageTailVisualSet,
        auxiliary: &figani::Animation,
        base: &figani::Animation,
        record0: bool,
        terminal: bool,
    ) -> Result<(bool, bool)> {
        todo!("ending/montage_tail_player.go:MontageTailPlayer.stepNativePair")
    }

    /// Go：`montage_tail_player.go:286`，`func (p *MontageTailPlayer) advanceBaseScheduler(base *figani.Animation)`
    pub(crate) fn advance_base_scheduler(&mut self, base: &figani::Animation) {
        todo!("ending/montage_tail_player.go:MontageTailPlayer.advanceBaseScheduler")
    }

    /// renderSourceBoundPair reproduces the reachable header-byte1-zero branch of
    /// the two 0x2939d calls. The native 400-stride staging margin is represented
    /// by clipped translated blits on the final 320-wide indexed surface.
    ///
    /// Go：`montage_tail_player.go:300`，`func (p *MontageTailPlayer) renderSourceBoundPair(set MontageTailVisualSet, auxiliary, base figani.Frame, record0 bool, effectStep int, opaqueFill *byte) error`
    pub(crate) fn render_source_bound_pair(
        &mut self,
        set: &MontageTailVisualSet,
        auxiliary: &figani::Frame,
        base: &figani::Frame,
        record0: bool,
        effect_step: i64,
        opaque_fill: &u8,
    ) -> Result<()> {
        todo!("ending/montage_tail_player.go:MontageTailPlayer.renderSourceBoundPair")
    }
}
