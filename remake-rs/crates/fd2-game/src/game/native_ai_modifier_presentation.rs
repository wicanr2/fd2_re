//! Go 來源：`remake/cmd/fd2/native_ai_modifier_presentation.go`（604 行）。
//!
//! 骨架：Go 檔的 8 個函式中產生 2 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_ai_modifier_presentation.go:15`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeAICommandModifierPresentationJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeAICommandModifierPlan>,
    pub(crate) plan2022: Option<battle::NativeAICommand2022Plan>,
    pub(crate) frames: Vec<NativeCompoundPresentedFrame>,
    pub(crate) frame: i64,
    pub(crate) repeat: i64,
    pub(crate) publish_at: i64,
    pub(crate) end_at: i64,
    pub(crate) drawn: bool,
    pub(crate) holding: bool,
    pub(crate) published: bool,
    pub(crate) skip_hold: bool,
    pub(crate) hold: i64,
    pub(crate) rng_before: u16,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) stage_work: Vec<u8>,
    pub(crate) stage_vga: Vec<u8>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, battle::NativeCommandModifierResult) + Send>>,
    pub(crate) then2022: Option<Box<dyn FnOnce(&mut Game, &battle::NativeAICommand2022Plan) + Send>>,
}

impl Game {
    /// Go：`native_ai_modifier_presentation.go:483`，`func (g *Game) cancelNativeAICommandModifierPresentation()`
    pub(crate) fn cancel_native_ai_command_modifier_presentation(&mut self) {
        let Some(mut j) = self.native_ai_command_modifier.take() else {
            return;
        };
        // Go 的計畫以指標指向 `g.st.Units`；Rust 交給規則層的是同一個 `units` 陣列（沒有戰場時是空陣列）。
        let mut empty = Vec::new();
        let units = self.st.as_mut().map_or(&mut empty, |st| &mut st.units);
        if j.plan2022.is_some() {
            let _ = battle::abort_native_ai_command2022(units, j.plan2022.as_mut());
        } else {
            let _ = battle::abort_native_ai_command_modifier(units, j.plan.as_mut());
        }
        self.native_rng_state = j.rng_before;
        self.native_map_work = j.baseline_work.clone();
        self.native_map_vga = j.baseline_vga.clone();
        self.native_ai_command_modifier = None;
    }

    /// Go：`native_ai_modifier_presentation.go:499`，`func (g *Game) failNativeAICommandModifierPresentation(err error)`
    /// （`r8`，隨 `stepNativeAICommandModifierPresentation` 補移植）。
    pub(crate) fn fail_native_ai_command_modifier_presentation(&mut self, err: &GameError) {
        self.cancel_native_ai_command_modifier_presentation();
        self.load_err = format!("native AI command modifier presentation: {}", err.message());
    }

    /// Go：`native_ai_modifier_presentation.go:504`，`func (g *Game) stepNativeAICommandModifierPresentation()`
    pub(crate) fn step_native_ai_command_modifier_presentation(&mut self) {
        let Some(j) = self.native_ai_command_modifier.as_mut() else {
            return;
        };
        if j.holding {
            if j.hold > 0 {
                j.hold -= 1;
                return;
            }
            let mut empty = Vec::new();
            let units = self.st.as_mut().map_or(&mut empty, |st| &mut st.units);
            let result = if j.plan2022.is_some() {
                battle::complete_native_ai_command2022(units, j.plan2022.as_mut())
            } else {
                battle::complete_native_ai_command_modifier(units, j.plan.as_mut())
            };
            if let Err(err) = result {
                self.fail_native_ai_command_modifier_presentation(&GameError::from(err));
                return;
            }
            let mut j = self.native_ai_command_modifier.take().unwrap();
            let (then, then2022) = (j.then.take(), j.then2022.take());
            let result = j.plan.as_ref().map(|plan| plan.result.clone()).unwrap_or_default();
            if let Some(then2022) = then2022 {
                // Go 傳 `plan2022` 指標（可為 nil）；Rust 的延續收 `&NativeAICommand2022Plan`，nil 時以零值代入。
                let plan2022 = j.plan2022.take().unwrap_or_default();
                then2022(self, &plan2022);
            } else if let Some(then) = then {
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
            frame.sound_played = true;
            self.play_raw(&sound);
        }
        let j = self.native_ai_command_modifier.as_mut().unwrap();
        j.repeat += 1;
        if j.repeat < j.frames[j.frame as usize].delay {
            return;
        }
        j.repeat = 0;
        let next = j.frame + 1;
        if next == j.publish_at && !j.published {
            let mut empty = Vec::new();
            let units = self.st.as_mut().map_or(&mut empty, |st| &mut st.units);
            let result = if j.plan2022.is_some() {
                battle::publish_native_ai_command2022(units, j.plan2022.as_mut())
            } else {
                battle::publish_native_ai_command_modifier(units, j.plan.as_mut())
            };
            if let Err(err) = result {
                self.fail_native_ai_command_modifier_presentation(&GameError::from(err));
                return;
            }
            let j = self.native_ai_command_modifier.as_mut().unwrap();
            self.native_rng_state = match (&j.plan2022, &j.plan) {
                (Some(plan2022), _) => plan2022.rng_state,
                (None, Some(plan)) => plan.result.rng_state,
                // Go 解參考 nil 的 `j.plan` 會 panic；Publish 已先回報 nil 計畫的錯誤，這裡到不了。
                (None, None) => unreachable!("Go：j.plan 為 nil 時 Publish 已回錯"),
            };
            self.native_map_work = j.stage_work.clone();
            self.native_map_vga = j.stage_vga.clone();
            j.published = true;
        }
        let j = self.native_ai_command_modifier.as_mut().unwrap();
        if next == j.end_at {
            if !j.published {
                self.fail_native_ai_command_modifier_presentation(&game_err!(
                    "required modifier boundary was not presented"
                ));
                return;
            }
            j.frame = next - 1;
            j.holding = true;
            if !j.skip_hold {
                j.hold = native_delay_ticks(500);
            }
            return;
        }
        j.frame = next;
    }
}
