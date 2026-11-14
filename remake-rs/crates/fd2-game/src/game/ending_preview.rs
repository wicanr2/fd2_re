//! Go 來源：`remake/cmd/fd2/ending_preview.go`（864 行）。
//!
//! 骨架：Go 檔的 32 個函式中產生 0 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// nativeEndingPreview preserves the recovered 0x2bce5 prefix and also owns
/// the source-bound campaign ending path. It still grades call-time record and
/// global continuity, sound/input ownership and player-path E2 separately;
/// FD2_ENDING_PREFIX=1 remains a developer-only direct preview entry.
///
/// Go：`ending_preview.go:27`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeEndingPreview {
    pub(crate) player: Option<ending::Player>,
    pub(crate) view: Option<Arc<Picture>>,
    pub(crate) last: GameInstant,
    pub(crate) remainder: GoDuration,
    pub(crate) chapter: i64,
    pub(crate) queued: bool,
    pub(crate) campaign_source_bound: bool,
    pub(crate) audio_cue_consumed: bool,
    pub(crate) montage: Option<ending::MontageCycle>,
    pub(crate) montage_wait: GoDuration,
    pub(crate) montage_input_pending: bool,
    pub(crate) montage_start_attempted: bool,
    pub(crate) montage_start_error: String,
    pub(crate) tail: Option<ending::MontageTailAssets>,
    pub(crate) tail_player: Option<ending::MontageTailPlayer>,
    pub(crate) tail_wait: GoDuration,
    pub(crate) tail_start_attempted: bool,
    pub(crate) tail_start_error: String,
    pub(crate) review_party_outcomes: bool,
    pub(crate) review_cycles: i64,
    pub(crate) dialogue: Option<ending::NativeDialoguePlayback>,
    pub(crate) dialogue_view: Option<Arc<Picture>>,
    pub(crate) dialogue_mouth: dato::MouthState,
    pub(crate) dialogue_mouth_ready: bool,
}
