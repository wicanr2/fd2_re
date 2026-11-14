//! Go 來源：`remake/cmd/fd2/native_class_ui_lifecycle.go`（286 行）。
//!
//! 骨架：Go 檔的 11 個函式中產生 5 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_class_ui_lifecycle.go:11`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeClassUIJob {
    pub(crate) service_menu: Option<Box<NativeServiceMenuPresentation>>,
    pub(crate) frames: Vec<Vec<u8>>,
    pub(crate) restore: Vec<u8>,
    pub(crate) frame: i64,
    pub(crate) drawn: bool,
    pub(crate) after: Option<Box<dyn FnOnce(&mut Game) + Send>>,
    pub(crate) timeline: Vec<NativeClassUITimelineStep>,
    pub(crate) started: GameInstant,
    pub(crate) elapsed: GoDuration,
    pub(crate) preparation_text: Option<NativePreparationTransitionText>,
    pub(crate) gold_text: Option<NativeGoldRollText>,
    pub(crate) facility_list: Option<NativeFacilityListPresentation>,
    pub(crate) church_confirmation: Option<NativeChurchConfirmationPresentation>,
    pub(crate) status_text: Option<NativeStatusTransitionText>,
    pub(crate) expiry_text: Option<NativeTransientExpiryPresentation>,
    /// 0x196CB 的戰場關框 owner 才宣告 0x19742；城鎮共用 job 不推定重繪。
    pub(crate) redraw_native_map_after_restore: bool,
}

/// Go：`native_class_ui_lifecycle.go:31`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeClassUITimelineStep {
    pub(crate) facility_success: Option<NativeFacilitySuccessStep>,
    pub(crate) frame: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) duration: GoDuration,
    pub(crate) shop_effect_mask: Vec<u8>,
}

impl Game {
    /// stepNativeClassUILifecycle advances only a frame that Draw acknowledged.
    /// The continuation runs after the final closing frame has been presented.
    ///
    /// Go：`native_class_ui_lifecycle.go:117`，`func (g *Game) stepNativeClassUILifecycle(now time.Time)`
    pub(crate) fn step_native_class_ui_lifecycle(&mut self, now: GameInstant) {
        todo!("native_class_ui_lifecycle.go:stepNativeClassUILifecycle")
    }

    /// Go：`native_class_ui_lifecycle.go:242`，`func (g *Game) nativeClassUIBlocksInput() bool`
    pub(crate) fn native_class_ui_blocks_input(&self) -> bool {
        todo!("native_class_ui_lifecycle.go:nativeClassUIBlocksInput")
    }

    /// Go：`native_class_ui_lifecycle.go:246`，`func (g *Game) resetNativeClassUIPulse()`
    pub(crate) fn reset_native_class_ui_pulse(&mut self) {
        todo!("native_class_ui_lifecycle.go:resetNativeClassUIPulse")
    }

    /// 0x19B1E sign-extends the BIOS word before a 32-bit subtraction. A negative
    /// delta also advances once; only deltas zero and one wait. See #35 evidence
    /// docs/data/ida/fd2_confirmation_pulse_20261005.json.
    ///
    /// Go：`native_class_ui_lifecycle.go:256`，`func (g *Game) stepNativeClassUIPulseTick(rawTick int) bool`
    pub(crate) fn step_native_class_ui_pulse_tick(&mut self, raw_tick: i64) -> bool {
        todo!("native_class_ui_lifecycle.go:stepNativeClassUIPulseTick")
    }

    /// The DOS loop reads again at 0x19B4E before storing the latch at 0x19B51.
    /// The host-time adapter approximates that short gap with the same sample.
    ///
    /// Go：`native_class_ui_lifecycle.go:262`，`func (g *Game) stepNativeClassUIPulseTicks(rawTick, latchTick int) bool`
    pub(crate) fn step_native_class_ui_pulse_ticks(&mut self, raw_tick: i64, latch_tick: i64) -> bool {
        todo!("native_class_ui_lifecycle.go:stepNativeClassUIPulseTicks")
    }
}
