//! Go 來源：`remake/cmd/fd2/native_command6_presentation.go`（560 行）。
//!
//! 骨架：Go 檔的 7 個函式中產生 1 個（階段 0 清單），其餘標 `r8`，見 `PORTING.json`。

use super::*;

/// Go：`native_command6_presentation.go:15`
#[derive(Debug, Clone, Copy, Default, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub(crate) struct NativeCommand6PresentationPhase(pub(crate) u8);

/// Go 原式：`iota`
pub(crate) const NATIVE_COMMAND6_COMMON_PRELUDE: NativeCommand6PresentationPhase = NativeCommand6PresentationPhase(0);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND6_ACTOR: NativeCommand6PresentationPhase = NativeCommand6PresentationPhase(1);

/// Go：沿用上一個常數的式子（`iota` 序列）。
pub(crate) const NATIVE_COMMAND6_HANDLER: NativeCommand6PresentationPhase = NativeCommand6PresentationPhase(2);

/// Go：`native_command6_presentation.go:23`
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub(crate) struct NativeCommand6HandlerFrame {
    pub(crate) image: Option<Arc<Picture>>,
    pub(crate) target_index: i64,
    pub(crate) hp_stage: i64,
    pub(crate) play_target: bool,
    pub(crate) play_front: bool,
    pub(crate) play_tail: bool,
}

/// Go：`native_command6_presentation.go:32`
///
/// 不 derive `Debug`／`Clone`：欄位含 callback（`Box<dyn FnOnce>`）、檔案或亂數產生器等做不到的型別。
#[derive(Default)]
pub(crate) struct NativeCommand6PresentationJob {
    pub(crate) hd: Option<ModernHDOrdinaryFrames>,
    pub(crate) actor: Option<UnitIdx>,
    pub(crate) plan: Option<battle::NativeCommandDamagePlan>,
    pub(crate) prelude: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_black: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_pulse: Vec<Option<Arc<Picture>>>,
    pub(crate) actor_specs: Vec<battlepresent::NativeCommand0ActorFrame>,
    pub(crate) handler: Vec<NativeCommand6HandlerFrame>,
    pub(crate) phase: NativeCommand6PresentationPhase,
    pub(crate) frame: i64,
    pub(crate) pulse_black: bool,
    pub(crate) drawn: bool,
    pub(crate) mp_published: bool,
    pub(crate) actor_mp_before: i64,
    pub(crate) actor_acted_before: bool,
    pub(crate) target_hp_before: Vec<i64>,
    pub(crate) then: Option<Box<dyn FnOnce(&mut Game, Vec<battle::NativeCommandDamageResult>) + Send>>,
}

impl Game {
    /// Go：`native_command6_presentation.go:443`，`func (g *Game) stepNativeCommand6Presentation()`
    pub(crate) fn step_native_command6_presentation(&mut self) {
        let Some(j) = self.native_cmd6_presentation.as_mut() else {
            return;
        };
        if !j.drawn {
            return;
        }
        j.drawn = false;
        match j.phase {
            NATIVE_COMMAND6_COMMON_PRELUDE => {
                j.frame += 1;
                if j.frame >= j.prelude.len() as i64 {
                    (j.phase, j.frame) = (NATIVE_COMMAND6_ACTOR, 0);
                }
            }
            NATIVE_COMMAND6_ACTOR => {
                let spec = &j.actor_specs[j.frame as usize];
                let (spec_pulse, spec_publish_mp) = (spec.pulse, spec.publish_mp);
                if spec_pulse && !j.pulse_black {
                    if spec_publish_mp && !j.mp_published {
                        if let Err(err) = self.publish_native_command6_mp() {
                            self.fail_native_command6_presentation(&err);
                            return;
                        }
                        let sfx = self.sfx_command6_actor.clone();
                        self.play_raw(&sfx);
                    }
                    let j = self.native_cmd6_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    j.pulse_black = true;
                    return;
                }
                if spec_publish_mp && !j.mp_published {
                    if let Err(err) = self.publish_native_command6_mp() {
                        self.fail_native_command6_presentation(&err);
                        return;
                    }
                    let sfx = self.sfx_command6_actor.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd6_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.pulse_black = false;
                j.frame += 1;
                if j.frame >= j.actor_black.len() as i64 {
                    if !j.mp_published {
                        self.fail_native_command6_presentation(&game_err!("actor MP marker was not presented"));
                        return;
                    }
                    (j.phase, j.frame) = (NATIVE_COMMAND6_HANDLER, 0);
                }
            }
            NATIVE_COMMAND6_HANDLER => {
                let frame = &j.handler[j.frame as usize];
                let (target_index, hp_stage) = (frame.target_index, frame.hp_stage);
                let play_front = frame.play_front && j.frame == 0;
                let play_target = frame.play_target;
                let play_tail = frame.play_tail && (j.frame == 0 || !j.handler[(j.frame - 1) as usize].play_tail);
                if hp_stage != 0 {
                    let units = presentation_units(&mut self.st);
                    let j = self.native_cmd6_presentation.as_mut().expect("Go：j 仍是同一個工作");
                    if let Err(err) =
                        battle::apply_native_command_damage_stage(units, j.plan.as_mut(), target_index, hp_stage)
                    {
                        self.fail_native_command6_presentation(&err.into());
                        return;
                    }
                }
                if play_front {
                    let sfx = self.sfx_command6_front.clone();
                    self.play_raw(&sfx);
                }
                if play_target {
                    let sfx = self.sfx_command6_target.clone();
                    self.play_raw(&sfx);
                }
                if play_tail {
                    let sfx = self.sfx_command6_tail.clone();
                    self.play_raw(&sfx);
                }
                let j = self.native_cmd6_presentation.as_mut().expect("Go：j 仍是同一個工作");
                j.frame += 1;
                if j.frame < j.handler.len() as i64 {
                    return;
                }
                let units = presentation_units(&mut self.st);
                let j = self.native_cmd6_presentation.as_mut().expect("Go：j 仍是同一個工作");
                if let Err(err) = battle::complete_native_command_damage(units, j.plan.as_mut()) {
                    self.fail_native_command6_presentation(&err.into());
                    return;
                }
                let plan = j.plan.as_ref().expect("Go：CompleteNativeCommandDamage 已檢查 plan");
                self.native_rng_state = plan.rng_after;
                let results = plan.results.clone();
                let then = j.then.take();
                self.native_cmd6_presentation = None;
                if let Some(then) = then {
                    then(self, results);
                }
            }
            _ => {}
        }
    }

    /// Go `stepNativeCommand6Presentation` 兩處相同的 MP 發布：`ApplyNativeCommandDamageMP` 成功後設 `mpPublished`。
    fn publish_native_command6_mp(&mut self) -> Result<()> {
        let units = presentation_units(&mut self.st);
        let j = self.native_cmd6_presentation.as_mut().expect("Go：j 仍是同一個工作");
        battle::apply_native_command_damage_mp(units, j.plan.as_mut())?;
        j.mp_published = true;
        Ok(())
    }

    /// Go：`native_command6_presentation.go:428`，`func (g *Game) failNativeCommand6Presentation(err error)`
    /// （`r8`，隨呼叫端 `stepNativeCommand6Presentation` 補移植）。
    pub(crate) fn fail_native_command6_presentation(&mut self, err: &GameError) {
        let Some(j) = self.native_cmd6_presentation.take() else {
            return;
        };
        let units = presentation_units(&mut self.st);
        if let Some(actor) = j.actor {
            (units[actor].mp, units[actor].acted) = (j.actor_mp_before, j.actor_acted_before);
        }
        if let Some(plan) = j.plan.as_ref() {
            for (index, result) in plan.results.iter().enumerate() {
                if let Some(target) = result.target
                    && index < j.target_hp_before.len()
                {
                    units[target].hp = j.target_hp_before[index];
                }
            }
        }
        self.load_err = format!("native command6 presentation: {}", err.message());
    }
}

#[cfg(test)]
#[path = "native_command6_presentation/native_command6_presentation_test.rs"]
mod native_command6_presentation_test;
