//! Go 來源：`remake/cmd/fd2/native_command_heal_presentation.go`（509 行）。
//!
//! 骨架：Go 檔的 5 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command_heal_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommandHealPresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND_HEAL_FRAMES: NativeCommandHealPresentationPhase = NativeCommandHealPresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_MID_HOLD: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_FRONT_HOLD: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_EFFECT_FRAMES: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(3);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_MASK_FRAMES: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(4);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_DIGIT_FRAMES: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(5);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND_HEAL_DIGIT_HOLD: NativeCommandHealPresentationPhase =
    NativeCommandHealPresentationPhase(6);

/// Go：`native_command_heal_presentation.go:27`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommandHealPresentationJob {
    /// Original wrapper identity, also used by read-only input audit.
    pub(crate) command_id: i64,
    pub(crate) front_schedule: fdother::NativeCommandHealPresentationSchedule,
    pub(crate) tail_schedule: fdother::NativeCommandHealTailSchedule,
    pub(crate) front_frames: Vec<Vec<u8>>,
    pub(crate) effect_frames: Vec<Vec<u8>>,
    pub(crate) mask_frame: Vec<u8>,
    pub(crate) digit_frames: Vec<Vec<u8>>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) digits: Vec<fdother::LMI1Entry>,
    pub(crate) targets: Vec<indexedmap::NativeCommandHealTailTarget>,
    pub(crate) palette: ColorPalette,
    pub(crate) frame: i64,
    pub(crate) phase: NativeCommandHealPresentationPhase,
    pub(crate) hold: i64,
    pub(crate) drawn: bool,
    pub(crate) transaction: Option<Box<dyn FnMut(&mut Game) -> Result<Vec<battle::NativeCommandHealResult>> + Send>>,
    pub(crate) results: Vec<battle::NativeCommandHealResult>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandHealResult>) + Send>>,
    pub(crate) map_stage: Option<NativeCh23AdapterState>,
    pub(crate) hd_before: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_signature: [u8; 32],
    pub(crate) hd_input: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_baseline: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) hd_passes: Vec<fdother::IndexedTransitionPass>,
    pub(crate) hd_luts: Vec<Vec<u8>>,
    pub(crate) effect_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) digit_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) effect_targets: Vec<indexedmap::NativeCommandHealTailTarget>,
    pub(crate) hd_frames: Option<ModernHDHealFrames>,
    pub(crate) hd_cached: Option<ModernHDHealFrames>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
    pub(crate) hd_cached_theme: Option<ModernStoryPortraitSet>,
}

impl Game {
    /// Go：`native_command_heal_presentation.go:341`，`func (g *Game) stepNativeCommandHealPresentation()`
    pub(crate) fn step_native_command_heal_presentation(&mut self) {
        let Some(j) = self.native_heal_presentation.as_mut() else {
            return;
        };
        match j.phase {
            NATIVE_COMMAND_HEAL_FRAMES => {
                if !j.drawn {
                    return;
                }
                if j.frame + 1 == j.front_schedule.mid_frame {
                    (j.phase, j.hold) =
                        (NATIVE_COMMAND_HEAL_MID_HOLD, native_delay_ticks(j.front_schedule.mid_delay_ms));
                    return;
                }
                if j.frame + 1 >= j.front_frames.len() as i64 {
                    (j.phase, j.hold, j.drawn) =
                        (NATIVE_COMMAND_HEAL_FRONT_HOLD, native_delay_ticks(j.front_schedule.tail_delay_ms), false);
                    return;
                }
                j.frame += 1;
                j.drawn = false;
            }
            NATIVE_COMMAND_HEAL_MID_HOLD => {
                if j.hold > 0 {
                    j.hold -= 1;
                    return;
                }
                (j.frame, j.phase, j.drawn) = (j.front_schedule.mid_frame, NATIVE_COMMAND_HEAL_FRAMES, false);
            }
            NATIVE_COMMAND_HEAL_FRONT_HOLD => {
                if j.hold > 0 {
                    j.hold -= 1;
                    return;
                }
                (j.frame, j.phase, j.drawn) = (0, NATIVE_COMMAND_HEAL_EFFECT_FRAMES, false);
                let sample = j.tail_schedule.effect_sample;
                self.play_sfx(sample);
            }
            NATIVE_COMMAND_HEAL_EFFECT_FRAMES => {
                if !j.drawn {
                    return;
                }
                if j.frame + 1 < j.effect_frames.len() as i64 {
                    j.frame += 1;
                    j.drawn = false;
                    return;
                }
                (j.frame, j.phase, j.drawn) = (0, NATIVE_COMMAND_HEAL_MASK_FRAMES, false);
                let sample = j.tail_schedule.mask_sample;
                self.play_sfx(sample);
            }
            NATIVE_COMMAND_HEAL_MASK_FRAMES => {
                if !j.drawn {
                    return;
                }
                if j.frame < j.tail_schedule.mask_pairs * 2 {
                    j.frame += 1;
                    j.drawn = false;
                    return;
                }
                if j.hd_input_err.is_none() {
                    // Go 在忠實模式也會走到（`hdInputErr` 由 start 的 `freezeModernHDCommandGeometry` 決定）。
                    todo!(
                        "R8: modern_hd_heal_source.go:validateModernHDHealSource（與 r8 的 startNativeCommandHealPresentation 一起移植）"
                    );
                }
                let rollback = self.native_heal_rollback();
                let mut transaction =
                    self.native_heal_presentation.as_mut().expect("Go：j 仍是同一個工作").transaction.take();
                // Go `j.transaction()`：nil func 解參考 panic。
                let results = (transaction.as_mut().expect("Go：j.transaction 為 nil 時呼叫 panic"))(self);
                if let Some(j) = self.native_heal_presentation.as_mut() {
                    j.transaction = transaction;
                }
                let results = match results {
                    Ok(results) => results,
                    Err(err) => {
                        rollback(self);
                        self.load_err = format!("native command heal post-presentation transaction: {}", err.message());
                        self.native_heal_presentation = None;
                        return;
                    }
                };
                let _ = results;
                todo!(
                    "R8: native_command_heal_presentation.go:buildNativeCommandHealDigitFrames（r8；含 HD 幾何凍結，與 start 一起移植）"
                );
            }
            NATIVE_COMMAND_HEAL_DIGIT_FRAMES => {
                if !j.drawn {
                    return;
                }
                if j.frame + 1 < j.digit_frames.len() as i64 {
                    j.frame += 1;
                    j.drawn = false;
                    return;
                }
                (j.phase, j.hold, j.drawn) =
                    (NATIVE_COMMAND_HEAL_DIGIT_HOLD, native_delay_ticks(j.tail_schedule.digit_hold_ms), false);
            }
            NATIVE_COMMAND_HEAL_DIGIT_HOLD => {
                if j.hold > 0 {
                    j.hold -= 1;
                    return;
                }
                let mut j = self.native_heal_presentation.take().expect("Go：j 仍是同一個工作");
                let (then, results) = (j.then.take(), std::mem::take(&mut j.results));
                self.native_map_vga = j.baseline_vga;
                self.native_map_work = j.baseline_work;
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Both production callbacks are the native player/AI heal transactions.
    /// They write unit scalar/array fields and the native RNG, then defer action
    /// cleanup to the continuation. Keep unit pointers stable on a failed redraw.
    ///
    /// Go：`modern_hd_heal_source.go:76`，`func (g *Game) nativeHealRollback() func()`（`r8`，隨呼叫端
    /// `stepNativeCommandHealPresentation` 補移植）。Go 以單位指標為鍵存值複本；Rust 依 `st.units` 的索引。
    pub(crate) fn native_heal_rollback(&self) -> Box<dyn FnOnce(&mut Game) + Send> {
        let units = self.st.as_ref().expect("Go：g.st 為 nil 時解參考 panic").units.clone();
        let rng = self.native_rng_state;
        let (work, vga) = (self.native_map_work.clone(), self.native_map_vga.clone());
        Box::new(move |g: &mut Game| {
            let current = presentation_units(&mut g.st);
            for (index, before) in units.into_iter().enumerate() {
                if let Some(unit) = current.get_mut(index) {
                    *unit = before;
                }
            }
            g.native_rng_state = rng;
            (g.native_map_work, g.native_map_vga) = (work, vga);
        })
    }
}
