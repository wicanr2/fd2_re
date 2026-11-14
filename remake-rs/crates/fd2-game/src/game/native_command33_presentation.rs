//! Go 來源：`remake/cmd/fd2/native_command33_presentation.go`（487 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command33_presentation.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand33PresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCompoundCommand33Plan>,
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
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, battle::NativeCompoundCommand33Result) + Send>>,
}

impl Game {
    /// Go：`native_command33_presentation.go:396`，`func (g *Game) cancelNativeCommand33Presentation()`
    pub(crate) fn cancel_native_command33_presentation(&mut self) {
        let Some(mut j) = self.native_cmd33_presentation.take() else {
            return;
        };
        let _ = battle::abort_native_compound_command33(presentation_units(&mut self.st), j.plan.as_mut());
        self.native_rng_state = j.rng_before;
        self.native_map_work = j.baseline_work;
        self.native_map_vga = j.baseline_vga;
    }

    /// Go：`native_command33_presentation.go:408`，`func (g *Game) failNativeCommand33Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand33Presentation` 補移植）。
    pub(crate) fn fail_native_command33_presentation(&mut self, err: &GameError) {
        self.cancel_native_command33_presentation();
        self.load_err = format!("native command33 presentation: {}", err.message());
    }

    /// Go：`native_command33_presentation.go:413`，`func (g *Game) stepNativeCommand33Presentation()`
    pub(crate) fn step_native_command33_presentation(&mut self) {
        let Some(j) = self.native_cmd33_presentation.as_mut() else {
            return;
        };
        if j.holding {
            if j.hold > 0 {
                j.hold -= 1;
                return;
            }
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd33_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::complete_native_compound_command33(units, j.plan.as_mut()) {
                self.fail_native_command33_presentation(&err.into());
                return;
            }
            let mut j = self.native_cmd33_presentation.take().expect("Go：j 仍是同一個工作");
            let (then, result) =
                (j.then.take(), j.plan.take().expect("Go：CompleteNativeCompoundCommand33 已檢查 plan").result);
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
            let j = self.native_cmd33_presentation.as_mut().expect("Go：j 仍是同一個工作");
            j.frames[j.frame as usize].sound_played = true;
        }
        let j = self.native_cmd33_presentation.as_mut().expect("Go：j 仍是同一個工作");
        j.repeat += 1;
        if j.repeat < j.frames[j.frame as usize].delay {
            return;
        }
        j.repeat = 0;
        let next = j.frame + 1;
        if next == j.publish_at && !j.published {
            let units = presentation_units(&mut self.st);
            let j = self.native_cmd33_presentation.as_mut().expect("Go：j 仍是同一個工作");
            if let Err(err) = battle::publish_native_compound_command33(units, j.plan.as_mut()) {
                self.fail_native_command33_presentation(&err.into());
                return;
            }
            self.native_rng_state = j.plan.as_ref().expect("Go：j.plan 已解參考").result.restore.rng_state;
            self.native_map_work = j.post_transaction_work.clone();
            self.native_map_vga = j.post_vga.clone();
            j.published = true;
        }
        let j = self.native_cmd33_presentation.as_mut().expect("Go：j 仍是同一個工作");
        j.frame = next;
        if j.frame >= j.frames.len() as i64 {
            if !j.published {
                self.fail_native_command33_presentation(&game_err!("required restore boundary was not presented"));
                return;
            }
            j.frame = j.frames.len() as i64 - 1;
            (j.holding, j.hold) = (true, native_delay_ticks(500));
        }
    }
}

#[cfg(test)]
#[path = "native_command33_presentation/native_command33_presentation_test.rs"]
mod native_command33_presentation_test;
