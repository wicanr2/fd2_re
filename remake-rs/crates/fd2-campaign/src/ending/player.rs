//! Go 來源：`remake/internal/ending/player.go`（370 行）。

use super::*;

/// Player advances the evidence-backed prefix of a native ending against a
/// millisecond presentation clock.  It deliberately blocks at the first
/// unsupported native operation, retaining the last verified VGA frame for a
/// UI adapter to display.  It is not a permissive generic scripting runtime.
///
/// Go：`player.go:15`
#[derive(Debug, Clone, Default)]
pub struct Player {
    pub timeline: Timeline,
    pub frames: Vec<fdother::Frame>,
    pub ani: Option<afm::Clip>,
    pub compositor: Option<SharedCompositor>,
    pub segment: i64,
    pub wait_ms: i64,
    /// Rust 名稱改為 `ani_player`：Go 欄位 `ani` 與 `ANI` 轉成 snake_case 後相同。
    pub(crate) ani_player: Option<ANIPlayer>,
    pub(crate) ramp: Option<PaletteRamp>,
    pub(crate) composite: Option<Composite40>,
    pub(crate) composite200: Option<Composite200>,
    pub(crate) phase0: Option<Phase0Player>,
    pub(crate) phase0_spec: Option<FinalePhase>,
    pub(crate) phase0_assets: Option<Phase0Assets>,
    pub blocked: Option<Segment>,
    pub state: PlaybackState,
}

impl Player {
    /// EnableRecoveredPhase0 supplies the separated text/font assets that 0x2c405
    /// consumes after the already-recovered 0x2bce5 prefix. It does not make the
    /// ending generally playable: it only permits the exact 0x2c172 hand-off and
    /// will block again at the verified 0x2c548 montage boundary.
    ///
    /// Palette input deliberately is not accepted here.  The native routine keeps
    /// the DAC state produced by the preceding ending presentation, so the bridge
    /// takes it only from this Player's compositor after PresentANI has captured
    /// it.
    ///
    /// Go：`player.go:43`，`func (p *Player) EnableRecoveredPhase0(phase FinalePhase, assets Phase0Assets) error`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn enable_recovered_phase0(&mut self, phase: &FinalePhase, assets: &Phase0Assets) -> Result<()> {
        todo!("ending/player.go:Player.EnableRecoveredPhase0")
    }
}

/// Go：`player.go:55`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct Composite40 {
    pub(crate) iteration: i64,
    pub(crate) delay: i64,
}

/// Go：`player.go:56`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct Composite200 {
    pub(crate) iteration: i64,
    pub(crate) delay: i64,
}

/// Go：`player.go:58`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq)]
pub(crate) struct PaletteRamp {
    pub(crate) value: i64,
    pub(crate) end: i64,
    pub(crate) step: i64,
    pub(crate) delay: i64,
    pub(crate) next: bool,
}

/// Go：`player.go:63`，`func NewPlayer(t Timeline, frames []fdother.Frame, ani *afm.Clip, compositor *IndexedCompositor) (*Player, error)`
pub fn new_player(
    t: Timeline,
    frames: &[fdother::Frame],
    ani: Option<&afm::Clip>,
    compositor: Option<SharedCompositor>,
) -> Result<Player> {
    todo!("ending/player.go:NewPlayer")
}

impl Player {
    /// VerifiedAudioCues returns the raw ending audio schedule in source order.
    /// The method is intentionally observational: callers must not treat the
    /// returned list as permission to bypass the Player's blocked montage gate.
    /// A future montage owner can consume each cue only after reaching its
    /// Trigger stage with the corresponding native renderer evidence.
    ///
    /// Go：`player.go:104`，`func (p *Player) VerifiedAudioCues() []AudioCue`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn verified_audio_cues(&self) -> Vec<AudioCue> {
        todo!("ending/player.go:Player.VerifiedAudioCues")
    }

    /// AudioCueForRuntimeStage 只依資料中已驗證且不重複的執行期階段取出 cue；
    /// 它不播放音訊，也不允許 caller 略過對應的畫面／資產 admission。
    ///
    /// Go：`player.go:113`，`func (p *Player) AudioCueForRuntimeStage(stage string) (AudioCue, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn audio_cue_for_runtime_stage(&self, stage: &str) -> (AudioCue, bool) {
        todo!("ending/player.go:Player.AudioCueForRuntimeStage")
    }

    /// AudioCueAtBlockedBoundary 只回傳 after_gate 與目前失敗即關閉邊界精確相符的
    /// 音訊 cue。它不前進播放、不略過 renderer，也不讓尚未還原 owner 的 cue 被
    /// 提前消費。
    ///
    /// Go：`player.go:128`，`func (p *Player) AudioCueAtBlockedBoundary() (AudioCue, bool)`
    /// Go 版先檢查 nil 接收者；Rust 的 `self` 不會是 null，該分支不移植。
    pub fn audio_cue_at_blocked_boundary(&self) -> (AudioCue, bool) {
        todo!("ending/player.go:Player.AudioCueAtBlockedBoundary")
    }

    /// Advance consumes up to elapsedMS of wall-clock time.  The caller should use
    /// a monotonic presentation clock; zero is useful to present the first ANI
    /// frame immediately, exactly as 0x20421 does.
    ///
    /// Go：`player.go:143`，`func (p *Player) Advance(elapsedMS int) (PlaybackState, error)`
    pub fn advance(&mut self, elapsed_ms: i64) -> Result<PlaybackState> {
        todo!("ending/player.go:Player.Advance")
    }

    /// Go：`player.go:326`，`func (p *Player) blit(s Segment) error`
    pub(crate) fn blit(&mut self, s: &Segment) -> Result<()> {
        todo!("ending/player.go:Player.blit")
    }

    /// BlockedDialogue returns the exact branch selected by the native chapter
    /// comparison at an opaque text call.  The player remains blocked: presenting
    /// these lines never grants permission to execute following opaque segments.
    ///
    /// Go：`player.go:345`，`func (p *Player) BlockedDialogue(chapter int) ([]DialogueBlock, bool)`
    pub fn blocked_dialogue(&self, chapter: i64) -> (Vec<DialogueBlock>, bool) {
        todo!("ending/player.go:Player.BlockedDialogue")
    }

    /// ResumeBlockedDialogue advances exactly one already-presented native text
    /// gate.  It refuses every other opaque operation, so a UI cannot accidentally
    /// skip a not-yet-recovered renderer boundary.
    ///
    /// Go：`player.go:362`，`func (p *Player) ResumeBlockedDialogue() bool`
    pub fn resume_blocked_dialogue(&mut self) -> bool {
        todo!("ending/player.go:Player.ResumeBlockedDialogue")
    }
}
