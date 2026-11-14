//! Go 來源：`remake/cmd/fd2/native_command9_player_presentation.go`（300 行）。
//!
//! 骨架：Go 檔的 4 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command9_player_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand9PlayerPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND9_PLAYER_PALETTE: NativeCommand9PlayerPhase = NativeCommand9PlayerPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_PLAYER_EFFECT: NativeCommand9PlayerPhase = NativeCommand9PlayerPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_PLAYER_RESULT: NativeCommand9PlayerPhase = NativeCommand9PlayerPhase(2);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND9_PLAYER_HOLD: NativeCommand9PlayerPhase = NativeCommand9PlayerPhase(3);

/// Go：`native_command9_player_presentation.go:24`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand9PlayerJob {
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) target: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) palettes: Vec<ColorPalette>,
    pub(crate) effect_frames: Vec<Vec<u8>>,
    pub(crate) result_frames: Vec<Vec<u8>>,
    pub(crate) baseline_work: Vec<u8>,
    pub(crate) baseline_vga: Vec<u8>,
    pub(crate) phase: NativeCommand9PlayerPhase,
    pub(crate) frame: i64,
    pub(crate) hold: i64,
    pub(crate) drawn: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) target_hp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) rng_before: u16,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
    pub(crate) effect_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) result_sources: Vec<Vec<indexedmap::NativeCommandTailLayer>>,
    pub(crate) hd_input: indexedmap::NativeFrameInput<'static>,
    pub(crate) hd_input_err: Option<GameError>,
    pub(crate) hd_scene: Option<ModernHDCompoundMapScene>,
    pub(crate) hd_frames: Option<ModernHDPlayer9Frames>,
    pub(crate) hd_theme: Option<ModernStoryPortraitSet>,
}

impl Game {
    /// Go：`native_command9_player_presentation.go:172`，`func (g *Game) stepNativeCommand9PlayerPresentation()`
    pub(crate) fn step_native_command9_player_presentation(&mut self) {
        let Some(j) = self.native_cmd9_player.as_mut() else {
            return;
        };
        match j.phase {
            NATIVE_COMMAND9_PLAYER_PALETTE => {
                if !j.drawn {
                    return;
                }
                j.drawn = false;
                j.frame += 1;
                if j.frame >= j.palettes.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND9_PLAYER_EFFECT, 0);
                    let sfx = self.sfx_command9_player_initial.clone();
                    self.play_raw(&sfx);
                }
            }
            NATIVE_COMMAND9_PLAYER_EFFECT => {
                if !j.drawn {
                    return;
                }
                j.drawn = false;
                j.frame += 1;
                if j.frame == 15 || j.frame == 19 {
                    let sfx = self.sfx_command9_player_repeat.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd9_player.as_mut().expect("Go：j 仍是同一個工作");
                if j.frame >= j.effect_frames.len() as i64 {
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd9_player.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) = battle::apply_native_command_damage_mp(units, j.plan.as_mut()) {
                        self.fail_native_command9_player_presentation(&err.into());
                        return;
                    }
                    let damage_stages = j.plan.as_ref().expect("Go：j.plan 已解參考").damage_stages;
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd9_player.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) =
                        battle::apply_native_command_damage_stage(units, j.plan.as_mut(), 0, damage_stages)
                    {
                        self.fail_native_command9_player_presentation(&err.into());
                        return;
                    }
                    self.native_rng_state = j.plan.as_ref().expect("Go：j.plan 已解參考").rng_after;
                    (j.phase, j.frame) = (NATIVE_COMMAND9_PLAYER_RESULT, 0);
                }
            }
            NATIVE_COMMAND9_PLAYER_RESULT => {
                if !j.drawn {
                    return;
                }
                j.drawn = false;
                j.frame += 1;
                if j.frame >= j.result_frames.len() as i64 {
                    (j.phase, j.hold) = (NATIVE_COMMAND9_PLAYER_HOLD, native_delay_ticks(500));
                }
            }
            NATIVE_COMMAND9_PLAYER_HOLD => {
                if j.hold > 0 {
                    j.hold -= 1;
                    return;
                }
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd9_player.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
                    self.fail_native_command9_player_presentation(&err.into());
                    return;
                }
                let mut j = self.native_cmd9_player.take().expect("Go：j 仍是同一個工作");
                let (then, results) =
                    (j.then.take(), j.plan.take().expect("Go：CompleteNativeCommandDamage 已檢查 plan").results);
                (self.native_map_work, self.native_map_vga) = (j.baseline_work, j.baseline_vga);
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Go：`native_command9_player_presentation.go:160`，`func (g *Game) failNativeCommand9PlayerPresentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand9PlayerPresentation` 補移植）。
    pub(crate) fn fail_native_command9_player_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd9_player.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        let actor = j.actor.expect("Go：j.actor 為 nil 時解參考 panic");
        let target = j.target.expect("Go：j.target 為 nil 時解參考 panic");
        (units[actor].mp, units[actor].acted, units[target].hp) =
            (j.actor_mp_before, j.actor_acted_before, j.target_hp_before);
        self.native_rng_state = j.rng_before;
        (self.native_map_work, self.native_map_vga) = (j.baseline_work, j.baseline_vga);
        self.load_err = format!("native command9 player presentation: {}", err.message());
    }
}

#[cfg(test)]
#[path = "native_command9_player_presentation/native_command9_player_presentation_test.rs"]
mod native_command9_player_presentation_test;
