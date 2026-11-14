//! Go 來源：`remake/cmd/fd2/native_command32_presentation.go`（522 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command32_presentation.go:16`
#[derive(Debug, Clone, Default)]
pub(crate) struct NativeCompoundPresentedFrame {
    pub(crate) hd_map: Option<ModernHDCompoundMapSource>,
    pub(crate) hd_sources: Option<ModernHDBattleSources>,
    pub(crate) hd_palette: ColorPalette,
    pub(crate) hd_resource: i64,
    pub(crate) hd_mix: f32,
    pub(crate) hd_target: [f32; 3],
    pub(crate) hd_subtract: f32,
    pub(crate) hd_phase: String,
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) delay: i64,
    pub(crate) sound: Vec<u8>,
    pub(crate) sound_played: bool,
}

/// Go：`native_command32_presentation.go:31`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand32PresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCompoundCommand32Plan>,
    pub(crate) frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) publish_at: i64,
    pub(crate) published: bool,
    pub(crate) drawn: bool,
    pub(crate) holding: bool,
    pub(crate) hold: i64,
    pub(crate) rng_before: u16,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) post_transaction_work: Vec<u8>,
    pub(crate) post_vga: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, battle::NativeCompoundCommand32Result) + Send>>,
}

impl Game {
    /// Go：`native_command32_presentation.go:429`，`func (g *Game) cancelNativeCommand32Presentation()`
    pub(crate) fn cancel_native_command32_presentation(&mut self) {
        let Some(mut j) = self.native_cmd32_presentation.take() else {
            return;
        };
        let _ = battle::abort_native_compound_command32(presentation_units(&mut self.st), j.plan.as_mut());
        self.native_rng_state = j.rng_before;
        self.native_map_work = j.baseline_work;
        self.native_map_vga = j.baseline_vga;
    }

    /// Go：`native_command32_presentation.go:441`，`func (g *Game) failNativeCommand32Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand32Presentation` 補移植）。
    pub(crate) fn fail_native_command32_presentation(&mut self, err: &GameError) {
        self.cancel_native_command32_presentation();
        self.load_err = format!("native command32 presentation: {}", err.message());
    }

    /// Go：`native_command32_presentation.go:446`，`func (g *Game) stepNativeCommand32Presentation()`
    pub(crate) fn step_native_command32_presentation(&mut self) {
        let Some(j) = self.native_cmd32_presentation.as_mut() else {
            return;
        };
        if j.holding {
            if j.hold > 0 {
                j.hold -= 1;
                return;
            }
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd32_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::complete_native_compound_command32(units, j.plan.as_mut()) {
                self.fail_native_command32_presentation(&err.into());
                return;
            }
            let mut j = self.native_cmd32_presentation.take().expect("Go：j 仍是同一個工作");
            let (then, result) =
                (j.then.take(), j.plan.take().expect("Go：CompleteNativeCompoundCommand32 已檢查 plan").result);
            if let Some(then) = then {
                then(self, result);
            }
            return;
        }
        if !j.drawn || j.frame < 0 || j.frame >= j.frames.len() as i64 {
            return;
        }
        j.drawn = false;
        let frame = &mut j.frames[j.frame as usize];
        if !frame.sound_played && !frame.sound.is_empty() {
            let sound = frame.sound.clone();
            self.play_raw(&sound);
            let j = self.native_cmd32_presentation.as_mut().expect("Go：j 仍是同一個工作");
            j.frames[j.frame as usize].sound_played = true;
        }
        let j = self.native_cmd32_presentation.as_mut().expect("Go：j 仍是同一個工作");
        j.repeat += 1;
        if j.repeat < j.frames[j.frame as usize].delay {
            return;
        }
        j.repeat = 0;
        let next = j.frame + 1;
        if next == j.publish_at && !j.published {
            let targets = j.plan.as_ref().map_or(0, |plan| plan.result.targets.len());
            for index in 0..targets {
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd32_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::apply_native_compound_command32_target(units, j.plan.as_mut(), index as i64) {
                    self.fail_native_command32_presentation(&err.into());
                    return;
                }
            }
            let j = self.native_cmd32_presentation.as_mut().expect("Go：j 仍是同一個工作");
            self.native_rng_state = j.plan.as_ref().expect("Go：j.plan 已解參考").result.rng_state;
            self.native_map_work = j.post_transaction_work.clone();
            self.native_map_vga = j.post_vga.clone();
            j.published = true;
        }
        let j = self.native_cmd32_presentation.as_mut().expect("Go：j 仍是同一個工作");
        j.frame = next;
        if j.frame >= j.frames.len() as i64 {
            if !j.published {
                self.fail_native_command32_presentation(&game_err!("required damage boundary was not presented"));
                return;
            }
            j.frame = j.frames.len() as i64 - 1;
            (j.holding, j.hold) = (true, native_delay_ticks(500));
        }
    }
}
