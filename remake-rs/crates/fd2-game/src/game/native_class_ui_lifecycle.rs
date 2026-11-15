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
        if self.native_class_ui_job.is_some() {
            self.native_confirmation_mouth_owner = None;
        }
        if let Some(job) = self.native_class_ui_job.as_mut()
            && !job.timeline.is_empty()
        {
            if job.started.is_zero() {
                job.started = now;
            }
            job.elapsed = now - job.started;
            // Go 第 128..131 行另算 `total`，本函式沒有使用。
            if job.frame == 1 && job.drawn {
                let mut job = self.native_class_ui_job.take().unwrap();
                if let Some(after) = job.after.take() {
                    after(self);
                }
            }
            return;
        }
        if let Some(job) = self.native_class_ui_job.as_mut()
            && job.drawn
        {
            job.drawn = false;
            let frames = job.frames.len() as i64;
            if job.frame < frames {
                job.frame += 1;
                if job.frame < frames || !job.restore.is_empty() {
                    return;
                }
            }
            if job.frame >= frames {
                let mut job = self.native_class_ui_job.take().unwrap();
                if job.redraw_native_map_after_restore && !job.restore.is_empty() {
                    self.redraw_native_map_hud();
                }
                if let Some(after) = job.after.take() {
                    after(self);
                }
            }
        }
        let mut preparation_confirm = false;
        let mut preparation_prompt = false;
        if let Some(node) = self.camp.as_ref().and_then(|c| c.node())
            && node.r#type == "preparation"
        {
            preparation_confirm = self.prep_confirm;
            preparation_prompt = !self.prep_selecting && !self.prep_confirm && !self.prep_record_slots;
        }
        if self.native_class_ui_job.is_none()
            && (self.church_mode == "class_confirm"
                || self.church_mode == "revive_confirm"
                || preparation_confirm
                || preparation_prompt
                || self.native_system_end_turn_confirm)
        {
            let owner = self.native_confirmation_mouth_waiting_owner();
            self.step_native_confirmation_mouth(owner, false);
            let tick = self.native_class_ui_clock.sample(now);
            let advanced = self.step_native_class_ui_pulse_tick(tick);
            self.step_native_confirmation_mouth(owner, advanced);
        } else {
            self.step_native_confirmation_mouth(None, false);
        }
    }

    /// Go：`native_class_ui_lifecycle.go:242`，`func (g *Game) nativeClassUIBlocksInput() bool`
    pub(crate) fn native_class_ui_blocks_input(&self) -> bool {
        self.native_class_ui_job.is_some()
    }

    /// Go：`native_class_ui_lifecycle.go:246`，`func (g *Game) resetNativeClassUIPulse()`
    pub(crate) fn reset_native_class_ui_pulse(&mut self) {
        self.native_class_ui_clock.reset();
        self.native_class_ui_pulse = 0;
        self.native_class_ui_last_tick = 0;
        self.native_class_ui_has_tick = false;
    }

    /// 0x19B1E sign-extends the BIOS word before a 32-bit subtraction. A negative
    /// delta also advances once; only deltas zero and one wait. See #35 evidence
    /// docs/data/ida/fd2_confirmation_pulse_20261005.json.
    ///
    /// Go：`native_class_ui_lifecycle.go:256`，`func (g *Game) stepNativeClassUIPulseTick(rawTick int) bool`
    pub(crate) fn step_native_class_ui_pulse_tick(&mut self, raw_tick: i64) -> bool {
        self.step_native_class_ui_pulse_ticks(raw_tick, raw_tick)
    }

    /// The DOS loop reads again at 0x19B4E before storing the latch at 0x19B51.
    /// The host-time adapter approximates that short gap with the same sample.
    ///
    /// Go：`native_class_ui_lifecycle.go:262`，`func (g *Game) stepNativeClassUIPulseTicks(rawTick, latchTick int) bool`
    pub(crate) fn step_native_class_ui_pulse_ticks(&mut self, raw_tick: i64, latch_tick: i64) -> bool {
        let raw_tick = raw_tick as u16 as i16 as i64;
        let latch_tick = latch_tick as u16 as i16 as i64;
        if !self.native_class_ui_has_tick {
            self.native_class_ui_last_tick = raw_tick;
            self.native_class_ui_has_tick = true;
            return false;
        }
        let delta = raw_tick.wrapping_sub(self.native_class_ui_last_tick);
        if (0..2).contains(&delta) {
            return false;
        }
        self.native_class_ui_last_tick = latch_tick;
        self.native_class_ui_pulse = (self.native_class_ui_pulse + 1) & 3;
        true
    }
}

#[cfg(test)]
mod native_class_ui_lifecycle_test;

// ---- R5 階段 1 骨架（tools/rust_r5_skeleton.sh 產生；本體由 R5 階段 2 移植）----

impl Game {
    /// Go：`native_class_ui_lifecycle.go:179`，`func (g *Game) drawNativeClassUIJob(screen *ebiten.Image) bool`
    pub(crate) fn draw_native_class_ui_job(&mut self, screen: &mut Picture) -> bool {
        todo!("r5:native_class_ui_lifecycle.go:drawNativeClassUIJob")
    }
}
