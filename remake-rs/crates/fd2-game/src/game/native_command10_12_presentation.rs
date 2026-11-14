//! Go 來源：`remake/cmd/fd2/native_command10_12_presentation.go`（346 行）。
//!
//! 骨架：Go 檔的 6 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command10_12_presentation.go:16`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand1012Phase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND1012_PRELUDE: NativeCommand1012Phase = NativeCommand1012Phase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND1012_MAIN: NativeCommand1012Phase = NativeCommand1012Phase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND1012_RESULT: NativeCommand1012Phase = NativeCommand1012Phase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND1012_HOLD: NativeCommand1012Phase = NativeCommand1012Phase(3);

/// Go：`native_command10_12_presentation.go:25`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand1012Job {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) schedule: fdother::NativeCommand10To12Schedule,
    pub(crate) main_frames: Vec<Vec<u8>>,
    pub(crate) result_frames: Vec<Vec<u8>>,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) palette: ColorPalette,
    pub(crate) phase: NativeCommand1012Phase,
    pub(crate) frame: i64,
    pub(crate) hold: i64,
    pub(crate) drawn: bool,
    pub(crate) prelude_done: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: Vec<i64>,
    pub(crate) rng_before: u16,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
    pub(crate) result_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) hd_input: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_baseline: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_signature: [u8; 32],
    pub(crate) hd_frames: Option<ModernHDCommand1012Frames>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
}

impl Game {
    /// Go：`native_command10_12_presentation.go:231`，`func (g *Game) stepNativeCommand1012Presentation()`
    pub(crate) fn step_native_command1012_presentation(&mut self) {
        let Some(j) = self.native_cmd1012.as_mut() else {
            return;
        };
        if j.phase == NATIVE_COMMAND1012_PRELUDE {
            if self.native2189_a.is_none() && !j.prelude_done {
                self.fail_native_command1012_presentation(&game_err!("prelude ended without completion"));
            }
            return;
        }
        if j.phase == NATIVE_COMMAND1012_MAIN {
            if !j.drawn {
                return;
            }
            j.drawn = false;
            if j.frame == 0 {
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd1012.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::apply_native_command_damage_mp(units, j.plan.as_mut()) {
                    self.fail_native_command1012_presentation(&err.into());
                    return;
                }
            }
            let j = self.native_cmd1012.as_mut().expect("Go：j 仍是同一個工作");
            j.frame += 1;
            let frame = j.frame;
            let markers = j.schedule.main_sample_frames;
            for marker in markers {
                if frame == marker {
                    let sfx = self.sfx_command1012_main.clone();
                    self.play_raw(&sfx);
                }
            }
            let j = self.native_cmd1012.as_mut().expect("Go：j 仍是同一個工作");
            if j.frame < j.main_frames.len() as i64 {
                return;
            }
            let (targets, damage_stages) =
                j.plan.as_ref().map_or((0, 0), |plan| (plan.results.len(), plan.damage_stages));
            for index in 0..targets {
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd1012.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) =
                    battle::apply_native_command_damage_stage(units, j.plan.as_mut(), index as i64, damage_stages)
                {
                    self.fail_native_command1012_presentation(&err.into());
                    return;
                }
            }
            let j = self.native_cmd1012.as_mut().expect("Go：j 仍是同一個工作");
            self.native_rng_state = j.plan.as_ref().expect("Go：j.plan 已解參考").rng_after;
            (j.phase, j.frame) = (NATIVE_COMMAND1012_RESULT, 0);
            return;
        }
        if j.phase == NATIVE_COMMAND1012_RESULT {
            if !j.drawn {
                return;
            }
            j.drawn = false;
            j.frame += 1;
            if j.frame >= j.result_frames.len() as i64 {
                (j.phase, j.hold) = (NATIVE_COMMAND1012_HOLD, native_delay_ticks(j.schedule.result_hold_ms));
            }
            return;
        }
        if j.hold > 0 {
            j.hold -= 1;
            return;
        }
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd1012.as_mut().expect("Go：j 仍是同一個工作");
        if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
            self.fail_native_command1012_presentation(&err.into());
            return;
        }
        let mut j = self.native_cmd1012.take().expect("Go：j 仍是同一個工作");
        let (then, results) =
            (j.then.take(), j.plan.take().expect("Go：CompleteNativeCommandDamage 已檢查 plan").results);
        (self.native_map_work, self.native_map_vga) = (j.baseline_work, j.baseline_vga);
        if let Some(then) = then {
            then(self, results);
        }
    }

    /// Go：`native_command10_12_presentation.go:216`，`func (g *Game) failNativeCommand1012Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand1012Presentation` 補移植）。
    pub(crate) fn fail_native_command1012_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd1012.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        let actor = j.actor.expect("Go：j.actor 為 nil 時解參考 panic");
        (units[actor].mp, units[actor].acted) = (j.actor_mp_before, j.actor_acted_before);
        for (i, result) in j.plan.as_ref().expect("Go：j.plan 為 nil 時解參考 panic").results.iter().enumerate() {
            units[result.target.expect("Go：result.Target 為 nil 時解參考 panic")].hp = j.target_hp_before[i];
        }
        self.native_rng_state = j.rng_before;
        (self.native_map_work, self.native_map_vga) = (j.baseline_work, j.baseline_vga);
        self.load_err = format!("native command10-12 presentation: {}", err.message());
    }
}

#[cfg(test)]
#[path = "native_command10_12_presentation/native_command10_12_presentation_test.rs"]
mod native_command10_12_presentation_test;
