//! Go 來源：`remake/internal/ending/montage_cycle.go`（403 行）。

use super::*;

/// MontageCycleAssets contains only the original indexed inputs consumed by
/// 0x2c405's 0x2c548 party cycle.  Nothing here is a PNG or a guessed UI
/// replacement; missing or malformed source assets reject the cycle.
///
/// Go：`montage_cycle.go:16`
#[derive(Debug, Clone, Default)]
pub struct MontageCycleAssets {
    pub backdrop: fdother::Frame,
    pub tai003: fdother::Frame,
    pub primary: BTreeMap<i64, figani::Animation>,
    pub secondary: BTreeMap<i64, figani::Animation>,
    pub grid: Vec<u8>,
    pub portraits: BTreeMap<i64, Vec<dato::Frame>>,
    pub current: Option<fdtxt::Strings>,
    pub permanent: Option<fdtxt::Strings>,
    pub font: Option<fdtxt::Font>,
}

/// MontageArchivePaths identifies the remaining player-provided archives and
/// separated asset roots used by the native cycle. Originals remain read-only.
///
/// Go：`montage_cycle.go:30`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MontageArchivePaths {
    pub surface_root: String,
    pub item_panel_root: String,
    pub animation_root: String,
    pub portrait_root: String,
    pub text_root: String,
    pub font_root: String,
}

/// LoadMontageCycleAssets decodes the exact resources named by the native
/// call-sites: FDOTHER#56, TAI#3, FDOTHER#5, FIGANI group*3/{+1}, DATO unit
/// +7, FDTXT_031/FDTXT_000 and the FDOTHER#4 font.  It deliberately takes the
/// raw unit records from the caller so no identity or slot meaning is guessed.
///
/// Go：`montage_cycle.go:43`，`func LoadMontageCycleAssets(montage Montage, paths MontageArchivePaths, units [][]byte) (MontageCycleAssets, error)`
pub fn load_montage_cycle_assets(
    montage: &Montage,
    paths: &MontageArchivePaths,
    units: &[Vec<u8>],
) -> Result<MontageCycleAssets> {
    todo!("ending/montage_cycle.go:LoadMontageCycleAssets")
}

/// MontageCyclePhase is the directly recovered first-party schedule.  The
/// final palette fade is kept as a distinct phase because its native helper
/// has a different delay unit from the BIOS-tick waits in the party loops.
///
/// Go：`montage_cycle.go:111`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub struct MontageCyclePhase(pub &'static str);

pub const MONTAGE_PHASE_FIGURE_FADE: MontageCyclePhase = MontageCyclePhase("figure_fade");

pub const MONTAGE_PHASE_SECONDARY: MontageCyclePhase = MontageCyclePhase("secondary_intro");

pub const MONTAGE_PHASE_PRIMARY: MontageCyclePhase = MontageCyclePhase("primary_animation");

pub const MONTAGE_PHASE_PORTRAIT: MontageCyclePhase = MontageCyclePhase("portrait_text");

pub const MONTAGE_PHASE_FADE_OUT: MontageCyclePhase = MontageCyclePhase("palette_fade_out");

pub const MONTAGE_PHASE_COMPLETED: MontageCyclePhase = MontageCyclePhase("completed");

/// MontageCycle is a deterministic indexed executor for the verified
/// 0x2c548 schedule.  Step performs one native render iteration; the caller
/// owns the returned delay units and the raw random byte used by 0x4e893.
/// ObserveInputChange is deliberately a separate raw event hook: 0x2c950 only
/// observes a changed input-buffer condition during portrait presentation; it
/// does not decode a particular key in this branch.
///
/// Go：`montage_cycle.go:128`
#[derive(Debug, Clone, Default)]
pub struct MontageCycle {
    pub montage: Montage,
    pub assets: MontageCycleAssets,
    pub units: Vec<Vec<u8>>,
    pub plans: Vec<PartyCyclePlan>,
    pub compositor: Option<SharedCompositor>,
    pub plan_index: i64,
    pub phase: MontageCyclePhase,
    pub fade_index: i64,
    pub secondary: Option<figani::Animation>,
    pub primary: Option<figani::Animation>,
    pub secondary_sm: figani::NativeScheduler,
    pub primary_idx: i64,
    pub portrait: i64,
    pub portrait_max: i64,
    pub portrait_sm: MontagePortraitState,
    pub fade_out: i64,
    pub delay_ticks: i64,
    pub delay_ms: i64,
    pub(crate) skip_to_final: bool,
    pub(crate) portrait_boundary_pending: bool,
}

/// NewMontageCycle validates all source provenance before exposing an
/// executable renderer.  In particular, the caller must provide the palette
/// baseline captured from a native indexed presentation; an all-zero default
/// is not accepted as a substitute.
///
/// Go：`montage_cycle.go:155`，`func NewMontageCycle(m Montage, assets MontageCycleAssets, units [][]byte, groups []byte, compositor *IndexedCompositor) (*MontageCycle, error)`
/// Rust：Go 把 `units` 切片直接存進 `MontageCycle.Units`，之後只讀（`montage_cycle.go` 238、251、334 行）；
/// Rust 複製一份給結構擁有，所以收 `&[Vec<u8>]`（產生器因「存進之後會被改寫的結構」誤判成可變）。
pub fn new_montage_cycle(
    m: &Montage,
    assets: MontageCycleAssets,
    units: &[Vec<u8>],
    groups: &[u8],
    compositor: Option<SharedCompositor>,
) -> Result<MontageCycle> {
    todo!("ending/montage_cycle.go:NewMontageCycle")
}

impl MontageCycle {
    /// Go：`montage_cycle.go:197`，`func (p *MontageCycle) preparePlan() error`
    pub(crate) fn prepare_plan(&mut self) -> Result<()> {
        todo!("ending/montage_cycle.go:MontageCycle.preparePlan")
    }

    /// Go：`montage_cycle.go:214`，`func (p *MontageCycle) renderBackdrop() error`
    pub(crate) fn render_backdrop(&mut self) -> Result<()> {
        todo!("ending/montage_cycle.go:MontageCycle.renderBackdrop")
    }

    /// Step advances exactly one recovered rendering iteration.  randomByte is
    /// supplied by the caller because the original obtains it from 0x4e893; the
    /// executor never invents a replacement RNG.  The delay fields describe the
    /// native wait following the produced frame.
    ///
    /// Go：`montage_cycle.go:226`，`func (p *MontageCycle) Step(randomByte byte) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn step(&mut self, random_byte: u8) -> Result<()> {
        todo!("ending/montage_cycle.go:MontageCycle.Step")
    }

    /// NeedsRandomByte reports the sole recovered call site where 0x2c99c uses
    /// 0x4e893: portrait countdown reset.  Callers must not advance the native RNG
    /// on every rendered frame.
    ///
    /// Go：`montage_cycle.go:375`，`func (p *MontageCycle) NeedsRandomByte() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn needs_random_byte(&self) -> bool {
        todo!("ending/montage_cycle.go:MontageCycle.NeedsRandomByte")
    }

    /// ObserveInputChange receives the raw condition observed at 0x2c950, not a
    /// named keyboard mapping.  The native branch is only polled while a portrait
    /// is being presented and, except for the final loop, causes middle portraits
    /// to be skipped after the current one finishes.
    ///
    /// Go：`montage_cycle.go:383`，`func (p *MontageCycle) ObserveInputChange() bool`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn observe_input_change(&mut self) -> bool {
        todo!("ending/montage_cycle.go:MontageCycle.ObserveInputChange")
    }

    /// Go：`montage_cycle.go:393`，`func (p *MontageCycle) renderFIGANI(frame figani.Frame) error`
    pub(crate) fn render_figani(&mut self, frame: &figani::Frame) -> Result<()> {
        todo!("ending/montage_cycle.go:MontageCycle.renderFIGANI")
    }

    /// Ready reports only completion of the standalone indexed cycle.  It does
    /// not grant campaign transition or generic input semantics.
    ///
    /// Go：`montage_cycle.go:403`，`func (p MontageCycle) Ready() bool`
    pub fn ready(&self) -> bool {
        self.phase == MONTAGE_PHASE_COMPLETED
    }
}
