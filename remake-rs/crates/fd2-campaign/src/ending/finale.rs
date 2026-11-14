//! Go 來源：`remake/internal/ending/finale.go`（223 行）。

use super::*;

/// FinalePhase is an editable, evidence-only transcription of a native finale
/// stage. Its recovered phase-0 adapter still blocks before the next native
/// montage, so it cannot make the finale generally playable.
///
/// Go：`finale.go:14`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FinalePhase {
    /// Go 標籤：`json:"schema_version"`
    pub schema_version: i64,
    /// Go 標籤：`json:"native_handler"`
    pub native_handler: String,
    /// Go 標籤：`json:"status"`
    pub status: String,
    /// Go 標籤：`json:"phase"`
    pub phase: FinalePhaseSpec,
    /// Go 標籤：`json:"gate"`
    pub gate: FinalePhaseGate,
}

/// Go：`finale.go:22`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FinalePhaseSpec {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"source_dat"`
    pub source_dat: String,
    /// Go 標籤：`json:"string_index"`
    pub string_index: i64,
    /// Go 標籤：`json:"script"`
    pub script: String,
    /// Go 標籤：`json:"scene_index"`
    pub scene_index: i64,
    /// Go 標籤：`json:"line"`
    pub line: i64,
    /// Go 標籤：`json:"count"`
    pub count: i64,
    /// Go 標籤：`json:"staging_bytes"`
    pub staging_bytes: i64,
    /// Go 標籤：`json:"text_offset"`
    pub text_offset: i64,
    /// Go 標籤：`json:"stride"`
    pub stride: i64,
    /// Go 標籤：`json:"line_advance_rows"`
    pub line_advance_rows: i64,
    /// Go 標籤：`json:"viewport_rows"`
    pub viewport_rows: i64,
    /// Go 標籤：`json:"iterations"`
    pub iterations: i64,
    /// Go 標籤：`json:"delay_ms"`
    pub delay_ms: i64,
    /// Go 標籤：`json:"baseline_palette_initial_delta"`
    pub baseline_palette_initial_delta: i64,
    /// Go 標籤：`json:"fade_out_through_iteration"`
    pub fade_out_through_iteration: i64,
    /// Go 標籤：`json:"palette_step_cadence"`
    pub palette_step_cadence: i64,
}

/// Go：`finale.go:42`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct FinalePhaseGate {
    /// Go 標籤：`json:"source"`
    pub source: String,
    /// Go 標籤：`json:"reason"`
    pub reason: String,
}

/// Go：`finale.go:47`，`func LoadFinalePhase(path string) (*FinalePhase, error)`
pub fn load_finale_phase(path: &str) -> Result<FinalePhase> {
    todo!("ending/finale.go:LoadFinalePhase")
}

impl FinalePhase {
    /// Ready always remains false until the exact native text compositor and its
    /// staging source have an evidence-backed remake adapter.
    ///
    /// Go：`finale.go:69`，`func (p FinalePhase) Ready() bool`
    pub fn ready(&self) -> bool {
        false
    }

    /// ComposePhase0Text reproduces the fully evidenced glyph-only part of
    /// 0x2c469: physical FDTXT_031 string #44 is written continuously from
    /// staging+0x12c30 with 16-byte glyph advance and the caller's native colour
    /// arguments (stride=320, foreground=CD, shadow=4C, transparent background).
    /// FFFE is the exactly recovered line advance to destination + line*25*stride;
    /// every other FFxx word remains rejected so this does not claim to be the
    /// unrecovered general 0x15f84 control-code renderer.
    ///
    /// Go：`finale.go:78`，`func (p FinalePhase) ComposePhase0Text(staging, textResource, fontResource []byte) (int, error)`
    pub fn compose_phase0_text(&self, staging: &mut [u8], text_resource: &[u8], font_resource: &[u8]) -> Result<i64> {
        todo!("ending/finale.go:FinalePhase.ComposePhase0Text")
    }

    /// ComposePhase0TextTyped consumes the separated FDTXT/font contract. The raw
    /// wrapper above remains only for source-oracle tests.
    ///
    /// Go：`finale.go:95`，`func (p FinalePhase) ComposePhase0TextTyped(staging []byte, strings *fdtxt.Strings, font *fdtxt.Font) (int, error)`
    pub fn compose_phase0_text_typed(
        &self,
        staging: &mut [u8],
        strings: Option<&fdtxt::Strings>,
        font: Option<&fdtxt::Font>,
    ) -> Result<i64> {
        todo!("ending/finale.go:FinalePhase.ComposePhase0TextTyped")
    }
}

/// Phase0Player is the exact 500-pass row-scroll portion after 0x2c469. It is
/// intentionally separate from the ending Player: completing this phase does
/// not authorise execution of the unrecovered 0x2c548 montage.
///
/// Go：`finale.go:125`
#[derive(Debug, Clone, Default)]
pub struct Phase0Player {
    pub(crate) phase: FinalePhase,
    pub(crate) staging: Vec<u8>,
    pub(crate) compositor: Option<SharedCompositor>,
    pub(crate) iteration: i64,
    pub(crate) palette_delta: i64,
    pub(crate) wait_ms: i64,
    pub(crate) done: bool,
}

/// Phase0Assets keeps every non-global input needed by the recovered phase.
/// Baseline must come from the preceding native presentation state; callers
/// cannot silently substitute a convenience palette.
///
/// Go：`finale.go:138`
#[derive(Debug, Clone)]
pub struct Phase0Assets {
    pub text_resource: Vec<u8>,
    pub font_resource: Vec<u8>,
    pub strings: Option<fdtxt::Strings>,
    pub font: Option<fdtxt::Font>,
    pub baseline: [u8; 768],
}

impl Default for Phase0Assets {
    fn default() -> Self {
        Self {
            text_resource: Default::default(),
            font_resource: Default::default(),
            strings: Default::default(),
            font: Default::default(),
            baseline: std::array::from_fn(|_| Default::default()),
        }
    }
}

/// Go：`finale.go:146`，`func NewPhase0PlayerFromAssets(phase FinalePhase, assets Phase0Assets, compositor *IndexedCompositor) (*Phase0Player, error)`
pub fn new_phase0_player_from_assets(
    phase: &FinalePhase,
    assets: &Phase0Assets,
    compositor: Option<SharedCompositor>,
) -> Result<Phase0Player> {
    todo!("ending/finale.go:NewPhase0PlayerFromAssets")
}

/// Go：`finale.go:170`，`func NewPhase0Player(phase FinalePhase, staging []byte, compositor *IndexedCompositor) (*Phase0Player, error)`
pub fn new_phase0_player(
    phase: &FinalePhase,
    staging: &[u8],
    compositor: Option<SharedCompositor>,
) -> Result<Phase0Player> {
    todo!("ending/finale.go:NewPhase0Player")
}

impl Phase0Player {
    /// Advance consumes the native 1ms waits. The first iteration presents
    /// immediately, as the original falls through to 0x2c4b4 before its first
    /// 0x17aa9(1) call.
    ///
    /// Go：`finale.go:180`，`func (p *Phase0Player) Advance(elapsedMS int) (done bool, err error)`
    /// Go 具名回傳值依序為：done、err。
    pub fn advance(&mut self, elapsed_ms: i64) -> Result<bool> {
        todo!("ending/finale.go:Phase0Player.Advance")
    }
}
